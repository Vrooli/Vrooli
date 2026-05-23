from google.protobuf.internal import containers as _containers
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Iterable as _Iterable, Mapping as _Mapping
from typing import ClassVar as _ClassVar, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class ReindexRequest(_message.Message):
    __slots__ = ("scenario", "dry_run")
    SCENARIO_FIELD_NUMBER: _ClassVar[int]
    DRY_RUN_FIELD_NUMBER: _ClassVar[int]
    scenario: str
    dry_run: bool
    def __init__(self, scenario: _Optional[str] = ..., dry_run: _Optional[bool] = ...) -> None: ...

class ReindexResponse(_message.Message):
    __slots__ = ("job_id", "planned_upserts", "planned_deletes", "dry_run")
    JOB_ID_FIELD_NUMBER: _ClassVar[int]
    PLANNED_UPSERTS_FIELD_NUMBER: _ClassVar[int]
    PLANNED_DELETES_FIELD_NUMBER: _ClassVar[int]
    DRY_RUN_FIELD_NUMBER: _ClassVar[int]
    job_id: str
    planned_upserts: int
    planned_deletes: int
    dry_run: bool
    def __init__(self, job_id: _Optional[str] = ..., planned_upserts: _Optional[int] = ..., planned_deletes: _Optional[int] = ..., dry_run: _Optional[bool] = ...) -> None: ...

class ReindexStatusRequest(_message.Message):
    __slots__ = ("job_id",)
    JOB_ID_FIELD_NUMBER: _ClassVar[int]
    job_id: str
    def __init__(self, job_id: _Optional[str] = ...) -> None: ...

class ReindexStatusResponse(_message.Message):
    __slots__ = ("job_id", "state", "processed", "total", "error", "warnings", "scenario_outcomes")
    JOB_ID_FIELD_NUMBER: _ClassVar[int]
    STATE_FIELD_NUMBER: _ClassVar[int]
    PROCESSED_FIELD_NUMBER: _ClassVar[int]
    TOTAL_FIELD_NUMBER: _ClassVar[int]
    ERROR_FIELD_NUMBER: _ClassVar[int]
    WARNINGS_FIELD_NUMBER: _ClassVar[int]
    SCENARIO_OUTCOMES_FIELD_NUMBER: _ClassVar[int]
    job_id: str
    state: str
    processed: int
    total: int
    error: str
    warnings: _containers.RepeatedScalarFieldContainer[str]
    scenario_outcomes: _containers.RepeatedCompositeFieldContainer[ScenarioDispatchOutcome]
    def __init__(self, job_id: _Optional[str] = ..., state: _Optional[str] = ..., processed: _Optional[int] = ..., total: _Optional[int] = ..., error: _Optional[str] = ..., warnings: _Optional[_Iterable[str]] = ..., scenario_outcomes: _Optional[_Iterable[_Union[ScenarioDispatchOutcome, _Mapping]]] = ...) -> None: ...

class ScenarioDispatchOutcome(_message.Message):
    __slots__ = ("scenario", "surfaces_found", "error", "template_id")
    SCENARIO_FIELD_NUMBER: _ClassVar[int]
    SURFACES_FOUND_FIELD_NUMBER: _ClassVar[int]
    ERROR_FIELD_NUMBER: _ClassVar[int]
    TEMPLATE_ID_FIELD_NUMBER: _ClassVar[int]
    scenario: str
    surfaces_found: int
    error: str
    template_id: str
    def __init__(self, scenario: _Optional[str] = ..., surfaces_found: _Optional[int] = ..., error: _Optional[str] = ..., template_id: _Optional[str] = ...) -> None: ...

class ReindexCancelRequest(_message.Message):
    __slots__ = ("job_id",)
    JOB_ID_FIELD_NUMBER: _ClassVar[int]
    job_id: str
    def __init__(self, job_id: _Optional[str] = ...) -> None: ...

class ReindexCancelResponse(_message.Message):
    __slots__ = ("job_id", "cancelled")
    JOB_ID_FIELD_NUMBER: _ClassVar[int]
    CANCELLED_FIELD_NUMBER: _ClassVar[int]
    job_id: str
    cancelled: bool
    def __init__(self, job_id: _Optional[str] = ..., cancelled: _Optional[bool] = ...) -> None: ...
