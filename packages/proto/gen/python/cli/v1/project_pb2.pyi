from cli.v1 import resource_list_pb2 as _resource_list_pb2
from google.protobuf import struct_pb2 as _struct_pb2
from google.protobuf.internal import containers as _containers
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Iterable as _Iterable, Mapping as _Mapping
from typing import ClassVar as _ClassVar, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class ProjectStatusResponse(_message.Message):
    __slots__ = ("success", "status")
    SUCCESS_FIELD_NUMBER: _ClassVar[int]
    STATUS_FIELD_NUMBER: _ClassVar[int]
    success: bool
    status: ProjectStatusReport
    def __init__(self, success: _Optional[bool] = ..., status: _Optional[_Union[ProjectStatusReport, _Mapping]] = ...) -> None: ...

class ProjectStatusReport(_message.Message):
    __slots__ = ("resources", "scenarios", "maintenance", "summary")
    class SummaryEntry(_message.Message):
        __slots__ = ("key", "value")
        KEY_FIELD_NUMBER: _ClassVar[int]
        VALUE_FIELD_NUMBER: _ClassVar[int]
        key: str
        value: int
        def __init__(self, key: _Optional[str] = ..., value: _Optional[int] = ...) -> None: ...
    RESOURCES_FIELD_NUMBER: _ClassVar[int]
    SCENARIOS_FIELD_NUMBER: _ClassVar[int]
    MAINTENANCE_FIELD_NUMBER: _ClassVar[int]
    SUMMARY_FIELD_NUMBER: _ClassVar[int]
    resources: _containers.RepeatedCompositeFieldContainer[ProjectResourceStatus]
    scenarios: _containers.RepeatedCompositeFieldContainer[ProjectScenarioStatus]
    maintenance: ProjectProcessSnapshot
    summary: _containers.ScalarMap[str, int]
    def __init__(self, resources: _Optional[_Iterable[_Union[ProjectResourceStatus, _Mapping]]] = ..., scenarios: _Optional[_Iterable[_Union[ProjectScenarioStatus, _Mapping]]] = ..., maintenance: _Optional[_Union[ProjectProcessSnapshot, _Mapping]] = ..., summary: _Optional[_Mapping[str, int]] = ...) -> None: ...

class ProjectResourceStatus(_message.Message):
    __slots__ = ("resource", "installed", "running", "healthy", "health", "status_code", "message", "probe_error", "raw")
    RESOURCE_FIELD_NUMBER: _ClassVar[int]
    INSTALLED_FIELD_NUMBER: _ClassVar[int]
    RUNNING_FIELD_NUMBER: _ClassVar[int]
    HEALTHY_FIELD_NUMBER: _ClassVar[int]
    HEALTH_FIELD_NUMBER: _ClassVar[int]
    STATUS_CODE_FIELD_NUMBER: _ClassVar[int]
    MESSAGE_FIELD_NUMBER: _ClassVar[int]
    PROBE_ERROR_FIELD_NUMBER: _ClassVar[int]
    RAW_FIELD_NUMBER: _ClassVar[int]
    resource: _resource_list_pb2.Resource
    installed: bool
    running: bool
    healthy: _struct_pb2.Value
    health: str
    status_code: str
    message: str
    probe_error: str
    raw: _struct_pb2.Value
    def __init__(self, resource: _Optional[_Union[_resource_list_pb2.Resource, _Mapping]] = ..., installed: _Optional[bool] = ..., running: _Optional[bool] = ..., healthy: _Optional[_Union[_struct_pb2.Value, _Mapping]] = ..., health: _Optional[str] = ..., status_code: _Optional[str] = ..., message: _Optional[str] = ..., probe_error: _Optional[str] = ..., raw: _Optional[_Union[_struct_pb2.Value, _Mapping]] = ...) -> None: ...

