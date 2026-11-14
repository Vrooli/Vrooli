from google.protobuf.internal import containers as _containers
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Iterable as _Iterable, Mapping as _Mapping
from typing import ClassVar as _ClassVar, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class PreviewApplyRequest(_message.Message):
    __slots__ = ("brand_id", "scenario_name", "elements")
    BRAND_ID_FIELD_NUMBER: _ClassVar[int]
    SCENARIO_NAME_FIELD_NUMBER: _ClassVar[int]
    ELEMENTS_FIELD_NUMBER: _ClassVar[int]
    brand_id: str
    scenario_name: str
    elements: _containers.RepeatedScalarFieldContainer[str]
    def __init__(self, brand_id: _Optional[str] = ..., scenario_name: _Optional[str] = ..., elements: _Optional[_Iterable[str]] = ...) -> None: ...

class ApplyBrandRequest(_message.Message):
    __slots__ = ("brand_id", "scenario_name", "elements")
    BRAND_ID_FIELD_NUMBER: _ClassVar[int]
    SCENARIO_NAME_FIELD_NUMBER: _ClassVar[int]
    ELEMENTS_FIELD_NUMBER: _ClassVar[int]
    brand_id: str
    scenario_name: str
    elements: _containers.RepeatedScalarFieldContainer[str]
    def __init__(self, brand_id: _Optional[str] = ..., scenario_name: _Optional[str] = ..., elements: _Optional[_Iterable[str]] = ...) -> None: ...

class ApplyAction(_message.Message):
    __slots__ = ("type", "file", "element")
    TYPE_FIELD_NUMBER: _ClassVar[int]
    FILE_FIELD_NUMBER: _ClassVar[int]
    ELEMENT_FIELD_NUMBER: _ClassVar[int]
    type: str
    file: str
    element: str
    def __init__(self, type: _Optional[str] = ..., file: _Optional[str] = ..., element: _Optional[str] = ...) -> None: ...

class SkipReason(_message.Message):
    __slots__ = ("element", "reason")
    ELEMENT_FIELD_NUMBER: _ClassVar[int]
    REASON_FIELD_NUMBER: _ClassVar[int]
    element: str
    reason: str
    def __init__(self, element: _Optional[str] = ..., reason: _Optional[str] = ...) -> None: ...

class ApplyResponse(_message.Message):
    __slots__ = ("scenario", "brand_id", "brand_version", "dry_run", "applied", "skipped")
    SCENARIO_FIELD_NUMBER: _ClassVar[int]
    BRAND_ID_FIELD_NUMBER: _ClassVar[int]
    BRAND_VERSION_FIELD_NUMBER: _ClassVar[int]
    DRY_RUN_FIELD_NUMBER: _ClassVar[int]
    APPLIED_FIELD_NUMBER: _ClassVar[int]
    SKIPPED_FIELD_NUMBER: _ClassVar[int]
    scenario: str
    brand_id: str
    brand_version: int
    dry_run: bool
    applied: _containers.RepeatedCompositeFieldContainer[ApplyAction]
    skipped: _containers.RepeatedCompositeFieldContainer[SkipReason]
    def __init__(self, scenario: _Optional[str] = ..., brand_id: _Optional[str] = ..., brand_version: _Optional[int] = ..., dry_run: _Optional[bool] = ..., applied: _Optional[_Iterable[_Union[ApplyAction, _Mapping]]] = ..., skipped: _Optional[_Iterable[_Union[SkipReason, _Mapping]]] = ...) -> None: ...
