import datetime

from google.protobuf import timestamp_pb2 as _timestamp_pb2
from google.protobuf.internal import containers as _containers
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Iterable as _Iterable, Mapping as _Mapping
from typing import ClassVar as _ClassVar, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class CalendarEvent(_message.Message):
    __slots__ = ("id", "title", "subject", "notes", "availability", "timezone", "all_day", "start_date", "end_date_exclusive", "start_at", "end_at", "provider", "provider_calendar_id", "provider_event_id", "occurrence_id", "revision", "created_at", "updated_at")
    ID_FIELD_NUMBER: _ClassVar[int]
    TITLE_FIELD_NUMBER: _ClassVar[int]
    SUBJECT_FIELD_NUMBER: _ClassVar[int]
    NOTES_FIELD_NUMBER: _ClassVar[int]
    AVAILABILITY_FIELD_NUMBER: _ClassVar[int]
    TIMEZONE_FIELD_NUMBER: _ClassVar[int]
    ALL_DAY_FIELD_NUMBER: _ClassVar[int]
    START_DATE_FIELD_NUMBER: _ClassVar[int]
    END_DATE_EXCLUSIVE_FIELD_NUMBER: _ClassVar[int]
    START_AT_FIELD_NUMBER: _ClassVar[int]
    END_AT_FIELD_NUMBER: _ClassVar[int]
    PROVIDER_FIELD_NUMBER: _ClassVar[int]
    PROVIDER_CALENDAR_ID_FIELD_NUMBER: _ClassVar[int]
    PROVIDER_EVENT_ID_FIELD_NUMBER: _ClassVar[int]
    OCCURRENCE_ID_FIELD_NUMBER: _ClassVar[int]
    REVISION_FIELD_NUMBER: _ClassVar[int]
    CREATED_AT_FIELD_NUMBER: _ClassVar[int]
    UPDATED_AT_FIELD_NUMBER: _ClassVar[int]
    id: str
    title: str
    subject: str
    notes: str
    availability: str
    timezone: str
    all_day: bool
    start_date: str
    end_date_exclusive: str
    start_at: str
    end_at: str
    provider: str
    provider_calendar_id: str
    provider_event_id: str
    occurrence_id: str
    revision: int
    created_at: _timestamp_pb2.Timestamp
    updated_at: _timestamp_pb2.Timestamp
    def __init__(self, id: _Optional[str] = ..., title: _Optional[str] = ..., subject: _Optional[str] = ..., notes: _Optional[str] = ..., availability: _Optional[str] = ..., timezone: _Optional[str] = ..., all_day: _Optional[bool] = ..., start_date: _Optional[str] = ..., end_date_exclusive: _Optional[str] = ..., start_at: _Optional[str] = ..., end_at: _Optional[str] = ..., provider: _Optional[str] = ..., provider_calendar_id: _Optional[str] = ..., provider_event_id: _Optional[str] = ..., occurrence_id: _Optional[str] = ..., revision: _Optional[int] = ..., created_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., updated_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ...) -> None: ...

class ListEventsRequest(_message.Message):
    __slots__ = ("start_local_date", "end_local_date")
    START_LOCAL_DATE_FIELD_NUMBER: _ClassVar[int]
    END_LOCAL_DATE_FIELD_NUMBER: _ClassVar[int]
    start_local_date: str
    end_local_date: str
    def __init__(self, start_local_date: _Optional[str] = ..., end_local_date: _Optional[str] = ...) -> None: ...

class ListEventsResponse(_message.Message):
    __slots__ = ("events",)
    EVENTS_FIELD_NUMBER: _ClassVar[int]
    events: _containers.RepeatedCompositeFieldContainer[CalendarEvent]
    def __init__(self, events: _Optional[_Iterable[_Union[CalendarEvent, _Mapping]]] = ...) -> None: ...

