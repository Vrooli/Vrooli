from google.protobuf.internal import containers as _containers
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Iterable as _Iterable, Mapping as _Mapping
from typing import ClassVar as _ClassVar, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class ExpirationPolicy(_message.Message):
    __slots__ = ("mode", "duration")
    MODE_FIELD_NUMBER: _ClassVar[int]
    DURATION_FIELD_NUMBER: _ClassVar[int]
    mode: str
    duration: str
    def __init__(self, mode: _Optional[str] = ..., duration: _Optional[str] = ...) -> None: ...

class Session(_message.Message):
    __slots__ = ("id", "shell", "created_at", "cols", "rows", "backend", "survives_restart", "policy", "busy", "recovered")
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
    def __init__(self, id: _Optional[str] = ..., shell: _Optional[str] = ..., created_at: _Optional[str] = ..., cols: _Optional[int] = ..., rows: _Optional[int] = ..., backend: _Optional[str] = ..., survives_restart: _Optional[bool] = ..., policy: _Optional[_Union[ExpirationPolicy, _Mapping]] = ..., busy: _Optional[bool] = ..., recovered: _Optional[bool] = ...) -> None: ...

class RecoverableSession(_message.Message):
    __slots__ = ("id", "backend", "shell", "cols", "rows", "created_at", "orphaned_at", "last_activity_at", "agent_type", "agent_session_id", "launch_command", "cwd", "last_rollout_path", "recoverable", "not_recoverable_reason")
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
    def __init__(self, id: _Optional[str] = ..., backend: _Optional[str] = ..., shell: _Optional[str] = ..., cols: _Optional[int] = ..., rows: _Optional[int] = ..., created_at: _Optional[str] = ..., orphaned_at: _Optional[str] = ..., last_activity_at: _Optional[str] = ..., agent_type: _Optional[str] = ..., agent_session_id: _Optional[str] = ..., launch_command: _Optional[str] = ..., cwd: _Optional[str] = ..., last_rollout_path: _Optional[str] = ..., recoverable: _Optional[bool] = ..., not_recoverable_reason: _Optional[str] = ...) -> None: ...

class CreateRequest(_message.Message):
    __slots__ = ("shell", "cols", "rows", "backend", "policy", "has_policy", "launch_command", "agent_type")
    SHELL_FIELD_NUMBER: _ClassVar[int]
    COLS_FIELD_NUMBER: _ClassVar[int]
    ROWS_FIELD_NUMBER: _ClassVar[int]
    BACKEND_FIELD_NUMBER: _ClassVar[int]
    POLICY_FIELD_NUMBER: _ClassVar[int]
    HAS_POLICY_FIELD_NUMBER: _ClassVar[int]
    LAUNCH_COMMAND_FIELD_NUMBER: _ClassVar[int]
    AGENT_TYPE_FIELD_NUMBER: _ClassVar[int]
    shell: str
    cols: int
    rows: int
    backend: str
    policy: ExpirationPolicy
    has_policy: bool
    launch_command: str
    agent_type: str
    def __init__(self, shell: _Optional[str] = ..., cols: _Optional[int] = ..., rows: _Optional[int] = ..., backend: _Optional[str] = ..., policy: _Optional[_Union[ExpirationPolicy, _Mapping]] = ..., has_policy: _Optional[bool] = ..., launch_command: _Optional[str] = ..., agent_type: _Optional[str] = ...) -> None: ...

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
