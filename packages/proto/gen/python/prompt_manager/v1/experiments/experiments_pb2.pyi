from google.protobuf import struct_pb2 as _struct_pb2
from google.protobuf.internal import containers as _containers
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Iterable as _Iterable, Mapping as _Mapping
from typing import ClassVar as _ClassVar, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class ListExperimentsRequest(_message.Message):
    __slots__ = ("skill_id",)
    SKILL_ID_FIELD_NUMBER: _ClassVar[int]
    skill_id: str
    def __init__(self, skill_id: _Optional[str] = ...) -> None: ...

class ListExperimentsResponse(_message.Message):
    __slots__ = ("experiments",)
    EXPERIMENTS_FIELD_NUMBER: _ClassVar[int]
    experiments: _containers.RepeatedCompositeFieldContainer[Experiment]
    def __init__(self, experiments: _Optional[_Iterable[_Union[Experiment, _Mapping]]] = ...) -> None: ...

class GetExperimentRequest(_message.Message):
    __slots__ = ("experiment_id",)
    EXPERIMENT_ID_FIELD_NUMBER: _ClassVar[int]
    experiment_id: str
    def __init__(self, experiment_id: _Optional[str] = ...) -> None: ...

class DeleteExperimentRequest(_message.Message):
    __slots__ = ("experiment_id",)
    EXPERIMENT_ID_FIELD_NUMBER: _ClassVar[int]
    experiment_id: str
    def __init__(self, experiment_id: _Optional[str] = ...) -> None: ...

class DeleteExperimentResponse(_message.Message):
    __slots__ = ()
    def __init__(self) -> None: ...

class StartExperimentRequest(_message.Message):
    __slots__ = ("experiment_id",)
    EXPERIMENT_ID_FIELD_NUMBER: _ClassVar[int]
    experiment_id: str
    def __init__(self, experiment_id: _Optional[str] = ...) -> None: ...

class ListOutcomesRequest(_message.Message):
    __slots__ = ("experiment_id",)
    EXPERIMENT_ID_FIELD_NUMBER: _ClassVar[int]
    experiment_id: str
    def __init__(self, experiment_id: _Optional[str] = ...) -> None: ...

class ListOutcomesResponse(_message.Message):
    __slots__ = ("outcomes",)
    OUTCOMES_FIELD_NUMBER: _ClassVar[int]
    outcomes: _containers.RepeatedCompositeFieldContainer[ExperimentOutcome]
    def __init__(self, outcomes: _Optional[_Iterable[_Union[ExperimentOutcome, _Mapping]]] = ...) -> None: ...

class GetExperimentReportRequest(_message.Message):
    __slots__ = ("experiment_id",)
    EXPERIMENT_ID_FIELD_NUMBER: _ClassVar[int]
    experiment_id: str
    def __init__(self, experiment_id: _Optional[str] = ...) -> None: ...

class ExperimentArmInput(_message.Message):
    __slots__ = ("variant_id", "weight")
    VARIANT_ID_FIELD_NUMBER: _ClassVar[int]
    WEIGHT_FIELD_NUMBER: _ClassVar[int]
    variant_id: str
    weight: float
    def __init__(self, variant_id: _Optional[str] = ..., weight: _Optional[float] = ...) -> None: ...

class ExperimentArm(_message.Message):
    __slots__ = ("variant_id", "variant_name", "weight")
    VARIANT_ID_FIELD_NUMBER: _ClassVar[int]
    VARIANT_NAME_FIELD_NUMBER: _ClassVar[int]
    WEIGHT_FIELD_NUMBER: _ClassVar[int]
    variant_id: str
    variant_name: str
    weight: float
    def __init__(self, variant_id: _Optional[str] = ..., variant_name: _Optional[str] = ..., weight: _Optional[float] = ...) -> None: ...

class ExperimentProtocol(_message.Message):
    __slots__ = ("value",)
    VALUE_FIELD_NUMBER: _ClassVar[int]
    value: _struct_pb2.Struct
    def __init__(self, value: _Optional[_Union[_struct_pb2.Struct, _Mapping]] = ...) -> None: ...

