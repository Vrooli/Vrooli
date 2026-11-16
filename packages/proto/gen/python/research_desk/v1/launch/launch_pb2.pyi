import datetime

from google.protobuf import timestamp_pb2 as _timestamp_pb2
from research_desk.v1.activity import activity_pb2 as _activity_pb2
from research_desk.v1.shared import commission_pb2 as _commission_pb2
from research_desk.v1.shared import evidence_pb2 as _evidence_pb2
from google.protobuf.internal import containers as _containers
from google.protobuf.internal import enum_type_wrapper as _enum_type_wrapper
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Iterable as _Iterable, Mapping as _Mapping
from typing import ClassVar as _ClassVar, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class Resolution(int, metaclass=_enum_type_wrapper.EnumTypeWrapper):
    __slots__ = ()
    RESOLUTION_UNSPECIFIED: _ClassVar[Resolution]
    RESOLUTION_SUPERSEDE: _ClassVar[Resolution]
    RESOLUTION_RUN_ALONGSIDE: _ClassVar[Resolution]

class ControlAction(int, metaclass=_enum_type_wrapper.EnumTypeWrapper):
    __slots__ = ()
    CONTROL_ACTION_UNSPECIFIED: _ClassVar[ControlAction]
    CONTROL_ACTION_PAUSE: _ClassVar[ControlAction]
    CONTROL_ACTION_RESUME: _ClassVar[ControlAction]
    CONTROL_ACTION_STOP: _ClassVar[ControlAction]
    CONTROL_ACTION_DISCARD: _ClassVar[ControlAction]
    CONTROL_ACTION_RESEND_STANCE: _ClassVar[ControlAction]
RESOLUTION_UNSPECIFIED: Resolution
RESOLUTION_SUPERSEDE: Resolution
RESOLUTION_RUN_ALONGSIDE: Resolution
CONTROL_ACTION_UNSPECIFIED: ControlAction
CONTROL_ACTION_PAUSE: ControlAction
CONTROL_ACTION_RESUME: ControlAction
CONTROL_ACTION_STOP: ControlAction
CONTROL_ACTION_DISCARD: ControlAction
CONTROL_ACTION_RESEND_STANCE: ControlAction

class GetLaunchOptionsRequest(_message.Message):
    __slots__ = ("topics",)
    TOPICS_FIELD_NUMBER: _ClassVar[int]
    topics: _containers.RepeatedScalarFieldContainer[str]
    def __init__(self, topics: _Optional[_Iterable[str]] = ...) -> None: ...

class SuggestedQuote(_message.Message):
    __slots__ = ("text", "source", "commission_id")
    TEXT_FIELD_NUMBER: _ClassVar[int]
    SOURCE_FIELD_NUMBER: _ClassVar[int]
    COMMISSION_ID_FIELD_NUMBER: _ClassVar[int]
    text: str
    source: str
    commission_id: str
    def __init__(self, text: _Optional[str] = ..., source: _Optional[str] = ..., commission_id: _Optional[str] = ...) -> None: ...

class TopicWorkstreams(_message.Message):
    __slots__ = ("topic", "workstreams", "evidence")
    TOPIC_FIELD_NUMBER: _ClassVar[int]
    WORKSTREAMS_FIELD_NUMBER: _ClassVar[int]
    EVIDENCE_FIELD_NUMBER: _ClassVar[int]
    topic: str
    workstreams: _containers.RepeatedCompositeFieldContainer[_commission_pb2.Workstream]
    evidence: _evidence_pb2.Evidence
    def __init__(self, topic: _Optional[str] = ..., workstreams: _Optional[_Iterable[_Union[_commission_pb2.Workstream, _Mapping]]] = ..., evidence: _Optional[_Union[_evidence_pb2.Evidence, _Mapping]] = ...) -> None: ...

