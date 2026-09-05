import datetime

from common.v1 import validation_target_pb2 as _validation_target_pb2
from google.protobuf import duration_pb2 as _duration_pb2
from google.protobuf import timestamp_pb2 as _timestamp_pb2
from google.protobuf.internal import containers as _containers
from google.protobuf.internal import enum_type_wrapper as _enum_type_wrapper
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Iterable as _Iterable, Mapping as _Mapping
from typing import ClassVar as _ClassVar, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class ValidationPurpose(int, metaclass=_enum_type_wrapper.EnumTypeWrapper):
    __slots__ = ()
    VALIDATION_PURPOSE_UNSPECIFIED: _ClassVar[ValidationPurpose]
    VALIDATION_PURPOSE_PHASE: _ClassVar[ValidationPurpose]
    VALIDATION_PURPOSE_REGRESSION_BEFORE: _ClassVar[ValidationPurpose]
    VALIDATION_PURPOSE_REGRESSION_CURRENT: _ClassVar[ValidationPurpose]
    VALIDATION_PURPOSE_CERTIFICATION: _ClassVar[ValidationPurpose]
    VALIDATION_PURPOSE_INVESTIGATION: _ClassVar[ValidationPurpose]

class ValidationStrength(int, metaclass=_enum_type_wrapper.EnumTypeWrapper):
    __slots__ = ()
    VALIDATION_STRENGTH_UNSPECIFIED: _ClassVar[ValidationStrength]
    VALIDATION_STRENGTH_SMOKE: _ClassVar[ValidationStrength]
    VALIDATION_STRENGTH_TARGETED: _ClassVar[ValidationStrength]
    VALIDATION_STRENGTH_COMPREHENSIVE: _ClassVar[ValidationStrength]
    VALIDATION_STRENGTH_CERTIFICATION: _ClassVar[ValidationStrength]

class ReuseMode(int, metaclass=_enum_type_wrapper.EnumTypeWrapper):
    __slots__ = ()
    REUSE_MODE_UNSPECIFIED: _ClassVar[ReuseMode]
    REUSE_MODE_NEVER: _ClassVar[ReuseMode]
    REUSE_MODE_ATTACH_ACTIVE: _ClassVar[ReuseMode]
    REUSE_MODE_COMPATIBLE_TERMINAL: _ClassVar[ReuseMode]
    REUSE_MODE_ATTACH_OR_TERMINAL: _ClassVar[ReuseMode]

class ConcurrencyMode(int, metaclass=_enum_type_wrapper.EnumTypeWrapper):
    __slots__ = ()
    CONCURRENCY_MODE_UNSPECIFIED: _ClassVar[ConcurrencyMode]
    CONCURRENCY_MODE_SHARED_COMPATIBLE: _ClassVar[ConcurrencyMode]
    CONCURRENCY_MODE_EXCLUSIVE: _ClassVar[ConcurrencyMode]

class ReceiptState(int, metaclass=_enum_type_wrapper.EnumTypeWrapper):
    __slots__ = ()
    RECEIPT_STATE_UNSPECIFIED: _ClassVar[ReceiptState]
    RECEIPT_STATE_ADMITTED: _ClassVar[ReceiptState]
    RECEIPT_STATE_ATTACHED: _ClassVar[ReceiptState]
    RECEIPT_STATE_QUEUED: _ClassVar[ReceiptState]
    RECEIPT_STATE_RUNNING: _ClassVar[ReceiptState]
    RECEIPT_STATE_RETRY_PENDING: _ClassVar[ReceiptState]
    RECEIPT_STATE_SUCCEEDED: _ClassVar[ReceiptState]
    RECEIPT_STATE_FAILED: _ClassVar[ReceiptState]
    RECEIPT_STATE_DEGRADED: _ClassVar[ReceiptState]
    RECEIPT_STATE_CANCELLED: _ClassVar[ReceiptState]
    RECEIPT_STATE_SUPERSEDED: _ClassVar[ReceiptState]

class CompatibilityKind(int, metaclass=_enum_type_wrapper.EnumTypeWrapper):
    __slots__ = ()
    COMPATIBILITY_KIND_UNSPECIFIED: _ClassVar[CompatibilityKind]
    COMPATIBILITY_KIND_NEW_WORK: _ClassVar[CompatibilityKind]
    COMPATIBILITY_KIND_ATTACHED_ACTIVE: _ClassVar[CompatibilityKind]
    COMPATIBILITY_KIND_REUSED_TERMINAL: _ClassVar[CompatibilityKind]
    COMPATIBILITY_KIND_INCOMPATIBLE: _ClassVar[CompatibilityKind]

class RetryKind(int, metaclass=_enum_type_wrapper.EnumTypeWrapper):
    __slots__ = ()
    RETRY_KIND_UNSPECIFIED: _ClassVar[RetryKind]
    RETRY_KIND_NOT_NEEDED: _ClassVar[RetryKind]
    RETRY_KIND_PENDING: _ClassVar[RetryKind]
    RETRY_KIND_SCHEDULED: _ClassVar[RetryKind]
    RETRY_KIND_EXHAUSTED: _ClassVar[RetryKind]
    RETRY_KIND_PROHIBITED: _ClassVar[RetryKind]

