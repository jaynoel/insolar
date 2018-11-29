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

package genesisdataprovider

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/insolar/insolar/component"
	"github.com/insolar/insolar/core"
	"github.com/insolar/insolar/core/reply"
	"github.com/insolar/insolar/instrumentation/inslogger"
	"github.com/insolar/insolar/testutils"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/require"
)

func mockContractRequesterWithError(t *testing.T) *testutils.ContractRequesterMock {
	contractRequesterMock := testutils.NewContractRequesterMock(t)
	contractRequesterMock.SendRequestFunc = func(p context.Context, p1 *core.RecordRef, p2 string, p3 []interface{}) (r core.Reply, r1 error) {
		return nil, errors.New("test reasons")
	}
	return contractRequesterMock
}

func mockContractRequester(t *testing.T, res core.Reply) *testutils.ContractRequesterMock {
	contractRequesterMock := testutils.NewContractRequesterMock(t)
	contractRequesterMock.SendRequestFunc = func(p context.Context, p1 *core.RecordRef, p2 string, p3 []interface{}) (r core.Reply, r1 error) {
		return res, nil
	}
	return contractRequesterMock
}

func mockCertificate(t *testing.T, rootDomainRef *core.RecordRef) *testutils.CertificateMock {
	certificateMock := testutils.NewCertificateMock(t)
	certificateMock.GetRootDomainReferenceFunc = func() (r *core.RecordRef) {
		return rootDomainRef
	}
	return certificateMock
}

func mockInfoResult(rootMemberRef core.RecordRef, nodeDomainRef core.RecordRef) core.Reply {
	result := map[string]interface{}{
		"root_member": rootMemberRef.String(),
		"node_domain": nodeDomainRef.String(),
	}
	resJSON, _ := json.Marshal(result)
	resSer, _ := core.MarshalArgs(resJSON, nil)
	return &reply.CallMethod{Result: resSer}
}

func TestNew(t *testing.T) {
	contractRequester := mockContractRequester(t, nil)
	certificate := mockCertificate(t, nil)

	result, err := New()

	cm := &component.Manager{}
	cm.Inject(contractRequester, certificate, result)

	require.NoError(t, err)
	require.Equal(t, result.Certificate, certificate)
	require.Equal(t, result.ContractRequester, contractRequester)
}

func TestGenesisDataProvider_setInfo(t *testing.T) {
	ctx := inslogger.TestContext(t)
	rootDomainRef := testutils.RandomRef()
	rootMemberRef := testutils.RandomRef()
	nodeDomainRef := testutils.RandomRef()

	infoRes := mockInfoResult(rootMemberRef, nodeDomainRef)

	gdp := &GenesisDataProvider{
		Certificate:       mockCertificate(t, &rootDomainRef),
		ContractRequester: mockContractRequester(t, infoRes),
	}

	err := gdp.setInfo(ctx)

	require.NoError(t, err)
	require.Equal(t, &rootDomainRef, gdp.rootDomainRef)
	require.Equal(t, &rootMemberRef, gdp.rootMemberRef)
	require.Equal(t, &nodeDomainRef, gdp.nodeDomainRef)
}

func TestGenesisDataProvider_setInfo_ErrorSendRequest(t *testing.T) {
	ctx := inslogger.TestContext(t)
	rootDomainRef := testutils.RandomRef()

	gdp := &GenesisDataProvider{
		Certificate:       mockCertificate(t, &rootDomainRef),
		ContractRequester: mockContractRequesterWithError(t),
	}

	err := gdp.setInfo(ctx)

	require.EqualError(t, err, "[ setInfo ] Can't send request: test reasons")
	require.Equal(t, &rootDomainRef, gdp.rootDomainRef)
}