class GetLaunchOptionsResponse(_message.Message):
    __slots__ = ("limits_path", "default_supervisor_model", "default_supervisor_effort", "default_owner_model_tier", "supervisor_models", "supervisor_efforts", "owner_model_tiers", "charter_bytes_limit", "stance_chars_limit", "kernel_chars_limit", "automatic_available", "automatic_reason", "web_console_access", "stance_quotes", "stance_quotes_reason", "topic_workstreams", "web_console_access_kind", "packet_limits")
    LIMITS_PATH_FIELD_NUMBER: _ClassVar[int]
    DEFAULT_SUPERVISOR_MODEL_FIELD_NUMBER: _ClassVar[int]
    DEFAULT_SUPERVISOR_EFFORT_FIELD_NUMBER: _ClassVar[int]
    DEFAULT_OWNER_MODEL_TIER_FIELD_NUMBER: _ClassVar[int]
    SUPERVISOR_MODELS_FIELD_NUMBER: _ClassVar[int]
    SUPERVISOR_EFFORTS_FIELD_NUMBER: _ClassVar[int]
    OWNER_MODEL_TIERS_FIELD_NUMBER: _ClassVar[int]
    CHARTER_BYTES_LIMIT_FIELD_NUMBER: _ClassVar[int]
    STANCE_CHARS_LIMIT_FIELD_NUMBER: _ClassVar[int]
    KERNEL_CHARS_LIMIT_FIELD_NUMBER: _ClassVar[int]
    AUTOMATIC_AVAILABLE_FIELD_NUMBER: _ClassVar[int]
    AUTOMATIC_REASON_FIELD_NUMBER: _ClassVar[int]
    WEB_CONSOLE_ACCESS_FIELD_NUMBER: _ClassVar[int]
    STANCE_QUOTES_FIELD_NUMBER: _ClassVar[int]
    STANCE_QUOTES_REASON_FIELD_NUMBER: _ClassVar[int]
    TOPIC_WORKSTREAMS_FIELD_NUMBER: _ClassVar[int]
    WEB_CONSOLE_ACCESS_KIND_FIELD_NUMBER: _ClassVar[int]
    PACKET_LIMITS_FIELD_NUMBER: _ClassVar[int]
    limits_path: str
    default_supervisor_model: str
    default_supervisor_effort: str
    default_owner_model_tier: str
    supervisor_models: _containers.RepeatedScalarFieldContainer[str]
    supervisor_efforts: _containers.RepeatedScalarFieldContainer[str]
    owner_model_tiers: _containers.RepeatedScalarFieldContainer[str]
    charter_bytes_limit: int
    stance_chars_limit: int
    kernel_chars_limit: int
    automatic_available: bool
    automatic_reason: str
    web_console_access: str
    stance_quotes: _containers.RepeatedCompositeFieldContainer[SuggestedQuote]
    stance_quotes_reason: str
    topic_workstreams: _containers.RepeatedCompositeFieldContainer[TopicWorkstreams]
    web_console_access_kind: _activity_pb2.WebConsoleAccess
    packet_limits: PacketLimits
    def __init__(self, limits_path: _Optional[str] = ..., default_supervisor_model: _Optional[str] = ..., default_supervisor_effort: _Optional[str] = ..., default_owner_model_tier: _Optional[str] = ..., supervisor_models: _Optional[_Iterable[str]] = ..., supervisor_efforts: _Optional[_Iterable[str]] = ..., owner_model_tiers: _Optional[_Iterable[str]] = ..., charter_bytes_limit: _Optional[int] = ..., stance_chars_limit: _Optional[int] = ..., kernel_chars_limit: _Optional[int] = ..., automatic_available: _Optional[bool] = ..., automatic_reason: _Optional[str] = ..., web_console_access: _Optional[str] = ..., stance_quotes: _Optional[_Iterable[_Union[SuggestedQuote, _Mapping]]] = ..., stance_quotes_reason: _Optional[str] = ..., topic_workstreams: _Optional[_Iterable[_Union[TopicWorkstreams, _Mapping]]] = ..., web_console_access_kind: _Optional[_Union[_activity_pb2.WebConsoleAccess, str]] = ..., packet_limits: _Optional[_Union[PacketLimits, _Mapping]] = ...) -> None: ...

class PacketLimits(_message.Message):
    __slots__ = ("topics_max", "allowance_hours_min", "allowance_hours_max", "allowance_hours_default", "ordered_workstreams_max")
    TOPICS_MAX_FIELD_NUMBER: _ClassVar[int]
    ALLOWANCE_HOURS_MIN_FIELD_NUMBER: _ClassVar[int]
    ALLOWANCE_HOURS_MAX_FIELD_NUMBER: _ClassVar[int]
    ALLOWANCE_HOURS_DEFAULT_FIELD_NUMBER: _ClassVar[int]
    ORDERED_WORKSTREAMS_MAX_FIELD_NUMBER: _ClassVar[int]
    topics_max: int
    allowance_hours_min: int
    allowance_hours_max: int
    allowance_hours_default: int
    ordered_workstreams_max: int
    def __init__(self, topics_max: _Optional[int] = ..., allowance_hours_min: _Optional[int] = ..., allowance_hours_max: _Optional[int] = ..., allowance_hours_default: _Optional[int] = ..., ordered_workstreams_max: _Optional[int] = ...) -> None: ...

