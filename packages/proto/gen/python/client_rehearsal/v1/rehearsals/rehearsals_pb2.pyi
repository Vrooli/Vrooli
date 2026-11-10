import datetime

from google.protobuf import timestamp_pb2 as _timestamp_pb2
from google.protobuf.internal import containers as _containers
from google.protobuf.internal import enum_type_wrapper as _enum_type_wrapper
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Iterable as _Iterable, Mapping as _Mapping
from typing import ClassVar as _ClassVar, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class RehearsalState(int, metaclass=_enum_type_wrapper.EnumTypeWrapper):
    __slots__ = ()
    REHEARSAL_STATE_UNSPECIFIED: _ClassVar[RehearsalState]
    REHEARSAL_STATE_NEEDS_REVIEW: _ClassVar[RehearsalState]
    REHEARSAL_STATE_APPROVED: _ClassVar[RehearsalState]
    REHEARSAL_STATE_CHANGES_REQUESTED: _ClassVar[RehearsalState]
    REHEARSAL_STATE_FAILED: _ClassVar[RehearsalState]

class ReviewDecision(int, metaclass=_enum_type_wrapper.EnumTypeWrapper):
    __slots__ = ()
    REVIEW_DECISION_UNSPECIFIED: _ClassVar[ReviewDecision]
    REVIEW_DECISION_APPROVE: _ClassVar[ReviewDecision]
    REVIEW_DECISION_REQUEST_CHANGES: _ClassVar[ReviewDecision]
REHEARSAL_STATE_UNSPECIFIED: RehearsalState
REHEARSAL_STATE_NEEDS_REVIEW: RehearsalState
REHEARSAL_STATE_APPROVED: RehearsalState
REHEARSAL_STATE_CHANGES_REQUESTED: RehearsalState
REHEARSAL_STATE_FAILED: RehearsalState
REVIEW_DECISION_UNSPECIFIED: ReviewDecision
REVIEW_DECISION_APPROVE: ReviewDecision
REVIEW_DECISION_REQUEST_CHANGES: ReviewDecision

class Rehearsal(_message.Message):
    __slots__ = ("id", "title", "source_format", "source_sha256", "credential_replacements", "simulator_profile", "state", "trace", "fault_summary", "repair_summary", "handoff_markdown", "review_decision", "review_note", "created_at", "reviewed_at", "simulator_source_sha256")
    ID_FIELD_NUMBER: _ClassVar[int]
    TITLE_FIELD_NUMBER: _ClassVar[int]
    SOURCE_FORMAT_FIELD_NUMBER: _ClassVar[int]
    SOURCE_SHA256_FIELD_NUMBER: _ClassVar[int]
    CREDENTIAL_REPLACEMENTS_FIELD_NUMBER: _ClassVar[int]
    SIMULATOR_PROFILE_FIELD_NUMBER: _ClassVar[int]
    STATE_FIELD_NUMBER: _ClassVar[int]
    TRACE_FIELD_NUMBER: _ClassVar[int]
    FAULT_SUMMARY_FIELD_NUMBER: _ClassVar[int]
    REPAIR_SUMMARY_FIELD_NUMBER: _ClassVar[int]
    HANDOFF_MARKDOWN_FIELD_NUMBER: _ClassVar[int]
    REVIEW_DECISION_FIELD_NUMBER: _ClassVar[int]
    REVIEW_NOTE_FIELD_NUMBER: _ClassVar[int]
    CREATED_AT_FIELD_NUMBER: _ClassVar[int]
    REVIEWED_AT_FIELD_NUMBER: _ClassVar[int]
    SIMULATOR_SOURCE_SHA256_FIELD_NUMBER: _ClassVar[int]
    id: str
    title: str
    source_format: str
    source_sha256: str
    credential_replacements: int
    simulator_profile: str
    state: RehearsalState
    trace: _containers.RepeatedCompositeFieldContainer[TraceEvent]
    fault_summary: str
    repair_summary: str
    handoff_markdown: str
    review_decision: ReviewDecision
    review_note: str
    created_at: _timestamp_pb2.Timestamp
    reviewed_at: _timestamp_pb2.Timestamp
    simulator_source_sha256: str
    def __init__(self, id: _Optional[str] = ..., title: _Optional[str] = ..., source_format: _Optional[str] = ..., source_sha256: _Optional[str] = ..., credential_replacements: _Optional[int] = ..., simulator_profile: _Optional[str] = ..., state: _Optional[_Union[RehearsalState, str]] = ..., trace: _Optional[_Iterable[_Union[TraceEvent, _Mapping]]] = ..., fault_summary: _Optional[str] = ..., repair_summary: _Optional[str] = ..., handoff_markdown: _Optional[str] = ..., review_decision: _Optional[_Union[ReviewDecision, str]] = ..., review_note: _Optional[str] = ..., created_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., reviewed_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., simulator_source_sha256: _Optional[str] = ...) -> None: ...

