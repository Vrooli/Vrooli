from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Mapping as _Mapping
from typing import ClassVar as _ClassVar, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class OpenSessionRequest(_message.Message):
    __slots__ = ("transport", "voice", "language")
    TRANSPORT_FIELD_NUMBER: _ClassVar[int]
    VOICE_FIELD_NUMBER: _ClassVar[int]
    LANGUAGE_FIELD_NUMBER: _ClassVar[int]
    transport: str
    voice: str
    language: str
    def __init__(self, transport: _Optional[str] = ..., voice: _Optional[str] = ..., language: _Optional[str] = ...) -> None: ...

class OpenSessionResponse(_message.Message):
    __slots__ = ("session_id", "transport")
    SESSION_ID_FIELD_NUMBER: _ClassVar[int]
    TRANSPORT_FIELD_NUMBER: _ClassVar[int]
    session_id: str
    transport: str
    def __init__(self, session_id: _Optional[str] = ..., transport: _Optional[str] = ...) -> None: ...

class CloseSessionRequest(_message.Message):
    __slots__ = ("session_id", "reason")
    SESSION_ID_FIELD_NUMBER: _ClassVar[int]
    REASON_FIELD_NUMBER: _ClassVar[int]
    session_id: str
    reason: str
    def __init__(self, session_id: _Optional[str] = ..., reason: _Optional[str] = ...) -> None: ...

class CloseSessionResponse(_message.Message):
    __slots__ = ()
    def __init__(self) -> None: ...

class SendTextRequest(_message.Message):
    __slots__ = ("session_id", "text", "cancel_inflight")
    SESSION_ID_FIELD_NUMBER: _ClassVar[int]
    TEXT_FIELD_NUMBER: _ClassVar[int]
    CANCEL_INFLIGHT_FIELD_NUMBER: _ClassVar[int]
    session_id: str
    text: str
    cancel_inflight: bool
    def __init__(self, session_id: _Optional[str] = ..., text: _Optional[str] = ..., cancel_inflight: _Optional[bool] = ...) -> None: ...

class SendTextResponse(_message.Message):
    __slots__ = ("event_id",)
    EVENT_ID_FIELD_NUMBER: _ClassVar[int]
    event_id: str
    def __init__(self, event_id: _Optional[str] = ...) -> None: ...

class SendCancelRequest(_message.Message):
    __slots__ = ("session_id", "reason")
    SESSION_ID_FIELD_NUMBER: _ClassVar[int]
    REASON_FIELD_NUMBER: _ClassVar[int]
    session_id: str
    reason: str
    def __init__(self, session_id: _Optional[str] = ..., reason: _Optional[str] = ...) -> None: ...

class SendCancelResponse(_message.Message):
    __slots__ = ()
    def __init__(self) -> None: ...

class SubscribeRequest(_message.Message):
    __slots__ = ("session_id", "from_event_id")
    SESSION_ID_FIELD_NUMBER: _ClassVar[int]
    FROM_EVENT_ID_FIELD_NUMBER: _ClassVar[int]
    session_id: str
    from_event_id: str
    def __init__(self, session_id: _Optional[str] = ..., from_event_id: _Optional[str] = ...) -> None: ...

class SubscribeResponse(_message.Message):
    __slots__ = ("event",)
    EVENT_FIELD_NUMBER: _ClassVar[int]
    event: SessionEvent
    def __init__(self, event: _Optional[_Union[SessionEvent, _Mapping]] = ...) -> None: ...

