from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from typing import ClassVar as _ClassVar, Optional as _Optional

DESCRIPTOR: _descriptor.FileDescriptor

class ExecutionMetrics(_message.Message):
    __slots__ = ("id", "execution_id", "step_count", "failed_steps", "total_retries", "average_step_duration_ms", "overall_friction_score", "computed_at", "created_at")
    ID_FIELD_NUMBER: _ClassVar[int]
    EXECUTION_ID_FIELD_NUMBER: _ClassVar[int]
    STEP_COUNT_FIELD_NUMBER: _ClassVar[int]
    FAILED_STEPS_FIELD_NUMBER: _ClassVar[int]
    TOTAL_RETRIES_FIELD_NUMBER: _ClassVar[int]
    AVERAGE_STEP_DURATION_MS_FIELD_NUMBER: _ClassVar[int]
    OVERALL_FRICTION_SCORE_FIELD_NUMBER: _ClassVar[int]
    COMPUTED_AT_FIELD_NUMBER: _ClassVar[int]
    CREATED_AT_FIELD_NUMBER: _ClassVar[int]
    id: str
    execution_id: str
    step_count: int
    failed_steps: int
    total_retries: int
    average_step_duration_ms: float
    overall_friction_score: float
    computed_at: str
    created_at: str
    def __init__(self, id: _Optional[str] = ..., execution_id: _Optional[str] = ..., step_count: _Optional[int] = ..., failed_steps: _Optional[int] = ..., total_retries: _Optional[int] = ..., average_step_duration_ms: _Optional[float] = ..., overall_friction_score: _Optional[float] = ..., computed_at: _Optional[str] = ..., created_at: _Optional[str] = ...) -> None: ...
