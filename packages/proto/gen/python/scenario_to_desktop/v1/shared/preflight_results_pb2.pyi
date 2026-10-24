import datetime

from google.protobuf import timestamp_pb2 as _timestamp_pb2
from scenario_to_desktop.v1.shared import common_pb2 as _common_pb2
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

class PreflightCheckStep(int, metaclass=_enum_type_wrapper.EnumTypeWrapper):
    __slots__ = ()
    PREFLIGHT_CHECK_STEP_UNSPECIFIED: _ClassVar[PreflightCheckStep]
    PREFLIGHT_CHECK_STEP_VALIDATION: _ClassVar[PreflightCheckStep]
    PREFLIGHT_CHECK_STEP_SECRETS: _ClassVar[PreflightCheckStep]
    PREFLIGHT_CHECK_STEP_RUNTIME: _ClassVar[PreflightCheckStep]
    PREFLIGHT_CHECK_STEP_SERVICES: _ClassVar[PreflightCheckStep]
    PREFLIGHT_CHECK_STEP_DIAGNOSTICS: _ClassVar[PreflightCheckStep]

class SecretClass(int, metaclass=_enum_type_wrapper.EnumTypeWrapper):
    __slots__ = ()
    SECRET_CLASS_UNSPECIFIED: _ClassVar[SecretClass]
    SECRET_CLASS_API_KEY: _ClassVar[SecretClass]
    SECRET_CLASS_PASSWORD: _ClassVar[SecretClass]
    SECRET_CLASS_TOKEN: _ClassVar[SecretClass]
    SECRET_CLASS_CONNECTION_STRING: _ClassVar[SecretClass]
    SECRET_CLASS_CERTIFICATE: _ClassVar[SecretClass]
    SECRET_CLASS_GENERIC: _ClassVar[SecretClass]
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
PREFLIGHT_CHECK_STEP_UNSPECIFIED: PreflightCheckStep
PREFLIGHT_CHECK_STEP_VALIDATION: PreflightCheckStep
PREFLIGHT_CHECK_STEP_SECRETS: PreflightCheckStep
PREFLIGHT_CHECK_STEP_RUNTIME: PreflightCheckStep
PREFLIGHT_CHECK_STEP_SERVICES: PreflightCheckStep
PREFLIGHT_CHECK_STEP_DIAGNOSTICS: PreflightCheckStep
SECRET_CLASS_UNSPECIFIED: SecretClass
SECRET_CLASS_API_KEY: SecretClass
SECRET_CLASS_PASSWORD: SecretClass
SECRET_CLASS_TOKEN: SecretClass
SECRET_CLASS_CONNECTION_STRING: SecretClass
SECRET_CLASS_CERTIFICATE: SecretClass
SECRET_CLASS_GENERIC: SecretClass

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
    READY_FIELD_NUMBER: _ClassVar[int]
    DETAILS_FIELD_NUMBER: _ClassVar[int]
    GPU_FIELD_NUMBER: _ClassVar[int]
    SNAPSHOT_AT_FIELD_NUMBER: _ClassVar[int]
    WAITED_SECONDS_FIELD_NUMBER: _ClassVar[int]
    ready: bool
    details: _containers.RepeatedCompositeFieldContainer[ServiceReadiness]
    gpu: GPUInfo
    snapshot_at: _timestamp_pb2.Timestamp
    waited_seconds: int
    def __init__(self, ready: _Optional[bool] = ..., details: _Optional[_Iterable[_Union[ServiceReadiness, _Mapping]]] = ..., gpu: _Optional[_Union[GPUInfo, _Mapping]] = ..., snapshot_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., waited_seconds: _Optional[int] = ...) -> None: ...

