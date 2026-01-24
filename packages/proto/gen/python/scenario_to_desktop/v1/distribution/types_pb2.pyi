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

class DistributionStatusValue(int, metaclass=_enum_type_wrapper.EnumTypeWrapper):
    __slots__ = ()
    DISTRIBUTION_STATUS_VALUE_UNSPECIFIED: _ClassVar[DistributionStatusValue]
    DISTRIBUTION_STATUS_VALUE_PENDING: _ClassVar[DistributionStatusValue]
    DISTRIBUTION_STATUS_VALUE_RUNNING: _ClassVar[DistributionStatusValue]
    DISTRIBUTION_STATUS_VALUE_COMPLETED: _ClassVar[DistributionStatusValue]
    DISTRIBUTION_STATUS_VALUE_PARTIAL: _ClassVar[DistributionStatusValue]
    DISTRIBUTION_STATUS_VALUE_FAILED: _ClassVar[DistributionStatusValue]
    DISTRIBUTION_STATUS_VALUE_CANCELLED: _ClassVar[DistributionStatusValue]

class ACL(int, metaclass=_enum_type_wrapper.EnumTypeWrapper):
    __slots__ = ()
    ACL_UNSPECIFIED: _ClassVar[ACL]
    ACL_PRIVATE: _ClassVar[ACL]
    ACL_PUBLIC_READ: _ClassVar[ACL]
    ACL_AUTHENTICATED_READ: _ClassVar[ACL]
DISTRIBUTION_STATUS_VALUE_UNSPECIFIED: DistributionStatusValue
DISTRIBUTION_STATUS_VALUE_PENDING: DistributionStatusValue
DISTRIBUTION_STATUS_VALUE_RUNNING: DistributionStatusValue
DISTRIBUTION_STATUS_VALUE_COMPLETED: DistributionStatusValue
DISTRIBUTION_STATUS_VALUE_PARTIAL: DistributionStatusValue
DISTRIBUTION_STATUS_VALUE_FAILED: DistributionStatusValue
DISTRIBUTION_STATUS_VALUE_CANCELLED: DistributionStatusValue
ACL_UNSPECIFIED: ACL
ACL_PRIVATE: ACL
ACL_PUBLIC_READ: ACL
ACL_AUTHENTICATED_READ: ACL

class RetryConfig(_message.Message):
    __slots__ = ("max_attempts", "initial_backoff_ms", "max_backoff_ms", "backoff_multiplier")
    MAX_ATTEMPTS_FIELD_NUMBER: _ClassVar[int]
    INITIAL_BACKOFF_MS_FIELD_NUMBER: _ClassVar[int]
    MAX_BACKOFF_MS_FIELD_NUMBER: _ClassVar[int]
    BACKOFF_MULTIPLIER_FIELD_NUMBER: _ClassVar[int]
    max_attempts: int
    initial_backoff_ms: int
    max_backoff_ms: int
    backoff_multiplier: float
    def __init__(self, max_attempts: _Optional[int] = ..., initial_backoff_ms: _Optional[int] = ..., max_backoff_ms: _Optional[int] = ..., backoff_multiplier: _Optional[float] = ...) -> None: ...

class DistributionTarget(_message.Message):
    __slots__ = ("name", "enabled", "provider", "endpoint", "region", "bucket", "path_prefix", "access_key_id_env", "secret_access_key_env", "acl", "cdn_url", "retry", "created_at", "updated_at")
    NAME_FIELD_NUMBER: _ClassVar[int]
    ENABLED_FIELD_NUMBER: _ClassVar[int]
    PROVIDER_FIELD_NUMBER: _ClassVar[int]
    ENDPOINT_FIELD_NUMBER: _ClassVar[int]
    REGION_FIELD_NUMBER: _ClassVar[int]
    BUCKET_FIELD_NUMBER: _ClassVar[int]
    PATH_PREFIX_FIELD_NUMBER: _ClassVar[int]
    ACCESS_KEY_ID_ENV_FIELD_NUMBER: _ClassVar[int]
    SECRET_ACCESS_KEY_ENV_FIELD_NUMBER: _ClassVar[int]
    ACL_FIELD_NUMBER: _ClassVar[int]
    CDN_URL_FIELD_NUMBER: _ClassVar[int]
    RETRY_FIELD_NUMBER: _ClassVar[int]
    CREATED_AT_FIELD_NUMBER: _ClassVar[int]
    UPDATED_AT_FIELD_NUMBER: _ClassVar[int]
    name: str
    enabled: bool
    provider: _shared_pb2.DistributionProvider
    endpoint: str
    region: str
    bucket: str
    path_prefix: str
    access_key_id_env: str
    secret_access_key_env: str
    acl: ACL
    cdn_url: str
    retry: RetryConfig
    created_at: _timestamp_pb2.Timestamp
    updated_at: _timestamp_pb2.Timestamp
    def __init__(self, name: _Optional[str] = ..., enabled: _Optional[bool] = ..., provider: _Optional[_Union[_shared_pb2.DistributionProvider, str]] = ..., endpoint: _Optional[str] = ..., region: _Optional[str] = ..., bucket: _Optional[str] = ..., path_prefix: _Optional[str] = ..., access_key_id_env: _Optional[str] = ..., secret_access_key_env: _Optional[str] = ..., acl: _Optional[_Union[ACL, str]] = ..., cdn_url: _Optional[str] = ..., retry: _Optional[_Union[RetryConfig, _Mapping]] = ..., created_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., updated_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ...) -> None: ...

