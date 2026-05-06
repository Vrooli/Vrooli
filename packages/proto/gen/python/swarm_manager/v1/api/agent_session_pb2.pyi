from buf.validate import validate_pb2 as _validate_pb2
from swarm_manager.v1.domain import agent_session_pb2 as _agent_session_pb2
from google.protobuf.internal import containers as _containers
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Iterable as _Iterable, Mapping as _Mapping
from typing import ClassVar as _ClassVar, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class ListAgentSessionsRequest(_message.Message):
    __slots__ = ("kind", "status", "active_only", "limit")
    KIND_FIELD_NUMBER: _ClassVar[int]
    STATUS_FIELD_NUMBER: _ClassVar[int]
    ACTIVE_ONLY_FIELD_NUMBER: _ClassVar[int]
    LIMIT_FIELD_NUMBER: _ClassVar[int]
    kind: str
    status: str
    active_only: bool
    limit: int
    def __init__(self, kind: _Optional[str] = ..., status: _Optional[str] = ..., active_only: _Optional[bool] = ..., limit: _Optional[int] = ...) -> None: ...

class ListAgentSessionsResponse(_message.Message):
    __slots__ = ("sessions",)
    SESSIONS_FIELD_NUMBER: _ClassVar[int]
    sessions: _containers.RepeatedCompositeFieldContainer[_agent_session_pb2.AgentSession]
    def __init__(self, sessions: _Optional[_Iterable[_Union[_agent_session_pb2.AgentSession, _Mapping]]] = ...) -> None: ...

class GetAgentSessionRequest(_message.Message):
    __slots__ = ("session_id",)
    SESSION_ID_FIELD_NUMBER: _ClassVar[int]
    session_id: str
    def __init__(self, session_id: _Optional[str] = ...) -> None: ...

class GetAgentSessionResponse(_message.Message):
    __slots__ = ("session",)
    SESSION_FIELD_NUMBER: _ClassVar[int]
    session: _agent_session_pb2.AgentSession
    def __init__(self, session: _Optional[_Union[_agent_session_pb2.AgentSession, _Mapping]] = ...) -> None: ...

class CreateAgentSessionRequest(_message.Message):
    __slots__ = ("kind", "title", "initial_message", "initiative")
    KIND_FIELD_NUMBER: _ClassVar[int]
    TITLE_FIELD_NUMBER: _ClassVar[int]
    INITIAL_MESSAGE_FIELD_NUMBER: _ClassVar[int]
    INITIATIVE_FIELD_NUMBER: _ClassVar[int]
    kind: str
    title: str
    initial_message: str
    initiative: str
    def __init__(self, kind: _Optional[str] = ..., title: _Optional[str] = ..., initial_message: _Optional[str] = ..., initiative: _Optional[str] = ...) -> None: ...

class CreateAgentSessionResponse(_message.Message):
    __slots__ = ("session",)
    SESSION_FIELD_NUMBER: _ClassVar[int]
    session: _agent_session_pb2.AgentSession
    def __init__(self, session: _Optional[_Union[_agent_session_pb2.AgentSession, _Mapping]] = ...) -> None: ...

class ContinueAgentSessionRequest(_message.Message):
    __slots__ = ("session_id", "message", "attachment_ids")
    SESSION_ID_FIELD_NUMBER: _ClassVar[int]
    MESSAGE_FIELD_NUMBER: _ClassVar[int]
    ATTACHMENT_IDS_FIELD_NUMBER: _ClassVar[int]
    session_id: str
    message: str
    attachment_ids: _containers.RepeatedScalarFieldContainer[str]
    def __init__(self, session_id: _Optional[str] = ..., message: _Optional[str] = ..., attachment_ids: _Optional[_Iterable[str]] = ...) -> None: ...

class ContinueAgentSessionResponse(_message.Message):
    __slots__ = ("session",)
    SESSION_FIELD_NUMBER: _ClassVar[int]
    session: _agent_session_pb2.AgentSession
    def __init__(self, session: _Optional[_Union[_agent_session_pb2.AgentSession, _Mapping]] = ...) -> None: ...

class RefreshAgentSessionRequest(_message.Message):
    __slots__ = ("session_id",)
    SESSION_ID_FIELD_NUMBER: _ClassVar[int]
    session_id: str
    def __init__(self, session_id: _Optional[str] = ...) -> None: ...

