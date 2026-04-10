import datetime

from agent_manager.v1.domain import types_pb2 as _types_pb2
from buf.validate import validate_pb2 as _validate_pb2
from google.protobuf import timestamp_pb2 as _timestamp_pb2
from google.protobuf.internal import containers as _containers
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Iterable as _Iterable, Mapping as _Mapping
from typing import ClassVar as _ClassVar, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class Task(_message.Message):
    __slots__ = ("id", "title", "description", "scope_path", "project_root", "phase_prompt_ids", "context_attachments", "status", "created_by", "created_at", "updated_at")
    ID_FIELD_NUMBER: _ClassVar[int]
    TITLE_FIELD_NUMBER: _ClassVar[int]
    DESCRIPTION_FIELD_NUMBER: _ClassVar[int]
    SCOPE_PATH_FIELD_NUMBER: _ClassVar[int]
    PROJECT_ROOT_FIELD_NUMBER: _ClassVar[int]
    PHASE_PROMPT_IDS_FIELD_NUMBER: _ClassVar[int]
    CONTEXT_ATTACHMENTS_FIELD_NUMBER: _ClassVar[int]
    STATUS_FIELD_NUMBER: _ClassVar[int]
    CREATED_BY_FIELD_NUMBER: _ClassVar[int]
    CREATED_AT_FIELD_NUMBER: _ClassVar[int]
    UPDATED_AT_FIELD_NUMBER: _ClassVar[int]
    id: str
    title: str
    description: str
    scope_path: str
    project_root: str
    phase_prompt_ids: _containers.RepeatedScalarFieldContainer[str]
    context_attachments: _containers.RepeatedCompositeFieldContainer[ContextAttachment]
    status: _types_pb2.TaskStatus
    created_by: str
    created_at: _timestamp_pb2.Timestamp
    updated_at: _timestamp_pb2.Timestamp
    def __init__(self, id: _Optional[str] = ..., title: _Optional[str] = ..., description: _Optional[str] = ..., scope_path: _Optional[str] = ..., project_root: _Optional[str] = ..., phase_prompt_ids: _Optional[_Iterable[str]] = ..., context_attachments: _Optional[_Iterable[_Union[ContextAttachment, _Mapping]]] = ..., status: _Optional[_Union[_types_pb2.TaskStatus, str]] = ..., created_by: _Optional[str] = ..., created_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., updated_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ...) -> None: ...

class ContextAttachment(_message.Message):
    __slots__ = ("type", "path", "url", "content", "label", "key", "tags", "summary", "format", "priority", "attachment_id")
    TYPE_FIELD_NUMBER: _ClassVar[int]
    PATH_FIELD_NUMBER: _ClassVar[int]
    URL_FIELD_NUMBER: _ClassVar[int]
    CONTENT_FIELD_NUMBER: _ClassVar[int]
    LABEL_FIELD_NUMBER: _ClassVar[int]
    KEY_FIELD_NUMBER: _ClassVar[int]
    TAGS_FIELD_NUMBER: _ClassVar[int]
    SUMMARY_FIELD_NUMBER: _ClassVar[int]
    FORMAT_FIELD_NUMBER: _ClassVar[int]
    PRIORITY_FIELD_NUMBER: _ClassVar[int]
    ATTACHMENT_ID_FIELD_NUMBER: _ClassVar[int]
    type: str
    path: str
    url: str
    content: str
    label: str
    key: str
    tags: _containers.RepeatedScalarFieldContainer[str]
    summary: str
    format: str
    priority: str
    attachment_id: str
    def __init__(self, type: _Optional[str] = ..., path: _Optional[str] = ..., url: _Optional[str] = ..., content: _Optional[str] = ..., label: _Optional[str] = ..., key: _Optional[str] = ..., tags: _Optional[_Iterable[str]] = ..., summary: _Optional[str] = ..., format: _Optional[str] = ..., priority: _Optional[str] = ..., attachment_id: _Optional[str] = ...) -> None: ...

class ScopeLock(_message.Message):
    __slots__ = ("id", "run_id", "scope_path", "project_root", "acquired_at", "expires_at")
    ID_FIELD_NUMBER: _ClassVar[int]
    RUN_ID_FIELD_NUMBER: _ClassVar[int]
    SCOPE_PATH_FIELD_NUMBER: _ClassVar[int]
    PROJECT_ROOT_FIELD_NUMBER: _ClassVar[int]
    ACQUIRED_AT_FIELD_NUMBER: _ClassVar[int]
    EXPIRES_AT_FIELD_NUMBER: _ClassVar[int]
    id: str
    run_id: str
    scope_path: str
    project_root: str
    acquired_at: _timestamp_pb2.Timestamp
    expires_at: _timestamp_pb2.Timestamp
    def __init__(self, id: _Optional[str] = ..., run_id: _Optional[str] = ..., scope_path: _Optional[str] = ..., project_root: _Optional[str] = ..., acquired_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., expires_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ...) -> None: ...

