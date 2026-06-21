from google.protobuf.internal import containers as _containers
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Iterable as _Iterable, Mapping as _Mapping
from typing import ClassVar as _ClassVar, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class CliVersion(_message.Message):
    __slots__ = ("cli_version", "platform_version", "root")
    CLI_VERSION_FIELD_NUMBER: _ClassVar[int]
    PLATFORM_VERSION_FIELD_NUMBER: _ClassVar[int]
    ROOT_FIELD_NUMBER: _ClassVar[int]
    cli_version: str
    platform_version: str
    root: str
    def __init__(self, cli_version: _Optional[str] = ..., platform_version: _Optional[str] = ..., root: _Optional[str] = ...) -> None: ...

class CliSupervisorStatus(_message.Message):
    __slots__ = ("supervisor_id", "status", "status_reason", "host_boot_id", "host_session_id", "pid", "last_heartbeat_at", "heartbeat_deadline_at", "supervised_instance_count", "unverified_instance_count", "effective_renew_interval", "effective_lease_ttl", "effective_health_interval", "effective_max_health_concurrency", "effective_batch_size", "last_tick")
    SUPERVISOR_ID_FIELD_NUMBER: _ClassVar[int]
    STATUS_FIELD_NUMBER: _ClassVar[int]
    STATUS_REASON_FIELD_NUMBER: _ClassVar[int]
    HOST_BOOT_ID_FIELD_NUMBER: _ClassVar[int]
    HOST_SESSION_ID_FIELD_NUMBER: _ClassVar[int]
    PID_FIELD_NUMBER: _ClassVar[int]
    LAST_HEARTBEAT_AT_FIELD_NUMBER: _ClassVar[int]
    HEARTBEAT_DEADLINE_AT_FIELD_NUMBER: _ClassVar[int]
    SUPERVISED_INSTANCE_COUNT_FIELD_NUMBER: _ClassVar[int]
    UNVERIFIED_INSTANCE_COUNT_FIELD_NUMBER: _ClassVar[int]
    EFFECTIVE_RENEW_INTERVAL_FIELD_NUMBER: _ClassVar[int]
    EFFECTIVE_LEASE_TTL_FIELD_NUMBER: _ClassVar[int]
    EFFECTIVE_HEALTH_INTERVAL_FIELD_NUMBER: _ClassVar[int]
    EFFECTIVE_MAX_HEALTH_CONCURRENCY_FIELD_NUMBER: _ClassVar[int]
    EFFECTIVE_BATCH_SIZE_FIELD_NUMBER: _ClassVar[int]
    LAST_TICK_FIELD_NUMBER: _ClassVar[int]
    supervisor_id: str
    status: str
    status_reason: str
    host_boot_id: str
    host_session_id: str
    pid: int
    last_heartbeat_at: str
    heartbeat_deadline_at: str
    supervised_instance_count: int
    unverified_instance_count: int
    effective_renew_interval: int
    effective_lease_ttl: int
    effective_health_interval: int
    effective_max_health_concurrency: int
    effective_batch_size: int
    last_tick: CliSupervisorTick
    def __init__(self, supervisor_id: _Optional[str] = ..., status: _Optional[str] = ..., status_reason: _Optional[str] = ..., host_boot_id: _Optional[str] = ..., host_session_id: _Optional[str] = ..., pid: _Optional[int] = ..., last_heartbeat_at: _Optional[str] = ..., heartbeat_deadline_at: _Optional[str] = ..., supervised_instance_count: _Optional[int] = ..., unverified_instance_count: _Optional[int] = ..., effective_renew_interval: _Optional[int] = ..., effective_lease_ttl: _Optional[int] = ..., effective_health_interval: _Optional[int] = ..., effective_max_health_concurrency: _Optional[int] = ..., effective_batch_size: _Optional[int] = ..., last_tick: _Optional[_Union[CliSupervisorTick, _Mapping]] = ...) -> None: ...

class CliSupervisorTick(_message.Message):
    __slots__ = ("supervisor_id", "renewed", "expired", "unverified", "health_probe_count")
    SUPERVISOR_ID_FIELD_NUMBER: _ClassVar[int]
    RENEWED_FIELD_NUMBER: _ClassVar[int]
    EXPIRED_FIELD_NUMBER: _ClassVar[int]
    UNVERIFIED_FIELD_NUMBER: _ClassVar[int]
    HEALTH_PROBE_COUNT_FIELD_NUMBER: _ClassVar[int]
    supervisor_id: str
    renewed: int
    expired: int
    unverified: int
    health_probe_count: int
    def __init__(self, supervisor_id: _Optional[str] = ..., renewed: _Optional[int] = ..., expired: _Optional[int] = ..., unverified: _Optional[int] = ..., health_probe_count: _Optional[int] = ...) -> None: ...

