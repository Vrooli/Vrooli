from buf.validate import validate_pb2 as _validate_pb2
from google.protobuf.internal import containers as _containers
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Iterable as _Iterable, Mapping as _Mapping
from typing import ClassVar as _ClassVar, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class Cookie(_message.Message):
    __slots__ = ("name", "value", "value_masked", "domain", "path", "expires", "http_only", "secure", "same_site")
    NAME_FIELD_NUMBER: _ClassVar[int]
    VALUE_FIELD_NUMBER: _ClassVar[int]
    VALUE_MASKED_FIELD_NUMBER: _ClassVar[int]
    DOMAIN_FIELD_NUMBER: _ClassVar[int]
    PATH_FIELD_NUMBER: _ClassVar[int]
    EXPIRES_FIELD_NUMBER: _ClassVar[int]
    HTTP_ONLY_FIELD_NUMBER: _ClassVar[int]
    SECURE_FIELD_NUMBER: _ClassVar[int]
    SAME_SITE_FIELD_NUMBER: _ClassVar[int]
    name: str
    value: str
    value_masked: bool
    domain: str
    path: str
    expires: float
    http_only: bool
    secure: bool
    same_site: str
    def __init__(self, name: _Optional[str] = ..., value: _Optional[str] = ..., value_masked: _Optional[bool] = ..., domain: _Optional[str] = ..., path: _Optional[str] = ..., expires: _Optional[float] = ..., http_only: _Optional[bool] = ..., secure: _Optional[bool] = ..., same_site: _Optional[str] = ...) -> None: ...

class LocalStorageItem(_message.Message):
    __slots__ = ("name", "value")
    NAME_FIELD_NUMBER: _ClassVar[int]
    VALUE_FIELD_NUMBER: _ClassVar[int]
    name: str
    value: str
    def __init__(self, name: _Optional[str] = ..., value: _Optional[str] = ...) -> None: ...

class Origin(_message.Message):
    __slots__ = ("origin", "local_storage")
    ORIGIN_FIELD_NUMBER: _ClassVar[int]
    LOCAL_STORAGE_FIELD_NUMBER: _ClassVar[int]
    origin: str
    local_storage: _containers.RepeatedCompositeFieldContainer[LocalStorageItem]
    def __init__(self, origin: _Optional[str] = ..., local_storage: _Optional[_Iterable[_Union[LocalStorageItem, _Mapping]]] = ...) -> None: ...

class StorageStats(_message.Message):
    __slots__ = ("cookie_count", "local_storage_count", "origin_count")
    COOKIE_COUNT_FIELD_NUMBER: _ClassVar[int]
    LOCAL_STORAGE_COUNT_FIELD_NUMBER: _ClassVar[int]
    ORIGIN_COUNT_FIELD_NUMBER: _ClassVar[int]
    cookie_count: int
    local_storage_count: int
    origin_count: int
    def __init__(self, cookie_count: _Optional[int] = ..., local_storage_count: _Optional[int] = ..., origin_count: _Optional[int] = ...) -> None: ...

class GetStorageStateRequest(_message.Message):
    __slots__ = ("profile_id",)
    PROFILE_ID_FIELD_NUMBER: _ClassVar[int]
    profile_id: str
    def __init__(self, profile_id: _Optional[str] = ...) -> None: ...

class GetStorageStateResponse(_message.Message):
    __slots__ = ("cookies", "origins", "stats")
    COOKIES_FIELD_NUMBER: _ClassVar[int]
    ORIGINS_FIELD_NUMBER: _ClassVar[int]
    STATS_FIELD_NUMBER: _ClassVar[int]
    cookies: _containers.RepeatedCompositeFieldContainer[Cookie]
    origins: _containers.RepeatedCompositeFieldContainer[Origin]
    stats: StorageStats
    def __init__(self, cookies: _Optional[_Iterable[_Union[Cookie, _Mapping]]] = ..., origins: _Optional[_Iterable[_Union[Origin, _Mapping]]] = ..., stats: _Optional[_Union[StorageStats, _Mapping]] = ...) -> None: ...

