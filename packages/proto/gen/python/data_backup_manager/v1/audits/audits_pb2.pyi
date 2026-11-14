import datetime

from buf.validate import validate_pb2 as _validate_pb2
from google.protobuf import timestamp_pb2 as _timestamp_pb2
from google.protobuf.internal import containers as _containers
from google.protobuf.internal import enum_type_wrapper as _enum_type_wrapper
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Iterable as _Iterable, Mapping as _Mapping
from typing import ClassVar as _ClassVar, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class AuditStatus(int, metaclass=_enum_type_wrapper.EnumTypeWrapper):
    __slots__ = ()
    AUDIT_STATUS_UNSPECIFIED: _ClassVar[AuditStatus]
    AUDIT_STATUS_REQUESTED: _ClassVar[AuditStatus]
    AUDIT_STATUS_RUNNING: _ClassVar[AuditStatus]
    AUDIT_STATUS_COMPLETED: _ClassVar[AuditStatus]
    AUDIT_STATUS_FAILED: _ClassVar[AuditStatus]
AUDIT_STATUS_UNSPECIFIED: AuditStatus
AUDIT_STATUS_REQUESTED: AuditStatus
AUDIT_STATUS_RUNNING: AuditStatus
AUDIT_STATUS_COMPLETED: AuditStatus
AUDIT_STATUS_FAILED: AuditStatus

class InventorySummary(_message.Message):
    __slots__ = ("files", "directories", "symlinks", "other", "regular_bytes", "path_list_sha256", "tree_content_sha256", "sqlite", "captured_at", "unreadable_paths")
    FILES_FIELD_NUMBER: _ClassVar[int]
    DIRECTORIES_FIELD_NUMBER: _ClassVar[int]
    SYMLINKS_FIELD_NUMBER: _ClassVar[int]
    OTHER_FIELD_NUMBER: _ClassVar[int]
    REGULAR_BYTES_FIELD_NUMBER: _ClassVar[int]
    PATH_LIST_SHA256_FIELD_NUMBER: _ClassVar[int]
    TREE_CONTENT_SHA256_FIELD_NUMBER: _ClassVar[int]
    SQLITE_FIELD_NUMBER: _ClassVar[int]
    CAPTURED_AT_FIELD_NUMBER: _ClassVar[int]
    UNREADABLE_PATHS_FIELD_NUMBER: _ClassVar[int]
    files: int
    directories: int
    symlinks: int
    other: int
    regular_bytes: int
    path_list_sha256: str
    tree_content_sha256: str
    sqlite: _containers.RepeatedCompositeFieldContainer[SqliteInventory]
    captured_at: _timestamp_pb2.Timestamp
    unreadable_paths: _containers.RepeatedScalarFieldContainer[str]
    def __init__(self, files: _Optional[int] = ..., directories: _Optional[int] = ..., symlinks: _Optional[int] = ..., other: _Optional[int] = ..., regular_bytes: _Optional[int] = ..., path_list_sha256: _Optional[str] = ..., tree_content_sha256: _Optional[str] = ..., sqlite: _Optional[_Iterable[_Union[SqliteInventory, _Mapping]]] = ..., captured_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., unreadable_paths: _Optional[_Iterable[str]] = ...) -> None: ...

class SqliteInventory(_message.Message):
    __slots__ = ("path", "integrity_status", "page_count", "page_size", "schema_sha256", "table_count")
    PATH_FIELD_NUMBER: _ClassVar[int]
    INTEGRITY_STATUS_FIELD_NUMBER: _ClassVar[int]
    PAGE_COUNT_FIELD_NUMBER: _ClassVar[int]
    PAGE_SIZE_FIELD_NUMBER: _ClassVar[int]
    SCHEMA_SHA256_FIELD_NUMBER: _ClassVar[int]
    TABLE_COUNT_FIELD_NUMBER: _ClassVar[int]
    path: str
    integrity_status: str
    page_count: int
    page_size: int
    schema_sha256: str
    table_count: int
    def __init__(self, path: _Optional[str] = ..., integrity_status: _Optional[str] = ..., page_count: _Optional[int] = ..., page_size: _Optional[int] = ..., schema_sha256: _Optional[str] = ..., table_count: _Optional[int] = ...) -> None: ...

class AuditComparison(_message.Message):
    __slots__ = ("matches", "mismatches", "live_newer_than_snapshot")
    MATCHES_FIELD_NUMBER: _ClassVar[int]
    MISMATCHES_FIELD_NUMBER: _ClassVar[int]
    LIVE_NEWER_THAN_SNAPSHOT_FIELD_NUMBER: _ClassVar[int]
    matches: bool
    mismatches: _containers.RepeatedScalarFieldContainer[str]
    live_newer_than_snapshot: bool
    def __init__(self, matches: _Optional[bool] = ..., mismatches: _Optional[_Iterable[str]] = ..., live_newer_than_snapshot: _Optional[bool] = ...) -> None: ...

