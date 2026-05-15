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
    __slots__ = ("kind", "title")
    KIND_FIELD_NUMBER: _ClassVar[int]
    TITLE_FIELD_NUMBER: _ClassVar[int]
    kind: str
    title: str
    def __init__(self, kind: _Optional[str] = ..., title: _Optional[str] = ...) -> None: ...

class CreateAgentSessionResponse(_message.Message):
    __slots__ = ("session",)
    SESSION_FIELD_NUMBER: _ClassVar[int]
    session: _agent_session_pb2.AgentSession
    def __init__(self, session: _Optional[_Union[_agent_session_pb2.AgentSession, _Mapping]] = ...) -> None: ...

class StartAgentSessionRequest(_message.Message):
    __slots__ = ("session_id", "message", "attachment_ids", "context_refs")
    SESSION_ID_FIELD_NUMBER: _ClassVar[int]
    MESSAGE_FIELD_NUMBER: _ClassVar[int]
    ATTACHMENT_IDS_FIELD_NUMBER: _ClassVar[int]
    CONTEXT_REFS_FIELD_NUMBER: _ClassVar[int]
    session_id: str
    message: str
    attachment_ids: _containers.RepeatedScalarFieldContainer[str]
    context_refs: _containers.RepeatedCompositeFieldContainer[AgentSessionContextRef]
    def __init__(self, session_id: _Optional[str] = ..., message: _Optional[str] = ..., attachment_ids: _Optional[_Iterable[str]] = ..., context_refs: _Optional[_Iterable[_Union[AgentSessionContextRef, _Mapping]]] = ...) -> None: ...

class StartAgentSessionResponse(_message.Message):
    __slots__ = ("session",)
    SESSION_FIELD_NUMBER: _ClassVar[int]
    session: _agent_session_pb2.AgentSession
    def __init__(self, session: _Optional[_Union[_agent_session_pb2.AgentSession, _Mapping]] = ...) -> None: ...

class ContinueAgentSessionRequest(_message.Message):
    __slots__ = ("session_id", "message", "attachment_ids", "context_refs")
    SESSION_ID_FIELD_NUMBER: _ClassVar[int]
    MESSAGE_FIELD_NUMBER: _ClassVar[int]
    ATTACHMENT_IDS_FIELD_NUMBER: _ClassVar[int]
    CONTEXT_REFS_FIELD_NUMBER: _ClassVar[int]
    session_id: str
    message: str
    attachment_ids: _containers.RepeatedScalarFieldContainer[str]
    context_refs: _containers.RepeatedCompositeFieldContainer[AgentSessionContextRef]
    def __init__(self, session_id: _Optional[str] = ..., message: _Optional[str] = ..., attachment_ids: _Optional[_Iterable[str]] = ..., context_refs: _Optional[_Iterable[_Union[AgentSessionContextRef, _Mapping]]] = ...) -> None: ...

class ContinueAgentSessionResponse(_message.Message):
    __slots__ = ("session",)
    SESSION_FIELD_NUMBER: _ClassVar[int]
    session: _agent_session_pb2.AgentSession
    def __init__(self, session: _Optional[_Union[_agent_session_pb2.AgentSession, _Mapping]] = ...) -> None: ...

class AgentSessionContextRef(_message.Message):
    __slots__ = ("type", "ref")
    TYPE_FIELD_NUMBER: _ClassVar[int]
    REF_FIELD_NUMBER: _ClassVar[int]
    type: str
    ref: str
    def __init__(self, type: _Optional[str] = ..., ref: _Optional[str] = ...) -> None: ...

class UploadAgentSessionAttachmentsResponse(_message.Message):
    __slots__ = ("attachments",)
    ATTACHMENTS_FIELD_NUMBER: _ClassVar[int]
    attachments: _containers.RepeatedCompositeFieldContainer[_agent_session_pb2.AgentSessionAttachment]
    def __init__(self, attachments: _Optional[_Iterable[_Union[_agent_session_pb2.AgentSessionAttachment, _Mapping]]] = ...) -> None: ...

