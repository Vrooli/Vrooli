from buf.validate import validate_pb2 as _validate_pb2
from swarm_manager.v1.shared import agent_session_pb2 as _agent_session_pb2
from google.protobuf.internal import containers as _containers
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Iterable as _Iterable, Mapping as _Mapping
from typing import ClassVar as _ClassVar, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class AgentSessionMessage(_message.Message):
    __slots__ = ("id", "role", "content", "created_at", "attachment_ids", "context")
    ID_FIELD_NUMBER: _ClassVar[int]
    ROLE_FIELD_NUMBER: _ClassVar[int]
    CONTENT_FIELD_NUMBER: _ClassVar[int]
    CREATED_AT_FIELD_NUMBER: _ClassVar[int]
    ATTACHMENT_IDS_FIELD_NUMBER: _ClassVar[int]
    CONTEXT_FIELD_NUMBER: _ClassVar[int]
    id: str
    role: str
    content: str
    created_at: str
    attachment_ids: _containers.RepeatedScalarFieldContainer[str]
    context: _containers.RepeatedCompositeFieldContainer[_agent_session_pb2.AgentSessionContextItem]
    def __init__(self, id: _Optional[str] = ..., role: _Optional[str] = ..., content: _Optional[str] = ..., created_at: _Optional[str] = ..., attachment_ids: _Optional[_Iterable[str]] = ..., context: _Optional[_Iterable[_Union[_agent_session_pb2.AgentSessionContextItem, _Mapping]]] = ...) -> None: ...

class AgentSessionProposal(_message.Message):
    __slots__ = ("id", "kind", "status", "summary", "payload_json", "created_at", "updated_at", "attribution")
    ID_FIELD_NUMBER: _ClassVar[int]
    KIND_FIELD_NUMBER: _ClassVar[int]
    STATUS_FIELD_NUMBER: _ClassVar[int]
    SUMMARY_FIELD_NUMBER: _ClassVar[int]
    PAYLOAD_JSON_FIELD_NUMBER: _ClassVar[int]
    CREATED_AT_FIELD_NUMBER: _ClassVar[int]
    UPDATED_AT_FIELD_NUMBER: _ClassVar[int]
    ATTRIBUTION_FIELD_NUMBER: _ClassVar[int]
    id: str
    kind: str
    status: str
    summary: str
    payload_json: str
    created_at: str
    updated_at: str
    attribution: _agent_session_pb2.AgentSessionAttribution
    def __init__(self, id: _Optional[str] = ..., kind: _Optional[str] = ..., status: _Optional[str] = ..., summary: _Optional[str] = ..., payload_json: _Optional[str] = ..., created_at: _Optional[str] = ..., updated_at: _Optional[str] = ..., attribution: _Optional[_Union[_agent_session_pb2.AgentSessionAttribution, _Mapping]] = ...) -> None: ...

class AgentSessionProposalTarget(_message.Message):
    __slots__ = ("type", "ref", "name")
    TYPE_FIELD_NUMBER: _ClassVar[int]
    REF_FIELD_NUMBER: _ClassVar[int]
    NAME_FIELD_NUMBER: _ClassVar[int]
    type: str
    ref: str
    name: str
    def __init__(self, type: _Optional[str] = ..., ref: _Optional[str] = ..., name: _Optional[str] = ...) -> None: ...

class AgentSession(_message.Message):
    __slots__ = ("id", "title", "kind", "status", "skill_id", "task_id", "run_id", "profile_key", "failure_reason", "created_at", "updated_at", "messages", "proposals", "artifacts", "created_by", "attachments", "proposal_target", "starter_job_id")
    ID_FIELD_NUMBER: _ClassVar[int]
    TITLE_FIELD_NUMBER: _ClassVar[int]
    KIND_FIELD_NUMBER: _ClassVar[int]
    STATUS_FIELD_NUMBER: _ClassVar[int]
    SKILL_ID_FIELD_NUMBER: _ClassVar[int]
    TASK_ID_FIELD_NUMBER: _ClassVar[int]
    RUN_ID_FIELD_NUMBER: _ClassVar[int]
    PROFILE_KEY_FIELD_NUMBER: _ClassVar[int]
    FAILURE_REASON_FIELD_NUMBER: _ClassVar[int]
    CREATED_AT_FIELD_NUMBER: _ClassVar[int]
    UPDATED_AT_FIELD_NUMBER: _ClassVar[int]
    MESSAGES_FIELD_NUMBER: _ClassVar[int]
    PROPOSALS_FIELD_NUMBER: _ClassVar[int]
    ARTIFACTS_FIELD_NUMBER: _ClassVar[int]
    CREATED_BY_FIELD_NUMBER: _ClassVar[int]
    ATTACHMENTS_FIELD_NUMBER: _ClassVar[int]
    PROPOSAL_TARGET_FIELD_NUMBER: _ClassVar[int]
    STARTER_JOB_ID_FIELD_NUMBER: _ClassVar[int]
    id: str
    title: str
    kind: str
    status: str
    skill_id: str
    task_id: str
    run_id: str
    profile_key: str
    failure_reason: str
    created_at: str
    updated_at: str
    messages: _containers.RepeatedCompositeFieldContainer[AgentSessionMessage]
    proposals: _containers.RepeatedCompositeFieldContainer[AgentSessionProposal]
    artifacts: _containers.RepeatedCompositeFieldContainer[_agent_session_pb2.AgentSessionArtifact]
    created_by: _agent_session_pb2.AgentSessionAttribution
    attachments: _containers.RepeatedCompositeFieldContainer[_agent_session_pb2.AgentSessionAttachment]
    proposal_target: AgentSessionProposalTarget
    starter_job_id: str
    def __init__(self, id: _Optional[str] = ..., title: _Optional[str] = ..., kind: _Optional[str] = ..., status: _Optional[str] = ..., skill_id: _Optional[str] = ..., task_id: _Optional[str] = ..., run_id: _Optional[str] = ..., profile_key: _Optional[str] = ..., failure_reason: _Optional[str] = ..., created_at: _Optional[str] = ..., updated_at: _Optional[str] = ..., messages: _Optional[_Iterable[_Union[AgentSessionMessage, _Mapping]]] = ..., proposals: _Optional[_Iterable[_Union[AgentSessionProposal, _Mapping]]] = ..., artifacts: _Optional[_Iterable[_Union[_agent_session_pb2.AgentSessionArtifact, _Mapping]]] = ..., created_by: _Optional[_Union[_agent_session_pb2.AgentSessionAttribution, _Mapping]] = ..., attachments: _Optional[_Iterable[_Union[_agent_session_pb2.AgentSessionAttachment, _Mapping]]] = ..., proposal_target: _Optional[_Union[AgentSessionProposalTarget, _Mapping]] = ..., starter_job_id: _Optional[str] = ...) -> None: ...
