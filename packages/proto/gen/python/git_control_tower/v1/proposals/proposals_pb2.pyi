import datetime

from google.protobuf import timestamp_pb2 as _timestamp_pb2
from google.protobuf.internal import containers as _containers
from google.protobuf.internal import enum_type_wrapper as _enum_type_wrapper
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Iterable as _Iterable, Mapping as _Mapping
from typing import ClassVar as _ClassVar, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class ProposalState(int, metaclass=_enum_type_wrapper.EnumTypeWrapper):
    __slots__ = ()
    PROPOSAL_STATE_UNSPECIFIED: _ClassVar[ProposalState]
    PROPOSAL_STATE_OPEN: _ClassVar[ProposalState]
    PROPOSAL_STATE_COMMITTED: _ClassVar[ProposalState]
    PROPOSAL_STATE_WITHDRAWN: _ClassVar[ProposalState]
    PROPOSAL_STATE_SUPERSEDED: _ClassVar[ProposalState]

class FileChangeKind(int, metaclass=_enum_type_wrapper.EnumTypeWrapper):
    __slots__ = ()
    FILE_CHANGE_KIND_UNSPECIFIED: _ClassVar[FileChangeKind]
    FILE_CHANGE_KIND_ADDED: _ClassVar[FileChangeKind]
    FILE_CHANGE_KIND_MODIFIED: _ClassVar[FileChangeKind]
    FILE_CHANGE_KIND_DELETED: _ClassVar[FileChangeKind]

class FreshnessState(int, metaclass=_enum_type_wrapper.EnumTypeWrapper):
    __slots__ = ()
    FRESHNESS_STATE_UNSPECIFIED: _ClassVar[FreshnessState]
    FRESHNESS_STATE_FRESH: _ClassVar[FreshnessState]
    FRESHNESS_STATE_DRIFTED: _ClassVar[FreshnessState]
    FRESHNESS_STATE_BASE_MOVED: _ClassVar[FreshnessState]
    FRESHNESS_STATE_UNKNOWN: _ClassVar[FreshnessState]
    FRESHNESS_STATE_NOT_APPLICABLE: _ClassVar[FreshnessState]

class TrailerResolution(int, metaclass=_enum_type_wrapper.EnumTypeWrapper):
    __slots__ = ()
    TRAILER_RESOLUTION_UNSPECIFIED: _ClassVar[TrailerResolution]
    TRAILER_RESOLUTION_RESOLVED: _ClassVar[TrailerResolution]
    TRAILER_RESOLUTION_UNRESOLVED: _ClassVar[TrailerResolution]
    TRAILER_RESOLUTION_AMBIGUOUS: _ClassVar[TrailerResolution]
    TRAILER_RESOLUTION_INACCESSIBLE: _ClassVar[TrailerResolution]
    TRAILER_RESOLUTION_DELETED: _ClassVar[TrailerResolution]
    TRAILER_RESOLUTION_LEGACY: _ClassVar[TrailerResolution]
    TRAILER_RESOLUTION_NOT_APPLICABLE: _ClassVar[TrailerResolution]

class IssueSeverity(int, metaclass=_enum_type_wrapper.EnumTypeWrapper):
    __slots__ = ()
    ISSUE_SEVERITY_UNSPECIFIED: _ClassVar[IssueSeverity]
    ISSUE_SEVERITY_ERROR: _ClassVar[IssueSeverity]
    ISSUE_SEVERITY_WARNING: _ClassVar[IssueSeverity]
