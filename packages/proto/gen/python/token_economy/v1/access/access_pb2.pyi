import datetime

from google.protobuf import timestamp_pb2 as _timestamp_pb2
from google.protobuf.internal import containers as _containers
from google.protobuf.internal import enum_type_wrapper as _enum_type_wrapper
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Iterable as _Iterable, Mapping as _Mapping
from typing import ClassVar as _ClassVar, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class SupplyPolicy(int, metaclass=_enum_type_wrapper.EnumTypeWrapper):
    __slots__ = ()
    SUPPLY_POLICY_UNSPECIFIED: _ClassVar[SupplyPolicy]
    SUPPLY_POLICY_UNBOUNDED: _ClassVar[SupplyPolicy]
    SUPPLY_POLICY_CAPPED: _ClassVar[SupplyPolicy]
    SUPPLY_POLICY_FIXED: _ClassVar[SupplyPolicy]

class GrantStatus(int, metaclass=_enum_type_wrapper.EnumTypeWrapper):
    __slots__ = ()
    GRANT_STATUS_UNSPECIFIED: _ClassVar[GrantStatus]
    GRANT_STATUS_DRAFT: _ClassVar[GrantStatus]
    GRANT_STATUS_LIVE: _ClassVar[GrantStatus]
    GRANT_STATUS_EXHAUSTED: _ClassVar[GrantStatus]
    GRANT_STATUS_EXPIRED: _ClassVar[GrantStatus]
    GRANT_STATUS_REVOKED: _ClassVar[GrantStatus]

class RuleCondition(int, metaclass=_enum_type_wrapper.EnumTypeWrapper):
    __slots__ = ()
    RULE_CONDITION_UNSPECIFIED: _ClassVar[RuleCondition]
    RULE_CONDITION_CATALOG_SCOPE_ALLOWED: _ClassVar[RuleCondition]
    RULE_CONDITION_CATALOG_SCOPE_DENIED: _ClassVar[RuleCondition]
    RULE_CONDITION_BEFORE_EXPIRY: _ClassVar[RuleCondition]
    RULE_CONDITION_REQUIRED_EVIDENCE: _ClassVar[RuleCondition]
    RULE_CONDITION_SUFFICIENT_BALANCE: _ClassVar[RuleCondition]

class ApprovalPosture(int, metaclass=_enum_type_wrapper.EnumTypeWrapper):
    __slots__ = ()
    APPROVAL_POSTURE_UNSPECIFIED: _ClassVar[ApprovalPosture]
    APPROVAL_POSTURE_IMMEDIATE: _ClassVar[ApprovalPosture]
    APPROVAL_POSTURE_REQUIRES_APPROVAL: _ClassVar[ApprovalPosture]

class EventKind(int, metaclass=_enum_type_wrapper.EnumTypeWrapper):
    __slots__ = ()
    EVENT_KIND_UNSPECIFIED: _ClassVar[EventKind]
    EVENT_KIND_MINT: _ClassVar[EventKind]
    EVENT_KIND_CREDIT: _ClassVar[EventKind]
    EVENT_KIND_DEBIT: _ClassVar[EventKind]
    EVENT_KIND_REVERSAL: _ClassVar[EventKind]
    EVENT_KIND_EXPIRY: _ClassVar[EventKind]

class ActorKind(int, metaclass=_enum_type_wrapper.EnumTypeWrapper):
    __slots__ = ()
    ACTOR_KIND_UNSPECIFIED: _ClassVar[ActorKind]
    ACTOR_KIND_OPERATOR: _ClassVar[ActorKind]
    ACTOR_KIND_AGENT: _ClassVar[ActorKind]

class VerificationStatus(int, metaclass=_enum_type_wrapper.EnumTypeWrapper):
    __slots__ = ()
    VERIFICATION_STATUS_UNSPECIFIED: _ClassVar[VerificationStatus]
    VERIFICATION_STATUS_VERIFIED: _ClassVar[VerificationStatus]
    VERIFICATION_STATUS_UNAVAILABLE: _ClassVar[VerificationStatus]
    VERIFICATION_STATUS_INVALID: _ClassVar[VerificationStatus]
    VERIFICATION_STATUS_ABSENT: _ClassVar[VerificationStatus]

class RedemptionState(int, metaclass=_enum_type_wrapper.EnumTypeWrapper):
    __slots__ = ()
    REDEMPTION_STATE_UNSPECIFIED: _ClassVar[RedemptionState]
    REDEMPTION_STATE_PENDING_APPROVAL: _ClassVar[RedemptionState]
    REDEMPTION_STATE_SETTLED: _ClassVar[RedemptionState]
    REDEMPTION_STATE_DENIED: _ClassVar[RedemptionState]
