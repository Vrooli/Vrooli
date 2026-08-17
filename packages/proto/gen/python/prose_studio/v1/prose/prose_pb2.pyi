from google.protobuf import struct_pb2 as _struct_pb2
from google.protobuf.internal import containers as _containers
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Iterable as _Iterable, Mapping as _Mapping
from typing import ClassVar as _ClassVar, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class RegistryRequest(_message.Message):
    __slots__ = ()
    def __init__(self) -> None: ...

class RegistryKind(_message.Message):
    __slots__ = ("kind", "description", "parameter_schema")
    KIND_FIELD_NUMBER: _ClassVar[int]
    DESCRIPTION_FIELD_NUMBER: _ClassVar[int]
    PARAMETER_SCHEMA_FIELD_NUMBER: _ClassVar[int]
    kind: str
    description: str
    parameter_schema: _struct_pb2.Struct
    def __init__(self, kind: _Optional[str] = ..., description: _Optional[str] = ..., parameter_schema: _Optional[_Union[_struct_pb2.Struct, _Mapping]] = ...) -> None: ...

class RegistryResponse(_message.Message):
    __slots__ = ("samplers", "policies", "metrics", "transforms")
    SAMPLERS_FIELD_NUMBER: _ClassVar[int]
    POLICIES_FIELD_NUMBER: _ClassVar[int]
    METRICS_FIELD_NUMBER: _ClassVar[int]
    TRANSFORMS_FIELD_NUMBER: _ClassVar[int]
    samplers: _containers.RepeatedCompositeFieldContainer[RegistryKind]
    policies: _containers.RepeatedCompositeFieldContainer[RegistryKind]
    metrics: _containers.RepeatedCompositeFieldContainer[RegistryKind]
    transforms: _containers.RepeatedCompositeFieldContainer[RegistryKind]
    def __init__(self, samplers: _Optional[_Iterable[_Union[RegistryKind, _Mapping]]] = ..., policies: _Optional[_Iterable[_Union[RegistryKind, _Mapping]]] = ..., metrics: _Optional[_Iterable[_Union[RegistryKind, _Mapping]]] = ..., transforms: _Optional[_Iterable[_Union[RegistryKind, _Mapping]]] = ...) -> None: ...

class Style(_message.Message):
    __slots__ = ("key", "version", "parent", "exemplars", "directives", "anti_patterns", "lexicon", "targets", "axis_defaults")
    class TargetsEntry(_message.Message):
        __slots__ = ("key", "value")
        KEY_FIELD_NUMBER: _ClassVar[int]
        VALUE_FIELD_NUMBER: _ClassVar[int]
        key: str
        value: float
        def __init__(self, key: _Optional[str] = ..., value: _Optional[float] = ...) -> None: ...
    class AxisDefaultsEntry(_message.Message):
        __slots__ = ("key", "value")
        KEY_FIELD_NUMBER: _ClassVar[int]
        VALUE_FIELD_NUMBER: _ClassVar[int]
        key: str
        value: str
        def __init__(self, key: _Optional[str] = ..., value: _Optional[str] = ...) -> None: ...
    KEY_FIELD_NUMBER: _ClassVar[int]
    VERSION_FIELD_NUMBER: _ClassVar[int]
    PARENT_FIELD_NUMBER: _ClassVar[int]
    EXEMPLARS_FIELD_NUMBER: _ClassVar[int]
    DIRECTIVES_FIELD_NUMBER: _ClassVar[int]
    ANTI_PATTERNS_FIELD_NUMBER: _ClassVar[int]
    LEXICON_FIELD_NUMBER: _ClassVar[int]
    TARGETS_FIELD_NUMBER: _ClassVar[int]
    AXIS_DEFAULTS_FIELD_NUMBER: _ClassVar[int]
    key: str
    version: int
    parent: str
    exemplars: _containers.RepeatedScalarFieldContainer[str]
    directives: _containers.RepeatedScalarFieldContainer[str]
    anti_patterns: _containers.RepeatedScalarFieldContainer[str]
    lexicon: _containers.RepeatedScalarFieldContainer[str]
    targets: _containers.ScalarMap[str, float]
    axis_defaults: _containers.ScalarMap[str, str]
    def __init__(self, key: _Optional[str] = ..., version: _Optional[int] = ..., parent: _Optional[str] = ..., exemplars: _Optional[_Iterable[str]] = ..., directives: _Optional[_Iterable[str]] = ..., anti_patterns: _Optional[_Iterable[str]] = ..., lexicon: _Optional[_Iterable[str]] = ..., targets: _Optional[_Mapping[str, float]] = ..., axis_defaults: _Optional[_Mapping[str, str]] = ...) -> None: ...

