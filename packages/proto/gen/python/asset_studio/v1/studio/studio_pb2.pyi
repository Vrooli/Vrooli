from google.protobuf.internal import containers as _containers
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Iterable as _Iterable, Mapping as _Mapping
from typing import ClassVar as _ClassVar, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class ConditioningReference(_message.Message):
    __slots__ = ("kind", "id", "version")
    KIND_FIELD_NUMBER: _ClassVar[int]
    ID_FIELD_NUMBER: _ClassVar[int]
    VERSION_FIELD_NUMBER: _ClassVar[int]
    kind: str
    id: str
    version: str
    def __init__(self, kind: _Optional[str] = ..., id: _Optional[str] = ..., version: _Optional[str] = ...) -> None: ...

class Identity(_message.Message):
    __slots__ = ("id", "name", "kind", "version", "traits", "reference_images", "conditioning_references", "credential_claims", "referenced")
    class TraitsEntry(_message.Message):
        __slots__ = ("key", "value")
        KEY_FIELD_NUMBER: _ClassVar[int]
        VALUE_FIELD_NUMBER: _ClassVar[int]
        key: str
        value: str
        def __init__(self, key: _Optional[str] = ..., value: _Optional[str] = ...) -> None: ...
    ID_FIELD_NUMBER: _ClassVar[int]
    NAME_FIELD_NUMBER: _ClassVar[int]
    KIND_FIELD_NUMBER: _ClassVar[int]
    VERSION_FIELD_NUMBER: _ClassVar[int]
    TRAITS_FIELD_NUMBER: _ClassVar[int]
    REFERENCE_IMAGES_FIELD_NUMBER: _ClassVar[int]
    CONDITIONING_REFERENCES_FIELD_NUMBER: _ClassVar[int]
    CREDENTIAL_CLAIMS_FIELD_NUMBER: _ClassVar[int]
    REFERENCED_FIELD_NUMBER: _ClassVar[int]
    id: str
    name: str
    kind: str
    version: int
    traits: _containers.ScalarMap[str, str]
    reference_images: _containers.RepeatedScalarFieldContainer[str]
    conditioning_references: _containers.RepeatedCompositeFieldContainer[ConditioningReference]
    credential_claims: str
    referenced: bool
    def __init__(self, id: _Optional[str] = ..., name: _Optional[str] = ..., kind: _Optional[str] = ..., version: _Optional[int] = ..., traits: _Optional[_Mapping[str, str]] = ..., reference_images: _Optional[_Iterable[str]] = ..., conditioning_references: _Optional[_Iterable[_Union[ConditioningReference, _Mapping]]] = ..., credential_claims: _Optional[str] = ..., referenced: _Optional[bool] = ...) -> None: ...

class AssetReference(_message.Message):
    __slots__ = ("id", "status", "alt_text", "disclosure", "ai_generated", "width", "height", "media_type", "parent_asset_id", "derivation_operation")
    ID_FIELD_NUMBER: _ClassVar[int]
    STATUS_FIELD_NUMBER: _ClassVar[int]
    ALT_TEXT_FIELD_NUMBER: _ClassVar[int]
    DISCLOSURE_FIELD_NUMBER: _ClassVar[int]
    AI_GENERATED_FIELD_NUMBER: _ClassVar[int]
    WIDTH_FIELD_NUMBER: _ClassVar[int]
    HEIGHT_FIELD_NUMBER: _ClassVar[int]
    MEDIA_TYPE_FIELD_NUMBER: _ClassVar[int]
    PARENT_ASSET_ID_FIELD_NUMBER: _ClassVar[int]
    DERIVATION_OPERATION_FIELD_NUMBER: _ClassVar[int]
    id: str
    status: str
    alt_text: str
    disclosure: str
    ai_generated: bool
    width: int
    height: int
    media_type: str
    parent_asset_id: str
    derivation_operation: str
    def __init__(self, id: _Optional[str] = ..., status: _Optional[str] = ..., alt_text: _Optional[str] = ..., disclosure: _Optional[str] = ..., ai_generated: _Optional[bool] = ..., width: _Optional[int] = ..., height: _Optional[int] = ..., media_type: _Optional[str] = ..., parent_asset_id: _Optional[str] = ..., derivation_operation: _Optional[str] = ...) -> None: ...

