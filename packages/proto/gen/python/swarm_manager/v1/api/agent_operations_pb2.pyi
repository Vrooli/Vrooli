from buf.validate import validate_pb2 as _validate_pb2
from swarm_manager.v1.domain import agent_operations_pb2 as _agent_operations_pb2
from swarm_manager.v1.domain import operating_mode_pb2 as _operating_mode_pb2
from google.protobuf.internal import containers as _containers
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Iterable as _Iterable, Mapping as _Mapping
from typing import ClassVar as _ClassVar, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class AgentOpsTargetSelector(_message.Message):
    __slots__ = ("kind", "id")
    KIND_FIELD_NUMBER: _ClassVar[int]
    ID_FIELD_NUMBER: _ClassVar[int]
    kind: _operating_mode_pb2.OperatingModeTargetKind
    id: str
    def __init__(self, kind: _Optional[_Union[_operating_mode_pb2.OperatingModeTargetKind, str]] = ..., id: _Optional[str] = ...) -> None: ...

class AgentOpsResolveBindingRequest(_message.Message):
    __slots__ = ("target", "operation", "operation_version")
    TARGET_FIELD_NUMBER: _ClassVar[int]
    OPERATION_FIELD_NUMBER: _ClassVar[int]
    OPERATION_VERSION_FIELD_NUMBER: _ClassVar[int]
    target: AgentOpsTargetSelector
    operation: str
    operation_version: str
    def __init__(self, target: _Optional[_Union[AgentOpsTargetSelector, _Mapping]] = ..., operation: _Optional[str] = ..., operation_version: _Optional[str] = ...) -> None: ...

class AgentOpsResolveBindingResponse(_message.Message):
    __slots__ = ("resolved", "policy_id", "policy_revision")
    RESOLVED_FIELD_NUMBER: _ClassVar[int]
    POLICY_ID_FIELD_NUMBER: _ClassVar[int]
    POLICY_REVISION_FIELD_NUMBER: _ClassVar[int]
    resolved: _agent_operations_pb2.AgentOpsOperationBinding
    policy_id: str
    policy_revision: str
    def __init__(self, resolved: _Optional[_Union[_agent_operations_pb2.AgentOpsOperationBinding, _Mapping]] = ..., policy_id: _Optional[str] = ..., policy_revision: _Optional[str] = ...) -> None: ...

class AgentOpsValidateInvocationRequest(_message.Message):
    __slots__ = ("target", "operation", "operation_version")
    TARGET_FIELD_NUMBER: _ClassVar[int]
    OPERATION_FIELD_NUMBER: _ClassVar[int]
    OPERATION_VERSION_FIELD_NUMBER: _ClassVar[int]
    target: AgentOpsTargetSelector
    operation: str
    operation_version: str
    def __init__(self, target: _Optional[_Union[AgentOpsTargetSelector, _Mapping]] = ..., operation: _Optional[str] = ..., operation_version: _Optional[str] = ...) -> None: ...

class AgentOpsValidateInvocationResponse(_message.Message):
    __slots__ = ("operation_declared", "target_compatible", "binding_resolved", "resolved", "missing_capabilities")
    OPERATION_DECLARED_FIELD_NUMBER: _ClassVar[int]
    TARGET_COMPATIBLE_FIELD_NUMBER: _ClassVar[int]
    BINDING_RESOLVED_FIELD_NUMBER: _ClassVar[int]
    RESOLVED_FIELD_NUMBER: _ClassVar[int]
    MISSING_CAPABILITIES_FIELD_NUMBER: _ClassVar[int]
    operation_declared: bool
    target_compatible: bool
    binding_resolved: bool
    resolved: _agent_operations_pb2.AgentOpsOperationBinding
    missing_capabilities: _containers.RepeatedScalarFieldContainer[_agent_operations_pb2.AgentOpsCapability]
    def __init__(self, operation_declared: _Optional[bool] = ..., target_compatible: _Optional[bool] = ..., binding_resolved: _Optional[bool] = ..., resolved: _Optional[_Union[_agent_operations_pb2.AgentOpsOperationBinding, _Mapping]] = ..., missing_capabilities: _Optional[_Iterable[_Union[_agent_operations_pb2.AgentOpsCapability, str]]] = ...) -> None: ...

