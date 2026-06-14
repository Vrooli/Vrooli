from proto_health.v1.shared import surface_pb2 as _surface_pb2
from google.protobuf.internal import containers as _containers
from google.protobuf.internal import enum_type_wrapper as _enum_type_wrapper
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Iterable as _Iterable, Mapping as _Mapping
from typing import ClassVar as _ClassVar, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class Severity(int, metaclass=_enum_type_wrapper.EnumTypeWrapper):
    __slots__ = ()
    SEVERITY_UNSPECIFIED: _ClassVar[Severity]
    SEVERITY_ERROR: _ClassVar[Severity]
    SEVERITY_WARNING: _ClassVar[Severity]
    SEVERITY_INFO: _ClassVar[Severity]
SEVERITY_UNSPECIFIED: Severity
SEVERITY_ERROR: Severity
SEVERITY_WARNING: Severity
SEVERITY_INFO: Severity

class Finding(_message.Message):
    __slots__ = ("severity", "code", "location", "message", "suggestion")
    SEVERITY_FIELD_NUMBER: _ClassVar[int]
    CODE_FIELD_NUMBER: _ClassVar[int]
    LOCATION_FIELD_NUMBER: _ClassVar[int]
    MESSAGE_FIELD_NUMBER: _ClassVar[int]
    SUGGESTION_FIELD_NUMBER: _ClassVar[int]
    severity: Severity
    code: str
    location: str
    message: str
    suggestion: str
    def __init__(self, severity: _Optional[_Union[Severity, str]] = ..., code: _Optional[str] = ..., location: _Optional[str] = ..., message: _Optional[str] = ..., suggestion: _Optional[str] = ...) -> None: ...

class Summary(_message.Message):
    __slots__ = ("errors", "warnings", "infos")
    ERRORS_FIELD_NUMBER: _ClassVar[int]
    WARNINGS_FIELD_NUMBER: _ClassVar[int]
    INFOS_FIELD_NUMBER: _ClassVar[int]
    errors: int
    warnings: int
    infos: int
    def __init__(self, errors: _Optional[int] = ..., warnings: _Optional[int] = ..., infos: _Optional[int] = ...) -> None: ...

class ValidateScenarioRequest(_message.Message):
    __slots__ = ("scenario",)
    SCENARIO_FIELD_NUMBER: _ClassVar[int]
    scenario: str
    def __init__(self, scenario: _Optional[str] = ...) -> None: ...

class ValidateScenarioResponse(_message.Message):
    __slots__ = ("scenario", "passed", "findings", "summary")
    SCENARIO_FIELD_NUMBER: _ClassVar[int]
    PASSED_FIELD_NUMBER: _ClassVar[int]
    FINDINGS_FIELD_NUMBER: _ClassVar[int]
    SUMMARY_FIELD_NUMBER: _ClassVar[int]
    scenario: str
    passed: bool
    findings: _containers.RepeatedCompositeFieldContainer[Finding]
    summary: Summary
    def __init__(self, scenario: _Optional[str] = ..., passed: _Optional[bool] = ..., findings: _Optional[_Iterable[_Union[Finding, _Mapping]]] = ..., summary: _Optional[_Union[Summary, _Mapping]] = ...) -> None: ...

class DescribeScenarioProtosRequest(_message.Message):
    __slots__ = ("scenario",)
    SCENARIO_FIELD_NUMBER: _ClassVar[int]
    scenario: str
    def __init__(self, scenario: _Optional[str] = ...) -> None: ...

class DescribeScenarioProtosResponse(_message.Message):
    __slots__ = ("surface",)
    SURFACE_FIELD_NUMBER: _ClassVar[int]
    surface: _surface_pb2.ProtoSurface
    def __init__(self, surface: _Optional[_Union[_surface_pb2.ProtoSurface, _Mapping]] = ...) -> None: ...

class DescribeScenariosProtosRequest(_message.Message):
    __slots__ = ("scenarios", "limit", "stability_filter")
    SCENARIOS_FIELD_NUMBER: _ClassVar[int]
    LIMIT_FIELD_NUMBER: _ClassVar[int]
    STABILITY_FILTER_FIELD_NUMBER: _ClassVar[int]
    scenarios: _containers.RepeatedScalarFieldContainer[str]
    limit: int
    stability_filter: str
    def __init__(self, scenarios: _Optional[_Iterable[str]] = ..., limit: _Optional[int] = ..., stability_filter: _Optional[str] = ...) -> None: ...

class ProtoSurfaceResult(_message.Message):
    __slots__ = ("scenario", "surface", "error")
    SCENARIO_FIELD_NUMBER: _ClassVar[int]
    SURFACE_FIELD_NUMBER: _ClassVar[int]
    ERROR_FIELD_NUMBER: _ClassVar[int]
    scenario: str
    surface: _surface_pb2.ProtoSurface
    error: str
    def __init__(self, scenario: _Optional[str] = ..., surface: _Optional[_Union[_surface_pb2.ProtoSurface, _Mapping]] = ..., error: _Optional[str] = ...) -> None: ...

class DescribeScenariosProtosResponse(_message.Message):
    __slots__ = ("results",)
    RESULTS_FIELD_NUMBER: _ClassVar[int]
    results: _containers.RepeatedCompositeFieldContainer[ProtoSurfaceResult]
    def __init__(self, results: _Optional[_Iterable[_Union[ProtoSurfaceResult, _Mapping]]] = ...) -> None: ...
