from test_genie.v1.runs import runs_pb2 as _runs_pb2
from common.v1 import operations_pb2 as _operations_pb2
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
    __slots__ = ("run_id", "captured_at", "capture_profile", "tree_digest", "phase_set_digest", "descriptor_snapshot_ref", "descriptor_snapshot_digest", "descriptor_snapshot_schema_version", "evidence_tier", "source_scope", "source_stable")
    RUN_ID_FIELD_NUMBER: _ClassVar[int]
    CAPTURED_AT_FIELD_NUMBER: _ClassVar[int]
    CAPTURE_PROFILE_FIELD_NUMBER: _ClassVar[int]
    TREE_DIGEST_FIELD_NUMBER: _ClassVar[int]
    PHASE_SET_DIGEST_FIELD_NUMBER: _ClassVar[int]
    DESCRIPTOR_SNAPSHOT_REF_FIELD_NUMBER: _ClassVar[int]
    DESCRIPTOR_SNAPSHOT_DIGEST_FIELD_NUMBER: _ClassVar[int]
    DESCRIPTOR_SNAPSHOT_SCHEMA_VERSION_FIELD_NUMBER: _ClassVar[int]
    EVIDENCE_TIER_FIELD_NUMBER: _ClassVar[int]
    SOURCE_SCOPE_FIELD_NUMBER: _ClassVar[int]
    SOURCE_STABLE_FIELD_NUMBER: _ClassVar[int]
    run_id: str
    captured_at: str
    capture_profile: str
    tree_digest: str
    phase_set_digest: str
    descriptor_snapshot_ref: str
    descriptor_snapshot_digest: str
    descriptor_snapshot_schema_version: int
    evidence_tier: str
    source_scope: str
    source_stable: bool
    def __init__(self, run_id: _Optional[str] = ..., captured_at: _Optional[str] = ..., capture_profile: _Optional[str] = ..., tree_digest: _Optional[str] = ..., phase_set_digest: _Optional[str] = ..., descriptor_snapshot_ref: _Optional[str] = ..., descriptor_snapshot_digest: _Optional[str] = ..., descriptor_snapshot_schema_version: _Optional[int] = ..., evidence_tier: _Optional[str] = ..., source_scope: _Optional[str] = ..., source_stable: _Optional[bool] = ...) -> None: ...

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
    __slots__ = ("base_run_id", "current_run_id", "base_catalog", "current_catalog", "visual_deltas", "degraded_reasons", "blocking_reasons", "advisory_warnings", "evidence_status")
    BASE_RUN_ID_FIELD_NUMBER: _ClassVar[int]
    CURRENT_RUN_ID_FIELD_NUMBER: _ClassVar[int]
    BASE_CATALOG_FIELD_NUMBER: _ClassVar[int]
    CURRENT_CATALOG_FIELD_NUMBER: _ClassVar[int]
    VISUAL_DELTAS_FIELD_NUMBER: _ClassVar[int]
    DEGRADED_REASONS_FIELD_NUMBER: _ClassVar[int]
    BLOCKING_REASONS_FIELD_NUMBER: _ClassVar[int]
    ADVISORY_WARNINGS_FIELD_NUMBER: _ClassVar[int]
    EVIDENCE_STATUS_FIELD_NUMBER: _ClassVar[int]
    base_run_id: str
    current_run_id: str
    base_catalog: RunArtifactCatalog
    current_catalog: RunArtifactCatalog
    visual_deltas: _containers.RepeatedCompositeFieldContainer[VisualDelta]
    degraded_reasons: _containers.RepeatedScalarFieldContainer[str]
    blocking_reasons: _containers.RepeatedScalarFieldContainer[str]
    advisory_warnings: _containers.RepeatedScalarFieldContainer[str]
    evidence_status: str
    def __init__(self, base_run_id: _Optional[str] = ..., current_run_id: _Optional[str] = ..., base_catalog: _Optional[_Union[RunArtifactCatalog, _Mapping]] = ..., current_catalog: _Optional[_Union[RunArtifactCatalog, _Mapping]] = ..., visual_deltas: _Optional[_Iterable[_Union[VisualDelta, _Mapping]]] = ..., degraded_reasons: _Optional[_Iterable[str]] = ..., blocking_reasons: _Optional[_Iterable[str]] = ..., advisory_warnings: _Optional[_Iterable[str]] = ..., evidence_status: _Optional[str] = ...) -> None: ...

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
    __slots__ = ("baseline", "current_git", "staleness", "verdict", "dirty_warning", "phases", "evidence", "comparison")
    BASELINE_FIELD_NUMBER: _ClassVar[int]
    CURRENT_GIT_FIELD_NUMBER: _ClassVar[int]
    STALENESS_FIELD_NUMBER: _ClassVar[int]
    VERDICT_FIELD_NUMBER: _ClassVar[int]
    DIRTY_WARNING_FIELD_NUMBER: _ClassVar[int]
    PHASES_FIELD_NUMBER: _ClassVar[int]
    EVIDENCE_FIELD_NUMBER: _ClassVar[int]
    COMPARISON_FIELD_NUMBER: _ClassVar[int]
    baseline: BaselineManifest
    current_git: GitState
    staleness: Staleness
    verdict: str
    dirty_warning: str
    phases: _containers.RepeatedCompositeFieldContainer[_runs_pb2.PhaseDiff]
    evidence: EvidenceComparison
    comparison: _runs_pb2.CompareRunsResponse
    def __init__(self, baseline: _Optional[_Union[BaselineManifest, _Mapping]] = ..., current_git: _Optional[_Union[GitState, _Mapping]] = ..., staleness: _Optional[_Union[Staleness, _Mapping]] = ..., verdict: _Optional[str] = ..., dirty_warning: _Optional[str] = ..., phases: _Optional[_Iterable[_Union[_runs_pb2.PhaseDiff, _Mapping]]] = ..., evidence: _Optional[_Union[EvidenceComparison, _Mapping]] = ..., comparison: _Optional[_Union[_runs_pb2.CompareRunsResponse, _Mapping]] = ...) -> None: ...

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

