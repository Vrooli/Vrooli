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
    __slots__ = ("id", "label", "frontier_target", "wake_budget", "max_entry_lines", "facets", "wake_budget_chars", "max_entry_chars")
    ID_FIELD_NUMBER: _ClassVar[int]
    LABEL_FIELD_NUMBER: _ClassVar[int]
    FRONTIER_TARGET_FIELD_NUMBER: _ClassVar[int]
    WAKE_BUDGET_FIELD_NUMBER: _ClassVar[int]
    MAX_ENTRY_LINES_FIELD_NUMBER: _ClassVar[int]
    FACETS_FIELD_NUMBER: _ClassVar[int]
    WAKE_BUDGET_CHARS_FIELD_NUMBER: _ClassVar[int]
    MAX_ENTRY_CHARS_FIELD_NUMBER: _ClassVar[int]
    id: str
    label: str
    frontier_target: int
    wake_budget: int
    max_entry_lines: int
    facets: _containers.RepeatedCompositeFieldContainer[FacetSpec]
    wake_budget_chars: int
    max_entry_chars: int
    def __init__(self, id: _Optional[str] = ..., label: _Optional[str] = ..., frontier_target: _Optional[int] = ..., wake_budget: _Optional[int] = ..., max_entry_lines: _Optional[int] = ..., facets: _Optional[_Iterable[_Union[FacetSpec, _Mapping]]] = ..., wake_budget_chars: _Optional[int] = ..., max_entry_chars: _Optional[int] = ...) -> None: ...

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

class PolicySnapshot(_message.Message):
    __slots__ = ("frontier_target", "wake_budget_lines", "wake_budget_chars", "max_entry_lines", "max_entry_chars", "frontier_target_origin", "wake_budget_lines_origin", "wake_budget_chars_origin", "max_entry_lines_origin", "max_entry_chars_origin")
    FRONTIER_TARGET_FIELD_NUMBER: _ClassVar[int]
    WAKE_BUDGET_LINES_FIELD_NUMBER: _ClassVar[int]
    WAKE_BUDGET_CHARS_FIELD_NUMBER: _ClassVar[int]
    MAX_ENTRY_LINES_FIELD_NUMBER: _ClassVar[int]
    MAX_ENTRY_CHARS_FIELD_NUMBER: _ClassVar[int]
    FRONTIER_TARGET_ORIGIN_FIELD_NUMBER: _ClassVar[int]
    WAKE_BUDGET_LINES_ORIGIN_FIELD_NUMBER: _ClassVar[int]
    WAKE_BUDGET_CHARS_ORIGIN_FIELD_NUMBER: _ClassVar[int]
    MAX_ENTRY_LINES_ORIGIN_FIELD_NUMBER: _ClassVar[int]
    MAX_ENTRY_CHARS_ORIGIN_FIELD_NUMBER: _ClassVar[int]
    frontier_target: int
    wake_budget_lines: int
    wake_budget_chars: int
    max_entry_lines: int
    max_entry_chars: int
    frontier_target_origin: str
    wake_budget_lines_origin: str
    wake_budget_chars_origin: str
    max_entry_lines_origin: str
    max_entry_chars_origin: str
    def __init__(self, frontier_target: _Optional[int] = ..., wake_budget_lines: _Optional[int] = ..., wake_budget_chars: _Optional[int] = ..., max_entry_lines: _Optional[int] = ..., max_entry_chars: _Optional[int] = ..., frontier_target_origin: _Optional[str] = ..., wake_budget_lines_origin: _Optional[str] = ..., wake_budget_chars_origin: _Optional[str] = ..., max_entry_lines_origin: _Optional[str] = ..., max_entry_chars_origin: _Optional[str] = ...) -> None: ...

class GetPolicyRequest(_message.Message):
    __slots__ = ("scope",)
    SCOPE_FIELD_NUMBER: _ClassVar[int]
    scope: str
    def __init__(self, scope: _Optional[str] = ...) -> None: ...

