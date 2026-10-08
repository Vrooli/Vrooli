import datetime

from google.protobuf import timestamp_pb2 as _timestamp_pb2
from google.protobuf.internal import containers as _containers
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Iterable as _Iterable, Mapping as _Mapping
from typing import ClassVar as _ClassVar, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class IssueReaderTokenRequest(_message.Message):
    __slots__ = ("label", "scope")
    LABEL_FIELD_NUMBER: _ClassVar[int]
    SCOPE_FIELD_NUMBER: _ClassVar[int]
    label: str
    scope: str
    def __init__(self, label: _Optional[str] = ..., scope: _Optional[str] = ...) -> None: ...

class IssuedReaderToken(_message.Message):
    __slots__ = ("id", "token", "prefix", "scope", "created_at")
    ID_FIELD_NUMBER: _ClassVar[int]
    TOKEN_FIELD_NUMBER: _ClassVar[int]
    PREFIX_FIELD_NUMBER: _ClassVar[int]
    SCOPE_FIELD_NUMBER: _ClassVar[int]
    CREATED_AT_FIELD_NUMBER: _ClassVar[int]
    id: str
    token: str
    prefix: str
    scope: str
    created_at: _timestamp_pb2.Timestamp
    def __init__(self, id: _Optional[str] = ..., token: _Optional[str] = ..., prefix: _Optional[str] = ..., scope: _Optional[str] = ..., created_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ...) -> None: ...

class ListReaderTokensRequest(_message.Message):
    __slots__ = ()
    def __init__(self) -> None: ...

class ReaderToken(_message.Message):
    __slots__ = ("id", "label", "scope", "prefix", "source", "created_by", "created_at", "last_used_at", "revoked_at")
    ID_FIELD_NUMBER: _ClassVar[int]
    LABEL_FIELD_NUMBER: _ClassVar[int]
    SCOPE_FIELD_NUMBER: _ClassVar[int]
    PREFIX_FIELD_NUMBER: _ClassVar[int]
    SOURCE_FIELD_NUMBER: _ClassVar[int]
    CREATED_BY_FIELD_NUMBER: _ClassVar[int]
    CREATED_AT_FIELD_NUMBER: _ClassVar[int]
    LAST_USED_AT_FIELD_NUMBER: _ClassVar[int]
    REVOKED_AT_FIELD_NUMBER: _ClassVar[int]
    id: str
    label: str
    scope: str
    prefix: str
    source: str
    created_by: str
    created_at: _timestamp_pb2.Timestamp
    last_used_at: _timestamp_pb2.Timestamp
    revoked_at: _timestamp_pb2.Timestamp
    def __init__(self, id: _Optional[str] = ..., label: _Optional[str] = ..., scope: _Optional[str] = ..., prefix: _Optional[str] = ..., source: _Optional[str] = ..., created_by: _Optional[str] = ..., created_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., last_used_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., revoked_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ...) -> None: ...

class ListReaderTokensResponse(_message.Message):
    __slots__ = ("tokens",)
    TOKENS_FIELD_NUMBER: _ClassVar[int]
    tokens: _containers.RepeatedCompositeFieldContainer[ReaderToken]
    def __init__(self, tokens: _Optional[_Iterable[_Union[ReaderToken, _Mapping]]] = ...) -> None: ...

class RevokeReaderTokenRequest(_message.Message):
    __slots__ = ("id",)
    ID_FIELD_NUMBER: _ClassVar[int]
    id: str
    def __init__(self, id: _Optional[str] = ...) -> None: ...

class RevokeReaderTokenResponse(_message.Message):
    __slots__ = ("revoked",)
    REVOKED_FIELD_NUMBER: _ClassVar[int]
    revoked: bool
    def __init__(self, revoked: _Optional[bool] = ...) -> None: ...
