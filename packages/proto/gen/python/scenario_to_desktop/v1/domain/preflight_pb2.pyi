import datetime

from buf.validate import validate_pb2 as _validate_pb2
from google.protobuf import timestamp_pb2 as _timestamp_pb2
from scenario_to_desktop.v1.shared import common_pb2 as _common_pb2
from scenario_to_desktop.v1.shared import preflight_results_pb2 as _preflight_results_pb2
from google.protobuf.internal import containers as _containers
from google.protobuf.internal import enum_type_wrapper as _enum_type_wrapper
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Iterable as _Iterable, Mapping as _Mapping
from typing import ClassVar as _ClassVar, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class JobStatus(int, metaclass=_enum_type_wrapper.EnumTypeWrapper):
    __slots__ = ()
    JOB_STATUS_UNSPECIFIED: _ClassVar[JobStatus]
    JOB_STATUS_PENDING: _ClassVar[JobStatus]
    JOB_STATUS_RUNNING: _ClassVar[JobStatus]
    JOB_STATUS_COMPLETED: _ClassVar[JobStatus]
    JOB_STATUS_FAILED: _ClassVar[JobStatus]
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

class JobStep(_message.Message):
    __slots__ = ("id", "name", "state", "detail")
    ID_FIELD_NUMBER: _ClassVar[int]
    NAME_FIELD_NUMBER: _ClassVar[int]
    STATE_FIELD_NUMBER: _ClassVar[int]
    DETAIL_FIELD_NUMBER: _ClassVar[int]
    id: str
    name: str
    state: _preflight_results_pb2.CheckStatus
    detail: str
    def __init__(self, id: _Optional[str] = ..., name: _Optional[str] = ..., state: _Optional[_Union[_preflight_results_pb2.CheckStatus, str]] = ..., detail: _Optional[str] = ...) -> None: ...

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
    result: _preflight_results_pb2.PreflightResponse
    error: str
    started_at: _timestamp_pb2.Timestamp
    updated_at: _timestamp_pb2.Timestamp
    def __init__(self, job_id: _Optional[str] = ..., status: _Optional[_Union[JobStatus, str]] = ..., steps: _Optional[_Iterable[_Union[JobStep, _Mapping]]] = ..., result: _Optional[_Union[_preflight_results_pb2.PreflightResponse, _Mapping]] = ..., error: _Optional[str] = ..., started_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., updated_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ...) -> None: ...

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
    errors: _containers.RepeatedCompositeFieldContainer[_common_pb2.ValidationError]
    def __init__(self, manifest: _Optional[_Mapping[str, str]] = ..., schema_version: _Optional[str] = ..., errors: _Optional[_Iterable[_Union[_common_pb2.ValidationError, _Mapping]]] = ...) -> None: ...

class GetPreflightJobRequest(_message.Message):
    __slots__ = ("job_id",)
    JOB_ID_FIELD_NUMBER: _ClassVar[int]
    job_id: str
    def __init__(self, job_id: _Optional[str] = ...) -> None: ...
