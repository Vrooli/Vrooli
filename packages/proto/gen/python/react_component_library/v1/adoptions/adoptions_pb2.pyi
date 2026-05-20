import datetime

from google.protobuf import timestamp_pb2 as _timestamp_pb2
from google.protobuf.internal import containers as _containers
from google.protobuf.internal import enum_type_wrapper as _enum_type_wrapper
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Iterable as _Iterable, Mapping as _Mapping
from typing import ClassVar as _ClassVar, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class LibraryVersionStatus(int, metaclass=_enum_type_wrapper.EnumTypeWrapper):
    __slots__ = ()
    LIBRARY_VERSION_STATUS_UNSPECIFIED: _ClassVar[LibraryVersionStatus]
    LIBRARY_VERSION_STATUS_CURRENT: _ClassVar[LibraryVersionStatus]
    LIBRARY_VERSION_STATUS_BEHIND: _ClassVar[LibraryVersionStatus]
    LIBRARY_VERSION_STATUS_DEPRECATED: _ClassVar[LibraryVersionStatus]
    LIBRARY_VERSION_STATUS_MISSING: _ClassVar[LibraryVersionStatus]
    LIBRARY_VERSION_STATUS_UNKNOWN: _ClassVar[LibraryVersionStatus]

class LocalStatus(int, metaclass=_enum_type_wrapper.EnumTypeWrapper):
    __slots__ = ()
    LOCAL_STATUS_UNSPECIFIED: _ClassVar[LocalStatus]
    LOCAL_STATUS_CLEAN: _ClassVar[LocalStatus]
    LOCAL_STATUS_MODIFIED: _ClassVar[LocalStatus]
    LOCAL_STATUS_MISSING: _ClassVar[LocalStatus]
    LOCAL_STATUS_UNKNOWN: _ClassVar[LocalStatus]

class ResolveSource(int, metaclass=_enum_type_wrapper.EnumTypeWrapper):
    __slots__ = ()
    RESOLVE_SOURCE_UNSPECIFIED: _ClassVar[ResolveSource]
    RESOLVE_SOURCE_EXPLICIT: _ClassVar[ResolveSource]
    RESOLVE_SOURCE_TEMPLATE_MANIFEST: _ClassVar[ResolveSource]
    RESOLVE_SOURCE_HEURISTIC: _ClassVar[ResolveSource]
    RESOLVE_SOURCE_FALLBACK: _ClassVar[ResolveSource]
LIBRARY_VERSION_STATUS_UNSPECIFIED: LibraryVersionStatus
LIBRARY_VERSION_STATUS_CURRENT: LibraryVersionStatus
LIBRARY_VERSION_STATUS_BEHIND: LibraryVersionStatus
LIBRARY_VERSION_STATUS_DEPRECATED: LibraryVersionStatus
LIBRARY_VERSION_STATUS_MISSING: LibraryVersionStatus
LIBRARY_VERSION_STATUS_UNKNOWN: LibraryVersionStatus
LOCAL_STATUS_UNSPECIFIED: LocalStatus
LOCAL_STATUS_CLEAN: LocalStatus
LOCAL_STATUS_MODIFIED: LocalStatus
LOCAL_STATUS_MISSING: LocalStatus
LOCAL_STATUS_UNKNOWN: LocalStatus
RESOLVE_SOURCE_UNSPECIFIED: ResolveSource
RESOLVE_SOURCE_EXPLICIT: ResolveSource
RESOLVE_SOURCE_TEMPLATE_MANIFEST: ResolveSource
RESOLVE_SOURCE_HEURISTIC: ResolveSource
RESOLVE_SOURCE_FALLBACK: ResolveSource