class Experiment(_message.Message):
    __slots__ = ("id", "skill_id", "name", "hypothesis", "protocol", "status", "arms", "outcome_counts", "started_at", "concluded_at", "winner_variant_id", "promotion_work_item_ref", "holdout_findings_hash", "holdout_completed_at", "promoted_at", "notes", "created_at", "updated_at", "revision")
    class OutcomeCountsEntry(_message.Message):
        __slots__ = ("key", "value")
        KEY_FIELD_NUMBER: _ClassVar[int]
        VALUE_FIELD_NUMBER: _ClassVar[int]
        key: str
        value: int
        def __init__(self, key: _Optional[str] = ..., value: _Optional[int] = ...) -> None: ...
    ID_FIELD_NUMBER: _ClassVar[int]
    SKILL_ID_FIELD_NUMBER: _ClassVar[int]
    NAME_FIELD_NUMBER: _ClassVar[int]
    HYPOTHESIS_FIELD_NUMBER: _ClassVar[int]
    PROTOCOL_FIELD_NUMBER: _ClassVar[int]
    STATUS_FIELD_NUMBER: _ClassVar[int]
    ARMS_FIELD_NUMBER: _ClassVar[int]
    OUTCOME_COUNTS_FIELD_NUMBER: _ClassVar[int]
    STARTED_AT_FIELD_NUMBER: _ClassVar[int]
    CONCLUDED_AT_FIELD_NUMBER: _ClassVar[int]
    WINNER_VARIANT_ID_FIELD_NUMBER: _ClassVar[int]
    PROMOTION_WORK_ITEM_REF_FIELD_NUMBER: _ClassVar[int]
    HOLDOUT_FINDINGS_HASH_FIELD_NUMBER: _ClassVar[int]
    HOLDOUT_COMPLETED_AT_FIELD_NUMBER: _ClassVar[int]
    PROMOTED_AT_FIELD_NUMBER: _ClassVar[int]
    NOTES_FIELD_NUMBER: _ClassVar[int]
    CREATED_AT_FIELD_NUMBER: _ClassVar[int]
    UPDATED_AT_FIELD_NUMBER: _ClassVar[int]
    REVISION_FIELD_NUMBER: _ClassVar[int]
    id: str
    skill_id: str
    name: str
    hypothesis: str
    protocol: _struct_pb2.Struct
    status: str
    arms: _containers.RepeatedCompositeFieldContainer[ExperimentArm]
    outcome_counts: _containers.ScalarMap[str, int]
    started_at: str
    concluded_at: str
    winner_variant_id: str
    promotion_work_item_ref: str
    holdout_findings_hash: str
    holdout_completed_at: str
    promoted_at: str
    notes: str
    created_at: str
    updated_at: str
    revision: int
    def __init__(self, id: _Optional[str] = ..., skill_id: _Optional[str] = ..., name: _Optional[str] = ..., hypothesis: _Optional[str] = ..., protocol: _Optional[_Union[_struct_pb2.Struct, _Mapping]] = ..., status: _Optional[str] = ..., arms: _Optional[_Iterable[_Union[ExperimentArm, _Mapping]]] = ..., outcome_counts: _Optional[_Mapping[str, int]] = ..., started_at: _Optional[str] = ..., concluded_at: _Optional[str] = ..., winner_variant_id: _Optional[str] = ..., promotion_work_item_ref: _Optional[str] = ..., holdout_findings_hash: _Optional[str] = ..., holdout_completed_at: _Optional[str] = ..., promoted_at: _Optional[str] = ..., notes: _Optional[str] = ..., created_at: _Optional[str] = ..., updated_at: _Optional[str] = ..., revision: _Optional[int] = ...) -> None: ...