class CreateStyleRequest(_message.Message):
    __slots__ = ("style",)
    STYLE_FIELD_NUMBER: _ClassVar[int]
    style: Style
    def __init__(self, style: _Optional[_Union[Style, _Mapping]] = ...) -> None: ...

class CreateStyleResponse(_message.Message):
    __slots__ = ("style",)
    STYLE_FIELD_NUMBER: _ClassVar[int]
    style: Style
    def __init__(self, style: _Optional[_Union[Style, _Mapping]] = ...) -> None: ...

class Sampler(_message.Message):
    __slots__ = ("kind", "k", "tau", "temperature_stance", "max_output_tokens")
    KIND_FIELD_NUMBER: _ClassVar[int]
    K_FIELD_NUMBER: _ClassVar[int]
    TAU_FIELD_NUMBER: _ClassVar[int]
    TEMPERATURE_STANCE_FIELD_NUMBER: _ClassVar[int]
    MAX_OUTPUT_TOKENS_FIELD_NUMBER: _ClassVar[int]
    kind: str
    k: int
    tau: float
    temperature_stance: str
    max_output_tokens: int
    def __init__(self, kind: _Optional[str] = ..., k: _Optional[int] = ..., tau: _Optional[float] = ..., temperature_stance: _Optional[str] = ..., max_output_tokens: _Optional[int] = ...) -> None: ...

class Constraints(_message.Message):
    __slots__ = ("min_words", "max_words", "min_grade", "max_grade", "banned_lexicon", "required_format")
    MIN_WORDS_FIELD_NUMBER: _ClassVar[int]
    MAX_WORDS_FIELD_NUMBER: _ClassVar[int]
    MIN_GRADE_FIELD_NUMBER: _ClassVar[int]
    MAX_GRADE_FIELD_NUMBER: _ClassVar[int]
    BANNED_LEXICON_FIELD_NUMBER: _ClassVar[int]
    REQUIRED_FORMAT_FIELD_NUMBER: _ClassVar[int]
    min_words: int
    max_words: int
    min_grade: float
    max_grade: float
    banned_lexicon: _containers.RepeatedScalarFieldContainer[str]
    required_format: str
    def __init__(self, min_words: _Optional[int] = ..., max_words: _Optional[int] = ..., min_grade: _Optional[float] = ..., max_grade: _Optional[float] = ..., banned_lexicon: _Optional[_Iterable[str]] = ..., required_format: _Optional[str] = ...) -> None: ...

class Budget(_message.Message):
    __slots__ = ("max_output_tokens", "max_session_cost_micros")
    MAX_OUTPUT_TOKENS_FIELD_NUMBER: _ClassVar[int]
    MAX_SESSION_COST_MICROS_FIELD_NUMBER: _ClassVar[int]
    max_output_tokens: int
    max_session_cost_micros: int
    def __init__(self, max_output_tokens: _Optional[int] = ..., max_session_cost_micros: _Optional[int] = ...) -> None: ...

class ContextPolicy(_message.Message):
    __slots__ = ("full_text_token_budget", "summarize_beyond", "always_full_previous", "declared_context_ceiling")
    FULL_TEXT_TOKEN_BUDGET_FIELD_NUMBER: _ClassVar[int]
    SUMMARIZE_BEYOND_FIELD_NUMBER: _ClassVar[int]
    ALWAYS_FULL_PREVIOUS_FIELD_NUMBER: _ClassVar[int]
    DECLARED_CONTEXT_CEILING_FIELD_NUMBER: _ClassVar[int]
    full_text_token_budget: int
    summarize_beyond: int
    always_full_previous: bool
    declared_context_ceiling: int
    def __init__(self, full_text_token_budget: _Optional[int] = ..., summarize_beyond: _Optional[int] = ..., always_full_previous: _Optional[bool] = ..., declared_context_ceiling: _Optional[int] = ...) -> None: ...

