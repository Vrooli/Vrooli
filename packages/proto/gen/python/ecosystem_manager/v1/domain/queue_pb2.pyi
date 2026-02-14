from buf.validate import validate_pb2 as _validate_pb2
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from typing import ClassVar as _ClassVar, Optional as _Optional

DESCRIPTOR: _descriptor.FileDescriptor

class ExecutionRecord(_message.Message):
    __slots__ = ("task_id", "task_title", "task_type", "task_operation", "execution_id", "agent_tag", "process_id", "start_time", "end_time", "duration", "success", "exit_reason", "prompt_size", "prompt_path", "output_path", "clean_output_path", "last_message_path", "transcript_path", "auto_steer_profile_id", "auto_steer_iteration", "steer_mode", "steer_phase_index", "steer_phase_iteration", "steering_source", "steering_queue_total", "timeout_allowed", "rate_limited", "retry_after")
    TASK_ID_FIELD_NUMBER: _ClassVar[int]
    TASK_TITLE_FIELD_NUMBER: _ClassVar[int]
    TASK_TYPE_FIELD_NUMBER: _ClassVar[int]
    TASK_OPERATION_FIELD_NUMBER: _ClassVar[int]
    EXECUTION_ID_FIELD_NUMBER: _ClassVar[int]
    AGENT_TAG_FIELD_NUMBER: _ClassVar[int]
    PROCESS_ID_FIELD_NUMBER: _ClassVar[int]
    START_TIME_FIELD_NUMBER: _ClassVar[int]
    END_TIME_FIELD_NUMBER: _ClassVar[int]
    DURATION_FIELD_NUMBER: _ClassVar[int]
    SUCCESS_FIELD_NUMBER: _ClassVar[int]
    EXIT_REASON_FIELD_NUMBER: _ClassVar[int]
    PROMPT_SIZE_FIELD_NUMBER: _ClassVar[int]
    PROMPT_PATH_FIELD_NUMBER: _ClassVar[int]
    OUTPUT_PATH_FIELD_NUMBER: _ClassVar[int]
    CLEAN_OUTPUT_PATH_FIELD_NUMBER: _ClassVar[int]
    LAST_MESSAGE_PATH_FIELD_NUMBER: _ClassVar[int]
    TRANSCRIPT_PATH_FIELD_NUMBER: _ClassVar[int]
    AUTO_STEER_PROFILE_ID_FIELD_NUMBER: _ClassVar[int]
    AUTO_STEER_ITERATION_FIELD_NUMBER: _ClassVar[int]
    STEER_MODE_FIELD_NUMBER: _ClassVar[int]
    STEER_PHASE_INDEX_FIELD_NUMBER: _ClassVar[int]
    STEER_PHASE_ITERATION_FIELD_NUMBER: _ClassVar[int]
    STEERING_SOURCE_FIELD_NUMBER: _ClassVar[int]
    STEERING_QUEUE_TOTAL_FIELD_NUMBER: _ClassVar[int]
    TIMEOUT_ALLOWED_FIELD_NUMBER: _ClassVar[int]
    RATE_LIMITED_FIELD_NUMBER: _ClassVar[int]
    RETRY_AFTER_FIELD_NUMBER: _ClassVar[int]
    task_id: str
    task_title: str
    task_type: str
    task_operation: str
    execution_id: str
    agent_tag: str
    process_id: int
    start_time: str
    end_time: str
    duration: str
    success: bool
    exit_reason: str
    prompt_size: int
    prompt_path: str
    output_path: str
    clean_output_path: str
    last_message_path: str
    transcript_path: str
    auto_steer_profile_id: str
    auto_steer_iteration: int
    steer_mode: str
    steer_phase_index: int
    steer_phase_iteration: int
    steering_source: str
    steering_queue_total: int
    timeout_allowed: str
    rate_limited: bool
    retry_after: int
    def __init__(self, task_id: _Optional[str] = ..., task_title: _Optional[str] = ..., task_type: _Optional[str] = ..., task_operation: _Optional[str] = ..., execution_id: _Optional[str] = ..., agent_tag: _Optional[str] = ..., process_id: _Optional[int] = ..., start_time: _Optional[str] = ..., end_time: _Optional[str] = ..., duration: _Optional[str] = ..., success: _Optional[bool] = ..., exit_reason: _Optional[str] = ..., prompt_size: _Optional[int] = ..., prompt_path: _Optional[str] = ..., output_path: _Optional[str] = ..., clean_output_path: _Optional[str] = ..., last_message_path: _Optional[str] = ..., transcript_path: _Optional[str] = ..., auto_steer_profile_id: _Optional[str] = ..., auto_steer_iteration: _Optional[int] = ..., steer_mode: _Optional[str] = ..., steer_phase_index: _Optional[int] = ..., steer_phase_iteration: _Optional[int] = ..., steering_source: _Optional[str] = ..., steering_queue_total: _Optional[int] = ..., timeout_allowed: _Optional[str] = ..., rate_limited: _Optional[bool] = ..., retry_after: _Optional[int] = ...) -> None: ...

class QueueStatus(_message.Message):
    __slots__ = ("is_active", "is_paused", "is_rate_limit_paused", "rate_limit_resume_at", "pending_count", "in_progress_count", "running_processes", "available_slots", "max_slots", "last_processed_at", "cooldown_seconds", "task_timeout_minutes")
    IS_ACTIVE_FIELD_NUMBER: _ClassVar[int]
    IS_PAUSED_FIELD_NUMBER: _ClassVar[int]
    IS_RATE_LIMIT_PAUSED_FIELD_NUMBER: _ClassVar[int]
    RATE_LIMIT_RESUME_AT_FIELD_NUMBER: _ClassVar[int]
    PENDING_COUNT_FIELD_NUMBER: _ClassVar[int]
    IN_PROGRESS_COUNT_FIELD_NUMBER: _ClassVar[int]
    RUNNING_PROCESSES_FIELD_NUMBER: _ClassVar[int]
    AVAILABLE_SLOTS_FIELD_NUMBER: _ClassVar[int]
    MAX_SLOTS_FIELD_NUMBER: _ClassVar[int]
    LAST_PROCESSED_AT_FIELD_NUMBER: _ClassVar[int]
    COOLDOWN_SECONDS_FIELD_NUMBER: _ClassVar[int]
    TASK_TIMEOUT_MINUTES_FIELD_NUMBER: _ClassVar[int]
    is_active: bool
    is_paused: bool
    is_rate_limit_paused: bool
    rate_limit_resume_at: str
    pending_count: int
    in_progress_count: int
    running_processes: int
    available_slots: int
    max_slots: int
    last_processed_at: str
    cooldown_seconds: int
    task_timeout_minutes: int
    def __init__(self, is_active: _Optional[bool] = ..., is_paused: _Optional[bool] = ..., is_rate_limit_paused: _Optional[bool] = ..., rate_limit_resume_at: _Optional[str] = ..., pending_count: _Optional[int] = ..., in_progress_count: _Optional[int] = ..., running_processes: _Optional[int] = ..., available_slots: _Optional[int] = ..., max_slots: _Optional[int] = ..., last_processed_at: _Optional[str] = ..., cooldown_seconds: _Optional[int] = ..., task_timeout_minutes: _Optional[int] = ...) -> None: ...
