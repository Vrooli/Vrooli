from buf.validate import validate_pb2 as _validate_pb2
from google.protobuf.internal import containers as _containers
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Iterable as _Iterable, Mapping as _Mapping
from typing import ClassVar as _ClassVar, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class Capture(_message.Message):
    __slots__ = ("id", "text", "attachments", "created", "status", "classification", "note")
    ID_FIELD_NUMBER: _ClassVar[int]
    TEXT_FIELD_NUMBER: _ClassVar[int]
    ATTACHMENTS_FIELD_NUMBER: _ClassVar[int]
    CREATED_FIELD_NUMBER: _ClassVar[int]
    STATUS_FIELD_NUMBER: _ClassVar[int]
    CLASSIFICATION_FIELD_NUMBER: _ClassVar[int]
    NOTE_FIELD_NUMBER: _ClassVar[int]
    id: str
    text: str
    attachments: _containers.RepeatedScalarFieldContainer[str]
    created: str
    status: str
    classification: CaptureClassification
    note: str
    def __init__(self, id: _Optional[str] = ..., text: _Optional[str] = ..., attachments: _Optional[_Iterable[str]] = ..., created: _Optional[str] = ..., status: _Optional[str] = ..., classification: _Optional[_Union[CaptureClassification, _Mapping]] = ..., note: _Optional[str] = ...) -> None: ...

class CaptureClassification(_message.Message):
    __slots__ = ("items", "classified_at")
    ITEMS_FIELD_NUMBER: _ClassVar[int]
    CLASSIFIED_AT_FIELD_NUMBER: _ClassVar[int]
    items: _containers.RepeatedCompositeFieldContainer[CaptureClassificationItem]
    classified_at: str
    def __init__(self, items: _Optional[_Iterable[_Union[CaptureClassificationItem, _Mapping]]] = ..., classified_at: _Optional[str] = ...) -> None: ...

class CaptureClassificationItem(_message.Message):
    __slots__ = ("kind", "title", "description", "priority", "tags", "confidence")
    KIND_FIELD_NUMBER: _ClassVar[int]
    TITLE_FIELD_NUMBER: _ClassVar[int]
    DESCRIPTION_FIELD_NUMBER: _ClassVar[int]
    PRIORITY_FIELD_NUMBER: _ClassVar[int]
    TAGS_FIELD_NUMBER: _ClassVar[int]
    CONFIDENCE_FIELD_NUMBER: _ClassVar[int]
    kind: str
    title: str
    description: str
    priority: int
    tags: _containers.RepeatedScalarFieldContainer[str]
    confidence: float
    def __init__(self, kind: _Optional[str] = ..., title: _Optional[str] = ..., description: _Optional[str] = ..., priority: _Optional[int] = ..., tags: _Optional[_Iterable[str]] = ..., confidence: _Optional[float] = ...) -> None: ...
