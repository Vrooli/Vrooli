from web_console.v1.shared import target_pb2 as _target_pb2
from google.protobuf.internal import containers as _containers
from google.protobuf.internal import enum_type_wrapper as _enum_type_wrapper
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Iterable as _Iterable, Mapping as _Mapping
from typing import ClassVar as _ClassVar, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class SessionOrigin(int, metaclass=_enum_type_wrapper.EnumTypeWrapper):
    __slots__ = ()
    SESSION_ORIGIN_UNSPECIFIED: _ClassVar[SessionOrigin]
    SESSION_ORIGIN_UI: _ClassVar[SessionOrigin]
    SESSION_ORIGIN_PROGRAMMATIC: _ClassVar[SessionOrigin]
    SESSION_ORIGIN_REMOTE: _ClassVar[SessionOrigin]

class ArchiveRestoreState(int, metaclass=_enum_type_wrapper.EnumTypeWrapper):
    __slots__ = ()
    ARCHIVE_RESTORE_STATE_UNSPECIFIED: _ClassVar[ArchiveRestoreState]
    ARCHIVE_RESTORE_STATE_REOPENABLE: _ClassVar[ArchiveRestoreState]
    ARCHIVE_RESTORE_STATE_READ_ONLY: _ClassVar[ArchiveRestoreState]
    ARCHIVE_RESTORE_STATE_NOTHING_TO_RESTORE: _ClassVar[ArchiveRestoreState]
SESSION_ORIGIN_UNSPECIFIED: SessionOrigin
SESSION_ORIGIN_UI: SessionOrigin
SESSION_ORIGIN_PROGRAMMATIC: SessionOrigin
SESSION_ORIGIN_REMOTE: SessionOrigin
ARCHIVE_RESTORE_STATE_UNSPECIFIED: ArchiveRestoreState
ARCHIVE_RESTORE_STATE_REOPENABLE: ArchiveRestoreState
ARCHIVE_RESTORE_STATE_READ_ONLY: ArchiveRestoreState
ARCHIVE_RESTORE_STATE_NOTHING_TO_RESTORE: ArchiveRestoreState

class ExpirationPolicy(_message.Message):
    __slots__ = ("mode", "duration")
    MODE_FIELD_NUMBER: _ClassVar[int]
    DURATION_FIELD_NUMBER: _ClassVar[int]
    mode: str
    duration: str
    def __init__(self, mode: _Optional[str] = ..., duration: _Optional[str] = ...) -> None: ...

class Session(_message.Message):
    __slots__ = ("id", "shell", "created_at", "cols", "rows", "backend", "survives_restart", "policy", "busy", "recovered", "origin", "owner", "display_label", "tracking_degraded", "target")
    ID_FIELD_NUMBER: _ClassVar[int]
    SHELL_FIELD_NUMBER: _ClassVar[int]
    CREATED_AT_FIELD_NUMBER: _ClassVar[int]
    COLS_FIELD_NUMBER: _ClassVar[int]
    ROWS_FIELD_NUMBER: _ClassVar[int]
    BACKEND_FIELD_NUMBER: _ClassVar[int]
    SURVIVES_RESTART_FIELD_NUMBER: _ClassVar[int]
    POLICY_FIELD_NUMBER: _ClassVar[int]
    BUSY_FIELD_NUMBER: _ClassVar[int]
    RECOVERED_FIELD_NUMBER: _ClassVar[int]
    ORIGIN_FIELD_NUMBER: _ClassVar[int]
    OWNER_FIELD_NUMBER: _ClassVar[int]
    DISPLAY_LABEL_FIELD_NUMBER: _ClassVar[int]
    TRACKING_DEGRADED_FIELD_NUMBER: _ClassVar[int]
    TARGET_FIELD_NUMBER: _ClassVar[int]
    id: str
    shell: str
    created_at: str
    cols: int
    rows: int
    backend: str
    survives_restart: bool
    policy: ExpirationPolicy
    busy: bool
    recovered: bool
    origin: SessionOrigin
    owner: str
    display_label: str
    tracking_degraded: bool
    target: _target_pb2.Target
    def __init__(self, id: _Optional[str] = ..., shell: _Optional[str] = ..., created_at: _Optional[str] = ..., cols: _Optional[int] = ..., rows: _Optional[int] = ..., backend: _Optional[str] = ..., survives_restart: _Optional[bool] = ..., policy: _Optional[_Union[ExpirationPolicy, _Mapping]] = ..., busy: _Optional[bool] = ..., recovered: _Optional[bool] = ..., origin: _Optional[_Union[SessionOrigin, str]] = ..., owner: _Optional[str] = ..., display_label: _Optional[str] = ..., tracking_degraded: _Optional[bool] = ..., target: _Optional[_Union[_target_pb2.Target, _Mapping]] = ...) -> None: ...

