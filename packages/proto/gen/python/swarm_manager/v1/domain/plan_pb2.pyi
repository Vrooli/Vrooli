from google.protobuf.internal import containers as _containers
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Iterable as _Iterable, Mapping as _Mapping
from typing import ClassVar as _ClassVar, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class PlanLaneStatus(_message.Message):
    __slots__ = ("lane", "active", "capacity")
    LANE_FIELD_NUMBER: _ClassVar[int]
    ACTIVE_FIELD_NUMBER: _ClassVar[int]
    CAPACITY_FIELD_NUMBER: _ClassVar[int]
    lane: str
    active: int
    capacity: int
    def __init__(self, lane: _Optional[str] = ..., active: _Optional[int] = ..., capacity: _Optional[int] = ...) -> None: ...

class PlanNowSummary(_message.Message):
    __slots__ = ("active_count", "queue_depth", "max_queue_depth", "lanes")
    ACTIVE_COUNT_FIELD_NUMBER: _ClassVar[int]
    QUEUE_DEPTH_FIELD_NUMBER: _ClassVar[int]
    MAX_QUEUE_DEPTH_FIELD_NUMBER: _ClassVar[int]
    LANES_FIELD_NUMBER: _ClassVar[int]
    active_count: int
    queue_depth: int
    max_queue_depth: int
    lanes: _containers.RepeatedCompositeFieldContainer[PlanLaneStatus]
    def __init__(self, active_count: _Optional[int] = ..., queue_depth: _Optional[int] = ..., max_queue_depth: _Optional[int] = ..., lanes: _Optional[_Iterable[_Union[PlanLaneStatus, _Mapping]]] = ...) -> None: ...

class PlanGate(_message.Message):
    __slots__ = ("id", "kind", "owner_type", "owner_kind", "owner_name", "owner_title", "count", "blocks", "decidable_since", "suggested")
    ID_FIELD_NUMBER: _ClassVar[int]
    KIND_FIELD_NUMBER: _ClassVar[int]
    OWNER_TYPE_FIELD_NUMBER: _ClassVar[int]
    OWNER_KIND_FIELD_NUMBER: _ClassVar[int]
    OWNER_NAME_FIELD_NUMBER: _ClassVar[int]
    OWNER_TITLE_FIELD_NUMBER: _ClassVar[int]
    COUNT_FIELD_NUMBER: _ClassVar[int]
    BLOCKS_FIELD_NUMBER: _ClassVar[int]
    DECIDABLE_SINCE_FIELD_NUMBER: _ClassVar[int]
    SUGGESTED_FIELD_NUMBER: _ClassVar[int]
    id: str
    kind: str
    owner_type: str
    owner_kind: str
    owner_name: str
    owner_title: str
    count: int
    blocks: _containers.RepeatedScalarFieldContainer[str]
    decidable_since: str
    suggested: str
    def __init__(self, id: _Optional[str] = ..., kind: _Optional[str] = ..., owner_type: _Optional[str] = ..., owner_kind: _Optional[str] = ..., owner_name: _Optional[str] = ..., owner_title: _Optional[str] = ..., count: _Optional[int] = ..., blocks: _Optional[_Iterable[str]] = ..., decidable_since: _Optional[str] = ..., suggested: _Optional[str] = ...) -> None: ...

