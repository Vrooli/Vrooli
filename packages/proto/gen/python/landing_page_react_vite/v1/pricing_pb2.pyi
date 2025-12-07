import datetime

from google.protobuf import struct_pb2 as _struct_pb2
from google.protobuf import timestamp_pb2 as _timestamp_pb2
from google.protobuf.internal import containers as _containers
from google.protobuf.internal import enum_type_wrapper as _enum_type_wrapper
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Iterable as _Iterable, Mapping as _Mapping
from typing import ClassVar as _ClassVar, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class PlanKind(int, metaclass=_enum_type_wrapper.EnumTypeWrapper):
    __slots__ = ()
    PLAN_KIND_UNSPECIFIED: _ClassVar[PlanKind]
    PLAN_KIND_SUBSCRIPTION: _ClassVar[PlanKind]
    PLAN_KIND_CREDITS_TOPUP: _ClassVar[PlanKind]
    PLAN_KIND_SUPPORTER_CONTRIBUTION: _ClassVar[PlanKind]

class BillingInterval(int, metaclass=_enum_type_wrapper.EnumTypeWrapper):
    __slots__ = ()
    BILLING_INTERVAL_UNSPECIFIED: _ClassVar[BillingInterval]
    BILLING_INTERVAL_MONTH: _ClassVar[BillingInterval]
    BILLING_INTERVAL_YEAR: _ClassVar[BillingInterval]
    BILLING_INTERVAL_ONE_TIME: _ClassVar[BillingInterval]

class IntroPricingType(int, metaclass=_enum_type_wrapper.EnumTypeWrapper):
    __slots__ = ()
    INTRO_PRICING_TYPE_UNSPECIFIED: _ClassVar[IntroPricingType]
    INTRO_PRICING_TYPE_FLAT_AMOUNT: _ClassVar[IntroPricingType]
    INTRO_PRICING_TYPE_PERCENTAGE: _ClassVar[IntroPricingType]
PLAN_KIND_UNSPECIFIED: PlanKind
PLAN_KIND_SUBSCRIPTION: PlanKind
PLAN_KIND_CREDITS_TOPUP: PlanKind
PLAN_KIND_SUPPORTER_CONTRIBUTION: PlanKind
BILLING_INTERVAL_UNSPECIFIED: BillingInterval
BILLING_INTERVAL_MONTH: BillingInterval
BILLING_INTERVAL_YEAR: BillingInterval
BILLING_INTERVAL_ONE_TIME: BillingInterval
INTRO_PRICING_TYPE_UNSPECIFIED: IntroPricingType
INTRO_PRICING_TYPE_FLAT_AMOUNT: IntroPricingType
INTRO_PRICING_TYPE_PERCENTAGE: IntroPricingType

class Bundle(_message.Message):
    __slots__ = ()
    class MetadataEntry(_message.Message):
        __slots__ = ()
        KEY_FIELD_NUMBER: _ClassVar[int]
        VALUE_FIELD_NUMBER: _ClassVar[int]
        key: str
        value: _struct_pb2.Value
        def __init__(self, key: _Optional[str] = ..., value: _Optional[_Union[_struct_pb2.Value, _Mapping]] = ...) -> None: ...
    BUNDLE_KEY_FIELD_NUMBER: _ClassVar[int]
    NAME_FIELD_NUMBER: _ClassVar[int]
    STRIPE_PRODUCT_ID_FIELD_NUMBER: _ClassVar[int]
    CREDITS_PER_USD_FIELD_NUMBER: _ClassVar[int]
    DISPLAY_CREDITS_MULTIPLIER_FIELD_NUMBER: _ClassVar[int]
    DISPLAY_CREDITS_LABEL_FIELD_NUMBER: _ClassVar[int]
    ENVIRONMENT_FIELD_NUMBER: _ClassVar[int]
    METADATA_FIELD_NUMBER: _ClassVar[int]
    bundle_key: str
    name: str
    stripe_product_id: str
    credits_per_usd: int
    display_credits_multiplier: float
    display_credits_label: str
    environment: str
    metadata: _containers.MessageMap[str, _struct_pb2.Value]
    def __init__(self, bundle_key: _Optional[str] = ..., name: _Optional[str] = ..., stripe_product_id: _Optional[str] = ..., credits_per_usd: _Optional[int] = ..., display_credits_multiplier: _Optional[float] = ..., display_credits_label: _Optional[str] = ..., environment: _Optional[str] = ..., metadata: _Optional[_Mapping[str, _struct_pb2.Value]] = ...) -> None: ...

