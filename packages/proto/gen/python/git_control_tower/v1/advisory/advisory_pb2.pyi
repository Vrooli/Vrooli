from google.protobuf.internal import containers as _containers
from google.protobuf.internal import enum_type_wrapper as _enum_type_wrapper
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Iterable as _Iterable, Mapping as _Mapping
from typing import ClassVar as _ClassVar, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class WorktreeSelectionSource(int, metaclass=_enum_type_wrapper.EnumTypeWrapper):
    __slots__ = ()
    WORKTREE_SELECTION_SOURCE_UNSPECIFIED: _ClassVar[WorktreeSelectionSource]
    WORKTREE_SELECTION_SOURCE_INDEX_TO_WORKTREE: _ClassVar[WorktreeSelectionSource]
    WORKTREE_SELECTION_SOURCE_HEAD_TO_INDEX: _ClassVar[WorktreeSelectionSource]

class QueuedWorktreeSelectionState(int, metaclass=_enum_type_wrapper.EnumTypeWrapper):
    __slots__ = ()
    QUEUED_WORKTREE_SELECTION_STATE_UNSPECIFIED: _ClassVar[QueuedWorktreeSelectionState]
    QUEUED_WORKTREE_SELECTION_STATE_QUEUED: _ClassVar[QueuedWorktreeSelectionState]
    QUEUED_WORKTREE_SELECTION_STATE_AWAITING_UNSTAGE: _ClassVar[QueuedWorktreeSelectionState]
    QUEUED_WORKTREE_SELECTION_STATE_FAILED: _ClassVar[QueuedWorktreeSelectionState]
    QUEUED_WORKTREE_SELECTION_STATE_STALE: _ClassVar[QueuedWorktreeSelectionState]

class StagedMessageMode(int, metaclass=_enum_type_wrapper.EnumTypeWrapper):
    __slots__ = ()
    STAGED_MESSAGE_MODE_UNSPECIFIED: _ClassVar[StagedMessageMode]
    STAGED_MESSAGE_MODE_QUICK: _ClassVar[StagedMessageMode]
    STAGED_MESSAGE_MODE_DEEP: _ClassVar[StagedMessageMode]

class WorktreeSuggestionQueueState(int, metaclass=_enum_type_wrapper.EnumTypeWrapper):
    __slots__ = ()
    WORKTREE_SUGGESTION_QUEUE_STATE_UNSPECIFIED: _ClassVar[WorktreeSuggestionQueueState]
    WORKTREE_SUGGESTION_QUEUE_STATE_QUEUED: _ClassVar[WorktreeSuggestionQueueState]
    WORKTREE_SUGGESTION_QUEUE_STATE_STAGED: _ClassVar[WorktreeSuggestionQueueState]
    WORKTREE_SUGGESTION_QUEUE_STATE_FAILED: _ClassVar[WorktreeSuggestionQueueState]
    WORKTREE_SUGGESTION_QUEUE_STATE_STALE: _ClassVar[WorktreeSuggestionQueueState]
    WORKTREE_SUGGESTION_QUEUE_STATE_COMMIT_PENDING: _ClassVar[WorktreeSuggestionQueueState]
    WORKTREE_SUGGESTION_QUEUE_STATE_AWAITING_UNSTAGE: _ClassVar[WorktreeSuggestionQueueState]

class WorktreeSuggestionQueueDisposition(int, metaclass=_enum_type_wrapper.EnumTypeWrapper):
    __slots__ = ()
    WORKTREE_SUGGESTION_QUEUE_DISPOSITION_UNSPECIFIED: _ClassVar[WorktreeSuggestionQueueDisposition]
    WORKTREE_SUGGESTION_QUEUE_DISPOSITION_COMMITTED: _ClassVar[WorktreeSuggestionQueueDisposition]
    WORKTREE_SUGGESTION_QUEUE_DISPOSITION_REMOVED: _ClassVar[WorktreeSuggestionQueueDisposition]

class StagedMessageScope(int, metaclass=_enum_type_wrapper.EnumTypeWrapper):
    __slots__ = ()
    STAGED_MESSAGE_SCOPE_UNSPECIFIED: _ClassVar[StagedMessageScope]
    STAGED_MESSAGE_SCOPE_WHOLE_SELECTION: _ClassVar[StagedMessageScope]
    STAGED_MESSAGE_SCOPE_LIMITED: _ClassVar[StagedMessageScope]
WORKTREE_SELECTION_SOURCE_UNSPECIFIED: WorktreeSelectionSource
WORKTREE_SELECTION_SOURCE_INDEX_TO_WORKTREE: WorktreeSelectionSource
WORKTREE_SELECTION_SOURCE_HEAD_TO_INDEX: WorktreeSelectionSource
QUEUED_WORKTREE_SELECTION_STATE_UNSPECIFIED: QueuedWorktreeSelectionState
QUEUED_WORKTREE_SELECTION_STATE_QUEUED: QueuedWorktreeSelectionState
QUEUED_WORKTREE_SELECTION_STATE_AWAITING_UNSTAGE: QueuedWorktreeSelectionState
QUEUED_WORKTREE_SELECTION_STATE_FAILED: QueuedWorktreeSelectionState
QUEUED_WORKTREE_SELECTION_STATE_STALE: QueuedWorktreeSelectionState
STAGED_MESSAGE_MODE_UNSPECIFIED: StagedMessageMode
STAGED_MESSAGE_MODE_QUICK: StagedMessageMode
STAGED_MESSAGE_MODE_DEEP: StagedMessageMode
WORKTREE_SUGGESTION_QUEUE_STATE_UNSPECIFIED: WorktreeSuggestionQueueState
WORKTREE_SUGGESTION_QUEUE_STATE_QUEUED: WorktreeSuggestionQueueState
WORKTREE_SUGGESTION_QUEUE_STATE_STAGED: WorktreeSuggestionQueueState
WORKTREE_SUGGESTION_QUEUE_STATE_FAILED: WorktreeSuggestionQueueState
WORKTREE_SUGGESTION_QUEUE_STATE_STALE: WorktreeSuggestionQueueState
WORKTREE_SUGGESTION_QUEUE_STATE_COMMIT_PENDING: WorktreeSuggestionQueueState
WORKTREE_SUGGESTION_QUEUE_STATE_AWAITING_UNSTAGE: WorktreeSuggestionQueueState
WORKTREE_SUGGESTION_QUEUE_DISPOSITION_UNSPECIFIED: WorktreeSuggestionQueueDisposition
WORKTREE_SUGGESTION_QUEUE_DISPOSITION_COMMITTED: WorktreeSuggestionQueueDisposition
WORKTREE_SUGGESTION_QUEUE_DISPOSITION_REMOVED: WorktreeSuggestionQueueDisposition
STAGED_MESSAGE_SCOPE_UNSPECIFIED: StagedMessageScope
STAGED_MESSAGE_SCOPE_WHOLE_SELECTION: StagedMessageScope
STAGED_MESSAGE_SCOPE_LIMITED: StagedMessageScope

