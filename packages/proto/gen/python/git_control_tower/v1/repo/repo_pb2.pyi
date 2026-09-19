from buf.validate import validate_pb2 as _validate_pb2
from google.protobuf.internal import containers as _containers
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Iterable as _Iterable, Mapping as _Mapping
from typing import ClassVar as _ClassVar, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class RepoRecord(_message.Message):
    __slots__ = ("id", "path", "name", "remote_url", "added_at", "last_opened_at", "favorite")
    ID_FIELD_NUMBER: _ClassVar[int]
    PATH_FIELD_NUMBER: _ClassVar[int]
    NAME_FIELD_NUMBER: _ClassVar[int]
    REMOTE_URL_FIELD_NUMBER: _ClassVar[int]
    ADDED_AT_FIELD_NUMBER: _ClassVar[int]
    LAST_OPENED_AT_FIELD_NUMBER: _ClassVar[int]
    FAVORITE_FIELD_NUMBER: _ClassVar[int]
    id: int
    path: str
    name: str
    remote_url: str
    added_at: str
    last_opened_at: str
    favorite: bool
    def __init__(self, id: _Optional[int] = ..., path: _Optional[str] = ..., name: _Optional[str] = ..., remote_url: _Optional[str] = ..., added_at: _Optional[str] = ..., last_opened_at: _Optional[str] = ..., favorite: _Optional[bool] = ...) -> None: ...

class ListRepositoriesRequest(_message.Message):
    __slots__ = ()
    def __init__(self) -> None: ...

class ListRepositoriesResponse(_message.Message):
    __slots__ = ("repos", "active_id", "timestamp")
    REPOS_FIELD_NUMBER: _ClassVar[int]
    ACTIVE_ID_FIELD_NUMBER: _ClassVar[int]
    TIMESTAMP_FIELD_NUMBER: _ClassVar[int]
    repos: _containers.RepeatedCompositeFieldContainer[RepoRecord]
    active_id: int
    timestamp: str
    def __init__(self, repos: _Optional[_Iterable[_Union[RepoRecord, _Mapping]]] = ..., active_id: _Optional[int] = ..., timestamp: _Optional[str] = ...) -> None: ...

class GetActiveRepositoryRequest(_message.Message):
    __slots__ = ()
    def __init__(self) -> None: ...

class GetActiveRepositoryResponse(_message.Message):
    __slots__ = ("repo", "timestamp")
    REPO_FIELD_NUMBER: _ClassVar[int]
    TIMESTAMP_FIELD_NUMBER: _ClassVar[int]
    repo: RepoRecord
    timestamp: str
    def __init__(self, repo: _Optional[_Union[RepoRecord, _Mapping]] = ..., timestamp: _Optional[str] = ...) -> None: ...

class GetRepoHistoryRequest(_message.Message):
    __slots__ = ("repository_id", "limit", "include_files", "include_checks", "grep_pattern")
    REPOSITORY_ID_FIELD_NUMBER: _ClassVar[int]
    LIMIT_FIELD_NUMBER: _ClassVar[int]
    INCLUDE_FILES_FIELD_NUMBER: _ClassVar[int]
    INCLUDE_CHECKS_FIELD_NUMBER: _ClassVar[int]
    GREP_PATTERN_FIELD_NUMBER: _ClassVar[int]
    repository_id: str
    limit: int
    include_files: bool
    include_checks: bool
    grep_pattern: str
    def __init__(self, repository_id: _Optional[str] = ..., limit: _Optional[int] = ..., include_files: _Optional[bool] = ..., include_checks: _Optional[bool] = ..., grep_pattern: _Optional[str] = ...) -> None: ...

class CommitCheckRun(_message.Message):
    __slots__ = ("kind", "status", "command", "exit_code", "summary", "stdout", "stderr", "duration_ms", "timestamp")
    KIND_FIELD_NUMBER: _ClassVar[int]
    STATUS_FIELD_NUMBER: _ClassVar[int]
    COMMAND_FIELD_NUMBER: _ClassVar[int]
    EXIT_CODE_FIELD_NUMBER: _ClassVar[int]
    SUMMARY_FIELD_NUMBER: _ClassVar[int]
    STDOUT_FIELD_NUMBER: _ClassVar[int]
    STDERR_FIELD_NUMBER: _ClassVar[int]
    DURATION_MS_FIELD_NUMBER: _ClassVar[int]
    TIMESTAMP_FIELD_NUMBER: _ClassVar[int]
    kind: str
    status: str
    command: str
    exit_code: int
    summary: str
    stdout: str
    stderr: str
    duration_ms: int
    timestamp: str
    def __init__(self, kind: _Optional[str] = ..., status: _Optional[str] = ..., command: _Optional[str] = ..., exit_code: _Optional[int] = ..., summary: _Optional[str] = ..., stdout: _Optional[str] = ..., stderr: _Optional[str] = ..., duration_ms: _Optional[int] = ..., timestamp: _Optional[str] = ...) -> None: ...

class RepoHistoryEntry(_message.Message):
    __slots__ = ("hash", "author", "date", "subject", "files", "checks")
    HASH_FIELD_NUMBER: _ClassVar[int]
    AUTHOR_FIELD_NUMBER: _ClassVar[int]
    DATE_FIELD_NUMBER: _ClassVar[int]
    SUBJECT_FIELD_NUMBER: _ClassVar[int]
    FILES_FIELD_NUMBER: _ClassVar[int]
    CHECKS_FIELD_NUMBER: _ClassVar[int]
    hash: str
    author: str
    date: str
    subject: str
    files: _containers.RepeatedScalarFieldContainer[str]
    checks: _containers.RepeatedCompositeFieldContainer[CommitCheckRun]
    def __init__(self, hash: _Optional[str] = ..., author: _Optional[str] = ..., date: _Optional[str] = ..., subject: _Optional[str] = ..., files: _Optional[_Iterable[str]] = ..., checks: _Optional[_Iterable[_Union[CommitCheckRun, _Mapping]]] = ...) -> None: ...

class GetRepoHistoryResponse(_message.Message):
    __slots__ = ("repo_dir", "lines", "entries", "limit", "grep_pattern", "timestamp")
    REPO_DIR_FIELD_NUMBER: _ClassVar[int]
    LINES_FIELD_NUMBER: _ClassVar[int]
    ENTRIES_FIELD_NUMBER: _ClassVar[int]
    LIMIT_FIELD_NUMBER: _ClassVar[int]
    GREP_PATTERN_FIELD_NUMBER: _ClassVar[int]
    TIMESTAMP_FIELD_NUMBER: _ClassVar[int]
    repo_dir: str
    lines: _containers.RepeatedScalarFieldContainer[str]
    entries: _containers.RepeatedCompositeFieldContainer[RepoHistoryEntry]
    limit: int
    grep_pattern: str
    timestamp: str
    def __init__(self, repo_dir: _Optional[str] = ..., lines: _Optional[_Iterable[str]] = ..., entries: _Optional[_Iterable[_Union[RepoHistoryEntry, _Mapping]]] = ..., limit: _Optional[int] = ..., grep_pattern: _Optional[str] = ..., timestamp: _Optional[str] = ...) -> None: ...

class GetApprovedChangesRequest(_message.Message):
    __slots__ = ("repository_id", "paths")
    REPOSITORY_ID_FIELD_NUMBER: _ClassVar[int]
    PATHS_FIELD_NUMBER: _ClassVar[int]
    repository_id: str
    paths: _containers.RepeatedScalarFieldContainer[str]
    def __init__(self, repository_id: _Optional[str] = ..., paths: _Optional[_Iterable[str]] = ...) -> None: ...

class ApprovedChangeFile(_message.Message):
    __slots__ = ("relative_path", "status", "sandbox_id", "sandbox_owner", "change_type", "agent_manager_run_id")
    RELATIVE_PATH_FIELD_NUMBER: _ClassVar[int]
    STATUS_FIELD_NUMBER: _ClassVar[int]
    SANDBOX_ID_FIELD_NUMBER: _ClassVar[int]
    SANDBOX_OWNER_FIELD_NUMBER: _ClassVar[int]
    CHANGE_TYPE_FIELD_NUMBER: _ClassVar[int]
    AGENT_MANAGER_RUN_ID_FIELD_NUMBER: _ClassVar[int]
    relative_path: str
    status: str
    sandbox_id: str
    sandbox_owner: str
    change_type: str
    agent_manager_run_id: str
    def __init__(self, relative_path: _Optional[str] = ..., status: _Optional[str] = ..., sandbox_id: _Optional[str] = ..., sandbox_owner: _Optional[str] = ..., change_type: _Optional[str] = ..., agent_manager_run_id: _Optional[str] = ...) -> None: ...

class GetApprovedChangesResponse(_message.Message):
    __slots__ = ("available", "committable_files", "suggested_message", "files", "warning")
    AVAILABLE_FIELD_NUMBER: _ClassVar[int]
    COMMITTABLE_FILES_FIELD_NUMBER: _ClassVar[int]
    SUGGESTED_MESSAGE_FIELD_NUMBER: _ClassVar[int]
    FILES_FIELD_NUMBER: _ClassVar[int]
    WARNING_FIELD_NUMBER: _ClassVar[int]
    available: bool
    committable_files: int
    suggested_message: str
    files: _containers.RepeatedCompositeFieldContainer[ApprovedChangeFile]
    warning: str
    def __init__(self, available: _Optional[bool] = ..., committable_files: _Optional[int] = ..., suggested_message: _Optional[str] = ..., files: _Optional[_Iterable[_Union[ApprovedChangeFile, _Mapping]]] = ..., warning: _Optional[str] = ...) -> None: ...

class GetProvenanceRequest(_message.Message):
    __slots__ = ("repository_id",)
    REPOSITORY_ID_FIELD_NUMBER: _ClassVar[int]
    repository_id: str
    def __init__(self, repository_id: _Optional[str] = ...) -> None: ...

class ProvenanceFile(_message.Message):
    __slots__ = ("file_path", "relative_path", "change_type", "applied_at", "visibility")
    FILE_PATH_FIELD_NUMBER: _ClassVar[int]
    RELATIVE_PATH_FIELD_NUMBER: _ClassVar[int]
    CHANGE_TYPE_FIELD_NUMBER: _ClassVar[int]
    APPLIED_AT_FIELD_NUMBER: _ClassVar[int]
    VISIBILITY_FIELD_NUMBER: _ClassVar[int]
    file_path: str
    relative_path: str
    change_type: str
    applied_at: str
    visibility: str
    def __init__(self, file_path: _Optional[str] = ..., relative_path: _Optional[str] = ..., change_type: _Optional[str] = ..., applied_at: _Optional[str] = ..., visibility: _Optional[str] = ...) -> None: ...

class ProvenanceRunGroup(_message.Message):
    __slots__ = ("run_id", "sandbox_id", "sandbox_owner", "files", "latest_applied_at")
    RUN_ID_FIELD_NUMBER: _ClassVar[int]
    SANDBOX_ID_FIELD_NUMBER: _ClassVar[int]
    SANDBOX_OWNER_FIELD_NUMBER: _ClassVar[int]
    FILES_FIELD_NUMBER: _ClassVar[int]
    LATEST_APPLIED_AT_FIELD_NUMBER: _ClassVar[int]
    run_id: str
    sandbox_id: str
    sandbox_owner: str
    files: _containers.RepeatedCompositeFieldContainer[ProvenanceFile]
    latest_applied_at: str
    def __init__(self, run_id: _Optional[str] = ..., sandbox_id: _Optional[str] = ..., sandbox_owner: _Optional[str] = ..., files: _Optional[_Iterable[_Union[ProvenanceFile, _Mapping]]] = ..., latest_applied_at: _Optional[str] = ...) -> None: ...

class GetProvenanceResponse(_message.Message):
    __slots__ = ("available", "run_groups", "warning")
    AVAILABLE_FIELD_NUMBER: _ClassVar[int]
    RUN_GROUPS_FIELD_NUMBER: _ClassVar[int]
    WARNING_FIELD_NUMBER: _ClassVar[int]
    available: bool
    run_groups: _containers.RepeatedCompositeFieldContainer[ProvenanceRunGroup]
    warning: str
    def __init__(self, available: _Optional[bool] = ..., run_groups: _Optional[_Iterable[_Union[ProvenanceRunGroup, _Mapping]]] = ..., warning: _Optional[str] = ...) -> None: ...

class GetBlameRequest(_message.Message):
    __slots__ = ("repository_id", "paths", "revision", "start_line", "end_line", "max_paths", "max_lines", "max_bytes", "enrich")
    REPOSITORY_ID_FIELD_NUMBER: _ClassVar[int]
    PATHS_FIELD_NUMBER: _ClassVar[int]
    REVISION_FIELD_NUMBER: _ClassVar[int]
    START_LINE_FIELD_NUMBER: _ClassVar[int]
    END_LINE_FIELD_NUMBER: _ClassVar[int]
    MAX_PATHS_FIELD_NUMBER: _ClassVar[int]
    MAX_LINES_FIELD_NUMBER: _ClassVar[int]
    MAX_BYTES_FIELD_NUMBER: _ClassVar[int]
    ENRICH_FIELD_NUMBER: _ClassVar[int]
    repository_id: str
    paths: _containers.RepeatedScalarFieldContainer[str]
    revision: str
    start_line: int
    end_line: int
    max_paths: int
    max_lines: int
    max_bytes: int
    enrich: bool
    def __init__(self, repository_id: _Optional[str] = ..., paths: _Optional[_Iterable[str]] = ..., revision: _Optional[str] = ..., start_line: _Optional[int] = ..., end_line: _Optional[int] = ..., max_paths: _Optional[int] = ..., max_lines: _Optional[int] = ..., max_bytes: _Optional[int] = ..., enrich: _Optional[bool] = ...) -> None: ...

class BlameLine(_message.Message):
    __slots__ = ("line", "content", "commit", "author", "author_time", "subject", "original_line", "original_path")
    LINE_FIELD_NUMBER: _ClassVar[int]
    CONTENT_FIELD_NUMBER: _ClassVar[int]
    COMMIT_FIELD_NUMBER: _ClassVar[int]
    AUTHOR_FIELD_NUMBER: _ClassVar[int]
    AUTHOR_TIME_FIELD_NUMBER: _ClassVar[int]
    SUBJECT_FIELD_NUMBER: _ClassVar[int]
    ORIGINAL_LINE_FIELD_NUMBER: _ClassVar[int]
    ORIGINAL_PATH_FIELD_NUMBER: _ClassVar[int]
    line: int
    content: str
    commit: str
    author: str
    author_time: str
    subject: str
    original_line: int
    original_path: str
    def __init__(self, line: _Optional[int] = ..., content: _Optional[str] = ..., commit: _Optional[str] = ..., author: _Optional[str] = ..., author_time: _Optional[str] = ..., subject: _Optional[str] = ..., original_line: _Optional[int] = ..., original_path: _Optional[str] = ...) -> None: ...

