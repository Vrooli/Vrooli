from program_runtime.v1.shared import shapes_pb2 as _shapes_pb2
from google.protobuf.internal import containers as _containers
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Iterable as _Iterable, Mapping as _Mapping
from typing import ClassVar as _ClassVar, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class ListShapesRequest(_message.Message):
    __slots__ = ("min_occurrences", "min_sessions", "uncovered_only", "state", "limit")
    MIN_OCCURRENCES_FIELD_NUMBER: _ClassVar[int]
    MIN_SESSIONS_FIELD_NUMBER: _ClassVar[int]
    UNCOVERED_ONLY_FIELD_NUMBER: _ClassVar[int]
    STATE_FIELD_NUMBER: _ClassVar[int]
    LIMIT_FIELD_NUMBER: _ClassVar[int]
    min_occurrences: int
    min_sessions: int
    uncovered_only: bool
    state: str
    limit: int
    def __init__(self, min_occurrences: _Optional[int] = ..., min_sessions: _Optional[int] = ..., uncovered_only: _Optional[bool] = ..., state: _Optional[str] = ..., limit: _Optional[int] = ...) -> None: ...

class ListShapesResponse(_message.Message):
    __slots__ = ("shapes", "observed", "nominated", "covered")
    SHAPES_FIELD_NUMBER: _ClassVar[int]
    OBSERVED_FIELD_NUMBER: _ClassVar[int]
    NOMINATED_FIELD_NUMBER: _ClassVar[int]
    COVERED_FIELD_NUMBER: _ClassVar[int]
    shapes: _containers.RepeatedCompositeFieldContainer[_shapes_pb2.ProgramShape]
    observed: int
    nominated: int
    covered: int
    def __init__(self, shapes: _Optional[_Iterable[_Union[_shapes_pb2.ProgramShape, _Mapping]]] = ..., observed: _Optional[int] = ..., nominated: _Optional[int] = ..., covered: _Optional[int] = ...) -> None: ...

class GetShapeRequest(_message.Message):
    __slots__ = ("shape_key",)
    SHAPE_KEY_FIELD_NUMBER: _ClassVar[int]
    shape_key: str
    def __init__(self, shape_key: _Optional[str] = ...) -> None: ...

class GetShapeResponse(_message.Message):
    __slots__ = ("shape",)
    SHAPE_FIELD_NUMBER: _ClassVar[int]
    shape: _shapes_pb2.ProgramShape
    def __init__(self, shape: _Optional[_Union[_shapes_pb2.ProgramShape, _Mapping]] = ...) -> None: ...

class ExpireShapesRequest(_message.Message):
    __slots__ = ("window_seconds",)
    WINDOW_SECONDS_FIELD_NUMBER: _ClassVar[int]
    window_seconds: int
    def __init__(self, window_seconds: _Optional[int] = ...) -> None: ...

class ExpireShapesResponse(_message.Message):
    __slots__ = ("deleted",)
    DELETED_FIELD_NUMBER: _ClassVar[int]
    deleted: int
    def __init__(self, deleted: _Optional[int] = ...) -> None: ...
