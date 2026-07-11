from plan_manager.v1.shared import model_pb2 as _model_pb2
from google.protobuf.internal import containers as _containers
from google.protobuf.internal import enum_type_wrapper as _enum_type_wrapper
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Iterable as _Iterable, Mapping as _Mapping
from typing import ClassVar as _ClassVar, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class ValidationOperationStatus(int, metaclass=_enum_type_wrapper.EnumTypeWrapper):
    __slots__ = ()
    VALIDATION_OPERATION_STATUS_UNSPECIFIED: _ClassVar[ValidationOperationStatus]
    VALIDATION_OPERATION_STATUS_QUEUED: _ClassVar[ValidationOperationStatus]
    VALIDATION_OPERATION_STATUS_RUNNING: _ClassVar[ValidationOperationStatus]
    VALIDATION_OPERATION_STATUS_TERMINAL: _ClassVar[ValidationOperationStatus]

class ValidationChildStatus(int, metaclass=_enum_type_wrapper.EnumTypeWrapper):
    __slots__ = ()
    VALIDATION_CHILD_STATUS_UNSPECIFIED: _ClassVar[ValidationChildStatus]
    VALIDATION_CHILD_STATUS_QUEUED: _ClassVar[ValidationChildStatus]
    VALIDATION_CHILD_STATUS_RUNNING: _ClassVar[ValidationChildStatus]
    VALIDATION_CHILD_STATUS_TERMINAL: _ClassVar[ValidationChildStatus]
VALIDATION_OPERATION_STATUS_UNSPECIFIED: ValidationOperationStatus
VALIDATION_OPERATION_STATUS_QUEUED: ValidationOperationStatus
VALIDATION_OPERATION_STATUS_RUNNING: ValidationOperationStatus
VALIDATION_OPERATION_STATUS_TERMINAL: ValidationOperationStatus
VALIDATION_CHILD_STATUS_UNSPECIFIED: ValidationChildStatus
VALIDATION_CHILD_STATUS_QUEUED: ValidationChildStatus
VALIDATION_CHILD_STATUS_RUNNING: ValidationChildStatus
VALIDATION_CHILD_STATUS_TERMINAL: ValidationChildStatus

class ResolveReferencesRequest(_message.Message):
    __slots__ = ("plan_id", "phase_id")
    PLAN_ID_FIELD_NUMBER: _ClassVar[int]
    PHASE_ID_FIELD_NUMBER: _ClassVar[int]
    plan_id: str
    phase_id: str
    def __init__(self, plan_id: _Optional[str] = ..., phase_id: _Optional[str] = ...) -> None: ...

class ResolveReferencesResponse(_message.Message):
    __slots__ = ("references", "degraded")
    REFERENCES_FIELD_NUMBER: _ClassVar[int]
    DEGRADED_FIELD_NUMBER: _ClassVar[int]
    references: _containers.RepeatedCompositeFieldContainer[_model_pb2.Reference]
    degraded: bool
    def __init__(self, references: _Optional[_Iterable[_Union[_model_pb2.Reference, _Mapping]]] = ..., degraded: _Optional[bool] = ...) -> None: ...

class ComputeStalenessRequest(_message.Message):
    __slots__ = ("plan_id", "phase_id")
    PLAN_ID_FIELD_NUMBER: _ClassVar[int]
    PHASE_ID_FIELD_NUMBER: _ClassVar[int]
    plan_id: str
    phase_id: str
    def __init__(self, plan_id: _Optional[str] = ..., phase_id: _Optional[str] = ...) -> None: ...

class ComputeStalenessResponse(_message.Message):
    __slots__ = ("overall", "references", "degraded")
    OVERALL_FIELD_NUMBER: _ClassVar[int]
    REFERENCES_FIELD_NUMBER: _ClassVar[int]
    DEGRADED_FIELD_NUMBER: _ClassVar[int]
    overall: _model_pb2.StalenessTier
    references: _containers.RepeatedCompositeFieldContainer[_model_pb2.Reference]
    degraded: bool
    def __init__(self, overall: _Optional[_Union[_model_pb2.StalenessTier, str]] = ..., references: _Optional[_Iterable[_Union[_model_pb2.Reference, _Mapping]]] = ..., degraded: _Optional[bool] = ...) -> None: ...

class DeriveBaselineScopeRequest(_message.Message):
    __slots__ = ("plan_id", "phase_id")
    PLAN_ID_FIELD_NUMBER: _ClassVar[int]
    PHASE_ID_FIELD_NUMBER: _ClassVar[int]
    plan_id: str
    phase_id: str
    def __init__(self, plan_id: _Optional[str] = ..., phase_id: _Optional[str] = ...) -> None: ...

