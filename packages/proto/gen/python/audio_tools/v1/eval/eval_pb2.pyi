from google.protobuf.internal import containers as _containers
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Iterable as _Iterable, Mapping as _Mapping
from typing import ClassVar as _ClassVar, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class EvalStrategy(_message.Message):
    __slots__ = ("kind", "label", "overlap_max_stall_rejects", "overlap_window_ms", "overlap_commit_runs", "vad_silence_ms", "overlap_max_window_ms")
    KIND_FIELD_NUMBER: _ClassVar[int]
    LABEL_FIELD_NUMBER: _ClassVar[int]
    OVERLAP_MAX_STALL_REJECTS_FIELD_NUMBER: _ClassVar[int]
    OVERLAP_WINDOW_MS_FIELD_NUMBER: _ClassVar[int]
    OVERLAP_COMMIT_RUNS_FIELD_NUMBER: _ClassVar[int]
    VAD_SILENCE_MS_FIELD_NUMBER: _ClassVar[int]
    OVERLAP_MAX_WINDOW_MS_FIELD_NUMBER: _ClassVar[int]
    kind: str
    label: str
    overlap_max_stall_rejects: int
    overlap_window_ms: int
    overlap_commit_runs: int
    vad_silence_ms: int
    overlap_max_window_ms: int
    def __init__(self, kind: _Optional[str] = ..., label: _Optional[str] = ..., overlap_max_stall_rejects: _Optional[int] = ..., overlap_window_ms: _Optional[int] = ..., overlap_commit_runs: _Optional[int] = ..., vad_silence_ms: _Optional[int] = ..., overlap_max_window_ms: _Optional[int] = ...) -> None: ...

class EvalReport(_message.Message):
    __slots__ = ("per_strategy", "quality_measured", "latency_measured", "summary", "warnings", "normalization_policy", "latency_honesty", "promotion_verdicts")
    PER_STRATEGY_FIELD_NUMBER: _ClassVar[int]
    QUALITY_MEASURED_FIELD_NUMBER: _ClassVar[int]
    LATENCY_MEASURED_FIELD_NUMBER: _ClassVar[int]
    SUMMARY_FIELD_NUMBER: _ClassVar[int]
    WARNINGS_FIELD_NUMBER: _ClassVar[int]
    NORMALIZATION_POLICY_FIELD_NUMBER: _ClassVar[int]
    LATENCY_HONESTY_FIELD_NUMBER: _ClassVar[int]
    PROMOTION_VERDICTS_FIELD_NUMBER: _ClassVar[int]
    per_strategy: _containers.RepeatedCompositeFieldContainer[StrategyReport]
    quality_measured: bool
    latency_measured: bool
    summary: EvalReportSummary
    warnings: _containers.RepeatedCompositeFieldContainer[ReportWarning]
    normalization_policy: NormalizationPolicy
    latency_honesty: str
    promotion_verdicts: _containers.RepeatedCompositeFieldContainer[PromotionVerdict]
    def __init__(self, per_strategy: _Optional[_Iterable[_Union[StrategyReport, _Mapping]]] = ..., quality_measured: _Optional[bool] = ..., latency_measured: _Optional[bool] = ..., summary: _Optional[_Union[EvalReportSummary, _Mapping]] = ..., warnings: _Optional[_Iterable[_Union[ReportWarning, _Mapping]]] = ..., normalization_policy: _Optional[_Union[NormalizationPolicy, _Mapping]] = ..., latency_honesty: _Optional[str] = ..., promotion_verdicts: _Optional[_Iterable[_Union[PromotionVerdict, _Mapping]]] = ...) -> None: ...

class PromotionVerdict(_message.Message):
    __slots__ = ("engine_id", "stable", "reasons", "strategy", "policy_profile", "model_id")
    ENGINE_ID_FIELD_NUMBER: _ClassVar[int]
    STABLE_FIELD_NUMBER: _ClassVar[int]
    REASONS_FIELD_NUMBER: _ClassVar[int]
    STRATEGY_FIELD_NUMBER: _ClassVar[int]
    POLICY_PROFILE_FIELD_NUMBER: _ClassVar[int]
    MODEL_ID_FIELD_NUMBER: _ClassVar[int]
    engine_id: str
    stable: bool
    reasons: _containers.RepeatedScalarFieldContainer[str]
    strategy: str
    policy_profile: str
    model_id: str
    def __init__(self, engine_id: _Optional[str] = ..., stable: _Optional[bool] = ..., reasons: _Optional[_Iterable[str]] = ..., strategy: _Optional[str] = ..., policy_profile: _Optional[str] = ..., model_id: _Optional[str] = ...) -> None: ...

