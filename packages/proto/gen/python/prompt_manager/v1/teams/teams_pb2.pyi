from google.protobuf import field_mask_pb2 as _field_mask_pb2
from google.protobuf import struct_pb2 as _struct_pb2
from google.protobuf.internal import containers as _containers
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Iterable as _Iterable, Mapping as _Mapping
from typing import ClassVar as _ClassVar, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class Team(_message.Message):
    __slots__ = ("id", "display_name", "mission", "enabled", "runtime", "coordination", "execution", "operating_contract", "validation_findings", "member_count", "created_at", "updated_at")
    ID_FIELD_NUMBER: _ClassVar[int]
    DISPLAY_NAME_FIELD_NUMBER: _ClassVar[int]
    MISSION_FIELD_NUMBER: _ClassVar[int]
    ENABLED_FIELD_NUMBER: _ClassVar[int]
    RUNTIME_FIELD_NUMBER: _ClassVar[int]
    COORDINATION_FIELD_NUMBER: _ClassVar[int]
    EXECUTION_FIELD_NUMBER: _ClassVar[int]
    OPERATING_CONTRACT_FIELD_NUMBER: _ClassVar[int]
    VALIDATION_FINDINGS_FIELD_NUMBER: _ClassVar[int]
    MEMBER_COUNT_FIELD_NUMBER: _ClassVar[int]
    CREATED_AT_FIELD_NUMBER: _ClassVar[int]
    UPDATED_AT_FIELD_NUMBER: _ClassVar[int]
    id: str
    display_name: str
    mission: str
    enabled: bool
    runtime: _struct_pb2.Struct
    coordination: _struct_pb2.Struct
    execution: _struct_pb2.Struct
    operating_contract: _struct_pb2.Struct
    validation_findings: _containers.RepeatedCompositeFieldContainer[_struct_pb2.Struct]
    member_count: int
    created_at: str
    updated_at: str
    def __init__(self, id: _Optional[str] = ..., display_name: _Optional[str] = ..., mission: _Optional[str] = ..., enabled: _Optional[bool] = ..., runtime: _Optional[_Union[_struct_pb2.Struct, _Mapping]] = ..., coordination: _Optional[_Union[_struct_pb2.Struct, _Mapping]] = ..., execution: _Optional[_Union[_struct_pb2.Struct, _Mapping]] = ..., operating_contract: _Optional[_Union[_struct_pb2.Struct, _Mapping]] = ..., validation_findings: _Optional[_Iterable[_Union[_struct_pb2.Struct, _Mapping]]] = ..., member_count: _Optional[int] = ..., created_at: _Optional[str] = ..., updated_at: _Optional[str] = ...) -> None: ...

class Role(_message.Message):
    __slots__ = ("id", "name", "description")
    ID_FIELD_NUMBER: _ClassVar[int]
    NAME_FIELD_NUMBER: _ClassVar[int]
    DESCRIPTION_FIELD_NUMBER: _ClassVar[int]
    id: str
    name: str
    description: str
    def __init__(self, id: _Optional[str] = ..., name: _Optional[str] = ..., description: _Optional[str] = ...) -> None: ...

class Member(_message.Message):
    __slots__ = ("agent_id", "display_name", "roles", "status")
    AGENT_ID_FIELD_NUMBER: _ClassVar[int]
    DISPLAY_NAME_FIELD_NUMBER: _ClassVar[int]
    ROLES_FIELD_NUMBER: _ClassVar[int]
    STATUS_FIELD_NUMBER: _ClassVar[int]
    agent_id: str
    display_name: str
    roles: _containers.RepeatedScalarFieldContainer[str]
    status: str
    def __init__(self, agent_id: _Optional[str] = ..., display_name: _Optional[str] = ..., roles: _Optional[_Iterable[str]] = ..., status: _Optional[str] = ...) -> None: ...