PROPOSAL_STATE_UNSPECIFIED: ProposalState
PROPOSAL_STATE_OPEN: ProposalState
PROPOSAL_STATE_COMMITTED: ProposalState
PROPOSAL_STATE_WITHDRAWN: ProposalState
PROPOSAL_STATE_SUPERSEDED: ProposalState
FILE_CHANGE_KIND_UNSPECIFIED: FileChangeKind
FILE_CHANGE_KIND_ADDED: FileChangeKind
FILE_CHANGE_KIND_MODIFIED: FileChangeKind
FILE_CHANGE_KIND_DELETED: FileChangeKind
FRESHNESS_STATE_UNSPECIFIED: FreshnessState
FRESHNESS_STATE_FRESH: FreshnessState
FRESHNESS_STATE_DRIFTED: FreshnessState
FRESHNESS_STATE_BASE_MOVED: FreshnessState
FRESHNESS_STATE_UNKNOWN: FreshnessState
FRESHNESS_STATE_NOT_APPLICABLE: FreshnessState
TRAILER_RESOLUTION_UNSPECIFIED: TrailerResolution
TRAILER_RESOLUTION_RESOLVED: TrailerResolution
TRAILER_RESOLUTION_UNRESOLVED: TrailerResolution
TRAILER_RESOLUTION_AMBIGUOUS: TrailerResolution
TRAILER_RESOLUTION_INACCESSIBLE: TrailerResolution
TRAILER_RESOLUTION_DELETED: TrailerResolution
TRAILER_RESOLUTION_LEGACY: TrailerResolution
TRAILER_RESOLUTION_NOT_APPLICABLE: TrailerResolution
ISSUE_SEVERITY_UNSPECIFIED: IssueSeverity
ISSUE_SEVERITY_ERROR: IssueSeverity
ISSUE_SEVERITY_WARNING: IssueSeverity

class Actor(_message.Message):
    __slots__ = ("subject", "kind", "verified", "run_id")
    SUBJECT_FIELD_NUMBER: _ClassVar[int]
    KIND_FIELD_NUMBER: _ClassVar[int]
    VERIFIED_FIELD_NUMBER: _ClassVar[int]
    RUN_ID_FIELD_NUMBER: _ClassVar[int]
    subject: str
    kind: str
    verified: bool
    run_id: str
    def __init__(self, subject: _Optional[str] = ..., kind: _Optional[str] = ..., verified: _Optional[bool] = ..., run_id: _Optional[str] = ...) -> None: ...

class ProposalWork(_message.Message):
    __slots__ = ("effort_ref", "effort_revision", "epoch", "run_ids", "plans")
    EFFORT_REF_FIELD_NUMBER: _ClassVar[int]
    EFFORT_REVISION_FIELD_NUMBER: _ClassVar[int]
    EPOCH_FIELD_NUMBER: _ClassVar[int]
    RUN_IDS_FIELD_NUMBER: _ClassVar[int]
    PLANS_FIELD_NUMBER: _ClassVar[int]
    effort_ref: str
    effort_revision: str
    epoch: str
    run_ids: _containers.RepeatedScalarFieldContainer[str]
    plans: _containers.RepeatedScalarFieldContainer[str]
    def __init__(self, effort_ref: _Optional[str] = ..., effort_revision: _Optional[str] = ..., epoch: _Optional[str] = ..., run_ids: _Optional[_Iterable[str]] = ..., plans: _Optional[_Iterable[str]] = ...) -> None: ...

class ProposalTrailer(_message.Message):
    __slots__ = ("key", "value", "kind", "resolution", "detail", "known")
    KEY_FIELD_NUMBER: _ClassVar[int]
    VALUE_FIELD_NUMBER: _ClassVar[int]
    KIND_FIELD_NUMBER: _ClassVar[int]
    RESOLUTION_FIELD_NUMBER: _ClassVar[int]
    DETAIL_FIELD_NUMBER: _ClassVar[int]
    KNOWN_FIELD_NUMBER: _ClassVar[int]
    key: str
    value: str
    kind: str
    resolution: TrailerResolution
    detail: str
    known: bool
    def __init__(self, key: _Optional[str] = ..., value: _Optional[str] = ..., kind: _Optional[str] = ..., resolution: _Optional[_Union[TrailerResolution, str]] = ..., detail: _Optional[str] = ..., known: _Optional[bool] = ...) -> None: ...

class TrailerIssue(_message.Message):
    __slots__ = ("index", "key", "code", "message", "severity")
    INDEX_FIELD_NUMBER: _ClassVar[int]
    KEY_FIELD_NUMBER: _ClassVar[int]
    CODE_FIELD_NUMBER: _ClassVar[int]
    MESSAGE_FIELD_NUMBER: _ClassVar[int]
    SEVERITY_FIELD_NUMBER: _ClassVar[int]
    index: int
    key: str
    code: str
    message: str
    severity: IssueSeverity
    def __init__(self, index: _Optional[int] = ..., key: _Optional[str] = ..., code: _Optional[str] = ..., message: _Optional[str] = ..., severity: _Optional[_Union[IssueSeverity, str]] = ...) -> None: ...