class ListIdentitiesRequest(_message.Message):
    __slots__ = ("kind",)
    KIND_FIELD_NUMBER: _ClassVar[int]
    kind: str
    def __init__(self, kind: _Optional[str] = ...) -> None: ...

class ListIdentitiesResponse(_message.Message):
    __slots__ = ("identities",)
    IDENTITIES_FIELD_NUMBER: _ClassVar[int]
    identities: _containers.RepeatedCompositeFieldContainer[Identity]
    def __init__(self, identities: _Optional[_Iterable[_Union[Identity, _Mapping]]] = ...) -> None: ...

class CreateIdentityRequest(_message.Message):
    __slots__ = ("identity", "actor_id", "actor_kind")
    IDENTITY_FIELD_NUMBER: _ClassVar[int]
    ACTOR_ID_FIELD_NUMBER: _ClassVar[int]
    ACTOR_KIND_FIELD_NUMBER: _ClassVar[int]
    identity: Identity
    actor_id: str
    actor_kind: str
    def __init__(self, identity: _Optional[_Union[Identity, _Mapping]] = ..., actor_id: _Optional[str] = ..., actor_kind: _Optional[str] = ...) -> None: ...

class CreateIdentityResponse(_message.Message):
    __slots__ = ("identity",)
    IDENTITY_FIELD_NUMBER: _ClassVar[int]
    identity: Identity
    def __init__(self, identity: _Optional[_Union[Identity, _Mapping]] = ...) -> None: ...

class ReviseIdentityRequest(_message.Message):
    __slots__ = ("identity", "actor_id", "actor_kind")
    IDENTITY_FIELD_NUMBER: _ClassVar[int]
    ACTOR_ID_FIELD_NUMBER: _ClassVar[int]
    ACTOR_KIND_FIELD_NUMBER: _ClassVar[int]
    identity: Identity
    actor_id: str
    actor_kind: str
    def __init__(self, identity: _Optional[_Union[Identity, _Mapping]] = ..., actor_id: _Optional[str] = ..., actor_kind: _Optional[str] = ...) -> None: ...

class ReviseIdentityResponse(_message.Message):
    __slots__ = ("identity",)
    IDENTITY_FIELD_NUMBER: _ClassVar[int]
    identity: Identity
    def __init__(self, identity: _Optional[_Union[Identity, _Mapping]] = ...) -> None: ...

class ResolveSpecRequest(_message.Message):
    __slots__ = ("template", "fields", "identity_version_ids", "campaign_ref")
    class FieldsEntry(_message.Message):
        __slots__ = ("key", "value")
        KEY_FIELD_NUMBER: _ClassVar[int]
        VALUE_FIELD_NUMBER: _ClassVar[int]
        key: str
        value: str
        def __init__(self, key: _Optional[str] = ..., value: _Optional[str] = ...) -> None: ...
    TEMPLATE_FIELD_NUMBER: _ClassVar[int]
    FIELDS_FIELD_NUMBER: _ClassVar[int]
    IDENTITY_VERSION_IDS_FIELD_NUMBER: _ClassVar[int]
    CAMPAIGN_REF_FIELD_NUMBER: _ClassVar[int]
    template: str
    fields: _containers.ScalarMap[str, str]
    identity_version_ids: _containers.RepeatedScalarFieldContainer[str]
    campaign_ref: str
    def __init__(self, template: _Optional[str] = ..., fields: _Optional[_Mapping[str, str]] = ..., identity_version_ids: _Optional[_Iterable[str]] = ..., campaign_ref: _Optional[str] = ...) -> None: ...

