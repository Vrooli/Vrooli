from landing_page_business_suite.v1.shared import commerce_pb2 as _commerce_pb2
from google.protobuf.internal import containers as _containers
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Iterable as _Iterable, Mapping as _Mapping
from typing import ClassVar as _ClassVar, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class GetMySubscriptionRequest(_message.Message):
    __slots__ = ()
    def __init__(self) -> None: ...

class GetMyCreditsRequest(_message.Message):
    __slots__ = ()
    def __init__(self) -> None: ...

class GetMyCreditsResponse(_message.Message):
    __slots__ = ("balance", "display_credits_label", "display_credits_multiplier")
    BALANCE_FIELD_NUMBER: _ClassVar[int]
    DISPLAY_CREDITS_LABEL_FIELD_NUMBER: _ClassVar[int]
    DISPLAY_CREDITS_MULTIPLIER_FIELD_NUMBER: _ClassVar[int]
    balance: _commerce_pb2.CreditsBalance
    display_credits_label: str
    display_credits_multiplier: float
    def __init__(self, balance: _Optional[_Union[_commerce_pb2.CreditsBalance, _Mapping]] = ..., display_credits_label: _Optional[str] = ..., display_credits_multiplier: _Optional[float] = ...) -> None: ...

class GetEntitlementsRequest(_message.Message):
    __slots__ = ()
    def __init__(self) -> None: ...

class GetEntitlementsResponse(_message.Message):
    __slots__ = ("status", "plan_tier", "price_id", "features", "credits", "subscription", "billing_cycle_start")
    STATUS_FIELD_NUMBER: _ClassVar[int]
    PLAN_TIER_FIELD_NUMBER: _ClassVar[int]
    PRICE_ID_FIELD_NUMBER: _ClassVar[int]
    FEATURES_FIELD_NUMBER: _ClassVar[int]
    CREDITS_FIELD_NUMBER: _ClassVar[int]
    SUBSCRIPTION_FIELD_NUMBER: _ClassVar[int]
    BILLING_CYCLE_START_FIELD_NUMBER: _ClassVar[int]
    status: str
    plan_tier: str
    price_id: str
    features: _containers.RepeatedScalarFieldContainer[str]
    credits: _commerce_pb2.CreditsBalance
    subscription: _commerce_pb2.SubscriptionStatus
    billing_cycle_start: int
    def __init__(self, status: _Optional[str] = ..., plan_tier: _Optional[str] = ..., price_id: _Optional[str] = ..., features: _Optional[_Iterable[str]] = ..., credits: _Optional[_Union[_commerce_pb2.CreditsBalance, _Mapping]] = ..., subscription: _Optional[_Union[_commerce_pb2.SubscriptionStatus, _Mapping]] = ..., billing_cycle_start: _Optional[int] = ...) -> None: ...
