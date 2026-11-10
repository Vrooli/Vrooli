from flow_verifier.v1.artifacts import artifacts_pb2 as _artifacts_pb2
from flow_verifier.v1.flows import flows_pb2 as _flows_pb2
from google.protobuf.internal import containers as _containers
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Iterable as _Iterable, Mapping as _Mapping
from typing import ClassVar as _ClassVar, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class ScenarioSummary(_message.Message):
    __slots__ = ("id", "display_name", "description", "path", "flow_count", "discovery_error")
    ID_FIELD_NUMBER: _ClassVar[int]
    DISPLAY_NAME_FIELD_NUMBER: _ClassVar[int]
    DESCRIPTION_FIELD_NUMBER: _ClassVar[int]
    PATH_FIELD_NUMBER: _ClassVar[int]
    FLOW_COUNT_FIELD_NUMBER: _ClassVar[int]
    DISCOVERY_ERROR_FIELD_NUMBER: _ClassVar[int]
    id: str
    display_name: str
    description: str
    path: str
    flow_count: int
    discovery_error: str
    def __init__(self, id: _Optional[str] = ..., display_name: _Optional[str] = ..., description: _Optional[str] = ..., path: _Optional[str] = ..., flow_count: _Optional[int] = ..., discovery_error: _Optional[str] = ...) -> None: ...

class ScenarioDetail(_message.Message):
    __slots__ = ("summary", "flows")
    SUMMARY_FIELD_NUMBER: _ClassVar[int]
    FLOWS_FIELD_NUMBER: _ClassVar[int]
    summary: ScenarioSummary
    flows: _containers.RepeatedCompositeFieldContainer[_flows_pb2.FlowSummary]
    def __init__(self, summary: _Optional[_Union[ScenarioSummary, _Mapping]] = ..., flows: _Optional[_Iterable[_Union[_flows_pb2.FlowSummary, _Mapping]]] = ...) -> None: ...

class ListScenariosRequest(_message.Message):
    __slots__ = ()
    def __init__(self) -> None: ...

class ListScenariosResponse(_message.Message):
    __slots__ = ("vrooli_root", "scenarios")
    VROOLI_ROOT_FIELD_NUMBER: _ClassVar[int]
    SCENARIOS_FIELD_NUMBER: _ClassVar[int]
    vrooli_root: str
    scenarios: _containers.RepeatedCompositeFieldContainer[ScenarioSummary]
    def __init__(self, vrooli_root: _Optional[str] = ..., scenarios: _Optional[_Iterable[_Union[ScenarioSummary, _Mapping]]] = ...) -> None: ...

class GetScenarioRequest(_message.Message):
    __slots__ = ("id",)
    ID_FIELD_NUMBER: _ClassVar[int]
    id: str
    def __init__(self, id: _Optional[str] = ...) -> None: ...

class GetScenarioResponse(_message.Message):
    __slots__ = ("scenario",)
    SCENARIO_FIELD_NUMBER: _ClassVar[int]
    scenario: ScenarioDetail
    def __init__(self, scenario: _Optional[_Union[ScenarioDetail, _Mapping]] = ...) -> None: ...

class ListLifecyclesRequest(_message.Message):
    __slots__ = ("scenario_id", "hunt_index_json")
    SCENARIO_ID_FIELD_NUMBER: _ClassVar[int]
    HUNT_INDEX_JSON_FIELD_NUMBER: _ClassVar[int]
    scenario_id: str
    hunt_index_json: bytes
    def __init__(self, scenario_id: _Optional[str] = ..., hunt_index_json: _Optional[bytes] = ...) -> None: ...

