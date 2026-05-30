import datetime

from architecture.v1 import findings_pb2 as _findings_pb2
from google.protobuf import timestamp_pb2 as _timestamp_pb2
from google.protobuf.internal import containers as _containers
from google.protobuf.internal import enum_type_wrapper as _enum_type_wrapper
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Iterable as _Iterable, Mapping as _Mapping
from typing import ClassVar as _ClassVar, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class CampaignItemStatus(int, metaclass=_enum_type_wrapper.EnumTypeWrapper):
    __slots__ = ()
    CAMPAIGN_ITEM_STATUS_UNSPECIFIED: _ClassVar[CampaignItemStatus]
    CAMPAIGN_ITEM_STATUS_DETECTED: _ClassVar[CampaignItemStatus]
    CAMPAIGN_ITEM_STATUS_ASSIGNED: _ClassVar[CampaignItemStatus]
    CAMPAIGN_ITEM_STATUS_SPLIT: _ClassVar[CampaignItemStatus]
    CAMPAIGN_ITEM_STATUS_RESOLVED: _ClassVar[CampaignItemStatus]
    CAMPAIGN_ITEM_STATUS_VALIDATED: _ClassVar[CampaignItemStatus]
    CAMPAIGN_ITEM_STATUS_COMMITTED: _ClassVar[CampaignItemStatus]
    CAMPAIGN_ITEM_STATUS_FORCE_RESOLVED: _ClassVar[CampaignItemStatus]

class CampaignLifecycle(int, metaclass=_enum_type_wrapper.EnumTypeWrapper):
    __slots__ = ()
    CAMPAIGN_LIFECYCLE_UNSPECIFIED: _ClassVar[CampaignLifecycle]
    CAMPAIGN_LIFECYCLE_OPEN: _ClassVar[CampaignLifecycle]
    CAMPAIGN_LIFECYCLE_CLOSED: _ClassVar[CampaignLifecycle]

class RankProfile(int, metaclass=_enum_type_wrapper.EnumTypeWrapper):
    __slots__ = ()
    RANK_PROFILE_UNSPECIFIED: _ClassVar[RankProfile]
    RANK_PROFILE_BALANCED: _ClassVar[RankProfile]
    RANK_PROFILE_FAST: _ClassVar[RankProfile]
    RANK_PROFILE_LONG_TERM: _ClassVar[RankProfile]
CAMPAIGN_ITEM_STATUS_UNSPECIFIED: CampaignItemStatus
CAMPAIGN_ITEM_STATUS_DETECTED: CampaignItemStatus
CAMPAIGN_ITEM_STATUS_ASSIGNED: CampaignItemStatus
CAMPAIGN_ITEM_STATUS_SPLIT: CampaignItemStatus
CAMPAIGN_ITEM_STATUS_RESOLVED: CampaignItemStatus
CAMPAIGN_ITEM_STATUS_VALIDATED: CampaignItemStatus
CAMPAIGN_ITEM_STATUS_COMMITTED: CampaignItemStatus
CAMPAIGN_ITEM_STATUS_FORCE_RESOLVED: CampaignItemStatus
CAMPAIGN_LIFECYCLE_UNSPECIFIED: CampaignLifecycle
CAMPAIGN_LIFECYCLE_OPEN: CampaignLifecycle
CAMPAIGN_LIFECYCLE_CLOSED: CampaignLifecycle
RANK_PROFILE_UNSPECIFIED: RankProfile
RANK_PROFILE_BALANCED: RankProfile
RANK_PROFILE_FAST: RankProfile
RANK_PROFILE_LONG_TERM: RankProfile

