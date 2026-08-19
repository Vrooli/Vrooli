import datetime

from google.protobuf import timestamp_pb2 as _timestamp_pb2
from treasury.v1.approval import approval_pb2 as _approval_pb2
from treasury.v1.book import book_pb2 as _book_pb2
from treasury.v1.budget import budget_pb2 as _budget_pb2
from treasury.v1.instrument import instrument_pb2 as _instrument_pb2
from treasury.v1.mandate import mandate_pb2 as _mandate_pb2
from treasury.v1.settlement import settlement_pb2 as _settlement_pb2
from google.protobuf.internal import containers as _containers
from google.protobuf.internal import enum_type_wrapper as _enum_type_wrapper
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Iterable as _Iterable, Mapping as _Mapping
from typing import ClassVar as _ClassVar, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class AuthorizationVerdict(int, metaclass=_enum_type_wrapper.EnumTypeWrapper):
    __slots__ = ()
    AUTHORIZATION_VERDICT_UNSPECIFIED: _ClassVar[AuthorizationVerdict]
    AUTHORIZATION_VERDICT_REFUSED: _ClassVar[AuthorizationVerdict]
    AUTHORIZATION_VERDICT_PENDING: _ClassVar[AuthorizationVerdict]
    AUTHORIZATION_VERDICT_APPROVED: _ClassVar[AuthorizationVerdict]
    AUTHORIZATION_VERDICT_RELEASED: _ClassVar[AuthorizationVerdict]
    AUTHORIZATION_VERDICT_SETTLED: _ClassVar[AuthorizationVerdict]
AUTHORIZATION_VERDICT_UNSPECIFIED: AuthorizationVerdict
AUTHORIZATION_VERDICT_REFUSED: AuthorizationVerdict
AUTHORIZATION_VERDICT_PENDING: AuthorizationVerdict
AUTHORIZATION_VERDICT_APPROVED: AuthorizationVerdict
AUTHORIZATION_VERDICT_RELEASED: AuthorizationVerdict
AUTHORIZATION_VERDICT_SETTLED: AuthorizationVerdict

class AuthorizationRecord(_message.Message):
    __slots__ = ("id", "mandate_id", "requesting_agent", "amount_minor", "currency", "counterparty", "verdict", "violated_constraint", "remediation", "hold_minor", "created_at", "expires_at", "idempotency_key", "budget_id")
    ID_FIELD_NUMBER: _ClassVar[int]
    MANDATE_ID_FIELD_NUMBER: _ClassVar[int]
    REQUESTING_AGENT_FIELD_NUMBER: _ClassVar[int]
    AMOUNT_MINOR_FIELD_NUMBER: _ClassVar[int]
    CURRENCY_FIELD_NUMBER: _ClassVar[int]
    COUNTERPARTY_FIELD_NUMBER: _ClassVar[int]
    VERDICT_FIELD_NUMBER: _ClassVar[int]
    VIOLATED_CONSTRAINT_FIELD_NUMBER: _ClassVar[int]
    REMEDIATION_FIELD_NUMBER: _ClassVar[int]
    HOLD_MINOR_FIELD_NUMBER: _ClassVar[int]
    CREATED_AT_FIELD_NUMBER: _ClassVar[int]
    EXPIRES_AT_FIELD_NUMBER: _ClassVar[int]
    IDEMPOTENCY_KEY_FIELD_NUMBER: _ClassVar[int]
    BUDGET_ID_FIELD_NUMBER: _ClassVar[int]
    id: str
    mandate_id: str
    requesting_agent: str
    amount_minor: int
    currency: str
    counterparty: str
    verdict: AuthorizationVerdict
    violated_constraint: str
    remediation: str
    hold_minor: int
    created_at: _timestamp_pb2.Timestamp
    expires_at: _timestamp_pb2.Timestamp
    idempotency_key: str
    budget_id: str
    def __init__(self, id: _Optional[str] = ..., mandate_id: _Optional[str] = ..., requesting_agent: _Optional[str] = ..., amount_minor: _Optional[int] = ..., currency: _Optional[str] = ..., counterparty: _Optional[str] = ..., verdict: _Optional[_Union[AuthorizationVerdict, str]] = ..., violated_constraint: _Optional[str] = ..., remediation: _Optional[str] = ..., hold_minor: _Optional[int] = ..., created_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., expires_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., idempotency_key: _Optional[str] = ..., budget_id: _Optional[str] = ...) -> None: ...

