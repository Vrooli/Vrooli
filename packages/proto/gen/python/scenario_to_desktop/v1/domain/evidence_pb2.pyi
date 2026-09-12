import datetime

from buf.validate import validate_pb2 as _validate_pb2
from google.protobuf import empty_pb2 as _empty_pb2
from google.protobuf import struct_pb2 as _struct_pb2
from google.protobuf import timestamp_pb2 as _timestamp_pb2
from scenario_to_desktop.v1.shared import common_pb2 as _common_pb2
from google.protobuf.internal import containers as _containers
from google.protobuf.internal import enum_type_wrapper as _enum_type_wrapper
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Iterable as _Iterable, Mapping as _Mapping
from typing import ClassVar as _ClassVar, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class DesktopSessionState(int, metaclass=_enum_type_wrapper.EnumTypeWrapper):
    __slots__ = ()
    DESKTOP_SESSION_STATE_UNSPECIFIED: _ClassVar[DesktopSessionState]
    DESKTOP_SESSION_STATE_CREATING: _ClassVar[DesktopSessionState]
    DESKTOP_SESSION_STATE_RUNNING: _ClassVar[DesktopSessionState]
    DESKTOP_SESSION_STATE_STOPPING: _ClassVar[DesktopSessionState]
    DESKTOP_SESSION_STATE_STOPPED: _ClassVar[DesktopSessionState]
    DESKTOP_SESSION_STATE_ERROR: _ClassVar[DesktopSessionState]

class DesktopNetworkMode(int, metaclass=_enum_type_wrapper.EnumTypeWrapper):
    __slots__ = ()
    DESKTOP_NETWORK_MODE_UNSPECIFIED: _ClassVar[DesktopNetworkMode]
    DESKTOP_NETWORK_MODE_NORMAL: _ClassVar[DesktopNetworkMode]
    DESKTOP_NETWORK_MODE_OFFLINE: _ClassVar[DesktopNetworkMode]
    DESKTOP_NETWORK_MODE_SLOW: _ClassVar[DesktopNetworkMode]
DESKTOP_SESSION_STATE_UNSPECIFIED: DesktopSessionState
DESKTOP_SESSION_STATE_CREATING: DesktopSessionState
DESKTOP_SESSION_STATE_RUNNING: DesktopSessionState
DESKTOP_SESSION_STATE_STOPPING: DesktopSessionState
DESKTOP_SESSION_STATE_STOPPED: DesktopSessionState
DESKTOP_SESSION_STATE_ERROR: DesktopSessionState
DESKTOP_NETWORK_MODE_UNSPECIFIED: DesktopNetworkMode
DESKTOP_NETWORK_MODE_NORMAL: DesktopNetworkMode
DESKTOP_NETWORK_MODE_OFFLINE: DesktopNetworkMode
DESKTOP_NETWORK_MODE_SLOW: DesktopNetworkMode

class EvidenceTarget(_message.Message):
    __slots__ = ("kind", "bridge_node_id", "bridge_job_id")
    class Kind(int, metaclass=_enum_type_wrapper.EnumTypeWrapper):
        __slots__ = ()
        KIND_UNSPECIFIED: _ClassVar[EvidenceTarget.Kind]
        KIND_LOCAL: _ClassVar[EvidenceTarget.Kind]
        KIND_BRIDGE_NODE: _ClassVar[EvidenceTarget.Kind]
    KIND_UNSPECIFIED: EvidenceTarget.Kind
    KIND_LOCAL: EvidenceTarget.Kind
    KIND_BRIDGE_NODE: EvidenceTarget.Kind
    KIND_FIELD_NUMBER: _ClassVar[int]
    BRIDGE_NODE_ID_FIELD_NUMBER: _ClassVar[int]
    BRIDGE_JOB_ID_FIELD_NUMBER: _ClassVar[int]
    kind: EvidenceTarget.Kind
    bridge_node_id: str
    bridge_job_id: str
    def __init__(self, kind: _Optional[_Union[EvidenceTarget.Kind, str]] = ..., bridge_node_id: _Optional[str] = ..., bridge_job_id: _Optional[str] = ...) -> None: ...