class PacketResult(_message.Message):
    __slots__ = ("ok", "commission_id", "folder", "written", "gates", "cards", "texts", "errors", "exit_code")
    OK_FIELD_NUMBER: _ClassVar[int]
    COMMISSION_ID_FIELD_NUMBER: _ClassVar[int]
    FOLDER_FIELD_NUMBER: _ClassVar[int]
    WRITTEN_FIELD_NUMBER: _ClassVar[int]
    GATES_FIELD_NUMBER: _ClassVar[int]
    CARDS_FIELD_NUMBER: _ClassVar[int]
    TEXTS_FIELD_NUMBER: _ClassVar[int]
    ERRORS_FIELD_NUMBER: _ClassVar[int]
    EXIT_CODE_FIELD_NUMBER: _ClassVar[int]
    ok: bool
    commission_id: str
    folder: str
    written: bool
    gates: _containers.RepeatedCompositeFieldContainer[Gate]
    cards: _containers.RepeatedCompositeFieldContainer[Card]
    texts: Texts
    errors: _containers.RepeatedCompositeFieldContainer[PacketError]
    exit_code: int
    def __init__(self, ok: _Optional[bool] = ..., commission_id: _Optional[str] = ..., folder: _Optional[str] = ..., written: _Optional[bool] = ..., gates: _Optional[_Iterable[_Union[Gate, _Mapping]]] = ..., cards: _Optional[_Iterable[_Union[Card, _Mapping]]] = ..., texts: _Optional[_Union[Texts, _Mapping]] = ..., errors: _Optional[_Iterable[_Union[PacketError, _Mapping]]] = ..., exit_code: _Optional[int] = ...) -> None: ...

class Gate(_message.Message):
    __slots__ = ("name", "value", "limit", "ok")
    NAME_FIELD_NUMBER: _ClassVar[int]
    VALUE_FIELD_NUMBER: _ClassVar[int]
    LIMIT_FIELD_NUMBER: _ClassVar[int]
    OK_FIELD_NUMBER: _ClassVar[int]
    name: str
    value: str
    limit: str
    ok: bool
    def __init__(self, name: _Optional[str] = ..., value: _Optional[str] = ..., limit: _Optional[str] = ..., ok: _Optional[bool] = ...) -> None: ...

class Card(_message.Message):
    __slots__ = ("topic", "mode", "order", "order_source", "read_first")
    TOPIC_FIELD_NUMBER: _ClassVar[int]
    MODE_FIELD_NUMBER: _ClassVar[int]
    ORDER_FIELD_NUMBER: _ClassVar[int]
    ORDER_SOURCE_FIELD_NUMBER: _ClassVar[int]
    READ_FIRST_FIELD_NUMBER: _ClassVar[int]
    topic: str
    mode: str
    order: _containers.RepeatedScalarFieldContainer[str]
    order_source: str
    read_first: str
    def __init__(self, topic: _Optional[str] = ..., mode: _Optional[str] = ..., order: _Optional[_Iterable[str]] = ..., order_source: _Optional[str] = ..., read_first: _Optional[str] = ...) -> None: ...

class Texts(_message.Message):
    __slots__ = ("charter", "stance", "kernel")
    CHARTER_FIELD_NUMBER: _ClassVar[int]
    STANCE_FIELD_NUMBER: _ClassVar[int]
    KERNEL_FIELD_NUMBER: _ClassVar[int]
    charter: str
    stance: str
    kernel: str
    def __init__(self, charter: _Optional[str] = ..., stance: _Optional[str] = ..., kernel: _Optional[str] = ...) -> None: ...

class PacketError(_message.Message):
    __slots__ = ("code", "message")
    CODE_FIELD_NUMBER: _ClassVar[int]
    MESSAGE_FIELD_NUMBER: _ClassVar[int]
    code: str
    message: str
    def __init__(self, code: _Optional[str] = ..., message: _Optional[str] = ...) -> None: ...

