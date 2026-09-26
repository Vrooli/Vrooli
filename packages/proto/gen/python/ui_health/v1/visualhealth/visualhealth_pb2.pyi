from google.protobuf.internal import containers as _containers
from google.protobuf.internal import enum_type_wrapper as _enum_type_wrapper
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Iterable as _Iterable, Mapping as _Mapping
from typing import ClassVar as _ClassVar, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class VisualSeverity(int, metaclass=_enum_type_wrapper.EnumTypeWrapper):
    __slots__ = ()
    VISUAL_SEVERITY_UNSPECIFIED: _ClassVar[VisualSeverity]
    VISUAL_SEVERITY_INFO: _ClassVar[VisualSeverity]
    VISUAL_SEVERITY_WARNING: _ClassVar[VisualSeverity]
    VISUAL_SEVERITY_ERROR: _ClassVar[VisualSeverity]

class VisualCategory(int, metaclass=_enum_type_wrapper.EnumTypeWrapper):
    __slots__ = ()
    VISUAL_CATEGORY_UNSPECIFIED: _ClassVar[VisualCategory]
    VISUAL_CATEGORY_PIXEL: _ClassVar[VisualCategory]
    VISUAL_CATEGORY_DOM: _ClassVar[VisualCategory]
    VISUAL_CATEGORY_LAYOUT: _ClassVar[VisualCategory]
    VISUAL_CATEGORY_ASSET: _ClassVar[VisualCategory]
    VISUAL_CATEGORY_FOCUS: _ClassVar[VisualCategory]
VISUAL_SEVERITY_UNSPECIFIED: VisualSeverity
VISUAL_SEVERITY_INFO: VisualSeverity
VISUAL_SEVERITY_WARNING: VisualSeverity
VISUAL_SEVERITY_ERROR: VisualSeverity
VISUAL_CATEGORY_UNSPECIFIED: VisualCategory
VISUAL_CATEGORY_PIXEL: VisualCategory
VISUAL_CATEGORY_DOM: VisualCategory
VISUAL_CATEGORY_LAYOUT: VisualCategory
VISUAL_CATEGORY_ASSET: VisualCategory
VISUAL_CATEGORY_FOCUS: VisualCategory

class ArtifactRef(_message.Message):
    __slots__ = ("scenario", "run_id", "step_id", "rel_path", "uri", "media_type")
    SCENARIO_FIELD_NUMBER: _ClassVar[int]
    RUN_ID_FIELD_NUMBER: _ClassVar[int]
    STEP_ID_FIELD_NUMBER: _ClassVar[int]
    REL_PATH_FIELD_NUMBER: _ClassVar[int]
    URI_FIELD_NUMBER: _ClassVar[int]
    MEDIA_TYPE_FIELD_NUMBER: _ClassVar[int]
    scenario: str
    run_id: str
    step_id: str
    rel_path: str
    uri: str
    media_type: str
    def __init__(self, scenario: _Optional[str] = ..., run_id: _Optional[str] = ..., step_id: _Optional[str] = ..., rel_path: _Optional[str] = ..., uri: _Optional[str] = ..., media_type: _Optional[str] = ...) -> None: ...

class Viewport(_message.Message):
    __slots__ = ("width", "height", "device_profile", "device_scale_factor")
    WIDTH_FIELD_NUMBER: _ClassVar[int]
    HEIGHT_FIELD_NUMBER: _ClassVar[int]
    DEVICE_PROFILE_FIELD_NUMBER: _ClassVar[int]
    DEVICE_SCALE_FACTOR_FIELD_NUMBER: _ClassVar[int]
    width: int
    height: int
    device_profile: str
    device_scale_factor: float
    def __init__(self, width: _Optional[int] = ..., height: _Optional[int] = ..., device_profile: _Optional[str] = ..., device_scale_factor: _Optional[float] = ...) -> None: ...

class ConsoleEntry(_message.Message):
    __slots__ = ("level", "message")
    LEVEL_FIELD_NUMBER: _ClassVar[int]
    MESSAGE_FIELD_NUMBER: _ClassVar[int]
    level: str
    message: str
    def __init__(self, level: _Optional[str] = ..., message: _Optional[str] = ...) -> None: ...

