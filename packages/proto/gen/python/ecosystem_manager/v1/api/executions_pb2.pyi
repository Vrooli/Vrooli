from ecosystem_manager.v1.domain import queue_pb2 as _queue_pb2
from google.protobuf.internal import containers as _containers
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Iterable as _Iterable, Mapping as _Mapping
from typing import ClassVar as _ClassVar, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class ExecutionHistoryListResponse(_message.Message):
    __slots__ = ("executions", "count")
    EXECUTIONS_FIELD_NUMBER: _ClassVar[int]
    COUNT_FIELD_NUMBER: _ClassVar[int]
    executions: _containers.RepeatedCompositeFieldContainer[_queue_pb2.ExecutionRecord]
    count: int
    def __init__(self, executions: _Optional[_Iterable[_Union[_queue_pb2.ExecutionRecord, _Mapping]]] = ..., count: _Optional[int] = ...) -> None: ...

class ExecutionHistoryResponse(_message.Message):
    __slots__ = ("execution",)
    EXECUTION_FIELD_NUMBER: _ClassVar[int]
    execution: _queue_pb2.ExecutionRecord
    def __init__(self, execution: _Optional[_Union[_queue_pb2.ExecutionRecord, _Mapping]] = ...) -> None: ...

class ExecutionPromptResponse(_message.Message):
    __slots__ = ("prompt", "content", "size")
    PROMPT_FIELD_NUMBER: _ClassVar[int]
    CONTENT_FIELD_NUMBER: _ClassVar[int]
    SIZE_FIELD_NUMBER: _ClassVar[int]
    prompt: str
    content: str
    size: int
    def __init__(self, prompt: _Optional[str] = ..., content: _Optional[str] = ..., size: _Optional[int] = ...) -> None: ...

class ExecutionOutputResponse(_message.Message):
    __slots__ = ("output", "content", "size")
    OUTPUT_FIELD_NUMBER: _ClassVar[int]
    CONTENT_FIELD_NUMBER: _ClassVar[int]
    SIZE_FIELD_NUMBER: _ClassVar[int]
    output: str
    content: str
    size: int
    def __init__(self, output: _Optional[str] = ..., content: _Optional[str] = ..., size: _Optional[int] = ...) -> None: ...