class Profile(_message.Message):
    __slots__ = ("key", "version", "parent", "style_refs", "sampler", "constraints", "selection_policy", "selection_params", "measurement_tiers", "budget", "context_policy", "gateway_role", "locality")
    class SelectionParamsEntry(_message.Message):
        __slots__ = ("key", "value")
        KEY_FIELD_NUMBER: _ClassVar[int]
        VALUE_FIELD_NUMBER: _ClassVar[int]
        key: str
        value: float
        def __init__(self, key: _Optional[str] = ..., value: _Optional[float] = ...) -> None: ...
    KEY_FIELD_NUMBER: _ClassVar[int]
    VERSION_FIELD_NUMBER: _ClassVar[int]
    PARENT_FIELD_NUMBER: _ClassVar[int]
    STYLE_REFS_FIELD_NUMBER: _ClassVar[int]
    SAMPLER_FIELD_NUMBER: _ClassVar[int]
    CONSTRAINTS_FIELD_NUMBER: _ClassVar[int]
    SELECTION_POLICY_FIELD_NUMBER: _ClassVar[int]
    SELECTION_PARAMS_FIELD_NUMBER: _ClassVar[int]
    MEASUREMENT_TIERS_FIELD_NUMBER: _ClassVar[int]
    BUDGET_FIELD_NUMBER: _ClassVar[int]
    CONTEXT_POLICY_FIELD_NUMBER: _ClassVar[int]
    GATEWAY_ROLE_FIELD_NUMBER: _ClassVar[int]
    LOCALITY_FIELD_NUMBER: _ClassVar[int]
    key: str
    version: int
    parent: str
    style_refs: _containers.RepeatedScalarFieldContainer[str]
    sampler: Sampler
    constraints: Constraints
    selection_policy: str
    selection_params: _containers.ScalarMap[str, float]
    measurement_tiers: _containers.RepeatedScalarFieldContainer[str]
    budget: Budget
    context_policy: ContextPolicy
    gateway_role: str
    locality: str
    def __init__(self, key: _Optional[str] = ..., version: _Optional[int] = ..., parent: _Optional[str] = ..., style_refs: _Optional[_Iterable[str]] = ..., sampler: _Optional[_Union[Sampler, _Mapping]] = ..., constraints: _Optional[_Union[Constraints, _Mapping]] = ..., selection_policy: _Optional[str] = ..., selection_params: _Optional[_Mapping[str, float]] = ..., measurement_tiers: _Optional[_Iterable[str]] = ..., budget: _Optional[_Union[Budget, _Mapping]] = ..., context_policy: _Optional[_Union[ContextPolicy, _Mapping]] = ..., gateway_role: _Optional[str] = ..., locality: _Optional[str] = ...) -> None: ...

class ResolveProfileRequest(_message.Message):
    __slots__ = ("key",)
    KEY_FIELD_NUMBER: _ClassVar[int]
    key: str
    def __init__(self, key: _Optional[str] = ...) -> None: ...

class ResolveProfileResponse(_message.Message):
    __slots__ = ("profile", "styles", "instruction_text")
    PROFILE_FIELD_NUMBER: _ClassVar[int]
    STYLES_FIELD_NUMBER: _ClassVar[int]
    INSTRUCTION_TEXT_FIELD_NUMBER: _ClassVar[int]
    profile: Profile
    styles: _containers.RepeatedCompositeFieldContainer[Style]
    instruction_text: str
    def __init__(self, profile: _Optional[_Union[Profile, _Mapping]] = ..., styles: _Optional[_Iterable[_Union[Style, _Mapping]]] = ..., instruction_text: _Optional[str] = ...) -> None: ...

class NegativeContext(_message.Message):
    __slots__ = ("pinned", "rejected")
    PINNED_FIELD_NUMBER: _ClassVar[int]
    REJECTED_FIELD_NUMBER: _ClassVar[int]
    pinned: _containers.RepeatedScalarFieldContainer[str]
    rejected: _containers.RepeatedScalarFieldContainer[str]
    def __init__(self, pinned: _Optional[_Iterable[str]] = ..., rejected: _Optional[_Iterable[str]] = ...) -> None: ...

class GenerateRequest(_message.Message):
    __slots__ = ("profile_key", "query", "include_candidates", "session_id", "negative")
    PROFILE_KEY_FIELD_NUMBER: _ClassVar[int]
    QUERY_FIELD_NUMBER: _ClassVar[int]
    INCLUDE_CANDIDATES_FIELD_NUMBER: _ClassVar[int]
    SESSION_ID_FIELD_NUMBER: _ClassVar[int]
    NEGATIVE_FIELD_NUMBER: _ClassVar[int]
    profile_key: str
    query: str
    include_candidates: bool
    session_id: str
    negative: NegativeContext
    def __init__(self, profile_key: _Optional[str] = ..., query: _Optional[str] = ..., include_candidates: _Optional[bool] = ..., session_id: _Optional[str] = ..., negative: _Optional[_Union[NegativeContext, _Mapping]] = ...) -> None: ...

