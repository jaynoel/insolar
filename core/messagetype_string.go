// Code generated by "stringer -type=MessageType"; DO NOT EDIT.

package core

import "strconv"

const _MessageType_name = "TypeCallMethodTypeCallConstructorTypeReturnResultsTypeExecutorResultsTypeValidateCaseBindTypeValidationResultsTypePendingFinishedTypeStillExecutingTypeGetCodeTypeGetObjectTypeGetDelegateTypeGetChildrenTypeUpdateObjectTypeRegisterChildTypeJetDropTypeSetRecordTypeValidateRecordTypeSetBlobTypeGetObjectIndexTypeGetPendingRequestsTypeHotRecordsTypeGetJetTypeAbandonedRequestsNotificationTypeGetRequestTypeGetPendingRequestIDTypeValidationCheckTypeHeavyStartStopTypeHeavyPayloadTypeBootstrapRequestTypeNodeSignRequest"

var _MessageType_index = [...]uint16{0, 14, 33, 50, 69, 89, 110, 129, 147, 158, 171, 186, 201, 217, 234, 245, 258, 276, 287, 305, 327, 341, 351, 384, 398, 421, 440, 458, 474, 494, 513}

func (i MessageType) String() string {
	if i >= MessageType(len(_MessageType_index)-1) {
		return "MessageType(" + strconv.FormatInt(int64(i), 10) + ")"
	}
	return _MessageType_name[_MessageType_index[i]:_MessageType_index[i+1]]
}
