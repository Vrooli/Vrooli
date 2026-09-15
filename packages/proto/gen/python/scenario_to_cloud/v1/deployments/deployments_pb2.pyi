import datetime

from google.protobuf import struct_pb2 as _struct_pb2
from google.protobuf import timestamp_pb2 as _timestamp_pb2
from scenario_to_cloud.v1.identity import identity_pb2 as _identity_pb2
from google.protobuf.internal import containers as _containers
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Iterable as _Iterable, Mapping as _Mapping
from typing import ClassVar as _ClassVar, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class DeploymentSelector(_message.Message):
    __slots__ = ("id", "scenario_id", "environment", "domain", "host")
    ID_FIELD_NUMBER: _ClassVar[int]
    SCENARIO_ID_FIELD_NUMBER: _ClassVar[int]
    ENVIRONMENT_FIELD_NUMBER: _ClassVar[int]
    DOMAIN_FIELD_NUMBER: _ClassVar[int]
    HOST_FIELD_NUMBER: _ClassVar[int]
    id: str
    scenario_id: str
    environment: str
    domain: str
    host: str
    def __init__(self, id: _Optional[str] = ..., scenario_id: _Optional[str] = ..., environment: _Optional[str] = ..., domain: _Optional[str] = ..., host: _Optional[str] = ...) -> None: ...

class ResolveDeploymentRequest(_message.Message):
    __slots__ = ("selector",)
    SELECTOR_FIELD_NUMBER: _ClassVar[int]
    selector: DeploymentSelector
    def __init__(self, selector: _Optional[_Union[DeploymentSelector, _Mapping]] = ...) -> None: ...

class ResolveDeploymentResponse(_message.Message):
    __slots__ = ("schema_version", "ref")
    SCHEMA_VERSION_FIELD_NUMBER: _ClassVar[int]
    REF_FIELD_NUMBER: _ClassVar[int]
    schema_version: str
    ref: _identity_pb2.DeploymentRef
    def __init__(self, schema_version: _Optional[str] = ..., ref: _Optional[_Union[_identity_pb2.DeploymentRef, _Mapping]] = ...) -> None: ...

class GetDeploymentRequest(_message.Message):
    __slots__ = ("id",)
    ID_FIELD_NUMBER: _ClassVar[int]
    id: str
    def __init__(self, id: _Optional[str] = ...) -> None: ...

class GetDeploymentResponse(_message.Message):
    __slots__ = ("schema_version", "deployment")
    SCHEMA_VERSION_FIELD_NUMBER: _ClassVar[int]
    DEPLOYMENT_FIELD_NUMBER: _ClassVar[int]
    schema_version: str
    deployment: Deployment
    def __init__(self, schema_version: _Optional[str] = ..., deployment: _Optional[_Union[Deployment, _Mapping]] = ...) -> None: ...

class ListDeploymentsRequest(_message.Message):
    __slots__ = ("scenario_id", "environment", "status", "page_size", "page_token")
    SCENARIO_ID_FIELD_NUMBER: _ClassVar[int]
    ENVIRONMENT_FIELD_NUMBER: _ClassVar[int]
    STATUS_FIELD_NUMBER: _ClassVar[int]
    PAGE_SIZE_FIELD_NUMBER: _ClassVar[int]
    PAGE_TOKEN_FIELD_NUMBER: _ClassVar[int]
    scenario_id: str
    environment: str
    status: str
    page_size: int
    page_token: str
    def __init__(self, scenario_id: _Optional[str] = ..., environment: _Optional[str] = ..., status: _Optional[str] = ..., page_size: _Optional[int] = ..., page_token: _Optional[str] = ...) -> None: ...

class ListDeploymentsResponse(_message.Message):
    __slots__ = ("schema_version", "deployments", "next_page_token")
    SCHEMA_VERSION_FIELD_NUMBER: _ClassVar[int]
    DEPLOYMENTS_FIELD_NUMBER: _ClassVar[int]
    NEXT_PAGE_TOKEN_FIELD_NUMBER: _ClassVar[int]
    schema_version: str
    deployments: _containers.RepeatedCompositeFieldContainer[DeploymentSummary]
    next_page_token: str
    def __init__(self, schema_version: _Optional[str] = ..., deployments: _Optional[_Iterable[_Union[DeploymentSummary, _Mapping]]] = ..., next_page_token: _Optional[str] = ...) -> None: ...