class ProvenanceWorkReference(_message.Message):
    __slots__ = ("kind", "id", "revision", "relationship", "verified", "visibility", "state", "unavailable_reason")
    KIND_FIELD_NUMBER: _ClassVar[int]
    ID_FIELD_NUMBER: _ClassVar[int]
    REVISION_FIELD_NUMBER: _ClassVar[int]
    RELATIONSHIP_FIELD_NUMBER: _ClassVar[int]
    VERIFIED_FIELD_NUMBER: _ClassVar[int]
    VISIBILITY_FIELD_NUMBER: _ClassVar[int]
    STATE_FIELD_NUMBER: _ClassVar[int]
    UNAVAILABLE_REASON_FIELD_NUMBER: _ClassVar[int]
    kind: str
    id: str
    revision: str
    relationship: str
    verified: bool
    visibility: str
    state: str
    unavailable_reason: str
    def __init__(self, kind: _Optional[str] = ..., id: _Optional[str] = ..., revision: _Optional[str] = ..., relationship: _Optional[str] = ..., verified: _Optional[bool] = ..., visibility: _Optional[str] = ..., state: _Optional[str] = ..., unavailable_reason: _Optional[str] = ...) -> None: ...

class BlameEvidence(_message.Message):
    __slots__ = ("run_id", "sandbox_id", "application_receipt", "content_digest", "commit_id", "visibility", "commit_state", "run_outcome", "conversation_id", "cost_usd", "committed_at", "unavailable", "work_references")
    RUN_ID_FIELD_NUMBER: _ClassVar[int]
    SANDBOX_ID_FIELD_NUMBER: _ClassVar[int]
    APPLICATION_RECEIPT_FIELD_NUMBER: _ClassVar[int]
    CONTENT_DIGEST_FIELD_NUMBER: _ClassVar[int]
    COMMIT_ID_FIELD_NUMBER: _ClassVar[int]
    VISIBILITY_FIELD_NUMBER: _ClassVar[int]
    COMMIT_STATE_FIELD_NUMBER: _ClassVar[int]
    RUN_OUTCOME_FIELD_NUMBER: _ClassVar[int]
    CONVERSATION_ID_FIELD_NUMBER: _ClassVar[int]
    COST_USD_FIELD_NUMBER: _ClassVar[int]
    COMMITTED_AT_FIELD_NUMBER: _ClassVar[int]
    UNAVAILABLE_FIELD_NUMBER: _ClassVar[int]
    WORK_REFERENCES_FIELD_NUMBER: _ClassVar[int]
    run_id: str
    sandbox_id: str
    application_receipt: str
    content_digest: str
    commit_id: str
    visibility: str
    commit_state: str
    run_outcome: str
    conversation_id: str
    cost_usd: float
    committed_at: str
    unavailable: _containers.RepeatedScalarFieldContainer[str]
    work_references: _containers.RepeatedCompositeFieldContainer[ProvenanceWorkReference]
    def __init__(self, run_id: _Optional[str] = ..., sandbox_id: _Optional[str] = ..., application_receipt: _Optional[str] = ..., content_digest: _Optional[str] = ..., commit_id: _Optional[str] = ..., visibility: _Optional[str] = ..., commit_state: _Optional[str] = ..., run_outcome: _Optional[str] = ..., conversation_id: _Optional[str] = ..., cost_usd: _Optional[float] = ..., committed_at: _Optional[str] = ..., unavailable: _Optional[_Iterable[str]] = ..., work_references: _Optional[_Iterable[_Union[ProvenanceWorkReference, _Mapping]]] = ...) -> None: ...

class BlameFile(_message.Message):
    __slots__ = ("path", "status", "lines", "content_digest", "reason", "standing", "downgrade_reasons", "evidence")
    PATH_FIELD_NUMBER: _ClassVar[int]
    STATUS_FIELD_NUMBER: _ClassVar[int]
    LINES_FIELD_NUMBER: _ClassVar[int]
    CONTENT_DIGEST_FIELD_NUMBER: _ClassVar[int]
    REASON_FIELD_NUMBER: _ClassVar[int]
    STANDING_FIELD_NUMBER: _ClassVar[int]
    DOWNGRADE_REASONS_FIELD_NUMBER: _ClassVar[int]
    EVIDENCE_FIELD_NUMBER: _ClassVar[int]
    path: str
    status: str
    lines: _containers.RepeatedCompositeFieldContainer[BlameLine]
    content_digest: str
    reason: str
    standing: str
    downgrade_reasons: _containers.RepeatedScalarFieldContainer[str]
    evidence: _containers.RepeatedCompositeFieldContainer[BlameEvidence]
    def __init__(self, path: _Optional[str] = ..., status: _Optional[str] = ..., lines: _Optional[_Iterable[_Union[BlameLine, _Mapping]]] = ..., content_digest: _Optional[str] = ..., reason: _Optional[str] = ..., standing: _Optional[str] = ..., downgrade_reasons: _Optional[_Iterable[str]] = ..., evidence: _Optional[_Iterable[_Union[BlameEvidence, _Mapping]]] = ...) -> None: ...

class ProvenanceChangeBundle(_message.Message):
    __slots__ = ("run_id", "sandbox_id", "files", "run_outcome", "conversation_id", "cost_usd", "work_references", "gaps")
    RUN_ID_FIELD_NUMBER: _ClassVar[int]
    SANDBOX_ID_FIELD_NUMBER: _ClassVar[int]
    FILES_FIELD_NUMBER: _ClassVar[int]
    RUN_OUTCOME_FIELD_NUMBER: _ClassVar[int]
    CONVERSATION_ID_FIELD_NUMBER: _ClassVar[int]
    COST_USD_FIELD_NUMBER: _ClassVar[int]
    WORK_REFERENCES_FIELD_NUMBER: _ClassVar[int]
    GAPS_FIELD_NUMBER: _ClassVar[int]
    run_id: str
    sandbox_id: str
    files: _containers.RepeatedScalarFieldContainer[str]
    run_outcome: str
    conversation_id: str
    cost_usd: float
    work_references: _containers.RepeatedCompositeFieldContainer[ProvenanceWorkReference]
    gaps: _containers.RepeatedScalarFieldContainer[str]
    def __init__(self, run_id: _Optional[str] = ..., sandbox_id: _Optional[str] = ..., files: _Optional[_Iterable[str]] = ..., run_outcome: _Optional[str] = ..., conversation_id: _Optional[str] = ..., cost_usd: _Optional[float] = ..., work_references: _Optional[_Iterable[_Union[ProvenanceWorkReference, _Mapping]]] = ..., gaps: _Optional[_Iterable[str]] = ...) -> None: ...

class GetBlameResponse(_message.Message):
    __slots__ = ("revision", "files", "truncated", "warnings", "change_bundles")
    REVISION_FIELD_NUMBER: _ClassVar[int]
    FILES_FIELD_NUMBER: _ClassVar[int]
    TRUNCATED_FIELD_NUMBER: _ClassVar[int]
    WARNINGS_FIELD_NUMBER: _ClassVar[int]
    CHANGE_BUNDLES_FIELD_NUMBER: _ClassVar[int]
    revision: str
    files: _containers.RepeatedCompositeFieldContainer[BlameFile]
    truncated: bool
    warnings: _containers.RepeatedScalarFieldContainer[str]
    change_bundles: _containers.RepeatedCompositeFieldContainer[ProvenanceChangeBundle]
    def __init__(self, revision: _Optional[str] = ..., files: _Optional[_Iterable[_Union[BlameFile, _Mapping]]] = ..., truncated: _Optional[bool] = ..., warnings: _Optional[_Iterable[str]] = ..., change_bundles: _Optional[_Iterable[_Union[ProvenanceChangeBundle, _Mapping]]] = ...) -> None: ...

class SearchProvenanceRequest(_message.Message):
    __slots__ = ("repository_id", "query", "limit", "scope")
    REPOSITORY_ID_FIELD_NUMBER: _ClassVar[int]
    QUERY_FIELD_NUMBER: _ClassVar[int]
    LIMIT_FIELD_NUMBER: _ClassVar[int]
    SCOPE_FIELD_NUMBER: _ClassVar[int]
    repository_id: int
    query: str
    limit: int
    scope: str
    def __init__(self, repository_id: _Optional[int] = ..., query: _Optional[str] = ..., limit: _Optional[int] = ..., scope: _Optional[str] = ...) -> None: ...

class ProvenanceSearchHit(_message.Message):
    __slots__ = ("id", "title", "snippet", "score", "run_id", "sandbox_id", "relative_path", "evidence_standing")
    ID_FIELD_NUMBER: _ClassVar[int]
    TITLE_FIELD_NUMBER: _ClassVar[int]
    SNIPPET_FIELD_NUMBER: _ClassVar[int]
    SCORE_FIELD_NUMBER: _ClassVar[int]
    RUN_ID_FIELD_NUMBER: _ClassVar[int]
    SANDBOX_ID_FIELD_NUMBER: _ClassVar[int]
    RELATIVE_PATH_FIELD_NUMBER: _ClassVar[int]
    EVIDENCE_STANDING_FIELD_NUMBER: _ClassVar[int]
    id: str
    title: str
    snippet: str
    score: float
    run_id: str
    sandbox_id: str
    relative_path: str
    evidence_standing: str
    def __init__(self, id: _Optional[str] = ..., title: _Optional[str] = ..., snippet: _Optional[str] = ..., score: _Optional[float] = ..., run_id: _Optional[str] = ..., sandbox_id: _Optional[str] = ..., relative_path: _Optional[str] = ..., evidence_standing: _Optional[str] = ...) -> None: ...

class SearchProvenanceResponse(_message.Message):
    __slots__ = ("available", "results", "warning")
    AVAILABLE_FIELD_NUMBER: _ClassVar[int]
    RESULTS_FIELD_NUMBER: _ClassVar[int]
    WARNING_FIELD_NUMBER: _ClassVar[int]
    available: bool
    results: _containers.RepeatedCompositeFieldContainer[ProvenanceSearchHit]
    warning: str
    def __init__(self, available: _Optional[bool] = ..., results: _Optional[_Iterable[_Union[ProvenanceSearchHit, _Mapping]]] = ..., warning: _Optional[str] = ...) -> None: ...

class SetActiveRepositoryRequest(_message.Message):
    __slots__ = ("repository_id",)
    REPOSITORY_ID_FIELD_NUMBER: _ClassVar[int]
    repository_id: int
    def __init__(self, repository_id: _Optional[int] = ...) -> None: ...

class OpenRepositoryRequest(_message.Message):
    __slots__ = ("path",)
    PATH_FIELD_NUMBER: _ClassVar[int]
    path: str
    def __init__(self, path: _Optional[str] = ...) -> None: ...

class CloneRepositoryRequest(_message.Message):
    __slots__ = ("url", "destination")
    URL_FIELD_NUMBER: _ClassVar[int]
    DESTINATION_FIELD_NUMBER: _ClassVar[int]
    url: str
    destination: str
    def __init__(self, url: _Optional[str] = ..., destination: _Optional[str] = ...) -> None: ...

class RepoMutationResponse(_message.Message):
    __slots__ = ("repo", "timestamp")
    REPO_FIELD_NUMBER: _ClassVar[int]
    TIMESTAMP_FIELD_NUMBER: _ClassVar[int]
    repo: RepoRecord
    timestamp: str
    def __init__(self, repo: _Optional[_Union[RepoRecord, _Mapping]] = ..., timestamp: _Optional[str] = ...) -> None: ...

class RemoveRepositoryRequest(_message.Message):
    __slots__ = ("repository_id",)
    REPOSITORY_ID_FIELD_NUMBER: _ClassVar[int]
    repository_id: int
    def __init__(self, repository_id: _Optional[int] = ...) -> None: ...

class RemoveRepositoryResponse(_message.Message):
    __slots__ = ("removed", "timestamp")
    REMOVED_FIELD_NUMBER: _ClassVar[int]
    TIMESTAMP_FIELD_NUMBER: _ClassVar[int]
    removed: bool
    timestamp: str
    def __init__(self, removed: _Optional[bool] = ..., timestamp: _Optional[str] = ...) -> None: ...

class GetRepoStatusRequest(_message.Message):
    __slots__ = ("repository_id", "repo_path", "include_hotspots")
    REPOSITORY_ID_FIELD_NUMBER: _ClassVar[int]
    REPO_PATH_FIELD_NUMBER: _ClassVar[int]
    INCLUDE_HOTSPOTS_FIELD_NUMBER: _ClassVar[int]
    repository_id: str
    repo_path: str
    include_hotspots: bool
    def __init__(self, repository_id: _Optional[str] = ..., repo_path: _Optional[str] = ..., include_hotspots: _Optional[bool] = ...) -> None: ...

class WorktreeIdentity(_message.Message):
    __slots__ = ("is_linked_worktree", "common_repo_root", "worktree_name", "worktree_head", "linked_worktree_count")
    IS_LINKED_WORKTREE_FIELD_NUMBER: _ClassVar[int]
    COMMON_REPO_ROOT_FIELD_NUMBER: _ClassVar[int]
    WORKTREE_NAME_FIELD_NUMBER: _ClassVar[int]
    WORKTREE_HEAD_FIELD_NUMBER: _ClassVar[int]
    LINKED_WORKTREE_COUNT_FIELD_NUMBER: _ClassVar[int]
    is_linked_worktree: bool
    common_repo_root: str
    worktree_name: str
    worktree_head: str
    linked_worktree_count: int
    def __init__(self, is_linked_worktree: _Optional[bool] = ..., common_repo_root: _Optional[str] = ..., worktree_name: _Optional[str] = ..., worktree_head: _Optional[str] = ..., linked_worktree_count: _Optional[int] = ...) -> None: ...

