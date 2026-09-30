from google.protobuf.internal import containers as _containers
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Iterable as _Iterable, Mapping as _Mapping
from typing import ClassVar as _ClassVar, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class Objective(_message.Message):
    __slots__ = ("id", "title", "evidence_source", "has_evidence", "gap_marker", "global_order", "meaning_revision")
    ID_FIELD_NUMBER: _ClassVar[int]
    TITLE_FIELD_NUMBER: _ClassVar[int]
    CLASS_FIELD_NUMBER: _ClassVar[int]
    EVIDENCE_SOURCE_FIELD_NUMBER: _ClassVar[int]
    HAS_EVIDENCE_FIELD_NUMBER: _ClassVar[int]
    GAP_MARKER_FIELD_NUMBER: _ClassVar[int]
    GLOBAL_ORDER_FIELD_NUMBER: _ClassVar[int]
    MEANING_REVISION_FIELD_NUMBER: _ClassVar[int]
    id: str
    title: str
    evidence_source: str
    has_evidence: bool
    gap_marker: str
    global_order: int
    meaning_revision: str
    def __init__(self, id: _Optional[str] = ..., title: _Optional[str] = ..., evidence_source: _Optional[str] = ..., has_evidence: _Optional[bool] = ..., gap_marker: _Optional[str] = ..., global_order: _Optional[int] = ..., meaning_revision: _Optional[str] = ..., **kwargs) -> None: ...

class ObjectiveInput(_message.Message):
    __slots__ = ("id", "title", "evidence_source", "gap_marker")
    ID_FIELD_NUMBER: _ClassVar[int]
    TITLE_FIELD_NUMBER: _ClassVar[int]
    CLASS_FIELD_NUMBER: _ClassVar[int]
    EVIDENCE_SOURCE_FIELD_NUMBER: _ClassVar[int]
    GAP_MARKER_FIELD_NUMBER: _ClassVar[int]
    id: str
    title: str
    evidence_source: str
    gap_marker: str
    def __init__(self, id: _Optional[str] = ..., title: _Optional[str] = ..., evidence_source: _Optional[str] = ..., gap_marker: _Optional[str] = ..., **kwargs) -> None: ...

class Attachment(_message.Message):
    __slots__ = ("objective_id", "team_id", "role", "coverage", "note", "priority", "acknowledged_revision", "attachment_revision", "restatement_pending")
    OBJECTIVE_ID_FIELD_NUMBER: _ClassVar[int]
    TEAM_ID_FIELD_NUMBER: _ClassVar[int]
    ROLE_FIELD_NUMBER: _ClassVar[int]
    COVERAGE_FIELD_NUMBER: _ClassVar[int]
    NOTE_FIELD_NUMBER: _ClassVar[int]
    PRIORITY_FIELD_NUMBER: _ClassVar[int]
    ACKNOWLEDGED_REVISION_FIELD_NUMBER: _ClassVar[int]
    ATTACHMENT_REVISION_FIELD_NUMBER: _ClassVar[int]
    RESTATEMENT_PENDING_FIELD_NUMBER: _ClassVar[int]
    objective_id: str
    team_id: str
    role: str
    coverage: str
    note: str
    priority: int
    acknowledged_revision: str
    attachment_revision: str
    restatement_pending: bool
    def __init__(self, objective_id: _Optional[str] = ..., team_id: _Optional[str] = ..., role: _Optional[str] = ..., coverage: _Optional[str] = ..., note: _Optional[str] = ..., priority: _Optional[int] = ..., acknowledged_revision: _Optional[str] = ..., attachment_revision: _Optional[str] = ..., restatement_pending: _Optional[bool] = ...) -> None: ...

class AttachmentInput(_message.Message):
    __slots__ = ("objective_id", "team_id", "role", "coverage", "note")
    OBJECTIVE_ID_FIELD_NUMBER: _ClassVar[int]
    TEAM_ID_FIELD_NUMBER: _ClassVar[int]
    ROLE_FIELD_NUMBER: _ClassVar[int]
    COVERAGE_FIELD_NUMBER: _ClassVar[int]
    NOTE_FIELD_NUMBER: _ClassVar[int]
    objective_id: str
    team_id: str
    role: str
    coverage: str
    note: str
    def __init__(self, objective_id: _Optional[str] = ..., team_id: _Optional[str] = ..., role: _Optional[str] = ..., coverage: _Optional[str] = ..., note: _Optional[str] = ...) -> None: ...

class Relation(_message.Message):
    __slots__ = ("from_objective_id", "to_objective_id")
    FROM_OBJECTIVE_ID_FIELD_NUMBER: _ClassVar[int]
    TO_OBJECTIVE_ID_FIELD_NUMBER: _ClassVar[int]
    from_objective_id: str
    to_objective_id: str
    def __init__(self, from_objective_id: _Optional[str] = ..., to_objective_id: _Optional[str] = ...) -> None: ...

