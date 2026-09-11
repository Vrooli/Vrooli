import datetime

from google.protobuf import struct_pb2 as _struct_pb2
from google.protobuf import timestamp_pb2 as _timestamp_pb2
from google.protobuf.internal import containers as _containers
from google.protobuf.internal import enum_type_wrapper as _enum_type_wrapper
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Iterable as _Iterable, Mapping as _Mapping
from typing import ClassVar as _ClassVar, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class CapabilityState(int, metaclass=_enum_type_wrapper.EnumTypeWrapper):
    __slots__ = ()
    CAPABILITY_STATE_UNSPECIFIED: _ClassVar[CapabilityState]
    CAPABILITY_STATE_DISCOVERED: _ClassVar[CapabilityState]
    CAPABILITY_STATE_NEEDS_OPERATOR_INPUT: _ClassVar[CapabilityState]
    CAPABILITY_STATE_READY_TO_PREVIEW: _ClassVar[CapabilityState]
    CAPABILITY_STATE_APPLYING: _ClassVar[CapabilityState]
    CAPABILITY_STATE_VERIFYING: _ClassVar[CapabilityState]
    CAPABILITY_STATE_READY: _ClassVar[CapabilityState]
    CAPABILITY_STATE_RETRYABLE_FAILURE: _ClassVar[CapabilityState]
    CAPABILITY_STATE_DEGRADED: _ClassVar[CapabilityState]
    CAPABILITY_STATE_UNSUPPORTED: _ClassVar[CapabilityState]
CAPABILITY_STATE_UNSPECIFIED: CapabilityState
CAPABILITY_STATE_DISCOVERED: CapabilityState
CAPABILITY_STATE_NEEDS_OPERATOR_INPUT: CapabilityState
CAPABILITY_STATE_READY_TO_PREVIEW: CapabilityState
CAPABILITY_STATE_APPLYING: CapabilityState
CAPABILITY_STATE_VERIFYING: CapabilityState
CAPABILITY_STATE_READY: CapabilityState
CAPABILITY_STATE_RETRYABLE_FAILURE: CapabilityState
CAPABILITY_STATE_DEGRADED: CapabilityState
CAPABILITY_STATE_UNSUPPORTED: CapabilityState

class ListCapabilitiesRequest(_message.Message):
    __slots__ = ("target",)
    TARGET_FIELD_NUMBER: _ClassVar[int]
    target: str
    def __init__(self, target: _Optional[str] = ...) -> None: ...

class GetCapabilityStatusRequest(_message.Message):
    __slots__ = ("target",)
    TARGET_FIELD_NUMBER: _ClassVar[int]
    target: str
    def __init__(self, target: _Optional[str] = ...) -> None: ...

class PreviewCapabilityRequest(_message.Message):
    __slots__ = ("target", "action")
    TARGET_FIELD_NUMBER: _ClassVar[int]
    ACTION_FIELD_NUMBER: _ClassVar[int]
    target: str
    action: ActionRequest
    def __init__(self, target: _Optional[str] = ..., action: _Optional[_Union[ActionRequest, _Mapping]] = ...) -> None: ...

class ApplyCapabilityRequest(_message.Message):
    __slots__ = ("target", "action")
    TARGET_FIELD_NUMBER: _ClassVar[int]
    ACTION_FIELD_NUMBER: _ClassVar[int]
    target: str
    action: ActionRequest
    def __init__(self, target: _Optional[str] = ..., action: _Optional[_Union[ActionRequest, _Mapping]] = ...) -> None: ...

class VerifyCapabilityRequest(_message.Message):
    __slots__ = ("target", "verification")
    TARGET_FIELD_NUMBER: _ClassVar[int]
    VERIFICATION_FIELD_NUMBER: _ClassVar[int]
    target: str
    verification: VerificationRequest
    def __init__(self, target: _Optional[str] = ..., verification: _Optional[_Union[VerificationRequest, _Mapping]] = ...) -> None: ...

