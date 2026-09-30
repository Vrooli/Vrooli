from google.protobuf.internal import containers as _containers
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Iterable as _Iterable, Mapping as _Mapping
from typing import ClassVar as _ClassVar, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class ListVersionLedgerRequest(_message.Message):
    __slots__ = ("library_id", "window")
    LIBRARY_ID_FIELD_NUMBER: _ClassVar[int]
    WINDOW_FIELD_NUMBER: _ClassVar[int]
    library_id: str
    window: str
    def __init__(self, library_id: _Optional[str] = ..., window: _Optional[str] = ...) -> None: ...

class VersionLedgerRow(_message.Message):
    __slots__ = ("library_id", "version", "created_at", "released_at", "retired_at", "lifecycle_state", "gate_pass_count", "gate_fail_count", "test_runs", "test_pass_rate", "adoption_current", "adoption_peak", "file_count", "lines_of_code", "dependency_count", "presence")
    LIBRARY_ID_FIELD_NUMBER: _ClassVar[int]
    VERSION_FIELD_NUMBER: _ClassVar[int]
    CREATED_AT_FIELD_NUMBER: _ClassVar[int]
    RELEASED_AT_FIELD_NUMBER: _ClassVar[int]
    RETIRED_AT_FIELD_NUMBER: _ClassVar[int]
    LIFECYCLE_STATE_FIELD_NUMBER: _ClassVar[int]
    GATE_PASS_COUNT_FIELD_NUMBER: _ClassVar[int]
    GATE_FAIL_COUNT_FIELD_NUMBER: _ClassVar[int]
    TEST_RUNS_FIELD_NUMBER: _ClassVar[int]
    TEST_PASS_RATE_FIELD_NUMBER: _ClassVar[int]
    ADOPTION_CURRENT_FIELD_NUMBER: _ClassVar[int]
    ADOPTION_PEAK_FIELD_NUMBER: _ClassVar[int]
    FILE_COUNT_FIELD_NUMBER: _ClassVar[int]
    LINES_OF_CODE_FIELD_NUMBER: _ClassVar[int]
    DEPENDENCY_COUNT_FIELD_NUMBER: _ClassVar[int]
    PRESENCE_FIELD_NUMBER: _ClassVar[int]
    library_id: str
    version: str
    created_at: str
    released_at: str
    retired_at: str
    lifecycle_state: str
    gate_pass_count: int
    gate_fail_count: int
    test_runs: int
    test_pass_rate: float
    adoption_current: int
    adoption_peak: int
    file_count: int
    lines_of_code: int
    dependency_count: int
    presence: str
    def __init__(self, library_id: _Optional[str] = ..., version: _Optional[str] = ..., created_at: _Optional[str] = ..., released_at: _Optional[str] = ..., retired_at: _Optional[str] = ..., lifecycle_state: _Optional[str] = ..., gate_pass_count: _Optional[int] = ..., gate_fail_count: _Optional[int] = ..., test_runs: _Optional[int] = ..., test_pass_rate: _Optional[float] = ..., adoption_current: _Optional[int] = ..., adoption_peak: _Optional[int] = ..., file_count: _Optional[int] = ..., lines_of_code: _Optional[int] = ..., dependency_count: _Optional[int] = ..., presence: _Optional[str] = ...) -> None: ...

class ListVersionLedgerResponse(_message.Message):
    __slots__ = ("rows",)
    ROWS_FIELD_NUMBER: _ClassVar[int]
    rows: _containers.RepeatedCompositeFieldContainer[VersionLedgerRow]
    def __init__(self, rows: _Optional[_Iterable[_Union[VersionLedgerRow, _Mapping]]] = ...) -> None: ...

class ListRetireCandidatesRequest(_message.Message):
    __slots__ = ("component_id",)
    COMPONENT_ID_FIELD_NUMBER: _ClassVar[int]
    component_id: str
    def __init__(self, component_id: _Optional[str] = ...) -> None: ...

class RetireCandidate(_message.Message):
    __slots__ = ("component_id", "library_id", "version", "status")
    COMPONENT_ID_FIELD_NUMBER: _ClassVar[int]
    LIBRARY_ID_FIELD_NUMBER: _ClassVar[int]
    VERSION_FIELD_NUMBER: _ClassVar[int]
    STATUS_FIELD_NUMBER: _ClassVar[int]
    component_id: str
    library_id: str
    version: str
    status: str
    def __init__(self, component_id: _Optional[str] = ..., library_id: _Optional[str] = ..., version: _Optional[str] = ..., status: _Optional[str] = ...) -> None: ...

class ListRetireCandidatesResponse(_message.Message):
    __slots__ = ("candidates",)
    CANDIDATES_FIELD_NUMBER: _ClassVar[int]
    candidates: _containers.RepeatedCompositeFieldContainer[RetireCandidate]
    def __init__(self, candidates: _Optional[_Iterable[_Union[RetireCandidate, _Mapping]]] = ...) -> None: ...