class ClearAllStorageRequest(_message.Message):
    __slots__ = ("profile_id",)
    PROFILE_ID_FIELD_NUMBER: _ClassVar[int]
    profile_id: str
    def __init__(self, profile_id: _Optional[str] = ...) -> None: ...

class ClearAllCookiesRequest(_message.Message):
    __slots__ = ("profile_id",)
    PROFILE_ID_FIELD_NUMBER: _ClassVar[int]
    profile_id: str
    def __init__(self, profile_id: _Optional[str] = ...) -> None: ...

class DeleteCookiesByDomainRequest(_message.Message):
    __slots__ = ("profile_id", "domain")
    PROFILE_ID_FIELD_NUMBER: _ClassVar[int]
    DOMAIN_FIELD_NUMBER: _ClassVar[int]
    profile_id: str
    domain: str
    def __init__(self, profile_id: _Optional[str] = ..., domain: _Optional[str] = ...) -> None: ...

class DeleteCookieRequest(_message.Message):
    __slots__ = ("profile_id", "domain", "name")
    PROFILE_ID_FIELD_NUMBER: _ClassVar[int]
    DOMAIN_FIELD_NUMBER: _ClassVar[int]
    NAME_FIELD_NUMBER: _ClassVar[int]
    profile_id: str
    domain: str
    name: str
    def __init__(self, profile_id: _Optional[str] = ..., domain: _Optional[str] = ..., name: _Optional[str] = ...) -> None: ...

class ClearAllLocalStorageRequest(_message.Message):
    __slots__ = ("profile_id",)
    PROFILE_ID_FIELD_NUMBER: _ClassVar[int]
    profile_id: str
    def __init__(self, profile_id: _Optional[str] = ...) -> None: ...

class DeleteLocalStorageByOriginRequest(_message.Message):
    __slots__ = ("profile_id", "origin")
    PROFILE_ID_FIELD_NUMBER: _ClassVar[int]
    ORIGIN_FIELD_NUMBER: _ClassVar[int]
    profile_id: str
    origin: str
    def __init__(self, profile_id: _Optional[str] = ..., origin: _Optional[str] = ...) -> None: ...

class DeleteLocalStorageItemRequest(_message.Message):
    __slots__ = ("profile_id", "origin", "name")
    PROFILE_ID_FIELD_NUMBER: _ClassVar[int]
    ORIGIN_FIELD_NUMBER: _ClassVar[int]
    NAME_FIELD_NUMBER: _ClassVar[int]
    profile_id: str
    origin: str
    name: str
    def __init__(self, profile_id: _Optional[str] = ..., origin: _Optional[str] = ..., name: _Optional[str] = ...) -> None: ...

class StorageMutationResponse(_message.Message):
    __slots__ = ("status",)
    STATUS_FIELD_NUMBER: _ClassVar[int]
    status: str
    def __init__(self, status: _Optional[str] = ...) -> None: ...

class ServiceWorkerInfo(_message.Message):
    __slots__ = ("registration_id", "scope_url", "script_url", "status", "version_id")
    REGISTRATION_ID_FIELD_NUMBER: _ClassVar[int]
    SCOPE_URL_FIELD_NUMBER: _ClassVar[int]
    SCRIPT_URL_FIELD_NUMBER: _ClassVar[int]
    STATUS_FIELD_NUMBER: _ClassVar[int]
    VERSION_ID_FIELD_NUMBER: _ClassVar[int]
    registration_id: str
    scope_url: str
    script_url: str
    status: str
    version_id: str
    def __init__(self, registration_id: _Optional[str] = ..., scope_url: _Optional[str] = ..., script_url: _Optional[str] = ..., status: _Optional[str] = ..., version_id: _Optional[str] = ...) -> None: ...

class ServiceWorkerDomainOverride(_message.Message):
    __slots__ = ("domain", "mode")
    DOMAIN_FIELD_NUMBER: _ClassVar[int]
    MODE_FIELD_NUMBER: _ClassVar[int]
    domain: str
    mode: str
    def __init__(self, domain: _Optional[str] = ..., mode: _Optional[str] = ...) -> None: ...

