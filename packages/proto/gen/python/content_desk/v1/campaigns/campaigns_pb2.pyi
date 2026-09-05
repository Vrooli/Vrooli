from google.protobuf.internal import containers as _containers
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Iterable as _Iterable, Mapping as _Mapping
from typing import ClassVar as _ClassVar, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class Campaign(_message.Message):
    __slots__ = ("id", "name", "status", "scenario_names")
    ID_FIELD_NUMBER: _ClassVar[int]
    NAME_FIELD_NUMBER: _ClassVar[int]
    STATUS_FIELD_NUMBER: _ClassVar[int]
    SCENARIO_NAMES_FIELD_NUMBER: _ClassVar[int]
    id: str
    name: str
    status: str
    scenario_names: _containers.RepeatedScalarFieldContainer[str]
    def __init__(self, id: _Optional[str] = ..., name: _Optional[str] = ..., status: _Optional[str] = ..., scenario_names: _Optional[_Iterable[str]] = ...) -> None: ...

class ListCampaignsRequest(_message.Message):
    __slots__ = ()
    def __init__(self) -> None: ...

class ListCampaignsResponse(_message.Message):
    __slots__ = ("campaigns",)
    CAMPAIGNS_FIELD_NUMBER: _ClassVar[int]
    campaigns: _containers.RepeatedCompositeFieldContainer[Campaign]
    def __init__(self, campaigns: _Optional[_Iterable[_Union[Campaign, _Mapping]]] = ...) -> None: ...

class CampaignSlot(_message.Message):
    __slots__ = ("channel", "format", "capacity")
    CHANNEL_FIELD_NUMBER: _ClassVar[int]
    FORMAT_FIELD_NUMBER: _ClassVar[int]
    CAPACITY_FIELD_NUMBER: _ClassVar[int]
    channel: str
    format: str
    capacity: int
    def __init__(self, channel: _Optional[str] = ..., format: _Optional[str] = ..., capacity: _Optional[int] = ...) -> None: ...

class CreateCampaignRequest(_message.Message):
    __slots__ = ("name", "evidence_refs", "slots", "scenario_names")
    NAME_FIELD_NUMBER: _ClassVar[int]
    EVIDENCE_REFS_FIELD_NUMBER: _ClassVar[int]
    SLOTS_FIELD_NUMBER: _ClassVar[int]
    SCENARIO_NAMES_FIELD_NUMBER: _ClassVar[int]
    name: str
    evidence_refs: _containers.RepeatedScalarFieldContainer[str]
    slots: _containers.RepeatedCompositeFieldContainer[CampaignSlot]
    scenario_names: _containers.RepeatedScalarFieldContainer[str]
    def __init__(self, name: _Optional[str] = ..., evidence_refs: _Optional[_Iterable[str]] = ..., slots: _Optional[_Iterable[_Union[CampaignSlot, _Mapping]]] = ..., scenario_names: _Optional[_Iterable[str]] = ...) -> None: ...

class CreateCampaignResponse(_message.Message):
    __slots__ = ("campaign",)
    CAMPAIGN_FIELD_NUMBER: _ClassVar[int]
    campaign: Campaign
    def __init__(self, campaign: _Optional[_Union[Campaign, _Mapping]] = ...) -> None: ...

class ActivateCampaignRequest(_message.Message):
    __slots__ = ("id",)
    ID_FIELD_NUMBER: _ClassVar[int]
    id: str
    def __init__(self, id: _Optional[str] = ...) -> None: ...

class ActivateCampaignResponse(_message.Message):
    __slots__ = ("campaign",)
    CAMPAIGN_FIELD_NUMBER: _ClassVar[int]
    campaign: Campaign
    def __init__(self, campaign: _Optional[_Union[Campaign, _Mapping]] = ...) -> None: ...

class GetLaunchAssetsRequest(_message.Message):
    __slots__ = ("scenario_name",)
    SCENARIO_NAME_FIELD_NUMBER: _ClassVar[int]
    scenario_name: str
    def __init__(self, scenario_name: _Optional[str] = ...) -> None: ...

class LaunchAssetSlot(_message.Message):
    __slots__ = ("campaign_id", "campaign_name", "channel", "format", "capacity", "reserved", "draft_count")
    CAMPAIGN_ID_FIELD_NUMBER: _ClassVar[int]
    CAMPAIGN_NAME_FIELD_NUMBER: _ClassVar[int]
    CHANNEL_FIELD_NUMBER: _ClassVar[int]
    FORMAT_FIELD_NUMBER: _ClassVar[int]
    CAPACITY_FIELD_NUMBER: _ClassVar[int]
    RESERVED_FIELD_NUMBER: _ClassVar[int]
    DRAFT_COUNT_FIELD_NUMBER: _ClassVar[int]
    campaign_id: str
    campaign_name: str
    channel: str
    format: str
    capacity: int
    reserved: int
    draft_count: int
    def __init__(self, campaign_id: _Optional[str] = ..., campaign_name: _Optional[str] = ..., channel: _Optional[str] = ..., format: _Optional[str] = ..., capacity: _Optional[int] = ..., reserved: _Optional[int] = ..., draft_count: _Optional[int] = ...) -> None: ...

class GetLaunchAssetsResponse(_message.Message):
    __slots__ = ("scenario_name", "slots")
    SCENARIO_NAME_FIELD_NUMBER: _ClassVar[int]
    SLOTS_FIELD_NUMBER: _ClassVar[int]
    scenario_name: str
    slots: _containers.RepeatedCompositeFieldContainer[LaunchAssetSlot]
    def __init__(self, scenario_name: _Optional[str] = ..., slots: _Optional[_Iterable[_Union[LaunchAssetSlot, _Mapping]]] = ...) -> None: ...
