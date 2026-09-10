import datetime

from google.protobuf import timestamp_pb2 as _timestamp_pb2
from scenario_to_cloud.v1.errors import errors_pb2 as _errors_pb2
from google.protobuf.internal import containers as _containers
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Iterable as _Iterable, Mapping as _Mapping
from typing import ClassVar as _ClassVar, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class EdgeRoute(_message.Message):
    __slots__ = ("host", "upstream_port", "listener_id")
    HOST_FIELD_NUMBER: _ClassVar[int]
    UPSTREAM_PORT_FIELD_NUMBER: _ClassVar[int]
    LISTENER_ID_FIELD_NUMBER: _ClassVar[int]
    host: str
    upstream_port: int
    listener_id: str
    def __init__(self, host: _Optional[str] = ..., upstream_port: _Optional[int] = ..., listener_id: _Optional[str] = ...) -> None: ...

class PrivateListener(_message.Message):
    __slots__ = ("id", "owner", "port_name", "port", "reason")
    ID_FIELD_NUMBER: _ClassVar[int]
    OWNER_FIELD_NUMBER: _ClassVar[int]
    PORT_NAME_FIELD_NUMBER: _ClassVar[int]
    PORT_FIELD_NUMBER: _ClassVar[int]
    REASON_FIELD_NUMBER: _ClassVar[int]
    id: str
    owner: str
    port_name: str
    port: int
    reason: str
    def __init__(self, id: _Optional[str] = ..., owner: _Optional[str] = ..., port_name: _Optional[str] = ..., port: _Optional[int] = ..., reason: _Optional[str] = ...) -> None: ...

class AddressBinding(_message.Message):
    __slots__ = ("expected", "observed", "state")
    EXPECTED_FIELD_NUMBER: _ClassVar[int]
    OBSERVED_FIELD_NUMBER: _ClassVar[int]
    STATE_FIELD_NUMBER: _ClassVar[int]
    expected: _containers.RepeatedScalarFieldContainer[str]
    observed: _containers.RepeatedScalarFieldContainer[str]
    state: str
    def __init__(self, expected: _Optional[_Iterable[str]] = ..., observed: _Optional[_Iterable[str]] = ..., state: _Optional[str] = ...) -> None: ...

class DNSBinding(_message.Message):
    __slots__ = ("host", "ipv4", "ipv6", "match", "reason_code")
    HOST_FIELD_NUMBER: _ClassVar[int]
    IPV4_FIELD_NUMBER: _ClassVar[int]
    IPV6_FIELD_NUMBER: _ClassVar[int]
    MATCH_FIELD_NUMBER: _ClassVar[int]
    REASON_CODE_FIELD_NUMBER: _ClassVar[int]
    host: str
    ipv4: AddressBinding
    ipv6: AddressBinding
    match: bool
    reason_code: str
    def __init__(self, host: _Optional[str] = ..., ipv4: _Optional[_Union[AddressBinding, _Mapping]] = ..., ipv6: _Optional[_Union[AddressBinding, _Mapping]] = ..., match: _Optional[bool] = ..., reason_code: _Optional[str] = ...) -> None: ...

class TLSState(_message.Message):
    __slots__ = ("host", "issuer", "not_after", "days_left", "renewal_state", "reason_code", "detail", "acme_environment")
    HOST_FIELD_NUMBER: _ClassVar[int]
    ISSUER_FIELD_NUMBER: _ClassVar[int]
    NOT_AFTER_FIELD_NUMBER: _ClassVar[int]
    DAYS_LEFT_FIELD_NUMBER: _ClassVar[int]
    RENEWAL_STATE_FIELD_NUMBER: _ClassVar[int]
    REASON_CODE_FIELD_NUMBER: _ClassVar[int]
    DETAIL_FIELD_NUMBER: _ClassVar[int]
    ACME_ENVIRONMENT_FIELD_NUMBER: _ClassVar[int]
    host: str
    issuer: str
    not_after: str
    days_left: int
    renewal_state: str
    reason_code: str
    detail: str
    acme_environment: str
    def __init__(self, host: _Optional[str] = ..., issuer: _Optional[str] = ..., not_after: _Optional[str] = ..., days_left: _Optional[int] = ..., renewal_state: _Optional[str] = ..., reason_code: _Optional[str] = ..., detail: _Optional[str] = ..., acme_environment: _Optional[str] = ...) -> None: ...

