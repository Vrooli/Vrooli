import datetime

from browser_automation_studio.v1.actions import action_pb2 as _action_pb2
from browser_automation_studio.v1.base import shared_pb2 as _shared_pb2
from browser_automation_studio.v1.execution import execution_pb2 as _execution_pb2
from browser_automation_studio.v1.timeline import container_pb2 as _container_pb2
from browser_automation_studio.v1.workflows import definition_pb2 as _definition_pb2
from buf.validate import validate_pb2 as _validate_pb2
from common.v1 import types_pb2 as _types_pb2
from google.protobuf import timestamp_pb2 as _timestamp_pb2
from google.protobuf.internal import containers as _containers
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Iterable as _Iterable, Mapping as _Mapping
from typing import ClassVar as _ClassVar, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class WorkflowSummary(_message.Message):
    __slots__ = ("id", "project_id", "name", "folder_path", "description", "tags", "version", "is_template", "created_by", "last_change_source", "last_change_description", "created_at", "updated_at", "flow_definition")
    ID_FIELD_NUMBER: _ClassVar[int]
    PROJECT_ID_FIELD_NUMBER: _ClassVar[int]
    NAME_FIELD_NUMBER: _ClassVar[int]
    FOLDER_PATH_FIELD_NUMBER: _ClassVar[int]
    DESCRIPTION_FIELD_NUMBER: _ClassVar[int]
    TAGS_FIELD_NUMBER: _ClassVar[int]
    VERSION_FIELD_NUMBER: _ClassVar[int]
    IS_TEMPLATE_FIELD_NUMBER: _ClassVar[int]
    CREATED_BY_FIELD_NUMBER: _ClassVar[int]
    LAST_CHANGE_SOURCE_FIELD_NUMBER: _ClassVar[int]
    LAST_CHANGE_DESCRIPTION_FIELD_NUMBER: _ClassVar[int]
    CREATED_AT_FIELD_NUMBER: _ClassVar[int]
    UPDATED_AT_FIELD_NUMBER: _ClassVar[int]
    FLOW_DEFINITION_FIELD_NUMBER: _ClassVar[int]
    id: str
    project_id: str
    name: str
    folder_path: str
    description: str
    tags: _containers.RepeatedScalarFieldContainer[str]
    version: int
    is_template: bool
    created_by: str
    last_change_source: _shared_pb2.ChangeSource
    last_change_description: str
    created_at: _timestamp_pb2.Timestamp
    updated_at: _timestamp_pb2.Timestamp
    flow_definition: _definition_pb2.WorkflowDefinitionV2
    def __init__(self, id: _Optional[str] = ..., project_id: _Optional[str] = ..., name: _Optional[str] = ..., folder_path: _Optional[str] = ..., description: _Optional[str] = ..., tags: _Optional[_Iterable[str]] = ..., version: _Optional[int] = ..., is_template: _Optional[bool] = ..., created_by: _Optional[str] = ..., last_change_source: _Optional[_Union[_shared_pb2.ChangeSource, str]] = ..., last_change_description: _Optional[str] = ..., created_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., updated_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., flow_definition: _Optional[_Union[_definition_pb2.WorkflowDefinitionV2, _Mapping]] = ...) -> None: ...

class WorkflowVersion(_message.Message):
    __slots__ = ("workflow_id", "version", "flow_definition", "change_description", "created_by", "created_at")
    WORKFLOW_ID_FIELD_NUMBER: _ClassVar[int]
    VERSION_FIELD_NUMBER: _ClassVar[int]
    FLOW_DEFINITION_FIELD_NUMBER: _ClassVar[int]
    CHANGE_DESCRIPTION_FIELD_NUMBER: _ClassVar[int]
    CREATED_BY_FIELD_NUMBER: _ClassVar[int]
    CREATED_AT_FIELD_NUMBER: _ClassVar[int]
    workflow_id: str
    version: int
    flow_definition: _definition_pb2.WorkflowDefinitionV2
    change_description: str
    created_by: str
    created_at: _timestamp_pb2.Timestamp
    def __init__(self, workflow_id: _Optional[str] = ..., version: _Optional[int] = ..., flow_definition: _Optional[_Union[_definition_pb2.WorkflowDefinitionV2, _Mapping]] = ..., change_description: _Optional[str] = ..., created_by: _Optional[str] = ..., created_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ...) -> None: ...

