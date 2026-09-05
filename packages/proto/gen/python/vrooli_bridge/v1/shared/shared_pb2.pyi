import datetime

from google.protobuf import timestamp_pb2 as _timestamp_pb2
from vrooli_bridge.v1.session import session_pb2 as _session_pb2
from google.protobuf.internal import containers as _containers
from google.protobuf.internal import enum_type_wrapper as _enum_type_wrapper
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Iterable as _Iterable, Mapping as _Mapping
from typing import ClassVar as _ClassVar, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class CompatibilityStatus(int, metaclass=_enum_type_wrapper.EnumTypeWrapper):
    __slots__ = ()
    COMPATIBILITY_STATUS_UNSPECIFIED: _ClassVar[CompatibilityStatus]
    COMPATIBILITY_STATUS_OK: _ClassVar[CompatibilityStatus]
    COMPATIBILITY_STATUS_NEEDS_UPDATE: _ClassVar[CompatibilityStatus]
    COMPATIBILITY_STATUS_INCOMPATIBLE: _ClassVar[CompatibilityStatus]

class RunEventKind(int, metaclass=_enum_type_wrapper.EnumTypeWrapper):
    __slots__ = ()
    RUN_EVENT_KIND_UNSPECIFIED: _ClassVar[RunEventKind]
    RUN_EVENT_KIND_LOG: _ClassVar[RunEventKind]
    RUN_EVENT_KIND_STATUS: _ClassVar[RunEventKind]
    RUN_EVENT_KIND_EXIT: _ClassVar[RunEventKind]
    RUN_EVENT_KIND_ARTIFACT_REF: _ClassVar[RunEventKind]

class CapabilityObservationState(int, metaclass=_enum_type_wrapper.EnumTypeWrapper):
    __slots__ = ()
    CAPABILITY_OBSERVATION_STATE_UNSPECIFIED: _ClassVar[CapabilityObservationState]
    CAPABILITY_OBSERVATION_STATE_READY: _ClassVar[CapabilityObservationState]
    CAPABILITY_OBSERVATION_STATE_MISSING: _ClassVar[CapabilityObservationState]
    CAPABILITY_OBSERVATION_STATE_NOT_APPLICABLE: _ClassVar[CapabilityObservationState]
    CAPABILITY_OBSERVATION_STATE_UNKNOWN: _ClassVar[CapabilityObservationState]

class RelayResponseKind(int, metaclass=_enum_type_wrapper.EnumTypeWrapper):
    __slots__ = ()
    RELAY_RESPONSE_KIND_UNSPECIFIED: _ClassVar[RelayResponseKind]
    RELAY_RESPONSE_KIND_ACCEPTED: _ClassVar[RelayResponseKind]
    RELAY_RESPONSE_KIND_DATA: _ClassVar[RelayResponseKind]
    RELAY_RESPONSE_KIND_COMPLETED: _ClassVar[RelayResponseKind]
    RELAY_RESPONSE_KIND_FAILED: _ClassVar[RelayResponseKind]
    RELAY_RESPONSE_KIND_TERMINATED: _ClassVar[RelayResponseKind]
COMPATIBILITY_STATUS_UNSPECIFIED: CompatibilityStatus
COMPATIBILITY_STATUS_OK: CompatibilityStatus
COMPATIBILITY_STATUS_NEEDS_UPDATE: CompatibilityStatus
COMPATIBILITY_STATUS_INCOMPATIBLE: CompatibilityStatus
RUN_EVENT_KIND_UNSPECIFIED: RunEventKind
RUN_EVENT_KIND_LOG: RunEventKind
RUN_EVENT_KIND_STATUS: RunEventKind
RUN_EVENT_KIND_EXIT: RunEventKind
RUN_EVENT_KIND_ARTIFACT_REF: RunEventKind
CAPABILITY_OBSERVATION_STATE_UNSPECIFIED: CapabilityObservationState
CAPABILITY_OBSERVATION_STATE_READY: CapabilityObservationState
CAPABILITY_OBSERVATION_STATE_MISSING: CapabilityObservationState
CAPABILITY_OBSERVATION_STATE_NOT_APPLICABLE: CapabilityObservationState
CAPABILITY_OBSERVATION_STATE_UNKNOWN: CapabilityObservationState
RELAY_RESPONSE_KIND_UNSPECIFIED: RelayResponseKind
RELAY_RESPONSE_KIND_ACCEPTED: RelayResponseKind
RELAY_RESPONSE_KIND_DATA: RelayResponseKind
RELAY_RESPONSE_KIND_COMPLETED: RelayResponseKind
RELAY_RESPONSE_KIND_FAILED: RelayResponseKind
RELAY_RESPONSE_KIND_TERMINATED: RelayResponseKind

