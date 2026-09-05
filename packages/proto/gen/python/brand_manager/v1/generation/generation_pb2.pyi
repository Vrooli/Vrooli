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

class GetImageBackendStatusRequest(_message.Message):
    __slots__ = ()
    def __init__(self) -> None: ...

class ImageOperationStatus(_message.Message):
    __slots__ = ("operation", "ready", "model_id", "tier", "hint", "warnings")
    OPERATION_FIELD_NUMBER: _ClassVar[int]
    READY_FIELD_NUMBER: _ClassVar[int]
    MODEL_ID_FIELD_NUMBER: _ClassVar[int]
    TIER_FIELD_NUMBER: _ClassVar[int]
    HINT_FIELD_NUMBER: _ClassVar[int]
    WARNINGS_FIELD_NUMBER: _ClassVar[int]
    operation: str
    ready: bool
    model_id: str
    tier: str
    hint: str
    warnings: _containers.RepeatedScalarFieldContainer[str]
    def __init__(self, operation: _Optional[str] = ..., ready: _Optional[bool] = ..., model_id: _Optional[str] = ..., tier: _Optional[str] = ..., hint: _Optional[str] = ..., warnings: _Optional[_Iterable[str]] = ...) -> None: ...

class GetImageBackendStatusResponse(_message.Message):
    __slots__ = ("available", "detail", "operations")
    AVAILABLE_FIELD_NUMBER: _ClassVar[int]
    DETAIL_FIELD_NUMBER: _ClassVar[int]
    OPERATIONS_FIELD_NUMBER: _ClassVar[int]
    available: bool
    detail: str
    operations: _containers.RepeatedCompositeFieldContainer[ImageOperationStatus]
    def __init__(self, available: _Optional[bool] = ..., detail: _Optional[str] = ..., operations: _Optional[_Iterable[_Union[ImageOperationStatus, _Mapping]]] = ...) -> None: ...

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

class BrandImageAsset(_message.Message):
    __slots__ = ("brand_id", "asset_id", "kind", "filename", "mime_type", "size", "model_id", "tier", "canonical", "warnings")
    BRAND_ID_FIELD_NUMBER: _ClassVar[int]
    ASSET_ID_FIELD_NUMBER: _ClassVar[int]
    KIND_FIELD_NUMBER: _ClassVar[int]
    FILENAME_FIELD_NUMBER: _ClassVar[int]
    MIME_TYPE_FIELD_NUMBER: _ClassVar[int]
    SIZE_FIELD_NUMBER: _ClassVar[int]
    MODEL_ID_FIELD_NUMBER: _ClassVar[int]
    TIER_FIELD_NUMBER: _ClassVar[int]
    CANONICAL_FIELD_NUMBER: _ClassVar[int]
    WARNINGS_FIELD_NUMBER: _ClassVar[int]
    brand_id: str
    asset_id: str
    kind: str
    filename: str
    mime_type: str
    size: int
    model_id: str
    tier: str
    canonical: bool
    warnings: _containers.RepeatedScalarFieldContainer[str]
    def __init__(self, brand_id: _Optional[str] = ..., asset_id: _Optional[str] = ..., kind: _Optional[str] = ..., filename: _Optional[str] = ..., mime_type: _Optional[str] = ..., size: _Optional[int] = ..., model_id: _Optional[str] = ..., tier: _Optional[str] = ..., canonical: _Optional[bool] = ..., warnings: _Optional[_Iterable[str]] = ...) -> None: ...

class GenerateBrandImageRequest(_message.Message):
    __slots__ = ("brand_id", "type", "model_override", "allow_byok", "seed", "set_canonical", "quality_policy", "fallback_policy", "priority", "allow_reclaim")
    BRAND_ID_FIELD_NUMBER: _ClassVar[int]
    TYPE_FIELD_NUMBER: _ClassVar[int]
    MODEL_OVERRIDE_FIELD_NUMBER: _ClassVar[int]
    ALLOW_BYOK_FIELD_NUMBER: _ClassVar[int]
    SEED_FIELD_NUMBER: _ClassVar[int]
    SET_CANONICAL_FIELD_NUMBER: _ClassVar[int]
    QUALITY_POLICY_FIELD_NUMBER: _ClassVar[int]
    FALLBACK_POLICY_FIELD_NUMBER: _ClassVar[int]
    PRIORITY_FIELD_NUMBER: _ClassVar[int]
    ALLOW_RECLAIM_FIELD_NUMBER: _ClassVar[int]
    brand_id: str
    type: str
    model_override: str
    allow_byok: bool
    seed: int
    set_canonical: bool
    quality_policy: str
    fallback_policy: str
    priority: str
    allow_reclaim: bool
    def __init__(self, brand_id: _Optional[str] = ..., type: _Optional[str] = ..., model_override: _Optional[str] = ..., allow_byok: _Optional[bool] = ..., seed: _Optional[int] = ..., set_canonical: _Optional[bool] = ..., quality_policy: _Optional[str] = ..., fallback_policy: _Optional[str] = ..., priority: _Optional[str] = ..., allow_reclaim: _Optional[bool] = ...) -> None: ...

