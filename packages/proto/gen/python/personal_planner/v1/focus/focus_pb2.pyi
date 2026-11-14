from google.protobuf.internal import containers as _containers
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Iterable as _Iterable, Mapping as _Mapping
from typing import ClassVar as _ClassVar, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class Actual(_message.Message):
    __slots__ = ("id", "work_item_id", "title", "local_date", "reported_minutes", "certainty", "note", "created_at_unix_seconds", "revision")
    ID_FIELD_NUMBER: _ClassVar[int]
    WORK_ITEM_ID_FIELD_NUMBER: _ClassVar[int]
    TITLE_FIELD_NUMBER: _ClassVar[int]
    LOCAL_DATE_FIELD_NUMBER: _ClassVar[int]
    REPORTED_MINUTES_FIELD_NUMBER: _ClassVar[int]
    CERTAINTY_FIELD_NUMBER: _ClassVar[int]
    NOTE_FIELD_NUMBER: _ClassVar[int]
    CREATED_AT_UNIX_SECONDS_FIELD_NUMBER: _ClassVar[int]
    REVISION_FIELD_NUMBER: _ClassVar[int]
    id: str
    work_item_id: str
    title: str
    local_date: str
    reported_minutes: int
    certainty: str
    note: str
    created_at_unix_seconds: int
    revision: int
    def __init__(self, id: _Optional[str] = ..., work_item_id: _Optional[str] = ..., title: _Optional[str] = ..., local_date: _Optional[str] = ..., reported_minutes: _Optional[int] = ..., certainty: _Optional[str] = ..., note: _Optional[str] = ..., created_at_unix_seconds: _Optional[int] = ..., revision: _Optional[int] = ...) -> None: ...

class RecordManualActualRequest(_message.Message):
    __slots__ = ("work_item_id", "title", "local_date", "reported_minutes", "certainty", "note")
    WORK_ITEM_ID_FIELD_NUMBER: _ClassVar[int]
    TITLE_FIELD_NUMBER: _ClassVar[int]
    LOCAL_DATE_FIELD_NUMBER: _ClassVar[int]
    REPORTED_MINUTES_FIELD_NUMBER: _ClassVar[int]
    CERTAINTY_FIELD_NUMBER: _ClassVar[int]
    NOTE_FIELD_NUMBER: _ClassVar[int]
    work_item_id: str
    title: str
    local_date: str
    reported_minutes: int
    certainty: str
    note: str
    def __init__(self, work_item_id: _Optional[str] = ..., title: _Optional[str] = ..., local_date: _Optional[str] = ..., reported_minutes: _Optional[int] = ..., certainty: _Optional[str] = ..., note: _Optional[str] = ...) -> None: ...

class ListActualsRequest(_message.Message):
    __slots__ = ("local_date",)
    LOCAL_DATE_FIELD_NUMBER: _ClassVar[int]
    local_date: str
    def __init__(self, local_date: _Optional[str] = ...) -> None: ...

class ListActualsResponse(_message.Message):
    __slots__ = ("actuals",)
    ACTUALS_FIELD_NUMBER: _ClassVar[int]
    actuals: _containers.RepeatedCompositeFieldContainer[Actual]
    def __init__(self, actuals: _Optional[_Iterable[_Union[Actual, _Mapping]]] = ...) -> None: ...

class ActualCorrection(_message.Message):
    __slots__ = ("id", "actual_id", "previous_minutes", "new_minutes", "previous_certainty", "new_certainty", "reason", "created_at_unix_seconds")
    ID_FIELD_NUMBER: _ClassVar[int]
    ACTUAL_ID_FIELD_NUMBER: _ClassVar[int]
    PREVIOUS_MINUTES_FIELD_NUMBER: _ClassVar[int]
    NEW_MINUTES_FIELD_NUMBER: _ClassVar[int]
    PREVIOUS_CERTAINTY_FIELD_NUMBER: _ClassVar[int]
    NEW_CERTAINTY_FIELD_NUMBER: _ClassVar[int]
    REASON_FIELD_NUMBER: _ClassVar[int]
    CREATED_AT_UNIX_SECONDS_FIELD_NUMBER: _ClassVar[int]
    id: str
    actual_id: str
    previous_minutes: int
    new_minutes: int
    previous_certainty: str
    new_certainty: str
    reason: str
    created_at_unix_seconds: int
    def __init__(self, id: _Optional[str] = ..., actual_id: _Optional[str] = ..., previous_minutes: _Optional[int] = ..., new_minutes: _Optional[int] = ..., previous_certainty: _Optional[str] = ..., new_certainty: _Optional[str] = ..., reason: _Optional[str] = ..., created_at_unix_seconds: _Optional[int] = ...) -> None: ...

class ListActualCorrectionsRequest(_message.Message):
    __slots__ = ("actual_id", "local_date", "limit")
    ACTUAL_ID_FIELD_NUMBER: _ClassVar[int]
    LOCAL_DATE_FIELD_NUMBER: _ClassVar[int]
    LIMIT_FIELD_NUMBER: _ClassVar[int]
    actual_id: str
    local_date: str
    limit: int
    def __init__(self, actual_id: _Optional[str] = ..., local_date: _Optional[str] = ..., limit: _Optional[int] = ...) -> None: ...

class ListActualCorrectionsResponse(_message.Message):
    __slots__ = ("corrections",)
    CORRECTIONS_FIELD_NUMBER: _ClassVar[int]
    corrections: _containers.RepeatedCompositeFieldContainer[ActualCorrection]
    def __init__(self, corrections: _Optional[_Iterable[_Union[ActualCorrection, _Mapping]]] = ...) -> None: ...

