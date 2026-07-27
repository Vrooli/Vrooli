from google.protobuf import struct_pb2 as _struct_pb2
from landing_page_business_suite.v1.shared import content_pb2 as _content_pb2
from google.protobuf.internal import containers as _containers
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Iterable as _Iterable, Mapping as _Mapping
from typing import ClassVar as _ClassVar, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class GetPublicSectionsRequest(_message.Message):
    __slots__ = ("variant_id",)
    VARIANT_ID_FIELD_NUMBER: _ClassVar[int]
    variant_id: int
    def __init__(self, variant_id: _Optional[int] = ...) -> None: ...

class GetSectionsRequest(_message.Message):
    __slots__ = ("variant_id",)
    VARIANT_ID_FIELD_NUMBER: _ClassVar[int]
    variant_id: int
    def __init__(self, variant_id: _Optional[int] = ...) -> None: ...

class SectionsResponse(_message.Message):
    __slots__ = ("sections",)
    SECTIONS_FIELD_NUMBER: _ClassVar[int]
    sections: _containers.RepeatedCompositeFieldContainer[_content_pb2.ContentSection]
    def __init__(self, sections: _Optional[_Iterable[_Union[_content_pb2.ContentSection, _Mapping]]] = ...) -> None: ...

class GetSectionRequest(_message.Message):
    __slots__ = ("id",)
    ID_FIELD_NUMBER: _ClassVar[int]
    id: int
    def __init__(self, id: _Optional[int] = ...) -> None: ...

class SectionResponse(_message.Message):
    __slots__ = ("section",)
    SECTION_FIELD_NUMBER: _ClassVar[int]
    section: _content_pb2.ContentSection
    def __init__(self, section: _Optional[_Union[_content_pb2.ContentSection, _Mapping]] = ...) -> None: ...

class CreateSectionRequest(_message.Message):
    __slots__ = ("variant_id", "section_type", "content", "order", "enabled")
    VARIANT_ID_FIELD_NUMBER: _ClassVar[int]
    SECTION_TYPE_FIELD_NUMBER: _ClassVar[int]
    CONTENT_FIELD_NUMBER: _ClassVar[int]
    ORDER_FIELD_NUMBER: _ClassVar[int]
    ENABLED_FIELD_NUMBER: _ClassVar[int]
    variant_id: int
    section_type: str
    content: _struct_pb2.Struct
    order: int
    enabled: bool
    def __init__(self, variant_id: _Optional[int] = ..., section_type: _Optional[str] = ..., content: _Optional[_Union[_struct_pb2.Struct, _Mapping]] = ..., order: _Optional[int] = ..., enabled: _Optional[bool] = ...) -> None: ...

class UpdateSectionRequest(_message.Message):
    __slots__ = ("id", "content")
    ID_FIELD_NUMBER: _ClassVar[int]
    CONTENT_FIELD_NUMBER: _ClassVar[int]
    id: int
    content: _struct_pb2.Struct
    def __init__(self, id: _Optional[int] = ..., content: _Optional[_Union[_struct_pb2.Struct, _Mapping]] = ...) -> None: ...

class DeleteSectionRequest(_message.Message):
    __slots__ = ("id",)
    ID_FIELD_NUMBER: _ClassVar[int]
    id: int
    def __init__(self, id: _Optional[int] = ...) -> None: ...

class DeleteSectionResponse(_message.Message):
    __slots__ = ("deleted",)
    DELETED_FIELD_NUMBER: _ClassVar[int]
    deleted: bool
    def __init__(self, deleted: _Optional[bool] = ...) -> None: ...