class RepairBaselineRequest(_message.Message):
    __slots__ = ("scenario", "name", "branch", "repo_id", "apply")
    SCENARIO_FIELD_NUMBER: _ClassVar[int]
    NAME_FIELD_NUMBER: _ClassVar[int]
    BRANCH_FIELD_NUMBER: _ClassVar[int]
    REPO_ID_FIELD_NUMBER: _ClassVar[int]
    APPLY_FIELD_NUMBER: _ClassVar[int]
    scenario: str
    name: str
    branch: str
    repo_id: int
    apply: bool
    def __init__(self, scenario: _Optional[str] = ..., name: _Optional[str] = ..., branch: _Optional[str] = ..., repo_id: _Optional[int] = ..., apply: _Optional[bool] = ...) -> None: ...

class RepairBaselineResponse(_message.Message):
    __slots__ = ("generation", "actions", "applied")
    GENERATION_FIELD_NUMBER: _ClassVar[int]
    ACTIONS_FIELD_NUMBER: _ClassVar[int]
    APPLIED_FIELD_NUMBER: _ClassVar[int]
    generation: int
    actions: _containers.RepeatedScalarFieldContainer[str]
    applied: bool
    def __init__(self, generation: _Optional[int] = ..., actions: _Optional[_Iterable[str]] = ..., applied: _Optional[bool] = ...) -> None: ...

class CollectionTarget(_message.Message):
    __slots__ = ("scenario", "baseline_name", "required")
    SCENARIO_FIELD_NUMBER: _ClassVar[int]
    BASELINE_NAME_FIELD_NUMBER: _ClassVar[int]
    REQUIRED_FIELD_NUMBER: _ClassVar[int]
    scenario: str
    baseline_name: str
    required: bool
    def __init__(self, scenario: _Optional[str] = ..., baseline_name: _Optional[str] = ..., required: _Optional[bool] = ...) -> None: ...

class CollectionMember(_message.Message):
    __slots__ = ("scenario", "baseline_name", "required", "status", "run_id", "error", "updated_at", "git_sha")
    SCENARIO_FIELD_NUMBER: _ClassVar[int]
    BASELINE_NAME_FIELD_NUMBER: _ClassVar[int]
    REQUIRED_FIELD_NUMBER: _ClassVar[int]
    STATUS_FIELD_NUMBER: _ClassVar[int]
    RUN_ID_FIELD_NUMBER: _ClassVar[int]
    ERROR_FIELD_NUMBER: _ClassVar[int]
    UPDATED_AT_FIELD_NUMBER: _ClassVar[int]
    GIT_SHA_FIELD_NUMBER: _ClassVar[int]
    scenario: str
    baseline_name: str
    required: bool
    status: str
    run_id: str
    error: str
    updated_at: str
    git_sha: str
    def __init__(self, scenario: _Optional[str] = ..., baseline_name: _Optional[str] = ..., required: _Optional[bool] = ..., status: _Optional[str] = ..., run_id: _Optional[str] = ..., error: _Optional[str] = ..., updated_at: _Optional[str] = ..., git_sha: _Optional[str] = ...) -> None: ...

class CollectionCoverage(_message.Message):
    __slots__ = ("required", "ready", "pending", "failed", "skipped", "stale", "complete")
    REQUIRED_FIELD_NUMBER: _ClassVar[int]
    READY_FIELD_NUMBER: _ClassVar[int]
    PENDING_FIELD_NUMBER: _ClassVar[int]
    FAILED_FIELD_NUMBER: _ClassVar[int]
    SKIPPED_FIELD_NUMBER: _ClassVar[int]
    STALE_FIELD_NUMBER: _ClassVar[int]
    COMPLETE_FIELD_NUMBER: _ClassVar[int]
    required: int
    ready: int
    pending: int
    failed: int
    skipped: int
    stale: int
    complete: bool
    def __init__(self, required: _Optional[int] = ..., ready: _Optional[int] = ..., pending: _Optional[int] = ..., failed: _Optional[int] = ..., skipped: _Optional[int] = ..., stale: _Optional[int] = ..., complete: _Optional[bool] = ...) -> None: ...

