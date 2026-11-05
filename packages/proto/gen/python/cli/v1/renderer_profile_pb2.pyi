from google.protobuf.internal import containers as _containers
from google.protobuf.internal import enum_type_wrapper as _enum_type_wrapper
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Iterable as _Iterable, Mapping as _Mapping
from typing import ClassVar as _ClassVar, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class RendererToolProfileState(int, metaclass=_enum_type_wrapper.EnumTypeWrapper):
    __slots__ = ()
    RENDERER_TOOL_PROFILE_STATE_UNSPECIFIED: _ClassVar[RendererToolProfileState]
    RENDERER_TOOL_PROFILE_STATE_UNSUPPORTED_PLATFORM: _ClassVar[RendererToolProfileState]
    RENDERER_TOOL_PROFILE_STATE_MISSING: _ClassVar[RendererToolProfileState]
    RENDERER_TOOL_PROFILE_STATE_UNMANAGED: _ClassVar[RendererToolProfileState]
    RENDERER_TOOL_PROFILE_STATE_PARTIAL: _ClassVar[RendererToolProfileState]
    RENDERER_TOOL_PROFILE_STATE_TAMPERED: _ClassVar[RendererToolProfileState]
    RENDERER_TOOL_PROFILE_STATE_PROBE_FAILED: _ClassVar[RendererToolProfileState]
    RENDERER_TOOL_PROFILE_STATE_ARTIFACT_PRESENT: _ClassVar[RendererToolProfileState]
RENDERER_TOOL_PROFILE_STATE_UNSPECIFIED: RendererToolProfileState
RENDERER_TOOL_PROFILE_STATE_UNSUPPORTED_PLATFORM: RendererToolProfileState
RENDERER_TOOL_PROFILE_STATE_MISSING: RendererToolProfileState
RENDERER_TOOL_PROFILE_STATE_UNMANAGED: RendererToolProfileState
RENDERER_TOOL_PROFILE_STATE_PARTIAL: RendererToolProfileState
RENDERER_TOOL_PROFILE_STATE_TAMPERED: RendererToolProfileState
RENDERER_TOOL_PROFILE_STATE_PROBE_FAILED: RendererToolProfileState
RENDERER_TOOL_PROFILE_STATE_ARTIFACT_PRESENT: RendererToolProfileState

class RendererToolProfile(_message.Message):
    __slots__ = ("schema_version", "profile_id", "state", "artifact_identity", "environment_identity", "os", "architecture", "archive_sha256", "ffmpeg", "ffprobe", "license_id", "license_sha256", "runtime_prerequisites", "qualification_scope", "detail", "observed_at", "archive_size_bytes", "license_details")
    SCHEMA_VERSION_FIELD_NUMBER: _ClassVar[int]
    PROFILE_ID_FIELD_NUMBER: _ClassVar[int]
    STATE_FIELD_NUMBER: _ClassVar[int]
    ARTIFACT_IDENTITY_FIELD_NUMBER: _ClassVar[int]
    ENVIRONMENT_IDENTITY_FIELD_NUMBER: _ClassVar[int]
    OS_FIELD_NUMBER: _ClassVar[int]
    ARCHITECTURE_FIELD_NUMBER: _ClassVar[int]
    ARCHIVE_SHA256_FIELD_NUMBER: _ClassVar[int]
    FFMPEG_FIELD_NUMBER: _ClassVar[int]
    FFPROBE_FIELD_NUMBER: _ClassVar[int]
    LICENSE_ID_FIELD_NUMBER: _ClassVar[int]
    LICENSE_SHA256_FIELD_NUMBER: _ClassVar[int]
    RUNTIME_PREREQUISITES_FIELD_NUMBER: _ClassVar[int]
    QUALIFICATION_SCOPE_FIELD_NUMBER: _ClassVar[int]
    DETAIL_FIELD_NUMBER: _ClassVar[int]
    OBSERVED_AT_FIELD_NUMBER: _ClassVar[int]
    ARCHIVE_SIZE_BYTES_FIELD_NUMBER: _ClassVar[int]
    LICENSE_DETAILS_FIELD_NUMBER: _ClassVar[int]
    schema_version: int
    profile_id: str
    state: RendererToolProfileState
    artifact_identity: str
    environment_identity: str
    os: str
    architecture: str
    archive_sha256: str
    ffmpeg: RendererToolBinary
    ffprobe: RendererToolBinary
    license_id: str
    license_sha256: str
    runtime_prerequisites: _containers.RepeatedScalarFieldContainer[str]
    qualification_scope: str
    detail: str
    observed_at: str
    archive_size_bytes: int
    license_details: str
    def __init__(self, schema_version: _Optional[int] = ..., profile_id: _Optional[str] = ..., state: _Optional[_Union[RendererToolProfileState, str]] = ..., artifact_identity: _Optional[str] = ..., environment_identity: _Optional[str] = ..., os: _Optional[str] = ..., architecture: _Optional[str] = ..., archive_sha256: _Optional[str] = ..., ffmpeg: _Optional[_Union[RendererToolBinary, _Mapping]] = ..., ffprobe: _Optional[_Union[RendererToolBinary, _Mapping]] = ..., license_id: _Optional[str] = ..., license_sha256: _Optional[str] = ..., runtime_prerequisites: _Optional[_Iterable[str]] = ..., qualification_scope: _Optional[str] = ..., detail: _Optional[str] = ..., observed_at: _Optional[str] = ..., archive_size_bytes: _Optional[int] = ..., license_details: _Optional[str] = ...) -> None: ...

