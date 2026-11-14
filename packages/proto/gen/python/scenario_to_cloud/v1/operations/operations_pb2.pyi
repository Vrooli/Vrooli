from scenario_to_cloud.v1.errors import errors_pb2 as _errors_pb2
from google.protobuf.internal import containers as _containers
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Iterable as _Iterable, Mapping as _Mapping
from typing import ClassVar as _ClassVar, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class StepReceipt(_message.Message):
    __slots__ = ("step", "owner_operation", "outcome", "fence", "replayed", "source", "detail", "error", "started_at", "completed_at")
    STEP_FIELD_NUMBER: _ClassVar[int]
    OWNER_OPERATION_FIELD_NUMBER: _ClassVar[int]
    OUTCOME_FIELD_NUMBER: _ClassVar[int]
    FENCE_FIELD_NUMBER: _ClassVar[int]
    REPLAYED_FIELD_NUMBER: _ClassVar[int]
    SOURCE_FIELD_NUMBER: _ClassVar[int]
    DETAIL_FIELD_NUMBER: _ClassVar[int]
    ERROR_FIELD_NUMBER: _ClassVar[int]
    STARTED_AT_FIELD_NUMBER: _ClassVar[int]
    COMPLETED_AT_FIELD_NUMBER: _ClassVar[int]
    step: str
    owner_operation: str
    outcome: str
    fence: int
    replayed: bool
    source: str
    detail: str
    error: str
    started_at: str
    completed_at: str
    def __init__(self, step: _Optional[str] = ..., owner_operation: _Optional[str] = ..., outcome: _Optional[str] = ..., fence: _Optional[int] = ..., replayed: _Optional[bool] = ..., source: _Optional[str] = ..., detail: _Optional[str] = ..., error: _Optional[str] = ..., started_at: _Optional[str] = ..., completed_at: _Optional[str] = ...) -> None: ...

class UnknownEffect(_message.Message):
    __slots__ = ("step", "fence", "reason", "retry", "next_action", "recorded_at")
    STEP_FIELD_NUMBER: _ClassVar[int]
    FENCE_FIELD_NUMBER: _ClassVar[int]
    REASON_FIELD_NUMBER: _ClassVar[int]
    RETRY_FIELD_NUMBER: _ClassVar[int]
    NEXT_ACTION_FIELD_NUMBER: _ClassVar[int]
    RECORDED_AT_FIELD_NUMBER: _ClassVar[int]
    step: str
    fence: int
    reason: str
    retry: str
    next_action: str
    recorded_at: str
    def __init__(self, step: _Optional[str] = ..., fence: _Optional[int] = ..., reason: _Optional[str] = ..., retry: _Optional[str] = ..., next_action: _Optional[str] = ..., recorded_at: _Optional[str] = ...) -> None: ...

class OperationResult(_message.Message):
    __slots__ = ("outcome", "recovery_outcome", "completed_steps", "message")
    OUTCOME_FIELD_NUMBER: _ClassVar[int]
    RECOVERY_OUTCOME_FIELD_NUMBER: _ClassVar[int]
    COMPLETED_STEPS_FIELD_NUMBER: _ClassVar[int]
    MESSAGE_FIELD_NUMBER: _ClassVar[int]
    outcome: str
    recovery_outcome: str
    completed_steps: int
    message: str
    def __init__(self, outcome: _Optional[str] = ..., recovery_outcome: _Optional[str] = ..., completed_steps: _Optional[int] = ..., message: _Optional[str] = ...) -> None: ...

