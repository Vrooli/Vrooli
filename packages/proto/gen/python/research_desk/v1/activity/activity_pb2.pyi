import datetime

from google.protobuf import timestamp_pb2 as _timestamp_pb2
from research_desk.v1.shared import entry_pb2 as _entry_pb2
from research_desk.v1.shared import commission_pb2 as _commission_pb2
from research_desk.v1.shared import evidence_pb2 as _evidence_pb2
from google.protobuf.internal import containers as _containers
from google.protobuf.internal import enum_type_wrapper as _enum_type_wrapper
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Iterable as _Iterable, Mapping as _Mapping
from typing import ClassVar as _ClassVar, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class CommissionState(int, metaclass=_enum_type_wrapper.EnumTypeWrapper):
    __slots__ = ()
    COMMISSION_STATE_UNSPECIFIED: _ClassVar[CommissionState]
    COMMISSION_STATE_ENDED: _ClassVar[CommissionState]
    COMMISSION_STATE_PREPARED: _ClassVar[CommissionState]
    COMMISSION_STATE_UNKNOWN: _ClassVar[CommissionState]
    COMMISSION_STATE_STOPPED: _ClassVar[CommissionState]
    COMMISSION_STATE_LIVE: _ClassVar[CommissionState]
    COMMISSION_STATE_PAUSED: _ClassVar[CommissionState]
    COMMISSION_STATE_LIMITED: _ClassVar[CommissionState]

class MatchKind(int, metaclass=_enum_type_wrapper.EnumTypeWrapper):
    __slots__ = ()
    MATCH_KIND_UNSPECIFIED: _ClassVar[MatchKind]
    MATCH_KIND_EXACT: _ClassVar[MatchKind]
    MATCH_KIND_RELATED: _ClassVar[MatchKind]

class WebConsoleAccess(int, metaclass=_enum_type_wrapper.EnumTypeWrapper):
    __slots__ = ()
    WEB_CONSOLE_ACCESS_UNSPECIFIED: _ClassVar[WebConsoleAccess]
    WEB_CONSOLE_ACCESS_SERVICE_CREDENTIAL: _ClassVar[WebConsoleAccess]
    WEB_CONSOLE_ACCESS_AVAILABILITY_POLICY: _ClassVar[WebConsoleAccess]
    WEB_CONSOLE_ACCESS_UNAVAILABLE: _ClassVar[WebConsoleAccess]

class CapabilityKind(int, metaclass=_enum_type_wrapper.EnumTypeWrapper):
    __slots__ = ()
    CAPABILITY_KIND_UNSPECIFIED: _ClassVar[CapabilityKind]
    CAPABILITY_KIND_SKILL: _ClassVar[CapabilityKind]
    CAPABILITY_KIND_PROGRAM: _ClassVar[CapabilityKind]

class ChangeSubject(int, metaclass=_enum_type_wrapper.EnumTypeWrapper):
    __slots__ = ()
    CHANGE_SUBJECT_UNSPECIFIED: _ClassVar[ChangeSubject]
    CHANGE_SUBJECT_COMMISSION: _ClassVar[ChangeSubject]
    CHANGE_SUBJECT_TOPIC: _ClassVar[ChangeSubject]

class Baseline(int, metaclass=_enum_type_wrapper.EnumTypeWrapper):
    __slots__ = ()
    BASELINE_UNSPECIFIED: _ClassVar[Baseline]
    BASELINE_LOOK: _ClassVar[Baseline]
    BASELINE_ASKED: _ClassVar[Baseline]
    BASELINE_LAST_DAY: _ClassVar[Baseline]
COMMISSION_STATE_UNSPECIFIED: CommissionState
COMMISSION_STATE_ENDED: CommissionState
COMMISSION_STATE_PREPARED: CommissionState
COMMISSION_STATE_UNKNOWN: CommissionState
COMMISSION_STATE_STOPPED: CommissionState
COMMISSION_STATE_LIVE: CommissionState
COMMISSION_STATE_PAUSED: CommissionState
COMMISSION_STATE_LIMITED: CommissionState
MATCH_KIND_UNSPECIFIED: MatchKind
MATCH_KIND_EXACT: MatchKind
MATCH_KIND_RELATED: MatchKind
WEB_CONSOLE_ACCESS_UNSPECIFIED: WebConsoleAccess
WEB_CONSOLE_ACCESS_SERVICE_CREDENTIAL: WebConsoleAccess
WEB_CONSOLE_ACCESS_AVAILABILITY_POLICY: WebConsoleAccess
WEB_CONSOLE_ACCESS_UNAVAILABLE: WebConsoleAccess
CAPABILITY_KIND_UNSPECIFIED: CapabilityKind
CAPABILITY_KIND_SKILL: CapabilityKind
CAPABILITY_KIND_PROGRAM: CapabilityKind
CHANGE_SUBJECT_UNSPECIFIED: ChangeSubject
CHANGE_SUBJECT_COMMISSION: ChangeSubject
CHANGE_SUBJECT_TOPIC: ChangeSubject
BASELINE_UNSPECIFIED: Baseline
BASELINE_LOOK: Baseline
BASELINE_ASKED: Baseline
BASELINE_LAST_DAY: Baseline

