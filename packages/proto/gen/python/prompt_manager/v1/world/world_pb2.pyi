from google.protobuf.internal import containers as _containers
from google.protobuf.internal import enum_type_wrapper as _enum_type_wrapper
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Iterable as _Iterable, Mapping as _Mapping
from typing import ClassVar as _ClassVar, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class WorldEventKind(int, metaclass=_enum_type_wrapper.EnumTypeWrapper):
    __slots__ = ()
    WORLD_EVENT_KIND_UNSPECIFIED: _ClassVar[WorldEventKind]
    WORLD_EVENT_KIND_SNAPSHOT: _ClassVar[WorldEventKind]
    WORLD_EVENT_KIND_RUN_STARTED: _ClassVar[WorldEventKind]
    WORLD_EVENT_KIND_RUN_FINISHED: _ClassVar[WorldEventKind]
    WORLD_EVENT_KIND_RUN_FAILED: _ClassVar[WorldEventKind]
    WORLD_EVENT_KIND_HEARTBEAT_UPCOMING: _ClassVar[WorldEventKind]
    WORLD_EVENT_KIND_HEARTBEAT_CANCELLED: _ClassVar[WorldEventKind]
    WORLD_EVENT_KIND_AGENT_MESSAGE: _ClassVar[WorldEventKind]
WORLD_EVENT_KIND_UNSPECIFIED: WorldEventKind
WORLD_EVENT_KIND_SNAPSHOT: WorldEventKind
WORLD_EVENT_KIND_RUN_STARTED: WorldEventKind
WORLD_EVENT_KIND_RUN_FINISHED: WorldEventKind
WORLD_EVENT_KIND_RUN_FAILED: WorldEventKind
WORLD_EVENT_KIND_HEARTBEAT_UPCOMING: WorldEventKind
WORLD_EVENT_KIND_HEARTBEAT_CANCELLED: WorldEventKind
WORLD_EVENT_KIND_AGENT_MESSAGE: WorldEventKind

class WorldConfig(_message.Message):
    __slots__ = ("scene", "quality_profile", "quality_auto", "period_mode", "two_d_mode", "show_diagnostics", "scale", "updated_at")
    SCENE_FIELD_NUMBER: _ClassVar[int]
    QUALITY_PROFILE_FIELD_NUMBER: _ClassVar[int]
    QUALITY_AUTO_FIELD_NUMBER: _ClassVar[int]
    PERIOD_MODE_FIELD_NUMBER: _ClassVar[int]
    TWO_D_MODE_FIELD_NUMBER: _ClassVar[int]
    SHOW_DIAGNOSTICS_FIELD_NUMBER: _ClassVar[int]
    SCALE_FIELD_NUMBER: _ClassVar[int]
    UPDATED_AT_FIELD_NUMBER: _ClassVar[int]
    scene: str
    quality_profile: str
    quality_auto: bool
    period_mode: str
    two_d_mode: bool
    show_diagnostics: bool
    scale: float
    updated_at: str
    def __init__(self, scene: _Optional[str] = ..., quality_profile: _Optional[str] = ..., quality_auto: _Optional[bool] = ..., period_mode: _Optional[str] = ..., two_d_mode: _Optional[bool] = ..., show_diagnostics: _Optional[bool] = ..., scale: _Optional[float] = ..., updated_at: _Optional[str] = ...) -> None: ...

class GetWorldConfigRequest(_message.Message):
    __slots__ = ()
    def __init__(self) -> None: ...

class SetWorldConfigRequest(_message.Message):
    __slots__ = ("config",)
    CONFIG_FIELD_NUMBER: _ClassVar[int]
    config: WorldConfig
    def __init__(self, config: _Optional[_Union[WorldConfig, _Mapping]] = ...) -> None: ...

class Vec2(_message.Message):
    __slots__ = ("x", "z")
    X_FIELD_NUMBER: _ClassVar[int]
    Z_FIELD_NUMBER: _ClassVar[int]
    x: float
    z: float
    def __init__(self, x: _Optional[float] = ..., z: _Optional[float] = ...) -> None: ...

class LayoutOverride(_message.Message):
    __slots__ = ("place_id", "position", "rotation", "removed")
    PLACE_ID_FIELD_NUMBER: _ClassVar[int]
    POSITION_FIELD_NUMBER: _ClassVar[int]
    ROTATION_FIELD_NUMBER: _ClassVar[int]
    REMOVED_FIELD_NUMBER: _ClassVar[int]
    place_id: str
    position: Vec2
    rotation: float
    removed: bool
    def __init__(self, place_id: _Optional[str] = ..., position: _Optional[_Union[Vec2, _Mapping]] = ..., rotation: _Optional[float] = ..., removed: _Optional[bool] = ...) -> None: ...

class DecorAddition(_message.Message):
    __slots__ = ("id", "prop_id", "position", "rotation", "scale")
    ID_FIELD_NUMBER: _ClassVar[int]
    PROP_ID_FIELD_NUMBER: _ClassVar[int]
    POSITION_FIELD_NUMBER: _ClassVar[int]
    ROTATION_FIELD_NUMBER: _ClassVar[int]
    SCALE_FIELD_NUMBER: _ClassVar[int]
    id: str
    prop_id: str
    position: Vec2
    rotation: float
    scale: float
    def __init__(self, id: _Optional[str] = ..., prop_id: _Optional[str] = ..., position: _Optional[_Union[Vec2, _Mapping]] = ..., rotation: _Optional[float] = ..., scale: _Optional[float] = ...) -> None: ...

