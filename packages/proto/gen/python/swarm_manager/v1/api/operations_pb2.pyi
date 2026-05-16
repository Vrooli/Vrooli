from buf.validate import validate_pb2 as _validate_pb2
from swarm_manager.v1.domain import operations_pb2 as _operations_pb2
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Mapping as _Mapping
from typing import ClassVar as _ClassVar, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class GetOperationsBriefingRequest(_message.Message):
    __slots__ = ("window",)
    WINDOW_FIELD_NUMBER: _ClassVar[int]
    window: str
    def __init__(self, window: _Optional[str] = ...) -> None: ...

class GetOperationsBriefingResponse(_message.Message):
    __slots__ = ("briefing",)
    BRIEFING_FIELD_NUMBER: _ClassVar[int]
    briefing: _operations_pb2.OperationsBriefing
    def __init__(self, briefing: _Optional[_Union[_operations_pb2.OperationsBriefing, _Mapping]] = ...) -> None: ...
