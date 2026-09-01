import datetime

from google.protobuf import timestamp_pb2 as _timestamp_pb2
from google.protobuf.internal import containers as _containers
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Iterable as _Iterable, Mapping as _Mapping
from typing import ClassVar as _ClassVar, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class ConnectionLocator(_message.Message):
    __slots__ = ("kind", "value", "ordinal")
    KIND_FIELD_NUMBER: _ClassVar[int]
    VALUE_FIELD_NUMBER: _ClassVar[int]
    ORDINAL_FIELD_NUMBER: _ClassVar[int]
    kind: str
    value: str
    ordinal: int
    def __init__(self, kind: _Optional[str] = ..., value: _Optional[str] = ..., ordinal: _Optional[int] = ...) -> None: ...

class NodeLineage(_message.Message):
    __slots__ = ("node_id", "current", "linked_at", "superseded_at", "correlation_id")
    NODE_ID_FIELD_NUMBER: _ClassVar[int]
    CURRENT_FIELD_NUMBER: _ClassVar[int]
    LINKED_AT_FIELD_NUMBER: _ClassVar[int]
    SUPERSEDED_AT_FIELD_NUMBER: _ClassVar[int]
    CORRELATION_ID_FIELD_NUMBER: _ClassVar[int]
    node_id: str
    current: bool
    linked_at: _timestamp_pb2.Timestamp
    superseded_at: _timestamp_pb2.Timestamp
    correlation_id: str
    def __init__(self, node_id: _Optional[str] = ..., current: _Optional[bool] = ..., linked_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., superseded_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., correlation_id: _Optional[str] = ...) -> None: ...

class Machine(_message.Message):
    __slots__ = ("id", "lifecycle", "version", "desired_profile_id", "desired_profile_version", "locators", "node_lineage", "created_at", "updated_at", "archived_at", "removed_at", "applied_profile_id", "applied_profile_version", "applied_at", "desired_selection_json", "applied_selection_json")
    ID_FIELD_NUMBER: _ClassVar[int]
    LIFECYCLE_FIELD_NUMBER: _ClassVar[int]
    VERSION_FIELD_NUMBER: _ClassVar[int]
    DESIRED_PROFILE_ID_FIELD_NUMBER: _ClassVar[int]
    DESIRED_PROFILE_VERSION_FIELD_NUMBER: _ClassVar[int]
    LOCATORS_FIELD_NUMBER: _ClassVar[int]
    NODE_LINEAGE_FIELD_NUMBER: _ClassVar[int]
    CREATED_AT_FIELD_NUMBER: _ClassVar[int]
    UPDATED_AT_FIELD_NUMBER: _ClassVar[int]
    ARCHIVED_AT_FIELD_NUMBER: _ClassVar[int]
    REMOVED_AT_FIELD_NUMBER: _ClassVar[int]
    APPLIED_PROFILE_ID_FIELD_NUMBER: _ClassVar[int]
    APPLIED_PROFILE_VERSION_FIELD_NUMBER: _ClassVar[int]
    APPLIED_AT_FIELD_NUMBER: _ClassVar[int]
    DESIRED_SELECTION_JSON_FIELD_NUMBER: _ClassVar[int]
    APPLIED_SELECTION_JSON_FIELD_NUMBER: _ClassVar[int]
    id: str
    lifecycle: str
    version: int
    desired_profile_id: str
    desired_profile_version: str
    locators: _containers.RepeatedCompositeFieldContainer[ConnectionLocator]
    node_lineage: _containers.RepeatedCompositeFieldContainer[NodeLineage]
    created_at: _timestamp_pb2.Timestamp
    updated_at: _timestamp_pb2.Timestamp
    archived_at: _timestamp_pb2.Timestamp
    removed_at: _timestamp_pb2.Timestamp
    applied_profile_id: str
    applied_profile_version: str
    applied_at: _timestamp_pb2.Timestamp
    desired_selection_json: str
    applied_selection_json: str
    def __init__(self, id: _Optional[str] = ..., lifecycle: _Optional[str] = ..., version: _Optional[int] = ..., desired_profile_id: _Optional[str] = ..., desired_profile_version: _Optional[str] = ..., locators: _Optional[_Iterable[_Union[ConnectionLocator, _Mapping]]] = ..., node_lineage: _Optional[_Iterable[_Union[NodeLineage, _Mapping]]] = ..., created_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., updated_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., archived_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., removed_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., applied_profile_id: _Optional[str] = ..., applied_profile_version: _Optional[str] = ..., applied_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., desired_selection_json: _Optional[str] = ..., applied_selection_json: _Optional[str] = ...) -> None: ...

