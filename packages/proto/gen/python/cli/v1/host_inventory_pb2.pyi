from google.protobuf.internal import containers as _containers
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Iterable as _Iterable, Mapping as _Mapping
from typing import ClassVar as _ClassVar, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class HostInventoryResponse(_message.Message):
    __slots__ = ("memory", "swap", "os", "arch", "cpu", "gpus")
    MEMORY_FIELD_NUMBER: _ClassVar[int]
    SWAP_FIELD_NUMBER: _ClassVar[int]
    OS_FIELD_NUMBER: _ClassVar[int]
    ARCH_FIELD_NUMBER: _ClassVar[int]
    CPU_FIELD_NUMBER: _ClassVar[int]
    GPUS_FIELD_NUMBER: _ClassVar[int]
    memory: HostMemory
    swap: HostSwap
    os: str
    arch: str
    cpu: HostCPU
    gpus: _containers.RepeatedCompositeFieldContainer[HostGPU]
    def __init__(self, memory: _Optional[_Union[HostMemory, _Mapping]] = ..., swap: _Optional[_Union[HostSwap, _Mapping]] = ..., os: _Optional[str] = ..., arch: _Optional[str] = ..., cpu: _Optional[_Union[HostCPU, _Mapping]] = ..., gpus: _Optional[_Iterable[_Union[HostGPU, _Mapping]]] = ...) -> None: ...

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

class HostCPU(_message.Message):
    __slots__ = ("cores",)
    CORES_FIELD_NUMBER: _ClassVar[int]
    cores: int
    def __init__(self, cores: _Optional[int] = ...) -> None: ...

class HostGPU(_message.Message):
    __slots__ = ("index", "name", "vram_bytes", "vram_used_bytes", "source")
    INDEX_FIELD_NUMBER: _ClassVar[int]
    NAME_FIELD_NUMBER: _ClassVar[int]
    VRAM_BYTES_FIELD_NUMBER: _ClassVar[int]
    VRAM_USED_BYTES_FIELD_NUMBER: _ClassVar[int]
    SOURCE_FIELD_NUMBER: _ClassVar[int]
    index: int
    name: str
    vram_bytes: int
    vram_used_bytes: int
    source: str
    def __init__(self, index: _Optional[int] = ..., name: _Optional[str] = ..., vram_bytes: _Optional[int] = ..., vram_used_bytes: _Optional[int] = ..., source: _Optional[str] = ...) -> None: ...