class ServiceWorkerControl(_message.Message):
    __slots__ = ("mode", "domain_overrides", "blocked_domains")
    MODE_FIELD_NUMBER: _ClassVar[int]
    DOMAIN_OVERRIDES_FIELD_NUMBER: _ClassVar[int]
    BLOCKED_DOMAINS_FIELD_NUMBER: _ClassVar[int]
    mode: str
    domain_overrides: _containers.RepeatedCompositeFieldContainer[ServiceWorkerDomainOverride]
    blocked_domains: _containers.RepeatedScalarFieldContainer[str]
    def __init__(self, mode: _Optional[str] = ..., domain_overrides: _Optional[_Iterable[_Union[ServiceWorkerDomainOverride, _Mapping]]] = ..., blocked_domains: _Optional[_Iterable[str]] = ...) -> None: ...

class GetServiceWorkersRequest(_message.Message):
    __slots__ = ("profile_id",)
    PROFILE_ID_FIELD_NUMBER: _ClassVar[int]
    profile_id: str
    def __init__(self, profile_id: _Optional[str] = ...) -> None: ...

class GetServiceWorkersResponse(_message.Message):
    __slots__ = ("session_id", "workers", "control", "message")
    SESSION_ID_FIELD_NUMBER: _ClassVar[int]
    WORKERS_FIELD_NUMBER: _ClassVar[int]
    CONTROL_FIELD_NUMBER: _ClassVar[int]
    MESSAGE_FIELD_NUMBER: _ClassVar[int]
    session_id: str
    workers: _containers.RepeatedCompositeFieldContainer[ServiceWorkerInfo]
    control: ServiceWorkerControl
    message: str
    def __init__(self, session_id: _Optional[str] = ..., workers: _Optional[_Iterable[_Union[ServiceWorkerInfo, _Mapping]]] = ..., control: _Optional[_Union[ServiceWorkerControl, _Mapping]] = ..., message: _Optional[str] = ...) -> None: ...

class ClearAllServiceWorkersRequest(_message.Message):
    __slots__ = ("profile_id",)
    PROFILE_ID_FIELD_NUMBER: _ClassVar[int]
    profile_id: str
    def __init__(self, profile_id: _Optional[str] = ...) -> None: ...

class ClearAllServiceWorkersResponse(_message.Message):
    __slots__ = ("session_id", "unregistered_count", "message")
    SESSION_ID_FIELD_NUMBER: _ClassVar[int]
    UNREGISTERED_COUNT_FIELD_NUMBER: _ClassVar[int]
    MESSAGE_FIELD_NUMBER: _ClassVar[int]
    session_id: str
    unregistered_count: int
    message: str
    def __init__(self, session_id: _Optional[str] = ..., unregistered_count: _Optional[int] = ..., message: _Optional[str] = ...) -> None: ...

class DeleteServiceWorkerRequest(_message.Message):
    __slots__ = ("profile_id", "scope_url")
    PROFILE_ID_FIELD_NUMBER: _ClassVar[int]
    SCOPE_URL_FIELD_NUMBER: _ClassVar[int]
    profile_id: str
    scope_url: str
    def __init__(self, profile_id: _Optional[str] = ..., scope_url: _Optional[str] = ...) -> None: ...

class DeleteServiceWorkerResponse(_message.Message):
    __slots__ = ("session_id", "unregistered")
    SESSION_ID_FIELD_NUMBER: _ClassVar[int]
    UNREGISTERED_FIELD_NUMBER: _ClassVar[int]
    session_id: str
    unregistered: str
    def __init__(self, session_id: _Optional[str] = ..., unregistered: _Optional[str] = ...) -> None: ...

class HistoryEntry(_message.Message):
    __slots__ = ("id", "url", "title", "timestamp", "thumbnail")
    ID_FIELD_NUMBER: _ClassVar[int]
    URL_FIELD_NUMBER: _ClassVar[int]
    TITLE_FIELD_NUMBER: _ClassVar[int]
    TIMESTAMP_FIELD_NUMBER: _ClassVar[int]
    THUMBNAIL_FIELD_NUMBER: _ClassVar[int]
    id: str
    url: str
    title: str
    timestamp: str
    thumbnail: str
    def __init__(self, id: _Optional[str] = ..., url: _Optional[str] = ..., title: _Optional[str] = ..., timestamp: _Optional[str] = ..., thumbnail: _Optional[str] = ...) -> None: ...

