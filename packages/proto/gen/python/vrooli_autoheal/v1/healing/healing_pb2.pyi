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
