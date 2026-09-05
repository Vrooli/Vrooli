from google.protobuf.internal import containers as _containers
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Iterable as _Iterable, Mapping as _Mapping
from typing import ClassVar as _ClassVar, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class PolicyChange(_message.Message):
    __slots__ = ("id", "target", "action", "status", "effects", "rollback_supported")
    ID_FIELD_NUMBER: _ClassVar[int]
    TARGET_FIELD_NUMBER: _ClassVar[int]
    ACTION_FIELD_NUMBER: _ClassVar[int]
    STATUS_FIELD_NUMBER: _ClassVar[int]
    EFFECTS_FIELD_NUMBER: _ClassVar[int]
    ROLLBACK_SUPPORTED_FIELD_NUMBER: _ClassVar[int]
    id: str
    target: str
    action: str
    status: str
    effects: _containers.RepeatedScalarFieldContainer[str]
    rollback_supported: bool
    def __init__(self, id: _Optional[str] = ..., target: _Optional[str] = ..., action: _Optional[str] = ..., status: _Optional[str] = ..., effects: _Optional[_Iterable[str]] = ..., rollback_supported: _Optional[bool] = ...) -> None: ...

class PolicyProfile(_message.Message):
    __slots__ = ("id", "name", "device_group", "filtering_strength", "schedule", "override_behavior", "status", "effects", "updated_at")
    ID_FIELD_NUMBER: _ClassVar[int]
    NAME_FIELD_NUMBER: _ClassVar[int]
    DEVICE_GROUP_FIELD_NUMBER: _ClassVar[int]
    FILTERING_STRENGTH_FIELD_NUMBER: _ClassVar[int]
    SCHEDULE_FIELD_NUMBER: _ClassVar[int]
    OVERRIDE_BEHAVIOR_FIELD_NUMBER: _ClassVar[int]
    STATUS_FIELD_NUMBER: _ClassVar[int]
    EFFECTS_FIELD_NUMBER: _ClassVar[int]
    UPDATED_AT_FIELD_NUMBER: _ClassVar[int]
    id: str
    name: str
    device_group: str
    filtering_strength: str
    schedule: str
    override_behavior: str
    status: str
    effects: _containers.RepeatedScalarFieldContainer[str]
    updated_at: str
    def __init__(self, id: _Optional[str] = ..., name: _Optional[str] = ..., device_group: _Optional[str] = ..., filtering_strength: _Optional[str] = ..., schedule: _Optional[str] = ..., override_behavior: _Optional[str] = ..., status: _Optional[str] = ..., effects: _Optional[_Iterable[str]] = ..., updated_at: _Optional[str] = ...) -> None: ...

class PolicyScheduleEvaluation(_message.Message):
    __slots__ = ("profile_id", "profile_name", "target", "active", "status", "effects", "next_change_at")
    PROFILE_ID_FIELD_NUMBER: _ClassVar[int]
    PROFILE_NAME_FIELD_NUMBER: _ClassVar[int]
    TARGET_FIELD_NUMBER: _ClassVar[int]
    ACTIVE_FIELD_NUMBER: _ClassVar[int]
    STATUS_FIELD_NUMBER: _ClassVar[int]
    EFFECTS_FIELD_NUMBER: _ClassVar[int]
    NEXT_CHANGE_AT_FIELD_NUMBER: _ClassVar[int]
    profile_id: str
    profile_name: str
    target: str
    active: bool
    status: str
    effects: _containers.RepeatedScalarFieldContainer[str]
    next_change_at: str
    def __init__(self, profile_id: _Optional[str] = ..., profile_name: _Optional[str] = ..., target: _Optional[str] = ..., active: _Optional[bool] = ..., status: _Optional[str] = ..., effects: _Optional[_Iterable[str]] = ..., next_change_at: _Optional[str] = ...) -> None: ...

class GuidanceCheck(_message.Message):
    __slots__ = ("id", "title", "status", "evidence", "recommendations")
    ID_FIELD_NUMBER: _ClassVar[int]
    TITLE_FIELD_NUMBER: _ClassVar[int]
    STATUS_FIELD_NUMBER: _ClassVar[int]
    EVIDENCE_FIELD_NUMBER: _ClassVar[int]
    RECOMMENDATIONS_FIELD_NUMBER: _ClassVar[int]
    id: str
    title: str
    status: str
    evidence: str
    recommendations: _containers.RepeatedScalarFieldContainer[str]
    def __init__(self, id: _Optional[str] = ..., title: _Optional[str] = ..., status: _Optional[str] = ..., evidence: _Optional[str] = ..., recommendations: _Optional[_Iterable[str]] = ...) -> None: ...

