from google.protobuf.internal import containers as _containers
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Iterable as _Iterable, Mapping as _Mapping
from typing import ClassVar as _ClassVar, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class ListQualityProfilesRequest(_message.Message):
    __slots__ = ()
    def __init__(self) -> None: ...

class SubmitEvaluationResponse(_message.Message):
    __slots__ = ("job_id",)
    JOB_ID_FIELD_NUMBER: _ClassVar[int]
    job_id: str
    def __init__(self, job_id: _Optional[str] = ...) -> None: ...

class ListQualityProfilesResponse(_message.Message):
    __slots__ = ("evaluator_version", "profiles", "operations")
    EVALUATOR_VERSION_FIELD_NUMBER: _ClassVar[int]
    PROFILES_FIELD_NUMBER: _ClassVar[int]
    OPERATIONS_FIELD_NUMBER: _ClassVar[int]
    evaluator_version: str
    profiles: _containers.RepeatedCompositeFieldContainer[QualityProfile]
    operations: _containers.RepeatedScalarFieldContainer[str]
    def __init__(self, evaluator_version: _Optional[str] = ..., profiles: _Optional[_Iterable[_Union[QualityProfile, _Mapping]]] = ..., operations: _Optional[_Iterable[str]] = ...) -> None: ...

class QualityProfile(_message.Message):
    __slots__ = ("id", "description", "metrics", "rubric")
    ID_FIELD_NUMBER: _ClassVar[int]
    DESCRIPTION_FIELD_NUMBER: _ClassVar[int]
    METRICS_FIELD_NUMBER: _ClassVar[int]
    RUBRIC_FIELD_NUMBER: _ClassVar[int]
    id: str
    description: str
    metrics: _containers.RepeatedScalarFieldContainer[str]
    rubric: str
    def __init__(self, id: _Optional[str] = ..., description: _Optional[str] = ..., metrics: _Optional[_Iterable[str]] = ..., rubric: _Optional[str] = ...) -> None: ...

class EvaluationCase(_message.Message):
    __slots__ = ("id", "input", "reference", "output", "reference_text", "output_text", "instruction", "cohort", "input_sha256", "reference_sha256", "output_sha256", "labels")
    class LabelsEntry(_message.Message):
        __slots__ = ("key", "value")
        KEY_FIELD_NUMBER: _ClassVar[int]
        VALUE_FIELD_NUMBER: _ClassVar[int]
        key: str
        value: str
        def __init__(self, key: _Optional[str] = ..., value: _Optional[str] = ...) -> None: ...
    ID_FIELD_NUMBER: _ClassVar[int]
    INPUT_FIELD_NUMBER: _ClassVar[int]
    REFERENCE_FIELD_NUMBER: _ClassVar[int]
    OUTPUT_FIELD_NUMBER: _ClassVar[int]
    REFERENCE_TEXT_FIELD_NUMBER: _ClassVar[int]
    OUTPUT_TEXT_FIELD_NUMBER: _ClassVar[int]
    INSTRUCTION_FIELD_NUMBER: _ClassVar[int]
    COHORT_FIELD_NUMBER: _ClassVar[int]
    INPUT_SHA256_FIELD_NUMBER: _ClassVar[int]
    REFERENCE_SHA256_FIELD_NUMBER: _ClassVar[int]
    OUTPUT_SHA256_FIELD_NUMBER: _ClassVar[int]
    LABELS_FIELD_NUMBER: _ClassVar[int]
    id: str
    input: str
    reference: str
    output: str
    reference_text: str
    output_text: str
    instruction: str
    cohort: str
    input_sha256: str
    reference_sha256: str
    output_sha256: str
    labels: _containers.ScalarMap[str, str]
    def __init__(self, id: _Optional[str] = ..., input: _Optional[str] = ..., reference: _Optional[str] = ..., output: _Optional[str] = ..., reference_text: _Optional[str] = ..., output_text: _Optional[str] = ..., instruction: _Optional[str] = ..., cohort: _Optional[str] = ..., input_sha256: _Optional[str] = ..., reference_sha256: _Optional[str] = ..., output_sha256: _Optional[str] = ..., labels: _Optional[_Mapping[str, str]] = ...) -> None: ...