class ProjectScenarioStatus(_message.Message):
    __slots__ = ("name", "display_name", "description", "tags", "status", "processes", "started_at", "runtime", "ports", "health_status")
    class PortsEntry(_message.Message):
        __slots__ = ("key", "value")
        KEY_FIELD_NUMBER: _ClassVar[int]
        VALUE_FIELD_NUMBER: _ClassVar[int]
        key: str
        value: int
        def __init__(self, key: _Optional[str] = ..., value: _Optional[int] = ...) -> None: ...
    NAME_FIELD_NUMBER: _ClassVar[int]
    DISPLAY_NAME_FIELD_NUMBER: _ClassVar[int]
    DESCRIPTION_FIELD_NUMBER: _ClassVar[int]
    TAGS_FIELD_NUMBER: _ClassVar[int]
    STATUS_FIELD_NUMBER: _ClassVar[int]
    PROCESSES_FIELD_NUMBER: _ClassVar[int]
    STARTED_AT_FIELD_NUMBER: _ClassVar[int]
    RUNTIME_FIELD_NUMBER: _ClassVar[int]
    PORTS_FIELD_NUMBER: _ClassVar[int]
    HEALTH_STATUS_FIELD_NUMBER: _ClassVar[int]
    name: str
    display_name: str
    description: str
    tags: _containers.RepeatedScalarFieldContainer[str]
    status: str
    processes: int
    started_at: str
    runtime: str
    ports: _containers.ScalarMap[str, int]
    health_status: _struct_pb2.Value
    def __init__(self, name: _Optional[str] = ..., display_name: _Optional[str] = ..., description: _Optional[str] = ..., tags: _Optional[_Iterable[str]] = ..., status: _Optional[str] = ..., processes: _Optional[int] = ..., started_at: _Optional[str] = ..., runtime: _Optional[str] = ..., ports: _Optional[_Mapping[str, int]] = ..., health_status: _Optional[_Union[_struct_pb2.Value, _Mapping]] = ...) -> None: ...

class ProjectProcessSnapshot(_message.Message):
    __slots__ = ("tracked_processes", "running_tracked", "child_processes", "total_processes", "zombie_processes", "orphan_processes", "orphans")
    TRACKED_PROCESSES_FIELD_NUMBER: _ClassVar[int]
    RUNNING_TRACKED_FIELD_NUMBER: _ClassVar[int]
    CHILD_PROCESSES_FIELD_NUMBER: _ClassVar[int]
    TOTAL_PROCESSES_FIELD_NUMBER: _ClassVar[int]
    ZOMBIE_PROCESSES_FIELD_NUMBER: _ClassVar[int]
    ORPHAN_PROCESSES_FIELD_NUMBER: _ClassVar[int]
    ORPHANS_FIELD_NUMBER: _ClassVar[int]
    tracked_processes: int
    running_tracked: int
    child_processes: int
    total_processes: int
    zombie_processes: int
    orphan_processes: int
    orphans: _containers.RepeatedCompositeFieldContainer[ProjectSystemProcess]
    def __init__(self, tracked_processes: _Optional[int] = ..., running_tracked: _Optional[int] = ..., child_processes: _Optional[int] = ..., total_processes: _Optional[int] = ..., zombie_processes: _Optional[int] = ..., orphan_processes: _Optional[int] = ..., orphans: _Optional[_Iterable[_Union[ProjectSystemProcess, _Mapping]]] = ...) -> None: ...

class ProjectSystemProcess(_message.Message):
    __slots__ = ("pid", "ppid", "command")
    PID_FIELD_NUMBER: _ClassVar[int]
    PPID_FIELD_NUMBER: _ClassVar[int]
    COMMAND_FIELD_NUMBER: _ClassVar[int]
    pid: int
    ppid: int
    command: str
    def __init__(self, pid: _Optional[int] = ..., ppid: _Optional[int] = ..., command: _Optional[str] = ...) -> None: ...

class ProjectDoctorResponse(_message.Message):
    __slots__ = ("success", "checks")
    SUCCESS_FIELD_NUMBER: _ClassVar[int]
    CHECKS_FIELD_NUMBER: _ClassVar[int]
    success: bool
    checks: _containers.RepeatedCompositeFieldContainer[ProjectDoctorCheck]
    def __init__(self, success: _Optional[bool] = ..., checks: _Optional[_Iterable[_Union[ProjectDoctorCheck, _Mapping]]] = ...) -> None: ...