class AgentOpsInspectWorkflowRequest(_message.Message):
    __slots__ = ("target",)
    TARGET_FIELD_NUMBER: _ClassVar[int]
    target: AgentOpsTargetSelector
    def __init__(self, target: _Optional[_Union[AgentOpsTargetSelector, _Mapping]] = ...) -> None: ...

class AgentOpsInspectWorkflowResponse(_message.Message):
    __slots__ = ("found", "workflow")
    FOUND_FIELD_NUMBER: _ClassVar[int]
    WORKFLOW_FIELD_NUMBER: _ClassVar[int]
    found: bool
    workflow: _agent_operations_pb2.AgentOpsWorkflowInstance
    def __init__(self, found: _Optional[bool] = ..., workflow: _Optional[_Union[_agent_operations_pb2.AgentOpsWorkflowInstance, _Mapping]] = ...) -> None: ...

class AgentOpsInspectExecutionRequest(_message.Message):
    __slots__ = ("target", "execution_id")
    TARGET_FIELD_NUMBER: _ClassVar[int]
    EXECUTION_ID_FIELD_NUMBER: _ClassVar[int]
    target: AgentOpsTargetSelector
    execution_id: str
    def __init__(self, target: _Optional[_Union[AgentOpsTargetSelector, _Mapping]] = ..., execution_id: _Optional[str] = ...) -> None: ...

class AgentOpsInspectExecutionResponse(_message.Message):
    __slots__ = ("found", "provenance", "outcome", "reproducible", "recorded_at")
    FOUND_FIELD_NUMBER: _ClassVar[int]
    PROVENANCE_FIELD_NUMBER: _ClassVar[int]
    OUTCOME_FIELD_NUMBER: _ClassVar[int]
    REPRODUCIBLE_FIELD_NUMBER: _ClassVar[int]
    RECORDED_AT_FIELD_NUMBER: _ClassVar[int]
    found: bool
    provenance: _agent_operations_pb2.AgentOpsExecutionProvenance
    outcome: str
    reproducible: bool
    recorded_at: str
    def __init__(self, found: _Optional[bool] = ..., provenance: _Optional[_Union[_agent_operations_pb2.AgentOpsExecutionProvenance, _Mapping]] = ..., outcome: _Optional[str] = ..., reproducible: _Optional[bool] = ..., recorded_at: _Optional[str] = ...) -> None: ...

class AgentOpsListOperationCatalogRequest(_message.Message):
    __slots__ = ()
    def __init__(self) -> None: ...

class AgentOpsCatalogEntry(_message.Message):
    __slots__ = ("contract", "revision", "compatible_targets")
    CONTRACT_FIELD_NUMBER: _ClassVar[int]
    REVISION_FIELD_NUMBER: _ClassVar[int]
    COMPATIBLE_TARGETS_FIELD_NUMBER: _ClassVar[int]
    contract: _agent_operations_pb2.AgentOpsOperationContract
    revision: str
    compatible_targets: _containers.RepeatedScalarFieldContainer[_operating_mode_pb2.OperatingModeTargetKind]
    def __init__(self, contract: _Optional[_Union[_agent_operations_pb2.AgentOpsOperationContract, _Mapping]] = ..., revision: _Optional[str] = ..., compatible_targets: _Optional[_Iterable[_Union[_operating_mode_pb2.OperatingModeTargetKind, str]]] = ...) -> None: ...

class AgentOpsListOperationCatalogResponse(_message.Message):
    __slots__ = ("entries",)
    ENTRIES_FIELD_NUMBER: _ClassVar[int]
    entries: _containers.RepeatedCompositeFieldContainer[AgentOpsCatalogEntry]
    def __init__(self, entries: _Optional[_Iterable[_Union[AgentOpsCatalogEntry, _Mapping]]] = ...) -> None: ...

