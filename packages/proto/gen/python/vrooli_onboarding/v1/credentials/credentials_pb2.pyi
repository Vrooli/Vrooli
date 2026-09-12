from buf.validate import validate_pb2 as _validate_pb2
from google.protobuf.internal import containers as _containers
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Iterable as _Iterable, Mapping as _Mapping
from typing import ClassVar as _ClassVar, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class ListCredentialsRequest(_message.Message):
    __slots__ = ("target",)
    TARGET_FIELD_NUMBER: _ClassVar[int]
    target: str
    def __init__(self, target: _Optional[str] = ...) -> None: ...

class ProvisionCredentialRequest(_message.Message):
    __slots__ = ("target", "logical_id", "field", "value")
    TARGET_FIELD_NUMBER: _ClassVar[int]
    LOGICAL_ID_FIELD_NUMBER: _ClassVar[int]
    FIELD_FIELD_NUMBER: _ClassVar[int]
    VALUE_FIELD_NUMBER: _ClassVar[int]
    target: str
    logical_id: str
    field: str
    value: str
    def __init__(self, target: _Optional[str] = ..., logical_id: _Optional[str] = ..., field: _Optional[str] = ..., value: _Optional[str] = ...) -> None: ...

class DiagnoseCredentialsRequest(_message.Message):
    __slots__ = ("target",)
    TARGET_FIELD_NUMBER: _ClassVar[int]
    target: str
    def __init__(self, target: _Optional[str] = ...) -> None: ...

class CredentialConsumerProvenance(_message.Message):
    __slots__ = ("logical_id", "address_pattern", "field", "kind", "consumer", "source_ref", "required", "reason", "tiers")
    LOGICAL_ID_FIELD_NUMBER: _ClassVar[int]
    ADDRESS_PATTERN_FIELD_NUMBER: _ClassVar[int]
    FIELD_FIELD_NUMBER: _ClassVar[int]
    KIND_FIELD_NUMBER: _ClassVar[int]
    CONSUMER_FIELD_NUMBER: _ClassVar[int]
    SOURCE_REF_FIELD_NUMBER: _ClassVar[int]
    REQUIRED_FIELD_NUMBER: _ClassVar[int]
    REASON_FIELD_NUMBER: _ClassVar[int]
    TIERS_FIELD_NUMBER: _ClassVar[int]
    logical_id: str
    address_pattern: str
    field: str
    kind: str
    consumer: str
    source_ref: str
    required: bool
    reason: str
    tiers: _containers.RepeatedScalarFieldContainer[str]
    def __init__(self, logical_id: _Optional[str] = ..., address_pattern: _Optional[str] = ..., field: _Optional[str] = ..., kind: _Optional[str] = ..., consumer: _Optional[str] = ..., source_ref: _Optional[str] = ..., required: _Optional[bool] = ..., reason: _Optional[str] = ..., tiers: _Optional[_Iterable[str]] = ...) -> None: ...

class CredentialApplicabilityMatch(_message.Message):
    __slots__ = ("context", "setting", "operation", "role", "target", "environment", "capability", "provider", "value")
    CONTEXT_FIELD_NUMBER: _ClassVar[int]
    SETTING_FIELD_NUMBER: _ClassVar[int]
    OPERATION_FIELD_NUMBER: _ClassVar[int]
    ROLE_FIELD_NUMBER: _ClassVar[int]
    TARGET_FIELD_NUMBER: _ClassVar[int]
    ENVIRONMENT_FIELD_NUMBER: _ClassVar[int]
    CAPABILITY_FIELD_NUMBER: _ClassVar[int]
    PROVIDER_FIELD_NUMBER: _ClassVar[int]
    VALUE_FIELD_NUMBER: _ClassVar[int]
    context: str
    setting: str
    operation: str
    role: str
    target: str
    environment: str
    capability: str
    provider: str
    value: str
    def __init__(self, context: _Optional[str] = ..., setting: _Optional[str] = ..., operation: _Optional[str] = ..., role: _Optional[str] = ..., target: _Optional[str] = ..., environment: _Optional[str] = ..., capability: _Optional[str] = ..., provider: _Optional[str] = ..., value: _Optional[str] = ...) -> None: ...

class CredentialApplicabilityRule(_message.Message):
    __slots__ = ("eq", "neq")
    EQ_FIELD_NUMBER: _ClassVar[int]
    NEQ_FIELD_NUMBER: _ClassVar[int]
    eq: CredentialApplicabilityMatch
    neq: CredentialApplicabilityMatch
    def __init__(self, eq: _Optional[_Union[CredentialApplicabilityMatch, _Mapping]] = ..., neq: _Optional[_Union[CredentialApplicabilityMatch, _Mapping]] = ...) -> None: ...

