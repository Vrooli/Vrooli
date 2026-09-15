import datetime

from buf.validate import validate_pb2 as _validate_pb2
from common.v1 import types_pb2 as _types_pb2
from google.protobuf import timestamp_pb2 as _timestamp_pb2
from google.protobuf.internal import containers as _containers
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Iterable as _Iterable, Mapping as _Mapping
from typing import ClassVar as _ClassVar, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class Profile(_message.Message):
    __slots__ = ("id", "name", "scenario", "tiers", "swaps", "secrets", "settings", "version", "created_at", "updated_at", "created_by", "updated_by")
    ID_FIELD_NUMBER: _ClassVar[int]
    NAME_FIELD_NUMBER: _ClassVar[int]
    SCENARIO_FIELD_NUMBER: _ClassVar[int]
    TIERS_FIELD_NUMBER: _ClassVar[int]
    SWAPS_FIELD_NUMBER: _ClassVar[int]
    SECRETS_FIELD_NUMBER: _ClassVar[int]
    SETTINGS_FIELD_NUMBER: _ClassVar[int]
    VERSION_FIELD_NUMBER: _ClassVar[int]
    CREATED_AT_FIELD_NUMBER: _ClassVar[int]
    UPDATED_AT_FIELD_NUMBER: _ClassVar[int]
    CREATED_BY_FIELD_NUMBER: _ClassVar[int]
    UPDATED_BY_FIELD_NUMBER: _ClassVar[int]
    id: str
    name: str
    scenario: str
    tiers: _containers.RepeatedScalarFieldContainer[int]
    swaps: _types_pb2.JsonObject
    secrets: _types_pb2.JsonObject
    settings: _types_pb2.JsonObject
    version: int
    created_at: _timestamp_pb2.Timestamp
    updated_at: _timestamp_pb2.Timestamp
    created_by: str
    updated_by: str
    def __init__(self, id: _Optional[str] = ..., name: _Optional[str] = ..., scenario: _Optional[str] = ..., tiers: _Optional[_Iterable[int]] = ..., swaps: _Optional[_Union[_types_pb2.JsonObject, _Mapping]] = ..., secrets: _Optional[_Union[_types_pb2.JsonObject, _Mapping]] = ..., settings: _Optional[_Union[_types_pb2.JsonObject, _Mapping]] = ..., version: _Optional[int] = ..., created_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., updated_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., created_by: _Optional[str] = ..., updated_by: _Optional[str] = ...) -> None: ...

class ProfileVersion(_message.Message):
    __slots__ = ("profile_id", "version", "name", "scenario", "tiers", "swaps", "secrets", "settings", "created_at", "created_by", "change_description")
    PROFILE_ID_FIELD_NUMBER: _ClassVar[int]
    VERSION_FIELD_NUMBER: _ClassVar[int]
    NAME_FIELD_NUMBER: _ClassVar[int]
    SCENARIO_FIELD_NUMBER: _ClassVar[int]
    TIERS_FIELD_NUMBER: _ClassVar[int]
    SWAPS_FIELD_NUMBER: _ClassVar[int]
    SECRETS_FIELD_NUMBER: _ClassVar[int]
    SETTINGS_FIELD_NUMBER: _ClassVar[int]
    CREATED_AT_FIELD_NUMBER: _ClassVar[int]
    CREATED_BY_FIELD_NUMBER: _ClassVar[int]
    CHANGE_DESCRIPTION_FIELD_NUMBER: _ClassVar[int]
    profile_id: str
    version: int
    name: str
    scenario: str
    tiers: _containers.RepeatedScalarFieldContainer[int]
    swaps: _types_pb2.JsonObject
    secrets: _types_pb2.JsonObject
    settings: _types_pb2.JsonObject
    created_at: _timestamp_pb2.Timestamp
    created_by: str
    change_description: str
    def __init__(self, profile_id: _Optional[str] = ..., version: _Optional[int] = ..., name: _Optional[str] = ..., scenario: _Optional[str] = ..., tiers: _Optional[_Iterable[int]] = ..., swaps: _Optional[_Union[_types_pb2.JsonObject, _Mapping]] = ..., secrets: _Optional[_Union[_types_pb2.JsonObject, _Mapping]] = ..., settings: _Optional[_Union[_types_pb2.JsonObject, _Mapping]] = ..., created_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., created_by: _Optional[str] = ..., change_description: _Optional[str] = ...) -> None: ...

class ListProfilesRequest(_message.Message):
    __slots__ = ("page_size", "page_token")
    PAGE_SIZE_FIELD_NUMBER: _ClassVar[int]
    PAGE_TOKEN_FIELD_NUMBER: _ClassVar[int]
    page_size: int
    page_token: str
    def __init__(self, page_size: _Optional[int] = ..., page_token: _Optional[str] = ...) -> None: ...

class ListProfilesResponse(_message.Message):
    __slots__ = ("profiles", "next_page_token")
    PROFILES_FIELD_NUMBER: _ClassVar[int]
    NEXT_PAGE_TOKEN_FIELD_NUMBER: _ClassVar[int]
    profiles: _containers.RepeatedCompositeFieldContainer[Profile]
    next_page_token: str
    def __init__(self, profiles: _Optional[_Iterable[_Union[Profile, _Mapping]]] = ..., next_page_token: _Optional[str] = ...) -> None: ...

