from google.protobuf.internal import containers as _containers
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Iterable as _Iterable, Mapping as _Mapping
from typing import ClassVar as _ClassVar, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class PriceObservation(_message.Message):
    __slots__ = ("id", "workspace_id", "item_id", "product_id", "package_label", "package_amount", "package_unit", "price_minor", "currency", "currency_exponent", "retailer", "observed_at", "valid_through", "available", "membership_required", "coupon_required", "minimum_buy", "source")
    ID_FIELD_NUMBER: _ClassVar[int]
    WORKSPACE_ID_FIELD_NUMBER: _ClassVar[int]
    ITEM_ID_FIELD_NUMBER: _ClassVar[int]
    PRODUCT_ID_FIELD_NUMBER: _ClassVar[int]
    PACKAGE_LABEL_FIELD_NUMBER: _ClassVar[int]
    PACKAGE_AMOUNT_FIELD_NUMBER: _ClassVar[int]
    PACKAGE_UNIT_FIELD_NUMBER: _ClassVar[int]
    PRICE_MINOR_FIELD_NUMBER: _ClassVar[int]
    CURRENCY_FIELD_NUMBER: _ClassVar[int]
    CURRENCY_EXPONENT_FIELD_NUMBER: _ClassVar[int]
    RETAILER_FIELD_NUMBER: _ClassVar[int]
    OBSERVED_AT_FIELD_NUMBER: _ClassVar[int]
    VALID_THROUGH_FIELD_NUMBER: _ClassVar[int]
    AVAILABLE_FIELD_NUMBER: _ClassVar[int]
    MEMBERSHIP_REQUIRED_FIELD_NUMBER: _ClassVar[int]
    COUPON_REQUIRED_FIELD_NUMBER: _ClassVar[int]
    MINIMUM_BUY_FIELD_NUMBER: _ClassVar[int]
    SOURCE_FIELD_NUMBER: _ClassVar[int]
    id: str
    workspace_id: str
    item_id: str
    product_id: str
    package_label: str
    package_amount: str
    package_unit: str
    price_minor: int
    currency: str
    currency_exponent: int
    retailer: str
    observed_at: str
    valid_through: str
    available: bool
    membership_required: bool
    coupon_required: bool
    minimum_buy: str
    source: str
    def __init__(self, id: _Optional[str] = ..., workspace_id: _Optional[str] = ..., item_id: _Optional[str] = ..., product_id: _Optional[str] = ..., package_label: _Optional[str] = ..., package_amount: _Optional[str] = ..., package_unit: _Optional[str] = ..., price_minor: _Optional[int] = ..., currency: _Optional[str] = ..., currency_exponent: _Optional[int] = ..., retailer: _Optional[str] = ..., observed_at: _Optional[str] = ..., valid_through: _Optional[str] = ..., available: _Optional[bool] = ..., membership_required: _Optional[bool] = ..., coupon_required: _Optional[bool] = ..., minimum_buy: _Optional[str] = ..., source: _Optional[str] = ...) -> None: ...

class ListPriceObservationsRequest(_message.Message):
    __slots__ = ("workspace_id", "item_id")
    WORKSPACE_ID_FIELD_NUMBER: _ClassVar[int]
    ITEM_ID_FIELD_NUMBER: _ClassVar[int]
    workspace_id: str
    item_id: str
    def __init__(self, workspace_id: _Optional[str] = ..., item_id: _Optional[str] = ...) -> None: ...

class ListPriceObservationsResponse(_message.Message):
    __slots__ = ("observations",)
    OBSERVATIONS_FIELD_NUMBER: _ClassVar[int]
    observations: _containers.RepeatedCompositeFieldContainer[PriceObservation]
    def __init__(self, observations: _Optional[_Iterable[_Union[PriceObservation, _Mapping]]] = ...) -> None: ...

class CreatePriceObservationRequest(_message.Message):
    __slots__ = ("workspace_id", "item_id", "product_id", "package_label", "package_amount", "package_unit", "price_minor", "currency", "currency_exponent", "retailer", "observed_at", "valid_through", "available", "membership_required", "coupon_required", "minimum_buy", "source")
    WORKSPACE_ID_FIELD_NUMBER: _ClassVar[int]
    ITEM_ID_FIELD_NUMBER: _ClassVar[int]
    PRODUCT_ID_FIELD_NUMBER: _ClassVar[int]
    PACKAGE_LABEL_FIELD_NUMBER: _ClassVar[int]
    PACKAGE_AMOUNT_FIELD_NUMBER: _ClassVar[int]
    PACKAGE_UNIT_FIELD_NUMBER: _ClassVar[int]
    PRICE_MINOR_FIELD_NUMBER: _ClassVar[int]
    CURRENCY_FIELD_NUMBER: _ClassVar[int]
    CURRENCY_EXPONENT_FIELD_NUMBER: _ClassVar[int]
    RETAILER_FIELD_NUMBER: _ClassVar[int]
    OBSERVED_AT_FIELD_NUMBER: _ClassVar[int]
    VALID_THROUGH_FIELD_NUMBER: _ClassVar[int]
    AVAILABLE_FIELD_NUMBER: _ClassVar[int]
    MEMBERSHIP_REQUIRED_FIELD_NUMBER: _ClassVar[int]
    COUPON_REQUIRED_FIELD_NUMBER: _ClassVar[int]
    MINIMUM_BUY_FIELD_NUMBER: _ClassVar[int]
    SOURCE_FIELD_NUMBER: _ClassVar[int]
    workspace_id: str
    item_id: str
    product_id: str
    package_label: str
    package_amount: str
    package_unit: str
    price_minor: int
    currency: str
    currency_exponent: int
    retailer: str
    observed_at: str
    valid_through: str
    available: bool
    membership_required: bool
    coupon_required: bool
    minimum_buy: str
    source: str
    def __init__(self, workspace_id: _Optional[str] = ..., item_id: _Optional[str] = ..., product_id: _Optional[str] = ..., package_label: _Optional[str] = ..., package_amount: _Optional[str] = ..., package_unit: _Optional[str] = ..., price_minor: _Optional[int] = ..., currency: _Optional[str] = ..., currency_exponent: _Optional[int] = ..., retailer: _Optional[str] = ..., observed_at: _Optional[str] = ..., valid_through: _Optional[str] = ..., available: _Optional[bool] = ..., membership_required: _Optional[bool] = ..., coupon_required: _Optional[bool] = ..., minimum_buy: _Optional[str] = ..., source: _Optional[str] = ...) -> None: ...

class CreatePriceObservationResponse(_message.Message):
    __slots__ = ("observation",)
    OBSERVATION_FIELD_NUMBER: _ClassVar[int]
    observation: PriceObservation
    def __init__(self, observation: _Optional[_Union[PriceObservation, _Mapping]] = ...) -> None: ...
