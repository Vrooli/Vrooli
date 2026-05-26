from buf.validate import validate_pb2 as _validate_pb2
from data_backup_manager.v1.destinations import destinations_pb2 as _destinations_pb2
from data_backup_manager.v1.sources import sources_pb2 as _sources_pb2
from google.protobuf.internal import containers as _containers
from google.protobuf.internal import enum_type_wrapper as _enum_type_wrapper
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Iterable as _Iterable, Mapping as _Mapping
from typing import ClassVar as _ClassVar, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class DriveClass(int, metaclass=_enum_type_wrapper.EnumTypeWrapper):
    __slots__ = ()
    DRIVE_CLASS_UNSPECIFIED: _ClassVar[DriveClass]
    DRIVE_CLASS_REMOVABLE: _ClassVar[DriveClass]
    DRIVE_CLASS_FIXED: _ClassVar[DriveClass]
    DRIVE_CLASS_NETWORK: _ClassVar[DriveClass]
DRIVE_CLASS_UNSPECIFIED: DriveClass
DRIVE_CLASS_REMOVABLE: DriveClass
DRIVE_CLASS_FIXED: DriveClass
DRIVE_CLASS_NETWORK: DriveClass

class TargetSuggestion(_message.Message):
    __slots__ = ("id", "owner", "name", "source_kind", "locator", "rationale", "approx_bytes")
    ID_FIELD_NUMBER: _ClassVar[int]
    OWNER_FIELD_NUMBER: _ClassVar[int]
    NAME_FIELD_NUMBER: _ClassVar[int]
    SOURCE_KIND_FIELD_NUMBER: _ClassVar[int]
    LOCATOR_FIELD_NUMBER: _ClassVar[int]
    RATIONALE_FIELD_NUMBER: _ClassVar[int]
    APPROX_BYTES_FIELD_NUMBER: _ClassVar[int]
    id: str
    owner: str
    name: str
    source_kind: _sources_pb2.SourceKind
    locator: str
    rationale: str
    approx_bytes: int
    def __init__(self, id: _Optional[str] = ..., owner: _Optional[str] = ..., name: _Optional[str] = ..., source_kind: _Optional[_Union[_sources_pb2.SourceKind, str]] = ..., locator: _Optional[str] = ..., rationale: _Optional[str] = ..., approx_bytes: _Optional[int] = ...) -> None: ...

class DestinationSuggestion(_message.Message):
    __slots__ = ("id", "label", "backend_kind", "location", "drive_class", "free_bytes", "total_bytes", "removable", "separate_root_ok", "rationale")
    ID_FIELD_NUMBER: _ClassVar[int]
    LABEL_FIELD_NUMBER: _ClassVar[int]
    BACKEND_KIND_FIELD_NUMBER: _ClassVar[int]
    LOCATION_FIELD_NUMBER: _ClassVar[int]
    DRIVE_CLASS_FIELD_NUMBER: _ClassVar[int]
    FREE_BYTES_FIELD_NUMBER: _ClassVar[int]
    TOTAL_BYTES_FIELD_NUMBER: _ClassVar[int]
    REMOVABLE_FIELD_NUMBER: _ClassVar[int]
    SEPARATE_ROOT_OK_FIELD_NUMBER: _ClassVar[int]
    RATIONALE_FIELD_NUMBER: _ClassVar[int]
    id: str
    label: str
    backend_kind: _destinations_pb2.BackendKind
    location: str
    drive_class: DriveClass
    free_bytes: int
    total_bytes: int
    removable: bool
    separate_root_ok: bool
    rationale: str
    def __init__(self, id: _Optional[str] = ..., label: _Optional[str] = ..., backend_kind: _Optional[_Union[_destinations_pb2.BackendKind, str]] = ..., location: _Optional[str] = ..., drive_class: _Optional[_Union[DriveClass, str]] = ..., free_bytes: _Optional[int] = ..., total_bytes: _Optional[int] = ..., removable: _Optional[bool] = ..., separate_root_ok: _Optional[bool] = ..., rationale: _Optional[str] = ...) -> None: ...

class ListTargetSuggestionsRequest(_message.Message):
    __slots__ = ()
    def __init__(self) -> None: ...

class ListTargetSuggestionsResponse(_message.Message):
    __slots__ = ("suggestions",)
    SUGGESTIONS_FIELD_NUMBER: _ClassVar[int]
    suggestions: _containers.RepeatedCompositeFieldContainer[TargetSuggestion]
    def __init__(self, suggestions: _Optional[_Iterable[_Union[TargetSuggestion, _Mapping]]] = ...) -> None: ...

class ListDestinationSuggestionsRequest(_message.Message):
    __slots__ = ()
    def __init__(self) -> None: ...

class ListDestinationSuggestionsResponse(_message.Message):
    __slots__ = ("suggestions",)
    SUGGESTIONS_FIELD_NUMBER: _ClassVar[int]
    suggestions: _containers.RepeatedCompositeFieldContainer[DestinationSuggestion]
    def __init__(self, suggestions: _Optional[_Iterable[_Union[DestinationSuggestion, _Mapping]]] = ...) -> None: ...

class DismissSuggestionRequest(_message.Message):
    __slots__ = ("id",)
    ID_FIELD_NUMBER: _ClassVar[int]
    id: str
    def __init__(self, id: _Optional[str] = ...) -> None: ...

class DismissSuggestionResponse(_message.Message):
    __slots__ = ("dismissed",)
    DISMISSED_FIELD_NUMBER: _ClassVar[int]
    dismissed: bool
    def __init__(self, dismissed: _Optional[bool] = ...) -> None: ...
