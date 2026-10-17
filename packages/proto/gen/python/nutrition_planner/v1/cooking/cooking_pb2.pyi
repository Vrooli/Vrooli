from google.protobuf.internal import containers as _containers
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Iterable as _Iterable, Mapping as _Mapping
from typing import ClassVar as _ClassVar, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class Timer(_message.Message):
    __slots__ = ("id", "step_id", "duration_seconds", "started_at", "paused_at", "elapsed_seconds")
    ID_FIELD_NUMBER: _ClassVar[int]
    STEP_ID_FIELD_NUMBER: _ClassVar[int]
    DURATION_SECONDS_FIELD_NUMBER: _ClassVar[int]
    STARTED_AT_FIELD_NUMBER: _ClassVar[int]
    PAUSED_AT_FIELD_NUMBER: _ClassVar[int]
    ELAPSED_SECONDS_FIELD_NUMBER: _ClassVar[int]
    id: str
    step_id: str
    duration_seconds: int
    started_at: str
    paused_at: str
    elapsed_seconds: int
    def __init__(self, id: _Optional[str] = ..., step_id: _Optional[str] = ..., duration_seconds: _Optional[int] = ..., started_at: _Optional[str] = ..., paused_at: _Optional[str] = ..., elapsed_seconds: _Optional[int] = ...) -> None: ...

class CookingSession(_message.Message):
    __slots__ = ("id", "workspace_id", "recipe_id", "recipe_revision", "method_id", "scale", "current_step_index", "completed_steps", "timers", "status", "actual_yield", "yield_unit", "version", "started_at", "updated_at", "finished_at")
    ID_FIELD_NUMBER: _ClassVar[int]
    WORKSPACE_ID_FIELD_NUMBER: _ClassVar[int]
    RECIPE_ID_FIELD_NUMBER: _ClassVar[int]
    RECIPE_REVISION_FIELD_NUMBER: _ClassVar[int]
    METHOD_ID_FIELD_NUMBER: _ClassVar[int]
    SCALE_FIELD_NUMBER: _ClassVar[int]
    CURRENT_STEP_INDEX_FIELD_NUMBER: _ClassVar[int]
    COMPLETED_STEPS_FIELD_NUMBER: _ClassVar[int]
    TIMERS_FIELD_NUMBER: _ClassVar[int]
    STATUS_FIELD_NUMBER: _ClassVar[int]
    ACTUAL_YIELD_FIELD_NUMBER: _ClassVar[int]
    YIELD_UNIT_FIELD_NUMBER: _ClassVar[int]
    VERSION_FIELD_NUMBER: _ClassVar[int]
    STARTED_AT_FIELD_NUMBER: _ClassVar[int]
    UPDATED_AT_FIELD_NUMBER: _ClassVar[int]
    FINISHED_AT_FIELD_NUMBER: _ClassVar[int]
    id: str
    workspace_id: str
    recipe_id: str
    recipe_revision: int
    method_id: str
    scale: str
    current_step_index: int
    completed_steps: _containers.RepeatedScalarFieldContainer[str]
    timers: _containers.RepeatedCompositeFieldContainer[Timer]
    status: str
    actual_yield: str
    yield_unit: str
    version: int
    started_at: str
    updated_at: str
    finished_at: str
    def __init__(self, id: _Optional[str] = ..., workspace_id: _Optional[str] = ..., recipe_id: _Optional[str] = ..., recipe_revision: _Optional[int] = ..., method_id: _Optional[str] = ..., scale: _Optional[str] = ..., current_step_index: _Optional[int] = ..., completed_steps: _Optional[_Iterable[str]] = ..., timers: _Optional[_Iterable[_Union[Timer, _Mapping]]] = ..., status: _Optional[str] = ..., actual_yield: _Optional[str] = ..., yield_unit: _Optional[str] = ..., version: _Optional[int] = ..., started_at: _Optional[str] = ..., updated_at: _Optional[str] = ..., finished_at: _Optional[str] = ...) -> None: ...

