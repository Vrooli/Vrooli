import datetime

from agent_manager.v1.domain import types_pb2 as _types_pb2
from buf.validate import validate_pb2 as _validate_pb2
from google.protobuf import duration_pb2 as _duration_pb2
from google.protobuf import timestamp_pb2 as _timestamp_pb2
from google.protobuf.internal import containers as _containers
from google.protobuf.internal import enum_type_wrapper as _enum_type_wrapper
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Iterable as _Iterable, Mapping as _Mapping
from typing import ClassVar as _ClassVar, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class ResultSpecKind(int, metaclass=_enum_type_wrapper.EnumTypeWrapper):
    __slots__ = ()
    RESULT_SPEC_KIND_UNSPECIFIED: _ClassVar[ResultSpecKind]
    RESULT_SPEC_KIND_NONE: _ClassVar[ResultSpecKind]
    RESULT_SPEC_KIND_JSON_SCHEMA: _ClassVar[ResultSpecKind]
    RESULT_SPEC_KIND_CLASSIFICATION: _ClassVar[ResultSpecKind]

class StructuredExtractionMode(int, metaclass=_enum_type_wrapper.EnumTypeWrapper):
    __slots__ = ()
    STRUCTURED_EXTRACTION_MODE_UNSPECIFIED: _ClassVar[StructuredExtractionMode]
    STRUCTURED_EXTRACTION_MODE_DETERMINISTIC_ONLY: _ClassVar[StructuredExtractionMode]
    STRUCTURED_EXTRACTION_MODE_CONSTRAINED_FALLBACK: _ClassVar[StructuredExtractionMode]
RESULT_SPEC_KIND_UNSPECIFIED: ResultSpecKind
RESULT_SPEC_KIND_NONE: ResultSpecKind
RESULT_SPEC_KIND_JSON_SCHEMA: ResultSpecKind
RESULT_SPEC_KIND_CLASSIFICATION: ResultSpecKind
STRUCTURED_EXTRACTION_MODE_UNSPECIFIED: StructuredExtractionMode
STRUCTURED_EXTRACTION_MODE_DETERMINISTIC_ONLY: StructuredExtractionMode
STRUCTURED_EXTRACTION_MODE_CONSTRAINED_FALLBACK: StructuredExtractionMode

