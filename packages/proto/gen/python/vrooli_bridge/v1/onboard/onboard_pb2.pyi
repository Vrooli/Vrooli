import datetime

from google.protobuf import timestamp_pb2 as _timestamp_pb2
from setup.v1 import selection_pb2 as _selection_pb2
from google.protobuf.internal import containers as _containers
from google.protobuf.internal import enum_type_wrapper as _enum_type_wrapper
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Iterable as _Iterable, Mapping as _Mapping
from typing import ClassVar as _ClassVar, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class OnboardingState(int, metaclass=_enum_type_wrapper.EnumTypeWrapper):
    __slots__ = ()
    ONBOARDING_STATE_UNSPECIFIED: _ClassVar[OnboardingState]
    ONBOARDING_STATE_PENDING: _ClassVar[OnboardingState]
    ONBOARDING_STATE_SSH_SETUP: _ClassVar[OnboardingState]
    ONBOARDING_STATE_PUSHING_SCRIPT: _ClassVar[OnboardingState]
    ONBOARDING_STATE_BOOTSTRAPPING: _ClassVar[OnboardingState]
    ONBOARDING_STATE_VERIFYING: _ClassVar[OnboardingState]
    ONBOARDING_STATE_SUCCEEDED: _ClassVar[OnboardingState]
    ONBOARDING_STATE_FAILED: _ClassVar[OnboardingState]
    ONBOARDING_STATE_CANCELLED: _ClassVar[OnboardingState]
    ONBOARDING_STATE_PAIRED: _ClassVar[OnboardingState]

class SourceMode(int, metaclass=_enum_type_wrapper.EnumTypeWrapper):
    __slots__ = ()
    SOURCE_MODE_UNSPECIFIED: _ClassVar[SourceMode]
    SOURCE_MODE_PINNED_REVISION: _ClassVar[SourceMode]
    SOURCE_MODE_WORKING_TREE: _ClassVar[SourceMode]

class OnboardingStepStatus(int, metaclass=_enum_type_wrapper.EnumTypeWrapper):
    __slots__ = ()
    ONBOARDING_STEP_STATUS_UNSPECIFIED: _ClassVar[OnboardingStepStatus]
    ONBOARDING_STEP_STATUS_STARTED: _ClassVar[OnboardingStepStatus]
    ONBOARDING_STEP_STATUS_OK: _ClassVar[OnboardingStepStatus]
    ONBOARDING_STEP_STATUS_SKIPPED: _ClassVar[OnboardingStepStatus]
    ONBOARDING_STEP_STATUS_FAILED: _ClassVar[OnboardingStepStatus]

class ConnectDecision(int, metaclass=_enum_type_wrapper.EnumTypeWrapper):
    __slots__ = ()
    CONNECT_DECISION_UNSPECIFIED: _ClassVar[ConnectDecision]
    CONNECT_DECISION_RECONNECT: _ClassVar[ConnectDecision]
    CONNECT_DECISION_FIRST_TOUCH: _ClassVar[ConnectDecision]
    CONNECT_DECISION_RECOVERY_REQUIRED: _ClassVar[ConnectDecision]
    CONNECT_DECISION_AMBIGUOUS: _ClassVar[ConnectDecision]
    CONNECT_DECISION_HOST_KEY_REVIEW: _ClassVar[ConnectDecision]