SUPPLY_POLICY_UNSPECIFIED: SupplyPolicy
SUPPLY_POLICY_UNBOUNDED: SupplyPolicy
SUPPLY_POLICY_CAPPED: SupplyPolicy
SUPPLY_POLICY_FIXED: SupplyPolicy
GRANT_STATUS_UNSPECIFIED: GrantStatus
GRANT_STATUS_DRAFT: GrantStatus
GRANT_STATUS_LIVE: GrantStatus
GRANT_STATUS_EXHAUSTED: GrantStatus
GRANT_STATUS_EXPIRED: GrantStatus
GRANT_STATUS_REVOKED: GrantStatus
RULE_CONDITION_UNSPECIFIED: RuleCondition
RULE_CONDITION_CATALOG_SCOPE_ALLOWED: RuleCondition
RULE_CONDITION_CATALOG_SCOPE_DENIED: RuleCondition
RULE_CONDITION_BEFORE_EXPIRY: RuleCondition
RULE_CONDITION_REQUIRED_EVIDENCE: RuleCondition
RULE_CONDITION_SUFFICIENT_BALANCE: RuleCondition
APPROVAL_POSTURE_UNSPECIFIED: ApprovalPosture
APPROVAL_POSTURE_IMMEDIATE: ApprovalPosture
APPROVAL_POSTURE_REQUIRES_APPROVAL: ApprovalPosture
EVENT_KIND_UNSPECIFIED: EventKind
EVENT_KIND_MINT: EventKind
EVENT_KIND_CREDIT: EventKind
EVENT_KIND_DEBIT: EventKind
EVENT_KIND_REVERSAL: EventKind
EVENT_KIND_EXPIRY: EventKind
ACTOR_KIND_UNSPECIFIED: ActorKind
ACTOR_KIND_OPERATOR: ActorKind
ACTOR_KIND_AGENT: ActorKind
VERIFICATION_STATUS_UNSPECIFIED: VerificationStatus
VERIFICATION_STATUS_VERIFIED: VerificationStatus
VERIFICATION_STATUS_UNAVAILABLE: VerificationStatus
VERIFICATION_STATUS_INVALID: VerificationStatus
VERIFICATION_STATUS_ABSENT: VerificationStatus
REDEMPTION_STATE_UNSPECIFIED: RedemptionState
REDEMPTION_STATE_PENDING_APPROVAL: RedemptionState
REDEMPTION_STATE_SETTLED: RedemptionState
REDEMPTION_STATE_DENIED: RedemptionState

class MinterAuthority(_message.Message):
    __slots__ = ("token_type_id", "subject")
    TOKEN_TYPE_ID_FIELD_NUMBER: _ClassVar[int]
    SUBJECT_FIELD_NUMBER: _ClassVar[int]
    token_type_id: str
    subject: str
    def __init__(self, token_type_id: _Optional[str] = ..., subject: _Optional[str] = ...) -> None: ...

class TokenType(_message.Message):
    __slots__ = ("id", "name", "symbol", "color", "supply_policy", "cap_amount", "minted_amount", "authority", "retired", "created_at", "retired_at")
    ID_FIELD_NUMBER: _ClassVar[int]
    NAME_FIELD_NUMBER: _ClassVar[int]
    SYMBOL_FIELD_NUMBER: _ClassVar[int]
    COLOR_FIELD_NUMBER: _ClassVar[int]
    SUPPLY_POLICY_FIELD_NUMBER: _ClassVar[int]
    CAP_AMOUNT_FIELD_NUMBER: _ClassVar[int]
    MINTED_AMOUNT_FIELD_NUMBER: _ClassVar[int]
    AUTHORITY_FIELD_NUMBER: _ClassVar[int]
    RETIRED_FIELD_NUMBER: _ClassVar[int]
    CREATED_AT_FIELD_NUMBER: _ClassVar[int]
    RETIRED_AT_FIELD_NUMBER: _ClassVar[int]
    id: str
    name: str
    symbol: str
    color: str
    supply_policy: SupplyPolicy
    cap_amount: int
    minted_amount: int
    authority: MinterAuthority
    retired: bool
    created_at: _timestamp_pb2.Timestamp
    retired_at: _timestamp_pb2.Timestamp
    def __init__(self, id: _Optional[str] = ..., name: _Optional[str] = ..., symbol: _Optional[str] = ..., color: _Optional[str] = ..., supply_policy: _Optional[_Union[SupplyPolicy, str]] = ..., cap_amount: _Optional[int] = ..., minted_amount: _Optional[int] = ..., authority: _Optional[_Union[MinterAuthority, _Mapping]] = ..., retired: _Optional[bool] = ..., created_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., retired_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ...) -> None: ...

class CreateTokenTypeRequest(_message.Message):
    __slots__ = ("name", "symbol", "color", "supply_policy", "cap_amount", "minter_subject")
    NAME_FIELD_NUMBER: _ClassVar[int]
    SYMBOL_FIELD_NUMBER: _ClassVar[int]
    COLOR_FIELD_NUMBER: _ClassVar[int]
    SUPPLY_POLICY_FIELD_NUMBER: _ClassVar[int]
    CAP_AMOUNT_FIELD_NUMBER: _ClassVar[int]
    MINTER_SUBJECT_FIELD_NUMBER: _ClassVar[int]
    name: str
    symbol: str
    color: str
    supply_policy: SupplyPolicy
    cap_amount: int
    minter_subject: str
    def __init__(self, name: _Optional[str] = ..., symbol: _Optional[str] = ..., color: _Optional[str] = ..., supply_policy: _Optional[_Union[SupplyPolicy, str]] = ..., cap_amount: _Optional[int] = ..., minter_subject: _Optional[str] = ...) -> None: ...

class CreateTokenTypeResponse(_message.Message):
    __slots__ = ("token_type",)
    TOKEN_TYPE_FIELD_NUMBER: _ClassVar[int]
    token_type: TokenType
    def __init__(self, token_type: _Optional[_Union[TokenType, _Mapping]] = ...) -> None: ...

class GetTokenTypeRequest(_message.Message):
    __slots__ = ("id",)
    ID_FIELD_NUMBER: _ClassVar[int]
    id: str
    def __init__(self, id: _Optional[str] = ...) -> None: ...

class GetTokenTypeResponse(_message.Message):
    __slots__ = ("token_type",)
    TOKEN_TYPE_FIELD_NUMBER: _ClassVar[int]
    token_type: TokenType
    def __init__(self, token_type: _Optional[_Union[TokenType, _Mapping]] = ...) -> None: ...