class TopicVerdict(_message.Message):
    __slots__ = ("topic", "commission_id", "state", "blocks", "message", "read_first", "quiet", "choices", "recommended", "recommended_reason", "chosen")
    TOPIC_FIELD_NUMBER: _ClassVar[int]
    COMMISSION_ID_FIELD_NUMBER: _ClassVar[int]
    STATE_FIELD_NUMBER: _ClassVar[int]
    BLOCKS_FIELD_NUMBER: _ClassVar[int]
    MESSAGE_FIELD_NUMBER: _ClassVar[int]
    READ_FIRST_FIELD_NUMBER: _ClassVar[int]
    QUIET_FIELD_NUMBER: _ClassVar[int]
    CHOICES_FIELD_NUMBER: _ClassVar[int]
    RECOMMENDED_FIELD_NUMBER: _ClassVar[int]
    RECOMMENDED_REASON_FIELD_NUMBER: _ClassVar[int]
    CHOSEN_FIELD_NUMBER: _ClassVar[int]
    topic: str
    commission_id: str
    state: str
    blocks: bool
    message: str
    read_first: str
    quiet: bool
    choices: _containers.RepeatedScalarFieldContainer[Resolution]
    recommended: Resolution
    recommended_reason: str
    chosen: Resolution
    def __init__(self, topic: _Optional[str] = ..., commission_id: _Optional[str] = ..., state: _Optional[str] = ..., blocks: _Optional[bool] = ..., message: _Optional[str] = ..., read_first: _Optional[str] = ..., quiet: _Optional[bool] = ..., choices: _Optional[_Iterable[_Union[Resolution, str]]] = ..., recommended: _Optional[_Union[Resolution, str]] = ..., recommended_reason: _Optional[str] = ..., chosen: _Optional[_Union[Resolution, str]] = ...) -> None: ...

class PreflightRequest(_message.Message):
    __slots__ = ("spec_json", "commission_id", "supersedes", "assert_never_started", "run_alongside")
    SPEC_JSON_FIELD_NUMBER: _ClassVar[int]
    COMMISSION_ID_FIELD_NUMBER: _ClassVar[int]
    SUPERSEDES_FIELD_NUMBER: _ClassVar[int]
    ASSERT_NEVER_STARTED_FIELD_NUMBER: _ClassVar[int]
    RUN_ALONGSIDE_FIELD_NUMBER: _ClassVar[int]
    spec_json: str
    commission_id: str
    supersedes: _containers.RepeatedScalarFieldContainer[str]
    assert_never_started: bool
    run_alongside: _containers.RepeatedScalarFieldContainer[str]
    def __init__(self, spec_json: _Optional[str] = ..., commission_id: _Optional[str] = ..., supersedes: _Optional[_Iterable[str]] = ..., assert_never_started: _Optional[bool] = ..., run_alongside: _Optional[_Iterable[str]] = ...) -> None: ...

class PreflightResponse(_message.Message):
    __slots__ = ("packet", "topics", "launchable", "blockers", "never_started", "packet_state", "owner_reading")
    PACKET_FIELD_NUMBER: _ClassVar[int]
    TOPICS_FIELD_NUMBER: _ClassVar[int]
    LAUNCHABLE_FIELD_NUMBER: _ClassVar[int]
    BLOCKERS_FIELD_NUMBER: _ClassVar[int]
    NEVER_STARTED_FIELD_NUMBER: _ClassVar[int]
    PACKET_STATE_FIELD_NUMBER: _ClassVar[int]
    OWNER_READING_FIELD_NUMBER: _ClassVar[int]
    packet: PacketResult
    topics: _containers.RepeatedCompositeFieldContainer[TopicVerdict]
    launchable: bool
    blockers: _containers.RepeatedScalarFieldContainer[str]
    never_started: _commission_pb2.NeverStarted
    packet_state: StateExcerpt
    owner_reading: _containers.RepeatedCompositeFieldContainer[_commission_pb2.ReadSet]
    def __init__(self, packet: _Optional[_Union[PacketResult, _Mapping]] = ..., topics: _Optional[_Iterable[_Union[TopicVerdict, _Mapping]]] = ..., launchable: _Optional[bool] = ..., blockers: _Optional[_Iterable[str]] = ..., never_started: _Optional[_Union[_commission_pb2.NeverStarted, _Mapping]] = ..., packet_state: _Optional[_Union[StateExcerpt, _Mapping]] = ..., owner_reading: _Optional[_Iterable[_Union[_commission_pb2.ReadSet, _Mapping]]] = ...) -> None: ...

