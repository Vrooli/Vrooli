import datetime

from buf.validate import validate_pb2 as _validate_pb2
from google.protobuf import timestamp_pb2 as _timestamp_pb2
from google.protobuf.internal import containers as _containers
from google.protobuf.internal import enum_type_wrapper as _enum_type_wrapper
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Iterable as _Iterable, Mapping as _Mapping
from typing import ClassVar as _ClassVar, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class BackendKind(int, metaclass=_enum_type_wrapper.EnumTypeWrapper):
    __slots__ = ()
    BACKEND_KIND_UNSPECIFIED: _ClassVar[BackendKind]
    BACKEND_KIND_FILESYSTEM: _ClassVar[BackendKind]
    BACKEND_KIND_S3: _ClassVar[BackendKind]

class CapPolicy(int, metaclass=_enum_type_wrapper.EnumTypeWrapper):
    __slots__ = ()
    CAP_POLICY_UNSPECIFIED: _ClassVar[CapPolicy]
    CAP_POLICY_ALERT_BLOCK: _ClassVar[CapPolicy]
    CAP_POLICY_ALERT_ONLY: _ClassVar[CapPolicy]

class UsageState(int, metaclass=_enum_type_wrapper.EnumTypeWrapper):
    __slots__ = ()
    USAGE_STATE_UNSPECIFIED: _ClassVar[UsageState]
    USAGE_STATE_WITHIN: _ClassVar[UsageState]
    USAGE_STATE_NEAR: _ClassVar[UsageState]
    USAGE_STATE_OVER: _ClassVar[UsageState]

class ReadinessSeverity(int, metaclass=_enum_type_wrapper.EnumTypeWrapper):
    __slots__ = ()
    READINESS_SEVERITY_UNSPECIFIED: _ClassVar[ReadinessSeverity]
    READINESS_SEVERITY_PASS: _ClassVar[ReadinessSeverity]
    READINESS_SEVERITY_WARNING: _ClassVar[ReadinessSeverity]
    READINESS_SEVERITY_FAIL: _ClassVar[ReadinessSeverity]
    READINESS_SEVERITY_UNKNOWN: _ClassVar[ReadinessSeverity]

class PreparationAction(int, metaclass=_enum_type_wrapper.EnumTypeWrapper):
    __slots__ = ()
    PREPARATION_ACTION_UNSPECIFIED: _ClassVar[PreparationAction]
    PREPARATION_ACTION_CREATE_SUBDIR: _ClassVar[PreparationAction]
    PREPARATION_ACTION_RELABEL: _ClassVar[PreparationAction]
    PREPARATION_ACTION_CLEAR_DIRECTORY: _ClassVar[PreparationAction]
    PREPARATION_ACTION_FORMAT: _ClassVar[PreparationAction]
BACKEND_KIND_UNSPECIFIED: BackendKind
BACKEND_KIND_FILESYSTEM: BackendKind
BACKEND_KIND_S3: BackendKind
CAP_POLICY_UNSPECIFIED: CapPolicy
CAP_POLICY_ALERT_BLOCK: CapPolicy
CAP_POLICY_ALERT_ONLY: CapPolicy
USAGE_STATE_UNSPECIFIED: UsageState
USAGE_STATE_WITHIN: UsageState
USAGE_STATE_NEAR: UsageState
USAGE_STATE_OVER: UsageState
READINESS_SEVERITY_UNSPECIFIED: ReadinessSeverity
READINESS_SEVERITY_PASS: ReadinessSeverity
READINESS_SEVERITY_WARNING: ReadinessSeverity
READINESS_SEVERITY_FAIL: ReadinessSeverity
READINESS_SEVERITY_UNKNOWN: ReadinessSeverity
PREPARATION_ACTION_UNSPECIFIED: PreparationAction
PREPARATION_ACTION_CREATE_SUBDIR: PreparationAction
PREPARATION_ACTION_RELABEL: PreparationAction
PREPARATION_ACTION_CLEAR_DIRECTORY: PreparationAction
PREPARATION_ACTION_FORMAT: PreparationAction