class AgentProfile(_message.Message):
    __slots__ = ("id", "name", "profile_key", "description", "role_ref", "max_turns", "timeout", "allowed_tools", "denied_tools", "tool_restriction_policy", "skip_permission_prompt", "features", "extra_flags", "network_access", "owner_scenario", "source_path", "source_hash", "last_applied_hash", "source_updated_at", "local_override", "sandbox_config", "allowed_paths", "denied_paths", "created_by", "created_at", "updated_at", "effort")
    class ExtraFlagsEntry(_message.Message):
        __slots__ = ("key", "value")
        KEY_FIELD_NUMBER: _ClassVar[int]
        VALUE_FIELD_NUMBER: _ClassVar[int]
        key: str
        value: _types_pb2.ExtraFlagList
        def __init__(self, key: _Optional[str] = ..., value: _Optional[_Union[_types_pb2.ExtraFlagList, _Mapping]] = ...) -> None: ...
    ID_FIELD_NUMBER: _ClassVar[int]
    NAME_FIELD_NUMBER: _ClassVar[int]
    PROFILE_KEY_FIELD_NUMBER: _ClassVar[int]
    DESCRIPTION_FIELD_NUMBER: _ClassVar[int]
    ROLE_REF_FIELD_NUMBER: _ClassVar[int]
    MAX_TURNS_FIELD_NUMBER: _ClassVar[int]
    TIMEOUT_FIELD_NUMBER: _ClassVar[int]
    ALLOWED_TOOLS_FIELD_NUMBER: _ClassVar[int]
    DENIED_TOOLS_FIELD_NUMBER: _ClassVar[int]
    TOOL_RESTRICTION_POLICY_FIELD_NUMBER: _ClassVar[int]
    SKIP_PERMISSION_PROMPT_FIELD_NUMBER: _ClassVar[int]
    FEATURES_FIELD_NUMBER: _ClassVar[int]
    EXTRA_FLAGS_FIELD_NUMBER: _ClassVar[int]
    NETWORK_ACCESS_FIELD_NUMBER: _ClassVar[int]
    OWNER_SCENARIO_FIELD_NUMBER: _ClassVar[int]
    SOURCE_PATH_FIELD_NUMBER: _ClassVar[int]
    SOURCE_HASH_FIELD_NUMBER: _ClassVar[int]
    LAST_APPLIED_HASH_FIELD_NUMBER: _ClassVar[int]
    SOURCE_UPDATED_AT_FIELD_NUMBER: _ClassVar[int]
    LOCAL_OVERRIDE_FIELD_NUMBER: _ClassVar[int]
    SANDBOX_CONFIG_FIELD_NUMBER: _ClassVar[int]
    ALLOWED_PATHS_FIELD_NUMBER: _ClassVar[int]
    DENIED_PATHS_FIELD_NUMBER: _ClassVar[int]
    CREATED_BY_FIELD_NUMBER: _ClassVar[int]
    CREATED_AT_FIELD_NUMBER: _ClassVar[int]
    UPDATED_AT_FIELD_NUMBER: _ClassVar[int]
    EFFORT_FIELD_NUMBER: _ClassVar[int]
    id: str
    name: str
    profile_key: str
    description: str
    role_ref: str
    max_turns: int
    timeout: _duration_pb2.Duration
    allowed_tools: _containers.RepeatedScalarFieldContainer[str]
    denied_tools: _containers.RepeatedScalarFieldContainer[str]
    tool_restriction_policy: str
    skip_permission_prompt: bool
    features: _types_pb2.FeatureFlags
    extra_flags: _containers.MessageMap[str, _types_pb2.ExtraFlagList]
    network_access: _types_pb2.NetworkAccess
    owner_scenario: str
    source_path: str
    source_hash: str
    last_applied_hash: str
    source_updated_at: _timestamp_pb2.Timestamp
    local_override: bool
    sandbox_config: _types_pb2.SandboxConfig
    allowed_paths: _containers.RepeatedScalarFieldContainer[str]
    denied_paths: _containers.RepeatedScalarFieldContainer[str]
    created_by: str
    created_at: _timestamp_pb2.Timestamp
    updated_at: _timestamp_pb2.Timestamp
    effort: str
    def __init__(self, id: _Optional[str] = ..., name: _Optional[str] = ..., profile_key: _Optional[str] = ..., description: _Optional[str] = ..., role_ref: _Optional[str] = ..., max_turns: _Optional[int] = ..., timeout: _Optional[_Union[datetime.timedelta, _duration_pb2.Duration, _Mapping]] = ..., allowed_tools: _Optional[_Iterable[str]] = ..., denied_tools: _Optional[_Iterable[str]] = ..., tool_restriction_policy: _Optional[str] = ..., skip_permission_prompt: _Optional[bool] = ..., features: _Optional[_Union[_types_pb2.FeatureFlags, _Mapping]] = ..., extra_flags: _Optional[_Mapping[str, _types_pb2.ExtraFlagList]] = ..., network_access: _Optional[_Union[_types_pb2.NetworkAccess, str]] = ..., owner_scenario: _Optional[str] = ..., source_path: _Optional[str] = ..., source_hash: _Optional[str] = ..., last_applied_hash: _Optional[str] = ..., source_updated_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., local_override: _Optional[bool] = ..., sandbox_config: _Optional[_Union[_types_pb2.SandboxConfig, _Mapping]] = ..., allowed_paths: _Optional[_Iterable[str]] = ..., denied_paths: _Optional[_Iterable[str]] = ..., created_by: _Optional[str] = ..., created_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., updated_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., effort: _Optional[str] = ...) -> None: ...