class ProposalMessage(_message.Message):
    __slots__ = ("subject", "body", "trailers", "rendered", "operator_edited")
    SUBJECT_FIELD_NUMBER: _ClassVar[int]
    BODY_FIELD_NUMBER: _ClassVar[int]
    TRAILERS_FIELD_NUMBER: _ClassVar[int]
    RENDERED_FIELD_NUMBER: _ClassVar[int]
    OPERATOR_EDITED_FIELD_NUMBER: _ClassVar[int]
    subject: str
    body: str
    trailers: _containers.RepeatedCompositeFieldContainer[ProposalTrailer]
    rendered: str
    operator_edited: bool
    def __init__(self, subject: _Optional[str] = ..., body: _Optional[str] = ..., trailers: _Optional[_Iterable[_Union[ProposalTrailer, _Mapping]]] = ..., rendered: _Optional[str] = ..., operator_edited: _Optional[bool] = ...) -> None: ...

class ProposalFlag(_message.Message):
    __slots__ = ("code", "detail")
    CODE_FIELD_NUMBER: _ClassVar[int]
    DETAIL_FIELD_NUMBER: _ClassVar[int]
    code: str
    detail: str
    def __init__(self, code: _Optional[str] = ..., detail: _Optional[str] = ...) -> None: ...

class ProposalFile(_message.Message):
    __slots__ = ("path", "kind", "sha256", "blob_id", "deleted", "source", "flags", "current_blob_id", "drifted")
    PATH_FIELD_NUMBER: _ClassVar[int]
    KIND_FIELD_NUMBER: _ClassVar[int]
    SHA256_FIELD_NUMBER: _ClassVar[int]
    BLOB_ID_FIELD_NUMBER: _ClassVar[int]
    DELETED_FIELD_NUMBER: _ClassVar[int]
    SOURCE_FIELD_NUMBER: _ClassVar[int]
    FLAGS_FIELD_NUMBER: _ClassVar[int]
    CURRENT_BLOB_ID_FIELD_NUMBER: _ClassVar[int]
    DRIFTED_FIELD_NUMBER: _ClassVar[int]
    path: str
    kind: FileChangeKind
    sha256: str
    blob_id: str
    deleted: bool
    source: str
    flags: _containers.RepeatedCompositeFieldContainer[ProposalFlag]
    current_blob_id: str
    drifted: bool
    def __init__(self, path: _Optional[str] = ..., kind: _Optional[_Union[FileChangeKind, str]] = ..., sha256: _Optional[str] = ..., blob_id: _Optional[str] = ..., deleted: _Optional[bool] = ..., source: _Optional[str] = ..., flags: _Optional[_Iterable[_Union[ProposalFlag, _Mapping]]] = ..., current_blob_id: _Optional[str] = ..., drifted: _Optional[bool] = ...) -> None: ...

class ProposalExclusion(_message.Message):
    __slots__ = ("path", "reason", "detail")
    PATH_FIELD_NUMBER: _ClassVar[int]
    REASON_FIELD_NUMBER: _ClassVar[int]
    DETAIL_FIELD_NUMBER: _ClassVar[int]
    path: str
    reason: str
    detail: str
    def __init__(self, path: _Optional[str] = ..., reason: _Optional[str] = ..., detail: _Optional[str] = ...) -> None: ...

class ProposalEvidence(_message.Message):
    __slots__ = ("anchor_id", "epoch_file", "accepted_line_sha256", "gate_receipts", "metric_before", "metric_after")
    ANCHOR_ID_FIELD_NUMBER: _ClassVar[int]
    EPOCH_FILE_FIELD_NUMBER: _ClassVar[int]
    ACCEPTED_LINE_SHA256_FIELD_NUMBER: _ClassVar[int]
    GATE_RECEIPTS_FIELD_NUMBER: _ClassVar[int]
    METRIC_BEFORE_FIELD_NUMBER: _ClassVar[int]
    METRIC_AFTER_FIELD_NUMBER: _ClassVar[int]
    anchor_id: str
    epoch_file: str
    accepted_line_sha256: str
    gate_receipts: _containers.RepeatedScalarFieldContainer[str]
    metric_before: str
    metric_after: str
    def __init__(self, anchor_id: _Optional[str] = ..., epoch_file: _Optional[str] = ..., accepted_line_sha256: _Optional[str] = ..., gate_receipts: _Optional[_Iterable[str]] = ..., metric_before: _Optional[str] = ..., metric_after: _Optional[str] = ...) -> None: ...