class SessionMatch(_message.Message):
    __slots__ = ("kind", "basis", "web_console_session_id", "thread_id", "goal_status", "at", "source", "window_open")
    KIND_FIELD_NUMBER: _ClassVar[int]
    BASIS_FIELD_NUMBER: _ClassVar[int]
    WEB_CONSOLE_SESSION_ID_FIELD_NUMBER: _ClassVar[int]
    THREAD_ID_FIELD_NUMBER: _ClassVar[int]
    GOAL_STATUS_FIELD_NUMBER: _ClassVar[int]
    AT_FIELD_NUMBER: _ClassVar[int]
    SOURCE_FIELD_NUMBER: _ClassVar[int]
    WINDOW_OPEN_FIELD_NUMBER: _ClassVar[int]
    kind: MatchKind
    basis: str
    web_console_session_id: str
    thread_id: str
    goal_status: str
    at: _timestamp_pb2.Timestamp
    source: str
    window_open: bool
    def __init__(self, kind: _Optional[_Union[MatchKind, str]] = ..., basis: _Optional[str] = ..., web_console_session_id: _Optional[str] = ..., thread_id: _Optional[str] = ..., goal_status: _Optional[str] = ..., at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., source: _Optional[str] = ..., window_open: _Optional[bool] = ...) -> None: ...

class TopicRef(_message.Message):
    __slots__ = ("topic", "basis", "source")
    TOPIC_FIELD_NUMBER: _ClassVar[int]
    BASIS_FIELD_NUMBER: _ClassVar[int]
    SOURCE_FIELD_NUMBER: _ClassVar[int]
    topic: str
    basis: str
    source: str
    def __init__(self, topic: _Optional[str] = ..., basis: _Optional[str] = ..., source: _Optional[str] = ...) -> None: ...

class CommissionSummary(_message.Message):
    __slots__ = ("id", "folder", "title", "state", "state_rule", "state_evidence", "stalled", "topics", "topics_evidence", "started", "allowance_end", "last_write", "claim", "session", "launched_by", "ended", "touched_after_end", "never_started", "entry_reading", "historical_settlement")
    ID_FIELD_NUMBER: _ClassVar[int]
    FOLDER_FIELD_NUMBER: _ClassVar[int]
    TITLE_FIELD_NUMBER: _ClassVar[int]
    STATE_FIELD_NUMBER: _ClassVar[int]
    STATE_RULE_FIELD_NUMBER: _ClassVar[int]
    STATE_EVIDENCE_FIELD_NUMBER: _ClassVar[int]
    STALLED_FIELD_NUMBER: _ClassVar[int]
    TOPICS_FIELD_NUMBER: _ClassVar[int]
    TOPICS_EVIDENCE_FIELD_NUMBER: _ClassVar[int]
    STARTED_FIELD_NUMBER: _ClassVar[int]
    ALLOWANCE_END_FIELD_NUMBER: _ClassVar[int]
    LAST_WRITE_FIELD_NUMBER: _ClassVar[int]
    CLAIM_FIELD_NUMBER: _ClassVar[int]
    SESSION_FIELD_NUMBER: _ClassVar[int]
    LAUNCHED_BY_FIELD_NUMBER: _ClassVar[int]
    ENDED_FIELD_NUMBER: _ClassVar[int]
    TOUCHED_AFTER_END_FIELD_NUMBER: _ClassVar[int]
    NEVER_STARTED_FIELD_NUMBER: _ClassVar[int]
    ENTRY_READING_FIELD_NUMBER: _ClassVar[int]
    HISTORICAL_SETTLEMENT_FIELD_NUMBER: _ClassVar[int]
    id: str
    folder: str
    title: _evidence_pb2.DerivedText
    state: CommissionState
    state_rule: str
    state_evidence: _containers.RepeatedCompositeFieldContainer[_evidence_pb2.Evidence]
    stalled: bool
    topics: _containers.RepeatedCompositeFieldContainer[TopicRef]
    topics_evidence: _evidence_pb2.Evidence
    started: _evidence_pb2.DerivedTime
    allowance_end: _evidence_pb2.DerivedTime
    last_write: _evidence_pb2.DerivedTime
    claim: _evidence_pb2.DerivedText
    session: SessionMatch
    launched_by: str
    ended: _evidence_pb2.DerivedTime
    touched_after_end: _evidence_pb2.DerivedTime
    never_started: _commission_pb2.NeverStarted
    entry_reading: _commission_pb2.EntryReading
    historical_settlement: HistoricalSettlement
    def __init__(self, id: _Optional[str] = ..., folder: _Optional[str] = ..., title: _Optional[_Union[_evidence_pb2.DerivedText, _Mapping]] = ..., state: _Optional[_Union[CommissionState, str]] = ..., state_rule: _Optional[str] = ..., state_evidence: _Optional[_Iterable[_Union[_evidence_pb2.Evidence, _Mapping]]] = ..., stalled: _Optional[bool] = ..., topics: _Optional[_Iterable[_Union[TopicRef, _Mapping]]] = ..., topics_evidence: _Optional[_Union[_evidence_pb2.Evidence, _Mapping]] = ..., started: _Optional[_Union[_evidence_pb2.DerivedTime, _Mapping]] = ..., allowance_end: _Optional[_Union[_evidence_pb2.DerivedTime, _Mapping]] = ..., last_write: _Optional[_Union[_evidence_pb2.DerivedTime, _Mapping]] = ..., claim: _Optional[_Union[_evidence_pb2.DerivedText, _Mapping]] = ..., session: _Optional[_Union[SessionMatch, _Mapping]] = ..., launched_by: _Optional[str] = ..., ended: _Optional[_Union[_evidence_pb2.DerivedTime, _Mapping]] = ..., touched_after_end: _Optional[_Union[_evidence_pb2.DerivedTime, _Mapping]] = ..., never_started: _Optional[_Union[_commission_pb2.NeverStarted, _Mapping]] = ..., entry_reading: _Optional[_Union[_commission_pb2.EntryReading, _Mapping]] = ..., historical_settlement: _Optional[_Union[HistoricalSettlement, _Mapping]] = ...) -> None: ...

