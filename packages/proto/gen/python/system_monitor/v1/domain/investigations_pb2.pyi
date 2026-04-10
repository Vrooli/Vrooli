import datetime

from google.protobuf import struct_pb2 as _struct_pb2
from google.protobuf import timestamp_pb2 as _timestamp_pb2
from system_monitor.v1.domain import types_pb2 as _types_pb2
from google.protobuf.internal import containers as _containers
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Iterable as _Iterable, Mapping as _Mapping
from typing import ClassVar as _ClassVar, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class Investigation(_message.Message):
    __slots__ = ("id", "status", "anomaly_id", "start_time", "end_time", "findings", "progress", "details", "steps", "trigger_reason", "confidence_score", "agent_id")
    ID_FIELD_NUMBER: _ClassVar[int]
    STATUS_FIELD_NUMBER: _ClassVar[int]
    ANOMALY_ID_FIELD_NUMBER: _ClassVar[int]
    START_TIME_FIELD_NUMBER: _ClassVar[int]
    END_TIME_FIELD_NUMBER: _ClassVar[int]
    FINDINGS_FIELD_NUMBER: _ClassVar[int]
    PROGRESS_FIELD_NUMBER: _ClassVar[int]
    DETAILS_FIELD_NUMBER: _ClassVar[int]
    STEPS_FIELD_NUMBER: _ClassVar[int]
    TRIGGER_REASON_FIELD_NUMBER: _ClassVar[int]
    CONFIDENCE_SCORE_FIELD_NUMBER: _ClassVar[int]
    AGENT_ID_FIELD_NUMBER: _ClassVar[int]
    id: str
    status: _types_pb2.InvestigationStatus
    anomaly_id: str
    start_time: _timestamp_pb2.Timestamp
    end_time: _timestamp_pb2.Timestamp
    findings: str
    progress: int
    details: _struct_pb2.Struct
    steps: _containers.RepeatedCompositeFieldContainer[InvestigationStep]
    trigger_reason: str
    confidence_score: float
    agent_id: str
    def __init__(self, id: _Optional[str] = ..., status: _Optional[_Union[_types_pb2.InvestigationStatus, str]] = ..., anomaly_id: _Optional[str] = ..., start_time: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., end_time: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., findings: _Optional[str] = ..., progress: _Optional[int] = ..., details: _Optional[_Union[_struct_pb2.Struct, _Mapping]] = ..., steps: _Optional[_Iterable[_Union[InvestigationStep, _Mapping]]] = ..., trigger_reason: _Optional[str] = ..., confidence_score: _Optional[float] = ..., agent_id: _Optional[str] = ...) -> None: ...

class InvestigationStep(_message.Message):
    __slots__ = ("name", "status", "start_time", "end_time", "findings")
    NAME_FIELD_NUMBER: _ClassVar[int]
    STATUS_FIELD_NUMBER: _ClassVar[int]
    START_TIME_FIELD_NUMBER: _ClassVar[int]
    END_TIME_FIELD_NUMBER: _ClassVar[int]
    FINDINGS_FIELD_NUMBER: _ClassVar[int]
    name: str
    status: _types_pb2.InvestigationStepStatus
    start_time: _timestamp_pb2.Timestamp
    end_time: _timestamp_pb2.Timestamp
    findings: str
    def __init__(self, name: _Optional[str] = ..., status: _Optional[_Union[_types_pb2.InvestigationStepStatus, str]] = ..., start_time: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., end_time: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., findings: _Optional[str] = ...) -> None: ...

class InvestigationFinding(_message.Message):
    __slots__ = ("type", "description", "confidence", "evidence", "timestamp", "impact")
    TYPE_FIELD_NUMBER: _ClassVar[int]
    DESCRIPTION_FIELD_NUMBER: _ClassVar[int]
    CONFIDENCE_FIELD_NUMBER: _ClassVar[int]
    EVIDENCE_FIELD_NUMBER: _ClassVar[int]
    TIMESTAMP_FIELD_NUMBER: _ClassVar[int]
    IMPACT_FIELD_NUMBER: _ClassVar[int]
    type: _types_pb2.FindingType
    description: str
    confidence: float
    evidence: _containers.RepeatedScalarFieldContainer[str]
    timestamp: _timestamp_pb2.Timestamp
    impact: _types_pb2.Severity
    def __init__(self, type: _Optional[_Union[_types_pb2.FindingType, str]] = ..., description: _Optional[str] = ..., confidence: _Optional[float] = ..., evidence: _Optional[_Iterable[str]] = ..., timestamp: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., impact: _Optional[_Union[_types_pb2.Severity, str]] = ...) -> None: ...

