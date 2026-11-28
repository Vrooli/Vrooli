import datetime

from cli.v1 import common_pb2 as _common_pb2
from google.protobuf import timestamp_pb2 as _timestamp_pb2
from google.protobuf.internal import containers as _containers
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Iterable as _Iterable, Mapping as _Mapping
from typing import ClassVar as _ClassVar, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class ScenarioListResponse(_message.Message):
    __slots__ = ("success", "summary", "scenarios", "discovery_failures", "observed_at")
    SUCCESS_FIELD_NUMBER: _ClassVar[int]
    SUMMARY_FIELD_NUMBER: _ClassVar[int]
    SCENARIOS_FIELD_NUMBER: _ClassVar[int]
    DISCOVERY_FAILURES_FIELD_NUMBER: _ClassVar[int]
    OBSERVED_AT_FIELD_NUMBER: _ClassVar[int]
    success: bool
    summary: ScenarioListSummary
    scenarios: _containers.RepeatedCompositeFieldContainer[Scenario]
    discovery_failures: _containers.RepeatedCompositeFieldContainer[_common_pb2.DiscoveryFailure]
    observed_at: _timestamp_pb2.Timestamp
    def __init__(self, success: _Optional[bool] = ..., summary: _Optional[_Union[ScenarioListSummary, _Mapping]] = ..., scenarios: _Optional[_Iterable[_Union[Scenario, _Mapping]]] = ..., discovery_failures: _Optional[_Iterable[_Union[_common_pb2.DiscoveryFailure, _Mapping]]] = ..., observed_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ...) -> None: ...

class ScenarioListSummary(_message.Message):
    __slots__ = ("total_scenarios", "running", "available")
    TOTAL_SCENARIOS_FIELD_NUMBER: _ClassVar[int]
    RUNNING_FIELD_NUMBER: _ClassVar[int]
    AVAILABLE_FIELD_NUMBER: _ClassVar[int]
    total_scenarios: int
    running: int
    available: int
    def __init__(self, total_scenarios: _Optional[int] = ..., running: _Optional[int] = ..., available: _Optional[int] = ...) -> None: ...

class Scenario(_message.Message):
    __slots__ = ("name", "description", "version", "status", "tags", "path", "ports", "health_status")
    NAME_FIELD_NUMBER: _ClassVar[int]
    DESCRIPTION_FIELD_NUMBER: _ClassVar[int]
    VERSION_FIELD_NUMBER: _ClassVar[int]
    STATUS_FIELD_NUMBER: _ClassVar[int]
    TAGS_FIELD_NUMBER: _ClassVar[int]
    PATH_FIELD_NUMBER: _ClassVar[int]
    PORTS_FIELD_NUMBER: _ClassVar[int]
    HEALTH_STATUS_FIELD_NUMBER: _ClassVar[int]
    name: str
    description: str
    version: str
    status: str
    tags: _containers.RepeatedScalarFieldContainer[str]
    path: str
    ports: _containers.RepeatedCompositeFieldContainer[ScenarioPort]
    health_status: str
    def __init__(self, name: _Optional[str] = ..., description: _Optional[str] = ..., version: _Optional[str] = ..., status: _Optional[str] = ..., tags: _Optional[_Iterable[str]] = ..., path: _Optional[str] = ..., ports: _Optional[_Iterable[_Union[ScenarioPort, _Mapping]]] = ..., health_status: _Optional[str] = ...) -> None: ...

class ScenarioPort(_message.Message):
    __slots__ = ("key", "step", "port", "listener_status")
    KEY_FIELD_NUMBER: _ClassVar[int]
    STEP_FIELD_NUMBER: _ClassVar[int]
    PORT_FIELD_NUMBER: _ClassVar[int]
    LISTENER_STATUS_FIELD_NUMBER: _ClassVar[int]
    key: str
    step: str
    port: int
    listener_status: str
    def __init__(self, key: _Optional[str] = ..., step: _Optional[str] = ..., port: _Optional[int] = ..., listener_status: _Optional[str] = ...) -> None: ...
