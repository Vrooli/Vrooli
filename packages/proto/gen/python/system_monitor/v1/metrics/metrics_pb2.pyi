import datetime

from google.api import annotations_pb2 as _annotations_pb2
from google.protobuf import timestamp_pb2 as _timestamp_pb2
from google.protobuf.internal import containers as _containers
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Iterable as _Iterable, Mapping as _Mapping
from typing import ClassVar as _ClassVar, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class MetricsResponse(_message.Message):
    __slots__ = ("cycle_id", "cpu_usage", "memory_usage", "tcp_connections", "gpu_usage", "timestamp", "cpu", "memory", "connections", "gpu", "disk", "swap_traffic", "major_faults", "fragmentation_index")
    CYCLE_ID_FIELD_NUMBER: _ClassVar[int]
    CPU_USAGE_FIELD_NUMBER: _ClassVar[int]
    MEMORY_USAGE_FIELD_NUMBER: _ClassVar[int]
    TCP_CONNECTIONS_FIELD_NUMBER: _ClassVar[int]
    GPU_USAGE_FIELD_NUMBER: _ClassVar[int]
    TIMESTAMP_FIELD_NUMBER: _ClassVar[int]
    CPU_FIELD_NUMBER: _ClassVar[int]
    MEMORY_FIELD_NUMBER: _ClassVar[int]
    CONNECTIONS_FIELD_NUMBER: _ClassVar[int]
    GPU_FIELD_NUMBER: _ClassVar[int]
    DISK_FIELD_NUMBER: _ClassVar[int]
    SWAP_TRAFFIC_FIELD_NUMBER: _ClassVar[int]
    MAJOR_FAULTS_FIELD_NUMBER: _ClassVar[int]
    FRAGMENTATION_INDEX_FIELD_NUMBER: _ClassVar[int]
    cycle_id: str
    cpu_usage: float
    memory_usage: float
    tcp_connections: int
    gpu_usage: float
    timestamp: _timestamp_pb2.Timestamp
    cpu: MetricValue
    memory: MetricValue
    connections: MetricValue
    gpu: MetricValue
    disk: MetricValue
    swap_traffic: MetricValue
    major_faults: MetricValue
    fragmentation_index: MetricValue
    def __init__(self, cycle_id: _Optional[str] = ..., cpu_usage: _Optional[float] = ..., memory_usage: _Optional[float] = ..., tcp_connections: _Optional[int] = ..., gpu_usage: _Optional[float] = ..., timestamp: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., cpu: _Optional[_Union[MetricValue, _Mapping]] = ..., memory: _Optional[_Union[MetricValue, _Mapping]] = ..., connections: _Optional[_Union[MetricValue, _Mapping]] = ..., gpu: _Optional[_Union[MetricValue, _Mapping]] = ..., disk: _Optional[_Union[MetricValue, _Mapping]] = ..., swap_traffic: _Optional[_Union[MetricValue, _Mapping]] = ..., major_faults: _Optional[_Union[MetricValue, _Mapping]] = ..., fragmentation_index: _Optional[_Union[MetricValue, _Mapping]] = ...) -> None: ...

class MetricValue(_message.Message):
    __slots__ = ("measured", "unsupported_reason", "failed_error", "stale_reason", "not_yet_sampled_reason", "provenance", "observed_at", "cycle_id", "freshness_seconds", "units")
    MEASURED_FIELD_NUMBER: _ClassVar[int]
    UNSUPPORTED_REASON_FIELD_NUMBER: _ClassVar[int]
    FAILED_ERROR_FIELD_NUMBER: _ClassVar[int]
    STALE_REASON_FIELD_NUMBER: _ClassVar[int]
    NOT_YET_SAMPLED_REASON_FIELD_NUMBER: _ClassVar[int]
    PROVENANCE_FIELD_NUMBER: _ClassVar[int]
    OBSERVED_AT_FIELD_NUMBER: _ClassVar[int]
    CYCLE_ID_FIELD_NUMBER: _ClassVar[int]
    FRESHNESS_SECONDS_FIELD_NUMBER: _ClassVar[int]
    UNITS_FIELD_NUMBER: _ClassVar[int]
    measured: float
    unsupported_reason: str
    failed_error: str
    stale_reason: str
    not_yet_sampled_reason: str
    provenance: str
    observed_at: _timestamp_pb2.Timestamp
    cycle_id: str
    freshness_seconds: float
    units: str
    def __init__(self, measured: _Optional[float] = ..., unsupported_reason: _Optional[str] = ..., failed_error: _Optional[str] = ..., stale_reason: _Optional[str] = ..., not_yet_sampled_reason: _Optional[str] = ..., provenance: _Optional[str] = ..., observed_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., cycle_id: _Optional[str] = ..., freshness_seconds: _Optional[float] = ..., units: _Optional[str] = ...) -> None: ...

