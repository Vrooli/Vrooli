import datetime

from google.protobuf import timestamp_pb2 as _timestamp_pb2
from google.protobuf.internal import containers as _containers
from google.protobuf.internal import enum_type_wrapper as _enum_type_wrapper
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Iterable as _Iterable, Mapping as _Mapping
from typing import ClassVar as _ClassVar, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class Reliability(int, metaclass=_enum_type_wrapper.EnumTypeWrapper):
    __slots__ = ()
    RELIABILITY_UNSPECIFIED: _ClassVar[Reliability]
    RELIABILITY_RELIABLE: _ClassVar[Reliability]
    RELIABILITY_BEST_EFFORT: _ClassVar[Reliability]
    RELIABILITY_UNAVAILABLE: _ClassVar[Reliability]
RELIABILITY_UNSPECIFIED: Reliability
RELIABILITY_RELIABLE: Reliability
RELIABILITY_BEST_EFFORT: Reliability
RELIABILITY_UNAVAILABLE: Reliability

class ExecutionMetrics(_message.Message):
    __slots__ = ("wall_clock_ms", "started_at", "completed_at", "stages", "resources", "gauges", "environment")
    class GaugesEntry(_message.Message):
        __slots__ = ("key", "value")
        KEY_FIELD_NUMBER: _ClassVar[int]
        VALUE_FIELD_NUMBER: _ClassVar[int]
        key: str
        value: float
        def __init__(self, key: _Optional[str] = ..., value: _Optional[float] = ...) -> None: ...
    WALL_CLOCK_MS_FIELD_NUMBER: _ClassVar[int]
    STARTED_AT_FIELD_NUMBER: _ClassVar[int]
    COMPLETED_AT_FIELD_NUMBER: _ClassVar[int]
    STAGES_FIELD_NUMBER: _ClassVar[int]
    RESOURCES_FIELD_NUMBER: _ClassVar[int]
    GAUGES_FIELD_NUMBER: _ClassVar[int]
    ENVIRONMENT_FIELD_NUMBER: _ClassVar[int]
    wall_clock_ms: int
    started_at: _timestamp_pb2.Timestamp
    completed_at: _timestamp_pb2.Timestamp
    stages: _containers.RepeatedCompositeFieldContainer[Stage]
    resources: ResourceUsage
    gauges: _containers.ScalarMap[str, float]
    environment: CaptureEnvironment
    def __init__(self, wall_clock_ms: _Optional[int] = ..., started_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., completed_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., stages: _Optional[_Iterable[_Union[Stage, _Mapping]]] = ..., resources: _Optional[_Union[ResourceUsage, _Mapping]] = ..., gauges: _Optional[_Mapping[str, float]] = ..., environment: _Optional[_Union[CaptureEnvironment, _Mapping]] = ...) -> None: ...

class Stage(_message.Message):
    __slots__ = ("name", "duration_ms", "resources", "gauges", "children")
    class GaugesEntry(_message.Message):
        __slots__ = ("key", "value")
        KEY_FIELD_NUMBER: _ClassVar[int]
        VALUE_FIELD_NUMBER: _ClassVar[int]
        key: str
        value: float
        def __init__(self, key: _Optional[str] = ..., value: _Optional[float] = ...) -> None: ...
    NAME_FIELD_NUMBER: _ClassVar[int]
    DURATION_MS_FIELD_NUMBER: _ClassVar[int]
    RESOURCES_FIELD_NUMBER: _ClassVar[int]
    GAUGES_FIELD_NUMBER: _ClassVar[int]
    CHILDREN_FIELD_NUMBER: _ClassVar[int]
    name: str
    duration_ms: int
    resources: ResourceUsage
    gauges: _containers.ScalarMap[str, float]
    children: _containers.RepeatedCompositeFieldContainer[Stage]
    def __init__(self, name: _Optional[str] = ..., duration_ms: _Optional[int] = ..., resources: _Optional[_Union[ResourceUsage, _Mapping]] = ..., gauges: _Optional[_Mapping[str, float]] = ..., children: _Optional[_Iterable[_Union[Stage, _Mapping]]] = ...) -> None: ...

