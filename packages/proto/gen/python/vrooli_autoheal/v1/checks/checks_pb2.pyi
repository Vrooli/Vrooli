import datetime

from google.protobuf import timestamp_pb2 as _timestamp_pb2
from google.protobuf.internal import containers as _containers
from google.protobuf.internal import enum_type_wrapper as _enum_type_wrapper
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Iterable as _Iterable, Mapping as _Mapping
from typing import ClassVar as _ClassVar, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class CheckStatus(int, metaclass=_enum_type_wrapper.EnumTypeWrapper):
    __slots__ = ()
    CHECK_STATUS_UNSPECIFIED: _ClassVar[CheckStatus]
    CHECK_STATUS_OK: _ClassVar[CheckStatus]
    CHECK_STATUS_WARNING: _ClassVar[CheckStatus]
    CHECK_STATUS_CRITICAL: _ClassVar[CheckStatus]
    CHECK_STATUS_NOT_APPLICABLE: _ClassVar[CheckStatus]
CHECK_STATUS_UNSPECIFIED: CheckStatus
CHECK_STATUS_OK: CheckStatus
CHECK_STATUS_WARNING: CheckStatus
CHECK_STATUS_CRITICAL: CheckStatus
CHECK_STATUS_NOT_APPLICABLE: CheckStatus

class CheckInfo(_message.Message):
    __slots__ = ("id", "title", "description", "importance", "category", "interval_seconds", "platforms")
    ID_FIELD_NUMBER: _ClassVar[int]
    TITLE_FIELD_NUMBER: _ClassVar[int]
    DESCRIPTION_FIELD_NUMBER: _ClassVar[int]
    IMPORTANCE_FIELD_NUMBER: _ClassVar[int]
    CATEGORY_FIELD_NUMBER: _ClassVar[int]
    INTERVAL_SECONDS_FIELD_NUMBER: _ClassVar[int]
    PLATFORMS_FIELD_NUMBER: _ClassVar[int]
    id: str
    title: str
    description: str
    importance: str
    category: str
    interval_seconds: int
    platforms: _containers.RepeatedScalarFieldContainer[str]
    def __init__(self, id: _Optional[str] = ..., title: _Optional[str] = ..., description: _Optional[str] = ..., importance: _Optional[str] = ..., category: _Optional[str] = ..., interval_seconds: _Optional[int] = ..., platforms: _Optional[_Iterable[str]] = ...) -> None: ...

class CheckResult(_message.Message):
    __slots__ = ("check_id", "status", "message", "observed_at", "duration_ms", "details_json")
    CHECK_ID_FIELD_NUMBER: _ClassVar[int]
    STATUS_FIELD_NUMBER: _ClassVar[int]
    MESSAGE_FIELD_NUMBER: _ClassVar[int]
    OBSERVED_AT_FIELD_NUMBER: _ClassVar[int]
    DURATION_MS_FIELD_NUMBER: _ClassVar[int]
    DETAILS_JSON_FIELD_NUMBER: _ClassVar[int]
    check_id: str
    status: CheckStatus
    message: str
    observed_at: _timestamp_pb2.Timestamp
    duration_ms: int
    details_json: str
    def __init__(self, check_id: _Optional[str] = ..., status: _Optional[_Union[CheckStatus, str]] = ..., message: _Optional[str] = ..., observed_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., duration_ms: _Optional[int] = ..., details_json: _Optional[str] = ...) -> None: ...

class Transition(_message.Message):
    __slots__ = ("check_id", "from_status", "to_status", "message", "observed_at")
    CHECK_ID_FIELD_NUMBER: _ClassVar[int]
    FROM_STATUS_FIELD_NUMBER: _ClassVar[int]
    TO_STATUS_FIELD_NUMBER: _ClassVar[int]
    MESSAGE_FIELD_NUMBER: _ClassVar[int]
    OBSERVED_AT_FIELD_NUMBER: _ClassVar[int]
    check_id: str
    from_status: CheckStatus
    to_status: CheckStatus
    message: str
    observed_at: _timestamp_pb2.Timestamp
    def __init__(self, check_id: _Optional[str] = ..., from_status: _Optional[_Union[CheckStatus, str]] = ..., to_status: _Optional[_Union[CheckStatus, str]] = ..., message: _Optional[str] = ..., observed_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ...) -> None: ...

class ListChecksRequest(_message.Message):
    __slots__ = ()
    def __init__(self) -> None: ...

class ListChecksResponse(_message.Message):
    __slots__ = ("checks",)
    CHECKS_FIELD_NUMBER: _ClassVar[int]
    checks: _containers.RepeatedCompositeFieldContainer[CheckInfo]
    def __init__(self, checks: _Optional[_Iterable[_Union[CheckInfo, _Mapping]]] = ...) -> None: ...

class GetCheckRequest(_message.Message):
    __slots__ = ("check_id",)
    CHECK_ID_FIELD_NUMBER: _ClassVar[int]
    check_id: str
    def __init__(self, check_id: _Optional[str] = ...) -> None: ...

