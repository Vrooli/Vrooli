from buf.validate import validate_pb2 as _validate_pb2
from swarm_manager.v1.domain import settings_pb2 as _settings_pb2
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Mapping as _Mapping
from typing import ClassVar as _ClassVar, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class SettingsResponse(_message.Message):
    __slots__ = ("settings",)
    SETTINGS_FIELD_NUMBER: _ClassVar[int]
    settings: _settings_pb2.Settings
    def __init__(self, settings: _Optional[_Union[_settings_pb2.Settings, _Mapping]] = ...) -> None: ...

class UpdateSettingsRequest(_message.Message):
    __slots__ = ("theme",)
    THEME_FIELD_NUMBER: _ClassVar[int]
    theme: str
    def __init__(self, theme: _Optional[str] = ...) -> None: ...
