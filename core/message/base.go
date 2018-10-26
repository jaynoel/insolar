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

// Package message represents message that messagebus can route
package message

import (
	"bytes"
	"encoding/gob"
	"io"
	"io/ioutil"

	"github.com/pkg/errors"

	"github.com/insolar/insolar/core"
)

// GetEmptyMessage constructs specified message
func getEmptyMessage(mt core.MessageType) (core.Message, error) {
	switch mt {
	// Logicrunner
	case core.TypeCallMethod:
		return &CallMethod{}, nil
	case core.TypeCallConstructor:
		return &CallConstructor{}, nil
	case core.TypeExecutorResults:
		return &ExecutorResults{}, nil
	case core.TypeValidateCaseBind:
		return &ValidateCaseBind{}, nil
	case core.TypeValidationResults:
		return &ValidationResults{}, nil

	// Ledger
	case core.TypeRequestCall:
		return &RequestCall{}, nil
	case core.TypeGetCode:
		return &GetCode{}, nil
	case core.TypeGetClass:
		return &GetClass{}, nil
	case core.TypeGetObject:
		return &GetObject{}, nil
	case core.TypeGetDelegate:
		return &GetDelegate{}, nil
	case core.TypeGetChildren:
		return &GetChildren{}, nil
	case core.TypeUpdateClass:
		return &UpdateClass{}, nil
	case core.TypeUpdateObject:
		return &UpdateObject{}, nil
	case core.TypeRegisterChild:
		return &RegisterChild{}, nil
	case core.TypeJetDrop:
		return &JetDrop{}, nil
	case core.TypeSetRecord:
		return &SetRecord{}, nil

	// Bootstrap
	case core.TypeBootstrapRequest:
		return &BootstrapRequest{}, nil
	default:
		return nil, errors.Errorf("unimplemented message type %d", mt)
	}
}

// MustSerializeBytes returns encoded core.Message, panics on error.
func MustSerializeBytes(msg core.Message) []byte {
	r, err := Serialize(msg)
	if err != nil {
		panic(err)
	}
	b, err := ioutil.ReadAll(r)
	if err != nil {
		panic(err)
	}
	return b
}

// Serialize returns io.Reader on buffer with encoded core.Message.
func Serialize(msg core.Message) (io.Reader, error) {
	buff := &bytes.Buffer{}
	_, err := buff.Write([]byte{byte(msg.Type())})
	if err != nil {
		return nil, err
	}

	enc := gob.NewEncoder(buff)
	err = enc.Encode(msg)
	return buff, err
}

// Deserialize returns decoded message.
func Deserialize(buff io.Reader) (core.SignedMessage, error) {
	b := make([]byte, 1)
	_, err := buff.Read(b)
	if err != nil {
		return nil, errors.New("too short slice for deserialize message")
	}

	msg, err := getEmptyMessage(core.MessageType(b[0]))
	if err != nil {
		return nil, err
	}
	signed, err := NewSignedMessage(msg, core.RecordRef{}, nil)
	enc := gob.NewDecoder(buff)
	err = enc.Decode(msg)
	return signed, err
}

// ToBytes deserialize a core.Message to bytes.
func ToBytes(msg core.Message) ([]byte, error) {
	reqBuff, err := Serialize(msg)
	if err != nil {
		return nil, errors.Wrap(err, "Failed to serialize event")
	}
	return ioutil.ReadAll(reqBuff)
}

// SignedToBytes deserialize a core.SignedMessage to bytes.
func SignedToBytes(msg core.SignedMessage) ([]byte, error) {
	reqBuff, err := Serialize(msg)
	if err != nil {
		return nil, errors.Wrap(err, "Failed to serialize event")
	}
	return ioutil.ReadAll(reqBuff)
}

func init() {
	// Bootstrap
	gob.Register(&BootstrapRequest{})
	// Logicrunner
	gob.Register(&CallConstructor{})
	gob.Register(&CallMethod{})
	gob.Register(&ExecutorResults{})
	gob.Register(&ValidateCaseBind{})
	gob.Register(&ValidationResults{})

	// Ledger
	gob.Register(&RequestCall{})
	gob.Register(&GetCode{})
	gob.Register(&GetClass{})
	gob.Register(&GetObject{})
	gob.Register(&GetDelegate{})
	gob.Register(&UpdateClass{})
	gob.Register(&UpdateObject{})
	gob.Register(&RegisterChild{})
	gob.Register(&JetDrop{})
	gob.Register(&SetRecord{})

	// Bootstrap
	gob.Register(&BootstrapRequest{})
	gob.Register(&SignedMessage{})
	gob.Register(core.RecordRef{})
}
