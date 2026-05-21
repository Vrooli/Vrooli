import datetime

from architecture_cartographer.v1.conflicts import conflicts_pb2 as _conflicts_pb2
from architecture_cartographer.v1.signals import signals_pb2 as _signals_pb2
from google.protobuf import timestamp_pb2 as _timestamp_pb2
from google.protobuf.internal import containers as _containers
from google.protobuf.internal import enum_type_wrapper as _enum_type_wrapper
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Iterable as _Iterable, Mapping as _Mapping
from typing import ClassVar as _ClassVar, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class EventKind(int, metaclass=_enum_type_wrapper.EnumTypeWrapper):
    __slots__ = ()
    EVENT_KIND_UNSPECIFIED: _ClassVar[EventKind]
    EVENT_KIND_CONFLICT_DETECTED: _ClassVar[EventKind]
    EVENT_KIND_CONFLICT_ASSIGNED: _ClassVar[EventKind]
    EVENT_KIND_CONFLICT_RESOLVED: _ClassVar[EventKind]
    EVENT_KIND_CONFLICT_REOPENED: _ClassVar[EventKind]
    EVENT_KIND_CONFLICT_FORCE_RESOLVED: _ClassVar[EventKind]
    EVENT_KIND_VERDICT_PRODUCED: _ClassVar[EventKind]
    EVENT_KIND_PLACEMENT_AUTO: _ClassVar[EventKind]
    EVENT_KIND_PLACEMENT_SUGGEST: _ClassVar[EventKind]
    EVENT_KIND_OVERRIDE_RECORDED: _ClassVar[EventKind]
    EVENT_KIND_APPLY_PLANNED: _ClassVar[EventKind]
    EVENT_KIND_APPLY_RAN: _ClassVar[EventKind]
    EVENT_KIND_APPLY_BUILD_GREEN: _ClassVar[EventKind]
    EVENT_KIND_APPLY_BUILD_RED: _ClassVar[EventKind]
    EVENT_KIND_APPLY_REVERTED: _ClassVar[EventKind]
EVENT_KIND_UNSPECIFIED: EventKind
EVENT_KIND_CONFLICT_DETECTED: EventKind
EVENT_KIND_CONFLICT_ASSIGNED: EventKind
EVENT_KIND_CONFLICT_RESOLVED: EventKind
EVENT_KIND_CONFLICT_REOPENED: EventKind
EVENT_KIND_CONFLICT_FORCE_RESOLVED: EventKind
EVENT_KIND_VERDICT_PRODUCED: EventKind
EVENT_KIND_PLACEMENT_AUTO: EventKind
EVENT_KIND_PLACEMENT_SUGGEST: EventKind
EVENT_KIND_OVERRIDE_RECORDED: EventKind
EVENT_KIND_APPLY_PLANNED: EventKind
EVENT_KIND_APPLY_RAN: EventKind
EVENT_KIND_APPLY_BUILD_GREEN: EventKind
EVENT_KIND_APPLY_BUILD_RED: EventKind
EVENT_KIND_APPLY_REVERTED: EventKind