class ValidationReasonCode(int, metaclass=_enum_type_wrapper.EnumTypeWrapper):
    __slots__ = ()
    VALIDATION_REASON_CODE_UNSPECIFIED: _ClassVar[ValidationReasonCode]
    VALIDATION_REASON_CODE_NONE: _ClassVar[ValidationReasonCode]
    VALIDATION_REASON_CODE_INVALID_INTENT: _ClassVar[ValidationReasonCode]
    VALIDATION_REASON_CODE_INVALID_TARGET: _ClassVar[ValidationReasonCode]
    VALIDATION_REASON_CODE_UNSUPPORTED_STRENGTH: _ClassVar[ValidationReasonCode]
    VALIDATION_REASON_CODE_UNRESOLVED_INPUT: _ClassVar[ValidationReasonCode]
    VALIDATION_REASON_CODE_IDENTITY_CHANGED: _ClassVar[ValidationReasonCode]
    VALIDATION_REASON_CODE_CAPACITY_UNAVAILABLE: _ClassVar[ValidationReasonCode]
    VALIDATION_REASON_CODE_PROVIDER_UNAVAILABLE: _ClassVar[ValidationReasonCode]
    VALIDATION_REASON_CODE_DEADLINE_EXCEEDED: _ClassVar[ValidationReasonCode]
    VALIDATION_REASON_CODE_REQUIRED_EVIDENCE_FAILED: _ClassVar[ValidationReasonCode]
    VALIDATION_REASON_CODE_REQUIRED_EVIDENCE_MISSING: _ClassVar[ValidationReasonCode]
    VALIDATION_REASON_CODE_ABORT_REQUESTED: _ClassVar[ValidationReasonCode]
    VALIDATION_REASON_CODE_ABORTED: _ClassVar[ValidationReasonCode]
    VALIDATION_REASON_CODE_SUPERSEDED: _ClassVar[ValidationReasonCode]
    VALIDATION_REASON_CODE_RETRY_EXHAUSTED: _ClassVar[ValidationReasonCode]
    VALIDATION_REASON_CODE_POLICY_REJECTED: _ClassVar[ValidationReasonCode]
    VALIDATION_REASON_CODE_CURSOR_EXPIRED: _ClassVar[ValidationReasonCode]
    VALIDATION_REASON_CODE_INTERNAL: _ClassVar[ValidationReasonCode]

class ChildOperationKind(int, metaclass=_enum_type_wrapper.EnumTypeWrapper):
    __slots__ = ()
    CHILD_OPERATION_KIND_UNSPECIFIED: _ClassVar[ChildOperationKind]
    CHILD_OPERATION_KIND_TEST_RUN: _ClassVar[ChildOperationKind]
    CHILD_OPERATION_KIND_GCT_BASELINE_COLLECTION: _ClassVar[ChildOperationKind]
    CHILD_OPERATION_KIND_GCT_COLLECTION_DIFF: _ClassVar[ChildOperationKind]
    CHILD_OPERATION_KIND_SOURCE_SNAPSHOT: _ClassVar[ChildOperationKind]

class ChildOperationState(int, metaclass=_enum_type_wrapper.EnumTypeWrapper):
    __slots__ = ()
    CHILD_OPERATION_STATE_UNSPECIFIED: _ClassVar[ChildOperationState]
    CHILD_OPERATION_STATE_PENDING: _ClassVar[ChildOperationState]
    CHILD_OPERATION_STATE_RUNNING: _ClassVar[ChildOperationState]
    CHILD_OPERATION_STATE_SUCCEEDED: _ClassVar[ChildOperationState]
    CHILD_OPERATION_STATE_FAILED: _ClassVar[ChildOperationState]
    CHILD_OPERATION_STATE_CANCELLED: _ClassVar[ChildOperationState]
VALIDATION_PURPOSE_UNSPECIFIED: ValidationPurpose
VALIDATION_PURPOSE_PHASE: ValidationPurpose
VALIDATION_PURPOSE_REGRESSION_BEFORE: ValidationPurpose
VALIDATION_PURPOSE_REGRESSION_CURRENT: ValidationPurpose
VALIDATION_PURPOSE_CERTIFICATION: ValidationPurpose
VALIDATION_PURPOSE_INVESTIGATION: ValidationPurpose
VALIDATION_STRENGTH_UNSPECIFIED: ValidationStrength
VALIDATION_STRENGTH_SMOKE: ValidationStrength
VALIDATION_STRENGTH_TARGETED: ValidationStrength
VALIDATION_STRENGTH_COMPREHENSIVE: ValidationStrength
VALIDATION_STRENGTH_CERTIFICATION: ValidationStrength
REUSE_MODE_UNSPECIFIED: ReuseMode
REUSE_MODE_NEVER: ReuseMode
REUSE_MODE_ATTACH_ACTIVE: ReuseMode
REUSE_MODE_COMPATIBLE_TERMINAL: ReuseMode
REUSE_MODE_ATTACH_OR_TERMINAL: ReuseMode
CONCURRENCY_MODE_UNSPECIFIED: ConcurrencyMode
CONCURRENCY_MODE_SHARED_COMPATIBLE: ConcurrencyMode
CONCURRENCY_MODE_EXCLUSIVE: ConcurrencyMode
RECEIPT_STATE_UNSPECIFIED: ReceiptState
RECEIPT_STATE_ADMITTED: ReceiptState
RECEIPT_STATE_ATTACHED: ReceiptState
RECEIPT_STATE_QUEUED: ReceiptState
RECEIPT_STATE_RUNNING: ReceiptState
RECEIPT_STATE_RETRY_PENDING: ReceiptState
RECEIPT_STATE_SUCCEEDED: ReceiptState
RECEIPT_STATE_FAILED: ReceiptState
RECEIPT_STATE_DEGRADED: ReceiptState
RECEIPT_STATE_CANCELLED: ReceiptState
RECEIPT_STATE_SUPERSEDED: ReceiptState
COMPATIBILITY_KIND_UNSPECIFIED: CompatibilityKind
COMPATIBILITY_KIND_NEW_WORK: CompatibilityKind
COMPATIBILITY_KIND_ATTACHED_ACTIVE: CompatibilityKind
COMPATIBILITY_KIND_REUSED_TERMINAL: CompatibilityKind
COMPATIBILITY_KIND_INCOMPATIBLE: CompatibilityKind
RETRY_KIND_UNSPECIFIED: RetryKind
RETRY_KIND_NOT_NEEDED: RetryKind
RETRY_KIND_PENDING: RetryKind
RETRY_KIND_SCHEDULED: RetryKind
RETRY_KIND_EXHAUSTED: RetryKind
RETRY_KIND_PROHIBITED: RetryKind
VALIDATION_REASON_CODE_UNSPECIFIED: ValidationReasonCode
VALIDATION_REASON_CODE_NONE: ValidationReasonCode
VALIDATION_REASON_CODE_INVALID_INTENT: ValidationReasonCode
VALIDATION_REASON_CODE_INVALID_TARGET: ValidationReasonCode
VALIDATION_REASON_CODE_UNSUPPORTED_STRENGTH: ValidationReasonCode
VALIDATION_REASON_CODE_UNRESOLVED_INPUT: ValidationReasonCode
VALIDATION_REASON_CODE_IDENTITY_CHANGED: ValidationReasonCode
VALIDATION_REASON_CODE_CAPACITY_UNAVAILABLE: ValidationReasonCode
VALIDATION_REASON_CODE_PROVIDER_UNAVAILABLE: ValidationReasonCode
VALIDATION_REASON_CODE_DEADLINE_EXCEEDED: ValidationReasonCode
VALIDATION_REASON_CODE_REQUIRED_EVIDENCE_FAILED: ValidationReasonCode
VALIDATION_REASON_CODE_REQUIRED_EVIDENCE_MISSING: ValidationReasonCode
VALIDATION_REASON_CODE_ABORT_REQUESTED: ValidationReasonCode
VALIDATION_REASON_CODE_ABORTED: ValidationReasonCode
VALIDATION_REASON_CODE_SUPERSEDED: ValidationReasonCode
VALIDATION_REASON_CODE_RETRY_EXHAUSTED: ValidationReasonCode
VALIDATION_REASON_CODE_POLICY_REJECTED: ValidationReasonCode
VALIDATION_REASON_CODE_CURSOR_EXPIRED: ValidationReasonCode
VALIDATION_REASON_CODE_INTERNAL: ValidationReasonCode
CHILD_OPERATION_KIND_UNSPECIFIED: ChildOperationKind
CHILD_OPERATION_KIND_TEST_RUN: ChildOperationKind
CHILD_OPERATION_KIND_GCT_BASELINE_COLLECTION: ChildOperationKind
CHILD_OPERATION_KIND_GCT_COLLECTION_DIFF: ChildOperationKind
CHILD_OPERATION_KIND_SOURCE_SNAPSHOT: ChildOperationKind
CHILD_OPERATION_STATE_UNSPECIFIED: ChildOperationState
CHILD_OPERATION_STATE_PENDING: ChildOperationState
CHILD_OPERATION_STATE_RUNNING: ChildOperationState
CHILD_OPERATION_STATE_SUCCEEDED: ChildOperationState
CHILD_OPERATION_STATE_FAILED: ChildOperationState
CHILD_OPERATION_STATE_CANCELLED: ChildOperationState