class ResultSpec(_message.Message):
    __slots__ = ("version", "kind", "schema", "schema_digest", "classification_values", "extraction_mode", "extraction_role", "schema_repair_attempts")
    VERSION_FIELD_NUMBER: _ClassVar[int]
    KIND_FIELD_NUMBER: _ClassVar[int]
    SCHEMA_FIELD_NUMBER: _ClassVar[int]
    SCHEMA_DIGEST_FIELD_NUMBER: _ClassVar[int]
    CLASSIFICATION_VALUES_FIELD_NUMBER: _ClassVar[int]
    EXTRACTION_MODE_FIELD_NUMBER: _ClassVar[int]
    EXTRACTION_ROLE_FIELD_NUMBER: _ClassVar[int]
    SCHEMA_REPAIR_ATTEMPTS_FIELD_NUMBER: _ClassVar[int]
    version: str
    kind: ResultSpecKind
    schema: bytes
    schema_digest: str
    classification_values: _containers.RepeatedScalarFieldContainer[str]
    extraction_mode: StructuredExtractionMode
    extraction_role: str
    schema_repair_attempts: int
    def __init__(self, version: _Optional[str] = ..., kind: _Optional[_Union[ResultSpecKind, str]] = ..., schema: _Optional[bytes] = ..., schema_digest: _Optional[str] = ..., classification_values: _Optional[_Iterable[str]] = ..., extraction_mode: _Optional[_Union[StructuredExtractionMode, str]] = ..., extraction_role: _Optional[str] = ..., schema_repair_attempts: _Optional[int] = ...) -> None: ...

class RunConfig(_message.Message):
    __slots__ = ("runner_type", "model", "role_ref", "result_spec", "max_turns", "timeout", "allowed_tools", "denied_tools", "tool_restriction_policy", "effort", "skip_permission_prompt", "features", "extra_flags", "network_access", "policy_snapshot", "sandbox_config", "allowed_paths", "denied_paths")
    class ExtraFlagsEntry(_message.Message):
        __slots__ = ("key", "value")
        KEY_FIELD_NUMBER: _ClassVar[int]
        VALUE_FIELD_NUMBER: _ClassVar[int]
        key: str
        value: _types_pb2.ExtraFlagList
        def __init__(self, key: _Optional[str] = ..., value: _Optional[_Union[_types_pb2.ExtraFlagList, _Mapping]] = ...) -> None: ...
    RUNNER_TYPE_FIELD_NUMBER: _ClassVar[int]
    MODEL_FIELD_NUMBER: _ClassVar[int]
    ROLE_REF_FIELD_NUMBER: _ClassVar[int]
    RESULT_SPEC_FIELD_NUMBER: _ClassVar[int]
    MAX_TURNS_FIELD_NUMBER: _ClassVar[int]
    TIMEOUT_FIELD_NUMBER: _ClassVar[int]
    ALLOWED_TOOLS_FIELD_NUMBER: _ClassVar[int]
    DENIED_TOOLS_FIELD_NUMBER: _ClassVar[int]
    TOOL_RESTRICTION_POLICY_FIELD_NUMBER: _ClassVar[int]
    EFFORT_FIELD_NUMBER: _ClassVar[int]
    SKIP_PERMISSION_PROMPT_FIELD_NUMBER: _ClassVar[int]
    FEATURES_FIELD_NUMBER: _ClassVar[int]
    EXTRA_FLAGS_FIELD_NUMBER: _ClassVar[int]
    NETWORK_ACCESS_FIELD_NUMBER: _ClassVar[int]
    POLICY_SNAPSHOT_FIELD_NUMBER: _ClassVar[int]
    SANDBOX_CONFIG_FIELD_NUMBER: _ClassVar[int]
    ALLOWED_PATHS_FIELD_NUMBER: _ClassVar[int]
    DENIED_PATHS_FIELD_NUMBER: _ClassVar[int]
    runner_type: _types_pb2.RunnerType
    model: str
    role_ref: str
    result_spec: ResultSpec
    max_turns: int
    timeout: _duration_pb2.Duration
    allowed_tools: _containers.RepeatedScalarFieldContainer[str]
    denied_tools: _containers.RepeatedScalarFieldContainer[str]
    tool_restriction_policy: str
    effort: str
    skip_permission_prompt: bool
    features: _types_pb2.FeatureFlags
    extra_flags: _containers.MessageMap[str, _types_pb2.ExtraFlagList]
    network_access: _types_pb2.NetworkAccess
    policy_snapshot: ExecutionPolicySnapshot
    sandbox_config: _types_pb2.SandboxConfig
    allowed_paths: _containers.RepeatedScalarFieldContainer[str]
    denied_paths: _containers.RepeatedScalarFieldContainer[str]
    def __init__(self, runner_type: _Optional[_Union[_types_pb2.RunnerType, str]] = ..., model: _Optional[str] = ..., role_ref: _Optional[str] = ..., result_spec: _Optional[_Union[ResultSpec, _Mapping]] = ..., max_turns: _Optional[int] = ..., timeout: _Optional[_Union[datetime.timedelta, _duration_pb2.Duration, _Mapping]] = ..., allowed_tools: _Optional[_Iterable[str]] = ..., denied_tools: _Optional[_Iterable[str]] = ..., tool_restriction_policy: _Optional[str] = ..., effort: _Optional[str] = ..., skip_permission_prompt: _Optional[bool] = ..., features: _Optional[_Union[_types_pb2.FeatureFlags, _Mapping]] = ..., extra_flags: _Optional[_Mapping[str, _types_pb2.ExtraFlagList]] = ..., network_access: _Optional[_Union[_types_pb2.NetworkAccess, str]] = ..., policy_snapshot: _Optional[_Union[ExecutionPolicySnapshot, _Mapping]] = ..., sandbox_config: _Optional[_Union[_types_pb2.SandboxConfig, _Mapping]] = ..., allowed_paths: _Optional[_Iterable[str]] = ..., denied_paths: _Optional[_Iterable[str]] = ...) -> None: ...

