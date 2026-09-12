import datetime

from buf.validate import validate_pb2 as _validate_pb2
from google.protobuf import struct_pb2 as _struct_pb2
from google.protobuf import timestamp_pb2 as _timestamp_pb2
from google.protobuf.internal import containers as _containers
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Iterable as _Iterable, Mapping as _Mapping
from typing import ClassVar as _ClassVar, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class WorkflowSchedule(_message.Message):
    __slots__ = ("id", "workflow_id", "name", "description", "cron_expression", "timezone", "is_active", "parameters", "next_run_at", "last_run_at", "created_at", "updated_at", "workflow_name", "next_run_human", "last_run_status")
    ID_FIELD_NUMBER: _ClassVar[int]
    WORKFLOW_ID_FIELD_NUMBER: _ClassVar[int]
    NAME_FIELD_NUMBER: _ClassVar[int]
    DESCRIPTION_FIELD_NUMBER: _ClassVar[int]
    CRON_EXPRESSION_FIELD_NUMBER: _ClassVar[int]
    TIMEZONE_FIELD_NUMBER: _ClassVar[int]
    IS_ACTIVE_FIELD_NUMBER: _ClassVar[int]
    PARAMETERS_FIELD_NUMBER: _ClassVar[int]
    NEXT_RUN_AT_FIELD_NUMBER: _ClassVar[int]
    LAST_RUN_AT_FIELD_NUMBER: _ClassVar[int]
    CREATED_AT_FIELD_NUMBER: _ClassVar[int]
    UPDATED_AT_FIELD_NUMBER: _ClassVar[int]
    WORKFLOW_NAME_FIELD_NUMBER: _ClassVar[int]
    NEXT_RUN_HUMAN_FIELD_NUMBER: _ClassVar[int]
    LAST_RUN_STATUS_FIELD_NUMBER: _ClassVar[int]
    id: str
    workflow_id: str
    name: str
    description: str
    cron_expression: str
    timezone: str
    is_active: bool
    parameters: _struct_pb2.Struct
    next_run_at: _timestamp_pb2.Timestamp
    last_run_at: _timestamp_pb2.Timestamp
    created_at: _timestamp_pb2.Timestamp
    updated_at: _timestamp_pb2.Timestamp
    workflow_name: str
    next_run_human: str
    last_run_status: str
    def __init__(self, id: _Optional[str] = ..., workflow_id: _Optional[str] = ..., name: _Optional[str] = ..., description: _Optional[str] = ..., cron_expression: _Optional[str] = ..., timezone: _Optional[str] = ..., is_active: _Optional[bool] = ..., parameters: _Optional[_Union[_struct_pb2.Struct, _Mapping]] = ..., next_run_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., last_run_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., created_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., updated_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., workflow_name: _Optional[str] = ..., next_run_human: _Optional[str] = ..., last_run_status: _Optional[str] = ...) -> None: ...

class CreateScheduleRequest(_message.Message):
    __slots__ = ("workflow_id", "name", "description", "cron_expression", "timezone", "parameters", "is_active")
    WORKFLOW_ID_FIELD_NUMBER: _ClassVar[int]
    NAME_FIELD_NUMBER: _ClassVar[int]
    DESCRIPTION_FIELD_NUMBER: _ClassVar[int]
    CRON_EXPRESSION_FIELD_NUMBER: _ClassVar[int]
    TIMEZONE_FIELD_NUMBER: _ClassVar[int]
    PARAMETERS_FIELD_NUMBER: _ClassVar[int]
    IS_ACTIVE_FIELD_NUMBER: _ClassVar[int]
    workflow_id: str
    name: str
    description: str
    cron_expression: str
    timezone: str
    parameters: _struct_pb2.Struct
    is_active: bool
    def __init__(self, workflow_id: _Optional[str] = ..., name: _Optional[str] = ..., description: _Optional[str] = ..., cron_expression: _Optional[str] = ..., timezone: _Optional[str] = ..., parameters: _Optional[_Union[_struct_pb2.Struct, _Mapping]] = ..., is_active: _Optional[bool] = ...) -> None: ...

