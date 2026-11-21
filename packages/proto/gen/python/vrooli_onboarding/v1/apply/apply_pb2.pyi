import datetime

from buf.validate import validate_pb2 as _validate_pb2
from google.protobuf import timestamp_pb2 as _timestamp_pb2
from vrooli_onboarding.v1.shared import shared_pb2 as _shared_pb2
from google.protobuf.internal import containers as _containers
from google.protobuf.internal import enum_type_wrapper as _enum_type_wrapper
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Iterable as _Iterable, Mapping as _Mapping
from typing import ClassVar as _ClassVar, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class ApplyRunState(int, metaclass=_enum_type_wrapper.EnumTypeWrapper):
    __slots__ = ()
    APPLY_RUN_STATE_UNSPECIFIED: _ClassVar[ApplyRunState]
    APPLY_RUN_STATE_PENDING: _ClassVar[ApplyRunState]
    APPLY_RUN_STATE_APPLYING: _ClassVar[ApplyRunState]
    APPLY_RUN_STATE_APPLIED: _ClassVar[ApplyRunState]
    APPLY_RUN_STATE_ALREADY_SATISFIED: _ClassVar[ApplyRunState]
    APPLY_RUN_STATE_PARTIALLY_APPLIED: _ClassVar[ApplyRunState]
    APPLY_RUN_STATE_CONFIGURATION_INCOMPLETE: _ClassVar[ApplyRunState]
    APPLY_RUN_STATE_FAILED: _ClassVar[ApplyRunState]
    APPLY_RUN_STATE_CANCELLED: _ClassVar[ApplyRunState]
    APPLY_RUN_STATE_INDETERMINATE: _ClassVar[ApplyRunState]

class ApplyStepState(int, metaclass=_enum_type_wrapper.EnumTypeWrapper):
    __slots__ = ()
    APPLY_STEP_STATE_UNSPECIFIED: _ClassVar[ApplyStepState]
    APPLY_STEP_STATE_PENDING: _ClassVar[ApplyStepState]
    APPLY_STEP_STATE_APPLYING: _ClassVar[ApplyStepState]
    APPLY_STEP_STATE_APPLIED: _ClassVar[ApplyStepState]
    APPLY_STEP_STATE_ALREADY_SATISFIED: _ClassVar[ApplyStepState]
    APPLY_STEP_STATE_BLOCKED: _ClassVar[ApplyStepState]
    APPLY_STEP_STATE_FAILED: _ClassVar[ApplyStepState]
    APPLY_STEP_STATE_TIMED_OUT: _ClassVar[ApplyStepState]
    APPLY_STEP_STATE_NEEDS_ELEVATION: _ClassVar[ApplyStepState]
    APPLY_STEP_STATE_NOT_APPLICABLE: _ClassVar[ApplyStepState]
    APPLY_STEP_STATE_SKIPPED_SELF: _ClassVar[ApplyStepState]
APPLY_RUN_STATE_UNSPECIFIED: ApplyRunState
APPLY_RUN_STATE_PENDING: ApplyRunState
APPLY_RUN_STATE_APPLYING: ApplyRunState
APPLY_RUN_STATE_APPLIED: ApplyRunState
APPLY_RUN_STATE_ALREADY_SATISFIED: ApplyRunState
APPLY_RUN_STATE_PARTIALLY_APPLIED: ApplyRunState
APPLY_RUN_STATE_CONFIGURATION_INCOMPLETE: ApplyRunState
APPLY_RUN_STATE_FAILED: ApplyRunState
APPLY_RUN_STATE_CANCELLED: ApplyRunState
APPLY_RUN_STATE_INDETERMINATE: ApplyRunState
APPLY_STEP_STATE_UNSPECIFIED: ApplyStepState
APPLY_STEP_STATE_PENDING: ApplyStepState
APPLY_STEP_STATE_APPLYING: ApplyStepState
APPLY_STEP_STATE_APPLIED: ApplyStepState
APPLY_STEP_STATE_ALREADY_SATISFIED: ApplyStepState
APPLY_STEP_STATE_BLOCKED: ApplyStepState
APPLY_STEP_STATE_FAILED: ApplyStepState
APPLY_STEP_STATE_TIMED_OUT: ApplyStepState
APPLY_STEP_STATE_NEEDS_ELEVATION: ApplyStepState
APPLY_STEP_STATE_NOT_APPLICABLE: ApplyStepState
APPLY_STEP_STATE_SKIPPED_SELF: ApplyStepState