class StateExcerpt(_message.Message):
    __slots__ = ("path", "text", "truncated", "reason")
    PATH_FIELD_NUMBER: _ClassVar[int]
    TEXT_FIELD_NUMBER: _ClassVar[int]
    TRUNCATED_FIELD_NUMBER: _ClassVar[int]
    REASON_FIELD_NUMBER: _ClassVar[int]
    path: str
    text: str
    truncated: bool
    reason: str
    def __init__(self, path: _Optional[str] = ..., text: _Optional[str] = ..., truncated: _Optional[bool] = ..., reason: _Optional[str] = ...) -> None: ...

class BuildRequest(_message.Message):
    __slots__ = ("spec_json", "relaunch_of", "supersedes", "run_alongside")
    SPEC_JSON_FIELD_NUMBER: _ClassVar[int]
    RELAUNCH_OF_FIELD_NUMBER: _ClassVar[int]
    SUPERSEDES_FIELD_NUMBER: _ClassVar[int]
    RUN_ALONGSIDE_FIELD_NUMBER: _ClassVar[int]
    spec_json: str
    relaunch_of: str
    supersedes: _containers.RepeatedScalarFieldContainer[str]
    run_alongside: _containers.RepeatedScalarFieldContainer[str]
    def __init__(self, spec_json: _Optional[str] = ..., relaunch_of: _Optional[str] = ..., supersedes: _Optional[_Iterable[str]] = ..., run_alongside: _Optional[_Iterable[str]] = ...) -> None: ...

class BuildResponse(_message.Message):
    __slots__ = ("packet", "topics")
    PACKET_FIELD_NUMBER: _ClassVar[int]
    TOPICS_FIELD_NUMBER: _ClassVar[int]
    packet: PacketResult
    topics: _containers.RepeatedCompositeFieldContainer[TopicVerdict]
    def __init__(self, packet: _Optional[_Union[PacketResult, _Mapping]] = ..., topics: _Optional[_Iterable[_Union[TopicVerdict, _Mapping]]] = ...) -> None: ...

class LaunchRequest(_message.Message):
    __slots__ = ("commission_id", "supersedes", "assert_never_started", "run_alongside")
    COMMISSION_ID_FIELD_NUMBER: _ClassVar[int]
    SUPERSEDES_FIELD_NUMBER: _ClassVar[int]
    ASSERT_NEVER_STARTED_FIELD_NUMBER: _ClassVar[int]
    RUN_ALONGSIDE_FIELD_NUMBER: _ClassVar[int]
    commission_id: str
    supersedes: _containers.RepeatedScalarFieldContainer[str]
    assert_never_started: bool
    run_alongside: _containers.RepeatedScalarFieldContainer[str]
    def __init__(self, commission_id: _Optional[str] = ..., supersedes: _Optional[_Iterable[str]] = ..., assert_never_started: _Optional[bool] = ..., run_alongside: _Optional[_Iterable[str]] = ...) -> None: ...

class LaunchResponse(_message.Message):
    __slots__ = ("launch",)
    LAUNCH_FIELD_NUMBER: _ClassVar[int]
    launch: LaunchRecord
    def __init__(self, launch: _Optional[_Union[LaunchRecord, _Mapping]] = ...) -> None: ...

class GetLaunchMessagesRequest(_message.Message):
    __slots__ = ("commission_id",)
    COMMISSION_ID_FIELD_NUMBER: _ClassVar[int]
    commission_id: str
    def __init__(self, commission_id: _Optional[str] = ...) -> None: ...

class GetLaunchMessagesResponse(_message.Message):
    __slots__ = ("charter_path", "stance", "kernel")
    CHARTER_PATH_FIELD_NUMBER: _ClassVar[int]
    STANCE_FIELD_NUMBER: _ClassVar[int]
    KERNEL_FIELD_NUMBER: _ClassVar[int]
    charter_path: str
    stance: str
    kernel: str
    def __init__(self, charter_path: _Optional[str] = ..., stance: _Optional[str] = ..., kernel: _Optional[str] = ...) -> None: ...

