from google.protobuf.internal import containers as _containers
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Iterable as _Iterable, Mapping as _Mapping
from typing import ClassVar as _ClassVar, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class Shortcut(_message.Message):
    __slots__ = ("label", "command", "description", "agent_id")
    LABEL_FIELD_NUMBER: _ClassVar[int]
    COMMAND_FIELD_NUMBER: _ClassVar[int]
    DESCRIPTION_FIELD_NUMBER: _ClassVar[int]
    AGENT_ID_FIELD_NUMBER: _ClassVar[int]
    label: str
    command: str
    description: str
    agent_id: str
    def __init__(self, label: _Optional[str] = ..., command: _Optional[str] = ..., description: _Optional[str] = ..., agent_id: _Optional[str] = ...) -> None: ...

class Profile(_message.Message):
    __slots__ = ("id", "scope", "name", "shortcuts", "created_at", "updated_at")
    ID_FIELD_NUMBER: _ClassVar[int]
    SCOPE_FIELD_NUMBER: _ClassVar[int]
    NAME_FIELD_NUMBER: _ClassVar[int]
    SHORTCUTS_FIELD_NUMBER: _ClassVar[int]
    CREATED_AT_FIELD_NUMBER: _ClassVar[int]
    UPDATED_AT_FIELD_NUMBER: _ClassVar[int]
    id: str
    scope: str
    name: str
    shortcuts: _containers.RepeatedCompositeFieldContainer[Shortcut]
    created_at: str
    updated_at: str
    def __init__(self, id: _Optional[str] = ..., scope: _Optional[str] = ..., name: _Optional[str] = ..., shortcuts: _Optional[_Iterable[_Union[Shortcut, _Mapping]]] = ..., created_at: _Optional[str] = ..., updated_at: _Optional[str] = ...) -> None: ...

class GetEffectiveRequest(_message.Message):
    __slots__ = ()
    def __init__(self) -> None: ...

class GetEffectiveResponse(_message.Message):
    __slots__ = ("shortcuts", "profile_id", "scope", "profile_name")
    SHORTCUTS_FIELD_NUMBER: _ClassVar[int]
    PROFILE_ID_FIELD_NUMBER: _ClassVar[int]
    SCOPE_FIELD_NUMBER: _ClassVar[int]
    PROFILE_NAME_FIELD_NUMBER: _ClassVar[int]
    shortcuts: _containers.RepeatedCompositeFieldContainer[Shortcut]
    profile_id: str
    scope: str
    profile_name: str
    def __init__(self, shortcuts: _Optional[_Iterable[_Union[Shortcut, _Mapping]]] = ..., profile_id: _Optional[str] = ..., scope: _Optional[str] = ..., profile_name: _Optional[str] = ...) -> None: ...

class ListProfilesRequest(_message.Message):
    __slots__ = ()
    def __init__(self) -> None: ...

class ListProfilesResponse(_message.Message):
    __slots__ = ("profiles",)
    PROFILES_FIELD_NUMBER: _ClassVar[int]
    profiles: _containers.RepeatedCompositeFieldContainer[Profile]
    def __init__(self, profiles: _Optional[_Iterable[_Union[Profile, _Mapping]]] = ...) -> None: ...

class UpsertProfileRequest(_message.Message):
    __slots__ = ("id", "scope", "name", "shortcuts")
    ID_FIELD_NUMBER: _ClassVar[int]
    SCOPE_FIELD_NUMBER: _ClassVar[int]
    NAME_FIELD_NUMBER: _ClassVar[int]
    SHORTCUTS_FIELD_NUMBER: _ClassVar[int]
    id: str
    scope: str
    name: str
    shortcuts: _containers.RepeatedCompositeFieldContainer[Shortcut]
    def __init__(self, id: _Optional[str] = ..., scope: _Optional[str] = ..., name: _Optional[str] = ..., shortcuts: _Optional[_Iterable[_Union[Shortcut, _Mapping]]] = ...) -> None: ...

class UpsertProfileResponse(_message.Message):
    __slots__ = ("profile",)
    PROFILE_FIELD_NUMBER: _ClassVar[int]
    profile: Profile
    def __init__(self, profile: _Optional[_Union[Profile, _Mapping]] = ...) -> None: ...

class DeleteProfileRequest(_message.Message):
    __slots__ = ("id",)
    ID_FIELD_NUMBER: _ClassVar[int]
    id: str
    def __init__(self, id: _Optional[str] = ...) -> None: ...

class DeleteProfileResponse(_message.Message):
    __slots__ = ()
    def __init__(self) -> None: ...