class Destination(_message.Message):
    __slots__ = ("id", "name", "backend_kind", "location", "cap_bytes", "cap_policy", "encryption_algorithm", "secret_ref", "usage_bytes", "usage_state", "created_at", "updated_at", "repository_location")
    ID_FIELD_NUMBER: _ClassVar[int]
    NAME_FIELD_NUMBER: _ClassVar[int]
    BACKEND_KIND_FIELD_NUMBER: _ClassVar[int]
    LOCATION_FIELD_NUMBER: _ClassVar[int]
    CAP_BYTES_FIELD_NUMBER: _ClassVar[int]
    CAP_POLICY_FIELD_NUMBER: _ClassVar[int]
    ENCRYPTION_ALGORITHM_FIELD_NUMBER: _ClassVar[int]
    SECRET_REF_FIELD_NUMBER: _ClassVar[int]
    USAGE_BYTES_FIELD_NUMBER: _ClassVar[int]
    USAGE_STATE_FIELD_NUMBER: _ClassVar[int]
    CREATED_AT_FIELD_NUMBER: _ClassVar[int]
    UPDATED_AT_FIELD_NUMBER: _ClassVar[int]
    REPOSITORY_LOCATION_FIELD_NUMBER: _ClassVar[int]
    id: str
    name: str
    backend_kind: BackendKind
    location: str
    cap_bytes: int
    cap_policy: CapPolicy
    encryption_algorithm: str
    secret_ref: str
    usage_bytes: int
    usage_state: UsageState
    created_at: _timestamp_pb2.Timestamp
    updated_at: _timestamp_pb2.Timestamp
    repository_location: str
    def __init__(self, id: _Optional[str] = ..., name: _Optional[str] = ..., backend_kind: _Optional[_Union[BackendKind, str]] = ..., location: _Optional[str] = ..., cap_bytes: _Optional[int] = ..., cap_policy: _Optional[_Union[CapPolicy, str]] = ..., encryption_algorithm: _Optional[str] = ..., secret_ref: _Optional[str] = ..., usage_bytes: _Optional[int] = ..., usage_state: _Optional[_Union[UsageState, str]] = ..., created_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., updated_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., repository_location: _Optional[str] = ...) -> None: ...

class CreateDestinationRequest(_message.Message):
    __slots__ = ("name", "backend_kind", "location", "cap_bytes", "cap_policy")
    NAME_FIELD_NUMBER: _ClassVar[int]
    BACKEND_KIND_FIELD_NUMBER: _ClassVar[int]
    LOCATION_FIELD_NUMBER: _ClassVar[int]
    CAP_BYTES_FIELD_NUMBER: _ClassVar[int]
    CAP_POLICY_FIELD_NUMBER: _ClassVar[int]
    name: str
    backend_kind: BackendKind
    location: str
    cap_bytes: int
    cap_policy: CapPolicy
    def __init__(self, name: _Optional[str] = ..., backend_kind: _Optional[_Union[BackendKind, str]] = ..., location: _Optional[str] = ..., cap_bytes: _Optional[int] = ..., cap_policy: _Optional[_Union[CapPolicy, str]] = ...) -> None: ...

class CreateDestinationResponse(_message.Message):
    __slots__ = ("destination",)
    DESTINATION_FIELD_NUMBER: _ClassVar[int]
    destination: Destination
    def __init__(self, destination: _Optional[_Union[Destination, _Mapping]] = ...) -> None: ...

class GetDestinationRequest(_message.Message):
    __slots__ = ("id",)
    ID_FIELD_NUMBER: _ClassVar[int]
    id: str
    def __init__(self, id: _Optional[str] = ...) -> None: ...

class GetDestinationResponse(_message.Message):
    __slots__ = ("destination",)
    DESTINATION_FIELD_NUMBER: _ClassVar[int]
    destination: Destination
    def __init__(self, destination: _Optional[_Union[Destination, _Mapping]] = ...) -> None: ...

class ListDestinationsRequest(_message.Message):
    __slots__ = ("page_size", "page_token")
    PAGE_SIZE_FIELD_NUMBER: _ClassVar[int]
    PAGE_TOKEN_FIELD_NUMBER: _ClassVar[int]
    page_size: int
    page_token: str
    def __init__(self, page_size: _Optional[int] = ..., page_token: _Optional[str] = ...) -> None: ...

class ListDestinationsResponse(_message.Message):
    __slots__ = ("destinations", "next_page_token")
    DESTINATIONS_FIELD_NUMBER: _ClassVar[int]
    NEXT_PAGE_TOKEN_FIELD_NUMBER: _ClassVar[int]
    destinations: _containers.RepeatedCompositeFieldContainer[Destination]
    next_page_token: str
    def __init__(self, destinations: _Optional[_Iterable[_Union[Destination, _Mapping]]] = ..., next_page_token: _Optional[str] = ...) -> None: ...

