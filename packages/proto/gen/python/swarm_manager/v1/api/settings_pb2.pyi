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
    __slots__ = ("theme", "custom_focus", "insights_enabled", "insights_auto_analyze")
    THEME_FIELD_NUMBER: _ClassVar[int]
    CUSTOM_FOCUS_FIELD_NUMBER: _ClassVar[int]
    INSIGHTS_ENABLED_FIELD_NUMBER: _ClassVar[int]
    INSIGHTS_AUTO_ANALYZE_FIELD_NUMBER: _ClassVar[int]
    theme: str
    custom_focus: str
    insights_enabled: bool
    insights_auto_analyze: bool
    def __init__(self, theme: _Optional[str] = ..., custom_focus: _Optional[str] = ..., insights_enabled: _Optional[bool] = ..., insights_auto_analyze: _Optional[bool] = ...) -> None: ...