class MetricTimelineSample(_message.Message):
    __slots__ = ("cycle_id", "timestamp", "cpu_usage", "memory_usage", "tcp_connections", "gpu_usage", "swap_usage", "cpu", "memory", "connections", "gpu", "swap", "swap_traffic", "major_faults", "fragmentation_index", "cpu_context_switches_per_second", "cpu_interrupts_per_second", "cpu_normalized_load_1", "cpu_normalized_load_5", "cpu_run_queue_depth", "cpu_stall_some_avg10", "cpu_stall_full_avg10", "cpu_core_imbalance_index", "cpu_mode_iowait", "cpu_mode_steal")
    CYCLE_ID_FIELD_NUMBER: _ClassVar[int]
    TIMESTAMP_FIELD_NUMBER: _ClassVar[int]
    CPU_USAGE_FIELD_NUMBER: _ClassVar[int]
    MEMORY_USAGE_FIELD_NUMBER: _ClassVar[int]
    TCP_CONNECTIONS_FIELD_NUMBER: _ClassVar[int]
    GPU_USAGE_FIELD_NUMBER: _ClassVar[int]
    SWAP_USAGE_FIELD_NUMBER: _ClassVar[int]
    CPU_FIELD_NUMBER: _ClassVar[int]
    MEMORY_FIELD_NUMBER: _ClassVar[int]
    CONNECTIONS_FIELD_NUMBER: _ClassVar[int]
    GPU_FIELD_NUMBER: _ClassVar[int]
    SWAP_FIELD_NUMBER: _ClassVar[int]
    SWAP_TRAFFIC_FIELD_NUMBER: _ClassVar[int]
    MAJOR_FAULTS_FIELD_NUMBER: _ClassVar[int]
    FRAGMENTATION_INDEX_FIELD_NUMBER: _ClassVar[int]
    CPU_CONTEXT_SWITCHES_PER_SECOND_FIELD_NUMBER: _ClassVar[int]
    CPU_INTERRUPTS_PER_SECOND_FIELD_NUMBER: _ClassVar[int]
    CPU_NORMALIZED_LOAD_1_FIELD_NUMBER: _ClassVar[int]
    CPU_NORMALIZED_LOAD_5_FIELD_NUMBER: _ClassVar[int]
    CPU_RUN_QUEUE_DEPTH_FIELD_NUMBER: _ClassVar[int]
    CPU_STALL_SOME_AVG10_FIELD_NUMBER: _ClassVar[int]
    CPU_STALL_FULL_AVG10_FIELD_NUMBER: _ClassVar[int]
    CPU_CORE_IMBALANCE_INDEX_FIELD_NUMBER: _ClassVar[int]
    CPU_MODE_IOWAIT_FIELD_NUMBER: _ClassVar[int]
    CPU_MODE_STEAL_FIELD_NUMBER: _ClassVar[int]
    cycle_id: str
    timestamp: _timestamp_pb2.Timestamp
    cpu_usage: float
    memory_usage: float
    tcp_connections: int
    gpu_usage: float
    swap_usage: float
    cpu: MetricValue
    memory: MetricValue
    connections: MetricValue
    gpu: MetricValue
    swap: MetricValue
    swap_traffic: MetricValue
    major_faults: MetricValue
    fragmentation_index: MetricValue
    cpu_context_switches_per_second: MetricValue
    cpu_interrupts_per_second: MetricValue
    cpu_normalized_load_1: MetricValue
    cpu_normalized_load_5: MetricValue
    cpu_run_queue_depth: MetricValue
    cpu_stall_some_avg10: MetricValue
    cpu_stall_full_avg10: MetricValue
    cpu_core_imbalance_index: MetricValue
    cpu_mode_iowait: MetricValue
    cpu_mode_steal: MetricValue
    def __init__(self, cycle_id: _Optional[str] = ..., timestamp: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., cpu_usage: _Optional[float] = ..., memory_usage: _Optional[float] = ..., tcp_connections: _Optional[int] = ..., gpu_usage: _Optional[float] = ..., swap_usage: _Optional[float] = ..., cpu: _Optional[_Union[MetricValue, _Mapping]] = ..., memory: _Optional[_Union[MetricValue, _Mapping]] = ..., connections: _Optional[_Union[MetricValue, _Mapping]] = ..., gpu: _Optional[_Union[MetricValue, _Mapping]] = ..., swap: _Optional[_Union[MetricValue, _Mapping]] = ..., swap_traffic: _Optional[_Union[MetricValue, _Mapping]] = ..., major_faults: _Optional[_Union[MetricValue, _Mapping]] = ..., fragmentation_index: _Optional[_Union[MetricValue, _Mapping]] = ..., cpu_context_switches_per_second: _Optional[_Union[MetricValue, _Mapping]] = ..., cpu_interrupts_per_second: _Optional[_Union[MetricValue, _Mapping]] = ..., cpu_normalized_load_1: _Optional[_Union[MetricValue, _Mapping]] = ..., cpu_normalized_load_5: _Optional[_Union[MetricValue, _Mapping]] = ..., cpu_run_queue_depth: _Optional[_Union[MetricValue, _Mapping]] = ..., cpu_stall_some_avg10: _Optional[_Union[MetricValue, _Mapping]] = ..., cpu_stall_full_avg10: _Optional[_Union[MetricValue, _Mapping]] = ..., cpu_core_imbalance_index: _Optional[_Union[MetricValue, _Mapping]] = ..., cpu_mode_iowait: _Optional[_Union[MetricValue, _Mapping]] = ..., cpu_mode_steal: _Optional[_Union[MetricValue, _Mapping]] = ...) -> None: ...

class MetricsTimelineResponse(_message.Message):
    __slots__ = ("window_seconds", "sample_interval_seconds", "samples")
    WINDOW_SECONDS_FIELD_NUMBER: _ClassVar[int]
    SAMPLE_INTERVAL_SECONDS_FIELD_NUMBER: _ClassVar[int]
    SAMPLES_FIELD_NUMBER: _ClassVar[int]
    window_seconds: int
    sample_interval_seconds: int
    samples: _containers.RepeatedCompositeFieldContainer[MetricTimelineSample]
    def __init__(self, window_seconds: _Optional[int] = ..., sample_interval_seconds: _Optional[int] = ..., samples: _Optional[_Iterable[_Union[MetricTimelineSample, _Mapping]]] = ...) -> None: ...

class DetailedMetrics(_message.Message):
    __slots__ = ("cpu_details", "memory_details", "network_details", "gpu_details", "system_details", "timestamp")
    CPU_DETAILS_FIELD_NUMBER: _ClassVar[int]
    MEMORY_DETAILS_FIELD_NUMBER: _ClassVar[int]
    NETWORK_DETAILS_FIELD_NUMBER: _ClassVar[int]
    GPU_DETAILS_FIELD_NUMBER: _ClassVar[int]
    SYSTEM_DETAILS_FIELD_NUMBER: _ClassVar[int]
    TIMESTAMP_FIELD_NUMBER: _ClassVar[int]
    cpu_details: CPUMetrics
    memory_details: MemoryMetrics
    network_details: NetworkMetrics
    gpu_details: GPUMetrics
    system_details: SystemHealth
    timestamp: _timestamp_pb2.Timestamp
    def __init__(self, cpu_details: _Optional[_Union[CPUMetrics, _Mapping]] = ..., memory_details: _Optional[_Union[MemoryMetrics, _Mapping]] = ..., network_details: _Optional[_Union[NetworkMetrics, _Mapping]] = ..., gpu_details: _Optional[_Union[GPUMetrics, _Mapping]] = ..., system_details: _Optional[_Union[SystemHealth, _Mapping]] = ..., timestamp: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ...) -> None: ...