class ReusePolicy(_message.Message):
    __slots__ = ("mode", "maximum_age")
    MODE_FIELD_NUMBER: _ClassVar[int]
    MAXIMUM_AGE_FIELD_NUMBER: _ClassVar[int]
    mode: ReuseMode
    maximum_age: _duration_pb2.Duration
    def __init__(self, mode: _Optional[_Union[ReuseMode, str]] = ..., maximum_age: _Optional[_Union[datetime.timedelta, _duration_pb2.Duration, _Mapping]] = ...) -> None: ...

class ConcurrencyPolicy(_message.Message):
    __slots__ = ("mode", "maximum_parallelism")
    MODE_FIELD_NUMBER: _ClassVar[int]
    MAXIMUM_PARALLELISM_FIELD_NUMBER: _ClassVar[int]
    mode: ConcurrencyMode
    maximum_parallelism: int
    def __init__(self, mode: _Optional[_Union[ConcurrencyMode, str]] = ..., maximum_parallelism: _Optional[int] = ...) -> None: ...

class DeadlinePolicy(_message.Message):
    __slots__ = ("queue_budget", "execution_budget", "maximum_attempts")
    QUEUE_BUDGET_FIELD_NUMBER: _ClassVar[int]
    EXECUTION_BUDGET_FIELD_NUMBER: _ClassVar[int]
    MAXIMUM_ATTEMPTS_FIELD_NUMBER: _ClassVar[int]
    queue_budget: _duration_pb2.Duration
    execution_budget: _duration_pb2.Duration
    maximum_attempts: int
    def __init__(self, queue_budget: _Optional[_Union[datetime.timedelta, _duration_pb2.Duration, _Mapping]] = ..., execution_budget: _Optional[_Union[datetime.timedelta, _duration_pb2.Duration, _Mapping]] = ..., maximum_attempts: _Optional[int] = ...) -> None: ...

class EvidencePolicy(_message.Message):
    __slots__ = ("require_behavioral_before", "require_source_snapshot", "allow_authorized_degradation", "required_evidence_kinds")
    REQUIRE_BEHAVIORAL_BEFORE_FIELD_NUMBER: _ClassVar[int]
    REQUIRE_SOURCE_SNAPSHOT_FIELD_NUMBER: _ClassVar[int]
    ALLOW_AUTHORIZED_DEGRADATION_FIELD_NUMBER: _ClassVar[int]
    REQUIRED_EVIDENCE_KINDS_FIELD_NUMBER: _ClassVar[int]
    require_behavioral_before: bool
    require_source_snapshot: bool
    allow_authorized_degradation: bool
    required_evidence_kinds: _containers.RepeatedScalarFieldContainer[str]
    def __init__(self, require_behavioral_before: _Optional[bool] = ..., require_source_snapshot: _Optional[bool] = ..., allow_authorized_degradation: _Optional[bool] = ..., required_evidence_kinds: _Optional[_Iterable[str]] = ...) -> None: ...

class ContentRootIdentity(_message.Message):
    __slots__ = ("name", "identity", "files")
    NAME_FIELD_NUMBER: _ClassVar[int]
    IDENTITY_FIELD_NUMBER: _ClassVar[int]
    FILES_FIELD_NUMBER: _ClassVar[int]
    name: str
    identity: str
    files: _containers.RepeatedCompositeFieldContainer[ContentFileIdentity]
    def __init__(self, name: _Optional[str] = ..., identity: _Optional[str] = ..., files: _Optional[_Iterable[_Union[ContentFileIdentity, _Mapping]]] = ...) -> None: ...

