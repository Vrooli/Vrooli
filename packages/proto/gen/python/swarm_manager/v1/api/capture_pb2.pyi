from buf.validate import validate_pb2 as _validate_pb2
from swarm_manager.v1.domain import capture_pb2 as _capture_pb2
from google.protobuf.internal import containers as _containers
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Iterable as _Iterable, Mapping as _Mapping
from typing import ClassVar as _ClassVar, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class CreateCaptureRequest(_message.Message):
    __slots__ = ("text",)
    TEXT_FIELD_NUMBER: _ClassVar[int]
    text: str
    def __init__(self, text: _Optional[str] = ...) -> None: ...

class CaptureResponse(_message.Message):
    __slots__ = ("capture", "task_id", "run_id", "base_url")
    CAPTURE_FIELD_NUMBER: _ClassVar[int]
    TASK_ID_FIELD_NUMBER: _ClassVar[int]
    RUN_ID_FIELD_NUMBER: _ClassVar[int]
    BASE_URL_FIELD_NUMBER: _ClassVar[int]
    capture: _capture_pb2.Capture
    task_id: str
    run_id: str
    base_url: str
    def __init__(self, capture: _Optional[_Union[_capture_pb2.Capture, _Mapping]] = ..., task_id: _Optional[str] = ..., run_id: _Optional[str] = ..., base_url: _Optional[str] = ...) -> None: ...

class ListCapturesResponse(_message.Message):
    __slots__ = ("captures",)
    CAPTURES_FIELD_NUMBER: _ClassVar[int]
    captures: _containers.RepeatedCompositeFieldContainer[_capture_pb2.Capture]
    def __init__(self, captures: _Optional[_Iterable[_Union[_capture_pb2.Capture, _Mapping]]] = ...) -> None: ...

class ClassifyCaptureResponse(_message.Message):
    __slots__ = ("task_id", "run_id", "base_url", "created")
    TASK_ID_FIELD_NUMBER: _ClassVar[int]
    RUN_ID_FIELD_NUMBER: _ClassVar[int]
    BASE_URL_FIELD_NUMBER: _ClassVar[int]
    CREATED_FIELD_NUMBER: _ClassVar[int]
    task_id: str
    run_id: str
    base_url: str
    created: str
    def __init__(self, task_id: _Optional[str] = ..., run_id: _Optional[str] = ..., base_url: _Optional[str] = ..., created: _Optional[str] = ...) -> None: ...
