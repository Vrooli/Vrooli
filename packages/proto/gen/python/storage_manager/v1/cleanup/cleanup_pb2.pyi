import datetime

from google.protobuf import timestamp_pb2 as _timestamp_pb2
from google.protobuf.internal import containers as _containers
from google.protobuf.internal import enum_type_wrapper as _enum_type_wrapper
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Iterable as _Iterable, Mapping as _Mapping
from typing import ClassVar as _ClassVar, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class SafetyTier(int, metaclass=_enum_type_wrapper.EnumTypeWrapper):
    __slots__ = ()
    SAFETY_TIER_UNSPECIFIED: _ClassVar[SafetyTier]
    SAFETY_TIER_SAFE: _ClassVar[SafetyTier]
    SAFETY_TIER_REGENERABLE: _ClassVar[SafetyTier]
    SAFETY_TIER_SAFE_WITH_OWNER: _ClassVar[SafetyTier]
    SAFETY_TIER_CONDITIONAL: _ClassVar[SafetyTier]
    SAFETY_TIER_FORBIDDEN: _ClassVar[SafetyTier]

class PressureBand(int, metaclass=_enum_type_wrapper.EnumTypeWrapper):
    __slots__ = ()
    PRESSURE_BAND_UNSPECIFIED: _ClassVar[PressureBand]
    PRESSURE_BAND_WARNING: _ClassVar[PressureBand]
    PRESSURE_BAND_HIGH: _ClassVar[PressureBand]
    PRESSURE_BAND_CRITICAL: _ClassVar[PressureBand]

class PressureTrigger(int, metaclass=_enum_type_wrapper.EnumTypeWrapper):
    __slots__ = ()
    PRESSURE_TRIGGER_UNSPECIFIED: _ClassVar[PressureTrigger]
    PRESSURE_TRIGGER_BAND: _ClassVar[PressureTrigger]
    PRESSURE_TRIGGER_FLOOR: _ClassVar[PressureTrigger]
    PRESSURE_TRIGGER_RATE: _ClassVar[PressureTrigger]
    PRESSURE_TRIGGER_MANUAL: _ClassVar[PressureTrigger]

class PressureAction(int, metaclass=_enum_type_wrapper.EnumTypeWrapper):
    __slots__ = ()
    PRESSURE_ACTION_UNSPECIFIED: _ClassVar[PressureAction]
    PRESSURE_ACTION_OBSERVED: _ClassVar[PressureAction]
    PRESSURE_ACTION_PREVIEWED: _ClassVar[PressureAction]
    PRESSURE_ACTION_APPLIED: _ClassVar[PressureAction]
    PRESSURE_ACTION_DEDUPLICATED: _ClassVar[PressureAction]
    PRESSURE_ACTION_SUPPRESSED: _ClassVar[PressureAction]
SAFETY_TIER_UNSPECIFIED: SafetyTier
SAFETY_TIER_SAFE: SafetyTier
SAFETY_TIER_REGENERABLE: SafetyTier
SAFETY_TIER_SAFE_WITH_OWNER: SafetyTier
SAFETY_TIER_CONDITIONAL: SafetyTier
SAFETY_TIER_FORBIDDEN: SafetyTier
PRESSURE_BAND_UNSPECIFIED: PressureBand
PRESSURE_BAND_WARNING: PressureBand
PRESSURE_BAND_HIGH: PressureBand
PRESSURE_BAND_CRITICAL: PressureBand
PRESSURE_TRIGGER_UNSPECIFIED: PressureTrigger
PRESSURE_TRIGGER_BAND: PressureTrigger
PRESSURE_TRIGGER_FLOOR: PressureTrigger
PRESSURE_TRIGGER_RATE: PressureTrigger
PRESSURE_TRIGGER_MANUAL: PressureTrigger
PRESSURE_ACTION_UNSPECIFIED: PressureAction
PRESSURE_ACTION_OBSERVED: PressureAction
PRESSURE_ACTION_PREVIEWED: PressureAction
PRESSURE_ACTION_APPLIED: PressureAction
PRESSURE_ACTION_DEDUPLICATED: PressureAction
PRESSURE_ACTION_SUPPRESSED: PressureAction

