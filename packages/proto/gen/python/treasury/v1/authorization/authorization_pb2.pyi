import datetime

from google.protobuf import timestamp_pb2 as _timestamp_pb2
from treasury.v1.settlement import settlement_pb2 as _settlement_pb2
from treasury.v1.shared import approval_pb2 as _approval_pb2
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
    __slots__ = ("id", "mandate_id", "requesting_agent", "amount_minor", "currency", "counterparty", "verdict", "violated_constraint", "remediation", "hold_minor", "created_at", "expires_at", "idempotency_key", "budget_id", "book_id", "settlement_id", "instrument_id", "settlement_idempotency_key")
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
    BOOK_ID_FIELD_NUMBER: _ClassVar[int]
    SETTLEMENT_ID_FIELD_NUMBER: _ClassVar[int]
    INSTRUMENT_ID_FIELD_NUMBER: _ClassVar[int]
    SETTLEMENT_IDEMPOTENCY_KEY_FIELD_NUMBER: _ClassVar[int]
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
    book_id: str
    settlement_id: str
    instrument_id: str
    settlement_idempotency_key: str
    def __init__(self, id: _Optional[str] = ..., mandate_id: _Optional[str] = ..., requesting_agent: _Optional[str] = ..., amount_minor: _Optional[int] = ..., currency: _Optional[str] = ..., counterparty: _Optional[str] = ..., verdict: _Optional[_Union[AuthorizationVerdict, str]] = ..., violated_constraint: _Optional[str] = ..., remediation: _Optional[str] = ..., hold_minor: _Optional[int] = ..., created_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., expires_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., idempotency_key: _Optional[str] = ..., budget_id: _Optional[str] = ..., book_id: _Optional[str] = ..., settlement_id: _Optional[str] = ..., instrument_id: _Optional[str] = ..., settlement_idempotency_key: _Optional[str] = ...) -> None: ...

class ProposeChargeRequest(_message.Message):
    __slots__ = ("id", "idempotency_key", "mandate_id", "amount_minor", "currency", "counterparty", "settlement_id", "instrument_id", "settlement_idempotency_key")
    ID_FIELD_NUMBER: _ClassVar[int]
    IDEMPOTENCY_KEY_FIELD_NUMBER: _ClassVar[int]
    MANDATE_ID_FIELD_NUMBER: _ClassVar[int]
    AMOUNT_MINOR_FIELD_NUMBER: _ClassVar[int]
    CURRENCY_FIELD_NUMBER: _ClassVar[int]
    COUNTERPARTY_FIELD_NUMBER: _ClassVar[int]
    SETTLEMENT_ID_FIELD_NUMBER: _ClassVar[int]
    INSTRUMENT_ID_FIELD_NUMBER: _ClassVar[int]
    SETTLEMENT_IDEMPOTENCY_KEY_FIELD_NUMBER: _ClassVar[int]
    id: str
    idempotency_key: str
    mandate_id: str
    amount_minor: int
    currency: str
    counterparty: str
    settlement_id: str
    instrument_id: str
    settlement_idempotency_key: str
    def __init__(self, id: _Optional[str] = ..., idempotency_key: _Optional[str] = ..., mandate_id: _Optional[str] = ..., amount_minor: _Optional[int] = ..., currency: _Optional[str] = ..., counterparty: _Optional[str] = ..., settlement_id: _Optional[str] = ..., instrument_id: _Optional[str] = ..., settlement_idempotency_key: _Optional[str] = ...) -> None: ...

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
    headroom: AgentBudgetHeadroom
    def __init__(self, headroom: _Optional[_Union[AgentBudgetHeadroom, _Mapping]] = ...) -> None: ...

