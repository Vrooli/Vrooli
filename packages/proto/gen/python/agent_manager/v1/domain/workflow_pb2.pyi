import datetime

from agent_manager.v1.domain import run_pb2 as _run_pb2
from google.protobuf import struct_pb2 as _struct_pb2
from google.protobuf import timestamp_pb2 as _timestamp_pb2
from google.protobuf.internal import containers as _containers
from google.protobuf.internal import enum_type_wrapper as _enum_type_wrapper
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Mapping as _Mapping
from typing import ClassVar as _ClassVar, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class WorkflowExecutionStatus(int, metaclass=_enum_type_wrapper.EnumTypeWrapper):
    __slots__ = ()
    WORKFLOW_EXECUTION_STATUS_UNSPECIFIED: _ClassVar[WorkflowExecutionStatus]
    WORKFLOW_EXECUTION_STATUS_PENDING: _ClassVar[WorkflowExecutionStatus]
    WORKFLOW_EXECUTION_STATUS_RUNNING: _ClassVar[WorkflowExecutionStatus]
    WORKFLOW_EXECUTION_STATUS_WAITING: _ClassVar[WorkflowExecutionStatus]
    WORKFLOW_EXECUTION_STATUS_SUCCEEDED: _ClassVar[WorkflowExecutionStatus]
    WORKFLOW_EXECUTION_STATUS_BLOCKED: _ClassVar[WorkflowExecutionStatus]
    WORKFLOW_EXECUTION_STATUS_ABSTAINED: _ClassVar[WorkflowExecutionStatus]
    WORKFLOW_EXECUTION_STATUS_BUDGET_EXHAUSTED: _ClassVar[WorkflowExecutionStatus]
    WORKFLOW_EXECUTION_STATUS_FAILED: _ClassVar[WorkflowExecutionStatus]
    WORKFLOW_EXECUTION_STATUS_CANCELLED: _ClassVar[WorkflowExecutionStatus]
    WORKFLOW_EXECUTION_STATUS_CANCELLING: _ClassVar[WorkflowExecutionStatus]
WORKFLOW_EXECUTION_STATUS_UNSPECIFIED: WorkflowExecutionStatus
WORKFLOW_EXECUTION_STATUS_PENDING: WorkflowExecutionStatus
WORKFLOW_EXECUTION_STATUS_RUNNING: WorkflowExecutionStatus
WORKFLOW_EXECUTION_STATUS_WAITING: WorkflowExecutionStatus
WORKFLOW_EXECUTION_STATUS_SUCCEEDED: WorkflowExecutionStatus
WORKFLOW_EXECUTION_STATUS_BLOCKED: WorkflowExecutionStatus
WORKFLOW_EXECUTION_STATUS_ABSTAINED: WorkflowExecutionStatus
WORKFLOW_EXECUTION_STATUS_BUDGET_EXHAUSTED: WorkflowExecutionStatus
WORKFLOW_EXECUTION_STATUS_FAILED: WorkflowExecutionStatus
WORKFLOW_EXECUTION_STATUS_CANCELLED: WorkflowExecutionStatus
WORKFLOW_EXECUTION_STATUS_CANCELLING: WorkflowExecutionStatus

class WorkflowRevision(_message.Message):
    __slots__ = ("id", "owner", "key", "semantic_version", "digest", "definition", "source_path", "source_hash", "source_updated_at", "active", "created_at", "prompt_stale")
    ID_FIELD_NUMBER: _ClassVar[int]
    OWNER_FIELD_NUMBER: _ClassVar[int]
    KEY_FIELD_NUMBER: _ClassVar[int]
    SEMANTIC_VERSION_FIELD_NUMBER: _ClassVar[int]
    DIGEST_FIELD_NUMBER: _ClassVar[int]
    DEFINITION_FIELD_NUMBER: _ClassVar[int]
    SOURCE_PATH_FIELD_NUMBER: _ClassVar[int]
    SOURCE_HASH_FIELD_NUMBER: _ClassVar[int]
    SOURCE_UPDATED_AT_FIELD_NUMBER: _ClassVar[int]
    ACTIVE_FIELD_NUMBER: _ClassVar[int]
    CREATED_AT_FIELD_NUMBER: _ClassVar[int]
    PROMPT_STALE_FIELD_NUMBER: _ClassVar[int]
    id: str
    owner: str
    key: str
    semantic_version: str
    digest: str
    definition: _struct_pb2.Struct
    source_path: str
    source_hash: str
    source_updated_at: _timestamp_pb2.Timestamp
    active: bool
    created_at: _timestamp_pb2.Timestamp
    prompt_stale: bool
    def __init__(self, id: _Optional[str] = ..., owner: _Optional[str] = ..., key: _Optional[str] = ..., semantic_version: _Optional[str] = ..., digest: _Optional[str] = ..., definition: _Optional[_Union[_struct_pb2.Struct, _Mapping]] = ..., source_path: _Optional[str] = ..., source_hash: _Optional[str] = ..., source_updated_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., active: _Optional[bool] = ..., created_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., prompt_stale: _Optional[bool] = ...) -> None: ...

