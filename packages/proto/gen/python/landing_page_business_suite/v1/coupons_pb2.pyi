from google.protobuf.internal import containers as _containers
from google.protobuf.internal import enum_type_wrapper as _enum_type_wrapper
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Iterable as _Iterable, Mapping as _Mapping
from typing import ClassVar as _ClassVar, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class CouponDuration(int, metaclass=_enum_type_wrapper.EnumTypeWrapper):
    __slots__ = ()
    COUPON_DURATION_UNSPECIFIED: _ClassVar[CouponDuration]
    COUPON_DURATION_ONCE: _ClassVar[CouponDuration]
    COUPON_DURATION_REPEATING: _ClassVar[CouponDuration]
    COUPON_DURATION_FOREVER: _ClassVar[CouponDuration]
COUPON_DURATION_UNSPECIFIED: CouponDuration
COUPON_DURATION_ONCE: CouponDuration
COUPON_DURATION_REPEATING: CouponDuration
COUPON_DURATION_FOREVER: CouponDuration

class Coupon(_message.Message):
    __slots__ = ("id", "name", "amount_off", "percent_off", "currency", "duration", "duration_in_months", "max_redemptions", "redeem_by", "times_redeemed", "valid", "created", "is_intro_coupon", "intro_tier")
    ID_FIELD_NUMBER: _ClassVar[int]
    NAME_FIELD_NUMBER: _ClassVar[int]
    AMOUNT_OFF_FIELD_NUMBER: _ClassVar[int]
    PERCENT_OFF_FIELD_NUMBER: _ClassVar[int]
    CURRENCY_FIELD_NUMBER: _ClassVar[int]
    DURATION_FIELD_NUMBER: _ClassVar[int]
    DURATION_IN_MONTHS_FIELD_NUMBER: _ClassVar[int]
    MAX_REDEMPTIONS_FIELD_NUMBER: _ClassVar[int]
    REDEEM_BY_FIELD_NUMBER: _ClassVar[int]
    TIMES_REDEEMED_FIELD_NUMBER: _ClassVar[int]
    VALID_FIELD_NUMBER: _ClassVar[int]
    CREATED_FIELD_NUMBER: _ClassVar[int]
    IS_INTRO_COUPON_FIELD_NUMBER: _ClassVar[int]
    INTRO_TIER_FIELD_NUMBER: _ClassVar[int]
    id: str
    name: str
    amount_off: int
    percent_off: float
    currency: str
    duration: CouponDuration
    duration_in_months: int
    max_redemptions: int
    redeem_by: int
    times_redeemed: int
    valid: bool
    created: int
    is_intro_coupon: bool
    intro_tier: str
    def __init__(self, id: _Optional[str] = ..., name: _Optional[str] = ..., amount_off: _Optional[int] = ..., percent_off: _Optional[float] = ..., currency: _Optional[str] = ..., duration: _Optional[_Union[CouponDuration, str]] = ..., duration_in_months: _Optional[int] = ..., max_redemptions: _Optional[int] = ..., redeem_by: _Optional[int] = ..., times_redeemed: _Optional[int] = ..., valid: _Optional[bool] = ..., created: _Optional[int] = ..., is_intro_coupon: _Optional[bool] = ..., intro_tier: _Optional[str] = ...) -> None: ...

class ListCouponsRequest(_message.Message):
    __slots__ = ()
    def __init__(self) -> None: ...

class ListCouponsResponse(_message.Message):
    __slots__ = ("coupons", "intro_coupon_map")
    class IntroCouponMapEntry(_message.Message):
        __slots__ = ("key", "value")
        KEY_FIELD_NUMBER: _ClassVar[int]
        VALUE_FIELD_NUMBER: _ClassVar[int]
        key: str
        value: str
        def __init__(self, key: _Optional[str] = ..., value: _Optional[str] = ...) -> None: ...
    COUPONS_FIELD_NUMBER: _ClassVar[int]
    INTRO_COUPON_MAP_FIELD_NUMBER: _ClassVar[int]
    coupons: _containers.RepeatedCompositeFieldContainer[Coupon]
    intro_coupon_map: _containers.ScalarMap[str, str]
    def __init__(self, coupons: _Optional[_Iterable[_Union[Coupon, _Mapping]]] = ..., intro_coupon_map: _Optional[_Mapping[str, str]] = ...) -> None: ...