class TeamDetails(_message.Message):
    __slots__ = ("id", "display_name", "mission", "enabled", "runtime", "coordination", "execution", "operating_contract", "validation_findings", "member_count", "created_at", "updated_at", "roles", "members")
    ID_FIELD_NUMBER: _ClassVar[int]
    DISPLAY_NAME_FIELD_NUMBER: _ClassVar[int]
    MISSION_FIELD_NUMBER: _ClassVar[int]
    ENABLED_FIELD_NUMBER: _ClassVar[int]
    RUNTIME_FIELD_NUMBER: _ClassVar[int]
    COORDINATION_FIELD_NUMBER: _ClassVar[int]
    EXECUTION_FIELD_NUMBER: _ClassVar[int]
    OPERATING_CONTRACT_FIELD_NUMBER: _ClassVar[int]
    VALIDATION_FINDINGS_FIELD_NUMBER: _ClassVar[int]
    MEMBER_COUNT_FIELD_NUMBER: _ClassVar[int]
    CREATED_AT_FIELD_NUMBER: _ClassVar[int]
    UPDATED_AT_FIELD_NUMBER: _ClassVar[int]
    ROLES_FIELD_NUMBER: _ClassVar[int]
    MEMBERS_FIELD_NUMBER: _ClassVar[int]
    id: str
    display_name: str
    mission: str
    enabled: bool
    runtime: _struct_pb2.Struct
    coordination: _struct_pb2.Struct
    execution: _struct_pb2.Struct
    operating_contract: _struct_pb2.Struct
    validation_findings: _containers.RepeatedCompositeFieldContainer[_struct_pb2.Struct]
    member_count: int
    created_at: str
    updated_at: str
    roles: _containers.RepeatedCompositeFieldContainer[Role]
    members: _containers.RepeatedCompositeFieldContainer[Member]
    def __init__(self, id: _Optional[str] = ..., display_name: _Optional[str] = ..., mission: _Optional[str] = ..., enabled: _Optional[bool] = ..., runtime: _Optional[_Union[_struct_pb2.Struct, _Mapping]] = ..., coordination: _Optional[_Union[_struct_pb2.Struct, _Mapping]] = ..., execution: _Optional[_Union[_struct_pb2.Struct, _Mapping]] = ..., operating_contract: _Optional[_Union[_struct_pb2.Struct, _Mapping]] = ..., validation_findings: _Optional[_Iterable[_Union[_struct_pb2.Struct, _Mapping]]] = ..., member_count: _Optional[int] = ..., created_at: _Optional[str] = ..., updated_at: _Optional[str] = ..., roles: _Optional[_Iterable[_Union[Role, _Mapping]]] = ..., members: _Optional[_Iterable[_Union[Member, _Mapping]]] = ...) -> None: ...

class TeamInput(_message.Message):
    __slots__ = ("id", "display_name", "mission", "enabled", "runtime", "coordination", "execution", "operating_contract")
    ID_FIELD_NUMBER: _ClassVar[int]
    DISPLAY_NAME_FIELD_NUMBER: _ClassVar[int]
    MISSION_FIELD_NUMBER: _ClassVar[int]
    ENABLED_FIELD_NUMBER: _ClassVar[int]
    RUNTIME_FIELD_NUMBER: _ClassVar[int]
    COORDINATION_FIELD_NUMBER: _ClassVar[int]
    EXECUTION_FIELD_NUMBER: _ClassVar[int]
    OPERATING_CONTRACT_FIELD_NUMBER: _ClassVar[int]
    id: str
    display_name: str
    mission: str
    enabled: bool
    runtime: _struct_pb2.Struct
    coordination: _struct_pb2.Struct
    execution: _struct_pb2.Struct
    operating_contract: _struct_pb2.Struct
    def __init__(self, id: _Optional[str] = ..., display_name: _Optional[str] = ..., mission: _Optional[str] = ..., enabled: _Optional[bool] = ..., runtime: _Optional[_Union[_struct_pb2.Struct, _Mapping]] = ..., coordination: _Optional[_Union[_struct_pb2.Struct, _Mapping]] = ..., execution: _Optional[_Union[_struct_pb2.Struct, _Mapping]] = ..., operating_contract: _Optional[_Union[_struct_pb2.Struct, _Mapping]] = ...) -> None: ...

class ListTeamsRequest(_message.Message):
    __slots__ = ()
    def __init__(self) -> None: ...

class ListTeamsResponse(_message.Message):
    __slots__ = ("teams",)
    TEAMS_FIELD_NUMBER: _ClassVar[int]
    teams: _containers.RepeatedCompositeFieldContainer[Team]
    def __init__(self, teams: _Optional[_Iterable[_Union[Team, _Mapping]]] = ...) -> None: ...

class GetTeamRequest(_message.Message):
    __slots__ = ("id",)
    ID_FIELD_NUMBER: _ClassVar[int]
    id: str
    def __init__(self, id: _Optional[str] = ...) -> None: ...

class CreateTeamRequest(_message.Message):
    __slots__ = ("team",)
    TEAM_FIELD_NUMBER: _ClassVar[int]
    team: TeamInput
    def __init__(self, team: _Optional[_Union[TeamInput, _Mapping]] = ...) -> None: ...

