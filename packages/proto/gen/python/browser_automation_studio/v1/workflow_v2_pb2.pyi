from browser_automation_studio.v1 import shared_pb2 as _shared_pb2
from browser_automation_studio.v1 import geometry_pb2 as _geometry_pb2
from browser_automation_studio.v1 import action_pb2 as _action_pb2
from google.protobuf.internal import containers as _containers
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Iterable as _Iterable, Mapping as _Mapping
from typing import ClassVar as _ClassVar, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class WorkflowDefinitionV2(_message.Message):
    __slots__ = ()
    METADATA_FIELD_NUMBER: _ClassVar[int]
    SETTINGS_FIELD_NUMBER: _ClassVar[int]
    NODES_FIELD_NUMBER: _ClassVar[int]
    EDGES_FIELD_NUMBER: _ClassVar[int]
    metadata: WorkflowMetadataV2
    settings: WorkflowSettingsV2
    nodes: _containers.RepeatedCompositeFieldContainer[WorkflowNodeV2]
    edges: _containers.RepeatedCompositeFieldContainer[WorkflowEdgeV2]
    def __init__(self, metadata: _Optional[_Union[WorkflowMetadataV2, _Mapping]] = ..., settings: _Optional[_Union[WorkflowSettingsV2, _Mapping]] = ..., nodes: _Optional[_Iterable[_Union[WorkflowNodeV2, _Mapping]]] = ..., edges: _Optional[_Iterable[_Union[WorkflowEdgeV2, _Mapping]]] = ...) -> None: ...

class WorkflowMetadataV2(_message.Message):
    __slots__ = ()
    class LabelsEntry(_message.Message):
        __slots__ = ()
        KEY_FIELD_NUMBER: _ClassVar[int]
        VALUE_FIELD_NUMBER: _ClassVar[int]
        key: str
        value: str
        def __init__(self, key: _Optional[str] = ..., value: _Optional[str] = ...) -> None: ...
    NAME_FIELD_NUMBER: _ClassVar[int]
    DESCRIPTION_FIELD_NUMBER: _ClassVar[int]
    LABELS_FIELD_NUMBER: _ClassVar[int]
    VERSION_FIELD_NUMBER: _ClassVar[int]
    REQUIREMENT_FIELD_NUMBER: _ClassVar[int]
    OWNER_FIELD_NUMBER: _ClassVar[int]
    name: str
    description: str
    labels: _containers.ScalarMap[str, str]
    version: str
    requirement: str
    owner: str
    def __init__(self, name: _Optional[str] = ..., description: _Optional[str] = ..., labels: _Optional[_Mapping[str, str]] = ..., version: _Optional[str] = ..., requirement: _Optional[str] = ..., owner: _Optional[str] = ...) -> None: ...

class WorkflowSettingsV2(_message.Message):
    __slots__ = ()
    VIEWPORT_WIDTH_FIELD_NUMBER: _ClassVar[int]
    VIEWPORT_HEIGHT_FIELD_NUMBER: _ClassVar[int]
    USER_AGENT_FIELD_NUMBER: _ClassVar[int]
    LOCALE_FIELD_NUMBER: _ClassVar[int]
    HEADLESS_FIELD_NUMBER: _ClassVar[int]
    ENTRY_SELECTOR_FIELD_NUMBER: _ClassVar[int]
    ENTRY_SELECTOR_TIMEOUT_MS_FIELD_NUMBER: _ClassVar[int]
    TIMEOUT_MS_FIELD_NUMBER: _ClassVar[int]
    viewport_width: int
    viewport_height: int
    user_agent: str
    locale: str
    headless: bool
    entry_selector: str
    entry_selector_timeout_ms: int
    timeout_ms: int
    def __init__(self, viewport_width: _Optional[int] = ..., viewport_height: _Optional[int] = ..., user_agent: _Optional[str] = ..., locale: _Optional[str] = ..., headless: _Optional[bool] = ..., entry_selector: _Optional[str] = ..., entry_selector_timeout_ms: _Optional[int] = ..., timeout_ms: _Optional[int] = ...) -> None: ...

