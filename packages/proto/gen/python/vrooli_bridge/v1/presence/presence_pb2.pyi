from vrooli_bridge.v1.shared import shared_pb2 as _shared_pb2
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Mapping as _Mapping
from typing import ClassVar as _ClassVar, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class ReportHeartbeatRequest(_message.Message):
    __slots__ = ("heartbeat",)
    HEARTBEAT_FIELD_NUMBER: _ClassVar[int]
    heartbeat: _shared_pb2.Heartbeat
    def __init__(self, heartbeat: _Optional[_Union[_shared_pb2.Heartbeat, _Mapping]] = ...) -> None: ...

class ReportHeartbeatResponse(_message.Message):
    __slots__ = ("compatibility",)
    COMPATIBILITY_FIELD_NUMBER: _ClassVar[int]
    compatibility: _shared_pb2.CompatibilityStatus
    def __init__(self, compatibility: _Optional[_Union[_shared_pb2.CompatibilityStatus, str]] = ...) -> None: ...
