from buf.validate import validate_pb2 as _validate_pb2
from swarm_manager.v1.domain import scenario_pb2 as _scenario_pb2
from google.protobuf.internal import containers as _containers
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Iterable as _Iterable, Mapping as _Mapping
from typing import ClassVar as _ClassVar, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class ListScenariosResponse(_message.Message):
    __slots__ = ("scenarios",)
    SCENARIOS_FIELD_NUMBER: _ClassVar[int]
    scenarios: _containers.RepeatedCompositeFieldContainer[_scenario_pb2.Scenario]
    def __init__(self, scenarios: _Optional[_Iterable[_Union[_scenario_pb2.Scenario, _Mapping]]] = ...) -> None: ...

class ScenarioResponse(_message.Message):
    __slots__ = ("scenario",)
    SCENARIO_FIELD_NUMBER: _ClassVar[int]
    scenario: _scenario_pb2.Scenario
    def __init__(self, scenario: _Optional[_Union[_scenario_pb2.Scenario, _Mapping]] = ...) -> None: ...

class UpdateScenarioMetadataRequest(_message.Message):
    __slots__ = ("is_greenfield", "recommendations_enabled")
    IS_GREENFIELD_FIELD_NUMBER: _ClassVar[int]
    RECOMMENDATIONS_ENABLED_FIELD_NUMBER: _ClassVar[int]
    is_greenfield: bool
    recommendations_enabled: bool
    def __init__(self, is_greenfield: _Optional[bool] = ..., recommendations_enabled: _Optional[bool] = ...) -> None: ...

class DeleteScenarioResponse(_message.Message):
    __slots__ = ("name", "archived", "message")
    NAME_FIELD_NUMBER: _ClassVar[int]
    ARCHIVED_FIELD_NUMBER: _ClassVar[int]
    MESSAGE_FIELD_NUMBER: _ClassVar[int]
    name: str
    archived: bool
    message: str
    def __init__(self, name: _Optional[str] = ..., archived: _Optional[bool] = ..., message: _Optional[str] = ...) -> None: ...
