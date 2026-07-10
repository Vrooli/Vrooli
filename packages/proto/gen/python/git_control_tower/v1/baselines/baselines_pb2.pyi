from test_genie.v1.runs import runs_pb2 as _runs_pb2
from google.protobuf.internal import containers as _containers
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Iterable as _Iterable, Mapping as _Mapping
from typing import ClassVar as _ClassVar, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class GitState(_message.Message):
    __slots__ = ("sha", "branch", "detached", "dirty", "dirty_summary", "commit_message", "commit_author", "commit_date", "sandboxed")
    SHA_FIELD_NUMBER: _ClassVar[int]
    BRANCH_FIELD_NUMBER: _ClassVar[int]
    DETACHED_FIELD_NUMBER: _ClassVar[int]
    DIRTY_FIELD_NUMBER: _ClassVar[int]
    DIRTY_SUMMARY_FIELD_NUMBER: _ClassVar[int]
    COMMIT_MESSAGE_FIELD_NUMBER: _ClassVar[int]
    COMMIT_AUTHOR_FIELD_NUMBER: _ClassVar[int]
    COMMIT_DATE_FIELD_NUMBER: _ClassVar[int]
    SANDBOXED_FIELD_NUMBER: _ClassVar[int]
    sha: str
    branch: str
    detached: bool
    dirty: bool
    dirty_summary: str
    commit_message: str
    commit_author: str
    commit_date: str
    sandboxed: bool
    def __init__(self, sha: _Optional[str] = ..., branch: _Optional[str] = ..., detached: _Optional[bool] = ..., dirty: _Optional[bool] = ..., dirty_summary: _Optional[str] = ..., commit_message: _Optional[str] = ..., commit_author: _Optional[str] = ..., commit_date: _Optional[str] = ..., sandboxed: _Optional[bool] = ...) -> None: ...

class RunAnchor(_message.Message):
    __slots__ = ("run_id", "captured_at", "capture_profile", "tree_digest", "phase_set_digest", "descriptor_snapshot_ref", "descriptor_snapshot_digest", "descriptor_snapshot_schema_version")
    RUN_ID_FIELD_NUMBER: _ClassVar[int]
    CAPTURED_AT_FIELD_NUMBER: _ClassVar[int]
    CAPTURE_PROFILE_FIELD_NUMBER: _ClassVar[int]
    TREE_DIGEST_FIELD_NUMBER: _ClassVar[int]
    PHASE_SET_DIGEST_FIELD_NUMBER: _ClassVar[int]
    DESCRIPTOR_SNAPSHOT_REF_FIELD_NUMBER: _ClassVar[int]
    DESCRIPTOR_SNAPSHOT_DIGEST_FIELD_NUMBER: _ClassVar[int]
    DESCRIPTOR_SNAPSHOT_SCHEMA_VERSION_FIELD_NUMBER: _ClassVar[int]
    run_id: str
    captured_at: str
    capture_profile: str
    tree_digest: str
    phase_set_digest: str
    descriptor_snapshot_ref: str
    descriptor_snapshot_digest: str
    descriptor_snapshot_schema_version: int
    def __init__(self, run_id: _Optional[str] = ..., captured_at: _Optional[str] = ..., capture_profile: _Optional[str] = ..., tree_digest: _Optional[str] = ..., phase_set_digest: _Optional[str] = ..., descriptor_snapshot_ref: _Optional[str] = ..., descriptor_snapshot_digest: _Optional[str] = ..., descriptor_snapshot_schema_version: _Optional[int] = ...) -> None: ...

class MigrationInfo(_message.Message):
    __slots__ = ("from_schema_version", "migrated_at", "degraded_reasons")
    FROM_SCHEMA_VERSION_FIELD_NUMBER: _ClassVar[int]
    MIGRATED_AT_FIELD_NUMBER: _ClassVar[int]
    DEGRADED_REASONS_FIELD_NUMBER: _ClassVar[int]
    from_schema_version: int
    migrated_at: str
    degraded_reasons: _containers.RepeatedScalarFieldContainer[str]
    def __init__(self, from_schema_version: _Optional[int] = ..., migrated_at: _Optional[str] = ..., degraded_reasons: _Optional[_Iterable[str]] = ...) -> None: ...

