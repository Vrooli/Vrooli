from google.protobuf.internal import containers as _containers
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Iterable as _Iterable, Mapping as _Mapping
from typing import ClassVar as _ClassVar, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class BrowserProfile(_message.Message):
    __slots__ = ()
    class ExtraHeadersEntry(_message.Message):
        __slots__ = ()
        KEY_FIELD_NUMBER: _ClassVar[int]
        VALUE_FIELD_NUMBER: _ClassVar[int]
        key: str
        value: str
        def __init__(self, key: _Optional[str] = ..., value: _Optional[str] = ...) -> None: ...
    PRESET_FIELD_NUMBER: _ClassVar[int]
    FINGERPRINT_FIELD_NUMBER: _ClassVar[int]
    BEHAVIOR_FIELD_NUMBER: _ClassVar[int]
    ANTI_DETECTION_FIELD_NUMBER: _ClassVar[int]
    PROXY_FIELD_NUMBER: _ClassVar[int]
    EXTRA_HEADERS_FIELD_NUMBER: _ClassVar[int]
    preset: str
    fingerprint: FingerprintSettings
    behavior: BehaviorSettings
    anti_detection: AntiDetectionSettings
    proxy: ProxySettings
    extra_headers: _containers.ScalarMap[str, str]
    def __init__(self, preset: _Optional[str] = ..., fingerprint: _Optional[_Union[FingerprintSettings, _Mapping]] = ..., behavior: _Optional[_Union[BehaviorSettings, _Mapping]] = ..., anti_detection: _Optional[_Union[AntiDetectionSettings, _Mapping]] = ..., proxy: _Optional[_Union[ProxySettings, _Mapping]] = ..., extra_headers: _Optional[_Mapping[str, str]] = ...) -> None: ...

class FingerprintSettings(_message.Message):
    __slots__ = ()
    VIEWPORT_WIDTH_FIELD_NUMBER: _ClassVar[int]
    VIEWPORT_HEIGHT_FIELD_NUMBER: _ClassVar[int]
    DEVICE_SCALE_FACTOR_FIELD_NUMBER: _ClassVar[int]
    HARDWARE_CONCURRENCY_FIELD_NUMBER: _ClassVar[int]
    DEVICE_MEMORY_FIELD_NUMBER: _ClassVar[int]
    USER_AGENT_FIELD_NUMBER: _ClassVar[int]
    USER_AGENT_PRESET_FIELD_NUMBER: _ClassVar[int]
    LOCALE_FIELD_NUMBER: _ClassVar[int]
    TIMEZONE_ID_FIELD_NUMBER: _ClassVar[int]
    GEOLOCATION_ENABLED_FIELD_NUMBER: _ClassVar[int]
    LATITUDE_FIELD_NUMBER: _ClassVar[int]
    LONGITUDE_FIELD_NUMBER: _ClassVar[int]
    ACCURACY_FIELD_NUMBER: _ClassVar[int]
    COLOR_SCHEME_FIELD_NUMBER: _ClassVar[int]
    viewport_width: int
    viewport_height: int
    device_scale_factor: float
    hardware_concurrency: int
    device_memory: int
    user_agent: str
    user_agent_preset: str
    locale: str
    timezone_id: str
    geolocation_enabled: bool
    latitude: float
    longitude: float
    accuracy: float
    color_scheme: str
    def __init__(self, viewport_width: _Optional[int] = ..., viewport_height: _Optional[int] = ..., device_scale_factor: _Optional[float] = ..., hardware_concurrency: _Optional[int] = ..., device_memory: _Optional[int] = ..., user_agent: _Optional[str] = ..., user_agent_preset: _Optional[str] = ..., locale: _Optional[str] = ..., timezone_id: _Optional[str] = ..., geolocation_enabled: _Optional[bool] = ..., latitude: _Optional[float] = ..., longitude: _Optional[float] = ..., accuracy: _Optional[float] = ..., color_scheme: _Optional[str] = ...) -> None: ...

