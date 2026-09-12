from backdrop_studio.v1.shared import shared_pb2 as _shared_pb2
from google.protobuf.internal import containers as _containers
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Iterable as _Iterable, Mapping as _Mapping
from typing import ClassVar as _ClassVar, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class Brief(_message.Message):
    __slots__ = ("brand_id", "placement", "seed", "prompt")
    BRAND_ID_FIELD_NUMBER: _ClassVar[int]
    PLACEMENT_FIELD_NUMBER: _ClassVar[int]
    SEED_FIELD_NUMBER: _ClassVar[int]
    PROMPT_FIELD_NUMBER: _ClassVar[int]
    brand_id: str
    placement: str
    seed: int
    prompt: str
    def __init__(self, brand_id: _Optional[str] = ..., placement: _Optional[str] = ..., seed: _Optional[int] = ..., prompt: _Optional[str] = ...) -> None: ...

class ResolvePlanRequest(_message.Message):
    __slots__ = ("style", "brief", "brand_tokens", "adapter", "adapter_commercial_use")
    class BrandTokensEntry(_message.Message):
        __slots__ = ("key", "value")
        KEY_FIELD_NUMBER: _ClassVar[int]
        VALUE_FIELD_NUMBER: _ClassVar[int]
        key: str
        value: str
        def __init__(self, key: _Optional[str] = ..., value: _Optional[str] = ...) -> None: ...
    STYLE_FIELD_NUMBER: _ClassVar[int]
    BRIEF_FIELD_NUMBER: _ClassVar[int]
    BRAND_TOKENS_FIELD_NUMBER: _ClassVar[int]
    ADAPTER_FIELD_NUMBER: _ClassVar[int]
    ADAPTER_COMMERCIAL_USE_FIELD_NUMBER: _ClassVar[int]
    style: _shared_pb2.Style
    brief: Brief
    brand_tokens: _containers.ScalarMap[str, str]
    adapter: str
    adapter_commercial_use: bool
    def __init__(self, style: _Optional[_Union[_shared_pb2.Style, _Mapping]] = ..., brief: _Optional[_Union[Brief, _Mapping]] = ..., brand_tokens: _Optional[_Mapping[str, str]] = ..., adapter: _Optional[str] = ..., adapter_commercial_use: _Optional[bool] = ...) -> None: ...

class Operation(_message.Message):
    __slots__ = ("name", "params_json")
    NAME_FIELD_NUMBER: _ClassVar[int]
    PARAMS_JSON_FIELD_NUMBER: _ClassVar[int]
    name: str
    params_json: str
    def __init__(self, name: _Optional[str] = ..., params_json: _Optional[str] = ...) -> None: ...

class ComposeDeviceFrameRequest(_message.Message):
    __slots__ = ("backdrop_png", "screenshot_png", "arrangement", "caption", "width", "height", "surface_id")
    BACKDROP_PNG_FIELD_NUMBER: _ClassVar[int]
    SCREENSHOT_PNG_FIELD_NUMBER: _ClassVar[int]
    ARRANGEMENT_FIELD_NUMBER: _ClassVar[int]
    CAPTION_FIELD_NUMBER: _ClassVar[int]
    WIDTH_FIELD_NUMBER: _ClassVar[int]
    HEIGHT_FIELD_NUMBER: _ClassVar[int]
    SURFACE_ID_FIELD_NUMBER: _ClassVar[int]
    backdrop_png: bytes
    screenshot_png: bytes
    arrangement: str
    caption: str
    width: int
    height: int
    surface_id: str
    def __init__(self, backdrop_png: _Optional[bytes] = ..., screenshot_png: _Optional[bytes] = ..., arrangement: _Optional[str] = ..., caption: _Optional[str] = ..., width: _Optional[int] = ..., height: _Optional[int] = ..., surface_id: _Optional[str] = ...) -> None: ...

class ComposeDeviceFrameResponse(_message.Message):
    __slots__ = ("image_png", "width", "height", "occlusion_region")
    IMAGE_PNG_FIELD_NUMBER: _ClassVar[int]
    WIDTH_FIELD_NUMBER: _ClassVar[int]
    HEIGHT_FIELD_NUMBER: _ClassVar[int]
    OCCLUSION_REGION_FIELD_NUMBER: _ClassVar[int]
    image_png: bytes
    width: int
    height: int
    occlusion_region: _shared_pb2.ReservedRegion
    def __init__(self, image_png: _Optional[bytes] = ..., width: _Optional[int] = ..., height: _Optional[int] = ..., occlusion_region: _Optional[_Union[_shared_pb2.ReservedRegion, _Mapping]] = ...) -> None: ...

class ResolvedPlan(_message.Message):
    __slots__ = ("style_id", "strategy", "operations", "resolved_slots", "expected_execution_path", "executable")
    class ResolvedSlotsEntry(_message.Message):
        __slots__ = ("key", "value")
        KEY_FIELD_NUMBER: _ClassVar[int]
        VALUE_FIELD_NUMBER: _ClassVar[int]
        key: str
        value: str
        def __init__(self, key: _Optional[str] = ..., value: _Optional[str] = ...) -> None: ...
    STYLE_ID_FIELD_NUMBER: _ClassVar[int]
    STRATEGY_FIELD_NUMBER: _ClassVar[int]
    OPERATIONS_FIELD_NUMBER: _ClassVar[int]
    RESOLVED_SLOTS_FIELD_NUMBER: _ClassVar[int]
    EXPECTED_EXECUTION_PATH_FIELD_NUMBER: _ClassVar[int]
    EXECUTABLE_FIELD_NUMBER: _ClassVar[int]
    style_id: str
    strategy: str
    operations: _containers.RepeatedCompositeFieldContainer[Operation]
    resolved_slots: _containers.ScalarMap[str, str]
    expected_execution_path: str
    executable: bool
    def __init__(self, style_id: _Optional[str] = ..., strategy: _Optional[str] = ..., operations: _Optional[_Iterable[_Union[Operation, _Mapping]]] = ..., resolved_slots: _Optional[_Mapping[str, str]] = ..., expected_execution_path: _Optional[str] = ..., executable: _Optional[bool] = ...) -> None: ...