class EditBrandImageRequest(_message.Message):
    __slots__ = ("brand_id", "source_asset_id", "instruction", "model_override", "allow_byok", "seed", "set_canonical", "quality_policy", "fallback_policy", "priority", "allow_reclaim")
    BRAND_ID_FIELD_NUMBER: _ClassVar[int]
    SOURCE_ASSET_ID_FIELD_NUMBER: _ClassVar[int]
    INSTRUCTION_FIELD_NUMBER: _ClassVar[int]
    MODEL_OVERRIDE_FIELD_NUMBER: _ClassVar[int]
    ALLOW_BYOK_FIELD_NUMBER: _ClassVar[int]
    SEED_FIELD_NUMBER: _ClassVar[int]
    SET_CANONICAL_FIELD_NUMBER: _ClassVar[int]
    QUALITY_POLICY_FIELD_NUMBER: _ClassVar[int]
    FALLBACK_POLICY_FIELD_NUMBER: _ClassVar[int]
    PRIORITY_FIELD_NUMBER: _ClassVar[int]
    ALLOW_RECLAIM_FIELD_NUMBER: _ClassVar[int]
    brand_id: str
    source_asset_id: str
    instruction: str
    model_override: str
    allow_byok: bool
    seed: int
    set_canonical: bool
    quality_policy: str
    fallback_policy: str
    priority: str
    allow_reclaim: bool
    def __init__(self, brand_id: _Optional[str] = ..., source_asset_id: _Optional[str] = ..., instruction: _Optional[str] = ..., model_override: _Optional[str] = ..., allow_byok: _Optional[bool] = ..., seed: _Optional[int] = ..., set_canonical: _Optional[bool] = ..., quality_policy: _Optional[str] = ..., fallback_policy: _Optional[str] = ..., priority: _Optional[str] = ..., allow_reclaim: _Optional[bool] = ...) -> None: ...

class RemoveBrandImageBackgroundRequest(_message.Message):
    __slots__ = ("brand_id", "source_asset_id", "model_override", "allow_byok", "set_canonical")
    BRAND_ID_FIELD_NUMBER: _ClassVar[int]
    SOURCE_ASSET_ID_FIELD_NUMBER: _ClassVar[int]
    MODEL_OVERRIDE_FIELD_NUMBER: _ClassVar[int]
    ALLOW_BYOK_FIELD_NUMBER: _ClassVar[int]
    SET_CANONICAL_FIELD_NUMBER: _ClassVar[int]
    brand_id: str
    source_asset_id: str
    model_override: str
    allow_byok: bool
    set_canonical: bool
    def __init__(self, brand_id: _Optional[str] = ..., source_asset_id: _Optional[str] = ..., model_override: _Optional[str] = ..., allow_byok: _Optional[bool] = ..., set_canonical: _Optional[bool] = ...) -> None: ...

class DeriveBrandIconsRequest(_message.Message):
    __slots__ = ("brand_id", "source_asset_id", "include_maskable", "include_apple_touch", "include_favicon")
    BRAND_ID_FIELD_NUMBER: _ClassVar[int]
    SOURCE_ASSET_ID_FIELD_NUMBER: _ClassVar[int]
    INCLUDE_MASKABLE_FIELD_NUMBER: _ClassVar[int]
    INCLUDE_APPLE_TOUCH_FIELD_NUMBER: _ClassVar[int]
    INCLUDE_FAVICON_FIELD_NUMBER: _ClassVar[int]
    brand_id: str
    source_asset_id: str
    include_maskable: bool
    include_apple_touch: bool
    include_favicon: bool
    def __init__(self, brand_id: _Optional[str] = ..., source_asset_id: _Optional[str] = ..., include_maskable: _Optional[bool] = ..., include_apple_touch: _Optional[bool] = ..., include_favicon: _Optional[bool] = ...) -> None: ...

class DeriveBrandIconsResponse(_message.Message):
    __slots__ = ("icons", "warnings")
    ICONS_FIELD_NUMBER: _ClassVar[int]
    WARNINGS_FIELD_NUMBER: _ClassVar[int]
    icons: _containers.RepeatedCompositeFieldContainer[BrandImageAsset]
    warnings: _containers.RepeatedScalarFieldContainer[str]
    def __init__(self, icons: _Optional[_Iterable[_Union[BrandImageAsset, _Mapping]]] = ..., warnings: _Optional[_Iterable[str]] = ...) -> None: ...
