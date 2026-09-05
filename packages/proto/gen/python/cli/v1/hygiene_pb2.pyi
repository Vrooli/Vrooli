from cli.v1 import contract_pb2 as _contract_pb2
from cli.v1 import shared_drift_pb2 as _shared_drift_pb2
from google.protobuf.internal import containers as _containers
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Iterable as _Iterable, Mapping as _Mapping
from typing import ClassVar as _ClassVar, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class HygieneReport(_message.Message):
    __slots__ = ("success", "root", "checks", "findings", "actions", "plan_candidates", "fixes_applied", "config_fixes", "contract", "shared_drift", "blocking_failures", "warnings", "plan_reconcile_outcomes")
    SUCCESS_FIELD_NUMBER: _ClassVar[int]
    ROOT_FIELD_NUMBER: _ClassVar[int]
    CHECKS_FIELD_NUMBER: _ClassVar[int]
    FINDINGS_FIELD_NUMBER: _ClassVar[int]
    ACTIONS_FIELD_NUMBER: _ClassVar[int]
    PLAN_CANDIDATES_FIELD_NUMBER: _ClassVar[int]
    FIXES_APPLIED_FIELD_NUMBER: _ClassVar[int]
    CONFIG_FIXES_FIELD_NUMBER: _ClassVar[int]
    CONTRACT_FIELD_NUMBER: _ClassVar[int]
    SHARED_DRIFT_FIELD_NUMBER: _ClassVar[int]
    BLOCKING_FAILURES_FIELD_NUMBER: _ClassVar[int]
    WARNINGS_FIELD_NUMBER: _ClassVar[int]
    PLAN_RECONCILE_OUTCOMES_FIELD_NUMBER: _ClassVar[int]
    success: bool
    root: str
    checks: _containers.RepeatedCompositeFieldContainer[HygieneCheck]
    findings: _containers.RepeatedCompositeFieldContainer[HygieneFinding]
    actions: _containers.RepeatedCompositeFieldContainer[HygieneAction]
    plan_candidates: _containers.RepeatedCompositeFieldContainer[HygienePlanCandidate]
    fixes_applied: _containers.RepeatedCompositeFieldContainer[HygienePlanFix]
    config_fixes: _containers.RepeatedScalarFieldContainer[str]
    contract: _contract_pb2.ContractValidationOutput
    shared_drift: _shared_drift_pb2.SharedDriftReport
    blocking_failures: int
    warnings: int
    plan_reconcile_outcomes: _containers.RepeatedCompositeFieldContainer[HygienePlanReconcileOutcome]
    def __init__(self, success: _Optional[bool] = ..., root: _Optional[str] = ..., checks: _Optional[_Iterable[_Union[HygieneCheck, _Mapping]]] = ..., findings: _Optional[_Iterable[_Union[HygieneFinding, _Mapping]]] = ..., actions: _Optional[_Iterable[_Union[HygieneAction, _Mapping]]] = ..., plan_candidates: _Optional[_Iterable[_Union[HygienePlanCandidate, _Mapping]]] = ..., fixes_applied: _Optional[_Iterable[_Union[HygienePlanFix, _Mapping]]] = ..., config_fixes: _Optional[_Iterable[str]] = ..., contract: _Optional[_Union[_contract_pb2.ContractValidationOutput, _Mapping]] = ..., shared_drift: _Optional[_Union[_shared_drift_pb2.SharedDriftReport, _Mapping]] = ..., blocking_failures: _Optional[int] = ..., warnings: _Optional[int] = ..., plan_reconcile_outcomes: _Optional[_Iterable[_Union[HygienePlanReconcileOutcome, _Mapping]]] = ...) -> None: ...

class HygieneCheck(_message.Message):
    __slots__ = ("name", "passed", "severity", "message")
    NAME_FIELD_NUMBER: _ClassVar[int]
    PASSED_FIELD_NUMBER: _ClassVar[int]
    SEVERITY_FIELD_NUMBER: _ClassVar[int]
    MESSAGE_FIELD_NUMBER: _ClassVar[int]
    name: str
    passed: bool
    severity: str
    message: str
    def __init__(self, name: _Optional[str] = ..., passed: _Optional[bool] = ..., severity: _Optional[str] = ..., message: _Optional[str] = ...) -> None: ...

class HygieneFinding(_message.Message):
    __slots__ = ("severity", "code", "path", "locations", "message", "why", "fixability", "next_actions")
    SEVERITY_FIELD_NUMBER: _ClassVar[int]
    CODE_FIELD_NUMBER: _ClassVar[int]
    PATH_FIELD_NUMBER: _ClassVar[int]
    LOCATIONS_FIELD_NUMBER: _ClassVar[int]
    MESSAGE_FIELD_NUMBER: _ClassVar[int]
    WHY_FIELD_NUMBER: _ClassVar[int]
    FIXABILITY_FIELD_NUMBER: _ClassVar[int]
    NEXT_ACTIONS_FIELD_NUMBER: _ClassVar[int]
    severity: str
    code: str
    path: str
    locations: _containers.RepeatedScalarFieldContainer[str]
    message: str
    why: str
    fixability: str
    next_actions: _containers.RepeatedCompositeFieldContainer[HygieneAction]
    def __init__(self, severity: _Optional[str] = ..., code: _Optional[str] = ..., path: _Optional[str] = ..., locations: _Optional[_Iterable[str]] = ..., message: _Optional[str] = ..., why: _Optional[str] = ..., fixability: _Optional[str] = ..., next_actions: _Optional[_Iterable[_Union[HygieneAction, _Mapping]]] = ...) -> None: ...

