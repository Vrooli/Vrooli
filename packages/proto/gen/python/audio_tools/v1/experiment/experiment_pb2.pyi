import datetime

from audio_tools.v1.eval import eval_pb2 as _eval_pb2
from audio_tools.v1.stt import stt_pb2 as _stt_pb2
from google.protobuf import timestamp_pb2 as _timestamp_pb2
from google.protobuf.internal import containers as _containers
from google.protobuf.internal import enum_type_wrapper as _enum_type_wrapper
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Iterable as _Iterable, Mapping as _Mapping
from typing import ClassVar as _ClassVar, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class ExperimentStatus(int, metaclass=_enum_type_wrapper.EnumTypeWrapper):
    __slots__ = ()
    EXPERIMENT_STATUS_UNSPECIFIED: _ClassVar[ExperimentStatus]
    EXPERIMENT_STATUS_QUEUED: _ClassVar[ExperimentStatus]
    EXPERIMENT_STATUS_RUNNING: _ClassVar[ExperimentStatus]
    EXPERIMENT_STATUS_SUCCEEDED: _ClassVar[ExperimentStatus]
    EXPERIMENT_STATUS_FAILED: _ClassVar[ExperimentStatus]
    EXPERIMENT_STATUS_CANCELED: _ClassVar[ExperimentStatus]

class ReplayLane(int, metaclass=_enum_type_wrapper.EnumTypeWrapper):
    __slots__ = ()
    REPLAY_LANE_UNSPECIFIED: _ClassVar[ReplayLane]
    REPLAY_LANE_DETERMINISTIC: _ClassVar[ReplayLane]
    REPLAY_LANE_REALTIME: _ClassVar[ReplayLane]
    REPLAY_LANE_PRODUCT_PATH: _ClassVar[ReplayLane]

class QualificationEvidenceKind(int, metaclass=_enum_type_wrapper.EnumTypeWrapper):
    __slots__ = ()
    QUALIFICATION_EVIDENCE_KIND_UNSPECIFIED: _ClassVar[QualificationEvidenceKind]
    QUALIFICATION_EVIDENCE_KIND_INTERVAL_ACCOUNTING: _ClassVar[QualificationEvidenceKind]
    QUALIFICATION_EVIDENCE_KIND_BOUNDED_RECOVERY: _ClassVar[QualificationEvidenceKind]
    QUALIFICATION_EVIDENCE_KIND_FAULT: _ClassVar[QualificationEvidenceKind]
    QUALIFICATION_EVIDENCE_KIND_BROWSER_PRODUCT_PATH: _ClassVar[QualificationEvidenceKind]
    QUALIFICATION_EVIDENCE_KIND_DEVICE: _ClassVar[QualificationEvidenceKind]
EXPERIMENT_STATUS_UNSPECIFIED: ExperimentStatus
EXPERIMENT_STATUS_QUEUED: ExperimentStatus
EXPERIMENT_STATUS_RUNNING: ExperimentStatus
EXPERIMENT_STATUS_SUCCEEDED: ExperimentStatus
EXPERIMENT_STATUS_FAILED: ExperimentStatus
EXPERIMENT_STATUS_CANCELED: ExperimentStatus
REPLAY_LANE_UNSPECIFIED: ReplayLane
REPLAY_LANE_DETERMINISTIC: ReplayLane
REPLAY_LANE_REALTIME: ReplayLane
REPLAY_LANE_PRODUCT_PATH: ReplayLane
QUALIFICATION_EVIDENCE_KIND_UNSPECIFIED: QualificationEvidenceKind
QUALIFICATION_EVIDENCE_KIND_INTERVAL_ACCOUNTING: QualificationEvidenceKind
QUALIFICATION_EVIDENCE_KIND_BOUNDED_RECOVERY: QualificationEvidenceKind
QUALIFICATION_EVIDENCE_KIND_FAULT: QualificationEvidenceKind
QUALIFICATION_EVIDENCE_KIND_BROWSER_PRODUCT_PATH: QualificationEvidenceKind
QUALIFICATION_EVIDENCE_KIND_DEVICE: QualificationEvidenceKind

