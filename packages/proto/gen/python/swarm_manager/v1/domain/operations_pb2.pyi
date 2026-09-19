from google.protobuf.internal import containers as _containers
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Iterable as _Iterable, Mapping as _Mapping
from typing import ClassVar as _ClassVar, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class OperationsBriefingSummary(_message.Message):
    __slots__ = ("active_activity_count", "recently_finished_count", "queue_depth", "max_queue_depth", "saturated_lanes", "active_lane_count_by_lane", "total_backlog_items", "active_initiatives", "blocked_items", "active_sessions")
    class ActiveLaneCountByLaneEntry(_message.Message):
        __slots__ = ("key", "value")
        KEY_FIELD_NUMBER: _ClassVar[int]
        VALUE_FIELD_NUMBER: _ClassVar[int]
        key: str
        value: int
        def __init__(self, key: _Optional[str] = ..., value: _Optional[int] = ...) -> None: ...
    ACTIVE_ACTIVITY_COUNT_FIELD_NUMBER: _ClassVar[int]
    RECENTLY_FINISHED_COUNT_FIELD_NUMBER: _ClassVar[int]
    QUEUE_DEPTH_FIELD_NUMBER: _ClassVar[int]
    MAX_QUEUE_DEPTH_FIELD_NUMBER: _ClassVar[int]
    SATURATED_LANES_FIELD_NUMBER: _ClassVar[int]
    ACTIVE_LANE_COUNT_BY_LANE_FIELD_NUMBER: _ClassVar[int]
    TOTAL_BACKLOG_ITEMS_FIELD_NUMBER: _ClassVar[int]
    ACTIVE_INITIATIVES_FIELD_NUMBER: _ClassVar[int]
    BLOCKED_ITEMS_FIELD_NUMBER: _ClassVar[int]
    ACTIVE_SESSIONS_FIELD_NUMBER: _ClassVar[int]
    active_activity_count: int
    recently_finished_count: int
    queue_depth: int
    max_queue_depth: int
    saturated_lanes: _containers.RepeatedScalarFieldContainer[str]
    active_lane_count_by_lane: _containers.ScalarMap[str, int]
    total_backlog_items: int
    active_initiatives: int
    blocked_items: int
    active_sessions: int
    def __init__(self, active_activity_count: _Optional[int] = ..., recently_finished_count: _Optional[int] = ..., queue_depth: _Optional[int] = ..., max_queue_depth: _Optional[int] = ..., saturated_lanes: _Optional[_Iterable[str]] = ..., active_lane_count_by_lane: _Optional[_Mapping[str, int]] = ..., total_backlog_items: _Optional[int] = ..., active_initiatives: _Optional[int] = ..., blocked_items: _Optional[int] = ..., active_sessions: _Optional[int] = ...) -> None: ...

class OperationsBriefingActivity(_message.Message):
    __slots__ = ("activity_id", "run_id", "owner_type", "owner_kind", "owner_name", "owner_title", "lane", "status", "purpose", "mode", "phase", "round", "initiative_name", "requested_at", "started_at", "finished_at", "runtime_seconds", "failure_reason")
    ACTIVITY_ID_FIELD_NUMBER: _ClassVar[int]
    RUN_ID_FIELD_NUMBER: _ClassVar[int]
    OWNER_TYPE_FIELD_NUMBER: _ClassVar[int]
    OWNER_KIND_FIELD_NUMBER: _ClassVar[int]
    OWNER_NAME_FIELD_NUMBER: _ClassVar[int]
    OWNER_TITLE_FIELD_NUMBER: _ClassVar[int]
    LANE_FIELD_NUMBER: _ClassVar[int]
    STATUS_FIELD_NUMBER: _ClassVar[int]
    PURPOSE_FIELD_NUMBER: _ClassVar[int]
    MODE_FIELD_NUMBER: _ClassVar[int]
    PHASE_FIELD_NUMBER: _ClassVar[int]
    ROUND_FIELD_NUMBER: _ClassVar[int]
    INITIATIVE_NAME_FIELD_NUMBER: _ClassVar[int]
    REQUESTED_AT_FIELD_NUMBER: _ClassVar[int]
    STARTED_AT_FIELD_NUMBER: _ClassVar[int]
    FINISHED_AT_FIELD_NUMBER: _ClassVar[int]
    RUNTIME_SECONDS_FIELD_NUMBER: _ClassVar[int]
    FAILURE_REASON_FIELD_NUMBER: _ClassVar[int]
    activity_id: str
    run_id: str
    owner_type: str
    owner_kind: str
    owner_name: str
    owner_title: str
    lane: str
    status: str
    purpose: str
    mode: str
    phase: str
    round: int
    initiative_name: str
    requested_at: str
    started_at: str
    finished_at: str
    runtime_seconds: int
    failure_reason: str
    def __init__(self, activity_id: _Optional[str] = ..., run_id: _Optional[str] = ..., owner_type: _Optional[str] = ..., owner_kind: _Optional[str] = ..., owner_name: _Optional[str] = ..., owner_title: _Optional[str] = ..., lane: _Optional[str] = ..., status: _Optional[str] = ..., purpose: _Optional[str] = ..., mode: _Optional[str] = ..., phase: _Optional[str] = ..., round: _Optional[int] = ..., initiative_name: _Optional[str] = ..., requested_at: _Optional[str] = ..., started_at: _Optional[str] = ..., finished_at: _Optional[str] = ..., runtime_seconds: _Optional[int] = ..., failure_reason: _Optional[str] = ...) -> None: ...

