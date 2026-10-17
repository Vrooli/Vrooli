import datetime

from data_backup_manager.v1.sources import sources_pb2 as _sources_pb2
from google.protobuf import timestamp_pb2 as _timestamp_pb2
from google.protobuf.internal import containers as _containers
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Iterable as _Iterable, Mapping as _Mapping
from typing import ClassVar as _ClassVar, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class RegisteredTarget(_message.Message):
    __slots__ = ("id", "owner", "name", "source_kind", "locator", "planned", "last_success_at", "last_verified_at", "critical")
    ID_FIELD_NUMBER: _ClassVar[int]
    OWNER_FIELD_NUMBER: _ClassVar[int]
    NAME_FIELD_NUMBER: _ClassVar[int]
    SOURCE_KIND_FIELD_NUMBER: _ClassVar[int]
    LOCATOR_FIELD_NUMBER: _ClassVar[int]
    PLANNED_FIELD_NUMBER: _ClassVar[int]
    LAST_SUCCESS_AT_FIELD_NUMBER: _ClassVar[int]
    LAST_VERIFIED_AT_FIELD_NUMBER: _ClassVar[int]
    CRITICAL_FIELD_NUMBER: _ClassVar[int]
    id: str
    owner: str
    name: str
    source_kind: _sources_pb2.SourceKind
    locator: str
    planned: bool
    last_success_at: _timestamp_pb2.Timestamp
    last_verified_at: _timestamp_pb2.Timestamp
    critical: bool
    def __init__(self, id: _Optional[str] = ..., owner: _Optional[str] = ..., name: _Optional[str] = ..., source_kind: _Optional[_Union[_sources_pb2.SourceKind, str]] = ..., locator: _Optional[str] = ..., planned: _Optional[bool] = ..., last_success_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., last_verified_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., critical: _Optional[bool] = ...) -> None: ...

class SuggestedTarget(_message.Message):
    __slots__ = ("id", "owner", "name", "source_kind", "locator", "rationale", "approx_bytes", "sensitive", "warning", "critical")
    ID_FIELD_NUMBER: _ClassVar[int]
    OWNER_FIELD_NUMBER: _ClassVar[int]
    NAME_FIELD_NUMBER: _ClassVar[int]
    SOURCE_KIND_FIELD_NUMBER: _ClassVar[int]
    LOCATOR_FIELD_NUMBER: _ClassVar[int]
    RATIONALE_FIELD_NUMBER: _ClassVar[int]
    APPROX_BYTES_FIELD_NUMBER: _ClassVar[int]
    SENSITIVE_FIELD_NUMBER: _ClassVar[int]
    WARNING_FIELD_NUMBER: _ClassVar[int]
    CRITICAL_FIELD_NUMBER: _ClassVar[int]
    id: str
    owner: str
    name: str
    source_kind: _sources_pb2.SourceKind
    locator: str
    rationale: str
    approx_bytes: int
    sensitive: bool
    warning: str
    critical: bool
    def __init__(self, id: _Optional[str] = ..., owner: _Optional[str] = ..., name: _Optional[str] = ..., source_kind: _Optional[_Union[_sources_pb2.SourceKind, str]] = ..., locator: _Optional[str] = ..., rationale: _Optional[str] = ..., approx_bytes: _Optional[int] = ..., sensitive: _Optional[bool] = ..., warning: _Optional[str] = ..., critical: _Optional[bool] = ...) -> None: ...

class CoverageSummary(_message.Message):
    __slots__ = ("registered_count", "recommended_count", "sensitive_count", "planned_count", "backed_up_count", "verified_count", "default_coverage_complete", "has_sensitive_unreviewed", "has_unplanned_registered_targets", "has_unverified_targets")
    REGISTERED_COUNT_FIELD_NUMBER: _ClassVar[int]
    RECOMMENDED_COUNT_FIELD_NUMBER: _ClassVar[int]
    SENSITIVE_COUNT_FIELD_NUMBER: _ClassVar[int]
    PLANNED_COUNT_FIELD_NUMBER: _ClassVar[int]
    BACKED_UP_COUNT_FIELD_NUMBER: _ClassVar[int]
    VERIFIED_COUNT_FIELD_NUMBER: _ClassVar[int]
    DEFAULT_COVERAGE_COMPLETE_FIELD_NUMBER: _ClassVar[int]
    HAS_SENSITIVE_UNREVIEWED_FIELD_NUMBER: _ClassVar[int]
    HAS_UNPLANNED_REGISTERED_TARGETS_FIELD_NUMBER: _ClassVar[int]
    HAS_UNVERIFIED_TARGETS_FIELD_NUMBER: _ClassVar[int]
    registered_count: int
    recommended_count: int
    sensitive_count: int
    planned_count: int
    backed_up_count: int
    verified_count: int
    default_coverage_complete: bool
    has_sensitive_unreviewed: bool
    has_unplanned_registered_targets: bool
    has_unverified_targets: bool
    def __init__(self, registered_count: _Optional[int] = ..., recommended_count: _Optional[int] = ..., sensitive_count: _Optional[int] = ..., planned_count: _Optional[int] = ..., backed_up_count: _Optional[int] = ..., verified_count: _Optional[int] = ..., default_coverage_complete: _Optional[bool] = ..., has_sensitive_unreviewed: _Optional[bool] = ..., has_unplanned_registered_targets: _Optional[bool] = ..., has_unverified_targets: _Optional[bool] = ...) -> None: ...