class ProposalFreshness(_message.Message):
    __slots__ = ("state", "drifted_paths", "base_changed_paths", "foreign_staged_paths", "current_head", "detail", "checked_at")
    STATE_FIELD_NUMBER: _ClassVar[int]
    DRIFTED_PATHS_FIELD_NUMBER: _ClassVar[int]
    BASE_CHANGED_PATHS_FIELD_NUMBER: _ClassVar[int]
    FOREIGN_STAGED_PATHS_FIELD_NUMBER: _ClassVar[int]
    CURRENT_HEAD_FIELD_NUMBER: _ClassVar[int]
    DETAIL_FIELD_NUMBER: _ClassVar[int]
    CHECKED_AT_FIELD_NUMBER: _ClassVar[int]
    state: FreshnessState
    drifted_paths: _containers.RepeatedScalarFieldContainer[str]
    base_changed_paths: _containers.RepeatedScalarFieldContainer[str]
    foreign_staged_paths: _containers.RepeatedScalarFieldContainer[str]
    current_head: str
    detail: str
    checked_at: _timestamp_pb2.Timestamp
    def __init__(self, state: _Optional[_Union[FreshnessState, str]] = ..., drifted_paths: _Optional[_Iterable[str]] = ..., base_changed_paths: _Optional[_Iterable[str]] = ..., foreign_staged_paths: _Optional[_Iterable[str]] = ..., current_head: _Optional[str] = ..., detail: _Optional[str] = ..., checked_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ...) -> None: ...

class ProposalEvent(_message.Message):
    __slots__ = ("action", "actor", "revision", "detail", "timestamp")
    ACTION_FIELD_NUMBER: _ClassVar[int]
    ACTOR_FIELD_NUMBER: _ClassVar[int]
    REVISION_FIELD_NUMBER: _ClassVar[int]
    DETAIL_FIELD_NUMBER: _ClassVar[int]
    TIMESTAMP_FIELD_NUMBER: _ClassVar[int]
    action: str
    actor: Actor
    revision: int
    detail: str
    timestamp: _timestamp_pb2.Timestamp
    def __init__(self, action: _Optional[str] = ..., actor: _Optional[_Union[Actor, _Mapping]] = ..., revision: _Optional[int] = ..., detail: _Optional[str] = ..., timestamp: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ...) -> None: ...

class Proposal(_message.Message):
    __slots__ = ("id", "repository_id", "state", "revision", "created_by", "base_head", "branch", "work", "message", "files", "excluded", "evidence", "proposal_digest", "commit_oid", "superseded_by", "freshness", "events", "issues", "created_at", "updated_at", "committed_at")
    ID_FIELD_NUMBER: _ClassVar[int]
    REPOSITORY_ID_FIELD_NUMBER: _ClassVar[int]
    STATE_FIELD_NUMBER: _ClassVar[int]
    REVISION_FIELD_NUMBER: _ClassVar[int]
    CREATED_BY_FIELD_NUMBER: _ClassVar[int]
    BASE_HEAD_FIELD_NUMBER: _ClassVar[int]
    BRANCH_FIELD_NUMBER: _ClassVar[int]
    WORK_FIELD_NUMBER: _ClassVar[int]
    MESSAGE_FIELD_NUMBER: _ClassVar[int]
    FILES_FIELD_NUMBER: _ClassVar[int]
    EXCLUDED_FIELD_NUMBER: _ClassVar[int]
    EVIDENCE_FIELD_NUMBER: _ClassVar[int]
    PROPOSAL_DIGEST_FIELD_NUMBER: _ClassVar[int]
    COMMIT_OID_FIELD_NUMBER: _ClassVar[int]
    SUPERSEDED_BY_FIELD_NUMBER: _ClassVar[int]
    FRESHNESS_FIELD_NUMBER: _ClassVar[int]
    EVENTS_FIELD_NUMBER: _ClassVar[int]
    ISSUES_FIELD_NUMBER: _ClassVar[int]
    CREATED_AT_FIELD_NUMBER: _ClassVar[int]
    UPDATED_AT_FIELD_NUMBER: _ClassVar[int]
    COMMITTED_AT_FIELD_NUMBER: _ClassVar[int]
    id: str
    repository_id: str
    state: ProposalState
    revision: int
    created_by: Actor
    base_head: str
    branch: str
    work: ProposalWork
    message: ProposalMessage
    files: _containers.RepeatedCompositeFieldContainer[ProposalFile]
    excluded: _containers.RepeatedCompositeFieldContainer[ProposalExclusion]
    evidence: ProposalEvidence
    proposal_digest: str
    commit_oid: str
    superseded_by: str
    freshness: ProposalFreshness
    events: _containers.RepeatedCompositeFieldContainer[ProposalEvent]
    issues: _containers.RepeatedCompositeFieldContainer[TrailerIssue]
    created_at: _timestamp_pb2.Timestamp
    updated_at: _timestamp_pb2.Timestamp
    committed_at: _timestamp_pb2.Timestamp
    def __init__(self, id: _Optional[str] = ..., repository_id: _Optional[str] = ..., state: _Optional[_Union[ProposalState, str]] = ..., revision: _Optional[int] = ..., created_by: _Optional[_Union[Actor, _Mapping]] = ..., base_head: _Optional[str] = ..., branch: _Optional[str] = ..., work: _Optional[_Union[ProposalWork, _Mapping]] = ..., message: _Optional[_Union[ProposalMessage, _Mapping]] = ..., files: _Optional[_Iterable[_Union[ProposalFile, _Mapping]]] = ..., excluded: _Optional[_Iterable[_Union[ProposalExclusion, _Mapping]]] = ..., evidence: _Optional[_Union[ProposalEvidence, _Mapping]] = ..., proposal_digest: _Optional[str] = ..., commit_oid: _Optional[str] = ..., superseded_by: _Optional[str] = ..., freshness: _Optional[_Union[ProposalFreshness, _Mapping]] = ..., events: _Optional[_Iterable[_Union[ProposalEvent, _Mapping]]] = ..., issues: _Optional[_Iterable[_Union[TrailerIssue, _Mapping]]] = ..., created_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., updated_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., committed_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ...) -> None: ...

