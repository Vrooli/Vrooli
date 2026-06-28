from google.protobuf.internal import containers as _containers
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Iterable as _Iterable, Mapping as _Mapping
from typing import ClassVar as _ClassVar, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class ProviderStatus(_message.Message):
    __slots__ = ("name", "available")
    NAME_FIELD_NUMBER: _ClassVar[int]
    AVAILABLE_FIELD_NUMBER: _ClassVar[int]
    name: str
    available: bool
    def __init__(self, name: _Optional[str] = ..., available: _Optional[bool] = ...) -> None: ...

class GetProviderStatusRequest(_message.Message):
    __slots__ = ()
    def __init__(self) -> None: ...

class GetProviderStatusResponse(_message.Message):
    __slots__ = ("available", "providers")
    AVAILABLE_FIELD_NUMBER: _ClassVar[int]
    PROVIDERS_FIELD_NUMBER: _ClassVar[int]
    available: bool
    providers: _containers.RepeatedCompositeFieldContainer[ProviderStatus]
    def __init__(self, available: _Optional[bool] = ..., providers: _Optional[_Iterable[_Union[ProviderStatus, _Mapping]]] = ...) -> None: ...

class GenerateBrandElementsRequest(_message.Message):
    __slots__ = ("brand_id", "elements", "model")
    BRAND_ID_FIELD_NUMBER: _ClassVar[int]
    ELEMENTS_FIELD_NUMBER: _ClassVar[int]
    MODEL_FIELD_NUMBER: _ClassVar[int]
    brand_id: str
    elements: _containers.RepeatedScalarFieldContainer[str]
    model: str
    def __init__(self, brand_id: _Optional[str] = ..., elements: _Optional[_Iterable[str]] = ..., model: _Optional[str] = ...) -> None: ...

class ElementResult(_message.Message):
    __slots__ = ("element", "status", "detail")
    ELEMENT_FIELD_NUMBER: _ClassVar[int]
    STATUS_FIELD_NUMBER: _ClassVar[int]
    DETAIL_FIELD_NUMBER: _ClassVar[int]
    element: str
    status: str
    detail: str
    def __init__(self, element: _Optional[str] = ..., status: _Optional[str] = ..., detail: _Optional[str] = ...) -> None: ...

class GenerateBrandElementsResponse(_message.Message):
    __slots__ = ("results", "applied", "provider", "model", "brand_version")
    RESULTS_FIELD_NUMBER: _ClassVar[int]
    APPLIED_FIELD_NUMBER: _ClassVar[int]
    PROVIDER_FIELD_NUMBER: _ClassVar[int]
    MODEL_FIELD_NUMBER: _ClassVar[int]
    BRAND_VERSION_FIELD_NUMBER: _ClassVar[int]
    results: _containers.RepeatedCompositeFieldContainer[ElementResult]
    applied: _containers.RepeatedScalarFieldContainer[str]
    provider: str
    model: str
    brand_version: int
    def __init__(self, results: _Optional[_Iterable[_Union[ElementResult, _Mapping]]] = ..., applied: _Optional[_Iterable[str]] = ..., provider: _Optional[str] = ..., model: _Optional[str] = ..., brand_version: _Optional[int] = ...) -> None: ...

class GenerateBrandImageRequest(_message.Message):
    __slots__ = ("brand_id", "type", "model")
    BRAND_ID_FIELD_NUMBER: _ClassVar[int]
    TYPE_FIELD_NUMBER: _ClassVar[int]
    MODEL_FIELD_NUMBER: _ClassVar[int]
    brand_id: str
    type: str
    model: str
    def __init__(self, brand_id: _Optional[str] = ..., type: _Optional[str] = ..., model: _Optional[str] = ...) -> None: ...

class GenerateBrandImageResponse(_message.Message):
    __slots__ = ("brand_id", "asset_id", "type", "filename", "mime_type", "size", "provider", "model")
    BRAND_ID_FIELD_NUMBER: _ClassVar[int]
    ASSET_ID_FIELD_NUMBER: _ClassVar[int]
    TYPE_FIELD_NUMBER: _ClassVar[int]
    FILENAME_FIELD_NUMBER: _ClassVar[int]
    MIME_TYPE_FIELD_NUMBER: _ClassVar[int]
    SIZE_FIELD_NUMBER: _ClassVar[int]
    PROVIDER_FIELD_NUMBER: _ClassVar[int]
    MODEL_FIELD_NUMBER: _ClassVar[int]
    brand_id: str
    asset_id: str
    type: str
    filename: str
    mime_type: str
    size: int
    provider: str
    model: str
    def __init__(self, brand_id: _Optional[str] = ..., asset_id: _Optional[str] = ..., type: _Optional[str] = ..., filename: _Optional[str] = ..., mime_type: _Optional[str] = ..., size: _Optional[int] = ..., provider: _Optional[str] = ..., model: _Optional[str] = ...) -> None: ...