class RootCause(_message.Message):
    __slots__ = ("category", "description", "confidence", "evidence", "timeline")
    CATEGORY_FIELD_NUMBER: _ClassVar[int]
    DESCRIPTION_FIELD_NUMBER: _ClassVar[int]
    CONFIDENCE_FIELD_NUMBER: _ClassVar[int]
    EVIDENCE_FIELD_NUMBER: _ClassVar[int]
    TIMELINE_FIELD_NUMBER: _ClassVar[int]
    category: str
    description: str
    confidence: float
    evidence: _containers.RepeatedScalarFieldContainer[str]
    timeline: _containers.RepeatedCompositeFieldContainer[TimelineEvent]
    def __init__(self, category: _Optional[str] = ..., description: _Optional[str] = ..., confidence: _Optional[float] = ..., evidence: _Optional[_Iterable[str]] = ..., timeline: _Optional[_Iterable[_Union[TimelineEvent, _Mapping]]] = ...) -> None: ...

class TimelineEvent(_message.Message):
    __slots__ = ("timestamp", "event", "source", "relevance")
    TIMESTAMP_FIELD_NUMBER: _ClassVar[int]
    EVENT_FIELD_NUMBER: _ClassVar[int]
    SOURCE_FIELD_NUMBER: _ClassVar[int]
    RELEVANCE_FIELD_NUMBER: _ClassVar[int]
    timestamp: _timestamp_pb2.Timestamp
    event: str
    source: str
    relevance: _types_pb2.Relevance
    def __init__(self, timestamp: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., event: _Optional[str] = ..., source: _Optional[str] = ..., relevance: _Optional[_Union[_types_pb2.Relevance, str]] = ...) -> None: ...

class ImpactAssessment(_message.Message):
    __slots__ = ("severity", "affected_users", "performance_loss", "estimated_cost", "business_impact")
    SEVERITY_FIELD_NUMBER: _ClassVar[int]
    AFFECTED_USERS_FIELD_NUMBER: _ClassVar[int]
    PERFORMANCE_LOSS_FIELD_NUMBER: _ClassVar[int]
    ESTIMATED_COST_FIELD_NUMBER: _ClassVar[int]
    BUSINESS_IMPACT_FIELD_NUMBER: _ClassVar[int]
    severity: _types_pb2.Severity
    affected_users: int
    performance_loss: float
    estimated_cost: float
    business_impact: _containers.RepeatedScalarFieldContainer[str]
    def __init__(self, severity: _Optional[_Union[_types_pb2.Severity, str]] = ..., affected_users: _Optional[int] = ..., performance_loss: _Optional[float] = ..., estimated_cost: _Optional[float] = ..., business_impact: _Optional[_Iterable[str]] = ...) -> None: ...

class ResolutionPlan(_message.Message):
    __slots__ = ("steps", "estimated_time", "required_skills", "risk_level", "auto_executable")
    STEPS_FIELD_NUMBER: _ClassVar[int]
    ESTIMATED_TIME_FIELD_NUMBER: _ClassVar[int]
    REQUIRED_SKILLS_FIELD_NUMBER: _ClassVar[int]
    RISK_LEVEL_FIELD_NUMBER: _ClassVar[int]
    AUTO_EXECUTABLE_FIELD_NUMBER: _ClassVar[int]
    steps: _containers.RepeatedCompositeFieldContainer[ResolutionStep]
    estimated_time: str
    required_skills: _containers.RepeatedScalarFieldContainer[str]
    risk_level: _types_pb2.RiskLevel
    auto_executable: bool
    def __init__(self, steps: _Optional[_Iterable[_Union[ResolutionStep, _Mapping]]] = ..., estimated_time: _Optional[str] = ..., required_skills: _Optional[_Iterable[str]] = ..., risk_level: _Optional[_Union[_types_pb2.RiskLevel, str]] = ..., auto_executable: _Optional[bool] = ...) -> None: ...