class ProposeChargeRequest(_message.Message):
    __slots__ = ("id", "idempotency_key", "mandate_id", "amount_minor", "currency", "counterparty")
    ID_FIELD_NUMBER: _ClassVar[int]
    IDEMPOTENCY_KEY_FIELD_NUMBER: _ClassVar[int]
    MANDATE_ID_FIELD_NUMBER: _ClassVar[int]
    AMOUNT_MINOR_FIELD_NUMBER: _ClassVar[int]
    CURRENCY_FIELD_NUMBER: _ClassVar[int]
    COUNTERPARTY_FIELD_NUMBER: _ClassVar[int]
    id: str
    idempotency_key: str
    mandate_id: str
    amount_minor: int
    currency: str
    counterparty: str
    def __init__(self, id: _Optional[str] = ..., idempotency_key: _Optional[str] = ..., mandate_id: _Optional[str] = ..., amount_minor: _Optional[int] = ..., currency: _Optional[str] = ..., counterparty: _Optional[str] = ...) -> None: ...

class ProposeChargeResponse(_message.Message):
    __slots__ = ("authorization",)
    AUTHORIZATION_FIELD_NUMBER: _ClassVar[int]
    authorization: AuthorizationRecord
    def __init__(self, authorization: _Optional[_Union[AuthorizationRecord, _Mapping]] = ...) -> None: ...

class GetAuthorizationRequest(_message.Message):
    __slots__ = ("id",)
    ID_FIELD_NUMBER: _ClassVar[int]
    id: str
    def __init__(self, id: _Optional[str] = ...) -> None: ...

class GetAuthorizationResponse(_message.Message):
    __slots__ = ("authorization",)
    AUTHORIZATION_FIELD_NUMBER: _ClassVar[int]
    authorization: AuthorizationRecord
    def __init__(self, authorization: _Optional[_Union[AuthorizationRecord, _Mapping]] = ...) -> None: ...

class GetBudgetHeadroomRequest(_message.Message):
    __slots__ = ("budget_id",)
    BUDGET_ID_FIELD_NUMBER: _ClassVar[int]
    budget_id: str
    def __init__(self, budget_id: _Optional[str] = ...) -> None: ...

class GetBudgetHeadroomResponse(_message.Message):
    __slots__ = ("headroom",)
    HEADROOM_FIELD_NUMBER: _ClassVar[int]
    headroom: _budget_pb2.Headroom
    def __init__(self, headroom: _Optional[_Union[_budget_pb2.Headroom, _Mapping]] = ...) -> None: ...

class ListMandatesRequest(_message.Message):
    __slots__ = ()
    def __init__(self) -> None: ...

class ListMandatesResponse(_message.Message):
    __slots__ = ("mandates",)
    MANDATES_FIELD_NUMBER: _ClassVar[int]
    mandates: _containers.RepeatedCompositeFieldContainer[_mandate_pb2.Mandate]
    def __init__(self, mandates: _Optional[_Iterable[_Union[_mandate_pb2.Mandate, _Mapping]]] = ...) -> None: ...

class ReportOutcomeRequest(_message.Message):
    __slots__ = ("authorization_id", "outcome", "rail_reference", "settlement_id", "instrument_id", "idempotency_key")
    AUTHORIZATION_ID_FIELD_NUMBER: _ClassVar[int]
    OUTCOME_FIELD_NUMBER: _ClassVar[int]
    RAIL_REFERENCE_FIELD_NUMBER: _ClassVar[int]
    SETTLEMENT_ID_FIELD_NUMBER: _ClassVar[int]
    INSTRUMENT_ID_FIELD_NUMBER: _ClassVar[int]
    IDEMPOTENCY_KEY_FIELD_NUMBER: _ClassVar[int]
    authorization_id: str
    outcome: str
    rail_reference: str
    settlement_id: str
    instrument_id: str
    idempotency_key: str
    def __init__(self, authorization_id: _Optional[str] = ..., outcome: _Optional[str] = ..., rail_reference: _Optional[str] = ..., settlement_id: _Optional[str] = ..., instrument_id: _Optional[str] = ..., idempotency_key: _Optional[str] = ...) -> None: ...

