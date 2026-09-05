import datetime

from google.protobuf import timestamp_pb2 as _timestamp_pb2
from google.protobuf.internal import containers as _containers
from google.protobuf.internal import enum_type_wrapper as _enum_type_wrapper
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Iterable as _Iterable, Mapping as _Mapping
from typing import ClassVar as _ClassVar, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class WorkflowKind(int, metaclass=_enum_type_wrapper.EnumTypeWrapper):
    __slots__ = ()
    WORKFLOW_KIND_UNSPECIFIED: _ClassVar[WorkflowKind]
    WORKFLOW_KIND_EXTRACT: _ClassVar[WorkflowKind]
    WORKFLOW_KIND_ADOPT: _ClassVar[WorkflowKind]

class WorkflowStatus(int, metaclass=_enum_type_wrapper.EnumTypeWrapper):
    __slots__ = ()
    WORKFLOW_STATUS_UNSPECIFIED: _ClassVar[WorkflowStatus]
    WORKFLOW_STATUS_QUEUED: _ClassVar[WorkflowStatus]
    WORKFLOW_STATUS_RUNNING: _ClassVar[WorkflowStatus]
    WORKFLOW_STATUS_PARKED: _ClassVar[WorkflowStatus]
    WORKFLOW_STATUS_SUCCEEDED: _ClassVar[WorkflowStatus]
    WORKFLOW_STATUS_FAILED: _ClassVar[WorkflowStatus]
    WORKFLOW_STATUS_STOPPED: _ClassVar[WorkflowStatus]
    WORKFLOW_STATUS_UNAVAILABLE: _ClassVar[WorkflowStatus]
WORKFLOW_KIND_UNSPECIFIED: WorkflowKind
WORKFLOW_KIND_EXTRACT: WorkflowKind
WORKFLOW_KIND_ADOPT: WorkflowKind
WORKFLOW_STATUS_UNSPECIFIED: WorkflowStatus
WORKFLOW_STATUS_QUEUED: WorkflowStatus
WORKFLOW_STATUS_RUNNING: WorkflowStatus
WORKFLOW_STATUS_PARKED: WorkflowStatus
WORKFLOW_STATUS_SUCCEEDED: WorkflowStatus
WORKFLOW_STATUS_FAILED: WorkflowStatus
WORKFLOW_STATUS_STOPPED: WorkflowStatus
WORKFLOW_STATUS_UNAVAILABLE: WorkflowStatus

