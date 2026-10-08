from google.protobuf.internal import containers as _containers
from google.protobuf.internal import enum_type_wrapper as _enum_type_wrapper
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Iterable as _Iterable, Mapping as _Mapping
from typing import ClassVar as _ClassVar, Optional as _Optional

DESCRIPTOR: _descriptor.FileDescriptor

class Platform(int, metaclass=_enum_type_wrapper.EnumTypeWrapper):
    __slots__ = ()
    PLATFORM_UNSPECIFIED: _ClassVar[Platform]
    PLATFORM_WIN: _ClassVar[Platform]
    PLATFORM_MAC: _ClassVar[Platform]
    PLATFORM_LINUX: _ClassVar[Platform]

class StageName(int, metaclass=_enum_type_wrapper.EnumTypeWrapper):
    __slots__ = ()
    STAGE_NAME_UNSPECIFIED: _ClassVar[StageName]
    STAGE_NAME_BUNDLE: _ClassVar[StageName]
    STAGE_NAME_PREFLIGHT: _ClassVar[StageName]
    STAGE_NAME_GENERATE: _ClassVar[StageName]
    STAGE_NAME_BUILD: _ClassVar[StageName]
    STAGE_NAME_SMOKE_TEST: _ClassVar[StageName]
    STAGE_NAME_DISTRIBUTION: _ClassVar[StageName]
    STAGE_NAME_DEPLOY: _ClassVar[StageName]
    STAGE_NAME_RESOLVE_DEPLOYMENT: _ClassVar[StageName]

class StageStatus(int, metaclass=_enum_type_wrapper.EnumTypeWrapper):
    __slots__ = ()
    STAGE_STATUS_UNSPECIFIED: _ClassVar[StageStatus]
    STAGE_STATUS_PENDING: _ClassVar[StageStatus]
    STAGE_STATUS_RUNNING: _ClassVar[StageStatus]
    STAGE_STATUS_COMPLETED: _ClassVar[StageStatus]
    STAGE_STATUS_FAILED: _ClassVar[StageStatus]
    STAGE_STATUS_SKIPPED: _ClassVar[StageStatus]
    STAGE_STATUS_CANCELLED: _ClassVar[StageStatus]
    STAGE_STATUS_IDLE: _ClassVar[StageStatus]

class BuildStatus(int, metaclass=_enum_type_wrapper.EnumTypeWrapper):
    __slots__ = ()
    BUILD_STATUS_UNSPECIFIED: _ClassVar[BuildStatus]
    BUILD_STATUS_BUILDING: _ClassVar[BuildStatus]
    BUILD_STATUS_READY: _ClassVar[BuildStatus]
    BUILD_STATUS_PARTIAL: _ClassVar[BuildStatus]
    BUILD_STATUS_FAILED: _ClassVar[BuildStatus]

class UploadStatus(int, metaclass=_enum_type_wrapper.EnumTypeWrapper):
    __slots__ = ()
    UPLOAD_STATUS_UNSPECIFIED: _ClassVar[UploadStatus]
    UPLOAD_STATUS_PENDING: _ClassVar[UploadStatus]
    UPLOAD_STATUS_UPLOADING: _ClassVar[UploadStatus]
    UPLOAD_STATUS_COMPLETED: _ClassVar[UploadStatus]
    UPLOAD_STATUS_FAILED: _ClassVar[UploadStatus]

class DeploymentMode(int, metaclass=_enum_type_wrapper.EnumTypeWrapper):
    __slots__ = ()
    DEPLOYMENT_MODE_UNSPECIFIED: _ClassVar[DeploymentMode]
    DEPLOYMENT_MODE_PROXY: _ClassVar[DeploymentMode]
    DEPLOYMENT_MODE_BUNDLED: _ClassVar[DeploymentMode]

class Framework(int, metaclass=_enum_type_wrapper.EnumTypeWrapper):
    __slots__ = ()
    FRAMEWORK_UNSPECIFIED: _ClassVar[Framework]
    FRAMEWORK_ELECTRON: _ClassVar[Framework]

class TemplateType(int, metaclass=_enum_type_wrapper.EnumTypeWrapper):
    __slots__ = ()
    TEMPLATE_TYPE_UNSPECIFIED: _ClassVar[TemplateType]
    TEMPLATE_TYPE_BASIC: _ClassVar[TemplateType]
    TEMPLATE_TYPE_ADVANCED: _ClassVar[TemplateType]
    TEMPLATE_TYPE_MULTI_WINDOW: _ClassVar[TemplateType]
    TEMPLATE_TYPE_KIOSK: _ClassVar[TemplateType]

class DistributionProvider(int, metaclass=_enum_type_wrapper.EnumTypeWrapper):
    __slots__ = ()
    DISTRIBUTION_PROVIDER_UNSPECIFIED: _ClassVar[DistributionProvider]
    DISTRIBUTION_PROVIDER_S3: _ClassVar[DistributionProvider]
    DISTRIBUTION_PROVIDER_R2: _ClassVar[DistributionProvider]
    DISTRIBUTION_PROVIDER_S3_COMPATIBLE: _ClassVar[DistributionProvider]
