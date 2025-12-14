import datetime

from google.protobuf import timestamp_pb2 as _timestamp_pb2
from browser_automation_studio.v1 import action_pb2 as _action_pb2
from browser_automation_studio.v1 import timeline_event_pb2 as _timeline_event_pb2
from google.protobuf.internal import containers as _containers
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Iterable as _Iterable, Mapping as _Mapping
from typing import ClassVar as _ClassVar, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class RecordingState(_message.Message):
    __slots__ = ()
    IS_RECORDING_FIELD_NUMBER: _ClassVar[int]
    RECORDING_ID_FIELD_NUMBER: _ClassVar[int]
    SESSION_ID_FIELD_NUMBER: _ClassVar[int]
    ACTION_COUNT_FIELD_NUMBER: _ClassVar[int]
    STARTED_AT_FIELD_NUMBER: _ClassVar[int]
    is_recording: bool
    recording_id: str
    session_id: str
    action_count: int
    started_at: _timestamp_pb2.Timestamp
    def __init__(self, is_recording: _Optional[bool] = ..., recording_id: _Optional[str] = ..., session_id: _Optional[str] = ..., action_count: _Optional[int] = ..., started_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ...) -> None: ...

class CreateRecordingSessionRequest(_message.Message):
    __slots__ = ()
    VIEWPORT_WIDTH_FIELD_NUMBER: _ClassVar[int]
    VIEWPORT_HEIGHT_FIELD_NUMBER: _ClassVar[int]
    INITIAL_URL_FIELD_NUMBER: _ClassVar[int]
    viewport_width: int
    viewport_height: int
    initial_url: str
    def __init__(self, viewport_width: _Optional[int] = ..., viewport_height: _Optional[int] = ..., initial_url: _Optional[str] = ...) -> None: ...

class CreateRecordingSessionResponse(_message.Message):
    __slots__ = ()
    SESSION_ID_FIELD_NUMBER: _ClassVar[int]
    CREATED_AT_FIELD_NUMBER: _ClassVar[int]
    session_id: str
    created_at: _timestamp_pb2.Timestamp
    def __init__(self, session_id: _Optional[str] = ..., created_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ...) -> None: ...

class StartRecordingRequest(_message.Message):
    __slots__ = ()
    SESSION_ID_FIELD_NUMBER: _ClassVar[int]
    CALLBACK_URL_FIELD_NUMBER: _ClassVar[int]
    session_id: str
    callback_url: str
    def __init__(self, session_id: _Optional[str] = ..., callback_url: _Optional[str] = ...) -> None: ...

class StartRecordingResponse(_message.Message):
    __slots__ = ()
    RECORDING_ID_FIELD_NUMBER: _ClassVar[int]
    SESSION_ID_FIELD_NUMBER: _ClassVar[int]
    STARTED_AT_FIELD_NUMBER: _ClassVar[int]
    recording_id: str
    session_id: str
    started_at: _timestamp_pb2.Timestamp
    def __init__(self, recording_id: _Optional[str] = ..., session_id: _Optional[str] = ..., started_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ...) -> None: ...

class StopRecordingResponse(_message.Message):
    __slots__ = ()
    RECORDING_ID_FIELD_NUMBER: _ClassVar[int]
    SESSION_ID_FIELD_NUMBER: _ClassVar[int]
    ACTION_COUNT_FIELD_NUMBER: _ClassVar[int]
    STOPPED_AT_FIELD_NUMBER: _ClassVar[int]
    recording_id: str
    session_id: str
    action_count: int
    stopped_at: _timestamp_pb2.Timestamp
    def __init__(self, recording_id: _Optional[str] = ..., session_id: _Optional[str] = ..., action_count: _Optional[int] = ..., stopped_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ...) -> None: ...