class Workflow(_message.Message):
    __slots__ = ("id", "kind", "asset_id", "source_scenario", "target_scenario", "source_path", "requested_version", "agent_manager_task_id", "agent_manager_run_id", "idempotency_key", "status", "last_event_sequence", "summary", "error", "created_at", "updated_at", "completed_at", "can_stop", "can_retry", "agent_manager_execution_id")
    ID_FIELD_NUMBER: _ClassVar[int]
    KIND_FIELD_NUMBER: _ClassVar[int]
    ASSET_ID_FIELD_NUMBER: _ClassVar[int]
    SOURCE_SCENARIO_FIELD_NUMBER: _ClassVar[int]
    TARGET_SCENARIO_FIELD_NUMBER: _ClassVar[int]
    SOURCE_PATH_FIELD_NUMBER: _ClassVar[int]
    REQUESTED_VERSION_FIELD_NUMBER: _ClassVar[int]
    AGENT_MANAGER_TASK_ID_FIELD_NUMBER: _ClassVar[int]
    AGENT_MANAGER_RUN_ID_FIELD_NUMBER: _ClassVar[int]
    IDEMPOTENCY_KEY_FIELD_NUMBER: _ClassVar[int]
    STATUS_FIELD_NUMBER: _ClassVar[int]
    LAST_EVENT_SEQUENCE_FIELD_NUMBER: _ClassVar[int]
    SUMMARY_FIELD_NUMBER: _ClassVar[int]
    ERROR_FIELD_NUMBER: _ClassVar[int]
    CREATED_AT_FIELD_NUMBER: _ClassVar[int]
    UPDATED_AT_FIELD_NUMBER: _ClassVar[int]
    COMPLETED_AT_FIELD_NUMBER: _ClassVar[int]
    CAN_STOP_FIELD_NUMBER: _ClassVar[int]
    CAN_RETRY_FIELD_NUMBER: _ClassVar[int]
    AGENT_MANAGER_EXECUTION_ID_FIELD_NUMBER: _ClassVar[int]
    id: str
    kind: WorkflowKind
    asset_id: str
    source_scenario: str
    target_scenario: str
    source_path: str
    requested_version: str
    agent_manager_task_id: str
    agent_manager_run_id: str
    idempotency_key: str
    status: WorkflowStatus
    last_event_sequence: int
    summary: str
    error: str
    created_at: _timestamp_pb2.Timestamp
    updated_at: _timestamp_pb2.Timestamp
    completed_at: _timestamp_pb2.Timestamp
    can_stop: bool
    can_retry: bool
    agent_manager_execution_id: str
    def __init__(self, id: _Optional[str] = ..., kind: _Optional[_Union[WorkflowKind, str]] = ..., asset_id: _Optional[str] = ..., source_scenario: _Optional[str] = ..., target_scenario: _Optional[str] = ..., source_path: _Optional[str] = ..., requested_version: _Optional[str] = ..., agent_manager_task_id: _Optional[str] = ..., agent_manager_run_id: _Optional[str] = ..., idempotency_key: _Optional[str] = ..., status: _Optional[_Union[WorkflowStatus, str]] = ..., last_event_sequence: _Optional[int] = ..., summary: _Optional[str] = ..., error: _Optional[str] = ..., created_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., updated_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., completed_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., can_stop: _Optional[bool] = ..., can_retry: _Optional[bool] = ..., agent_manager_execution_id: _Optional[str] = ...) -> None: ...

class StartWorkflowRequest(_message.Message):
    __slots__ = ("kind", "asset_id", "source_scenario", "target_scenario", "source_path", "requested_version", "idempotency_key", "confirm_overwrite", "override_validation")
    KIND_FIELD_NUMBER: _ClassVar[int]
    ASSET_ID_FIELD_NUMBER: _ClassVar[int]
    SOURCE_SCENARIO_FIELD_NUMBER: _ClassVar[int]
    TARGET_SCENARIO_FIELD_NUMBER: _ClassVar[int]
    SOURCE_PATH_FIELD_NUMBER: _ClassVar[int]
    REQUESTED_VERSION_FIELD_NUMBER: _ClassVar[int]
    IDEMPOTENCY_KEY_FIELD_NUMBER: _ClassVar[int]
    CONFIRM_OVERWRITE_FIELD_NUMBER: _ClassVar[int]
    OVERRIDE_VALIDATION_FIELD_NUMBER: _ClassVar[int]
    kind: WorkflowKind
    asset_id: str
    source_scenario: str
    target_scenario: str
    source_path: str
    requested_version: str
    idempotency_key: str
    confirm_overwrite: bool
    override_validation: bool
    def __init__(self, kind: _Optional[_Union[WorkflowKind, str]] = ..., asset_id: _Optional[str] = ..., source_scenario: _Optional[str] = ..., target_scenario: _Optional[str] = ..., source_path: _Optional[str] = ..., requested_version: _Optional[str] = ..., idempotency_key: _Optional[str] = ..., confirm_overwrite: _Optional[bool] = ..., override_validation: _Optional[bool] = ...) -> None: ...

class StartWorkflowResponse(_message.Message):
    __slots__ = ("workflow", "queue_depth")
    WORKFLOW_FIELD_NUMBER: _ClassVar[int]
    QUEUE_DEPTH_FIELD_NUMBER: _ClassVar[int]
    workflow: Workflow
    queue_depth: int
    def __init__(self, workflow: _Optional[_Union[Workflow, _Mapping]] = ..., queue_depth: _Optional[int] = ...) -> None: ...