class CreateExperimentRequest(_message.Message):
    __slots__ = ("id", "skill_id", "name", "hypothesis", "protocol", "arms")
    ID_FIELD_NUMBER: _ClassVar[int]
    SKILL_ID_FIELD_NUMBER: _ClassVar[int]
    NAME_FIELD_NUMBER: _ClassVar[int]
    HYPOTHESIS_FIELD_NUMBER: _ClassVar[int]
    PROTOCOL_FIELD_NUMBER: _ClassVar[int]
    ARMS_FIELD_NUMBER: _ClassVar[int]
    id: str
    skill_id: str
    name: str
    hypothesis: str
    protocol: _struct_pb2.Struct
    arms: _containers.RepeatedCompositeFieldContainer[ExperimentArmInput]
    def __init__(self, id: _Optional[str] = ..., skill_id: _Optional[str] = ..., name: _Optional[str] = ..., hypothesis: _Optional[str] = ..., protocol: _Optional[_Union[_struct_pb2.Struct, _Mapping]] = ..., arms: _Optional[_Iterable[_Union[ExperimentArmInput, _Mapping]]] = ...) -> None: ...

class UpdateExperimentRequest(_message.Message):
    __slots__ = ("experiment_id", "name", "hypothesis", "protocol", "arms")
    EXPERIMENT_ID_FIELD_NUMBER: _ClassVar[int]
    NAME_FIELD_NUMBER: _ClassVar[int]
    HYPOTHESIS_FIELD_NUMBER: _ClassVar[int]
    PROTOCOL_FIELD_NUMBER: _ClassVar[int]
    ARMS_FIELD_NUMBER: _ClassVar[int]
    experiment_id: str
    name: str
    hypothesis: str
    protocol: _struct_pb2.Struct
    arms: _containers.RepeatedCompositeFieldContainer[ExperimentArmInput]
    def __init__(self, experiment_id: _Optional[str] = ..., name: _Optional[str] = ..., hypothesis: _Optional[str] = ..., protocol: _Optional[_Union[_struct_pb2.Struct, _Mapping]] = ..., arms: _Optional[_Iterable[_Union[ExperimentArmInput, _Mapping]]] = ...) -> None: ...

class ConcludeExperimentRequest(_message.Message):
    __slots__ = ("experiment_id", "winner_variant_id", "notes", "override", "override_justification")
    EXPERIMENT_ID_FIELD_NUMBER: _ClassVar[int]
    WINNER_VARIANT_ID_FIELD_NUMBER: _ClassVar[int]
    NOTES_FIELD_NUMBER: _ClassVar[int]
    OVERRIDE_FIELD_NUMBER: _ClassVar[int]
    OVERRIDE_JUSTIFICATION_FIELD_NUMBER: _ClassVar[int]
    experiment_id: str
    winner_variant_id: str
    notes: str
    override: bool
    override_justification: str
    def __init__(self, experiment_id: _Optional[str] = ..., winner_variant_id: _Optional[str] = ..., notes: _Optional[str] = ..., override: _Optional[bool] = ..., override_justification: _Optional[str] = ...) -> None: ...

class RecordOutcomeRequest(_message.Message):
    __slots__ = ("experiment_id", "idempotency_key", "variant_id", "source", "schema_version", "data", "controlled")
    EXPERIMENT_ID_FIELD_NUMBER: _ClassVar[int]
    IDEMPOTENCY_KEY_FIELD_NUMBER: _ClassVar[int]
    VARIANT_ID_FIELD_NUMBER: _ClassVar[int]
    SOURCE_FIELD_NUMBER: _ClassVar[int]
    SCHEMA_VERSION_FIELD_NUMBER: _ClassVar[int]
    DATA_FIELD_NUMBER: _ClassVar[int]
    CONTROLLED_FIELD_NUMBER: _ClassVar[int]
    experiment_id: str
    idempotency_key: str
    variant_id: str
    source: str
    schema_version: int
    data: _struct_pb2.Value
    controlled: _struct_pb2.Struct
    def __init__(self, experiment_id: _Optional[str] = ..., idempotency_key: _Optional[str] = ..., variant_id: _Optional[str] = ..., source: _Optional[str] = ..., schema_version: _Optional[int] = ..., data: _Optional[_Union[_struct_pb2.Value, _Mapping]] = ..., controlled: _Optional[_Union[_struct_pb2.Struct, _Mapping]] = ...) -> None: ...