class NetworkEntry(_message.Message):
    __slots__ = ("url", "method", "resource_type", "status", "error_text")
    URL_FIELD_NUMBER: _ClassVar[int]
    METHOD_FIELD_NUMBER: _ClassVar[int]
    RESOURCE_TYPE_FIELD_NUMBER: _ClassVar[int]
    STATUS_FIELD_NUMBER: _ClassVar[int]
    ERROR_TEXT_FIELD_NUMBER: _ClassVar[int]
    url: str
    method: str
    resource_type: str
    status: int
    error_text: str
    def __init__(self, url: _Optional[str] = ..., method: _Optional[str] = ..., resource_type: _Optional[str] = ..., status: _Optional[int] = ..., error_text: _Optional[str] = ...) -> None: ...

class PageError(_message.Message):
    __slots__ = ("message", "stack")
    MESSAGE_FIELD_NUMBER: _ClassVar[int]
    STACK_FIELD_NUMBER: _ClassVar[int]
    message: str
    stack: str
    def __init__(self, message: _Optional[str] = ..., stack: _Optional[str] = ...) -> None: ...

class VisualStepArtifact(_message.Message):
    __slots__ = ("step_id", "label", "url", "viewport", "screenshot_ref", "screenshot_png", "dom_ref", "dom_html", "layout_ref", "layout_json", "console", "network", "page_errors")
    STEP_ID_FIELD_NUMBER: _ClassVar[int]
    LABEL_FIELD_NUMBER: _ClassVar[int]
    URL_FIELD_NUMBER: _ClassVar[int]
    VIEWPORT_FIELD_NUMBER: _ClassVar[int]
    SCREENSHOT_REF_FIELD_NUMBER: _ClassVar[int]
    SCREENSHOT_PNG_FIELD_NUMBER: _ClassVar[int]
    DOM_REF_FIELD_NUMBER: _ClassVar[int]
    DOM_HTML_FIELD_NUMBER: _ClassVar[int]
    LAYOUT_REF_FIELD_NUMBER: _ClassVar[int]
    LAYOUT_JSON_FIELD_NUMBER: _ClassVar[int]
    CONSOLE_FIELD_NUMBER: _ClassVar[int]
    NETWORK_FIELD_NUMBER: _ClassVar[int]
    PAGE_ERRORS_FIELD_NUMBER: _ClassVar[int]
    step_id: str
    label: str
    url: str
    viewport: Viewport
    screenshot_ref: ArtifactRef
    screenshot_png: bytes
    dom_ref: ArtifactRef
    dom_html: str
    layout_ref: ArtifactRef
    layout_json: str
    console: _containers.RepeatedCompositeFieldContainer[ConsoleEntry]
    network: _containers.RepeatedCompositeFieldContainer[NetworkEntry]
    page_errors: _containers.RepeatedCompositeFieldContainer[PageError]
    def __init__(self, step_id: _Optional[str] = ..., label: _Optional[str] = ..., url: _Optional[str] = ..., viewport: _Optional[_Union[Viewport, _Mapping]] = ..., screenshot_ref: _Optional[_Union[ArtifactRef, _Mapping]] = ..., screenshot_png: _Optional[bytes] = ..., dom_ref: _Optional[_Union[ArtifactRef, _Mapping]] = ..., dom_html: _Optional[str] = ..., layout_ref: _Optional[_Union[ArtifactRef, _Mapping]] = ..., layout_json: _Optional[str] = ..., console: _Optional[_Iterable[_Union[ConsoleEntry, _Mapping]]] = ..., network: _Optional[_Iterable[_Union[NetworkEntry, _Mapping]]] = ..., page_errors: _Optional[_Iterable[_Union[PageError, _Mapping]]] = ...) -> None: ...

class VisualMetric(_message.Message):
    __slots__ = ("name", "value")
    NAME_FIELD_NUMBER: _ClassVar[int]
    VALUE_FIELD_NUMBER: _ClassVar[int]
    name: str
    value: float
    def __init__(self, name: _Optional[str] = ..., value: _Optional[float] = ...) -> None: ...

