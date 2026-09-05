from cli.v1 import scenario_list_pb2 as _scenario_list_pb2
from cli.v1 import scenario_status_pb2 as _scenario_status_pb2
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from typing import ClassVar as _ClassVar, Optional as _Optional

DESCRIPTOR: _descriptor.FileDescriptor

class ListScenariosRequest(_message.Message):
    __slots__ = ("include_ports",)
    INCLUDE_PORTS_FIELD_NUMBER: _ClassVar[int]
    include_ports: bool
    def __init__(self, include_ports: _Optional[bool] = ...) -> None: ...

class GetScenarioStatusRequest(_message.Message):
    __slots__ = ("name",)
    NAME_FIELD_NUMBER: _ClassVar[int]
    name: str
    def __init__(self, name: _Optional[str] = ...) -> None: ...

class GetScenarioLogsRequest(_message.Message):
    __slots__ = ("name", "tail_lines")
    NAME_FIELD_NUMBER: _ClassVar[int]
    TAIL_LINES_FIELD_NUMBER: _ClassVar[int]
    name: str
    tail_lines: int
    def __init__(self, name: _Optional[str] = ..., tail_lines: _Optional[int] = ...) -> None: ...

class StartScenarioRequest(_message.Message):
    __slots__ = ("name", "timeout_seconds")
    NAME_FIELD_NUMBER: _ClassVar[int]
    TIMEOUT_SECONDS_FIELD_NUMBER: _ClassVar[int]
    name: str
    timeout_seconds: int
    def __init__(self, name: _Optional[str] = ..., timeout_seconds: _Optional[int] = ...) -> None: ...

class StopScenarioRequest(_message.Message):
    __slots__ = ("name",)
    NAME_FIELD_NUMBER: _ClassVar[int]
    name: str
    def __init__(self, name: _Optional[str] = ...) -> None: ...

class RestartScenarioRequest(_message.Message):
    __slots__ = ("name", "timeout_seconds")
    NAME_FIELD_NUMBER: _ClassVar[int]
    TIMEOUT_SECONDS_FIELD_NUMBER: _ClassVar[int]
    name: str
    timeout_seconds: int
    def __init__(self, name: _Optional[str] = ..., timeout_seconds: _Optional[int] = ...) -> None: ...

class SetupScenarioRequest(_message.Message):
    __slots__ = ("name",)
    NAME_FIELD_NUMBER: _ClassVar[int]
    name: str
    def __init__(self, name: _Optional[str] = ...) -> None: ...