class ListProvidersRequest(_message.Message):
    __slots__ = ()
    def __init__(self) -> None: ...

class ListProvidersResponse(_message.Message):
    __slots__ = ("providers",)
    PROVIDERS_FIELD_NUMBER: _ClassVar[int]
    providers: _containers.RepeatedCompositeFieldContainer[Provider]
    def __init__(self, providers: _Optional[_Iterable[_Union[Provider, _Mapping]]] = ...) -> None: ...

class Provider(_message.Message):
    __slots__ = ("id", "name", "version", "owner_scenario", "safety_tier", "default_mode", "default_approval", "supported_platforms", "required_privileges", "irreversible_effects")
    ID_FIELD_NUMBER: _ClassVar[int]
    NAME_FIELD_NUMBER: _ClassVar[int]
    VERSION_FIELD_NUMBER: _ClassVar[int]
    OWNER_SCENARIO_FIELD_NUMBER: _ClassVar[int]
    SAFETY_TIER_FIELD_NUMBER: _ClassVar[int]
    DEFAULT_MODE_FIELD_NUMBER: _ClassVar[int]
    DEFAULT_APPROVAL_FIELD_NUMBER: _ClassVar[int]
    SUPPORTED_PLATFORMS_FIELD_NUMBER: _ClassVar[int]
    REQUIRED_PRIVILEGES_FIELD_NUMBER: _ClassVar[int]
    IRREVERSIBLE_EFFECTS_FIELD_NUMBER: _ClassVar[int]
    id: str
    name: str
    version: str
    owner_scenario: str
    safety_tier: str
    default_mode: str
    default_approval: str
    supported_platforms: _containers.RepeatedScalarFieldContainer[str]
    required_privileges: _containers.RepeatedScalarFieldContainer[str]
    irreversible_effects: _containers.RepeatedScalarFieldContainer[str]
    def __init__(self, id: _Optional[str] = ..., name: _Optional[str] = ..., version: _Optional[str] = ..., owner_scenario: _Optional[str] = ..., safety_tier: _Optional[str] = ..., default_mode: _Optional[str] = ..., default_approval: _Optional[str] = ..., supported_platforms: _Optional[_Iterable[str]] = ..., required_privileges: _Optional[_Iterable[str]] = ..., irreversible_effects: _Optional[_Iterable[str]] = ...) -> None: ...

class GetPolicyRequest(_message.Message):
    __slots__ = ()
    def __init__(self) -> None: ...

class GetPolicyResponse(_message.Message):
    __slots__ = ("policy",)
    POLICY_FIELD_NUMBER: _ClassVar[int]
    policy: Policy
    def __init__(self, policy: _Optional[_Union[Policy, _Mapping]] = ...) -> None: ...

class SetPolicyProfileRequest(_message.Message):
    __slots__ = ("profile",)
    PROFILE_FIELD_NUMBER: _ClassVar[int]
    profile: str
    def __init__(self, profile: _Optional[str] = ...) -> None: ...

class SetPolicyProfileResponse(_message.Message):
    __slots__ = ("policy",)
    POLICY_FIELD_NUMBER: _ClassVar[int]
    policy: Policy
    def __init__(self, policy: _Optional[_Union[Policy, _Mapping]] = ...) -> None: ...

