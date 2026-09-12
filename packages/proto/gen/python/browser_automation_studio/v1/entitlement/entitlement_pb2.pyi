import datetime

from buf.validate import validate_pb2 as _validate_pb2
from google.protobuf import struct_pb2 as _struct_pb2
from google.protobuf import timestamp_pb2 as _timestamp_pb2
from google.protobuf.internal import containers as _containers
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Iterable as _Iterable, Mapping as _Mapping
from typing import ClassVar as _ClassVar, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class FeatureAccess(_message.Message):
    __slots__ = ("id", "label", "description", "required_tier", "has_access")
    ID_FIELD_NUMBER: _ClassVar[int]
    LABEL_FIELD_NUMBER: _ClassVar[int]
    DESCRIPTION_FIELD_NUMBER: _ClassVar[int]
    REQUIRED_TIER_FIELD_NUMBER: _ClassVar[int]
    HAS_ACCESS_FIELD_NUMBER: _ClassVar[int]
    id: str
    label: str
    description: str
    required_tier: str
    has_access: bool
    def __init__(self, id: _Optional[str] = ..., label: _Optional[str] = ..., description: _Optional[str] = ..., required_tier: _Optional[str] = ..., has_access: _Optional[bool] = ...) -> None: ...

class EntitlementStatus(_message.Message):
    __slots__ = ("user_identity", "status", "tier", "is_active", "features", "feature_access", "monthly_limit", "monthly_used", "monthly_remaining", "requires_watermark", "can_use_ai", "can_use_recording", "entitlements_enabled", "override_tier", "ai_credits_used", "ai_credits_limit", "ai_credits_remaining", "ai_requests_count", "ai_reset_date")
    USER_IDENTITY_FIELD_NUMBER: _ClassVar[int]
    STATUS_FIELD_NUMBER: _ClassVar[int]
    TIER_FIELD_NUMBER: _ClassVar[int]
    IS_ACTIVE_FIELD_NUMBER: _ClassVar[int]
    FEATURES_FIELD_NUMBER: _ClassVar[int]
    FEATURE_ACCESS_FIELD_NUMBER: _ClassVar[int]
    MONTHLY_LIMIT_FIELD_NUMBER: _ClassVar[int]
    MONTHLY_USED_FIELD_NUMBER: _ClassVar[int]
    MONTHLY_REMAINING_FIELD_NUMBER: _ClassVar[int]
    REQUIRES_WATERMARK_FIELD_NUMBER: _ClassVar[int]
    CAN_USE_AI_FIELD_NUMBER: _ClassVar[int]
    CAN_USE_RECORDING_FIELD_NUMBER: _ClassVar[int]
    ENTITLEMENTS_ENABLED_FIELD_NUMBER: _ClassVar[int]
    OVERRIDE_TIER_FIELD_NUMBER: _ClassVar[int]
    AI_CREDITS_USED_FIELD_NUMBER: _ClassVar[int]
    AI_CREDITS_LIMIT_FIELD_NUMBER: _ClassVar[int]
    AI_CREDITS_REMAINING_FIELD_NUMBER: _ClassVar[int]
    AI_REQUESTS_COUNT_FIELD_NUMBER: _ClassVar[int]
    AI_RESET_DATE_FIELD_NUMBER: _ClassVar[int]
    user_identity: str
    status: str
    tier: str
    is_active: bool
    features: _containers.RepeatedScalarFieldContainer[str]
    feature_access: _containers.RepeatedCompositeFieldContainer[FeatureAccess]
    monthly_limit: int
    monthly_used: int
    monthly_remaining: int
    requires_watermark: bool
    can_use_ai: bool
    can_use_recording: bool
    entitlements_enabled: bool
    override_tier: str
    ai_credits_used: int
    ai_credits_limit: int
    ai_credits_remaining: int
    ai_requests_count: int
    ai_reset_date: str
    def __init__(self, user_identity: _Optional[str] = ..., status: _Optional[str] = ..., tier: _Optional[str] = ..., is_active: _Optional[bool] = ..., features: _Optional[_Iterable[str]] = ..., feature_access: _Optional[_Iterable[_Union[FeatureAccess, _Mapping]]] = ..., monthly_limit: _Optional[int] = ..., monthly_used: _Optional[int] = ..., monthly_remaining: _Optional[int] = ..., requires_watermark: _Optional[bool] = ..., can_use_ai: _Optional[bool] = ..., can_use_recording: _Optional[bool] = ..., entitlements_enabled: _Optional[bool] = ..., override_tier: _Optional[str] = ..., ai_credits_used: _Optional[int] = ..., ai_credits_limit: _Optional[int] = ..., ai_credits_remaining: _Optional[int] = ..., ai_requests_count: _Optional[int] = ..., ai_reset_date: _Optional[str] = ...) -> None: ...