PLATFORM_UNSPECIFIED: Platform
PLATFORM_WIN: Platform
PLATFORM_MAC: Platform
PLATFORM_LINUX: Platform
STAGE_NAME_UNSPECIFIED: StageName
STAGE_NAME_BUNDLE: StageName
STAGE_NAME_PREFLIGHT: StageName
STAGE_NAME_GENERATE: StageName
STAGE_NAME_BUILD: StageName
STAGE_NAME_SMOKE_TEST: StageName
STAGE_NAME_DISTRIBUTION: StageName
STAGE_NAME_DEPLOY: StageName
STAGE_NAME_RESOLVE_DEPLOYMENT: StageName
STAGE_STATUS_UNSPECIFIED: StageStatus
STAGE_STATUS_PENDING: StageStatus
STAGE_STATUS_RUNNING: StageStatus
STAGE_STATUS_COMPLETED: StageStatus
STAGE_STATUS_FAILED: StageStatus
STAGE_STATUS_SKIPPED: StageStatus
STAGE_STATUS_CANCELLED: StageStatus
STAGE_STATUS_IDLE: StageStatus
BUILD_STATUS_UNSPECIFIED: BuildStatus
BUILD_STATUS_BUILDING: BuildStatus
BUILD_STATUS_READY: BuildStatus
BUILD_STATUS_PARTIAL: BuildStatus
BUILD_STATUS_FAILED: BuildStatus
UPLOAD_STATUS_UNSPECIFIED: UploadStatus
UPLOAD_STATUS_PENDING: UploadStatus
UPLOAD_STATUS_UPLOADING: UploadStatus
UPLOAD_STATUS_COMPLETED: UploadStatus
UPLOAD_STATUS_FAILED: UploadStatus
DEPLOYMENT_MODE_UNSPECIFIED: DeploymentMode
DEPLOYMENT_MODE_PROXY: DeploymentMode
DEPLOYMENT_MODE_BUNDLED: DeploymentMode
FRAMEWORK_UNSPECIFIED: Framework
FRAMEWORK_ELECTRON: Framework
TEMPLATE_TYPE_UNSPECIFIED: TemplateType
TEMPLATE_TYPE_BASIC: TemplateType
TEMPLATE_TYPE_ADVANCED: TemplateType
TEMPLATE_TYPE_MULTI_WINDOW: TemplateType
TEMPLATE_TYPE_KIOSK: TemplateType
DISTRIBUTION_PROVIDER_UNSPECIFIED: DistributionProvider
DISTRIBUTION_PROVIDER_S3: DistributionProvider
DISTRIBUTION_PROVIDER_R2: DistributionProvider
DISTRIBUTION_PROVIDER_S3_COMPATIBLE: DistributionProvider

class ValidationError(_message.Message):
    __slots__ = ("code", "message", "field")
    CODE_FIELD_NUMBER: _ClassVar[int]
    MESSAGE_FIELD_NUMBER: _ClassVar[int]
    FIELD_FIELD_NUMBER: _ClassVar[int]
    code: str
    message: str
    field: str
    def __init__(self, code: _Optional[str] = ..., message: _Optional[str] = ..., field: _Optional[str] = ...) -> None: ...

class ValidationWarning(_message.Message):
    __slots__ = ("code", "message", "field")
    CODE_FIELD_NUMBER: _ClassVar[int]
    MESSAGE_FIELD_NUMBER: _ClassVar[int]
    FIELD_FIELD_NUMBER: _ClassVar[int]
    code: str
    message: str
    field: str
    def __init__(self, code: _Optional[str] = ..., message: _Optional[str] = ..., field: _Optional[str] = ...) -> None: ...

class ErrorEnvelope(_message.Message):
    __slots__ = ("code", "category", "recovery", "recovery_hint", "details", "retry_strategy", "auto_fix", "manual_steps", "diagnostic")
    class DetailsEntry(_message.Message):
        __slots__ = ("key", "value")
        KEY_FIELD_NUMBER: _ClassVar[int]
        VALUE_FIELD_NUMBER: _ClassVar[int]
        key: str
        value: str
        def __init__(self, key: _Optional[str] = ..., value: _Optional[str] = ...) -> None: ...
    CODE_FIELD_NUMBER: _ClassVar[int]
    CATEGORY_FIELD_NUMBER: _ClassVar[int]
    RECOVERY_FIELD_NUMBER: _ClassVar[int]
    RECOVERY_HINT_FIELD_NUMBER: _ClassVar[int]
    DETAILS_FIELD_NUMBER: _ClassVar[int]
    RETRY_STRATEGY_FIELD_NUMBER: _ClassVar[int]
    AUTO_FIX_FIELD_NUMBER: _ClassVar[int]
    MANUAL_STEPS_FIELD_NUMBER: _ClassVar[int]
    DIAGNOSTIC_FIELD_NUMBER: _ClassVar[int]
    code: str
    category: str
    recovery: str
    recovery_hint: str
    details: _containers.ScalarMap[str, str]
    retry_strategy: str
    auto_fix: str
    manual_steps: _containers.RepeatedScalarFieldContainer[str]
    diagnostic: str
    def __init__(self, code: _Optional[str] = ..., category: _Optional[str] = ..., recovery: _Optional[str] = ..., recovery_hint: _Optional[str] = ..., details: _Optional[_Mapping[str, str]] = ..., retry_strategy: _Optional[str] = ..., auto_fix: _Optional[str] = ..., manual_steps: _Optional[_Iterable[str]] = ..., diagnostic: _Optional[str] = ...) -> None: ...
