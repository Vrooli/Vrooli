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

class TrafficDimension(int, metaclass=_enum_type_wrapper.EnumTypeWrapper):
    __slots__ = ()
    TRAFFIC_DIMENSION_UNSPECIFIED: _ClassVar[TrafficDimension]
    COUNTRY: _ClassVar[TrafficDimension]
    REFERRER_KIND: _ClassVar[TrafficDimension]
    UTM_SOURCE: _ClassVar[TrafficDimension]
    UTM_CAMPAIGN: _ClassVar[TrafficDimension]
    DEVICE_CLASS: _ClassVar[TrafficDimension]
    LANDING_PATH: _ClassVar[TrafficDimension]
    VARIANT: _ClassVar[TrafficDimension]
TRAFFIC_DIMENSION_UNSPECIFIED: TrafficDimension
COUNTRY: TrafficDimension
REFERRER_KIND: TrafficDimension
UTM_SOURCE: TrafficDimension
UTM_CAMPAIGN: TrafficDimension
DEVICE_CLASS: TrafficDimension
LANDING_PATH: TrafficDimension
VARIANT: TrafficDimension

class TrackEventRequest(_message.Message):
    __slots__ = ("event_type", "variant_id", "event_data", "session_id", "visitor_id", "event_id", "variant_slug", "utm_source", "utm_medium", "utm_campaign", "landing_path", "referrer")
    EVENT_TYPE_FIELD_NUMBER: _ClassVar[int]
    VARIANT_ID_FIELD_NUMBER: _ClassVar[int]
    EVENT_DATA_FIELD_NUMBER: _ClassVar[int]
    SESSION_ID_FIELD_NUMBER: _ClassVar[int]
    VISITOR_ID_FIELD_NUMBER: _ClassVar[int]
    EVENT_ID_FIELD_NUMBER: _ClassVar[int]
    VARIANT_SLUG_FIELD_NUMBER: _ClassVar[int]
    UTM_SOURCE_FIELD_NUMBER: _ClassVar[int]
    UTM_MEDIUM_FIELD_NUMBER: _ClassVar[int]
    UTM_CAMPAIGN_FIELD_NUMBER: _ClassVar[int]
    LANDING_PATH_FIELD_NUMBER: _ClassVar[int]
    REFERRER_FIELD_NUMBER: _ClassVar[int]
    event_type: str
    variant_id: int
    event_data: _struct_pb2.Struct
    session_id: str
    visitor_id: str
    event_id: str
    variant_slug: str
    utm_source: str
    utm_medium: str
    utm_campaign: str
    landing_path: str
    referrer: str
    def __init__(self, event_type: _Optional[str] = ..., variant_id: _Optional[int] = ..., event_data: _Optional[_Union[_struct_pb2.Struct, _Mapping]] = ..., session_id: _Optional[str] = ..., visitor_id: _Optional[str] = ..., event_id: _Optional[str] = ..., variant_slug: _Optional[str] = ..., utm_source: _Optional[str] = ..., utm_medium: _Optional[str] = ..., utm_campaign: _Optional[str] = ..., landing_path: _Optional[str] = ..., referrer: _Optional[str] = ...) -> None: ...

class TrafficBreakdownRow(_message.Message):
    __slots__ = ("key", "label", "sessions", "conversions", "revenue_minor", "share")
    KEY_FIELD_NUMBER: _ClassVar[int]
    LABEL_FIELD_NUMBER: _ClassVar[int]
    SESSIONS_FIELD_NUMBER: _ClassVar[int]
    CONVERSIONS_FIELD_NUMBER: _ClassVar[int]
    REVENUE_MINOR_FIELD_NUMBER: _ClassVar[int]
    SHARE_FIELD_NUMBER: _ClassVar[int]
    key: str
    label: str
    sessions: int
    conversions: int
    revenue_minor: int
    share: float
    def __init__(self, key: _Optional[str] = ..., label: _Optional[str] = ..., sessions: _Optional[int] = ..., conversions: _Optional[int] = ..., revenue_minor: _Optional[int] = ..., share: _Optional[float] = ...) -> None: ...

