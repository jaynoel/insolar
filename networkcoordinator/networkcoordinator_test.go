/*
 * The Clear BSD License
 *
 * Copyright (c) 2019 Insolar Technologies
 *
 * All rights reserved.
 *
 * Redistribution and use in source and binary forms, with or without modification, are permitted (subject to the limitations in the disclaimer below) provided that the following conditions are met:
 *
 *  Redistributions of source code must retain the above copyright notice, this list of conditions and the following disclaimer.
 *  Redistributions in binary form must reproduce the above copyright notice, this list of conditions and the following disclaimer in the documentation and/or other materials provided with the distribution.
 *  Neither the name of Insolar Technologies nor the names of its contributors may be used to endorse or promote products derived from this software without specific prior written permission.
 *
 * NO EXPRESS OR IMPLIED LICENSES TO ANY PARTY'S PATENT RIGHTS ARE GRANTED BY THIS LICENSE. THIS SOFTWARE IS PROVIDED BY THE COPYRIGHT HOLDERS AND CONTRIBUTORS "AS IS" AND ANY EXPRESS OR IMPLIED WARRANTIES, INCLUDING, BUT NOT LIMITED TO, THE IMPLIED WARRANTIES OF MERCHANTABILITY AND FITNESS FOR A PARTICULAR PURPOSE ARE DISCLAIMED. IN NO EVENT SHALL THE COPYRIGHT HOLDER OR CONTRIBUTORS BE LIABLE FOR ANY DIRECT, INDIRECT, INCIDENTAL, SPECIAL, EXEMPLARY, OR CONSEQUENTIAL DAMAGES (INCLUDING, BUT NOT LIMITED TO, PROCUREMENT OF SUBSTITUTE GOODS OR SERVICES; LOSS OF USE, DATA, OR PROFITS; OR BUSINESS INTERRUPTION) HOWEVER CAUSED AND ON ANY THEORY OF LIABILITY, WHETHER IN CONTRACT, STRICT LIABILITY, OR TORT (INCLUDING NEGLIGENCE OR OTHERWISE) ARISING IN ANY WAY OUT OF THE USE OF THIS SOFTWARE, EVEN IF ADVISED OF THE POSSIBILITY OF SUCH DAMAGE.
 *
 */

package networkcoordinator

import (
	"context"
	"testing"

	"github.com/insolar/insolar/certificate"
	"github.com/insolar/insolar/component"
	"github.com/insolar/insolar/core"
	"github.com/insolar/insolar/core/message"
	"github.com/insolar/insolar/core/reply"
	"github.com/insolar/insolar/testutils"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/require"
)

func TestNewNetworkCoordinator(t *testing.T) {
	certificateManager := testutils.NewCertificateManagerMock(t)
	networkSwitcher := testutils.NewNetworkSwitcherMock(t)
	contractRequester := testutils.NewContractRequesterMock(t)
	messageBus := testutils.NewMessageBusMock(t)
	cs := testutils.NewCryptographyServiceMock(t)
	ps := testutils.NewPulseStorageMock(t)

	nc, err := New()
	require.NoError(t, err)
	require.Equal(t, &NetworkCoordinator{}, nc)

	cm := &component.Manager{}
	cm.Inject(certificateManager, networkSwitcher, contractRequester, messageBus, cs, ps, nc)
	require.Equal(t, certificateManager, nc.CertificateManager)
	require.Equal(t, networkSwitcher, nc.NetworkSwitcher)
	require.Equal(t, contractRequester, nc.ContractRequester)
	require.Equal(t, messageBus, nc.MessageBus)
	require.Equal(t, cs, nc.CS)
	require.Equal(t, ps, nc.PS)
}

func TestNetworkCoordinator_Start(t *testing.T) {
	nc, err := New()
	require.NoError(t, err)
	nc.MessageBus = mockMessageBus(t, true, nil, nil)
	ctx := context.Background()
	err = nc.Start(ctx)
	require.NoError(t, err)
	require.NotNil(t, nc.realCoordinator)
	require.NotNil(t, nc.zeroCoordinator)
}