class ProjectDoctorCheck(_message.Message):
    __slots__ = ("name", "status", "message")
    NAME_FIELD_NUMBER: _ClassVar[int]
    STATUS_FIELD_NUMBER: _ClassVar[int]
    MESSAGE_FIELD_NUMBER: _ClassVar[int]
    name: str
    status: str
    message: str
    def __init__(self, name: _Optional[str] = ..., status: _Optional[str] = ..., message: _Optional[str] = ...) -> None: ...

class ProjectStopResponse(_message.Message):
    __slots__ = ("success", "data")
    SUCCESS_FIELD_NUMBER: _ClassVar[int]
    DATA_FIELD_NUMBER: _ClassVar[int]
    success: bool
    data: ProjectStopReport
    def __init__(self, success: _Optional[bool] = ..., data: _Optional[_Union[ProjectStopReport, _Mapping]] = ...) -> None: ...

class ProjectStopReport(_message.Message):
    __slots__ = ("stopped", "failed", "message")
    STOPPED_FIELD_NUMBER: _ClassVar[int]
    FAILED_FIELD_NUMBER: _ClassVar[int]
    MESSAGE_FIELD_NUMBER: _ClassVar[int]
    stopped: _containers.RepeatedCompositeFieldContainer[ProjectStopItem]
    failed: _containers.RepeatedCompositeFieldContainer[ProjectStopItem]
    message: str
    def __init__(self, stopped: _Optional[_Iterable[_Union[ProjectStopItem, _Mapping]]] = ..., failed: _Optional[_Iterable[_Union[ProjectStopItem, _Mapping]]] = ..., message: _Optional[str] = ...) -> None: ...

class ProjectStopItem(_message.Message):
    __slots__ = ("name", "message", "error")
    NAME_FIELD_NUMBER: _ClassVar[int]
    MESSAGE_FIELD_NUMBER: _ClassVar[int]
    ERROR_FIELD_NUMBER: _ClassVar[int]
    name: str
    message: str
    error: str
    def __init__(self, name: _Optional[str] = ..., message: _Optional[str] = ..., error: _Optional[str] = ...) -> None: ...

class ProjectOrphansResponse(_message.Message):
    __slots__ = ("success", "orphans")
    SUCCESS_FIELD_NUMBER: _ClassVar[int]
    ORPHANS_FIELD_NUMBER: _ClassVar[int]
    success: bool
    orphans: _containers.RepeatedCompositeFieldContainer[ProjectSystemProcess]
    def __init__(self, success: _Optional[bool] = ..., orphans: _Optional[_Iterable[_Union[ProjectSystemProcess, _Mapping]]] = ...) -> None: ...

class ProjectOrphansDryRunResponse(_message.Message):
    __slots__ = ("success", "dry_run")
    SUCCESS_FIELD_NUMBER: _ClassVar[int]
    DRY_RUN_FIELD_NUMBER: _ClassVar[int]
    success: bool
    dry_run: ProjectOrphansDryRun
    def __init__(self, success: _Optional[bool] = ..., dry_run: _Optional[_Union[ProjectOrphansDryRun, _Mapping]] = ...) -> None: ...

class ProjectOrphansDryRun(_message.Message):
    __slots__ = ("orphans",)
    ORPHANS_FIELD_NUMBER: _ClassVar[int]
    orphans: _containers.RepeatedCompositeFieldContainer[ProjectSystemProcess]
    def __init__(self, orphans: _Optional[_Iterable[_Union[ProjectSystemProcess, _Mapping]]] = ...) -> None: ...

class ProjectLocksResponse(_message.Message):
    __slots__ = ("success", "registry_claims")
    SUCCESS_FIELD_NUMBER: _ClassVar[int]
    REGISTRY_CLAIMS_FIELD_NUMBER: _ClassVar[int]
    success: bool
    registry_claims: _containers.RepeatedCompositeFieldContainer[ProjectRuntimeClaim]
    def __init__(self, success: _Optional[bool] = ..., registry_claims: _Optional[_Iterable[_Union[ProjectRuntimeClaim, _Mapping]]] = ...) -> None: ...

