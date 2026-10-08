from buf.validate import validate_pb2 as _validate_pb2
from swarm_manager.v1.shared import health_pb2 as _health_pb2
from google.protobuf.internal import containers as _containers
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Iterable as _Iterable, Mapping as _Mapping
from typing import ClassVar as _ClassVar, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class Scenario(_message.Message):
    __slots__ = ("name", "display_name", "description", "status", "priority", "completeness_score", "is_greenfield", "tags", "last_review_classification", "last_review_at", "health")
    NAME_FIELD_NUMBER: _ClassVar[int]
    DISPLAY_NAME_FIELD_NUMBER: _ClassVar[int]
    DESCRIPTION_FIELD_NUMBER: _ClassVar[int]
    STATUS_FIELD_NUMBER: _ClassVar[int]
    PRIORITY_FIELD_NUMBER: _ClassVar[int]
    COMPLETENESS_SCORE_FIELD_NUMBER: _ClassVar[int]
    IS_GREENFIELD_FIELD_NUMBER: _ClassVar[int]
    TAGS_FIELD_NUMBER: _ClassVar[int]
    LAST_REVIEW_CLASSIFICATION_FIELD_NUMBER: _ClassVar[int]
    LAST_REVIEW_AT_FIELD_NUMBER: _ClassVar[int]
    HEALTH_FIELD_NUMBER: _ClassVar[int]
    name: str
    display_name: str
    description: str
    status: str
    priority: int
    completeness_score: int
    is_greenfield: bool
    tags: _containers.RepeatedScalarFieldContainer[str]
    last_review_classification: str
    last_review_at: str
    health: ScenarioHealthSnapshot
    def __init__(self, name: _Optional[str] = ..., display_name: _Optional[str] = ..., description: _Optional[str] = ..., status: _Optional[str] = ..., priority: _Optional[int] = ..., completeness_score: _Optional[int] = ..., is_greenfield: _Optional[bool] = ..., tags: _Optional[_Iterable[str]] = ..., last_review_classification: _Optional[str] = ..., last_review_at: _Optional[str] = ..., health: _Optional[_Union[ScenarioHealthSnapshot, _Mapping]] = ...) -> None: ...

class ScenarioHealthSnapshot(_message.Message):
    __slots__ = ("evidence_state", "reason", "source_run_id", "observed_at", "freshness", "verdict", "phases", "remediation")
    EVIDENCE_STATE_FIELD_NUMBER: _ClassVar[int]
    REASON_FIELD_NUMBER: _ClassVar[int]
    SOURCE_RUN_ID_FIELD_NUMBER: _ClassVar[int]
    OBSERVED_AT_FIELD_NUMBER: _ClassVar[int]
    FRESHNESS_FIELD_NUMBER: _ClassVar[int]
    VERDICT_FIELD_NUMBER: _ClassVar[int]
    PHASES_FIELD_NUMBER: _ClassVar[int]
    REMEDIATION_FIELD_NUMBER: _ClassVar[int]
    evidence_state: str
    reason: str
    source_run_id: str
    observed_at: str
    freshness: str
    verdict: str
    phases: _containers.RepeatedCompositeFieldContainer[ScenarioHealthPhase]
    remediation: _containers.RepeatedCompositeFieldContainer[_health_pb2.ScenarioRemediationSummary]
    def __init__(self, evidence_state: _Optional[str] = ..., reason: _Optional[str] = ..., source_run_id: _Optional[str] = ..., observed_at: _Optional[str] = ..., freshness: _Optional[str] = ..., verdict: _Optional[str] = ..., phases: _Optional[_Iterable[_Union[ScenarioHealthPhase, _Mapping]]] = ..., remediation: _Optional[_Iterable[_Union[_health_pb2.ScenarioRemediationSummary, _Mapping]]] = ...) -> None: ...

class ScenarioHealthPhase(_message.Message):
    __slots__ = ("phase", "label", "verdict", "current_rung", "next_rung", "priority_capability_id", "priority_capability_label", "blocking_codes", "remediation_topics")
    PHASE_FIELD_NUMBER: _ClassVar[int]
    LABEL_FIELD_NUMBER: _ClassVar[int]
    VERDICT_FIELD_NUMBER: _ClassVar[int]
    CURRENT_RUNG_FIELD_NUMBER: _ClassVar[int]
    NEXT_RUNG_FIELD_NUMBER: _ClassVar[int]
    PRIORITY_CAPABILITY_ID_FIELD_NUMBER: _ClassVar[int]
    PRIORITY_CAPABILITY_LABEL_FIELD_NUMBER: _ClassVar[int]
    BLOCKING_CODES_FIELD_NUMBER: _ClassVar[int]
    REMEDIATION_TOPICS_FIELD_NUMBER: _ClassVar[int]
    phase: str
    label: str
    verdict: str
    current_rung: str
    next_rung: str
    priority_capability_id: str
    priority_capability_label: str
    blocking_codes: _containers.RepeatedScalarFieldContainer[str]
    remediation_topics: _containers.RepeatedScalarFieldContainer[str]
    def __init__(self, phase: _Optional[str] = ..., label: _Optional[str] = ..., verdict: _Optional[str] = ..., current_rung: _Optional[str] = ..., next_rung: _Optional[str] = ..., priority_capability_id: _Optional[str] = ..., priority_capability_label: _Optional[str] = ..., blocking_codes: _Optional[_Iterable[str]] = ..., remediation_topics: _Optional[_Iterable[str]] = ...) -> None: ...

class ScenarioMetadata(_message.Message):
    __slots__ = ("is_greenfield",)
    IS_GREENFIELD_FIELD_NUMBER: _ClassVar[int]
    is_greenfield: bool
    def __init__(self, is_greenfield: _Optional[bool] = ...) -> None: ...