ONBOARDING_STATE_UNSPECIFIED: OnboardingState
ONBOARDING_STATE_PENDING: OnboardingState
ONBOARDING_STATE_SSH_SETUP: OnboardingState
ONBOARDING_STATE_PUSHING_SCRIPT: OnboardingState
ONBOARDING_STATE_BOOTSTRAPPING: OnboardingState
ONBOARDING_STATE_VERIFYING: OnboardingState
ONBOARDING_STATE_SUCCEEDED: OnboardingState
ONBOARDING_STATE_FAILED: OnboardingState
ONBOARDING_STATE_CANCELLED: OnboardingState
ONBOARDING_STATE_PAIRED: OnboardingState
SOURCE_MODE_UNSPECIFIED: SourceMode
SOURCE_MODE_PINNED_REVISION: SourceMode
SOURCE_MODE_WORKING_TREE: SourceMode
ONBOARDING_STEP_STATUS_UNSPECIFIED: OnboardingStepStatus
ONBOARDING_STEP_STATUS_STARTED: OnboardingStepStatus
ONBOARDING_STEP_STATUS_OK: OnboardingStepStatus
ONBOARDING_STEP_STATUS_SKIPPED: OnboardingStepStatus
ONBOARDING_STEP_STATUS_FAILED: OnboardingStepStatus
CONNECT_DECISION_UNSPECIFIED: ConnectDecision
CONNECT_DECISION_RECONNECT: ConnectDecision
CONNECT_DECISION_FIRST_TOUCH: ConnectDecision
CONNECT_DECISION_RECOVERY_REQUIRED: ConnectDecision
CONNECT_DECISION_AMBIGUOUS: ConnectDecision
CONNECT_DECISION_HOST_KEY_REVIEW: ConnectDecision

class OnboardingOp(_message.Message):
    __slots__ = ("id", "host", "port", "user", "node_name", "target_revision", "repo_url", "state", "node_id", "failure_reason", "exit_code", "created_at", "started_at", "finished_at", "source_mode", "base_revision", "working_tree_digest", "failure_detail", "control_plane_url", "reachability_mode", "machine_id", "enrollment_attempt_id", "configuration_dispositions")
    ID_FIELD_NUMBER: _ClassVar[int]
    HOST_FIELD_NUMBER: _ClassVar[int]
    PORT_FIELD_NUMBER: _ClassVar[int]
    USER_FIELD_NUMBER: _ClassVar[int]
    NODE_NAME_FIELD_NUMBER: _ClassVar[int]
    TARGET_REVISION_FIELD_NUMBER: _ClassVar[int]
    REPO_URL_FIELD_NUMBER: _ClassVar[int]
    STATE_FIELD_NUMBER: _ClassVar[int]
    NODE_ID_FIELD_NUMBER: _ClassVar[int]
    FAILURE_REASON_FIELD_NUMBER: _ClassVar[int]
    EXIT_CODE_FIELD_NUMBER: _ClassVar[int]
    CREATED_AT_FIELD_NUMBER: _ClassVar[int]
    STARTED_AT_FIELD_NUMBER: _ClassVar[int]
    FINISHED_AT_FIELD_NUMBER: _ClassVar[int]
    SOURCE_MODE_FIELD_NUMBER: _ClassVar[int]
    BASE_REVISION_FIELD_NUMBER: _ClassVar[int]
    WORKING_TREE_DIGEST_FIELD_NUMBER: _ClassVar[int]
    FAILURE_DETAIL_FIELD_NUMBER: _ClassVar[int]
    CONTROL_PLANE_URL_FIELD_NUMBER: _ClassVar[int]
    REACHABILITY_MODE_FIELD_NUMBER: _ClassVar[int]
    MACHINE_ID_FIELD_NUMBER: _ClassVar[int]
    ENROLLMENT_ATTEMPT_ID_FIELD_NUMBER: _ClassVar[int]
    CONFIGURATION_DISPOSITIONS_FIELD_NUMBER: _ClassVar[int]
    id: str
    host: str
    port: int
    user: str
    node_name: str
    target_revision: str
    repo_url: str
    state: OnboardingState
    node_id: str
    failure_reason: str
    exit_code: int
    created_at: _timestamp_pb2.Timestamp
    started_at: _timestamp_pb2.Timestamp
    finished_at: _timestamp_pb2.Timestamp
    source_mode: SourceMode
    base_revision: str
    working_tree_digest: str
    failure_detail: str
    control_plane_url: str
    reachability_mode: str
    machine_id: str
    enrollment_attempt_id: str
    configuration_dispositions: _containers.RepeatedCompositeFieldContainer[ConfigurationDisposition]
    def __init__(self, id: _Optional[str] = ..., host: _Optional[str] = ..., port: _Optional[int] = ..., user: _Optional[str] = ..., node_name: _Optional[str] = ..., target_revision: _Optional[str] = ..., repo_url: _Optional[str] = ..., state: _Optional[_Union[OnboardingState, str]] = ..., node_id: _Optional[str] = ..., failure_reason: _Optional[str] = ..., exit_code: _Optional[int] = ..., created_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., started_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., finished_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., source_mode: _Optional[_Union[SourceMode, str]] = ..., base_revision: _Optional[str] = ..., working_tree_digest: _Optional[str] = ..., failure_detail: _Optional[str] = ..., control_plane_url: _Optional[str] = ..., reachability_mode: _Optional[str] = ..., machine_id: _Optional[str] = ..., enrollment_attempt_id: _Optional[str] = ..., configuration_dispositions: _Optional[_Iterable[_Union[ConfigurationDisposition, _Mapping]]] = ...) -> None: ...

