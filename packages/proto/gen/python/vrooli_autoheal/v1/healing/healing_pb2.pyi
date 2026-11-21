import datetime

from google.protobuf import timestamp_pb2 as _timestamp_pb2
from google.protobuf.internal import containers as _containers
from google.protobuf.internal import enum_type_wrapper as _enum_type_wrapper
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Iterable as _Iterable, Mapping as _Mapping
from typing import ClassVar as _ClassVar, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class Outcome(int, metaclass=_enum_type_wrapper.EnumTypeWrapper):
    __slots__ = ()
    OUTCOME_UNSPECIFIED: _ClassVar[Outcome]
    OUTCOME_SUCCEEDED: _ClassVar[Outcome]
    OUTCOME_FAILED: _ClassVar[Outcome]
    OUTCOME_TIMED_OUT: _ClassVar[Outcome]
    OUTCOME_SKIPPED: _ClassVar[Outcome]
OUTCOME_UNSPECIFIED: Outcome
OUTCOME_SUCCEEDED: Outcome
OUTCOME_FAILED: Outcome
OUTCOME_TIMED_OUT: Outcome
OUTCOME_SKIPPED: Outcome

class HealOutcome(_message.Message):
    __slots__ = ("check_id", "action_id", "outcome", "message", "observed_at", "duration_ms")
    CHECK_ID_FIELD_NUMBER: _ClassVar[int]
    ACTION_ID_FIELD_NUMBER: _ClassVar[int]
    OUTCOME_FIELD_NUMBER: _ClassVar[int]
    MESSAGE_FIELD_NUMBER: _ClassVar[int]
    OBSERVED_AT_FIELD_NUMBER: _ClassVar[int]
    DURATION_MS_FIELD_NUMBER: _ClassVar[int]
    check_id: str
    action_id: str
    outcome: Outcome
    message: str
    observed_at: _timestamp_pb2.Timestamp
    duration_ms: int
    def __init__(self, check_id: _Optional[str] = ..., action_id: _Optional[str] = ..., outcome: _Optional[_Union[Outcome, str]] = ..., message: _Optional[str] = ..., observed_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., duration_ms: _Optional[int] = ...) -> None: ...

class HealEpisode(_message.Message):
    __slots__ = ("id", "check_id", "trigger", "outcome", "attempts", "started_at", "completed_at", "evidence_json")
    ID_FIELD_NUMBER: _ClassVar[int]
    CHECK_ID_FIELD_NUMBER: _ClassVar[int]
    TRIGGER_FIELD_NUMBER: _ClassVar[int]
    OUTCOME_FIELD_NUMBER: _ClassVar[int]
    ATTEMPTS_FIELD_NUMBER: _ClassVar[int]
    STARTED_AT_FIELD_NUMBER: _ClassVar[int]
    COMPLETED_AT_FIELD_NUMBER: _ClassVar[int]
    EVIDENCE_JSON_FIELD_NUMBER: _ClassVar[int]
    id: str
    check_id: str
    trigger: str
    outcome: Outcome
    attempts: int
    started_at: _timestamp_pb2.Timestamp
    completed_at: _timestamp_pb2.Timestamp
    evidence_json: str
    def __init__(self, id: _Optional[str] = ..., check_id: _Optional[str] = ..., trigger: _Optional[str] = ..., outcome: _Optional[_Union[Outcome, str]] = ..., attempts: _Optional[int] = ..., started_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., completed_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., evidence_json: _Optional[str] = ...) -> None: ...

class ListOutcomesRequest(_message.Message):
    __slots__ = ("window_hours",)
    WINDOW_HOURS_FIELD_NUMBER: _ClassVar[int]
    window_hours: int
    def __init__(self, window_hours: _Optional[int] = ...) -> None: ...

class ListOutcomesResponse(_message.Message):
    __slots__ = ("outcomes",)
    OUTCOMES_FIELD_NUMBER: _ClassVar[int]
    outcomes: _containers.RepeatedCompositeFieldContainer[HealOutcome]
    def __init__(self, outcomes: _Optional[_Iterable[_Union[HealOutcome, _Mapping]]] = ...) -> None: ...

class GetEpisodesRequest(_message.Message):
    __slots__ = ("check_id", "limit")
    CHECK_ID_FIELD_NUMBER: _ClassVar[int]
    LIMIT_FIELD_NUMBER: _ClassVar[int]
    check_id: str
    limit: int
    def __init__(self, check_id: _Optional[str] = ..., limit: _Optional[int] = ...) -> None: ...

class GetEpisodesResponse(_message.Message):
    __slots__ = ("episodes",)
    EPISODES_FIELD_NUMBER: _ClassVar[int]
    episodes: _containers.RepeatedCompositeFieldContainer[HealEpisode]
    def __init__(self, episodes: _Optional[_Iterable[_Union[HealEpisode, _Mapping]]] = ...) -> None: ...