class ExperimentOutcome(_message.Message):
    __slots__ = ("idempotency_key", "variant_id", "source", "schema_version", "recorded_at", "data", "controlled")
    IDEMPOTENCY_KEY_FIELD_NUMBER: _ClassVar[int]
    VARIANT_ID_FIELD_NUMBER: _ClassVar[int]
    SOURCE_FIELD_NUMBER: _ClassVar[int]
    SCHEMA_VERSION_FIELD_NUMBER: _ClassVar[int]
    RECORDED_AT_FIELD_NUMBER: _ClassVar[int]
    DATA_FIELD_NUMBER: _ClassVar[int]
    CONTROLLED_FIELD_NUMBER: _ClassVar[int]
    idempotency_key: str
    variant_id: str
    source: str
    schema_version: int
    recorded_at: str
    data: _struct_pb2.Value
    controlled: _struct_pb2.Struct
    def __init__(self, idempotency_key: _Optional[str] = ..., variant_id: _Optional[str] = ..., source: _Optional[str] = ..., schema_version: _Optional[int] = ..., recorded_at: _Optional[str] = ..., data: _Optional[_Union[_struct_pb2.Value, _Mapping]] = ..., controlled: _Optional[_Union[_struct_pb2.Struct, _Mapping]] = ...) -> None: ...

class AssignExperimentRequest(_message.Message):
    __slots__ = ("experiment_id", "execution_id", "node_id", "attempt_key", "idempotency_key", "variables", "with_scope")
    class VariablesEntry(_message.Message):
        __slots__ = ("key", "value")
        KEY_FIELD_NUMBER: _ClassVar[int]
        VALUE_FIELD_NUMBER: _ClassVar[int]
        key: str
        value: str
        def __init__(self, key: _Optional[str] = ..., value: _Optional[str] = ...) -> None: ...
    EXPERIMENT_ID_FIELD_NUMBER: _ClassVar[int]
    EXECUTION_ID_FIELD_NUMBER: _ClassVar[int]
    NODE_ID_FIELD_NUMBER: _ClassVar[int]
    ATTEMPT_KEY_FIELD_NUMBER: _ClassVar[int]
    IDEMPOTENCY_KEY_FIELD_NUMBER: _ClassVar[int]
    VARIABLES_FIELD_NUMBER: _ClassVar[int]
    WITH_SCOPE_FIELD_NUMBER: _ClassVar[int]
    experiment_id: str
    execution_id: str
    node_id: str
    attempt_key: str
    idempotency_key: str
    variables: _containers.ScalarMap[str, str]
    with_scope: bool
    def __init__(self, experiment_id: _Optional[str] = ..., execution_id: _Optional[str] = ..., node_id: _Optional[str] = ..., attempt_key: _Optional[str] = ..., idempotency_key: _Optional[str] = ..., variables: _Optional[_Mapping[str, str]] = ..., with_scope: _Optional[bool] = ...) -> None: ...

class ExperimentAssignment(_message.Message):
    __slots__ = ("experiment_id", "skill_id", "variant_id", "content", "content_hash", "assigned_at")
    EXPERIMENT_ID_FIELD_NUMBER: _ClassVar[int]
    SKILL_ID_FIELD_NUMBER: _ClassVar[int]
    VARIANT_ID_FIELD_NUMBER: _ClassVar[int]
    CONTENT_FIELD_NUMBER: _ClassVar[int]
    CONTENT_HASH_FIELD_NUMBER: _ClassVar[int]
    ASSIGNED_AT_FIELD_NUMBER: _ClassVar[int]
    experiment_id: str
    skill_id: str
    variant_id: str
    content: str
    content_hash: str
    assigned_at: str
    def __init__(self, experiment_id: _Optional[str] = ..., skill_id: _Optional[str] = ..., variant_id: _Optional[str] = ..., content: _Optional[str] = ..., content_hash: _Optional[str] = ..., assigned_at: _Optional[str] = ...) -> None: ...

