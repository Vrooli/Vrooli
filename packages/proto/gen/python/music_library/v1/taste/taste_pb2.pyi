import datetime

from google.protobuf import timestamp_pb2 as _timestamp_pb2
from google.protobuf.internal import containers as _containers
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Iterable as _Iterable, Mapping as _Mapping
from typing import ClassVar as _ClassVar, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class TasteProfile(_message.Message):
    __slots__ = ("context", "model_version", "feature_space", "revision", "controls", "weights", "judgments", "updated_at")
    CONTEXT_FIELD_NUMBER: _ClassVar[int]
    MODEL_VERSION_FIELD_NUMBER: _ClassVar[int]
    FEATURE_SPACE_FIELD_NUMBER: _ClassVar[int]
    REVISION_FIELD_NUMBER: _ClassVar[int]
    CONTROLS_FIELD_NUMBER: _ClassVar[int]
    WEIGHTS_FIELD_NUMBER: _ClassVar[int]
    JUDGMENTS_FIELD_NUMBER: _ClassVar[int]
    UPDATED_AT_FIELD_NUMBER: _ClassVar[int]
    context: str
    model_version: str
    feature_space: str
    revision: int
    controls: _containers.RepeatedCompositeFieldContainer[TasteControl]
    weights: _containers.RepeatedCompositeFieldContainer[TasteWeight]
    judgments: _containers.RepeatedCompositeFieldContainer[PairwiseJudgment]
    updated_at: _timestamp_pb2.Timestamp
    def __init__(self, context: _Optional[str] = ..., model_version: _Optional[str] = ..., feature_space: _Optional[str] = ..., revision: _Optional[int] = ..., controls: _Optional[_Iterable[_Union[TasteControl, _Mapping]]] = ..., weights: _Optional[_Iterable[_Union[TasteWeight, _Mapping]]] = ..., judgments: _Optional[_Iterable[_Union[PairwiseJudgment, _Mapping]]] = ..., updated_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ...) -> None: ...

class TasteControl(_message.Message):
    __slots__ = ("attribute", "mode", "value")
    ATTRIBUTE_FIELD_NUMBER: _ClassVar[int]
    MODE_FIELD_NUMBER: _ClassVar[int]
    VALUE_FIELD_NUMBER: _ClassVar[int]
    attribute: str
    mode: str
    value: float
    def __init__(self, attribute: _Optional[str] = ..., mode: _Optional[str] = ..., value: _Optional[float] = ...) -> None: ...

class TasteWeight(_message.Message):
    __slots__ = ("attribute", "mean", "variance", "evidence_count")
    ATTRIBUTE_FIELD_NUMBER: _ClassVar[int]
    MEAN_FIELD_NUMBER: _ClassVar[int]
    VARIANCE_FIELD_NUMBER: _ClassVar[int]
    EVIDENCE_COUNT_FIELD_NUMBER: _ClassVar[int]
    attribute: str
    mean: float
    variance: float
    evidence_count: int
    def __init__(self, attribute: _Optional[str] = ..., mean: _Optional[float] = ..., variance: _Optional[float] = ..., evidence_count: _Optional[int] = ...) -> None: ...

