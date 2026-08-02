from google.protobuf import struct_pb2 as _struct_pb2
from google.protobuf.internal import containers as _containers
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Iterable as _Iterable, Mapping as _Mapping
from typing import ClassVar as _ClassVar, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class TrackEventRequest(_message.Message):
    __slots__ = ("event_type", "variant_id", "event_data", "session_id", "visitor_id", "event_id", "variant_slug")
    EVENT_TYPE_FIELD_NUMBER: _ClassVar[int]
    VARIANT_ID_FIELD_NUMBER: _ClassVar[int]
    EVENT_DATA_FIELD_NUMBER: _ClassVar[int]
    SESSION_ID_FIELD_NUMBER: _ClassVar[int]
    VISITOR_ID_FIELD_NUMBER: _ClassVar[int]
    EVENT_ID_FIELD_NUMBER: _ClassVar[int]
    VARIANT_SLUG_FIELD_NUMBER: _ClassVar[int]
    event_type: str
    variant_id: int
    event_data: _struct_pb2.Struct
    session_id: str
    visitor_id: str
    event_id: str
    variant_slug: str
    def __init__(self, event_type: _Optional[str] = ..., variant_id: _Optional[int] = ..., event_data: _Optional[_Union[_struct_pb2.Struct, _Mapping]] = ..., session_id: _Optional[str] = ..., visitor_id: _Optional[str] = ..., event_id: _Optional[str] = ..., variant_slug: _Optional[str] = ...) -> None: ...

class TrackEventResponse(_message.Message):
    __slots__ = ("success", "message")
    SUCCESS_FIELD_NUMBER: _ClassVar[int]
    MESSAGE_FIELD_NUMBER: _ClassVar[int]
    success: bool
    message: str
    def __init__(self, success: _Optional[bool] = ..., message: _Optional[str] = ...) -> None: ...

class VariantStats(_message.Message):
    __slots__ = ("variant_id", "variant_slug", "variant_name", "views", "cta_clicks", "conversions", "downloads", "conversion_rate", "trend", "avg_scroll_depth")
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
    def __init__(self, variant_id: _Optional[int] = ..., variant_slug: _Optional[str] = ..., variant_name: _Optional[str] = ..., views: _Optional[int] = ..., cta_clicks: _Optional[int] = ..., conversions: _Optional[int] = ..., downloads: _Optional[int] = ..., conversion_rate: _Optional[float] = ..., trend: _Optional[float] = ..., avg_scroll_depth: _Optional[float] = ...) -> None: ...

class AnalyticsSummary(_message.Message):
    __slots__ = ("total_visitors", "total_downloads", "variant_stats", "top_cta", "top_cta_ctr")
    TOTAL_VISITORS_FIELD_NUMBER: _ClassVar[int]
    TOTAL_DOWNLOADS_FIELD_NUMBER: _ClassVar[int]
    VARIANT_STATS_FIELD_NUMBER: _ClassVar[int]
    TOP_CTA_FIELD_NUMBER: _ClassVar[int]
    TOP_CTA_CTR_FIELD_NUMBER: _ClassVar[int]
    total_visitors: int
    total_downloads: int
    variant_stats: _containers.RepeatedCompositeFieldContainer[VariantStats]
    top_cta: str
    top_cta_ctr: float
    def __init__(self, total_visitors: _Optional[int] = ..., total_downloads: _Optional[int] = ..., variant_stats: _Optional[_Iterable[_Union[VariantStats, _Mapping]]] = ..., top_cta: _Optional[str] = ..., top_cta_ctr: _Optional[float] = ...) -> None: ...

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
