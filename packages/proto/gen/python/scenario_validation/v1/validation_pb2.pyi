from common.v1 import maturity_pb2 as _maturity_pb2
from common.v1 import metrics_pb2 as _metrics_pb2
from google.protobuf import any_pb2 as _any_pb2
from google.protobuf.internal import containers as _containers
from google.protobuf.internal import enum_type_wrapper as _enum_type_wrapper
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Iterable as _Iterable, Mapping as _Mapping
from typing import ClassVar as _ClassVar, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class ValidationStatus(int, metaclass=_enum_type_wrapper.EnumTypeWrapper):
    __slots__ = ()
    VALIDATION_STATUS_UNSPECIFIED: _ClassVar[ValidationStatus]
    VALIDATION_STATUS_PASSED: _ClassVar[ValidationStatus]
    VALIDATION_STATUS_FAILED: _ClassVar[ValidationStatus]
    VALIDATION_STATUS_DEGRADED: _ClassVar[ValidationStatus]
    VALIDATION_STATUS_ERROR: _ClassVar[ValidationStatus]
    VALIDATION_STATUS_SKIPPED: _ClassVar[ValidationStatus]
VALIDATION_STATUS_UNSPECIFIED: ValidationStatus
VALIDATION_STATUS_PASSED: ValidationStatus
VALIDATION_STATUS_FAILED: ValidationStatus
VALIDATION_STATUS_DEGRADED: ValidationStatus
VALIDATION_STATUS_ERROR: ValidationStatus
VALIDATION_STATUS_SKIPPED: ValidationStatus

class ValidateScenarioRequest(_message.Message):
    __slots__ = ("scenario", "path", "include_execution")
    SCENARIO_FIELD_NUMBER: _ClassVar[int]
    PATH_FIELD_NUMBER: _ClassVar[int]
    INCLUDE_EXECUTION_FIELD_NUMBER: _ClassVar[int]
    scenario: str
    path: str
    include_execution: bool
    def __init__(self, scenario: _Optional[str] = ..., path: _Optional[str] = ..., include_execution: _Optional[bool] = ...) -> None: ...

class ValidateScenarioResponse(_message.Message):
    __slots__ = ("scenario", "status", "assessment", "native_detail", "metrics")
    SCENARIO_FIELD_NUMBER: _ClassVar[int]
    STATUS_FIELD_NUMBER: _ClassVar[int]
    ASSESSMENT_FIELD_NUMBER: _ClassVar[int]
    NATIVE_DETAIL_FIELD_NUMBER: _ClassVar[int]
    METRICS_FIELD_NUMBER: _ClassVar[int]
    scenario: str
    status: ValidationStatus
    assessment: _maturity_pb2.MaturityAssessment
    native_detail: _any_pb2.Any
    metrics: _metrics_pb2.ExecutionMetrics
    def __init__(self, scenario: _Optional[str] = ..., status: _Optional[_Union[ValidationStatus, str]] = ..., assessment: _Optional[_Union[_maturity_pb2.MaturityAssessment, _Mapping]] = ..., native_detail: _Optional[_Union[_any_pb2.Any, _Mapping]] = ..., metrics: _Optional[_Union[_metrics_pb2.ExecutionMetrics, _Mapping]] = ...) -> None: ...

class FixRequest(_message.Message):
    __slots__ = ("scenario", "path", "rule_ids")
    SCENARIO_FIELD_NUMBER: _ClassVar[int]
    PATH_FIELD_NUMBER: _ClassVar[int]
    RULE_IDS_FIELD_NUMBER: _ClassVar[int]
    scenario: str
    path: str
    rule_ids: _containers.RepeatedScalarFieldContainer[str]
    def __init__(self, scenario: _Optional[str] = ..., path: _Optional[str] = ..., rule_ids: _Optional[_Iterable[str]] = ...) -> None: ...

class FixCandidate(_message.Message):
    __slots__ = ("rule_id", "file_path", "description", "before", "after", "applied")
    RULE_ID_FIELD_NUMBER: _ClassVar[int]
    FILE_PATH_FIELD_NUMBER: _ClassVar[int]
    DESCRIPTION_FIELD_NUMBER: _ClassVar[int]
    BEFORE_FIELD_NUMBER: _ClassVar[int]
    AFTER_FIELD_NUMBER: _ClassVar[int]
    APPLIED_FIELD_NUMBER: _ClassVar[int]
    rule_id: str
    file_path: str
    description: str
    before: str
    after: str
    applied: bool
    def __init__(self, rule_id: _Optional[str] = ..., file_path: _Optional[str] = ..., description: _Optional[str] = ..., before: _Optional[str] = ..., after: _Optional[str] = ..., applied: _Optional[bool] = ...) -> None: ...

class FixResponse(_message.Message):
    __slots__ = ("scenario", "applied", "candidates", "messages")
    SCENARIO_FIELD_NUMBER: _ClassVar[int]
    APPLIED_FIELD_NUMBER: _ClassVar[int]
    CANDIDATES_FIELD_NUMBER: _ClassVar[int]
    MESSAGES_FIELD_NUMBER: _ClassVar[int]
    scenario: str
    applied: bool
    candidates: _containers.RepeatedCompositeFieldContainer[FixCandidate]
    messages: _containers.RepeatedScalarFieldContainer[str]
    def __init__(self, scenario: _Optional[str] = ..., applied: _Optional[bool] = ..., candidates: _Optional[_Iterable[_Union[FixCandidate, _Mapping]]] = ..., messages: _Optional[_Iterable[str]] = ...) -> None: ...