class QueueWorktreeSelectionRequest(_message.Message):
    __slots__ = ("repository_id", "selection_id", "source", "paths")
    REPOSITORY_ID_FIELD_NUMBER: _ClassVar[int]
    SELECTION_ID_FIELD_NUMBER: _ClassVar[int]
    SOURCE_FIELD_NUMBER: _ClassVar[int]
    PATHS_FIELD_NUMBER: _ClassVar[int]
    repository_id: str
    selection_id: str
    source: WorktreeSelectionSource
    paths: _containers.RepeatedScalarFieldContainer[str]
    def __init__(self, repository_id: _Optional[str] = ..., selection_id: _Optional[str] = ..., source: _Optional[_Union[WorktreeSelectionSource, str]] = ..., paths: _Optional[_Iterable[str]] = ...) -> None: ...

class FinalizeQueuedSelectionRequest(_message.Message):
    __slots__ = ("repository_id", "selection_id", "expected_snapshot_digest")
    REPOSITORY_ID_FIELD_NUMBER: _ClassVar[int]
    SELECTION_ID_FIELD_NUMBER: _ClassVar[int]
    EXPECTED_SNAPSHOT_DIGEST_FIELD_NUMBER: _ClassVar[int]
    repository_id: str
    selection_id: str
    expected_snapshot_digest: str
    def __init__(self, repository_id: _Optional[str] = ..., selection_id: _Optional[str] = ..., expected_snapshot_digest: _Optional[str] = ...) -> None: ...

class StartQueuedSelectionRequest(_message.Message):
    __slots__ = ("repository_id", "selection_id", "snapshot_digest", "request_key", "mode", "operator_note")
    REPOSITORY_ID_FIELD_NUMBER: _ClassVar[int]
    SELECTION_ID_FIELD_NUMBER: _ClassVar[int]
    SNAPSHOT_DIGEST_FIELD_NUMBER: _ClassVar[int]
    REQUEST_KEY_FIELD_NUMBER: _ClassVar[int]
    MODE_FIELD_NUMBER: _ClassVar[int]
    OPERATOR_NOTE_FIELD_NUMBER: _ClassVar[int]
    repository_id: str
    selection_id: str
    snapshot_digest: str
    request_key: str
    mode: StagedMessageMode
    operator_note: str
    def __init__(self, repository_id: _Optional[str] = ..., selection_id: _Optional[str] = ..., snapshot_digest: _Optional[str] = ..., request_key: _Optional[str] = ..., mode: _Optional[_Union[StagedMessageMode, str]] = ..., operator_note: _Optional[str] = ...) -> None: ...

class QueuedWorktreeSelection(_message.Message):
    __slots__ = ("selection_id", "repository_id", "source", "paths", "snapshot_digest", "state", "failure_reason")
    SELECTION_ID_FIELD_NUMBER: _ClassVar[int]
    REPOSITORY_ID_FIELD_NUMBER: _ClassVar[int]
    SOURCE_FIELD_NUMBER: _ClassVar[int]
    PATHS_FIELD_NUMBER: _ClassVar[int]
    SNAPSHOT_DIGEST_FIELD_NUMBER: _ClassVar[int]
    STATE_FIELD_NUMBER: _ClassVar[int]
    FAILURE_REASON_FIELD_NUMBER: _ClassVar[int]
    selection_id: str
    repository_id: str
    source: WorktreeSelectionSource
    paths: _containers.RepeatedScalarFieldContainer[str]
    snapshot_digest: str
    state: QueuedWorktreeSelectionState
    failure_reason: str
    def __init__(self, selection_id: _Optional[str] = ..., repository_id: _Optional[str] = ..., source: _Optional[_Union[WorktreeSelectionSource, str]] = ..., paths: _Optional[_Iterable[str]] = ..., snapshot_digest: _Optional[str] = ..., state: _Optional[_Union[QueuedWorktreeSelectionState, str]] = ..., failure_reason: _Optional[str] = ...) -> None: ...

class DraftRequest(_message.Message):
    __slots__ = ("kind", "subject", "evidence")
    KIND_FIELD_NUMBER: _ClassVar[int]
    SUBJECT_FIELD_NUMBER: _ClassVar[int]
    EVIDENCE_FIELD_NUMBER: _ClassVar[int]
    kind: str
    subject: ChangeSubject
    evidence: EvidenceBundle
    def __init__(self, kind: _Optional[str] = ..., subject: _Optional[_Union[ChangeSubject, _Mapping]] = ..., evidence: _Optional[_Union[EvidenceBundle, _Mapping]] = ...) -> None: ...

class ChangeSubject(_message.Message):
    __slots__ = ("repository_id", "kind", "host", "base_revision", "head_revision", "parent_revision", "scope", "snapshot_digest")
    REPOSITORY_ID_FIELD_NUMBER: _ClassVar[int]
    KIND_FIELD_NUMBER: _ClassVar[int]
    HOST_FIELD_NUMBER: _ClassVar[int]
    BASE_REVISION_FIELD_NUMBER: _ClassVar[int]
    HEAD_REVISION_FIELD_NUMBER: _ClassVar[int]
    PARENT_REVISION_FIELD_NUMBER: _ClassVar[int]
    SCOPE_FIELD_NUMBER: _ClassVar[int]
    SNAPSHOT_DIGEST_FIELD_NUMBER: _ClassVar[int]
    repository_id: str
    kind: str
    host: HostSubject
    base_revision: str
    head_revision: str
    parent_revision: str
    scope: SubjectScope
    snapshot_digest: str
    def __init__(self, repository_id: _Optional[str] = ..., kind: _Optional[str] = ..., host: _Optional[_Union[HostSubject, _Mapping]] = ..., base_revision: _Optional[str] = ..., head_revision: _Optional[str] = ..., parent_revision: _Optional[str] = ..., scope: _Optional[_Union[SubjectScope, _Mapping]] = ..., snapshot_digest: _Optional[str] = ...) -> None: ...

class HostSubject(_message.Message):
    __slots__ = ("provider", "instance_id", "repository_id", "change_number")
    PROVIDER_FIELD_NUMBER: _ClassVar[int]
    INSTANCE_ID_FIELD_NUMBER: _ClassVar[int]
    REPOSITORY_ID_FIELD_NUMBER: _ClassVar[int]
    CHANGE_NUMBER_FIELD_NUMBER: _ClassVar[int]
    provider: str
    instance_id: str
    repository_id: str
    change_number: str
    def __init__(self, provider: _Optional[str] = ..., instance_id: _Optional[str] = ..., repository_id: _Optional[str] = ..., change_number: _Optional[str] = ...) -> None: ...

class SubjectScope(_message.Message):
    __slots__ = ("paths", "selection_digest")
    PATHS_FIELD_NUMBER: _ClassVar[int]
    SELECTION_DIGEST_FIELD_NUMBER: _ClassVar[int]
    paths: _containers.RepeatedScalarFieldContainer[str]
    selection_digest: str
    def __init__(self, paths: _Optional[_Iterable[str]] = ..., selection_digest: _Optional[str] = ...) -> None: ...