class VerificationRequest(_message.Message):
    __slots__ = ("capability_id", "credential_ref", "target_id", "environment", "account_identity", "operation", "context", "effect_class", "max_operations", "cleanup_policy", "timeout_seconds", "context_digest", "catalog_revision", "configuration_revision", "provider_adapter_version")
    class ContextEntry(_message.Message):
        __slots__ = ("key", "value")
        KEY_FIELD_NUMBER: _ClassVar[int]
        VALUE_FIELD_NUMBER: _ClassVar[int]
        key: str
        value: str
        def __init__(self, key: _Optional[str] = ..., value: _Optional[str] = ...) -> None: ...
    CAPABILITY_ID_FIELD_NUMBER: _ClassVar[int]
    CREDENTIAL_REF_FIELD_NUMBER: _ClassVar[int]
    TARGET_ID_FIELD_NUMBER: _ClassVar[int]
    ENVIRONMENT_FIELD_NUMBER: _ClassVar[int]
    ACCOUNT_IDENTITY_FIELD_NUMBER: _ClassVar[int]
    OPERATION_FIELD_NUMBER: _ClassVar[int]
    CONTEXT_FIELD_NUMBER: _ClassVar[int]
    EFFECT_CLASS_FIELD_NUMBER: _ClassVar[int]
    MAX_OPERATIONS_FIELD_NUMBER: _ClassVar[int]
    CLEANUP_POLICY_FIELD_NUMBER: _ClassVar[int]
    TIMEOUT_SECONDS_FIELD_NUMBER: _ClassVar[int]
    CONTEXT_DIGEST_FIELD_NUMBER: _ClassVar[int]
    CATALOG_REVISION_FIELD_NUMBER: _ClassVar[int]
    CONFIGURATION_REVISION_FIELD_NUMBER: _ClassVar[int]
    PROVIDER_ADAPTER_VERSION_FIELD_NUMBER: _ClassVar[int]
    capability_id: str
    credential_ref: CredentialEvidenceRef
    target_id: str
    environment: str
    account_identity: str
    operation: str
    context: _containers.ScalarMap[str, str]
    effect_class: str
    max_operations: int
    cleanup_policy: str
    timeout_seconds: int
    context_digest: str
    catalog_revision: str
    configuration_revision: str
    provider_adapter_version: str
    def __init__(self, capability_id: _Optional[str] = ..., credential_ref: _Optional[_Union[CredentialEvidenceRef, _Mapping]] = ..., target_id: _Optional[str] = ..., environment: _Optional[str] = ..., account_identity: _Optional[str] = ..., operation: _Optional[str] = ..., context: _Optional[_Mapping[str, str]] = ..., effect_class: _Optional[str] = ..., max_operations: _Optional[int] = ..., cleanup_policy: _Optional[str] = ..., timeout_seconds: _Optional[int] = ..., context_digest: _Optional[str] = ..., catalog_revision: _Optional[str] = ..., configuration_revision: _Optional[str] = ..., provider_adapter_version: _Optional[str] = ...) -> None: ...

class ActionRequest(_message.Message):
    __slots__ = ("capability_id", "idempotency_key", "confirm", "inputs", "target_id")
    CAPABILITY_ID_FIELD_NUMBER: _ClassVar[int]
    IDEMPOTENCY_KEY_FIELD_NUMBER: _ClassVar[int]
    CONFIRM_FIELD_NUMBER: _ClassVar[int]
    INPUTS_FIELD_NUMBER: _ClassVar[int]
    TARGET_ID_FIELD_NUMBER: _ClassVar[int]
    capability_id: str
    idempotency_key: str
    confirm: bool
    inputs: _struct_pb2.Struct
    target_id: str
    def __init__(self, capability_id: _Optional[str] = ..., idempotency_key: _Optional[str] = ..., confirm: _Optional[bool] = ..., inputs: _Optional[_Union[_struct_pb2.Struct, _Mapping]] = ..., target_id: _Optional[str] = ...) -> None: ...

class CapabilityCandidate(_message.Message):
    __slots__ = ("id", "kind", "label", "location", "stable_identity", "device_identity", "writable", "physical_independence", "status", "risk", "remediation", "metadata")
    class MetadataEntry(_message.Message):
        __slots__ = ("key", "value")
        KEY_FIELD_NUMBER: _ClassVar[int]
        VALUE_FIELD_NUMBER: _ClassVar[int]
        key: str
        value: str
        def __init__(self, key: _Optional[str] = ..., value: _Optional[str] = ...) -> None: ...
    ID_FIELD_NUMBER: _ClassVar[int]
    KIND_FIELD_NUMBER: _ClassVar[int]
    LABEL_FIELD_NUMBER: _ClassVar[int]
    LOCATION_FIELD_NUMBER: _ClassVar[int]
    STABLE_IDENTITY_FIELD_NUMBER: _ClassVar[int]
    DEVICE_IDENTITY_FIELD_NUMBER: _ClassVar[int]
    WRITABLE_FIELD_NUMBER: _ClassVar[int]
    PHYSICAL_INDEPENDENCE_FIELD_NUMBER: _ClassVar[int]
    STATUS_FIELD_NUMBER: _ClassVar[int]
    RISK_FIELD_NUMBER: _ClassVar[int]
    REMEDIATION_FIELD_NUMBER: _ClassVar[int]
    METADATA_FIELD_NUMBER: _ClassVar[int]
    id: str
    kind: str
    label: str
    location: str
    stable_identity: str
    device_identity: str
    writable: bool
    physical_independence: str
    status: str
    risk: str
    remediation: str
    metadata: _containers.ScalarMap[str, str]
    def __init__(self, id: _Optional[str] = ..., kind: _Optional[str] = ..., label: _Optional[str] = ..., location: _Optional[str] = ..., stable_identity: _Optional[str] = ..., device_identity: _Optional[str] = ..., writable: _Optional[bool] = ..., physical_independence: _Optional[str] = ..., status: _Optional[str] = ..., risk: _Optional[str] = ..., remediation: _Optional[str] = ..., metadata: _Optional[_Mapping[str, str]] = ...) -> None: ...

