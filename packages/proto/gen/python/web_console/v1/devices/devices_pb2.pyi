import datetime

from google.protobuf import timestamp_pb2 as _timestamp_pb2
from google.protobuf.internal import containers as _containers
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Iterable as _Iterable, Mapping as _Mapping
from typing import ClassVar as _ClassVar, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class ListRequest(_message.Message):
    __slots__ = ("self_device_id",)
    SELF_DEVICE_ID_FIELD_NUMBER: _ClassVar[int]
    self_device_id: str
    def __init__(self, self_device_id: _Optional[str] = ...) -> None: ...

class ListResponse(_message.Message):
    __slots__ = ("devices",)
    DEVICES_FIELD_NUMBER: _ClassVar[int]
    devices: _containers.RepeatedCompositeFieldContainer[Device]
    def __init__(self, devices: _Optional[_Iterable[_Union[Device, _Mapping]]] = ...) -> None: ...

class SessionAttachment(_message.Message):
    __slots__ = ("session_id", "session_name", "holds_lease")
    SESSION_ID_FIELD_NUMBER: _ClassVar[int]
    SESSION_NAME_FIELD_NUMBER: _ClassVar[int]
    HOLDS_LEASE_FIELD_NUMBER: _ClassVar[int]
    session_id: str
    session_name: str
    holds_lease: bool
    def __init__(self, session_id: _Optional[str] = ..., session_name: _Optional[str] = ..., holds_lease: _Optional[bool] = ...) -> None: ...

class Device(_message.Message):
    __slots__ = ("device_id", "device_label", "device_class", "connection_count", "first_seen_at", "sessions", "is_self", "reconnecting")
    DEVICE_ID_FIELD_NUMBER: _ClassVar[int]
    DEVICE_LABEL_FIELD_NUMBER: _ClassVar[int]
    DEVICE_CLASS_FIELD_NUMBER: _ClassVar[int]
    CONNECTION_COUNT_FIELD_NUMBER: _ClassVar[int]
    FIRST_SEEN_AT_FIELD_NUMBER: _ClassVar[int]
    SESSIONS_FIELD_NUMBER: _ClassVar[int]
    IS_SELF_FIELD_NUMBER: _ClassVar[int]
    RECONNECTING_FIELD_NUMBER: _ClassVar[int]
    device_id: str
    device_label: str
    device_class: str
    connection_count: int
    first_seen_at: _timestamp_pb2.Timestamp
    sessions: _containers.RepeatedCompositeFieldContainer[SessionAttachment]
    is_self: bool
    reconnecting: bool
    def __init__(self, device_id: _Optional[str] = ..., device_label: _Optional[str] = ..., device_class: _Optional[str] = ..., connection_count: _Optional[int] = ..., first_seen_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., sessions: _Optional[_Iterable[_Union[SessionAttachment, _Mapping]]] = ..., is_self: _Optional[bool] = ..., reconnecting: _Optional[bool] = ...) -> None: ...

class DisconnectRequest(_message.Message):
    __slots__ = ("device_id", "connection_id")
    DEVICE_ID_FIELD_NUMBER: _ClassVar[int]
    CONNECTION_ID_FIELD_NUMBER: _ClassVar[int]
    device_id: str
    connection_id: str
    def __init__(self, device_id: _Optional[str] = ..., connection_id: _Optional[str] = ...) -> None: ...

class DisconnectResponse(_message.Message):
    __slots__ = ("closed_connections",)
    CLOSED_CONNECTIONS_FIELD_NUMBER: _ClassVar[int]
    closed_connections: int
    def __init__(self, closed_connections: _Optional[int] = ...) -> None: ...