class ConfigurationDisposition(_message.Message):
    __slots__ = ("id", "kind", "name", "disposition", "reason", "remediation")
    ID_FIELD_NUMBER: _ClassVar[int]
    KIND_FIELD_NUMBER: _ClassVar[int]
    NAME_FIELD_NUMBER: _ClassVar[int]
    DISPOSITION_FIELD_NUMBER: _ClassVar[int]
    REASON_FIELD_NUMBER: _ClassVar[int]
    REMEDIATION_FIELD_NUMBER: _ClassVar[int]
    id: str
    kind: str
    name: str
    disposition: str
    reason: str
    remediation: str
    def __init__(self, id: _Optional[str] = ..., kind: _Optional[str] = ..., name: _Optional[str] = ..., disposition: _Optional[str] = ..., reason: _Optional[str] = ..., remediation: _Optional[str] = ...) -> None: ...

class OnboardingStepEvent(_message.Message):
    __slots__ = ("op_id", "sequence", "step_id", "status", "detail", "emitted_at")
    OP_ID_FIELD_NUMBER: _ClassVar[int]
    SEQUENCE_FIELD_NUMBER: _ClassVar[int]
    STEP_ID_FIELD_NUMBER: _ClassVar[int]
    STATUS_FIELD_NUMBER: _ClassVar[int]
    DETAIL_FIELD_NUMBER: _ClassVar[int]
    EMITTED_AT_FIELD_NUMBER: _ClassVar[int]
    op_id: str
    sequence: int
    step_id: str
    status: OnboardingStepStatus
    detail: str
    emitted_at: _timestamp_pb2.Timestamp
    def __init__(self, op_id: _Optional[str] = ..., sequence: _Optional[int] = ..., step_id: _Optional[str] = ..., status: _Optional[_Union[OnboardingStepStatus, str]] = ..., detail: _Optional[str] = ..., emitted_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ...) -> None: ...

class PreflightOnboardingRequest(_message.Message):
    __slots__ = ("host", "port", "user", "machine_id")
    HOST_FIELD_NUMBER: _ClassVar[int]
    PORT_FIELD_NUMBER: _ClassVar[int]
    USER_FIELD_NUMBER: _ClassVar[int]
    MACHINE_ID_FIELD_NUMBER: _ClassVar[int]
    host: str
    port: int
    user: str
    machine_id: str
    def __init__(self, host: _Optional[str] = ..., port: _Optional[int] = ..., user: _Optional[str] = ..., machine_id: _Optional[str] = ...) -> None: ...