class BaselineCollection(_message.Message):
    __slots__ = ("name", "branch", "created_at", "updated_at", "schema_version", "members", "coverage", "path_snapshots", "reanchored", "reanchor_detail", "generation")
    NAME_FIELD_NUMBER: _ClassVar[int]
    BRANCH_FIELD_NUMBER: _ClassVar[int]
    CREATED_AT_FIELD_NUMBER: _ClassVar[int]
    UPDATED_AT_FIELD_NUMBER: _ClassVar[int]
    SCHEMA_VERSION_FIELD_NUMBER: _ClassVar[int]
    MEMBERS_FIELD_NUMBER: _ClassVar[int]
    COVERAGE_FIELD_NUMBER: _ClassVar[int]
    PATH_SNAPSHOTS_FIELD_NUMBER: _ClassVar[int]
    REANCHORED_FIELD_NUMBER: _ClassVar[int]
    REANCHOR_DETAIL_FIELD_NUMBER: _ClassVar[int]
    GENERATION_FIELD_NUMBER: _ClassVar[int]
    name: str
    branch: str
    created_at: str
    updated_at: str
    schema_version: int
    members: _containers.RepeatedCompositeFieldContainer[CollectionMember]
    coverage: CollectionCoverage
    path_snapshots: _containers.RepeatedCompositeFieldContainer[PathSnapshotReference]
    reanchored: bool
    reanchor_detail: str
    generation: int
    def __init__(self, name: _Optional[str] = ..., branch: _Optional[str] = ..., created_at: _Optional[str] = ..., updated_at: _Optional[str] = ..., schema_version: _Optional[int] = ..., members: _Optional[_Iterable[_Union[CollectionMember, _Mapping]]] = ..., coverage: _Optional[_Union[CollectionCoverage, _Mapping]] = ..., path_snapshots: _Optional[_Iterable[_Union[PathSnapshotReference, _Mapping]]] = ..., reanchored: _Optional[bool] = ..., reanchor_detail: _Optional[str] = ..., generation: _Optional[int] = ...) -> None: ...

class StartCollectionCaptureRequest(_message.Message):
    __slots__ = ("name", "branch", "targets", "created_by", "reason", "repo_id", "path_selections", "include_ignored", "retain_content", "acknowledge_reanchor")
    NAME_FIELD_NUMBER: _ClassVar[int]
    BRANCH_FIELD_NUMBER: _ClassVar[int]
    TARGETS_FIELD_NUMBER: _ClassVar[int]
    CREATED_BY_FIELD_NUMBER: _ClassVar[int]
    REASON_FIELD_NUMBER: _ClassVar[int]
    REPO_ID_FIELD_NUMBER: _ClassVar[int]
    PATH_SELECTIONS_FIELD_NUMBER: _ClassVar[int]
    INCLUDE_IGNORED_FIELD_NUMBER: _ClassVar[int]
    RETAIN_CONTENT_FIELD_NUMBER: _ClassVar[int]
    ACKNOWLEDGE_REANCHOR_FIELD_NUMBER: _ClassVar[int]
    name: str
    branch: str
    targets: _containers.RepeatedCompositeFieldContainer[CollectionTarget]
    created_by: str
    reason: str
    repo_id: int
    path_selections: _containers.RepeatedScalarFieldContainer[str]
    include_ignored: bool
    retain_content: bool
    acknowledge_reanchor: bool
    def __init__(self, name: _Optional[str] = ..., branch: _Optional[str] = ..., targets: _Optional[_Iterable[_Union[CollectionTarget, _Mapping]]] = ..., created_by: _Optional[str] = ..., reason: _Optional[str] = ..., repo_id: _Optional[int] = ..., path_selections: _Optional[_Iterable[str]] = ..., include_ignored: _Optional[bool] = ..., retain_content: _Optional[bool] = ..., acknowledge_reanchor: _Optional[bool] = ...) -> None: ...

class StartCollectionCaptureResponse(_message.Message):
    __slots__ = ("collection", "resumed")
    COLLECTION_FIELD_NUMBER: _ClassVar[int]
    RESUMED_FIELD_NUMBER: _ClassVar[int]
    collection: BaselineCollection
    resumed: bool
    def __init__(self, collection: _Optional[_Union[BaselineCollection, _Mapping]] = ..., resumed: _Optional[bool] = ...) -> None: ...

class GetCollectionStatusRequest(_message.Message):
    __slots__ = ("name", "branch", "repo_id")
    NAME_FIELD_NUMBER: _ClassVar[int]
    BRANCH_FIELD_NUMBER: _ClassVar[int]
    REPO_ID_FIELD_NUMBER: _ClassVar[int]
    name: str
    branch: str
    repo_id: int
    def __init__(self, name: _Optional[str] = ..., branch: _Optional[str] = ..., repo_id: _Optional[int] = ...) -> None: ...

class GetCollectionStatusResponse(_message.Message):
    __slots__ = ("collection", "standing")
    COLLECTION_FIELD_NUMBER: _ClassVar[int]
    STANDING_FIELD_NUMBER: _ClassVar[int]
    collection: BaselineCollection
    standing: _operations_pb2.OperationStanding
    def __init__(self, collection: _Optional[_Union[BaselineCollection, _Mapping]] = ..., standing: _Optional[_Union[_operations_pb2.OperationStanding, _Mapping]] = ...) -> None: ...

