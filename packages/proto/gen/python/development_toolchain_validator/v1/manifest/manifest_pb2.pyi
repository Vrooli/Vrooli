import datetime

from google.protobuf import timestamp_pb2 as _timestamp_pb2
from google.protobuf.internal import containers as _containers
from google.protobuf.internal import enum_type_wrapper as _enum_type_wrapper
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Iterable as _Iterable, Mapping as _Mapping
from typing import ClassVar as _ClassVar, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class ConvergenceTarget(int, metaclass=_enum_type_wrapper.EnumTypeWrapper):
    __slots__ = ()
    CONVERGENCE_TARGET_UNSPECIFIED: _ClassVar[ConvergenceTarget]
    CONVERGENCE_TARGET_NONE: _ClassVar[ConvergenceTarget]
    CONVERGENCE_TARGET_EMPTY_DIFF: _ClassVar[ConvergenceTarget]
CONVERGENCE_TARGET_UNSPECIFIED: ConvergenceTarget
CONVERGENCE_TARGET_NONE: ConvergenceTarget
CONVERGENCE_TARGET_EMPTY_DIFF: ConvergenceTarget

class ContentRule(_message.Message):
    __slots__ = ("path_glob", "must_contain", "must_not_contain")
    PATH_GLOB_FIELD_NUMBER: _ClassVar[int]
    MUST_CONTAIN_FIELD_NUMBER: _ClassVar[int]
    MUST_NOT_CONTAIN_FIELD_NUMBER: _ClassVar[int]
    path_glob: str
    must_contain: _containers.RepeatedScalarFieldContainer[str]
    must_not_contain: _containers.RepeatedScalarFieldContainer[str]
    def __init__(self, path_glob: _Optional[str] = ..., must_contain: _Optional[_Iterable[str]] = ..., must_not_contain: _Optional[_Iterable[str]] = ...) -> None: ...

class Manifest(_message.Message):
    __slots__ = ("skill_id", "golden_slug", "allowed_paths", "content_rules", "wildcard_allowed", "convergence_target", "template_version_pinned", "skill_version_pinned", "updated_at")
    SKILL_ID_FIELD_NUMBER: _ClassVar[int]
    GOLDEN_SLUG_FIELD_NUMBER: _ClassVar[int]
    ALLOWED_PATHS_FIELD_NUMBER: _ClassVar[int]
    CONTENT_RULES_FIELD_NUMBER: _ClassVar[int]
    WILDCARD_ALLOWED_FIELD_NUMBER: _ClassVar[int]
    CONVERGENCE_TARGET_FIELD_NUMBER: _ClassVar[int]
    TEMPLATE_VERSION_PINNED_FIELD_NUMBER: _ClassVar[int]
    SKILL_VERSION_PINNED_FIELD_NUMBER: _ClassVar[int]
    UPDATED_AT_FIELD_NUMBER: _ClassVar[int]
    skill_id: str
    golden_slug: str
    allowed_paths: _containers.RepeatedScalarFieldContainer[str]
    content_rules: _containers.RepeatedCompositeFieldContainer[ContentRule]
    wildcard_allowed: bool
    convergence_target: ConvergenceTarget
    template_version_pinned: str
    skill_version_pinned: str
    updated_at: _timestamp_pb2.Timestamp
    def __init__(self, skill_id: _Optional[str] = ..., golden_slug: _Optional[str] = ..., allowed_paths: _Optional[_Iterable[str]] = ..., content_rules: _Optional[_Iterable[_Union[ContentRule, _Mapping]]] = ..., wildcard_allowed: _Optional[bool] = ..., convergence_target: _Optional[_Union[ConvergenceTarget, str]] = ..., template_version_pinned: _Optional[str] = ..., skill_version_pinned: _Optional[str] = ..., updated_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ...) -> None: ...

class ListManifestsRequest(_message.Message):
    __slots__ = ()
    def __init__(self) -> None: ...

class ListManifestsResponse(_message.Message):
    __slots__ = ("manifests",)
    MANIFESTS_FIELD_NUMBER: _ClassVar[int]
    manifests: _containers.RepeatedCompositeFieldContainer[Manifest]
    def __init__(self, manifests: _Optional[_Iterable[_Union[Manifest, _Mapping]]] = ...) -> None: ...

class GetManifestRequest(_message.Message):
    __slots__ = ("skill_id", "golden_slug")
    SKILL_ID_FIELD_NUMBER: _ClassVar[int]
    GOLDEN_SLUG_FIELD_NUMBER: _ClassVar[int]
    skill_id: str
    golden_slug: str
    def __init__(self, skill_id: _Optional[str] = ..., golden_slug: _Optional[str] = ...) -> None: ...

class GetManifestResponse(_message.Message):
    __slots__ = ("manifest",)
    MANIFEST_FIELD_NUMBER: _ClassVar[int]
    manifest: Manifest
    def __init__(self, manifest: _Optional[_Union[Manifest, _Mapping]] = ...) -> None: ...

class UpsertManifestRequest(_message.Message):
    __slots__ = ("manifest",)
    MANIFEST_FIELD_NUMBER: _ClassVar[int]
    manifest: Manifest
    def __init__(self, manifest: _Optional[_Union[Manifest, _Mapping]] = ...) -> None: ...

class UpsertManifestResponse(_message.Message):
    __slots__ = ("manifest",)
    MANIFEST_FIELD_NUMBER: _ClassVar[int]
    manifest: Manifest
    def __init__(self, manifest: _Optional[_Union[Manifest, _Mapping]] = ...) -> None: ...

class ClearStaleRequest(_message.Message):
    __slots__ = ("skill_id", "golden_slug")
    SKILL_ID_FIELD_NUMBER: _ClassVar[int]
    GOLDEN_SLUG_FIELD_NUMBER: _ClassVar[int]
    skill_id: str
    golden_slug: str
    def __init__(self, skill_id: _Optional[str] = ..., golden_slug: _Optional[str] = ...) -> None: ...

class ClearStaleResponse(_message.Message):
    __slots__ = ("cleared_at",)
    CLEARED_AT_FIELD_NUMBER: _ClassVar[int]
    cleared_at: _timestamp_pb2.Timestamp
    def __init__(self, cleared_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ...) -> None: ...
