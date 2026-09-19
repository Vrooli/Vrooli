from google.protobuf.internal import containers as _containers
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Iterable as _Iterable, Mapping as _Mapping
from typing import ClassVar as _ClassVar, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class Style(_message.Message):
    __slots__ = ("id", "name", "description", "caption", "bpm", "keyscale", "duration", "inference_steps", "guidance_scale", "variant", "builtin")
    ID_FIELD_NUMBER: _ClassVar[int]
    NAME_FIELD_NUMBER: _ClassVar[int]
    DESCRIPTION_FIELD_NUMBER: _ClassVar[int]
    CAPTION_FIELD_NUMBER: _ClassVar[int]
    BPM_FIELD_NUMBER: _ClassVar[int]
    KEYSCALE_FIELD_NUMBER: _ClassVar[int]
    DURATION_FIELD_NUMBER: _ClassVar[int]
    INFERENCE_STEPS_FIELD_NUMBER: _ClassVar[int]
    GUIDANCE_SCALE_FIELD_NUMBER: _ClassVar[int]
    VARIANT_FIELD_NUMBER: _ClassVar[int]
    BUILTIN_FIELD_NUMBER: _ClassVar[int]
    id: str
    name: str
    description: str
    caption: str
    bpm: int
    keyscale: str
    duration: int
    inference_steps: int
    guidance_scale: float
    variant: str
    builtin: bool
    def __init__(self, id: _Optional[str] = ..., name: _Optional[str] = ..., description: _Optional[str] = ..., caption: _Optional[str] = ..., bpm: _Optional[int] = ..., keyscale: _Optional[str] = ..., duration: _Optional[int] = ..., inference_steps: _Optional[int] = ..., guidance_scale: _Optional[float] = ..., variant: _Optional[str] = ..., builtin: _Optional[bool] = ...) -> None: ...

class CompiledStyle(_message.Message):
    __slots__ = ("style", "caption_as_sent")
    STYLE_FIELD_NUMBER: _ClassVar[int]
    CAPTION_AS_SENT_FIELD_NUMBER: _ClassVar[int]
    style: Style
    caption_as_sent: str
    def __init__(self, style: _Optional[_Union[Style, _Mapping]] = ..., caption_as_sent: _Optional[str] = ...) -> None: ...

class CreateStyleRequest(_message.Message):
    __slots__ = ("style",)
    STYLE_FIELD_NUMBER: _ClassVar[int]
    style: Style
    def __init__(self, style: _Optional[_Union[Style, _Mapping]] = ...) -> None: ...

class GetStyleRequest(_message.Message):
    __slots__ = ("id",)
    ID_FIELD_NUMBER: _ClassVar[int]
    id: str
    def __init__(self, id: _Optional[str] = ...) -> None: ...

class CompileStyleRequest(_message.Message):
    __slots__ = ("id",)
    ID_FIELD_NUMBER: _ClassVar[int]
    id: str
    def __init__(self, id: _Optional[str] = ...) -> None: ...

class ListStylesRequest(_message.Message):
    __slots__ = ()
    def __init__(self) -> None: ...

class ListStylesResponse(_message.Message):
    __slots__ = ("styles",)
    STYLES_FIELD_NUMBER: _ClassVar[int]
    styles: _containers.RepeatedCompositeFieldContainer[Style]
    def __init__(self, styles: _Optional[_Iterable[_Union[Style, _Mapping]]] = ...) -> None: ...
