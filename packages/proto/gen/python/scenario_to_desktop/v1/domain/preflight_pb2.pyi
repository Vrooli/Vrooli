import datetime

from google.protobuf import timestamp_pb2 as _timestamp_pb2
from scenario_to_desktop.v1.base import shared_pb2 as _shared_pb2
from google.protobuf.internal import containers as _containers
from google.protobuf.internal import enum_type_wrapper as _enum_type_wrapper
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Iterable as _Iterable, Mapping as _Mapping
from typing import ClassVar as _ClassVar, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class PreflightStatus(int, metaclass=_enum_type_wrapper.EnumTypeWrapper):
    __slots__ = ()
    PREFLIGHT_STATUS_UNSPECIFIED: _ClassVar[PreflightStatus]
    PREFLIGHT_STATUS_RUNNING: _ClassVar[PreflightStatus]
    PREFLIGHT_STATUS_PASSED: _ClassVar[PreflightStatus]
    PREFLIGHT_STATUS_FAILED: _ClassVar[PreflightStatus]
    PREFLIGHT_STATUS_WARNINGS: _ClassVar[PreflightStatus]

class CheckStatus(int, metaclass=_enum_type_wrapper.EnumTypeWrapper):
    __slots__ = ()
    CHECK_STATUS_UNSPECIFIED: _ClassVar[CheckStatus]
    CHECK_STATUS_PENDING: _ClassVar[CheckStatus]
    CHECK_STATUS_RUNNING: _ClassVar[CheckStatus]
    CHECK_STATUS_PASSED: _ClassVar[CheckStatus]
    CHECK_STATUS_FAILED: _ClassVar[CheckStatus]
    CHECK_STATUS_SKIPPED: _ClassVar[CheckStatus]

class SecretClass(int, metaclass=_enum_type_wrapper.EnumTypeWrapper):
    __slots__ = ()
    SECRET_CLASS_UNSPECIFIED: _ClassVar[SecretClass]
    SECRET_CLASS_API_KEY: _ClassVar[SecretClass]
    SECRET_CLASS_PASSWORD: _ClassVar[SecretClass]
    SECRET_CLASS_TOKEN: _ClassVar[SecretClass]
    SECRET_CLASS_CONNECTION_STRING: _ClassVar[SecretClass]
    SECRET_CLASS_CERTIFICATE: _ClassVar[SecretClass]
    SECRET_CLASS_GENERIC: _ClassVar[SecretClass]

class JobStatus(int, metaclass=_enum_type_wrapper.EnumTypeWrapper):
    __slots__ = ()
    JOB_STATUS_UNSPECIFIED: _ClassVar[JobStatus]
    JOB_STATUS_PENDING: _ClassVar[JobStatus]
    JOB_STATUS_RUNNING: _ClassVar[JobStatus]
    JOB_STATUS_COMPLETED: _ClassVar[JobStatus]
    JOB_STATUS_FAILED: _ClassVar[JobStatus]
PREFLIGHT_STATUS_UNSPECIFIED: PreflightStatus
PREFLIGHT_STATUS_RUNNING: PreflightStatus
PREFLIGHT_STATUS_PASSED: PreflightStatus
PREFLIGHT_STATUS_FAILED: PreflightStatus
PREFLIGHT_STATUS_WARNINGS: PreflightStatus
CHECK_STATUS_UNSPECIFIED: CheckStatus
CHECK_STATUS_PENDING: CheckStatus
CHECK_STATUS_RUNNING: CheckStatus
CHECK_STATUS_PASSED: CheckStatus
CHECK_STATUS_FAILED: CheckStatus
CHECK_STATUS_SKIPPED: CheckStatus
SECRET_CLASS_UNSPECIFIED: SecretClass
SECRET_CLASS_API_KEY: SecretClass
SECRET_CLASS_PASSWORD: SecretClass
SECRET_CLASS_TOKEN: SecretClass
SECRET_CLASS_CONNECTION_STRING: SecretClass
SECRET_CLASS_CERTIFICATE: SecretClass
SECRET_CLASS_GENERIC: SecretClass
JOB_STATUS_UNSPECIFIED: JobStatus
JOB_STATUS_PENDING: JobStatus
JOB_STATUS_RUNNING: JobStatus
JOB_STATUS_COMPLETED: JobStatus
JOB_STATUS_FAILED: JobStatus