class GetStatusRequest(_message.Message):
    __slots__ = ("user",)
    USER_FIELD_NUMBER: _ClassVar[int]
    user: str
    def __init__(self, user: _Optional[str] = ...) -> None: ...

class GetStatusResponse(_message.Message):
    __slots__ = ("status",)
    STATUS_FIELD_NUMBER: _ClassVar[int]
    status: EntitlementStatus
    def __init__(self, status: _Optional[_Union[EntitlementStatus, _Mapping]] = ...) -> None: ...

class GetIdentityRequest(_message.Message):
    __slots__ = ()
    def __init__(self) -> None: ...

class GetIdentityResponse(_message.Message):
    __slots__ = ("email",)
    EMAIL_FIELD_NUMBER: _ClassVar[int]
    email: str
    def __init__(self, email: _Optional[str] = ...) -> None: ...

class SetIdentityRequest(_message.Message):
    __slots__ = ("email",)
    EMAIL_FIELD_NUMBER: _ClassVar[int]
    email: str
    def __init__(self, email: _Optional[str] = ...) -> None: ...

class ClearIdentityRequest(_message.Message):
    __slots__ = ()
    def __init__(self) -> None: ...

class ClearIdentityResponse(_message.Message):
    __slots__ = ("status",)
    STATUS_FIELD_NUMBER: _ClassVar[int]
    status: str
    def __init__(self, status: _Optional[str] = ...) -> None: ...

class RefreshStatusRequest(_message.Message):
    __slots__ = ("user",)
    USER_FIELD_NUMBER: _ClassVar[int]
    user: str
    def __init__(self, user: _Optional[str] = ...) -> None: ...

class UsageSummary(_message.Message):
    __slots__ = ("user_identity", "billing_month", "total_credits_used", "total_operations", "by_operation", "operation_counts", "credits_limit", "credits_remaining", "period_start", "period_end", "reset_date")
    class ByOperationEntry(_message.Message):
        __slots__ = ("key", "value")
        KEY_FIELD_NUMBER: _ClassVar[int]
        VALUE_FIELD_NUMBER: _ClassVar[int]
        key: str
        value: int
        def __init__(self, key: _Optional[str] = ..., value: _Optional[int] = ...) -> None: ...
    class OperationCountsEntry(_message.Message):
        __slots__ = ("key", "value")
        KEY_FIELD_NUMBER: _ClassVar[int]
        VALUE_FIELD_NUMBER: _ClassVar[int]
        key: str
        value: int
        def __init__(self, key: _Optional[str] = ..., value: _Optional[int] = ...) -> None: ...
    USER_IDENTITY_FIELD_NUMBER: _ClassVar[int]
    BILLING_MONTH_FIELD_NUMBER: _ClassVar[int]
    TOTAL_CREDITS_USED_FIELD_NUMBER: _ClassVar[int]
    TOTAL_OPERATIONS_FIELD_NUMBER: _ClassVar[int]
    BY_OPERATION_FIELD_NUMBER: _ClassVar[int]
    OPERATION_COUNTS_FIELD_NUMBER: _ClassVar[int]
    CREDITS_LIMIT_FIELD_NUMBER: _ClassVar[int]
    CREDITS_REMAINING_FIELD_NUMBER: _ClassVar[int]
    PERIOD_START_FIELD_NUMBER: _ClassVar[int]
    PERIOD_END_FIELD_NUMBER: _ClassVar[int]
    RESET_DATE_FIELD_NUMBER: _ClassVar[int]
    user_identity: str
    billing_month: str
    total_credits_used: int
    total_operations: int
    by_operation: _containers.ScalarMap[str, int]
    operation_counts: _containers.ScalarMap[str, int]
    credits_limit: int
    credits_remaining: int
    period_start: _timestamp_pb2.Timestamp
    period_end: _timestamp_pb2.Timestamp
    reset_date: _timestamp_pb2.Timestamp
    def __init__(self, user_identity: _Optional[str] = ..., billing_month: _Optional[str] = ..., total_credits_used: _Optional[int] = ..., total_operations: _Optional[int] = ..., by_operation: _Optional[_Mapping[str, int]] = ..., operation_counts: _Optional[_Mapping[str, int]] = ..., credits_limit: _Optional[int] = ..., credits_remaining: _Optional[int] = ..., period_start: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., period_end: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., reset_date: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ...) -> None: ...

