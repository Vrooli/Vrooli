from google.protobuf.internal import containers as _containers
from google.protobuf.internal import enum_type_wrapper as _enum_type_wrapper
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Iterable as _Iterable, Mapping as _Mapping
from typing import ClassVar as _ClassVar, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class OwnershipState(int, metaclass=_enum_type_wrapper.EnumTypeWrapper):
    __slots__ = ()
    OWNERSHIP_STATE_UNSPECIFIED: _ClassVar[OwnershipState]
    OWNERSHIP_STATE_MANAGED: _ClassVar[OwnershipState]
    OWNERSHIP_STATE_MISSING: _ClassVar[OwnershipState]
    OWNERSHIP_STATE_EXTERNAL_OK: _ClassVar[OwnershipState]
    OWNERSHIP_STATE_ORPHANED: _ClassVar[OwnershipState]
    OWNERSHIP_STATE_IGNORED: _ClassVar[OwnershipState]
    OWNERSHIP_STATE_UNMANAGED: _ClassVar[OwnershipState]

class IngressSource(int, metaclass=_enum_type_wrapper.EnumTypeWrapper):
    __slots__ = ()
    INGRESS_SOURCE_UNSPECIFIED: _ClassVar[IngressSource]
    INGRESS_SOURCE_SCENARIO: _ClassVar[IngressSource]
    INGRESS_SOURCE_EXTERNAL: _ClassVar[IngressSource]

class Mode(int, metaclass=_enum_type_wrapper.EnumTypeWrapper):
    __slots__ = ()
    MODE_UNSPECIFIED: _ClassVar[Mode]
    MODE_REMOTE: _ClassVar[Mode]
    MODE_LOCAL: _ClassVar[Mode]

class CheckState(int, metaclass=_enum_type_wrapper.EnumTypeWrapper):
    __slots__ = ()
    CHECK_STATE_UNSPECIFIED: _ClassVar[CheckState]
    CHECK_STATE_OK: _ClassVar[CheckState]
    CHECK_STATE_MISSING: _ClassVar[CheckState]
    CHECK_STATE_INVALID: _ClassVar[CheckState]
    CHECK_STATE_INSUFFICIENT_SCOPE: _ClassVar[CheckState]
OWNERSHIP_STATE_UNSPECIFIED: OwnershipState
OWNERSHIP_STATE_MANAGED: OwnershipState
OWNERSHIP_STATE_MISSING: OwnershipState
OWNERSHIP_STATE_EXTERNAL_OK: OwnershipState
OWNERSHIP_STATE_ORPHANED: OwnershipState
OWNERSHIP_STATE_IGNORED: OwnershipState
OWNERSHIP_STATE_UNMANAGED: OwnershipState
INGRESS_SOURCE_UNSPECIFIED: IngressSource
INGRESS_SOURCE_SCENARIO: IngressSource
INGRESS_SOURCE_EXTERNAL: IngressSource
MODE_UNSPECIFIED: Mode
MODE_REMOTE: Mode
MODE_LOCAL: Mode
CHECK_STATE_UNSPECIFIED: CheckState
CHECK_STATE_OK: CheckState
CHECK_STATE_MISSING: CheckState
CHECK_STATE_INVALID: CheckState
CHECK_STATE_INSUFFICIENT_SCOPE: CheckState

class IngressEntry(_message.Message):
    __slots__ = ("hostname", "service_target", "state", "source", "scenario", "lease_id", "note")
    HOSTNAME_FIELD_NUMBER: _ClassVar[int]
    SERVICE_TARGET_FIELD_NUMBER: _ClassVar[int]
    STATE_FIELD_NUMBER: _ClassVar[int]
    SOURCE_FIELD_NUMBER: _ClassVar[int]
    SCENARIO_FIELD_NUMBER: _ClassVar[int]
    LEASE_ID_FIELD_NUMBER: _ClassVar[int]
    NOTE_FIELD_NUMBER: _ClassVar[int]
    hostname: str
    service_target: str
    state: OwnershipState
    source: IngressSource
    scenario: str
    lease_id: str
    note: str
    def __init__(self, hostname: _Optional[str] = ..., service_target: _Optional[str] = ..., state: _Optional[_Union[OwnershipState, str]] = ..., source: _Optional[_Union[IngressSource, str]] = ..., scenario: _Optional[str] = ..., lease_id: _Optional[str] = ..., note: _Optional[str] = ...) -> None: ...