class RecordingStatusResponse(_message.Message):
    __slots__ = ()
    SESSION_ID_FIELD_NUMBER: _ClassVar[int]
    IS_RECORDING_FIELD_NUMBER: _ClassVar[int]
    RECORDING_ID_FIELD_NUMBER: _ClassVar[int]
    ACTION_COUNT_FIELD_NUMBER: _ClassVar[int]
    STARTED_AT_FIELD_NUMBER: _ClassVar[int]
    session_id: str
    is_recording: bool
    recording_id: str
    action_count: int
    started_at: _timestamp_pb2.Timestamp
    def __init__(self, session_id: _Optional[str] = ..., is_recording: _Optional[bool] = ..., recording_id: _Optional[str] = ..., action_count: _Optional[int] = ..., started_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ...) -> None: ...

class GetActionsResponse(_message.Message):
    __slots__ = ()
    SESSION_ID_FIELD_NUMBER: _ClassVar[int]
    EVENTS_FIELD_NUMBER: _ClassVar[int]
    COUNT_FIELD_NUMBER: _ClassVar[int]
    session_id: str
    events: _containers.RepeatedCompositeFieldContainer[_timeline_event_pb2.TimelineEvent]
    count: int
    def __init__(self, session_id: _Optional[str] = ..., events: _Optional[_Iterable[_Union[_timeline_event_pb2.TimelineEvent, _Mapping]]] = ..., count: _Optional[int] = ...) -> None: ...

class GenerateWorkflowRequest(_message.Message):
    __slots__ = ()
    class EventRange(_message.Message):
        __slots__ = ()
        START_FIELD_NUMBER: _ClassVar[int]
        END_FIELD_NUMBER: _ClassVar[int]
        start: int
        end: int
        def __init__(self, start: _Optional[int] = ..., end: _Optional[int] = ...) -> None: ...
    SESSION_ID_FIELD_NUMBER: _ClassVar[int]
    NAME_FIELD_NUMBER: _ClassVar[int]
    PROJECT_ID_FIELD_NUMBER: _ClassVar[int]
    PROJECT_NAME_FIELD_NUMBER: _ClassVar[int]
    EVENT_RANGE_FIELD_NUMBER: _ClassVar[int]
    EVENTS_FIELD_NUMBER: _ClassVar[int]
    session_id: str
    name: str
    project_id: str
    project_name: str
    event_range: GenerateWorkflowRequest.EventRange
    events: _containers.RepeatedCompositeFieldContainer[_timeline_event_pb2.TimelineEvent]
    def __init__(self, session_id: _Optional[str] = ..., name: _Optional[str] = ..., project_id: _Optional[str] = ..., project_name: _Optional[str] = ..., event_range: _Optional[_Union[GenerateWorkflowRequest.EventRange, _Mapping]] = ..., events: _Optional[_Iterable[_Union[_timeline_event_pb2.TimelineEvent, _Mapping]]] = ...) -> None: ...

class GenerateWorkflowResponse(_message.Message):
    __slots__ = ()
    WORKFLOW_ID_FIELD_NUMBER: _ClassVar[int]
    PROJECT_ID_FIELD_NUMBER: _ClassVar[int]
    NAME_FIELD_NUMBER: _ClassVar[int]
    NODE_COUNT_FIELD_NUMBER: _ClassVar[int]
    EVENT_COUNT_FIELD_NUMBER: _ClassVar[int]
    workflow_id: str
    project_id: str
    name: str
    node_count: int
    event_count: int
    def __init__(self, workflow_id: _Optional[str] = ..., project_id: _Optional[str] = ..., name: _Optional[str] = ..., node_count: _Optional[int] = ..., event_count: _Optional[int] = ...) -> None: ...

class ReplayPreviewRequest(_message.Message):
    __slots__ = ()
    SESSION_ID_FIELD_NUMBER: _ClassVar[int]
    EVENTS_FIELD_NUMBER: _ClassVar[int]
    LIMIT_FIELD_NUMBER: _ClassVar[int]
    STOP_ON_FAILURE_FIELD_NUMBER: _ClassVar[int]
    ACTION_TIMEOUT_MS_FIELD_NUMBER: _ClassVar[int]
    session_id: str
    events: _containers.RepeatedCompositeFieldContainer[_timeline_event_pb2.TimelineEvent]
    limit: int
    stop_on_failure: bool
    action_timeout_ms: int
    def __init__(self, session_id: _Optional[str] = ..., events: _Optional[_Iterable[_Union[_timeline_event_pb2.TimelineEvent, _Mapping]]] = ..., limit: _Optional[int] = ..., stop_on_failure: _Optional[bool] = ..., action_timeout_ms: _Optional[int] = ...) -> None: ...

