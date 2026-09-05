from google.protobuf.internal import containers as _containers
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Iterable as _Iterable
from typing import ClassVar as _ClassVar, Optional as _Optional

DESCRIPTOR: _descriptor.FileDescriptor

class LibraryProgram(_message.Message):
    __slots__ = ("id", "name", "version", "source", "description", "origin", "created_at", "source_program_id", "promoted_by", "promotion_reason", "current", "called_binding_ids", "tier", "declared_inputs", "declared_outputs", "coverage", "validated_at", "kind", "scenario", "purpose", "rung", "owner_skill", "validation_error", "path", "score")
    ID_FIELD_NUMBER: _ClassVar[int]
    NAME_FIELD_NUMBER: _ClassVar[int]
    VERSION_FIELD_NUMBER: _ClassVar[int]
    SOURCE_FIELD_NUMBER: _ClassVar[int]
    DESCRIPTION_FIELD_NUMBER: _ClassVar[int]
    ORIGIN_FIELD_NUMBER: _ClassVar[int]
    CREATED_AT_FIELD_NUMBER: _ClassVar[int]
    SOURCE_PROGRAM_ID_FIELD_NUMBER: _ClassVar[int]
    PROMOTED_BY_FIELD_NUMBER: _ClassVar[int]
    PROMOTION_REASON_FIELD_NUMBER: _ClassVar[int]
    CURRENT_FIELD_NUMBER: _ClassVar[int]
    CALLED_BINDING_IDS_FIELD_NUMBER: _ClassVar[int]
    TIER_FIELD_NUMBER: _ClassVar[int]
    DECLARED_INPUTS_FIELD_NUMBER: _ClassVar[int]
    DECLARED_OUTPUTS_FIELD_NUMBER: _ClassVar[int]
    COVERAGE_FIELD_NUMBER: _ClassVar[int]
    VALIDATED_AT_FIELD_NUMBER: _ClassVar[int]
    KIND_FIELD_NUMBER: _ClassVar[int]
    SCENARIO_FIELD_NUMBER: _ClassVar[int]
    PURPOSE_FIELD_NUMBER: _ClassVar[int]
    RUNG_FIELD_NUMBER: _ClassVar[int]
    OWNER_SKILL_FIELD_NUMBER: _ClassVar[int]
    VALIDATION_ERROR_FIELD_NUMBER: _ClassVar[int]
    PATH_FIELD_NUMBER: _ClassVar[int]
    SCORE_FIELD_NUMBER: _ClassVar[int]
    id: str
    name: str
    version: int
    source: str
    description: str
    origin: str
    created_at: str
    source_program_id: str
    promoted_by: str
    promotion_reason: str
    current: bool
    called_binding_ids: _containers.RepeatedScalarFieldContainer[str]
    tier: str
    declared_inputs: _containers.RepeatedScalarFieldContainer[str]
    declared_outputs: _containers.RepeatedScalarFieldContainer[str]
    coverage: str
    validated_at: str
    kind: str
    scenario: str
    purpose: str
    rung: str
    owner_skill: str
    validation_error: str
    path: str
    score: float
    def __init__(self, id: _Optional[str] = ..., name: _Optional[str] = ..., version: _Optional[int] = ..., source: _Optional[str] = ..., description: _Optional[str] = ..., origin: _Optional[str] = ..., created_at: _Optional[str] = ..., source_program_id: _Optional[str] = ..., promoted_by: _Optional[str] = ..., promotion_reason: _Optional[str] = ..., current: _Optional[bool] = ..., called_binding_ids: _Optional[_Iterable[str]] = ..., tier: _Optional[str] = ..., declared_inputs: _Optional[_Iterable[str]] = ..., declared_outputs: _Optional[_Iterable[str]] = ..., coverage: _Optional[str] = ..., validated_at: _Optional[str] = ..., kind: _Optional[str] = ..., scenario: _Optional[str] = ..., purpose: _Optional[str] = ..., rung: _Optional[str] = ..., owner_skill: _Optional[str] = ..., validation_error: _Optional[str] = ..., path: _Optional[str] = ..., score: _Optional[float] = ...) -> None: ...
