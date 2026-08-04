import datetime

from buf.validate import validate_pb2 as _validate_pb2
from google.protobuf import timestamp_pb2 as _timestamp_pb2
from scenario_to_desktop.v1.shared import common_pb2 as _common_pb2
from scenario_to_desktop.v1.shared import metadata_pb2 as _metadata_pb2
from scenario_to_desktop.v1.shared import operation_results_pb2 as _operation_results_pb2
from scenario_to_desktop.v1.shared import preflight_results_pb2 as _preflight_results_pb2
from scenario_to_desktop.v1.shared import update_config_pb2 as _update_config_pb2
from google.protobuf.internal import containers as _containers
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Iterable as _Iterable, Mapping as _Mapping
from typing import ClassVar as _ClassVar, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class PipelineConfig(_message.Message):
    __slots__ = ("scenario_name", "platforms", "skip_preflight", "skip_smoke_test", "stop_on_failure", "deployment_mode", "framework", "template_type", "webhook_url", "proxy_url", "bundle_manifest_path", "resource_artifact_root", "location_mode", "clean", "sign", "publish", "distribute", "distribution_targets", "version", "preflight_timeout_seconds", "preflight_secrets", "stop_after_stage", "resume_from_stage", "parent_pipeline_id", "idempotency_key", "stages", "artifact_trust_mode", "update_config")
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
    RESOURCE_ARTIFACT_ROOT_FIELD_NUMBER: _ClassVar[int]
    LOCATION_MODE_FIELD_NUMBER: _ClassVar[int]
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
    STAGES_FIELD_NUMBER: _ClassVar[int]
    ARTIFACT_TRUST_MODE_FIELD_NUMBER: _ClassVar[int]
    UPDATE_CONFIG_FIELD_NUMBER: _ClassVar[int]
    scenario_name: str
    platforms: _containers.RepeatedScalarFieldContainer[_common_pb2.Platform]
    skip_preflight: bool
    skip_smoke_test: bool
    stop_on_failure: bool
    deployment_mode: _common_pb2.DeploymentMode
    framework: _common_pb2.Framework
    template_type: _common_pb2.TemplateType
    webhook_url: str
    proxy_url: str
    bundle_manifest_path: str
    resource_artifact_root: str
    location_mode: str
    clean: bool
    sign: bool
    publish: bool
    distribute: bool
    distribution_targets: _containers.RepeatedScalarFieldContainer[str]
    version: str
    preflight_timeout_seconds: int
    preflight_secrets: _containers.ScalarMap[str, str]
    stop_after_stage: _common_pb2.StageName
    resume_from_stage: _common_pb2.StageName
    parent_pipeline_id: str
    idempotency_key: str
    stages: _containers.RepeatedScalarFieldContainer[_common_pb2.StageName]
    artifact_trust_mode: str
    update_config: _update_config_pb2.UpdateConfig
    def __init__(self, scenario_name: _Optional[str] = ..., platforms: _Optional[_Iterable[_Union[_common_pb2.Platform, str]]] = ..., skip_preflight: _Optional[bool] = ..., skip_smoke_test: _Optional[bool] = ..., stop_on_failure: _Optional[bool] = ..., deployment_mode: _Optional[_Union[_common_pb2.DeploymentMode, str]] = ..., framework: _Optional[_Union[_common_pb2.Framework, str]] = ..., template_type: _Optional[_Union[_common_pb2.TemplateType, str]] = ..., webhook_url: _Optional[str] = ..., proxy_url: _Optional[str] = ..., bundle_manifest_path: _Optional[str] = ..., resource_artifact_root: _Optional[str] = ..., location_mode: _Optional[str] = ..., clean: _Optional[bool] = ..., sign: _Optional[bool] = ..., publish: _Optional[bool] = ..., distribute: _Optional[bool] = ..., distribution_targets: _Optional[_Iterable[str]] = ..., version: _Optional[str] = ..., preflight_timeout_seconds: _Optional[int] = ..., preflight_secrets: _Optional[_Mapping[str, str]] = ..., stop_after_stage: _Optional[_Union[_common_pb2.StageName, str]] = ..., resume_from_stage: _Optional[_Union[_common_pb2.StageName, str]] = ..., parent_pipeline_id: _Optional[str] = ..., idempotency_key: _Optional[str] = ..., stages: _Optional[_Iterable[_Union[_common_pb2.StageName, str]]] = ..., artifact_trust_mode: _Optional[str] = ..., update_config: _Optional[_Union[_update_config_pb2.UpdateConfig, _Mapping]] = ...) -> None: ...