class CredentialApplicability(_message.Message):
    __slots__ = ("all", "any")
    ALL_FIELD_NUMBER: _ClassVar[int]
    ANY_FIELD_NUMBER: _ClassVar[int]
    NOT_FIELD_NUMBER: _ClassVar[int]
    all: _containers.RepeatedCompositeFieldContainer[CredentialApplicabilityRule]
    any: _containers.RepeatedCompositeFieldContainer[CredentialApplicabilityRule]
    def __init__(self, all: _Optional[_Iterable[_Union[CredentialApplicabilityRule, _Mapping]]] = ..., any: _Optional[_Iterable[_Union[CredentialApplicabilityRule, _Mapping]]] = ..., **kwargs) -> None: ...

class CredentialMigrationDiagnostic(_message.Message):
    __slots__ = ("address", "code", "severity", "message")
    ADDRESS_FIELD_NUMBER: _ClassVar[int]
    CODE_FIELD_NUMBER: _ClassVar[int]
    SEVERITY_FIELD_NUMBER: _ClassVar[int]
    MESSAGE_FIELD_NUMBER: _ClassVar[int]
    address: str
    code: str
    severity: str
    message: str
    def __init__(self, address: _Optional[str] = ..., code: _Optional[str] = ..., severity: _Optional[str] = ..., message: _Optional[str] = ...) -> None: ...

class CredentialProvenance(_message.Message):
    __slots__ = ("owner", "source_ref", "kind", "env", "label", "description", "obtain_url", "provisioning", "derived_from", "required", "consumers", "version", "provider", "applies_when", "requirement_group", "consumer_refs", "companion_settings", "companion_credentials", "acquisition_ref", "verification_ref", "recovery_ref", "help_ref", "evidence_policy", "provider_version", "tiers", "placeholder")
    OWNER_FIELD_NUMBER: _ClassVar[int]
    SOURCE_REF_FIELD_NUMBER: _ClassVar[int]
    KIND_FIELD_NUMBER: _ClassVar[int]
    ENV_FIELD_NUMBER: _ClassVar[int]
    LABEL_FIELD_NUMBER: _ClassVar[int]
    DESCRIPTION_FIELD_NUMBER: _ClassVar[int]
    OBTAIN_URL_FIELD_NUMBER: _ClassVar[int]
    PROVISIONING_FIELD_NUMBER: _ClassVar[int]
    DERIVED_FROM_FIELD_NUMBER: _ClassVar[int]
    REQUIRED_FIELD_NUMBER: _ClassVar[int]
    CONSUMERS_FIELD_NUMBER: _ClassVar[int]
    VERSION_FIELD_NUMBER: _ClassVar[int]
    PROVIDER_FIELD_NUMBER: _ClassVar[int]
    APPLIES_WHEN_FIELD_NUMBER: _ClassVar[int]
    REQUIREMENT_GROUP_FIELD_NUMBER: _ClassVar[int]
    CONSUMER_REFS_FIELD_NUMBER: _ClassVar[int]
    COMPANION_SETTINGS_FIELD_NUMBER: _ClassVar[int]
    COMPANION_CREDENTIALS_FIELD_NUMBER: _ClassVar[int]
    ACQUISITION_REF_FIELD_NUMBER: _ClassVar[int]
    VERIFICATION_REF_FIELD_NUMBER: _ClassVar[int]
    RECOVERY_REF_FIELD_NUMBER: _ClassVar[int]
    HELP_REF_FIELD_NUMBER: _ClassVar[int]
    EVIDENCE_POLICY_FIELD_NUMBER: _ClassVar[int]
    PROVIDER_VERSION_FIELD_NUMBER: _ClassVar[int]
    TIERS_FIELD_NUMBER: _ClassVar[int]
    PLACEHOLDER_FIELD_NUMBER: _ClassVar[int]
    owner: str
    source_ref: str
    kind: str
    env: str
    label: str
    description: str
    obtain_url: str
    provisioning: str
    derived_from: str
    required: bool
    consumers: _containers.RepeatedCompositeFieldContainer[CredentialConsumerProvenance]
    version: str
    provider: str
    applies_when: CredentialApplicability
    requirement_group: str
    consumer_refs: _containers.RepeatedScalarFieldContainer[str]
    companion_settings: _containers.RepeatedScalarFieldContainer[str]
    companion_credentials: _containers.RepeatedScalarFieldContainer[str]
    acquisition_ref: str
    verification_ref: str
    recovery_ref: str
    help_ref: str
    evidence_policy: str
    provider_version: str
    tiers: _containers.RepeatedScalarFieldContainer[str]
    placeholder: str
    def __init__(self, owner: _Optional[str] = ..., source_ref: _Optional[str] = ..., kind: _Optional[str] = ..., env: _Optional[str] = ..., label: _Optional[str] = ..., description: _Optional[str] = ..., obtain_url: _Optional[str] = ..., provisioning: _Optional[str] = ..., derived_from: _Optional[str] = ..., required: _Optional[bool] = ..., consumers: _Optional[_Iterable[_Union[CredentialConsumerProvenance, _Mapping]]] = ..., version: _Optional[str] = ..., provider: _Optional[str] = ..., applies_when: _Optional[_Union[CredentialApplicability, _Mapping]] = ..., requirement_group: _Optional[str] = ..., consumer_refs: _Optional[_Iterable[str]] = ..., companion_settings: _Optional[_Iterable[str]] = ..., companion_credentials: _Optional[_Iterable[str]] = ..., acquisition_ref: _Optional[str] = ..., verification_ref: _Optional[str] = ..., recovery_ref: _Optional[str] = ..., help_ref: _Optional[str] = ..., evidence_policy: _Optional[str] = ..., provider_version: _Optional[str] = ..., tiers: _Optional[_Iterable[str]] = ..., placeholder: _Optional[str] = ...) -> None: ...

