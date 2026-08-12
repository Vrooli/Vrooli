from browser_automation_studio.v1.base import browser_profile_pb2 as _browser_profile_pb2
from buf.validate import validate_pb2 as _validate_pb2
from google.protobuf.internal import containers as _containers
from google.protobuf.internal import enum_type_wrapper as _enum_type_wrapper
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Iterable as _Iterable, Mapping as _Mapping
from typing import ClassVar as _ClassVar, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class CaptureType(int, metaclass=_enum_type_wrapper.EnumTypeWrapper):
    __slots__ = ()
    CAPTURE_TYPE_UNSPECIFIED: _ClassVar[CaptureType]
    CAPTURE_TYPE_SCREENSHOT: _ClassVar[CaptureType]
    CAPTURE_TYPE_CONSOLE_LOGS: _ClassVar[CaptureType]
    CAPTURE_TYPE_NETWORK: _ClassVar[CaptureType]
    CAPTURE_TYPE_VIDEO: _ClassVar[CaptureType]
    CAPTURE_TYPE_DOM: _ClassVar[CaptureType]
    CAPTURE_TYPE_PERFORMANCE: _ClassVar[CaptureType]
    CAPTURE_TYPE_ACCESSIBILITY: _ClassVar[CaptureType]

class DimensionsPreset(int, metaclass=_enum_type_wrapper.EnumTypeWrapper):
    __slots__ = ()
    DIMENSIONS_PRESET_UNSPECIFIED: _ClassVar[DimensionsPreset]
    DIMENSIONS_PRESET_MOBILE: _ClassVar[DimensionsPreset]
    DIMENSIONS_PRESET_TABLET: _ClassVar[DimensionsPreset]
    DIMENSIONS_PRESET_DESKTOP: _ClassVar[DimensionsPreset]
CAPTURE_TYPE_UNSPECIFIED: CaptureType
CAPTURE_TYPE_SCREENSHOT: CaptureType
CAPTURE_TYPE_CONSOLE_LOGS: CaptureType
CAPTURE_TYPE_NETWORK: CaptureType
CAPTURE_TYPE_VIDEO: CaptureType
CAPTURE_TYPE_DOM: CaptureType
CAPTURE_TYPE_PERFORMANCE: CaptureType
CAPTURE_TYPE_ACCESSIBILITY: CaptureType
DIMENSIONS_PRESET_UNSPECIFIED: DimensionsPreset
DIMENSIONS_PRESET_MOBILE: DimensionsPreset
DIMENSIONS_PRESET_TABLET: DimensionsPreset
DIMENSIONS_PRESET_DESKTOP: DimensionsPreset

class Dimensions(_message.Message):
    __slots__ = ("preset", "width", "height", "device_scale_factor")
    PRESET_FIELD_NUMBER: _ClassVar[int]
    WIDTH_FIELD_NUMBER: _ClassVar[int]
    HEIGHT_FIELD_NUMBER: _ClassVar[int]
    DEVICE_SCALE_FACTOR_FIELD_NUMBER: _ClassVar[int]
    preset: DimensionsPreset
    width: int
    height: int
    device_scale_factor: float
    def __init__(self, preset: _Optional[_Union[DimensionsPreset, str]] = ..., width: _Optional[int] = ..., height: _Optional[int] = ..., device_scale_factor: _Optional[float] = ...) -> None: ...

class WaitFor(_message.Message):
    __slots__ = ("selector", "networkidle", "timeout_ms")
    SELECTOR_FIELD_NUMBER: _ClassVar[int]
    NETWORKIDLE_FIELD_NUMBER: _ClassVar[int]
    TIMEOUT_MS_FIELD_NUMBER: _ClassVar[int]
    selector: str
    networkidle: bool
    timeout_ms: int
    def __init__(self, selector: _Optional[str] = ..., networkidle: _Optional[bool] = ..., timeout_ms: _Optional[int] = ...) -> None: ...

class CaptureRequest(_message.Message):
    __slots__ = ("url", "captures", "dimensions", "wait_for", "out_dir", "label", "inline_dom", "interaction_flow_json", "inline_accessibility", "inline_computed_style", "browser_profile", "interaction_state")
    URL_FIELD_NUMBER: _ClassVar[int]
    CAPTURES_FIELD_NUMBER: _ClassVar[int]
    DIMENSIONS_FIELD_NUMBER: _ClassVar[int]
    WAIT_FOR_FIELD_NUMBER: _ClassVar[int]
    OUT_DIR_FIELD_NUMBER: _ClassVar[int]
    LABEL_FIELD_NUMBER: _ClassVar[int]
    INLINE_DOM_FIELD_NUMBER: _ClassVar[int]
    INTERACTION_FLOW_JSON_FIELD_NUMBER: _ClassVar[int]
    INLINE_ACCESSIBILITY_FIELD_NUMBER: _ClassVar[int]
    INLINE_COMPUTED_STYLE_FIELD_NUMBER: _ClassVar[int]
    BROWSER_PROFILE_FIELD_NUMBER: _ClassVar[int]
    INTERACTION_STATE_FIELD_NUMBER: _ClassVar[int]
    url: str
    captures: _containers.RepeatedScalarFieldContainer[CaptureType]
    dimensions: Dimensions
    wait_for: WaitFor
    out_dir: str
    label: str
    inline_dom: bool
    interaction_flow_json: str
    inline_accessibility: bool
    inline_computed_style: bool
    browser_profile: _browser_profile_pb2.BrowserProfile
    interaction_state: str
    def __init__(self, url: _Optional[str] = ..., captures: _Optional[_Iterable[_Union[CaptureType, str]]] = ..., dimensions: _Optional[_Union[Dimensions, _Mapping]] = ..., wait_for: _Optional[_Union[WaitFor, _Mapping]] = ..., out_dir: _Optional[str] = ..., label: _Optional[str] = ..., inline_dom: _Optional[bool] = ..., interaction_flow_json: _Optional[str] = ..., inline_accessibility: _Optional[bool] = ..., inline_computed_style: _Optional[bool] = ..., browser_profile: _Optional[_Union[_browser_profile_pb2.BrowserProfile, _Mapping]] = ..., interaction_state: _Optional[str] = ...) -> None: ...