class ListTokenTypesRequest(_message.Message):
    __slots__ = ("include_retired",)
    INCLUDE_RETIRED_FIELD_NUMBER: _ClassVar[int]
    include_retired: bool
    def __init__(self, include_retired: _Optional[bool] = ...) -> None: ...

class ListTokenTypesResponse(_message.Message):
    __slots__ = ("token_types",)
    TOKEN_TYPES_FIELD_NUMBER: _ClassVar[int]
    token_types: _containers.RepeatedCompositeFieldContainer[TokenType]
    def __init__(self, token_types: _Optional[_Iterable[_Union[TokenType, _Mapping]]] = ...) -> None: ...

class RetireTokenTypeRequest(_message.Message):
    __slots__ = ("id",)
    ID_FIELD_NUMBER: _ClassVar[int]
    id: str
    def __init__(self, id: _Optional[str] = ...) -> None: ...

class RetireTokenTypeResponse(_message.Message):
    __slots__ = ("token_type",)
    TOKEN_TYPE_FIELD_NUMBER: _ClassVar[int]
    token_type: TokenType
    def __init__(self, token_type: _Optional[_Union[TokenType, _Mapping]] = ...) -> None: ...

class MintSupplyRequest(_message.Message):
    __slots__ = ("token_type_id", "amount")
    TOKEN_TYPE_ID_FIELD_NUMBER: _ClassVar[int]
    AMOUNT_FIELD_NUMBER: _ClassVar[int]
    token_type_id: str
    amount: int
    def __init__(self, token_type_id: _Optional[str] = ..., amount: _Optional[int] = ...) -> None: ...

class MintSupplyResponse(_message.Message):
    __slots__ = ("token_type",)
    TOKEN_TYPE_FIELD_NUMBER: _ClassVar[int]
    token_type: TokenType
    def __init__(self, token_type: _Optional[_Union[TokenType, _Mapping]] = ...) -> None: ...

class GrantRule(_message.Message):
    __slots__ = ("id", "condition", "operands", "amount_limit")
    ID_FIELD_NUMBER: _ClassVar[int]
    CONDITION_FIELD_NUMBER: _ClassVar[int]
    OPERANDS_FIELD_NUMBER: _ClassVar[int]
    AMOUNT_LIMIT_FIELD_NUMBER: _ClassVar[int]
    id: str
    condition: RuleCondition
    operands: _containers.RepeatedScalarFieldContainer[str]
    amount_limit: int
    def __init__(self, id: _Optional[str] = ..., condition: _Optional[_Union[RuleCondition, str]] = ..., operands: _Optional[_Iterable[str]] = ..., amount_limit: _Optional[int] = ...) -> None: ...

class Grant(_message.Message):
    __slots__ = ("id", "token_type_id", "grant_source_id", "authorizer", "amount_minor", "allowed_catalog_scopes", "denied_catalog_scopes", "expires_at", "issued_at", "status", "idempotency_key", "required_evidence", "recurrence_seconds", "next_issue_at", "cancelled_at", "holder_id", "rules")
    ID_FIELD_NUMBER: _ClassVar[int]
    TOKEN_TYPE_ID_FIELD_NUMBER: _ClassVar[int]
    GRANT_SOURCE_ID_FIELD_NUMBER: _ClassVar[int]
    AUTHORIZER_FIELD_NUMBER: _ClassVar[int]
    AMOUNT_MINOR_FIELD_NUMBER: _ClassVar[int]
    ALLOWED_CATALOG_SCOPES_FIELD_NUMBER: _ClassVar[int]
    DENIED_CATALOG_SCOPES_FIELD_NUMBER: _ClassVar[int]
    EXPIRES_AT_FIELD_NUMBER: _ClassVar[int]
    ISSUED_AT_FIELD_NUMBER: _ClassVar[int]
    STATUS_FIELD_NUMBER: _ClassVar[int]
    IDEMPOTENCY_KEY_FIELD_NUMBER: _ClassVar[int]
    REQUIRED_EVIDENCE_FIELD_NUMBER: _ClassVar[int]
    RECURRENCE_SECONDS_FIELD_NUMBER: _ClassVar[int]
    NEXT_ISSUE_AT_FIELD_NUMBER: _ClassVar[int]
    CANCELLED_AT_FIELD_NUMBER: _ClassVar[int]
    HOLDER_ID_FIELD_NUMBER: _ClassVar[int]
    RULES_FIELD_NUMBER: _ClassVar[int]
    id: str
    token_type_id: str
    grant_source_id: str
    authorizer: str
    amount_minor: int
    allowed_catalog_scopes: _containers.RepeatedScalarFieldContainer[str]
    denied_catalog_scopes: _containers.RepeatedScalarFieldContainer[str]
    expires_at: _timestamp_pb2.Timestamp
    issued_at: _timestamp_pb2.Timestamp
    status: GrantStatus
    idempotency_key: str
    required_evidence: _containers.RepeatedScalarFieldContainer[str]
    recurrence_seconds: int
    next_issue_at: _timestamp_pb2.Timestamp
    cancelled_at: _timestamp_pb2.Timestamp
    holder_id: str
    rules: _containers.RepeatedCompositeFieldContainer[GrantRule]
    def __init__(self, id: _Optional[str] = ..., token_type_id: _Optional[str] = ..., grant_source_id: _Optional[str] = ..., authorizer: _Optional[str] = ..., amount_minor: _Optional[int] = ..., allowed_catalog_scopes: _Optional[_Iterable[str]] = ..., denied_catalog_scopes: _Optional[_Iterable[str]] = ..., expires_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., issued_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., status: _Optional[_Union[GrantStatus, str]] = ..., idempotency_key: _Optional[str] = ..., required_evidence: _Optional[_Iterable[str]] = ..., recurrence_seconds: _Optional[int] = ..., next_issue_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., cancelled_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., holder_id: _Optional[str] = ..., rules: _Optional[_Iterable[_Union[GrantRule, _Mapping]]] = ...) -> None: ...

