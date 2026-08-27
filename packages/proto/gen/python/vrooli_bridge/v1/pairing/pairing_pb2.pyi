import datetime

from google.protobuf import timestamp_pb2 as _timestamp_pb2
from google.protobuf.internal import containers as _containers
from google.protobuf.internal import enum_type_wrapper as _enum_type_wrapper
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Iterable as _Iterable, Mapping as _Mapping
from typing import ClassVar as _ClassVar, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class PairingRequestStatus(int, metaclass=_enum_type_wrapper.EnumTypeWrapper):
    __slots__ = ()
    PAIRING_REQUEST_STATUS_UNSPECIFIED: _ClassVar[PairingRequestStatus]
    PAIRING_REQUEST_STATUS_PENDING: _ClassVar[PairingRequestStatus]
    PAIRING_REQUEST_STATUS_APPROVED: _ClassVar[PairingRequestStatus]
    PAIRING_REQUEST_STATUS_REJECTED: _ClassVar[PairingRequestStatus]
PAIRING_REQUEST_STATUS_UNSPECIFIED: PairingRequestStatus
PAIRING_REQUEST_STATUS_PENDING: PairingRequestStatus
PAIRING_REQUEST_STATUS_APPROVED: PairingRequestStatus
PAIRING_REQUEST_STATUS_REJECTED: PairingRequestStatus

class RegisterEncryptionKeyRequest(_message.Message):
    __slots__ = ("node_id", "encryption_public_key")
    NODE_ID_FIELD_NUMBER: _ClassVar[int]
    ENCRYPTION_PUBLIC_KEY_FIELD_NUMBER: _ClassVar[int]
    node_id: str
    encryption_public_key: str
    def __init__(self, node_id: _Optional[str] = ..., encryption_public_key: _Optional[str] = ...) -> None: ...

class RegisterEncryptionKeyResponse(_message.Message):
    __slots__ = ("node_id", "algorithm", "registered")
    NODE_ID_FIELD_NUMBER: _ClassVar[int]
    ALGORITHM_FIELD_NUMBER: _ClassVar[int]
    REGISTERED_FIELD_NUMBER: _ClassVar[int]
    node_id: str
    algorithm: str
    registered: bool
    def __init__(self, node_id: _Optional[str] = ..., algorithm: _Optional[str] = ..., registered: _Optional[bool] = ...) -> None: ...

class IssuePairingCodeRequest(_message.Message):
    __slots__ = ("name", "scopes", "ttl_seconds")
    NAME_FIELD_NUMBER: _ClassVar[int]
    SCOPES_FIELD_NUMBER: _ClassVar[int]
    TTL_SECONDS_FIELD_NUMBER: _ClassVar[int]
    name: str
    scopes: _containers.RepeatedScalarFieldContainer[str]
    ttl_seconds: int
    def __init__(self, name: _Optional[str] = ..., scopes: _Optional[_Iterable[str]] = ..., ttl_seconds: _Optional[int] = ...) -> None: ...

class IssuePairingCodeResponse(_message.Message):
    __slots__ = ("code", "control_plane_public_key", "expires_at")
    CODE_FIELD_NUMBER: _ClassVar[int]
    CONTROL_PLANE_PUBLIC_KEY_FIELD_NUMBER: _ClassVar[int]
    EXPIRES_AT_FIELD_NUMBER: _ClassVar[int]
    code: str
    control_plane_public_key: str
    expires_at: _timestamp_pb2.Timestamp
    def __init__(self, code: _Optional[str] = ..., control_plane_public_key: _Optional[str] = ..., expires_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ...) -> None: ...

class RedeemPairingCodeRequest(_message.Message):
    __slots__ = ("code", "node_public_key", "name", "os", "arch", "endpoint", "capabilities")
    CODE_FIELD_NUMBER: _ClassVar[int]
    NODE_PUBLIC_KEY_FIELD_NUMBER: _ClassVar[int]
    NAME_FIELD_NUMBER: _ClassVar[int]
    OS_FIELD_NUMBER: _ClassVar[int]
    ARCH_FIELD_NUMBER: _ClassVar[int]
    ENDPOINT_FIELD_NUMBER: _ClassVar[int]
    CAPABILITIES_FIELD_NUMBER: _ClassVar[int]
    code: str
    node_public_key: str
    name: str
    os: str
    arch: str
    endpoint: str
    capabilities: _containers.RepeatedScalarFieldContainer[str]
    def __init__(self, code: _Optional[str] = ..., node_public_key: _Optional[str] = ..., name: _Optional[str] = ..., os: _Optional[str] = ..., arch: _Optional[str] = ..., endpoint: _Optional[str] = ..., capabilities: _Optional[_Iterable[str]] = ...) -> None: ...