class CreateMachineRequest(_message.Message):
    __slots__ = ("locators", "desired_profile_id", "desired_profile_version")
    LOCATORS_FIELD_NUMBER: _ClassVar[int]
    DESIRED_PROFILE_ID_FIELD_NUMBER: _ClassVar[int]
    DESIRED_PROFILE_VERSION_FIELD_NUMBER: _ClassVar[int]
    locators: _containers.RepeatedCompositeFieldContainer[ConnectionLocator]
    desired_profile_id: str
    desired_profile_version: str
    def __init__(self, locators: _Optional[_Iterable[_Union[ConnectionLocator, _Mapping]]] = ..., desired_profile_id: _Optional[str] = ..., desired_profile_version: _Optional[str] = ...) -> None: ...

class CreateMachineResponse(_message.Message):
    __slots__ = ("machine",)
    MACHINE_FIELD_NUMBER: _ClassVar[int]
    machine: Machine
    def __init__(self, machine: _Optional[_Union[Machine, _Mapping]] = ...) -> None: ...

class GetMachineRequest(_message.Message):
    __slots__ = ("id",)
    ID_FIELD_NUMBER: _ClassVar[int]
    id: str
    def __init__(self, id: _Optional[str] = ...) -> None: ...

class EnrollmentAttempt(_message.Message):
    __slots__ = ("id", "retry_of_attempt_id", "correlation_id", "state", "terminal_result", "diagnostics", "created_at", "terminal_at")
    ID_FIELD_NUMBER: _ClassVar[int]
    RETRY_OF_ATTEMPT_ID_FIELD_NUMBER: _ClassVar[int]
    CORRELATION_ID_FIELD_NUMBER: _ClassVar[int]
    STATE_FIELD_NUMBER: _ClassVar[int]
    TERMINAL_RESULT_FIELD_NUMBER: _ClassVar[int]
    DIAGNOSTICS_FIELD_NUMBER: _ClassVar[int]
    CREATED_AT_FIELD_NUMBER: _ClassVar[int]
    TERMINAL_AT_FIELD_NUMBER: _ClassVar[int]
    id: str
    retry_of_attempt_id: str
    correlation_id: str
    state: str
    terminal_result: str
    diagnostics: str
    created_at: _timestamp_pb2.Timestamp
    terminal_at: _timestamp_pb2.Timestamp
    def __init__(self, id: _Optional[str] = ..., retry_of_attempt_id: _Optional[str] = ..., correlation_id: _Optional[str] = ..., state: _Optional[str] = ..., terminal_result: _Optional[str] = ..., diagnostics: _Optional[str] = ..., created_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., terminal_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ...) -> None: ...

class CurrentNodeProjection(_message.Message):
    __slots__ = ("node_id", "name", "capabilities", "online")
    NODE_ID_FIELD_NUMBER: _ClassVar[int]
    NAME_FIELD_NUMBER: _ClassVar[int]
    CAPABILITIES_FIELD_NUMBER: _ClassVar[int]
    ONLINE_FIELD_NUMBER: _ClassVar[int]
    node_id: str
    name: str
    capabilities: _containers.RepeatedScalarFieldContainer[str]
    online: bool
    def __init__(self, node_id: _Optional[str] = ..., name: _Optional[str] = ..., capabilities: _Optional[_Iterable[str]] = ..., online: _Optional[bool] = ...) -> None: ...

class MachineAuditEvent(_message.Message):
    __slots__ = ("id", "action", "actor", "detail", "created_at")
    ID_FIELD_NUMBER: _ClassVar[int]
    ACTION_FIELD_NUMBER: _ClassVar[int]
    ACTOR_FIELD_NUMBER: _ClassVar[int]
    DETAIL_FIELD_NUMBER: _ClassVar[int]
    CREATED_AT_FIELD_NUMBER: _ClassVar[int]
    id: str
    action: str
    actor: str
    detail: str
    created_at: _timestamp_pb2.Timestamp
    def __init__(self, id: _Optional[str] = ..., action: _Optional[str] = ..., actor: _Optional[str] = ..., detail: _Optional[str] = ..., created_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ...) -> None: ...

