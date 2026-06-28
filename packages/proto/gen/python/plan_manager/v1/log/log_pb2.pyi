from plan_manager.v1.shared import model_pb2 as _model_pb2
from google.protobuf.internal import containers as _containers
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Iterable as _Iterable, Mapping as _Mapping
from typing import ClassVar as _ClassVar, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class AddEntryResponse(_message.Message):
    __slots__ = ("entry", "step", "deduplicated")
    ENTRY_FIELD_NUMBER: _ClassVar[int]
    STEP_FIELD_NUMBER: _ClassVar[int]
    DEDUPLICATED_FIELD_NUMBER: _ClassVar[int]
    entry: _model_pb2.LogEntry
    step: _model_pb2.GuidedStep
    deduplicated: bool
    def __init__(self, entry: _Optional[_Union[_model_pb2.LogEntry, _Mapping]] = ..., step: _Optional[_Union[_model_pb2.GuidedStep, _Mapping]] = ..., deduplicated: _Optional[bool] = ...) -> None: ...

class AddDecisionRequest(_message.Message):
    __slots__ = ("plan_or_execution", "phase_id", "title", "detail", "evidence", "source_command", "idempotency_key", "run_id")
    PLAN_OR_EXECUTION_FIELD_NUMBER: _ClassVar[int]
    PHASE_ID_FIELD_NUMBER: _ClassVar[int]
    TITLE_FIELD_NUMBER: _ClassVar[int]
    DETAIL_FIELD_NUMBER: _ClassVar[int]
    EVIDENCE_FIELD_NUMBER: _ClassVar[int]
    SOURCE_COMMAND_FIELD_NUMBER: _ClassVar[int]
    IDEMPOTENCY_KEY_FIELD_NUMBER: _ClassVar[int]
    RUN_ID_FIELD_NUMBER: _ClassVar[int]
    plan_or_execution: str
    phase_id: str
    title: str
    detail: str
    evidence: _containers.RepeatedScalarFieldContainer[str]
    source_command: str
    idempotency_key: str
    run_id: str
    def __init__(self, plan_or_execution: _Optional[str] = ..., phase_id: _Optional[str] = ..., title: _Optional[str] = ..., detail: _Optional[str] = ..., evidence: _Optional[_Iterable[str]] = ..., source_command: _Optional[str] = ..., idempotency_key: _Optional[str] = ..., run_id: _Optional[str] = ...) -> None: ...

class AddFindingRequest(_message.Message):
    __slots__ = ("plan_or_execution", "phase_id", "title", "detail", "severity", "evidence", "source_command", "idempotency_key", "run_id")
    PLAN_OR_EXECUTION_FIELD_NUMBER: _ClassVar[int]
    PHASE_ID_FIELD_NUMBER: _ClassVar[int]
    TITLE_FIELD_NUMBER: _ClassVar[int]
    DETAIL_FIELD_NUMBER: _ClassVar[int]
    SEVERITY_FIELD_NUMBER: _ClassVar[int]
    EVIDENCE_FIELD_NUMBER: _ClassVar[int]
    SOURCE_COMMAND_FIELD_NUMBER: _ClassVar[int]
    IDEMPOTENCY_KEY_FIELD_NUMBER: _ClassVar[int]
    RUN_ID_FIELD_NUMBER: _ClassVar[int]
    plan_or_execution: str
    phase_id: str
    title: str
    detail: str
    severity: _model_pb2.LogSeverity
    evidence: _containers.RepeatedScalarFieldContainer[str]
    source_command: str
    idempotency_key: str
    run_id: str
    def __init__(self, plan_or_execution: _Optional[str] = ..., phase_id: _Optional[str] = ..., title: _Optional[str] = ..., detail: _Optional[str] = ..., severity: _Optional[_Union[_model_pb2.LogSeverity, str]] = ..., evidence: _Optional[_Iterable[str]] = ..., source_command: _Optional[str] = ..., idempotency_key: _Optional[str] = ..., run_id: _Optional[str] = ...) -> None: ...