class HealthSnapshot(_message.Message):
    __slots__ = ("toolchain_present", "disk_headroom_bytes", "container_runtime_up", "details", "reported_at", "capabilities")
    class DetailsEntry(_message.Message):
        __slots__ = ("key", "value")
        KEY_FIELD_NUMBER: _ClassVar[int]
        VALUE_FIELD_NUMBER: _ClassVar[int]
        key: str
        value: str
        def __init__(self, key: _Optional[str] = ..., value: _Optional[str] = ...) -> None: ...
    TOOLCHAIN_PRESENT_FIELD_NUMBER: _ClassVar[int]
    DISK_HEADROOM_BYTES_FIELD_NUMBER: _ClassVar[int]
    CONTAINER_RUNTIME_UP_FIELD_NUMBER: _ClassVar[int]
    DETAILS_FIELD_NUMBER: _ClassVar[int]
    REPORTED_AT_FIELD_NUMBER: _ClassVar[int]
    CAPABILITIES_FIELD_NUMBER: _ClassVar[int]
    toolchain_present: bool
    disk_headroom_bytes: int
    container_runtime_up: bool
    details: _containers.ScalarMap[str, str]
    reported_at: _timestamp_pb2.Timestamp
    capabilities: _containers.RepeatedCompositeFieldContainer[CapabilityObservation]
    def __init__(self, toolchain_present: _Optional[bool] = ..., disk_headroom_bytes: _Optional[int] = ..., container_runtime_up: _Optional[bool] = ..., details: _Optional[_Mapping[str, str]] = ..., reported_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., capabilities: _Optional[_Iterable[_Union[CapabilityObservation, _Mapping]]] = ...) -> None: ...

class CapabilityObservation(_message.Message):
    __slots__ = ("capability", "id", "label", "state", "path", "version", "probed_at", "detail")
    CAPABILITY_FIELD_NUMBER: _ClassVar[int]
    ID_FIELD_NUMBER: _ClassVar[int]
    LABEL_FIELD_NUMBER: _ClassVar[int]
    STATE_FIELD_NUMBER: _ClassVar[int]
    PATH_FIELD_NUMBER: _ClassVar[int]
    VERSION_FIELD_NUMBER: _ClassVar[int]
    PROBED_AT_FIELD_NUMBER: _ClassVar[int]
    DETAIL_FIELD_NUMBER: _ClassVar[int]
    capability: str
    id: str
    label: str
    state: CapabilityObservationState
    path: str
    version: str
    probed_at: _timestamp_pb2.Timestamp
    detail: str
    def __init__(self, capability: _Optional[str] = ..., id: _Optional[str] = ..., label: _Optional[str] = ..., state: _Optional[_Union[CapabilityObservationState, str]] = ..., path: _Optional[str] = ..., version: _Optional[str] = ..., probed_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., detail: _Optional[str] = ...) -> None: ...

class Heartbeat(_message.Message):
    __slots__ = ("node_id", "sequence", "health", "sent_at", "rejected_credential_pushes")
    NODE_ID_FIELD_NUMBER: _ClassVar[int]
    SEQUENCE_FIELD_NUMBER: _ClassVar[int]
    HEALTH_FIELD_NUMBER: _ClassVar[int]
    SENT_AT_FIELD_NUMBER: _ClassVar[int]
    REJECTED_CREDENTIAL_PUSHES_FIELD_NUMBER: _ClassVar[int]
    node_id: str
    sequence: int
    health: HealthSnapshot
    sent_at: _timestamp_pb2.Timestamp
    rejected_credential_pushes: int
    def __init__(self, node_id: _Optional[str] = ..., sequence: _Optional[int] = ..., health: _Optional[_Union[HealthSnapshot, _Mapping]] = ..., sent_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., rejected_credential_pushes: _Optional[int] = ...) -> None: ...

