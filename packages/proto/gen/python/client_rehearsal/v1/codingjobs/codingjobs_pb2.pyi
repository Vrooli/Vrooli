import datetime

from google.protobuf import timestamp_pb2 as _timestamp_pb2
from google.protobuf.internal import containers as _containers
from google.protobuf.internal import enum_type_wrapper as _enum_type_wrapper
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Iterable as _Iterable, Mapping as _Mapping
from typing import ClassVar as _ClassVar, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class CodingJobState(int, metaclass=_enum_type_wrapper.EnumTypeWrapper):
    __slots__ = ()
    CODING_JOB_STATE_UNSPECIFIED: _ClassVar[CodingJobState]
    CODING_JOB_STATE_READY: _ClassVar[CodingJobState]
    CODING_JOB_STATE_SOLVING: _ClassVar[CodingJobState]
    CODING_JOB_STATE_SCORING: _ClassVar[CodingJobState]
    CODING_JOB_STATE_REVIEWABLE: _ClassVar[CodingJobState]
    CODING_JOB_STATE_FAILED: _ClassVar[CodingJobState]
CODING_JOB_STATE_UNSPECIFIED: CodingJobState
CODING_JOB_STATE_READY: CodingJobState
CODING_JOB_STATE_SOLVING: CodingJobState
CODING_JOB_STATE_SCORING: CodingJobState
CODING_JOB_STATE_REVIEWABLE: CodingJobState
CODING_JOB_STATE_FAILED: CodingJobState

class CodingJob(_message.Message):
    __slots__ = ("id", "case_id", "project", "title", "task", "state", "source_seal_ref", "solver_run_ref", "evaluator_receipt_ref", "evaluator_receipt_sha256", "score", "events", "created_at", "updated_at", "revision", "replay_key")
    ID_FIELD_NUMBER: _ClassVar[int]
    CASE_ID_FIELD_NUMBER: _ClassVar[int]
    PROJECT_FIELD_NUMBER: _ClassVar[int]
    TITLE_FIELD_NUMBER: _ClassVar[int]
    TASK_FIELD_NUMBER: _ClassVar[int]
    STATE_FIELD_NUMBER: _ClassVar[int]
    SOURCE_SEAL_REF_FIELD_NUMBER: _ClassVar[int]
    SOLVER_RUN_REF_FIELD_NUMBER: _ClassVar[int]
    EVALUATOR_RECEIPT_REF_FIELD_NUMBER: _ClassVar[int]
    EVALUATOR_RECEIPT_SHA256_FIELD_NUMBER: _ClassVar[int]
    SCORE_FIELD_NUMBER: _ClassVar[int]
    EVENTS_FIELD_NUMBER: _ClassVar[int]
    CREATED_AT_FIELD_NUMBER: _ClassVar[int]
    UPDATED_AT_FIELD_NUMBER: _ClassVar[int]
    REVISION_FIELD_NUMBER: _ClassVar[int]
    REPLAY_KEY_FIELD_NUMBER: _ClassVar[int]
    id: str
    case_id: str
    project: str
    title: str
    task: str
    state: CodingJobState
    source_seal_ref: str
    solver_run_ref: str
    evaluator_receipt_ref: str
    evaluator_receipt_sha256: str
    score: str
    events: _containers.RepeatedCompositeFieldContainer[CodingJobEvent]
    created_at: _timestamp_pb2.Timestamp
    updated_at: _timestamp_pb2.Timestamp
    revision: int
    replay_key: str
    def __init__(self, id: _Optional[str] = ..., case_id: _Optional[str] = ..., project: _Optional[str] = ..., title: _Optional[str] = ..., task: _Optional[str] = ..., state: _Optional[_Union[CodingJobState, str]] = ..., source_seal_ref: _Optional[str] = ..., solver_run_ref: _Optional[str] = ..., evaluator_receipt_ref: _Optional[str] = ..., evaluator_receipt_sha256: _Optional[str] = ..., score: _Optional[str] = ..., events: _Optional[_Iterable[_Union[CodingJobEvent, _Mapping]]] = ..., created_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., updated_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., revision: _Optional[int] = ..., replay_key: _Optional[str] = ...) -> None: ...

class CodingJobEvent(_message.Message):
    __slots__ = ("sequence", "stage", "actor", "evidence_ref", "evidence_sha256", "note", "occurred_at")
    SEQUENCE_FIELD_NUMBER: _ClassVar[int]
    STAGE_FIELD_NUMBER: _ClassVar[int]
    ACTOR_FIELD_NUMBER: _ClassVar[int]
    EVIDENCE_REF_FIELD_NUMBER: _ClassVar[int]
    EVIDENCE_SHA256_FIELD_NUMBER: _ClassVar[int]
    NOTE_FIELD_NUMBER: _ClassVar[int]
    OCCURRED_AT_FIELD_NUMBER: _ClassVar[int]
    sequence: int
    stage: str
    actor: str
    evidence_ref: str
    evidence_sha256: str
    note: str
    occurred_at: _timestamp_pb2.Timestamp
    def __init__(self, sequence: _Optional[int] = ..., stage: _Optional[str] = ..., actor: _Optional[str] = ..., evidence_ref: _Optional[str] = ..., evidence_sha256: _Optional[str] = ..., note: _Optional[str] = ..., occurred_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ...) -> None: ...