class ListLifecyclesResponse(_message.Message):
    __slots__ = ("vrooli_root", "scenario_count", "candidate_count", "parse_error_count", "scenarios", "candidates", "flow_contract_parse_error_count", "preselection_candidate_count", "hunt_coverage_complete", "hunt_coverage_scope", "preselection_candidates", "unranked_candidates")
    VROOLI_ROOT_FIELD_NUMBER: _ClassVar[int]
    SCENARIO_COUNT_FIELD_NUMBER: _ClassVar[int]
    CANDIDATE_COUNT_FIELD_NUMBER: _ClassVar[int]
    PARSE_ERROR_COUNT_FIELD_NUMBER: _ClassVar[int]
    SCENARIOS_FIELD_NUMBER: _ClassVar[int]
    CANDIDATES_FIELD_NUMBER: _ClassVar[int]
    FLOW_CONTRACT_PARSE_ERROR_COUNT_FIELD_NUMBER: _ClassVar[int]
    PRESELECTION_CANDIDATE_COUNT_FIELD_NUMBER: _ClassVar[int]
    HUNT_COVERAGE_COMPLETE_FIELD_NUMBER: _ClassVar[int]
    HUNT_COVERAGE_SCOPE_FIELD_NUMBER: _ClassVar[int]
    PRESELECTION_CANDIDATES_FIELD_NUMBER: _ClassVar[int]
    UNRANKED_CANDIDATES_FIELD_NUMBER: _ClassVar[int]
    vrooli_root: str
    scenario_count: int
    candidate_count: int
    parse_error_count: int
    scenarios: _containers.RepeatedCompositeFieldContainer[LifecycleScenarioSummary]
    candidates: _containers.RepeatedCompositeFieldContainer[LifecycleCandidate]
    flow_contract_parse_error_count: int
    preselection_candidate_count: int
    hunt_coverage_complete: bool
    hunt_coverage_scope: str
    preselection_candidates: _containers.RepeatedCompositeFieldContainer[LifecycleCandidate]
    unranked_candidates: _containers.RepeatedCompositeFieldContainer[LifecycleCandidate]
    def __init__(self, vrooli_root: _Optional[str] = ..., scenario_count: _Optional[int] = ..., candidate_count: _Optional[int] = ..., parse_error_count: _Optional[int] = ..., scenarios: _Optional[_Iterable[_Union[LifecycleScenarioSummary, _Mapping]]] = ..., candidates: _Optional[_Iterable[_Union[LifecycleCandidate, _Mapping]]] = ..., flow_contract_parse_error_count: _Optional[int] = ..., preselection_candidate_count: _Optional[int] = ..., hunt_coverage_complete: _Optional[bool] = ..., hunt_coverage_scope: _Optional[str] = ..., preselection_candidates: _Optional[_Iterable[_Union[LifecycleCandidate, _Mapping]]] = ..., unranked_candidates: _Optional[_Iterable[_Union[LifecycleCandidate, _Mapping]]] = ...) -> None: ...

class LifecycleScenarioSummary(_message.Message):
    __slots__ = ("id", "source_files", "parse_errors", "candidate_count", "explicit_flow_contracts", "flow_contract_parse_errors")
    ID_FIELD_NUMBER: _ClassVar[int]
    SOURCE_FILES_FIELD_NUMBER: _ClassVar[int]
    PARSE_ERRORS_FIELD_NUMBER: _ClassVar[int]
    CANDIDATE_COUNT_FIELD_NUMBER: _ClassVar[int]
    EXPLICIT_FLOW_CONTRACTS_FIELD_NUMBER: _ClassVar[int]
    FLOW_CONTRACT_PARSE_ERRORS_FIELD_NUMBER: _ClassVar[int]
    id: str
    source_files: int
    parse_errors: int
    candidate_count: int
    explicit_flow_contracts: int
    flow_contract_parse_errors: int
    def __init__(self, id: _Optional[str] = ..., source_files: _Optional[int] = ..., parse_errors: _Optional[int] = ..., candidate_count: _Optional[int] = ..., explicit_flow_contracts: _Optional[int] = ..., flow_contract_parse_errors: _Optional[int] = ...) -> None: ...

class LifecycleEvidence(_message.Message):
    __slots__ = ("path", "line", "text")
    PATH_FIELD_NUMBER: _ClassVar[int]
    LINE_FIELD_NUMBER: _ClassVar[int]
    TEXT_FIELD_NUMBER: _ClassVar[int]
    path: str
    line: int
    text: str
    def __init__(self, path: _Optional[str] = ..., line: _Optional[int] = ..., text: _Optional[str] = ...) -> None: ...

class LifecycleStateEvidence(_message.Message):
    __slots__ = ("name", "value", "at")
    NAME_FIELD_NUMBER: _ClassVar[int]
    VALUE_FIELD_NUMBER: _ClassVar[int]
    AT_FIELD_NUMBER: _ClassVar[int]
    name: str
    value: str
    at: LifecycleEvidence
    def __init__(self, name: _Optional[str] = ..., value: _Optional[str] = ..., at: _Optional[_Union[LifecycleEvidence, _Mapping]] = ...) -> None: ...

class LifecycleHuntReference(_message.Message):
    __slots__ = ("entry_id", "round", "status")
    ENTRY_ID_FIELD_NUMBER: _ClassVar[int]
    ROUND_FIELD_NUMBER: _ClassVar[int]
    STATUS_FIELD_NUMBER: _ClassVar[int]
    entry_id: str
    round: str
    status: str
    def __init__(self, entry_id: _Optional[str] = ..., round: _Optional[str] = ..., status: _Optional[str] = ...) -> None: ...

