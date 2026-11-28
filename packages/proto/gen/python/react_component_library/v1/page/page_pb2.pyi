from google.protobuf import struct_pb2 as _struct_pb2
from google.protobuf.internal import containers as _containers
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Iterable as _Iterable, Mapping as _Mapping
from typing import ClassVar as _ClassVar, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class InspectRequest(_message.Message):
    __slots__ = ("scenario", "route", "wait_selector")
    SCENARIO_FIELD_NUMBER: _ClassVar[int]
    ROUTE_FIELD_NUMBER: _ClassVar[int]
    WAIT_SELECTOR_FIELD_NUMBER: _ClassVar[int]
    scenario: str
    route: str
    wait_selector: str
    def __init__(self, scenario: _Optional[str] = ..., route: _Optional[str] = ..., wait_selector: _Optional[str] = ...) -> None: ...

class AssetSource(_message.Message):
    __slots__ = ("status", "stamped_asset", "stamped_version", "library_id", "resolved_version", "source_path", "resolution_rule", "reason")
    STATUS_FIELD_NUMBER: _ClassVar[int]
    STAMPED_ASSET_FIELD_NUMBER: _ClassVar[int]
    STAMPED_VERSION_FIELD_NUMBER: _ClassVar[int]
    LIBRARY_ID_FIELD_NUMBER: _ClassVar[int]
    RESOLVED_VERSION_FIELD_NUMBER: _ClassVar[int]
    SOURCE_PATH_FIELD_NUMBER: _ClassVar[int]
    RESOLUTION_RULE_FIELD_NUMBER: _ClassVar[int]
    REASON_FIELD_NUMBER: _ClassVar[int]
    status: str
    stamped_asset: str
    stamped_version: str
    library_id: str
    resolved_version: str
    source_path: str
    resolution_rule: str
    reason: str
    def __init__(self, status: _Optional[str] = ..., stamped_asset: _Optional[str] = ..., stamped_version: _Optional[str] = ..., library_id: _Optional[str] = ..., resolved_version: _Optional[str] = ..., source_path: _Optional[str] = ..., resolution_rule: _Optional[str] = ..., reason: _Optional[str] = ...) -> None: ...

class PageNode(_message.Message):
    __slots__ = ("observation", "source", "children")
    OBSERVATION_FIELD_NUMBER: _ClassVar[int]
    SOURCE_FIELD_NUMBER: _ClassVar[int]
    CHILDREN_FIELD_NUMBER: _ClassVar[int]
    observation: _struct_pb2.Struct
    source: AssetSource
    children: _containers.RepeatedCompositeFieldContainer[PageNode]
    def __init__(self, observation: _Optional[_Union[_struct_pb2.Struct, _Mapping]] = ..., source: _Optional[_Union[AssetSource, _Mapping]] = ..., children: _Optional[_Iterable[_Union[PageNode, _Mapping]]] = ...) -> None: ...

class InspectResponse(_message.Message):
    __slots__ = ("scenario", "route", "url", "execution_id", "screenshot_path", "screenshot_url", "tree", "node_count", "stamped_count", "resolved_count", "warnings")
    SCENARIO_FIELD_NUMBER: _ClassVar[int]
    ROUTE_FIELD_NUMBER: _ClassVar[int]
    URL_FIELD_NUMBER: _ClassVar[int]
    EXECUTION_ID_FIELD_NUMBER: _ClassVar[int]
    SCREENSHOT_PATH_FIELD_NUMBER: _ClassVar[int]
    SCREENSHOT_URL_FIELD_NUMBER: _ClassVar[int]
    TREE_FIELD_NUMBER: _ClassVar[int]
    NODE_COUNT_FIELD_NUMBER: _ClassVar[int]
    STAMPED_COUNT_FIELD_NUMBER: _ClassVar[int]
    RESOLVED_COUNT_FIELD_NUMBER: _ClassVar[int]
    WARNINGS_FIELD_NUMBER: _ClassVar[int]
    scenario: str
    route: str
    url: str
    execution_id: str
    screenshot_path: str
    screenshot_url: str
    tree: PageNode
    node_count: int
    stamped_count: int
    resolved_count: int
    warnings: _containers.RepeatedScalarFieldContainer[str]
    def __init__(self, scenario: _Optional[str] = ..., route: _Optional[str] = ..., url: _Optional[str] = ..., execution_id: _Optional[str] = ..., screenshot_path: _Optional[str] = ..., screenshot_url: _Optional[str] = ..., tree: _Optional[_Union[PageNode, _Mapping]] = ..., node_count: _Optional[int] = ..., stamped_count: _Optional[int] = ..., resolved_count: _Optional[int] = ..., warnings: _Optional[_Iterable[str]] = ...) -> None: ...