class ResolveSpecResponse(_message.Message):
    __slots__ = ("spec_id", "resolved_payload")
    SPEC_ID_FIELD_NUMBER: _ClassVar[int]
    RESOLVED_PAYLOAD_FIELD_NUMBER: _ClassVar[int]
    spec_id: str
    resolved_payload: str
    def __init__(self, spec_id: _Optional[str] = ..., resolved_payload: _Optional[str] = ...) -> None: ...

class CreateRenderRequest(_message.Message):
    __slots__ = ("spec_id", "estimated_cost", "candidate_count", "producer_kind", "frame_count", "parent_asset_id", "capture_url", "confirm_over_budget", "budget_confirmation_actor_id", "composition_slots")
    SPEC_ID_FIELD_NUMBER: _ClassVar[int]
    ESTIMATED_COST_FIELD_NUMBER: _ClassVar[int]
    CANDIDATE_COUNT_FIELD_NUMBER: _ClassVar[int]
    PRODUCER_KIND_FIELD_NUMBER: _ClassVar[int]
    FRAME_COUNT_FIELD_NUMBER: _ClassVar[int]
    PARENT_ASSET_ID_FIELD_NUMBER: _ClassVar[int]
    CAPTURE_URL_FIELD_NUMBER: _ClassVar[int]
    CONFIRM_OVER_BUDGET_FIELD_NUMBER: _ClassVar[int]
    BUDGET_CONFIRMATION_ACTOR_ID_FIELD_NUMBER: _ClassVar[int]
    COMPOSITION_SLOTS_FIELD_NUMBER: _ClassVar[int]
    spec_id: str
    estimated_cost: float
    candidate_count: int
    producer_kind: str
    frame_count: int
    parent_asset_id: str
    capture_url: str
    confirm_over_budget: bool
    budget_confirmation_actor_id: str
    composition_slots: _containers.RepeatedCompositeFieldContainer[CompositionSlot]
    def __init__(self, spec_id: _Optional[str] = ..., estimated_cost: _Optional[float] = ..., candidate_count: _Optional[int] = ..., producer_kind: _Optional[str] = ..., frame_count: _Optional[int] = ..., parent_asset_id: _Optional[str] = ..., capture_url: _Optional[str] = ..., confirm_over_budget: _Optional[bool] = ..., budget_confirmation_actor_id: _Optional[str] = ..., composition_slots: _Optional[_Iterable[_Union[CompositionSlot, _Mapping]]] = ...) -> None: ...

class CompositionSlot(_message.Message):
    __slots__ = ("name", "asset_id", "order")
    NAME_FIELD_NUMBER: _ClassVar[int]
    ASSET_ID_FIELD_NUMBER: _ClassVar[int]
    ORDER_FIELD_NUMBER: _ClassVar[int]
    name: str
    asset_id: str
    order: int
    def __init__(self, name: _Optional[str] = ..., asset_id: _Optional[str] = ..., order: _Optional[int] = ...) -> None: ...

class CreateRenderResponse(_message.Message):
    __slots__ = ("render_id", "status", "candidates")
    RENDER_ID_FIELD_NUMBER: _ClassVar[int]
    STATUS_FIELD_NUMBER: _ClassVar[int]
    CANDIDATES_FIELD_NUMBER: _ClassVar[int]
    render_id: str
    status: str
    candidates: _containers.RepeatedCompositeFieldContainer[AssetReference]
    def __init__(self, render_id: _Optional[str] = ..., status: _Optional[str] = ..., candidates: _Optional[_Iterable[_Union[AssetReference, _Mapping]]] = ...) -> None: ...

class RegenerateRenderRequest(_message.Message):
    __slots__ = ("source_render_id", "confirm_over_budget", "budget_confirmation_actor_id")
    SOURCE_RENDER_ID_FIELD_NUMBER: _ClassVar[int]
    CONFIRM_OVER_BUDGET_FIELD_NUMBER: _ClassVar[int]
    BUDGET_CONFIRMATION_ACTOR_ID_FIELD_NUMBER: _ClassVar[int]
    source_render_id: str
    confirm_over_budget: bool
    budget_confirmation_actor_id: str
    def __init__(self, source_render_id: _Optional[str] = ..., confirm_over_budget: _Optional[bool] = ..., budget_confirmation_actor_id: _Optional[str] = ...) -> None: ...

