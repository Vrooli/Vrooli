import datetime

from buf.validate import validate_pb2 as _validate_pb2
from google.protobuf import struct_pb2 as _struct_pb2
from google.protobuf import timestamp_pb2 as _timestamp_pb2
from vrooli_onboarding.v1.shared import shared_pb2 as _shared_pb2
from google.protobuf.internal import containers as _containers
from google.protobuf.internal import enum_type_wrapper as _enum_type_wrapper
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Iterable as _Iterable, Mapping as _Mapping
from typing import ClassVar as _ClassVar, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class ReadinessState(int, metaclass=_enum_type_wrapper.EnumTypeWrapper):
    __slots__ = ()
    READINESS_STATE_UNSPECIFIED: _ClassVar[ReadinessState]
    READINESS_STATE_READY: _ClassVar[ReadinessState]
    READINESS_STATE_MISSING: _ClassVar[ReadinessState]
    READINESS_STATE_DEGRADED: _ClassVar[ReadinessState]
    READINESS_STATE_UNSUPPORTED: _ClassVar[ReadinessState]
    READINESS_STATE_DEFERRED: _ClassVar[ReadinessState]
    READINESS_STATE_NOT_APPLICABLE: _ClassVar[ReadinessState]
READINESS_STATE_UNSPECIFIED: ReadinessState
READINESS_STATE_READY: ReadinessState
READINESS_STATE_MISSING: ReadinessState
READINESS_STATE_DEGRADED: ReadinessState
READINESS_STATE_UNSUPPORTED: ReadinessState
READINESS_STATE_DEFERRED: ReadinessState
READINESS_STATE_NOT_APPLICABLE: ReadinessState

class GetReadinessRequest(_message.Message):
    __slots__ = ("target",)
    TARGET_FIELD_NUMBER: _ClassVar[int]
    target: str
    def __init__(self, target: _Optional[str] = ...) -> None: ...

class AcknowledgeDegradedReadinessRequest(_message.Message):
    __slots__ = ("target", "readiness_digest")
    TARGET_FIELD_NUMBER: _ClassVar[int]
    READINESS_DIGEST_FIELD_NUMBER: _ClassVar[int]
    target: str
    readiness_digest: str
    def __init__(self, target: _Optional[str] = ..., readiness_digest: _Optional[str] = ...) -> None: ...

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
    __slots__ = ("resource", "logical_id", "field", "label", "description", "obtain_url", "placeholder", "provisioning", "derived_from", "required", "status", "legacy_status", "detail", "provenance", "owner", "source_ref", "kind", "consumer_refs", "version", "provider", "applies_when", "requirement_group", "companion_settings", "companion_credentials", "acquisition_ref", "verification_ref", "recovery_ref", "help_ref", "evidence_policy", "provider_version", "migration_diagnostics", "evidence_status", "evidence_detail", "evidence_next_action", "evidence_credential_version", "provider_state", "provider_detail", "tiers")
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
    LEGACY_STATUS_FIELD_NUMBER: _ClassVar[int]
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
    EVIDENCE_STATUS_FIELD_NUMBER: _ClassVar[int]
    EVIDENCE_DETAIL_FIELD_NUMBER: _ClassVar[int]
    EVIDENCE_NEXT_ACTION_FIELD_NUMBER: _ClassVar[int]
    EVIDENCE_CREDENTIAL_VERSION_FIELD_NUMBER: _ClassVar[int]
    PROVIDER_STATE_FIELD_NUMBER: _ClassVar[int]
    PROVIDER_DETAIL_FIELD_NUMBER: _ClassVar[int]
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
    status: ReadinessState
    legacy_status: str
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
    evidence_status: str
    evidence_detail: str
    evidence_next_action: str
    evidence_credential_version: str
    provider_state: str
    provider_detail: str
    tiers: _containers.RepeatedScalarFieldContainer[str]
    def __init__(self, resource: _Optional[str] = ..., logical_id: _Optional[str] = ..., field: _Optional[str] = ..., label: _Optional[str] = ..., description: _Optional[str] = ..., obtain_url: _Optional[str] = ..., placeholder: _Optional[str] = ..., provisioning: _Optional[str] = ..., derived_from: _Optional[str] = ..., required: _Optional[bool] = ..., status: _Optional[_Union[ReadinessState, str]] = ..., legacy_status: _Optional[str] = ..., detail: _Optional[str] = ..., provenance: _Optional[_Iterable[_Union[CredentialProvenance, _Mapping]]] = ..., owner: _Optional[str] = ..., source_ref: _Optional[str] = ..., kind: _Optional[str] = ..., consumer_refs: _Optional[_Iterable[str]] = ..., version: _Optional[str] = ..., provider: _Optional[str] = ..., applies_when: _Optional[_Union[CredentialApplicability, _Mapping]] = ..., requirement_group: _Optional[str] = ..., companion_settings: _Optional[_Iterable[str]] = ..., companion_credentials: _Optional[_Iterable[str]] = ..., acquisition_ref: _Optional[str] = ..., verification_ref: _Optional[str] = ..., recovery_ref: _Optional[str] = ..., help_ref: _Optional[str] = ..., evidence_policy: _Optional[str] = ..., provider_version: _Optional[str] = ..., migration_diagnostics: _Optional[_Iterable[_Union[CredentialMigrationDiagnostic, _Mapping]]] = ..., evidence_status: _Optional[str] = ..., evidence_detail: _Optional[str] = ..., evidence_next_action: _Optional[str] = ..., evidence_credential_version: _Optional[str] = ..., provider_state: _Optional[str] = ..., provider_detail: _Optional[str] = ..., tiers: _Optional[_Iterable[str]] = ...) -> None: ...