class WaitCollectionCaptureRequest(_message.Message):
    __slots__ = ("name", "branch", "repo_id", "timeout_seconds")
    NAME_FIELD_NUMBER: _ClassVar[int]
    BRANCH_FIELD_NUMBER: _ClassVar[int]
    REPO_ID_FIELD_NUMBER: _ClassVar[int]
    TIMEOUT_SECONDS_FIELD_NUMBER: _ClassVar[int]
    name: str
    branch: str
    repo_id: int
    timeout_seconds: int
    def __init__(self, name: _Optional[str] = ..., branch: _Optional[str] = ..., repo_id: _Optional[int] = ..., timeout_seconds: _Optional[int] = ...) -> None: ...

class WaitCollectionCaptureResponse(_message.Message):
    __slots__ = ("collection", "standing", "detached")
    COLLECTION_FIELD_NUMBER: _ClassVar[int]
    STANDING_FIELD_NUMBER: _ClassVar[int]
    DETACHED_FIELD_NUMBER: _ClassVar[int]
    collection: BaselineCollection
    standing: _operations_pb2.OperationStanding
    detached: bool
    def __init__(self, collection: _Optional[_Union[BaselineCollection, _Mapping]] = ..., standing: _Optional[_Union[_operations_pb2.OperationStanding, _Mapping]] = ..., detached: _Optional[bool] = ...) -> None: ...

class ExtendCollectionRequest(_message.Message):
    __slots__ = ("name", "branch", "targets", "created_by", "reason", "repo_id")
    NAME_FIELD_NUMBER: _ClassVar[int]
    BRANCH_FIELD_NUMBER: _ClassVar[int]
    TARGETS_FIELD_NUMBER: _ClassVar[int]
    CREATED_BY_FIELD_NUMBER: _ClassVar[int]
    REASON_FIELD_NUMBER: _ClassVar[int]
    REPO_ID_FIELD_NUMBER: _ClassVar[int]
    name: str
    branch: str
    targets: _containers.RepeatedCompositeFieldContainer[CollectionTarget]
    created_by: str
    reason: str
    repo_id: int
    def __init__(self, name: _Optional[str] = ..., branch: _Optional[str] = ..., targets: _Optional[_Iterable[_Union[CollectionTarget, _Mapping]]] = ..., created_by: _Optional[str] = ..., reason: _Optional[str] = ..., repo_id: _Optional[int] = ...) -> None: ...

class ExtendCollectionResponse(_message.Message):
    __slots__ = ("collection", "added_scenarios", "resumed")
    COLLECTION_FIELD_NUMBER: _ClassVar[int]
    ADDED_SCENARIOS_FIELD_NUMBER: _ClassVar[int]
    RESUMED_FIELD_NUMBER: _ClassVar[int]
    collection: BaselineCollection
    added_scenarios: _containers.RepeatedScalarFieldContainer[str]
    resumed: bool
    def __init__(self, collection: _Optional[_Union[BaselineCollection, _Mapping]] = ..., added_scenarios: _Optional[_Iterable[str]] = ..., resumed: _Optional[bool] = ...) -> None: ...

class CollectionDiffMember(_message.Message):
    __slots__ = ("scenario", "required", "status", "run_id", "verdict", "detail")
    SCENARIO_FIELD_NUMBER: _ClassVar[int]
    REQUIRED_FIELD_NUMBER: _ClassVar[int]
    STATUS_FIELD_NUMBER: _ClassVar[int]
    RUN_ID_FIELD_NUMBER: _ClassVar[int]
    VERDICT_FIELD_NUMBER: _ClassVar[int]
    DETAIL_FIELD_NUMBER: _ClassVar[int]
    scenario: str
    required: bool
    status: str
    run_id: str
    verdict: str
    detail: str
    def __init__(self, scenario: _Optional[str] = ..., required: _Optional[bool] = ..., status: _Optional[str] = ..., run_id: _Optional[str] = ..., verdict: _Optional[str] = ..., detail: _Optional[str] = ...) -> None: ...

class StartCollectionDiffRequest(_message.Message):
    __slots__ = ("name", "branch", "scenarios", "repo_id", "operation_id")
    NAME_FIELD_NUMBER: _ClassVar[int]
    BRANCH_FIELD_NUMBER: _ClassVar[int]
    SCENARIOS_FIELD_NUMBER: _ClassVar[int]
    REPO_ID_FIELD_NUMBER: _ClassVar[int]
    OPERATION_ID_FIELD_NUMBER: _ClassVar[int]
    name: str
    branch: str
    scenarios: _containers.RepeatedScalarFieldContainer[str]
    repo_id: int
    operation_id: str
    def __init__(self, name: _Optional[str] = ..., branch: _Optional[str] = ..., scenarios: _Optional[_Iterable[str]] = ..., repo_id: _Optional[int] = ..., operation_id: _Optional[str] = ...) -> None: ...

