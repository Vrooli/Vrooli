from buf.validate import validate_pb2 as _validate_pb2
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from typing import ClassVar as _ClassVar, Optional as _Optional

DESCRIPTOR: _descriptor.FileDescriptor

class Recommendation(_message.Message):
    __slots__ = ("id", "scenario_name", "type", "description", "status", "priority", "created", "source", "task_id", "run_id", "started_at", "started_by", "auto_approved")
    ID_FIELD_NUMBER: _ClassVar[int]
    SCENARIO_NAME_FIELD_NUMBER: _ClassVar[int]
    TYPE_FIELD_NUMBER: _ClassVar[int]
    DESCRIPTION_FIELD_NUMBER: _ClassVar[int]
    STATUS_FIELD_NUMBER: _ClassVar[int]
    PRIORITY_FIELD_NUMBER: _ClassVar[int]
    CREATED_FIELD_NUMBER: _ClassVar[int]
    SOURCE_FIELD_NUMBER: _ClassVar[int]
    TASK_ID_FIELD_NUMBER: _ClassVar[int]
    RUN_ID_FIELD_NUMBER: _ClassVar[int]
    STARTED_AT_FIELD_NUMBER: _ClassVar[int]
    STARTED_BY_FIELD_NUMBER: _ClassVar[int]
    AUTO_APPROVED_FIELD_NUMBER: _ClassVar[int]
    id: str
    scenario_name: str
    type: str
    description: str
    status: str
    priority: int
    created: str
    source: str
    task_id: str
    run_id: str
    started_at: str
    started_by: str
    auto_approved: bool
    def __init__(self, id: _Optional[str] = ..., scenario_name: _Optional[str] = ..., type: _Optional[str] = ..., description: _Optional[str] = ..., status: _Optional[str] = ..., priority: _Optional[int] = ..., created: _Optional[str] = ..., source: _Optional[str] = ..., task_id: _Optional[str] = ..., run_id: _Optional[str] = ..., started_at: _Optional[str] = ..., started_by: _Optional[str] = ..., auto_approved: _Optional[bool] = ...) -> None: ...