class DeriveBaselineScopeResponse(_message.Message):
    __slots__ = ("commands", "locations")
    COMMANDS_FIELD_NUMBER: _ClassVar[int]
    LOCATIONS_FIELD_NUMBER: _ClassVar[int]
    commands: _containers.RepeatedScalarFieldContainer[str]
    locations: _containers.RepeatedScalarFieldContainer[str]
    def __init__(self, commands: _Optional[_Iterable[str]] = ..., locations: _Optional[_Iterable[str]] = ...) -> None: ...

class ValidationOperationError(_message.Message):
    __slots__ = ("code", "detail")
    CODE_FIELD_NUMBER: _ClassVar[int]
    DETAIL_FIELD_NUMBER: _ClassVar[int]
    code: str
    detail: str
    def __init__(self, code: _Optional[str] = ..., detail: _Optional[str] = ...) -> None: ...

class ValidationChildOperation(_message.Message):
    __slots__ = ("id", "command", "oracle", "status", "attempt", "external_id", "verdict", "detail", "error", "queued_at", "started_at", "terminal_at")
    ID_FIELD_NUMBER: _ClassVar[int]
    COMMAND_FIELD_NUMBER: _ClassVar[int]
    ORACLE_FIELD_NUMBER: _ClassVar[int]
    STATUS_FIELD_NUMBER: _ClassVar[int]
    ATTEMPT_FIELD_NUMBER: _ClassVar[int]
    EXTERNAL_ID_FIELD_NUMBER: _ClassVar[int]
    VERDICT_FIELD_NUMBER: _ClassVar[int]
    DETAIL_FIELD_NUMBER: _ClassVar[int]
    ERROR_FIELD_NUMBER: _ClassVar[int]
    QUEUED_AT_FIELD_NUMBER: _ClassVar[int]
    STARTED_AT_FIELD_NUMBER: _ClassVar[int]
    TERMINAL_AT_FIELD_NUMBER: _ClassVar[int]
    id: str
    command: str
    oracle: bool
    status: ValidationChildStatus
    attempt: int
    external_id: str
    verdict: _model_pb2.ValidationVerdict
    detail: str
    error: ValidationOperationError
    queued_at: str
    started_at: str
    terminal_at: str
    def __init__(self, id: _Optional[str] = ..., command: _Optional[str] = ..., oracle: _Optional[bool] = ..., status: _Optional[_Union[ValidationChildStatus, str]] = ..., attempt: _Optional[int] = ..., external_id: _Optional[str] = ..., verdict: _Optional[_Union[_model_pb2.ValidationVerdict, str]] = ..., detail: _Optional[str] = ..., error: _Optional[_Union[ValidationOperationError, _Mapping]] = ..., queued_at: _Optional[str] = ..., started_at: _Optional[str] = ..., terminal_at: _Optional[str] = ...) -> None: ...

class ValidationOperation(_message.Message):
    __slots__ = ("id", "plan_id", "phase_id", "idempotency_key", "status", "attempt", "children", "result", "result_ref", "error", "queued_at", "started_at", "terminal_at", "queue_budget_seconds", "execution_budget_seconds", "transport_wait_budget_seconds", "recommended_wait_seconds", "schema_version", "scope_fingerprint", "queue_reason")
    ID_FIELD_NUMBER: _ClassVar[int]
    PLAN_ID_FIELD_NUMBER: _ClassVar[int]
    PHASE_ID_FIELD_NUMBER: _ClassVar[int]
    IDEMPOTENCY_KEY_FIELD_NUMBER: _ClassVar[int]
    STATUS_FIELD_NUMBER: _ClassVar[int]
    ATTEMPT_FIELD_NUMBER: _ClassVar[int]
    CHILDREN_FIELD_NUMBER: _ClassVar[int]
    RESULT_FIELD_NUMBER: _ClassVar[int]
    RESULT_REF_FIELD_NUMBER: _ClassVar[int]
    ERROR_FIELD_NUMBER: _ClassVar[int]
    QUEUED_AT_FIELD_NUMBER: _ClassVar[int]
    STARTED_AT_FIELD_NUMBER: _ClassVar[int]
    TERMINAL_AT_FIELD_NUMBER: _ClassVar[int]
    QUEUE_BUDGET_SECONDS_FIELD_NUMBER: _ClassVar[int]
    EXECUTION_BUDGET_SECONDS_FIELD_NUMBER: _ClassVar[int]
    TRANSPORT_WAIT_BUDGET_SECONDS_FIELD_NUMBER: _ClassVar[int]
    RECOMMENDED_WAIT_SECONDS_FIELD_NUMBER: _ClassVar[int]
    SCHEMA_VERSION_FIELD_NUMBER: _ClassVar[int]
    SCOPE_FINGERPRINT_FIELD_NUMBER: _ClassVar[int]
    QUEUE_REASON_FIELD_NUMBER: _ClassVar[int]
    id: str
    plan_id: str
    phase_id: str
    idempotency_key: str
    status: ValidationOperationStatus
    attempt: int
    children: _containers.RepeatedCompositeFieldContainer[ValidationChildOperation]
    result: _model_pb2.ValidationResult
    result_ref: str
    error: ValidationOperationError
    queued_at: str
    started_at: str
    terminal_at: str
    queue_budget_seconds: int
    execution_budget_seconds: int
    transport_wait_budget_seconds: int
    recommended_wait_seconds: int
    schema_version: int
    scope_fingerprint: str
    queue_reason: str
    def __init__(self, id: _Optional[str] = ..., plan_id: _Optional[str] = ..., phase_id: _Optional[str] = ..., idempotency_key: _Optional[str] = ..., status: _Optional[_Union[ValidationOperationStatus, str]] = ..., attempt: _Optional[int] = ..., children: _Optional[_Iterable[_Union[ValidationChildOperation, _Mapping]]] = ..., result: _Optional[_Union[_model_pb2.ValidationResult, _Mapping]] = ..., result_ref: _Optional[str] = ..., error: _Optional[_Union[ValidationOperationError, _Mapping]] = ..., queued_at: _Optional[str] = ..., started_at: _Optional[str] = ..., terminal_at: _Optional[str] = ..., queue_budget_seconds: _Optional[int] = ..., execution_budget_seconds: _Optional[int] = ..., transport_wait_budget_seconds: _Optional[int] = ..., recommended_wait_seconds: _Optional[int] = ..., schema_version: _Optional[int] = ..., scope_fingerprint: _Optional[str] = ..., queue_reason: _Optional[str] = ...) -> None: ...