class ListWorkflowsRequest(_message.Message):
    __slots__ = ("asset_id", "target_scenario", "active_only", "limit")
    ASSET_ID_FIELD_NUMBER: _ClassVar[int]
    TARGET_SCENARIO_FIELD_NUMBER: _ClassVar[int]
    ACTIVE_ONLY_FIELD_NUMBER: _ClassVar[int]
    LIMIT_FIELD_NUMBER: _ClassVar[int]
    asset_id: str
    target_scenario: str
    active_only: bool
    limit: int
    def __init__(self, asset_id: _Optional[str] = ..., target_scenario: _Optional[str] = ..., active_only: _Optional[bool] = ..., limit: _Optional[int] = ...) -> None: ...

class ListWorkflowsResponse(_message.Message):
    __slots__ = ("workflows",)
    WORKFLOWS_FIELD_NUMBER: _ClassVar[int]
    workflows: _containers.RepeatedCompositeFieldContainer[Workflow]
    def __init__(self, workflows: _Optional[_Iterable[_Union[Workflow, _Mapping]]] = ...) -> None: ...

class GetWorkflowRequest(_message.Message):
    __slots__ = ("id",)
    ID_FIELD_NUMBER: _ClassVar[int]
    id: str
    def __init__(self, id: _Optional[str] = ...) -> None: ...

class GetWorkflowResponse(_message.Message):
    __slots__ = ("workflow",)
    WORKFLOW_FIELD_NUMBER: _ClassVar[int]
    workflow: Workflow
    def __init__(self, workflow: _Optional[_Union[Workflow, _Mapping]] = ...) -> None: ...

class RefreshWorkflowRequest(_message.Message):
    __slots__ = ("id",)
    ID_FIELD_NUMBER: _ClassVar[int]
    id: str
    def __init__(self, id: _Optional[str] = ...) -> None: ...

class RefreshWorkflowResponse(_message.Message):
    __slots__ = ("workflow",)
    WORKFLOW_FIELD_NUMBER: _ClassVar[int]
    workflow: Workflow
    def __init__(self, workflow: _Optional[_Union[Workflow, _Mapping]] = ...) -> None: ...

class StopWorkflowRequest(_message.Message):
    __slots__ = ("id",)
    ID_FIELD_NUMBER: _ClassVar[int]
    id: str
    def __init__(self, id: _Optional[str] = ...) -> None: ...

class StopWorkflowResponse(_message.Message):
    __slots__ = ("workflow",)
    WORKFLOW_FIELD_NUMBER: _ClassVar[int]
    workflow: Workflow
    def __init__(self, workflow: _Optional[_Union[Workflow, _Mapping]] = ...) -> None: ...

class RetryWorkflowRequest(_message.Message):
    __slots__ = ("id", "idempotency_key")
    ID_FIELD_NUMBER: _ClassVar[int]
    IDEMPOTENCY_KEY_FIELD_NUMBER: _ClassVar[int]
    id: str
    idempotency_key: str
    def __init__(self, id: _Optional[str] = ..., idempotency_key: _Optional[str] = ...) -> None: ...

class RetryWorkflowResponse(_message.Message):
    __slots__ = ("workflow", "queue_depth")
    WORKFLOW_FIELD_NUMBER: _ClassVar[int]
    QUEUE_DEPTH_FIELD_NUMBER: _ClassVar[int]
    workflow: Workflow
    queue_depth: int
    def __init__(self, workflow: _Optional[_Union[Workflow, _Mapping]] = ..., queue_depth: _Optional[int] = ...) -> None: ...

