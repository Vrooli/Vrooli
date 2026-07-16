from common.v1 import maturity_pb2 as _maturity_pb2
from google.protobuf.internal import containers as _containers
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Iterable as _Iterable, Mapping as _Mapping
from typing import ClassVar as _ClassVar, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class ValidateScenarioRequest(_message.Message):
    __slots__ = ("scenario", "path", "include_execution")
    SCENARIO_FIELD_NUMBER: _ClassVar[int]
    PATH_FIELD_NUMBER: _ClassVar[int]
    INCLUDE_EXECUTION_FIELD_NUMBER: _ClassVar[int]
    scenario: str
    path: str
    include_execution: bool
    def __init__(self, scenario: _Optional[str] = ..., path: _Optional[str] = ..., include_execution: _Optional[bool] = ...) -> None: ...

class ValidateScenarioResponse(_message.Message):
    __slots__ = ("scenario", "status", "summary", "target_path", "degraded_reason", "report", "assessment", "next_steps")
    SCENARIO_FIELD_NUMBER: _ClassVar[int]
    STATUS_FIELD_NUMBER: _ClassVar[int]
    SUMMARY_FIELD_NUMBER: _ClassVar[int]
    TARGET_PATH_FIELD_NUMBER: _ClassVar[int]
    DEGRADED_REASON_FIELD_NUMBER: _ClassVar[int]
    REPORT_FIELD_NUMBER: _ClassVar[int]
    ASSESSMENT_FIELD_NUMBER: _ClassVar[int]
    NEXT_STEPS_FIELD_NUMBER: _ClassVar[int]
    scenario: str
    status: str
    summary: str
    target_path: str
    degraded_reason: str
    report: ExperienceContractReport
    assessment: _maturity_pb2.MaturityAssessment
    next_steps: _containers.RepeatedScalarFieldContainer[str]
    def __init__(self, scenario: _Optional[str] = ..., status: _Optional[str] = ..., summary: _Optional[str] = ..., target_path: _Optional[str] = ..., degraded_reason: _Optional[str] = ..., report: _Optional[_Union[ExperienceContractReport, _Mapping]] = ..., assessment: _Optional[_Union[_maturity_pb2.MaturityAssessment, _Mapping]] = ..., next_steps: _Optional[_Iterable[str]] = ...) -> None: ...

class ListFleetRequest(_message.Message):
    __slots__ = ("repo_root",)
    REPO_ROOT_FIELD_NUMBER: _ClassVar[int]
    repo_root: str
    def __init__(self, repo_root: _Optional[str] = ...) -> None: ...

class ListFleetResponse(_message.Message):
    __slots__ = ("scenarios", "scenario_count", "with_experience_count", "total_pages")
    SCENARIOS_FIELD_NUMBER: _ClassVar[int]
    SCENARIO_COUNT_FIELD_NUMBER: _ClassVar[int]
    WITH_EXPERIENCE_COUNT_FIELD_NUMBER: _ClassVar[int]
    TOTAL_PAGES_FIELD_NUMBER: _ClassVar[int]
    scenarios: _containers.RepeatedCompositeFieldContainer[FleetScenario]
    scenario_count: int
    with_experience_count: int
    total_pages: int
    def __init__(self, scenarios: _Optional[_Iterable[_Union[FleetScenario, _Mapping]]] = ..., scenario_count: _Optional[int] = ..., with_experience_count: _Optional[int] = ..., total_pages: _Optional[int] = ...) -> None: ...

class FleetScenario(_message.Message):
    __slots__ = ("scenario", "has_experience", "max_depth", "max_depth_value", "page_count", "finding_count", "debt_score", "status")
    SCENARIO_FIELD_NUMBER: _ClassVar[int]
    HAS_EXPERIENCE_FIELD_NUMBER: _ClassVar[int]
    MAX_DEPTH_FIELD_NUMBER: _ClassVar[int]
    MAX_DEPTH_VALUE_FIELD_NUMBER: _ClassVar[int]
    PAGE_COUNT_FIELD_NUMBER: _ClassVar[int]
    FINDING_COUNT_FIELD_NUMBER: _ClassVar[int]
    DEBT_SCORE_FIELD_NUMBER: _ClassVar[int]
    STATUS_FIELD_NUMBER: _ClassVar[int]
    scenario: str
    has_experience: bool
    max_depth: str
    max_depth_value: int
    page_count: int
    finding_count: int
    debt_score: int
    status: str
    def __init__(self, scenario: _Optional[str] = ..., has_experience: _Optional[bool] = ..., max_depth: _Optional[str] = ..., max_depth_value: _Optional[int] = ..., page_count: _Optional[int] = ..., finding_count: _Optional[int] = ..., debt_score: _Optional[int] = ..., status: _Optional[str] = ...) -> None: ...

