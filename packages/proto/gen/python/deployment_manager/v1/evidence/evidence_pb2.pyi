import datetime

from buf.validate import validate_pb2 as _validate_pb2
from common.v1 import evidence_pb2 as _evidence_pb2
from google.protobuf import timestamp_pb2 as _timestamp_pb2
from google.protobuf.internal import containers as _containers
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Iterable as _Iterable, Mapping as _Mapping
from typing import ClassVar as _ClassVar, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class ReportTargetVerdictRequest(_message.Message):
    __slots__ = ("profile_id", "git_commit_hash", "verdict")
    PROFILE_ID_FIELD_NUMBER: _ClassVar[int]
    GIT_COMMIT_HASH_FIELD_NUMBER: _ClassVar[int]
    VERDICT_FIELD_NUMBER: _ClassVar[int]
    profile_id: str
    git_commit_hash: str
    verdict: _evidence_pb2.TargetVerdict
    def __init__(self, profile_id: _Optional[str] = ..., git_commit_hash: _Optional[str] = ..., verdict: _Optional[_Union[_evidence_pb2.TargetVerdict, _Mapping]] = ...) -> None: ...

class ReportTargetVerdictResponse(_message.Message):
    __slots__ = ("verdict",)
    VERDICT_FIELD_NUMBER: _ClassVar[int]
    verdict: _evidence_pb2.TargetVerdict
    def __init__(self, verdict: _Optional[_Union[_evidence_pb2.TargetVerdict, _Mapping]] = ...) -> None: ...

class ListTargetVerdictsRequest(_message.Message):
    __slots__ = ("profile_id", "git_commit_hash", "page_size", "page_token")
    PROFILE_ID_FIELD_NUMBER: _ClassVar[int]
    GIT_COMMIT_HASH_FIELD_NUMBER: _ClassVar[int]
    PAGE_SIZE_FIELD_NUMBER: _ClassVar[int]
    PAGE_TOKEN_FIELD_NUMBER: _ClassVar[int]
    profile_id: str
    git_commit_hash: str
    page_size: int
    page_token: str
    def __init__(self, profile_id: _Optional[str] = ..., git_commit_hash: _Optional[str] = ..., page_size: _Optional[int] = ..., page_token: _Optional[str] = ...) -> None: ...

class ListTargetVerdictsResponse(_message.Message):
    __slots__ = ("verdicts", "next_page_token", "count")
    VERDICTS_FIELD_NUMBER: _ClassVar[int]
    NEXT_PAGE_TOKEN_FIELD_NUMBER: _ClassVar[int]
    COUNT_FIELD_NUMBER: _ClassVar[int]
    verdicts: _containers.RepeatedCompositeFieldContainer[_evidence_pb2.TargetVerdict]
    next_page_token: str
    count: int
    def __init__(self, verdicts: _Optional[_Iterable[_Union[_evidence_pb2.TargetVerdict, _Mapping]]] = ..., next_page_token: _Optional[str] = ..., count: _Optional[int] = ...) -> None: ...

class GetEvidenceReviewRequest(_message.Message):
    __slots__ = ("profile_id", "git_commit_hash")
    PROFILE_ID_FIELD_NUMBER: _ClassVar[int]
    GIT_COMMIT_HASH_FIELD_NUMBER: _ClassVar[int]
    profile_id: str
    git_commit_hash: str
    def __init__(self, profile_id: _Optional[str] = ..., git_commit_hash: _Optional[str] = ...) -> None: ...

class EvidenceReview(_message.Message):
    __slots__ = ("profile_id", "git_commit_hash", "verdicts", "ready", "reason", "created_at")
    PROFILE_ID_FIELD_NUMBER: _ClassVar[int]
    GIT_COMMIT_HASH_FIELD_NUMBER: _ClassVar[int]
    VERDICTS_FIELD_NUMBER: _ClassVar[int]
    READY_FIELD_NUMBER: _ClassVar[int]
    REASON_FIELD_NUMBER: _ClassVar[int]
    CREATED_AT_FIELD_NUMBER: _ClassVar[int]
    profile_id: str
    git_commit_hash: str
    verdicts: _containers.RepeatedCompositeFieldContainer[_evidence_pb2.TargetVerdict]
    ready: bool
    reason: str
    created_at: _timestamp_pb2.Timestamp
    def __init__(self, profile_id: _Optional[str] = ..., git_commit_hash: _Optional[str] = ..., verdicts: _Optional[_Iterable[_Union[_evidence_pb2.TargetVerdict, _Mapping]]] = ..., ready: _Optional[bool] = ..., reason: _Optional[str] = ..., created_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ...) -> None: ...
