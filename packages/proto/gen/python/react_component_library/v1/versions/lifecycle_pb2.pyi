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
    __slots__ = ("library_id", "version", "created_at", "released_at", "retired_at", "lifecycle_state", "gate_pass_count", "gate_fail_count", "test_runs", "test_pass_rate", "adoption_current", "adoption_peak", "file_count", "lines_of_code", "dependency_count")
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
    def __init__(self, library_id: _Optional[str] = ..., version: _Optional[str] = ..., created_at: _Optional[str] = ..., released_at: _Optional[str] = ..., retired_at: _Optional[str] = ..., lifecycle_state: _Optional[str] = ..., gate_pass_count: _Optional[int] = ..., gate_fail_count: _Optional[int] = ..., test_runs: _Optional[int] = ..., test_pass_rate: _Optional[float] = ..., adoption_current: _Optional[int] = ..., adoption_peak: _Optional[int] = ..., file_count: _Optional[int] = ..., lines_of_code: _Optional[int] = ..., dependency_count: _Optional[int] = ...) -> None: ...

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
    __slots__ = ("component_id", "version", "confirm")
    COMPONENT_ID_FIELD_NUMBER: _ClassVar[int]
    VERSION_FIELD_NUMBER: _ClassVar[int]
    CONFIRM_FIELD_NUMBER: _ClassVar[int]
    component_id: str
    version: str
    confirm: bool
    def __init__(self, component_id: _Optional[str] = ..., version: _Optional[str] = ..., confirm: _Optional[bool] = ...) -> None: ...

class VersionLifecycleResponse(_message.Message):
    __slots__ = ("version", "lifecycle_state")
    VERSION_FIELD_NUMBER: _ClassVar[int]
    LIFECYCLE_STATE_FIELD_NUMBER: _ClassVar[int]
    version: RetireCandidate
    lifecycle_state: str
    def __init__(self, version: _Optional[_Union[RetireCandidate, _Mapping]] = ..., lifecycle_state: _Optional[str] = ...) -> None: ...

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
    __slots__ = ("version", "eligible", "reason", "adoption_count", "dependency_count", "age_days")
    VERSION_FIELD_NUMBER: _ClassVar[int]
    ELIGIBLE_FIELD_NUMBER: _ClassVar[int]
    REASON_FIELD_NUMBER: _ClassVar[int]
    ADOPTION_COUNT_FIELD_NUMBER: _ClassVar[int]
    DEPENDENCY_COUNT_FIELD_NUMBER: _ClassVar[int]
    AGE_DAYS_FIELD_NUMBER: _ClassVar[int]
    version: RetireCandidate
    eligible: bool
    reason: str
    adoption_count: int
    dependency_count: int
    age_days: int
    def __init__(self, version: _Optional[_Union[RetireCandidate, _Mapping]] = ..., eligible: _Optional[bool] = ..., reason: _Optional[str] = ..., adoption_count: _Optional[int] = ..., dependency_count: _Optional[int] = ..., age_days: _Optional[int] = ...) -> None: ...

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