class PreflightOnboardingResponse(_message.Message):
    __slots__ = ("decision", "machine_id", "host", "port", "user", "client_key_fingerprint", "host_key_fingerprint", "password_required", "message")
    DECISION_FIELD_NUMBER: _ClassVar[int]
    MACHINE_ID_FIELD_NUMBER: _ClassVar[int]
    HOST_FIELD_NUMBER: _ClassVar[int]
    PORT_FIELD_NUMBER: _ClassVar[int]
    USER_FIELD_NUMBER: _ClassVar[int]
    CLIENT_KEY_FINGERPRINT_FIELD_NUMBER: _ClassVar[int]
    HOST_KEY_FINGERPRINT_FIELD_NUMBER: _ClassVar[int]
    PASSWORD_REQUIRED_FIELD_NUMBER: _ClassVar[int]
    MESSAGE_FIELD_NUMBER: _ClassVar[int]
    decision: ConnectDecision
    machine_id: str
    host: str
    port: int
    user: str
    client_key_fingerprint: str
    host_key_fingerprint: str
    password_required: bool
    message: str
    def __init__(self, decision: _Optional[_Union[ConnectDecision, str]] = ..., machine_id: _Optional[str] = ..., host: _Optional[str] = ..., port: _Optional[int] = ..., user: _Optional[str] = ..., client_key_fingerprint: _Optional[str] = ..., host_key_fingerprint: _Optional[str] = ..., password_required: _Optional[bool] = ..., message: _Optional[str] = ...) -> None: ...

class StartOnboardingRequest(_message.Message):
    __slots__ = ("host", "port", "user", "ssh_password", "node_name", "target_revision", "repo_url", "checkout_dir", "control_plane_url", "capabilities", "verify_timeout_seconds", "skip_setup", "skip_prereqs", "provision_sudo", "setup_preset", "source_mode", "selection", "reachability_mode", "machine_id", "retry_of_enrollment_attempt_id", "setup_passphrase", "provision_service_user")
    HOST_FIELD_NUMBER: _ClassVar[int]
    PORT_FIELD_NUMBER: _ClassVar[int]
    USER_FIELD_NUMBER: _ClassVar[int]
    SSH_PASSWORD_FIELD_NUMBER: _ClassVar[int]
    NODE_NAME_FIELD_NUMBER: _ClassVar[int]
    TARGET_REVISION_FIELD_NUMBER: _ClassVar[int]
    REPO_URL_FIELD_NUMBER: _ClassVar[int]
    CHECKOUT_DIR_FIELD_NUMBER: _ClassVar[int]
    CONTROL_PLANE_URL_FIELD_NUMBER: _ClassVar[int]
    CAPABILITIES_FIELD_NUMBER: _ClassVar[int]
    VERIFY_TIMEOUT_SECONDS_FIELD_NUMBER: _ClassVar[int]
    SKIP_SETUP_FIELD_NUMBER: _ClassVar[int]
    SKIP_PREREQS_FIELD_NUMBER: _ClassVar[int]
    PROVISION_SUDO_FIELD_NUMBER: _ClassVar[int]
    SETUP_PRESET_FIELD_NUMBER: _ClassVar[int]
    SOURCE_MODE_FIELD_NUMBER: _ClassVar[int]
    SELECTION_FIELD_NUMBER: _ClassVar[int]
    REACHABILITY_MODE_FIELD_NUMBER: _ClassVar[int]
    MACHINE_ID_FIELD_NUMBER: _ClassVar[int]
    RETRY_OF_ENROLLMENT_ATTEMPT_ID_FIELD_NUMBER: _ClassVar[int]
    SETUP_PASSPHRASE_FIELD_NUMBER: _ClassVar[int]
    PROVISION_SERVICE_USER_FIELD_NUMBER: _ClassVar[int]
    host: str
    port: int
    user: str
    ssh_password: str
    node_name: str
    target_revision: str
    repo_url: str
    checkout_dir: str
    control_plane_url: str
    capabilities: _containers.RepeatedScalarFieldContainer[str]
    verify_timeout_seconds: int
    skip_setup: bool
    skip_prereqs: bool
    provision_sudo: bool
    setup_preset: str
    source_mode: SourceMode
    selection: _selection_pb2.Selection
    reachability_mode: str
    machine_id: str
    retry_of_enrollment_attempt_id: str
    setup_passphrase: str
    provision_service_user: str
    def __init__(self, host: _Optional[str] = ..., port: _Optional[int] = ..., user: _Optional[str] = ..., ssh_password: _Optional[str] = ..., node_name: _Optional[str] = ..., target_revision: _Optional[str] = ..., repo_url: _Optional[str] = ..., checkout_dir: _Optional[str] = ..., control_plane_url: _Optional[str] = ..., capabilities: _Optional[_Iterable[str]] = ..., verify_timeout_seconds: _Optional[int] = ..., skip_setup: _Optional[bool] = ..., skip_prereqs: _Optional[bool] = ..., provision_sudo: _Optional[bool] = ..., setup_preset: _Optional[str] = ..., source_mode: _Optional[_Union[SourceMode, str]] = ..., selection: _Optional[_Union[_selection_pb2.Selection, _Mapping]] = ..., reachability_mode: _Optional[str] = ..., machine_id: _Optional[str] = ..., retry_of_enrollment_attempt_id: _Optional[str] = ..., setup_passphrase: _Optional[str] = ..., provision_service_user: _Optional[str] = ...) -> None: ...

