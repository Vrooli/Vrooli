from search_hub.v1.registry import registry_pb2 as _registry_pb2
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Mapping as _Mapping
from typing import ClassVar as _ClassVar, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class ReindexRequest(_message.Message):
    __slots__ = ("scope", "dry_run", "control_token")
    SCOPE_FIELD_NUMBER: _ClassVar[int]
    DRY_RUN_FIELD_NUMBER: _ClassVar[int]
    CONTROL_TOKEN_FIELD_NUMBER: _ClassVar[int]
    scope: str
    dry_run: bool
    control_token: str
    def __init__(self, scope: _Optional[str] = ..., dry_run: _Optional[bool] = ..., control_token: _Optional[str] = ...) -> None: ...

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
    __slots__ = ("job_id", "control_token")
    JOB_ID_FIELD_NUMBER: _ClassVar[int]
    CONTROL_TOKEN_FIELD_NUMBER: _ClassVar[int]
    job_id: str
    control_token: str
    def __init__(self, job_id: _Optional[str] = ..., control_token: _Optional[str] = ...) -> None: ...

class ReindexStatusResponse(_message.Message):
    __slots__ = ("job_id", "state", "processed", "total", "error")
    JOB_ID_FIELD_NUMBER: _ClassVar[int]
    STATE_FIELD_NUMBER: _ClassVar[int]
    PROCESSED_FIELD_NUMBER: _ClassVar[int]
    TOTAL_FIELD_NUMBER: _ClassVar[int]
    ERROR_FIELD_NUMBER: _ClassVar[int]
    job_id: str
    state: str
    processed: int
    total: int
    error: str
    def __init__(self, job_id: _Optional[str] = ..., state: _Optional[str] = ..., processed: _Optional[int] = ..., total: _Optional[int] = ..., error: _Optional[str] = ...) -> None: ...

class ReindexCancelRequest(_message.Message):
    __slots__ = ("job_id", "control_token")
    JOB_ID_FIELD_NUMBER: _ClassVar[int]
    CONTROL_TOKEN_FIELD_NUMBER: _ClassVar[int]
    job_id: str
    control_token: str
    def __init__(self, job_id: _Optional[str] = ..., control_token: _Optional[str] = ...) -> None: ...

class ReindexCancelResponse(_message.Message):
    __slots__ = ("job_id", "cancelled")
    JOB_ID_FIELD_NUMBER: _ClassVar[int]
    CANCELLED_FIELD_NUMBER: _ClassVar[int]
    job_id: str
    cancelled: bool
    def __init__(self, job_id: _Optional[str] = ..., cancelled: _Optional[bool] = ...) -> None: ...

class WriteConfigRequest(_message.Message):
    __slots__ = ("provider_id", "tuning", "control_token", "dry_run")
    PROVIDER_ID_FIELD_NUMBER: _ClassVar[int]
    TUNING_FIELD_NUMBER: _ClassVar[int]
    CONTROL_TOKEN_FIELD_NUMBER: _ClassVar[int]
    DRY_RUN_FIELD_NUMBER: _ClassVar[int]
    provider_id: str
    tuning: _registry_pb2.Tuning
    control_token: str
    dry_run: bool
    def __init__(self, provider_id: _Optional[str] = ..., tuning: _Optional[_Union[_registry_pb2.Tuning, _Mapping]] = ..., control_token: _Optional[str] = ..., dry_run: _Optional[bool] = ...) -> None: ...

class WriteConfigResponse(_message.Message):
    __slots__ = ("written", "reindex_triggered", "reindex_job_id", "effective")
    WRITTEN_FIELD_NUMBER: _ClassVar[int]
    REINDEX_TRIGGERED_FIELD_NUMBER: _ClassVar[int]
    REINDEX_JOB_ID_FIELD_NUMBER: _ClassVar[int]
    EFFECTIVE_FIELD_NUMBER: _ClassVar[int]
    written: bool
    reindex_triggered: bool
    reindex_job_id: str
    effective: _registry_pb2.Tuning
    def __init__(self, written: _Optional[bool] = ..., reindex_triggered: _Optional[bool] = ..., reindex_job_id: _Optional[str] = ..., effective: _Optional[_Union[_registry_pb2.Tuning, _Mapping]] = ...) -> None: ...
