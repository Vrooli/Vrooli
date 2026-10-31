from google.protobuf.internal import containers as _containers
from google.protobuf.internal import enum_type_wrapper as _enum_type_wrapper
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Iterable as _Iterable, Mapping as _Mapping
from typing import ClassVar as _ClassVar, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class LookKind(int, metaclass=_enum_type_wrapper.EnumTypeWrapper):
    __slots__ = ()
    LOOK_KIND_UNSPECIFIED: _ClassVar[LookKind]
    LOOK_KIND_STYLE: _ClassVar[LookKind]
    LOOK_KIND_FILM: _ClassVar[LookKind]
    LOOK_KIND_CAMERA: _ClassVar[LookKind]
    LOOK_KIND_ENHANCE: _ClassVar[LookKind]
    LOOK_KIND_CUSTOM: _ClassVar[LookKind]

class StepKind(int, metaclass=_enum_type_wrapper.EnumTypeWrapper):
    __slots__ = ()
    STEP_KIND_UNSPECIFIED: _ClassVar[StepKind]
    STEP_KIND_DETERMINISTIC: _ClassVar[StepKind]
    STEP_KIND_AI: _ClassVar[StepKind]
LOOK_KIND_UNSPECIFIED: LookKind
LOOK_KIND_STYLE: LookKind
LOOK_KIND_FILM: LookKind
LOOK_KIND_CAMERA: LookKind
LOOK_KIND_ENHANCE: LookKind
LOOK_KIND_CUSTOM: LookKind
STEP_KIND_UNSPECIFIED: StepKind
STEP_KIND_DETERMINISTIC: StepKind
STEP_KIND_AI: StepKind

class LookStep(_message.Message):
    __slots__ = ("operation", "kind", "params")
    class ParamsEntry(_message.Message):
        __slots__ = ("key", "value")
        KEY_FIELD_NUMBER: _ClassVar[int]
        VALUE_FIELD_NUMBER: _ClassVar[int]
        key: str
        value: str
        def __init__(self, key: _Optional[str] = ..., value: _Optional[str] = ...) -> None: ...
    OPERATION_FIELD_NUMBER: _ClassVar[int]
    KIND_FIELD_NUMBER: _ClassVar[int]
    PARAMS_FIELD_NUMBER: _ClassVar[int]
    operation: str
    kind: StepKind
    params: _containers.ScalarMap[str, str]
    def __init__(self, operation: _Optional[str] = ..., kind: _Optional[_Union[StepKind, str]] = ..., params: _Optional[_Mapping[str, str]] = ...) -> None: ...

class Look(_message.Message):
    __slots__ = ("id", "name", "description", "kind", "steps", "prompt_template", "params", "thumbnail_ref", "builtin", "created_at", "updated_at")
    class ParamsEntry(_message.Message):
        __slots__ = ("key", "value")
        KEY_FIELD_NUMBER: _ClassVar[int]
        VALUE_FIELD_NUMBER: _ClassVar[int]
        key: str
        value: str
        def __init__(self, key: _Optional[str] = ..., value: _Optional[str] = ...) -> None: ...
    ID_FIELD_NUMBER: _ClassVar[int]
    NAME_FIELD_NUMBER: _ClassVar[int]
    DESCRIPTION_FIELD_NUMBER: _ClassVar[int]
    KIND_FIELD_NUMBER: _ClassVar[int]
    STEPS_FIELD_NUMBER: _ClassVar[int]
    PROMPT_TEMPLATE_FIELD_NUMBER: _ClassVar[int]
    PARAMS_FIELD_NUMBER: _ClassVar[int]
    THUMBNAIL_REF_FIELD_NUMBER: _ClassVar[int]
    BUILTIN_FIELD_NUMBER: _ClassVar[int]
    CREATED_AT_FIELD_NUMBER: _ClassVar[int]
    UPDATED_AT_FIELD_NUMBER: _ClassVar[int]
    id: str
    name: str
    description: str
    kind: LookKind
    steps: _containers.RepeatedCompositeFieldContainer[LookStep]
    prompt_template: str
    params: _containers.ScalarMap[str, str]
    thumbnail_ref: str
    builtin: bool
    created_at: str
    updated_at: str
    def __init__(self, id: _Optional[str] = ..., name: _Optional[str] = ..., description: _Optional[str] = ..., kind: _Optional[_Union[LookKind, str]] = ..., steps: _Optional[_Iterable[_Union[LookStep, _Mapping]]] = ..., prompt_template: _Optional[str] = ..., params: _Optional[_Mapping[str, str]] = ..., thumbnail_ref: _Optional[str] = ..., builtin: _Optional[bool] = ..., created_at: _Optional[str] = ..., updated_at: _Optional[str] = ...) -> None: ...

class ListLooksRequest(_message.Message):
    __slots__ = ("kind",)
    KIND_FIELD_NUMBER: _ClassVar[int]
    kind: LookKind
    def __init__(self, kind: _Optional[_Union[LookKind, str]] = ...) -> None: ...

class ListLooksResponse(_message.Message):
    __slots__ = ("looks",)
    LOOKS_FIELD_NUMBER: _ClassVar[int]
    looks: _containers.RepeatedCompositeFieldContainer[Look]
    def __init__(self, looks: _Optional[_Iterable[_Union[Look, _Mapping]]] = ...) -> None: ...

