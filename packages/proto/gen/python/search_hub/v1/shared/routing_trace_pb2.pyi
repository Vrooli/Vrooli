from google.protobuf.internal import containers as _containers
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Iterable as _Iterable, Mapping as _Mapping
from typing import ClassVar as _ClassVar, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class ProviderRoutingEvidence(_message.Message):
    __slots__ = ("provider_id", "dense_rank", "dense_score", "lexical_rank", "lexical_score", "in_evidence_union", "cross_encoder_rank", "cross_encoder_score", "selected")
    PROVIDER_ID_FIELD_NUMBER: _ClassVar[int]
    DENSE_RANK_FIELD_NUMBER: _ClassVar[int]
    DENSE_SCORE_FIELD_NUMBER: _ClassVar[int]
    LEXICAL_RANK_FIELD_NUMBER: _ClassVar[int]
    LEXICAL_SCORE_FIELD_NUMBER: _ClassVar[int]
    IN_EVIDENCE_UNION_FIELD_NUMBER: _ClassVar[int]
    CROSS_ENCODER_RANK_FIELD_NUMBER: _ClassVar[int]
    CROSS_ENCODER_SCORE_FIELD_NUMBER: _ClassVar[int]
    SELECTED_FIELD_NUMBER: _ClassVar[int]
    provider_id: str
    dense_rank: int
    dense_score: float
    lexical_rank: int
    lexical_score: float
    in_evidence_union: bool
    cross_encoder_rank: int
    cross_encoder_score: float
    selected: bool
    def __init__(self, provider_id: _Optional[str] = ..., dense_rank: _Optional[int] = ..., dense_score: _Optional[float] = ..., lexical_rank: _Optional[int] = ..., lexical_score: _Optional[float] = ..., in_evidence_union: _Optional[bool] = ..., cross_encoder_rank: _Optional[int] = ..., cross_encoder_score: _Optional[float] = ..., selected: _Optional[bool] = ...) -> None: ...

class RoutingTrace(_message.Message):
    __slots__ = ("strategy_name", "index_status", "index_reason", "dense_top_provider_ids", "lexical_top_provider_ids", "candidates", "selected_provider_ids", "selection_reason", "returned_evidence", "unavailable_reason")
    STRATEGY_NAME_FIELD_NUMBER: _ClassVar[int]
    INDEX_STATUS_FIELD_NUMBER: _ClassVar[int]
    INDEX_REASON_FIELD_NUMBER: _ClassVar[int]
    DENSE_TOP_PROVIDER_IDS_FIELD_NUMBER: _ClassVar[int]
    LEXICAL_TOP_PROVIDER_IDS_FIELD_NUMBER: _ClassVar[int]
    CANDIDATES_FIELD_NUMBER: _ClassVar[int]
    SELECTED_PROVIDER_IDS_FIELD_NUMBER: _ClassVar[int]
    SELECTION_REASON_FIELD_NUMBER: _ClassVar[int]
    RETURNED_EVIDENCE_FIELD_NUMBER: _ClassVar[int]
    UNAVAILABLE_REASON_FIELD_NUMBER: _ClassVar[int]
    strategy_name: str
    index_status: str
    index_reason: str
    dense_top_provider_ids: _containers.RepeatedScalarFieldContainer[str]
    lexical_top_provider_ids: _containers.RepeatedScalarFieldContainer[str]
    candidates: _containers.RepeatedCompositeFieldContainer[ProviderRoutingEvidence]
    selected_provider_ids: _containers.RepeatedScalarFieldContainer[str]
    selection_reason: str
    returned_evidence: str
    unavailable_reason: str
    def __init__(self, strategy_name: _Optional[str] = ..., index_status: _Optional[str] = ..., index_reason: _Optional[str] = ..., dense_top_provider_ids: _Optional[_Iterable[str]] = ..., lexical_top_provider_ids: _Optional[_Iterable[str]] = ..., candidates: _Optional[_Iterable[_Union[ProviderRoutingEvidence, _Mapping]]] = ..., selected_provider_ids: _Optional[_Iterable[str]] = ..., selection_reason: _Optional[str] = ..., returned_evidence: _Optional[str] = ..., unavailable_reason: _Optional[str] = ...) -> None: ...