class WorkflowList(_message.Message):
    __slots__ = ("workflows",)
    WORKFLOWS_FIELD_NUMBER: _ClassVar[int]
    workflows: _containers.RepeatedCompositeFieldContainer[WorkflowSummary]
    def __init__(self, workflows: _Optional[_Iterable[_Union[WorkflowSummary, _Mapping]]] = ...) -> None: ...

class WorkflowVersionList(_message.Message):
    __slots__ = ("versions",)
    VERSIONS_FIELD_NUMBER: _ClassVar[int]
    versions: _containers.RepeatedCompositeFieldContainer[WorkflowVersion]
    def __init__(self, versions: _Optional[_Iterable[_Union[WorkflowVersion, _Mapping]]] = ...) -> None: ...

class ListWorkflowsRequest(_message.Message):
    __slots__ = ("project_id", "folder_path", "limit", "offset")
    PROJECT_ID_FIELD_NUMBER: _ClassVar[int]
    FOLDER_PATH_FIELD_NUMBER: _ClassVar[int]
    LIMIT_FIELD_NUMBER: _ClassVar[int]
    OFFSET_FIELD_NUMBER: _ClassVar[int]
    project_id: str
    folder_path: str
    limit: int
    offset: int
    def __init__(self, project_id: _Optional[str] = ..., folder_path: _Optional[str] = ..., limit: _Optional[int] = ..., offset: _Optional[int] = ...) -> None: ...

class ListWorkflowsResponse(_message.Message):
    __slots__ = ("workflows", "total", "has_more")
    WORKFLOWS_FIELD_NUMBER: _ClassVar[int]
    TOTAL_FIELD_NUMBER: _ClassVar[int]
    HAS_MORE_FIELD_NUMBER: _ClassVar[int]
    workflows: _containers.RepeatedCompositeFieldContainer[WorkflowSummary]
    total: int
    has_more: bool
    def __init__(self, workflows: _Optional[_Iterable[_Union[WorkflowSummary, _Mapping]]] = ..., total: _Optional[int] = ..., has_more: _Optional[bool] = ...) -> None: ...

class GetWorkflowRequest(_message.Message):
    __slots__ = ("workflow_id", "version")
    WORKFLOW_ID_FIELD_NUMBER: _ClassVar[int]
    VERSION_FIELD_NUMBER: _ClassVar[int]
    workflow_id: str
    version: int
    def __init__(self, workflow_id: _Optional[str] = ..., version: _Optional[int] = ...) -> None: ...

class GetWorkflowResponse(_message.Message):
    __slots__ = ("workflow",)
    WORKFLOW_FIELD_NUMBER: _ClassVar[int]
    workflow: WorkflowSummary
    def __init__(self, workflow: _Optional[_Union[WorkflowSummary, _Mapping]] = ...) -> None: ...

class CreateWorkflowRequest(_message.Message):
    __slots__ = ("project_id", "name", "folder_path", "flow_definition", "ai_prompt")
    PROJECT_ID_FIELD_NUMBER: _ClassVar[int]
    NAME_FIELD_NUMBER: _ClassVar[int]
    FOLDER_PATH_FIELD_NUMBER: _ClassVar[int]
    FLOW_DEFINITION_FIELD_NUMBER: _ClassVar[int]
    AI_PROMPT_FIELD_NUMBER: _ClassVar[int]
    project_id: str
    name: str
    folder_path: str
    flow_definition: _definition_pb2.WorkflowDefinitionV2
    ai_prompt: str
    def __init__(self, project_id: _Optional[str] = ..., name: _Optional[str] = ..., folder_path: _Optional[str] = ..., flow_definition: _Optional[_Union[_definition_pb2.WorkflowDefinitionV2, _Mapping]] = ..., ai_prompt: _Optional[str] = ...) -> None: ...

class CreateWorkflowResponse(_message.Message):
    __slots__ = ("workflow", "flow_definition")
    WORKFLOW_FIELD_NUMBER: _ClassVar[int]
    FLOW_DEFINITION_FIELD_NUMBER: _ClassVar[int]
    workflow: WorkflowSummary
    flow_definition: _definition_pb2.WorkflowDefinitionV2
    def __init__(self, workflow: _Optional[_Union[WorkflowSummary, _Mapping]] = ..., flow_definition: _Optional[_Union[_definition_pb2.WorkflowDefinitionV2, _Mapping]] = ...) -> None: ...

