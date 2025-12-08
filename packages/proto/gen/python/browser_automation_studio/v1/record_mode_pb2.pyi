import datetime

from google.protobuf import struct_pb2 as _struct_pb2
from google.protobuf import timestamp_pb2 as _timestamp_pb2
from browser_automation_studio.v1 import shared_pb2 as _shared_pb2
from google.protobuf.internal import containers as _containers
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Iterable as _Iterable, Mapping as _Mapping
from typing import ClassVar as _ClassVar, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class SelectorCandidate(_message.Message):
    __slots__ = ()
    TYPE_FIELD_NUMBER: _ClassVar[int]
    VALUE_FIELD_NUMBER: _ClassVar[int]
    CONFIDENCE_FIELD_NUMBER: _ClassVar[int]
    SPECIFICITY_FIELD_NUMBER: _ClassVar[int]
    type: str
    value: str
    confidence: float
    specificity: int
    def __init__(self, type: _Optional[str] = ..., value: _Optional[str] = ..., confidence: _Optional[float] = ..., specificity: _Optional[int] = ...) -> None: ...

class SelectorSet(_message.Message):
    __slots__ = ()
    PRIMARY_FIELD_NUMBER: _ClassVar[int]
    CANDIDATES_FIELD_NUMBER: _ClassVar[int]
    primary: str
    candidates: _containers.RepeatedCompositeFieldContainer[SelectorCandidate]
    def __init__(self, primary: _Optional[str] = ..., candidates: _Optional[_Iterable[_Union[SelectorCandidate, _Mapping]]] = ...) -> None: ...

class ElementMeta(_message.Message):
    __slots__ = ()
    class AttributesEntry(_message.Message):
        __slots__ = ()
        KEY_FIELD_NUMBER: _ClassVar[int]
        VALUE_FIELD_NUMBER: _ClassVar[int]
        key: str
        value: str
        def __init__(self, key: _Optional[str] = ..., value: _Optional[str] = ...) -> None: ...
    TAG_NAME_FIELD_NUMBER: _ClassVar[int]
    ID_FIELD_NUMBER: _ClassVar[int]
    CLASS_NAME_FIELD_NUMBER: _ClassVar[int]
    INNER_TEXT_FIELD_NUMBER: _ClassVar[int]
    ATTRIBUTES_FIELD_NUMBER: _ClassVar[int]
    IS_VISIBLE_FIELD_NUMBER: _ClassVar[int]
    IS_ENABLED_FIELD_NUMBER: _ClassVar[int]
    ROLE_FIELD_NUMBER: _ClassVar[int]
    ARIA_LABEL_FIELD_NUMBER: _ClassVar[int]
    tag_name: str
    id: str
    class_name: str
    inner_text: str
    attributes: _containers.ScalarMap[str, str]
    is_visible: bool
    is_enabled: bool
    role: str
    aria_label: str
    def __init__(self, tag_name: _Optional[str] = ..., id: _Optional[str] = ..., class_name: _Optional[str] = ..., inner_text: _Optional[str] = ..., attributes: _Optional[_Mapping[str, str]] = ..., is_visible: _Optional[bool] = ..., is_enabled: _Optional[bool] = ..., role: _Optional[str] = ..., aria_label: _Optional[str] = ...) -> None: ...

class RecordBoundingBox(_message.Message):
    __slots__ = ()
    X_FIELD_NUMBER: _ClassVar[int]
    Y_FIELD_NUMBER: _ClassVar[int]
    WIDTH_FIELD_NUMBER: _ClassVar[int]
    HEIGHT_FIELD_NUMBER: _ClassVar[int]
    x: float
    y: float
    width: float
    height: float
    def __init__(self, x: _Optional[float] = ..., y: _Optional[float] = ..., width: _Optional[float] = ..., height: _Optional[float] = ...) -> None: ...

class RecordPoint(_message.Message):
    __slots__ = ()
    X_FIELD_NUMBER: _ClassVar[int]
    Y_FIELD_NUMBER: _ClassVar[int]
    x: float
    y: float
    def __init__(self, x: _Optional[float] = ..., y: _Optional[float] = ...) -> None: ...