class AppendAttestationRequest(_message.Message):
    __slots__ = ("scenario", "page", "claim", "author", "rationale", "expires_at")
    SCENARIO_FIELD_NUMBER: _ClassVar[int]
    PAGE_FIELD_NUMBER: _ClassVar[int]
    CLAIM_FIELD_NUMBER: _ClassVar[int]
    AUTHOR_FIELD_NUMBER: _ClassVar[int]
    RATIONALE_FIELD_NUMBER: _ClassVar[int]
    EXPIRES_AT_FIELD_NUMBER: _ClassVar[int]
    scenario: str
    page: str
    claim: str
    author: str
    rationale: str
    expires_at: str
    def __init__(self, scenario: _Optional[str] = ..., page: _Optional[str] = ..., claim: _Optional[str] = ..., author: _Optional[str] = ..., rationale: _Optional[str] = ..., expires_at: _Optional[str] = ...) -> None: ...

class AppendAttestationResponse(_message.Message):
    __slots__ = ("attestation",)
    ATTESTATION_FIELD_NUMBER: _ClassVar[int]
    attestation: ManualAttestation
    def __init__(self, attestation: _Optional[_Union[ManualAttestation, _Mapping]] = ...) -> None: ...

class ScaffoldCasesRequest(_message.Message):
    __slots__ = ("scenario", "path", "dry_run")
    SCENARIO_FIELD_NUMBER: _ClassVar[int]
    PATH_FIELD_NUMBER: _ClassVar[int]
    DRY_RUN_FIELD_NUMBER: _ClassVar[int]
    scenario: str
    path: str
    dry_run: bool
    def __init__(self, scenario: _Optional[str] = ..., path: _Optional[str] = ..., dry_run: _Optional[bool] = ...) -> None: ...

class ScaffoldCasesResponse(_message.Message):
    __slots__ = ("scenario", "applied", "diffs", "messages")
    SCENARIO_FIELD_NUMBER: _ClassVar[int]
    APPLIED_FIELD_NUMBER: _ClassVar[int]
    DIFFS_FIELD_NUMBER: _ClassVar[int]
    MESSAGES_FIELD_NUMBER: _ClassVar[int]
    scenario: str
    applied: bool
    diffs: _containers.RepeatedCompositeFieldContainer[FileDiff]
    messages: _containers.RepeatedScalarFieldContainer[str]
    def __init__(self, scenario: _Optional[str] = ..., applied: _Optional[bool] = ..., diffs: _Optional[_Iterable[_Union[FileDiff, _Mapping]]] = ..., messages: _Optional[_Iterable[str]] = ...) -> None: ...

class ManualAttestation(_message.Message):
    __slots__ = ("id", "scenario", "page", "claim", "author", "rationale", "expires_at", "created_at")
    ID_FIELD_NUMBER: _ClassVar[int]
    SCENARIO_FIELD_NUMBER: _ClassVar[int]
    PAGE_FIELD_NUMBER: _ClassVar[int]
    CLAIM_FIELD_NUMBER: _ClassVar[int]
    AUTHOR_FIELD_NUMBER: _ClassVar[int]
    RATIONALE_FIELD_NUMBER: _ClassVar[int]
    EXPIRES_AT_FIELD_NUMBER: _ClassVar[int]
    CREATED_AT_FIELD_NUMBER: _ClassVar[int]
    id: str
    scenario: str
    page: str
    claim: str
    author: str
    rationale: str
    expires_at: str
    created_at: str
    def __init__(self, id: _Optional[str] = ..., scenario: _Optional[str] = ..., page: _Optional[str] = ..., claim: _Optional[str] = ..., author: _Optional[str] = ..., rationale: _Optional[str] = ..., expires_at: _Optional[str] = ..., created_at: _Optional[str] = ...) -> None: ...

class ExperienceContractReport(_message.Message):
    __slots__ = ("findings",)
    FINDINGS_FIELD_NUMBER: _ClassVar[int]
    findings: _containers.RepeatedCompositeFieldContainer[ExperienceFinding]
    def __init__(self, findings: _Optional[_Iterable[_Union[ExperienceFinding, _Mapping]]] = ...) -> None: ...

class ExperienceFinding(_message.Message):
    __slots__ = ("code", "severity", "title", "message", "location", "remediation", "autofix_available", "fix_class")
    CODE_FIELD_NUMBER: _ClassVar[int]
    SEVERITY_FIELD_NUMBER: _ClassVar[int]
    TITLE_FIELD_NUMBER: _ClassVar[int]
    MESSAGE_FIELD_NUMBER: _ClassVar[int]
    LOCATION_FIELD_NUMBER: _ClassVar[int]
    REMEDIATION_FIELD_NUMBER: _ClassVar[int]
    AUTOFIX_AVAILABLE_FIELD_NUMBER: _ClassVar[int]
    FIX_CLASS_FIELD_NUMBER: _ClassVar[int]
    code: str
    severity: str
    title: str
    message: str
    location: str
    remediation: str
    autofix_available: bool
    fix_class: str
    def __init__(self, code: _Optional[str] = ..., severity: _Optional[str] = ..., title: _Optional[str] = ..., message: _Optional[str] = ..., location: _Optional[str] = ..., remediation: _Optional[str] = ..., autofix_available: _Optional[bool] = ..., fix_class: _Optional[str] = ...) -> None: ...