class ReportOutcomeResponse(_message.Message):
    __slots__ = ("authorization", "settlement")
    AUTHORIZATION_FIELD_NUMBER: _ClassVar[int]
    SETTLEMENT_FIELD_NUMBER: _ClassVar[int]
    authorization: AuthorizationRecord
    settlement: _settlement_pb2.Charge
    def __init__(self, authorization: _Optional[_Union[AuthorizationRecord, _Mapping]] = ..., settlement: _Optional[_Union[_settlement_pb2.Charge, _Mapping]] = ...) -> None: ...

class ReportManualOutcomeRequest(_message.Message):
    __slots__ = ("authorization_id", "settlement_id", "instrument_id", "idempotency_key", "attestation")
    AUTHORIZATION_ID_FIELD_NUMBER: _ClassVar[int]
    SETTLEMENT_ID_FIELD_NUMBER: _ClassVar[int]
    INSTRUMENT_ID_FIELD_NUMBER: _ClassVar[int]
    IDEMPOTENCY_KEY_FIELD_NUMBER: _ClassVar[int]
    ATTESTATION_FIELD_NUMBER: _ClassVar[int]
    authorization_id: str
    settlement_id: str
    instrument_id: str
    idempotency_key: str
    attestation: _settlement_pb2.ManualAttestation
    def __init__(self, authorization_id: _Optional[str] = ..., settlement_id: _Optional[str] = ..., instrument_id: _Optional[str] = ..., idempotency_key: _Optional[str] = ..., attestation: _Optional[_Union[_settlement_pb2.ManualAttestation, _Mapping]] = ...) -> None: ...

class ReportManualOutcomeResponse(_message.Message):
    __slots__ = ("authorization", "settlement")
    AUTHORIZATION_FIELD_NUMBER: _ClassVar[int]
    SETTLEMENT_FIELD_NUMBER: _ClassVar[int]
    authorization: AuthorizationRecord
    settlement: _settlement_pb2.Charge
    def __init__(self, authorization: _Optional[_Union[AuthorizationRecord, _Mapping]] = ..., settlement: _Optional[_Union[_settlement_pb2.Charge, _Mapping]] = ...) -> None: ...

class CreateBookRequest(_message.Message):
    __slots__ = ("book",)
    BOOK_FIELD_NUMBER: _ClassVar[int]
    book: _book_pb2.Book
    def __init__(self, book: _Optional[_Union[_book_pb2.Book, _Mapping]] = ...) -> None: ...

class CreateBookResponse(_message.Message):
    __slots__ = ("book",)
    BOOK_FIELD_NUMBER: _ClassVar[int]
    book: _book_pb2.Book
    def __init__(self, book: _Optional[_Union[_book_pb2.Book, _Mapping]] = ...) -> None: ...

class GetBookRequest(_message.Message):
    __slots__ = ("book_id",)
    BOOK_ID_FIELD_NUMBER: _ClassVar[int]
    book_id: str
    def __init__(self, book_id: _Optional[str] = ...) -> None: ...

class GetBookResponse(_message.Message):
    __slots__ = ("book",)
    BOOK_FIELD_NUMBER: _ClassVar[int]
    book: _book_pb2.Book
    def __init__(self, book: _Optional[_Union[_book_pb2.Book, _Mapping]] = ...) -> None: ...

class CreateMandateRequest(_message.Message):
    __slots__ = ("mandate",)
    MANDATE_FIELD_NUMBER: _ClassVar[int]
    mandate: _mandate_pb2.Mandate
    def __init__(self, mandate: _Optional[_Union[_mandate_pb2.Mandate, _Mapping]] = ...) -> None: ...

class CreateMandateResponse(_message.Message):
    __slots__ = ("mandate",)
    MANDATE_FIELD_NUMBER: _ClassVar[int]
    mandate: _mandate_pb2.Mandate
    def __init__(self, mandate: _Optional[_Union[_mandate_pb2.Mandate, _Mapping]] = ...) -> None: ...

class RevokeMandateRequest(_message.Message):
    __slots__ = ("mandate_id",)
    MANDATE_ID_FIELD_NUMBER: _ClassVar[int]
    mandate_id: str
    def __init__(self, mandate_id: _Optional[str] = ...) -> None: ...