class ContentFileIdentity(_message.Message):
    __slots__ = ("path", "digest", "size")
    PATH_FIELD_NUMBER: _ClassVar[int]
    DIGEST_FIELD_NUMBER: _ClassVar[int]
    SIZE_FIELD_NUMBER: _ClassVar[int]
    path: str
    digest: str
    size: int
    def __init__(self, path: _Optional[str] = ..., digest: _Optional[str] = ..., size: _Optional[int] = ...) -> None: ...

class SourceIdentity(_message.Message):
    __slots__ = ("schema_version", "identity", "roots", "configuration", "toolchain", "commit", "branch", "dirty")
    class ConfigurationEntry(_message.Message):
        __slots__ = ("key", "value")
        KEY_FIELD_NUMBER: _ClassVar[int]
        VALUE_FIELD_NUMBER: _ClassVar[int]
        key: str
        value: str
        def __init__(self, key: _Optional[str] = ..., value: _Optional[str] = ...) -> None: ...
    class ToolchainEntry(_message.Message):
        __slots__ = ("key", "value")
        KEY_FIELD_NUMBER: _ClassVar[int]
        VALUE_FIELD_NUMBER: _ClassVar[int]
        key: str
        value: str
        def __init__(self, key: _Optional[str] = ..., value: _Optional[str] = ...) -> None: ...
    SCHEMA_VERSION_FIELD_NUMBER: _ClassVar[int]
    IDENTITY_FIELD_NUMBER: _ClassVar[int]
    ROOTS_FIELD_NUMBER: _ClassVar[int]
    CONFIGURATION_FIELD_NUMBER: _ClassVar[int]
    TOOLCHAIN_FIELD_NUMBER: _ClassVar[int]
    COMMIT_FIELD_NUMBER: _ClassVar[int]
    BRANCH_FIELD_NUMBER: _ClassVar[int]
    DIRTY_FIELD_NUMBER: _ClassVar[int]
    schema_version: int
    identity: str
    roots: _containers.RepeatedCompositeFieldContainer[ContentRootIdentity]
    configuration: _containers.ScalarMap[str, str]
    toolchain: _containers.ScalarMap[str, str]
    commit: str
    branch: str
    dirty: bool
    def __init__(self, schema_version: _Optional[int] = ..., identity: _Optional[str] = ..., roots: _Optional[_Iterable[_Union[ContentRootIdentity, _Mapping]]] = ..., configuration: _Optional[_Mapping[str, str]] = ..., toolchain: _Optional[_Mapping[str, str]] = ..., commit: _Optional[str] = ..., branch: _Optional[str] = ..., dirty: _Optional[bool] = ...) -> None: ...

class InputSelection(_message.Message):
    __slots__ = ("glob", "required")
    GLOB_FIELD_NUMBER: _ClassVar[int]
    REQUIRED_FIELD_NUMBER: _ClassVar[int]
    glob: str
    required: bool
    def __init__(self, glob: _Optional[str] = ..., required: _Optional[bool] = ...) -> None: ...

class ContentInputRoot(_message.Message):
    __slots__ = ("name", "root", "selections", "dependency")
    NAME_FIELD_NUMBER: _ClassVar[int]
    ROOT_FIELD_NUMBER: _ClassVar[int]
    SELECTIONS_FIELD_NUMBER: _ClassVar[int]
    DEPENDENCY_FIELD_NUMBER: _ClassVar[int]
    name: str
    root: str
    selections: _containers.RepeatedCompositeFieldContainer[InputSelection]
    dependency: bool
    def __init__(self, name: _Optional[str] = ..., root: _Optional[str] = ..., selections: _Optional[_Iterable[_Union[InputSelection, _Mapping]]] = ..., dependency: _Optional[bool] = ...) -> None: ...

class ValidationIntent(_message.Message):
    __slots__ = ("schema_version", "intent_id", "idempotency_key", "caller_scenario", "caller_execution_id", "plan_id", "phase_id", "targets", "purpose", "required_strength", "reuse_policy", "concurrency_policy", "expected_identity", "evidence_policy", "deadline_policy", "caller_attributes", "content_inputs", "behavioral_prior", "phases")
    class CallerAttributesEntry(_message.Message):
        __slots__ = ("key", "value")
        KEY_FIELD_NUMBER: _ClassVar[int]
        VALUE_FIELD_NUMBER: _ClassVar[int]
        key: str
        value: str
        def __init__(self, key: _Optional[str] = ..., value: _Optional[str] = ...) -> None: ...
    SCHEMA_VERSION_FIELD_NUMBER: _ClassVar[int]
    INTENT_ID_FIELD_NUMBER: _ClassVar[int]
    IDEMPOTENCY_KEY_FIELD_NUMBER: _ClassVar[int]
    CALLER_SCENARIO_FIELD_NUMBER: _ClassVar[int]
    CALLER_EXECUTION_ID_FIELD_NUMBER: _ClassVar[int]
    PLAN_ID_FIELD_NUMBER: _ClassVar[int]
    PHASE_ID_FIELD_NUMBER: _ClassVar[int]
    TARGETS_FIELD_NUMBER: _ClassVar[int]
    PURPOSE_FIELD_NUMBER: _ClassVar[int]
    REQUIRED_STRENGTH_FIELD_NUMBER: _ClassVar[int]
    REUSE_POLICY_FIELD_NUMBER: _ClassVar[int]
    CONCURRENCY_POLICY_FIELD_NUMBER: _ClassVar[int]
    EXPECTED_IDENTITY_FIELD_NUMBER: _ClassVar[int]
    EVIDENCE_POLICY_FIELD_NUMBER: _ClassVar[int]
    DEADLINE_POLICY_FIELD_NUMBER: _ClassVar[int]
    CALLER_ATTRIBUTES_FIELD_NUMBER: _ClassVar[int]
    CONTENT_INPUTS_FIELD_NUMBER: _ClassVar[int]
    BEHAVIORAL_PRIOR_FIELD_NUMBER: _ClassVar[int]
    PHASES_FIELD_NUMBER: _ClassVar[int]
    schema_version: int
    intent_id: str
    idempotency_key: str
    caller_scenario: str
    caller_execution_id: str
    plan_id: str
    phase_id: str
    targets: _containers.RepeatedCompositeFieldContainer[_validation_target_pb2.ValidationTarget]
    purpose: ValidationPurpose
    required_strength: ValidationStrength
    reuse_policy: ReusePolicy
    concurrency_policy: ConcurrencyPolicy
    expected_identity: SourceIdentity
    evidence_policy: EvidencePolicy
    deadline_policy: DeadlinePolicy
    caller_attributes: _containers.ScalarMap[str, str]
    content_inputs: _containers.RepeatedCompositeFieldContainer[ContentInputRoot]
    behavioral_prior: str
    phases: _containers.RepeatedScalarFieldContainer[str]
    def __init__(self, schema_version: _Optional[int] = ..., intent_id: _Optional[str] = ..., idempotency_key: _Optional[str] = ..., caller_scenario: _Optional[str] = ..., caller_execution_id: _Optional[str] = ..., plan_id: _Optional[str] = ..., phase_id: _Optional[str] = ..., targets: _Optional[_Iterable[_Union[_validation_target_pb2.ValidationTarget, _Mapping]]] = ..., purpose: _Optional[_Union[ValidationPurpose, str]] = ..., required_strength: _Optional[_Union[ValidationStrength, str]] = ..., reuse_policy: _Optional[_Union[ReusePolicy, _Mapping]] = ..., concurrency_policy: _Optional[_Union[ConcurrencyPolicy, _Mapping]] = ..., expected_identity: _Optional[_Union[SourceIdentity, _Mapping]] = ..., evidence_policy: _Optional[_Union[EvidencePolicy, _Mapping]] = ..., deadline_policy: _Optional[_Union[DeadlinePolicy, _Mapping]] = ..., caller_attributes: _Optional[_Mapping[str, str]] = ..., content_inputs: _Optional[_Iterable[_Union[ContentInputRoot, _Mapping]]] = ..., behavioral_prior: _Optional[str] = ..., phases: _Optional[_Iterable[str]] = ...) -> None: ...