class DistributionConfig(_message.Message):
    __slots__ = ("schema_version", "targets")
    class TargetsEntry(_message.Message):
        __slots__ = ("key", "value")
        KEY_FIELD_NUMBER: _ClassVar[int]
        VALUE_FIELD_NUMBER: _ClassVar[int]
        key: str
        value: DistributionTarget
        def __init__(self, key: _Optional[str] = ..., value: _Optional[_Union[DistributionTarget, _Mapping]] = ...) -> None: ...
    SCHEMA_VERSION_FIELD_NUMBER: _ClassVar[int]
    TARGETS_FIELD_NUMBER: _ClassVar[int]
    schema_version: str
    targets: _containers.MessageMap[str, DistributionTarget]
    def __init__(self, schema_version: _Optional[str] = ..., targets: _Optional[_Mapping[str, DistributionTarget]] = ...) -> None: ...

class PlatformUpload(_message.Message):
    __slots__ = ("platform", "status", "local_path", "remote_key", "url", "size", "bytes_uploaded", "error", "started_at", "completed_at")
    PLATFORM_FIELD_NUMBER: _ClassVar[int]
    STATUS_FIELD_NUMBER: _ClassVar[int]
    LOCAL_PATH_FIELD_NUMBER: _ClassVar[int]
    REMOTE_KEY_FIELD_NUMBER: _ClassVar[int]
    URL_FIELD_NUMBER: _ClassVar[int]
    SIZE_FIELD_NUMBER: _ClassVar[int]
    BYTES_UPLOADED_FIELD_NUMBER: _ClassVar[int]
    ERROR_FIELD_NUMBER: _ClassVar[int]
    STARTED_AT_FIELD_NUMBER: _ClassVar[int]
    COMPLETED_AT_FIELD_NUMBER: _ClassVar[int]
    platform: _shared_pb2.Platform
    status: _shared_pb2.UploadStatus
    local_path: str
    remote_key: str
    url: str
    size: int
    bytes_uploaded: int
    error: str
    started_at: _timestamp_pb2.Timestamp
    completed_at: _timestamp_pb2.Timestamp
    def __init__(self, platform: _Optional[_Union[_shared_pb2.Platform, str]] = ..., status: _Optional[_Union[_shared_pb2.UploadStatus, str]] = ..., local_path: _Optional[str] = ..., remote_key: _Optional[str] = ..., url: _Optional[str] = ..., size: _Optional[int] = ..., bytes_uploaded: _Optional[int] = ..., error: _Optional[str] = ..., started_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., completed_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ...) -> None: ...

class TargetDistribution(_message.Message):
    __slots__ = ("target_name", "status", "started_at", "completed_at", "uploads", "error")
    class UploadsEntry(_message.Message):
        __slots__ = ("key", "value")
        KEY_FIELD_NUMBER: _ClassVar[int]
        VALUE_FIELD_NUMBER: _ClassVar[int]
        key: str
        value: PlatformUpload
        def __init__(self, key: _Optional[str] = ..., value: _Optional[_Union[PlatformUpload, _Mapping]] = ...) -> None: ...
    TARGET_NAME_FIELD_NUMBER: _ClassVar[int]
    STATUS_FIELD_NUMBER: _ClassVar[int]
    STARTED_AT_FIELD_NUMBER: _ClassVar[int]
    COMPLETED_AT_FIELD_NUMBER: _ClassVar[int]
    UPLOADS_FIELD_NUMBER: _ClassVar[int]
    ERROR_FIELD_NUMBER: _ClassVar[int]
    target_name: str
    status: DistributionStatusValue
    started_at: _timestamp_pb2.Timestamp
    completed_at: _timestamp_pb2.Timestamp
    uploads: _containers.MessageMap[str, PlatformUpload]
    error: str
    def __init__(self, target_name: _Optional[str] = ..., status: _Optional[_Union[DistributionStatusValue, str]] = ..., started_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., completed_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., uploads: _Optional[_Mapping[str, PlatformUpload]] = ..., error: _Optional[str] = ...) -> None: ...