class RevokeMandateResponse(_message.Message):
    __slots__ = ("mandate",)
    MANDATE_FIELD_NUMBER: _ClassVar[int]
    mandate: _mandate_pb2.Mandate
    def __init__(self, mandate: _Optional[_Union[_mandate_pb2.Mandate, _Mapping]] = ...) -> None: ...

class CancelStandingMandateRequest(_message.Message):
    __slots__ = ("mandate_id",)
    MANDATE_ID_FIELD_NUMBER: _ClassVar[int]
    mandate_id: str
    def __init__(self, mandate_id: _Optional[str] = ...) -> None: ...

class CancelStandingMandateResponse(_message.Message):
    __slots__ = ("mandate",)
    MANDATE_FIELD_NUMBER: _ClassVar[int]
    mandate: _mandate_pb2.Mandate
    def __init__(self, mandate: _Optional[_Union[_mandate_pb2.Mandate, _Mapping]] = ...) -> None: ...

class SetBudgetCapsRequest(_message.Message):
    __slots__ = ("budget",)
    BUDGET_FIELD_NUMBER: _ClassVar[int]
    budget: _budget_pb2.Budget
    def __init__(self, budget: _Optional[_Union[_budget_pb2.Budget, _Mapping]] = ...) -> None: ...

class SetBudgetCapsResponse(_message.Message):
    __slots__ = ("budget",)
    BUDGET_FIELD_NUMBER: _ClassVar[int]
    budget: _budget_pb2.Budget
    def __init__(self, budget: _Optional[_Union[_budget_pb2.Budget, _Mapping]] = ...) -> None: ...

class SetGatingRequest(_message.Message):
    __slots__ = ("budget_id", "requires_approval")
    BUDGET_ID_FIELD_NUMBER: _ClassVar[int]
    REQUIRES_APPROVAL_FIELD_NUMBER: _ClassVar[int]
    budget_id: str
    requires_approval: bool
    def __init__(self, budget_id: _Optional[str] = ..., requires_approval: _Optional[bool] = ...) -> None: ...

class SetGatingResponse(_message.Message):
    __slots__ = ("budget",)
    BUDGET_FIELD_NUMBER: _ClassVar[int]
    budget: _budget_pb2.Budget
    def __init__(self, budget: _Optional[_Union[_budget_pb2.Budget, _Mapping]] = ...) -> None: ...

class ResolveApprovalRequest(_message.Message):
    __slots__ = ("approval_id", "resolution")
    APPROVAL_ID_FIELD_NUMBER: _ClassVar[int]
    RESOLUTION_FIELD_NUMBER: _ClassVar[int]
    approval_id: str
    resolution: _approval_pb2.ApprovalStatus
    def __init__(self, approval_id: _Optional[str] = ..., resolution: _Optional[_Union[_approval_pb2.ApprovalStatus, str]] = ...) -> None: ...

class ResolveApprovalResponse(_message.Message):
    __slots__ = ("approval",)
    APPROVAL_FIELD_NUMBER: _ClassVar[int]
    approval: _approval_pb2.ApprovalRequest
    def __init__(self, approval: _Optional[_Union[_approval_pb2.ApprovalRequest, _Mapping]] = ...) -> None: ...

class ListApprovalsRequest(_message.Message):
    __slots__ = ("status",)
    STATUS_FIELD_NUMBER: _ClassVar[int]
    status: _approval_pb2.ApprovalStatus
    def __init__(self, status: _Optional[_Union[_approval_pb2.ApprovalStatus, str]] = ...) -> None: ...

class ListApprovalsResponse(_message.Message):
    __slots__ = ("approvals",)
    APPROVALS_FIELD_NUMBER: _ClassVar[int]
    approvals: _containers.RepeatedCompositeFieldContainer[_approval_pb2.ApprovalRequest]
    def __init__(self, approvals: _Optional[_Iterable[_Union[_approval_pb2.ApprovalRequest, _Mapping]]] = ...) -> None: ...

class FreezeBudgetRequest(_message.Message):
    __slots__ = ("budget_id",)
    BUDGET_ID_FIELD_NUMBER: _ClassVar[int]
    budget_id: str
    def __init__(self, budget_id: _Optional[str] = ...) -> None: ...

