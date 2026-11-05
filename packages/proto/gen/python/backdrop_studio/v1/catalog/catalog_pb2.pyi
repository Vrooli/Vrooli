from backdrop_studio.v1.shared import shared_pb2 as _shared_pb2
from google.protobuf.internal import containers as _containers
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Iterable as _Iterable, Mapping as _Mapping
from typing import ClassVar as _ClassVar, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class ListStylesRequest(_message.Message):
    __slots__ = ("role", "subject", "treatment", "lineage", "placement")
    ROLE_FIELD_NUMBER: _ClassVar[int]
    SUBJECT_FIELD_NUMBER: _ClassVar[int]
    TREATMENT_FIELD_NUMBER: _ClassVar[int]
    LINEAGE_FIELD_NUMBER: _ClassVar[int]
    PLACEMENT_FIELD_NUMBER: _ClassVar[int]
    role: str
    subject: str
    treatment: str
    lineage: str
    placement: str
    def __init__(self, role: _Optional[str] = ..., subject: _Optional[str] = ..., treatment: _Optional[str] = ..., lineage: _Optional[str] = ..., placement: _Optional[str] = ...) -> None: ...

class ListStylesResponse(_message.Message):
    __slots__ = ("styles",)
    STYLES_FIELD_NUMBER: _ClassVar[int]
    styles: _containers.RepeatedCompositeFieldContainer[_shared_pb2.Style]
    def __init__(self, styles: _Optional[_Iterable[_Union[_shared_pb2.Style, _Mapping]]] = ...) -> None: ...

class CreateStyleRequest(_message.Message):
    __slots__ = ("style",)
    STYLE_FIELD_NUMBER: _ClassVar[int]
    style: _shared_pb2.Style
    def __init__(self, style: _Optional[_Union[_shared_pb2.Style, _Mapping]] = ...) -> None: ...