class ListByWorkflowRequest(_message.Message):
    __slots__ = ("workflow_id", "active_only", "limit", "offset")
    WORKFLOW_ID_FIELD_NUMBER: _ClassVar[int]
    ACTIVE_ONLY_FIELD_NUMBER: _ClassVar[int]
    LIMIT_FIELD_NUMBER: _ClassVar[int]
    OFFSET_FIELD_NUMBER: _ClassVar[int]
    workflow_id: str
    active_only: bool
    limit: int
    offset: int
    def __init__(self, workflow_id: _Optional[str] = ..., active_only: _Optional[bool] = ..., limit: _Optional[int] = ..., offset: _Optional[int] = ...) -> None: ...

class ListSchedulesRequest(_message.Message):
    __slots__ = ("workflow_id", "active_only", "limit", "offset")
    WORKFLOW_ID_FIELD_NUMBER: _ClassVar[int]
    ACTIVE_ONLY_FIELD_NUMBER: _ClassVar[int]
    LIMIT_FIELD_NUMBER: _ClassVar[int]
    OFFSET_FIELD_NUMBER: _ClassVar[int]
    workflow_id: str
    active_only: bool
    limit: int
    offset: int
    def __init__(self, workflow_id: _Optional[str] = ..., active_only: _Optional[bool] = ..., limit: _Optional[int] = ..., offset: _Optional[int] = ...) -> None: ...

class ListSchedulesResponse(_message.Message):
    __slots__ = ("schedules", "total")
    SCHEDULES_FIELD_NUMBER: _ClassVar[int]
    TOTAL_FIELD_NUMBER: _ClassVar[int]
    schedules: _containers.RepeatedCompositeFieldContainer[WorkflowSchedule]
    total: int
    def __init__(self, schedules: _Optional[_Iterable[_Union[WorkflowSchedule, _Mapping]]] = ..., total: _Optional[int] = ...) -> None: ...

class GetScheduleRequest(_message.Message):
    __slots__ = ("schedule_id",)
    SCHEDULE_ID_FIELD_NUMBER: _ClassVar[int]
    schedule_id: str
    def __init__(self, schedule_id: _Optional[str] = ...) -> None: ...

class ScheduleResponse(_message.Message):
    __slots__ = ("schedule",)
    SCHEDULE_FIELD_NUMBER: _ClassVar[int]
    schedule: WorkflowSchedule
    def __init__(self, schedule: _Optional[_Union[WorkflowSchedule, _Mapping]] = ...) -> None: ...

class UpdateScheduleRequest(_message.Message):
    __slots__ = ("schedule_id", "name", "description", "cron_expression", "timezone", "parameters", "is_active")
    SCHEDULE_ID_FIELD_NUMBER: _ClassVar[int]
    NAME_FIELD_NUMBER: _ClassVar[int]
    DESCRIPTION_FIELD_NUMBER: _ClassVar[int]
    CRON_EXPRESSION_FIELD_NUMBER: _ClassVar[int]
    TIMEZONE_FIELD_NUMBER: _ClassVar[int]
    PARAMETERS_FIELD_NUMBER: _ClassVar[int]
    IS_ACTIVE_FIELD_NUMBER: _ClassVar[int]
    schedule_id: str
    name: str
    description: str
    cron_expression: str
    timezone: str
    parameters: _struct_pb2.Struct
    is_active: bool
    def __init__(self, schedule_id: _Optional[str] = ..., name: _Optional[str] = ..., description: _Optional[str] = ..., cron_expression: _Optional[str] = ..., timezone: _Optional[str] = ..., parameters: _Optional[_Union[_struct_pb2.Struct, _Mapping]] = ..., is_active: _Optional[bool] = ...) -> None: ...

