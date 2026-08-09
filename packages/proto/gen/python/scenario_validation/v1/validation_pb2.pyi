import datetime

from common.v1 import maturity_pb2 as _maturity_pb2
from common.v1 import metrics_pb2 as _metrics_pb2
from common.v1 import validation_target_pb2 as _validation_target_pb2
from google.protobuf import any_pb2 as _any_pb2
from google.protobuf import duration_pb2 as _duration_pb2
from google.protobuf import timestamp_pb2 as _timestamp_pb2
from google.protobuf.internal import containers as _containers
from google.protobuf.internal import enum_type_wrapper as _enum_type_wrapper
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Iterable as _Iterable, Mapping as _Mapping
from typing import ClassVar as _ClassVar, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class ValidationStatus(int, metaclass=_enum_type_wrapper.EnumTypeWrapper):
    __slots__ = ()
    VALIDATION_STATUS_UNSPECIFIED: _ClassVar[ValidationStatus]
    VALIDATION_STATUS_PASSED: _ClassVar[ValidationStatus]
    VALIDATION_STATUS_FAILED: _ClassVar[ValidationStatus]
    VALIDATION_STATUS_DEGRADED: _ClassVar[ValidationStatus]
    VALIDATION_STATUS_ERROR: _ClassVar[ValidationStatus]
    VALIDATION_STATUS_SKIPPED: _ClassVar[ValidationStatus]

class ValidationRunState(int, metaclass=_enum_type_wrapper.EnumTypeWrapper):
    __slots__ = ()
    VALIDATION_RUN_STATE_UNSPECIFIED: _ClassVar[ValidationRunState]
    VALIDATION_RUN_STATE_QUEUED: _ClassVar[ValidationRunState]
    VALIDATION_RUN_STATE_RUNNING: _ClassVar[ValidationRunState]
    VALIDATION_RUN_STATE_SUCCEEDED: _ClassVar[ValidationRunState]
    VALIDATION_RUN_STATE_FAILED: _ClassVar[ValidationRunState]
    VALIDATION_RUN_STATE_CANCELED: _ClassVar[ValidationRunState]
    VALIDATION_RUN_STATE_RECOVERY_FAILED: _ClassVar[ValidationRunState]

class ValidationRunErrorCode(int, metaclass=_enum_type_wrapper.EnumTypeWrapper):
    __slots__ = ()
    VALIDATION_RUN_ERROR_CODE_UNSPECIFIED: _ClassVar[ValidationRunErrorCode]
    VALIDATION_RUN_ERROR_CODE_INVALID_TRANSITION: _ClassVar[ValidationRunErrorCode]
    VALIDATION_RUN_ERROR_CODE_NOT_FOUND: _ClassVar[ValidationRunErrorCode]
    VALIDATION_RUN_ERROR_CODE_IDEMPOTENCY_CONFLICT: _ClassVar[ValidationRunErrorCode]
    VALIDATION_RUN_ERROR_CODE_ABORT_REJECTED: _ClassVar[ValidationRunErrorCode]
    VALIDATION_RUN_ERROR_CODE_EXECUTION_FAILED: _ClassVar[ValidationRunErrorCode]
    VALIDATION_RUN_ERROR_CODE_RECOVERY_FAILED: _ClassVar[ValidationRunErrorCode]
    VALIDATION_RUN_ERROR_CODE_WAIT_TIMEOUT: _ClassVar[ValidationRunErrorCode]
