import datetime

from common.v1 import types_pb2 as _types_pb2
from google.protobuf import timestamp_pb2 as _timestamp_pb2
from landing_page_business_suite.v1.shared import commerce_pb2 as _commerce_pb2
from landing_page_business_suite.v1 import pricing_pb2 as _pricing_pb2
from landing_page_business_suite.v1 import settings_pb2 as _settings_pb2
from google.protobuf.internal import containers as _containers
from google.protobuf.internal import enum_type_wrapper as _enum_type_wrapper
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Iterable as _Iterable, Mapping as _Mapping
from typing import ClassVar as _ClassVar, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class SessionKind(int, metaclass=_enum_type_wrapper.EnumTypeWrapper):
    __slots__ = ()
    SESSION_KIND_UNSPECIFIED: _ClassVar[SessionKind]
    SESSION_KIND_SUBSCRIPTION: _ClassVar[SessionKind]
    SESSION_KIND_CREDITS_TOPUP: _ClassVar[SessionKind]
    SESSION_KIND_SUPPORTER_CONTRIBUTION: _ClassVar[SessionKind]

class CheckoutSessionStatus(int, metaclass=_enum_type_wrapper.EnumTypeWrapper):
    __slots__ = ()
    CHECKOUT_SESSION_STATUS_UNSPECIFIED: _ClassVar[CheckoutSessionStatus]
    CHECKOUT_SESSION_STATUS_OPEN: _ClassVar[CheckoutSessionStatus]
    CHECKOUT_SESSION_STATUS_COMPLETE: _ClassVar[CheckoutSessionStatus]
    CHECKOUT_SESSION_STATUS_EXPIRED: _ClassVar[CheckoutSessionStatus]
    CHECKOUT_SESSION_STATUS_CANCELED: _ClassVar[CheckoutSessionStatus]

class TransactionType(int, metaclass=_enum_type_wrapper.EnumTypeWrapper):
    __slots__ = ()
    TRANSACTION_TYPE_UNSPECIFIED: _ClassVar[TransactionType]
    TRANSACTION_TYPE_CREDIT_TOPUP: _ClassVar[TransactionType]
    TRANSACTION_TYPE_CONSUMPTION: _ClassVar[TransactionType]
    TRANSACTION_TYPE_ADJUSTMENT: _ClassVar[TransactionType]
    TRANSACTION_TYPE_REFUND: _ClassVar[TransactionType]
    TRANSACTION_TYPE_GRANT: _ClassVar[TransactionType]
SESSION_KIND_UNSPECIFIED: SessionKind
SESSION_KIND_SUBSCRIPTION: SessionKind
SESSION_KIND_CREDITS_TOPUP: SessionKind
SESSION_KIND_SUPPORTER_CONTRIBUTION: SessionKind
CHECKOUT_SESSION_STATUS_UNSPECIFIED: CheckoutSessionStatus
CHECKOUT_SESSION_STATUS_OPEN: CheckoutSessionStatus
CHECKOUT_SESSION_STATUS_COMPLETE: CheckoutSessionStatus
CHECKOUT_SESSION_STATUS_EXPIRED: CheckoutSessionStatus
CHECKOUT_SESSION_STATUS_CANCELED: CheckoutSessionStatus
TRANSACTION_TYPE_UNSPECIFIED: TransactionType
TRANSACTION_TYPE_CREDIT_TOPUP: TransactionType
TRANSACTION_TYPE_CONSUMPTION: TransactionType
TRANSACTION_TYPE_ADJUSTMENT: TransactionType
TRANSACTION_TYPE_REFUND: TransactionType
TRANSACTION_TYPE_GRANT: TransactionType

