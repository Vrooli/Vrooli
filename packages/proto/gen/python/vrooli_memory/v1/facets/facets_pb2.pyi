from google.protobuf.internal import containers as _containers
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Iterable as _Iterable, Mapping as _Mapping
from typing import ClassVar as _ClassVar, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class Facet(_message.Message):
    __slots__ = ("id", "label", "retention_policy")
    ID_FIELD_NUMBER: _ClassVar[int]
    LABEL_FIELD_NUMBER: _ClassVar[int]
    RETENTION_POLICY_FIELD_NUMBER: _ClassVar[int]
    id: str
    label: str
    retention_policy: str
    def __init__(self, id: _Optional[str] = ..., label: _Optional[str] = ..., retention_policy: _Optional[str] = ...) -> None: ...

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
    __slots__ = ()
    def __init__(self) -> None: ...

class ListFacetsResponse(_message.Message):
    __slots__ = ("facets",)
    FACETS_FIELD_NUMBER: _ClassVar[int]
    facets: _containers.RepeatedCompositeFieldContainer[Facet]
    def __init__(self, facets: _Optional[_Iterable[_Union[Facet, _Mapping]]] = ...) -> None: ...

class AssignFacetRequest(_message.Message):
    __slots__ = ("entry_id", "facet_id")
    ENTRY_ID_FIELD_NUMBER: _ClassVar[int]
    FACET_ID_FIELD_NUMBER: _ClassVar[int]
    entry_id: str
    facet_id: str
    def __init__(self, entry_id: _Optional[str] = ..., facet_id: _Optional[str] = ...) -> None: ...

class AssignFacetResponse(_message.Message):
    __slots__ = ()
    def __init__(self) -> None: ...

class SetPinRequest(_message.Message):
    __slots__ = ("entry_id", "pinned")
    ENTRY_ID_FIELD_NUMBER: _ClassVar[int]
    PINNED_FIELD_NUMBER: _ClassVar[int]
    entry_id: str
    pinned: bool
    def __init__(self, entry_id: _Optional[str] = ..., pinned: _Optional[bool] = ...) -> None: ...

class SetPinResponse(_message.Message):
    __slots__ = ()
    def __init__(self) -> None: ...

class ListPinProposalsRequest(_message.Message):
    __slots__ = ()
    def __init__(self) -> None: ...

class ListPinProposalsResponse(_message.Message):
    __slots__ = ("proposals",)
    PROPOSALS_FIELD_NUMBER: _ClassVar[int]
    proposals: _containers.RepeatedCompositeFieldContainer[PinProposal]
    def __init__(self, proposals: _Optional[_Iterable[_Union[PinProposal, _Mapping]]] = ...) -> None: ...

class ResolvePinProposalRequest(_message.Message):
    __slots__ = ("proposal_id", "accept")
    PROPOSAL_ID_FIELD_NUMBER: _ClassVar[int]
    ACCEPT_FIELD_NUMBER: _ClassVar[int]
    proposal_id: str
    accept: bool
    def __init__(self, proposal_id: _Optional[str] = ..., accept: _Optional[bool] = ...) -> None: ...

class ResolvePinProposalResponse(_message.Message):
    __slots__ = ()
    def __init__(self) -> None: ...

class MarkSupersededRequest(_message.Message):
    __slots__ = ("entry_id", "replacement_entry_id")
    ENTRY_ID_FIELD_NUMBER: _ClassVar[int]
    REPLACEMENT_ENTRY_ID_FIELD_NUMBER: _ClassVar[int]
    entry_id: str
    replacement_entry_id: str
    def __init__(self, entry_id: _Optional[str] = ..., replacement_entry_id: _Optional[str] = ...) -> None: ...

class MarkSupersededResponse(_message.Message):
    __slots__ = ()
    def __init__(self) -> None: ...