class StartAuthoringSessionRequest(_message.Message):
    __slots__ = ("scenario", "path")
    SCENARIO_FIELD_NUMBER: _ClassVar[int]
    PATH_FIELD_NUMBER: _ClassVar[int]
    scenario: str
    path: str
    def __init__(self, scenario: _Optional[str] = ..., path: _Optional[str] = ...) -> None: ...

class StartAuthoringSessionResponse(_message.Message):
    __slots__ = ("session",)
    SESSION_FIELD_NUMBER: _ClassVar[int]
    session: AuthoringSession
    def __init__(self, session: _Optional[_Union[AuthoringSession, _Mapping]] = ...) -> None: ...

class SubmitPageRequest(_message.Message):
    __slots__ = ("session_id", "page")
    SESSION_ID_FIELD_NUMBER: _ClassVar[int]
    PAGE_FIELD_NUMBER: _ClassVar[int]
    session_id: str
    page: PageForm
    def __init__(self, session_id: _Optional[str] = ..., page: _Optional[_Union[PageForm, _Mapping]] = ...) -> None: ...

class SubmitPageResponse(_message.Message):
    __slots__ = ("session", "page")
    SESSION_FIELD_NUMBER: _ClassVar[int]
    PAGE_FIELD_NUMBER: _ClassVar[int]
    session: AuthoringSession
    page: PageDraft
    def __init__(self, session: _Optional[_Union[AuthoringSession, _Mapping]] = ..., page: _Optional[_Union[PageDraft, _Mapping]] = ...) -> None: ...

class PreviewSessionRequest(_message.Message):
    __slots__ = ("session_id",)
    SESSION_ID_FIELD_NUMBER: _ClassVar[int]
    session_id: str
    def __init__(self, session_id: _Optional[str] = ...) -> None: ...

class PreviewSessionResponse(_message.Message):
    __slots__ = ("session", "diffs", "validation")
    SESSION_FIELD_NUMBER: _ClassVar[int]
    DIFFS_FIELD_NUMBER: _ClassVar[int]
    VALIDATION_FIELD_NUMBER: _ClassVar[int]
    session: AuthoringSession
    diffs: _containers.RepeatedCompositeFieldContainer[FileDiff]
    validation: ValidateScenarioResponse
    def __init__(self, session: _Optional[_Union[AuthoringSession, _Mapping]] = ..., diffs: _Optional[_Iterable[_Union[FileDiff, _Mapping]]] = ..., validation: _Optional[_Union[ValidateScenarioResponse, _Mapping]] = ...) -> None: ...

class ApplySessionRequest(_message.Message):
    __slots__ = ("session_id",)
    SESSION_ID_FIELD_NUMBER: _ClassVar[int]
    session_id: str
    def __init__(self, session_id: _Optional[str] = ...) -> None: ...

class ApplySessionResponse(_message.Message):
    __slots__ = ("session", "diffs", "validation")
    SESSION_FIELD_NUMBER: _ClassVar[int]
    DIFFS_FIELD_NUMBER: _ClassVar[int]
    VALIDATION_FIELD_NUMBER: _ClassVar[int]
    session: AuthoringSession
    diffs: _containers.RepeatedCompositeFieldContainer[FileDiff]
    validation: ValidateScenarioResponse
    def __init__(self, session: _Optional[_Union[AuthoringSession, _Mapping]] = ..., diffs: _Optional[_Iterable[_Union[FileDiff, _Mapping]]] = ..., validation: _Optional[_Union[ValidateScenarioResponse, _Mapping]] = ...) -> None: ...

class DiscardSessionRequest(_message.Message):
    __slots__ = ("session_id",)
    SESSION_ID_FIELD_NUMBER: _ClassVar[int]
    session_id: str
    def __init__(self, session_id: _Optional[str] = ...) -> None: ...

class DiscardSessionResponse(_message.Message):
    __slots__ = ("session_id", "discarded")
    SESSION_ID_FIELD_NUMBER: _ClassVar[int]
    DISCARDED_FIELD_NUMBER: _ClassVar[int]
    session_id: str
    discarded: bool
    def __init__(self, session_id: _Optional[str] = ..., discarded: _Optional[bool] = ...) -> None: ...