class LifecycleFlowReference(_message.Message):
    __slots__ = ("flow_id", "path", "status_type", "matched_states", "candidate_state_count", "contract_state_count", "exact_lifecycle_match")
    FLOW_ID_FIELD_NUMBER: _ClassVar[int]
    PATH_FIELD_NUMBER: _ClassVar[int]
    STATUS_TYPE_FIELD_NUMBER: _ClassVar[int]
    MATCHED_STATES_FIELD_NUMBER: _ClassVar[int]
    CANDIDATE_STATE_COUNT_FIELD_NUMBER: _ClassVar[int]
    CONTRACT_STATE_COUNT_FIELD_NUMBER: _ClassVar[int]
    EXACT_LIFECYCLE_MATCH_FIELD_NUMBER: _ClassVar[int]
    flow_id: str
    path: str
    status_type: str
    matched_states: _containers.RepeatedScalarFieldContainer[str]
    candidate_state_count: int
    contract_state_count: int
    exact_lifecycle_match: bool
    def __init__(self, flow_id: _Optional[str] = ..., path: _Optional[str] = ..., status_type: _Optional[str] = ..., matched_states: _Optional[_Iterable[str]] = ..., candidate_state_count: _Optional[int] = ..., contract_state_count: _Optional[int] = ..., exact_lifecycle_match: _Optional[bool] = ...) -> None: ...

class LifecycleCandidate(_message.Message):
    __slots__ = ("id", "scenario_id", "package", "type_name", "field_name", "field_type", "declaration", "states", "related_flags", "time_fields", "write_sites", "guard_sites", "test_files", "hunt_records", "burden_score", "score_breakdown", "confidence", "explicit_flows", "explicitly_modeled", "transition_owners", "persistence_class", "persistence_reason", "persistence_evidence", "persistence_scope", "persistence_files_scanned", "persistence_scan_complete", "preselection_status")
    ID_FIELD_NUMBER: _ClassVar[int]
    SCENARIO_ID_FIELD_NUMBER: _ClassVar[int]
    PACKAGE_FIELD_NUMBER: _ClassVar[int]
    TYPE_NAME_FIELD_NUMBER: _ClassVar[int]
    FIELD_NAME_FIELD_NUMBER: _ClassVar[int]
    FIELD_TYPE_FIELD_NUMBER: _ClassVar[int]
    DECLARATION_FIELD_NUMBER: _ClassVar[int]
    STATES_FIELD_NUMBER: _ClassVar[int]
    RELATED_FLAGS_FIELD_NUMBER: _ClassVar[int]
    TIME_FIELDS_FIELD_NUMBER: _ClassVar[int]
    WRITE_SITES_FIELD_NUMBER: _ClassVar[int]
    GUARD_SITES_FIELD_NUMBER: _ClassVar[int]
    TEST_FILES_FIELD_NUMBER: _ClassVar[int]
    HUNT_RECORDS_FIELD_NUMBER: _ClassVar[int]
    BURDEN_SCORE_FIELD_NUMBER: _ClassVar[int]
    SCORE_BREAKDOWN_FIELD_NUMBER: _ClassVar[int]
    CONFIDENCE_FIELD_NUMBER: _ClassVar[int]
    EXPLICIT_FLOWS_FIELD_NUMBER: _ClassVar[int]
    EXPLICITLY_MODELED_FIELD_NUMBER: _ClassVar[int]
    TRANSITION_OWNERS_FIELD_NUMBER: _ClassVar[int]
    PERSISTENCE_CLASS_FIELD_NUMBER: _ClassVar[int]
    PERSISTENCE_REASON_FIELD_NUMBER: _ClassVar[int]
    PERSISTENCE_EVIDENCE_FIELD_NUMBER: _ClassVar[int]
    PERSISTENCE_SCOPE_FIELD_NUMBER: _ClassVar[int]
    PERSISTENCE_FILES_SCANNED_FIELD_NUMBER: _ClassVar[int]
    PERSISTENCE_SCAN_COMPLETE_FIELD_NUMBER: _ClassVar[int]
    PRESELECTION_STATUS_FIELD_NUMBER: _ClassVar[int]
    id: str
    scenario_id: str
    package: str
    type_name: str
    field_name: str
    field_type: str
    declaration: LifecycleEvidence
    states: _containers.RepeatedCompositeFieldContainer[LifecycleStateEvidence]
    related_flags: _containers.RepeatedCompositeFieldContainer[LifecycleEvidence]
    time_fields: _containers.RepeatedCompositeFieldContainer[LifecycleEvidence]
    write_sites: _containers.RepeatedCompositeFieldContainer[LifecycleEvidence]
    guard_sites: _containers.RepeatedCompositeFieldContainer[LifecycleEvidence]
    test_files: _containers.RepeatedScalarFieldContainer[str]
    hunt_records: _containers.RepeatedCompositeFieldContainer[LifecycleHuntReference]
    burden_score: int
    score_breakdown: _containers.RepeatedScalarFieldContainer[str]
    confidence: str
    explicit_flows: _containers.RepeatedCompositeFieldContainer[LifecycleFlowReference]
    explicitly_modeled: bool
    transition_owners: _containers.RepeatedCompositeFieldContainer[LifecycleEvidence]
    persistence_class: str
    persistence_reason: str
    persistence_evidence: _containers.RepeatedCompositeFieldContainer[LifecycleEvidence]
    persistence_scope: str
    persistence_files_scanned: int
    persistence_scan_complete: bool
    preselection_status: str
    def __init__(self, id: _Optional[str] = ..., scenario_id: _Optional[str] = ..., package: _Optional[str] = ..., type_name: _Optional[str] = ..., field_name: _Optional[str] = ..., field_type: _Optional[str] = ..., declaration: _Optional[_Union[LifecycleEvidence, _Mapping]] = ..., states: _Optional[_Iterable[_Union[LifecycleStateEvidence, _Mapping]]] = ..., related_flags: _Optional[_Iterable[_Union[LifecycleEvidence, _Mapping]]] = ..., time_fields: _Optional[_Iterable[_Union[LifecycleEvidence, _Mapping]]] = ..., write_sites: _Optional[_Iterable[_Union[LifecycleEvidence, _Mapping]]] = ..., guard_sites: _Optional[_Iterable[_Union[LifecycleEvidence, _Mapping]]] = ..., test_files: _Optional[_Iterable[str]] = ..., hunt_records: _Optional[_Iterable[_Union[LifecycleHuntReference, _Mapping]]] = ..., burden_score: _Optional[int] = ..., score_breakdown: _Optional[_Iterable[str]] = ..., confidence: _Optional[str] = ..., explicit_flows: _Optional[_Iterable[_Union[LifecycleFlowReference, _Mapping]]] = ..., explicitly_modeled: _Optional[bool] = ..., transition_owners: _Optional[_Iterable[_Union[LifecycleEvidence, _Mapping]]] = ..., persistence_class: _Optional[str] = ..., persistence_reason: _Optional[str] = ..., persistence_evidence: _Optional[_Iterable[_Union[LifecycleEvidence, _Mapping]]] = ..., persistence_scope: _Optional[str] = ..., persistence_files_scanned: _Optional[int] = ..., persistence_scan_complete: _Optional[bool] = ..., preselection_status: _Optional[str] = ...) -> None: ...

