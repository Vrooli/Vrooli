import datetime

from common.v1 import attestation_pb2 as _attestation_pb2
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

class FindingClass(int, metaclass=_enum_type_wrapper.EnumTypeWrapper):
    __slots__ = ()
    FINDING_CLASS_UNSPECIFIED: _ClassVar[FindingClass]
    FINDING_CLASS_DETERMINISTIC: _ClassVar[FindingClass]
    FINDING_CLASS_HEURISTIC: _ClassVar[FindingClass]

class FixKind(int, metaclass=_enum_type_wrapper.EnumTypeWrapper):
    __slots__ = ()
    FIX_KIND_UNSPECIFIED: _ClassVar[FixKind]
    FIX_KIND_MOVE_FILE: _ClassVar[FixKind]
    FIX_KIND_REASSIGN_DOMAIN: _ClassVar[FixKind]
    FIX_KIND_BREAK_CYCLE: _ClassVar[FixKind]
    FIX_KIND_ADD_DEPENDENCY: _ClassVar[FixKind]
    FIX_KIND_ADD_TRANSITIONAL: _ClassVar[FixKind]

class Tier(int, metaclass=_enum_type_wrapper.EnumTypeWrapper):
    __slots__ = ()
    TIER_UNSPECIFIED: _ClassVar[Tier]
    TIER_AUTO_PLACE: _ClassVar[Tier]
    TIER_SUGGEST: _ClassVar[Tier]
    TIER_CONFLICT: _ClassVar[Tier]
SEVERITY_UNSPECIFIED: Severity
SEVERITY_INFO: Severity
SEVERITY_WARN: Severity
SEVERITY_ERROR: Severity
SEVERITY_BLOCKER: Severity
FINDING_CLASS_UNSPECIFIED: FindingClass
FINDING_CLASS_DETERMINISTIC: FindingClass
FINDING_CLASS_HEURISTIC: FindingClass
FIX_KIND_UNSPECIFIED: FixKind
FIX_KIND_MOVE_FILE: FixKind
FIX_KIND_REASSIGN_DOMAIN: FixKind
FIX_KIND_BREAK_CYCLE: FixKind
FIX_KIND_ADD_DEPENDENCY: FixKind
FIX_KIND_ADD_TRANSITIONAL: FixKind
TIER_UNSPECIFIED: Tier
TIER_AUTO_PLACE: Tier
TIER_SUGGEST: Tier
TIER_CONFLICT: Tier

class Evidence(_message.Message):
    __slots__ = ("kind", "summary", "locator", "weight")
    KIND_FIELD_NUMBER: _ClassVar[int]
    SUMMARY_FIELD_NUMBER: _ClassVar[int]
    LOCATOR_FIELD_NUMBER: _ClassVar[int]
    WEIGHT_FIELD_NUMBER: _ClassVar[int]
    kind: str
    summary: str
    locator: str
    weight: float
    def __init__(self, kind: _Optional[str] = ..., summary: _Optional[str] = ..., locator: _Optional[str] = ..., weight: _Optional[float] = ...) -> None: ...

class Score(_message.Message):
    __slots__ = ("signal", "domain", "value", "reason", "evidence")
    SIGNAL_FIELD_NUMBER: _ClassVar[int]
    DOMAIN_FIELD_NUMBER: _ClassVar[int]
    VALUE_FIELD_NUMBER: _ClassVar[int]
    REASON_FIELD_NUMBER: _ClassVar[int]
    EVIDENCE_FIELD_NUMBER: _ClassVar[int]
    signal: str
    domain: str
    value: float
    reason: str
    evidence: _containers.RepeatedCompositeFieldContainer[Evidence]
    def __init__(self, signal: _Optional[str] = ..., domain: _Optional[str] = ..., value: _Optional[float] = ..., reason: _Optional[str] = ..., evidence: _Optional[_Iterable[_Union[Evidence, _Mapping]]] = ...) -> None: ...

class Abstention(_message.Message):
    __slots__ = ("signal", "reason", "evidence")
    SIGNAL_FIELD_NUMBER: _ClassVar[int]
    REASON_FIELD_NUMBER: _ClassVar[int]
    EVIDENCE_FIELD_NUMBER: _ClassVar[int]
    signal: str
    reason: str
    evidence: _containers.RepeatedCompositeFieldContainer[Evidence]
    def __init__(self, signal: _Optional[str] = ..., reason: _Optional[str] = ..., evidence: _Optional[_Iterable[_Union[Evidence, _Mapping]]] = ...) -> None: ...

class DomainValue(_message.Message):
    __slots__ = ("domain", "value")
    DOMAIN_FIELD_NUMBER: _ClassVar[int]
    VALUE_FIELD_NUMBER: _ClassVar[int]
    domain: str
    value: float
    def __init__(self, domain: _Optional[str] = ..., value: _Optional[float] = ...) -> None: ...

