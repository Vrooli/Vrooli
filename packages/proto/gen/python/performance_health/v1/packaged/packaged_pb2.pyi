from google.protobuf.internal import containers as _containers
from google.protobuf.internal import enum_type_wrapper as _enum_type_wrapper
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Iterable as _Iterable, Mapping as _Mapping
from typing import ClassVar as _ClassVar, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class PackagedLaunchMeasurement(_message.Message):
    __slots__ = ("schema_version", "sample_id", "scenario", "ramp", "strategy", "platform", "captured_at", "experiment_id", "host_fingerprint", "artifact_digest", "condition", "display", "profiler_mode", "power_state", "marks", "metrics", "process_roles", "app_interactive_source", "platform_detail", "toggle_attribution", "journey_evidence")
    SCHEMA_VERSION_FIELD_NUMBER: _ClassVar[int]
    SAMPLE_ID_FIELD_NUMBER: _ClassVar[int]
    SCENARIO_FIELD_NUMBER: _ClassVar[int]
    RAMP_FIELD_NUMBER: _ClassVar[int]
    STRATEGY_FIELD_NUMBER: _ClassVar[int]
    PLATFORM_FIELD_NUMBER: _ClassVar[int]
    CAPTURED_AT_FIELD_NUMBER: _ClassVar[int]
    EXPERIMENT_ID_FIELD_NUMBER: _ClassVar[int]
    HOST_FINGERPRINT_FIELD_NUMBER: _ClassVar[int]
    ARTIFACT_DIGEST_FIELD_NUMBER: _ClassVar[int]
    CONDITION_FIELD_NUMBER: _ClassVar[int]
    DISPLAY_FIELD_NUMBER: _ClassVar[int]
    PROFILER_MODE_FIELD_NUMBER: _ClassVar[int]
    POWER_STATE_FIELD_NUMBER: _ClassVar[int]
    MARKS_FIELD_NUMBER: _ClassVar[int]
    METRICS_FIELD_NUMBER: _ClassVar[int]
    PROCESS_ROLES_FIELD_NUMBER: _ClassVar[int]
    APP_INTERACTIVE_SOURCE_FIELD_NUMBER: _ClassVar[int]
    PLATFORM_DETAIL_FIELD_NUMBER: _ClassVar[int]
    TOGGLE_ATTRIBUTION_FIELD_NUMBER: _ClassVar[int]
    JOURNEY_EVIDENCE_FIELD_NUMBER: _ClassVar[int]
    schema_version: str
    sample_id: str
    scenario: str
    ramp: str
    strategy: str
    platform: str
    captured_at: str
    experiment_id: str
    host_fingerprint: str
    artifact_digest: str
    condition: str
    display: str
    profiler_mode: str
    power_state: str
    marks: _containers.RepeatedCompositeFieldContainer[LaunchMark]
    metrics: _containers.RepeatedCompositeFieldContainer[Metric]
    process_roles: _containers.RepeatedCompositeFieldContainer[ProcessRole]
    app_interactive_source: str
    platform_detail: PlatformDetail
    toggle_attribution: OptimizationToggles
    journey_evidence: JourneyEvidence
    def __init__(self, schema_version: _Optional[str] = ..., sample_id: _Optional[str] = ..., scenario: _Optional[str] = ..., ramp: _Optional[str] = ..., strategy: _Optional[str] = ..., platform: _Optional[str] = ..., captured_at: _Optional[str] = ..., experiment_id: _Optional[str] = ..., host_fingerprint: _Optional[str] = ..., artifact_digest: _Optional[str] = ..., condition: _Optional[str] = ..., display: _Optional[str] = ..., profiler_mode: _Optional[str] = ..., power_state: _Optional[str] = ..., marks: _Optional[_Iterable[_Union[LaunchMark, _Mapping]]] = ..., metrics: _Optional[_Iterable[_Union[Metric, _Mapping]]] = ..., process_roles: _Optional[_Iterable[_Union[ProcessRole, _Mapping]]] = ..., app_interactive_source: _Optional[str] = ..., platform_detail: _Optional[_Union[PlatformDetail, _Mapping]] = ..., toggle_attribution: _Optional[_Union[OptimizationToggles, _Mapping]] = ..., journey_evidence: _Optional[_Union[JourneyEvidence, _Mapping]] = ...) -> None: ...

