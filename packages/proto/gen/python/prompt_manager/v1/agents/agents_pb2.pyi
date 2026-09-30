from google.protobuf import field_mask_pb2 as _field_mask_pb2
from google.protobuf.internal import containers as _containers
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Iterable as _Iterable, Mapping as _Mapping
from typing import ClassVar as _ClassVar, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class Appearance(_message.Message):
    __slots__ = ("body", "head", "accent")
    BODY_FIELD_NUMBER: _ClassVar[int]
    HEAD_FIELD_NUMBER: _ClassVar[int]
    ACCENT_FIELD_NUMBER: _ClassVar[int]
    body: str
    head: str
    accent: str
    def __init__(self, body: _Optional[str] = ..., head: _Optional[str] = ..., accent: _Optional[str] = ...) -> None: ...

class Capability(_message.Message):
    __slots__ = ("capability_id", "verbs")
    CAPABILITY_ID_FIELD_NUMBER: _ClassVar[int]
    VERBS_FIELD_NUMBER: _ClassVar[int]
    capability_id: str
    verbs: _containers.RepeatedScalarFieldContainer[str]
    def __init__(self, capability_id: _Optional[str] = ..., verbs: _Optional[_Iterable[str]] = ...) -> None: ...

class Capabilities(_message.Message):
    __slots__ = ("provides", "requires")
    PROVIDES_FIELD_NUMBER: _ClassVar[int]
    REQUIRES_FIELD_NUMBER: _ClassVar[int]
    provides: _containers.RepeatedCompositeFieldContainer[Capability]
    requires: _containers.RepeatedCompositeFieldContainer[Capability]
    def __init__(self, provides: _Optional[_Iterable[_Union[Capability, _Mapping]]] = ..., requires: _Optional[_Iterable[_Union[Capability, _Mapping]]] = ...) -> None: ...

class Connector(_message.Message):
    __slots__ = ("type", "id", "enabled")
    TYPE_FIELD_NUMBER: _ClassVar[int]
    ID_FIELD_NUMBER: _ClassVar[int]
    ENABLED_FIELD_NUMBER: _ClassVar[int]
    type: str
    id: str
    enabled: bool
    def __init__(self, type: _Optional[str] = ..., id: _Optional[str] = ..., enabled: _Optional[bool] = ...) -> None: ...

class Heartbeat(_message.Message):
    __slots__ = ("interval_seconds", "timeout_seconds", "max_missed_beats")
    INTERVAL_SECONDS_FIELD_NUMBER: _ClassVar[int]
    TIMEOUT_SECONDS_FIELD_NUMBER: _ClassVar[int]
    MAX_MISSED_BEATS_FIELD_NUMBER: _ClassVar[int]
    interval_seconds: int
    timeout_seconds: int
    max_missed_beats: int
    def __init__(self, interval_seconds: _Optional[int] = ..., timeout_seconds: _Optional[int] = ..., max_missed_beats: _Optional[int] = ...) -> None: ...