class UpdateWorkflowRequest(_message.Message):
    __slots__ = ("name", "description", "folder_path", "tags", "flow_definition", "change_description", "source", "expected_version", "workflow_id")
    NAME_FIELD_NUMBER: _ClassVar[int]
    DESCRIPTION_FIELD_NUMBER: _ClassVar[int]
    FOLDER_PATH_FIELD_NUMBER: _ClassVar[int]
    TAGS_FIELD_NUMBER: _ClassVar[int]
    FLOW_DEFINITION_FIELD_NUMBER: _ClassVar[int]
    CHANGE_DESCRIPTION_FIELD_NUMBER: _ClassVar[int]
    SOURCE_FIELD_NUMBER: _ClassVar[int]
    EXPECTED_VERSION_FIELD_NUMBER: _ClassVar[int]
    WORKFLOW_ID_FIELD_NUMBER: _ClassVar[int]
    name: str
    description: str
    folder_path: str
    tags: _containers.RepeatedScalarFieldContainer[str]
    flow_definition: _definition_pb2.WorkflowDefinitionV2
    change_description: str
    source: _shared_pb2.ChangeSource
    expected_version: int
    workflow_id: str
    def __init__(self, name: _Optional[str] = ..., description: _Optional[str] = ..., folder_path: _Optional[str] = ..., tags: _Optional[_Iterable[str]] = ..., flow_definition: _Optional[_Union[_definition_pb2.WorkflowDefinitionV2, _Mapping]] = ..., change_description: _Optional[str] = ..., source: _Optional[_Union[_shared_pb2.ChangeSource, str]] = ..., expected_version: _Optional[int] = ..., workflow_id: _Optional[str] = ...) -> None: ...

class UpdateWorkflowResponse(_message.Message):
    __slots__ = ("workflow", "flow_definition")
    WORKFLOW_FIELD_NUMBER: _ClassVar[int]
    FLOW_DEFINITION_FIELD_NUMBER: _ClassVar[int]
    workflow: WorkflowSummary
    flow_definition: _definition_pb2.WorkflowDefinitionV2
    def __init__(self, workflow: _Optional[_Union[WorkflowSummary, _Mapping]] = ..., flow_definition: _Optional[_Union[_definition_pb2.WorkflowDefinitionV2, _Mapping]] = ...) -> None: ...

class DeleteWorkflowRequest(_message.Message):
    __slots__ = ("workflow_id",)
    WORKFLOW_ID_FIELD_NUMBER: _ClassVar[int]
    workflow_id: str
    def __init__(self, workflow_id: _Optional[str] = ...) -> None: ...

class DeleteWorkflowResponse(_message.Message):
    __slots__ = ("success", "workflow_id")
    SUCCESS_FIELD_NUMBER: _ClassVar[int]
    WORKFLOW_ID_FIELD_NUMBER: _ClassVar[int]
    success: bool
    workflow_id: str
    def __init__(self, success: _Optional[bool] = ..., workflow_id: _Optional[str] = ...) -> None: ...

class ExecuteWorkflowRequest(_message.Message):
    __slots__ = ("wait_for_completion", "workflow_id", "workflow_version", "parameters", "options")
    WAIT_FOR_COMPLETION_FIELD_NUMBER: _ClassVar[int]
    WORKFLOW_ID_FIELD_NUMBER: _ClassVar[int]
    WORKFLOW_VERSION_FIELD_NUMBER: _ClassVar[int]
    PARAMETERS_FIELD_NUMBER: _ClassVar[int]
    OPTIONS_FIELD_NUMBER: _ClassVar[int]
    wait_for_completion: bool
    workflow_id: str
    workflow_version: int
    parameters: _execution_pb2.ExecutionParameters
    options: _execution_pb2.ExecuteWorkflowOptions
    def __init__(self, wait_for_completion: _Optional[bool] = ..., workflow_id: _Optional[str] = ..., workflow_version: _Optional[int] = ..., parameters: _Optional[_Union[_execution_pb2.ExecutionParameters, _Mapping]] = ..., options: _Optional[_Union[_execution_pb2.ExecuteWorkflowOptions, _Mapping]] = ...) -> None: ...

class ExecuteWorkflowResponse(_message.Message):
    __slots__ = ("execution_id", "status", "completed_at", "error")
    EXECUTION_ID_FIELD_NUMBER: _ClassVar[int]
    STATUS_FIELD_NUMBER: _ClassVar[int]
    COMPLETED_AT_FIELD_NUMBER: _ClassVar[int]
    ERROR_FIELD_NUMBER: _ClassVar[int]
    execution_id: str
    status: _shared_pb2.ExecutionStatus
    completed_at: _timestamp_pb2.Timestamp
    error: str
    def __init__(self, execution_id: _Optional[str] = ..., status: _Optional[_Union[_shared_pb2.ExecutionStatus, str]] = ..., completed_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., error: _Optional[str] = ...) -> None: ...