class WorkflowNodeV2(_message.Message):
    __slots__ = ()
    ID_FIELD_NUMBER: _ClassVar[int]
    ACTION_FIELD_NUMBER: _ClassVar[int]
    POSITION_FIELD_NUMBER: _ClassVar[int]
    EXECUTION_SETTINGS_FIELD_NUMBER: _ClassVar[int]
    id: str
    action: _action_pb2.ActionDefinition
    position: _geometry_pb2.NodePosition
    execution_settings: NodeExecutionSettings
    def __init__(self, id: _Optional[str] = ..., action: _Optional[_Union[_action_pb2.ActionDefinition, _Mapping]] = ..., position: _Optional[_Union[_geometry_pb2.NodePosition, _Mapping]] = ..., execution_settings: _Optional[_Union[NodeExecutionSettings, _Mapping]] = ...) -> None: ...

class NodeExecutionSettings(_message.Message):
    __slots__ = ()
    TIMEOUT_MS_FIELD_NUMBER: _ClassVar[int]
    WAIT_AFTER_MS_FIELD_NUMBER: _ClassVar[int]
    CONTINUE_ON_ERROR_FIELD_NUMBER: _ClassVar[int]
    RESILIENCE_FIELD_NUMBER: _ClassVar[int]
    timeout_ms: int
    wait_after_ms: int
    continue_on_error: bool
    resilience: ResilienceConfig
    def __init__(self, timeout_ms: _Optional[int] = ..., wait_after_ms: _Optional[int] = ..., continue_on_error: _Optional[bool] = ..., resilience: _Optional[_Union[ResilienceConfig, _Mapping]] = ...) -> None: ...

class ResilienceConfig(_message.Message):
    __slots__ = ()
    MAX_ATTEMPTS_FIELD_NUMBER: _ClassVar[int]
    DELAY_MS_FIELD_NUMBER: _ClassVar[int]
    BACKOFF_FACTOR_FIELD_NUMBER: _ClassVar[int]
    PRECONDITION_SELECTOR_FIELD_NUMBER: _ClassVar[int]
    PRECONDITION_TIMEOUT_MS_FIELD_NUMBER: _ClassVar[int]
    PRECONDITION_WAIT_MS_FIELD_NUMBER: _ClassVar[int]
    SUCCESS_SELECTOR_FIELD_NUMBER: _ClassVar[int]
    SUCCESS_TIMEOUT_MS_FIELD_NUMBER: _ClassVar[int]
    SUCCESS_WAIT_MS_FIELD_NUMBER: _ClassVar[int]
    max_attempts: int
    delay_ms: int
    backoff_factor: float
    precondition_selector: str
    precondition_timeout_ms: int
    precondition_wait_ms: int
    success_selector: str
    success_timeout_ms: int
    success_wait_ms: int
    def __init__(self, max_attempts: _Optional[int] = ..., delay_ms: _Optional[int] = ..., backoff_factor: _Optional[float] = ..., precondition_selector: _Optional[str] = ..., precondition_timeout_ms: _Optional[int] = ..., precondition_wait_ms: _Optional[int] = ..., success_selector: _Optional[str] = ..., success_timeout_ms: _Optional[int] = ..., success_wait_ms: _Optional[int] = ...) -> None: ...

class WorkflowEdgeV2(_message.Message):
    __slots__ = ()
    ID_FIELD_NUMBER: _ClassVar[int]
    SOURCE_FIELD_NUMBER: _ClassVar[int]
    TARGET_FIELD_NUMBER: _ClassVar[int]
    TYPE_FIELD_NUMBER: _ClassVar[int]
    LABEL_FIELD_NUMBER: _ClassVar[int]
    SOURCE_HANDLE_FIELD_NUMBER: _ClassVar[int]
    TARGET_HANDLE_FIELD_NUMBER: _ClassVar[int]
    id: str
    source: str
    target: str
    type: _shared_pb2.WorkflowEdgeType
    label: str
    source_handle: str
    target_handle: str
    def __init__(self, id: _Optional[str] = ..., source: _Optional[str] = ..., target: _Optional[str] = ..., type: _Optional[_Union[_shared_pb2.WorkflowEdgeType, str]] = ..., label: _Optional[str] = ..., source_handle: _Optional[str] = ..., target_handle: _Optional[str] = ...) -> None: ...