class ReadinessItem(_message.Message):
    __slots__ = ("name", "category", "status", "legacy_status", "detail", "remediation", "required")
    NAME_FIELD_NUMBER: _ClassVar[int]
    CATEGORY_FIELD_NUMBER: _ClassVar[int]
    STATUS_FIELD_NUMBER: _ClassVar[int]
    LEGACY_STATUS_FIELD_NUMBER: _ClassVar[int]
    DETAIL_FIELD_NUMBER: _ClassVar[int]
    REMEDIATION_FIELD_NUMBER: _ClassVar[int]
    REQUIRED_FIELD_NUMBER: _ClassVar[int]
    name: str
    category: str
    status: ReadinessState
    legacy_status: str
    detail: str
    remediation: str
    required: bool
    def __init__(self, name: _Optional[str] = ..., category: _Optional[str] = ..., status: _Optional[_Union[ReadinessState, str]] = ..., legacy_status: _Optional[str] = ..., detail: _Optional[str] = ..., remediation: _Optional[str] = ..., required: _Optional[bool] = ...) -> None: ...

class HostReadiness(_message.Message):
    __slots__ = ("item", "kind", "required")
    ITEM_FIELD_NUMBER: _ClassVar[int]
    KIND_FIELD_NUMBER: _ClassVar[int]
    REQUIRED_FIELD_NUMBER: _ClassVar[int]
    item: ReadinessItem
    kind: str
    required: bool
    def __init__(self, item: _Optional[_Union[ReadinessItem, _Mapping]] = ..., kind: _Optional[str] = ..., required: _Optional[bool] = ...) -> None: ...

class RecoveryGap(_message.Message):
    __slots__ = ("address", "description")
    ADDRESS_FIELD_NUMBER: _ClassVar[int]
    DESCRIPTION_FIELD_NUMBER: _ClassVar[int]
    address: str
    description: str
    def __init__(self, address: _Optional[str] = ..., description: _Optional[str] = ...) -> None: ...

class Recovery(_message.Message):
    __slots__ = ("receipt_exists", "exported_at", "entry_count", "uncovered", "required_absent", "required_absent_details", "root_copy", "root_copy_issues", "status", "age_seconds", "freshness_reason")
    RECEIPT_EXISTS_FIELD_NUMBER: _ClassVar[int]
    EXPORTED_AT_FIELD_NUMBER: _ClassVar[int]
    ENTRY_COUNT_FIELD_NUMBER: _ClassVar[int]
    UNCOVERED_FIELD_NUMBER: _ClassVar[int]
    REQUIRED_ABSENT_FIELD_NUMBER: _ClassVar[int]
    REQUIRED_ABSENT_DETAILS_FIELD_NUMBER: _ClassVar[int]
    ROOT_COPY_FIELD_NUMBER: _ClassVar[int]
    ROOT_COPY_ISSUES_FIELD_NUMBER: _ClassVar[int]
    STATUS_FIELD_NUMBER: _ClassVar[int]
    AGE_SECONDS_FIELD_NUMBER: _ClassVar[int]
    FRESHNESS_REASON_FIELD_NUMBER: _ClassVar[int]
    receipt_exists: bool
    exported_at: str
    entry_count: int
    uncovered: _containers.RepeatedScalarFieldContainer[str]
    required_absent: _containers.RepeatedScalarFieldContainer[str]
    required_absent_details: _containers.RepeatedCompositeFieldContainer[RecoveryGap]
    root_copy: _struct_pb2.Struct
    root_copy_issues: _containers.RepeatedScalarFieldContainer[str]
    status: str
    age_seconds: int
    freshness_reason: str
    def __init__(self, receipt_exists: _Optional[bool] = ..., exported_at: _Optional[str] = ..., entry_count: _Optional[int] = ..., uncovered: _Optional[_Iterable[str]] = ..., required_absent: _Optional[_Iterable[str]] = ..., required_absent_details: _Optional[_Iterable[_Union[RecoveryGap, _Mapping]]] = ..., root_copy: _Optional[_Union[_struct_pb2.Struct, _Mapping]] = ..., root_copy_issues: _Optional[_Iterable[str]] = ..., status: _Optional[str] = ..., age_seconds: _Optional[int] = ..., freshness_reason: _Optional[str] = ...) -> None: ...

