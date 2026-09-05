from google.protobuf.internal import containers as _containers
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Iterable as _Iterable, Mapping as _Mapping
from typing import ClassVar as _ClassVar, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class RelatedBacklogTarget(_message.Message):
    __slots__ = ("kind", "name")
    KIND_FIELD_NUMBER: _ClassVar[int]
    NAME_FIELD_NUMBER: _ClassVar[int]
    kind: str
    name: str
    def __init__(self, kind: _Optional[str] = ..., name: _Optional[str] = ...) -> None: ...

class RelatedGoalTarget(_message.Message):
    __slots__ = ("name",)
    NAME_FIELD_NUMBER: _ClassVar[int]
    name: str
    def __init__(self, name: _Optional[str] = ...) -> None: ...

class GetRelatedRequest(_message.Message):
    __slots__ = ("backlog", "goal", "exclude_historical", "entity_kinds", "limit")
    BACKLOG_FIELD_NUMBER: _ClassVar[int]
    GOAL_FIELD_NUMBER: _ClassVar[int]
    EXCLUDE_HISTORICAL_FIELD_NUMBER: _ClassVar[int]
    ENTITY_KINDS_FIELD_NUMBER: _ClassVar[int]
    LIMIT_FIELD_NUMBER: _ClassVar[int]
    backlog: RelatedBacklogTarget
    goal: RelatedGoalTarget
    exclude_historical: bool
    entity_kinds: _containers.RepeatedScalarFieldContainer[str]
    limit: int
    def __init__(self, backlog: _Optional[_Union[RelatedBacklogTarget, _Mapping]] = ..., goal: _Optional[_Union[RelatedGoalTarget, _Mapping]] = ..., exclude_historical: _Optional[bool] = ..., entity_kinds: _Optional[_Iterable[str]] = ..., limit: _Optional[int] = ...) -> None: ...

class RelatedEntity(_message.Message):
    __slots__ = ("entity_kind", "key", "title", "status", "archived", "reasons", "score_percent")
    ENTITY_KIND_FIELD_NUMBER: _ClassVar[int]
    KEY_FIELD_NUMBER: _ClassVar[int]
    TITLE_FIELD_NUMBER: _ClassVar[int]
    STATUS_FIELD_NUMBER: _ClassVar[int]
    ARCHIVED_FIELD_NUMBER: _ClassVar[int]
    REASONS_FIELD_NUMBER: _ClassVar[int]
    SCORE_PERCENT_FIELD_NUMBER: _ClassVar[int]
    entity_kind: str
    key: str
    title: str
    status: str
    archived: bool
    reasons: _containers.RepeatedScalarFieldContainer[str]
    score_percent: int
    def __init__(self, entity_kind: _Optional[str] = ..., key: _Optional[str] = ..., title: _Optional[str] = ..., status: _Optional[str] = ..., archived: _Optional[bool] = ..., reasons: _Optional[_Iterable[str]] = ..., score_percent: _Optional[int] = ...) -> None: ...

class RelatedGroup(_message.Message):
    __slots__ = ("name", "entities", "degraded")
    NAME_FIELD_NUMBER: _ClassVar[int]
    ENTITIES_FIELD_NUMBER: _ClassVar[int]
    DEGRADED_FIELD_NUMBER: _ClassVar[int]
    name: str
    entities: _containers.RepeatedCompositeFieldContainer[RelatedEntity]
    degraded: bool
    def __init__(self, name: _Optional[str] = ..., entities: _Optional[_Iterable[_Union[RelatedEntity, _Mapping]]] = ..., degraded: _Optional[bool] = ...) -> None: ...

class GetRelatedResponse(_message.Message):
    __slots__ = ("groups",)
    GROUPS_FIELD_NUMBER: _ClassVar[int]
    groups: _containers.RepeatedCompositeFieldContainer[RelatedGroup]
    def __init__(self, groups: _Optional[_Iterable[_Union[RelatedGroup, _Mapping]]] = ...) -> None: ...