class CapabilityInputDescriptor(_message.Message):
    __slots__ = ("id", "kind", "label", "description", "required", "options", "default_value", "candidates", "validation", "declinable", "constraints", "credential_logical_id", "credential_field", "provider", "requirement_group", "consumer_refs", "companion_settings", "acquisition_ref", "verification_ref", "recovery_ref", "help_ref", "evidence_policy", "companion_credentials")
    ID_FIELD_NUMBER: _ClassVar[int]
    KIND_FIELD_NUMBER: _ClassVar[int]
    LABEL_FIELD_NUMBER: _ClassVar[int]
    DESCRIPTION_FIELD_NUMBER: _ClassVar[int]
    REQUIRED_FIELD_NUMBER: _ClassVar[int]
    OPTIONS_FIELD_NUMBER: _ClassVar[int]
    DEFAULT_VALUE_FIELD_NUMBER: _ClassVar[int]
    CANDIDATES_FIELD_NUMBER: _ClassVar[int]
    VALIDATION_FIELD_NUMBER: _ClassVar[int]
    DECLINABLE_FIELD_NUMBER: _ClassVar[int]
    CONSTRAINTS_FIELD_NUMBER: _ClassVar[int]
    CREDENTIAL_LOGICAL_ID_FIELD_NUMBER: _ClassVar[int]
    CREDENTIAL_FIELD_FIELD_NUMBER: _ClassVar[int]
    PROVIDER_FIELD_NUMBER: _ClassVar[int]
    REQUIREMENT_GROUP_FIELD_NUMBER: _ClassVar[int]
    CONSUMER_REFS_FIELD_NUMBER: _ClassVar[int]
    COMPANION_SETTINGS_FIELD_NUMBER: _ClassVar[int]
    ACQUISITION_REF_FIELD_NUMBER: _ClassVar[int]
    VERIFICATION_REF_FIELD_NUMBER: _ClassVar[int]
    RECOVERY_REF_FIELD_NUMBER: _ClassVar[int]
    HELP_REF_FIELD_NUMBER: _ClassVar[int]
    EVIDENCE_POLICY_FIELD_NUMBER: _ClassVar[int]
    COMPANION_CREDENTIALS_FIELD_NUMBER: _ClassVar[int]
    id: str
    kind: str
    label: str
    description: str
    required: bool
    options: _containers.RepeatedScalarFieldContainer[str]
    default_value: str
    candidates: _containers.RepeatedCompositeFieldContainer[CapabilityCandidate]
    validation: str
    declinable: bool
    constraints: CapabilityInputConstraints
    credential_logical_id: str
    credential_field: str
    provider: str
    requirement_group: str
    consumer_refs: _containers.RepeatedScalarFieldContainer[str]
    companion_settings: _containers.RepeatedScalarFieldContainer[str]
    acquisition_ref: str
    verification_ref: str
    recovery_ref: str
    help_ref: str
    evidence_policy: str
    companion_credentials: _containers.RepeatedScalarFieldContainer[str]
    def __init__(self, id: _Optional[str] = ..., kind: _Optional[str] = ..., label: _Optional[str] = ..., description: _Optional[str] = ..., required: _Optional[bool] = ..., options: _Optional[_Iterable[str]] = ..., default_value: _Optional[str] = ..., candidates: _Optional[_Iterable[_Union[CapabilityCandidate, _Mapping]]] = ..., validation: _Optional[str] = ..., declinable: _Optional[bool] = ..., constraints: _Optional[_Union[CapabilityInputConstraints, _Mapping]] = ..., credential_logical_id: _Optional[str] = ..., credential_field: _Optional[str] = ..., provider: _Optional[str] = ..., requirement_group: _Optional[str] = ..., consumer_refs: _Optional[_Iterable[str]] = ..., companion_settings: _Optional[_Iterable[str]] = ..., acquisition_ref: _Optional[str] = ..., verification_ref: _Optional[str] = ..., recovery_ref: _Optional[str] = ..., help_ref: _Optional[str] = ..., evidence_policy: _Optional[str] = ..., companion_credentials: _Optional[_Iterable[str]] = ...) -> None: ...