class StageResult(_message.Message):
    __slots__ = ("stage", "status", "started_at", "completed_at", "error", "logs", "details")
    STAGE_FIELD_NUMBER: _ClassVar[int]
    STATUS_FIELD_NUMBER: _ClassVar[int]
    STARTED_AT_FIELD_NUMBER: _ClassVar[int]
    COMPLETED_AT_FIELD_NUMBER: _ClassVar[int]
    ERROR_FIELD_NUMBER: _ClassVar[int]
    LOGS_FIELD_NUMBER: _ClassVar[int]
    DETAILS_FIELD_NUMBER: _ClassVar[int]
    stage: _common_pb2.StageName
    status: _common_pb2.StageStatus
    started_at: _timestamp_pb2.Timestamp
    completed_at: _timestamp_pb2.Timestamp
    error: str
    logs: _containers.RepeatedScalarFieldContainer[str]
    details: StageDetails
    def __init__(self, stage: _Optional[_Union[_common_pb2.StageName, str]] = ..., status: _Optional[_Union[_common_pb2.StageStatus, str]] = ..., started_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., completed_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., error: _Optional[str] = ..., logs: _Optional[_Iterable[str]] = ..., details: _Optional[_Union[StageDetails, _Mapping]] = ...) -> None: ...

class StageDetails(_message.Message):
    __slots__ = ("resolve_deployment", "bundle", "preflight", "generate", "build", "smoke_test", "deploy")
    RESOLVE_DEPLOYMENT_FIELD_NUMBER: _ClassVar[int]
    BUNDLE_FIELD_NUMBER: _ClassVar[int]
    PREFLIGHT_FIELD_NUMBER: _ClassVar[int]
    GENERATE_FIELD_NUMBER: _ClassVar[int]
    BUILD_FIELD_NUMBER: _ClassVar[int]
    SMOKE_TEST_FIELD_NUMBER: _ClassVar[int]
    DEPLOY_FIELD_NUMBER: _ClassVar[int]
    resolve_deployment: ResourceDeploymentPlan
    bundle: BundleStageDetails
    preflight: _preflight_results_pb2.PreflightResponse
    generate: GenerateResponse
    build: _operation_results_pb2.BuildStatusResponse
    smoke_test: _operation_results_pb2.SmokeTestStatusResponse
    deploy: DeployStageDetails
    def __init__(self, resolve_deployment: _Optional[_Union[ResourceDeploymentPlan, _Mapping]] = ..., bundle: _Optional[_Union[BundleStageDetails, _Mapping]] = ..., preflight: _Optional[_Union[_preflight_results_pb2.PreflightResponse, _Mapping]] = ..., generate: _Optional[_Union[GenerateResponse, _Mapping]] = ..., build: _Optional[_Union[_operation_results_pb2.BuildStatusResponse, _Mapping]] = ..., smoke_test: _Optional[_Union[_operation_results_pb2.SmokeTestStatusResponse, _Mapping]] = ..., deploy: _Optional[_Union[DeployStageDetails, _Mapping]] = ...) -> None: ...

class ResourceDeploymentPlan(_message.Message):
    __slots__ = ("schema_version", "resources", "artifact_trust_mode", "promotable", "host_requirements")
    SCHEMA_VERSION_FIELD_NUMBER: _ClassVar[int]
    RESOURCES_FIELD_NUMBER: _ClassVar[int]
    ARTIFACT_TRUST_MODE_FIELD_NUMBER: _ClassVar[int]
    PROMOTABLE_FIELD_NUMBER: _ClassVar[int]
    HOST_REQUIREMENTS_FIELD_NUMBER: _ClassVar[int]
    schema_version: str
    resources: _containers.RepeatedCompositeFieldContainer[ResourceDeploymentPlanItem]
    artifact_trust_mode: str
    promotable: bool
    host_requirements: _containers.RepeatedCompositeFieldContainer[HostRequirementPlanItem]
    def __init__(self, schema_version: _Optional[str] = ..., resources: _Optional[_Iterable[_Union[ResourceDeploymentPlanItem, _Mapping]]] = ..., artifact_trust_mode: _Optional[str] = ..., promotable: _Optional[bool] = ..., host_requirements: _Optional[_Iterable[_Union[HostRequirementPlanItem, _Mapping]]] = ...) -> None: ...