class EvidenceBundle(_message.Message):
    __slots__ = ("operation_id", "subject_digest", "status", "claims", "coverage", "validation", "unknowns", "versions")
    class VersionsEntry(_message.Message):
        __slots__ = ("key", "value")
        KEY_FIELD_NUMBER: _ClassVar[int]
        VALUE_FIELD_NUMBER: _ClassVar[int]
        key: str
        value: str
        def __init__(self, key: _Optional[str] = ..., value: _Optional[str] = ...) -> None: ...
    OPERATION_ID_FIELD_NUMBER: _ClassVar[int]
    SUBJECT_DIGEST_FIELD_NUMBER: _ClassVar[int]
    STATUS_FIELD_NUMBER: _ClassVar[int]
    CLAIMS_FIELD_NUMBER: _ClassVar[int]
    COVERAGE_FIELD_NUMBER: _ClassVar[int]
    VALIDATION_FIELD_NUMBER: _ClassVar[int]
    UNKNOWNS_FIELD_NUMBER: _ClassVar[int]
    VERSIONS_FIELD_NUMBER: _ClassVar[int]
    operation_id: str
    subject_digest: str
    status: str
    claims: _containers.RepeatedCompositeFieldContainer[Claim]
    coverage: Coverage
    validation: _containers.RepeatedCompositeFieldContainer[ValidationRef]
    unknowns: _containers.RepeatedScalarFieldContainer[str]
    versions: _containers.ScalarMap[str, str]
    def __init__(self, operation_id: _Optional[str] = ..., subject_digest: _Optional[str] = ..., status: _Optional[str] = ..., claims: _Optional[_Iterable[_Union[Claim, _Mapping]]] = ..., coverage: _Optional[_Union[Coverage, _Mapping]] = ..., validation: _Optional[_Iterable[_Union[ValidationRef, _Mapping]]] = ..., unknowns: _Optional[_Iterable[str]] = ..., versions: _Optional[_Mapping[str, str]] = ...) -> None: ...

class Claim(_message.Message):
    __slots__ = ("text", "evidence_refs")
    TEXT_FIELD_NUMBER: _ClassVar[int]
    EVIDENCE_REFS_FIELD_NUMBER: _ClassVar[int]
    text: str
    evidence_refs: _containers.RepeatedScalarFieldContainer[str]
    def __init__(self, text: _Optional[str] = ..., evidence_refs: _Optional[_Iterable[str]] = ...) -> None: ...

class Coverage(_message.Message):
    __slots__ = ("included_files", "omitted_files", "omissions")
    INCLUDED_FILES_FIELD_NUMBER: _ClassVar[int]
    OMITTED_FILES_FIELD_NUMBER: _ClassVar[int]
    OMISSIONS_FIELD_NUMBER: _ClassVar[int]
    included_files: int
    omitted_files: int
    omissions: _containers.RepeatedCompositeFieldContainer[Omission]
    def __init__(self, included_files: _Optional[int] = ..., omitted_files: _Optional[int] = ..., omissions: _Optional[_Iterable[_Union[Omission, _Mapping]]] = ...) -> None: ...

class Omission(_message.Message):
    __slots__ = ("path", "reason")
    PATH_FIELD_NUMBER: _ClassVar[int]
    REASON_FIELD_NUMBER: _ClassVar[int]
    path: str
    reason: str
    def __init__(self, path: _Optional[str] = ..., reason: _Optional[str] = ...) -> None: ...

class ValidationRef(_message.Message):
    __slots__ = ("execution_id", "availability", "verdict")
    EXECUTION_ID_FIELD_NUMBER: _ClassVar[int]
    AVAILABILITY_FIELD_NUMBER: _ClassVar[int]
    VERDICT_FIELD_NUMBER: _ClassVar[int]
    execution_id: str
    availability: str
    verdict: str
    def __init__(self, execution_id: _Optional[str] = ..., availability: _Optional[str] = ..., verdict: _Optional[str] = ...) -> None: ...

class DraftResponse(_message.Message):
    __slots__ = ("status", "kind", "subject_digest", "title", "body", "unknowns", "coverage", "evidence_refs")
    STATUS_FIELD_NUMBER: _ClassVar[int]
    KIND_FIELD_NUMBER: _ClassVar[int]
    SUBJECT_DIGEST_FIELD_NUMBER: _ClassVar[int]
    TITLE_FIELD_NUMBER: _ClassVar[int]
    BODY_FIELD_NUMBER: _ClassVar[int]
    UNKNOWNS_FIELD_NUMBER: _ClassVar[int]
    COVERAGE_FIELD_NUMBER: _ClassVar[int]
    EVIDENCE_REFS_FIELD_NUMBER: _ClassVar[int]
    status: str
    kind: str
    subject_digest: str
    title: str
    body: str
    unknowns: _containers.RepeatedScalarFieldContainer[str]
    coverage: Coverage
    evidence_refs: _containers.RepeatedScalarFieldContainer[str]
    def __init__(self, status: _Optional[str] = ..., kind: _Optional[str] = ..., subject_digest: _Optional[str] = ..., title: _Optional[str] = ..., body: _Optional[str] = ..., unknowns: _Optional[_Iterable[str]] = ..., coverage: _Optional[_Union[Coverage, _Mapping]] = ..., evidence_refs: _Optional[_Iterable[str]] = ...) -> None: ...

class CollectStagedMessageRequest(_message.Message):
    __slots__ = ("repository_id",)
    REPOSITORY_ID_FIELD_NUMBER: _ClassVar[int]
    repository_id: str
    def __init__(self, repository_id: _Optional[str] = ...) -> None: ...

class StartStagedMessageRequest(_message.Message):
    __slots__ = ("repository_id", "collection_id", "snapshot_digest", "selected_context_ids", "request_key", "operator_note", "continuation_of_job_id", "mode")
    REPOSITORY_ID_FIELD_NUMBER: _ClassVar[int]
    COLLECTION_ID_FIELD_NUMBER: _ClassVar[int]
    SNAPSHOT_DIGEST_FIELD_NUMBER: _ClassVar[int]
    SELECTED_CONTEXT_IDS_FIELD_NUMBER: _ClassVar[int]
    REQUEST_KEY_FIELD_NUMBER: _ClassVar[int]
    OPERATOR_NOTE_FIELD_NUMBER: _ClassVar[int]
    CONTINUATION_OF_JOB_ID_FIELD_NUMBER: _ClassVar[int]
    MODE_FIELD_NUMBER: _ClassVar[int]
    repository_id: str
    collection_id: str
    snapshot_digest: str
    selected_context_ids: _containers.RepeatedScalarFieldContainer[str]
    request_key: str
    operator_note: str
    continuation_of_job_id: str
    mode: StagedMessageMode
    def __init__(self, repository_id: _Optional[str] = ..., collection_id: _Optional[str] = ..., snapshot_digest: _Optional[str] = ..., selected_context_ids: _Optional[_Iterable[str]] = ..., request_key: _Optional[str] = ..., operator_note: _Optional[str] = ..., continuation_of_job_id: _Optional[str] = ..., mode: _Optional[_Union[StagedMessageMode, str]] = ...) -> None: ...

