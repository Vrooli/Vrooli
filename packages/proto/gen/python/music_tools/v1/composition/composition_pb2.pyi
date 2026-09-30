import datetime

from google.protobuf import timestamp_pb2 as _timestamp_pb2
from google.protobuf.internal import containers as _containers
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Iterable as _Iterable, Mapping as _Mapping
from typing import ClassVar as _ClassVar, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class Provenance(_message.Message):
    __slots__ = ("model_id", "license_lane", "applied_rung", "seed", "caption_as_authored", "caption_as_sent")
    MODEL_ID_FIELD_NUMBER: _ClassVar[int]
    LICENSE_LANE_FIELD_NUMBER: _ClassVar[int]
    APPLIED_RUNG_FIELD_NUMBER: _ClassVar[int]
    SEED_FIELD_NUMBER: _ClassVar[int]
    CAPTION_AS_AUTHORED_FIELD_NUMBER: _ClassVar[int]
    CAPTION_AS_SENT_FIELD_NUMBER: _ClassVar[int]
    model_id: str
    license_lane: str
    applied_rung: str
    seed: int
    caption_as_authored: str
    caption_as_sent: str
    def __init__(self, model_id: _Optional[str] = ..., license_lane: _Optional[str] = ..., applied_rung: _Optional[str] = ..., seed: _Optional[int] = ..., caption_as_authored: _Optional[str] = ..., caption_as_sent: _Optional[str] = ...) -> None: ...

class Take(_message.Message):
    __slots__ = ("id", "job_id", "style_id", "pool_state", "blob_ref", "provenance", "times_offered", "reserved_by", "created_at", "reserved_at")
    ID_FIELD_NUMBER: _ClassVar[int]
    JOB_ID_FIELD_NUMBER: _ClassVar[int]
    STYLE_ID_FIELD_NUMBER: _ClassVar[int]
    POOL_STATE_FIELD_NUMBER: _ClassVar[int]
    BLOB_REF_FIELD_NUMBER: _ClassVar[int]
    PROVENANCE_FIELD_NUMBER: _ClassVar[int]
    TIMES_OFFERED_FIELD_NUMBER: _ClassVar[int]
    RESERVED_BY_FIELD_NUMBER: _ClassVar[int]
    CREATED_AT_FIELD_NUMBER: _ClassVar[int]
    RESERVED_AT_FIELD_NUMBER: _ClassVar[int]
    id: str
    job_id: str
    style_id: str
    pool_state: str
    blob_ref: str
    provenance: Provenance
    times_offered: int
    reserved_by: str
    created_at: _timestamp_pb2.Timestamp
    reserved_at: _timestamp_pb2.Timestamp
    def __init__(self, id: _Optional[str] = ..., job_id: _Optional[str] = ..., style_id: _Optional[str] = ..., pool_state: _Optional[str] = ..., blob_ref: _Optional[str] = ..., provenance: _Optional[_Union[Provenance, _Mapping]] = ..., times_offered: _Optional[int] = ..., reserved_by: _Optional[str] = ..., created_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., reserved_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ...) -> None: ...

class SubmitRequest(_message.Message):
    __slots__ = ("style_id", "takes", "duration", "seed")
    STYLE_ID_FIELD_NUMBER: _ClassVar[int]
    TAKES_FIELD_NUMBER: _ClassVar[int]
    DURATION_FIELD_NUMBER: _ClassVar[int]
    SEED_FIELD_NUMBER: _ClassVar[int]
    style_id: str
    takes: int
    duration: int
    seed: int
    def __init__(self, style_id: _Optional[str] = ..., takes: _Optional[int] = ..., duration: _Optional[int] = ..., seed: _Optional[int] = ...) -> None: ...

class SubmitResponse(_message.Message):
    __slots__ = ("job_id",)
    JOB_ID_FIELD_NUMBER: _ClassVar[int]
    job_id: str
    def __init__(self, job_id: _Optional[str] = ...) -> None: ...

class ListTakesRequest(_message.Message):
    __slots__ = ("style_id", "job_id")
    STYLE_ID_FIELD_NUMBER: _ClassVar[int]
    JOB_ID_FIELD_NUMBER: _ClassVar[int]
    style_id: str
    job_id: str
    def __init__(self, style_id: _Optional[str] = ..., job_id: _Optional[str] = ...) -> None: ...

class ListTakesResponse(_message.Message):
    __slots__ = ("takes",)
    TAKES_FIELD_NUMBER: _ClassVar[int]
    takes: _containers.RepeatedCompositeFieldContainer[Take]
    def __init__(self, takes: _Optional[_Iterable[_Union[Take, _Mapping]]] = ...) -> None: ...
