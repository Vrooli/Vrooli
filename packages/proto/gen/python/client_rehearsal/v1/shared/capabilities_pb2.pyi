from google.protobuf import struct_pb2 as _struct_pb2
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Mapping as _Mapping
from typing import ClassVar as _ClassVar, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class DescribeCapabilitiesRequest(_message.Message):
    __slots__ = ()
    def __init__(self) -> None: ...

class DescribeCapabilitiesResponse(_message.Message):
    __slots__ = ("description",)
    DESCRIPTION_FIELD_NUMBER: _ClassVar[int]
    description: _struct_pb2.Struct
    def __init__(self, description: _Optional[_Union[_struct_pb2.Struct, _Mapping]] = ...) -> None: ...
