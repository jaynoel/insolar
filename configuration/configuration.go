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

package configuration

import (
	"fmt"
	"reflect"
	"strings"

	"github.com/prometheus/common/log"
	"github.com/spf13/viper"
	"gopkg.in/yaml.v2"
)

// Configuration contains configuration params for all Insolar components
type Configuration struct {
	Host        HostNetwork
	Node        NodeNetwork
	Service     ServiceNetwork
	Ledger      Ledger
	Log         Log
	Stats       Stats
	LogicRunner LogicRunner
}

// Holder provides methods to manage configuration
type Holder struct {
	Configuration Configuration
	viper         *viper.Viper
}

// NewConfiguration creates new default configuration
func NewConfiguration() Configuration {
	cfg := Configuration{
		Host:        NewHostNetwork(),
		Node:        NewNodeNetwork(),
		Service:     NewServiceNetwork(),
		Ledger:      NewLedger(),
		Log:         NewLog(),
		Stats:       NewStats(),
		LogicRunner: NewLogicRunner(),
	}

	return cfg
}

// NewHolder creates new Holder with default configuration
func NewHolder() Holder {
	cfg := NewConfiguration()
	holder := Holder{Configuration: cfg, viper: viper.New()}

	holder.viper.SetConfigName(".insolar")
	holder.viper.AddConfigPath("$HOME/")
	holder.viper.AddConfigPath(".")
	holder.viper.SetConfigType("yml")

	holder.viper.SetDefault("insolar", cfg)

	holder.viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	holder.viper.SetEnvPrefix("insolar")
	return holder
}

// Load method reads configuration from default file path
func (c *Holder) Load() error {
	err := c.viper.ReadInConfig()
	if err != nil {
		return err
	}

	return c.viper.UnmarshalKey("insolar", &c.Configuration)
}

// LoadEnv overrides configuration with env variables
func (c *Holder) LoadEnv() error {
	// workaround for AutomaticEnv issue https://github.com/spf13/viper/issues/188
	bindEnvs(c.viper, c.Configuration)
	return c.viper.Unmarshal(&c.Configuration)
}

// LoadFromFile method reads configuration from particular file path
func (c *Holder) LoadFromFile(path string) error {
	c.viper.SetConfigFile(path)
	return c.Load()
}

// Save method writes configuration to default file path
func (c *Holder) Save() error {
	c.viper.Set("insolar", c.Configuration)
	return c.viper.WriteConfig()
}

// SaveAs method writes configuration to particular file path
func (c *Holder) SaveAs(path string) error {
	return c.viper.WriteConfigAs(path)
}

func bindEnvs(v *viper.Viper, iface interface{}, parts ...string) {
	ifv := reflect.ValueOf(iface)
	ift := reflect.TypeOf(iface)
	for i := 0; i < ift.NumField(); i++ {
		fieldv := ifv.Field(i)
		t := ift.Field(i)
		name := strings.ToLower(t.Name)
		tag, ok := t.Tag.Lookup("mapstructure")
		if ok {
			name = tag
		}
		path := append(parts, name)
		switch fieldv.Kind() {
		case reflect.Struct:
			bindEnvs(v, fieldv.Interface(), path...)
		default:
			err := v.BindEnv(strings.Join(path, "."))
			if err != nil {
				log.Warnln(err.Error())
			}
		}
	}
}

// ToString converts any configuration struct to yaml string
func ToString(in interface{}) string {
	d, err := yaml.Marshal(in)
	if err != nil {
		return fmt.Sprintf("error: %v", err)
	}
	return string(d)
}