class Adoption(_message.Message):
    __slots__ = ("id", "component_id", "library_id", "scenario", "adopted_path", "adopted_version", "library_version_status", "local_status", "status_detail", "created_at", "refreshed_at", "source_sha256", "applied_at")
    ID_FIELD_NUMBER: _ClassVar[int]
    COMPONENT_ID_FIELD_NUMBER: _ClassVar[int]
    LIBRARY_ID_FIELD_NUMBER: _ClassVar[int]
    SCENARIO_FIELD_NUMBER: _ClassVar[int]
    ADOPTED_PATH_FIELD_NUMBER: _ClassVar[int]
    ADOPTED_VERSION_FIELD_NUMBER: _ClassVar[int]
    LIBRARY_VERSION_STATUS_FIELD_NUMBER: _ClassVar[int]
    LOCAL_STATUS_FIELD_NUMBER: _ClassVar[int]
    STATUS_DETAIL_FIELD_NUMBER: _ClassVar[int]
    CREATED_AT_FIELD_NUMBER: _ClassVar[int]
    REFRESHED_AT_FIELD_NUMBER: _ClassVar[int]
    SOURCE_SHA256_FIELD_NUMBER: _ClassVar[int]
    APPLIED_AT_FIELD_NUMBER: _ClassVar[int]
    id: str
    component_id: str
    library_id: str
    scenario: str
    adopted_path: str
    adopted_version: str
    library_version_status: LibraryVersionStatus
    local_status: LocalStatus
    status_detail: str
    created_at: _timestamp_pb2.Timestamp
    refreshed_at: _timestamp_pb2.Timestamp
    source_sha256: str
    applied_at: str
    def __init__(self, id: _Optional[str] = ..., component_id: _Optional[str] = ..., library_id: _Optional[str] = ..., scenario: _Optional[str] = ..., adopted_path: _Optional[str] = ..., adopted_version: _Optional[str] = ..., library_version_status: _Optional[_Union[LibraryVersionStatus, str]] = ..., local_status: _Optional[_Union[LocalStatus, str]] = ..., status_detail: _Optional[str] = ..., created_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., refreshed_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., source_sha256: _Optional[str] = ..., applied_at: _Optional[str] = ...) -> None: ...

class ListAdoptionsRequest(_message.Message):
    __slots__ = ("component_id", "scenario", "limit")
    COMPONENT_ID_FIELD_NUMBER: _ClassVar[int]
    SCENARIO_FIELD_NUMBER: _ClassVar[int]
    LIMIT_FIELD_NUMBER: _ClassVar[int]
    component_id: str
    scenario: str
    limit: int
    def __init__(self, component_id: _Optional[str] = ..., scenario: _Optional[str] = ..., limit: _Optional[int] = ...) -> None: ...

class ListAdoptionsResponse(_message.Message):
    __slots__ = ("adoptions",)
    ADOPTIONS_FIELD_NUMBER: _ClassVar[int]
    adoptions: _containers.RepeatedCompositeFieldContainer[Adoption]
    def __init__(self, adoptions: _Optional[_Iterable[_Union[Adoption, _Mapping]]] = ...) -> None: ...

class ApplyAdoptionRequest(_message.Message):
    __slots__ = ("component_id", "scenario", "adopted_path", "version", "confirm_overwrite")
    COMPONENT_ID_FIELD_NUMBER: _ClassVar[int]
    SCENARIO_FIELD_NUMBER: _ClassVar[int]
    ADOPTED_PATH_FIELD_NUMBER: _ClassVar[int]
    VERSION_FIELD_NUMBER: _ClassVar[int]
    CONFIRM_OVERWRITE_FIELD_NUMBER: _ClassVar[int]
    component_id: str
    scenario: str
    adopted_path: str
    version: str
    confirm_overwrite: bool
    def __init__(self, component_id: _Optional[str] = ..., scenario: _Optional[str] = ..., adopted_path: _Optional[str] = ..., version: _Optional[str] = ..., confirm_overwrite: _Optional[bool] = ...) -> None: ...

class ApplyAdoptionResponse(_message.Message):
    __slots__ = ("adoption", "written_path")
    ADOPTION_FIELD_NUMBER: _ClassVar[int]
    WRITTEN_PATH_FIELD_NUMBER: _ClassVar[int]
    adoption: Adoption
    written_path: str
    def __init__(self, adoption: _Optional[_Union[Adoption, _Mapping]] = ..., written_path: _Optional[str] = ...) -> None: ...

class ReapplyAdoptionRequest(_message.Message):
    __slots__ = ("id", "version", "confirm_local_overwrite")
    ID_FIELD_NUMBER: _ClassVar[int]
    VERSION_FIELD_NUMBER: _ClassVar[int]
    CONFIRM_LOCAL_OVERWRITE_FIELD_NUMBER: _ClassVar[int]
    id: str
    version: str
    confirm_local_overwrite: bool
    def __init__(self, id: _Optional[str] = ..., version: _Optional[str] = ..., confirm_local_overwrite: _Optional[bool] = ...) -> None: ...

