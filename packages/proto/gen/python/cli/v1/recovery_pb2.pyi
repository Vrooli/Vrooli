from google.protobuf.internal import containers as _containers
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Iterable as _Iterable, Mapping as _Mapping
from typing import ClassVar as _ClassVar, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class RecoveryCopyStats(_message.Message):
    __slots__ = ("dirs", "symlinks", "reflink_files", "deep_copy_files", "bytes_copied", "excluded")
    DIRS_FIELD_NUMBER: _ClassVar[int]
    SYMLINKS_FIELD_NUMBER: _ClassVar[int]
    REFLINK_FILES_FIELD_NUMBER: _ClassVar[int]
    DEEP_COPY_FILES_FIELD_NUMBER: _ClassVar[int]
    BYTES_COPIED_FIELD_NUMBER: _ClassVar[int]
    EXCLUDED_FIELD_NUMBER: _ClassVar[int]
    dirs: int
    symlinks: int
    reflink_files: int
    deep_copy_files: int
    bytes_copied: int
    excluded: int
    def __init__(self, dirs: _Optional[int] = ..., symlinks: _Optional[int] = ..., reflink_files: _Optional[int] = ..., deep_copy_files: _Optional[int] = ..., bytes_copied: _Optional[int] = ..., excluded: _Optional[int] = ...) -> None: ...

class RecoveryCaptureOutput(_message.Message):
    __slots__ = ("scenario", "slug", "source", "restore_point_path", "stats")
    SCENARIO_FIELD_NUMBER: _ClassVar[int]
    SLUG_FIELD_NUMBER: _ClassVar[int]
    SOURCE_FIELD_NUMBER: _ClassVar[int]
    RESTORE_POINT_PATH_FIELD_NUMBER: _ClassVar[int]
    STATS_FIELD_NUMBER: _ClassVar[int]
    scenario: str
    slug: str
    source: str
    restore_point_path: str
    stats: RecoveryCopyStats
    def __init__(self, scenario: _Optional[str] = ..., slug: _Optional[str] = ..., source: _Optional[str] = ..., restore_point_path: _Optional[str] = ..., stats: _Optional[_Union[RecoveryCopyStats, _Mapping]] = ...) -> None: ...

class RecoveryRestoreOutput(_message.Message):
    __slots__ = ("scenario", "slug", "restore_point_path", "dest", "stats")
    SCENARIO_FIELD_NUMBER: _ClassVar[int]
    SLUG_FIELD_NUMBER: _ClassVar[int]
    RESTORE_POINT_PATH_FIELD_NUMBER: _ClassVar[int]
    DEST_FIELD_NUMBER: _ClassVar[int]
    STATS_FIELD_NUMBER: _ClassVar[int]
    scenario: str
    slug: str
    restore_point_path: str
    dest: str
    stats: RecoveryCopyStats
    def __init__(self, scenario: _Optional[str] = ..., slug: _Optional[str] = ..., restore_point_path: _Optional[str] = ..., dest: _Optional[str] = ..., stats: _Optional[_Union[RecoveryCopyStats, _Mapping]] = ...) -> None: ...

class RecoveryEngagementView(_message.Message):
    __slots__ = ("scenario", "slug", "variant", "mode", "restore_point_path", "anchor_baseline_name", "ambient_var", "shadow_instance_key", "created_at", "last_touched_at", "ttl", "expires_at", "expired")
    SCENARIO_FIELD_NUMBER: _ClassVar[int]
    SLUG_FIELD_NUMBER: _ClassVar[int]
    VARIANT_FIELD_NUMBER: _ClassVar[int]
    MODE_FIELD_NUMBER: _ClassVar[int]
    RESTORE_POINT_PATH_FIELD_NUMBER: _ClassVar[int]
    ANCHOR_BASELINE_NAME_FIELD_NUMBER: _ClassVar[int]
    AMBIENT_VAR_FIELD_NUMBER: _ClassVar[int]
    SHADOW_INSTANCE_KEY_FIELD_NUMBER: _ClassVar[int]
    CREATED_AT_FIELD_NUMBER: _ClassVar[int]
    LAST_TOUCHED_AT_FIELD_NUMBER: _ClassVar[int]
    TTL_FIELD_NUMBER: _ClassVar[int]
    EXPIRES_AT_FIELD_NUMBER: _ClassVar[int]
    EXPIRED_FIELD_NUMBER: _ClassVar[int]
    scenario: str
    slug: str
    variant: str
    mode: str
    restore_point_path: str
    anchor_baseline_name: str
    ambient_var: str
    shadow_instance_key: str
    created_at: str
    last_touched_at: str
    ttl: str
    expires_at: str
    expired: bool
    def __init__(self, scenario: _Optional[str] = ..., slug: _Optional[str] = ..., variant: _Optional[str] = ..., mode: _Optional[str] = ..., restore_point_path: _Optional[str] = ..., anchor_baseline_name: _Optional[str] = ..., ambient_var: _Optional[str] = ..., shadow_instance_key: _Optional[str] = ..., created_at: _Optional[str] = ..., last_touched_at: _Optional[str] = ..., ttl: _Optional[str] = ..., expires_at: _Optional[str] = ..., expired: _Optional[bool] = ...) -> None: ...