class GetEventRequest(_message.Message):
    __slots__ = ("event_id",)
    EVENT_ID_FIELD_NUMBER: _ClassVar[int]
    event_id: str
    def __init__(self, event_id: _Optional[str] = ...) -> None: ...

class GetEventResponse(_message.Message):
    __slots__ = ("event",)
    EVENT_FIELD_NUMBER: _ClassVar[int]
    event: CalendarEvent
    def __init__(self, event: _Optional[_Union[CalendarEvent, _Mapping]] = ...) -> None: ...

class CreateEventRequest(_message.Message):
    __slots__ = ("event", "idempotency_key")
    EVENT_FIELD_NUMBER: _ClassVar[int]
    IDEMPOTENCY_KEY_FIELD_NUMBER: _ClassVar[int]
    event: CalendarEvent
    idempotency_key: str
    def __init__(self, event: _Optional[_Union[CalendarEvent, _Mapping]] = ..., idempotency_key: _Optional[str] = ...) -> None: ...

class CreateEventResponse(_message.Message):
    __slots__ = ("event",)
    EVENT_FIELD_NUMBER: _ClassVar[int]
    event: CalendarEvent
    def __init__(self, event: _Optional[_Union[CalendarEvent, _Mapping]] = ...) -> None: ...

class UpdateEventRequest(_message.Message):
    __slots__ = ("event", "expected_revision")
    EVENT_FIELD_NUMBER: _ClassVar[int]
    EXPECTED_REVISION_FIELD_NUMBER: _ClassVar[int]
    event: CalendarEvent
    expected_revision: int
    def __init__(self, event: _Optional[_Union[CalendarEvent, _Mapping]] = ..., expected_revision: _Optional[int] = ...) -> None: ...

class UpdateEventResponse(_message.Message):
    __slots__ = ("event",)
    EVENT_FIELD_NUMBER: _ClassVar[int]
    event: CalendarEvent
    def __init__(self, event: _Optional[_Union[CalendarEvent, _Mapping]] = ...) -> None: ...

class Allocation(_message.Message):
    __slots__ = ("id", "work_item_id", "title", "source_label", "local_date", "start_minutes", "duration_minutes", "state", "created_at", "carried_from_id")
    ID_FIELD_NUMBER: _ClassVar[int]
    WORK_ITEM_ID_FIELD_NUMBER: _ClassVar[int]
    TITLE_FIELD_NUMBER: _ClassVar[int]
    SOURCE_LABEL_FIELD_NUMBER: _ClassVar[int]
    LOCAL_DATE_FIELD_NUMBER: _ClassVar[int]
    START_MINUTES_FIELD_NUMBER: _ClassVar[int]
    DURATION_MINUTES_FIELD_NUMBER: _ClassVar[int]
    STATE_FIELD_NUMBER: _ClassVar[int]
    CREATED_AT_FIELD_NUMBER: _ClassVar[int]
    CARRIED_FROM_ID_FIELD_NUMBER: _ClassVar[int]
    id: str
    work_item_id: str
    title: str
    source_label: str
    local_date: str
    start_minutes: int
    duration_minutes: int
    state: str
    created_at: _timestamp_pb2.Timestamp
    carried_from_id: str
    def __init__(self, id: _Optional[str] = ..., work_item_id: _Optional[str] = ..., title: _Optional[str] = ..., source_label: _Optional[str] = ..., local_date: _Optional[str] = ..., start_minutes: _Optional[int] = ..., duration_minutes: _Optional[int] = ..., state: _Optional[str] = ..., created_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., carried_from_id: _Optional[str] = ...) -> None: ...

class ListTodayAllocationsRequest(_message.Message):
    __slots__ = ("local_date",)
    LOCAL_DATE_FIELD_NUMBER: _ClassVar[int]
    local_date: str
    def __init__(self, local_date: _Optional[str] = ...) -> None: ...