class StrategyReport(_message.Message):
    __slots__ = ("strategy", "label", "wer", "substitutions", "insertions", "deletions", "ref_words", "whisper_calls", "whisper_audio_seconds", "rtf", "finalization_latency_p50_ms", "finalization_latency_p95_ms", "partial_revisions", "per_clip", "wer_delta_vs_winner", "p95_delta_ms_vs_winner", "call_multiplier_vs_winner", "verdict", "reasons", "warnings", "safety", "stage_attribution", "length_curves", "commit_count", "speaker_rejection_count", "scaling", "engine_id", "policy_profile", "replay_lane", "fault_profile", "model_id")
    STRATEGY_FIELD_NUMBER: _ClassVar[int]
    LABEL_FIELD_NUMBER: _ClassVar[int]
    WER_FIELD_NUMBER: _ClassVar[int]
    SUBSTITUTIONS_FIELD_NUMBER: _ClassVar[int]
    INSERTIONS_FIELD_NUMBER: _ClassVar[int]
    DELETIONS_FIELD_NUMBER: _ClassVar[int]
    REF_WORDS_FIELD_NUMBER: _ClassVar[int]
    WHISPER_CALLS_FIELD_NUMBER: _ClassVar[int]
    WHISPER_AUDIO_SECONDS_FIELD_NUMBER: _ClassVar[int]
    RTF_FIELD_NUMBER: _ClassVar[int]
    FINALIZATION_LATENCY_P50_MS_FIELD_NUMBER: _ClassVar[int]
    FINALIZATION_LATENCY_P95_MS_FIELD_NUMBER: _ClassVar[int]
    PARTIAL_REVISIONS_FIELD_NUMBER: _ClassVar[int]
    PER_CLIP_FIELD_NUMBER: _ClassVar[int]
    WER_DELTA_VS_WINNER_FIELD_NUMBER: _ClassVar[int]
    P95_DELTA_MS_VS_WINNER_FIELD_NUMBER: _ClassVar[int]
    CALL_MULTIPLIER_VS_WINNER_FIELD_NUMBER: _ClassVar[int]
    VERDICT_FIELD_NUMBER: _ClassVar[int]
    REASONS_FIELD_NUMBER: _ClassVar[int]
    WARNINGS_FIELD_NUMBER: _ClassVar[int]
    SAFETY_FIELD_NUMBER: _ClassVar[int]
    STAGE_ATTRIBUTION_FIELD_NUMBER: _ClassVar[int]
    LENGTH_CURVES_FIELD_NUMBER: _ClassVar[int]
    COMMIT_COUNT_FIELD_NUMBER: _ClassVar[int]
    SPEAKER_REJECTION_COUNT_FIELD_NUMBER: _ClassVar[int]
    SCALING_FIELD_NUMBER: _ClassVar[int]
    ENGINE_ID_FIELD_NUMBER: _ClassVar[int]
    POLICY_PROFILE_FIELD_NUMBER: _ClassVar[int]
    REPLAY_LANE_FIELD_NUMBER: _ClassVar[int]
    FAULT_PROFILE_FIELD_NUMBER: _ClassVar[int]
    MODEL_ID_FIELD_NUMBER: _ClassVar[int]
    strategy: str
    label: str
    wer: float
    substitutions: int
    insertions: int
    deletions: int
    ref_words: int
    whisper_calls: int
    whisper_audio_seconds: float
    rtf: float
    finalization_latency_p50_ms: float
    finalization_latency_p95_ms: float
    partial_revisions: int
    per_clip: _containers.RepeatedCompositeFieldContainer[ClipReport]
    wer_delta_vs_winner: float
    p95_delta_ms_vs_winner: float
    call_multiplier_vs_winner: float
    verdict: str
    reasons: _containers.RepeatedScalarFieldContainer[str]
    warnings: _containers.RepeatedCompositeFieldContainer[ReportWarning]
    safety: SafetyGateReport
    stage_attribution: StageAttribution
    length_curves: _containers.RepeatedCompositeFieldContainer[LengthBucketCurve]
    commit_count: int
    speaker_rejection_count: int
    scaling: ScalingAnalysis
    engine_id: str
    policy_profile: str
    replay_lane: str
    fault_profile: str
    model_id: str
    def __init__(self, strategy: _Optional[str] = ..., label: _Optional[str] = ..., wer: _Optional[float] = ..., substitutions: _Optional[int] = ..., insertions: _Optional[int] = ..., deletions: _Optional[int] = ..., ref_words: _Optional[int] = ..., whisper_calls: _Optional[int] = ..., whisper_audio_seconds: _Optional[float] = ..., rtf: _Optional[float] = ..., finalization_latency_p50_ms: _Optional[float] = ..., finalization_latency_p95_ms: _Optional[float] = ..., partial_revisions: _Optional[int] = ..., per_clip: _Optional[_Iterable[_Union[ClipReport, _Mapping]]] = ..., wer_delta_vs_winner: _Optional[float] = ..., p95_delta_ms_vs_winner: _Optional[float] = ..., call_multiplier_vs_winner: _Optional[float] = ..., verdict: _Optional[str] = ..., reasons: _Optional[_Iterable[str]] = ..., warnings: _Optional[_Iterable[_Union[ReportWarning, _Mapping]]] = ..., safety: _Optional[_Union[SafetyGateReport, _Mapping]] = ..., stage_attribution: _Optional[_Union[StageAttribution, _Mapping]] = ..., length_curves: _Optional[_Iterable[_Union[LengthBucketCurve, _Mapping]]] = ..., commit_count: _Optional[int] = ..., speaker_rejection_count: _Optional[int] = ..., scaling: _Optional[_Union[ScalingAnalysis, _Mapping]] = ..., engine_id: _Optional[str] = ..., policy_profile: _Optional[str] = ..., replay_lane: _Optional[str] = ..., fault_profile: _Optional[str] = ..., model_id: _Optional[str] = ...) -> None: ...

