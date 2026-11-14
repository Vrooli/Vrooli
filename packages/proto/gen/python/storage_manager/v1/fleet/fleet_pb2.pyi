from google.protobuf.internal import containers as _containers
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Iterable as _Iterable, Mapping as _Mapping
from typing import ClassVar as _ClassVar, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class ScanFleetRequest(_message.Message):
    __slots__ = ("scenarios",)
    SCENARIOS_FIELD_NUMBER: _ClassVar[int]
    scenarios: _containers.RepeatedScalarFieldContainer[str]
    def __init__(self, scenarios: _Optional[_Iterable[str]] = ...) -> None: ...

class GetInventoryRequest(_message.Message):
    __slots__ = ()
    def __init__(self) -> None: ...

class ScanFleetResponse(_message.Message):
    __slots__ = ("entries", "engine_distribution", "stage_distribution", "scenario_count", "isolation_unready_count", "no_backup_count", "finding_count", "errors", "scanned_at", "data_dir_over_budget_count")
    ENTRIES_FIELD_NUMBER: _ClassVar[int]
    ENGINE_DISTRIBUTION_FIELD_NUMBER: _ClassVar[int]
    STAGE_DISTRIBUTION_FIELD_NUMBER: _ClassVar[int]
    SCENARIO_COUNT_FIELD_NUMBER: _ClassVar[int]
    ISOLATION_UNREADY_COUNT_FIELD_NUMBER: _ClassVar[int]
    NO_BACKUP_COUNT_FIELD_NUMBER: _ClassVar[int]
    FINDING_COUNT_FIELD_NUMBER: _ClassVar[int]
    ERRORS_FIELD_NUMBER: _ClassVar[int]
    SCANNED_AT_FIELD_NUMBER: _ClassVar[int]
    DATA_DIR_OVER_BUDGET_COUNT_FIELD_NUMBER: _ClassVar[int]
    entries: _containers.RepeatedCompositeFieldContainer[FleetScenarioEntry]
    engine_distribution: _containers.RepeatedCompositeFieldContainer[EngineCount]
    stage_distribution: _containers.RepeatedCompositeFieldContainer[StageCount]
    scenario_count: int
    isolation_unready_count: int
    no_backup_count: int
    finding_count: int
    errors: _containers.RepeatedCompositeFieldContainer[FleetScanError]
    scanned_at: str
    data_dir_over_budget_count: int
    def __init__(self, entries: _Optional[_Iterable[_Union[FleetScenarioEntry, _Mapping]]] = ..., engine_distribution: _Optional[_Iterable[_Union[EngineCount, _Mapping]]] = ..., stage_distribution: _Optional[_Iterable[_Union[StageCount, _Mapping]]] = ..., scenario_count: _Optional[int] = ..., isolation_unready_count: _Optional[int] = ..., no_backup_count: _Optional[int] = ..., finding_count: _Optional[int] = ..., errors: _Optional[_Iterable[_Union[FleetScanError, _Mapping]]] = ..., scanned_at: _Optional[str] = ..., data_dir_over_budget_count: _Optional[int] = ...) -> None: ...

class FleetScenarioEntry(_message.Message):
    __slots__ = ("scenario", "engines", "primary_engine", "language", "storage_stage", "isolation_ready", "isolation_reason", "namespace_adopted", "has_backup_target", "finding_count", "error_count", "autofixable_count", "data_dir_bytes", "data_dir_budget_bytes", "data_dir_utilization", "data_dir_over_budget", "data_dir_severity", "data_dir_paths")
    SCENARIO_FIELD_NUMBER: _ClassVar[int]
    ENGINES_FIELD_NUMBER: _ClassVar[int]
    PRIMARY_ENGINE_FIELD_NUMBER: _ClassVar[int]
    LANGUAGE_FIELD_NUMBER: _ClassVar[int]
    STORAGE_STAGE_FIELD_NUMBER: _ClassVar[int]
    ISOLATION_READY_FIELD_NUMBER: _ClassVar[int]
    ISOLATION_REASON_FIELD_NUMBER: _ClassVar[int]
    NAMESPACE_ADOPTED_FIELD_NUMBER: _ClassVar[int]
    HAS_BACKUP_TARGET_FIELD_NUMBER: _ClassVar[int]
    FINDING_COUNT_FIELD_NUMBER: _ClassVar[int]
    ERROR_COUNT_FIELD_NUMBER: _ClassVar[int]
    AUTOFIXABLE_COUNT_FIELD_NUMBER: _ClassVar[int]
    DATA_DIR_BYTES_FIELD_NUMBER: _ClassVar[int]
    DATA_DIR_BUDGET_BYTES_FIELD_NUMBER: _ClassVar[int]
    DATA_DIR_UTILIZATION_FIELD_NUMBER: _ClassVar[int]
    DATA_DIR_OVER_BUDGET_FIELD_NUMBER: _ClassVar[int]
    DATA_DIR_SEVERITY_FIELD_NUMBER: _ClassVar[int]
    DATA_DIR_PATHS_FIELD_NUMBER: _ClassVar[int]
    scenario: str
    engines: _containers.RepeatedScalarFieldContainer[str]
    primary_engine: str
    language: str
    storage_stage: str
    isolation_ready: bool
    isolation_reason: str
    namespace_adopted: bool
    has_backup_target: bool
    finding_count: int
    error_count: int
    autofixable_count: int
    data_dir_bytes: int
    data_dir_budget_bytes: int
    data_dir_utilization: float
    data_dir_over_budget: bool
    data_dir_severity: str
    data_dir_paths: _containers.RepeatedScalarFieldContainer[str]
    def __init__(self, scenario: _Optional[str] = ..., engines: _Optional[_Iterable[str]] = ..., primary_engine: _Optional[str] = ..., language: _Optional[str] = ..., storage_stage: _Optional[str] = ..., isolation_ready: _Optional[bool] = ..., isolation_reason: _Optional[str] = ..., namespace_adopted: _Optional[bool] = ..., has_backup_target: _Optional[bool] = ..., finding_count: _Optional[int] = ..., error_count: _Optional[int] = ..., autofixable_count: _Optional[int] = ..., data_dir_bytes: _Optional[int] = ..., data_dir_budget_bytes: _Optional[int] = ..., data_dir_utilization: _Optional[float] = ..., data_dir_over_budget: _Optional[bool] = ..., data_dir_severity: _Optional[str] = ..., data_dir_paths: _Optional[_Iterable[str]] = ...) -> None: ...

class EngineCount(_message.Message):
    __slots__ = ("engine", "scenario_count")
    ENGINE_FIELD_NUMBER: _ClassVar[int]
    SCENARIO_COUNT_FIELD_NUMBER: _ClassVar[int]
    engine: str
    scenario_count: int
    def __init__(self, engine: _Optional[str] = ..., scenario_count: _Optional[int] = ...) -> None: ...

class StageCount(_message.Message):
    __slots__ = ("stage", "scenario_count")
    STAGE_FIELD_NUMBER: _ClassVar[int]
    SCENARIO_COUNT_FIELD_NUMBER: _ClassVar[int]
    stage: str
    scenario_count: int
    def __init__(self, stage: _Optional[str] = ..., scenario_count: _Optional[int] = ...) -> None: ...

class FleetScanError(_message.Message):
    __slots__ = ("scenario", "reason")
    SCENARIO_FIELD_NUMBER: _ClassVar[int]
    REASON_FIELD_NUMBER: _ClassVar[int]
    scenario: str
    reason: str
    def __init__(self, scenario: _Optional[str] = ..., reason: _Optional[str] = ...) -> None: ...