class GetHistoryRequest(_message.Message):
    __slots__ = ("check_id", "limit")
    CHECK_ID_FIELD_NUMBER: _ClassVar[int]
    LIMIT_FIELD_NUMBER: _ClassVar[int]
    check_id: str
    limit: int
    def __init__(self, check_id: _Optional[str] = ..., limit: _Optional[int] = ...) -> None: ...

class GetHistoryResponse(_message.Message):
    __slots__ = ("outcomes",)
    OUTCOMES_FIELD_NUMBER: _ClassVar[int]
    outcomes: _containers.RepeatedCompositeFieldContainer[HealOutcome]
    def __init__(self, outcomes: _Optional[_Iterable[_Union[HealOutcome, _Mapping]]] = ...) -> None: ...

class GetReadinessRequest(_message.Message):
    __slots__ = ("limit",)
    LIMIT_FIELD_NUMBER: _ClassVar[int]
    limit: int
    def __init__(self, limit: _Optional[int] = ...) -> None: ...

class ReadinessElement(_message.Message):
    __slots__ = ("check_id", "status", "ready_at", "latency_ms", "starter", "evidence")
    CHECK_ID_FIELD_NUMBER: _ClassVar[int]
    STATUS_FIELD_NUMBER: _ClassVar[int]
    READY_AT_FIELD_NUMBER: _ClassVar[int]
    LATENCY_MS_FIELD_NUMBER: _ClassVar[int]
    STARTER_FIELD_NUMBER: _ClassVar[int]
    EVIDENCE_FIELD_NUMBER: _ClassVar[int]
    check_id: str
    status: str
    ready_at: _timestamp_pb2.Timestamp
    latency_ms: int
    starter: str
    evidence: str
    def __init__(self, check_id: _Optional[str] = ..., status: _Optional[str] = ..., ready_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., latency_ms: _Optional[int] = ..., starter: _Optional[str] = ..., evidence: _Optional[str] = ...) -> None: ...

class BootRecoveryPrecondition(_message.Message):
    __slots__ = ("name", "state", "reason")
    NAME_FIELD_NUMBER: _ClassVar[int]
    STATE_FIELD_NUMBER: _ClassVar[int]
    REASON_FIELD_NUMBER: _ClassVar[int]
    name: str
    state: str
    reason: str
    def __init__(self, name: _Optional[str] = ..., state: _Optional[str] = ..., reason: _Optional[str] = ...) -> None: ...

class BootRecovery(_message.Message):
    __slots__ = ("status", "preconditions", "evaluated_at", "remediation", "message")
    STATUS_FIELD_NUMBER: _ClassVar[int]
    PRECONDITIONS_FIELD_NUMBER: _ClassVar[int]
    EVALUATED_AT_FIELD_NUMBER: _ClassVar[int]
    REMEDIATION_FIELD_NUMBER: _ClassVar[int]
    MESSAGE_FIELD_NUMBER: _ClassVar[int]
    status: str
    preconditions: _containers.RepeatedCompositeFieldContainer[BootRecoveryPrecondition]
    evaluated_at: _timestamp_pb2.Timestamp
    remediation: str
    message: str
    def __init__(self, status: _Optional[str] = ..., preconditions: _Optional[_Iterable[_Union[BootRecoveryPrecondition, _Mapping]]] = ..., evaluated_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., remediation: _Optional[str] = ..., message: _Optional[str] = ...) -> None: ...

class GetReadinessResponse(_message.Message):
    __slots__ = ("available", "unavailable_reason", "boot_id", "process_started_at", "elements", "episodes", "computed_at", "boot_recovery")
    AVAILABLE_FIELD_NUMBER: _ClassVar[int]
    UNAVAILABLE_REASON_FIELD_NUMBER: _ClassVar[int]
    BOOT_ID_FIELD_NUMBER: _ClassVar[int]
    PROCESS_STARTED_AT_FIELD_NUMBER: _ClassVar[int]
    ELEMENTS_FIELD_NUMBER: _ClassVar[int]
    EPISODES_FIELD_NUMBER: _ClassVar[int]
    COMPUTED_AT_FIELD_NUMBER: _ClassVar[int]
    BOOT_RECOVERY_FIELD_NUMBER: _ClassVar[int]
    available: bool
    unavailable_reason: str
    boot_id: str
    process_started_at: _timestamp_pb2.Timestamp
    elements: _containers.RepeatedCompositeFieldContainer[ReadinessElement]
    episodes: _containers.RepeatedCompositeFieldContainer[HealEpisode]
    computed_at: _timestamp_pb2.Timestamp
    boot_recovery: BootRecovery
    def __init__(self, available: _Optional[bool] = ..., unavailable_reason: _Optional[str] = ..., boot_id: _Optional[str] = ..., process_started_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., elements: _Optional[_Iterable[_Union[ReadinessElement, _Mapping]]] = ..., episodes: _Optional[_Iterable[_Union[HealEpisode, _Mapping]]] = ..., computed_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., boot_recovery: _Optional[_Union[BootRecovery, _Mapping]] = ...) -> None: ...
