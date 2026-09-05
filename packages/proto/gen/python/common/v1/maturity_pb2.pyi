from common.v1 import validation_target_pb2 as _validation_target_pb2
from google.protobuf.internal import containers as _containers
from google.protobuf.internal import enum_type_wrapper as _enum_type_wrapper
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Iterable as _Iterable, Mapping as _Mapping
from typing import ClassVar as _ClassVar, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class GlobalImpact(int, metaclass=_enum_type_wrapper.EnumTypeWrapper):
    __slots__ = ()
    GLOBAL_IMPACT_UNSPECIFIED: _ClassVar[GlobalImpact]
    GLOBAL_IMPACT_FOUNDATION_BLOCKER: _ClassVar[GlobalImpact]
    GLOBAL_IMPACT_SAFETY_BLOCKER: _ClassVar[GlobalImpact]
    GLOBAL_IMPACT_EVOLVABILITY_GAP: _ClassVar[GlobalImpact]
    GLOBAL_IMPACT_HARDENING_GAP: _ClassVar[GlobalImpact]
    GLOBAL_IMPACT_CAPABILITY_GAP: _ClassVar[GlobalImpact]
    GLOBAL_IMPACT_ADVISORY: _ClassVar[GlobalImpact]
    GLOBAL_IMPACT_UNKNOWN: _ClassVar[GlobalImpact]

class CleanRequirement(int, metaclass=_enum_type_wrapper.EnumTypeWrapper):
    __slots__ = ()
    CLEAN_REQUIREMENT_UNSPECIFIED: _ClassVar[CleanRequirement]
    CLEAN_REQUIREMENT_REQUIRED: _ClassVar[CleanRequirement]
    CLEAN_REQUIREMENT_ADVISORY: _ClassVar[CleanRequirement]
    CLEAN_REQUIREMENT_UNCHECKABLE: _ClassVar[CleanRequirement]

class FixAffordance(int, metaclass=_enum_type_wrapper.EnumTypeWrapper):
    __slots__ = ()
    FIX_AFFORDANCE_UNSPECIFIED: _ClassVar[FixAffordance]
    FIX_AFFORDANCE_DETECTION_ONLY: _ClassVar[FixAffordance]
    FIX_AFFORDANCE_MANUAL: _ClassVar[FixAffordance]
    FIX_AFFORDANCE_PREVIEW_AVAILABLE: _ClassVar[FixAffordance]
GLOBAL_IMPACT_UNSPECIFIED: GlobalImpact
GLOBAL_IMPACT_FOUNDATION_BLOCKER: GlobalImpact
GLOBAL_IMPACT_SAFETY_BLOCKER: GlobalImpact
GLOBAL_IMPACT_EVOLVABILITY_GAP: GlobalImpact
GLOBAL_IMPACT_HARDENING_GAP: GlobalImpact
GLOBAL_IMPACT_CAPABILITY_GAP: GlobalImpact
GLOBAL_IMPACT_ADVISORY: GlobalImpact
GLOBAL_IMPACT_UNKNOWN: GlobalImpact
CLEAN_REQUIREMENT_UNSPECIFIED: CleanRequirement
CLEAN_REQUIREMENT_REQUIRED: CleanRequirement
CLEAN_REQUIREMENT_ADVISORY: CleanRequirement
CLEAN_REQUIREMENT_UNCHECKABLE: CleanRequirement
FIX_AFFORDANCE_UNSPECIFIED: FixAffordance
FIX_AFFORDANCE_DETECTION_ONLY: FixAffordance
FIX_AFFORDANCE_MANUAL: FixAffordance
FIX_AFFORDANCE_PREVIEW_AVAILABLE: FixAffordance

