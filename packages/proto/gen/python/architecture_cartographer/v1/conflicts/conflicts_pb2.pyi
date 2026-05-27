import datetime

from architecture_cartographer.v1.signals import signals_pb2 as _signals_pb2
from google.protobuf import timestamp_pb2 as _timestamp_pb2
from google.protobuf.internal import containers as _containers
from google.protobuf.internal import enum_type_wrapper as _enum_type_wrapper
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Iterable as _Iterable, Mapping as _Mapping
from typing import ClassVar as _ClassVar, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class Severity(int, metaclass=_enum_type_wrapper.EnumTypeWrapper):
    __slots__ = ()
    SEVERITY_UNSPECIFIED: _ClassVar[Severity]
    SEVERITY_INFO: _ClassVar[Severity]
    SEVERITY_WARN: _ClassVar[Severity]
    SEVERITY_ERROR: _ClassVar[Severity]
    SEVERITY_BLOCKER: _ClassVar[Severity]

class ResolutionStatus(int, metaclass=_enum_type_wrapper.EnumTypeWrapper):
    __slots__ = ()
    RESOLUTION_STATUS_UNSPECIFIED: _ClassVar[ResolutionStatus]
    RESOLUTION_STATUS_DETECTED: _ClassVar[ResolutionStatus]
    RESOLUTION_STATUS_ASSIGNED: _ClassVar[ResolutionStatus]
    RESOLUTION_STATUS_SPLIT: _ClassVar[ResolutionStatus]
    RESOLUTION_STATUS_RESOLVED: _ClassVar[ResolutionStatus]
    RESOLUTION_STATUS_VALIDATED: _ClassVar[ResolutionStatus]
    RESOLUTION_STATUS_COMMITTED: _ClassVar[ResolutionStatus]
    RESOLUTION_STATUS_FORCE_RESOLVED: _ClassVar[ResolutionStatus]

class FixKind(int, metaclass=_enum_type_wrapper.EnumTypeWrapper):
    __slots__ = ()
    FIX_KIND_UNSPECIFIED: _ClassVar[FixKind]
    FIX_KIND_MOVE_FILE: _ClassVar[FixKind]
    FIX_KIND_REASSIGN_DOMAIN: _ClassVar[FixKind]
    FIX_KIND_BREAK_CYCLE: _ClassVar[FixKind]
    FIX_KIND_ADD_DEPENDENCY: _ClassVar[FixKind]
    FIX_KIND_ADD_TRANSITIONAL: _ClassVar[FixKind]
SEVERITY_UNSPECIFIED: Severity
SEVERITY_INFO: Severity
SEVERITY_WARN: Severity
SEVERITY_ERROR: Severity
SEVERITY_BLOCKER: Severity
RESOLUTION_STATUS_UNSPECIFIED: ResolutionStatus
RESOLUTION_STATUS_DETECTED: ResolutionStatus
RESOLUTION_STATUS_ASSIGNED: ResolutionStatus
RESOLUTION_STATUS_SPLIT: ResolutionStatus
RESOLUTION_STATUS_RESOLVED: ResolutionStatus
RESOLUTION_STATUS_VALIDATED: ResolutionStatus
RESOLUTION_STATUS_COMMITTED: ResolutionStatus
RESOLUTION_STATUS_FORCE_RESOLVED: ResolutionStatus
FIX_KIND_UNSPECIFIED: FixKind
FIX_KIND_MOVE_FILE: FixKind
FIX_KIND_REASSIGN_DOMAIN: FixKind
FIX_KIND_BREAK_CYCLE: FixKind
FIX_KIND_ADD_DEPENDENCY: FixKind
FIX_KIND_ADD_TRANSITIONAL: FixKind

class ConflictEvidence(_message.Message):
    __slots__ = ("kind", "summary", "locator", "payload")
    KIND_FIELD_NUMBER: _ClassVar[int]
    SUMMARY_FIELD_NUMBER: _ClassVar[int]
    LOCATOR_FIELD_NUMBER: _ClassVar[int]
    PAYLOAD_FIELD_NUMBER: _ClassVar[int]
    kind: str
    summary: str
    locator: str
    payload: bytes
    def __init__(self, kind: _Optional[str] = ..., summary: _Optional[str] = ..., locator: _Optional[str] = ..., payload: _Optional[bytes] = ...) -> None: ...

