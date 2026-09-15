from google.protobuf import struct_pb2 as _struct_pb2
from google.protobuf.internal import containers as _containers
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Iterable as _Iterable, Mapping as _Mapping
from typing import ClassVar as _ClassVar, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class NativeCLI(_message.Message):
    __slots__ = ("sha256", "goos", "goarch")
    SHA256_FIELD_NUMBER: _ClassVar[int]
    GOOS_FIELD_NUMBER: _ClassVar[int]
    GOARCH_FIELD_NUMBER: _ClassVar[int]
    sha256: str
    goos: str
    goarch: str
    def __init__(self, sha256: _Optional[str] = ..., goos: _Optional[str] = ..., goarch: _Optional[str] = ...) -> None: ...

class Provenance(_message.Message):
    __slots__ = ("builder", "policy")
    BUILDER_FIELD_NUMBER: _ClassVar[int]
    POLICY_FIELD_NUMBER: _ClassVar[int]
    builder: str
    policy: str
    def __init__(self, builder: _Optional[str] = ..., policy: _Optional[str] = ...) -> None: ...

class Limits(_message.Message):
    __slots__ = ("max_entries", "max_expanded_bytes", "max_entry_bytes")
    MAX_ENTRIES_FIELD_NUMBER: _ClassVar[int]
    MAX_EXPANDED_BYTES_FIELD_NUMBER: _ClassVar[int]
    MAX_ENTRY_BYTES_FIELD_NUMBER: _ClassVar[int]
    max_entries: int
    max_expanded_bytes: int
    max_entry_bytes: int
    def __init__(self, max_entries: _Optional[int] = ..., max_expanded_bytes: _Optional[int] = ..., max_entry_bytes: _Optional[int] = ...) -> None: ...

class ReleaseManifest(_message.Message):
    __slots__ = ("schema_version", "release_digest", "bundle_sha256", "native_cli", "closure_digest", "configuration_digest", "provenance", "limits")
    SCHEMA_VERSION_FIELD_NUMBER: _ClassVar[int]
    RELEASE_DIGEST_FIELD_NUMBER: _ClassVar[int]
    BUNDLE_SHA256_FIELD_NUMBER: _ClassVar[int]
    NATIVE_CLI_FIELD_NUMBER: _ClassVar[int]
    CLOSURE_DIGEST_FIELD_NUMBER: _ClassVar[int]
    CONFIGURATION_DIGEST_FIELD_NUMBER: _ClassVar[int]
    PROVENANCE_FIELD_NUMBER: _ClassVar[int]
    LIMITS_FIELD_NUMBER: _ClassVar[int]
    schema_version: int
    release_digest: str
    bundle_sha256: str
    native_cli: NativeCLI
    closure_digest: str
    configuration_digest: str
    provenance: Provenance
    limits: Limits
    def __init__(self, schema_version: _Optional[int] = ..., release_digest: _Optional[str] = ..., bundle_sha256: _Optional[str] = ..., native_cli: _Optional[_Union[NativeCLI, _Mapping]] = ..., closure_digest: _Optional[str] = ..., configuration_digest: _Optional[str] = ..., provenance: _Optional[_Union[Provenance, _Mapping]] = ..., limits: _Optional[_Union[Limits, _Mapping]] = ...) -> None: ...