class ResolutionStep(_message.Message):
    __slots__ = ("order", "description", "command", "expected", "risk")
    ORDER_FIELD_NUMBER: _ClassVar[int]
    DESCRIPTION_FIELD_NUMBER: _ClassVar[int]
    COMMAND_FIELD_NUMBER: _ClassVar[int]
    EXPECTED_FIELD_NUMBER: _ClassVar[int]
    RISK_FIELD_NUMBER: _ClassVar[int]
    order: int
    description: str
    command: str
    expected: str
    risk: str
    def __init__(self, order: _Optional[int] = ..., description: _Optional[str] = ..., command: _Optional[str] = ..., expected: _Optional[str] = ..., risk: _Optional[str] = ...) -> None: ...

class TriggerConfig(_message.Message):
    __slots__ = ("id", "name", "description", "icon", "enabled", "auto_fix", "threshold", "unit", "condition")
    ID_FIELD_NUMBER: _ClassVar[int]
    NAME_FIELD_NUMBER: _ClassVar[int]
    DESCRIPTION_FIELD_NUMBER: _ClassVar[int]
    ICON_FIELD_NUMBER: _ClassVar[int]
    ENABLED_FIELD_NUMBER: _ClassVar[int]
    AUTO_FIX_FIELD_NUMBER: _ClassVar[int]
    THRESHOLD_FIELD_NUMBER: _ClassVar[int]
    UNIT_FIELD_NUMBER: _ClassVar[int]
    CONDITION_FIELD_NUMBER: _ClassVar[int]
    id: str
    name: str
    description: str
    icon: str
    enabled: bool
    auto_fix: bool
    threshold: float
    unit: str
    condition: _types_pb2.TriggerCondition
    def __init__(self, id: _Optional[str] = ..., name: _Optional[str] = ..., description: _Optional[str] = ..., icon: _Optional[str] = ..., enabled: _Optional[bool] = ..., auto_fix: _Optional[bool] = ..., threshold: _Optional[float] = ..., unit: _Optional[str] = ..., condition: _Optional[_Union[_types_pb2.TriggerCondition, str]] = ...) -> None: ...

class CooldownStatus(_message.Message):
    __slots__ = ("cooldown_period_seconds", "remaining_seconds", "last_trigger_time", "is_ready")
    COOLDOWN_PERIOD_SECONDS_FIELD_NUMBER: _ClassVar[int]
    REMAINING_SECONDS_FIELD_NUMBER: _ClassVar[int]
    LAST_TRIGGER_TIME_FIELD_NUMBER: _ClassVar[int]
    IS_READY_FIELD_NUMBER: _ClassVar[int]
    cooldown_period_seconds: int
    remaining_seconds: int
    last_trigger_time: _timestamp_pb2.Timestamp
    is_ready: bool
    def __init__(self, cooldown_period_seconds: _Optional[int] = ..., remaining_seconds: _Optional[int] = ..., last_trigger_time: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., is_ready: _Optional[bool] = ...) -> None: ...

class Anomaly(_message.Message):
    __slots__ = ("id", "type", "severity", "description", "metric_data", "detected_at", "resolved_at", "status")
    ID_FIELD_NUMBER: _ClassVar[int]
    TYPE_FIELD_NUMBER: _ClassVar[int]
    SEVERITY_FIELD_NUMBER: _ClassVar[int]
    DESCRIPTION_FIELD_NUMBER: _ClassVar[int]
    METRIC_DATA_FIELD_NUMBER: _ClassVar[int]
    DETECTED_AT_FIELD_NUMBER: _ClassVar[int]
    RESOLVED_AT_FIELD_NUMBER: _ClassVar[int]
    STATUS_FIELD_NUMBER: _ClassVar[int]
    id: str
    type: str
    severity: _types_pb2.Severity
    description: str
    metric_data: _struct_pb2.Struct
    detected_at: _timestamp_pb2.Timestamp
    resolved_at: _timestamp_pb2.Timestamp
    status: _types_pb2.InvestigationStatus
    def __init__(self, id: _Optional[str] = ..., type: _Optional[str] = ..., severity: _Optional[_Union[_types_pb2.Severity, str]] = ..., description: _Optional[str] = ..., metric_data: _Optional[_Union[_struct_pb2.Struct, _Mapping]] = ..., detected_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., resolved_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., status: _Optional[_Union[_types_pb2.InvestigationStatus, str]] = ...) -> None: ...