class ExperimentRecipe(_message.Message):
    __slots__ = ("clip_ids", "strategies", "realtime_repeats", "chunk_ms", "seed", "long_form", "realized_clip_ids", "realized_reference", "realized_duration_ms", "augmentation", "realized_augmentation_conditions", "speaker", "realized_speaker_conditions", "dropped_span_threshold_words", "latency_tail_seconds", "cells")
    CLIP_IDS_FIELD_NUMBER: _ClassVar[int]
    STRATEGIES_FIELD_NUMBER: _ClassVar[int]
    REALTIME_REPEATS_FIELD_NUMBER: _ClassVar[int]
    CHUNK_MS_FIELD_NUMBER: _ClassVar[int]
    SEED_FIELD_NUMBER: _ClassVar[int]
    LONG_FORM_FIELD_NUMBER: _ClassVar[int]
    REALIZED_CLIP_IDS_FIELD_NUMBER: _ClassVar[int]
    REALIZED_REFERENCE_FIELD_NUMBER: _ClassVar[int]
    REALIZED_DURATION_MS_FIELD_NUMBER: _ClassVar[int]
    AUGMENTATION_FIELD_NUMBER: _ClassVar[int]
    REALIZED_AUGMENTATION_CONDITIONS_FIELD_NUMBER: _ClassVar[int]
    SPEAKER_FIELD_NUMBER: _ClassVar[int]
    REALIZED_SPEAKER_CONDITIONS_FIELD_NUMBER: _ClassVar[int]
    DROPPED_SPAN_THRESHOLD_WORDS_FIELD_NUMBER: _ClassVar[int]
    LATENCY_TAIL_SECONDS_FIELD_NUMBER: _ClassVar[int]
    CELLS_FIELD_NUMBER: _ClassVar[int]
    clip_ids: _containers.RepeatedScalarFieldContainer[str]
    strategies: _containers.RepeatedCompositeFieldContainer[_eval_pb2.EvalStrategy]
    realtime_repeats: int
    chunk_ms: int
    seed: int
    long_form: LongFormRecipe
    realized_clip_ids: _containers.RepeatedScalarFieldContainer[str]
    realized_reference: str
    realized_duration_ms: int
    augmentation: AugmentationRecipe
    realized_augmentation_conditions: _containers.RepeatedCompositeFieldContainer[AugmentationCondition]
    speaker: SpeakerExperimentRecipe
    realized_speaker_conditions: _containers.RepeatedCompositeFieldContainer[SpeakerCondition]
    dropped_span_threshold_words: int
    latency_tail_seconds: int
    cells: _containers.RepeatedCompositeFieldContainer[EvaluationCell]
    def __init__(self, clip_ids: _Optional[_Iterable[str]] = ..., strategies: _Optional[_Iterable[_Union[_eval_pb2.EvalStrategy, _Mapping]]] = ..., realtime_repeats: _Optional[int] = ..., chunk_ms: _Optional[int] = ..., seed: _Optional[int] = ..., long_form: _Optional[_Union[LongFormRecipe, _Mapping]] = ..., realized_clip_ids: _Optional[_Iterable[str]] = ..., realized_reference: _Optional[str] = ..., realized_duration_ms: _Optional[int] = ..., augmentation: _Optional[_Union[AugmentationRecipe, _Mapping]] = ..., realized_augmentation_conditions: _Optional[_Iterable[_Union[AugmentationCondition, _Mapping]]] = ..., speaker: _Optional[_Union[SpeakerExperimentRecipe, _Mapping]] = ..., realized_speaker_conditions: _Optional[_Iterable[_Union[SpeakerCondition, _Mapping]]] = ..., dropped_span_threshold_words: _Optional[int] = ..., latency_tail_seconds: _Optional[int] = ..., cells: _Optional[_Iterable[_Union[EvaluationCell, _Mapping]]] = ...) -> None: ...