class RegenerateRenderResponse(_message.Message):
    __slots__ = ("render_id", "status", "source_render_id")
    RENDER_ID_FIELD_NUMBER: _ClassVar[int]
    STATUS_FIELD_NUMBER: _ClassVar[int]
    SOURCE_RENDER_ID_FIELD_NUMBER: _ClassVar[int]
    render_id: str
    status: str
    source_render_id: str
    def __init__(self, render_id: _Optional[str] = ..., status: _Optional[str] = ..., source_render_id: _Optional[str] = ...) -> None: ...

class AdvisoryConformance(_message.Message):
    __slots__ = ("asset_id", "source", "score", "notes", "recorded_at")
    ASSET_ID_FIELD_NUMBER: _ClassVar[int]
    SOURCE_FIELD_NUMBER: _ClassVar[int]
    SCORE_FIELD_NUMBER: _ClassVar[int]
    NOTES_FIELD_NUMBER: _ClassVar[int]
    RECORDED_AT_FIELD_NUMBER: _ClassVar[int]
    asset_id: str
    source: str
    score: float
    notes: _containers.RepeatedScalarFieldContainer[str]
    recorded_at: str
    def __init__(self, asset_id: _Optional[str] = ..., source: _Optional[str] = ..., score: _Optional[float] = ..., notes: _Optional[_Iterable[str]] = ..., recorded_at: _Optional[str] = ...) -> None: ...

class AnalyzeConformanceRequest(_message.Message):
    __slots__ = ("asset_id",)
    ASSET_ID_FIELD_NUMBER: _ClassVar[int]
    asset_id: str
    def __init__(self, asset_id: _Optional[str] = ...) -> None: ...

class AnalyzeConformanceResponse(_message.Message):
    __slots__ = ("advisory",)
    ADVISORY_FIELD_NUMBER: _ClassVar[int]
    advisory: AdvisoryConformance
    def __init__(self, advisory: _Optional[_Union[AdvisoryConformance, _Mapping]] = ...) -> None: ...

class AgentCommission(_message.Message):
    __slots__ = ("id", "agent_task_id", "agent_identity", "request", "source_identity_version_ids", "status", "created_at")
    ID_FIELD_NUMBER: _ClassVar[int]
    AGENT_TASK_ID_FIELD_NUMBER: _ClassVar[int]
    AGENT_IDENTITY_FIELD_NUMBER: _ClassVar[int]
    REQUEST_FIELD_NUMBER: _ClassVar[int]
    SOURCE_IDENTITY_VERSION_IDS_FIELD_NUMBER: _ClassVar[int]
    STATUS_FIELD_NUMBER: _ClassVar[int]
    CREATED_AT_FIELD_NUMBER: _ClassVar[int]
    id: str
    agent_task_id: str
    agent_identity: str
    request: str
    source_identity_version_ids: _containers.RepeatedScalarFieldContainer[str]
    status: str
    created_at: str
    def __init__(self, id: _Optional[str] = ..., agent_task_id: _Optional[str] = ..., agent_identity: _Optional[str] = ..., request: _Optional[str] = ..., source_identity_version_ids: _Optional[_Iterable[str]] = ..., status: _Optional[str] = ..., created_at: _Optional[str] = ...) -> None: ...

class CommissionAgentRequest(_message.Message):
    __slots__ = ("request", "source_identity_version_ids", "agent_identity")
    REQUEST_FIELD_NUMBER: _ClassVar[int]
    SOURCE_IDENTITY_VERSION_IDS_FIELD_NUMBER: _ClassVar[int]
    AGENT_IDENTITY_FIELD_NUMBER: _ClassVar[int]
    request: str
    source_identity_version_ids: _containers.RepeatedScalarFieldContainer[str]
    agent_identity: str
    def __init__(self, request: _Optional[str] = ..., source_identity_version_ids: _Optional[_Iterable[str]] = ..., agent_identity: _Optional[str] = ...) -> None: ...