class Eligibility(_message.Message):
    __slots__ = ("eligible", "reason")
    ELIGIBLE_FIELD_NUMBER: _ClassVar[int]
    REASON_FIELD_NUMBER: _ClassVar[int]
    eligible: bool
    reason: str
    def __init__(self, eligible: _Optional[bool] = ..., reason: _Optional[str] = ...) -> None: ...

class VerbalizedHint(_message.Message):
    __slots__ = ("ordinal", "calibrated")
    ORDINAL_FIELD_NUMBER: _ClassVar[int]
    CALIBRATED_FIELD_NUMBER: _ClassVar[int]
    ordinal: int
    calibrated: bool
    def __init__(self, ordinal: _Optional[int] = ..., calibrated: _Optional[bool] = ...) -> None: ...

class Provenance(_message.Message):
    __slots__ = ("profile_version", "style_versions", "strategy", "strategy_parameters", "provider", "resolved_model_ref", "gateway_role", "temperature_sent", "temperature_support", "max_output_tokens_effective", "max_output_tokens_source", "input_tokens", "output_tokens", "cost_micros", "machine_generated", "disclosure", "context_snapshot")
    PROFILE_VERSION_FIELD_NUMBER: _ClassVar[int]
    STYLE_VERSIONS_FIELD_NUMBER: _ClassVar[int]
    STRATEGY_FIELD_NUMBER: _ClassVar[int]
    STRATEGY_PARAMETERS_FIELD_NUMBER: _ClassVar[int]
    PROVIDER_FIELD_NUMBER: _ClassVar[int]
    RESOLVED_MODEL_REF_FIELD_NUMBER: _ClassVar[int]
    GATEWAY_ROLE_FIELD_NUMBER: _ClassVar[int]
    TEMPERATURE_SENT_FIELD_NUMBER: _ClassVar[int]
    TEMPERATURE_SUPPORT_FIELD_NUMBER: _ClassVar[int]
    MAX_OUTPUT_TOKENS_EFFECTIVE_FIELD_NUMBER: _ClassVar[int]
    MAX_OUTPUT_TOKENS_SOURCE_FIELD_NUMBER: _ClassVar[int]
    INPUT_TOKENS_FIELD_NUMBER: _ClassVar[int]
    OUTPUT_TOKENS_FIELD_NUMBER: _ClassVar[int]
    COST_MICROS_FIELD_NUMBER: _ClassVar[int]
    MACHINE_GENERATED_FIELD_NUMBER: _ClassVar[int]
    DISCLOSURE_FIELD_NUMBER: _ClassVar[int]
    CONTEXT_SNAPSHOT_FIELD_NUMBER: _ClassVar[int]
    profile_version: str
    style_versions: _containers.RepeatedScalarFieldContainer[str]
    strategy: str
    strategy_parameters: Sampler
    provider: str
    resolved_model_ref: str
    gateway_role: str
    temperature_sent: float
    temperature_support: str
    max_output_tokens_effective: int
    max_output_tokens_source: str
    input_tokens: int
    output_tokens: int
    cost_micros: int
    machine_generated: bool
    disclosure: str
    context_snapshot: ContextSnapshot
    def __init__(self, profile_version: _Optional[str] = ..., style_versions: _Optional[_Iterable[str]] = ..., strategy: _Optional[str] = ..., strategy_parameters: _Optional[_Union[Sampler, _Mapping]] = ..., provider: _Optional[str] = ..., resolved_model_ref: _Optional[str] = ..., gateway_role: _Optional[str] = ..., temperature_sent: _Optional[float] = ..., temperature_support: _Optional[str] = ..., max_output_tokens_effective: _Optional[int] = ..., max_output_tokens_source: _Optional[str] = ..., input_tokens: _Optional[int] = ..., output_tokens: _Optional[int] = ..., cost_micros: _Optional[int] = ..., machine_generated: _Optional[bool] = ..., disclosure: _Optional[str] = ..., context_snapshot: _Optional[_Union[ContextSnapshot, _Mapping]] = ...) -> None: ...

