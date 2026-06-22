import datetime

from google.protobuf import struct_pb2 as _struct_pb2
from google.protobuf import timestamp_pb2 as _timestamp_pb2
from google.api import annotations_pb2 as _annotations_pb2
from google.protobuf.internal import containers as _containers
from google.protobuf.internal import enum_type_wrapper as _enum_type_wrapper
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Iterable as _Iterable, Mapping as _Mapping
from typing import ClassVar as _ClassVar, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class InvestigationStatus(int, metaclass=_enum_type_wrapper.EnumTypeWrapper):
    __slots__ = ()
    INVESTIGATION_STATUS_UNSPECIFIED: _ClassVar[InvestigationStatus]
    INVESTIGATION_STATUS_QUEUED: _ClassVar[InvestigationStatus]
    INVESTIGATION_STATUS_IN_PROGRESS: _ClassVar[InvestigationStatus]
    INVESTIGATION_STATUS_COMPLETED: _ClassVar[InvestigationStatus]
    INVESTIGATION_STATUS_FAILED: _ClassVar[InvestigationStatus]
    INVESTIGATION_STATUS_STOPPED: _ClassVar[InvestigationStatus]
    INVESTIGATION_STATUS_CANCELLED: _ClassVar[InvestigationStatus]

class InvestigationStepStatus(int, metaclass=_enum_type_wrapper.EnumTypeWrapper):
    __slots__ = ()
    INVESTIGATION_STEP_STATUS_UNSPECIFIED: _ClassVar[InvestigationStepStatus]
    INVESTIGATION_STEP_STATUS_PENDING: _ClassVar[InvestigationStepStatus]
    INVESTIGATION_STEP_STATUS_IN_PROGRESS: _ClassVar[InvestigationStepStatus]
    INVESTIGATION_STEP_STATUS_COMPLETED: _ClassVar[InvestigationStepStatus]
    INVESTIGATION_STEP_STATUS_FAILED: _ClassVar[InvestigationStepStatus]
    INVESTIGATION_STEP_STATUS_SKIPPED: _ClassVar[InvestigationStepStatus]

class FindingType(int, metaclass=_enum_type_wrapper.EnumTypeWrapper):
    __slots__ = ()
    FINDING_TYPE_UNSPECIFIED: _ClassVar[FindingType]
    FINDING_TYPE_METRIC: _ClassVar[FindingType]
    FINDING_TYPE_LOG: _ClassVar[FindingType]
    FINDING_TYPE_PROCESS: _ClassVar[FindingType]
    FINDING_TYPE_NETWORK: _ClassVar[FindingType]

class Severity(int, metaclass=_enum_type_wrapper.EnumTypeWrapper):
    __slots__ = ()
    SEVERITY_UNSPECIFIED: _ClassVar[Severity]
    SEVERITY_LOW: _ClassVar[Severity]
    SEVERITY_MEDIUM: _ClassVar[Severity]
    SEVERITY_HIGH: _ClassVar[Severity]
    SEVERITY_CRITICAL: _ClassVar[Severity]

class Relevance(int, metaclass=_enum_type_wrapper.EnumTypeWrapper):
    __slots__ = ()
    RELEVANCE_UNSPECIFIED: _ClassVar[Relevance]
    RELEVANCE_LOW: _ClassVar[Relevance]
    RELEVANCE_MEDIUM: _ClassVar[Relevance]
    RELEVANCE_HIGH: _ClassVar[Relevance]

class TriggerCondition(int, metaclass=_enum_type_wrapper.EnumTypeWrapper):
    __slots__ = ()
    TRIGGER_CONDITION_UNSPECIFIED: _ClassVar[TriggerCondition]
    TRIGGER_CONDITION_ABOVE: _ClassVar[TriggerCondition]
    TRIGGER_CONDITION_BELOW: _ClassVar[TriggerCondition]

