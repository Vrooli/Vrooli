import datetime

from google.protobuf import timestamp_pb2 as _timestamp_pb2
from google.protobuf.internal import containers as _containers
from google.protobuf.internal import enum_type_wrapper as _enum_type_wrapper
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Iterable as _Iterable, Mapping as _Mapping
from typing import ClassVar as _ClassVar, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class IntentState(int, metaclass=_enum_type_wrapper.EnumTypeWrapper):
    __slots__ = ()
    INTENT_STATE_UNSPECIFIED: _ClassVar[IntentState]
    INTENT_STATE_RESERVING: _ClassVar[IntentState]
    INTENT_STATE_OPEN: _ClassVar[IntentState]
    INTENT_STATE_FULFILLED: _ClassVar[IntentState]
    INTENT_STATE_REFUSED: _ClassVar[IntentState]
    INTENT_STATE_ABANDONED: _ClassVar[IntentState]
INTENT_STATE_UNSPECIFIED: IntentState
INTENT_STATE_RESERVING: IntentState
INTENT_STATE_OPEN: IntentState
INTENT_STATE_FULFILLED: IntentState
INTENT_STATE_REFUSED: IntentState
INTENT_STATE_ABANDONED: IntentState

class Intent(_message.Message):
    __slots__ = ("id", "idempotency_key", "requested_by", "provider", "reservation_id", "instance_id", "state", "created_at", "resolved_at")
    ID_FIELD_NUMBER: _ClassVar[int]
    IDEMPOTENCY_KEY_FIELD_NUMBER: _ClassVar[int]
    REQUESTED_BY_FIELD_NUMBER: _ClassVar[int]
    PROVIDER_FIELD_NUMBER: _ClassVar[int]
    RESERVATION_ID_FIELD_NUMBER: _ClassVar[int]
    INSTANCE_ID_FIELD_NUMBER: _ClassVar[int]
    STATE_FIELD_NUMBER: _ClassVar[int]
    CREATED_AT_FIELD_NUMBER: _ClassVar[int]
    RESOLVED_AT_FIELD_NUMBER: _ClassVar[int]
    id: str
    idempotency_key: str
    requested_by: str
    provider: str
    reservation_id: str
    instance_id: str
    state: IntentState
    created_at: _timestamp_pb2.Timestamp
    resolved_at: _timestamp_pb2.Timestamp
    def __init__(self, id: _Optional[str] = ..., idempotency_key: _Optional[str] = ..., requested_by: _Optional[str] = ..., provider: _Optional[str] = ..., reservation_id: _Optional[str] = ..., instance_id: _Optional[str] = ..., state: _Optional[_Union[IntentState, str]] = ..., created_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., resolved_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ...) -> None: ...

class ListIntentsRequest(_message.Message):
    __slots__ = ("state", "provider")
    STATE_FIELD_NUMBER: _ClassVar[int]
    PROVIDER_FIELD_NUMBER: _ClassVar[int]
    state: str
    provider: str
    def __init__(self, state: _Optional[str] = ..., provider: _Optional[str] = ...) -> None: ...

class ListIntentsResponse(_message.Message):
    __slots__ = ("intents",)
    INTENTS_FIELD_NUMBER: _ClassVar[int]
    intents: _containers.RepeatedCompositeFieldContainer[Intent]
    def __init__(self, intents: _Optional[_Iterable[_Union[Intent, _Mapping]]] = ...) -> None: ...

class GetIntentRequest(_message.Message):
    __slots__ = ("id_or_key",)
    ID_OR_KEY_FIELD_NUMBER: _ClassVar[int]
    id_or_key: str
    def __init__(self, id_or_key: _Optional[str] = ...) -> None: ...

class GetIntentResponse(_message.Message):
    __slots__ = ("intent",)
    INTENT_FIELD_NUMBER: _ClassVar[int]
    intent: Intent
    def __init__(self, intent: _Optional[_Union[Intent, _Mapping]] = ...) -> None: ...
