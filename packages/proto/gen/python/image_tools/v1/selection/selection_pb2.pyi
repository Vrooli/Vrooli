from google.protobuf.internal import containers as _containers
from google.protobuf.internal import enum_type_wrapper as _enum_type_wrapper
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Iterable as _Iterable, Mapping as _Mapping
from typing import ClassVar as _ClassVar, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class SegmentMode(int, metaclass=_enum_type_wrapper.EnumTypeWrapper):
    __slots__ = ()
    SEGMENT_MODE_UNSPECIFIED: _ClassVar[SegmentMode]
    SEGMENT_MODE_POINT: _ClassVar[SegmentMode]
    SEGMENT_MODE_BOX: _ClassVar[SegmentMode]
    SEGMENT_MODE_AUTO: _ClassVar[SegmentMode]
SEGMENT_MODE_UNSPECIFIED: SegmentMode
SEGMENT_MODE_POINT: SegmentMode
SEGMENT_MODE_BOX: SegmentMode
SEGMENT_MODE_AUTO: SegmentMode

class Point(_message.Message):
    __slots__ = ("x", "y", "negative")
    X_FIELD_NUMBER: _ClassVar[int]
    Y_FIELD_NUMBER: _ClassVar[int]
    NEGATIVE_FIELD_NUMBER: _ClassVar[int]
    x: float
    y: float
    negative: bool
    def __init__(self, x: _Optional[float] = ..., y: _Optional[float] = ..., negative: _Optional[bool] = ...) -> None: ...

class Box(_message.Message):
    __slots__ = ("x", "y", "width", "height")
    X_FIELD_NUMBER: _ClassVar[int]
    Y_FIELD_NUMBER: _ClassVar[int]
    WIDTH_FIELD_NUMBER: _ClassVar[int]
    HEIGHT_FIELD_NUMBER: _ClassVar[int]
    x: float
    y: float
    width: float
    height: float
    def __init__(self, x: _Optional[float] = ..., y: _Optional[float] = ..., width: _Optional[float] = ..., height: _Optional[float] = ...) -> None: ...

class SegmentParams(_message.Message):
    __slots__ = ("mode", "points", "box", "tolerance", "model_override")
    MODE_FIELD_NUMBER: _ClassVar[int]
    POINTS_FIELD_NUMBER: _ClassVar[int]
    BOX_FIELD_NUMBER: _ClassVar[int]
    TOLERANCE_FIELD_NUMBER: _ClassVar[int]
    MODEL_OVERRIDE_FIELD_NUMBER: _ClassVar[int]
    mode: SegmentMode
    points: _containers.RepeatedCompositeFieldContainer[Point]
    box: Box
    tolerance: float
    model_override: str
    def __init__(self, mode: _Optional[_Union[SegmentMode, str]] = ..., points: _Optional[_Iterable[_Union[Point, _Mapping]]] = ..., box: _Optional[_Union[Box, _Mapping]] = ..., tolerance: _Optional[float] = ..., model_override: _Optional[str] = ...) -> None: ...

class SuggestedEdit(_message.Message):
    __slots__ = ("id", "label", "description", "operation", "prompt", "requires_prompt", "requires_mask", "params")
    class ParamsEntry(_message.Message):
        __slots__ = ("key", "value")
        KEY_FIELD_NUMBER: _ClassVar[int]
        VALUE_FIELD_NUMBER: _ClassVar[int]
        key: str
        value: str
        def __init__(self, key: _Optional[str] = ..., value: _Optional[str] = ...) -> None: ...
    ID_FIELD_NUMBER: _ClassVar[int]
    LABEL_FIELD_NUMBER: _ClassVar[int]
    DESCRIPTION_FIELD_NUMBER: _ClassVar[int]
    OPERATION_FIELD_NUMBER: _ClassVar[int]
    PROMPT_FIELD_NUMBER: _ClassVar[int]
    REQUIRES_PROMPT_FIELD_NUMBER: _ClassVar[int]
    REQUIRES_MASK_FIELD_NUMBER: _ClassVar[int]
    PARAMS_FIELD_NUMBER: _ClassVar[int]
    id: str
    label: str
    description: str
    operation: str
    prompt: str
    requires_prompt: bool
    requires_mask: bool
    params: _containers.ScalarMap[str, str]
    def __init__(self, id: _Optional[str] = ..., label: _Optional[str] = ..., description: _Optional[str] = ..., operation: _Optional[str] = ..., prompt: _Optional[str] = ..., requires_prompt: _Optional[bool] = ..., requires_mask: _Optional[bool] = ..., params: _Optional[_Mapping[str, str]] = ...) -> None: ...