class HostRequirementPlanItem(_message.Message):
    __slots__ = ("name", "kind", "os", "privilege", "bundling", "required", "verdict", "reason", "provenance")
    NAME_FIELD_NUMBER: _ClassVar[int]
    KIND_FIELD_NUMBER: _ClassVar[int]
    OS_FIELD_NUMBER: _ClassVar[int]
    PRIVILEGE_FIELD_NUMBER: _ClassVar[int]
    BUNDLING_FIELD_NUMBER: _ClassVar[int]
    REQUIRED_FIELD_NUMBER: _ClassVar[int]
    VERDICT_FIELD_NUMBER: _ClassVar[int]
    REASON_FIELD_NUMBER: _ClassVar[int]
    PROVENANCE_FIELD_NUMBER: _ClassVar[int]
    name: str
    kind: str
    os: str
    privilege: str
    bundling: str
    required: bool
    verdict: str
    reason: str
    provenance: _containers.RepeatedScalarFieldContainer[str]
    def __init__(self, name: _Optional[str] = ..., kind: _Optional[str] = ..., os: _Optional[str] = ..., privilege: _Optional[str] = ..., bundling: _Optional[str] = ..., required: _Optional[bool] = ..., verdict: _Optional[str] = ..., reason: _Optional[str] = ..., provenance: _Optional[_Iterable[str]] = ...) -> None: ...

class ResourceDeploymentPlanItem(_message.Message):
    __slots__ = ("requested_resource", "resource", "os", "architecture", "mode", "support", "requires", "limitations", "evidence", "selected_fallback", "artifact", "files", "service", "privilege", "bundling", "eligibility", "eligibility_reason")
    REQUESTED_RESOURCE_FIELD_NUMBER: _ClassVar[int]
    RESOURCE_FIELD_NUMBER: _ClassVar[int]
    OS_FIELD_NUMBER: _ClassVar[int]
    ARCHITECTURE_FIELD_NUMBER: _ClassVar[int]
    MODE_FIELD_NUMBER: _ClassVar[int]
    SUPPORT_FIELD_NUMBER: _ClassVar[int]
    REQUIRES_FIELD_NUMBER: _ClassVar[int]
    LIMITATIONS_FIELD_NUMBER: _ClassVar[int]
    EVIDENCE_FIELD_NUMBER: _ClassVar[int]
    SELECTED_FALLBACK_FIELD_NUMBER: _ClassVar[int]
    ARTIFACT_FIELD_NUMBER: _ClassVar[int]
    FILES_FIELD_NUMBER: _ClassVar[int]
    SERVICE_FIELD_NUMBER: _ClassVar[int]
    PRIVILEGE_FIELD_NUMBER: _ClassVar[int]
    BUNDLING_FIELD_NUMBER: _ClassVar[int]
    ELIGIBILITY_FIELD_NUMBER: _ClassVar[int]
    ELIGIBILITY_REASON_FIELD_NUMBER: _ClassVar[int]
    requested_resource: str
    resource: str
    os: str
    architecture: str
    mode: str
    support: str
    requires: _containers.RepeatedScalarFieldContainer[str]
    limitations: _containers.RepeatedScalarFieldContainer[str]
    evidence: _containers.RepeatedScalarFieldContainer[str]
    selected_fallback: ResourceDeploymentFallback
    artifact: str
    files: _containers.RepeatedCompositeFieldContainer[ResourceDeploymentArtifact]
    service: ResourceDeploymentService
    privilege: str
    bundling: str
    eligibility: str
    eligibility_reason: str
    def __init__(self, requested_resource: _Optional[str] = ..., resource: _Optional[str] = ..., os: _Optional[str] = ..., architecture: _Optional[str] = ..., mode: _Optional[str] = ..., support: _Optional[str] = ..., requires: _Optional[_Iterable[str]] = ..., limitations: _Optional[_Iterable[str]] = ..., evidence: _Optional[_Iterable[str]] = ..., selected_fallback: _Optional[_Union[ResourceDeploymentFallback, _Mapping]] = ..., artifact: _Optional[str] = ..., files: _Optional[_Iterable[_Union[ResourceDeploymentArtifact, _Mapping]]] = ..., service: _Optional[_Union[ResourceDeploymentService, _Mapping]] = ..., privilege: _Optional[str] = ..., bundling: _Optional[str] = ..., eligibility: _Optional[str] = ..., eligibility_reason: _Optional[str] = ...) -> None: ...

class ResourceDeploymentFallback(_message.Message):
    __slots__ = ("resource", "reason")
    RESOURCE_FIELD_NUMBER: _ClassVar[int]
    REASON_FIELD_NUMBER: _ClassVar[int]
    resource: str
    reason: str
    def __init__(self, resource: _Optional[str] = ..., reason: _Optional[str] = ...) -> None: ...

class ResourceDeploymentArtifact(_message.Message):
    __slots__ = ("name", "sha256")
    NAME_FIELD_NUMBER: _ClassVar[int]
    SHA256_FIELD_NUMBER: _ClassVar[int]
    name: str
    sha256: str
    def __init__(self, name: _Optional[str] = ..., sha256: _Optional[str] = ...) -> None: ...