class OperationsBriefingAttentionItem(_message.Message):
    __slots__ = ("id", "severity", "reason", "title", "status", "lane", "ref", "command")
    ID_FIELD_NUMBER: _ClassVar[int]
    SEVERITY_FIELD_NUMBER: _ClassVar[int]
    REASON_FIELD_NUMBER: _ClassVar[int]
    TITLE_FIELD_NUMBER: _ClassVar[int]
    STATUS_FIELD_NUMBER: _ClassVar[int]
    LANE_FIELD_NUMBER: _ClassVar[int]
    REF_FIELD_NUMBER: _ClassVar[int]
    COMMAND_FIELD_NUMBER: _ClassVar[int]
    id: str
    severity: str
    reason: str
    title: str
    status: str
    lane: str
    ref: str
    command: str
    def __init__(self, id: _Optional[str] = ..., severity: _Optional[str] = ..., reason: _Optional[str] = ..., title: _Optional[str] = ..., status: _Optional[str] = ..., lane: _Optional[str] = ..., ref: _Optional[str] = ..., command: _Optional[str] = ...) -> None: ...

class OperationsDirectorHandoff(_message.Message):
    __slots__ = ("source_path", "title", "observed_at", "excerpt")
    SOURCE_PATH_FIELD_NUMBER: _ClassVar[int]
    TITLE_FIELD_NUMBER: _ClassVar[int]
    OBSERVED_AT_FIELD_NUMBER: _ClassVar[int]
    EXCERPT_FIELD_NUMBER: _ClassVar[int]
    source_path: str
    title: str
    observed_at: str
    excerpt: str
    def __init__(self, source_path: _Optional[str] = ..., title: _Optional[str] = ..., observed_at: _Optional[str] = ..., excerpt: _Optional[str] = ...) -> None: ...

class OperationsRecommendedAction(_message.Message):
    __slots__ = ("id", "label", "reason", "command", "ui_path")
    ID_FIELD_NUMBER: _ClassVar[int]
    LABEL_FIELD_NUMBER: _ClassVar[int]
    REASON_FIELD_NUMBER: _ClassVar[int]
    COMMAND_FIELD_NUMBER: _ClassVar[int]
    UI_PATH_FIELD_NUMBER: _ClassVar[int]
    id: str
    label: str
    reason: str
    command: str
    ui_path: str
    def __init__(self, id: _Optional[str] = ..., label: _Optional[str] = ..., reason: _Optional[str] = ..., command: _Optional[str] = ..., ui_path: _Optional[str] = ...) -> None: ...

class OperationsDrillDownCommand(_message.Message):
    __slots__ = ("label", "command")
    LABEL_FIELD_NUMBER: _ClassVar[int]
    COMMAND_FIELD_NUMBER: _ClassVar[int]
    label: str
    command: str
    def __init__(self, label: _Optional[str] = ..., command: _Optional[str] = ...) -> None: ...

class OperationsBriefing(_message.Message):
    __slots__ = ("generated_at", "freshness_seconds", "window_seconds", "summary", "active_work", "needs_attention", "recent_completions", "director_handoffs", "recommended_next_actions", "drill_down_commands", "warnings")
    GENERATED_AT_FIELD_NUMBER: _ClassVar[int]
    FRESHNESS_SECONDS_FIELD_NUMBER: _ClassVar[int]
    WINDOW_SECONDS_FIELD_NUMBER: _ClassVar[int]
    SUMMARY_FIELD_NUMBER: _ClassVar[int]
    ACTIVE_WORK_FIELD_NUMBER: _ClassVar[int]
    NEEDS_ATTENTION_FIELD_NUMBER: _ClassVar[int]
    RECENT_COMPLETIONS_FIELD_NUMBER: _ClassVar[int]
    DIRECTOR_HANDOFFS_FIELD_NUMBER: _ClassVar[int]
    RECOMMENDED_NEXT_ACTIONS_FIELD_NUMBER: _ClassVar[int]
    DRILL_DOWN_COMMANDS_FIELD_NUMBER: _ClassVar[int]
    WARNINGS_FIELD_NUMBER: _ClassVar[int]
    generated_at: str
    freshness_seconds: int
    window_seconds: int
    summary: OperationsBriefingSummary
    active_work: _containers.RepeatedCompositeFieldContainer[OperationsBriefingActivity]
    needs_attention: _containers.RepeatedCompositeFieldContainer[OperationsBriefingAttentionItem]
    recent_completions: _containers.RepeatedCompositeFieldContainer[OperationsBriefingActivity]
    director_handoffs: _containers.RepeatedCompositeFieldContainer[OperationsDirectorHandoff]
    recommended_next_actions: _containers.RepeatedCompositeFieldContainer[OperationsRecommendedAction]
    drill_down_commands: _containers.RepeatedCompositeFieldContainer[OperationsDrillDownCommand]
    warnings: _containers.RepeatedScalarFieldContainer[str]
    def __init__(self, generated_at: _Optional[str] = ..., freshness_seconds: _Optional[int] = ..., window_seconds: _Optional[int] = ..., summary: _Optional[_Union[OperationsBriefingSummary, _Mapping]] = ..., active_work: _Optional[_Iterable[_Union[OperationsBriefingActivity, _Mapping]]] = ..., needs_attention: _Optional[_Iterable[_Union[OperationsBriefingAttentionItem, _Mapping]]] = ..., recent_completions: _Optional[_Iterable[_Union[OperationsBriefingActivity, _Mapping]]] = ..., director_handoffs: _Optional[_Iterable[_Union[OperationsDirectorHandoff, _Mapping]]] = ..., recommended_next_actions: _Optional[_Iterable[_Union[OperationsRecommendedAction, _Mapping]]] = ..., drill_down_commands: _Optional[_Iterable[_Union[OperationsDrillDownCommand, _Mapping]]] = ..., warnings: _Optional[_Iterable[str]] = ...) -> None: ...