class UpdateTeamRequest(_message.Message):
    __slots__ = ("id", "team", "update_mask")
    ID_FIELD_NUMBER: _ClassVar[int]
    TEAM_FIELD_NUMBER: _ClassVar[int]
    UPDATE_MASK_FIELD_NUMBER: _ClassVar[int]
    id: str
    team: TeamInput
    update_mask: _field_mask_pb2.FieldMask
    def __init__(self, id: _Optional[str] = ..., team: _Optional[_Union[TeamInput, _Mapping]] = ..., update_mask: _Optional[_Union[_field_mask_pb2.FieldMask, _Mapping]] = ...) -> None: ...

class DeleteTeamRequest(_message.Message):
    __slots__ = ("id",)
    ID_FIELD_NUMBER: _ClassVar[int]
    id: str
    def __init__(self, id: _Optional[str] = ...) -> None: ...

class DeleteTeamResponse(_message.Message):
    __slots__ = ()
    def __init__(self) -> None: ...

class ExclusiveMember(_message.Message):
    __slots__ = ("agent_id", "display_name")
    AGENT_ID_FIELD_NUMBER: _ClassVar[int]
    DISPLAY_NAME_FIELD_NUMBER: _ClassVar[int]
    agent_id: str
    display_name: str
    def __init__(self, agent_id: _Optional[str] = ..., display_name: _Optional[str] = ...) -> None: ...

class GetExclusiveMembersRequest(_message.Message):
    __slots__ = ("team_id",)
    TEAM_ID_FIELD_NUMBER: _ClassVar[int]
    team_id: str
    def __init__(self, team_id: _Optional[str] = ...) -> None: ...

class ExclusiveMembersResponse(_message.Message):
    __slots__ = ("team_id", "members")
    TEAM_ID_FIELD_NUMBER: _ClassVar[int]
    MEMBERS_FIELD_NUMBER: _ClassVar[int]
    team_id: str
    members: _containers.RepeatedCompositeFieldContainer[ExclusiveMember]
    def __init__(self, team_id: _Optional[str] = ..., members: _Optional[_Iterable[_Union[ExclusiveMember, _Mapping]]] = ...) -> None: ...

class AddMemberRequest(_message.Message):
    __slots__ = ("team_id", "agent_id", "roles")
    TEAM_ID_FIELD_NUMBER: _ClassVar[int]
    AGENT_ID_FIELD_NUMBER: _ClassVar[int]
    ROLES_FIELD_NUMBER: _ClassVar[int]
    team_id: str
    agent_id: str
    roles: _containers.RepeatedScalarFieldContainer[str]
    def __init__(self, team_id: _Optional[str] = ..., agent_id: _Optional[str] = ..., roles: _Optional[_Iterable[str]] = ...) -> None: ...

class UpdateMemberRequest(_message.Message):
    __slots__ = ("team_id", "agent_id", "roles", "status")
    TEAM_ID_FIELD_NUMBER: _ClassVar[int]
    AGENT_ID_FIELD_NUMBER: _ClassVar[int]
    ROLES_FIELD_NUMBER: _ClassVar[int]
    STATUS_FIELD_NUMBER: _ClassVar[int]
    team_id: str
    agent_id: str
    roles: _containers.RepeatedScalarFieldContainer[str]
    status: str
    def __init__(self, team_id: _Optional[str] = ..., agent_id: _Optional[str] = ..., roles: _Optional[_Iterable[str]] = ..., status: _Optional[str] = ...) -> None: ...

class RemoveMemberRequest(_message.Message):
    __slots__ = ("team_id", "agent_id")
    TEAM_ID_FIELD_NUMBER: _ClassVar[int]
    AGENT_ID_FIELD_NUMBER: _ClassVar[int]
    team_id: str
    agent_id: str
    def __init__(self, team_id: _Optional[str] = ..., agent_id: _Optional[str] = ...) -> None: ...

class RemoveMemberResponse(_message.Message):
    __slots__ = ()
    def __init__(self) -> None: ...

class GetRolesRequest(_message.Message):
    __slots__ = ("team_id",)
    TEAM_ID_FIELD_NUMBER: _ClassVar[int]
    team_id: str
    def __init__(self, team_id: _Optional[str] = ...) -> None: ...