class CPUMetrics(_message.Message):
    __slots__ = ("usage", "top_processes", "load_average", "usage_state", "context_switches_per_second", "interrupts_per_second", "load_average_state", "normalized_load_1", "normalized_load_5", "run_queue_depth", "cpu_psi_some_avg10", "cpu_psi_full_avg10", "mode_breakdown", "per_core_utilization", "core_imbalance_index", "quota_throttling", "frequency_derate_ratio", "thermal_throttle_evidence", "fork_rate", "thermal_trip_point_celsius", "top_cpu_seconds_processes", "historical_cpu_attribution")
    class ModeBreakdownEntry(_message.Message):
        __slots__ = ("key", "value")
        KEY_FIELD_NUMBER: _ClassVar[int]
        VALUE_FIELD_NUMBER: _ClassVar[int]
        key: str
        value: MetricValue
        def __init__(self, key: _Optional[str] = ..., value: _Optional[_Union[MetricValue, _Mapping]] = ...) -> None: ...
    class PerCoreUtilizationEntry(_message.Message):
        __slots__ = ("key", "value")
        KEY_FIELD_NUMBER: _ClassVar[int]
        VALUE_FIELD_NUMBER: _ClassVar[int]
        key: str
        value: MetricValue
        def __init__(self, key: _Optional[str] = ..., value: _Optional[_Union[MetricValue, _Mapping]] = ...) -> None: ...
    USAGE_FIELD_NUMBER: _ClassVar[int]
    TOP_PROCESSES_FIELD_NUMBER: _ClassVar[int]
    LOAD_AVERAGE_FIELD_NUMBER: _ClassVar[int]
    USAGE_STATE_FIELD_NUMBER: _ClassVar[int]
    CONTEXT_SWITCHES_PER_SECOND_FIELD_NUMBER: _ClassVar[int]
    INTERRUPTS_PER_SECOND_FIELD_NUMBER: _ClassVar[int]
    LOAD_AVERAGE_STATE_FIELD_NUMBER: _ClassVar[int]
    NORMALIZED_LOAD_1_FIELD_NUMBER: _ClassVar[int]
    NORMALIZED_LOAD_5_FIELD_NUMBER: _ClassVar[int]
    RUN_QUEUE_DEPTH_FIELD_NUMBER: _ClassVar[int]
    CPU_PSI_SOME_AVG10_FIELD_NUMBER: _ClassVar[int]
    CPU_PSI_FULL_AVG10_FIELD_NUMBER: _ClassVar[int]
    MODE_BREAKDOWN_FIELD_NUMBER: _ClassVar[int]
    PER_CORE_UTILIZATION_FIELD_NUMBER: _ClassVar[int]
    CORE_IMBALANCE_INDEX_FIELD_NUMBER: _ClassVar[int]
    QUOTA_THROTTLING_FIELD_NUMBER: _ClassVar[int]
    FREQUENCY_DERATE_RATIO_FIELD_NUMBER: _ClassVar[int]
    THERMAL_THROTTLE_EVIDENCE_FIELD_NUMBER: _ClassVar[int]
    FORK_RATE_FIELD_NUMBER: _ClassVar[int]
    THERMAL_TRIP_POINT_CELSIUS_FIELD_NUMBER: _ClassVar[int]
    TOP_CPU_SECONDS_PROCESSES_FIELD_NUMBER: _ClassVar[int]
    HISTORICAL_CPU_ATTRIBUTION_FIELD_NUMBER: _ClassVar[int]
    usage: float
    top_processes: _containers.RepeatedCompositeFieldContainer[ProcessInfo]
    load_average: _containers.RepeatedScalarFieldContainer[float]
    usage_state: MetricValue
    context_switches_per_second: MetricValue
    interrupts_per_second: MetricValue
    load_average_state: MetricValue
    normalized_load_1: MetricValue
    normalized_load_5: MetricValue
    run_queue_depth: MetricValue
    cpu_psi_some_avg10: MetricValue
    cpu_psi_full_avg10: MetricValue
    mode_breakdown: _containers.MessageMap[str, MetricValue]
    per_core_utilization: _containers.MessageMap[str, MetricValue]
    core_imbalance_index: MetricValue
    quota_throttling: MetricValue
    frequency_derate_ratio: MetricValue
    thermal_throttle_evidence: MetricValue
    fork_rate: MetricValue
    thermal_trip_point_celsius: MetricValue
    top_cpu_seconds_processes: _containers.RepeatedCompositeFieldContainer[ProcessInfo]
    historical_cpu_attribution: ProcessTimelineResponse
    def __init__(self, usage: _Optional[float] = ..., top_processes: _Optional[_Iterable[_Union[ProcessInfo, _Mapping]]] = ..., load_average: _Optional[_Iterable[float]] = ..., usage_state: _Optional[_Union[MetricValue, _Mapping]] = ..., context_switches_per_second: _Optional[_Union[MetricValue, _Mapping]] = ..., interrupts_per_second: _Optional[_Union[MetricValue, _Mapping]] = ..., load_average_state: _Optional[_Union[MetricValue, _Mapping]] = ..., normalized_load_1: _Optional[_Union[MetricValue, _Mapping]] = ..., normalized_load_5: _Optional[_Union[MetricValue, _Mapping]] = ..., run_queue_depth: _Optional[_Union[MetricValue, _Mapping]] = ..., cpu_psi_some_avg10: _Optional[_Union[MetricValue, _Mapping]] = ..., cpu_psi_full_avg10: _Optional[_Union[MetricValue, _Mapping]] = ..., mode_breakdown: _Optional[_Mapping[str, MetricValue]] = ..., per_core_utilization: _Optional[_Mapping[str, MetricValue]] = ..., core_imbalance_index: _Optional[_Union[MetricValue, _Mapping]] = ..., quota_throttling: _Optional[_Union[MetricValue, _Mapping]] = ..., frequency_derate_ratio: _Optional[_Union[MetricValue, _Mapping]] = ..., thermal_throttle_evidence: _Optional[_Union[MetricValue, _Mapping]] = ..., fork_rate: _Optional[_Union[MetricValue, _Mapping]] = ..., thermal_trip_point_celsius: _Optional[_Union[MetricValue, _Mapping]] = ..., top_cpu_seconds_processes: _Optional[_Iterable[_Union[ProcessInfo, _Mapping]]] = ..., historical_cpu_attribution: _Optional[_Union[ProcessTimelineResponse, _Mapping]] = ...) -> None: ...

class MemoryMetrics(_message.Message):
    __slots__ = ("usage", "top_processes", "swap_usage", "disk_usage", "paging", "fragmentation", "top_paging_processes")
    USAGE_FIELD_NUMBER: _ClassVar[int]
    TOP_PROCESSES_FIELD_NUMBER: _ClassVar[int]
    SWAP_USAGE_FIELD_NUMBER: _ClassVar[int]
    DISK_USAGE_FIELD_NUMBER: _ClassVar[int]
    PAGING_FIELD_NUMBER: _ClassVar[int]
    FRAGMENTATION_FIELD_NUMBER: _ClassVar[int]
    TOP_PAGING_PROCESSES_FIELD_NUMBER: _ClassVar[int]
    usage: float
    top_processes: _containers.RepeatedCompositeFieldContainer[ProcessInfo]
    swap_usage: SwapInfo
    disk_usage: DiskInfo
    paging: PagingMetrics
    fragmentation: FragmentationMetrics
    top_paging_processes: _containers.RepeatedCompositeFieldContainer[ProcessInfo]
    def __init__(self, usage: _Optional[float] = ..., top_processes: _Optional[_Iterable[_Union[ProcessInfo, _Mapping]]] = ..., swap_usage: _Optional[_Union[SwapInfo, _Mapping]] = ..., disk_usage: _Optional[_Union[DiskInfo, _Mapping]] = ..., paging: _Optional[_Union[PagingMetrics, _Mapping]] = ..., fragmentation: _Optional[_Union[FragmentationMetrics, _Mapping]] = ..., top_paging_processes: _Optional[_Iterable[_Union[ProcessInfo, _Mapping]]] = ...) -> None: ...

class NetworkMetrics(_message.Message):
    __slots__ = ("tcp_states", "port_usage", "network_stats", "connection_pools")
    TCP_STATES_FIELD_NUMBER: _ClassVar[int]
    PORT_USAGE_FIELD_NUMBER: _ClassVar[int]
    NETWORK_STATS_FIELD_NUMBER: _ClassVar[int]
    CONNECTION_POOLS_FIELD_NUMBER: _ClassVar[int]
    tcp_states: TCPConnectionStates
    port_usage: PortUsageInfo
    network_stats: NetworkStatistics
    connection_pools: _containers.RepeatedCompositeFieldContainer[ConnectionPool]
    def __init__(self, tcp_states: _Optional[_Union[TCPConnectionStates, _Mapping]] = ..., port_usage: _Optional[_Union[PortUsageInfo, _Mapping]] = ..., network_stats: _Optional[_Union[NetworkStatistics, _Mapping]] = ..., connection_pools: _Optional[_Iterable[_Union[ConnectionPool, _Mapping]]] = ...) -> None: ...