class RegionClassInfo(_message.Message):
    __slots__ = ("name", "summary", "edits")
    NAME_FIELD_NUMBER: _ClassVar[int]
    SUMMARY_FIELD_NUMBER: _ClassVar[int]
    EDITS_FIELD_NUMBER: _ClassVar[int]
    name: str
    summary: str
    edits: _containers.RepeatedCompositeFieldContainer[SuggestedEdit]
    def __init__(self, name: _Optional[str] = ..., summary: _Optional[str] = ..., edits: _Optional[_Iterable[_Union[SuggestedEdit, _Mapping]]] = ...) -> None: ...

class ListRegionClassesRequest(_message.Message):
    __slots__ = ()
    def __init__(self) -> None: ...

class ListRegionClassesResponse(_message.Message):
    __slots__ = ("classes",)
    CLASSES_FIELD_NUMBER: _ClassVar[int]
    classes: _containers.RepeatedCompositeFieldContainer[RegionClassInfo]
    def __init__(self, classes: _Optional[_Iterable[_Union[RegionClassInfo, _Mapping]]] = ...) -> None: ...

class SuggestEditsRequest(_message.Message):
    __slots__ = ("region_class",)
    REGION_CLASS_FIELD_NUMBER: _ClassVar[int]
    region_class: str
    def __init__(self, region_class: _Optional[str] = ...) -> None: ...

class SuggestEditsResponse(_message.Message):
    __slots__ = ("region_class", "edits")
    REGION_CLASS_FIELD_NUMBER: _ClassVar[int]
    EDITS_FIELD_NUMBER: _ClassVar[int]
    region_class: str
    edits: _containers.RepeatedCompositeFieldContainer[SuggestedEdit]
    def __init__(self, region_class: _Optional[str] = ..., edits: _Optional[_Iterable[_Union[SuggestedEdit, _Mapping]]] = ...) -> None: ...

class SegmentResult(_message.Message):
    __slots__ = ("job_id", "mask_ref", "box", "region_class", "confidence", "area_fraction", "tier", "model_id", "suggested_edits", "warnings")
    JOB_ID_FIELD_NUMBER: _ClassVar[int]
    MASK_REF_FIELD_NUMBER: _ClassVar[int]
    BOX_FIELD_NUMBER: _ClassVar[int]
    REGION_CLASS_FIELD_NUMBER: _ClassVar[int]
    CONFIDENCE_FIELD_NUMBER: _ClassVar[int]
    AREA_FRACTION_FIELD_NUMBER: _ClassVar[int]
    TIER_FIELD_NUMBER: _ClassVar[int]
    MODEL_ID_FIELD_NUMBER: _ClassVar[int]
    SUGGESTED_EDITS_FIELD_NUMBER: _ClassVar[int]
    WARNINGS_FIELD_NUMBER: _ClassVar[int]
    job_id: str
    mask_ref: str
    box: Box
    region_class: str
    confidence: float
    area_fraction: float
    tier: str
    model_id: str
    suggested_edits: _containers.RepeatedCompositeFieldContainer[SuggestedEdit]
    warnings: _containers.RepeatedScalarFieldContainer[str]
    def __init__(self, job_id: _Optional[str] = ..., mask_ref: _Optional[str] = ..., box: _Optional[_Union[Box, _Mapping]] = ..., region_class: _Optional[str] = ..., confidence: _Optional[float] = ..., area_fraction: _Optional[float] = ..., tier: _Optional[str] = ..., model_id: _Optional[str] = ..., suggested_edits: _Optional[_Iterable[_Union[SuggestedEdit, _Mapping]]] = ..., warnings: _Optional[_Iterable[str]] = ...) -> None: ...