class ClipReport(_message.Message):
    __slots__ = ("clip_id", "reference", "hypothesis", "wer", "whisper_calls", "whisper_audio_seconds", "rtf", "segment_count", "partial_revisions", "finalization_latency_p50_ms", "finalization_latency_p95_ms", "error", "substitutions", "insertions", "deletions", "ref_words", "hyp_words", "normalized_reference", "normalized_hypothesis", "edit_operations", "commit_timeline", "time_to_first_commit_ms", "commit_count", "speaker_rejection_count", "audio_duration_ms", "safety")
    CLIP_ID_FIELD_NUMBER: _ClassVar[int]
    REFERENCE_FIELD_NUMBER: _ClassVar[int]
    HYPOTHESIS_FIELD_NUMBER: _ClassVar[int]
    WER_FIELD_NUMBER: _ClassVar[int]
    WHISPER_CALLS_FIELD_NUMBER: _ClassVar[int]
    WHISPER_AUDIO_SECONDS_FIELD_NUMBER: _ClassVar[int]
    RTF_FIELD_NUMBER: _ClassVar[int]
    SEGMENT_COUNT_FIELD_NUMBER: _ClassVar[int]
    PARTIAL_REVISIONS_FIELD_NUMBER: _ClassVar[int]
    FINALIZATION_LATENCY_P50_MS_FIELD_NUMBER: _ClassVar[int]
    FINALIZATION_LATENCY_P95_MS_FIELD_NUMBER: _ClassVar[int]
    ERROR_FIELD_NUMBER: _ClassVar[int]
    SUBSTITUTIONS_FIELD_NUMBER: _ClassVar[int]
    INSERTIONS_FIELD_NUMBER: _ClassVar[int]
    DELETIONS_FIELD_NUMBER: _ClassVar[int]
    REF_WORDS_FIELD_NUMBER: _ClassVar[int]
    HYP_WORDS_FIELD_NUMBER: _ClassVar[int]
    NORMALIZED_REFERENCE_FIELD_NUMBER: _ClassVar[int]
    NORMALIZED_HYPOTHESIS_FIELD_NUMBER: _ClassVar[int]
    EDIT_OPERATIONS_FIELD_NUMBER: _ClassVar[int]
    COMMIT_TIMELINE_FIELD_NUMBER: _ClassVar[int]
    TIME_TO_FIRST_COMMIT_MS_FIELD_NUMBER: _ClassVar[int]
    COMMIT_COUNT_FIELD_NUMBER: _ClassVar[int]
    SPEAKER_REJECTION_COUNT_FIELD_NUMBER: _ClassVar[int]
    AUDIO_DURATION_MS_FIELD_NUMBER: _ClassVar[int]
    SAFETY_FIELD_NUMBER: _ClassVar[int]
    clip_id: str
    reference: str
    hypothesis: str
    wer: float
    whisper_calls: int
    whisper_audio_seconds: float
    rtf: float
    segment_count: int
    partial_revisions: int
    finalization_latency_p50_ms: float
    finalization_latency_p95_ms: float
    error: str
    substitutions: int
    insertions: int
    deletions: int
    ref_words: int
    hyp_words: int
    normalized_reference: str
    normalized_hypothesis: str
    edit_operations: _containers.RepeatedCompositeFieldContainer[EditOperation]
    commit_timeline: _containers.RepeatedCompositeFieldContainer[CommitState]
    time_to_first_commit_ms: float
    commit_count: int
    speaker_rejection_count: int
    audio_duration_ms: int
    safety: SafetyGateReport
    def __init__(self, clip_id: _Optional[str] = ..., reference: _Optional[str] = ..., hypothesis: _Optional[str] = ..., wer: _Optional[float] = ..., whisper_calls: _Optional[int] = ..., whisper_audio_seconds: _Optional[float] = ..., rtf: _Optional[float] = ..., segment_count: _Optional[int] = ..., partial_revisions: _Optional[int] = ..., finalization_latency_p50_ms: _Optional[float] = ..., finalization_latency_p95_ms: _Optional[float] = ..., error: _Optional[str] = ..., substitutions: _Optional[int] = ..., insertions: _Optional[int] = ..., deletions: _Optional[int] = ..., ref_words: _Optional[int] = ..., hyp_words: _Optional[int] = ..., normalized_reference: _Optional[str] = ..., normalized_hypothesis: _Optional[str] = ..., edit_operations: _Optional[_Iterable[_Union[EditOperation, _Mapping]]] = ..., commit_timeline: _Optional[_Iterable[_Union[CommitState, _Mapping]]] = ..., time_to_first_commit_ms: _Optional[float] = ..., commit_count: _Optional[int] = ..., speaker_rejection_count: _Optional[int] = ..., audio_duration_ms: _Optional[int] = ..., safety: _Optional[_Union[SafetyGateReport, _Mapping]] = ...) -> None: ...

