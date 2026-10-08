import datetime

from google.protobuf import timestamp_pb2 as _timestamp_pb2
from scenario_to_cloud.v1.errors import errors_pb2 as _errors_pb2
from google.protobuf.internal import containers as _containers
from google.protobuf.internal import enum_type_wrapper as _enum_type_wrapper
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Iterable as _Iterable, Mapping as _Mapping
from typing import ClassVar as _ClassVar, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class HealthStatus(int, metaclass=_enum_type_wrapper.EnumTypeWrapper):
    __slots__ = ()
    HEALTH_STATUS_UNSPECIFIED: _ClassVar[HealthStatus]
    HEALTH_STATUS_HEALTHY: _ClassVar[HealthStatus]
    HEALTH_STATUS_DEGRADED: _ClassVar[HealthStatus]
    HEALTH_STATUS_UNHEALTHY: _ClassVar[HealthStatus]
    HEALTH_STATUS_UNKNOWN: _ClassVar[HealthStatus]

class Freshness(int, metaclass=_enum_type_wrapper.EnumTypeWrapper):
    __slots__ = ()
    FRESHNESS_UNSPECIFIED: _ClassVar[Freshness]
    FRESHNESS_CURRENT: _ClassVar[Freshness]
    FRESHNESS_STALE: _ClassVar[Freshness]
    FRESHNESS_UNKNOWN: _ClassVar[Freshness]

class CheckStatus(int, metaclass=_enum_type_wrapper.EnumTypeWrapper):
    __slots__ = ()
    CHECK_STATUS_UNSPECIFIED: _ClassVar[CheckStatus]
    CHECK_STATUS_PASSED: _ClassVar[CheckStatus]
    CHECK_STATUS_WARNED: _ClassVar[CheckStatus]
    CHECK_STATUS_FAILED: _ClassVar[CheckStatus]
    CHECK_STATUS_SKIPPED: _ClassVar[CheckStatus]
    CHECK_STATUS_UNAVAILABLE: _ClassVar[CheckStatus]
HEALTH_STATUS_UNSPECIFIED: HealthStatus
HEALTH_STATUS_HEALTHY: HealthStatus
HEALTH_STATUS_DEGRADED: HealthStatus
HEALTH_STATUS_UNHEALTHY: HealthStatus
HEALTH_STATUS_UNKNOWN: HealthStatus
FRESHNESS_UNSPECIFIED: Freshness
FRESHNESS_CURRENT: Freshness
FRESHNESS_STALE: Freshness
FRESHNESS_UNKNOWN: Freshness
CHECK_STATUS_UNSPECIFIED: CheckStatus
CHECK_STATUS_PASSED: CheckStatus
CHECK_STATUS_WARNED: CheckStatus
CHECK_STATUS_FAILED: CheckStatus
CHECK_STATUS_SKIPPED: CheckStatus
CHECK_STATUS_UNAVAILABLE: CheckStatus

class HealthCheck(_message.Message):
    __slots__ = ("id", "status", "reason_code", "detail")
    ID_FIELD_NUMBER: _ClassVar[int]
    STATUS_FIELD_NUMBER: _ClassVar[int]
    REASON_CODE_FIELD_NUMBER: _ClassVar[int]
    DETAIL_FIELD_NUMBER: _ClassVar[int]
    id: str
    status: CheckStatus
    reason_code: str
    detail: str
    def __init__(self, id: _Optional[str] = ..., status: _Optional[_Union[CheckStatus, str]] = ..., reason_code: _Optional[str] = ..., detail: _Optional[str] = ...) -> None: ...

class HealthObservation(_message.Message):
    __slots__ = ("deployment_id", "target_id", "observed_release_digest", "observed_configuration_digest", "observed_at", "status", "checks", "freshness", "producer_ref", "partial", "missing_dependencies", "next_actions")
    DEPLOYMENT_ID_FIELD_NUMBER: _ClassVar[int]
    TARGET_ID_FIELD_NUMBER: _ClassVar[int]
    OBSERVED_RELEASE_DIGEST_FIELD_NUMBER: _ClassVar[int]
    OBSERVED_CONFIGURATION_DIGEST_FIELD_NUMBER: _ClassVar[int]
    OBSERVED_AT_FIELD_NUMBER: _ClassVar[int]
    STATUS_FIELD_NUMBER: _ClassVar[int]
    CHECKS_FIELD_NUMBER: _ClassVar[int]
    FRESHNESS_FIELD_NUMBER: _ClassVar[int]
    PRODUCER_REF_FIELD_NUMBER: _ClassVar[int]
    PARTIAL_FIELD_NUMBER: _ClassVar[int]
    MISSING_DEPENDENCIES_FIELD_NUMBER: _ClassVar[int]
    NEXT_ACTIONS_FIELD_NUMBER: _ClassVar[int]
    deployment_id: str
    target_id: str
    observed_release_digest: str
    observed_configuration_digest: str
    observed_at: _timestamp_pb2.Timestamp
    status: HealthStatus
    checks: _containers.RepeatedCompositeFieldContainer[HealthCheck]
    freshness: Freshness
    producer_ref: str
    partial: bool
    missing_dependencies: _containers.RepeatedScalarFieldContainer[str]
    next_actions: _containers.RepeatedCompositeFieldContainer[_errors_pb2.NextAction]
    def __init__(self, deployment_id: _Optional[str] = ..., target_id: _Optional[str] = ..., observed_release_digest: _Optional[str] = ..., observed_configuration_digest: _Optional[str] = ..., observed_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., status: _Optional[_Union[HealthStatus, str]] = ..., checks: _Optional[_Iterable[_Union[HealthCheck, _Mapping]]] = ..., freshness: _Optional[_Union[Freshness, str]] = ..., producer_ref: _Optional[str] = ..., partial: _Optional[bool] = ..., missing_dependencies: _Optional[_Iterable[str]] = ..., next_actions: _Optional[_Iterable[_Union[_errors_pb2.NextAction, _Mapping]]] = ...) -> None: ...

class GetHealthObservationRequest(_message.Message):
    __slots__ = ("deployment_id",)
    DEPLOYMENT_ID_FIELD_NUMBER: _ClassVar[int]
    deployment_id: str
    def __init__(self, deployment_id: _Optional[str] = ...) -> None: ...

class GetHealthObservationResponse(_message.Message):
    __slots__ = ("schema_version", "observation")
    SCHEMA_VERSION_FIELD_NUMBER: _ClassVar[int]
    OBSERVATION_FIELD_NUMBER: _ClassVar[int]
    schema_version: str
    observation: HealthObservation
    def __init__(self, schema_version: _Optional[str] = ..., observation: _Optional[_Union[HealthObservation, _Mapping]] = ...) -> None: ...
