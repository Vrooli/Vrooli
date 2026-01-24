import datetime

from google.protobuf import struct_pb2 as _struct_pb2
from google.protobuf import timestamp_pb2 as _timestamp_pb2
from scenario_to_desktop.v1.base import shared_pb2 as _shared_pb2
from google.protobuf.internal import containers as _containers
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Iterable as _Iterable, Mapping as _Mapping
from typing import ClassVar as _ClassVar, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class PipelineConfig(_message.Message):
    __slots__ = ("scenario_name", "platforms", "skip_preflight", "skip_smoke_test", "stop_on_failure", "deployment_mode", "framework", "template_type", "webhook_url", "proxy_url", "bundle_manifest_path", "clean", "sign", "publish", "distribute", "distribution_targets", "version", "preflight_timeout_seconds", "preflight_secrets", "stop_after_stage", "resume_from_stage", "parent_pipeline_id", "idempotency_key")
    class PreflightSecretsEntry(_message.Message):
        __slots__ = ("key", "value")
        KEY_FIELD_NUMBER: _ClassVar[int]
        VALUE_FIELD_NUMBER: _ClassVar[int]
        key: str
        value: str
        def __init__(self, key: _Optional[str] = ..., value: _Optional[str] = ...) -> None: ...
    SCENARIO_NAME_FIELD_NUMBER: _ClassVar[int]
    PLATFORMS_FIELD_NUMBER: _ClassVar[int]
    SKIP_PREFLIGHT_FIELD_NUMBER: _ClassVar[int]
    SKIP_SMOKE_TEST_FIELD_NUMBER: _ClassVar[int]
    STOP_ON_FAILURE_FIELD_NUMBER: _ClassVar[int]
    DEPLOYMENT_MODE_FIELD_NUMBER: _ClassVar[int]
    FRAMEWORK_FIELD_NUMBER: _ClassVar[int]
    TEMPLATE_TYPE_FIELD_NUMBER: _ClassVar[int]
    WEBHOOK_URL_FIELD_NUMBER: _ClassVar[int]
    PROXY_URL_FIELD_NUMBER: _ClassVar[int]
    BUNDLE_MANIFEST_PATH_FIELD_NUMBER: _ClassVar[int]
    CLEAN_FIELD_NUMBER: _ClassVar[int]
    SIGN_FIELD_NUMBER: _ClassVar[int]
    PUBLISH_FIELD_NUMBER: _ClassVar[int]
    DISTRIBUTE_FIELD_NUMBER: _ClassVar[int]
    DISTRIBUTION_TARGETS_FIELD_NUMBER: _ClassVar[int]
    VERSION_FIELD_NUMBER: _ClassVar[int]
    PREFLIGHT_TIMEOUT_SECONDS_FIELD_NUMBER: _ClassVar[int]
    PREFLIGHT_SECRETS_FIELD_NUMBER: _ClassVar[int]
    STOP_AFTER_STAGE_FIELD_NUMBER: _ClassVar[int]
    RESUME_FROM_STAGE_FIELD_NUMBER: _ClassVar[int]
    PARENT_PIPELINE_ID_FIELD_NUMBER: _ClassVar[int]
    IDEMPOTENCY_KEY_FIELD_NUMBER: _ClassVar[int]
    scenario_name: str
    platforms: _containers.RepeatedScalarFieldContainer[_shared_pb2.Platform]
    skip_preflight: bool
    skip_smoke_test: bool
    stop_on_failure: bool
    deployment_mode: _shared_pb2.DeploymentMode
    framework: _shared_pb2.Framework
    template_type: _shared_pb2.TemplateType
    webhook_url: str
    proxy_url: str
    bundle_manifest_path: str
    clean: bool
    sign: bool
    publish: bool
    distribute: bool
    distribution_targets: _containers.RepeatedScalarFieldContainer[str]
    version: str
    preflight_timeout_seconds: int
    preflight_secrets: _containers.ScalarMap[str, str]
    stop_after_stage: _shared_pb2.StageName
    resume_from_stage: _shared_pb2.StageName
    parent_pipeline_id: str
    idempotency_key: str
    def __init__(self, scenario_name: _Optional[str] = ..., platforms: _Optional[_Iterable[_Union[_shared_pb2.Platform, str]]] = ..., skip_preflight: _Optional[bool] = ..., skip_smoke_test: _Optional[bool] = ..., stop_on_failure: _Optional[bool] = ..., deployment_mode: _Optional[_Union[_shared_pb2.DeploymentMode, str]] = ..., framework: _Optional[_Union[_shared_pb2.Framework, str]] = ..., template_type: _Optional[_Union[_shared_pb2.TemplateType, str]] = ..., webhook_url: _Optional[str] = ..., proxy_url: _Optional[str] = ..., bundle_manifest_path: _Optional[str] = ..., clean: _Optional[bool] = ..., sign: _Optional[bool] = ..., publish: _Optional[bool] = ..., distribute: _Optional[bool] = ..., distribution_targets: _Optional[_Iterable[str]] = ..., version: _Optional[str] = ..., preflight_timeout_seconds: _Optional[int] = ..., preflight_secrets: _Optional[_Mapping[str, str]] = ..., stop_after_stage: _Optional[_Union[_shared_pb2.StageName, str]] = ..., resume_from_stage: _Optional[_Union[_shared_pb2.StageName, str]] = ..., parent_pipeline_id: _Optional[str] = ..., idempotency_key: _Optional[str] = ...) -> None: ...