class RiskLevel(int, metaclass=_enum_type_wrapper.EnumTypeWrapper):
    __slots__ = ()
    RISK_LEVEL_UNSPECIFIED: _ClassVar[RiskLevel]
    RISK_LEVEL_LOW: _ClassVar[RiskLevel]
    RISK_LEVEL_MEDIUM: _ClassVar[RiskLevel]
    RISK_LEVEL_HIGH: _ClassVar[RiskLevel]
    RISK_LEVEL_CRITICAL: _ClassVar[RiskLevel]
INVESTIGATION_STATUS_UNSPECIFIED: InvestigationStatus
INVESTIGATION_STATUS_QUEUED: InvestigationStatus
INVESTIGATION_STATUS_IN_PROGRESS: InvestigationStatus
INVESTIGATION_STATUS_COMPLETED: InvestigationStatus
INVESTIGATION_STATUS_FAILED: InvestigationStatus
INVESTIGATION_STATUS_STOPPED: InvestigationStatus
INVESTIGATION_STATUS_CANCELLED: InvestigationStatus
INVESTIGATION_STEP_STATUS_UNSPECIFIED: InvestigationStepStatus
INVESTIGATION_STEP_STATUS_PENDING: InvestigationStepStatus
INVESTIGATION_STEP_STATUS_IN_PROGRESS: InvestigationStepStatus
INVESTIGATION_STEP_STATUS_COMPLETED: InvestigationStepStatus
INVESTIGATION_STEP_STATUS_FAILED: InvestigationStepStatus
INVESTIGATION_STEP_STATUS_SKIPPED: InvestigationStepStatus
FINDING_TYPE_UNSPECIFIED: FindingType
FINDING_TYPE_METRIC: FindingType
FINDING_TYPE_LOG: FindingType
FINDING_TYPE_PROCESS: FindingType
FINDING_TYPE_NETWORK: FindingType
SEVERITY_UNSPECIFIED: Severity
SEVERITY_LOW: Severity
SEVERITY_MEDIUM: Severity
SEVERITY_HIGH: Severity
SEVERITY_CRITICAL: Severity
RELEVANCE_UNSPECIFIED: Relevance
RELEVANCE_LOW: Relevance
RELEVANCE_MEDIUM: Relevance
RELEVANCE_HIGH: Relevance
TRIGGER_CONDITION_UNSPECIFIED: TriggerCondition
TRIGGER_CONDITION_ABOVE: TriggerCondition
TRIGGER_CONDITION_BELOW: TriggerCondition
RISK_LEVEL_UNSPECIFIED: RiskLevel
RISK_LEVEL_LOW: RiskLevel
RISK_LEVEL_MEDIUM: RiskLevel
RISK_LEVEL_HIGH: RiskLevel
RISK_LEVEL_CRITICAL: RiskLevel

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
    status: InvestigationStatus
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
    def __init__(self, id: _Optional[str] = ..., status: _Optional[_Union[InvestigationStatus, str]] = ..., anomaly_id: _Optional[str] = ..., start_time: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., end_time: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., findings: _Optional[str] = ..., progress: _Optional[int] = ..., details: _Optional[_Union[_struct_pb2.Struct, _Mapping]] = ..., steps: _Optional[_Iterable[_Union[InvestigationStep, _Mapping]]] = ..., trigger_reason: _Optional[str] = ..., confidence_score: _Optional[float] = ..., agent_id: _Optional[str] = ...) -> None: ...

class InvestigationStep(_message.Message):
    __slots__ = ("name", "status", "start_time", "end_time", "findings")
    NAME_FIELD_NUMBER: _ClassVar[int]
    STATUS_FIELD_NUMBER: _ClassVar[int]
    START_TIME_FIELD_NUMBER: _ClassVar[int]
    END_TIME_FIELD_NUMBER: _ClassVar[int]
    FINDINGS_FIELD_NUMBER: _ClassVar[int]
    name: str
    status: InvestigationStepStatus
    start_time: _timestamp_pb2.Timestamp
    end_time: _timestamp_pb2.Timestamp
    findings: str
    def __init__(self, name: _Optional[str] = ..., status: _Optional[_Union[InvestigationStepStatus, str]] = ..., start_time: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., end_time: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., findings: _Optional[str] = ...) -> None: ...