class Policy(_message.Message):
    __slots__ = ("version", "profile", "created_at", "providers")
    VERSION_FIELD_NUMBER: _ClassVar[int]
    PROFILE_FIELD_NUMBER: _ClassVar[int]
    CREATED_AT_FIELD_NUMBER: _ClassVar[int]
    PROVIDERS_FIELD_NUMBER: _ClassVar[int]
    version: str
    profile: str
    created_at: _timestamp_pb2.Timestamp
    providers: _containers.RepeatedCompositeFieldContainer[ProviderPolicy]
    def __init__(self, version: _Optional[str] = ..., profile: _Optional[str] = ..., created_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., providers: _Optional[_Iterable[_Union[ProviderPolicy, _Mapping]]] = ...) -> None: ...

class ProviderPolicy(_message.Message):
    __slots__ = ("provider_id", "enabled", "min_age_seconds", "max_bytes", "approval_mode")
    PROVIDER_ID_FIELD_NUMBER: _ClassVar[int]
    ENABLED_FIELD_NUMBER: _ClassVar[int]
    MIN_AGE_SECONDS_FIELD_NUMBER: _ClassVar[int]
    MAX_BYTES_FIELD_NUMBER: _ClassVar[int]
    APPROVAL_MODE_FIELD_NUMBER: _ClassVar[int]
    provider_id: str
    enabled: bool
    min_age_seconds: int
    max_bytes: int
    approval_mode: str
    def __init__(self, provider_id: _Optional[str] = ..., enabled: _Optional[bool] = ..., min_age_seconds: _Optional[int] = ..., max_bytes: _Optional[int] = ..., approval_mode: _Optional[str] = ...) -> None: ...

class CreatePlanRequest(_message.Message):
    __slots__ = ()
    def __init__(self) -> None: ...

class CreatePlanResponse(_message.Message):
    __slots__ = ("plan",)
    PLAN_FIELD_NUMBER: _ClassVar[int]
    plan: Plan
    def __init__(self, plan: _Optional[_Union[Plan, _Mapping]] = ...) -> None: ...

class Plan(_message.Message):
    __slots__ = ("id", "policy_version", "created_at", "total_bytes", "total_items", "providers", "census_id", "census_status", "census_started_at", "census_completed_at")
    ID_FIELD_NUMBER: _ClassVar[int]
    POLICY_VERSION_FIELD_NUMBER: _ClassVar[int]
    CREATED_AT_FIELD_NUMBER: _ClassVar[int]
    TOTAL_BYTES_FIELD_NUMBER: _ClassVar[int]
    TOTAL_ITEMS_FIELD_NUMBER: _ClassVar[int]
    PROVIDERS_FIELD_NUMBER: _ClassVar[int]
    CENSUS_ID_FIELD_NUMBER: _ClassVar[int]
    CENSUS_STATUS_FIELD_NUMBER: _ClassVar[int]
    CENSUS_STARTED_AT_FIELD_NUMBER: _ClassVar[int]
    CENSUS_COMPLETED_AT_FIELD_NUMBER: _ClassVar[int]
    id: str
    policy_version: str
    created_at: _timestamp_pb2.Timestamp
    total_bytes: int
    total_items: int
    providers: _containers.RepeatedCompositeFieldContainer[ProviderPlan]
    census_id: str
    census_status: str
    census_started_at: _timestamp_pb2.Timestamp
    census_completed_at: _timestamp_pb2.Timestamp
    def __init__(self, id: _Optional[str] = ..., policy_version: _Optional[str] = ..., created_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., total_bytes: _Optional[int] = ..., total_items: _Optional[int] = ..., providers: _Optional[_Iterable[_Union[ProviderPlan, _Mapping]]] = ..., census_id: _Optional[str] = ..., census_status: _Optional[str] = ..., census_started_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., census_completed_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ...) -> None: ...