class AgentOpsListCompatibleModesRequest(_message.Message):
    __slots__ = ("target", "operation")
    TARGET_FIELD_NUMBER: _ClassVar[int]
    OPERATION_FIELD_NUMBER: _ClassVar[int]
    target: AgentOpsTargetSelector
    operation: str
    def __init__(self, target: _Optional[_Union[AgentOpsTargetSelector, _Mapping]] = ..., operation: _Optional[str] = ...) -> None: ...

class AgentOpsModeOperationVerdict(_message.Message):
    __slots__ = ("operation", "operation_version", "compatible", "reason")
    OPERATION_FIELD_NUMBER: _ClassVar[int]
    OPERATION_VERSION_FIELD_NUMBER: _ClassVar[int]
    COMPATIBLE_FIELD_NUMBER: _ClassVar[int]
    REASON_FIELD_NUMBER: _ClassVar[int]
    operation: str
    operation_version: str
    compatible: bool
    reason: str
    def __init__(self, operation: _Optional[str] = ..., operation_version: _Optional[str] = ..., compatible: _Optional[bool] = ..., reason: _Optional[str] = ...) -> None: ...

class AgentOpsCompatibleMode(_message.Message):
    __slots__ = ("mode", "mode_revision", "mode_digest", "target_kind", "verdicts")
    MODE_FIELD_NUMBER: _ClassVar[int]
    MODE_REVISION_FIELD_NUMBER: _ClassVar[int]
    MODE_DIGEST_FIELD_NUMBER: _ClassVar[int]
    TARGET_KIND_FIELD_NUMBER: _ClassVar[int]
    VERDICTS_FIELD_NUMBER: _ClassVar[int]
    mode: str
    mode_revision: str
    mode_digest: str
    target_kind: _operating_mode_pb2.OperatingModeTargetKind
    verdicts: _containers.RepeatedCompositeFieldContainer[AgentOpsModeOperationVerdict]
    def __init__(self, mode: _Optional[str] = ..., mode_revision: _Optional[str] = ..., mode_digest: _Optional[str] = ..., target_kind: _Optional[_Union[_operating_mode_pb2.OperatingModeTargetKind, str]] = ..., verdicts: _Optional[_Iterable[_Union[AgentOpsModeOperationVerdict, _Mapping]]] = ...) -> None: ...

class AgentOpsListCompatibleModesResponse(_message.Message):
    __slots__ = ("modes",)
    MODES_FIELD_NUMBER: _ClassVar[int]
    modes: _containers.RepeatedCompositeFieldContainer[AgentOpsCompatibleMode]
    def __init__(self, modes: _Optional[_Iterable[_Union[AgentOpsCompatibleMode, _Mapping]]] = ...) -> None: ...

class AgentOpsGetResolvedBindingsRequest(_message.Message):
    __slots__ = ("target",)
    TARGET_FIELD_NUMBER: _ClassVar[int]
    target: AgentOpsTargetSelector
    def __init__(self, target: _Optional[_Union[AgentOpsTargetSelector, _Mapping]] = ...) -> None: ...

class AgentOpsBindingContribution(_message.Message):
    __slots__ = ("binding", "winning")
    BINDING_FIELD_NUMBER: _ClassVar[int]
    WINNING_FIELD_NUMBER: _ClassVar[int]
    binding: _agent_operations_pb2.AgentOpsOperationBinding
    winning: bool
    def __init__(self, binding: _Optional[_Union[_agent_operations_pb2.AgentOpsOperationBinding, _Mapping]] = ..., winning: _Optional[bool] = ...) -> None: ...