class ListSpecRequest(_message.Message):
    __slots__ = ("scenario", "path")
    SCENARIO_FIELD_NUMBER: _ClassVar[int]
    PATH_FIELD_NUMBER: _ClassVar[int]
    scenario: str
    path: str
    def __init__(self, scenario: _Optional[str] = ..., path: _Optional[str] = ...) -> None: ...

class ListSpecResponse(_message.Message):
    __slots__ = ("scenario", "pages", "journeys", "components")
    SCENARIO_FIELD_NUMBER: _ClassVar[int]
    PAGES_FIELD_NUMBER: _ClassVar[int]
    JOURNEYS_FIELD_NUMBER: _ClassVar[int]
    COMPONENTS_FIELD_NUMBER: _ClassVar[int]
    scenario: str
    pages: _containers.RepeatedCompositeFieldContainer[SpecDocument]
    journeys: _containers.RepeatedCompositeFieldContainer[SpecDocument]
    components: _containers.RepeatedCompositeFieldContainer[SpecDocument]
    def __init__(self, scenario: _Optional[str] = ..., pages: _Optional[_Iterable[_Union[SpecDocument, _Mapping]]] = ..., journeys: _Optional[_Iterable[_Union[SpecDocument, _Mapping]]] = ..., components: _Optional[_Iterable[_Union[SpecDocument, _Mapping]]] = ...) -> None: ...

class ShowSpecRequest(_message.Message):
    __slots__ = ("scenario", "path", "page")
    SCENARIO_FIELD_NUMBER: _ClassVar[int]
    PATH_FIELD_NUMBER: _ClassVar[int]
    PAGE_FIELD_NUMBER: _ClassVar[int]
    scenario: str
    path: str
    page: str
    def __init__(self, scenario: _Optional[str] = ..., path: _Optional[str] = ..., page: _Optional[str] = ...) -> None: ...

class ShowSpecResponse(_message.Message):
    __slots__ = ("scenario", "page", "json")
    SCENARIO_FIELD_NUMBER: _ClassVar[int]
    PAGE_FIELD_NUMBER: _ClassVar[int]
    JSON_FIELD_NUMBER: _ClassVar[int]
    scenario: str
    page: str
    json: str
    def __init__(self, scenario: _Optional[str] = ..., page: _Optional[str] = ..., json: _Optional[str] = ...) -> None: ...

class ListEvidenceRequest(_message.Message):
    __slots__ = ("scenario", "path", "page", "claim", "limit", "component")
    SCENARIO_FIELD_NUMBER: _ClassVar[int]
    PATH_FIELD_NUMBER: _ClassVar[int]
    PAGE_FIELD_NUMBER: _ClassVar[int]
    CLAIM_FIELD_NUMBER: _ClassVar[int]
    LIMIT_FIELD_NUMBER: _ClassVar[int]
    COMPONENT_FIELD_NUMBER: _ClassVar[int]
    scenario: str
    path: str
    page: str
    claim: str
    limit: int
    component: str
    def __init__(self, scenario: _Optional[str] = ..., path: _Optional[str] = ..., page: _Optional[str] = ..., claim: _Optional[str] = ..., limit: _Optional[int] = ..., component: _Optional[str] = ...) -> None: ...

class ListEvidenceResponse(_message.Message):
    __slots__ = ("scenario", "page", "evidence")
    SCENARIO_FIELD_NUMBER: _ClassVar[int]
    PAGE_FIELD_NUMBER: _ClassVar[int]
    EVIDENCE_FIELD_NUMBER: _ClassVar[int]
    scenario: str
    page: str
    evidence: _containers.RepeatedCompositeFieldContainer[ReconciliationEvidence]
    def __init__(self, scenario: _Optional[str] = ..., page: _Optional[str] = ..., evidence: _Optional[_Iterable[_Union[ReconciliationEvidence, _Mapping]]] = ...) -> None: ...