class RetryStagedMessageRequest(_message.Message):
    __slots__ = ("repository_id", "job_id", "snapshot_digest", "request_key", "mode")
    REPOSITORY_ID_FIELD_NUMBER: _ClassVar[int]
    JOB_ID_FIELD_NUMBER: _ClassVar[int]
    SNAPSHOT_DIGEST_FIELD_NUMBER: _ClassVar[int]
    REQUEST_KEY_FIELD_NUMBER: _ClassVar[int]
    MODE_FIELD_NUMBER: _ClassVar[int]
    repository_id: str
    job_id: str
    snapshot_digest: str
    request_key: str
    mode: StagedMessageMode
    def __init__(self, repository_id: _Optional[str] = ..., job_id: _Optional[str] = ..., snapshot_digest: _Optional[str] = ..., request_key: _Optional[str] = ..., mode: _Optional[_Union[StagedMessageMode, str]] = ...) -> None: ...

class StagedMessageProviderReadinessRequest(_message.Message):
    __slots__ = ()
    def __init__(self) -> None: ...

class StagedMessageProviderReadiness(_message.Message):
    __slots__ = ("provider", "state", "known", "observed_at", "message")
    PROVIDER_FIELD_NUMBER: _ClassVar[int]
    STATE_FIELD_NUMBER: _ClassVar[int]
    KNOWN_FIELD_NUMBER: _ClassVar[int]
    OBSERVED_AT_FIELD_NUMBER: _ClassVar[int]
    MESSAGE_FIELD_NUMBER: _ClassVar[int]
    provider: str
    state: str
    known: bool
    observed_at: str
    message: str
    def __init__(self, provider: _Optional[str] = ..., state: _Optional[str] = ..., known: _Optional[bool] = ..., observed_at: _Optional[str] = ..., message: _Optional[str] = ...) -> None: ...

class StartWorktreeStagedMessageRequest(_message.Message):
    __slots__ = ("repository_id", "paths", "request_key", "operator_note", "mode", "replace_queue_entry_id", "expected_queue_revision", "expected_snapshot_digest")
    REPOSITORY_ID_FIELD_NUMBER: _ClassVar[int]
    PATHS_FIELD_NUMBER: _ClassVar[int]
    REQUEST_KEY_FIELD_NUMBER: _ClassVar[int]
    OPERATOR_NOTE_FIELD_NUMBER: _ClassVar[int]
    MODE_FIELD_NUMBER: _ClassVar[int]
    REPLACE_QUEUE_ENTRY_ID_FIELD_NUMBER: _ClassVar[int]
    EXPECTED_QUEUE_REVISION_FIELD_NUMBER: _ClassVar[int]
    EXPECTED_SNAPSHOT_DIGEST_FIELD_NUMBER: _ClassVar[int]
    repository_id: str
    paths: _containers.RepeatedScalarFieldContainer[str]
    request_key: str
    operator_note: str
    mode: StagedMessageMode
    replace_queue_entry_id: str
    expected_queue_revision: int
    expected_snapshot_digest: str
    def __init__(self, repository_id: _Optional[str] = ..., paths: _Optional[_Iterable[str]] = ..., request_key: _Optional[str] = ..., operator_note: _Optional[str] = ..., mode: _Optional[_Union[StagedMessageMode, str]] = ..., replace_queue_entry_id: _Optional[str] = ..., expected_queue_revision: _Optional[int] = ..., expected_snapshot_digest: _Optional[str] = ...) -> None: ...

class ListWorktreeSuggestionQueueRequest(_message.Message):
    __slots__ = ("repository_id",)
    REPOSITORY_ID_FIELD_NUMBER: _ClassVar[int]
    repository_id: str
    def __init__(self, repository_id: _Optional[str] = ...) -> None: ...

class WorktreeSuggestionQueueEntry(_message.Message):
    __slots__ = ("id", "repository_id", "job_id", "snapshot_digest", "paths", "mode", "request_key", "sequence", "state", "failure_reason", "created_at", "updated_at", "selection_source", "selection_id")
    ID_FIELD_NUMBER: _ClassVar[int]
    REPOSITORY_ID_FIELD_NUMBER: _ClassVar[int]
    JOB_ID_FIELD_NUMBER: _ClassVar[int]
    SNAPSHOT_DIGEST_FIELD_NUMBER: _ClassVar[int]
    PATHS_FIELD_NUMBER: _ClassVar[int]
    MODE_FIELD_NUMBER: _ClassVar[int]
    REQUEST_KEY_FIELD_NUMBER: _ClassVar[int]
    SEQUENCE_FIELD_NUMBER: _ClassVar[int]
    STATE_FIELD_NUMBER: _ClassVar[int]
    FAILURE_REASON_FIELD_NUMBER: _ClassVar[int]
    CREATED_AT_FIELD_NUMBER: _ClassVar[int]
    UPDATED_AT_FIELD_NUMBER: _ClassVar[int]
    SELECTION_SOURCE_FIELD_NUMBER: _ClassVar[int]
    SELECTION_ID_FIELD_NUMBER: _ClassVar[int]
    id: str
    repository_id: str
    job_id: str
    snapshot_digest: str
    paths: _containers.RepeatedScalarFieldContainer[str]
    mode: StagedMessageMode
    request_key: str
    sequence: int
    state: WorktreeSuggestionQueueState
    failure_reason: str
    created_at: str
    updated_at: str
    selection_source: WorktreeSelectionSource
    selection_id: str
    def __init__(self, id: _Optional[str] = ..., repository_id: _Optional[str] = ..., job_id: _Optional[str] = ..., snapshot_digest: _Optional[str] = ..., paths: _Optional[_Iterable[str]] = ..., mode: _Optional[_Union[StagedMessageMode, str]] = ..., request_key: _Optional[str] = ..., sequence: _Optional[int] = ..., state: _Optional[_Union[WorktreeSuggestionQueueState, str]] = ..., failure_reason: _Optional[str] = ..., created_at: _Optional[str] = ..., updated_at: _Optional[str] = ..., selection_source: _Optional[_Union[WorktreeSelectionSource, str]] = ..., selection_id: _Optional[str] = ...) -> None: ...

class WorktreeSuggestionQueue(_message.Message):
    __slots__ = ("repository_id", "revision", "entries")
    REPOSITORY_ID_FIELD_NUMBER: _ClassVar[int]
    REVISION_FIELD_NUMBER: _ClassVar[int]
    ENTRIES_FIELD_NUMBER: _ClassVar[int]
    repository_id: str
    revision: int
    entries: _containers.RepeatedCompositeFieldContainer[WorktreeSuggestionQueueEntry]
    def __init__(self, repository_id: _Optional[str] = ..., revision: _Optional[int] = ..., entries: _Optional[_Iterable[_Union[WorktreeSuggestionQueueEntry, _Mapping]]] = ...) -> None: ...