class ResourceDeploymentService(_message.Message):
    __slots__ = ("provider_policy", "artifact", "version", "sha256", "arguments", "environment", "ports", "health_checks", "files", "config")
    class EnvironmentEntry(_message.Message):
        __slots__ = ("key", "value")
        KEY_FIELD_NUMBER: _ClassVar[int]
        VALUE_FIELD_NUMBER: _ClassVar[int]
        key: str
        value: str
        def __init__(self, key: _Optional[str] = ..., value: _Optional[str] = ...) -> None: ...
    PROVIDER_POLICY_FIELD_NUMBER: _ClassVar[int]
    ARTIFACT_FIELD_NUMBER: _ClassVar[int]
    VERSION_FIELD_NUMBER: _ClassVar[int]
    SHA256_FIELD_NUMBER: _ClassVar[int]
    ARGUMENTS_FIELD_NUMBER: _ClassVar[int]
    ENVIRONMENT_FIELD_NUMBER: _ClassVar[int]
    PORTS_FIELD_NUMBER: _ClassVar[int]
    HEALTH_CHECKS_FIELD_NUMBER: _ClassVar[int]
    FILES_FIELD_NUMBER: _ClassVar[int]
    CONFIG_FIELD_NUMBER: _ClassVar[int]
    provider_policy: ResourceProviderPolicy
    artifact: str
    version: str
    sha256: str
    arguments: _containers.RepeatedScalarFieldContainer[str]
    environment: _containers.ScalarMap[str, str]
    ports: _containers.RepeatedCompositeFieldContainer[ResourceDeploymentServicePort]
    health_checks: _containers.RepeatedCompositeFieldContainer[ResourceDeploymentHealthCheck]
    files: _containers.RepeatedCompositeFieldContainer[ResourceDeploymentArtifact]
    config: ResourceDeploymentServiceConfig
    def __init__(self, provider_policy: _Optional[_Union[ResourceProviderPolicy, _Mapping]] = ..., artifact: _Optional[str] = ..., version: _Optional[str] = ..., sha256: _Optional[str] = ..., arguments: _Optional[_Iterable[str]] = ..., environment: _Optional[_Mapping[str, str]] = ..., ports: _Optional[_Iterable[_Union[ResourceDeploymentServicePort, _Mapping]]] = ..., health_checks: _Optional[_Iterable[_Union[ResourceDeploymentHealthCheck, _Mapping]]] = ..., files: _Optional[_Iterable[_Union[ResourceDeploymentArtifact, _Mapping]]] = ..., config: _Optional[_Union[ResourceDeploymentServiceConfig, _Mapping]] = ...) -> None: ...

class ResourceProviderPolicy(_message.Message):
    __slots__ = ("default_mode", "target_defaults", "allowed_modes", "shared_reuse_requires_consent", "external_management", "external_access_capabilities")
    class TargetDefaultsEntry(_message.Message):
        __slots__ = ("key", "value")
        KEY_FIELD_NUMBER: _ClassVar[int]
        VALUE_FIELD_NUMBER: _ClassVar[int]
        key: str
        value: str
        def __init__(self, key: _Optional[str] = ..., value: _Optional[str] = ...) -> None: ...
    DEFAULT_MODE_FIELD_NUMBER: _ClassVar[int]
    TARGET_DEFAULTS_FIELD_NUMBER: _ClassVar[int]
    ALLOWED_MODES_FIELD_NUMBER: _ClassVar[int]
    SHARED_REUSE_REQUIRES_CONSENT_FIELD_NUMBER: _ClassVar[int]
    EXTERNAL_MANAGEMENT_FIELD_NUMBER: _ClassVar[int]
    EXTERNAL_ACCESS_CAPABILITIES_FIELD_NUMBER: _ClassVar[int]
    default_mode: str
    target_defaults: _containers.ScalarMap[str, str]
    allowed_modes: _containers.RepeatedScalarFieldContainer[str]
    shared_reuse_requires_consent: bool
    external_management: str
    external_access_capabilities: _containers.RepeatedScalarFieldContainer[str]
    def __init__(self, default_mode: _Optional[str] = ..., target_defaults: _Optional[_Mapping[str, str]] = ..., allowed_modes: _Optional[_Iterable[str]] = ..., shared_reuse_requires_consent: _Optional[bool] = ..., external_management: _Optional[str] = ..., external_access_capabilities: _Optional[_Iterable[str]] = ...) -> None: ...

class ResourceDeploymentServiceConfig(_message.Message):
    __slots__ = ("path", "content")
    PATH_FIELD_NUMBER: _ClassVar[int]
    CONTENT_FIELD_NUMBER: _ClassVar[int]
    path: str
    content: str
    def __init__(self, path: _Optional[str] = ..., content: _Optional[str] = ...) -> None: ...

class ResourceDeploymentServicePort(_message.Message):
    __slots__ = ("name", "host")
    NAME_FIELD_NUMBER: _ClassVar[int]
    HOST_FIELD_NUMBER: _ClassVar[int]
    name: str
    host: int
    def __init__(self, name: _Optional[str] = ..., host: _Optional[int] = ...) -> None: ...