class AnchorFile(_message.Message):
    __slots__ = ("path", "sha256", "blob_id", "deleted")
    PATH_FIELD_NUMBER: _ClassVar[int]
    SHA256_FIELD_NUMBER: _ClassVar[int]
    BLOB_ID_FIELD_NUMBER: _ClassVar[int]
    DELETED_FIELD_NUMBER: _ClassVar[int]
    path: str
    sha256: str
    blob_id: str
    deleted: bool
    def __init__(self, path: _Optional[str] = ..., sha256: _Optional[str] = ..., blob_id: _Optional[str] = ..., deleted: _Optional[bool] = ...) -> None: ...

class Anchor(_message.Message):
    __slots__ = ("id", "repository_id", "effort_ref", "epoch", "scopes", "head", "files", "created_by", "created_at")
    ID_FIELD_NUMBER: _ClassVar[int]
    REPOSITORY_ID_FIELD_NUMBER: _ClassVar[int]
    EFFORT_REF_FIELD_NUMBER: _ClassVar[int]
    EPOCH_FIELD_NUMBER: _ClassVar[int]
    SCOPES_FIELD_NUMBER: _ClassVar[int]
    HEAD_FIELD_NUMBER: _ClassVar[int]
    FILES_FIELD_NUMBER: _ClassVar[int]
    CREATED_BY_FIELD_NUMBER: _ClassVar[int]
    CREATED_AT_FIELD_NUMBER: _ClassVar[int]
    id: str
    repository_id: str
    effort_ref: str
    epoch: str
    scopes: _containers.RepeatedScalarFieldContainer[str]
    head: str
    files: _containers.RepeatedCompositeFieldContainer[AnchorFile]
    created_by: Actor
    created_at: _timestamp_pb2.Timestamp
    def __init__(self, id: _Optional[str] = ..., repository_id: _Optional[str] = ..., effort_ref: _Optional[str] = ..., epoch: _Optional[str] = ..., scopes: _Optional[_Iterable[str]] = ..., head: _Optional[str] = ..., files: _Optional[_Iterable[_Union[AnchorFile, _Mapping]]] = ..., created_by: _Optional[_Union[Actor, _Mapping]] = ..., created_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ...) -> None: ...