class ReadinessCheck(_message.Message):
    __slots__ = ("status", "reason_code", "detail")
    STATUS_FIELD_NUMBER: _ClassVar[int]
    REASON_CODE_FIELD_NUMBER: _ClassVar[int]
    DETAIL_FIELD_NUMBER: _ClassVar[int]
    status: str
    reason_code: str
    detail: str
    def __init__(self, status: _Optional[str] = ..., reason_code: _Optional[str] = ..., detail: _Optional[str] = ...) -> None: ...

class Readiness(_message.Message):
    __slots__ = ("local", "external")
    LOCAL_FIELD_NUMBER: _ClassVar[int]
    EXTERNAL_FIELD_NUMBER: _ClassVar[int]
    local: ReadinessCheck
    external: ReadinessCheck
    def __init__(self, local: _Optional[_Union[ReadinessCheck, _Mapping]] = ..., external: _Optional[_Union[ReadinessCheck, _Mapping]] = ...) -> None: ...

class EdgeObservation(_message.Message):
    __slots__ = ("schema_version", "deployment_id", "domain", "spec_digest", "routes", "private_listeners", "dns", "tls", "acme_environment", "readiness", "observed_at", "producer_ref", "next_actions")
    SCHEMA_VERSION_FIELD_NUMBER: _ClassVar[int]
    DEPLOYMENT_ID_FIELD_NUMBER: _ClassVar[int]
    DOMAIN_FIELD_NUMBER: _ClassVar[int]
    SPEC_DIGEST_FIELD_NUMBER: _ClassVar[int]
    ROUTES_FIELD_NUMBER: _ClassVar[int]
    PRIVATE_LISTENERS_FIELD_NUMBER: _ClassVar[int]
    DNS_FIELD_NUMBER: _ClassVar[int]
    TLS_FIELD_NUMBER: _ClassVar[int]
    ACME_ENVIRONMENT_FIELD_NUMBER: _ClassVar[int]
    READINESS_FIELD_NUMBER: _ClassVar[int]
    OBSERVED_AT_FIELD_NUMBER: _ClassVar[int]
    PRODUCER_REF_FIELD_NUMBER: _ClassVar[int]
    NEXT_ACTIONS_FIELD_NUMBER: _ClassVar[int]
    schema_version: str
    deployment_id: str
    domain: str
    spec_digest: str
    routes: _containers.RepeatedCompositeFieldContainer[EdgeRoute]
    private_listeners: _containers.RepeatedCompositeFieldContainer[PrivateListener]
    dns: _containers.RepeatedCompositeFieldContainer[DNSBinding]
    tls: _containers.RepeatedCompositeFieldContainer[TLSState]
    acme_environment: str
    readiness: Readiness
    observed_at: _timestamp_pb2.Timestamp
    producer_ref: str
    next_actions: _containers.RepeatedCompositeFieldContainer[_errors_pb2.NextAction]
    def __init__(self, schema_version: _Optional[str] = ..., deployment_id: _Optional[str] = ..., domain: _Optional[str] = ..., spec_digest: _Optional[str] = ..., routes: _Optional[_Iterable[_Union[EdgeRoute, _Mapping]]] = ..., private_listeners: _Optional[_Iterable[_Union[PrivateListener, _Mapping]]] = ..., dns: _Optional[_Iterable[_Union[DNSBinding, _Mapping]]] = ..., tls: _Optional[_Iterable[_Union[TLSState, _Mapping]]] = ..., acme_environment: _Optional[str] = ..., readiness: _Optional[_Union[Readiness, _Mapping]] = ..., observed_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., producer_ref: _Optional[str] = ..., next_actions: _Optional[_Iterable[_Union[_errors_pb2.NextAction, _Mapping]]] = ...) -> None: ...

class GetEdgeObservationRequest(_message.Message):
    __slots__ = ("deployment_id",)
    DEPLOYMENT_ID_FIELD_NUMBER: _ClassVar[int]
    deployment_id: str
    def __init__(self, deployment_id: _Optional[str] = ...) -> None: ...

class GetEdgeObservationResponse(_message.Message):
    __slots__ = ("schema_version", "observation")
    SCHEMA_VERSION_FIELD_NUMBER: _ClassVar[int]
    OBSERVATION_FIELD_NUMBER: _ClassVar[int]
    schema_version: str
    observation: EdgeObservation
    def __init__(self, schema_version: _Optional[str] = ..., observation: _Optional[_Union[EdgeObservation, _Mapping]] = ...) -> None: ...
