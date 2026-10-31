from google.protobuf.internal import containers as _containers
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Iterable as _Iterable, Mapping as _Mapping
from typing import ClassVar as _ClassVar, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class GetAppStateRequest(_message.Message):
    __slots__ = ("endpoint", "user_agent")
    ENDPOINT_FIELD_NUMBER: _ClassVar[int]
    USER_AGENT_FIELD_NUMBER: _ClassVar[int]
    endpoint: str
    user_agent: str
    def __init__(self, endpoint: _Optional[str] = ..., user_agent: _Optional[str] = ...) -> None: ...

class AppState(_message.Message):
    __slots__ = ("scenario", "display_name", "vapid_public_key", "eligible", "ineligible_reason", "public_url", "subscription", "devices")
    SCENARIO_FIELD_NUMBER: _ClassVar[int]
    DISPLAY_NAME_FIELD_NUMBER: _ClassVar[int]
    VAPID_PUBLIC_KEY_FIELD_NUMBER: _ClassVar[int]
    ELIGIBLE_FIELD_NUMBER: _ClassVar[int]
    INELIGIBLE_REASON_FIELD_NUMBER: _ClassVar[int]
    PUBLIC_URL_FIELD_NUMBER: _ClassVar[int]
    SUBSCRIPTION_FIELD_NUMBER: _ClassVar[int]
    DEVICES_FIELD_NUMBER: _ClassVar[int]
    scenario: str
    display_name: str
    vapid_public_key: str
    eligible: bool
    ineligible_reason: str
    public_url: str
    subscription: AppSubscription
    devices: _containers.RepeatedCompositeFieldContainer[DeviceChoice]
    def __init__(self, scenario: _Optional[str] = ..., display_name: _Optional[str] = ..., vapid_public_key: _Optional[str] = ..., eligible: _Optional[bool] = ..., ineligible_reason: _Optional[str] = ..., public_url: _Optional[str] = ..., subscription: _Optional[_Union[AppSubscription, _Mapping]] = ..., devices: _Optional[_Iterable[_Union[DeviceChoice, _Mapping]]] = ...) -> None: ...

class AppSubscription(_message.Message):
    __slots__ = ("id", "device_group", "device_label", "permission", "confirmed_at", "healthy", "unhealthy_reason", "standalone")
    ID_FIELD_NUMBER: _ClassVar[int]
    DEVICE_GROUP_FIELD_NUMBER: _ClassVar[int]
    DEVICE_LABEL_FIELD_NUMBER: _ClassVar[int]
    PERMISSION_FIELD_NUMBER: _ClassVar[int]
    CONFIRMED_AT_FIELD_NUMBER: _ClassVar[int]
    HEALTHY_FIELD_NUMBER: _ClassVar[int]
    UNHEALTHY_REASON_FIELD_NUMBER: _ClassVar[int]
    STANDALONE_FIELD_NUMBER: _ClassVar[int]
    id: str
    device_group: str
    device_label: str
    permission: str
    confirmed_at: str
    healthy: bool
    unhealthy_reason: str
    standalone: bool
    def __init__(self, id: _Optional[str] = ..., device_group: _Optional[str] = ..., device_label: _Optional[str] = ..., permission: _Optional[str] = ..., confirmed_at: _Optional[str] = ..., healthy: _Optional[bool] = ..., unhealthy_reason: _Optional[str] = ..., standalone: _Optional[bool] = ...) -> None: ...

class DeviceChoice(_message.Message):
    __slots__ = ("group", "label", "last_used", "best_match", "apps")
    GROUP_FIELD_NUMBER: _ClassVar[int]
    LABEL_FIELD_NUMBER: _ClassVar[int]
    LAST_USED_FIELD_NUMBER: _ClassVar[int]
    BEST_MATCH_FIELD_NUMBER: _ClassVar[int]
    APPS_FIELD_NUMBER: _ClassVar[int]
    group: str
    label: str
    last_used: str
    best_match: bool
    apps: _containers.RepeatedScalarFieldContainer[str]
    def __init__(self, group: _Optional[str] = ..., label: _Optional[str] = ..., last_used: _Optional[str] = ..., best_match: _Optional[bool] = ..., apps: _Optional[_Iterable[str]] = ...) -> None: ...