class GetRolesResponse(_message.Message):
    __slots__ = ("roles",)
    ROLES_FIELD_NUMBER: _ClassVar[int]
    roles: _containers.RepeatedCompositeFieldContainer[Role]
    def __init__(self, roles: _Optional[_Iterable[_Union[Role, _Mapping]]] = ...) -> None: ...

class SetRolesRequest(_message.Message):
    __slots__ = ("team_id", "roles")
    TEAM_ID_FIELD_NUMBER: _ClassVar[int]
    ROLES_FIELD_NUMBER: _ClassVar[int]
    team_id: str
    roles: _containers.RepeatedCompositeFieldContainer[Role]
    def __init__(self, team_id: _Optional[str] = ..., roles: _Optional[_Iterable[_Union[Role, _Mapping]]] = ...) -> None: ...

class SharedFileEntry(_message.Message):
    __slots__ = ("path", "is_dir", "size")
    PATH_FIELD_NUMBER: _ClassVar[int]
    IS_DIR_FIELD_NUMBER: _ClassVar[int]
    SIZE_FIELD_NUMBER: _ClassVar[int]
    path: str
    is_dir: bool
    size: int
    def __init__(self, path: _Optional[str] = ..., is_dir: _Optional[bool] = ..., size: _Optional[int] = ...) -> None: ...

class ListSharedFilesRequest(_message.Message):
    __slots__ = ("team_id",)
    TEAM_ID_FIELD_NUMBER: _ClassVar[int]
    team_id: str
    def __init__(self, team_id: _Optional[str] = ...) -> None: ...

class ListSharedFilesResponse(_message.Message):
    __slots__ = ("team_id", "files")
    TEAM_ID_FIELD_NUMBER: _ClassVar[int]
    FILES_FIELD_NUMBER: _ClassVar[int]
    team_id: str
    files: _containers.RepeatedCompositeFieldContainer[SharedFileEntry]
    def __init__(self, team_id: _Optional[str] = ..., files: _Optional[_Iterable[_Union[SharedFileEntry, _Mapping]]] = ...) -> None: ...

class SharedFileContent(_message.Message):
    __slots__ = ("team_id", "path", "content")
    TEAM_ID_FIELD_NUMBER: _ClassVar[int]
    PATH_FIELD_NUMBER: _ClassVar[int]
    CONTENT_FIELD_NUMBER: _ClassVar[int]
    team_id: str
    path: str
    content: str
    def __init__(self, team_id: _Optional[str] = ..., path: _Optional[str] = ..., content: _Optional[str] = ...) -> None: ...

class GetSharedFileRequest(_message.Message):
    __slots__ = ("team_id", "path")
    TEAM_ID_FIELD_NUMBER: _ClassVar[int]
    PATH_FIELD_NUMBER: _ClassVar[int]
    team_id: str
    path: str
    def __init__(self, team_id: _Optional[str] = ..., path: _Optional[str] = ...) -> None: ...

class SetSharedFileRequest(_message.Message):
    __slots__ = ("team_id", "path", "content")
    TEAM_ID_FIELD_NUMBER: _ClassVar[int]
    PATH_FIELD_NUMBER: _ClassVar[int]
    CONTENT_FIELD_NUMBER: _ClassVar[int]
    team_id: str
    path: str
    content: str
    def __init__(self, team_id: _Optional[str] = ..., path: _Optional[str] = ..., content: _Optional[str] = ...) -> None: ...

class CreateSharedFileRequest(_message.Message):
    __slots__ = ("team_id", "path", "content", "is_dir")
    TEAM_ID_FIELD_NUMBER: _ClassVar[int]
    PATH_FIELD_NUMBER: _ClassVar[int]
    CONTENT_FIELD_NUMBER: _ClassVar[int]
    IS_DIR_FIELD_NUMBER: _ClassVar[int]
    team_id: str
    path: str
    content: str
    is_dir: bool
    def __init__(self, team_id: _Optional[str] = ..., path: _Optional[str] = ..., content: _Optional[str] = ..., is_dir: _Optional[bool] = ...) -> None: ...

class RenameSharedFileRequest(_message.Message):
    __slots__ = ("team_id", "to")
    TEAM_ID_FIELD_NUMBER: _ClassVar[int]
    FROM_FIELD_NUMBER: _ClassVar[int]
    TO_FIELD_NUMBER: _ClassVar[int]
    team_id: str
    to: str
    def __init__(self, team_id: _Optional[str] = ..., to: _Optional[str] = ..., **kwargs) -> None: ...