class LocalMaturityLevel(_message.Message):
    __slots__ = ("id", "name", "description", "entry_criteria", "exit_criteria", "status_label", "capability_summary", "next_unlock")
    ID_FIELD_NUMBER: _ClassVar[int]
    NAME_FIELD_NUMBER: _ClassVar[int]
    DESCRIPTION_FIELD_NUMBER: _ClassVar[int]
    ENTRY_CRITERIA_FIELD_NUMBER: _ClassVar[int]
    EXIT_CRITERIA_FIELD_NUMBER: _ClassVar[int]
    STATUS_LABEL_FIELD_NUMBER: _ClassVar[int]
    CAPABILITY_SUMMARY_FIELD_NUMBER: _ClassVar[int]
    NEXT_UNLOCK_FIELD_NUMBER: _ClassVar[int]
    id: str
    name: str
    description: str
    entry_criteria: _containers.RepeatedScalarFieldContainer[str]
    exit_criteria: _containers.RepeatedScalarFieldContainer[str]
    status_label: str
    capability_summary: str
    next_unlock: str
    def __init__(self, id: _Optional[str] = ..., name: _Optional[str] = ..., description: _Optional[str] = ..., entry_criteria: _Optional[_Iterable[str]] = ..., exit_criteria: _Optional[_Iterable[str]] = ..., status_label: _Optional[str] = ..., capability_summary: _Optional[str] = ..., next_unlock: _Optional[str] = ...) -> None: ...

class FindingMaturity(_message.Message):
    __slots__ = ("local_level", "global_impact", "dimension", "recommended_skill_ids", "clean_requirement", "capability_id")
    LOCAL_LEVEL_FIELD_NUMBER: _ClassVar[int]
    GLOBAL_IMPACT_FIELD_NUMBER: _ClassVar[int]
    DIMENSION_FIELD_NUMBER: _ClassVar[int]
    RECOMMENDED_SKILL_IDS_FIELD_NUMBER: _ClassVar[int]
    CLEAN_REQUIREMENT_FIELD_NUMBER: _ClassVar[int]
    CAPABILITY_ID_FIELD_NUMBER: _ClassVar[int]
    local_level: str
    global_impact: GlobalImpact
    dimension: str
    recommended_skill_ids: _containers.RepeatedScalarFieldContainer[str]
    clean_requirement: CleanRequirement
    capability_id: str
    def __init__(self, local_level: _Optional[str] = ..., global_impact: _Optional[_Union[GlobalImpact, str]] = ..., dimension: _Optional[str] = ..., recommended_skill_ids: _Optional[_Iterable[str]] = ..., clean_requirement: _Optional[_Union[CleanRequirement, str]] = ..., capability_id: _Optional[str] = ...) -> None: ...

class AssessmentFinding(_message.Message):
    __slots__ = ("code", "severity", "title", "message", "location", "remediation", "maturity", "autofix_available", "fix_class", "subject")
    CODE_FIELD_NUMBER: _ClassVar[int]
    SEVERITY_FIELD_NUMBER: _ClassVar[int]
    TITLE_FIELD_NUMBER: _ClassVar[int]
    MESSAGE_FIELD_NUMBER: _ClassVar[int]
    LOCATION_FIELD_NUMBER: _ClassVar[int]
    REMEDIATION_FIELD_NUMBER: _ClassVar[int]
    MATURITY_FIELD_NUMBER: _ClassVar[int]
    AUTOFIX_AVAILABLE_FIELD_NUMBER: _ClassVar[int]
    FIX_CLASS_FIELD_NUMBER: _ClassVar[int]
    SUBJECT_FIELD_NUMBER: _ClassVar[int]
    code: str
    severity: str
    title: str
    message: str
    location: str
    remediation: str
    maturity: FindingMaturity
    autofix_available: bool
    fix_class: str
    subject: _validation_target_pb2.ValidationTarget
    def __init__(self, code: _Optional[str] = ..., severity: _Optional[str] = ..., title: _Optional[str] = ..., message: _Optional[str] = ..., location: _Optional[str] = ..., remediation: _Optional[str] = ..., maturity: _Optional[_Union[FindingMaturity, _Mapping]] = ..., autofix_available: _Optional[bool] = ..., fix_class: _Optional[str] = ..., subject: _Optional[_Union[_validation_target_pb2.ValidationTarget, _Mapping]] = ...) -> None: ...