class JourneyEvidence(_message.Message):
    __slots__ = ("journey_id", "capture_id", "disposition", "degraded_reason")
    JOURNEY_ID_FIELD_NUMBER: _ClassVar[int]
    CAPTURE_ID_FIELD_NUMBER: _ClassVar[int]
    DISPOSITION_FIELD_NUMBER: _ClassVar[int]
    DEGRADED_REASON_FIELD_NUMBER: _ClassVar[int]
    journey_id: str
    capture_id: str
    disposition: str
    degraded_reason: str
    def __init__(self, journey_id: _Optional[str] = ..., capture_id: _Optional[str] = ..., disposition: _Optional[str] = ..., degraded_reason: _Optional[str] = ...) -> None: ...

class OptimizationToggles(_message.Message):
    __slots__ = ("shared_runtime_readiness", "secure_custom_scheme_ui", "v8_snapshot_code_cache", "electron_fuses")
    SHARED_RUNTIME_READINESS_FIELD_NUMBER: _ClassVar[int]
    SECURE_CUSTOM_SCHEME_UI_FIELD_NUMBER: _ClassVar[int]
    V8_SNAPSHOT_CODE_CACHE_FIELD_NUMBER: _ClassVar[int]
    ELECTRON_FUSES_FIELD_NUMBER: _ClassVar[int]
    shared_runtime_readiness: bool
    secure_custom_scheme_ui: bool
    v8_snapshot_code_cache: bool
    electron_fuses: bool
    def __init__(self, shared_runtime_readiness: _Optional[bool] = ..., secure_custom_scheme_ui: _Optional[bool] = ..., v8_snapshot_code_cache: _Optional[bool] = ..., electron_fuses: _Optional[bool] = ...) -> None: ...

class LaunchMark(_message.Message):
    __slots__ = ("name", "elapsed_ns", "availability")
    NAME_FIELD_NUMBER: _ClassVar[int]
    ELAPSED_NS_FIELD_NUMBER: _ClassVar[int]
    AVAILABILITY_FIELD_NUMBER: _ClassVar[int]
    name: str
    elapsed_ns: int
    availability: Availability
    def __init__(self, name: _Optional[str] = ..., elapsed_ns: _Optional[int] = ..., availability: _Optional[_Union[Availability, _Mapping]] = ...) -> None: ...

class Metric(_message.Message):
    __slots__ = ("name", "value", "unit", "availability")
    NAME_FIELD_NUMBER: _ClassVar[int]
    VALUE_FIELD_NUMBER: _ClassVar[int]
    UNIT_FIELD_NUMBER: _ClassVar[int]
    AVAILABILITY_FIELD_NUMBER: _ClassVar[int]
    name: str
    value: float
    unit: str
    availability: Availability
    def __init__(self, name: _Optional[str] = ..., value: _Optional[float] = ..., unit: _Optional[str] = ..., availability: _Optional[_Union[Availability, _Mapping]] = ...) -> None: ...

class Availability(_message.Message):
    __slots__ = ("state", "reason")
    class State(int, metaclass=_enum_type_wrapper.EnumTypeWrapper):
        __slots__ = ()
        STATE_UNSPECIFIED: _ClassVar[Availability.State]
        STATE_AVAILABLE: _ClassVar[Availability.State]
        STATE_UNAVAILABLE: _ClassVar[Availability.State]
    STATE_UNSPECIFIED: Availability.State
    STATE_AVAILABLE: Availability.State
    STATE_UNAVAILABLE: Availability.State
    STATE_FIELD_NUMBER: _ClassVar[int]
    REASON_FIELD_NUMBER: _ClassVar[int]
    state: Availability.State
    reason: str
    def __init__(self, state: _Optional[_Union[Availability.State, str]] = ..., reason: _Optional[str] = ...) -> None: ...

