import datetime

from google.protobuf import timestamp_pb2 as _timestamp_pb2
from google.protobuf.internal import containers as _containers
from google.protobuf.internal import enum_type_wrapper as _enum_type_wrapper
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Iterable as _Iterable, Mapping as _Mapping
from typing import ClassVar as _ClassVar, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

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