class ListTodayAllocationsResponse(_message.Message):
    __slots__ = ("allocations", "planned_minutes", "available_minutes", "breathing_room_minutes", "external_busy_minutes", "external_event_count", "external_freshness")
    ALLOCATIONS_FIELD_NUMBER: _ClassVar[int]
    PLANNED_MINUTES_FIELD_NUMBER: _ClassVar[int]
    AVAILABLE_MINUTES_FIELD_NUMBER: _ClassVar[int]
    BREATHING_ROOM_MINUTES_FIELD_NUMBER: _ClassVar[int]
    EXTERNAL_BUSY_MINUTES_FIELD_NUMBER: _ClassVar[int]
    EXTERNAL_EVENT_COUNT_FIELD_NUMBER: _ClassVar[int]
    EXTERNAL_FRESHNESS_FIELD_NUMBER: _ClassVar[int]
    allocations: _containers.RepeatedCompositeFieldContainer[Allocation]
    planned_minutes: int
    available_minutes: int
    breathing_room_minutes: int
    external_busy_minutes: int
    external_event_count: int
    external_freshness: str
    def __init__(self, allocations: _Optional[_Iterable[_Union[Allocation, _Mapping]]] = ..., planned_minutes: _Optional[int] = ..., available_minutes: _Optional[int] = ..., breathing_room_minutes: _Optional[int] = ..., external_busy_minutes: _Optional[int] = ..., external_event_count: _Optional[int] = ..., external_freshness: _Optional[str] = ...) -> None: ...

class ListAllocationsRequest(_message.Message):
    __slots__ = ("start_local_date", "end_local_date")
    START_LOCAL_DATE_FIELD_NUMBER: _ClassVar[int]
    END_LOCAL_DATE_FIELD_NUMBER: _ClassVar[int]
    start_local_date: str
    end_local_date: str
    def __init__(self, start_local_date: _Optional[str] = ..., end_local_date: _Optional[str] = ...) -> None: ...

class ListAllocationsResponse(_message.Message):
    __slots__ = ("allocations",)
    ALLOCATIONS_FIELD_NUMBER: _ClassVar[int]
    allocations: _containers.RepeatedCompositeFieldContainer[Allocation]
    def __init__(self, allocations: _Optional[_Iterable[_Union[Allocation, _Mapping]]] = ...) -> None: ...

class CreateAllocationRequest(_message.Message):
    __slots__ = ("work_item_id", "local_date", "start_minutes", "duration_minutes")
    WORK_ITEM_ID_FIELD_NUMBER: _ClassVar[int]
    LOCAL_DATE_FIELD_NUMBER: _ClassVar[int]
    START_MINUTES_FIELD_NUMBER: _ClassVar[int]
    DURATION_MINUTES_FIELD_NUMBER: _ClassVar[int]
    work_item_id: str
    local_date: str
    start_minutes: int
    duration_minutes: int
    def __init__(self, work_item_id: _Optional[str] = ..., local_date: _Optional[str] = ..., start_minutes: _Optional[int] = ..., duration_minutes: _Optional[int] = ...) -> None: ...

class CreateAllocationResponse(_message.Message):
    __slots__ = ("allocation",)
    ALLOCATION_FIELD_NUMBER: _ClassVar[int]
    allocation: Allocation
    def __init__(self, allocation: _Optional[_Union[Allocation, _Mapping]] = ...) -> None: ...

class PreviewAllocationRequest(_message.Message):
    __slots__ = ("work_item_id", "local_date", "start_minutes", "duration_minutes")
    WORK_ITEM_ID_FIELD_NUMBER: _ClassVar[int]
    LOCAL_DATE_FIELD_NUMBER: _ClassVar[int]
    START_MINUTES_FIELD_NUMBER: _ClassVar[int]
    DURATION_MINUTES_FIELD_NUMBER: _ClassVar[int]
    work_item_id: str
    local_date: str
    start_minutes: int
    duration_minutes: int
    def __init__(self, work_item_id: _Optional[str] = ..., local_date: _Optional[str] = ..., start_minutes: _Optional[int] = ..., duration_minutes: _Optional[int] = ...) -> None: ...