class RecoveryListOutput(_message.Message):
    __slots__ = ("engagements",)
    ENGAGEMENTS_FIELD_NUMBER: _ClassVar[int]
    engagements: _containers.RepeatedCompositeFieldContainer[RecoveryEngagementView]
    def __init__(self, engagements: _Optional[_Iterable[_Union[RecoveryEngagementView, _Mapping]]] = ...) -> None: ...

class RecoveryCleanOutput(_message.Message):
    __slots__ = ("scenario", "slug", "engagement_dir")
    SCENARIO_FIELD_NUMBER: _ClassVar[int]
    SLUG_FIELD_NUMBER: _ClassVar[int]
    ENGAGEMENT_DIR_FIELD_NUMBER: _ClassVar[int]
    scenario: str
    slug: str
    engagement_dir: str
    def __init__(self, scenario: _Optional[str] = ..., slug: _Optional[str] = ..., engagement_dir: _Optional[str] = ...) -> None: ...

class RecoveryMigrateOutput(_message.Message):
    __slots__ = ("scenario", "slug", "migrations_dir", "db_path_auto_resolved", "engine", "database", "dry_run", "fast_path", "scripts_seen", "applied", "skipped")
    SCENARIO_FIELD_NUMBER: _ClassVar[int]
    SLUG_FIELD_NUMBER: _ClassVar[int]
    MIGRATIONS_DIR_FIELD_NUMBER: _ClassVar[int]
    DB_PATH_AUTO_RESOLVED_FIELD_NUMBER: _ClassVar[int]
    ENGINE_FIELD_NUMBER: _ClassVar[int]
    DATABASE_FIELD_NUMBER: _ClassVar[int]
    DRY_RUN_FIELD_NUMBER: _ClassVar[int]
    FAST_PATH_FIELD_NUMBER: _ClassVar[int]
    SCRIPTS_SEEN_FIELD_NUMBER: _ClassVar[int]
    APPLIED_FIELD_NUMBER: _ClassVar[int]
    SKIPPED_FIELD_NUMBER: _ClassVar[int]
    scenario: str
    slug: str
    migrations_dir: str
    db_path_auto_resolved: bool
    engine: str
    database: str
    dry_run: bool
    fast_path: bool
    scripts_seen: int
    applied: _containers.RepeatedScalarFieldContainer[str]
    skipped: _containers.RepeatedScalarFieldContainer[str]
    def __init__(self, scenario: _Optional[str] = ..., slug: _Optional[str] = ..., migrations_dir: _Optional[str] = ..., db_path_auto_resolved: _Optional[bool] = ..., engine: _Optional[str] = ..., database: _Optional[str] = ..., dry_run: _Optional[bool] = ..., fast_path: _Optional[bool] = ..., scripts_seen: _Optional[int] = ..., applied: _Optional[_Iterable[str]] = ..., skipped: _Optional[_Iterable[str]] = ...) -> None: ...

class RecoveryNamespaceOutput(_message.Message):
    __slots__ = ("scenario", "variant", "instance_key", "postgres_db", "data_dir", "data_dir_name", "storage_namespace")
    SCENARIO_FIELD_NUMBER: _ClassVar[int]
    VARIANT_FIELD_NUMBER: _ClassVar[int]
    INSTANCE_KEY_FIELD_NUMBER: _ClassVar[int]
    POSTGRES_DB_FIELD_NUMBER: _ClassVar[int]
    DATA_DIR_FIELD_NUMBER: _ClassVar[int]
    DATA_DIR_NAME_FIELD_NUMBER: _ClassVar[int]
    STORAGE_NAMESPACE_FIELD_NUMBER: _ClassVar[int]
    scenario: str
    variant: str
    instance_key: str
    postgres_db: str
    data_dir: str
    data_dir_name: str
    storage_namespace: str
    def __init__(self, scenario: _Optional[str] = ..., variant: _Optional[str] = ..., instance_key: _Optional[str] = ..., postgres_db: _Optional[str] = ..., data_dir: _Optional[str] = ..., data_dir_name: _Optional[str] = ..., storage_namespace: _Optional[str] = ...) -> None: ...
