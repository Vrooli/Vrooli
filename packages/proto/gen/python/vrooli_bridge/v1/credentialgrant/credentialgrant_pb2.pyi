import datetime

from google.protobuf import timestamp_pb2 as _timestamp_pb2
from google.protobuf.internal import containers as _containers
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Iterable as _Iterable, Mapping as _Mapping
from typing import ClassVar as _ClassVar, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class CredentialGrant(_message.Message):
    __slots__ = ("id", "node_id", "logical_id", "field", "retention", "generation", "acked_generation", "granted_at", "revoked_at")
    ID_FIELD_NUMBER: _ClassVar[int]
    NODE_ID_FIELD_NUMBER: _ClassVar[int]
    LOGICAL_ID_FIELD_NUMBER: _ClassVar[int]
    FIELD_FIELD_NUMBER: _ClassVar[int]
    CLASS_FIELD_NUMBER: _ClassVar[int]
    RETENTION_FIELD_NUMBER: _ClassVar[int]
    GENERATION_FIELD_NUMBER: _ClassVar[int]
    ACKED_GENERATION_FIELD_NUMBER: _ClassVar[int]
    GRANTED_AT_FIELD_NUMBER: _ClassVar[int]
    REVOKED_AT_FIELD_NUMBER: _ClassVar[int]
    id: str
    node_id: str
    logical_id: str
    field: str
    retention: str
    generation: int
    acked_generation: int
    granted_at: _timestamp_pb2.Timestamp
    revoked_at: _timestamp_pb2.Timestamp
    def __init__(self, id: _Optional[str] = ..., node_id: _Optional[str] = ..., logical_id: _Optional[str] = ..., field: _Optional[str] = ..., retention: _Optional[str] = ..., generation: _Optional[int] = ..., acked_generation: _Optional[int] = ..., granted_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., revoked_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., **kwargs) -> None: ...

class CreateGrantRequest(_message.Message):
    __slots__ = ("node_id", "logical_id", "field", "retention", "generation")
    NODE_ID_FIELD_NUMBER: _ClassVar[int]
    LOGICAL_ID_FIELD_NUMBER: _ClassVar[int]
    FIELD_FIELD_NUMBER: _ClassVar[int]
    CLASS_FIELD_NUMBER: _ClassVar[int]
    RETENTION_FIELD_NUMBER: _ClassVar[int]
    GENERATION_FIELD_NUMBER: _ClassVar[int]
    node_id: str
    logical_id: str
    field: str
    retention: str
    generation: int
    def __init__(self, node_id: _Optional[str] = ..., logical_id: _Optional[str] = ..., field: _Optional[str] = ..., retention: _Optional[str] = ..., generation: _Optional[int] = ..., **kwargs) -> None: ...

class ListGrantsRequest(_message.Message):
    __slots__ = ("node_id",)
    NODE_ID_FIELD_NUMBER: _ClassVar[int]
    node_id: str
    def __init__(self, node_id: _Optional[str] = ...) -> None: ...

class ListGrantsResponse(_message.Message):
    __slots__ = ("grants",)
    GRANTS_FIELD_NUMBER: _ClassVar[int]
    grants: _containers.RepeatedCompositeFieldContainer[CredentialGrant]
    def __init__(self, grants: _Optional[_Iterable[_Union[CredentialGrant, _Mapping]]] = ...) -> None: ...

class RevokeGrantRequest(_message.Message):
    __slots__ = ("id",)
    ID_FIELD_NUMBER: _ClassVar[int]
    id: str
    def __init__(self, id: _Optional[str] = ...) -> None: ...

class SyncNodeGrantsRequest(_message.Message):
    __slots__ = ("node_id",)
    NODE_ID_FIELD_NUMBER: _ClassVar[int]
    node_id: str
    def __init__(self, node_id: _Optional[str] = ...) -> None: ...

class RotateAddressRequest(_message.Message):
    __slots__ = ("logical_id", "field")
    LOGICAL_ID_FIELD_NUMBER: _ClassVar[int]
    FIELD_FIELD_NUMBER: _ClassVar[int]
    logical_id: str
    field: str
    def __init__(self, logical_id: _Optional[str] = ..., field: _Optional[str] = ...) -> None: ...

class RotationResponse(_message.Message):
    __slots__ = ("logical_id", "field", "generation", "grants")
    LOGICAL_ID_FIELD_NUMBER: _ClassVar[int]
    FIELD_FIELD_NUMBER: _ClassVar[int]
    GENERATION_FIELD_NUMBER: _ClassVar[int]
    GRANTS_FIELD_NUMBER: _ClassVar[int]
    logical_id: str
    field: str
    generation: int
    grants: _containers.RepeatedCompositeFieldContainer[CredentialGrant]
    def __init__(self, logical_id: _Optional[str] = ..., field: _Optional[str] = ..., generation: _Optional[int] = ..., grants: _Optional[_Iterable[_Union[CredentialGrant, _Mapping]]] = ...) -> None: ...