class GetPromotionReadinessRequest(_message.Message):
    __slots__ = ("asset_id", "origin_scenario", "version")
    ASSET_ID_FIELD_NUMBER: _ClassVar[int]
    ORIGIN_SCENARIO_FIELD_NUMBER: _ClassVar[int]
    VERSION_FIELD_NUMBER: _ClassVar[int]
    asset_id: str
    origin_scenario: str
    version: str
    def __init__(self, asset_id: _Optional[str] = ..., origin_scenario: _Optional[str] = ..., version: _Optional[str] = ...) -> None: ...

class PromotionReadiness(_message.Message):
    __slots__ = ("asset_id", "library_id", "selected_version", "origin_scenario", "dependency_library_ids", "origin_files", "required_example_count", "available_example_count", "parity_report_present", "parity_waived", "parity_findings", "origin_replacement_present", "origin_replacement_clean", "blockers", "ready", "next_validation_command")
    ASSET_ID_FIELD_NUMBER: _ClassVar[int]
    LIBRARY_ID_FIELD_NUMBER: _ClassVar[int]
    SELECTED_VERSION_FIELD_NUMBER: _ClassVar[int]
    ORIGIN_SCENARIO_FIELD_NUMBER: _ClassVar[int]
    DEPENDENCY_LIBRARY_IDS_FIELD_NUMBER: _ClassVar[int]
    ORIGIN_FILES_FIELD_NUMBER: _ClassVar[int]
    REQUIRED_EXAMPLE_COUNT_FIELD_NUMBER: _ClassVar[int]
    AVAILABLE_EXAMPLE_COUNT_FIELD_NUMBER: _ClassVar[int]
    PARITY_REPORT_PRESENT_FIELD_NUMBER: _ClassVar[int]
    PARITY_WAIVED_FIELD_NUMBER: _ClassVar[int]
    PARITY_FINDINGS_FIELD_NUMBER: _ClassVar[int]
    ORIGIN_REPLACEMENT_PRESENT_FIELD_NUMBER: _ClassVar[int]
    ORIGIN_REPLACEMENT_CLEAN_FIELD_NUMBER: _ClassVar[int]
    BLOCKERS_FIELD_NUMBER: _ClassVar[int]
    READY_FIELD_NUMBER: _ClassVar[int]
    NEXT_VALIDATION_COMMAND_FIELD_NUMBER: _ClassVar[int]
    asset_id: str
    library_id: str
    selected_version: str
    origin_scenario: str
    dependency_library_ids: _containers.RepeatedScalarFieldContainer[str]
    origin_files: _containers.RepeatedScalarFieldContainer[str]
    required_example_count: int
    available_example_count: int
    parity_report_present: bool
    parity_waived: bool
    parity_findings: _containers.RepeatedScalarFieldContainer[str]
    origin_replacement_present: bool
    origin_replacement_clean: bool
    blockers: _containers.RepeatedScalarFieldContainer[str]
    ready: bool
    next_validation_command: str
    def __init__(self, asset_id: _Optional[str] = ..., library_id: _Optional[str] = ..., selected_version: _Optional[str] = ..., origin_scenario: _Optional[str] = ..., dependency_library_ids: _Optional[_Iterable[str]] = ..., origin_files: _Optional[_Iterable[str]] = ..., required_example_count: _Optional[int] = ..., available_example_count: _Optional[int] = ..., parity_report_present: _Optional[bool] = ..., parity_waived: _Optional[bool] = ..., parity_findings: _Optional[_Iterable[str]] = ..., origin_replacement_present: _Optional[bool] = ..., origin_replacement_clean: _Optional[bool] = ..., blockers: _Optional[_Iterable[str]] = ..., ready: _Optional[bool] = ..., next_validation_command: _Optional[str] = ...) -> None: ...

class GetPromotionReadinessResponse(_message.Message):
    __slots__ = ("readiness",)
    READINESS_FIELD_NUMBER: _ClassVar[int]
    readiness: PromotionReadiness
    def __init__(self, readiness: _Optional[_Union[PromotionReadiness, _Mapping]] = ...) -> None: ...