class ReleaseInputs(_message.Message):
    __slots__ = ("schema_version", "release_digest", "built_at", "builder", "source", "bundle", "dependencies", "native_cli", "resource_artifacts", "toolchain", "configuration", "credentials", "provenance", "reproducibility", "limitations")
    SCHEMA_VERSION_FIELD_NUMBER: _ClassVar[int]
    RELEASE_DIGEST_FIELD_NUMBER: _ClassVar[int]
    BUILT_AT_FIELD_NUMBER: _ClassVar[int]
    BUILDER_FIELD_NUMBER: _ClassVar[int]
    SOURCE_FIELD_NUMBER: _ClassVar[int]
    BUNDLE_FIELD_NUMBER: _ClassVar[int]
    DEPENDENCIES_FIELD_NUMBER: _ClassVar[int]
    NATIVE_CLI_FIELD_NUMBER: _ClassVar[int]
    RESOURCE_ARTIFACTS_FIELD_NUMBER: _ClassVar[int]
    TOOLCHAIN_FIELD_NUMBER: _ClassVar[int]
    CONFIGURATION_FIELD_NUMBER: _ClassVar[int]
    CREDENTIALS_FIELD_NUMBER: _ClassVar[int]
    PROVENANCE_FIELD_NUMBER: _ClassVar[int]
    REPRODUCIBILITY_FIELD_NUMBER: _ClassVar[int]
    LIMITATIONS_FIELD_NUMBER: _ClassVar[int]
    schema_version: int
    release_digest: str
    built_at: str
    builder: ReleaseBuilder
    source: ReleaseSource
    bundle: ReleaseBundle
    dependencies: ReleaseDependencies
    native_cli: ReleaseNativeCLIBuild
    resource_artifacts: _containers.RepeatedCompositeFieldContainer[ReleaseResourceArtifact]
    toolchain: ReleaseToolchain
    configuration: ReleaseConfiguration
    credentials: _containers.RepeatedCompositeFieldContainer[ReleaseCredentialRef]
    provenance: ReleaseProvenance
    reproducibility: ReleaseReproducibility
    limitations: _containers.RepeatedScalarFieldContainer[str]
    def __init__(self, schema_version: _Optional[int] = ..., release_digest: _Optional[str] = ..., built_at: _Optional[str] = ..., builder: _Optional[_Union[ReleaseBuilder, _Mapping]] = ..., source: _Optional[_Union[ReleaseSource, _Mapping]] = ..., bundle: _Optional[_Union[ReleaseBundle, _Mapping]] = ..., dependencies: _Optional[_Union[ReleaseDependencies, _Mapping]] = ..., native_cli: _Optional[_Union[ReleaseNativeCLIBuild, _Mapping]] = ..., resource_artifacts: _Optional[_Iterable[_Union[ReleaseResourceArtifact, _Mapping]]] = ..., toolchain: _Optional[_Union[ReleaseToolchain, _Mapping]] = ..., configuration: _Optional[_Union[ReleaseConfiguration, _Mapping]] = ..., credentials: _Optional[_Iterable[_Union[ReleaseCredentialRef, _Mapping]]] = ..., provenance: _Optional[_Union[ReleaseProvenance, _Mapping]] = ..., reproducibility: _Optional[_Union[ReleaseReproducibility, _Mapping]] = ..., limitations: _Optional[_Iterable[str]] = ...) -> None: ...

class ReleaseBuilder(_message.Message):
    __slots__ = ("identity", "node", "user")
    IDENTITY_FIELD_NUMBER: _ClassVar[int]
    NODE_FIELD_NUMBER: _ClassVar[int]
    USER_FIELD_NUMBER: _ClassVar[int]
    identity: str
    node: str
    user: str
    def __init__(self, identity: _Optional[str] = ..., node: _Optional[str] = ..., user: _Optional[str] = ...) -> None: ...

class ReleaseSource(_message.Message):
    __slots__ = ("commit", "dirty", "dirty_paths", "snapshot", "content_manifest_sha256", "file_count", "include_roots", "excludes")
    COMMIT_FIELD_NUMBER: _ClassVar[int]
    DIRTY_FIELD_NUMBER: _ClassVar[int]
    DIRTY_PATHS_FIELD_NUMBER: _ClassVar[int]
    SNAPSHOT_FIELD_NUMBER: _ClassVar[int]
    CONTENT_MANIFEST_SHA256_FIELD_NUMBER: _ClassVar[int]
    FILE_COUNT_FIELD_NUMBER: _ClassVar[int]
    INCLUDE_ROOTS_FIELD_NUMBER: _ClassVar[int]
    EXCLUDES_FIELD_NUMBER: _ClassVar[int]
    commit: str
    dirty: bool
    dirty_paths: int
    snapshot: str
    content_manifest_sha256: str
    file_count: int
    include_roots: _containers.RepeatedScalarFieldContainer[str]
    excludes: _containers.RepeatedScalarFieldContainer[str]
    def __init__(self, commit: _Optional[str] = ..., dirty: _Optional[bool] = ..., dirty_paths: _Optional[int] = ..., snapshot: _Optional[str] = ..., content_manifest_sha256: _Optional[str] = ..., file_count: _Optional[int] = ..., include_roots: _Optional[_Iterable[str]] = ..., excludes: _Optional[_Iterable[str]] = ...) -> None: ...

class ReleaseBundle(_message.Message):
    __slots__ = ("file_name", "sha256", "size_bytes")
    FILE_NAME_FIELD_NUMBER: _ClassVar[int]
    SHA256_FIELD_NUMBER: _ClassVar[int]
    SIZE_BYTES_FIELD_NUMBER: _ClassVar[int]
    file_name: str
    sha256: str
    size_bytes: int
    def __init__(self, file_name: _Optional[str] = ..., sha256: _Optional[str] = ..., size_bytes: _Optional[int] = ...) -> None: ...