class CreateCouponRequest(_message.Message):
    __slots__ = ("id", "name", "amount_off", "percent_off", "currency", "duration", "duration_in_months", "max_redemptions", "redeem_by")
    ID_FIELD_NUMBER: _ClassVar[int]
    NAME_FIELD_NUMBER: _ClassVar[int]
    AMOUNT_OFF_FIELD_NUMBER: _ClassVar[int]
    PERCENT_OFF_FIELD_NUMBER: _ClassVar[int]
    CURRENCY_FIELD_NUMBER: _ClassVar[int]
    DURATION_FIELD_NUMBER: _ClassVar[int]
    DURATION_IN_MONTHS_FIELD_NUMBER: _ClassVar[int]
    MAX_REDEMPTIONS_FIELD_NUMBER: _ClassVar[int]
    REDEEM_BY_FIELD_NUMBER: _ClassVar[int]
    id: str
    name: str
    amount_off: int
    percent_off: float
    currency: str
    duration: CouponDuration
    duration_in_months: int
    max_redemptions: int
    redeem_by: int
    def __init__(self, id: _Optional[str] = ..., name: _Optional[str] = ..., amount_off: _Optional[int] = ..., percent_off: _Optional[float] = ..., currency: _Optional[str] = ..., duration: _Optional[_Union[CouponDuration, str]] = ..., duration_in_months: _Optional[int] = ..., max_redemptions: _Optional[int] = ..., redeem_by: _Optional[int] = ...) -> None: ...

class CreateCouponResponse(_message.Message):
    __slots__ = ("coupon",)
    COUPON_FIELD_NUMBER: _ClassVar[int]
    coupon: Coupon
    def __init__(self, coupon: _Optional[_Union[Coupon, _Mapping]] = ...) -> None: ...

class GetCouponRequest(_message.Message):
    __slots__ = ("coupon_id",)
    COUPON_ID_FIELD_NUMBER: _ClassVar[int]
    coupon_id: str
    def __init__(self, coupon_id: _Optional[str] = ...) -> None: ...

class GetCouponResponse(_message.Message):
    __slots__ = ("coupon",)
    COUPON_FIELD_NUMBER: _ClassVar[int]
    coupon: Coupon
    def __init__(self, coupon: _Optional[_Union[Coupon, _Mapping]] = ...) -> None: ...

class UpdateCouponRequest(_message.Message):
    __slots__ = ("coupon_id", "name")
    COUPON_ID_FIELD_NUMBER: _ClassVar[int]
    NAME_FIELD_NUMBER: _ClassVar[int]
    coupon_id: str
    name: str
    def __init__(self, coupon_id: _Optional[str] = ..., name: _Optional[str] = ...) -> None: ...

class UpdateCouponResponse(_message.Message):
    __slots__ = ("coupon",)
    COUPON_FIELD_NUMBER: _ClassVar[int]
    coupon: Coupon
    def __init__(self, coupon: _Optional[_Union[Coupon, _Mapping]] = ...) -> None: ...

class DeleteCouponRequest(_message.Message):
    __slots__ = ("coupon_id",)
    COUPON_ID_FIELD_NUMBER: _ClassVar[int]
    coupon_id: str
    def __init__(self, coupon_id: _Optional[str] = ...) -> None: ...

class DeleteCouponResponse(_message.Message):
    __slots__ = ("deleted",)
    DELETED_FIELD_NUMBER: _ClassVar[int]
    deleted: bool
    def __init__(self, deleted: _Optional[bool] = ...) -> None: ...

class CouponUsageStat(_message.Message):
    __slots__ = ("coupon_id", "total_uses", "last_used_at")
    COUPON_ID_FIELD_NUMBER: _ClassVar[int]
    TOTAL_USES_FIELD_NUMBER: _ClassVar[int]
    LAST_USED_AT_FIELD_NUMBER: _ClassVar[int]
    coupon_id: str
    total_uses: int
    last_used_at: str
    def __init__(self, coupon_id: _Optional[str] = ..., total_uses: _Optional[int] = ..., last_used_at: _Optional[str] = ...) -> None: ...

class ListCouponUsageRequest(_message.Message):
    __slots__ = ()
    def __init__(self) -> None: ...

class ListCouponUsageResponse(_message.Message):
    __slots__ = ("usage",)
    USAGE_FIELD_NUMBER: _ClassVar[int]
    usage: _containers.RepeatedCompositeFieldContainer[CouponUsageStat]
    def __init__(self, usage: _Optional[_Iterable[_Union[CouponUsageStat, _Mapping]]] = ...) -> None: ...

class GetCouponMappingsRequest(_message.Message):
    __slots__ = ()
    def __init__(self) -> None: ...