class GetUsageRequest(_message.Message):
    __slots__ = ("user",)
    USER_FIELD_NUMBER: _ClassVar[int]
    user: str
    def __init__(self, user: _Optional[str] = ...) -> None: ...

class GetUsageHistoryRequest(_message.Message):
    __slots__ = ("user", "months", "offset")
    USER_FIELD_NUMBER: _ClassVar[int]
    MONTHS_FIELD_NUMBER: _ClassVar[int]
    OFFSET_FIELD_NUMBER: _ClassVar[int]
    user: str
    months: int
    offset: int
    def __init__(self, user: _Optional[str] = ..., months: _Optional[int] = ..., offset: _Optional[int] = ...) -> None: ...

class GetUsageHistoryResponse(_message.Message):
    __slots__ = ("user_identity", "periods", "has_more")
    USER_IDENTITY_FIELD_NUMBER: _ClassVar[int]
    PERIODS_FIELD_NUMBER: _ClassVar[int]
    HAS_MORE_FIELD_NUMBER: _ClassVar[int]
    user_identity: str
    periods: _containers.RepeatedCompositeFieldContainer[UsageSummary]
    has_more: bool
    def __init__(self, user_identity: _Optional[str] = ..., periods: _Optional[_Iterable[_Union[UsageSummary, _Mapping]]] = ..., has_more: _Optional[bool] = ...) -> None: ...

class OperationLogEntry(_message.Message):
    __slots__ = ("id", "operation_type", "credits_charged", "success", "created_at", "metadata", "error_message")
    ID_FIELD_NUMBER: _ClassVar[int]
    OPERATION_TYPE_FIELD_NUMBER: _ClassVar[int]
    CREDITS_CHARGED_FIELD_NUMBER: _ClassVar[int]
    SUCCESS_FIELD_NUMBER: _ClassVar[int]
    CREATED_AT_FIELD_NUMBER: _ClassVar[int]
    METADATA_FIELD_NUMBER: _ClassVar[int]
    ERROR_MESSAGE_FIELD_NUMBER: _ClassVar[int]
    id: str
    operation_type: str
    credits_charged: int
    success: bool
    created_at: _timestamp_pb2.Timestamp
    metadata: _struct_pb2.Struct
    error_message: str
    def __init__(self, id: _Optional[str] = ..., operation_type: _Optional[str] = ..., credits_charged: _Optional[int] = ..., success: _Optional[bool] = ..., created_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., metadata: _Optional[_Union[_struct_pb2.Struct, _Mapping]] = ..., error_message: _Optional[str] = ...) -> None: ...