class DeleteSharedFileRequest(_message.Message):
    __slots__ = ("team_id", "path")
    TEAM_ID_FIELD_NUMBER: _ClassVar[int]
    PATH_FIELD_NUMBER: _ClassVar[int]
    team_id: str
    path: str
    def __init__(self, team_id: _Optional[str] = ..., path: _Optional[str] = ...) -> None: ...

class DeleteSharedFileResponse(_message.Message):
    __slots__ = ()
    def __init__(self) -> None: ...

class OrgEdge(_message.Message):
    __slots__ = ("manager_agent_id", "report_agent_id")
    MANAGER_AGENT_ID_FIELD_NUMBER: _ClassVar[int]
    REPORT_AGENT_ID_FIELD_NUMBER: _ClassVar[int]
    manager_agent_id: str
    report_agent_id: str
    def __init__(self, manager_agent_id: _Optional[str] = ..., report_agent_id: _Optional[str] = ...) -> None: ...

class OrgChart(_message.Message):
    __slots__ = ("team_id", "edges")
    TEAM_ID_FIELD_NUMBER: _ClassVar[int]
    EDGES_FIELD_NUMBER: _ClassVar[int]
    team_id: str
    edges: _containers.RepeatedCompositeFieldContainer[OrgEdge]
    def __init__(self, team_id: _Optional[str] = ..., edges: _Optional[_Iterable[_Union[OrgEdge, _Mapping]]] = ...) -> None: ...

class GetOrgChartRequest(_message.Message):
    __slots__ = ("team_id",)
    TEAM_ID_FIELD_NUMBER: _ClassVar[int]
    team_id: str
    def __init__(self, team_id: _Optional[str] = ...) -> None: ...

class SetOrgChartRequest(_message.Message):
    __slots__ = ("team_id", "edges")
    TEAM_ID_FIELD_NUMBER: _ClassVar[int]
    EDGES_FIELD_NUMBER: _ClassVar[int]
    team_id: str
    edges: _containers.RepeatedCompositeFieldContainer[OrgEdge]
    def __init__(self, team_id: _Optional[str] = ..., edges: _Optional[_Iterable[_Union[OrgEdge, _Mapping]]] = ...) -> None: ...

class UpdateOrgChartEdgeRequest(_message.Message):
    __slots__ = ("team_id", "report_agent_id", "manager_agent_id")
    TEAM_ID_FIELD_NUMBER: _ClassVar[int]
    REPORT_AGENT_ID_FIELD_NUMBER: _ClassVar[int]
    MANAGER_AGENT_ID_FIELD_NUMBER: _ClassVar[int]
    team_id: str
    report_agent_id: str
    manager_agent_id: str
    def __init__(self, team_id: _Optional[str] = ..., report_agent_id: _Optional[str] = ..., manager_agent_id: _Optional[str] = ...) -> None: ...

class DeleteOrgChartEdgeRequest(_message.Message):
    __slots__ = ("team_id", "report_agent_id")
    TEAM_ID_FIELD_NUMBER: _ClassVar[int]
    REPORT_AGENT_ID_FIELD_NUMBER: _ClassVar[int]
    team_id: str
    report_agent_id: str
    def __init__(self, team_id: _Optional[str] = ..., report_agent_id: _Optional[str] = ...) -> None: ...

class Message(_message.Message):
    __slots__ = ("id", "team_id", "from_agent_id", "to_agent_id", "content", "created_at")
    ID_FIELD_NUMBER: _ClassVar[int]
    TEAM_ID_FIELD_NUMBER: _ClassVar[int]
    FROM_AGENT_ID_FIELD_NUMBER: _ClassVar[int]
    TO_AGENT_ID_FIELD_NUMBER: _ClassVar[int]
    CONTENT_FIELD_NUMBER: _ClassVar[int]
    CREATED_AT_FIELD_NUMBER: _ClassVar[int]
    id: str
    team_id: str
    from_agent_id: str
    to_agent_id: str
    content: str
    created_at: str
    def __init__(self, id: _Optional[str] = ..., team_id: _Optional[str] = ..., from_agent_id: _Optional[str] = ..., to_agent_id: _Optional[str] = ..., content: _Optional[str] = ..., created_at: _Optional[str] = ...) -> None: ...

class Inbox(_message.Message):
    __slots__ = ("team_id", "agent_id", "messages")
    TEAM_ID_FIELD_NUMBER: _ClassVar[int]
    AGENT_ID_FIELD_NUMBER: _ClassVar[int]
    MESSAGES_FIELD_NUMBER: _ClassVar[int]
    team_id: str
    agent_id: str
    messages: _containers.RepeatedCompositeFieldContainer[Message]
    def __init__(self, team_id: _Optional[str] = ..., agent_id: _Optional[str] = ..., messages: _Optional[_Iterable[_Union[Message, _Mapping]]] = ...) -> None: ...