class MetricBudget(_message.Message):
    __slots__ = ("metric", "scope", "minimum", "maximum", "max_regression", "min_improvement")
    METRIC_FIELD_NUMBER: _ClassVar[int]
    SCOPE_FIELD_NUMBER: _ClassVar[int]
    MINIMUM_FIELD_NUMBER: _ClassVar[int]
    MAXIMUM_FIELD_NUMBER: _ClassVar[int]
    MAX_REGRESSION_FIELD_NUMBER: _ClassVar[int]
    MIN_IMPROVEMENT_FIELD_NUMBER: _ClassVar[int]
    metric: str
    scope: str
    minimum: float
    maximum: float
    max_regression: float
    min_improvement: float
    def __init__(self, metric: _Optional[str] = ..., scope: _Optional[str] = ..., minimum: _Optional[float] = ..., maximum: _Optional[float] = ..., max_regression: _Optional[float] = ..., min_improvement: _Optional[float] = ...) -> None: ...

class EvaluationManifest(_message.Message):
    __slots__ = ("schema_version", "name", "operation", "profile", "dataset_id", "candidate_model", "candidate_revision", "cases", "budgets", "judge", "rubric", "judge_model")
    SCHEMA_VERSION_FIELD_NUMBER: _ClassVar[int]
    NAME_FIELD_NUMBER: _ClassVar[int]
    OPERATION_FIELD_NUMBER: _ClassVar[int]
    PROFILE_FIELD_NUMBER: _ClassVar[int]
    DATASET_ID_FIELD_NUMBER: _ClassVar[int]
    CANDIDATE_MODEL_FIELD_NUMBER: _ClassVar[int]
    CANDIDATE_REVISION_FIELD_NUMBER: _ClassVar[int]
    CASES_FIELD_NUMBER: _ClassVar[int]
    BUDGETS_FIELD_NUMBER: _ClassVar[int]
    JUDGE_FIELD_NUMBER: _ClassVar[int]
    RUBRIC_FIELD_NUMBER: _ClassVar[int]
    JUDGE_MODEL_FIELD_NUMBER: _ClassVar[int]
    schema_version: int
    name: str
    operation: str
    profile: str
    dataset_id: str
    candidate_model: str
    candidate_revision: str
    cases: _containers.RepeatedCompositeFieldContainer[EvaluationCase]
    budgets: _containers.RepeatedCompositeFieldContainer[MetricBudget]
    judge: str
    rubric: str
    judge_model: str
    def __init__(self, schema_version: _Optional[int] = ..., name: _Optional[str] = ..., operation: _Optional[str] = ..., profile: _Optional[str] = ..., dataset_id: _Optional[str] = ..., candidate_model: _Optional[str] = ..., candidate_revision: _Optional[str] = ..., cases: _Optional[_Iterable[_Union[EvaluationCase, _Mapping]]] = ..., budgets: _Optional[_Iterable[_Union[MetricBudget, _Mapping]]] = ..., judge: _Optional[str] = ..., rubric: _Optional[str] = ..., judge_model: _Optional[str] = ...) -> None: ...

class JudgeResult(_message.Message):
    __slots__ = ("provider", "model", "fidelity", "instruction", "artifacts", "rationale", "duration_ms", "rubric_sha256", "validated", "route_event_id", "capacity_claim_id", "capacity_ownership")
    PROVIDER_FIELD_NUMBER: _ClassVar[int]
    MODEL_FIELD_NUMBER: _ClassVar[int]
    FIDELITY_FIELD_NUMBER: _ClassVar[int]
    INSTRUCTION_FIELD_NUMBER: _ClassVar[int]
    ARTIFACTS_FIELD_NUMBER: _ClassVar[int]
    RATIONALE_FIELD_NUMBER: _ClassVar[int]
    DURATION_MS_FIELD_NUMBER: _ClassVar[int]
    RUBRIC_SHA256_FIELD_NUMBER: _ClassVar[int]
    VALIDATED_FIELD_NUMBER: _ClassVar[int]
    ROUTE_EVENT_ID_FIELD_NUMBER: _ClassVar[int]
    CAPACITY_CLAIM_ID_FIELD_NUMBER: _ClassVar[int]
    CAPACITY_OWNERSHIP_FIELD_NUMBER: _ClassVar[int]
    provider: str
    model: str
    fidelity: float
    instruction: float
    artifacts: float
    rationale: str
    duration_ms: int
    rubric_sha256: str
    validated: bool
    route_event_id: str
    capacity_claim_id: str
    capacity_ownership: str
    def __init__(self, provider: _Optional[str] = ..., model: _Optional[str] = ..., fidelity: _Optional[float] = ..., instruction: _Optional[float] = ..., artifacts: _Optional[float] = ..., rationale: _Optional[str] = ..., duration_ms: _Optional[int] = ..., rubric_sha256: _Optional[str] = ..., validated: _Optional[bool] = ..., route_event_id: _Optional[str] = ..., capacity_claim_id: _Optional[str] = ..., capacity_ownership: _Optional[str] = ...) -> None: ...