class MachineReadiness(_message.Message):
    __slots__ = ("ready", "reasons")
    READY_FIELD_NUMBER: _ClassVar[int]
    REASONS_FIELD_NUMBER: _ClassVar[int]
    ready: bool
    reasons: _containers.RepeatedScalarFieldContainer[str]
    def __init__(self, ready: _Optional[bool] = ..., reasons: _Optional[_Iterable[str]] = ...) -> None: ...

class MachineDrift(_message.Message):
    __slots__ = ("kind", "name", "reason")
    KIND_FIELD_NUMBER: _ClassVar[int]
    NAME_FIELD_NUMBER: _ClassVar[int]
    REASON_FIELD_NUMBER: _ClassVar[int]
    kind: str
    name: str
    reason: str
    def __init__(self, kind: _Optional[str] = ..., name: _Optional[str] = ..., reason: _Optional[str] = ...) -> None: ...

class GetMachineResponse(_message.Message):
    __slots__ = ("machine", "enrollment_attempts", "current_node", "audit_events", "readiness", "cleanup_tombstones", "drift", "effective_policy")
    MACHINE_FIELD_NUMBER: _ClassVar[int]
    ENROLLMENT_ATTEMPTS_FIELD_NUMBER: _ClassVar[int]
    CURRENT_NODE_FIELD_NUMBER: _ClassVar[int]
    AUDIT_EVENTS_FIELD_NUMBER: _ClassVar[int]
    READINESS_FIELD_NUMBER: _ClassVar[int]
    CLEANUP_TOMBSTONES_FIELD_NUMBER: _ClassVar[int]
    DRIFT_FIELD_NUMBER: _ClassVar[int]
    EFFECTIVE_POLICY_FIELD_NUMBER: _ClassVar[int]
    machine: Machine
    enrollment_attempts: _containers.RepeatedCompositeFieldContainer[EnrollmentAttempt]
    current_node: CurrentNodeProjection
    audit_events: _containers.RepeatedCompositeFieldContainer[MachineAuditEvent]
    readiness: MachineReadiness
    cleanup_tombstones: _containers.RepeatedCompositeFieldContainer[MachineCleanup]
    drift: _containers.RepeatedCompositeFieldContainer[MachineDrift]
    effective_policy: EffectivePolicy
    def __init__(self, machine: _Optional[_Union[Machine, _Mapping]] = ..., enrollment_attempts: _Optional[_Iterable[_Union[EnrollmentAttempt, _Mapping]]] = ..., current_node: _Optional[_Union[CurrentNodeProjection, _Mapping]] = ..., audit_events: _Optional[_Iterable[_Union[MachineAuditEvent, _Mapping]]] = ..., readiness: _Optional[_Union[MachineReadiness, _Mapping]] = ..., cleanup_tombstones: _Optional[_Iterable[_Union[MachineCleanup, _Mapping]]] = ..., drift: _Optional[_Iterable[_Union[MachineDrift, _Mapping]]] = ..., effective_policy: _Optional[_Union[EffectivePolicy, _Mapping]] = ...) -> None: ...

class ListMachinesRequest(_message.Message):
    __slots__ = ()
    def __init__(self) -> None: ...

class ListMachinesResponse(_message.Message):
    __slots__ = ("machines",)
    MACHINES_FIELD_NUMBER: _ClassVar[int]
    machines: _containers.RepeatedCompositeFieldContainer[Machine]
    def __init__(self, machines: _Optional[_Iterable[_Union[Machine, _Mapping]]] = ...) -> None: ...

class ArchiveMachineRequest(_message.Message):
    __slots__ = ("id", "version")
    ID_FIELD_NUMBER: _ClassVar[int]
    VERSION_FIELD_NUMBER: _ClassVar[int]
    id: str
    version: int
    def __init__(self, id: _Optional[str] = ..., version: _Optional[int] = ...) -> None: ...

