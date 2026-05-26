from google.protobuf.internal import enum_type_wrapper as _enum_type_wrapper
from google.protobuf import descriptor as _descriptor
from typing import ClassVar as _ClassVar

DESCRIPTOR: _descriptor.FileDescriptor

class SourceKind(int, metaclass=_enum_type_wrapper.EnumTypeWrapper):
    __slots__ = ()
    SOURCE_KIND_UNSPECIFIED: _ClassVar[SourceKind]
    SOURCE_KIND_FILESYSTEM: _ClassVar[SourceKind]
    SOURCE_KIND_SQLITE: _ClassVar[SourceKind]
    SOURCE_KIND_POSTGRES: _ClassVar[SourceKind]
    SOURCE_KIND_REDIS: _ClassVar[SourceKind]
    SOURCE_KIND_QDRANT: _ClassVar[SourceKind]
    SOURCE_KIND_OBJECT_STORAGE: _ClassVar[SourceKind]
SOURCE_KIND_UNSPECIFIED: SourceKind
SOURCE_KIND_FILESYSTEM: SourceKind
SOURCE_KIND_SQLITE: SourceKind
SOURCE_KIND_POSTGRES: SourceKind
SOURCE_KIND_REDIS: SourceKind
SOURCE_KIND_QDRANT: SourceKind
SOURCE_KIND_OBJECT_STORAGE: SourceKind