class PolicyGuidanceReport(_message.Message):
    __slots__ = ("id", "target", "profile", "status", "checks", "manual_steps", "adapter_actions", "guardrails", "generated_at")
    ID_FIELD_NUMBER: _ClassVar[int]
    TARGET_FIELD_NUMBER: _ClassVar[int]
    PROFILE_FIELD_NUMBER: _ClassVar[int]
    STATUS_FIELD_NUMBER: _ClassVar[int]
    CHECKS_FIELD_NUMBER: _ClassVar[int]
    MANUAL_STEPS_FIELD_NUMBER: _ClassVar[int]
    ADAPTER_ACTIONS_FIELD_NUMBER: _ClassVar[int]
    GUARDRAILS_FIELD_NUMBER: _ClassVar[int]
    GENERATED_AT_FIELD_NUMBER: _ClassVar[int]
    id: str
    target: str
    profile: str
    status: str
    checks: _containers.RepeatedCompositeFieldContainer[GuidanceCheck]
    manual_steps: _containers.RepeatedScalarFieldContainer[str]
    adapter_actions: _containers.RepeatedScalarFieldContainer[str]
    guardrails: _containers.RepeatedScalarFieldContainer[str]
    generated_at: str
    def __init__(self, id: _Optional[str] = ..., target: _Optional[str] = ..., profile: _Optional[str] = ..., status: _Optional[str] = ..., checks: _Optional[_Iterable[_Union[GuidanceCheck, _Mapping]]] = ..., manual_steps: _Optional[_Iterable[str]] = ..., adapter_actions: _Optional[_Iterable[str]] = ..., guardrails: _Optional[_Iterable[str]] = ..., generated_at: _Optional[str] = ...) -> None: ...

class PreviewPolicyChangeRequest(_message.Message):
    __slots__ = ("target", "action", "values")
    TARGET_FIELD_NUMBER: _ClassVar[int]
    ACTION_FIELD_NUMBER: _ClassVar[int]
    VALUES_FIELD_NUMBER: _ClassVar[int]
    target: str
    action: str
    values: _containers.RepeatedScalarFieldContainer[str]
    def __init__(self, target: _Optional[str] = ..., action: _Optional[str] = ..., values: _Optional[_Iterable[str]] = ...) -> None: ...

class PreviewPolicyChangeResponse(_message.Message):
    __slots__ = ("preview",)
    PREVIEW_FIELD_NUMBER: _ClassVar[int]
    preview: PolicyChange
    def __init__(self, preview: _Optional[_Union[PolicyChange, _Mapping]] = ...) -> None: ...

class ApplyPolicyChangeRequest(_message.Message):
    __slots__ = ("preview_id", "approved")
    PREVIEW_ID_FIELD_NUMBER: _ClassVar[int]
    APPROVED_FIELD_NUMBER: _ClassVar[int]
    preview_id: str
    approved: bool
    def __init__(self, preview_id: _Optional[str] = ..., approved: _Optional[bool] = ...) -> None: ...

class ApplyPolicyChangeResponse(_message.Message):
    __slots__ = ("change",)
    CHANGE_FIELD_NUMBER: _ClassVar[int]
    change: PolicyChange
    def __init__(self, change: _Optional[_Union[PolicyChange, _Mapping]] = ...) -> None: ...

class RollbackPolicyChangeRequest(_message.Message):
    __slots__ = ("id",)
    ID_FIELD_NUMBER: _ClassVar[int]
    id: str
    def __init__(self, id: _Optional[str] = ...) -> None: ...

class RollbackPolicyChangeResponse(_message.Message):
    __slots__ = ("change",)
    CHANGE_FIELD_NUMBER: _ClassVar[int]
    change: PolicyChange
    def __init__(self, change: _Optional[_Union[PolicyChange, _Mapping]] = ...) -> None: ...

class PauseFilteringRequest(_message.Message):
    __slots__ = ("target", "duration")
    TARGET_FIELD_NUMBER: _ClassVar[int]
    DURATION_FIELD_NUMBER: _ClassVar[int]
    target: str
    duration: str
    def __init__(self, target: _Optional[str] = ..., duration: _Optional[str] = ...) -> None: ...

