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
    __slots__ = ("url", "captures", "dimensions", "wait_for", "out_dir", "label", "inline_dom")
    URL_FIELD_NUMBER: _ClassVar[int]
    CAPTURES_FIELD_NUMBER: _ClassVar[int]
    DIMENSIONS_FIELD_NUMBER: _ClassVar[int]
    WAIT_FOR_FIELD_NUMBER: _ClassVar[int]
    OUT_DIR_FIELD_NUMBER: _ClassVar[int]
    LABEL_FIELD_NUMBER: _ClassVar[int]
    INLINE_DOM_FIELD_NUMBER: _ClassVar[int]
    url: str
    captures: _containers.RepeatedScalarFieldContainer[CaptureType]
    dimensions: Dimensions
    wait_for: WaitFor
    out_dir: str
    label: str
    inline_dom: bool
    def __init__(self, url: _Optional[str] = ..., captures: _Optional[_Iterable[_Union[CaptureType, str]]] = ..., dimensions: _Optional[_Union[Dimensions, _Mapping]] = ..., wait_for: _Optional[_Union[WaitFor, _Mapping]] = ..., out_dir: _Optional[str] = ..., label: _Optional[str] = ..., inline_dom: _Optional[bool] = ...) -> None: ...

class CaptureArtifact(_message.Message):
    __slots__ = ("type", "path", "size_bytes", "metadata")
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
    type: CaptureType
    path: str
    size_bytes: int
    metadata: _containers.ScalarMap[str, str]
    def __init__(self, type: _Optional[_Union[CaptureType, str]] = ..., path: _Optional[str] = ..., size_bytes: _Optional[int] = ..., metadata: _Optional[_Mapping[str, str]] = ...) -> None: ...

class CaptureResponse(_message.Message):
    __slots__ = ("execution_id", "out_dir", "artifacts", "duration_ms", "dry_run", "dom_html")
    EXECUTION_ID_FIELD_NUMBER: _ClassVar[int]
    OUT_DIR_FIELD_NUMBER: _ClassVar[int]
    ARTIFACTS_FIELD_NUMBER: _ClassVar[int]
    DURATION_MS_FIELD_NUMBER: _ClassVar[int]
    DRY_RUN_FIELD_NUMBER: _ClassVar[int]
    DOM_HTML_FIELD_NUMBER: _ClassVar[int]
    execution_id: str
    out_dir: str
    artifacts: _containers.RepeatedCompositeFieldContainer[CaptureArtifact]
    duration_ms: int
    dry_run: bool
    dom_html: str
    def __init__(self, execution_id: _Optional[str] = ..., out_dir: _Optional[str] = ..., artifacts: _Optional[_Iterable[_Union[CaptureArtifact, _Mapping]]] = ..., duration_ms: _Optional[int] = ..., dry_run: _Optional[bool] = ..., dom_html: _Optional[str] = ...) -> None: ...