class ReconciliationEvidence(_message.Message):
    __slots__ = ("id", "scenario", "page", "route", "state", "claim", "claim_type", "verdict", "capture_ref", "ax_node_json", "message", "checked_at", "viewport", "viewport_width", "viewport_height", "document_kind", "component_id", "component_title", "example_name")
    ID_FIELD_NUMBER: _ClassVar[int]
    SCENARIO_FIELD_NUMBER: _ClassVar[int]
    PAGE_FIELD_NUMBER: _ClassVar[int]
    ROUTE_FIELD_NUMBER: _ClassVar[int]
    STATE_FIELD_NUMBER: _ClassVar[int]
    CLAIM_FIELD_NUMBER: _ClassVar[int]
    CLAIM_TYPE_FIELD_NUMBER: _ClassVar[int]
    VERDICT_FIELD_NUMBER: _ClassVar[int]
    CAPTURE_REF_FIELD_NUMBER: _ClassVar[int]
    AX_NODE_JSON_FIELD_NUMBER: _ClassVar[int]
    MESSAGE_FIELD_NUMBER: _ClassVar[int]
    CHECKED_AT_FIELD_NUMBER: _ClassVar[int]
    VIEWPORT_FIELD_NUMBER: _ClassVar[int]
    VIEWPORT_WIDTH_FIELD_NUMBER: _ClassVar[int]
    VIEWPORT_HEIGHT_FIELD_NUMBER: _ClassVar[int]
    DOCUMENT_KIND_FIELD_NUMBER: _ClassVar[int]
    COMPONENT_ID_FIELD_NUMBER: _ClassVar[int]
    COMPONENT_TITLE_FIELD_NUMBER: _ClassVar[int]
    EXAMPLE_NAME_FIELD_NUMBER: _ClassVar[int]
    id: str
    scenario: str
    page: str
    route: str
    state: str
    claim: str
    claim_type: str
    verdict: str
    capture_ref: str
    ax_node_json: str
    message: str
    checked_at: str
    viewport: str
    viewport_width: int
    viewport_height: int
    document_kind: str
    component_id: str
    component_title: str
    example_name: str
    def __init__(self, id: _Optional[str] = ..., scenario: _Optional[str] = ..., page: _Optional[str] = ..., route: _Optional[str] = ..., state: _Optional[str] = ..., claim: _Optional[str] = ..., claim_type: _Optional[str] = ..., verdict: _Optional[str] = ..., capture_ref: _Optional[str] = ..., ax_node_json: _Optional[str] = ..., message: _Optional[str] = ..., checked_at: _Optional[str] = ..., viewport: _Optional[str] = ..., viewport_width: _Optional[int] = ..., viewport_height: _Optional[int] = ..., document_kind: _Optional[str] = ..., component_id: _Optional[str] = ..., component_title: _Optional[str] = ..., example_name: _Optional[str] = ...) -> None: ...

class SuggestBindingsRequest(_message.Message):
    __slots__ = ("scenario", "path", "page", "limit")
    SCENARIO_FIELD_NUMBER: _ClassVar[int]
    PATH_FIELD_NUMBER: _ClassVar[int]
    PAGE_FIELD_NUMBER: _ClassVar[int]
    LIMIT_FIELD_NUMBER: _ClassVar[int]
    scenario: str
    path: str
    page: str
    limit: int
    def __init__(self, scenario: _Optional[str] = ..., path: _Optional[str] = ..., page: _Optional[str] = ..., limit: _Optional[int] = ...) -> None: ...

class SuggestBindingsResponse(_message.Message):
    __slots__ = ("scenario", "page", "suggestions")
    SCENARIO_FIELD_NUMBER: _ClassVar[int]
    PAGE_FIELD_NUMBER: _ClassVar[int]
    SUGGESTIONS_FIELD_NUMBER: _ClassVar[int]
    scenario: str
    page: str
    suggestions: _containers.RepeatedCompositeFieldContainer[BindingSuggestion]
    def __init__(self, scenario: _Optional[str] = ..., page: _Optional[str] = ..., suggestions: _Optional[_Iterable[_Union[BindingSuggestion, _Mapping]]] = ...) -> None: ...

class RenderSpecRequest(_message.Message):
    __slots__ = ("scenario", "path", "page", "mode")
    SCENARIO_FIELD_NUMBER: _ClassVar[int]
    PATH_FIELD_NUMBER: _ClassVar[int]
    PAGE_FIELD_NUMBER: _ClassVar[int]
    MODE_FIELD_NUMBER: _ClassVar[int]
    scenario: str
    path: str
    page: str
    mode: str
    def __init__(self, scenario: _Optional[str] = ..., path: _Optional[str] = ..., page: _Optional[str] = ..., mode: _Optional[str] = ...) -> None: ...

class RenderSpecResponse(_message.Message):
    __slots__ = ("scenario", "page", "mode", "html", "artifact_path", "degraded_reason")
    SCENARIO_FIELD_NUMBER: _ClassVar[int]
    PAGE_FIELD_NUMBER: _ClassVar[int]
    MODE_FIELD_NUMBER: _ClassVar[int]
    HTML_FIELD_NUMBER: _ClassVar[int]
    ARTIFACT_PATH_FIELD_NUMBER: _ClassVar[int]
    DEGRADED_REASON_FIELD_NUMBER: _ClassVar[int]
    scenario: str
    page: str
    mode: str
    html: str
    artifact_path: str
    degraded_reason: str
    def __init__(self, scenario: _Optional[str] = ..., page: _Optional[str] = ..., mode: _Optional[str] = ..., html: _Optional[str] = ..., artifact_path: _Optional[str] = ..., degraded_reason: _Optional[str] = ...) -> None: ...

