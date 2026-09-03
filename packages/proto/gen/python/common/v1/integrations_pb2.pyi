from google.protobuf.internal import containers as _containers
from google.protobuf.internal import enum_type_wrapper as _enum_type_wrapper
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Iterable as _Iterable, Mapping as _Mapping
from typing import ClassVar as _ClassVar, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class DependencyKind(int, metaclass=_enum_type_wrapper.EnumTypeWrapper):
    __slots__ = ()
    DEPENDENCY_KIND_UNSPECIFIED: _ClassVar[DependencyKind]
    DEPENDENCY_KIND_SCENARIO: _ClassVar[DependencyKind]
    DEPENDENCY_KIND_RESOURCE: _ClassVar[DependencyKind]
    DEPENDENCY_KIND_CONTROL_PLANE: _ClassVar[DependencyKind]

class LifecycleStatus(int, metaclass=_enum_type_wrapper.EnumTypeWrapper):
    __slots__ = ()
    LIFECYCLE_STATUS_UNSPECIFIED: _ClassVar[LifecycleStatus]
    LIFECYCLE_STATUS_AVAILABLE: _ClassVar[LifecycleStatus]
    LIFECYCLE_STATUS_UNAVAILABLE: _ClassVar[LifecycleStatus]
    LIFECYCLE_STATUS_UNKNOWN: _ClassVar[LifecycleStatus]

class FeatureStatus(int, metaclass=_enum_type_wrapper.EnumTypeWrapper):
    __slots__ = ()
    FEATURE_STATUS_UNSPECIFIED: _ClassVar[FeatureStatus]
    FEATURE_STATUS_COMPATIBLE: _ClassVar[FeatureStatus]
    FEATURE_STATUS_INCOMPATIBLE: _ClassVar[FeatureStatus]
    FEATURE_STATUS_MISSING: _ClassVar[FeatureStatus]
    FEATURE_STATUS_UNKNOWN: _ClassVar[FeatureStatus]

class ActionKind(int, metaclass=_enum_type_wrapper.EnumTypeWrapper):
    __slots__ = ()
    ACTION_KIND_UNSPECIFIED: _ClassVar[ActionKind]
    ACTION_KIND_OWNER_GUIDANCE: _ClassVar[ActionKind]
    ACTION_KIND_SCENARIO_START: _ClassVar[ActionKind]
    ACTION_KIND_SCENARIO_RESTART: _ClassVar[ActionKind]
    ACTION_KIND_OPERATOR_COMMAND: _ClassVar[ActionKind]

class Criticality(int, metaclass=_enum_type_wrapper.EnumTypeWrapper):
    __slots__ = ()
    CRITICALITY_UNSPECIFIED: _ClassVar[Criticality]
    CRITICALITY_REQUIRED: _ClassVar[Criticality]
    CRITICALITY_OPTIONAL: _ClassVar[Criticality]

class PlatformSupport(int, metaclass=_enum_type_wrapper.EnumTypeWrapper):
    __slots__ = ()
    PLATFORM_SUPPORT_UNSPECIFIED: _ClassVar[PlatformSupport]
    PLATFORM_SUPPORT_SUPPORTED: _ClassVar[PlatformSupport]
    PLATFORM_SUPPORT_DEGRADED: _ClassVar[PlatformSupport]
    PLATFORM_SUPPORT_UNSUPPORTED: _ClassVar[PlatformSupport]

class ConnectionStatus(int, metaclass=_enum_type_wrapper.EnumTypeWrapper):
    __slots__ = ()
    CONNECTION_STATUS_UNSPECIFIED: _ClassVar[ConnectionStatus]
    CONNECTION_STATUS_CONNECTED: _ClassVar[ConnectionStatus]
    CONNECTION_STATUS_CHECKING: _ClassVar[ConnectionStatus]
    CONNECTION_STATUS_NEEDS_REAUTHORIZATION: _ClassVar[ConnectionStatus]
    CONNECTION_STATUS_REVOKED: _ClassVar[ConnectionStatus]
    CONNECTION_STATUS_PROVIDER_OUTAGE: _ClassVar[ConnectionStatus]
    CONNECTION_STATUS_UNKNOWN: _ClassVar[ConnectionStatus]
    CONNECTION_STATUS_DISCONNECTED: _ClassVar[ConnectionStatus]
    CONNECTION_STATUS_EXPIRED: _ClassVar[ConnectionStatus]
    CONNECTION_STATUS_INSUFFICIENT_SCOPE: _ClassVar[ConnectionStatus]
    CONNECTION_STATUS_PROVIDER_UNAVAILABLE: _ClassVar[ConnectionStatus]
    CONNECTION_STATUS_OFFLINE: _ClassVar[ConnectionStatus]