class ListExecutionsRequest(_message.Message):
    __slots__ = ("workflow_id", "status", "limit", "offset", "project_id", "include_exportability")
    WORKFLOW_ID_FIELD_NUMBER: _ClassVar[int]
    STATUS_FIELD_NUMBER: _ClassVar[int]
    LIMIT_FIELD_NUMBER: _ClassVar[int]
    OFFSET_FIELD_NUMBER: _ClassVar[int]
    PROJECT_ID_FIELD_NUMBER: _ClassVar[int]
    INCLUDE_EXPORTABILITY_FIELD_NUMBER: _ClassVar[int]
    workflow_id: str
    status: _shared_pb2.ExecutionStatus
    limit: int
    offset: int
    project_id: str
    include_exportability: bool
    def __init__(self, workflow_id: _Optional[str] = ..., status: _Optional[_Union[_shared_pb2.ExecutionStatus, str]] = ..., limit: _Optional[int] = ..., offset: _Optional[int] = ..., project_id: _Optional[str] = ..., include_exportability: _Optional[bool] = ...) -> None: ...

class ListExecutionsResponse(_message.Message):
    __slots__ = ("executions", "total", "has_more", "exportability")
    class ExportabilityEntry(_message.Message):
        __slots__ = ("key", "value")
        KEY_FIELD_NUMBER: _ClassVar[int]
        VALUE_FIELD_NUMBER: _ClassVar[int]
        key: str
        value: ExecutionExportability
        def __init__(self, key: _Optional[str] = ..., value: _Optional[_Union[ExecutionExportability, _Mapping]] = ...) -> None: ...
    EXECUTIONS_FIELD_NUMBER: _ClassVar[int]
    TOTAL_FIELD_NUMBER: _ClassVar[int]
    HAS_MORE_FIELD_NUMBER: _ClassVar[int]
    EXPORTABILITY_FIELD_NUMBER: _ClassVar[int]
    executions: _containers.RepeatedCompositeFieldContainer[_execution_pb2.Execution]
    total: int
    has_more: bool
    exportability: _containers.MessageMap[str, ExecutionExportability]
    def __init__(self, executions: _Optional[_Iterable[_Union[_execution_pb2.Execution, _Mapping]]] = ..., total: _Optional[int] = ..., has_more: _Optional[bool] = ..., exportability: _Optional[_Mapping[str, ExecutionExportability]] = ...) -> None: ...

class ExecutionExportability(_message.Message):
    __slots__ = ("has_timeline", "has_screenshots", "has_recorded_video", "is_exportable")
    HAS_TIMELINE_FIELD_NUMBER: _ClassVar[int]
    HAS_SCREENSHOTS_FIELD_NUMBER: _ClassVar[int]
    HAS_RECORDED_VIDEO_FIELD_NUMBER: _ClassVar[int]
    IS_EXPORTABLE_FIELD_NUMBER: _ClassVar[int]
    has_timeline: bool
    has_screenshots: bool
    has_recorded_video: bool
    is_exportable: bool
    def __init__(self, has_timeline: _Optional[bool] = ..., has_screenshots: _Optional[bool] = ..., has_recorded_video: _Optional[bool] = ..., is_exportable: _Optional[bool] = ...) -> None: ...

class GetExecutionRequest(_message.Message):
    __slots__ = ("execution_id",)
    EXECUTION_ID_FIELD_NUMBER: _ClassVar[int]
    execution_id: str
    def __init__(self, execution_id: _Optional[str] = ...) -> None: ...

class GetExecutionResponse(_message.Message):
    __slots__ = ("execution",)
    EXECUTION_FIELD_NUMBER: _ClassVar[int]
    execution: _execution_pb2.Execution
    def __init__(self, execution: _Optional[_Union[_execution_pb2.Execution, _Mapping]] = ...) -> None: ...

class ValidateWorkflowRequest(_message.Message):
    __slots__ = ("workflow",)
    WORKFLOW_FIELD_NUMBER: _ClassVar[int]
    workflow: _definition_pb2.WorkflowDefinitionV2
    def __init__(self, workflow: _Optional[_Union[_definition_pb2.WorkflowDefinitionV2, _Mapping]] = ...) -> None: ...