class CompareVariantsRequest(_message.Message):
    __slots__ = ("scenario", "path", "page", "mode", "variants")
    SCENARIO_FIELD_NUMBER: _ClassVar[int]
    PATH_FIELD_NUMBER: _ClassVar[int]
    PAGE_FIELD_NUMBER: _ClassVar[int]
    MODE_FIELD_NUMBER: _ClassVar[int]
    VARIANTS_FIELD_NUMBER: _ClassVar[int]
    scenario: str
    path: str
    page: str
    mode: str
    variants: _containers.RepeatedCompositeFieldContainer[SpecVariant]
    def __init__(self, scenario: _Optional[str] = ..., path: _Optional[str] = ..., page: _Optional[str] = ..., mode: _Optional[str] = ..., variants: _Optional[_Iterable[_Union[SpecVariant, _Mapping]]] = ...) -> None: ...

class CompareVariantsResponse(_message.Message):
    __slots__ = ("scenario", "page", "mode", "html", "artifact_path", "degraded_reason", "variants")
    SCENARIO_FIELD_NUMBER: _ClassVar[int]
    PAGE_FIELD_NUMBER: _ClassVar[int]
    MODE_FIELD_NUMBER: _ClassVar[int]
    HTML_FIELD_NUMBER: _ClassVar[int]
    ARTIFACT_PATH_FIELD_NUMBER: _ClassVar[int]
    DEGRADED_REASON_FIELD_NUMBER: _ClassVar[int]
    VARIANTS_FIELD_NUMBER: _ClassVar[int]
    scenario: str
    page: str
    mode: str
    html: str
    artifact_path: str
    degraded_reason: str
    variants: _containers.RepeatedCompositeFieldContainer[RenderedVariant]
    def __init__(self, scenario: _Optional[str] = ..., page: _Optional[str] = ..., mode: _Optional[str] = ..., html: _Optional[str] = ..., artifact_path: _Optional[str] = ..., degraded_reason: _Optional[str] = ..., variants: _Optional[_Iterable[_Union[RenderedVariant, _Mapping]]] = ...) -> None: ...

class PromoteVariantRequest(_message.Message):
    __slots__ = ("scenario", "path", "page", "variant")
    SCENARIO_FIELD_NUMBER: _ClassVar[int]
    PATH_FIELD_NUMBER: _ClassVar[int]
    PAGE_FIELD_NUMBER: _ClassVar[int]
    VARIANT_FIELD_NUMBER: _ClassVar[int]
    scenario: str
    path: str
    page: str
    variant: SpecVariant
    def __init__(self, scenario: _Optional[str] = ..., path: _Optional[str] = ..., page: _Optional[str] = ..., variant: _Optional[_Union[SpecVariant, _Mapping]] = ...) -> None: ...

class PromoteVariantResponse(_message.Message):
    __slots__ = ("scenario", "page", "variant", "diffs", "validation")
    SCENARIO_FIELD_NUMBER: _ClassVar[int]
    PAGE_FIELD_NUMBER: _ClassVar[int]
    VARIANT_FIELD_NUMBER: _ClassVar[int]
    DIFFS_FIELD_NUMBER: _ClassVar[int]
    VALIDATION_FIELD_NUMBER: _ClassVar[int]
    scenario: str
    page: str
    variant: RenderedVariant
    diffs: _containers.RepeatedCompositeFieldContainer[FileDiff]
    validation: ValidateScenarioResponse
    def __init__(self, scenario: _Optional[str] = ..., page: _Optional[str] = ..., variant: _Optional[_Union[RenderedVariant, _Mapping]] = ..., diffs: _Optional[_Iterable[_Union[FileDiff, _Mapping]]] = ..., validation: _Optional[_Union[ValidateScenarioResponse, _Mapping]] = ...) -> None: ...

class SpecVariant(_message.Message):
    __slots__ = ("id", "title", "page")
    ID_FIELD_NUMBER: _ClassVar[int]
    TITLE_FIELD_NUMBER: _ClassVar[int]
    PAGE_FIELD_NUMBER: _ClassVar[int]
    id: str
    title: str
    page: PageForm
    def __init__(self, id: _Optional[str] = ..., title: _Optional[str] = ..., page: _Optional[_Union[PageForm, _Mapping]] = ...) -> None: ...

class RenderedVariant(_message.Message):
    __slots__ = ("id", "title", "html")
    ID_FIELD_NUMBER: _ClassVar[int]
    TITLE_FIELD_NUMBER: _ClassVar[int]
    HTML_FIELD_NUMBER: _ClassVar[int]
    id: str
    title: str
    html: str
    def __init__(self, id: _Optional[str] = ..., title: _Optional[str] = ..., html: _Optional[str] = ...) -> None: ...

