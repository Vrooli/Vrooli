from common.v1 import metrics_pb2 as _metrics_pb2
from google.protobuf.internal import containers as _containers
from google.protobuf.internal import enum_type_wrapper as _enum_type_wrapper
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Iterable as _Iterable, Mapping as _Mapping
from typing import ClassVar as _ClassVar, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class LighthouseOutcome(int, metaclass=_enum_type_wrapper.EnumTypeWrapper):
    __slots__ = ()
    LIGHTHOUSE_OUTCOME_UNSPECIFIED: _ClassVar[LighthouseOutcome]
    LIGHTHOUSE_OUTCOME_SCORED: _ClassVar[LighthouseOutcome]
    LIGHTHOUSE_OUTCOME_SKIPPED: _ClassVar[LighthouseOutcome]
    LIGHTHOUSE_OUTCOME_FAILED: _ClassVar[LighthouseOutcome]
LIGHTHOUSE_OUTCOME_UNSPECIFIED: LighthouseOutcome
LIGHTHOUSE_OUTCOME_SCORED: LighthouseOutcome
LIGHTHOUSE_OUTCOME_SKIPPED: LighthouseOutcome
LIGHTHOUSE_OUTCOME_FAILED: LighthouseOutcome

class RunLighthouseRequest(_message.Message):
    __slots__ = ("scenario", "path")
    SCENARIO_FIELD_NUMBER: _ClassVar[int]
    PATH_FIELD_NUMBER: _ClassVar[int]
    scenario: str
    path: str
    def __init__(self, scenario: _Optional[str] = ..., path: _Optional[str] = ...) -> None: ...

class RunLighthouseResponse(_message.Message):
    __slots__ = ("scenario", "outcome", "pages", "reason", "metrics")
    SCENARIO_FIELD_NUMBER: _ClassVar[int]
    OUTCOME_FIELD_NUMBER: _ClassVar[int]
    PAGES_FIELD_NUMBER: _ClassVar[int]
    REASON_FIELD_NUMBER: _ClassVar[int]
    METRICS_FIELD_NUMBER: _ClassVar[int]
    scenario: str
    outcome: LighthouseOutcome
    pages: _containers.RepeatedCompositeFieldContainer[PageScore]
    reason: str
    metrics: _metrics_pb2.ExecutionMetrics
    def __init__(self, scenario: _Optional[str] = ..., outcome: _Optional[_Union[LighthouseOutcome, str]] = ..., pages: _Optional[_Iterable[_Union[PageScore, _Mapping]]] = ..., reason: _Optional[str] = ..., metrics: _Optional[_Union[_metrics_pb2.ExecutionMetrics, _Mapping]] = ...) -> None: ...

class PageScore(_message.Message):
    __slots__ = ("url", "performance", "accessibility", "best_practices", "seo", "violations")
    URL_FIELD_NUMBER: _ClassVar[int]
    PERFORMANCE_FIELD_NUMBER: _ClassVar[int]
    ACCESSIBILITY_FIELD_NUMBER: _ClassVar[int]
    BEST_PRACTICES_FIELD_NUMBER: _ClassVar[int]
    SEO_FIELD_NUMBER: _ClassVar[int]
    VIOLATIONS_FIELD_NUMBER: _ClassVar[int]
    url: str
    performance: float
    accessibility: float
    best_practices: float
    seo: float
    violations: _containers.RepeatedScalarFieldContainer[str]
    def __init__(self, url: _Optional[str] = ..., performance: _Optional[float] = ..., accessibility: _Optional[float] = ..., best_practices: _Optional[float] = ..., seo: _Optional[float] = ..., violations: _Optional[_Iterable[str]] = ...) -> None: ...
