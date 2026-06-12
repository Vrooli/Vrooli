from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Mapping as _Mapping
from typing import ClassVar as _ClassVar, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class StatusResponse(_message.Message):
    __slots__ = ("success", "status")
    SUCCESS_FIELD_NUMBER: _ClassVar[int]
    STATUS_FIELD_NUMBER: _ClassVar[int]
    success: bool
    status: Status
    def __init__(self, success: _Optional[bool] = ..., status: _Optional[_Union[Status, _Mapping]] = ...) -> None: ...

class Status(_message.Message):
    __slots__ = ("summary",)
    SUMMARY_FIELD_NUMBER: _ClassVar[int]
    summary: StatusSummary
    def __init__(self, summary: _Optional[_Union[StatusSummary, _Mapping]] = ...) -> None: ...

class StatusSummary(_message.Message):
    __slots__ = ("resources_total", "resources_enabled", "resources_running", "resources_healthy", "scenarios_total", "scenarios_running", "scenarios_stopped", "maintenance_tracked_processes", "maintenance_orphan_processes", "maintenance_stale_locks", "maintenance_zombie_processes")
    RESOURCES_TOTAL_FIELD_NUMBER: _ClassVar[int]
    RESOURCES_ENABLED_FIELD_NUMBER: _ClassVar[int]
    RESOURCES_RUNNING_FIELD_NUMBER: _ClassVar[int]
    RESOURCES_HEALTHY_FIELD_NUMBER: _ClassVar[int]
    SCENARIOS_TOTAL_FIELD_NUMBER: _ClassVar[int]
    SCENARIOS_RUNNING_FIELD_NUMBER: _ClassVar[int]
    SCENARIOS_STOPPED_FIELD_NUMBER: _ClassVar[int]
    MAINTENANCE_TRACKED_PROCESSES_FIELD_NUMBER: _ClassVar[int]
    MAINTENANCE_ORPHAN_PROCESSES_FIELD_NUMBER: _ClassVar[int]
    MAINTENANCE_STALE_LOCKS_FIELD_NUMBER: _ClassVar[int]
    MAINTENANCE_ZOMBIE_PROCESSES_FIELD_NUMBER: _ClassVar[int]
    resources_total: int
    resources_enabled: int
    resources_running: int
    resources_healthy: int
    scenarios_total: int
    scenarios_running: int
    scenarios_stopped: int
    maintenance_tracked_processes: int
    maintenance_orphan_processes: int
    maintenance_stale_locks: int
    maintenance_zombie_processes: int
    def __init__(self, resources_total: _Optional[int] = ..., resources_enabled: _Optional[int] = ..., resources_running: _Optional[int] = ..., resources_healthy: _Optional[int] = ..., scenarios_total: _Optional[int] = ..., scenarios_running: _Optional[int] = ..., scenarios_stopped: _Optional[int] = ..., maintenance_tracked_processes: _Optional[int] = ..., maintenance_orphan_processes: _Optional[int] = ..., maintenance_stale_locks: _Optional[int] = ..., maintenance_zombie_processes: _Optional[int] = ...) -> None: ...
