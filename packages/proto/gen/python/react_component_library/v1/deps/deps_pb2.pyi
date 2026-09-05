from google.protobuf.internal import containers as _containers
from google.protobuf.internal import enum_type_wrapper as _enum_type_wrapper
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Iterable as _Iterable, Mapping as _Mapping
from typing import ClassVar as _ClassVar, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class VerdictKind(int, metaclass=_enum_type_wrapper.EnumTypeWrapper):
    __slots__ = ()
    VERDICT_KIND_UNSPECIFIED: _ClassVar[VerdictKind]
    VERDICT_KIND_OK: _ClassVar[VerdictKind]
    VERDICT_KIND_WARN: _ClassVar[VerdictKind]
    VERDICT_KIND_BLOCK: _ClassVar[VerdictKind]

class IssueKind(int, metaclass=_enum_type_wrapper.EnumTypeWrapper):
    __slots__ = ()
    ISSUE_KIND_UNSPECIFIED: _ClassVar[IssueKind]
    ISSUE_KIND_MISSING_DEP: _ClassVar[IssueKind]
    ISSUE_KIND_RANGE_DOES_NOT_MATCH: _ClassVar[IssueKind]
    ISSUE_KIND_INCOMPATIBLE_MAJOR: _ClassVar[IssueKind]
    ISSUE_KIND_UNPARSEABLE_RANGE: _ClassVar[IssueKind]
    ISSUE_KIND_UNPARSEABLE_TARGET: _ClassVar[IssueKind]
VERDICT_KIND_UNSPECIFIED: VerdictKind
VERDICT_KIND_OK: VerdictKind
VERDICT_KIND_WARN: VerdictKind
VERDICT_KIND_BLOCK: VerdictKind
ISSUE_KIND_UNSPECIFIED: IssueKind
ISSUE_KIND_MISSING_DEP: IssueKind
ISSUE_KIND_RANGE_DOES_NOT_MATCH: IssueKind
ISSUE_KIND_INCOMPATIBLE_MAJOR: IssueKind
ISSUE_KIND_UNPARSEABLE_RANGE: IssueKind
ISSUE_KIND_UNPARSEABLE_TARGET: IssueKind

class DepDeclaration(_message.Message):
    __slots__ = ("component_id", "library_id", "dep_name", "version_range", "version", "kind")
    COMPONENT_ID_FIELD_NUMBER: _ClassVar[int]
    LIBRARY_ID_FIELD_NUMBER: _ClassVar[int]
    DEP_NAME_FIELD_NUMBER: _ClassVar[int]
    VERSION_RANGE_FIELD_NUMBER: _ClassVar[int]
    VERSION_FIELD_NUMBER: _ClassVar[int]
    KIND_FIELD_NUMBER: _ClassVar[int]
    component_id: str
    library_id: str
    dep_name: str
    version_range: str
    version: str
    kind: str
    def __init__(self, component_id: _Optional[str] = ..., library_id: _Optional[str] = ..., dep_name: _Optional[str] = ..., version_range: _Optional[str] = ..., version: _Optional[str] = ..., kind: _Optional[str] = ...) -> None: ...

class DepIssue(_message.Message):
    __slots__ = ("dep_name", "declared_range", "scenario_version", "kind", "detail", "version", "dep_kind")
    DEP_NAME_FIELD_NUMBER: _ClassVar[int]
    DECLARED_RANGE_FIELD_NUMBER: _ClassVar[int]
    SCENARIO_VERSION_FIELD_NUMBER: _ClassVar[int]
    KIND_FIELD_NUMBER: _ClassVar[int]
    DETAIL_FIELD_NUMBER: _ClassVar[int]
    VERSION_FIELD_NUMBER: _ClassVar[int]
    DEP_KIND_FIELD_NUMBER: _ClassVar[int]
    dep_name: str
    declared_range: str
    scenario_version: str
    kind: IssueKind
    detail: str
    version: str
    dep_kind: str
    def __init__(self, dep_name: _Optional[str] = ..., declared_range: _Optional[str] = ..., scenario_version: _Optional[str] = ..., kind: _Optional[_Union[IssueKind, str]] = ..., detail: _Optional[str] = ..., version: _Optional[str] = ..., dep_kind: _Optional[str] = ...) -> None: ...

class ListDeclarationsRequest(_message.Message):
    __slots__ = ("component_id",)
    COMPONENT_ID_FIELD_NUMBER: _ClassVar[int]
    component_id: str
    def __init__(self, component_id: _Optional[str] = ...) -> None: ...

class ListDeclarationsResponse(_message.Message):
    __slots__ = ("declarations",)
    DECLARATIONS_FIELD_NUMBER: _ClassVar[int]
    declarations: _containers.RepeatedCompositeFieldContainer[DepDeclaration]
    def __init__(self, declarations: _Optional[_Iterable[_Union[DepDeclaration, _Mapping]]] = ...) -> None: ...

class ValidateAdoptionRequest(_message.Message):
    __slots__ = ("component_id", "scenario", "version")
    COMPONENT_ID_FIELD_NUMBER: _ClassVar[int]
    SCENARIO_FIELD_NUMBER: _ClassVar[int]
    VERSION_FIELD_NUMBER: _ClassVar[int]
    component_id: str
    scenario: str
    version: str
    def __init__(self, component_id: _Optional[str] = ..., scenario: _Optional[str] = ..., version: _Optional[str] = ...) -> None: ...

class ValidateAdoptionResponse(_message.Message):
    __slots__ = ("kind", "issues")
    KIND_FIELD_NUMBER: _ClassVar[int]
    ISSUES_FIELD_NUMBER: _ClassVar[int]
    kind: VerdictKind
    issues: _containers.RepeatedCompositeFieldContainer[DepIssue]
    def __init__(self, kind: _Optional[_Union[VerdictKind, str]] = ..., issues: _Optional[_Iterable[_Union[DepIssue, _Mapping]]] = ...) -> None: ...