class HygieneAction(_message.Message):
    __slots__ = ("code", "message", "command", "fixability")
    CODE_FIELD_NUMBER: _ClassVar[int]
    MESSAGE_FIELD_NUMBER: _ClassVar[int]
    COMMAND_FIELD_NUMBER: _ClassVar[int]
    FIXABILITY_FIELD_NUMBER: _ClassVar[int]
    code: str
    message: str
    command: str
    fixability: str
    def __init__(self, code: _Optional[str] = ..., message: _Optional[str] = ..., command: _Optional[str] = ..., fixability: _Optional[str] = ...) -> None: ...

class HygienePlanCandidate(_message.Message):
    __slots__ = ("path", "status", "reason")
    PATH_FIELD_NUMBER: _ClassVar[int]
    STATUS_FIELD_NUMBER: _ClassVar[int]
    REASON_FIELD_NUMBER: _ClassVar[int]
    path: str
    status: str
    reason: str
    def __init__(self, path: _Optional[str] = ..., status: _Optional[str] = ..., reason: _Optional[str] = ...) -> None: ...

class HygienePlanFix(_message.Message):
    __slots__ = ("source", "plan", "action", "mirror")
    SOURCE_FIELD_NUMBER: _ClassVar[int]
    PLAN_FIELD_NUMBER: _ClassVar[int]
    ACTION_FIELD_NUMBER: _ClassVar[int]
    MIRROR_FIELD_NUMBER: _ClassVar[int]
    source: str
    plan: HygienePlanRecord
    action: str
    mirror: HygieneMirror
    def __init__(self, source: _Optional[str] = ..., plan: _Optional[_Union[HygienePlanRecord, _Mapping]] = ..., action: _Optional[str] = ..., mirror: _Optional[_Union[HygieneMirror, _Mapping]] = ...) -> None: ...

class HygienePlanReconcileOutcome(_message.Message):
    __slots__ = ("action", "source", "plan", "mirror", "source_untouched", "error", "source_retirement_planned", "source_removed")
    ACTION_FIELD_NUMBER: _ClassVar[int]
    SOURCE_FIELD_NUMBER: _ClassVar[int]
    PLAN_FIELD_NUMBER: _ClassVar[int]
    MIRROR_FIELD_NUMBER: _ClassVar[int]
    SOURCE_UNTOUCHED_FIELD_NUMBER: _ClassVar[int]
    ERROR_FIELD_NUMBER: _ClassVar[int]
    SOURCE_RETIREMENT_PLANNED_FIELD_NUMBER: _ClassVar[int]
    SOURCE_REMOVED_FIELD_NUMBER: _ClassVar[int]
    action: str
    source: str
    plan: HygienePlanRecord
    mirror: HygieneMirror
    source_untouched: bool
    error: str
    source_retirement_planned: bool
    source_removed: bool
    def __init__(self, action: _Optional[str] = ..., source: _Optional[str] = ..., plan: _Optional[_Union[HygienePlanRecord, _Mapping]] = ..., mirror: _Optional[_Union[HygieneMirror, _Mapping]] = ..., source_untouched: _Optional[bool] = ..., error: _Optional[str] = ..., source_retirement_planned: _Optional[bool] = ..., source_removed: _Optional[bool] = ...) -> None: ...

class HygieneMirror(_message.Message):
    __slots__ = ("path", "status")
    PATH_FIELD_NUMBER: _ClassVar[int]
    STATUS_FIELD_NUMBER: _ClassVar[int]
    path: str
    status: str
    def __init__(self, path: _Optional[str] = ..., status: _Optional[str] = ...) -> None: ...

class HygienePlanRecord(_message.Message):
    __slots__ = ("id", "title", "slug", "path", "created_at", "updated_at", "archived", "archived_at", "source_path", "content_hash")
    ID_FIELD_NUMBER: _ClassVar[int]
    TITLE_FIELD_NUMBER: _ClassVar[int]
    SLUG_FIELD_NUMBER: _ClassVar[int]
    PATH_FIELD_NUMBER: _ClassVar[int]
    CREATED_AT_FIELD_NUMBER: _ClassVar[int]
    UPDATED_AT_FIELD_NUMBER: _ClassVar[int]
    ARCHIVED_FIELD_NUMBER: _ClassVar[int]
    ARCHIVED_AT_FIELD_NUMBER: _ClassVar[int]
    SOURCE_PATH_FIELD_NUMBER: _ClassVar[int]
    CONTENT_HASH_FIELD_NUMBER: _ClassVar[int]
    id: str
    title: str
    slug: str
    path: str
    created_at: str
    updated_at: str
    archived: bool
    archived_at: str
    source_path: str
    content_hash: str
    def __init__(self, id: _Optional[str] = ..., title: _Optional[str] = ..., slug: _Optional[str] = ..., path: _Optional[str] = ..., created_at: _Optional[str] = ..., updated_at: _Optional[str] = ..., archived: _Optional[bool] = ..., archived_at: _Optional[str] = ..., source_path: _Optional[str] = ..., content_hash: _Optional[str] = ...) -> None: ...
