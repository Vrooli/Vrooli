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
    PROVENANCE_TEST: _ClassVar[Provenance]
    PROVENANCE_REPLAY: _ClassVar[Provenance]

class ProgramStatus(int, metaclass=_enum_type_wrapper.EnumTypeWrapper):
    __slots__ = ()
    PROGRAM_STATUS_UNSPECIFIED: _ClassVar[ProgramStatus]
    PROGRAM_STATUS_ACCEPTED: _ClassVar[ProgramStatus]
    PROGRAM_STATUS_RUNNING: _ClassVar[ProgramStatus]
    PROGRAM_STATUS_SUCCEEDED: _ClassVar[ProgramStatus]
    PROGRAM_STATUS_FAILED: _ClassVar[ProgramStatus]
    PROGRAM_STATUS_CANCELLED: _ClassVar[ProgramStatus]
PROVENANCE_UNSPECIFIED: Provenance
PROVENANCE_AGENT: Provenance
PROVENANCE_OPERATOR: Provenance
PROVENANCE_TEST: Provenance
PROVENANCE_REPLAY: Provenance
PROGRAM_STATUS_UNSPECIFIED: ProgramStatus
PROGRAM_STATUS_ACCEPTED: ProgramStatus
PROGRAM_STATUS_RUNNING: ProgramStatus
PROGRAM_STATUS_SUCCEEDED: ProgramStatus
PROGRAM_STATUS_FAILED: ProgramStatus
PROGRAM_STATUS_CANCELLED: ProgramStatus

class Program(_message.Message):
    __slots__ = ("id", "session_id", "source", "provenance", "status", "stdout", "failure_detail", "failure_shape", "context_bytes", "created_at", "output_limit_bytes", "agent_bytes", "completed_at", "wall_time_millis", "cpu_time_millis", "library_version")
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
    OUTPUT_LIMIT_BYTES_FIELD_NUMBER: _ClassVar[int]
    AGENT_BYTES_FIELD_NUMBER: _ClassVar[int]
    COMPLETED_AT_FIELD_NUMBER: _ClassVar[int]
    WALL_TIME_MILLIS_FIELD_NUMBER: _ClassVar[int]
    CPU_TIME_MILLIS_FIELD_NUMBER: _ClassVar[int]
    LIBRARY_VERSION_FIELD_NUMBER: _ClassVar[int]
    id: str
    session_id: str
    source: str
    provenance: Provenance
    status: ProgramStatus
    stdout: str
    failure_detail: str
    failure_shape: str
    context_bytes: int
    created_at: str
    output_limit_bytes: int
    agent_bytes: int
    completed_at: str
    wall_time_millis: int
    cpu_time_millis: int
    library_version: str
    def __init__(self, id: _Optional[str] = ..., session_id: _Optional[str] = ..., source: _Optional[str] = ..., provenance: _Optional[_Union[Provenance, str]] = ..., status: _Optional[_Union[ProgramStatus, str]] = ..., stdout: _Optional[str] = ..., failure_detail: _Optional[str] = ..., failure_shape: _Optional[str] = ..., context_bytes: _Optional[int] = ..., created_at: _Optional[str] = ..., output_limit_bytes: _Optional[int] = ..., agent_bytes: _Optional[int] = ..., completed_at: _Optional[str] = ..., wall_time_millis: _Optional[int] = ..., cpu_time_millis: _Optional[int] = ..., library_version: _Optional[str] = ...) -> None: ...

