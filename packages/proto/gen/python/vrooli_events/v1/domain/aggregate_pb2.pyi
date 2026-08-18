import datetime

from google.protobuf import timestamp_pb2 as _timestamp_pb2
from google.protobuf.internal import containers as _containers
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Iterable as _Iterable, Mapping as _Mapping
from typing import ClassVar as _ClassVar, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class ReceiptAggregateRequest(_message.Message):
    __slots__ = ("since", "until", "target_scenario", "operation")
    SINCE_FIELD_NUMBER: _ClassVar[int]
    UNTIL_FIELD_NUMBER: _ClassVar[int]
    TARGET_SCENARIO_FIELD_NUMBER: _ClassVar[int]
    OPERATION_FIELD_NUMBER: _ClassVar[int]
    since: _timestamp_pb2.Timestamp
    until: _timestamp_pb2.Timestamp
    target_scenario: str
    operation: str
    def __init__(self, since: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., until: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., target_scenario: _Optional[str] = ..., operation: _Optional[str] = ...) -> None: ...

class ReceiptAggregate(_message.Message):
    __slots__ = ("target_scenario", "operation", "invocation_count", "distinct_verified_callers", "unattributed_remainder", "last_invoked_at")
    TARGET_SCENARIO_FIELD_NUMBER: _ClassVar[int]
    OPERATION_FIELD_NUMBER: _ClassVar[int]
    INVOCATION_COUNT_FIELD_NUMBER: _ClassVar[int]
    DISTINCT_VERIFIED_CALLERS_FIELD_NUMBER: _ClassVar[int]
    UNATTRIBUTED_REMAINDER_FIELD_NUMBER: _ClassVar[int]
    LAST_INVOKED_AT_FIELD_NUMBER: _ClassVar[int]
    target_scenario: str
    operation: str
    invocation_count: int
    distinct_verified_callers: int
    unattributed_remainder: int
    last_invoked_at: str
    def __init__(self, target_scenario: _Optional[str] = ..., operation: _Optional[str] = ..., invocation_count: _Optional[int] = ..., distinct_verified_callers: _Optional[int] = ..., unattributed_remainder: _Optional[int] = ..., last_invoked_at: _Optional[str] = ...) -> None: ...

class ReceiptAggregateResponse(_message.Message):
    __slots__ = ("aggregates", "window_since", "window_until")
    AGGREGATES_FIELD_NUMBER: _ClassVar[int]
    WINDOW_SINCE_FIELD_NUMBER: _ClassVar[int]
    WINDOW_UNTIL_FIELD_NUMBER: _ClassVar[int]
    aggregates: _containers.RepeatedCompositeFieldContainer[ReceiptAggregate]
    window_since: str
    window_until: str
    def __init__(self, aggregates: _Optional[_Iterable[_Union[ReceiptAggregate, _Mapping]]] = ..., window_since: _Optional[str] = ..., window_until: _Optional[str] = ...) -> None: ...