class ProviderPlan(_message.Message):
    __slots__ = ("provider_id", "provider_version", "estimated_bytes", "item_count", "blocked_reason", "items", "warnings", "approval_mode")
    PROVIDER_ID_FIELD_NUMBER: _ClassVar[int]
    PROVIDER_VERSION_FIELD_NUMBER: _ClassVar[int]
    ESTIMATED_BYTES_FIELD_NUMBER: _ClassVar[int]
    ITEM_COUNT_FIELD_NUMBER: _ClassVar[int]
    BLOCKED_REASON_FIELD_NUMBER: _ClassVar[int]
    ITEMS_FIELD_NUMBER: _ClassVar[int]
    WARNINGS_FIELD_NUMBER: _ClassVar[int]
    APPROVAL_MODE_FIELD_NUMBER: _ClassVar[int]
    provider_id: str
    provider_version: str
    estimated_bytes: int
    item_count: int
    blocked_reason: str
    items: _containers.RepeatedCompositeFieldContainer[PreviewItem]
    warnings: _containers.RepeatedScalarFieldContainer[str]
    approval_mode: str
    def __init__(self, provider_id: _Optional[str] = ..., provider_version: _Optional[str] = ..., estimated_bytes: _Optional[int] = ..., item_count: _Optional[int] = ..., blocked_reason: _Optional[str] = ..., items: _Optional[_Iterable[_Union[PreviewItem, _Mapping]]] = ..., warnings: _Optional[_Iterable[str]] = ..., approval_mode: _Optional[str] = ...) -> None: ...

class PreviewItem(_message.Message):
    __slots__ = ("id", "path", "description", "bytes", "action", "safety_tier")
    ID_FIELD_NUMBER: _ClassVar[int]
    PATH_FIELD_NUMBER: _ClassVar[int]
    DESCRIPTION_FIELD_NUMBER: _ClassVar[int]
    BYTES_FIELD_NUMBER: _ClassVar[int]
    ACTION_FIELD_NUMBER: _ClassVar[int]
    SAFETY_TIER_FIELD_NUMBER: _ClassVar[int]
    id: str
    path: str
    description: str
    bytes: int
    action: str
    safety_tier: str
    def __init__(self, id: _Optional[str] = ..., path: _Optional[str] = ..., description: _Optional[str] = ..., bytes: _Optional[int] = ..., action: _Optional[str] = ..., safety_tier: _Optional[str] = ...) -> None: ...

class ApplyPlanRequest(_message.Message):
    __slots__ = ("plan_id", "policy_version", "approval_mode", "approval_token", "idempotency_key")
    PLAN_ID_FIELD_NUMBER: _ClassVar[int]
    POLICY_VERSION_FIELD_NUMBER: _ClassVar[int]
    APPROVAL_MODE_FIELD_NUMBER: _ClassVar[int]
    APPROVAL_TOKEN_FIELD_NUMBER: _ClassVar[int]
    IDEMPOTENCY_KEY_FIELD_NUMBER: _ClassVar[int]
    plan_id: str
    policy_version: str
    approval_mode: str
    approval_token: str
    idempotency_key: str
    def __init__(self, plan_id: _Optional[str] = ..., policy_version: _Optional[str] = ..., approval_mode: _Optional[str] = ..., approval_token: _Optional[str] = ..., idempotency_key: _Optional[str] = ...) -> None: ...

class ApplyPlanResponse(_message.Message):
    __slots__ = ("plan_id", "idempotency_key", "already_applied", "reclaimed_bytes", "results")
    PLAN_ID_FIELD_NUMBER: _ClassVar[int]
    IDEMPOTENCY_KEY_FIELD_NUMBER: _ClassVar[int]
    ALREADY_APPLIED_FIELD_NUMBER: _ClassVar[int]
    RECLAIMED_BYTES_FIELD_NUMBER: _ClassVar[int]
    RESULTS_FIELD_NUMBER: _ClassVar[int]
    plan_id: str
    idempotency_key: str
    already_applied: bool
    reclaimed_bytes: int
    results: _containers.RepeatedCompositeFieldContainer[ApplyResult]
    def __init__(self, plan_id: _Optional[str] = ..., idempotency_key: _Optional[str] = ..., already_applied: _Optional[bool] = ..., reclaimed_bytes: _Optional[int] = ..., results: _Optional[_Iterable[_Union[ApplyResult, _Mapping]]] = ...) -> None: ...