class CreateGrantRequest(_message.Message):
    __slots__ = ("token_type_id", "grant_source_id", "authorizer", "holder_id", "amount_minor", "allowed_catalog_scopes", "denied_catalog_scopes", "expires_at", "idempotency_key", "required_evidence", "rules")
    TOKEN_TYPE_ID_FIELD_NUMBER: _ClassVar[int]
    GRANT_SOURCE_ID_FIELD_NUMBER: _ClassVar[int]
    AUTHORIZER_FIELD_NUMBER: _ClassVar[int]
    HOLDER_ID_FIELD_NUMBER: _ClassVar[int]
    AMOUNT_MINOR_FIELD_NUMBER: _ClassVar[int]
    ALLOWED_CATALOG_SCOPES_FIELD_NUMBER: _ClassVar[int]
    DENIED_CATALOG_SCOPES_FIELD_NUMBER: _ClassVar[int]
    EXPIRES_AT_FIELD_NUMBER: _ClassVar[int]
    IDEMPOTENCY_KEY_FIELD_NUMBER: _ClassVar[int]
    REQUIRED_EVIDENCE_FIELD_NUMBER: _ClassVar[int]
    RULES_FIELD_NUMBER: _ClassVar[int]
    token_type_id: str
    grant_source_id: str
    authorizer: str
    holder_id: str
    amount_minor: int
    allowed_catalog_scopes: _containers.RepeatedScalarFieldContainer[str]
    denied_catalog_scopes: _containers.RepeatedScalarFieldContainer[str]
    expires_at: _timestamp_pb2.Timestamp
    idempotency_key: str
    required_evidence: _containers.RepeatedScalarFieldContainer[str]
    rules: _containers.RepeatedCompositeFieldContainer[GrantRule]
    def __init__(self, token_type_id: _Optional[str] = ..., grant_source_id: _Optional[str] = ..., authorizer: _Optional[str] = ..., holder_id: _Optional[str] = ..., amount_minor: _Optional[int] = ..., allowed_catalog_scopes: _Optional[_Iterable[str]] = ..., denied_catalog_scopes: _Optional[_Iterable[str]] = ..., expires_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., idempotency_key: _Optional[str] = ..., required_evidence: _Optional[_Iterable[str]] = ..., rules: _Optional[_Iterable[_Union[GrantRule, _Mapping]]] = ...) -> None: ...

class CreateGrantResponse(_message.Message):
    __slots__ = ("grant",)
    GRANT_FIELD_NUMBER: _ClassVar[int]
    grant: Grant
    def __init__(self, grant: _Optional[_Union[Grant, _Mapping]] = ...) -> None: ...

class GetGrantRequest(_message.Message):
    __slots__ = ("id",)
    ID_FIELD_NUMBER: _ClassVar[int]
    id: str
    def __init__(self, id: _Optional[str] = ...) -> None: ...

class GetGrantResponse(_message.Message):
    __slots__ = ("grant",)
    GRANT_FIELD_NUMBER: _ClassVar[int]
    grant: Grant
    def __init__(self, grant: _Optional[_Union[Grant, _Mapping]] = ...) -> None: ...

class ListGrantsRequest(_message.Message):
    __slots__ = ("holder_id", "token_type_id", "include_inactive")
    HOLDER_ID_FIELD_NUMBER: _ClassVar[int]
    TOKEN_TYPE_ID_FIELD_NUMBER: _ClassVar[int]
    INCLUDE_INACTIVE_FIELD_NUMBER: _ClassVar[int]
    holder_id: str
    token_type_id: str
    include_inactive: bool
    def __init__(self, holder_id: _Optional[str] = ..., token_type_id: _Optional[str] = ..., include_inactive: _Optional[bool] = ...) -> None: ...

class ListGrantsResponse(_message.Message):
    __slots__ = ("grants",)
    GRANTS_FIELD_NUMBER: _ClassVar[int]
    grants: _containers.RepeatedCompositeFieldContainer[Grant]
    def __init__(self, grants: _Optional[_Iterable[_Union[Grant, _Mapping]]] = ...) -> None: ...

class RevokeGrantRequest(_message.Message):
    __slots__ = ("id", "reason", "idempotency_key")
    ID_FIELD_NUMBER: _ClassVar[int]
    REASON_FIELD_NUMBER: _ClassVar[int]
    IDEMPOTENCY_KEY_FIELD_NUMBER: _ClassVar[int]
    id: str
    reason: str
    idempotency_key: str
    def __init__(self, id: _Optional[str] = ..., reason: _Optional[str] = ..., idempotency_key: _Optional[str] = ...) -> None: ...

class RevokeGrantResponse(_message.Message):
    __slots__ = ("grant",)
    GRANT_FIELD_NUMBER: _ClassVar[int]
    grant: Grant
    def __init__(self, grant: _Optional[_Union[Grant, _Mapping]] = ...) -> None: ...