class CliSupervisorServiceResult(_message.Message):
    __slots__ = ("unit_name", "unit_path", "scope", "active")
    UNIT_NAME_FIELD_NUMBER: _ClassVar[int]
    UNIT_PATH_FIELD_NUMBER: _ClassVar[int]
    SCOPE_FIELD_NUMBER: _ClassVar[int]
    ACTIVE_FIELD_NUMBER: _ClassVar[int]
    unit_name: str
    unit_path: str
    scope: str
    active: bool
    def __init__(self, unit_name: _Optional[str] = ..., unit_path: _Optional[str] = ..., scope: _Optional[str] = ..., active: _Optional[bool] = ...) -> None: ...

class CliHostSnapshot(_message.Message):
    __slots__ = ("os", "arch", "cpu", "load", "memory", "swap", "gpus", "gpu_processes", "runtime_tools", "docker_gpu", "warnings", "probe_statuses", "field_provenance")
    class RuntimeToolsEntry(_message.Message):
        __slots__ = ("key", "value")
        KEY_FIELD_NUMBER: _ClassVar[int]
        VALUE_FIELD_NUMBER: _ClassVar[int]
        key: str
        value: CliHostTool
        def __init__(self, key: _Optional[str] = ..., value: _Optional[_Union[CliHostTool, _Mapping]] = ...) -> None: ...
    class ProbeStatusesEntry(_message.Message):
        __slots__ = ("key", "value")
        KEY_FIELD_NUMBER: _ClassVar[int]
        VALUE_FIELD_NUMBER: _ClassVar[int]
        key: str
        value: str
        def __init__(self, key: _Optional[str] = ..., value: _Optional[str] = ...) -> None: ...
    class FieldProvenanceEntry(_message.Message):
        __slots__ = ("key", "value")
        KEY_FIELD_NUMBER: _ClassVar[int]
        VALUE_FIELD_NUMBER: _ClassVar[int]
        key: str
        value: CliHostProvenance
        def __init__(self, key: _Optional[str] = ..., value: _Optional[_Union[CliHostProvenance, _Mapping]] = ...) -> None: ...
    OS_FIELD_NUMBER: _ClassVar[int]
    ARCH_FIELD_NUMBER: _ClassVar[int]
    CPU_FIELD_NUMBER: _ClassVar[int]
    LOAD_FIELD_NUMBER: _ClassVar[int]
    MEMORY_FIELD_NUMBER: _ClassVar[int]
    SWAP_FIELD_NUMBER: _ClassVar[int]
    GPUS_FIELD_NUMBER: _ClassVar[int]
    GPU_PROCESSES_FIELD_NUMBER: _ClassVar[int]
    RUNTIME_TOOLS_FIELD_NUMBER: _ClassVar[int]
    DOCKER_GPU_FIELD_NUMBER: _ClassVar[int]
    WARNINGS_FIELD_NUMBER: _ClassVar[int]
    PROBE_STATUSES_FIELD_NUMBER: _ClassVar[int]
    FIELD_PROVENANCE_FIELD_NUMBER: _ClassVar[int]
    os: str
    arch: str
    cpu: CliHostCPU
    load: CliHostLoad
    memory: CliHostMemory
    swap: CliHostSwap
    gpus: _containers.RepeatedCompositeFieldContainer[CliHostGPU]
    gpu_processes: _containers.RepeatedCompositeFieldContainer[CliHostGPUProcess]
    runtime_tools: _containers.MessageMap[str, CliHostTool]
    docker_gpu: CliHostDockerGPU
    warnings: _containers.RepeatedScalarFieldContainer[str]
    probe_statuses: _containers.ScalarMap[str, str]
    field_provenance: _containers.MessageMap[str, CliHostProvenance]
    def __init__(self, os: _Optional[str] = ..., arch: _Optional[str] = ..., cpu: _Optional[_Union[CliHostCPU, _Mapping]] = ..., load: _Optional[_Union[CliHostLoad, _Mapping]] = ..., memory: _Optional[_Union[CliHostMemory, _Mapping]] = ..., swap: _Optional[_Union[CliHostSwap, _Mapping]] = ..., gpus: _Optional[_Iterable[_Union[CliHostGPU, _Mapping]]] = ..., gpu_processes: _Optional[_Iterable[_Union[CliHostGPUProcess, _Mapping]]] = ..., runtime_tools: _Optional[_Mapping[str, CliHostTool]] = ..., docker_gpu: _Optional[_Union[CliHostDockerGPU, _Mapping]] = ..., warnings: _Optional[_Iterable[str]] = ..., probe_statuses: _Optional[_Mapping[str, str]] = ..., field_provenance: _Optional[_Mapping[str, CliHostProvenance]] = ...) -> None: ...