class EvalReportSummary(_message.Message):
    __slots__ = ("winner_strategy", "winner_label", "recommendation", "confidence", "reasons", "confidence_notes")
    WINNER_STRATEGY_FIELD_NUMBER: _ClassVar[int]
    WINNER_LABEL_FIELD_NUMBER: _ClassVar[int]
    RECOMMENDATION_FIELD_NUMBER: _ClassVar[int]
    CONFIDENCE_FIELD_NUMBER: _ClassVar[int]
    REASONS_FIELD_NUMBER: _ClassVar[int]
    CONFIDENCE_NOTES_FIELD_NUMBER: _ClassVar[int]
    winner_strategy: str
    winner_label: str
    recommendation: str
    confidence: str
    reasons: _containers.RepeatedScalarFieldContainer[str]
    confidence_notes: _containers.RepeatedScalarFieldContainer[str]
    def __init__(self, winner_strategy: _Optional[str] = ..., winner_label: _Optional[str] = ..., recommendation: _Optional[str] = ..., confidence: _Optional[str] = ..., reasons: _Optional[_Iterable[str]] = ..., confidence_notes: _Optional[_Iterable[str]] = ...) -> None: ...

class ReportWarning(_message.Message):
    __slots__ = ("code", "message", "severity")
    CODE_FIELD_NUMBER: _ClassVar[int]
    MESSAGE_FIELD_NUMBER: _ClassVar[int]
    SEVERITY_FIELD_NUMBER: _ClassVar[int]
    code: str
    message: str
    severity: str
    def __init__(self, code: _Optional[str] = ..., message: _Optional[str] = ..., severity: _Optional[str] = ...) -> None: ...

class NormalizationPolicy(_message.Message):
    __slots__ = ("wer_policy", "overlap_agreement_policy")
    WER_POLICY_FIELD_NUMBER: _ClassVar[int]
    OVERLAP_AGREEMENT_POLICY_FIELD_NUMBER: _ClassVar[int]
    wer_policy: str
    overlap_agreement_policy: str
    def __init__(self, wer_policy: _Optional[str] = ..., overlap_agreement_policy: _Optional[str] = ...) -> None: ...