class CampaignItem(_message.Message):
    __slots__ = ("stable_id", "scenario", "source", "code", "severity", "locations", "domains", "message", "suggestion", "status", "resolution_note", "regressed", "first_seen_at", "updated_at", "effort")
    STABLE_ID_FIELD_NUMBER: _ClassVar[int]
    SCENARIO_FIELD_NUMBER: _ClassVar[int]
    SOURCE_FIELD_NUMBER: _ClassVar[int]
    CODE_FIELD_NUMBER: _ClassVar[int]
    SEVERITY_FIELD_NUMBER: _ClassVar[int]
    LOCATIONS_FIELD_NUMBER: _ClassVar[int]
    DOMAINS_FIELD_NUMBER: _ClassVar[int]
    MESSAGE_FIELD_NUMBER: _ClassVar[int]
    SUGGESTION_FIELD_NUMBER: _ClassVar[int]
    STATUS_FIELD_NUMBER: _ClassVar[int]
    RESOLUTION_NOTE_FIELD_NUMBER: _ClassVar[int]
    REGRESSED_FIELD_NUMBER: _ClassVar[int]
    FIRST_SEEN_AT_FIELD_NUMBER: _ClassVar[int]
    UPDATED_AT_FIELD_NUMBER: _ClassVar[int]
    EFFORT_FIELD_NUMBER: _ClassVar[int]
    stable_id: str
    scenario: str
    source: str
    code: str
    severity: str
    locations: _containers.RepeatedScalarFieldContainer[str]
    domains: _containers.RepeatedScalarFieldContainer[str]
    message: str
    suggestion: str
    status: CampaignItemStatus
    resolution_note: str
    regressed: bool
    first_seen_at: _timestamp_pb2.Timestamp
    updated_at: _timestamp_pb2.Timestamp
    effort: str
    def __init__(self, stable_id: _Optional[str] = ..., scenario: _Optional[str] = ..., source: _Optional[str] = ..., code: _Optional[str] = ..., severity: _Optional[str] = ..., locations: _Optional[_Iterable[str]] = ..., domains: _Optional[_Iterable[str]] = ..., message: _Optional[str] = ..., suggestion: _Optional[str] = ..., status: _Optional[_Union[CampaignItemStatus, str]] = ..., resolution_note: _Optional[str] = ..., regressed: _Optional[bool] = ..., first_seen_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., updated_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., effort: _Optional[str] = ...) -> None: ...

class Campaign(_message.Message):
    __slots__ = ("id", "scenario", "name", "status", "created_at", "updated_at")
    ID_FIELD_NUMBER: _ClassVar[int]
    SCENARIO_FIELD_NUMBER: _ClassVar[int]
    NAME_FIELD_NUMBER: _ClassVar[int]
    STATUS_FIELD_NUMBER: _ClassVar[int]
    CREATED_AT_FIELD_NUMBER: _ClassVar[int]
    UPDATED_AT_FIELD_NUMBER: _ClassVar[int]
    id: str
    scenario: str
    name: str
    status: CampaignLifecycle
    created_at: _timestamp_pb2.Timestamp
    updated_at: _timestamp_pb2.Timestamp
    def __init__(self, id: _Optional[str] = ..., scenario: _Optional[str] = ..., name: _Optional[str] = ..., status: _Optional[_Union[CampaignLifecycle, str]] = ..., created_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., updated_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ...) -> None: ...