class Candidate(_message.Message):
    __slots__ = ("id", "round_id", "derived_from", "text", "measurements", "set_measurements", "provenance", "verbalized_hint", "eligibility", "committed", "set_index")
    ID_FIELD_NUMBER: _ClassVar[int]
    ROUND_ID_FIELD_NUMBER: _ClassVar[int]
    DERIVED_FROM_FIELD_NUMBER: _ClassVar[int]
    TEXT_FIELD_NUMBER: _ClassVar[int]
    MEASUREMENTS_FIELD_NUMBER: _ClassVar[int]
    SET_MEASUREMENTS_FIELD_NUMBER: _ClassVar[int]
    PROVENANCE_FIELD_NUMBER: _ClassVar[int]
    VERBALIZED_HINT_FIELD_NUMBER: _ClassVar[int]
    ELIGIBILITY_FIELD_NUMBER: _ClassVar[int]
    COMMITTED_FIELD_NUMBER: _ClassVar[int]
    SET_INDEX_FIELD_NUMBER: _ClassVar[int]
    id: str
    round_id: str
    derived_from: _containers.RepeatedScalarFieldContainer[str]
    text: str
    measurements: _struct_pb2.Struct
    set_measurements: _struct_pb2.Struct
    provenance: Provenance
    verbalized_hint: VerbalizedHint
    eligibility: Eligibility
    committed: bool
    set_index: int
    def __init__(self, id: _Optional[str] = ..., round_id: _Optional[str] = ..., derived_from: _Optional[_Iterable[str]] = ..., text: _Optional[str] = ..., measurements: _Optional[_Union[_struct_pb2.Struct, _Mapping]] = ..., set_measurements: _Optional[_Union[_struct_pb2.Struct, _Mapping]] = ..., provenance: _Optional[_Union[Provenance, _Mapping]] = ..., verbalized_hint: _Optional[_Union[VerbalizedHint, _Mapping]] = ..., eligibility: _Optional[_Union[Eligibility, _Mapping]] = ..., committed: _Optional[bool] = ..., set_index: _Optional[int] = ...) -> None: ...

class Round(_message.Message):
    __slots__ = ("id", "session_id", "strategy", "candidate_ids", "candidate_count", "sampling_key", "total_cost_micros", "negative_context", "selection_seed")
    ID_FIELD_NUMBER: _ClassVar[int]
    SESSION_ID_FIELD_NUMBER: _ClassVar[int]
    STRATEGY_FIELD_NUMBER: _ClassVar[int]
    CANDIDATE_IDS_FIELD_NUMBER: _ClassVar[int]
    CANDIDATE_COUNT_FIELD_NUMBER: _ClassVar[int]
    SAMPLING_KEY_FIELD_NUMBER: _ClassVar[int]
    TOTAL_COST_MICROS_FIELD_NUMBER: _ClassVar[int]
    NEGATIVE_CONTEXT_FIELD_NUMBER: _ClassVar[int]
    SELECTION_SEED_FIELD_NUMBER: _ClassVar[int]
    id: str
    session_id: str
    strategy: Sampler
    candidate_ids: _containers.RepeatedScalarFieldContainer[str]
    candidate_count: int
    sampling_key: _struct_pb2.Struct
    total_cost_micros: int
    negative_context: NegativeContext
    selection_seed: int
    def __init__(self, id: _Optional[str] = ..., session_id: _Optional[str] = ..., strategy: _Optional[_Union[Sampler, _Mapping]] = ..., candidate_ids: _Optional[_Iterable[str]] = ..., candidate_count: _Optional[int] = ..., sampling_key: _Optional[_Union[_struct_pb2.Struct, _Mapping]] = ..., total_cost_micros: _Optional[int] = ..., negative_context: _Optional[_Union[NegativeContext, _Mapping]] = ..., selection_seed: _Optional[int] = ...) -> None: ...

class Session(_message.Message):
    __slots__ = ("id", "profile_key", "query", "status", "pinned", "rejected", "round_ids", "budget_used_micros")
    ID_FIELD_NUMBER: _ClassVar[int]
    PROFILE_KEY_FIELD_NUMBER: _ClassVar[int]
    QUERY_FIELD_NUMBER: _ClassVar[int]
    STATUS_FIELD_NUMBER: _ClassVar[int]
    PINNED_FIELD_NUMBER: _ClassVar[int]
    REJECTED_FIELD_NUMBER: _ClassVar[int]
    ROUND_IDS_FIELD_NUMBER: _ClassVar[int]
    BUDGET_USED_MICROS_FIELD_NUMBER: _ClassVar[int]
    id: str
    profile_key: str
    query: str
    status: str
    pinned: _containers.RepeatedScalarFieldContainer[str]
    rejected: _containers.RepeatedScalarFieldContainer[str]
    round_ids: _containers.RepeatedScalarFieldContainer[str]
    budget_used_micros: int
    def __init__(self, id: _Optional[str] = ..., profile_key: _Optional[str] = ..., query: _Optional[str] = ..., status: _Optional[str] = ..., pinned: _Optional[_Iterable[str]] = ..., rejected: _Optional[_Iterable[str]] = ..., round_ids: _Optional[_Iterable[str]] = ..., budget_used_micros: _Optional[int] = ...) -> None: ...