class GetCouponMappingsResponse(_message.Message):
    __slots__ = ("mappings",)
    class MappingsEntry(_message.Message):
        __slots__ = ("key", "value")
        KEY_FIELD_NUMBER: _ClassVar[int]
        VALUE_FIELD_NUMBER: _ClassVar[int]
        key: str
        value: str
        def __init__(self, key: _Optional[str] = ..., value: _Optional[str] = ...) -> None: ...
    MAPPINGS_FIELD_NUMBER: _ClassVar[int]
    mappings: _containers.ScalarMap[str, str]
    def __init__(self, mappings: _Optional[_Mapping[str, str]] = ...) -> None: ...

class SetCouponForPlanRequest(_message.Message):
    __slots__ = ("price_id", "coupon_id")
    PRICE_ID_FIELD_NUMBER: _ClassVar[int]
    COUPON_ID_FIELD_NUMBER: _ClassVar[int]
    price_id: str
    coupon_id: str
    def __init__(self, price_id: _Optional[str] = ..., coupon_id: _Optional[str] = ...) -> None: ...

class SetCouponForPlanResponse(_message.Message):
    __slots__ = ("assigned",)
    ASSIGNED_FIELD_NUMBER: _ClassVar[int]
    assigned: bool
    def __init__(self, assigned: _Optional[bool] = ...) -> None: ...

class RemoveCouponFromPlanRequest(_message.Message):
    __slots__ = ("price_id",)
    PRICE_ID_FIELD_NUMBER: _ClassVar[int]
    price_id: str
    def __init__(self, price_id: _Optional[str] = ...) -> None: ...

class RemoveCouponFromPlanResponse(_message.Message):
    __slots__ = ("removed",)
    REMOVED_FIELD_NUMBER: _ClassVar[int]
    removed: bool
    def __init__(self, removed: _Optional[bool] = ...) -> None: ...

class CouponImportPreviewItem(_message.Message):
    __slots__ = ("id", "name", "amount_off", "percent_off", "currency", "duration", "duration_in_months", "times_redeemed", "valid", "exists_locally")
    ID_FIELD_NUMBER: _ClassVar[int]
    NAME_FIELD_NUMBER: _ClassVar[int]
    AMOUNT_OFF_FIELD_NUMBER: _ClassVar[int]
    PERCENT_OFF_FIELD_NUMBER: _ClassVar[int]
    CURRENCY_FIELD_NUMBER: _ClassVar[int]
    DURATION_FIELD_NUMBER: _ClassVar[int]
    DURATION_IN_MONTHS_FIELD_NUMBER: _ClassVar[int]
    TIMES_REDEEMED_FIELD_NUMBER: _ClassVar[int]
    VALID_FIELD_NUMBER: _ClassVar[int]
    EXISTS_LOCALLY_FIELD_NUMBER: _ClassVar[int]
    id: str
    name: str
    amount_off: int
    percent_off: float
    currency: str
    duration: CouponDuration
    duration_in_months: int
    times_redeemed: int
    valid: bool
    exists_locally: bool
    def __init__(self, id: _Optional[str] = ..., name: _Optional[str] = ..., amount_off: _Optional[int] = ..., percent_off: _Optional[float] = ..., currency: _Optional[str] = ..., duration: _Optional[_Union[CouponDuration, str]] = ..., duration_in_months: _Optional[int] = ..., times_redeemed: _Optional[int] = ..., valid: _Optional[bool] = ..., exists_locally: _Optional[bool] = ...) -> None: ...

class GetCouponImportPreviewRequest(_message.Message):
    __slots__ = ()
    def __init__(self) -> None: ...

class GetCouponImportPreviewResponse(_message.Message):
    __slots__ = ("coupons", "total_coupons", "existing_count", "new_count")
    COUPONS_FIELD_NUMBER: _ClassVar[int]
    TOTAL_COUPONS_FIELD_NUMBER: _ClassVar[int]
    EXISTING_COUNT_FIELD_NUMBER: _ClassVar[int]
    NEW_COUNT_FIELD_NUMBER: _ClassVar[int]
    coupons: _containers.RepeatedCompositeFieldContainer[CouponImportPreviewItem]
    total_coupons: int
    existing_count: int
    new_count: int
    def __init__(self, coupons: _Optional[_Iterable[_Union[CouponImportPreviewItem, _Mapping]]] = ..., total_coupons: _Optional[int] = ..., existing_count: _Optional[int] = ..., new_count: _Optional[int] = ...) -> None: ...