class Finding(_message.Message):
    __slots__ = ("rule", "severity", "objective_id", "team_id", "detail")
    RULE_FIELD_NUMBER: _ClassVar[int]
    SEVERITY_FIELD_NUMBER: _ClassVar[int]
    OBJECTIVE_ID_FIELD_NUMBER: _ClassVar[int]
    TEAM_ID_FIELD_NUMBER: _ClassVar[int]
    DETAIL_FIELD_NUMBER: _ClassVar[int]
    rule: str
    severity: str
    objective_id: str
    team_id: str
    detail: str
    def __init__(self, rule: _Optional[str] = ..., severity: _Optional[str] = ..., objective_id: _Optional[str] = ..., team_id: _Optional[str] = ..., detail: _Optional[str] = ...) -> None: ...

class ValidationResult(_message.Message):
    __slots__ = ("findings", "errors", "warnings")
    FINDINGS_FIELD_NUMBER: _ClassVar[int]
    ERRORS_FIELD_NUMBER: _ClassVar[int]
    WARNINGS_FIELD_NUMBER: _ClassVar[int]
    findings: _containers.RepeatedCompositeFieldContainer[Finding]
    errors: int
    warnings: int
    def __init__(self, findings: _Optional[_Iterable[_Union[Finding, _Mapping]]] = ..., errors: _Optional[int] = ..., warnings: _Optional[int] = ...) -> None: ...

class ListObjectivesRequest(_message.Message):
    __slots__ = ()
    def __init__(self) -> None: ...

class ListObjectivesResponse(_message.Message):
    __slots__ = ("objectives",)
    OBJECTIVES_FIELD_NUMBER: _ClassVar[int]
    objectives: _containers.RepeatedCompositeFieldContainer[Objective]
    def __init__(self, objectives: _Optional[_Iterable[_Union[Objective, _Mapping]]] = ...) -> None: ...

class GetObjectiveRequest(_message.Message):
    __slots__ = ("id",)
    ID_FIELD_NUMBER: _ClassVar[int]
    id: str
    def __init__(self, id: _Optional[str] = ...) -> None: ...

class UpsertObjectiveRequest(_message.Message):
    __slots__ = ("objective", "expected_meaning_revision")
    OBJECTIVE_FIELD_NUMBER: _ClassVar[int]
    EXPECTED_MEANING_REVISION_FIELD_NUMBER: _ClassVar[int]
    objective: ObjectiveInput
    expected_meaning_revision: str
    def __init__(self, objective: _Optional[_Union[ObjectiveInput, _Mapping]] = ..., expected_meaning_revision: _Optional[str] = ...) -> None: ...

class DeleteObjectiveRequest(_message.Message):
    __slots__ = ("id", "expected_meaning_revision")
    ID_FIELD_NUMBER: _ClassVar[int]
    EXPECTED_MEANING_REVISION_FIELD_NUMBER: _ClassVar[int]
    id: str
    expected_meaning_revision: str
    def __init__(self, id: _Optional[str] = ..., expected_meaning_revision: _Optional[str] = ...) -> None: ...

class DeleteObjectiveResponse(_message.Message):
    __slots__ = ()
    def __init__(self) -> None: ...

class ReorderObjectivesRequest(_message.Message):
    __slots__ = ("objective_ids",)
    OBJECTIVE_IDS_FIELD_NUMBER: _ClassVar[int]
    objective_ids: _containers.RepeatedScalarFieldContainer[str]
    def __init__(self, objective_ids: _Optional[_Iterable[str]] = ...) -> None: ...

class ListTeamAttachmentsRequest(_message.Message):
    __slots__ = ("team_id",)
    TEAM_ID_FIELD_NUMBER: _ClassVar[int]
    team_id: str
    def __init__(self, team_id: _Optional[str] = ...) -> None: ...

class ListTeamAttachmentsResponse(_message.Message):
    __slots__ = ("team_id", "attachment_revision", "attachments")
    TEAM_ID_FIELD_NUMBER: _ClassVar[int]
    ATTACHMENT_REVISION_FIELD_NUMBER: _ClassVar[int]
    ATTACHMENTS_FIELD_NUMBER: _ClassVar[int]
    team_id: str
    attachment_revision: str
    attachments: _containers.RepeatedCompositeFieldContainer[Attachment]
    def __init__(self, team_id: _Optional[str] = ..., attachment_revision: _Optional[str] = ..., attachments: _Optional[_Iterable[_Union[Attachment, _Mapping]]] = ...) -> None: ...

class GetTeamAttachmentRevisionRequest(_message.Message):
    __slots__ = ("team_id",)
    TEAM_ID_FIELD_NUMBER: _ClassVar[int]
    team_id: str
    def __init__(self, team_id: _Optional[str] = ...) -> None: ...

class GetTeamAttachmentRevisionResponse(_message.Message):
    __slots__ = ("team_id", "attachment_revision")
    TEAM_ID_FIELD_NUMBER: _ClassVar[int]
    ATTACHMENT_REVISION_FIELD_NUMBER: _ClassVar[int]
    team_id: str
    attachment_revision: str
    def __init__(self, team_id: _Optional[str] = ..., attachment_revision: _Optional[str] = ...) -> None: ...

