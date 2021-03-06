// +build functest

/*
 *    Copyright 2019 Insolar Technologies
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

package functest

import (
	"testing"
	"time"

	"github.com/insolar/insolar/testutils"
	"github.com/stretchr/testify/require"
)

const times = 5

func checkBalanceFewTimes(t *testing.T, caller *user, ref string, expected int) {
	for i := 0; i < times; i++ {
		balance := getBalanceNoErr(t, caller, ref)
		if balance == expected {
			return
		}
		time.Sleep(time.Second)
	}
	t.Error("Received balance is not equal expected")
}

// TODO: uncomment after undoing of all transaction in failed request will be supported
func TestTransferMoney(t *testing.T) {
	firstMember := createMember(t, "Member1")
	secondMember := createMember(t, "Member2")
	// Skip validation of balance before/after transfer
	// oldFirstBalance := getBalanceNoErr(t, firstMember, firstMember.ref)
	// oldSecondBalance := getBalanceNoErr(t, secondMember, secondMember.ref)

	amount := 111

	_, err := signedRequest(firstMember, "Transfer", amount, secondMember.ref)
	require.NoError(t, err)

	// Skip validation of balance before/after transfer
	// checkBalanceFewTimes(t, secondMember, secondMember.ref, oldSecondBalance+amount)
	// newFirstBalance := getBalanceNoErr(t, firstMember, firstMember.ref)
	// require.Equal(t, oldFirstBalance-amount, newFirstBalance)
}

func TestTransferMoneyFromNotExist(t *testing.T) {
	firstMember := createMember(t, "Member1")
	firstMember.ref = testutils.RandomRef().String()

	secondMember := createMember(t, "Member2")
	oldSecondBalance := getBalanceNoErr(t, secondMember, secondMember.ref)

	amount := 111

	_, err := signedRequest(firstMember, "Transfer", amount, secondMember.ref)
	require.Contains(t, err.Error(), "not found")

	newSecondBalance := getBalanceNoErr(t, secondMember, secondMember.ref)
	require.Equal(t, oldSecondBalance, newSecondBalance)
}

func TestTransferMoneyToNotExist(t *testing.T) {
	firstMember := createMember(t, "Member1")
	oldFirstBalance := getBalanceNoErr(t, firstMember, firstMember.ref)

	amount := 111

	_, err := signedRequest(firstMember, "Transfer", amount, testutils.RandomRef().String())
	require.Contains(t, err.Error(), "[ Transfer ] Can't get implementation: [ GetDelegate ] on calling main API")

	newFirstBalance := getBalanceNoErr(t, firstMember, firstMember.ref)
	require.Equal(t, oldFirstBalance, newFirstBalance)
}

func TestTransferNegativeAmount(t *testing.T) {
	firstMember := createMember(t, "Member1")
	secondMember := createMember(t, "Member2")
	oldFirstBalance := getBalanceNoErr(t, firstMember, firstMember.ref)
	oldSecondBalance := getBalanceNoErr(t, secondMember, secondMember.ref)

	amount := -111

	_, err := signedRequest(firstMember, "Transfer", amount, secondMember.ref)
	require.Error(t, err)

	newFirstBalance := getBalanceNoErr(t, firstMember, firstMember.ref)
	newSecondBalance := getBalanceNoErr(t, secondMember, secondMember.ref)
	require.Equal(t, oldFirstBalance, newFirstBalance)
	require.Equal(t, oldSecondBalance, newSecondBalance)
}

// TODO: unskip test after undoing of all transaction in failed request will be supported
func TestTransferAllAmount(t *testing.T) {
	t.Skip()
	firstMember := createMember(t, "Member1")
	secondMember := createMember(t, "Member2")
	oldFirstBalance := getBalanceNoErr(t, firstMember, firstMember.ref)
	oldSecondBalance := getBalanceNoErr(t, secondMember, secondMember.ref)

	amount := oldFirstBalance

	_, err := signedRequest(firstMember, "Transfer", amount, secondMember.ref)
	require.NoError(t, err)

	checkBalanceFewTimes(t, secondMember, secondMember.ref, oldSecondBalance+oldFirstBalance)
	newFirstBalance := getBalanceNoErr(t, firstMember, firstMember.ref)
	require.Equal(t, 0, newFirstBalance)
}

func TestTransferMoreThanAvailableAmount(t *testing.T) {
	firstMember := createMember(t, "Member1")
	secondMember := createMember(t, "Member2")
	oldFirstBalance := getBalanceNoErr(t, firstMember, firstMember.ref)
	oldSecondBalance := getBalanceNoErr(t, secondMember, secondMember.ref)

	amount := oldFirstBalance + 100

	_, err := signedRequest(firstMember, "Transfer", amount, secondMember.ref)
	require.Contains(t, err.Error(), "[ Transfer ] Not enough balance for transfer: subtrahend must be smaller than minuend")

	newFirstBalance := getBalanceNoErr(t, firstMember, firstMember.ref)
	newSecondBalance := getBalanceNoErr(t, secondMember, secondMember.ref)
	require.Equal(t, oldFirstBalance, newFirstBalance)
	require.Equal(t, oldSecondBalance, newSecondBalance)
}

func TestTransferToMyself(t *testing.T) {
	member := createMember(t, "Member1")
	oldMemberBalance := getBalanceNoErr(t, member, member.ref)

	amount := 100

	_, err := signedRequest(member, "Transfer", amount, member.ref)
	require.Contains(t, err.Error(), "[ transferCall ] Recipient must be different from the sender")

	newMemberBalance := getBalanceNoErr(t, member, member.ref)
	require.Equal(t, oldMemberBalance, newMemberBalance)
}

// TODO: test to check overflow of balance
// TODO: check transfer zero amount

// TODO: uncomment after undoing of all transaction in failed request will be supported
func TestTransferTwoTimes(t *testing.T) {
	firstMember := createMember(t, "Member1")
	secondMember := createMember(t, "Member2")
	// Skip validation of balance before/after transfer
	// oldFirstBalance := getBalanceNoErr(t, firstMember, firstMember.ref)
	// oldSecondBalance := getBalanceNoErr(t, secondMember, secondMember.ref)

	amount := 100

	_, err := signedRequest(firstMember, "Transfer", amount, secondMember.ref)
	require.NoError(t, err)
	_, err = signedRequest(firstMember, "Transfer", amount, secondMember.ref)
	require.NoError(t, err)

	// Skip validation of balance before/after transfer
	// checkBalanceFewTimes(t, secondMember, secondMember.ref, oldSecondBalance+2*amount)
	// newFirstBalance := getBalanceNoErr(t, firstMember, firstMember.ref)
	// require.Equal(t, oldFirstBalance-2*amount, newFirstBalance)
}