class ValidateWorkflowResponse(_message.Message):
    __slots__ = ("result",)
    RESULT_FIELD_NUMBER: _ClassVar[int]
    result: WorkflowValidationResult
    def __init__(self, result: _Optional[_Union[WorkflowValidationResult, _Mapping]] = ...) -> None: ...

class WorkflowValidationIssue(_message.Message):
    __slots__ = ("severity", "code", "message", "node_id", "node_type", "field", "pointer", "hint")
    SEVERITY_FIELD_NUMBER: _ClassVar[int]
    CODE_FIELD_NUMBER: _ClassVar[int]
    MESSAGE_FIELD_NUMBER: _ClassVar[int]
    NODE_ID_FIELD_NUMBER: _ClassVar[int]
    NODE_TYPE_FIELD_NUMBER: _ClassVar[int]
    FIELD_FIELD_NUMBER: _ClassVar[int]
    POINTER_FIELD_NUMBER: _ClassVar[int]
    HINT_FIELD_NUMBER: _ClassVar[int]
    severity: _shared_pb2.ValidationSeverity
    code: str
    message: str
    node_id: str
    node_type: _action_pb2.ActionType
    field: str
    pointer: str
    hint: str
    def __init__(self, severity: _Optional[_Union[_shared_pb2.ValidationSeverity, str]] = ..., code: _Optional[str] = ..., message: _Optional[str] = ..., node_id: _Optional[str] = ..., node_type: _Optional[_Union[_action_pb2.ActionType, str]] = ..., field: _Optional[str] = ..., pointer: _Optional[str] = ..., hint: _Optional[str] = ...) -> None: ...

class WorkflowValidationStats(_message.Message):
    __slots__ = ("node_count", "edge_count", "selector_count", "unique_selector_count", "element_wait_count", "has_metadata", "has_execution_viewport")
    NODE_COUNT_FIELD_NUMBER: _ClassVar[int]
    EDGE_COUNT_FIELD_NUMBER: _ClassVar[int]
    SELECTOR_COUNT_FIELD_NUMBER: _ClassVar[int]
    UNIQUE_SELECTOR_COUNT_FIELD_NUMBER: _ClassVar[int]
    ELEMENT_WAIT_COUNT_FIELD_NUMBER: _ClassVar[int]
    HAS_METADATA_FIELD_NUMBER: _ClassVar[int]
    HAS_EXECUTION_VIEWPORT_FIELD_NUMBER: _ClassVar[int]
    node_count: int
    edge_count: int
    selector_count: int
    unique_selector_count: int
    element_wait_count: int
    has_metadata: bool
    has_execution_viewport: bool
    def __init__(self, node_count: _Optional[int] = ..., edge_count: _Optional[int] = ..., selector_count: _Optional[int] = ..., unique_selector_count: _Optional[int] = ..., element_wait_count: _Optional[int] = ..., has_metadata: _Optional[bool] = ..., has_execution_viewport: _Optional[bool] = ...) -> None: ...

class WorkflowValidationResult(_message.Message):
    __slots__ = ("valid", "errors", "warnings", "stats", "schema_version", "checked_at", "duration_ms")
    VALID_FIELD_NUMBER: _ClassVar[int]
    ERRORS_FIELD_NUMBER: _ClassVar[int]
    WARNINGS_FIELD_NUMBER: _ClassVar[int]
    STATS_FIELD_NUMBER: _ClassVar[int]
    SCHEMA_VERSION_FIELD_NUMBER: _ClassVar[int]
    CHECKED_AT_FIELD_NUMBER: _ClassVar[int]
    DURATION_MS_FIELD_NUMBER: _ClassVar[int]
    valid: bool
    errors: _containers.RepeatedCompositeFieldContainer[WorkflowValidationIssue]
    warnings: _containers.RepeatedCompositeFieldContainer[WorkflowValidationIssue]
    stats: WorkflowValidationStats
    schema_version: str
    checked_at: _timestamp_pb2.Timestamp
    duration_ms: int
    def __init__(self, valid: _Optional[bool] = ..., errors: _Optional[_Iterable[_Union[WorkflowValidationIssue, _Mapping]]] = ..., warnings: _Optional[_Iterable[_Union[WorkflowValidationIssue, _Mapping]]] = ..., stats: _Optional[_Union[WorkflowValidationStats, _Mapping]] = ..., schema_version: _Optional[str] = ..., checked_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., duration_ms: _Optional[int] = ...) -> None: ...

