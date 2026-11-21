from google.protobuf.internal import containers as _containers
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Iterable as _Iterable, Mapping as _Mapping
from typing import ClassVar as _ClassVar, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class KitchenDevice(_message.Message):
    __slots__ = ("id", "type", "count", "capacity", "capabilities")
    ID_FIELD_NUMBER: _ClassVar[int]
    TYPE_FIELD_NUMBER: _ClassVar[int]
    COUNT_FIELD_NUMBER: _ClassVar[int]
    CAPACITY_FIELD_NUMBER: _ClassVar[int]
    CAPABILITIES_FIELD_NUMBER: _ClassVar[int]
    id: str
    type: str
    count: int
    capacity: str
    capabilities: _containers.RepeatedScalarFieldContainer[str]
    def __init__(self, id: _Optional[str] = ..., type: _Optional[str] = ..., count: _Optional[int] = ..., capacity: _Optional[str] = ..., capabilities: _Optional[_Iterable[str]] = ...) -> None: ...

class Profile(_message.Message):
    __slots__ = ("workspace_id", "revision", "preset", "preset_version", "active_rules", "excluded_groups", "allergies", "appliances", "cost_weight", "effort_weight", "variety_weight", "draft_json", "cookware", "tools", "kitchen_devices")
    WORKSPACE_ID_FIELD_NUMBER: _ClassVar[int]
    REVISION_FIELD_NUMBER: _ClassVar[int]
    PRESET_FIELD_NUMBER: _ClassVar[int]
    PRESET_VERSION_FIELD_NUMBER: _ClassVar[int]
    ACTIVE_RULES_FIELD_NUMBER: _ClassVar[int]
    EXCLUDED_GROUPS_FIELD_NUMBER: _ClassVar[int]
    ALLERGIES_FIELD_NUMBER: _ClassVar[int]
    APPLIANCES_FIELD_NUMBER: _ClassVar[int]
    COST_WEIGHT_FIELD_NUMBER: _ClassVar[int]
    EFFORT_WEIGHT_FIELD_NUMBER: _ClassVar[int]
    VARIETY_WEIGHT_FIELD_NUMBER: _ClassVar[int]
    DRAFT_JSON_FIELD_NUMBER: _ClassVar[int]
    COOKWARE_FIELD_NUMBER: _ClassVar[int]
    TOOLS_FIELD_NUMBER: _ClassVar[int]
    KITCHEN_DEVICES_FIELD_NUMBER: _ClassVar[int]
    workspace_id: str
    revision: int
    preset: str
    preset_version: int
    active_rules: _containers.RepeatedScalarFieldContainer[str]
    excluded_groups: _containers.RepeatedScalarFieldContainer[str]
    allergies: _containers.RepeatedScalarFieldContainer[str]
    appliances: _containers.RepeatedScalarFieldContainer[str]
    cost_weight: float
    effort_weight: float
    variety_weight: float
    draft_json: str
    cookware: _containers.RepeatedScalarFieldContainer[str]
    tools: _containers.RepeatedScalarFieldContainer[str]
    kitchen_devices: _containers.RepeatedCompositeFieldContainer[KitchenDevice]
    def __init__(self, workspace_id: _Optional[str] = ..., revision: _Optional[int] = ..., preset: _Optional[str] = ..., preset_version: _Optional[int] = ..., active_rules: _Optional[_Iterable[str]] = ..., excluded_groups: _Optional[_Iterable[str]] = ..., allergies: _Optional[_Iterable[str]] = ..., appliances: _Optional[_Iterable[str]] = ..., cost_weight: _Optional[float] = ..., effort_weight: _Optional[float] = ..., variety_weight: _Optional[float] = ..., draft_json: _Optional[str] = ..., cookware: _Optional[_Iterable[str]] = ..., tools: _Optional[_Iterable[str]] = ..., kitchen_devices: _Optional[_Iterable[_Union[KitchenDevice, _Mapping]]] = ...) -> None: ...

class GetProfileRequest(_message.Message):
    __slots__ = ("workspace_id",)
    WORKSPACE_ID_FIELD_NUMBER: _ClassVar[int]
    workspace_id: str
    def __init__(self, workspace_id: _Optional[str] = ...) -> None: ...

class GetProfileResponse(_message.Message):
    __slots__ = ("profile",)
    PROFILE_FIELD_NUMBER: _ClassVar[int]
    profile: Profile
    def __init__(self, profile: _Optional[_Union[Profile, _Mapping]] = ...) -> None: ...

