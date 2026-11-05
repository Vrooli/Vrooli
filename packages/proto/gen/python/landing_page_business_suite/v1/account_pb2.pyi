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

class CommercialContextRequest(_message.Message):
    __slots__ = ("placement", "capability_id")
    PLACEMENT_FIELD_NUMBER: _ClassVar[int]
    CAPABILITY_ID_FIELD_NUMBER: _ClassVar[int]
    placement: str
    capability_id: str
    def __init__(self, placement: _Optional[str] = ..., capability_id: _Optional[str] = ...) -> None: ...

class CommercialAccountFacts(_message.Message):
    __slots__ = ("subscription_status", "plan_tier", "credit_balance", "entitlement_ids", "evaluated_at")
    SUBSCRIPTION_STATUS_FIELD_NUMBER: _ClassVar[int]
    PLAN_TIER_FIELD_NUMBER: _ClassVar[int]
    CREDIT_BALANCE_FIELD_NUMBER: _ClassVar[int]
    ENTITLEMENT_IDS_FIELD_NUMBER: _ClassVar[int]
    EVALUATED_AT_FIELD_NUMBER: _ClassVar[int]
    subscription_status: str
    plan_tier: str
    credit_balance: int
    entitlement_ids: _containers.RepeatedScalarFieldContainer[str]
    evaluated_at: str
    def __init__(self, subscription_status: _Optional[str] = ..., plan_tier: _Optional[str] = ..., credit_balance: _Optional[int] = ..., entitlement_ids: _Optional[_Iterable[str]] = ..., evaluated_at: _Optional[str] = ...) -> None: ...

class CommercialContent(_message.Message):
    __slots__ = ("content_id", "placement", "title", "description", "priority", "eligible", "cta_label", "cta_destination", "expires_at", "dismissible", "dismissed_until")
    CONTENT_ID_FIELD_NUMBER: _ClassVar[int]
    PLACEMENT_FIELD_NUMBER: _ClassVar[int]
    TITLE_FIELD_NUMBER: _ClassVar[int]
    DESCRIPTION_FIELD_NUMBER: _ClassVar[int]
    PRIORITY_FIELD_NUMBER: _ClassVar[int]
    ELIGIBLE_FIELD_NUMBER: _ClassVar[int]
    CTA_LABEL_FIELD_NUMBER: _ClassVar[int]
    CTA_DESTINATION_FIELD_NUMBER: _ClassVar[int]
    EXPIRES_AT_FIELD_NUMBER: _ClassVar[int]
    DISMISSIBLE_FIELD_NUMBER: _ClassVar[int]
    DISMISSED_UNTIL_FIELD_NUMBER: _ClassVar[int]
    content_id: str
    placement: str
    title: str
    description: str
    priority: str
    eligible: bool
    cta_label: str
    cta_destination: str
    expires_at: str
    dismissible: bool
    dismissed_until: str
    def __init__(self, content_id: _Optional[str] = ..., placement: _Optional[str] = ..., title: _Optional[str] = ..., description: _Optional[str] = ..., priority: _Optional[str] = ..., eligible: _Optional[bool] = ..., cta_label: _Optional[str] = ..., cta_destination: _Optional[str] = ..., expires_at: _Optional[str] = ..., dismissible: _Optional[bool] = ..., dismissed_until: _Optional[str] = ...) -> None: ...

class CommercialContextResponse(_message.Message):
    __slots__ = ("account", "content", "generated_at", "stale_after", "source")
    ACCOUNT_FIELD_NUMBER: _ClassVar[int]
    CONTENT_FIELD_NUMBER: _ClassVar[int]
    GENERATED_AT_FIELD_NUMBER: _ClassVar[int]
    STALE_AFTER_FIELD_NUMBER: _ClassVar[int]
    SOURCE_FIELD_NUMBER: _ClassVar[int]
    account: CommercialAccountFacts
    content: _containers.RepeatedCompositeFieldContainer[CommercialContent]
    generated_at: str
    stale_after: str
    source: str
    def __init__(self, account: _Optional[_Union[CommercialAccountFacts, _Mapping]] = ..., content: _Optional[_Iterable[_Union[CommercialContent, _Mapping]]] = ..., generated_at: _Optional[str] = ..., stale_after: _Optional[str] = ..., source: _Optional[str] = ...) -> None: ...