class ConnectionActionKind(int, metaclass=_enum_type_wrapper.EnumTypeWrapper):
    __slots__ = ()
    CONNECTION_ACTION_KIND_UNSPECIFIED: _ClassVar[ConnectionActionKind]
    CONNECTION_ACTION_KIND_CONNECT: _ClassVar[ConnectionActionKind]
    CONNECTION_ACTION_KIND_TEST: _ClassVar[ConnectionActionKind]
    CONNECTION_ACTION_KIND_REFRESH: _ClassVar[ConnectionActionKind]
    CONNECTION_ACTION_KIND_ROTATE: _ClassVar[ConnectionActionKind]
    CONNECTION_ACTION_KIND_BIND: _ClassVar[ConnectionActionKind]
    CONNECTION_ACTION_KIND_UNBIND: _ClassVar[ConnectionActionKind]
    CONNECTION_ACTION_KIND_REVOKE: _ClassVar[ConnectionActionKind]
    CONNECTION_ACTION_KIND_DELETE: _ClassVar[ConnectionActionKind]
DEPENDENCY_KIND_UNSPECIFIED: DependencyKind
DEPENDENCY_KIND_SCENARIO: DependencyKind
DEPENDENCY_KIND_RESOURCE: DependencyKind
DEPENDENCY_KIND_CONTROL_PLANE: DependencyKind
LIFECYCLE_STATUS_UNSPECIFIED: LifecycleStatus
LIFECYCLE_STATUS_AVAILABLE: LifecycleStatus
LIFECYCLE_STATUS_UNAVAILABLE: LifecycleStatus
LIFECYCLE_STATUS_UNKNOWN: LifecycleStatus
FEATURE_STATUS_UNSPECIFIED: FeatureStatus
FEATURE_STATUS_COMPATIBLE: FeatureStatus
FEATURE_STATUS_INCOMPATIBLE: FeatureStatus
FEATURE_STATUS_MISSING: FeatureStatus
FEATURE_STATUS_UNKNOWN: FeatureStatus
ACTION_KIND_UNSPECIFIED: ActionKind
ACTION_KIND_OWNER_GUIDANCE: ActionKind
ACTION_KIND_SCENARIO_START: ActionKind
ACTION_KIND_SCENARIO_RESTART: ActionKind
ACTION_KIND_OPERATOR_COMMAND: ActionKind
CRITICALITY_UNSPECIFIED: Criticality
CRITICALITY_REQUIRED: Criticality
CRITICALITY_OPTIONAL: Criticality
PLATFORM_SUPPORT_UNSPECIFIED: PlatformSupport
PLATFORM_SUPPORT_SUPPORTED: PlatformSupport
PLATFORM_SUPPORT_DEGRADED: PlatformSupport
PLATFORM_SUPPORT_UNSUPPORTED: PlatformSupport
CONNECTION_STATUS_UNSPECIFIED: ConnectionStatus
CONNECTION_STATUS_CONNECTED: ConnectionStatus
CONNECTION_STATUS_CHECKING: ConnectionStatus
CONNECTION_STATUS_NEEDS_REAUTHORIZATION: ConnectionStatus
CONNECTION_STATUS_REVOKED: ConnectionStatus
CONNECTION_STATUS_PROVIDER_OUTAGE: ConnectionStatus
CONNECTION_STATUS_UNKNOWN: ConnectionStatus
CONNECTION_STATUS_DISCONNECTED: ConnectionStatus
CONNECTION_STATUS_EXPIRED: ConnectionStatus
CONNECTION_STATUS_INSUFFICIENT_SCOPE: ConnectionStatus
CONNECTION_STATUS_PROVIDER_UNAVAILABLE: ConnectionStatus
CONNECTION_STATUS_OFFLINE: ConnectionStatus
CONNECTION_ACTION_KIND_UNSPECIFIED: ConnectionActionKind
CONNECTION_ACTION_KIND_CONNECT: ConnectionActionKind
CONNECTION_ACTION_KIND_TEST: ConnectionActionKind
CONNECTION_ACTION_KIND_REFRESH: ConnectionActionKind
CONNECTION_ACTION_KIND_ROTATE: ConnectionActionKind
CONNECTION_ACTION_KIND_BIND: ConnectionActionKind
CONNECTION_ACTION_KIND_UNBIND: ConnectionActionKind
CONNECTION_ACTION_KIND_REVOKE: ConnectionActionKind
CONNECTION_ACTION_KIND_DELETE: ConnectionActionKind

