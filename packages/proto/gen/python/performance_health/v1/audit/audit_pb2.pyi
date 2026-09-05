from common.v1 import metrics_pb2 as _metrics_pb2
from performance_health.v1.readiness import readiness_pb2 as _readiness_pb2
from google.protobuf.internal import enum_type_wrapper as _enum_type_wrapper
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Mapping as _Mapping
from typing import ClassVar as _ClassVar, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class AuditOutcome(int, metaclass=_enum_type_wrapper.EnumTypeWrapper):
    __slots__ = ()
    AUDIT_OUTCOME_UNSPECIFIED: _ClassVar[AuditOutcome]
    AUDIT_OUTCOME_CAPTURED: _ClassVar[AuditOutcome]
    AUDIT_OUTCOME_SKIPPED: _ClassVar[AuditOutcome]
    AUDIT_OUTCOME_FAILED: _ClassVar[AuditOutcome]
    AUDIT_OUTCOME_UNAVAILABLE: _ClassVar[AuditOutcome]
AUDIT_OUTCOME_UNSPECIFIED: AuditOutcome
AUDIT_OUTCOME_CAPTURED: AuditOutcome
AUDIT_OUTCOME_SKIPPED: AuditOutcome
AUDIT_OUTCOME_FAILED: AuditOutcome
AUDIT_OUTCOME_UNAVAILABLE: AuditOutcome

class RunAuditRequest(_message.Message):
    __slots__ = ("scenario", "path", "workflow")
    SCENARIO_FIELD_NUMBER: _ClassVar[int]
    PATH_FIELD_NUMBER: _ClassVar[int]
    WORKFLOW_FIELD_NUMBER: _ClassVar[int]
    scenario: str
    path: str
    workflow: str
    def __init__(self, scenario: _Optional[str] = ..., path: _Optional[str] = ..., workflow: _Optional[str] = ...) -> None: ...

class RunAuditResponse(_message.Message):
    __slots__ = ("scenario", "outcome", "tier", "trace_artifact", "web_vitals_artifact", "reason", "metrics")
    SCENARIO_FIELD_NUMBER: _ClassVar[int]
    OUTCOME_FIELD_NUMBER: _ClassVar[int]
    TIER_FIELD_NUMBER: _ClassVar[int]
    TRACE_ARTIFACT_FIELD_NUMBER: _ClassVar[int]
    WEB_VITALS_ARTIFACT_FIELD_NUMBER: _ClassVar[int]
    REASON_FIELD_NUMBER: _ClassVar[int]
    METRICS_FIELD_NUMBER: _ClassVar[int]
    scenario: str
    outcome: AuditOutcome
    tier: _readiness_pb2.CaptureTier
    trace_artifact: str
    web_vitals_artifact: str
    reason: str
    metrics: _metrics_pb2.ExecutionMetrics
    def __init__(self, scenario: _Optional[str] = ..., outcome: _Optional[_Union[AuditOutcome, str]] = ..., tier: _Optional[_Union[_readiness_pb2.CaptureTier, str]] = ..., trace_artifact: _Optional[str] = ..., web_vitals_artifact: _Optional[str] = ..., reason: _Optional[str] = ..., metrics: _Optional[_Union[_metrics_pb2.ExecutionMetrics, _Mapping]] = ...) -> None: ...