class CheckoutSession(_message.Message):
    __slots__ = ("session_id", "session_kind", "status", "url", "publishable_key", "customer_email", "stripe_price_id", "stripe_product_id", "subscription_id", "schedule_id", "amount_cents", "currency", "success_url", "cancel_url", "created_at", "metadata")
    class MetadataEntry(_message.Message):
        __slots__ = ("key", "value")
        KEY_FIELD_NUMBER: _ClassVar[int]
        VALUE_FIELD_NUMBER: _ClassVar[int]
        key: str
        value: _types_pb2.JsonValue
        def __init__(self, key: _Optional[str] = ..., value: _Optional[_Union[_types_pb2.JsonValue, _Mapping]] = ...) -> None: ...
    SESSION_ID_FIELD_NUMBER: _ClassVar[int]
    SESSION_KIND_FIELD_NUMBER: _ClassVar[int]
    STATUS_FIELD_NUMBER: _ClassVar[int]
    URL_FIELD_NUMBER: _ClassVar[int]
    PUBLISHABLE_KEY_FIELD_NUMBER: _ClassVar[int]
    CUSTOMER_EMAIL_FIELD_NUMBER: _ClassVar[int]
    STRIPE_PRICE_ID_FIELD_NUMBER: _ClassVar[int]
    STRIPE_PRODUCT_ID_FIELD_NUMBER: _ClassVar[int]
    SUBSCRIPTION_ID_FIELD_NUMBER: _ClassVar[int]
    SCHEDULE_ID_FIELD_NUMBER: _ClassVar[int]
    AMOUNT_CENTS_FIELD_NUMBER: _ClassVar[int]
    CURRENCY_FIELD_NUMBER: _ClassVar[int]
    SUCCESS_URL_FIELD_NUMBER: _ClassVar[int]
    CANCEL_URL_FIELD_NUMBER: _ClassVar[int]
    CREATED_AT_FIELD_NUMBER: _ClassVar[int]
    METADATA_FIELD_NUMBER: _ClassVar[int]
    session_id: str
    session_kind: SessionKind
    status: CheckoutSessionStatus
    url: str
    publishable_key: str
    customer_email: str
    stripe_price_id: str
    stripe_product_id: str
    subscription_id: str
    schedule_id: str
    amount_cents: int
    currency: str
    success_url: str
    cancel_url: str
    created_at: _timestamp_pb2.Timestamp
    metadata: _containers.MessageMap[str, _types_pb2.JsonValue]
    def __init__(self, session_id: _Optional[str] = ..., session_kind: _Optional[_Union[SessionKind, str]] = ..., status: _Optional[_Union[CheckoutSessionStatus, str]] = ..., url: _Optional[str] = ..., publishable_key: _Optional[str] = ..., customer_email: _Optional[str] = ..., stripe_price_id: _Optional[str] = ..., stripe_product_id: _Optional[str] = ..., subscription_id: _Optional[str] = ..., schedule_id: _Optional[str] = ..., amount_cents: _Optional[int] = ..., currency: _Optional[str] = ..., success_url: _Optional[str] = ..., cancel_url: _Optional[str] = ..., created_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., metadata: _Optional[_Mapping[str, _types_pb2.JsonValue]] = ...) -> None: ...

class CreateCheckoutSessionRequest(_message.Message):
    __slots__ = ("price_id", "customer_email", "success_url", "cancel_url", "session_kind", "metadata")
    class MetadataEntry(_message.Message):
        __slots__ = ("key", "value")
        KEY_FIELD_NUMBER: _ClassVar[int]
        VALUE_FIELD_NUMBER: _ClassVar[int]
        key: str
        value: _types_pb2.JsonValue
        def __init__(self, key: _Optional[str] = ..., value: _Optional[_Union[_types_pb2.JsonValue, _Mapping]] = ...) -> None: ...
    PRICE_ID_FIELD_NUMBER: _ClassVar[int]
    CUSTOMER_EMAIL_FIELD_NUMBER: _ClassVar[int]
    SUCCESS_URL_FIELD_NUMBER: _ClassVar[int]
    CANCEL_URL_FIELD_NUMBER: _ClassVar[int]
    SESSION_KIND_FIELD_NUMBER: _ClassVar[int]
    METADATA_FIELD_NUMBER: _ClassVar[int]
    price_id: str
    customer_email: str
    success_url: str
    cancel_url: str
    session_kind: SessionKind
    metadata: _containers.MessageMap[str, _types_pb2.JsonValue]
    def __init__(self, price_id: _Optional[str] = ..., customer_email: _Optional[str] = ..., success_url: _Optional[str] = ..., cancel_url: _Optional[str] = ..., session_kind: _Optional[_Union[SessionKind, str]] = ..., metadata: _Optional[_Mapping[str, _types_pb2.JsonValue]] = ...) -> None: ...

