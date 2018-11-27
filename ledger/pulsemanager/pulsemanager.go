/*
 *    Copyright 2018 Insolar
 *
 *    Licensed under the Apache License, Version 2.0 (the "License");
 *    you may not use this file except in compliance with the License.
 *    You may obtain a copy of the License at
 *
 *        http://www.apache.org/licenses/LICENSE-2.0
 *
 *    Unless required by applicable law or agreed to in writing, software
 *    distributed under the License is distributed on an "AS IS" BASIS,
 *    WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 *    See the License for the specific language governing permissions and
 *    limitations under the License.
 */

package pulsemanager

import (
	"context"
	"sync"

	"github.com/pkg/errors"

	"github.com/insolar/insolar/core"
	"github.com/insolar/insolar/core/message"
	"github.com/insolar/insolar/instrumentation/inslogger"
	"github.com/insolar/insolar/ledger/jetdrop"
	"github.com/insolar/insolar/ledger/storage"
)

// PulseManager implements core.PulseManager.
type PulseManager struct {
	db      *storage.DB
	Recent  core.RecentStorage `inject:""`
	LR      core.LogicRunner   `inject:""`
	Bus     core.MessageBus    `inject:""`
	NodeNet core.NodeNetwork   `inject:""`
	// setLock locks Set method call.
	setLock sync.Mutex
	stopped bool
	// gotpulse signals if there is something to sync to Heavy
	gotpulse chan struct{}
	// syncdone closes when sync is over
	syncdone chan struct{}
	// stores pulse manager options
	options pmOptions
}

type pmOptions struct {
	enablesync       bool
	syncmessagelimit int
}

// Option provides functional option for TmpDB.
type Option func(*pmOptions)

// EnableSync defines is sync to heavy enabled or not.
// (suitable for tests)
func EnableSync(flag bool) Option {
	return func(opts *pmOptions) {
		opts.enablesync = flag
	}
}

// SyncMessageLimit sets soft limit in bytes for sync message size.
func SyncMessageLimit(size int) Option {
	return func(opts *pmOptions) {
		opts.syncmessagelimit = size
	}
}

// NewPulseManager creates PulseManager instance.
func NewPulseManager(db *storage.DB, options ...Option) *PulseManager {
	opts := &pmOptions{}
	for _, o := range options {
		o(opts)
	}
	return &PulseManager{
		db:       db,
		gotpulse: make(chan struct{}, 1),
		options:  *opts,
	}
}

// Current returns current pulse structure.
func (m *PulseManager) Current(ctx context.Context) (*core.Pulse, error) {
	latestPulse, err := m.db.GetLatestPulseNumber(ctx)
	if err != nil {
		return nil, err
	}
	pulse, err := m.db.GetPulse(ctx, latestPulse)
	if err != nil {
		return nil, err
	}
	return &pulse.Pulse, nil
}

func (m *PulseManager) createDrop(ctx context.Context, latestPulse *storage.Pulse) (
	drop *jetdrop.JetDrop,
	dropSerialized []byte,
	messages [][]byte,
	err error,
) {
	// latestPulseNumber, err := m.db.GetLatestPulseNumber(ctx)
	// if err != nil {
	// 	return nil, nil, err
	// }
	// latestPulse, err := m.db.GetPulse(ctx, latestPulseNumber)
	// if err != nil {
	// 	return  nil, nil, err
	// }

	prevDrop, err := m.db.GetDrop(ctx, *latestPulse.Prev)
	if err != nil {
		return nil, nil, nil, err
	}
	drop, messages, err = m.db.CreateDrop(ctx, latestPulse.Pulse.PulseNumber, prevDrop.Hash)
	if err != nil {
		return nil, nil, nil, err
	}
	err = m.db.SetDrop(ctx, drop)
	if err != nil {
		return nil, nil, nil, err
	}

	dropSerialized, err = jetdrop.Encode(drop)
	if err != nil {
		return nil, nil, nil, err
	}

	return
}

func (m *PulseManager) processDrop(ctx context.Context, latestPulse *storage.Pulse, dropSerialized []byte, messages [][]byte) error {
	msg := &message.JetDrop{
		Drop:        dropSerialized,
		Messages:    messages,
		PulseNumber: *latestPulse.Prev,
	}
	_, err := m.Bus.Send(ctx, msg, nil)
	if err != nil {
		return err
	}
	return nil
}

func (m *PulseManager) processRecentObjects(
	ctx context.Context,
	latestPulse *storage.Pulse,
	drop *jetdrop.JetDrop,
	dropSerialized []byte,
) error {

	m.Recent.ClearZeroTTLObjects()
	recentObjectsIds := m.Recent.GetObjects()
	recentRequestsIds := m.Recent.GetRequests()
	m.Recent.ClearObjects()

	var recentObjectsIdsConverted []core.RecordID
	for k := range recentObjectsIds {
		recentObjectsIdsConverted = append(recentObjectsIdsConverted, k)
	}

	recentRequests := m.getIndexes(ctx, recentRequestsIds)
	recentObjects := m.getIndexes(ctx, recentObjectsIdsConverted)

	msg := &message.HotRecords{
		Drop:            dropSerialized,
		PulseNumber:     *latestPulse.Prev,
		RecentObjects:   recentObjects,
		PendingRequests: recentRequests,
	}
	_, err := m.Bus.Send(ctx, msg, nil)
	if err != nil {
		return err
	}
	return nil
}

