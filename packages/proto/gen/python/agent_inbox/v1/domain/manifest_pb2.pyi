import datetime

from google.protobuf import timestamp_pb2 as _timestamp_pb2
from agent_inbox.v1.domain import tool_pb2 as _tool_pb2
from google.protobuf.internal import containers as _containers
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Iterable as _Iterable, Mapping as _Mapping
from typing import ClassVar as _ClassVar, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class ToolManifest(_message.Message):
    __slots__ = ()
    PROTOCOL_VERSION_FIELD_NUMBER: _ClassVar[int]
    SCENARIO_FIELD_NUMBER: _ClassVar[int]
    TOOLS_FIELD_NUMBER: _ClassVar[int]
    CATEGORIES_FIELD_NUMBER: _ClassVar[int]
    GENERATED_AT_FIELD_NUMBER: _ClassVar[int]
    CAPABILITIES_FIELD_NUMBER: _ClassVar[int]
    protocol_version: str
    scenario: ScenarioInfo
    tools: _containers.RepeatedCompositeFieldContainer[_tool_pb2.ToolDefinition]
    categories: _containers.RepeatedCompositeFieldContainer[_tool_pb2.ToolCategory]
    generated_at: _timestamp_pb2.Timestamp
    capabilities: ManifestCapabilities
    def __init__(self, protocol_version: _Optional[str] = ..., scenario: _Optional[_Union[ScenarioInfo, _Mapping]] = ..., tools: _Optional[_Iterable[_Union[_tool_pb2.ToolDefinition, _Mapping]]] = ..., categories: _Optional[_Iterable[_Union[_tool_pb2.ToolCategory, _Mapping]]] = ..., generated_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., capabilities: _Optional[_Union[ManifestCapabilities, _Mapping]] = ...) -> None: ...

class ScenarioInfo(_message.Message):
    __slots__ = ()
    NAME_FIELD_NUMBER: _ClassVar[int]
    VERSION_FIELD_NUMBER: _ClassVar[int]
    DESCRIPTION_FIELD_NUMBER: _ClassVar[int]
    BASE_URL_FIELD_NUMBER: _ClassVar[int]
    HEALTH_ENDPOINT_FIELD_NUMBER: _ClassVar[int]
    EXECUTE_ENDPOINT_TEMPLATE_FIELD_NUMBER: _ClassVar[int]
    name: str
    version: str
    description: str
    base_url: str
    health_endpoint: str
    execute_endpoint_template: str
    def __init__(self, name: _Optional[str] = ..., version: _Optional[str] = ..., description: _Optional[str] = ..., base_url: _Optional[str] = ..., health_endpoint: _Optional[str] = ..., execute_endpoint_template: _Optional[str] = ...) -> None: ...

class ManifestCapabilities(_message.Message):
    __slots__ = ()
    SUPPORTS_ASYNC_FIELD_NUMBER: _ClassVar[int]
    SUPPORTS_WEBSOCKET_STATUS_FIELD_NUMBER: _ClassVar[int]
    WEBSOCKET_ENDPOINT_FIELD_NUMBER: _ClassVar[int]
    SUPPORTS_BATCH_EXECUTION_FIELD_NUMBER: _ClassVar[int]
    SUPPORTS_CACHING_FIELD_NUMBER: _ClassVar[int]
    CACHE_TTL_SECONDS_FIELD_NUMBER: _ClassVar[int]
    supports_async: bool
    supports_websocket_status: bool
    websocket_endpoint: str
    supports_batch_execution: bool
    supports_caching: bool
    cache_ttl_seconds: int
    def __init__(self, supports_async: _Optional[bool] = ..., supports_websocket_status: _Optional[bool] = ..., websocket_endpoint: _Optional[str] = ..., supports_batch_execution: _Optional[bool] = ..., supports_caching: _Optional[bool] = ..., cache_ttl_seconds: _Optional[int] = ...) -> None: ...