class GetCheckResponse(_message.Message):
    __slots__ = ("result",)
    RESULT_FIELD_NUMBER: _ClassVar[int]
    result: CheckResult
    def __init__(self, result: _Optional[_Union[CheckResult, _Mapping]] = ...) -> None: ...

class GetHistoryRequest(_message.Message):
    __slots__ = ("check_id", "limit")
    CHECK_ID_FIELD_NUMBER: _ClassVar[int]
    LIMIT_FIELD_NUMBER: _ClassVar[int]
    check_id: str
    limit: int
    def __init__(self, check_id: _Optional[str] = ..., limit: _Optional[int] = ...) -> None: ...

class GetHistoryResponse(_message.Message):
    __slots__ = ("results",)
    RESULTS_FIELD_NUMBER: _ClassVar[int]
    results: _containers.RepeatedCompositeFieldContainer[CheckResult]
    def __init__(self, results: _Optional[_Iterable[_Union[CheckResult, _Mapping]]] = ...) -> None: ...

class GetStatusRequest(_message.Message):
    __slots__ = ()
    def __init__(self) -> None: ...

class GetStatusResponse(_message.Message):
    __slots__ = ("status", "total_count", "ok_count", "warning_count", "critical_count", "not_applicable_count", "checks", "computed_at")
    STATUS_FIELD_NUMBER: _ClassVar[int]
    TOTAL_COUNT_FIELD_NUMBER: _ClassVar[int]
    OK_COUNT_FIELD_NUMBER: _ClassVar[int]
    WARNING_COUNT_FIELD_NUMBER: _ClassVar[int]
    CRITICAL_COUNT_FIELD_NUMBER: _ClassVar[int]
    NOT_APPLICABLE_COUNT_FIELD_NUMBER: _ClassVar[int]
    CHECKS_FIELD_NUMBER: _ClassVar[int]
    COMPUTED_AT_FIELD_NUMBER: _ClassVar[int]
    status: CheckStatus
    total_count: int
    ok_count: int
    warning_count: int
    critical_count: int
    not_applicable_count: int
    checks: _containers.RepeatedCompositeFieldContainer[CheckResult]
    computed_at: _timestamp_pb2.Timestamp
    def __init__(self, status: _Optional[_Union[CheckStatus, str]] = ..., total_count: _Optional[int] = ..., ok_count: _Optional[int] = ..., warning_count: _Optional[int] = ..., critical_count: _Optional[int] = ..., not_applicable_count: _Optional[int] = ..., checks: _Optional[_Iterable[_Union[CheckResult, _Mapping]]] = ..., computed_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ...) -> None: ...

class GetTransitionsRequest(_message.Message):
    __slots__ = ("window_hours", "limit")
    WINDOW_HOURS_FIELD_NUMBER: _ClassVar[int]
    LIMIT_FIELD_NUMBER: _ClassVar[int]
    window_hours: int
    limit: int
    def __init__(self, window_hours: _Optional[int] = ..., limit: _Optional[int] = ...) -> None: ...

class GetTransitionsResponse(_message.Message):
    __slots__ = ("transitions",)
    TRANSITIONS_FIELD_NUMBER: _ClassVar[int]
    transitions: _containers.RepeatedCompositeFieldContainer[Transition]
    def __init__(self, transitions: _Optional[_Iterable[_Union[Transition, _Mapping]]] = ...) -> None: ...

class Reconcile(_message.Message):
    __slots__ = ("ghost_check_ids", "unsupervised_plant", "available", "unavailable_reason", "computed_at", "out_of_scope_check_ids", "ghost_detection_available", "ghost_unavailable_reason")
    GHOST_CHECK_IDS_FIELD_NUMBER: _ClassVar[int]
    UNSUPERVISED_PLANT_FIELD_NUMBER: _ClassVar[int]
    AVAILABLE_FIELD_NUMBER: _ClassVar[int]
    UNAVAILABLE_REASON_FIELD_NUMBER: _ClassVar[int]
    COMPUTED_AT_FIELD_NUMBER: _ClassVar[int]
    OUT_OF_SCOPE_CHECK_IDS_FIELD_NUMBER: _ClassVar[int]
    GHOST_DETECTION_AVAILABLE_FIELD_NUMBER: _ClassVar[int]
    GHOST_UNAVAILABLE_REASON_FIELD_NUMBER: _ClassVar[int]
    ghost_check_ids: _containers.RepeatedScalarFieldContainer[str]
    unsupervised_plant: _containers.RepeatedScalarFieldContainer[str]
    available: bool
    unavailable_reason: str
    computed_at: _timestamp_pb2.Timestamp
    out_of_scope_check_ids: _containers.RepeatedScalarFieldContainer[str]
    ghost_detection_available: bool
    ghost_unavailable_reason: str
    def __init__(self, ghost_check_ids: _Optional[_Iterable[str]] = ..., unsupervised_plant: _Optional[_Iterable[str]] = ..., available: _Optional[bool] = ..., unavailable_reason: _Optional[str] = ..., computed_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., out_of_scope_check_ids: _Optional[_Iterable[str]] = ..., ghost_detection_available: _Optional[bool] = ..., ghost_unavailable_reason: _Optional[str] = ...) -> None: ...