class PlacementProposal(_message.Message):
    __slots__ = ("id", "work_item_id", "local_date", "start_minutes", "duration_minutes", "state", "reason", "base_revision")
    ID_FIELD_NUMBER: _ClassVar[int]
    WORK_ITEM_ID_FIELD_NUMBER: _ClassVar[int]
    LOCAL_DATE_FIELD_NUMBER: _ClassVar[int]
    START_MINUTES_FIELD_NUMBER: _ClassVar[int]
    DURATION_MINUTES_FIELD_NUMBER: _ClassVar[int]
    STATE_FIELD_NUMBER: _ClassVar[int]
    REASON_FIELD_NUMBER: _ClassVar[int]
    BASE_REVISION_FIELD_NUMBER: _ClassVar[int]
    id: str
    work_item_id: str
    local_date: str
    start_minutes: int
    duration_minutes: int
    state: str
    reason: str
    base_revision: int
    def __init__(self, id: _Optional[str] = ..., work_item_id: _Optional[str] = ..., local_date: _Optional[str] = ..., start_minutes: _Optional[int] = ..., duration_minutes: _Optional[int] = ..., state: _Optional[str] = ..., reason: _Optional[str] = ..., base_revision: _Optional[int] = ...) -> None: ...

class PreviewAllocationResponse(_message.Message):
    __slots__ = ("proposal",)
    PROPOSAL_FIELD_NUMBER: _ClassVar[int]
    proposal: PlacementProposal
    def __init__(self, proposal: _Optional[_Union[PlacementProposal, _Mapping]] = ...) -> None: ...

class ApplyAllocationProposalRequest(_message.Message):
    __slots__ = ("proposal_id", "expected_revision", "idempotency_key")
    PROPOSAL_ID_FIELD_NUMBER: _ClassVar[int]
    EXPECTED_REVISION_FIELD_NUMBER: _ClassVar[int]
    IDEMPOTENCY_KEY_FIELD_NUMBER: _ClassVar[int]
    proposal_id: str
    expected_revision: int
    idempotency_key: str
    def __init__(self, proposal_id: _Optional[str] = ..., expected_revision: _Optional[int] = ..., idempotency_key: _Optional[str] = ...) -> None: ...

class ApplyAllocationProposalResponse(_message.Message):
    __slots__ = ("allocation",)
    ALLOCATION_FIELD_NUMBER: _ClassVar[int]
    allocation: Allocation
    def __init__(self, allocation: _Optional[_Union[Allocation, _Mapping]] = ...) -> None: ...

class PreviewScheduleRequest(_message.Message):
    __slots__ = ("local_date", "start_minutes", "work_item_ids")
    LOCAL_DATE_FIELD_NUMBER: _ClassVar[int]
    START_MINUTES_FIELD_NUMBER: _ClassVar[int]
    WORK_ITEM_IDS_FIELD_NUMBER: _ClassVar[int]
    local_date: str
    start_minutes: int
    work_item_ids: _containers.RepeatedScalarFieldContainer[str]
    def __init__(self, local_date: _Optional[str] = ..., start_minutes: _Optional[int] = ..., work_item_ids: _Optional[_Iterable[str]] = ...) -> None: ...

class ProposedPlacement(_message.Message):
    __slots__ = ("work_item_id", "title", "local_date", "start_minutes", "duration_minutes", "state", "reason")
    WORK_ITEM_ID_FIELD_NUMBER: _ClassVar[int]
    TITLE_FIELD_NUMBER: _ClassVar[int]
    LOCAL_DATE_FIELD_NUMBER: _ClassVar[int]
    START_MINUTES_FIELD_NUMBER: _ClassVar[int]
    DURATION_MINUTES_FIELD_NUMBER: _ClassVar[int]
    STATE_FIELD_NUMBER: _ClassVar[int]
    REASON_FIELD_NUMBER: _ClassVar[int]
    work_item_id: str
    title: str
    local_date: str
    start_minutes: int
    duration_minutes: int
    state: str
    reason: str
    def __init__(self, work_item_id: _Optional[str] = ..., title: _Optional[str] = ..., local_date: _Optional[str] = ..., start_minutes: _Optional[int] = ..., duration_minutes: _Optional[int] = ..., state: _Optional[str] = ..., reason: _Optional[str] = ...) -> None: ...