class DegradedOutcome(_message.Message):
    __slots__ = ("kind", "reason", "requested_candidates", "received_candidates", "max_output_tokens_effective", "max_output_tokens_source")
    KIND_FIELD_NUMBER: _ClassVar[int]
    REASON_FIELD_NUMBER: _ClassVar[int]
    REQUESTED_CANDIDATES_FIELD_NUMBER: _ClassVar[int]
    RECEIVED_CANDIDATES_FIELD_NUMBER: _ClassVar[int]
    MAX_OUTPUT_TOKENS_EFFECTIVE_FIELD_NUMBER: _ClassVar[int]
    MAX_OUTPUT_TOKENS_SOURCE_FIELD_NUMBER: _ClassVar[int]
    kind: str
    reason: str
    requested_candidates: int
    received_candidates: int
    max_output_tokens_effective: int
    max_output_tokens_source: str
    def __init__(self, kind: _Optional[str] = ..., reason: _Optional[str] = ..., requested_candidates: _Optional[int] = ..., received_candidates: _Optional[int] = ..., max_output_tokens_effective: _Optional[int] = ..., max_output_tokens_source: _Optional[str] = ...) -> None: ...

class GenerateResponse(_message.Message):
    __slots__ = ("session", "round", "selected", "candidates", "selected_candidates", "degraded")
    SESSION_FIELD_NUMBER: _ClassVar[int]
    ROUND_FIELD_NUMBER: _ClassVar[int]
    SELECTED_FIELD_NUMBER: _ClassVar[int]
    CANDIDATES_FIELD_NUMBER: _ClassVar[int]
    SELECTED_CANDIDATES_FIELD_NUMBER: _ClassVar[int]
    DEGRADED_FIELD_NUMBER: _ClassVar[int]
    session: Session
    round: Round
    selected: Candidate
    candidates: _containers.RepeatedCompositeFieldContainer[Candidate]
    selected_candidates: _containers.RepeatedCompositeFieldContainer[Candidate]
    degraded: DegradedOutcome
    def __init__(self, session: _Optional[_Union[Session, _Mapping]] = ..., round: _Optional[_Union[Round, _Mapping]] = ..., selected: _Optional[_Union[Candidate, _Mapping]] = ..., candidates: _Optional[_Iterable[_Union[Candidate, _Mapping]]] = ..., selected_candidates: _Optional[_Iterable[_Union[Candidate, _Mapping]]] = ..., degraded: _Optional[_Union[DegradedOutcome, _Mapping]] = ...) -> None: ...

class RerollRequest(_message.Message):
    __slots__ = ("session_id", "include_candidates")
    SESSION_ID_FIELD_NUMBER: _ClassVar[int]
    INCLUDE_CANDIDATES_FIELD_NUMBER: _ClassVar[int]
    session_id: str
    include_candidates: bool
    def __init__(self, session_id: _Optional[str] = ..., include_candidates: _Optional[bool] = ...) -> None: ...

class RerollResponse(_message.Message):
    __slots__ = ("result",)
    RESULT_FIELD_NUMBER: _ClassVar[int]
    result: GenerateResponse
    def __init__(self, result: _Optional[_Union[GenerateResponse, _Mapping]] = ...) -> None: ...

class SessionActionRequest(_message.Message):
    __slots__ = ("action", "session_id", "candidate_id")
    ACTION_FIELD_NUMBER: _ClassVar[int]
    SESSION_ID_FIELD_NUMBER: _ClassVar[int]
    CANDIDATE_ID_FIELD_NUMBER: _ClassVar[int]
    action: str
    session_id: str
    candidate_id: str
    def __init__(self, action: _Optional[str] = ..., session_id: _Optional[str] = ..., candidate_id: _Optional[str] = ...) -> None: ...

class SessionActionResponse(_message.Message):
    __slots__ = ("session",)
    SESSION_FIELD_NUMBER: _ClassVar[int]
    session: Session
    def __init__(self, session: _Optional[_Union[Session, _Mapping]] = ...) -> None: ...