class EnableAppRequest(_message.Message):
    __slots__ = ("endpoint", "p256dh", "auth", "origin", "user_agent", "permission", "pair_with_group", "standalone")
    ENDPOINT_FIELD_NUMBER: _ClassVar[int]
    P256DH_FIELD_NUMBER: _ClassVar[int]
    AUTH_FIELD_NUMBER: _ClassVar[int]
    ORIGIN_FIELD_NUMBER: _ClassVar[int]
    USER_AGENT_FIELD_NUMBER: _ClassVar[int]
    PERMISSION_FIELD_NUMBER: _ClassVar[int]
    PAIR_WITH_GROUP_FIELD_NUMBER: _ClassVar[int]
    STANDALONE_FIELD_NUMBER: _ClassVar[int]
    endpoint: str
    p256dh: str
    auth: str
    origin: str
    user_agent: str
    permission: str
    pair_with_group: str
    standalone: bool
    def __init__(self, endpoint: _Optional[str] = ..., p256dh: _Optional[str] = ..., auth: _Optional[str] = ..., origin: _Optional[str] = ..., user_agent: _Optional[str] = ..., permission: _Optional[str] = ..., pair_with_group: _Optional[str] = ..., standalone: _Optional[bool] = ...) -> None: ...

class ConfirmAppRequest(_message.Message):
    __slots__ = ("endpoint", "permission", "user_agent")
    ENDPOINT_FIELD_NUMBER: _ClassVar[int]
    PERMISSION_FIELD_NUMBER: _ClassVar[int]
    USER_AGENT_FIELD_NUMBER: _ClassVar[int]
    endpoint: str
    permission: str
    user_agent: str
    def __init__(self, endpoint: _Optional[str] = ..., permission: _Optional[str] = ..., user_agent: _Optional[str] = ...) -> None: ...

class ConfirmAppResponse(_message.Message):
    __slots__ = ("registered", "healthy")
    REGISTERED_FIELD_NUMBER: _ClassVar[int]
    HEALTHY_FIELD_NUMBER: _ClassVar[int]
    registered: bool
    healthy: bool
    def __init__(self, registered: _Optional[bool] = ..., healthy: _Optional[bool] = ...) -> None: ...

class DisableAppRequest(_message.Message):
    __slots__ = ("endpoint",)
    ENDPOINT_FIELD_NUMBER: _ClassVar[int]
    endpoint: str
    def __init__(self, endpoint: _Optional[str] = ...) -> None: ...

class DisableAppResponse(_message.Message):
    __slots__ = ()
    def __init__(self) -> None: ...

class ListDeviceAppsRequest(_message.Message):
    __slots__ = ()
    def __init__(self) -> None: ...

class ListDeviceAppsResponse(_message.Message):
    __slots__ = ("devices",)
    DEVICES_FIELD_NUMBER: _ClassVar[int]
    devices: _containers.RepeatedCompositeFieldContainer[DeviceApps]
    def __init__(self, devices: _Optional[_Iterable[_Union[DeviceApps, _Mapping]]] = ...) -> None: ...

class DeviceApps(_message.Message):
    __slots__ = ("group", "label", "last_used", "apps")
    GROUP_FIELD_NUMBER: _ClassVar[int]
    LABEL_FIELD_NUMBER: _ClassVar[int]
    LAST_USED_FIELD_NUMBER: _ClassVar[int]
    APPS_FIELD_NUMBER: _ClassVar[int]
    group: str
    label: str
    last_used: str
    apps: _containers.RepeatedCompositeFieldContainer[DeviceAppEntry]
    def __init__(self, group: _Optional[str] = ..., label: _Optional[str] = ..., last_used: _Optional[str] = ..., apps: _Optional[_Iterable[_Union[DeviceAppEntry, _Mapping]]] = ...) -> None: ...

class DeviceAppEntry(_message.Message):
    __slots__ = ("subscription_id", "scenario", "display_name", "healthy", "status", "confirmed_at")
    SUBSCRIPTION_ID_FIELD_NUMBER: _ClassVar[int]
    SCENARIO_FIELD_NUMBER: _ClassVar[int]
    DISPLAY_NAME_FIELD_NUMBER: _ClassVar[int]
    HEALTHY_FIELD_NUMBER: _ClassVar[int]
    STATUS_FIELD_NUMBER: _ClassVar[int]
    CONFIRMED_AT_FIELD_NUMBER: _ClassVar[int]
    subscription_id: str
    scenario: str
    display_name: str
    healthy: bool
    status: str
    confirmed_at: str
    def __init__(self, subscription_id: _Optional[str] = ..., scenario: _Optional[str] = ..., display_name: _Optional[str] = ..., healthy: _Optional[bool] = ..., status: _Optional[str] = ..., confirmed_at: _Optional[str] = ...) -> None: ...

class RemoveDeviceAppRequest(_message.Message):
    __slots__ = ("subscription_id",)
    SUBSCRIPTION_ID_FIELD_NUMBER: _ClassVar[int]
    subscription_id: str
    def __init__(self, subscription_id: _Optional[str] = ...) -> None: ...

class RemoveDeviceAppResponse(_message.Message):
    __slots__ = ()
    def __init__(self) -> None: ...