class RecordAuditReceiptRequest(_message.Message):
    __slots__ = ("experiment_id", "sampled_assignment_ids", "findings_hash", "challenge_state", "anomaly_count", "gaming_count", "idempotency_key")
    EXPERIMENT_ID_FIELD_NUMBER: _ClassVar[int]
    SAMPLED_ASSIGNMENT_IDS_FIELD_NUMBER: _ClassVar[int]
    FINDINGS_HASH_FIELD_NUMBER: _ClassVar[int]
    CHALLENGE_STATE_FIELD_NUMBER: _ClassVar[int]
    ANOMALY_COUNT_FIELD_NUMBER: _ClassVar[int]
    GAMING_COUNT_FIELD_NUMBER: _ClassVar[int]
    IDEMPOTENCY_KEY_FIELD_NUMBER: _ClassVar[int]
    experiment_id: str
    sampled_assignment_ids: _containers.RepeatedScalarFieldContainer[str]
    findings_hash: str
    challenge_state: str
    anomaly_count: int
    gaming_count: int
    idempotency_key: str
    def __init__(self, experiment_id: _Optional[str] = ..., sampled_assignment_ids: _Optional[_Iterable[str]] = ..., findings_hash: _Optional[str] = ..., challenge_state: _Optional[str] = ..., anomaly_count: _Optional[int] = ..., gaming_count: _Optional[int] = ..., idempotency_key: _Optional[str] = ...) -> None: ...

class RecordHoldoutReceiptRequest(_message.Message):
    __slots__ = ("experiment_id", "findings_hash", "idempotency_key")
    EXPERIMENT_ID_FIELD_NUMBER: _ClassVar[int]
    FINDINGS_HASH_FIELD_NUMBER: _ClassVar[int]
    IDEMPOTENCY_KEY_FIELD_NUMBER: _ClassVar[int]
    experiment_id: str
    findings_hash: str
    idempotency_key: str
    def __init__(self, experiment_id: _Optional[str] = ..., findings_hash: _Optional[str] = ..., idempotency_key: _Optional[str] = ...) -> None: ...

class PromoteExperimentRequest(_message.Message):
    __slots__ = ("experiment_id", "work_item_ref")
    EXPERIMENT_ID_FIELD_NUMBER: _ClassVar[int]
    WORK_ITEM_REF_FIELD_NUMBER: _ClassVar[int]
    experiment_id: str
    work_item_ref: str
    def __init__(self, experiment_id: _Optional[str] = ..., work_item_ref: _Optional[str] = ...) -> None: ...

class ExperimentReport(_message.Message):
    __slots__ = ("experiment_id", "skill_id", "name", "status", "total_serves", "total_outcomes", "arms", "zero_data_arms", "controlled")
    EXPERIMENT_ID_FIELD_NUMBER: _ClassVar[int]
    SKILL_ID_FIELD_NUMBER: _ClassVar[int]
    NAME_FIELD_NUMBER: _ClassVar[int]
    STATUS_FIELD_NUMBER: _ClassVar[int]
    TOTAL_SERVES_FIELD_NUMBER: _ClassVar[int]
    TOTAL_OUTCOMES_FIELD_NUMBER: _ClassVar[int]
    ARMS_FIELD_NUMBER: _ClassVar[int]
    ZERO_DATA_ARMS_FIELD_NUMBER: _ClassVar[int]
    CONTROLLED_FIELD_NUMBER: _ClassVar[int]
    experiment_id: str
    skill_id: str
    name: str
    status: str
    total_serves: int
    total_outcomes: int
    arms: _containers.RepeatedCompositeFieldContainer[_struct_pb2.Struct]
    zero_data_arms: _containers.RepeatedScalarFieldContainer[str]
    controlled: _struct_pb2.Struct
    def __init__(self, experiment_id: _Optional[str] = ..., skill_id: _Optional[str] = ..., name: _Optional[str] = ..., status: _Optional[str] = ..., total_serves: _Optional[int] = ..., total_outcomes: _Optional[int] = ..., arms: _Optional[_Iterable[_Union[_struct_pb2.Struct, _Mapping]]] = ..., zero_data_arms: _Optional[_Iterable[str]] = ..., controlled: _Optional[_Union[_struct_pb2.Struct, _Mapping]] = ...) -> None: ...