class DriftCounts(_message.Message):
    __slots__ = ("managed", "missing", "external_ok", "orphaned", "ignored", "unmanaged")
    MANAGED_FIELD_NUMBER: _ClassVar[int]
    MISSING_FIELD_NUMBER: _ClassVar[int]
    EXTERNAL_OK_FIELD_NUMBER: _ClassVar[int]
    ORPHANED_FIELD_NUMBER: _ClassVar[int]
    IGNORED_FIELD_NUMBER: _ClassVar[int]
    UNMANAGED_FIELD_NUMBER: _ClassVar[int]
    managed: int
    missing: int
    external_ok: int
    orphaned: int
    ignored: int
    unmanaged: int
    def __init__(self, managed: _Optional[int] = ..., missing: _Optional[int] = ..., external_ok: _Optional[int] = ..., orphaned: _Optional[int] = ..., ignored: _Optional[int] = ..., unmanaged: _Optional[int] = ...) -> None: ...

class TunnelConfig(_message.Message):
    __slots__ = ("mode", "tunnel_id", "account_id", "cred_ref", "prom_endpoint", "public_exposure_enabled")
    MODE_FIELD_NUMBER: _ClassVar[int]
    TUNNEL_ID_FIELD_NUMBER: _ClassVar[int]
    ACCOUNT_ID_FIELD_NUMBER: _ClassVar[int]
    CRED_REF_FIELD_NUMBER: _ClassVar[int]
    PROM_ENDPOINT_FIELD_NUMBER: _ClassVar[int]
    PUBLIC_EXPOSURE_ENABLED_FIELD_NUMBER: _ClassVar[int]
    mode: Mode
    tunnel_id: str
    account_id: str
    cred_ref: str
    prom_endpoint: str
    public_exposure_enabled: bool
    def __init__(self, mode: _Optional[_Union[Mode, str]] = ..., tunnel_id: _Optional[str] = ..., account_id: _Optional[str] = ..., cred_ref: _Optional[str] = ..., prom_endpoint: _Optional[str] = ..., public_exposure_enabled: _Optional[bool] = ...) -> None: ...

class ConfigReadiness(_message.Message):
    __slots__ = ("desired_mode", "remote_available", "missing_fields", "credential_source", "credential_ref", "local_config_path", "sync_ready", "mode_reason", "credential_fields")
    DESIRED_MODE_FIELD_NUMBER: _ClassVar[int]
    REMOTE_AVAILABLE_FIELD_NUMBER: _ClassVar[int]
    MISSING_FIELDS_FIELD_NUMBER: _ClassVar[int]
    CREDENTIAL_SOURCE_FIELD_NUMBER: _ClassVar[int]
    CREDENTIAL_REF_FIELD_NUMBER: _ClassVar[int]
    LOCAL_CONFIG_PATH_FIELD_NUMBER: _ClassVar[int]
    SYNC_READY_FIELD_NUMBER: _ClassVar[int]
    MODE_REASON_FIELD_NUMBER: _ClassVar[int]
    CREDENTIAL_FIELDS_FIELD_NUMBER: _ClassVar[int]
    desired_mode: Mode
    remote_available: bool
    missing_fields: _containers.RepeatedScalarFieldContainer[str]
    credential_source: str
    credential_ref: str
    local_config_path: str
    sync_ready: bool
    mode_reason: str
    credential_fields: _containers.RepeatedCompositeFieldContainer[CredentialFieldStatus]
    def __init__(self, desired_mode: _Optional[_Union[Mode, str]] = ..., remote_available: _Optional[bool] = ..., missing_fields: _Optional[_Iterable[str]] = ..., credential_source: _Optional[str] = ..., credential_ref: _Optional[str] = ..., local_config_path: _Optional[str] = ..., sync_ready: _Optional[bool] = ..., mode_reason: _Optional[str] = ..., credential_fields: _Optional[_Iterable[_Union[CredentialFieldStatus, _Mapping]]] = ...) -> None: ...

class CredentialFieldStatus(_message.Message):
    __slots__ = ("name", "present", "source", "ref", "writable")
    NAME_FIELD_NUMBER: _ClassVar[int]
    PRESENT_FIELD_NUMBER: _ClassVar[int]
    SOURCE_FIELD_NUMBER: _ClassVar[int]
    REF_FIELD_NUMBER: _ClassVar[int]
    WRITABLE_FIELD_NUMBER: _ClassVar[int]
    name: str
    present: bool
    source: str
    ref: str
    writable: bool
    def __init__(self, name: _Optional[str] = ..., present: _Optional[bool] = ..., source: _Optional[str] = ..., ref: _Optional[str] = ..., writable: _Optional[bool] = ...) -> None: ...