class PlanOption(_message.Message):
    __slots__ = ()
    class MetadataEntry(_message.Message):
        __slots__ = ()
        KEY_FIELD_NUMBER: _ClassVar[int]
        VALUE_FIELD_NUMBER: _ClassVar[int]
        key: str
        value: _struct_pb2.Value
        def __init__(self, key: _Optional[str] = ..., value: _Optional[_Union[_struct_pb2.Value, _Mapping]] = ...) -> None: ...
    PLAN_NAME_FIELD_NUMBER: _ClassVar[int]
    PLAN_TIER_FIELD_NUMBER: _ClassVar[int]
    BILLING_INTERVAL_FIELD_NUMBER: _ClassVar[int]
    AMOUNT_CENTS_FIELD_NUMBER: _ClassVar[int]
    CURRENCY_FIELD_NUMBER: _ClassVar[int]
    INTRO_ENABLED_FIELD_NUMBER: _ClassVar[int]
    INTRO_TYPE_FIELD_NUMBER: _ClassVar[int]
    INTRO_AMOUNT_CENTS_FIELD_NUMBER: _ClassVar[int]
    INTRO_PERIODS_FIELD_NUMBER: _ClassVar[int]
    INTRO_PRICE_LOOKUP_KEY_FIELD_NUMBER: _ClassVar[int]
    STRIPE_PRICE_ID_FIELD_NUMBER: _ClassVar[int]
    MONTHLY_INCLUDED_CREDITS_FIELD_NUMBER: _ClassVar[int]
    ONE_TIME_BONUS_CREDITS_FIELD_NUMBER: _ClassVar[int]
    PLAN_RANK_FIELD_NUMBER: _ClassVar[int]
    BONUS_TYPE_FIELD_NUMBER: _ClassVar[int]
    KIND_FIELD_NUMBER: _ClassVar[int]
    IS_VARIABLE_AMOUNT_FIELD_NUMBER: _ClassVar[int]
    DISPLAY_ENABLED_FIELD_NUMBER: _ClassVar[int]
    BUNDLE_KEY_FIELD_NUMBER: _ClassVar[int]
    DISPLAY_WEIGHT_FIELD_NUMBER: _ClassVar[int]
    METADATA_FIELD_NUMBER: _ClassVar[int]
    plan_name: str
    plan_tier: str
    billing_interval: BillingInterval
    amount_cents: int
    currency: str
    intro_enabled: bool
    intro_type: IntroPricingType
    intro_amount_cents: int
    intro_periods: int
    intro_price_lookup_key: str
    stripe_price_id: str
    monthly_included_credits: int
    one_time_bonus_credits: int
    plan_rank: int
    bonus_type: str
    kind: PlanKind
    is_variable_amount: bool
    display_enabled: bool
    bundle_key: str
    display_weight: int
    metadata: _containers.MessageMap[str, _struct_pb2.Value]
    def __init__(self, plan_name: _Optional[str] = ..., plan_tier: _Optional[str] = ..., billing_interval: _Optional[_Union[BillingInterval, str]] = ..., amount_cents: _Optional[int] = ..., currency: _Optional[str] = ..., intro_enabled: _Optional[bool] = ..., intro_type: _Optional[_Union[IntroPricingType, str]] = ..., intro_amount_cents: _Optional[int] = ..., intro_periods: _Optional[int] = ..., intro_price_lookup_key: _Optional[str] = ..., stripe_price_id: _Optional[str] = ..., monthly_included_credits: _Optional[int] = ..., one_time_bonus_credits: _Optional[int] = ..., plan_rank: _Optional[int] = ..., bonus_type: _Optional[str] = ..., kind: _Optional[_Union[PlanKind, str]] = ..., is_variable_amount: _Optional[bool] = ..., display_enabled: _Optional[bool] = ..., bundle_key: _Optional[str] = ..., display_weight: _Optional[int] = ..., metadata: _Optional[_Mapping[str, _struct_pb2.Value]] = ...) -> None: ...

class PricingOverview(_message.Message):
    __slots__ = ()
    BUNDLE_FIELD_NUMBER: _ClassVar[int]
    MONTHLY_FIELD_NUMBER: _ClassVar[int]
    YEARLY_FIELD_NUMBER: _ClassVar[int]
    UPDATED_AT_FIELD_NUMBER: _ClassVar[int]
    bundle: Bundle
    monthly: _containers.RepeatedCompositeFieldContainer[PlanOption]
    yearly: _containers.RepeatedCompositeFieldContainer[PlanOption]
    updated_at: _timestamp_pb2.Timestamp
    def __init__(self, bundle: _Optional[_Union[Bundle, _Mapping]] = ..., monthly: _Optional[_Iterable[_Union[PlanOption, _Mapping]]] = ..., yearly: _Optional[_Iterable[_Union[PlanOption, _Mapping]]] = ..., updated_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ...) -> None: ...

class GetPricingRequest(_message.Message):
    __slots__ = ()
    BUNDLE_KEY_FIELD_NUMBER: _ClassVar[int]
    INCLUDE_HIDDEN_FIELD_NUMBER: _ClassVar[int]
    bundle_key: str
    include_hidden: bool
    def __init__(self, bundle_key: _Optional[str] = ..., include_hidden: _Optional[bool] = ...) -> None: ...

class GetPricingResponse(_message.Message):
    __slots__ = ()
    PRICING_FIELD_NUMBER: _ClassVar[int]
    pricing: PricingOverview
    def __init__(self, pricing: _Optional[_Union[PricingOverview, _Mapping]] = ...) -> None: ...