class RendererToolBinary(_message.Message):
    __slots__ = ("path", "sha256", "size_bytes", "version", "build_config_sha256", "build_config", "archive_member_path")
    PATH_FIELD_NUMBER: _ClassVar[int]
    SHA256_FIELD_NUMBER: _ClassVar[int]
    SIZE_BYTES_FIELD_NUMBER: _ClassVar[int]
    VERSION_FIELD_NUMBER: _ClassVar[int]
    BUILD_CONFIG_SHA256_FIELD_NUMBER: _ClassVar[int]
    BUILD_CONFIG_FIELD_NUMBER: _ClassVar[int]
    ARCHIVE_MEMBER_PATH_FIELD_NUMBER: _ClassVar[int]
    path: str
    sha256: str
    size_bytes: int
    version: str
    build_config_sha256: str
    build_config: str
    archive_member_path: str
    def __init__(self, path: _Optional[str] = ..., sha256: _Optional[str] = ..., size_bytes: _Optional[int] = ..., version: _Optional[str] = ..., build_config_sha256: _Optional[str] = ..., build_config: _Optional[str] = ..., archive_member_path: _Optional[str] = ...) -> None: ...

class BlenderToolProfile(_message.Message):
    __slots__ = ("schema_version", "profile_id", "state", "artifact_identity", "environment_identity", "os", "architecture", "archive_sha256", "archive_size_bytes", "expected_tree_sha256", "blender", "license_id", "license_sha256", "license_details", "runtime_prerequisites", "qualification_scope", "detail", "observed_at")
    SCHEMA_VERSION_FIELD_NUMBER: _ClassVar[int]
    PROFILE_ID_FIELD_NUMBER: _ClassVar[int]
    STATE_FIELD_NUMBER: _ClassVar[int]
    ARTIFACT_IDENTITY_FIELD_NUMBER: _ClassVar[int]
    ENVIRONMENT_IDENTITY_FIELD_NUMBER: _ClassVar[int]
    OS_FIELD_NUMBER: _ClassVar[int]
    ARCHITECTURE_FIELD_NUMBER: _ClassVar[int]
    ARCHIVE_SHA256_FIELD_NUMBER: _ClassVar[int]
    ARCHIVE_SIZE_BYTES_FIELD_NUMBER: _ClassVar[int]
    EXPECTED_TREE_SHA256_FIELD_NUMBER: _ClassVar[int]
    BLENDER_FIELD_NUMBER: _ClassVar[int]
    LICENSE_ID_FIELD_NUMBER: _ClassVar[int]
    LICENSE_SHA256_FIELD_NUMBER: _ClassVar[int]
    LICENSE_DETAILS_FIELD_NUMBER: _ClassVar[int]
    RUNTIME_PREREQUISITES_FIELD_NUMBER: _ClassVar[int]
    QUALIFICATION_SCOPE_FIELD_NUMBER: _ClassVar[int]
    DETAIL_FIELD_NUMBER: _ClassVar[int]
    OBSERVED_AT_FIELD_NUMBER: _ClassVar[int]
    schema_version: int
    profile_id: str
    state: RendererToolProfileState
    artifact_identity: str
    environment_identity: str
    os: str
    architecture: str
    archive_sha256: str
    archive_size_bytes: int
    expected_tree_sha256: str
    blender: RendererToolBinary
    license_id: str
    license_sha256: str
    license_details: str
    runtime_prerequisites: _containers.RepeatedScalarFieldContainer[str]
    qualification_scope: str
    detail: str
    observed_at: str
    def __init__(self, schema_version: _Optional[int] = ..., profile_id: _Optional[str] = ..., state: _Optional[_Union[RendererToolProfileState, str]] = ..., artifact_identity: _Optional[str] = ..., environment_identity: _Optional[str] = ..., os: _Optional[str] = ..., architecture: _Optional[str] = ..., archive_sha256: _Optional[str] = ..., archive_size_bytes: _Optional[int] = ..., expected_tree_sha256: _Optional[str] = ..., blender: _Optional[_Union[RendererToolBinary, _Mapping]] = ..., license_id: _Optional[str] = ..., license_sha256: _Optional[str] = ..., license_details: _Optional[str] = ..., runtime_prerequisites: _Optional[_Iterable[str]] = ..., qualification_scope: _Optional[str] = ..., detail: _Optional[str] = ..., observed_at: _Optional[str] = ...) -> None: ...