class GetRepoStatusResponse(_message.Message):
    __slots__ = ("branch", "detached", "worktree", "branch_status", "files", "file_stats", "file_hotspots", "scopes", "summary", "author", "repo_dir", "timestamp")
    class FileHotspotsEntry(_message.Message):
        __slots__ = ("key", "value")
        KEY_FIELD_NUMBER: _ClassVar[int]
        VALUE_FIELD_NUMBER: _ClassVar[int]
        key: str
        value: int
        def __init__(self, key: _Optional[str] = ..., value: _Optional[int] = ...) -> None: ...
    class ScopesEntry(_message.Message):
        __slots__ = ("key", "value")
        KEY_FIELD_NUMBER: _ClassVar[int]
        VALUE_FIELD_NUMBER: _ClassVar[int]
        key: str
        value: StringList
        def __init__(self, key: _Optional[str] = ..., value: _Optional[_Union[StringList, _Mapping]] = ...) -> None: ...
    BRANCH_FIELD_NUMBER: _ClassVar[int]
    DETACHED_FIELD_NUMBER: _ClassVar[int]
    WORKTREE_FIELD_NUMBER: _ClassVar[int]
    BRANCH_STATUS_FIELD_NUMBER: _ClassVar[int]
    FILES_FIELD_NUMBER: _ClassVar[int]
    FILE_STATS_FIELD_NUMBER: _ClassVar[int]
    FILE_HOTSPOTS_FIELD_NUMBER: _ClassVar[int]
    SCOPES_FIELD_NUMBER: _ClassVar[int]
    SUMMARY_FIELD_NUMBER: _ClassVar[int]
    AUTHOR_FIELD_NUMBER: _ClassVar[int]
    REPO_DIR_FIELD_NUMBER: _ClassVar[int]
    TIMESTAMP_FIELD_NUMBER: _ClassVar[int]
    branch: str
    detached: bool
    worktree: WorktreeIdentity
    branch_status: BranchStatus
    files: FilesStatus
    file_stats: FileStats
    file_hotspots: _containers.ScalarMap[str, int]
    scopes: _containers.MessageMap[str, StringList]
    summary: StatusSummary
    author: AuthorStatus
    repo_dir: str
    timestamp: str
    def __init__(self, branch: _Optional[str] = ..., detached: _Optional[bool] = ..., worktree: _Optional[_Union[WorktreeIdentity, _Mapping]] = ..., branch_status: _Optional[_Union[BranchStatus, _Mapping]] = ..., files: _Optional[_Union[FilesStatus, _Mapping]] = ..., file_stats: _Optional[_Union[FileStats, _Mapping]] = ..., file_hotspots: _Optional[_Mapping[str, int]] = ..., scopes: _Optional[_Mapping[str, StringList]] = ..., summary: _Optional[_Union[StatusSummary, _Mapping]] = ..., author: _Optional[_Union[AuthorStatus, _Mapping]] = ..., repo_dir: _Optional[str] = ..., timestamp: _Optional[str] = ...) -> None: ...

class BranchStatus(_message.Message):
    __slots__ = ("head", "upstream", "ahead", "behind", "oid")
    HEAD_FIELD_NUMBER: _ClassVar[int]
    UPSTREAM_FIELD_NUMBER: _ClassVar[int]
    AHEAD_FIELD_NUMBER: _ClassVar[int]
    BEHIND_FIELD_NUMBER: _ClassVar[int]
    OID_FIELD_NUMBER: _ClassVar[int]
    head: str
    upstream: str
    ahead: int
    behind: int
    oid: str
    def __init__(self, head: _Optional[str] = ..., upstream: _Optional[str] = ..., ahead: _Optional[int] = ..., behind: _Optional[int] = ..., oid: _Optional[str] = ...) -> None: ...

class FilesStatus(_message.Message):
    __slots__ = ("staged", "unstaged", "untracked", "conflicts", "binary", "ignored", "statuses", "renames")
    class StatusesEntry(_message.Message):
        __slots__ = ("key", "value")
        KEY_FIELD_NUMBER: _ClassVar[int]
        VALUE_FIELD_NUMBER: _ClassVar[int]
        key: str
        value: str
        def __init__(self, key: _Optional[str] = ..., value: _Optional[str] = ...) -> None: ...
    class RenamesEntry(_message.Message):
        __slots__ = ("key", "value")
        KEY_FIELD_NUMBER: _ClassVar[int]
        VALUE_FIELD_NUMBER: _ClassVar[int]
        key: str
        value: str
        def __init__(self, key: _Optional[str] = ..., value: _Optional[str] = ...) -> None: ...
    STAGED_FIELD_NUMBER: _ClassVar[int]
    UNSTAGED_FIELD_NUMBER: _ClassVar[int]
    UNTRACKED_FIELD_NUMBER: _ClassVar[int]
    CONFLICTS_FIELD_NUMBER: _ClassVar[int]
    BINARY_FIELD_NUMBER: _ClassVar[int]
    IGNORED_FIELD_NUMBER: _ClassVar[int]
    STATUSES_FIELD_NUMBER: _ClassVar[int]
    RENAMES_FIELD_NUMBER: _ClassVar[int]
    staged: _containers.RepeatedScalarFieldContainer[str]
    unstaged: _containers.RepeatedScalarFieldContainer[str]
    untracked: _containers.RepeatedScalarFieldContainer[str]
    conflicts: _containers.RepeatedScalarFieldContainer[str]
    binary: _containers.RepeatedScalarFieldContainer[str]
    ignored: _containers.RepeatedScalarFieldContainer[str]
    statuses: _containers.ScalarMap[str, str]
    renames: _containers.ScalarMap[str, str]
    def __init__(self, staged: _Optional[_Iterable[str]] = ..., unstaged: _Optional[_Iterable[str]] = ..., untracked: _Optional[_Iterable[str]] = ..., conflicts: _Optional[_Iterable[str]] = ..., binary: _Optional[_Iterable[str]] = ..., ignored: _Optional[_Iterable[str]] = ..., statuses: _Optional[_Mapping[str, str]] = ..., renames: _Optional[_Mapping[str, str]] = ...) -> None: ...

class FileStats(_message.Message):
    __slots__ = ("staged", "unstaged", "untracked")
    class StagedEntry(_message.Message):
        __slots__ = ("key", "value")
        KEY_FIELD_NUMBER: _ClassVar[int]
        VALUE_FIELD_NUMBER: _ClassVar[int]
        key: str
        value: DiffStats
        def __init__(self, key: _Optional[str] = ..., value: _Optional[_Union[DiffStats, _Mapping]] = ...) -> None: ...
    class UnstagedEntry(_message.Message):
        __slots__ = ("key", "value")
        KEY_FIELD_NUMBER: _ClassVar[int]
        VALUE_FIELD_NUMBER: _ClassVar[int]
        key: str
        value: DiffStats
        def __init__(self, key: _Optional[str] = ..., value: _Optional[_Union[DiffStats, _Mapping]] = ...) -> None: ...
    class UntrackedEntry(_message.Message):
        __slots__ = ("key", "value")
        KEY_FIELD_NUMBER: _ClassVar[int]
        VALUE_FIELD_NUMBER: _ClassVar[int]
        key: str
        value: DiffStats
        def __init__(self, key: _Optional[str] = ..., value: _Optional[_Union[DiffStats, _Mapping]] = ...) -> None: ...
    STAGED_FIELD_NUMBER: _ClassVar[int]
    UNSTAGED_FIELD_NUMBER: _ClassVar[int]
    UNTRACKED_FIELD_NUMBER: _ClassVar[int]
    staged: _containers.MessageMap[str, DiffStats]
    unstaged: _containers.MessageMap[str, DiffStats]
    untracked: _containers.MessageMap[str, DiffStats]
    def __init__(self, staged: _Optional[_Mapping[str, DiffStats]] = ..., unstaged: _Optional[_Mapping[str, DiffStats]] = ..., untracked: _Optional[_Mapping[str, DiffStats]] = ...) -> None: ...

class DiffStats(_message.Message):
    __slots__ = ("additions", "deletions", "files", "net_lines", "hunk_count", "largest_hunk", "density", "is_binary", "is_rename", "old_path", "comment_additions", "comment_deletions", "is_new_file", "is_deleted_file")
    ADDITIONS_FIELD_NUMBER: _ClassVar[int]
    DELETIONS_FIELD_NUMBER: _ClassVar[int]
    FILES_FIELD_NUMBER: _ClassVar[int]
    NET_LINES_FIELD_NUMBER: _ClassVar[int]
    HUNK_COUNT_FIELD_NUMBER: _ClassVar[int]
    LARGEST_HUNK_FIELD_NUMBER: _ClassVar[int]
    DENSITY_FIELD_NUMBER: _ClassVar[int]
    IS_BINARY_FIELD_NUMBER: _ClassVar[int]
    IS_RENAME_FIELD_NUMBER: _ClassVar[int]
    OLD_PATH_FIELD_NUMBER: _ClassVar[int]
    COMMENT_ADDITIONS_FIELD_NUMBER: _ClassVar[int]
    COMMENT_DELETIONS_FIELD_NUMBER: _ClassVar[int]
    IS_NEW_FILE_FIELD_NUMBER: _ClassVar[int]
    IS_DELETED_FILE_FIELD_NUMBER: _ClassVar[int]
    additions: int
    deletions: int
    files: int
    net_lines: int
    hunk_count: int
    largest_hunk: int
    density: float
    is_binary: bool
    is_rename: bool
    old_path: str
    comment_additions: int
    comment_deletions: int
    is_new_file: bool
    is_deleted_file: bool
    def __init__(self, additions: _Optional[int] = ..., deletions: _Optional[int] = ..., files: _Optional[int] = ..., net_lines: _Optional[int] = ..., hunk_count: _Optional[int] = ..., largest_hunk: _Optional[int] = ..., density: _Optional[float] = ..., is_binary: _Optional[bool] = ..., is_rename: _Optional[bool] = ..., old_path: _Optional[str] = ..., comment_additions: _Optional[int] = ..., comment_deletions: _Optional[int] = ..., is_new_file: _Optional[bool] = ..., is_deleted_file: _Optional[bool] = ...) -> None: ...

class StringList(_message.Message):
    __slots__ = ("values",)
    VALUES_FIELD_NUMBER: _ClassVar[int]
    values: _containers.RepeatedScalarFieldContainer[str]
    def __init__(self, values: _Optional[_Iterable[str]] = ...) -> None: ...

class StatusSummary(_message.Message):
    __slots__ = ("staged", "unstaged", "untracked", "conflicts", "ignored")
    STAGED_FIELD_NUMBER: _ClassVar[int]
    UNSTAGED_FIELD_NUMBER: _ClassVar[int]
    UNTRACKED_FIELD_NUMBER: _ClassVar[int]
    CONFLICTS_FIELD_NUMBER: _ClassVar[int]
    IGNORED_FIELD_NUMBER: _ClassVar[int]
    staged: int
    unstaged: int
    untracked: int
    conflicts: int
    ignored: int
    def __init__(self, staged: _Optional[int] = ..., unstaged: _Optional[int] = ..., untracked: _Optional[int] = ..., conflicts: _Optional[int] = ..., ignored: _Optional[int] = ...) -> None: ...

class AuthorStatus(_message.Message):
    __slots__ = ("name", "email")
    NAME_FIELD_NUMBER: _ClassVar[int]
    EMAIL_FIELD_NUMBER: _ClassVar[int]
    name: str
    email: str
    def __init__(self, name: _Optional[str] = ..., email: _Optional[str] = ...) -> None: ...

class GetRepoDiffRequest(_message.Message):
    __slots__ = ("repository_id", "repo_path", "path", "staged", "untracked", "base", "commit", "mode", "any")
    REPOSITORY_ID_FIELD_NUMBER: _ClassVar[int]
    REPO_PATH_FIELD_NUMBER: _ClassVar[int]
    PATH_FIELD_NUMBER: _ClassVar[int]
    STAGED_FIELD_NUMBER: _ClassVar[int]
    UNTRACKED_FIELD_NUMBER: _ClassVar[int]
    BASE_FIELD_NUMBER: _ClassVar[int]
    COMMIT_FIELD_NUMBER: _ClassVar[int]
    MODE_FIELD_NUMBER: _ClassVar[int]
    ANY_FIELD_NUMBER: _ClassVar[int]
    repository_id: str
    repo_path: str
    path: str
    staged: bool
    untracked: bool
    base: str
    commit: str
    mode: str
    any: bool
    def __init__(self, repository_id: _Optional[str] = ..., repo_path: _Optional[str] = ..., path: _Optional[str] = ..., staged: _Optional[bool] = ..., untracked: _Optional[bool] = ..., base: _Optional[str] = ..., commit: _Optional[str] = ..., mode: _Optional[str] = ..., any: _Optional[bool] = ...) -> None: ...

class GetRepoDiffResponse(_message.Message):
    __slots__ = ("repo_dir", "path", "staged", "untracked", "base", "has_diff", "hunks", "stats", "raw", "full_content", "content_hash", "annotated_lines", "mode", "timestamp")
    REPO_DIR_FIELD_NUMBER: _ClassVar[int]
    PATH_FIELD_NUMBER: _ClassVar[int]
    STAGED_FIELD_NUMBER: _ClassVar[int]
    UNTRACKED_FIELD_NUMBER: _ClassVar[int]
    BASE_FIELD_NUMBER: _ClassVar[int]
    HAS_DIFF_FIELD_NUMBER: _ClassVar[int]
    HUNKS_FIELD_NUMBER: _ClassVar[int]
    STATS_FIELD_NUMBER: _ClassVar[int]
    RAW_FIELD_NUMBER: _ClassVar[int]
    FULL_CONTENT_FIELD_NUMBER: _ClassVar[int]
    CONTENT_HASH_FIELD_NUMBER: _ClassVar[int]
    ANNOTATED_LINES_FIELD_NUMBER: _ClassVar[int]
    MODE_FIELD_NUMBER: _ClassVar[int]
    TIMESTAMP_FIELD_NUMBER: _ClassVar[int]
    repo_dir: str
    path: str
    staged: bool
    untracked: bool
    base: str
    has_diff: bool
    hunks: _containers.RepeatedCompositeFieldContainer[DiffHunk]
    stats: DiffStats
    raw: str
    full_content: str
    content_hash: str
    annotated_lines: _containers.RepeatedCompositeFieldContainer[AnnotatedLine]
    mode: str
    timestamp: str
    def __init__(self, repo_dir: _Optional[str] = ..., path: _Optional[str] = ..., staged: _Optional[bool] = ..., untracked: _Optional[bool] = ..., base: _Optional[str] = ..., has_diff: _Optional[bool] = ..., hunks: _Optional[_Iterable[_Union[DiffHunk, _Mapping]]] = ..., stats: _Optional[_Union[DiffStats, _Mapping]] = ..., raw: _Optional[str] = ..., full_content: _Optional[str] = ..., content_hash: _Optional[str] = ..., annotated_lines: _Optional[_Iterable[_Union[AnnotatedLine, _Mapping]]] = ..., mode: _Optional[str] = ..., timestamp: _Optional[str] = ...) -> None: ...

