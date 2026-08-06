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
    platform: _common_pb2.Platform
    status: PlatformBuildStatus
    started_at: _timestamp_pb2.Timestamp
    completed_at: _timestamp_pb2.Timestamp
    error_log: _containers.RepeatedScalarFieldContainer[str]
    artifact: str
    file_size: int
    skip_reason: str
    def __init__(self, platform: _Optional[_Union[_common_pb2.Platform, str]] = ..., status: _Optional[_Union[PlatformBuildStatus, str]] = ..., started_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., completed_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., error_log: _Optional[_Iterable[str]] = ..., artifact: _Optional[str] = ..., file_size: _Optional[int] = ..., skip_reason: _Optional[str] = ...) -> None: ...

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
    status: _common_pb2.BuildStatus
    framework: _common_pb2.Framework
    template_type: _common_pb2.TemplateType
    requested_platforms: _containers.RepeatedScalarFieldContainer[_common_pb2.Platform]
    platform_results: _containers.MessageMap[str, PlatformBuildResult]
    output_path: str
    created_at: _timestamp_pb2.Timestamp
    completed_at: _timestamp_pb2.Timestamp
    error_log: _containers.RepeatedScalarFieldContainer[str]
    build_log: _containers.RepeatedScalarFieldContainer[str]
    artifacts: _containers.ScalarMap[str, str]
    metadata: _containers.ScalarMap[str, str]
    def __init__(self, build_id: _Optional[str] = ..., scenario_name: _Optional[str] = ..., status: _Optional[_Union[_common_pb2.BuildStatus, str]] = ..., framework: _Optional[_Union[_common_pb2.Framework, str]] = ..., template_type: _Optional[_Union[_common_pb2.TemplateType, str]] = ..., requested_platforms: _Optional[_Iterable[_Union[_common_pb2.Platform, str]]] = ..., platform_results: _Optional[_Mapping[str, PlatformBuildResult]] = ..., output_path: _Optional[str] = ..., created_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., completed_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., error_log: _Optional[_Iterable[str]] = ..., build_log: _Optional[_Iterable[str]] = ..., artifacts: _Optional[_Mapping[str, str]] = ..., metadata: _Optional[_Mapping[str, str]] = ...) -> None: ...

class ScreenRecordingSummary(_message.Message):
    __slots__ = ("recorded", "duration_ms", "file_size_bytes", "error", "capture_id")
    RECORDED_FIELD_NUMBER: _ClassVar[int]
    DURATION_MS_FIELD_NUMBER: _ClassVar[int]
    FILE_SIZE_BYTES_FIELD_NUMBER: _ClassVar[int]
    ERROR_FIELD_NUMBER: _ClassVar[int]
    CAPTURE_ID_FIELD_NUMBER: _ClassVar[int]
    recorded: bool
    duration_ms: int
    file_size_bytes: int
    error: str
    capture_id: str
    def __init__(self, recorded: _Optional[bool] = ..., duration_ms: _Optional[int] = ..., file_size_bytes: _Optional[int] = ..., error: _Optional[str] = ..., capture_id: _Optional[str] = ...) -> None: ...

class EvidenceChapter(_message.Message):
    __slots__ = ("id", "purpose", "action", "disposition", "assertion_id", "expected", "observed", "error", "video_start_offset_ms", "video_end_offset_ms", "evidence_ids")
    ID_FIELD_NUMBER: _ClassVar[int]
    PURPOSE_FIELD_NUMBER: _ClassVar[int]
    ACTION_FIELD_NUMBER: _ClassVar[int]
    DISPOSITION_FIELD_NUMBER: _ClassVar[int]
    ASSERTION_ID_FIELD_NUMBER: _ClassVar[int]
    EXPECTED_FIELD_NUMBER: _ClassVar[int]
    OBSERVED_FIELD_NUMBER: _ClassVar[int]
    ERROR_FIELD_NUMBER: _ClassVar[int]
    VIDEO_START_OFFSET_MS_FIELD_NUMBER: _ClassVar[int]
    VIDEO_END_OFFSET_MS_FIELD_NUMBER: _ClassVar[int]
    EVIDENCE_IDS_FIELD_NUMBER: _ClassVar[int]
    id: str
    purpose: str
    action: str
    disposition: str
    assertion_id: str
    expected: str
    observed: str
    error: str
    video_start_offset_ms: int
    video_end_offset_ms: int
    evidence_ids: _containers.RepeatedScalarFieldContainer[str]
    def __init__(self, id: _Optional[str] = ..., purpose: _Optional[str] = ..., action: _Optional[str] = ..., disposition: _Optional[str] = ..., assertion_id: _Optional[str] = ..., expected: _Optional[str] = ..., observed: _Optional[str] = ..., error: _Optional[str] = ..., video_start_offset_ms: _Optional[int] = ..., video_end_offset_ms: _Optional[int] = ..., evidence_ids: _Optional[_Iterable[str]] = ...) -> None: ...

