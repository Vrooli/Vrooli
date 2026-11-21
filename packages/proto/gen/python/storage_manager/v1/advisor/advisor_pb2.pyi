from google.protobuf.internal import containers as _containers
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Iterable as _Iterable, Mapping as _Mapping
from typing import ClassVar as _ClassVar, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class AnalyzeMigrationsRequest(_message.Message):
    __slots__ = ("scenarios",)
    SCENARIOS_FIELD_NUMBER: _ClassVar[int]
    scenarios: _containers.RepeatedScalarFieldContainer[str]
    def __init__(self, scenarios: _Optional[_Iterable[str]] = ...) -> None: ...

class AnalyzeMigrationsResponse(_message.Message):
    __slots__ = ("entries", "scenario_count", "with_migrations_count", "debt_count", "errors")
    ENTRIES_FIELD_NUMBER: _ClassVar[int]
    SCENARIO_COUNT_FIELD_NUMBER: _ClassVar[int]
    WITH_MIGRATIONS_COUNT_FIELD_NUMBER: _ClassVar[int]
    DEBT_COUNT_FIELD_NUMBER: _ClassVar[int]
    ERRORS_FIELD_NUMBER: _ClassVar[int]
    entries: _containers.RepeatedCompositeFieldContainer[MigrationHygiene]
    scenario_count: int
    with_migrations_count: int
    debt_count: int
    errors: _containers.RepeatedCompositeFieldContainer[AdvisorScanError]
    def __init__(self, entries: _Optional[_Iterable[_Union[MigrationHygiene, _Mapping]]] = ..., scenario_count: _Optional[int] = ..., with_migrations_count: _Optional[int] = ..., debt_count: _Optional[int] = ..., errors: _Optional[_Iterable[_Union[AdvisorScanError, _Mapping]]] = ...) -> None: ...

class MigrationHygiene(_message.Message):
    __slots__ = ("scenario", "storage_stage", "has_migrations", "has_alter_in_schema", "non_idempotent_schema", "migration_debt", "notes")
    SCENARIO_FIELD_NUMBER: _ClassVar[int]
    STORAGE_STAGE_FIELD_NUMBER: _ClassVar[int]
    HAS_MIGRATIONS_FIELD_NUMBER: _ClassVar[int]
    HAS_ALTER_IN_SCHEMA_FIELD_NUMBER: _ClassVar[int]
    NON_IDEMPOTENT_SCHEMA_FIELD_NUMBER: _ClassVar[int]
    MIGRATION_DEBT_FIELD_NUMBER: _ClassVar[int]
    NOTES_FIELD_NUMBER: _ClassVar[int]
    scenario: str
    storage_stage: str
    has_migrations: bool
    has_alter_in_schema: bool
    non_idempotent_schema: bool
    migration_debt: int
    notes: _containers.RepeatedScalarFieldContainer[str]
    def __init__(self, scenario: _Optional[str] = ..., storage_stage: _Optional[str] = ..., has_migrations: _Optional[bool] = ..., has_alter_in_schema: _Optional[bool] = ..., non_idempotent_schema: _Optional[bool] = ..., migration_debt: _Optional[int] = ..., notes: _Optional[_Iterable[str]] = ...) -> None: ...

class AdviseEnginesRequest(_message.Message):
    __slots__ = ("scenarios",)
    SCENARIOS_FIELD_NUMBER: _ClassVar[int]
    scenarios: _containers.RepeatedScalarFieldContainer[str]
    def __init__(self, scenarios: _Optional[_Iterable[str]] = ...) -> None: ...

class AdviseEnginesResponse(_message.Message):
    __slots__ = ("candidates", "scenario_count", "errors")
    CANDIDATES_FIELD_NUMBER: _ClassVar[int]
    SCENARIO_COUNT_FIELD_NUMBER: _ClassVar[int]
    ERRORS_FIELD_NUMBER: _ClassVar[int]
    candidates: _containers.RepeatedCompositeFieldContainer[EngineCandidate]
    scenario_count: int
    errors: _containers.RepeatedCompositeFieldContainer[AdvisorScanError]
    def __init__(self, candidates: _Optional[_Iterable[_Union[EngineCandidate, _Mapping]]] = ..., scenario_count: _Optional[int] = ..., errors: _Optional[_Iterable[_Union[AdvisorScanError, _Mapping]]] = ...) -> None: ...

class EngineCandidate(_message.Message):
    __slots__ = ("scenario", "current_engine", "recommended_engine", "fitness_score", "rationale", "autofixable", "blockers")
    SCENARIO_FIELD_NUMBER: _ClassVar[int]
    CURRENT_ENGINE_FIELD_NUMBER: _ClassVar[int]
    RECOMMENDED_ENGINE_FIELD_NUMBER: _ClassVar[int]
    FITNESS_SCORE_FIELD_NUMBER: _ClassVar[int]
    RATIONALE_FIELD_NUMBER: _ClassVar[int]
    AUTOFIXABLE_FIELD_NUMBER: _ClassVar[int]
    BLOCKERS_FIELD_NUMBER: _ClassVar[int]
    scenario: str
    current_engine: str
    recommended_engine: str
    fitness_score: float
    rationale: str
    autofixable: bool
    blockers: _containers.RepeatedScalarFieldContainer[str]
    def __init__(self, scenario: _Optional[str] = ..., current_engine: _Optional[str] = ..., recommended_engine: _Optional[str] = ..., fitness_score: _Optional[float] = ..., rationale: _Optional[str] = ..., autofixable: _Optional[bool] = ..., blockers: _Optional[_Iterable[str]] = ...) -> None: ...

class AdvisorScanError(_message.Message):
    __slots__ = ("scenario", "reason")
    SCENARIO_FIELD_NUMBER: _ClassVar[int]
    REASON_FIELD_NUMBER: _ClassVar[int]
    scenario: str
    reason: str
    def __init__(self, scenario: _Optional[str] = ..., reason: _Optional[str] = ...) -> None: ...
