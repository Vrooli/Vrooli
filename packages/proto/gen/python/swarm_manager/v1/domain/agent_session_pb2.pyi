from buf.validate import validate_pb2 as _validate_pb2
from google.protobuf.internal import containers as _containers
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Iterable as _Iterable, Mapping as _Mapping
from typing import ClassVar as _ClassVar, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class AgentSessionAttribution(_message.Message):
    __slots__ = ("type", "run_id", "task_id", "profile_key", "session_id", "session_kind", "source")
    TYPE_FIELD_NUMBER: _ClassVar[int]
    RUN_ID_FIELD_NUMBER: _ClassVar[int]
    TASK_ID_FIELD_NUMBER: _ClassVar[int]
    PROFILE_KEY_FIELD_NUMBER: _ClassVar[int]
    SESSION_ID_FIELD_NUMBER: _ClassVar[int]
    SESSION_KIND_FIELD_NUMBER: _ClassVar[int]
    SOURCE_FIELD_NUMBER: _ClassVar[int]
    type: str
    run_id: str
    task_id: str
    profile_key: str
    session_id: str
    session_kind: str
    source: str
    def __init__(self, type: _Optional[str] = ..., run_id: _Optional[str] = ..., task_id: _Optional[str] = ..., profile_key: _Optional[str] = ..., session_id: _Optional[str] = ..., session_kind: _Optional[str] = ..., source: _Optional[str] = ...) -> None: ...

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
    context: _containers.RepeatedCompositeFieldContainer[AgentSessionContextItem]
    def __init__(self, id: _Optional[str] = ..., role: _Optional[str] = ..., content: _Optional[str] = ..., created_at: _Optional[str] = ..., attachment_ids: _Optional[_Iterable[str]] = ..., context: _Optional[_Iterable[_Union[AgentSessionContextItem, _Mapping]]] = ...) -> None: ...

class AgentSessionContextItem(_message.Message):
    __slots__ = ("type", "ref", "title", "summary", "node_id", "metadata_json", "selected_at")
    TYPE_FIELD_NUMBER: _ClassVar[int]
    REF_FIELD_NUMBER: _ClassVar[int]
    TITLE_FIELD_NUMBER: _ClassVar[int]
    SUMMARY_FIELD_NUMBER: _ClassVar[int]
    NODE_ID_FIELD_NUMBER: _ClassVar[int]
    METADATA_JSON_FIELD_NUMBER: _ClassVar[int]
    SELECTED_AT_FIELD_NUMBER: _ClassVar[int]
    type: str
    ref: str
    title: str
    summary: str
    node_id: str
    metadata_json: str
    selected_at: str
    def __init__(self, type: _Optional[str] = ..., ref: _Optional[str] = ..., title: _Optional[str] = ..., summary: _Optional[str] = ..., node_id: _Optional[str] = ..., metadata_json: _Optional[str] = ..., selected_at: _Optional[str] = ...) -> None: ...

class AgentSessionAttachment(_message.Message):
    __slots__ = ("id", "filename", "content_type", "size_bytes", "created_at")
    ID_FIELD_NUMBER: _ClassVar[int]
    FILENAME_FIELD_NUMBER: _ClassVar[int]
    CONTENT_TYPE_FIELD_NUMBER: _ClassVar[int]
    SIZE_BYTES_FIELD_NUMBER: _ClassVar[int]
    CREATED_AT_FIELD_NUMBER: _ClassVar[int]
    id: str
    filename: str
    content_type: str
    size_bytes: int
    created_at: str
    def __init__(self, id: _Optional[str] = ..., filename: _Optional[str] = ..., content_type: _Optional[str] = ..., size_bytes: _Optional[int] = ..., created_at: _Optional[str] = ...) -> None: ...

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
    attribution: AgentSessionAttribution
    def __init__(self, id: _Optional[str] = ..., kind: _Optional[str] = ..., status: _Optional[str] = ..., summary: _Optional[str] = ..., payload_json: _Optional[str] = ..., created_at: _Optional[str] = ..., updated_at: _Optional[str] = ..., attribution: _Optional[_Union[AgentSessionAttribution, _Mapping]] = ...) -> None: ...

class AgentSessionArtifact(_message.Message):
    __slots__ = ("id", "session_id", "artifact_type", "action", "entity_ref", "title", "proposal_id", "activity_id", "run_id", "mutation_source", "attribution", "created_at")
    ID_FIELD_NUMBER: _ClassVar[int]
    SESSION_ID_FIELD_NUMBER: _ClassVar[int]
    ARTIFACT_TYPE_FIELD_NUMBER: _ClassVar[int]
    ACTION_FIELD_NUMBER: _ClassVar[int]
    ENTITY_REF_FIELD_NUMBER: _ClassVar[int]
    TITLE_FIELD_NUMBER: _ClassVar[int]
    PROPOSAL_ID_FIELD_NUMBER: _ClassVar[int]
    ACTIVITY_ID_FIELD_NUMBER: _ClassVar[int]
    RUN_ID_FIELD_NUMBER: _ClassVar[int]
    MUTATION_SOURCE_FIELD_NUMBER: _ClassVar[int]
    ATTRIBUTION_FIELD_NUMBER: _ClassVar[int]
    CREATED_AT_FIELD_NUMBER: _ClassVar[int]
    id: str
    session_id: str
    artifact_type: str
    action: str
    entity_ref: str
    title: str
    proposal_id: str
    activity_id: str
    run_id: str
    mutation_source: str
    attribution: AgentSessionAttribution
    created_at: str
    def __init__(self, id: _Optional[str] = ..., session_id: _Optional[str] = ..., artifact_type: _Optional[str] = ..., action: _Optional[str] = ..., entity_ref: _Optional[str] = ..., title: _Optional[str] = ..., proposal_id: _Optional[str] = ..., activity_id: _Optional[str] = ..., run_id: _Optional[str] = ..., mutation_source: _Optional[str] = ..., attribution: _Optional[_Union[AgentSessionAttribution, _Mapping]] = ..., created_at: _Optional[str] = ...) -> None: ...

class AgentSession(_message.Message):
    __slots__ = ("id", "title", "kind", "status", "skill_id", "task_id", "run_id", "profile_key", "failure_reason", "created_at", "updated_at", "messages", "proposals", "artifacts", "created_by", "attachments")
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
    artifacts: _containers.RepeatedCompositeFieldContainer[AgentSessionArtifact]
    created_by: AgentSessionAttribution
    attachments: _containers.RepeatedCompositeFieldContainer[AgentSessionAttachment]
    def __init__(self, id: _Optional[str] = ..., title: _Optional[str] = ..., kind: _Optional[str] = ..., status: _Optional[str] = ..., skill_id: _Optional[str] = ..., task_id: _Optional[str] = ..., run_id: _Optional[str] = ..., profile_key: _Optional[str] = ..., failure_reason: _Optional[str] = ..., created_at: _Optional[str] = ..., updated_at: _Optional[str] = ..., messages: _Optional[_Iterable[_Union[AgentSessionMessage, _Mapping]]] = ..., proposals: _Optional[_Iterable[_Union[AgentSessionProposal, _Mapping]]] = ..., artifacts: _Optional[_Iterable[_Union[AgentSessionArtifact, _Mapping]]] = ..., created_by: _Optional[_Union[AgentSessionAttribution, _Mapping]] = ..., attachments: _Optional[_Iterable[_Union[AgentSessionAttachment, _Mapping]]] = ...) -> None: ...