class AnchorScopeRequest(_message.Message):
    __slots__ = ("repository_id", "effort_ref", "epoch", "scopes", "creator_run_id")
    REPOSITORY_ID_FIELD_NUMBER: _ClassVar[int]
    EFFORT_REF_FIELD_NUMBER: _ClassVar[int]
    EPOCH_FIELD_NUMBER: _ClassVar[int]
    SCOPES_FIELD_NUMBER: _ClassVar[int]
    CREATOR_RUN_ID_FIELD_NUMBER: _ClassVar[int]
    repository_id: str
    effort_ref: str
    epoch: str
    scopes: _containers.RepeatedScalarFieldContainer[str]
    creator_run_id: str
    def __init__(self, repository_id: _Optional[str] = ..., effort_ref: _Optional[str] = ..., epoch: _Optional[str] = ..., scopes: _Optional[_Iterable[str]] = ..., creator_run_id: _Optional[str] = ...) -> None: ...

class AnchorScopeResponse(_message.Message):
    __slots__ = ("anchor",)
    ANCHOR_FIELD_NUMBER: _ClassVar[int]
    anchor: Anchor
    def __init__(self, anchor: _Optional[_Union[Anchor, _Mapping]] = ...) -> None: ...

class CreateProposalRequest(_message.Message):
    __slots__ = ("repository_id", "anchor_id", "work", "subject", "body", "trailers", "paths", "evidence", "creator_run_id", "validate_only")
    REPOSITORY_ID_FIELD_NUMBER: _ClassVar[int]
    ANCHOR_ID_FIELD_NUMBER: _ClassVar[int]
    WORK_FIELD_NUMBER: _ClassVar[int]
    SUBJECT_FIELD_NUMBER: _ClassVar[int]
    BODY_FIELD_NUMBER: _ClassVar[int]
    TRAILERS_FIELD_NUMBER: _ClassVar[int]
    PATHS_FIELD_NUMBER: _ClassVar[int]
    EVIDENCE_FIELD_NUMBER: _ClassVar[int]
    CREATOR_RUN_ID_FIELD_NUMBER: _ClassVar[int]
    VALIDATE_ONLY_FIELD_NUMBER: _ClassVar[int]
    repository_id: str
    anchor_id: str
    work: ProposalWork
    subject: str
    body: str
    trailers: _containers.RepeatedCompositeFieldContainer[ProposalTrailer]
    paths: _containers.RepeatedScalarFieldContainer[str]
    evidence: ProposalEvidence
    creator_run_id: str
    validate_only: bool
    def __init__(self, repository_id: _Optional[str] = ..., anchor_id: _Optional[str] = ..., work: _Optional[_Union[ProposalWork, _Mapping]] = ..., subject: _Optional[str] = ..., body: _Optional[str] = ..., trailers: _Optional[_Iterable[_Union[ProposalTrailer, _Mapping]]] = ..., paths: _Optional[_Iterable[str]] = ..., evidence: _Optional[_Union[ProposalEvidence, _Mapping]] = ..., creator_run_id: _Optional[str] = ..., validate_only: _Optional[bool] = ...) -> None: ...

class CreateProposalResponse(_message.Message):
    __slots__ = ("proposal", "issues", "stored", "superseded_id")
    PROPOSAL_FIELD_NUMBER: _ClassVar[int]
    ISSUES_FIELD_NUMBER: _ClassVar[int]
    STORED_FIELD_NUMBER: _ClassVar[int]
    SUPERSEDED_ID_FIELD_NUMBER: _ClassVar[int]
    proposal: Proposal
    issues: _containers.RepeatedCompositeFieldContainer[TrailerIssue]
    stored: bool
    superseded_id: str
    def __init__(self, proposal: _Optional[_Union[Proposal, _Mapping]] = ..., issues: _Optional[_Iterable[_Union[TrailerIssue, _Mapping]]] = ..., stored: _Optional[bool] = ..., superseded_id: _Optional[str] = ...) -> None: ...

class ListProposalsRequest(_message.Message):
    __slots__ = ("repository_id", "states", "effort_ref", "limit")
    REPOSITORY_ID_FIELD_NUMBER: _ClassVar[int]
    STATES_FIELD_NUMBER: _ClassVar[int]
    EFFORT_REF_FIELD_NUMBER: _ClassVar[int]
    LIMIT_FIELD_NUMBER: _ClassVar[int]
    repository_id: str
    states: _containers.RepeatedScalarFieldContainer[ProposalState]
    effort_ref: str
    limit: int
    def __init__(self, repository_id: _Optional[str] = ..., states: _Optional[_Iterable[_Union[ProposalState, str]]] = ..., effort_ref: _Optional[str] = ..., limit: _Optional[int] = ...) -> None: ...