class RedeemPairingCodeResponse(_message.Message):
    __slots__ = ("node_id", "control_plane_public_key")
    NODE_ID_FIELD_NUMBER: _ClassVar[int]
    CONTROL_PLANE_PUBLIC_KEY_FIELD_NUMBER: _ClassVar[int]
    node_id: str
    control_plane_public_key: str
    def __init__(self, node_id: _Optional[str] = ..., control_plane_public_key: _Optional[str] = ...) -> None: ...

class RequestPairingRequest(_message.Message):
    __slots__ = ("node_public_key", "name", "os", "arch", "endpoint", "capabilities")
    NODE_PUBLIC_KEY_FIELD_NUMBER: _ClassVar[int]
    NAME_FIELD_NUMBER: _ClassVar[int]
    OS_FIELD_NUMBER: _ClassVar[int]
    ARCH_FIELD_NUMBER: _ClassVar[int]
    ENDPOINT_FIELD_NUMBER: _ClassVar[int]
    CAPABILITIES_FIELD_NUMBER: _ClassVar[int]
    node_public_key: str
    name: str
    os: str
    arch: str
    endpoint: str
    capabilities: _containers.RepeatedScalarFieldContainer[str]
    def __init__(self, node_public_key: _Optional[str] = ..., name: _Optional[str] = ..., os: _Optional[str] = ..., arch: _Optional[str] = ..., endpoint: _Optional[str] = ..., capabilities: _Optional[_Iterable[str]] = ...) -> None: ...

class RequestPairingResponse(_message.Message):
    __slots__ = ("request_id", "status", "confirmation_words", "key_fingerprint")
    REQUEST_ID_FIELD_NUMBER: _ClassVar[int]
    STATUS_FIELD_NUMBER: _ClassVar[int]
    CONFIRMATION_WORDS_FIELD_NUMBER: _ClassVar[int]
    KEY_FINGERPRINT_FIELD_NUMBER: _ClassVar[int]
    request_id: str
    status: PairingRequestStatus
    confirmation_words: _containers.RepeatedScalarFieldContainer[str]
    key_fingerprint: str
    def __init__(self, request_id: _Optional[str] = ..., status: _Optional[_Union[PairingRequestStatus, str]] = ..., confirmation_words: _Optional[_Iterable[str]] = ..., key_fingerprint: _Optional[str] = ...) -> None: ...

class GetPairingRequestRequest(_message.Message):
    __slots__ = ("request_id",)
    REQUEST_ID_FIELD_NUMBER: _ClassVar[int]
    request_id: str
    def __init__(self, request_id: _Optional[str] = ...) -> None: ...

class GetPairingRequestResponse(_message.Message):
    __slots__ = ("request", "control_plane_public_key")
    REQUEST_FIELD_NUMBER: _ClassVar[int]
    CONTROL_PLANE_PUBLIC_KEY_FIELD_NUMBER: _ClassVar[int]
    request: PairingRequest
    control_plane_public_key: str
    def __init__(self, request: _Optional[_Union[PairingRequest, _Mapping]] = ..., control_plane_public_key: _Optional[str] = ...) -> None: ...

class ApprovePairingRequest(_message.Message):
    __slots__ = ("request_id", "approve", "scopes", "confirmation_words")
    REQUEST_ID_FIELD_NUMBER: _ClassVar[int]
    APPROVE_FIELD_NUMBER: _ClassVar[int]
    SCOPES_FIELD_NUMBER: _ClassVar[int]
    CONFIRMATION_WORDS_FIELD_NUMBER: _ClassVar[int]
    request_id: str
    approve: bool
    scopes: _containers.RepeatedScalarFieldContainer[str]
    confirmation_words: _containers.RepeatedScalarFieldContainer[str]
    def __init__(self, request_id: _Optional[str] = ..., approve: _Optional[bool] = ..., scopes: _Optional[_Iterable[str]] = ..., confirmation_words: _Optional[_Iterable[str]] = ...) -> None: ...