class PairwiseJudgment(_message.Message):
    __slots__ = ("id", "context", "left_track_sha256", "right_track_sha256", "choice", "reason", "attribute_scope", "source", "reverses_judgment_id", "left_evidence", "right_evidence", "judged_at")
    ID_FIELD_NUMBER: _ClassVar[int]
    CONTEXT_FIELD_NUMBER: _ClassVar[int]
    LEFT_TRACK_SHA256_FIELD_NUMBER: _ClassVar[int]
    RIGHT_TRACK_SHA256_FIELD_NUMBER: _ClassVar[int]
    CHOICE_FIELD_NUMBER: _ClassVar[int]
    REASON_FIELD_NUMBER: _ClassVar[int]
    ATTRIBUTE_SCOPE_FIELD_NUMBER: _ClassVar[int]
    SOURCE_FIELD_NUMBER: _ClassVar[int]
    REVERSES_JUDGMENT_ID_FIELD_NUMBER: _ClassVar[int]
    LEFT_EVIDENCE_FIELD_NUMBER: _ClassVar[int]
    RIGHT_EVIDENCE_FIELD_NUMBER: _ClassVar[int]
    JUDGED_AT_FIELD_NUMBER: _ClassVar[int]
    id: str
    context: str
    left_track_sha256: str
    right_track_sha256: str
    choice: str
    reason: str
    attribute_scope: str
    source: str
    reverses_judgment_id: str
    left_evidence: _containers.RepeatedCompositeFieldContainer[TasteFeature]
    right_evidence: _containers.RepeatedCompositeFieldContainer[TasteFeature]
    judged_at: _timestamp_pb2.Timestamp
    def __init__(self, id: _Optional[str] = ..., context: _Optional[str] = ..., left_track_sha256: _Optional[str] = ..., right_track_sha256: _Optional[str] = ..., choice: _Optional[str] = ..., reason: _Optional[str] = ..., attribute_scope: _Optional[str] = ..., source: _Optional[str] = ..., reverses_judgment_id: _Optional[str] = ..., left_evidence: _Optional[_Iterable[_Union[TasteFeature, _Mapping]]] = ..., right_evidence: _Optional[_Iterable[_Union[TasteFeature, _Mapping]]] = ..., judged_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ...) -> None: ...

class TasteFeature(_message.Message):
    __slots__ = ("attribute", "value", "confidence", "evidence_source")
    ATTRIBUTE_FIELD_NUMBER: _ClassVar[int]
    VALUE_FIELD_NUMBER: _ClassVar[int]
    CONFIDENCE_FIELD_NUMBER: _ClassVar[int]
    EVIDENCE_SOURCE_FIELD_NUMBER: _ClassVar[int]
    attribute: str
    value: float
    confidence: float
    evidence_source: str
    def __init__(self, attribute: _Optional[str] = ..., value: _Optional[float] = ..., confidence: _Optional[float] = ..., evidence_source: _Optional[str] = ...) -> None: ...

class TasteCandidate(_message.Message):
    __slots__ = ("track_sha256", "title", "artist", "genre", "tags", "features")
    TRACK_SHA256_FIELD_NUMBER: _ClassVar[int]
    TITLE_FIELD_NUMBER: _ClassVar[int]
    ARTIST_FIELD_NUMBER: _ClassVar[int]
    GENRE_FIELD_NUMBER: _ClassVar[int]
    TAGS_FIELD_NUMBER: _ClassVar[int]
    FEATURES_FIELD_NUMBER: _ClassVar[int]
    track_sha256: str
    title: str
    artist: str
    genre: str
    tags: _containers.RepeatedScalarFieldContainer[str]
    features: _containers.RepeatedCompositeFieldContainer[TasteFeature]
    def __init__(self, track_sha256: _Optional[str] = ..., title: _Optional[str] = ..., artist: _Optional[str] = ..., genre: _Optional[str] = ..., tags: _Optional[_Iterable[str]] = ..., features: _Optional[_Iterable[_Union[TasteFeature, _Mapping]]] = ...) -> None: ...

class TasteContribution(_message.Message):
    __slots__ = ("attribute", "feature_value", "feature_confidence", "learned_weight", "contribution", "evidence_source")
    ATTRIBUTE_FIELD_NUMBER: _ClassVar[int]
    FEATURE_VALUE_FIELD_NUMBER: _ClassVar[int]
    FEATURE_CONFIDENCE_FIELD_NUMBER: _ClassVar[int]
    LEARNED_WEIGHT_FIELD_NUMBER: _ClassVar[int]
    CONTRIBUTION_FIELD_NUMBER: _ClassVar[int]
    EVIDENCE_SOURCE_FIELD_NUMBER: _ClassVar[int]
    attribute: str
    feature_value: float
    feature_confidence: float
    learned_weight: float
    contribution: float
    evidence_source: str
    def __init__(self, attribute: _Optional[str] = ..., feature_value: _Optional[float] = ..., feature_confidence: _Optional[float] = ..., learned_weight: _Optional[float] = ..., contribution: _Optional[float] = ..., evidence_source: _Optional[str] = ...) -> None: ...

