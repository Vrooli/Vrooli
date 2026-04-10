from system_monitor.v1.domain import scripts_pb2 as _scripts_pb2
from google.protobuf.internal import containers as _containers
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Iterable as _Iterable, Mapping as _Mapping
from typing import ClassVar as _ClassVar, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class ListScriptsRequest(_message.Message):
    __slots__ = ()
    def __init__(self) -> None: ...

class ListScriptsResponse(_message.Message):
    __slots__ = ("scripts",)
    SCRIPTS_FIELD_NUMBER: _ClassVar[int]
    scripts: _containers.RepeatedCompositeFieldContainer[_scripts_pb2.InvestigationScript]
    def __init__(self, scripts: _Optional[_Iterable[_Union[_scripts_pb2.InvestigationScript, _Mapping]]] = ...) -> None: ...

class GetScriptRequest(_message.Message):
    __slots__ = ("id",)
    ID_FIELD_NUMBER: _ClassVar[int]
    id: str
    def __init__(self, id: _Optional[str] = ...) -> None: ...

class GetScriptResponse(_message.Message):
    __slots__ = ("script", "content")
    SCRIPT_FIELD_NUMBER: _ClassVar[int]
    CONTENT_FIELD_NUMBER: _ClassVar[int]
    script: _scripts_pb2.InvestigationScript
    content: str
    def __init__(self, script: _Optional[_Union[_scripts_pb2.InvestigationScript, _Mapping]] = ..., content: _Optional[str] = ...) -> None: ...

class ExecuteScriptRequest(_message.Message):
    __slots__ = ("id", "content")
    ID_FIELD_NUMBER: _ClassVar[int]
    CONTENT_FIELD_NUMBER: _ClassVar[int]
    id: str
    content: str
    def __init__(self, id: _Optional[str] = ..., content: _Optional[str] = ...) -> None: ...

class ExecuteScriptResponse(_message.Message):
    __slots__ = ("execution",)
    EXECUTION_FIELD_NUMBER: _ClassVar[int]
    execution: _scripts_pb2.ScriptExecution
    def __init__(self, execution: _Optional[_Union[_scripts_pb2.ScriptExecution, _Mapping]] = ...) -> None: ...