class Agent(_message.Message):
    __slots__ = ("id", "display_name", "description", "status", "appearance", "capabilities", "connectors", "default_profile_ref", "heartbeat", "tags", "file_order", "agent_dir", "created_at", "updated_at")
    ID_FIELD_NUMBER: _ClassVar[int]
    DISPLAY_NAME_FIELD_NUMBER: _ClassVar[int]
    DESCRIPTION_FIELD_NUMBER: _ClassVar[int]
    STATUS_FIELD_NUMBER: _ClassVar[int]
    APPEARANCE_FIELD_NUMBER: _ClassVar[int]
    CAPABILITIES_FIELD_NUMBER: _ClassVar[int]
    CONNECTORS_FIELD_NUMBER: _ClassVar[int]
    DEFAULT_PROFILE_REF_FIELD_NUMBER: _ClassVar[int]
    HEARTBEAT_FIELD_NUMBER: _ClassVar[int]
    TAGS_FIELD_NUMBER: _ClassVar[int]
    FILE_ORDER_FIELD_NUMBER: _ClassVar[int]
    AGENT_DIR_FIELD_NUMBER: _ClassVar[int]
    CREATED_AT_FIELD_NUMBER: _ClassVar[int]
    UPDATED_AT_FIELD_NUMBER: _ClassVar[int]
    id: str
    display_name: str
    description: str
    status: str
    appearance: Appearance
    capabilities: Capabilities
    connectors: _containers.RepeatedCompositeFieldContainer[Connector]
    default_profile_ref: str
    heartbeat: Heartbeat
    tags: _containers.RepeatedScalarFieldContainer[str]
    file_order: _containers.RepeatedScalarFieldContainer[str]
    agent_dir: str
    created_at: str
    updated_at: str
    def __init__(self, id: _Optional[str] = ..., display_name: _Optional[str] = ..., description: _Optional[str] = ..., status: _Optional[str] = ..., appearance: _Optional[_Union[Appearance, _Mapping]] = ..., capabilities: _Optional[_Union[Capabilities, _Mapping]] = ..., connectors: _Optional[_Iterable[_Union[Connector, _Mapping]]] = ..., default_profile_ref: _Optional[str] = ..., heartbeat: _Optional[_Union[Heartbeat, _Mapping]] = ..., tags: _Optional[_Iterable[str]] = ..., file_order: _Optional[_Iterable[str]] = ..., agent_dir: _Optional[str] = ..., created_at: _Optional[str] = ..., updated_at: _Optional[str] = ...) -> None: ...

class AgentInput(_message.Message):
    __slots__ = ("id", "display_name", "description", "status", "appearance", "capabilities", "connectors", "default_profile_ref", "heartbeat", "tags", "file_order")
    ID_FIELD_NUMBER: _ClassVar[int]
    DISPLAY_NAME_FIELD_NUMBER: _ClassVar[int]
    DESCRIPTION_FIELD_NUMBER: _ClassVar[int]
    STATUS_FIELD_NUMBER: _ClassVar[int]
    APPEARANCE_FIELD_NUMBER: _ClassVar[int]
    CAPABILITIES_FIELD_NUMBER: _ClassVar[int]
    CONNECTORS_FIELD_NUMBER: _ClassVar[int]
    DEFAULT_PROFILE_REF_FIELD_NUMBER: _ClassVar[int]
    HEARTBEAT_FIELD_NUMBER: _ClassVar[int]
    TAGS_FIELD_NUMBER: _ClassVar[int]
    FILE_ORDER_FIELD_NUMBER: _ClassVar[int]
    id: str
    display_name: str
    description: str
    status: str
    appearance: Appearance
    capabilities: Capabilities
    connectors: _containers.RepeatedCompositeFieldContainer[Connector]
    default_profile_ref: str
    heartbeat: Heartbeat
    tags: _containers.RepeatedScalarFieldContainer[str]
    file_order: _containers.RepeatedScalarFieldContainer[str]
    def __init__(self, id: _Optional[str] = ..., display_name: _Optional[str] = ..., description: _Optional[str] = ..., status: _Optional[str] = ..., appearance: _Optional[_Union[Appearance, _Mapping]] = ..., capabilities: _Optional[_Union[Capabilities, _Mapping]] = ..., connectors: _Optional[_Iterable[_Union[Connector, _Mapping]]] = ..., default_profile_ref: _Optional[str] = ..., heartbeat: _Optional[_Union[Heartbeat, _Mapping]] = ..., tags: _Optional[_Iterable[str]] = ..., file_order: _Optional[_Iterable[str]] = ...) -> None: ...

class ListAgentsRequest(_message.Message):
    __slots__ = ()
    def __init__(self) -> None: ...

class ListAgentsResponse(_message.Message):
    __slots__ = ("agents",)
    AGENTS_FIELD_NUMBER: _ClassVar[int]
    agents: _containers.RepeatedCompositeFieldContainer[Agent]
    def __init__(self, agents: _Optional[_Iterable[_Union[Agent, _Mapping]]] = ...) -> None: ...

class GetAgentRequest(_message.Message):
    __slots__ = ("id",)
    ID_FIELD_NUMBER: _ClassVar[int]
    id: str
    def __init__(self, id: _Optional[str] = ...) -> None: ...