class EvaluationCell(_message.Message):
    __slots__ = ("engine_id", "strategy", "policy_profile", "replay_lane", "fault_profile", "warm_start", "repeat_count", "label")
    ENGINE_ID_FIELD_NUMBER: _ClassVar[int]
    STRATEGY_FIELD_NUMBER: _ClassVar[int]
    POLICY_PROFILE_FIELD_NUMBER: _ClassVar[int]
    REPLAY_LANE_FIELD_NUMBER: _ClassVar[int]
    FAULT_PROFILE_FIELD_NUMBER: _ClassVar[int]
    WARM_START_FIELD_NUMBER: _ClassVar[int]
    REPEAT_COUNT_FIELD_NUMBER: _ClassVar[int]
    LABEL_FIELD_NUMBER: _ClassVar[int]
    engine_id: str
    strategy: str
    policy_profile: str
    replay_lane: ReplayLane
    fault_profile: str
    warm_start: bool
    repeat_count: int
    label: str
    def __init__(self, engine_id: _Optional[str] = ..., strategy: _Optional[str] = ..., policy_profile: _Optional[str] = ..., replay_lane: _Optional[_Union[ReplayLane, str]] = ..., fault_profile: _Optional[str] = ..., warm_start: _Optional[bool] = ..., repeat_count: _Optional[int] = ..., label: _Optional[str] = ...) -> None: ...

class LongFormRecipe(_message.Message):
    __slots__ = ("enabled", "target_duration_seconds", "gap_ms", "tag_contains", "sweep_durations_seconds")
    ENABLED_FIELD_NUMBER: _ClassVar[int]
    TARGET_DURATION_SECONDS_FIELD_NUMBER: _ClassVar[int]
    GAP_MS_FIELD_NUMBER: _ClassVar[int]
    TAG_CONTAINS_FIELD_NUMBER: _ClassVar[int]
    SWEEP_DURATIONS_SECONDS_FIELD_NUMBER: _ClassVar[int]
    enabled: bool
    target_duration_seconds: int
    gap_ms: int
    tag_contains: str
    sweep_durations_seconds: _containers.RepeatedScalarFieldContainer[int]
    def __init__(self, enabled: _Optional[bool] = ..., target_duration_seconds: _Optional[int] = ..., gap_ms: _Optional[int] = ..., tag_contains: _Optional[str] = ..., sweep_durations_seconds: _Optional[_Iterable[int]] = ...) -> None: ...

class AugmentationRecipe(_message.Message):
    __slots__ = ("noise_types", "snr_db", "competing_voice_ids", "competing_text")
    NOISE_TYPES_FIELD_NUMBER: _ClassVar[int]
    SNR_DB_FIELD_NUMBER: _ClassVar[int]
    COMPETING_VOICE_IDS_FIELD_NUMBER: _ClassVar[int]
    COMPETING_TEXT_FIELD_NUMBER: _ClassVar[int]
    noise_types: _containers.RepeatedScalarFieldContainer[str]
    snr_db: _containers.RepeatedScalarFieldContainer[float]
    competing_voice_ids: _containers.RepeatedScalarFieldContainer[str]
    competing_text: str
    def __init__(self, noise_types: _Optional[_Iterable[str]] = ..., snr_db: _Optional[_Iterable[float]] = ..., competing_voice_ids: _Optional[_Iterable[str]] = ..., competing_text: _Optional[str] = ...) -> None: ...

class AugmentationCondition(_message.Message):
    __slots__ = ("id", "kind", "source", "snr_db", "skipped", "note")
    ID_FIELD_NUMBER: _ClassVar[int]
    KIND_FIELD_NUMBER: _ClassVar[int]
    SOURCE_FIELD_NUMBER: _ClassVar[int]
    SNR_DB_FIELD_NUMBER: _ClassVar[int]
    SKIPPED_FIELD_NUMBER: _ClassVar[int]
    NOTE_FIELD_NUMBER: _ClassVar[int]
    id: str
    kind: str
    source: str
    snr_db: float
    skipped: bool
    note: str
    def __init__(self, id: _Optional[str] = ..., kind: _Optional[str] = ..., source: _Optional[str] = ..., snr_db: _Optional[float] = ..., skipped: _Optional[bool] = ..., note: _Optional[str] = ...) -> None: ...