class ResourceDeploymentHealthCheck(_message.Message):
    __slots__ = ("type", "target", "expected_status", "timeout_seconds")
    TYPE_FIELD_NUMBER: _ClassVar[int]
    TARGET_FIELD_NUMBER: _ClassVar[int]
    EXPECTED_STATUS_FIELD_NUMBER: _ClassVar[int]
    TIMEOUT_SECONDS_FIELD_NUMBER: _ClassVar[int]
    type: str
    target: str
    expected_status: _containers.RepeatedScalarFieldContainer[int]
    timeout_seconds: int
    def __init__(self, type: _Optional[str] = ..., target: _Optional[str] = ..., expected_status: _Optional[_Iterable[int]] = ..., timeout_seconds: _Optional[int] = ...) -> None: ...

class BundleStageDetails(_message.Message):
    __slots__ = ("bundle_dir", "manifest_path", "runtime_binaries", "copied_artifacts", "total_size_bytes", "total_size_human", "size_warning")
    class RuntimeBinariesEntry(_message.Message):
        __slots__ = ("key", "value")
        KEY_FIELD_NUMBER: _ClassVar[int]
        VALUE_FIELD_NUMBER: _ClassVar[int]
        key: str
        value: str
        def __init__(self, key: _Optional[str] = ..., value: _Optional[str] = ...) -> None: ...
    BUNDLE_DIR_FIELD_NUMBER: _ClassVar[int]
    MANIFEST_PATH_FIELD_NUMBER: _ClassVar[int]
    RUNTIME_BINARIES_FIELD_NUMBER: _ClassVar[int]
    COPIED_ARTIFACTS_FIELD_NUMBER: _ClassVar[int]
    TOTAL_SIZE_BYTES_FIELD_NUMBER: _ClassVar[int]
    TOTAL_SIZE_HUMAN_FIELD_NUMBER: _ClassVar[int]
    SIZE_WARNING_FIELD_NUMBER: _ClassVar[int]
    bundle_dir: str
    manifest_path: str
    runtime_binaries: _containers.ScalarMap[str, str]
    copied_artifacts: _containers.RepeatedScalarFieldContainer[str]
    total_size_bytes: int
    total_size_human: str
    size_warning: BundleSizeWarning
    def __init__(self, bundle_dir: _Optional[str] = ..., manifest_path: _Optional[str] = ..., runtime_binaries: _Optional[_Mapping[str, str]] = ..., copied_artifacts: _Optional[_Iterable[str]] = ..., total_size_bytes: _Optional[int] = ..., total_size_human: _Optional[str] = ..., size_warning: _Optional[_Union[BundleSizeWarning, _Mapping]] = ...) -> None: ...

class BundleSizeWarning(_message.Message):
    __slots__ = ("level", "message", "total_bytes", "total_human", "large_files")
    LEVEL_FIELD_NUMBER: _ClassVar[int]
    MESSAGE_FIELD_NUMBER: _ClassVar[int]
    TOTAL_BYTES_FIELD_NUMBER: _ClassVar[int]
    TOTAL_HUMAN_FIELD_NUMBER: _ClassVar[int]
    LARGE_FILES_FIELD_NUMBER: _ClassVar[int]
    level: str
    message: str
    total_bytes: int
    total_human: str
    large_files: _containers.RepeatedCompositeFieldContainer[BundleLargeFile]
    def __init__(self, level: _Optional[str] = ..., message: _Optional[str] = ..., total_bytes: _Optional[int] = ..., total_human: _Optional[str] = ..., large_files: _Optional[_Iterable[_Union[BundleLargeFile, _Mapping]]] = ...) -> None: ...

class BundleLargeFile(_message.Message):
    __slots__ = ("path", "size_bytes", "size_human")
    PATH_FIELD_NUMBER: _ClassVar[int]
    SIZE_BYTES_FIELD_NUMBER: _ClassVar[int]
    SIZE_HUMAN_FIELD_NUMBER: _ClassVar[int]
    path: str
    size_bytes: int
    size_human: str
    def __init__(self, path: _Optional[str] = ..., size_bytes: _Optional[int] = ..., size_human: _Optional[str] = ...) -> None: ...

class DeployStageDetails(_message.Message):
    __slots__ = ("artifacts", "update_url")
    ARTIFACTS_FIELD_NUMBER: _ClassVar[int]
    UPDATE_URL_FIELD_NUMBER: _ClassVar[int]
    artifacts: _containers.RepeatedCompositeFieldContainer[DeployArtifactResult]
    update_url: str
    def __init__(self, artifacts: _Optional[_Iterable[_Union[DeployArtifactResult, _Mapping]]] = ..., update_url: _Optional[str] = ...) -> None: ...