class CredentialStatus(_message.Message):
    __slots__ = ("fields", "missing_fields", "source", "ref", "ready")
    FIELDS_FIELD_NUMBER: _ClassVar[int]
    MISSING_FIELDS_FIELD_NUMBER: _ClassVar[int]
    SOURCE_FIELD_NUMBER: _ClassVar[int]
    REF_FIELD_NUMBER: _ClassVar[int]
    READY_FIELD_NUMBER: _ClassVar[int]
    fields: _containers.RepeatedCompositeFieldContainer[CredentialFieldStatus]
    missing_fields: _containers.RepeatedScalarFieldContainer[str]
    source: str
    ref: str
    ready: bool
    def __init__(self, fields: _Optional[_Iterable[_Union[CredentialFieldStatus, _Mapping]]] = ..., missing_fields: _Optional[_Iterable[str]] = ..., source: _Optional[str] = ..., ref: _Optional[str] = ..., ready: _Optional[bool] = ...) -> None: ...

class GetConfigRequest(_message.Message):
    __slots__ = ()
    def __init__(self) -> None: ...

class GetConfigResponse(_message.Message):
    __slots__ = ("config", "readiness")
    CONFIG_FIELD_NUMBER: _ClassVar[int]
    READINESS_FIELD_NUMBER: _ClassVar[int]
    config: TunnelConfig
    readiness: ConfigReadiness
    def __init__(self, config: _Optional[_Union[TunnelConfig, _Mapping]] = ..., readiness: _Optional[_Union[ConfigReadiness, _Mapping]] = ...) -> None: ...

class GetCredentialStatusRequest(_message.Message):
    __slots__ = ()
    def __init__(self) -> None: ...

class GetCredentialStatusResponse(_message.Message):
    __slots__ = ("status",)
    STATUS_FIELD_NUMBER: _ClassVar[int]
    status: CredentialStatus
    def __init__(self, status: _Optional[_Union[CredentialStatus, _Mapping]] = ...) -> None: ...

class CredentialCheck(_message.Message):
    __slots__ = ("name", "state", "detail", "remediation")
    NAME_FIELD_NUMBER: _ClassVar[int]
    STATE_FIELD_NUMBER: _ClassVar[int]
    DETAIL_FIELD_NUMBER: _ClassVar[int]
    REMEDIATION_FIELD_NUMBER: _ClassVar[int]
    name: str
    state: CheckState
    detail: str
    remediation: str
    def __init__(self, name: _Optional[str] = ..., state: _Optional[_Union[CheckState, str]] = ..., detail: _Optional[str] = ..., remediation: _Optional[str] = ...) -> None: ...

class VerifyCredentialsRequest(_message.Message):
    __slots__ = ()
    def __init__(self) -> None: ...

class VerifyCredentialsResponse(_message.Message):
    __slots__ = ("checks", "ready")
    CHECKS_FIELD_NUMBER: _ClassVar[int]
    READY_FIELD_NUMBER: _ClassVar[int]
    checks: _containers.RepeatedCompositeFieldContainer[CredentialCheck]
    ready: bool
    def __init__(self, checks: _Optional[_Iterable[_Union[CredentialCheck, _Mapping]]] = ..., ready: _Optional[bool] = ...) -> None: ...

class SetCloudflareCredentialsRequest(_message.Message):
    __slots__ = ("account_id", "tunnel_id", "api_token")
    ACCOUNT_ID_FIELD_NUMBER: _ClassVar[int]
    TUNNEL_ID_FIELD_NUMBER: _ClassVar[int]
    API_TOKEN_FIELD_NUMBER: _ClassVar[int]
    account_id: str
    tunnel_id: str
    api_token: str
    def __init__(self, account_id: _Optional[str] = ..., tunnel_id: _Optional[str] = ..., api_token: _Optional[str] = ...) -> None: ...

class SetCloudflareCredentialsResponse(_message.Message):
    __slots__ = ("status",)
    STATUS_FIELD_NUMBER: _ClassVar[int]
    status: CredentialStatus
    def __init__(self, status: _Optional[_Union[CredentialStatus, _Mapping]] = ...) -> None: ...

class ClearCloudflareCredentialsRequest(_message.Message):
    __slots__ = ("fields",)
    FIELDS_FIELD_NUMBER: _ClassVar[int]
    fields: _containers.RepeatedScalarFieldContainer[str]
    def __init__(self, fields: _Optional[_Iterable[str]] = ...) -> None: ...