class EvidenceReference(_message.Message):
    __slots__ = ("evidence_id", "kind", "owner", "subject_id", "uri", "digest")
    EVIDENCE_ID_FIELD_NUMBER: _ClassVar[int]
    KIND_FIELD_NUMBER: _ClassVar[int]
    OWNER_FIELD_NUMBER: _ClassVar[int]
    SUBJECT_ID_FIELD_NUMBER: _ClassVar[int]
    URI_FIELD_NUMBER: _ClassVar[int]
    DIGEST_FIELD_NUMBER: _ClassVar[int]
    evidence_id: str
    kind: str
    owner: str
    subject_id: str
    uri: str
    digest: str
    def __init__(self, evidence_id: _Optional[str] = ..., kind: _Optional[str] = ..., owner: _Optional[str] = ..., subject_id: _Optional[str] = ..., uri: _Optional[str] = ..., digest: _Optional[str] = ...) -> None: ...

class ChildOperation(_message.Message):
    __slots__ = ("child_id", "kind", "state", "owner", "operation_id", "reason_code", "detail")
    CHILD_ID_FIELD_NUMBER: _ClassVar[int]
    KIND_FIELD_NUMBER: _ClassVar[int]
    STATE_FIELD_NUMBER: _ClassVar[int]
    OWNER_FIELD_NUMBER: _ClassVar[int]
    OPERATION_ID_FIELD_NUMBER: _ClassVar[int]
    REASON_CODE_FIELD_NUMBER: _ClassVar[int]
    DETAIL_FIELD_NUMBER: _ClassVar[int]
    child_id: str
    kind: ChildOperationKind
    state: ChildOperationState
    owner: str
    operation_id: str
    reason_code: ValidationReasonCode
    detail: str
    def __init__(self, child_id: _Optional[str] = ..., kind: _Optional[_Union[ChildOperationKind, str]] = ..., state: _Optional[_Union[ChildOperationState, str]] = ..., owner: _Optional[str] = ..., operation_id: _Optional[str] = ..., reason_code: _Optional[_Union[ValidationReasonCode, str]] = ..., detail: _Optional[str] = ...) -> None: ...

class CompatibilityDecision(_message.Message):
    __slots__ = ("kind", "compatible_receipt_id", "execution_key", "reason_code", "detail")
    KIND_FIELD_NUMBER: _ClassVar[int]
    COMPATIBLE_RECEIPT_ID_FIELD_NUMBER: _ClassVar[int]
    EXECUTION_KEY_FIELD_NUMBER: _ClassVar[int]
    REASON_CODE_FIELD_NUMBER: _ClassVar[int]
    DETAIL_FIELD_NUMBER: _ClassVar[int]
    kind: CompatibilityKind
    compatible_receipt_id: str
    execution_key: str
    reason_code: ValidationReasonCode
    detail: str
    def __init__(self, kind: _Optional[_Union[CompatibilityKind, str]] = ..., compatible_receipt_id: _Optional[str] = ..., execution_key: _Optional[str] = ..., reason_code: _Optional[_Union[ValidationReasonCode, str]] = ..., detail: _Optional[str] = ...) -> None: ...

class RetryDisposition(_message.Message):
    __slots__ = ("kind", "attempt", "maximum_attempts", "reason_code", "retry_at")
    KIND_FIELD_NUMBER: _ClassVar[int]
    ATTEMPT_FIELD_NUMBER: _ClassVar[int]
    MAXIMUM_ATTEMPTS_FIELD_NUMBER: _ClassVar[int]
    REASON_CODE_FIELD_NUMBER: _ClassVar[int]
    RETRY_AT_FIELD_NUMBER: _ClassVar[int]
    kind: RetryKind
    attempt: int
    maximum_attempts: int
    reason_code: ValidationReasonCode
    retry_at: _timestamp_pb2.Timestamp
    def __init__(self, kind: _Optional[_Union[RetryKind, str]] = ..., attempt: _Optional[int] = ..., maximum_attempts: _Optional[int] = ..., reason_code: _Optional[_Union[ValidationReasonCode, str]] = ..., retry_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ...) -> None: ...