class Event(_message.Message):
    __slots__ = ("id", "kind", "scenario", "domain", "conflict_id", "chunk_id", "plan_id", "run_id", "corrects_event_id", "payload", "actor", "recorded_at")
    ID_FIELD_NUMBER: _ClassVar[int]
    KIND_FIELD_NUMBER: _ClassVar[int]
    SCENARIO_FIELD_NUMBER: _ClassVar[int]
    DOMAIN_FIELD_NUMBER: _ClassVar[int]
    CONFLICT_ID_FIELD_NUMBER: _ClassVar[int]
    CHUNK_ID_FIELD_NUMBER: _ClassVar[int]
    PLAN_ID_FIELD_NUMBER: _ClassVar[int]
    RUN_ID_FIELD_NUMBER: _ClassVar[int]
    CORRECTS_EVENT_ID_FIELD_NUMBER: _ClassVar[int]
    PAYLOAD_FIELD_NUMBER: _ClassVar[int]
    ACTOR_FIELD_NUMBER: _ClassVar[int]
    RECORDED_AT_FIELD_NUMBER: _ClassVar[int]
    id: str
    kind: EventKind
    scenario: str
    domain: str
    conflict_id: str
    chunk_id: str
    plan_id: str
    run_id: str
    corrects_event_id: str
    payload: bytes
    actor: str
    recorded_at: _timestamp_pb2.Timestamp
    def __init__(self, id: _Optional[str] = ..., kind: _Optional[_Union[EventKind, str]] = ..., scenario: _Optional[str] = ..., domain: _Optional[str] = ..., conflict_id: _Optional[str] = ..., chunk_id: _Optional[str] = ..., plan_id: _Optional[str] = ..., run_id: _Optional[str] = ..., corrects_event_id: _Optional[str] = ..., payload: _Optional[bytes] = ..., actor: _Optional[str] = ..., recorded_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ...) -> None: ...

class Placement(_message.Message):
    __slots__ = ("id", "scenario", "chunk_id", "chunk_path", "verdict", "outcome", "auto_acted", "recorded_at")
    ID_FIELD_NUMBER: _ClassVar[int]
    SCENARIO_FIELD_NUMBER: _ClassVar[int]
    CHUNK_ID_FIELD_NUMBER: _ClassVar[int]
    CHUNK_PATH_FIELD_NUMBER: _ClassVar[int]
    VERDICT_FIELD_NUMBER: _ClassVar[int]
    OUTCOME_FIELD_NUMBER: _ClassVar[int]
    AUTO_ACTED_FIELD_NUMBER: _ClassVar[int]
    RECORDED_AT_FIELD_NUMBER: _ClassVar[int]
    id: str
    scenario: str
    chunk_id: str
    chunk_path: str
    verdict: _signals_pb2.Verdict
    outcome: str
    auto_acted: bool
    recorded_at: _timestamp_pb2.Timestamp
    def __init__(self, id: _Optional[str] = ..., scenario: _Optional[str] = ..., chunk_id: _Optional[str] = ..., chunk_path: _Optional[str] = ..., verdict: _Optional[_Union[_signals_pb2.Verdict, _Mapping]] = ..., outcome: _Optional[str] = ..., auto_acted: _Optional[bool] = ..., recorded_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ...) -> None: ...

class Override(_message.Message):
    __slots__ = ("id", "scenario", "chunk_id", "verdict_domain", "chosen_domain", "note", "verdict_event_id", "recorded_at")
    ID_FIELD_NUMBER: _ClassVar[int]
    SCENARIO_FIELD_NUMBER: _ClassVar[int]
    CHUNK_ID_FIELD_NUMBER: _ClassVar[int]
    VERDICT_DOMAIN_FIELD_NUMBER: _ClassVar[int]
    CHOSEN_DOMAIN_FIELD_NUMBER: _ClassVar[int]
    NOTE_FIELD_NUMBER: _ClassVar[int]
    VERDICT_EVENT_ID_FIELD_NUMBER: _ClassVar[int]
    RECORDED_AT_FIELD_NUMBER: _ClassVar[int]
    id: str
    scenario: str
    chunk_id: str
    verdict_domain: str
    chosen_domain: str
    note: str
    verdict_event_id: str
    recorded_at: _timestamp_pb2.Timestamp
    def __init__(self, id: _Optional[str] = ..., scenario: _Optional[str] = ..., chunk_id: _Optional[str] = ..., verdict_domain: _Optional[str] = ..., chosen_domain: _Optional[str] = ..., note: _Optional[str] = ..., verdict_event_id: _Optional[str] = ..., recorded_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ...) -> None: ...