class ListMessagesRequest(_message.Message):
    __slots__ = ("team_id", "agent_id")
    TEAM_ID_FIELD_NUMBER: _ClassVar[int]
    AGENT_ID_FIELD_NUMBER: _ClassVar[int]
    team_id: str
    agent_id: str
    def __init__(self, team_id: _Optional[str] = ..., agent_id: _Optional[str] = ...) -> None: ...

class SendMessageRequest(_message.Message):
    __slots__ = ("team_id", "agent_id", "from_agent_id", "content")
    TEAM_ID_FIELD_NUMBER: _ClassVar[int]
    AGENT_ID_FIELD_NUMBER: _ClassVar[int]
    FROM_AGENT_ID_FIELD_NUMBER: _ClassVar[int]
    CONTENT_FIELD_NUMBER: _ClassVar[int]
    team_id: str
    agent_id: str
    from_agent_id: str
    content: str
    def __init__(self, team_id: _Optional[str] = ..., agent_id: _Optional[str] = ..., from_agent_id: _Optional[str] = ..., content: _Optional[str] = ...) -> None: ...

class ClearMessagesRequest(_message.Message):
    __slots__ = ("team_id", "agent_id")
    TEAM_ID_FIELD_NUMBER: _ClassVar[int]
    AGENT_ID_FIELD_NUMBER: _ClassVar[int]
    team_id: str
    agent_id: str
    def __init__(self, team_id: _Optional[str] = ..., agent_id: _Optional[str] = ...) -> None: ...

class ClearMessagesResponse(_message.Message):
    __slots__ = ()
    def __init__(self) -> None: ...

class DeleteMessageRequest(_message.Message):
    __slots__ = ("team_id", "agent_id", "message_id")
    TEAM_ID_FIELD_NUMBER: _ClassVar[int]
    AGENT_ID_FIELD_NUMBER: _ClassVar[int]
    MESSAGE_ID_FIELD_NUMBER: _ClassVar[int]
    team_id: str
    agent_id: str
    message_id: str
    def __init__(self, team_id: _Optional[str] = ..., agent_id: _Optional[str] = ..., message_id: _Optional[str] = ...) -> None: ...

class DeleteMessageResponse(_message.Message):
    __slots__ = ()
    def __init__(self) -> None: ...

class AvailableClaudeCodeTeam(_message.Message):
    __slots__ = ("name", "member_count")
    NAME_FIELD_NUMBER: _ClassVar[int]
    MEMBER_COUNT_FIELD_NUMBER: _ClassVar[int]
    name: str
    member_count: int
    def __init__(self, name: _Optional[str] = ..., member_count: _Optional[int] = ...) -> None: ...

class ListAvailableClaudeCodeTeamsRequest(_message.Message):
    __slots__ = ()
    def __init__(self) -> None: ...

class ListAvailableClaudeCodeTeamsResponse(_message.Message):
    __slots__ = ("teams",)
    TEAMS_FIELD_NUMBER: _ClassVar[int]
    teams: _containers.RepeatedCompositeFieldContainer[AvailableClaudeCodeTeam]
    def __init__(self, teams: _Optional[_Iterable[_Union[AvailableClaudeCodeTeam, _Mapping]]] = ...) -> None: ...

class ImportClaudeCodeTeamRequest(_message.Message):
    __slots__ = ("team_name",)
    TEAM_NAME_FIELD_NUMBER: _ClassVar[int]
    team_name: str
    def __init__(self, team_name: _Optional[str] = ...) -> None: ...

class ExportClaudeCodeTeamRequest(_message.Message):
    __slots__ = ("team_id",)
    TEAM_ID_FIELD_NUMBER: _ClassVar[int]
    team_id: str
    def __init__(self, team_id: _Optional[str] = ...) -> None: ...

class ExportClaudeCodeTeamResponse(_message.Message):
    __slots__ = ("team_id", "export")
    TEAM_ID_FIELD_NUMBER: _ClassVar[int]
    EXPORT_FIELD_NUMBER: _ClassVar[int]
    team_id: str
    export: _struct_pb2.Struct
    def __init__(self, team_id: _Optional[str] = ..., export: _Optional[_Union[_struct_pb2.Struct, _Mapping]] = ...) -> None: ...