class CoverageReport(_message.Message):
    __slots__ = ("summary", "registered_targets", "recommended_targets", "sensitive_targets")
    SUMMARY_FIELD_NUMBER: _ClassVar[int]
    REGISTERED_TARGETS_FIELD_NUMBER: _ClassVar[int]
    RECOMMENDED_TARGETS_FIELD_NUMBER: _ClassVar[int]
    SENSITIVE_TARGETS_FIELD_NUMBER: _ClassVar[int]
    summary: CoverageSummary
    registered_targets: _containers.RepeatedCompositeFieldContainer[RegisteredTarget]
    recommended_targets: _containers.RepeatedCompositeFieldContainer[SuggestedTarget]
    sensitive_targets: _containers.RepeatedCompositeFieldContainer[SuggestedTarget]
    def __init__(self, summary: _Optional[_Union[CoverageSummary, _Mapping]] = ..., registered_targets: _Optional[_Iterable[_Union[RegisteredTarget, _Mapping]]] = ..., recommended_targets: _Optional[_Iterable[_Union[SuggestedTarget, _Mapping]]] = ..., sensitive_targets: _Optional[_Iterable[_Union[SuggestedTarget, _Mapping]]] = ...) -> None: ...

class GetCoverageReportRequest(_message.Message):
    __slots__ = ()
    def __init__(self) -> None: ...

class GetCoverageReportResponse(_message.Message):
    __slots__ = ("report",)
    REPORT_FIELD_NUMBER: _ClassVar[int]
    report: CoverageReport
    def __init__(self, report: _Optional[_Union[CoverageReport, _Mapping]] = ...) -> None: ...

class AcceptedTarget(_message.Message):
    __slots__ = ("target_id", "suggestion_id", "owner", "name", "source_kind", "locator", "sensitive", "critical")
    TARGET_ID_FIELD_NUMBER: _ClassVar[int]
    SUGGESTION_ID_FIELD_NUMBER: _ClassVar[int]
    OWNER_FIELD_NUMBER: _ClassVar[int]
    NAME_FIELD_NUMBER: _ClassVar[int]
    SOURCE_KIND_FIELD_NUMBER: _ClassVar[int]
    LOCATOR_FIELD_NUMBER: _ClassVar[int]
    SENSITIVE_FIELD_NUMBER: _ClassVar[int]
    CRITICAL_FIELD_NUMBER: _ClassVar[int]
    target_id: str
    suggestion_id: str
    owner: str
    name: str
    source_kind: _sources_pb2.SourceKind
    locator: str
    sensitive: bool
    critical: bool
    def __init__(self, target_id: _Optional[str] = ..., suggestion_id: _Optional[str] = ..., owner: _Optional[str] = ..., name: _Optional[str] = ..., source_kind: _Optional[_Union[_sources_pb2.SourceKind, str]] = ..., locator: _Optional[str] = ..., sensitive: _Optional[bool] = ..., critical: _Optional[bool] = ...) -> None: ...

class AcceptError(_message.Message):
    __slots__ = ("suggestion_id", "owner", "name", "message")
    SUGGESTION_ID_FIELD_NUMBER: _ClassVar[int]
    OWNER_FIELD_NUMBER: _ClassVar[int]
    NAME_FIELD_NUMBER: _ClassVar[int]
    MESSAGE_FIELD_NUMBER: _ClassVar[int]
    suggestion_id: str
    owner: str
    name: str
    message: str
    def __init__(self, suggestion_id: _Optional[str] = ..., owner: _Optional[str] = ..., name: _Optional[str] = ..., message: _Optional[str] = ...) -> None: ...

class AcceptDefaultTargetsRequest(_message.Message):
    __slots__ = ("include_sensitive", "dry_run")
    INCLUDE_SENSITIVE_FIELD_NUMBER: _ClassVar[int]
    DRY_RUN_FIELD_NUMBER: _ClassVar[int]
    include_sensitive: bool
    dry_run: bool
    def __init__(self, include_sensitive: _Optional[bool] = ..., dry_run: _Optional[bool] = ...) -> None: ...

class AcceptDefaultTargetsResponse(_message.Message):
    __slots__ = ("accepted", "skipped_sensitive", "failed", "dry_run")
    ACCEPTED_FIELD_NUMBER: _ClassVar[int]
    SKIPPED_SENSITIVE_FIELD_NUMBER: _ClassVar[int]
    FAILED_FIELD_NUMBER: _ClassVar[int]
    DRY_RUN_FIELD_NUMBER: _ClassVar[int]
    accepted: _containers.RepeatedCompositeFieldContainer[AcceptedTarget]
    skipped_sensitive: _containers.RepeatedCompositeFieldContainer[SuggestedTarget]
    failed: _containers.RepeatedCompositeFieldContainer[AcceptError]
    dry_run: bool
    def __init__(self, accepted: _Optional[_Iterable[_Union[AcceptedTarget, _Mapping]]] = ..., skipped_sensitive: _Optional[_Iterable[_Union[SuggestedTarget, _Mapping]]] = ..., failed: _Optional[_Iterable[_Union[AcceptError, _Mapping]]] = ..., dry_run: _Optional[bool] = ...) -> None: ...