class ServiceReadiness(_message.Message):
    __slots__ = ("service_id", "ready", "skipped", "message", "exit_code", "started_at", "ready_at", "updated_at")
    SERVICE_ID_FIELD_NUMBER: _ClassVar[int]
    READY_FIELD_NUMBER: _ClassVar[int]
    SKIPPED_FIELD_NUMBER: _ClassVar[int]
    MESSAGE_FIELD_NUMBER: _ClassVar[int]
    EXIT_CODE_FIELD_NUMBER: _ClassVar[int]
    STARTED_AT_FIELD_NUMBER: _ClassVar[int]
    READY_AT_FIELD_NUMBER: _ClassVar[int]
    UPDATED_AT_FIELD_NUMBER: _ClassVar[int]
    service_id: str
    ready: bool
    skipped: bool
    message: str
    exit_code: int
    started_at: _timestamp_pb2.Timestamp
    ready_at: _timestamp_pb2.Timestamp
    updated_at: _timestamp_pb2.Timestamp
    def __init__(self, service_id: _Optional[str] = ..., ready: _Optional[bool] = ..., skipped: _Optional[bool] = ..., message: _Optional[str] = ..., exit_code: _Optional[int] = ..., started_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., ready_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., updated_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ...) -> None: ...

class ServicePort(_message.Message):
    __slots__ = ("service_id", "name", "port")
    SERVICE_ID_FIELD_NUMBER: _ClassVar[int]
    NAME_FIELD_NUMBER: _ClassVar[int]
    PORT_FIELD_NUMBER: _ClassVar[int]
    service_id: str
    name: str
    port: int
    def __init__(self, service_id: _Optional[str] = ..., name: _Optional[str] = ..., port: _Optional[int] = ...) -> None: ...

class BundleValidationIssue(_message.Message):
    __slots__ = ("code", "service", "path", "message")
    CODE_FIELD_NUMBER: _ClassVar[int]
    SERVICE_FIELD_NUMBER: _ClassVar[int]
    PATH_FIELD_NUMBER: _ClassVar[int]
    MESSAGE_FIELD_NUMBER: _ClassVar[int]
    code: str
    service: str
    path: str
    message: str
    def __init__(self, code: _Optional[str] = ..., service: _Optional[str] = ..., path: _Optional[str] = ..., message: _Optional[str] = ...) -> None: ...

class MissingBinary(_message.Message):
    __slots__ = ("service_id", "platform", "path")
    SERVICE_ID_FIELD_NUMBER: _ClassVar[int]
    PLATFORM_FIELD_NUMBER: _ClassVar[int]
    PATH_FIELD_NUMBER: _ClassVar[int]
    service_id: str
    platform: str
    path: str
    def __init__(self, service_id: _Optional[str] = ..., platform: _Optional[str] = ..., path: _Optional[str] = ...) -> None: ...

class MissingAsset(_message.Message):
    __slots__ = ("service_id", "path")
    SERVICE_ID_FIELD_NUMBER: _ClassVar[int]
    PATH_FIELD_NUMBER: _ClassVar[int]
    service_id: str
    path: str
    def __init__(self, service_id: _Optional[str] = ..., path: _Optional[str] = ...) -> None: ...

class InvalidChecksum(_message.Message):
    __slots__ = ("service_id", "path", "expected", "actual")
    SERVICE_ID_FIELD_NUMBER: _ClassVar[int]
    PATH_FIELD_NUMBER: _ClassVar[int]
    EXPECTED_FIELD_NUMBER: _ClassVar[int]
    ACTUAL_FIELD_NUMBER: _ClassVar[int]
    service_id: str
    path: str
    expected: str
    actual: str
    def __init__(self, service_id: _Optional[str] = ..., path: _Optional[str] = ..., expected: _Optional[str] = ..., actual: _Optional[str] = ...) -> None: ...