class VisualFinding(_message.Message):
    __slots__ = ("code", "severity", "category", "message", "location", "evidence", "remediation", "step_id", "metrics")
    CODE_FIELD_NUMBER: _ClassVar[int]
    SEVERITY_FIELD_NUMBER: _ClassVar[int]
    CATEGORY_FIELD_NUMBER: _ClassVar[int]
    MESSAGE_FIELD_NUMBER: _ClassVar[int]
    LOCATION_FIELD_NUMBER: _ClassVar[int]
    EVIDENCE_FIELD_NUMBER: _ClassVar[int]
    REMEDIATION_FIELD_NUMBER: _ClassVar[int]
    STEP_ID_FIELD_NUMBER: _ClassVar[int]
    METRICS_FIELD_NUMBER: _ClassVar[int]
    code: str
    severity: VisualSeverity
    category: VisualCategory
    message: str
    location: str
    evidence: str
    remediation: str
    step_id: str
    metrics: _containers.RepeatedCompositeFieldContainer[VisualMetric]
    def __init__(self, code: _Optional[str] = ..., severity: _Optional[_Union[VisualSeverity, str]] = ..., category: _Optional[_Union[VisualCategory, str]] = ..., message: _Optional[str] = ..., location: _Optional[str] = ..., evidence: _Optional[str] = ..., remediation: _Optional[str] = ..., step_id: _Optional[str] = ..., metrics: _Optional[_Iterable[_Union[VisualMetric, _Mapping]]] = ...) -> None: ...

class StepVerdict(_message.Message):
    __slots__ = ("step_id", "status", "findings", "metrics")
    STEP_ID_FIELD_NUMBER: _ClassVar[int]
    STATUS_FIELD_NUMBER: _ClassVar[int]
    FINDINGS_FIELD_NUMBER: _ClassVar[int]
    METRICS_FIELD_NUMBER: _ClassVar[int]
    step_id: str
    status: str
    findings: _containers.RepeatedCompositeFieldContainer[VisualFinding]
    metrics: _containers.RepeatedCompositeFieldContainer[VisualMetric]
    def __init__(self, step_id: _Optional[str] = ..., status: _Optional[str] = ..., findings: _Optional[_Iterable[_Union[VisualFinding, _Mapping]]] = ..., metrics: _Optional[_Iterable[_Union[VisualMetric, _Mapping]]] = ...) -> None: ...

class AnalyzeArtifactsRequest(_message.Message):
    __slots__ = ("scenario", "run_id", "steps")
    SCENARIO_FIELD_NUMBER: _ClassVar[int]
    RUN_ID_FIELD_NUMBER: _ClassVar[int]
    STEPS_FIELD_NUMBER: _ClassVar[int]
    scenario: str
    run_id: str
    steps: _containers.RepeatedCompositeFieldContainer[VisualStepArtifact]
    def __init__(self, scenario: _Optional[str] = ..., run_id: _Optional[str] = ..., steps: _Optional[_Iterable[_Union[VisualStepArtifact, _Mapping]]] = ...) -> None: ...

class AnalyzeArtifactsResponse(_message.Message):
    __slots__ = ("scenario", "run_id", "verdict", "steps", "findings")
    SCENARIO_FIELD_NUMBER: _ClassVar[int]
    RUN_ID_FIELD_NUMBER: _ClassVar[int]
    VERDICT_FIELD_NUMBER: _ClassVar[int]
    STEPS_FIELD_NUMBER: _ClassVar[int]
    FINDINGS_FIELD_NUMBER: _ClassVar[int]
    scenario: str
    run_id: str
    verdict: str
    steps: _containers.RepeatedCompositeFieldContainer[StepVerdict]
    findings: _containers.RepeatedCompositeFieldContainer[VisualFinding]
    def __init__(self, scenario: _Optional[str] = ..., run_id: _Optional[str] = ..., verdict: _Optional[str] = ..., steps: _Optional[_Iterable[_Union[StepVerdict, _Mapping]]] = ..., findings: _Optional[_Iterable[_Union[VisualFinding, _Mapping]]] = ...) -> None: ...