class ListProposalsResponse(_message.Message):
    __slots__ = ("proposals", "open_count")
    PROPOSALS_FIELD_NUMBER: _ClassVar[int]
    OPEN_COUNT_FIELD_NUMBER: _ClassVar[int]
    proposals: _containers.RepeatedCompositeFieldContainer[Proposal]
    open_count: int
    def __init__(self, proposals: _Optional[_Iterable[_Union[Proposal, _Mapping]]] = ..., open_count: _Optional[int] = ...) -> None: ...

class GetProposalRequest(_message.Message):
    __slots__ = ("id",)
    ID_FIELD_NUMBER: _ClassVar[int]
    id: str
    def __init__(self, id: _Optional[str] = ...) -> None: ...

class GetProposalResponse(_message.Message):
    __slots__ = ("proposal",)
    PROPOSAL_FIELD_NUMBER: _ClassVar[int]
    proposal: Proposal
    def __init__(self, proposal: _Optional[_Union[Proposal, _Mapping]] = ...) -> None: ...

class EditProposalRequest(_message.Message):
    __slots__ = ("id", "expected_revision", "subject", "body", "trailers", "replace_trailers", "remove_paths")
    ID_FIELD_NUMBER: _ClassVar[int]
    EXPECTED_REVISION_FIELD_NUMBER: _ClassVar[int]
    SUBJECT_FIELD_NUMBER: _ClassVar[int]
    BODY_FIELD_NUMBER: _ClassVar[int]
    TRAILERS_FIELD_NUMBER: _ClassVar[int]
    REPLACE_TRAILERS_FIELD_NUMBER: _ClassVar[int]
    REMOVE_PATHS_FIELD_NUMBER: _ClassVar[int]
    id: str
    expected_revision: int
    subject: str
    body: str
    trailers: _containers.RepeatedCompositeFieldContainer[ProposalTrailer]
    replace_trailers: bool
    remove_paths: _containers.RepeatedScalarFieldContainer[str]
    def __init__(self, id: _Optional[str] = ..., expected_revision: _Optional[int] = ..., subject: _Optional[str] = ..., body: _Optional[str] = ..., trailers: _Optional[_Iterable[_Union[ProposalTrailer, _Mapping]]] = ..., replace_trailers: _Optional[bool] = ..., remove_paths: _Optional[_Iterable[str]] = ...) -> None: ...

class EditProposalResponse(_message.Message):
    __slots__ = ("proposal", "issues")
    PROPOSAL_FIELD_NUMBER: _ClassVar[int]
    ISSUES_FIELD_NUMBER: _ClassVar[int]
    proposal: Proposal
    issues: _containers.RepeatedCompositeFieldContainer[TrailerIssue]
    def __init__(self, proposal: _Optional[_Union[Proposal, _Mapping]] = ..., issues: _Optional[_Iterable[_Union[TrailerIssue, _Mapping]]] = ...) -> None: ...

class WithdrawProposalRequest(_message.Message):
    __slots__ = ("id", "reason")
    ID_FIELD_NUMBER: _ClassVar[int]
    REASON_FIELD_NUMBER: _ClassVar[int]
    id: str
    reason: str
    def __init__(self, id: _Optional[str] = ..., reason: _Optional[str] = ...) -> None: ...

class WithdrawProposalResponse(_message.Message):
    __slots__ = ("proposal",)
    PROPOSAL_FIELD_NUMBER: _ClassVar[int]
    proposal: Proposal
    def __init__(self, proposal: _Optional[_Union[Proposal, _Mapping]] = ...) -> None: ...

class RefreshProposalRequest(_message.Message):
    __slots__ = ("id", "expected_revision")
    ID_FIELD_NUMBER: _ClassVar[int]
    EXPECTED_REVISION_FIELD_NUMBER: _ClassVar[int]
    id: str
    expected_revision: int
    def __init__(self, id: _Optional[str] = ..., expected_revision: _Optional[int] = ...) -> None: ...

