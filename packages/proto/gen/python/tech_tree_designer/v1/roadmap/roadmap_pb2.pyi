from google.protobuf.internal import containers as _containers
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Iterable as _Iterable, Mapping as _Mapping
from typing import ClassVar as _ClassVar, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class Sector(_message.Message):
    __slots__ = ("slug", "name", "description")
    SLUG_FIELD_NUMBER: _ClassVar[int]
    NAME_FIELD_NUMBER: _ClassVar[int]
    DESCRIPTION_FIELD_NUMBER: _ClassVar[int]
    slug: str
    name: str
    description: str
    def __init__(self, slug: _Optional[str] = ..., name: _Optional[str] = ..., description: _Optional[str] = ...) -> None: ...

class Milestone(_message.Message):
    __slots__ = ("id", "name", "description", "required_scenarios")
    ID_FIELD_NUMBER: _ClassVar[int]
    NAME_FIELD_NUMBER: _ClassVar[int]
    DESCRIPTION_FIELD_NUMBER: _ClassVar[int]
    REQUIRED_SCENARIOS_FIELD_NUMBER: _ClassVar[int]
    id: str
    name: str
    description: str
    required_scenarios: _containers.RepeatedScalarFieldContainer[str]
    def __init__(self, id: _Optional[str] = ..., name: _Optional[str] = ..., description: _Optional[str] = ..., required_scenarios: _Optional[_Iterable[str]] = ...) -> None: ...

class ProgressRollup(_message.Message):
    __slots__ = ("buckets",)
    BUCKETS_FIELD_NUMBER: _ClassVar[int]
    buckets: _containers.RepeatedCompositeFieldContainer[ProgressBucket]
    def __init__(self, buckets: _Optional[_Iterable[_Union[ProgressBucket, _Mapping]]] = ...) -> None: ...

class ProgressBucket(_message.Message):
    __slots__ = ("sector", "tier", "planned", "live", "beta", "stable")
    SECTOR_FIELD_NUMBER: _ClassVar[int]
    TIER_FIELD_NUMBER: _ClassVar[int]
    PLANNED_FIELD_NUMBER: _ClassVar[int]
    LIVE_FIELD_NUMBER: _ClassVar[int]
    BETA_FIELD_NUMBER: _ClassVar[int]
    STABLE_FIELD_NUMBER: _ClassVar[int]
    sector: str
    tier: str
    planned: int
    live: int
    beta: int
    stable: int
    def __init__(self, sector: _Optional[str] = ..., tier: _Optional[str] = ..., planned: _Optional[int] = ..., live: _Optional[int] = ..., beta: _Optional[int] = ..., stable: _Optional[int] = ...) -> None: ...

class ListSectorsRequest(_message.Message):
    __slots__ = ()
    def __init__(self) -> None: ...

class ListSectorsResponse(_message.Message):
    __slots__ = ("sectors",)
    SECTORS_FIELD_NUMBER: _ClassVar[int]
    sectors: _containers.RepeatedCompositeFieldContainer[Sector]
    def __init__(self, sectors: _Optional[_Iterable[_Union[Sector, _Mapping]]] = ...) -> None: ...

class UpsertSectorRequest(_message.Message):
    __slots__ = ("sector",)
    SECTOR_FIELD_NUMBER: _ClassVar[int]
    sector: Sector
    def __init__(self, sector: _Optional[_Union[Sector, _Mapping]] = ...) -> None: ...

class ListMilestonesRequest(_message.Message):
    __slots__ = ()
    def __init__(self) -> None: ...

class ListMilestonesResponse(_message.Message):
    __slots__ = ("milestones",)
    MILESTONES_FIELD_NUMBER: _ClassVar[int]
    milestones: _containers.RepeatedCompositeFieldContainer[Milestone]
    def __init__(self, milestones: _Optional[_Iterable[_Union[Milestone, _Mapping]]] = ...) -> None: ...

class UpsertMilestoneRequest(_message.Message):
    __slots__ = ("milestone",)
    MILESTONE_FIELD_NUMBER: _ClassVar[int]
    milestone: Milestone
    def __init__(self, milestone: _Optional[_Union[Milestone, _Mapping]] = ...) -> None: ...

class GetProgressRequest(_message.Message):
    __slots__ = ("sector", "tier")
    SECTOR_FIELD_NUMBER: _ClassVar[int]
    TIER_FIELD_NUMBER: _ClassVar[int]
    sector: str
    tier: str
    def __init__(self, sector: _Optional[str] = ..., tier: _Optional[str] = ...) -> None: ...