class CommissionAgentResponse(_message.Message):
    __slots__ = ("commission",)
    COMMISSION_FIELD_NUMBER: _ClassVar[int]
    commission: AgentCommission
    def __init__(self, commission: _Optional[_Union[AgentCommission, _Mapping]] = ...) -> None: ...

class CampaignBudget(_message.Message):
    __slots__ = ("campaign_ref", "limit_usd", "spent_usd")
    CAMPAIGN_REF_FIELD_NUMBER: _ClassVar[int]
    LIMIT_USD_FIELD_NUMBER: _ClassVar[int]
    SPENT_USD_FIELD_NUMBER: _ClassVar[int]
    campaign_ref: str
    limit_usd: float
    spent_usd: float
    def __init__(self, campaign_ref: _Optional[str] = ..., limit_usd: _Optional[float] = ..., spent_usd: _Optional[float] = ...) -> None: ...

class SetCampaignBudgetRequest(_message.Message):
    __slots__ = ("campaign_ref", "limit_usd")
    CAMPAIGN_REF_FIELD_NUMBER: _ClassVar[int]
    LIMIT_USD_FIELD_NUMBER: _ClassVar[int]
    campaign_ref: str
    limit_usd: float
    def __init__(self, campaign_ref: _Optional[str] = ..., limit_usd: _Optional[float] = ...) -> None: ...

class SetCampaignBudgetResponse(_message.Message):
    __slots__ = ("budget",)
    BUDGET_FIELD_NUMBER: _ClassVar[int]
    budget: CampaignBudget
    def __init__(self, budget: _Optional[_Union[CampaignBudget, _Mapping]] = ...) -> None: ...

class RenderProvenance(_message.Message):
    __slots__ = ("spec_id", "identity_version_ids", "backend", "model", "seed", "parameters")
    SPEC_ID_FIELD_NUMBER: _ClassVar[int]
    IDENTITY_VERSION_IDS_FIELD_NUMBER: _ClassVar[int]
    BACKEND_FIELD_NUMBER: _ClassVar[int]
    MODEL_FIELD_NUMBER: _ClassVar[int]
    SEED_FIELD_NUMBER: _ClassVar[int]
    PARAMETERS_FIELD_NUMBER: _ClassVar[int]
    spec_id: str
    identity_version_ids: _containers.RepeatedScalarFieldContainer[str]
    backend: str
    model: str
    seed: str
    parameters: str
    def __init__(self, spec_id: _Optional[str] = ..., identity_version_ids: _Optional[_Iterable[str]] = ..., backend: _Optional[str] = ..., model: _Optional[str] = ..., seed: _Optional[str] = ..., parameters: _Optional[str] = ...) -> None: ...

class Render(_message.Message):
    __slots__ = ("id", "status", "estimated_cost", "actual_cost", "actual_cost_recorded", "provenance", "candidates", "failure_code", "producer_kind", "frame_count", "parent_asset_id")
    ID_FIELD_NUMBER: _ClassVar[int]
    STATUS_FIELD_NUMBER: _ClassVar[int]
    ESTIMATED_COST_FIELD_NUMBER: _ClassVar[int]
    ACTUAL_COST_FIELD_NUMBER: _ClassVar[int]
    ACTUAL_COST_RECORDED_FIELD_NUMBER: _ClassVar[int]
    PROVENANCE_FIELD_NUMBER: _ClassVar[int]
    CANDIDATES_FIELD_NUMBER: _ClassVar[int]
    FAILURE_CODE_FIELD_NUMBER: _ClassVar[int]
    PRODUCER_KIND_FIELD_NUMBER: _ClassVar[int]
    FRAME_COUNT_FIELD_NUMBER: _ClassVar[int]
    PARENT_ASSET_ID_FIELD_NUMBER: _ClassVar[int]
    id: str
    status: str
    estimated_cost: float
    actual_cost: float
    actual_cost_recorded: bool
    provenance: RenderProvenance
    candidates: _containers.RepeatedCompositeFieldContainer[AssetReference]
    failure_code: str
    producer_kind: str
    frame_count: int
    parent_asset_id: str
    def __init__(self, id: _Optional[str] = ..., status: _Optional[str] = ..., estimated_cost: _Optional[float] = ..., actual_cost: _Optional[float] = ..., actual_cost_recorded: _Optional[bool] = ..., provenance: _Optional[_Union[RenderProvenance, _Mapping]] = ..., candidates: _Optional[_Iterable[_Union[AssetReference, _Mapping]]] = ..., failure_code: _Optional[str] = ..., producer_kind: _Optional[str] = ..., frame_count: _Optional[int] = ..., parent_asset_id: _Optional[str] = ...) -> None: ...