class PreflightRequest(_message.Message):
    __slots__ = ("bundle_manifest_path", "bundle_root", "secrets", "timeout_seconds", "start_services", "log_tail_lines", "status_only", "session_id", "session_ttl", "session_stop")
    class SecretsEntry(_message.Message):
        __slots__ = ("key", "value")
        KEY_FIELD_NUMBER: _ClassVar[int]
        VALUE_FIELD_NUMBER: _ClassVar[int]
        key: str
        value: str
        def __init__(self, key: _Optional[str] = ..., value: _Optional[str] = ...) -> None: ...
    BUNDLE_MANIFEST_PATH_FIELD_NUMBER: _ClassVar[int]
    BUNDLE_ROOT_FIELD_NUMBER: _ClassVar[int]
    SECRETS_FIELD_NUMBER: _ClassVar[int]
    TIMEOUT_SECONDS_FIELD_NUMBER: _ClassVar[int]
    START_SERVICES_FIELD_NUMBER: _ClassVar[int]
    LOG_TAIL_LINES_FIELD_NUMBER: _ClassVar[int]
    STATUS_ONLY_FIELD_NUMBER: _ClassVar[int]
    SESSION_ID_FIELD_NUMBER: _ClassVar[int]
    SESSION_TTL_FIELD_NUMBER: _ClassVar[int]
    SESSION_STOP_FIELD_NUMBER: _ClassVar[int]
    bundle_manifest_path: str
    bundle_root: str
    secrets: _containers.ScalarMap[str, str]
    timeout_seconds: int
    start_services: bool
    log_tail_lines: int
    status_only: bool
    session_id: str
    session_ttl: int
    session_stop: bool
    def __init__(self, bundle_manifest_path: _Optional[str] = ..., bundle_root: _Optional[str] = ..., secrets: _Optional[_Mapping[str, str]] = ..., timeout_seconds: _Optional[int] = ..., start_services: _Optional[bool] = ..., log_tail_lines: _Optional[int] = ..., status_only: _Optional[bool] = ..., session_id: _Optional[str] = ..., session_ttl: _Optional[int] = ..., session_stop: _Optional[bool] = ...) -> None: ...

class PreflightSecret(_message.Message):
    __slots__ = ("id", "secret_class", "required", "has_value", "description", "format", "prompt")
    class PromptEntry(_message.Message):
        __slots__ = ("key", "value")
        KEY_FIELD_NUMBER: _ClassVar[int]
        VALUE_FIELD_NUMBER: _ClassVar[int]
        key: str
        value: str
        def __init__(self, key: _Optional[str] = ..., value: _Optional[str] = ...) -> None: ...
    ID_FIELD_NUMBER: _ClassVar[int]
    SECRET_CLASS_FIELD_NUMBER: _ClassVar[int]
    REQUIRED_FIELD_NUMBER: _ClassVar[int]
    HAS_VALUE_FIELD_NUMBER: _ClassVar[int]
    DESCRIPTION_FIELD_NUMBER: _ClassVar[int]
    FORMAT_FIELD_NUMBER: _ClassVar[int]
    PROMPT_FIELD_NUMBER: _ClassVar[int]
    id: str
    secret_class: SecretClass
    required: bool
    has_value: bool
    description: str
    format: str
    prompt: _containers.ScalarMap[str, str]
    def __init__(self, id: _Optional[str] = ..., secret_class: _Optional[_Union[SecretClass, str]] = ..., required: _Optional[bool] = ..., has_value: _Optional[bool] = ..., description: _Optional[str] = ..., format: _Optional[str] = ..., prompt: _Optional[_Mapping[str, str]] = ...) -> None: ...

class GPUInfo(_message.Message):
    __slots__ = ("available", "method", "reason", "requirements")
    class RequirementsEntry(_message.Message):
        __slots__ = ("key", "value")
        KEY_FIELD_NUMBER: _ClassVar[int]
        VALUE_FIELD_NUMBER: _ClassVar[int]
        key: str
        value: str
        def __init__(self, key: _Optional[str] = ..., value: _Optional[str] = ...) -> None: ...
    AVAILABLE_FIELD_NUMBER: _ClassVar[int]
    METHOD_FIELD_NUMBER: _ClassVar[int]
    REASON_FIELD_NUMBER: _ClassVar[int]
    REQUIREMENTS_FIELD_NUMBER: _ClassVar[int]
    available: bool
    method: str
    reason: str
    requirements: _containers.ScalarMap[str, str]
    def __init__(self, available: _Optional[bool] = ..., method: _Optional[str] = ..., reason: _Optional[str] = ..., requirements: _Optional[_Mapping[str, str]] = ...) -> None: ...

