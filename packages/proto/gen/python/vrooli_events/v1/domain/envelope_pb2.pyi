import datetime

from google.protobuf import any_pb2 as _any_pb2
from google.protobuf import timestamp_pb2 as _timestamp_pb2
from google.protobuf.internal import containers as _containers
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Mapping as _Mapping
from typing import ClassVar as _ClassVar, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class EventEnvelope(_message.Message):
    __slots__ = ("event_id", "source_scenario", "target_scenario", "event_type", "timestamp", "correlation_id", "payload", "metadata")
    class MetadataEntry(_message.Message):
        __slots__ = ("key", "value")
        KEY_FIELD_NUMBER: _ClassVar[int]
        VALUE_FIELD_NUMBER: _ClassVar[int]
        key: str
        value: str
        def __init__(self, key: _Optional[str] = ..., value: _Optional[str] = ...) -> None: ...
    EVENT_ID_FIELD_NUMBER: _ClassVar[int]
    SOURCE_SCENARIO_FIELD_NUMBER: _ClassVar[int]
    TARGET_SCENARIO_FIELD_NUMBER: _ClassVar[int]
    EVENT_TYPE_FIELD_NUMBER: _ClassVar[int]
    TIMESTAMP_FIELD_NUMBER: _ClassVar[int]
    CORRELATION_ID_FIELD_NUMBER: _ClassVar[int]
    PAYLOAD_FIELD_NUMBER: _ClassVar[int]
    METADATA_FIELD_NUMBER: _ClassVar[int]
    event_id: str
    source_scenario: str
    target_scenario: str
    event_type: str
    timestamp: _timestamp_pb2.Timestamp
    correlation_id: str
    payload: _any_pb2.Any
    metadata: _containers.ScalarMap[str, str]
    def __init__(self, event_id: _Optional[str] = ..., source_scenario: _Optional[str] = ..., target_scenario: _Optional[str] = ..., event_type: _Optional[str] = ..., timestamp: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., correlation_id: _Optional[str] = ..., payload: _Optional[_Union[_any_pb2.Any, _Mapping]] = ..., metadata: _Optional[_Mapping[str, str]] = ...) -> None: ...