class ScheduleProposal(_message.Message):
    __slots__ = ("id", "local_date", "base_revision", "state", "reason", "placements")
    ID_FIELD_NUMBER: _ClassVar[int]
    LOCAL_DATE_FIELD_NUMBER: _ClassVar[int]
    BASE_REVISION_FIELD_NUMBER: _ClassVar[int]
    STATE_FIELD_NUMBER: _ClassVar[int]
    REASON_FIELD_NUMBER: _ClassVar[int]
    PLACEMENTS_FIELD_NUMBER: _ClassVar[int]
    id: str
    local_date: str
    base_revision: int
    state: str
    reason: str
    placements: _containers.RepeatedCompositeFieldContainer[ProposedPlacement]
    def __init__(self, id: _Optional[str] = ..., local_date: _Optional[str] = ..., base_revision: _Optional[int] = ..., state: _Optional[str] = ..., reason: _Optional[str] = ..., placements: _Optional[_Iterable[_Union[ProposedPlacement, _Mapping]]] = ...) -> None: ...

class PreviewScheduleResponse(_message.Message):
    __slots__ = ("proposal",)
    PROPOSAL_FIELD_NUMBER: _ClassVar[int]
    proposal: ScheduleProposal
    def __init__(self, proposal: _Optional[_Union[ScheduleProposal, _Mapping]] = ...) -> None: ...

class ApplyScheduleProposalRequest(_message.Message):
    __slots__ = ("proposal_id", "expected_revision", "idempotency_key")
    PROPOSAL_ID_FIELD_NUMBER: _ClassVar[int]
    EXPECTED_REVISION_FIELD_NUMBER: _ClassVar[int]
    IDEMPOTENCY_KEY_FIELD_NUMBER: _ClassVar[int]
    proposal_id: str
    expected_revision: int
    idempotency_key: str
    def __init__(self, proposal_id: _Optional[str] = ..., expected_revision: _Optional[int] = ..., idempotency_key: _Optional[str] = ...) -> None: ...

class ApplyScheduleProposalResponse(_message.Message):
    __slots__ = ("allocations",)
    ALLOCATIONS_FIELD_NUMBER: _ClassVar[int]
    allocations: _containers.RepeatedCompositeFieldContainer[Allocation]
    def __init__(self, allocations: _Optional[_Iterable[_Union[Allocation, _Mapping]]] = ...) -> None: ...

class CarryForwardAllocationRequest(_message.Message):
    __slots__ = ("allocation_id", "target_local_date", "start_minutes")
    ALLOCATION_ID_FIELD_NUMBER: _ClassVar[int]
    TARGET_LOCAL_DATE_FIELD_NUMBER: _ClassVar[int]
    START_MINUTES_FIELD_NUMBER: _ClassVar[int]
    allocation_id: str
    target_local_date: str
    start_minutes: int
    def __init__(self, allocation_id: _Optional[str] = ..., target_local_date: _Optional[str] = ..., start_minutes: _Optional[int] = ...) -> None: ...

class CarryForwardAllocationResponse(_message.Message):
    __slots__ = ("allocation",)
    ALLOCATION_FIELD_NUMBER: _ClassVar[int]
    allocation: Allocation
    def __init__(self, allocation: _Optional[_Union[Allocation, _Mapping]] = ...) -> None: ...