class BundleValidationResult(_message.Message):
    __slots__ = ("valid", "errors", "warnings", "missing_binaries", "missing_assets", "invalid_checksums")
    VALID_FIELD_NUMBER: _ClassVar[int]
    ERRORS_FIELD_NUMBER: _ClassVar[int]
    WARNINGS_FIELD_NUMBER: _ClassVar[int]
    MISSING_BINARIES_FIELD_NUMBER: _ClassVar[int]
    MISSING_ASSETS_FIELD_NUMBER: _ClassVar[int]
    INVALID_CHECKSUMS_FIELD_NUMBER: _ClassVar[int]
    valid: bool
    errors: _containers.RepeatedCompositeFieldContainer[BundleValidationIssue]
    warnings: _containers.RepeatedCompositeFieldContainer[BundleValidationIssue]
    missing_binaries: _containers.RepeatedCompositeFieldContainer[MissingBinary]
    missing_assets: _containers.RepeatedCompositeFieldContainer[MissingAsset]
    invalid_checksums: _containers.RepeatedCompositeFieldContainer[InvalidChecksum]
    def __init__(self, valid: _Optional[bool] = ..., errors: _Optional[_Iterable[_Union[BundleValidationIssue, _Mapping]]] = ..., warnings: _Optional[_Iterable[_Union[BundleValidationIssue, _Mapping]]] = ..., missing_binaries: _Optional[_Iterable[_Union[MissingBinary, _Mapping]]] = ..., missing_assets: _Optional[_Iterable[_Union[MissingAsset, _Mapping]]] = ..., invalid_checksums: _Optional[_Iterable[_Union[InvalidChecksum, _Mapping]]] = ...) -> None: ...

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
    step: PreflightCheckStep
    name: str
    status: CheckStatus
    detail: str
    def __init__(self, id: _Optional[str] = ..., step: _Optional[_Union[PreflightCheckStep, str]] = ..., name: _Optional[str] = ..., status: _Optional[_Union[CheckStatus, str]] = ..., detail: _Optional[str] = ...) -> None: ...

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
    validation: BundleValidationResult
    ready: PreflightReady
    secrets: _containers.RepeatedCompositeFieldContainer[PreflightSecret]
    ports: _containers.RepeatedCompositeFieldContainer[ServicePort]
    telemetry: TelemetryInfo
    log_tails: _containers.RepeatedCompositeFieldContainer[ServiceLogTail]
    checks: _containers.RepeatedCompositeFieldContainer[PreflightCheck]
    runtime: PreflightRuntime
    service_fingerprints: _containers.RepeatedCompositeFieldContainer[ServiceFingerprint]
    errors: _containers.RepeatedCompositeFieldContainer[_common_pb2.ValidationError]
    warnings: _containers.RepeatedCompositeFieldContainer[_common_pb2.ValidationWarning]
    session_id: str
    expires_at: _timestamp_pb2.Timestamp
    def __init__(self, status: _Optional[_Union[PreflightStatus, str]] = ..., validation: _Optional[_Union[BundleValidationResult, _Mapping]] = ..., ready: _Optional[_Union[PreflightReady, _Mapping]] = ..., secrets: _Optional[_Iterable[_Union[PreflightSecret, _Mapping]]] = ..., ports: _Optional[_Iterable[_Union[ServicePort, _Mapping]]] = ..., telemetry: _Optional[_Union[TelemetryInfo, _Mapping]] = ..., log_tails: _Optional[_Iterable[_Union[ServiceLogTail, _Mapping]]] = ..., checks: _Optional[_Iterable[_Union[PreflightCheck, _Mapping]]] = ..., runtime: _Optional[_Union[PreflightRuntime, _Mapping]] = ..., service_fingerprints: _Optional[_Iterable[_Union[ServiceFingerprint, _Mapping]]] = ..., errors: _Optional[_Iterable[_Union[_common_pb2.ValidationError, _Mapping]]] = ..., warnings: _Optional[_Iterable[_Union[_common_pb2.ValidationWarning, _Mapping]]] = ..., session_id: _Optional[str] = ..., expires_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ...) -> None: ...