class VersionLifecycleRequest(_message.Message):
    __slots__ = ("component_id", "version", "confirm", "plan_hash")
    COMPONENT_ID_FIELD_NUMBER: _ClassVar[int]
    VERSION_FIELD_NUMBER: _ClassVar[int]
    CONFIRM_FIELD_NUMBER: _ClassVar[int]
    PLAN_HASH_FIELD_NUMBER: _ClassVar[int]
    component_id: str
    version: str
    confirm: bool
    plan_hash: str
    def __init__(self, component_id: _Optional[str] = ..., version: _Optional[str] = ..., confirm: _Optional[bool] = ..., plan_hash: _Optional[str] = ...) -> None: ...

class VersionLifecycleResponse(_message.Message):
    __slots__ = ("version", "lifecycle_state")
    VERSION_FIELD_NUMBER: _ClassVar[int]
    LIFECYCLE_STATE_FIELD_NUMBER: _ClassVar[int]
    version: RetireCandidate
    lifecycle_state: str
    def __init__(self, version: _Optional[_Union[RetireCandidate, _Mapping]] = ..., lifecycle_state: _Optional[str] = ...) -> None: ...

class MaterializeVersionRequest(_message.Message):
    __slots__ = ("component_id", "version", "all", "into")
    COMPONENT_ID_FIELD_NUMBER: _ClassVar[int]
    VERSION_FIELD_NUMBER: _ClassVar[int]
    ALL_FIELD_NUMBER: _ClassVar[int]
    INTO_FIELD_NUMBER: _ClassVar[int]
    component_id: str
    version: str
    all: bool
    into: str
    def __init__(self, component_id: _Optional[str] = ..., version: _Optional[str] = ..., all: _Optional[bool] = ..., into: _Optional[str] = ...) -> None: ...

class MaterializedVersion(_message.Message):
    __slots__ = ("component_id", "library_id", "version", "directory", "files_written", "already_present")
    COMPONENT_ID_FIELD_NUMBER: _ClassVar[int]
    LIBRARY_ID_FIELD_NUMBER: _ClassVar[int]
    VERSION_FIELD_NUMBER: _ClassVar[int]
    DIRECTORY_FIELD_NUMBER: _ClassVar[int]
    FILES_WRITTEN_FIELD_NUMBER: _ClassVar[int]
    ALREADY_PRESENT_FIELD_NUMBER: _ClassVar[int]
    component_id: str
    library_id: str
    version: str
    directory: str
    files_written: int
    already_present: bool
    def __init__(self, component_id: _Optional[str] = ..., library_id: _Optional[str] = ..., version: _Optional[str] = ..., directory: _Optional[str] = ..., files_written: _Optional[int] = ..., already_present: _Optional[bool] = ...) -> None: ...

class MaterializeVersionResponse(_message.Message):
    __slots__ = ("versions",)
    VERSIONS_FIELD_NUMBER: _ClassVar[int]
    versions: _containers.RepeatedCompositeFieldContainer[MaterializedVersion]
    def __init__(self, versions: _Optional[_Iterable[_Union[MaterializedVersion, _Mapping]]] = ...) -> None: ...

class ReconcilePresenceRequest(_message.Message):
    __slots__ = ("component_id", "apply")
    COMPONENT_ID_FIELD_NUMBER: _ClassVar[int]
    APPLY_FIELD_NUMBER: _ClassVar[int]
    component_id: str
    apply: bool
    def __init__(self, component_id: _Optional[str] = ..., apply: _Optional[bool] = ...) -> None: ...

class ReconcilePresenceResponse(_message.Message):
    __slots__ = ("evict", "materialize", "unchanged", "applied")
    EVICT_FIELD_NUMBER: _ClassVar[int]
    MATERIALIZE_FIELD_NUMBER: _ClassVar[int]
    UNCHANGED_FIELD_NUMBER: _ClassVar[int]
    APPLIED_FIELD_NUMBER: _ClassVar[int]
    evict: _containers.RepeatedCompositeFieldContainer[RetireCandidate]
    materialize: _containers.RepeatedCompositeFieldContainer[RetireCandidate]
    unchanged: _containers.RepeatedCompositeFieldContainer[RetireCandidate]
    applied: bool
    def __init__(self, evict: _Optional[_Iterable[_Union[RetireCandidate, _Mapping]]] = ..., materialize: _Optional[_Iterable[_Union[RetireCandidate, _Mapping]]] = ..., unchanged: _Optional[_Iterable[_Union[RetireCandidate, _Mapping]]] = ..., applied: _Optional[bool] = ...) -> None: ...