class LocalMaturityAssessment(_message.Message):
    __slots__ = ("current_level", "next_level", "levels", "blocking_finding_codes", "clean", "unknown_count")
    CURRENT_LEVEL_FIELD_NUMBER: _ClassVar[int]
    NEXT_LEVEL_FIELD_NUMBER: _ClassVar[int]
    LEVELS_FIELD_NUMBER: _ClassVar[int]
    BLOCKING_FINDING_CODES_FIELD_NUMBER: _ClassVar[int]
    CLEAN_FIELD_NUMBER: _ClassVar[int]
    UNKNOWN_COUNT_FIELD_NUMBER: _ClassVar[int]
    current_level: str
    next_level: str
    levels: _containers.RepeatedCompositeFieldContainer[LocalMaturityLevel]
    blocking_finding_codes: _containers.RepeatedScalarFieldContainer[str]
    clean: bool
    unknown_count: int
    def __init__(self, current_level: _Optional[str] = ..., next_level: _Optional[str] = ..., levels: _Optional[_Iterable[_Union[LocalMaturityLevel, _Mapping]]] = ..., blocking_finding_codes: _Optional[_Iterable[str]] = ..., clean: _Optional[bool] = ..., unknown_count: _Optional[int] = ...) -> None: ...

class CapabilityMaturityAssessment(_message.Message):
    __slots__ = ("id", "label", "description", "current_level", "next_level", "levels", "current_summary", "next_unlock", "blocking_finding_codes", "clean", "unknown_count", "findings_by_global_impact", "findings_by_severity", "findings_by_clean_requirement", "priority_rank", "priority_reason")
    class FindingsByGlobalImpactEntry(_message.Message):
        __slots__ = ("key", "value")
        KEY_FIELD_NUMBER: _ClassVar[int]
        VALUE_FIELD_NUMBER: _ClassVar[int]
        key: str
        value: int
        def __init__(self, key: _Optional[str] = ..., value: _Optional[int] = ...) -> None: ...
    class FindingsBySeverityEntry(_message.Message):
        __slots__ = ("key", "value")
        KEY_FIELD_NUMBER: _ClassVar[int]
        VALUE_FIELD_NUMBER: _ClassVar[int]
        key: str
        value: int
        def __init__(self, key: _Optional[str] = ..., value: _Optional[int] = ...) -> None: ...
    class FindingsByCleanRequirementEntry(_message.Message):
        __slots__ = ("key", "value")
        KEY_FIELD_NUMBER: _ClassVar[int]
        VALUE_FIELD_NUMBER: _ClassVar[int]
        key: str
        value: int
        def __init__(self, key: _Optional[str] = ..., value: _Optional[int] = ...) -> None: ...
    ID_FIELD_NUMBER: _ClassVar[int]
    LABEL_FIELD_NUMBER: _ClassVar[int]
    DESCRIPTION_FIELD_NUMBER: _ClassVar[int]
    CURRENT_LEVEL_FIELD_NUMBER: _ClassVar[int]
    NEXT_LEVEL_FIELD_NUMBER: _ClassVar[int]
    LEVELS_FIELD_NUMBER: _ClassVar[int]
    CURRENT_SUMMARY_FIELD_NUMBER: _ClassVar[int]
    NEXT_UNLOCK_FIELD_NUMBER: _ClassVar[int]
    BLOCKING_FINDING_CODES_FIELD_NUMBER: _ClassVar[int]
    CLEAN_FIELD_NUMBER: _ClassVar[int]
    UNKNOWN_COUNT_FIELD_NUMBER: _ClassVar[int]
    FINDINGS_BY_GLOBAL_IMPACT_FIELD_NUMBER: _ClassVar[int]
    FINDINGS_BY_SEVERITY_FIELD_NUMBER: _ClassVar[int]
    FINDINGS_BY_CLEAN_REQUIREMENT_FIELD_NUMBER: _ClassVar[int]
    PRIORITY_RANK_FIELD_NUMBER: _ClassVar[int]
    PRIORITY_REASON_FIELD_NUMBER: _ClassVar[int]
    id: str
    label: str
    description: str
    current_level: str
    next_level: str
    levels: _containers.RepeatedCompositeFieldContainer[LocalMaturityLevel]
    current_summary: str
    next_unlock: str
    blocking_finding_codes: _containers.RepeatedScalarFieldContainer[str]
    clean: bool
    unknown_count: int
    findings_by_global_impact: _containers.ScalarMap[str, int]
    findings_by_severity: _containers.ScalarMap[str, int]
    findings_by_clean_requirement: _containers.ScalarMap[str, int]
    priority_rank: int
    priority_reason: str
    def __init__(self, id: _Optional[str] = ..., label: _Optional[str] = ..., description: _Optional[str] = ..., current_level: _Optional[str] = ..., next_level: _Optional[str] = ..., levels: _Optional[_Iterable[_Union[LocalMaturityLevel, _Mapping]]] = ..., current_summary: _Optional[str] = ..., next_unlock: _Optional[str] = ..., blocking_finding_codes: _Optional[_Iterable[str]] = ..., clean: _Optional[bool] = ..., unknown_count: _Optional[int] = ..., findings_by_global_impact: _Optional[_Mapping[str, int]] = ..., findings_by_severity: _Optional[_Mapping[str, int]] = ..., findings_by_clean_requirement: _Optional[_Mapping[str, int]] = ..., priority_rank: _Optional[int] = ..., priority_reason: _Optional[str] = ...) -> None: ...