class AttachObjectiveRequest(_message.Message):
    __slots__ = ("attachment", "expected_team_revision")
    ATTACHMENT_FIELD_NUMBER: _ClassVar[int]
    EXPECTED_TEAM_REVISION_FIELD_NUMBER: _ClassVar[int]
    attachment: AttachmentInput
    expected_team_revision: str
    def __init__(self, attachment: _Optional[_Union[AttachmentInput, _Mapping]] = ..., expected_team_revision: _Optional[str] = ...) -> None: ...

class UpdateAttachmentRequest(_message.Message):
    __slots__ = ("attachment", "expected_team_revision")
    ATTACHMENT_FIELD_NUMBER: _ClassVar[int]
    EXPECTED_TEAM_REVISION_FIELD_NUMBER: _ClassVar[int]
    attachment: AttachmentInput
    expected_team_revision: str
    def __init__(self, attachment: _Optional[_Union[AttachmentInput, _Mapping]] = ..., expected_team_revision: _Optional[str] = ...) -> None: ...

class DetachObjectiveRequest(_message.Message):
    __slots__ = ("objective_id", "team_id", "expected_team_revision")
    OBJECTIVE_ID_FIELD_NUMBER: _ClassVar[int]
    TEAM_ID_FIELD_NUMBER: _ClassVar[int]
    EXPECTED_TEAM_REVISION_FIELD_NUMBER: _ClassVar[int]
    objective_id: str
    team_id: str
    expected_team_revision: str
    def __init__(self, objective_id: _Optional[str] = ..., team_id: _Optional[str] = ..., expected_team_revision: _Optional[str] = ...) -> None: ...

class DetachObjectiveResponse(_message.Message):
    __slots__ = ()
    def __init__(self) -> None: ...

class ReorderTeamAttachmentsRequest(_message.Message):
    __slots__ = ("team_id", "objective_ids", "expected_team_revision")
    TEAM_ID_FIELD_NUMBER: _ClassVar[int]
    OBJECTIVE_IDS_FIELD_NUMBER: _ClassVar[int]
    EXPECTED_TEAM_REVISION_FIELD_NUMBER: _ClassVar[int]
    team_id: str
    objective_ids: _containers.RepeatedScalarFieldContainer[str]
    expected_team_revision: str
    def __init__(self, team_id: _Optional[str] = ..., objective_ids: _Optional[_Iterable[str]] = ..., expected_team_revision: _Optional[str] = ...) -> None: ...

class AcknowledgeObjectiveRequest(_message.Message):
    __slots__ = ("objective_id", "team_id", "revision")
    OBJECTIVE_ID_FIELD_NUMBER: _ClassVar[int]
    TEAM_ID_FIELD_NUMBER: _ClassVar[int]
    REVISION_FIELD_NUMBER: _ClassVar[int]
    objective_id: str
    team_id: str
    revision: str
    def __init__(self, objective_id: _Optional[str] = ..., team_id: _Optional[str] = ..., revision: _Optional[str] = ...) -> None: ...

class ListRelationsRequest(_message.Message):
    __slots__ = ()
    def __init__(self) -> None: ...

class ListRelationsResponse(_message.Message):
    __slots__ = ("relations",)
    RELATIONS_FIELD_NUMBER: _ClassVar[int]
    relations: _containers.RepeatedCompositeFieldContainer[Relation]
    def __init__(self, relations: _Optional[_Iterable[_Union[Relation, _Mapping]]] = ...) -> None: ...

class AddRelationRequest(_message.Message):
    __slots__ = ("from_objective_id", "to_objective_id")
    FROM_OBJECTIVE_ID_FIELD_NUMBER: _ClassVar[int]
    TO_OBJECTIVE_ID_FIELD_NUMBER: _ClassVar[int]
    from_objective_id: str
    to_objective_id: str
    def __init__(self, from_objective_id: _Optional[str] = ..., to_objective_id: _Optional[str] = ...) -> None: ...

class AddRelationResponse(_message.Message):
    __slots__ = ()
    def __init__(self) -> None: ...

class DeleteRelationRequest(_message.Message):
    __slots__ = ("from_objective_id", "to_objective_id")
    FROM_OBJECTIVE_ID_FIELD_NUMBER: _ClassVar[int]
    TO_OBJECTIVE_ID_FIELD_NUMBER: _ClassVar[int]
    from_objective_id: str
    to_objective_id: str
    def __init__(self, from_objective_id: _Optional[str] = ..., to_objective_id: _Optional[str] = ...) -> None: ...

class DeleteRelationResponse(_message.Message):
    __slots__ = ()
    def __init__(self) -> None: ...

class ValidateObjectivesRequest(_message.Message):
    __slots__ = ()
    def __init__(self) -> None: ...