class GetLaunchRequest(_message.Message):
    __slots__ = ("commission_id",)
    COMMISSION_ID_FIELD_NUMBER: _ClassVar[int]
    commission_id: str
    def __init__(self, commission_id: _Optional[str] = ...) -> None: ...

class GetLaunchResponse(_message.Message):
    __slots__ = ("launch",)
    LAUNCH_FIELD_NUMBER: _ClassVar[int]
    launch: LaunchRecord
    def __init__(self, launch: _Optional[_Union[LaunchRecord, _Mapping]] = ...) -> None: ...

class LaunchRecord(_message.Message):
    __slots__ = ("commission_id", "path", "launched_by", "web_console_session_id", "thread_id", "steps", "failed_step", "failed_reason", "discarded_at", "discarded_reason", "controls", "running", "supersedes", "run_alongside", "never_started", "superseded_by")
    COMMISSION_ID_FIELD_NUMBER: _ClassVar[int]
    PATH_FIELD_NUMBER: _ClassVar[int]
    LAUNCHED_BY_FIELD_NUMBER: _ClassVar[int]
    WEB_CONSOLE_SESSION_ID_FIELD_NUMBER: _ClassVar[int]
    THREAD_ID_FIELD_NUMBER: _ClassVar[int]
    STEPS_FIELD_NUMBER: _ClassVar[int]
    FAILED_STEP_FIELD_NUMBER: _ClassVar[int]
    FAILED_REASON_FIELD_NUMBER: _ClassVar[int]
    DISCARDED_AT_FIELD_NUMBER: _ClassVar[int]
    DISCARDED_REASON_FIELD_NUMBER: _ClassVar[int]
    CONTROLS_FIELD_NUMBER: _ClassVar[int]
    RUNNING_FIELD_NUMBER: _ClassVar[int]
    SUPERSEDES_FIELD_NUMBER: _ClassVar[int]
    RUN_ALONGSIDE_FIELD_NUMBER: _ClassVar[int]
    NEVER_STARTED_FIELD_NUMBER: _ClassVar[int]
    SUPERSEDED_BY_FIELD_NUMBER: _ClassVar[int]
    commission_id: str
    path: str
    launched_by: str
    web_console_session_id: str
    thread_id: str
    steps: _containers.RepeatedCompositeFieldContainer[LaunchStep]
    failed_step: str
    failed_reason: str
    discarded_at: _timestamp_pb2.Timestamp
    discarded_reason: str
    controls: _containers.RepeatedCompositeFieldContainer[ControlRecord]
    running: bool
    supersedes: _containers.RepeatedScalarFieldContainer[str]
    run_alongside: _containers.RepeatedScalarFieldContainer[str]
    never_started: NeverStartedAssertion
    superseded_by: str
    def __init__(self, commission_id: _Optional[str] = ..., path: _Optional[str] = ..., launched_by: _Optional[str] = ..., web_console_session_id: _Optional[str] = ..., thread_id: _Optional[str] = ..., steps: _Optional[_Iterable[_Union[LaunchStep, _Mapping]]] = ..., failed_step: _Optional[str] = ..., failed_reason: _Optional[str] = ..., discarded_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., discarded_reason: _Optional[str] = ..., controls: _Optional[_Iterable[_Union[ControlRecord, _Mapping]]] = ..., running: _Optional[bool] = ..., supersedes: _Optional[_Iterable[str]] = ..., run_alongside: _Optional[_Iterable[str]] = ..., never_started: _Optional[_Union[NeverStartedAssertion, _Mapping]] = ..., superseded_by: _Optional[str] = ...) -> None: ...

class NeverStartedAssertion(_message.Message):
    __slots__ = ("at", "evidence")
    AT_FIELD_NUMBER: _ClassVar[int]
    EVIDENCE_FIELD_NUMBER: _ClassVar[int]
    at: _timestamp_pb2.Timestamp
    evidence: str
    def __init__(self, at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., evidence: _Optional[str] = ...) -> None: ...

class LaunchStep(_message.Message):
    __slots__ = ("step", "at")
    STEP_FIELD_NUMBER: _ClassVar[int]
    AT_FIELD_NUMBER: _ClassVar[int]
    step: str
    at: _timestamp_pb2.Timestamp
    def __init__(self, step: _Optional[str] = ..., at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ...) -> None: ...