class AgentBudgetHeadroom(_message.Message):
    __slots__ = ("budget_id", "book_id", "currency", "total_cap_minor", "total_used_minor", "total_remaining_minor", "periodic_cap_minor", "period_used_minor", "period_remaining_minor", "per_transaction_cap_minor", "available_minor", "period_started_at", "computed_at", "funding_required", "funding_credited_minor", "funding_remaining_minor", "funding_atomic_amount", "funding_asset_id")
    BUDGET_ID_FIELD_NUMBER: _ClassVar[int]
    BOOK_ID_FIELD_NUMBER: _ClassVar[int]
    CURRENCY_FIELD_NUMBER: _ClassVar[int]
    TOTAL_CAP_MINOR_FIELD_NUMBER: _ClassVar[int]
    TOTAL_USED_MINOR_FIELD_NUMBER: _ClassVar[int]
    TOTAL_REMAINING_MINOR_FIELD_NUMBER: _ClassVar[int]
    PERIODIC_CAP_MINOR_FIELD_NUMBER: _ClassVar[int]
    PERIOD_USED_MINOR_FIELD_NUMBER: _ClassVar[int]
    PERIOD_REMAINING_MINOR_FIELD_NUMBER: _ClassVar[int]
    PER_TRANSACTION_CAP_MINOR_FIELD_NUMBER: _ClassVar[int]
    AVAILABLE_MINOR_FIELD_NUMBER: _ClassVar[int]
    PERIOD_STARTED_AT_FIELD_NUMBER: _ClassVar[int]
    COMPUTED_AT_FIELD_NUMBER: _ClassVar[int]
    FUNDING_REQUIRED_FIELD_NUMBER: _ClassVar[int]
    FUNDING_CREDITED_MINOR_FIELD_NUMBER: _ClassVar[int]
    FUNDING_REMAINING_MINOR_FIELD_NUMBER: _ClassVar[int]
    FUNDING_ATOMIC_AMOUNT_FIELD_NUMBER: _ClassVar[int]
    FUNDING_ASSET_ID_FIELD_NUMBER: _ClassVar[int]
    budget_id: str
    book_id: str
    currency: str
    total_cap_minor: int
    total_used_minor: int
    total_remaining_minor: int
    periodic_cap_minor: int
    period_used_minor: int
    period_remaining_minor: int
    per_transaction_cap_minor: int
    available_minor: int
    period_started_at: _timestamp_pb2.Timestamp
    computed_at: _timestamp_pb2.Timestamp
    funding_required: bool
    funding_credited_minor: int
    funding_remaining_minor: int
    funding_atomic_amount: str
    funding_asset_id: str
    def __init__(self, budget_id: _Optional[str] = ..., book_id: _Optional[str] = ..., currency: _Optional[str] = ..., total_cap_minor: _Optional[int] = ..., total_used_minor: _Optional[int] = ..., total_remaining_minor: _Optional[int] = ..., periodic_cap_minor: _Optional[int] = ..., period_used_minor: _Optional[int] = ..., period_remaining_minor: _Optional[int] = ..., per_transaction_cap_minor: _Optional[int] = ..., available_minor: _Optional[int] = ..., period_started_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., computed_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., funding_required: _Optional[bool] = ..., funding_credited_minor: _Optional[int] = ..., funding_remaining_minor: _Optional[int] = ..., funding_atomic_amount: _Optional[str] = ..., funding_asset_id: _Optional[str] = ...) -> None: ...

class AgentListMandatesRequest(_message.Message):
    __slots__ = ()
    def __init__(self) -> None: ...