class DeploymentSummary(_message.Message):
    __slots__ = ("ref", "name", "status", "domain", "error_message", "progress_step", "progress_percent", "created_at", "last_deployed_at")
    REF_FIELD_NUMBER: _ClassVar[int]
    NAME_FIELD_NUMBER: _ClassVar[int]
    STATUS_FIELD_NUMBER: _ClassVar[int]
    DOMAIN_FIELD_NUMBER: _ClassVar[int]
    ERROR_MESSAGE_FIELD_NUMBER: _ClassVar[int]
    PROGRESS_STEP_FIELD_NUMBER: _ClassVar[int]
    PROGRESS_PERCENT_FIELD_NUMBER: _ClassVar[int]
    CREATED_AT_FIELD_NUMBER: _ClassVar[int]
    LAST_DEPLOYED_AT_FIELD_NUMBER: _ClassVar[int]
    ref: _identity_pb2.DeploymentRef
    name: str
    status: str
    domain: str
    error_message: str
    progress_step: str
    progress_percent: float
    created_at: _timestamp_pb2.Timestamp
    last_deployed_at: _timestamp_pb2.Timestamp
    def __init__(self, ref: _Optional[_Union[_identity_pb2.DeploymentRef, _Mapping]] = ..., name: _Optional[str] = ..., status: _Optional[str] = ..., domain: _Optional[str] = ..., error_message: _Optional[str] = ..., progress_step: _Optional[str] = ..., progress_percent: _Optional[float] = ..., created_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., last_deployed_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ...) -> None: ...

class Deployment(_message.Message):
    __slots__ = ("ref", "name", "status", "domain", "fence", "manifest", "bundle_sha256", "error_message", "error_step", "progress_step", "progress_percent", "created_at", "updated_at", "last_deployed_at", "last_inspected_at", "desired_state")
    REF_FIELD_NUMBER: _ClassVar[int]
    NAME_FIELD_NUMBER: _ClassVar[int]
    STATUS_FIELD_NUMBER: _ClassVar[int]
    DOMAIN_FIELD_NUMBER: _ClassVar[int]
    FENCE_FIELD_NUMBER: _ClassVar[int]
    MANIFEST_FIELD_NUMBER: _ClassVar[int]
    BUNDLE_SHA256_FIELD_NUMBER: _ClassVar[int]
    ERROR_MESSAGE_FIELD_NUMBER: _ClassVar[int]
    ERROR_STEP_FIELD_NUMBER: _ClassVar[int]
    PROGRESS_STEP_FIELD_NUMBER: _ClassVar[int]
    PROGRESS_PERCENT_FIELD_NUMBER: _ClassVar[int]
    CREATED_AT_FIELD_NUMBER: _ClassVar[int]
    UPDATED_AT_FIELD_NUMBER: _ClassVar[int]
    LAST_DEPLOYED_AT_FIELD_NUMBER: _ClassVar[int]
    LAST_INSPECTED_AT_FIELD_NUMBER: _ClassVar[int]
    DESIRED_STATE_FIELD_NUMBER: _ClassVar[int]
    ref: _identity_pb2.DeploymentRef
    name: str
    status: str
    domain: str
    fence: int
    manifest: _struct_pb2.Struct
    bundle_sha256: str
    error_message: str
    error_step: str
    progress_step: str
    progress_percent: float
    created_at: _timestamp_pb2.Timestamp
    updated_at: _timestamp_pb2.Timestamp
    last_deployed_at: _timestamp_pb2.Timestamp
    last_inspected_at: _timestamp_pb2.Timestamp
    desired_state: str
    def __init__(self, ref: _Optional[_Union[_identity_pb2.DeploymentRef, _Mapping]] = ..., name: _Optional[str] = ..., status: _Optional[str] = ..., domain: _Optional[str] = ..., fence: _Optional[int] = ..., manifest: _Optional[_Union[_struct_pb2.Struct, _Mapping]] = ..., bundle_sha256: _Optional[str] = ..., error_message: _Optional[str] = ..., error_step: _Optional[str] = ..., progress_step: _Optional[str] = ..., progress_percent: _Optional[float] = ..., created_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., updated_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., last_deployed_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., last_inspected_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., desired_state: _Optional[str] = ...) -> None: ...
