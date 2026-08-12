import datetime

from common.v1 import attestation_pb2 as _attestation_pb2
from common.v1 import confidence_pb2 as _confidence_pb2
from google.protobuf import timestamp_pb2 as _timestamp_pb2
from google.protobuf.internal import containers as _containers
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Iterable as _Iterable, Mapping as _Mapping
from typing import ClassVar as _ClassVar, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class QueryRequest(_message.Message):
    __slots__ = ("query", "types", "all", "limit", "group", "explain", "overrides", "control_token")
    QUERY_FIELD_NUMBER: _ClassVar[int]
    TYPES_FIELD_NUMBER: _ClassVar[int]
    ALL_FIELD_NUMBER: _ClassVar[int]
    LIMIT_FIELD_NUMBER: _ClassVar[int]
    GROUP_FIELD_NUMBER: _ClassVar[int]
    EXPLAIN_FIELD_NUMBER: _ClassVar[int]
    OVERRIDES_FIELD_NUMBER: _ClassVar[int]
    CONTROL_TOKEN_FIELD_NUMBER: _ClassVar[int]
    query: str
    types: _containers.RepeatedScalarFieldContainer[str]
    all: bool
    limit: int
    group: str
    explain: bool
    overrides: SearchOverrides
    control_token: str
    def __init__(self, query: _Optional[str] = ..., types: _Optional[_Iterable[str]] = ..., all: _Optional[bool] = ..., limit: _Optional[int] = ..., group: _Optional[str] = ..., explain: _Optional[bool] = ..., overrides: _Optional[_Union[SearchOverrides, _Mapping]] = ..., control_token: _Optional[str] = ...) -> None: ...

class SearchOverrides(_message.Message):
    __slots__ = ("rerank_enabled", "rerank_blend", "rerank_shortlist", "floor_max_gap", "floor_hard_floor", "hybrid_fusion")
    RERANK_ENABLED_FIELD_NUMBER: _ClassVar[int]
    RERANK_BLEND_FIELD_NUMBER: _ClassVar[int]
    RERANK_SHORTLIST_FIELD_NUMBER: _ClassVar[int]
    FLOOR_MAX_GAP_FIELD_NUMBER: _ClassVar[int]
    FLOOR_HARD_FLOOR_FIELD_NUMBER: _ClassVar[int]
    HYBRID_FUSION_FIELD_NUMBER: _ClassVar[int]
    rerank_enabled: bool
    rerank_blend: bool
    rerank_shortlist: int
    floor_max_gap: float
    floor_hard_floor: float
    hybrid_fusion: str
    def __init__(self, rerank_enabled: _Optional[bool] = ..., rerank_blend: _Optional[bool] = ..., rerank_shortlist: _Optional[int] = ..., floor_max_gap: _Optional[float] = ..., floor_hard_floor: _Optional[float] = ..., hybrid_fusion: _Optional[str] = ...) -> None: ...

