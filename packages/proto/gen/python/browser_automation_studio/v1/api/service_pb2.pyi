import datetime

from google.protobuf import timestamp_pb2 as _timestamp_pb2
from google.api import annotations_pb2 as _annotations_pb2
from browser_automation_studio.v1.base import shared_pb2 as _shared_pb2
from browser_automation_studio.v1.actions import action_pb2 as _action_pb2
from browser_automation_studio.v1.workflows import definition_pb2 as _definition_pb2
from browser_automation_studio.v1.execution import execution_pb2 as _execution_pb2
from google.protobuf.internal import containers as _containers
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Iterable as _Iterable, Mapping as _Mapping
from typing import ClassVar as _ClassVar, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class WorkflowSummary(_message.Message):
    __slots__ = ()
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
    __slots__ = ()
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
    __slots__ = ()
    WORKFLOWS_FIELD_NUMBER: _ClassVar[int]
    workflows: _containers.RepeatedCompositeFieldContainer[WorkflowSummary]
    def __init__(self, workflows: _Optional[_Iterable[_Union[WorkflowSummary, _Mapping]]] = ...) -> None: ...

class WorkflowVersionList(_message.Message):
    __slots__ = ()
    VERSIONS_FIELD_NUMBER: _ClassVar[int]
    versions: _containers.RepeatedCompositeFieldContainer[WorkflowVersion]
    def __init__(self, versions: _Optional[_Iterable[_Union[WorkflowVersion, _Mapping]]] = ...) -> None: ...

class ListWorkflowsRequest(_message.Message):
    __slots__ = ()
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
    __slots__ = ()
    WORKFLOWS_FIELD_NUMBER: _ClassVar[int]
    TOTAL_FIELD_NUMBER: _ClassVar[int]
    HAS_MORE_FIELD_NUMBER: _ClassVar[int]
    workflows: _containers.RepeatedCompositeFieldContainer[WorkflowSummary]
    total: int
    has_more: bool
    def __init__(self, workflows: _Optional[_Iterable[_Union[WorkflowSummary, _Mapping]]] = ..., total: _Optional[int] = ..., has_more: _Optional[bool] = ...) -> None: ...

class GetWorkflowRequest(_message.Message):
    __slots__ = ()
    WORKFLOW_ID_FIELD_NUMBER: _ClassVar[int]
    VERSION_FIELD_NUMBER: _ClassVar[int]
    workflow_id: str
    version: int
    def __init__(self, workflow_id: _Optional[str] = ..., version: _Optional[int] = ...) -> None: ...

class GetWorkflowResponse(_message.Message):
    __slots__ = ()
    WORKFLOW_FIELD_NUMBER: _ClassVar[int]
    workflow: WorkflowSummary
    def __init__(self, workflow: _Optional[_Union[WorkflowSummary, _Mapping]] = ...) -> None: ...

class CreateWorkflowRequest(_message.Message):
    __slots__ = ()
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
    __slots__ = ()
    WORKFLOW_FIELD_NUMBER: _ClassVar[int]
    FLOW_DEFINITION_FIELD_NUMBER: _ClassVar[int]
    workflow: WorkflowSummary
    flow_definition: _definition_pb2.WorkflowDefinitionV2
    def __init__(self, workflow: _Optional[_Union[WorkflowSummary, _Mapping]] = ..., flow_definition: _Optional[_Union[_definition_pb2.WorkflowDefinitionV2, _Mapping]] = ...) -> None: ...

class UpdateWorkflowRequest(_message.Message):
    __slots__ = ()
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
    __slots__ = ()
    WORKFLOW_FIELD_NUMBER: _ClassVar[int]
    FLOW_DEFINITION_FIELD_NUMBER: _ClassVar[int]
    workflow: WorkflowSummary
    flow_definition: _definition_pb2.WorkflowDefinitionV2
    def __init__(self, workflow: _Optional[_Union[WorkflowSummary, _Mapping]] = ..., flow_definition: _Optional[_Union[_definition_pb2.WorkflowDefinitionV2, _Mapping]] = ...) -> None: ...

class DeleteWorkflowRequest(_message.Message):
    __slots__ = ()
    WORKFLOW_ID_FIELD_NUMBER: _ClassVar[int]
    workflow_id: str
    def __init__(self, workflow_id: _Optional[str] = ...) -> None: ...