class RefreshAgentSessionResponse(_message.Message):
    __slots__ = ("session",)
    SESSION_FIELD_NUMBER: _ClassVar[int]
    session: _agent_session_pb2.AgentSession
    def __init__(self, session: _Optional[_Union[_agent_session_pb2.AgentSession, _Mapping]] = ...) -> None: ...

class CancelAgentSessionRequest(_message.Message):
    __slots__ = ("session_id",)
    SESSION_ID_FIELD_NUMBER: _ClassVar[int]
    session_id: str
    def __init__(self, session_id: _Optional[str] = ...) -> None: ...

class CancelAgentSessionResponse(_message.Message):
    __slots__ = ("session",)
    SESSION_FIELD_NUMBER: _ClassVar[int]
    session: _agent_session_pb2.AgentSession
    def __init__(self, session: _Optional[_Union[_agent_session_pb2.AgentSession, _Mapping]] = ...) -> None: ...

class DeleteAgentSessionRequest(_message.Message):
    __slots__ = ("session_id",)
    SESSION_ID_FIELD_NUMBER: _ClassVar[int]
    session_id: str
    def __init__(self, session_id: _Optional[str] = ...) -> None: ...

class DeleteAgentSessionResponse(_message.Message):
    __slots__ = ("session_id",)
    SESSION_ID_FIELD_NUMBER: _ClassVar[int]
    session_id: str
    def __init__(self, session_id: _Optional[str] = ...) -> None: ...

class ApplyAgentSessionProposalRequest(_message.Message):
    __slots__ = ("session_id", "proposal_id")
    SESSION_ID_FIELD_NUMBER: _ClassVar[int]
    PROPOSAL_ID_FIELD_NUMBER: _ClassVar[int]
    session_id: str
    proposal_id: str
    def __init__(self, session_id: _Optional[str] = ..., proposal_id: _Optional[str] = ...) -> None: ...

class ApplyAgentSessionProposalResponse(_message.Message):
    __slots__ = ("session", "artifacts")
    SESSION_FIELD_NUMBER: _ClassVar[int]
    ARTIFACTS_FIELD_NUMBER: _ClassVar[int]
    session: _agent_session_pb2.AgentSession
    artifacts: _containers.RepeatedCompositeFieldContainer[_agent_session_pb2.AgentSessionArtifact]
    def __init__(self, session: _Optional[_Union[_agent_session_pb2.AgentSession, _Mapping]] = ..., artifacts: _Optional[_Iterable[_Union[_agent_session_pb2.AgentSessionArtifact, _Mapping]]] = ...) -> None: ...

class ListAgentSessionArtifactsRequest(_message.Message):
    __slots__ = ("session_id",)
    SESSION_ID_FIELD_NUMBER: _ClassVar[int]
    session_id: str
    def __init__(self, session_id: _Optional[str] = ...) -> None: ...

class ListAgentSessionArtifactsResponse(_message.Message):
    __slots__ = ("artifacts",)
    ARTIFACTS_FIELD_NUMBER: _ClassVar[int]
    artifacts: _containers.RepeatedCompositeFieldContainer[_agent_session_pb2.AgentSessionArtifact]
    def __init__(self, artifacts: _Optional[_Iterable[_Union[_agent_session_pb2.AgentSessionArtifact, _Mapping]]] = ...) -> None: ...

class GetArtifactsByEntityRequest(_message.Message):
    __slots__ = ("artifact_type", "entity_ref")
    ARTIFACT_TYPE_FIELD_NUMBER: _ClassVar[int]
    ENTITY_REF_FIELD_NUMBER: _ClassVar[int]
    artifact_type: str
    entity_ref: str
    def __init__(self, artifact_type: _Optional[str] = ..., entity_ref: _Optional[str] = ...) -> None: ...

class GetArtifactsByEntityResponse(_message.Message):
    __slots__ = ("artifacts",)
    ARTIFACTS_FIELD_NUMBER: _ClassVar[int]
    artifacts: _containers.RepeatedCompositeFieldContainer[_agent_session_pb2.AgentSessionArtifact]
    def __init__(self, artifacts: _Optional[_Iterable[_Union[_agent_session_pb2.AgentSessionArtifact, _Mapping]]] = ...) -> None: ...