class TasteRecommendation(_message.Message):
    __slots__ = ("candidate", "rank", "predicted_preference", "uncertainty", "exploration", "contributions", "selection_score", "exploration_bonus")
    CANDIDATE_FIELD_NUMBER: _ClassVar[int]
    RANK_FIELD_NUMBER: _ClassVar[int]
    PREDICTED_PREFERENCE_FIELD_NUMBER: _ClassVar[int]
    UNCERTAINTY_FIELD_NUMBER: _ClassVar[int]
    EXPLORATION_FIELD_NUMBER: _ClassVar[int]
    CONTRIBUTIONS_FIELD_NUMBER: _ClassVar[int]
    SELECTION_SCORE_FIELD_NUMBER: _ClassVar[int]
    EXPLORATION_BONUS_FIELD_NUMBER: _ClassVar[int]
    candidate: TasteCandidate
    rank: int
    predicted_preference: float
    uncertainty: float
    exploration: bool
    contributions: _containers.RepeatedCompositeFieldContainer[TasteContribution]
    selection_score: float
    exploration_bonus: float
    def __init__(self, candidate: _Optional[_Union[TasteCandidate, _Mapping]] = ..., rank: _Optional[int] = ..., predicted_preference: _Optional[float] = ..., uncertainty: _Optional[float] = ..., exploration: _Optional[bool] = ..., contributions: _Optional[_Iterable[_Union[TasteContribution, _Mapping]]] = ..., selection_score: _Optional[float] = ..., exploration_bonus: _Optional[float] = ...) -> None: ...

class GetProfileRequest(_message.Message):
    __slots__ = ("context",)
    CONTEXT_FIELD_NUMBER: _ClassVar[int]
    context: str
    def __init__(self, context: _Optional[str] = ...) -> None: ...

class GetProfileResponse(_message.Message):
    __slots__ = ("profile",)
    PROFILE_FIELD_NUMBER: _ClassVar[int]
    profile: TasteProfile
    def __init__(self, profile: _Optional[_Union[TasteProfile, _Mapping]] = ...) -> None: ...

class SetControlRequest(_message.Message):
    __slots__ = ("context", "control")
    CONTEXT_FIELD_NUMBER: _ClassVar[int]
    CONTROL_FIELD_NUMBER: _ClassVar[int]
    context: str
    control: TasteControl
    def __init__(self, context: _Optional[str] = ..., control: _Optional[_Union[TasteControl, _Mapping]] = ...) -> None: ...

class SetControlResponse(_message.Message):
    __slots__ = ("profile",)
    PROFILE_FIELD_NUMBER: _ClassVar[int]
    profile: TasteProfile
    def __init__(self, profile: _Optional[_Union[TasteProfile, _Mapping]] = ...) -> None: ...

class SelectPairRequest(_message.Message):
    __slots__ = ("context", "random_baseline", "random_seed")
    CONTEXT_FIELD_NUMBER: _ClassVar[int]
    RANDOM_BASELINE_FIELD_NUMBER: _ClassVar[int]
    RANDOM_SEED_FIELD_NUMBER: _ClassVar[int]
    context: str
    random_baseline: bool
    random_seed: int
    def __init__(self, context: _Optional[str] = ..., random_baseline: _Optional[bool] = ..., random_seed: _Optional[int] = ...) -> None: ...

class SelectPairResponse(_message.Message):
    __slots__ = ("left", "right", "expected_information", "available_candidate_count", "insufficient_evidence")
    LEFT_FIELD_NUMBER: _ClassVar[int]
    RIGHT_FIELD_NUMBER: _ClassVar[int]
    EXPECTED_INFORMATION_FIELD_NUMBER: _ClassVar[int]
    AVAILABLE_CANDIDATE_COUNT_FIELD_NUMBER: _ClassVar[int]
    INSUFFICIENT_EVIDENCE_FIELD_NUMBER: _ClassVar[int]
    left: TasteCandidate
    right: TasteCandidate
    expected_information: float
    available_candidate_count: int
    insufficient_evidence: bool
    def __init__(self, left: _Optional[_Union[TasteCandidate, _Mapping]] = ..., right: _Optional[_Union[TasteCandidate, _Mapping]] = ..., expected_information: _Optional[float] = ..., available_candidate_count: _Optional[int] = ..., insufficient_evidence: _Optional[bool] = ...) -> None: ...