class DiffHunk(_message.Message):
    __slots__ = ("old_start", "old_count", "new_start", "new_count", "header", "lines")
    OLD_START_FIELD_NUMBER: _ClassVar[int]
    OLD_COUNT_FIELD_NUMBER: _ClassVar[int]
    NEW_START_FIELD_NUMBER: _ClassVar[int]
    NEW_COUNT_FIELD_NUMBER: _ClassVar[int]
    HEADER_FIELD_NUMBER: _ClassVar[int]
    LINES_FIELD_NUMBER: _ClassVar[int]
    old_start: int
    old_count: int
    new_start: int
    new_count: int
    header: str
    lines: _containers.RepeatedScalarFieldContainer[str]
    def __init__(self, old_start: _Optional[int] = ..., old_count: _Optional[int] = ..., new_start: _Optional[int] = ..., new_count: _Optional[int] = ..., header: _Optional[str] = ..., lines: _Optional[_Iterable[str]] = ...) -> None: ...

class AnnotatedLine(_message.Message):
    __slots__ = ("number", "content", "change", "old_number")
    NUMBER_FIELD_NUMBER: _ClassVar[int]
    CONTENT_FIELD_NUMBER: _ClassVar[int]
    CHANGE_FIELD_NUMBER: _ClassVar[int]
    OLD_NUMBER_FIELD_NUMBER: _ClassVar[int]
    number: int
    content: str
    change: str
    old_number: int
    def __init__(self, number: _Optional[int] = ..., content: _Optional[str] = ..., change: _Optional[str] = ..., old_number: _Optional[int] = ...) -> None: ...

class GetRepoGroupsRequest(_message.Message):
    __slots__ = ("repository_id", "repo_path")
    REPOSITORY_ID_FIELD_NUMBER: _ClassVar[int]
    REPO_PATH_FIELD_NUMBER: _ClassVar[int]
    repository_id: str
    repo_path: str
    def __init__(self, repository_id: _Optional[str] = ..., repo_path: _Optional[str] = ...) -> None: ...

class GetRepoGroupsResponse(_message.Message):
    __slots__ = ("groups",)
    GROUPS_FIELD_NUMBER: _ClassVar[int]
    groups: _containers.RepeatedCompositeFieldContainer[ChangeGroup]
    def __init__(self, groups: _Optional[_Iterable[_Union[ChangeGroup, _Mapping]]] = ...) -> None: ...

class ChangeGroup(_message.Message):
    __slots__ = ("key", "kind", "id", "label", "root", "source", "files")
    KEY_FIELD_NUMBER: _ClassVar[int]
    KIND_FIELD_NUMBER: _ClassVar[int]
    ID_FIELD_NUMBER: _ClassVar[int]
    LABEL_FIELD_NUMBER: _ClassVar[int]
    ROOT_FIELD_NUMBER: _ClassVar[int]
    SOURCE_FIELD_NUMBER: _ClassVar[int]
    FILES_FIELD_NUMBER: _ClassVar[int]
    key: str
    kind: str
    id: str
    label: str
    root: str
    source: str
    files: _containers.RepeatedScalarFieldContainer[str]
    def __init__(self, key: _Optional[str] = ..., kind: _Optional[str] = ..., id: _Optional[str] = ..., label: _Optional[str] = ..., root: _Optional[str] = ..., source: _Optional[str] = ..., files: _Optional[_Iterable[str]] = ...) -> None: ...

class GetSyncStatusRequest(_message.Message):
    __slots__ = ("repository_id", "repo_path", "fetch", "remote")
    REPOSITORY_ID_FIELD_NUMBER: _ClassVar[int]
    REPO_PATH_FIELD_NUMBER: _ClassVar[int]
    FETCH_FIELD_NUMBER: _ClassVar[int]
    REMOTE_FIELD_NUMBER: _ClassVar[int]
    repository_id: str
    repo_path: str
    fetch: bool
    remote: str
    def __init__(self, repository_id: _Optional[str] = ..., repo_path: _Optional[str] = ..., fetch: _Optional[bool] = ..., remote: _Optional[str] = ...) -> None: ...

class GetSyncStatusResponse(_message.Message):
    __slots__ = ("branch", "upstream", "remote_url", "ahead", "behind", "has_upstream", "can_push", "can_pull", "needs_pull", "needs_push", "has_uncommitted_changes", "safety_warnings", "recommendations", "fetched", "fetch_error", "timestamp")
    BRANCH_FIELD_NUMBER: _ClassVar[int]
    UPSTREAM_FIELD_NUMBER: _ClassVar[int]
    REMOTE_URL_FIELD_NUMBER: _ClassVar[int]
    AHEAD_FIELD_NUMBER: _ClassVar[int]
    BEHIND_FIELD_NUMBER: _ClassVar[int]
    HAS_UPSTREAM_FIELD_NUMBER: _ClassVar[int]
    CAN_PUSH_FIELD_NUMBER: _ClassVar[int]
    CAN_PULL_FIELD_NUMBER: _ClassVar[int]
    NEEDS_PULL_FIELD_NUMBER: _ClassVar[int]
    NEEDS_PUSH_FIELD_NUMBER: _ClassVar[int]
    HAS_UNCOMMITTED_CHANGES_FIELD_NUMBER: _ClassVar[int]
    SAFETY_WARNINGS_FIELD_NUMBER: _ClassVar[int]
    RECOMMENDATIONS_FIELD_NUMBER: _ClassVar[int]
    FETCHED_FIELD_NUMBER: _ClassVar[int]
    FETCH_ERROR_FIELD_NUMBER: _ClassVar[int]
    TIMESTAMP_FIELD_NUMBER: _ClassVar[int]
    branch: str
    upstream: str
    remote_url: str
    ahead: int
    behind: int
    has_upstream: bool
    can_push: bool
    can_pull: bool
    needs_pull: bool
    needs_push: bool
    has_uncommitted_changes: bool
    safety_warnings: _containers.RepeatedScalarFieldContainer[str]
    recommendations: _containers.RepeatedScalarFieldContainer[str]
    fetched: bool
    fetch_error: str
    timestamp: str
    def __init__(self, branch: _Optional[str] = ..., upstream: _Optional[str] = ..., remote_url: _Optional[str] = ..., ahead: _Optional[int] = ..., behind: _Optional[int] = ..., has_upstream: _Optional[bool] = ..., can_push: _Optional[bool] = ..., can_pull: _Optional[bool] = ..., needs_pull: _Optional[bool] = ..., needs_push: _Optional[bool] = ..., has_uncommitted_changes: _Optional[bool] = ..., safety_warnings: _Optional[_Iterable[str]] = ..., recommendations: _Optional[_Iterable[str]] = ..., fetched: _Optional[bool] = ..., fetch_error: _Optional[str] = ..., timestamp: _Optional[str] = ...) -> None: ...

class GetFilesRequest(_message.Message):
    __slots__ = ("repository_id", "pattern", "limit", "deep", "timeout_ms")
    REPOSITORY_ID_FIELD_NUMBER: _ClassVar[int]
    PATTERN_FIELD_NUMBER: _ClassVar[int]
    LIMIT_FIELD_NUMBER: _ClassVar[int]
    DEEP_FIELD_NUMBER: _ClassVar[int]
    TIMEOUT_MS_FIELD_NUMBER: _ClassVar[int]
    repository_id: str
    pattern: str
    limit: int
    deep: bool
    timeout_ms: int
    def __init__(self, repository_id: _Optional[str] = ..., pattern: _Optional[str] = ..., limit: _Optional[int] = ..., deep: _Optional[bool] = ..., timeout_ms: _Optional[int] = ...) -> None: ...

class RepoFileInfo(_message.Message):
    __slots__ = ("path", "language", "status")
    PATH_FIELD_NUMBER: _ClassVar[int]
    LANGUAGE_FIELD_NUMBER: _ClassVar[int]
    STATUS_FIELD_NUMBER: _ClassVar[int]
    path: str
    language: str
    status: str
    def __init__(self, path: _Optional[str] = ..., language: _Optional[str] = ..., status: _Optional[str] = ...) -> None: ...

class GetFilesResponse(_message.Message):
    __slots__ = ("files", "truncated", "cancelled", "search_mode", "timestamp")
    FILES_FIELD_NUMBER: _ClassVar[int]
    TRUNCATED_FIELD_NUMBER: _ClassVar[int]
    CANCELLED_FIELD_NUMBER: _ClassVar[int]
    SEARCH_MODE_FIELD_NUMBER: _ClassVar[int]
    TIMESTAMP_FIELD_NUMBER: _ClassVar[int]
    files: _containers.RepeatedCompositeFieldContainer[RepoFileInfo]
    truncated: bool
    cancelled: bool
    search_mode: str
    timestamp: str
    def __init__(self, files: _Optional[_Iterable[_Union[RepoFileInfo, _Mapping]]] = ..., truncated: _Optional[bool] = ..., cancelled: _Optional[bool] = ..., search_mode: _Optional[str] = ..., timestamp: _Optional[str] = ...) -> None: ...

class GetDirectoryContentsRequest(_message.Message):
    __slots__ = ("repository_id", "path")
    REPOSITORY_ID_FIELD_NUMBER: _ClassVar[int]
    PATH_FIELD_NUMBER: _ClassVar[int]
    repository_id: str
    path: str
    def __init__(self, repository_id: _Optional[str] = ..., path: _Optional[str] = ...) -> None: ...

class DirectoryEntry(_message.Message):
    __slots__ = ("name", "path", "is_dir", "language", "tracked")
    NAME_FIELD_NUMBER: _ClassVar[int]
    PATH_FIELD_NUMBER: _ClassVar[int]
    IS_DIR_FIELD_NUMBER: _ClassVar[int]
    LANGUAGE_FIELD_NUMBER: _ClassVar[int]
    TRACKED_FIELD_NUMBER: _ClassVar[int]
    name: str
    path: str
    is_dir: bool
    language: str
    tracked: bool
    def __init__(self, name: _Optional[str] = ..., path: _Optional[str] = ..., is_dir: _Optional[bool] = ..., language: _Optional[str] = ..., tracked: _Optional[bool] = ...) -> None: ...

class GetDirectoryContentsResponse(_message.Message):
    __slots__ = ("path", "entries", "timestamp")
    PATH_FIELD_NUMBER: _ClassVar[int]
    ENTRIES_FIELD_NUMBER: _ClassVar[int]
    TIMESTAMP_FIELD_NUMBER: _ClassVar[int]
    path: str
    entries: _containers.RepeatedCompositeFieldContainer[DirectoryEntry]
    timestamp: str
    def __init__(self, path: _Optional[str] = ..., entries: _Optional[_Iterable[_Union[DirectoryEntry, _Mapping]]] = ..., timestamp: _Optional[str] = ...) -> None: ...

class GetRelatedFilesRequest(_message.Message):
    __slots__ = ("repository_id", "path")
    REPOSITORY_ID_FIELD_NUMBER: _ClassVar[int]
    PATH_FIELD_NUMBER: _ClassVar[int]
    repository_id: str
    path: str
    def __init__(self, repository_id: _Optional[str] = ..., path: _Optional[str] = ...) -> None: ...

class RelatedFile(_message.Message):
    __slots__ = ("path", "relation_type")
    PATH_FIELD_NUMBER: _ClassVar[int]
    RELATION_TYPE_FIELD_NUMBER: _ClassVar[int]
    path: str
    relation_type: str
    def __init__(self, path: _Optional[str] = ..., relation_type: _Optional[str] = ...) -> None: ...

class GetRelatedFilesResponse(_message.Message):
    __slots__ = ("path", "related", "timestamp")
    PATH_FIELD_NUMBER: _ClassVar[int]
    RELATED_FIELD_NUMBER: _ClassVar[int]
    TIMESTAMP_FIELD_NUMBER: _ClassVar[int]
    path: str
    related: _containers.RepeatedCompositeFieldContainer[RelatedFile]
    timestamp: str
    def __init__(self, path: _Optional[str] = ..., related: _Optional[_Iterable[_Union[RelatedFile, _Mapping]]] = ..., timestamp: _Optional[str] = ...) -> None: ...

class SearchContentRequest(_message.Message):
    __slots__ = ("repository_id", "query", "case_sensitive", "whole_word", "regex", "include", "exclude", "context_lines", "limit", "timeout_ms")
    REPOSITORY_ID_FIELD_NUMBER: _ClassVar[int]
    QUERY_FIELD_NUMBER: _ClassVar[int]
    CASE_SENSITIVE_FIELD_NUMBER: _ClassVar[int]
    WHOLE_WORD_FIELD_NUMBER: _ClassVar[int]
    REGEX_FIELD_NUMBER: _ClassVar[int]
    INCLUDE_FIELD_NUMBER: _ClassVar[int]
    EXCLUDE_FIELD_NUMBER: _ClassVar[int]
    CONTEXT_LINES_FIELD_NUMBER: _ClassVar[int]
    LIMIT_FIELD_NUMBER: _ClassVar[int]
    TIMEOUT_MS_FIELD_NUMBER: _ClassVar[int]
    repository_id: str
    query: str
    case_sensitive: bool
    whole_word: bool
    regex: bool
    include: str
    exclude: str
    context_lines: int
    limit: int
    timeout_ms: int
    def __init__(self, repository_id: _Optional[str] = ..., query: _Optional[str] = ..., case_sensitive: _Optional[bool] = ..., whole_word: _Optional[bool] = ..., regex: _Optional[bool] = ..., include: _Optional[str] = ..., exclude: _Optional[str] = ..., context_lines: _Optional[int] = ..., limit: _Optional[int] = ..., timeout_ms: _Optional[int] = ...) -> None: ...

class ContentMatch(_message.Message):
    __slots__ = ("path", "line_number", "content", "context_before", "context_after")
    PATH_FIELD_NUMBER: _ClassVar[int]
    LINE_NUMBER_FIELD_NUMBER: _ClassVar[int]
    CONTENT_FIELD_NUMBER: _ClassVar[int]
    CONTEXT_BEFORE_FIELD_NUMBER: _ClassVar[int]
    CONTEXT_AFTER_FIELD_NUMBER: _ClassVar[int]
    path: str
    line_number: int
    content: str
    context_before: str
    context_after: str
    def __init__(self, path: _Optional[str] = ..., line_number: _Optional[int] = ..., content: _Optional[str] = ..., context_before: _Optional[str] = ..., context_after: _Optional[str] = ...) -> None: ...