class ReleaseDependencies(_message.Message):
    __slots__ = ("closure_digest", "analyzer_tool", "analyzer_fingerprint", "analyzer_generated_at", "scenarios", "resources")
    CLOSURE_DIGEST_FIELD_NUMBER: _ClassVar[int]
    ANALYZER_TOOL_FIELD_NUMBER: _ClassVar[int]
    ANALYZER_FINGERPRINT_FIELD_NUMBER: _ClassVar[int]
    ANALYZER_GENERATED_AT_FIELD_NUMBER: _ClassVar[int]
    SCENARIOS_FIELD_NUMBER: _ClassVar[int]
    RESOURCES_FIELD_NUMBER: _ClassVar[int]
    closure_digest: str
    analyzer_tool: str
    analyzer_fingerprint: str
    analyzer_generated_at: str
    scenarios: _containers.RepeatedScalarFieldContainer[str]
    resources: _containers.RepeatedScalarFieldContainer[str]
    def __init__(self, closure_digest: _Optional[str] = ..., analyzer_tool: _Optional[str] = ..., analyzer_fingerprint: _Optional[str] = ..., analyzer_generated_at: _Optional[str] = ..., scenarios: _Optional[_Iterable[str]] = ..., resources: _Optional[_Iterable[str]] = ...) -> None: ...

class ReleaseNativeCLIBuild(_message.Message):
    __slots__ = ("file_name", "sha256", "goos", "goarch", "size_bytes", "package", "module_dir", "go_version", "args", "env")
    FILE_NAME_FIELD_NUMBER: _ClassVar[int]
    SHA256_FIELD_NUMBER: _ClassVar[int]
    GOOS_FIELD_NUMBER: _ClassVar[int]
    GOARCH_FIELD_NUMBER: _ClassVar[int]
    SIZE_BYTES_FIELD_NUMBER: _ClassVar[int]
    PACKAGE_FIELD_NUMBER: _ClassVar[int]
    MODULE_DIR_FIELD_NUMBER: _ClassVar[int]
    GO_VERSION_FIELD_NUMBER: _ClassVar[int]
    ARGS_FIELD_NUMBER: _ClassVar[int]
    ENV_FIELD_NUMBER: _ClassVar[int]
    file_name: str
    sha256: str
    goos: str
    goarch: str
    size_bytes: int
    package: str
    module_dir: str
    go_version: str
    args: _containers.RepeatedScalarFieldContainer[str]
    env: _containers.RepeatedScalarFieldContainer[str]
    def __init__(self, file_name: _Optional[str] = ..., sha256: _Optional[str] = ..., goos: _Optional[str] = ..., goarch: _Optional[str] = ..., size_bytes: _Optional[int] = ..., package: _Optional[str] = ..., module_dir: _Optional[str] = ..., go_version: _Optional[str] = ..., args: _Optional[_Iterable[str]] = ..., env: _Optional[_Iterable[str]] = ...) -> None: ...

class ReleaseResourceArtifact(_message.Message):
    __slots__ = ("component", "platform", "name", "digest", "mode", "eligibility", "license_refs")
    COMPONENT_FIELD_NUMBER: _ClassVar[int]
    PLATFORM_FIELD_NUMBER: _ClassVar[int]
    NAME_FIELD_NUMBER: _ClassVar[int]
    DIGEST_FIELD_NUMBER: _ClassVar[int]
    MODE_FIELD_NUMBER: _ClassVar[int]
    ELIGIBILITY_FIELD_NUMBER: _ClassVar[int]
    LICENSE_REFS_FIELD_NUMBER: _ClassVar[int]
    component: str
    platform: str
    name: str
    digest: str
    mode: str
    eligibility: str
    license_refs: _containers.RepeatedScalarFieldContainer[str]
    def __init__(self, component: _Optional[str] = ..., platform: _Optional[str] = ..., name: _Optional[str] = ..., digest: _Optional[str] = ..., mode: _Optional[str] = ..., eligibility: _Optional[str] = ..., license_refs: _Optional[_Iterable[str]] = ...) -> None: ...