class BaselineManifest(_message.Message):
    __slots__ = ("name", "scenario", "branch", "created_at", "created_by", "git", "run", "schema_version", "migration")
    NAME_FIELD_NUMBER: _ClassVar[int]
    SCENARIO_FIELD_NUMBER: _ClassVar[int]
    BRANCH_FIELD_NUMBER: _ClassVar[int]
    CREATED_AT_FIELD_NUMBER: _ClassVar[int]
    CREATED_BY_FIELD_NUMBER: _ClassVar[int]
    GIT_FIELD_NUMBER: _ClassVar[int]
    RUN_FIELD_NUMBER: _ClassVar[int]
    SCHEMA_VERSION_FIELD_NUMBER: _ClassVar[int]
    MIGRATION_FIELD_NUMBER: _ClassVar[int]
    name: str
    scenario: str
    branch: str
    created_at: str
    created_by: str
    git: GitState
    run: RunAnchor
    schema_version: int
    migration: MigrationInfo
    def __init__(self, name: _Optional[str] = ..., scenario: _Optional[str] = ..., branch: _Optional[str] = ..., created_at: _Optional[str] = ..., created_by: _Optional[str] = ..., git: _Optional[_Union[GitState, _Mapping]] = ..., run: _Optional[_Union[RunAnchor, _Mapping]] = ..., schema_version: _Optional[int] = ..., migration: _Optional[_Union[MigrationInfo, _Mapping]] = ...) -> None: ...

class Staleness(_message.Message):
    __slots__ = ("commits_since", "files_changed", "likely_stale")
    COMMITS_SINCE_FIELD_NUMBER: _ClassVar[int]
    FILES_CHANGED_FIELD_NUMBER: _ClassVar[int]
    LIKELY_STALE_FIELD_NUMBER: _ClassVar[int]
    commits_since: int
    files_changed: int
    likely_stale: bool
    def __init__(self, commits_since: _Optional[int] = ..., files_changed: _Optional[int] = ..., likely_stale: _Optional[bool] = ...) -> None: ...

class RunArtifactCatalog(_message.Message):
    __slots__ = ("run_id", "schema_version", "digest", "artifacts", "legacy_discovered", "degraded_reasons")
    RUN_ID_FIELD_NUMBER: _ClassVar[int]
    SCHEMA_VERSION_FIELD_NUMBER: _ClassVar[int]
    DIGEST_FIELD_NUMBER: _ClassVar[int]
    ARTIFACTS_FIELD_NUMBER: _ClassVar[int]
    LEGACY_DISCOVERED_FIELD_NUMBER: _ClassVar[int]
    DEGRADED_REASONS_FIELD_NUMBER: _ClassVar[int]
    run_id: str
    schema_version: int
    digest: str
    artifacts: _containers.RepeatedCompositeFieldContainer[_runs_pb2.ArtifactRef]
    legacy_discovered: bool
    degraded_reasons: _containers.RepeatedScalarFieldContainer[str]
    def __init__(self, run_id: _Optional[str] = ..., schema_version: _Optional[int] = ..., digest: _Optional[str] = ..., artifacts: _Optional[_Iterable[_Union[_runs_pb2.ArtifactRef, _Mapping]]] = ..., legacy_discovered: _Optional[bool] = ..., degraded_reasons: _Optional[_Iterable[str]] = ...) -> None: ...

class VisualDelta(_message.Message):
    __slots__ = ("page", "label", "status", "changed_fraction")
    PAGE_FIELD_NUMBER: _ClassVar[int]
    LABEL_FIELD_NUMBER: _ClassVar[int]
    STATUS_FIELD_NUMBER: _ClassVar[int]
    CHANGED_FRACTION_FIELD_NUMBER: _ClassVar[int]
    page: str
    label: str
    status: str
    changed_fraction: float
    def __init__(self, page: _Optional[str] = ..., label: _Optional[str] = ..., status: _Optional[str] = ..., changed_fraction: _Optional[float] = ...) -> None: ...