class DeleteScheduleRequest(_message.Message):
    __slots__ = ("schedule_id",)
    SCHEDULE_ID_FIELD_NUMBER: _ClassVar[int]
    schedule_id: str
    def __init__(self, schedule_id: _Optional[str] = ...) -> None: ...

class DeleteScheduleResponse(_message.Message):
    __slots__ = ("schedule_id",)
    SCHEDULE_ID_FIELD_NUMBER: _ClassVar[int]
    schedule_id: str
    def __init__(self, schedule_id: _Optional[str] = ...) -> None: ...

class ToggleScheduleRequest(_message.Message):
    __slots__ = ("schedule_id",)
    SCHEDULE_ID_FIELD_NUMBER: _ClassVar[int]
    schedule_id: str
    def __init__(self, schedule_id: _Optional[str] = ...) -> None: ...

class ToggleScheduleResponse(_message.Message):
    __slots__ = ("schedule_id", "is_active", "schedule")
    SCHEDULE_ID_FIELD_NUMBER: _ClassVar[int]
    IS_ACTIVE_FIELD_NUMBER: _ClassVar[int]
    SCHEDULE_FIELD_NUMBER: _ClassVar[int]
    schedule_id: str
    is_active: bool
    schedule: WorkflowSchedule
    def __init__(self, schedule_id: _Optional[str] = ..., is_active: _Optional[bool] = ..., schedule: _Optional[_Union[WorkflowSchedule, _Mapping]] = ...) -> None: ...

class TriggerScheduleRequest(_message.Message):
    __slots__ = ("schedule_id",)
    SCHEDULE_ID_FIELD_NUMBER: _ClassVar[int]
    schedule_id: str
    def __init__(self, schedule_id: _Optional[str] = ...) -> None: ...

class TriggerScheduleResponse(_message.Message):
    __slots__ = ("schedule_id", "execution_id", "workflow_id", "triggered_at")
    SCHEDULE_ID_FIELD_NUMBER: _ClassVar[int]
    EXECUTION_ID_FIELD_NUMBER: _ClassVar[int]
    WORKFLOW_ID_FIELD_NUMBER: _ClassVar[int]
    TRIGGERED_AT_FIELD_NUMBER: _ClassVar[int]
    schedule_id: str
    execution_id: str
    workflow_id: str
    triggered_at: _timestamp_pb2.Timestamp
    def __init__(self, schedule_id: _Optional[str] = ..., execution_id: _Optional[str] = ..., workflow_id: _Optional[str] = ..., triggered_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ...) -> None: ...

class ScheduleMutationResponse(_message.Message):
    __slots__ = ("schedule_id", "status", "schedule")
    SCHEDULE_ID_FIELD_NUMBER: _ClassVar[int]
    STATUS_FIELD_NUMBER: _ClassVar[int]
    SCHEDULE_FIELD_NUMBER: _ClassVar[int]
    schedule_id: str
    status: str
    schedule: WorkflowSchedule
    def __init__(self, schedule_id: _Optional[str] = ..., status: _Optional[str] = ..., schedule: _Optional[_Union[WorkflowSchedule, _Mapping]] = ...) -> None: ...

class OccurrencesRequest(_message.Message):
    __slots__ = ("start", "end", "max_per_schedule", "workflow_id")
    START_FIELD_NUMBER: _ClassVar[int]
    END_FIELD_NUMBER: _ClassVar[int]
    MAX_PER_SCHEDULE_FIELD_NUMBER: _ClassVar[int]
    WORKFLOW_ID_FIELD_NUMBER: _ClassVar[int]
    start: _timestamp_pb2.Timestamp
    end: _timestamp_pb2.Timestamp
    max_per_schedule: int
    workflow_id: str
    def __init__(self, start: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., end: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., max_per_schedule: _Optional[int] = ..., workflow_id: _Optional[str] = ...) -> None: ...