class CompareArtifact(_message.Message):
    __slots__ = ("page", "label", "screenshot_ref", "screenshot_png")
    PAGE_FIELD_NUMBER: _ClassVar[int]
    LABEL_FIELD_NUMBER: _ClassVar[int]
    SCREENSHOT_REF_FIELD_NUMBER: _ClassVar[int]
    SCREENSHOT_PNG_FIELD_NUMBER: _ClassVar[int]
    page: str
    label: str
    screenshot_ref: ArtifactRef
    screenshot_png: bytes
    def __init__(self, page: _Optional[str] = ..., label: _Optional[str] = ..., screenshot_ref: _Optional[_Union[ArtifactRef, _Mapping]] = ..., screenshot_png: _Optional[bytes] = ...) -> None: ...

class CompareArtifactsRequest(_message.Message):
    __slots__ = ("scenario", "base_run_id", "current_run_id", "base", "current")
    SCENARIO_FIELD_NUMBER: _ClassVar[int]
    BASE_RUN_ID_FIELD_NUMBER: _ClassVar[int]
    CURRENT_RUN_ID_FIELD_NUMBER: _ClassVar[int]
    BASE_FIELD_NUMBER: _ClassVar[int]
    CURRENT_FIELD_NUMBER: _ClassVar[int]
    scenario: str
    base_run_id: str
    current_run_id: str
    base: _containers.RepeatedCompositeFieldContainer[CompareArtifact]
    current: _containers.RepeatedCompositeFieldContainer[CompareArtifact]
    def __init__(self, scenario: _Optional[str] = ..., base_run_id: _Optional[str] = ..., current_run_id: _Optional[str] = ..., base: _Optional[_Iterable[_Union[CompareArtifact, _Mapping]]] = ..., current: _Optional[_Iterable[_Union[CompareArtifact, _Mapping]]] = ...) -> None: ...

class VisualDelta(_message.Message):
    __slots__ = ("page", "label", "status", "changed_fraction")
    PAGE_FIELD_NUMBER: _ClassVar[int]
    LABEL_FIELD_NUMBER: _ClassVar[int]
    STATUS_FIELD_NUMBER: _ClassVar[int]
    CHANGED_FRACTION_FIELD_NUMBER: _ClassVar[int]
    page: str
    label: str
    status: str
    changed_fraction: float
    def __init__(self, page: _Optional[str] = ..., label: _Optional[str] = ..., status: _Optional[str] = ..., changed_fraction: _Optional[float] = ...) -> None: ...

class CompareArtifactsResponse(_message.Message):
    __slots__ = ("deltas",)
    DELTAS_FIELD_NUMBER: _ClassVar[int]
    deltas: _containers.RepeatedCompositeFieldContainer[VisualDelta]
    def __init__(self, deltas: _Optional[_Iterable[_Union[VisualDelta, _Mapping]]] = ...) -> None: ...

class ListRulesRequest(_message.Message):
    __slots__ = ()
    def __init__(self) -> None: ...

class VisualRule(_message.Message):
    __slots__ = ("id", "category", "severity", "required_artifacts", "remediation")
    ID_FIELD_NUMBER: _ClassVar[int]
    CATEGORY_FIELD_NUMBER: _ClassVar[int]
    SEVERITY_FIELD_NUMBER: _ClassVar[int]
    REQUIRED_ARTIFACTS_FIELD_NUMBER: _ClassVar[int]
    REMEDIATION_FIELD_NUMBER: _ClassVar[int]
    id: str
    category: VisualCategory
    severity: VisualSeverity
    required_artifacts: _containers.RepeatedScalarFieldContainer[str]
    remediation: str
    def __init__(self, id: _Optional[str] = ..., category: _Optional[_Union[VisualCategory, str]] = ..., severity: _Optional[_Union[VisualSeverity, str]] = ..., required_artifacts: _Optional[_Iterable[str]] = ..., remediation: _Optional[str] = ...) -> None: ...

class ListRulesResponse(_message.Message):
    __slots__ = ("rules",)
    RULES_FIELD_NUMBER: _ClassVar[int]
    rules: _containers.RepeatedCompositeFieldContainer[VisualRule]
    def __init__(self, rules: _Optional[_Iterable[_Union[VisualRule, _Mapping]]] = ...) -> None: ...