class EvidenceComparison(_message.Message):
    __slots__ = ("base_run_id", "current_run_id", "base_catalog", "current_catalog", "visual_deltas", "degraded_reasons")
    BASE_RUN_ID_FIELD_NUMBER: _ClassVar[int]
    CURRENT_RUN_ID_FIELD_NUMBER: _ClassVar[int]
    BASE_CATALOG_FIELD_NUMBER: _ClassVar[int]
    CURRENT_CATALOG_FIELD_NUMBER: _ClassVar[int]
    VISUAL_DELTAS_FIELD_NUMBER: _ClassVar[int]
    DEGRADED_REASONS_FIELD_NUMBER: _ClassVar[int]
    base_run_id: str
    current_run_id: str
    base_catalog: RunArtifactCatalog
    current_catalog: RunArtifactCatalog
    visual_deltas: _containers.RepeatedCompositeFieldContainer[VisualDelta]
    degraded_reasons: _containers.RepeatedScalarFieldContainer[str]
    def __init__(self, base_run_id: _Optional[str] = ..., current_run_id: _Optional[str] = ..., base_catalog: _Optional[_Union[RunArtifactCatalog, _Mapping]] = ..., current_catalog: _Optional[_Union[RunArtifactCatalog, _Mapping]] = ..., visual_deltas: _Optional[_Iterable[_Union[VisualDelta, _Mapping]]] = ..., degraded_reasons: _Optional[_Iterable[str]] = ...) -> None: ...

class SnapshotForBaselineRequest(_message.Message):
    __slots__ = ("scenario", "name", "branch", "created_by", "reason", "repo_id")
    SCENARIO_FIELD_NUMBER: _ClassVar[int]
    NAME_FIELD_NUMBER: _ClassVar[int]
    BRANCH_FIELD_NUMBER: _ClassVar[int]
    CREATED_BY_FIELD_NUMBER: _ClassVar[int]
    REASON_FIELD_NUMBER: _ClassVar[int]
    REPO_ID_FIELD_NUMBER: _ClassVar[int]
    scenario: str
    name: str
    branch: str
    created_by: str
    reason: str
    repo_id: int
    def __init__(self, scenario: _Optional[str] = ..., name: _Optional[str] = ..., branch: _Optional[str] = ..., created_by: _Optional[str] = ..., reason: _Optional[str] = ..., repo_id: _Optional[int] = ...) -> None: ...

class SnapshotForBaselineResponse(_message.Message):
    __slots__ = ("run_id", "scenario", "name", "branch", "estimated_total_seconds", "eta_known", "dirty_warning", "coalesced")
    RUN_ID_FIELD_NUMBER: _ClassVar[int]
    SCENARIO_FIELD_NUMBER: _ClassVar[int]
    NAME_FIELD_NUMBER: _ClassVar[int]
    BRANCH_FIELD_NUMBER: _ClassVar[int]
    ESTIMATED_TOTAL_SECONDS_FIELD_NUMBER: _ClassVar[int]
    ETA_KNOWN_FIELD_NUMBER: _ClassVar[int]
    DIRTY_WARNING_FIELD_NUMBER: _ClassVar[int]
    COALESCED_FIELD_NUMBER: _ClassVar[int]
    run_id: str
    scenario: str
    name: str
    branch: str
    estimated_total_seconds: int
    eta_known: bool
    dirty_warning: str
    coalesced: bool
    def __init__(self, run_id: _Optional[str] = ..., scenario: _Optional[str] = ..., name: _Optional[str] = ..., branch: _Optional[str] = ..., estimated_total_seconds: _Optional[int] = ..., eta_known: _Optional[bool] = ..., dirty_warning: _Optional[str] = ..., coalesced: _Optional[bool] = ...) -> None: ...