class SearchContentResponse(_message.Message):
    __slots__ = ("matches", "total", "truncated", "cancelled", "query", "timestamp")
    MATCHES_FIELD_NUMBER: _ClassVar[int]
    TOTAL_FIELD_NUMBER: _ClassVar[int]
    TRUNCATED_FIELD_NUMBER: _ClassVar[int]
    CANCELLED_FIELD_NUMBER: _ClassVar[int]
    QUERY_FIELD_NUMBER: _ClassVar[int]
    TIMESTAMP_FIELD_NUMBER: _ClassVar[int]
    matches: _containers.RepeatedCompositeFieldContainer[ContentMatch]
    total: int
    truncated: bool
    cancelled: bool
    query: str
    timestamp: str
    def __init__(self, matches: _Optional[_Iterable[_Union[ContentMatch, _Mapping]]] = ..., total: _Optional[int] = ..., truncated: _Optional[bool] = ..., cancelled: _Optional[bool] = ..., query: _Optional[str] = ..., timestamp: _Optional[str] = ...) -> None: ...

class DeletePathRequest(_message.Message):
    __slots__ = ("repository_id", "intent_id", "path")
    REPOSITORY_ID_FIELD_NUMBER: _ClassVar[int]
    INTENT_ID_FIELD_NUMBER: _ClassVar[int]
    PATH_FIELD_NUMBER: _ClassVar[int]
    repository_id: str
    intent_id: str
    path: str
    def __init__(self, repository_id: _Optional[str] = ..., intent_id: _Optional[str] = ..., path: _Optional[str] = ...) -> None: ...

class DeletePathResponse(_message.Message):
    __slots__ = ("success", "path", "is_dir", "error", "timestamp")
    SUCCESS_FIELD_NUMBER: _ClassVar[int]
    PATH_FIELD_NUMBER: _ClassVar[int]
    IS_DIR_FIELD_NUMBER: _ClassVar[int]
    ERROR_FIELD_NUMBER: _ClassVar[int]
    TIMESTAMP_FIELD_NUMBER: _ClassVar[int]
    success: bool
    path: str
    is_dir: bool
    error: str
    timestamp: str
    def __init__(self, success: _Optional[bool] = ..., path: _Optional[str] = ..., is_dir: _Optional[bool] = ..., error: _Optional[str] = ..., timestamp: _Optional[str] = ...) -> None: ...

class SaveFileContentRequest(_message.Message):
    __slots__ = ("repository_id", "intent_id", "path", "content", "expected_hash")
    REPOSITORY_ID_FIELD_NUMBER: _ClassVar[int]
    INTENT_ID_FIELD_NUMBER: _ClassVar[int]
    PATH_FIELD_NUMBER: _ClassVar[int]
    CONTENT_FIELD_NUMBER: _ClassVar[int]
    EXPECTED_HASH_FIELD_NUMBER: _ClassVar[int]
    repository_id: str
    intent_id: str
    path: str
    content: str
    expected_hash: str
    def __init__(self, repository_id: _Optional[str] = ..., intent_id: _Optional[str] = ..., path: _Optional[str] = ..., content: _Optional[str] = ..., expected_hash: _Optional[str] = ...) -> None: ...

class SaveFileContentResponse(_message.Message):
    __slots__ = ("success", "path", "content_hash", "bytes_written", "timestamp")
    SUCCESS_FIELD_NUMBER: _ClassVar[int]
    PATH_FIELD_NUMBER: _ClassVar[int]
    CONTENT_HASH_FIELD_NUMBER: _ClassVar[int]
    BYTES_WRITTEN_FIELD_NUMBER: _ClassVar[int]
    TIMESTAMP_FIELD_NUMBER: _ClassVar[int]
    success: bool
    path: str
    content_hash: str
    bytes_written: int
    timestamp: str
    def __init__(self, success: _Optional[bool] = ..., path: _Optional[str] = ..., content_hash: _Optional[str] = ..., bytes_written: _Optional[int] = ..., timestamp: _Optional[str] = ...) -> None: ...

class DiscardFilesRequest(_message.Message):
    __slots__ = ("repository_id", "intent_id", "paths", "untracked")
    REPOSITORY_ID_FIELD_NUMBER: _ClassVar[int]
    INTENT_ID_FIELD_NUMBER: _ClassVar[int]
    PATHS_FIELD_NUMBER: _ClassVar[int]
    UNTRACKED_FIELD_NUMBER: _ClassVar[int]
    repository_id: str
    intent_id: str
    paths: _containers.RepeatedScalarFieldContainer[str]
    untracked: bool
    def __init__(self, repository_id: _Optional[str] = ..., intent_id: _Optional[str] = ..., paths: _Optional[_Iterable[str]] = ..., untracked: _Optional[bool] = ...) -> None: ...

class DiscardFilesResponse(_message.Message):
    __slots__ = ("success", "discarded", "failed", "errors", "timestamp")
    SUCCESS_FIELD_NUMBER: _ClassVar[int]
    DISCARDED_FIELD_NUMBER: _ClassVar[int]
    FAILED_FIELD_NUMBER: _ClassVar[int]
    ERRORS_FIELD_NUMBER: _ClassVar[int]
    TIMESTAMP_FIELD_NUMBER: _ClassVar[int]
    success: bool
    discarded: _containers.RepeatedScalarFieldContainer[str]
    failed: _containers.RepeatedScalarFieldContainer[str]
    errors: _containers.RepeatedScalarFieldContainer[str]
    timestamp: str
    def __init__(self, success: _Optional[bool] = ..., discarded: _Optional[_Iterable[str]] = ..., failed: _Optional[_Iterable[str]] = ..., errors: _Optional[_Iterable[str]] = ..., timestamp: _Optional[str] = ...) -> None: ...

class IgnorePathRequest(_message.Message):
    __slots__ = ("repository_id", "intent_id", "path", "level", "group_dir")
    REPOSITORY_ID_FIELD_NUMBER: _ClassVar[int]
    INTENT_ID_FIELD_NUMBER: _ClassVar[int]
    PATH_FIELD_NUMBER: _ClassVar[int]
    LEVEL_FIELD_NUMBER: _ClassVar[int]
    GROUP_DIR_FIELD_NUMBER: _ClassVar[int]
    repository_id: str
    intent_id: str
    path: str
    level: str
    group_dir: str
    def __init__(self, repository_id: _Optional[str] = ..., intent_id: _Optional[str] = ..., path: _Optional[str] = ..., level: _Optional[str] = ..., group_dir: _Optional[str] = ...) -> None: ...

class IgnorePathResponse(_message.Message):
    __slots__ = ("success", "ignored", "failed", "errors", "gitignore_path", "timestamp")
    SUCCESS_FIELD_NUMBER: _ClassVar[int]
    IGNORED_FIELD_NUMBER: _ClassVar[int]
    FAILED_FIELD_NUMBER: _ClassVar[int]
    ERRORS_FIELD_NUMBER: _ClassVar[int]
    GITIGNORE_PATH_FIELD_NUMBER: _ClassVar[int]
    TIMESTAMP_FIELD_NUMBER: _ClassVar[int]
    success: bool
    ignored: _containers.RepeatedScalarFieldContainer[str]
    failed: _containers.RepeatedScalarFieldContainer[str]
    errors: _containers.RepeatedScalarFieldContainer[str]
    gitignore_path: str
    timestamp: str
    def __init__(self, success: _Optional[bool] = ..., ignored: _Optional[_Iterable[str]] = ..., failed: _Optional[_Iterable[str]] = ..., errors: _Optional[_Iterable[str]] = ..., gitignore_path: _Optional[str] = ..., timestamp: _Optional[str] = ...) -> None: ...

class PushToRemoteRequest(_message.Message):
    __slots__ = ("repository_id", "intent_id", "remote", "branch", "set_upstream")
    REPOSITORY_ID_FIELD_NUMBER: _ClassVar[int]
    INTENT_ID_FIELD_NUMBER: _ClassVar[int]
    REMOTE_FIELD_NUMBER: _ClassVar[int]
    BRANCH_FIELD_NUMBER: _ClassVar[int]
    SET_UPSTREAM_FIELD_NUMBER: _ClassVar[int]
    repository_id: str
    intent_id: str
    remote: str
    branch: str
    set_upstream: bool
    def __init__(self, repository_id: _Optional[str] = ..., intent_id: _Optional[str] = ..., remote: _Optional[str] = ..., branch: _Optional[str] = ..., set_upstream: _Optional[bool] = ...) -> None: ...

class PushToRemoteResponse(_message.Message):
    __slots__ = ("success", "remote", "branch", "pushed", "up_to_date", "verified", "verification_error", "error", "timestamp")
    SUCCESS_FIELD_NUMBER: _ClassVar[int]
    REMOTE_FIELD_NUMBER: _ClassVar[int]
    BRANCH_FIELD_NUMBER: _ClassVar[int]
    PUSHED_FIELD_NUMBER: _ClassVar[int]
    UP_TO_DATE_FIELD_NUMBER: _ClassVar[int]
    VERIFIED_FIELD_NUMBER: _ClassVar[int]
    VERIFICATION_ERROR_FIELD_NUMBER: _ClassVar[int]
    ERROR_FIELD_NUMBER: _ClassVar[int]
    TIMESTAMP_FIELD_NUMBER: _ClassVar[int]
    success: bool
    remote: str
    branch: str
    pushed: bool
    up_to_date: bool
    verified: bool
    verification_error: str
    error: str
    timestamp: str
    def __init__(self, success: _Optional[bool] = ..., remote: _Optional[str] = ..., branch: _Optional[str] = ..., pushed: _Optional[bool] = ..., up_to_date: _Optional[bool] = ..., verified: _Optional[bool] = ..., verification_error: _Optional[str] = ..., error: _Optional[str] = ..., timestamp: _Optional[str] = ...) -> None: ...

class PullFromRemoteRequest(_message.Message):
    __slots__ = ("repository_id", "intent_id", "remote", "branch")
    REPOSITORY_ID_FIELD_NUMBER: _ClassVar[int]
    INTENT_ID_FIELD_NUMBER: _ClassVar[int]
    REMOTE_FIELD_NUMBER: _ClassVar[int]
    BRANCH_FIELD_NUMBER: _ClassVar[int]
    repository_id: str
    intent_id: str
    remote: str
    branch: str
    def __init__(self, repository_id: _Optional[str] = ..., intent_id: _Optional[str] = ..., remote: _Optional[str] = ..., branch: _Optional[str] = ...) -> None: ...

class PullFromRemoteResponse(_message.Message):
    __slots__ = ("success", "remote", "branch", "error", "has_conflicts", "timestamp")
    SUCCESS_FIELD_NUMBER: _ClassVar[int]
    REMOTE_FIELD_NUMBER: _ClassVar[int]
    BRANCH_FIELD_NUMBER: _ClassVar[int]
    ERROR_FIELD_NUMBER: _ClassVar[int]
    HAS_CONFLICTS_FIELD_NUMBER: _ClassVar[int]
    TIMESTAMP_FIELD_NUMBER: _ClassVar[int]
    success: bool
    remote: str
    branch: str
    error: str
    has_conflicts: bool
    timestamp: str
    def __init__(self, success: _Optional[bool] = ..., remote: _Optional[str] = ..., branch: _Optional[str] = ..., error: _Optional[str] = ..., has_conflicts: _Optional[bool] = ..., timestamp: _Optional[str] = ...) -> None: ...

class RunUpstreamActionRequest(_message.Message):
    __slots__ = ("repository_id", "intent_id", "action", "remote", "branch", "upstream")
    REPOSITORY_ID_FIELD_NUMBER: _ClassVar[int]
    INTENT_ID_FIELD_NUMBER: _ClassVar[int]
    ACTION_FIELD_NUMBER: _ClassVar[int]
    REMOTE_FIELD_NUMBER: _ClassVar[int]
    BRANCH_FIELD_NUMBER: _ClassVar[int]
    UPSTREAM_FIELD_NUMBER: _ClassVar[int]
    repository_id: str
    intent_id: str
    action: str
    remote: str
    branch: str
    upstream: str
    def __init__(self, repository_id: _Optional[str] = ..., intent_id: _Optional[str] = ..., action: _Optional[str] = ..., remote: _Optional[str] = ..., branch: _Optional[str] = ..., upstream: _Optional[str] = ...) -> None: ...

class RunUpstreamActionResponse(_message.Message):
    __slots__ = ("success", "action", "remote", "branch", "upstream", "error", "timestamp")
    SUCCESS_FIELD_NUMBER: _ClassVar[int]
    ACTION_FIELD_NUMBER: _ClassVar[int]
    REMOTE_FIELD_NUMBER: _ClassVar[int]
    BRANCH_FIELD_NUMBER: _ClassVar[int]
    UPSTREAM_FIELD_NUMBER: _ClassVar[int]
    ERROR_FIELD_NUMBER: _ClassVar[int]
    TIMESTAMP_FIELD_NUMBER: _ClassVar[int]
    success: bool
    action: str
    remote: str
    branch: str
    upstream: str
    error: str
    timestamp: str
    def __init__(self, success: _Optional[bool] = ..., action: _Optional[str] = ..., remote: _Optional[str] = ..., branch: _Optional[str] = ..., upstream: _Optional[str] = ..., error: _Optional[str] = ..., timestamp: _Optional[str] = ...) -> None: ...

class GetGroupingRulesRequest(_message.Message):
    __slots__ = ("repository_id",)
    REPOSITORY_ID_FIELD_NUMBER: _ClassVar[int]
    repository_id: str
    def __init__(self, repository_id: _Optional[str] = ...) -> None: ...

class GroupingRule(_message.Message):
    __slots__ = ("id", "label", "prefixes", "mode")
    ID_FIELD_NUMBER: _ClassVar[int]
    LABEL_FIELD_NUMBER: _ClassVar[int]
    PREFIXES_FIELD_NUMBER: _ClassVar[int]
    MODE_FIELD_NUMBER: _ClassVar[int]
    id: str
    label: str
    prefixes: _containers.RepeatedScalarFieldContainer[str]
    mode: str
    def __init__(self, id: _Optional[str] = ..., label: _Optional[str] = ..., prefixes: _Optional[_Iterable[str]] = ..., mode: _Optional[str] = ...) -> None: ...

class GroupingRulesResponse(_message.Message):
    __slots__ = ("enabled", "rules")
    ENABLED_FIELD_NUMBER: _ClassVar[int]
    RULES_FIELD_NUMBER: _ClassVar[int]
    enabled: bool
    rules: _containers.RepeatedCompositeFieldContainer[GroupingRule]
    def __init__(self, enabled: _Optional[bool] = ..., rules: _Optional[_Iterable[_Union[GroupingRule, _Mapping]]] = ...) -> None: ...