class Fix(_message.Message):
    __slots__ = ("id", "kind", "resolver", "summary", "payload", "confidence")
    ID_FIELD_NUMBER: _ClassVar[int]
    KIND_FIELD_NUMBER: _ClassVar[int]
    RESOLVER_FIELD_NUMBER: _ClassVar[int]
    SUMMARY_FIELD_NUMBER: _ClassVar[int]
    PAYLOAD_FIELD_NUMBER: _ClassVar[int]
    CONFIDENCE_FIELD_NUMBER: _ClassVar[int]
    id: str
    kind: FixKind
    resolver: str
    summary: str
    payload: bytes
    confidence: float
    def __init__(self, id: _Optional[str] = ..., kind: _Optional[_Union[FixKind, str]] = ..., resolver: _Optional[str] = ..., summary: _Optional[str] = ..., payload: _Optional[bytes] = ..., confidence: _Optional[float] = ...) -> None: ...

class Conflict(_message.Message):
    __slots__ = ("id", "scenario", "detector", "type", "subtype", "severity", "locations", "domains", "evidence", "suggested_fixes", "status", "assigned_domain", "resolution_note", "snapshot_id", "verdict", "detected_at", "updated_at", "suppressed", "suppression_reason")
    ID_FIELD_NUMBER: _ClassVar[int]
    SCENARIO_FIELD_NUMBER: _ClassVar[int]
    DETECTOR_FIELD_NUMBER: _ClassVar[int]
    TYPE_FIELD_NUMBER: _ClassVar[int]
    SUBTYPE_FIELD_NUMBER: _ClassVar[int]
    SEVERITY_FIELD_NUMBER: _ClassVar[int]
    LOCATIONS_FIELD_NUMBER: _ClassVar[int]
    DOMAINS_FIELD_NUMBER: _ClassVar[int]
    EVIDENCE_FIELD_NUMBER: _ClassVar[int]
    SUGGESTED_FIXES_FIELD_NUMBER: _ClassVar[int]
    STATUS_FIELD_NUMBER: _ClassVar[int]
    ASSIGNED_DOMAIN_FIELD_NUMBER: _ClassVar[int]
    RESOLUTION_NOTE_FIELD_NUMBER: _ClassVar[int]
    SNAPSHOT_ID_FIELD_NUMBER: _ClassVar[int]
    VERDICT_FIELD_NUMBER: _ClassVar[int]
    DETECTED_AT_FIELD_NUMBER: _ClassVar[int]
    UPDATED_AT_FIELD_NUMBER: _ClassVar[int]
    SUPPRESSED_FIELD_NUMBER: _ClassVar[int]
    SUPPRESSION_REASON_FIELD_NUMBER: _ClassVar[int]
    id: str
    scenario: str
    detector: str
    type: str
    subtype: str
    severity: Severity
    locations: _containers.RepeatedScalarFieldContainer[str]
    domains: _containers.RepeatedScalarFieldContainer[str]
    evidence: _containers.RepeatedCompositeFieldContainer[ConflictEvidence]
    suggested_fixes: _containers.RepeatedCompositeFieldContainer[Fix]
    status: ResolutionStatus
    assigned_domain: str
    resolution_note: str
    snapshot_id: str
    verdict: _signals_pb2.Verdict
    detected_at: _timestamp_pb2.Timestamp
    updated_at: _timestamp_pb2.Timestamp
    suppressed: bool
    suppression_reason: str
    def __init__(self, id: _Optional[str] = ..., scenario: _Optional[str] = ..., detector: _Optional[str] = ..., type: _Optional[str] = ..., subtype: _Optional[str] = ..., severity: _Optional[_Union[Severity, str]] = ..., locations: _Optional[_Iterable[str]] = ..., domains: _Optional[_Iterable[str]] = ..., evidence: _Optional[_Iterable[_Union[ConflictEvidence, _Mapping]]] = ..., suggested_fixes: _Optional[_Iterable[_Union[Fix, _Mapping]]] = ..., status: _Optional[_Union[ResolutionStatus, str]] = ..., assigned_domain: _Optional[str] = ..., resolution_note: _Optional[str] = ..., snapshot_id: _Optional[str] = ..., verdict: _Optional[_Union[_signals_pb2.Verdict, _Mapping]] = ..., detected_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., updated_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., suppressed: _Optional[bool] = ..., suppression_reason: _Optional[str] = ...) -> None: ...

class DetectorDescriptor(_message.Message):
    __slots__ = ("name", "description", "stability", "emits_types")
    NAME_FIELD_NUMBER: _ClassVar[int]
    DESCRIPTION_FIELD_NUMBER: _ClassVar[int]
    STABILITY_FIELD_NUMBER: _ClassVar[int]
    EMITS_TYPES_FIELD_NUMBER: _ClassVar[int]
    name: str
    description: str
    stability: str
    emits_types: _containers.RepeatedScalarFieldContainer[str]
    def __init__(self, name: _Optional[str] = ..., description: _Optional[str] = ..., stability: _Optional[str] = ..., emits_types: _Optional[_Iterable[str]] = ...) -> None: ...