class ProjectRuntimeClaim(_message.Message):
    __slots__ = ("claim_id", "instance_id", "scenario", "generation", "port_name", "env_var", "port", "bind_host", "url", "claim_status", "instance_status", "supervisor_id", "supervisor_status", "supervisor_fresh", "lease_fresh", "heartbeat_deadline", "supervisor_heartbeat_deadline", "health_status", "health_ready", "reconciliation", "reconcile_reason", "authoritative", "created_at", "updated_at", "expires_at", "last_bound_at", "last_listener_check_at", "last_listener_seen_at", "first_unbound_at", "consecutive_listener_misses", "listener_status", "listener_pid", "listener_process_label", "recommendation_code", "recommendation_confidence", "recommendation_rationale")
    CLAIM_ID_FIELD_NUMBER: _ClassVar[int]
    INSTANCE_ID_FIELD_NUMBER: _ClassVar[int]
    SCENARIO_FIELD_NUMBER: _ClassVar[int]
    GENERATION_FIELD_NUMBER: _ClassVar[int]
    PORT_NAME_FIELD_NUMBER: _ClassVar[int]
    ENV_VAR_FIELD_NUMBER: _ClassVar[int]
    PORT_FIELD_NUMBER: _ClassVar[int]
    BIND_HOST_FIELD_NUMBER: _ClassVar[int]
    URL_FIELD_NUMBER: _ClassVar[int]
    CLAIM_STATUS_FIELD_NUMBER: _ClassVar[int]
    INSTANCE_STATUS_FIELD_NUMBER: _ClassVar[int]
    SUPERVISOR_ID_FIELD_NUMBER: _ClassVar[int]
    SUPERVISOR_STATUS_FIELD_NUMBER: _ClassVar[int]
    SUPERVISOR_FRESH_FIELD_NUMBER: _ClassVar[int]
    LEASE_FRESH_FIELD_NUMBER: _ClassVar[int]
    HEARTBEAT_DEADLINE_FIELD_NUMBER: _ClassVar[int]
    SUPERVISOR_HEARTBEAT_DEADLINE_FIELD_NUMBER: _ClassVar[int]
    HEALTH_STATUS_FIELD_NUMBER: _ClassVar[int]
    HEALTH_READY_FIELD_NUMBER: _ClassVar[int]
    RECONCILIATION_FIELD_NUMBER: _ClassVar[int]
    RECONCILE_REASON_FIELD_NUMBER: _ClassVar[int]
    AUTHORITATIVE_FIELD_NUMBER: _ClassVar[int]
    CREATED_AT_FIELD_NUMBER: _ClassVar[int]
    UPDATED_AT_FIELD_NUMBER: _ClassVar[int]
    EXPIRES_AT_FIELD_NUMBER: _ClassVar[int]
    LAST_BOUND_AT_FIELD_NUMBER: _ClassVar[int]
    LAST_LISTENER_CHECK_AT_FIELD_NUMBER: _ClassVar[int]
    LAST_LISTENER_SEEN_AT_FIELD_NUMBER: _ClassVar[int]
    FIRST_UNBOUND_AT_FIELD_NUMBER: _ClassVar[int]
    CONSECUTIVE_LISTENER_MISSES_FIELD_NUMBER: _ClassVar[int]
    LISTENER_STATUS_FIELD_NUMBER: _ClassVar[int]
    LISTENER_PID_FIELD_NUMBER: _ClassVar[int]
    LISTENER_PROCESS_LABEL_FIELD_NUMBER: _ClassVar[int]
    RECOMMENDATION_CODE_FIELD_NUMBER: _ClassVar[int]
    RECOMMENDATION_CONFIDENCE_FIELD_NUMBER: _ClassVar[int]
    RECOMMENDATION_RATIONALE_FIELD_NUMBER: _ClassVar[int]
    claim_id: str
    instance_id: str
    scenario: str
    generation: int
    port_name: str
    env_var: str
    port: int
    bind_host: str
    url: str
    claim_status: str
    instance_status: str
    supervisor_id: str
    supervisor_status: str
    supervisor_fresh: _struct_pb2.Value
    lease_fresh: _struct_pb2.Value
    heartbeat_deadline: str
    supervisor_heartbeat_deadline: str
    health_status: str
    health_ready: _struct_pb2.Value
    reconciliation: str
    reconcile_reason: str
    authoritative: _struct_pb2.Value
    created_at: str
    updated_at: str
    expires_at: str
    last_bound_at: str
    last_listener_check_at: str
    last_listener_seen_at: str
    first_unbound_at: str
    consecutive_listener_misses: int
    listener_status: str
    listener_pid: _struct_pb2.Value
    listener_process_label: str
    recommendation_code: str
    recommendation_confidence: str
    recommendation_rationale: str
    def __init__(self, claim_id: _Optional[str] = ..., instance_id: _Optional[str] = ..., scenario: _Optional[str] = ..., generation: _Optional[int] = ..., port_name: _Optional[str] = ..., env_var: _Optional[str] = ..., port: _Optional[int] = ..., bind_host: _Optional[str] = ..., url: _Optional[str] = ..., claim_status: _Optional[str] = ..., instance_status: _Optional[str] = ..., supervisor_id: _Optional[str] = ..., supervisor_status: _Optional[str] = ..., supervisor_fresh: _Optional[_Union[_struct_pb2.Value, _Mapping]] = ..., lease_fresh: _Optional[_Union[_struct_pb2.Value, _Mapping]] = ..., heartbeat_deadline: _Optional[str] = ..., supervisor_heartbeat_deadline: _Optional[str] = ..., health_status: _Optional[str] = ..., health_ready: _Optional[_Union[_struct_pb2.Value, _Mapping]] = ..., reconciliation: _Optional[str] = ..., reconcile_reason: _Optional[str] = ..., authoritative: _Optional[_Union[_struct_pb2.Value, _Mapping]] = ..., created_at: _Optional[str] = ..., updated_at: _Optional[str] = ..., expires_at: _Optional[str] = ..., last_bound_at: _Optional[str] = ..., last_listener_check_at: _Optional[str] = ..., last_listener_seen_at: _Optional[str] = ..., first_unbound_at: _Optional[str] = ..., consecutive_listener_misses: _Optional[int] = ..., listener_status: _Optional[str] = ..., listener_pid: _Optional[_Union[_struct_pb2.Value, _Mapping]] = ..., listener_process_label: _Optional[str] = ..., recommendation_code: _Optional[str] = ..., recommendation_confidence: _Optional[str] = ..., recommendation_rationale: _Optional[str] = ...) -> None: ...

