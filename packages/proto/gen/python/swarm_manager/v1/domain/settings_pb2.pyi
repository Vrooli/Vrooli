from buf.validate import validate_pb2 as _validate_pb2
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from typing import ClassVar as _ClassVar, Optional as _Optional

DESCRIPTOR: _descriptor.FileDescriptor

class Settings(_message.Message):
    __slots__ = ("theme", "default_mode", "auto_fixup", "max_fixup_attempts", "review_agent_enabled", "max_auto_rounds", "auto_initialize_workshop", "auto_advance_workshop", "auto_cascade_workshop", "auto_advance_delay_seconds", "agent_max_turns", "agent_timeout_seconds", "agent_requires_approval", "search_debounce_ms", "toast_duration_ms", "confirm_destructive_actions", "review_code_quality_min_score", "review_test_min_pass_rate", "review_max_blocking_violations", "review_max_warnings", "review_require_screenshots", "review_require_tests", "max_concurrent_executions", "max_queue_depth", "circuit_breaker_threshold", "circuit_breaker_cooldown_minutes", "execution_cost_cap_per_run", "cost_per_turn_estimate")
    THEME_FIELD_NUMBER: _ClassVar[int]
    DEFAULT_MODE_FIELD_NUMBER: _ClassVar[int]
    AUTO_FIXUP_FIELD_NUMBER: _ClassVar[int]
    MAX_FIXUP_ATTEMPTS_FIELD_NUMBER: _ClassVar[int]
    REVIEW_AGENT_ENABLED_FIELD_NUMBER: _ClassVar[int]
    MAX_AUTO_ROUNDS_FIELD_NUMBER: _ClassVar[int]
    AUTO_INITIALIZE_WORKSHOP_FIELD_NUMBER: _ClassVar[int]
    AUTO_ADVANCE_WORKSHOP_FIELD_NUMBER: _ClassVar[int]
    AUTO_CASCADE_WORKSHOP_FIELD_NUMBER: _ClassVar[int]
    AUTO_ADVANCE_DELAY_SECONDS_FIELD_NUMBER: _ClassVar[int]
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
    MAX_CONCURRENT_EXECUTIONS_FIELD_NUMBER: _ClassVar[int]
    MAX_QUEUE_DEPTH_FIELD_NUMBER: _ClassVar[int]
    CIRCUIT_BREAKER_THRESHOLD_FIELD_NUMBER: _ClassVar[int]
    CIRCUIT_BREAKER_COOLDOWN_MINUTES_FIELD_NUMBER: _ClassVar[int]
    EXECUTION_COST_CAP_PER_RUN_FIELD_NUMBER: _ClassVar[int]
    COST_PER_TURN_ESTIMATE_FIELD_NUMBER: _ClassVar[int]
    theme: str
    default_mode: str
    auto_fixup: bool
    max_fixup_attempts: int
    review_agent_enabled: bool
    max_auto_rounds: int
    auto_initialize_workshop: bool
    auto_advance_workshop: bool
    auto_cascade_workshop: bool
    auto_advance_delay_seconds: int
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
    max_concurrent_executions: int
    max_queue_depth: int
    circuit_breaker_threshold: int
    circuit_breaker_cooldown_minutes: int
    execution_cost_cap_per_run: float
    cost_per_turn_estimate: float
    def __init__(self, theme: _Optional[str] = ..., default_mode: _Optional[str] = ..., auto_fixup: _Optional[bool] = ..., max_fixup_attempts: _Optional[int] = ..., review_agent_enabled: _Optional[bool] = ..., max_auto_rounds: _Optional[int] = ..., auto_initialize_workshop: _Optional[bool] = ..., auto_advance_workshop: _Optional[bool] = ..., auto_cascade_workshop: _Optional[bool] = ..., auto_advance_delay_seconds: _Optional[int] = ..., agent_max_turns: _Optional[int] = ..., agent_timeout_seconds: _Optional[int] = ..., agent_requires_approval: _Optional[bool] = ..., search_debounce_ms: _Optional[int] = ..., toast_duration_ms: _Optional[int] = ..., confirm_destructive_actions: _Optional[bool] = ..., review_code_quality_min_score: _Optional[float] = ..., review_test_min_pass_rate: _Optional[float] = ..., review_max_blocking_violations: _Optional[int] = ..., review_max_warnings: _Optional[int] = ..., review_require_screenshots: _Optional[bool] = ..., review_require_tests: _Optional[bool] = ..., max_concurrent_executions: _Optional[int] = ..., max_queue_depth: _Optional[int] = ..., circuit_breaker_threshold: _Optional[int] = ..., circuit_breaker_cooldown_minutes: _Optional[int] = ..., execution_cost_cap_per_run: _Optional[float] = ..., cost_per_turn_estimate: _Optional[float] = ...) -> None: ...
