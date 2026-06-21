from common.v1 import maturity_pb2 as _maturity_pb2
from common.v1 import metrics_pb2 as _metrics_pb2
from scenario_validation.v1 import validation_pb2 as _validation_pb2
from google.protobuf.internal import containers as _containers
from google.protobuf.internal import enum_type_wrapper as _enum_type_wrapper
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Iterable as _Iterable, Mapping as _Mapping
from typing import ClassVar as _ClassVar, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class CaptureTier(int, metaclass=_enum_type_wrapper.EnumTypeWrapper):
    __slots__ = ()
    CAPTURE_TIER_UNSPECIFIED: _ClassVar[CaptureTier]
    CAPTURE_TIER_NONE: _ClassVar[CaptureTier]
    CAPTURE_TIER_0: _ClassVar[CaptureTier]
    CAPTURE_TIER_1: _ClassVar[CaptureTier]
CAPTURE_TIER_UNSPECIFIED: CaptureTier
CAPTURE_TIER_NONE: CaptureTier
CAPTURE_TIER_0: CaptureTier
CAPTURE_TIER_1: CaptureTier

class ValidateReadinessRequest(_message.Message):
    __slots__ = ("scenario", "path")
    SCENARIO_FIELD_NUMBER: _ClassVar[int]
    PATH_FIELD_NUMBER: _ClassVar[int]
    scenario: str
    path: str
    def __init__(self, scenario: _Optional[str] = ..., path: _Optional[str] = ...) -> None: ...

class ValidateReadinessResponse(_message.Message):
    __slots__ = ("scenario", "status", "tier", "ui_framework", "surfaces", "assessment", "autofixable_count", "degraded_reason", "metrics")
    SCENARIO_FIELD_NUMBER: _ClassVar[int]
    STATUS_FIELD_NUMBER: _ClassVar[int]
    TIER_FIELD_NUMBER: _ClassVar[int]
    UI_FRAMEWORK_FIELD_NUMBER: _ClassVar[int]
    SURFACES_FIELD_NUMBER: _ClassVar[int]
    ASSESSMENT_FIELD_NUMBER: _ClassVar[int]
    AUTOFIXABLE_COUNT_FIELD_NUMBER: _ClassVar[int]
    DEGRADED_REASON_FIELD_NUMBER: _ClassVar[int]
    METRICS_FIELD_NUMBER: _ClassVar[int]
    scenario: str
    status: _validation_pb2.ValidationStatus
    tier: CaptureTier
    ui_framework: str
    surfaces: _containers.RepeatedScalarFieldContainer[str]
    assessment: _maturity_pb2.MaturityAssessment
    autofixable_count: int
    degraded_reason: str
    metrics: _metrics_pb2.ExecutionMetrics
    def __init__(self, scenario: _Optional[str] = ..., status: _Optional[_Union[_validation_pb2.ValidationStatus, str]] = ..., tier: _Optional[_Union[CaptureTier, str]] = ..., ui_framework: _Optional[str] = ..., surfaces: _Optional[_Iterable[str]] = ..., assessment: _Optional[_Union[_maturity_pb2.MaturityAssessment, _Mapping]] = ..., autofixable_count: _Optional[int] = ..., degraded_reason: _Optional[str] = ..., metrics: _Optional[_Union[_metrics_pb2.ExecutionMetrics, _Mapping]] = ...) -> None: ...

class ReadinessFixRequest(_message.Message):
    __slots__ = ("scenario", "path", "rule_ids")
    SCENARIO_FIELD_NUMBER: _ClassVar[int]
    PATH_FIELD_NUMBER: _ClassVar[int]
    RULE_IDS_FIELD_NUMBER: _ClassVar[int]
    scenario: str
    path: str
    rule_ids: _containers.RepeatedScalarFieldContainer[str]
    def __init__(self, scenario: _Optional[str] = ..., path: _Optional[str] = ..., rule_ids: _Optional[_Iterable[str]] = ...) -> None: ...

class ReadinessFixResponse(_message.Message):
    __slots__ = ("scenario", "applied", "candidates", "messages")
    SCENARIO_FIELD_NUMBER: _ClassVar[int]
    APPLIED_FIELD_NUMBER: _ClassVar[int]
    CANDIDATES_FIELD_NUMBER: _ClassVar[int]
    MESSAGES_FIELD_NUMBER: _ClassVar[int]
    scenario: str
    applied: bool
    candidates: _containers.RepeatedCompositeFieldContainer[_validation_pb2.FixCandidate]
    messages: _containers.RepeatedScalarFieldContainer[str]
    def __init__(self, scenario: _Optional[str] = ..., applied: _Optional[bool] = ..., candidates: _Optional[_Iterable[_Union[_validation_pb2.FixCandidate, _Mapping]]] = ..., messages: _Optional[_Iterable[str]] = ...) -> None: ...