class HistoricalSettlement(_message.Message):
    __slots__ = ("status", "as_of", "recorded_at", "reason", "evidence_paths", "recorded_by", "activity_start", "activity_end")
    STATUS_FIELD_NUMBER: _ClassVar[int]
    AS_OF_FIELD_NUMBER: _ClassVar[int]
    RECORDED_AT_FIELD_NUMBER: _ClassVar[int]
    REASON_FIELD_NUMBER: _ClassVar[int]
    EVIDENCE_PATHS_FIELD_NUMBER: _ClassVar[int]
    RECORDED_BY_FIELD_NUMBER: _ClassVar[int]
    ACTIVITY_START_FIELD_NUMBER: _ClassVar[int]
    ACTIVITY_END_FIELD_NUMBER: _ClassVar[int]
    status: str
    as_of: _timestamp_pb2.Timestamp
    recorded_at: _timestamp_pb2.Timestamp
    reason: str
    evidence_paths: _containers.RepeatedScalarFieldContainer[str]
    recorded_by: str
    activity_start: _timestamp_pb2.Timestamp
    activity_end: _timestamp_pb2.Timestamp
    def __init__(self, status: _Optional[str] = ..., as_of: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., recorded_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., reason: _Optional[str] = ..., evidence_paths: _Optional[_Iterable[str]] = ..., recorded_by: _Optional[str] = ..., activity_start: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., activity_end: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ...) -> None: ...

class RunFolder(_message.Message):
    __slots__ = ("topic", "path", "checkpoint", "last_write")
    TOPIC_FIELD_NUMBER: _ClassVar[int]
    PATH_FIELD_NUMBER: _ClassVar[int]
    CHECKPOINT_FIELD_NUMBER: _ClassVar[int]
    LAST_WRITE_FIELD_NUMBER: _ClassVar[int]
    topic: str
    path: str
    checkpoint: _evidence_pb2.FileFact
    last_write: _evidence_pb2.DerivedTime
    def __init__(self, topic: _Optional[str] = ..., path: _Optional[str] = ..., checkpoint: _Optional[_Union[_evidence_pb2.FileFact, _Mapping]] = ..., last_write: _Optional[_Union[_evidence_pb2.DerivedTime, _Mapping]] = ...) -> None: ...

class Overlap(_message.Message):
    __slots__ = ("topic", "commission_ids")
    TOPIC_FIELD_NUMBER: _ClassVar[int]
    COMMISSION_IDS_FIELD_NUMBER: _ClassVar[int]
    topic: str
    commission_ids: _containers.RepeatedScalarFieldContainer[str]
    def __init__(self, topic: _Optional[str] = ..., commission_ids: _Optional[_Iterable[str]] = ...) -> None: ...

