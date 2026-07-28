import datetime

from google.protobuf import timestamp_pb2 as _timestamp_pb2
from landing_page_business_suite.v1.shared import content_pb2 as _content_pb2
from landing_page_business_suite.v1.shared import presentation_pb2 as _presentation_pb2
from google.protobuf.internal import containers as _containers
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Iterable as _Iterable, Mapping as _Mapping
from typing import ClassVar as _ClassVar, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class Variant(_message.Message):
    __slots__ = ("id", "slug", "name", "description", "weight", "status", "created_at", "updated_at", "archived_at", "axes", "header_config", "seo_config")
    class AxesEntry(_message.Message):
        __slots__ = ("key", "value")
        KEY_FIELD_NUMBER: _ClassVar[int]
        VALUE_FIELD_NUMBER: _ClassVar[int]
        key: str
        value: str
        def __init__(self, key: _Optional[str] = ..., value: _Optional[str] = ...) -> None: ...
    ID_FIELD_NUMBER: _ClassVar[int]
    SLUG_FIELD_NUMBER: _ClassVar[int]
    NAME_FIELD_NUMBER: _ClassVar[int]
    DESCRIPTION_FIELD_NUMBER: _ClassVar[int]
    WEIGHT_FIELD_NUMBER: _ClassVar[int]
    STATUS_FIELD_NUMBER: _ClassVar[int]
    CREATED_AT_FIELD_NUMBER: _ClassVar[int]
    UPDATED_AT_FIELD_NUMBER: _ClassVar[int]
    ARCHIVED_AT_FIELD_NUMBER: _ClassVar[int]
    AXES_FIELD_NUMBER: _ClassVar[int]
    HEADER_CONFIG_FIELD_NUMBER: _ClassVar[int]
    SEO_CONFIG_FIELD_NUMBER: _ClassVar[int]
    id: int
    slug: str
    name: str
    description: str
    weight: int
    status: str
    created_at: _timestamp_pb2.Timestamp
    updated_at: _timestamp_pb2.Timestamp
    archived_at: _timestamp_pb2.Timestamp
    axes: _containers.ScalarMap[str, str]
    header_config: _presentation_pb2.LandingHeaderConfig
    seo_config: _presentation_pb2.VariantSEOConfig
    def __init__(self, id: _Optional[int] = ..., slug: _Optional[str] = ..., name: _Optional[str] = ..., description: _Optional[str] = ..., weight: _Optional[int] = ..., status: _Optional[str] = ..., created_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., updated_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., archived_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., axes: _Optional[_Mapping[str, str]] = ..., header_config: _Optional[_Union[_presentation_pb2.LandingHeaderConfig, _Mapping]] = ..., seo_config: _Optional[_Union[_presentation_pb2.VariantSEOConfig, _Mapping]] = ...) -> None: ...

class AxesSelection(_message.Message):
    __slots__ = ("values",)
    class ValuesEntry(_message.Message):
        __slots__ = ("key", "value")
        KEY_FIELD_NUMBER: _ClassVar[int]
        VALUE_FIELD_NUMBER: _ClassVar[int]
        key: str
        value: str
        def __init__(self, key: _Optional[str] = ..., value: _Optional[str] = ...) -> None: ...
    VALUES_FIELD_NUMBER: _ClassVar[int]
    values: _containers.ScalarMap[str, str]
    def __init__(self, values: _Optional[_Mapping[str, str]] = ...) -> None: ...

class VariantSnapshot(_message.Message):
    __slots__ = ("slug", "name", "description", "weight", "status", "axes", "header_config", "seo_config", "sections")
    class AxesEntry(_message.Message):
        __slots__ = ("key", "value")
        KEY_FIELD_NUMBER: _ClassVar[int]
        VALUE_FIELD_NUMBER: _ClassVar[int]
        key: str
        value: str
        def __init__(self, key: _Optional[str] = ..., value: _Optional[str] = ...) -> None: ...
    SLUG_FIELD_NUMBER: _ClassVar[int]
    NAME_FIELD_NUMBER: _ClassVar[int]
    DESCRIPTION_FIELD_NUMBER: _ClassVar[int]
    WEIGHT_FIELD_NUMBER: _ClassVar[int]
    STATUS_FIELD_NUMBER: _ClassVar[int]
    AXES_FIELD_NUMBER: _ClassVar[int]
    HEADER_CONFIG_FIELD_NUMBER: _ClassVar[int]
    SEO_CONFIG_FIELD_NUMBER: _ClassVar[int]
    SECTIONS_FIELD_NUMBER: _ClassVar[int]
    slug: str
    name: str
    description: str
    weight: int
    status: str
    axes: _containers.ScalarMap[str, str]
    header_config: _presentation_pb2.LandingHeaderConfig
    seo_config: _presentation_pb2.VariantSEOConfig
    sections: _containers.RepeatedCompositeFieldContainer[_content_pb2.ContentSection]
    def __init__(self, slug: _Optional[str] = ..., name: _Optional[str] = ..., description: _Optional[str] = ..., weight: _Optional[int] = ..., status: _Optional[str] = ..., axes: _Optional[_Mapping[str, str]] = ..., header_config: _Optional[_Union[_presentation_pb2.LandingHeaderConfig, _Mapping]] = ..., seo_config: _Optional[_Union[_presentation_pb2.VariantSEOConfig, _Mapping]] = ..., sections: _Optional[_Iterable[_Union[_content_pb2.ContentSection, _Mapping]]] = ...) -> None: ...

class SelectVariantRequest(_message.Message):
    __slots__ = ()
    def __init__(self) -> None: ...