class StartApplyRequest(_message.Message):
    __slots__ = ("target", "plan_id", "plan_digest", "expected_revision", "consent_receipt_id", "idempotency_key")
    TARGET_FIELD_NUMBER: _ClassVar[int]
    PLAN_ID_FIELD_NUMBER: _ClassVar[int]
    PLAN_DIGEST_FIELD_NUMBER: _ClassVar[int]
    EXPECTED_REVISION_FIELD_NUMBER: _ClassVar[int]
    CONSENT_RECEIPT_ID_FIELD_NUMBER: _ClassVar[int]
    IDEMPOTENCY_KEY_FIELD_NUMBER: _ClassVar[int]
    target: str
    plan_id: str
    plan_digest: str
    expected_revision: str
    consent_receipt_id: str
    idempotency_key: str
    def __init__(self, target: _Optional[str] = ..., plan_id: _Optional[str] = ..., plan_digest: _Optional[str] = ..., expected_revision: _Optional[str] = ..., consent_receipt_id: _Optional[str] = ..., idempotency_key: _Optional[str] = ...) -> None: ...

class GetApplyRunRequest(_message.Message):
    __slots__ = ("target", "run_id")
    TARGET_FIELD_NUMBER: _ClassVar[int]
    RUN_ID_FIELD_NUMBER: _ClassVar[int]
    target: str
    run_id: str
    def __init__(self, target: _Optional[str] = ..., run_id: _Optional[str] = ...) -> None: ...

class GetApplyPlanRequest(_message.Message):
    __slots__ = ("target",)
    TARGET_FIELD_NUMBER: _ClassVar[int]
    target: str
    def __init__(self, target: _Optional[str] = ...) -> None: ...

class ApplyItem(_message.Message):
    __slots__ = ("id", "kind", "name", "dependencies", "required", "privileged", "observed_state")
    ID_FIELD_NUMBER: _ClassVar[int]
    KIND_FIELD_NUMBER: _ClassVar[int]
    NAME_FIELD_NUMBER: _ClassVar[int]
    DEPENDENCIES_FIELD_NUMBER: _ClassVar[int]
    REQUIRED_FIELD_NUMBER: _ClassVar[int]
    PRIVILEGED_FIELD_NUMBER: _ClassVar[int]
    OBSERVED_STATE_FIELD_NUMBER: _ClassVar[int]
    id: str
    kind: str
    name: str
    dependencies: _containers.RepeatedScalarFieldContainer[str]
    required: bool
    privileged: bool
    observed_state: str
    def __init__(self, id: _Optional[str] = ..., kind: _Optional[str] = ..., name: _Optional[str] = ..., dependencies: _Optional[_Iterable[str]] = ..., required: _Optional[bool] = ..., privileged: _Optional[bool] = ..., observed_state: _Optional[str] = ...) -> None: ...