class ClearCloudflareCredentialsResponse(_message.Message):
    __slots__ = ("status",)
    STATUS_FIELD_NUMBER: _ClassVar[int]
    status: CredentialStatus
    def __init__(self, status: _Optional[_Union[CredentialStatus, _Mapping]] = ...) -> None: ...

class SyncRequest(_message.Message):
    __slots__ = ("dry_run", "prune")
    DRY_RUN_FIELD_NUMBER: _ClassVar[int]
    PRUNE_FIELD_NUMBER: _ClassVar[int]
    dry_run: bool
    prune: bool
    def __init__(self, dry_run: _Optional[bool] = ..., prune: _Optional[bool] = ...) -> None: ...

class SyncResponse(_message.Message):
    __slots__ = ("mode", "added", "removed", "no_changes", "setup_required", "missing_fields", "message", "drift_unmanaged", "orphaned", "pruned")
    MODE_FIELD_NUMBER: _ClassVar[int]
    ADDED_FIELD_NUMBER: _ClassVar[int]
    REMOVED_FIELD_NUMBER: _ClassVar[int]
    NO_CHANGES_FIELD_NUMBER: _ClassVar[int]
    SETUP_REQUIRED_FIELD_NUMBER: _ClassVar[int]
    MISSING_FIELDS_FIELD_NUMBER: _ClassVar[int]
    MESSAGE_FIELD_NUMBER: _ClassVar[int]
    DRIFT_UNMANAGED_FIELD_NUMBER: _ClassVar[int]
    ORPHANED_FIELD_NUMBER: _ClassVar[int]
    PRUNED_FIELD_NUMBER: _ClassVar[int]
    mode: Mode
    added: _containers.RepeatedScalarFieldContainer[str]
    removed: _containers.RepeatedScalarFieldContainer[str]
    no_changes: bool
    setup_required: bool
    missing_fields: _containers.RepeatedScalarFieldContainer[str]
    message: str
    drift_unmanaged: _containers.RepeatedScalarFieldContainer[str]
    orphaned: _containers.RepeatedScalarFieldContainer[str]
    pruned: _containers.RepeatedScalarFieldContainer[str]
    def __init__(self, mode: _Optional[_Union[Mode, str]] = ..., added: _Optional[_Iterable[str]] = ..., removed: _Optional[_Iterable[str]] = ..., no_changes: _Optional[bool] = ..., setup_required: _Optional[bool] = ..., missing_fields: _Optional[_Iterable[str]] = ..., message: _Optional[str] = ..., drift_unmanaged: _Optional[_Iterable[str]] = ..., orphaned: _Optional[_Iterable[str]] = ..., pruned: _Optional[_Iterable[str]] = ...) -> None: ...

class SwitchModeRequest(_message.Message):
    __slots__ = ("target_mode",)
    TARGET_MODE_FIELD_NUMBER: _ClassVar[int]
    target_mode: Mode
    def __init__(self, target_mode: _Optional[_Union[Mode, str]] = ...) -> None: ...

class SwitchModeResponse(_message.Message):
    __slots__ = ("previous_mode", "current_mode")
    PREVIOUS_MODE_FIELD_NUMBER: _ClassVar[int]
    CURRENT_MODE_FIELD_NUMBER: _ClassVar[int]
    previous_mode: Mode
    current_mode: Mode
    def __init__(self, previous_mode: _Optional[_Union[Mode, str]] = ..., current_mode: _Optional[_Union[Mode, str]] = ...) -> None: ...

class GetDriftRequest(_message.Message):
    __slots__ = ()
    def __init__(self) -> None: ...

class GetDriftResponse(_message.Message):
    __slots__ = ("mode", "entries", "counts")
    MODE_FIELD_NUMBER: _ClassVar[int]
    ENTRIES_FIELD_NUMBER: _ClassVar[int]
    COUNTS_FIELD_NUMBER: _ClassVar[int]
    mode: Mode
    entries: _containers.RepeatedCompositeFieldContainer[IngressEntry]
    counts: DriftCounts
    def __init__(self, mode: _Optional[_Union[Mode, str]] = ..., entries: _Optional[_Iterable[_Union[IngressEntry, _Mapping]]] = ..., counts: _Optional[_Union[DriftCounts, _Mapping]] = ...) -> None: ...

class AdoptIngressRequest(_message.Message):
    __slots__ = ("hostname", "scenario", "target")
    HOSTNAME_FIELD_NUMBER: _ClassVar[int]
    SCENARIO_FIELD_NUMBER: _ClassVar[int]
    TARGET_FIELD_NUMBER: _ClassVar[int]
    hostname: str
    scenario: str
    target: str
    def __init__(self, hostname: _Optional[str] = ..., scenario: _Optional[str] = ..., target: _Optional[str] = ...) -> None: ...

