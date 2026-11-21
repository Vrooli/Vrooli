from google.protobuf.internal import containers as _containers
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Iterable as _Iterable, Mapping as _Mapping
from typing import ClassVar as _ClassVar, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class GetDailySummaryRequest(_message.Message):
    __slots__ = ("local_date",)
    LOCAL_DATE_FIELD_NUMBER: _ClassVar[int]
    local_date: str
    def __init__(self, local_date: _Optional[str] = ...) -> None: ...

class DailySummary(_message.Message):
    __slots__ = ("local_date", "planned_minutes", "recorded_active_minutes", "focus_session_count", "active_goal_count", "unrecorded_minutes", "coverage_note")
    LOCAL_DATE_FIELD_NUMBER: _ClassVar[int]
    PLANNED_MINUTES_FIELD_NUMBER: _ClassVar[int]
    RECORDED_ACTIVE_MINUTES_FIELD_NUMBER: _ClassVar[int]
    FOCUS_SESSION_COUNT_FIELD_NUMBER: _ClassVar[int]
    ACTIVE_GOAL_COUNT_FIELD_NUMBER: _ClassVar[int]
    UNRECORDED_MINUTES_FIELD_NUMBER: _ClassVar[int]
    COVERAGE_NOTE_FIELD_NUMBER: _ClassVar[int]
    local_date: str
    planned_minutes: int
    recorded_active_minutes: int
    focus_session_count: int
    active_goal_count: int
    unrecorded_minutes: int
    coverage_note: str
    def __init__(self, local_date: _Optional[str] = ..., planned_minutes: _Optional[int] = ..., recorded_active_minutes: _Optional[int] = ..., focus_session_count: _Optional[int] = ..., active_goal_count: _Optional[int] = ..., unrecorded_minutes: _Optional[int] = ..., coverage_note: _Optional[str] = ...) -> None: ...

class GetDailySummaryResponse(_message.Message):
    __slots__ = ("summary",)
    SUMMARY_FIELD_NUMBER: _ClassVar[int]
    summary: DailySummary
    def __init__(self, summary: _Optional[_Union[DailySummary, _Mapping]] = ...) -> None: ...

class GetWeeklySummaryRequest(_message.Message):
    __slots__ = ("week_start_local_date",)
    WEEK_START_LOCAL_DATE_FIELD_NUMBER: _ClassVar[int]
    week_start_local_date: str
    def __init__(self, week_start_local_date: _Optional[str] = ...) -> None: ...

class WeeklySummary(_message.Message):
    __slots__ = ("week_start_local_date", "days", "planned_minutes", "recorded_active_minutes", "focus_session_count", "active_goal_count", "coverage_note")
    WEEK_START_LOCAL_DATE_FIELD_NUMBER: _ClassVar[int]
    DAYS_FIELD_NUMBER: _ClassVar[int]
    PLANNED_MINUTES_FIELD_NUMBER: _ClassVar[int]
    RECORDED_ACTIVE_MINUTES_FIELD_NUMBER: _ClassVar[int]
    FOCUS_SESSION_COUNT_FIELD_NUMBER: _ClassVar[int]
    ACTIVE_GOAL_COUNT_FIELD_NUMBER: _ClassVar[int]
    COVERAGE_NOTE_FIELD_NUMBER: _ClassVar[int]
    week_start_local_date: str
    days: _containers.RepeatedCompositeFieldContainer[DailySummary]
    planned_minutes: int
    recorded_active_minutes: int
    focus_session_count: int
    active_goal_count: int
    coverage_note: str
    def __init__(self, week_start_local_date: _Optional[str] = ..., days: _Optional[_Iterable[_Union[DailySummary, _Mapping]]] = ..., planned_minutes: _Optional[int] = ..., recorded_active_minutes: _Optional[int] = ..., focus_session_count: _Optional[int] = ..., active_goal_count: _Optional[int] = ..., coverage_note: _Optional[str] = ...) -> None: ...

class GetWeeklySummaryResponse(_message.Message):
    __slots__ = ("summary",)
    SUMMARY_FIELD_NUMBER: _ClassVar[int]
    summary: WeeklySummary
    def __init__(self, summary: _Optional[_Union[WeeklySummary, _Mapping]] = ...) -> None: ...

class ReviewReflection(_message.Message):
    __slots__ = ("local_date", "text", "updated_at")
    LOCAL_DATE_FIELD_NUMBER: _ClassVar[int]
    TEXT_FIELD_NUMBER: _ClassVar[int]
    UPDATED_AT_FIELD_NUMBER: _ClassVar[int]
    local_date: str
    text: str
    updated_at: str
    def __init__(self, local_date: _Optional[str] = ..., text: _Optional[str] = ..., updated_at: _Optional[str] = ...) -> None: ...

class SaveReflectionRequest(_message.Message):
    __slots__ = ("local_date", "text")
    LOCAL_DATE_FIELD_NUMBER: _ClassVar[int]
    TEXT_FIELD_NUMBER: _ClassVar[int]
    local_date: str
    text: str
    def __init__(self, local_date: _Optional[str] = ..., text: _Optional[str] = ...) -> None: ...

class SaveReflectionResponse(_message.Message):
    __slots__ = ("reflection",)
    REFLECTION_FIELD_NUMBER: _ClassVar[int]
    reflection: ReviewReflection
    def __init__(self, reflection: _Optional[_Union[ReviewReflection, _Mapping]] = ...) -> None: ...

class GetReflectionRequest(_message.Message):
    __slots__ = ("local_date",)
    LOCAL_DATE_FIELD_NUMBER: _ClassVar[int]
    local_date: str
    def __init__(self, local_date: _Optional[str] = ...) -> None: ...

class GetReflectionResponse(_message.Message):
    __slots__ = ("reflection",)
    REFLECTION_FIELD_NUMBER: _ClassVar[int]
    reflection: ReviewReflection
    def __init__(self, reflection: _Optional[_Union[ReviewReflection, _Mapping]] = ...) -> None: ...