class StartCollectionDiffResponse(_message.Message):
    __slots__ = ("collection", "members", "classification", "operation_id")
    COLLECTION_FIELD_NUMBER: _ClassVar[int]
    MEMBERS_FIELD_NUMBER: _ClassVar[int]
    CLASSIFICATION_FIELD_NUMBER: _ClassVar[int]
    OPERATION_ID_FIELD_NUMBER: _ClassVar[int]
    collection: BaselineCollection
    members: _containers.RepeatedCompositeFieldContainer[CollectionDiffMember]
    classification: str
    operation_id: str
    def __init__(self, collection: _Optional[_Union[BaselineCollection, _Mapping]] = ..., members: _Optional[_Iterable[_Union[CollectionDiffMember, _Mapping]]] = ..., classification: _Optional[str] = ..., operation_id: _Optional[str] = ...) -> None: ...

class GetCollectionDiffStatusRequest(_message.Message):
    __slots__ = ("name", "branch", "operation_id", "repo_id")
    NAME_FIELD_NUMBER: _ClassVar[int]
    BRANCH_FIELD_NUMBER: _ClassVar[int]
    OPERATION_ID_FIELD_NUMBER: _ClassVar[int]
    REPO_ID_FIELD_NUMBER: _ClassVar[int]
    name: str
    branch: str
    operation_id: str
    repo_id: int
    def __init__(self, name: _Optional[str] = ..., branch: _Optional[str] = ..., operation_id: _Optional[str] = ..., repo_id: _Optional[int] = ...) -> None: ...

class GetCollectionDiffStatusResponse(_message.Message):
    __slots__ = ("collection", "members", "classification", "operation_id", "standing")
    COLLECTION_FIELD_NUMBER: _ClassVar[int]
    MEMBERS_FIELD_NUMBER: _ClassVar[int]
    CLASSIFICATION_FIELD_NUMBER: _ClassVar[int]
    OPERATION_ID_FIELD_NUMBER: _ClassVar[int]
    STANDING_FIELD_NUMBER: _ClassVar[int]
    collection: BaselineCollection
    members: _containers.RepeatedCompositeFieldContainer[CollectionDiffMember]
    classification: str
    operation_id: str
    standing: _operations_pb2.OperationStanding
    def __init__(self, collection: _Optional[_Union[BaselineCollection, _Mapping]] = ..., members: _Optional[_Iterable[_Union[CollectionDiffMember, _Mapping]]] = ..., classification: _Optional[str] = ..., operation_id: _Optional[str] = ..., standing: _Optional[_Union[_operations_pb2.OperationStanding, _Mapping]] = ...) -> None: ...

class WaitCollectionDiffRequest(_message.Message):
    __slots__ = ("name", "branch", "operation_id", "repo_id", "timeout_seconds")
    NAME_FIELD_NUMBER: _ClassVar[int]
    BRANCH_FIELD_NUMBER: _ClassVar[int]
    OPERATION_ID_FIELD_NUMBER: _ClassVar[int]
    REPO_ID_FIELD_NUMBER: _ClassVar[int]
    TIMEOUT_SECONDS_FIELD_NUMBER: _ClassVar[int]
    name: str
    branch: str
    operation_id: str
    repo_id: int
    timeout_seconds: int
    def __init__(self, name: _Optional[str] = ..., branch: _Optional[str] = ..., operation_id: _Optional[str] = ..., repo_id: _Optional[int] = ..., timeout_seconds: _Optional[int] = ...) -> None: ...

class WaitCollectionDiffResponse(_message.Message):
    __slots__ = ("collection", "members", "classification", "operation_id", "standing", "detached")
    COLLECTION_FIELD_NUMBER: _ClassVar[int]
    MEMBERS_FIELD_NUMBER: _ClassVar[int]
    CLASSIFICATION_FIELD_NUMBER: _ClassVar[int]
    OPERATION_ID_FIELD_NUMBER: _ClassVar[int]
    STANDING_FIELD_NUMBER: _ClassVar[int]
    DETACHED_FIELD_NUMBER: _ClassVar[int]
    collection: BaselineCollection
    members: _containers.RepeatedCompositeFieldContainer[CollectionDiffMember]
    classification: str
    operation_id: str
    standing: _operations_pb2.OperationStanding
    detached: bool
    def __init__(self, collection: _Optional[_Union[BaselineCollection, _Mapping]] = ..., members: _Optional[_Iterable[_Union[CollectionDiffMember, _Mapping]]] = ..., classification: _Optional[str] = ..., operation_id: _Optional[str] = ..., standing: _Optional[_Union[_operations_pb2.OperationStanding, _Mapping]] = ..., detached: _Optional[bool] = ...) -> None: ...

class DeleteCollectionRequest(_message.Message):
    __slots__ = ("name", "branch", "repo_id")
    NAME_FIELD_NUMBER: _ClassVar[int]
    BRANCH_FIELD_NUMBER: _ClassVar[int]
    REPO_ID_FIELD_NUMBER: _ClassVar[int]
    name: str
    branch: str
    repo_id: int
    def __init__(self, name: _Optional[str] = ..., branch: _Optional[str] = ..., repo_id: _Optional[int] = ...) -> None: ...