class GetProfileRequest(_message.Message):
    __slots__ = ("profile_id",)
    PROFILE_ID_FIELD_NUMBER: _ClassVar[int]
    profile_id: str
    def __init__(self, profile_id: _Optional[str] = ...) -> None: ...

class GetProfileResponse(_message.Message):
    __slots__ = ("profile",)
    PROFILE_FIELD_NUMBER: _ClassVar[int]
    profile: Profile
    def __init__(self, profile: _Optional[_Union[Profile, _Mapping]] = ...) -> None: ...

class CreateProfileRequest(_message.Message):
    __slots__ = ("name", "scenario", "tiers", "swaps", "secrets", "settings")
    NAME_FIELD_NUMBER: _ClassVar[int]
    SCENARIO_FIELD_NUMBER: _ClassVar[int]
    TIERS_FIELD_NUMBER: _ClassVar[int]
    SWAPS_FIELD_NUMBER: _ClassVar[int]
    SECRETS_FIELD_NUMBER: _ClassVar[int]
    SETTINGS_FIELD_NUMBER: _ClassVar[int]
    name: str
    scenario: str
    tiers: _containers.RepeatedScalarFieldContainer[int]
    swaps: _types_pb2.JsonObject
    secrets: _types_pb2.JsonObject
    settings: _types_pb2.JsonObject
    def __init__(self, name: _Optional[str] = ..., scenario: _Optional[str] = ..., tiers: _Optional[_Iterable[int]] = ..., swaps: _Optional[_Union[_types_pb2.JsonObject, _Mapping]] = ..., secrets: _Optional[_Union[_types_pb2.JsonObject, _Mapping]] = ..., settings: _Optional[_Union[_types_pb2.JsonObject, _Mapping]] = ...) -> None: ...

class CreateProfileResponse(_message.Message):
    __slots__ = ("profile",)
    PROFILE_FIELD_NUMBER: _ClassVar[int]
    profile: Profile
    def __init__(self, profile: _Optional[_Union[Profile, _Mapping]] = ...) -> None: ...

class UpdateProfileRequest(_message.Message):
    __slots__ = ("profile_id", "name", "scenario", "tiers", "swaps", "secrets", "settings")
    PROFILE_ID_FIELD_NUMBER: _ClassVar[int]
    NAME_FIELD_NUMBER: _ClassVar[int]
    SCENARIO_FIELD_NUMBER: _ClassVar[int]
    TIERS_FIELD_NUMBER: _ClassVar[int]
    SWAPS_FIELD_NUMBER: _ClassVar[int]
    SECRETS_FIELD_NUMBER: _ClassVar[int]
    SETTINGS_FIELD_NUMBER: _ClassVar[int]
    profile_id: str
    name: str
    scenario: str
    tiers: _containers.RepeatedScalarFieldContainer[int]
    swaps: _types_pb2.JsonObject
    secrets: _types_pb2.JsonObject
    settings: _types_pb2.JsonObject
    def __init__(self, profile_id: _Optional[str] = ..., name: _Optional[str] = ..., scenario: _Optional[str] = ..., tiers: _Optional[_Iterable[int]] = ..., swaps: _Optional[_Union[_types_pb2.JsonObject, _Mapping]] = ..., secrets: _Optional[_Union[_types_pb2.JsonObject, _Mapping]] = ..., settings: _Optional[_Union[_types_pb2.JsonObject, _Mapping]] = ...) -> None: ...

class UpdateProfileResponse(_message.Message):
    __slots__ = ("profile",)
    PROFILE_FIELD_NUMBER: _ClassVar[int]
    profile: Profile
    def __init__(self, profile: _Optional[_Union[Profile, _Mapping]] = ...) -> None: ...

class DeleteProfileRequest(_message.Message):
    __slots__ = ("profile_id",)
    PROFILE_ID_FIELD_NUMBER: _ClassVar[int]
    profile_id: str
    def __init__(self, profile_id: _Optional[str] = ...) -> None: ...

class DeleteProfileResponse(_message.Message):
    __slots__ = ("profile_id",)
    PROFILE_ID_FIELD_NUMBER: _ClassVar[int]
    profile_id: str
    def __init__(self, profile_id: _Optional[str] = ...) -> None: ...

class ListProfileVersionsRequest(_message.Message):
    __slots__ = ("profile_id", "page_size", "page_token")
    PROFILE_ID_FIELD_NUMBER: _ClassVar[int]
    PAGE_SIZE_FIELD_NUMBER: _ClassVar[int]
    PAGE_TOKEN_FIELD_NUMBER: _ClassVar[int]
    profile_id: str
    page_size: int
    page_token: str
    def __init__(self, profile_id: _Optional[str] = ..., page_size: _Optional[int] = ..., page_token: _Optional[str] = ...) -> None: ...

class ListProfileVersionsResponse(_message.Message):
    __slots__ = ("versions", "next_page_token")
    VERSIONS_FIELD_NUMBER: _ClassVar[int]
    NEXT_PAGE_TOKEN_FIELD_NUMBER: _ClassVar[int]
    versions: _containers.RepeatedCompositeFieldContainer[ProfileVersion]
    next_page_token: str
    def __init__(self, versions: _Optional[_Iterable[_Union[ProfileVersion, _Mapping]]] = ..., next_page_token: _Optional[str] = ...) -> None: ...