class ProjectTemplateCleanupResponse(_message.Message):
    __slots__ = ("success", "cleanup")
    SUCCESS_FIELD_NUMBER: _ClassVar[int]
    CLEANUP_FIELD_NUMBER: _ClassVar[int]
    success: bool
    cleanup: ProjectTemplateCleanupResult
    def __init__(self, success: _Optional[bool] = ..., cleanup: _Optional[_Union[ProjectTemplateCleanupResult, _Mapping]] = ...) -> None: ...

class ProjectTemplateCleanupResult(_message.Message):
    __slots__ = ("dry_run", "older_than", "include_retained", "run_id", "eligible", "skipped", "failures", "removed", "needs_proto_generate", "proto_generate_ran", "message")
    DRY_RUN_FIELD_NUMBER: _ClassVar[int]
    OLDER_THAN_FIELD_NUMBER: _ClassVar[int]
    INCLUDE_RETAINED_FIELD_NUMBER: _ClassVar[int]
    RUN_ID_FIELD_NUMBER: _ClassVar[int]
    ELIGIBLE_FIELD_NUMBER: _ClassVar[int]
    SKIPPED_FIELD_NUMBER: _ClassVar[int]
    FAILURES_FIELD_NUMBER: _ClassVar[int]
    REMOVED_FIELD_NUMBER: _ClassVar[int]
    NEEDS_PROTO_GENERATE_FIELD_NUMBER: _ClassVar[int]
    PROTO_GENERATE_RAN_FIELD_NUMBER: _ClassVar[int]
    MESSAGE_FIELD_NUMBER: _ClassVar[int]
    dry_run: bool
    older_than: int
    include_retained: bool
    run_id: str
    eligible: _containers.RepeatedCompositeFieldContainer[ProjectTemplateRun]
    skipped: _containers.RepeatedCompositeFieldContainer[ProjectTemplateSkippedRun]
    failures: _containers.RepeatedCompositeFieldContainer[ProjectTemplateFailedRun]
    removed: _containers.RepeatedCompositeFieldContainer[ProjectTemplateRun]
    needs_proto_generate: bool
    proto_generate_ran: bool
    message: str
    def __init__(self, dry_run: _Optional[bool] = ..., older_than: _Optional[int] = ..., include_retained: _Optional[bool] = ..., run_id: _Optional[str] = ..., eligible: _Optional[_Iterable[_Union[ProjectTemplateRun, _Mapping]]] = ..., skipped: _Optional[_Iterable[_Union[ProjectTemplateSkippedRun, _Mapping]]] = ..., failures: _Optional[_Iterable[_Union[ProjectTemplateFailedRun, _Mapping]]] = ..., removed: _Optional[_Iterable[_Union[ProjectTemplateRun, _Mapping]]] = ..., needs_proto_generate: _Optional[bool] = ..., proto_generate_ran: _Optional[bool] = ..., message: _Optional[str] = ...) -> None: ...

