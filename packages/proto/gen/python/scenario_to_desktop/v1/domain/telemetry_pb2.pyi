from buf.validate import validate_pb2 as _validate_pb2
from google.protobuf import struct_pb2 as _struct_pb2
from google.protobuf.internal import containers as _containers
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Iterable as _Iterable, Mapping as _Mapping
from typing import ClassVar as _ClassVar, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class TelemetryScenarioRequest(_message.Message):
    __slots__ = ("scenario_name",)
    SCENARIO_NAME_FIELD_NUMBER: _ClassVar[int]
    scenario_name: str
    def __init__(self, scenario_name: _Optional[str] = ...) -> None: ...

class IngestTelemetryRequest(_message.Message):
    __slots__ = ("scenario_name", "deployment_mode", "source", "events")
    SCENARIO_NAME_FIELD_NUMBER: _ClassVar[int]
    DEPLOYMENT_MODE_FIELD_NUMBER: _ClassVar[int]
    SOURCE_FIELD_NUMBER: _ClassVar[int]
    EVENTS_FIELD_NUMBER: _ClassVar[int]
    scenario_name: str
    deployment_mode: str
    source: str
    events: _containers.RepeatedCompositeFieldContainer[_struct_pb2.Struct]
    def __init__(self, scenario_name: _Optional[str] = ..., deployment_mode: _Optional[str] = ..., source: _Optional[str] = ..., events: _Optional[_Iterable[_Union[_struct_pb2.Struct, _Mapping]]] = ...) -> None: ...

class IngestTelemetryResponse(_message.Message):
    __slots__ = ("output_path", "events_ingested")
    OUTPUT_PATH_FIELD_NUMBER: _ClassVar[int]
    EVENTS_INGESTED_FIELD_NUMBER: _ClassVar[int]
    output_path: str
    events_ingested: int
    def __init__(self, output_path: _Optional[str] = ..., events_ingested: _Optional[int] = ...) -> None: ...

class TelemetryPayloadResponse(_message.Message):
    __slots__ = ("payload",)
    PAYLOAD_FIELD_NUMBER: _ClassVar[int]
    payload: _struct_pb2.Struct
    def __init__(self, payload: _Optional[_Union[_struct_pb2.Struct, _Mapping]] = ...) -> None: ...

class TelemetryTailRequest(_message.Message):
    __slots__ = ("scenario_name", "limit")
    SCENARIO_NAME_FIELD_NUMBER: _ClassVar[int]
    LIMIT_FIELD_NUMBER: _ClassVar[int]
    scenario_name: str
    limit: int
    def __init__(self, scenario_name: _Optional[str] = ..., limit: _Optional[int] = ...) -> None: ...

class TelemetryDeleteResponse(_message.Message):
    __slots__ = ("scenario_name", "deleted")
    SCENARIO_NAME_FIELD_NUMBER: _ClassVar[int]
    DELETED_FIELD_NUMBER: _ClassVar[int]
    scenario_name: str
    deleted: bool
    def __init__(self, scenario_name: _Optional[str] = ..., deleted: _Optional[bool] = ...) -> None: ...