class UpdateDestinationRequest(_message.Message):
    __slots__ = ("id", "cap_bytes", "cap_policy")
    ID_FIELD_NUMBER: _ClassVar[int]
    CAP_BYTES_FIELD_NUMBER: _ClassVar[int]
    CAP_POLICY_FIELD_NUMBER: _ClassVar[int]
    id: str
    cap_bytes: int
    cap_policy: CapPolicy
    def __init__(self, id: _Optional[str] = ..., cap_bytes: _Optional[int] = ..., cap_policy: _Optional[_Union[CapPolicy, str]] = ...) -> None: ...

class UpdateDestinationResponse(_message.Message):
    __slots__ = ("destination",)
    DESTINATION_FIELD_NUMBER: _ClassVar[int]
    destination: Destination
    def __init__(self, destination: _Optional[_Union[Destination, _Mapping]] = ...) -> None: ...

class DeleteDestinationRequest(_message.Message):
    __slots__ = ("id", "delete_repository")
    ID_FIELD_NUMBER: _ClassVar[int]
    DELETE_REPOSITORY_FIELD_NUMBER: _ClassVar[int]
    id: str
    delete_repository: bool
    def __init__(self, id: _Optional[str] = ..., delete_repository: _Optional[bool] = ...) -> None: ...

class DeleteDestinationResponse(_message.Message):
    __slots__ = ("removed",)
    REMOVED_FIELD_NUMBER: _ClassVar[int]
    removed: bool
    def __init__(self, removed: _Optional[bool] = ...) -> None: ...

class GetDestinationUsageRequest(_message.Message):
    __slots__ = ("id",)
    ID_FIELD_NUMBER: _ClassVar[int]
    id: str
    def __init__(self, id: _Optional[str] = ...) -> None: ...

class GetDestinationUsageResponse(_message.Message):
    __slots__ = ("usage_bytes", "cap_bytes", "usage_state", "cap_policy")
    USAGE_BYTES_FIELD_NUMBER: _ClassVar[int]
    CAP_BYTES_FIELD_NUMBER: _ClassVar[int]
    USAGE_STATE_FIELD_NUMBER: _ClassVar[int]
    CAP_POLICY_FIELD_NUMBER: _ClassVar[int]
    usage_bytes: int
    cap_bytes: int
    usage_state: UsageState
    cap_policy: CapPolicy
    def __init__(self, usage_bytes: _Optional[int] = ..., cap_bytes: _Optional[int] = ..., usage_state: _Optional[_Union[UsageState, str]] = ..., cap_policy: _Optional[_Union[CapPolicy, str]] = ...) -> None: ...

class DestinationDeviceIdentity(_message.Message):
    __slots__ = ("device_path", "mountpoint", "label", "filesystem", "total_bytes", "model", "serial", "uuid")
    DEVICE_PATH_FIELD_NUMBER: _ClassVar[int]
    MOUNTPOINT_FIELD_NUMBER: _ClassVar[int]
    LABEL_FIELD_NUMBER: _ClassVar[int]
    FILESYSTEM_FIELD_NUMBER: _ClassVar[int]
    TOTAL_BYTES_FIELD_NUMBER: _ClassVar[int]
    MODEL_FIELD_NUMBER: _ClassVar[int]
    SERIAL_FIELD_NUMBER: _ClassVar[int]
    UUID_FIELD_NUMBER: _ClassVar[int]
    device_path: str
    mountpoint: str
    label: str
    filesystem: str
    total_bytes: int
    model: str
    serial: str
    uuid: str
    def __init__(self, device_path: _Optional[str] = ..., mountpoint: _Optional[str] = ..., label: _Optional[str] = ..., filesystem: _Optional[str] = ..., total_bytes: _Optional[int] = ..., model: _Optional[str] = ..., serial: _Optional[str] = ..., uuid: _Optional[str] = ...) -> None: ...

class DestinationReadinessCheck(_message.Message):
    __slots__ = ("code", "severity", "message", "next_action")
    CODE_FIELD_NUMBER: _ClassVar[int]
    SEVERITY_FIELD_NUMBER: _ClassVar[int]
    MESSAGE_FIELD_NUMBER: _ClassVar[int]
    NEXT_ACTION_FIELD_NUMBER: _ClassVar[int]
    code: str
    severity: ReadinessSeverity
    message: str
    next_action: str
    def __init__(self, code: _Optional[str] = ..., severity: _Optional[_Union[ReadinessSeverity, str]] = ..., message: _Optional[str] = ..., next_action: _Optional[str] = ...) -> None: ...