class EditOperation(_message.Message):
    __slots__ = ("kind", "reference_token", "hypothesis_token", "reference_index", "hypothesis_index")
    KIND_FIELD_NUMBER: _ClassVar[int]
    REFERENCE_TOKEN_FIELD_NUMBER: _ClassVar[int]
    HYPOTHESIS_TOKEN_FIELD_NUMBER: _ClassVar[int]
    REFERENCE_INDEX_FIELD_NUMBER: _ClassVar[int]
    HYPOTHESIS_INDEX_FIELD_NUMBER: _ClassVar[int]
    kind: str
    reference_token: str
    hypothesis_token: str
    reference_index: int
    hypothesis_index: int
    def __init__(self, kind: _Optional[str] = ..., reference_token: _Optional[str] = ..., hypothesis_token: _Optional[str] = ..., reference_index: _Optional[int] = ..., hypothesis_index: _Optional[int] = ...) -> None: ...

class CommitState(_message.Message):
    __slots__ = ("text", "at_ms", "audio_end_ms")
    TEXT_FIELD_NUMBER: _ClassVar[int]
    AT_MS_FIELD_NUMBER: _ClassVar[int]
    AUDIO_END_MS_FIELD_NUMBER: _ClassVar[int]
    text: str
    at_ms: int
    audio_end_ms: int
    def __init__(self, text: _Optional[str] = ..., at_ms: _Optional[int] = ..., audio_end_ms: _Optional[int] = ...) -> None: ...

class RetractionEvent(_message.Message):
    __slots__ = ("previous_text", "current_text", "at_ms")
    PREVIOUS_TEXT_FIELD_NUMBER: _ClassVar[int]
    CURRENT_TEXT_FIELD_NUMBER: _ClassVar[int]
    AT_MS_FIELD_NUMBER: _ClassVar[int]
    previous_text: str
    current_text: str
    at_ms: int
    def __init__(self, previous_text: _Optional[str] = ..., current_text: _Optional[str] = ..., at_ms: _Optional[int] = ...) -> None: ...

class SafetyGateReport(_message.Message):
    __slots__ = ("passed", "retraction_free", "dropped_span_free", "retraction_events", "max_dropped_span_words", "dropped_span_threshold_words", "reasons")
    PASSED_FIELD_NUMBER: _ClassVar[int]
    RETRACTION_FREE_FIELD_NUMBER: _ClassVar[int]
    DROPPED_SPAN_FREE_FIELD_NUMBER: _ClassVar[int]
    RETRACTION_EVENTS_FIELD_NUMBER: _ClassVar[int]
    MAX_DROPPED_SPAN_WORDS_FIELD_NUMBER: _ClassVar[int]
    DROPPED_SPAN_THRESHOLD_WORDS_FIELD_NUMBER: _ClassVar[int]
    REASONS_FIELD_NUMBER: _ClassVar[int]
    passed: bool
    retraction_free: bool
    dropped_span_free: bool
    retraction_events: _containers.RepeatedCompositeFieldContainer[RetractionEvent]
    max_dropped_span_words: int
    dropped_span_threshold_words: int
    reasons: _containers.RepeatedScalarFieldContainer[str]
    def __init__(self, passed: _Optional[bool] = ..., retraction_free: _Optional[bool] = ..., dropped_span_free: _Optional[bool] = ..., retraction_events: _Optional[_Iterable[_Union[RetractionEvent, _Mapping]]] = ..., max_dropped_span_words: _Optional[int] = ..., dropped_span_threshold_words: _Optional[int] = ..., reasons: _Optional[_Iterable[str]] = ...) -> None: ...

class StageAttribution(_message.Message):
    __slots__ = ("ingress_lost_words", "strategy_lost_words", "egress_lost_words", "egress_reject_events", "notes")
    INGRESS_LOST_WORDS_FIELD_NUMBER: _ClassVar[int]
    STRATEGY_LOST_WORDS_FIELD_NUMBER: _ClassVar[int]
    EGRESS_LOST_WORDS_FIELD_NUMBER: _ClassVar[int]
    EGRESS_REJECT_EVENTS_FIELD_NUMBER: _ClassVar[int]
    NOTES_FIELD_NUMBER: _ClassVar[int]
    ingress_lost_words: int
    strategy_lost_words: int
    egress_lost_words: int
    egress_reject_events: int
    notes: _containers.RepeatedScalarFieldContainer[str]
    def __init__(self, ingress_lost_words: _Optional[int] = ..., strategy_lost_words: _Optional[int] = ..., egress_lost_words: _Optional[int] = ..., egress_reject_events: _Optional[int] = ..., notes: _Optional[_Iterable[str]] = ...) -> None: ...