class SaveGroupingRulesRequest(_message.Message):
    __slots__ = ("repository_id", "intent_id", "enabled", "rules")
    REPOSITORY_ID_FIELD_NUMBER: _ClassVar[int]
    INTENT_ID_FIELD_NUMBER: _ClassVar[int]
    ENABLED_FIELD_NUMBER: _ClassVar[int]
    RULES_FIELD_NUMBER: _ClassVar[int]
    repository_id: str
    intent_id: str
    enabled: bool
    rules: _containers.RepeatedCompositeFieldContainer[GroupingRule]
    def __init__(self, repository_id: _Optional[str] = ..., intent_id: _Optional[str] = ..., enabled: _Optional[bool] = ..., rules: _Optional[_Iterable[_Union[GroupingRule, _Mapping]]] = ...) -> None: ...

class GetGitignoreHealthRequest(_message.Message):
    __slots__ = ("repository_id",)
    REPOSITORY_ID_FIELD_NUMBER: _ClassVar[int]
    repository_id: str
    def __init__(self, repository_id: _Optional[str] = ...) -> None: ...

class GitignoreSuggestion(_message.Message):
    __slots__ = ("line", "pattern", "type", "group_label", "group_dir", "target_pattern", "has_gitignore")
    LINE_FIELD_NUMBER: _ClassVar[int]
    PATTERN_FIELD_NUMBER: _ClassVar[int]
    TYPE_FIELD_NUMBER: _ClassVar[int]
    GROUP_LABEL_FIELD_NUMBER: _ClassVar[int]
    GROUP_DIR_FIELD_NUMBER: _ClassVar[int]
    TARGET_PATTERN_FIELD_NUMBER: _ClassVar[int]
    HAS_GITIGNORE_FIELD_NUMBER: _ClassVar[int]
    line: int
    pattern: str
    type: str
    group_label: str
    group_dir: str
    target_pattern: str
    has_gitignore: bool
    def __init__(self, line: _Optional[int] = ..., pattern: _Optional[str] = ..., type: _Optional[str] = ..., group_label: _Optional[str] = ..., group_dir: _Optional[str] = ..., target_pattern: _Optional[str] = ..., has_gitignore: _Optional[bool] = ...) -> None: ...

class GitignoreHealthResponse(_message.Message):
    __slots__ = ("root_entry_count", "suggestions")
    ROOT_ENTRY_COUNT_FIELD_NUMBER: _ClassVar[int]
    SUGGESTIONS_FIELD_NUMBER: _ClassVar[int]
    root_entry_count: int
    suggestions: _containers.RepeatedCompositeFieldContainer[GitignoreSuggestion]
    def __init__(self, root_entry_count: _Optional[int] = ..., suggestions: _Optional[_Iterable[_Union[GitignoreSuggestion, _Mapping]]] = ...) -> None: ...

class MoveGitignoreEntryRequest(_message.Message):
    __slots__ = ("repository_id", "intent_id", "line", "pattern", "group_dir", "target_pattern")
    REPOSITORY_ID_FIELD_NUMBER: _ClassVar[int]
    INTENT_ID_FIELD_NUMBER: _ClassVar[int]
    LINE_FIELD_NUMBER: _ClassVar[int]
    PATTERN_FIELD_NUMBER: _ClassVar[int]
    GROUP_DIR_FIELD_NUMBER: _ClassVar[int]
    TARGET_PATTERN_FIELD_NUMBER: _ClassVar[int]
    repository_id: str
    intent_id: str
    line: int
    pattern: str
    group_dir: str
    target_pattern: str
    def __init__(self, repository_id: _Optional[str] = ..., intent_id: _Optional[str] = ..., line: _Optional[int] = ..., pattern: _Optional[str] = ..., group_dir: _Optional[str] = ..., target_pattern: _Optional[str] = ...) -> None: ...

class MoveGitignoreEntryResponse(_message.Message):
    __slots__ = ("success", "removed_from", "added_to", "error")
    SUCCESS_FIELD_NUMBER: _ClassVar[int]
    REMOVED_FROM_FIELD_NUMBER: _ClassVar[int]
    ADDED_TO_FIELD_NUMBER: _ClassVar[int]
    ERROR_FIELD_NUMBER: _ClassVar[int]
    success: bool
    removed_from: str
    added_to: str
    error: str
    def __init__(self, success: _Optional[bool] = ..., removed_from: _Optional[str] = ..., added_to: _Optional[str] = ..., error: _Optional[str] = ...) -> None: ...

class GetTrackedBinariesRequest(_message.Message):
    __slots__ = ("repository_id",)
    REPOSITORY_ID_FIELD_NUMBER: _ClassVar[int]
    repository_id: str
    def __init__(self, repository_id: _Optional[str] = ...) -> None: ...

class TrackedBinary(_message.Message):
    __slots__ = ("path", "bytes", "format", "owner_dir", "ignore_pattern", "already_ignored")
    PATH_FIELD_NUMBER: _ClassVar[int]
    BYTES_FIELD_NUMBER: _ClassVar[int]
    FORMAT_FIELD_NUMBER: _ClassVar[int]
    OWNER_DIR_FIELD_NUMBER: _ClassVar[int]
    IGNORE_PATTERN_FIELD_NUMBER: _ClassVar[int]
    ALREADY_IGNORED_FIELD_NUMBER: _ClassVar[int]
    path: str
    bytes: int
    format: str
    owner_dir: str
    ignore_pattern: str
    already_ignored: bool
    def __init__(self, path: _Optional[str] = ..., bytes: _Optional[int] = ..., format: _Optional[str] = ..., owner_dir: _Optional[str] = ..., ignore_pattern: _Optional[str] = ..., already_ignored: _Optional[bool] = ...) -> None: ...

class TrackedBinariesResponse(_message.Message):
    __slots__ = ("binaries", "total_bytes", "history_warning")
    BINARIES_FIELD_NUMBER: _ClassVar[int]
    TOTAL_BYTES_FIELD_NUMBER: _ClassVar[int]
    HISTORY_WARNING_FIELD_NUMBER: _ClassVar[int]
    binaries: _containers.RepeatedCompositeFieldContainer[TrackedBinary]
    total_bytes: int
    history_warning: str
    def __init__(self, binaries: _Optional[_Iterable[_Union[TrackedBinary, _Mapping]]] = ..., total_bytes: _Optional[int] = ..., history_warning: _Optional[str] = ...) -> None: ...

class UntrackBinaryRequest(_message.Message):
    __slots__ = ("repository_id", "intent_id", "path", "owner_dir", "ignore_pattern")
    REPOSITORY_ID_FIELD_NUMBER: _ClassVar[int]
    INTENT_ID_FIELD_NUMBER: _ClassVar[int]
    PATH_FIELD_NUMBER: _ClassVar[int]
    OWNER_DIR_FIELD_NUMBER: _ClassVar[int]
    IGNORE_PATTERN_FIELD_NUMBER: _ClassVar[int]
    repository_id: str
    intent_id: str
    path: str
    owner_dir: str
    ignore_pattern: str
    def __init__(self, repository_id: _Optional[str] = ..., intent_id: _Optional[str] = ..., path: _Optional[str] = ..., owner_dir: _Optional[str] = ..., ignore_pattern: _Optional[str] = ...) -> None: ...

class UntrackBinaryResponse(_message.Message):
    __slots__ = ("success", "removed_from_index", "ignore_added_to", "error")
    SUCCESS_FIELD_NUMBER: _ClassVar[int]
    REMOVED_FROM_INDEX_FIELD_NUMBER: _ClassVar[int]
    IGNORE_ADDED_TO_FIELD_NUMBER: _ClassVar[int]
    ERROR_FIELD_NUMBER: _ClassVar[int]
    success: bool
    removed_from_index: bool
    ignore_added_to: str
    error: str
    def __init__(self, success: _Optional[bool] = ..., removed_from_index: _Optional[bool] = ..., ignore_added_to: _Optional[str] = ..., error: _Optional[str] = ...) -> None: ...

class GetPrecommitConfigRequest(_message.Message):
    __slots__ = ("repository_id",)
    REPOSITORY_ID_FIELD_NUMBER: _ClassVar[int]
    repository_id: str
    def __init__(self, repository_id: _Optional[str] = ...) -> None: ...

class PrecommitHookState(_message.Message):
    __slots__ = ("status", "reason", "existing_kind", "existing_hook_preview", "path", "hooks_path", "installed_at")
    STATUS_FIELD_NUMBER: _ClassVar[int]
    REASON_FIELD_NUMBER: _ClassVar[int]
    EXISTING_KIND_FIELD_NUMBER: _ClassVar[int]
    EXISTING_HOOK_PREVIEW_FIELD_NUMBER: _ClassVar[int]
    PATH_FIELD_NUMBER: _ClassVar[int]
    HOOKS_PATH_FIELD_NUMBER: _ClassVar[int]
    INSTALLED_AT_FIELD_NUMBER: _ClassVar[int]
    status: str
    reason: str
    existing_kind: str
    existing_hook_preview: str
    path: str
    hooks_path: str
    installed_at: str
    def __init__(self, status: _Optional[str] = ..., reason: _Optional[str] = ..., existing_kind: _Optional[str] = ..., existing_hook_preview: _Optional[str] = ..., path: _Optional[str] = ..., hooks_path: _Optional[str] = ..., installed_at: _Optional[str] = ...) -> None: ...

class PrecommitConfigResponse(_message.Message):
    __slots__ = ("enabled", "command", "working_directory", "timeout_seconds", "run_before_commit", "allow_override", "last_result", "hook")
    ENABLED_FIELD_NUMBER: _ClassVar[int]
    COMMAND_FIELD_NUMBER: _ClassVar[int]
    WORKING_DIRECTORY_FIELD_NUMBER: _ClassVar[int]
    TIMEOUT_SECONDS_FIELD_NUMBER: _ClassVar[int]
    RUN_BEFORE_COMMIT_FIELD_NUMBER: _ClassVar[int]
    ALLOW_OVERRIDE_FIELD_NUMBER: _ClassVar[int]
    LAST_RESULT_FIELD_NUMBER: _ClassVar[int]
    HOOK_FIELD_NUMBER: _ClassVar[int]
    enabled: bool
    command: str
    working_directory: str
    timeout_seconds: int
    run_before_commit: bool
    allow_override: bool
    last_result: PrecommitRunResult
    hook: PrecommitHookState
    def __init__(self, enabled: _Optional[bool] = ..., command: _Optional[str] = ..., working_directory: _Optional[str] = ..., timeout_seconds: _Optional[int] = ..., run_before_commit: _Optional[bool] = ..., allow_override: _Optional[bool] = ..., last_result: _Optional[_Union[PrecommitRunResult, _Mapping]] = ..., hook: _Optional[_Union[PrecommitHookState, _Mapping]] = ...) -> None: ...

class SavePrecommitConfigRequest(_message.Message):
    __slots__ = ("repository_id", "intent_id", "enabled", "command", "working_directory", "timeout_seconds", "run_before_commit", "allow_override")
    REPOSITORY_ID_FIELD_NUMBER: _ClassVar[int]
    INTENT_ID_FIELD_NUMBER: _ClassVar[int]
    ENABLED_FIELD_NUMBER: _ClassVar[int]
    COMMAND_FIELD_NUMBER: _ClassVar[int]
    WORKING_DIRECTORY_FIELD_NUMBER: _ClassVar[int]
    TIMEOUT_SECONDS_FIELD_NUMBER: _ClassVar[int]
    RUN_BEFORE_COMMIT_FIELD_NUMBER: _ClassVar[int]
    ALLOW_OVERRIDE_FIELD_NUMBER: _ClassVar[int]
    repository_id: str
    intent_id: str
    enabled: bool
    command: str
    working_directory: str
    timeout_seconds: int
    run_before_commit: bool
    allow_override: bool
    def __init__(self, repository_id: _Optional[str] = ..., intent_id: _Optional[str] = ..., enabled: _Optional[bool] = ..., command: _Optional[str] = ..., working_directory: _Optional[str] = ..., timeout_seconds: _Optional[int] = ..., run_before_commit: _Optional[bool] = ..., allow_override: _Optional[bool] = ...) -> None: ...

class RunPrecommitRequest(_message.Message):
    __slots__ = ("repository_id", "command", "working_directory", "timeout_seconds")
    REPOSITORY_ID_FIELD_NUMBER: _ClassVar[int]
    COMMAND_FIELD_NUMBER: _ClassVar[int]
    WORKING_DIRECTORY_FIELD_NUMBER: _ClassVar[int]
    TIMEOUT_SECONDS_FIELD_NUMBER: _ClassVar[int]
    repository_id: str
    command: str
    working_directory: str
    timeout_seconds: int
    def __init__(self, repository_id: _Optional[str] = ..., command: _Optional[str] = ..., working_directory: _Optional[str] = ..., timeout_seconds: _Optional[int] = ...) -> None: ...

class PrecommitRunResponse(_message.Message):
    __slots__ = ("success", "result")
    SUCCESS_FIELD_NUMBER: _ClassVar[int]
    RESULT_FIELD_NUMBER: _ClassVar[int]
    success: bool
    result: PrecommitRunResult
    def __init__(self, success: _Optional[bool] = ..., result: _Optional[_Union[PrecommitRunResult, _Mapping]] = ...) -> None: ...

class StageFilesRequest(_message.Message):
    __slots__ = ("repository_id", "intent_id", "paths", "scope")
    REPOSITORY_ID_FIELD_NUMBER: _ClassVar[int]
    INTENT_ID_FIELD_NUMBER: _ClassVar[int]
    PATHS_FIELD_NUMBER: _ClassVar[int]
    SCOPE_FIELD_NUMBER: _ClassVar[int]
    repository_id: str
    intent_id: str
    paths: _containers.RepeatedScalarFieldContainer[str]
    scope: str
    def __init__(self, repository_id: _Optional[str] = ..., intent_id: _Optional[str] = ..., paths: _Optional[_Iterable[str]] = ..., scope: _Optional[str] = ...) -> None: ...

class StageFilesResponse(_message.Message):
    __slots__ = ("success", "staged", "unstaged", "failed", "errors", "warnings", "timestamp")
    SUCCESS_FIELD_NUMBER: _ClassVar[int]
    STAGED_FIELD_NUMBER: _ClassVar[int]
    UNSTAGED_FIELD_NUMBER: _ClassVar[int]
    FAILED_FIELD_NUMBER: _ClassVar[int]
    ERRORS_FIELD_NUMBER: _ClassVar[int]
    WARNINGS_FIELD_NUMBER: _ClassVar[int]
    TIMESTAMP_FIELD_NUMBER: _ClassVar[int]
    success: bool
    staged: _containers.RepeatedScalarFieldContainer[str]
    unstaged: _containers.RepeatedScalarFieldContainer[str]
    failed: _containers.RepeatedScalarFieldContainer[str]
    errors: _containers.RepeatedScalarFieldContainer[str]
    warnings: _containers.RepeatedScalarFieldContainer[str]
    timestamp: str
    def __init__(self, success: _Optional[bool] = ..., staged: _Optional[_Iterable[str]] = ..., unstaged: _Optional[_Iterable[str]] = ..., failed: _Optional[_Iterable[str]] = ..., errors: _Optional[_Iterable[str]] = ..., warnings: _Optional[_Iterable[str]] = ..., timestamp: _Optional[str] = ...) -> None: ...

