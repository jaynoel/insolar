/*
 *    Copyright 2018 INS Ecosystem
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

// Package logicrunner - infrastructure for executing smartcontracts
package logicrunner

import (
	"github.com/pkg/errors"
)

// MachineType is a type of virtual machine
type MachineType int

// Real constants of MachineType
const (
	MachineTypeBuiltin MachineType = iota
	MachineTypeGoPlugin

	MachineTypesTotalCount
)

// Executor is an interface for implementers of one particular machine type
type Executor interface {
	Exec(codeRef Reference, data []byte, method string, args Arguments) (newObjectState []byte, methodResults Arguments, err error)
}

// ArtifactManager interface
type ArtifactManager interface {
	Get(ref string) (data []byte, codeRef Reference, err error)
}

// LogicRunner is a general interface of contract executor
type LogicRunner struct {
	Executors       [MachineTypesTotalCount]Executor
	ArtifactManager ArtifactManager
}

// NewLogicRunner is constructor for `LogicRunner`
func NewLogicRunner(am ArtifactManager) (*LogicRunner, error) {
	res := LogicRunner{ArtifactManager: am}

	return &res, nil
}

// RegisterExecutor registers an executor for particular `MachineType`
func (r *LogicRunner) RegisterExecutor(t MachineType, e Executor) error {
	r.Executors[int(t)] = e
	return nil
}

// GetExecutor returns an executor for the `MachineType` if it was registered (`RegisterExecutor`),
// returns error otherwise
func (r *LogicRunner) GetExecutor(t MachineType) (Executor, error) {
	if res := r.Executors[int(t)]; res != nil {
		return res, nil
	}

	return nil, errors.New("No executor registered for machine")
}

// Execute runs a method on an object, ATM just thin proxy to `GoPlugin.Exec`
func (r *LogicRunner) Execute(ref string, method string, args Arguments) ([]byte, []byte, error) {
	data, codeRef, err := r.ArtifactManager.Get(ref)
	if err != nil {
		return nil, nil, errors.Wrap(err, "couldn't ")
	}

	executor, err := r.GetExecutor(MachineTypeGoPlugin)
	if err != nil {
		return nil, nil, errors.Wrap(err, "no executer registered")
	}

	return executor.Exec(codeRef, data, method, args)
}
