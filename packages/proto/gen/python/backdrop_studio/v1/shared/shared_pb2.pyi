from google.protobuf.internal import containers as _containers
from google.protobuf.internal import enum_type_wrapper as _enum_type_wrapper
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Iterable as _Iterable, Mapping as _Mapping
from typing import ClassVar as _ClassVar, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class QualityTier(int, metaclass=_enum_type_wrapper.EnumTypeWrapper):
    __slots__ = ()
    QUALITY_TIER_UNSPECIFIED: _ClassVar[QualityTier]
    QUALITY_TIER_PROCEDURAL: _ClassVar[QualityTier]
    QUALITY_TIER_LOCAL_MODEL: _ClassVar[QualityTier]
    QUALITY_TIER_FRONTIER_MODEL: _ClassVar[QualityTier]
QUALITY_TIER_UNSPECIFIED: QualityTier
QUALITY_TIER_PROCEDURAL: QualityTier
QUALITY_TIER_LOCAL_MODEL: QualityTier
QUALITY_TIER_FRONTIER_MODEL: QualityTier

class ReservedRegion(_message.Message):
    __slots__ = ("x", "y", "width", "height", "kind", "text_color")
    X_FIELD_NUMBER: _ClassVar[int]
    Y_FIELD_NUMBER: _ClassVar[int]
    WIDTH_FIELD_NUMBER: _ClassVar[int]
    HEIGHT_FIELD_NUMBER: _ClassVar[int]
    KIND_FIELD_NUMBER: _ClassVar[int]
    TEXT_COLOR_FIELD_NUMBER: _ClassVar[int]
    x: float
    y: float
    width: float
    height: float
    kind: str
    text_color: str
    def __init__(self, x: _Optional[float] = ..., y: _Optional[float] = ..., width: _Optional[float] = ..., height: _Optional[float] = ..., kind: _Optional[str] = ..., text_color: _Optional[str] = ...) -> None: ...

class ScaffoldBinding(_message.Message):
    __slots__ = ("preset", "conditioner", "params_json")
    PRESET_FIELD_NUMBER: _ClassVar[int]
    CONDITIONER_FIELD_NUMBER: _ClassVar[int]
    PARAMS_JSON_FIELD_NUMBER: _ClassVar[int]
    preset: str
    conditioner: str
    params_json: str
    def __init__(self, preset: _Optional[str] = ..., conditioner: _Optional[str] = ..., params_json: _Optional[str] = ...) -> None: ...

class GenerationBlock(_message.Message):
    __slots__ = ("prompt_template", "negative", "model", "provider_url", "credential")
    PROMPT_TEMPLATE_FIELD_NUMBER: _ClassVar[int]
    NEGATIVE_FIELD_NUMBER: _ClassVar[int]
    MODEL_FIELD_NUMBER: _ClassVar[int]
    PROVIDER_URL_FIELD_NUMBER: _ClassVar[int]
    CREDENTIAL_FIELD_NUMBER: _ClassVar[int]
    prompt_template: str
    negative: str
    model: str
    provider_url: str
    credential: str
    def __init__(self, prompt_template: _Optional[str] = ..., negative: _Optional[str] = ..., model: _Optional[str] = ..., provider_url: _Optional[str] = ..., credential: _Optional[str] = ...) -> None: ...

class RoutingRecord(_message.Message):
    __slots__ = ("declared_tier", "served_lane", "model_id", "execution_tier", "cost_usd", "attempted_lanes", "attempt_details")
    DECLARED_TIER_FIELD_NUMBER: _ClassVar[int]
    SERVED_LANE_FIELD_NUMBER: _ClassVar[int]
    MODEL_ID_FIELD_NUMBER: _ClassVar[int]
    EXECUTION_TIER_FIELD_NUMBER: _ClassVar[int]
    COST_USD_FIELD_NUMBER: _ClassVar[int]
    ATTEMPTED_LANES_FIELD_NUMBER: _ClassVar[int]
    ATTEMPT_DETAILS_FIELD_NUMBER: _ClassVar[int]
    declared_tier: QualityTier
    served_lane: str
    model_id: str
    execution_tier: str
    cost_usd: float
    attempted_lanes: _containers.RepeatedScalarFieldContainer[str]
    attempt_details: _containers.RepeatedScalarFieldContainer[str]
    def __init__(self, declared_tier: _Optional[_Union[QualityTier, str]] = ..., served_lane: _Optional[str] = ..., model_id: _Optional[str] = ..., execution_tier: _Optional[str] = ..., cost_usd: _Optional[float] = ..., attempted_lanes: _Optional[_Iterable[str]] = ..., attempt_details: _Optional[_Iterable[str]] = ...) -> None: ...