class UnstageFilesRequest(_message.Message):
    __slots__ = ("repository_id", "intent_id", "paths", "scope")
    REPOSITORY_ID_FIELD_NUMBER: _ClassVar[int]
    INTENT_ID_FIELD_NUMBER: _ClassVar[int]
    PATHS_FIELD_NUMBER: _ClassVar[int]
    SCOPE_FIELD_NUMBER: _ClassVar[int]
    repository_id: str
    intent_id: str
    paths: _containers.RepeatedScalarFieldContainer[str]
    scope: str
    def __init__(self, repository_id: _Optional[str] = ..., intent_id: _Optional[str] = ..., paths: _Optional[_Iterable[str]] = ..., scope: _Optional[str] = ...) -> None: ...

class UnstageFilesResponse(_message.Message):
    __slots__ = ("success", "unstaged", "failed", "errors", "timestamp")
    SUCCESS_FIELD_NUMBER: _ClassVar[int]
    UNSTAGED_FIELD_NUMBER: _ClassVar[int]
    FAILED_FIELD_NUMBER: _ClassVar[int]
    ERRORS_FIELD_NUMBER: _ClassVar[int]
    TIMESTAMP_FIELD_NUMBER: _ClassVar[int]
    success: bool
    unstaged: _containers.RepeatedScalarFieldContainer[str]
    failed: _containers.RepeatedScalarFieldContainer[str]
    errors: _containers.RepeatedScalarFieldContainer[str]
    timestamp: str
    def __init__(self, success: _Optional[bool] = ..., unstaged: _Optional[_Iterable[str]] = ..., failed: _Optional[_Iterable[str]] = ..., errors: _Optional[_Iterable[str]] = ..., timestamp: _Optional[str] = ...) -> None: ...

class CreateCommitRequest(_message.Message):
    __slots__ = ("repository_id", "intent_id", "message", "validate_conventional", "amend", "author_name", "author_email", "skip_precommit_once")
    REPOSITORY_ID_FIELD_NUMBER: _ClassVar[int]
    INTENT_ID_FIELD_NUMBER: _ClassVar[int]
    MESSAGE_FIELD_NUMBER: _ClassVar[int]
    VALIDATE_CONVENTIONAL_FIELD_NUMBER: _ClassVar[int]
    AMEND_FIELD_NUMBER: _ClassVar[int]
    AUTHOR_NAME_FIELD_NUMBER: _ClassVar[int]
    AUTHOR_EMAIL_FIELD_NUMBER: _ClassVar[int]
    SKIP_PRECOMMIT_ONCE_FIELD_NUMBER: _ClassVar[int]
    repository_id: str
    intent_id: str
    message: str
    validate_conventional: bool
    amend: bool
    author_name: str
    author_email: str
    skip_precommit_once: bool
    def __init__(self, repository_id: _Optional[str] = ..., intent_id: _Optional[str] = ..., message: _Optional[str] = ..., validate_conventional: _Optional[bool] = ..., amend: _Optional[bool] = ..., author_name: _Optional[str] = ..., author_email: _Optional[str] = ..., skip_precommit_once: _Optional[bool] = ...) -> None: ...

class PrecommitRunResult(_message.Message):
    __slots__ = ("status", "command", "exit_code", "summary", "stdout", "stderr", "duration_ms", "override_allowed", "timestamp")
    STATUS_FIELD_NUMBER: _ClassVar[int]
    COMMAND_FIELD_NUMBER: _ClassVar[int]
    EXIT_CODE_FIELD_NUMBER: _ClassVar[int]
    SUMMARY_FIELD_NUMBER: _ClassVar[int]
    STDOUT_FIELD_NUMBER: _ClassVar[int]
    STDERR_FIELD_NUMBER: _ClassVar[int]
    DURATION_MS_FIELD_NUMBER: _ClassVar[int]
    OVERRIDE_ALLOWED_FIELD_NUMBER: _ClassVar[int]
    TIMESTAMP_FIELD_NUMBER: _ClassVar[int]
    status: str
    command: str
    exit_code: int
    summary: str
    stdout: str
    stderr: str
    duration_ms: int
    override_allowed: bool
    timestamp: str
    def __init__(self, status: _Optional[str] = ..., command: _Optional[str] = ..., exit_code: _Optional[int] = ..., summary: _Optional[str] = ..., stdout: _Optional[str] = ..., stderr: _Optional[str] = ..., duration_ms: _Optional[int] = ..., override_allowed: _Optional[bool] = ..., timestamp: _Optional[str] = ...) -> None: ...

class CreateCommitResponse(_message.Message):
    __slots__ = ("success", "hash", "message", "amended", "validation_errors", "error", "precommit", "timestamp")
    SUCCESS_FIELD_NUMBER: _ClassVar[int]
    HASH_FIELD_NUMBER: _ClassVar[int]
    MESSAGE_FIELD_NUMBER: _ClassVar[int]
    AMENDED_FIELD_NUMBER: _ClassVar[int]
    VALIDATION_ERRORS_FIELD_NUMBER: _ClassVar[int]
    ERROR_FIELD_NUMBER: _ClassVar[int]
    PRECOMMIT_FIELD_NUMBER: _ClassVar[int]
    TIMESTAMP_FIELD_NUMBER: _ClassVar[int]
    success: bool
    hash: str
    message: str
    amended: bool
    validation_errors: _containers.RepeatedScalarFieldContainer[str]
    error: str
    precommit: PrecommitRunResult
    timestamp: str
    def __init__(self, success: _Optional[bool] = ..., hash: _Optional[str] = ..., message: _Optional[str] = ..., amended: _Optional[bool] = ..., validation_errors: _Optional[_Iterable[str]] = ..., error: _Optional[str] = ..., precommit: _Optional[_Union[PrecommitRunResult, _Mapping]] = ..., timestamp: _Optional[str] = ...) -> None: ...

class Credential(_message.Message):
    __slots__ = ("id", "remote", "url", "type", "username", "token_masked", "ssh_key_path", "is_configured", "created_at", "updated_at")
    ID_FIELD_NUMBER: _ClassVar[int]
    REMOTE_FIELD_NUMBER: _ClassVar[int]
    URL_FIELD_NUMBER: _ClassVar[int]
    TYPE_FIELD_NUMBER: _ClassVar[int]
    USERNAME_FIELD_NUMBER: _ClassVar[int]
    TOKEN_MASKED_FIELD_NUMBER: _ClassVar[int]
    SSH_KEY_PATH_FIELD_NUMBER: _ClassVar[int]
    IS_CONFIGURED_FIELD_NUMBER: _ClassVar[int]
    CREATED_AT_FIELD_NUMBER: _ClassVar[int]
    UPDATED_AT_FIELD_NUMBER: _ClassVar[int]
    id: str
    remote: str
    url: str
    type: str
    username: str
    token_masked: str
    ssh_key_path: str
    is_configured: bool
    created_at: str
    updated_at: str
    def __init__(self, id: _Optional[str] = ..., remote: _Optional[str] = ..., url: _Optional[str] = ..., type: _Optional[str] = ..., username: _Optional[str] = ..., token_masked: _Optional[str] = ..., ssh_key_path: _Optional[str] = ..., is_configured: _Optional[bool] = ..., created_at: _Optional[str] = ..., updated_at: _Optional[str] = ...) -> None: ...

class ListCredentialsRequest(_message.Message):
    __slots__ = ("repository_id",)
    REPOSITORY_ID_FIELD_NUMBER: _ClassVar[int]
    repository_id: str
    def __init__(self, repository_id: _Optional[str] = ...) -> None: ...

class CredentialsListResponse(_message.Message):
    __slots__ = ("credentials", "timestamp")
    CREDENTIALS_FIELD_NUMBER: _ClassVar[int]
    TIMESTAMP_FIELD_NUMBER: _ClassVar[int]
    credentials: _containers.RepeatedCompositeFieldContainer[Credential]
    timestamp: str
    def __init__(self, credentials: _Optional[_Iterable[_Union[Credential, _Mapping]]] = ..., timestamp: _Optional[str] = ...) -> None: ...

class SaveCredentialRequest(_message.Message):
    __slots__ = ("repository_id", "intent_id", "remote", "url", "username", "token", "ssh_key_path")
    REPOSITORY_ID_FIELD_NUMBER: _ClassVar[int]
    INTENT_ID_FIELD_NUMBER: _ClassVar[int]
    REMOTE_FIELD_NUMBER: _ClassVar[int]
    URL_FIELD_NUMBER: _ClassVar[int]
    USERNAME_FIELD_NUMBER: _ClassVar[int]
    TOKEN_FIELD_NUMBER: _ClassVar[int]
    SSH_KEY_PATH_FIELD_NUMBER: _ClassVar[int]
    repository_id: str
    intent_id: str
    remote: str
    url: str
    username: str
    token: str
    ssh_key_path: str
    def __init__(self, repository_id: _Optional[str] = ..., intent_id: _Optional[str] = ..., remote: _Optional[str] = ..., url: _Optional[str] = ..., username: _Optional[str] = ..., token: _Optional[str] = ..., ssh_key_path: _Optional[str] = ...) -> None: ...

class CredentialSaveResponse(_message.Message):
    __slots__ = ("success", "credential", "error", "timestamp")
    SUCCESS_FIELD_NUMBER: _ClassVar[int]
    CREDENTIAL_FIELD_NUMBER: _ClassVar[int]
    ERROR_FIELD_NUMBER: _ClassVar[int]
    TIMESTAMP_FIELD_NUMBER: _ClassVar[int]
    success: bool
    credential: Credential
    error: str
    timestamp: str
    def __init__(self, success: _Optional[bool] = ..., credential: _Optional[_Union[Credential, _Mapping]] = ..., error: _Optional[str] = ..., timestamp: _Optional[str] = ...) -> None: ...

class DeleteCredentialRequest(_message.Message):
    __slots__ = ("repository_id", "intent_id", "id")
    REPOSITORY_ID_FIELD_NUMBER: _ClassVar[int]
    INTENT_ID_FIELD_NUMBER: _ClassVar[int]
    ID_FIELD_NUMBER: _ClassVar[int]
    repository_id: str
    intent_id: str
    id: str
    def __init__(self, repository_id: _Optional[str] = ..., intent_id: _Optional[str] = ..., id: _Optional[str] = ...) -> None: ...

class CredentialDeleteResponse(_message.Message):
    __slots__ = ("success", "error", "timestamp")
    SUCCESS_FIELD_NUMBER: _ClassVar[int]
    ERROR_FIELD_NUMBER: _ClassVar[int]
    TIMESTAMP_FIELD_NUMBER: _ClassVar[int]
    success: bool
    error: str
    timestamp: str
    def __init__(self, success: _Optional[bool] = ..., error: _Optional[str] = ..., timestamp: _Optional[str] = ...) -> None: ...

class TestCredentialRequest(_message.Message):
    __slots__ = ("repository_id", "remote", "use_stored")
    REPOSITORY_ID_FIELD_NUMBER: _ClassVar[int]
    REMOTE_FIELD_NUMBER: _ClassVar[int]
    USE_STORED_FIELD_NUMBER: _ClassVar[int]
    repository_id: str
    remote: str
    use_stored: bool
    def __init__(self, repository_id: _Optional[str] = ..., remote: _Optional[str] = ..., use_stored: _Optional[bool] = ...) -> None: ...

class CredentialTestResponse(_message.Message):
    __slots__ = ("success", "reachable", "authorized", "error", "timestamp")
    SUCCESS_FIELD_NUMBER: _ClassVar[int]
    REACHABLE_FIELD_NUMBER: _ClassVar[int]
    AUTHORIZED_FIELD_NUMBER: _ClassVar[int]
    ERROR_FIELD_NUMBER: _ClassVar[int]
    TIMESTAMP_FIELD_NUMBER: _ClassVar[int]
    success: bool
    reachable: bool
    authorized: bool
    error: str
    timestamp: str
    def __init__(self, success: _Optional[bool] = ..., reachable: _Optional[bool] = ..., authorized: _Optional[bool] = ..., error: _Optional[str] = ..., timestamp: _Optional[str] = ...) -> None: ...

class UpdateRemoteURLRequest(_message.Message):
    __slots__ = ("repository_id", "intent_id", "remote", "url")
    REPOSITORY_ID_FIELD_NUMBER: _ClassVar[int]
    INTENT_ID_FIELD_NUMBER: _ClassVar[int]
    REMOTE_FIELD_NUMBER: _ClassVar[int]
    URL_FIELD_NUMBER: _ClassVar[int]
    repository_id: str
    intent_id: str
    remote: str
    url: str
    def __init__(self, repository_id: _Optional[str] = ..., intent_id: _Optional[str] = ..., remote: _Optional[str] = ..., url: _Optional[str] = ...) -> None: ...

class RemoteURLUpdateResponse(_message.Message):
    __slots__ = ("success", "old_url", "new_url", "error", "timestamp")
    SUCCESS_FIELD_NUMBER: _ClassVar[int]
    OLD_URL_FIELD_NUMBER: _ClassVar[int]
    NEW_URL_FIELD_NUMBER: _ClassVar[int]
    ERROR_FIELD_NUMBER: _ClassVar[int]
    TIMESTAMP_FIELD_NUMBER: _ClassVar[int]
    success: bool
    old_url: str
    new_url: str
    error: str
    timestamp: str
    def __init__(self, success: _Optional[bool] = ..., old_url: _Optional[str] = ..., new_url: _Optional[str] = ..., error: _Optional[str] = ..., timestamp: _Optional[str] = ...) -> None: ...

class SSHKeyInfo(_message.Message):
    __slots__ = ("path", "filename", "type", "bits", "fingerprint", "comment", "created_at", "has_public")
    PATH_FIELD_NUMBER: _ClassVar[int]
    FILENAME_FIELD_NUMBER: _ClassVar[int]
    TYPE_FIELD_NUMBER: _ClassVar[int]
    BITS_FIELD_NUMBER: _ClassVar[int]
    FINGERPRINT_FIELD_NUMBER: _ClassVar[int]
    COMMENT_FIELD_NUMBER: _ClassVar[int]
    CREATED_AT_FIELD_NUMBER: _ClassVar[int]
    HAS_PUBLIC_FIELD_NUMBER: _ClassVar[int]
    path: str
    filename: str
    type: str
    bits: int
    fingerprint: str
    comment: str
    created_at: str
    has_public: bool
    def __init__(self, path: _Optional[str] = ..., filename: _Optional[str] = ..., type: _Optional[str] = ..., bits: _Optional[int] = ..., fingerprint: _Optional[str] = ..., comment: _Optional[str] = ..., created_at: _Optional[str] = ..., has_public: _Optional[bool] = ...) -> None: ...