class ResolverDescriptor(_message.Message):
    __slots__ = ("name", "description", "stability", "handles_kinds", "requires_apply")
    NAME_FIELD_NUMBER: _ClassVar[int]
    DESCRIPTION_FIELD_NUMBER: _ClassVar[int]
    STABILITY_FIELD_NUMBER: _ClassVar[int]
    HANDLES_KINDS_FIELD_NUMBER: _ClassVar[int]
    REQUIRES_APPLY_FIELD_NUMBER: _ClassVar[int]
    name: str
    description: str
    stability: str
    handles_kinds: _containers.RepeatedScalarFieldContainer[FixKind]
    requires_apply: bool
    def __init__(self, name: _Optional[str] = ..., description: _Optional[str] = ..., stability: _Optional[str] = ..., handles_kinds: _Optional[_Iterable[_Union[FixKind, str]]] = ..., requires_apply: _Optional[bool] = ...) -> None: ...

class DetectConflictsRequest(_message.Message):
    __slots__ = ("scenario", "snapshot_id", "idempotency_key")
    SCENARIO_FIELD_NUMBER: _ClassVar[int]
    SNAPSHOT_ID_FIELD_NUMBER: _ClassVar[int]
    IDEMPOTENCY_KEY_FIELD_NUMBER: _ClassVar[int]
    scenario: str
    snapshot_id: str
    idempotency_key: str
    def __init__(self, scenario: _Optional[str] = ..., snapshot_id: _Optional[str] = ..., idempotency_key: _Optional[str] = ...) -> None: ...

class DetectConflictsResponse(_message.Message):
    __slots__ = ("conflicts",)
    CONFLICTS_FIELD_NUMBER: _ClassVar[int]
    conflicts: _containers.RepeatedCompositeFieldContainer[Conflict]
    def __init__(self, conflicts: _Optional[_Iterable[_Union[Conflict, _Mapping]]] = ...) -> None: ...

class ListConflictsRequest(_message.Message):
    __slots__ = ("scenario", "statuses", "types", "page_size", "page_token")
    SCENARIO_FIELD_NUMBER: _ClassVar[int]
    STATUSES_FIELD_NUMBER: _ClassVar[int]
    TYPES_FIELD_NUMBER: _ClassVar[int]
    PAGE_SIZE_FIELD_NUMBER: _ClassVar[int]
    PAGE_TOKEN_FIELD_NUMBER: _ClassVar[int]
    scenario: str
    statuses: _containers.RepeatedScalarFieldContainer[ResolutionStatus]
    types: _containers.RepeatedScalarFieldContainer[str]
    page_size: int
    page_token: str
    def __init__(self, scenario: _Optional[str] = ..., statuses: _Optional[_Iterable[_Union[ResolutionStatus, str]]] = ..., types: _Optional[_Iterable[str]] = ..., page_size: _Optional[int] = ..., page_token: _Optional[str] = ...) -> None: ...

class ListConflictsResponse(_message.Message):
    __slots__ = ("conflicts", "next_page_token")
    CONFLICTS_FIELD_NUMBER: _ClassVar[int]
    NEXT_PAGE_TOKEN_FIELD_NUMBER: _ClassVar[int]
    conflicts: _containers.RepeatedCompositeFieldContainer[Conflict]
    next_page_token: str
    def __init__(self, conflicts: _Optional[_Iterable[_Union[Conflict, _Mapping]]] = ..., next_page_token: _Optional[str] = ...) -> None: ...

class GetConflictRequest(_message.Message):
    __slots__ = ("id",)
    ID_FIELD_NUMBER: _ClassVar[int]
    id: str
    def __init__(self, id: _Optional[str] = ...) -> None: ...

class GetConflictResponse(_message.Message):
    __slots__ = ("conflict",)
    CONFLICT_FIELD_NUMBER: _ClassVar[int]
    conflict: Conflict
    def __init__(self, conflict: _Optional[_Union[Conflict, _Mapping]] = ...) -> None: ...

class AssignConflictRequest(_message.Message):
    __slots__ = ("id", "domain", "note", "dry_run")
    ID_FIELD_NUMBER: _ClassVar[int]
    DOMAIN_FIELD_NUMBER: _ClassVar[int]
    NOTE_FIELD_NUMBER: _ClassVar[int]
    DRY_RUN_FIELD_NUMBER: _ClassVar[int]
    id: str
    domain: str
    note: str
    dry_run: bool
    def __init__(self, id: _Optional[str] = ..., domain: _Optional[str] = ..., note: _Optional[str] = ..., dry_run: _Optional[bool] = ...) -> None: ...

