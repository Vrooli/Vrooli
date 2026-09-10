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
    __slots__ = ("name", "tail_lines", "step", "runtime", "lifecycle", "previous")
    NAME_FIELD_NUMBER: _ClassVar[int]
    TAIL_LINES_FIELD_NUMBER: _ClassVar[int]
    STEP_FIELD_NUMBER: _ClassVar[int]
    RUNTIME_FIELD_NUMBER: _ClassVar[int]
    LIFECYCLE_FIELD_NUMBER: _ClassVar[int]
    PREVIOUS_FIELD_NUMBER: _ClassVar[int]
    name: str
    tail_lines: int
    step: str
    runtime: bool
    lifecycle: bool
    previous: bool
    def __init__(self, name: _Optional[str] = ..., tail_lines: _Optional[int] = ..., step: _Optional[str] = ..., runtime: _Optional[bool] = ..., lifecycle: _Optional[bool] = ..., previous: _Optional[bool] = ...) -> None: ...

class StartScenarioRequest(_message.Message):
    __slots__ = ("name", "timeout_seconds", "path", "best_effort", "clean_stale", "force", "accept_credential_loss", "demand_managed")
    NAME_FIELD_NUMBER: _ClassVar[int]
    TIMEOUT_SECONDS_FIELD_NUMBER: _ClassVar[int]
    PATH_FIELD_NUMBER: _ClassVar[int]
    BEST_EFFORT_FIELD_NUMBER: _ClassVar[int]
    CLEAN_STALE_FIELD_NUMBER: _ClassVar[int]
    FORCE_FIELD_NUMBER: _ClassVar[int]
    ACCEPT_CREDENTIAL_LOSS_FIELD_NUMBER: _ClassVar[int]
    DEMAND_MANAGED_FIELD_NUMBER: _ClassVar[int]
    name: str
    timeout_seconds: int
    path: str
    best_effort: bool
    clean_stale: bool
    force: bool
    accept_credential_loss: bool
    demand_managed: bool
    def __init__(self, name: _Optional[str] = ..., timeout_seconds: _Optional[int] = ..., path: _Optional[str] = ..., best_effort: _Optional[bool] = ..., clean_stale: _Optional[bool] = ..., force: _Optional[bool] = ..., accept_credential_loss: _Optional[bool] = ..., demand_managed: _Optional[bool] = ...) -> None: ...

class StopScenarioRequest(_message.Message):
    __slots__ = ("name",)
    NAME_FIELD_NUMBER: _ClassVar[int]
    name: str
    def __init__(self, name: _Optional[str] = ...) -> None: ...

class RestartScenarioRequest(_message.Message):
    __slots__ = ("name", "timeout_seconds", "path", "best_effort", "clean_stale", "force", "accept_credential_loss", "demand_managed")
    NAME_FIELD_NUMBER: _ClassVar[int]
    TIMEOUT_SECONDS_FIELD_NUMBER: _ClassVar[int]
    PATH_FIELD_NUMBER: _ClassVar[int]
    BEST_EFFORT_FIELD_NUMBER: _ClassVar[int]
    CLEAN_STALE_FIELD_NUMBER: _ClassVar[int]
    FORCE_FIELD_NUMBER: _ClassVar[int]
    ACCEPT_CREDENTIAL_LOSS_FIELD_NUMBER: _ClassVar[int]
    DEMAND_MANAGED_FIELD_NUMBER: _ClassVar[int]
    name: str
    timeout_seconds: int
    path: str
    best_effort: bool
    clean_stale: bool
    force: bool
    accept_credential_loss: bool
    demand_managed: bool
    def __init__(self, name: _Optional[str] = ..., timeout_seconds: _Optional[int] = ..., path: _Optional[str] = ..., best_effort: _Optional[bool] = ..., clean_stale: _Optional[bool] = ..., force: _Optional[bool] = ..., accept_credential_loss: _Optional[bool] = ..., demand_managed: _Optional[bool] = ...) -> None: ...

class SetupScenarioRequest(_message.Message):
    __slots__ = ("name", "path")
    NAME_FIELD_NUMBER: _ClassVar[int]
    PATH_FIELD_NUMBER: _ClassVar[int]
    name: str
    path: str
    def __init__(self, name: _Optional[str] = ..., path: _Optional[str] = ...) -> None: ...