class DeleteCollectionResponse(_message.Message):
    __slots__ = ("deleted",)
    DELETED_FIELD_NUMBER: _ClassVar[int]
    deleted: bool
    def __init__(self, deleted: _Optional[bool] = ...) -> None: ...

class PathSnapshotReference(_message.Message):
    __slots__ = ("name", "branch", "created_at")
    NAME_FIELD_NUMBER: _ClassVar[int]
    BRANCH_FIELD_NUMBER: _ClassVar[int]
    CREATED_AT_FIELD_NUMBER: _ClassVar[int]
    name: str
    branch: str
    created_at: str
    def __init__(self, name: _Optional[str] = ..., branch: _Optional[str] = ..., created_at: _Optional[str] = ...) -> None: ...

class PathEntry(_message.Message):
    __slots__ = ("path", "mode", "type", "size", "digest", "state", "detail")
    PATH_FIELD_NUMBER: _ClassVar[int]
    MODE_FIELD_NUMBER: _ClassVar[int]
    TYPE_FIELD_NUMBER: _ClassVar[int]
    SIZE_FIELD_NUMBER: _ClassVar[int]
    DIGEST_FIELD_NUMBER: _ClassVar[int]
    STATE_FIELD_NUMBER: _ClassVar[int]
    DETAIL_FIELD_NUMBER: _ClassVar[int]
    path: str
    mode: int
    type: str
    size: int
    digest: str
    state: str
    detail: str
    def __init__(self, path: _Optional[str] = ..., mode: _Optional[int] = ..., type: _Optional[str] = ..., size: _Optional[int] = ..., digest: _Optional[str] = ..., state: _Optional[str] = ..., detail: _Optional[str] = ...) -> None: ...

class PathSnapshot(_message.Message):
    __slots__ = ("name", "branch", "created_at", "schema_version", "selections", "entries", "classification", "expires_at", "include_ignored", "retain_content", "policy_version")
    NAME_FIELD_NUMBER: _ClassVar[int]
    BRANCH_FIELD_NUMBER: _ClassVar[int]
    CREATED_AT_FIELD_NUMBER: _ClassVar[int]
    SCHEMA_VERSION_FIELD_NUMBER: _ClassVar[int]
    SELECTIONS_FIELD_NUMBER: _ClassVar[int]
    ENTRIES_FIELD_NUMBER: _ClassVar[int]
    CLASSIFICATION_FIELD_NUMBER: _ClassVar[int]
    EXPIRES_AT_FIELD_NUMBER: _ClassVar[int]
    INCLUDE_IGNORED_FIELD_NUMBER: _ClassVar[int]
    RETAIN_CONTENT_FIELD_NUMBER: _ClassVar[int]
    POLICY_VERSION_FIELD_NUMBER: _ClassVar[int]
    name: str
    branch: str
    created_at: str
    schema_version: int
    selections: _containers.RepeatedScalarFieldContainer[str]
    entries: _containers.RepeatedCompositeFieldContainer[PathEntry]
    classification: str
    expires_at: str
    include_ignored: bool
    retain_content: bool
    policy_version: int
    def __init__(self, name: _Optional[str] = ..., branch: _Optional[str] = ..., created_at: _Optional[str] = ..., schema_version: _Optional[int] = ..., selections: _Optional[_Iterable[str]] = ..., entries: _Optional[_Iterable[_Union[PathEntry, _Mapping]]] = ..., classification: _Optional[str] = ..., expires_at: _Optional[str] = ..., include_ignored: _Optional[bool] = ..., retain_content: _Optional[bool] = ..., policy_version: _Optional[int] = ...) -> None: ...

class PathSnapshotContributor(_message.Message):
    __slots__ = ("path", "files", "bytes")
    PATH_FIELD_NUMBER: _ClassVar[int]
    FILES_FIELD_NUMBER: _ClassVar[int]
    BYTES_FIELD_NUMBER: _ClassVar[int]
    path: str
    files: int
    bytes: int
    def __init__(self, path: _Optional[str] = ..., files: _Optional[int] = ..., bytes: _Optional[int] = ...) -> None: ...

class PathSnapshotIssue(_message.Message):
    __slots__ = ("code", "severity", "detail")
    CODE_FIELD_NUMBER: _ClassVar[int]
    SEVERITY_FIELD_NUMBER: _ClassVar[int]
    DETAIL_FIELD_NUMBER: _ClassVar[int]
    code: str
    severity: str
    detail: str
    def __init__(self, code: _Optional[str] = ..., severity: _Optional[str] = ..., detail: _Optional[str] = ...) -> None: ...

class PathSnapshotRecommendation(_message.Message):
    __slots__ = ("selection", "reason")
    SELECTION_FIELD_NUMBER: _ClassVar[int]
    REASON_FIELD_NUMBER: _ClassVar[int]
    selection: str
    reason: str
    def __init__(self, selection: _Optional[str] = ..., reason: _Optional[str] = ...) -> None: ...

