from browser_automation_studio.v1 import timeline_entry_pb2 as _timeline_entry_pb2
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Mapping as _Mapping
from typing import ClassVar as _ClassVar, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class TimelineEvent(_message.Message):
    __slots__ = ()
    ENTRY_FIELD_NUMBER: _ClassVar[int]
    entry: _timeline_entry_pb2.TimelineEntry
    def __init__(self, entry: _Optional[_Union[_timeline_entry_pb2.TimelineEntry, _Mapping]] = ...) -> None: ...

class RecordingContext(_message.Message):
    __slots__ = ()
    SESSION_ID_FIELD_NUMBER: _ClassVar[int]
    session_id: str
    def __init__(self, session_id: _Optional[str] = ...) -> None: ...

class ExecutionContext(_message.Message):
    __slots__ = ()
    EXECUTION_ID_FIELD_NUMBER: _ClassVar[int]
    execution_id: str
    def __init__(self, execution_id: _Optional[str] = ...) -> None: ...