class SpeakerExperimentRecipe(_message.Message):
    __slots__ = ("target_profile_id", "extraction_enabled", "verification_enabled", "verification_mode", "threshold", "fallback_without_verification", "ablation_enabled")
    TARGET_PROFILE_ID_FIELD_NUMBER: _ClassVar[int]
    EXTRACTION_ENABLED_FIELD_NUMBER: _ClassVar[int]
    VERIFICATION_ENABLED_FIELD_NUMBER: _ClassVar[int]
    VERIFICATION_MODE_FIELD_NUMBER: _ClassVar[int]
    THRESHOLD_FIELD_NUMBER: _ClassVar[int]
    FALLBACK_WITHOUT_VERIFICATION_FIELD_NUMBER: _ClassVar[int]
    ABLATION_ENABLED_FIELD_NUMBER: _ClassVar[int]
    target_profile_id: str
    extraction_enabled: bool
    verification_enabled: bool
    verification_mode: _stt_pb2.SpeakerMode
    threshold: float
    fallback_without_verification: bool
    ablation_enabled: bool
    def __init__(self, target_profile_id: _Optional[str] = ..., extraction_enabled: _Optional[bool] = ..., verification_enabled: _Optional[bool] = ..., verification_mode: _Optional[_Union[_stt_pb2.SpeakerMode, str]] = ..., threshold: _Optional[float] = ..., fallback_without_verification: _Optional[bool] = ..., ablation_enabled: _Optional[bool] = ...) -> None: ...

class SpeakerCondition(_message.Message):
    __slots__ = ("id", "extraction_enabled", "verification_enabled", "verification_mode", "skipped", "note")
    ID_FIELD_NUMBER: _ClassVar[int]
    EXTRACTION_ENABLED_FIELD_NUMBER: _ClassVar[int]
    VERIFICATION_ENABLED_FIELD_NUMBER: _ClassVar[int]
    VERIFICATION_MODE_FIELD_NUMBER: _ClassVar[int]
    SKIPPED_FIELD_NUMBER: _ClassVar[int]
    NOTE_FIELD_NUMBER: _ClassVar[int]
    id: str
    extraction_enabled: bool
    verification_enabled: bool
    verification_mode: _stt_pb2.SpeakerMode
    skipped: bool
    note: str
    def __init__(self, id: _Optional[str] = ..., extraction_enabled: _Optional[bool] = ..., verification_enabled: _Optional[bool] = ..., verification_mode: _Optional[_Union[_stt_pb2.SpeakerMode, str]] = ..., skipped: _Optional[bool] = ..., note: _Optional[str] = ...) -> None: ...

class Experiment(_message.Message):
    __slots__ = ("id", "name", "status", "recipe", "created_at", "started_at", "finished_at", "error", "result_ref", "machine_json")
    ID_FIELD_NUMBER: _ClassVar[int]
    NAME_FIELD_NUMBER: _ClassVar[int]
    STATUS_FIELD_NUMBER: _ClassVar[int]
    RECIPE_FIELD_NUMBER: _ClassVar[int]
    CREATED_AT_FIELD_NUMBER: _ClassVar[int]
    STARTED_AT_FIELD_NUMBER: _ClassVar[int]
    FINISHED_AT_FIELD_NUMBER: _ClassVar[int]
    ERROR_FIELD_NUMBER: _ClassVar[int]
    RESULT_REF_FIELD_NUMBER: _ClassVar[int]
    MACHINE_JSON_FIELD_NUMBER: _ClassVar[int]
    id: str
    name: str
    status: ExperimentStatus
    recipe: ExperimentRecipe
    created_at: _timestamp_pb2.Timestamp
    started_at: _timestamp_pb2.Timestamp
    finished_at: _timestamp_pb2.Timestamp
    error: str
    result_ref: str
    machine_json: str
    def __init__(self, id: _Optional[str] = ..., name: _Optional[str] = ..., status: _Optional[_Union[ExperimentStatus, str]] = ..., recipe: _Optional[_Union[ExperimentRecipe, _Mapping]] = ..., created_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., started_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., finished_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., error: _Optional[str] = ..., result_ref: _Optional[str] = ..., machine_json: _Optional[str] = ...) -> None: ...

class ExperimentRun(_message.Message):
    __slots__ = ("id", "experiment_id", "strategy", "condition_json", "created_at", "engine_id", "policy_profile", "replay_lane", "fault_profile")
    ID_FIELD_NUMBER: _ClassVar[int]
    EXPERIMENT_ID_FIELD_NUMBER: _ClassVar[int]
    STRATEGY_FIELD_NUMBER: _ClassVar[int]
    CONDITION_JSON_FIELD_NUMBER: _ClassVar[int]
    CREATED_AT_FIELD_NUMBER: _ClassVar[int]
    ENGINE_ID_FIELD_NUMBER: _ClassVar[int]
    POLICY_PROFILE_FIELD_NUMBER: _ClassVar[int]
    REPLAY_LANE_FIELD_NUMBER: _ClassVar[int]
    FAULT_PROFILE_FIELD_NUMBER: _ClassVar[int]
    id: str
    experiment_id: str
    strategy: str
    condition_json: str
    created_at: _timestamp_pb2.Timestamp
    engine_id: str
    policy_profile: str
    replay_lane: ReplayLane
    fault_profile: str
    def __init__(self, id: _Optional[str] = ..., experiment_id: _Optional[str] = ..., strategy: _Optional[str] = ..., condition_json: _Optional[str] = ..., created_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., engine_id: _Optional[str] = ..., policy_profile: _Optional[str] = ..., replay_lane: _Optional[_Union[ReplayLane, str]] = ..., fault_profile: _Optional[str] = ...) -> None: ...