class GetTrafficBreakdownRequest(_message.Message):
    __slots__ = ("dimension", "start_date", "end_date", "limit")
    DIMENSION_FIELD_NUMBER: _ClassVar[int]
    START_DATE_FIELD_NUMBER: _ClassVar[int]
    END_DATE_FIELD_NUMBER: _ClassVar[int]
    LIMIT_FIELD_NUMBER: _ClassVar[int]
    dimension: TrafficDimension
    start_date: str
    end_date: str
    limit: int
    def __init__(self, dimension: _Optional[_Union[TrafficDimension, str]] = ..., start_date: _Optional[str] = ..., end_date: _Optional[str] = ..., limit: _Optional[int] = ...) -> None: ...

class GetTrafficBreakdownResponse(_message.Message):
    __slots__ = ("rows", "total_sessions", "exhaustive", "currency", "observed_at")
    ROWS_FIELD_NUMBER: _ClassVar[int]
    TOTAL_SESSIONS_FIELD_NUMBER: _ClassVar[int]
    EXHAUSTIVE_FIELD_NUMBER: _ClassVar[int]
    CURRENCY_FIELD_NUMBER: _ClassVar[int]
    OBSERVED_AT_FIELD_NUMBER: _ClassVar[int]
    rows: _containers.RepeatedCompositeFieldContainer[TrafficBreakdownRow]
    total_sessions: int
    exhaustive: bool
    currency: str
    observed_at: _timestamp_pb2.Timestamp
    def __init__(self, rows: _Optional[_Iterable[_Union[TrafficBreakdownRow, _Mapping]]] = ..., total_sessions: _Optional[int] = ..., exhaustive: _Optional[bool] = ..., currency: _Optional[str] = ..., observed_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ...) -> None: ...

class TrafficSeriesPoint(_message.Message):
    __slots__ = ("bucket_start", "value")
    BUCKET_START_FIELD_NUMBER: _ClassVar[int]
    VALUE_FIELD_NUMBER: _ClassVar[int]
    bucket_start: str
    value: float
    def __init__(self, bucket_start: _Optional[str] = ..., value: _Optional[float] = ...) -> None: ...

class GetTrafficSeriesRequest(_message.Message):
    __slots__ = ("metric", "start_date", "end_date", "bucket")
    METRIC_FIELD_NUMBER: _ClassVar[int]
    START_DATE_FIELD_NUMBER: _ClassVar[int]
    END_DATE_FIELD_NUMBER: _ClassVar[int]
    BUCKET_FIELD_NUMBER: _ClassVar[int]
    metric: str
    start_date: str
    end_date: str
    bucket: str
    def __init__(self, metric: _Optional[str] = ..., start_date: _Optional[str] = ..., end_date: _Optional[str] = ..., bucket: _Optional[str] = ...) -> None: ...

class GetTrafficSeriesResponse(_message.Message):
    __slots__ = ("points", "unit", "observed_at")
    POINTS_FIELD_NUMBER: _ClassVar[int]
    UNIT_FIELD_NUMBER: _ClassVar[int]
    OBSERVED_AT_FIELD_NUMBER: _ClassVar[int]
    points: _containers.RepeatedCompositeFieldContainer[TrafficSeriesPoint]
    unit: str
    observed_at: _timestamp_pb2.Timestamp
    def __init__(self, points: _Optional[_Iterable[_Union[TrafficSeriesPoint, _Mapping]]] = ..., unit: _Optional[str] = ..., observed_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ...) -> None: ...

class TrackEventResponse(_message.Message):
    __slots__ = ("success", "message")
    SUCCESS_FIELD_NUMBER: _ClassVar[int]
    MESSAGE_FIELD_NUMBER: _ClassVar[int]
    success: bool
    message: str
    def __init__(self, success: _Optional[bool] = ..., message: _Optional[str] = ...) -> None: ...