class RestoreWorkflowVersionResponse(_message.Message):
    __slots__ = ("workflow", "restored_version")
    WORKFLOW_FIELD_NUMBER: _ClassVar[int]
    RESTORED_VERSION_FIELD_NUMBER: _ClassVar[int]
    workflow: WorkflowSummary
    restored_version: WorkflowVersion
    def __init__(self, workflow: _Optional[_Union[WorkflowSummary, _Mapping]] = ..., restored_version: _Optional[_Union[WorkflowVersion, _Mapping]] = ...) -> None: ...

class ListWorkflowVersionsRequest(_message.Message):
    __slots__ = ("workflow_id",)
    WORKFLOW_ID_FIELD_NUMBER: _ClassVar[int]
    workflow_id: str
    def __init__(self, workflow_id: _Optional[str] = ...) -> None: ...

class GetWorkflowVersionRequest(_message.Message):
    __slots__ = ("workflow_id", "version")
    WORKFLOW_ID_FIELD_NUMBER: _ClassVar[int]
    VERSION_FIELD_NUMBER: _ClassVar[int]
    workflow_id: str
    version: int
    def __init__(self, workflow_id: _Optional[str] = ..., version: _Optional[int] = ...) -> None: ...

class RestoreWorkflowVersionRequest(_message.Message):
    __slots__ = ("workflow_id", "version", "change_description")
    WORKFLOW_ID_FIELD_NUMBER: _ClassVar[int]
    VERSION_FIELD_NUMBER: _ClassVar[int]
    CHANGE_DESCRIPTION_FIELD_NUMBER: _ClassVar[int]
    workflow_id: str
    version: int
    change_description: str
    def __init__(self, workflow_id: _Optional[str] = ..., version: _Optional[int] = ..., change_description: _Optional[str] = ...) -> None: ...

class ModifyWorkflowRequest(_message.Message):
    __slots__ = ("workflow_id", "modification_prompt", "current_flow")
    WORKFLOW_ID_FIELD_NUMBER: _ClassVar[int]
    MODIFICATION_PROMPT_FIELD_NUMBER: _ClassVar[int]
    CURRENT_FLOW_FIELD_NUMBER: _ClassVar[int]
    workflow_id: str
    modification_prompt: str
    current_flow: _definition_pb2.WorkflowDefinitionV2
    def __init__(self, workflow_id: _Optional[str] = ..., modification_prompt: _Optional[str] = ..., current_flow: _Optional[_Union[_definition_pb2.WorkflowDefinitionV2, _Mapping]] = ...) -> None: ...

class GetExecutionTimelineRequest(_message.Message):
    __slots__ = ("execution_id",)
    EXECUTION_ID_FIELD_NUMBER: _ClassVar[int]
    execution_id: str
    def __init__(self, execution_id: _Optional[str] = ...) -> None: ...

class StopExecutionRequest(_message.Message):
    __slots__ = ("execution_id",)
    EXECUTION_ID_FIELD_NUMBER: _ClassVar[int]
    execution_id: str
    def __init__(self, execution_id: _Optional[str] = ...) -> None: ...

class StopExecutionResponse(_message.Message):
    __slots__ = ("status",)
    STATUS_FIELD_NUMBER: _ClassVar[int]
    status: str
    def __init__(self, status: _Optional[str] = ...) -> None: ...

class ResumeExecutionRequest(_message.Message):
    __slots__ = ("execution_id", "parameters", "resume_url")
    EXECUTION_ID_FIELD_NUMBER: _ClassVar[int]
    PARAMETERS_FIELD_NUMBER: _ClassVar[int]
    RESUME_URL_FIELD_NUMBER: _ClassVar[int]
    execution_id: str
    parameters: _types_pb2.JsonObject
    resume_url: str
    def __init__(self, execution_id: _Optional[str] = ..., parameters: _Optional[_Union[_types_pb2.JsonObject, _Mapping]] = ..., resume_url: _Optional[str] = ...) -> None: ...

class ResumeExecutionResponse(_message.Message):
    __slots__ = ("execution",)
    EXECUTION_FIELD_NUMBER: _ClassVar[int]
    execution: _execution_pb2.Execution
    def __init__(self, execution: _Optional[_Union[_execution_pb2.Execution, _Mapping]] = ...) -> None: ...

