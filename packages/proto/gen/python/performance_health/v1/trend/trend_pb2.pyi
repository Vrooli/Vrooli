from google.protobuf.internal import containers as _containers
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Iterable as _Iterable, Mapping as _Mapping
from typing import ClassVar as _ClassVar, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class GetTrendRequest(_message.Message):
    __slots__ = ("scenario", "limit")
    SCENARIO_FIELD_NUMBER: _ClassVar[int]
    LIMIT_FIELD_NUMBER: _ClassVar[int]
    scenario: str
    limit: int
    def __init__(self, scenario: _Optional[str] = ..., limit: _Optional[int] = ...) -> None: ...

class GetTrendResponse(_message.Message):
    __slots__ = ("scenario", "samples")
    SCENARIO_FIELD_NUMBER: _ClassVar[int]
    SAMPLES_FIELD_NUMBER: _ClassVar[int]
    scenario: str
    samples: _containers.RepeatedCompositeFieldContainer[TrendSample]
    def __init__(self, scenario: _Optional[str] = ..., samples: _Optional[_Iterable[_Union[TrendSample, _Mapping]]] = ...) -> None: ...

class TrendSample(_message.Message):
    __slots__ = ("scenario", "captured_at", "go_build_ms", "ui_build_ms", "bundle_bytes", "lcp_ms", "cls", "response_end_ms", "dom_interactive_ms", "dom_content_loaded_ms", "load_event_end_ms", "navigation_type", "startup_ms", "note", "slowest_component", "slowest_component_avg_ms", "slowest_component_max_ms")
    SCENARIO_FIELD_NUMBER: _ClassVar[int]
    CAPTURED_AT_FIELD_NUMBER: _ClassVar[int]
    GO_BUILD_MS_FIELD_NUMBER: _ClassVar[int]
    UI_BUILD_MS_FIELD_NUMBER: _ClassVar[int]
    BUNDLE_BYTES_FIELD_NUMBER: _ClassVar[int]
    LCP_MS_FIELD_NUMBER: _ClassVar[int]
    CLS_FIELD_NUMBER: _ClassVar[int]
    RESPONSE_END_MS_FIELD_NUMBER: _ClassVar[int]
    DOM_INTERACTIVE_MS_FIELD_NUMBER: _ClassVar[int]
    DOM_CONTENT_LOADED_MS_FIELD_NUMBER: _ClassVar[int]
    LOAD_EVENT_END_MS_FIELD_NUMBER: _ClassVar[int]
    NAVIGATION_TYPE_FIELD_NUMBER: _ClassVar[int]
    STARTUP_MS_FIELD_NUMBER: _ClassVar[int]
    NOTE_FIELD_NUMBER: _ClassVar[int]
    SLOWEST_COMPONENT_FIELD_NUMBER: _ClassVar[int]
    SLOWEST_COMPONENT_AVG_MS_FIELD_NUMBER: _ClassVar[int]
    SLOWEST_COMPONENT_MAX_MS_FIELD_NUMBER: _ClassVar[int]
    scenario: str
    captured_at: str
    go_build_ms: int
    ui_build_ms: int
    bundle_bytes: int
    lcp_ms: int
    cls: float
    response_end_ms: int
    dom_interactive_ms: int
    dom_content_loaded_ms: int
    load_event_end_ms: int
    navigation_type: str
    startup_ms: int
    note: str
    slowest_component: str
    slowest_component_avg_ms: float
    slowest_component_max_ms: float
    def __init__(self, scenario: _Optional[str] = ..., captured_at: _Optional[str] = ..., go_build_ms: _Optional[int] = ..., ui_build_ms: _Optional[int] = ..., bundle_bytes: _Optional[int] = ..., lcp_ms: _Optional[int] = ..., cls: _Optional[float] = ..., response_end_ms: _Optional[int] = ..., dom_interactive_ms: _Optional[int] = ..., dom_content_loaded_ms: _Optional[int] = ..., load_event_end_ms: _Optional[int] = ..., navigation_type: _Optional[str] = ..., startup_ms: _Optional[int] = ..., note: _Optional[str] = ..., slowest_component: _Optional[str] = ..., slowest_component_avg_ms: _Optional[float] = ..., slowest_component_max_ms: _Optional[float] = ...) -> None: ...