class AgentOpsResolvedOperationBinding(_message.Message):
    __slots__ = ("operation", "operation_version", "resolved", "binding", "policy_id", "policy_revision", "error", "error_message", "contributions")
    OPERATION_FIELD_NUMBER: _ClassVar[int]
    OPERATION_VERSION_FIELD_NUMBER: _ClassVar[int]
    RESOLVED_FIELD_NUMBER: _ClassVar[int]
    BINDING_FIELD_NUMBER: _ClassVar[int]
    POLICY_ID_FIELD_NUMBER: _ClassVar[int]
    POLICY_REVISION_FIELD_NUMBER: _ClassVar[int]
    ERROR_FIELD_NUMBER: _ClassVar[int]
    ERROR_MESSAGE_FIELD_NUMBER: _ClassVar[int]
    CONTRIBUTIONS_FIELD_NUMBER: _ClassVar[int]
    operation: str
    operation_version: str
    resolved: bool
    binding: _agent_operations_pb2.AgentOpsOperationBinding
    policy_id: str
    policy_revision: str
    error: str
    error_message: str
    contributions: _containers.RepeatedCompositeFieldContainer[AgentOpsBindingContribution]
    def __init__(self, operation: _Optional[str] = ..., operation_version: _Optional[str] = ..., resolved: _Optional[bool] = ..., binding: _Optional[_Union[_agent_operations_pb2.AgentOpsOperationBinding, _Mapping]] = ..., policy_id: _Optional[str] = ..., policy_revision: _Optional[str] = ..., error: _Optional[str] = ..., error_message: _Optional[str] = ..., contributions: _Optional[_Iterable[_Union[AgentOpsBindingContribution, _Mapping]]] = ...) -> None: ...

class AgentOpsGetResolvedBindingsResponse(_message.Message):
    __slots__ = ("operations",)
    OPERATIONS_FIELD_NUMBER: _ClassVar[int]
    operations: _containers.RepeatedCompositeFieldContainer[AgentOpsResolvedOperationBinding]
    def __init__(self, operations: _Optional[_Iterable[_Union[AgentOpsResolvedOperationBinding, _Mapping]]] = ...) -> None: ...

class AgentOpsBindingOverrideDocument(_message.Message):
    __slots__ = ("binding", "file", "revision", "updated_at")
    BINDING_FIELD_NUMBER: _ClassVar[int]
    FILE_FIELD_NUMBER: _ClassVar[int]
    REVISION_FIELD_NUMBER: _ClassVar[int]
    UPDATED_AT_FIELD_NUMBER: _ClassVar[int]
    binding: _agent_operations_pb2.AgentOpsOperationBinding
    file: str
    revision: str
    updated_at: str
    def __init__(self, binding: _Optional[_Union[_agent_operations_pb2.AgentOpsOperationBinding, _Mapping]] = ..., file: _Optional[str] = ..., revision: _Optional[str] = ..., updated_at: _Optional[str] = ...) -> None: ...

class AgentOpsListBindingOverridesRequest(_message.Message):
    __slots__ = ("owner",)
    OWNER_FIELD_NUMBER: _ClassVar[int]
    owner: AgentOpsTargetSelector
    def __init__(self, owner: _Optional[_Union[AgentOpsTargetSelector, _Mapping]] = ...) -> None: ...

class AgentOpsListBindingOverridesResponse(_message.Message):
    __slots__ = ("overrides",)
    OVERRIDES_FIELD_NUMBER: _ClassVar[int]
    overrides: _containers.RepeatedCompositeFieldContainer[AgentOpsBindingOverrideDocument]
    def __init__(self, overrides: _Optional[_Iterable[_Union[AgentOpsBindingOverrideDocument, _Mapping]]] = ...) -> None: ...

class AgentOpsPutBindingOverrideRequest(_message.Message):
    __slots__ = ("owner", "operation", "operation_version", "mode", "mode_revision", "disabled")
    OWNER_FIELD_NUMBER: _ClassVar[int]
    OPERATION_FIELD_NUMBER: _ClassVar[int]
    OPERATION_VERSION_FIELD_NUMBER: _ClassVar[int]
    MODE_FIELD_NUMBER: _ClassVar[int]
    MODE_REVISION_FIELD_NUMBER: _ClassVar[int]
    DISABLED_FIELD_NUMBER: _ClassVar[int]
    owner: AgentOpsTargetSelector
    operation: str
    operation_version: str
    mode: str
    mode_revision: str
    disabled: bool
    def __init__(self, owner: _Optional[_Union[AgentOpsTargetSelector, _Mapping]] = ..., operation: _Optional[str] = ..., operation_version: _Optional[str] = ..., mode: _Optional[str] = ..., mode_revision: _Optional[str] = ..., disabled: _Optional[bool] = ...) -> None: ...

