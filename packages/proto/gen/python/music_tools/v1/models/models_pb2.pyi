from google.protobuf.internal import containers as _containers
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Iterable as _Iterable, Mapping as _Mapping
from typing import ClassVar as _ClassVar, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class Model(_message.Message):
    __slots__ = ("id", "display_name", "variant", "license_lane", "status", "revision", "bytes", "engine", "sung_lyrics", "voice_reference", "composition_controls")
    ID_FIELD_NUMBER: _ClassVar[int]
    DISPLAY_NAME_FIELD_NUMBER: _ClassVar[int]
    VARIANT_FIELD_NUMBER: _ClassVar[int]
    LICENSE_LANE_FIELD_NUMBER: _ClassVar[int]
    STATUS_FIELD_NUMBER: _ClassVar[int]
    REVISION_FIELD_NUMBER: _ClassVar[int]
    BYTES_FIELD_NUMBER: _ClassVar[int]
    ENGINE_FIELD_NUMBER: _ClassVar[int]
    SUNG_LYRICS_FIELD_NUMBER: _ClassVar[int]
    VOICE_REFERENCE_FIELD_NUMBER: _ClassVar[int]
    COMPOSITION_CONTROLS_FIELD_NUMBER: _ClassVar[int]
    id: str
    display_name: str
    variant: str
    license_lane: str
    status: str
    revision: str
    bytes: int
    engine: str
    sung_lyrics: ModelCapability
    voice_reference: ModelCapability
    composition_controls: CompositionControls
    def __init__(self, id: _Optional[str] = ..., display_name: _Optional[str] = ..., variant: _Optional[str] = ..., license_lane: _Optional[str] = ..., status: _Optional[str] = ..., revision: _Optional[str] = ..., bytes: _Optional[int] = ..., engine: _Optional[str] = ..., sung_lyrics: _Optional[_Union[ModelCapability, _Mapping]] = ..., voice_reference: _Optional[_Union[ModelCapability, _Mapping]] = ..., composition_controls: _Optional[_Union[CompositionControls, _Mapping]] = ...) -> None: ...

class ModelCapability(_message.Message):
    __slots__ = ("supported", "reason", "source")
    SUPPORTED_FIELD_NUMBER: _ClassVar[int]
    REASON_FIELD_NUMBER: _ClassVar[int]
    SOURCE_FIELD_NUMBER: _ClassVar[int]
    supported: bool
    reason: str
    source: str
    def __init__(self, supported: _Optional[bool] = ..., reason: _Optional[str] = ..., source: _Optional[str] = ...) -> None: ...

class CompositionControls(_message.Message):
    __slots__ = ("min_duration_seconds", "max_duration_seconds", "default_steps", "min_steps", "max_steps", "default_guidance", "min_guidance", "max_guidance", "duration_control")
    MIN_DURATION_SECONDS_FIELD_NUMBER: _ClassVar[int]
    MAX_DURATION_SECONDS_FIELD_NUMBER: _ClassVar[int]
    DEFAULT_STEPS_FIELD_NUMBER: _ClassVar[int]
    MIN_STEPS_FIELD_NUMBER: _ClassVar[int]
    MAX_STEPS_FIELD_NUMBER: _ClassVar[int]
    DEFAULT_GUIDANCE_FIELD_NUMBER: _ClassVar[int]
    MIN_GUIDANCE_FIELD_NUMBER: _ClassVar[int]
    MAX_GUIDANCE_FIELD_NUMBER: _ClassVar[int]
    DURATION_CONTROL_FIELD_NUMBER: _ClassVar[int]
    min_duration_seconds: float
    max_duration_seconds: float
    default_steps: int
    min_steps: int
    max_steps: int
    default_guidance: float
    min_guidance: float
    max_guidance: float
    duration_control: str
    def __init__(self, min_duration_seconds: _Optional[float] = ..., max_duration_seconds: _Optional[float] = ..., default_steps: _Optional[int] = ..., min_steps: _Optional[int] = ..., max_steps: _Optional[int] = ..., default_guidance: _Optional[float] = ..., min_guidance: _Optional[float] = ..., max_guidance: _Optional[float] = ..., duration_control: _Optional[str] = ...) -> None: ...

class ListModelsRequest(_message.Message):
    __slots__ = ("variant",)
    VARIANT_FIELD_NUMBER: _ClassVar[int]
    variant: str
    def __init__(self, variant: _Optional[str] = ...) -> None: ...

class ListModelsResponse(_message.Message):
    __slots__ = ("models",)
    MODELS_FIELD_NUMBER: _ClassVar[int]
    models: _containers.RepeatedCompositeFieldContainer[Model]
    def __init__(self, models: _Optional[_Iterable[_Union[Model, _Mapping]]] = ...) -> None: ...

class GetModelRequest(_message.Message):
    __slots__ = ("id",)
    ID_FIELD_NUMBER: _ClassVar[int]
    id: str
    def __init__(self, id: _Optional[str] = ...) -> None: ...
