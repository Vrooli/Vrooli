from google.protobuf.internal import enum_type_wrapper as _enum_type_wrapper
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from typing import ClassVar as _ClassVar, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class Provenance(int, metaclass=_enum_type_wrapper.EnumTypeWrapper):
    __slots__ = ()
    PROVENANCE_UNSPECIFIED: _ClassVar[Provenance]
    PROVENANCE_CUSTOM: _ClassVar[Provenance]
    PROVENANCE_ADOPTED_UNMODIFIED: _ClassVar[Provenance]
    PROVENANCE_ADOPTED_MODIFIED: _ClassVar[Provenance]
    PROVENANCE_UNKNOWN: _ClassVar[Provenance]
PROVENANCE_UNSPECIFIED: Provenance
PROVENANCE_CUSTOM: Provenance
PROVENANCE_ADOPTED_UNMODIFIED: Provenance
PROVENANCE_ADOPTED_MODIFIED: Provenance
PROVENANCE_UNKNOWN: Provenance

class ComponentProvenance(_message.Message):
    __slots__ = ("provenance", "library", "library_version", "component_name", "adoption_id", "applied_at", "source_sha256", "drift_hash", "file_path")
    PROVENANCE_FIELD_NUMBER: _ClassVar[int]
    LIBRARY_FIELD_NUMBER: _ClassVar[int]
    LIBRARY_VERSION_FIELD_NUMBER: _ClassVar[int]
    COMPONENT_NAME_FIELD_NUMBER: _ClassVar[int]
    ADOPTION_ID_FIELD_NUMBER: _ClassVar[int]
    APPLIED_AT_FIELD_NUMBER: _ClassVar[int]
    SOURCE_SHA256_FIELD_NUMBER: _ClassVar[int]
    DRIFT_HASH_FIELD_NUMBER: _ClassVar[int]
    FILE_PATH_FIELD_NUMBER: _ClassVar[int]
    provenance: Provenance
    library: str
    library_version: str
    component_name: str
    adoption_id: str
    applied_at: str
    source_sha256: str
    drift_hash: str
    file_path: str
    def __init__(self, provenance: _Optional[_Union[Provenance, str]] = ..., library: _Optional[str] = ..., library_version: _Optional[str] = ..., component_name: _Optional[str] = ..., adoption_id: _Optional[str] = ..., applied_at: _Optional[str] = ..., source_sha256: _Optional[str] = ..., drift_hash: _Optional[str] = ..., file_path: _Optional[str] = ...) -> None: ...