class AgentOpsPutBindingOverrideResponse(_message.Message):
    __slots__ = ("stored", "file", "revision")
    STORED_FIELD_NUMBER: _ClassVar[int]
    FILE_FIELD_NUMBER: _ClassVar[int]
    REVISION_FIELD_NUMBER: _ClassVar[int]
    stored: _agent_operations_pb2.AgentOpsOperationBinding
    file: str
    revision: str
    def __init__(self, stored: _Optional[_Union[_agent_operations_pb2.AgentOpsOperationBinding, _Mapping]] = ..., file: _Optional[str] = ..., revision: _Optional[str] = ...) -> None: ...

class AgentOpsDeleteBindingOverrideRequest(_message.Message):
    __slots__ = ("owner", "operation", "operation_version")
    OWNER_FIELD_NUMBER: _ClassVar[int]
    OPERATION_FIELD_NUMBER: _ClassVar[int]
    OPERATION_VERSION_FIELD_NUMBER: _ClassVar[int]
    owner: AgentOpsTargetSelector
    operation: str
    operation_version: str
    def __init__(self, owner: _Optional[_Union[AgentOpsTargetSelector, _Mapping]] = ..., operation: _Optional[str] = ..., operation_version: _Optional[str] = ...) -> None: ...

class AgentOpsDeleteBindingOverrideResponse(_message.Message):
    __slots__ = ("found",)
    FOUND_FIELD_NUMBER: _ClassVar[int]
    found: bool
    def __init__(self, found: _Optional[bool] = ...) -> None: ...

class AgentOpsGetWorkflowProjectionRequest(_message.Message):
    __slots__ = ("target",)
    TARGET_FIELD_NUMBER: _ClassVar[int]
    target: AgentOpsTargetSelector
    def __init__(self, target: _Optional[_Union[AgentOpsTargetSelector, _Mapping]] = ...) -> None: ...

class AgentOpsOperationProjection(_message.Message):
    __slots__ = ("operation", "operation_version", "execution_id", "run_id", "state", "outcome", "idempotency_key", "provenance_digest", "mode", "mode_revision", "binding_layer", "binding_owner_kind", "binding_owner_id", "recorded_at", "snapshot_found", "attempt", "prior_execution_id", "legacy_import")
    OPERATION_FIELD_NUMBER: _ClassVar[int]
    OPERATION_VERSION_FIELD_NUMBER: _ClassVar[int]
    EXECUTION_ID_FIELD_NUMBER: _ClassVar[int]
    RUN_ID_FIELD_NUMBER: _ClassVar[int]
    STATE_FIELD_NUMBER: _ClassVar[int]
    OUTCOME_FIELD_NUMBER: _ClassVar[int]
    IDEMPOTENCY_KEY_FIELD_NUMBER: _ClassVar[int]
    PROVENANCE_DIGEST_FIELD_NUMBER: _ClassVar[int]
    MODE_FIELD_NUMBER: _ClassVar[int]
    MODE_REVISION_FIELD_NUMBER: _ClassVar[int]
    BINDING_LAYER_FIELD_NUMBER: _ClassVar[int]
    BINDING_OWNER_KIND_FIELD_NUMBER: _ClassVar[int]
    BINDING_OWNER_ID_FIELD_NUMBER: _ClassVar[int]
    RECORDED_AT_FIELD_NUMBER: _ClassVar[int]
    SNAPSHOT_FOUND_FIELD_NUMBER: _ClassVar[int]
    ATTEMPT_FIELD_NUMBER: _ClassVar[int]
    PRIOR_EXECUTION_ID_FIELD_NUMBER: _ClassVar[int]
    LEGACY_IMPORT_FIELD_NUMBER: _ClassVar[int]
    operation: str
    operation_version: str
    execution_id: str
    run_id: str
    state: str
    outcome: str
    idempotency_key: str
    provenance_digest: str
    mode: str
    mode_revision: str
    binding_layer: _agent_operations_pb2.AgentOpsBindingLayer
    binding_owner_kind: str
    binding_owner_id: str
    recorded_at: str
    snapshot_found: bool
    attempt: int
    prior_execution_id: str
    legacy_import: bool
    def __init__(self, operation: _Optional[str] = ..., operation_version: _Optional[str] = ..., execution_id: _Optional[str] = ..., run_id: _Optional[str] = ..., state: _Optional[str] = ..., outcome: _Optional[str] = ..., idempotency_key: _Optional[str] = ..., provenance_digest: _Optional[str] = ..., mode: _Optional[str] = ..., mode_revision: _Optional[str] = ..., binding_layer: _Optional[_Union[_agent_operations_pb2.AgentOpsBindingLayer, str]] = ..., binding_owner_kind: _Optional[str] = ..., binding_owner_id: _Optional[str] = ..., recorded_at: _Optional[str] = ..., snapshot_found: _Optional[bool] = ..., attempt: _Optional[int] = ..., prior_execution_id: _Optional[str] = ..., legacy_import: _Optional[bool] = ...) -> None: ...