class SystemHealth(_message.Message):
    __slots__ = ("file_descriptors", "service_dependencies", "certificates", "inotify_watchers", "api_process_goroutines")
    FILE_DESCRIPTORS_FIELD_NUMBER: _ClassVar[int]
    SERVICE_DEPENDENCIES_FIELD_NUMBER: _ClassVar[int]
    CERTIFICATES_FIELD_NUMBER: _ClassVar[int]
    INOTIFY_WATCHERS_FIELD_NUMBER: _ClassVar[int]
    API_PROCESS_GOROUTINES_FIELD_NUMBER: _ClassVar[int]
    file_descriptors: FileDescriptorInfo
    service_dependencies: _containers.RepeatedCompositeFieldContainer[ServiceHealth]
    certificates: _containers.RepeatedCompositeFieldContainer[CertificateInfo]
    inotify_watchers: InotifyWatcherInfo
    api_process_goroutines: int
    def __init__(self, file_descriptors: _Optional[_Union[FileDescriptorInfo, _Mapping]] = ..., service_dependencies: _Optional[_Iterable[_Union[ServiceHealth, _Mapping]]] = ..., certificates: _Optional[_Iterable[_Union[CertificateInfo, _Mapping]]] = ..., inotify_watchers: _Optional[_Union[InotifyWatcherInfo, _Mapping]] = ..., api_process_goroutines: _Optional[int] = ...) -> None: ...

class GPUMetrics(_message.Message):
    __slots__ = ("summary", "devices", "errors", "driver_version", "primary_model")
    SUMMARY_FIELD_NUMBER: _ClassVar[int]
    DEVICES_FIELD_NUMBER: _ClassVar[int]
    ERRORS_FIELD_NUMBER: _ClassVar[int]
    DRIVER_VERSION_FIELD_NUMBER: _ClassVar[int]
    PRIMARY_MODEL_FIELD_NUMBER: _ClassVar[int]
    summary: GPUSummary
    devices: _containers.RepeatedCompositeFieldContainer[GPUDeviceMetrics]
    errors: _containers.RepeatedScalarFieldContainer[str]
    driver_version: str
    primary_model: str
    def __init__(self, summary: _Optional[_Union[GPUSummary, _Mapping]] = ..., devices: _Optional[_Iterable[_Union[GPUDeviceMetrics, _Mapping]]] = ..., errors: _Optional[_Iterable[str]] = ..., driver_version: _Optional[str] = ..., primary_model: _Optional[str] = ...) -> None: ...

class GPUSummary(_message.Message):
    __slots__ = ("total_utilization_percent", "average_utilization_percent", "total_memory_mb", "used_memory_mb", "average_temperature_c", "device_count")
    TOTAL_UTILIZATION_PERCENT_FIELD_NUMBER: _ClassVar[int]
    AVERAGE_UTILIZATION_PERCENT_FIELD_NUMBER: _ClassVar[int]
    TOTAL_MEMORY_MB_FIELD_NUMBER: _ClassVar[int]
    USED_MEMORY_MB_FIELD_NUMBER: _ClassVar[int]
    AVERAGE_TEMPERATURE_C_FIELD_NUMBER: _ClassVar[int]
    DEVICE_COUNT_FIELD_NUMBER: _ClassVar[int]
    total_utilization_percent: float
    average_utilization_percent: float
    total_memory_mb: float
    used_memory_mb: float
    average_temperature_c: float
    device_count: int
    def __init__(self, total_utilization_percent: _Optional[float] = ..., average_utilization_percent: _Optional[float] = ..., total_memory_mb: _Optional[float] = ..., used_memory_mb: _Optional[float] = ..., average_temperature_c: _Optional[float] = ..., device_count: _Optional[int] = ...) -> None: ...

class GPUDeviceMetrics(_message.Message):
    __slots__ = ("index", "uuid", "name", "utilization_percent", "memory_utilization_percent", "memory_used_mb", "memory_total_mb", "temperature_c", "fan_speed_percent", "power_draw_w", "power_limit_w", "sm_clock_mhz", "memory_clock_mhz", "processes")
    INDEX_FIELD_NUMBER: _ClassVar[int]
    UUID_FIELD_NUMBER: _ClassVar[int]
    NAME_FIELD_NUMBER: _ClassVar[int]
    UTILIZATION_PERCENT_FIELD_NUMBER: _ClassVar[int]
    MEMORY_UTILIZATION_PERCENT_FIELD_NUMBER: _ClassVar[int]
    MEMORY_USED_MB_FIELD_NUMBER: _ClassVar[int]
    MEMORY_TOTAL_MB_FIELD_NUMBER: _ClassVar[int]
    TEMPERATURE_C_FIELD_NUMBER: _ClassVar[int]
    FAN_SPEED_PERCENT_FIELD_NUMBER: _ClassVar[int]
    POWER_DRAW_W_FIELD_NUMBER: _ClassVar[int]
    POWER_LIMIT_W_FIELD_NUMBER: _ClassVar[int]
    SM_CLOCK_MHZ_FIELD_NUMBER: _ClassVar[int]
    MEMORY_CLOCK_MHZ_FIELD_NUMBER: _ClassVar[int]
    PROCESSES_FIELD_NUMBER: _ClassVar[int]
    index: int
    uuid: str
    name: str
    utilization_percent: float
    memory_utilization_percent: float
    memory_used_mb: float
    memory_total_mb: float
    temperature_c: float
    fan_speed_percent: float
    power_draw_w: float
    power_limit_w: float
    sm_clock_mhz: float
    memory_clock_mhz: float
    processes: _containers.RepeatedCompositeFieldContainer[GPUProcessInfo]
    def __init__(self, index: _Optional[int] = ..., uuid: _Optional[str] = ..., name: _Optional[str] = ..., utilization_percent: _Optional[float] = ..., memory_utilization_percent: _Optional[float] = ..., memory_used_mb: _Optional[float] = ..., memory_total_mb: _Optional[float] = ..., temperature_c: _Optional[float] = ..., fan_speed_percent: _Optional[float] = ..., power_draw_w: _Optional[float] = ..., power_limit_w: _Optional[float] = ..., sm_clock_mhz: _Optional[float] = ..., memory_clock_mhz: _Optional[float] = ..., processes: _Optional[_Iterable[_Union[GPUProcessInfo, _Mapping]]] = ...) -> None: ...

class GPUProcessInfo(_message.Message):
    __slots__ = ("pid", "process_name", "memory_used_mb", "sm_utilization_percent", "gpu_instance_id")
    PID_FIELD_NUMBER: _ClassVar[int]
    PROCESS_NAME_FIELD_NUMBER: _ClassVar[int]
    MEMORY_USED_MB_FIELD_NUMBER: _ClassVar[int]
    SM_UTILIZATION_PERCENT_FIELD_NUMBER: _ClassVar[int]
    GPU_INSTANCE_ID_FIELD_NUMBER: _ClassVar[int]
    pid: int
    process_name: str
    memory_used_mb: float
    sm_utilization_percent: float
    gpu_instance_id: str
    def __init__(self, pid: _Optional[int] = ..., process_name: _Optional[str] = ..., memory_used_mb: _Optional[float] = ..., sm_utilization_percent: _Optional[float] = ..., gpu_instance_id: _Optional[str] = ...) -> None: ...