class Audit(_message.Message):
    __slots__ = ("id", "target_id", "destination_id", "snapshot_id", "status", "include_content_hash", "include_sqlite_checks", "restorable", "live", "snapshot", "comparison", "snapshot_time", "requested_at", "finished_at", "error")
    ID_FIELD_NUMBER: _ClassVar[int]
    TARGET_ID_FIELD_NUMBER: _ClassVar[int]
    DESTINATION_ID_FIELD_NUMBER: _ClassVar[int]
    SNAPSHOT_ID_FIELD_NUMBER: _ClassVar[int]
    STATUS_FIELD_NUMBER: _ClassVar[int]
    INCLUDE_CONTENT_HASH_FIELD_NUMBER: _ClassVar[int]
    INCLUDE_SQLITE_CHECKS_FIELD_NUMBER: _ClassVar[int]
    RESTORABLE_FIELD_NUMBER: _ClassVar[int]
    LIVE_FIELD_NUMBER: _ClassVar[int]
    SNAPSHOT_FIELD_NUMBER: _ClassVar[int]
    COMPARISON_FIELD_NUMBER: _ClassVar[int]
    SNAPSHOT_TIME_FIELD_NUMBER: _ClassVar[int]
    REQUESTED_AT_FIELD_NUMBER: _ClassVar[int]
    FINISHED_AT_FIELD_NUMBER: _ClassVar[int]
    ERROR_FIELD_NUMBER: _ClassVar[int]
    id: str
    target_id: str
    destination_id: str
    snapshot_id: str
    status: AuditStatus
    include_content_hash: bool
    include_sqlite_checks: bool
    restorable: bool
    live: InventorySummary
    snapshot: InventorySummary
    comparison: AuditComparison
    snapshot_time: _timestamp_pb2.Timestamp
    requested_at: _timestamp_pb2.Timestamp
    finished_at: _timestamp_pb2.Timestamp
    error: str
    def __init__(self, id: _Optional[str] = ..., target_id: _Optional[str] = ..., destination_id: _Optional[str] = ..., snapshot_id: _Optional[str] = ..., status: _Optional[_Union[AuditStatus, str]] = ..., include_content_hash: _Optional[bool] = ..., include_sqlite_checks: _Optional[bool] = ..., restorable: _Optional[bool] = ..., live: _Optional[_Union[InventorySummary, _Mapping]] = ..., snapshot: _Optional[_Union[InventorySummary, _Mapping]] = ..., comparison: _Optional[_Union[AuditComparison, _Mapping]] = ..., snapshot_time: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., requested_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., finished_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., error: _Optional[str] = ...) -> None: ...

class RunSnapshotAuditRequest(_message.Message):
    __slots__ = ("target_id", "destination_id", "snapshot_id", "include_content_hash", "include_sqlite_checks")
    TARGET_ID_FIELD_NUMBER: _ClassVar[int]
    DESTINATION_ID_FIELD_NUMBER: _ClassVar[int]
    SNAPSHOT_ID_FIELD_NUMBER: _ClassVar[int]
    INCLUDE_CONTENT_HASH_FIELD_NUMBER: _ClassVar[int]
    INCLUDE_SQLITE_CHECKS_FIELD_NUMBER: _ClassVar[int]
    target_id: str
    destination_id: str
    snapshot_id: str
    include_content_hash: bool
    include_sqlite_checks: bool
    def __init__(self, target_id: _Optional[str] = ..., destination_id: _Optional[str] = ..., snapshot_id: _Optional[str] = ..., include_content_hash: _Optional[bool] = ..., include_sqlite_checks: _Optional[bool] = ...) -> None: ...

class RunSnapshotAuditResponse(_message.Message):
    __slots__ = ("audit",)
    AUDIT_FIELD_NUMBER: _ClassVar[int]
    audit: Audit
    def __init__(self, audit: _Optional[_Union[Audit, _Mapping]] = ...) -> None: ...

class GetAuditRequest(_message.Message):
    __slots__ = ("id",)
    ID_FIELD_NUMBER: _ClassVar[int]
    id: str
    def __init__(self, id: _Optional[str] = ...) -> None: ...

class GetAuditResponse(_message.Message):
    __slots__ = ("audit",)
    AUDIT_FIELD_NUMBER: _ClassVar[int]
    audit: Audit
    def __init__(self, audit: _Optional[_Union[Audit, _Mapping]] = ...) -> None: ...

class ListAuditsRequest(_message.Message):
    __slots__ = ("target_id", "page_size", "page_token")
    TARGET_ID_FIELD_NUMBER: _ClassVar[int]
    PAGE_SIZE_FIELD_NUMBER: _ClassVar[int]
    PAGE_TOKEN_FIELD_NUMBER: _ClassVar[int]
    target_id: str
    page_size: int
    page_token: str
    def __init__(self, target_id: _Optional[str] = ..., page_size: _Optional[int] = ..., page_token: _Optional[str] = ...) -> None: ...

class ListAuditsResponse(_message.Message):
    __slots__ = ("audits", "next_page_token")
    AUDITS_FIELD_NUMBER: _ClassVar[int]
    NEXT_PAGE_TOKEN_FIELD_NUMBER: _ClassVar[int]
    audits: _containers.RepeatedCompositeFieldContainer[Audit]
    next_page_token: str
    def __init__(self, audits: _Optional[_Iterable[_Union[Audit, _Mapping]]] = ..., next_page_token: _Optional[str] = ...) -> None: ...