class GetLookRequest(_message.Message):
    __slots__ = ("id",)
    ID_FIELD_NUMBER: _ClassVar[int]
    id: str
    def __init__(self, id: _Optional[str] = ...) -> None: ...

class GetLookResponse(_message.Message):
    __slots__ = ("look",)
    LOOK_FIELD_NUMBER: _ClassVar[int]
    look: Look
    def __init__(self, look: _Optional[_Union[Look, _Mapping]] = ...) -> None: ...

class CreateLookRequest(_message.Message):
    __slots__ = ("look",)
    LOOK_FIELD_NUMBER: _ClassVar[int]
    look: Look
    def __init__(self, look: _Optional[_Union[Look, _Mapping]] = ...) -> None: ...

class CreateLookResponse(_message.Message):
    __slots__ = ("look",)
    LOOK_FIELD_NUMBER: _ClassVar[int]
    look: Look
    def __init__(self, look: _Optional[_Union[Look, _Mapping]] = ...) -> None: ...

class UpdateLookRequest(_message.Message):
    __slots__ = ("look",)
    LOOK_FIELD_NUMBER: _ClassVar[int]
    look: Look
    def __init__(self, look: _Optional[_Union[Look, _Mapping]] = ...) -> None: ...

class UpdateLookResponse(_message.Message):
    __slots__ = ("look",)
    LOOK_FIELD_NUMBER: _ClassVar[int]
    look: Look
    def __init__(self, look: _Optional[_Union[Look, _Mapping]] = ...) -> None: ...

class DeleteLookRequest(_message.Message):
    __slots__ = ("id",)
    ID_FIELD_NUMBER: _ClassVar[int]
    id: str
    def __init__(self, id: _Optional[str] = ...) -> None: ...

class DeleteLookResponse(_message.Message):
    __slots__ = ("deleted",)
    DELETED_FIELD_NUMBER: _ClassVar[int]
    deleted: bool
    def __init__(self, deleted: _Optional[bool] = ...) -> None: ...

class CompileLookRequest(_message.Message):
    __slots__ = ("look_id", "subject", "prompt", "has_input")
    LOOK_ID_FIELD_NUMBER: _ClassVar[int]
    SUBJECT_FIELD_NUMBER: _ClassVar[int]
    PROMPT_FIELD_NUMBER: _ClassVar[int]
    HAS_INPUT_FIELD_NUMBER: _ClassVar[int]
    look_id: str
    subject: str
    prompt: str
    has_input: bool
    def __init__(self, look_id: _Optional[str] = ..., subject: _Optional[str] = ..., prompt: _Optional[str] = ..., has_input: _Optional[bool] = ...) -> None: ...

class CompiledStep(_message.Message):
    __slots__ = ("operation", "kind", "params")
    class ParamsEntry(_message.Message):
        __slots__ = ("key", "value")
        KEY_FIELD_NUMBER: _ClassVar[int]
        VALUE_FIELD_NUMBER: _ClassVar[int]
        key: str
        value: str
        def __init__(self, key: _Optional[str] = ..., value: _Optional[str] = ...) -> None: ...
    OPERATION_FIELD_NUMBER: _ClassVar[int]
    KIND_FIELD_NUMBER: _ClassVar[int]
    PARAMS_FIELD_NUMBER: _ClassVar[int]
    operation: str
    kind: StepKind
    params: _containers.ScalarMap[str, str]
    def __init__(self, operation: _Optional[str] = ..., kind: _Optional[_Union[StepKind, str]] = ..., params: _Optional[_Mapping[str, str]] = ...) -> None: ...

class CompileLookResponse(_message.Message):
    __slots__ = ("steps", "primary_prompt", "requires_image", "requires_mask", "warnings")
    STEPS_FIELD_NUMBER: _ClassVar[int]
    PRIMARY_PROMPT_FIELD_NUMBER: _ClassVar[int]
    REQUIRES_IMAGE_FIELD_NUMBER: _ClassVar[int]
    REQUIRES_MASK_FIELD_NUMBER: _ClassVar[int]
    WARNINGS_FIELD_NUMBER: _ClassVar[int]
    steps: _containers.RepeatedCompositeFieldContainer[CompiledStep]
    primary_prompt: str
    requires_image: bool
    requires_mask: bool
    warnings: _containers.RepeatedScalarFieldContainer[str]
    def __init__(self, steps: _Optional[_Iterable[_Union[CompiledStep, _Mapping]]] = ..., primary_prompt: _Optional[str] = ..., requires_image: _Optional[bool] = ..., requires_mask: _Optional[bool] = ..., warnings: _Optional[_Iterable[str]] = ...) -> None: ...

class RenderPreviewRequest(_message.Message):
    __slots__ = ("look_id",)
    LOOK_ID_FIELD_NUMBER: _ClassVar[int]
    look_id: str
    def __init__(self, look_id: _Optional[str] = ...) -> None: ...

class RenderPreviewResponse(_message.Message):
    __slots__ = ("thumbnail_ref", "deferred_steps")
    THUMBNAIL_REF_FIELD_NUMBER: _ClassVar[int]
    DEFERRED_STEPS_FIELD_NUMBER: _ClassVar[int]
    thumbnail_ref: str
    deferred_steps: _containers.RepeatedScalarFieldContainer[str]
    def __init__(self, thumbnail_ref: _Optional[str] = ..., deferred_steps: _Optional[_Iterable[str]] = ...) -> None: ...