class LengthBucketCurve(_message.Message):
    __slots__ = ("bucket", "min_duration_ms", "max_duration_ms", "clip_count", "wer", "finalization_latency_p95_ms", "mean_time_to_first_commit_ms", "max_dropped_span_words")
    BUCKET_FIELD_NUMBER: _ClassVar[int]
    MIN_DURATION_MS_FIELD_NUMBER: _ClassVar[int]
    MAX_DURATION_MS_FIELD_NUMBER: _ClassVar[int]
    CLIP_COUNT_FIELD_NUMBER: _ClassVar[int]
    WER_FIELD_NUMBER: _ClassVar[int]
    FINALIZATION_LATENCY_P95_MS_FIELD_NUMBER: _ClassVar[int]
    MEAN_TIME_TO_FIRST_COMMIT_MS_FIELD_NUMBER: _ClassVar[int]
    MAX_DROPPED_SPAN_WORDS_FIELD_NUMBER: _ClassVar[int]
    bucket: str
    min_duration_ms: int
    max_duration_ms: int
    clip_count: int
    wer: float
    finalization_latency_p95_ms: float
    mean_time_to_first_commit_ms: float
    max_dropped_span_words: int
    def __init__(self, bucket: _Optional[str] = ..., min_duration_ms: _Optional[int] = ..., max_duration_ms: _Optional[int] = ..., clip_count: _Optional[int] = ..., wer: _Optional[float] = ..., finalization_latency_p95_ms: _Optional[float] = ..., mean_time_to_first_commit_ms: _Optional[float] = ..., max_dropped_span_words: _Optional[int] = ...) -> None: ...

class ScalingPoint(_message.Message):
    __slots__ = ("clip_id", "target_duration_ms", "realized_duration_ms", "wer", "finalization_latency_p50_ms", "finalization_latency_p95_ms", "finalization_latency_sample_count", "time_to_first_commit_ms", "commit_count", "partial_revisions", "max_dropped_span_words", "whisper_calls", "whisper_audio_seconds", "provider_latency_ms", "rtf")
    CLIP_ID_FIELD_NUMBER: _ClassVar[int]
    TARGET_DURATION_MS_FIELD_NUMBER: _ClassVar[int]
    REALIZED_DURATION_MS_FIELD_NUMBER: _ClassVar[int]
    WER_FIELD_NUMBER: _ClassVar[int]
    FINALIZATION_LATENCY_P50_MS_FIELD_NUMBER: _ClassVar[int]
    FINALIZATION_LATENCY_P95_MS_FIELD_NUMBER: _ClassVar[int]
    FINALIZATION_LATENCY_SAMPLE_COUNT_FIELD_NUMBER: _ClassVar[int]
    TIME_TO_FIRST_COMMIT_MS_FIELD_NUMBER: _ClassVar[int]
    COMMIT_COUNT_FIELD_NUMBER: _ClassVar[int]
    PARTIAL_REVISIONS_FIELD_NUMBER: _ClassVar[int]
    MAX_DROPPED_SPAN_WORDS_FIELD_NUMBER: _ClassVar[int]
    WHISPER_CALLS_FIELD_NUMBER: _ClassVar[int]
    WHISPER_AUDIO_SECONDS_FIELD_NUMBER: _ClassVar[int]
    PROVIDER_LATENCY_MS_FIELD_NUMBER: _ClassVar[int]
    RTF_FIELD_NUMBER: _ClassVar[int]
    clip_id: str
    target_duration_ms: int
    realized_duration_ms: int
    wer: float
    finalization_latency_p50_ms: float
    finalization_latency_p95_ms: float
    finalization_latency_sample_count: int
    time_to_first_commit_ms: float
    commit_count: int
    partial_revisions: int
    max_dropped_span_words: int
    whisper_calls: int
    whisper_audio_seconds: float
    provider_latency_ms: float
    rtf: float
    def __init__(self, clip_id: _Optional[str] = ..., target_duration_ms: _Optional[int] = ..., realized_duration_ms: _Optional[int] = ..., wer: _Optional[float] = ..., finalization_latency_p50_ms: _Optional[float] = ..., finalization_latency_p95_ms: _Optional[float] = ..., finalization_latency_sample_count: _Optional[int] = ..., time_to_first_commit_ms: _Optional[float] = ..., commit_count: _Optional[int] = ..., partial_revisions: _Optional[int] = ..., max_dropped_span_words: _Optional[int] = ..., whisper_calls: _Optional[int] = ..., whisper_audio_seconds: _Optional[float] = ..., provider_latency_ms: _Optional[float] = ..., rtf: _Optional[float] = ...) -> None: ...