class ExperimentEvent(_message.Message):
    __slots__ = ("experiment_id", "status", "progress", "message", "at")
    EXPERIMENT_ID_FIELD_NUMBER: _ClassVar[int]
    STATUS_FIELD_NUMBER: _ClassVar[int]
    PROGRESS_FIELD_NUMBER: _ClassVar[int]
    MESSAGE_FIELD_NUMBER: _ClassVar[int]
    AT_FIELD_NUMBER: _ClassVar[int]
    experiment_id: str
    status: ExperimentStatus
    progress: int
    message: str
    at: _timestamp_pb2.Timestamp
    def __init__(self, experiment_id: _Optional[str] = ..., status: _Optional[_Union[ExperimentStatus, str]] = ..., progress: _Optional[int] = ..., message: _Optional[str] = ..., at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ...) -> None: ...

class QualificationEvidence(_message.Message):
    __slots__ = ("id", "engine_id", "strategy", "policy_profile", "kind", "fault_profile", "passed", "artifact_ref", "notes", "machine_json", "observed_at", "model_id")
    ID_FIELD_NUMBER: _ClassVar[int]
    ENGINE_ID_FIELD_NUMBER: _ClassVar[int]
    STRATEGY_FIELD_NUMBER: _ClassVar[int]
    POLICY_PROFILE_FIELD_NUMBER: _ClassVar[int]
    KIND_FIELD_NUMBER: _ClassVar[int]
    FAULT_PROFILE_FIELD_NUMBER: _ClassVar[int]
    PASSED_FIELD_NUMBER: _ClassVar[int]
    ARTIFACT_REF_FIELD_NUMBER: _ClassVar[int]
    NOTES_FIELD_NUMBER: _ClassVar[int]
    MACHINE_JSON_FIELD_NUMBER: _ClassVar[int]
    OBSERVED_AT_FIELD_NUMBER: _ClassVar[int]
    MODEL_ID_FIELD_NUMBER: _ClassVar[int]
    id: str
    engine_id: str
    strategy: str
    policy_profile: str
    kind: QualificationEvidenceKind
    fault_profile: str
    passed: bool
    artifact_ref: str
    notes: str
    machine_json: str
    observed_at: _timestamp_pb2.Timestamp
    model_id: str
    def __init__(self, id: _Optional[str] = ..., engine_id: _Optional[str] = ..., strategy: _Optional[str] = ..., policy_profile: _Optional[str] = ..., kind: _Optional[_Union[QualificationEvidenceKind, str]] = ..., fault_profile: _Optional[str] = ..., passed: _Optional[bool] = ..., artifact_ref: _Optional[str] = ..., notes: _Optional[str] = ..., machine_json: _Optional[str] = ..., observed_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., model_id: _Optional[str] = ...) -> None: ...

class StartExperimentRequest(_message.Message):
    __slots__ = ("name", "recipe", "estimated_seconds", "dry_run")
    NAME_FIELD_NUMBER: _ClassVar[int]
    RECIPE_FIELD_NUMBER: _ClassVar[int]
    ESTIMATED_SECONDS_FIELD_NUMBER: _ClassVar[int]
    DRY_RUN_FIELD_NUMBER: _ClassVar[int]
    name: str
    recipe: ExperimentRecipe
    estimated_seconds: int
    dry_run: bool
    def __init__(self, name: _Optional[str] = ..., recipe: _Optional[_Union[ExperimentRecipe, _Mapping]] = ..., estimated_seconds: _Optional[int] = ..., dry_run: _Optional[bool] = ...) -> None: ...

