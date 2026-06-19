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

class LocalMaturityLevel(_message.Message):
    __slots__ = ("id", "name", "description", "entry_criteria", "exit_criteria")
    ID_FIELD_NUMBER: _ClassVar[int]
    NAME_FIELD_NUMBER: _ClassVar[int]
    DESCRIPTION_FIELD_NUMBER: _ClassVar[int]
    ENTRY_CRITERIA_FIELD_NUMBER: _ClassVar[int]
    EXIT_CRITERIA_FIELD_NUMBER: _ClassVar[int]
    id: str
    name: str
    description: str
    entry_criteria: _containers.RepeatedScalarFieldContainer[str]
    exit_criteria: _containers.RepeatedScalarFieldContainer[str]
    def __init__(self, id: _Optional[str] = ..., name: _Optional[str] = ..., description: _Optional[str] = ..., entry_criteria: _Optional[_Iterable[str]] = ..., exit_criteria: _Optional[_Iterable[str]] = ...) -> None: ...

class FindingMaturity(_message.Message):
    __slots__ = ("local_level", "global_impact", "dimension", "recommended_skill_ids", "clean_requirement")
    LOCAL_LEVEL_FIELD_NUMBER: _ClassVar[int]
    GLOBAL_IMPACT_FIELD_NUMBER: _ClassVar[int]
    DIMENSION_FIELD_NUMBER: _ClassVar[int]
    RECOMMENDED_SKILL_IDS_FIELD_NUMBER: _ClassVar[int]
    CLEAN_REQUIREMENT_FIELD_NUMBER: _ClassVar[int]
    local_level: str
    global_impact: GlobalImpact
    dimension: str
    recommended_skill_ids: _containers.RepeatedScalarFieldContainer[str]
    clean_requirement: CleanRequirement
    def __init__(self, local_level: _Optional[str] = ..., global_impact: _Optional[_Union[GlobalImpact, str]] = ..., dimension: _Optional[str] = ..., recommended_skill_ids: _Optional[_Iterable[str]] = ..., clean_requirement: _Optional[_Union[CleanRequirement, str]] = ...) -> None: ...

class AssessmentFinding(_message.Message):
    __slots__ = ("code", "severity", "title", "message", "location", "remediation", "maturity")
    CODE_FIELD_NUMBER: _ClassVar[int]
    SEVERITY_FIELD_NUMBER: _ClassVar[int]
    TITLE_FIELD_NUMBER: _ClassVar[int]
    MESSAGE_FIELD_NUMBER: _ClassVar[int]
    LOCATION_FIELD_NUMBER: _ClassVar[int]
    REMEDIATION_FIELD_NUMBER: _ClassVar[int]
    MATURITY_FIELD_NUMBER: _ClassVar[int]
    code: str
    severity: str
    title: str
    message: str
    location: str
    remediation: str
    maturity: FindingMaturity
    def __init__(self, code: _Optional[str] = ..., severity: _Optional[str] = ..., title: _Optional[str] = ..., message: _Optional[str] = ..., location: _Optional[str] = ..., remediation: _Optional[str] = ..., maturity: _Optional[_Union[FindingMaturity, _Mapping]] = ...) -> None: ...

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

class MaturityAssessment(_message.Message):
    __slots__ = ("scenario", "provider", "phase", "version", "local", "findings", "findings_by_global_impact", "findings_by_severity", "recommended_skill_ids", "findings_by_clean_requirement")
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
    def __init__(self, scenario: _Optional[str] = ..., provider: _Optional[str] = ..., phase: _Optional[str] = ..., version: _Optional[str] = ..., local: _Optional[_Union[LocalMaturityAssessment, _Mapping]] = ..., findings: _Optional[_Iterable[_Union[AssessmentFinding, _Mapping]]] = ..., findings_by_global_impact: _Optional[_Mapping[str, int]] = ..., findings_by_severity: _Optional[_Mapping[str, int]] = ..., recommended_skill_ids: _Optional[_Iterable[str]] = ..., findings_by_clean_requirement: _Optional[_Mapping[str, int]] = ...) -> None: ...