class ArchiveMachineResponse(_message.Message):
    __slots__ = ("machine",)
    MACHINE_FIELD_NUMBER: _ClassVar[int]
    machine: Machine
    def __init__(self, machine: _Optional[_Union[Machine, _Mapping]] = ...) -> None: ...

class RemoveMachineRequest(_message.Message):
    __slots__ = ("id", "version")
    ID_FIELD_NUMBER: _ClassVar[int]
    VERSION_FIELD_NUMBER: _ClassVar[int]
    id: str
    version: int
    def __init__(self, id: _Optional[str] = ..., version: _Optional[int] = ...) -> None: ...

class RemoveMachineResponse(_message.Message):
    __slots__ = ("machine",)
    MACHINE_FIELD_NUMBER: _ClassVar[int]
    machine: Machine
    def __init__(self, machine: _Optional[_Union[Machine, _Mapping]] = ...) -> None: ...

class MachineTrust(_message.Message):
    __slots__ = ("client_key_fingerprint", "host_key_fingerprint", "host_key_state", "updated_at", "ssh_user", "ssh_port", "connection_state")
    CLIENT_KEY_FINGERPRINT_FIELD_NUMBER: _ClassVar[int]
    HOST_KEY_FINGERPRINT_FIELD_NUMBER: _ClassVar[int]
    HOST_KEY_STATE_FIELD_NUMBER: _ClassVar[int]
    UPDATED_AT_FIELD_NUMBER: _ClassVar[int]
    SSH_USER_FIELD_NUMBER: _ClassVar[int]
    SSH_PORT_FIELD_NUMBER: _ClassVar[int]
    CONNECTION_STATE_FIELD_NUMBER: _ClassVar[int]
    client_key_fingerprint: str
    host_key_fingerprint: str
    host_key_state: str
    updated_at: _timestamp_pb2.Timestamp
    ssh_user: str
    ssh_port: int
    connection_state: str
    def __init__(self, client_key_fingerprint: _Optional[str] = ..., host_key_fingerprint: _Optional[str] = ..., host_key_state: _Optional[str] = ..., updated_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., ssh_user: _Optional[str] = ..., ssh_port: _Optional[int] = ..., connection_state: _Optional[str] = ...) -> None: ...

class GetMachineTrustRequest(_message.Message):
    __slots__ = ("machine_id",)
    MACHINE_ID_FIELD_NUMBER: _ClassVar[int]
    machine_id: str
    def __init__(self, machine_id: _Optional[str] = ...) -> None: ...

class GetMachineTrustResponse(_message.Message):
    __slots__ = ("trust",)
    TRUST_FIELD_NUMBER: _ClassVar[int]
    trust: MachineTrust
    def __init__(self, trust: _Optional[_Union[MachineTrust, _Mapping]] = ...) -> None: ...

class ReviewMachineHostKeyRequest(_message.Message):
    __slots__ = ("machine_id", "replacement_host_key_fingerprint")
    MACHINE_ID_FIELD_NUMBER: _ClassVar[int]
    REPLACEMENT_HOST_KEY_FINGERPRINT_FIELD_NUMBER: _ClassVar[int]
    machine_id: str
    replacement_host_key_fingerprint: str
    def __init__(self, machine_id: _Optional[str] = ..., replacement_host_key_fingerprint: _Optional[str] = ...) -> None: ...

class ReviewMachineHostKeyResponse(_message.Message):
    __slots__ = ("trust",)
    TRUST_FIELD_NUMBER: _ClassVar[int]
    trust: MachineTrust
    def __init__(self, trust: _Optional[_Union[MachineTrust, _Mapping]] = ...) -> None: ...

class MachineCleanup(_message.Message):
    __slots__ = ("id", "machine_id", "action", "status", "detail", "created_at", "updated_at", "acknowledged_at")
    ID_FIELD_NUMBER: _ClassVar[int]
    MACHINE_ID_FIELD_NUMBER: _ClassVar[int]
    ACTION_FIELD_NUMBER: _ClassVar[int]
    STATUS_FIELD_NUMBER: _ClassVar[int]
    DETAIL_FIELD_NUMBER: _ClassVar[int]
    CREATED_AT_FIELD_NUMBER: _ClassVar[int]
    UPDATED_AT_FIELD_NUMBER: _ClassVar[int]
    ACKNOWLEDGED_AT_FIELD_NUMBER: _ClassVar[int]
    id: str
    machine_id: str
    action: str
    status: str
    detail: str
    created_at: _timestamp_pb2.Timestamp
    updated_at: _timestamp_pb2.Timestamp
    acknowledged_at: _timestamp_pb2.Timestamp
    def __init__(self, id: _Optional[str] = ..., machine_id: _Optional[str] = ..., action: _Optional[str] = ..., status: _Optional[str] = ..., detail: _Optional[str] = ..., created_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., updated_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., acknowledged_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ...) -> None: ...