class TraceEvent(_message.Message):
    __slots__ = ("sequence", "operation", "request_summary", "response_summary", "status", "occurred_at")
    SEQUENCE_FIELD_NUMBER: _ClassVar[int]
    OPERATION_FIELD_NUMBER: _ClassVar[int]
    REQUEST_SUMMARY_FIELD_NUMBER: _ClassVar[int]
    RESPONSE_SUMMARY_FIELD_NUMBER: _ClassVar[int]
    STATUS_FIELD_NUMBER: _ClassVar[int]
    OCCURRED_AT_FIELD_NUMBER: _ClassVar[int]
    sequence: int
    operation: str
    request_summary: str
    response_summary: str
    status: str
    occurred_at: _timestamp_pb2.Timestamp
    def __init__(self, sequence: _Optional[int] = ..., operation: _Optional[str] = ..., request_summary: _Optional[str] = ..., response_summary: _Optional[str] = ..., status: _Optional[str] = ..., occurred_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ...) -> None: ...

class CreateAndRunRehearsalRequest(_message.Message):
    __slots__ = ("title", "workflow_json", "simulator_profile", "inject_seeded_fault", "simulator_source_json", "trigger_node_id")
    TITLE_FIELD_NUMBER: _ClassVar[int]
    WORKFLOW_JSON_FIELD_NUMBER: _ClassVar[int]
    SIMULATOR_PROFILE_FIELD_NUMBER: _ClassVar[int]
    INJECT_SEEDED_FAULT_FIELD_NUMBER: _ClassVar[int]
    SIMULATOR_SOURCE_JSON_FIELD_NUMBER: _ClassVar[int]
    TRIGGER_NODE_ID_FIELD_NUMBER: _ClassVar[int]
    title: str
    workflow_json: str
    simulator_profile: str
    inject_seeded_fault: bool
    simulator_source_json: str
    trigger_node_id: str
    def __init__(self, title: _Optional[str] = ..., workflow_json: _Optional[str] = ..., simulator_profile: _Optional[str] = ..., inject_seeded_fault: _Optional[bool] = ..., simulator_source_json: _Optional[str] = ..., trigger_node_id: _Optional[str] = ...) -> None: ...

class CreateAndRunRehearsalResponse(_message.Message):
    __slots__ = ("rehearsal",)
    REHEARSAL_FIELD_NUMBER: _ClassVar[int]
    rehearsal: Rehearsal
    def __init__(self, rehearsal: _Optional[_Union[Rehearsal, _Mapping]] = ...) -> None: ...

class ListRehearsalsRequest(_message.Message):
    __slots__ = ("limit",)
    LIMIT_FIELD_NUMBER: _ClassVar[int]
    limit: int
    def __init__(self, limit: _Optional[int] = ...) -> None: ...

class ListRehearsalsResponse(_message.Message):
    __slots__ = ("rehearsals",)
    REHEARSALS_FIELD_NUMBER: _ClassVar[int]
    rehearsals: _containers.RepeatedCompositeFieldContainer[Rehearsal]
    def __init__(self, rehearsals: _Optional[_Iterable[_Union[Rehearsal, _Mapping]]] = ...) -> None: ...

class GetRehearsalRequest(_message.Message):
    __slots__ = ("id",)
    ID_FIELD_NUMBER: _ClassVar[int]
    id: str
    def __init__(self, id: _Optional[str] = ...) -> None: ...

class GetRehearsalResponse(_message.Message):
    __slots__ = ("rehearsal",)
    REHEARSAL_FIELD_NUMBER: _ClassVar[int]
    rehearsal: Rehearsal
    def __init__(self, rehearsal: _Optional[_Union[Rehearsal, _Mapping]] = ...) -> None: ...

class RecordReviewRequest(_message.Message):
    __slots__ = ("id", "decision", "note")
    ID_FIELD_NUMBER: _ClassVar[int]
    DECISION_FIELD_NUMBER: _ClassVar[int]
    NOTE_FIELD_NUMBER: _ClassVar[int]
    id: str
    decision: ReviewDecision
    note: str
    def __init__(self, id: _Optional[str] = ..., decision: _Optional[_Union[ReviewDecision, str]] = ..., note: _Optional[str] = ...) -> None: ...

class RecordReviewResponse(_message.Message):
    __slots__ = ("rehearsal",)
    REHEARSAL_FIELD_NUMBER: _ClassVar[int]
    rehearsal: Rehearsal
    def __init__(self, rehearsal: _Optional[_Union[Rehearsal, _Mapping]] = ...) -> None: ...