class RecoverableSession(_message.Message):
    __slots__ = ("id", "backend", "shell", "cols", "rows", "created_at", "orphaned_at", "last_activity_at", "agent_type", "agent_session_id", "launch_command", "cwd", "last_rollout_path", "recoverable", "not_recoverable_reason", "pane_name", "header_color", "group_name")
    ID_FIELD_NUMBER: _ClassVar[int]
    BACKEND_FIELD_NUMBER: _ClassVar[int]
    SHELL_FIELD_NUMBER: _ClassVar[int]
    COLS_FIELD_NUMBER: _ClassVar[int]
    ROWS_FIELD_NUMBER: _ClassVar[int]
    CREATED_AT_FIELD_NUMBER: _ClassVar[int]
    ORPHANED_AT_FIELD_NUMBER: _ClassVar[int]
    LAST_ACTIVITY_AT_FIELD_NUMBER: _ClassVar[int]
    AGENT_TYPE_FIELD_NUMBER: _ClassVar[int]
    AGENT_SESSION_ID_FIELD_NUMBER: _ClassVar[int]
    LAUNCH_COMMAND_FIELD_NUMBER: _ClassVar[int]
    CWD_FIELD_NUMBER: _ClassVar[int]
    LAST_ROLLOUT_PATH_FIELD_NUMBER: _ClassVar[int]
    RECOVERABLE_FIELD_NUMBER: _ClassVar[int]
    NOT_RECOVERABLE_REASON_FIELD_NUMBER: _ClassVar[int]
    PANE_NAME_FIELD_NUMBER: _ClassVar[int]
    HEADER_COLOR_FIELD_NUMBER: _ClassVar[int]
    GROUP_NAME_FIELD_NUMBER: _ClassVar[int]
    id: str
    backend: str
    shell: str
    cols: int
    rows: int
    created_at: str
    orphaned_at: str
    last_activity_at: str
    agent_type: str
    agent_session_id: str
    launch_command: str
    cwd: str
    last_rollout_path: str
    recoverable: bool
    not_recoverable_reason: str
    pane_name: str
    header_color: str
    group_name: str
    def __init__(self, id: _Optional[str] = ..., backend: _Optional[str] = ..., shell: _Optional[str] = ..., cols: _Optional[int] = ..., rows: _Optional[int] = ..., created_at: _Optional[str] = ..., orphaned_at: _Optional[str] = ..., last_activity_at: _Optional[str] = ..., agent_type: _Optional[str] = ..., agent_session_id: _Optional[str] = ..., launch_command: _Optional[str] = ..., cwd: _Optional[str] = ..., last_rollout_path: _Optional[str] = ..., recoverable: _Optional[bool] = ..., not_recoverable_reason: _Optional[str] = ..., pane_name: _Optional[str] = ..., header_color: _Optional[str] = ..., group_name: _Optional[str] = ...) -> None: ...