class EnqueueWorktreeSuggestionRequest(_message.Message):
    __slots__ = ("repository_id", "job_id", "snapshot_digest", "paths", "mode", "request_key")
    REPOSITORY_ID_FIELD_NUMBER: _ClassVar[int]
    JOB_ID_FIELD_NUMBER: _ClassVar[int]
    SNAPSHOT_DIGEST_FIELD_NUMBER: _ClassVar[int]
    PATHS_FIELD_NUMBER: _ClassVar[int]
    MODE_FIELD_NUMBER: _ClassVar[int]
    REQUEST_KEY_FIELD_NUMBER: _ClassVar[int]
    repository_id: str
    job_id: str
    snapshot_digest: str
    paths: _containers.RepeatedScalarFieldContainer[str]
    mode: StagedMessageMode
    request_key: str
    def __init__(self, repository_id: _Optional[str] = ..., job_id: _Optional[str] = ..., snapshot_digest: _Optional[str] = ..., paths: _Optional[_Iterable[str]] = ..., mode: _Optional[_Union[StagedMessageMode, str]] = ..., request_key: _Optional[str] = ...) -> None: ...

class UpdateWorktreeSuggestionQueueEntryRequest(_message.Message):
    __slots__ = ("repository_id", "entry_id", "expected_head_entry_id", "expected_revision", "state", "failure_reason", "replacement_job_id", "replacement_snapshot_digest", "replacement_paths", "replacement_mode", "replacement_request_key")
    REPOSITORY_ID_FIELD_NUMBER: _ClassVar[int]
    ENTRY_ID_FIELD_NUMBER: _ClassVar[int]
    EXPECTED_HEAD_ENTRY_ID_FIELD_NUMBER: _ClassVar[int]
    EXPECTED_REVISION_FIELD_NUMBER: _ClassVar[int]
    STATE_FIELD_NUMBER: _ClassVar[int]
    FAILURE_REASON_FIELD_NUMBER: _ClassVar[int]
    REPLACEMENT_JOB_ID_FIELD_NUMBER: _ClassVar[int]
    REPLACEMENT_SNAPSHOT_DIGEST_FIELD_NUMBER: _ClassVar[int]
    REPLACEMENT_PATHS_FIELD_NUMBER: _ClassVar[int]
    REPLACEMENT_MODE_FIELD_NUMBER: _ClassVar[int]
    REPLACEMENT_REQUEST_KEY_FIELD_NUMBER: _ClassVar[int]
    repository_id: str
    entry_id: str
    expected_head_entry_id: str
    expected_revision: int
    state: WorktreeSuggestionQueueState
    failure_reason: str
    replacement_job_id: str
    replacement_snapshot_digest: str
    replacement_paths: _containers.RepeatedScalarFieldContainer[str]
    replacement_mode: StagedMessageMode
    replacement_request_key: str
    def __init__(self, repository_id: _Optional[str] = ..., entry_id: _Optional[str] = ..., expected_head_entry_id: _Optional[str] = ..., expected_revision: _Optional[int] = ..., state: _Optional[_Union[WorktreeSuggestionQueueState, str]] = ..., failure_reason: _Optional[str] = ..., replacement_job_id: _Optional[str] = ..., replacement_snapshot_digest: _Optional[str] = ..., replacement_paths: _Optional[_Iterable[str]] = ..., replacement_mode: _Optional[_Union[StagedMessageMode, str]] = ..., replacement_request_key: _Optional[str] = ...) -> None: ...

class RetireWorktreeSuggestionQueueHeadRequest(_message.Message):
    __slots__ = ("repository_id", "expected_head_entry_id", "expected_revision", "disposition")
    REPOSITORY_ID_FIELD_NUMBER: _ClassVar[int]
    EXPECTED_HEAD_ENTRY_ID_FIELD_NUMBER: _ClassVar[int]
    EXPECTED_REVISION_FIELD_NUMBER: _ClassVar[int]
    DISPOSITION_FIELD_NUMBER: _ClassVar[int]
    repository_id: str
    expected_head_entry_id: str
    expected_revision: int
    disposition: WorktreeSuggestionQueueDisposition
    def __init__(self, repository_id: _Optional[str] = ..., expected_head_entry_id: _Optional[str] = ..., expected_revision: _Optional[int] = ..., disposition: _Optional[_Union[WorktreeSuggestionQueueDisposition, str]] = ...) -> None: ...

class RemoveWorktreeSuggestionQueueEntryRequest(_message.Message):
    __slots__ = ("repository_id", "entry_id", "expected_revision")
    REPOSITORY_ID_FIELD_NUMBER: _ClassVar[int]
    ENTRY_ID_FIELD_NUMBER: _ClassVar[int]
    EXPECTED_REVISION_FIELD_NUMBER: _ClassVar[int]
    repository_id: str
    entry_id: str
    expected_revision: int
    def __init__(self, repository_id: _Optional[str] = ..., entry_id: _Optional[str] = ..., expected_revision: _Optional[int] = ...) -> None: ...

class WorktreeSuggestionQueueMutationResponse(_message.Message):
    __slots__ = ("queue", "changed_entry")
    QUEUE_FIELD_NUMBER: _ClassVar[int]
    CHANGED_ENTRY_FIELD_NUMBER: _ClassVar[int]
    queue: WorktreeSuggestionQueue
    changed_entry: WorktreeSuggestionQueueEntry
    def __init__(self, queue: _Optional[_Union[WorktreeSuggestionQueue, _Mapping]] = ..., changed_entry: _Optional[_Union[WorktreeSuggestionQueueEntry, _Mapping]] = ...) -> None: ...

class StagedMessageJobRequest(_message.Message):
    __slots__ = ("repository_id", "job_id", "timeout_seconds", "request_key")
    REPOSITORY_ID_FIELD_NUMBER: _ClassVar[int]
    JOB_ID_FIELD_NUMBER: _ClassVar[int]
    TIMEOUT_SECONDS_FIELD_NUMBER: _ClassVar[int]
    REQUEST_KEY_FIELD_NUMBER: _ClassVar[int]
    repository_id: str
    job_id: str
    timeout_seconds: int
    request_key: str
    def __init__(self, repository_id: _Optional[str] = ..., job_id: _Optional[str] = ..., timeout_seconds: _Optional[int] = ..., request_key: _Optional[str] = ...) -> None: ...

class StagedMessageTrailer(_message.Message):
    __slots__ = ("key", "value")
    KEY_FIELD_NUMBER: _ClassVar[int]
    VALUE_FIELD_NUMBER: _ClassVar[int]
    key: str
    value: str
    def __init__(self, key: _Optional[str] = ..., value: _Optional[str] = ...) -> None: ...

class StagedMessageTimingHint(_message.Message):
    __slots__ = ("source", "timestamp", "content_matches_index", "limitation")
    SOURCE_FIELD_NUMBER: _ClassVar[int]
    TIMESTAMP_FIELD_NUMBER: _ClassVar[int]
    CONTENT_MATCHES_INDEX_FIELD_NUMBER: _ClassVar[int]
    LIMITATION_FIELD_NUMBER: _ClassVar[int]
    source: str
    timestamp: str
    content_matches_index: bool
    limitation: str
    def __init__(self, source: _Optional[str] = ..., timestamp: _Optional[str] = ..., content_matches_index: _Optional[bool] = ..., limitation: _Optional[str] = ...) -> None: ...