class ApplyResult(_message.Message):
    __slots__ = ("provider_id", "applied", "already_done", "reclaimed_bytes", "skipped_items", "warnings")
    PROVIDER_ID_FIELD_NUMBER: _ClassVar[int]
    APPLIED_FIELD_NUMBER: _ClassVar[int]
    ALREADY_DONE_FIELD_NUMBER: _ClassVar[int]
    RECLAIMED_BYTES_FIELD_NUMBER: _ClassVar[int]
    SKIPPED_ITEMS_FIELD_NUMBER: _ClassVar[int]
    WARNINGS_FIELD_NUMBER: _ClassVar[int]
    provider_id: str
    applied: bool
    already_done: bool
    reclaimed_bytes: int
    skipped_items: _containers.RepeatedScalarFieldContainer[str]
    warnings: _containers.RepeatedScalarFieldContainer[str]
    def __init__(self, provider_id: _Optional[str] = ..., applied: _Optional[bool] = ..., already_done: _Optional[bool] = ..., reclaimed_bytes: _Optional[int] = ..., skipped_items: _Optional[_Iterable[str]] = ..., warnings: _Optional[_Iterable[str]] = ...) -> None: ...

class ListAuditRequest(_message.Message):
    __slots__ = ()
    def __init__(self) -> None: ...

class ListAuditResponse(_message.Message):
    __slots__ = ("events",)
    EVENTS_FIELD_NUMBER: _ClassVar[int]
    events: _containers.RepeatedCompositeFieldContainer[AuditEvent]
    def __init__(self, events: _Optional[_Iterable[_Union[AuditEvent, _Mapping]]] = ...) -> None: ...

class AuditEvent(_message.Message):
    __slots__ = ("id", "time", "type", "plan_id", "provider_id", "idempotency_key", "message", "redacted")
    ID_FIELD_NUMBER: _ClassVar[int]
    TIME_FIELD_NUMBER: _ClassVar[int]
    TYPE_FIELD_NUMBER: _ClassVar[int]
    PLAN_ID_FIELD_NUMBER: _ClassVar[int]
    PROVIDER_ID_FIELD_NUMBER: _ClassVar[int]
    IDEMPOTENCY_KEY_FIELD_NUMBER: _ClassVar[int]
    MESSAGE_FIELD_NUMBER: _ClassVar[int]
    REDACTED_FIELD_NUMBER: _ClassVar[int]
    id: str
    time: _timestamp_pb2.Timestamp
    type: str
    plan_id: str
    provider_id: str
    idempotency_key: str
    message: str
    redacted: bool
    def __init__(self, id: _Optional[str] = ..., time: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., type: _Optional[str] = ..., plan_id: _Optional[str] = ..., provider_id: _Optional[str] = ..., idempotency_key: _Optional[str] = ..., message: _Optional[str] = ..., redacted: _Optional[bool] = ...) -> None: ...

class HotWriter(_message.Message):
    __slots__ = ("root", "bytes_per_hour", "window_seconds", "current_bytes")
    ROOT_FIELD_NUMBER: _ClassVar[int]
    BYTES_PER_HOUR_FIELD_NUMBER: _ClassVar[int]
    WINDOW_SECONDS_FIELD_NUMBER: _ClassVar[int]
    CURRENT_BYTES_FIELD_NUMBER: _ClassVar[int]
    root: str
    bytes_per_hour: int
    window_seconds: int
    current_bytes: int
    def __init__(self, root: _Optional[str] = ..., bytes_per_hour: _Optional[int] = ..., window_seconds: _Optional[int] = ..., current_bytes: _Optional[int] = ...) -> None: ...

