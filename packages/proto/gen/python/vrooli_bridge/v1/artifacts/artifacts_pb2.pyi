import datetime

from google.protobuf import timestamp_pb2 as _timestamp_pb2
from google.protobuf.internal import containers as _containers
from google.protobuf.internal import enum_type_wrapper as _enum_type_wrapper
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Iterable as _Iterable, Mapping as _Mapping
from typing import ClassVar as _ClassVar, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class DeliveryStatus(int, metaclass=_enum_type_wrapper.EnumTypeWrapper):
    __slots__ = ()
    DELIVERY_STATUS_UNSPECIFIED: _ClassVar[DeliveryStatus]
    DELIVERY_STATUS_PENDING: _ClassVar[DeliveryStatus]
    DELIVERY_STATUS_DELIVERED: _ClassVar[DeliveryStatus]
    DELIVERY_STATUS_FAILED: _ClassVar[DeliveryStatus]
DELIVERY_STATUS_UNSPECIFIED: DeliveryStatus
DELIVERY_STATUS_PENDING: DeliveryStatus
DELIVERY_STATUS_DELIVERED: DeliveryStatus
DELIVERY_STATUS_FAILED: DeliveryStatus

class Distribution(_message.Message):
    __slots__ = ("id", "node_id", "name", "source_ref", "destination_path", "status", "delivery_ref", "detail", "created_at", "updated_at")
    ID_FIELD_NUMBER: _ClassVar[int]
    NODE_ID_FIELD_NUMBER: _ClassVar[int]
    NAME_FIELD_NUMBER: _ClassVar[int]
    SOURCE_REF_FIELD_NUMBER: _ClassVar[int]
    DESTINATION_PATH_FIELD_NUMBER: _ClassVar[int]
    STATUS_FIELD_NUMBER: _ClassVar[int]
    DELIVERY_REF_FIELD_NUMBER: _ClassVar[int]
    DETAIL_FIELD_NUMBER: _ClassVar[int]
    CREATED_AT_FIELD_NUMBER: _ClassVar[int]
    UPDATED_AT_FIELD_NUMBER: _ClassVar[int]
    id: str
    node_id: str
    name: str
    source_ref: str
    destination_path: str
    status: DeliveryStatus
    delivery_ref: str
    detail: str
    created_at: _timestamp_pb2.Timestamp
    updated_at: _timestamp_pb2.Timestamp
    def __init__(self, id: _Optional[str] = ..., node_id: _Optional[str] = ..., name: _Optional[str] = ..., source_ref: _Optional[str] = ..., destination_path: _Optional[str] = ..., status: _Optional[_Union[DeliveryStatus, str]] = ..., delivery_ref: _Optional[str] = ..., detail: _Optional[str] = ..., created_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., updated_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ...) -> None: ...

class DistributeArtifactRequest(_message.Message):
    __slots__ = ("node_id", "name", "source_ref", "destination_path")
    NODE_ID_FIELD_NUMBER: _ClassVar[int]
    NAME_FIELD_NUMBER: _ClassVar[int]
    SOURCE_REF_FIELD_NUMBER: _ClassVar[int]
    DESTINATION_PATH_FIELD_NUMBER: _ClassVar[int]
    node_id: str
    name: str
    source_ref: str
    destination_path: str
    def __init__(self, node_id: _Optional[str] = ..., name: _Optional[str] = ..., source_ref: _Optional[str] = ..., destination_path: _Optional[str] = ...) -> None: ...

class DistributeArtifactResponse(_message.Message):
    __slots__ = ("distribution_id", "dry_run", "status", "delivery_ref")
    DISTRIBUTION_ID_FIELD_NUMBER: _ClassVar[int]
    DRY_RUN_FIELD_NUMBER: _ClassVar[int]
    STATUS_FIELD_NUMBER: _ClassVar[int]
    DELIVERY_REF_FIELD_NUMBER: _ClassVar[int]
    distribution_id: str
    dry_run: bool
    status: DeliveryStatus
    delivery_ref: str
    def __init__(self, distribution_id: _Optional[str] = ..., dry_run: _Optional[bool] = ..., status: _Optional[_Union[DeliveryStatus, str]] = ..., delivery_ref: _Optional[str] = ...) -> None: ...