class StagedMessageFile(_message.Message):
    __slots__ = ("path", "old_path", "status", "old_mode", "new_mode", "old_blob_id", "new_blob_id", "patch", "binary", "omission_reason", "timing_hints", "edit_count")
    PATH_FIELD_NUMBER: _ClassVar[int]
    OLD_PATH_FIELD_NUMBER: _ClassVar[int]
    STATUS_FIELD_NUMBER: _ClassVar[int]
    OLD_MODE_FIELD_NUMBER: _ClassVar[int]
    NEW_MODE_FIELD_NUMBER: _ClassVar[int]
    OLD_BLOB_ID_FIELD_NUMBER: _ClassVar[int]
    NEW_BLOB_ID_FIELD_NUMBER: _ClassVar[int]
    PATCH_FIELD_NUMBER: _ClassVar[int]
    BINARY_FIELD_NUMBER: _ClassVar[int]
    OMISSION_REASON_FIELD_NUMBER: _ClassVar[int]
    TIMING_HINTS_FIELD_NUMBER: _ClassVar[int]
    EDIT_COUNT_FIELD_NUMBER: _ClassVar[int]
    path: str
    old_path: str
    status: str
    old_mode: str
    new_mode: str
    old_blob_id: str
    new_blob_id: str
    patch: str
    binary: bool
    omission_reason: str
    timing_hints: _containers.RepeatedCompositeFieldContainer[StagedMessageTimingHint]
    edit_count: int
    def __init__(self, path: _Optional[str] = ..., old_path: _Optional[str] = ..., status: _Optional[str] = ..., old_mode: _Optional[str] = ..., new_mode: _Optional[str] = ..., old_blob_id: _Optional[str] = ..., new_blob_id: _Optional[str] = ..., patch: _Optional[str] = ..., binary: _Optional[bool] = ..., omission_reason: _Optional[str] = ..., timing_hints: _Optional[_Iterable[_Union[StagedMessageTimingHint, _Mapping]]] = ..., edit_count: _Optional[int] = ...) -> None: ...

class StagedMessageSnapshot(_message.Message):
    __slots__ = ("digest", "repository_id", "base_head", "files", "captured_at")
    DIGEST_FIELD_NUMBER: _ClassVar[int]
    REPOSITORY_ID_FIELD_NUMBER: _ClassVar[int]
    BASE_HEAD_FIELD_NUMBER: _ClassVar[int]
    FILES_FIELD_NUMBER: _ClassVar[int]
    CAPTURED_AT_FIELD_NUMBER: _ClassVar[int]
    digest: str
    repository_id: str
    base_head: str
    files: _containers.RepeatedCompositeFieldContainer[StagedMessageFile]
    captured_at: str
    def __init__(self, digest: _Optional[str] = ..., repository_id: _Optional[str] = ..., base_head: _Optional[str] = ..., files: _Optional[_Iterable[_Union[StagedMessageFile, _Mapping]]] = ..., captured_at: _Optional[str] = ...) -> None: ...

class StagedMessageCoverage(_message.Message):
    __slots__ = ("covered_paths", "uncovered_paths", "matched_edits", "total_edits", "partial", "omissions")
    COVERED_PATHS_FIELD_NUMBER: _ClassVar[int]
    UNCOVERED_PATHS_FIELD_NUMBER: _ClassVar[int]
    MATCHED_EDITS_FIELD_NUMBER: _ClassVar[int]
    TOTAL_EDITS_FIELD_NUMBER: _ClassVar[int]
    PARTIAL_FIELD_NUMBER: _ClassVar[int]
    OMISSIONS_FIELD_NUMBER: _ClassVar[int]
    covered_paths: _containers.RepeatedScalarFieldContainer[str]
    uncovered_paths: _containers.RepeatedScalarFieldContainer[str]
    matched_edits: int
    total_edits: int
    partial: bool
    omissions: _containers.RepeatedCompositeFieldContainer[Omission]
    def __init__(self, covered_paths: _Optional[_Iterable[str]] = ..., uncovered_paths: _Optional[_Iterable[str]] = ..., matched_edits: _Optional[int] = ..., total_edits: _Optional[int] = ..., partial: _Optional[bool] = ..., omissions: _Optional[_Iterable[_Union[Omission, _Mapping]]] = ...) -> None: ...

class StagedMessageContext(_message.Message):
    __slots__ = ("id", "kind", "reference", "revision", "title", "body", "status", "detail", "matched_paths", "matched_edits", "full_match", "unknowns", "activity_start", "activity_end")
    ID_FIELD_NUMBER: _ClassVar[int]
    KIND_FIELD_NUMBER: _ClassVar[int]
    REFERENCE_FIELD_NUMBER: _ClassVar[int]
    REVISION_FIELD_NUMBER: _ClassVar[int]
    TITLE_FIELD_NUMBER: _ClassVar[int]
    BODY_FIELD_NUMBER: _ClassVar[int]
    STATUS_FIELD_NUMBER: _ClassVar[int]
    DETAIL_FIELD_NUMBER: _ClassVar[int]
    MATCHED_PATHS_FIELD_NUMBER: _ClassVar[int]
    MATCHED_EDITS_FIELD_NUMBER: _ClassVar[int]
    FULL_MATCH_FIELD_NUMBER: _ClassVar[int]
    UNKNOWNS_FIELD_NUMBER: _ClassVar[int]
    ACTIVITY_START_FIELD_NUMBER: _ClassVar[int]
    ACTIVITY_END_FIELD_NUMBER: _ClassVar[int]
    id: str
    kind: str
    reference: str
    revision: str
    title: str
    body: str
    status: str
    detail: str
    matched_paths: _containers.RepeatedScalarFieldContainer[str]
    matched_edits: int
    full_match: bool
    unknowns: _containers.RepeatedScalarFieldContainer[str]
    activity_start: str
    activity_end: str
    def __init__(self, id: _Optional[str] = ..., kind: _Optional[str] = ..., reference: _Optional[str] = ..., revision: _Optional[str] = ..., title: _Optional[str] = ..., body: _Optional[str] = ..., status: _Optional[str] = ..., detail: _Optional[str] = ..., matched_paths: _Optional[_Iterable[str]] = ..., matched_edits: _Optional[int] = ..., full_match: _Optional[bool] = ..., unknowns: _Optional[_Iterable[str]] = ..., activity_start: _Optional[str] = ..., activity_end: _Optional[str] = ...) -> None: ...