class VariantStats(_message.Message):
    __slots__ = ("variant_id", "variant_slug", "variant_name", "views", "cta_clicks", "conversions", "downloads", "conversion_rate", "trend", "avg_scroll_depth", "exposures")
    VARIANT_ID_FIELD_NUMBER: _ClassVar[int]
    VARIANT_SLUG_FIELD_NUMBER: _ClassVar[int]
    VARIANT_NAME_FIELD_NUMBER: _ClassVar[int]
    VIEWS_FIELD_NUMBER: _ClassVar[int]
    CTA_CLICKS_FIELD_NUMBER: _ClassVar[int]
    CONVERSIONS_FIELD_NUMBER: _ClassVar[int]
    DOWNLOADS_FIELD_NUMBER: _ClassVar[int]
    CONVERSION_RATE_FIELD_NUMBER: _ClassVar[int]
    TREND_FIELD_NUMBER: _ClassVar[int]
    AVG_SCROLL_DEPTH_FIELD_NUMBER: _ClassVar[int]
    EXPOSURES_FIELD_NUMBER: _ClassVar[int]
    variant_id: int
    variant_slug: str
    variant_name: str
    views: int
    cta_clicks: int
    conversions: int
    downloads: int
    conversion_rate: float
    trend: float
    avg_scroll_depth: float
    exposures: int
    def __init__(self, variant_id: _Optional[int] = ..., variant_slug: _Optional[str] = ..., variant_name: _Optional[str] = ..., views: _Optional[int] = ..., cta_clicks: _Optional[int] = ..., conversions: _Optional[int] = ..., downloads: _Optional[int] = ..., conversion_rate: _Optional[float] = ..., trend: _Optional[float] = ..., avg_scroll_depth: _Optional[float] = ..., exposures: _Optional[int] = ...) -> None: ...

class AnalyticsSummary(_message.Message):
    __slots__ = ("total_visitors", "total_downloads", "variant_stats", "top_cta", "top_cta_ctr", "observed_at")
    TOTAL_VISITORS_FIELD_NUMBER: _ClassVar[int]
    TOTAL_DOWNLOADS_FIELD_NUMBER: _ClassVar[int]
    VARIANT_STATS_FIELD_NUMBER: _ClassVar[int]
    TOP_CTA_FIELD_NUMBER: _ClassVar[int]
    TOP_CTA_CTR_FIELD_NUMBER: _ClassVar[int]
    OBSERVED_AT_FIELD_NUMBER: _ClassVar[int]
    total_visitors: int
    total_downloads: int
    variant_stats: _containers.RepeatedCompositeFieldContainer[VariantStats]
    top_cta: str
    top_cta_ctr: float
    observed_at: _timestamp_pb2.Timestamp
    def __init__(self, total_visitors: _Optional[int] = ..., total_downloads: _Optional[int] = ..., variant_stats: _Optional[_Iterable[_Union[VariantStats, _Mapping]]] = ..., top_cta: _Optional[str] = ..., top_cta_ctr: _Optional[float] = ..., observed_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ...) -> None: ...

class AdminRevenue(_message.Message):
    __slots__ = ("mrr", "mrr_unit", "today", "today_unit", "currency", "sample_size", "observed_at")
    MRR_FIELD_NUMBER: _ClassVar[int]
    MRR_UNIT_FIELD_NUMBER: _ClassVar[int]
    TODAY_FIELD_NUMBER: _ClassVar[int]
    TODAY_UNIT_FIELD_NUMBER: _ClassVar[int]
    CURRENCY_FIELD_NUMBER: _ClassVar[int]
    SAMPLE_SIZE_FIELD_NUMBER: _ClassVar[int]
    OBSERVED_AT_FIELD_NUMBER: _ClassVar[int]
    mrr: float
    mrr_unit: str
    today: float
    today_unit: str
    currency: str
    sample_size: int
    observed_at: _timestamp_pb2.Timestamp
    def __init__(self, mrr: _Optional[float] = ..., mrr_unit: _Optional[str] = ..., today: _Optional[float] = ..., today_unit: _Optional[str] = ..., currency: _Optional[str] = ..., sample_size: _Optional[int] = ..., observed_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ...) -> None: ...

