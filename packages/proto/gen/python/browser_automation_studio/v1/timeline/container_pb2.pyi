import datetime

from google.protobuf import timestamp_pb2 as _timestamp_pb2
from browser_automation_studio.v1.base import shared_pb2 as _shared_pb2
from browser_automation_studio.v1.timeline import entry_pb2 as _entry_pb2
from google.protobuf.internal import containers as _containers
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Iterable as _Iterable, Mapping as _Mapping
from typing import ClassVar as _ClassVar, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class ExecutionTimeline(_message.Message):
    __slots__ = ()
    EXECUTION_ID_FIELD_NUMBER: _ClassVar[int]
    WORKFLOW_ID_FIELD_NUMBER: _ClassVar[int]
    STATUS_FIELD_NUMBER: _ClassVar[int]
    PROGRESS_FIELD_NUMBER: _ClassVar[int]
    STARTED_AT_FIELD_NUMBER: _ClassVar[int]
    COMPLETED_AT_FIELD_NUMBER: _ClassVar[int]
    ENTRIES_FIELD_NUMBER: _ClassVar[int]
    LOGS_FIELD_NUMBER: _ClassVar[int]
    execution_id: str
    workflow_id: str
    status: _shared_pb2.ExecutionStatus
    progress: int
    started_at: _timestamp_pb2.Timestamp
    completed_at: _timestamp_pb2.Timestamp
    entries: _containers.RepeatedCompositeFieldContainer[_entry_pb2.TimelineEntry]
    logs: _containers.RepeatedCompositeFieldContainer[_entry_pb2.TimelineLog]
    def __init__(self, execution_id: _Optional[str] = ..., workflow_id: _Optional[str] = ..., status: _Optional[_Union[_shared_pb2.ExecutionStatus, str]] = ..., progress: _Optional[int] = ..., started_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., completed_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., entries: _Optional[_Iterable[_Union[_entry_pb2.TimelineEntry, _Mapping]]] = ..., logs: _Optional[_Iterable[_Union[_entry_pb2.TimelineLog, _Mapping]]] = ...) -> None: ...

class TimelineFrame(_message.Message):
    __slots__ = ()
    ENTRY_FIELD_NUMBER: _ClassVar[int]
    entry: _entry_pb2.TimelineEntry
    def __init__(self, entry: _Optional[_Union[_entry_pb2.TimelineEntry, _Mapping]] = ...) -> None: ...
