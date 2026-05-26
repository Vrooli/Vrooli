import datetime

from buf.validate import validate_pb2 as _validate_pb2
from google.protobuf import timestamp_pb2 as _timestamp_pb2
from google.protobuf.internal import containers as _containers
from google.protobuf.internal import enum_type_wrapper as _enum_type_wrapper
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Iterable as _Iterable, Mapping as _Mapping
from typing import ClassVar as _ClassVar, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class BackendKind(int, metaclass=_enum_type_wrapper.EnumTypeWrapper):
    __slots__ = ()
    BACKEND_KIND_UNSPECIFIED: _ClassVar[BackendKind]
    BACKEND_KIND_FILESYSTEM: _ClassVar[BackendKind]
    BACKEND_KIND_S3: _ClassVar[BackendKind]

class CapPolicy(int, metaclass=_enum_type_wrapper.EnumTypeWrapper):
    __slots__ = ()
    CAP_POLICY_UNSPECIFIED: _ClassVar[CapPolicy]
    CAP_POLICY_ALERT_BLOCK: _ClassVar[CapPolicy]
    CAP_POLICY_ALERT_ONLY: _ClassVar[CapPolicy]

class UsageState(int, metaclass=_enum_type_wrapper.EnumTypeWrapper):
    __slots__ = ()
    USAGE_STATE_UNSPECIFIED: _ClassVar[UsageState]
    USAGE_STATE_WITHIN: _ClassVar[UsageState]
    USAGE_STATE_NEAR: _ClassVar[UsageState]
    USAGE_STATE_OVER: _ClassVar[UsageState]
BACKEND_KIND_UNSPECIFIED: BackendKind
BACKEND_KIND_FILESYSTEM: BackendKind
BACKEND_KIND_S3: BackendKind
CAP_POLICY_UNSPECIFIED: CapPolicy
CAP_POLICY_ALERT_BLOCK: CapPolicy
CAP_POLICY_ALERT_ONLY: CapPolicy
USAGE_STATE_UNSPECIFIED: UsageState
USAGE_STATE_WITHIN: UsageState
USAGE_STATE_NEAR: UsageState
USAGE_STATE_OVER: UsageState

class Destination(_message.Message):
    __slots__ = ("id", "name", "backend_kind", "location", "cap_bytes", "cap_policy", "encryption_algorithm", "secret_ref", "usage_bytes", "usage_state", "created_at", "updated_at")
    ID_FIELD_NUMBER: _ClassVar[int]
    NAME_FIELD_NUMBER: _ClassVar[int]
    BACKEND_KIND_FIELD_NUMBER: _ClassVar[int]
    LOCATION_FIELD_NUMBER: _ClassVar[int]
    CAP_BYTES_FIELD_NUMBER: _ClassVar[int]
    CAP_POLICY_FIELD_NUMBER: _ClassVar[int]
    ENCRYPTION_ALGORITHM_FIELD_NUMBER: _ClassVar[int]
    SECRET_REF_FIELD_NUMBER: _ClassVar[int]
    USAGE_BYTES_FIELD_NUMBER: _ClassVar[int]
    USAGE_STATE_FIELD_NUMBER: _ClassVar[int]
    CREATED_AT_FIELD_NUMBER: _ClassVar[int]
    UPDATED_AT_FIELD_NUMBER: _ClassVar[int]
    id: str
    name: str
    backend_kind: BackendKind
    location: str
    cap_bytes: int
    cap_policy: CapPolicy
    encryption_algorithm: str
    secret_ref: str
    usage_bytes: int
    usage_state: UsageState
    created_at: _timestamp_pb2.Timestamp
    updated_at: _timestamp_pb2.Timestamp
    def __init__(self, id: _Optional[str] = ..., name: _Optional[str] = ..., backend_kind: _Optional[_Union[BackendKind, str]] = ..., location: _Optional[str] = ..., cap_bytes: _Optional[int] = ..., cap_policy: _Optional[_Union[CapPolicy, str]] = ..., encryption_algorithm: _Optional[str] = ..., secret_ref: _Optional[str] = ..., usage_bytes: _Optional[int] = ..., usage_state: _Optional[_Union[UsageState, str]] = ..., created_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., updated_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ...) -> None: ...

class CreateDestinationRequest(_message.Message):
    __slots__ = ("name", "backend_kind", "location", "cap_bytes", "cap_policy")
    NAME_FIELD_NUMBER: _ClassVar[int]
    BACKEND_KIND_FIELD_NUMBER: _ClassVar[int]
    LOCATION_FIELD_NUMBER: _ClassVar[int]
    CAP_BYTES_FIELD_NUMBER: _ClassVar[int]
    CAP_POLICY_FIELD_NUMBER: _ClassVar[int]
    name: str
    backend_kind: BackendKind
    location: str
    cap_bytes: int
    cap_policy: CapPolicy
    def __init__(self, name: _Optional[str] = ..., backend_kind: _Optional[_Union[BackendKind, str]] = ..., location: _Optional[str] = ..., cap_bytes: _Optional[int] = ..., cap_policy: _Optional[_Union[CapPolicy, str]] = ...) -> None: ...