class BehaviorSettings(_message.Message):
    __slots__ = ()
    TYPING_DELAY_MIN_FIELD_NUMBER: _ClassVar[int]
    TYPING_DELAY_MAX_FIELD_NUMBER: _ClassVar[int]
    TYPING_START_DELAY_MIN_FIELD_NUMBER: _ClassVar[int]
    TYPING_START_DELAY_MAX_FIELD_NUMBER: _ClassVar[int]
    TYPING_PASTE_THRESHOLD_FIELD_NUMBER: _ClassVar[int]
    TYPING_VARIANCE_ENABLED_FIELD_NUMBER: _ClassVar[int]
    MOUSE_MOVEMENT_STYLE_FIELD_NUMBER: _ClassVar[int]
    MOUSE_JITTER_AMOUNT_FIELD_NUMBER: _ClassVar[int]
    CLICK_DELAY_MIN_FIELD_NUMBER: _ClassVar[int]
    CLICK_DELAY_MAX_FIELD_NUMBER: _ClassVar[int]
    SCROLL_STYLE_FIELD_NUMBER: _ClassVar[int]
    SCROLL_SPEED_MIN_FIELD_NUMBER: _ClassVar[int]
    SCROLL_SPEED_MAX_FIELD_NUMBER: _ClassVar[int]
    MICRO_PAUSE_ENABLED_FIELD_NUMBER: _ClassVar[int]
    MICRO_PAUSE_MIN_MS_FIELD_NUMBER: _ClassVar[int]
    MICRO_PAUSE_MAX_MS_FIELD_NUMBER: _ClassVar[int]
    MICRO_PAUSE_FREQUENCY_FIELD_NUMBER: _ClassVar[int]
    typing_delay_min: int
    typing_delay_max: int
    typing_start_delay_min: int
    typing_start_delay_max: int
    typing_paste_threshold: int
    typing_variance_enabled: bool
    mouse_movement_style: str
    mouse_jitter_amount: float
    click_delay_min: int
    click_delay_max: int
    scroll_style: str
    scroll_speed_min: int
    scroll_speed_max: int
    micro_pause_enabled: bool
    micro_pause_min_ms: int
    micro_pause_max_ms: int
    micro_pause_frequency: float
    def __init__(self, typing_delay_min: _Optional[int] = ..., typing_delay_max: _Optional[int] = ..., typing_start_delay_min: _Optional[int] = ..., typing_start_delay_max: _Optional[int] = ..., typing_paste_threshold: _Optional[int] = ..., typing_variance_enabled: _Optional[bool] = ..., mouse_movement_style: _Optional[str] = ..., mouse_jitter_amount: _Optional[float] = ..., click_delay_min: _Optional[int] = ..., click_delay_max: _Optional[int] = ..., scroll_style: _Optional[str] = ..., scroll_speed_min: _Optional[int] = ..., scroll_speed_max: _Optional[int] = ..., micro_pause_enabled: _Optional[bool] = ..., micro_pause_min_ms: _Optional[int] = ..., micro_pause_max_ms: _Optional[int] = ..., micro_pause_frequency: _Optional[float] = ...) -> None: ...

class AntiDetectionSettings(_message.Message):
    __slots__ = ()
    DISABLE_AUTOMATION_CONTROLLED_FIELD_NUMBER: _ClassVar[int]
    DISABLE_WEBRTC_FIELD_NUMBER: _ClassVar[int]
    PATCH_NAVIGATOR_WEBDRIVER_FIELD_NUMBER: _ClassVar[int]
    PATCH_NAVIGATOR_PLUGINS_FIELD_NUMBER: _ClassVar[int]
    PATCH_NAVIGATOR_LANGUAGES_FIELD_NUMBER: _ClassVar[int]
    PATCH_WEBGL_FIELD_NUMBER: _ClassVar[int]
    PATCH_CANVAS_FIELD_NUMBER: _ClassVar[int]
    PATCH_AUDIO_CONTEXT_FIELD_NUMBER: _ClassVar[int]
    HEADLESS_DETECTION_BYPASS_FIELD_NUMBER: _ClassVar[int]
    PATCH_FONTS_FIELD_NUMBER: _ClassVar[int]
    PATCH_SCREEN_PROPERTIES_FIELD_NUMBER: _ClassVar[int]
    PATCH_BATTERY_API_FIELD_NUMBER: _ClassVar[int]
    PATCH_CONNECTION_API_FIELD_NUMBER: _ClassVar[int]
    AD_BLOCKING_MODE_FIELD_NUMBER: _ClassVar[int]
    AD_BLOCKING_WHITELIST_FIELD_NUMBER: _ClassVar[int]
    disable_automation_controlled: bool
    disable_webrtc: bool
    patch_navigator_webdriver: bool
    patch_navigator_plugins: bool
    patch_navigator_languages: bool
    patch_webgl: bool
    patch_canvas: bool
    patch_audio_context: bool
    headless_detection_bypass: bool
    patch_fonts: bool
    patch_screen_properties: bool
    patch_battery_api: bool
    patch_connection_api: bool
    ad_blocking_mode: str
    ad_blocking_whitelist: _containers.RepeatedScalarFieldContainer[str]
    def __init__(self, disable_automation_controlled: _Optional[bool] = ..., disable_webrtc: _Optional[bool] = ..., patch_navigator_webdriver: _Optional[bool] = ..., patch_navigator_plugins: _Optional[bool] = ..., patch_navigator_languages: _Optional[bool] = ..., patch_webgl: _Optional[bool] = ..., patch_canvas: _Optional[bool] = ..., patch_audio_context: _Optional[bool] = ..., headless_detection_bypass: _Optional[bool] = ..., patch_fonts: _Optional[bool] = ..., patch_screen_properties: _Optional[bool] = ..., patch_battery_api: _Optional[bool] = ..., patch_connection_api: _Optional[bool] = ..., ad_blocking_mode: _Optional[str] = ..., ad_blocking_whitelist: _Optional[_Iterable[str]] = ...) -> None: ...

class ProxySettings(_message.Message):
    __slots__ = ()
    ENABLED_FIELD_NUMBER: _ClassVar[int]
    SERVER_FIELD_NUMBER: _ClassVar[int]
    BYPASS_FIELD_NUMBER: _ClassVar[int]
    USERNAME_FIELD_NUMBER: _ClassVar[int]
    PASSWORD_FIELD_NUMBER: _ClassVar[int]
    enabled: bool
    server: str
    bypass: str
    username: str
    password: str
    def __init__(self, enabled: _Optional[bool] = ..., server: _Optional[str] = ..., bypass: _Optional[str] = ..., username: _Optional[str] = ..., password: _Optional[str] = ...) -> None: ...