class CapabilityInputConstraints(_message.Message):
    __slots__ = ("min_length", "max_length", "min_duration", "max_duration")
    MIN_LENGTH_FIELD_NUMBER: _ClassVar[int]
    MAX_LENGTH_FIELD_NUMBER: _ClassVar[int]
    MIN_DURATION_FIELD_NUMBER: _ClassVar[int]
    MAX_DURATION_FIELD_NUMBER: _ClassVar[int]
    min_length: int
    max_length: int
    min_duration: str
    max_duration: str
    def __init__(self, min_length: _Optional[int] = ..., max_length: _Optional[int] = ..., min_duration: _Optional[str] = ..., max_duration: _Optional[str] = ...) -> None: ...

class CapabilityPolicy(_message.Message):
    __slots__ = ("requires_confirmation", "idempotent", "retryable", "protected_roots", "remediation")
    REQUIRES_CONFIRMATION_FIELD_NUMBER: _ClassVar[int]
    IDEMPOTENT_FIELD_NUMBER: _ClassVar[int]
    RETRYABLE_FIELD_NUMBER: _ClassVar[int]
    PROTECTED_ROOTS_FIELD_NUMBER: _ClassVar[int]
    REMEDIATION_FIELD_NUMBER: _ClassVar[int]
    requires_confirmation: bool
    idempotent: bool
    retryable: bool
    protected_roots: _containers.RepeatedScalarFieldContainer[str]
    remediation: str
    def __init__(self, requires_confirmation: _Optional[bool] = ..., idempotent: _Optional[bool] = ..., retryable: _Optional[bool] = ..., protected_roots: _Optional[_Iterable[str]] = ..., remediation: _Optional[str] = ...) -> None: ...

class CapabilityEvidenceContract(_message.Message):
    __slots__ = ("kinds", "required_fields", "secret_free", "freshness", "stages")
    KINDS_FIELD_NUMBER: _ClassVar[int]
    REQUIRED_FIELDS_FIELD_NUMBER: _ClassVar[int]
    SECRET_FREE_FIELD_NUMBER: _ClassVar[int]
    FRESHNESS_FIELD_NUMBER: _ClassVar[int]
    STAGES_FIELD_NUMBER: _ClassVar[int]
    kinds: _containers.RepeatedScalarFieldContainer[str]
    required_fields: _containers.RepeatedScalarFieldContainer[str]
    secret_free: bool
    freshness: str
    stages: _containers.RepeatedScalarFieldContainer[str]
    def __init__(self, kinds: _Optional[_Iterable[str]] = ..., required_fields: _Optional[_Iterable[str]] = ..., secret_free: _Optional[bool] = ..., freshness: _Optional[str] = ..., stages: _Optional[_Iterable[str]] = ...) -> None: ...