class StartExperimentResponse(_message.Message):
    __slots__ = ("experiment", "estimated_seconds", "dry_run")
    EXPERIMENT_FIELD_NUMBER: _ClassVar[int]
    ESTIMATED_SECONDS_FIELD_NUMBER: _ClassVar[int]
    DRY_RUN_FIELD_NUMBER: _ClassVar[int]
    experiment: Experiment
    estimated_seconds: int
    dry_run: bool
    def __init__(self, experiment: _Optional[_Union[Experiment, _Mapping]] = ..., estimated_seconds: _Optional[int] = ..., dry_run: _Optional[bool] = ...) -> None: ...

class GetExperimentRequest(_message.Message):
    __slots__ = ("id",)
    ID_FIELD_NUMBER: _ClassVar[int]
    id: str
    def __init__(self, id: _Optional[str] = ...) -> None: ...

class GetExperimentResponse(_message.Message):
    __slots__ = ("experiment", "runs")
    EXPERIMENT_FIELD_NUMBER: _ClassVar[int]
    RUNS_FIELD_NUMBER: _ClassVar[int]
    experiment: Experiment
    runs: _containers.RepeatedCompositeFieldContainer[ExperimentRun]
    def __init__(self, experiment: _Optional[_Union[Experiment, _Mapping]] = ..., runs: _Optional[_Iterable[_Union[ExperimentRun, _Mapping]]] = ...) -> None: ...

class WaitExperimentRequest(_message.Message):
    __slots__ = ("id",)
    ID_FIELD_NUMBER: _ClassVar[int]
    id: str
    def __init__(self, id: _Optional[str] = ...) -> None: ...

class WaitExperimentResponse(_message.Message):
    __slots__ = ("experiment", "runs")
    EXPERIMENT_FIELD_NUMBER: _ClassVar[int]
    RUNS_FIELD_NUMBER: _ClassVar[int]
    experiment: Experiment
    runs: _containers.RepeatedCompositeFieldContainer[ExperimentRun]
    def __init__(self, experiment: _Optional[_Union[Experiment, _Mapping]] = ..., runs: _Optional[_Iterable[_Union[ExperimentRun, _Mapping]]] = ...) -> None: ...

class ListExperimentsRequest(_message.Message):
    __slots__ = ("status", "limit", "offset")
    STATUS_FIELD_NUMBER: _ClassVar[int]
    LIMIT_FIELD_NUMBER: _ClassVar[int]
    OFFSET_FIELD_NUMBER: _ClassVar[int]
    status: ExperimentStatus
    limit: int
    offset: int
    def __init__(self, status: _Optional[_Union[ExperimentStatus, str]] = ..., limit: _Optional[int] = ..., offset: _Optional[int] = ...) -> None: ...

class ListExperimentsResponse(_message.Message):
    __slots__ = ("experiments",)
    EXPERIMENTS_FIELD_NUMBER: _ClassVar[int]
    experiments: _containers.RepeatedCompositeFieldContainer[Experiment]
    def __init__(self, experiments: _Optional[_Iterable[_Union[Experiment, _Mapping]]] = ...) -> None: ...

class CancelExperimentRequest(_message.Message):
    __slots__ = ("id",)
    ID_FIELD_NUMBER: _ClassVar[int]
    id: str
    def __init__(self, id: _Optional[str] = ...) -> None: ...

class CancelExperimentResponse(_message.Message):
    __slots__ = ("experiment",)
    EXPERIMENT_FIELD_NUMBER: _ClassVar[int]
    experiment: Experiment
    def __init__(self, experiment: _Optional[_Union[Experiment, _Mapping]] = ...) -> None: ...

class DeleteExperimentRequest(_message.Message):
    __slots__ = ("id",)
    ID_FIELD_NUMBER: _ClassVar[int]
    id: str
    def __init__(self, id: _Optional[str] = ...) -> None: ...

class DeleteExperimentResponse(_message.Message):
    __slots__ = ("id", "deleted_report")
    ID_FIELD_NUMBER: _ClassVar[int]
    DELETED_REPORT_FIELD_NUMBER: _ClassVar[int]
    id: str
    deleted_report: bool
    def __init__(self, id: _Optional[str] = ..., deleted_report: _Optional[bool] = ...) -> None: ...

class StreamExperimentEventsRequest(_message.Message):
    __slots__ = ("id",)
    ID_FIELD_NUMBER: _ClassVar[int]
    id: str
    def __init__(self, id: _Optional[str] = ...) -> None: ...