class ProjectTemplateRun(_message.Message):
    __slots__ = ("marker_path", "marker", "age")
    MARKER_PATH_FIELD_NUMBER: _ClassVar[int]
    MARKER_FIELD_NUMBER: _ClassVar[int]
    AGE_FIELD_NUMBER: _ClassVar[int]
    marker_path: str
    marker: ProjectTemplateRunMarker
    age: str
    def __init__(self, marker_path: _Optional[str] = ..., marker: _Optional[_Union[ProjectTemplateRunMarker, _Mapping]] = ..., age: _Optional[str] = ...) -> None: ...

class ProjectTemplateRunMarker(_message.Message):
    __slots__ = ("version", "run_id", "repo_root", "template", "scenario_id", "scenario_path", "temp_root", "created_at", "retained", "creator_pid", "completed", "cleanup_status", "relocation_artifacts")
    VERSION_FIELD_NUMBER: _ClassVar[int]
    RUN_ID_FIELD_NUMBER: _ClassVar[int]
    REPO_ROOT_FIELD_NUMBER: _ClassVar[int]
    TEMPLATE_FIELD_NUMBER: _ClassVar[int]
    SCENARIO_ID_FIELD_NUMBER: _ClassVar[int]
    SCENARIO_PATH_FIELD_NUMBER: _ClassVar[int]
    TEMP_ROOT_FIELD_NUMBER: _ClassVar[int]
    CREATED_AT_FIELD_NUMBER: _ClassVar[int]
    RETAINED_FIELD_NUMBER: _ClassVar[int]
    CREATOR_PID_FIELD_NUMBER: _ClassVar[int]
    COMPLETED_FIELD_NUMBER: _ClassVar[int]
    CLEANUP_STATUS_FIELD_NUMBER: _ClassVar[int]
    RELOCATION_ARTIFACTS_FIELD_NUMBER: _ClassVar[int]
    version: str
    run_id: str
    repo_root: str
    template: str
    scenario_id: str
    scenario_path: str
    temp_root: str
    created_at: str
    retained: bool
    creator_pid: int
    completed: bool
    cleanup_status: str
    relocation_artifacts: _containers.RepeatedScalarFieldContainer[str]
    def __init__(self, version: _Optional[str] = ..., run_id: _Optional[str] = ..., repo_root: _Optional[str] = ..., template: _Optional[str] = ..., scenario_id: _Optional[str] = ..., scenario_path: _Optional[str] = ..., temp_root: _Optional[str] = ..., created_at: _Optional[str] = ..., retained: _Optional[bool] = ..., creator_pid: _Optional[int] = ..., completed: _Optional[bool] = ..., cleanup_status: _Optional[str] = ..., relocation_artifacts: _Optional[_Iterable[str]] = ...) -> None: ...