class PreflightReady(_message.Message):
    __slots__ = ("ready", "details", "gpu", "snapshot_at", "waited_seconds")
    class DetailsEntry(_message.Message):
        __slots__ = ("key", "value")
        KEY_FIELD_NUMBER: _ClassVar[int]
        VALUE_FIELD_NUMBER: _ClassVar[int]
        key: str
        value: bool
        def __init__(self, key: _Optional[str] = ..., value: _Optional[bool] = ...) -> None: ...
    READY_FIELD_NUMBER: _ClassVar[int]
    DETAILS_FIELD_NUMBER: _ClassVar[int]
    GPU_FIELD_NUMBER: _ClassVar[int]
    SNAPSHOT_AT_FIELD_NUMBER: _ClassVar[int]
    WAITED_SECONDS_FIELD_NUMBER: _ClassVar[int]
    ready: bool
    details: _containers.ScalarMap[str, bool]
    gpu: GPUInfo
    snapshot_at: _timestamp_pb2.Timestamp
    waited_seconds: int
    def __init__(self, ready: _Optional[bool] = ..., details: _Optional[_Mapping[str, bool]] = ..., gpu: _Optional[_Union[GPUInfo, _Mapping]] = ..., snapshot_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., waited_seconds: _Optional[int] = ...) -> None: ...

class PreflightRuntime(_message.Message):
    __slots__ = ("instance_id", "started_at", "app_data_dir", "bundle_root", "dry_run", "manifest_hash", "manifest_schema", "target", "app_name", "app_version", "ipc_host", "ipc_port", "runtime_version", "build_version")
    INSTANCE_ID_FIELD_NUMBER: _ClassVar[int]
    STARTED_AT_FIELD_NUMBER: _ClassVar[int]
    APP_DATA_DIR_FIELD_NUMBER: _ClassVar[int]
    BUNDLE_ROOT_FIELD_NUMBER: _ClassVar[int]
    DRY_RUN_FIELD_NUMBER: _ClassVar[int]
    MANIFEST_HASH_FIELD_NUMBER: _ClassVar[int]
    MANIFEST_SCHEMA_FIELD_NUMBER: _ClassVar[int]
    TARGET_FIELD_NUMBER: _ClassVar[int]
    APP_NAME_FIELD_NUMBER: _ClassVar[int]
    APP_VERSION_FIELD_NUMBER: _ClassVar[int]
    IPC_HOST_FIELD_NUMBER: _ClassVar[int]
    IPC_PORT_FIELD_NUMBER: _ClassVar[int]
    RUNTIME_VERSION_FIELD_NUMBER: _ClassVar[int]
    BUILD_VERSION_FIELD_NUMBER: _ClassVar[int]
    instance_id: str
    started_at: _timestamp_pb2.Timestamp
    app_data_dir: str
    bundle_root: str
    dry_run: bool
    manifest_hash: str
    manifest_schema: str
    target: str
    app_name: str
    app_version: str
    ipc_host: str
    ipc_port: int
    runtime_version: str
    build_version: str
    def __init__(self, instance_id: _Optional[str] = ..., started_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., app_data_dir: _Optional[str] = ..., bundle_root: _Optional[str] = ..., dry_run: _Optional[bool] = ..., manifest_hash: _Optional[str] = ..., manifest_schema: _Optional[str] = ..., target: _Optional[str] = ..., app_name: _Optional[str] = ..., app_version: _Optional[str] = ..., ipc_host: _Optional[str] = ..., ipc_port: _Optional[int] = ..., runtime_version: _Optional[str] = ..., build_version: _Optional[str] = ...) -> None: ...

class PreflightCheck(_message.Message):
    __slots__ = ("id", "step", "name", "status", "detail")
    ID_FIELD_NUMBER: _ClassVar[int]
    STEP_FIELD_NUMBER: _ClassVar[int]
    NAME_FIELD_NUMBER: _ClassVar[int]
    STATUS_FIELD_NUMBER: _ClassVar[int]
    DETAIL_FIELD_NUMBER: _ClassVar[int]
    id: str
    step: int
    name: str
    status: CheckStatus
    detail: str
    def __init__(self, id: _Optional[str] = ..., step: _Optional[int] = ..., name: _Optional[str] = ..., status: _Optional[_Union[CheckStatus, str]] = ..., detail: _Optional[str] = ...) -> None: ...

class ServiceLogTail(_message.Message):
    __slots__ = ("service_id", "lines", "content", "error")
    SERVICE_ID_FIELD_NUMBER: _ClassVar[int]
    LINES_FIELD_NUMBER: _ClassVar[int]
    CONTENT_FIELD_NUMBER: _ClassVar[int]
    ERROR_FIELD_NUMBER: _ClassVar[int]
    service_id: str
    lines: int
    content: str
    error: str
    def __init__(self, service_id: _Optional[str] = ..., lines: _Optional[int] = ..., content: _Optional[str] = ..., error: _Optional[str] = ...) -> None: ...