class AuthoringSession(_message.Message):
    __slots__ = ("id", "scenario", "target_path", "status", "page_count", "created_at", "updated_at")
    ID_FIELD_NUMBER: _ClassVar[int]
    SCENARIO_FIELD_NUMBER: _ClassVar[int]
    TARGET_PATH_FIELD_NUMBER: _ClassVar[int]
    STATUS_FIELD_NUMBER: _ClassVar[int]
    PAGE_COUNT_FIELD_NUMBER: _ClassVar[int]
    CREATED_AT_FIELD_NUMBER: _ClassVar[int]
    UPDATED_AT_FIELD_NUMBER: _ClassVar[int]
    id: str
    scenario: str
    target_path: str
    status: str
    page_count: int
    created_at: str
    updated_at: str
    def __init__(self, id: _Optional[str] = ..., scenario: _Optional[str] = ..., target_path: _Optional[str] = ..., status: _Optional[str] = ..., page_count: _Optional[int] = ..., created_at: _Optional[str] = ..., updated_at: _Optional[str] = ...) -> None: ...

class PageDraft(_message.Message):
    __slots__ = ("id", "path", "title", "status")
    ID_FIELD_NUMBER: _ClassVar[int]
    PATH_FIELD_NUMBER: _ClassVar[int]
    TITLE_FIELD_NUMBER: _ClassVar[int]
    STATUS_FIELD_NUMBER: _ClassVar[int]
    id: str
    path: str
    title: str
    status: str
    def __init__(self, id: _Optional[str] = ..., path: _Optional[str] = ..., title: _Optional[str] = ..., status: _Optional[str] = ...) -> None: ...

class FileDiff(_message.Message):
    __slots__ = ("path", "action", "before", "after")
    PATH_FIELD_NUMBER: _ClassVar[int]
    ACTION_FIELD_NUMBER: _ClassVar[int]
    BEFORE_FIELD_NUMBER: _ClassVar[int]
    AFTER_FIELD_NUMBER: _ClassVar[int]
    path: str
    action: str
    before: str
    after: str
    def __init__(self, path: _Optional[str] = ..., action: _Optional[str] = ..., before: _Optional[str] = ..., after: _Optional[str] = ...) -> None: ...

class SpecDocument(_message.Message):
    __slots__ = ("id", "path", "title", "status")
    ID_FIELD_NUMBER: _ClassVar[int]
    PATH_FIELD_NUMBER: _ClassVar[int]
    TITLE_FIELD_NUMBER: _ClassVar[int]
    STATUS_FIELD_NUMBER: _ClassVar[int]
    id: str
    path: str
    title: str
    status: str
    def __init__(self, id: _Optional[str] = ..., path: _Optional[str] = ..., title: _Optional[str] = ..., status: _Optional[str] = ...) -> None: ...

class BindingSuggestion(_message.Message):
    __slots__ = ("element_id", "testid", "role", "accessible_name", "source")
    ELEMENT_ID_FIELD_NUMBER: _ClassVar[int]
    TESTID_FIELD_NUMBER: _ClassVar[int]
    ROLE_FIELD_NUMBER: _ClassVar[int]
    ACCESSIBLE_NAME_FIELD_NUMBER: _ClassVar[int]
    SOURCE_FIELD_NUMBER: _ClassVar[int]
    element_id: str
    testid: str
    role: str
    accessible_name: str
    source: str
    def __init__(self, element_id: _Optional[str] = ..., testid: _Optional[str] = ..., role: _Optional[str] = ..., accessible_name: _Optional[str] = ..., source: _Optional[str] = ...) -> None: ...

class PageForm(_message.Message):
    __slots__ = ("id", "title", "purpose", "routes", "prd_refs", "status", "priorities", "states", "elements", "claims", "bindings", "sketch_regions")
    ID_FIELD_NUMBER: _ClassVar[int]
    TITLE_FIELD_NUMBER: _ClassVar[int]
    PURPOSE_FIELD_NUMBER: _ClassVar[int]
    ROUTES_FIELD_NUMBER: _ClassVar[int]
    PRD_REFS_FIELD_NUMBER: _ClassVar[int]
    STATUS_FIELD_NUMBER: _ClassVar[int]
    PRIORITIES_FIELD_NUMBER: _ClassVar[int]
    STATES_FIELD_NUMBER: _ClassVar[int]
    ELEMENTS_FIELD_NUMBER: _ClassVar[int]
    CLAIMS_FIELD_NUMBER: _ClassVar[int]
    BINDINGS_FIELD_NUMBER: _ClassVar[int]
    SKETCH_REGIONS_FIELD_NUMBER: _ClassVar[int]
    id: str
    title: str
    purpose: str
    routes: _containers.RepeatedScalarFieldContainer[str]
    prd_refs: _containers.RepeatedScalarFieldContainer[str]
    status: str
    priorities: _containers.RepeatedCompositeFieldContainer[PriorityForm]
    states: _containers.RepeatedCompositeFieldContainer[StateForm]
    elements: _containers.RepeatedCompositeFieldContainer[ElementForm]
    claims: _containers.RepeatedCompositeFieldContainer[ClaimForm]
    bindings: _containers.RepeatedCompositeFieldContainer[BindingForm]
    sketch_regions: _containers.RepeatedCompositeFieldContainer[SketchRegionForm]
    def __init__(self, id: _Optional[str] = ..., title: _Optional[str] = ..., purpose: _Optional[str] = ..., routes: _Optional[_Iterable[str]] = ..., prd_refs: _Optional[_Iterable[str]] = ..., status: _Optional[str] = ..., priorities: _Optional[_Iterable[_Union[PriorityForm, _Mapping]]] = ..., states: _Optional[_Iterable[_Union[StateForm, _Mapping]]] = ..., elements: _Optional[_Iterable[_Union[ElementForm, _Mapping]]] = ..., claims: _Optional[_Iterable[_Union[ClaimForm, _Mapping]]] = ..., bindings: _Optional[_Iterable[_Union[BindingForm, _Mapping]]] = ..., sketch_regions: _Optional[_Iterable[_Union[SketchRegionForm, _Mapping]]] = ...) -> None: ...