class ResourceUsage(_message.Message):
    __slots__ = ("cpu_user_ms", "cpu_sys_ms", "cpu", "peak_rss_bytes", "memory", "gpus", "gpu")
    CPU_USER_MS_FIELD_NUMBER: _ClassVar[int]
    CPU_SYS_MS_FIELD_NUMBER: _ClassVar[int]
    CPU_FIELD_NUMBER: _ClassVar[int]
    PEAK_RSS_BYTES_FIELD_NUMBER: _ClassVar[int]
    MEMORY_FIELD_NUMBER: _ClassVar[int]
    GPUS_FIELD_NUMBER: _ClassVar[int]
    GPU_FIELD_NUMBER: _ClassVar[int]
    cpu_user_ms: int
    cpu_sys_ms: int
    cpu: Reliability
    peak_rss_bytes: int
    memory: Reliability
    gpus: _containers.RepeatedCompositeFieldContainer[GpuUsage]
    gpu: Reliability
    def __init__(self, cpu_user_ms: _Optional[int] = ..., cpu_sys_ms: _Optional[int] = ..., cpu: _Optional[_Union[Reliability, str]] = ..., peak_rss_bytes: _Optional[int] = ..., memory: _Optional[_Union[Reliability, str]] = ..., gpus: _Optional[_Iterable[_Union[GpuUsage, _Mapping]]] = ..., gpu: _Optional[_Union[Reliability, str]] = ...) -> None: ...

class GpuUsage(_message.Message):
    __slots__ = ("index", "name", "vendor", "util_percent", "mem_used_bytes", "mem_total_bytes", "process_scoped")
    INDEX_FIELD_NUMBER: _ClassVar[int]
    NAME_FIELD_NUMBER: _ClassVar[int]
    VENDOR_FIELD_NUMBER: _ClassVar[int]
    UTIL_PERCENT_FIELD_NUMBER: _ClassVar[int]
    MEM_USED_BYTES_FIELD_NUMBER: _ClassVar[int]
    MEM_TOTAL_BYTES_FIELD_NUMBER: _ClassVar[int]
    PROCESS_SCOPED_FIELD_NUMBER: _ClassVar[int]
    index: int
    name: str
    vendor: str
    util_percent: float
    mem_used_bytes: int
    mem_total_bytes: int
    process_scoped: bool
    def __init__(self, index: _Optional[int] = ..., name: _Optional[str] = ..., vendor: _Optional[str] = ..., util_percent: _Optional[float] = ..., mem_used_bytes: _Optional[int] = ..., mem_total_bytes: _Optional[int] = ..., process_scoped: _Optional[bool] = ...) -> None: ...

class CaptureEnvironment(_message.Message):
    __slots__ = ("os", "arch", "num_cpu", "total_mem_bytes", "gpus", "runtime_version", "host_id")
    OS_FIELD_NUMBER: _ClassVar[int]
    ARCH_FIELD_NUMBER: _ClassVar[int]
    NUM_CPU_FIELD_NUMBER: _ClassVar[int]
    TOTAL_MEM_BYTES_FIELD_NUMBER: _ClassVar[int]
    GPUS_FIELD_NUMBER: _ClassVar[int]
    RUNTIME_VERSION_FIELD_NUMBER: _ClassVar[int]
    HOST_ID_FIELD_NUMBER: _ClassVar[int]
    os: str
    arch: str
    num_cpu: int
    total_mem_bytes: int
    gpus: _containers.RepeatedCompositeFieldContainer[GpuInfo]
    runtime_version: str
    host_id: str
    def __init__(self, os: _Optional[str] = ..., arch: _Optional[str] = ..., num_cpu: _Optional[int] = ..., total_mem_bytes: _Optional[int] = ..., gpus: _Optional[_Iterable[_Union[GpuInfo, _Mapping]]] = ..., runtime_version: _Optional[str] = ..., host_id: _Optional[str] = ...) -> None: ...

class GpuInfo(_message.Message):
    __slots__ = ("index", "name", "vendor", "mem_total_bytes")
    INDEX_FIELD_NUMBER: _ClassVar[int]
    NAME_FIELD_NUMBER: _ClassVar[int]
    VENDOR_FIELD_NUMBER: _ClassVar[int]
    MEM_TOTAL_BYTES_FIELD_NUMBER: _ClassVar[int]
    index: int
    name: str
    vendor: str
    mem_total_bytes: int
    def __init__(self, index: _Optional[int] = ..., name: _Optional[str] = ..., vendor: _Optional[str] = ..., mem_total_bytes: _Optional[int] = ...) -> None: ...