class Routine(_message.Message):
    __slots__ = ("id", "title", "kind", "timezone", "start_date", "end_date", "weekdays", "start_minute", "duration_minutes", "frequency_per_week", "revision", "active")
    ID_FIELD_NUMBER: _ClassVar[int]
    TITLE_FIELD_NUMBER: _ClassVar[int]
    KIND_FIELD_NUMBER: _ClassVar[int]
    TIMEZONE_FIELD_NUMBER: _ClassVar[int]
    START_DATE_FIELD_NUMBER: _ClassVar[int]
    END_DATE_FIELD_NUMBER: _ClassVar[int]
    WEEKDAYS_FIELD_NUMBER: _ClassVar[int]
    START_MINUTE_FIELD_NUMBER: _ClassVar[int]
    DURATION_MINUTES_FIELD_NUMBER: _ClassVar[int]
    FREQUENCY_PER_WEEK_FIELD_NUMBER: _ClassVar[int]
    REVISION_FIELD_NUMBER: _ClassVar[int]
    ACTIVE_FIELD_NUMBER: _ClassVar[int]
    id: str
    title: str
    kind: str
    timezone: str
    start_date: str
    end_date: str
    weekdays: _containers.RepeatedScalarFieldContainer[int]
    start_minute: int
    duration_minutes: int
    frequency_per_week: int
    revision: int
    active: bool
    def __init__(self, id: _Optional[str] = ..., title: _Optional[str] = ..., kind: _Optional[str] = ..., timezone: _Optional[str] = ..., start_date: _Optional[str] = ..., end_date: _Optional[str] = ..., weekdays: _Optional[_Iterable[int]] = ..., start_minute: _Optional[int] = ..., duration_minutes: _Optional[int] = ..., frequency_per_week: _Optional[int] = ..., revision: _Optional[int] = ..., active: _Optional[bool] = ...) -> None: ...

class RoutineOccurrence(_message.Message):
    __slots__ = ("routine_id", "title", "local_date", "start_minute", "duration_minutes", "kind", "generated")
    ROUTINE_ID_FIELD_NUMBER: _ClassVar[int]
    TITLE_FIELD_NUMBER: _ClassVar[int]
    LOCAL_DATE_FIELD_NUMBER: _ClassVar[int]
    START_MINUTE_FIELD_NUMBER: _ClassVar[int]
    DURATION_MINUTES_FIELD_NUMBER: _ClassVar[int]
    KIND_FIELD_NUMBER: _ClassVar[int]
    GENERATED_FIELD_NUMBER: _ClassVar[int]
    routine_id: str
    title: str
    local_date: str
    start_minute: int
    duration_minutes: int
    kind: str
    generated: bool
    def __init__(self, routine_id: _Optional[str] = ..., title: _Optional[str] = ..., local_date: _Optional[str] = ..., start_minute: _Optional[int] = ..., duration_minutes: _Optional[int] = ..., kind: _Optional[str] = ..., generated: _Optional[bool] = ...) -> None: ...

class ListRoutinesRequest(_message.Message):
    __slots__ = ()
    def __init__(self) -> None: ...

class ListRoutinesResponse(_message.Message):
    __slots__ = ("routines",)
    ROUTINES_FIELD_NUMBER: _ClassVar[int]
    routines: _containers.RepeatedCompositeFieldContainer[Routine]
    def __init__(self, routines: _Optional[_Iterable[_Union[Routine, _Mapping]]] = ...) -> None: ...

