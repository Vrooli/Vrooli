import datetime

from google.protobuf import timestamp_pb2 as _timestamp_pb2
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
ONBOARDING_STATE_UNSPECIFIED: OnboardingState
ONBOARDING_STATE_PENDING: OnboardingState
ONBOARDING_STATE_SSH_SETUP: OnboardingState
ONBOARDING_STATE_PUSHING_SCRIPT: OnboardingState
ONBOARDING_STATE_BOOTSTRAPPING: OnboardingState
ONBOARDING_STATE_VERIFYING: OnboardingState
ONBOARDING_STATE_SUCCEEDED: OnboardingState
ONBOARDING_STATE_FAILED: OnboardingState
ONBOARDING_STATE_CANCELLED: OnboardingState
SOURCE_MODE_UNSPECIFIED: SourceMode
SOURCE_MODE_PINNED_REVISION: SourceMode
SOURCE_MODE_WORKING_TREE: SourceMode
ONBOARDING_STEP_STATUS_UNSPECIFIED: OnboardingStepStatus
ONBOARDING_STEP_STATUS_STARTED: OnboardingStepStatus
ONBOARDING_STEP_STATUS_OK: OnboardingStepStatus
ONBOARDING_STEP_STATUS_SKIPPED: OnboardingStepStatus
ONBOARDING_STEP_STATUS_FAILED: OnboardingStepStatus

class OnboardingOp(_message.Message):
    __slots__ = ("id", "host", "port", "user", "node_name", "target_revision", "repo_url", "state", "node_id", "failure_reason", "exit_code", "created_at", "started_at", "finished_at", "source_mode", "base_revision", "working_tree_digest", "failure_detail")
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
    def __init__(self, id: _Optional[str] = ..., host: _Optional[str] = ..., port: _Optional[int] = ..., user: _Optional[str] = ..., node_name: _Optional[str] = ..., target_revision: _Optional[str] = ..., repo_url: _Optional[str] = ..., state: _Optional[_Union[OnboardingState, str]] = ..., node_id: _Optional[str] = ..., failure_reason: _Optional[str] = ..., exit_code: _Optional[int] = ..., created_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., started_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., finished_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., source_mode: _Optional[_Union[SourceMode, str]] = ..., base_revision: _Optional[str] = ..., working_tree_digest: _Optional[str] = ..., failure_detail: _Optional[str] = ...) -> None: ...

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

class StartOnboardingRequest(_message.Message):
    __slots__ = ("host", "port", "user", "ssh_password", "node_name", "target_revision", "repo_url", "checkout_dir", "control_plane_url", "capabilities", "verify_timeout_seconds", "skip_setup", "skip_prereqs", "provision_sudo", "setup_environment", "setup_resources", "setup_scenarios", "include_optional", "source_mode")
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
    SETUP_ENVIRONMENT_FIELD_NUMBER: _ClassVar[int]
    SETUP_RESOURCES_FIELD_NUMBER: _ClassVar[int]
    SETUP_SCENARIOS_FIELD_NUMBER: _ClassVar[int]
    INCLUDE_OPTIONAL_FIELD_NUMBER: _ClassVar[int]
    SOURCE_MODE_FIELD_NUMBER: _ClassVar[int]
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
    setup_environment: str
    setup_resources: str
    setup_scenarios: str
    include_optional: bool
    source_mode: SourceMode
    def __init__(self, host: _Optional[str] = ..., port: _Optional[int] = ..., user: _Optional[str] = ..., ssh_password: _Optional[str] = ..., node_name: _Optional[str] = ..., target_revision: _Optional[str] = ..., repo_url: _Optional[str] = ..., checkout_dir: _Optional[str] = ..., control_plane_url: _Optional[str] = ..., capabilities: _Optional[_Iterable[str]] = ..., verify_timeout_seconds: _Optional[int] = ..., skip_setup: _Optional[bool] = ..., skip_prereqs: _Optional[bool] = ..., provision_sudo: _Optional[bool] = ..., setup_environment: _Optional[str] = ..., setup_resources: _Optional[str] = ..., setup_scenarios: _Optional[str] = ..., include_optional: _Optional[bool] = ..., source_mode: _Optional[_Union[SourceMode, str]] = ...) -> None: ...

class StartOnboardingResponse(_message.Message):
    __slots__ = ("op_id", "dry_run", "host", "port", "user")
    OP_ID_FIELD_NUMBER: _ClassVar[int]
    DRY_RUN_FIELD_NUMBER: _ClassVar[int]
    HOST_FIELD_NUMBER: _ClassVar[int]
    PORT_FIELD_NUMBER: _ClassVar[int]
    USER_FIELD_NUMBER: _ClassVar[int]
    op_id: str
    dry_run: bool
    host: str
    port: int
    user: str
    def __init__(self, op_id: _Optional[str] = ..., dry_run: _Optional[bool] = ..., host: _Optional[str] = ..., port: _Optional[int] = ..., user: _Optional[str] = ...) -> None: ...

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