class HistorySettings(_message.Message):
    __slots__ = ("max_entries", "retention_days", "capture_thumbnails")
    MAX_ENTRIES_FIELD_NUMBER: _ClassVar[int]
    RETENTION_DAYS_FIELD_NUMBER: _ClassVar[int]
    CAPTURE_THUMBNAILS_FIELD_NUMBER: _ClassVar[int]
    max_entries: int
    retention_days: int
    capture_thumbnails: bool
    def __init__(self, max_entries: _Optional[int] = ..., retention_days: _Optional[int] = ..., capture_thumbnails: _Optional[bool] = ...) -> None: ...

class HistoryStats(_message.Message):
    __slots__ = ("total_entries", "oldest_entry", "newest_entry")
    TOTAL_ENTRIES_FIELD_NUMBER: _ClassVar[int]
    OLDEST_ENTRY_FIELD_NUMBER: _ClassVar[int]
    NEWEST_ENTRY_FIELD_NUMBER: _ClassVar[int]
    total_entries: int
    oldest_entry: str
    newest_entry: str
    def __init__(self, total_entries: _Optional[int] = ..., oldest_entry: _Optional[str] = ..., newest_entry: _Optional[str] = ...) -> None: ...

class GetHistoryRequest(_message.Message):
    __slots__ = ("profile_id",)
    PROFILE_ID_FIELD_NUMBER: _ClassVar[int]
    profile_id: str
    def __init__(self, profile_id: _Optional[str] = ...) -> None: ...

class GetHistoryResponse(_message.Message):
    __slots__ = ("entries", "settings", "stats")
    ENTRIES_FIELD_NUMBER: _ClassVar[int]
    SETTINGS_FIELD_NUMBER: _ClassVar[int]
    STATS_FIELD_NUMBER: _ClassVar[int]
    entries: _containers.RepeatedCompositeFieldContainer[HistoryEntry]
    settings: HistorySettings
    stats: HistoryStats
    def __init__(self, entries: _Optional[_Iterable[_Union[HistoryEntry, _Mapping]]] = ..., settings: _Optional[_Union[HistorySettings, _Mapping]] = ..., stats: _Optional[_Union[HistoryStats, _Mapping]] = ...) -> None: ...

class ClearHistoryRequest(_message.Message):
    __slots__ = ("profile_id",)
    PROFILE_ID_FIELD_NUMBER: _ClassVar[int]
    profile_id: str
    def __init__(self, profile_id: _Optional[str] = ...) -> None: ...

class DeleteHistoryEntryRequest(_message.Message):
    __slots__ = ("profile_id", "entry_id")
    PROFILE_ID_FIELD_NUMBER: _ClassVar[int]
    ENTRY_ID_FIELD_NUMBER: _ClassVar[int]
    profile_id: str
    entry_id: str
    def __init__(self, profile_id: _Optional[str] = ..., entry_id: _Optional[str] = ...) -> None: ...

class HistoryMutationResponse(_message.Message):
    __slots__ = ("status", "id")
    STATUS_FIELD_NUMBER: _ClassVar[int]
    ID_FIELD_NUMBER: _ClassVar[int]
    status: str
    id: str
    def __init__(self, status: _Optional[str] = ..., id: _Optional[str] = ...) -> None: ...

class UpdateHistorySettingsRequest(_message.Message):
    __slots__ = ("profile_id", "settings")
    PROFILE_ID_FIELD_NUMBER: _ClassVar[int]
    SETTINGS_FIELD_NUMBER: _ClassVar[int]
    profile_id: str
    settings: HistorySettings
    def __init__(self, profile_id: _Optional[str] = ..., settings: _Optional[_Union[HistorySettings, _Mapping]] = ...) -> None: ...

class UpdateHistorySettingsResponse(_message.Message):
    __slots__ = ("settings", "history_count")
    SETTINGS_FIELD_NUMBER: _ClassVar[int]
    HISTORY_COUNT_FIELD_NUMBER: _ClassVar[int]
    settings: HistorySettings
    history_count: int
    def __init__(self, settings: _Optional[_Union[HistorySettings, _Mapping]] = ..., history_count: _Optional[int] = ...) -> None: ...