class CaseScore(_message.Message):
    __slots__ = ("id", "cohort", "input_sha256", "reference_sha256", "output_sha256", "metrics", "judge", "errors", "warnings", "duration_ms", "case_sha256")
    class MetricsEntry(_message.Message):
        __slots__ = ("key", "value")
        KEY_FIELD_NUMBER: _ClassVar[int]
        VALUE_FIELD_NUMBER: _ClassVar[int]
        key: str
        value: float
        def __init__(self, key: _Optional[str] = ..., value: _Optional[float] = ...) -> None: ...
    ID_FIELD_NUMBER: _ClassVar[int]
    COHORT_FIELD_NUMBER: _ClassVar[int]
    INPUT_SHA256_FIELD_NUMBER: _ClassVar[int]
    REFERENCE_SHA256_FIELD_NUMBER: _ClassVar[int]
    OUTPUT_SHA256_FIELD_NUMBER: _ClassVar[int]
    METRICS_FIELD_NUMBER: _ClassVar[int]
    JUDGE_FIELD_NUMBER: _ClassVar[int]
    ERRORS_FIELD_NUMBER: _ClassVar[int]
    WARNINGS_FIELD_NUMBER: _ClassVar[int]
    DURATION_MS_FIELD_NUMBER: _ClassVar[int]
    CASE_SHA256_FIELD_NUMBER: _ClassVar[int]
    id: str
    cohort: str
    input_sha256: str
    reference_sha256: str
    output_sha256: str
    metrics: _containers.ScalarMap[str, float]
    judge: JudgeResult
    errors: _containers.RepeatedScalarFieldContainer[str]
    warnings: _containers.RepeatedScalarFieldContainer[str]
    duration_ms: int
    case_sha256: str
    def __init__(self, id: _Optional[str] = ..., cohort: _Optional[str] = ..., input_sha256: _Optional[str] = ..., reference_sha256: _Optional[str] = ..., output_sha256: _Optional[str] = ..., metrics: _Optional[_Mapping[str, float]] = ..., judge: _Optional[_Union[JudgeResult, _Mapping]] = ..., errors: _Optional[_Iterable[str]] = ..., warnings: _Optional[_Iterable[str]] = ..., duration_ms: _Optional[int] = ..., case_sha256: _Optional[str] = ...) -> None: ...

class GateViolation(_message.Message):
    __slots__ = ("case_id", "metric", "reason", "observed", "limit")
    CASE_ID_FIELD_NUMBER: _ClassVar[int]
    METRIC_FIELD_NUMBER: _ClassVar[int]
    REASON_FIELD_NUMBER: _ClassVar[int]
    OBSERVED_FIELD_NUMBER: _ClassVar[int]
    LIMIT_FIELD_NUMBER: _ClassVar[int]
    case_id: str
    metric: str
    reason: str
    observed: float
    limit: float
    def __init__(self, case_id: _Optional[str] = ..., metric: _Optional[str] = ..., reason: _Optional[str] = ..., observed: _Optional[float] = ..., limit: _Optional[float] = ...) -> None: ...

