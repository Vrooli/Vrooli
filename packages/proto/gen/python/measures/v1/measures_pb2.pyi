import datetime

from buf.validate import validate_pb2 as _validate_pb2
from google.protobuf import timestamp_pb2 as _timestamp_pb2
from google.protobuf.internal import enum_type_wrapper as _enum_type_wrapper
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Mapping as _Mapping
from typing import ClassVar as _ClassVar, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class TimeWindowToken(int, metaclass=_enum_type_wrapper.EnumTypeWrapper):
    __slots__ = ()
    TIME_WINDOW_TOKEN_UNSPECIFIED: _ClassVar[TimeWindowToken]
    TIME_WINDOW_TOKEN_THIS_WEEK: _ClassVar[TimeWindowToken]
    TIME_WINDOW_TOKEN_LAST_7D: _ClassVar[TimeWindowToken]
    TIME_WINDOW_TOKEN_LAST_30D: _ClassVar[TimeWindowToken]
    TIME_WINDOW_TOKEN_THIS_MONTH: _ClassVar[TimeWindowToken]
    TIME_WINDOW_TOKEN_LAST_MONTH: _ClassVar[TimeWindowToken]
    TIME_WINDOW_TOKEN_THIS_QUARTER: _ClassVar[TimeWindowToken]
TIME_WINDOW_TOKEN_UNSPECIFIED: TimeWindowToken
TIME_WINDOW_TOKEN_THIS_WEEK: TimeWindowToken
TIME_WINDOW_TOKEN_LAST_7D: TimeWindowToken
TIME_WINDOW_TOKEN_LAST_30D: TimeWindowToken
TIME_WINDOW_TOKEN_THIS_MONTH: TimeWindowToken
TIME_WINDOW_TOKEN_LAST_MONTH: TimeWindowToken
TIME_WINDOW_TOKEN_THIS_QUARTER: TimeWindowToken

class CustomRange(_message.Message):
    __slots__ = ("to",)
    FROM_FIELD_NUMBER: _ClassVar[int]
    TO_FIELD_NUMBER: _ClassVar[int]
    to: _timestamp_pb2.Timestamp
    def __init__(self, to: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., **kwargs) -> None: ...

class TimeWindow(_message.Message):
    __slots__ = ("token", "custom")
    TOKEN_FIELD_NUMBER: _ClassVar[int]
    CUSTOM_FIELD_NUMBER: _ClassVar[int]
    token: TimeWindowToken
    custom: CustomRange
    def __init__(self, token: _Optional[_Union[TimeWindowToken, str]] = ..., custom: _Optional[_Union[CustomRange, _Mapping]] = ...) -> None: ...