class RecordJudgmentRequest(_message.Message):
    __slots__ = ("context", "left_track_sha256", "right_track_sha256", "choice", "reason", "attribute_scope", "reverses_judgment_id")
    CONTEXT_FIELD_NUMBER: _ClassVar[int]
    LEFT_TRACK_SHA256_FIELD_NUMBER: _ClassVar[int]
    RIGHT_TRACK_SHA256_FIELD_NUMBER: _ClassVar[int]
    CHOICE_FIELD_NUMBER: _ClassVar[int]
    REASON_FIELD_NUMBER: _ClassVar[int]
    ATTRIBUTE_SCOPE_FIELD_NUMBER: _ClassVar[int]
    REVERSES_JUDGMENT_ID_FIELD_NUMBER: _ClassVar[int]
    context: str
    left_track_sha256: str
    right_track_sha256: str
    choice: str
    reason: str
    attribute_scope: str
    reverses_judgment_id: str
    def __init__(self, context: _Optional[str] = ..., left_track_sha256: _Optional[str] = ..., right_track_sha256: _Optional[str] = ..., choice: _Optional[str] = ..., reason: _Optional[str] = ..., attribute_scope: _Optional[str] = ..., reverses_judgment_id: _Optional[str] = ...) -> None: ...

class RecordJudgmentResponse(_message.Message):
    __slots__ = ("profile",)
    PROFILE_FIELD_NUMBER: _ClassVar[int]
    profile: TasteProfile
    def __init__(self, profile: _Optional[_Union[TasteProfile, _Mapping]] = ...) -> None: ...

class RecommendRequest(_message.Message):
    __slots__ = ("context", "seed_track_sha256", "limit")
    CONTEXT_FIELD_NUMBER: _ClassVar[int]
    SEED_TRACK_SHA256_FIELD_NUMBER: _ClassVar[int]
    LIMIT_FIELD_NUMBER: _ClassVar[int]
    context: str
    seed_track_sha256: str
    limit: int
    def __init__(self, context: _Optional[str] = ..., seed_track_sha256: _Optional[str] = ..., limit: _Optional[int] = ...) -> None: ...

class RecommendResponse(_message.Message):
    __slots__ = ("seed", "recommendations", "evidence_state", "mq2_brief", "chosen_features", "avoided_features", "provenance")
    SEED_FIELD_NUMBER: _ClassVar[int]
    RECOMMENDATIONS_FIELD_NUMBER: _ClassVar[int]
    EVIDENCE_STATE_FIELD_NUMBER: _ClassVar[int]
    MQ2_BRIEF_FIELD_NUMBER: _ClassVar[int]
    CHOSEN_FEATURES_FIELD_NUMBER: _ClassVar[int]
    AVOIDED_FEATURES_FIELD_NUMBER: _ClassVar[int]
    PROVENANCE_FIELD_NUMBER: _ClassVar[int]
    seed: TasteCandidate
    recommendations: _containers.RepeatedCompositeFieldContainer[TasteRecommendation]
    evidence_state: str
    mq2_brief: str
    chosen_features: _containers.RepeatedScalarFieldContainer[str]
    avoided_features: _containers.RepeatedScalarFieldContainer[str]
    provenance: _containers.RepeatedScalarFieldContainer[str]
    def __init__(self, seed: _Optional[_Union[TasteCandidate, _Mapping]] = ..., recommendations: _Optional[_Iterable[_Union[TasteRecommendation, _Mapping]]] = ..., evidence_state: _Optional[str] = ..., mq2_brief: _Optional[str] = ..., chosen_features: _Optional[_Iterable[str]] = ..., avoided_features: _Optional[_Iterable[str]] = ..., provenance: _Optional[_Iterable[str]] = ...) -> None: ...