class Plate(_message.Message):
    __slots__ = ("name", "depth", "blend", "opacity", "ref", "treatments", "motion")
    NAME_FIELD_NUMBER: _ClassVar[int]
    DEPTH_FIELD_NUMBER: _ClassVar[int]
    BLEND_FIELD_NUMBER: _ClassVar[int]
    OPACITY_FIELD_NUMBER: _ClassVar[int]
    REF_FIELD_NUMBER: _ClassVar[int]
    TREATMENTS_FIELD_NUMBER: _ClassVar[int]
    MOTION_FIELD_NUMBER: _ClassVar[int]
    name: str
    depth: int
    blend: str
    opacity: float
    ref: str
    treatments: _containers.RepeatedScalarFieldContainer[str]
    motion: MotionProfile
    def __init__(self, name: _Optional[str] = ..., depth: _Optional[int] = ..., blend: _Optional[str] = ..., opacity: _Optional[float] = ..., ref: _Optional[str] = ..., treatments: _Optional[_Iterable[str]] = ..., motion: _Optional[_Union[MotionProfile, _Mapping]] = ...) -> None: ...

class MotionProfile(_message.Message):
    __slots__ = ("parallax", "ambient", "ambient_seconds", "ambient_amplitude")
    PARALLAX_FIELD_NUMBER: _ClassVar[int]
    AMBIENT_FIELD_NUMBER: _ClassVar[int]
    AMBIENT_SECONDS_FIELD_NUMBER: _ClassVar[int]
    AMBIENT_AMPLITUDE_FIELD_NUMBER: _ClassVar[int]
    parallax: float
    ambient: str
    ambient_seconds: float
    ambient_amplitude: float
    def __init__(self, parallax: _Optional[float] = ..., ambient: _Optional[str] = ..., ambient_seconds: _Optional[float] = ..., ambient_amplitude: _Optional[float] = ...) -> None: ...

class PlateSpec(_message.Message):
    __slots__ = ("name", "depth", "blend", "opacity", "treatments", "motion", "planes", "treatment_params")
    class TreatmentParamsEntry(_message.Message):
        __slots__ = ("key", "value")
        KEY_FIELD_NUMBER: _ClassVar[int]
        VALUE_FIELD_NUMBER: _ClassVar[int]
        key: str
        value: str
        def __init__(self, key: _Optional[str] = ..., value: _Optional[str] = ...) -> None: ...
    NAME_FIELD_NUMBER: _ClassVar[int]
    DEPTH_FIELD_NUMBER: _ClassVar[int]
    BLEND_FIELD_NUMBER: _ClassVar[int]
    OPACITY_FIELD_NUMBER: _ClassVar[int]
    TREATMENTS_FIELD_NUMBER: _ClassVar[int]
    MOTION_FIELD_NUMBER: _ClassVar[int]
    PLANES_FIELD_NUMBER: _ClassVar[int]
    TREATMENT_PARAMS_FIELD_NUMBER: _ClassVar[int]
    name: str
    depth: int
    blend: str
    opacity: float
    treatments: _containers.RepeatedScalarFieldContainer[str]
    motion: MotionProfile
    planes: _containers.RepeatedScalarFieldContainer[str]
    treatment_params: _containers.ScalarMap[str, str]
    def __init__(self, name: _Optional[str] = ..., depth: _Optional[int] = ..., blend: _Optional[str] = ..., opacity: _Optional[float] = ..., treatments: _Optional[_Iterable[str]] = ..., motion: _Optional[_Union[MotionProfile, _Mapping]] = ..., planes: _Optional[_Iterable[str]] = ..., treatment_params: _Optional[_Mapping[str, str]] = ...) -> None: ...