class ProjectTemplateSkippedRun(_message.Message):
    __slots__ = ("run", "path", "reason")
    RUN_FIELD_NUMBER: _ClassVar[int]
    PATH_FIELD_NUMBER: _ClassVar[int]
    REASON_FIELD_NUMBER: _ClassVar[int]
    run: ProjectTemplateRun
    path: str
    reason: str
    def __init__(self, run: _Optional[_Union[ProjectTemplateRun, _Mapping]] = ..., path: _Optional[str] = ..., reason: _Optional[str] = ...) -> None: ...

class ProjectTemplateFailedRun(_message.Message):
    __slots__ = ("run", "path", "error")
    RUN_FIELD_NUMBER: _ClassVar[int]
    PATH_FIELD_NUMBER: _ClassVar[int]
    ERROR_FIELD_NUMBER: _ClassVar[int]
    run: ProjectTemplateRun
    path: str
    error: str
    def __init__(self, run: _Optional[_Union[ProjectTemplateRun, _Mapping]] = ..., path: _Optional[str] = ..., error: _Optional[str] = ...) -> None: ...

class ProjectPortDiagnosticResponse(_message.Message):
    __slots__ = ("success", "diagnostic")
    SUCCESS_FIELD_NUMBER: _ClassVar[int]
    DIAGNOSTIC_FIELD_NUMBER: _ClassVar[int]
    success: bool
    diagnostic: ProjectPortDiagnostic
    def __init__(self, success: _Optional[bool] = ..., diagnostic: _Optional[_Union[ProjectPortDiagnostic, _Mapping]] = ...) -> None: ...

class ProjectPortDiagnostic(_message.Message):
    __slots__ = ("port", "scenario", "in_use", "listeners", "listener_inspection", "registry_claims", "registry_processes", "host_orphan_count", "recommendations", "port_policy")
    PORT_FIELD_NUMBER: _ClassVar[int]
    SCENARIO_FIELD_NUMBER: _ClassVar[int]
    IN_USE_FIELD_NUMBER: _ClassVar[int]
    LISTENERS_FIELD_NUMBER: _ClassVar[int]
    LISTENER_INSPECTION_FIELD_NUMBER: _ClassVar[int]
    REGISTRY_CLAIMS_FIELD_NUMBER: _ClassVar[int]
    REGISTRY_PROCESSES_FIELD_NUMBER: _ClassVar[int]
    HOST_ORPHAN_COUNT_FIELD_NUMBER: _ClassVar[int]
    RECOMMENDATIONS_FIELD_NUMBER: _ClassVar[int]
    PORT_POLICY_FIELD_NUMBER: _ClassVar[int]
    port: int
    scenario: str
    in_use: bool
    listeners: _containers.RepeatedCompositeFieldContainer[ProjectPortListener]
    listener_inspection: ProjectListenerInspection
    registry_claims: _containers.RepeatedCompositeFieldContainer[ProjectRuntimeClaim]
    registry_processes: _containers.RepeatedCompositeFieldContainer[ProjectRuntimeProcessRef]
    host_orphan_count: int
    recommendations: _containers.RepeatedScalarFieldContainer[str]
    port_policy: ProjectPortPolicy
    def __init__(self, port: _Optional[int] = ..., scenario: _Optional[str] = ..., in_use: _Optional[bool] = ..., listeners: _Optional[_Iterable[_Union[ProjectPortListener, _Mapping]]] = ..., listener_inspection: _Optional[_Union[ProjectListenerInspection, _Mapping]] = ..., registry_claims: _Optional[_Iterable[_Union[ProjectRuntimeClaim, _Mapping]]] = ..., registry_processes: _Optional[_Iterable[_Union[ProjectRuntimeProcessRef, _Mapping]]] = ..., host_orphan_count: _Optional[int] = ..., recommendations: _Optional[_Iterable[str]] = ..., port_policy: _Optional[_Union[ProjectPortPolicy, _Mapping]] = ...) -> None: ...