class Lens(_message.Message):
    __slots__ = ("name", "available", "reason", "command")
    NAME_FIELD_NUMBER: _ClassVar[int]
    AVAILABLE_FIELD_NUMBER: _ClassVar[int]
    REASON_FIELD_NUMBER: _ClassVar[int]
    COMMAND_FIELD_NUMBER: _ClassVar[int]
    name: str
    available: bool
    reason: str
    command: str
    def __init__(self, name: _Optional[str] = ..., available: _Optional[bool] = ..., reason: _Optional[str] = ..., command: _Optional[str] = ...) -> None: ...

class StateCount(_message.Message):
    __slots__ = ("state", "count")
    STATE_FIELD_NUMBER: _ClassVar[int]
    COUNT_FIELD_NUMBER: _ClassVar[int]
    state: CommissionState
    count: int
    def __init__(self, state: _Optional[_Union[CommissionState, str]] = ..., count: _Optional[int] = ...) -> None: ...

class MissionCheck(_message.Message):
    __slots__ = ("mode", "open", "active", "waiting_operator", "landed", "errors", "warnings", "evidence", "workstreams")
    MODE_FIELD_NUMBER: _ClassVar[int]
    OPEN_FIELD_NUMBER: _ClassVar[int]
    ACTIVE_FIELD_NUMBER: _ClassVar[int]
    WAITING_OPERATOR_FIELD_NUMBER: _ClassVar[int]
    LANDED_FIELD_NUMBER: _ClassVar[int]
    ERRORS_FIELD_NUMBER: _ClassVar[int]
    WARNINGS_FIELD_NUMBER: _ClassVar[int]
    EVIDENCE_FIELD_NUMBER: _ClassVar[int]
    WORKSTREAMS_FIELD_NUMBER: _ClassVar[int]
    mode: str
    open: int
    active: int
    waiting_operator: int
    landed: int
    errors: _containers.RepeatedScalarFieldContainer[str]
    warnings: _containers.RepeatedScalarFieldContainer[str]
    evidence: _evidence_pb2.Evidence
    workstreams: _containers.RepeatedCompositeFieldContainer[_commission_pb2.Workstream]
    def __init__(self, mode: _Optional[str] = ..., open: _Optional[int] = ..., active: _Optional[int] = ..., waiting_operator: _Optional[int] = ..., landed: _Optional[int] = ..., errors: _Optional[_Iterable[str]] = ..., warnings: _Optional[_Iterable[str]] = ..., evidence: _Optional[_Union[_evidence_pb2.Evidence, _Mapping]] = ..., workstreams: _Optional[_Iterable[_Union[_commission_pb2.Workstream, _Mapping]]] = ...) -> None: ...

class ResearchWrite(_message.Message):
    __slots__ = ("at", "commission_id", "commission_state", "note", "topic")
    AT_FIELD_NUMBER: _ClassVar[int]
    COMMISSION_ID_FIELD_NUMBER: _ClassVar[int]
    COMMISSION_STATE_FIELD_NUMBER: _ClassVar[int]
    NOTE_FIELD_NUMBER: _ClassVar[int]
    TOPIC_FIELD_NUMBER: _ClassVar[int]
    at: _evidence_pb2.DerivedTime
    commission_id: str
    commission_state: CommissionState
    note: str
    topic: str
    def __init__(self, at: _Optional[_Union[_evidence_pb2.DerivedTime, _Mapping]] = ..., commission_id: _Optional[str] = ..., commission_state: _Optional[_Union[CommissionState, str]] = ..., note: _Optional[str] = ..., topic: _Optional[str] = ...) -> None: ...

class ReviewTouch(_message.Message):
    __slots__ = ("review_id", "file")
    REVIEW_ID_FIELD_NUMBER: _ClassVar[int]
    FILE_FIELD_NUMBER: _ClassVar[int]
    review_id: str
    file: _evidence_pb2.FileFact
    def __init__(self, review_id: _Optional[str] = ..., file: _Optional[_Union[_evidence_pb2.FileFact, _Mapping]] = ...) -> None: ...