class ScalingModelFit(_message.Message):
    __slots__ = ("metric", "model", "slope_per_second", "intercept", "r_squared", "sample_count", "reason", "exponent", "exponent_r_squared", "unit")
    METRIC_FIELD_NUMBER: _ClassVar[int]
    MODEL_FIELD_NUMBER: _ClassVar[int]
    SLOPE_PER_SECOND_FIELD_NUMBER: _ClassVar[int]
    INTERCEPT_FIELD_NUMBER: _ClassVar[int]
    R_SQUARED_FIELD_NUMBER: _ClassVar[int]
    SAMPLE_COUNT_FIELD_NUMBER: _ClassVar[int]
    REASON_FIELD_NUMBER: _ClassVar[int]
    EXPONENT_FIELD_NUMBER: _ClassVar[int]
    EXPONENT_R_SQUARED_FIELD_NUMBER: _ClassVar[int]
    UNIT_FIELD_NUMBER: _ClassVar[int]
    metric: str
    model: str
    slope_per_second: float
    intercept: float
    r_squared: float
    sample_count: int
    reason: str
    exponent: float
    exponent_r_squared: float
    unit: str
    def __init__(self, metric: _Optional[str] = ..., model: _Optional[str] = ..., slope_per_second: _Optional[float] = ..., intercept: _Optional[float] = ..., r_squared: _Optional[float] = ..., sample_count: _Optional[int] = ..., reason: _Optional[str] = ..., exponent: _Optional[float] = ..., exponent_r_squared: _Optional[float] = ..., unit: _Optional[str] = ...) -> None: ...

class ScalingAnalysis(_message.Message):
    __slots__ = ("points", "latency_classification", "compute_classification", "confidence", "reasons", "warnings", "latency_fit", "compute_fit", "metric_fits")
    POINTS_FIELD_NUMBER: _ClassVar[int]
    LATENCY_CLASSIFICATION_FIELD_NUMBER: _ClassVar[int]
    COMPUTE_CLASSIFICATION_FIELD_NUMBER: _ClassVar[int]
    CONFIDENCE_FIELD_NUMBER: _ClassVar[int]
    REASONS_FIELD_NUMBER: _ClassVar[int]
    WARNINGS_FIELD_NUMBER: _ClassVar[int]
    LATENCY_FIT_FIELD_NUMBER: _ClassVar[int]
    COMPUTE_FIT_FIELD_NUMBER: _ClassVar[int]
    METRIC_FITS_FIELD_NUMBER: _ClassVar[int]
    points: _containers.RepeatedCompositeFieldContainer[ScalingPoint]
    latency_classification: str
    compute_classification: str
    confidence: str
    reasons: _containers.RepeatedScalarFieldContainer[str]
    warnings: _containers.RepeatedCompositeFieldContainer[ReportWarning]
    latency_fit: ScalingModelFit
    compute_fit: ScalingModelFit
    metric_fits: _containers.RepeatedCompositeFieldContainer[ScalingModelFit]
    def __init__(self, points: _Optional[_Iterable[_Union[ScalingPoint, _Mapping]]] = ..., latency_classification: _Optional[str] = ..., compute_classification: _Optional[str] = ..., confidence: _Optional[str] = ..., reasons: _Optional[_Iterable[str]] = ..., warnings: _Optional[_Iterable[_Union[ReportWarning, _Mapping]]] = ..., latency_fit: _Optional[_Union[ScalingModelFit, _Mapping]] = ..., compute_fit: _Optional[_Union[ScalingModelFit, _Mapping]] = ..., metric_fits: _Optional[_Iterable[_Union[ScalingModelFit, _Mapping]]] = ...) -> None: ...