class ExecutionCandidate(_message.Message):
    __slots__ = ("runner_type", "selection_type", "model", "resource_role", "fallbacks", "available", "failure_code", "failure", "provenance", "enforcement", "policy_path", "policy_digest")
    RUNNER_TYPE_FIELD_NUMBER: _ClassVar[int]
    SELECTION_TYPE_FIELD_NUMBER: _ClassVar[int]
    MODEL_FIELD_NUMBER: _ClassVar[int]
    RESOURCE_ROLE_FIELD_NUMBER: _ClassVar[int]
    FALLBACKS_FIELD_NUMBER: _ClassVar[int]
    AVAILABLE_FIELD_NUMBER: _ClassVar[int]
    FAILURE_CODE_FIELD_NUMBER: _ClassVar[int]
    FAILURE_FIELD_NUMBER: _ClassVar[int]
    PROVENANCE_FIELD_NUMBER: _ClassVar[int]
    ENFORCEMENT_FIELD_NUMBER: _ClassVar[int]
    POLICY_PATH_FIELD_NUMBER: _ClassVar[int]
    POLICY_DIGEST_FIELD_NUMBER: _ClassVar[int]
    runner_type: _types_pb2.RunnerType
    selection_type: _types_pb2.ModelSelectionType
    model: str
    resource_role: str
    fallbacks: _containers.RepeatedScalarFieldContainer[str]
    available: bool
    failure_code: str
    failure: str
    provenance: ResourceProvenance
    enforcement: PermissionEnforcement
    policy_path: str
    policy_digest: str
    def __init__(self, runner_type: _Optional[_Union[_types_pb2.RunnerType, str]] = ..., selection_type: _Optional[_Union[_types_pb2.ModelSelectionType, str]] = ..., model: _Optional[str] = ..., resource_role: _Optional[str] = ..., fallbacks: _Optional[_Iterable[str]] = ..., available: _Optional[bool] = ..., failure_code: _Optional[str] = ..., failure: _Optional[str] = ..., provenance: _Optional[_Union[ResourceProvenance, _Mapping]] = ..., enforcement: _Optional[_Union[PermissionEnforcement, _Mapping]] = ..., policy_path: _Optional[str] = ..., policy_digest: _Optional[str] = ...) -> None: ...

class ResourceProvenance(_message.Message):
    __slots__ = ("source", "observed_at")
    SOURCE_FIELD_NUMBER: _ClassVar[int]
    OBSERVED_AT_FIELD_NUMBER: _ClassVar[int]
    source: str
    observed_at: str
    def __init__(self, source: _Optional[str] = ..., observed_at: _Optional[str] = ...) -> None: ...

class PermissionEnforcement(_message.Message):
    __slots__ = ("permissions", "caveats")
    PERMISSIONS_FIELD_NUMBER: _ClassVar[int]
    CAVEATS_FIELD_NUMBER: _ClassVar[int]
    permissions: str
    caveats: _containers.RepeatedScalarFieldContainer[str]
    def __init__(self, permissions: _Optional[str] = ..., caveats: _Optional[_Iterable[str]] = ...) -> None: ...