class StageResult(_message.Message):
    __slots__ = ("stage", "status", "started_at", "completed_at", "error", "details", "logs")
    STAGE_FIELD_NUMBER: _ClassVar[int]
    STATUS_FIELD_NUMBER: _ClassVar[int]
    STARTED_AT_FIELD_NUMBER: _ClassVar[int]
    COMPLETED_AT_FIELD_NUMBER: _ClassVar[int]
    ERROR_FIELD_NUMBER: _ClassVar[int]
    DETAILS_FIELD_NUMBER: _ClassVar[int]
    LOGS_FIELD_NUMBER: _ClassVar[int]
    stage: _shared_pb2.StageName
    status: _shared_pb2.StageStatus
    started_at: _timestamp_pb2.Timestamp
    completed_at: _timestamp_pb2.Timestamp
    error: str
    details: _struct_pb2.Struct
    logs: _containers.RepeatedScalarFieldContainer[str]
    def __init__(self, stage: _Optional[_Union[_shared_pb2.StageName, str]] = ..., status: _Optional[_Union[_shared_pb2.StageStatus, str]] = ..., started_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., completed_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., error: _Optional[str] = ..., details: _Optional[_Union[_struct_pb2.Struct, _Mapping]] = ..., logs: _Optional[_Iterable[str]] = ...) -> None: ...