class ProcessInfo(_message.Message):
    __slots__ = ("pid", "name", "cpu_percent", "memory_mb", "connections", "threads", "file_descriptors", "status", "goroutines", "swap_kb", "major_faults_per_second", "cpu_seconds", "cpu_seconds_state")
    PID_FIELD_NUMBER: _ClassVar[int]
    NAME_FIELD_NUMBER: _ClassVar[int]
    CPU_PERCENT_FIELD_NUMBER: _ClassVar[int]
    MEMORY_MB_FIELD_NUMBER: _ClassVar[int]
    CONNECTIONS_FIELD_NUMBER: _ClassVar[int]
    THREADS_FIELD_NUMBER: _ClassVar[int]
    FILE_DESCRIPTORS_FIELD_NUMBER: _ClassVar[int]
    STATUS_FIELD_NUMBER: _ClassVar[int]
    GOROUTINES_FIELD_NUMBER: _ClassVar[int]
    SWAP_KB_FIELD_NUMBER: _ClassVar[int]
    MAJOR_FAULTS_PER_SECOND_FIELD_NUMBER: _ClassVar[int]
    CPU_SECONDS_FIELD_NUMBER: _ClassVar[int]
    CPU_SECONDS_STATE_FIELD_NUMBER: _ClassVar[int]
    pid: int
    name: str
    cpu_percent: float
    memory_mb: float
    connections: int
    threads: int
    file_descriptors: int
    status: str
    goroutines: int
    swap_kb: int
    major_faults_per_second: float
    cpu_seconds: float
    cpu_seconds_state: MetricValue
    def __init__(self, pid: _Optional[int] = ..., name: _Optional[str] = ..., cpu_percent: _Optional[float] = ..., memory_mb: _Optional[float] = ..., connections: _Optional[int] = ..., threads: _Optional[int] = ..., file_descriptors: _Optional[int] = ..., status: _Optional[str] = ..., goroutines: _Optional[int] = ..., swap_kb: _Optional[int] = ..., major_faults_per_second: _Optional[float] = ..., cpu_seconds: _Optional[float] = ..., cpu_seconds_state: _Optional[_Union[MetricValue, _Mapping]] = ...) -> None: ...

class PagingMetrics(_message.Message):
    __slots__ = ("swap_in_per_second", "swap_out_per_second", "swap_traffic_pages_per_second", "major_faults_per_second", "page_faults_per_second")
    SWAP_IN_PER_SECOND_FIELD_NUMBER: _ClassVar[int]
    SWAP_OUT_PER_SECOND_FIELD_NUMBER: _ClassVar[int]
    SWAP_TRAFFIC_PAGES_PER_SECOND_FIELD_NUMBER: _ClassVar[int]
    MAJOR_FAULTS_PER_SECOND_FIELD_NUMBER: _ClassVar[int]
    PAGE_FAULTS_PER_SECOND_FIELD_NUMBER: _ClassVar[int]
    swap_in_per_second: MetricValue
    swap_out_per_second: MetricValue
    swap_traffic_pages_per_second: MetricValue
    major_faults_per_second: MetricValue
    page_faults_per_second: MetricValue
    def __init__(self, swap_in_per_second: _Optional[_Union[MetricValue, _Mapping]] = ..., swap_out_per_second: _Optional[_Union[MetricValue, _Mapping]] = ..., swap_traffic_pages_per_second: _Optional[_Union[MetricValue, _Mapping]] = ..., major_faults_per_second: _Optional[_Union[MetricValue, _Mapping]] = ..., page_faults_per_second: _Optional[_Union[MetricValue, _Mapping]] = ...) -> None: ...

class FragmentationMetrics(_message.Message):
    __slots__ = ("max_free_order", "low_order_share", "compaction_failure_ratio", "compaction_rates", "buddyinfo")
    class CompactionRatesEntry(_message.Message):
        __slots__ = ("key", "value")
        KEY_FIELD_NUMBER: _ClassVar[int]
        VALUE_FIELD_NUMBER: _ClassVar[int]
        key: str
        value: MetricValue
        def __init__(self, key: _Optional[str] = ..., value: _Optional[_Union[MetricValue, _Mapping]] = ...) -> None: ...
    class BuddyinfoEntry(_message.Message):
        __slots__ = ("key", "value")
        KEY_FIELD_NUMBER: _ClassVar[int]
        VALUE_FIELD_NUMBER: _ClassVar[int]
        key: str
        value: str
        def __init__(self, key: _Optional[str] = ..., value: _Optional[str] = ...) -> None: ...
    MAX_FREE_ORDER_FIELD_NUMBER: _ClassVar[int]
    LOW_ORDER_SHARE_FIELD_NUMBER: _ClassVar[int]
    COMPACTION_FAILURE_RATIO_FIELD_NUMBER: _ClassVar[int]
    COMPACTION_RATES_FIELD_NUMBER: _ClassVar[int]
    BUDDYINFO_FIELD_NUMBER: _ClassVar[int]
    max_free_order: MetricValue
    low_order_share: MetricValue
    compaction_failure_ratio: MetricValue
    compaction_rates: _containers.MessageMap[str, MetricValue]
    buddyinfo: _containers.ScalarMap[str, str]
    def __init__(self, max_free_order: _Optional[_Union[MetricValue, _Mapping]] = ..., low_order_share: _Optional[_Union[MetricValue, _Mapping]] = ..., compaction_failure_ratio: _Optional[_Union[MetricValue, _Mapping]] = ..., compaction_rates: _Optional[_Mapping[str, MetricValue]] = ..., buddyinfo: _Optional[_Mapping[str, str]] = ...) -> None: ...

class TCPConnectionStates(_message.Message):
    __slots__ = ("established", "time_wait", "close_wait", "fin_wait1", "fin_wait2", "syn_sent", "syn_recv", "closing", "last_ack", "listen", "total")
    ESTABLISHED_FIELD_NUMBER: _ClassVar[int]
    TIME_WAIT_FIELD_NUMBER: _ClassVar[int]
    CLOSE_WAIT_FIELD_NUMBER: _ClassVar[int]
    FIN_WAIT1_FIELD_NUMBER: _ClassVar[int]
    FIN_WAIT2_FIELD_NUMBER: _ClassVar[int]
    SYN_SENT_FIELD_NUMBER: _ClassVar[int]
    SYN_RECV_FIELD_NUMBER: _ClassVar[int]
    CLOSING_FIELD_NUMBER: _ClassVar[int]
    LAST_ACK_FIELD_NUMBER: _ClassVar[int]
    LISTEN_FIELD_NUMBER: _ClassVar[int]
    TOTAL_FIELD_NUMBER: _ClassVar[int]
    established: int
    time_wait: int
    close_wait: int
    fin_wait1: int
    fin_wait2: int
    syn_sent: int
    syn_recv: int
    closing: int
    last_ack: int
    listen: int
    total: int
    def __init__(self, established: _Optional[int] = ..., time_wait: _Optional[int] = ..., close_wait: _Optional[int] = ..., fin_wait1: _Optional[int] = ..., fin_wait2: _Optional[int] = ..., syn_sent: _Optional[int] = ..., syn_recv: _Optional[int] = ..., closing: _Optional[int] = ..., last_ack: _Optional[int] = ..., listen: _Optional[int] = ..., total: _Optional[int] = ...) -> None: ...

class ConnectionPool(_message.Message):
    __slots__ = ("name", "active", "idle", "max_size", "waiting", "healthy", "leak_risk")
    NAME_FIELD_NUMBER: _ClassVar[int]
    ACTIVE_FIELD_NUMBER: _ClassVar[int]
    IDLE_FIELD_NUMBER: _ClassVar[int]
    MAX_SIZE_FIELD_NUMBER: _ClassVar[int]
    WAITING_FIELD_NUMBER: _ClassVar[int]
    HEALTHY_FIELD_NUMBER: _ClassVar[int]
    LEAK_RISK_FIELD_NUMBER: _ClassVar[int]
    name: str
    active: int
    idle: int
    max_size: int
    waiting: int
    healthy: bool
    leak_risk: str
    def __init__(self, name: _Optional[str] = ..., active: _Optional[int] = ..., idle: _Optional[int] = ..., max_size: _Optional[int] = ..., waiting: _Optional[int] = ..., healthy: _Optional[bool] = ..., leak_risk: _Optional[str] = ...) -> None: ...