class ReleaseToolchain(_message.Message):
    __slots__ = ("go_version", "host_goos", "host_goarch", "go_flags")
    GO_VERSION_FIELD_NUMBER: _ClassVar[int]
    HOST_GOOS_FIELD_NUMBER: _ClassVar[int]
    HOST_GOARCH_FIELD_NUMBER: _ClassVar[int]
    GO_FLAGS_FIELD_NUMBER: _ClassVar[int]
    go_version: str
    host_goos: str
    host_goarch: str
    go_flags: _containers.RepeatedScalarFieldContainer[str]
    def __init__(self, go_version: _Optional[str] = ..., host_goos: _Optional[str] = ..., host_goarch: _Optional[str] = ..., go_flags: _Optional[_Iterable[str]] = ...) -> None: ...

class ReleaseConfiguration(_message.Message):
    __slots__ = ("digest", "schema", "rule")
    DIGEST_FIELD_NUMBER: _ClassVar[int]
    SCHEMA_FIELD_NUMBER: _ClassVar[int]
    RULE_FIELD_NUMBER: _ClassVar[int]
    digest: str
    schema: str
    rule: str
    def __init__(self, digest: _Optional[str] = ..., schema: _Optional[str] = ..., rule: _Optional[str] = ...) -> None: ...

class ReleaseCredentialRef(_message.Message):
    __slots__ = ("id", "required", "logical_id", "field", "target_type", "target_name")
    ID_FIELD_NUMBER: _ClassVar[int]
    CLASS_FIELD_NUMBER: _ClassVar[int]
    REQUIRED_FIELD_NUMBER: _ClassVar[int]
    LOGICAL_ID_FIELD_NUMBER: _ClassVar[int]
    FIELD_FIELD_NUMBER: _ClassVar[int]
    TARGET_TYPE_FIELD_NUMBER: _ClassVar[int]
    TARGET_NAME_FIELD_NUMBER: _ClassVar[int]
    id: str
    required: bool
    logical_id: str
    field: str
    target_type: str
    target_name: str
    def __init__(self, id: _Optional[str] = ..., required: _Optional[bool] = ..., logical_id: _Optional[str] = ..., field: _Optional[str] = ..., target_type: _Optional[str] = ..., target_name: _Optional[str] = ..., **kwargs) -> None: ...

class ReleaseProvenance(_message.Message):
    __slots__ = ("policy", "trust_mode", "signer_key_id", "signature_file")
    POLICY_FIELD_NUMBER: _ClassVar[int]
    TRUST_MODE_FIELD_NUMBER: _ClassVar[int]
    SIGNER_KEY_ID_FIELD_NUMBER: _ClassVar[int]
    SIGNATURE_FILE_FIELD_NUMBER: _ClassVar[int]
    policy: str
    trust_mode: str
    signer_key_id: str
    signature_file: str
    def __init__(self, policy: _Optional[str] = ..., trust_mode: _Optional[str] = ..., signer_key_id: _Optional[str] = ..., signature_file: _Optional[str] = ...) -> None: ...

class ReleaseReproducibility(_message.Message):
    __slots__ = ("bundle", "native_cli", "reason")
    BUNDLE_FIELD_NUMBER: _ClassVar[int]
    NATIVE_CLI_FIELD_NUMBER: _ClassVar[int]
    REASON_FIELD_NUMBER: _ClassVar[int]
    bundle: str
    native_cli: str
    reason: str
    def __init__(self, bundle: _Optional[str] = ..., native_cli: _Optional[str] = ..., reason: _Optional[str] = ...) -> None: ...

class Release(_message.Message):
    __slots__ = ("release_digest", "dir", "complete", "manifest", "inputs")
    RELEASE_DIGEST_FIELD_NUMBER: _ClassVar[int]
    DIR_FIELD_NUMBER: _ClassVar[int]
    COMPLETE_FIELD_NUMBER: _ClassVar[int]
    MANIFEST_FIELD_NUMBER: _ClassVar[int]
    INPUTS_FIELD_NUMBER: _ClassVar[int]
    release_digest: str
    dir: str
    complete: bool
    manifest: ReleaseManifest
    inputs: ReleaseInputs
    def __init__(self, release_digest: _Optional[str] = ..., dir: _Optional[str] = ..., complete: _Optional[bool] = ..., manifest: _Optional[_Union[ReleaseManifest, _Mapping]] = ..., inputs: _Optional[_Union[ReleaseInputs, _Mapping]] = ...) -> None: ...