class TopicSummary(_message.Message):
    __slots__ = ("topic", "title", "commission_ids", "mission", "mission_check", "live_commission_ids", "paused_commission_ids", "limited_commission_ids", "aliases", "last_research_write", "last_review", "workspace_files")
    TOPIC_FIELD_NUMBER: _ClassVar[int]
    TITLE_FIELD_NUMBER: _ClassVar[int]
    COMMISSION_IDS_FIELD_NUMBER: _ClassVar[int]
    MISSION_FIELD_NUMBER: _ClassVar[int]
    MISSION_CHECK_FIELD_NUMBER: _ClassVar[int]
    LIVE_COMMISSION_IDS_FIELD_NUMBER: _ClassVar[int]
    PAUSED_COMMISSION_IDS_FIELD_NUMBER: _ClassVar[int]
    LIMITED_COMMISSION_IDS_FIELD_NUMBER: _ClassVar[int]
    ALIASES_FIELD_NUMBER: _ClassVar[int]
    LAST_RESEARCH_WRITE_FIELD_NUMBER: _ClassVar[int]
    LAST_REVIEW_FIELD_NUMBER: _ClassVar[int]
    WORKSPACE_FILES_FIELD_NUMBER: _ClassVar[int]
    topic: str
    title: _evidence_pb2.DerivedText
    commission_ids: _containers.RepeatedScalarFieldContainer[str]
    mission: _evidence_pb2.FileFact
    mission_check: MissionCheck
    live_commission_ids: _containers.RepeatedScalarFieldContainer[str]
    paused_commission_ids: _containers.RepeatedScalarFieldContainer[str]
    limited_commission_ids: _containers.RepeatedScalarFieldContainer[str]
    aliases: _containers.RepeatedScalarFieldContainer[str]
    last_research_write: ResearchWrite
    last_review: ReviewTouch
    workspace_files: TopicWorkspaceFiles
    def __init__(self, topic: _Optional[str] = ..., title: _Optional[_Union[_evidence_pb2.DerivedText, _Mapping]] = ..., commission_ids: _Optional[_Iterable[str]] = ..., mission: _Optional[_Union[_evidence_pb2.FileFact, _Mapping]] = ..., mission_check: _Optional[_Union[MissionCheck, _Mapping]] = ..., live_commission_ids: _Optional[_Iterable[str]] = ..., paused_commission_ids: _Optional[_Iterable[str]] = ..., limited_commission_ids: _Optional[_Iterable[str]] = ..., aliases: _Optional[_Iterable[str]] = ..., last_research_write: _Optional[_Union[ResearchWrite, _Mapping]] = ..., last_review: _Optional[_Union[ReviewTouch, _Mapping]] = ..., workspace_files: _Optional[_Union[TopicWorkspaceFiles, _Mapping]] = ...) -> None: ...

class TopicOptionalFile(_message.Message):
    __slots__ = ("file", "evidence")
    FILE_FIELD_NUMBER: _ClassVar[int]
    EVIDENCE_FIELD_NUMBER: _ClassVar[int]
    file: _evidence_pb2.FileFact
    evidence: _evidence_pb2.Evidence
    def __init__(self, file: _Optional[_Union[_evidence_pb2.FileFact, _Mapping]] = ..., evidence: _Optional[_Union[_evidence_pb2.Evidence, _Mapping]] = ...) -> None: ...

class TopicWorkspaceFiles(_message.Message):
    __slots__ = ("current_view", "tools", "feedback_files", "feedback_evidence", "feedback_truncated", "capabilities", "capabilities_evidence")
    CURRENT_VIEW_FIELD_NUMBER: _ClassVar[int]
    TOOLS_FIELD_NUMBER: _ClassVar[int]
    FEEDBACK_FILES_FIELD_NUMBER: _ClassVar[int]
    FEEDBACK_EVIDENCE_FIELD_NUMBER: _ClassVar[int]
    FEEDBACK_TRUNCATED_FIELD_NUMBER: _ClassVar[int]
    CAPABILITIES_FIELD_NUMBER: _ClassVar[int]
    CAPABILITIES_EVIDENCE_FIELD_NUMBER: _ClassVar[int]
    current_view: TopicOptionalFile
    tools: TopicOptionalFile
    feedback_files: _containers.RepeatedCompositeFieldContainer[_evidence_pb2.FileFact]
    feedback_evidence: _evidence_pb2.Evidence
    feedback_truncated: bool
    capabilities: _containers.RepeatedCompositeFieldContainer[ResearchCapability]
    capabilities_evidence: _evidence_pb2.Evidence
    def __init__(self, current_view: _Optional[_Union[TopicOptionalFile, _Mapping]] = ..., tools: _Optional[_Union[TopicOptionalFile, _Mapping]] = ..., feedback_files: _Optional[_Iterable[_Union[_evidence_pb2.FileFact, _Mapping]]] = ..., feedback_evidence: _Optional[_Union[_evidence_pb2.Evidence, _Mapping]] = ..., feedback_truncated: _Optional[bool] = ..., capabilities: _Optional[_Iterable[_Union[ResearchCapability, _Mapping]]] = ..., capabilities_evidence: _Optional[_Union[_evidence_pb2.Evidence, _Mapping]] = ...) -> None: ...