class NavigateToHistoryURLRequest(_message.Message):
    __slots__ = ("profile_id", "url", "wait_until", "timeout_ms")
    PROFILE_ID_FIELD_NUMBER: _ClassVar[int]
    URL_FIELD_NUMBER: _ClassVar[int]
    WAIT_UNTIL_FIELD_NUMBER: _ClassVar[int]
    TIMEOUT_MS_FIELD_NUMBER: _ClassVar[int]
    profile_id: str
    url: str
    wait_until: str
    timeout_ms: int
    def __init__(self, profile_id: _Optional[str] = ..., url: _Optional[str] = ..., wait_until: _Optional[str] = ..., timeout_ms: _Optional[int] = ...) -> None: ...

class NavigateToHistoryURLResponse(_message.Message):
    __slots__ = ("url", "title", "can_go_back", "can_go_forward")
    URL_FIELD_NUMBER: _ClassVar[int]
    TITLE_FIELD_NUMBER: _ClassVar[int]
    CAN_GO_BACK_FIELD_NUMBER: _ClassVar[int]
    CAN_GO_FORWARD_FIELD_NUMBER: _ClassVar[int]
    url: str
    title: str
    can_go_back: bool
    can_go_forward: bool
    def __init__(self, url: _Optional[str] = ..., title: _Optional[str] = ..., can_go_back: _Optional[bool] = ..., can_go_forward: _Optional[bool] = ...) -> None: ...

class TabInfo(_message.Message):
    __slots__ = ("url", "title", "is_active", "order")
    URL_FIELD_NUMBER: _ClassVar[int]
    TITLE_FIELD_NUMBER: _ClassVar[int]
    IS_ACTIVE_FIELD_NUMBER: _ClassVar[int]
    ORDER_FIELD_NUMBER: _ClassVar[int]
    url: str
    title: str
    is_active: bool
    order: int
    def __init__(self, url: _Optional[str] = ..., title: _Optional[str] = ..., is_active: _Optional[bool] = ..., order: _Optional[int] = ...) -> None: ...

class GetSessionTabsRequest(_message.Message):
    __slots__ = ("profile_id",)
    PROFILE_ID_FIELD_NUMBER: _ClassVar[int]
    profile_id: str
    def __init__(self, profile_id: _Optional[str] = ...) -> None: ...

class GetSessionTabsResponse(_message.Message):
    __slots__ = ("tabs",)
    TABS_FIELD_NUMBER: _ClassVar[int]
    tabs: _containers.RepeatedCompositeFieldContainer[TabInfo]
    def __init__(self, tabs: _Optional[_Iterable[_Union[TabInfo, _Mapping]]] = ...) -> None: ...

class ClearSessionTabsRequest(_message.Message):
    __slots__ = ("profile_id",)
    PROFILE_ID_FIELD_NUMBER: _ClassVar[int]
    profile_id: str
    def __init__(self, profile_id: _Optional[str] = ...) -> None: ...

class ClearSessionTabsResponse(_message.Message):
    __slots__ = ("status", "profile_id")
    STATUS_FIELD_NUMBER: _ClassVar[int]
    PROFILE_ID_FIELD_NUMBER: _ClassVar[int]
    status: str
    profile_id: str
    def __init__(self, status: _Optional[str] = ..., profile_id: _Optional[str] = ...) -> None: ...

class DeleteSessionTabRequest(_message.Message):
    __slots__ = ("profile_id", "order")
    PROFILE_ID_FIELD_NUMBER: _ClassVar[int]
    ORDER_FIELD_NUMBER: _ClassVar[int]
    profile_id: str
    order: int
    def __init__(self, profile_id: _Optional[str] = ..., order: _Optional[int] = ...) -> None: ...

class DeleteSessionTabResponse(_message.Message):
    __slots__ = ("status", "profile_id")
    STATUS_FIELD_NUMBER: _ClassVar[int]
    PROFILE_ID_FIELD_NUMBER: _ClassVar[int]
    status: str
    profile_id: str
    def __init__(self, status: _Optional[str] = ..., profile_id: _Optional[str] = ...) -> None: ...