class EvaluationReport(_message.Message):
    __slots__ = ("schema_version", "evaluator_version", "job_id", "result_ref", "name", "operation", "profile", "dataset_id", "candidate_model", "candidate_revision", "suite_sha256", "policy_sha256", "report_sha256", "baseline_sha256", "cases", "means", "violations", "passed", "complete", "promotion_eligible", "judge_model", "started_at", "duration_ms", "limits", "budgets")
    class MeansEntry(_message.Message):
        __slots__ = ("key", "value")
        KEY_FIELD_NUMBER: _ClassVar[int]
        VALUE_FIELD_NUMBER: _ClassVar[int]
        key: str
        value: float
        def __init__(self, key: _Optional[str] = ..., value: _Optional[float] = ...) -> None: ...
    SCHEMA_VERSION_FIELD_NUMBER: _ClassVar[int]
    EVALUATOR_VERSION_FIELD_NUMBER: _ClassVar[int]
    JOB_ID_FIELD_NUMBER: _ClassVar[int]
    RESULT_REF_FIELD_NUMBER: _ClassVar[int]
    NAME_FIELD_NUMBER: _ClassVar[int]
    OPERATION_FIELD_NUMBER: _ClassVar[int]
    PROFILE_FIELD_NUMBER: _ClassVar[int]
    DATASET_ID_FIELD_NUMBER: _ClassVar[int]
    CANDIDATE_MODEL_FIELD_NUMBER: _ClassVar[int]
    CANDIDATE_REVISION_FIELD_NUMBER: _ClassVar[int]
    SUITE_SHA256_FIELD_NUMBER: _ClassVar[int]
    POLICY_SHA256_FIELD_NUMBER: _ClassVar[int]
    REPORT_SHA256_FIELD_NUMBER: _ClassVar[int]
    BASELINE_SHA256_FIELD_NUMBER: _ClassVar[int]
    CASES_FIELD_NUMBER: _ClassVar[int]
    MEANS_FIELD_NUMBER: _ClassVar[int]
    VIOLATIONS_FIELD_NUMBER: _ClassVar[int]
    PASSED_FIELD_NUMBER: _ClassVar[int]
    COMPLETE_FIELD_NUMBER: _ClassVar[int]
    PROMOTION_ELIGIBLE_FIELD_NUMBER: _ClassVar[int]
    JUDGE_MODEL_FIELD_NUMBER: _ClassVar[int]
    STARTED_AT_FIELD_NUMBER: _ClassVar[int]
    DURATION_MS_FIELD_NUMBER: _ClassVar[int]
    LIMITS_FIELD_NUMBER: _ClassVar[int]
    BUDGETS_FIELD_NUMBER: _ClassVar[int]
    schema_version: int
    evaluator_version: str
    job_id: str
    result_ref: str
    name: str
    operation: str
    profile: str
    dataset_id: str
    candidate_model: str
    candidate_revision: str
    suite_sha256: str
    policy_sha256: str
    report_sha256: str
    baseline_sha256: str
    cases: _containers.RepeatedCompositeFieldContainer[CaseScore]
    means: _containers.ScalarMap[str, float]
    violations: _containers.RepeatedCompositeFieldContainer[GateViolation]
    passed: bool
    complete: bool
    promotion_eligible: bool
    judge_model: str
    started_at: str
    duration_ms: int
    limits: _containers.RepeatedScalarFieldContainer[str]
    budgets: _containers.RepeatedCompositeFieldContainer[MetricBudget]
    def __init__(self, schema_version: _Optional[int] = ..., evaluator_version: _Optional[str] = ..., job_id: _Optional[str] = ..., result_ref: _Optional[str] = ..., name: _Optional[str] = ..., operation: _Optional[str] = ..., profile: _Optional[str] = ..., dataset_id: _Optional[str] = ..., candidate_model: _Optional[str] = ..., candidate_revision: _Optional[str] = ..., suite_sha256: _Optional[str] = ..., policy_sha256: _Optional[str] = ..., report_sha256: _Optional[str] = ..., baseline_sha256: _Optional[str] = ..., cases: _Optional[_Iterable[_Union[CaseScore, _Mapping]]] = ..., means: _Optional[_Mapping[str, float]] = ..., violations: _Optional[_Iterable[_Union[GateViolation, _Mapping]]] = ..., passed: _Optional[bool] = ..., complete: _Optional[bool] = ..., promotion_eligible: _Optional[bool] = ..., judge_model: _Optional[str] = ..., started_at: _Optional[str] = ..., duration_ms: _Optional[int] = ..., limits: _Optional[_Iterable[str]] = ..., budgets: _Optional[_Iterable[_Union[MetricBudget, _Mapping]]] = ...) -> None: ...