class AddBugRequest(_message.Message):
    __slots__ = ("plan_or_execution", "phase_id", "title", "detail", "severity", "evidence", "source_command", "idempotency_key", "run_id")
    PLAN_OR_EXECUTION_FIELD_NUMBER: _ClassVar[int]
    PHASE_ID_FIELD_NUMBER: _ClassVar[int]
    TITLE_FIELD_NUMBER: _ClassVar[int]
    DETAIL_FIELD_NUMBER: _ClassVar[int]
    SEVERITY_FIELD_NUMBER: _ClassVar[int]
    EVIDENCE_FIELD_NUMBER: _ClassVar[int]
    SOURCE_COMMAND_FIELD_NUMBER: _ClassVar[int]
    IDEMPOTENCY_KEY_FIELD_NUMBER: _ClassVar[int]
    RUN_ID_FIELD_NUMBER: _ClassVar[int]
    plan_or_execution: str
    phase_id: str
    title: str
    detail: str
    severity: _model_pb2.LogSeverity
    evidence: _containers.RepeatedScalarFieldContainer[str]
    source_command: str
    idempotency_key: str
    run_id: str
    def __init__(self, plan_or_execution: _Optional[str] = ..., phase_id: _Optional[str] = ..., title: _Optional[str] = ..., detail: _Optional[str] = ..., severity: _Optional[_Union[_model_pb2.LogSeverity, str]] = ..., evidence: _Optional[_Iterable[str]] = ..., source_command: _Optional[str] = ..., idempotency_key: _Optional[str] = ..., run_id: _Optional[str] = ...) -> None: ...

class AddRecordRequest(_message.Message):
    __slots__ = ("plan_or_execution", "phase_id", "title", "detail", "evidence", "source_command", "idempotency_key", "run_id")
    PLAN_OR_EXECUTION_FIELD_NUMBER: _ClassVar[int]
    PHASE_ID_FIELD_NUMBER: _ClassVar[int]
    TITLE_FIELD_NUMBER: _ClassVar[int]
    DETAIL_FIELD_NUMBER: _ClassVar[int]
    EVIDENCE_FIELD_NUMBER: _ClassVar[int]
    SOURCE_COMMAND_FIELD_NUMBER: _ClassVar[int]
    IDEMPOTENCY_KEY_FIELD_NUMBER: _ClassVar[int]
    RUN_ID_FIELD_NUMBER: _ClassVar[int]
    plan_or_execution: str
    phase_id: str
    title: str
    detail: str
    evidence: _containers.RepeatedScalarFieldContainer[str]
    source_command: str
    idempotency_key: str
    run_id: str
    def __init__(self, plan_or_execution: _Optional[str] = ..., phase_id: _Optional[str] = ..., title: _Optional[str] = ..., detail: _Optional[str] = ..., evidence: _Optional[_Iterable[str]] = ..., source_command: _Optional[str] = ..., idempotency_key: _Optional[str] = ..., run_id: _Optional[str] = ...) -> None: ...

class AddNoteRequest(_message.Message):
    __slots__ = ("plan_or_execution", "phase_id", "title", "detail", "evidence", "source_command", "idempotency_key", "run_id")
    PLAN_OR_EXECUTION_FIELD_NUMBER: _ClassVar[int]
    PHASE_ID_FIELD_NUMBER: _ClassVar[int]
    TITLE_FIELD_NUMBER: _ClassVar[int]
    DETAIL_FIELD_NUMBER: _ClassVar[int]
    EVIDENCE_FIELD_NUMBER: _ClassVar[int]
    SOURCE_COMMAND_FIELD_NUMBER: _ClassVar[int]
    IDEMPOTENCY_KEY_FIELD_NUMBER: _ClassVar[int]
    RUN_ID_FIELD_NUMBER: _ClassVar[int]
    plan_or_execution: str
    phase_id: str
    title: str
    detail: str
    evidence: _containers.RepeatedScalarFieldContainer[str]
    source_command: str
    idempotency_key: str
    run_id: str
    def __init__(self, plan_or_execution: _Optional[str] = ..., phase_id: _Optional[str] = ..., title: _Optional[str] = ..., detail: _Optional[str] = ..., evidence: _Optional[_Iterable[str]] = ..., source_command: _Optional[str] = ..., idempotency_key: _Optional[str] = ..., run_id: _Optional[str] = ...) -> None: ...

class ListEntriesRequest(_message.Message):
    __slots__ = ("plan_or_execution", "phase_id", "type", "triage", "sync_status")
    PLAN_OR_EXECUTION_FIELD_NUMBER: _ClassVar[int]
    PHASE_ID_FIELD_NUMBER: _ClassVar[int]
    TYPE_FIELD_NUMBER: _ClassVar[int]
    TRIAGE_FIELD_NUMBER: _ClassVar[int]
    SYNC_STATUS_FIELD_NUMBER: _ClassVar[int]
    plan_or_execution: str
    phase_id: str
    type: _model_pb2.LogEntryType
    triage: _model_pb2.FindingTriage
    sync_status: _model_pb2.LogSyncStatus
    def __init__(self, plan_or_execution: _Optional[str] = ..., phase_id: _Optional[str] = ..., type: _Optional[_Union[_model_pb2.LogEntryType, str]] = ..., triage: _Optional[_Union[_model_pb2.FindingTriage, str]] = ..., sync_status: _Optional[_Union[_model_pb2.LogSyncStatus, str]] = ...) -> None: ...