class CapabilityDescriptor(_message.Message):
    __slots__ = ("version", "id", "owner", "title", "description", "risk", "inputs", "prerequisites", "policy", "evidence", "remediation", "scope", "purpose", "sensitivity", "applicability", "disposition", "disposition_reason", "provenance", "lifecycle", "reference_url")
    VERSION_FIELD_NUMBER: _ClassVar[int]
    ID_FIELD_NUMBER: _ClassVar[int]
    OWNER_FIELD_NUMBER: _ClassVar[int]
    TITLE_FIELD_NUMBER: _ClassVar[int]
    DESCRIPTION_FIELD_NUMBER: _ClassVar[int]
    RISK_FIELD_NUMBER: _ClassVar[int]
    INPUTS_FIELD_NUMBER: _ClassVar[int]
    PREREQUISITES_FIELD_NUMBER: _ClassVar[int]
    POLICY_FIELD_NUMBER: _ClassVar[int]
    EVIDENCE_FIELD_NUMBER: _ClassVar[int]
    REMEDIATION_FIELD_NUMBER: _ClassVar[int]
    SCOPE_FIELD_NUMBER: _ClassVar[int]
    PURPOSE_FIELD_NUMBER: _ClassVar[int]
    SENSITIVITY_FIELD_NUMBER: _ClassVar[int]
    APPLICABILITY_FIELD_NUMBER: _ClassVar[int]
    DISPOSITION_FIELD_NUMBER: _ClassVar[int]
    DISPOSITION_REASON_FIELD_NUMBER: _ClassVar[int]
    PROVENANCE_FIELD_NUMBER: _ClassVar[int]
    LIFECYCLE_FIELD_NUMBER: _ClassVar[int]
    REFERENCE_URL_FIELD_NUMBER: _ClassVar[int]
    version: str
    id: str
    owner: str
    title: str
    description: str
    risk: str
    inputs: _containers.RepeatedCompositeFieldContainer[CapabilityInputDescriptor]
    prerequisites: _containers.RepeatedScalarFieldContainer[str]
    policy: CapabilityPolicy
    evidence: CapabilityEvidenceContract
    remediation: str
    scope: str
    purpose: str
    sensitivity: str
    applicability: CapabilityApplicability
    disposition: str
    disposition_reason: str
    provenance: CapabilityPermissionProvenance
    lifecycle: CapabilityLifecycle
    reference_url: str
    def __init__(self, version: _Optional[str] = ..., id: _Optional[str] = ..., owner: _Optional[str] = ..., title: _Optional[str] = ..., description: _Optional[str] = ..., risk: _Optional[str] = ..., inputs: _Optional[_Iterable[_Union[CapabilityInputDescriptor, _Mapping]]] = ..., prerequisites: _Optional[_Iterable[str]] = ..., policy: _Optional[_Union[CapabilityPolicy, _Mapping]] = ..., evidence: _Optional[_Union[CapabilityEvidenceContract, _Mapping]] = ..., remediation: _Optional[str] = ..., scope: _Optional[str] = ..., purpose: _Optional[str] = ..., sensitivity: _Optional[str] = ..., applicability: _Optional[_Union[CapabilityApplicability, _Mapping]] = ..., disposition: _Optional[str] = ..., disposition_reason: _Optional[str] = ..., provenance: _Optional[_Union[CapabilityPermissionProvenance, _Mapping]] = ..., lifecycle: _Optional[_Union[CapabilityLifecycle, _Mapping]] = ..., reference_url: _Optional[str] = ...) -> None: ...

class CapabilityApplicability(_message.Message):
    __slots__ = ("platforms", "environments", "targets")
    PLATFORMS_FIELD_NUMBER: _ClassVar[int]
    ENVIRONMENTS_FIELD_NUMBER: _ClassVar[int]
    TARGETS_FIELD_NUMBER: _ClassVar[int]
    platforms: _containers.RepeatedScalarFieldContainer[str]
    environments: _containers.RepeatedScalarFieldContainer[str]
    targets: _containers.RepeatedScalarFieldContainer[str]
    def __init__(self, platforms: _Optional[_Iterable[str]] = ..., environments: _Optional[_Iterable[str]] = ..., targets: _Optional[_Iterable[str]] = ...) -> None: ...

class CapabilityPermissionProvenance(_message.Message):
    __slots__ = ("requester", "scope", "grant_source", "revocation_limit")
    REQUESTER_FIELD_NUMBER: _ClassVar[int]
    SCOPE_FIELD_NUMBER: _ClassVar[int]
    GRANT_SOURCE_FIELD_NUMBER: _ClassVar[int]
    REVOCATION_LIMIT_FIELD_NUMBER: _ClassVar[int]
    requester: str
    scope: str
    grant_source: str
    revocation_limit: str
    def __init__(self, requester: _Optional[str] = ..., scope: _Optional[str] = ..., grant_source: _Optional[str] = ..., revocation_limit: _Optional[str] = ...) -> None: ...

class CapabilityLifecycle(_message.Message):
    __slots__ = ("preview", "apply", "verify", "revoke", "recover", "recovery")
    PREVIEW_FIELD_NUMBER: _ClassVar[int]
    APPLY_FIELD_NUMBER: _ClassVar[int]
    VERIFY_FIELD_NUMBER: _ClassVar[int]
    REVOKE_FIELD_NUMBER: _ClassVar[int]
    RECOVER_FIELD_NUMBER: _ClassVar[int]
    RECOVERY_FIELD_NUMBER: _ClassVar[int]
    preview: bool
    apply: bool
    verify: bool
    revoke: bool
    recover: bool
    recovery: str
    def __init__(self, preview: _Optional[bool] = ..., apply: _Optional[bool] = ..., verify: _Optional[bool] = ..., revoke: _Optional[bool] = ..., recover: _Optional[bool] = ..., recovery: _Optional[str] = ...) -> None: ...

