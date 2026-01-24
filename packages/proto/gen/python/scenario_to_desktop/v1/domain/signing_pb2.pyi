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

class CertificateSource(int, metaclass=_enum_type_wrapper.EnumTypeWrapper):
    __slots__ = ()
    CERTIFICATE_SOURCE_UNSPECIFIED: _ClassVar[CertificateSource]
    CERTIFICATE_SOURCE_FILE: _ClassVar[CertificateSource]
    CERTIFICATE_SOURCE_STORE: _ClassVar[CertificateSource]
    CERTIFICATE_SOURCE_AZURE_KEY_VAULT: _ClassVar[CertificateSource]
    CERTIFICATE_SOURCE_AWS_KMS: _ClassVar[CertificateSource]

class SignAlgorithm(int, metaclass=_enum_type_wrapper.EnumTypeWrapper):
    __slots__ = ()
    SIGN_ALGORITHM_UNSPECIFIED: _ClassVar[SignAlgorithm]
    SIGN_ALGORITHM_SHA256: _ClassVar[SignAlgorithm]
    SIGN_ALGORITHM_SHA384: _ClassVar[SignAlgorithm]
    SIGN_ALGORITHM_SHA512: _ClassVar[SignAlgorithm]
CERTIFICATE_SOURCE_UNSPECIFIED: CertificateSource
CERTIFICATE_SOURCE_FILE: CertificateSource
CERTIFICATE_SOURCE_STORE: CertificateSource
CERTIFICATE_SOURCE_AZURE_KEY_VAULT: CertificateSource
CERTIFICATE_SOURCE_AWS_KMS: CertificateSource
SIGN_ALGORITHM_UNSPECIFIED: SignAlgorithm
SIGN_ALGORITHM_SHA256: SignAlgorithm
SIGN_ALGORITHM_SHA384: SignAlgorithm
SIGN_ALGORITHM_SHA512: SignAlgorithm

class WindowsSigningConfig(_message.Message):
    __slots__ = ("enabled", "certificate_source", "certificate_path", "certificate_thumbprint", "certificate_subject_name", "password_env", "sign_algorithm", "timestamp_server", "description", "description_url", "azure_key_vault_url", "azure_certificate_name", "azure_tenant_id", "azure_client_id", "azure_client_secret_env")
    ENABLED_FIELD_NUMBER: _ClassVar[int]
    CERTIFICATE_SOURCE_FIELD_NUMBER: _ClassVar[int]
    CERTIFICATE_PATH_FIELD_NUMBER: _ClassVar[int]
    CERTIFICATE_THUMBPRINT_FIELD_NUMBER: _ClassVar[int]
    CERTIFICATE_SUBJECT_NAME_FIELD_NUMBER: _ClassVar[int]
    PASSWORD_ENV_FIELD_NUMBER: _ClassVar[int]
    SIGN_ALGORITHM_FIELD_NUMBER: _ClassVar[int]
    TIMESTAMP_SERVER_FIELD_NUMBER: _ClassVar[int]
    DESCRIPTION_FIELD_NUMBER: _ClassVar[int]
    DESCRIPTION_URL_FIELD_NUMBER: _ClassVar[int]
    AZURE_KEY_VAULT_URL_FIELD_NUMBER: _ClassVar[int]
    AZURE_CERTIFICATE_NAME_FIELD_NUMBER: _ClassVar[int]
    AZURE_TENANT_ID_FIELD_NUMBER: _ClassVar[int]
    AZURE_CLIENT_ID_FIELD_NUMBER: _ClassVar[int]
    AZURE_CLIENT_SECRET_ENV_FIELD_NUMBER: _ClassVar[int]
    enabled: bool
    certificate_source: CertificateSource
    certificate_path: str
    certificate_thumbprint: str
    certificate_subject_name: str
    password_env: str
    sign_algorithm: SignAlgorithm
    timestamp_server: str
    description: str
    description_url: str
    azure_key_vault_url: str
    azure_certificate_name: str
    azure_tenant_id: str
    azure_client_id: str
    azure_client_secret_env: str
    def __init__(self, enabled: _Optional[bool] = ..., certificate_source: _Optional[_Union[CertificateSource, str]] = ..., certificate_path: _Optional[str] = ..., certificate_thumbprint: _Optional[str] = ..., certificate_subject_name: _Optional[str] = ..., password_env: _Optional[str] = ..., sign_algorithm: _Optional[_Union[SignAlgorithm, str]] = ..., timestamp_server: _Optional[str] = ..., description: _Optional[str] = ..., description_url: _Optional[str] = ..., azure_key_vault_url: _Optional[str] = ..., azure_certificate_name: _Optional[str] = ..., azure_tenant_id: _Optional[str] = ..., azure_client_id: _Optional[str] = ..., azure_client_secret_env: _Optional[str] = ...) -> None: ...

