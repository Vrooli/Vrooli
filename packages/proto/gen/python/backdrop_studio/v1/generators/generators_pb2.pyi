from google.protobuf.internal import containers as _containers
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Iterable as _Iterable, Mapping as _Mapping
from typing import ClassVar as _ClassVar, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class ParamSpec(_message.Message):
    __slots__ = ("name", "min", "max", "default", "description")
    NAME_FIELD_NUMBER: _ClassVar[int]
    MIN_FIELD_NUMBER: _ClassVar[int]
    MAX_FIELD_NUMBER: _ClassVar[int]
    DEFAULT_FIELD_NUMBER: _ClassVar[int]
    DESCRIPTION_FIELD_NUMBER: _ClassVar[int]
    name: str
    min: float
    max: float
    default: float
    description: str
    def __init__(self, name: _Optional[str] = ..., min: _Optional[float] = ..., max: _Optional[float] = ..., default: _Optional[float] = ..., description: _Optional[str] = ...) -> None: ...

class Check(_message.Message):
    __slots__ = ("name", "passed", "detail")
    NAME_FIELD_NUMBER: _ClassVar[int]
    PASSED_FIELD_NUMBER: _ClassVar[int]
    DETAIL_FIELD_NUMBER: _ClassVar[int]
    name: str
    passed: bool
    detail: str
    def __init__(self, name: _Optional[str] = ..., passed: _Optional[bool] = ..., detail: _Optional[str] = ...) -> None: ...

class ValidationReport(_message.Message):
    __slots__ = ("passed", "checks", "refusals")
    PASSED_FIELD_NUMBER: _ClassVar[int]
    CHECKS_FIELD_NUMBER: _ClassVar[int]
    REFUSALS_FIELD_NUMBER: _ClassVar[int]
    passed: bool
    checks: _containers.RepeatedCompositeFieldContainer[Check]
    refusals: _containers.RepeatedScalarFieldContainer[str]
    def __init__(self, passed: _Optional[bool] = ..., checks: _Optional[_Iterable[_Union[Check, _Mapping]]] = ..., refusals: _Optional[_Iterable[str]] = ...) -> None: ...

class Generator(_message.Message):
    __slots__ = ("id", "name", "template", "params", "inks", "prompt", "model_id", "validation")
    ID_FIELD_NUMBER: _ClassVar[int]
    NAME_FIELD_NUMBER: _ClassVar[int]
    TEMPLATE_FIELD_NUMBER: _ClassVar[int]
    PARAMS_FIELD_NUMBER: _ClassVar[int]
    INKS_FIELD_NUMBER: _ClassVar[int]
    PROMPT_FIELD_NUMBER: _ClassVar[int]
    MODEL_ID_FIELD_NUMBER: _ClassVar[int]
    VALIDATION_FIELD_NUMBER: _ClassVar[int]
    id: str
    name: str
    template: str
    params: _containers.RepeatedCompositeFieldContainer[ParamSpec]
    inks: _containers.RepeatedScalarFieldContainer[str]
    prompt: str
    model_id: str
    validation: ValidationReport
    def __init__(self, id: _Optional[str] = ..., name: _Optional[str] = ..., template: _Optional[str] = ..., params: _Optional[_Iterable[_Union[ParamSpec, _Mapping]]] = ..., inks: _Optional[_Iterable[str]] = ..., prompt: _Optional[str] = ..., model_id: _Optional[str] = ..., validation: _Optional[_Union[ValidationReport, _Mapping]] = ...) -> None: ...

class AuthorGeneratorRequest(_message.Message):
    __slots__ = ("id", "brief", "store")
    ID_FIELD_NUMBER: _ClassVar[int]
    BRIEF_FIELD_NUMBER: _ClassVar[int]
    STORE_FIELD_NUMBER: _ClassVar[int]
    id: str
    brief: str
    store: bool
    def __init__(self, id: _Optional[str] = ..., brief: _Optional[str] = ..., store: _Optional[bool] = ...) -> None: ...

class AuthorGeneratorResponse(_message.Message):
    __slots__ = ("generator", "stored")
    GENERATOR_FIELD_NUMBER: _ClassVar[int]
    STORED_FIELD_NUMBER: _ClassVar[int]
    generator: Generator
    stored: bool
    def __init__(self, generator: _Optional[_Union[Generator, _Mapping]] = ..., stored: _Optional[bool] = ...) -> None: ...

class ListGeneratorsRequest(_message.Message):
    __slots__ = ()
    def __init__(self) -> None: ...

class ListGeneratorsResponse(_message.Message):
    __slots__ = ("generators",)
    GENERATORS_FIELD_NUMBER: _ClassVar[int]
    generators: _containers.RepeatedCompositeFieldContainer[Generator]
    def __init__(self, generators: _Optional[_Iterable[_Union[Generator, _Mapping]]] = ...) -> None: ...

class DeleteGeneratorRequest(_message.Message):
    __slots__ = ("id",)
    ID_FIELD_NUMBER: _ClassVar[int]
    id: str
    def __init__(self, id: _Optional[str] = ...) -> None: ...

class DeleteGeneratorResponse(_message.Message):
    __slots__ = ("deleted",)
    DELETED_FIELD_NUMBER: _ClassVar[int]
    deleted: bool
    def __init__(self, deleted: _Optional[bool] = ...) -> None: ...