class ApprovePairingResponse(_message.Message):
    __slots__ = ("node_id", "status")
    NODE_ID_FIELD_NUMBER: _ClassVar[int]
    STATUS_FIELD_NUMBER: _ClassVar[int]
    node_id: str
    status: PairingRequestStatus
    def __init__(self, node_id: _Optional[str] = ..., status: _Optional[_Union[PairingRequestStatus, str]] = ...) -> None: ...

class PairingRequest(_message.Message):
    __slots__ = ("id", "name", "os", "arch", "endpoint", "capabilities", "status", "created_at", "decided_at", "node_id", "confirmation_words", "key_fingerprint")
    ID_FIELD_NUMBER: _ClassVar[int]
    NAME_FIELD_NUMBER: _ClassVar[int]
    OS_FIELD_NUMBER: _ClassVar[int]
    ARCH_FIELD_NUMBER: _ClassVar[int]
    ENDPOINT_FIELD_NUMBER: _ClassVar[int]
    CAPABILITIES_FIELD_NUMBER: _ClassVar[int]
    STATUS_FIELD_NUMBER: _ClassVar[int]
    CREATED_AT_FIELD_NUMBER: _ClassVar[int]
    DECIDED_AT_FIELD_NUMBER: _ClassVar[int]
    NODE_ID_FIELD_NUMBER: _ClassVar[int]
    CONFIRMATION_WORDS_FIELD_NUMBER: _ClassVar[int]
    KEY_FINGERPRINT_FIELD_NUMBER: _ClassVar[int]
    id: str
    name: str
    os: str
    arch: str
    endpoint: str
    capabilities: _containers.RepeatedScalarFieldContainer[str]
    status: PairingRequestStatus
    created_at: _timestamp_pb2.Timestamp
    decided_at: _timestamp_pb2.Timestamp
    node_id: str
    confirmation_words: _containers.RepeatedScalarFieldContainer[str]
    key_fingerprint: str
    def __init__(self, id: _Optional[str] = ..., name: _Optional[str] = ..., os: _Optional[str] = ..., arch: _Optional[str] = ..., endpoint: _Optional[str] = ..., capabilities: _Optional[_Iterable[str]] = ..., status: _Optional[_Union[PairingRequestStatus, str]] = ..., created_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., decided_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., node_id: _Optional[str] = ..., confirmation_words: _Optional[_Iterable[str]] = ..., key_fingerprint: _Optional[str] = ...) -> None: ...

class ListPairingRequestsRequest(_message.Message):
    __slots__ = ("include_decided",)
    INCLUDE_DECIDED_FIELD_NUMBER: _ClassVar[int]
    include_decided: bool
    def __init__(self, include_decided: _Optional[bool] = ...) -> None: ...

class ListPairingRequestsResponse(_message.Message):
    __slots__ = ("requests", "presets")
    REQUESTS_FIELD_NUMBER: _ClassVar[int]
    PRESETS_FIELD_NUMBER: _ClassVar[int]
    requests: _containers.RepeatedCompositeFieldContainer[PairingRequest]
    presets: _containers.RepeatedCompositeFieldContainer[PermissionPreset]
    def __init__(self, requests: _Optional[_Iterable[_Union[PairingRequest, _Mapping]]] = ..., presets: _Optional[_Iterable[_Union[PermissionPreset, _Mapping]]] = ...) -> None: ...

class PermissionPreset(_message.Message):
    __slots__ = ("name", "description", "scopes", "withholds")
    NAME_FIELD_NUMBER: _ClassVar[int]
    DESCRIPTION_FIELD_NUMBER: _ClassVar[int]
    SCOPES_FIELD_NUMBER: _ClassVar[int]
    WITHHOLDS_FIELD_NUMBER: _ClassVar[int]
    name: str
    description: str
    scopes: _containers.RepeatedScalarFieldContainer[str]
    withholds: _containers.RepeatedScalarFieldContainer[str]
    def __init__(self, name: _Optional[str] = ..., description: _Optional[str] = ..., scopes: _Optional[_Iterable[str]] = ..., withholds: _Optional[_Iterable[str]] = ...) -> None: ...
