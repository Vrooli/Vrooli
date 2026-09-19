from google.protobuf.internal import containers as _containers
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Iterable as _Iterable, Mapping as _Mapping
from typing import ClassVar as _ClassVar, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class Job(_message.Message):
    __slots__ = ("id", "workspace_id", "type", "dedup_key", "state", "input_revisions", "attempts", "provider", "budget_units", "result_reference", "error_code")
    ID_FIELD_NUMBER: _ClassVar[int]
    WORKSPACE_ID_FIELD_NUMBER: _ClassVar[int]
    TYPE_FIELD_NUMBER: _ClassVar[int]
    DEDUP_KEY_FIELD_NUMBER: _ClassVar[int]
    STATE_FIELD_NUMBER: _ClassVar[int]
    INPUT_REVISIONS_FIELD_NUMBER: _ClassVar[int]
    ATTEMPTS_FIELD_NUMBER: _ClassVar[int]
    PROVIDER_FIELD_NUMBER: _ClassVar[int]
    BUDGET_UNITS_FIELD_NUMBER: _ClassVar[int]
    RESULT_REFERENCE_FIELD_NUMBER: _ClassVar[int]
    ERROR_CODE_FIELD_NUMBER: _ClassVar[int]
    id: str
    workspace_id: str
    type: str
    dedup_key: str
    state: str
    input_revisions: _containers.RepeatedScalarFieldContainer[str]
    attempts: int
    provider: str
    budget_units: int
    result_reference: str
    error_code: str
    def __init__(self, id: _Optional[str] = ..., workspace_id: _Optional[str] = ..., type: _Optional[str] = ..., dedup_key: _Optional[str] = ..., state: _Optional[str] = ..., input_revisions: _Optional[_Iterable[str]] = ..., attempts: _Optional[int] = ..., provider: _Optional[str] = ..., budget_units: _Optional[int] = ..., result_reference: _Optional[str] = ..., error_code: _Optional[str] = ...) -> None: ...

class ListJobsRequest(_message.Message):
    __slots__ = ("workspace_id",)
    WORKSPACE_ID_FIELD_NUMBER: _ClassVar[int]
    workspace_id: str
    def __init__(self, workspace_id: _Optional[str] = ...) -> None: ...

class ListJobsResponse(_message.Message):
    __slots__ = ("jobs",)
    JOBS_FIELD_NUMBER: _ClassVar[int]
    jobs: _containers.RepeatedCompositeFieldContainer[Job]
    def __init__(self, jobs: _Optional[_Iterable[_Union[Job, _Mapping]]] = ...) -> None: ...

class CreateJobRequest(_message.Message):
    __slots__ = ("workspace_id", "id", "type", "dedup_key", "input_revisions", "provider", "budget_units")
    WORKSPACE_ID_FIELD_NUMBER: _ClassVar[int]
    ID_FIELD_NUMBER: _ClassVar[int]
    TYPE_FIELD_NUMBER: _ClassVar[int]
    DEDUP_KEY_FIELD_NUMBER: _ClassVar[int]
    INPUT_REVISIONS_FIELD_NUMBER: _ClassVar[int]
    PROVIDER_FIELD_NUMBER: _ClassVar[int]
    BUDGET_UNITS_FIELD_NUMBER: _ClassVar[int]
    workspace_id: str
    id: str
    type: str
    dedup_key: str
    input_revisions: _containers.RepeatedScalarFieldContainer[str]
    provider: str
    budget_units: int
    def __init__(self, workspace_id: _Optional[str] = ..., id: _Optional[str] = ..., type: _Optional[str] = ..., dedup_key: _Optional[str] = ..., input_revisions: _Optional[_Iterable[str]] = ..., provider: _Optional[str] = ..., budget_units: _Optional[int] = ...) -> None: ...

class CreateJobResponse(_message.Message):
    __slots__ = ("job",)
    JOB_FIELD_NUMBER: _ClassVar[int]
    job: Job
    def __init__(self, job: _Optional[_Union[Job, _Mapping]] = ...) -> None: ...

class TransitionJobRequest(_message.Message):
    __slots__ = ("workspace_id", "id", "next_state", "provider", "attempts", "result_reference", "error_code")
    WORKSPACE_ID_FIELD_NUMBER: _ClassVar[int]
    ID_FIELD_NUMBER: _ClassVar[int]
    NEXT_STATE_FIELD_NUMBER: _ClassVar[int]
    PROVIDER_FIELD_NUMBER: _ClassVar[int]
    ATTEMPTS_FIELD_NUMBER: _ClassVar[int]
    RESULT_REFERENCE_FIELD_NUMBER: _ClassVar[int]
    ERROR_CODE_FIELD_NUMBER: _ClassVar[int]
    workspace_id: str
    id: str
    next_state: str
    provider: str
    attempts: int
    result_reference: str
    error_code: str
    def __init__(self, workspace_id: _Optional[str] = ..., id: _Optional[str] = ..., next_state: _Optional[str] = ..., provider: _Optional[str] = ..., attempts: _Optional[int] = ..., result_reference: _Optional[str] = ..., error_code: _Optional[str] = ...) -> None: ...

class TransitionJobResponse(_message.Message):
    __slots__ = ("job",)
    JOB_FIELD_NUMBER: _ClassVar[int]
    job: Job
    def __init__(self, job: _Optional[_Union[Job, _Mapping]] = ...) -> None: ...

class CancelJobRequest(_message.Message):
    __slots__ = ("workspace_id", "id")
    WORKSPACE_ID_FIELD_NUMBER: _ClassVar[int]
    ID_FIELD_NUMBER: _ClassVar[int]
    workspace_id: str
    id: str
    def __init__(self, workspace_id: _Optional[str] = ..., id: _Optional[str] = ...) -> None: ...

class CancelJobResponse(_message.Message):
    __slots__ = ("job",)
    JOB_FIELD_NUMBER: _ClassVar[int]
    job: Job
    def __init__(self, job: _Optional[_Union[Job, _Mapping]] = ...) -> None: ...