class Verdict(_message.Message):
    __slots__ = ("chunk_id", "chunk_path", "tier", "top_domain", "top_value", "runner_up_domain", "runner_up_value", "scores", "domain_values", "tied", "abstentions")
    CHUNK_ID_FIELD_NUMBER: _ClassVar[int]
    CHUNK_PATH_FIELD_NUMBER: _ClassVar[int]
    TIER_FIELD_NUMBER: _ClassVar[int]
    TOP_DOMAIN_FIELD_NUMBER: _ClassVar[int]
    TOP_VALUE_FIELD_NUMBER: _ClassVar[int]
    RUNNER_UP_DOMAIN_FIELD_NUMBER: _ClassVar[int]
    RUNNER_UP_VALUE_FIELD_NUMBER: _ClassVar[int]
    SCORES_FIELD_NUMBER: _ClassVar[int]
    DOMAIN_VALUES_FIELD_NUMBER: _ClassVar[int]
    TIED_FIELD_NUMBER: _ClassVar[int]
    ABSTENTIONS_FIELD_NUMBER: _ClassVar[int]
    chunk_id: str
    chunk_path: str
    tier: Tier
    top_domain: str
    top_value: float
    runner_up_domain: str
    runner_up_value: float
    scores: _containers.RepeatedCompositeFieldContainer[Score]
    domain_values: _containers.RepeatedCompositeFieldContainer[DomainValue]
    tied: bool
    abstentions: _containers.RepeatedCompositeFieldContainer[Abstention]
    def __init__(self, chunk_id: _Optional[str] = ..., chunk_path: _Optional[str] = ..., tier: _Optional[_Union[Tier, str]] = ..., top_domain: _Optional[str] = ..., top_value: _Optional[float] = ..., runner_up_domain: _Optional[str] = ..., runner_up_value: _Optional[float] = ..., scores: _Optional[_Iterable[_Union[Score, _Mapping]]] = ..., domain_values: _Optional[_Iterable[_Union[DomainValue, _Mapping]]] = ..., tied: _Optional[bool] = ..., abstentions: _Optional[_Iterable[_Union[Abstention, _Mapping]]] = ...) -> None: ...

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
    __slots__ = ("id", "scenario", "detector", "type", "subtype", "severity", "locations", "domains", "evidence", "suggested_fixes", "snapshot_id", "verdict", "detected_at", "updated_at", "suppressed", "suppression_reason", "stable_id", "instance_id", "finding_class", "attestation")
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
    SNAPSHOT_ID_FIELD_NUMBER: _ClassVar[int]
    VERDICT_FIELD_NUMBER: _ClassVar[int]
    DETECTED_AT_FIELD_NUMBER: _ClassVar[int]
    UPDATED_AT_FIELD_NUMBER: _ClassVar[int]
    SUPPRESSED_FIELD_NUMBER: _ClassVar[int]
    SUPPRESSION_REASON_FIELD_NUMBER: _ClassVar[int]
    STABLE_ID_FIELD_NUMBER: _ClassVar[int]
    INSTANCE_ID_FIELD_NUMBER: _ClassVar[int]
    FINDING_CLASS_FIELD_NUMBER: _ClassVar[int]
    ATTESTATION_FIELD_NUMBER: _ClassVar[int]
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
    snapshot_id: str
    verdict: Verdict
    detected_at: _timestamp_pb2.Timestamp
    updated_at: _timestamp_pb2.Timestamp
    suppressed: bool
    suppression_reason: str
    stable_id: str
    instance_id: str
    finding_class: FindingClass
    attestation: _attestation_pb2.AttestedAnswer
    def __init__(self, id: _Optional[str] = ..., scenario: _Optional[str] = ..., detector: _Optional[str] = ..., type: _Optional[str] = ..., subtype: _Optional[str] = ..., severity: _Optional[_Union[Severity, str]] = ..., locations: _Optional[_Iterable[str]] = ..., domains: _Optional[_Iterable[str]] = ..., evidence: _Optional[_Iterable[_Union[ConflictEvidence, _Mapping]]] = ..., suggested_fixes: _Optional[_Iterable[_Union[Fix, _Mapping]]] = ..., snapshot_id: _Optional[str] = ..., verdict: _Optional[_Union[Verdict, _Mapping]] = ..., detected_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., updated_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., suppressed: _Optional[bool] = ..., suppression_reason: _Optional[str] = ..., stable_id: _Optional[str] = ..., instance_id: _Optional[str] = ..., finding_class: _Optional[_Union[FindingClass, str]] = ..., attestation: _Optional[_Union[_attestation_pb2.AttestedAnswer, _Mapping]] = ...) -> None: ...