class DesktopSessionRequest(_message.Message):
    __slots__ = ("scenario_name", "artifact_path", "platform", "width", "height", "target")
    SCENARIO_NAME_FIELD_NUMBER: _ClassVar[int]
    ARTIFACT_PATH_FIELD_NUMBER: _ClassVar[int]
    PLATFORM_FIELD_NUMBER: _ClassVar[int]
    WIDTH_FIELD_NUMBER: _ClassVar[int]
    HEIGHT_FIELD_NUMBER: _ClassVar[int]
    TARGET_FIELD_NUMBER: _ClassVar[int]
    scenario_name: str
    artifact_path: str
    platform: _common_pb2.Platform
    width: int
    height: int
    target: EvidenceTarget
    def __init__(self, scenario_name: _Optional[str] = ..., artifact_path: _Optional[str] = ..., platform: _Optional[_Union[_common_pb2.Platform, str]] = ..., width: _Optional[int] = ..., height: _Optional[int] = ..., target: _Optional[_Union[EvidenceTarget, _Mapping]] = ...) -> None: ...

class DesktopSessionRef(_message.Message):
    __slots__ = ("session_id",)
    SESSION_ID_FIELD_NUMBER: _ClassVar[int]
    session_id: str
    def __init__(self, session_id: _Optional[str] = ...) -> None: ...

class DesktopSessionMetrics(_message.Message):
    __slots__ = ("splash_duration_ms", "splash_detected", "ready_duration_ms", "ready_detected", "current_cpu_percent", "current_rss_mb", "peak_rss_mb", "sample_count", "performance_status", "performance_reason", "protocol_startup_duration_ms", "demo_startup_duration_ms", "protocol_trace_ref", "demo_trace_ref", "process_roles")
    SPLASH_DURATION_MS_FIELD_NUMBER: _ClassVar[int]
    SPLASH_DETECTED_FIELD_NUMBER: _ClassVar[int]
    READY_DURATION_MS_FIELD_NUMBER: _ClassVar[int]
    READY_DETECTED_FIELD_NUMBER: _ClassVar[int]
    CURRENT_CPU_PERCENT_FIELD_NUMBER: _ClassVar[int]
    CURRENT_RSS_MB_FIELD_NUMBER: _ClassVar[int]
    PEAK_RSS_MB_FIELD_NUMBER: _ClassVar[int]
    SAMPLE_COUNT_FIELD_NUMBER: _ClassVar[int]
    PERFORMANCE_STATUS_FIELD_NUMBER: _ClassVar[int]
    PERFORMANCE_REASON_FIELD_NUMBER: _ClassVar[int]
    PROTOCOL_STARTUP_DURATION_MS_FIELD_NUMBER: _ClassVar[int]
    DEMO_STARTUP_DURATION_MS_FIELD_NUMBER: _ClassVar[int]
    PROTOCOL_TRACE_REF_FIELD_NUMBER: _ClassVar[int]
    DEMO_TRACE_REF_FIELD_NUMBER: _ClassVar[int]
    PROCESS_ROLES_FIELD_NUMBER: _ClassVar[int]
    splash_duration_ms: int
    splash_detected: bool
    ready_duration_ms: int
    ready_detected: bool
    current_cpu_percent: float
    current_rss_mb: float
    peak_rss_mb: float
    sample_count: int
    performance_status: str
    performance_reason: str
    protocol_startup_duration_ms: int
    demo_startup_duration_ms: int
    protocol_trace_ref: str
    demo_trace_ref: str
    process_roles: _containers.RepeatedCompositeFieldContainer[DesktopProcessRoleMetric]
    def __init__(self, splash_duration_ms: _Optional[int] = ..., splash_detected: _Optional[bool] = ..., ready_duration_ms: _Optional[int] = ..., ready_detected: _Optional[bool] = ..., current_cpu_percent: _Optional[float] = ..., current_rss_mb: _Optional[float] = ..., peak_rss_mb: _Optional[float] = ..., sample_count: _Optional[int] = ..., performance_status: _Optional[str] = ..., performance_reason: _Optional[str] = ..., protocol_startup_duration_ms: _Optional[int] = ..., demo_startup_duration_ms: _Optional[int] = ..., protocol_trace_ref: _Optional[str] = ..., demo_trace_ref: _Optional[str] = ..., process_roles: _Optional[_Iterable[_Union[DesktopProcessRoleMetric, _Mapping]]] = ...) -> None: ...