class CampaignStatus(_message.Message):
    __slots__ = ("campaign", "items", "total", "open", "resolved", "validated", "regressions", "by_severity", "by_status")
    class BySeverityEntry(_message.Message):
        __slots__ = ("key", "value")
        KEY_FIELD_NUMBER: _ClassVar[int]
        VALUE_FIELD_NUMBER: _ClassVar[int]
        key: str
        value: int
        def __init__(self, key: _Optional[str] = ..., value: _Optional[int] = ...) -> None: ...
    class ByStatusEntry(_message.Message):
        __slots__ = ("key", "value")
        KEY_FIELD_NUMBER: _ClassVar[int]
        VALUE_FIELD_NUMBER: _ClassVar[int]
        key: str
        value: int
        def __init__(self, key: _Optional[str] = ..., value: _Optional[int] = ...) -> None: ...
    CAMPAIGN_FIELD_NUMBER: _ClassVar[int]
    ITEMS_FIELD_NUMBER: _ClassVar[int]
    TOTAL_FIELD_NUMBER: _ClassVar[int]
    OPEN_FIELD_NUMBER: _ClassVar[int]
    RESOLVED_FIELD_NUMBER: _ClassVar[int]
    VALIDATED_FIELD_NUMBER: _ClassVar[int]
    REGRESSIONS_FIELD_NUMBER: _ClassVar[int]
    BY_SEVERITY_FIELD_NUMBER: _ClassVar[int]
    BY_STATUS_FIELD_NUMBER: _ClassVar[int]
    campaign: Campaign
    items: _containers.RepeatedCompositeFieldContainer[CampaignItem]
    total: int
    open: int
    resolved: int
    validated: int
    regressions: int
    by_severity: _containers.ScalarMap[str, int]
    by_status: _containers.ScalarMap[str, int]
    def __init__(self, campaign: _Optional[_Union[Campaign, _Mapping]] = ..., items: _Optional[_Iterable[_Union[CampaignItem, _Mapping]]] = ..., total: _Optional[int] = ..., open: _Optional[int] = ..., resolved: _Optional[int] = ..., validated: _Optional[int] = ..., regressions: _Optional[int] = ..., by_severity: _Optional[_Mapping[str, int]] = ..., by_status: _Optional[_Mapping[str, int]] = ...) -> None: ...

class CreateCampaignRequest(_message.Message):
    __slots__ = ("scenario", "name", "findings")
    SCENARIO_FIELD_NUMBER: _ClassVar[int]
    NAME_FIELD_NUMBER: _ClassVar[int]
    FINDINGS_FIELD_NUMBER: _ClassVar[int]
    scenario: str
    name: str
    findings: _containers.RepeatedCompositeFieldContainer[_findings_pb2.ArchitectureFinding]
    def __init__(self, scenario: _Optional[str] = ..., name: _Optional[str] = ..., findings: _Optional[_Iterable[_Union[_findings_pb2.ArchitectureFinding, _Mapping]]] = ...) -> None: ...

class CreateCampaignResponse(_message.Message):
    __slots__ = ("status",)
    STATUS_FIELD_NUMBER: _ClassVar[int]
    status: CampaignStatus
    def __init__(self, status: _Optional[_Union[CampaignStatus, _Mapping]] = ...) -> None: ...

class ListCampaignsRequest(_message.Message):
    __slots__ = ("scenario",)
    SCENARIO_FIELD_NUMBER: _ClassVar[int]
    scenario: str
    def __init__(self, scenario: _Optional[str] = ...) -> None: ...

class ListCampaignsResponse(_message.Message):
    __slots__ = ("campaigns",)
    CAMPAIGNS_FIELD_NUMBER: _ClassVar[int]
    campaigns: _containers.RepeatedCompositeFieldContainer[Campaign]
    def __init__(self, campaigns: _Optional[_Iterable[_Union[Campaign, _Mapping]]] = ...) -> None: ...

class GetCampaignStatusRequest(_message.Message):
    __slots__ = ("campaign_id",)
    CAMPAIGN_ID_FIELD_NUMBER: _ClassVar[int]
    campaign_id: str
    def __init__(self, campaign_id: _Optional[str] = ...) -> None: ...

class GetCampaignStatusResponse(_message.Message):
    __slots__ = ("status",)
    STATUS_FIELD_NUMBER: _ClassVar[int]
    status: CampaignStatus
    def __init__(self, status: _Optional[_Union[CampaignStatus, _Mapping]] = ...) -> None: ...

class NextCampaignStepRequest(_message.Message):
    __slots__ = ("campaign_id", "profile")
    CAMPAIGN_ID_FIELD_NUMBER: _ClassVar[int]
    PROFILE_FIELD_NUMBER: _ClassVar[int]
    campaign_id: str
    profile: RankProfile
    def __init__(self, campaign_id: _Optional[str] = ..., profile: _Optional[_Union[RankProfile, str]] = ...) -> None: ...