class CreateCodingJobRequest(_message.Message):
    __slots__ = ("case_id", "project", "title", "task", "source_seal_ref", "replay_key")
    CASE_ID_FIELD_NUMBER: _ClassVar[int]
    PROJECT_FIELD_NUMBER: _ClassVar[int]
    TITLE_FIELD_NUMBER: _ClassVar[int]
    TASK_FIELD_NUMBER: _ClassVar[int]
    SOURCE_SEAL_REF_FIELD_NUMBER: _ClassVar[int]
    REPLAY_KEY_FIELD_NUMBER: _ClassVar[int]
    case_id: str
    project: str
    title: str
    task: str
    source_seal_ref: str
    replay_key: str
    def __init__(self, case_id: _Optional[str] = ..., project: _Optional[str] = ..., title: _Optional[str] = ..., task: _Optional[str] = ..., source_seal_ref: _Optional[str] = ..., replay_key: _Optional[str] = ...) -> None: ...

class CreateCodingJobResponse(_message.Message):
    __slots__ = ("job",)
    JOB_FIELD_NUMBER: _ClassVar[int]
    job: CodingJob
    def __init__(self, job: _Optional[_Union[CodingJob, _Mapping]] = ...) -> None: ...

class RecordCodingJobStageRequest(_message.Message):
    __slots__ = ("id", "stage", "actor", "evidence_ref", "evidence_sha256", "note", "expected_revision")
    ID_FIELD_NUMBER: _ClassVar[int]
    STAGE_FIELD_NUMBER: _ClassVar[int]
    ACTOR_FIELD_NUMBER: _ClassVar[int]
    EVIDENCE_REF_FIELD_NUMBER: _ClassVar[int]
    EVIDENCE_SHA256_FIELD_NUMBER: _ClassVar[int]
    NOTE_FIELD_NUMBER: _ClassVar[int]
    EXPECTED_REVISION_FIELD_NUMBER: _ClassVar[int]
    id: str
    stage: str
    actor: str
    evidence_ref: str
    evidence_sha256: str
    note: str
    expected_revision: int
    def __init__(self, id: _Optional[str] = ..., stage: _Optional[str] = ..., actor: _Optional[str] = ..., evidence_ref: _Optional[str] = ..., evidence_sha256: _Optional[str] = ..., note: _Optional[str] = ..., expected_revision: _Optional[int] = ...) -> None: ...

class RecordCodingJobStageResponse(_message.Message):
    __slots__ = ("job",)
    JOB_FIELD_NUMBER: _ClassVar[int]
    job: CodingJob
    def __init__(self, job: _Optional[_Union[CodingJob, _Mapping]] = ...) -> None: ...

class RecordCodingJobEvaluationRequest(_message.Message):
    __slots__ = ("id", "receipt_ref", "receipt_sha256", "score", "expected_revision")
    ID_FIELD_NUMBER: _ClassVar[int]
    RECEIPT_REF_FIELD_NUMBER: _ClassVar[int]
    RECEIPT_SHA256_FIELD_NUMBER: _ClassVar[int]
    SCORE_FIELD_NUMBER: _ClassVar[int]
    EXPECTED_REVISION_FIELD_NUMBER: _ClassVar[int]
    id: str
    receipt_ref: str
    receipt_sha256: str
    score: str
    expected_revision: int
    def __init__(self, id: _Optional[str] = ..., receipt_ref: _Optional[str] = ..., receipt_sha256: _Optional[str] = ..., score: _Optional[str] = ..., expected_revision: _Optional[int] = ...) -> None: ...

class RecordCodingJobEvaluationResponse(_message.Message):
    __slots__ = ("job",)
    JOB_FIELD_NUMBER: _ClassVar[int]
    job: CodingJob
    def __init__(self, job: _Optional[_Union[CodingJob, _Mapping]] = ...) -> None: ...

class GetCodingJobRequest(_message.Message):
    __slots__ = ("id",)
    ID_FIELD_NUMBER: _ClassVar[int]
    id: str
    def __init__(self, id: _Optional[str] = ...) -> None: ...

class GetCodingJobResponse(_message.Message):
    __slots__ = ("job",)
    JOB_FIELD_NUMBER: _ClassVar[int]
    job: CodingJob
    def __init__(self, job: _Optional[_Union[CodingJob, _Mapping]] = ...) -> None: ...

class ListCodingJobsRequest(_message.Message):
    __slots__ = ("limit", "cursor")
    LIMIT_FIELD_NUMBER: _ClassVar[int]
    CURSOR_FIELD_NUMBER: _ClassVar[int]
    limit: int
    cursor: str
    def __init__(self, limit: _Optional[int] = ..., cursor: _Optional[str] = ...) -> None: ...

class ListCodingJobsResponse(_message.Message):
    __slots__ = ("jobs", "next_cursor")
    JOBS_FIELD_NUMBER: _ClassVar[int]
    NEXT_CURSOR_FIELD_NUMBER: _ClassVar[int]
    jobs: _containers.RepeatedCompositeFieldContainer[CodingJob]
    next_cursor: str
    def __init__(self, jobs: _Optional[_Iterable[_Union[CodingJob, _Mapping]]] = ..., next_cursor: _Optional[str] = ...) -> None: ...
