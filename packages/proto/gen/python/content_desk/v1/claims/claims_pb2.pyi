from google.protobuf.internal import containers as _containers
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Iterable as _Iterable, Mapping as _Mapping
from typing import ClassVar as _ClassVar, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class Claim(_message.Message):
    __slots__ = ("id", "statement", "verification_status", "kind")
    ID_FIELD_NUMBER: _ClassVar[int]
    STATEMENT_FIELD_NUMBER: _ClassVar[int]
    VERIFICATION_STATUS_FIELD_NUMBER: _ClassVar[int]
    KIND_FIELD_NUMBER: _ClassVar[int]
    id: str
    statement: str
    verification_status: str
    kind: str
    def __init__(self, id: _Optional[str] = ..., statement: _Optional[str] = ..., verification_status: _Optional[str] = ..., kind: _Optional[str] = ...) -> None: ...

class ListClaimsRequest(_message.Message):
    __slots__ = ()
    def __init__(self) -> None: ...

class ListClaimsResponse(_message.Message):
    __slots__ = ("claims",)
    CLAIMS_FIELD_NUMBER: _ClassVar[int]
    claims: _containers.RepeatedCompositeFieldContainer[Claim]
    def __init__(self, claims: _Optional[_Iterable[_Union[Claim, _Mapping]]] = ...) -> None: ...

class ListDraftClaimsRequest(_message.Message):
    __slots__ = ("draft_id",)
    DRAFT_ID_FIELD_NUMBER: _ClassVar[int]
    draft_id: str
    def __init__(self, draft_id: _Optional[str] = ...) -> None: ...

class ListDraftClaimsResponse(_message.Message):
    __slots__ = ("claims",)
    CLAIMS_FIELD_NUMBER: _ClassVar[int]
    claims: _containers.RepeatedCompositeFieldContainer[Claim]
    def __init__(self, claims: _Optional[_Iterable[_Union[Claim, _Mapping]]] = ...) -> None: ...

class CreateClaimRequest(_message.Message):
    __slots__ = ("statement", "kind", "evidence_kind", "reference", "command", "expected_result", "observed_at")
    STATEMENT_FIELD_NUMBER: _ClassVar[int]
    KIND_FIELD_NUMBER: _ClassVar[int]
    EVIDENCE_KIND_FIELD_NUMBER: _ClassVar[int]
    REFERENCE_FIELD_NUMBER: _ClassVar[int]
    COMMAND_FIELD_NUMBER: _ClassVar[int]
    EXPECTED_RESULT_FIELD_NUMBER: _ClassVar[int]
    OBSERVED_AT_FIELD_NUMBER: _ClassVar[int]
    statement: str
    kind: str
    evidence_kind: str
    reference: str
    command: str
    expected_result: str
    observed_at: str
    def __init__(self, statement: _Optional[str] = ..., kind: _Optional[str] = ..., evidence_kind: _Optional[str] = ..., reference: _Optional[str] = ..., command: _Optional[str] = ..., expected_result: _Optional[str] = ..., observed_at: _Optional[str] = ...) -> None: ...

class CreateClaimResponse(_message.Message):
    __slots__ = ("claim",)
    CLAIM_FIELD_NUMBER: _ClassVar[int]
    claim: Claim
    def __init__(self, claim: _Optional[_Union[Claim, _Mapping]] = ...) -> None: ...

class CiteClaimRequest(_message.Message):
    __slots__ = ("draft_id", "claim_id", "span_start", "span_end", "body")
    DRAFT_ID_FIELD_NUMBER: _ClassVar[int]
    CLAIM_ID_FIELD_NUMBER: _ClassVar[int]
    SPAN_START_FIELD_NUMBER: _ClassVar[int]
    SPAN_END_FIELD_NUMBER: _ClassVar[int]
    BODY_FIELD_NUMBER: _ClassVar[int]
    draft_id: str
    claim_id: str
    span_start: int
    span_end: int
    body: str
    def __init__(self, draft_id: _Optional[str] = ..., claim_id: _Optional[str] = ..., span_start: _Optional[int] = ..., span_end: _Optional[int] = ..., body: _Optional[str] = ...) -> None: ...

class CiteClaimResponse(_message.Message):
    __slots__ = ()
    def __init__(self) -> None: ...

class VerifyClaimRequest(_message.Message):
    __slots__ = ("id",)
    ID_FIELD_NUMBER: _ClassVar[int]
    id: str
    def __init__(self, id: _Optional[str] = ...) -> None: ...

class VerifyClaimResponse(_message.Message):
    __slots__ = ("claim",)
    CLAIM_FIELD_NUMBER: _ClassVar[int]
    claim: Claim
    def __init__(self, claim: _Optional[_Union[Claim, _Mapping]] = ...) -> None: ...

class SweepClaimsRequest(_message.Message):
    __slots__ = ()
    def __init__(self) -> None: ...

class SweepClaimsResponse(_message.Message):
    __slots__ = ("claims",)
    CLAIMS_FIELD_NUMBER: _ClassVar[int]
    claims: _containers.RepeatedCompositeFieldContainer[Claim]
    def __init__(self, claims: _Optional[_Iterable[_Union[Claim, _Mapping]]] = ...) -> None: ...
