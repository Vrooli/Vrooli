from buf.validate import validate_pb2 as _validate_pb2
from google.protobuf import empty_pb2 as _empty_pb2
from google.protobuf import struct_pb2 as _struct_pb2
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Mapping as _Mapping
from typing import ClassVar as _ClassVar, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class StateRequest(_message.Message):
    __slots__ = ("scenario_name",)
    SCENARIO_NAME_FIELD_NUMBER: _ClassVar[int]
    scenario_name: str
    def __init__(self, scenario_name: _Optional[str] = ...) -> None: ...

class LoadScenarioStateRequest(_message.Message):
    __slots__ = ("scenario_name", "include_logs", "validate_manifest", "manifest_path")
    SCENARIO_NAME_FIELD_NUMBER: _ClassVar[int]
    INCLUDE_LOGS_FIELD_NUMBER: _ClassVar[int]
    VALIDATE_MANIFEST_FIELD_NUMBER: _ClassVar[int]
    MANIFEST_PATH_FIELD_NUMBER: _ClassVar[int]
    scenario_name: str
    include_logs: bool
    validate_manifest: bool
    manifest_path: str
    def __init__(self, scenario_name: _Optional[str] = ..., include_logs: _Optional[bool] = ..., validate_manifest: _Optional[bool] = ..., manifest_path: _Optional[str] = ...) -> None: ...

class StateResponse(_message.Message):
    __slots__ = ("payload", "found")
    PAYLOAD_FIELD_NUMBER: _ClassVar[int]
    FOUND_FIELD_NUMBER: _ClassVar[int]
    payload: _struct_pb2.Struct
    found: bool
    def __init__(self, payload: _Optional[_Union[_struct_pb2.Struct, _Mapping]] = ..., found: _Optional[bool] = ...) -> None: ...

class SaveScenarioStateRequest(_message.Message):
    __slots__ = ("scenario_name", "payload")
    SCENARIO_NAME_FIELD_NUMBER: _ClassVar[int]
    PAYLOAD_FIELD_NUMBER: _ClassVar[int]
    scenario_name: str
    payload: _struct_pb2.Struct
    def __init__(self, scenario_name: _Optional[str] = ..., payload: _Optional[_Union[_struct_pb2.Struct, _Mapping]] = ...) -> None: ...

class ScenarioStateLogRequest(_message.Message):
    __slots__ = ("scenario_name", "service_id")
    SCENARIO_NAME_FIELD_NUMBER: _ClassVar[int]
    SERVICE_ID_FIELD_NUMBER: _ClassVar[int]
    scenario_name: str
    service_id: str
    def __init__(self, scenario_name: _Optional[str] = ..., service_id: _Optional[str] = ...) -> None: ...

class ScenarioStateLogResponse(_message.Message):
    __slots__ = ("payload", "found")
    PAYLOAD_FIELD_NUMBER: _ClassVar[int]
    FOUND_FIELD_NUMBER: _ClassVar[int]
    payload: _struct_pb2.Struct
    found: bool
    def __init__(self, payload: _Optional[_Union[_struct_pb2.Struct, _Mapping]] = ..., found: _Optional[bool] = ...) -> None: ...

class CheckScenarioStateRequest(_message.Message):
    __slots__ = ("scenario_name", "current_config")
    SCENARIO_NAME_FIELD_NUMBER: _ClassVar[int]
    CURRENT_CONFIG_FIELD_NUMBER: _ClassVar[int]
    scenario_name: str
    current_config: _struct_pb2.Struct
    def __init__(self, scenario_name: _Optional[str] = ..., current_config: _Optional[_Union[_struct_pb2.Struct, _Mapping]] = ...) -> None: ...

class InvalidateScenarioStateRequest(_message.Message):
    __slots__ = ("scenario_name", "from_stage", "reason")
    SCENARIO_NAME_FIELD_NUMBER: _ClassVar[int]
    FROM_STAGE_FIELD_NUMBER: _ClassVar[int]
    REASON_FIELD_NUMBER: _ClassVar[int]
    scenario_name: str
    from_stage: str
    reason: str
    def __init__(self, scenario_name: _Optional[str] = ..., from_stage: _Optional[str] = ..., reason: _Optional[str] = ...) -> None: ...

class StateOperationResponse(_message.Message):
    __slots__ = ("payload",)
    PAYLOAD_FIELD_NUMBER: _ClassVar[int]
    payload: _struct_pb2.Struct
    def __init__(self, payload: _Optional[_Union[_struct_pb2.Struct, _Mapping]] = ...) -> None: ...
