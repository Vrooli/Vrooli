from backdrop_studio.v1.shared import shared_pb2 as _shared_pb2
from google.protobuf.internal import containers as _containers
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Iterable as _Iterable, Mapping as _Mapping
from typing import ClassVar as _ClassVar, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class SubmitRequest(_message.Message):
    __slots__ = ("style", "placement", "seed", "candidate_count", "brand_tokens", "brand_id", "surface_id")
    class BrandTokensEntry(_message.Message):
        __slots__ = ("key", "value")
        KEY_FIELD_NUMBER: _ClassVar[int]
        VALUE_FIELD_NUMBER: _ClassVar[int]
        key: str
        value: str
        def __init__(self, key: _Optional[str] = ..., value: _Optional[str] = ...) -> None: ...
    STYLE_FIELD_NUMBER: _ClassVar[int]
    PLACEMENT_FIELD_NUMBER: _ClassVar[int]
    SEED_FIELD_NUMBER: _ClassVar[int]
    CANDIDATE_COUNT_FIELD_NUMBER: _ClassVar[int]
    BRAND_TOKENS_FIELD_NUMBER: _ClassVar[int]
    BRAND_ID_FIELD_NUMBER: _ClassVar[int]
    SURFACE_ID_FIELD_NUMBER: _ClassVar[int]
    style: _shared_pb2.Style
    placement: str
    seed: int
    candidate_count: int
    brand_tokens: _containers.ScalarMap[str, str]
    brand_id: str
    surface_id: str
    def __init__(self, style: _Optional[_Union[_shared_pb2.Style, _Mapping]] = ..., placement: _Optional[str] = ..., seed: _Optional[int] = ..., candidate_count: _Optional[int] = ..., brand_tokens: _Optional[_Mapping[str, str]] = ..., brand_id: _Optional[str] = ..., surface_id: _Optional[str] = ...) -> None: ...

class GetJobRequest(_message.Message):
    __slots__ = ("job_id",)
    JOB_ID_FIELD_NUMBER: _ClassVar[int]
    job_id: str
    def __init__(self, job_id: _Optional[str] = ...) -> None: ...

class ListCandidatesRequest(_message.Message):
    __slots__ = ("job_id",)
    JOB_ID_FIELD_NUMBER: _ClassVar[int]
    job_id: str
    def __init__(self, job_id: _Optional[str] = ...) -> None: ...

class ListCandidatesResponse(_message.Message):
    __slots__ = ("candidates",)
    CANDIDATES_FIELD_NUMBER: _ClassVar[int]
    candidates: _containers.RepeatedCompositeFieldContainer[Candidate]
    def __init__(self, candidates: _Optional[_Iterable[_Union[Candidate, _Mapping]]] = ...) -> None: ...

class SelectCandidateRequest(_message.Message):
    __slots__ = ("job_id", "candidate_id", "actor")
    JOB_ID_FIELD_NUMBER: _ClassVar[int]
    CANDIDATE_ID_FIELD_NUMBER: _ClassVar[int]
    ACTOR_FIELD_NUMBER: _ClassVar[int]
    job_id: str
    candidate_id: str
    actor: str
    def __init__(self, job_id: _Optional[str] = ..., candidate_id: _Optional[str] = ..., actor: _Optional[str] = ...) -> None: ...

class RenderJob(_message.Message):
    __slots__ = ("id", "style_id", "status", "seed", "execution_path", "candidates", "selected_candidate_id", "selected_by", "surface_id")
    ID_FIELD_NUMBER: _ClassVar[int]
    STYLE_ID_FIELD_NUMBER: _ClassVar[int]
    STATUS_FIELD_NUMBER: _ClassVar[int]
    SEED_FIELD_NUMBER: _ClassVar[int]
    EXECUTION_PATH_FIELD_NUMBER: _ClassVar[int]
    CANDIDATES_FIELD_NUMBER: _ClassVar[int]
    SELECTED_CANDIDATE_ID_FIELD_NUMBER: _ClassVar[int]
    SELECTED_BY_FIELD_NUMBER: _ClassVar[int]
    SURFACE_ID_FIELD_NUMBER: _ClassVar[int]
    id: str
    style_id: str
    status: str
    seed: int
    execution_path: str
    candidates: _containers.RepeatedCompositeFieldContainer[Candidate]
    selected_candidate_id: str
    selected_by: str
    surface_id: str
    def __init__(self, id: _Optional[str] = ..., style_id: _Optional[str] = ..., status: _Optional[str] = ..., seed: _Optional[int] = ..., execution_path: _Optional[str] = ..., candidates: _Optional[_Iterable[_Union[Candidate, _Mapping]]] = ..., selected_candidate_id: _Optional[str] = ..., selected_by: _Optional[str] = ..., surface_id: _Optional[str] = ...) -> None: ...

class Candidate(_message.Message):
    __slots__ = ("id", "job_id", "image_png", "width", "height", "strategy", "execution_path", "treatment_applied", "seed", "conditioning_submitted", "disclosure_required", "prompt", "provenance_json", "quality_json", "routing", "plates")
    ID_FIELD_NUMBER: _ClassVar[int]
    JOB_ID_FIELD_NUMBER: _ClassVar[int]
    IMAGE_PNG_FIELD_NUMBER: _ClassVar[int]
    WIDTH_FIELD_NUMBER: _ClassVar[int]
    HEIGHT_FIELD_NUMBER: _ClassVar[int]
    STRATEGY_FIELD_NUMBER: _ClassVar[int]
    EXECUTION_PATH_FIELD_NUMBER: _ClassVar[int]
    TREATMENT_APPLIED_FIELD_NUMBER: _ClassVar[int]
    SEED_FIELD_NUMBER: _ClassVar[int]
    CONDITIONING_SUBMITTED_FIELD_NUMBER: _ClassVar[int]
    DISCLOSURE_REQUIRED_FIELD_NUMBER: _ClassVar[int]
    PROMPT_FIELD_NUMBER: _ClassVar[int]
    PROVENANCE_JSON_FIELD_NUMBER: _ClassVar[int]
    QUALITY_JSON_FIELD_NUMBER: _ClassVar[int]
    ROUTING_FIELD_NUMBER: _ClassVar[int]
    PLATES_FIELD_NUMBER: _ClassVar[int]
    id: str
    job_id: str
    image_png: bytes
    width: int
    height: int
    strategy: str
    execution_path: str
    treatment_applied: bool
    seed: int
    conditioning_submitted: bool
    disclosure_required: bool
    prompt: str
    provenance_json: str
    quality_json: str
    routing: _shared_pb2.RoutingRecord
    plates: _containers.RepeatedCompositeFieldContainer[_shared_pb2.Plate]
    def __init__(self, id: _Optional[str] = ..., job_id: _Optional[str] = ..., image_png: _Optional[bytes] = ..., width: _Optional[int] = ..., height: _Optional[int] = ..., strategy: _Optional[str] = ..., execution_path: _Optional[str] = ..., treatment_applied: _Optional[bool] = ..., seed: _Optional[int] = ..., conditioning_submitted: _Optional[bool] = ..., disclosure_required: _Optional[bool] = ..., prompt: _Optional[str] = ..., provenance_json: _Optional[str] = ..., quality_json: _Optional[str] = ..., routing: _Optional[_Union[_shared_pb2.RoutingRecord, _Mapping]] = ..., plates: _Optional[_Iterable[_Union[_shared_pb2.Plate, _Mapping]]] = ...) -> None: ...