class MacOSSigningConfig(_message.Message):
    __slots__ = ("enabled", "identity", "team_id", "notarize", "apple_id", "apple_password_env", "entitlements_path", "hardened_runtime", "keychain_profile", "provisioning_profile")
    ENABLED_FIELD_NUMBER: _ClassVar[int]
    IDENTITY_FIELD_NUMBER: _ClassVar[int]
    TEAM_ID_FIELD_NUMBER: _ClassVar[int]
    NOTARIZE_FIELD_NUMBER: _ClassVar[int]
    APPLE_ID_FIELD_NUMBER: _ClassVar[int]
    APPLE_PASSWORD_ENV_FIELD_NUMBER: _ClassVar[int]
    ENTITLEMENTS_PATH_FIELD_NUMBER: _ClassVar[int]
    HARDENED_RUNTIME_FIELD_NUMBER: _ClassVar[int]
    KEYCHAIN_PROFILE_FIELD_NUMBER: _ClassVar[int]
    PROVISIONING_PROFILE_FIELD_NUMBER: _ClassVar[int]
    enabled: bool
    identity: str
    team_id: str
    notarize: bool
    apple_id: str
    apple_password_env: str
    entitlements_path: str
    hardened_runtime: bool
    keychain_profile: str
    provisioning_profile: str
    def __init__(self, enabled: _Optional[bool] = ..., identity: _Optional[str] = ..., team_id: _Optional[str] = ..., notarize: _Optional[bool] = ..., apple_id: _Optional[str] = ..., apple_password_env: _Optional[str] = ..., entitlements_path: _Optional[str] = ..., hardened_runtime: _Optional[bool] = ..., keychain_profile: _Optional[str] = ..., provisioning_profile: _Optional[str] = ...) -> None: ...

class LinuxSigningConfig(_message.Message):
    __slots__ = ("enabled", "gpg_key_id", "passphrase_env", "keyring_path")
    ENABLED_FIELD_NUMBER: _ClassVar[int]
    GPG_KEY_ID_FIELD_NUMBER: _ClassVar[int]
    PASSPHRASE_ENV_FIELD_NUMBER: _ClassVar[int]
    KEYRING_PATH_FIELD_NUMBER: _ClassVar[int]
    enabled: bool
    gpg_key_id: str
    passphrase_env: str
    keyring_path: str
    def __init__(self, enabled: _Optional[bool] = ..., gpg_key_id: _Optional[str] = ..., passphrase_env: _Optional[str] = ..., keyring_path: _Optional[str] = ...) -> None: ...

class SigningConfig(_message.Message):
    __slots__ = ("enabled", "windows", "macos", "linux", "schema_version")
    ENABLED_FIELD_NUMBER: _ClassVar[int]
    WINDOWS_FIELD_NUMBER: _ClassVar[int]
    MACOS_FIELD_NUMBER: _ClassVar[int]
    LINUX_FIELD_NUMBER: _ClassVar[int]
    SCHEMA_VERSION_FIELD_NUMBER: _ClassVar[int]
    enabled: bool
    windows: WindowsSigningConfig
    macos: MacOSSigningConfig
    linux: LinuxSigningConfig
    schema_version: str
    def __init__(self, enabled: _Optional[bool] = ..., windows: _Optional[_Union[WindowsSigningConfig, _Mapping]] = ..., macos: _Optional[_Union[MacOSSigningConfig, _Mapping]] = ..., linux: _Optional[_Union[LinuxSigningConfig, _Mapping]] = ..., schema_version: _Optional[str] = ...) -> None: ...

class CertificateInfo(_message.Message):
    __slots__ = ("subject", "issuer", "thumbprint", "expires_at", "is_valid", "is_expired", "days_until_expiry")
    SUBJECT_FIELD_NUMBER: _ClassVar[int]
    ISSUER_FIELD_NUMBER: _ClassVar[int]
    THUMBPRINT_FIELD_NUMBER: _ClassVar[int]
    EXPIRES_AT_FIELD_NUMBER: _ClassVar[int]
    IS_VALID_FIELD_NUMBER: _ClassVar[int]
    IS_EXPIRED_FIELD_NUMBER: _ClassVar[int]
    DAYS_UNTIL_EXPIRY_FIELD_NUMBER: _ClassVar[int]
    subject: str
    issuer: str
    thumbprint: str
    expires_at: _timestamp_pb2.Timestamp
    is_valid: bool
    is_expired: bool
    days_until_expiry: int
    def __init__(self, subject: _Optional[str] = ..., issuer: _Optional[str] = ..., thumbprint: _Optional[str] = ..., expires_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., is_valid: _Optional[bool] = ..., is_expired: _Optional[bool] = ..., days_until_expiry: _Optional[int] = ...) -> None: ...