class ArchiveRequest(_message.Message):
    __slots__ = ("path",)
    PATH_FIELD_NUMBER: _ClassVar[int]
    path: str
    def __init__(self, path: _Optional[str] = ...) -> None: ...

class ImportArchiveRequest(_message.Message):
    __slots__ = ("path", "overwrite")
    PATH_FIELD_NUMBER: _ClassVar[int]
    OVERWRITE_FIELD_NUMBER: _ClassVar[int]
    path: str
    overwrite: bool
    def __init__(self, path: _Optional[str] = ..., overwrite: _Optional[bool] = ...) -> None: ...

class ArchiveResponse(_message.Message):
    __slots__ = ("path", "schema_version", "row_counts", "checksum")
    class RowCountsEntry(_message.Message):
        __slots__ = ("key", "value")
        KEY_FIELD_NUMBER: _ClassVar[int]
        VALUE_FIELD_NUMBER: _ClassVar[int]
        key: str
        value: int
        def __init__(self, key: _Optional[str] = ..., value: _Optional[int] = ...) -> None: ...
    PATH_FIELD_NUMBER: _ClassVar[int]
    SCHEMA_VERSION_FIELD_NUMBER: _ClassVar[int]
    ROW_COUNTS_FIELD_NUMBER: _ClassVar[int]
    CHECKSUM_FIELD_NUMBER: _ClassVar[int]
    path: str
    schema_version: int
    row_counts: _containers.ScalarMap[str, int]
    checksum: str
    def __init__(self, path: _Optional[str] = ..., schema_version: _Optional[int] = ..., row_counts: _Optional[_Mapping[str, int]] = ..., checksum: _Optional[str] = ...) -> None: ...

class DoctorRequest(_message.Message):
    __slots__ = ()
    def __init__(self) -> None: ...

class DoctorIssue(_message.Message):
    __slots__ = ("library_id", "version", "path", "expected_sha256", "actual_sha256", "reason")
    LIBRARY_ID_FIELD_NUMBER: _ClassVar[int]
    VERSION_FIELD_NUMBER: _ClassVar[int]
    PATH_FIELD_NUMBER: _ClassVar[int]
    EXPECTED_SHA256_FIELD_NUMBER: _ClassVar[int]
    ACTUAL_SHA256_FIELD_NUMBER: _ClassVar[int]
    REASON_FIELD_NUMBER: _ClassVar[int]
    library_id: str
    version: str
    path: str
    expected_sha256: str
    actual_sha256: str
    reason: str
    def __init__(self, library_id: _Optional[str] = ..., version: _Optional[str] = ..., path: _Optional[str] = ..., expected_sha256: _Optional[str] = ..., actual_sha256: _Optional[str] = ..., reason: _Optional[str] = ...) -> None: ...

class DoctorResponse(_message.Message):
    __slots__ = ("issues",)
    ISSUES_FIELD_NUMBER: _ClassVar[int]
    issues: _containers.RepeatedCompositeFieldContainer[DoctorIssue]
    def __init__(self, issues: _Optional[_Iterable[_Union[DoctorIssue, _Mapping]]] = ...) -> None: ...

class CleanupScope(_message.Message):
    __slots__ = ("component_id", "library_id", "older_than_days")
    COMPONENT_ID_FIELD_NUMBER: _ClassVar[int]
    LIBRARY_ID_FIELD_NUMBER: _ClassVar[int]
    OLDER_THAN_DAYS_FIELD_NUMBER: _ClassVar[int]
    component_id: str
    library_id: str
    older_than_days: int
    def __init__(self, component_id: _Optional[str] = ..., library_id: _Optional[str] = ..., older_than_days: _Optional[int] = ...) -> None: ...

class CleanupItem(_message.Message):
    __slots__ = ("version", "eligible", "reason", "adoption_count", "dependency_count", "age_days", "references")
    VERSION_FIELD_NUMBER: _ClassVar[int]
    ELIGIBLE_FIELD_NUMBER: _ClassVar[int]
    REASON_FIELD_NUMBER: _ClassVar[int]
    ADOPTION_COUNT_FIELD_NUMBER: _ClassVar[int]
    DEPENDENCY_COUNT_FIELD_NUMBER: _ClassVar[int]
    AGE_DAYS_FIELD_NUMBER: _ClassVar[int]
    REFERENCES_FIELD_NUMBER: _ClassVar[int]
    version: RetireCandidate
    eligible: bool
    reason: str
    adoption_count: int
    dependency_count: int
    age_days: int
    references: _containers.RepeatedCompositeFieldContainer[VersionReference]
    def __init__(self, version: _Optional[_Union[RetireCandidate, _Mapping]] = ..., eligible: _Optional[bool] = ..., reason: _Optional[str] = ..., adoption_count: _Optional[int] = ..., dependency_count: _Optional[int] = ..., age_days: _Optional[int] = ..., references: _Optional[_Iterable[_Union[VersionReference, _Mapping]]] = ...) -> None: ...