class RunEvent(_message.Message):
    __slots__ = ("run_id", "kind", "sequence", "log_chunk", "status", "exit_code", "artifact_ref", "emitted_at")
    RUN_ID_FIELD_NUMBER: _ClassVar[int]
    KIND_FIELD_NUMBER: _ClassVar[int]
    SEQUENCE_FIELD_NUMBER: _ClassVar[int]
    LOG_CHUNK_FIELD_NUMBER: _ClassVar[int]
    STATUS_FIELD_NUMBER: _ClassVar[int]
    EXIT_CODE_FIELD_NUMBER: _ClassVar[int]
    ARTIFACT_REF_FIELD_NUMBER: _ClassVar[int]
    EMITTED_AT_FIELD_NUMBER: _ClassVar[int]
    run_id: str
    kind: RunEventKind
    sequence: int
    log_chunk: str
    status: str
    exit_code: int
    artifact_ref: str
    emitted_at: _timestamp_pb2.Timestamp
    def __init__(self, run_id: _Optional[str] = ..., kind: _Optional[_Union[RunEventKind, str]] = ..., sequence: _Optional[int] = ..., log_chunk: _Optional[str] = ..., status: _Optional[str] = ..., exit_code: _Optional[int] = ..., artifact_ref: _Optional[str] = ..., emitted_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ...) -> None: ...

class DeliveryAck(_message.Message):
    __slots__ = ("frame_id", "run_id", "op_id", "received_at")
    FRAME_ID_FIELD_NUMBER: _ClassVar[int]
    RUN_ID_FIELD_NUMBER: _ClassVar[int]
    OP_ID_FIELD_NUMBER: _ClassVar[int]
    RECEIVED_AT_FIELD_NUMBER: _ClassVar[int]
    frame_id: str
    run_id: str
    op_id: str
    received_at: _timestamp_pb2.Timestamp
    def __init__(self, frame_id: _Optional[str] = ..., run_id: _Optional[str] = ..., op_id: _Optional[str] = ..., received_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ...) -> None: ...

class RelayResponse(_message.Message):
    __slots__ = ("correlation_id", "kind", "sequence", "data", "reason", "exit_code", "total_bytes")
    CORRELATION_ID_FIELD_NUMBER: _ClassVar[int]
    KIND_FIELD_NUMBER: _ClassVar[int]
    SEQUENCE_FIELD_NUMBER: _ClassVar[int]
    DATA_FIELD_NUMBER: _ClassVar[int]
    REASON_FIELD_NUMBER: _ClassVar[int]
    EXIT_CODE_FIELD_NUMBER: _ClassVar[int]
    TOTAL_BYTES_FIELD_NUMBER: _ClassVar[int]
    correlation_id: str
    kind: RelayResponseKind
    sequence: int
    data: bytes
    reason: str
    exit_code: int
    total_bytes: int
    def __init__(self, correlation_id: _Optional[str] = ..., kind: _Optional[_Union[RelayResponseKind, str]] = ..., sequence: _Optional[int] = ..., data: _Optional[bytes] = ..., reason: _Optional[str] = ..., exit_code: _Optional[int] = ..., total_bytes: _Optional[int] = ...) -> None: ...

class SessionFrame(_message.Message):
    __slots__ = ("session_id", "frame")
    SESSION_ID_FIELD_NUMBER: _ClassVar[int]
    FRAME_FIELD_NUMBER: _ClassVar[int]
    session_id: str
    frame: _session_pb2.Frame
    def __init__(self, session_id: _Optional[str] = ..., frame: _Optional[_Union[_session_pb2.Frame, _Mapping]] = ...) -> None: ...