class ApplyStep(_message.Message):
    __slots__ = ("id", "kind", "name", "dependencies", "required", "privileged", "observed_state", "state", "legacy_outcome", "disposition", "error", "remediation", "blocked_by", "error_code", "started_at", "completed_at")
    ID_FIELD_NUMBER: _ClassVar[int]
    KIND_FIELD_NUMBER: _ClassVar[int]
    NAME_FIELD_NUMBER: _ClassVar[int]
    DEPENDENCIES_FIELD_NUMBER: _ClassVar[int]
    REQUIRED_FIELD_NUMBER: _ClassVar[int]
    PRIVILEGED_FIELD_NUMBER: _ClassVar[int]
    OBSERVED_STATE_FIELD_NUMBER: _ClassVar[int]
    STATE_FIELD_NUMBER: _ClassVar[int]
    LEGACY_OUTCOME_FIELD_NUMBER: _ClassVar[int]
    DISPOSITION_FIELD_NUMBER: _ClassVar[int]
    ERROR_FIELD_NUMBER: _ClassVar[int]
    REMEDIATION_FIELD_NUMBER: _ClassVar[int]
    BLOCKED_BY_FIELD_NUMBER: _ClassVar[int]
    ERROR_CODE_FIELD_NUMBER: _ClassVar[int]
    STARTED_AT_FIELD_NUMBER: _ClassVar[int]
    COMPLETED_AT_FIELD_NUMBER: _ClassVar[int]
    id: str
    kind: str
    name: str
    dependencies: _containers.RepeatedScalarFieldContainer[str]
    required: bool
    privileged: bool
    observed_state: str
    state: ApplyStepState
    legacy_outcome: str
    disposition: str
    error: str
    remediation: str
    blocked_by: str
    error_code: str
    started_at: _timestamp_pb2.Timestamp
    completed_at: _timestamp_pb2.Timestamp
    def __init__(self, id: _Optional[str] = ..., kind: _Optional[str] = ..., name: _Optional[str] = ..., dependencies: _Optional[_Iterable[str]] = ..., required: _Optional[bool] = ..., privileged: _Optional[bool] = ..., observed_state: _Optional[str] = ..., state: _Optional[_Union[ApplyStepState, str]] = ..., legacy_outcome: _Optional[str] = ..., disposition: _Optional[str] = ..., error: _Optional[str] = ..., remediation: _Optional[str] = ..., blocked_by: _Optional[str] = ..., error_code: _Optional[str] = ..., started_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., completed_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ...) -> None: ...

class GetApplyPlanResponse(_message.Message):
    __slots__ = ("items", "target", "plan_id", "plan_digest", "revision", "expires_at")
    ITEMS_FIELD_NUMBER: _ClassVar[int]
    TARGET_FIELD_NUMBER: _ClassVar[int]
    PLAN_ID_FIELD_NUMBER: _ClassVar[int]
    PLAN_DIGEST_FIELD_NUMBER: _ClassVar[int]
    REVISION_FIELD_NUMBER: _ClassVar[int]
    EXPIRES_AT_FIELD_NUMBER: _ClassVar[int]
    items: _containers.RepeatedCompositeFieldContainer[ApplyItem]
    target: str
    plan_id: str
    plan_digest: str
    revision: str
    expires_at: _timestamp_pb2.Timestamp
    def __init__(self, items: _Optional[_Iterable[_Union[ApplyItem, _Mapping]]] = ..., target: _Optional[str] = ..., plan_id: _Optional[str] = ..., plan_digest: _Optional[str] = ..., revision: _Optional[str] = ..., expires_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ...) -> None: ...

class ReviewApplyRequest(_message.Message):
    __slots__ = ("target", "plan_id", "plan_digest", "expected_revision")
    TARGET_FIELD_NUMBER: _ClassVar[int]
    PLAN_ID_FIELD_NUMBER: _ClassVar[int]
    PLAN_DIGEST_FIELD_NUMBER: _ClassVar[int]
    EXPECTED_REVISION_FIELD_NUMBER: _ClassVar[int]
    target: str
    plan_id: str
    plan_digest: str
    expected_revision: str
    def __init__(self, target: _Optional[str] = ..., plan_id: _Optional[str] = ..., plan_digest: _Optional[str] = ..., expected_revision: _Optional[str] = ...) -> None: ...

class ReviewApplyResponse(_message.Message):
    __slots__ = ("target", "plan_id", "plan_digest", "revision", "consent_receipt_id", "expires_at")
    TARGET_FIELD_NUMBER: _ClassVar[int]
    PLAN_ID_FIELD_NUMBER: _ClassVar[int]
    PLAN_DIGEST_FIELD_NUMBER: _ClassVar[int]
    REVISION_FIELD_NUMBER: _ClassVar[int]
    CONSENT_RECEIPT_ID_FIELD_NUMBER: _ClassVar[int]
    EXPIRES_AT_FIELD_NUMBER: _ClassVar[int]
    target: str
    plan_id: str
    plan_digest: str
    revision: str
    consent_receipt_id: str
    expires_at: _timestamp_pb2.Timestamp
    def __init__(self, target: _Optional[str] = ..., plan_id: _Optional[str] = ..., plan_digest: _Optional[str] = ..., revision: _Optional[str] = ..., consent_receipt_id: _Optional[str] = ..., expires_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ...) -> None: ...