class GetExecutionScreenshotsRequest(_message.Message):
    __slots__ = ("execution_id",)
    EXECUTION_ID_FIELD_NUMBER: _ClassVar[int]
    execution_id: str
    def __init__(self, execution_id: _Optional[str] = ...) -> None: ...

class GetExecutionArtifactsRequest(_message.Message):
    __slots__ = ("execution_id",)
    EXECUTION_ID_FIELD_NUMBER: _ClassVar[int]
    execution_id: str
    def __init__(self, execution_id: _Optional[str] = ...) -> None: ...

class ExecutionFileArtifact(_message.Message):
    __slots__ = ("artifact_id", "storage_url", "content_type", "label", "size_bytes", "payload")
    ARTIFACT_ID_FIELD_NUMBER: _ClassVar[int]
    STORAGE_URL_FIELD_NUMBER: _ClassVar[int]
    CONTENT_TYPE_FIELD_NUMBER: _ClassVar[int]
    LABEL_FIELD_NUMBER: _ClassVar[int]
    SIZE_BYTES_FIELD_NUMBER: _ClassVar[int]
    PAYLOAD_FIELD_NUMBER: _ClassVar[int]
    artifact_id: str
    storage_url: str
    content_type: str
    label: str
    size_bytes: int
    payload: _types_pb2.JsonObject
    def __init__(self, artifact_id: _Optional[str] = ..., storage_url: _Optional[str] = ..., content_type: _Optional[str] = ..., label: _Optional[str] = ..., size_bytes: _Optional[int] = ..., payload: _Optional[_Union[_types_pb2.JsonObject, _Mapping]] = ...) -> None: ...

class GetExecutionVideosResponse(_message.Message):
    __slots__ = ("execution_id", "videos")
    EXECUTION_ID_FIELD_NUMBER: _ClassVar[int]
    VIDEOS_FIELD_NUMBER: _ClassVar[int]
    execution_id: str
    videos: _containers.RepeatedCompositeFieldContainer[ExecutionFileArtifact]
    def __init__(self, execution_id: _Optional[str] = ..., videos: _Optional[_Iterable[_Union[ExecutionFileArtifact, _Mapping]]] = ...) -> None: ...

class GetExecutionTracesResponse(_message.Message):
    __slots__ = ("execution_id", "traces")
    EXECUTION_ID_FIELD_NUMBER: _ClassVar[int]
    TRACES_FIELD_NUMBER: _ClassVar[int]
    execution_id: str
    traces: _containers.RepeatedCompositeFieldContainer[ExecutionFileArtifact]
    def __init__(self, execution_id: _Optional[str] = ..., traces: _Optional[_Iterable[_Union[ExecutionFileArtifact, _Mapping]]] = ...) -> None: ...

class GetExecutionHarResponse(_message.Message):
    __slots__ = ("execution_id", "har_files")
    EXECUTION_ID_FIELD_NUMBER: _ClassVar[int]
    HAR_FILES_FIELD_NUMBER: _ClassVar[int]
    execution_id: str
    har_files: _containers.RepeatedCompositeFieldContainer[ExecutionFileArtifact]
    def __init__(self, execution_id: _Optional[str] = ..., har_files: _Optional[_Iterable[_Union[ExecutionFileArtifact, _Mapping]]] = ...) -> None: ...

class ScheduleSeedCleanupRequest(_message.Message):
    __slots__ = ("execution_id", "cleanup_token", "seed_scenario")
    EXECUTION_ID_FIELD_NUMBER: _ClassVar[int]
    CLEANUP_TOKEN_FIELD_NUMBER: _ClassVar[int]
    SEED_SCENARIO_FIELD_NUMBER: _ClassVar[int]
    execution_id: str
    cleanup_token: str
    seed_scenario: str
    def __init__(self, execution_id: _Optional[str] = ..., cleanup_token: _Optional[str] = ..., seed_scenario: _Optional[str] = ...) -> None: ...

class ScheduleSeedCleanupResponse(_message.Message):
    __slots__ = ("status",)
    STATUS_FIELD_NUMBER: _ClassVar[int]
    status: str
    def __init__(self, status: _Optional[str] = ...) -> None: ...