class CaptureArtifact(_message.Message):
    __slots__ = ("type", "path", "size_bytes", "metadata", "primary", "reference")
    class MetadataEntry(_message.Message):
        __slots__ = ("key", "value")
        KEY_FIELD_NUMBER: _ClassVar[int]
        VALUE_FIELD_NUMBER: _ClassVar[int]
        key: str
        value: str
        def __init__(self, key: _Optional[str] = ..., value: _Optional[str] = ...) -> None: ...
    TYPE_FIELD_NUMBER: _ClassVar[int]
    PATH_FIELD_NUMBER: _ClassVar[int]
    SIZE_BYTES_FIELD_NUMBER: _ClassVar[int]
    METADATA_FIELD_NUMBER: _ClassVar[int]
    PRIMARY_FIELD_NUMBER: _ClassVar[int]
    REFERENCE_FIELD_NUMBER: _ClassVar[int]
    type: CaptureType
    path: str
    size_bytes: int
    metadata: _containers.ScalarMap[str, str]
    primary: bool
    reference: str
    def __init__(self, type: _Optional[_Union[CaptureType, str]] = ..., path: _Optional[str] = ..., size_bytes: _Optional[int] = ..., metadata: _Optional[_Mapping[str, str]] = ..., primary: _Optional[bool] = ..., reference: _Optional[str] = ...) -> None: ...

class CaptureResponse(_message.Message):
    __slots__ = ("execution_id", "out_dir", "artifacts", "duration_ms", "dry_run", "dom_html", "accessibility_json", "readiness")
    EXECUTION_ID_FIELD_NUMBER: _ClassVar[int]
    OUT_DIR_FIELD_NUMBER: _ClassVar[int]
    ARTIFACTS_FIELD_NUMBER: _ClassVar[int]
    DURATION_MS_FIELD_NUMBER: _ClassVar[int]
    DRY_RUN_FIELD_NUMBER: _ClassVar[int]
    DOM_HTML_FIELD_NUMBER: _ClassVar[int]
    ACCESSIBILITY_JSON_FIELD_NUMBER: _ClassVar[int]
    READINESS_FIELD_NUMBER: _ClassVar[int]
    execution_id: str
    out_dir: str
    artifacts: _containers.RepeatedCompositeFieldContainer[CaptureArtifact]
    duration_ms: int
    dry_run: bool
    dom_html: str
    accessibility_json: str
    readiness: CaptureReadinessDiagnostics
    def __init__(self, execution_id: _Optional[str] = ..., out_dir: _Optional[str] = ..., artifacts: _Optional[_Iterable[_Union[CaptureArtifact, _Mapping]]] = ..., duration_ms: _Optional[int] = ..., dry_run: _Optional[bool] = ..., dom_html: _Optional[str] = ..., accessibility_json: _Optional[str] = ..., readiness: _Optional[_Union[CaptureReadinessDiagnostics, _Mapping]] = ...) -> None: ...

class CaptureReadinessDiagnostics(_message.Message):
    __slots__ = ("requested_strategy", "selected_strategy", "outcome", "duration_ms", "fallback_reason", "profile_version", "route", "required_surface_ids", "navigation_duration_ms", "readiness_wait_duration_ms")
    REQUESTED_STRATEGY_FIELD_NUMBER: _ClassVar[int]
    SELECTED_STRATEGY_FIELD_NUMBER: _ClassVar[int]
    OUTCOME_FIELD_NUMBER: _ClassVar[int]
    DURATION_MS_FIELD_NUMBER: _ClassVar[int]
    FALLBACK_REASON_FIELD_NUMBER: _ClassVar[int]
    PROFILE_VERSION_FIELD_NUMBER: _ClassVar[int]
    ROUTE_FIELD_NUMBER: _ClassVar[int]
    REQUIRED_SURFACE_IDS_FIELD_NUMBER: _ClassVar[int]
    NAVIGATION_DURATION_MS_FIELD_NUMBER: _ClassVar[int]
    READINESS_WAIT_DURATION_MS_FIELD_NUMBER: _ClassVar[int]
    requested_strategy: str
    selected_strategy: str
    outcome: str
    duration_ms: int
    fallback_reason: str
    profile_version: str
    route: str
    required_surface_ids: _containers.RepeatedScalarFieldContainer[str]
    navigation_duration_ms: int
    readiness_wait_duration_ms: int
    def __init__(self, requested_strategy: _Optional[str] = ..., selected_strategy: _Optional[str] = ..., outcome: _Optional[str] = ..., duration_ms: _Optional[int] = ..., fallback_reason: _Optional[str] = ..., profile_version: _Optional[str] = ..., route: _Optional[str] = ..., required_surface_ids: _Optional[_Iterable[str]] = ..., navigation_duration_ms: _Optional[int] = ..., readiness_wait_duration_ms: _Optional[int] = ...) -> None: ...