class CreateDestinationResponse(_message.Message):
    __slots__ = ("destination",)
    DESTINATION_FIELD_NUMBER: _ClassVar[int]
    destination: Destination
    def __init__(self, destination: _Optional[_Union[Destination, _Mapping]] = ...) -> None: ...

class GetDestinationRequest(_message.Message):
    __slots__ = ("id",)
    ID_FIELD_NUMBER: _ClassVar[int]
    id: str
    def __init__(self, id: _Optional[str] = ...) -> None: ...

class GetDestinationResponse(_message.Message):
    __slots__ = ("destination",)
    DESTINATION_FIELD_NUMBER: _ClassVar[int]
    destination: Destination
    def __init__(self, destination: _Optional[_Union[Destination, _Mapping]] = ...) -> None: ...

class ListDestinationsRequest(_message.Message):
    __slots__ = ("page_size", "page_token")
    PAGE_SIZE_FIELD_NUMBER: _ClassVar[int]
    PAGE_TOKEN_FIELD_NUMBER: _ClassVar[int]
    page_size: int
    page_token: str
    def __init__(self, page_size: _Optional[int] = ..., page_token: _Optional[str] = ...) -> None: ...

class ListDestinationsResponse(_message.Message):
    __slots__ = ("destinations", "next_page_token")
    DESTINATIONS_FIELD_NUMBER: _ClassVar[int]
    NEXT_PAGE_TOKEN_FIELD_NUMBER: _ClassVar[int]
    destinations: _containers.RepeatedCompositeFieldContainer[Destination]
    next_page_token: str
    def __init__(self, destinations: _Optional[_Iterable[_Union[Destination, _Mapping]]] = ..., next_page_token: _Optional[str] = ...) -> None: ...

class UpdateDestinationRequest(_message.Message):
    __slots__ = ("id", "cap_bytes", "cap_policy")
    ID_FIELD_NUMBER: _ClassVar[int]
    CAP_BYTES_FIELD_NUMBER: _ClassVar[int]
    CAP_POLICY_FIELD_NUMBER: _ClassVar[int]
    id: str
    cap_bytes: int
    cap_policy: CapPolicy
    def __init__(self, id: _Optional[str] = ..., cap_bytes: _Optional[int] = ..., cap_policy: _Optional[_Union[CapPolicy, str]] = ...) -> None: ...

class UpdateDestinationResponse(_message.Message):
    __slots__ = ("destination",)
    DESTINATION_FIELD_NUMBER: _ClassVar[int]
    destination: Destination
    def __init__(self, destination: _Optional[_Union[Destination, _Mapping]] = ...) -> None: ...

class DeleteDestinationRequest(_message.Message):
    __slots__ = ("id", "delete_repository")
    ID_FIELD_NUMBER: _ClassVar[int]
    DELETE_REPOSITORY_FIELD_NUMBER: _ClassVar[int]
    id: str
    delete_repository: bool
    def __init__(self, id: _Optional[str] = ..., delete_repository: _Optional[bool] = ...) -> None: ...

class DeleteDestinationResponse(_message.Message):
    __slots__ = ("removed",)
    REMOVED_FIELD_NUMBER: _ClassVar[int]
    removed: bool
    def __init__(self, removed: _Optional[bool] = ...) -> None: ...

class GetDestinationUsageRequest(_message.Message):
    __slots__ = ("id",)
    ID_FIELD_NUMBER: _ClassVar[int]
    id: str
    def __init__(self, id: _Optional[str] = ...) -> None: ...

class GetDestinationUsageResponse(_message.Message):
    __slots__ = ("usage_bytes", "cap_bytes", "usage_state", "cap_policy")
    USAGE_BYTES_FIELD_NUMBER: _ClassVar[int]
    CAP_BYTES_FIELD_NUMBER: _ClassVar[int]
    USAGE_STATE_FIELD_NUMBER: _ClassVar[int]
    CAP_POLICY_FIELD_NUMBER: _ClassVar[int]
    usage_bytes: int
    cap_bytes: int
    usage_state: UsageState
    cap_policy: CapPolicy
    def __init__(self, usage_bytes: _Optional[int] = ..., cap_bytes: _Optional[int] = ..., usage_state: _Optional[_Union[UsageState, str]] = ..., cap_policy: _Optional[_Union[CapPolicy, str]] = ...) -> None: ...