class PriorityFocus(_message.Message):
    __slots__ = ("capability_id", "capability_label", "current_level", "next_level", "reason")
    CAPABILITY_ID_FIELD_NUMBER: _ClassVar[int]
    CAPABILITY_LABEL_FIELD_NUMBER: _ClassVar[int]
    CURRENT_LEVEL_FIELD_NUMBER: _ClassVar[int]
    NEXT_LEVEL_FIELD_NUMBER: _ClassVar[int]
    REASON_FIELD_NUMBER: _ClassVar[int]
    capability_id: str
    capability_label: str
    current_level: str
    next_level: str
    reason: str
    def __init__(self, capability_id: _Optional[str] = ..., capability_label: _Optional[str] = ..., current_level: _Optional[str] = ..., next_level: _Optional[str] = ..., reason: _Optional[str] = ...) -> None: ...

class PhasePresentationFinding(_message.Message):
    __slots__ = ("code", "severity", "count", "title", "message", "locations", "remediation", "fix_affordance")
    CODE_FIELD_NUMBER: _ClassVar[int]
    SEVERITY_FIELD_NUMBER: _ClassVar[int]
    COUNT_FIELD_NUMBER: _ClassVar[int]
    TITLE_FIELD_NUMBER: _ClassVar[int]
    MESSAGE_FIELD_NUMBER: _ClassVar[int]
    LOCATIONS_FIELD_NUMBER: _ClassVar[int]
    REMEDIATION_FIELD_NUMBER: _ClassVar[int]
    FIX_AFFORDANCE_FIELD_NUMBER: _ClassVar[int]
    code: str
    severity: str
    count: int
    title: str
    message: str
    locations: _containers.RepeatedScalarFieldContainer[str]
    remediation: str
    fix_affordance: FixAffordance
    def __init__(self, code: _Optional[str] = ..., severity: _Optional[str] = ..., count: _Optional[int] = ..., title: _Optional[str] = ..., message: _Optional[str] = ..., locations: _Optional[_Iterable[str]] = ..., remediation: _Optional[str] = ..., fix_affordance: _Optional[_Union[FixAffordance, str]] = ...) -> None: ...

class PhaseCapabilityPresentation(_message.Message):
    __slots__ = ("id", "label", "current_level", "current_level_label", "next_level", "current_summary", "next_unlock", "clean", "unknown_count", "blocking_finding_codes", "priority_rank", "priority_reason", "findings")
    ID_FIELD_NUMBER: _ClassVar[int]
    LABEL_FIELD_NUMBER: _ClassVar[int]
    CURRENT_LEVEL_FIELD_NUMBER: _ClassVar[int]
    CURRENT_LEVEL_LABEL_FIELD_NUMBER: _ClassVar[int]
    NEXT_LEVEL_FIELD_NUMBER: _ClassVar[int]
    CURRENT_SUMMARY_FIELD_NUMBER: _ClassVar[int]
    NEXT_UNLOCK_FIELD_NUMBER: _ClassVar[int]
    CLEAN_FIELD_NUMBER: _ClassVar[int]
    UNKNOWN_COUNT_FIELD_NUMBER: _ClassVar[int]
    BLOCKING_FINDING_CODES_FIELD_NUMBER: _ClassVar[int]
    PRIORITY_RANK_FIELD_NUMBER: _ClassVar[int]
    PRIORITY_REASON_FIELD_NUMBER: _ClassVar[int]
    FINDINGS_FIELD_NUMBER: _ClassVar[int]
    id: str
    label: str
    current_level: str
    current_level_label: str
    next_level: str
    current_summary: str
    next_unlock: str
    clean: bool
    unknown_count: int
    blocking_finding_codes: _containers.RepeatedScalarFieldContainer[str]
    priority_rank: int
    priority_reason: str
    findings: _containers.RepeatedCompositeFieldContainer[PhasePresentationFinding]
    def __init__(self, id: _Optional[str] = ..., label: _Optional[str] = ..., current_level: _Optional[str] = ..., current_level_label: _Optional[str] = ..., next_level: _Optional[str] = ..., current_summary: _Optional[str] = ..., next_unlock: _Optional[str] = ..., clean: _Optional[bool] = ..., unknown_count: _Optional[int] = ..., blocking_finding_codes: _Optional[_Iterable[str]] = ..., priority_rank: _Optional[int] = ..., priority_reason: _Optional[str] = ..., findings: _Optional[_Iterable[_Union[PhasePresentationFinding, _Mapping]]] = ...) -> None: ...