class StatsSummary(_message.Message):
    __slots__ = ("scenario", "conflicts_detected", "conflicts_resolved", "conflicts_force_resolved", "placements_auto", "placements_suggest", "overrides", "verdict_success_rate", "verdict_success_rate_suppressed", "verdict_observation_count")
    SCENARIO_FIELD_NUMBER: _ClassVar[int]
    CONFLICTS_DETECTED_FIELD_NUMBER: _ClassVar[int]
    CONFLICTS_RESOLVED_FIELD_NUMBER: _ClassVar[int]
    CONFLICTS_FORCE_RESOLVED_FIELD_NUMBER: _ClassVar[int]
    PLACEMENTS_AUTO_FIELD_NUMBER: _ClassVar[int]
    PLACEMENTS_SUGGEST_FIELD_NUMBER: _ClassVar[int]
    OVERRIDES_FIELD_NUMBER: _ClassVar[int]
    VERDICT_SUCCESS_RATE_FIELD_NUMBER: _ClassVar[int]
    VERDICT_SUCCESS_RATE_SUPPRESSED_FIELD_NUMBER: _ClassVar[int]
    VERDICT_OBSERVATION_COUNT_FIELD_NUMBER: _ClassVar[int]
    scenario: str
    conflicts_detected: int
    conflicts_resolved: int
    conflicts_force_resolved: int
    placements_auto: int
    placements_suggest: int
    overrides: int
    verdict_success_rate: float
    verdict_success_rate_suppressed: bool
    verdict_observation_count: int
    def __init__(self, scenario: _Optional[str] = ..., conflicts_detected: _Optional[int] = ..., conflicts_resolved: _Optional[int] = ..., conflicts_force_resolved: _Optional[int] = ..., placements_auto: _Optional[int] = ..., placements_suggest: _Optional[int] = ..., overrides: _Optional[int] = ..., verdict_success_rate: _Optional[float] = ..., verdict_success_rate_suppressed: _Optional[bool] = ..., verdict_observation_count: _Optional[int] = ...) -> None: ...

class ListEventsRequest(_message.Message):
    __slots__ = ("scenario", "kinds", "since", "page_size", "page_token")
    SCENARIO_FIELD_NUMBER: _ClassVar[int]
    KINDS_FIELD_NUMBER: _ClassVar[int]
    SINCE_FIELD_NUMBER: _ClassVar[int]
    PAGE_SIZE_FIELD_NUMBER: _ClassVar[int]
    PAGE_TOKEN_FIELD_NUMBER: _ClassVar[int]
    scenario: str
    kinds: _containers.RepeatedScalarFieldContainer[EventKind]
    since: _timestamp_pb2.Timestamp
    page_size: int
    page_token: str
    def __init__(self, scenario: _Optional[str] = ..., kinds: _Optional[_Iterable[_Union[EventKind, str]]] = ..., since: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., page_size: _Optional[int] = ..., page_token: _Optional[str] = ...) -> None: ...

class ListEventsResponse(_message.Message):
    __slots__ = ("events", "next_page_token")
    EVENTS_FIELD_NUMBER: _ClassVar[int]
    NEXT_PAGE_TOKEN_FIELD_NUMBER: _ClassVar[int]
    events: _containers.RepeatedCompositeFieldContainer[Event]
    next_page_token: str
    def __init__(self, events: _Optional[_Iterable[_Union[Event, _Mapping]]] = ..., next_page_token: _Optional[str] = ...) -> None: ...

class GetStatsRequest(_message.Message):
    __slots__ = ("scenario",)
    SCENARIO_FIELD_NUMBER: _ClassVar[int]
    scenario: str
    def __init__(self, scenario: _Optional[str] = ...) -> None: ...

class GetStatsResponse(_message.Message):
    __slots__ = ("stats",)
    STATS_FIELD_NUMBER: _ClassVar[int]
    stats: StatsSummary
    def __init__(self, stats: _Optional[_Union[StatsSummary, _Mapping]] = ...) -> None: ...