VALIDATION_STATUS_UNSPECIFIED: ValidationStatus
VALIDATION_STATUS_PASSED: ValidationStatus
VALIDATION_STATUS_FAILED: ValidationStatus
VALIDATION_STATUS_DEGRADED: ValidationStatus
VALIDATION_STATUS_ERROR: ValidationStatus
VALIDATION_STATUS_SKIPPED: ValidationStatus
VALIDATION_RUN_STATE_UNSPECIFIED: ValidationRunState
VALIDATION_RUN_STATE_QUEUED: ValidationRunState
VALIDATION_RUN_STATE_RUNNING: ValidationRunState
VALIDATION_RUN_STATE_SUCCEEDED: ValidationRunState
VALIDATION_RUN_STATE_FAILED: ValidationRunState
VALIDATION_RUN_STATE_CANCELED: ValidationRunState
VALIDATION_RUN_STATE_RECOVERY_FAILED: ValidationRunState
VALIDATION_RUN_ERROR_CODE_UNSPECIFIED: ValidationRunErrorCode
VALIDATION_RUN_ERROR_CODE_INVALID_TRANSITION: ValidationRunErrorCode
VALIDATION_RUN_ERROR_CODE_NOT_FOUND: ValidationRunErrorCode
VALIDATION_RUN_ERROR_CODE_IDEMPOTENCY_CONFLICT: ValidationRunErrorCode
VALIDATION_RUN_ERROR_CODE_ABORT_REJECTED: ValidationRunErrorCode
VALIDATION_RUN_ERROR_CODE_EXECUTION_FAILED: ValidationRunErrorCode
VALIDATION_RUN_ERROR_CODE_RECOVERY_FAILED: ValidationRunErrorCode
VALIDATION_RUN_ERROR_CODE_WAIT_TIMEOUT: ValidationRunErrorCode

class DescribeProviderRequest(_message.Message):
    __slots__ = ()
    def __init__(self) -> None: ...

class DescribeProviderResponse(_message.Message):
    __slots__ = ("provider", "phase", "spec_version", "contract", "build", "capabilities")
    PROVIDER_FIELD_NUMBER: _ClassVar[int]
    PHASE_FIELD_NUMBER: _ClassVar[int]
    SPEC_VERSION_FIELD_NUMBER: _ClassVar[int]
    CONTRACT_FIELD_NUMBER: _ClassVar[int]
    BUILD_FIELD_NUMBER: _ClassVar[int]
    CAPABILITIES_FIELD_NUMBER: _ClassVar[int]
    provider: str
    phase: str
    spec_version: str
    contract: str
    build: ProviderBuild
    capabilities: ProviderCapabilities
    def __init__(self, provider: _Optional[str] = ..., phase: _Optional[str] = ..., spec_version: _Optional[str] = ..., contract: _Optional[str] = ..., build: _Optional[_Union[ProviderBuild, _Mapping]] = ..., capabilities: _Optional[_Union[ProviderCapabilities, _Mapping]] = ...) -> None: ...

class ProviderBuild(_message.Message):
    __slots__ = ("revision", "built_at", "binary_modified_at", "freshness_digest")
    REVISION_FIELD_NUMBER: _ClassVar[int]
    BUILT_AT_FIELD_NUMBER: _ClassVar[int]
    BINARY_MODIFIED_AT_FIELD_NUMBER: _ClassVar[int]
    FRESHNESS_DIGEST_FIELD_NUMBER: _ClassVar[int]
    revision: str
    built_at: _timestamp_pb2.Timestamp
    binary_modified_at: _timestamp_pb2.Timestamp
    freshness_digest: str
    def __init__(self, revision: _Optional[str] = ..., built_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., binary_modified_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., freshness_digest: _Optional[str] = ...) -> None: ...

class ProviderCapabilities(_message.Message):
    __slots__ = ("supports_execution", "delivery_mode", "supports_fixes", "target_kinds")
    SUPPORTS_EXECUTION_FIELD_NUMBER: _ClassVar[int]
    DELIVERY_MODE_FIELD_NUMBER: _ClassVar[int]
    SUPPORTS_FIXES_FIELD_NUMBER: _ClassVar[int]
    TARGET_KINDS_FIELD_NUMBER: _ClassVar[int]
    supports_execution: bool
    delivery_mode: str
    supports_fixes: bool
    target_kinds: _containers.RepeatedScalarFieldContainer[_validation_target_pb2.ValidationTargetKind]
    def __init__(self, supports_execution: _Optional[bool] = ..., delivery_mode: _Optional[str] = ..., supports_fixes: _Optional[bool] = ..., target_kinds: _Optional[_Iterable[_Union[_validation_target_pb2.ValidationTargetKind, str]]] = ...) -> None: ...

