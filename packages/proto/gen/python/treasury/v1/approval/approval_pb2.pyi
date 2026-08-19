import datetime

from google.protobuf import timestamp_pb2 as _timestamp_pb2
from google.protobuf.internal import enum_type_wrapper as _enum_type_wrapper
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Mapping as _Mapping
from typing import ClassVar as _ClassVar, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class ApprovalStatus(int, metaclass=_enum_type_wrapper.EnumTypeWrapper):
    __slots__ = ()
    APPROVAL_STATUS_UNSPECIFIED: _ClassVar[ApprovalStatus]
    APPROVAL_STATUS_QUEUED: _ClassVar[ApprovalStatus]
    APPROVAL_STATUS_APPROVED: _ClassVar[ApprovalStatus]
    APPROVAL_STATUS_DECLINED: _ClassVar[ApprovalStatus]
    APPROVAL_STATUS_EXPIRED: _ClassVar[ApprovalStatus]
APPROVAL_STATUS_UNSPECIFIED: ApprovalStatus
APPROVAL_STATUS_QUEUED: ApprovalStatus
APPROVAL_STATUS_APPROVED: ApprovalStatus
APPROVAL_STATUS_DECLINED: ApprovalStatus
APPROVAL_STATUS_EXPIRED: ApprovalStatus

class ApprovalRequest(_message.Message):
    __slots__ = ("id", "authorization_id", "mandate_id", "requesting_agent", "amount_minor", "currency", "counterparty", "status", "resolver_identity", "created_at", "resolved_at", "expires_at")
    ID_FIELD_NUMBER: _ClassVar[int]
    AUTHORIZATION_ID_FIELD_NUMBER: _ClassVar[int]
    MANDATE_ID_FIELD_NUMBER: _ClassVar[int]
    REQUESTING_AGENT_FIELD_NUMBER: _ClassVar[int]
    AMOUNT_MINOR_FIELD_NUMBER: _ClassVar[int]
    CURRENCY_FIELD_NUMBER: _ClassVar[int]
    COUNTERPARTY_FIELD_NUMBER: _ClassVar[int]
    STATUS_FIELD_NUMBER: _ClassVar[int]
    RESOLVER_IDENTITY_FIELD_NUMBER: _ClassVar[int]
    CREATED_AT_FIELD_NUMBER: _ClassVar[int]
    RESOLVED_AT_FIELD_NUMBER: _ClassVar[int]
    EXPIRES_AT_FIELD_NUMBER: _ClassVar[int]
    id: str
    authorization_id: str
    mandate_id: str
    requesting_agent: str
    amount_minor: int
    currency: str
    counterparty: str
    status: ApprovalStatus
    resolver_identity: str
    created_at: _timestamp_pb2.Timestamp
    resolved_at: _timestamp_pb2.Timestamp
    expires_at: _timestamp_pb2.Timestamp
    def __init__(self, id: _Optional[str] = ..., authorization_id: _Optional[str] = ..., mandate_id: _Optional[str] = ..., requesting_agent: _Optional[str] = ..., amount_minor: _Optional[int] = ..., currency: _Optional[str] = ..., counterparty: _Optional[str] = ..., status: _Optional[_Union[ApprovalStatus, str]] = ..., resolver_identity: _Optional[str] = ..., created_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., resolved_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., expires_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ...) -> None: ...