class StartValidationRequest(_message.Message):
    __slots__ = ("plan_id", "phase_id", "idempotency_key")
    PLAN_ID_FIELD_NUMBER: _ClassVar[int]
    PHASE_ID_FIELD_NUMBER: _ClassVar[int]
    IDEMPOTENCY_KEY_FIELD_NUMBER: _ClassVar[int]
    plan_id: str
    phase_id: str
    idempotency_key: str
    def __init__(self, plan_id: _Optional[str] = ..., phase_id: _Optional[str] = ..., idempotency_key: _Optional[str] = ...) -> None: ...

class StartValidationResponse(_message.Message):
    __slots__ = ("operation", "deduplicated")
    OPERATION_FIELD_NUMBER: _ClassVar[int]
    DEDUPLICATED_FIELD_NUMBER: _ClassVar[int]
    operation: ValidationOperation
    deduplicated: bool
    def __init__(self, operation: _Optional[_Union[ValidationOperation, _Mapping]] = ..., deduplicated: _Optional[bool] = ...) -> None: ...

class GetValidationOperationRequest(_message.Message):
    __slots__ = ("operation_id", "wait")
    OPERATION_ID_FIELD_NUMBER: _ClassVar[int]
    WAIT_FIELD_NUMBER: _ClassVar[int]
    operation_id: str
    wait: bool
    def __init__(self, operation_id: _Optional[str] = ..., wait: _Optional[bool] = ...) -> None: ...

class GetValidationOperationResponse(_message.Message):
    __slots__ = ("operation",)
    OPERATION_FIELD_NUMBER: _ClassVar[int]
    operation: ValidationOperation
    def __init__(self, operation: _Optional[_Union[ValidationOperation, _Mapping]] = ...) -> None: ...

class RunValidationRequest(_message.Message):
    __slots__ = ("plan_id", "phase_id")
    PLAN_ID_FIELD_NUMBER: _ClassVar[int]
    PHASE_ID_FIELD_NUMBER: _ClassVar[int]
    plan_id: str
    phase_id: str
    def __init__(self, plan_id: _Optional[str] = ..., phase_id: _Optional[str] = ...) -> None: ...

class RunValidationResponse(_message.Message):
    __slots__ = ("result",)
    RESULT_FIELD_NUMBER: _ClassVar[int]
    result: _model_pb2.ValidationResult
    def __init__(self, result: _Optional[_Union[_model_pb2.ValidationResult, _Mapping]] = ...) -> None: ...

class VerifyDefinitionOfDoneRequest(_message.Message):
    __slots__ = ("plan_id",)
    PLAN_ID_FIELD_NUMBER: _ClassVar[int]
    plan_id: str
    def __init__(self, plan_id: _Optional[str] = ...) -> None: ...

class VerifyDefinitionOfDoneResponse(_message.Message):
    __slots__ = ("result", "dod_met")
    RESULT_FIELD_NUMBER: _ClassVar[int]
    DOD_MET_FIELD_NUMBER: _ClassVar[int]
    result: _model_pb2.ValidationResult
    dod_met: bool
    def __init__(self, result: _Optional[_Union[_model_pb2.ValidationResult, _Mapping]] = ..., dod_met: _Optional[bool] = ...) -> None: ...
