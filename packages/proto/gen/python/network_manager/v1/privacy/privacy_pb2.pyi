from google.protobuf.internal import containers as _containers
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Iterable as _Iterable, Mapping as _Mapping
from typing import ClassVar as _ClassVar, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class RetentionSettings(_message.Message):
    __slots__ = ("query_log_days", "snapshot_days", "experiment_days", "profile")
    QUERY_LOG_DAYS_FIELD_NUMBER: _ClassVar[int]
    SNAPSHOT_DAYS_FIELD_NUMBER: _ClassVar[int]
    EXPERIMENT_DAYS_FIELD_NUMBER: _ClassVar[int]
    PROFILE_FIELD_NUMBER: _ClassVar[int]
    query_log_days: int
    snapshot_days: int
    experiment_days: int
    profile: str
    def __init__(self, query_log_days: _Optional[int] = ..., snapshot_days: _Optional[int] = ..., experiment_days: _Optional[int] = ..., profile: _Optional[str] = ...) -> None: ...

class VisibilitySettings(_message.Message):
    __slots__ = ("show_query_domains", "show_device_history", "household_mode", "notes")
    SHOW_QUERY_DOMAINS_FIELD_NUMBER: _ClassVar[int]
    SHOW_DEVICE_HISTORY_FIELD_NUMBER: _ClassVar[int]
    HOUSEHOLD_MODE_FIELD_NUMBER: _ClassVar[int]
    NOTES_FIELD_NUMBER: _ClassVar[int]
    show_query_domains: bool
    show_device_history: bool
    household_mode: bool
    notes: _containers.RepeatedScalarFieldContainer[str]
    def __init__(self, show_query_domains: _Optional[bool] = ..., show_device_history: _Optional[bool] = ..., household_mode: _Optional[bool] = ..., notes: _Optional[_Iterable[str]] = ...) -> None: ...

class GetRetentionSettingsRequest(_message.Message):
    __slots__ = ()
    def __init__(self) -> None: ...

class GetRetentionSettingsResponse(_message.Message):
    __slots__ = ("settings",)
    SETTINGS_FIELD_NUMBER: _ClassVar[int]
    settings: RetentionSettings
    def __init__(self, settings: _Optional[_Union[RetentionSettings, _Mapping]] = ...) -> None: ...

class UpdateRetentionSettingsRequest(_message.Message):
    __slots__ = ("settings",)
    SETTINGS_FIELD_NUMBER: _ClassVar[int]
    settings: RetentionSettings
    def __init__(self, settings: _Optional[_Union[RetentionSettings, _Mapping]] = ...) -> None: ...

class UpdateRetentionSettingsResponse(_message.Message):
    __slots__ = ("settings",)
    SETTINGS_FIELD_NUMBER: _ClassVar[int]
    settings: RetentionSettings
    def __init__(self, settings: _Optional[_Union[RetentionSettings, _Mapping]] = ...) -> None: ...

class GetVisibilitySettingsRequest(_message.Message):
    __slots__ = ()
    def __init__(self) -> None: ...

class GetVisibilitySettingsResponse(_message.Message):
    __slots__ = ("settings",)
    SETTINGS_FIELD_NUMBER: _ClassVar[int]
    settings: VisibilitySettings
    def __init__(self, settings: _Optional[_Union[VisibilitySettings, _Mapping]] = ...) -> None: ...
