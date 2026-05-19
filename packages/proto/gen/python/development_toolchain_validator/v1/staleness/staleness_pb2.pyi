from google.protobuf.internal import containers as _containers
from google.protobuf.internal import enum_type_wrapper as _enum_type_wrapper
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Iterable as _Iterable, Mapping as _Mapping
from typing import ClassVar as _ClassVar, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class StaleKind(int, metaclass=_enum_type_wrapper.EnumTypeWrapper):
    __slots__ = ()
    STALE_KIND_UNSPECIFIED: _ClassVar[StaleKind]
    STALE_KIND_TEMPLATE_DRIFT: _ClassVar[StaleKind]
    STALE_KIND_SKILL_DRIFT: _ClassVar[StaleKind]
    STALE_KIND_BOTH: _ClassVar[StaleKind]
STALE_KIND_UNSPECIFIED: StaleKind
STALE_KIND_TEMPLATE_DRIFT: StaleKind
STALE_KIND_SKILL_DRIFT: StaleKind
STALE_KIND_BOTH: StaleKind

class StaleEntry(_message.Message):
    __slots__ = ("skill_id", "golden_slug", "kind", "manifest_template_version_pinned", "manifest_skill_version_pinned", "golden_template_version_current", "skill_version_current")
    SKILL_ID_FIELD_NUMBER: _ClassVar[int]
    GOLDEN_SLUG_FIELD_NUMBER: _ClassVar[int]
    KIND_FIELD_NUMBER: _ClassVar[int]
    MANIFEST_TEMPLATE_VERSION_PINNED_FIELD_NUMBER: _ClassVar[int]
    MANIFEST_SKILL_VERSION_PINNED_FIELD_NUMBER: _ClassVar[int]
    GOLDEN_TEMPLATE_VERSION_CURRENT_FIELD_NUMBER: _ClassVar[int]
    SKILL_VERSION_CURRENT_FIELD_NUMBER: _ClassVar[int]
    skill_id: str
    golden_slug: str
    kind: StaleKind
    manifest_template_version_pinned: str
    manifest_skill_version_pinned: str
    golden_template_version_current: str
    skill_version_current: str
    def __init__(self, skill_id: _Optional[str] = ..., golden_slug: _Optional[str] = ..., kind: _Optional[_Union[StaleKind, str]] = ..., manifest_template_version_pinned: _Optional[str] = ..., manifest_skill_version_pinned: _Optional[str] = ..., golden_template_version_current: _Optional[str] = ..., skill_version_current: _Optional[str] = ...) -> None: ...

class ListStaleRequest(_message.Message):
    __slots__ = ()
    def __init__(self) -> None: ...

class ListStaleResponse(_message.Message):
    __slots__ = ("entries",)
    ENTRIES_FIELD_NUMBER: _ClassVar[int]
    entries: _containers.RepeatedCompositeFieldContainer[StaleEntry]
    def __init__(self, entries: _Optional[_Iterable[_Union[StaleEntry, _Mapping]]] = ...) -> None: ...