class GetExperimentReportRequest(_message.Message):
    __slots__ = ("id",)
    ID_FIELD_NUMBER: _ClassVar[int]
    id: str
    def __init__(self, id: _Optional[str] = ...) -> None: ...

class GetExperimentReportResponse(_message.Message):
    __slots__ = ("experiment", "report", "runs")
    EXPERIMENT_FIELD_NUMBER: _ClassVar[int]
    REPORT_FIELD_NUMBER: _ClassVar[int]
    RUNS_FIELD_NUMBER: _ClassVar[int]
    experiment: Experiment
    report: _eval_pb2.EvalReport
    runs: _containers.RepeatedCompositeFieldContainer[ExperimentRun]
    def __init__(self, experiment: _Optional[_Union[Experiment, _Mapping]] = ..., report: _Optional[_Union[_eval_pb2.EvalReport, _Mapping]] = ..., runs: _Optional[_Iterable[_Union[ExperimentRun, _Mapping]]] = ...) -> None: ...

class CompareExperimentsRequest(_message.Message):
    __slots__ = ("ids",)
    IDS_FIELD_NUMBER: _ClassVar[int]
    ids: _containers.RepeatedScalarFieldContainer[str]
    def __init__(self, ids: _Optional[_Iterable[str]] = ...) -> None: ...

class ComparedExperiment(_message.Message):
    __slots__ = ("experiment", "report", "runs")
    EXPERIMENT_FIELD_NUMBER: _ClassVar[int]
    REPORT_FIELD_NUMBER: _ClassVar[int]
    RUNS_FIELD_NUMBER: _ClassVar[int]
    experiment: Experiment
    report: _eval_pb2.EvalReport
    runs: _containers.RepeatedCompositeFieldContainer[ExperimentRun]
    def __init__(self, experiment: _Optional[_Union[Experiment, _Mapping]] = ..., report: _Optional[_Union[_eval_pb2.EvalReport, _Mapping]] = ..., runs: _Optional[_Iterable[_Union[ExperimentRun, _Mapping]]] = ...) -> None: ...

class CompareExperimentsResponse(_message.Message):
    __slots__ = ("experiments",)
    EXPERIMENTS_FIELD_NUMBER: _ClassVar[int]
    experiments: _containers.RepeatedCompositeFieldContainer[ComparedExperiment]
    def __init__(self, experiments: _Optional[_Iterable[_Union[ComparedExperiment, _Mapping]]] = ...) -> None: ...

class RecordQualificationEvidenceRequest(_message.Message):
    __slots__ = ("evidence",)
    EVIDENCE_FIELD_NUMBER: _ClassVar[int]
    evidence: QualificationEvidence
    def __init__(self, evidence: _Optional[_Union[QualificationEvidence, _Mapping]] = ...) -> None: ...

class RecordQualificationEvidenceResponse(_message.Message):
    __slots__ = ("evidence",)
    EVIDENCE_FIELD_NUMBER: _ClassVar[int]
    evidence: QualificationEvidence
    def __init__(self, evidence: _Optional[_Union[QualificationEvidence, _Mapping]] = ...) -> None: ...

class ListQualificationEvidenceRequest(_message.Message):
    __slots__ = ("engine_id", "strategy", "policy_profile", "model_id")
    ENGINE_ID_FIELD_NUMBER: _ClassVar[int]
    STRATEGY_FIELD_NUMBER: _ClassVar[int]
    POLICY_PROFILE_FIELD_NUMBER: _ClassVar[int]
    MODEL_ID_FIELD_NUMBER: _ClassVar[int]
    engine_id: str
    strategy: str
    policy_profile: str
    model_id: str
    def __init__(self, engine_id: _Optional[str] = ..., strategy: _Optional[str] = ..., policy_profile: _Optional[str] = ..., model_id: _Optional[str] = ...) -> None: ...

class ListQualificationEvidenceResponse(_message.Message):
    __slots__ = ("evidence",)
    EVIDENCE_FIELD_NUMBER: _ClassVar[int]
    evidence: _containers.RepeatedCompositeFieldContainer[QualificationEvidence]
    def __init__(self, evidence: _Optional[_Iterable[_Union[QualificationEvidence, _Mapping]]] = ...) -> None: ...