class DistributionStatus(_message.Message):
    __slots__ = ("distribution_id", "scenario_name", "version", "status", "started_at", "completed_at", "targets", "error")
    class TargetsEntry(_message.Message):
        __slots__ = ("key", "value")
        KEY_FIELD_NUMBER: _ClassVar[int]
        VALUE_FIELD_NUMBER: _ClassVar[int]
        key: str
        value: TargetDistribution
        def __init__(self, key: _Optional[str] = ..., value: _Optional[_Union[TargetDistribution, _Mapping]] = ...) -> None: ...
    DISTRIBUTION_ID_FIELD_NUMBER: _ClassVar[int]
    SCENARIO_NAME_FIELD_NUMBER: _ClassVar[int]
    VERSION_FIELD_NUMBER: _ClassVar[int]
    STATUS_FIELD_NUMBER: _ClassVar[int]
    STARTED_AT_FIELD_NUMBER: _ClassVar[int]
    COMPLETED_AT_FIELD_NUMBER: _ClassVar[int]
    TARGETS_FIELD_NUMBER: _ClassVar[int]
    ERROR_FIELD_NUMBER: _ClassVar[int]
    distribution_id: str
    scenario_name: str
    version: str
    status: DistributionStatusValue
    started_at: _timestamp_pb2.Timestamp
    completed_at: _timestamp_pb2.Timestamp
    targets: _containers.MessageMap[str, TargetDistribution]
    error: str
    def __init__(self, distribution_id: _Optional[str] = ..., scenario_name: _Optional[str] = ..., version: _Optional[str] = ..., status: _Optional[_Union[DistributionStatusValue, str]] = ..., started_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., completed_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., targets: _Optional[_Mapping[str, TargetDistribution]] = ..., error: _Optional[str] = ...) -> None: ...

class DistributeRequest(_message.Message):
    __slots__ = ("scenario_name", "version", "artifacts", "target_names", "parallel", "inline_credentials")
    class ArtifactsEntry(_message.Message):
        __slots__ = ("key", "value")
        KEY_FIELD_NUMBER: _ClassVar[int]
        VALUE_FIELD_NUMBER: _ClassVar[int]
        key: str
        value: str
        def __init__(self, key: _Optional[str] = ..., value: _Optional[str] = ...) -> None: ...
    class InlineCredentialsEntry(_message.Message):
        __slots__ = ("key", "value")
        KEY_FIELD_NUMBER: _ClassVar[int]
        VALUE_FIELD_NUMBER: _ClassVar[int]
        key: str
        value: str
        def __init__(self, key: _Optional[str] = ..., value: _Optional[str] = ...) -> None: ...
    SCENARIO_NAME_FIELD_NUMBER: _ClassVar[int]
    VERSION_FIELD_NUMBER: _ClassVar[int]
    ARTIFACTS_FIELD_NUMBER: _ClassVar[int]
    TARGET_NAMES_FIELD_NUMBER: _ClassVar[int]
    PARALLEL_FIELD_NUMBER: _ClassVar[int]
    INLINE_CREDENTIALS_FIELD_NUMBER: _ClassVar[int]
    scenario_name: str
    version: str
    artifacts: _containers.ScalarMap[str, str]
    target_names: _containers.RepeatedScalarFieldContainer[str]
    parallel: bool
    inline_credentials: _containers.ScalarMap[str, str]
    def __init__(self, scenario_name: _Optional[str] = ..., version: _Optional[str] = ..., artifacts: _Optional[_Mapping[str, str]] = ..., target_names: _Optional[_Iterable[str]] = ..., parallel: _Optional[bool] = ..., inline_credentials: _Optional[_Mapping[str, str]] = ...) -> None: ...

class DistributeResponse(_message.Message):
    __slots__ = ("distribution_id", "status", "status_url")
    DISTRIBUTION_ID_FIELD_NUMBER: _ClassVar[int]
    STATUS_FIELD_NUMBER: _ClassVar[int]
    STATUS_URL_FIELD_NUMBER: _ClassVar[int]
    distribution_id: str
    status: DistributionStatusValue
    status_url: str
    def __init__(self, distribution_id: _Optional[str] = ..., status: _Optional[_Union[DistributionStatusValue, str]] = ..., status_url: _Optional[str] = ...) -> None: ...