class Policy(_message.Message):
    __slots__ = ("id", "name", "description", "priority", "scope_pattern", "rules", "created_by", "created_at", "updated_at", "enabled")
    ID_FIELD_NUMBER: _ClassVar[int]
    NAME_FIELD_NUMBER: _ClassVar[int]
    DESCRIPTION_FIELD_NUMBER: _ClassVar[int]
    PRIORITY_FIELD_NUMBER: _ClassVar[int]
    SCOPE_PATTERN_FIELD_NUMBER: _ClassVar[int]
    RULES_FIELD_NUMBER: _ClassVar[int]
    CREATED_BY_FIELD_NUMBER: _ClassVar[int]
    CREATED_AT_FIELD_NUMBER: _ClassVar[int]
    UPDATED_AT_FIELD_NUMBER: _ClassVar[int]
    ENABLED_FIELD_NUMBER: _ClassVar[int]
    id: str
    name: str
    description: str
    priority: int
    scope_pattern: str
    rules: PolicyRules
    created_by: str
    created_at: _timestamp_pb2.Timestamp
    updated_at: _timestamp_pb2.Timestamp
    enabled: bool
    def __init__(self, id: _Optional[str] = ..., name: _Optional[str] = ..., description: _Optional[str] = ..., priority: _Optional[int] = ..., scope_pattern: _Optional[str] = ..., rules: _Optional[_Union[PolicyRules, _Mapping]] = ..., created_by: _Optional[str] = ..., created_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., updated_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., enabled: _Optional[bool] = ...) -> None: ...

class PolicyRules(_message.Message):
    __slots__ = ("require_sandbox", "allow_in_place", "in_place_requires_approval", "require_approval", "auto_approve_patterns", "max_concurrent_runs", "max_concurrent_per_scope", "max_files_changed", "max_total_size_bytes", "max_execution_time_ms", "allowed_runners", "denied_runners")
    REQUIRE_SANDBOX_FIELD_NUMBER: _ClassVar[int]
    ALLOW_IN_PLACE_FIELD_NUMBER: _ClassVar[int]
    IN_PLACE_REQUIRES_APPROVAL_FIELD_NUMBER: _ClassVar[int]
    REQUIRE_APPROVAL_FIELD_NUMBER: _ClassVar[int]
    AUTO_APPROVE_PATTERNS_FIELD_NUMBER: _ClassVar[int]
    MAX_CONCURRENT_RUNS_FIELD_NUMBER: _ClassVar[int]
    MAX_CONCURRENT_PER_SCOPE_FIELD_NUMBER: _ClassVar[int]
    MAX_FILES_CHANGED_FIELD_NUMBER: _ClassVar[int]
    MAX_TOTAL_SIZE_BYTES_FIELD_NUMBER: _ClassVar[int]
    MAX_EXECUTION_TIME_MS_FIELD_NUMBER: _ClassVar[int]
    ALLOWED_RUNNERS_FIELD_NUMBER: _ClassVar[int]
    DENIED_RUNNERS_FIELD_NUMBER: _ClassVar[int]
    require_sandbox: bool
    allow_in_place: bool
    in_place_requires_approval: bool
    require_approval: bool
    auto_approve_patterns: _containers.RepeatedScalarFieldContainer[str]
    max_concurrent_runs: int
    max_concurrent_per_scope: int
    max_files_changed: int
    max_total_size_bytes: int
    max_execution_time_ms: int
    allowed_runners: _containers.RepeatedScalarFieldContainer[_types_pb2.RunnerType]
    denied_runners: _containers.RepeatedScalarFieldContainer[_types_pb2.RunnerType]
    def __init__(self, require_sandbox: _Optional[bool] = ..., allow_in_place: _Optional[bool] = ..., in_place_requires_approval: _Optional[bool] = ..., require_approval: _Optional[bool] = ..., auto_approve_patterns: _Optional[_Iterable[str]] = ..., max_concurrent_runs: _Optional[int] = ..., max_concurrent_per_scope: _Optional[int] = ..., max_files_changed: _Optional[int] = ..., max_total_size_bytes: _Optional[int] = ..., max_execution_time_ms: _Optional[int] = ..., allowed_runners: _Optional[_Iterable[_Union[_types_pb2.RunnerType, str]]] = ..., denied_runners: _Optional[_Iterable[_Union[_types_pb2.RunnerType, str]]] = ...) -> None: ...