class CreateRoutineRequest(_message.Message):
    __slots__ = ("title", "kind", "timezone", "start_date", "end_date", "weekdays", "start_minute", "duration_minutes", "frequency_per_week")
    TITLE_FIELD_NUMBER: _ClassVar[int]
    KIND_FIELD_NUMBER: _ClassVar[int]
    TIMEZONE_FIELD_NUMBER: _ClassVar[int]
    START_DATE_FIELD_NUMBER: _ClassVar[int]
    END_DATE_FIELD_NUMBER: _ClassVar[int]
    WEEKDAYS_FIELD_NUMBER: _ClassVar[int]
    START_MINUTE_FIELD_NUMBER: _ClassVar[int]
    DURATION_MINUTES_FIELD_NUMBER: _ClassVar[int]
    FREQUENCY_PER_WEEK_FIELD_NUMBER: _ClassVar[int]
    title: str
    kind: str
    timezone: str
    start_date: str
    end_date: str
    weekdays: _containers.RepeatedScalarFieldContainer[int]
    start_minute: int
    duration_minutes: int
    frequency_per_week: int
    def __init__(self, title: _Optional[str] = ..., kind: _Optional[str] = ..., timezone: _Optional[str] = ..., start_date: _Optional[str] = ..., end_date: _Optional[str] = ..., weekdays: _Optional[_Iterable[int]] = ..., start_minute: _Optional[int] = ..., duration_minutes: _Optional[int] = ..., frequency_per_week: _Optional[int] = ...) -> None: ...

class CreateRoutineResponse(_message.Message):
    __slots__ = ("routine",)
    ROUTINE_FIELD_NUMBER: _ClassVar[int]
    routine: Routine
    def __init__(self, routine: _Optional[_Union[Routine, _Mapping]] = ...) -> None: ...

class ListRoutineOccurrencesRequest(_message.Message):
    __slots__ = ("start_local_date", "end_local_date")
    START_LOCAL_DATE_FIELD_NUMBER: _ClassVar[int]
    END_LOCAL_DATE_FIELD_NUMBER: _ClassVar[int]
    start_local_date: str
    end_local_date: str
    def __init__(self, start_local_date: _Optional[str] = ..., end_local_date: _Optional[str] = ...) -> None: ...

class ListRoutineOccurrencesResponse(_message.Message):
    __slots__ = ("occurrences",)
    OCCURRENCES_FIELD_NUMBER: _ClassVar[int]
    occurrences: _containers.RepeatedCompositeFieldContainer[RoutineOccurrence]
    def __init__(self, occurrences: _Optional[_Iterable[_Union[RoutineOccurrence, _Mapping]]] = ...) -> None: ...

class SkipRoutineOccurrenceRequest(_message.Message):
    __slots__ = ("routine_id", "local_date", "expected_revision")
    ROUTINE_ID_FIELD_NUMBER: _ClassVar[int]
    LOCAL_DATE_FIELD_NUMBER: _ClassVar[int]
    EXPECTED_REVISION_FIELD_NUMBER: _ClassVar[int]
    routine_id: str
    local_date: str
    expected_revision: int
    def __init__(self, routine_id: _Optional[str] = ..., local_date: _Optional[str] = ..., expected_revision: _Optional[int] = ...) -> None: ...

class SkipRoutineOccurrenceResponse(_message.Message):
    __slots__ = ("skipped",)
    SKIPPED_FIELD_NUMBER: _ClassVar[int]
    skipped: bool
    def __init__(self, skipped: _Optional[bool] = ...) -> None: ...

class RescheduleRoutineOccurrenceRequest(_message.Message):
    __slots__ = ("routine_id", "local_date", "start_minute", "expected_revision")
    ROUTINE_ID_FIELD_NUMBER: _ClassVar[int]
    LOCAL_DATE_FIELD_NUMBER: _ClassVar[int]
    START_MINUTE_FIELD_NUMBER: _ClassVar[int]
    EXPECTED_REVISION_FIELD_NUMBER: _ClassVar[int]
    routine_id: str
    local_date: str
    start_minute: int
    expected_revision: int
    def __init__(self, routine_id: _Optional[str] = ..., local_date: _Optional[str] = ..., start_minute: _Optional[int] = ..., expected_revision: _Optional[int] = ...) -> None: ...

class RescheduleRoutineOccurrenceResponse(_message.Message):
    __slots__ = ("rescheduled",)
    RESCHEDULED_FIELD_NUMBER: _ClassVar[int]
    rescheduled: bool
    def __init__(self, rescheduled: _Optional[bool] = ...) -> None: ...