class ExecutionArtifactRetentionRequest(_message.Message):
    __slots__ = ("max_age_days", "keep_latest", "workflow_id", "project_id", "status", "confirm")
    MAX_AGE_DAYS_FIELD_NUMBER: _ClassVar[int]
    KEEP_LATEST_FIELD_NUMBER: _ClassVar[int]
    WORKFLOW_ID_FIELD_NUMBER: _ClassVar[int]
    PROJECT_ID_FIELD_NUMBER: _ClassVar[int]
    STATUS_FIELD_NUMBER: _ClassVar[int]
    CONFIRM_FIELD_NUMBER: _ClassVar[int]
    max_age_days: int
    keep_latest: int
    workflow_id: str
    project_id: str
    status: _shared_pb2.ExecutionStatus
    confirm: bool
    def __init__(self, max_age_days: _Optional[int] = ..., keep_latest: _Optional[int] = ..., workflow_id: _Optional[str] = ..., project_id: _Optional[str] = ..., status: _Optional[_Union[_shared_pb2.ExecutionStatus, str]] = ..., confirm: _Optional[bool] = ...) -> None: ...

class ExecutionRetentionItem(_message.Message):
    __slots__ = ("execution_id", "status", "workflow_id", "started_at", "completed_at", "result_path", "artifact_dir", "estimated_bytes", "reason")
    EXECUTION_ID_FIELD_NUMBER: _ClassVar[int]
    STATUS_FIELD_NUMBER: _ClassVar[int]
    WORKFLOW_ID_FIELD_NUMBER: _ClassVar[int]
    STARTED_AT_FIELD_NUMBER: _ClassVar[int]
    COMPLETED_AT_FIELD_NUMBER: _ClassVar[int]
    RESULT_PATH_FIELD_NUMBER: _ClassVar[int]
    ARTIFACT_DIR_FIELD_NUMBER: _ClassVar[int]
    ESTIMATED_BYTES_FIELD_NUMBER: _ClassVar[int]
    REASON_FIELD_NUMBER: _ClassVar[int]
    execution_id: str
    status: _shared_pb2.ExecutionStatus
    workflow_id: str
    started_at: _timestamp_pb2.Timestamp
    completed_at: _timestamp_pb2.Timestamp
    result_path: str
    artifact_dir: str
    estimated_bytes: int
    reason: str
    def __init__(self, execution_id: _Optional[str] = ..., status: _Optional[_Union[_shared_pb2.ExecutionStatus, str]] = ..., workflow_id: _Optional[str] = ..., started_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., completed_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., result_path: _Optional[str] = ..., artifact_dir: _Optional[str] = ..., estimated_bytes: _Optional[int] = ..., reason: _Optional[str] = ...) -> None: ...

class ExecutionArtifactRetentionResponse(_message.Message):
    __slots__ = ("dry_run", "removed", "skipped", "estimated_bytes", "removed_count", "skipped_count", "error_count", "removed_by_status")
    class RemovedByStatusEntry(_message.Message):
        __slots__ = ("key", "value")
        KEY_FIELD_NUMBER: _ClassVar[int]
        VALUE_FIELD_NUMBER: _ClassVar[int]
        key: str
        value: int
        def __init__(self, key: _Optional[str] = ..., value: _Optional[int] = ...) -> None: ...
    DRY_RUN_FIELD_NUMBER: _ClassVar[int]
    REMOVED_FIELD_NUMBER: _ClassVar[int]
    SKIPPED_FIELD_NUMBER: _ClassVar[int]
    ESTIMATED_BYTES_FIELD_NUMBER: _ClassVar[int]
    REMOVED_COUNT_FIELD_NUMBER: _ClassVar[int]
    SKIPPED_COUNT_FIELD_NUMBER: _ClassVar[int]
    ERROR_COUNT_FIELD_NUMBER: _ClassVar[int]
    REMOVED_BY_STATUS_FIELD_NUMBER: _ClassVar[int]
    dry_run: bool
    removed: _containers.RepeatedCompositeFieldContainer[ExecutionRetentionItem]
    skipped: _containers.RepeatedCompositeFieldContainer[ExecutionRetentionItem]
    estimated_bytes: int
    removed_count: int
    skipped_count: int
    error_count: int
    removed_by_status: _containers.ScalarMap[str, int]
    def __init__(self, dry_run: _Optional[bool] = ..., removed: _Optional[_Iterable[_Union[ExecutionRetentionItem, _Mapping]]] = ..., skipped: _Optional[_Iterable[_Union[ExecutionRetentionItem, _Mapping]]] = ..., estimated_bytes: _Optional[int] = ..., removed_count: _Optional[int] = ..., skipped_count: _Optional[int] = ..., error_count: _Optional[int] = ..., removed_by_status: _Optional[_Mapping[str, int]] = ...) -> None: ...