class ValidateScenarioRequest(_message.Message):
    __slots__ = ("scenario", "path", "include_execution", "capability_subset")
    SCENARIO_FIELD_NUMBER: _ClassVar[int]
    PATH_FIELD_NUMBER: _ClassVar[int]
    INCLUDE_EXECUTION_FIELD_NUMBER: _ClassVar[int]
    CAPABILITY_SUBSET_FIELD_NUMBER: _ClassVar[int]
    scenario: str
    path: str
    include_execution: bool
    capability_subset: _containers.RepeatedScalarFieldContainer[str]
    def __init__(self, scenario: _Optional[str] = ..., path: _Optional[str] = ..., include_execution: _Optional[bool] = ..., capability_subset: _Optional[_Iterable[str]] = ...) -> None: ...

class ValidateTargetRequest(_message.Message):
    __slots__ = ("target", "include_execution", "path", "capability_subset")
    TARGET_FIELD_NUMBER: _ClassVar[int]
    INCLUDE_EXECUTION_FIELD_NUMBER: _ClassVar[int]
    PATH_FIELD_NUMBER: _ClassVar[int]
    CAPABILITY_SUBSET_FIELD_NUMBER: _ClassVar[int]
    target: _validation_target_pb2.ValidationTarget
    include_execution: bool
    path: str
    capability_subset: _containers.RepeatedScalarFieldContainer[str]
    def __init__(self, target: _Optional[_Union[_validation_target_pb2.ValidationTarget, _Mapping]] = ..., include_execution: _Optional[bool] = ..., path: _Optional[str] = ..., capability_subset: _Optional[_Iterable[str]] = ...) -> None: ...

class ValidateTargetResponse(_message.Message):
    __slots__ = ("target", "status", "assessment", "native_detail", "metrics")
    TARGET_FIELD_NUMBER: _ClassVar[int]
    STATUS_FIELD_NUMBER: _ClassVar[int]
    ASSESSMENT_FIELD_NUMBER: _ClassVar[int]
    NATIVE_DETAIL_FIELD_NUMBER: _ClassVar[int]
    METRICS_FIELD_NUMBER: _ClassVar[int]
    target: _validation_target_pb2.ValidationTarget
    status: ValidationStatus
    assessment: _maturity_pb2.MaturityAssessment
    native_detail: _any_pb2.Any
    metrics: _metrics_pb2.ExecutionMetrics
    def __init__(self, target: _Optional[_Union[_validation_target_pb2.ValidationTarget, _Mapping]] = ..., status: _Optional[_Union[ValidationStatus, str]] = ..., assessment: _Optional[_Union[_maturity_pb2.MaturityAssessment, _Mapping]] = ..., native_detail: _Optional[_Union[_any_pb2.Any, _Mapping]] = ..., metrics: _Optional[_Union[_metrics_pb2.ExecutionMetrics, _Mapping]] = ...) -> None: ...

class ValidateScenarioResponse(_message.Message):
    __slots__ = ("scenario", "status", "assessment", "native_detail", "metrics")
    SCENARIO_FIELD_NUMBER: _ClassVar[int]
    STATUS_FIELD_NUMBER: _ClassVar[int]
    ASSESSMENT_FIELD_NUMBER: _ClassVar[int]
    NATIVE_DETAIL_FIELD_NUMBER: _ClassVar[int]
    METRICS_FIELD_NUMBER: _ClassVar[int]
    scenario: str
    status: ValidationStatus
    assessment: _maturity_pb2.MaturityAssessment
    native_detail: _any_pb2.Any
    metrics: _metrics_pb2.ExecutionMetrics
    def __init__(self, scenario: _Optional[str] = ..., status: _Optional[_Union[ValidationStatus, str]] = ..., assessment: _Optional[_Union[_maturity_pb2.MaturityAssessment, _Mapping]] = ..., native_detail: _Optional[_Union[_any_pb2.Any, _Mapping]] = ..., metrics: _Optional[_Union[_metrics_pb2.ExecutionMetrics, _Mapping]] = ...) -> None: ...

