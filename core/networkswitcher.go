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

package core

const (
	// NoNetworkState state means that nodes doesn`t match majority_rule
	NoNetworkState = iota
	// VoidNetworkState state means that nodes have not complete min_role_count rule for proper work
	VoidNetworkState
	// JetlessNetworkState state means that every Jet need proof completeness of stored data
	JetlessNetworkState
	// AuthorizationNetworkState state means that every node need to validate ActiveNodeList using NodeDomain
	AuthorizationNetworkState
	// CompleteNetworkState state means network is ok and ready for proper work
	CompleteNetworkState
)

// NetworkState type for bootstrapping process
type NetworkState int

// Switcher is a network FSM using for bootstrapping
type NetworkSwitcher interface {
	// GetState method returns current network state
	GetState() NetworkState
}