class SearchHit(_message.Message):
    __slots__ = ("provider_id", "provider_group", "type", "id", "title", "snippet", "path", "score", "rerank_score", "measure", "attestation", "confidence", "locations", "merged_count")
    PROVIDER_ID_FIELD_NUMBER: _ClassVar[int]
    PROVIDER_GROUP_FIELD_NUMBER: _ClassVar[int]
    TYPE_FIELD_NUMBER: _ClassVar[int]
    ID_FIELD_NUMBER: _ClassVar[int]
    TITLE_FIELD_NUMBER: _ClassVar[int]
    SNIPPET_FIELD_NUMBER: _ClassVar[int]
    PATH_FIELD_NUMBER: _ClassVar[int]
    SCORE_FIELD_NUMBER: _ClassVar[int]
    RERANK_SCORE_FIELD_NUMBER: _ClassVar[int]
    MEASURE_FIELD_NUMBER: _ClassVar[int]
    ATTESTATION_FIELD_NUMBER: _ClassVar[int]
    CONFIDENCE_FIELD_NUMBER: _ClassVar[int]
    LOCATIONS_FIELD_NUMBER: _ClassVar[int]
    MERGED_COUNT_FIELD_NUMBER: _ClassVar[int]
    provider_id: str
    provider_group: str
    type: str
    id: str
    title: str
    snippet: str
    path: str
    score: float
    rerank_score: float
    measure: MeasureHit
    attestation: _attestation_pb2.AttestedAnswer
    confidence: _confidence_pb2.Confidence
    locations: _containers.RepeatedScalarFieldContainer[str]
    merged_count: int
    def __init__(self, provider_id: _Optional[str] = ..., provider_group: _Optional[str] = ..., type: _Optional[str] = ..., id: _Optional[str] = ..., title: _Optional[str] = ..., snippet: _Optional[str] = ..., path: _Optional[str] = ..., score: _Optional[float] = ..., rerank_score: _Optional[float] = ..., measure: _Optional[_Union[MeasureHit, _Mapping]] = ..., attestation: _Optional[_Union[_attestation_pb2.AttestedAnswer, _Mapping]] = ..., confidence: _Optional[_Union[_confidence_pb2.Confidence, _Mapping]] = ..., locations: _Optional[_Iterable[str]] = ..., merged_count: _Optional[int] = ...) -> None: ...

class MeasureHit(_message.Message):
    __slots__ = ("measure_id", "scenario", "params", "answer", "needs", "effect", "executed_query", "confidence")
    class ParamsEntry(_message.Message):
        __slots__ = ("key", "value")
        KEY_FIELD_NUMBER: _ClassVar[int]
        VALUE_FIELD_NUMBER: _ClassVar[int]
        key: str
        value: str
        def __init__(self, key: _Optional[str] = ..., value: _Optional[str] = ...) -> None: ...
    MEASURE_ID_FIELD_NUMBER: _ClassVar[int]
    SCENARIO_FIELD_NUMBER: _ClassVar[int]
    PARAMS_FIELD_NUMBER: _ClassVar[int]
    ANSWER_FIELD_NUMBER: _ClassVar[int]
    NEEDS_FIELD_NUMBER: _ClassVar[int]
    EFFECT_FIELD_NUMBER: _ClassVar[int]
    EXECUTED_QUERY_FIELD_NUMBER: _ClassVar[int]
    CONFIDENCE_FIELD_NUMBER: _ClassVar[int]
    measure_id: str
    scenario: str
    params: _containers.ScalarMap[str, str]
    answer: str
    needs: _containers.RepeatedScalarFieldContainer[str]
    effect: str
    executed_query: str
    confidence: float
    def __init__(self, measure_id: _Optional[str] = ..., scenario: _Optional[str] = ..., params: _Optional[_Mapping[str, str]] = ..., answer: _Optional[str] = ..., needs: _Optional[_Iterable[str]] = ..., effect: _Optional[str] = ..., executed_query: _Optional[str] = ..., confidence: _Optional[float] = ...) -> None: ...

class ProviderResultGroup(_message.Message):
    __slots__ = ("provider_id", "hits", "count", "degraded", "note", "latency_ms")
    PROVIDER_ID_FIELD_NUMBER: _ClassVar[int]
    HITS_FIELD_NUMBER: _ClassVar[int]
    COUNT_FIELD_NUMBER: _ClassVar[int]
    DEGRADED_FIELD_NUMBER: _ClassVar[int]
    NOTE_FIELD_NUMBER: _ClassVar[int]
    LATENCY_MS_FIELD_NUMBER: _ClassVar[int]
    provider_id: str
    hits: _containers.RepeatedCompositeFieldContainer[SearchHit]
    count: int
    degraded: bool
    note: str
    latency_ms: int
    def __init__(self, provider_id: _Optional[str] = ..., hits: _Optional[_Iterable[_Union[SearchHit, _Mapping]]] = ..., count: _Optional[int] = ..., degraded: _Optional[bool] = ..., note: _Optional[str] = ..., latency_ms: _Optional[int] = ...) -> None: ...