class EvidenceReview(_message.Message):
    __slots__ = ("schema_version", "capability", "plan_id", "profile", "disposition", "reason", "chapters", "event_count", "deployment_mode", "provider_tier", "service_identity", "readiness", "fallback_decision", "safe_route_class")
    SCHEMA_VERSION_FIELD_NUMBER: _ClassVar[int]
    CAPABILITY_FIELD_NUMBER: _ClassVar[int]
    PLAN_ID_FIELD_NUMBER: _ClassVar[int]
    PROFILE_FIELD_NUMBER: _ClassVar[int]
    DISPOSITION_FIELD_NUMBER: _ClassVar[int]
    REASON_FIELD_NUMBER: _ClassVar[int]
    CHAPTERS_FIELD_NUMBER: _ClassVar[int]
    EVENT_COUNT_FIELD_NUMBER: _ClassVar[int]
    DEPLOYMENT_MODE_FIELD_NUMBER: _ClassVar[int]
    PROVIDER_TIER_FIELD_NUMBER: _ClassVar[int]
    SERVICE_IDENTITY_FIELD_NUMBER: _ClassVar[int]
    READINESS_FIELD_NUMBER: _ClassVar[int]
    FALLBACK_DECISION_FIELD_NUMBER: _ClassVar[int]
    SAFE_ROUTE_CLASS_FIELD_NUMBER: _ClassVar[int]
    schema_version: str
    capability: str
    plan_id: str
    profile: str
    disposition: str
    reason: str
    chapters: _containers.RepeatedCompositeFieldContainer[EvidenceChapter]
    event_count: int
    deployment_mode: str
    provider_tier: str
    service_identity: str
    readiness: str
    fallback_decision: str
    safe_route_class: str
    def __init__(self, schema_version: _Optional[str] = ..., capability: _Optional[str] = ..., plan_id: _Optional[str] = ..., profile: _Optional[str] = ..., disposition: _Optional[str] = ..., reason: _Optional[str] = ..., chapters: _Optional[_Iterable[_Union[EvidenceChapter, _Mapping]]] = ..., event_count: _Optional[int] = ..., deployment_mode: _Optional[str] = ..., provider_tier: _Optional[str] = ..., service_identity: _Optional[str] = ..., readiness: _Optional[str] = ..., fallback_decision: _Optional[str] = ..., safe_route_class: _Optional[str] = ...) -> None: ...

class SmokeTestStatusResponse(_message.Message):
    __slots__ = ("smoke_test_id", "scenario_name", "platform", "status", "artifact_path", "started_at", "completed_at", "logs", "error", "telemetry_uploaded", "telemetry_upload_error", "screen_recording", "evidence_review")
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
    SCREEN_RECORDING_FIELD_NUMBER: _ClassVar[int]
    EVIDENCE_REVIEW_FIELD_NUMBER: _ClassVar[int]
    smoke_test_id: str
    scenario_name: str
    platform: _common_pb2.Platform
    status: SmokeTestStatus
    artifact_path: str
    started_at: _timestamp_pb2.Timestamp
    completed_at: _timestamp_pb2.Timestamp
    logs: _containers.RepeatedScalarFieldContainer[str]
    error: str
    telemetry_uploaded: bool
    telemetry_upload_error: str
    screen_recording: ScreenRecordingSummary
    evidence_review: EvidenceReview
    def __init__(self, smoke_test_id: _Optional[str] = ..., scenario_name: _Optional[str] = ..., platform: _Optional[_Union[_common_pb2.Platform, str]] = ..., status: _Optional[_Union[SmokeTestStatus, str]] = ..., artifact_path: _Optional[str] = ..., started_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., completed_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., logs: _Optional[_Iterable[str]] = ..., error: _Optional[str] = ..., telemetry_uploaded: _Optional[bool] = ..., telemetry_upload_error: _Optional[str] = ..., screen_recording: _Optional[_Union[ScreenRecordingSummary, _Mapping]] = ..., evidence_review: _Optional[_Union[EvidenceReview, _Mapping]] = ...) -> None: ...