class ReplayEventError(_message.Message):
    __slots__ = ()
    MESSAGE_FIELD_NUMBER: _ClassVar[int]
    CODE_FIELD_NUMBER: _ClassVar[int]
    MATCH_COUNT_FIELD_NUMBER: _ClassVar[int]
    SELECTOR_FIELD_NUMBER: _ClassVar[int]
    message: str
    code: str
    match_count: int
    selector: str
    def __init__(self, message: _Optional[str] = ..., code: _Optional[str] = ..., match_count: _Optional[int] = ..., selector: _Optional[str] = ...) -> None: ...

class ReplayEventResult(_message.Message):
    __slots__ = ()
    EVENT_ID_FIELD_NUMBER: _ClassVar[int]
    SEQUENCE_NUM_FIELD_NUMBER: _ClassVar[int]
    SUCCESS_FIELD_NUMBER: _ClassVar[int]
    DURATION_MS_FIELD_NUMBER: _ClassVar[int]
    ERROR_FIELD_NUMBER: _ClassVar[int]
    SCREENSHOT_ON_ERROR_FIELD_NUMBER: _ClassVar[int]
    ACTION_FIELD_NUMBER: _ClassVar[int]
    event_id: str
    sequence_num: int
    success: bool
    duration_ms: int
    error: ReplayEventError
    screenshot_on_error: str
    action: _action_pb2.ActionDefinition
    def __init__(self, event_id: _Optional[str] = ..., sequence_num: _Optional[int] = ..., success: _Optional[bool] = ..., duration_ms: _Optional[int] = ..., error: _Optional[_Union[ReplayEventError, _Mapping]] = ..., screenshot_on_error: _Optional[str] = ..., action: _Optional[_Union[_action_pb2.ActionDefinition, _Mapping]] = ...) -> None: ...

class ReplayPreviewResponse(_message.Message):
    __slots__ = ()
    SUCCESS_FIELD_NUMBER: _ClassVar[int]
    TOTAL_EVENTS_FIELD_NUMBER: _ClassVar[int]
    PASSED_EVENTS_FIELD_NUMBER: _ClassVar[int]
    FAILED_EVENTS_FIELD_NUMBER: _ClassVar[int]
    RESULTS_FIELD_NUMBER: _ClassVar[int]
    TOTAL_DURATION_MS_FIELD_NUMBER: _ClassVar[int]
    STOPPED_EARLY_FIELD_NUMBER: _ClassVar[int]
    success: bool
    total_events: int
    passed_events: int
    failed_events: int
    results: _containers.RepeatedCompositeFieldContainer[ReplayEventResult]
    total_duration_ms: int
    stopped_early: bool
    def __init__(self, success: _Optional[bool] = ..., total_events: _Optional[int] = ..., passed_events: _Optional[int] = ..., failed_events: _Optional[int] = ..., results: _Optional[_Iterable[_Union[ReplayEventResult, _Mapping]]] = ..., total_duration_ms: _Optional[int] = ..., stopped_early: _Optional[bool] = ...) -> None: ...

class SelectorValidation(_message.Message):
    __slots__ = ()
    VALID_FIELD_NUMBER: _ClassVar[int]
    MATCH_COUNT_FIELD_NUMBER: _ClassVar[int]
    SELECTOR_FIELD_NUMBER: _ClassVar[int]
    ERROR_FIELD_NUMBER: _ClassVar[int]
    valid: bool
    match_count: int
    selector: str
    error: str
    def __init__(self, valid: _Optional[bool] = ..., match_count: _Optional[int] = ..., selector: _Optional[str] = ..., error: _Optional[str] = ...) -> None: ...