class DeployArtifactResult(_message.Message):
    __slots__ = ("artifact_id", "platform")
    ARTIFACT_ID_FIELD_NUMBER: _ClassVar[int]
    PLATFORM_FIELD_NUMBER: _ClassVar[int]
    artifact_id: int
    platform: _common_pb2.Platform
    def __init__(self, artifact_id: _Optional[int] = ..., platform: _Optional[_Union[_common_pb2.Platform, str]] = ...) -> None: ...

class PipelineStatus(_message.Message):
    __slots__ = ("pipeline_id", "scenario_name", "status", "current_stage", "progress_percent", "progress_message", "current_state", "stages", "stage_order", "config", "started_at", "completed_at", "error", "final_artifacts", "stopped_after_stage", "parent_pipeline_id", "idempotency_key")
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
    CURRENT_STATE_FIELD_NUMBER: _ClassVar[int]
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
    status: _common_pb2.StageStatus
    current_stage: _common_pb2.StageName
    progress_percent: int
    progress_message: str
    current_state: str
    stages: _containers.MessageMap[str, StageResult]
    stage_order: _containers.RepeatedScalarFieldContainer[_common_pb2.StageName]
    config: PipelineConfig
    started_at: _timestamp_pb2.Timestamp
    completed_at: _timestamp_pb2.Timestamp
    error: str
    final_artifacts: _containers.ScalarMap[str, str]
    stopped_after_stage: _common_pb2.StageName
    parent_pipeline_id: str
    idempotency_key: str
    def __init__(self, pipeline_id: _Optional[str] = ..., scenario_name: _Optional[str] = ..., status: _Optional[_Union[_common_pb2.StageStatus, str]] = ..., current_stage: _Optional[_Union[_common_pb2.StageName, str]] = ..., progress_percent: _Optional[int] = ..., progress_message: _Optional[str] = ..., current_state: _Optional[str] = ..., stages: _Optional[_Mapping[str, StageResult]] = ..., stage_order: _Optional[_Iterable[_Union[_common_pb2.StageName, str]]] = ..., config: _Optional[_Union[PipelineConfig, _Mapping]] = ..., started_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., completed_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., error: _Optional[str] = ..., final_artifacts: _Optional[_Mapping[str, str]] = ..., stopped_after_stage: _Optional[_Union[_common_pb2.StageName, str]] = ..., parent_pipeline_id: _Optional[str] = ..., idempotency_key: _Optional[str] = ...) -> None: ...

class PipelineRunRequest(_message.Message):
    __slots__ = ("config",)
    CONFIG_FIELD_NUMBER: _ClassVar[int]
    config: PipelineConfig
    def __init__(self, config: _Optional[_Union[PipelineConfig, _Mapping]] = ...) -> None: ...

class PipelineRunResponse(_message.Message):
    __slots__ = ("pipeline_id", "message")
    PIPELINE_ID_FIELD_NUMBER: _ClassVar[int]
    MESSAGE_FIELD_NUMBER: _ClassVar[int]
    pipeline_id: str
    message: str
    def __init__(self, pipeline_id: _Optional[str] = ..., message: _Optional[str] = ...) -> None: ...

class PipelineGetRequest(_message.Message):
    __slots__ = ("pipeline_id",)
    PIPELINE_ID_FIELD_NUMBER: _ClassVar[int]
    pipeline_id: str
    def __init__(self, pipeline_id: _Optional[str] = ...) -> None: ...

class PipelineResumeRequest(_message.Message):
    __slots__ = ("pipeline_id", "config")
    PIPELINE_ID_FIELD_NUMBER: _ClassVar[int]
    CONFIG_FIELD_NUMBER: _ClassVar[int]
    pipeline_id: str
    config: PipelineConfig
    def __init__(self, pipeline_id: _Optional[str] = ..., config: _Optional[_Union[PipelineConfig, _Mapping]] = ...) -> None: ...

class PipelineCancelRequest(_message.Message):
    __slots__ = ("pipeline_id",)
    PIPELINE_ID_FIELD_NUMBER: _ClassVar[int]
    pipeline_id: str
    def __init__(self, pipeline_id: _Optional[str] = ...) -> None: ...

class PipelineListRequest(_message.Message):
    __slots__ = ("scenario_name",)
    SCENARIO_NAME_FIELD_NUMBER: _ClassVar[int]
    scenario_name: str
    def __init__(self, scenario_name: _Optional[str] = ...) -> None: ...

class PipelineCancelResponse(_message.Message):
    __slots__ = ("status", "message")
    STATUS_FIELD_NUMBER: _ClassVar[int]
    MESSAGE_FIELD_NUMBER: _ClassVar[int]
    status: str
    message: str
    def __init__(self, status: _Optional[str] = ..., message: _Optional[str] = ...) -> None: ...