class InvestigationFinding(_message.Message):
    __slots__ = ("type", "description", "confidence", "evidence", "timestamp", "impact")
    TYPE_FIELD_NUMBER: _ClassVar[int]
    DESCRIPTION_FIELD_NUMBER: _ClassVar[int]
    CONFIDENCE_FIELD_NUMBER: _ClassVar[int]
    EVIDENCE_FIELD_NUMBER: _ClassVar[int]
    TIMESTAMP_FIELD_NUMBER: _ClassVar[int]
    IMPACT_FIELD_NUMBER: _ClassVar[int]
    type: FindingType
    description: str
    confidence: float
    evidence: _containers.RepeatedScalarFieldContainer[str]
    timestamp: _timestamp_pb2.Timestamp
    impact: Severity
    def __init__(self, type: _Optional[_Union[FindingType, str]] = ..., description: _Optional[str] = ..., confidence: _Optional[float] = ..., evidence: _Optional[_Iterable[str]] = ..., timestamp: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., impact: _Optional[_Union[Severity, str]] = ...) -> None: ...

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
    relevance: Relevance
    def __init__(self, timestamp: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., event: _Optional[str] = ..., source: _Optional[str] = ..., relevance: _Optional[_Union[Relevance, str]] = ...) -> None: ...

class ImpactAssessment(_message.Message):
    __slots__ = ("severity", "affected_users", "performance_loss", "estimated_cost", "business_impact")
    SEVERITY_FIELD_NUMBER: _ClassVar[int]
    AFFECTED_USERS_FIELD_NUMBER: _ClassVar[int]
    PERFORMANCE_LOSS_FIELD_NUMBER: _ClassVar[int]
    ESTIMATED_COST_FIELD_NUMBER: _ClassVar[int]
    BUSINESS_IMPACT_FIELD_NUMBER: _ClassVar[int]
    severity: Severity
    affected_users: int
    performance_loss: float
    estimated_cost: float
    business_impact: _containers.RepeatedScalarFieldContainer[str]
    def __init__(self, severity: _Optional[_Union[Severity, str]] = ..., affected_users: _Optional[int] = ..., performance_loss: _Optional[float] = ..., estimated_cost: _Optional[float] = ..., business_impact: _Optional[_Iterable[str]] = ...) -> None: ...

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
    risk_level: RiskLevel
    auto_executable: bool
    def __init__(self, steps: _Optional[_Iterable[_Union[ResolutionStep, _Mapping]]] = ..., estimated_time: _Optional[str] = ..., required_skills: _Optional[_Iterable[str]] = ..., risk_level: _Optional[_Union[RiskLevel, str]] = ..., auto_executable: _Optional[bool] = ...) -> None: ...

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
    condition: TriggerCondition
    def __init__(self, id: _Optional[str] = ..., name: _Optional[str] = ..., description: _Optional[str] = ..., icon: _Optional[str] = ..., enabled: _Optional[bool] = ..., auto_fix: _Optional[bool] = ..., threshold: _Optional[float] = ..., unit: _Optional[str] = ..., condition: _Optional[_Union[TriggerCondition, str]] = ...) -> None: ...

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
    severity: Severity
    description: str
    metric_data: _struct_pb2.Struct
    detected_at: _timestamp_pb2.Timestamp
    resolved_at: _timestamp_pb2.Timestamp
    status: InvestigationStatus
    def __init__(self, id: _Optional[str] = ..., type: _Optional[str] = ..., severity: _Optional[_Union[Severity, str]] = ..., description: _Optional[str] = ..., metric_data: _Optional[_Union[_struct_pb2.Struct, _Mapping]] = ..., detected_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., resolved_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., status: _Optional[_Union[InvestigationStatus, str]] = ...) -> None: ...

