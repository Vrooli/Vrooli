from buf.validate import validate_pb2 as _validate_pb2
from google.protobuf.internal import containers as _containers
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Mapping as _Mapping
from typing import ClassVar as _ClassVar, Optional as _Optional

DESCRIPTOR: _descriptor.FileDescriptor

class AgentActivity(_message.Message):
    __slots__ = ("activity_id", "owner_type", "owner_kind", "owner_name", "owner_title", "execution_id", "purpose", "interaction_type", "task_id", "run_id", "status", "requested_at", "started_at", "finished_at", "failure_reason", "requested_by", "metadata", "updated_at")
    class MetadataEntry(_message.Message):
        __slots__ = ("key", "value")
        KEY_FIELD_NUMBER: _ClassVar[int]
        VALUE_FIELD_NUMBER: _ClassVar[int]
        key: str
        value: str
        def __init__(self, key: _Optional[str] = ..., value: _Optional[str] = ...) -> None: ...
    ACTIVITY_ID_FIELD_NUMBER: _ClassVar[int]
    OWNER_TYPE_FIELD_NUMBER: _ClassVar[int]
    OWNER_KIND_FIELD_NUMBER: _ClassVar[int]
    OWNER_NAME_FIELD_NUMBER: _ClassVar[int]
    OWNER_TITLE_FIELD_NUMBER: _ClassVar[int]
    EXECUTION_ID_FIELD_NUMBER: _ClassVar[int]
    PURPOSE_FIELD_NUMBER: _ClassVar[int]
    INTERACTION_TYPE_FIELD_NUMBER: _ClassVar[int]
    TASK_ID_FIELD_NUMBER: _ClassVar[int]
    RUN_ID_FIELD_NUMBER: _ClassVar[int]
    STATUS_FIELD_NUMBER: _ClassVar[int]
    REQUESTED_AT_FIELD_NUMBER: _ClassVar[int]
    STARTED_AT_FIELD_NUMBER: _ClassVar[int]
    FINISHED_AT_FIELD_NUMBER: _ClassVar[int]
    FAILURE_REASON_FIELD_NUMBER: _ClassVar[int]
    REQUESTED_BY_FIELD_NUMBER: _ClassVar[int]
    METADATA_FIELD_NUMBER: _ClassVar[int]
    UPDATED_AT_FIELD_NUMBER: _ClassVar[int]
    activity_id: str
    owner_type: str
    owner_kind: str
    owner_name: str
    owner_title: str
    execution_id: str
    purpose: str
    interaction_type: str
    task_id: str
    run_id: str
    status: str
    requested_at: str
    started_at: str
    finished_at: str
    failure_reason: str
    requested_by: str
    metadata: _containers.ScalarMap[str, str]
    updated_at: str
    def __init__(self, activity_id: _Optional[str] = ..., owner_type: _Optional[str] = ..., owner_kind: _Optional[str] = ..., owner_name: _Optional[str] = ..., owner_title: _Optional[str] = ..., execution_id: _Optional[str] = ..., purpose: _Optional[str] = ..., interaction_type: _Optional[str] = ..., task_id: _Optional[str] = ..., run_id: _Optional[str] = ..., status: _Optional[str] = ..., requested_at: _Optional[str] = ..., started_at: _Optional[str] = ..., finished_at: _Optional[str] = ..., failure_reason: _Optional[str] = ..., requested_by: _Optional[str] = ..., metadata: _Optional[_Mapping[str, str]] = ..., updated_at: _Optional[str] = ...) -> None: ...
