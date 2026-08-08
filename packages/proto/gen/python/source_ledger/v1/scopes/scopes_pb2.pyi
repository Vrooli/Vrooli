from google.protobuf.internal import containers as _containers
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Iterable as _Iterable, Mapping as _Mapping
from typing import ClassVar as _ClassVar, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class FacetSpec(_message.Message):
    __slots__ = ("id", "label", "guidance", "retention_policy", "compaction_eligible", "resident_budget")
    ID_FIELD_NUMBER: _ClassVar[int]
    LABEL_FIELD_NUMBER: _ClassVar[int]
    GUIDANCE_FIELD_NUMBER: _ClassVar[int]
    RETENTION_POLICY_FIELD_NUMBER: _ClassVar[int]
    COMPACTION_ELIGIBLE_FIELD_NUMBER: _ClassVar[int]
    RESIDENT_BUDGET_FIELD_NUMBER: _ClassVar[int]
    id: str
    label: str
    guidance: str
    retention_policy: str
    compaction_eligible: bool
    resident_budget: int
    def __init__(self, id: _Optional[str] = ..., label: _Optional[str] = ..., guidance: _Optional[str] = ..., retention_policy: _Optional[str] = ..., compaction_eligible: _Optional[bool] = ..., resident_budget: _Optional[int] = ...) -> None: ...

class Scope(_message.Message):
    __slots__ = ("id", "label", "frontier_target", "wake_budget", "max_entry_lines", "facets")
    ID_FIELD_NUMBER: _ClassVar[int]
    LABEL_FIELD_NUMBER: _ClassVar[int]
    FRONTIER_TARGET_FIELD_NUMBER: _ClassVar[int]
    WAKE_BUDGET_FIELD_NUMBER: _ClassVar[int]
    MAX_ENTRY_LINES_FIELD_NUMBER: _ClassVar[int]
    FACETS_FIELD_NUMBER: _ClassVar[int]
    id: str
    label: str
    frontier_target: int
    wake_budget: int
    max_entry_lines: int
    facets: _containers.RepeatedCompositeFieldContainer[FacetSpec]
    def __init__(self, id: _Optional[str] = ..., label: _Optional[str] = ..., frontier_target: _Optional[int] = ..., wake_budget: _Optional[int] = ..., max_entry_lines: _Optional[int] = ..., facets: _Optional[_Iterable[_Union[FacetSpec, _Mapping]]] = ...) -> None: ...

class CreateScopeRequest(_message.Message):
    __slots__ = ("scope",)
    SCOPE_FIELD_NUMBER: _ClassVar[int]
    scope: Scope
    def __init__(self, scope: _Optional[_Union[Scope, _Mapping]] = ...) -> None: ...

class CreateScopeResponse(_message.Message):
    __slots__ = ("scope",)
    SCOPE_FIELD_NUMBER: _ClassVar[int]
    scope: Scope
    def __init__(self, scope: _Optional[_Union[Scope, _Mapping]] = ...) -> None: ...

class ListScopesRequest(_message.Message):
    __slots__ = ("scope",)
    SCOPE_FIELD_NUMBER: _ClassVar[int]
    scope: str
    def __init__(self, scope: _Optional[str] = ...) -> None: ...

class ListScopesResponse(_message.Message):
    __slots__ = ("scopes",)
    SCOPES_FIELD_NUMBER: _ClassVar[int]
    scopes: _containers.RepeatedCompositeFieldContainer[Scope]
    def __init__(self, scopes: _Optional[_Iterable[_Union[Scope, _Mapping]]] = ...) -> None: ...