class GetPublicVariantRequest(_message.Message):
    __slots__ = ("slug",)
    SLUG_FIELD_NUMBER: _ClassVar[int]
    slug: str
    def __init__(self, slug: _Optional[str] = ...) -> None: ...

class GetVariantRequest(_message.Message):
    __slots__ = ("slug",)
    SLUG_FIELD_NUMBER: _ClassVar[int]
    slug: str
    def __init__(self, slug: _Optional[str] = ...) -> None: ...

class VariantResponse(_message.Message):
    __slots__ = ("variant",)
    VARIANT_FIELD_NUMBER: _ClassVar[int]
    variant: Variant
    def __init__(self, variant: _Optional[_Union[Variant, _Mapping]] = ...) -> None: ...

class ListVariantsRequest(_message.Message):
    __slots__ = ("status_filter",)
    STATUS_FILTER_FIELD_NUMBER: _ClassVar[int]
    status_filter: str
    def __init__(self, status_filter: _Optional[str] = ...) -> None: ...

class ListVariantsResponse(_message.Message):
    __slots__ = ("variants",)
    VARIANTS_FIELD_NUMBER: _ClassVar[int]
    variants: _containers.RepeatedCompositeFieldContainer[Variant]
    def __init__(self, variants: _Optional[_Iterable[_Union[Variant, _Mapping]]] = ...) -> None: ...

class CreateVariantRequest(_message.Message):
    __slots__ = ("slug", "name", "description", "weight", "axes")
    class AxesEntry(_message.Message):
        __slots__ = ("key", "value")
        KEY_FIELD_NUMBER: _ClassVar[int]
        VALUE_FIELD_NUMBER: _ClassVar[int]
        key: str
        value: str
        def __init__(self, key: _Optional[str] = ..., value: _Optional[str] = ...) -> None: ...
    SLUG_FIELD_NUMBER: _ClassVar[int]
    NAME_FIELD_NUMBER: _ClassVar[int]
    DESCRIPTION_FIELD_NUMBER: _ClassVar[int]
    WEIGHT_FIELD_NUMBER: _ClassVar[int]
    AXES_FIELD_NUMBER: _ClassVar[int]
    slug: str
    name: str
    description: str
    weight: int
    axes: _containers.ScalarMap[str, str]
    def __init__(self, slug: _Optional[str] = ..., name: _Optional[str] = ..., description: _Optional[str] = ..., weight: _Optional[int] = ..., axes: _Optional[_Mapping[str, str]] = ...) -> None: ...

class UpdateVariantRequest(_message.Message):
    __slots__ = ("slug", "name", "description", "weight", "axes", "header_config")
    SLUG_FIELD_NUMBER: _ClassVar[int]
    NAME_FIELD_NUMBER: _ClassVar[int]
    DESCRIPTION_FIELD_NUMBER: _ClassVar[int]
    WEIGHT_FIELD_NUMBER: _ClassVar[int]
    AXES_FIELD_NUMBER: _ClassVar[int]
    HEADER_CONFIG_FIELD_NUMBER: _ClassVar[int]
    slug: str
    name: str
    description: str
    weight: int
    axes: AxesSelection
    header_config: _presentation_pb2.LandingHeaderConfig
    def __init__(self, slug: _Optional[str] = ..., name: _Optional[str] = ..., description: _Optional[str] = ..., weight: _Optional[int] = ..., axes: _Optional[_Union[AxesSelection, _Mapping]] = ..., header_config: _Optional[_Union[_presentation_pb2.LandingHeaderConfig, _Mapping]] = ...) -> None: ...

class ArchiveVariantRequest(_message.Message):
    __slots__ = ("slug",)
    SLUG_FIELD_NUMBER: _ClassVar[int]
    slug: str
    def __init__(self, slug: _Optional[str] = ...) -> None: ...

class DeleteVariantRequest(_message.Message):
    __slots__ = ("slug",)
    SLUG_FIELD_NUMBER: _ClassVar[int]
    slug: str
    def __init__(self, slug: _Optional[str] = ...) -> None: ...

class DeleteVariantResponse(_message.Message):
    __slots__ = ("deleted",)
    DELETED_FIELD_NUMBER: _ClassVar[int]
    deleted: bool
    def __init__(self, deleted: _Optional[bool] = ...) -> None: ...

class ExportVariantSnapshotRequest(_message.Message):
    __slots__ = ("slug",)
    SLUG_FIELD_NUMBER: _ClassVar[int]
    slug: str
    def __init__(self, slug: _Optional[str] = ...) -> None: ...

class ExportVariantSnapshotResponse(_message.Message):
    __slots__ = ("snapshot",)
    SNAPSHOT_FIELD_NUMBER: _ClassVar[int]
    snapshot: VariantSnapshot
    def __init__(self, snapshot: _Optional[_Union[VariantSnapshot, _Mapping]] = ...) -> None: ...

class ImportVariantSnapshotRequest(_message.Message):
    __slots__ = ("slug", "snapshot")
    SLUG_FIELD_NUMBER: _ClassVar[int]
    SNAPSHOT_FIELD_NUMBER: _ClassVar[int]
    slug: str
    snapshot: VariantSnapshot
    def __init__(self, slug: _Optional[str] = ..., snapshot: _Optional[_Union[VariantSnapshot, _Mapping]] = ...) -> None: ...

class ImportVariantSnapshotResponse(_message.Message):
    __slots__ = ("snapshot",)
    SNAPSHOT_FIELD_NUMBER: _ClassVar[int]
    snapshot: VariantSnapshot
    def __init__(self, snapshot: _Optional[_Union[VariantSnapshot, _Mapping]] = ...) -> None: ...