class StartSessionRequest(_message.Message):
    __slots__ = ("workspace_id", "session_id", "recipe_id", "recipe_revision", "method_id", "scale")
    WORKSPACE_ID_FIELD_NUMBER: _ClassVar[int]
    SESSION_ID_FIELD_NUMBER: _ClassVar[int]
    RECIPE_ID_FIELD_NUMBER: _ClassVar[int]
    RECIPE_REVISION_FIELD_NUMBER: _ClassVar[int]
    METHOD_ID_FIELD_NUMBER: _ClassVar[int]
    SCALE_FIELD_NUMBER: _ClassVar[int]
    workspace_id: str
    session_id: str
    recipe_id: str
    recipe_revision: int
    method_id: str
    scale: str
    def __init__(self, workspace_id: _Optional[str] = ..., session_id: _Optional[str] = ..., recipe_id: _Optional[str] = ..., recipe_revision: _Optional[int] = ..., method_id: _Optional[str] = ..., scale: _Optional[str] = ...) -> None: ...

class GetSessionRequest(_message.Message):
    __slots__ = ("workspace_id", "session_id")
    WORKSPACE_ID_FIELD_NUMBER: _ClassVar[int]
    SESSION_ID_FIELD_NUMBER: _ClassVar[int]
    workspace_id: str
    session_id: str
    def __init__(self, workspace_id: _Optional[str] = ..., session_id: _Optional[str] = ...) -> None: ...

class ListSessionsRequest(_message.Message):
    __slots__ = ("workspace_id",)
    WORKSPACE_ID_FIELD_NUMBER: _ClassVar[int]
    workspace_id: str
    def __init__(self, workspace_id: _Optional[str] = ...) -> None: ...

class ListSessionsResponse(_message.Message):
    __slots__ = ("sessions",)
    SESSIONS_FIELD_NUMBER: _ClassVar[int]
    sessions: _containers.RepeatedCompositeFieldContainer[CookingSession]
    def __init__(self, sessions: _Optional[_Iterable[_Union[CookingSession, _Mapping]]] = ...) -> None: ...

class SaveSessionRequest(_message.Message):
    __slots__ = ("workspace_id", "session_id", "event_id", "expected_version", "current_step_index", "completed_steps", "timers", "finish", "actual_yield", "yield_unit")
    WORKSPACE_ID_FIELD_NUMBER: _ClassVar[int]
    SESSION_ID_FIELD_NUMBER: _ClassVar[int]
    EVENT_ID_FIELD_NUMBER: _ClassVar[int]
    EXPECTED_VERSION_FIELD_NUMBER: _ClassVar[int]
    CURRENT_STEP_INDEX_FIELD_NUMBER: _ClassVar[int]
    COMPLETED_STEPS_FIELD_NUMBER: _ClassVar[int]
    TIMERS_FIELD_NUMBER: _ClassVar[int]
    FINISH_FIELD_NUMBER: _ClassVar[int]
    ACTUAL_YIELD_FIELD_NUMBER: _ClassVar[int]
    YIELD_UNIT_FIELD_NUMBER: _ClassVar[int]
    workspace_id: str
    session_id: str
    event_id: str
    expected_version: int
    current_step_index: int
    completed_steps: _containers.RepeatedScalarFieldContainer[str]
    timers: _containers.RepeatedCompositeFieldContainer[Timer]
    finish: bool
    actual_yield: str
    yield_unit: str
    def __init__(self, workspace_id: _Optional[str] = ..., session_id: _Optional[str] = ..., event_id: _Optional[str] = ..., expected_version: _Optional[int] = ..., current_step_index: _Optional[int] = ..., completed_steps: _Optional[_Iterable[str]] = ..., timers: _Optional[_Iterable[_Union[Timer, _Mapping]]] = ..., finish: _Optional[bool] = ..., actual_yield: _Optional[str] = ..., yield_unit: _Optional[str] = ...) -> None: ...

class SessionResponse(_message.Message):
    __slots__ = ("session",)
    SESSION_FIELD_NUMBER: _ClassVar[int]
    session: CookingSession
    def __init__(self, session: _Optional[_Union[CookingSession, _Mapping]] = ...) -> None: ...