class TriggerInvestigationRequest(_message.Message):
    __slots__ = ("auto_fix", "note")
    AUTO_FIX_FIELD_NUMBER: _ClassVar[int]
    NOTE_FIELD_NUMBER: _ClassVar[int]
    auto_fix: bool
    note: str
    def __init__(self, auto_fix: _Optional[bool] = ..., note: _Optional[str] = ...) -> None: ...

class TriggerInvestigationResponse(_message.Message):
    __slots__ = ("status", "investigation_id", "api_base_url", "message", "auto_fix", "note")
    STATUS_FIELD_NUMBER: _ClassVar[int]
    INVESTIGATION_ID_FIELD_NUMBER: _ClassVar[int]
    API_BASE_URL_FIELD_NUMBER: _ClassVar[int]
    MESSAGE_FIELD_NUMBER: _ClassVar[int]
    AUTO_FIX_FIELD_NUMBER: _ClassVar[int]
    NOTE_FIELD_NUMBER: _ClassVar[int]
    status: str
    investigation_id: str
    api_base_url: str
    message: str
    auto_fix: bool
    note: str
    def __init__(self, status: _Optional[str] = ..., investigation_id: _Optional[str] = ..., api_base_url: _Optional[str] = ..., message: _Optional[str] = ..., auto_fix: _Optional[bool] = ..., note: _Optional[str] = ...) -> None: ...

class GetInvestigationRequest(_message.Message):
    __slots__ = ("id",)
    ID_FIELD_NUMBER: _ClassVar[int]
    id: str
    def __init__(self, id: _Optional[str] = ...) -> None: ...

class GetInvestigationResponse(_message.Message):
    __slots__ = ("investigation",)
    INVESTIGATION_FIELD_NUMBER: _ClassVar[int]
    investigation: Investigation
    def __init__(self, investigation: _Optional[_Union[Investigation, _Mapping]] = ...) -> None: ...

class GetLatestInvestigationRequest(_message.Message):
    __slots__ = ()
    def __init__(self) -> None: ...

class GetLatestInvestigationResponse(_message.Message):
    __slots__ = ("investigation",)
    INVESTIGATION_FIELD_NUMBER: _ClassVar[int]
    investigation: Investigation
    def __init__(self, investigation: _Optional[_Union[Investigation, _Mapping]] = ...) -> None: ...

class ListInvestigationsRequest(_message.Message):
    __slots__ = ("limit",)
    LIMIT_FIELD_NUMBER: _ClassVar[int]
    limit: int
    def __init__(self, limit: _Optional[int] = ...) -> None: ...

class ListInvestigationsResponse(_message.Message):
    __slots__ = ("investigations",)
    INVESTIGATIONS_FIELD_NUMBER: _ClassVar[int]
    investigations: _containers.RepeatedCompositeFieldContainer[Investigation]
    def __init__(self, investigations: _Optional[_Iterable[_Union[Investigation, _Mapping]]] = ...) -> None: ...

class UpdateInvestigationStatusRequest(_message.Message):
    __slots__ = ("id", "status")
    ID_FIELD_NUMBER: _ClassVar[int]
    STATUS_FIELD_NUMBER: _ClassVar[int]
    id: str
    status: InvestigationStatus
    def __init__(self, id: _Optional[str] = ..., status: _Optional[_Union[InvestigationStatus, str]] = ...) -> None: ...

class UpdateInvestigationStatusResponse(_message.Message):
    __slots__ = ("status",)
    STATUS_FIELD_NUMBER: _ClassVar[int]
    status: str
    def __init__(self, status: _Optional[str] = ...) -> None: ...

class UpdateInvestigationFindingsRequest(_message.Message):
    __slots__ = ("id", "findings", "details")
    ID_FIELD_NUMBER: _ClassVar[int]
    FINDINGS_FIELD_NUMBER: _ClassVar[int]
    DETAILS_FIELD_NUMBER: _ClassVar[int]
    id: str
    findings: str
    details: _struct_pb2.Struct
    def __init__(self, id: _Optional[str] = ..., findings: _Optional[str] = ..., details: _Optional[_Union[_struct_pb2.Struct, _Mapping]] = ...) -> None: ...

