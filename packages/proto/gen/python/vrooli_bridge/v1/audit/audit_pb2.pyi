import datetime

from google.protobuf import timestamp_pb2 as _timestamp_pb2
from google.protobuf.internal import containers as _containers
from google.protobuf.internal import enum_type_wrapper as _enum_type_wrapper
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Iterable as _Iterable, Mapping as _Mapping
from typing import ClassVar as _ClassVar, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class AuditAction(int, metaclass=_enum_type_wrapper.EnumTypeWrapper):
    __slots__ = ()
    AUDIT_ACTION_UNSPECIFIED: _ClassVar[AuditAction]
    AUDIT_ACTION_DISPATCH: _ClassVar[AuditAction]
    AUDIT_ACTION_PROVISION: _ClassVar[AuditAction]
    AUDIT_ACTION_BREAK_GLASS: _ClassVar[AuditAction]

class AuditOutcome(int, metaclass=_enum_type_wrapper.EnumTypeWrapper):
    __slots__ = ()
    AUDIT_OUTCOME_UNSPECIFIED: _ClassVar[AuditOutcome]
    AUDIT_OUTCOME_ACCEPTED: _ClassVar[AuditOutcome]
    AUDIT_OUTCOME_REJECTED: _ClassVar[AuditOutcome]
    AUDIT_OUTCOME_COMPLETED: _ClassVar[AuditOutcome]
    AUDIT_OUTCOME_FAILED: _ClassVar[AuditOutcome]
AUDIT_ACTION_UNSPECIFIED: AuditAction
AUDIT_ACTION_DISPATCH: AuditAction
AUDIT_ACTION_PROVISION: AuditAction
AUDIT_ACTION_BREAK_GLASS: AuditAction
AUDIT_OUTCOME_UNSPECIFIED: AuditOutcome
AUDIT_OUTCOME_ACCEPTED: AuditOutcome
AUDIT_OUTCOME_REJECTED: AuditOutcome
AUDIT_OUTCOME_COMPLETED: AuditOutcome
AUDIT_OUTCOME_FAILED: AuditOutcome

class AuditRecord(_message.Message):
    __slots__ = ("id", "action", "actor", "node_id", "scenario", "verb", "args", "outcome", "detail", "run_id", "recorded_at")
    ID_FIELD_NUMBER: _ClassVar[int]
    ACTION_FIELD_NUMBER: _ClassVar[int]
    ACTOR_FIELD_NUMBER: _ClassVar[int]
    NODE_ID_FIELD_NUMBER: _ClassVar[int]
    SCENARIO_FIELD_NUMBER: _ClassVar[int]
    VERB_FIELD_NUMBER: _ClassVar[int]
    ARGS_FIELD_NUMBER: _ClassVar[int]
    OUTCOME_FIELD_NUMBER: _ClassVar[int]
    DETAIL_FIELD_NUMBER: _ClassVar[int]
    RUN_ID_FIELD_NUMBER: _ClassVar[int]
    RECORDED_AT_FIELD_NUMBER: _ClassVar[int]
    id: str
    action: AuditAction
    actor: str
    node_id: str
    scenario: str
    verb: str
    args: _containers.RepeatedScalarFieldContainer[str]
    outcome: AuditOutcome
    detail: str
    run_id: str
    recorded_at: _timestamp_pb2.Timestamp
    def __init__(self, id: _Optional[str] = ..., action: _Optional[_Union[AuditAction, str]] = ..., actor: _Optional[str] = ..., node_id: _Optional[str] = ..., scenario: _Optional[str] = ..., verb: _Optional[str] = ..., args: _Optional[_Iterable[str]] = ..., outcome: _Optional[_Union[AuditOutcome, str]] = ..., detail: _Optional[str] = ..., run_id: _Optional[str] = ..., recorded_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ...) -> None: ...

class ListAuditRecordsRequest(_message.Message):
    __slots__ = ("node_id", "run_id", "limit")
    NODE_ID_FIELD_NUMBER: _ClassVar[int]
    RUN_ID_FIELD_NUMBER: _ClassVar[int]
    LIMIT_FIELD_NUMBER: _ClassVar[int]
    node_id: str
    run_id: str
    limit: int
    def __init__(self, node_id: _Optional[str] = ..., run_id: _Optional[str] = ..., limit: _Optional[int] = ...) -> None: ...

class ListAuditRecordsResponse(_message.Message):
    __slots__ = ("records",)
    RECORDS_FIELD_NUMBER: _ClassVar[int]
    records: _containers.RepeatedCompositeFieldContainer[AuditRecord]
    def __init__(self, records: _Optional[_Iterable[_Union[AuditRecord, _Mapping]]] = ...) -> None: ...