class ListSSHKeysRequest(_message.Message):
    __slots__ = ()
    def __init__(self) -> None: ...

class SSHListKeysResponse(_message.Message):
    __slots__ = ("keys", "ssh_dir", "timestamp")
    KEYS_FIELD_NUMBER: _ClassVar[int]
    SSH_DIR_FIELD_NUMBER: _ClassVar[int]
    TIMESTAMP_FIELD_NUMBER: _ClassVar[int]
    keys: _containers.RepeatedCompositeFieldContainer[SSHKeyInfo]
    ssh_dir: str
    timestamp: str
    def __init__(self, keys: _Optional[_Iterable[_Union[SSHKeyInfo, _Mapping]]] = ..., ssh_dir: _Optional[str] = ..., timestamp: _Optional[str] = ...) -> None: ...

class GenerateSSHKeyRequest(_message.Message):
    __slots__ = ("repository_id", "intent_id", "type", "bits", "comment", "filename")
    REPOSITORY_ID_FIELD_NUMBER: _ClassVar[int]
    INTENT_ID_FIELD_NUMBER: _ClassVar[int]
    TYPE_FIELD_NUMBER: _ClassVar[int]
    BITS_FIELD_NUMBER: _ClassVar[int]
    COMMENT_FIELD_NUMBER: _ClassVar[int]
    FILENAME_FIELD_NUMBER: _ClassVar[int]
    repository_id: str
    intent_id: str
    type: str
    bits: int
    comment: str
    filename: str
    def __init__(self, repository_id: _Optional[str] = ..., intent_id: _Optional[str] = ..., type: _Optional[str] = ..., bits: _Optional[int] = ..., comment: _Optional[str] = ..., filename: _Optional[str] = ...) -> None: ...

class SSHGenerateKeyResponse(_message.Message):
    __slots__ = ("success", "key", "public_key", "error", "timestamp")
    SUCCESS_FIELD_NUMBER: _ClassVar[int]
    KEY_FIELD_NUMBER: _ClassVar[int]
    PUBLIC_KEY_FIELD_NUMBER: _ClassVar[int]
    ERROR_FIELD_NUMBER: _ClassVar[int]
    TIMESTAMP_FIELD_NUMBER: _ClassVar[int]
    success: bool
    key: SSHKeyInfo
    public_key: str
    error: str
    timestamp: str
    def __init__(self, success: _Optional[bool] = ..., key: _Optional[_Union[SSHKeyInfo, _Mapping]] = ..., public_key: _Optional[str] = ..., error: _Optional[str] = ..., timestamp: _Optional[str] = ...) -> None: ...

class GetSSHPublicKeyRequest(_message.Message):
    __slots__ = ("key_path",)
    KEY_PATH_FIELD_NUMBER: _ClassVar[int]
    key_path: str
    def __init__(self, key_path: _Optional[str] = ...) -> None: ...

class SSHGetPublicKeyResponse(_message.Message):
    __slots__ = ("success", "public_key", "fingerprint", "error", "timestamp")
    SUCCESS_FIELD_NUMBER: _ClassVar[int]
    PUBLIC_KEY_FIELD_NUMBER: _ClassVar[int]
    FINGERPRINT_FIELD_NUMBER: _ClassVar[int]
    ERROR_FIELD_NUMBER: _ClassVar[int]
    TIMESTAMP_FIELD_NUMBER: _ClassVar[int]
    success: bool
    public_key: str
    fingerprint: str
    error: str
    timestamp: str
    def __init__(self, success: _Optional[bool] = ..., public_key: _Optional[str] = ..., fingerprint: _Optional[str] = ..., error: _Optional[str] = ..., timestamp: _Optional[str] = ...) -> None: ...

class TestSSHConnectionRequest(_message.Message):
    __slots__ = ("key_path",)
    KEY_PATH_FIELD_NUMBER: _ClassVar[int]
    key_path: str
    def __init__(self, key_path: _Optional[str] = ...) -> None: ...

class SSHTestConnectionResponse(_message.Message):
    __slots__ = ("success", "status", "message", "hint", "github_user", "fingerprint", "latency_ms", "timestamp")
    SUCCESS_FIELD_NUMBER: _ClassVar[int]
    STATUS_FIELD_NUMBER: _ClassVar[int]
    MESSAGE_FIELD_NUMBER: _ClassVar[int]
    HINT_FIELD_NUMBER: _ClassVar[int]
    GITHUB_USER_FIELD_NUMBER: _ClassVar[int]
    FINGERPRINT_FIELD_NUMBER: _ClassVar[int]
    LATENCY_MS_FIELD_NUMBER: _ClassVar[int]
    TIMESTAMP_FIELD_NUMBER: _ClassVar[int]
    success: bool
    status: str
    message: str
    hint: str
    github_user: str
    fingerprint: str
    latency_ms: int
    timestamp: str
    def __init__(self, success: _Optional[bool] = ..., status: _Optional[str] = ..., message: _Optional[str] = ..., hint: _Optional[str] = ..., github_user: _Optional[str] = ..., fingerprint: _Optional[str] = ..., latency_ms: _Optional[int] = ..., timestamp: _Optional[str] = ...) -> None: ...

class DeleteSSHKeyRequest(_message.Message):
    __slots__ = ("repository_id", "intent_id", "key_path")
    REPOSITORY_ID_FIELD_NUMBER: _ClassVar[int]
    INTENT_ID_FIELD_NUMBER: _ClassVar[int]
    KEY_PATH_FIELD_NUMBER: _ClassVar[int]
    repository_id: str
    intent_id: str
    key_path: str
    def __init__(self, repository_id: _Optional[str] = ..., intent_id: _Optional[str] = ..., key_path: _Optional[str] = ...) -> None: ...

class SSHDeleteKeyResponse(_message.Message):
    __slots__ = ("success", "message", "error", "private_deleted", "public_deleted", "timestamp")
    SUCCESS_FIELD_NUMBER: _ClassVar[int]
    MESSAGE_FIELD_NUMBER: _ClassVar[int]
    ERROR_FIELD_NUMBER: _ClassVar[int]
    PRIVATE_DELETED_FIELD_NUMBER: _ClassVar[int]
    PUBLIC_DELETED_FIELD_NUMBER: _ClassVar[int]
    TIMESTAMP_FIELD_NUMBER: _ClassVar[int]
    success: bool
    message: str
    error: str
    private_deleted: bool
    public_deleted: bool
    timestamp: str
    def __init__(self, success: _Optional[bool] = ..., message: _Optional[str] = ..., error: _Optional[str] = ..., private_deleted: _Optional[bool] = ..., public_deleted: _Optional[bool] = ..., timestamp: _Optional[str] = ...) -> None: ...

class InspectPushSafetyRequest(_message.Message):
    __slots__ = ("repository_id", "remote", "branch")
    REPOSITORY_ID_FIELD_NUMBER: _ClassVar[int]
    REMOTE_FIELD_NUMBER: _ClassVar[int]
    BRANCH_FIELD_NUMBER: _ClassVar[int]
    repository_id: str
    remote: str
    branch: str
    def __init__(self, repository_id: _Optional[str] = ..., remote: _Optional[str] = ..., branch: _Optional[str] = ...) -> None: ...

class PushSafetyFile(_message.Message):
    __slots__ = ("oid", "bytes", "paths", "commits", "blocked")
    OID_FIELD_NUMBER: _ClassVar[int]
    BYTES_FIELD_NUMBER: _ClassVar[int]
    PATHS_FIELD_NUMBER: _ClassVar[int]
    COMMITS_FIELD_NUMBER: _ClassVar[int]
    BLOCKED_FIELD_NUMBER: _ClassVar[int]
    oid: str
    bytes: int
    paths: _containers.RepeatedScalarFieldContainer[str]
    commits: _containers.RepeatedScalarFieldContainer[str]
    blocked: bool
    def __init__(self, oid: _Optional[str] = ..., bytes: _Optional[int] = ..., paths: _Optional[_Iterable[str]] = ..., commits: _Optional[_Iterable[str]] = ..., blocked: _Optional[bool] = ...) -> None: ...

class PushSafetyReport(_message.Message):
    __slots__ = ("complete", "state", "reason", "head", "base", "remote", "branch", "limit", "commits", "files", "fingerprint", "can_prepare", "recovery_reason", "staged_complete", "staged_reason", "staged_files")
    COMPLETE_FIELD_NUMBER: _ClassVar[int]
    STATE_FIELD_NUMBER: _ClassVar[int]
    REASON_FIELD_NUMBER: _ClassVar[int]
    HEAD_FIELD_NUMBER: _ClassVar[int]
    BASE_FIELD_NUMBER: _ClassVar[int]
    REMOTE_FIELD_NUMBER: _ClassVar[int]
    BRANCH_FIELD_NUMBER: _ClassVar[int]
    LIMIT_FIELD_NUMBER: _ClassVar[int]
    COMMITS_FIELD_NUMBER: _ClassVar[int]
    FILES_FIELD_NUMBER: _ClassVar[int]
    FINGERPRINT_FIELD_NUMBER: _ClassVar[int]
    CAN_PREPARE_FIELD_NUMBER: _ClassVar[int]
    RECOVERY_REASON_FIELD_NUMBER: _ClassVar[int]
    STAGED_COMPLETE_FIELD_NUMBER: _ClassVar[int]
    STAGED_REASON_FIELD_NUMBER: _ClassVar[int]
    STAGED_FILES_FIELD_NUMBER: _ClassVar[int]
    complete: bool
    state: str
    reason: str
    head: str
    base: str
    remote: str
    branch: str
    limit: int
    commits: _containers.RepeatedScalarFieldContainer[str]
    files: _containers.RepeatedCompositeFieldContainer[PushSafetyFile]
    fingerprint: str
    can_prepare: bool
    recovery_reason: str
    staged_complete: bool
    staged_reason: str
    staged_files: _containers.RepeatedCompositeFieldContainer[PushSafetyFile]
    def __init__(self, complete: _Optional[bool] = ..., state: _Optional[str] = ..., reason: _Optional[str] = ..., head: _Optional[str] = ..., base: _Optional[str] = ..., remote: _Optional[str] = ..., branch: _Optional[str] = ..., limit: _Optional[int] = ..., commits: _Optional[_Iterable[str]] = ..., files: _Optional[_Iterable[_Union[PushSafetyFile, _Mapping]]] = ..., fingerprint: _Optional[str] = ..., can_prepare: _Optional[bool] = ..., recovery_reason: _Optional[str] = ..., staged_complete: _Optional[bool] = ..., staged_reason: _Optional[str] = ..., staged_files: _Optional[_Iterable[_Union[PushSafetyFile, _Mapping]]] = ...) -> None: ...

class PreparePushRecoveryRequest(_message.Message):
    __slots__ = ("repository_id", "intent_id", "remote", "branch", "fingerprint")
    REPOSITORY_ID_FIELD_NUMBER: _ClassVar[int]
    INTENT_ID_FIELD_NUMBER: _ClassVar[int]
    REMOTE_FIELD_NUMBER: _ClassVar[int]
    BRANCH_FIELD_NUMBER: _ClassVar[int]
    FINGERPRINT_FIELD_NUMBER: _ClassVar[int]
    repository_id: str
    intent_id: str
    remote: str
    branch: str
    fingerprint: str
    def __init__(self, repository_id: _Optional[str] = ..., intent_id: _Optional[str] = ..., remote: _Optional[str] = ..., branch: _Optional[str] = ..., fingerprint: _Optional[str] = ...) -> None: ...

class RecoveryCommitMapping(_message.Message):
    __slots__ = ("original", "replacement")
    ORIGINAL_FIELD_NUMBER: _ClassVar[int]
    REPLACEMENT_FIELD_NUMBER: _ClassVar[int]
    original: str
    replacement: str
    def __init__(self, original: _Optional[str] = ..., replacement: _Optional[str] = ...) -> None: ...

class PushRecoveryArtifact(_message.Message):
    __slots__ = ("state", "message", "fingerprint", "head", "base", "candidate", "original_bundle", "repaired_bundle", "mappings", "paths", "signatures_removed")
    STATE_FIELD_NUMBER: _ClassVar[int]
    MESSAGE_FIELD_NUMBER: _ClassVar[int]
    FINGERPRINT_FIELD_NUMBER: _ClassVar[int]
    HEAD_FIELD_NUMBER: _ClassVar[int]
    BASE_FIELD_NUMBER: _ClassVar[int]
    CANDIDATE_FIELD_NUMBER: _ClassVar[int]
    ORIGINAL_BUNDLE_FIELD_NUMBER: _ClassVar[int]
    REPAIRED_BUNDLE_FIELD_NUMBER: _ClassVar[int]
    MAPPINGS_FIELD_NUMBER: _ClassVar[int]
    PATHS_FIELD_NUMBER: _ClassVar[int]
    SIGNATURES_REMOVED_FIELD_NUMBER: _ClassVar[int]
    state: str
    message: str
    fingerprint: str
    head: str
    base: str
    candidate: str
    original_bundle: str
    repaired_bundle: str
    mappings: _containers.RepeatedCompositeFieldContainer[RecoveryCommitMapping]
    paths: _containers.RepeatedScalarFieldContainer[str]
    signatures_removed: bool
    def __init__(self, state: _Optional[str] = ..., message: _Optional[str] = ..., fingerprint: _Optional[str] = ..., head: _Optional[str] = ..., base: _Optional[str] = ..., candidate: _Optional[str] = ..., original_bundle: _Optional[str] = ..., repaired_bundle: _Optional[str] = ..., mappings: _Optional[_Iterable[_Union[RecoveryCommitMapping, _Mapping]]] = ..., paths: _Optional[_Iterable[str]] = ..., signatures_removed: _Optional[bool] = ...) -> None: ...

class GetPushRecoveryRequest(_message.Message):
    __slots__ = ("repository_id", "fingerprint")
    REPOSITORY_ID_FIELD_NUMBER: _ClassVar[int]
    FINGERPRINT_FIELD_NUMBER: _ClassVar[int]
    repository_id: str
    fingerprint: str
    def __init__(self, repository_id: _Optional[str] = ..., fingerprint: _Optional[str] = ...) -> None: ...