class GetRenderRequest(_message.Message):
    __slots__ = ("render_id",)
    RENDER_ID_FIELD_NUMBER: _ClassVar[int]
    render_id: str
    def __init__(self, render_id: _Optional[str] = ...) -> None: ...

class GetRenderResponse(_message.Message):
    __slots__ = ("render",)
    RENDER_FIELD_NUMBER: _ClassVar[int]
    render: Render
    def __init__(self, render: _Optional[_Union[Render, _Mapping]] = ...) -> None: ...

class SelectCandidateRequest(_message.Message):
    __slots__ = ("asset_id",)
    ASSET_ID_FIELD_NUMBER: _ClassVar[int]
    asset_id: str
    def __init__(self, asset_id: _Optional[str] = ...) -> None: ...

class SelectCandidateResponse(_message.Message):
    __slots__ = ("selected",)
    SELECTED_FIELD_NUMBER: _ClassVar[int]
    selected: AssetReference
    def __init__(self, selected: _Optional[_Union[AssetReference, _Mapping]] = ...) -> None: ...

class JudgeConformanceRequest(_message.Message):
    __slots__ = ("asset_id", "identity_version_id", "actor_id", "actor_kind", "passed", "basis")
    ASSET_ID_FIELD_NUMBER: _ClassVar[int]
    IDENTITY_VERSION_ID_FIELD_NUMBER: _ClassVar[int]
    ACTOR_ID_FIELD_NUMBER: _ClassVar[int]
    ACTOR_KIND_FIELD_NUMBER: _ClassVar[int]
    PASSED_FIELD_NUMBER: _ClassVar[int]
    BASIS_FIELD_NUMBER: _ClassVar[int]
    asset_id: str
    identity_version_id: str
    actor_id: str
    actor_kind: str
    passed: bool
    basis: str
    def __init__(self, asset_id: _Optional[str] = ..., identity_version_id: _Optional[str] = ..., actor_id: _Optional[str] = ..., actor_kind: _Optional[str] = ..., passed: _Optional[bool] = ..., basis: _Optional[str] = ...) -> None: ...

class JudgeConformanceResponse(_message.Message):
    __slots__ = ()
    def __init__(self) -> None: ...

class ReleaseAssetRequest(_message.Message):
    __slots__ = ("asset_id",)
    ASSET_ID_FIELD_NUMBER: _ClassVar[int]
    asset_id: str
    def __init__(self, asset_id: _Optional[str] = ...) -> None: ...

class ReleaseAssetResponse(_message.Message):
    __slots__ = ("asset",)
    ASSET_FIELD_NUMBER: _ClassVar[int]
    asset: AssetReference
    def __init__(self, asset: _Optional[_Union[AssetReference, _Mapping]] = ...) -> None: ...

class GetReleasedAssetReferenceRequest(_message.Message):
    __slots__ = ("asset_id",)
    ASSET_ID_FIELD_NUMBER: _ClassVar[int]
    asset_id: str
    def __init__(self, asset_id: _Optional[str] = ...) -> None: ...

class GetReleasedAssetReferenceResponse(_message.Message):
    __slots__ = ("asset",)
    ASSET_FIELD_NUMBER: _ClassVar[int]
    asset: AssetReference
    def __init__(self, asset: _Optional[_Union[AssetReference, _Mapping]] = ...) -> None: ...