class PathSnapshotEstimate(_message.Message):
    __slots__ = ("selections", "include_ignored", "retain_content", "eligible_files", "eligible_bytes", "excluded_ignored_files", "excluded_ignored_bytes", "excluded_sensitive_files", "excluded_binary_files", "oversized_files", "retained_content_bytes", "top_contributors", "issues", "recommendations", "repair_required", "policy_version")
    SELECTIONS_FIELD_NUMBER: _ClassVar[int]
    INCLUDE_IGNORED_FIELD_NUMBER: _ClassVar[int]
    RETAIN_CONTENT_FIELD_NUMBER: _ClassVar[int]
    ELIGIBLE_FILES_FIELD_NUMBER: _ClassVar[int]
    ELIGIBLE_BYTES_FIELD_NUMBER: _ClassVar[int]
    EXCLUDED_IGNORED_FILES_FIELD_NUMBER: _ClassVar[int]
    EXCLUDED_IGNORED_BYTES_FIELD_NUMBER: _ClassVar[int]
    EXCLUDED_SENSITIVE_FILES_FIELD_NUMBER: _ClassVar[int]
    EXCLUDED_BINARY_FILES_FIELD_NUMBER: _ClassVar[int]
    OVERSIZED_FILES_FIELD_NUMBER: _ClassVar[int]
    RETAINED_CONTENT_BYTES_FIELD_NUMBER: _ClassVar[int]
    TOP_CONTRIBUTORS_FIELD_NUMBER: _ClassVar[int]
    ISSUES_FIELD_NUMBER: _ClassVar[int]
    RECOMMENDATIONS_FIELD_NUMBER: _ClassVar[int]
    REPAIR_REQUIRED_FIELD_NUMBER: _ClassVar[int]
    POLICY_VERSION_FIELD_NUMBER: _ClassVar[int]
    selections: _containers.RepeatedScalarFieldContainer[str]
    include_ignored: bool
    retain_content: bool
    eligible_files: int
    eligible_bytes: int
    excluded_ignored_files: int
    excluded_ignored_bytes: int
    excluded_sensitive_files: int
    excluded_binary_files: int
    oversized_files: int
    retained_content_bytes: int
    top_contributors: _containers.RepeatedCompositeFieldContainer[PathSnapshotContributor]
    issues: _containers.RepeatedCompositeFieldContainer[PathSnapshotIssue]
    recommendations: _containers.RepeatedCompositeFieldContainer[PathSnapshotRecommendation]
    repair_required: bool
    policy_version: int
    def __init__(self, selections: _Optional[_Iterable[str]] = ..., include_ignored: _Optional[bool] = ..., retain_content: _Optional[bool] = ..., eligible_files: _Optional[int] = ..., eligible_bytes: _Optional[int] = ..., excluded_ignored_files: _Optional[int] = ..., excluded_ignored_bytes: _Optional[int] = ..., excluded_sensitive_files: _Optional[int] = ..., excluded_binary_files: _Optional[int] = ..., oversized_files: _Optional[int] = ..., retained_content_bytes: _Optional[int] = ..., top_contributors: _Optional[_Iterable[_Union[PathSnapshotContributor, _Mapping]]] = ..., issues: _Optional[_Iterable[_Union[PathSnapshotIssue, _Mapping]]] = ..., recommendations: _Optional[_Iterable[_Union[PathSnapshotRecommendation, _Mapping]]] = ..., repair_required: _Optional[bool] = ..., policy_version: _Optional[int] = ...) -> None: ...

class EstimatePathSnapshotRequest(_message.Message):
    __slots__ = ("selections", "repo_id", "include_ignored", "retain_content")
    SELECTIONS_FIELD_NUMBER: _ClassVar[int]
    REPO_ID_FIELD_NUMBER: _ClassVar[int]
    INCLUDE_IGNORED_FIELD_NUMBER: _ClassVar[int]
    RETAIN_CONTENT_FIELD_NUMBER: _ClassVar[int]
    selections: _containers.RepeatedScalarFieldContainer[str]
    repo_id: int
    include_ignored: bool
    retain_content: bool
    def __init__(self, selections: _Optional[_Iterable[str]] = ..., repo_id: _Optional[int] = ..., include_ignored: _Optional[bool] = ..., retain_content: _Optional[bool] = ...) -> None: ...

class EstimatePathSnapshotResponse(_message.Message):
    __slots__ = ("estimate",)
    ESTIMATE_FIELD_NUMBER: _ClassVar[int]
    estimate: PathSnapshotEstimate
    def __init__(self, estimate: _Optional[_Union[PathSnapshotEstimate, _Mapping]] = ...) -> None: ...

class PathSnapshotPolicyViolation(_message.Message):
    __slots__ = ("estimate",)
    ESTIMATE_FIELD_NUMBER: _ClassVar[int]
    estimate: PathSnapshotEstimate
    def __init__(self, estimate: _Optional[_Union[PathSnapshotEstimate, _Mapping]] = ...) -> None: ...