class RequestMachineSSHCleanupRequest(_message.Message):
    __slots__ = ("machine_id",)
    MACHINE_ID_FIELD_NUMBER: _ClassVar[int]
    machine_id: str
    def __init__(self, machine_id: _Optional[str] = ...) -> None: ...

class RequestMachineSSHCleanupResponse(_message.Message):
    __slots__ = ("cleanup",)
    CLEANUP_FIELD_NUMBER: _ClassVar[int]
    cleanup: MachineCleanup
    def __init__(self, cleanup: _Optional[_Union[MachineCleanup, _Mapping]] = ...) -> None: ...

class UpdateMachineCleanupRequest(_message.Message):
    __slots__ = ("id", "status", "detail")
    ID_FIELD_NUMBER: _ClassVar[int]
    STATUS_FIELD_NUMBER: _ClassVar[int]
    DETAIL_FIELD_NUMBER: _ClassVar[int]
    id: str
    status: str
    detail: str
    def __init__(self, id: _Optional[str] = ..., status: _Optional[str] = ..., detail: _Optional[str] = ...) -> None: ...

class UpdateMachineCleanupResponse(_message.Message):
    __slots__ = ("cleanup",)
    CLEANUP_FIELD_NUMBER: _ClassVar[int]
    cleanup: MachineCleanup
    def __init__(self, cleanup: _Optional[_Union[MachineCleanup, _Mapping]] = ...) -> None: ...

class EffectivePolicy(_message.Message):
    __slots__ = ("profile_id", "profile_version", "setup_preset", "suggested_scopes", "required_capabilities", "snapshot_json", "scenarios", "optional_resources")
    PROFILE_ID_FIELD_NUMBER: _ClassVar[int]
    PROFILE_VERSION_FIELD_NUMBER: _ClassVar[int]
    SETUP_PRESET_FIELD_NUMBER: _ClassVar[int]
    SUGGESTED_SCOPES_FIELD_NUMBER: _ClassVar[int]
    REQUIRED_CAPABILITIES_FIELD_NUMBER: _ClassVar[int]
    SNAPSHOT_JSON_FIELD_NUMBER: _ClassVar[int]
    SCENARIOS_FIELD_NUMBER: _ClassVar[int]
    OPTIONAL_RESOURCES_FIELD_NUMBER: _ClassVar[int]
    profile_id: str
    profile_version: str
    setup_preset: str
    suggested_scopes: _containers.RepeatedScalarFieldContainer[str]
    required_capabilities: _containers.RepeatedScalarFieldContainer[str]
    snapshot_json: str
    scenarios: _containers.RepeatedScalarFieldContainer[str]
    optional_resources: _containers.RepeatedScalarFieldContainer[str]
    def __init__(self, profile_id: _Optional[str] = ..., profile_version: _Optional[str] = ..., setup_preset: _Optional[str] = ..., suggested_scopes: _Optional[_Iterable[str]] = ..., required_capabilities: _Optional[_Iterable[str]] = ..., snapshot_json: _Optional[str] = ..., scenarios: _Optional[_Iterable[str]] = ..., optional_resources: _Optional[_Iterable[str]] = ...) -> None: ...