class ListPlacementsRequest(_message.Message):
    __slots__ = ("scenario", "outcomes", "page_size", "page_token")
    SCENARIO_FIELD_NUMBER: _ClassVar[int]
    OUTCOMES_FIELD_NUMBER: _ClassVar[int]
    PAGE_SIZE_FIELD_NUMBER: _ClassVar[int]
    PAGE_TOKEN_FIELD_NUMBER: _ClassVar[int]
    scenario: str
    outcomes: _containers.RepeatedScalarFieldContainer[str]
    page_size: int
    page_token: str
    def __init__(self, scenario: _Optional[str] = ..., outcomes: _Optional[_Iterable[str]] = ..., page_size: _Optional[int] = ..., page_token: _Optional[str] = ...) -> None: ...

class ListPlacementsResponse(_message.Message):
    __slots__ = ("placements", "next_page_token")
    PLACEMENTS_FIELD_NUMBER: _ClassVar[int]
    NEXT_PAGE_TOKEN_FIELD_NUMBER: _ClassVar[int]
    placements: _containers.RepeatedCompositeFieldContainer[Placement]
    next_page_token: str
    def __init__(self, placements: _Optional[_Iterable[_Union[Placement, _Mapping]]] = ..., next_page_token: _Optional[str] = ...) -> None: ...

class RecordOverrideRequest(_message.Message):
    __slots__ = ("scenario", "chunk_id", "verdict_domain", "chosen_domain", "note", "verdict_event_id", "idempotency_key", "dry_run")
    SCENARIO_FIELD_NUMBER: _ClassVar[int]
    CHUNK_ID_FIELD_NUMBER: _ClassVar[int]
    VERDICT_DOMAIN_FIELD_NUMBER: _ClassVar[int]
    CHOSEN_DOMAIN_FIELD_NUMBER: _ClassVar[int]
    NOTE_FIELD_NUMBER: _ClassVar[int]
    VERDICT_EVENT_ID_FIELD_NUMBER: _ClassVar[int]
    IDEMPOTENCY_KEY_FIELD_NUMBER: _ClassVar[int]
    DRY_RUN_FIELD_NUMBER: _ClassVar[int]
    scenario: str
    chunk_id: str
    verdict_domain: str
    chosen_domain: str
    note: str
    verdict_event_id: str
    idempotency_key: str
    dry_run: bool
    def __init__(self, scenario: _Optional[str] = ..., chunk_id: _Optional[str] = ..., verdict_domain: _Optional[str] = ..., chosen_domain: _Optional[str] = ..., note: _Optional[str] = ..., verdict_event_id: _Optional[str] = ..., idempotency_key: _Optional[str] = ..., dry_run: _Optional[bool] = ...) -> None: ...

class RecordOverrideResponse(_message.Message):
    __slots__ = ("override", "dry_run")
    OVERRIDE_FIELD_NUMBER: _ClassVar[int]
    DRY_RUN_FIELD_NUMBER: _ClassVar[int]
    override: Override
    dry_run: bool
    def __init__(self, override: _Optional[_Union[Override, _Mapping]] = ..., dry_run: _Optional[bool] = ...) -> None: ...

class VerdictPayload(_message.Message):
    __slots__ = ("verdict",)
    VERDICT_FIELD_NUMBER: _ClassVar[int]
    verdict: _signals_pb2.Verdict
    def __init__(self, verdict: _Optional[_Union[_signals_pb2.Verdict, _Mapping]] = ...) -> None: ...

class ConflictPayload(_message.Message):
    __slots__ = ("conflict",)
    CONFLICT_FIELD_NUMBER: _ClassVar[int]
    conflict: _conflicts_pb2.Conflict
    def __init__(self, conflict: _Optional[_Union[_conflicts_pb2.Conflict, _Mapping]] = ...) -> None: ...