class ReapplyAdoptionResponse(_message.Message):
    __slots__ = ("adoption", "written_path")
    ADOPTION_FIELD_NUMBER: _ClassVar[int]
    WRITTEN_PATH_FIELD_NUMBER: _ClassVar[int]
    adoption: Adoption
    written_path: str
    def __init__(self, adoption: _Optional[_Union[Adoption, _Mapping]] = ..., written_path: _Optional[str] = ...) -> None: ...

class DeleteAdoptionRequest(_message.Message):
    __slots__ = ("id",)
    ID_FIELD_NUMBER: _ClassVar[int]
    id: str
    def __init__(self, id: _Optional[str] = ...) -> None: ...

class DeleteAdoptionResponse(_message.Message):
    __slots__ = ()
    def __init__(self) -> None: ...

class RefreshAdoptionsRequest(_message.Message):
    __slots__ = ("component_id",)
    COMPONENT_ID_FIELD_NUMBER: _ClassVar[int]
    component_id: str
    def __init__(self, component_id: _Optional[str] = ...) -> None: ...

class ResolveAdoptionPathRequest(_message.Message):
    __slots__ = ("component_id", "scenario", "override_path", "feature")
    COMPONENT_ID_FIELD_NUMBER: _ClassVar[int]
    SCENARIO_FIELD_NUMBER: _ClassVar[int]
    OVERRIDE_PATH_FIELD_NUMBER: _ClassVar[int]
    FEATURE_FIELD_NUMBER: _ClassVar[int]
    component_id: str
    scenario: str
    override_path: str
    feature: str
    def __init__(self, component_id: _Optional[str] = ..., scenario: _Optional[str] = ..., override_path: _Optional[str] = ..., feature: _Optional[str] = ...) -> None: ...

class ResolveAdoptionPathResponse(_message.Message):
    __slots__ = ("path", "source", "slot", "warnings")
    PATH_FIELD_NUMBER: _ClassVar[int]
    SOURCE_FIELD_NUMBER: _ClassVar[int]
    SLOT_FIELD_NUMBER: _ClassVar[int]
    WARNINGS_FIELD_NUMBER: _ClassVar[int]
    path: str
    source: ResolveSource
    slot: str
    warnings: _containers.RepeatedScalarFieldContainer[str]
    def __init__(self, path: _Optional[str] = ..., source: _Optional[_Union[ResolveSource, str]] = ..., slot: _Optional[str] = ..., warnings: _Optional[_Iterable[str]] = ...) -> None: ...

class RefreshAdoptionsResponse(_message.Message):
    __slots__ = ("adoptions", "library_current", "library_behind", "library_deprecated", "library_missing", "library_unknown", "local_clean", "local_modified", "local_missing", "local_unknown")
    ADOPTIONS_FIELD_NUMBER: _ClassVar[int]
    LIBRARY_CURRENT_FIELD_NUMBER: _ClassVar[int]
    LIBRARY_BEHIND_FIELD_NUMBER: _ClassVar[int]
    LIBRARY_DEPRECATED_FIELD_NUMBER: _ClassVar[int]
    LIBRARY_MISSING_FIELD_NUMBER: _ClassVar[int]
    LIBRARY_UNKNOWN_FIELD_NUMBER: _ClassVar[int]
    LOCAL_CLEAN_FIELD_NUMBER: _ClassVar[int]
    LOCAL_MODIFIED_FIELD_NUMBER: _ClassVar[int]
    LOCAL_MISSING_FIELD_NUMBER: _ClassVar[int]
    LOCAL_UNKNOWN_FIELD_NUMBER: _ClassVar[int]
    adoptions: _containers.RepeatedCompositeFieldContainer[Adoption]
    library_current: int
    library_behind: int
    library_deprecated: int
    library_missing: int
    library_unknown: int
    local_clean: int
    local_modified: int
    local_missing: int
    local_unknown: int
    def __init__(self, adoptions: _Optional[_Iterable[_Union[Adoption, _Mapping]]] = ..., library_current: _Optional[int] = ..., library_behind: _Optional[int] = ..., library_deprecated: _Optional[int] = ..., library_missing: _Optional[int] = ..., library_unknown: _Optional[int] = ..., local_clean: _Optional[int] = ..., local_modified: _Optional[int] = ..., local_missing: _Optional[int] = ..., local_unknown: _Optional[int] = ...) -> None: ...