class CreateRequest(_message.Message):
    __slots__ = ("shell", "cols", "rows", "backend", "policy", "has_policy", "launch_command", "agent_type", "origin", "owner", "display_label", "execute_launch_command", "target_id", "working_dir")
    SHELL_FIELD_NUMBER: _ClassVar[int]
    COLS_FIELD_NUMBER: _ClassVar[int]
    ROWS_FIELD_NUMBER: _ClassVar[int]
    BACKEND_FIELD_NUMBER: _ClassVar[int]
    POLICY_FIELD_NUMBER: _ClassVar[int]
    HAS_POLICY_FIELD_NUMBER: _ClassVar[int]
    LAUNCH_COMMAND_FIELD_NUMBER: _ClassVar[int]
    AGENT_TYPE_FIELD_NUMBER: _ClassVar[int]
    ORIGIN_FIELD_NUMBER: _ClassVar[int]
    OWNER_FIELD_NUMBER: _ClassVar[int]
    DISPLAY_LABEL_FIELD_NUMBER: _ClassVar[int]
    EXECUTE_LAUNCH_COMMAND_FIELD_NUMBER: _ClassVar[int]
    TARGET_ID_FIELD_NUMBER: _ClassVar[int]
    WORKING_DIR_FIELD_NUMBER: _ClassVar[int]
    shell: str
    cols: int
    rows: int
    backend: str
    policy: ExpirationPolicy
    has_policy: bool
    launch_command: str
    agent_type: str
    origin: SessionOrigin
    owner: str
    display_label: str
    execute_launch_command: bool
    target_id: str
    working_dir: str
    def __init__(self, shell: _Optional[str] = ..., cols: _Optional[int] = ..., rows: _Optional[int] = ..., backend: _Optional[str] = ..., policy: _Optional[_Union[ExpirationPolicy, _Mapping]] = ..., has_policy: _Optional[bool] = ..., launch_command: _Optional[str] = ..., agent_type: _Optional[str] = ..., origin: _Optional[_Union[SessionOrigin, str]] = ..., owner: _Optional[str] = ..., display_label: _Optional[str] = ..., execute_launch_command: _Optional[bool] = ..., target_id: _Optional[str] = ..., working_dir: _Optional[str] = ...) -> None: ...

class CreateResponse(_message.Message):
    __slots__ = ("session",)
    SESSION_FIELD_NUMBER: _ClassVar[int]
    session: Session
    def __init__(self, session: _Optional[_Union[Session, _Mapping]] = ...) -> None: ...

class ListRequest(_message.Message):
    __slots__ = ()
    def __init__(self) -> None: ...

class ListResponse(_message.Message):
    __slots__ = ("sessions", "recovery")
    SESSIONS_FIELD_NUMBER: _ClassVar[int]
    RECOVERY_FIELD_NUMBER: _ClassVar[int]
    sessions: _containers.RepeatedCompositeFieldContainer[Session]
    recovery: RecoveryStatus
    def __init__(self, sessions: _Optional[_Iterable[_Union[Session, _Mapping]]] = ..., recovery: _Optional[_Union[RecoveryStatus, _Mapping]] = ...) -> None: ...

class ArchivedSession(_message.Message):
    __slots__ = ("id", "archived_at", "created_at", "agent_type", "agent_session_id", "cwd", "pane_name", "header_color", "group_name", "message_count", "restore_state", "restore_state_reason", "awaiting_recovery")
    ID_FIELD_NUMBER: _ClassVar[int]
    ARCHIVED_AT_FIELD_NUMBER: _ClassVar[int]
    CREATED_AT_FIELD_NUMBER: _ClassVar[int]
    AGENT_TYPE_FIELD_NUMBER: _ClassVar[int]
    AGENT_SESSION_ID_FIELD_NUMBER: _ClassVar[int]
    CWD_FIELD_NUMBER: _ClassVar[int]
    PANE_NAME_FIELD_NUMBER: _ClassVar[int]
    HEADER_COLOR_FIELD_NUMBER: _ClassVar[int]
    GROUP_NAME_FIELD_NUMBER: _ClassVar[int]
    MESSAGE_COUNT_FIELD_NUMBER: _ClassVar[int]
    RESTORE_STATE_FIELD_NUMBER: _ClassVar[int]
    RESTORE_STATE_REASON_FIELD_NUMBER: _ClassVar[int]
    AWAITING_RECOVERY_FIELD_NUMBER: _ClassVar[int]
    id: str
    archived_at: str
    created_at: str
    agent_type: str
    agent_session_id: str
    cwd: str
    pane_name: str
    header_color: str
    group_name: str
    message_count: int
    restore_state: ArchiveRestoreState
    restore_state_reason: str
    awaiting_recovery: bool
    def __init__(self, id: _Optional[str] = ..., archived_at: _Optional[str] = ..., created_at: _Optional[str] = ..., agent_type: _Optional[str] = ..., agent_session_id: _Optional[str] = ..., cwd: _Optional[str] = ..., pane_name: _Optional[str] = ..., header_color: _Optional[str] = ..., group_name: _Optional[str] = ..., message_count: _Optional[int] = ..., restore_state: _Optional[_Union[ArchiveRestoreState, str]] = ..., restore_state_reason: _Optional[str] = ..., awaiting_recovery: _Optional[bool] = ...) -> None: ...

class ListArchivedRequest(_message.Message):
    __slots__ = ()
    def __init__(self) -> None: ...

