import datetime

from google.protobuf import timestamp_pb2 as _timestamp_pb2
from google.protobuf.internal import containers as _containers
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Iterable as _Iterable, Mapping as _Mapping
from typing import ClassVar as _ClassVar, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class EarningSubmission(_message.Message):
    __slots__ = ("id", "holder_id", "token_type_id", "amount_minor", "reason", "dedup_key", "adapter_identity", "actor_identity", "grant_id", "submitted_at", "replayed", "payload_summary")
    ID_FIELD_NUMBER: _ClassVar[int]
    HOLDER_ID_FIELD_NUMBER: _ClassVar[int]
    TOKEN_TYPE_ID_FIELD_NUMBER: _ClassVar[int]
    AMOUNT_MINOR_FIELD_NUMBER: _ClassVar[int]
    REASON_FIELD_NUMBER: _ClassVar[int]
    DEDUP_KEY_FIELD_NUMBER: _ClassVar[int]
    ADAPTER_IDENTITY_FIELD_NUMBER: _ClassVar[int]
    ACTOR_IDENTITY_FIELD_NUMBER: _ClassVar[int]
    GRANT_ID_FIELD_NUMBER: _ClassVar[int]
    SUBMITTED_AT_FIELD_NUMBER: _ClassVar[int]
    REPLAYED_FIELD_NUMBER: _ClassVar[int]
    PAYLOAD_SUMMARY_FIELD_NUMBER: _ClassVar[int]
    id: str
    holder_id: str
    token_type_id: str
    amount_minor: int
    reason: str
    dedup_key: str
    adapter_identity: str
    actor_identity: str
    grant_id: str
    submitted_at: _timestamp_pb2.Timestamp
    replayed: bool
    payload_summary: str
    def __init__(self, id: _Optional[str] = ..., holder_id: _Optional[str] = ..., token_type_id: _Optional[str] = ..., amount_minor: _Optional[int] = ..., reason: _Optional[str] = ..., dedup_key: _Optional[str] = ..., adapter_identity: _Optional[str] = ..., actor_identity: _Optional[str] = ..., grant_id: _Optional[str] = ..., submitted_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., replayed: _Optional[bool] = ..., payload_summary: _Optional[str] = ...) -> None: ...

class SubmitEarningRequest(_message.Message):
    __slots__ = ("holder_id", "token_type_id", "amount_minor", "reason", "dedup_key")
    HOLDER_ID_FIELD_NUMBER: _ClassVar[int]
    TOKEN_TYPE_ID_FIELD_NUMBER: _ClassVar[int]
    AMOUNT_MINOR_FIELD_NUMBER: _ClassVar[int]
    REASON_FIELD_NUMBER: _ClassVar[int]
    DEDUP_KEY_FIELD_NUMBER: _ClassVar[int]
    holder_id: str
    token_type_id: str
    amount_minor: int
    reason: str
    dedup_key: str
    def __init__(self, holder_id: _Optional[str] = ..., token_type_id: _Optional[str] = ..., amount_minor: _Optional[int] = ..., reason: _Optional[str] = ..., dedup_key: _Optional[str] = ...) -> None: ...

class SubmitEarningResponse(_message.Message):
    __slots__ = ("submission",)
    SUBMISSION_FIELD_NUMBER: _ClassVar[int]
    submission: EarningSubmission
    def __init__(self, submission: _Optional[_Union[EarningSubmission, _Mapping]] = ...) -> None: ...

class ListEarningsRequest(_message.Message):
    __slots__ = ()
    def __init__(self) -> None: ...

class ListEarningsResponse(_message.Message):
    __slots__ = ("submissions",)
    SUBMISSIONS_FIELD_NUMBER: _ClassVar[int]
    submissions: _containers.RepeatedCompositeFieldContainer[EarningSubmission]
    def __init__(self, submissions: _Optional[_Iterable[_Union[EarningSubmission, _Mapping]]] = ...) -> None: ...