class CreateAgentRequest(_message.Message):
    __slots__ = ("agent",)
    AGENT_FIELD_NUMBER: _ClassVar[int]
    agent: AgentInput
    def __init__(self, agent: _Optional[_Union[AgentInput, _Mapping]] = ...) -> None: ...

class UpdateAgentRequest(_message.Message):
    __slots__ = ("id", "agent", "update_mask")
    ID_FIELD_NUMBER: _ClassVar[int]
    AGENT_FIELD_NUMBER: _ClassVar[int]
    UPDATE_MASK_FIELD_NUMBER: _ClassVar[int]
    id: str
    agent: AgentInput
    update_mask: _field_mask_pb2.FieldMask
    def __init__(self, id: _Optional[str] = ..., agent: _Optional[_Union[AgentInput, _Mapping]] = ..., update_mask: _Optional[_Union[_field_mask_pb2.FieldMask, _Mapping]] = ...) -> None: ...

class DeleteAgentRequest(_message.Message):
    __slots__ = ("id",)
    ID_FIELD_NUMBER: _ClassVar[int]
    id: str
    def __init__(self, id: _Optional[str] = ...) -> None: ...

class DeleteAgentResponse(_message.Message):
    __slots__ = ()
    def __init__(self) -> None: ...

class TeamMembership(_message.Message):
    __slots__ = ("team_id", "team_display_name", "roles", "status")
    TEAM_ID_FIELD_NUMBER: _ClassVar[int]
    TEAM_DISPLAY_NAME_FIELD_NUMBER: _ClassVar[int]
    ROLES_FIELD_NUMBER: _ClassVar[int]
    STATUS_FIELD_NUMBER: _ClassVar[int]
    team_id: str
    team_display_name: str
    roles: _containers.RepeatedScalarFieldContainer[str]
    status: str
    def __init__(self, team_id: _Optional[str] = ..., team_display_name: _Optional[str] = ..., roles: _Optional[_Iterable[str]] = ..., status: _Optional[str] = ...) -> None: ...

class ListAgentTeamsRequest(_message.Message):
    __slots__ = ("id",)
    ID_FIELD_NUMBER: _ClassVar[int]
    id: str
    def __init__(self, id: _Optional[str] = ...) -> None: ...

class ListAgentTeamsResponse(_message.Message):
    __slots__ = ("agent_id", "memberships")
    AGENT_ID_FIELD_NUMBER: _ClassVar[int]
    MEMBERSHIPS_FIELD_NUMBER: _ClassVar[int]
    agent_id: str
    memberships: _containers.RepeatedCompositeFieldContainer[TeamMembership]
    def __init__(self, agent_id: _Optional[str] = ..., memberships: _Optional[_Iterable[_Union[TeamMembership, _Mapping]]] = ...) -> None: ...

class Soul(_message.Message):
    __slots__ = ("agent_id", "content")
    AGENT_ID_FIELD_NUMBER: _ClassVar[int]
    CONTENT_FIELD_NUMBER: _ClassVar[int]
    agent_id: str
    content: str
    def __init__(self, agent_id: _Optional[str] = ..., content: _Optional[str] = ...) -> None: ...

class GetSoulRequest(_message.Message):
    __slots__ = ("id",)
    ID_FIELD_NUMBER: _ClassVar[int]
    id: str
    def __init__(self, id: _Optional[str] = ...) -> None: ...

class SetSoulRequest(_message.Message):
    __slots__ = ("id", "content")
    ID_FIELD_NUMBER: _ClassVar[int]
    CONTENT_FIELD_NUMBER: _ClassVar[int]
    id: str
    content: str
    def __init__(self, id: _Optional[str] = ..., content: _Optional[str] = ...) -> None: ...

class ManageSoulRequest(_message.Message):
    __slots__ = ("id", "content")
    ID_FIELD_NUMBER: _ClassVar[int]
    CONTENT_FIELD_NUMBER: _ClassVar[int]
    id: str
    content: str
    def __init__(self, id: _Optional[str] = ..., content: _Optional[str] = ...) -> None: ...

