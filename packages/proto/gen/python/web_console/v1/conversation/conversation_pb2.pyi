from google.protobuf.internal import containers as _containers
from google.protobuf.internal import enum_type_wrapper as _enum_type_wrapper
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Iterable as _Iterable, Mapping as _Mapping
from typing import ClassVar as _ClassVar, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class MessageCaptureState(int, metaclass=_enum_type_wrapper.EnumTypeWrapper):
    __slots__ = ()
    MESSAGE_CAPTURE_STATE_UNSPECIFIED: _ClassVar[MessageCaptureState]
    MESSAGE_CAPTURE_STATE_CAPTURING: _ClassVar[MessageCaptureState]
    MESSAGE_CAPTURE_STATE_NOT_APPLICABLE: _ClassVar[MessageCaptureState]
    MESSAGE_CAPTURE_STATE_PENDING: _ClassVar[MessageCaptureState]
    MESSAGE_CAPTURE_STATE_UNAVAILABLE: _ClassVar[MessageCaptureState]
MESSAGE_CAPTURE_STATE_UNSPECIFIED: MessageCaptureState
MESSAGE_CAPTURE_STATE_CAPTURING: MessageCaptureState
MESSAGE_CAPTURE_STATE_NOT_APPLICABLE: MessageCaptureState
MESSAGE_CAPTURE_STATE_PENDING: MessageCaptureState
MESSAGE_CAPTURE_STATE_UNAVAILABLE: MessageCaptureState

class SearchRequest(_message.Message):
    __slots__ = ("session_id", "query", "limit", "role_filter")
    SESSION_ID_FIELD_NUMBER: _ClassVar[int]
    QUERY_FIELD_NUMBER: _ClassVar[int]
    LIMIT_FIELD_NUMBER: _ClassVar[int]
    ROLE_FILTER_FIELD_NUMBER: _ClassVar[int]
    session_id: str
    query: str
    limit: int
    role_filter: str
    def __init__(self, session_id: _Optional[str] = ..., query: _Optional[str] = ..., limit: _Optional[int] = ..., role_filter: _Optional[str] = ...) -> None: ...

class SearchMatch(_message.Message):
    __slots__ = ("event_id", "sequence", "excerpt")
    EVENT_ID_FIELD_NUMBER: _ClassVar[int]
    SEQUENCE_FIELD_NUMBER: _ClassVar[int]
    EXCERPT_FIELD_NUMBER: _ClassVar[int]
    event_id: str
    sequence: int
    excerpt: str
    def __init__(self, event_id: _Optional[str] = ..., sequence: _Optional[int] = ..., excerpt: _Optional[str] = ...) -> None: ...

class SearchResponse(_message.Message):
    __slots__ = ("matches", "truncated", "total_matches")
    MATCHES_FIELD_NUMBER: _ClassVar[int]
    TRUNCATED_FIELD_NUMBER: _ClassVar[int]
    TOTAL_MATCHES_FIELD_NUMBER: _ClassVar[int]
    matches: _containers.RepeatedCompositeFieldContainer[SearchMatch]
    truncated: bool
    total_matches: int
    def __init__(self, matches: _Optional[_Iterable[_Union[SearchMatch, _Mapping]]] = ..., truncated: _Optional[bool] = ..., total_matches: _Optional[int] = ...) -> None: ...

class SearchArchivedRequest(_message.Message):
    __slots__ = ("query", "limit", "agent_type", "role", "created_after")
    QUERY_FIELD_NUMBER: _ClassVar[int]
    LIMIT_FIELD_NUMBER: _ClassVar[int]
    AGENT_TYPE_FIELD_NUMBER: _ClassVar[int]
    ROLE_FIELD_NUMBER: _ClassVar[int]
    CREATED_AFTER_FIELD_NUMBER: _ClassVar[int]
    query: str
    limit: int
    agent_type: str
    role: str
    created_after: str
    def __init__(self, query: _Optional[str] = ..., limit: _Optional[int] = ..., agent_type: _Optional[str] = ..., role: _Optional[str] = ..., created_after: _Optional[str] = ...) -> None: ...

class ArchivedSearchMatch(_message.Message):
    __slots__ = ("event_id", "session_id", "sequence", "role", "created_at", "excerpt")
    EVENT_ID_FIELD_NUMBER: _ClassVar[int]
    SESSION_ID_FIELD_NUMBER: _ClassVar[int]
    SEQUENCE_FIELD_NUMBER: _ClassVar[int]
    ROLE_FIELD_NUMBER: _ClassVar[int]
    CREATED_AT_FIELD_NUMBER: _ClassVar[int]
    EXCERPT_FIELD_NUMBER: _ClassVar[int]
    event_id: str
    session_id: str
    sequence: int
    role: str
    created_at: str
    excerpt: str
    def __init__(self, event_id: _Optional[str] = ..., session_id: _Optional[str] = ..., sequence: _Optional[int] = ..., role: _Optional[str] = ..., created_at: _Optional[str] = ..., excerpt: _Optional[str] = ...) -> None: ...