class Availability(_message.Message):
    __slots__ = ("available_from", "available_until", "remaining_quantity")
    AVAILABLE_FROM_FIELD_NUMBER: _ClassVar[int]
    AVAILABLE_UNTIL_FIELD_NUMBER: _ClassVar[int]
    REMAINING_QUANTITY_FIELD_NUMBER: _ClassVar[int]
    available_from: _timestamp_pb2.Timestamp
    available_until: _timestamp_pb2.Timestamp
    remaining_quantity: int
    def __init__(self, available_from: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., available_until: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., remaining_quantity: _Optional[int] = ...) -> None: ...

class CatalogEntry(_message.Message):
    __slots__ = ("id", "token_type_id", "title", "description", "cost_amount", "availability", "approval_posture", "retired", "created_at", "updated_at", "retired_at")
    ID_FIELD_NUMBER: _ClassVar[int]
    TOKEN_TYPE_ID_FIELD_NUMBER: _ClassVar[int]
    TITLE_FIELD_NUMBER: _ClassVar[int]
    DESCRIPTION_FIELD_NUMBER: _ClassVar[int]
    COST_AMOUNT_FIELD_NUMBER: _ClassVar[int]
    AVAILABILITY_FIELD_NUMBER: _ClassVar[int]
    APPROVAL_POSTURE_FIELD_NUMBER: _ClassVar[int]
    RETIRED_FIELD_NUMBER: _ClassVar[int]
    CREATED_AT_FIELD_NUMBER: _ClassVar[int]
    UPDATED_AT_FIELD_NUMBER: _ClassVar[int]
    RETIRED_AT_FIELD_NUMBER: _ClassVar[int]
    id: str
    token_type_id: str
    title: str
    description: str
    cost_amount: int
    availability: Availability
    approval_posture: ApprovalPosture
    retired: bool
    created_at: _timestamp_pb2.Timestamp
    updated_at: _timestamp_pb2.Timestamp
    retired_at: _timestamp_pb2.Timestamp
    def __init__(self, id: _Optional[str] = ..., token_type_id: _Optional[str] = ..., title: _Optional[str] = ..., description: _Optional[str] = ..., cost_amount: _Optional[int] = ..., availability: _Optional[_Union[Availability, _Mapping]] = ..., approval_posture: _Optional[_Union[ApprovalPosture, str]] = ..., retired: _Optional[bool] = ..., created_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., updated_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., retired_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ...) -> None: ...

class Holder(_message.Message):
    __slots__ = ("id", "display_name", "authenticator_subject", "created_at")
    ID_FIELD_NUMBER: _ClassVar[int]
    DISPLAY_NAME_FIELD_NUMBER: _ClassVar[int]
    AUTHENTICATOR_SUBJECT_FIELD_NUMBER: _ClassVar[int]
    CREATED_AT_FIELD_NUMBER: _ClassVar[int]
    id: str
    display_name: str
    authenticator_subject: str
    created_at: _timestamp_pb2.Timestamp
    def __init__(self, id: _Optional[str] = ..., display_name: _Optional[str] = ..., authenticator_subject: _Optional[str] = ..., created_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ...) -> None: ...

class Event(_message.Message):
    __slots__ = ("id", "token_type_id", "amount", "kind", "reason", "created_at", "actor_identity", "cause_reference", "actor_kind", "actor_verification_status", "actor_run_id")
    ID_FIELD_NUMBER: _ClassVar[int]
    TOKEN_TYPE_ID_FIELD_NUMBER: _ClassVar[int]
    AMOUNT_FIELD_NUMBER: _ClassVar[int]
    KIND_FIELD_NUMBER: _ClassVar[int]
    REASON_FIELD_NUMBER: _ClassVar[int]
    CREATED_AT_FIELD_NUMBER: _ClassVar[int]
    ACTOR_IDENTITY_FIELD_NUMBER: _ClassVar[int]
    CAUSE_REFERENCE_FIELD_NUMBER: _ClassVar[int]
    ACTOR_KIND_FIELD_NUMBER: _ClassVar[int]
    ACTOR_VERIFICATION_STATUS_FIELD_NUMBER: _ClassVar[int]
    ACTOR_RUN_ID_FIELD_NUMBER: _ClassVar[int]
    id: str
    token_type_id: str
    amount: int
    kind: EventKind
    reason: str
    created_at: _timestamp_pb2.Timestamp
    actor_identity: str
    cause_reference: str
    actor_kind: ActorKind
    actor_verification_status: VerificationStatus
    actor_run_id: str
    def __init__(self, id: _Optional[str] = ..., token_type_id: _Optional[str] = ..., amount: _Optional[int] = ..., kind: _Optional[_Union[EventKind, str]] = ..., reason: _Optional[str] = ..., created_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., actor_identity: _Optional[str] = ..., cause_reference: _Optional[str] = ..., actor_kind: _Optional[_Union[ActorKind, str]] = ..., actor_verification_status: _Optional[_Union[VerificationStatus, str]] = ..., actor_run_id: _Optional[str] = ...) -> None: ...

class Balance(_message.Message):
    __slots__ = ("token_type_id", "amount")
    TOKEN_TYPE_ID_FIELD_NUMBER: _ClassVar[int]
    AMOUNT_FIELD_NUMBER: _ClassVar[int]
    token_type_id: str
    amount: int
    def __init__(self, token_type_id: _Optional[str] = ..., amount: _Optional[int] = ...) -> None: ...