class PhasePresentation(_message.Message):
    __slots__ = ("contract_version", "provider", "phase", "current_level", "current_level_label", "next_level", "ceiling_level", "clean", "unknown_count", "blocking_finding_codes", "next_action", "next_action_reason", "focus_capability_id", "focus_capability_label", "north_star", "at_maximum", "capabilities", "documentation_topics")
    CONTRACT_VERSION_FIELD_NUMBER: _ClassVar[int]
    PROVIDER_FIELD_NUMBER: _ClassVar[int]
    PHASE_FIELD_NUMBER: _ClassVar[int]
    CURRENT_LEVEL_FIELD_NUMBER: _ClassVar[int]
    CURRENT_LEVEL_LABEL_FIELD_NUMBER: _ClassVar[int]
    NEXT_LEVEL_FIELD_NUMBER: _ClassVar[int]
    CEILING_LEVEL_FIELD_NUMBER: _ClassVar[int]
    CLEAN_FIELD_NUMBER: _ClassVar[int]
    UNKNOWN_COUNT_FIELD_NUMBER: _ClassVar[int]
    BLOCKING_FINDING_CODES_FIELD_NUMBER: _ClassVar[int]
    NEXT_ACTION_FIELD_NUMBER: _ClassVar[int]
    NEXT_ACTION_REASON_FIELD_NUMBER: _ClassVar[int]
    FOCUS_CAPABILITY_ID_FIELD_NUMBER: _ClassVar[int]
    FOCUS_CAPABILITY_LABEL_FIELD_NUMBER: _ClassVar[int]
    NORTH_STAR_FIELD_NUMBER: _ClassVar[int]
    AT_MAXIMUM_FIELD_NUMBER: _ClassVar[int]
    CAPABILITIES_FIELD_NUMBER: _ClassVar[int]
    DOCUMENTATION_TOPICS_FIELD_NUMBER: _ClassVar[int]
    contract_version: str
    provider: str
    phase: str
    current_level: str
    current_level_label: str
    next_level: str
    ceiling_level: str
    clean: bool
    unknown_count: int
    blocking_finding_codes: _containers.RepeatedScalarFieldContainer[str]
    next_action: str
    next_action_reason: str
    focus_capability_id: str
    focus_capability_label: str
    north_star: str
    at_maximum: bool
    capabilities: _containers.RepeatedCompositeFieldContainer[PhaseCapabilityPresentation]
    documentation_topics: _containers.RepeatedScalarFieldContainer[str]
    def __init__(self, contract_version: _Optional[str] = ..., provider: _Optional[str] = ..., phase: _Optional[str] = ..., current_level: _Optional[str] = ..., current_level_label: _Optional[str] = ..., next_level: _Optional[str] = ..., ceiling_level: _Optional[str] = ..., clean: _Optional[bool] = ..., unknown_count: _Optional[int] = ..., blocking_finding_codes: _Optional[_Iterable[str]] = ..., next_action: _Optional[str] = ..., next_action_reason: _Optional[str] = ..., focus_capability_id: _Optional[str] = ..., focus_capability_label: _Optional[str] = ..., north_star: _Optional[str] = ..., at_maximum: _Optional[bool] = ..., capabilities: _Optional[_Iterable[_Union[PhaseCapabilityPresentation, _Mapping]]] = ..., documentation_topics: _Optional[_Iterable[str]] = ...) -> None: ...