func TestNetworkCoordinator_GetCoordinator_Zero(t *testing.T) {
	nc, err := New()
	require.NoError(t, err)
	ns := testutils.NewNetworkSwitcherMock(t)
	ns.GetStateFunc = func() core.NetworkState {
		return core.NoNetworkState
	}
	nc.NetworkSwitcher = ns
	nc.MessageBus = mockMessageBus(t, true, nil, nil)
	ctx := context.Background()
	nc.Start(ctx)
	crd := nc.getCoordinator()
	require.Equal(t, nc.zeroCoordinator, crd)
}

func TestNetworkCoordinator_GetCoordinator_Real(t *testing.T) {
	nc, err := New()
	require.NoError(t, err)
	ns := testutils.NewNetworkSwitcherMock(t)
	ns.GetStateFunc = func() core.NetworkState {
		return core.CompleteNetworkState
	}
	nc.NetworkSwitcher = ns
	nc.MessageBus = mockMessageBus(t, true, nil, nil)
	ctx := context.Background()
	nc.Start(ctx)
	crd := nc.getCoordinator()
	require.Equal(t, nc.realCoordinator, crd)
}

func mockReply(t *testing.T) []byte {
	node, err := core.MarshalArgs(struct {
		PublicKey string
		Role      core.StaticRole
	}{
		PublicKey: "test_node_public_key",
		Role:      core.StaticRoleVirtual,
	}, nil)
	require.NoError(t, err)
	return []byte(node)
}

func mockMessageBus(t *testing.T, ok bool, ref *core.RecordRef, discovery *core.RecordRef) *testutils.MessageBusMock {
	mb := testutils.NewMessageBusMock(t)
	mb.MustRegisterFunc = func(p core.MessageType, handler core.MessageHandler) {
		require.Equal(t, p, core.TypeNodeSignRequest)
	}
	mb.SendFunc = func(p context.Context, msg core.Message, options *core.MessageSendOptions) (core.Reply, error) {
		require.Equal(t, ref, msg.(*message.NodeSignPayload).NodeRef)
		require.Equal(t, discovery, options.Receiver)
		if ok {
			return &reply.NodeSign{
				Sign: []byte("test_sig"),
			}, nil
		}
		return nil, errors.New("test_error")
	}
	return mb
}

func mockCertificateManager(t *testing.T, certNodeRef *core.RecordRef, discoveryNodeRef *core.RecordRef, unsignCertOk bool) *testutils.CertificateManagerMock {
	cm := testutils.NewCertificateManagerMock(t)
	cm.GetCertificateFunc = func() core.Certificate {
		return &certificate.Certificate{
			AuthorizationCertificate: certificate.AuthorizationCertificate{
				PublicKey: "test_public_key",
				Reference: certNodeRef.String(),
				Role:      "virtual",
			},
			MajorityRule: 0,
			BootstrapNodes: []certificate.BootstrapNode{
				certificate.BootstrapNode{
					NodeRef:     discoveryNodeRef.String(),
					PublicKey:   "test_discovery_public_key",
					Host:        "test_discovery_host",
					NetworkSign: []byte("test_network_sign"),
				},
			},
		}
	}
	cm.NewUnsignedCertificateFunc = func(key string, role string, nodeRef string) (core.Certificate, error) {
		require.Equal(t, "test_node_public_key", key)
		require.Equal(t, "virtual", role)

		if unsignCertOk {
			return &certificate.Certificate{
				AuthorizationCertificate: certificate.AuthorizationCertificate{
					PublicKey: key,
					Reference: nodeRef,
					Role:      role,
				},
				RootDomainReference: "test_root_domain_ref",
				MajorityRule:        0,
				PulsarPublicKeys:    []string{},
				BootstrapNodes: []certificate.BootstrapNode{
					certificate.BootstrapNode{
						PublicKey:   "test_discovery_public_key",
						Host:        "test_discovery_host",
						NetworkSign: []byte("test_network_sign"),
						NodeRef:     discoveryNodeRef.String(),
					},
				},
			}, nil
		}
		return nil, errors.New("test_error")
	}
	return cm
}

func mockCryptographyService(t *testing.T, ok bool) core.CryptographyService {
	cs := testutils.NewCryptographyServiceMock(t)
	cs.SignFunc = func(data []byte) (*core.Signature, error) {
		if ok {
			sig := core.SignatureFromBytes([]byte("test_sig"))
			return &sig, nil
		}
		return nil, errors.New("test_error")
	}
	return cs
}