class WorldLayout(_message.Message):
    __slots__ = ("scene", "overrides", "decor", "updated_at")
    SCENE_FIELD_NUMBER: _ClassVar[int]
    OVERRIDES_FIELD_NUMBER: _ClassVar[int]
    DECOR_FIELD_NUMBER: _ClassVar[int]
    UPDATED_AT_FIELD_NUMBER: _ClassVar[int]
    scene: str
    overrides: _containers.RepeatedCompositeFieldContainer[LayoutOverride]
    decor: _containers.RepeatedCompositeFieldContainer[DecorAddition]
    updated_at: str
    def __init__(self, scene: _Optional[str] = ..., overrides: _Optional[_Iterable[_Union[LayoutOverride, _Mapping]]] = ..., decor: _Optional[_Iterable[_Union[DecorAddition, _Mapping]]] = ..., updated_at: _Optional[str] = ...) -> None: ...

class GetLayoutRequest(_message.Message):
    __slots__ = ("scene",)
    SCENE_FIELD_NUMBER: _ClassVar[int]
    scene: str
    def __init__(self, scene: _Optional[str] = ...) -> None: ...

class SetLayoutRequest(_message.Message):
    __slots__ = ("layout",)
    LAYOUT_FIELD_NUMBER: _ClassVar[int]
    layout: WorldLayout
    def __init__(self, layout: _Optional[_Union[WorldLayout, _Mapping]] = ...) -> None: ...

class ActiveRunSummary(_message.Message):
    __slots__ = ("team_id", "agent_id", "run_id", "started_at")
    TEAM_ID_FIELD_NUMBER: _ClassVar[int]
    AGENT_ID_FIELD_NUMBER: _ClassVar[int]
    RUN_ID_FIELD_NUMBER: _ClassVar[int]
    STARTED_AT_FIELD_NUMBER: _ClassVar[int]
    team_id: str
    agent_id: str
    run_id: str
    started_at: str
    def __init__(self, team_id: _Optional[str] = ..., agent_id: _Optional[str] = ..., run_id: _Optional[str] = ..., started_at: _Optional[str] = ...) -> None: ...

class UpcomingHeartbeat(_message.Message):
    __slots__ = ("team_id", "agent_id", "scheduled_at")
    TEAM_ID_FIELD_NUMBER: _ClassVar[int]
    AGENT_ID_FIELD_NUMBER: _ClassVar[int]
    SCHEDULED_AT_FIELD_NUMBER: _ClassVar[int]
    team_id: str
    agent_id: str
    scheduled_at: str
    def __init__(self, team_id: _Optional[str] = ..., agent_id: _Optional[str] = ..., scheduled_at: _Optional[str] = ...) -> None: ...

class WorldEvent(_message.Message):
    __slots__ = ("kind", "seq", "at", "agent_id", "team_id", "run_id", "message", "scheduled_at", "active_runs", "upcoming")
    KIND_FIELD_NUMBER: _ClassVar[int]
    SEQ_FIELD_NUMBER: _ClassVar[int]
    AT_FIELD_NUMBER: _ClassVar[int]
    AGENT_ID_FIELD_NUMBER: _ClassVar[int]
    TEAM_ID_FIELD_NUMBER: _ClassVar[int]
    RUN_ID_FIELD_NUMBER: _ClassVar[int]
    MESSAGE_FIELD_NUMBER: _ClassVar[int]
    SCHEDULED_AT_FIELD_NUMBER: _ClassVar[int]
    ACTIVE_RUNS_FIELD_NUMBER: _ClassVar[int]
    UPCOMING_FIELD_NUMBER: _ClassVar[int]
    kind: WorldEventKind
    seq: int
    at: str
    agent_id: str
    team_id: str
    run_id: str
    message: str
    scheduled_at: str
    active_runs: _containers.RepeatedCompositeFieldContainer[ActiveRunSummary]
    upcoming: _containers.RepeatedCompositeFieldContainer[UpcomingHeartbeat]
    def __init__(self, kind: _Optional[_Union[WorldEventKind, str]] = ..., seq: _Optional[int] = ..., at: _Optional[str] = ..., agent_id: _Optional[str] = ..., team_id: _Optional[str] = ..., run_id: _Optional[str] = ..., message: _Optional[str] = ..., scheduled_at: _Optional[str] = ..., active_runs: _Optional[_Iterable[_Union[ActiveRunSummary, _Mapping]]] = ..., upcoming: _Optional[_Iterable[_Union[UpcomingHeartbeat, _Mapping]]] = ...) -> None: ...

class StreamWorldFeedRequest(_message.Message):
    __slots__ = ("since_seq",)
    SINCE_SEQ_FIELD_NUMBER: _ClassVar[int]
    since_seq: int
    def __init__(self, since_seq: _Optional[int] = ...) -> None: ...