class Degradation(_message.Message):
    __slots__ = ("authorized", "authorized_by", "authorization_reason", "missing_evidence_kinds", "reason_code")
    AUTHORIZED_FIELD_NUMBER: _ClassVar[int]
    AUTHORIZED_BY_FIELD_NUMBER: _ClassVar[int]
    AUTHORIZATION_REASON_FIELD_NUMBER: _ClassVar[int]
    MISSING_EVIDENCE_KINDS_FIELD_NUMBER: _ClassVar[int]
    REASON_CODE_FIELD_NUMBER: _ClassVar[int]
    authorized: bool
    authorized_by: str
    authorization_reason: str
    missing_evidence_kinds: _containers.RepeatedScalarFieldContainer[str]
    reason_code: ValidationReasonCode
    def __init__(self, authorized: _Optional[bool] = ..., authorized_by: _Optional[str] = ..., authorization_reason: _Optional[str] = ..., missing_evidence_kinds: _Optional[_Iterable[str]] = ..., reason_code: _Optional[_Union[ValidationReasonCode, str]] = ...) -> None: ...

class ValidationReceipt(_message.Message):
    __slots__ = ("schema_version", "receipt_id", "lineage_id", "intent_id", "state", "achieved_strength", "admitted_identity", "observed_identity", "evidence", "children", "compatibility", "retry", "degradation", "reason_code", "detail", "created_at", "updated_at", "terminal_at", "revision")
    SCHEMA_VERSION_FIELD_NUMBER: _ClassVar[int]
    RECEIPT_ID_FIELD_NUMBER: _ClassVar[int]
    LINEAGE_ID_FIELD_NUMBER: _ClassVar[int]
    INTENT_ID_FIELD_NUMBER: _ClassVar[int]
    STATE_FIELD_NUMBER: _ClassVar[int]
    ACHIEVED_STRENGTH_FIELD_NUMBER: _ClassVar[int]
    ADMITTED_IDENTITY_FIELD_NUMBER: _ClassVar[int]
    OBSERVED_IDENTITY_FIELD_NUMBER: _ClassVar[int]
    EVIDENCE_FIELD_NUMBER: _ClassVar[int]
    CHILDREN_FIELD_NUMBER: _ClassVar[int]
    COMPATIBILITY_FIELD_NUMBER: _ClassVar[int]
    RETRY_FIELD_NUMBER: _ClassVar[int]
    DEGRADATION_FIELD_NUMBER: _ClassVar[int]
    REASON_CODE_FIELD_NUMBER: _ClassVar[int]
    DETAIL_FIELD_NUMBER: _ClassVar[int]
    CREATED_AT_FIELD_NUMBER: _ClassVar[int]
    UPDATED_AT_FIELD_NUMBER: _ClassVar[int]
    TERMINAL_AT_FIELD_NUMBER: _ClassVar[int]
    REVISION_FIELD_NUMBER: _ClassVar[int]
    schema_version: int
    receipt_id: str
    lineage_id: str
    intent_id: str
    state: ReceiptState
    achieved_strength: ValidationStrength
    admitted_identity: SourceIdentity
    observed_identity: SourceIdentity
    evidence: _containers.RepeatedCompositeFieldContainer[EvidenceReference]
    children: _containers.RepeatedCompositeFieldContainer[ChildOperation]
    compatibility: CompatibilityDecision
    retry: RetryDisposition
    degradation: Degradation
    reason_code: ValidationReasonCode
    detail: str
    created_at: _timestamp_pb2.Timestamp
    updated_at: _timestamp_pb2.Timestamp
    terminal_at: _timestamp_pb2.Timestamp
    revision: int
    def __init__(self, schema_version: _Optional[int] = ..., receipt_id: _Optional[str] = ..., lineage_id: _Optional[str] = ..., intent_id: _Optional[str] = ..., state: _Optional[_Union[ReceiptState, str]] = ..., achieved_strength: _Optional[_Union[ValidationStrength, str]] = ..., admitted_identity: _Optional[_Union[SourceIdentity, _Mapping]] = ..., observed_identity: _Optional[_Union[SourceIdentity, _Mapping]] = ..., evidence: _Optional[_Iterable[_Union[EvidenceReference, _Mapping]]] = ..., children: _Optional[_Iterable[_Union[ChildOperation, _Mapping]]] = ..., compatibility: _Optional[_Union[CompatibilityDecision, _Mapping]] = ..., retry: _Optional[_Union[RetryDisposition, _Mapping]] = ..., degradation: _Optional[_Union[Degradation, _Mapping]] = ..., reason_code: _Optional[_Union[ValidationReasonCode, str]] = ..., detail: _Optional[str] = ..., created_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., updated_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., terminal_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., revision: _Optional[int] = ...) -> None: ...

class CreateValidationRequest(_message.Message):
    __slots__ = ("intent",)
    INTENT_FIELD_NUMBER: _ClassVar[int]
    intent: ValidationIntent
    def __init__(self, intent: _Optional[_Union[ValidationIntent, _Mapping]] = ...) -> None: ...

class CreateValidationResponse(_message.Message):
    __slots__ = ("receipt",)
    RECEIPT_FIELD_NUMBER: _ClassVar[int]
    receipt: ValidationReceipt
    def __init__(self, receipt: _Optional[_Union[ValidationReceipt, _Mapping]] = ...) -> None: ...

class GetValidationRequest(_message.Message):
    __slots__ = ("receipt_id",)
    RECEIPT_ID_FIELD_NUMBER: _ClassVar[int]
    receipt_id: str
    def __init__(self, receipt_id: _Optional[str] = ...) -> None: ...

class GetValidationResponse(_message.Message):
    __slots__ = ("receipt",)
    RECEIPT_FIELD_NUMBER: _ClassVar[int]
    receipt: ValidationReceipt
    def __init__(self, receipt: _Optional[_Union[ValidationReceipt, _Mapping]] = ...) -> None: ...

