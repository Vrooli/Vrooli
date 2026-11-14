from google.protobuf.internal import containers as _containers
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Iterable as _Iterable, Mapping as _Mapping
from typing import ClassVar as _ClassVar, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class Candidate(_message.Message):
    __slots__ = ("id", "description", "status", "score", "evidence", "approval_required")
    ID_FIELD_NUMBER: _ClassVar[int]
    DESCRIPTION_FIELD_NUMBER: _ClassVar[int]
    STATUS_FIELD_NUMBER: _ClassVar[int]
    SCORE_FIELD_NUMBER: _ClassVar[int]
    EVIDENCE_FIELD_NUMBER: _ClassVar[int]
    APPROVAL_REQUIRED_FIELD_NUMBER: _ClassVar[int]
    id: str
    description: str
    status: str
    score: float
    evidence: _containers.RepeatedScalarFieldContainer[str]
    approval_required: bool
    def __init__(self, id: _Optional[str] = ..., description: _Optional[str] = ..., status: _Optional[str] = ..., score: _Optional[float] = ..., evidence: _Optional[_Iterable[str]] = ..., approval_required: _Optional[bool] = ...) -> None: ...

class OptimizationRun(_message.Message):
    __slots__ = ("id", "status", "scoring_profile", "candidates", "recommendation")
    ID_FIELD_NUMBER: _ClassVar[int]
    STATUS_FIELD_NUMBER: _ClassVar[int]
    SCORING_PROFILE_FIELD_NUMBER: _ClassVar[int]
    CANDIDATES_FIELD_NUMBER: _ClassVar[int]
    RECOMMENDATION_FIELD_NUMBER: _ClassVar[int]
    id: str
    status: str
    scoring_profile: str
    candidates: _containers.RepeatedCompositeFieldContainer[Candidate]
    recommendation: str
    def __init__(self, id: _Optional[str] = ..., status: _Optional[str] = ..., scoring_profile: _Optional[str] = ..., candidates: _Optional[_Iterable[_Union[Candidate, _Mapping]]] = ..., recommendation: _Optional[str] = ...) -> None: ...

class CreateOptimizationRunRequest(_message.Message):
    __slots__ = ("scoring_profile", "dry_run")
    SCORING_PROFILE_FIELD_NUMBER: _ClassVar[int]
    DRY_RUN_FIELD_NUMBER: _ClassVar[int]
    scoring_profile: str
    dry_run: bool
    def __init__(self, scoring_profile: _Optional[str] = ..., dry_run: _Optional[bool] = ...) -> None: ...

class CreateOptimizationRunResponse(_message.Message):
    __slots__ = ("run",)
    RUN_FIELD_NUMBER: _ClassVar[int]
    run: OptimizationRun
    def __init__(self, run: _Optional[_Union[OptimizationRun, _Mapping]] = ...) -> None: ...

class RunCandidateRequest(_message.Message):
    __slots__ = ("run_id", "candidate_id")
    RUN_ID_FIELD_NUMBER: _ClassVar[int]
    CANDIDATE_ID_FIELD_NUMBER: _ClassVar[int]
    run_id: str
    candidate_id: str
    def __init__(self, run_id: _Optional[str] = ..., candidate_id: _Optional[str] = ...) -> None: ...

class RunCandidateResponse(_message.Message):
    __slots__ = ("run",)
    RUN_FIELD_NUMBER: _ClassVar[int]
    run: OptimizationRun
    def __init__(self, run: _Optional[_Union[OptimizationRun, _Mapping]] = ...) -> None: ...

class ScoreCandidatesRequest(_message.Message):
    __slots__ = ("run_id",)
    RUN_ID_FIELD_NUMBER: _ClassVar[int]
    run_id: str
    def __init__(self, run_id: _Optional[str] = ...) -> None: ...

class ScoreCandidatesResponse(_message.Message):
    __slots__ = ("run",)
    RUN_FIELD_NUMBER: _ClassVar[int]
    run: OptimizationRun
    def __init__(self, run: _Optional[_Union[OptimizationRun, _Mapping]] = ...) -> None: ...

class ApproveCandidateRequest(_message.Message):
    __slots__ = ("run_id", "candidate_id", "approved")
    RUN_ID_FIELD_NUMBER: _ClassVar[int]
    CANDIDATE_ID_FIELD_NUMBER: _ClassVar[int]
    APPROVED_FIELD_NUMBER: _ClassVar[int]
    run_id: str
    candidate_id: str
    approved: bool
    def __init__(self, run_id: _Optional[str] = ..., candidate_id: _Optional[str] = ..., approved: _Optional[bool] = ...) -> None: ...

class ApproveCandidateResponse(_message.Message):
    __slots__ = ("run",)
    RUN_FIELD_NUMBER: _ClassVar[int]
    run: OptimizationRun
    def __init__(self, run: _Optional[_Union[OptimizationRun, _Mapping]] = ...) -> None: ...

class RollbackOptimizationRequest(_message.Message):
    __slots__ = ("run_id",)
    RUN_ID_FIELD_NUMBER: _ClassVar[int]
    run_id: str
    def __init__(self, run_id: _Optional[str] = ...) -> None: ...

class RollbackOptimizationResponse(_message.Message):
    __slots__ = ("run",)
    RUN_FIELD_NUMBER: _ClassVar[int]
    run: OptimizationRun
    def __init__(self, run: _Optional[_Union[OptimizationRun, _Mapping]] = ...) -> None: ...
