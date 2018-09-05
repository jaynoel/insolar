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

package goplugin

import (
	"io/ioutil"
	"os"
	"os/exec"
	"testing"

	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	"github.com/ugorji/go/codec"

	"github.com/insolar/insolar/configuration"
	"github.com/insolar/insolar/core"
	"github.com/insolar/insolar/logicrunner/goplugin/testutil"
)

func TestTypeCompatibility(t *testing.T) {
	var _ core.MachineLogicExecutor = (*GoPlugin)(nil)
}

func init() {
	log.SetLevel(log.DebugLevel)
}

func buildCLI(name string) error {
	out, err := exec.Command("go", "build", "-o", "./"+name+"/"+name, "./"+name+"/").CombinedOutput()
	if err != nil {
		return errors.Wrapf(err, "can't build %s: %s", name, string(out))
	}
	return nil
}

func buildInciderCLI() error {
	return buildCLI("ginsider-cli")
}

func buildPreprocessor() error {
	return buildCLI("preprocessor")
}

const contractOneCode = `
package main

import "github.com/insolar/insolar/logicrunner/goplugin/foundation"
import "contract-proxy/two"

type One struct {
	foundation.BaseContract
}

func (r *One) Hello(s string) string {
	friend := two.GetObject("some")
	res := friend.Hello(s)

	return "Hi, " + s + "! Two said: " + res
}
`

const contractTwoCode = `
package main

import "github.com/insolar/insolar/logicrunner/goplugin/foundation"

type Two struct {
	foundation.BaseContract
}

func (r *Two) Hello(s string) string {
	return "Hello you too, " + s
}
`

func generateContractProxy(root string, name string) error {
	dstDir := root + "/src/contract-proxy/" + name

	err := os.MkdirAll(dstDir, 0777)
	if err != nil {
		return err
	}

	contractPath := root + "/src/contract/" + name + "/main.go"

	out, err := exec.Command("./preprocessor/preprocessor", "proxy", "-o", dstDir+"/main.go", "--code-reference", "testReference", contractPath).CombinedOutput()
	if err != nil {
		return errors.Wrap(err, "can't generate proxy: "+string(out))
	}
	return nil
}

func buildContractPlugin(root string, name string) error {
	dstDir := root + "/plugins/"

	err := os.MkdirAll(dstDir, 0777)
	if err != nil {
		return err
	}

	origGoPath, err := testutil.ChangeGoPath(root)
	if err != nil {
		return err
	}
	defer os.Setenv("GOPATH", origGoPath) // nolint: errcheck

	//contractPath := root + "/src/contract/" + name + "/main.go"

	out, err := exec.Command("go", "build", "-buildmode=plugin", "-o", dstDir+"/"+name+".so", "contract/"+name).CombinedOutput()
	if err != nil {
		return errors.Wrap(err, "can't build contract: "+string(out))
	}
	return nil
}

func generateContractWrapper(root string, name string) error {
	contractPath := root + "/src/contract/" + name + "/main.go"
	wrapperPath := root + "/src/contract/" + name + "/main_wrapper.go"

	out, err := exec.Command("./preprocessor/preprocessor", "wrapper", "-o", wrapperPath, contractPath).CombinedOutput()
	if err != nil {
		return errors.Wrap(err, "can't generate wrapper for contract '"+name+"': "+string(out))
	}
	return nil
}

func buildContracts(root string, names ...string) error {
	for _, name := range names {
		err := generateContractProxy(root, name)
		if err != nil {
			return err
		}
		err = generateContractWrapper(root, name)
		if err != nil {
			return err
		}
	}

	for _, name := range names {
		err := buildContractPlugin(root, name)
		if err != nil {
			return err
		}
	}
	return nil
}

func suckInContracts(am *testutil.TestArtifactManager, root string, names ...string) {
	for _, name := range names {
		pluginBinary, err := ioutil.ReadFile(root + "/plugins/" + name + ".so")
		if err != nil {
			panic(err)
		}

		ref := core.String2Ref(name)
		am.Codes[ref] = &testutil.TestCodeDescriptor{ARef: &ref, ACode: pluginBinary}
	}
}

type testMessageRouter struct {
	plugin *GoPlugin
}

func (testMessageRouter) Start(components core.Components) error { return nil }
func (testMessageRouter) Stop() error                            { return nil }

func (r *testMessageRouter) Route(msg core.Message) (resp core.Response, err error) {
	ch := new(codec.CborHandle)

	var data []byte
	err = codec.NewEncoderBytes(&data, ch).Encode(
		&struct{}{},
	)
	if err != nil {
		return core.Response{}, err
	}
	resdata, reslist, err := r.plugin.CallMethod(core.String2Ref("two"), data, msg.Method, msg.Arguments)
	return core.Response{Data: resdata, Result: reslist, Error: err}, nil
}

func TestContractCallingContract(t *testing.T) {
	err := buildInciderCLI()
	if err != nil {
		t.Fatal(err)
	}

	err = buildPreprocessor()
	if err != nil {
		t.Fatal(err)
	}

	cwd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	defer os.Chdir(cwd) // nolint: errcheck

	tmpDir, err := ioutil.TempDir("", "test-")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir) // nolint: errcheck

	err = testutil.WriteFile(tmpDir+"/src/contract/one/", "main.go", contractOneCode)
	if err != nil {
		t.Fatal(err)
	}
	err = testutil.WriteFile(tmpDir+"/src/contract/two/", "main.go", contractTwoCode)
	if err != nil {
		t.Fatal(err)
	}

	err = buildContracts(tmpDir, "one", "two")
	if err != nil {
		t.Fatal(err)
	}

	insiderStorage := tmpDir + "/insider-storage/"

	err = os.MkdirAll(insiderStorage, 0777)
	if err != nil {
		t.Fatal(err)
	}

	mr := &testMessageRouter{}
	am := testutil.NewTestArtifactManager()

	gp, err := NewGoPlugin(
		configuration.Goplugin{
			MainListen:     "127.0.0.1:7778",
			RunnerListen:   "127.0.0.1:7777",
			RunnerCodePath: insiderStorage,
		},
		mr,
		am,
	)
	if err != nil {
		t.Fatal(err)
	}
	defer gp.Stop()

	mr.plugin = gp

	ch := new(codec.CborHandle)
	var data []byte
	err = codec.NewEncoderBytes(&data, ch).Encode(
		&struct{}{},
	)
	if err != nil {
		t.Fatal(err)
	}

	var argsSerialized []byte
	err = codec.NewEncoderBytes(&argsSerialized, ch).Encode(
		[]interface{}{"ins"},
	)
	if err != nil {
		panic(err)
	}

	suckInContracts(am, tmpDir, "one", "two")

	am.Objects[core.String2Ref("some")] = &testutil.TestObjectDescriptor{
		Data: []byte{},
		Code: am.Codes[core.String2Ref("two")],
	}

	_, res, err := gp.CallMethod(core.String2Ref("one"), data, "Hello", argsSerialized)
	if err != nil {
		panic(err)
	}

	var resParsed []interface{}
	err = codec.NewDecoderBytes(res, ch).Decode(&resParsed)
	if err != nil {
		t.Fatal(err)
	}

	if resParsed[0].(string) != "Hi, ins! Two said: Hello you too, ins" {
		t.Fatal("unexpected result")
	}
}
