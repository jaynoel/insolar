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

package messagebus

import (
	"context"
	"crypto"

	"github.com/insolar/insolar/core"
	"github.com/insolar/insolar/core/message"
	"github.com/insolar/insolar/instrumentation/inslogger"
	"github.com/insolar/insolar/instrumentation/instracer"
	"github.com/pkg/errors"
)

type parcelFactory struct {
	RoutingTokenFactory        message.RoutingTokenFactory     `inject:""`
	PlatformCryptographyScheme core.PlatformCryptographyScheme `inject:""`
	Cryptography               core.CryptographyService        `inject:""`
}

func NewParcelFactory() message.ParcelFactory {
	return &parcelFactory{}
}

func (pf *parcelFactory) Create(
	ctx context.Context,
	msg core.Message,
	sender core.RecordRef,
	pulse core.PulseNumber,
	token core.RoutingToken,
) (core.Parcel, error) {
	if msg == nil {
		return nil, errors.New("failed to signature a nil message")
	}
	serialized := message.ToBytes(msg)
	signature, err := pf.Cryptography.Sign(serialized)
	if err != nil {
		return nil, err
	}

	if token == nil {
		target := message.ExtractTarget(msg)
		hash := pf.PlatformCryptographyScheme.IntegrityHasher().Hash(serialized)
		token = pf.RoutingTokenFactory.Create(&target, &sender, pulse, hash)
	}
	return &message.Parcel{
		Token:         token,
		Msg:           msg,
		Signature:     signature.Bytes(),
		LogTraceID:    inslogger.TraceID(ctx),
		TraceSpanData: instracer.MustSerialize(ctx),
	}, nil
}

func (pf *parcelFactory) Validate(publicKey crypto.PublicKey, parcel core.Parcel) error {
	ok := pf.Cryptography.Verify(publicKey, core.SignatureFromBytes(parcel.GetSign()), message.ToBytes(parcel.Message()))
	if !ok {
		return errors.New("parcel isn't valid")
	}
	return nil
}