class StagedMessageSuggestion(_message.Message):
    __slots__ = ("id", "title", "body", "trailers", "coverage", "source_ids", "unknowns", "message_scope", "message_scope_explanation")
    ID_FIELD_NUMBER: _ClassVar[int]
    TITLE_FIELD_NUMBER: _ClassVar[int]
    BODY_FIELD_NUMBER: _ClassVar[int]
    TRAILERS_FIELD_NUMBER: _ClassVar[int]
    COVERAGE_FIELD_NUMBER: _ClassVar[int]
    SOURCE_IDS_FIELD_NUMBER: _ClassVar[int]
    UNKNOWNS_FIELD_NUMBER: _ClassVar[int]
    MESSAGE_SCOPE_FIELD_NUMBER: _ClassVar[int]
    MESSAGE_SCOPE_EXPLANATION_FIELD_NUMBER: _ClassVar[int]
    id: str
    title: str
    body: str
    trailers: _containers.RepeatedCompositeFieldContainer[StagedMessageTrailer]
    coverage: StagedMessageCoverage
    source_ids: _containers.RepeatedScalarFieldContainer[str]
    unknowns: _containers.RepeatedScalarFieldContainer[str]
    message_scope: StagedMessageScope
    message_scope_explanation: str
    def __init__(self, id: _Optional[str] = ..., title: _Optional[str] = ..., body: _Optional[str] = ..., trailers: _Optional[_Iterable[_Union[StagedMessageTrailer, _Mapping]]] = ..., coverage: _Optional[_Union[StagedMessageCoverage, _Mapping]] = ..., source_ids: _Optional[_Iterable[str]] = ..., unknowns: _Optional[_Iterable[str]] = ..., message_scope: _Optional[_Union[StagedMessageScope, str]] = ..., message_scope_explanation: _Optional[str] = ...) -> None: ...

class StagedMessageContinuation(_message.Message):
    __slots__ = ("commit", "subject", "reason", "trailers", "shared_paths", "committed_at")
    COMMIT_FIELD_NUMBER: _ClassVar[int]
    SUBJECT_FIELD_NUMBER: _ClassVar[int]
    REASON_FIELD_NUMBER: _ClassVar[int]
    TRAILERS_FIELD_NUMBER: _ClassVar[int]
    SHARED_PATHS_FIELD_NUMBER: _ClassVar[int]
    COMMITTED_AT_FIELD_NUMBER: _ClassVar[int]
    commit: str
    subject: str
    reason: str
    trailers: _containers.RepeatedCompositeFieldContainer[StagedMessageTrailer]
    shared_paths: int
    committed_at: str
    def __init__(self, commit: _Optional[str] = ..., subject: _Optional[str] = ..., reason: _Optional[str] = ..., trailers: _Optional[_Iterable[_Union[StagedMessageTrailer, _Mapping]]] = ..., shared_paths: _Optional[int] = ..., committed_at: _Optional[str] = ...) -> None: ...

class StagedMessageCollection(_message.Message):
    __slots__ = ("id", "snapshot", "suggestions", "contexts", "omissions", "status", "collected_at", "continuations")
    ID_FIELD_NUMBER: _ClassVar[int]
    SNAPSHOT_FIELD_NUMBER: _ClassVar[int]
    SUGGESTIONS_FIELD_NUMBER: _ClassVar[int]
    CONTEXTS_FIELD_NUMBER: _ClassVar[int]
    OMISSIONS_FIELD_NUMBER: _ClassVar[int]
    STATUS_FIELD_NUMBER: _ClassVar[int]
    COLLECTED_AT_FIELD_NUMBER: _ClassVar[int]
    CONTINUATIONS_FIELD_NUMBER: _ClassVar[int]
    id: str
    snapshot: StagedMessageSnapshot
    suggestions: _containers.RepeatedCompositeFieldContainer[StagedMessageSuggestion]
    contexts: _containers.RepeatedCompositeFieldContainer[StagedMessageContext]
    omissions: _containers.RepeatedCompositeFieldContainer[Omission]
    status: str
    collected_at: str
    continuations: _containers.RepeatedCompositeFieldContainer[StagedMessageContinuation]
    def __init__(self, id: _Optional[str] = ..., snapshot: _Optional[_Union[StagedMessageSnapshot, _Mapping]] = ..., suggestions: _Optional[_Iterable[_Union[StagedMessageSuggestion, _Mapping]]] = ..., contexts: _Optional[_Iterable[_Union[StagedMessageContext, _Mapping]]] = ..., omissions: _Optional[_Iterable[_Union[Omission, _Mapping]]] = ..., status: _Optional[str] = ..., collected_at: _Optional[str] = ..., continuations: _Optional[_Iterable[_Union[StagedMessageContinuation, _Mapping]]] = ...) -> None: ...

class StagedMessageJob(_message.Message):
    __slots__ = ("id", "repository_id", "snapshot_digest", "status", "created_at", "updated_at", "execution_id", "cancel_requested", "result", "error", "applicable", "applicability_reason", "snapshot", "phase", "continuation_of_job_id", "evidence_files_read", "evidence_files_total", "continuations", "error_code", "retryable", "execution_started", "tokens_used", "tokens_known", "omissions", "execution_started_known", "charge_micro_usd", "charge_known")
    ID_FIELD_NUMBER: _ClassVar[int]
    REPOSITORY_ID_FIELD_NUMBER: _ClassVar[int]
    SNAPSHOT_DIGEST_FIELD_NUMBER: _ClassVar[int]
    STATUS_FIELD_NUMBER: _ClassVar[int]
    CREATED_AT_FIELD_NUMBER: _ClassVar[int]
    UPDATED_AT_FIELD_NUMBER: _ClassVar[int]
    EXECUTION_ID_FIELD_NUMBER: _ClassVar[int]
    CANCEL_REQUESTED_FIELD_NUMBER: _ClassVar[int]
    RESULT_FIELD_NUMBER: _ClassVar[int]
    ERROR_FIELD_NUMBER: _ClassVar[int]
    APPLICABLE_FIELD_NUMBER: _ClassVar[int]
    APPLICABILITY_REASON_FIELD_NUMBER: _ClassVar[int]
    SNAPSHOT_FIELD_NUMBER: _ClassVar[int]
    PHASE_FIELD_NUMBER: _ClassVar[int]
    CONTINUATION_OF_JOB_ID_FIELD_NUMBER: _ClassVar[int]
    EVIDENCE_FILES_READ_FIELD_NUMBER: _ClassVar[int]
    EVIDENCE_FILES_TOTAL_FIELD_NUMBER: _ClassVar[int]
    CONTINUATIONS_FIELD_NUMBER: _ClassVar[int]
    ERROR_CODE_FIELD_NUMBER: _ClassVar[int]
    RETRYABLE_FIELD_NUMBER: _ClassVar[int]
    EXECUTION_STARTED_FIELD_NUMBER: _ClassVar[int]
    TOKENS_USED_FIELD_NUMBER: _ClassVar[int]
    TOKENS_KNOWN_FIELD_NUMBER: _ClassVar[int]
    OMISSIONS_FIELD_NUMBER: _ClassVar[int]
    EXECUTION_STARTED_KNOWN_FIELD_NUMBER: _ClassVar[int]
    CHARGE_MICRO_USD_FIELD_NUMBER: _ClassVar[int]
    CHARGE_KNOWN_FIELD_NUMBER: _ClassVar[int]
    id: str
    repository_id: str
    snapshot_digest: str
    status: str
    created_at: str
    updated_at: str
    execution_id: str
    cancel_requested: bool
    result: StagedMessageSuggestion
    error: str
    applicable: bool
    applicability_reason: str
    snapshot: StagedMessageSnapshot
    phase: str
    continuation_of_job_id: str
    evidence_files_read: int
    evidence_files_total: int
    continuations: _containers.RepeatedCompositeFieldContainer[StagedMessageContinuation]
    error_code: str
    retryable: bool
    execution_started: bool
    tokens_used: int
    tokens_known: bool
    omissions: _containers.RepeatedCompositeFieldContainer[Omission]
    execution_started_known: bool
    charge_micro_usd: int
    charge_known: bool
    def __init__(self, id: _Optional[str] = ..., repository_id: _Optional[str] = ..., snapshot_digest: _Optional[str] = ..., status: _Optional[str] = ..., created_at: _Optional[str] = ..., updated_at: _Optional[str] = ..., execution_id: _Optional[str] = ..., cancel_requested: _Optional[bool] = ..., result: _Optional[_Union[StagedMessageSuggestion, _Mapping]] = ..., error: _Optional[str] = ..., applicable: _Optional[bool] = ..., applicability_reason: _Optional[str] = ..., snapshot: _Optional[_Union[StagedMessageSnapshot, _Mapping]] = ..., phase: _Optional[str] = ..., continuation_of_job_id: _Optional[str] = ..., evidence_files_read: _Optional[int] = ..., evidence_files_total: _Optional[int] = ..., continuations: _Optional[_Iterable[_Union[StagedMessageContinuation, _Mapping]]] = ..., error_code: _Optional[str] = ..., retryable: _Optional[bool] = ..., execution_started: _Optional[bool] = ..., tokens_used: _Optional[int] = ..., tokens_known: _Optional[bool] = ..., omissions: _Optional[_Iterable[_Union[Omission, _Mapping]]] = ..., execution_started_known: _Optional[bool] = ..., charge_micro_usd: _Optional[int] = ..., charge_known: _Optional[bool] = ...) -> None: ...