class FileEntry(_message.Message):
    __slots__ = ("path", "is_dir", "size")
    PATH_FIELD_NUMBER: _ClassVar[int]
    IS_DIR_FIELD_NUMBER: _ClassVar[int]
    SIZE_FIELD_NUMBER: _ClassVar[int]
    path: str
    is_dir: bool
    size: int
    def __init__(self, path: _Optional[str] = ..., is_dir: _Optional[bool] = ..., size: _Optional[int] = ...) -> None: ...

class ListFilesRequest(_message.Message):
    __slots__ = ("id",)
    ID_FIELD_NUMBER: _ClassVar[int]
    id: str
    def __init__(self, id: _Optional[str] = ...) -> None: ...

class ListFilesResponse(_message.Message):
    __slots__ = ("agent_id", "files")
    AGENT_ID_FIELD_NUMBER: _ClassVar[int]
    FILES_FIELD_NUMBER: _ClassVar[int]
    agent_id: str
    files: _containers.RepeatedCompositeFieldContainer[FileEntry]
    def __init__(self, agent_id: _Optional[str] = ..., files: _Optional[_Iterable[_Union[FileEntry, _Mapping]]] = ...) -> None: ...

class FileContent(_message.Message):
    __slots__ = ("agent_id", "path", "content")
    AGENT_ID_FIELD_NUMBER: _ClassVar[int]
    PATH_FIELD_NUMBER: _ClassVar[int]
    CONTENT_FIELD_NUMBER: _ClassVar[int]
    agent_id: str
    path: str
    content: str
    def __init__(self, agent_id: _Optional[str] = ..., path: _Optional[str] = ..., content: _Optional[str] = ...) -> None: ...

class GetFileRequest(_message.Message):
    __slots__ = ("id", "path")
    ID_FIELD_NUMBER: _ClassVar[int]
    PATH_FIELD_NUMBER: _ClassVar[int]
    id: str
    path: str
    def __init__(self, id: _Optional[str] = ..., path: _Optional[str] = ...) -> None: ...

class SetFileRequest(_message.Message):
    __slots__ = ("id", "path", "content")
    ID_FIELD_NUMBER: _ClassVar[int]
    PATH_FIELD_NUMBER: _ClassVar[int]
    CONTENT_FIELD_NUMBER: _ClassVar[int]
    id: str
    path: str
    content: str
    def __init__(self, id: _Optional[str] = ..., path: _Optional[str] = ..., content: _Optional[str] = ...) -> None: ...

class CreateFileRequest(_message.Message):
    __slots__ = ("id", "path", "content", "is_dir")
    ID_FIELD_NUMBER: _ClassVar[int]
    PATH_FIELD_NUMBER: _ClassVar[int]
    CONTENT_FIELD_NUMBER: _ClassVar[int]
    IS_DIR_FIELD_NUMBER: _ClassVar[int]
    id: str
    path: str
    content: str
    is_dir: bool
    def __init__(self, id: _Optional[str] = ..., path: _Optional[str] = ..., content: _Optional[str] = ..., is_dir: _Optional[bool] = ...) -> None: ...

class RenameFileRequest(_message.Message):
    __slots__ = ("id", "to")
    ID_FIELD_NUMBER: _ClassVar[int]
    FROM_FIELD_NUMBER: _ClassVar[int]
    TO_FIELD_NUMBER: _ClassVar[int]
    id: str
    to: str
    def __init__(self, id: _Optional[str] = ..., to: _Optional[str] = ..., **kwargs) -> None: ...

class DeleteFileRequest(_message.Message):
    __slots__ = ("id", "path")
    ID_FIELD_NUMBER: _ClassVar[int]
    PATH_FIELD_NUMBER: _ClassVar[int]
    id: str
    path: str
    def __init__(self, id: _Optional[str] = ..., path: _Optional[str] = ...) -> None: ...

class DeleteFileResponse(_message.Message):
    __slots__ = ()
    def __init__(self) -> None: ...
