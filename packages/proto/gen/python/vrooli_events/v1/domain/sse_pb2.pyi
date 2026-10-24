import datetime

from google.protobuf import timestamp_pb2 as _timestamp_pb2
from vrooli_events.v1.domain import envelope_pb2 as _envelope_pb2
from vrooli_events.v1.domain import policy_pb2 as _policy_pb2
from google.protobuf.internal import containers as _containers
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Iterable as _Iterable, Mapping as _Mapping
from typing import ClassVar as _ClassVar, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class EventNotification(_message.Message):
    __slots__ = ("stream_sequence", "envelope")
    STREAM_SEQUENCE_FIELD_NUMBER: _ClassVar[int]
    ENVELOPE_FIELD_NUMBER: _ClassVar[int]
    stream_sequence: int
    envelope: _envelope_pb2.EventEnvelope
    def __init__(self, stream_sequence: _Optional[int] = ..., envelope: _Optional[_Union[_envelope_pb2.EventEnvelope, _Mapping]] = ...) -> None: ...

class PolicySnapshot(_message.Message):
    __slots__ = ("version", "generated_at", "receipt_capture_policies")
    VERSION_FIELD_NUMBER: _ClassVar[int]
    GENERATED_AT_FIELD_NUMBER: _ClassVar[int]
    RECEIPT_CAPTURE_POLICIES_FIELD_NUMBER: _ClassVar[int]
    version: str
    generated_at: _timestamp_pb2.Timestamp
    receipt_capture_policies: _containers.RepeatedCompositeFieldContainer[_policy_pb2.ReceiptCapturePolicy]
    def __init__(self, version: _Optional[str] = ..., generated_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., receipt_capture_policies: _Optional[_Iterable[_Union[_policy_pb2.ReceiptCapturePolicy, _Mapping]]] = ...) -> None: ...

class HeartbeatMessage(_message.Message):
    __slots__ = ("timestamp", "dropped_count")
    TIMESTAMP_FIELD_NUMBER: _ClassVar[int]
    DROPPED_COUNT_FIELD_NUMBER: _ClassVar[int]
    timestamp: _timestamp_pb2.Timestamp
    dropped_count: int
    def __init__(self, timestamp: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., dropped_count: _Optional[int] = ...) -> None: ...