class SubmitProgramRequest(_message.Message):
    __slots__ = ("session_id", "source", "provenance", "include_materialized")
    SESSION_ID_FIELD_NUMBER: _ClassVar[int]
    SOURCE_FIELD_NUMBER: _ClassVar[int]
    PROVENANCE_FIELD_NUMBER: _ClassVar[int]
    INCLUDE_MATERIALIZED_FIELD_NUMBER: _ClassVar[int]
    ASYNC_FIELD_NUMBER: _ClassVar[int]
    session_id: str
    source: str
    provenance: Provenance
    include_materialized: bool
    def __init__(self, session_id: _Optional[str] = ..., source: _Optional[str] = ..., provenance: _Optional[_Union[Provenance, str]] = ..., include_materialized: _Optional[bool] = ..., **kwargs) -> None: ...

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
    __slots__ = ("shape", "count", "first_seen", "last_seen", "sample_program_id")
    SHAPE_FIELD_NUMBER: _ClassVar[int]
    COUNT_FIELD_NUMBER: _ClassVar[int]
    FIRST_SEEN_FIELD_NUMBER: _ClassVar[int]
    LAST_SEEN_FIELD_NUMBER: _ClassVar[int]
    SAMPLE_PROGRAM_ID_FIELD_NUMBER: _ClassVar[int]
    shape: str
    count: int
    first_seen: str
    last_seen: str
    sample_program_id: str
    def __init__(self, shape: _Optional[str] = ..., count: _Optional[int] = ..., first_seen: _Optional[str] = ..., last_seen: _Optional[str] = ..., sample_program_id: _Optional[str] = ...) -> None: ...

class MineFailuresResponse(_message.Message):
    __slots__ = ("shapes", "count")
    SHAPES_FIELD_NUMBER: _ClassVar[int]
    COUNT_FIELD_NUMBER: _ClassVar[int]
    shapes: _containers.RepeatedCompositeFieldContainer[FailureShape]
    count: int
    def __init__(self, shapes: _Optional[_Iterable[_Union[FailureShape, _Mapping]]] = ..., count: _Optional[int] = ...) -> None: ...

class MineRefusalsRequest(_message.Message):
    __slots__ = ("include_operator",)
    INCLUDE_OPERATOR_FIELD_NUMBER: _ClassVar[int]
    include_operator: bool
    def __init__(self, include_operator: _Optional[bool] = ...) -> None: ...

class RefusalShape(_message.Message):
    __slots__ = ("binding_id", "reason", "count", "last_seen")
    BINDING_ID_FIELD_NUMBER: _ClassVar[int]
    REASON_FIELD_NUMBER: _ClassVar[int]
    COUNT_FIELD_NUMBER: _ClassVar[int]
    LAST_SEEN_FIELD_NUMBER: _ClassVar[int]
    binding_id: str
    reason: str
    count: int
    last_seen: str
    def __init__(self, binding_id: _Optional[str] = ..., reason: _Optional[str] = ..., count: _Optional[int] = ..., last_seen: _Optional[str] = ...) -> None: ...

class MineRefusalsResponse(_message.Message):
    __slots__ = ("shapes", "count")
    SHAPES_FIELD_NUMBER: _ClassVar[int]
    COUNT_FIELD_NUMBER: _ClassVar[int]
    shapes: _containers.RepeatedCompositeFieldContainer[RefusalShape]
    count: int
    def __init__(self, shapes: _Optional[_Iterable[_Union[RefusalShape, _Mapping]]] = ..., count: _Optional[int] = ...) -> None: ...

class MineUnresolvedBindingsRequest(_message.Message):
    __slots__ = ()
    def __init__(self) -> None: ...

class UnresolvedBindingShape(_message.Message):
    __slots__ = ("attempted_name", "count", "last_seen")
    ATTEMPTED_NAME_FIELD_NUMBER: _ClassVar[int]
    COUNT_FIELD_NUMBER: _ClassVar[int]
    LAST_SEEN_FIELD_NUMBER: _ClassVar[int]
    attempted_name: str
    count: int
    last_seen: str
    def __init__(self, attempted_name: _Optional[str] = ..., count: _Optional[int] = ..., last_seen: _Optional[str] = ...) -> None: ...

class MineUnresolvedBindingsResponse(_message.Message):
    __slots__ = ("shapes", "count")
    SHAPES_FIELD_NUMBER: _ClassVar[int]
    COUNT_FIELD_NUMBER: _ClassVar[int]
    shapes: _containers.RepeatedCompositeFieldContainer[UnresolvedBindingShape]
    count: int
    def __init__(self, shapes: _Optional[_Iterable[_Union[UnresolvedBindingShape, _Mapping]]] = ..., count: _Optional[int] = ...) -> None: ...