class PlanCard(_message.Message):
    __slots__ = ("id", "card_type", "action", "item_kind", "item_name", "title", "status", "priority", "wave", "initiative", "effort", "gate", "outcome", "finished_at", "execution_id", "unblocks")
    ID_FIELD_NUMBER: _ClassVar[int]
    CARD_TYPE_FIELD_NUMBER: _ClassVar[int]
    ACTION_FIELD_NUMBER: _ClassVar[int]
    ITEM_KIND_FIELD_NUMBER: _ClassVar[int]
    ITEM_NAME_FIELD_NUMBER: _ClassVar[int]
    TITLE_FIELD_NUMBER: _ClassVar[int]
    STATUS_FIELD_NUMBER: _ClassVar[int]
    PRIORITY_FIELD_NUMBER: _ClassVar[int]
    WAVE_FIELD_NUMBER: _ClassVar[int]
    INITIATIVE_FIELD_NUMBER: _ClassVar[int]
    EFFORT_FIELD_NUMBER: _ClassVar[int]
    GATE_FIELD_NUMBER: _ClassVar[int]
    OUTCOME_FIELD_NUMBER: _ClassVar[int]
    FINISHED_AT_FIELD_NUMBER: _ClassVar[int]
    EXECUTION_ID_FIELD_NUMBER: _ClassVar[int]
    UNBLOCKS_FIELD_NUMBER: _ClassVar[int]
    id: str
    card_type: str
    action: str
    item_kind: str
    item_name: str
    title: str
    status: str
    priority: int
    wave: int
    initiative: str
    effort: str
    gate: PlanGate
    outcome: str
    finished_at: str
    execution_id: str
    unblocks: int
    def __init__(self, id: _Optional[str] = ..., card_type: _Optional[str] = ..., action: _Optional[str] = ..., item_kind: _Optional[str] = ..., item_name: _Optional[str] = ..., title: _Optional[str] = ..., status: _Optional[str] = ..., priority: _Optional[int] = ..., wave: _Optional[int] = ..., initiative: _Optional[str] = ..., effort: _Optional[str] = ..., gate: _Optional[_Union[PlanGate, _Mapping]] = ..., outcome: _Optional[str] = ..., finished_at: _Optional[str] = ..., execution_id: _Optional[str] = ..., unblocks: _Optional[int] = ...) -> None: ...

class PlanCardGroup(_message.Message):
    __slots__ = ("id", "label", "blocker_kind", "gate_id", "blocker_keys", "cards")
    ID_FIELD_NUMBER: _ClassVar[int]
    LABEL_FIELD_NUMBER: _ClassVar[int]
    BLOCKER_KIND_FIELD_NUMBER: _ClassVar[int]
    GATE_ID_FIELD_NUMBER: _ClassVar[int]
    BLOCKER_KEYS_FIELD_NUMBER: _ClassVar[int]
    CARDS_FIELD_NUMBER: _ClassVar[int]
    id: str
    label: str
    blocker_kind: str
    gate_id: str
    blocker_keys: _containers.RepeatedScalarFieldContainer[str]
    cards: _containers.RepeatedCompositeFieldContainer[PlanCard]
    def __init__(self, id: _Optional[str] = ..., label: _Optional[str] = ..., blocker_kind: _Optional[str] = ..., gate_id: _Optional[str] = ..., blocker_keys: _Optional[_Iterable[str]] = ..., cards: _Optional[_Iterable[_Union[PlanCard, _Mapping]]] = ...) -> None: ...

class PlanColumn(_message.Message):
    __slots__ = ("groups", "card_count")
    GROUPS_FIELD_NUMBER: _ClassVar[int]
    CARD_COUNT_FIELD_NUMBER: _ClassVar[int]
    groups: _containers.RepeatedCompositeFieldContainer[PlanCardGroup]
    card_count: int
    def __init__(self, groups: _Optional[_Iterable[_Union[PlanCardGroup, _Mapping]]] = ..., card_count: _Optional[int] = ...) -> None: ...

class PlanBoardMeta(_message.Message):
    __slots__ = ("generated_at", "window_seconds", "max_wave", "cycles")
    GENERATED_AT_FIELD_NUMBER: _ClassVar[int]
    WINDOW_SECONDS_FIELD_NUMBER: _ClassVar[int]
    MAX_WAVE_FIELD_NUMBER: _ClassVar[int]
    CYCLES_FIELD_NUMBER: _ClassVar[int]
    generated_at: str
    window_seconds: int
    max_wave: int
    cycles: _containers.RepeatedScalarFieldContainer[str]
    def __init__(self, generated_at: _Optional[str] = ..., window_seconds: _Optional[int] = ..., max_wave: _Optional[int] = ..., cycles: _Optional[_Iterable[str]] = ...) -> None: ...