class AdoptIngressResponse(_message.Message):
    __slots__ = ("entry",)
    ENTRY_FIELD_NUMBER: _ClassVar[int]
    entry: IngressEntry
    def __init__(self, entry: _Optional[_Union[IngressEntry, _Mapping]] = ...) -> None: ...

class IgnoreIngressRequest(_message.Message):
    __slots__ = ("hostname", "note")
    HOSTNAME_FIELD_NUMBER: _ClassVar[int]
    NOTE_FIELD_NUMBER: _ClassVar[int]
    hostname: str
    note: str
    def __init__(self, hostname: _Optional[str] = ..., note: _Optional[str] = ...) -> None: ...

class IgnoreIngressResponse(_message.Message):
    __slots__ = ("entry",)
    ENTRY_FIELD_NUMBER: _ClassVar[int]
    entry: IngressEntry
    def __init__(self, entry: _Optional[_Union[IngressEntry, _Mapping]] = ...) -> None: ...

class PruneIngressRequest(_message.Message):
    __slots__ = ("hostname",)
    HOSTNAME_FIELD_NUMBER: _ClassVar[int]
    hostname: str
    def __init__(self, hostname: _Optional[str] = ...) -> None: ...

class PruneIngressResponse(_message.Message):
    __slots__ = ("pruned",)
    PRUNED_FIELD_NUMBER: _ClassVar[int]
    pruned: bool
    def __init__(self, pruned: _Optional[bool] = ...) -> None: ...

class SetPublicExposureRequest(_message.Message):
    __slots__ = ("enabled",)
    ENABLED_FIELD_NUMBER: _ClassVar[int]
    enabled: bool
    def __init__(self, enabled: _Optional[bool] = ...) -> None: ...

class SetPublicExposureResponse(_message.Message):
    __slots__ = ("config",)
    CONFIG_FIELD_NUMBER: _ClassVar[int]
    config: TunnelConfig
    def __init__(self, config: _Optional[_Union[TunnelConfig, _Mapping]] = ...) -> None: ...

class GetAccessStatusRequest(_message.Message):
    __slots__ = ()
    def __init__(self) -> None: ...

class AccessHostState(_message.Message):
    __slots__ = ("host", "override", "effective_bypass", "managed", "app_id")
    HOST_FIELD_NUMBER: _ClassVar[int]
    OVERRIDE_FIELD_NUMBER: _ClassVar[int]
    EFFECTIVE_BYPASS_FIELD_NUMBER: _ClassVar[int]
    MANAGED_FIELD_NUMBER: _ClassVar[int]
    APP_ID_FIELD_NUMBER: _ClassVar[int]
    host: str
    override: str
    effective_bypass: bool
    managed: bool
    app_id: str
    def __init__(self, host: _Optional[str] = ..., override: _Optional[str] = ..., effective_bypass: _Optional[bool] = ..., managed: _Optional[bool] = ..., app_id: _Optional[str] = ...) -> None: ...

class AccessStatus(_message.Message):
    __slots__ = ("enabled", "configured", "hosts", "to_create", "to_remove")
    ENABLED_FIELD_NUMBER: _ClassVar[int]
    CONFIGURED_FIELD_NUMBER: _ClassVar[int]
    HOSTS_FIELD_NUMBER: _ClassVar[int]
    TO_CREATE_FIELD_NUMBER: _ClassVar[int]
    TO_REMOVE_FIELD_NUMBER: _ClassVar[int]
    enabled: bool
    configured: bool
    hosts: _containers.RepeatedCompositeFieldContainer[AccessHostState]
    to_create: _containers.RepeatedScalarFieldContainer[str]
    to_remove: _containers.RepeatedScalarFieldContainer[str]
    def __init__(self, enabled: _Optional[bool] = ..., configured: _Optional[bool] = ..., hosts: _Optional[_Iterable[_Union[AccessHostState, _Mapping]]] = ..., to_create: _Optional[_Iterable[str]] = ..., to_remove: _Optional[_Iterable[str]] = ...) -> None: ...

class GetAccessStatusResponse(_message.Message):
    __slots__ = ("status",)
    STATUS_FIELD_NUMBER: _ClassVar[int]
    status: AccessStatus
    def __init__(self, status: _Optional[_Union[AccessStatus, _Mapping]] = ...) -> None: ...
