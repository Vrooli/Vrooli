import datetime

from google.protobuf import timestamp_pb2 as _timestamp_pb2
from google.protobuf.internal import containers as _containers
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Iterable as _Iterable, Mapping as _Mapping
from typing import ClassVar as _ClassVar, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class ProviderConnection(_message.Message):
    __slots__ = ("id", "provider", "display_name", "source_kind", "status", "health_message", "read_only", "calendar_count", "imported_event_count", "busy_minutes", "revision", "last_sync_at")
    ID_FIELD_NUMBER: _ClassVar[int]
    PROVIDER_FIELD_NUMBER: _ClassVar[int]
    DISPLAY_NAME_FIELD_NUMBER: _ClassVar[int]
    SOURCE_KIND_FIELD_NUMBER: _ClassVar[int]
    STATUS_FIELD_NUMBER: _ClassVar[int]
    HEALTH_MESSAGE_FIELD_NUMBER: _ClassVar[int]
    READ_ONLY_FIELD_NUMBER: _ClassVar[int]
    CALENDAR_COUNT_FIELD_NUMBER: _ClassVar[int]
    IMPORTED_EVENT_COUNT_FIELD_NUMBER: _ClassVar[int]
    BUSY_MINUTES_FIELD_NUMBER: _ClassVar[int]
    REVISION_FIELD_NUMBER: _ClassVar[int]
    LAST_SYNC_AT_FIELD_NUMBER: _ClassVar[int]
    id: str
    provider: str
    display_name: str
    source_kind: str
    status: str
    health_message: str
    read_only: bool
    calendar_count: int
    imported_event_count: int
    busy_minutes: int
    revision: int
    last_sync_at: _timestamp_pb2.Timestamp
    def __init__(self, id: _Optional[str] = ..., provider: _Optional[str] = ..., display_name: _Optional[str] = ..., source_kind: _Optional[str] = ..., status: _Optional[str] = ..., health_message: _Optional[str] = ..., read_only: _Optional[bool] = ..., calendar_count: _Optional[int] = ..., imported_event_count: _Optional[int] = ..., busy_minutes: _Optional[int] = ..., revision: _Optional[int] = ..., last_sync_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ...) -> None: ...

class ListConnectionsRequest(_message.Message):
    __slots__ = ()
    def __init__(self) -> None: ...

class ListConnectionsResponse(_message.Message):
    __slots__ = ("connections",)
    CONNECTIONS_FIELD_NUMBER: _ClassVar[int]
    connections: _containers.RepeatedCompositeFieldContainer[ProviderConnection]
    def __init__(self, connections: _Optional[_Iterable[_Union[ProviderConnection, _Mapping]]] = ...) -> None: ...

class CreateFixtureConnectionRequest(_message.Message):
    __slots__ = ("display_name",)
    DISPLAY_NAME_FIELD_NUMBER: _ClassVar[int]
    display_name: str
    def __init__(self, display_name: _Optional[str] = ...) -> None: ...

class CreateFixtureConnectionResponse(_message.Message):
    __slots__ = ("connection",)
    CONNECTION_FIELD_NUMBER: _ClassVar[int]
    connection: ProviderConnection
    def __init__(self, connection: _Optional[_Union[ProviderConnection, _Mapping]] = ...) -> None: ...

class SyncConnectionRequest(_message.Message):
    __slots__ = ("id", "expected_revision")
    ID_FIELD_NUMBER: _ClassVar[int]
    EXPECTED_REVISION_FIELD_NUMBER: _ClassVar[int]
    id: str
    expected_revision: int
    def __init__(self, id: _Optional[str] = ..., expected_revision: _Optional[int] = ...) -> None: ...

class SyncConnectionResponse(_message.Message):
    __slots__ = ("connection", "imported_event_count", "busy_minutes")
    CONNECTION_FIELD_NUMBER: _ClassVar[int]
    IMPORTED_EVENT_COUNT_FIELD_NUMBER: _ClassVar[int]
    BUSY_MINUTES_FIELD_NUMBER: _ClassVar[int]
    connection: ProviderConnection
    imported_event_count: int
    busy_minutes: int
    def __init__(self, connection: _Optional[_Union[ProviderConnection, _Mapping]] = ..., imported_event_count: _Optional[int] = ..., busy_minutes: _Optional[int] = ...) -> None: ...

class DisconnectConnectionRequest(_message.Message):
    __slots__ = ("id", "expected_revision")
    ID_FIELD_NUMBER: _ClassVar[int]
    EXPECTED_REVISION_FIELD_NUMBER: _ClassVar[int]
    id: str
    expected_revision: int
    def __init__(self, id: _Optional[str] = ..., expected_revision: _Optional[int] = ...) -> None: ...

class DisconnectConnectionResponse(_message.Message):
    __slots__ = ("connection",)
    CONNECTION_FIELD_NUMBER: _ClassVar[int]
    connection: ProviderConnection
    def __init__(self, connection: _Optional[_Union[ProviderConnection, _Mapping]] = ...) -> None: ...
