from google.protobuf.internal import containers as _containers
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Iterable as _Iterable, Mapping as _Mapping
from typing import ClassVar as _ClassVar, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class ReviewRun(_message.Message):
    __slots__ = ("id", "draft_id", "outcome")
    ID_FIELD_NUMBER: _ClassVar[int]
    DRAFT_ID_FIELD_NUMBER: _ClassVar[int]
    OUTCOME_FIELD_NUMBER: _ClassVar[int]
    id: str
    draft_id: str
    outcome: str
    def __init__(self, id: _Optional[str] = ..., draft_id: _Optional[str] = ..., outcome: _Optional[str] = ...) -> None: ...

class ListReviewRunsRequest(_message.Message):
    __slots__ = ()
    def __init__(self) -> None: ...

class ListReviewRunsResponse(_message.Message):
    __slots__ = ("review_runs",)
    REVIEW_RUNS_FIELD_NUMBER: _ClassVar[int]
    review_runs: _containers.RepeatedCompositeFieldContainer[ReviewRun]
    def __init__(self, review_runs: _Optional[_Iterable[_Union[ReviewRun, _Mapping]]] = ...) -> None: ...

class Verdict(_message.Message):
    __slots__ = ("mode", "passed", "evidence", "finding")
    MODE_FIELD_NUMBER: _ClassVar[int]
    PASSED_FIELD_NUMBER: _ClassVar[int]
    EVIDENCE_FIELD_NUMBER: _ClassVar[int]
    FINDING_FIELD_NUMBER: _ClassVar[int]
    mode: str
    passed: bool
    evidence: str
    finding: str
    def __init__(self, mode: _Optional[str] = ..., passed: _Optional[bool] = ..., evidence: _Optional[str] = ..., finding: _Optional[str] = ...) -> None: ...

class RecordReviewRunRequest(_message.Message):
    __slots__ = ("draft_id", "verdicts")
    DRAFT_ID_FIELD_NUMBER: _ClassVar[int]
    VERDICTS_FIELD_NUMBER: _ClassVar[int]
    draft_id: str
    verdicts: _containers.RepeatedCompositeFieldContainer[Verdict]
    def __init__(self, draft_id: _Optional[str] = ..., verdicts: _Optional[_Iterable[_Union[Verdict, _Mapping]]] = ...) -> None: ...

class RecordReviewRunResponse(_message.Message):
    __slots__ = ("review_run",)
    REVIEW_RUN_FIELD_NUMBER: _ClassVar[int]
    review_run: ReviewRun
    def __init__(self, review_run: _Optional[_Union[ReviewRun, _Mapping]] = ...) -> None: ...