class OperationLogPage(_message.Message):
    __slots__ = ("user_identity", "billing_month", "operations", "total", "limit", "offset", "has_more")
    USER_IDENTITY_FIELD_NUMBER: _ClassVar[int]
    BILLING_MONTH_FIELD_NUMBER: _ClassVar[int]
    OPERATIONS_FIELD_NUMBER: _ClassVar[int]
    TOTAL_FIELD_NUMBER: _ClassVar[int]
    LIMIT_FIELD_NUMBER: _ClassVar[int]
    OFFSET_FIELD_NUMBER: _ClassVar[int]
    HAS_MORE_FIELD_NUMBER: _ClassVar[int]
    user_identity: str
    billing_month: str
    operations: _containers.RepeatedCompositeFieldContainer[OperationLogEntry]
    total: int
    limit: int
    offset: int
    has_more: bool
    def __init__(self, user_identity: _Optional[str] = ..., billing_month: _Optional[str] = ..., operations: _Optional[_Iterable[_Union[OperationLogEntry, _Mapping]]] = ..., total: _Optional[int] = ..., limit: _Optional[int] = ..., offset: _Optional[int] = ..., has_more: _Optional[bool] = ...) -> None: ...

class GetOperationLogRequest(_message.Message):
    __slots__ = ("user", "month", "category", "limit", "offset")
    USER_FIELD_NUMBER: _ClassVar[int]
    MONTH_FIELD_NUMBER: _ClassVar[int]
    CATEGORY_FIELD_NUMBER: _ClassVar[int]
    LIMIT_FIELD_NUMBER: _ClassVar[int]
    OFFSET_FIELD_NUMBER: _ClassVar[int]
    user: str
    month: str
    category: str
    limit: int
    offset: int
    def __init__(self, user: _Optional[str] = ..., month: _Optional[str] = ..., category: _Optional[str] = ..., limit: _Optional[int] = ..., offset: _Optional[int] = ...) -> None: ...

class GetOverrideRequest(_message.Message):
    __slots__ = ()
    def __init__(self) -> None: ...

class GetOverrideResponse(_message.Message):
    __slots__ = ("tier",)
    TIER_FIELD_NUMBER: _ClassVar[int]
    tier: str
    def __init__(self, tier: _Optional[str] = ...) -> None: ...

class SetOverrideRequest(_message.Message):
    __slots__ = ("tier",)
    TIER_FIELD_NUMBER: _ClassVar[int]
    tier: str
    def __init__(self, tier: _Optional[str] = ...) -> None: ...

class SetOverrideResponse(_message.Message):
    __slots__ = ("tier",)
    TIER_FIELD_NUMBER: _ClassVar[int]
    tier: str
    def __init__(self, tier: _Optional[str] = ...) -> None: ...

class ClearOverrideRequest(_message.Message):
    __slots__ = ()
    def __init__(self) -> None: ...

class ClearOverrideResponse(_message.Message):
    __slots__ = ()
    def __init__(self) -> None: ...

class ApiSourceConfig(_message.Message):
    __slots__ = ("source", "local_port")
    SOURCE_FIELD_NUMBER: _ClassVar[int]
    LOCAL_PORT_FIELD_NUMBER: _ClassVar[int]
    source: str
    local_port: int
    def __init__(self, source: _Optional[str] = ..., local_port: _Optional[int] = ...) -> None: ...

class GetApiSourceRequest(_message.Message):
    __slots__ = ()
    def __init__(self) -> None: ...

class SetApiSourceRequest(_message.Message):
    __slots__ = ("source", "local_port")
    SOURCE_FIELD_NUMBER: _ClassVar[int]
    LOCAL_PORT_FIELD_NUMBER: _ClassVar[int]
    source: str
    local_port: int
    def __init__(self, source: _Optional[str] = ..., local_port: _Optional[int] = ...) -> None: ...

class ClearApiSourceRequest(_message.Message):
    __slots__ = ()
    def __init__(self) -> None: ...

class ClearApiSourceResponse(_message.Message):
    __slots__ = ()
    def __init__(self) -> None: ...