class ListArchivedResponse(_message.Message):
    __slots__ = ("sessions", "total")
    SESSIONS_FIELD_NUMBER: _ClassVar[int]
    TOTAL_FIELD_NUMBER: _ClassVar[int]
    sessions: _containers.RepeatedCompositeFieldContainer[ArchivedSession]
    total: int
    def __init__(self, sessions: _Optional[_Iterable[_Union[ArchivedSession, _Mapping]]] = ..., total: _Optional[int] = ...) -> None: ...

class RecoveryStatus(_message.Message):
    __slots__ = ("in_progress", "total", "recovered", "awaiting_recovery", "adopted", "started_at_unix_ms", "completed_at_unix_ms")
    IN_PROGRESS_FIELD_NUMBER: _ClassVar[int]
    TOTAL_FIELD_NUMBER: _ClassVar[int]
    RECOVERED_FIELD_NUMBER: _ClassVar[int]
    AWAITING_RECOVERY_FIELD_NUMBER: _ClassVar[int]
    ADOPTED_FIELD_NUMBER: _ClassVar[int]
    STARTED_AT_UNIX_MS_FIELD_NUMBER: _ClassVar[int]
    COMPLETED_AT_UNIX_MS_FIELD_NUMBER: _ClassVar[int]
    in_progress: bool
    total: int
    recovered: int
    awaiting_recovery: int
    adopted: int
    started_at_unix_ms: int
    completed_at_unix_ms: int
    def __init__(self, in_progress: _Optional[bool] = ..., total: _Optional[int] = ..., recovered: _Optional[int] = ..., awaiting_recovery: _Optional[int] = ..., adopted: _Optional[int] = ..., started_at_unix_ms: _Optional[int] = ..., completed_at_unix_ms: _Optional[int] = ...) -> None: ...

class GetRequest(_message.Message):
    __slots__ = ("id",)
    ID_FIELD_NUMBER: _ClassVar[int]
    id: str
    def __init__(self, id: _Optional[str] = ...) -> None: ...

class GetResponse(_message.Message):
    __slots__ = ("session",)
    SESSION_FIELD_NUMBER: _ClassVar[int]
    session: Session
    def __init__(self, session: _Optional[_Union[Session, _Mapping]] = ...) -> None: ...

class ArchiveRequest(_message.Message):
    __slots__ = ("id",)
    ID_FIELD_NUMBER: _ClassVar[int]
    id: str
    def __init__(self, id: _Optional[str] = ...) -> None: ...

class ArchiveResponse(_message.Message):
    __slots__ = ("id",)
    ID_FIELD_NUMBER: _ClassVar[int]
    id: str
    def __init__(self, id: _Optional[str] = ...) -> None: ...

class UnarchiveRequest(_message.Message):
    __slots__ = ("id",)
    ID_FIELD_NUMBER: _ClassVar[int]
    id: str
    def __init__(self, id: _Optional[str] = ...) -> None: ...

class UnarchiveResponse(_message.Message):
    __slots__ = ("id",)
    ID_FIELD_NUMBER: _ClassVar[int]
    id: str
    def __init__(self, id: _Optional[str] = ...) -> None: ...

class DeleteRequest(_message.Message):
    __slots__ = ("id",)
    ID_FIELD_NUMBER: _ClassVar[int]
    id: str
    def __init__(self, id: _Optional[str] = ...) -> None: ...

class DeleteResponse(_message.Message):
    __slots__ = ()
    def __init__(self) -> None: ...

class ListRecoverableRequest(_message.Message):
    __slots__ = ()
    def __init__(self) -> None: ...

class ListRecoverableResponse(_message.Message):
    __slots__ = ("sessions",)
    SESSIONS_FIELD_NUMBER: _ClassVar[int]
    sessions: _containers.RepeatedCompositeFieldContainer[RecoverableSession]
    def __init__(self, sessions: _Optional[_Iterable[_Union[RecoverableSession, _Mapping]]] = ...) -> None: ...

class DismissRecoverableRequest(_message.Message):
    __slots__ = ("id",)
    ID_FIELD_NUMBER: _ClassVar[int]
    id: str
    def __init__(self, id: _Optional[str] = ...) -> None: ...

class DismissRecoverableResponse(_message.Message):
    __slots__ = ("id",)
    ID_FIELD_NUMBER: _ClassVar[int]
    id: str
    def __init__(self, id: _Optional[str] = ...) -> None: ...

