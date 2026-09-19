from landing_page_react_vite.v1 import pricing_pb2 as _pricing_pb2
from google.protobuf.internal import containers as _containers
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Iterable as _Iterable, Mapping as _Mapping
from typing import ClassVar as _ClassVar, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class BundleCatalogEntry(_message.Message):
    __slots__ = ("bundle", "prices")
    BUNDLE_FIELD_NUMBER: _ClassVar[int]
    PRICES_FIELD_NUMBER: _ClassVar[int]
    bundle: _pricing_pb2.Bundle
    prices: _containers.RepeatedCompositeFieldContainer[_pricing_pb2.PlanOption]
    def __init__(self, bundle: _Optional[_Union[_pricing_pb2.Bundle, _Mapping]] = ..., prices: _Optional[_Iterable[_Union[_pricing_pb2.PlanOption, _Mapping]]] = ...) -> None: ...

class ListBundleCatalogRequest(_message.Message):
    __slots__ = ()
    def __init__(self) -> None: ...

class ListBundleCatalogResponse(_message.Message):
    __slots__ = ("bundles",)
    BUNDLES_FIELD_NUMBER: _ClassVar[int]
    bundles: _containers.RepeatedCompositeFieldContainer[BundleCatalogEntry]
    def __init__(self, bundles: _Optional[_Iterable[_Union[BundleCatalogEntry, _Mapping]]] = ...) -> None: ...

class UpdateBundlePriceRequest(_message.Message):
    __slots__ = ("bundle_key", "price_id", "plan_name", "display_weight", "display_enabled", "subtitle", "badge", "cta_label", "highlight", "features")
    BUNDLE_KEY_FIELD_NUMBER: _ClassVar[int]
    PRICE_ID_FIELD_NUMBER: _ClassVar[int]
    PLAN_NAME_FIELD_NUMBER: _ClassVar[int]
    DISPLAY_WEIGHT_FIELD_NUMBER: _ClassVar[int]
    DISPLAY_ENABLED_FIELD_NUMBER: _ClassVar[int]
    SUBTITLE_FIELD_NUMBER: _ClassVar[int]
    BADGE_FIELD_NUMBER: _ClassVar[int]
    CTA_LABEL_FIELD_NUMBER: _ClassVar[int]
    HIGHLIGHT_FIELD_NUMBER: _ClassVar[int]
    FEATURES_FIELD_NUMBER: _ClassVar[int]
    bundle_key: str
    price_id: str
    plan_name: str
    display_weight: int
    display_enabled: bool
    subtitle: str
    badge: str
    cta_label: str
    highlight: bool
    features: _containers.RepeatedScalarFieldContainer[str]
    def __init__(self, bundle_key: _Optional[str] = ..., price_id: _Optional[str] = ..., plan_name: _Optional[str] = ..., display_weight: _Optional[int] = ..., display_enabled: _Optional[bool] = ..., subtitle: _Optional[str] = ..., badge: _Optional[str] = ..., cta_label: _Optional[str] = ..., highlight: _Optional[bool] = ..., features: _Optional[_Iterable[str]] = ...) -> None: ...

class UpdateBundlePriceResponse(_message.Message):
    __slots__ = ("price",)
    PRICE_FIELD_NUMBER: _ClassVar[int]
    price: _pricing_pb2.PlanOption
    def __init__(self, price: _Optional[_Union[_pricing_pb2.PlanOption, _Mapping]] = ...) -> None: ...