class GetSnapshotStatusRequest(_message.Message):
    __slots__ = ("scenario", "name", "branch", "run_id", "repo_id", "wait")
    SCENARIO_FIELD_NUMBER: _ClassVar[int]
    NAME_FIELD_NUMBER: _ClassVar[int]
    BRANCH_FIELD_NUMBER: _ClassVar[int]
    RUN_ID_FIELD_NUMBER: _ClassVar[int]
    REPO_ID_FIELD_NUMBER: _ClassVar[int]
    WAIT_FIELD_NUMBER: _ClassVar[int]
    scenario: str
    name: str
    branch: str
    run_id: str
    repo_id: int
    wait: bool
    def __init__(self, scenario: _Optional[str] = ..., name: _Optional[str] = ..., branch: _Optional[str] = ..., run_id: _Optional[str] = ..., repo_id: _Optional[int] = ..., wait: _Optional[bool] = ...) -> None: ...

class GetSnapshotStatusResponse(_message.Message):
    __slots__ = ("status", "scenario", "name", "branch", "run_id", "run_status", "baseline", "error", "similar_baselines", "recommended_next_check_seconds")
    STATUS_FIELD_NUMBER: _ClassVar[int]
    SCENARIO_FIELD_NUMBER: _ClassVar[int]
    NAME_FIELD_NUMBER: _ClassVar[int]
    BRANCH_FIELD_NUMBER: _ClassVar[int]
    RUN_ID_FIELD_NUMBER: _ClassVar[int]
    RUN_STATUS_FIELD_NUMBER: _ClassVar[int]
    BASELINE_FIELD_NUMBER: _ClassVar[int]
    ERROR_FIELD_NUMBER: _ClassVar[int]
    SIMILAR_BASELINES_FIELD_NUMBER: _ClassVar[int]
    RECOMMENDED_NEXT_CHECK_SECONDS_FIELD_NUMBER: _ClassVar[int]
    status: str
    scenario: str
    name: str
    branch: str
    run_id: str
    run_status: str
    baseline: BaselineManifest
    error: str
    similar_baselines: _containers.RepeatedScalarFieldContainer[str]
    recommended_next_check_seconds: int
    def __init__(self, status: _Optional[str] = ..., scenario: _Optional[str] = ..., name: _Optional[str] = ..., branch: _Optional[str] = ..., run_id: _Optional[str] = ..., run_status: _Optional[str] = ..., baseline: _Optional[_Union[BaselineManifest, _Mapping]] = ..., error: _Optional[str] = ..., similar_baselines: _Optional[_Iterable[str]] = ..., recommended_next_check_seconds: _Optional[int] = ...) -> None: ...

class GetBaselineRequest(_message.Message):
    __slots__ = ("scenario", "name", "branch", "repo_id")
    SCENARIO_FIELD_NUMBER: _ClassVar[int]
    NAME_FIELD_NUMBER: _ClassVar[int]
    BRANCH_FIELD_NUMBER: _ClassVar[int]
    REPO_ID_FIELD_NUMBER: _ClassVar[int]
    scenario: str
    name: str
    branch: str
    repo_id: int
    def __init__(self, scenario: _Optional[str] = ..., name: _Optional[str] = ..., branch: _Optional[str] = ..., repo_id: _Optional[int] = ...) -> None: ...

class GetBaselineResponse(_message.Message):
    __slots__ = ("baseline",)
    BASELINE_FIELD_NUMBER: _ClassVar[int]
    baseline: BaselineManifest
    def __init__(self, baseline: _Optional[_Union[BaselineManifest, _Mapping]] = ...) -> None: ...