class DesktopProcessRoleMetric(_message.Message):
    __slots__ = ("role", "available", "unsupported", "process_count", "cpu_percent", "peak_cpu_percent", "rss_bytes", "peak_rss_bytes", "threads", "duration_ms", "sample_count")
    ROLE_FIELD_NUMBER: _ClassVar[int]
    AVAILABLE_FIELD_NUMBER: _ClassVar[int]
    UNSUPPORTED_FIELD_NUMBER: _ClassVar[int]
    PROCESS_COUNT_FIELD_NUMBER: _ClassVar[int]
    CPU_PERCENT_FIELD_NUMBER: _ClassVar[int]
    PEAK_CPU_PERCENT_FIELD_NUMBER: _ClassVar[int]
    RSS_BYTES_FIELD_NUMBER: _ClassVar[int]
    PEAK_RSS_BYTES_FIELD_NUMBER: _ClassVar[int]
    THREADS_FIELD_NUMBER: _ClassVar[int]
    DURATION_MS_FIELD_NUMBER: _ClassVar[int]
    SAMPLE_COUNT_FIELD_NUMBER: _ClassVar[int]
    role: str
    available: bool
    unsupported: bool
    process_count: int
    cpu_percent: float
    peak_cpu_percent: float
    rss_bytes: int
    peak_rss_bytes: int
    threads: int
    duration_ms: int
    sample_count: int
    def __init__(self, role: _Optional[str] = ..., available: _Optional[bool] = ..., unsupported: _Optional[bool] = ..., process_count: _Optional[int] = ..., cpu_percent: _Optional[float] = ..., peak_cpu_percent: _Optional[float] = ..., rss_bytes: _Optional[int] = ..., peak_rss_bytes: _Optional[int] = ..., threads: _Optional[int] = ..., duration_ms: _Optional[int] = ..., sample_count: _Optional[int] = ...) -> None: ...

class DesktopSession(_message.Message):
    __slots__ = ("session_id", "scenario_name", "platform", "state", "width", "height", "created_at", "last_heartbeat_at", "error", "app_running", "target", "vnc_port", "websocket_port", "recording", "network_mode", "bandwidth_kbps", "dark_mode", "locale", "metrics")
    SESSION_ID_FIELD_NUMBER: _ClassVar[int]
    SCENARIO_NAME_FIELD_NUMBER: _ClassVar[int]
    PLATFORM_FIELD_NUMBER: _ClassVar[int]
    STATE_FIELD_NUMBER: _ClassVar[int]
    WIDTH_FIELD_NUMBER: _ClassVar[int]
    HEIGHT_FIELD_NUMBER: _ClassVar[int]
    CREATED_AT_FIELD_NUMBER: _ClassVar[int]
    LAST_HEARTBEAT_AT_FIELD_NUMBER: _ClassVar[int]
    ERROR_FIELD_NUMBER: _ClassVar[int]
    APP_RUNNING_FIELD_NUMBER: _ClassVar[int]
    TARGET_FIELD_NUMBER: _ClassVar[int]
    VNC_PORT_FIELD_NUMBER: _ClassVar[int]
    WEBSOCKET_PORT_FIELD_NUMBER: _ClassVar[int]
    RECORDING_FIELD_NUMBER: _ClassVar[int]
    NETWORK_MODE_FIELD_NUMBER: _ClassVar[int]
    BANDWIDTH_KBPS_FIELD_NUMBER: _ClassVar[int]
    DARK_MODE_FIELD_NUMBER: _ClassVar[int]
    LOCALE_FIELD_NUMBER: _ClassVar[int]
    METRICS_FIELD_NUMBER: _ClassVar[int]
    session_id: str
    scenario_name: str
    platform: _common_pb2.Platform
    state: DesktopSessionState
    width: int
    height: int
    created_at: _timestamp_pb2.Timestamp
    last_heartbeat_at: _timestamp_pb2.Timestamp
    error: str
    app_running: bool
    target: EvidenceTarget
    vnc_port: int
    websocket_port: int
    recording: bool
    network_mode: DesktopNetworkMode
    bandwidth_kbps: int
    dark_mode: bool
    locale: str
    metrics: DesktopSessionMetrics
    def __init__(self, session_id: _Optional[str] = ..., scenario_name: _Optional[str] = ..., platform: _Optional[_Union[_common_pb2.Platform, str]] = ..., state: _Optional[_Union[DesktopSessionState, str]] = ..., width: _Optional[int] = ..., height: _Optional[int] = ..., created_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., last_heartbeat_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., error: _Optional[str] = ..., app_running: _Optional[bool] = ..., target: _Optional[_Union[EvidenceTarget, _Mapping]] = ..., vnc_port: _Optional[int] = ..., websocket_port: _Optional[int] = ..., recording: _Optional[bool] = ..., network_mode: _Optional[_Union[DesktopNetworkMode, str]] = ..., bandwidth_kbps: _Optional[int] = ..., dark_mode: _Optional[bool] = ..., locale: _Optional[str] = ..., metrics: _Optional[_Union[DesktopSessionMetrics, _Mapping]] = ...) -> None: ...

