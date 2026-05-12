from flow_verifier.v1.flows import flows_pb2 as _flows_pb2
from flow_verifier.v1.artifacts import artifacts_pb2 as _artifacts_pb2
from google.protobuf.internal import containers as _containers
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Iterable as _Iterable, Mapping as _Mapping
from typing import ClassVar as _ClassVar, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class ScenarioSummary(_message.Message):
    __slots__ = ("id", "display_name", "description", "path", "flow_count", "discovery_error")
    ID_FIELD_NUMBER: _ClassVar[int]
    DISPLAY_NAME_FIELD_NUMBER: _ClassVar[int]
    DESCRIPTION_FIELD_NUMBER: _ClassVar[int]
    PATH_FIELD_NUMBER: _ClassVar[int]
    FLOW_COUNT_FIELD_NUMBER: _ClassVar[int]
    DISCOVERY_ERROR_FIELD_NUMBER: _ClassVar[int]
    id: str
    display_name: str
    description: str
    path: str
    flow_count: int
    discovery_error: str
    def __init__(self, id: _Optional[str] = ..., display_name: _Optional[str] = ..., description: _Optional[str] = ..., path: _Optional[str] = ..., flow_count: _Optional[int] = ..., discovery_error: _Optional[str] = ...) -> None: ...

class ScenarioDetail(_message.Message):
    __slots__ = ("summary", "flows")
    SUMMARY_FIELD_NUMBER: _ClassVar[int]
    FLOWS_FIELD_NUMBER: _ClassVar[int]
    summary: ScenarioSummary
    flows: _containers.RepeatedCompositeFieldContainer[_flows_pb2.FlowSummary]
    def __init__(self, summary: _Optional[_Union[ScenarioSummary, _Mapping]] = ..., flows: _Optional[_Iterable[_Union[_flows_pb2.FlowSummary, _Mapping]]] = ...) -> None: ...

class ListScenariosRequest(_message.Message):
    __slots__ = ()
    def __init__(self) -> None: ...

class ListScenariosResponse(_message.Message):
    __slots__ = ("vrooli_root", "scenarios")
    VROOLI_ROOT_FIELD_NUMBER: _ClassVar[int]
    SCENARIOS_FIELD_NUMBER: _ClassVar[int]
    vrooli_root: str
    scenarios: _containers.RepeatedCompositeFieldContainer[ScenarioSummary]
    def __init__(self, vrooli_root: _Optional[str] = ..., scenarios: _Optional[_Iterable[_Union[ScenarioSummary, _Mapping]]] = ...) -> None: ...

class GetScenarioRequest(_message.Message):
    __slots__ = ("id",)
    ID_FIELD_NUMBER: _ClassVar[int]
    id: str
    def __init__(self, id: _Optional[str] = ...) -> None: ...

class GetScenarioResponse(_message.Message):
    __slots__ = ("scenario",)
    SCENARIO_FIELD_NUMBER: _ClassVar[int]
    scenario: ScenarioDetail
    def __init__(self, scenario: _Optional[_Union[ScenarioDetail, _Mapping]] = ...) -> None: ...

class GenerateScenarioArtifactsRequest(_message.Message):
    __slots__ = ("scenario_id",)
    SCENARIO_ID_FIELD_NUMBER: _ClassVar[int]
    scenario_id: str
    def __init__(self, scenario_id: _Optional[str] = ...) -> None: ...

class GenerateScenarioArtifactsResponse(_message.Message):
    __slots__ = ("flow_id", "report", "error_message")
    FLOW_ID_FIELD_NUMBER: _ClassVar[int]
    REPORT_FIELD_NUMBER: _ClassVar[int]
    ERROR_MESSAGE_FIELD_NUMBER: _ClassVar[int]
    flow_id: str
    report: _artifacts_pb2.ArtifactReport
    error_message: str
    def __init__(self, flow_id: _Optional[str] = ..., report: _Optional[_Union[_artifacts_pb2.ArtifactReport, _Mapping]] = ..., error_message: _Optional[str] = ...) -> None: ...

class ClearScenarioArtifactsRequest(_message.Message):
    __slots__ = ("scenario_id",)
    SCENARIO_ID_FIELD_NUMBER: _ClassVar[int]
    scenario_id: str
    def __init__(self, scenario_id: _Optional[str] = ...) -> None: ...

class ClearScenarioArtifactsResponse(_message.Message):
    __slots__ = ("flows",)
    FLOWS_FIELD_NUMBER: _ClassVar[int]
    flows: _containers.RepeatedCompositeFieldContainer[_artifacts_pb2.ClearArtifactsResponse]
    def __init__(self, flows: _Optional[_Iterable[_Union[_artifacts_pb2.ClearArtifactsResponse, _Mapping]]] = ...) -> None: ...