class ListBaselinesRequest(_message.Message):
    __slots__ = ("scenario", "branch", "all_branches", "repo_id")
    SCENARIO_FIELD_NUMBER: _ClassVar[int]
    BRANCH_FIELD_NUMBER: _ClassVar[int]
    ALL_BRANCHES_FIELD_NUMBER: _ClassVar[int]
    REPO_ID_FIELD_NUMBER: _ClassVar[int]
    scenario: str
    branch: str
    all_branches: bool
    repo_id: int
    def __init__(self, scenario: _Optional[str] = ..., branch: _Optional[str] = ..., all_branches: _Optional[bool] = ..., repo_id: _Optional[int] = ...) -> None: ...

class ListBaselinesResponse(_message.Message):
    __slots__ = ("baselines",)
    BASELINES_FIELD_NUMBER: _ClassVar[int]
    baselines: _containers.RepeatedCompositeFieldContainer[BaselineManifest]
    def __init__(self, baselines: _Optional[_Iterable[_Union[BaselineManifest, _Mapping]]] = ...) -> None: ...

class StartDiffRequest(_message.Message):
    __slots__ = ("scenario", "name", "branch", "repo_id")
    SCENARIO_FIELD_NUMBER: _ClassVar[int]
    NAME_FIELD_NUMBER: _ClassVar[int]
    BRANCH_FIELD_NUMBER: _ClassVar[int]
    REPO_ID_FIELD_NUMBER: _ClassVar[int]
    scenario: str
    name: str
    branch: str
    repo_id: int
    def __init__(self, scenario: _Optional[str] = ..., name: _Optional[str] = ..., branch: _Optional[str] = ..., repo_id: _Optional[int] = ...) -> None: ...

class StartDiffResponse(_message.Message):
    __slots__ = ("run_id", "scenario", "name", "branch", "estimated_total_seconds", "eta_known", "coalesced", "reused_run", "reused_sha", "dirty_warning")
    RUN_ID_FIELD_NUMBER: _ClassVar[int]
    SCENARIO_FIELD_NUMBER: _ClassVar[int]
    NAME_FIELD_NUMBER: _ClassVar[int]
    BRANCH_FIELD_NUMBER: _ClassVar[int]
    ESTIMATED_TOTAL_SECONDS_FIELD_NUMBER: _ClassVar[int]
    ETA_KNOWN_FIELD_NUMBER: _ClassVar[int]
    COALESCED_FIELD_NUMBER: _ClassVar[int]
    REUSED_RUN_FIELD_NUMBER: _ClassVar[int]
    REUSED_SHA_FIELD_NUMBER: _ClassVar[int]
    DIRTY_WARNING_FIELD_NUMBER: _ClassVar[int]
    run_id: str
    scenario: str
    name: str
    branch: str
    estimated_total_seconds: int
    eta_known: bool
    coalesced: bool
    reused_run: bool
    reused_sha: str
    dirty_warning: str
    def __init__(self, run_id: _Optional[str] = ..., scenario: _Optional[str] = ..., name: _Optional[str] = ..., branch: _Optional[str] = ..., estimated_total_seconds: _Optional[int] = ..., eta_known: _Optional[bool] = ..., coalesced: _Optional[bool] = ..., reused_run: _Optional[bool] = ..., reused_sha: _Optional[str] = ..., dirty_warning: _Optional[str] = ...) -> None: ...

class GetDiffResultRequest(_message.Message):
    __slots__ = ("scenario", "name", "branch", "run_id", "repo_id", "wait", "latest")
    SCENARIO_FIELD_NUMBER: _ClassVar[int]
    NAME_FIELD_NUMBER: _ClassVar[int]
    BRANCH_FIELD_NUMBER: _ClassVar[int]
    RUN_ID_FIELD_NUMBER: _ClassVar[int]
    REPO_ID_FIELD_NUMBER: _ClassVar[int]
    WAIT_FIELD_NUMBER: _ClassVar[int]
    LATEST_FIELD_NUMBER: _ClassVar[int]
    scenario: str
    name: str
    branch: str
    run_id: str
    repo_id: int
    wait: bool
    latest: bool
    def __init__(self, scenario: _Optional[str] = ..., name: _Optional[str] = ..., branch: _Optional[str] = ..., run_id: _Optional[str] = ..., repo_id: _Optional[int] = ..., wait: _Optional[bool] = ..., latest: _Optional[bool] = ...) -> None: ...