class ListDesktopSessionsRequest(_message.Message):
    __slots__ = ("scenario_name",)
    SCENARIO_NAME_FIELD_NUMBER: _ClassVar[int]
    scenario_name: str
    def __init__(self, scenario_name: _Optional[str] = ...) -> None: ...

class ListDesktopSessionsResponse(_message.Message):
    __slots__ = ("sessions",)
    SESSIONS_FIELD_NUMBER: _ClassVar[int]
    sessions: _containers.RepeatedCompositeFieldContainer[DesktopSession]
    def __init__(self, sessions: _Optional[_Iterable[_Union[DesktopSession, _Mapping]]] = ...) -> None: ...

class LaunchDesktopArtifactRequest(_message.Message):
    __slots__ = ("session_id", "artifact_path")
    SESSION_ID_FIELD_NUMBER: _ClassVar[int]
    ARTIFACT_PATH_FIELD_NUMBER: _ClassVar[int]
    session_id: str
    artifact_path: str
    def __init__(self, session_id: _Optional[str] = ..., artifact_path: _Optional[str] = ...) -> None: ...

class FindDesktopArtifactRequest(_message.Message):
    __slots__ = ("scenario_name",)
    SCENARIO_NAME_FIELD_NUMBER: _ClassVar[int]
    scenario_name: str
    def __init__(self, scenario_name: _Optional[str] = ...) -> None: ...

class FindDesktopArtifactResponse(_message.Message):
    __slots__ = ("artifact_path",)
    ARTIFACT_PATH_FIELD_NUMBER: _ClassVar[int]
    artifact_path: str
    def __init__(self, artifact_path: _Optional[str] = ...) -> None: ...

class CaptureScreenshotRequest(_message.Message):
    __slots__ = ("session_id",)
    SESSION_ID_FIELD_NUMBER: _ClassVar[int]
    session_id: str
    def __init__(self, session_id: _Optional[str] = ...) -> None: ...

class EvidenceCapture(_message.Message):
    __slots__ = ("capture_id", "scenario_name", "kind", "source_session_id", "filename", "file_size_bytes", "width", "height", "duration_ms", "created_at", "checksum")
    CAPTURE_ID_FIELD_NUMBER: _ClassVar[int]
    SCENARIO_NAME_FIELD_NUMBER: _ClassVar[int]
    KIND_FIELD_NUMBER: _ClassVar[int]
    SOURCE_SESSION_ID_FIELD_NUMBER: _ClassVar[int]
    FILENAME_FIELD_NUMBER: _ClassVar[int]
    FILE_SIZE_BYTES_FIELD_NUMBER: _ClassVar[int]
    WIDTH_FIELD_NUMBER: _ClassVar[int]
    HEIGHT_FIELD_NUMBER: _ClassVar[int]
    DURATION_MS_FIELD_NUMBER: _ClassVar[int]
    CREATED_AT_FIELD_NUMBER: _ClassVar[int]
    CHECKSUM_FIELD_NUMBER: _ClassVar[int]
    capture_id: str
    scenario_name: str
    kind: str
    source_session_id: str
    filename: str
    file_size_bytes: int
    width: int
    height: int
    duration_ms: int
    created_at: _timestamp_pb2.Timestamp
    checksum: str
    def __init__(self, capture_id: _Optional[str] = ..., scenario_name: _Optional[str] = ..., kind: _Optional[str] = ..., source_session_id: _Optional[str] = ..., filename: _Optional[str] = ..., file_size_bytes: _Optional[int] = ..., width: _Optional[int] = ..., height: _Optional[int] = ..., duration_ms: _Optional[int] = ..., created_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., checksum: _Optional[str] = ...) -> None: ...