class CandidatePreflight(_message.Message):
    __slots__ = ("index", "candidate", "available", "reason")
    INDEX_FIELD_NUMBER: _ClassVar[int]
    CANDIDATE_FIELD_NUMBER: _ClassVar[int]
    AVAILABLE_FIELD_NUMBER: _ClassVar[int]
    REASON_FIELD_NUMBER: _ClassVar[int]
    index: int
    candidate: ExecutionCandidate
    available: bool
    reason: str
    def __init__(self, index: _Optional[int] = ..., candidate: _Optional[_Union[ExecutionCandidate, _Mapping]] = ..., available: _Optional[bool] = ..., reason: _Optional[str] = ...) -> None: ...

class PolicyResolutionExplanation(_message.Message):
    __slots__ = ("source", "summary", "requested_runner", "requested_model", "requested_role_ref", "preflight")
    SOURCE_FIELD_NUMBER: _ClassVar[int]
    SUMMARY_FIELD_NUMBER: _ClassVar[int]
    REQUESTED_RUNNER_FIELD_NUMBER: _ClassVar[int]
    REQUESTED_MODEL_FIELD_NUMBER: _ClassVar[int]
    REQUESTED_ROLE_REF_FIELD_NUMBER: _ClassVar[int]
    PREFLIGHT_FIELD_NUMBER: _ClassVar[int]
    source: str
    summary: str
    requested_runner: _types_pb2.RunnerType
    requested_model: str
    requested_role_ref: str
    preflight: _containers.RepeatedCompositeFieldContainer[CandidatePreflight]
    def __init__(self, source: _Optional[str] = ..., summary: _Optional[str] = ..., requested_runner: _Optional[_Union[_types_pb2.RunnerType, str]] = ..., requested_model: _Optional[str] = ..., requested_role_ref: _Optional[str] = ..., preflight: _Optional[_Iterable[_Union[CandidatePreflight, _Mapping]]] = ...) -> None: ...

class ExecutionPolicySnapshot(_message.Message):
    __slots__ = ("catalog_digest", "candidates", "selected_index", "selected_candidate", "explanation", "role_ref")
    CATALOG_DIGEST_FIELD_NUMBER: _ClassVar[int]
    CANDIDATES_FIELD_NUMBER: _ClassVar[int]
    SELECTED_INDEX_FIELD_NUMBER: _ClassVar[int]
    SELECTED_CANDIDATE_FIELD_NUMBER: _ClassVar[int]
    EXPLANATION_FIELD_NUMBER: _ClassVar[int]
    ROLE_REF_FIELD_NUMBER: _ClassVar[int]
    catalog_digest: str
    candidates: _containers.RepeatedCompositeFieldContainer[ExecutionCandidate]
    selected_index: int
    selected_candidate: ExecutionCandidate
    explanation: PolicyResolutionExplanation
    role_ref: str
    def __init__(self, catalog_digest: _Optional[str] = ..., candidates: _Optional[_Iterable[_Union[ExecutionCandidate, _Mapping]]] = ..., selected_index: _Optional[int] = ..., selected_candidate: _Optional[_Union[ExecutionCandidate, _Mapping]] = ..., explanation: _Optional[_Union[PolicyResolutionExplanation, _Mapping]] = ..., role_ref: _Optional[str] = ...) -> None: ...