class PriorityForm(_message.Message):
    __slots__ = ("statement", "notes")
    STATEMENT_FIELD_NUMBER: _ClassVar[int]
    NOTES_FIELD_NUMBER: _ClassVar[int]
    statement: str
    notes: str
    def __init__(self, statement: _Optional[str] = ..., notes: _Optional[str] = ...) -> None: ...

class StateForm(_message.Message):
    __slots__ = ("id", "description")
    ID_FIELD_NUMBER: _ClassVar[int]
    DESCRIPTION_FIELD_NUMBER: _ClassVar[int]
    id: str
    description: str
    def __init__(self, id: _Optional[str] = ..., description: _Optional[str] = ...) -> None: ...

class ElementForm(_message.Message):
    __slots__ = ("id", "role", "name", "description")
    ID_FIELD_NUMBER: _ClassVar[int]
    ROLE_FIELD_NUMBER: _ClassVar[int]
    NAME_FIELD_NUMBER: _ClassVar[int]
    DESCRIPTION_FIELD_NUMBER: _ClassVar[int]
    id: str
    role: str
    name: str
    description: str
    def __init__(self, id: _Optional[str] = ..., role: _Optional[str] = ..., name: _Optional[str] = ..., description: _Optional[str] = ...) -> None: ...

class ClaimForm(_message.Message):
    __slots__ = ("id", "type", "statement", "tier", "elements", "states", "viewports", "locales", "rationale")
    ID_FIELD_NUMBER: _ClassVar[int]
    TYPE_FIELD_NUMBER: _ClassVar[int]
    STATEMENT_FIELD_NUMBER: _ClassVar[int]
    TIER_FIELD_NUMBER: _ClassVar[int]
    ELEMENTS_FIELD_NUMBER: _ClassVar[int]
    STATES_FIELD_NUMBER: _ClassVar[int]
    VIEWPORTS_FIELD_NUMBER: _ClassVar[int]
    LOCALES_FIELD_NUMBER: _ClassVar[int]
    RATIONALE_FIELD_NUMBER: _ClassVar[int]
    id: str
    type: str
    statement: str
    tier: str
    elements: _containers.RepeatedScalarFieldContainer[str]
    states: _containers.RepeatedScalarFieldContainer[str]
    viewports: _containers.RepeatedScalarFieldContainer[str]
    locales: _containers.RepeatedScalarFieldContainer[str]
    rationale: str
    def __init__(self, id: _Optional[str] = ..., type: _Optional[str] = ..., statement: _Optional[str] = ..., tier: _Optional[str] = ..., elements: _Optional[_Iterable[str]] = ..., states: _Optional[_Iterable[str]] = ..., viewports: _Optional[_Iterable[str]] = ..., locales: _Optional[_Iterable[str]] = ..., rationale: _Optional[str] = ...) -> None: ...

class BindingForm(_message.Message):
    __slots__ = ("element_id", "testid", "selector", "note")
    ELEMENT_ID_FIELD_NUMBER: _ClassVar[int]
    TESTID_FIELD_NUMBER: _ClassVar[int]
    SELECTOR_FIELD_NUMBER: _ClassVar[int]
    NOTE_FIELD_NUMBER: _ClassVar[int]
    element_id: str
    testid: str
    selector: str
    note: str
    def __init__(self, element_id: _Optional[str] = ..., testid: _Optional[str] = ..., selector: _Optional[str] = ..., note: _Optional[str] = ...) -> None: ...

class SketchRegionForm(_message.Message):
    __slots__ = ("id", "elements")
    ID_FIELD_NUMBER: _ClassVar[int]
    ELEMENTS_FIELD_NUMBER: _ClassVar[int]
    id: str
    elements: _containers.RepeatedScalarFieldContainer[str]
    def __init__(self, id: _Optional[str] = ..., elements: _Optional[_Iterable[str]] = ...) -> None: ...
