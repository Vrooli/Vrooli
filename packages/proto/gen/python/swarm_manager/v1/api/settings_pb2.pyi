from buf.validate import validate_pb2 as _validate_pb2
from swarm_manager.v1.domain import settings_pb2 as _settings_pb2
from google.protobuf.internal import containers as _containers
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

class AutoFilerSettingsPatch(_message.Message):
    __slots__ = ("enabled", "mode", "strategy", "max_open_auto_filed", "velocity_window_days", "min_velocity_transitions", "interval_minutes", "goal_name")
    ENABLED_FIELD_NUMBER: _ClassVar[int]
    MODE_FIELD_NUMBER: _ClassVar[int]
    STRATEGY_FIELD_NUMBER: _ClassVar[int]
    MAX_OPEN_AUTO_FILED_FIELD_NUMBER: _ClassVar[int]
    VELOCITY_WINDOW_DAYS_FIELD_NUMBER: _ClassVar[int]
    MIN_VELOCITY_TRANSITIONS_FIELD_NUMBER: _ClassVar[int]
    INTERVAL_MINUTES_FIELD_NUMBER: _ClassVar[int]
    GOAL_NAME_FIELD_NUMBER: _ClassVar[int]
    enabled: bool
    mode: str
    strategy: str
    max_open_auto_filed: int
    velocity_window_days: int
    min_velocity_transitions: int
    interval_minutes: int
    goal_name: str
    def __init__(self, enabled: _Optional[bool] = ..., mode: _Optional[str] = ..., strategy: _Optional[str] = ..., max_open_auto_filed: _Optional[int] = ..., velocity_window_days: _Optional[int] = ..., min_velocity_transitions: _Optional[int] = ..., interval_minutes: _Optional[int] = ..., goal_name: _Optional[str] = ...) -> None: ...