class CreateCheckoutSessionResponse(_message.Message):
    __slots__ = ("session",)
    SESSION_FIELD_NUMBER: _ClassVar[int]
    session: CheckoutSession
    def __init__(self, session: _Optional[_Union[CheckoutSession, _Mapping]] = ...) -> None: ...

class VerifySubscriptionRequest(_message.Message):
    __slots__ = ("user_identity",)
    USER_IDENTITY_FIELD_NUMBER: _ClassVar[int]
    user_identity: str
    def __init__(self, user_identity: _Optional[str] = ...) -> None: ...

class CancelSubscriptionRequest(_message.Message):
    __slots__ = ("user_identity",)
    USER_IDENTITY_FIELD_NUMBER: _ClassVar[int]
    user_identity: str
    def __init__(self, user_identity: _Optional[str] = ...) -> None: ...

class CancelSubscriptionResponse(_message.Message):
    __slots__ = ("subscription_id", "state", "canceled_at", "message")
    SUBSCRIPTION_ID_FIELD_NUMBER: _ClassVar[int]
    STATE_FIELD_NUMBER: _ClassVar[int]
    CANCELED_AT_FIELD_NUMBER: _ClassVar[int]
    MESSAGE_FIELD_NUMBER: _ClassVar[int]
    subscription_id: str
    state: _commerce_pb2.SubscriptionState
    canceled_at: _timestamp_pb2.Timestamp
    message: str
    def __init__(self, subscription_id: _Optional[str] = ..., state: _Optional[_Union[_commerce_pb2.SubscriptionState, str]] = ..., canceled_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., message: _Optional[str] = ...) -> None: ...

class CreditTransaction(_message.Message):
    __slots__ = ("id", "customer_email", "amount_credits", "type", "metadata", "created_at")
    class MetadataEntry(_message.Message):
        __slots__ = ("key", "value")
        KEY_FIELD_NUMBER: _ClassVar[int]
        VALUE_FIELD_NUMBER: _ClassVar[int]
        key: str
        value: _types_pb2.JsonValue
        def __init__(self, key: _Optional[str] = ..., value: _Optional[_Union[_types_pb2.JsonValue, _Mapping]] = ...) -> None: ...
    ID_FIELD_NUMBER: _ClassVar[int]
    CUSTOMER_EMAIL_FIELD_NUMBER: _ClassVar[int]
    AMOUNT_CREDITS_FIELD_NUMBER: _ClassVar[int]
    TYPE_FIELD_NUMBER: _ClassVar[int]
    METADATA_FIELD_NUMBER: _ClassVar[int]
    CREATED_AT_FIELD_NUMBER: _ClassVar[int]
    id: str
    customer_email: str
    amount_credits: int
    type: TransactionType
    metadata: _containers.MessageMap[str, _types_pb2.JsonValue]
    created_at: _timestamp_pb2.Timestamp
    def __init__(self, id: _Optional[str] = ..., customer_email: _Optional[str] = ..., amount_credits: _Optional[int] = ..., type: _Optional[_Union[TransactionType, str]] = ..., metadata: _Optional[_Mapping[str, _types_pb2.JsonValue]] = ..., created_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ...) -> None: ...

class GetCreditsResponse(_message.Message):
    __slots__ = ("balance", "transactions")
    BALANCE_FIELD_NUMBER: _ClassVar[int]
    TRANSACTIONS_FIELD_NUMBER: _ClassVar[int]
    balance: _commerce_pb2.CreditsBalance
    transactions: _containers.RepeatedCompositeFieldContainer[CreditTransaction]
    def __init__(self, balance: _Optional[_Union[_commerce_pb2.CreditsBalance, _Mapping]] = ..., transactions: _Optional[_Iterable[_Union[CreditTransaction, _Mapping]]] = ...) -> None: ...

class BillingPortalResponse(_message.Message):
    __slots__ = ("url",)
    URL_FIELD_NUMBER: _ClassVar[int]
    url: str
    def __init__(self, url: _Optional[str] = ...) -> None: ...

class GetBillingPortalRequest(_message.Message):
    __slots__ = ()
    def __init__(self) -> None: ...
