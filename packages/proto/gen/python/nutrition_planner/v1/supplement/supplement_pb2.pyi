from google.protobuf.internal import containers as _containers
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Iterable as _Iterable, Mapping as _Mapping
from typing import ClassVar as _ClassVar, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class Schedule(_message.Message):
    __slots__ = ("id", "revision", "product_revision_id", "dose", "dose_unit", "weekdays", "start_date", "end_date", "paused", "confirmed", "created_at")
    ID_FIELD_NUMBER: _ClassVar[int]
    REVISION_FIELD_NUMBER: _ClassVar[int]
    PRODUCT_REVISION_ID_FIELD_NUMBER: _ClassVar[int]
    DOSE_FIELD_NUMBER: _ClassVar[int]
    DOSE_UNIT_FIELD_NUMBER: _ClassVar[int]
    WEEKDAYS_FIELD_NUMBER: _ClassVar[int]
    START_DATE_FIELD_NUMBER: _ClassVar[int]
    END_DATE_FIELD_NUMBER: _ClassVar[int]
    PAUSED_FIELD_NUMBER: _ClassVar[int]
    CONFIRMED_FIELD_NUMBER: _ClassVar[int]
    CREATED_AT_FIELD_NUMBER: _ClassVar[int]
    id: str
    revision: int
    product_revision_id: str
    dose: str
    dose_unit: str
    weekdays: _containers.RepeatedScalarFieldContainer[int]
    start_date: str
    end_date: str
    paused: bool
    confirmed: bool
    created_at: str
    def __init__(self, id: _Optional[str] = ..., revision: _Optional[int] = ..., product_revision_id: _Optional[str] = ..., dose: _Optional[str] = ..., dose_unit: _Optional[str] = ..., weekdays: _Optional[_Iterable[int]] = ..., start_date: _Optional[str] = ..., end_date: _Optional[str] = ..., paused: _Optional[bool] = ..., confirmed: _Optional[bool] = ..., created_at: _Optional[str] = ...) -> None: ...

class ListSchedulesRequest(_message.Message):
    __slots__ = ("workspace_id",)
    WORKSPACE_ID_FIELD_NUMBER: _ClassVar[int]
    workspace_id: str
    def __init__(self, workspace_id: _Optional[str] = ...) -> None: ...

class ListSchedulesResponse(_message.Message):
    __slots__ = ("schedules",)
    SCHEDULES_FIELD_NUMBER: _ClassVar[int]
    schedules: _containers.RepeatedCompositeFieldContainer[Schedule]
    def __init__(self, schedules: _Optional[_Iterable[_Union[Schedule, _Mapping]]] = ...) -> None: ...

class CreateScheduleRequest(_message.Message):
    __slots__ = ("workspace_id", "product_revision_id", "dose", "dose_unit", "weekdays", "start_date", "end_date", "confirmed")
    WORKSPACE_ID_FIELD_NUMBER: _ClassVar[int]
    PRODUCT_REVISION_ID_FIELD_NUMBER: _ClassVar[int]
    DOSE_FIELD_NUMBER: _ClassVar[int]
    DOSE_UNIT_FIELD_NUMBER: _ClassVar[int]
    WEEKDAYS_FIELD_NUMBER: _ClassVar[int]
    START_DATE_FIELD_NUMBER: _ClassVar[int]
    END_DATE_FIELD_NUMBER: _ClassVar[int]
    CONFIRMED_FIELD_NUMBER: _ClassVar[int]
    workspace_id: str
    product_revision_id: str
    dose: str
    dose_unit: str
    weekdays: _containers.RepeatedScalarFieldContainer[int]
    start_date: str
    end_date: str
    confirmed: bool
    def __init__(self, workspace_id: _Optional[str] = ..., product_revision_id: _Optional[str] = ..., dose: _Optional[str] = ..., dose_unit: _Optional[str] = ..., weekdays: _Optional[_Iterable[int]] = ..., start_date: _Optional[str] = ..., end_date: _Optional[str] = ..., confirmed: _Optional[bool] = ...) -> None: ...

class CreateScheduleResponse(_message.Message):
    __slots__ = ("schedule",)
    SCHEDULE_FIELD_NUMBER: _ClassVar[int]
    schedule: Schedule
    def __init__(self, schedule: _Optional[_Union[Schedule, _Mapping]] = ...) -> None: ...

class UpdateScheduleRequest(_message.Message):
    __slots__ = ("workspace_id", "id", "expected_revision", "dose", "dose_unit", "weekdays", "start_date", "end_date", "paused", "confirmed")
    WORKSPACE_ID_FIELD_NUMBER: _ClassVar[int]
    ID_FIELD_NUMBER: _ClassVar[int]
    EXPECTED_REVISION_FIELD_NUMBER: _ClassVar[int]
    DOSE_FIELD_NUMBER: _ClassVar[int]
    DOSE_UNIT_FIELD_NUMBER: _ClassVar[int]
    WEEKDAYS_FIELD_NUMBER: _ClassVar[int]
    START_DATE_FIELD_NUMBER: _ClassVar[int]
    END_DATE_FIELD_NUMBER: _ClassVar[int]
    PAUSED_FIELD_NUMBER: _ClassVar[int]
    CONFIRMED_FIELD_NUMBER: _ClassVar[int]
    workspace_id: str
    id: str
    expected_revision: int
    dose: str
    dose_unit: str
    weekdays: _containers.RepeatedScalarFieldContainer[int]
    start_date: str
    end_date: str
    paused: bool
    confirmed: bool
    def __init__(self, workspace_id: _Optional[str] = ..., id: _Optional[str] = ..., expected_revision: _Optional[int] = ..., dose: _Optional[str] = ..., dose_unit: _Optional[str] = ..., weekdays: _Optional[_Iterable[int]] = ..., start_date: _Optional[str] = ..., end_date: _Optional[str] = ..., paused: _Optional[bool] = ..., confirmed: _Optional[bool] = ...) -> None: ...

class UpdateScheduleResponse(_message.Message):
    __slots__ = ("schedule",)
    SCHEDULE_FIELD_NUMBER: _ClassVar[int]
    schedule: Schedule
    def __init__(self, schedule: _Optional[_Union[Schedule, _Mapping]] = ...) -> None: ...

class GetScheduleRequest(_message.Message):
    __slots__ = ("workspace_id", "id", "revision")
    WORKSPACE_ID_FIELD_NUMBER: _ClassVar[int]
    ID_FIELD_NUMBER: _ClassVar[int]
    REVISION_FIELD_NUMBER: _ClassVar[int]
    workspace_id: str
    id: str
    revision: int
    def __init__(self, workspace_id: _Optional[str] = ..., id: _Optional[str] = ..., revision: _Optional[int] = ...) -> None: ...

class GetScheduleResponse(_message.Message):
    __slots__ = ("schedule",)
    SCHEDULE_FIELD_NUMBER: _ClassVar[int]
    schedule: Schedule
    def __init__(self, schedule: _Optional[_Union[Schedule, _Mapping]] = ...) -> None: ...