class WorkflowDiagnostic(_message.Message):
    __slots__ = ("code", "path", "message", "severity")
    CODE_FIELD_NUMBER: _ClassVar[int]
    PATH_FIELD_NUMBER: _ClassVar[int]
    MESSAGE_FIELD_NUMBER: _ClassVar[int]
    SEVERITY_FIELD_NUMBER: _ClassVar[int]
    code: str
    path: str
    message: str
    severity: str
    def __init__(self, code: _Optional[str] = ..., path: _Optional[str] = ..., message: _Optional[str] = ..., severity: _Optional[str] = ...) -> None: ...

class WorkflowTerminalReason(_message.Message):
    __slots__ = ("code", "message", "retryable", "budget_name")
    CODE_FIELD_NUMBER: _ClassVar[int]
    MESSAGE_FIELD_NUMBER: _ClassVar[int]
    RETRYABLE_FIELD_NUMBER: _ClassVar[int]
    BUDGET_NAME_FIELD_NUMBER: _ClassVar[int]
    code: str
    message: str
    retryable: bool
    budget_name: str
    def __init__(self, code: _Optional[str] = ..., message: _Optional[str] = ..., retryable: _Optional[bool] = ..., budget_name: _Optional[str] = ...) -> None: ...

class WorkflowBudgetUsage(_message.Message):
    __slots__ = ("turns", "tokens", "cost_usd", "node_attempts", "children", "retries")
    TURNS_FIELD_NUMBER: _ClassVar[int]
    TOKENS_FIELD_NUMBER: _ClassVar[int]
    COST_USD_FIELD_NUMBER: _ClassVar[int]
    NODE_ATTEMPTS_FIELD_NUMBER: _ClassVar[int]
    CHILDREN_FIELD_NUMBER: _ClassVar[int]
    RETRIES_FIELD_NUMBER: _ClassVar[int]
    turns: int
    tokens: int
    cost_usd: float
    node_attempts: int
    children: int
    retries: int
    def __init__(self, turns: _Optional[int] = ..., tokens: _Optional[int] = ..., cost_usd: _Optional[float] = ..., node_attempts: _Optional[int] = ..., children: _Optional[int] = ..., retries: _Optional[int] = ...) -> None: ...

class ChargeReceipt(_message.Message):
    __slots__ = ("amount_micro_usd", "currency", "metering_basis", "measured", "note")
    AMOUNT_MICRO_USD_FIELD_NUMBER: _ClassVar[int]
    CURRENCY_FIELD_NUMBER: _ClassVar[int]
    METERING_BASIS_FIELD_NUMBER: _ClassVar[int]
    MEASURED_FIELD_NUMBER: _ClassVar[int]
    NOTE_FIELD_NUMBER: _ClassVar[int]
    amount_micro_usd: int
    currency: str
    metering_basis: str
    measured: bool
    note: str
    def __init__(self, amount_micro_usd: _Optional[int] = ..., currency: _Optional[str] = ..., metering_basis: _Optional[str] = ..., measured: _Optional[bool] = ..., note: _Optional[str] = ...) -> None: ...