class ProtectOnboardingRequest(_message.Message):
    __slots__ = ("onboarding_op_id", "machine_id", "node_id", "target", "scope", "cleanup_operation_id", "sealed_passphrase", "declined")
    ONBOARDING_OP_ID_FIELD_NUMBER: _ClassVar[int]
    MACHINE_ID_FIELD_NUMBER: _ClassVar[int]
    NODE_ID_FIELD_NUMBER: _ClassVar[int]
    TARGET_FIELD_NUMBER: _ClassVar[int]
    SCOPE_FIELD_NUMBER: _ClassVar[int]
    CLEANUP_OPERATION_ID_FIELD_NUMBER: _ClassVar[int]
    SEALED_PASSPHRASE_FIELD_NUMBER: _ClassVar[int]
    DECLINED_FIELD_NUMBER: _ClassVar[int]
    onboarding_op_id: str
    machine_id: str
    node_id: str
    target: str
    scope: str
    cleanup_operation_id: str
    sealed_passphrase: bytes
    declined: bool
    def __init__(self, onboarding_op_id: _Optional[str] = ..., machine_id: _Optional[str] = ..., node_id: _Optional[str] = ..., target: _Optional[str] = ..., scope: _Optional[str] = ..., cleanup_operation_id: _Optional[str] = ..., sealed_passphrase: _Optional[bytes] = ..., declined: _Optional[bool] = ...) -> None: ...

class ProtectOnboardingResponse(_message.Message):
    __slots__ = ("op", "protection_status", "protection_operation_id", "detail")
    OP_FIELD_NUMBER: _ClassVar[int]
    PROTECTION_STATUS_FIELD_NUMBER: _ClassVar[int]
    PROTECTION_OPERATION_ID_FIELD_NUMBER: _ClassVar[int]
    DETAIL_FIELD_NUMBER: _ClassVar[int]
    op: OnboardingOp
    protection_status: str
    protection_operation_id: str
    detail: str
    def __init__(self, op: _Optional[_Union[OnboardingOp, _Mapping]] = ..., protection_status: _Optional[str] = ..., protection_operation_id: _Optional[str] = ..., detail: _Optional[str] = ...) -> None: ...

