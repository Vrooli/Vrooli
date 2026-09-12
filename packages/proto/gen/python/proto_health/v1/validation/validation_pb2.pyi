from proto_health.v1.shared import surface_pb2 as _surface_pb2
from google.protobuf.internal import containers as _containers
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Iterable as _Iterable, Mapping as _Mapping
from typing import ClassVar as _ClassVar, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

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