class RecordedAction(_message.Message):
    __slots__ = ()
    ID_FIELD_NUMBER: _ClassVar[int]
    SESSION_ID_FIELD_NUMBER: _ClassVar[int]
    SEQUENCE_NUM_FIELD_NUMBER: _ClassVar[int]
    TIMESTAMP_FIELD_NUMBER: _ClassVar[int]
    DURATION_MS_FIELD_NUMBER: _ClassVar[int]
    ACTION_TYPE_FIELD_NUMBER: _ClassVar[int]
    CONFIDENCE_FIELD_NUMBER: _ClassVar[int]
    SELECTOR_FIELD_NUMBER: _ClassVar[int]
    ELEMENT_META_FIELD_NUMBER: _ClassVar[int]
    BOUNDING_BOX_FIELD_NUMBER: _ClassVar[int]
    PAYLOAD_FIELD_NUMBER: _ClassVar[int]
    PAYLOAD_TYPED_FIELD_NUMBER: _ClassVar[int]
    URL_FIELD_NUMBER: _ClassVar[int]
    FRAME_ID_FIELD_NUMBER: _ClassVar[int]
    CURSOR_POS_FIELD_NUMBER: _ClassVar[int]
    id: str
    session_id: str
    sequence_num: int
    timestamp: _timestamp_pb2.Timestamp
    duration_ms: int
    action_type: str
    confidence: float
    selector: SelectorSet
    element_meta: ElementMeta
    bounding_box: RecordBoundingBox
    payload: _struct_pb2.Struct
    payload_typed: _shared_pb2.JsonObject
    url: str
    frame_id: str
    cursor_pos: RecordPoint
    def __init__(self, id: _Optional[str] = ..., session_id: _Optional[str] = ..., sequence_num: _Optional[int] = ..., timestamp: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., duration_ms: _Optional[int] = ..., action_type: _Optional[str] = ..., confidence: _Optional[float] = ..., selector: _Optional[_Union[SelectorSet, _Mapping]] = ..., element_meta: _Optional[_Union[ElementMeta, _Mapping]] = ..., bounding_box: _Optional[_Union[RecordBoundingBox, _Mapping]] = ..., payload: _Optional[_Union[_struct_pb2.Struct, _Mapping]] = ..., payload_typed: _Optional[_Union[_shared_pb2.JsonObject, _Mapping]] = ..., url: _Optional[str] = ..., frame_id: _Optional[str] = ..., cursor_pos: _Optional[_Union[RecordPoint, _Mapping]] = ...) -> None: ...

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
    ACTIONS_FIELD_NUMBER: _ClassVar[int]
    COUNT_FIELD_NUMBER: _ClassVar[int]
    session_id: str
    actions: _containers.RepeatedCompositeFieldContainer[RecordedAction]
    count: int
    def __init__(self, session_id: _Optional[str] = ..., actions: _Optional[_Iterable[_Union[RecordedAction, _Mapping]]] = ..., count: _Optional[int] = ...) -> None: ...

class GenerateWorkflowRequest(_message.Message):
    __slots__ = ()
    class ActionRange(_message.Message):
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
    ACTION_RANGE_FIELD_NUMBER: _ClassVar[int]
    ACTIONS_FIELD_NUMBER: _ClassVar[int]
    session_id: str
    name: str
    project_id: str
    project_name: str
    action_range: GenerateWorkflowRequest.ActionRange
    actions: _containers.RepeatedCompositeFieldContainer[RecordedAction]
    def __init__(self, session_id: _Optional[str] = ..., name: _Optional[str] = ..., project_id: _Optional[str] = ..., project_name: _Optional[str] = ..., action_range: _Optional[_Union[GenerateWorkflowRequest.ActionRange, _Mapping]] = ..., actions: _Optional[_Iterable[_Union[RecordedAction, _Mapping]]] = ...) -> None: ...