class SessionEvent(_message.Message):
    __slots__ = ("event_id", "session_id", "emitted_at", "transcript_delta", "transcript_final", "assistant_delta", "assistant_final", "vad", "tool", "barge_in_cancel", "closed")
    EVENT_ID_FIELD_NUMBER: _ClassVar[int]
    SESSION_ID_FIELD_NUMBER: _ClassVar[int]
    EMITTED_AT_FIELD_NUMBER: _ClassVar[int]
    TRANSCRIPT_DELTA_FIELD_NUMBER: _ClassVar[int]
    TRANSCRIPT_FINAL_FIELD_NUMBER: _ClassVar[int]
    ASSISTANT_DELTA_FIELD_NUMBER: _ClassVar[int]
    ASSISTANT_FINAL_FIELD_NUMBER: _ClassVar[int]
    VAD_FIELD_NUMBER: _ClassVar[int]
    TOOL_FIELD_NUMBER: _ClassVar[int]
    BARGE_IN_CANCEL_FIELD_NUMBER: _ClassVar[int]
    CLOSED_FIELD_NUMBER: _ClassVar[int]
    event_id: str
    session_id: str
    emitted_at: str
    transcript_delta: TranscriptDelta
    transcript_final: TranscriptFinal
    assistant_delta: AssistantDelta
    assistant_final: AssistantFinal
    vad: VadEvent
    tool: ToolEvent
    barge_in_cancel: BargeInCancel
    closed: SessionClosed
    def __init__(self, event_id: _Optional[str] = ..., session_id: _Optional[str] = ..., emitted_at: _Optional[str] = ..., transcript_delta: _Optional[_Union[TranscriptDelta, _Mapping]] = ..., transcript_final: _Optional[_Union[TranscriptFinal, _Mapping]] = ..., assistant_delta: _Optional[_Union[AssistantDelta, _Mapping]] = ..., assistant_final: _Optional[_Union[AssistantFinal, _Mapping]] = ..., vad: _Optional[_Union[VadEvent, _Mapping]] = ..., tool: _Optional[_Union[ToolEvent, _Mapping]] = ..., barge_in_cancel: _Optional[_Union[BargeInCancel, _Mapping]] = ..., closed: _Optional[_Union[SessionClosed, _Mapping]] = ...) -> None: ...

class TranscriptDelta(_message.Message):
    __slots__ = ("text", "from_seconds", "to_seconds")
    TEXT_FIELD_NUMBER: _ClassVar[int]
    FROM_SECONDS_FIELD_NUMBER: _ClassVar[int]
    TO_SECONDS_FIELD_NUMBER: _ClassVar[int]
    text: str
    from_seconds: float
    to_seconds: float
    def __init__(self, text: _Optional[str] = ..., from_seconds: _Optional[float] = ..., to_seconds: _Optional[float] = ...) -> None: ...

class TranscriptFinal(_message.Message):
    __slots__ = ("text", "duration_seconds", "speaker_verified")
    TEXT_FIELD_NUMBER: _ClassVar[int]
    DURATION_SECONDS_FIELD_NUMBER: _ClassVar[int]
    SPEAKER_VERIFIED_FIELD_NUMBER: _ClassVar[int]
    text: str
    duration_seconds: float
    speaker_verified: bool
    def __init__(self, text: _Optional[str] = ..., duration_seconds: _Optional[float] = ..., speaker_verified: _Optional[bool] = ...) -> None: ...

class AssistantDelta(_message.Message):
    __slots__ = ("text",)
    TEXT_FIELD_NUMBER: _ClassVar[int]
    text: str
    def __init__(self, text: _Optional[str] = ...) -> None: ...

class AssistantFinal(_message.Message):
    __slots__ = ("text", "had_audio")
    TEXT_FIELD_NUMBER: _ClassVar[int]
    HAD_AUDIO_FIELD_NUMBER: _ClassVar[int]
    text: str
    had_audio: bool
    def __init__(self, text: _Optional[str] = ..., had_audio: _Optional[bool] = ...) -> None: ...

class VadEvent(_message.Message):
    __slots__ = ("state",)
    STATE_FIELD_NUMBER: _ClassVar[int]
    state: str
    def __init__(self, state: _Optional[str] = ...) -> None: ...

class ToolEvent(_message.Message):
    __slots__ = ("name", "payload_json")
    NAME_FIELD_NUMBER: _ClassVar[int]
    PAYLOAD_JSON_FIELD_NUMBER: _ClassVar[int]
    name: str
    payload_json: str
    def __init__(self, name: _Optional[str] = ..., payload_json: _Optional[str] = ...) -> None: ...

class BargeInCancel(_message.Message):
    __slots__ = ("reason", "canceled_event_id")
    REASON_FIELD_NUMBER: _ClassVar[int]
    CANCELED_EVENT_ID_FIELD_NUMBER: _ClassVar[int]
    reason: str
    canceled_event_id: str
    def __init__(self, reason: _Optional[str] = ..., canceled_event_id: _Optional[str] = ...) -> None: ...

class SessionClosed(_message.Message):
    __slots__ = ("reason",)
    REASON_FIELD_NUMBER: _ClassVar[int]
    reason: str
    def __init__(self, reason: _Optional[str] = ...) -> None: ...