class ExternalProvenance(_message.Message):
    __slots__ = ("producing_scenario", "strategy", "model_backed", "model", "prompt", "negative_prompt", "seed", "conditioning", "parameters")
    PRODUCING_SCENARIO_FIELD_NUMBER: _ClassVar[int]
    STRATEGY_FIELD_NUMBER: _ClassVar[int]
    MODEL_BACKED_FIELD_NUMBER: _ClassVar[int]
    MODEL_FIELD_NUMBER: _ClassVar[int]
    PROMPT_FIELD_NUMBER: _ClassVar[int]
    NEGATIVE_PROMPT_FIELD_NUMBER: _ClassVar[int]
    SEED_FIELD_NUMBER: _ClassVar[int]
    CONDITIONING_FIELD_NUMBER: _ClassVar[int]
    PARAMETERS_FIELD_NUMBER: _ClassVar[int]
    producing_scenario: str
    strategy: str
    model_backed: bool
    model: str
    prompt: str
    negative_prompt: str
    seed: str
    conditioning: ConditioningReference
    parameters: str
    def __init__(self, producing_scenario: _Optional[str] = ..., strategy: _Optional[str] = ..., model_backed: _Optional[bool] = ..., model: _Optional[str] = ..., prompt: _Optional[str] = ..., negative_prompt: _Optional[str] = ..., seed: _Optional[str] = ..., conditioning: _Optional[_Union[ConditioningReference, _Mapping]] = ..., parameters: _Optional[str] = ...) -> None: ...

class IngestExternalAssetRequest(_message.Message):
    __slots__ = ("image", "media_type", "alt_text", "decorative", "width", "height", "provenance", "actor_id", "actor_kind")
    IMAGE_FIELD_NUMBER: _ClassVar[int]
    MEDIA_TYPE_FIELD_NUMBER: _ClassVar[int]
    ALT_TEXT_FIELD_NUMBER: _ClassVar[int]
    DECORATIVE_FIELD_NUMBER: _ClassVar[int]
    WIDTH_FIELD_NUMBER: _ClassVar[int]
    HEIGHT_FIELD_NUMBER: _ClassVar[int]
    PROVENANCE_FIELD_NUMBER: _ClassVar[int]
    ACTOR_ID_FIELD_NUMBER: _ClassVar[int]
    ACTOR_KIND_FIELD_NUMBER: _ClassVar[int]
    image: bytes
    media_type: str
    alt_text: str
    decorative: bool
    width: int
    height: int
    provenance: ExternalProvenance
    actor_id: str
    actor_kind: str
    def __init__(self, image: _Optional[bytes] = ..., media_type: _Optional[str] = ..., alt_text: _Optional[str] = ..., decorative: _Optional[bool] = ..., width: _Optional[int] = ..., height: _Optional[int] = ..., provenance: _Optional[_Union[ExternalProvenance, _Mapping]] = ..., actor_id: _Optional[str] = ..., actor_kind: _Optional[str] = ...) -> None: ...

class IngestExternalAssetResponse(_message.Message):
    __slots__ = ("asset",)
    ASSET_FIELD_NUMBER: _ClassVar[int]
    asset: AssetReference
    def __init__(self, asset: _Optional[_Union[AssetReference, _Mapping]] = ...) -> None: ...

class ImportCanonRequest(_message.Message):
    __slots__ = ("root",)
    ROOT_FIELD_NUMBER: _ClassVar[int]
    root: str
    def __init__(self, root: _Optional[str] = ...) -> None: ...

class ImportCanonResponse(_message.Message):
    __slots__ = ("created", "revised", "errors")
    CREATED_FIELD_NUMBER: _ClassVar[int]
    REVISED_FIELD_NUMBER: _ClassVar[int]
    ERRORS_FIELD_NUMBER: _ClassVar[int]
    created: int
    revised: int
    errors: _containers.RepeatedScalarFieldContainer[str]
    def __init__(self, created: _Optional[int] = ..., revised: _Optional[int] = ..., errors: _Optional[_Iterable[str]] = ...) -> None: ...