class UpdateInvestigationFindingsResponse(_message.Message):
    __slots__ = ("status",)
    STATUS_FIELD_NUMBER: _ClassVar[int]
    status: str
    def __init__(self, status: _Optional[str] = ...) -> None: ...

class UpdateInvestigationProgressRequest(_message.Message):
    __slots__ = ("id", "progress")
    ID_FIELD_NUMBER: _ClassVar[int]
    PROGRESS_FIELD_NUMBER: _ClassVar[int]
    id: str
    progress: int
    def __init__(self, id: _Optional[str] = ..., progress: _Optional[int] = ...) -> None: ...

class UpdateInvestigationProgressResponse(_message.Message):
    __slots__ = ("status",)
    STATUS_FIELD_NUMBER: _ClassVar[int]
    status: str
    def __init__(self, status: _Optional[str] = ...) -> None: ...

class AddInvestigationStepRequest(_message.Message):
    __slots__ = ("id", "step")
    ID_FIELD_NUMBER: _ClassVar[int]
    STEP_FIELD_NUMBER: _ClassVar[int]
    id: str
    step: InvestigationStep
    def __init__(self, id: _Optional[str] = ..., step: _Optional[_Union[InvestigationStep, _Mapping]] = ...) -> None: ...

class AddInvestigationStepResponse(_message.Message):
    __slots__ = ("status",)
    STATUS_FIELD_NUMBER: _ClassVar[int]
    status: str
    def __init__(self, status: _Optional[str] = ...) -> None: ...

class GetCooldownStatusRequest(_message.Message):
    __slots__ = ()
    def __init__(self) -> None: ...

class GetCooldownStatusResponse(_message.Message):
    __slots__ = ("cooldown",)
    COOLDOWN_FIELD_NUMBER: _ClassVar[int]
    cooldown: CooldownStatus
    def __init__(self, cooldown: _Optional[_Union[CooldownStatus, _Mapping]] = ...) -> None: ...

class GetTriggersRequest(_message.Message):
    __slots__ = ()
    def __init__(self) -> None: ...

class GetTriggersResponse(_message.Message):
    __slots__ = ("triggers",)
    class TriggersEntry(_message.Message):
        __slots__ = ("key", "value")
        KEY_FIELD_NUMBER: _ClassVar[int]
        VALUE_FIELD_NUMBER: _ClassVar[int]
        key: str
        value: TriggerConfig
        def __init__(self, key: _Optional[str] = ..., value: _Optional[_Union[TriggerConfig, _Mapping]] = ...) -> None: ...
    TRIGGERS_FIELD_NUMBER: _ClassVar[int]
    triggers: _containers.MessageMap[str, TriggerConfig]
    def __init__(self, triggers: _Optional[_Mapping[str, TriggerConfig]] = ...) -> None: ...

class UpdateTriggerRequest(_message.Message):
    __slots__ = ("id", "enabled", "auto_fix", "threshold")
    ID_FIELD_NUMBER: _ClassVar[int]
    ENABLED_FIELD_NUMBER: _ClassVar[int]
    AUTO_FIX_FIELD_NUMBER: _ClassVar[int]
    THRESHOLD_FIELD_NUMBER: _ClassVar[int]
    id: str
    enabled: bool
    auto_fix: bool
    threshold: float
    def __init__(self, id: _Optional[str] = ..., enabled: _Optional[bool] = ..., auto_fix: _Optional[bool] = ..., threshold: _Optional[float] = ...) -> None: ...

class UpdateTriggerResponse(_message.Message):
    __slots__ = ("status",)
    STATUS_FIELD_NUMBER: _ClassVar[int]
    status: str
    def __init__(self, status: _Optional[str] = ...) -> None: ...