class SaveProfileDraftRequest(_message.Message):
    __slots__ = ("workspace_id", "draft_json")
    WORKSPACE_ID_FIELD_NUMBER: _ClassVar[int]
    DRAFT_JSON_FIELD_NUMBER: _ClassVar[int]
    workspace_id: str
    draft_json: str
    def __init__(self, workspace_id: _Optional[str] = ..., draft_json: _Optional[str] = ...) -> None: ...

class SaveProfileDraftResponse(_message.Message):
    __slots__ = ("profile",)
    PROFILE_FIELD_NUMBER: _ClassVar[int]
    profile: Profile
    def __init__(self, profile: _Optional[_Union[Profile, _Mapping]] = ...) -> None: ...

class ApplyProfileRequest(_message.Message):
    __slots__ = ("workspace_id", "preset", "excluded_groups", "allergies", "appliances", "cost_weight", "effort_weight", "variety_weight", "cookware", "tools", "kitchen_devices", "replace_kitchen_devices")
    WORKSPACE_ID_FIELD_NUMBER: _ClassVar[int]
    PRESET_FIELD_NUMBER: _ClassVar[int]
    EXCLUDED_GROUPS_FIELD_NUMBER: _ClassVar[int]
    ALLERGIES_FIELD_NUMBER: _ClassVar[int]
    APPLIANCES_FIELD_NUMBER: _ClassVar[int]
    COST_WEIGHT_FIELD_NUMBER: _ClassVar[int]
    EFFORT_WEIGHT_FIELD_NUMBER: _ClassVar[int]
    VARIETY_WEIGHT_FIELD_NUMBER: _ClassVar[int]
    COOKWARE_FIELD_NUMBER: _ClassVar[int]
    TOOLS_FIELD_NUMBER: _ClassVar[int]
    KITCHEN_DEVICES_FIELD_NUMBER: _ClassVar[int]
    REPLACE_KITCHEN_DEVICES_FIELD_NUMBER: _ClassVar[int]
    workspace_id: str
    preset: str
    excluded_groups: _containers.RepeatedScalarFieldContainer[str]
    allergies: _containers.RepeatedScalarFieldContainer[str]
    appliances: _containers.RepeatedScalarFieldContainer[str]
    cost_weight: float
    effort_weight: float
    variety_weight: float
    cookware: _containers.RepeatedScalarFieldContainer[str]
    tools: _containers.RepeatedScalarFieldContainer[str]
    kitchen_devices: _containers.RepeatedCompositeFieldContainer[KitchenDevice]
    replace_kitchen_devices: bool
    def __init__(self, workspace_id: _Optional[str] = ..., preset: _Optional[str] = ..., excluded_groups: _Optional[_Iterable[str]] = ..., allergies: _Optional[_Iterable[str]] = ..., appliances: _Optional[_Iterable[str]] = ..., cost_weight: _Optional[float] = ..., effort_weight: _Optional[float] = ..., variety_weight: _Optional[float] = ..., cookware: _Optional[_Iterable[str]] = ..., tools: _Optional[_Iterable[str]] = ..., kitchen_devices: _Optional[_Iterable[_Union[KitchenDevice, _Mapping]]] = ..., replace_kitchen_devices: _Optional[bool] = ...) -> None: ...

class ApplyProfileResponse(_message.Message):
    __slots__ = ("profile", "conflicts", "matching_meals", "needs_review_meals", "excluded_meals")
    PROFILE_FIELD_NUMBER: _ClassVar[int]
    CONFLICTS_FIELD_NUMBER: _ClassVar[int]
    MATCHING_MEALS_FIELD_NUMBER: _ClassVar[int]
    NEEDS_REVIEW_MEALS_FIELD_NUMBER: _ClassVar[int]
    EXCLUDED_MEALS_FIELD_NUMBER: _ClassVar[int]
    profile: Profile
    conflicts: _containers.RepeatedScalarFieldContainer[str]
    matching_meals: int
    needs_review_meals: int
    excluded_meals: int
    def __init__(self, profile: _Optional[_Union[Profile, _Mapping]] = ..., conflicts: _Optional[_Iterable[str]] = ..., matching_meals: _Optional[int] = ..., needs_review_meals: _Optional[int] = ..., excluded_meals: _Optional[int] = ...) -> None: ...
