import datetime

from google.protobuf import timestamp_pb2 as _timestamp_pb2
from google.protobuf.internal import containers as _containers
from google.protobuf.internal import enum_type_wrapper as _enum_type_wrapper
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Iterable as _Iterable, Mapping as _Mapping
from typing import ClassVar as _ClassVar, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class TupleKind(int, metaclass=_enum_type_wrapper.EnumTypeWrapper):
    __slots__ = ()
    TUPLE_KIND_UNSPECIFIED: _ClassVar[TupleKind]
    TUPLE_KIND_SKILL: _ClassVar[TupleKind]
    TUPLE_KIND_TOOL: _ClassVar[TupleKind]

class Verdict(int, metaclass=_enum_type_wrapper.EnumTypeWrapper):
    __slots__ = ()
    VERDICT_UNSPECIFIED: _ClassVar[Verdict]
    VERDICT_PASS: _ClassVar[Verdict]
    VERDICT_UNEXPECTED_MUTATION: _ClassVar[Verdict]
    VERDICT_RUN_FAILURE: _ClassVar[Verdict]
    VERDICT_TOOL_FAILURE: _ClassVar[Verdict]
TUPLE_KIND_UNSPECIFIED: TupleKind
TUPLE_KIND_SKILL: TupleKind
TUPLE_KIND_TOOL: TupleKind
VERDICT_UNSPECIFIED: Verdict
VERDICT_PASS: Verdict
VERDICT_UNEXPECTED_MUTATION: Verdict
VERDICT_RUN_FAILURE: Verdict
VERDICT_TOOL_FAILURE: Verdict

class ValidationRecord(_message.Message):
    __slots__ = ("id", "tuple_kind", "subject_id", "golden_slug", "started_at", "ended_at", "duration_ms", "tokens_used", "cost_usd_micro", "verdict", "diff_hash", "diff_path_count", "agent_manager_run_id", "manifest_template_version_at_run", "manifest_skill_version_at_run", "error_message", "tool_detail", "tool_raw_output")
    ID_FIELD_NUMBER: _ClassVar[int]
    TUPLE_KIND_FIELD_NUMBER: _ClassVar[int]
    SUBJECT_ID_FIELD_NUMBER: _ClassVar[int]
    GOLDEN_SLUG_FIELD_NUMBER: _ClassVar[int]
    STARTED_AT_FIELD_NUMBER: _ClassVar[int]
    ENDED_AT_FIELD_NUMBER: _ClassVar[int]
    DURATION_MS_FIELD_NUMBER: _ClassVar[int]
    TOKENS_USED_FIELD_NUMBER: _ClassVar[int]
    COST_USD_MICRO_FIELD_NUMBER: _ClassVar[int]
    VERDICT_FIELD_NUMBER: _ClassVar[int]
    DIFF_HASH_FIELD_NUMBER: _ClassVar[int]
    DIFF_PATH_COUNT_FIELD_NUMBER: _ClassVar[int]
    AGENT_MANAGER_RUN_ID_FIELD_NUMBER: _ClassVar[int]
    MANIFEST_TEMPLATE_VERSION_AT_RUN_FIELD_NUMBER: _ClassVar[int]
    MANIFEST_SKILL_VERSION_AT_RUN_FIELD_NUMBER: _ClassVar[int]
    ERROR_MESSAGE_FIELD_NUMBER: _ClassVar[int]
    TOOL_DETAIL_FIELD_NUMBER: _ClassVar[int]
    TOOL_RAW_OUTPUT_FIELD_NUMBER: _ClassVar[int]
    id: str
    tuple_kind: TupleKind
    subject_id: str
    golden_slug: str
    started_at: _timestamp_pb2.Timestamp
    ended_at: _timestamp_pb2.Timestamp
    duration_ms: int
    tokens_used: int
    cost_usd_micro: int
    verdict: Verdict
    diff_hash: str
    diff_path_count: int
    agent_manager_run_id: str
    manifest_template_version_at_run: str
    manifest_skill_version_at_run: str
    error_message: str
    tool_detail: str
    tool_raw_output: str
    def __init__(self, id: _Optional[str] = ..., tuple_kind: _Optional[_Union[TupleKind, str]] = ..., subject_id: _Optional[str] = ..., golden_slug: _Optional[str] = ..., started_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., ended_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., duration_ms: _Optional[int] = ..., tokens_used: _Optional[int] = ..., cost_usd_micro: _Optional[int] = ..., verdict: _Optional[_Union[Verdict, str]] = ..., diff_hash: _Optional[str] = ..., diff_path_count: _Optional[int] = ..., agent_manager_run_id: _Optional[str] = ..., manifest_template_version_at_run: _Optional[str] = ..., manifest_skill_version_at_run: _Optional[str] = ..., error_message: _Optional[str] = ..., tool_detail: _Optional[str] = ..., tool_raw_output: _Optional[str] = ...) -> None: ...

class ListRecordsRequest(_message.Message):
    __slots__ = ("golden_slug", "subject_id", "tuple_kind", "page_size", "page_token")
    GOLDEN_SLUG_FIELD_NUMBER: _ClassVar[int]
    SUBJECT_ID_FIELD_NUMBER: _ClassVar[int]
    TUPLE_KIND_FIELD_NUMBER: _ClassVar[int]
    PAGE_SIZE_FIELD_NUMBER: _ClassVar[int]
    PAGE_TOKEN_FIELD_NUMBER: _ClassVar[int]
    golden_slug: str
    subject_id: str
    tuple_kind: TupleKind
    page_size: int
    page_token: str
    def __init__(self, golden_slug: _Optional[str] = ..., subject_id: _Optional[str] = ..., tuple_kind: _Optional[_Union[TupleKind, str]] = ..., page_size: _Optional[int] = ..., page_token: _Optional[str] = ...) -> None: ...

class ListRecordsResponse(_message.Message):
    __slots__ = ("records", "next_page_token")
    RECORDS_FIELD_NUMBER: _ClassVar[int]
    NEXT_PAGE_TOKEN_FIELD_NUMBER: _ClassVar[int]
    records: _containers.RepeatedCompositeFieldContainer[ValidationRecord]
    next_page_token: str
    def __init__(self, records: _Optional[_Iterable[_Union[ValidationRecord, _Mapping]]] = ..., next_page_token: _Optional[str] = ...) -> None: ...

class GetRecordRequest(_message.Message):
    __slots__ = ("id",)
    ID_FIELD_NUMBER: _ClassVar[int]
    id: str
    def __init__(self, id: _Optional[str] = ...) -> None: ...

class GetRecordResponse(_message.Message):
    __slots__ = ("record",)
    RECORD_FIELD_NUMBER: _ClassVar[int]
    record: ValidationRecord
    def __init__(self, record: _Optional[_Union[ValidationRecord, _Mapping]] = ...) -> None: ...
