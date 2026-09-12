from buf.validate import validate_pb2 as _validate_pb2
from google.protobuf import struct_pb2 as _struct_pb2
from google.protobuf.internal import containers as _containers
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Iterable as _Iterable, Mapping as _Mapping
from typing import ClassVar as _ClassVar, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class GetObservabilityRequest(_message.Message):
    __slots__ = ("depth", "no_cache")
    DEPTH_FIELD_NUMBER: _ClassVar[int]
    NO_CACHE_FIELD_NUMBER: _ClassVar[int]
    depth: str
    no_cache: bool
    def __init__(self, depth: _Optional[str] = ..., no_cache: _Optional[bool] = ...) -> None: ...

class GetObservabilityResponse(_message.Message):
    __slots__ = ("snapshot",)
    SNAPSHOT_FIELD_NUMBER: _ClassVar[int]
    snapshot: _struct_pb2.Struct
    def __init__(self, snapshot: _Optional[_Union[_struct_pb2.Struct, _Mapping]] = ...) -> None: ...

class RefreshObservabilityRequest(_message.Message):
    __slots__ = ()
    def __init__(self) -> None: ...

class RefreshObservabilityResponse(_message.Message):
    __slots__ = ("result",)
    RESULT_FIELD_NUMBER: _ClassVar[int]
    result: _struct_pb2.Struct
    def __init__(self, result: _Optional[_Union[_struct_pb2.Struct, _Mapping]] = ...) -> None: ...

class RunDiagnosticsRequest(_message.Message):
    __slots__ = ("options",)
    OPTIONS_FIELD_NUMBER: _ClassVar[int]
    options: _struct_pb2.Struct
    def __init__(self, options: _Optional[_Union[_struct_pb2.Struct, _Mapping]] = ...) -> None: ...

class RunDiagnosticsResponse(_message.Message):
    __slots__ = ("result",)
    RESULT_FIELD_NUMBER: _ClassVar[int]
    result: _struct_pb2.Struct
    def __init__(self, result: _Optional[_Union[_struct_pb2.Struct, _Mapping]] = ...) -> None: ...

class GetSessionListRequest(_message.Message):
    __slots__ = ()
    def __init__(self) -> None: ...

class GetSessionListResponse(_message.Message):
    __slots__ = ("result",)
    RESULT_FIELD_NUMBER: _ClassVar[int]
    result: _struct_pb2.Struct
    def __init__(self, result: _Optional[_Union[_struct_pb2.Struct, _Mapping]] = ...) -> None: ...

class RunCleanupRequest(_message.Message):
    __slots__ = ()
    def __init__(self) -> None: ...

class RunCleanupResponse(_message.Message):
    __slots__ = ("result",)
    RESULT_FIELD_NUMBER: _ClassVar[int]
    result: _struct_pb2.Struct
    def __init__(self, result: _Optional[_Union[_struct_pb2.Struct, _Mapping]] = ...) -> None: ...

class GetMetricsRequest(_message.Message):
    __slots__ = ()
    def __init__(self) -> None: ...

class GetMetricsResponse(_message.Message):
    __slots__ = ("result",)
    RESULT_FIELD_NUMBER: _ClassVar[int]
    result: _struct_pb2.Struct
    def __init__(self, result: _Optional[_Union[_struct_pb2.Struct, _Mapping]] = ...) -> None: ...

class RunPipelineTestRequest(_message.Message):
    __slots__ = ("options",)
    OPTIONS_FIELD_NUMBER: _ClassVar[int]
    options: _struct_pb2.Struct
    def __init__(self, options: _Optional[_Union[_struct_pb2.Struct, _Mapping]] = ...) -> None: ...

class RunPipelineTestResponse(_message.Message):
    __slots__ = ("result",)
    RESULT_FIELD_NUMBER: _ClassVar[int]
    result: _struct_pb2.Struct
    def __init__(self, result: _Optional[_Union[_struct_pb2.Struct, _Mapping]] = ...) -> None: ...

class GetConfigRuntimeRequest(_message.Message):
    __slots__ = ()
    def __init__(self) -> None: ...

class GetConfigRuntimeResponse(_message.Message):
    __slots__ = ("result",)
    RESULT_FIELD_NUMBER: _ClassVar[int]
    result: _struct_pb2.Struct
    def __init__(self, result: _Optional[_Union[_struct_pb2.Struct, _Mapping]] = ...) -> None: ...

class UpdateConfigRequest(_message.Message):
    __slots__ = ("env_var", "value")
    ENV_VAR_FIELD_NUMBER: _ClassVar[int]
    VALUE_FIELD_NUMBER: _ClassVar[int]
    env_var: str
    value: str
    def __init__(self, env_var: _Optional[str] = ..., value: _Optional[str] = ...) -> None: ...

class UpdateConfigResponse(_message.Message):
    __slots__ = ("result",)
    RESULT_FIELD_NUMBER: _ClassVar[int]
    result: _struct_pb2.Struct
    def __init__(self, result: _Optional[_Union[_struct_pb2.Struct, _Mapping]] = ...) -> None: ...

class ResetConfigRequest(_message.Message):
    __slots__ = ("env_var",)
    ENV_VAR_FIELD_NUMBER: _ClassVar[int]
    env_var: str
    def __init__(self, env_var: _Optional[str] = ...) -> None: ...

class ResetConfigResponse(_message.Message):
    __slots__ = ("result",)
    RESULT_FIELD_NUMBER: _ClassVar[int]
    result: _struct_pb2.Struct
    def __init__(self, result: _Optional[_Union[_struct_pb2.Struct, _Mapping]] = ...) -> None: ...

class GetDebugModeRequest(_message.Message):
    __slots__ = ()
    def __init__(self) -> None: ...

class SetDebugModeRequest(_message.Message):
    __slots__ = ("enabled", "components", "duration_minutes")
    ENABLED_FIELD_NUMBER: _ClassVar[int]
    COMPONENTS_FIELD_NUMBER: _ClassVar[int]
    DURATION_MINUTES_FIELD_NUMBER: _ClassVar[int]
    enabled: bool
    components: _containers.RepeatedScalarFieldContainer[str]
    duration_minutes: int
    def __init__(self, enabled: _Optional[bool] = ..., components: _Optional[_Iterable[str]] = ..., duration_minutes: _Optional[int] = ...) -> None: ...

class DebugModeState(_message.Message):
    __slots__ = ("enabled", "components", "expires_at", "remaining_mins")
    ENABLED_FIELD_NUMBER: _ClassVar[int]
    COMPONENTS_FIELD_NUMBER: _ClassVar[int]
    EXPIRES_AT_FIELD_NUMBER: _ClassVar[int]
    REMAINING_MINS_FIELD_NUMBER: _ClassVar[int]
    enabled: bool
    components: _containers.RepeatedScalarFieldContainer[str]
    expires_at: str
    remaining_mins: int
    def __init__(self, enabled: _Optional[bool] = ..., components: _Optional[_Iterable[str]] = ..., expires_at: _Optional[str] = ..., remaining_mins: _Optional[int] = ...) -> None: ...