class StartOnboardingResponse(_message.Message):
    __slots__ = ("op_id", "dry_run", "host", "port", "user", "machine_id", "enrollment_attempt_id")
    OP_ID_FIELD_NUMBER: _ClassVar[int]
    DRY_RUN_FIELD_NUMBER: _ClassVar[int]
    HOST_FIELD_NUMBER: _ClassVar[int]
    PORT_FIELD_NUMBER: _ClassVar[int]
    USER_FIELD_NUMBER: _ClassVar[int]
    MACHINE_ID_FIELD_NUMBER: _ClassVar[int]
    ENROLLMENT_ATTEMPT_ID_FIELD_NUMBER: _ClassVar[int]
    op_id: str
    dry_run: bool
    host: str
    port: int
    user: str
    machine_id: str
    enrollment_attempt_id: str
    def __init__(self, op_id: _Optional[str] = ..., dry_run: _Optional[bool] = ..., host: _Optional[str] = ..., port: _Optional[int] = ..., user: _Optional[str] = ..., machine_id: _Optional[str] = ..., enrollment_attempt_id: _Optional[str] = ...) -> None: ...

class GetOnboardingRequest(_message.Message):
    __slots__ = ("id",)
    ID_FIELD_NUMBER: _ClassVar[int]
    id: str
    def __init__(self, id: _Optional[str] = ...) -> None: ...

class GetOnboardingResponse(_message.Message):
    __slots__ = ("op", "events")
    OP_FIELD_NUMBER: _ClassVar[int]
    EVENTS_FIELD_NUMBER: _ClassVar[int]
    op: OnboardingOp
    events: _containers.RepeatedCompositeFieldContainer[OnboardingStepEvent]
    def __init__(self, op: _Optional[_Union[OnboardingOp, _Mapping]] = ..., events: _Optional[_Iterable[_Union[OnboardingStepEvent, _Mapping]]] = ...) -> None: ...

class ListOnboardingsRequest(_message.Message):
    __slots__ = ("host", "limit")
    HOST_FIELD_NUMBER: _ClassVar[int]
    LIMIT_FIELD_NUMBER: _ClassVar[int]
    host: str
    limit: int
    def __init__(self, host: _Optional[str] = ..., limit: _Optional[int] = ...) -> None: ...

class ListOnboardingsResponse(_message.Message):
    __slots__ = ("ops",)
    OPS_FIELD_NUMBER: _ClassVar[int]
    ops: _containers.RepeatedCompositeFieldContainer[OnboardingOp]
    def __init__(self, ops: _Optional[_Iterable[_Union[OnboardingOp, _Mapping]]] = ...) -> None: ...

class WaitOnboardingRequest(_message.Message):
    __slots__ = ("id", "timeout_seconds")
    ID_FIELD_NUMBER: _ClassVar[int]
    TIMEOUT_SECONDS_FIELD_NUMBER: _ClassVar[int]
    id: str
    timeout_seconds: int
    def __init__(self, id: _Optional[str] = ..., timeout_seconds: _Optional[int] = ...) -> None: ...

class WaitOnboardingResponse(_message.Message):
    __slots__ = ("op", "timed_out")
    OP_FIELD_NUMBER: _ClassVar[int]
    TIMED_OUT_FIELD_NUMBER: _ClassVar[int]
    op: OnboardingOp
    timed_out: bool
    def __init__(self, op: _Optional[_Union[OnboardingOp, _Mapping]] = ..., timed_out: _Optional[bool] = ...) -> None: ...

class CancelOnboardingRequest(_message.Message):
    __slots__ = ("id",)
    ID_FIELD_NUMBER: _ClassVar[int]
    id: str
    def __init__(self, id: _Optional[str] = ...) -> None: ...

class CancelOnboardingResponse(_message.Message):
    __slots__ = ("op",)
    OP_FIELD_NUMBER: _ClassVar[int]
    op: OnboardingOp
    def __init__(self, op: _Optional[_Union[OnboardingOp, _Mapping]] = ...) -> None: ...

class RemoveFailedOnboardingRequest(_message.Message):
    __slots__ = ("id",)
    ID_FIELD_NUMBER: _ClassVar[int]
    id: str
    def __init__(self, id: _Optional[str] = ...) -> None: ...

class RemoveFailedOnboardingResponse(_message.Message):
    __slots__ = ()
    def __init__(self) -> None: ...