class RecoverRequest(_message.Message):
    __slots__ = ("id",)
    ID_FIELD_NUMBER: _ClassVar[int]
    id: str
    def __init__(self, id: _Optional[str] = ...) -> None: ...

class RecoverResponse(_message.Message):
    __slots__ = ("old_session_id", "new_session_id", "agent_type", "command_sent", "codex_home_copied")
    OLD_SESSION_ID_FIELD_NUMBER: _ClassVar[int]
    NEW_SESSION_ID_FIELD_NUMBER: _ClassVar[int]
    AGENT_TYPE_FIELD_NUMBER: _ClassVar[int]
    COMMAND_SENT_FIELD_NUMBER: _ClassVar[int]
    CODEX_HOME_COPIED_FIELD_NUMBER: _ClassVar[int]
    old_session_id: str
    new_session_id: str
    agent_type: str
    command_sent: str
    codex_home_copied: bool
    def __init__(self, old_session_id: _Optional[str] = ..., new_session_id: _Optional[str] = ..., agent_type: _Optional[str] = ..., command_sent: _Optional[str] = ..., codex_home_copied: _Optional[bool] = ...) -> None: ...

class ReopenRequest(_message.Message):
    __slots__ = ("id",)
    ID_FIELD_NUMBER: _ClassVar[int]
    id: str
    def __init__(self, id: _Optional[str] = ...) -> None: ...

class ReopenResponse(_message.Message):
    __slots__ = ("old_session_id", "new_session_id", "agent_type", "command_sent", "codex_home_copied")
    OLD_SESSION_ID_FIELD_NUMBER: _ClassVar[int]
    NEW_SESSION_ID_FIELD_NUMBER: _ClassVar[int]
    AGENT_TYPE_FIELD_NUMBER: _ClassVar[int]
    COMMAND_SENT_FIELD_NUMBER: _ClassVar[int]
    CODEX_HOME_COPIED_FIELD_NUMBER: _ClassVar[int]
    old_session_id: str
    new_session_id: str
    agent_type: str
    command_sent: str
    codex_home_copied: bool
    def __init__(self, old_session_id: _Optional[str] = ..., new_session_id: _Optional[str] = ..., agent_type: _Optional[str] = ..., command_sent: _Optional[str] = ..., codex_home_copied: _Optional[bool] = ...) -> None: ...

class ArchiveRetentionPolicy(_message.Message):
    __slots__ = ("message_less_age_days", "agent_home_age_days", "max_bytes")
    MESSAGE_LESS_AGE_DAYS_FIELD_NUMBER: _ClassVar[int]
    AGENT_HOME_AGE_DAYS_FIELD_NUMBER: _ClassVar[int]
    MAX_BYTES_FIELD_NUMBER: _ClassVar[int]
    message_less_age_days: int
    agent_home_age_days: int
    max_bytes: int
    def __init__(self, message_less_age_days: _Optional[int] = ..., agent_home_age_days: _Optional[int] = ..., max_bytes: _Optional[int] = ...) -> None: ...

class ArchiveRetentionStats(_message.Message):
    __slots__ = ("entry_count", "message_count", "transcript_bytes", "agent_home_bytes", "total_bytes")
    ENTRY_COUNT_FIELD_NUMBER: _ClassVar[int]
    MESSAGE_COUNT_FIELD_NUMBER: _ClassVar[int]
    TRANSCRIPT_BYTES_FIELD_NUMBER: _ClassVar[int]
    AGENT_HOME_BYTES_FIELD_NUMBER: _ClassVar[int]
    TOTAL_BYTES_FIELD_NUMBER: _ClassVar[int]
    entry_count: int
    message_count: int
    transcript_bytes: int
    agent_home_bytes: int
    total_bytes: int
    def __init__(self, entry_count: _Optional[int] = ..., message_count: _Optional[int] = ..., transcript_bytes: _Optional[int] = ..., agent_home_bytes: _Optional[int] = ..., total_bytes: _Optional[int] = ...) -> None: ...

class GetArchiveRetentionRequest(_message.Message):
    __slots__ = ()
    def __init__(self) -> None: ...

class GetArchiveRetentionResponse(_message.Message):
    __slots__ = ("policy", "stats")
    POLICY_FIELD_NUMBER: _ClassVar[int]
    STATS_FIELD_NUMBER: _ClassVar[int]
    policy: ArchiveRetentionPolicy
    stats: ArchiveRetentionStats
    def __init__(self, policy: _Optional[_Union[ArchiveRetentionPolicy, _Mapping]] = ..., stats: _Optional[_Union[ArchiveRetentionStats, _Mapping]] = ...) -> None: ...

