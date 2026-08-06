from google.protobuf.internal import containers as _containers
from google.protobuf.internal import enum_type_wrapper as _enum_type_wrapper
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Iterable as _Iterable, Mapping as _Mapping
from typing import ClassVar as _ClassVar, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class Provenance(int, metaclass=_enum_type_wrapper.EnumTypeWrapper):
    __slots__ = ()
    PROVENANCE_UNSPECIFIED: _ClassVar[Provenance]
    PROVENANCE_AGENT: _ClassVar[Provenance]
    PROVENANCE_OPERATOR: _ClassVar[Provenance]
PROVENANCE_UNSPECIFIED: Provenance
PROVENANCE_AGENT: Provenance
PROVENANCE_OPERATOR: Provenance

class Program(_message.Message):
    __slots__ = ("id", "session_id", "source", "provenance", "status", "stdout", "failure_detail", "failure_shape", "context_bytes", "created_at")
    ID_FIELD_NUMBER: _ClassVar[int]
    SESSION_ID_FIELD_NUMBER: _ClassVar[int]
    SOURCE_FIELD_NUMBER: _ClassVar[int]
    PROVENANCE_FIELD_NUMBER: _ClassVar[int]
    STATUS_FIELD_NUMBER: _ClassVar[int]
    STDOUT_FIELD_NUMBER: _ClassVar[int]
    FAILURE_DETAIL_FIELD_NUMBER: _ClassVar[int]
    FAILURE_SHAPE_FIELD_NUMBER: _ClassVar[int]
    CONTEXT_BYTES_FIELD_NUMBER: _ClassVar[int]
    CREATED_AT_FIELD_NUMBER: _ClassVar[int]
    id: str
    session_id: str
    source: str
    provenance: Provenance
    status: str
    stdout: str
    failure_detail: str
    failure_shape: str
    context_bytes: int
    created_at: str
    def __init__(self, id: _Optional[str] = ..., session_id: _Optional[str] = ..., source: _Optional[str] = ..., provenance: _Optional[_Union[Provenance, str]] = ..., status: _Optional[str] = ..., stdout: _Optional[str] = ..., failure_detail: _Optional[str] = ..., failure_shape: _Optional[str] = ..., context_bytes: _Optional[int] = ..., created_at: _Optional[str] = ...) -> None: ...

class SubmitProgramRequest(_message.Message):
    __slots__ = ("session_id", "source", "provenance", "include_materialized")
    SESSION_ID_FIELD_NUMBER: _ClassVar[int]
    SOURCE_FIELD_NUMBER: _ClassVar[int]
    PROVENANCE_FIELD_NUMBER: _ClassVar[int]
    INCLUDE_MATERIALIZED_FIELD_NUMBER: _ClassVar[int]
    session_id: str
    source: str
    provenance: Provenance
    include_materialized: bool
    def __init__(self, session_id: _Optional[str] = ..., source: _Optional[str] = ..., provenance: _Optional[_Union[Provenance, str]] = ..., include_materialized: _Optional[bool] = ...) -> None: ...

class SubmitProgramResponse(_message.Message):
    __slots__ = ("program",)
    PROGRAM_FIELD_NUMBER: _ClassVar[int]
    program: Program
    def __init__(self, program: _Optional[_Union[Program, _Mapping]] = ...) -> None: ...

class GetProgramRequest(_message.Message):
    __slots__ = ("id",)
    ID_FIELD_NUMBER: _ClassVar[int]
    id: str
    def __init__(self, id: _Optional[str] = ...) -> None: ...

class GetProgramResponse(_message.Message):
    __slots__ = ("program",)
    PROGRAM_FIELD_NUMBER: _ClassVar[int]
    program: Program
    def __init__(self, program: _Optional[_Union[Program, _Mapping]] = ...) -> None: ...

class ListProgramsRequest(_message.Message):
    __slots__ = ("session_id", "include_operator")
    SESSION_ID_FIELD_NUMBER: _ClassVar[int]
    INCLUDE_OPERATOR_FIELD_NUMBER: _ClassVar[int]
    session_id: str
    include_operator: bool
    def __init__(self, session_id: _Optional[str] = ..., include_operator: _Optional[bool] = ...) -> None: ...

class ListProgramsResponse(_message.Message):
    __slots__ = ("programs",)
    PROGRAMS_FIELD_NUMBER: _ClassVar[int]
    programs: _containers.RepeatedCompositeFieldContainer[Program]
    def __init__(self, programs: _Optional[_Iterable[_Union[Program, _Mapping]]] = ...) -> None: ...

class MineFailuresRequest(_message.Message):
    __slots__ = ("include_operator",)
    INCLUDE_OPERATOR_FIELD_NUMBER: _ClassVar[int]
    include_operator: bool
    def __init__(self, include_operator: _Optional[bool] = ...) -> None: ...

class FailureShape(_message.Message):
    __slots__ = ("shape", "count")
    SHAPE_FIELD_NUMBER: _ClassVar[int]
    COUNT_FIELD_NUMBER: _ClassVar[int]
    shape: str
    count: int
    def __init__(self, shape: _Optional[str] = ..., count: _Optional[int] = ...) -> None: ...

class MineFailuresResponse(_message.Message):
    __slots__ = ("shapes", "count")
    SHAPES_FIELD_NUMBER: _ClassVar[int]
    COUNT_FIELD_NUMBER: _ClassVar[int]
    shapes: _containers.RepeatedCompositeFieldContainer[FailureShape]
    count: int
    def __init__(self, shapes: _Optional[_Iterable[_Union[FailureShape, _Mapping]]] = ..., count: _Optional[int] = ...) -> None: ...
