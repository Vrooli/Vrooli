from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Mapping as _Mapping
from typing import ClassVar as _ClassVar, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class GetRequest(_message.Message):
    __slots__ = ()
    def __init__(self) -> None: ...

class GetResponse(_message.Message):
    __slots__ = ("sessions", "connections", "messages", "reattach", "recovery", "ai_generations", "ai_suggestions", "stdin_before_ready_total", "voice_skip_verification_total", "uptime")
    SESSIONS_FIELD_NUMBER: _ClassVar[int]
    CONNECTIONS_FIELD_NUMBER: _ClassVar[int]
    MESSAGES_FIELD_NUMBER: _ClassVar[int]
    REATTACH_FIELD_NUMBER: _ClassVar[int]
    RECOVERY_FIELD_NUMBER: _ClassVar[int]
    AI_GENERATIONS_FIELD_NUMBER: _ClassVar[int]
    AI_SUGGESTIONS_FIELD_NUMBER: _ClassVar[int]
    STDIN_BEFORE_READY_TOTAL_FIELD_NUMBER: _ClassVar[int]
    VOICE_SKIP_VERIFICATION_TOTAL_FIELD_NUMBER: _ClassVar[int]
    UPTIME_FIELD_NUMBER: _ClassVar[int]
    sessions: SessionMetrics
    connections: ConnectionMetrics
    messages: MessageMetrics
    reattach: ReattachMetrics
    recovery: RecoveryMetrics
    ai_generations: int
    ai_suggestions: int
    stdin_before_ready_total: int
    voice_skip_verification_total: int
    uptime: str
    def __init__(self, sessions: _Optional[_Union[SessionMetrics, _Mapping]] = ..., connections: _Optional[_Union[ConnectionMetrics, _Mapping]] = ..., messages: _Optional[_Union[MessageMetrics, _Mapping]] = ..., reattach: _Optional[_Union[ReattachMetrics, _Mapping]] = ..., recovery: _Optional[_Union[RecoveryMetrics, _Mapping]] = ..., ai_generations: _Optional[int] = ..., ai_suggestions: _Optional[int] = ..., stdin_before_ready_total: _Optional[int] = ..., voice_skip_verification_total: _Optional[int] = ..., uptime: _Optional[str] = ...) -> None: ...

class SessionMetrics(_message.Message):
    __slots__ = ("created", "deleted", "active", "resizes")
    CREATED_FIELD_NUMBER: _ClassVar[int]
    DELETED_FIELD_NUMBER: _ClassVar[int]
    ACTIVE_FIELD_NUMBER: _ClassVar[int]
    RESIZES_FIELD_NUMBER: _ClassVar[int]
    created: int
    deleted: int
    active: int
    resizes: int
    def __init__(self, created: _Optional[int] = ..., deleted: _Optional[int] = ..., active: _Optional[int] = ..., resizes: _Optional[int] = ...) -> None: ...

class ConnectionMetrics(_message.Message):
    __slots__ = ("total", "active")
    TOTAL_FIELD_NUMBER: _ClassVar[int]
    ACTIVE_FIELD_NUMBER: _ClassVar[int]
    total: int
    active: int
    def __init__(self, total: _Optional[int] = ..., active: _Optional[int] = ...) -> None: ...

class MessageMetrics(_message.Message):
    __slots__ = ("sent", "received")
    SENT_FIELD_NUMBER: _ClassVar[int]
    RECEIVED_FIELD_NUMBER: _ClassVar[int]
    sent: int
    received: int
    def __init__(self, sent: _Optional[int] = ..., received: _Optional[int] = ...) -> None: ...

class ReattachMetrics(_message.Message):
    __slots__ = ("attempts", "successes", "failures")
    ATTEMPTS_FIELD_NUMBER: _ClassVar[int]
    SUCCESSES_FIELD_NUMBER: _ClassVar[int]
    FAILURES_FIELD_NUMBER: _ClassVar[int]
    attempts: int
    successes: int
    failures: int
    def __init__(self, attempts: _Optional[int] = ..., successes: _Optional[int] = ..., failures: _Optional[int] = ...) -> None: ...

class RecoveryMetrics(_message.Message):
    __slots__ = ("recovered", "orphaned_metadata", "orphaned_tmux", "attach_retries", "preserved_for_future_recovery")
    RECOVERED_FIELD_NUMBER: _ClassVar[int]
    ORPHANED_METADATA_FIELD_NUMBER: _ClassVar[int]
    ORPHANED_TMUX_FIELD_NUMBER: _ClassVar[int]
    ATTACH_RETRIES_FIELD_NUMBER: _ClassVar[int]
    PRESERVED_FOR_FUTURE_RECOVERY_FIELD_NUMBER: _ClassVar[int]
    recovered: int
    orphaned_metadata: int
    orphaned_tmux: int
    attach_retries: int
    preserved_for_future_recovery: int
    def __init__(self, recovered: _Optional[int] = ..., orphaned_metadata: _Optional[int] = ..., orphaned_tmux: _Optional[int] = ..., attach_retries: _Optional[int] = ..., preserved_for_future_recovery: _Optional[int] = ...) -> None: ...
