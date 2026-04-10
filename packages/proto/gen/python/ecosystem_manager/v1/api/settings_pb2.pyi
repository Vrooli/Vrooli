from buf.validate import validate_pb2 as _validate_pb2
from ecosystem_manager.v1.domain import settings_pb2 as _settings_pb2
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
    __slots__ = ("theme", "condensed_mode", "slots", "cooldown_seconds", "active", "max_turns", "allowed_tools", "skip_permissions", "task_timeout", "idle_timeout_cap", "runner_type", "recycler")
    THEME_FIELD_NUMBER: _ClassVar[int]
    CONDENSED_MODE_FIELD_NUMBER: _ClassVar[int]
    SLOTS_FIELD_NUMBER: _ClassVar[int]
    COOLDOWN_SECONDS_FIELD_NUMBER: _ClassVar[int]
    ACTIVE_FIELD_NUMBER: _ClassVar[int]
    MAX_TURNS_FIELD_NUMBER: _ClassVar[int]
    ALLOWED_TOOLS_FIELD_NUMBER: _ClassVar[int]
    SKIP_PERMISSIONS_FIELD_NUMBER: _ClassVar[int]
    TASK_TIMEOUT_FIELD_NUMBER: _ClassVar[int]
    IDLE_TIMEOUT_CAP_FIELD_NUMBER: _ClassVar[int]
    RUNNER_TYPE_FIELD_NUMBER: _ClassVar[int]
    RECYCLER_FIELD_NUMBER: _ClassVar[int]
    theme: str
    condensed_mode: bool
    slots: int
    cooldown_seconds: int
    active: bool
    max_turns: int
    allowed_tools: str
    skip_permissions: bool
    task_timeout: int
    idle_timeout_cap: int
    runner_type: str
    recycler: _settings_pb2.RecyclerSettings
    def __init__(self, theme: _Optional[str] = ..., condensed_mode: _Optional[bool] = ..., slots: _Optional[int] = ..., cooldown_seconds: _Optional[int] = ..., active: _Optional[bool] = ..., max_turns: _Optional[int] = ..., allowed_tools: _Optional[str] = ..., skip_permissions: _Optional[bool] = ..., task_timeout: _Optional[int] = ..., idle_timeout_cap: _Optional[int] = ..., runner_type: _Optional[str] = ..., recycler: _Optional[_Union[_settings_pb2.RecyclerSettings, _Mapping]] = ...) -> None: ...