class CapabilityEvidence(_message.Message):
    __slots__ = ("kind", "artifact_identity", "source_generation", "checksum", "coverage", "observed_at", "verified", "remediation", "schema_version", "capability_id", "credential_ref", "target_id", "environment", "account_identity", "operation", "status", "expires_at", "artifact_refs", "limitations", "next_action", "effect_class", "effects_used", "cleanup_completed", "stage", "owner", "context_digest", "catalog_revision", "configuration_revision", "provider_adapter_version")
    KIND_FIELD_NUMBER: _ClassVar[int]
    ARTIFACT_IDENTITY_FIELD_NUMBER: _ClassVar[int]
    SOURCE_GENERATION_FIELD_NUMBER: _ClassVar[int]
    CHECKSUM_FIELD_NUMBER: _ClassVar[int]
    COVERAGE_FIELD_NUMBER: _ClassVar[int]
    OBSERVED_AT_FIELD_NUMBER: _ClassVar[int]
    VERIFIED_FIELD_NUMBER: _ClassVar[int]
    REMEDIATION_FIELD_NUMBER: _ClassVar[int]
    SCHEMA_VERSION_FIELD_NUMBER: _ClassVar[int]
    CAPABILITY_ID_FIELD_NUMBER: _ClassVar[int]
    CREDENTIAL_REF_FIELD_NUMBER: _ClassVar[int]
    TARGET_ID_FIELD_NUMBER: _ClassVar[int]
    ENVIRONMENT_FIELD_NUMBER: _ClassVar[int]
    ACCOUNT_IDENTITY_FIELD_NUMBER: _ClassVar[int]
    OPERATION_FIELD_NUMBER: _ClassVar[int]
    STATUS_FIELD_NUMBER: _ClassVar[int]
    EXPIRES_AT_FIELD_NUMBER: _ClassVar[int]
    ARTIFACT_REFS_FIELD_NUMBER: _ClassVar[int]
    LIMITATIONS_FIELD_NUMBER: _ClassVar[int]
    NEXT_ACTION_FIELD_NUMBER: _ClassVar[int]
    EFFECT_CLASS_FIELD_NUMBER: _ClassVar[int]
    EFFECTS_USED_FIELD_NUMBER: _ClassVar[int]
    CLEANUP_COMPLETED_FIELD_NUMBER: _ClassVar[int]
    STAGE_FIELD_NUMBER: _ClassVar[int]
    OWNER_FIELD_NUMBER: _ClassVar[int]
    CONTEXT_DIGEST_FIELD_NUMBER: _ClassVar[int]
    CATALOG_REVISION_FIELD_NUMBER: _ClassVar[int]
    CONFIGURATION_REVISION_FIELD_NUMBER: _ClassVar[int]
    PROVIDER_ADAPTER_VERSION_FIELD_NUMBER: _ClassVar[int]
    kind: str
    artifact_identity: str
    source_generation: str
    checksum: str
    coverage: _containers.RepeatedScalarFieldContainer[str]
    observed_at: _timestamp_pb2.Timestamp
    verified: bool
    remediation: str
    schema_version: str
    capability_id: str
    credential_ref: CredentialEvidenceRef
    target_id: str
    environment: str
    account_identity: str
    operation: str
    status: str
    expires_at: _timestamp_pb2.Timestamp
    artifact_refs: _containers.RepeatedScalarFieldContainer[str]
    limitations: _containers.RepeatedScalarFieldContainer[str]
    next_action: str
    effect_class: str
    effects_used: int
    cleanup_completed: bool
    stage: str
    owner: str
    context_digest: str
    catalog_revision: str
    configuration_revision: str
    provider_adapter_version: str
    def __init__(self, kind: _Optional[str] = ..., artifact_identity: _Optional[str] = ..., source_generation: _Optional[str] = ..., checksum: _Optional[str] = ..., coverage: _Optional[_Iterable[str]] = ..., observed_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., verified: _Optional[bool] = ..., remediation: _Optional[str] = ..., schema_version: _Optional[str] = ..., capability_id: _Optional[str] = ..., credential_ref: _Optional[_Union[CredentialEvidenceRef, _Mapping]] = ..., target_id: _Optional[str] = ..., environment: _Optional[str] = ..., account_identity: _Optional[str] = ..., operation: _Optional[str] = ..., status: _Optional[str] = ..., expires_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., artifact_refs: _Optional[_Iterable[str]] = ..., limitations: _Optional[_Iterable[str]] = ..., next_action: _Optional[str] = ..., effect_class: _Optional[str] = ..., effects_used: _Optional[int] = ..., cleanup_completed: _Optional[bool] = ..., stage: _Optional[str] = ..., owner: _Optional[str] = ..., context_digest: _Optional[str] = ..., catalog_revision: _Optional[str] = ..., configuration_revision: _Optional[str] = ..., provider_adapter_version: _Optional[str] = ...) -> None: ...