class GetPolicyResponse(_message.Message):
    __slots__ = ("effective", "defaults", "liveness")
    EFFECTIVE_FIELD_NUMBER: _ClassVar[int]
    DEFAULTS_FIELD_NUMBER: _ClassVar[int]
    LIVENESS_FIELD_NUMBER: _ClassVar[int]
    effective: PolicySnapshot
    defaults: PolicySnapshot
    liveness: CompactionLiveness
    def __init__(self, effective: _Optional[_Union[PolicySnapshot, _Mapping]] = ..., defaults: _Optional[_Union[PolicySnapshot, _Mapping]] = ..., liveness: _Optional[_Union[CompactionLiveness, _Mapping]] = ...) -> None: ...

class CompactionLiveness(_message.Message):
    __slots__ = ("unsummarized_leaf_count", "oldest_unsummarized_leaf_at", "last_summary_at")
    UNSUMMARIZED_LEAF_COUNT_FIELD_NUMBER: _ClassVar[int]
    OLDEST_UNSUMMARIZED_LEAF_AT_FIELD_NUMBER: _ClassVar[int]
    LAST_SUMMARY_AT_FIELD_NUMBER: _ClassVar[int]
    unsummarized_leaf_count: int
    oldest_unsummarized_leaf_at: str
    last_summary_at: str
    def __init__(self, unsummarized_leaf_count: _Optional[int] = ..., oldest_unsummarized_leaf_at: _Optional[str] = ..., last_summary_at: _Optional[str] = ...) -> None: ...

class SetPolicyRequest(_message.Message):
    __slots__ = ("scope", "frontier_target", "wake_budget_lines", "wake_budget_chars", "max_entry_lines", "max_entry_chars")
    SCOPE_FIELD_NUMBER: _ClassVar[int]
    FRONTIER_TARGET_FIELD_NUMBER: _ClassVar[int]
    WAKE_BUDGET_LINES_FIELD_NUMBER: _ClassVar[int]
    WAKE_BUDGET_CHARS_FIELD_NUMBER: _ClassVar[int]
    MAX_ENTRY_LINES_FIELD_NUMBER: _ClassVar[int]
    MAX_ENTRY_CHARS_FIELD_NUMBER: _ClassVar[int]
    scope: str
    frontier_target: int
    wake_budget_lines: int
    wake_budget_chars: int
    max_entry_lines: int
    max_entry_chars: int
    def __init__(self, scope: _Optional[str] = ..., frontier_target: _Optional[int] = ..., wake_budget_lines: _Optional[int] = ..., wake_budget_chars: _Optional[int] = ..., max_entry_lines: _Optional[int] = ..., max_entry_chars: _Optional[int] = ...) -> None: ...

class SetPolicyResponse(_message.Message):
    __slots__ = ("effective", "defaults", "liveness")
    EFFECTIVE_FIELD_NUMBER: _ClassVar[int]
    DEFAULTS_FIELD_NUMBER: _ClassVar[int]
    LIVENESS_FIELD_NUMBER: _ClassVar[int]
    effective: PolicySnapshot
    defaults: PolicySnapshot
    liveness: CompactionLiveness
    def __init__(self, effective: _Optional[_Union[PolicySnapshot, _Mapping]] = ..., defaults: _Optional[_Union[PolicySnapshot, _Mapping]] = ..., liveness: _Optional[_Union[CompactionLiveness, _Mapping]] = ...) -> None: ...

class ResetPolicyRequest(_message.Message):
    __slots__ = ("scope",)
    SCOPE_FIELD_NUMBER: _ClassVar[int]
    scope: str
    def __init__(self, scope: _Optional[str] = ...) -> None: ...

class ResetPolicyResponse(_message.Message):
    __slots__ = ("effective", "defaults", "liveness")
    EFFECTIVE_FIELD_NUMBER: _ClassVar[int]
    DEFAULTS_FIELD_NUMBER: _ClassVar[int]
    LIVENESS_FIELD_NUMBER: _ClassVar[int]
    effective: PolicySnapshot
    defaults: PolicySnapshot
    liveness: CompactionLiveness
    def __init__(self, effective: _Optional[_Union[PolicySnapshot, _Mapping]] = ..., defaults: _Optional[_Union[PolicySnapshot, _Mapping]] = ..., liveness: _Optional[_Union[CompactionLiveness, _Mapping]] = ...) -> None: ...