class RefreshProposalResponse(_message.Message):
    __slots__ = ("proposal", "dropped_paths")
    PROPOSAL_FIELD_NUMBER: _ClassVar[int]
    DROPPED_PATHS_FIELD_NUMBER: _ClassVar[int]
    proposal: Proposal
    dropped_paths: _containers.RepeatedScalarFieldContainer[str]
    def __init__(self, proposal: _Optional[_Union[Proposal, _Mapping]] = ..., dropped_paths: _Optional[_Iterable[str]] = ...) -> None: ...

class ApplyProposalRequest(_message.Message):
    __slots__ = ("repository_id", "intent_id", "id", "revision", "skip_precommit_once")
    REPOSITORY_ID_FIELD_NUMBER: _ClassVar[int]
    INTENT_ID_FIELD_NUMBER: _ClassVar[int]
    ID_FIELD_NUMBER: _ClassVar[int]
    REVISION_FIELD_NUMBER: _ClassVar[int]
    SKIP_PRECOMMIT_ONCE_FIELD_NUMBER: _ClassVar[int]
    repository_id: str
    intent_id: str
    id: str
    revision: int
    skip_precommit_once: bool
    def __init__(self, repository_id: _Optional[str] = ..., intent_id: _Optional[str] = ..., id: _Optional[str] = ..., revision: _Optional[int] = ..., skip_precommit_once: _Optional[bool] = ...) -> None: ...

class ApplyRefusal(_message.Message):
    __slots__ = ("code", "paths", "detail")
    CODE_FIELD_NUMBER: _ClassVar[int]
    PATHS_FIELD_NUMBER: _ClassVar[int]
    DETAIL_FIELD_NUMBER: _ClassVar[int]
    code: str
    paths: _containers.RepeatedScalarFieldContainer[str]
    detail: str
    def __init__(self, code: _Optional[str] = ..., paths: _Optional[_Iterable[str]] = ..., detail: _Optional[str] = ...) -> None: ...

class ApplyPrecommit(_message.Message):
    __slots__ = ("status", "command", "exit_code", "summary", "stdout", "stderr", "duration_ms")
    STATUS_FIELD_NUMBER: _ClassVar[int]
    COMMAND_FIELD_NUMBER: _ClassVar[int]
    EXIT_CODE_FIELD_NUMBER: _ClassVar[int]
    SUMMARY_FIELD_NUMBER: _ClassVar[int]
    STDOUT_FIELD_NUMBER: _ClassVar[int]
    STDERR_FIELD_NUMBER: _ClassVar[int]
    DURATION_MS_FIELD_NUMBER: _ClassVar[int]
    status: str
    command: str
    exit_code: int
    summary: str
    stdout: str
    stderr: str
    duration_ms: int
    def __init__(self, status: _Optional[str] = ..., command: _Optional[str] = ..., exit_code: _Optional[int] = ..., summary: _Optional[str] = ..., stdout: _Optional[str] = ..., stderr: _Optional[str] = ..., duration_ms: _Optional[int] = ...) -> None: ...

class ApplyProposalResponse(_message.Message):
    __slots__ = ("success", "commit_oid", "proposal", "refusal", "error", "precommit", "commit_verified", "verification_notes")
    SUCCESS_FIELD_NUMBER: _ClassVar[int]
    COMMIT_OID_FIELD_NUMBER: _ClassVar[int]
    PROPOSAL_FIELD_NUMBER: _ClassVar[int]
    REFUSAL_FIELD_NUMBER: _ClassVar[int]
    ERROR_FIELD_NUMBER: _ClassVar[int]
    PRECOMMIT_FIELD_NUMBER: _ClassVar[int]
    COMMIT_VERIFIED_FIELD_NUMBER: _ClassVar[int]
    VERIFICATION_NOTES_FIELD_NUMBER: _ClassVar[int]
    success: bool
    commit_oid: str
    proposal: Proposal
    refusal: ApplyRefusal
    error: str
    precommit: ApplyPrecommit
    commit_verified: bool
    verification_notes: _containers.RepeatedScalarFieldContainer[str]
    def __init__(self, success: _Optional[bool] = ..., commit_oid: _Optional[str] = ..., proposal: _Optional[_Union[Proposal, _Mapping]] = ..., refusal: _Optional[_Union[ApplyRefusal, _Mapping]] = ..., error: _Optional[str] = ..., precommit: _Optional[_Union[ApplyPrecommit, _Mapping]] = ..., commit_verified: _Optional[bool] = ..., verification_notes: _Optional[_Iterable[str]] = ...) -> None: ...
