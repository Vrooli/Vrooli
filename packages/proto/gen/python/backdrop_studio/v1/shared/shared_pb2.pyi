from google.protobuf.internal import containers as _containers
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Iterable as _Iterable, Mapping as _Mapping
from typing import ClassVar as _ClassVar, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

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
    __slots__ = ("role", "profile", "prompt_template", "negative", "model", "provider_url", "credential")
    ROLE_FIELD_NUMBER: _ClassVar[int]
    PROFILE_FIELD_NUMBER: _ClassVar[int]
    PROMPT_TEMPLATE_FIELD_NUMBER: _ClassVar[int]
    NEGATIVE_FIELD_NUMBER: _ClassVar[int]
    MODEL_FIELD_NUMBER: _ClassVar[int]
    PROVIDER_URL_FIELD_NUMBER: _ClassVar[int]
    CREDENTIAL_FIELD_NUMBER: _ClassVar[int]
    role: str
    profile: str
    prompt_template: str
    negative: str
    model: str
    provider_url: str
    credential: str
    def __init__(self, role: _Optional[str] = ..., profile: _Optional[str] = ..., prompt_template: _Optional[str] = ..., negative: _Optional[str] = ..., model: _Optional[str] = ..., provider_url: _Optional[str] = ..., credential: _Optional[str] = ...) -> None: ...

class Style(_message.Message):
    __slots__ = ("id", "name", "version", "role", "subject", "treatments", "lineage", "placements", "strategy", "regions", "contrast_threshold", "scaffold", "generation")
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
    def __init__(self, id: _Optional[str] = ..., name: _Optional[str] = ..., version: _Optional[int] = ..., role: _Optional[str] = ..., subject: _Optional[str] = ..., treatments: _Optional[_Iterable[str]] = ..., lineage: _Optional[str] = ..., placements: _Optional[_Iterable[str]] = ..., strategy: _Optional[str] = ..., regions: _Optional[_Iterable[_Union[ReservedRegion, _Mapping]]] = ..., contrast_threshold: _Optional[float] = ..., scaffold: _Optional[_Union[ScaffoldBinding, _Mapping]] = ..., generation: _Optional[_Union[GenerationBlock, _Mapping]] = ...) -> None: ...