class StartApplyResponse(_message.Message):
    __slots__ = ("run",)
    RUN_FIELD_NUMBER: _ClassVar[int]
    run: GetApplyRunResponse
    def __init__(self, run: _Optional[_Union[GetApplyRunResponse, _Mapping]] = ...) -> None: ...

class CancelApplyRequest(_message.Message):
    __slots__ = ("target", "run_id")
    TARGET_FIELD_NUMBER: _ClassVar[int]
    RUN_ID_FIELD_NUMBER: _ClassVar[int]
    target: str
    run_id: str
    def __init__(self, target: _Optional[str] = ..., run_id: _Optional[str] = ...) -> None: ...

class CancelApplyResponse(_message.Message):
    __slots__ = ("run",)
    RUN_FIELD_NUMBER: _ClassVar[int]
    run: GetApplyRunResponse
    def __init__(self, run: _Optional[_Union[GetApplyRunResponse, _Mapping]] = ...) -> None: ...

class GetApplyRunResponse(_message.Message):
    __slots__ = ("run_id", "status", "legacy_status", "selection_digest", "started_at", "completed_at", "error", "steps", "blockers", "degraded", "degraded_digest", "runner_pid", "heartbeat")
    RUN_ID_FIELD_NUMBER: _ClassVar[int]
    STATUS_FIELD_NUMBER: _ClassVar[int]
    LEGACY_STATUS_FIELD_NUMBER: _ClassVar[int]
    SELECTION_DIGEST_FIELD_NUMBER: _ClassVar[int]
    STARTED_AT_FIELD_NUMBER: _ClassVar[int]
    COMPLETED_AT_FIELD_NUMBER: _ClassVar[int]
    ERROR_FIELD_NUMBER: _ClassVar[int]
    STEPS_FIELD_NUMBER: _ClassVar[int]
    BLOCKERS_FIELD_NUMBER: _ClassVar[int]
    DEGRADED_FIELD_NUMBER: _ClassVar[int]
    DEGRADED_DIGEST_FIELD_NUMBER: _ClassVar[int]
    RUNNER_PID_FIELD_NUMBER: _ClassVar[int]
    HEARTBEAT_FIELD_NUMBER: _ClassVar[int]
    run_id: str
    status: ApplyRunState
    legacy_status: str
    selection_digest: str
    started_at: _timestamp_pb2.Timestamp
    completed_at: _timestamp_pb2.Timestamp
    error: str
    steps: _containers.RepeatedCompositeFieldContainer[ApplyStep]
    blockers: _containers.RepeatedCompositeFieldContainer[_shared_pb2.CompletionBlocker]
    degraded: _containers.RepeatedCompositeFieldContainer[_shared_pb2.CompletionBlocker]
    degraded_digest: str
    runner_pid: int
    heartbeat: _timestamp_pb2.Timestamp
    def __init__(self, run_id: _Optional[str] = ..., status: _Optional[_Union[ApplyRunState, str]] = ..., legacy_status: _Optional[str] = ..., selection_digest: _Optional[str] = ..., started_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., completed_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., error: _Optional[str] = ..., steps: _Optional[_Iterable[_Union[ApplyStep, _Mapping]]] = ..., blockers: _Optional[_Iterable[_Union[_shared_pb2.CompletionBlocker, _Mapping]]] = ..., degraded: _Optional[_Iterable[_Union[_shared_pb2.CompletionBlocker, _Mapping]]] = ..., degraded_digest: _Optional[str] = ..., runner_pid: _Optional[int] = ..., heartbeat: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ...) -> None: ...