class BuildReleaseRequest(_message.Message):
    __slots__ = ("manifest", "closure_digest", "goos", "goarch", "trust_mode", "verify_reproducible")
    MANIFEST_FIELD_NUMBER: _ClassVar[int]
    CLOSURE_DIGEST_FIELD_NUMBER: _ClassVar[int]
    GOOS_FIELD_NUMBER: _ClassVar[int]
    GOARCH_FIELD_NUMBER: _ClassVar[int]
    TRUST_MODE_FIELD_NUMBER: _ClassVar[int]
    VERIFY_REPRODUCIBLE_FIELD_NUMBER: _ClassVar[int]
    manifest: _struct_pb2.Struct
    closure_digest: str
    goos: str
    goarch: str
    trust_mode: str
    verify_reproducible: bool
    def __init__(self, manifest: _Optional[_Union[_struct_pb2.Struct, _Mapping]] = ..., closure_digest: _Optional[str] = ..., goos: _Optional[str] = ..., goarch: _Optional[str] = ..., trust_mode: _Optional[str] = ..., verify_reproducible: _Optional[bool] = ...) -> None: ...

class BuildReleaseResponse(_message.Message):
    __slots__ = ("schema_version", "release")
    SCHEMA_VERSION_FIELD_NUMBER: _ClassVar[int]
    RELEASE_FIELD_NUMBER: _ClassVar[int]
    schema_version: str
    release: Release
    def __init__(self, schema_version: _Optional[str] = ..., release: _Optional[_Union[Release, _Mapping]] = ...) -> None: ...

class GetReleaseRequest(_message.Message):
    __slots__ = ("release_digest",)
    RELEASE_DIGEST_FIELD_NUMBER: _ClassVar[int]
    release_digest: str
    def __init__(self, release_digest: _Optional[str] = ...) -> None: ...

class GetReleaseResponse(_message.Message):
    __slots__ = ("schema_version", "release")
    SCHEMA_VERSION_FIELD_NUMBER: _ClassVar[int]
    RELEASE_FIELD_NUMBER: _ClassVar[int]
    schema_version: str
    release: Release
    def __init__(self, schema_version: _Optional[str] = ..., release: _Optional[_Union[Release, _Mapping]] = ...) -> None: ...

class VerifyReleaseRequest(_message.Message):
    __slots__ = ("release_digest", "trust_mode", "goos", "goarch")
    RELEASE_DIGEST_FIELD_NUMBER: _ClassVar[int]
    TRUST_MODE_FIELD_NUMBER: _ClassVar[int]
    GOOS_FIELD_NUMBER: _ClassVar[int]
    GOARCH_FIELD_NUMBER: _ClassVar[int]
    release_digest: str
    trust_mode: str
    goos: str
    goarch: str
    def __init__(self, release_digest: _Optional[str] = ..., trust_mode: _Optional[str] = ..., goos: _Optional[str] = ..., goarch: _Optional[str] = ...) -> None: ...

class VerifyCheck(_message.Message):
    __slots__ = ("id", "status", "code")
    ID_FIELD_NUMBER: _ClassVar[int]
    STATUS_FIELD_NUMBER: _ClassVar[int]
    CODE_FIELD_NUMBER: _ClassVar[int]
    id: str
    status: str
    code: str
    def __init__(self, id: _Optional[str] = ..., status: _Optional[str] = ..., code: _Optional[str] = ...) -> None: ...

class VerifyReleaseResponse(_message.Message):
    __slots__ = ("schema_version", "release_digest", "verified", "checks", "policy", "trust_mode", "signer_key_id")
    SCHEMA_VERSION_FIELD_NUMBER: _ClassVar[int]
    RELEASE_DIGEST_FIELD_NUMBER: _ClassVar[int]
    VERIFIED_FIELD_NUMBER: _ClassVar[int]
    CHECKS_FIELD_NUMBER: _ClassVar[int]
    POLICY_FIELD_NUMBER: _ClassVar[int]
    TRUST_MODE_FIELD_NUMBER: _ClassVar[int]
    SIGNER_KEY_ID_FIELD_NUMBER: _ClassVar[int]
    schema_version: str
    release_digest: str
    verified: bool
    checks: _containers.RepeatedCompositeFieldContainer[VerifyCheck]
    policy: str
    trust_mode: str
    signer_key_id: str
    def __init__(self, schema_version: _Optional[str] = ..., release_digest: _Optional[str] = ..., verified: _Optional[bool] = ..., checks: _Optional[_Iterable[_Union[VerifyCheck, _Mapping]]] = ..., policy: _Optional[str] = ..., trust_mode: _Optional[str] = ..., signer_key_id: _Optional[str] = ...) -> None: ...
