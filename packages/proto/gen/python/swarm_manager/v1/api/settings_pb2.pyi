from buf.validate import validate_pb2 as _validate_pb2
from swarm_manager.v1.domain import settings_pb2 as _settings_pb2
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Mapping as _Mapping
from typing import ClassVar as _ClassVar, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class SettingsResponse(_message.Message):
    __slots__ = ("settings",)
    SETTINGS_FIELD_NUMBER: _ClassVar[int]
    settings: _settings_pb2.Settings
    def __init__(self, settings: _Optional[_Union[_settings_pb2.Settings, _Mapping]] = ...) -> None: ...

class UpdateSettingsRequest(_message.Message):
    __slots__ = ("theme", "default_mode", "default_delay_seconds", "auto_fixup", "max_fixup_attempts", "max_auto_rounds", "auto_initialize_workshop", "auto_advance_workshop", "auto_cascade_workshop", "agent_max_turns", "agent_timeout_seconds", "agent_requires_approval", "search_debounce_ms", "toast_duration_ms", "confirm_destructive_actions", "review_code_quality_min_score", "review_test_min_pass_rate", "review_max_blocking_violations", "review_max_warnings", "review_require_screenshots", "review_require_tests")
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
    REVIEW_CODE_QUALITY_MIN_SCORE_FIELD_NUMBER: _ClassVar[int]
    REVIEW_TEST_MIN_PASS_RATE_FIELD_NUMBER: _ClassVar[int]
    REVIEW_MAX_BLOCKING_VIOLATIONS_FIELD_NUMBER: _ClassVar[int]
    REVIEW_MAX_WARNINGS_FIELD_NUMBER: _ClassVar[int]
    REVIEW_REQUIRE_SCREENSHOTS_FIELD_NUMBER: _ClassVar[int]
    REVIEW_REQUIRE_TESTS_FIELD_NUMBER: _ClassVar[int]
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
    review_code_quality_min_score: float
    review_test_min_pass_rate: float
    review_max_blocking_violations: int
    review_max_warnings: int
    review_require_screenshots: bool
    review_require_tests: bool
    def __init__(self, theme: _Optional[str] = ..., default_mode: _Optional[str] = ..., default_delay_seconds: _Optional[int] = ..., auto_fixup: _Optional[bool] = ..., max_fixup_attempts: _Optional[int] = ..., max_auto_rounds: _Optional[int] = ..., auto_initialize_workshop: _Optional[bool] = ..., auto_advance_workshop: _Optional[bool] = ..., auto_cascade_workshop: _Optional[bool] = ..., agent_max_turns: _Optional[int] = ..., agent_timeout_seconds: _Optional[int] = ..., agent_requires_approval: _Optional[bool] = ..., search_debounce_ms: _Optional[int] = ..., toast_duration_ms: _Optional[int] = ..., confirm_destructive_actions: _Optional[bool] = ..., review_code_quality_min_score: _Optional[float] = ..., review_test_min_pass_rate: _Optional[float] = ..., review_max_blocking_violations: _Optional[int] = ..., review_max_warnings: _Optional[int] = ..., review_require_screenshots: _Optional[bool] = ..., review_require_tests: _Optional[bool] = ...) -> None: ...
