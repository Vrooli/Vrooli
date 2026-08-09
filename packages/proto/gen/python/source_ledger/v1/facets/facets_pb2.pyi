from google.protobuf.internal import containers as _containers
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Iterable as _Iterable, Mapping as _Mapping
from typing import ClassVar as _ClassVar, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class Facet(_message.Message):
    __slots__ = ("id", "label", "retention_policy", "guidance", "compaction_eligible", "resident_budget")
    ID_FIELD_NUMBER: _ClassVar[int]
    LABEL_FIELD_NUMBER: _ClassVar[int]
    RETENTION_POLICY_FIELD_NUMBER: _ClassVar[int]
    GUIDANCE_FIELD_NUMBER: _ClassVar[int]
    COMPACTION_ELIGIBLE_FIELD_NUMBER: _ClassVar[int]
    RESIDENT_BUDGET_FIELD_NUMBER: _ClassVar[int]
    id: str
    label: str
    retention_policy: str
    guidance: str
    compaction_eligible: bool
    resident_budget: int
    def __init__(self, id: _Optional[str] = ..., label: _Optional[str] = ..., retention_policy: _Optional[str] = ..., guidance: _Optional[str] = ..., compaction_eligible: _Optional[bool] = ..., resident_budget: _Optional[int] = ...) -> None: ...

class PinProposal(_message.Message):
    __slots__ = ("id", "entry_ids", "rationale")
    ID_FIELD_NUMBER: _ClassVar[int]
    ENTRY_IDS_FIELD_NUMBER: _ClassVar[int]
    RATIONALE_FIELD_NUMBER: _ClassVar[int]
    id: str
    entry_ids: _containers.RepeatedScalarFieldContainer[str]
    rationale: str
    def __init__(self, id: _Optional[str] = ..., entry_ids: _Optional[_Iterable[str]] = ..., rationale: _Optional[str] = ...) -> None: ...

class ListFacetsRequest(_message.Message):
    __slots__ = ("scope",)
    SCOPE_FIELD_NUMBER: _ClassVar[int]
    scope: str
    def __init__(self, scope: _Optional[str] = ...) -> None: ...

class ListFacetsResponse(_message.Message):
    __slots__ = ("facets",)
    FACETS_FIELD_NUMBER: _ClassVar[int]
    facets: _containers.RepeatedCompositeFieldContainer[Facet]
    def __init__(self, facets: _Optional[_Iterable[_Union[Facet, _Mapping]]] = ...) -> None: ...

class SetFacetPolicyRequest(_message.Message):
    __slots__ = ("scope", "facet_id", "retention_policy", "compaction_eligible", "resident_budget")
    SCOPE_FIELD_NUMBER: _ClassVar[int]
    FACET_ID_FIELD_NUMBER: _ClassVar[int]
    RETENTION_POLICY_FIELD_NUMBER: _ClassVar[int]
    COMPACTION_ELIGIBLE_FIELD_NUMBER: _ClassVar[int]
    RESIDENT_BUDGET_FIELD_NUMBER: _ClassVar[int]
    scope: str
    facet_id: str
    retention_policy: str
    compaction_eligible: bool
    resident_budget: int
    def __init__(self, scope: _Optional[str] = ..., facet_id: _Optional[str] = ..., retention_policy: _Optional[str] = ..., compaction_eligible: _Optional[bool] = ..., resident_budget: _Optional[int] = ...) -> None: ...

class SetFacetPolicyResponse(_message.Message):
    __slots__ = ("facet",)
    FACET_FIELD_NUMBER: _ClassVar[int]
    facet: Facet
    def __init__(self, facet: _Optional[_Union[Facet, _Mapping]] = ...) -> None: ...

class AssignFacetRequest(_message.Message):
    __slots__ = ("entry_id", "facet_id", "scope")
    ENTRY_ID_FIELD_NUMBER: _ClassVar[int]
    FACET_ID_FIELD_NUMBER: _ClassVar[int]
    SCOPE_FIELD_NUMBER: _ClassVar[int]
    entry_id: str
    facet_id: str
    scope: str
    def __init__(self, entry_id: _Optional[str] = ..., facet_id: _Optional[str] = ..., scope: _Optional[str] = ...) -> None: ...

class AssignFacetResponse(_message.Message):
    __slots__ = ()
    def __init__(self) -> None: ...