class AgentMandate(_message.Message):
    __slots__ = ("id", "book_id", "budget_id", "cap_minor", "currency", "allowed_counterparties", "denied_counterparties", "expires_at", "status", "required_evidence", "recurrence_seconds", "next_charge_at")
    ID_FIELD_NUMBER: _ClassVar[int]
    BOOK_ID_FIELD_NUMBER: _ClassVar[int]
    BUDGET_ID_FIELD_NUMBER: _ClassVar[int]
    CAP_MINOR_FIELD_NUMBER: _ClassVar[int]
    CURRENCY_FIELD_NUMBER: _ClassVar[int]
    ALLOWED_COUNTERPARTIES_FIELD_NUMBER: _ClassVar[int]
    DENIED_COUNTERPARTIES_FIELD_NUMBER: _ClassVar[int]
    EXPIRES_AT_FIELD_NUMBER: _ClassVar[int]
    STATUS_FIELD_NUMBER: _ClassVar[int]
    REQUIRED_EVIDENCE_FIELD_NUMBER: _ClassVar[int]
    RECURRENCE_SECONDS_FIELD_NUMBER: _ClassVar[int]
    NEXT_CHARGE_AT_FIELD_NUMBER: _ClassVar[int]
    id: str
    book_id: str
    budget_id: str
    cap_minor: int
    currency: str
    allowed_counterparties: _containers.RepeatedScalarFieldContainer[str]
    denied_counterparties: _containers.RepeatedScalarFieldContainer[str]
    expires_at: _timestamp_pb2.Timestamp
    status: str
    required_evidence: _containers.RepeatedScalarFieldContainer[str]
    recurrence_seconds: int
    next_charge_at: _timestamp_pb2.Timestamp
    def __init__(self, id: _Optional[str] = ..., book_id: _Optional[str] = ..., budget_id: _Optional[str] = ..., cap_minor: _Optional[int] = ..., currency: _Optional[str] = ..., allowed_counterparties: _Optional[_Iterable[str]] = ..., denied_counterparties: _Optional[_Iterable[str]] = ..., expires_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., status: _Optional[str] = ..., required_evidence: _Optional[_Iterable[str]] = ..., recurrence_seconds: _Optional[int] = ..., next_charge_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ...) -> None: ...

class AgentListMandatesResponse(_message.Message):
    __slots__ = ("mandates",)
    MANDATES_FIELD_NUMBER: _ClassVar[int]
    mandates: _containers.RepeatedCompositeFieldContainer[AgentMandate]
    def __init__(self, mandates: _Optional[_Iterable[_Union[AgentMandate, _Mapping]]] = ...) -> None: ...

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

class ResolveUnknownSettlementRequest(_message.Message):
    __slots__ = ("settlement_id", "confirm_prior_process_stopped")
    SETTLEMENT_ID_FIELD_NUMBER: _ClassVar[int]
    CONFIRM_PRIOR_PROCESS_STOPPED_FIELD_NUMBER: _ClassVar[int]
    settlement_id: str
    confirm_prior_process_stopped: bool
    def __init__(self, settlement_id: _Optional[str] = ..., confirm_prior_process_stopped: _Optional[bool] = ...) -> None: ...

class ResolveUnknownSettlementResponse(_message.Message):
    __slots__ = ("authorization", "settlement")
    AUTHORIZATION_FIELD_NUMBER: _ClassVar[int]
    SETTLEMENT_FIELD_NUMBER: _ClassVar[int]
    authorization: AuthorizationRecord
    settlement: _settlement_pb2.Charge
    def __init__(self, authorization: _Optional[_Union[AuthorizationRecord, _Mapping]] = ..., settlement: _Optional[_Union[_settlement_pb2.Charge, _Mapping]] = ...) -> None: ...

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
    __slots__ = ("status", "book_id")
    STATUS_FIELD_NUMBER: _ClassVar[int]
    BOOK_ID_FIELD_NUMBER: _ClassVar[int]
    status: _approval_pb2.ApprovalStatus
    book_id: str
    def __init__(self, status: _Optional[_Union[_approval_pb2.ApprovalStatus, str]] = ..., book_id: _Optional[str] = ...) -> None: ...

class ListApprovalsResponse(_message.Message):
    __slots__ = ("approvals",)
    APPROVALS_FIELD_NUMBER: _ClassVar[int]
    approvals: _containers.RepeatedCompositeFieldContainer[_approval_pb2.ApprovalRequest]
    def __init__(self, approvals: _Optional[_Iterable[_Union[_approval_pb2.ApprovalRequest, _Mapping]]] = ...) -> None: ...