class CorrectActualRequest(_message.Message):
    __slots__ = ("id", "expected_revision", "reported_minutes", "certainty", "note")
    ID_FIELD_NUMBER: _ClassVar[int]
    EXPECTED_REVISION_FIELD_NUMBER: _ClassVar[int]
    REPORTED_MINUTES_FIELD_NUMBER: _ClassVar[int]
    CERTAINTY_FIELD_NUMBER: _ClassVar[int]
    NOTE_FIELD_NUMBER: _ClassVar[int]
    id: str
    expected_revision: int
    reported_minutes: int
    certainty: str
    note: str
    def __init__(self, id: _Optional[str] = ..., expected_revision: _Optional[int] = ..., reported_minutes: _Optional[int] = ..., certainty: _Optional[str] = ..., note: _Optional[str] = ...) -> None: ...

class ActualResponse(_message.Message):
    __slots__ = ("actual",)
    ACTUAL_FIELD_NUMBER: _ClassVar[int]
    actual: Actual
    def __init__(self, actual: _Optional[_Union[Actual, _Mapping]] = ...) -> None: ...

class FocusSession(_message.Message):
    __slots__ = ("id", "work_item_id", "title", "mode", "state", "started_at_unix_seconds", "ended_at_unix_seconds", "active_seconds", "wall_seconds", "revision", "active_started_at_unix_seconds")
    ID_FIELD_NUMBER: _ClassVar[int]
    WORK_ITEM_ID_FIELD_NUMBER: _ClassVar[int]
    TITLE_FIELD_NUMBER: _ClassVar[int]
    MODE_FIELD_NUMBER: _ClassVar[int]
    STATE_FIELD_NUMBER: _ClassVar[int]
    STARTED_AT_UNIX_SECONDS_FIELD_NUMBER: _ClassVar[int]
    ENDED_AT_UNIX_SECONDS_FIELD_NUMBER: _ClassVar[int]
    ACTIVE_SECONDS_FIELD_NUMBER: _ClassVar[int]
    WALL_SECONDS_FIELD_NUMBER: _ClassVar[int]
    REVISION_FIELD_NUMBER: _ClassVar[int]
    ACTIVE_STARTED_AT_UNIX_SECONDS_FIELD_NUMBER: _ClassVar[int]
    id: str
    work_item_id: str
    title: str
    mode: str
    state: str
    started_at_unix_seconds: int
    ended_at_unix_seconds: int
    active_seconds: int
    wall_seconds: int
    revision: int
    active_started_at_unix_seconds: int
    def __init__(self, id: _Optional[str] = ..., work_item_id: _Optional[str] = ..., title: _Optional[str] = ..., mode: _Optional[str] = ..., state: _Optional[str] = ..., started_at_unix_seconds: _Optional[int] = ..., ended_at_unix_seconds: _Optional[int] = ..., active_seconds: _Optional[int] = ..., wall_seconds: _Optional[int] = ..., revision: _Optional[int] = ..., active_started_at_unix_seconds: _Optional[int] = ...) -> None: ...

class GetCurrentSessionRequest(_message.Message):
    __slots__ = ()
    def __init__(self) -> None: ...

class GetCurrentSessionResponse(_message.Message):
    __slots__ = ("has_session", "session")
    HAS_SESSION_FIELD_NUMBER: _ClassVar[int]
    SESSION_FIELD_NUMBER: _ClassVar[int]
    has_session: bool
    session: FocusSession
    def __init__(self, has_session: _Optional[bool] = ..., session: _Optional[_Union[FocusSession, _Mapping]] = ...) -> None: ...

class StartFocusRequest(_message.Message):
    __slots__ = ("work_item_id", "title", "mode")
    WORK_ITEM_ID_FIELD_NUMBER: _ClassVar[int]
    TITLE_FIELD_NUMBER: _ClassVar[int]
    MODE_FIELD_NUMBER: _ClassVar[int]
    work_item_id: str
    title: str
    mode: str
    def __init__(self, work_item_id: _Optional[str] = ..., title: _Optional[str] = ..., mode: _Optional[str] = ...) -> None: ...

class TransitionFocusRequest(_message.Message):
    __slots__ = ("session_id", "expected_revision")
    SESSION_ID_FIELD_NUMBER: _ClassVar[int]
    EXPECTED_REVISION_FIELD_NUMBER: _ClassVar[int]
    session_id: str
    expected_revision: int
    def __init__(self, session_id: _Optional[str] = ..., expected_revision: _Optional[int] = ...) -> None: ...

class EndFocusRequest(_message.Message):
    __slots__ = ("session_id", "expected_revision", "reported_remaining_minutes")
    SESSION_ID_FIELD_NUMBER: _ClassVar[int]
    EXPECTED_REVISION_FIELD_NUMBER: _ClassVar[int]
    REPORTED_REMAINING_MINUTES_FIELD_NUMBER: _ClassVar[int]
    session_id: str
    expected_revision: int
    reported_remaining_minutes: int
    def __init__(self, session_id: _Optional[str] = ..., expected_revision: _Optional[int] = ..., reported_remaining_minutes: _Optional[int] = ...) -> None: ...

class FocusSessionResponse(_message.Message):
    __slots__ = ("session",)
    SESSION_FIELD_NUMBER: _ClassVar[int]
    session: FocusSession
    def __init__(self, session: _Optional[_Union[FocusSession, _Mapping]] = ...) -> None: ...