class Declaration(_message.Message):
    __slots__ = ("path", "schema_version", "kind", "key", "created_by", "content_hash", "status", "error", "record")
    PATH_FIELD_NUMBER: _ClassVar[int]
    SCHEMA_VERSION_FIELD_NUMBER: _ClassVar[int]
    KIND_FIELD_NUMBER: _ClassVar[int]
    KEY_FIELD_NUMBER: _ClassVar[int]
    CREATED_BY_FIELD_NUMBER: _ClassVar[int]
    CONTENT_HASH_FIELD_NUMBER: _ClassVar[int]
    STATUS_FIELD_NUMBER: _ClassVar[int]
    ERROR_FIELD_NUMBER: _ClassVar[int]
    RECORD_FIELD_NUMBER: _ClassVar[int]
    path: str
    schema_version: str
    kind: str
    key: str
    created_by: str
    content_hash: str
    status: str
    error: str
    record: _struct_pb2.Struct
    def __init__(self, path: _Optional[str] = ..., schema_version: _Optional[str] = ..., kind: _Optional[str] = ..., key: _Optional[str] = ..., created_by: _Optional[str] = ..., content_hash: _Optional[str] = ..., status: _Optional[str] = ..., error: _Optional[str] = ..., record: _Optional[_Union[_struct_pb2.Struct, _Mapping]] = ...) -> None: ...

class ReindexDeclarationsRequest(_message.Message):
    __slots__ = ("root",)
    ROOT_FIELD_NUMBER: _ClassVar[int]
    root: str
    def __init__(self, root: _Optional[str] = ...) -> None: ...

class ReindexDeclarationsResponse(_message.Message):
    __slots__ = ("declarations",)
    DECLARATIONS_FIELD_NUMBER: _ClassVar[int]
    declarations: _containers.RepeatedCompositeFieldContainer[Declaration]
    def __init__(self, declarations: _Optional[_Iterable[_Union[Declaration, _Mapping]]] = ...) -> None: ...

class ValidateDeclarationsRequest(_message.Message):
    __slots__ = ("root",)
    ROOT_FIELD_NUMBER: _ClassVar[int]
    root: str
    def __init__(self, root: _Optional[str] = ...) -> None: ...

class ValidateDeclarationsResponse(_message.Message):
    __slots__ = ("declarations",)
    DECLARATIONS_FIELD_NUMBER: _ClassVar[int]
    declarations: _containers.RepeatedCompositeFieldContainer[Declaration]
    def __init__(self, declarations: _Optional[_Iterable[_Union[Declaration, _Mapping]]] = ...) -> None: ...

class ContextSnapshot(_message.Message):
    __slots__ = ("outline_ref", "prior_section_refs", "following_intents", "summarized_section_refs", "estimated_tokens")
    OUTLINE_REF_FIELD_NUMBER: _ClassVar[int]
    PRIOR_SECTION_REFS_FIELD_NUMBER: _ClassVar[int]
    FOLLOWING_INTENTS_FIELD_NUMBER: _ClassVar[int]
    SUMMARIZED_SECTION_REFS_FIELD_NUMBER: _ClassVar[int]
    ESTIMATED_TOKENS_FIELD_NUMBER: _ClassVar[int]
    outline_ref: str
    prior_section_refs: _containers.RepeatedScalarFieldContainer[str]
    following_intents: _containers.RepeatedScalarFieldContainer[str]
    summarized_section_refs: _containers.RepeatedScalarFieldContainer[str]
    estimated_tokens: int
    def __init__(self, outline_ref: _Optional[str] = ..., prior_section_refs: _Optional[_Iterable[str]] = ..., following_intents: _Optional[_Iterable[str]] = ..., summarized_section_refs: _Optional[_Iterable[str]] = ..., estimated_tokens: _Optional[int] = ...) -> None: ...

class Section(_message.Message):
    __slots__ = ("id", "document_id", "position", "intent", "profile_key", "session_id", "committed_candidate_id", "context")
    ID_FIELD_NUMBER: _ClassVar[int]
    DOCUMENT_ID_FIELD_NUMBER: _ClassVar[int]
    POSITION_FIELD_NUMBER: _ClassVar[int]
    INTENT_FIELD_NUMBER: _ClassVar[int]
    PROFILE_KEY_FIELD_NUMBER: _ClassVar[int]
    SESSION_ID_FIELD_NUMBER: _ClassVar[int]
    COMMITTED_CANDIDATE_ID_FIELD_NUMBER: _ClassVar[int]
    CONTEXT_FIELD_NUMBER: _ClassVar[int]
    id: str
    document_id: str
    position: int
    intent: str
    profile_key: str
    session_id: str
    committed_candidate_id: str
    context: ContextSnapshot
    def __init__(self, id: _Optional[str] = ..., document_id: _Optional[str] = ..., position: _Optional[int] = ..., intent: _Optional[str] = ..., profile_key: _Optional[str] = ..., session_id: _Optional[str] = ..., committed_candidate_id: _Optional[str] = ..., context: _Optional[_Union[ContextSnapshot, _Mapping]] = ...) -> None: ...

