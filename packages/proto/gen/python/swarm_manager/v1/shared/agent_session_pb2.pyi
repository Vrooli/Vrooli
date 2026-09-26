from buf.validate import validate_pb2 as _validate_pb2
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Mapping as _Mapping
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