class AssignConflictResponse(_message.Message):
    __slots__ = ("conflict", "dry_run")
    CONFLICT_FIELD_NUMBER: _ClassVar[int]
    DRY_RUN_FIELD_NUMBER: _ClassVar[int]
    conflict: Conflict
    dry_run: bool
    def __init__(self, conflict: _Optional[_Union[Conflict, _Mapping]] = ..., dry_run: _Optional[bool] = ...) -> None: ...

class ResolveConflictRequest(_message.Message):
    __slots__ = ("id", "note", "force", "dry_run")
    ID_FIELD_NUMBER: _ClassVar[int]
    NOTE_FIELD_NUMBER: _ClassVar[int]
    FORCE_FIELD_NUMBER: _ClassVar[int]
    DRY_RUN_FIELD_NUMBER: _ClassVar[int]
    id: str
    note: str
    force: bool
    dry_run: bool
    def __init__(self, id: _Optional[str] = ..., note: _Optional[str] = ..., force: _Optional[bool] = ..., dry_run: _Optional[bool] = ...) -> None: ...

class ResolveConflictResponse(_message.Message):
    __slots__ = ("conflict", "dry_run", "apply_deferred")
    CONFLICT_FIELD_NUMBER: _ClassVar[int]
    DRY_RUN_FIELD_NUMBER: _ClassVar[int]
    APPLY_DEFERRED_FIELD_NUMBER: _ClassVar[int]
    conflict: Conflict
    dry_run: bool
    apply_deferred: bool
    def __init__(self, conflict: _Optional[_Union[Conflict, _Mapping]] = ..., dry_run: _Optional[bool] = ..., apply_deferred: _Optional[bool] = ...) -> None: ...

class ReopenConflictRequest(_message.Message):
    __slots__ = ("id", "note", "dry_run")
    ID_FIELD_NUMBER: _ClassVar[int]
    NOTE_FIELD_NUMBER: _ClassVar[int]
    DRY_RUN_FIELD_NUMBER: _ClassVar[int]
    id: str
    note: str
    dry_run: bool
    def __init__(self, id: _Optional[str] = ..., note: _Optional[str] = ..., dry_run: _Optional[bool] = ...) -> None: ...

class ReopenConflictResponse(_message.Message):
    __slots__ = ("conflict", "dry_run")
    CONFLICT_FIELD_NUMBER: _ClassVar[int]
    DRY_RUN_FIELD_NUMBER: _ClassVar[int]
    conflict: Conflict
    dry_run: bool
    def __init__(self, conflict: _Optional[_Union[Conflict, _Mapping]] = ..., dry_run: _Optional[bool] = ...) -> None: ...

class ValidateConflictsRequest(_message.Message):
    __slots__ = ("scenario",)
    SCENARIO_FIELD_NUMBER: _ClassVar[int]
    scenario: str
    def __init__(self, scenario: _Optional[str] = ...) -> None: ...

class ValidateConflictsResponse(_message.Message):
    __slots__ = ("conflicts", "clean")
    CONFLICTS_FIELD_NUMBER: _ClassVar[int]
    CLEAN_FIELD_NUMBER: _ClassVar[int]
    conflicts: _containers.RepeatedCompositeFieldContainer[Conflict]
    clean: bool
    def __init__(self, conflicts: _Optional[_Iterable[_Union[Conflict, _Mapping]]] = ..., clean: _Optional[bool] = ...) -> None: ...

class ListDetectorsRequest(_message.Message):
    __slots__ = ()
    def __init__(self) -> None: ...

class ListDetectorsResponse(_message.Message):
    __slots__ = ("detectors",)
    DETECTORS_FIELD_NUMBER: _ClassVar[int]
    detectors: _containers.RepeatedCompositeFieldContainer[DetectorDescriptor]
    def __init__(self, detectors: _Optional[_Iterable[_Union[DetectorDescriptor, _Mapping]]] = ...) -> None: ...

class ListResolversRequest(_message.Message):
    __slots__ = ()
    def __init__(self) -> None: ...

class ListResolversResponse(_message.Message):
    __slots__ = ("resolvers",)
    RESOLVERS_FIELD_NUMBER: _ClassVar[int]
    resolvers: _containers.RepeatedCompositeFieldContainer[ResolverDescriptor]
    def __init__(self, resolvers: _Optional[_Iterable[_Union[ResolverDescriptor, _Mapping]]] = ...) -> None: ...
