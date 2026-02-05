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

class RecommendationSourcesPatch(_message.Message):
    __slots__ = ("problems", "completeness", "tests", "coverage", "custom_focus", "scenario_notes")
    PROBLEMS_FIELD_NUMBER: _ClassVar[int]
    COMPLETENESS_FIELD_NUMBER: _ClassVar[int]
    TESTS_FIELD_NUMBER: _ClassVar[int]
    COVERAGE_FIELD_NUMBER: _ClassVar[int]
    CUSTOM_FOCUS_FIELD_NUMBER: _ClassVar[int]
    SCENARIO_NOTES_FIELD_NUMBER: _ClassVar[int]
    problems: bool
    completeness: bool
    tests: bool
    coverage: bool
    custom_focus: bool
    scenario_notes: bool
    def __init__(self, problems: _Optional[bool] = ..., completeness: _Optional[bool] = ..., tests: _Optional[bool] = ..., coverage: _Optional[bool] = ..., custom_focus: _Optional[bool] = ..., scenario_notes: _Optional[bool] = ...) -> None: ...

class RecommendationAutoSyncPatch(_message.Message):
    __slots__ = ("enabled", "interval", "last_refresh", "next_refresh", "refresh_scope")
    ENABLED_FIELD_NUMBER: _ClassVar[int]
    INTERVAL_FIELD_NUMBER: _ClassVar[int]
    LAST_REFRESH_FIELD_NUMBER: _ClassVar[int]
    NEXT_REFRESH_FIELD_NUMBER: _ClassVar[int]
    REFRESH_SCOPE_FIELD_NUMBER: _ClassVar[int]
    enabled: bool
    interval: str
    last_refresh: str
    next_refresh: str
    refresh_scope: str
    def __init__(self, enabled: _Optional[bool] = ..., interval: _Optional[str] = ..., last_refresh: _Optional[str] = ..., next_refresh: _Optional[str] = ..., refresh_scope: _Optional[str] = ...) -> None: ...

class UpdateSettingsRequest(_message.Message):
    __slots__ = ("theme", "recommendation_mode", "custom_focus", "insights_enabled", "insights_auto_analyze", "recommendation_sources", "recommendation_auto_sync")
    THEME_FIELD_NUMBER: _ClassVar[int]
    RECOMMENDATION_MODE_FIELD_NUMBER: _ClassVar[int]
    CUSTOM_FOCUS_FIELD_NUMBER: _ClassVar[int]
    INSIGHTS_ENABLED_FIELD_NUMBER: _ClassVar[int]
    INSIGHTS_AUTO_ANALYZE_FIELD_NUMBER: _ClassVar[int]
    RECOMMENDATION_SOURCES_FIELD_NUMBER: _ClassVar[int]
    RECOMMENDATION_AUTO_SYNC_FIELD_NUMBER: _ClassVar[int]
    theme: str
    recommendation_mode: str
    custom_focus: str
    insights_enabled: bool
    insights_auto_analyze: bool
    recommendation_sources: RecommendationSourcesPatch
    recommendation_auto_sync: RecommendationAutoSyncPatch
    def __init__(self, theme: _Optional[str] = ..., recommendation_mode: _Optional[str] = ..., custom_focus: _Optional[str] = ..., insights_enabled: _Optional[bool] = ..., insights_auto_analyze: _Optional[bool] = ..., recommendation_sources: _Optional[_Union[RecommendationSourcesPatch, _Mapping]] = ..., recommendation_auto_sync: _Optional[_Union[RecommendationAutoSyncPatch, _Mapping]] = ...) -> None: ...