class ListAgentSessionEventsRequest(_message.Message):
    __slots__ = ("session_id", "after_sequence", "limit")
    SESSION_ID_FIELD_NUMBER: _ClassVar[int]
    AFTER_SEQUENCE_FIELD_NUMBER: _ClassVar[int]
    LIMIT_FIELD_NUMBER: _ClassVar[int]
    session_id: str
    after_sequence: int
    limit: int
    def __init__(self, session_id: _Optional[str] = ..., after_sequence: _Optional[int] = ..., limit: _Optional[int] = ...) -> None: ...

class AgentSessionRunEvent(_message.Message):
    __slots__ = ("id", "run_id", "sequence", "created_at", "event_type", "role", "content", "tool_name", "tool_call_id", "input", "output", "error", "status", "previous_status", "progress_phase", "progress_percent", "progress_message", "summary", "raw_json")
    ID_FIELD_NUMBER: _ClassVar[int]
    RUN_ID_FIELD_NUMBER: _ClassVar[int]
    SEQUENCE_FIELD_NUMBER: _ClassVar[int]
    CREATED_AT_FIELD_NUMBER: _ClassVar[int]
    EVENT_TYPE_FIELD_NUMBER: _ClassVar[int]
    ROLE_FIELD_NUMBER: _ClassVar[int]
    CONTENT_FIELD_NUMBER: _ClassVar[int]
    TOOL_NAME_FIELD_NUMBER: _ClassVar[int]
    TOOL_CALL_ID_FIELD_NUMBER: _ClassVar[int]
    INPUT_FIELD_NUMBER: _ClassVar[int]
    OUTPUT_FIELD_NUMBER: _ClassVar[int]
    ERROR_FIELD_NUMBER: _ClassVar[int]
    STATUS_FIELD_NUMBER: _ClassVar[int]
    PREVIOUS_STATUS_FIELD_NUMBER: _ClassVar[int]
    PROGRESS_PHASE_FIELD_NUMBER: _ClassVar[int]
    PROGRESS_PERCENT_FIELD_NUMBER: _ClassVar[int]
    PROGRESS_MESSAGE_FIELD_NUMBER: _ClassVar[int]
    SUMMARY_FIELD_NUMBER: _ClassVar[int]
    RAW_JSON_FIELD_NUMBER: _ClassVar[int]
    id: str
    run_id: str
    sequence: int
    created_at: str
    event_type: str
    role: str
    content: str
    tool_name: str
    tool_call_id: str
    input: str
    output: str
    error: str
    status: str
    previous_status: str
    progress_phase: str
    progress_percent: int
    progress_message: str
    summary: str
    raw_json: str
    def __init__(self, id: _Optional[str] = ..., run_id: _Optional[str] = ..., sequence: _Optional[int] = ..., created_at: _Optional[str] = ..., event_type: _Optional[str] = ..., role: _Optional[str] = ..., content: _Optional[str] = ..., tool_name: _Optional[str] = ..., tool_call_id: _Optional[str] = ..., input: _Optional[str] = ..., output: _Optional[str] = ..., error: _Optional[str] = ..., status: _Optional[str] = ..., previous_status: _Optional[str] = ..., progress_phase: _Optional[str] = ..., progress_percent: _Optional[int] = ..., progress_message: _Optional[str] = ..., summary: _Optional[str] = ..., raw_json: _Optional[str] = ...) -> None: ...

class ListAgentSessionEventsResponse(_message.Message):
    __slots__ = ("events", "has_more", "next_after_sequence")
    EVENTS_FIELD_NUMBER: _ClassVar[int]
    HAS_MORE_FIELD_NUMBER: _ClassVar[int]
    NEXT_AFTER_SEQUENCE_FIELD_NUMBER: _ClassVar[int]
    events: _containers.RepeatedCompositeFieldContainer[AgentSessionRunEvent]
    has_more: bool
    next_after_sequence: int
    def __init__(self, events: _Optional[_Iterable[_Union[AgentSessionRunEvent, _Mapping]]] = ..., has_more: _Optional[bool] = ..., next_after_sequence: _Optional[int] = ...) -> None: ...

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