class Document(_message.Message):
    __slots__ = ("id", "title", "profile_key", "style_key", "outline_id", "section_ids", "status", "assembled_text", "coherence", "sections")
    ID_FIELD_NUMBER: _ClassVar[int]
    TITLE_FIELD_NUMBER: _ClassVar[int]
    PROFILE_KEY_FIELD_NUMBER: _ClassVar[int]
    STYLE_KEY_FIELD_NUMBER: _ClassVar[int]
    OUTLINE_ID_FIELD_NUMBER: _ClassVar[int]
    SECTION_IDS_FIELD_NUMBER: _ClassVar[int]
    STATUS_FIELD_NUMBER: _ClassVar[int]
    ASSEMBLED_TEXT_FIELD_NUMBER: _ClassVar[int]
    COHERENCE_FIELD_NUMBER: _ClassVar[int]
    SECTIONS_FIELD_NUMBER: _ClassVar[int]
    id: str
    title: str
    profile_key: str
    style_key: str
    outline_id: str
    section_ids: _containers.RepeatedScalarFieldContainer[str]
    status: str
    assembled_text: str
    coherence: _struct_pb2.Struct
    sections: _containers.RepeatedCompositeFieldContainer[Section]
    def __init__(self, id: _Optional[str] = ..., title: _Optional[str] = ..., profile_key: _Optional[str] = ..., style_key: _Optional[str] = ..., outline_id: _Optional[str] = ..., section_ids: _Optional[_Iterable[str]] = ..., status: _Optional[str] = ..., assembled_text: _Optional[str] = ..., coherence: _Optional[_Union[_struct_pb2.Struct, _Mapping]] = ..., sections: _Optional[_Iterable[_Union[Section, _Mapping]]] = ...) -> None: ...

class CreateDocumentRequest(_message.Message):
    __slots__ = ("document", "sections")
    DOCUMENT_FIELD_NUMBER: _ClassVar[int]
    SECTIONS_FIELD_NUMBER: _ClassVar[int]
    document: Document
    sections: _containers.RepeatedCompositeFieldContainer[Section]
    def __init__(self, document: _Optional[_Union[Document, _Mapping]] = ..., sections: _Optional[_Iterable[_Union[Section, _Mapping]]] = ...) -> None: ...

class CreateDocumentResponse(_message.Message):
    __slots__ = ("document",)
    DOCUMENT_FIELD_NUMBER: _ClassVar[int]
    document: Document
    def __init__(self, document: _Optional[_Union[Document, _Mapping]] = ...) -> None: ...

class AssembleDocumentRequest(_message.Message):
    __slots__ = ("id",)
    ID_FIELD_NUMBER: _ClassVar[int]
    id: str
    def __init__(self, id: _Optional[str] = ...) -> None: ...

class AssembleDocumentResponse(_message.Message):
    __slots__ = ("document",)
    DOCUMENT_FIELD_NUMBER: _ClassVar[int]
    document: Document
    def __init__(self, document: _Optional[_Union[Document, _Mapping]] = ...) -> None: ...

class ResumeDocumentRequest(_message.Message):
    __slots__ = ("id",)
    ID_FIELD_NUMBER: _ClassVar[int]
    id: str
    def __init__(self, id: _Optional[str] = ...) -> None: ...

class ResumeDocumentResponse(_message.Message):
    __slots__ = ("document",)
    DOCUMENT_FIELD_NUMBER: _ClassVar[int]
    document: Document
    def __init__(self, document: _Optional[_Union[Document, _Mapping]] = ...) -> None: ...

class ConformanceRequest(_message.Message):
    __slots__ = ("style_key", "text")
    STYLE_KEY_FIELD_NUMBER: _ClassVar[int]
    TEXT_FIELD_NUMBER: _ClassVar[int]
    style_key: str
    text: str
    def __init__(self, style_key: _Optional[str] = ..., text: _Optional[str] = ...) -> None: ...

class ConformanceResponse(_message.Message):
    __slots__ = ("report",)
    REPORT_FIELD_NUMBER: _ClassVar[int]
    report: _struct_pb2.Struct
    def __init__(self, report: _Optional[_Union[_struct_pb2.Struct, _Mapping]] = ...) -> None: ...
