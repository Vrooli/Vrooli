from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Mapping as _Mapping
from typing import ClassVar as _ClassVar, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class HostInventoryResponse(_message.Message):
    __slots__ = ("memory", "swap")
    MEMORY_FIELD_NUMBER: _ClassVar[int]
    SWAP_FIELD_NUMBER: _ClassVar[int]
    memory: HostMemory
    swap: HostSwap
    def __init__(self, memory: _Optional[_Union[HostMemory, _Mapping]] = ..., swap: _Optional[_Union[HostSwap, _Mapping]] = ...) -> None: ...

class HostMemory(_message.Message):
    __slots__ = ("total_bytes", "available_bytes", "buffers_bytes", "cached_bytes")
    TOTAL_BYTES_FIELD_NUMBER: _ClassVar[int]
    AVAILABLE_BYTES_FIELD_NUMBER: _ClassVar[int]
    BUFFERS_BYTES_FIELD_NUMBER: _ClassVar[int]
    CACHED_BYTES_FIELD_NUMBER: _ClassVar[int]
    total_bytes: int
    available_bytes: int
    buffers_bytes: int
    cached_bytes: int
    def __init__(self, total_bytes: _Optional[int] = ..., available_bytes: _Optional[int] = ..., buffers_bytes: _Optional[int] = ..., cached_bytes: _Optional[int] = ...) -> None: ...

class HostSwap(_message.Message):
    __slots__ = ("total_bytes",)
    TOTAL_BYTES_FIELD_NUMBER: _ClassVar[int]
    total_bytes: int
    def __init__(self, total_bytes: _Optional[int] = ...) -> None: ...
