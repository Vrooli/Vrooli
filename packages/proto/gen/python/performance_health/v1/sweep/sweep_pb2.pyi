from google.protobuf.internal import containers as _containers
from google.protobuf.internal import enum_type_wrapper as _enum_type_wrapper
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Iterable as _Iterable, Mapping as _Mapping
from typing import ClassVar as _ClassVar, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class WorkloadOutcome(int, metaclass=_enum_type_wrapper.EnumTypeWrapper):
    __slots__ = ()
    WORKLOAD_OUTCOME_UNSPECIFIED: _ClassVar[WorkloadOutcome]
    WORKLOAD_OUTCOME_MEASURED: _ClassVar[WorkloadOutcome]
    WORKLOAD_OUTCOME_UNAVAILABLE: _ClassVar[WorkloadOutcome]
    WORKLOAD_OUTCOME_FAILED: _ClassVar[WorkloadOutcome]
WORKLOAD_OUTCOME_UNSPECIFIED: WorkloadOutcome
WORKLOAD_OUTCOME_MEASURED: WorkloadOutcome
WORKLOAD_OUTCOME_UNAVAILABLE: WorkloadOutcome
WORKLOAD_OUTCOME_FAILED: WorkloadOutcome

class WorkloadRequest(_message.Message):
    __slots__ = ("scenario", "workload")
    SCENARIO_FIELD_NUMBER: _ClassVar[int]
    WORKLOAD_FIELD_NUMBER: _ClassVar[int]
    scenario: str
    workload: str
    def __init__(self, scenario: _Optional[str] = ..., workload: _Optional[str] = ...) -> None: ...

class WorkloadReading(_message.Message):
    __slots__ = ("scenario", "workload", "operation_id", "outcome", "reason", "within_budget", "p95_ms", "wall_p95_ms", "budget_ms", "sample_count", "attempt_count", "declared_warmups", "captured_at", "build_identity", "producer_digest", "contract_digest", "configuration_digest", "fixture_revision", "receipt_path", "receipt_sha256")
    SCENARIO_FIELD_NUMBER: _ClassVar[int]
    WORKLOAD_FIELD_NUMBER: _ClassVar[int]
    OPERATION_ID_FIELD_NUMBER: _ClassVar[int]
    OUTCOME_FIELD_NUMBER: _ClassVar[int]
    REASON_FIELD_NUMBER: _ClassVar[int]
    WITHIN_BUDGET_FIELD_NUMBER: _ClassVar[int]
    P95_MS_FIELD_NUMBER: _ClassVar[int]
    WALL_P95_MS_FIELD_NUMBER: _ClassVar[int]
    BUDGET_MS_FIELD_NUMBER: _ClassVar[int]
    SAMPLE_COUNT_FIELD_NUMBER: _ClassVar[int]
    ATTEMPT_COUNT_FIELD_NUMBER: _ClassVar[int]
    DECLARED_WARMUPS_FIELD_NUMBER: _ClassVar[int]
    CAPTURED_AT_FIELD_NUMBER: _ClassVar[int]
    BUILD_IDENTITY_FIELD_NUMBER: _ClassVar[int]
    PRODUCER_DIGEST_FIELD_NUMBER: _ClassVar[int]
    CONTRACT_DIGEST_FIELD_NUMBER: _ClassVar[int]
    CONFIGURATION_DIGEST_FIELD_NUMBER: _ClassVar[int]
    FIXTURE_REVISION_FIELD_NUMBER: _ClassVar[int]
    RECEIPT_PATH_FIELD_NUMBER: _ClassVar[int]
    RECEIPT_SHA256_FIELD_NUMBER: _ClassVar[int]
    scenario: str
    workload: str
    operation_id: str
    outcome: WorkloadOutcome
    reason: str
    within_budget: bool
    p95_ms: float
    wall_p95_ms: float
    budget_ms: float
    sample_count: int
    attempt_count: int
    declared_warmups: int
    captured_at: str
    build_identity: str
    producer_digest: str
    contract_digest: str
    configuration_digest: str
    fixture_revision: str
    receipt_path: str
    receipt_sha256: str
    def __init__(self, scenario: _Optional[str] = ..., workload: _Optional[str] = ..., operation_id: _Optional[str] = ..., outcome: _Optional[_Union[WorkloadOutcome, str]] = ..., reason: _Optional[str] = ..., within_budget: _Optional[bool] = ..., p95_ms: _Optional[float] = ..., wall_p95_ms: _Optional[float] = ..., budget_ms: _Optional[float] = ..., sample_count: _Optional[int] = ..., attempt_count: _Optional[int] = ..., declared_warmups: _Optional[int] = ..., captured_at: _Optional[str] = ..., build_identity: _Optional[str] = ..., producer_digest: _Optional[str] = ..., contract_digest: _Optional[str] = ..., configuration_digest: _Optional[str] = ..., fixture_revision: _Optional[str] = ..., receipt_path: _Optional[str] = ..., receipt_sha256: _Optional[str] = ...) -> None: ...

class RunSweepRequest(_message.Message):
    __slots__ = ("scenario",)
    SCENARIO_FIELD_NUMBER: _ClassVar[int]
    scenario: str
    def __init__(self, scenario: _Optional[str] = ...) -> None: ...

class FlowSweepResult(_message.Message):
    __slots__ = ("flow", "outcome", "within_budget", "violations", "reason")
    FLOW_FIELD_NUMBER: _ClassVar[int]
    OUTCOME_FIELD_NUMBER: _ClassVar[int]
    WITHIN_BUDGET_FIELD_NUMBER: _ClassVar[int]
    VIOLATIONS_FIELD_NUMBER: _ClassVar[int]
    REASON_FIELD_NUMBER: _ClassVar[int]
    flow: str
    outcome: str
    within_budget: bool
    violations: _containers.RepeatedScalarFieldContainer[str]
    reason: str
    def __init__(self, flow: _Optional[str] = ..., outcome: _Optional[str] = ..., within_budget: _Optional[bool] = ..., violations: _Optional[_Iterable[str]] = ..., reason: _Optional[str] = ...) -> None: ...

class RunSweepResponse(_message.Message):
    __slots__ = ("scenario", "results")
    SCENARIO_FIELD_NUMBER: _ClassVar[int]
    RESULTS_FIELD_NUMBER: _ClassVar[int]
    scenario: str
    results: _containers.RepeatedCompositeFieldContainer[FlowSweepResult]
    def __init__(self, scenario: _Optional[str] = ..., results: _Optional[_Iterable[_Union[FlowSweepResult, _Mapping]]] = ...) -> None: ...
