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

class PlatformBuildStatus(int, metaclass=_enum_type_wrapper.EnumTypeWrapper):
    __slots__ = ()
    PLATFORM_BUILD_STATUS_UNSPECIFIED: _ClassVar[PlatformBuildStatus]
    PLATFORM_BUILD_STATUS_BUILDING: _ClassVar[PlatformBuildStatus]
    PLATFORM_BUILD_STATUS_READY: _ClassVar[PlatformBuildStatus]
    PLATFORM_BUILD_STATUS_FAILED: _ClassVar[PlatformBuildStatus]
    PLATFORM_BUILD_STATUS_SKIPPED: _ClassVar[PlatformBuildStatus]

class SmokeTestStatus(int, metaclass=_enum_type_wrapper.EnumTypeWrapper):
    __slots__ = ()
    SMOKE_TEST_STATUS_UNSPECIFIED: _ClassVar[SmokeTestStatus]
    SMOKE_TEST_STATUS_RUNNING: _ClassVar[SmokeTestStatus]
    SMOKE_TEST_STATUS_PASSED: _ClassVar[SmokeTestStatus]
    SMOKE_TEST_STATUS_FAILED: _ClassVar[SmokeTestStatus]
PLATFORM_BUILD_STATUS_UNSPECIFIED: PlatformBuildStatus
PLATFORM_BUILD_STATUS_BUILDING: PlatformBuildStatus
PLATFORM_BUILD_STATUS_READY: PlatformBuildStatus
PLATFORM_BUILD_STATUS_FAILED: PlatformBuildStatus
PLATFORM_BUILD_STATUS_SKIPPED: PlatformBuildStatus
SMOKE_TEST_STATUS_UNSPECIFIED: SmokeTestStatus
SMOKE_TEST_STATUS_RUNNING: SmokeTestStatus
SMOKE_TEST_STATUS_PASSED: SmokeTestStatus
SMOKE_TEST_STATUS_FAILED: SmokeTestStatus

class PlatformBuildResult(_message.Message):
    __slots__ = ("platform", "status", "started_at", "completed_at", "error_log", "artifact", "file_size", "skip_reason")
    PLATFORM_FIELD_NUMBER: _ClassVar[int]
    STATUS_FIELD_NUMBER: _ClassVar[int]
    STARTED_AT_FIELD_NUMBER: _ClassVar[int]
    COMPLETED_AT_FIELD_NUMBER: _ClassVar[int]
    ERROR_LOG_FIELD_NUMBER: _ClassVar[int]
    ARTIFACT_FIELD_NUMBER: _ClassVar[int]
    FILE_SIZE_FIELD_NUMBER: _ClassVar[int]
    SKIP_REASON_FIELD_NUMBER: _ClassVar[int]
    platform: _shared_pb2.Platform
    status: PlatformBuildStatus
    started_at: _timestamp_pb2.Timestamp
    completed_at: _timestamp_pb2.Timestamp
    error_log: _containers.RepeatedScalarFieldContainer[str]
    artifact: str
    file_size: int
    skip_reason: str
    def __init__(self, platform: _Optional[_Union[_shared_pb2.Platform, str]] = ..., status: _Optional[_Union[PlatformBuildStatus, str]] = ..., started_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., completed_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., error_log: _Optional[_Iterable[str]] = ..., artifact: _Optional[str] = ..., file_size: _Optional[int] = ..., skip_reason: _Optional[str] = ...) -> None: ...