class ScheduleOccurrence(_message.Message):
    __slots__ = ("schedule_id", "schedule_name", "workflow_id", "workflow_name", "run_at", "is_active", "cron_expression", "timezone")
    SCHEDULE_ID_FIELD_NUMBER: _ClassVar[int]
    SCHEDULE_NAME_FIELD_NUMBER: _ClassVar[int]
    WORKFLOW_ID_FIELD_NUMBER: _ClassVar[int]
    WORKFLOW_NAME_FIELD_NUMBER: _ClassVar[int]
    RUN_AT_FIELD_NUMBER: _ClassVar[int]
    IS_ACTIVE_FIELD_NUMBER: _ClassVar[int]
    CRON_EXPRESSION_FIELD_NUMBER: _ClassVar[int]
    TIMEZONE_FIELD_NUMBER: _ClassVar[int]
    schedule_id: str
    schedule_name: str
    workflow_id: str
    workflow_name: str
    run_at: _timestamp_pb2.Timestamp
    is_active: bool
    cron_expression: str
    timezone: str
    def __init__(self, schedule_id: _Optional[str] = ..., schedule_name: _Optional[str] = ..., workflow_id: _Optional[str] = ..., workflow_name: _Optional[str] = ..., run_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., is_active: _Optional[bool] = ..., cron_expression: _Optional[str] = ..., timezone: _Optional[str] = ...) -> None: ...

class ScheduleAggregate(_message.Message):
    __slots__ = ("schedule_id", "schedule_name", "total_runs", "truncated", "cron_expression")
    SCHEDULE_ID_FIELD_NUMBER: _ClassVar[int]
    SCHEDULE_NAME_FIELD_NUMBER: _ClassVar[int]
    TOTAL_RUNS_FIELD_NUMBER: _ClassVar[int]
    TRUNCATED_FIELD_NUMBER: _ClassVar[int]
    CRON_EXPRESSION_FIELD_NUMBER: _ClassVar[int]
    schedule_id: str
    schedule_name: str
    total_runs: int
    truncated: bool
    cron_expression: str
    def __init__(self, schedule_id: _Optional[str] = ..., schedule_name: _Optional[str] = ..., total_runs: _Optional[int] = ..., truncated: _Optional[bool] = ..., cron_expression: _Optional[str] = ...) -> None: ...

class OccurrencesResponse(_message.Message):
    __slots__ = ("occurrences", "aggregates", "total", "range_start", "range_end")
    class AggregatesEntry(_message.Message):
        __slots__ = ("key", "value")
        KEY_FIELD_NUMBER: _ClassVar[int]
        VALUE_FIELD_NUMBER: _ClassVar[int]
        key: str
        value: ScheduleAggregate
        def __init__(self, key: _Optional[str] = ..., value: _Optional[_Union[ScheduleAggregate, _Mapping]] = ...) -> None: ...
    OCCURRENCES_FIELD_NUMBER: _ClassVar[int]
    AGGREGATES_FIELD_NUMBER: _ClassVar[int]
    TOTAL_FIELD_NUMBER: _ClassVar[int]
    RANGE_START_FIELD_NUMBER: _ClassVar[int]
    RANGE_END_FIELD_NUMBER: _ClassVar[int]
    occurrences: _containers.RepeatedCompositeFieldContainer[ScheduleOccurrence]
    aggregates: _containers.MessageMap[str, ScheduleAggregate]
    total: int
    range_start: _timestamp_pb2.Timestamp
    range_end: _timestamp_pb2.Timestamp
    def __init__(self, occurrences: _Optional[_Iterable[_Union[ScheduleOccurrence, _Mapping]]] = ..., aggregates: _Optional[_Mapping[str, ScheduleAggregate]] = ..., total: _Optional[int] = ..., range_start: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., range_end: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ...) -> None: ...