class AgentOpsGetWorkflowProjectionResponse(_message.Message):
    __slots__ = ("found", "workflow", "operations", "policy_id", "policy_revision")
    FOUND_FIELD_NUMBER: _ClassVar[int]
    WORKFLOW_FIELD_NUMBER: _ClassVar[int]
    OPERATIONS_FIELD_NUMBER: _ClassVar[int]
    POLICY_ID_FIELD_NUMBER: _ClassVar[int]
    POLICY_REVISION_FIELD_NUMBER: _ClassVar[int]
    found: bool
    workflow: _agent_operations_pb2.AgentOpsWorkflowInstance
    operations: _containers.RepeatedCompositeFieldContainer[AgentOpsOperationProjection]
    policy_id: str
    policy_revision: str
    def __init__(self, found: _Optional[bool] = ..., workflow: _Optional[_Union[_agent_operations_pb2.AgentOpsWorkflowInstance, _Mapping]] = ..., operations: _Optional[_Iterable[_Union[AgentOpsOperationProjection, _Mapping]]] = ..., policy_id: _Optional[str] = ..., policy_revision: _Optional[str] = ...) -> None: ...

class AgentOpsListExecutionHistoryRequest(_message.Message):
    __slots__ = ("target", "limit")
    TARGET_FIELD_NUMBER: _ClassVar[int]
    LIMIT_FIELD_NUMBER: _ClassVar[int]
    target: AgentOpsTargetSelector
    limit: int
    def __init__(self, target: _Optional[_Union[AgentOpsTargetSelector, _Mapping]] = ..., limit: _Optional[int] = ...) -> None: ...

class AgentOpsExecutionSummary(_message.Message):
    __slots__ = ("execution_id", "operation", "operation_version", "mode", "mode_revision", "binding_layer", "compiled_mode_digest", "prompt_catalog_digest", "caller_input_digest", "outcome", "reproducible", "recorded_at", "legacy_import")
    EXECUTION_ID_FIELD_NUMBER: _ClassVar[int]
    OPERATION_FIELD_NUMBER: _ClassVar[int]
    OPERATION_VERSION_FIELD_NUMBER: _ClassVar[int]
    MODE_FIELD_NUMBER: _ClassVar[int]
    MODE_REVISION_FIELD_NUMBER: _ClassVar[int]
    BINDING_LAYER_FIELD_NUMBER: _ClassVar[int]
    COMPILED_MODE_DIGEST_FIELD_NUMBER: _ClassVar[int]
    PROMPT_CATALOG_DIGEST_FIELD_NUMBER: _ClassVar[int]
    CALLER_INPUT_DIGEST_FIELD_NUMBER: _ClassVar[int]
    OUTCOME_FIELD_NUMBER: _ClassVar[int]
    REPRODUCIBLE_FIELD_NUMBER: _ClassVar[int]
    RECORDED_AT_FIELD_NUMBER: _ClassVar[int]
    LEGACY_IMPORT_FIELD_NUMBER: _ClassVar[int]
    execution_id: str
    operation: str
    operation_version: str
    mode: str
    mode_revision: str
    binding_layer: _agent_operations_pb2.AgentOpsBindingLayer
    compiled_mode_digest: str
    prompt_catalog_digest: str
    caller_input_digest: str
    outcome: str
    reproducible: bool
    recorded_at: str
    legacy_import: bool
    def __init__(self, execution_id: _Optional[str] = ..., operation: _Optional[str] = ..., operation_version: _Optional[str] = ..., mode: _Optional[str] = ..., mode_revision: _Optional[str] = ..., binding_layer: _Optional[_Union[_agent_operations_pb2.AgentOpsBindingLayer, str]] = ..., compiled_mode_digest: _Optional[str] = ..., prompt_catalog_digest: _Optional[str] = ..., caller_input_digest: _Optional[str] = ..., outcome: _Optional[str] = ..., reproducible: _Optional[bool] = ..., recorded_at: _Optional[str] = ..., legacy_import: _Optional[bool] = ...) -> None: ...