class CredentialEvidenceRef(_message.Message):
    __slots__ = ("logical_id", "field", "version")
    LOGICAL_ID_FIELD_NUMBER: _ClassVar[int]
    FIELD_FIELD_NUMBER: _ClassVar[int]
    VERSION_FIELD_NUMBER: _ClassVar[int]
    logical_id: str
    field: str
    version: str
    def __init__(self, logical_id: _Optional[str] = ..., field: _Optional[str] = ..., version: _Optional[str] = ...) -> None: ...

class CapabilityStatus(_message.Message):
    __slots__ = ("descriptor", "state", "candidates", "missing_inputs", "evidence", "remediation", "updated_at")
    DESCRIPTOR_FIELD_NUMBER: _ClassVar[int]
    STATE_FIELD_NUMBER: _ClassVar[int]
    CANDIDATES_FIELD_NUMBER: _ClassVar[int]
    MISSING_INPUTS_FIELD_NUMBER: _ClassVar[int]
    EVIDENCE_FIELD_NUMBER: _ClassVar[int]
    REMEDIATION_FIELD_NUMBER: _ClassVar[int]
    UPDATED_AT_FIELD_NUMBER: _ClassVar[int]
    descriptor: CapabilityDescriptor
    state: CapabilityState
    candidates: _containers.RepeatedCompositeFieldContainer[CapabilityCandidate]
    missing_inputs: _containers.RepeatedScalarFieldContainer[str]
    evidence: _containers.RepeatedCompositeFieldContainer[CapabilityEvidence]
    remediation: str
    updated_at: _timestamp_pb2.Timestamp
    def __init__(self, descriptor: _Optional[_Union[CapabilityDescriptor, _Mapping]] = ..., state: _Optional[_Union[CapabilityState, str]] = ..., candidates: _Optional[_Iterable[_Union[CapabilityCandidate, _Mapping]]] = ..., missing_inputs: _Optional[_Iterable[str]] = ..., evidence: _Optional[_Iterable[_Union[CapabilityEvidence, _Mapping]]] = ..., remediation: _Optional[str] = ..., updated_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ...) -> None: ...

class ListCapabilitiesResponse(_message.Message):
    __slots__ = ("capabilities", "count")
    CAPABILITIES_FIELD_NUMBER: _ClassVar[int]
    COUNT_FIELD_NUMBER: _ClassVar[int]
    capabilities: _containers.RepeatedCompositeFieldContainer[CapabilityStatus]
    count: int
    def __init__(self, capabilities: _Optional[_Iterable[_Union[CapabilityStatus, _Mapping]]] = ..., count: _Optional[int] = ...) -> None: ...

class GetCapabilityStatusResponse(_message.Message):
    __slots__ = ("statuses", "count")
    STATUSES_FIELD_NUMBER: _ClassVar[int]
    COUNT_FIELD_NUMBER: _ClassVar[int]
    statuses: _containers.RepeatedCompositeFieldContainer[CapabilityStatus]
    count: int
    def __init__(self, statuses: _Optional[_Iterable[_Union[CapabilityStatus, _Mapping]]] = ..., count: _Optional[int] = ...) -> None: ...

class CapabilityMutation(_message.Message):
    __slots__ = ("id", "summary", "reversible")
    ID_FIELD_NUMBER: _ClassVar[int]
    SUMMARY_FIELD_NUMBER: _ClassVar[int]
    REVERSIBLE_FIELD_NUMBER: _ClassVar[int]
    id: str
    summary: str
    reversible: bool
    def __init__(self, id: _Optional[str] = ..., summary: _Optional[str] = ..., reversible: _Optional[bool] = ...) -> None: ...