class ResearchCapability(_message.Message):
    __slots__ = ("id", "kind", "path", "problems")
    ID_FIELD_NUMBER: _ClassVar[int]
    KIND_FIELD_NUMBER: _ClassVar[int]
    PATH_FIELD_NUMBER: _ClassVar[int]
    PROBLEMS_FIELD_NUMBER: _ClassVar[int]
    id: str
    kind: CapabilityKind
    path: str
    problems: _containers.RepeatedScalarFieldContainer[str]
    def __init__(self, id: _Optional[str] = ..., kind: _Optional[_Union[CapabilityKind, str]] = ..., path: _Optional[str] = ..., problems: _Optional[_Iterable[str]] = ...) -> None: ...

class GetOverviewRequest(_message.Message):
    __slots__ = ()
    def __init__(self) -> None: ...

class GetOverviewResponse(_message.Message):
    __slots__ = ("generated_at", "research_home", "state_counts", "running", "overlaps", "lenses", "unreadable", "latest_research_write", "web_console_access_kind", "web_console_access", "open_overlaps")
    GENERATED_AT_FIELD_NUMBER: _ClassVar[int]
    RESEARCH_HOME_FIELD_NUMBER: _ClassVar[int]
    STATE_COUNTS_FIELD_NUMBER: _ClassVar[int]
    RUNNING_FIELD_NUMBER: _ClassVar[int]
    OVERLAPS_FIELD_NUMBER: _ClassVar[int]
    LENSES_FIELD_NUMBER: _ClassVar[int]
    UNREADABLE_FIELD_NUMBER: _ClassVar[int]
    LATEST_RESEARCH_WRITE_FIELD_NUMBER: _ClassVar[int]
    WEB_CONSOLE_ACCESS_KIND_FIELD_NUMBER: _ClassVar[int]
    WEB_CONSOLE_ACCESS_FIELD_NUMBER: _ClassVar[int]
    OPEN_OVERLAPS_FIELD_NUMBER: _ClassVar[int]
    generated_at: _timestamp_pb2.Timestamp
    research_home: str
    state_counts: _containers.RepeatedCompositeFieldContainer[StateCount]
    running: _containers.RepeatedCompositeFieldContainer[CommissionSummary]
    overlaps: _containers.RepeatedCompositeFieldContainer[Overlap]
    lenses: _containers.RepeatedCompositeFieldContainer[Lens]
    unreadable: _containers.RepeatedCompositeFieldContainer[_evidence_pb2.Evidence]
    latest_research_write: ResearchWrite
    web_console_access_kind: WebConsoleAccess
    web_console_access: str
    open_overlaps: _containers.RepeatedCompositeFieldContainer[Overlap]
    def __init__(self, generated_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., research_home: _Optional[str] = ..., state_counts: _Optional[_Iterable[_Union[StateCount, _Mapping]]] = ..., running: _Optional[_Iterable[_Union[CommissionSummary, _Mapping]]] = ..., overlaps: _Optional[_Iterable[_Union[Overlap, _Mapping]]] = ..., lenses: _Optional[_Iterable[_Union[Lens, _Mapping]]] = ..., unreadable: _Optional[_Iterable[_Union[_evidence_pb2.Evidence, _Mapping]]] = ..., latest_research_write: _Optional[_Union[ResearchWrite, _Mapping]] = ..., web_console_access_kind: _Optional[_Union[WebConsoleAccess, str]] = ..., web_console_access: _Optional[str] = ..., open_overlaps: _Optional[_Iterable[_Union[Overlap, _Mapping]]] = ...) -> None: ...

class ListCommissionsRequest(_message.Message):
    __slots__ = ("states",)
    STATES_FIELD_NUMBER: _ClassVar[int]
    states: _containers.RepeatedScalarFieldContainer[CommissionState]
    def __init__(self, states: _Optional[_Iterable[_Union[CommissionState, str]]] = ...) -> None: ...

class ListCommissionsResponse(_message.Message):
    __slots__ = ("commissions",)
    COMMISSIONS_FIELD_NUMBER: _ClassVar[int]
    commissions: _containers.RepeatedCompositeFieldContainer[CommissionSummary]
    def __init__(self, commissions: _Optional[_Iterable[_Union[CommissionSummary, _Mapping]]] = ...) -> None: ...

class GetCommissionRequest(_message.Message):
    __slots__ = ("id",)
    ID_FIELD_NUMBER: _ClassVar[int]
    id: str
    def __init__(self, id: _Optional[str] = ...) -> None: ...

