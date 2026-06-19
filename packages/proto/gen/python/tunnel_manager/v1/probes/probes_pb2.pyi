import datetime

from google.protobuf import timestamp_pb2 as _timestamp_pb2
from google.protobuf.internal import containers as _containers
from google.protobuf.internal import enum_type_wrapper as _enum_type_wrapper
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Iterable as _Iterable, Mapping as _Mapping
from typing import ClassVar as _ClassVar, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class ProbeKind(int, metaclass=_enum_type_wrapper.EnumTypeWrapper):
    __slots__ = ()
    PROBE_KIND_UNSPECIFIED: _ClassVar[ProbeKind]
    PROBE_KIND_INTERNAL: _ClassVar[ProbeKind]
    PROBE_KIND_EXTERNAL: _ClassVar[ProbeKind]

class ProbeStatus(int, metaclass=_enum_type_wrapper.EnumTypeWrapper):
    __slots__ = ()
    PROBE_STATUS_UNSPECIFIED: _ClassVar[ProbeStatus]
    PROBE_STATUS_UP: _ClassVar[ProbeStatus]
    PROBE_STATUS_DOWN: _ClassVar[ProbeStatus]
    PROBE_STATUS_TIMEOUT: _ClassVar[ProbeStatus]
    PROBE_STATUS_ERROR: _ClassVar[ProbeStatus]

class FailureClass(int, metaclass=_enum_type_wrapper.EnumTypeWrapper):
    __slots__ = ()
    FAILURE_CLASS_UNSPECIFIED: _ClassVar[FailureClass]
    FAILURE_CLASS_HEALTHY: _ClassVar[FailureClass]
    FAILURE_CLASS_TUNNEL_DOWN: _ClassVar[FailureClass]
    FAILURE_CLASS_SCENARIO_DOWN: _ClassVar[FailureClass]
    FAILURE_CLASS_CLOUDFLARE_OUTAGE: _ClassVar[FailureClass]
    FAILURE_CLASS_DNS_FAILURE: _ClassVar[FailureClass]
    FAILURE_CLASS_CONFIG_DRIFT: _ClassVar[FailureClass]
PROBE_KIND_UNSPECIFIED: ProbeKind
PROBE_KIND_INTERNAL: ProbeKind
PROBE_KIND_EXTERNAL: ProbeKind
PROBE_STATUS_UNSPECIFIED: ProbeStatus
PROBE_STATUS_UP: ProbeStatus
PROBE_STATUS_DOWN: ProbeStatus
PROBE_STATUS_TIMEOUT: ProbeStatus
PROBE_STATUS_ERROR: ProbeStatus
FAILURE_CLASS_UNSPECIFIED: FailureClass
FAILURE_CLASS_HEALTHY: FailureClass
FAILURE_CLASS_TUNNEL_DOWN: FailureClass
FAILURE_CLASS_SCENARIO_DOWN: FailureClass
FAILURE_CLASS_CLOUDFLARE_OUTAGE: FailureClass
FAILURE_CLASS_DNS_FAILURE: FailureClass
FAILURE_CLASS_CONFIG_DRIFT: FailureClass

class ProbeResult(_message.Message):
    __slots__ = ("id", "subdomain", "kind", "status", "latency_ms", "status_code", "error_msg", "created_at")
    ID_FIELD_NUMBER: _ClassVar[int]
    SUBDOMAIN_FIELD_NUMBER: _ClassVar[int]
    KIND_FIELD_NUMBER: _ClassVar[int]
    STATUS_FIELD_NUMBER: _ClassVar[int]
    LATENCY_MS_FIELD_NUMBER: _ClassVar[int]
    STATUS_CODE_FIELD_NUMBER: _ClassVar[int]
    ERROR_MSG_FIELD_NUMBER: _ClassVar[int]
    CREATED_AT_FIELD_NUMBER: _ClassVar[int]
    id: str
    subdomain: str
    kind: ProbeKind
    status: ProbeStatus
    latency_ms: int
    status_code: int
    error_msg: str
    created_at: _timestamp_pb2.Timestamp
    def __init__(self, id: _Optional[str] = ..., subdomain: _Optional[str] = ..., kind: _Optional[_Union[ProbeKind, str]] = ..., status: _Optional[_Union[ProbeStatus, str]] = ..., latency_ms: _Optional[int] = ..., status_code: _Optional[int] = ..., error_msg: _Optional[str] = ..., created_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ...) -> None: ...

class RouteClassification(_message.Message):
    __slots__ = ("subdomain", "classification", "internal", "external", "assessment")
    SUBDOMAIN_FIELD_NUMBER: _ClassVar[int]
    CLASSIFICATION_FIELD_NUMBER: _ClassVar[int]
    INTERNAL_FIELD_NUMBER: _ClassVar[int]
    EXTERNAL_FIELD_NUMBER: _ClassVar[int]
    ASSESSMENT_FIELD_NUMBER: _ClassVar[int]
    subdomain: str
    classification: FailureClass
    internal: ProbeStatus
    external: ProbeStatus
    assessment: str
    def __init__(self, subdomain: _Optional[str] = ..., classification: _Optional[_Union[FailureClass, str]] = ..., internal: _Optional[_Union[ProbeStatus, str]] = ..., external: _Optional[_Union[ProbeStatus, str]] = ..., assessment: _Optional[str] = ...) -> None: ...

class RunProbesRequest(_message.Message):
    __slots__ = ()
    def __init__(self) -> None: ...

class RunProbesResponse(_message.Message):
    __slots__ = ("results",)
    RESULTS_FIELD_NUMBER: _ClassVar[int]
    results: _containers.RepeatedCompositeFieldContainer[ProbeResult]
    def __init__(self, results: _Optional[_Iterable[_Union[ProbeResult, _Mapping]]] = ...) -> None: ...

class ListProbesRequest(_message.Message):
    __slots__ = ("subdomain", "limit")
    SUBDOMAIN_FIELD_NUMBER: _ClassVar[int]
    LIMIT_FIELD_NUMBER: _ClassVar[int]
    subdomain: str
    limit: int
    def __init__(self, subdomain: _Optional[str] = ..., limit: _Optional[int] = ...) -> None: ...

class ListProbesResponse(_message.Message):
    __slots__ = ("results",)
    RESULTS_FIELD_NUMBER: _ClassVar[int]
    results: _containers.RepeatedCompositeFieldContainer[ProbeResult]
    def __init__(self, results: _Optional[_Iterable[_Union[ProbeResult, _Mapping]]] = ...) -> None: ...

class ClassifyRequest(_message.Message):
    __slots__ = ()
    def __init__(self) -> None: ...

class ClassifyResponse(_message.Message):
    __slots__ = ("classifications",)
    CLASSIFICATIONS_FIELD_NUMBER: _ClassVar[int]
    classifications: _containers.RepeatedCompositeFieldContainer[RouteClassification]
    def __init__(self, classifications: _Optional[_Iterable[_Union[RouteClassification, _Mapping]]] = ...) -> None: ...
