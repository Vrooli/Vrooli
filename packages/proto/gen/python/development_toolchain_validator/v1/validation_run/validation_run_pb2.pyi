import datetime

from google.protobuf import timestamp_pb2 as _timestamp_pb2
from development_toolchain_validator.v1.validation_record import validation_record_pb2 as _validation_record_pb2
from google.protobuf.internal import containers as _containers
from google.protobuf.internal import enum_type_wrapper as _enum_type_wrapper
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Iterable as _Iterable, Mapping as _Mapping
from typing import ClassVar as _ClassVar, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class Status(int, metaclass=_enum_type_wrapper.EnumTypeWrapper):
    __slots__ = ()
    STATUS_UNSPECIFIED: _ClassVar[Status]
    STATUS_QUEUED: _ClassVar[Status]
    STATUS_RUNNING: _ClassVar[Status]
    STATUS_EVALUATING: _ClassVar[Status]
    STATUS_TERMINAL: _ClassVar[Status]
STATUS_UNSPECIFIED: Status
STATUS_QUEUED: Status
STATUS_RUNNING: Status
STATUS_EVALUATING: Status
STATUS_TERMINAL: Status

class ValidationRun(_message.Message):
    __slots__ = ("id", "tuple_kind", "subject_id", "golden_slug", "status", "terminal_verdict", "agent_manager_run_id", "created_at", "started_at", "ended_at", "error_message")
    ID_FIELD_NUMBER: _ClassVar[int]
    TUPLE_KIND_FIELD_NUMBER: _ClassVar[int]
    SUBJECT_ID_FIELD_NUMBER: _ClassVar[int]
    GOLDEN_SLUG_FIELD_NUMBER: _ClassVar[int]
    STATUS_FIELD_NUMBER: _ClassVar[int]
    TERMINAL_VERDICT_FIELD_NUMBER: _ClassVar[int]
    AGENT_MANAGER_RUN_ID_FIELD_NUMBER: _ClassVar[int]
    CREATED_AT_FIELD_NUMBER: _ClassVar[int]
    STARTED_AT_FIELD_NUMBER: _ClassVar[int]
    ENDED_AT_FIELD_NUMBER: _ClassVar[int]
    ERROR_MESSAGE_FIELD_NUMBER: _ClassVar[int]
    id: str
    tuple_kind: _validation_record_pb2.TupleKind
    subject_id: str
    golden_slug: str
    status: Status
    terminal_verdict: _validation_record_pb2.Verdict
    agent_manager_run_id: str
    created_at: _timestamp_pb2.Timestamp
    started_at: _timestamp_pb2.Timestamp
    ended_at: _timestamp_pb2.Timestamp
    error_message: str
    def __init__(self, id: _Optional[str] = ..., tuple_kind: _Optional[_Union[_validation_record_pb2.TupleKind, str]] = ..., subject_id: _Optional[str] = ..., golden_slug: _Optional[str] = ..., status: _Optional[_Union[Status, str]] = ..., terminal_verdict: _Optional[_Union[_validation_record_pb2.Verdict, str]] = ..., agent_manager_run_id: _Optional[str] = ..., created_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., started_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., ended_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., error_message: _Optional[str] = ...) -> None: ...

class StartRequest(_message.Message):
    __slots__ = ("tuple_kind", "subject_id", "golden_slug", "force")
    TUPLE_KIND_FIELD_NUMBER: _ClassVar[int]
    SUBJECT_ID_FIELD_NUMBER: _ClassVar[int]
    GOLDEN_SLUG_FIELD_NUMBER: _ClassVar[int]
    FORCE_FIELD_NUMBER: _ClassVar[int]
    tuple_kind: _validation_record_pb2.TupleKind
    subject_id: str
    golden_slug: str
    force: bool
    def __init__(self, tuple_kind: _Optional[_Union[_validation_record_pb2.TupleKind, str]] = ..., subject_id: _Optional[str] = ..., golden_slug: _Optional[str] = ..., force: _Optional[bool] = ...) -> None: ...

class StartResponse(_message.Message):
    __slots__ = ("run",)
    RUN_FIELD_NUMBER: _ClassVar[int]
    run: ValidationRun
    def __init__(self, run: _Optional[_Union[ValidationRun, _Mapping]] = ...) -> None: ...

class GetRequest(_message.Message):
    __slots__ = ("id",)
    ID_FIELD_NUMBER: _ClassVar[int]
    id: str
    def __init__(self, id: _Optional[str] = ...) -> None: ...

class GetResponse(_message.Message):
    __slots__ = ("run",)
    RUN_FIELD_NUMBER: _ClassVar[int]
    run: ValidationRun
    def __init__(self, run: _Optional[_Union[ValidationRun, _Mapping]] = ...) -> None: ...

class ListActiveRequest(_message.Message):
    __slots__ = ()
    def __init__(self) -> None: ...

class ListActiveResponse(_message.Message):
    __slots__ = ("runs",)
    RUNS_FIELD_NUMBER: _ClassVar[int]
    runs: _containers.RepeatedCompositeFieldContainer[ValidationRun]
    def __init__(self, runs: _Optional[_Iterable[_Union[ValidationRun, _Mapping]]] = ...) -> None: ...