class ProcessRole(_message.Message):
    __slots__ = ("role", "generic_role", "process_count", "rss_mb", "pss_mb", "uss_mb", "availability")
    ROLE_FIELD_NUMBER: _ClassVar[int]
    GENERIC_ROLE_FIELD_NUMBER: _ClassVar[int]
    PROCESS_COUNT_FIELD_NUMBER: _ClassVar[int]
    RSS_MB_FIELD_NUMBER: _ClassVar[int]
    PSS_MB_FIELD_NUMBER: _ClassVar[int]
    USS_MB_FIELD_NUMBER: _ClassVar[int]
    AVAILABILITY_FIELD_NUMBER: _ClassVar[int]
    role: str
    generic_role: str
    process_count: int
    rss_mb: float
    pss_mb: float
    uss_mb: float
    availability: Availability
    def __init__(self, role: _Optional[str] = ..., generic_role: _Optional[str] = ..., process_count: _Optional[int] = ..., rss_mb: _Optional[float] = ..., pss_mb: _Optional[float] = ..., uss_mb: _Optional[float] = ..., availability: _Optional[_Union[Availability, _Mapping]] = ...) -> None: ...

class PlatformDetail(_message.Message):
    __slots__ = ("linux_desktop",)
    LINUX_DESKTOP_FIELD_NUMBER: _ClassVar[int]
    linux_desktop: LinuxDesktopDetail
    def __init__(self, linux_desktop: _Optional[_Union[LinuxDesktopDetail, _Mapping]] = ...) -> None: ...

class LinuxDesktopDetail(_message.Message):
    __slots__ = ("artifact_page_cache_residency_pct",)
    ARTIFACT_PAGE_CACHE_RESIDENCY_PCT_FIELD_NUMBER: _ClassVar[int]
    artifact_page_cache_residency_pct: float
    def __init__(self, artifact_page_cache_residency_pct: _Optional[float] = ...) -> None: ...

class ExperimentRequest(_message.Message):
    __slots__ = ("manifest_json",)
    MANIFEST_JSON_FIELD_NUMBER: _ClassVar[int]
    manifest_json: str
    def __init__(self, manifest_json: _Optional[str] = ...) -> None: ...

class ExperimentResponse(_message.Message):
    __slots__ = ("valid", "error")
    VALID_FIELD_NUMBER: _ClassVar[int]
    ERROR_FIELD_NUMBER: _ClassVar[int]
    valid: bool
    error: str
    def __init__(self, valid: _Optional[bool] = ..., error: _Optional[str] = ...) -> None: ...

class CaptureRequest(_message.Message):
    __slots__ = ("request_json",)
    REQUEST_JSON_FIELD_NUMBER: _ClassVar[int]
    request_json: str
    def __init__(self, request_json: _Optional[str] = ...) -> None: ...

class CaptureResponse(_message.Message):
    __slots__ = ("result_json",)
    RESULT_JSON_FIELD_NUMBER: _ClassVar[int]
    result_json: str
    def __init__(self, result_json: _Optional[str] = ...) -> None: ...

class ReportRequest(_message.Message):
    __slots__ = ("query_json",)
    QUERY_JSON_FIELD_NUMBER: _ClassVar[int]
    query_json: str
    def __init__(self, query_json: _Optional[str] = ...) -> None: ...

class ReportResponse(_message.Message):
    __slots__ = ("report_json", "markdown")
    REPORT_JSON_FIELD_NUMBER: _ClassVar[int]
    MARKDOWN_FIELD_NUMBER: _ClassVar[int]
    report_json: str
    markdown: str
    def __init__(self, report_json: _Optional[str] = ..., markdown: _Optional[str] = ...) -> None: ...