class CliHostCPU(_message.Message):
    __slots__ = ("cores",)
    CORES_FIELD_NUMBER: _ClassVar[int]
    cores: int
    def __init__(self, cores: _Optional[int] = ...) -> None: ...

class CliHostLoad(_message.Message):
    __slots__ = ("load1", "load5", "load15", "running_procs", "total_procs", "last_pid", "normalized_load1", "normalized_load5")
    LOAD1_FIELD_NUMBER: _ClassVar[int]
    LOAD5_FIELD_NUMBER: _ClassVar[int]
    LOAD15_FIELD_NUMBER: _ClassVar[int]
    RUNNING_PROCS_FIELD_NUMBER: _ClassVar[int]
    TOTAL_PROCS_FIELD_NUMBER: _ClassVar[int]
    LAST_PID_FIELD_NUMBER: _ClassVar[int]
    NORMALIZED_LOAD1_FIELD_NUMBER: _ClassVar[int]
    NORMALIZED_LOAD5_FIELD_NUMBER: _ClassVar[int]
    load1: float
    load5: float
    load15: float
    running_procs: int
    total_procs: int
    last_pid: int
    normalized_load1: float
    normalized_load5: float
    def __init__(self, load1: _Optional[float] = ..., load5: _Optional[float] = ..., load15: _Optional[float] = ..., running_procs: _Optional[int] = ..., total_procs: _Optional[int] = ..., last_pid: _Optional[int] = ..., normalized_load1: _Optional[float] = ..., normalized_load5: _Optional[float] = ...) -> None: ...

class CliHostMemory(_message.Message):
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

class CliHostSwap(_message.Message):
    __slots__ = ("total_bytes", "free_bytes")
    TOTAL_BYTES_FIELD_NUMBER: _ClassVar[int]
    FREE_BYTES_FIELD_NUMBER: _ClassVar[int]
    total_bytes: int
    free_bytes: int
    def __init__(self, total_bytes: _Optional[int] = ..., free_bytes: _Optional[int] = ...) -> None: ...

class CliHostGPU(_message.Message):
    __slots__ = ("index", "uuid", "name", "driver_version", "vram_bytes", "vram_used_bytes", "utilization_percent", "memory_utilization_percent", "temperature_c", "fan_speed_percent", "power_draw_w", "power_limit_w", "sm_clock_mhz", "memory_clock_mhz", "source")
    INDEX_FIELD_NUMBER: _ClassVar[int]
    UUID_FIELD_NUMBER: _ClassVar[int]
    NAME_FIELD_NUMBER: _ClassVar[int]
    DRIVER_VERSION_FIELD_NUMBER: _ClassVar[int]
    VRAM_BYTES_FIELD_NUMBER: _ClassVar[int]
    VRAM_USED_BYTES_FIELD_NUMBER: _ClassVar[int]
    UTILIZATION_PERCENT_FIELD_NUMBER: _ClassVar[int]
    MEMORY_UTILIZATION_PERCENT_FIELD_NUMBER: _ClassVar[int]
    TEMPERATURE_C_FIELD_NUMBER: _ClassVar[int]
    FAN_SPEED_PERCENT_FIELD_NUMBER: _ClassVar[int]
    POWER_DRAW_W_FIELD_NUMBER: _ClassVar[int]
    POWER_LIMIT_W_FIELD_NUMBER: _ClassVar[int]
    SM_CLOCK_MHZ_FIELD_NUMBER: _ClassVar[int]
    MEMORY_CLOCK_MHZ_FIELD_NUMBER: _ClassVar[int]
    SOURCE_FIELD_NUMBER: _ClassVar[int]
    index: int
    uuid: str
    name: str
    driver_version: str
    vram_bytes: int
    vram_used_bytes: int
    utilization_percent: float
    memory_utilization_percent: float
    temperature_c: float
    fan_speed_percent: float
    power_draw_w: float
    power_limit_w: float
    sm_clock_mhz: float
    memory_clock_mhz: float
    source: str
    def __init__(self, index: _Optional[int] = ..., uuid: _Optional[str] = ..., name: _Optional[str] = ..., driver_version: _Optional[str] = ..., vram_bytes: _Optional[int] = ..., vram_used_bytes: _Optional[int] = ..., utilization_percent: _Optional[float] = ..., memory_utilization_percent: _Optional[float] = ..., temperature_c: _Optional[float] = ..., fan_speed_percent: _Optional[float] = ..., power_draw_w: _Optional[float] = ..., power_limit_w: _Optional[float] = ..., sm_clock_mhz: _Optional[float] = ..., memory_clock_mhz: _Optional[float] = ..., source: _Optional[str] = ...) -> None: ...