class GenerateWorkflowResponse(_message.Message):
    __slots__ = ()
    WORKFLOW_ID_FIELD_NUMBER: _ClassVar[int]
    PROJECT_ID_FIELD_NUMBER: _ClassVar[int]
    NAME_FIELD_NUMBER: _ClassVar[int]
    NODE_COUNT_FIELD_NUMBER: _ClassVar[int]
    ACTION_COUNT_FIELD_NUMBER: _ClassVar[int]
    workflow_id: str
    project_id: str
    name: str
    node_count: int
    action_count: int
    def __init__(self, workflow_id: _Optional[str] = ..., project_id: _Optional[str] = ..., name: _Optional[str] = ..., node_count: _Optional[int] = ..., action_count: _Optional[int] = ...) -> None: ...

class ReplayPreviewRequest(_message.Message):
    __slots__ = ()
    SESSION_ID_FIELD_NUMBER: _ClassVar[int]
    ACTIONS_FIELD_NUMBER: _ClassVar[int]
    LIMIT_FIELD_NUMBER: _ClassVar[int]
    STOP_ON_FAILURE_FIELD_NUMBER: _ClassVar[int]
    ACTION_TIMEOUT_FIELD_NUMBER: _ClassVar[int]
    session_id: str
    actions: _containers.RepeatedCompositeFieldContainer[RecordedAction]
    limit: int
    stop_on_failure: bool
    action_timeout: int
    def __init__(self, session_id: _Optional[str] = ..., actions: _Optional[_Iterable[_Union[RecordedAction, _Mapping]]] = ..., limit: _Optional[int] = ..., stop_on_failure: _Optional[bool] = ..., action_timeout: _Optional[int] = ...) -> None: ...

class ActionReplayError(_message.Message):
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

class ActionReplayResult(_message.Message):
    __slots__ = ()
    ACTION_ID_FIELD_NUMBER: _ClassVar[int]
    SEQUENCE_NUM_FIELD_NUMBER: _ClassVar[int]
    ACTION_TYPE_FIELD_NUMBER: _ClassVar[int]
    SUCCESS_FIELD_NUMBER: _ClassVar[int]
    DURATION_MS_FIELD_NUMBER: _ClassVar[int]
    ERROR_FIELD_NUMBER: _ClassVar[int]
    SCREENSHOT_ON_ERROR_FIELD_NUMBER: _ClassVar[int]
    action_id: str
    sequence_num: int
    action_type: str
    success: bool
    duration_ms: int
    error: ActionReplayError
    screenshot_on_error: str
    def __init__(self, action_id: _Optional[str] = ..., sequence_num: _Optional[int] = ..., action_type: _Optional[str] = ..., success: _Optional[bool] = ..., duration_ms: _Optional[int] = ..., error: _Optional[_Union[ActionReplayError, _Mapping]] = ..., screenshot_on_error: _Optional[str] = ...) -> None: ...

class ReplayPreviewResponse(_message.Message):
    __slots__ = ()
    SUCCESS_FIELD_NUMBER: _ClassVar[int]
    TOTAL_ACTIONS_FIELD_NUMBER: _ClassVar[int]
    PASSED_ACTIONS_FIELD_NUMBER: _ClassVar[int]
    FAILED_ACTIONS_FIELD_NUMBER: _ClassVar[int]
    RESULTS_FIELD_NUMBER: _ClassVar[int]
    TOTAL_DURATION_MS_FIELD_NUMBER: _ClassVar[int]
    STOPPED_EARLY_FIELD_NUMBER: _ClassVar[int]
    success: bool
    total_actions: int
    passed_actions: int
    failed_actions: int
    results: _containers.RepeatedCompositeFieldContainer[ActionReplayResult]
    total_duration_ms: int
    stopped_early: bool
    def __init__(self, success: _Optional[bool] = ..., total_actions: _Optional[int] = ..., passed_actions: _Optional[int] = ..., failed_actions: _Optional[int] = ..., results: _Optional[_Iterable[_Union[ActionReplayResult, _Mapping]]] = ..., total_duration_ms: _Optional[int] = ..., stopped_early: _Optional[bool] = ...) -> None: ...

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