class PauseFilteringResponse(_message.Message):
    __slots__ = ("change",)
    CHANGE_FIELD_NUMBER: _ClassVar[int]
    change: PolicyChange
    def __init__(self, change: _Optional[_Union[PolicyChange, _Mapping]] = ...) -> None: ...

class ResumeFilteringRequest(_message.Message):
    __slots__ = ("target",)
    TARGET_FIELD_NUMBER: _ClassVar[int]
    target: str
    def __init__(self, target: _Optional[str] = ...) -> None: ...

class ResumeFilteringResponse(_message.Message):
    __slots__ = ("change",)
    CHANGE_FIELD_NUMBER: _ClassVar[int]
    change: PolicyChange
    def __init__(self, change: _Optional[_Union[PolicyChange, _Mapping]] = ...) -> None: ...

class ListPolicyProfilesRequest(_message.Message):
    __slots__ = ("device_group",)
    DEVICE_GROUP_FIELD_NUMBER: _ClassVar[int]
    device_group: str
    def __init__(self, device_group: _Optional[str] = ...) -> None: ...

class ListPolicyProfilesResponse(_message.Message):
    __slots__ = ("profiles",)
    PROFILES_FIELD_NUMBER: _ClassVar[int]
    profiles: _containers.RepeatedCompositeFieldContainer[PolicyProfile]
    def __init__(self, profiles: _Optional[_Iterable[_Union[PolicyProfile, _Mapping]]] = ...) -> None: ...

class UpsertPolicyProfileRequest(_message.Message):
    __slots__ = ("profile",)
    PROFILE_FIELD_NUMBER: _ClassVar[int]
    profile: PolicyProfile
    def __init__(self, profile: _Optional[_Union[PolicyProfile, _Mapping]] = ...) -> None: ...

class UpsertPolicyProfileResponse(_message.Message):
    __slots__ = ("profile",)
    PROFILE_FIELD_NUMBER: _ClassVar[int]
    profile: PolicyProfile
    def __init__(self, profile: _Optional[_Union[PolicyProfile, _Mapping]] = ...) -> None: ...

class EvaluatePolicyScheduleRequest(_message.Message):
    __slots__ = ("profile_id", "target", "now")
    PROFILE_ID_FIELD_NUMBER: _ClassVar[int]
    TARGET_FIELD_NUMBER: _ClassVar[int]
    NOW_FIELD_NUMBER: _ClassVar[int]
    profile_id: str
    target: str
    now: str
    def __init__(self, profile_id: _Optional[str] = ..., target: _Optional[str] = ..., now: _Optional[str] = ...) -> None: ...

class EvaluatePolicyScheduleResponse(_message.Message):
    __slots__ = ("evaluation",)
    EVALUATION_FIELD_NUMBER: _ClassVar[int]
    evaluation: PolicyScheduleEvaluation
    def __init__(self, evaluation: _Optional[_Union[PolicyScheduleEvaluation, _Mapping]] = ...) -> None: ...

class DiagnoseEncryptedDnsBypassRequest(_message.Message):
    __slots__ = ("target", "adapter_backed")
    TARGET_FIELD_NUMBER: _ClassVar[int]
    ADAPTER_BACKED_FIELD_NUMBER: _ClassVar[int]
    target: str
    adapter_backed: bool
    def __init__(self, target: _Optional[str] = ..., adapter_backed: _Optional[bool] = ...) -> None: ...

class DiagnoseEncryptedDnsBypassResponse(_message.Message):
    __slots__ = ("report",)
    REPORT_FIELD_NUMBER: _ClassVar[int]
    report: PolicyGuidanceReport
    def __init__(self, report: _Optional[_Union[PolicyGuidanceReport, _Mapping]] = ...) -> None: ...

class GetEndpointDohGuidanceRequest(_message.Message):
    __slots__ = ("platform", "browser", "management_mode")
    PLATFORM_FIELD_NUMBER: _ClassVar[int]
    BROWSER_FIELD_NUMBER: _ClassVar[int]
    MANAGEMENT_MODE_FIELD_NUMBER: _ClassVar[int]
    platform: str
    browser: str
    management_mode: str
    def __init__(self, platform: _Optional[str] = ..., browser: _Optional[str] = ..., management_mode: _Optional[str] = ...) -> None: ...

class GetEndpointDohGuidanceResponse(_message.Message):
    __slots__ = ("report",)
    REPORT_FIELD_NUMBER: _ClassVar[int]
    report: PolicyGuidanceReport
    def __init__(self, report: _Optional[_Union[PolicyGuidanceReport, _Mapping]] = ...) -> None: ...
