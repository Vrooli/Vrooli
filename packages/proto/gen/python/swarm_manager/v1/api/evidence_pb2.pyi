from buf.validate import validate_pb2 as _validate_pb2
from google.protobuf import struct_pb2 as _struct_pb2
from google.protobuf.internal import containers as _containers
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Iterable as _Iterable, Mapping as _Mapping
from typing import ClassVar as _ClassVar, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class EvidenceRecord(_message.Message):
    __slots__ = ("owner_kind", "owner_id", "owner_round", "source_system", "source_event_id", "run_id", "subject_kind", "subject_id", "action", "confidence", "verification", "content_digest", "metadata", "observed_at", "linked_at")
    OWNER_KIND_FIELD_NUMBER: _ClassVar[int]
    OWNER_ID_FIELD_NUMBER: _ClassVar[int]
    OWNER_ROUND_FIELD_NUMBER: _ClassVar[int]
    SOURCE_SYSTEM_FIELD_NUMBER: _ClassVar[int]
    SOURCE_EVENT_ID_FIELD_NUMBER: _ClassVar[int]
    RUN_ID_FIELD_NUMBER: _ClassVar[int]
    SUBJECT_KIND_FIELD_NUMBER: _ClassVar[int]
    SUBJECT_ID_FIELD_NUMBER: _ClassVar[int]
    ACTION_FIELD_NUMBER: _ClassVar[int]
    CONFIDENCE_FIELD_NUMBER: _ClassVar[int]
    VERIFICATION_FIELD_NUMBER: _ClassVar[int]
    CONTENT_DIGEST_FIELD_NUMBER: _ClassVar[int]
    METADATA_FIELD_NUMBER: _ClassVar[int]
    OBSERVED_AT_FIELD_NUMBER: _ClassVar[int]
    LINKED_AT_FIELD_NUMBER: _ClassVar[int]
    owner_kind: str
    owner_id: str
    owner_round: int
    source_system: str
    source_event_id: str
    run_id: str
    subject_kind: str
    subject_id: str
    action: str
    confidence: str
    verification: str
    content_digest: str
    metadata: _struct_pb2.Struct
    observed_at: str
    linked_at: str
    def __init__(self, owner_kind: _Optional[str] = ..., owner_id: _Optional[str] = ..., owner_round: _Optional[int] = ..., source_system: _Optional[str] = ..., source_event_id: _Optional[str] = ..., run_id: _Optional[str] = ..., subject_kind: _Optional[str] = ..., subject_id: _Optional[str] = ..., action: _Optional[str] = ..., confidence: _Optional[str] = ..., verification: _Optional[str] = ..., content_digest: _Optional[str] = ..., metadata: _Optional[_Union[_struct_pb2.Struct, _Mapping]] = ..., observed_at: _Optional[str] = ..., linked_at: _Optional[str] = ...) -> None: ...

class EvidenceListRunRequest(_message.Message):
    __slots__ = ("run_id",)
    RUN_ID_FIELD_NUMBER: _ClassVar[int]
    run_id: str
    def __init__(self, run_id: _Optional[str] = ...) -> None: ...

class EvidenceListEntityRequest(_message.Message):
    __slots__ = ("subject_kind", "subject_id")
    SUBJECT_KIND_FIELD_NUMBER: _ClassVar[int]
    SUBJECT_ID_FIELD_NUMBER: _ClassVar[int]
    subject_kind: str
    subject_id: str
    def __init__(self, subject_kind: _Optional[str] = ..., subject_id: _Optional[str] = ...) -> None: ...

class EvidenceListResponse(_message.Message):
    __slots__ = ("records",)
    RECORDS_FIELD_NUMBER: _ClassVar[int]
    records: _containers.RepeatedCompositeFieldContainer[EvidenceRecord]
    def __init__(self, records: _Optional[_Iterable[_Union[EvidenceRecord, _Mapping]]] = ...) -> None: ...

class EvidenceReconcileRequest(_message.Message):
    __slots__ = ("run_id",)
    RUN_ID_FIELD_NUMBER: _ClassVar[int]
    run_id: str
    def __init__(self, run_id: _Optional[str] = ...) -> None: ...

class EvidenceReconcileResponse(_message.Message):
    __slots__ = ("run_id", "status")
    RUN_ID_FIELD_NUMBER: _ClassVar[int]
    STATUS_FIELD_NUMBER: _ClassVar[int]
    run_id: str
    status: str
    def __init__(self, run_id: _Optional[str] = ..., status: _Optional[str] = ...) -> None: ...

class EvidenceOperatorVerificationRequest(_message.Message):
    __slots__ = ("owner_kind", "owner_id", "owner_round", "event_id", "run_id", "subject_kind", "subject_id", "action", "actor", "reason", "metadata")
    OWNER_KIND_FIELD_NUMBER: _ClassVar[int]
    OWNER_ID_FIELD_NUMBER: _ClassVar[int]
    OWNER_ROUND_FIELD_NUMBER: _ClassVar[int]
    EVENT_ID_FIELD_NUMBER: _ClassVar[int]
    RUN_ID_FIELD_NUMBER: _ClassVar[int]
    SUBJECT_KIND_FIELD_NUMBER: _ClassVar[int]
    SUBJECT_ID_FIELD_NUMBER: _ClassVar[int]
    ACTION_FIELD_NUMBER: _ClassVar[int]
    ACTOR_FIELD_NUMBER: _ClassVar[int]
    REASON_FIELD_NUMBER: _ClassVar[int]
    METADATA_FIELD_NUMBER: _ClassVar[int]
    owner_kind: str
    owner_id: str
    owner_round: int
    event_id: str
    run_id: str
    subject_kind: str
    subject_id: str
    action: str
    actor: str
    reason: str
    metadata: _struct_pb2.Struct
    def __init__(self, owner_kind: _Optional[str] = ..., owner_id: _Optional[str] = ..., owner_round: _Optional[int] = ..., event_id: _Optional[str] = ..., run_id: _Optional[str] = ..., subject_kind: _Optional[str] = ..., subject_id: _Optional[str] = ..., action: _Optional[str] = ..., actor: _Optional[str] = ..., reason: _Optional[str] = ..., metadata: _Optional[_Union[_struct_pb2.Struct, _Mapping]] = ...) -> None: ...