class NextCampaignStepResponse(_message.Message):
    __slots__ = ("items",)
    ITEMS_FIELD_NUMBER: _ClassVar[int]
    items: _containers.RepeatedCompositeFieldContainer[CampaignItem]
    def __init__(self, items: _Optional[_Iterable[_Union[CampaignItem, _Mapping]]] = ...) -> None: ...

class ResolveItemRequest(_message.Message):
    __slots__ = ("campaign_id", "stable_id", "note")
    CAMPAIGN_ID_FIELD_NUMBER: _ClassVar[int]
    STABLE_ID_FIELD_NUMBER: _ClassVar[int]
    NOTE_FIELD_NUMBER: _ClassVar[int]
    campaign_id: str
    stable_id: str
    note: str
    def __init__(self, campaign_id: _Optional[str] = ..., stable_id: _Optional[str] = ..., note: _Optional[str] = ...) -> None: ...

class ResolveItemResponse(_message.Message):
    __slots__ = ("item",)
    ITEM_FIELD_NUMBER: _ClassVar[int]
    item: CampaignItem
    def __init__(self, item: _Optional[_Union[CampaignItem, _Mapping]] = ...) -> None: ...

class ApplyItemRequest(_message.Message):
    __slots__ = ("campaign_id", "stable_id")
    CAMPAIGN_ID_FIELD_NUMBER: _ClassVar[int]
    STABLE_ID_FIELD_NUMBER: _ClassVar[int]
    campaign_id: str
    stable_id: str
    def __init__(self, campaign_id: _Optional[str] = ..., stable_id: _Optional[str] = ...) -> None: ...

class ApplyItemResponse(_message.Message):
    __slots__ = ("item",)
    ITEM_FIELD_NUMBER: _ClassVar[int]
    item: CampaignItem
    def __init__(self, item: _Optional[_Union[CampaignItem, _Mapping]] = ...) -> None: ...

class ReauditCampaignRequest(_message.Message):
    __slots__ = ("campaign_id", "findings")
    CAMPAIGN_ID_FIELD_NUMBER: _ClassVar[int]
    FINDINGS_FIELD_NUMBER: _ClassVar[int]
    campaign_id: str
    findings: _containers.RepeatedCompositeFieldContainer[_findings_pb2.ArchitectureFinding]
    def __init__(self, campaign_id: _Optional[str] = ..., findings: _Optional[_Iterable[_Union[_findings_pb2.ArchitectureFinding, _Mapping]]] = ...) -> None: ...

class ReauditCampaignResponse(_message.Message):
    __slots__ = ("validated", "still_open", "regressions", "status")
    VALIDATED_FIELD_NUMBER: _ClassVar[int]
    STILL_OPEN_FIELD_NUMBER: _ClassVar[int]
    REGRESSIONS_FIELD_NUMBER: _ClassVar[int]
    STATUS_FIELD_NUMBER: _ClassVar[int]
    validated: _containers.RepeatedCompositeFieldContainer[CampaignItem]
    still_open: _containers.RepeatedCompositeFieldContainer[CampaignItem]
    regressions: _containers.RepeatedCompositeFieldContainer[CampaignItem]
    status: CampaignStatus
    def __init__(self, validated: _Optional[_Iterable[_Union[CampaignItem, _Mapping]]] = ..., still_open: _Optional[_Iterable[_Union[CampaignItem, _Mapping]]] = ..., regressions: _Optional[_Iterable[_Union[CampaignItem, _Mapping]]] = ..., status: _Optional[_Union[CampaignStatus, _Mapping]] = ...) -> None: ...

class CloseCampaignRequest(_message.Message):
    __slots__ = ("campaign_id",)
    CAMPAIGN_ID_FIELD_NUMBER: _ClassVar[int]
    campaign_id: str
    def __init__(self, campaign_id: _Optional[str] = ...) -> None: ...

class CloseCampaignResponse(_message.Message):
    __slots__ = ("status",)
    STATUS_FIELD_NUMBER: _ClassVar[int]
    status: CampaignStatus
    def __init__(self, status: _Optional[_Union[CampaignStatus, _Mapping]] = ...) -> None: ...