class ValidationRunError(_message.Message):
    __slots__ = ("code", "message", "retryable")
    CODE_FIELD_NUMBER: _ClassVar[int]
    MESSAGE_FIELD_NUMBER: _ClassVar[int]
    RETRYABLE_FIELD_NUMBER: _ClassVar[int]
    code: ValidationRunErrorCode
    message: str
    retryable: bool
    def __init__(self, code: _Optional[_Union[ValidationRunErrorCode, str]] = ..., message: _Optional[str] = ..., retryable: _Optional[bool] = ...) -> None: ...

class ValidationRun(_message.Message):
    __slots__ = ("run_id", "scenario", "path", "idempotency_key", "parent_run_id", "state", "created_at", "started_at", "completed_at", "estimated_remaining", "preliminary_static_result", "terminal_result", "error", "artifact_references", "cancellation_requested")
    RUN_ID_FIELD_NUMBER: _ClassVar[int]
    SCENARIO_FIELD_NUMBER: _ClassVar[int]
    PATH_FIELD_NUMBER: _ClassVar[int]
    IDEMPOTENCY_KEY_FIELD_NUMBER: _ClassVar[int]
    PARENT_RUN_ID_FIELD_NUMBER: _ClassVar[int]
    STATE_FIELD_NUMBER: _ClassVar[int]
    CREATED_AT_FIELD_NUMBER: _ClassVar[int]
    STARTED_AT_FIELD_NUMBER: _ClassVar[int]
    COMPLETED_AT_FIELD_NUMBER: _ClassVar[int]
    ESTIMATED_REMAINING_FIELD_NUMBER: _ClassVar[int]
    PRELIMINARY_STATIC_RESULT_FIELD_NUMBER: _ClassVar[int]
    TERMINAL_RESULT_FIELD_NUMBER: _ClassVar[int]
    ERROR_FIELD_NUMBER: _ClassVar[int]
    ARTIFACT_REFERENCES_FIELD_NUMBER: _ClassVar[int]
    CANCELLATION_REQUESTED_FIELD_NUMBER: _ClassVar[int]
    run_id: str
    scenario: str
    path: str
    idempotency_key: str
    parent_run_id: str
    state: ValidationRunState
    created_at: _timestamp_pb2.Timestamp
    started_at: _timestamp_pb2.Timestamp
    completed_at: _timestamp_pb2.Timestamp
    estimated_remaining: _duration_pb2.Duration
    preliminary_static_result: ValidateScenarioResponse
    terminal_result: ValidateScenarioResponse
    error: ValidationRunError
    artifact_references: _containers.RepeatedCompositeFieldContainer[_any_pb2.Any]
    cancellation_requested: bool
    def __init__(self, run_id: _Optional[str] = ..., scenario: _Optional[str] = ..., path: _Optional[str] = ..., idempotency_key: _Optional[str] = ..., parent_run_id: _Optional[str] = ..., state: _Optional[_Union[ValidationRunState, str]] = ..., created_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., started_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., completed_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., estimated_remaining: _Optional[_Union[datetime.timedelta, _duration_pb2.Duration, _Mapping]] = ..., preliminary_static_result: _Optional[_Union[ValidateScenarioResponse, _Mapping]] = ..., terminal_result: _Optional[_Union[ValidateScenarioResponse, _Mapping]] = ..., error: _Optional[_Union[ValidationRunError, _Mapping]] = ..., artifact_references: _Optional[_Iterable[_Union[_any_pb2.Any, _Mapping]]] = ..., cancellation_requested: _Optional[bool] = ...) -> None: ...