class VersionReference(_message.Message):
    __slots__ = ("kind", "owner_library_id", "owner_version", "owner_path", "import_specifier", "evidence", "owner_scenario", "adoption_id")
    KIND_FIELD_NUMBER: _ClassVar[int]
    OWNER_LIBRARY_ID_FIELD_NUMBER: _ClassVar[int]
    OWNER_VERSION_FIELD_NUMBER: _ClassVar[int]
    OWNER_PATH_FIELD_NUMBER: _ClassVar[int]
    IMPORT_SPECIFIER_FIELD_NUMBER: _ClassVar[int]
    EVIDENCE_FIELD_NUMBER: _ClassVar[int]
    OWNER_SCENARIO_FIELD_NUMBER: _ClassVar[int]
    ADOPTION_ID_FIELD_NUMBER: _ClassVar[int]
    kind: str
    owner_library_id: str
    owner_version: str
    owner_path: str
    import_specifier: str
    evidence: str
    owner_scenario: str
    adoption_id: str
    def __init__(self, kind: _Optional[str] = ..., owner_library_id: _Optional[str] = ..., owner_version: _Optional[str] = ..., owner_path: _Optional[str] = ..., import_specifier: _Optional[str] = ..., evidence: _Optional[str] = ..., owner_scenario: _Optional[str] = ..., adoption_id: _Optional[str] = ...) -> None: ...

class PlanCleanupRequest(_message.Message):
    __slots__ = ("scope",)
    SCOPE_FIELD_NUMBER: _ClassVar[int]
    scope: CleanupScope
    def __init__(self, scope: _Optional[_Union[CleanupScope, _Mapping]] = ...) -> None: ...

class PlanCleanupResponse(_message.Message):
    __slots__ = ("items", "plan_hash")
    ITEMS_FIELD_NUMBER: _ClassVar[int]
    PLAN_HASH_FIELD_NUMBER: _ClassVar[int]
    items: _containers.RepeatedCompositeFieldContainer[CleanupItem]
    plan_hash: str
    def __init__(self, items: _Optional[_Iterable[_Union[CleanupItem, _Mapping]]] = ..., plan_hash: _Optional[str] = ...) -> None: ...

class CleanupVersionsRequest(_message.Message):
    __slots__ = ("scope", "plan_hash", "confirm")
    SCOPE_FIELD_NUMBER: _ClassVar[int]
    PLAN_HASH_FIELD_NUMBER: _ClassVar[int]
    CONFIRM_FIELD_NUMBER: _ClassVar[int]
    scope: CleanupScope
    plan_hash: str
    confirm: bool
    def __init__(self, scope: _Optional[_Union[CleanupScope, _Mapping]] = ..., plan_hash: _Optional[str] = ..., confirm: _Optional[bool] = ...) -> None: ...

class CleanupVersionsResponse(_message.Message):
    __slots__ = ("items", "plan_hash", "retired_count", "applied")
    ITEMS_FIELD_NUMBER: _ClassVar[int]
    PLAN_HASH_FIELD_NUMBER: _ClassVar[int]
    RETIRED_COUNT_FIELD_NUMBER: _ClassVar[int]
    APPLIED_FIELD_NUMBER: _ClassVar[int]
    items: _containers.RepeatedCompositeFieldContainer[CleanupItem]
    plan_hash: str
    retired_count: int
    applied: bool
    def __init__(self, items: _Optional[_Iterable[_Union[CleanupItem, _Mapping]]] = ..., plan_hash: _Optional[str] = ..., retired_count: _Optional[int] = ..., applied: _Optional[bool] = ...) -> None: ...

class CleanupDraftRequest(_message.Message):
    __slots__ = ("component_id", "older_than_days", "confirm")
    COMPONENT_ID_FIELD_NUMBER: _ClassVar[int]
    OLDER_THAN_DAYS_FIELD_NUMBER: _ClassVar[int]
    CONFIRM_FIELD_NUMBER: _ClassVar[int]
    component_id: str
    older_than_days: int
    confirm: bool
    def __init__(self, component_id: _Optional[str] = ..., older_than_days: _Optional[int] = ..., confirm: _Optional[bool] = ...) -> None: ...

class CleanupDraftResponse(_message.Message):
    __slots__ = ("item", "applied")
    ITEM_FIELD_NUMBER: _ClassVar[int]
    APPLIED_FIELD_NUMBER: _ClassVar[int]
    item: CleanupItem
    applied: bool
    def __init__(self, item: _Optional[_Union[CleanupItem, _Mapping]] = ..., applied: _Optional[bool] = ...) -> None: ...