class WaitValidationRequest(_message.Message):
    __slots__ = ("receipt_id", "wait_id", "timeout", "after_revision")
    RECEIPT_ID_FIELD_NUMBER: _ClassVar[int]
    WAIT_ID_FIELD_NUMBER: _ClassVar[int]
    TIMEOUT_FIELD_NUMBER: _ClassVar[int]
    AFTER_REVISION_FIELD_NUMBER: _ClassVar[int]
    receipt_id: str
    wait_id: str
    timeout: _duration_pb2.Duration
    after_revision: int
    def __init__(self, receipt_id: _Optional[str] = ..., wait_id: _Optional[str] = ..., timeout: _Optional[_Union[datetime.timedelta, _duration_pb2.Duration, _Mapping]] = ..., after_revision: _Optional[int] = ...) -> None: ...

class WaitValidationResponse(_message.Message):
    __slots__ = ("receipt", "timed_out", "wait_cancelled")
    RECEIPT_FIELD_NUMBER: _ClassVar[int]
    TIMED_OUT_FIELD_NUMBER: _ClassVar[int]
    WAIT_CANCELLED_FIELD_NUMBER: _ClassVar[int]
    receipt: ValidationReceipt
    timed_out: bool
    wait_cancelled: bool
    def __init__(self, receipt: _Optional[_Union[ValidationReceipt, _Mapping]] = ..., timed_out: _Optional[bool] = ..., wait_cancelled: _Optional[bool] = ...) -> None: ...

class ListValidationsRequest(_message.Message):
    __slots__ = ("caller_scenario", "caller_execution_id", "plan_id", "state", "page_size", "page_token")
    CALLER_SCENARIO_FIELD_NUMBER: _ClassVar[int]
    CALLER_EXECUTION_ID_FIELD_NUMBER: _ClassVar[int]
    PLAN_ID_FIELD_NUMBER: _ClassVar[int]
    STATE_FIELD_NUMBER: _ClassVar[int]
    PAGE_SIZE_FIELD_NUMBER: _ClassVar[int]
    PAGE_TOKEN_FIELD_NUMBER: _ClassVar[int]
    caller_scenario: str
    caller_execution_id: str
    plan_id: str
    state: ReceiptState
    page_size: int
    page_token: str
    def __init__(self, caller_scenario: _Optional[str] = ..., caller_execution_id: _Optional[str] = ..., plan_id: _Optional[str] = ..., state: _Optional[_Union[ReceiptState, str]] = ..., page_size: _Optional[int] = ..., page_token: _Optional[str] = ...) -> None: ...

class ListValidationsResponse(_message.Message):
    __slots__ = ("receipts", "next_page_token")
    RECEIPTS_FIELD_NUMBER: _ClassVar[int]
    NEXT_PAGE_TOKEN_FIELD_NUMBER: _ClassVar[int]
    receipts: _containers.RepeatedCompositeFieldContainer[ValidationReceipt]
    next_page_token: str
    def __init__(self, receipts: _Optional[_Iterable[_Union[ValidationReceipt, _Mapping]]] = ..., next_page_token: _Optional[str] = ...) -> None: ...

class CancelValidationWaitRequest(_message.Message):
    __slots__ = ("receipt_id", "wait_id")
    RECEIPT_ID_FIELD_NUMBER: _ClassVar[int]
    WAIT_ID_FIELD_NUMBER: _ClassVar[int]
    receipt_id: str
    wait_id: str
    def __init__(self, receipt_id: _Optional[str] = ..., wait_id: _Optional[str] = ...) -> None: ...

class CancelValidationWaitResponse(_message.Message):
    __slots__ = ("cancelled", "receipt")
    CANCELLED_FIELD_NUMBER: _ClassVar[int]
    RECEIPT_FIELD_NUMBER: _ClassVar[int]
    cancelled: bool
    receipt: ValidationReceipt
    def __init__(self, cancelled: _Optional[bool] = ..., receipt: _Optional[_Union[ValidationReceipt, _Mapping]] = ...) -> None: ...

class AbortValidationWorkRequest(_message.Message):
    __slots__ = ("receipt_id", "reason", "requested_by")
    RECEIPT_ID_FIELD_NUMBER: _ClassVar[int]
    REASON_FIELD_NUMBER: _ClassVar[int]
    REQUESTED_BY_FIELD_NUMBER: _ClassVar[int]
    receipt_id: str
    reason: str
    requested_by: str
    def __init__(self, receipt_id: _Optional[str] = ..., reason: _Optional[str] = ..., requested_by: _Optional[str] = ...) -> None: ...

class AbortValidationWorkResponse(_message.Message):
    __slots__ = ("receipt",)
    RECEIPT_FIELD_NUMBER: _ClassVar[int]
    receipt: ValidationReceipt
    def __init__(self, receipt: _Optional[_Union[ValidationReceipt, _Mapping]] = ...) -> None: ...

class ExplainValidationRequest(_message.Message):
    __slots__ = ("receipt_id",)
    RECEIPT_ID_FIELD_NUMBER: _ClassVar[int]
    receipt_id: str
    def __init__(self, receipt_id: _Optional[str] = ...) -> None: ...

class ExplainValidationResponse(_message.Message):
    __slots__ = ("receipt", "decisions", "next_actions")
    RECEIPT_FIELD_NUMBER: _ClassVar[int]
    DECISIONS_FIELD_NUMBER: _ClassVar[int]
    NEXT_ACTIONS_FIELD_NUMBER: _ClassVar[int]
    receipt: ValidationReceipt
    decisions: _containers.RepeatedScalarFieldContainer[str]
    next_actions: _containers.RepeatedScalarFieldContainer[str]
    def __init__(self, receipt: _Optional[_Union[ValidationReceipt, _Mapping]] = ..., decisions: _Optional[_Iterable[str]] = ..., next_actions: _Optional[_Iterable[str]] = ...) -> None: ...