class PipelineResumeResponse(_message.Message):
    __slots__ = ("pipeline_id", "parent_pipeline_id", "resume_from_stage", "message")
    PIPELINE_ID_FIELD_NUMBER: _ClassVar[int]
    PARENT_PIPELINE_ID_FIELD_NUMBER: _ClassVar[int]
    RESUME_FROM_STAGE_FIELD_NUMBER: _ClassVar[int]
    MESSAGE_FIELD_NUMBER: _ClassVar[int]
    pipeline_id: str
    parent_pipeline_id: str
    resume_from_stage: _common_pb2.StageName
    message: str
    def __init__(self, pipeline_id: _Optional[str] = ..., parent_pipeline_id: _Optional[str] = ..., resume_from_stage: _Optional[_Union[_common_pb2.StageName, str]] = ..., message: _Optional[str] = ...) -> None: ...

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
    status: _common_pb2.StageStatus
    progress_percent: int
    current_stage: _common_pb2.StageName
    created_at: _timestamp_pb2.Timestamp
    updated_at: _timestamp_pb2.Timestamp
    completed_at: _timestamp_pb2.Timestamp
    can_resume: bool
    def __init__(self, pipeline_id: _Optional[str] = ..., scenario_name: _Optional[str] = ..., status: _Optional[_Union[_common_pb2.StageStatus, str]] = ..., progress_percent: _Optional[int] = ..., current_stage: _Optional[_Union[_common_pb2.StageName, str]] = ..., created_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., updated_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., completed_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., can_resume: _Optional[bool] = ...) -> None: ...

class PipelineListResponse(_message.Message):
    __slots__ = ("pipelines", "total")
    PIPELINES_FIELD_NUMBER: _ClassVar[int]
    TOTAL_FIELD_NUMBER: _ClassVar[int]
    pipelines: _containers.RepeatedCompositeFieldContainer[PipelineListItem]
    total: int
    def __init__(self, pipelines: _Optional[_Iterable[_Union[PipelineListItem, _Mapping]]] = ..., total: _Optional[int] = ...) -> None: ...

class ScenarioPipelineRequest(_message.Message):
    __slots__ = ("scenario_name",)
    SCENARIO_NAME_FIELD_NUMBER: _ClassVar[int]
    scenario_name: str
    def __init__(self, scenario_name: _Optional[str] = ...) -> None: ...

class GetActivePipelineRequest(_message.Message):
    __slots__ = ("scenario_name", "auto_create")
    SCENARIO_NAME_FIELD_NUMBER: _ClassVar[int]
    AUTO_CREATE_FIELD_NUMBER: _ClassVar[int]
    scenario_name: str
    auto_create: bool
    def __init__(self, scenario_name: _Optional[str] = ..., auto_create: _Optional[bool] = ...) -> None: ...

class ActivePipelineResponse(_message.Message):
    __slots__ = ("pipeline", "created")
    PIPELINE_FIELD_NUMBER: _ClassVar[int]
    CREATED_FIELD_NUMBER: _ClassVar[int]
    pipeline: PipelineStatus
    created: bool
    def __init__(self, pipeline: _Optional[_Union[PipelineStatus, _Mapping]] = ..., created: _Optional[bool] = ...) -> None: ...

class CreatePipelineRequest(_message.Message):
    __slots__ = ("scenario_name", "config")
    SCENARIO_NAME_FIELD_NUMBER: _ClassVar[int]
    CONFIG_FIELD_NUMBER: _ClassVar[int]
    scenario_name: str
    config: PipelineConfig
    def __init__(self, scenario_name: _Optional[str] = ..., config: _Optional[_Union[PipelineConfig, _Mapping]] = ...) -> None: ...

class CreatePipelineResponse(_message.Message):
    __slots__ = ("pipeline", "archived_pipeline_id")
    PIPELINE_FIELD_NUMBER: _ClassVar[int]
    ARCHIVED_PIPELINE_ID_FIELD_NUMBER: _ClassVar[int]
    pipeline: PipelineStatus
    archived_pipeline_id: str
    def __init__(self, pipeline: _Optional[_Union[PipelineStatus, _Mapping]] = ..., archived_pipeline_id: _Optional[str] = ...) -> None: ...

class ResetPipelineResponse(_message.Message):
    __slots__ = ("archived_pipeline_id", "cleared")
    ARCHIVED_PIPELINE_ID_FIELD_NUMBER: _ClassVar[int]
    CLEARED_FIELD_NUMBER: _ClassVar[int]
    archived_pipeline_id: str
    cleared: bool
    def __init__(self, archived_pipeline_id: _Optional[str] = ..., cleared: _Optional[bool] = ...) -> None: ...