class MaturityAssessment(_message.Message):
    __slots__ = ("scenario", "provider", "phase", "version", "local", "findings", "findings_by_global_impact", "findings_by_severity", "recommended_skill_ids", "findings_by_clean_requirement", "autofixable_count", "autofixable_total", "capabilities", "highest_priority_capability", "presentation")
    class FindingsByGlobalImpactEntry(_message.Message):
        __slots__ = ("key", "value")
        KEY_FIELD_NUMBER: _ClassVar[int]
        VALUE_FIELD_NUMBER: _ClassVar[int]
        key: str
        value: int
        def __init__(self, key: _Optional[str] = ..., value: _Optional[int] = ...) -> None: ...
    class FindingsBySeverityEntry(_message.Message):
        __slots__ = ("key", "value")
        KEY_FIELD_NUMBER: _ClassVar[int]
        VALUE_FIELD_NUMBER: _ClassVar[int]
        key: str
        value: int
        def __init__(self, key: _Optional[str] = ..., value: _Optional[int] = ...) -> None: ...
    class FindingsByCleanRequirementEntry(_message.Message):
        __slots__ = ("key", "value")
        KEY_FIELD_NUMBER: _ClassVar[int]
        VALUE_FIELD_NUMBER: _ClassVar[int]
        key: str
        value: int
        def __init__(self, key: _Optional[str] = ..., value: _Optional[int] = ...) -> None: ...
    SCENARIO_FIELD_NUMBER: _ClassVar[int]
    PROVIDER_FIELD_NUMBER: _ClassVar[int]
    PHASE_FIELD_NUMBER: _ClassVar[int]
    VERSION_FIELD_NUMBER: _ClassVar[int]
    LOCAL_FIELD_NUMBER: _ClassVar[int]
    FINDINGS_FIELD_NUMBER: _ClassVar[int]
    FINDINGS_BY_GLOBAL_IMPACT_FIELD_NUMBER: _ClassVar[int]
    FINDINGS_BY_SEVERITY_FIELD_NUMBER: _ClassVar[int]
    RECOMMENDED_SKILL_IDS_FIELD_NUMBER: _ClassVar[int]
    FINDINGS_BY_CLEAN_REQUIREMENT_FIELD_NUMBER: _ClassVar[int]
    AUTOFIXABLE_COUNT_FIELD_NUMBER: _ClassVar[int]
    AUTOFIXABLE_TOTAL_FIELD_NUMBER: _ClassVar[int]
    CAPABILITIES_FIELD_NUMBER: _ClassVar[int]
    HIGHEST_PRIORITY_CAPABILITY_FIELD_NUMBER: _ClassVar[int]
    PRESENTATION_FIELD_NUMBER: _ClassVar[int]
    scenario: str
    provider: str
    phase: str
    version: str
    local: LocalMaturityAssessment
    findings: _containers.RepeatedCompositeFieldContainer[AssessmentFinding]
    findings_by_global_impact: _containers.ScalarMap[str, int]
    findings_by_severity: _containers.ScalarMap[str, int]
    recommended_skill_ids: _containers.RepeatedScalarFieldContainer[str]
    findings_by_clean_requirement: _containers.ScalarMap[str, int]
    autofixable_count: int
    autofixable_total: int
    capabilities: _containers.RepeatedCompositeFieldContainer[CapabilityMaturityAssessment]
    highest_priority_capability: PriorityFocus
    presentation: PhasePresentation
    def __init__(self, scenario: _Optional[str] = ..., provider: _Optional[str] = ..., phase: _Optional[str] = ..., version: _Optional[str] = ..., local: _Optional[_Union[LocalMaturityAssessment, _Mapping]] = ..., findings: _Optional[_Iterable[_Union[AssessmentFinding, _Mapping]]] = ..., findings_by_global_impact: _Optional[_Mapping[str, int]] = ..., findings_by_severity: _Optional[_Mapping[str, int]] = ..., recommended_skill_ids: _Optional[_Iterable[str]] = ..., findings_by_clean_requirement: _Optional[_Mapping[str, int]] = ..., autofixable_count: _Optional[int] = ..., autofixable_total: _Optional[int] = ..., capabilities: _Optional[_Iterable[_Union[CapabilityMaturityAssessment, _Mapping]]] = ..., highest_priority_capability: _Optional[_Union[PriorityFocus, _Mapping]] = ..., presentation: _Optional[_Union[PhasePresentation, _Mapping]] = ...) -> None: ...