class PipelineStatus(_message.Message):
    __slots__ = ("pipeline_id", "scenario_name", "status", "current_stage", "progress_percent", "progress_message", "stages", "stage_order", "config", "started_at", "completed_at", "error", "final_artifacts", "stopped_after_stage", "parent_pipeline_id", "idempotency_key")
    class StagesEntry(_message.Message):
        __slots__ = ("key", "value")
        KEY_FIELD_NUMBER: _ClassVar[int]
        VALUE_FIELD_NUMBER: _ClassVar[int]
        key: str
        value: StageResult
        def __init__(self, key: _Optional[str] = ..., value: _Optional[_Union[StageResult, _Mapping]] = ...) -> None: ...
    class FinalArtifactsEntry(_message.Message):
        __slots__ = ("key", "value")
        KEY_FIELD_NUMBER: _ClassVar[int]
        VALUE_FIELD_NUMBER: _ClassVar[int]
        key: str
        value: str
        def __init__(self, key: _Optional[str] = ..., value: _Optional[str] = ...) -> None: ...
    PIPELINE_ID_FIELD_NUMBER: _ClassVar[int]
    SCENARIO_NAME_FIELD_NUMBER: _ClassVar[int]
    STATUS_FIELD_NUMBER: _ClassVar[int]
    CURRENT_STAGE_FIELD_NUMBER: _ClassVar[int]
    PROGRESS_PERCENT_FIELD_NUMBER: _ClassVar[int]
    PROGRESS_MESSAGE_FIELD_NUMBER: _ClassVar[int]
    STAGES_FIELD_NUMBER: _ClassVar[int]
    STAGE_ORDER_FIELD_NUMBER: _ClassVar[int]
    CONFIG_FIELD_NUMBER: _ClassVar[int]
    STARTED_AT_FIELD_NUMBER: _ClassVar[int]
    COMPLETED_AT_FIELD_NUMBER: _ClassVar[int]
    ERROR_FIELD_NUMBER: _ClassVar[int]
    FINAL_ARTIFACTS_FIELD_NUMBER: _ClassVar[int]
    STOPPED_AFTER_STAGE_FIELD_NUMBER: _ClassVar[int]
    PARENT_PIPELINE_ID_FIELD_NUMBER: _ClassVar[int]
    IDEMPOTENCY_KEY_FIELD_NUMBER: _ClassVar[int]
    pipeline_id: str
    scenario_name: str
    status: _shared_pb2.StageStatus
    current_stage: _shared_pb2.StageName
    progress_percent: int
    progress_message: str
    stages: _containers.MessageMap[str, StageResult]
    stage_order: _containers.RepeatedScalarFieldContainer[_shared_pb2.StageName]
    config: PipelineConfig
    started_at: _timestamp_pb2.Timestamp
    completed_at: _timestamp_pb2.Timestamp
    error: str
    final_artifacts: _containers.ScalarMap[str, str]
    stopped_after_stage: _shared_pb2.StageName
    parent_pipeline_id: str
    idempotency_key: str
    def __init__(self, pipeline_id: _Optional[str] = ..., scenario_name: _Optional[str] = ..., status: _Optional[_Union[_shared_pb2.StageStatus, str]] = ..., current_stage: _Optional[_Union[_shared_pb2.StageName, str]] = ..., progress_percent: _Optional[int] = ..., progress_message: _Optional[str] = ..., stages: _Optional[_Mapping[str, StageResult]] = ..., stage_order: _Optional[_Iterable[_Union[_shared_pb2.StageName, str]]] = ..., config: _Optional[_Union[PipelineConfig, _Mapping]] = ..., started_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., completed_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., error: _Optional[str] = ..., final_artifacts: _Optional[_Mapping[str, str]] = ..., stopped_after_stage: _Optional[_Union[_shared_pb2.StageName, str]] = ..., parent_pipeline_id: _Optional[str] = ..., idempotency_key: _Optional[str] = ...) -> None: ...

class PipelineRunRequest(_message.Message):
    __slots__ = ("config",)
    CONFIG_FIELD_NUMBER: _ClassVar[int]
    config: PipelineConfig
    def __init__(self, config: _Optional[_Union[PipelineConfig, _Mapping]] = ...) -> None: ...

class PipelineRunResponse(_message.Message):
    __slots__ = ("pipeline_id", "status_url", "message")
    PIPELINE_ID_FIELD_NUMBER: _ClassVar[int]
    STATUS_URL_FIELD_NUMBER: _ClassVar[int]
    MESSAGE_FIELD_NUMBER: _ClassVar[int]
    pipeline_id: str
    status_url: str
    message: str
    def __init__(self, pipeline_id: _Optional[str] = ..., status_url: _Optional[str] = ..., message: _Optional[str] = ...) -> None: ...

class PipelineCancelResponse(_message.Message):
    __slots__ = ("status", "message")
    STATUS_FIELD_NUMBER: _ClassVar[int]
    MESSAGE_FIELD_NUMBER: _ClassVar[int]
    status: str
    message: str
    def __init__(self, status: _Optional[str] = ..., message: _Optional[str] = ...) -> None: ...

class PipelineResumeResponse(_message.Message):
    __slots__ = ("pipeline_id", "parent_pipeline_id", "status_url", "resume_from_stage", "message")
    PIPELINE_ID_FIELD_NUMBER: _ClassVar[int]
    PARENT_PIPELINE_ID_FIELD_NUMBER: _ClassVar[int]
    STATUS_URL_FIELD_NUMBER: _ClassVar[int]
    RESUME_FROM_STAGE_FIELD_NUMBER: _ClassVar[int]
    MESSAGE_FIELD_NUMBER: _ClassVar[int]
    pipeline_id: str
    parent_pipeline_id: str
    status_url: str
    resume_from_stage: _shared_pb2.StageName
    message: str
    def __init__(self, pipeline_id: _Optional[str] = ..., parent_pipeline_id: _Optional[str] = ..., status_url: _Optional[str] = ..., resume_from_stage: _Optional[_Union[_shared_pb2.StageName, str]] = ..., message: _Optional[str] = ...) -> None: ...

