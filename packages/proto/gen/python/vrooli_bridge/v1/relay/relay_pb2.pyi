from google.protobuf.internal import containers as _containers
from google.protobuf.internal import enum_type_wrapper as _enum_type_wrapper
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Iterable as _Iterable
from typing import ClassVar as _ClassVar, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class RelayCallOutcome(int, metaclass=_enum_type_wrapper.EnumTypeWrapper):
    __slots__ = ()
    RELAY_CALL_OUTCOME_UNSPECIFIED: _ClassVar[RelayCallOutcome]
    RELAY_CALL_OUTCOME_COMPLETED: _ClassVar[RelayCallOutcome]
    RELAY_CALL_OUTCOME_FAILED: _ClassVar[RelayCallOutcome]
    RELAY_CALL_OUTCOME_TERMINATED: _ClassVar[RelayCallOutcome]
RELAY_CALL_OUTCOME_UNSPECIFIED: RelayCallOutcome
RELAY_CALL_OUTCOME_COMPLETED: RelayCallOutcome
RELAY_CALL_OUTCOME_FAILED: RelayCallOutcome
RELAY_CALL_OUTCOME_TERMINATED: RelayCallOutcome

class RelayCallRequest(_message.Message):
    __slots__ = ("node_id", "scenario", "command", "args", "timeout_seconds", "max_response_bytes")
    NODE_ID_FIELD_NUMBER: _ClassVar[int]
    SCENARIO_FIELD_NUMBER: _ClassVar[int]
    COMMAND_FIELD_NUMBER: _ClassVar[int]
    ARGS_FIELD_NUMBER: _ClassVar[int]
    TIMEOUT_SECONDS_FIELD_NUMBER: _ClassVar[int]
    MAX_RESPONSE_BYTES_FIELD_NUMBER: _ClassVar[int]
    node_id: str
    scenario: str
    command: str
    args: _containers.RepeatedScalarFieldContainer[str]
    timeout_seconds: int
    max_response_bytes: int
    def __init__(self, node_id: _Optional[str] = ..., scenario: _Optional[str] = ..., command: _Optional[str] = ..., args: _Optional[_Iterable[str]] = ..., timeout_seconds: _Optional[int] = ..., max_response_bytes: _Optional[int] = ...) -> None: ...

class RelayCallResponse(_message.Message):
    __slots__ = ("correlation_id", "outcome", "data", "reason", "exit_code", "total_bytes")
    CORRELATION_ID_FIELD_NUMBER: _ClassVar[int]
    OUTCOME_FIELD_NUMBER: _ClassVar[int]
    DATA_FIELD_NUMBER: _ClassVar[int]
    REASON_FIELD_NUMBER: _ClassVar[int]
    EXIT_CODE_FIELD_NUMBER: _ClassVar[int]
    TOTAL_BYTES_FIELD_NUMBER: _ClassVar[int]
    correlation_id: str
    outcome: RelayCallOutcome
    data: bytes
    reason: str
    exit_code: int
    total_bytes: int
    def __init__(self, correlation_id: _Optional[str] = ..., outcome: _Optional[_Union[RelayCallOutcome, str]] = ..., data: _Optional[bytes] = ..., reason: _Optional[str] = ..., exit_code: _Optional[int] = ..., total_bytes: _Optional[int] = ...) -> None: ...
