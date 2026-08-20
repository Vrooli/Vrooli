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
    __slots__ = ("id", "title", "description", "importance", "category", "interval_seconds")
    ID_FIELD_NUMBER: _ClassVar[int]
    TITLE_FIELD_NUMBER: _ClassVar[int]
    DESCRIPTION_FIELD_NUMBER: _ClassVar[int]
    IMPORTANCE_FIELD_NUMBER: _ClassVar[int]
    CATEGORY_FIELD_NUMBER: _ClassVar[int]
    INTERVAL_SECONDS_FIELD_NUMBER: _ClassVar[int]
    id: str
    title: str
    description: str
    importance: str
    category: str
    interval_seconds: int
    def __init__(self, id: _Optional[str] = ..., title: _Optional[str] = ..., description: _Optional[str] = ..., importance: _Optional[str] = ..., category: _Optional[str] = ..., interval_seconds: _Optional[int] = ...) -> None: ...

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