class CliHostGPUProcess(_message.Message):
    __slots__ = ("gpu_index", "gpu_uuid", "pid", "process_name", "used_bytes")
    GPU_INDEX_FIELD_NUMBER: _ClassVar[int]
    GPU_UUID_FIELD_NUMBER: _ClassVar[int]
    PID_FIELD_NUMBER: _ClassVar[int]
    PROCESS_NAME_FIELD_NUMBER: _ClassVar[int]
    USED_BYTES_FIELD_NUMBER: _ClassVar[int]
    gpu_index: int
    gpu_uuid: str
    pid: int
    process_name: str
    used_bytes: int
    def __init__(self, gpu_index: _Optional[int] = ..., gpu_uuid: _Optional[str] = ..., pid: _Optional[int] = ..., process_name: _Optional[str] = ..., used_bytes: _Optional[int] = ...) -> None: ...

class CliHostDockerGPU(_message.Message):
    __slots__ = ("nvidia_runtime",)
    NVIDIA_RUNTIME_FIELD_NUMBER: _ClassVar[int]
    nvidia_runtime: bool
    def __init__(self, nvidia_runtime: _Optional[bool] = ...) -> None: ...

class CliHostTool(_message.Message):
    __slots__ = ("present", "path")
    PRESENT_FIELD_NUMBER: _ClassVar[int]
    PATH_FIELD_NUMBER: _ClassVar[int]
    present: bool
    path: str
    def __init__(self, present: _Optional[bool] = ..., path: _Optional[str] = ...) -> None: ...

class CliHostProvenance(_message.Message):
    __slots__ = ("source_kind", "source", "observed_at", "confidence", "command", "file")
    SOURCE_KIND_FIELD_NUMBER: _ClassVar[int]
    SOURCE_FIELD_NUMBER: _ClassVar[int]
    OBSERVED_AT_FIELD_NUMBER: _ClassVar[int]
    CONFIDENCE_FIELD_NUMBER: _ClassVar[int]
    COMMAND_FIELD_NUMBER: _ClassVar[int]
    FILE_FIELD_NUMBER: _ClassVar[int]
    source_kind: str
    source: str
    observed_at: str
    confidence: str
    command: str
    file: str
    def __init__(self, source_kind: _Optional[str] = ..., source: _Optional[str] = ..., observed_at: _Optional[str] = ..., confidence: _Optional[str] = ..., command: _Optional[str] = ..., file: _Optional[str] = ...) -> None: ...

class CliHostInstallStatus(_message.Message):
    __slots__ = ("name", "command", "installed", "support_class", "execution_state", "blocking_reason", "version", "notes", "ok")
    NAME_FIELD_NUMBER: _ClassVar[int]
    COMMAND_FIELD_NUMBER: _ClassVar[int]
    INSTALLED_FIELD_NUMBER: _ClassVar[int]
    SUPPORT_CLASS_FIELD_NUMBER: _ClassVar[int]
    EXECUTION_STATE_FIELD_NUMBER: _ClassVar[int]
    BLOCKING_REASON_FIELD_NUMBER: _ClassVar[int]
    VERSION_FIELD_NUMBER: _ClassVar[int]
    NOTES_FIELD_NUMBER: _ClassVar[int]
    OK_FIELD_NUMBER: _ClassVar[int]
    name: str
    command: str
    installed: bool
    support_class: str
    execution_state: str
    blocking_reason: str
    version: str
    notes: _containers.RepeatedScalarFieldContainer[str]
    ok: bool
    def __init__(self, name: _Optional[str] = ..., command: _Optional[str] = ..., installed: _Optional[bool] = ..., support_class: _Optional[str] = ..., execution_state: _Optional[str] = ..., blocking_reason: _Optional[str] = ..., version: _Optional[str] = ..., notes: _Optional[_Iterable[str]] = ..., ok: _Optional[bool] = ...) -> None: ...
