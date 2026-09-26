import datetime

from google.protobuf import timestamp_pb2 as _timestamp_pb2
from google.protobuf.internal import containers as _containers
from google.protobuf.internal import enum_type_wrapper as _enum_type_wrapper
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Iterable as _Iterable, Mapping as _Mapping
from typing import ClassVar as _ClassVar, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class TemplateKind(int, metaclass=_enum_type_wrapper.EnumTypeWrapper):
    __slots__ = ()
    TEMPLATE_KIND_UNSPECIFIED: _ClassVar[TemplateKind]
    TEMPLATE_KIND_SCENARIO: _ClassVar[TemplateKind]
    TEMPLATE_KIND_DESIGN: _ClassVar[TemplateKind]
    TEMPLATE_KIND_RESOURCE: _ClassVar[TemplateKind]
TEMPLATE_KIND_UNSPECIFIED: TemplateKind
TEMPLATE_KIND_SCENARIO: TemplateKind
TEMPLATE_KIND_DESIGN: TemplateKind
TEMPLATE_KIND_RESOURCE: TemplateKind

class TemplateRecord(_message.Message):
    __slots__ = ("id", "kind", "display_name", "version", "manifest_path", "source_path", "tags", "status", "updated_at", "version_lag")
    ID_FIELD_NUMBER: _ClassVar[int]
    KIND_FIELD_NUMBER: _ClassVar[int]
    DISPLAY_NAME_FIELD_NUMBER: _ClassVar[int]
    VERSION_FIELD_NUMBER: _ClassVar[int]
    MANIFEST_PATH_FIELD_NUMBER: _ClassVar[int]
    SOURCE_PATH_FIELD_NUMBER: _ClassVar[int]
    TAGS_FIELD_NUMBER: _ClassVar[int]
    STATUS_FIELD_NUMBER: _ClassVar[int]
    UPDATED_AT_FIELD_NUMBER: _ClassVar[int]
    VERSION_LAG_FIELD_NUMBER: _ClassVar[int]
    id: str
    kind: TemplateKind
    display_name: str
    version: str
    manifest_path: str
    source_path: str
    tags: _containers.RepeatedScalarFieldContainer[str]
    status: str
    updated_at: _timestamp_pb2.Timestamp
    version_lag: TemplateVersionLag
    def __init__(self, id: _Optional[str] = ..., kind: _Optional[_Union[TemplateKind, str]] = ..., display_name: _Optional[str] = ..., version: _Optional[str] = ..., manifest_path: _Optional[str] = ..., source_path: _Optional[str] = ..., tags: _Optional[_Iterable[str]] = ..., status: _Optional[str] = ..., updated_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., version_lag: _Optional[_Union[TemplateVersionLag, _Mapping]] = ...) -> None: ...

class TemplateVersionLag(_message.Message):
    __slots__ = ("current_version", "latest_version", "lag_count")
    CURRENT_VERSION_FIELD_NUMBER: _ClassVar[int]
    LATEST_VERSION_FIELD_NUMBER: _ClassVar[int]
    LAG_COUNT_FIELD_NUMBER: _ClassVar[int]
    current_version: str
    latest_version: str
    lag_count: int
    def __init__(self, current_version: _Optional[str] = ..., latest_version: _Optional[str] = ..., lag_count: _Optional[int] = ...) -> None: ...

class ListTemplatesRequest(_message.Message):
    __slots__ = ("kind",)
    KIND_FIELD_NUMBER: _ClassVar[int]
    kind: TemplateKind
    def __init__(self, kind: _Optional[_Union[TemplateKind, str]] = ...) -> None: ...

class ListTemplatesResponse(_message.Message):
    __slots__ = ("templates",)
    TEMPLATES_FIELD_NUMBER: _ClassVar[int]
    templates: _containers.RepeatedCompositeFieldContainer[TemplateRecord]
    def __init__(self, templates: _Optional[_Iterable[_Union[TemplateRecord, _Mapping]]] = ...) -> None: ...

class GetTemplateRequest(_message.Message):
    __slots__ = ("id",)
    ID_FIELD_NUMBER: _ClassVar[int]
    id: str
    def __init__(self, id: _Optional[str] = ...) -> None: ...

class GetTemplateResponse(_message.Message):
    __slots__ = ("template",)
    TEMPLATE_FIELD_NUMBER: _ClassVar[int]
    template: TemplateRecord
    def __init__(self, template: _Optional[_Union[TemplateRecord, _Mapping]] = ...) -> None: ...
