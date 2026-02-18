from google.protobuf.internal import enum_type_wrapper as _enum_type_wrapper
from google.protobuf import descriptor as _descriptor
from typing import ClassVar as _ClassVar

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