class RevenueSummary(_message.Message):
    __slots__ = ("observed_at", "currency", "mrr_minor", "revenue_today_minor", "revenue_window_minor", "active_subscriptions", "subscriptions_churned_window", "churn_rate_percent", "credit_balance_total", "credit_burned_window", "usage_records_window", "sample_size", "trials_without_payment_method", "mrr_unit", "revenue_today_unit", "revenue_window_unit", "credit_unit", "currency_excluded_count")
    OBSERVED_AT_FIELD_NUMBER: _ClassVar[int]
    CURRENCY_FIELD_NUMBER: _ClassVar[int]
    MRR_MINOR_FIELD_NUMBER: _ClassVar[int]
    REVENUE_TODAY_MINOR_FIELD_NUMBER: _ClassVar[int]
    REVENUE_WINDOW_MINOR_FIELD_NUMBER: _ClassVar[int]
    ACTIVE_SUBSCRIPTIONS_FIELD_NUMBER: _ClassVar[int]
    SUBSCRIPTIONS_CHURNED_WINDOW_FIELD_NUMBER: _ClassVar[int]
    CHURN_RATE_PERCENT_FIELD_NUMBER: _ClassVar[int]
    CREDIT_BALANCE_TOTAL_FIELD_NUMBER: _ClassVar[int]
    CREDIT_BURNED_WINDOW_FIELD_NUMBER: _ClassVar[int]
    USAGE_RECORDS_WINDOW_FIELD_NUMBER: _ClassVar[int]
    SAMPLE_SIZE_FIELD_NUMBER: _ClassVar[int]
    TRIALS_WITHOUT_PAYMENT_METHOD_FIELD_NUMBER: _ClassVar[int]
    MRR_UNIT_FIELD_NUMBER: _ClassVar[int]
    REVENUE_TODAY_UNIT_FIELD_NUMBER: _ClassVar[int]
    REVENUE_WINDOW_UNIT_FIELD_NUMBER: _ClassVar[int]
    CREDIT_UNIT_FIELD_NUMBER: _ClassVar[int]
    CURRENCY_EXCLUDED_COUNT_FIELD_NUMBER: _ClassVar[int]
    observed_at: _timestamp_pb2.Timestamp
    currency: str
    mrr_minor: int
    revenue_today_minor: int
    revenue_window_minor: int
    active_subscriptions: int
    subscriptions_churned_window: int
    churn_rate_percent: float
    credit_balance_total: int
    credit_burned_window: int
    usage_records_window: int
    sample_size: int
    trials_without_payment_method: int
    mrr_unit: str
    revenue_today_unit: str
    revenue_window_unit: str
    credit_unit: str
    currency_excluded_count: int
    def __init__(self, observed_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., currency: _Optional[str] = ..., mrr_minor: _Optional[int] = ..., revenue_today_minor: _Optional[int] = ..., revenue_window_minor: _Optional[int] = ..., active_subscriptions: _Optional[int] = ..., subscriptions_churned_window: _Optional[int] = ..., churn_rate_percent: _Optional[float] = ..., credit_balance_total: _Optional[int] = ..., credit_burned_window: _Optional[int] = ..., usage_records_window: _Optional[int] = ..., sample_size: _Optional[int] = ..., trials_without_payment_method: _Optional[int] = ..., mrr_unit: _Optional[str] = ..., revenue_today_unit: _Optional[str] = ..., revenue_window_unit: _Optional[str] = ..., credit_unit: _Optional[str] = ..., currency_excluded_count: _Optional[int] = ...) -> None: ...

class GetAnalyticsSummaryRequest(_message.Message):
    __slots__ = ("start_date", "end_date")
    START_DATE_FIELD_NUMBER: _ClassVar[int]
    END_DATE_FIELD_NUMBER: _ClassVar[int]
    start_date: str
    end_date: str
    def __init__(self, start_date: _Optional[str] = ..., end_date: _Optional[str] = ...) -> None: ...

class GetVariantStatsRequest(_message.Message):
    __slots__ = ("start_date", "end_date", "variant")
    START_DATE_FIELD_NUMBER: _ClassVar[int]
    END_DATE_FIELD_NUMBER: _ClassVar[int]
    VARIANT_FIELD_NUMBER: _ClassVar[int]
    start_date: str
    end_date: str
    variant: str
    def __init__(self, start_date: _Optional[str] = ..., end_date: _Optional[str] = ..., variant: _Optional[str] = ...) -> None: ...

class GetVariantStatsResponse(_message.Message):
    __slots__ = ("start_date", "end_date", "stats")
    START_DATE_FIELD_NUMBER: _ClassVar[int]
    END_DATE_FIELD_NUMBER: _ClassVar[int]
    STATS_FIELD_NUMBER: _ClassVar[int]
    start_date: str
    end_date: str
    stats: _containers.RepeatedCompositeFieldContainer[VariantStats]
    def __init__(self, start_date: _Optional[str] = ..., end_date: _Optional[str] = ..., stats: _Optional[_Iterable[_Union[VariantStats, _Mapping]]] = ...) -> None: ...

class GetAdminRevenueRequest(_message.Message):
    __slots__ = ()
    def __init__(self) -> None: ...

class GetRevenueSummaryRequest(_message.Message):
    __slots__ = ()
    def __init__(self) -> None: ...