class QueryResponse(_message.Message):
    __slots__ = ("ranked", "groups", "corpora_searched", "routing_explanation", "reranked", "degraded", "latency_ms", "partial", "pending_providers")
    RANKED_FIELD_NUMBER: _ClassVar[int]
    GROUPS_FIELD_NUMBER: _ClassVar[int]
    CORPORA_SEARCHED_FIELD_NUMBER: _ClassVar[int]
    ROUTING_EXPLANATION_FIELD_NUMBER: _ClassVar[int]
    RERANKED_FIELD_NUMBER: _ClassVar[int]
    DEGRADED_FIELD_NUMBER: _ClassVar[int]
    LATENCY_MS_FIELD_NUMBER: _ClassVar[int]
    PARTIAL_FIELD_NUMBER: _ClassVar[int]
    PENDING_PROVIDERS_FIELD_NUMBER: _ClassVar[int]
    ranked: _containers.RepeatedCompositeFieldContainer[SearchHit]
    groups: _containers.RepeatedCompositeFieldContainer[ProviderResultGroup]
    corpora_searched: _containers.RepeatedScalarFieldContainer[str]
    routing_explanation: _containers.RepeatedScalarFieldContainer[str]
    reranked: bool
    degraded: bool
    latency_ms: int
    partial: bool
    pending_providers: int
    def __init__(self, ranked: _Optional[_Iterable[_Union[SearchHit, _Mapping]]] = ..., groups: _Optional[_Iterable[_Union[ProviderResultGroup, _Mapping]]] = ..., corpora_searched: _Optional[_Iterable[str]] = ..., routing_explanation: _Optional[_Iterable[str]] = ..., reranked: _Optional[bool] = ..., degraded: _Optional[bool] = ..., latency_ms: _Optional[int] = ..., partial: _Optional[bool] = ..., pending_providers: _Optional[int] = ...) -> None: ...

class StatusRequest(_message.Message):
    __slots__ = ()
    def __init__(self) -> None: ...

class RepromoteRequest(_message.Message):
    __slots__ = ("provider_id",)
    PROVIDER_ID_FIELD_NUMBER: _ClassVar[int]
    provider_id: str
    def __init__(self, provider_id: _Optional[str] = ...) -> None: ...

class RepromoteResponse(_message.Message):
    __slots__ = ("provider_id", "reset", "message")
    PROVIDER_ID_FIELD_NUMBER: _ClassVar[int]
    RESET_FIELD_NUMBER: _ClassVar[int]
    MESSAGE_FIELD_NUMBER: _ClassVar[int]
    provider_id: str
    reset: bool
    message: str
    def __init__(self, provider_id: _Optional[str] = ..., reset: _Optional[bool] = ..., message: _Optional[str] = ...) -> None: ...