class GetDistributionRequest(_message.Message):
    __slots__ = ("id",)
    ID_FIELD_NUMBER: _ClassVar[int]
    id: str
    def __init__(self, id: _Optional[str] = ...) -> None: ...

class GetDistributionResponse(_message.Message):
    __slots__ = ("distribution",)
    DISTRIBUTION_FIELD_NUMBER: _ClassVar[int]
    distribution: Distribution
    def __init__(self, distribution: _Optional[_Union[Distribution, _Mapping]] = ...) -> None: ...

class ListDistributionsRequest(_message.Message):
    __slots__ = ("node_id", "limit")
    NODE_ID_FIELD_NUMBER: _ClassVar[int]
    LIMIT_FIELD_NUMBER: _ClassVar[int]
    node_id: str
    limit: int
    def __init__(self, node_id: _Optional[str] = ..., limit: _Optional[int] = ...) -> None: ...

class ListDistributionsResponse(_message.Message):
    __slots__ = ("distributions",)
    DISTRIBUTIONS_FIELD_NUMBER: _ClassVar[int]
    distributions: _containers.RepeatedCompositeFieldContainer[Distribution]
    def __init__(self, distributions: _Optional[_Iterable[_Union[Distribution, _Mapping]]] = ...) -> None: ...

class UploadRunArtifactRequest(_message.Message):
    __slots__ = ("run_id", "name", "media_type", "data")
    RUN_ID_FIELD_NUMBER: _ClassVar[int]
    NAME_FIELD_NUMBER: _ClassVar[int]
    MEDIA_TYPE_FIELD_NUMBER: _ClassVar[int]
    DATA_FIELD_NUMBER: _ClassVar[int]
    run_id: str
    name: str
    media_type: str
    data: bytes
    def __init__(self, run_id: _Optional[str] = ..., name: _Optional[str] = ..., media_type: _Optional[str] = ..., data: _Optional[bytes] = ...) -> None: ...

class UploadRunArtifactResponse(_message.Message):
    __slots__ = ("artifact_ref", "size_bytes")
    ARTIFACT_REF_FIELD_NUMBER: _ClassVar[int]
    SIZE_BYTES_FIELD_NUMBER: _ClassVar[int]
    artifact_ref: str
    size_bytes: int
    def __init__(self, artifact_ref: _Optional[str] = ..., size_bytes: _Optional[int] = ...) -> None: ...

class GetRunArtifactRequest(_message.Message):
    __slots__ = ("run_id", "name")
    RUN_ID_FIELD_NUMBER: _ClassVar[int]
    NAME_FIELD_NUMBER: _ClassVar[int]
    run_id: str
    name: str
    def __init__(self, run_id: _Optional[str] = ..., name: _Optional[str] = ...) -> None: ...

class GetRunArtifactResponse(_message.Message):
    __slots__ = ("run_id", "name", "media_type", "data", "artifact_ref")
    RUN_ID_FIELD_NUMBER: _ClassVar[int]
    NAME_FIELD_NUMBER: _ClassVar[int]
    MEDIA_TYPE_FIELD_NUMBER: _ClassVar[int]
    DATA_FIELD_NUMBER: _ClassVar[int]
    ARTIFACT_REF_FIELD_NUMBER: _ClassVar[int]
    run_id: str
    name: str
    media_type: str
    data: bytes
    artifact_ref: str
    def __init__(self, run_id: _Optional[str] = ..., name: _Optional[str] = ..., media_type: _Optional[str] = ..., data: _Optional[bytes] = ..., artifact_ref: _Optional[str] = ...) -> None: ...