class Redemption(_message.Message):
    __slots__ = ("id", "catalog_entry_id", "holder_id", "token_type_id", "grant_id", "amount", "idempotency_key", "state", "decider_subject", "decision_reason", "requested_at", "decided_at", "settled_at")
    ID_FIELD_NUMBER: _ClassVar[int]
    CATALOG_ENTRY_ID_FIELD_NUMBER: _ClassVar[int]
    HOLDER_ID_FIELD_NUMBER: _ClassVar[int]
    TOKEN_TYPE_ID_FIELD_NUMBER: _ClassVar[int]
    GRANT_ID_FIELD_NUMBER: _ClassVar[int]
    AMOUNT_FIELD_NUMBER: _ClassVar[int]
    IDEMPOTENCY_KEY_FIELD_NUMBER: _ClassVar[int]
    STATE_FIELD_NUMBER: _ClassVar[int]
    DECIDER_SUBJECT_FIELD_NUMBER: _ClassVar[int]
    DECISION_REASON_FIELD_NUMBER: _ClassVar[int]
    REQUESTED_AT_FIELD_NUMBER: _ClassVar[int]
    DECIDED_AT_FIELD_NUMBER: _ClassVar[int]
    SETTLED_AT_FIELD_NUMBER: _ClassVar[int]
    id: str
    catalog_entry_id: str
    holder_id: str
    token_type_id: str
    grant_id: str
    amount: int
    idempotency_key: str
    state: RedemptionState
    decider_subject: str
    decision_reason: str
    requested_at: _timestamp_pb2.Timestamp
    decided_at: _timestamp_pb2.Timestamp
    settled_at: _timestamp_pb2.Timestamp
    def __init__(self, id: _Optional[str] = ..., catalog_entry_id: _Optional[str] = ..., holder_id: _Optional[str] = ..., token_type_id: _Optional[str] = ..., grant_id: _Optional[str] = ..., amount: _Optional[int] = ..., idempotency_key: _Optional[str] = ..., state: _Optional[_Union[RedemptionState, str]] = ..., decider_subject: _Optional[str] = ..., decision_reason: _Optional[str] = ..., requested_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., decided_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., settled_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ...) -> None: ...

class CreateCatalogEntryRequest(_message.Message):
    __slots__ = ("entry", "idempotency_key")
    ENTRY_FIELD_NUMBER: _ClassVar[int]
    IDEMPOTENCY_KEY_FIELD_NUMBER: _ClassVar[int]
    entry: CatalogEntry
    idempotency_key: str
    def __init__(self, entry: _Optional[_Union[CatalogEntry, _Mapping]] = ..., idempotency_key: _Optional[str] = ...) -> None: ...

class CreateCatalogEntryResponse(_message.Message):
    __slots__ = ("entry",)
    ENTRY_FIELD_NUMBER: _ClassVar[int]
    entry: CatalogEntry
    def __init__(self, entry: _Optional[_Union[CatalogEntry, _Mapping]] = ...) -> None: ...

class UpdateCatalogEntryRequest(_message.Message):
    __slots__ = ("entry", "idempotency_key")
    ENTRY_FIELD_NUMBER: _ClassVar[int]
    IDEMPOTENCY_KEY_FIELD_NUMBER: _ClassVar[int]
    entry: CatalogEntry
    idempotency_key: str
    def __init__(self, entry: _Optional[_Union[CatalogEntry, _Mapping]] = ..., idempotency_key: _Optional[str] = ...) -> None: ...

class UpdateCatalogEntryResponse(_message.Message):
    __slots__ = ("entry",)
    ENTRY_FIELD_NUMBER: _ClassVar[int]
    entry: CatalogEntry
    def __init__(self, entry: _Optional[_Union[CatalogEntry, _Mapping]] = ...) -> None: ...

class GetCatalogEntryRequest(_message.Message):
    __slots__ = ("id",)
    ID_FIELD_NUMBER: _ClassVar[int]
    id: str
    def __init__(self, id: _Optional[str] = ...) -> None: ...

class GetCatalogEntryResponse(_message.Message):
    __slots__ = ("entry",)
    ENTRY_FIELD_NUMBER: _ClassVar[int]
    entry: CatalogEntry
    def __init__(self, entry: _Optional[_Union[CatalogEntry, _Mapping]] = ...) -> None: ...

class ListCatalogEntriesRequest(_message.Message):
    __slots__ = ("include_retired",)
    INCLUDE_RETIRED_FIELD_NUMBER: _ClassVar[int]
    include_retired: bool
    def __init__(self, include_retired: _Optional[bool] = ...) -> None: ...

class ListCatalogEntriesResponse(_message.Message):
    __slots__ = ("entries",)
    ENTRIES_FIELD_NUMBER: _ClassVar[int]
    entries: _containers.RepeatedCompositeFieldContainer[CatalogEntry]
    def __init__(self, entries: _Optional[_Iterable[_Union[CatalogEntry, _Mapping]]] = ...) -> None: ...

class RetireCatalogEntryRequest(_message.Message):
    __slots__ = ("id", "idempotency_key")
    ID_FIELD_NUMBER: _ClassVar[int]
    IDEMPOTENCY_KEY_FIELD_NUMBER: _ClassVar[int]
    id: str
    idempotency_key: str
    def __init__(self, id: _Optional[str] = ..., idempotency_key: _Optional[str] = ...) -> None: ...

class RetireCatalogEntryResponse(_message.Message):
    __slots__ = ("entry",)
    ENTRY_FIELD_NUMBER: _ClassVar[int]
    entry: CatalogEntry
    def __init__(self, entry: _Optional[_Union[CatalogEntry, _Mapping]] = ...) -> None: ...