class CaptureScreenshotResponse(_message.Message):
    __slots__ = ("capture",)
    CAPTURE_FIELD_NUMBER: _ClassVar[int]
    capture: EvidenceCapture
    def __init__(self, capture: _Optional[_Union[EvidenceCapture, _Mapping]] = ...) -> None: ...

class ListEvidenceCapturesRequest(_message.Message):
    __slots__ = ("scenario_name",)
    SCENARIO_NAME_FIELD_NUMBER: _ClassVar[int]
    scenario_name: str
    def __init__(self, scenario_name: _Optional[str] = ...) -> None: ...

class ListEvidenceCapturesResponse(_message.Message):
    __slots__ = ("captures",)
    CAPTURES_FIELD_NUMBER: _ClassVar[int]
    captures: _containers.RepeatedCompositeFieldContainer[EvidenceCapture]
    def __init__(self, captures: _Optional[_Iterable[_Union[EvidenceCapture, _Mapping]]] = ...) -> None: ...

class GetEvidenceCaptureRequest(_message.Message):
    __slots__ = ("scenario_name", "capture_id")
    SCENARIO_NAME_FIELD_NUMBER: _ClassVar[int]
    CAPTURE_ID_FIELD_NUMBER: _ClassVar[int]
    scenario_name: str
    capture_id: str
    def __init__(self, scenario_name: _Optional[str] = ..., capture_id: _Optional[str] = ...) -> None: ...

class GetEvidenceCaptureResponse(_message.Message):
    __slots__ = ("capture", "content")
    CAPTURE_FIELD_NUMBER: _ClassVar[int]
    CONTENT_FIELD_NUMBER: _ClassVar[int]
    capture: EvidenceCapture
    content: bytes
    def __init__(self, capture: _Optional[_Union[EvidenceCapture, _Mapping]] = ..., content: _Optional[bytes] = ...) -> None: ...

class EvidenceCapturesSummary(_message.Message):
    __slots__ = ("count", "total_bytes")
    COUNT_FIELD_NUMBER: _ClassVar[int]
    TOTAL_BYTES_FIELD_NUMBER: _ClassVar[int]
    count: int
    total_bytes: int
    def __init__(self, count: _Optional[int] = ..., total_bytes: _Optional[int] = ...) -> None: ...

class EvidenceCaptureRef(_message.Message):
    __slots__ = ("scenario_name", "capture_id")
    SCENARIO_NAME_FIELD_NUMBER: _ClassVar[int]
    CAPTURE_ID_FIELD_NUMBER: _ClassVar[int]
    scenario_name: str
    capture_id: str
    def __init__(self, scenario_name: _Optional[str] = ..., capture_id: _Optional[str] = ...) -> None: ...

class DesktopControlRequest(_message.Message):
    __slots__ = ("session_id", "action", "params")
    SESSION_ID_FIELD_NUMBER: _ClassVar[int]
    ACTION_FIELD_NUMBER: _ClassVar[int]
    PARAMS_FIELD_NUMBER: _ClassVar[int]
    session_id: str
    action: str
    params: _struct_pb2.Struct
    def __init__(self, session_id: _Optional[str] = ..., action: _Optional[str] = ..., params: _Optional[_Union[_struct_pb2.Struct, _Mapping]] = ...) -> None: ...

class DesktopControlResponse(_message.Message):
    __slots__ = ("result",)
    RESULT_FIELD_NUMBER: _ClassVar[int]
    result: _struct_pb2.Struct
    def __init__(self, result: _Optional[_Union[_struct_pb2.Struct, _Mapping]] = ...) -> None: ...