class TargetCredentialStatus(_message.Message):
    __slots__ = ("target_name", "all_present", "missing_credentials", "required_credentials")
    TARGET_NAME_FIELD_NUMBER: _ClassVar[int]
    ALL_PRESENT_FIELD_NUMBER: _ClassVar[int]
    MISSING_CREDENTIALS_FIELD_NUMBER: _ClassVar[int]
    REQUIRED_CREDENTIALS_FIELD_NUMBER: _ClassVar[int]
    target_name: str
    all_present: bool
    missing_credentials: _containers.RepeatedScalarFieldContainer[str]
    required_credentials: _containers.RepeatedScalarFieldContainer[str]
    def __init__(self, target_name: _Optional[str] = ..., all_present: _Optional[bool] = ..., missing_credentials: _Optional[_Iterable[str]] = ..., required_credentials: _Optional[_Iterable[str]] = ...) -> None: ...

class CheckCredentialsRequest(_message.Message):
    __slots__ = ("target_names",)
    TARGET_NAMES_FIELD_NUMBER: _ClassVar[int]
    target_names: _containers.RepeatedScalarFieldContainer[str]
    def __init__(self, target_names: _Optional[_Iterable[str]] = ...) -> None: ...

class CheckCredentialsResponse(_message.Message):
    __slots__ = ("all_present", "target_status")
    class TargetStatusEntry(_message.Message):
        __slots__ = ("key", "value")
        KEY_FIELD_NUMBER: _ClassVar[int]
        VALUE_FIELD_NUMBER: _ClassVar[int]
        key: str
        value: TargetCredentialStatus
        def __init__(self, key: _Optional[str] = ..., value: _Optional[_Union[TargetCredentialStatus, _Mapping]] = ...) -> None: ...
    ALL_PRESENT_FIELD_NUMBER: _ClassVar[int]
    TARGET_STATUS_FIELD_NUMBER: _ClassVar[int]
    all_present: bool
    target_status: _containers.MessageMap[str, TargetCredentialStatus]
    def __init__(self, all_present: _Optional[bool] = ..., target_status: _Optional[_Mapping[str, TargetCredentialStatus]] = ...) -> None: ...

class TargetValidation(_message.Message):
    __slots__ = ("target_name", "valid", "connected", "permissions_ok", "errors", "warnings")
    TARGET_NAME_FIELD_NUMBER: _ClassVar[int]
    VALID_FIELD_NUMBER: _ClassVar[int]
    CONNECTED_FIELD_NUMBER: _ClassVar[int]
    PERMISSIONS_OK_FIELD_NUMBER: _ClassVar[int]
    ERRORS_FIELD_NUMBER: _ClassVar[int]
    WARNINGS_FIELD_NUMBER: _ClassVar[int]
    target_name: str
    valid: bool
    connected: bool
    permissions_ok: bool
    errors: _containers.RepeatedCompositeFieldContainer[_shared_pb2.ValidationError]
    warnings: _containers.RepeatedCompositeFieldContainer[_shared_pb2.ValidationWarning]
    def __init__(self, target_name: _Optional[str] = ..., valid: _Optional[bool] = ..., connected: _Optional[bool] = ..., permissions_ok: _Optional[bool] = ..., errors: _Optional[_Iterable[_Union[_shared_pb2.ValidationError, _Mapping]]] = ..., warnings: _Optional[_Iterable[_Union[_shared_pb2.ValidationWarning, _Mapping]]] = ...) -> None: ...

class DistributionValidationResult(_message.Message):
    __slots__ = ("valid", "targets", "errors", "warnings", "validated_at")
    class TargetsEntry(_message.Message):
        __slots__ = ("key", "value")
        KEY_FIELD_NUMBER: _ClassVar[int]
        VALUE_FIELD_NUMBER: _ClassVar[int]
        key: str
        value: TargetValidation
        def __init__(self, key: _Optional[str] = ..., value: _Optional[_Union[TargetValidation, _Mapping]] = ...) -> None: ...
    VALID_FIELD_NUMBER: _ClassVar[int]
    TARGETS_FIELD_NUMBER: _ClassVar[int]
    ERRORS_FIELD_NUMBER: _ClassVar[int]
    WARNINGS_FIELD_NUMBER: _ClassVar[int]
    VALIDATED_AT_FIELD_NUMBER: _ClassVar[int]
    valid: bool
    targets: _containers.MessageMap[str, TargetValidation]
    errors: _containers.RepeatedCompositeFieldContainer[_shared_pb2.ValidationError]
    warnings: _containers.RepeatedCompositeFieldContainer[_shared_pb2.ValidationWarning]
    validated_at: _timestamp_pb2.Timestamp
    def __init__(self, valid: _Optional[bool] = ..., targets: _Optional[_Mapping[str, TargetValidation]] = ..., errors: _Optional[_Iterable[_Union[_shared_pb2.ValidationError, _Mapping]]] = ..., warnings: _Optional[_Iterable[_Union[_shared_pb2.ValidationWarning, _Mapping]]] = ..., validated_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ...) -> None: ...