class Credential(_message.Message):
    __slots__ = ("resource", "logical_id", "field", "label", "description", "obtain_url", "placeholder", "provisioning", "derived_from", "required", "status", "detail", "provenance", "owner", "source_ref", "kind", "consumer_refs", "version", "provider", "applies_when", "requirement_group", "companion_settings", "companion_credentials", "acquisition_ref", "verification_ref", "recovery_ref", "help_ref", "evidence_policy", "provider_version", "migration_diagnostics", "tiers")
    RESOURCE_FIELD_NUMBER: _ClassVar[int]
    LOGICAL_ID_FIELD_NUMBER: _ClassVar[int]
    FIELD_FIELD_NUMBER: _ClassVar[int]
    LABEL_FIELD_NUMBER: _ClassVar[int]
    DESCRIPTION_FIELD_NUMBER: _ClassVar[int]
    OBTAIN_URL_FIELD_NUMBER: _ClassVar[int]
    PLACEHOLDER_FIELD_NUMBER: _ClassVar[int]
    PROVISIONING_FIELD_NUMBER: _ClassVar[int]
    DERIVED_FROM_FIELD_NUMBER: _ClassVar[int]
    REQUIRED_FIELD_NUMBER: _ClassVar[int]
    STATUS_FIELD_NUMBER: _ClassVar[int]
    DETAIL_FIELD_NUMBER: _ClassVar[int]
    PROVENANCE_FIELD_NUMBER: _ClassVar[int]
    OWNER_FIELD_NUMBER: _ClassVar[int]
    SOURCE_REF_FIELD_NUMBER: _ClassVar[int]
    KIND_FIELD_NUMBER: _ClassVar[int]
    CONSUMER_REFS_FIELD_NUMBER: _ClassVar[int]
    VERSION_FIELD_NUMBER: _ClassVar[int]
    PROVIDER_FIELD_NUMBER: _ClassVar[int]
    APPLIES_WHEN_FIELD_NUMBER: _ClassVar[int]
    REQUIREMENT_GROUP_FIELD_NUMBER: _ClassVar[int]
    COMPANION_SETTINGS_FIELD_NUMBER: _ClassVar[int]
    COMPANION_CREDENTIALS_FIELD_NUMBER: _ClassVar[int]
    ACQUISITION_REF_FIELD_NUMBER: _ClassVar[int]
    VERIFICATION_REF_FIELD_NUMBER: _ClassVar[int]
    RECOVERY_REF_FIELD_NUMBER: _ClassVar[int]
    HELP_REF_FIELD_NUMBER: _ClassVar[int]
    EVIDENCE_POLICY_FIELD_NUMBER: _ClassVar[int]
    PROVIDER_VERSION_FIELD_NUMBER: _ClassVar[int]
    MIGRATION_DIAGNOSTICS_FIELD_NUMBER: _ClassVar[int]
    TIERS_FIELD_NUMBER: _ClassVar[int]
    resource: str
    logical_id: str
    field: str
    label: str
    description: str
    obtain_url: str
    placeholder: str
    provisioning: str
    derived_from: str
    required: bool
    status: str
    detail: str
    provenance: _containers.RepeatedCompositeFieldContainer[CredentialProvenance]
    owner: str
    source_ref: str
    kind: str
    consumer_refs: _containers.RepeatedScalarFieldContainer[str]
    version: str
    provider: str
    applies_when: CredentialApplicability
    requirement_group: str
    companion_settings: _containers.RepeatedScalarFieldContainer[str]
    companion_credentials: _containers.RepeatedScalarFieldContainer[str]
    acquisition_ref: str
    verification_ref: str
    recovery_ref: str
    help_ref: str
    evidence_policy: str
    provider_version: str
    migration_diagnostics: _containers.RepeatedCompositeFieldContainer[CredentialMigrationDiagnostic]
    tiers: _containers.RepeatedScalarFieldContainer[str]
    def __init__(self, resource: _Optional[str] = ..., logical_id: _Optional[str] = ..., field: _Optional[str] = ..., label: _Optional[str] = ..., description: _Optional[str] = ..., obtain_url: _Optional[str] = ..., placeholder: _Optional[str] = ..., provisioning: _Optional[str] = ..., derived_from: _Optional[str] = ..., required: _Optional[bool] = ..., status: _Optional[str] = ..., detail: _Optional[str] = ..., provenance: _Optional[_Iterable[_Union[CredentialProvenance, _Mapping]]] = ..., owner: _Optional[str] = ..., source_ref: _Optional[str] = ..., kind: _Optional[str] = ..., consumer_refs: _Optional[_Iterable[str]] = ..., version: _Optional[str] = ..., provider: _Optional[str] = ..., applies_when: _Optional[_Union[CredentialApplicability, _Mapping]] = ..., requirement_group: _Optional[str] = ..., companion_settings: _Optional[_Iterable[str]] = ..., companion_credentials: _Optional[_Iterable[str]] = ..., acquisition_ref: _Optional[str] = ..., verification_ref: _Optional[str] = ..., recovery_ref: _Optional[str] = ..., help_ref: _Optional[str] = ..., evidence_policy: _Optional[str] = ..., provider_version: _Optional[str] = ..., migration_diagnostics: _Optional[_Iterable[_Union[CredentialMigrationDiagnostic, _Mapping]]] = ..., tiers: _Optional[_Iterable[str]] = ...) -> None: ...