class GetCommissionResponse(_message.Message):
    __slots__ = ("commission", "matches", "files", "runs", "read_sets", "ceilings", "record_errors", "operator_entries", "steering", "decisions", "unreadable", "entry_chars_max", "operator_unparsed_headings")
    COMMISSION_FIELD_NUMBER: _ClassVar[int]
    MATCHES_FIELD_NUMBER: _ClassVar[int]
    FILES_FIELD_NUMBER: _ClassVar[int]
    RUNS_FIELD_NUMBER: _ClassVar[int]
    READ_SETS_FIELD_NUMBER: _ClassVar[int]
    CEILINGS_FIELD_NUMBER: _ClassVar[int]
    RECORD_ERRORS_FIELD_NUMBER: _ClassVar[int]
    OPERATOR_ENTRIES_FIELD_NUMBER: _ClassVar[int]
    STEERING_FIELD_NUMBER: _ClassVar[int]
    DECISIONS_FIELD_NUMBER: _ClassVar[int]
    UNREADABLE_FIELD_NUMBER: _ClassVar[int]
    ENTRY_CHARS_MAX_FIELD_NUMBER: _ClassVar[int]
    OPERATOR_UNPARSED_HEADINGS_FIELD_NUMBER: _ClassVar[int]
    commission: CommissionSummary
    matches: _containers.RepeatedCompositeFieldContainer[SessionMatch]
    files: _containers.RepeatedCompositeFieldContainer[_evidence_pb2.FileFact]
    runs: _containers.RepeatedCompositeFieldContainer[RunFolder]
    read_sets: _containers.RepeatedCompositeFieldContainer[_commission_pb2.ReadSet]
    ceilings: _evidence_pb2.Evidence
    record_errors: _containers.RepeatedScalarFieldContainer[str]
    operator_entries: _containers.RepeatedCompositeFieldContainer[_entry_pb2.OperatorEntry]
    steering: _evidence_pb2.DerivedFlag
    decisions: _evidence_pb2.DerivedFlag
    unreadable: _containers.RepeatedCompositeFieldContainer[_evidence_pb2.Evidence]
    entry_chars_max: int
    operator_unparsed_headings: _containers.RepeatedCompositeFieldContainer[_entry_pb2.UnparsedHeading]
    def __init__(self, commission: _Optional[_Union[CommissionSummary, _Mapping]] = ..., matches: _Optional[_Iterable[_Union[SessionMatch, _Mapping]]] = ..., files: _Optional[_Iterable[_Union[_evidence_pb2.FileFact, _Mapping]]] = ..., runs: _Optional[_Iterable[_Union[RunFolder, _Mapping]]] = ..., read_sets: _Optional[_Iterable[_Union[_commission_pb2.ReadSet, _Mapping]]] = ..., ceilings: _Optional[_Union[_evidence_pb2.Evidence, _Mapping]] = ..., record_errors: _Optional[_Iterable[str]] = ..., operator_entries: _Optional[_Iterable[_Union[_entry_pb2.OperatorEntry, _Mapping]]] = ..., steering: _Optional[_Union[_evidence_pb2.DerivedFlag, _Mapping]] = ..., decisions: _Optional[_Union[_evidence_pb2.DerivedFlag, _Mapping]] = ..., unreadable: _Optional[_Iterable[_Union[_evidence_pb2.Evidence, _Mapping]]] = ..., entry_chars_max: _Optional[int] = ..., operator_unparsed_headings: _Optional[_Iterable[_Union[_entry_pb2.UnparsedHeading, _Mapping]]] = ...) -> None: ...

class ListTopicsRequest(_message.Message):
    __slots__ = ()
    def __init__(self) -> None: ...

class ListTopicsResponse(_message.Message):
    __slots__ = ("topics", "research_home", "observed_at", "home_capabilities", "home_capabilities_evidence")
    TOPICS_FIELD_NUMBER: _ClassVar[int]
    RESEARCH_HOME_FIELD_NUMBER: _ClassVar[int]
    OBSERVED_AT_FIELD_NUMBER: _ClassVar[int]
    HOME_CAPABILITIES_FIELD_NUMBER: _ClassVar[int]
    HOME_CAPABILITIES_EVIDENCE_FIELD_NUMBER: _ClassVar[int]
    topics: _containers.RepeatedCompositeFieldContainer[TopicSummary]
    research_home: str
    observed_at: _timestamp_pb2.Timestamp
    home_capabilities: _containers.RepeatedCompositeFieldContainer[ResearchCapability]
    home_capabilities_evidence: _evidence_pb2.Evidence
    def __init__(self, topics: _Optional[_Iterable[_Union[TopicSummary, _Mapping]]] = ..., research_home: _Optional[str] = ..., observed_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., home_capabilities: _Optional[_Iterable[_Union[ResearchCapability, _Mapping]]] = ..., home_capabilities_evidence: _Optional[_Union[_evidence_pb2.Evidence, _Mapping]] = ...) -> None: ...