class GetDiffResultResponse(_message.Message):
    __slots__ = ("status", "diff", "error", "recommended_next_check_seconds", "run_id")
    STATUS_FIELD_NUMBER: _ClassVar[int]
    DIFF_FIELD_NUMBER: _ClassVar[int]
    ERROR_FIELD_NUMBER: _ClassVar[int]
    RECOMMENDED_NEXT_CHECK_SECONDS_FIELD_NUMBER: _ClassVar[int]
    RUN_ID_FIELD_NUMBER: _ClassVar[int]
    status: str
    diff: DiffResult
    error: str
    recommended_next_check_seconds: int
    run_id: str
    def __init__(self, status: _Optional[str] = ..., diff: _Optional[_Union[DiffResult, _Mapping]] = ..., error: _Optional[str] = ..., recommended_next_check_seconds: _Optional[int] = ..., run_id: _Optional[str] = ...) -> None: ...

class DiffResult(_message.Message):
    __slots__ = ("baseline", "current_git", "staleness", "verdict", "dirty_warning", "phases", "evidence")
    BASELINE_FIELD_NUMBER: _ClassVar[int]
    CURRENT_GIT_FIELD_NUMBER: _ClassVar[int]
    STALENESS_FIELD_NUMBER: _ClassVar[int]
    VERDICT_FIELD_NUMBER: _ClassVar[int]
    DIRTY_WARNING_FIELD_NUMBER: _ClassVar[int]
    PHASES_FIELD_NUMBER: _ClassVar[int]
    EVIDENCE_FIELD_NUMBER: _ClassVar[int]
    baseline: BaselineManifest
    current_git: GitState
    staleness: Staleness
    verdict: str
    dirty_warning: str
    phases: _containers.RepeatedCompositeFieldContainer[_runs_pb2.PhaseDiff]
    evidence: EvidenceComparison
    def __init__(self, baseline: _Optional[_Union[BaselineManifest, _Mapping]] = ..., current_git: _Optional[_Union[GitState, _Mapping]] = ..., staleness: _Optional[_Union[Staleness, _Mapping]] = ..., verdict: _Optional[str] = ..., dirty_warning: _Optional[str] = ..., phases: _Optional[_Iterable[_Union[_runs_pb2.PhaseDiff, _Mapping]]] = ..., evidence: _Optional[_Union[EvidenceComparison, _Mapping]] = ...) -> None: ...

class RunBusyInfo(_message.Message):
    __slots__ = ("scenario", "run_id", "preset")
    SCENARIO_FIELD_NUMBER: _ClassVar[int]
    RUN_ID_FIELD_NUMBER: _ClassVar[int]
    PRESET_FIELD_NUMBER: _ClassVar[int]
    scenario: str
    run_id: str
    preset: str
    def __init__(self, scenario: _Optional[str] = ..., run_id: _Optional[str] = ..., preset: _Optional[str] = ...) -> None: ...

class DeleteBaselineRequest(_message.Message):
    __slots__ = ("scenario", "name", "branch", "repo_id")
    SCENARIO_FIELD_NUMBER: _ClassVar[int]
    NAME_FIELD_NUMBER: _ClassVar[int]
    BRANCH_FIELD_NUMBER: _ClassVar[int]
    REPO_ID_FIELD_NUMBER: _ClassVar[int]
    scenario: str
    name: str
    branch: str
    repo_id: int
    def __init__(self, scenario: _Optional[str] = ..., name: _Optional[str] = ..., branch: _Optional[str] = ..., repo_id: _Optional[int] = ...) -> None: ...

class DeleteBaselineResponse(_message.Message):
    __slots__ = ("deleted",)
    DELETED_FIELD_NUMBER: _ClassVar[int]
    deleted: bool
    def __init__(self, deleted: _Optional[bool] = ...) -> None: ...