class DeleteWorkflowResponse(_message.Message):
    __slots__ = ()
    SUCCESS_FIELD_NUMBER: _ClassVar[int]
    WORKFLOW_ID_FIELD_NUMBER: _ClassVar[int]
    success: bool
    workflow_id: str
    def __init__(self, success: _Optional[bool] = ..., workflow_id: _Optional[str] = ...) -> None: ...

class ExecuteWorkflowRequest(_message.Message):
    __slots__ = ()
    WAIT_FOR_COMPLETION_FIELD_NUMBER: _ClassVar[int]
    WORKFLOW_ID_FIELD_NUMBER: _ClassVar[int]
    WORKFLOW_VERSION_FIELD_NUMBER: _ClassVar[int]
    PARAMETERS_FIELD_NUMBER: _ClassVar[int]
    wait_for_completion: bool
    workflow_id: str
    workflow_version: int
    parameters: _execution_pb2.ExecutionParameters
    def __init__(self, wait_for_completion: _Optional[bool] = ..., workflow_id: _Optional[str] = ..., workflow_version: _Optional[int] = ..., parameters: _Optional[_Union[_execution_pb2.ExecutionParameters, _Mapping]] = ...) -> None: ...

class ExecuteWorkflowResponse(_message.Message):
    __slots__ = ()
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
    __slots__ = ()
    WORKFLOW_ID_FIELD_NUMBER: _ClassVar[int]
    STATUS_FIELD_NUMBER: _ClassVar[int]
    LIMIT_FIELD_NUMBER: _ClassVar[int]
    OFFSET_FIELD_NUMBER: _ClassVar[int]
    workflow_id: str
    status: _shared_pb2.ExecutionStatus
    limit: int
    offset: int
    def __init__(self, workflow_id: _Optional[str] = ..., status: _Optional[_Union[_shared_pb2.ExecutionStatus, str]] = ..., limit: _Optional[int] = ..., offset: _Optional[int] = ...) -> None: ...

class ListExecutionsResponse(_message.Message):
    __slots__ = ()
    EXECUTIONS_FIELD_NUMBER: _ClassVar[int]
    TOTAL_FIELD_NUMBER: _ClassVar[int]
    HAS_MORE_FIELD_NUMBER: _ClassVar[int]
    executions: _containers.RepeatedCompositeFieldContainer[_execution_pb2.Execution]
    total: int
    has_more: bool
    def __init__(self, executions: _Optional[_Iterable[_Union[_execution_pb2.Execution, _Mapping]]] = ..., total: _Optional[int] = ..., has_more: _Optional[bool] = ...) -> None: ...

class GetExecutionRequest(_message.Message):
    __slots__ = ()
    EXECUTION_ID_FIELD_NUMBER: _ClassVar[int]
    execution_id: str
    def __init__(self, execution_id: _Optional[str] = ...) -> None: ...

class GetExecutionResponse(_message.Message):
    __slots__ = ()
    EXECUTION_FIELD_NUMBER: _ClassVar[int]
    execution: _execution_pb2.Execution
    def __init__(self, execution: _Optional[_Union[_execution_pb2.Execution, _Mapping]] = ...) -> None: ...

class ValidateWorkflowRequest(_message.Message):
    __slots__ = ()
    WORKFLOW_FIELD_NUMBER: _ClassVar[int]
    workflow: _definition_pb2.WorkflowDefinitionV2
    def __init__(self, workflow: _Optional[_Union[_definition_pb2.WorkflowDefinitionV2, _Mapping]] = ...) -> None: ...

class ValidateWorkflowResponse(_message.Message):
    __slots__ = ()
    RESULT_FIELD_NUMBER: _ClassVar[int]
    result: WorkflowValidationResult
    def __init__(self, result: _Optional[_Union[WorkflowValidationResult, _Mapping]] = ...) -> None: ...

class WorkflowValidationIssue(_message.Message):
    __slots__ = ()
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
    __slots__ = ()
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
    __slots__ = ()
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
    __slots__ = ()
    WORKFLOW_FIELD_NUMBER: _ClassVar[int]
    RESTORED_VERSION_FIELD_NUMBER: _ClassVar[int]
    workflow: WorkflowSummary
    restored_version: WorkflowVersion
    def __init__(self, workflow: _Optional[_Union[WorkflowSummary, _Mapping]] = ..., restored_version: _Optional[_Union[WorkflowVersion, _Mapping]] = ...) -> None: ...
