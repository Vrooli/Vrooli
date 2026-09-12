import datetime

from buf.validate import validate_pb2 as _validate_pb2
from google.protobuf import timestamp_pb2 as _timestamp_pb2
from google.protobuf.internal import containers as _containers
from google.protobuf.internal import enum_type_wrapper as _enum_type_wrapper
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Iterable as _Iterable, Mapping as _Mapping
from typing import ClassVar as _ClassVar, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class Basis(int, metaclass=_enum_type_wrapper.EnumTypeWrapper):
    __slots__ = ()
    BASIS_UNSPECIFIED: _ClassVar[Basis]
    BASIS_AUTHORITATIVE: _ClassVar[Basis]
    BASIS_DERIVED: _ClassVar[Basis]
    BASIS_OPERATOR_ASSERTED: _ClassVar[Basis]
    BASIS_PROJECTED: _ClassVar[Basis]
BASIS_UNSPECIFIED: Basis
BASIS_AUTHORITATIVE: Basis
BASIS_DERIVED: Basis
BASIS_OPERATOR_ASSERTED: Basis
BASIS_PROJECTED: Basis

class MoneyEvent(_message.Message):
    __slots__ = ("id", "external_id", "adapter_id", "account_id", "book_id", "amount_minor", "currency", "occurred_at", "fetched_at", "basis", "description", "category")
    ID_FIELD_NUMBER: _ClassVar[int]
    EXTERNAL_ID_FIELD_NUMBER: _ClassVar[int]
    ADAPTER_ID_FIELD_NUMBER: _ClassVar[int]
    ACCOUNT_ID_FIELD_NUMBER: _ClassVar[int]
    BOOK_ID_FIELD_NUMBER: _ClassVar[int]
    AMOUNT_MINOR_FIELD_NUMBER: _ClassVar[int]
    CURRENCY_FIELD_NUMBER: _ClassVar[int]
    OCCURRED_AT_FIELD_NUMBER: _ClassVar[int]
    FETCHED_AT_FIELD_NUMBER: _ClassVar[int]
    BASIS_FIELD_NUMBER: _ClassVar[int]
    DESCRIPTION_FIELD_NUMBER: _ClassVar[int]
    CATEGORY_FIELD_NUMBER: _ClassVar[int]
    id: str
    external_id: str
    adapter_id: str
    account_id: str
    book_id: str
    amount_minor: int
    currency: str
    occurred_at: _timestamp_pb2.Timestamp
    fetched_at: _timestamp_pb2.Timestamp
    basis: Basis
    description: str
    category: str
    def __init__(self, id: _Optional[str] = ..., external_id: _Optional[str] = ..., adapter_id: _Optional[str] = ..., account_id: _Optional[str] = ..., book_id: _Optional[str] = ..., amount_minor: _Optional[int] = ..., currency: _Optional[str] = ..., occurred_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., fetched_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., basis: _Optional[_Union[Basis, str]] = ..., description: _Optional[str] = ..., category: _Optional[str] = ...) -> None: ...

class AuditEntry(_message.Message):
    __slots__ = ("id", "entity_type", "entity_id", "actor", "reason", "prior_value", "created_at")
    ID_FIELD_NUMBER: _ClassVar[int]
    ENTITY_TYPE_FIELD_NUMBER: _ClassVar[int]
    ENTITY_ID_FIELD_NUMBER: _ClassVar[int]
    ACTOR_FIELD_NUMBER: _ClassVar[int]
    REASON_FIELD_NUMBER: _ClassVar[int]
    PRIOR_VALUE_FIELD_NUMBER: _ClassVar[int]
    CREATED_AT_FIELD_NUMBER: _ClassVar[int]
    id: str
    entity_type: str
    entity_id: str
    actor: str
    reason: str
    prior_value: str
    created_at: _timestamp_pb2.Timestamp
    def __init__(self, id: _Optional[str] = ..., entity_type: _Optional[str] = ..., entity_id: _Optional[str] = ..., actor: _Optional[str] = ..., reason: _Optional[str] = ..., prior_value: _Optional[str] = ..., created_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ...) -> None: ...

class Posting(_message.Message):
    __slots__ = ("id", "event", "reversal_of", "actor", "audit")
    ID_FIELD_NUMBER: _ClassVar[int]
    EVENT_FIELD_NUMBER: _ClassVar[int]
    REVERSAL_OF_FIELD_NUMBER: _ClassVar[int]
    ACTOR_FIELD_NUMBER: _ClassVar[int]
    AUDIT_FIELD_NUMBER: _ClassVar[int]
    id: str
    event: MoneyEvent
    reversal_of: str
    actor: str
    audit: _containers.RepeatedCompositeFieldContainer[AuditEntry]
    def __init__(self, id: _Optional[str] = ..., event: _Optional[_Union[MoneyEvent, _Mapping]] = ..., reversal_of: _Optional[str] = ..., actor: _Optional[str] = ..., audit: _Optional[_Iterable[_Union[AuditEntry, _Mapping]]] = ...) -> None: ...