class SearchArchivedResponse(_message.Message):
    __slots__ = ("matches", "truncated", "total_matches", "distinct_sessions")
    MATCHES_FIELD_NUMBER: _ClassVar[int]
    TRUNCATED_FIELD_NUMBER: _ClassVar[int]
    TOTAL_MATCHES_FIELD_NUMBER: _ClassVar[int]
    DISTINCT_SESSIONS_FIELD_NUMBER: _ClassVar[int]
    matches: _containers.RepeatedCompositeFieldContainer[ArchivedSearchMatch]
    truncated: bool
    total_matches: int
    distinct_sessions: int
    def __init__(self, matches: _Optional[_Iterable[_Union[ArchivedSearchMatch, _Mapping]]] = ..., truncated: _Optional[bool] = ..., total_matches: _Optional[int] = ..., distinct_sessions: _Optional[int] = ...) -> None: ...

class GetRangeRequest(_message.Message):
    __slots__ = ("session_id", "from_sequence", "to_sequence")
    SESSION_ID_FIELD_NUMBER: _ClassVar[int]
    FROM_SEQUENCE_FIELD_NUMBER: _ClassVar[int]
    TO_SEQUENCE_FIELD_NUMBER: _ClassVar[int]
    session_id: str
    from_sequence: int
    to_sequence: int
    def __init__(self, session_id: _Optional[str] = ..., from_sequence: _Optional[int] = ..., to_sequence: _Optional[int] = ...) -> None: ...

class ConversationEvent(_message.Message):
    __slots__ = ("id", "session_id", "source", "role", "text", "speech_paragraphs", "original_speech_paragraphs", "summarized", "created_at", "sequence", "delivery_state", "tts_state", "consumption_state")
    ID_FIELD_NUMBER: _ClassVar[int]
    SESSION_ID_FIELD_NUMBER: _ClassVar[int]
    SOURCE_FIELD_NUMBER: _ClassVar[int]
    ROLE_FIELD_NUMBER: _ClassVar[int]
    TEXT_FIELD_NUMBER: _ClassVar[int]
    SPEECH_PARAGRAPHS_FIELD_NUMBER: _ClassVar[int]
    ORIGINAL_SPEECH_PARAGRAPHS_FIELD_NUMBER: _ClassVar[int]
    SUMMARIZED_FIELD_NUMBER: _ClassVar[int]
    CREATED_AT_FIELD_NUMBER: _ClassVar[int]
    SEQUENCE_FIELD_NUMBER: _ClassVar[int]
    DELIVERY_STATE_FIELD_NUMBER: _ClassVar[int]
    TTS_STATE_FIELD_NUMBER: _ClassVar[int]
    CONSUMPTION_STATE_FIELD_NUMBER: _ClassVar[int]
    id: str
    session_id: str
    source: str
    role: str
    text: str
    speech_paragraphs: _containers.RepeatedScalarFieldContainer[str]
    original_speech_paragraphs: _containers.RepeatedScalarFieldContainer[str]
    summarized: bool
    created_at: str
    sequence: int
    delivery_state: str
    tts_state: str
    consumption_state: str
    def __init__(self, id: _Optional[str] = ..., session_id: _Optional[str] = ..., source: _Optional[str] = ..., role: _Optional[str] = ..., text: _Optional[str] = ..., speech_paragraphs: _Optional[_Iterable[str]] = ..., original_speech_paragraphs: _Optional[_Iterable[str]] = ..., summarized: _Optional[bool] = ..., created_at: _Optional[str] = ..., sequence: _Optional[int] = ..., delivery_state: _Optional[str] = ..., tts_state: _Optional[str] = ..., consumption_state: _Optional[str] = ...) -> None: ...

class ConversationCursor(_message.Message):
    __slots__ = ("last_seen_sequence", "last_listened_sequence")
    LAST_SEEN_SEQUENCE_FIELD_NUMBER: _ClassVar[int]
    LAST_LISTENED_SEQUENCE_FIELD_NUMBER: _ClassVar[int]
    last_seen_sequence: int
    last_listened_sequence: int
    def __init__(self, last_seen_sequence: _Optional[int] = ..., last_listened_sequence: _Optional[int] = ...) -> None: ...

class GetRequest(_message.Message):
    __slots__ = ("session_id", "since_sequence", "limit", "before_sequence")
    SESSION_ID_FIELD_NUMBER: _ClassVar[int]
    SINCE_SEQUENCE_FIELD_NUMBER: _ClassVar[int]
    LIMIT_FIELD_NUMBER: _ClassVar[int]
    BEFORE_SEQUENCE_FIELD_NUMBER: _ClassVar[int]
    session_id: str
    since_sequence: int
    limit: int
    before_sequence: int
    def __init__(self, session_id: _Optional[str] = ..., since_sequence: _Optional[int] = ..., limit: _Optional[int] = ..., before_sequence: _Optional[int] = ...) -> None: ...