class ProjectPortListener(_message.Message):
    __slots__ = ("pid", "command", "zombie")
    PID_FIELD_NUMBER: _ClassVar[int]
    COMMAND_FIELD_NUMBER: _ClassVar[int]
    ZOMBIE_FIELD_NUMBER: _ClassVar[int]
    pid: int
    command: str
    zombie: bool
    def __init__(self, pid: _Optional[int] = ..., command: _Optional[str] = ..., zombie: _Optional[bool] = ...) -> None: ...

class ProjectListenerInspection(_message.Message):
    __slots__ = ("available", "tool", "reason")
    AVAILABLE_FIELD_NUMBER: _ClassVar[int]
    TOOL_FIELD_NUMBER: _ClassVar[int]
    REASON_FIELD_NUMBER: _ClassVar[int]
    available: bool
    tool: str
    reason: str
    def __init__(self, available: _Optional[bool] = ..., tool: _Optional[str] = ..., reason: _Optional[str] = ...) -> None: ...

class ProjectRuntimeProcessRef(_message.Message):
    __slots__ = ("ref_id", "instance_id", "scenario", "instance_status", "pid", "pgid", "process_id", "step", "command", "status", "pid_running")
    REF_ID_FIELD_NUMBER: _ClassVar[int]
    INSTANCE_ID_FIELD_NUMBER: _ClassVar[int]
    SCENARIO_FIELD_NUMBER: _ClassVar[int]
    INSTANCE_STATUS_FIELD_NUMBER: _ClassVar[int]
    PID_FIELD_NUMBER: _ClassVar[int]
    PGID_FIELD_NUMBER: _ClassVar[int]
    PROCESS_ID_FIELD_NUMBER: _ClassVar[int]
    STEP_FIELD_NUMBER: _ClassVar[int]
    COMMAND_FIELD_NUMBER: _ClassVar[int]
    STATUS_FIELD_NUMBER: _ClassVar[int]
    PID_RUNNING_FIELD_NUMBER: _ClassVar[int]
    ref_id: str
    instance_id: str
    scenario: str
    instance_status: str
    pid: _struct_pb2.Value
    pgid: _struct_pb2.Value
    process_id: str
    step: str
    command: str
    status: str
    pid_running: _struct_pb2.Value
    def __init__(self, ref_id: _Optional[str] = ..., instance_id: _Optional[str] = ..., scenario: _Optional[str] = ..., instance_status: _Optional[str] = ..., pid: _Optional[_Union[_struct_pb2.Value, _Mapping]] = ..., pgid: _Optional[_Union[_struct_pb2.Value, _Mapping]] = ..., process_id: _Optional[str] = ..., step: _Optional[str] = ..., command: _Optional[str] = ..., status: _Optional[str] = ..., pid_running: _Optional[_Union[_struct_pb2.Value, _Mapping]] = ...) -> None: ...

class ProjectPortPolicy(_message.Message):
    __slots__ = ("ephemeral_min", "ephemeral_max", "ephemeral_source", "inside_ephemeral_range", "canonical_band", "above_canonical_max")
    EPHEMERAL_MIN_FIELD_NUMBER: _ClassVar[int]
    EPHEMERAL_MAX_FIELD_NUMBER: _ClassVar[int]
    EPHEMERAL_SOURCE_FIELD_NUMBER: _ClassVar[int]
    INSIDE_EPHEMERAL_RANGE_FIELD_NUMBER: _ClassVar[int]
    CANONICAL_BAND_FIELD_NUMBER: _ClassVar[int]
    ABOVE_CANONICAL_MAX_FIELD_NUMBER: _ClassVar[int]
    ephemeral_min: int
    ephemeral_max: int
    ephemeral_source: str
    inside_ephemeral_range: bool
    canonical_band: str
    above_canonical_max: bool
    def __init__(self, ephemeral_min: _Optional[int] = ..., ephemeral_max: _Optional[int] = ..., ephemeral_source: _Optional[str] = ..., inside_ephemeral_range: _Optional[bool] = ..., canonical_band: _Optional[str] = ..., above_canonical_max: _Optional[bool] = ...) -> None: ...