class BuildStatusResponse(_message.Message):
    __slots__ = ("build_id", "scenario_name", "status", "framework", "template_type", "requested_platforms", "platform_results", "output_path", "created_at", "completed_at", "error_log", "build_log", "artifacts", "metadata")
    class PlatformResultsEntry(_message.Message):
        __slots__ = ("key", "value")
        KEY_FIELD_NUMBER: _ClassVar[int]
        VALUE_FIELD_NUMBER: _ClassVar[int]
        key: str
        value: PlatformBuildResult
        def __init__(self, key: _Optional[str] = ..., value: _Optional[_Union[PlatformBuildResult, _Mapping]] = ...) -> None: ...
    class ArtifactsEntry(_message.Message):
        __slots__ = ("key", "value")
        KEY_FIELD_NUMBER: _ClassVar[int]
        VALUE_FIELD_NUMBER: _ClassVar[int]
        key: str
        value: str
        def __init__(self, key: _Optional[str] = ..., value: _Optional[str] = ...) -> None: ...
    class MetadataEntry(_message.Message):
        __slots__ = ("key", "value")
        KEY_FIELD_NUMBER: _ClassVar[int]
        VALUE_FIELD_NUMBER: _ClassVar[int]
        key: str
        value: str
        def __init__(self, key: _Optional[str] = ..., value: _Optional[str] = ...) -> None: ...
    BUILD_ID_FIELD_NUMBER: _ClassVar[int]
    SCENARIO_NAME_FIELD_NUMBER: _ClassVar[int]
    STATUS_FIELD_NUMBER: _ClassVar[int]
    FRAMEWORK_FIELD_NUMBER: _ClassVar[int]
    TEMPLATE_TYPE_FIELD_NUMBER: _ClassVar[int]
    REQUESTED_PLATFORMS_FIELD_NUMBER: _ClassVar[int]
    PLATFORM_RESULTS_FIELD_NUMBER: _ClassVar[int]
    OUTPUT_PATH_FIELD_NUMBER: _ClassVar[int]
    CREATED_AT_FIELD_NUMBER: _ClassVar[int]
    COMPLETED_AT_FIELD_NUMBER: _ClassVar[int]
    ERROR_LOG_FIELD_NUMBER: _ClassVar[int]
    BUILD_LOG_FIELD_NUMBER: _ClassVar[int]
    ARTIFACTS_FIELD_NUMBER: _ClassVar[int]
    METADATA_FIELD_NUMBER: _ClassVar[int]
    build_id: str
    scenario_name: str
    status: _shared_pb2.BuildStatus
    framework: _shared_pb2.Framework
    template_type: _shared_pb2.TemplateType
    requested_platforms: _containers.RepeatedScalarFieldContainer[_shared_pb2.Platform]
    platform_results: _containers.MessageMap[str, PlatformBuildResult]
    output_path: str
    created_at: _timestamp_pb2.Timestamp
    completed_at: _timestamp_pb2.Timestamp
    error_log: _containers.RepeatedScalarFieldContainer[str]
    build_log: _containers.RepeatedScalarFieldContainer[str]
    artifacts: _containers.ScalarMap[str, str]
    metadata: _containers.ScalarMap[str, str]
    def __init__(self, build_id: _Optional[str] = ..., scenario_name: _Optional[str] = ..., status: _Optional[_Union[_shared_pb2.BuildStatus, str]] = ..., framework: _Optional[_Union[_shared_pb2.Framework, str]] = ..., template_type: _Optional[_Union[_shared_pb2.TemplateType, str]] = ..., requested_platforms: _Optional[_Iterable[_Union[_shared_pb2.Platform, str]]] = ..., platform_results: _Optional[_Mapping[str, PlatformBuildResult]] = ..., output_path: _Optional[str] = ..., created_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., completed_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., error_log: _Optional[_Iterable[str]] = ..., build_log: _Optional[_Iterable[str]] = ..., artifacts: _Optional[_Mapping[str, str]] = ..., metadata: _Optional[_Mapping[str, str]] = ...) -> None: ...

class BuildRequest(_message.Message):
    __slots__ = ("desktop_path", "platforms", "sign", "publish")
    DESKTOP_PATH_FIELD_NUMBER: _ClassVar[int]
    PLATFORMS_FIELD_NUMBER: _ClassVar[int]
    SIGN_FIELD_NUMBER: _ClassVar[int]
    PUBLISH_FIELD_NUMBER: _ClassVar[int]
    desktop_path: str
    platforms: _containers.RepeatedScalarFieldContainer[_shared_pb2.Platform]
    sign: bool
    publish: bool
    def __init__(self, desktop_path: _Optional[str] = ..., platforms: _Optional[_Iterable[_Union[_shared_pb2.Platform, str]]] = ..., sign: _Optional[bool] = ..., publish: _Optional[bool] = ...) -> None: ...

class ScenarioBuildRequest(_message.Message):
    __slots__ = ("scenario_name", "desktop_path", "platforms", "clean")
    SCENARIO_NAME_FIELD_NUMBER: _ClassVar[int]
    DESKTOP_PATH_FIELD_NUMBER: _ClassVar[int]
    PLATFORMS_FIELD_NUMBER: _ClassVar[int]
    CLEAN_FIELD_NUMBER: _ClassVar[int]
    scenario_name: str
    desktop_path: str
    platforms: _containers.RepeatedScalarFieldContainer[_shared_pb2.Platform]
    clean: bool
    def __init__(self, scenario_name: _Optional[str] = ..., desktop_path: _Optional[str] = ..., platforms: _Optional[_Iterable[_Union[_shared_pb2.Platform, str]]] = ..., clean: _Optional[bool] = ...) -> None: ...

