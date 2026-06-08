from ecosystem_manager.v1.domain import queue_pb2 as _queue_pb2
from ecosystem_manager.v1.domain import task_pb2 as _task_pb2
from google.protobuf import struct_pb2 as _struct_pb2
from google.protobuf.internal import containers as _containers
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Iterable as _Iterable, Mapping as _Mapping
from typing import ClassVar as _ClassVar, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class QueueStatusResponse(_message.Message):
    __slots__ = ("status",)
    STATUS_FIELD_NUMBER: _ClassVar[int]
    status: _queue_pb2.QueueStatus
    def __init__(self, status: _Optional[_Union[_queue_pb2.QueueStatus, _Mapping]] = ...) -> None: ...

class ProcessListResponse(_message.Message):
    __slots__ = ("processes", "count", "timestamp")
    PROCESSES_FIELD_NUMBER: _ClassVar[int]
    COUNT_FIELD_NUMBER: _ClassVar[int]
    TIMESTAMP_FIELD_NUMBER: _ClassVar[int]
    processes: _containers.RepeatedCompositeFieldContainer[_task_pb2.ProcessInfo]
    count: int
    timestamp: int
    def __init__(self, processes: _Optional[_Iterable[_Union[_task_pb2.ProcessInfo, _Mapping]]] = ..., count: _Optional[int] = ..., timestamp: _Optional[int] = ...) -> None: ...

class QueueTriggerResponse(_message.Message):
    __slots__ = ("success", "message", "timestamp", "status")
    SUCCESS_FIELD_NUMBER: _ClassVar[int]
    MESSAGE_FIELD_NUMBER: _ClassVar[int]
    TIMESTAMP_FIELD_NUMBER: _ClassVar[int]
    STATUS_FIELD_NUMBER: _ClassVar[int]
    success: bool
    message: str
    timestamp: int
    status: _queue_pb2.QueueStatus
    def __init__(self, success: _Optional[bool] = ..., message: _Optional[str] = ..., timestamp: _Optional[int] = ..., status: _Optional[_Union[_queue_pb2.QueueStatus, _Mapping]] = ...) -> None: ...

class QueuePauseResponse(_message.Message):
    __slots__ = ("success", "message", "paused")
    SUCCESS_FIELD_NUMBER: _ClassVar[int]
    MESSAGE_FIELD_NUMBER: _ClassVar[int]
    PAUSED_FIELD_NUMBER: _ClassVar[int]
    success: bool
    message: str
    paused: bool
    def __init__(self, success: _Optional[bool] = ..., message: _Optional[str] = ..., paused: _Optional[bool] = ...) -> None: ...

class ResumeDiagnosticsResponse(_message.Message):
    __slots__ = ("success", "diagnostics", "generated_at")
    SUCCESS_FIELD_NUMBER: _ClassVar[int]
    DIAGNOSTICS_FIELD_NUMBER: _ClassVar[int]
    GENERATED_AT_FIELD_NUMBER: _ClassVar[int]
    success: bool
    diagnostics: _struct_pb2.Struct
    generated_at: str
    def __init__(self, success: _Optional[bool] = ..., diagnostics: _Optional[_Union[_struct_pb2.Struct, _Mapping]] = ..., generated_at: _Optional[str] = ...) -> None: ...

class ResetResponse(_message.Message):
    __slots__ = ("success", "message", "summary")
    SUCCESS_FIELD_NUMBER: _ClassVar[int]
    MESSAGE_FIELD_NUMBER: _ClassVar[int]
    SUMMARY_FIELD_NUMBER: _ClassVar[int]
    success: bool
    message: str
    summary: str
    def __init__(self, success: _Optional[bool] = ..., message: _Optional[str] = ..., summary: _Optional[str] = ...) -> None: ...
