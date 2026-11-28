from google.protobuf.internal import containers as _containers
from google.protobuf.internal import enum_type_wrapper as _enum_type_wrapper
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Iterable as _Iterable, Mapping as _Mapping
from typing import ClassVar as _ClassVar, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class AnalysisOperationKind(int, metaclass=_enum_type_wrapper.EnumTypeWrapper):
    __slots__ = ()
    ANALYSIS_OPERATION_KIND_UNSPECIFIED: _ClassVar[AnalysisOperationKind]
    ANALYSIS_OPERATION_KIND_FLEET_ANALYSIS: _ClassVar[AnalysisOperationKind]
    ANALYSIS_OPERATION_KIND_GRAPH_REBUILD: _ClassVar[AnalysisOperationKind]

class AnalysisOperationState(int, metaclass=_enum_type_wrapper.EnumTypeWrapper):
    __slots__ = ()
    ANALYSIS_OPERATION_STATE_UNSPECIFIED: _ClassVar[AnalysisOperationState]
    ANALYSIS_OPERATION_STATE_QUEUED: _ClassVar[AnalysisOperationState]
    ANALYSIS_OPERATION_STATE_RUNNING: _ClassVar[AnalysisOperationState]
    ANALYSIS_OPERATION_STATE_SUCCEEDED: _ClassVar[AnalysisOperationState]
    ANALYSIS_OPERATION_STATE_PARTIAL: _ClassVar[AnalysisOperationState]
    ANALYSIS_OPERATION_STATE_FAILED: _ClassVar[AnalysisOperationState]
    ANALYSIS_OPERATION_STATE_INTERRUPTED: _ClassVar[AnalysisOperationState]
ANALYSIS_OPERATION_KIND_UNSPECIFIED: AnalysisOperationKind
ANALYSIS_OPERATION_KIND_FLEET_ANALYSIS: AnalysisOperationKind
ANALYSIS_OPERATION_KIND_GRAPH_REBUILD: AnalysisOperationKind
ANALYSIS_OPERATION_STATE_UNSPECIFIED: AnalysisOperationState
ANALYSIS_OPERATION_STATE_QUEUED: AnalysisOperationState
ANALYSIS_OPERATION_STATE_RUNNING: AnalysisOperationState
ANALYSIS_OPERATION_STATE_SUCCEEDED: AnalysisOperationState
ANALYSIS_OPERATION_STATE_PARTIAL: AnalysisOperationState
ANALYSIS_OPERATION_STATE_FAILED: AnalysisOperationState
ANALYSIS_OPERATION_STATE_INTERRUPTED: AnalysisOperationState

class StartOperationRequest(_message.Message):
    __slots__ = ("client_operation_id", "kind", "apply_graph")
    CLIENT_OPERATION_ID_FIELD_NUMBER: _ClassVar[int]
    KIND_FIELD_NUMBER: _ClassVar[int]
    APPLY_GRAPH_FIELD_NUMBER: _ClassVar[int]
    client_operation_id: str
    kind: AnalysisOperationKind
    apply_graph: bool
    def __init__(self, client_operation_id: _Optional[str] = ..., kind: _Optional[_Union[AnalysisOperationKind, str]] = ..., apply_graph: _Optional[bool] = ...) -> None: ...

class StartOperationResponse(_message.Message):
    __slots__ = ("operation", "reused")
    OPERATION_FIELD_NUMBER: _ClassVar[int]
    REUSED_FIELD_NUMBER: _ClassVar[int]
    operation: AnalysisOperation
    reused: bool
    def __init__(self, operation: _Optional[_Union[AnalysisOperation, _Mapping]] = ..., reused: _Optional[bool] = ...) -> None: ...

class GetOperationRequest(_message.Message):
    __slots__ = ("operation_id",)
    OPERATION_ID_FIELD_NUMBER: _ClassVar[int]
    operation_id: str
    def __init__(self, operation_id: _Optional[str] = ...) -> None: ...

class GetOperationResponse(_message.Message):
    __slots__ = ("operation",)
    OPERATION_FIELD_NUMBER: _ClassVar[int]
    operation: AnalysisOperation
    def __init__(self, operation: _Optional[_Union[AnalysisOperation, _Mapping]] = ...) -> None: ...

class WaitOperationRequest(_message.Message):
    __slots__ = ("operation_id",)
    OPERATION_ID_FIELD_NUMBER: _ClassVar[int]
    operation_id: str
    def __init__(self, operation_id: _Optional[str] = ...) -> None: ...

class WaitOperationResponse(_message.Message):
    __slots__ = ("operation",)
    OPERATION_FIELD_NUMBER: _ClassVar[int]
    operation: AnalysisOperation
    def __init__(self, operation: _Optional[_Union[AnalysisOperation, _Mapping]] = ...) -> None: ...

