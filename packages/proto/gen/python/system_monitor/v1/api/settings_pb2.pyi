from system_monitor.v1.domain import settings_pb2 as _settings_pb2
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Mapping as _Mapping
from typing import ClassVar as _ClassVar, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class GetSettingsRequest(_message.Message):
    __slots__ = ()
    def __init__(self) -> None: ...

class GetSettingsResponse(_message.Message):
    __slots__ = ("success", "settings", "error")
    SUCCESS_FIELD_NUMBER: _ClassVar[int]
    SETTINGS_FIELD_NUMBER: _ClassVar[int]
    ERROR_FIELD_NUMBER: _ClassVar[int]
    success: bool
    settings: _settings_pb2.SystemSettings
    error: str
    def __init__(self, success: _Optional[bool] = ..., settings: _Optional[_Union[_settings_pb2.SystemSettings, _Mapping]] = ..., error: _Optional[str] = ...) -> None: ...

class UpdateSettingsRequest(_message.Message):
    __slots__ = ("settings",)
    SETTINGS_FIELD_NUMBER: _ClassVar[int]
    settings: _settings_pb2.SystemSettings
    def __init__(self, settings: _Optional[_Union[_settings_pb2.SystemSettings, _Mapping]] = ...) -> None: ...

class UpdateSettingsResponse(_message.Message):
    __slots__ = ("success", "settings", "error")
    SUCCESS_FIELD_NUMBER: _ClassVar[int]
    SETTINGS_FIELD_NUMBER: _ClassVar[int]
    ERROR_FIELD_NUMBER: _ClassVar[int]
    success: bool
    settings: _settings_pb2.SystemSettings
    error: str
    def __init__(self, success: _Optional[bool] = ..., settings: _Optional[_Union[_settings_pb2.SystemSettings, _Mapping]] = ..., error: _Optional[str] = ...) -> None: ...

class ResetSettingsRequest(_message.Message):
    __slots__ = ()
    def __init__(self) -> None: ...

class ResetSettingsResponse(_message.Message):
    __slots__ = ("success", "settings", "error")
    SUCCESS_FIELD_NUMBER: _ClassVar[int]
    SETTINGS_FIELD_NUMBER: _ClassVar[int]
    ERROR_FIELD_NUMBER: _ClassVar[int]
    success: bool
    settings: _settings_pb2.SystemSettings
    error: str
    def __init__(self, success: _Optional[bool] = ..., settings: _Optional[_Union[_settings_pb2.SystemSettings, _Mapping]] = ..., error: _Optional[str] = ...) -> None: ...

class GetMaintenanceStateRequest(_message.Message):
    __slots__ = ()
    def __init__(self) -> None: ...

class GetMaintenanceStateResponse(_message.Message):
    __slots__ = ("success", "maintenance_state", "error")
    SUCCESS_FIELD_NUMBER: _ClassVar[int]
    MAINTENANCE_STATE_FIELD_NUMBER: _ClassVar[int]
    ERROR_FIELD_NUMBER: _ClassVar[int]
    success: bool
    maintenance_state: str
    error: str
    def __init__(self, success: _Optional[bool] = ..., maintenance_state: _Optional[str] = ..., error: _Optional[str] = ...) -> None: ...

class SetMaintenanceStateRequest(_message.Message):
    __slots__ = ("maintenance_state",)
    MAINTENANCE_STATE_FIELD_NUMBER: _ClassVar[int]
    maintenance_state: str
    def __init__(self, maintenance_state: _Optional[str] = ...) -> None: ...

class SetMaintenanceStateResponse(_message.Message):
    __slots__ = ("success", "maintenance_state", "error")
    SUCCESS_FIELD_NUMBER: _ClassVar[int]
    MAINTENANCE_STATE_FIELD_NUMBER: _ClassVar[int]
    ERROR_FIELD_NUMBER: _ClassVar[int]
    success: bool
    maintenance_state: str
    error: str
    def __init__(self, success: _Optional[bool] = ..., maintenance_state: _Optional[str] = ..., error: _Optional[str] = ...) -> None: ...