class ServiceFingerprint(_message.Message):
    __slots__ = ("service_id", "platform", "binary_path", "binary_resolved_path", "binary_sha256", "binary_size_bytes", "binary_mtime", "error")
    SERVICE_ID_FIELD_NUMBER: _ClassVar[int]
    PLATFORM_FIELD_NUMBER: _ClassVar[int]
    BINARY_PATH_FIELD_NUMBER: _ClassVar[int]
    BINARY_RESOLVED_PATH_FIELD_NUMBER: _ClassVar[int]
    BINARY_SHA256_FIELD_NUMBER: _ClassVar[int]
    BINARY_SIZE_BYTES_FIELD_NUMBER: _ClassVar[int]
    BINARY_MTIME_FIELD_NUMBER: _ClassVar[int]
    ERROR_FIELD_NUMBER: _ClassVar[int]
    service_id: str
    platform: str
    binary_path: str
    binary_resolved_path: str
    binary_sha256: str
    binary_size_bytes: int
    binary_mtime: _timestamp_pb2.Timestamp
    error: str
    def __init__(self, service_id: _Optional[str] = ..., platform: _Optional[str] = ..., binary_path: _Optional[str] = ..., binary_resolved_path: _Optional[str] = ..., binary_sha256: _Optional[str] = ..., binary_size_bytes: _Optional[int] = ..., binary_mtime: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., error: _Optional[str] = ...) -> None: ...

class TelemetryInfo(_message.Message):
    __slots__ = ("path", "upload_url")
    PATH_FIELD_NUMBER: _ClassVar[int]
    UPLOAD_URL_FIELD_NUMBER: _ClassVar[int]
    path: str
    upload_url: str
    def __init__(self, path: _Optional[str] = ..., upload_url: _Optional[str] = ...) -> None: ...

class PreflightResponse(_message.Message):
    __slots__ = ("status", "validation", "ready", "secrets", "ports", "telemetry", "log_tails", "checks", "runtime", "service_fingerprints", "errors", "warnings", "session_id", "expires_at")
    class PortsEntry(_message.Message):
        __slots__ = ("key", "value")
        KEY_FIELD_NUMBER: _ClassVar[int]
        VALUE_FIELD_NUMBER: _ClassVar[int]
        key: str
        value: int
        def __init__(self, key: _Optional[str] = ..., value: _Optional[int] = ...) -> None: ...
    STATUS_FIELD_NUMBER: _ClassVar[int]
    VALIDATION_FIELD_NUMBER: _ClassVar[int]
    READY_FIELD_NUMBER: _ClassVar[int]
    SECRETS_FIELD_NUMBER: _ClassVar[int]
    PORTS_FIELD_NUMBER: _ClassVar[int]
    TELEMETRY_FIELD_NUMBER: _ClassVar[int]
    LOG_TAILS_FIELD_NUMBER: _ClassVar[int]
    CHECKS_FIELD_NUMBER: _ClassVar[int]
    RUNTIME_FIELD_NUMBER: _ClassVar[int]
    SERVICE_FINGERPRINTS_FIELD_NUMBER: _ClassVar[int]
    ERRORS_FIELD_NUMBER: _ClassVar[int]
    WARNINGS_FIELD_NUMBER: _ClassVar[int]
    SESSION_ID_FIELD_NUMBER: _ClassVar[int]
    EXPIRES_AT_FIELD_NUMBER: _ClassVar[int]
    status: PreflightStatus
    validation: str
    ready: PreflightReady
    secrets: _containers.RepeatedCompositeFieldContainer[PreflightSecret]
    ports: _containers.ScalarMap[str, int]
    telemetry: TelemetryInfo
    log_tails: _containers.RepeatedCompositeFieldContainer[ServiceLogTail]
    checks: _containers.RepeatedCompositeFieldContainer[PreflightCheck]
    runtime: PreflightRuntime
    service_fingerprints: _containers.RepeatedCompositeFieldContainer[ServiceFingerprint]
    errors: _containers.RepeatedCompositeFieldContainer[_shared_pb2.ValidationError]
    warnings: _containers.RepeatedCompositeFieldContainer[_shared_pb2.ValidationWarning]
    session_id: str
    expires_at: _timestamp_pb2.Timestamp
    def __init__(self, status: _Optional[_Union[PreflightStatus, str]] = ..., validation: _Optional[str] = ..., ready: _Optional[_Union[PreflightReady, _Mapping]] = ..., secrets: _Optional[_Iterable[_Union[PreflightSecret, _Mapping]]] = ..., ports: _Optional[_Mapping[str, int]] = ..., telemetry: _Optional[_Union[TelemetryInfo, _Mapping]] = ..., log_tails: _Optional[_Iterable[_Union[ServiceLogTail, _Mapping]]] = ..., checks: _Optional[_Iterable[_Union[PreflightCheck, _Mapping]]] = ..., runtime: _Optional[_Union[PreflightRuntime, _Mapping]] = ..., service_fingerprints: _Optional[_Iterable[_Union[ServiceFingerprint, _Mapping]]] = ..., errors: _Optional[_Iterable[_Union[_shared_pb2.ValidationError, _Mapping]]] = ..., warnings: _Optional[_Iterable[_Union[_shared_pb2.ValidationWarning, _Mapping]]] = ..., session_id: _Optional[str] = ..., expires_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ...) -> None: ...