class StartValidationRunRequest(_message.Message):
    __slots__ = ("scenario", "path", "idempotency_key", "parent_run_id", "desktop_binding", "capability_subset")
    SCENARIO_FIELD_NUMBER: _ClassVar[int]
    PATH_FIELD_NUMBER: _ClassVar[int]
    IDEMPOTENCY_KEY_FIELD_NUMBER: _ClassVar[int]
    PARENT_RUN_ID_FIELD_NUMBER: _ClassVar[int]
    DESKTOP_BINDING_FIELD_NUMBER: _ClassVar[int]
    CAPABILITY_SUBSET_FIELD_NUMBER: _ClassVar[int]
    scenario: str
    path: str
    idempotency_key: str
    parent_run_id: str
    desktop_binding: DesktopValidationBinding
    capability_subset: _containers.RepeatedScalarFieldContainer[str]
    def __init__(self, scenario: _Optional[str] = ..., path: _Optional[str] = ..., idempotency_key: _Optional[str] = ..., parent_run_id: _Optional[str] = ..., desktop_binding: _Optional[_Union[DesktopValidationBinding, _Mapping]] = ..., capability_subset: _Optional[_Iterable[str]] = ...) -> None: ...

class DesktopValidationBinding(_message.Message):
    __slots__ = ("target_id", "cdp_endpoint", "renderer_id", "renderer_url", "renderer_title", "scenario_name", "artifact_digest", "context_id", "profile_id", "cdp_transport", "workflow_path", "workflow_id")
    TARGET_ID_FIELD_NUMBER: _ClassVar[int]
    CDP_ENDPOINT_FIELD_NUMBER: _ClassVar[int]
    RENDERER_ID_FIELD_NUMBER: _ClassVar[int]
    RENDERER_URL_FIELD_NUMBER: _ClassVar[int]
    RENDERER_TITLE_FIELD_NUMBER: _ClassVar[int]
    SCENARIO_NAME_FIELD_NUMBER: _ClassVar[int]
    ARTIFACT_DIGEST_FIELD_NUMBER: _ClassVar[int]
    CONTEXT_ID_FIELD_NUMBER: _ClassVar[int]
    PROFILE_ID_FIELD_NUMBER: _ClassVar[int]
    CDP_TRANSPORT_FIELD_NUMBER: _ClassVar[int]
    WORKFLOW_PATH_FIELD_NUMBER: _ClassVar[int]
    WORKFLOW_ID_FIELD_NUMBER: _ClassVar[int]
    target_id: str
    cdp_endpoint: str
    renderer_id: str
    renderer_url: str
    renderer_title: str
    scenario_name: str
    artifact_digest: str
    context_id: str
    profile_id: str
    cdp_transport: str
    workflow_path: str
    workflow_id: str
    def __init__(self, target_id: _Optional[str] = ..., cdp_endpoint: _Optional[str] = ..., renderer_id: _Optional[str] = ..., renderer_url: _Optional[str] = ..., renderer_title: _Optional[str] = ..., scenario_name: _Optional[str] = ..., artifact_digest: _Optional[str] = ..., context_id: _Optional[str] = ..., profile_id: _Optional[str] = ..., cdp_transport: _Optional[str] = ..., workflow_path: _Optional[str] = ..., workflow_id: _Optional[str] = ...) -> None: ...

class StartValidationRunResponse(_message.Message):
    __slots__ = ("run",)
    RUN_FIELD_NUMBER: _ClassVar[int]
    run: ValidationRun
    def __init__(self, run: _Optional[_Union[ValidationRun, _Mapping]] = ...) -> None: ...

class GetValidationRunRequest(_message.Message):
    __slots__ = ("run_id",)
    RUN_ID_FIELD_NUMBER: _ClassVar[int]
    run_id: str
    def __init__(self, run_id: _Optional[str] = ...) -> None: ...