class NetworkStatistics(_message.Message):
    __slots__ = ("bandwidth_in_mbps", "bandwidth_out_mbps", "packet_loss", "dns_success_rate", "dns_latency_ms")
    BANDWIDTH_IN_MBPS_FIELD_NUMBER: _ClassVar[int]
    BANDWIDTH_OUT_MBPS_FIELD_NUMBER: _ClassVar[int]
    PACKET_LOSS_FIELD_NUMBER: _ClassVar[int]
    DNS_SUCCESS_RATE_FIELD_NUMBER: _ClassVar[int]
    DNS_LATENCY_MS_FIELD_NUMBER: _ClassVar[int]
    bandwidth_in_mbps: float
    bandwidth_out_mbps: float
    packet_loss: float
    dns_success_rate: float
    dns_latency_ms: float
    def __init__(self, bandwidth_in_mbps: _Optional[float] = ..., bandwidth_out_mbps: _Optional[float] = ..., packet_loss: _Optional[float] = ..., dns_success_rate: _Optional[float] = ..., dns_latency_ms: _Optional[float] = ...) -> None: ...

class ServiceHealth(_message.Message):
    __slots__ = ("name", "status", "latency_ms", "last_check", "endpoint")
    NAME_FIELD_NUMBER: _ClassVar[int]
    STATUS_FIELD_NUMBER: _ClassVar[int]
    LATENCY_MS_FIELD_NUMBER: _ClassVar[int]
    LAST_CHECK_FIELD_NUMBER: _ClassVar[int]
    ENDPOINT_FIELD_NUMBER: _ClassVar[int]
    name: str
    status: str
    latency_ms: float
    last_check: _timestamp_pb2.Timestamp
    endpoint: str
    def __init__(self, name: _Optional[str] = ..., status: _Optional[str] = ..., latency_ms: _Optional[float] = ..., last_check: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., endpoint: _Optional[str] = ...) -> None: ...

class CertificateInfo(_message.Message):
    __slots__ = ("domain", "days_to_expiry", "status")
    DOMAIN_FIELD_NUMBER: _ClassVar[int]
    DAYS_TO_EXPIRY_FIELD_NUMBER: _ClassVar[int]
    STATUS_FIELD_NUMBER: _ClassVar[int]
    domain: str
    days_to_expiry: int
    status: str
    def __init__(self, domain: _Optional[str] = ..., days_to_expiry: _Optional[int] = ..., status: _Optional[str] = ...) -> None: ...

class SwapInfo(_message.Message):
    __slots__ = ("used", "total", "percent")
    USED_FIELD_NUMBER: _ClassVar[int]
    TOTAL_FIELD_NUMBER: _ClassVar[int]
    PERCENT_FIELD_NUMBER: _ClassVar[int]
    used: int
    total: int
    percent: float
    def __init__(self, used: _Optional[int] = ..., total: _Optional[int] = ..., percent: _Optional[float] = ...) -> None: ...

class DiskInfo(_message.Message):
    __slots__ = ("used", "total", "percent")
    USED_FIELD_NUMBER: _ClassVar[int]
    TOTAL_FIELD_NUMBER: _ClassVar[int]
    PERCENT_FIELD_NUMBER: _ClassVar[int]
    used: int
    total: int
    percent: float
    def __init__(self, used: _Optional[int] = ..., total: _Optional[int] = ..., percent: _Optional[float] = ...) -> None: ...

class DiskPartitionInfo(_message.Message):
    __slots__ = ("device", "mount_point", "size_bytes", "size_human", "used_bytes", "used_human", "available_bytes", "available_human", "use_percent")
    DEVICE_FIELD_NUMBER: _ClassVar[int]
    MOUNT_POINT_FIELD_NUMBER: _ClassVar[int]
    SIZE_BYTES_FIELD_NUMBER: _ClassVar[int]
    SIZE_HUMAN_FIELD_NUMBER: _ClassVar[int]
    USED_BYTES_FIELD_NUMBER: _ClassVar[int]
    USED_HUMAN_FIELD_NUMBER: _ClassVar[int]
    AVAILABLE_BYTES_FIELD_NUMBER: _ClassVar[int]
    AVAILABLE_HUMAN_FIELD_NUMBER: _ClassVar[int]
    USE_PERCENT_FIELD_NUMBER: _ClassVar[int]
    device: str
    mount_point: str
    size_bytes: int
    size_human: str
    used_bytes: int
    used_human: str
    available_bytes: int
    available_human: str
    use_percent: float
    def __init__(self, device: _Optional[str] = ..., mount_point: _Optional[str] = ..., size_bytes: _Optional[int] = ..., size_human: _Optional[str] = ..., used_bytes: _Optional[int] = ..., used_human: _Optional[str] = ..., available_bytes: _Optional[int] = ..., available_human: _Optional[str] = ..., use_percent: _Optional[float] = ...) -> None: ...

class DiskUsageEntry(_message.Message):
    __slots__ = ("path", "size_bytes", "size_human", "category")
    PATH_FIELD_NUMBER: _ClassVar[int]
    SIZE_BYTES_FIELD_NUMBER: _ClassVar[int]
    SIZE_HUMAN_FIELD_NUMBER: _ClassVar[int]
    CATEGORY_FIELD_NUMBER: _ClassVar[int]
    path: str
    size_bytes: int
    size_human: str
    category: str
    def __init__(self, path: _Optional[str] = ..., size_bytes: _Optional[int] = ..., size_human: _Optional[str] = ..., category: _Optional[str] = ...) -> None: ...

class DiskDetailResponse(_message.Message):
    __slots__ = ("partitions", "active_mount", "depth", "top_directories", "largest_files", "notes", "timestamp")
    PARTITIONS_FIELD_NUMBER: _ClassVar[int]
    ACTIVE_MOUNT_FIELD_NUMBER: _ClassVar[int]
    DEPTH_FIELD_NUMBER: _ClassVar[int]
    TOP_DIRECTORIES_FIELD_NUMBER: _ClassVar[int]
    LARGEST_FILES_FIELD_NUMBER: _ClassVar[int]
    NOTES_FIELD_NUMBER: _ClassVar[int]
    TIMESTAMP_FIELD_NUMBER: _ClassVar[int]
    partitions: _containers.RepeatedCompositeFieldContainer[DiskPartitionInfo]
    active_mount: str
    depth: int
    top_directories: _containers.RepeatedCompositeFieldContainer[DiskUsageEntry]
    largest_files: _containers.RepeatedCompositeFieldContainer[DiskUsageEntry]
    notes: _containers.RepeatedScalarFieldContainer[str]
    timestamp: _timestamp_pb2.Timestamp
    def __init__(self, partitions: _Optional[_Iterable[_Union[DiskPartitionInfo, _Mapping]]] = ..., active_mount: _Optional[str] = ..., depth: _Optional[int] = ..., top_directories: _Optional[_Iterable[_Union[DiskUsageEntry, _Mapping]]] = ..., largest_files: _Optional[_Iterable[_Union[DiskUsageEntry, _Mapping]]] = ..., notes: _Optional[_Iterable[str]] = ..., timestamp: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ...) -> None: ...

