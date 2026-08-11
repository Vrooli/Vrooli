import datetime

from google.protobuf import timestamp_pb2 as _timestamp_pb2
from google.protobuf.internal import containers as _containers
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Iterable as _Iterable, Mapping as _Mapping
from typing import ClassVar as _ClassVar, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class ListAuditRequest(_message.Message):
    __slots__ = ("session_id",)
    SESSION_ID_FIELD_NUMBER: _ClassVar[int]
    session_id: str
    def __init__(self, session_id: _Optional[str] = ...) -> None: ...

class ListAuditResponse(_message.Message):
    __slots__ = ("records",)
    RECORDS_FIELD_NUMBER: _ClassVar[int]
    records: _containers.RepeatedCompositeFieldContainer[AuditRecord]
    def __init__(self, records: _Optional[_Iterable[_Union[AuditRecord, _Mapping]]] = ...) -> None: ...

class AuditRecord(_message.Message):
    __slots__ = ("id", "actor", "device_id", "lease_id", "verb", "outcome", "created_at", "redaction_verified")
    ID_FIELD_NUMBER: _ClassVar[int]
    ACTOR_FIELD_NUMBER: _ClassVar[int]
    DEVICE_ID_FIELD_NUMBER: _ClassVar[int]
    LEASE_ID_FIELD_NUMBER: _ClassVar[int]
    VERB_FIELD_NUMBER: _ClassVar[int]
    OUTCOME_FIELD_NUMBER: _ClassVar[int]
    CREATED_AT_FIELD_NUMBER: _ClassVar[int]
    REDACTION_VERIFIED_FIELD_NUMBER: _ClassVar[int]
    id: str
    actor: str
    device_id: str
    lease_id: str
    verb: str
    outcome: str
    created_at: _timestamp_pb2.Timestamp
    redaction_verified: bool
    def __init__(self, id: _Optional[str] = ..., actor: _Optional[str] = ..., device_id: _Optional[str] = ..., lease_id: _Optional[str] = ..., verb: _Optional[str] = ..., outcome: _Optional[str] = ..., created_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., redaction_verified: _Optional[bool] = ...) -> None: ...