class GetValidationRunResponse(_message.Message):
    __slots__ = ("run",)
    RUN_FIELD_NUMBER: _ClassVar[int]
    run: ValidationRun
    def __init__(self, run: _Optional[_Union[ValidationRun, _Mapping]] = ...) -> None: ...

class WaitValidationRunRequest(_message.Message):
    __slots__ = ("run_id", "timeout")
    RUN_ID_FIELD_NUMBER: _ClassVar[int]
    TIMEOUT_FIELD_NUMBER: _ClassVar[int]
    run_id: str
    timeout: _duration_pb2.Duration
    def __init__(self, run_id: _Optional[str] = ..., timeout: _Optional[_Union[datetime.timedelta, _duration_pb2.Duration, _Mapping]] = ...) -> None: ...

class WaitValidationRunResponse(_message.Message):
    __slots__ = ("run",)
    RUN_FIELD_NUMBER: _ClassVar[int]
    run: ValidationRun
    def __init__(self, run: _Optional[_Union[ValidationRun, _Mapping]] = ...) -> None: ...

class AbortValidationRunRequest(_message.Message):
    __slots__ = ("run_id", "reason")
    RUN_ID_FIELD_NUMBER: _ClassVar[int]
    REASON_FIELD_NUMBER: _ClassVar[int]
    run_id: str
    reason: str
    def __init__(self, run_id: _Optional[str] = ..., reason: _Optional[str] = ...) -> None: ...

class AbortValidationRunResponse(_message.Message):
    __slots__ = ("run",)
    RUN_FIELD_NUMBER: _ClassVar[int]
    run: ValidationRun
    def __init__(self, run: _Optional[_Union[ValidationRun, _Mapping]] = ...) -> None: ...

class FixRequest(_message.Message):
    __slots__ = ("scenario", "path", "rule_ids")
    SCENARIO_FIELD_NUMBER: _ClassVar[int]
    PATH_FIELD_NUMBER: _ClassVar[int]
    RULE_IDS_FIELD_NUMBER: _ClassVar[int]
    scenario: str
    path: str
    rule_ids: _containers.RepeatedScalarFieldContainer[str]
    def __init__(self, scenario: _Optional[str] = ..., path: _Optional[str] = ..., rule_ids: _Optional[_Iterable[str]] = ...) -> None: ...

class FixCandidate(_message.Message):
    __slots__ = ("rule_id", "file_path", "description", "before", "after", "applied")
    RULE_ID_FIELD_NUMBER: _ClassVar[int]
    FILE_PATH_FIELD_NUMBER: _ClassVar[int]
    DESCRIPTION_FIELD_NUMBER: _ClassVar[int]
    BEFORE_FIELD_NUMBER: _ClassVar[int]
    AFTER_FIELD_NUMBER: _ClassVar[int]
    APPLIED_FIELD_NUMBER: _ClassVar[int]
    rule_id: str
    file_path: str
    description: str
    before: str
    after: str
    applied: bool
    def __init__(self, rule_id: _Optional[str] = ..., file_path: _Optional[str] = ..., description: _Optional[str] = ..., before: _Optional[str] = ..., after: _Optional[str] = ..., applied: _Optional[bool] = ...) -> None: ...

class FixResponse(_message.Message):
    __slots__ = ("scenario", "applied", "candidates", "messages")
    SCENARIO_FIELD_NUMBER: _ClassVar[int]
    APPLIED_FIELD_NUMBER: _ClassVar[int]
    CANDIDATES_FIELD_NUMBER: _ClassVar[int]
    MESSAGES_FIELD_NUMBER: _ClassVar[int]
    scenario: str
    applied: bool
    candidates: _containers.RepeatedCompositeFieldContainer[FixCandidate]
    messages: _containers.RepeatedScalarFieldContainer[str]
    def __init__(self, scenario: _Optional[str] = ..., applied: _Optional[bool] = ..., candidates: _Optional[_Iterable[_Union[FixCandidate, _Mapping]]] = ..., messages: _Optional[_Iterable[str]] = ...) -> None: ...