func (m *PulseManager) getIndexes(ctx context.Context, ids []core.RecordID) []*index.ObjectLifeline {
	recentObjects := make([]*index.ObjectLifeline, 0, len(ids))
	for _, id := range ids {
		lifeline, err := m.db.GetObjectIndex(ctx, &id, false)
		if err != nil {
			inslogger.FromContext(ctx).Error(err)
			continue
		}

		recentObjects = append(recentObjects, lifeline)
	}

	return recentObjects
}

// Set set's new pulse and closes current jet drop.
func (m *PulseManager) Set(ctx context.Context, pulse core.Pulse) error {
	// Ensure this does not execute in parallel.
	m.setLock.Lock()
	defer m.setLock.Unlock()
	if m.stopped {
		return errors.New("can't call Set method on PulseManager after stop")
	}

	// Run only on material executor.
	if err := m.db.AddPulse(ctx, pulse); err != nil {
		return errors.Wrap(err, "call of AddPulse failed")
	}

	err := m.db.SetActiveNodes(pulse.PulseNumber, m.NodeNet.GetActiveNodes())
	if err != nil {
		return errors.Wrap(err, "call of SetActiveNodes failed")
	}

	// execute only on material executor
	if m.NodeNet.GetOrigin().Role() == core.RoleLightMaterial {

		latestPulseNumber, err := m.db.GetLatestPulseNumber(ctx)
		if err != nil {
			return err
		}
		latestPulse, err := m.db.GetPulse(ctx, latestPulseNumber)
		if err != nil {
			return err
		}

		drop, dropSerialized, messages, err := m.createDrop(ctx, latestPulse)
		if err != nil {
			return err
		}

		wg := sync.WaitGroup{}
		wg.Add(2)

		var dropError error
		go func() {
			if err = m.processDrop(ctx, latestPulse, dropSerialized, messages); err != nil {
				dropError = errors.Wrap(err, "processDrop failed")
			}
			wg.Done()
		}()

		var hotRecordsError error
		go func() {
			if hotRecordsError := m.processRecentObjects(ctx, latestPulse, drop, dropSerialized); hotRecordsError != nil {
				hotRecordsError = errors.Wrap(err, "processRecentObjects failed")
			}
			wg.Done()
		}()

		wg.Wait()

		if dropError != nil {
			return dropError
		}
		if hotRecordsError != nil {
			return hotRecordsError
		}

		if err = m.db.SetLastPulseAsLightMaterial(ctx, *latestPulse.Prev); err != nil {
			return errors.Wrap(err, "call of SetLastPulseAsLightMaterial failed")
		}
		m.SyncToHeavy()
	}

	return m.LR.OnPulse(ctx, pulse)
}

// SyncToHeavy signals to sync loop there is something to sync.
//
// Should never be called after Stop.
func (m *PulseManager) SyncToHeavy() {
	// TODO: save current pulse as
	if len(m.gotpulse) == 0 {
		m.gotpulse <- struct{}{}
	}
}

// Start starts pulse manager, spawns replication goroutine under a hood.
func (m *PulseManager) Start(ctx context.Context) error {
	startPN, endPN, err := m.NextSyncPulses(ctx)
	if err != nil {
		return err
	}
	m.syncdone = make(chan struct{})
	if m.options.enablesync {
		go m.syncloop(ctx, startPN, endPN)
	}
	return nil
}

// Stop stops PulseManager. Waits replication goroutine is done.
func (m *PulseManager) Stop(ctx context.Context) error {
	// There should not to be any Set call after Stop call
	m.setLock.Lock()
	m.stopped = true
	m.setLock.Unlock()

	if m.options.enablesync {
		close(m.gotpulse)
		inslogger.FromContext(ctx).Info("waiting finish of replication to heavy node...")
		<-m.syncdone
	}
	return nil
}

func (m *PulseManager) syncloop(ctx context.Context, start, end core.PulseNumber) {
	defer close(m.syncdone)

	var err error
	inslog := inslogger.FromContext(ctx)
	for {
		for {
			if start != 0 {
				break

			}
			inslog.Debug("syncronization waiting next chunk of work")
			_, ok := <-m.gotpulse
			if !ok {
				inslog.Debug("stop is called, so we are should just stop syncronization loop")
				return
			}
			inslog.Debug("syncronization got next chunk of work")
			// get latest RP
			start, end, err = m.NextSyncPulses(ctx)
			if err != nil {
				err = errors.Wrap(err,
					"PulseManager syncloop failed on NextSyncPulseNumber call")
				inslog.Error(err)
				panic(err)
			}
		}
		inslog.Debugf("syncronization sync pulses: [%v:%v]", start, end)

		lastprocessed, syncerr := m.HeavySync(ctx, start, end)
		if syncerr != nil {
			syncerr = errors.Wrap(syncerr, "HeavySync failed")
			inslog.Error(syncerr.Error())
			// TODO: add sleep and some retry logic here?
			continue
		}
		err = m.db.SetReplicatedPulse(ctx, lastprocessed)
		if err != nil {
			err = errors.Wrap(err,
				"SetReplicatedPulse failed after success HeavySync in Pulsemanager")
			inslog.Error(err)
			panic(err)
		}
		start = 0
	}
}