class Style(_message.Message):
    __slots__ = ("id", "name", "version", "role", "subject", "treatments", "lineage", "placements", "strategy", "regions", "contrast_threshold", "scaffold", "generation", "treatment_params", "inks", "parent_id", "quality_tier", "plate_spec")
    class TreatmentParamsEntry(_message.Message):
        __slots__ = ("key", "value")
        KEY_FIELD_NUMBER: _ClassVar[int]
        VALUE_FIELD_NUMBER: _ClassVar[int]
        key: str
        value: str
        def __init__(self, key: _Optional[str] = ..., value: _Optional[str] = ...) -> None: ...
    class InksEntry(_message.Message):
        __slots__ = ("key", "value")
        KEY_FIELD_NUMBER: _ClassVar[int]
        VALUE_FIELD_NUMBER: _ClassVar[int]
        key: str
        value: str
        def __init__(self, key: _Optional[str] = ..., value: _Optional[str] = ...) -> None: ...
    ID_FIELD_NUMBER: _ClassVar[int]
    NAME_FIELD_NUMBER: _ClassVar[int]
    VERSION_FIELD_NUMBER: _ClassVar[int]
    ROLE_FIELD_NUMBER: _ClassVar[int]
    SUBJECT_FIELD_NUMBER: _ClassVar[int]
    TREATMENTS_FIELD_NUMBER: _ClassVar[int]
    LINEAGE_FIELD_NUMBER: _ClassVar[int]
    PLACEMENTS_FIELD_NUMBER: _ClassVar[int]
    STRATEGY_FIELD_NUMBER: _ClassVar[int]
    REGIONS_FIELD_NUMBER: _ClassVar[int]
    CONTRAST_THRESHOLD_FIELD_NUMBER: _ClassVar[int]
    SCAFFOLD_FIELD_NUMBER: _ClassVar[int]
    GENERATION_FIELD_NUMBER: _ClassVar[int]
    TREATMENT_PARAMS_FIELD_NUMBER: _ClassVar[int]
    INKS_FIELD_NUMBER: _ClassVar[int]
    PARENT_ID_FIELD_NUMBER: _ClassVar[int]
    QUALITY_TIER_FIELD_NUMBER: _ClassVar[int]
    PLATE_SPEC_FIELD_NUMBER: _ClassVar[int]
    id: str
    name: str
    version: int
    role: str
    subject: str
    treatments: _containers.RepeatedScalarFieldContainer[str]
    lineage: str
    placements: _containers.RepeatedScalarFieldContainer[str]
    strategy: str
    regions: _containers.RepeatedCompositeFieldContainer[ReservedRegion]
    contrast_threshold: float
    scaffold: ScaffoldBinding
    generation: GenerationBlock
    treatment_params: _containers.ScalarMap[str, str]
    inks: _containers.ScalarMap[str, str]
    parent_id: str
    quality_tier: QualityTier
    plate_spec: _containers.RepeatedCompositeFieldContainer[PlateSpec]
    def __init__(self, id: _Optional[str] = ..., name: _Optional[str] = ..., version: _Optional[int] = ..., role: _Optional[str] = ..., subject: _Optional[str] = ..., treatments: _Optional[_Iterable[str]] = ..., lineage: _Optional[str] = ..., placements: _Optional[_Iterable[str]] = ..., strategy: _Optional[str] = ..., regions: _Optional[_Iterable[_Union[ReservedRegion, _Mapping]]] = ..., contrast_threshold: _Optional[float] = ..., scaffold: _Optional[_Union[ScaffoldBinding, _Mapping]] = ..., generation: _Optional[_Union[GenerationBlock, _Mapping]] = ..., treatment_params: _Optional[_Mapping[str, str]] = ..., inks: _Optional[_Mapping[str, str]] = ..., parent_id: _Optional[str] = ..., quality_tier: _Optional[_Union[QualityTier, str]] = ..., plate_spec: _Optional[_Iterable[_Union[PlateSpec, _Mapping]]] = ...) -> None: ...