class PipelineHistoryRequest(_message.Message):
    __slots__ = ("scenario_name", "limit")
    SCENARIO_NAME_FIELD_NUMBER: _ClassVar[int]
    LIMIT_FIELD_NUMBER: _ClassVar[int]
    scenario_name: str
    limit: int
    def __init__(self, scenario_name: _Optional[str] = ..., limit: _Optional[int] = ...) -> None: ...

class PipelineHistoryResponse(_message.Message):
    __slots__ = ("pipelines", "total")
    PIPELINES_FIELD_NUMBER: _ClassVar[int]
    TOTAL_FIELD_NUMBER: _ClassVar[int]
    pipelines: _containers.RepeatedCompositeFieldContainer[PipelineStatus]
    total: int
    def __init__(self, pipelines: _Optional[_Iterable[_Union[PipelineStatus, _Mapping]]] = ..., total: _Optional[int] = ...) -> None: ...

class StartActivePipelineRequest(_message.Message):
    __slots__ = ("scenario_name", "config_overrides")
    SCENARIO_NAME_FIELD_NUMBER: _ClassVar[int]
    CONFIG_OVERRIDES_FIELD_NUMBER: _ClassVar[int]
    scenario_name: str
    config_overrides: PipelineConfig
    def __init__(self, scenario_name: _Optional[str] = ..., config_overrides: _Optional[_Union[PipelineConfig, _Mapping]] = ...) -> None: ...

class StartActivePipelineResponse(_message.Message):
    __slots__ = ("pipeline", "message")
    PIPELINE_FIELD_NUMBER: _ClassVar[int]
    MESSAGE_FIELD_NUMBER: _ClassVar[int]
    pipeline: PipelineStatus
    message: str
    def __init__(self, pipeline: _Optional[_Union[PipelineStatus, _Mapping]] = ..., message: _Optional[str] = ...) -> None: ...

class BundleCleanRequest(_message.Message):
    __slots__ = ("scenario_name", "location_mode", "pipeline_id")
    SCENARIO_NAME_FIELD_NUMBER: _ClassVar[int]
    LOCATION_MODE_FIELD_NUMBER: _ClassVar[int]
    PIPELINE_ID_FIELD_NUMBER: _ClassVar[int]
    scenario_name: str
    location_mode: str
    pipeline_id: str
    def __init__(self, scenario_name: _Optional[str] = ..., location_mode: _Optional[str] = ..., pipeline_id: _Optional[str] = ...) -> None: ...

class BundleCleanResponse(_message.Message):
    __slots__ = ("scenario_name", "location_mode", "pipeline_id", "path", "removed")
    SCENARIO_NAME_FIELD_NUMBER: _ClassVar[int]
    LOCATION_MODE_FIELD_NUMBER: _ClassVar[int]
    PIPELINE_ID_FIELD_NUMBER: _ClassVar[int]
    PATH_FIELD_NUMBER: _ClassVar[int]
    REMOVED_FIELD_NUMBER: _ClassVar[int]
    scenario_name: str
    location_mode: str
    pipeline_id: str
    path: str
    removed: bool
    def __init__(self, scenario_name: _Optional[str] = ..., location_mode: _Optional[str] = ..., pipeline_id: _Optional[str] = ..., path: _Optional[str] = ..., removed: _Optional[bool] = ...) -> None: ...

class GenerateResponse(_message.Message):
    __slots__ = ("pipeline_id", "status", "scenario_name", "desktop_path", "detected_metadata", "install_instructions", "test_command")
    PIPELINE_ID_FIELD_NUMBER: _ClassVar[int]
    STATUS_FIELD_NUMBER: _ClassVar[int]
    SCENARIO_NAME_FIELD_NUMBER: _ClassVar[int]
    DESKTOP_PATH_FIELD_NUMBER: _ClassVar[int]
    DETECTED_METADATA_FIELD_NUMBER: _ClassVar[int]
    INSTALL_INSTRUCTIONS_FIELD_NUMBER: _ClassVar[int]
    TEST_COMMAND_FIELD_NUMBER: _ClassVar[int]
    pipeline_id: str
    status: str
    scenario_name: str
    desktop_path: str
    detected_metadata: _metadata_pb2.ScenarioMetadata
    install_instructions: str
    test_command: str
    def __init__(self, pipeline_id: _Optional[str] = ..., status: _Optional[str] = ..., scenario_name: _Optional[str] = ..., desktop_path: _Optional[str] = ..., detected_metadata: _Optional[_Union[_metadata_pb2.ScenarioMetadata, _Mapping]] = ..., install_instructions: _Optional[str] = ..., test_command: _Optional[str] = ...) -> None: ...