class PlatformValidation(_message.Message):
    __slots__ = ("platform", "valid", "enabled", "certificate", "errors", "warnings", "tools_available", "missing_tools")
    PLATFORM_FIELD_NUMBER: _ClassVar[int]
    VALID_FIELD_NUMBER: _ClassVar[int]
    ENABLED_FIELD_NUMBER: _ClassVar[int]
    CERTIFICATE_FIELD_NUMBER: _ClassVar[int]
    ERRORS_FIELD_NUMBER: _ClassVar[int]
    WARNINGS_FIELD_NUMBER: _ClassVar[int]
    TOOLS_AVAILABLE_FIELD_NUMBER: _ClassVar[int]
    MISSING_TOOLS_FIELD_NUMBER: _ClassVar[int]
    platform: _shared_pb2.Platform
    valid: bool
    enabled: bool
    certificate: CertificateInfo
    errors: _containers.RepeatedCompositeFieldContainer[_shared_pb2.ValidationError]
    warnings: _containers.RepeatedCompositeFieldContainer[_shared_pb2.ValidationWarning]
    tools_available: bool
    missing_tools: _containers.RepeatedScalarFieldContainer[str]
    def __init__(self, platform: _Optional[_Union[_shared_pb2.Platform, str]] = ..., valid: _Optional[bool] = ..., enabled: _Optional[bool] = ..., certificate: _Optional[_Union[CertificateInfo, _Mapping]] = ..., errors: _Optional[_Iterable[_Union[_shared_pb2.ValidationError, _Mapping]]] = ..., warnings: _Optional[_Iterable[_Union[_shared_pb2.ValidationWarning, _Mapping]]] = ..., tools_available: _Optional[bool] = ..., missing_tools: _Optional[_Iterable[str]] = ...) -> None: ...

class SigningValidationResult(_message.Message):
    __slots__ = ("valid", "signing_enabled", "platforms", "errors", "warnings", "validated_at")
    class PlatformsEntry(_message.Message):
        __slots__ = ("key", "value")
        KEY_FIELD_NUMBER: _ClassVar[int]
        VALUE_FIELD_NUMBER: _ClassVar[int]
        key: str
        value: PlatformValidation
        def __init__(self, key: _Optional[str] = ..., value: _Optional[_Union[PlatformValidation, _Mapping]] = ...) -> None: ...
    VALID_FIELD_NUMBER: _ClassVar[int]
    SIGNING_ENABLED_FIELD_NUMBER: _ClassVar[int]
    PLATFORMS_FIELD_NUMBER: _ClassVar[int]
    ERRORS_FIELD_NUMBER: _ClassVar[int]
    WARNINGS_FIELD_NUMBER: _ClassVar[int]
    VALIDATED_AT_FIELD_NUMBER: _ClassVar[int]
    valid: bool
    signing_enabled: bool
    platforms: _containers.MessageMap[str, PlatformValidation]
    errors: _containers.RepeatedCompositeFieldContainer[_shared_pb2.ValidationError]
    warnings: _containers.RepeatedCompositeFieldContainer[_shared_pb2.ValidationWarning]
    validated_at: _timestamp_pb2.Timestamp
    def __init__(self, valid: _Optional[bool] = ..., signing_enabled: _Optional[bool] = ..., platforms: _Optional[_Mapping[str, PlatformValidation]] = ..., errors: _Optional[_Iterable[_Union[_shared_pb2.ValidationError, _Mapping]]] = ..., warnings: _Optional[_Iterable[_Union[_shared_pb2.ValidationWarning, _Mapping]]] = ..., validated_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ...) -> None: ...

class PlatformStatus(_message.Message):
    __slots__ = ("platform", "ready", "enabled", "message")
    PLATFORM_FIELD_NUMBER: _ClassVar[int]
    READY_FIELD_NUMBER: _ClassVar[int]
    ENABLED_FIELD_NUMBER: _ClassVar[int]
    MESSAGE_FIELD_NUMBER: _ClassVar[int]
    platform: _shared_pb2.Platform
    ready: bool
    enabled: bool
    message: str
    def __init__(self, platform: _Optional[_Union[_shared_pb2.Platform, str]] = ..., ready: _Optional[bool] = ..., enabled: _Optional[bool] = ..., message: _Optional[str] = ...) -> None: ...

class ReadinessResponse(_message.Message):
    __slots__ = ("ready", "platforms", "message")
    READY_FIELD_NUMBER: _ClassVar[int]
    PLATFORMS_FIELD_NUMBER: _ClassVar[int]
    MESSAGE_FIELD_NUMBER: _ClassVar[int]
    ready: bool
    platforms: _containers.RepeatedCompositeFieldContainer[PlatformStatus]
    message: str
    def __init__(self, ready: _Optional[bool] = ..., platforms: _Optional[_Iterable[_Union[PlatformStatus, _Mapping]]] = ..., message: _Optional[str] = ...) -> None: ...

class SigningConfigResponse(_message.Message):
    __slots__ = ("config", "validation")
    CONFIG_FIELD_NUMBER: _ClassVar[int]
    VALIDATION_FIELD_NUMBER: _ClassVar[int]
    config: SigningConfig
    validation: SigningValidationResult
    def __init__(self, config: _Optional[_Union[SigningConfig, _Mapping]] = ..., validation: _Optional[_Union[SigningValidationResult, _Mapping]] = ...) -> None: ...