class GenerateScenarioArtifactsRequest(_message.Message):
    __slots__ = ("scenario_id",)
    SCENARIO_ID_FIELD_NUMBER: _ClassVar[int]
    scenario_id: str
    def __init__(self, scenario_id: _Optional[str] = ...) -> None: ...

class GenerateScenarioArtifactsResponse(_message.Message):
    __slots__ = ("flow_id", "report", "error_message")
    FLOW_ID_FIELD_NUMBER: _ClassVar[int]
    REPORT_FIELD_NUMBER: _ClassVar[int]
    ERROR_MESSAGE_FIELD_NUMBER: _ClassVar[int]
    flow_id: str
    report: _artifacts_pb2.ArtifactReport
    error_message: str
    def __init__(self, flow_id: _Optional[str] = ..., report: _Optional[_Union[_artifacts_pb2.ArtifactReport, _Mapping]] = ..., error_message: _Optional[str] = ...) -> None: ...

class ClearScenarioArtifactsRequest(_message.Message):
    __slots__ = ("scenario_id",)
    SCENARIO_ID_FIELD_NUMBER: _ClassVar[int]
    scenario_id: str
    def __init__(self, scenario_id: _Optional[str] = ...) -> None: ...

class ClearScenarioArtifactsResponse(_message.Message):
    __slots__ = ("flows",)
    FLOWS_FIELD_NUMBER: _ClassVar[int]
    flows: _containers.RepeatedCompositeFieldContainer[_artifacts_pb2.ClearArtifactsResponse]
    def __init__(self, flows: _Optional[_Iterable[_Union[_artifacts_pb2.ClearArtifactsResponse, _Mapping]]] = ...) -> None: ...
