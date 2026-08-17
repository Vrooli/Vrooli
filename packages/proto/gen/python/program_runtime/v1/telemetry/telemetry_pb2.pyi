from google.protobuf.internal import containers as _containers
from google.protobuf.internal import enum_type_wrapper as _enum_type_wrapper
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Iterable as _Iterable, Mapping as _Mapping
from typing import ClassVar as _ClassVar, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class EventKind(int, metaclass=_enum_type_wrapper.EnumTypeWrapper):
    __slots__ = ()
    EVENT_KIND_UNSPECIFIED: _ClassVar[EventKind]
    PROGRAM_SUBMITTED: _ClassVar[EventKind]
    PROGRAM_SUCCEEDED: _ClassVar[EventKind]
    PROGRAM_FAILED: _ClassVar[EventKind]
    BINDING_REFUSED: _ClassVar[EventKind]
    SESSION_RECLAIMED: _ClassVar[EventKind]
    BINDING_INVOKED: _ClassVar[EventKind]
    PROGRAM_ACCEPTED: _ClassVar[EventKind]
    PROGRAM_RUNNING: _ClassVar[EventKind]
    PROGRAM_CANCELLED: _ClassVar[EventKind]
EVENT_KIND_UNSPECIFIED: EventKind
PROGRAM_SUBMITTED: EventKind
PROGRAM_SUCCEEDED: EventKind
PROGRAM_FAILED: EventKind
BINDING_REFUSED: EventKind
SESSION_RECLAIMED: EventKind
BINDING_INVOKED: EventKind
PROGRAM_ACCEPTED: EventKind
PROGRAM_RUNNING: EventKind
PROGRAM_CANCELLED: EventKind

class ProgramEvent(_message.Message):
    __slots__ = ("event_id", "occurred_at", "kind", "program_id", "session_id", "binding_id", "effect", "provenance", "failure_shape", "context_bytes", "reason", "failure_location", "sequence")
    EVENT_ID_FIELD_NUMBER: _ClassVar[int]
    OCCURRED_AT_FIELD_NUMBER: _ClassVar[int]
    KIND_FIELD_NUMBER: _ClassVar[int]
    PROGRAM_ID_FIELD_NUMBER: _ClassVar[int]
    SESSION_ID_FIELD_NUMBER: _ClassVar[int]
    BINDING_ID_FIELD_NUMBER: _ClassVar[int]
    EFFECT_FIELD_NUMBER: _ClassVar[int]
    PROVENANCE_FIELD_NUMBER: _ClassVar[int]
    FAILURE_SHAPE_FIELD_NUMBER: _ClassVar[int]
    CONTEXT_BYTES_FIELD_NUMBER: _ClassVar[int]
    REASON_FIELD_NUMBER: _ClassVar[int]
    FAILURE_LOCATION_FIELD_NUMBER: _ClassVar[int]
    SEQUENCE_FIELD_NUMBER: _ClassVar[int]
    event_id: str
    occurred_at: str
    kind: EventKind
    program_id: str
    session_id: str
    binding_id: str
    effect: str
    provenance: str
    failure_shape: str
    context_bytes: int
    reason: str
    failure_location: str
    sequence: int
    def __init__(self, event_id: _Optional[str] = ..., occurred_at: _Optional[str] = ..., kind: _Optional[_Union[EventKind, str]] = ..., program_id: _Optional[str] = ..., session_id: _Optional[str] = ..., binding_id: _Optional[str] = ..., effect: _Optional[str] = ..., provenance: _Optional[str] = ..., failure_shape: _Optional[str] = ..., context_bytes: _Optional[int] = ..., reason: _Optional[str] = ..., failure_location: _Optional[str] = ..., sequence: _Optional[int] = ...) -> None: ...

class ListEventsRequest(_message.Message):
    __slots__ = ("session_id", "kind")
    SESSION_ID_FIELD_NUMBER: _ClassVar[int]
    KIND_FIELD_NUMBER: _ClassVar[int]
    session_id: str
    kind: EventKind
    def __init__(self, session_id: _Optional[str] = ..., kind: _Optional[_Union[EventKind, str]] = ...) -> None: ...

class ListEventsResponse(_message.Message):
    __slots__ = ("events",)
    EVENTS_FIELD_NUMBER: _ClassVar[int]
    events: _containers.RepeatedCompositeFieldContainer[ProgramEvent]
    def __init__(self, events: _Optional[_Iterable[_Union[ProgramEvent, _Mapping]]] = ...) -> None: ...