class GetReconcileRequest(_message.Message):
    __slots__ = ()
    def __init__(self) -> None: ...

class GetReconcileResponse(_message.Message):
    __slots__ = ("reconcile",)
    RECONCILE_FIELD_NUMBER: _ClassVar[int]
    reconcile: Reconcile
    def __init__(self, reconcile: _Optional[_Union[Reconcile, _Mapping]] = ...) -> None: ...

class Shelf(_message.Message):
    __slots__ = ("check_id", "reason", "expires_at", "set_by", "created_at")
    CHECK_ID_FIELD_NUMBER: _ClassVar[int]
    REASON_FIELD_NUMBER: _ClassVar[int]
    EXPIRES_AT_FIELD_NUMBER: _ClassVar[int]
    SET_BY_FIELD_NUMBER: _ClassVar[int]
    CREATED_AT_FIELD_NUMBER: _ClassVar[int]
    check_id: str
    reason: str
    expires_at: _timestamp_pb2.Timestamp
    set_by: str
    created_at: _timestamp_pb2.Timestamp
    def __init__(self, check_id: _Optional[str] = ..., reason: _Optional[str] = ..., expires_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., set_by: _Optional[str] = ..., created_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ...) -> None: ...

class ListShelvesRequest(_message.Message):
    __slots__ = ("include_expired",)
    INCLUDE_EXPIRED_FIELD_NUMBER: _ClassVar[int]
    include_expired: bool
    def __init__(self, include_expired: _Optional[bool] = ...) -> None: ...

class ListShelvesResponse(_message.Message):
    __slots__ = ("shelves",)
    SHELVES_FIELD_NUMBER: _ClassVar[int]
    shelves: _containers.RepeatedCompositeFieldContainer[Shelf]
    def __init__(self, shelves: _Optional[_Iterable[_Union[Shelf, _Mapping]]] = ...) -> None: ...

class Saturation(_message.Message):
    __slots__ = ("check_id", "transitioned", "transition_count", "current_status", "saturated")
    CHECK_ID_FIELD_NUMBER: _ClassVar[int]
    TRANSITIONED_FIELD_NUMBER: _ClassVar[int]
    TRANSITION_COUNT_FIELD_NUMBER: _ClassVar[int]
    CURRENT_STATUS_FIELD_NUMBER: _ClassVar[int]
    SATURATED_FIELD_NUMBER: _ClassVar[int]
    check_id: str
    transitioned: bool
    transition_count: int
    current_status: CheckStatus
    saturated: bool
    def __init__(self, check_id: _Optional[str] = ..., transitioned: _Optional[bool] = ..., transition_count: _Optional[int] = ..., current_status: _Optional[_Union[CheckStatus, str]] = ..., saturated: _Optional[bool] = ...) -> None: ...

class ListSaturationRequest(_message.Message):
    __slots__ = ("window_hours",)
    WINDOW_HOURS_FIELD_NUMBER: _ClassVar[int]
    window_hours: int
    def __init__(self, window_hours: _Optional[int] = ...) -> None: ...

class ListSaturationResponse(_message.Message):
    __slots__ = ("saturations", "window_hours", "computed_at", "truncated")
    SATURATIONS_FIELD_NUMBER: _ClassVar[int]
    WINDOW_HOURS_FIELD_NUMBER: _ClassVar[int]
    COMPUTED_AT_FIELD_NUMBER: _ClassVar[int]
    TRUNCATED_FIELD_NUMBER: _ClassVar[int]
    saturations: _containers.RepeatedCompositeFieldContainer[Saturation]
    window_hours: int
    computed_at: _timestamp_pb2.Timestamp
    truncated: bool
    def __init__(self, saturations: _Optional[_Iterable[_Union[Saturation, _Mapping]]] = ..., window_hours: _Optional[int] = ..., computed_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., truncated: _Optional[bool] = ...) -> None: ...

class GetSaturationRequest(_message.Message):
    __slots__ = ("check_id", "window_hours")
    CHECK_ID_FIELD_NUMBER: _ClassVar[int]
    WINDOW_HOURS_FIELD_NUMBER: _ClassVar[int]
    check_id: str
    window_hours: int
    def __init__(self, check_id: _Optional[str] = ..., window_hours: _Optional[int] = ...) -> None: ...

class GetSaturationResponse(_message.Message):
    __slots__ = ("check_id", "transitioned", "transition_count", "window_hours", "computed_at")
    CHECK_ID_FIELD_NUMBER: _ClassVar[int]
    TRANSITIONED_FIELD_NUMBER: _ClassVar[int]
    TRANSITION_COUNT_FIELD_NUMBER: _ClassVar[int]
    WINDOW_HOURS_FIELD_NUMBER: _ClassVar[int]
    COMPUTED_AT_FIELD_NUMBER: _ClassVar[int]
    check_id: str
    transitioned: bool
    transition_count: int
    window_hours: int
    computed_at: _timestamp_pb2.Timestamp
    def __init__(self, check_id: _Optional[str] = ..., transitioned: _Optional[bool] = ..., transition_count: _Optional[int] = ..., window_hours: _Optional[int] = ..., computed_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ...) -> None: ...