class ListCredentialsResponse(_message.Message):
    __slots__ = ("credentials", "count")
    CREDENTIALS_FIELD_NUMBER: _ClassVar[int]
    COUNT_FIELD_NUMBER: _ClassVar[int]
    credentials: _containers.RepeatedCompositeFieldContainer[Credential]
    count: int
    def __init__(self, credentials: _Optional[_Iterable[_Union[Credential, _Mapping]]] = ..., count: _Optional[int] = ...) -> None: ...

class ProvisionCredentialResponse(_message.Message):
    __slots__ = ("status", "logical_id", "field", "version")
    STATUS_FIELD_NUMBER: _ClassVar[int]
    LOGICAL_ID_FIELD_NUMBER: _ClassVar[int]
    FIELD_FIELD_NUMBER: _ClassVar[int]
    VERSION_FIELD_NUMBER: _ClassVar[int]
    status: str
    logical_id: str
    field: str
    version: str
    def __init__(self, status: _Optional[str] = ..., logical_id: _Optional[str] = ..., field: _Optional[str] = ..., version: _Optional[str] = ...) -> None: ...

class ProviderDiagnosis(_message.Message):
    __slots__ = ("platform", "adapter", "backend", "condition", "available", "writable", "explanation", "fix", "native_storage_caveat", "write_condition", "write_explanation", "write_fix", "session_repair")
    PLATFORM_FIELD_NUMBER: _ClassVar[int]
    ADAPTER_FIELD_NUMBER: _ClassVar[int]
    BACKEND_FIELD_NUMBER: _ClassVar[int]
    CONDITION_FIELD_NUMBER: _ClassVar[int]
    AVAILABLE_FIELD_NUMBER: _ClassVar[int]
    WRITABLE_FIELD_NUMBER: _ClassVar[int]
    EXPLANATION_FIELD_NUMBER: _ClassVar[int]
    FIX_FIELD_NUMBER: _ClassVar[int]
    NATIVE_STORAGE_CAVEAT_FIELD_NUMBER: _ClassVar[int]
    WRITE_CONDITION_FIELD_NUMBER: _ClassVar[int]
    WRITE_EXPLANATION_FIELD_NUMBER: _ClassVar[int]
    WRITE_FIX_FIELD_NUMBER: _ClassVar[int]
    SESSION_REPAIR_FIELD_NUMBER: _ClassVar[int]
    platform: str
    adapter: str
    backend: str
    condition: str
    available: bool
    writable: bool
    explanation: str
    fix: str
    native_storage_caveat: str
    write_condition: str
    write_explanation: str
    write_fix: str
    session_repair: str
    def __init__(self, platform: _Optional[str] = ..., adapter: _Optional[str] = ..., backend: _Optional[str] = ..., condition: _Optional[str] = ..., available: _Optional[bool] = ..., writable: _Optional[bool] = ..., explanation: _Optional[str] = ..., fix: _Optional[str] = ..., native_storage_caveat: _Optional[str] = ..., write_condition: _Optional[str] = ..., write_explanation: _Optional[str] = ..., write_fix: _Optional[str] = ..., session_repair: _Optional[str] = ...) -> None: ...

