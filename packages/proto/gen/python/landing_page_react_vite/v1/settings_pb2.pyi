import datetime

from google.protobuf import timestamp_pb2 as _timestamp_pb2
from google.protobuf.internal import enum_type_wrapper as _enum_type_wrapper
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Mapping as _Mapping
from typing import ClassVar as _ClassVar, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class ConfigSource(int, metaclass=_enum_type_wrapper.EnumTypeWrapper):
    __slots__ = ()
    CONFIG_SOURCE_UNSPECIFIED: _ClassVar[ConfigSource]
    CONFIG_SOURCE_ENV: _ClassVar[ConfigSource]
    CONFIG_SOURCE_DATABASE: _ClassVar[ConfigSource]
CONFIG_SOURCE_UNSPECIFIED: ConfigSource
CONFIG_SOURCE_ENV: ConfigSource
CONFIG_SOURCE_DATABASE: ConfigSource

class StripeConfigSnapshot(_message.Message):
    __slots__ = ()
    PUBLISHABLE_KEY_PREVIEW_FIELD_NUMBER: _ClassVar[int]
    PUBLISHABLE_KEY_SET_FIELD_NUMBER: _ClassVar[int]
    SECRET_KEY_SET_FIELD_NUMBER: _ClassVar[int]
    WEBHOOK_SECRET_SET_FIELD_NUMBER: _ClassVar[int]
    SOURCE_FIELD_NUMBER: _ClassVar[int]
    publishable_key_preview: str
    publishable_key_set: bool
    secret_key_set: bool
    webhook_secret_set: bool
    source: ConfigSource
    def __init__(self, publishable_key_preview: _Optional[str] = ..., publishable_key_set: _Optional[bool] = ..., secret_key_set: _Optional[bool] = ..., webhook_secret_set: _Optional[bool] = ..., source: _Optional[_Union[ConfigSource, str]] = ...) -> None: ...

class StripeSettings(_message.Message):
    __slots__ = ()
    PUBLISHABLE_KEY_FIELD_NUMBER: _ClassVar[int]
    SECRET_KEY_FIELD_NUMBER: _ClassVar[int]
    WEBHOOK_SECRET_FIELD_NUMBER: _ClassVar[int]
    DASHBOARD_URL_FIELD_NUMBER: _ClassVar[int]
    UPDATED_AT_FIELD_NUMBER: _ClassVar[int]
    publishable_key: str
    secret_key: str
    webhook_secret: str
    dashboard_url: str
    updated_at: _timestamp_pb2.Timestamp
    def __init__(self, publishable_key: _Optional[str] = ..., secret_key: _Optional[str] = ..., webhook_secret: _Optional[str] = ..., dashboard_url: _Optional[str] = ..., updated_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ...) -> None: ...

class GetStripeSettingsRequest(_message.Message):
    __slots__ = ()
    def __init__(self) -> None: ...

class GetStripeSettingsResponse(_message.Message):
    __slots__ = ()
    SETTINGS_FIELD_NUMBER: _ClassVar[int]
    SNAPSHOT_FIELD_NUMBER: _ClassVar[int]
    settings: StripeSettings
    snapshot: StripeConfigSnapshot
    def __init__(self, settings: _Optional[_Union[StripeSettings, _Mapping]] = ..., snapshot: _Optional[_Union[StripeConfigSnapshot, _Mapping]] = ...) -> None: ...

class UpdateStripeSettingsRequest(_message.Message):
    __slots__ = ()
    PUBLISHABLE_KEY_FIELD_NUMBER: _ClassVar[int]
    SECRET_KEY_FIELD_NUMBER: _ClassVar[int]
    WEBHOOK_SECRET_FIELD_NUMBER: _ClassVar[int]
    DASHBOARD_URL_FIELD_NUMBER: _ClassVar[int]
    publishable_key: str
    secret_key: str
    webhook_secret: str
    dashboard_url: str
    def __init__(self, publishable_key: _Optional[str] = ..., secret_key: _Optional[str] = ..., webhook_secret: _Optional[str] = ..., dashboard_url: _Optional[str] = ...) -> None: ...

class UpdateStripeSettingsResponse(_message.Message):
    __slots__ = ()
    SETTINGS_FIELD_NUMBER: _ClassVar[int]
    SNAPSHOT_FIELD_NUMBER: _ClassVar[int]
    settings: StripeSettings
    snapshot: StripeConfigSnapshot
    def __init__(self, settings: _Optional[_Union[StripeSettings, _Mapping]] = ..., snapshot: _Optional[_Union[StripeConfigSnapshot, _Mapping]] = ...) -> None: ...