class PruneArchiveRequest(_message.Message):
    __slots__ = ("apply",)
    APPLY_FIELD_NUMBER: _ClassVar[int]
    apply: bool
    def __init__(self, apply: _Optional[bool] = ...) -> None: ...

class ArchivePruneAction(_message.Message):
    __slots__ = ("session_id", "kind", "bytes", "applied")
    SESSION_ID_FIELD_NUMBER: _ClassVar[int]
    KIND_FIELD_NUMBER: _ClassVar[int]
    BYTES_FIELD_NUMBER: _ClassVar[int]
    APPLIED_FIELD_NUMBER: _ClassVar[int]
    session_id: str
    kind: str
    bytes: int
    applied: bool
    def __init__(self, session_id: _Optional[str] = ..., kind: _Optional[str] = ..., bytes: _Optional[int] = ..., applied: _Optional[bool] = ...) -> None: ...

class PruneArchiveResponse(_message.Message):
    __slots__ = ("dry_run", "actions", "reclaimed_bytes", "before", "after")
    DRY_RUN_FIELD_NUMBER: _ClassVar[int]
    ACTIONS_FIELD_NUMBER: _ClassVar[int]
    RECLAIMED_BYTES_FIELD_NUMBER: _ClassVar[int]
    BEFORE_FIELD_NUMBER: _ClassVar[int]
    AFTER_FIELD_NUMBER: _ClassVar[int]
    dry_run: bool
    actions: _containers.RepeatedCompositeFieldContainer[ArchivePruneAction]
    reclaimed_bytes: int
    before: ArchiveRetentionStats
    after: ArchiveRetentionStats
    def __init__(self, dry_run: _Optional[bool] = ..., actions: _Optional[_Iterable[_Union[ArchivePruneAction, _Mapping]]] = ..., reclaimed_bytes: _Optional[int] = ..., before: _Optional[_Union[ArchiveRetentionStats, _Mapping]] = ..., after: _Optional[_Union[ArchiveRetentionStats, _Mapping]] = ...) -> None: ...

class GetPolicyRequest(_message.Message):
    __slots__ = ("id",)
    ID_FIELD_NUMBER: _ClassVar[int]
    id: str
    def __init__(self, id: _Optional[str] = ...) -> None: ...

class PolicyView(_message.Message):
    __slots__ = ("session_id", "policy", "expires_at", "ttl_seconds", "has_expiry")
    SESSION_ID_FIELD_NUMBER: _ClassVar[int]
    POLICY_FIELD_NUMBER: _ClassVar[int]
    EXPIRES_AT_FIELD_NUMBER: _ClassVar[int]
    TTL_SECONDS_FIELD_NUMBER: _ClassVar[int]
    HAS_EXPIRY_FIELD_NUMBER: _ClassVar[int]
    session_id: str
    policy: ExpirationPolicy
    expires_at: str
    ttl_seconds: float
    has_expiry: bool
    def __init__(self, session_id: _Optional[str] = ..., policy: _Optional[_Union[ExpirationPolicy, _Mapping]] = ..., expires_at: _Optional[str] = ..., ttl_seconds: _Optional[float] = ..., has_expiry: _Optional[bool] = ...) -> None: ...

class GetPolicyResponse(_message.Message):
    __slots__ = ("policy",)
    POLICY_FIELD_NUMBER: _ClassVar[int]
    policy: PolicyView
    def __init__(self, policy: _Optional[_Union[PolicyView, _Mapping]] = ...) -> None: ...

class UpdatePolicyRequest(_message.Message):
    __slots__ = ("id", "policy")
    ID_FIELD_NUMBER: _ClassVar[int]
    POLICY_FIELD_NUMBER: _ClassVar[int]
    id: str
    policy: ExpirationPolicy
    def __init__(self, id: _Optional[str] = ..., policy: _Optional[_Union[ExpirationPolicy, _Mapping]] = ...) -> None: ...

class UpdatePolicyResponse(_message.Message):
    __slots__ = ("policy",)
    POLICY_FIELD_NUMBER: _ClassVar[int]
    policy: PolicyView
    def __init__(self, policy: _Optional[_Union[PolicyView, _Mapping]] = ...) -> None: ...