class UpdateGrantRuleRequest(_message.Message):
    __slots__ = ("grant_id", "rule", "idempotency_key")
    GRANT_ID_FIELD_NUMBER: _ClassVar[int]
    RULE_FIELD_NUMBER: _ClassVar[int]
    IDEMPOTENCY_KEY_FIELD_NUMBER: _ClassVar[int]
    grant_id: str
    rule: GrantRule
    idempotency_key: str
    def __init__(self, grant_id: _Optional[str] = ..., rule: _Optional[_Union[GrantRule, _Mapping]] = ..., idempotency_key: _Optional[str] = ...) -> None: ...

class UpdateGrantRuleResponse(_message.Message):
    __slots__ = ("grant",)
    GRANT_FIELD_NUMBER: _ClassVar[int]
    grant: Grant
    def __init__(self, grant: _Optional[_Union[Grant, _Mapping]] = ...) -> None: ...

class ViewEconomyRequest(_message.Message):
    __slots__ = ()
    def __init__(self) -> None: ...

class ViewEconomyResponse(_message.Message):
    __slots__ = ("holder", "events", "balances", "redemptions")
    HOLDER_FIELD_NUMBER: _ClassVar[int]
    EVENTS_FIELD_NUMBER: _ClassVar[int]
    BALANCES_FIELD_NUMBER: _ClassVar[int]
    REDEMPTIONS_FIELD_NUMBER: _ClassVar[int]
    holder: Holder
    events: _containers.RepeatedCompositeFieldContainer[Event]
    balances: _containers.RepeatedCompositeFieldContainer[Balance]
    redemptions: _containers.RepeatedCompositeFieldContainer[Redemption]
    def __init__(self, holder: _Optional[_Union[Holder, _Mapping]] = ..., events: _Optional[_Iterable[_Union[Event, _Mapping]]] = ..., balances: _Optional[_Iterable[_Union[Balance, _Mapping]]] = ..., redemptions: _Optional[_Iterable[_Union[Redemption, _Mapping]]] = ...) -> None: ...

class CreateHolderRequest(_message.Message):
    __slots__ = ("display_name", "authenticator_subject", "idempotency_key")
    DISPLAY_NAME_FIELD_NUMBER: _ClassVar[int]
    AUTHENTICATOR_SUBJECT_FIELD_NUMBER: _ClassVar[int]
    IDEMPOTENCY_KEY_FIELD_NUMBER: _ClassVar[int]
    display_name: str
    authenticator_subject: str
    idempotency_key: str
    def __init__(self, display_name: _Optional[str] = ..., authenticator_subject: _Optional[str] = ..., idempotency_key: _Optional[str] = ...) -> None: ...

class CreateHolderResponse(_message.Message):
    __slots__ = ("holder",)
    HOLDER_FIELD_NUMBER: _ClassVar[int]
    holder: Holder
    def __init__(self, holder: _Optional[_Union[Holder, _Mapping]] = ...) -> None: ...

class GetHolderRequest(_message.Message):
    __slots__ = ("id",)
    ID_FIELD_NUMBER: _ClassVar[int]
    id: str
    def __init__(self, id: _Optional[str] = ...) -> None: ...

class GetHolderResponse(_message.Message):
    __slots__ = ("holder",)
    HOLDER_FIELD_NUMBER: _ClassVar[int]
    holder: Holder
    def __init__(self, holder: _Optional[_Union[Holder, _Mapping]] = ...) -> None: ...

class ListHoldersRequest(_message.Message):
    __slots__ = ()
    def __init__(self) -> None: ...

class ListHoldersResponse(_message.Message):
    __slots__ = ("holders",)
    HOLDERS_FIELD_NUMBER: _ClassVar[int]
    holders: _containers.RepeatedCompositeFieldContainer[Holder]
    def __init__(self, holders: _Optional[_Iterable[_Union[Holder, _Mapping]]] = ...) -> None: ...

class ListJournalEventsRequest(_message.Message):
    __slots__ = ("holder_id", "token_type_id")
    HOLDER_ID_FIELD_NUMBER: _ClassVar[int]
    TOKEN_TYPE_ID_FIELD_NUMBER: _ClassVar[int]
    holder_id: str
    token_type_id: str
    def __init__(self, holder_id: _Optional[str] = ..., token_type_id: _Optional[str] = ...) -> None: ...

class ListJournalEventsResponse(_message.Message):
    __slots__ = ("events",)
    EVENTS_FIELD_NUMBER: _ClassVar[int]
    events: _containers.RepeatedCompositeFieldContainer[Event]
    def __init__(self, events: _Optional[_Iterable[_Union[Event, _Mapping]]] = ...) -> None: ...

class ShowBalanceRequest(_message.Message):
    __slots__ = ("holder_id", "token_type_id")
    HOLDER_ID_FIELD_NUMBER: _ClassVar[int]
    TOKEN_TYPE_ID_FIELD_NUMBER: _ClassVar[int]
    holder_id: str
    token_type_id: str
    def __init__(self, holder_id: _Optional[str] = ..., token_type_id: _Optional[str] = ...) -> None: ...

class ShowBalanceResponse(_message.Message):
    __slots__ = ("balance",)
    BALANCE_FIELD_NUMBER: _ClassVar[int]
    balance: Balance
    def __init__(self, balance: _Optional[_Union[Balance, _Mapping]] = ...) -> None: ...

class ExportJournalRequest(_message.Message):
    __slots__ = ("holder_id", "token_type_id")
    HOLDER_ID_FIELD_NUMBER: _ClassVar[int]
    TOKEN_TYPE_ID_FIELD_NUMBER: _ClassVar[int]
    holder_id: str
    token_type_id: str
    def __init__(self, holder_id: _Optional[str] = ..., token_type_id: _Optional[str] = ...) -> None: ...

