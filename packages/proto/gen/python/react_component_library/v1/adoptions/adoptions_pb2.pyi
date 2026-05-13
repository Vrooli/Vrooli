import datetime

from google.protobuf import timestamp_pb2 as _timestamp_pb2
from google.protobuf.internal import containers as _containers
from google.protobuf.internal import enum_type_wrapper as _enum_type_wrapper
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Iterable as _Iterable, Mapping as _Mapping
from typing import ClassVar as _ClassVar, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class AdoptionStatus(int, metaclass=_enum_type_wrapper.EnumTypeWrapper):
    __slots__ = ()
    ADOPTION_STATUS_UNSPECIFIED: _ClassVar[AdoptionStatus]
    ADOPTION_STATUS_CURRENT: _ClassVar[AdoptionStatus]
    ADOPTION_STATUS_BEHIND: _ClassVar[AdoptionStatus]
    ADOPTION_STATUS_MODIFIED: _ClassVar[AdoptionStatus]
    ADOPTION_STATUS_UNKNOWN: _ClassVar[AdoptionStatus]
ADOPTION_STATUS_UNSPECIFIED: AdoptionStatus
ADOPTION_STATUS_CURRENT: AdoptionStatus
ADOPTION_STATUS_BEHIND: AdoptionStatus
ADOPTION_STATUS_MODIFIED: AdoptionStatus
ADOPTION_STATUS_UNKNOWN: AdoptionStatus

class Adoption(_message.Message):
    __slots__ = ("id", "component_id", "library_id", "scenario", "adopted_path", "adopted_version", "status", "status_detail", "created_at", "refreshed_at")
    ID_FIELD_NUMBER: _ClassVar[int]
    COMPONENT_ID_FIELD_NUMBER: _ClassVar[int]
    LIBRARY_ID_FIELD_NUMBER: _ClassVar[int]
    SCENARIO_FIELD_NUMBER: _ClassVar[int]
    ADOPTED_PATH_FIELD_NUMBER: _ClassVar[int]
    ADOPTED_VERSION_FIELD_NUMBER: _ClassVar[int]
    STATUS_FIELD_NUMBER: _ClassVar[int]
    STATUS_DETAIL_FIELD_NUMBER: _ClassVar[int]
    CREATED_AT_FIELD_NUMBER: _ClassVar[int]
    REFRESHED_AT_FIELD_NUMBER: _ClassVar[int]
    id: str
    component_id: str
    library_id: str
    scenario: str
    adopted_path: str
    adopted_version: str
    status: AdoptionStatus
    status_detail: str
    created_at: _timestamp_pb2.Timestamp
    refreshed_at: _timestamp_pb2.Timestamp
    def __init__(self, id: _Optional[str] = ..., component_id: _Optional[str] = ..., library_id: _Optional[str] = ..., scenario: _Optional[str] = ..., adopted_path: _Optional[str] = ..., adopted_version: _Optional[str] = ..., status: _Optional[_Union[AdoptionStatus, str]] = ..., status_detail: _Optional[str] = ..., created_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., refreshed_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ...) -> None: ...

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

class CreateAdoptionRequest(_message.Message):
    __slots__ = ("component_id", "scenario", "adopted_path", "adopted_version")
    COMPONENT_ID_FIELD_NUMBER: _ClassVar[int]
    SCENARIO_FIELD_NUMBER: _ClassVar[int]
    ADOPTED_PATH_FIELD_NUMBER: _ClassVar[int]
    ADOPTED_VERSION_FIELD_NUMBER: _ClassVar[int]
    component_id: str
    scenario: str
    adopted_path: str
    adopted_version: str
    def __init__(self, component_id: _Optional[str] = ..., scenario: _Optional[str] = ..., adopted_path: _Optional[str] = ..., adopted_version: _Optional[str] = ...) -> None: ...

class CreateAdoptionResponse(_message.Message):
    __slots__ = ("adoption",)
    ADOPTION_FIELD_NUMBER: _ClassVar[int]
    adoption: Adoption
    def __init__(self, adoption: _Optional[_Union[Adoption, _Mapping]] = ...) -> None: ...

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

class RefreshAdoptionsResponse(_message.Message):
    __slots__ = ("adoptions", "current", "behind", "modified", "unknown")
    ADOPTIONS_FIELD_NUMBER: _ClassVar[int]
    CURRENT_FIELD_NUMBER: _ClassVar[int]
    BEHIND_FIELD_NUMBER: _ClassVar[int]
    MODIFIED_FIELD_NUMBER: _ClassVar[int]
    UNKNOWN_FIELD_NUMBER: _ClassVar[int]
    adoptions: _containers.RepeatedCompositeFieldContainer[Adoption]
    current: int
    behind: int
    modified: int
    unknown: int
    def __init__(self, adoptions: _Optional[_Iterable[_Union[Adoption, _Mapping]]] = ..., current: _Optional[int] = ..., behind: _Optional[int] = ..., modified: _Optional[int] = ..., unknown: _Optional[int] = ...) -> None: ...