class FreezeBudgetResponse(_message.Message):
    __slots__ = ("budget",)
    BUDGET_FIELD_NUMBER: _ClassVar[int]
    budget: _budget_pb2.Budget
    def __init__(self, budget: _Optional[_Union[_budget_pb2.Budget, _Mapping]] = ...) -> None: ...

class UnfreezeBudgetRequest(_message.Message):
    __slots__ = ("budget_id",)
    BUDGET_ID_FIELD_NUMBER: _ClassVar[int]
    budget_id: str
    def __init__(self, budget_id: _Optional[str] = ...) -> None: ...

class UnfreezeBudgetResponse(_message.Message):
    __slots__ = ("budget",)
    BUDGET_FIELD_NUMBER: _ClassVar[int]
    budget: _budget_pb2.Budget
    def __init__(self, budget: _Optional[_Union[_budget_pb2.Budget, _Mapping]] = ...) -> None: ...

class FreezeBookRequest(_message.Message):
    __slots__ = ("book_id",)
    BOOK_ID_FIELD_NUMBER: _ClassVar[int]
    book_id: str
    def __init__(self, book_id: _Optional[str] = ...) -> None: ...

class FreezeBookResponse(_message.Message):
    __slots__ = ("book_id", "frozen", "updated_at")
    BOOK_ID_FIELD_NUMBER: _ClassVar[int]
    FROZEN_FIELD_NUMBER: _ClassVar[int]
    UPDATED_AT_FIELD_NUMBER: _ClassVar[int]
    book_id: str
    frozen: bool
    updated_at: _timestamp_pb2.Timestamp
    def __init__(self, book_id: _Optional[str] = ..., frozen: _Optional[bool] = ..., updated_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ...) -> None: ...

class UnfreezeBookRequest(_message.Message):
    __slots__ = ("book_id",)
    BOOK_ID_FIELD_NUMBER: _ClassVar[int]
    book_id: str
    def __init__(self, book_id: _Optional[str] = ...) -> None: ...

class UnfreezeBookResponse(_message.Message):
    __slots__ = ("book_id", "frozen", "updated_at")
    BOOK_ID_FIELD_NUMBER: _ClassVar[int]
    FROZEN_FIELD_NUMBER: _ClassVar[int]
    UPDATED_AT_FIELD_NUMBER: _ClassVar[int]
    book_id: str
    frozen: bool
    updated_at: _timestamp_pb2.Timestamp
    def __init__(self, book_id: _Optional[str] = ..., frozen: _Optional[bool] = ..., updated_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ...) -> None: ...

class FreezeAllRequest(_message.Message):
    __slots__ = ()
    def __init__(self) -> None: ...

class FreezeAllResponse(_message.Message):
    __slots__ = ("frozen", "updated_at")
    FROZEN_FIELD_NUMBER: _ClassVar[int]
    UPDATED_AT_FIELD_NUMBER: _ClassVar[int]
    frozen: bool
    updated_at: _timestamp_pb2.Timestamp
    def __init__(self, frozen: _Optional[bool] = ..., updated_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ...) -> None: ...

class UnfreezeAllRequest(_message.Message):
    __slots__ = ()
    def __init__(self) -> None: ...

class UnfreezeAllResponse(_message.Message):
    __slots__ = ("frozen", "updated_at")
    FROZEN_FIELD_NUMBER: _ClassVar[int]
    UPDATED_AT_FIELD_NUMBER: _ClassVar[int]
    frozen: bool
    updated_at: _timestamp_pb2.Timestamp
    def __init__(self, frozen: _Optional[bool] = ..., updated_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ...) -> None: ...

class RegisterInstrumentRequest(_message.Message):
    __slots__ = ("instrument",)
    INSTRUMENT_FIELD_NUMBER: _ClassVar[int]
    instrument: _instrument_pb2.Instrument
    def __init__(self, instrument: _Optional[_Union[_instrument_pb2.Instrument, _Mapping]] = ...) -> None: ...

class RegisterInstrumentResponse(_message.Message):
    __slots__ = ("instrument",)
    INSTRUMENT_FIELD_NUMBER: _ClassVar[int]
    instrument: _instrument_pb2.Instrument
    def __init__(self, instrument: _Optional[_Union[_instrument_pb2.Instrument, _Mapping]] = ...) -> None: ...