class WorkflowExecution(_message.Message):
    __slots__ = ("id", "owner", "workflow_key", "definition_digest", "status", "current_node_id", "input", "output", "terminal_reason", "budget_usage", "edge_traversals", "version", "idempotency_key", "parent_execution_id", "created_at", "updated_at", "ended_at", "parent_attempt_id", "depth", "observations", "charge_receipt")
    class EdgeTraversalsEntry(_message.Message):
        __slots__ = ("key", "value")
        KEY_FIELD_NUMBER: _ClassVar[int]
        VALUE_FIELD_NUMBER: _ClassVar[int]
        key: str
        value: int
        def __init__(self, key: _Optional[str] = ..., value: _Optional[int] = ...) -> None: ...
    ID_FIELD_NUMBER: _ClassVar[int]
    OWNER_FIELD_NUMBER: _ClassVar[int]
    WORKFLOW_KEY_FIELD_NUMBER: _ClassVar[int]
    DEFINITION_DIGEST_FIELD_NUMBER: _ClassVar[int]
    STATUS_FIELD_NUMBER: _ClassVar[int]
    CURRENT_NODE_ID_FIELD_NUMBER: _ClassVar[int]
    INPUT_FIELD_NUMBER: _ClassVar[int]
    OUTPUT_FIELD_NUMBER: _ClassVar[int]
    TERMINAL_REASON_FIELD_NUMBER: _ClassVar[int]
    BUDGET_USAGE_FIELD_NUMBER: _ClassVar[int]
    EDGE_TRAVERSALS_FIELD_NUMBER: _ClassVar[int]
    VERSION_FIELD_NUMBER: _ClassVar[int]
    IDEMPOTENCY_KEY_FIELD_NUMBER: _ClassVar[int]
    PARENT_EXECUTION_ID_FIELD_NUMBER: _ClassVar[int]
    CREATED_AT_FIELD_NUMBER: _ClassVar[int]
    UPDATED_AT_FIELD_NUMBER: _ClassVar[int]
    ENDED_AT_FIELD_NUMBER: _ClassVar[int]
    PARENT_ATTEMPT_ID_FIELD_NUMBER: _ClassVar[int]
    DEPTH_FIELD_NUMBER: _ClassVar[int]
    OBSERVATIONS_FIELD_NUMBER: _ClassVar[int]
    CHARGE_RECEIPT_FIELD_NUMBER: _ClassVar[int]
    id: str
    owner: str
    workflow_key: str
    definition_digest: str
    status: WorkflowExecutionStatus
    current_node_id: str
    input: _struct_pb2.Value
    output: _struct_pb2.Value
    terminal_reason: WorkflowTerminalReason
    budget_usage: WorkflowBudgetUsage
    edge_traversals: _containers.ScalarMap[str, int]
    version: int
    idempotency_key: str
    parent_execution_id: str
    created_at: _timestamp_pb2.Timestamp
    updated_at: _timestamp_pb2.Timestamp
    ended_at: _timestamp_pb2.Timestamp
    parent_attempt_id: str
    depth: int
    observations: _run_pb2.ReceiptObservations
    charge_receipt: ChargeReceipt
    def __init__(self, id: _Optional[str] = ..., owner: _Optional[str] = ..., workflow_key: _Optional[str] = ..., definition_digest: _Optional[str] = ..., status: _Optional[_Union[WorkflowExecutionStatus, str]] = ..., current_node_id: _Optional[str] = ..., input: _Optional[_Union[_struct_pb2.Value, _Mapping]] = ..., output: _Optional[_Union[_struct_pb2.Value, _Mapping]] = ..., terminal_reason: _Optional[_Union[WorkflowTerminalReason, _Mapping]] = ..., budget_usage: _Optional[_Union[WorkflowBudgetUsage, _Mapping]] = ..., edge_traversals: _Optional[_Mapping[str, int]] = ..., version: _Optional[int] = ..., idempotency_key: _Optional[str] = ..., parent_execution_id: _Optional[str] = ..., created_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., updated_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., ended_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., parent_attempt_id: _Optional[str] = ..., depth: _Optional[int] = ..., observations: _Optional[_Union[_run_pb2.ReceiptObservations, _Mapping]] = ..., charge_receipt: _Optional[_Union[ChargeReceipt, _Mapping]] = ...) -> None: ...

