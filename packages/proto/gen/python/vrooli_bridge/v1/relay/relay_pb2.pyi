from google.protobuf.internal import containers as _containers
from google.protobuf.internal import enum_type_wrapper as _enum_type_wrapper
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Iterable as _Iterable, Mapping as _Mapping
from typing import ClassVar as _ClassVar, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class RelayCallOutcome(int, metaclass=_enum_type_wrapper.EnumTypeWrapper):
    __slots__ = ()
    RELAY_CALL_OUTCOME_UNSPECIFIED: _ClassVar[RelayCallOutcome]
    RELAY_CALL_OUTCOME_COMPLETED: _ClassVar[RelayCallOutcome]
    RELAY_CALL_OUTCOME_FAILED: _ClassVar[RelayCallOutcome]
    RELAY_CALL_OUTCOME_TERMINATED: _ClassVar[RelayCallOutcome]

class RelayReconcileState(int, metaclass=_enum_type_wrapper.EnumTypeWrapper):
    __slots__ = ()
    RELAY_RECONCILE_STATE_UNSPECIFIED: _ClassVar[RelayReconcileState]
    RELAY_RECONCILE_STATE_NOT_ADMITTED: _ClassVar[RelayReconcileState]
    RELAY_RECONCILE_STATE_SUBMITTED: _ClassVar[RelayReconcileState]
    RELAY_RECONCILE_STATE_COMPLETED: _ClassVar[RelayReconcileState]
    RELAY_RECONCILE_STATE_FAILED: _ClassVar[RelayReconcileState]
    RELAY_RECONCILE_STATE_OUTCOME_UNKNOWN: _ClassVar[RelayReconcileState]
RELAY_CALL_OUTCOME_UNSPECIFIED: RelayCallOutcome
RELAY_CALL_OUTCOME_COMPLETED: RelayCallOutcome
RELAY_CALL_OUTCOME_FAILED: RelayCallOutcome
RELAY_CALL_OUTCOME_TERMINATED: RelayCallOutcome
RELAY_RECONCILE_STATE_UNSPECIFIED: RelayReconcileState
RELAY_RECONCILE_STATE_NOT_ADMITTED: RelayReconcileState
RELAY_RECONCILE_STATE_SUBMITTED: RelayReconcileState
RELAY_RECONCILE_STATE_COMPLETED: RelayReconcileState
RELAY_RECONCILE_STATE_FAILED: RelayReconcileState
RELAY_RECONCILE_STATE_OUTCOME_UNKNOWN: RelayReconcileState

class RelayCallRequest(_message.Message):
    __slots__ = ("node_id", "scenario", "command", "args", "timeout_seconds", "max_response_bytes", "command_id")
    NODE_ID_FIELD_NUMBER: _ClassVar[int]
    SCENARIO_FIELD_NUMBER: _ClassVar[int]
    COMMAND_FIELD_NUMBER: _ClassVar[int]
    ARGS_FIELD_NUMBER: _ClassVar[int]
    TIMEOUT_SECONDS_FIELD_NUMBER: _ClassVar[int]
    MAX_RESPONSE_BYTES_FIELD_NUMBER: _ClassVar[int]
    COMMAND_ID_FIELD_NUMBER: _ClassVar[int]
    node_id: str
    scenario: str
    command: str
    args: _containers.RepeatedScalarFieldContainer[str]
    timeout_seconds: int
    max_response_bytes: int
    command_id: str
    def __init__(self, node_id: _Optional[str] = ..., scenario: _Optional[str] = ..., command: _Optional[str] = ..., args: _Optional[_Iterable[str]] = ..., timeout_seconds: _Optional[int] = ..., max_response_bytes: _Optional[int] = ..., command_id: _Optional[str] = ...) -> None: ...

class RelayCallResponse(_message.Message):
    __slots__ = ("correlation_id", "outcome", "data", "reason", "exit_code", "total_bytes", "command_id", "route", "route_cost_units", "route_latency_ms")
    CORRELATION_ID_FIELD_NUMBER: _ClassVar[int]
    OUTCOME_FIELD_NUMBER: _ClassVar[int]
    DATA_FIELD_NUMBER: _ClassVar[int]
    REASON_FIELD_NUMBER: _ClassVar[int]
    EXIT_CODE_FIELD_NUMBER: _ClassVar[int]
    TOTAL_BYTES_FIELD_NUMBER: _ClassVar[int]
    COMMAND_ID_FIELD_NUMBER: _ClassVar[int]
    ROUTE_FIELD_NUMBER: _ClassVar[int]
    ROUTE_COST_UNITS_FIELD_NUMBER: _ClassVar[int]
    ROUTE_LATENCY_MS_FIELD_NUMBER: _ClassVar[int]
    correlation_id: str
    outcome: RelayCallOutcome
    data: bytes
    reason: str
    exit_code: int
    total_bytes: int
    command_id: str
    route: str
    route_cost_units: int
    route_latency_ms: int
    def __init__(self, correlation_id: _Optional[str] = ..., outcome: _Optional[_Union[RelayCallOutcome, str]] = ..., data: _Optional[bytes] = ..., reason: _Optional[str] = ..., exit_code: _Optional[int] = ..., total_bytes: _Optional[int] = ..., command_id: _Optional[str] = ..., route: _Optional[str] = ..., route_cost_units: _Optional[int] = ..., route_latency_ms: _Optional[int] = ...) -> None: ...

class RelayReconcileRequest(_message.Message):
    __slots__ = ("node_id", "command_id")
    NODE_ID_FIELD_NUMBER: _ClassVar[int]
    COMMAND_ID_FIELD_NUMBER: _ClassVar[int]
    node_id: str
    command_id: str
    def __init__(self, node_id: _Optional[str] = ..., command_id: _Optional[str] = ...) -> None: ...

class RelayReconcileResponse(_message.Message):
    __slots__ = ("command_id", "state", "response", "reason")
    COMMAND_ID_FIELD_NUMBER: _ClassVar[int]
    STATE_FIELD_NUMBER: _ClassVar[int]
    RESPONSE_FIELD_NUMBER: _ClassVar[int]
    REASON_FIELD_NUMBER: _ClassVar[int]
    command_id: str
    state: RelayReconcileState
    response: RelayCallResponse
    reason: str
    def __init__(self, command_id: _Optional[str] = ..., state: _Optional[_Union[RelayReconcileState, str]] = ..., response: _Optional[_Union[RelayCallResponse, _Mapping]] = ..., reason: _Optional[str] = ...) -> None: ...