class ControlRecord(_message.Message):
    __slots__ = ("action", "at", "outcome")
    ACTION_FIELD_NUMBER: _ClassVar[int]
    AT_FIELD_NUMBER: _ClassVar[int]
    OUTCOME_FIELD_NUMBER: _ClassVar[int]
    action: str
    at: _timestamp_pb2.Timestamp
    outcome: str
    def __init__(self, action: _Optional[str] = ..., at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., outcome: _Optional[str] = ...) -> None: ...

class ControlRequest(_message.Message):
    __slots__ = ("commission_id", "action", "reason")
    COMMISSION_ID_FIELD_NUMBER: _ClassVar[int]
    ACTION_FIELD_NUMBER: _ClassVar[int]
    REASON_FIELD_NUMBER: _ClassVar[int]
    commission_id: str
    action: ControlAction
    reason: str
    def __init__(self, commission_id: _Optional[str] = ..., action: _Optional[_Union[ControlAction, str]] = ..., reason: _Optional[str] = ...) -> None: ...

class ControlResponse(_message.Message):
    __slots__ = ("launch", "goal_status", "message")
    LAUNCH_FIELD_NUMBER: _ClassVar[int]
    GOAL_STATUS_FIELD_NUMBER: _ClassVar[int]
    MESSAGE_FIELD_NUMBER: _ClassVar[int]
    launch: LaunchRecord
    goal_status: str
    message: str
    def __init__(self, launch: _Optional[_Union[LaunchRecord, _Mapping]] = ..., goal_status: _Optional[str] = ..., message: _Optional[str] = ...) -> None: ...

class TestLaunchRequest(_message.Message):
    __slots__ = ()
    def __init__(self) -> None: ...

class GetLaunchTestRequest(_message.Message):
    __slots__ = ()
    def __init__(self) -> None: ...

class LaunchTestReport(_message.Message):
    __slots__ = ("session_id", "codex_home", "steps", "kernel_chars", "status_after_goal", "status_after_pause", "status_after_resume", "status_final", "stopped", "failed_step", "failed_reason", "report_path", "running", "at")
    SESSION_ID_FIELD_NUMBER: _ClassVar[int]
    CODEX_HOME_FIELD_NUMBER: _ClassVar[int]
    STEPS_FIELD_NUMBER: _ClassVar[int]
    KERNEL_CHARS_FIELD_NUMBER: _ClassVar[int]
    STATUS_AFTER_GOAL_FIELD_NUMBER: _ClassVar[int]
    STATUS_AFTER_PAUSE_FIELD_NUMBER: _ClassVar[int]
    STATUS_AFTER_RESUME_FIELD_NUMBER: _ClassVar[int]
    STATUS_FINAL_FIELD_NUMBER: _ClassVar[int]
    STOPPED_FIELD_NUMBER: _ClassVar[int]
    FAILED_STEP_FIELD_NUMBER: _ClassVar[int]
    FAILED_REASON_FIELD_NUMBER: _ClassVar[int]
    REPORT_PATH_FIELD_NUMBER: _ClassVar[int]
    RUNNING_FIELD_NUMBER: _ClassVar[int]
    AT_FIELD_NUMBER: _ClassVar[int]
    session_id: str
    codex_home: str
    steps: _containers.RepeatedCompositeFieldContainer[LaunchStep]
    kernel_chars: int
    status_after_goal: str
    status_after_pause: str
    status_after_resume: str
    status_final: str
    stopped: bool
    failed_step: str
    failed_reason: str
    report_path: str
    running: bool
    at: _timestamp_pb2.Timestamp
    def __init__(self, session_id: _Optional[str] = ..., codex_home: _Optional[str] = ..., steps: _Optional[_Iterable[_Union[LaunchStep, _Mapping]]] = ..., kernel_chars: _Optional[int] = ..., status_after_goal: _Optional[str] = ..., status_after_pause: _Optional[str] = ..., status_after_resume: _Optional[str] = ..., status_final: _Optional[str] = ..., stopped: _Optional[bool] = ..., failed_step: _Optional[str] = ..., failed_reason: _Optional[str] = ..., report_path: _Optional[str] = ..., running: _Optional[bool] = ..., at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ...) -> None: ...
