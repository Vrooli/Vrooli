from google.protobuf import struct_pb2 as _struct_pb2
from browser_automation_studio.v1 import shared_pb2 as _shared_pb2
from google.protobuf.internal import containers as _containers
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Iterable as _Iterable, Mapping as _Mapping
from typing import ClassVar as _ClassVar, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class WorkflowDefinition(_message.Message):
    __slots__ = ()
    class MetadataEntry(_message.Message):
        __slots__ = ()
        KEY_FIELD_NUMBER: _ClassVar[int]
        VALUE_FIELD_NUMBER: _ClassVar[int]
        key: str
        value: _struct_pb2.Value
        def __init__(self, key: _Optional[str] = ..., value: _Optional[_Union[_struct_pb2.Value, _Mapping]] = ...) -> None: ...
    class SettingsEntry(_message.Message):
        __slots__ = ()
        KEY_FIELD_NUMBER: _ClassVar[int]
        VALUE_FIELD_NUMBER: _ClassVar[int]
        key: str
        value: _struct_pb2.Value
        def __init__(self, key: _Optional[str] = ..., value: _Optional[_Union[_struct_pb2.Value, _Mapping]] = ...) -> None: ...
    NODES_FIELD_NUMBER: _ClassVar[int]
    EDGES_FIELD_NUMBER: _ClassVar[int]
    METADATA_FIELD_NUMBER: _ClassVar[int]
    SETTINGS_FIELD_NUMBER: _ClassVar[int]
    METADATA_TYPED_FIELD_NUMBER: _ClassVar[int]
    SETTINGS_TYPED_FIELD_NUMBER: _ClassVar[int]
    nodes: _containers.RepeatedCompositeFieldContainer[WorkflowNode]
    edges: _containers.RepeatedCompositeFieldContainer[WorkflowEdge]
    metadata: _containers.MessageMap[str, _struct_pb2.Value]
    settings: _containers.MessageMap[str, _struct_pb2.Value]
    metadata_typed: WorkflowMetadata
    settings_typed: WorkflowSettings
    def __init__(self, nodes: _Optional[_Iterable[_Union[WorkflowNode, _Mapping]]] = ..., edges: _Optional[_Iterable[_Union[WorkflowEdge, _Mapping]]] = ..., metadata: _Optional[_Mapping[str, _struct_pb2.Value]] = ..., settings: _Optional[_Mapping[str, _struct_pb2.Value]] = ..., metadata_typed: _Optional[_Union[WorkflowMetadata, _Mapping]] = ..., settings_typed: _Optional[_Union[WorkflowSettings, _Mapping]] = ...) -> None: ...

class WorkflowNode(_message.Message):
    __slots__ = ()
    ID_FIELD_NUMBER: _ClassVar[int]
    TYPE_FIELD_NUMBER: _ClassVar[int]
    DATA_FIELD_NUMBER: _ClassVar[int]
    DATA_TYPED_FIELD_NUMBER: _ClassVar[int]
    id: str
    type: _shared_pb2.StepType
    data: _struct_pb2.Struct
    data_typed: _shared_pb2.JsonObject
    def __init__(self, id: _Optional[str] = ..., type: _Optional[_Union[_shared_pb2.StepType, str]] = ..., data: _Optional[_Union[_struct_pb2.Struct, _Mapping]] = ..., data_typed: _Optional[_Union[_shared_pb2.JsonObject, _Mapping]] = ...) -> None: ...

class WorkflowEdge(_message.Message):
    __slots__ = ()
    ID_FIELD_NUMBER: _ClassVar[int]
    SOURCE_FIELD_NUMBER: _ClassVar[int]
    TARGET_FIELD_NUMBER: _ClassVar[int]
    TYPE_FIELD_NUMBER: _ClassVar[int]
    DATA_FIELD_NUMBER: _ClassVar[int]
    DATA_TYPED_FIELD_NUMBER: _ClassVar[int]
    id: str
    source: str
    target: str
    type: str
    data: _struct_pb2.Struct
    data_typed: _shared_pb2.JsonObject
    def __init__(self, id: _Optional[str] = ..., source: _Optional[str] = ..., target: _Optional[str] = ..., type: _Optional[str] = ..., data: _Optional[_Union[_struct_pb2.Struct, _Mapping]] = ..., data_typed: _Optional[_Union[_shared_pb2.JsonObject, _Mapping]] = ...) -> None: ...

class WorkflowMetadata(_message.Message):
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
    name: str
    description: str
    labels: _containers.ScalarMap[str, str]
    version: str
    def __init__(self, name: _Optional[str] = ..., description: _Optional[str] = ..., labels: _Optional[_Mapping[str, str]] = ..., version: _Optional[str] = ...) -> None: ...

class WorkflowSettings(_message.Message):
    __slots__ = ()
    class ExtrasEntry(_message.Message):
        __slots__ = ()
        KEY_FIELD_NUMBER: _ClassVar[int]
        VALUE_FIELD_NUMBER: _ClassVar[int]
        key: str
        value: _struct_pb2.Value
        def __init__(self, key: _Optional[str] = ..., value: _Optional[_Union[_struct_pb2.Value, _Mapping]] = ...) -> None: ...
    VIEWPORT_WIDTH_FIELD_NUMBER: _ClassVar[int]
    VIEWPORT_HEIGHT_FIELD_NUMBER: _ClassVar[int]
    USER_AGENT_FIELD_NUMBER: _ClassVar[int]
    LOCALE_FIELD_NUMBER: _ClassVar[int]
    TIMEOUT_SECONDS_FIELD_NUMBER: _ClassVar[int]
    HEADLESS_FIELD_NUMBER: _ClassVar[int]
    EXTRAS_FIELD_NUMBER: _ClassVar[int]
    viewport_width: int
    viewport_height: int
    user_agent: str
    locale: str
    timeout_seconds: int
    headless: bool
    extras: _containers.MessageMap[str, _struct_pb2.Value]
    def __init__(self, viewport_width: _Optional[int] = ..., viewport_height: _Optional[int] = ..., user_agent: _Optional[str] = ..., locale: _Optional[str] = ..., timeout_seconds: _Optional[int] = ..., headless: _Optional[bool] = ..., extras: _Optional[_Mapping[str, _struct_pb2.Value]] = ...) -> None: ...