class PreviewCapabilityResponse(_message.Message):
    __slots__ = ("capability_id", "plan_id", "state", "mutations", "candidates", "remediation", "expires_at")
    CAPABILITY_ID_FIELD_NUMBER: _ClassVar[int]
    PLAN_ID_FIELD_NUMBER: _ClassVar[int]
    STATE_FIELD_NUMBER: _ClassVar[int]
    MUTATIONS_FIELD_NUMBER: _ClassVar[int]
    CANDIDATES_FIELD_NUMBER: _ClassVar[int]
    REMEDIATION_FIELD_NUMBER: _ClassVar[int]
    EXPIRES_AT_FIELD_NUMBER: _ClassVar[int]
    capability_id: str
    plan_id: str
    state: CapabilityState
    mutations: _containers.RepeatedCompositeFieldContainer[CapabilityMutation]
    candidates: _containers.RepeatedCompositeFieldContainer[CapabilityCandidate]
    remediation: str
    expires_at: _timestamp_pb2.Timestamp
    def __init__(self, capability_id: _Optional[str] = ..., plan_id: _Optional[str] = ..., state: _Optional[_Union[CapabilityState, str]] = ..., mutations: _Optional[_Iterable[_Union[CapabilityMutation, _Mapping]]] = ..., candidates: _Optional[_Iterable[_Union[CapabilityCandidate, _Mapping]]] = ..., remediation: _Optional[str] = ..., expires_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ...) -> None: ...

class ApplyCapabilityResponse(_message.Message):
    __slots__ = ("capability_id", "state", "outcome", "retryable", "error_code", "remediation", "evidence", "mutations", "completed_at")
    CAPABILITY_ID_FIELD_NUMBER: _ClassVar[int]
    STATE_FIELD_NUMBER: _ClassVar[int]
    OUTCOME_FIELD_NUMBER: _ClassVar[int]
    RETRYABLE_FIELD_NUMBER: _ClassVar[int]
    ERROR_CODE_FIELD_NUMBER: _ClassVar[int]
    REMEDIATION_FIELD_NUMBER: _ClassVar[int]
    EVIDENCE_FIELD_NUMBER: _ClassVar[int]
    MUTATIONS_FIELD_NUMBER: _ClassVar[int]
    COMPLETED_AT_FIELD_NUMBER: _ClassVar[int]
    capability_id: str
    state: CapabilityState
    outcome: str
    retryable: bool
    error_code: str
    remediation: str
    evidence: _containers.RepeatedCompositeFieldContainer[CapabilityEvidence]
    mutations: _containers.RepeatedCompositeFieldContainer[CapabilityMutation]
    completed_at: _timestamp_pb2.Timestamp
    def __init__(self, capability_id: _Optional[str] = ..., state: _Optional[_Union[CapabilityState, str]] = ..., outcome: _Optional[str] = ..., retryable: _Optional[bool] = ..., error_code: _Optional[str] = ..., remediation: _Optional[str] = ..., evidence: _Optional[_Iterable[_Union[CapabilityEvidence, _Mapping]]] = ..., mutations: _Optional[_Iterable[_Union[CapabilityMutation, _Mapping]]] = ..., completed_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ...) -> None: ...

class VerifyCapabilityResponse(_message.Message):
    __slots__ = ("capability_id", "evidence", "remediation", "error_code", "retryable", "retry_after_seconds", "next_action", "outcome")
    CAPABILITY_ID_FIELD_NUMBER: _ClassVar[int]
    EVIDENCE_FIELD_NUMBER: _ClassVar[int]
    REMEDIATION_FIELD_NUMBER: _ClassVar[int]
    ERROR_CODE_FIELD_NUMBER: _ClassVar[int]
    RETRYABLE_FIELD_NUMBER: _ClassVar[int]
    RETRY_AFTER_SECONDS_FIELD_NUMBER: _ClassVar[int]
    NEXT_ACTION_FIELD_NUMBER: _ClassVar[int]
    OUTCOME_FIELD_NUMBER: _ClassVar[int]
    capability_id: str
    evidence: _containers.RepeatedCompositeFieldContainer[CapabilityEvidence]
    remediation: str
    error_code: str
    retryable: bool
    retry_after_seconds: int
    next_action: str
    outcome: str
    def __init__(self, capability_id: _Optional[str] = ..., evidence: _Optional[_Iterable[_Union[CapabilityEvidence, _Mapping]]] = ..., remediation: _Optional[str] = ..., error_code: _Optional[str] = ..., retryable: _Optional[bool] = ..., retry_after_seconds: _Optional[int] = ..., next_action: _Optional[str] = ..., outcome: _Optional[str] = ...) -> None: ...