class ListEntriesResponse(_message.Message):
    __slots__ = ("entries", "summary", "step")
    ENTRIES_FIELD_NUMBER: _ClassVar[int]
    SUMMARY_FIELD_NUMBER: _ClassVar[int]
    STEP_FIELD_NUMBER: _ClassVar[int]
    entries: _containers.RepeatedCompositeFieldContainer[_model_pb2.LogEntry]
    summary: _model_pb2.LogSummary
    step: _model_pb2.GuidedStep
    def __init__(self, entries: _Optional[_Iterable[_Union[_model_pb2.LogEntry, _Mapping]]] = ..., summary: _Optional[_Union[_model_pb2.LogSummary, _Mapping]] = ..., step: _Optional[_Union[_model_pb2.GuidedStep, _Mapping]] = ...) -> None: ...

class GetEntryRequest(_message.Message):
    __slots__ = ("id",)
    ID_FIELD_NUMBER: _ClassVar[int]
    id: str
    def __init__(self, id: _Optional[str] = ...) -> None: ...

class GetEntryResponse(_message.Message):
    __slots__ = ("entry", "step")
    ENTRY_FIELD_NUMBER: _ClassVar[int]
    STEP_FIELD_NUMBER: _ClassVar[int]
    entry: _model_pb2.LogEntry
    step: _model_pb2.GuidedStep
    def __init__(self, entry: _Optional[_Union[_model_pb2.LogEntry, _Mapping]] = ..., step: _Optional[_Union[_model_pb2.GuidedStep, _Mapping]] = ...) -> None: ...

class UpdateEntryRequest(_message.Message):
    __slots__ = ("id", "title", "detail", "severity", "triage", "add_evidence")
    ID_FIELD_NUMBER: _ClassVar[int]
    TITLE_FIELD_NUMBER: _ClassVar[int]
    DETAIL_FIELD_NUMBER: _ClassVar[int]
    SEVERITY_FIELD_NUMBER: _ClassVar[int]
    TRIAGE_FIELD_NUMBER: _ClassVar[int]
    ADD_EVIDENCE_FIELD_NUMBER: _ClassVar[int]
    id: str
    title: str
    detail: str
    severity: _model_pb2.LogSeverity
    triage: _model_pb2.FindingTriage
    add_evidence: _containers.RepeatedScalarFieldContainer[str]
    def __init__(self, id: _Optional[str] = ..., title: _Optional[str] = ..., detail: _Optional[str] = ..., severity: _Optional[_Union[_model_pb2.LogSeverity, str]] = ..., triage: _Optional[_Union[_model_pb2.FindingTriage, str]] = ..., add_evidence: _Optional[_Iterable[str]] = ...) -> None: ...

class PromoteEntryRequest(_message.Message):
    __slots__ = ("id", "to_type", "title", "detail", "severity")
    ID_FIELD_NUMBER: _ClassVar[int]
    TO_TYPE_FIELD_NUMBER: _ClassVar[int]
    TITLE_FIELD_NUMBER: _ClassVar[int]
    DETAIL_FIELD_NUMBER: _ClassVar[int]
    SEVERITY_FIELD_NUMBER: _ClassVar[int]
    id: str
    to_type: _model_pb2.LogEntryType
    title: str
    detail: str
    severity: _model_pb2.LogSeverity
    def __init__(self, id: _Optional[str] = ..., to_type: _Optional[_Union[_model_pb2.LogEntryType, str]] = ..., title: _Optional[str] = ..., detail: _Optional[str] = ..., severity: _Optional[_Union[_model_pb2.LogSeverity, str]] = ...) -> None: ...

class PromoteEntryResponse(_message.Message):
    __slots__ = ("entry", "source", "step")
    ENTRY_FIELD_NUMBER: _ClassVar[int]
    SOURCE_FIELD_NUMBER: _ClassVar[int]
    STEP_FIELD_NUMBER: _ClassVar[int]
    entry: _model_pb2.LogEntry
    source: _model_pb2.LogEntry
    step: _model_pb2.GuidedStep
    def __init__(self, entry: _Optional[_Union[_model_pb2.LogEntry, _Mapping]] = ..., source: _Optional[_Union[_model_pb2.LogEntry, _Mapping]] = ..., step: _Optional[_Union[_model_pb2.GuidedStep, _Mapping]] = ...) -> None: ...

class SyncEntryRequest(_message.Message):
    __slots__ = ("id",)
    ID_FIELD_NUMBER: _ClassVar[int]
    id: str
    def __init__(self, id: _Optional[str] = ...) -> None: ...