class GetChangesRequest(_message.Message):
    __slots__ = ("subject", "id", "since")
    SUBJECT_FIELD_NUMBER: _ClassVar[int]
    ID_FIELD_NUMBER: _ClassVar[int]
    SINCE_FIELD_NUMBER: _ClassVar[int]
    subject: ChangeSubject
    id: str
    since: _timestamp_pb2.Timestamp
    def __init__(self, subject: _Optional[_Union[ChangeSubject, str]] = ..., id: _Optional[str] = ..., since: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ...) -> None: ...

class LedgerAddition(_message.Message):
    __slots__ = ("path", "text", "truncated", "from_dated_headings")
    PATH_FIELD_NUMBER: _ClassVar[int]
    TEXT_FIELD_NUMBER: _ClassVar[int]
    TRUNCATED_FIELD_NUMBER: _ClassVar[int]
    FROM_DATED_HEADINGS_FIELD_NUMBER: _ClassVar[int]
    path: str
    text: str
    truncated: bool
    from_dated_headings: bool
    def __init__(self, path: _Optional[str] = ..., text: _Optional[str] = ..., truncated: _Optional[bool] = ..., from_dated_headings: _Optional[bool] = ...) -> None: ...

class StatusChange(_message.Message):
    __slots__ = ("topic", "workstream", "before", "after")
    TOPIC_FIELD_NUMBER: _ClassVar[int]
    WORKSTREAM_FIELD_NUMBER: _ClassVar[int]
    BEFORE_FIELD_NUMBER: _ClassVar[int]
    AFTER_FIELD_NUMBER: _ClassVar[int]
    topic: str
    workstream: str
    before: str
    after: str
    def __init__(self, topic: _Optional[str] = ..., workstream: _Optional[str] = ..., before: _Optional[str] = ..., after: _Optional[str] = ...) -> None: ...

class GetChangesResponse(_message.Message):
    __slots__ = ("since", "ledger", "checkpoints", "showcase_lines", "status_changes", "files", "truncated", "baseline", "unreadable", "baseline_kind", "not_shown")
    SINCE_FIELD_NUMBER: _ClassVar[int]
    LEDGER_FIELD_NUMBER: _ClassVar[int]
    CHECKPOINTS_FIELD_NUMBER: _ClassVar[int]
    SHOWCASE_LINES_FIELD_NUMBER: _ClassVar[int]
    STATUS_CHANGES_FIELD_NUMBER: _ClassVar[int]
    FILES_FIELD_NUMBER: _ClassVar[int]
    TRUNCATED_FIELD_NUMBER: _ClassVar[int]
    BASELINE_FIELD_NUMBER: _ClassVar[int]
    UNREADABLE_FIELD_NUMBER: _ClassVar[int]
    BASELINE_KIND_FIELD_NUMBER: _ClassVar[int]
    NOT_SHOWN_FIELD_NUMBER: _ClassVar[int]
    since: _timestamp_pb2.Timestamp
    ledger: _containers.RepeatedCompositeFieldContainer[LedgerAddition]
    checkpoints: _containers.RepeatedCompositeFieldContainer[_evidence_pb2.FileFact]
    showcase_lines: _containers.RepeatedScalarFieldContainer[str]
    status_changes: _containers.RepeatedCompositeFieldContainer[StatusChange]
    files: _containers.RepeatedCompositeFieldContainer[_evidence_pb2.FileFact]
    truncated: bool
    baseline: str
    unreadable: _containers.RepeatedCompositeFieldContainer[_evidence_pb2.Evidence]
    baseline_kind: Baseline
    not_shown: _containers.RepeatedScalarFieldContainer[str]
    def __init__(self, since: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., ledger: _Optional[_Iterable[_Union[LedgerAddition, _Mapping]]] = ..., checkpoints: _Optional[_Iterable[_Union[_evidence_pb2.FileFact, _Mapping]]] = ..., showcase_lines: _Optional[_Iterable[str]] = ..., status_changes: _Optional[_Iterable[_Union[StatusChange, _Mapping]]] = ..., files: _Optional[_Iterable[_Union[_evidence_pb2.FileFact, _Mapping]]] = ..., truncated: _Optional[bool] = ..., baseline: _Optional[str] = ..., unreadable: _Optional[_Iterable[_Union[_evidence_pb2.Evidence, _Mapping]]] = ..., baseline_kind: _Optional[_Union[Baseline, str]] = ..., not_shown: _Optional[_Iterable[str]] = ...) -> None: ...