class SourceDelta(_message.Message):
    __slots__ = ("path", "status", "before", "after")
    PATH_FIELD_NUMBER: _ClassVar[int]
    STATUS_FIELD_NUMBER: _ClassVar[int]
    BEFORE_FIELD_NUMBER: _ClassVar[int]
    AFTER_FIELD_NUMBER: _ClassVar[int]
    path: str
    status: str
    before: PathEntry
    after: PathEntry
    def __init__(self, path: _Optional[str] = ..., status: _Optional[str] = ..., before: _Optional[_Union[PathEntry, _Mapping]] = ..., after: _Optional[_Union[PathEntry, _Mapping]] = ...) -> None: ...

class CapturePathSnapshotRequest(_message.Message):
    __slots__ = ("name", "branch", "selections", "repo_id", "retention_seconds", "include_ignored", "retain_content")
    NAME_FIELD_NUMBER: _ClassVar[int]
    BRANCH_FIELD_NUMBER: _ClassVar[int]
    SELECTIONS_FIELD_NUMBER: _ClassVar[int]
    REPO_ID_FIELD_NUMBER: _ClassVar[int]
    RETENTION_SECONDS_FIELD_NUMBER: _ClassVar[int]
    INCLUDE_IGNORED_FIELD_NUMBER: _ClassVar[int]
    RETAIN_CONTENT_FIELD_NUMBER: _ClassVar[int]
    name: str
    branch: str
    selections: _containers.RepeatedScalarFieldContainer[str]
    repo_id: int
    retention_seconds: int
    include_ignored: bool
    retain_content: bool
    def __init__(self, name: _Optional[str] = ..., branch: _Optional[str] = ..., selections: _Optional[_Iterable[str]] = ..., repo_id: _Optional[int] = ..., retention_seconds: _Optional[int] = ..., include_ignored: _Optional[bool] = ..., retain_content: _Optional[bool] = ...) -> None: ...

class CapturePathSnapshotResponse(_message.Message):
    __slots__ = ("snapshot", "resumed")
    SNAPSHOT_FIELD_NUMBER: _ClassVar[int]
    RESUMED_FIELD_NUMBER: _ClassVar[int]
    snapshot: PathSnapshot
    resumed: bool
    def __init__(self, snapshot: _Optional[_Union[PathSnapshot, _Mapping]] = ..., resumed: _Optional[bool] = ...) -> None: ...

class GetPathSnapshotRequest(_message.Message):
    __slots__ = ("name", "branch", "repo_id")
    NAME_FIELD_NUMBER: _ClassVar[int]
    BRANCH_FIELD_NUMBER: _ClassVar[int]
    REPO_ID_FIELD_NUMBER: _ClassVar[int]
    name: str
    branch: str
    repo_id: int
    def __init__(self, name: _Optional[str] = ..., branch: _Optional[str] = ..., repo_id: _Optional[int] = ...) -> None: ...

class GetPathSnapshotResponse(_message.Message):
    __slots__ = ("snapshot",)
    SNAPSHOT_FIELD_NUMBER: _ClassVar[int]
    snapshot: PathSnapshot
    def __init__(self, snapshot: _Optional[_Union[PathSnapshot, _Mapping]] = ...) -> None: ...

class DiffPathSnapshotsRequest(_message.Message):
    __slots__ = ("before_name", "after_name", "branch", "repo_id", "selections")
    BEFORE_NAME_FIELD_NUMBER: _ClassVar[int]
    AFTER_NAME_FIELD_NUMBER: _ClassVar[int]
    BRANCH_FIELD_NUMBER: _ClassVar[int]
    REPO_ID_FIELD_NUMBER: _ClassVar[int]
    SELECTIONS_FIELD_NUMBER: _ClassVar[int]
    before_name: str
    after_name: str
    branch: str
    repo_id: int
    selections: _containers.RepeatedScalarFieldContainer[str]
    def __init__(self, before_name: _Optional[str] = ..., after_name: _Optional[str] = ..., branch: _Optional[str] = ..., repo_id: _Optional[int] = ..., selections: _Optional[_Iterable[str]] = ...) -> None: ...

class DiffPathSnapshotsResponse(_message.Message):
    __slots__ = ("classification", "deltas")
    CLASSIFICATION_FIELD_NUMBER: _ClassVar[int]
    DELTAS_FIELD_NUMBER: _ClassVar[int]
    classification: str
    deltas: _containers.RepeatedCompositeFieldContainer[SourceDelta]
    def __init__(self, classification: _Optional[str] = ..., deltas: _Optional[_Iterable[_Union[SourceDelta, _Mapping]]] = ...) -> None: ...

class DeletePathSnapshotRequest(_message.Message):
    __slots__ = ("name", "branch", "repo_id")
    NAME_FIELD_NUMBER: _ClassVar[int]
    BRANCH_FIELD_NUMBER: _ClassVar[int]
    REPO_ID_FIELD_NUMBER: _ClassVar[int]
    name: str
    branch: str
    repo_id: int
    def __init__(self, name: _Optional[str] = ..., branch: _Optional[str] = ..., repo_id: _Optional[int] = ...) -> None: ...

class DeletePathSnapshotResponse(_message.Message):
    __slots__ = ("deleted",)
    DELETED_FIELD_NUMBER: _ClassVar[int]
    deleted: bool
    def __init__(self, deleted: _Optional[bool] = ...) -> None: ...