class ReportPressureRequest(_message.Message):
    __slots__ = ("source_scenario", "partition", "used_percent", "band", "available_bytes", "fill_rate_bytes_per_hour", "hot_writers", "trigger")
    SOURCE_SCENARIO_FIELD_NUMBER: _ClassVar[int]
    PARTITION_FIELD_NUMBER: _ClassVar[int]
    USED_PERCENT_FIELD_NUMBER: _ClassVar[int]
    BAND_FIELD_NUMBER: _ClassVar[int]
    AVAILABLE_BYTES_FIELD_NUMBER: _ClassVar[int]
    FILL_RATE_BYTES_PER_HOUR_FIELD_NUMBER: _ClassVar[int]
    HOT_WRITERS_FIELD_NUMBER: _ClassVar[int]
    TRIGGER_FIELD_NUMBER: _ClassVar[int]
    source_scenario: str
    partition: str
    used_percent: float
    band: PressureBand
    available_bytes: int
    fill_rate_bytes_per_hour: int
    hot_writers: _containers.RepeatedCompositeFieldContainer[HotWriter]
    trigger: PressureTrigger
    def __init__(self, source_scenario: _Optional[str] = ..., partition: _Optional[str] = ..., used_percent: _Optional[float] = ..., band: _Optional[_Union[PressureBand, str]] = ..., available_bytes: _Optional[int] = ..., fill_rate_bytes_per_hour: _Optional[int] = ..., hot_writers: _Optional[_Iterable[_Union[HotWriter, _Mapping]]] = ..., trigger: _Optional[_Union[PressureTrigger, str]] = ...) -> None: ...

class ReportPressureResponse(_message.Message):
    __slots__ = ("band", "action", "plan_id", "estimated_bytes", "reclaimed_bytes", "providers_applied", "providers_withheld", "reason", "autonomous_apply_enabled", "bug_reference", "run_id")
    BAND_FIELD_NUMBER: _ClassVar[int]
    ACTION_FIELD_NUMBER: _ClassVar[int]
    PLAN_ID_FIELD_NUMBER: _ClassVar[int]
    ESTIMATED_BYTES_FIELD_NUMBER: _ClassVar[int]
    RECLAIMED_BYTES_FIELD_NUMBER: _ClassVar[int]
    PROVIDERS_APPLIED_FIELD_NUMBER: _ClassVar[int]
    PROVIDERS_WITHHELD_FIELD_NUMBER: _ClassVar[int]
    REASON_FIELD_NUMBER: _ClassVar[int]
    AUTONOMOUS_APPLY_ENABLED_FIELD_NUMBER: _ClassVar[int]
    BUG_REFERENCE_FIELD_NUMBER: _ClassVar[int]
    RUN_ID_FIELD_NUMBER: _ClassVar[int]
    band: PressureBand
    action: PressureAction
    plan_id: str
    estimated_bytes: int
    reclaimed_bytes: int
    providers_applied: _containers.RepeatedScalarFieldContainer[str]
    providers_withheld: _containers.RepeatedScalarFieldContainer[str]
    reason: str
    autonomous_apply_enabled: bool
    bug_reference: str
    run_id: str
    def __init__(self, band: _Optional[_Union[PressureBand, str]] = ..., action: _Optional[_Union[PressureAction, str]] = ..., plan_id: _Optional[str] = ..., estimated_bytes: _Optional[int] = ..., reclaimed_bytes: _Optional[int] = ..., providers_applied: _Optional[_Iterable[str]] = ..., providers_withheld: _Optional[_Iterable[str]] = ..., reason: _Optional[str] = ..., autonomous_apply_enabled: _Optional[bool] = ..., bug_reference: _Optional[str] = ..., run_id: _Optional[str] = ...) -> None: ...