class PortUsageInfo(_message.Message):
    __slots__ = ("used", "total")
    USED_FIELD_NUMBER: _ClassVar[int]
    TOTAL_FIELD_NUMBER: _ClassVar[int]
    used: int
    total: int
    def __init__(self, used: _Optional[int] = ..., total: _Optional[int] = ...) -> None: ...

class FileDescriptorInfo(_message.Message):
    __slots__ = ("used", "max", "percent")
    USED_FIELD_NUMBER: _ClassVar[int]
    MAX_FIELD_NUMBER: _ClassVar[int]
    PERCENT_FIELD_NUMBER: _ClassVar[int]
    used: int
    max: int
    percent: float
    def __init__(self, used: _Optional[int] = ..., max: _Optional[int] = ..., percent: _Optional[float] = ...) -> None: ...

class InotifyWatcherInfo(_message.Message):
    __slots__ = ("supported", "watches_used", "watches_max", "watches_percent", "instances_used", "instances_max", "instances_percent")
    SUPPORTED_FIELD_NUMBER: _ClassVar[int]
    WATCHES_USED_FIELD_NUMBER: _ClassVar[int]
    WATCHES_MAX_FIELD_NUMBER: _ClassVar[int]
    WATCHES_PERCENT_FIELD_NUMBER: _ClassVar[int]
    INSTANCES_USED_FIELD_NUMBER: _ClassVar[int]
    INSTANCES_MAX_FIELD_NUMBER: _ClassVar[int]
    INSTANCES_PERCENT_FIELD_NUMBER: _ClassVar[int]
    supported: bool
    watches_used: int
    watches_max: int
    watches_percent: float
    instances_used: int
    instances_max: int
    instances_percent: float
    def __init__(self, supported: _Optional[bool] = ..., watches_used: _Optional[int] = ..., watches_max: _Optional[int] = ..., watches_percent: _Optional[float] = ..., instances_used: _Optional[int] = ..., instances_max: _Optional[int] = ..., instances_percent: _Optional[float] = ...) -> None: ...

class ProcessMonitorData(_message.Message):
    __slots__ = ("process_health", "resource_matrix", "timestamp")
    PROCESS_HEALTH_FIELD_NUMBER: _ClassVar[int]
    RESOURCE_MATRIX_FIELD_NUMBER: _ClassVar[int]
    TIMESTAMP_FIELD_NUMBER: _ClassVar[int]
    process_health: ProcessHealthInfo
    resource_matrix: _containers.RepeatedCompositeFieldContainer[ProcessInfo]
    timestamp: _timestamp_pb2.Timestamp
    def __init__(self, process_health: _Optional[_Union[ProcessHealthInfo, _Mapping]] = ..., resource_matrix: _Optional[_Iterable[_Union[ProcessInfo, _Mapping]]] = ..., timestamp: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ...) -> None: ...

class ProcessTimelineEntry(_message.Message):
    __slots__ = ("owner", "comm", "pid", "aggregated", "cpu_pct", "rss_kb", "sample_count", "first_seen", "last_seen", "cpu_seconds", "max_cpu_pct")
    OWNER_FIELD_NUMBER: _ClassVar[int]
    COMM_FIELD_NUMBER: _ClassVar[int]
    PID_FIELD_NUMBER: _ClassVar[int]
    AGGREGATED_FIELD_NUMBER: _ClassVar[int]
    CPU_PCT_FIELD_NUMBER: _ClassVar[int]
    RSS_KB_FIELD_NUMBER: _ClassVar[int]
    SAMPLE_COUNT_FIELD_NUMBER: _ClassVar[int]
    FIRST_SEEN_FIELD_NUMBER: _ClassVar[int]
    LAST_SEEN_FIELD_NUMBER: _ClassVar[int]
    CPU_SECONDS_FIELD_NUMBER: _ClassVar[int]
    MAX_CPU_PCT_FIELD_NUMBER: _ClassVar[int]
    owner: str
    comm: str
    pid: int
    aggregated: bool
    cpu_pct: float
    rss_kb: int
    sample_count: int
    first_seen: _timestamp_pb2.Timestamp
    last_seen: _timestamp_pb2.Timestamp
    cpu_seconds: float
    max_cpu_pct: float
    def __init__(self, owner: _Optional[str] = ..., comm: _Optional[str] = ..., pid: _Optional[int] = ..., aggregated: _Optional[bool] = ..., cpu_pct: _Optional[float] = ..., rss_kb: _Optional[int] = ..., sample_count: _Optional[int] = ..., first_seen: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., last_seen: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., cpu_seconds: _Optional[float] = ..., max_cpu_pct: _Optional[float] = ...) -> None: ...

class ProcessTimelineResponse(_message.Message):
    __slots__ = ("window_seconds", "owner", "top", "count", "entries", "covered_start", "covered_end")
    WINDOW_SECONDS_FIELD_NUMBER: _ClassVar[int]
    OWNER_FIELD_NUMBER: _ClassVar[int]
    TOP_FIELD_NUMBER: _ClassVar[int]
    COUNT_FIELD_NUMBER: _ClassVar[int]
    ENTRIES_FIELD_NUMBER: _ClassVar[int]
    COVERED_START_FIELD_NUMBER: _ClassVar[int]
    COVERED_END_FIELD_NUMBER: _ClassVar[int]
    window_seconds: int
    owner: str
    top: int
    count: int
    entries: _containers.RepeatedCompositeFieldContainer[ProcessTimelineEntry]
    covered_start: _timestamp_pb2.Timestamp
    covered_end: _timestamp_pb2.Timestamp
    def __init__(self, window_seconds: _Optional[int] = ..., owner: _Optional[str] = ..., top: _Optional[int] = ..., count: _Optional[int] = ..., entries: _Optional[_Iterable[_Union[ProcessTimelineEntry, _Mapping]]] = ..., covered_start: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., covered_end: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ...) -> None: ...

class ProcessHealthInfo(_message.Message):
    __slots__ = ("total_processes", "zombie_processes", "high_thread_count", "leak_candidates")
    TOTAL_PROCESSES_FIELD_NUMBER: _ClassVar[int]
    ZOMBIE_PROCESSES_FIELD_NUMBER: _ClassVar[int]
    HIGH_THREAD_COUNT_FIELD_NUMBER: _ClassVar[int]
    LEAK_CANDIDATES_FIELD_NUMBER: _ClassVar[int]
    total_processes: int
    zombie_processes: _containers.RepeatedCompositeFieldContainer[ProcessInfo]
    high_thread_count: _containers.RepeatedCompositeFieldContainer[ProcessInfo]
    leak_candidates: _containers.RepeatedCompositeFieldContainer[ProcessInfo]
    def __init__(self, total_processes: _Optional[int] = ..., zombie_processes: _Optional[_Iterable[_Union[ProcessInfo, _Mapping]]] = ..., high_thread_count: _Optional[_Iterable[_Union[ProcessInfo, _Mapping]]] = ..., leak_candidates: _Optional[_Iterable[_Union[ProcessInfo, _Mapping]]] = ...) -> None: ...

