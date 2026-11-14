import datetime

from google.protobuf import timestamp_pb2 as _timestamp_pb2
from google.protobuf.internal import enum_type_wrapper as _enum_type_wrapper
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Mapping as _Mapping
from typing import ClassVar as _ClassVar, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class ApprovalPosture(int, metaclass=_enum_type_wrapper.EnumTypeWrapper):
    __slots__ = ()
    APPROVAL_POSTURE_UNSPECIFIED: _ClassVar[ApprovalPosture]
    APPROVAL_POSTURE_IMMEDIATE: _ClassVar[ApprovalPosture]
    APPROVAL_POSTURE_REQUIRES_APPROVAL: _ClassVar[ApprovalPosture]
APPROVAL_POSTURE_UNSPECIFIED: ApprovalPosture
APPROVAL_POSTURE_IMMEDIATE: ApprovalPosture
APPROVAL_POSTURE_REQUIRES_APPROVAL: ApprovalPosture

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