class WorkflowNodeAttempt(_message.Message):
    __slots__ = ("id", "execution_id", "node_id", "ordinal", "strategy", "status", "idempotency_key", "run_id", "conversation_id", "source_attempt_id", "error_code", "version", "created_at", "updated_at", "completed_at", "child_execution_id", "profile_identity", "input_snapshot_digest", "input_snapshot_size_bytes", "raw_output", "validation_error")
    ID_FIELD_NUMBER: _ClassVar[int]
    EXECUTION_ID_FIELD_NUMBER: _ClassVar[int]
    NODE_ID_FIELD_NUMBER: _ClassVar[int]
    ORDINAL_FIELD_NUMBER: _ClassVar[int]
    STRATEGY_FIELD_NUMBER: _ClassVar[int]
    STATUS_FIELD_NUMBER: _ClassVar[int]
    IDEMPOTENCY_KEY_FIELD_NUMBER: _ClassVar[int]
    RUN_ID_FIELD_NUMBER: _ClassVar[int]
    CONVERSATION_ID_FIELD_NUMBER: _ClassVar[int]
    SOURCE_ATTEMPT_ID_FIELD_NUMBER: _ClassVar[int]
    ERROR_CODE_FIELD_NUMBER: _ClassVar[int]
    VERSION_FIELD_NUMBER: _ClassVar[int]
    CREATED_AT_FIELD_NUMBER: _ClassVar[int]
    UPDATED_AT_FIELD_NUMBER: _ClassVar[int]
    COMPLETED_AT_FIELD_NUMBER: _ClassVar[int]
    CHILD_EXECUTION_ID_FIELD_NUMBER: _ClassVar[int]
    PROFILE_IDENTITY_FIELD_NUMBER: _ClassVar[int]
    INPUT_SNAPSHOT_DIGEST_FIELD_NUMBER: _ClassVar[int]
    INPUT_SNAPSHOT_SIZE_BYTES_FIELD_NUMBER: _ClassVar[int]
    RAW_OUTPUT_FIELD_NUMBER: _ClassVar[int]
    VALIDATION_ERROR_FIELD_NUMBER: _ClassVar[int]
    id: str
    execution_id: str
    node_id: str
    ordinal: int
    strategy: str
    status: str
    idempotency_key: str
    run_id: str
    conversation_id: str
    source_attempt_id: str
    error_code: str
    version: int
    created_at: _timestamp_pb2.Timestamp
    updated_at: _timestamp_pb2.Timestamp
    completed_at: _timestamp_pb2.Timestamp
    child_execution_id: str
    profile_identity: str
    input_snapshot_digest: str
    input_snapshot_size_bytes: int
    raw_output: str
    validation_error: str
    def __init__(self, id: _Optional[str] = ..., execution_id: _Optional[str] = ..., node_id: _Optional[str] = ..., ordinal: _Optional[int] = ..., strategy: _Optional[str] = ..., status: _Optional[str] = ..., idempotency_key: _Optional[str] = ..., run_id: _Optional[str] = ..., conversation_id: _Optional[str] = ..., source_attempt_id: _Optional[str] = ..., error_code: _Optional[str] = ..., version: _Optional[int] = ..., created_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., updated_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., completed_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., child_execution_id: _Optional[str] = ..., profile_identity: _Optional[str] = ..., input_snapshot_digest: _Optional[str] = ..., input_snapshot_size_bytes: _Optional[int] = ..., raw_output: _Optional[str] = ..., validation_error: _Optional[str] = ...) -> None: ...

class WorkflowJournalEntry(_message.Message):
    __slots__ = ("id", "execution_id", "sequence", "kind", "node_id", "attempt_id", "payload_digest", "payload_size_bytes", "created_at")
    ID_FIELD_NUMBER: _ClassVar[int]
    EXECUTION_ID_FIELD_NUMBER: _ClassVar[int]
    SEQUENCE_FIELD_NUMBER: _ClassVar[int]
    KIND_FIELD_NUMBER: _ClassVar[int]
    NODE_ID_FIELD_NUMBER: _ClassVar[int]
    ATTEMPT_ID_FIELD_NUMBER: _ClassVar[int]
    PAYLOAD_DIGEST_FIELD_NUMBER: _ClassVar[int]
    PAYLOAD_SIZE_BYTES_FIELD_NUMBER: _ClassVar[int]
    CREATED_AT_FIELD_NUMBER: _ClassVar[int]
    id: str
    execution_id: str
    sequence: int
    kind: str
    node_id: str
    attempt_id: str
    payload_digest: str
    payload_size_bytes: int
    created_at: _timestamp_pb2.Timestamp
    def __init__(self, id: _Optional[str] = ..., execution_id: _Optional[str] = ..., sequence: _Optional[int] = ..., kind: _Optional[str] = ..., node_id: _Optional[str] = ..., attempt_id: _Optional[str] = ..., payload_digest: _Optional[str] = ..., payload_size_bytes: _Optional[int] = ..., created_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ...) -> None: ...