class InfrastructureMonitorData(_message.Message):
    __slots__ = ("database_pools", "http_client_pools", "message_queues", "storage_io", "timestamp")
    DATABASE_POOLS_FIELD_NUMBER: _ClassVar[int]
    HTTP_CLIENT_POOLS_FIELD_NUMBER: _ClassVar[int]
    MESSAGE_QUEUES_FIELD_NUMBER: _ClassVar[int]
    STORAGE_IO_FIELD_NUMBER: _ClassVar[int]
    TIMESTAMP_FIELD_NUMBER: _ClassVar[int]
    database_pools: _containers.RepeatedCompositeFieldContainer[ConnectionPool]
    http_client_pools: _containers.RepeatedCompositeFieldContainer[ConnectionPool]
    message_queues: MessageQueueInfo
    storage_io: StorageIOInfo
    timestamp: _timestamp_pb2.Timestamp
    def __init__(self, database_pools: _Optional[_Iterable[_Union[ConnectionPool, _Mapping]]] = ..., http_client_pools: _Optional[_Iterable[_Union[ConnectionPool, _Mapping]]] = ..., message_queues: _Optional[_Union[MessageQueueInfo, _Mapping]] = ..., storage_io: _Optional[_Union[StorageIOInfo, _Mapping]] = ..., timestamp: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ...) -> None: ...

class MessageQueueInfo(_message.Message):
    __slots__ = ("redis_pubsub", "background_jobs")
    REDIS_PUBSUB_FIELD_NUMBER: _ClassVar[int]
    BACKGROUND_JOBS_FIELD_NUMBER: _ClassVar[int]
    redis_pubsub: RedisPubSubInfo
    background_jobs: BackgroundJobsInfo
    def __init__(self, redis_pubsub: _Optional[_Union[RedisPubSubInfo, _Mapping]] = ..., background_jobs: _Optional[_Union[BackgroundJobsInfo, _Mapping]] = ...) -> None: ...

class RedisPubSubInfo(_message.Message):
    __slots__ = ("subscribers", "channels")
    SUBSCRIBERS_FIELD_NUMBER: _ClassVar[int]
    CHANNELS_FIELD_NUMBER: _ClassVar[int]
    subscribers: int
    channels: int
    def __init__(self, subscribers: _Optional[int] = ..., channels: _Optional[int] = ...) -> None: ...

class BackgroundJobsInfo(_message.Message):
    __slots__ = ("pending", "active", "failed")
    PENDING_FIELD_NUMBER: _ClassVar[int]
    ACTIVE_FIELD_NUMBER: _ClassVar[int]
    FAILED_FIELD_NUMBER: _ClassVar[int]
    pending: int
    active: int
    failed: int
    def __init__(self, pending: _Optional[int] = ..., active: _Optional[int] = ..., failed: _Optional[int] = ...) -> None: ...

class StorageIOInfo(_message.Message):
    __slots__ = ("disk_queue_depth", "io_wait_percent", "read_mb_per_sec", "write_mb_per_sec")
    DISK_QUEUE_DEPTH_FIELD_NUMBER: _ClassVar[int]
    IO_WAIT_PERCENT_FIELD_NUMBER: _ClassVar[int]
    READ_MB_PER_SEC_FIELD_NUMBER: _ClassVar[int]
    WRITE_MB_PER_SEC_FIELD_NUMBER: _ClassVar[int]
    disk_queue_depth: float
    io_wait_percent: float
    read_mb_per_sec: float
    write_mb_per_sec: float
    def __init__(self, disk_queue_depth: _Optional[float] = ..., io_wait_percent: _Optional[float] = ..., read_mb_per_sec: _Optional[float] = ..., write_mb_per_sec: _Optional[float] = ...) -> None: ...

class GetCurrentMetricsRequest(_message.Message):
    __slots__ = ("fresh",)
    FRESH_FIELD_NUMBER: _ClassVar[int]
    fresh: bool
    def __init__(self, fresh: _Optional[bool] = ...) -> None: ...

class GetCurrentMetricsResponse(_message.Message):
    __slots__ = ("metrics",)
    METRICS_FIELD_NUMBER: _ClassVar[int]
    metrics: MetricsResponse
    def __init__(self, metrics: _Optional[_Union[MetricsResponse, _Mapping]] = ...) -> None: ...

class GetDetailedMetricsRequest(_message.Message):
    __slots__ = ()
    def __init__(self) -> None: ...

class GetDetailedMetricsResponse(_message.Message):
    __slots__ = ("metrics",)
    METRICS_FIELD_NUMBER: _ClassVar[int]
    metrics: DetailedMetrics
    def __init__(self, metrics: _Optional[_Union[DetailedMetrics, _Mapping]] = ...) -> None: ...

class GetProcessMonitorRequest(_message.Message):
    __slots__ = ()
    def __init__(self) -> None: ...

class GetProcessMonitorResponse(_message.Message):
    __slots__ = ("data",)
    DATA_FIELD_NUMBER: _ClassVar[int]
    data: ProcessMonitorData
    def __init__(self, data: _Optional[_Union[ProcessMonitorData, _Mapping]] = ...) -> None: ...

class GetProcessTimelineRequest(_message.Message):
    __slots__ = ("window_seconds", "owner", "top", "rank")
    WINDOW_SECONDS_FIELD_NUMBER: _ClassVar[int]
    OWNER_FIELD_NUMBER: _ClassVar[int]
    TOP_FIELD_NUMBER: _ClassVar[int]
    RANK_FIELD_NUMBER: _ClassVar[int]
    window_seconds: int
    owner: str
    top: int
    rank: str
    def __init__(self, window_seconds: _Optional[int] = ..., owner: _Optional[str] = ..., top: _Optional[int] = ..., rank: _Optional[str] = ...) -> None: ...

class GetProcessTimelineResponse(_message.Message):
    __slots__ = ("timeline",)
    TIMELINE_FIELD_NUMBER: _ClassVar[int]
    timeline: ProcessTimelineResponse
    def __init__(self, timeline: _Optional[_Union[ProcessTimelineResponse, _Mapping]] = ...) -> None: ...

class GetInfrastructureMonitorRequest(_message.Message):
    __slots__ = ()
    def __init__(self) -> None: ...

class GetInfrastructureMonitorResponse(_message.Message):
    __slots__ = ("data",)
    DATA_FIELD_NUMBER: _ClassVar[int]
    data: InfrastructureMonitorData
    def __init__(self, data: _Optional[_Union[InfrastructureMonitorData, _Mapping]] = ...) -> None: ...

class GetMetricsTimelineRequest(_message.Message):
    __slots__ = ("window_seconds", "sample_interval_seconds")
    WINDOW_SECONDS_FIELD_NUMBER: _ClassVar[int]
    SAMPLE_INTERVAL_SECONDS_FIELD_NUMBER: _ClassVar[int]
    window_seconds: int
    sample_interval_seconds: int
    def __init__(self, window_seconds: _Optional[int] = ..., sample_interval_seconds: _Optional[int] = ...) -> None: ...

class GetMetricsTimelineResponse(_message.Message):
    __slots__ = ("timeline",)
    TIMELINE_FIELD_NUMBER: _ClassVar[int]
    timeline: MetricsTimelineResponse
    def __init__(self, timeline: _Optional[_Union[MetricsTimelineResponse, _Mapping]] = ...) -> None: ...

class GetDiskDetailRequest(_message.Message):
    __slots__ = ()
    def __init__(self) -> None: ...

class GetDiskDetailResponse(_message.Message):
    __slots__ = ("data",)
    DATA_FIELD_NUMBER: _ClassVar[int]
    data: DiskDetailResponse
    def __init__(self, data: _Optional[_Union[DiskDetailResponse, _Mapping]] = ...) -> None: ...