class LegacyValidationRecord(_message.Message):
    __slots__ = ("source_kind", "source_id", "caller_scenario", "target_scenario", "state", "evidence")
    SOURCE_KIND_FIELD_NUMBER: _ClassVar[int]
    SOURCE_ID_FIELD_NUMBER: _ClassVar[int]
    CALLER_SCENARIO_FIELD_NUMBER: _ClassVar[int]
    TARGET_SCENARIO_FIELD_NUMBER: _ClassVar[int]
    STATE_FIELD_NUMBER: _ClassVar[int]
    EVIDENCE_FIELD_NUMBER: _ClassVar[int]
    source_kind: str
    source_id: str
    caller_scenario: str
    target_scenario: str
    state: str
    evidence: _containers.RepeatedCompositeFieldContainer[EvidenceReference]
    def __init__(self, source_kind: _Optional[str] = ..., source_id: _Optional[str] = ..., caller_scenario: _Optional[str] = ..., target_scenario: _Optional[str] = ..., state: _Optional[str] = ..., evidence: _Optional[_Iterable[_Union[EvidenceReference, _Mapping]]] = ...) -> None: ...

class MigrateLegacyValidationRequest(_message.Message):
    __slots__ = ("record", "actor")
    RECORD_FIELD_NUMBER: _ClassVar[int]
    ACTOR_FIELD_NUMBER: _ClassVar[int]
    record: LegacyValidationRecord
    actor: str
    def __init__(self, record: _Optional[_Union[LegacyValidationRecord, _Mapping]] = ..., actor: _Optional[str] = ...) -> None: ...

class MigrateLegacyValidationResponse(_message.Message):
    __slots__ = ("receipt", "migrated", "read_only", "reason")
    RECEIPT_FIELD_NUMBER: _ClassVar[int]
    MIGRATED_FIELD_NUMBER: _ClassVar[int]
    READ_ONLY_FIELD_NUMBER: _ClassVar[int]
    REASON_FIELD_NUMBER: _ClassVar[int]
    receipt: ValidationReceipt
    migrated: bool
    read_only: bool
    reason: str
    def __init__(self, receipt: _Optional[_Union[ValidationReceipt, _Mapping]] = ..., migrated: _Optional[bool] = ..., read_only: _Optional[bool] = ..., reason: _Optional[str] = ...) -> None: ...

class RecordValidationShadowRequest(_message.Message):
    __slots__ = ("record", "receipt_id", "actor")
    RECORD_FIELD_NUMBER: _ClassVar[int]
    RECEIPT_ID_FIELD_NUMBER: _ClassVar[int]
    ACTOR_FIELD_NUMBER: _ClassVar[int]
    record: LegacyValidationRecord
    receipt_id: str
    actor: str
    def __init__(self, record: _Optional[_Union[LegacyValidationRecord, _Mapping]] = ..., receipt_id: _Optional[str] = ..., actor: _Optional[str] = ...) -> None: ...

class RecordValidationShadowResponse(_message.Message):
    __slots__ = ("comparison",)
    COMPARISON_FIELD_NUMBER: _ClassVar[int]
    comparison: ValidationShadowComparison
    def __init__(self, comparison: _Optional[_Union[ValidationShadowComparison, _Mapping]] = ...) -> None: ...

class ListValidationShadowsRequest(_message.Message):
    __slots__ = ("page_size",)
    PAGE_SIZE_FIELD_NUMBER: _ClassVar[int]
    page_size: int
    def __init__(self, page_size: _Optional[int] = ...) -> None: ...

class ListValidationShadowsResponse(_message.Message):
    __slots__ = ("comparisons",)
    COMPARISONS_FIELD_NUMBER: _ClassVar[int]
    comparisons: _containers.RepeatedCompositeFieldContainer[ValidationShadowComparison]
    def __init__(self, comparisons: _Optional[_Iterable[_Union[ValidationShadowComparison, _Mapping]]] = ...) -> None: ...

class ValidationShadowComparison(_message.Message):
    __slots__ = ("comparison_id", "source_kind", "source_id", "receipt_id", "receipt_revision", "legacy_state", "receipt_state", "matched", "reason_code", "legacy_evidence_count", "receipt_evidence_count", "observed_at")
    COMPARISON_ID_FIELD_NUMBER: _ClassVar[int]
    SOURCE_KIND_FIELD_NUMBER: _ClassVar[int]
    SOURCE_ID_FIELD_NUMBER: _ClassVar[int]
    RECEIPT_ID_FIELD_NUMBER: _ClassVar[int]
    RECEIPT_REVISION_FIELD_NUMBER: _ClassVar[int]
    LEGACY_STATE_FIELD_NUMBER: _ClassVar[int]
    RECEIPT_STATE_FIELD_NUMBER: _ClassVar[int]
    MATCHED_FIELD_NUMBER: _ClassVar[int]
    REASON_CODE_FIELD_NUMBER: _ClassVar[int]
    LEGACY_EVIDENCE_COUNT_FIELD_NUMBER: _ClassVar[int]
    RECEIPT_EVIDENCE_COUNT_FIELD_NUMBER: _ClassVar[int]
    OBSERVED_AT_FIELD_NUMBER: _ClassVar[int]
    comparison_id: str
    source_kind: str
    source_id: str
    receipt_id: str
    receipt_revision: int
    legacy_state: str
    receipt_state: ReceiptState
    matched: bool
    reason_code: str
    legacy_evidence_count: int
    receipt_evidence_count: int
    observed_at: _timestamp_pb2.Timestamp
    def __init__(self, comparison_id: _Optional[str] = ..., source_kind: _Optional[str] = ..., source_id: _Optional[str] = ..., receipt_id: _Optional[str] = ..., receipt_revision: _Optional[int] = ..., legacy_state: _Optional[str] = ..., receipt_state: _Optional[_Union[ReceiptState, str]] = ..., matched: _Optional[bool] = ..., reason_code: _Optional[str] = ..., legacy_evidence_count: _Optional[int] = ..., receipt_evidence_count: _Optional[int] = ..., observed_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ...) -> None: ...