class RecoveryStatus(_message.Message):
    __slots__ = ("receipt_exists", "exported_at", "entry_count", "uncovered", "required_absent", "basis", "managed_instances_included", "status", "age_seconds", "freshness_reason")
    RECEIPT_EXISTS_FIELD_NUMBER: _ClassVar[int]
    EXPORTED_AT_FIELD_NUMBER: _ClassVar[int]
    ENTRY_COUNT_FIELD_NUMBER: _ClassVar[int]
    UNCOVERED_FIELD_NUMBER: _ClassVar[int]
    REQUIRED_ABSENT_FIELD_NUMBER: _ClassVar[int]
    BASIS_FIELD_NUMBER: _ClassVar[int]
    MANAGED_INSTANCES_INCLUDED_FIELD_NUMBER: _ClassVar[int]
    STATUS_FIELD_NUMBER: _ClassVar[int]
    AGE_SECONDS_FIELD_NUMBER: _ClassVar[int]
    FRESHNESS_REASON_FIELD_NUMBER: _ClassVar[int]
    receipt_exists: bool
    exported_at: str
    entry_count: int
    uncovered: _containers.RepeatedScalarFieldContainer[str]
    required_absent: _containers.RepeatedScalarFieldContainer[str]
    basis: str
    managed_instances_included: bool
    status: str
    age_seconds: int
    freshness_reason: str
    def __init__(self, receipt_exists: _Optional[bool] = ..., exported_at: _Optional[str] = ..., entry_count: _Optional[int] = ..., uncovered: _Optional[_Iterable[str]] = ..., required_absent: _Optional[_Iterable[str]] = ..., basis: _Optional[str] = ..., managed_instances_included: _Optional[bool] = ..., status: _Optional[str] = ..., age_seconds: _Optional[int] = ..., freshness_reason: _Optional[str] = ...) -> None: ...

class DiagnoseCredentialsResponse(_message.Message):
    __slots__ = ("provider", "credentials", "credential_count", "declaration_site_count", "inventory_basis", "managed_instances_included", "recovery")
    PROVIDER_FIELD_NUMBER: _ClassVar[int]
    CREDENTIALS_FIELD_NUMBER: _ClassVar[int]
    CREDENTIAL_COUNT_FIELD_NUMBER: _ClassVar[int]
    DECLARATION_SITE_COUNT_FIELD_NUMBER: _ClassVar[int]
    INVENTORY_BASIS_FIELD_NUMBER: _ClassVar[int]
    MANAGED_INSTANCES_INCLUDED_FIELD_NUMBER: _ClassVar[int]
    RECOVERY_FIELD_NUMBER: _ClassVar[int]
    provider: ProviderDiagnosis
    credentials: _containers.RepeatedCompositeFieldContainer[Credential]
    credential_count: int
    declaration_site_count: int
    inventory_basis: str
    managed_instances_included: bool
    recovery: RecoveryStatus
    def __init__(self, provider: _Optional[_Union[ProviderDiagnosis, _Mapping]] = ..., credentials: _Optional[_Iterable[_Union[Credential, _Mapping]]] = ..., credential_count: _Optional[int] = ..., declaration_site_count: _Optional[int] = ..., inventory_basis: _Optional[str] = ..., managed_instances_included: _Optional[bool] = ..., recovery: _Optional[_Union[RecoveryStatus, _Mapping]] = ...) -> None: ...