class ReadStagedMessageEvidenceRequest(_message.Message):
    __slots__ = ("repository_id", "job_id", "snapshot_digest", "cursor", "max_bytes", "inventory_only", "file_ids")
    REPOSITORY_ID_FIELD_NUMBER: _ClassVar[int]
    JOB_ID_FIELD_NUMBER: _ClassVar[int]
    SNAPSHOT_DIGEST_FIELD_NUMBER: _ClassVar[int]
    CURSOR_FIELD_NUMBER: _ClassVar[int]
    MAX_BYTES_FIELD_NUMBER: _ClassVar[int]
    INVENTORY_ONLY_FIELD_NUMBER: _ClassVar[int]
    FILE_IDS_FIELD_NUMBER: _ClassVar[int]
    repository_id: str
    job_id: str
    snapshot_digest: str
    cursor: int
    max_bytes: int
    inventory_only: bool
    file_ids: _containers.RepeatedScalarFieldContainer[int]
    def __init__(self, repository_id: _Optional[str] = ..., job_id: _Optional[str] = ..., snapshot_digest: _Optional[str] = ..., cursor: _Optional[int] = ..., max_bytes: _Optional[int] = ..., inventory_only: _Optional[bool] = ..., file_ids: _Optional[_Iterable[int]] = ...) -> None: ...

class StagedMessageEvidenceFile(_message.Message):
    __slots__ = ("file_id", "path", "old_path", "patch", "complete_patch", "omission_reason", "old_blob_id", "new_blob_id", "status", "edit_count", "old_mode", "new_mode")
    FILE_ID_FIELD_NUMBER: _ClassVar[int]
    PATH_FIELD_NUMBER: _ClassVar[int]
    OLD_PATH_FIELD_NUMBER: _ClassVar[int]
    PATCH_FIELD_NUMBER: _ClassVar[int]
    COMPLETE_PATCH_FIELD_NUMBER: _ClassVar[int]
    OMISSION_REASON_FIELD_NUMBER: _ClassVar[int]
    OLD_BLOB_ID_FIELD_NUMBER: _ClassVar[int]
    NEW_BLOB_ID_FIELD_NUMBER: _ClassVar[int]
    STATUS_FIELD_NUMBER: _ClassVar[int]
    EDIT_COUNT_FIELD_NUMBER: _ClassVar[int]
    OLD_MODE_FIELD_NUMBER: _ClassVar[int]
    NEW_MODE_FIELD_NUMBER: _ClassVar[int]
    file_id: int
    path: str
    old_path: str
    patch: str
    complete_patch: bool
    omission_reason: str
    old_blob_id: str
    new_blob_id: str
    status: str
    edit_count: int
    old_mode: str
    new_mode: str
    def __init__(self, file_id: _Optional[int] = ..., path: _Optional[str] = ..., old_path: _Optional[str] = ..., patch: _Optional[str] = ..., complete_patch: _Optional[bool] = ..., omission_reason: _Optional[str] = ..., old_blob_id: _Optional[str] = ..., new_blob_id: _Optional[str] = ..., status: _Optional[str] = ..., edit_count: _Optional[int] = ..., old_mode: _Optional[str] = ..., new_mode: _Optional[str] = ...) -> None: ...

class ReadStagedMessageEvidenceResponse(_message.Message):
    __slots__ = ("snapshot_digest", "files", "next_cursor", "complete", "receipt", "total_files")
    SNAPSHOT_DIGEST_FIELD_NUMBER: _ClassVar[int]
    FILES_FIELD_NUMBER: _ClassVar[int]
    NEXT_CURSOR_FIELD_NUMBER: _ClassVar[int]
    COMPLETE_FIELD_NUMBER: _ClassVar[int]
    RECEIPT_FIELD_NUMBER: _ClassVar[int]
    TOTAL_FILES_FIELD_NUMBER: _ClassVar[int]
    snapshot_digest: str
    files: _containers.RepeatedCompositeFieldContainer[StagedMessageEvidenceFile]
    next_cursor: int
    complete: bool
    receipt: str
    total_files: int
    def __init__(self, snapshot_digest: _Optional[str] = ..., files: _Optional[_Iterable[_Union[StagedMessageEvidenceFile, _Mapping]]] = ..., next_cursor: _Optional[int] = ..., complete: _Optional[bool] = ..., receipt: _Optional[str] = ..., total_files: _Optional[int] = ...) -> None: ...

class CheckStagedMessageRequest(_message.Message):
    __slots__ = ("repository_id", "snapshot_digest")
    REPOSITORY_ID_FIELD_NUMBER: _ClassVar[int]
    SNAPSHOT_DIGEST_FIELD_NUMBER: _ClassVar[int]
    repository_id: str
    snapshot_digest: str
    def __init__(self, repository_id: _Optional[str] = ..., snapshot_digest: _Optional[str] = ...) -> None: ...

class CheckStagedMessageResponse(_message.Message):
    __slots__ = ("applicable", "reason", "current_snapshot_digest")
    APPLICABLE_FIELD_NUMBER: _ClassVar[int]
    REASON_FIELD_NUMBER: _ClassVar[int]
    CURRENT_SNAPSHOT_DIGEST_FIELD_NUMBER: _ClassVar[int]
    applicable: bool
    reason: str
    current_snapshot_digest: str
    def __init__(self, applicable: _Optional[bool] = ..., reason: _Optional[str] = ..., current_snapshot_digest: _Optional[str] = ...) -> None: ...