class PlatformVerdict(_message.Message):
    __slots__ = ("support", "reason")
    SUPPORT_FIELD_NUMBER: _ClassVar[int]
    REASON_FIELD_NUMBER: _ClassVar[int]
    support: PlatformSupport
    reason: str
    def __init__(self, support: _Optional[_Union[PlatformSupport, str]] = ..., reason: _Optional[str] = ...) -> None: ...

class Integration(_message.Message):
    __slots__ = ("id", "name", "description", "dependency_kind", "dependency_slug", "enabled", "required", "startup_policy", "features", "lifecycle", "action", "origin", "criticality", "platform")
    ID_FIELD_NUMBER: _ClassVar[int]
    NAME_FIELD_NUMBER: _ClassVar[int]
    DESCRIPTION_FIELD_NUMBER: _ClassVar[int]
    DEPENDENCY_KIND_FIELD_NUMBER: _ClassVar[int]
    DEPENDENCY_SLUG_FIELD_NUMBER: _ClassVar[int]
    ENABLED_FIELD_NUMBER: _ClassVar[int]
    REQUIRED_FIELD_NUMBER: _ClassVar[int]
    STARTUP_POLICY_FIELD_NUMBER: _ClassVar[int]
    FEATURES_FIELD_NUMBER: _ClassVar[int]
    LIFECYCLE_FIELD_NUMBER: _ClassVar[int]
    ACTION_FIELD_NUMBER: _ClassVar[int]
    ORIGIN_FIELD_NUMBER: _ClassVar[int]
    CRITICALITY_FIELD_NUMBER: _ClassVar[int]
    PLATFORM_FIELD_NUMBER: _ClassVar[int]
    id: str
    name: str
    description: str
    dependency_kind: DependencyKind
    dependency_slug: str
    enabled: bool
    required: bool
    startup_policy: str
    features: _containers.RepeatedCompositeFieldContainer[Feature]
    lifecycle: LifecycleState
    action: ActionPolicy
    origin: str
    criticality: Criticality
    platform: PlatformVerdict
    def __init__(self, id: _Optional[str] = ..., name: _Optional[str] = ..., description: _Optional[str] = ..., dependency_kind: _Optional[_Union[DependencyKind, str]] = ..., dependency_slug: _Optional[str] = ..., enabled: _Optional[bool] = ..., required: _Optional[bool] = ..., startup_policy: _Optional[str] = ..., features: _Optional[_Iterable[_Union[Feature, _Mapping]]] = ..., lifecycle: _Optional[_Union[LifecycleState, _Mapping]] = ..., action: _Optional[_Union[ActionPolicy, _Mapping]] = ..., origin: _Optional[str] = ..., criticality: _Optional[_Union[Criticality, str]] = ..., platform: _Optional[_Union[PlatformVerdict, _Mapping]] = ...) -> None: ...

class Feature(_message.Message):
    __slots__ = ("id", "contract_version", "expected_unit")
    ID_FIELD_NUMBER: _ClassVar[int]
    CONTRACT_VERSION_FIELD_NUMBER: _ClassVar[int]
    EXPECTED_UNIT_FIELD_NUMBER: _ClassVar[int]
    id: str
    contract_version: str
    expected_unit: str
    def __init__(self, id: _Optional[str] = ..., contract_version: _Optional[str] = ..., expected_unit: _Optional[str] = ...) -> None: ...

