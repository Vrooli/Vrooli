from buf.validate import validate_pb2 as _validate_pb2
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from typing import ClassVar as _ClassVar, Optional as _Optional

DESCRIPTOR: _descriptor.FileDescriptor

class Settings(_message.Message):
    __slots__ = ("theme", "default_mode", "default_delay_seconds", "auto_fixup", "max_fixup_attempts", "max_auto_rounds", "auto_initialize_workshop", "auto_advance_workshop", "auto_cascade_workshop", "agent_max_turns", "agent_timeout_seconds", "agent_requires_approval", "search_debounce_ms", "toast_duration_ms", "confirm_destructive_actions")
    THEME_FIELD_NUMBER: _ClassVar[int]
    DEFAULT_MODE_FIELD_NUMBER: _ClassVar[int]
    DEFAULT_DELAY_SECONDS_FIELD_NUMBER: _ClassVar[int]
    AUTO_FIXUP_FIELD_NUMBER: _ClassVar[int]
    MAX_FIXUP_ATTEMPTS_FIELD_NUMBER: _ClassVar[int]
    MAX_AUTO_ROUNDS_FIELD_NUMBER: _ClassVar[int]
    AUTO_INITIALIZE_WORKSHOP_FIELD_NUMBER: _ClassVar[int]
    AUTO_ADVANCE_WORKSHOP_FIELD_NUMBER: _ClassVar[int]
    AUTO_CASCADE_WORKSHOP_FIELD_NUMBER: _ClassVar[int]
    AGENT_MAX_TURNS_FIELD_NUMBER: _ClassVar[int]
    AGENT_TIMEOUT_SECONDS_FIELD_NUMBER: _ClassVar[int]
    AGENT_REQUIRES_APPROVAL_FIELD_NUMBER: _ClassVar[int]
    SEARCH_DEBOUNCE_MS_FIELD_NUMBER: _ClassVar[int]
    TOAST_DURATION_MS_FIELD_NUMBER: _ClassVar[int]
    CONFIRM_DESTRUCTIVE_ACTIONS_FIELD_NUMBER: _ClassVar[int]
    theme: str
    default_mode: str
    default_delay_seconds: int
    auto_fixup: bool
    max_fixup_attempts: int
    max_auto_rounds: int
    auto_initialize_workshop: bool
    auto_advance_workshop: bool
    auto_cascade_workshop: bool
    agent_max_turns: int
    agent_timeout_seconds: int
    agent_requires_approval: bool
    search_debounce_ms: int
    toast_duration_ms: int
    confirm_destructive_actions: bool
    def __init__(self, theme: _Optional[str] = ..., default_mode: _Optional[str] = ..., default_delay_seconds: _Optional[int] = ..., auto_fixup: _Optional[bool] = ..., max_fixup_attempts: _Optional[int] = ..., max_auto_rounds: _Optional[int] = ..., auto_initialize_workshop: _Optional[bool] = ..., auto_advance_workshop: _Optional[bool] = ..., auto_cascade_workshop: _Optional[bool] = ..., agent_max_turns: _Optional[int] = ..., agent_timeout_seconds: _Optional[int] = ..., agent_requires_approval: _Optional[bool] = ..., search_debounce_ms: _Optional[int] = ..., toast_duration_ms: _Optional[int] = ..., confirm_destructive_actions: _Optional[bool] = ...) -> None: ...