class JobStep(_message.Message):
    __slots__ = ("id", "name", "state", "detail")
    ID_FIELD_NUMBER: _ClassVar[int]
    NAME_FIELD_NUMBER: _ClassVar[int]
    STATE_FIELD_NUMBER: _ClassVar[int]
    DETAIL_FIELD_NUMBER: _ClassVar[int]
    id: str
    name: str
    state: CheckStatus
    detail: str
    def __init__(self, id: _Optional[str] = ..., name: _Optional[str] = ..., state: _Optional[_Union[CheckStatus, str]] = ..., detail: _Optional[str] = ...) -> None: ...

class JobStartResponse(_message.Message):
    __slots__ = ("job_id",)
    JOB_ID_FIELD_NUMBER: _ClassVar[int]
    job_id: str
    def __init__(self, job_id: _Optional[str] = ...) -> None: ...

class JobStatusResponse(_message.Message):
    __slots__ = ("job_id", "status", "steps", "result", "error", "started_at", "updated_at")
    JOB_ID_FIELD_NUMBER: _ClassVar[int]
    STATUS_FIELD_NUMBER: _ClassVar[int]
    STEPS_FIELD_NUMBER: _ClassVar[int]
    RESULT_FIELD_NUMBER: _ClassVar[int]
    ERROR_FIELD_NUMBER: _ClassVar[int]
    STARTED_AT_FIELD_NUMBER: _ClassVar[int]
    UPDATED_AT_FIELD_NUMBER: _ClassVar[int]
    job_id: str
    status: JobStatus
    steps: _containers.RepeatedCompositeFieldContainer[JobStep]
    result: PreflightResponse
    error: str
    started_at: _timestamp_pb2.Timestamp
    updated_at: _timestamp_pb2.Timestamp
    def __init__(self, job_id: _Optional[str] = ..., status: _Optional[_Union[JobStatus, str]] = ..., steps: _Optional[_Iterable[_Union[JobStep, _Mapping]]] = ..., result: _Optional[_Union[PreflightResponse, _Mapping]] = ..., error: _Optional[str] = ..., started_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., updated_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ...) -> None: ...

class ManifestRequest(_message.Message):
    __slots__ = ("manifest_path",)
    MANIFEST_PATH_FIELD_NUMBER: _ClassVar[int]
    manifest_path: str
    def __init__(self, manifest_path: _Optional[str] = ...) -> None: ...

class ManifestResponse(_message.Message):
    __slots__ = ("manifest", "schema_version", "errors")
    class ManifestEntry(_message.Message):
        __slots__ = ("key", "value")
        KEY_FIELD_NUMBER: _ClassVar[int]
        VALUE_FIELD_NUMBER: _ClassVar[int]
        key: str
        value: str
        def __init__(self, key: _Optional[str] = ..., value: _Optional[str] = ...) -> None: ...
    MANIFEST_FIELD_NUMBER: _ClassVar[int]
    SCHEMA_VERSION_FIELD_NUMBER: _ClassVar[int]
    ERRORS_FIELD_NUMBER: _ClassVar[int]
    manifest: _containers.ScalarMap[str, str]
    schema_version: str
    errors: _containers.RepeatedCompositeFieldContainer[_shared_pb2.ValidationError]
    def __init__(self, manifest: _Optional[_Mapping[str, str]] = ..., schema_version: _Optional[str] = ..., errors: _Optional[_Iterable[_Union[_shared_pb2.ValidationError, _Mapping]]] = ...) -> None: ...