class LifecycleState(_message.Message):
    __slots__ = ("status", "reason_code", "message", "checked_at", "latency_ms")
    STATUS_FIELD_NUMBER: _ClassVar[int]
    REASON_CODE_FIELD_NUMBER: _ClassVar[int]
    MESSAGE_FIELD_NUMBER: _ClassVar[int]
    CHECKED_AT_FIELD_NUMBER: _ClassVar[int]
    LATENCY_MS_FIELD_NUMBER: _ClassVar[int]
    status: LifecycleStatus
    reason_code: str
    message: str
    checked_at: str
    latency_ms: int
    def __init__(self, status: _Optional[_Union[LifecycleStatus, str]] = ..., reason_code: _Optional[str] = ..., message: _Optional[str] = ..., checked_at: _Optional[str] = ..., latency_ms: _Optional[int] = ...) -> None: ...

class FeatureState(_message.Message):
    __slots__ = ("feature_id", "status", "reason_code", "message", "checked_at")
    FEATURE_ID_FIELD_NUMBER: _ClassVar[int]
    STATUS_FIELD_NUMBER: _ClassVar[int]
    REASON_CODE_FIELD_NUMBER: _ClassVar[int]
    MESSAGE_FIELD_NUMBER: _ClassVar[int]
    CHECKED_AT_FIELD_NUMBER: _ClassVar[int]
    feature_id: str
    status: FeatureStatus
    reason_code: str
    message: str
    checked_at: str
    def __init__(self, feature_id: _Optional[str] = ..., status: _Optional[_Union[FeatureStatus, str]] = ..., reason_code: _Optional[str] = ..., message: _Optional[str] = ..., checked_at: _Optional[str] = ...) -> None: ...

class ActionPolicy(_message.Message):
    __slots__ = ("kind", "label", "requires_confirmation", "eligible", "owner_guidance")
    KIND_FIELD_NUMBER: _ClassVar[int]
    LABEL_FIELD_NUMBER: _ClassVar[int]
    REQUIRES_CONFIRMATION_FIELD_NUMBER: _ClassVar[int]
    ELIGIBLE_FIELD_NUMBER: _ClassVar[int]
    OWNER_GUIDANCE_FIELD_NUMBER: _ClassVar[int]
    kind: ActionKind
    label: str
    requires_confirmation: bool
    eligible: bool
    owner_guidance: str
    def __init__(self, kind: _Optional[_Union[ActionKind, str]] = ..., label: _Optional[str] = ..., requires_confirmation: _Optional[bool] = ..., eligible: _Optional[bool] = ..., owner_guidance: _Optional[str] = ...) -> None: ...

class ListIntegrationsRequest(_message.Message):
    __slots__ = ()
    def __init__(self) -> None: ...

class ListIntegrationsResponse(_message.Message):
    __slots__ = ("integrations", "generated_at")
    INTEGRATIONS_FIELD_NUMBER: _ClassVar[int]
    GENERATED_AT_FIELD_NUMBER: _ClassVar[int]
    integrations: _containers.RepeatedCompositeFieldContainer[Integration]
    generated_at: str
    def __init__(self, integrations: _Optional[_Iterable[_Union[Integration, _Mapping]]] = ..., generated_at: _Optional[str] = ...) -> None: ...

class GetIntegrationRequest(_message.Message):
    __slots__ = ("integration_id",)
    INTEGRATION_ID_FIELD_NUMBER: _ClassVar[int]
    integration_id: str
    def __init__(self, integration_id: _Optional[str] = ...) -> None: ...

class GetIntegrationResponse(_message.Message):
    __slots__ = ("integration", "features")
    INTEGRATION_FIELD_NUMBER: _ClassVar[int]
    FEATURES_FIELD_NUMBER: _ClassVar[int]
    integration: Integration
    features: _containers.RepeatedCompositeFieldContainer[FeatureState]
    def __init__(self, integration: _Optional[_Union[Integration, _Mapping]] = ..., features: _Optional[_Iterable[_Union[FeatureState, _Mapping]]] = ...) -> None: ...

class RefreshIntegrationsRequest(_message.Message):
    __slots__ = ()
    def __init__(self) -> None: ...