class AgentOpsListExecutionHistoryResponse(_message.Message):
    __slots__ = ("executions",)
    EXECUTIONS_FIELD_NUMBER: _ClassVar[int]
    executions: _containers.RepeatedCompositeFieldContainer[AgentOpsExecutionSummary]
    def __init__(self, executions: _Optional[_Iterable[_Union[AgentOpsExecutionSummary, _Mapping]]] = ...) -> None: ...

class AgentOpsGetMigrationStatusRequest(_message.Message):
    __slots__ = ()
    def __init__(self) -> None: ...

class AgentOpsRunReconciliationRequest(_message.Message):
    __slots__ = ()
    def __init__(self) -> None: ...

class AgentOpsRunReconciliationResponse(_message.Message):
    __slots__ = ("dirs_scanned", "snapshots_seen", "reaped", "skipped_too_recent")
    DIRS_SCANNED_FIELD_NUMBER: _ClassVar[int]
    SNAPSHOTS_SEEN_FIELD_NUMBER: _ClassVar[int]
    REAPED_FIELD_NUMBER: _ClassVar[int]
    SKIPPED_TOO_RECENT_FIELD_NUMBER: _ClassVar[int]
    dirs_scanned: int
    snapshots_seen: int
    reaped: _containers.RepeatedScalarFieldContainer[str]
    skipped_too_recent: int
    def __init__(self, dirs_scanned: _Optional[int] = ..., snapshots_seen: _Optional[int] = ..., reaped: _Optional[_Iterable[str]] = ..., skipped_too_recent: _Optional[int] = ...) -> None: ...

class AgentOpsGetMigrationStatusResponse(_message.Message):
    __slots__ = ("state", "epoch", "staged_count", "promoted_count", "quarantined_count", "started_at", "updated_at", "report_path", "document_found")
    STATE_FIELD_NUMBER: _ClassVar[int]
    EPOCH_FIELD_NUMBER: _ClassVar[int]
    STAGED_COUNT_FIELD_NUMBER: _ClassVar[int]
    PROMOTED_COUNT_FIELD_NUMBER: _ClassVar[int]
    QUARANTINED_COUNT_FIELD_NUMBER: _ClassVar[int]
    STARTED_AT_FIELD_NUMBER: _ClassVar[int]
    UPDATED_AT_FIELD_NUMBER: _ClassVar[int]
    REPORT_PATH_FIELD_NUMBER: _ClassVar[int]
    DOCUMENT_FOUND_FIELD_NUMBER: _ClassVar[int]
    state: str
    epoch: int
    staged_count: int
    promoted_count: int
    quarantined_count: int
    started_at: str
    updated_at: str
    report_path: str
    document_found: bool
    def __init__(self, state: _Optional[str] = ..., epoch: _Optional[int] = ..., staged_count: _Optional[int] = ..., promoted_count: _Optional[int] = ..., quarantined_count: _Optional[int] = ..., started_at: _Optional[str] = ..., updated_at: _Optional[str] = ..., report_path: _Optional[str] = ..., document_found: _Optional[bool] = ...) -> None: ...