class AnalysisOperation(_message.Message):
    __slots__ = ("id", "kind", "apply_graph", "state", "created_at", "started_at", "completed_at", "scenarios_total", "scenarios_completed", "scenarios_failed", "error", "scenarios", "graph", "details_pruned")
    ID_FIELD_NUMBER: _ClassVar[int]
    KIND_FIELD_NUMBER: _ClassVar[int]
    APPLY_GRAPH_FIELD_NUMBER: _ClassVar[int]
    STATE_FIELD_NUMBER: _ClassVar[int]
    CREATED_AT_FIELD_NUMBER: _ClassVar[int]
    STARTED_AT_FIELD_NUMBER: _ClassVar[int]
    COMPLETED_AT_FIELD_NUMBER: _ClassVar[int]
    SCENARIOS_TOTAL_FIELD_NUMBER: _ClassVar[int]
    SCENARIOS_COMPLETED_FIELD_NUMBER: _ClassVar[int]
    SCENARIOS_FAILED_FIELD_NUMBER: _ClassVar[int]
    ERROR_FIELD_NUMBER: _ClassVar[int]
    SCENARIOS_FIELD_NUMBER: _ClassVar[int]
    GRAPH_FIELD_NUMBER: _ClassVar[int]
    DETAILS_PRUNED_FIELD_NUMBER: _ClassVar[int]
    id: str
    kind: AnalysisOperationKind
    apply_graph: bool
    state: AnalysisOperationState
    created_at: str
    started_at: str
    completed_at: str
    scenarios_total: int
    scenarios_completed: int
    scenarios_failed: int
    error: str
    scenarios: _containers.RepeatedCompositeFieldContainer[ScenarioAnalysisOutcome]
    graph: GraphRebuildOutcome
    details_pruned: bool
    def __init__(self, id: _Optional[str] = ..., kind: _Optional[_Union[AnalysisOperationKind, str]] = ..., apply_graph: _Optional[bool] = ..., state: _Optional[_Union[AnalysisOperationState, str]] = ..., created_at: _Optional[str] = ..., started_at: _Optional[str] = ..., completed_at: _Optional[str] = ..., scenarios_total: _Optional[int] = ..., scenarios_completed: _Optional[int] = ..., scenarios_failed: _Optional[int] = ..., error: _Optional[str] = ..., scenarios: _Optional[_Iterable[_Union[ScenarioAnalysisOutcome, _Mapping]]] = ..., graph: _Optional[_Union[GraphRebuildOutcome, _Mapping]] = ..., details_pruned: _Optional[bool] = ...) -> None: ...

class ScenarioAnalysisOutcome(_message.Message):
    __slots__ = ("scenario", "state", "detected_resources", "scenario_dependencies", "deployment_dependencies", "duration_ms", "error")
    SCENARIO_FIELD_NUMBER: _ClassVar[int]
    STATE_FIELD_NUMBER: _ClassVar[int]
    DETECTED_RESOURCES_FIELD_NUMBER: _ClassVar[int]
    SCENARIO_DEPENDENCIES_FIELD_NUMBER: _ClassVar[int]
    DEPLOYMENT_DEPENDENCIES_FIELD_NUMBER: _ClassVar[int]
    DURATION_MS_FIELD_NUMBER: _ClassVar[int]
    ERROR_FIELD_NUMBER: _ClassVar[int]
    scenario: str
    state: AnalysisOperationState
    detected_resources: int
    scenario_dependencies: int
    deployment_dependencies: int
    duration_ms: int
    error: str
    def __init__(self, scenario: _Optional[str] = ..., state: _Optional[_Union[AnalysisOperationState, str]] = ..., detected_resources: _Optional[int] = ..., scenario_dependencies: _Optional[int] = ..., deployment_dependencies: _Optional[int] = ..., duration_ms: _Optional[int] = ..., error: _Optional[str] = ...) -> None: ...

class GraphRebuildOutcome(_message.Message):
    __slots__ = ("edges", "scenario_edges", "resource_edges", "degraded_sources", "applied")
    EDGES_FIELD_NUMBER: _ClassVar[int]
    SCENARIO_EDGES_FIELD_NUMBER: _ClassVar[int]
    RESOURCE_EDGES_FIELD_NUMBER: _ClassVar[int]
    DEGRADED_SOURCES_FIELD_NUMBER: _ClassVar[int]
    APPLIED_FIELD_NUMBER: _ClassVar[int]
    edges: int
    scenario_edges: int
    resource_edges: int
    degraded_sources: _containers.RepeatedScalarFieldContainer[str]
    applied: bool
    def __init__(self, edges: _Optional[int] = ..., scenario_edges: _Optional[int] = ..., resource_edges: _Optional[int] = ..., degraded_sources: _Optional[_Iterable[str]] = ..., applied: _Optional[bool] = ...) -> None: ...