class RunIntegrationActionRequest(_message.Message):
    __slots__ = ("integration_id", "action", "confirmed", "request_id")
    INTEGRATION_ID_FIELD_NUMBER: _ClassVar[int]
    ACTION_FIELD_NUMBER: _ClassVar[int]
    CONFIRMED_FIELD_NUMBER: _ClassVar[int]
    REQUEST_ID_FIELD_NUMBER: _ClassVar[int]
    integration_id: str
    action: ActionKind
    confirmed: bool
    request_id: str
    def __init__(self, integration_id: _Optional[str] = ..., action: _Optional[_Union[ActionKind, str]] = ..., confirmed: _Optional[bool] = ..., request_id: _Optional[str] = ...) -> None: ...

class RunIntegrationActionResponse(_message.Message):
    __slots__ = ("integration_id", "action", "status", "message", "request_id")
    INTEGRATION_ID_FIELD_NUMBER: _ClassVar[int]
    ACTION_FIELD_NUMBER: _ClassVar[int]
    STATUS_FIELD_NUMBER: _ClassVar[int]
    MESSAGE_FIELD_NUMBER: _ClassVar[int]
    REQUEST_ID_FIELD_NUMBER: _ClassVar[int]
    integration_id: str
    action: ActionKind
    status: str
    message: str
    request_id: str
    def __init__(self, integration_id: _Optional[str] = ..., action: _Optional[_Union[ActionKind, str]] = ..., status: _Optional[str] = ..., message: _Optional[str] = ..., request_id: _Optional[str] = ...) -> None: ...

class ConnectionScope(_message.Message):
    __slots__ = ("name", "purpose", "granted")
    NAME_FIELD_NUMBER: _ClassVar[int]
    PURPOSE_FIELD_NUMBER: _ClassVar[int]
    GRANTED_FIELD_NUMBER: _ClassVar[int]
    name: str
    purpose: str
    granted: bool
    def __init__(self, name: _Optional[str] = ..., purpose: _Optional[str] = ..., granted: _Optional[bool] = ...) -> None: ...

class ConnectionBinding(_message.Message):
    __slots__ = ("scenario_slug", "scenario_name", "context")
    SCENARIO_SLUG_FIELD_NUMBER: _ClassVar[int]
    SCENARIO_NAME_FIELD_NUMBER: _ClassVar[int]
    CONTEXT_FIELD_NUMBER: _ClassVar[int]
    scenario_slug: str
    scenario_name: str
    context: str
    def __init__(self, scenario_slug: _Optional[str] = ..., scenario_name: _Optional[str] = ..., context: _Optional[str] = ...) -> None: ...

class Connection(_message.Message):
    __slots__ = ("id", "connector_id", "connector_name", "display_name", "account_label", "account_identity", "status", "scopes", "bindings", "last_verified_at", "freshness", "reason_code", "next_action", "supported_actions", "credential_authority_ref")
    ID_FIELD_NUMBER: _ClassVar[int]
    CONNECTOR_ID_FIELD_NUMBER: _ClassVar[int]
    CONNECTOR_NAME_FIELD_NUMBER: _ClassVar[int]
    DISPLAY_NAME_FIELD_NUMBER: _ClassVar[int]
    ACCOUNT_LABEL_FIELD_NUMBER: _ClassVar[int]
    ACCOUNT_IDENTITY_FIELD_NUMBER: _ClassVar[int]
    STATUS_FIELD_NUMBER: _ClassVar[int]
    SCOPES_FIELD_NUMBER: _ClassVar[int]
    BINDINGS_FIELD_NUMBER: _ClassVar[int]
    LAST_VERIFIED_AT_FIELD_NUMBER: _ClassVar[int]
    FRESHNESS_FIELD_NUMBER: _ClassVar[int]
    REASON_CODE_FIELD_NUMBER: _ClassVar[int]
    NEXT_ACTION_FIELD_NUMBER: _ClassVar[int]
    SUPPORTED_ACTIONS_FIELD_NUMBER: _ClassVar[int]
    CREDENTIAL_AUTHORITY_REF_FIELD_NUMBER: _ClassVar[int]
    id: str
    connector_id: str
    connector_name: str
    display_name: str
    account_label: str
    account_identity: str
    status: ConnectionStatus
    scopes: _containers.RepeatedCompositeFieldContainer[ConnectionScope]
    bindings: _containers.RepeatedCompositeFieldContainer[ConnectionBinding]
    last_verified_at: str
    freshness: str
    reason_code: str
    next_action: str
    supported_actions: _containers.RepeatedScalarFieldContainer[ConnectionActionKind]
    credential_authority_ref: str
    def __init__(self, id: _Optional[str] = ..., connector_id: _Optional[str] = ..., connector_name: _Optional[str] = ..., display_name: _Optional[str] = ..., account_label: _Optional[str] = ..., account_identity: _Optional[str] = ..., status: _Optional[_Union[ConnectionStatus, str]] = ..., scopes: _Optional[_Iterable[_Union[ConnectionScope, _Mapping]]] = ..., bindings: _Optional[_Iterable[_Union[ConnectionBinding, _Mapping]]] = ..., last_verified_at: _Optional[str] = ..., freshness: _Optional[str] = ..., reason_code: _Optional[str] = ..., next_action: _Optional[str] = ..., supported_actions: _Optional[_Iterable[_Union[ConnectionActionKind, str]]] = ..., credential_authority_ref: _Optional[str] = ...) -> None: ...