class DestinationReadinessReport(_message.Message):
    __slots__ = ("location", "overall_severity", "identity", "checks", "recommended_destination_location", "recommended_action", "platform", "confidence", "evidence_source", "observed_at", "repair_steps")
    LOCATION_FIELD_NUMBER: _ClassVar[int]
    OVERALL_SEVERITY_FIELD_NUMBER: _ClassVar[int]
    IDENTITY_FIELD_NUMBER: _ClassVar[int]
    CHECKS_FIELD_NUMBER: _ClassVar[int]
    RECOMMENDED_DESTINATION_LOCATION_FIELD_NUMBER: _ClassVar[int]
    RECOMMENDED_ACTION_FIELD_NUMBER: _ClassVar[int]
    PLATFORM_FIELD_NUMBER: _ClassVar[int]
    CONFIDENCE_FIELD_NUMBER: _ClassVar[int]
    EVIDENCE_SOURCE_FIELD_NUMBER: _ClassVar[int]
    OBSERVED_AT_FIELD_NUMBER: _ClassVar[int]
    REPAIR_STEPS_FIELD_NUMBER: _ClassVar[int]
    location: str
    overall_severity: ReadinessSeverity
    identity: DestinationDeviceIdentity
    checks: _containers.RepeatedCompositeFieldContainer[DestinationReadinessCheck]
    recommended_destination_location: str
    recommended_action: str
    platform: str
    confidence: str
    evidence_source: str
    observed_at: _timestamp_pb2.Timestamp
    repair_steps: _containers.RepeatedScalarFieldContainer[str]
    def __init__(self, location: _Optional[str] = ..., overall_severity: _Optional[_Union[ReadinessSeverity, str]] = ..., identity: _Optional[_Union[DestinationDeviceIdentity, _Mapping]] = ..., checks: _Optional[_Iterable[_Union[DestinationReadinessCheck, _Mapping]]] = ..., recommended_destination_location: _Optional[str] = ..., recommended_action: _Optional[str] = ..., platform: _Optional[str] = ..., confidence: _Optional[str] = ..., evidence_source: _Optional[str] = ..., observed_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., repair_steps: _Optional[_Iterable[str]] = ...) -> None: ...

class AnalyzeDestinationRequest(_message.Message):
    __slots__ = ("location", "proposed_subdir", "selected_target_bytes", "retention_copies", "cross_platform_required")
    LOCATION_FIELD_NUMBER: _ClassVar[int]
    PROPOSED_SUBDIR_FIELD_NUMBER: _ClassVar[int]
    SELECTED_TARGET_BYTES_FIELD_NUMBER: _ClassVar[int]
    RETENTION_COPIES_FIELD_NUMBER: _ClassVar[int]
    CROSS_PLATFORM_REQUIRED_FIELD_NUMBER: _ClassVar[int]
    location: str
    proposed_subdir: str
    selected_target_bytes: int
    retention_copies: int
    cross_platform_required: bool
    def __init__(self, location: _Optional[str] = ..., proposed_subdir: _Optional[str] = ..., selected_target_bytes: _Optional[int] = ..., retention_copies: _Optional[int] = ..., cross_platform_required: _Optional[bool] = ...) -> None: ...

class AnalyzeDestinationResponse(_message.Message):
    __slots__ = ("report",)
    REPORT_FIELD_NUMBER: _ClassVar[int]
    report: DestinationReadinessReport
    def __init__(self, report: _Optional[_Union[DestinationReadinessReport, _Mapping]] = ...) -> None: ...