class GetReadinessResponse(_message.Message):
    __slots__ = ("target", "configuration_revision", "expires_at", "status", "scenarios", "resources", "credentials", "hosts", "integrations", "checked_at", "credential_diagnosis", "recovery", "blockers", "degraded", "degraded_digest", "degraded_acknowledged", "managed_key_configured", "trust_anchor_match")
    TARGET_FIELD_NUMBER: _ClassVar[int]
    CONFIGURATION_REVISION_FIELD_NUMBER: _ClassVar[int]
    EXPIRES_AT_FIELD_NUMBER: _ClassVar[int]
    STATUS_FIELD_NUMBER: _ClassVar[int]
    SCENARIOS_FIELD_NUMBER: _ClassVar[int]
    RESOURCES_FIELD_NUMBER: _ClassVar[int]
    CREDENTIALS_FIELD_NUMBER: _ClassVar[int]
    HOSTS_FIELD_NUMBER: _ClassVar[int]
    INTEGRATIONS_FIELD_NUMBER: _ClassVar[int]
    CHECKED_AT_FIELD_NUMBER: _ClassVar[int]
    CREDENTIAL_DIAGNOSIS_FIELD_NUMBER: _ClassVar[int]
    RECOVERY_FIELD_NUMBER: _ClassVar[int]
    BLOCKERS_FIELD_NUMBER: _ClassVar[int]
    DEGRADED_FIELD_NUMBER: _ClassVar[int]
    DEGRADED_DIGEST_FIELD_NUMBER: _ClassVar[int]
    DEGRADED_ACKNOWLEDGED_FIELD_NUMBER: _ClassVar[int]
    MANAGED_KEY_CONFIGURED_FIELD_NUMBER: _ClassVar[int]
    TRUST_ANCHOR_MATCH_FIELD_NUMBER: _ClassVar[int]
    target: str
    configuration_revision: str
    expires_at: _timestamp_pb2.Timestamp
    status: ReadinessState
    scenarios: _containers.RepeatedScalarFieldContainer[str]
    resources: _containers.RepeatedScalarFieldContainer[str]
    credentials: _containers.RepeatedCompositeFieldContainer[Credential]
    hosts: _containers.RepeatedCompositeFieldContainer[HostReadiness]
    integrations: _containers.RepeatedCompositeFieldContainer[ReadinessItem]
    checked_at: _timestamp_pb2.Timestamp
    credential_diagnosis: _struct_pb2.Struct
    recovery: Recovery
    blockers: _containers.RepeatedCompositeFieldContainer[_shared_pb2.CompletionBlocker]
    degraded: _containers.RepeatedCompositeFieldContainer[_shared_pb2.CompletionBlocker]
    degraded_digest: str
    degraded_acknowledged: bool
    managed_key_configured: bool
    trust_anchor_match: bool
    def __init__(self, target: _Optional[str] = ..., configuration_revision: _Optional[str] = ..., expires_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., status: _Optional[_Union[ReadinessState, str]] = ..., scenarios: _Optional[_Iterable[str]] = ..., resources: _Optional[_Iterable[str]] = ..., credentials: _Optional[_Iterable[_Union[Credential, _Mapping]]] = ..., hosts: _Optional[_Iterable[_Union[HostReadiness, _Mapping]]] = ..., integrations: _Optional[_Iterable[_Union[ReadinessItem, _Mapping]]] = ..., checked_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., credential_diagnosis: _Optional[_Union[_struct_pb2.Struct, _Mapping]] = ..., recovery: _Optional[_Union[Recovery, _Mapping]] = ..., blockers: _Optional[_Iterable[_Union[_shared_pb2.CompletionBlocker, _Mapping]]] = ..., degraded: _Optional[_Iterable[_Union[_shared_pb2.CompletionBlocker, _Mapping]]] = ..., degraded_digest: _Optional[str] = ..., degraded_acknowledged: _Optional[bool] = ..., managed_key_configured: _Optional[bool] = ..., trust_anchor_match: _Optional[bool] = ...) -> None: ...

class AcknowledgeDegradedReadinessResponse(_message.Message):
    __slots__ = ("status", "readiness_digest", "degraded")
    STATUS_FIELD_NUMBER: _ClassVar[int]
    READINESS_DIGEST_FIELD_NUMBER: _ClassVar[int]
    DEGRADED_FIELD_NUMBER: _ClassVar[int]
    status: str
    readiness_digest: str
    degraded: _containers.RepeatedCompositeFieldContainer[_shared_pb2.CompletionBlocker]
    def __init__(self, status: _Optional[str] = ..., readiness_digest: _Optional[str] = ..., degraded: _Optional[_Iterable[_Union[_shared_pb2.CompletionBlocker, _Mapping]]] = ...) -> None: ...