class BuildResponse(_message.Message):
    __slots__ = ("build_id", "status", "status_url")
    BUILD_ID_FIELD_NUMBER: _ClassVar[int]
    STATUS_FIELD_NUMBER: _ClassVar[int]
    STATUS_URL_FIELD_NUMBER: _ClassVar[int]
    build_id: str
    status: str
    status_url: str
    def __init__(self, build_id: _Optional[str] = ..., status: _Optional[str] = ..., status_url: _Optional[str] = ...) -> None: ...

class SmokeTestStatusResponse(_message.Message):
    __slots__ = ("smoke_test_id", "scenario_name", "platform", "status", "artifact_path", "started_at", "completed_at", "logs", "error", "telemetry_uploaded", "telemetry_upload_error")
    SMOKE_TEST_ID_FIELD_NUMBER: _ClassVar[int]
    SCENARIO_NAME_FIELD_NUMBER: _ClassVar[int]
    PLATFORM_FIELD_NUMBER: _ClassVar[int]
    STATUS_FIELD_NUMBER: _ClassVar[int]
    ARTIFACT_PATH_FIELD_NUMBER: _ClassVar[int]
    STARTED_AT_FIELD_NUMBER: _ClassVar[int]
    COMPLETED_AT_FIELD_NUMBER: _ClassVar[int]
    LOGS_FIELD_NUMBER: _ClassVar[int]
    ERROR_FIELD_NUMBER: _ClassVar[int]
    TELEMETRY_UPLOADED_FIELD_NUMBER: _ClassVar[int]
    TELEMETRY_UPLOAD_ERROR_FIELD_NUMBER: _ClassVar[int]
    smoke_test_id: str
    scenario_name: str
    platform: _shared_pb2.Platform
    status: SmokeTestStatus
    artifact_path: str
    started_at: _timestamp_pb2.Timestamp
    completed_at: _timestamp_pb2.Timestamp
    logs: _containers.RepeatedScalarFieldContainer[str]
    error: str
    telemetry_uploaded: bool
    telemetry_upload_error: str
    def __init__(self, smoke_test_id: _Optional[str] = ..., scenario_name: _Optional[str] = ..., platform: _Optional[_Union[_shared_pb2.Platform, str]] = ..., status: _Optional[_Union[SmokeTestStatus, str]] = ..., artifact_path: _Optional[str] = ..., started_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., completed_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., logs: _Optional[_Iterable[str]] = ..., error: _Optional[str] = ..., telemetry_uploaded: _Optional[bool] = ..., telemetry_upload_error: _Optional[str] = ...) -> None: ...

class SmokeTestStartRequest(_message.Message):
    __slots__ = ("scenario_name", "platform")
    SCENARIO_NAME_FIELD_NUMBER: _ClassVar[int]
    PLATFORM_FIELD_NUMBER: _ClassVar[int]
    scenario_name: str
    platform: _shared_pb2.Platform
    def __init__(self, scenario_name: _Optional[str] = ..., platform: _Optional[_Union[_shared_pb2.Platform, str]] = ...) -> None: ...

class SmokeTestStartResponse(_message.Message):
    __slots__ = ("smoke_test_id", "scenario_name", "platform", "status", "artifact_path", "started_at", "logs")
    SMOKE_TEST_ID_FIELD_NUMBER: _ClassVar[int]
    SCENARIO_NAME_FIELD_NUMBER: _ClassVar[int]
    PLATFORM_FIELD_NUMBER: _ClassVar[int]
    STATUS_FIELD_NUMBER: _ClassVar[int]
    ARTIFACT_PATH_FIELD_NUMBER: _ClassVar[int]
    STARTED_AT_FIELD_NUMBER: _ClassVar[int]
    LOGS_FIELD_NUMBER: _ClassVar[int]
    smoke_test_id: str
    scenario_name: str
    platform: _shared_pb2.Platform
    status: SmokeTestStatus
    artifact_path: str
    started_at: _timestamp_pb2.Timestamp
    logs: _containers.RepeatedScalarFieldContainer[str]
    def __init__(self, smoke_test_id: _Optional[str] = ..., scenario_name: _Optional[str] = ..., platform: _Optional[_Union[_shared_pb2.Platform, str]] = ..., status: _Optional[_Union[SmokeTestStatus, str]] = ..., artifact_path: _Optional[str] = ..., started_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., logs: _Optional[_Iterable[str]] = ...) -> None: ...

class SmokeTestCancelResponse(_message.Message):
    __slots__ = ("status",)
    STATUS_FIELD_NUMBER: _ClassVar[int]
    status: str
    def __init__(self, status: _Optional[str] = ...) -> None: ...