class ExportJournalResponse(_message.Message):
    __slots__ = ("events",)
    EVENTS_FIELD_NUMBER: _ClassVar[int]
    events: _containers.RepeatedCompositeFieldContainer[Event]
    def __init__(self, events: _Optional[_Iterable[_Union[Event, _Mapping]]] = ...) -> None: ...

class BrowseCatalogRequest(_message.Message):
    __slots__ = ()
    def __init__(self) -> None: ...

class BrowseCatalogResponse(_message.Message):
    __slots__ = ("entries",)
    ENTRIES_FIELD_NUMBER: _ClassVar[int]
    entries: _containers.RepeatedCompositeFieldContainer[CatalogEntry]
    def __init__(self, entries: _Optional[_Iterable[_Union[CatalogEntry, _Mapping]]] = ...) -> None: ...

class RequestRedemptionRequest(_message.Message):
    __slots__ = ("redemption", "idempotency_key", "evidence")
    REDEMPTION_FIELD_NUMBER: _ClassVar[int]
    IDEMPOTENCY_KEY_FIELD_NUMBER: _ClassVar[int]
    EVIDENCE_FIELD_NUMBER: _ClassVar[int]
    redemption: Redemption
    idempotency_key: str
    evidence: _containers.RepeatedScalarFieldContainer[str]
    def __init__(self, redemption: _Optional[_Union[Redemption, _Mapping]] = ..., idempotency_key: _Optional[str] = ..., evidence: _Optional[_Iterable[str]] = ...) -> None: ...

class RequestRedemptionResponse(_message.Message):
    __slots__ = ("redemption",)
    REDEMPTION_FIELD_NUMBER: _ClassVar[int]
    redemption: Redemption
    def __init__(self, redemption: _Optional[_Union[Redemption, _Mapping]] = ...) -> None: ...

class ListPendingRedemptionsRequest(_message.Message):
    __slots__ = ()
    def __init__(self) -> None: ...

class ListPendingRedemptionsResponse(_message.Message):
    __slots__ = ("redemptions",)
    REDEMPTIONS_FIELD_NUMBER: _ClassVar[int]
    redemptions: _containers.RepeatedCompositeFieldContainer[Redemption]
    def __init__(self, redemptions: _Optional[_Iterable[_Union[Redemption, _Mapping]]] = ...) -> None: ...

class ApproveRedemptionRequest(_message.Message):
    __slots__ = ("redemption_id", "reason", "idempotency_key")
    REDEMPTION_ID_FIELD_NUMBER: _ClassVar[int]
    REASON_FIELD_NUMBER: _ClassVar[int]
    IDEMPOTENCY_KEY_FIELD_NUMBER: _ClassVar[int]
    redemption_id: str
    reason: str
    idempotency_key: str
    def __init__(self, redemption_id: _Optional[str] = ..., reason: _Optional[str] = ..., idempotency_key: _Optional[str] = ...) -> None: ...

class ApproveRedemptionResponse(_message.Message):
    __slots__ = ("redemption",)
    REDEMPTION_FIELD_NUMBER: _ClassVar[int]
    redemption: Redemption
    def __init__(self, redemption: _Optional[_Union[Redemption, _Mapping]] = ...) -> None: ...

class DenyRedemptionRequest(_message.Message):
    __slots__ = ("redemption_id", "reason", "idempotency_key")
    REDEMPTION_ID_FIELD_NUMBER: _ClassVar[int]
    REASON_FIELD_NUMBER: _ClassVar[int]
    IDEMPOTENCY_KEY_FIELD_NUMBER: _ClassVar[int]
    redemption_id: str
    reason: str
    idempotency_key: str
    def __init__(self, redemption_id: _Optional[str] = ..., reason: _Optional[str] = ..., idempotency_key: _Optional[str] = ...) -> None: ...

class DenyRedemptionResponse(_message.Message):
    __slots__ = ("redemption",)
    REDEMPTION_FIELD_NUMBER: _ClassVar[int]
    redemption: Redemption
    def __init__(self, redemption: _Optional[_Union[Redemption, _Mapping]] = ...) -> None: ...

class ReverseEventRequest(_message.Message):
    __slots__ = ("original_event_id", "reason", "idempotency_key")
    ORIGINAL_EVENT_ID_FIELD_NUMBER: _ClassVar[int]
    REASON_FIELD_NUMBER: _ClassVar[int]
    IDEMPOTENCY_KEY_FIELD_NUMBER: _ClassVar[int]
    original_event_id: str
    reason: str
    idempotency_key: str
    def __init__(self, original_event_id: _Optional[str] = ..., reason: _Optional[str] = ..., idempotency_key: _Optional[str] = ...) -> None: ...

class ReverseEventResponse(_message.Message):
    __slots__ = ("reversal",)
    REVERSAL_FIELD_NUMBER: _ClassVar[int]
    reversal: Event
    def __init__(self, reversal: _Optional[_Union[Event, _Mapping]] = ...) -> None: ...

class SubmitRequestRequest(_message.Message):
    __slots__ = ("reason", "idempotency_key")
    REASON_FIELD_NUMBER: _ClassVar[int]
    IDEMPOTENCY_KEY_FIELD_NUMBER: _ClassVar[int]
    reason: str
    idempotency_key: str
    def __init__(self, reason: _Optional[str] = ..., idempotency_key: _Optional[str] = ...) -> None: ...

class SubmitRequestResponse(_message.Message):
    __slots__ = ("request_id",)
    REQUEST_ID_FIELD_NUMBER: _ClassVar[int]
    request_id: str
    def __init__(self, request_id: _Optional[str] = ...) -> None: ...
