from program_runtime.v1.shared import library_pb2 as _library_pb2
from google.protobuf.internal import containers as _containers
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Iterable as _Iterable, Mapping as _Mapping
from typing import ClassVar as _ClassVar, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class ListLibraryRequest(_message.Message):
    __slots__ = ()
    def __init__(self) -> None: ...

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

class GetLibraryResponse(_message.Message):
    __slots__ = ("program",)
    PROGRAM_FIELD_NUMBER: _ClassVar[int]
    program: _library_pb2.LibraryProgram
    def __init__(self, program: _Optional[_Union[_library_pb2.LibraryProgram, _Mapping]] = ...) -> None: ...

class PromoteLibraryRequest(_message.Message):
    __slots__ = ("program_id", "name", "description", "promoted_by", "reason")
    PROGRAM_ID_FIELD_NUMBER: _ClassVar[int]
    NAME_FIELD_NUMBER: _ClassVar[int]
    DESCRIPTION_FIELD_NUMBER: _ClassVar[int]
    PROMOTED_BY_FIELD_NUMBER: _ClassVar[int]
    REASON_FIELD_NUMBER: _ClassVar[int]
    program_id: str
    name: str
    description: str
    promoted_by: str
    reason: str
    def __init__(self, program_id: _Optional[str] = ..., name: _Optional[str] = ..., description: _Optional[str] = ..., promoted_by: _Optional[str] = ..., reason: _Optional[str] = ...) -> None: ...

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