class RunConfigOverrides(_message.Message):
    __slots__ = ("role_ref", "result_spec", "max_turns", "timeout", "allowed_tools", "denied_tools", "skip_permission_prompt", "features", "extra_flags", "clear_extra_flags", "network_access", "effort", "sandbox_config", "allowed_paths", "denied_paths", "clear_allowed_tools", "clear_denied_tools", "clear_allowed_paths", "clear_denied_paths")
    class ExtraFlagsEntry(_message.Message):
        __slots__ = ("key", "value")
        KEY_FIELD_NUMBER: _ClassVar[int]
        VALUE_FIELD_NUMBER: _ClassVar[int]
        key: str
        value: _types_pb2.ExtraFlagList
        def __init__(self, key: _Optional[str] = ..., value: _Optional[_Union[_types_pb2.ExtraFlagList, _Mapping]] = ...) -> None: ...
    ROLE_REF_FIELD_NUMBER: _ClassVar[int]
    RESULT_SPEC_FIELD_NUMBER: _ClassVar[int]
    MAX_TURNS_FIELD_NUMBER: _ClassVar[int]
    TIMEOUT_FIELD_NUMBER: _ClassVar[int]
    ALLOWED_TOOLS_FIELD_NUMBER: _ClassVar[int]
    DENIED_TOOLS_FIELD_NUMBER: _ClassVar[int]
    SKIP_PERMISSION_PROMPT_FIELD_NUMBER: _ClassVar[int]
    FEATURES_FIELD_NUMBER: _ClassVar[int]
    EXTRA_FLAGS_FIELD_NUMBER: _ClassVar[int]
    CLEAR_EXTRA_FLAGS_FIELD_NUMBER: _ClassVar[int]
    NETWORK_ACCESS_FIELD_NUMBER: _ClassVar[int]
    EFFORT_FIELD_NUMBER: _ClassVar[int]
    SANDBOX_CONFIG_FIELD_NUMBER: _ClassVar[int]
    ALLOWED_PATHS_FIELD_NUMBER: _ClassVar[int]
    DENIED_PATHS_FIELD_NUMBER: _ClassVar[int]
    CLEAR_ALLOWED_TOOLS_FIELD_NUMBER: _ClassVar[int]
    CLEAR_DENIED_TOOLS_FIELD_NUMBER: _ClassVar[int]
    CLEAR_ALLOWED_PATHS_FIELD_NUMBER: _ClassVar[int]
    CLEAR_DENIED_PATHS_FIELD_NUMBER: _ClassVar[int]
    role_ref: str
    result_spec: ResultSpec
    max_turns: int
    timeout: _duration_pb2.Duration
    allowed_tools: _containers.RepeatedScalarFieldContainer[str]
    denied_tools: _containers.RepeatedScalarFieldContainer[str]
    skip_permission_prompt: bool
    features: _types_pb2.FeatureFlags
    extra_flags: _containers.MessageMap[str, _types_pb2.ExtraFlagList]
    clear_extra_flags: bool
    network_access: _types_pb2.NetworkAccess
    effort: str
    sandbox_config: _types_pb2.SandboxConfig
    allowed_paths: _containers.RepeatedScalarFieldContainer[str]
    denied_paths: _containers.RepeatedScalarFieldContainer[str]
    clear_allowed_tools: bool
    clear_denied_tools: bool
    clear_allowed_paths: bool
    clear_denied_paths: bool
    def __init__(self, role_ref: _Optional[str] = ..., result_spec: _Optional[_Union[ResultSpec, _Mapping]] = ..., max_turns: _Optional[int] = ..., timeout: _Optional[_Union[datetime.timedelta, _duration_pb2.Duration, _Mapping]] = ..., allowed_tools: _Optional[_Iterable[str]] = ..., denied_tools: _Optional[_Iterable[str]] = ..., skip_permission_prompt: _Optional[bool] = ..., features: _Optional[_Union[_types_pb2.FeatureFlags, _Mapping]] = ..., extra_flags: _Optional[_Mapping[str, _types_pb2.ExtraFlagList]] = ..., clear_extra_flags: _Optional[bool] = ..., network_access: _Optional[_Union[_types_pb2.NetworkAccess, str]] = ..., effort: _Optional[str] = ..., sandbox_config: _Optional[_Union[_types_pb2.SandboxConfig, _Mapping]] = ..., allowed_paths: _Optional[_Iterable[str]] = ..., denied_paths: _Optional[_Iterable[str]] = ..., clear_allowed_tools: _Optional[bool] = ..., clear_denied_tools: _Optional[bool] = ..., clear_allowed_paths: _Optional[bool] = ..., clear_denied_paths: _Optional[bool] = ...) -> None: ...

class HeartbeatConfig(_message.Message):
    __slots__ = ("interval", "timeout", "max_missed_beats")
    INTERVAL_FIELD_NUMBER: _ClassVar[int]
    TIMEOUT_FIELD_NUMBER: _ClassVar[int]
    MAX_MISSED_BEATS_FIELD_NUMBER: _ClassVar[int]
    interval: _duration_pb2.Duration
    timeout: _duration_pb2.Duration
    max_missed_beats: int
    def __init__(self, interval: _Optional[_Union[datetime.timedelta, _duration_pb2.Duration, _Mapping]] = ..., timeout: _Optional[_Union[datetime.timedelta, _duration_pb2.Duration, _Mapping]] = ..., max_missed_beats: _Optional[int] = ...) -> None: ...
