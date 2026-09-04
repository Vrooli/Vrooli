from program_runtime.v1.shared import library_pb2 as _library_pb2
from google.protobuf.internal import containers as _containers
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Iterable as _Iterable, Mapping as _Mapping
from typing import ClassVar as _ClassVar, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class ListLibraryRequest(_message.Message):
    __slots__ = ("query", "limit", "offset")
    QUERY_FIELD_NUMBER: _ClassVar[int]
    LIMIT_FIELD_NUMBER: _ClassVar[int]
    OFFSET_FIELD_NUMBER: _ClassVar[int]
    query: str
    limit: int
    offset: int
    def __init__(self, query: _Optional[str] = ..., limit: _Optional[int] = ..., offset: _Optional[int] = ...) -> None: ...

class ListLibraryResponse(_message.Message):
    __slots__ = ("programs",)
    PROGRAMS_FIELD_NUMBER: _ClassVar[int]
    programs: _containers.RepeatedCompositeFieldContainer[_library_pb2.LibraryProgram]
    def __init__(self, programs: _Optional[_Iterable[_Union[_library_pb2.LibraryProgram, _Mapping]]] = ...) -> None: ...

class GetLibraryRequest(_message.Message):
    __slots__ = ("name", "version")
    NAME_FIELD_NUMBER: _ClassVar[int]
    VERSION_FIELD_NUMBER: _ClassVar[int]
    name: str
    version: int
    def __init__(self, name: _Optional[str] = ..., version: _Optional[int] = ...) -> None: ...

class BindingDrift(_message.Message):
    __slots__ = ("binding_id", "validated_at", "generation_mtime", "drift_status", "changed", "reason")
    BINDING_ID_FIELD_NUMBER: _ClassVar[int]
    VALIDATED_AT_FIELD_NUMBER: _ClassVar[int]
    GENERATION_MTIME_FIELD_NUMBER: _ClassVar[int]
    DRIFT_STATUS_FIELD_NUMBER: _ClassVar[int]
    CHANGED_FIELD_NUMBER: _ClassVar[int]
    REASON_FIELD_NUMBER: _ClassVar[int]
    binding_id: str
    validated_at: str
    generation_mtime: str
    drift_status: str
    changed: bool
    reason: str
    def __init__(self, binding_id: _Optional[str] = ..., validated_at: _Optional[str] = ..., generation_mtime: _Optional[str] = ..., drift_status: _Optional[str] = ..., changed: _Optional[bool] = ..., reason: _Optional[str] = ...) -> None: ...

class GetLibraryResponse(_message.Message):
    __slots__ = ("program", "drift")
    PROGRAM_FIELD_NUMBER: _ClassVar[int]
    DRIFT_FIELD_NUMBER: _ClassVar[int]
    program: _library_pb2.LibraryProgram
    drift: _containers.RepeatedCompositeFieldContainer[BindingDrift]
    def __init__(self, program: _Optional[_Union[_library_pb2.LibraryProgram, _Mapping]] = ..., drift: _Optional[_Iterable[_Union[BindingDrift, _Mapping]]] = ...) -> None: ...

class PromoteLibraryRequest(_message.Message):
    __slots__ = ("program_id", "name", "description", "promoted_by", "reason", "coverage", "declared_inputs", "declared_outputs")
    PROGRAM_ID_FIELD_NUMBER: _ClassVar[int]
    NAME_FIELD_NUMBER: _ClassVar[int]
    DESCRIPTION_FIELD_NUMBER: _ClassVar[int]
    PROMOTED_BY_FIELD_NUMBER: _ClassVar[int]
    REASON_FIELD_NUMBER: _ClassVar[int]
    COVERAGE_FIELD_NUMBER: _ClassVar[int]
    DECLARED_INPUTS_FIELD_NUMBER: _ClassVar[int]
    DECLARED_OUTPUTS_FIELD_NUMBER: _ClassVar[int]
    program_id: str
    name: str
    description: str
    promoted_by: str
    reason: str
    coverage: str
    declared_inputs: _containers.RepeatedScalarFieldContainer[str]
    declared_outputs: _containers.RepeatedScalarFieldContainer[str]
    def __init__(self, program_id: _Optional[str] = ..., name: _Optional[str] = ..., description: _Optional[str] = ..., promoted_by: _Optional[str] = ..., reason: _Optional[str] = ..., coverage: _Optional[str] = ..., declared_inputs: _Optional[_Iterable[str]] = ..., declared_outputs: _Optional[_Iterable[str]] = ...) -> None: ...

class PromoteLibraryResponse(_message.Message):
    __slots__ = ("program",)
    PROGRAM_FIELD_NUMBER: _ClassVar[int]
    program: _library_pb2.LibraryProgram
    def __init__(self, program: _Optional[_Union[_library_pb2.LibraryProgram, _Mapping]] = ...) -> None: ...

class SetCurrentLibraryRequest(_message.Message):
    __slots__ = ("name", "version")
    NAME_FIELD_NUMBER: _ClassVar[int]
    VERSION_FIELD_NUMBER: _ClassVar[int]
    name: str
    version: int
    def __init__(self, name: _Optional[str] = ..., version: _Optional[int] = ...) -> None: ...

class SetCurrentLibraryResponse(_message.Message):
    __slots__ = ("program",)
    PROGRAM_FIELD_NUMBER: _ClassVar[int]
    program: _library_pb2.LibraryProgram
    def __init__(self, program: _Optional[_Union[_library_pb2.LibraryProgram, _Mapping]] = ...) -> None: ...