class SetPinRequest(_message.Message):
    __slots__ = ("entry_id", "pinned", "scope")
    ENTRY_ID_FIELD_NUMBER: _ClassVar[int]
    PINNED_FIELD_NUMBER: _ClassVar[int]
    SCOPE_FIELD_NUMBER: _ClassVar[int]
    entry_id: str
    pinned: bool
    scope: str
    def __init__(self, entry_id: _Optional[str] = ..., pinned: _Optional[bool] = ..., scope: _Optional[str] = ...) -> None: ...

class SetPinResponse(_message.Message):
    __slots__ = ()
    def __init__(self) -> None: ...

class ListPinProposalsRequest(_message.Message):
    __slots__ = ("scope",)
    SCOPE_FIELD_NUMBER: _ClassVar[int]
    scope: str
    def __init__(self, scope: _Optional[str] = ...) -> None: ...

class ListPinProposalsResponse(_message.Message):
    __slots__ = ("proposals",)
    PROPOSALS_FIELD_NUMBER: _ClassVar[int]
    proposals: _containers.RepeatedCompositeFieldContainer[PinProposal]
    def __init__(self, proposals: _Optional[_Iterable[_Union[PinProposal, _Mapping]]] = ...) -> None: ...

class PinCandidate(_message.Message):
    __slots__ = ("entry_id", "body", "recall_count", "created_at", "last_recalled_at")
    ENTRY_ID_FIELD_NUMBER: _ClassVar[int]
    BODY_FIELD_NUMBER: _ClassVar[int]
    RECALL_COUNT_FIELD_NUMBER: _ClassVar[int]
    CREATED_AT_FIELD_NUMBER: _ClassVar[int]
    LAST_RECALLED_AT_FIELD_NUMBER: _ClassVar[int]
    entry_id: str
    body: str
    recall_count: int
    created_at: str
    last_recalled_at: str
    def __init__(self, entry_id: _Optional[str] = ..., body: _Optional[str] = ..., recall_count: _Optional[int] = ..., created_at: _Optional[str] = ..., last_recalled_at: _Optional[str] = ...) -> None: ...

class ListPinCandidatesRequest(_message.Message):
    __slots__ = ("limit", "scope")
    LIMIT_FIELD_NUMBER: _ClassVar[int]
    SCOPE_FIELD_NUMBER: _ClassVar[int]
    limit: int
    scope: str
    def __init__(self, limit: _Optional[int] = ..., scope: _Optional[str] = ...) -> None: ...

class ListPinCandidatesResponse(_message.Message):
    __slots__ = ("candidates",)
    CANDIDATES_FIELD_NUMBER: _ClassVar[int]
    candidates: _containers.RepeatedCompositeFieldContainer[PinCandidate]
    def __init__(self, candidates: _Optional[_Iterable[_Union[PinCandidate, _Mapping]]] = ...) -> None: ...

class ResolvePinProposalRequest(_message.Message):
    __slots__ = ("proposal_id", "accept", "scope")
    PROPOSAL_ID_FIELD_NUMBER: _ClassVar[int]
    ACCEPT_FIELD_NUMBER: _ClassVar[int]
    SCOPE_FIELD_NUMBER: _ClassVar[int]
    proposal_id: str
    accept: bool
    scope: str
    def __init__(self, proposal_id: _Optional[str] = ..., accept: _Optional[bool] = ..., scope: _Optional[str] = ...) -> None: ...

class ResolvePinProposalResponse(_message.Message):
    __slots__ = ()
    def __init__(self) -> None: ...

class MarkSupersededRequest(_message.Message):
    __slots__ = ("entry_id", "replacement_entry_id", "scope")
    ENTRY_ID_FIELD_NUMBER: _ClassVar[int]
    REPLACEMENT_ENTRY_ID_FIELD_NUMBER: _ClassVar[int]
    SCOPE_FIELD_NUMBER: _ClassVar[int]
    entry_id: str
    replacement_entry_id: str
    scope: str
    def __init__(self, entry_id: _Optional[str] = ..., replacement_entry_id: _Optional[str] = ..., scope: _Optional[str] = ...) -> None: ...

class MarkSupersededResponse(_message.Message):
    __slots__ = ()
    def __init__(self) -> None: ...

class ResolveThreadRequest(_message.Message):
    __slots__ = ("entry_id", "scope")
    ENTRY_ID_FIELD_NUMBER: _ClassVar[int]
    SCOPE_FIELD_NUMBER: _ClassVar[int]
    entry_id: str
    scope: str
    def __init__(self, entry_id: _Optional[str] = ..., scope: _Optional[str] = ...) -> None: ...

class ResolveThreadResponse(_message.Message):
    __slots__ = ()
    def __init__(self) -> None: ...