class ListConnectionsRequest(_message.Message):
    __slots__ = ("connector_id",)
    CONNECTOR_ID_FIELD_NUMBER: _ClassVar[int]
    connector_id: str
    def __init__(self, connector_id: _Optional[str] = ...) -> None: ...

class ListConnectionsResponse(_message.Message):
    __slots__ = ("connections", "generated_at")
    CONNECTIONS_FIELD_NUMBER: _ClassVar[int]
    GENERATED_AT_FIELD_NUMBER: _ClassVar[int]
    connections: _containers.RepeatedCompositeFieldContainer[Connection]
    generated_at: str
    def __init__(self, connections: _Optional[_Iterable[_Union[Connection, _Mapping]]] = ..., generated_at: _Optional[str] = ...) -> None: ...

class GetConnectionRequest(_message.Message):
    __slots__ = ("connection_id",)
    CONNECTION_ID_FIELD_NUMBER: _ClassVar[int]
    connection_id: str
    def __init__(self, connection_id: _Optional[str] = ...) -> None: ...

class GetConnectionResponse(_message.Message):
    __slots__ = ("connection",)
    CONNECTION_FIELD_NUMBER: _ClassVar[int]
    connection: Connection
    def __init__(self, connection: _Optional[_Union[Connection, _Mapping]] = ...) -> None: ...

class ConnectionMutationRequest(_message.Message):
    __slots__ = ("connection_id", "connector_id", "display_name", "request_id", "credential_value", "binding_scenario_slug", "binding_context", "required_scopes")
    CONNECTION_ID_FIELD_NUMBER: _ClassVar[int]
    CONNECTOR_ID_FIELD_NUMBER: _ClassVar[int]
    DISPLAY_NAME_FIELD_NUMBER: _ClassVar[int]
    REQUEST_ID_FIELD_NUMBER: _ClassVar[int]
    CREDENTIAL_VALUE_FIELD_NUMBER: _ClassVar[int]
    BINDING_SCENARIO_SLUG_FIELD_NUMBER: _ClassVar[int]
    BINDING_CONTEXT_FIELD_NUMBER: _ClassVar[int]
    REQUIRED_SCOPES_FIELD_NUMBER: _ClassVar[int]
    connection_id: str
    connector_id: str
    display_name: str
    request_id: str
    credential_value: str
    binding_scenario_slug: str
    binding_context: str
    required_scopes: _containers.RepeatedScalarFieldContainer[str]
    def __init__(self, connection_id: _Optional[str] = ..., connector_id: _Optional[str] = ..., display_name: _Optional[str] = ..., request_id: _Optional[str] = ..., credential_value: _Optional[str] = ..., binding_scenario_slug: _Optional[str] = ..., binding_context: _Optional[str] = ..., required_scopes: _Optional[_Iterable[str]] = ...) -> None: ...

class ConnectionMutationResponse(_message.Message):
    __slots__ = ("connection", "request_id")
    CONNECTION_FIELD_NUMBER: _ClassVar[int]
    REQUEST_ID_FIELD_NUMBER: _ClassVar[int]
    connection: Connection
    request_id: str
    def __init__(self, connection: _Optional[_Union[Connection, _Mapping]] = ..., request_id: _Optional[str] = ...) -> None: ...
