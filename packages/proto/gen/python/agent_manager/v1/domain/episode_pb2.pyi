from google.protobuf import struct_pb2 as _struct_pb2
from google.protobuf.internal import containers as _containers
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Iterable as _Iterable, Mapping as _Mapping
from typing import ClassVar as _ClassVar, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class GetCrossScenarioLedgerRequest(_message.Message):
    __slots__ = ("run_id", "with_projections")
    RUN_ID_FIELD_NUMBER: _ClassVar[int]
    WITH_PROJECTIONS_FIELD_NUMBER: _ClassVar[int]
    run_id: str
    with_projections: bool
    def __init__(self, run_id: _Optional[str] = ..., with_projections: _Optional[bool] = ...) -> None: ...

class Availability(_message.Message):
    __slots__ = ("state", "reason")
    STATE_FIELD_NUMBER: _ClassVar[int]
    REASON_FIELD_NUMBER: _ClassVar[int]
    state: str
    reason: str
    def __init__(self, state: _Optional[str] = ..., reason: _Optional[str] = ...) -> None: ...

class CrossScenarioCall(_message.Message):
    __slots__ = ("occurred_at", "target_scenario", "operation", "outcome", "status_code", "duration_ms", "receipt_event_id", "verified", "projection", "projection_drop_count", "policy_version")
    OCCURRED_AT_FIELD_NUMBER: _ClassVar[int]
    TARGET_SCENARIO_FIELD_NUMBER: _ClassVar[int]
    OPERATION_FIELD_NUMBER: _ClassVar[int]
    OUTCOME_FIELD_NUMBER: _ClassVar[int]
    STATUS_CODE_FIELD_NUMBER: _ClassVar[int]
    DURATION_MS_FIELD_NUMBER: _ClassVar[int]
    RECEIPT_EVENT_ID_FIELD_NUMBER: _ClassVar[int]
    VERIFIED_FIELD_NUMBER: _ClassVar[int]
    PROJECTION_FIELD_NUMBER: _ClassVar[int]
    PROJECTION_DROP_COUNT_FIELD_NUMBER: _ClassVar[int]
    POLICY_VERSION_FIELD_NUMBER: _ClassVar[int]
    occurred_at: str
    target_scenario: str
    operation: str
    outcome: str
    status_code: int
    duration_ms: int
    receipt_event_id: str
    verified: bool
    projection: _struct_pb2.Struct
    projection_drop_count: int
    policy_version: str
    def __init__(self, occurred_at: _Optional[str] = ..., target_scenario: _Optional[str] = ..., operation: _Optional[str] = ..., outcome: _Optional[str] = ..., status_code: _Optional[int] = ..., duration_ms: _Optional[int] = ..., receipt_event_id: _Optional[str] = ..., verified: _Optional[bool] = ..., projection: _Optional[_Union[_struct_pb2.Struct, _Mapping]] = ..., projection_drop_count: _Optional[int] = ..., policy_version: _Optional[str] = ...) -> None: ...

class LedgerTargetRollup(_message.Message):
    __slots__ = ("target_scenario", "calls", "failures", "total_duration_ms", "median_duration_ms")
    TARGET_SCENARIO_FIELD_NUMBER: _ClassVar[int]
    CALLS_FIELD_NUMBER: _ClassVar[int]
    FAILURES_FIELD_NUMBER: _ClassVar[int]
    TOTAL_DURATION_MS_FIELD_NUMBER: _ClassVar[int]
    MEDIAN_DURATION_MS_FIELD_NUMBER: _ClassVar[int]
    target_scenario: str
    calls: int
    failures: int
    total_duration_ms: int
    median_duration_ms: int
    def __init__(self, target_scenario: _Optional[str] = ..., calls: _Optional[int] = ..., failures: _Optional[int] = ..., total_duration_ms: _Optional[int] = ..., median_duration_ms: _Optional[int] = ...) -> None: ...

class GetCrossScenarioLedgerResponse(_message.Message):
    __slots__ = ("ledger_availability", "projection_availability", "target_rollups", "calls")
    LEDGER_AVAILABILITY_FIELD_NUMBER: _ClassVar[int]
    PROJECTION_AVAILABILITY_FIELD_NUMBER: _ClassVar[int]
    TARGET_ROLLUPS_FIELD_NUMBER: _ClassVar[int]
    CALLS_FIELD_NUMBER: _ClassVar[int]
    ledger_availability: Availability
    projection_availability: Availability
    target_rollups: _containers.RepeatedCompositeFieldContainer[LedgerTargetRollup]
    calls: _containers.RepeatedCompositeFieldContainer[CrossScenarioCall]
    def __init__(self, ledger_availability: _Optional[_Union[Availability, _Mapping]] = ..., projection_availability: _Optional[_Union[Availability, _Mapping]] = ..., target_rollups: _Optional[_Iterable[_Union[LedgerTargetRollup, _Mapping]]] = ..., calls: _Optional[_Iterable[_Union[CrossScenarioCall, _Mapping]]] = ...) -> None: ...

class ImportTranscriptRequest(_message.Message):
    __slots__ = ("path", "runner_type", "label", "source_harness", "source_session_id")
    PATH_FIELD_NUMBER: _ClassVar[int]
    RUNNER_TYPE_FIELD_NUMBER: _ClassVar[int]
    LABEL_FIELD_NUMBER: _ClassVar[int]
    SOURCE_HARNESS_FIELD_NUMBER: _ClassVar[int]
    SOURCE_SESSION_ID_FIELD_NUMBER: _ClassVar[int]
    path: str
    runner_type: str
    label: str
    source_harness: str
    source_session_id: str
    def __init__(self, path: _Optional[str] = ..., runner_type: _Optional[str] = ..., label: _Optional[str] = ..., source_harness: _Optional[str] = ..., source_session_id: _Optional[str] = ...) -> None: ...

class ImportTranscriptResponse(_message.Message):
    __slots__ = ("run_id", "status", "execution_mode")
    RUN_ID_FIELD_NUMBER: _ClassVar[int]
    STATUS_FIELD_NUMBER: _ClassVar[int]
    EXECUTION_MODE_FIELD_NUMBER: _ClassVar[int]
    run_id: str
    status: str
    execution_mode: str
    def __init__(self, run_id: _Optional[str] = ..., status: _Optional[str] = ..., execution_mode: _Optional[str] = ...) -> None: ...