class OperationStanding(_message.Message):
    __slots__ = ("schema_version", "operation_id", "deployment_id", "request_key", "plan_digest", "state", "terminal", "fence", "worker_id", "lease_expires_at", "heartbeat_at", "cancel_requested", "active_step", "completed_steps", "step_receipts", "unknown_effects", "result", "error", "next_action", "reattach_command", "still_pending", "recommended_next_check_seconds", "created_at", "updated_at", "terminal_at")
    SCHEMA_VERSION_FIELD_NUMBER: _ClassVar[int]
    OPERATION_ID_FIELD_NUMBER: _ClassVar[int]
    DEPLOYMENT_ID_FIELD_NUMBER: _ClassVar[int]
    REQUEST_KEY_FIELD_NUMBER: _ClassVar[int]
    PLAN_DIGEST_FIELD_NUMBER: _ClassVar[int]
    STATE_FIELD_NUMBER: _ClassVar[int]
    TERMINAL_FIELD_NUMBER: _ClassVar[int]
    FENCE_FIELD_NUMBER: _ClassVar[int]
    WORKER_ID_FIELD_NUMBER: _ClassVar[int]
    LEASE_EXPIRES_AT_FIELD_NUMBER: _ClassVar[int]
    HEARTBEAT_AT_FIELD_NUMBER: _ClassVar[int]
    CANCEL_REQUESTED_FIELD_NUMBER: _ClassVar[int]
    ACTIVE_STEP_FIELD_NUMBER: _ClassVar[int]
    COMPLETED_STEPS_FIELD_NUMBER: _ClassVar[int]
    STEP_RECEIPTS_FIELD_NUMBER: _ClassVar[int]
    UNKNOWN_EFFECTS_FIELD_NUMBER: _ClassVar[int]
    RESULT_FIELD_NUMBER: _ClassVar[int]
    ERROR_FIELD_NUMBER: _ClassVar[int]
    NEXT_ACTION_FIELD_NUMBER: _ClassVar[int]
    REATTACH_COMMAND_FIELD_NUMBER: _ClassVar[int]
    STILL_PENDING_FIELD_NUMBER: _ClassVar[int]
    RECOMMENDED_NEXT_CHECK_SECONDS_FIELD_NUMBER: _ClassVar[int]
    CREATED_AT_FIELD_NUMBER: _ClassVar[int]
    UPDATED_AT_FIELD_NUMBER: _ClassVar[int]
    TERMINAL_AT_FIELD_NUMBER: _ClassVar[int]
    schema_version: str
    operation_id: str
    deployment_id: str
    request_key: str
    plan_digest: str
    state: str
    terminal: bool
    fence: int
    worker_id: str
    lease_expires_at: str
    heartbeat_at: str
    cancel_requested: bool
    active_step: str
    completed_steps: _containers.RepeatedScalarFieldContainer[str]
    step_receipts: _containers.RepeatedCompositeFieldContainer[StepReceipt]
    unknown_effects: _containers.RepeatedCompositeFieldContainer[UnknownEffect]
    result: OperationResult
    error: _errors_pb2.Error
    next_action: _errors_pb2.NextAction
    reattach_command: str
    still_pending: bool
    recommended_next_check_seconds: int
    created_at: str
    updated_at: str
    terminal_at: str
    def __init__(self, schema_version: _Optional[str] = ..., operation_id: _Optional[str] = ..., deployment_id: _Optional[str] = ..., request_key: _Optional[str] = ..., plan_digest: _Optional[str] = ..., state: _Optional[str] = ..., terminal: _Optional[bool] = ..., fence: _Optional[int] = ..., worker_id: _Optional[str] = ..., lease_expires_at: _Optional[str] = ..., heartbeat_at: _Optional[str] = ..., cancel_requested: _Optional[bool] = ..., active_step: _Optional[str] = ..., completed_steps: _Optional[_Iterable[str]] = ..., step_receipts: _Optional[_Iterable[_Union[StepReceipt, _Mapping]]] = ..., unknown_effects: _Optional[_Iterable[_Union[UnknownEffect, _Mapping]]] = ..., result: _Optional[_Union[OperationResult, _Mapping]] = ..., error: _Optional[_Union[_errors_pb2.Error, _Mapping]] = ..., next_action: _Optional[_Union[_errors_pb2.NextAction, _Mapping]] = ..., reattach_command: _Optional[str] = ..., still_pending: _Optional[bool] = ..., recommended_next_check_seconds: _Optional[int] = ..., created_at: _Optional[str] = ..., updated_at: _Optional[str] = ..., terminal_at: _Optional[str] = ...) -> None: ...

class GetOperationRequest(_message.Message):
    __slots__ = ("operation_id",)
    OPERATION_ID_FIELD_NUMBER: _ClassVar[int]
    operation_id: str
    def __init__(self, operation_id: _Optional[str] = ...) -> None: ...

class WaitOperationRequest(_message.Message):
    __slots__ = ("operation_id", "timeout_seconds")
    OPERATION_ID_FIELD_NUMBER: _ClassVar[int]
    TIMEOUT_SECONDS_FIELD_NUMBER: _ClassVar[int]
    operation_id: str
    timeout_seconds: int
    def __init__(self, operation_id: _Optional[str] = ..., timeout_seconds: _Optional[int] = ...) -> None: ...

class CancelOperationRequest(_message.Message):
    __slots__ = ("operation_id",)
    OPERATION_ID_FIELD_NUMBER: _ClassVar[int]
    operation_id: str
    def __init__(self, operation_id: _Optional[str] = ...) -> None: ...

class ListDeploymentOperationsRequest(_message.Message):
    __slots__ = ("deployment_id",)
    DEPLOYMENT_ID_FIELD_NUMBER: _ClassVar[int]
    deployment_id: str
    def __init__(self, deployment_id: _Optional[str] = ...) -> None: ...

class ListDeploymentOperationsResponse(_message.Message):
    __slots__ = ("schema_version", "deployment_id", "operations")
    SCHEMA_VERSION_FIELD_NUMBER: _ClassVar[int]
    DEPLOYMENT_ID_FIELD_NUMBER: _ClassVar[int]
    OPERATIONS_FIELD_NUMBER: _ClassVar[int]
    schema_version: str
    deployment_id: str
    operations: _containers.RepeatedCompositeFieldContainer[OperationStanding]
    def __init__(self, schema_version: _Optional[str] = ..., deployment_id: _Optional[str] = ..., operations: _Optional[_Iterable[_Union[OperationStanding, _Mapping]]] = ...) -> None: ...

class ReconcileOperationsRequest(_message.Message):
    __slots__ = ()
    def __init__(self) -> None: ...

class ReconcileOperationsResponse(_message.Message):
    __slots__ = ("schema_version", "worker_id", "acquired")
    SCHEMA_VERSION_FIELD_NUMBER: _ClassVar[int]
    WORKER_ID_FIELD_NUMBER: _ClassVar[int]
    ACQUIRED_FIELD_NUMBER: _ClassVar[int]
    schema_version: str
    worker_id: str
    acquired: _containers.RepeatedScalarFieldContainer[str]
    def __init__(self, schema_version: _Optional[str] = ..., worker_id: _Optional[str] = ..., acquired: _Optional[_Iterable[str]] = ...) -> None: ...
