import datetime

from buf.validate import validate_pb2 as _validate_pb2
from google.protobuf import timestamp_pb2 as _timestamp_pb2
from setup.v1 import operator_input_pb2 as _operator_input_pb2
from google.protobuf.internal import containers as _containers
from google.protobuf.internal import enum_type_wrapper as _enum_type_wrapper
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Iterable as _Iterable, Mapping as _Mapping
from typing import ClassVar as _ClassVar, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class AnswerOutcomeStatus(int, metaclass=_enum_type_wrapper.EnumTypeWrapper):
    __slots__ = ()
    ANSWER_OUTCOME_STATUS_UNSPECIFIED: _ClassVar[AnswerOutcomeStatus]
    ANSWER_OUTCOME_STATUS_ACCEPTED: _ClassVar[AnswerOutcomeStatus]
    ANSWER_OUTCOME_STATUS_APPLIED: _ClassVar[AnswerOutcomeStatus]
    ANSWER_OUTCOME_STATUS_DECLINED: _ClassVar[AnswerOutcomeStatus]
ANSWER_OUTCOME_STATUS_UNSPECIFIED: AnswerOutcomeStatus
ANSWER_OUTCOME_STATUS_ACCEPTED: AnswerOutcomeStatus
ANSWER_OUTCOME_STATUS_APPLIED: AnswerOutcomeStatus
ANSWER_OUTCOME_STATUS_DECLINED: AnswerOutcomeStatus

class ListOperatorInputsRequest(_message.Message):
    __slots__ = ("target",)
    TARGET_FIELD_NUMBER: _ClassVar[int]
    target: str
    def __init__(self, target: _Optional[str] = ...) -> None: ...

class ListOperatorInputsResponse(_message.Message):
    __slots__ = ("version", "updated_at", "requests", "contract_version")
    VERSION_FIELD_NUMBER: _ClassVar[int]
    UPDATED_AT_FIELD_NUMBER: _ClassVar[int]
    REQUESTS_FIELD_NUMBER: _ClassVar[int]
    CONTRACT_VERSION_FIELD_NUMBER: _ClassVar[int]
    version: int
    updated_at: _timestamp_pb2.Timestamp
    requests: _containers.RepeatedCompositeFieldContainer[_operator_input_pb2.OperatorInputRequest]
    contract_version: str
    def __init__(self, version: _Optional[int] = ..., updated_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., requests: _Optional[_Iterable[_Union[_operator_input_pb2.OperatorInputRequest, _Mapping]]] = ..., contract_version: _Optional[str] = ...) -> None: ...

class Answer(_message.Message):
    __slots__ = ("request_id", "value", "declined")
    REQUEST_ID_FIELD_NUMBER: _ClassVar[int]
    VALUE_FIELD_NUMBER: _ClassVar[int]
    DECLINED_FIELD_NUMBER: _ClassVar[int]
    request_id: str
    value: str
    declined: bool
    def __init__(self, request_id: _Optional[str] = ..., value: _Optional[str] = ..., declined: _Optional[bool] = ...) -> None: ...

class ResolveOperatorInputsRequest(_message.Message):
    __slots__ = ("target", "expected_revision", "answers")
    TARGET_FIELD_NUMBER: _ClassVar[int]
    EXPECTED_REVISION_FIELD_NUMBER: _ClassVar[int]
    ANSWERS_FIELD_NUMBER: _ClassVar[int]
    target: str
    expected_revision: str
    answers: _containers.RepeatedCompositeFieldContainer[Answer]
    def __init__(self, target: _Optional[str] = ..., expected_revision: _Optional[str] = ..., answers: _Optional[_Iterable[_Union[Answer, _Mapping]]] = ...) -> None: ...

class AnswerOutcome(_message.Message):
    __slots__ = ("request_id", "status", "remediation")
    REQUEST_ID_FIELD_NUMBER: _ClassVar[int]
    STATUS_FIELD_NUMBER: _ClassVar[int]
    REMEDIATION_FIELD_NUMBER: _ClassVar[int]
    request_id: str
    status: AnswerOutcomeStatus
    remediation: str
    def __init__(self, request_id: _Optional[str] = ..., status: _Optional[_Union[AnswerOutcomeStatus, str]] = ..., remediation: _Optional[str] = ...) -> None: ...

class ResolveOperatorInputsResponse(_message.Message):
    __slots__ = ("configuration_pending", "outcomes")
    CONFIGURATION_PENDING_FIELD_NUMBER: _ClassVar[int]
    OUTCOMES_FIELD_NUMBER: _ClassVar[int]
    configuration_pending: bool
    outcomes: _containers.RepeatedCompositeFieldContainer[AnswerOutcome]
    def __init__(self, configuration_pending: _Optional[bool] = ..., outcomes: _Optional[_Iterable[_Union[AnswerOutcome, _Mapping]]] = ...) -> None: ...