class PipelineListItem(_message.Message):
    __slots__ = ("pipeline_id", "scenario_name", "status", "progress_percent", "current_stage", "created_at", "updated_at", "completed_at", "can_resume")
    PIPELINE_ID_FIELD_NUMBER: _ClassVar[int]
    SCENARIO_NAME_FIELD_NUMBER: _ClassVar[int]
    STATUS_FIELD_NUMBER: _ClassVar[int]
    PROGRESS_PERCENT_FIELD_NUMBER: _ClassVar[int]
    CURRENT_STAGE_FIELD_NUMBER: _ClassVar[int]
    CREATED_AT_FIELD_NUMBER: _ClassVar[int]
    UPDATED_AT_FIELD_NUMBER: _ClassVar[int]
    COMPLETED_AT_FIELD_NUMBER: _ClassVar[int]
    CAN_RESUME_FIELD_NUMBER: _ClassVar[int]
    pipeline_id: str
    scenario_name: str
    status: _shared_pb2.StageStatus
    progress_percent: int
    current_stage: _shared_pb2.StageName
    created_at: _timestamp_pb2.Timestamp
    updated_at: _timestamp_pb2.Timestamp
    completed_at: _timestamp_pb2.Timestamp
    can_resume: bool
    def __init__(self, pipeline_id: _Optional[str] = ..., scenario_name: _Optional[str] = ..., status: _Optional[_Union[_shared_pb2.StageStatus, str]] = ..., progress_percent: _Optional[int] = ..., current_stage: _Optional[_Union[_shared_pb2.StageName, str]] = ..., created_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., updated_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., completed_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., can_resume: _Optional[bool] = ...) -> None: ...

class PipelineListResponse(_message.Message):
    __slots__ = ("pipelines", "total")
    PIPELINES_FIELD_NUMBER: _ClassVar[int]
    TOTAL_FIELD_NUMBER: _ClassVar[int]
    pipelines: _containers.RepeatedCompositeFieldContainer[PipelineListItem]
    total: int
    def __init__(self, pipelines: _Optional[_Iterable[_Union[PipelineListItem, _Mapping]]] = ..., total: _Optional[int] = ...) -> None: ...

class GenerateResponse(_message.Message):
    __slots__ = ("build_id", "pipeline_id", "status", "scenario_name", "desktop_path", "detected_metadata", "install_instructions", "test_command", "status_url")
    BUILD_ID_FIELD_NUMBER: _ClassVar[int]
    PIPELINE_ID_FIELD_NUMBER: _ClassVar[int]
    STATUS_FIELD_NUMBER: _ClassVar[int]
    SCENARIO_NAME_FIELD_NUMBER: _ClassVar[int]
    DESKTOP_PATH_FIELD_NUMBER: _ClassVar[int]
    DETECTED_METADATA_FIELD_NUMBER: _ClassVar[int]
    INSTALL_INSTRUCTIONS_FIELD_NUMBER: _ClassVar[int]
    TEST_COMMAND_FIELD_NUMBER: _ClassVar[int]
    STATUS_URL_FIELD_NUMBER: _ClassVar[int]
    build_id: str
    pipeline_id: str
    status: str
    scenario_name: str
    desktop_path: str
    detected_metadata: _struct_pb2.Struct
    install_instructions: str
    test_command: str
    status_url: str
    def __init__(self, build_id: _Optional[str] = ..., pipeline_id: _Optional[str] = ..., status: _Optional[str] = ..., scenario_name: _Optional[str] = ..., desktop_path: _Optional[str] = ..., detected_metadata: _Optional[_Union[_struct_pb2.Struct, _Mapping]] = ..., install_instructions: _Optional[str] = ..., test_command: _Optional[str] = ..., status_url: _Optional[str] = ...) -> None: ...