class GetResponse(_message.Message):
    __slots__ = ("session_id", "events", "cursor", "has_more", "oldest_sequence", "newest_sequence", "total_count", "capture")
    SESSION_ID_FIELD_NUMBER: _ClassVar[int]
    EVENTS_FIELD_NUMBER: _ClassVar[int]
    CURSOR_FIELD_NUMBER: _ClassVar[int]
    HAS_MORE_FIELD_NUMBER: _ClassVar[int]
    OLDEST_SEQUENCE_FIELD_NUMBER: _ClassVar[int]
    NEWEST_SEQUENCE_FIELD_NUMBER: _ClassVar[int]
    TOTAL_COUNT_FIELD_NUMBER: _ClassVar[int]
    CAPTURE_FIELD_NUMBER: _ClassVar[int]
    session_id: str
    events: _containers.RepeatedCompositeFieldContainer[ConversationEvent]
    cursor: ConversationCursor
    has_more: bool
    oldest_sequence: int
    newest_sequence: int
    total_count: int
    capture: MessageCaptureStatus
    def __init__(self, session_id: _Optional[str] = ..., events: _Optional[_Iterable[_Union[ConversationEvent, _Mapping]]] = ..., cursor: _Optional[_Union[ConversationCursor, _Mapping]] = ..., has_more: _Optional[bool] = ..., oldest_sequence: _Optional[int] = ..., newest_sequence: _Optional[int] = ..., total_count: _Optional[int] = ..., capture: _Optional[_Union[MessageCaptureStatus, _Mapping]] = ...) -> None: ...

class MessageCaptureStatus(_message.Message):
    __slots__ = ("state", "reason_code", "summary", "detail", "remediation", "transcript_path", "last_captured_at")
    STATE_FIELD_NUMBER: _ClassVar[int]
    REASON_CODE_FIELD_NUMBER: _ClassVar[int]
    SUMMARY_FIELD_NUMBER: _ClassVar[int]
    DETAIL_FIELD_NUMBER: _ClassVar[int]
    REMEDIATION_FIELD_NUMBER: _ClassVar[int]
    TRANSCRIPT_PATH_FIELD_NUMBER: _ClassVar[int]
    LAST_CAPTURED_AT_FIELD_NUMBER: _ClassVar[int]
    state: MessageCaptureState
    reason_code: str
    summary: str
    detail: str
    remediation: str
    transcript_path: str
    last_captured_at: str
    def __init__(self, state: _Optional[_Union[MessageCaptureState, str]] = ..., reason_code: _Optional[str] = ..., summary: _Optional[str] = ..., detail: _Optional[str] = ..., remediation: _Optional[str] = ..., transcript_path: _Optional[str] = ..., last_captured_at: _Optional[str] = ...) -> None: ...

class UpdateCursorRequest(_message.Message):
    __slots__ = ("session_id", "last_seen_sequence", "has_last_seen_sequence", "last_listened_sequence", "has_last_listened_sequence")
    SESSION_ID_FIELD_NUMBER: _ClassVar[int]
    LAST_SEEN_SEQUENCE_FIELD_NUMBER: _ClassVar[int]
    HAS_LAST_SEEN_SEQUENCE_FIELD_NUMBER: _ClassVar[int]
    LAST_LISTENED_SEQUENCE_FIELD_NUMBER: _ClassVar[int]
    HAS_LAST_LISTENED_SEQUENCE_FIELD_NUMBER: _ClassVar[int]
    session_id: str
    last_seen_sequence: int
    has_last_seen_sequence: bool
    last_listened_sequence: int
    has_last_listened_sequence: bool
    def __init__(self, session_id: _Optional[str] = ..., last_seen_sequence: _Optional[int] = ..., has_last_seen_sequence: _Optional[bool] = ..., last_listened_sequence: _Optional[int] = ..., has_last_listened_sequence: _Optional[bool] = ...) -> None: ...

class UpdateCursorResponse(_message.Message):
    __slots__ = ("cursor",)
    CURSOR_FIELD_NUMBER: _ClassVar[int]
    cursor: ConversationCursor
    def __init__(self, cursor: _Optional[_Union[ConversationCursor, _Mapping]] = ...) -> None: ...

class SummarizeEventRequest(_message.Message):
    __slots__ = ("session_id", "event_id")
    SESSION_ID_FIELD_NUMBER: _ClassVar[int]
    EVENT_ID_FIELD_NUMBER: _ClassVar[int]
    session_id: str
    event_id: str
    def __init__(self, session_id: _Optional[str] = ..., event_id: _Optional[str] = ...) -> None: ...

class SummarizeEventResponse(_message.Message):
    __slots__ = ("summarized", "speech_paragraphs", "error")
    SUMMARIZED_FIELD_NUMBER: _ClassVar[int]
    SPEECH_PARAGRAPHS_FIELD_NUMBER: _ClassVar[int]
    ERROR_FIELD_NUMBER: _ClassVar[int]
    summarized: bool
    speech_paragraphs: _containers.RepeatedScalarFieldContainer[str]
    error: str
    def __init__(self, summarized: _Optional[bool] = ..., speech_paragraphs: _Optional[_Iterable[str]] = ..., error: _Optional[str] = ...) -> None: ...
