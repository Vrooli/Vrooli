from agent_inbox.v1.domain import manifest_pb2 as _manifest_pb2
from agent_inbox.v1.domain import tool_pb2 as _tool_pb2
from buf.validate import validate_pb2 as _validate_pb2
from google.protobuf import struct_pb2 as _struct_pb2
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Mapping as _Mapping
from typing import ClassVar as _ClassVar, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class ListToolsRequest(_message.Message):
    __slots__ = ()
    def __init__(self) -> None: ...

class ListToolsResponse(_message.Message):
    __slots__ = ("manifest",)
    MANIFEST_FIELD_NUMBER: _ClassVar[int]
    manifest: _manifest_pb2.ToolManifest
    def __init__(self, manifest: _Optional[_Union[_manifest_pb2.ToolManifest, _Mapping]] = ...) -> None: ...

class GetToolRequest(_message.Message):
    __slots__ = ("name",)
    NAME_FIELD_NUMBER: _ClassVar[int]
    name: str
    def __init__(self, name: _Optional[str] = ...) -> None: ...

class GetToolResponse(_message.Message):
    __slots__ = ("tool",)
    TOOL_FIELD_NUMBER: _ClassVar[int]
    tool: _tool_pb2.ToolDefinition
    def __init__(self, tool: _Optional[_Union[_tool_pb2.ToolDefinition, _Mapping]] = ...) -> None: ...

class ExecuteToolRequest(_message.Message):
    __slots__ = ("tool_name", "arguments")
    TOOL_NAME_FIELD_NUMBER: _ClassVar[int]
    ARGUMENTS_FIELD_NUMBER: _ClassVar[int]
    tool_name: str
    arguments: _struct_pb2.Struct
    def __init__(self, tool_name: _Optional[str] = ..., arguments: _Optional[_Union[_struct_pb2.Struct, _Mapping]] = ...) -> None: ...

class ExecuteToolResponse(_message.Message):
    __slots__ = ("success", "result", "error", "code", "is_async", "run_id", "status")
    SUCCESS_FIELD_NUMBER: _ClassVar[int]
    RESULT_FIELD_NUMBER: _ClassVar[int]
    ERROR_FIELD_NUMBER: _ClassVar[int]
    CODE_FIELD_NUMBER: _ClassVar[int]
    IS_ASYNC_FIELD_NUMBER: _ClassVar[int]
    RUN_ID_FIELD_NUMBER: _ClassVar[int]
    STATUS_FIELD_NUMBER: _ClassVar[int]
    success: bool
    result: _struct_pb2.Struct
    error: str
    code: str
    is_async: bool
    run_id: str
    status: str
    def __init__(self, success: _Optional[bool] = ..., result: _Optional[_Union[_struct_pb2.Struct, _Mapping]] = ..., error: _Optional[str] = ..., code: _Optional[str] = ..., is_async: _Optional[bool] = ..., run_id: _Optional[str] = ..., status: _Optional[str] = ...) -> None: ...