class DestinationPreparationPlan(_message.Message):
    __slots__ = ("id", "action", "location", "target_path", "identity", "desired_label", "desired_filesystem", "requires_confirmation", "destructive", "confirmation_phrase", "supported", "unsupported_reason")
    ID_FIELD_NUMBER: _ClassVar[int]
    ACTION_FIELD_NUMBER: _ClassVar[int]
    LOCATION_FIELD_NUMBER: _ClassVar[int]
    TARGET_PATH_FIELD_NUMBER: _ClassVar[int]
    IDENTITY_FIELD_NUMBER: _ClassVar[int]
    DESIRED_LABEL_FIELD_NUMBER: _ClassVar[int]
    DESIRED_FILESYSTEM_FIELD_NUMBER: _ClassVar[int]
    REQUIRES_CONFIRMATION_FIELD_NUMBER: _ClassVar[int]
    DESTRUCTIVE_FIELD_NUMBER: _ClassVar[int]
    CONFIRMATION_PHRASE_FIELD_NUMBER: _ClassVar[int]
    SUPPORTED_FIELD_NUMBER: _ClassVar[int]
    UNSUPPORTED_REASON_FIELD_NUMBER: _ClassVar[int]
    id: str
    action: PreparationAction
    location: str
    target_path: str
    identity: DestinationDeviceIdentity
    desired_label: str
    desired_filesystem: str
    requires_confirmation: bool
    destructive: bool
    confirmation_phrase: str
    supported: bool
    unsupported_reason: str
    def __init__(self, id: _Optional[str] = ..., action: _Optional[_Union[PreparationAction, str]] = ..., location: _Optional[str] = ..., target_path: _Optional[str] = ..., identity: _Optional[_Union[DestinationDeviceIdentity, _Mapping]] = ..., desired_label: _Optional[str] = ..., desired_filesystem: _Optional[str] = ..., requires_confirmation: _Optional[bool] = ..., destructive: _Optional[bool] = ..., confirmation_phrase: _Optional[str] = ..., supported: _Optional[bool] = ..., unsupported_reason: _Optional[str] = ...) -> None: ...

class PlanDestinationPreparationRequest(_message.Message):
    __slots__ = ("location", "action", "desired_subdir", "desired_label", "desired_filesystem", "expected_identity")
    LOCATION_FIELD_NUMBER: _ClassVar[int]
    ACTION_FIELD_NUMBER: _ClassVar[int]
    DESIRED_SUBDIR_FIELD_NUMBER: _ClassVar[int]
    DESIRED_LABEL_FIELD_NUMBER: _ClassVar[int]
    DESIRED_FILESYSTEM_FIELD_NUMBER: _ClassVar[int]
    EXPECTED_IDENTITY_FIELD_NUMBER: _ClassVar[int]
    location: str
    action: PreparationAction
    desired_subdir: str
    desired_label: str
    desired_filesystem: str
    expected_identity: DestinationDeviceIdentity
    def __init__(self, location: _Optional[str] = ..., action: _Optional[_Union[PreparationAction, str]] = ..., desired_subdir: _Optional[str] = ..., desired_label: _Optional[str] = ..., desired_filesystem: _Optional[str] = ..., expected_identity: _Optional[_Union[DestinationDeviceIdentity, _Mapping]] = ...) -> None: ...

class PlanDestinationPreparationResponse(_message.Message):
    __slots__ = ("plan",)
    PLAN_FIELD_NUMBER: _ClassVar[int]
    plan: DestinationPreparationPlan
    def __init__(self, plan: _Optional[_Union[DestinationPreparationPlan, _Mapping]] = ...) -> None: ...

class ExecuteDestinationPreparationRequest(_message.Message):
    __slots__ = ("plan", "confirmation", "dry_run", "acknowledge_data_loss")
    PLAN_FIELD_NUMBER: _ClassVar[int]
    CONFIRMATION_FIELD_NUMBER: _ClassVar[int]
    DRY_RUN_FIELD_NUMBER: _ClassVar[int]
    ACKNOWLEDGE_DATA_LOSS_FIELD_NUMBER: _ClassVar[int]
    plan: DestinationPreparationPlan
    confirmation: str
    dry_run: bool
    acknowledge_data_loss: bool
    def __init__(self, plan: _Optional[_Union[DestinationPreparationPlan, _Mapping]] = ..., confirmation: _Optional[str] = ..., dry_run: _Optional[bool] = ..., acknowledge_data_loss: _Optional[bool] = ...) -> None: ...

class ExecuteDestinationPreparationResponse(_message.Message):
    __slots__ = ("dry_run", "action", "location", "post_action_report")
    DRY_RUN_FIELD_NUMBER: _ClassVar[int]
    ACTION_FIELD_NUMBER: _ClassVar[int]
    LOCATION_FIELD_NUMBER: _ClassVar[int]
    POST_ACTION_REPORT_FIELD_NUMBER: _ClassVar[int]
    dry_run: bool
    action: PreparationAction
    location: str
    post_action_report: DestinationReadinessReport
    def __init__(self, dry_run: _Optional[bool] = ..., action: _Optional[_Union[PreparationAction, str]] = ..., location: _Optional[str] = ..., post_action_report: _Optional[_Union[DestinationReadinessReport, _Mapping]] = ...) -> None: ...