class UpdateSettingsRequest(_message.Message):
    __slots__ = ("theme", "default_mode", "auto_fixup", "max_fixup_attempts", "review_agent_enabled", "max_auto_rounds", "auto_initialize_workshop", "auto_advance_workshop", "auto_cascade_workshop", "agent_max_turns", "agent_timeout_seconds", "search_debounce_ms", "toast_duration_ms", "delete_confirmation_levels", "review_code_quality_min_score", "review_test_min_pass_rate", "review_max_blocking_violations", "review_max_warnings", "review_require_screenshots", "review_require_tests", "lane_concurrency_limits", "max_queue_depth", "circuit_breaker_threshold", "circuit_breaker_cooldown_minutes", "execution_cost_cap_per_run", "cost_per_turn_estimate", "fix_before_feature", "auto_filer")
    class DeleteConfirmationLevelsEntry(_message.Message):
        __slots__ = ("key", "value")
        KEY_FIELD_NUMBER: _ClassVar[int]
        VALUE_FIELD_NUMBER: _ClassVar[int]
        key: str
        value: _settings_pb2.DeleteConfirmLevel
        def __init__(self, key: _Optional[str] = ..., value: _Optional[_Union[_settings_pb2.DeleteConfirmLevel, str]] = ...) -> None: ...
    class LaneConcurrencyLimitsEntry(_message.Message):
        __slots__ = ("key", "value")
        KEY_FIELD_NUMBER: _ClassVar[int]
        VALUE_FIELD_NUMBER: _ClassVar[int]
        key: str
        value: int
        def __init__(self, key: _Optional[str] = ..., value: _Optional[int] = ...) -> None: ...
    THEME_FIELD_NUMBER: _ClassVar[int]
    DEFAULT_MODE_FIELD_NUMBER: _ClassVar[int]
    AUTO_FIXUP_FIELD_NUMBER: _ClassVar[int]
    MAX_FIXUP_ATTEMPTS_FIELD_NUMBER: _ClassVar[int]
    REVIEW_AGENT_ENABLED_FIELD_NUMBER: _ClassVar[int]
    MAX_AUTO_ROUNDS_FIELD_NUMBER: _ClassVar[int]
    AUTO_INITIALIZE_WORKSHOP_FIELD_NUMBER: _ClassVar[int]
    AUTO_ADVANCE_WORKSHOP_FIELD_NUMBER: _ClassVar[int]
    AUTO_CASCADE_WORKSHOP_FIELD_NUMBER: _ClassVar[int]
    AGENT_MAX_TURNS_FIELD_NUMBER: _ClassVar[int]
    AGENT_TIMEOUT_SECONDS_FIELD_NUMBER: _ClassVar[int]
    SEARCH_DEBOUNCE_MS_FIELD_NUMBER: _ClassVar[int]
    TOAST_DURATION_MS_FIELD_NUMBER: _ClassVar[int]
    DELETE_CONFIRMATION_LEVELS_FIELD_NUMBER: _ClassVar[int]
    REVIEW_CODE_QUALITY_MIN_SCORE_FIELD_NUMBER: _ClassVar[int]
    REVIEW_TEST_MIN_PASS_RATE_FIELD_NUMBER: _ClassVar[int]
    REVIEW_MAX_BLOCKING_VIOLATIONS_FIELD_NUMBER: _ClassVar[int]
    REVIEW_MAX_WARNINGS_FIELD_NUMBER: _ClassVar[int]
    REVIEW_REQUIRE_SCREENSHOTS_FIELD_NUMBER: _ClassVar[int]
    REVIEW_REQUIRE_TESTS_FIELD_NUMBER: _ClassVar[int]
    LANE_CONCURRENCY_LIMITS_FIELD_NUMBER: _ClassVar[int]
    MAX_QUEUE_DEPTH_FIELD_NUMBER: _ClassVar[int]
    CIRCUIT_BREAKER_THRESHOLD_FIELD_NUMBER: _ClassVar[int]
    CIRCUIT_BREAKER_COOLDOWN_MINUTES_FIELD_NUMBER: _ClassVar[int]
    EXECUTION_COST_CAP_PER_RUN_FIELD_NUMBER: _ClassVar[int]
    COST_PER_TURN_ESTIMATE_FIELD_NUMBER: _ClassVar[int]
    FIX_BEFORE_FEATURE_FIELD_NUMBER: _ClassVar[int]
    AUTO_FILER_FIELD_NUMBER: _ClassVar[int]
    theme: str
    default_mode: str
    auto_fixup: bool
    max_fixup_attempts: int
    review_agent_enabled: bool
    max_auto_rounds: int
    auto_initialize_workshop: bool
    auto_advance_workshop: bool
    auto_cascade_workshop: bool
    agent_max_turns: int
    agent_timeout_seconds: int
    search_debounce_ms: int
    toast_duration_ms: int
    delete_confirmation_levels: _containers.ScalarMap[str, _settings_pb2.DeleteConfirmLevel]
    review_code_quality_min_score: float
    review_test_min_pass_rate: float
    review_max_blocking_violations: int
    review_max_warnings: int
    review_require_screenshots: bool
    review_require_tests: bool
    lane_concurrency_limits: _containers.ScalarMap[str, int]
    max_queue_depth: int
    circuit_breaker_threshold: int
    circuit_breaker_cooldown_minutes: int
    execution_cost_cap_per_run: float
    cost_per_turn_estimate: float
    fix_before_feature: str
    auto_filer: AutoFilerSettingsPatch
    def __init__(self, theme: _Optional[str] = ..., default_mode: _Optional[str] = ..., auto_fixup: _Optional[bool] = ..., max_fixup_attempts: _Optional[int] = ..., review_agent_enabled: _Optional[bool] = ..., max_auto_rounds: _Optional[int] = ..., auto_initialize_workshop: _Optional[bool] = ..., auto_advance_workshop: _Optional[bool] = ..., auto_cascade_workshop: _Optional[bool] = ..., agent_max_turns: _Optional[int] = ..., agent_timeout_seconds: _Optional[int] = ..., search_debounce_ms: _Optional[int] = ..., toast_duration_ms: _Optional[int] = ..., delete_confirmation_levels: _Optional[_Mapping[str, _settings_pb2.DeleteConfirmLevel]] = ..., review_code_quality_min_score: _Optional[float] = ..., review_test_min_pass_rate: _Optional[float] = ..., review_max_blocking_violations: _Optional[int] = ..., review_max_warnings: _Optional[int] = ..., review_require_screenshots: _Optional[bool] = ..., review_require_tests: _Optional[bool] = ..., lane_concurrency_limits: _Optional[_Mapping[str, int]] = ..., max_queue_depth: _Optional[int] = ..., circuit_breaker_threshold: _Optional[int] = ..., circuit_breaker_cooldown_minutes: _Optional[int] = ..., execution_cost_cap_per_run: _Optional[float] = ..., cost_per_turn_estimate: _Optional[float] = ..., fix_before_feature: _Optional[str] = ..., auto_filer: _Optional[_Union[AutoFilerSettingsPatch, _Mapping]] = ...) -> None: ...