class RecoveryRunRequest(_message.Message):
    __slots__ = ("trigger", "partition", "used_percent", "available_bytes", "dry_run", "target_free_percent")
    TRIGGER_FIELD_NUMBER: _ClassVar[int]
    PARTITION_FIELD_NUMBER: _ClassVar[int]
    USED_PERCENT_FIELD_NUMBER: _ClassVar[int]
    AVAILABLE_BYTES_FIELD_NUMBER: _ClassVar[int]
    DRY_RUN_FIELD_NUMBER: _ClassVar[int]
    TARGET_FREE_PERCENT_FIELD_NUMBER: _ClassVar[int]
    trigger: PressureTrigger
    partition: str
    used_percent: float
    available_bytes: int
    dry_run: bool
    target_free_percent: float
    def __init__(self, trigger: _Optional[_Union[PressureTrigger, str]] = ..., partition: _Optional[str] = ..., used_percent: _Optional[float] = ..., available_bytes: _Optional[int] = ..., dry_run: _Optional[bool] = ..., target_free_percent: _Optional[float] = ...) -> None: ...

class RecoveryWaitRequest(_message.Message):
    __slots__ = ("run_id",)
    RUN_ID_FIELD_NUMBER: _ClassVar[int]
    run_id: str
    def __init__(self, run_id: _Optional[str] = ...) -> None: ...

class RecoveryHistoryRequest(_message.Message):
    __slots__ = ("limit",)
    LIMIT_FIELD_NUMBER: _ClassVar[int]
    limit: int
    def __init__(self, limit: _Optional[int] = ...) -> None: ...

class RecoveryRunResponse(_message.Message):
    __slots__ = ("run_id", "status", "trigger", "partition", "action", "estimated_bytes", "reclaimed_bytes", "plan_id", "reason", "started_at", "completed_at", "target_free_bytes", "stopped_because")
    RUN_ID_FIELD_NUMBER: _ClassVar[int]
    STATUS_FIELD_NUMBER: _ClassVar[int]
    TRIGGER_FIELD_NUMBER: _ClassVar[int]
    PARTITION_FIELD_NUMBER: _ClassVar[int]
    ACTION_FIELD_NUMBER: _ClassVar[int]
    ESTIMATED_BYTES_FIELD_NUMBER: _ClassVar[int]
    RECLAIMED_BYTES_FIELD_NUMBER: _ClassVar[int]
    PLAN_ID_FIELD_NUMBER: _ClassVar[int]
    REASON_FIELD_NUMBER: _ClassVar[int]
    STARTED_AT_FIELD_NUMBER: _ClassVar[int]
    COMPLETED_AT_FIELD_NUMBER: _ClassVar[int]
    TARGET_FREE_BYTES_FIELD_NUMBER: _ClassVar[int]
    STOPPED_BECAUSE_FIELD_NUMBER: _ClassVar[int]
    run_id: str
    status: str
    trigger: str
    partition: str
    action: str
    estimated_bytes: int
    reclaimed_bytes: int
    plan_id: str
    reason: str
    started_at: _timestamp_pb2.Timestamp
    completed_at: _timestamp_pb2.Timestamp
    target_free_bytes: int
    stopped_because: str
    def __init__(self, run_id: _Optional[str] = ..., status: _Optional[str] = ..., trigger: _Optional[str] = ..., partition: _Optional[str] = ..., action: _Optional[str] = ..., estimated_bytes: _Optional[int] = ..., reclaimed_bytes: _Optional[int] = ..., plan_id: _Optional[str] = ..., reason: _Optional[str] = ..., started_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., completed_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., target_free_bytes: _Optional[int] = ..., stopped_because: _Optional[str] = ...) -> None: ...

class RecoveryHistoryResponse(_message.Message):
    __slots__ = ("runs",)
    RUNS_FIELD_NUMBER: _ClassVar[int]
    runs: _containers.RepeatedCompositeFieldContainer[RecoveryRunResponse]
    def __init__(self, runs: _Optional[_Iterable[_Union[RecoveryRunResponse, _Mapping]]] = ...) -> None: ...