class ProviderHealth(_message.Message):
    __slots__ = ("provider_id", "reachable", "point_count", "degraded", "demoted", "demotion_reason", "times_routed", "total_hits", "reachability", "index_age", "automatic_eligible", "automatic_exclusion_reason", "circuit_state", "quality_withheld", "quality_withheld_reason", "quality_evidence_run_id", "quality_gate_opted_out", "quality_gate_opt_out_reason", "last_indexed_at")
    PROVIDER_ID_FIELD_NUMBER: _ClassVar[int]
    REACHABLE_FIELD_NUMBER: _ClassVar[int]
    POINT_COUNT_FIELD_NUMBER: _ClassVar[int]
    DEGRADED_FIELD_NUMBER: _ClassVar[int]
    DEMOTED_FIELD_NUMBER: _ClassVar[int]
    DEMOTION_REASON_FIELD_NUMBER: _ClassVar[int]
    TIMES_ROUTED_FIELD_NUMBER: _ClassVar[int]
    TOTAL_HITS_FIELD_NUMBER: _ClassVar[int]
    REACHABILITY_FIELD_NUMBER: _ClassVar[int]
    INDEX_AGE_FIELD_NUMBER: _ClassVar[int]
    AUTOMATIC_ELIGIBLE_FIELD_NUMBER: _ClassVar[int]
    AUTOMATIC_EXCLUSION_REASON_FIELD_NUMBER: _ClassVar[int]
    CIRCUIT_STATE_FIELD_NUMBER: _ClassVar[int]
    QUALITY_WITHHELD_FIELD_NUMBER: _ClassVar[int]
    QUALITY_WITHHELD_REASON_FIELD_NUMBER: _ClassVar[int]
    QUALITY_EVIDENCE_RUN_ID_FIELD_NUMBER: _ClassVar[int]
    QUALITY_GATE_OPTED_OUT_FIELD_NUMBER: _ClassVar[int]
    QUALITY_GATE_OPT_OUT_REASON_FIELD_NUMBER: _ClassVar[int]
    LAST_INDEXED_AT_FIELD_NUMBER: _ClassVar[int]
    provider_id: str
    reachable: bool
    point_count: int
    degraded: bool
    demoted: bool
    demotion_reason: str
    times_routed: int
    total_hits: int
    reachability: str
    index_age: str
    automatic_eligible: bool
    automatic_exclusion_reason: str
    circuit_state: str
    quality_withheld: bool
    quality_withheld_reason: str
    quality_evidence_run_id: str
    quality_gate_opted_out: bool
    quality_gate_opt_out_reason: str
    last_indexed_at: _timestamp_pb2.Timestamp
    def __init__(self, provider_id: _Optional[str] = ..., reachable: _Optional[bool] = ..., point_count: _Optional[int] = ..., degraded: _Optional[bool] = ..., demoted: _Optional[bool] = ..., demotion_reason: _Optional[str] = ..., times_routed: _Optional[int] = ..., total_hits: _Optional[int] = ..., reachability: _Optional[str] = ..., index_age: _Optional[str] = ..., automatic_eligible: _Optional[bool] = ..., automatic_exclusion_reason: _Optional[str] = ..., circuit_state: _Optional[str] = ..., quality_withheld: _Optional[bool] = ..., quality_withheld_reason: _Optional[str] = ..., quality_evidence_run_id: _Optional[str] = ..., quality_gate_opted_out: _Optional[bool] = ..., quality_gate_opt_out_reason: _Optional[str] = ..., last_indexed_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ...) -> None: ...

class StatusResponse(_message.Message):
    __slots__ = ("providers", "classifier_available", "reranker_available", "circuit_open_share", "circuit_open_quorum", "federation_degraded")
    PROVIDERS_FIELD_NUMBER: _ClassVar[int]
    CLASSIFIER_AVAILABLE_FIELD_NUMBER: _ClassVar[int]
    RERANKER_AVAILABLE_FIELD_NUMBER: _ClassVar[int]
    CIRCUIT_OPEN_SHARE_FIELD_NUMBER: _ClassVar[int]
    CIRCUIT_OPEN_QUORUM_FIELD_NUMBER: _ClassVar[int]
    FEDERATION_DEGRADED_FIELD_NUMBER: _ClassVar[int]
    providers: _containers.RepeatedCompositeFieldContainer[ProviderHealth]
    classifier_available: bool
    reranker_available: bool
    circuit_open_share: float
    circuit_open_quorum: float
    federation_degraded: bool
    def __init__(self, providers: _Optional[_Iterable[_Union[ProviderHealth, _Mapping]]] = ..., classifier_available: _Optional[bool] = ..., reranker_available: _Optional[bool] = ..., circuit_open_share: _Optional[float] = ..., circuit_open_quorum: _Optional[float] = ..., federation_degraded: _Optional[bool] = ...) -> None: ...
