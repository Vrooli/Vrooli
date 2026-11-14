import datetime

from google.protobuf import timestamp_pb2 as _timestamp_pb2
from google.protobuf.internal import containers as _containers
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Iterable as _Iterable, Mapping as _Mapping
from typing import ClassVar as _ClassVar, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class Golden(_message.Message):
    __slots__ = ("id", "slug", "template_id", "template_version_pinned", "path", "created_at", "last_regenerated_at", "generation_options_json", "materialization_mode", "logical_root", "last_materialized_path", "last_materialized_status")
    ID_FIELD_NUMBER: _ClassVar[int]
    SLUG_FIELD_NUMBER: _ClassVar[int]
    TEMPLATE_ID_FIELD_NUMBER: _ClassVar[int]
    TEMPLATE_VERSION_PINNED_FIELD_NUMBER: _ClassVar[int]
    PATH_FIELD_NUMBER: _ClassVar[int]
    CREATED_AT_FIELD_NUMBER: _ClassVar[int]
    LAST_REGENERATED_AT_FIELD_NUMBER: _ClassVar[int]
    GENERATION_OPTIONS_JSON_FIELD_NUMBER: _ClassVar[int]
    MATERIALIZATION_MODE_FIELD_NUMBER: _ClassVar[int]
    LOGICAL_ROOT_FIELD_NUMBER: _ClassVar[int]
    LAST_MATERIALIZED_PATH_FIELD_NUMBER: _ClassVar[int]
    LAST_MATERIALIZED_STATUS_FIELD_NUMBER: _ClassVar[int]
    id: str
    slug: str
    template_id: str
    template_version_pinned: str
    path: str
    created_at: _timestamp_pb2.Timestamp
    last_regenerated_at: _timestamp_pb2.Timestamp
    generation_options_json: str
    materialization_mode: str
    logical_root: str
    last_materialized_path: str
    last_materialized_status: str
    def __init__(self, id: _Optional[str] = ..., slug: _Optional[str] = ..., template_id: _Optional[str] = ..., template_version_pinned: _Optional[str] = ..., path: _Optional[str] = ..., created_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., last_regenerated_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., generation_options_json: _Optional[str] = ..., materialization_mode: _Optional[str] = ..., logical_root: _Optional[str] = ..., last_materialized_path: _Optional[str] = ..., last_materialized_status: _Optional[str] = ...) -> None: ...

class ListGoldensRequest(_message.Message):
    __slots__ = ()
    def __init__(self) -> None: ...

class ListGoldensResponse(_message.Message):
    __slots__ = ("goldens",)
    GOLDENS_FIELD_NUMBER: _ClassVar[int]
    goldens: _containers.RepeatedCompositeFieldContainer[Golden]
    def __init__(self, goldens: _Optional[_Iterable[_Union[Golden, _Mapping]]] = ...) -> None: ...

class GetGoldenRequest(_message.Message):
    __slots__ = ("slug",)
    SLUG_FIELD_NUMBER: _ClassVar[int]
    slug: str
    def __init__(self, slug: _Optional[str] = ...) -> None: ...

class GetGoldenResponse(_message.Message):
    __slots__ = ("golden",)
    GOLDEN_FIELD_NUMBER: _ClassVar[int]
    golden: Golden
    def __init__(self, golden: _Optional[_Union[Golden, _Mapping]] = ...) -> None: ...

class RegisterGoldenRequest(_message.Message):
    __slots__ = ("slug", "template_id", "template_version", "path", "generation_options_json", "materialization_mode", "logical_root")
    SLUG_FIELD_NUMBER: _ClassVar[int]
    TEMPLATE_ID_FIELD_NUMBER: _ClassVar[int]
    TEMPLATE_VERSION_FIELD_NUMBER: _ClassVar[int]
    PATH_FIELD_NUMBER: _ClassVar[int]
    GENERATION_OPTIONS_JSON_FIELD_NUMBER: _ClassVar[int]
    MATERIALIZATION_MODE_FIELD_NUMBER: _ClassVar[int]
    LOGICAL_ROOT_FIELD_NUMBER: _ClassVar[int]
    slug: str
    template_id: str
    template_version: str
    path: str
    generation_options_json: str
    materialization_mode: str
    logical_root: str
    def __init__(self, slug: _Optional[str] = ..., template_id: _Optional[str] = ..., template_version: _Optional[str] = ..., path: _Optional[str] = ..., generation_options_json: _Optional[str] = ..., materialization_mode: _Optional[str] = ..., logical_root: _Optional[str] = ...) -> None: ...

class RegisterGoldenResponse(_message.Message):
    __slots__ = ("golden",)
    GOLDEN_FIELD_NUMBER: _ClassVar[int]
    golden: Golden
    def __init__(self, golden: _Optional[_Union[Golden, _Mapping]] = ...) -> None: ...

class UpdateGoldenRequest(_message.Message):
    __slots__ = ("slug", "path", "template_version", "generation_options_json", "materialization_mode", "logical_root")
    SLUG_FIELD_NUMBER: _ClassVar[int]
    PATH_FIELD_NUMBER: _ClassVar[int]
    TEMPLATE_VERSION_FIELD_NUMBER: _ClassVar[int]
    GENERATION_OPTIONS_JSON_FIELD_NUMBER: _ClassVar[int]
    MATERIALIZATION_MODE_FIELD_NUMBER: _ClassVar[int]
    LOGICAL_ROOT_FIELD_NUMBER: _ClassVar[int]
    slug: str
    path: str
    template_version: str
    generation_options_json: str
    materialization_mode: str
    logical_root: str
    def __init__(self, slug: _Optional[str] = ..., path: _Optional[str] = ..., template_version: _Optional[str] = ..., generation_options_json: _Optional[str] = ..., materialization_mode: _Optional[str] = ..., logical_root: _Optional[str] = ...) -> None: ...

class UpdateGoldenResponse(_message.Message):
    __slots__ = ("golden",)
    GOLDEN_FIELD_NUMBER: _ClassVar[int]
    golden: Golden
    def __init__(self, golden: _Optional[_Union[Golden, _Mapping]] = ...) -> None: ...

class DeleteGoldenRequest(_message.Message):
    __slots__ = ("slug",)
    SLUG_FIELD_NUMBER: _ClassVar[int]
    slug: str
    def __init__(self, slug: _Optional[str] = ...) -> None: ...

class DeleteGoldenResponse(_message.Message):
    __slots__ = ()
    def __init__(self) -> None: ...

class RegenerateGoldenRequest(_message.Message):
    __slots__ = ("slug",)
    SLUG_FIELD_NUMBER: _ClassVar[int]
    slug: str
    def __init__(self, slug: _Optional[str] = ...) -> None: ...

class RegenerateGoldenResponse(_message.Message):
    __slots__ = ("golden",)
    GOLDEN_FIELD_NUMBER: _ClassVar[int]
    golden: Golden
    def __init__(self, golden: _Optional[_Union[Golden, _Mapping]] = ...) -> None: ...
