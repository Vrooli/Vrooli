import datetime

from google.protobuf import timestamp_pb2 as _timestamp_pb2
from google.protobuf.internal import containers as _containers
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Iterable as _Iterable, Mapping as _Mapping
from typing import ClassVar as _ClassVar, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class DocumentBinding(_message.Message):
    __slots__ = ("id", "persona_id", "document_id", "document_kind", "valid_until", "created_at")
    ID_FIELD_NUMBER: _ClassVar[int]
    PERSONA_ID_FIELD_NUMBER: _ClassVar[int]
    DOCUMENT_ID_FIELD_NUMBER: _ClassVar[int]
    DOCUMENT_KIND_FIELD_NUMBER: _ClassVar[int]
    VALID_UNTIL_FIELD_NUMBER: _ClassVar[int]
    CREATED_AT_FIELD_NUMBER: _ClassVar[int]
    id: str
    persona_id: str
    document_id: str
    document_kind: str
    valid_until: _timestamp_pb2.Timestamp
    created_at: _timestamp_pb2.Timestamp
    def __init__(self, id: _Optional[str] = ..., persona_id: _Optional[str] = ..., document_id: _Optional[str] = ..., document_kind: _Optional[str] = ..., valid_until: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., created_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ...) -> None: ...

class BindDocumentRequest(_message.Message):
    __slots__ = ("persona_id", "document_id", "document_kind", "valid_until")
    PERSONA_ID_FIELD_NUMBER: _ClassVar[int]
    DOCUMENT_ID_FIELD_NUMBER: _ClassVar[int]
    DOCUMENT_KIND_FIELD_NUMBER: _ClassVar[int]
    VALID_UNTIL_FIELD_NUMBER: _ClassVar[int]
    persona_id: str
    document_id: str
    document_kind: str
    valid_until: _timestamp_pb2.Timestamp
    def __init__(self, persona_id: _Optional[str] = ..., document_id: _Optional[str] = ..., document_kind: _Optional[str] = ..., valid_until: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ...) -> None: ...

class BindDocumentResponse(_message.Message):
    __slots__ = ("binding",)
    BINDING_FIELD_NUMBER: _ClassVar[int]
    binding: DocumentBinding
    def __init__(self, binding: _Optional[_Union[DocumentBinding, _Mapping]] = ...) -> None: ...

class ListBindingsRequest(_message.Message):
    __slots__ = ("persona_id",)
    PERSONA_ID_FIELD_NUMBER: _ClassVar[int]
    persona_id: str
    def __init__(self, persona_id: _Optional[str] = ...) -> None: ...

class ListBindingsResponse(_message.Message):
    __slots__ = ("bindings",)
    BINDINGS_FIELD_NUMBER: _ClassVar[int]
    bindings: _containers.RepeatedCompositeFieldContainer[DocumentBinding]
    def __init__(self, bindings: _Optional[_Iterable[_Union[DocumentBinding, _Mapping]]] = ...) -> None: ...

class ReleaseIntoHandoffRequest(_message.Message):
    __slots__ = ("persona_id", "document_id", "handoff_id")
    PERSONA_ID_FIELD_NUMBER: _ClassVar[int]
    DOCUMENT_ID_FIELD_NUMBER: _ClassVar[int]
    HANDOFF_ID_FIELD_NUMBER: _ClassVar[int]
    persona_id: str
    document_id: str
    handoff_id: str
    def __init__(self, persona_id: _Optional[str] = ..., document_id: _Optional[str] = ..., handoff_id: _Optional[str] = ...) -> None: ...

class ReleaseIntoHandoffResponse(_message.Message):
    __slots__ = ("release_id", "handoff_id", "document_id", "released_at")
    RELEASE_ID_FIELD_NUMBER: _ClassVar[int]
    HANDOFF_ID_FIELD_NUMBER: _ClassVar[int]
    DOCUMENT_ID_FIELD_NUMBER: _ClassVar[int]
    RELEASED_AT_FIELD_NUMBER: _ClassVar[int]
    release_id: str
    handoff_id: str
    document_id: str
    released_at: _timestamp_pb2.Timestamp
    def __init__(self, release_id: _Optional[str] = ..., handoff_id: _Optional[str] = ..., document_id: _Optional[str] = ..., released_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ...) -> None: ...