class ApplyMachinePolicyRequest(_message.Message):
    __slots__ = ("machine_id", "version", "profile_id", "profile_version", "overrides", "reason", "confirm_removal")
    class OverridesEntry(_message.Message):
        __slots__ = ("key", "value")
        KEY_FIELD_NUMBER: _ClassVar[int]
        VALUE_FIELD_NUMBER: _ClassVar[int]
        key: str
        value: str
        def __init__(self, key: _Optional[str] = ..., value: _Optional[str] = ...) -> None: ...
    MACHINE_ID_FIELD_NUMBER: _ClassVar[int]
    VERSION_FIELD_NUMBER: _ClassVar[int]
    PROFILE_ID_FIELD_NUMBER: _ClassVar[int]
    PROFILE_VERSION_FIELD_NUMBER: _ClassVar[int]
    OVERRIDES_FIELD_NUMBER: _ClassVar[int]
    REASON_FIELD_NUMBER: _ClassVar[int]
    CONFIRM_REMOVAL_FIELD_NUMBER: _ClassVar[int]
    machine_id: str
    version: int
    profile_id: str
    profile_version: str
    overrides: _containers.ScalarMap[str, str]
    reason: str
    confirm_removal: bool
    def __init__(self, machine_id: _Optional[str] = ..., version: _Optional[int] = ..., profile_id: _Optional[str] = ..., profile_version: _Optional[str] = ..., overrides: _Optional[_Mapping[str, str]] = ..., reason: _Optional[str] = ..., confirm_removal: _Optional[bool] = ...) -> None: ...

class ApplyMachinePolicyResponse(_message.Message):
    __slots__ = ("machine", "policy")
    MACHINE_FIELD_NUMBER: _ClassVar[int]
    POLICY_FIELD_NUMBER: _ClassVar[int]
    machine: Machine
    policy: EffectivePolicy
    def __init__(self, machine: _Optional[_Union[Machine, _Mapping]] = ..., policy: _Optional[_Union[EffectivePolicy, _Mapping]] = ...) -> None: ...

class RevokeMachineNodeRequest(_message.Message):
    __slots__ = ("machine_id",)
    MACHINE_ID_FIELD_NUMBER: _ClassVar[int]
    machine_id: str
    def __init__(self, machine_id: _Optional[str] = ...) -> None: ...

class RevokeMachineNodeResponse(_message.Message):
    __slots__ = ("machine", "revoked_node_id")
    MACHINE_FIELD_NUMBER: _ClassVar[int]
    REVOKED_NODE_ID_FIELD_NUMBER: _ClassVar[int]
    machine: Machine
    revoked_node_id: str
    def __init__(self, machine: _Optional[_Union[Machine, _Mapping]] = ..., revoked_node_id: _Optional[str] = ...) -> None: ...

class RepairMachineRequest(_message.Message):
    __slots__ = ("machine_id",)
    MACHINE_ID_FIELD_NUMBER: _ClassVar[int]
    machine_id: str
    def __init__(self, machine_id: _Optional[str] = ...) -> None: ...

class RepairMachineResponse(_message.Message):
    __slots__ = ("machine", "onboarding_op_id", "enrollment_attempt_id")
    MACHINE_FIELD_NUMBER: _ClassVar[int]
    ONBOARDING_OP_ID_FIELD_NUMBER: _ClassVar[int]
    ENROLLMENT_ATTEMPT_ID_FIELD_NUMBER: _ClassVar[int]
    machine: Machine
    onboarding_op_id: str
    enrollment_attempt_id: str
    def __init__(self, machine: _Optional[_Union[Machine, _Mapping]] = ..., onboarding_op_id: _Optional[str] = ..., enrollment_attempt_id: _Optional[str] = ...) -> None: ...

class MergeMachinesRequest(_message.Message):
    __slots__ = ("from_machine_id", "into_machine_id")
    FROM_MACHINE_ID_FIELD_NUMBER: _ClassVar[int]
    INTO_MACHINE_ID_FIELD_NUMBER: _ClassVar[int]
    from_machine_id: str
    into_machine_id: str
    def __init__(self, from_machine_id: _Optional[str] = ..., into_machine_id: _Optional[str] = ...) -> None: ...

class MergeMachinesResponse(_message.Message):
    __slots__ = ("machine", "archived_machine_id")
    MACHINE_FIELD_NUMBER: _ClassVar[int]
    ARCHIVED_MACHINE_ID_FIELD_NUMBER: _ClassVar[int]
    machine: Machine
    archived_machine_id: str
    def __init__(self, machine: _Optional[_Union[Machine, _Mapping]] = ..., archived_machine_id: _Optional[str] = ...) -> None: ...
