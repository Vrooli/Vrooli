from google.protobuf.internal import containers as _containers
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Iterable as _Iterable, Mapping as _Mapping
from typing import ClassVar as _ClassVar, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class Session(_message.Message):
    __slots__ = ("id", "state", "created_at", "last_activity_at", "grants", "sandbox_workspace", "reclaimed_reason", "name", "inference_cost_micros", "inference_tokens", "delegation_cost_micros", "inference_ceiling_micros", "delegation_ceiling_micros", "delegation_spend_measured", "delegation_spend_note", "wall_budget_millis", "wall_consumed_millis", "cpu_budget_millis", "cpu_consumed_millis")
    ID_FIELD_NUMBER: _ClassVar[int]
    STATE_FIELD_NUMBER: _ClassVar[int]
    CREATED_AT_FIELD_NUMBER: _ClassVar[int]
    LAST_ACTIVITY_AT_FIELD_NUMBER: _ClassVar[int]
    GRANTS_FIELD_NUMBER: _ClassVar[int]
    SANDBOX_WORKSPACE_FIELD_NUMBER: _ClassVar[int]
    RECLAIMED_REASON_FIELD_NUMBER: _ClassVar[int]
    NAME_FIELD_NUMBER: _ClassVar[int]
    INFERENCE_COST_MICROS_FIELD_NUMBER: _ClassVar[int]
    INFERENCE_TOKENS_FIELD_NUMBER: _ClassVar[int]
    DELEGATION_COST_MICROS_FIELD_NUMBER: _ClassVar[int]
    INFERENCE_CEILING_MICROS_FIELD_NUMBER: _ClassVar[int]
    DELEGATION_CEILING_MICROS_FIELD_NUMBER: _ClassVar[int]
    DELEGATION_SPEND_MEASURED_FIELD_NUMBER: _ClassVar[int]
    DELEGATION_SPEND_NOTE_FIELD_NUMBER: _ClassVar[int]
    WALL_BUDGET_MILLIS_FIELD_NUMBER: _ClassVar[int]
    WALL_CONSUMED_MILLIS_FIELD_NUMBER: _ClassVar[int]
    CPU_BUDGET_MILLIS_FIELD_NUMBER: _ClassVar[int]
    CPU_CONSUMED_MILLIS_FIELD_NUMBER: _ClassVar[int]
    id: str
    state: str
    created_at: str
    last_activity_at: str
    grants: _containers.RepeatedScalarFieldContainer[str]
    sandbox_workspace: str
    reclaimed_reason: str
    name: str
    inference_cost_micros: int
    inference_tokens: int
    delegation_cost_micros: int
    inference_ceiling_micros: int
    delegation_ceiling_micros: int
    delegation_spend_measured: bool
    delegation_spend_note: str
    wall_budget_millis: int
    wall_consumed_millis: int
    cpu_budget_millis: int
    cpu_consumed_millis: int
    def __init__(self, id: _Optional[str] = ..., state: _Optional[str] = ..., created_at: _Optional[str] = ..., last_activity_at: _Optional[str] = ..., grants: _Optional[_Iterable[str]] = ..., sandbox_workspace: _Optional[str] = ..., reclaimed_reason: _Optional[str] = ..., name: _Optional[str] = ..., inference_cost_micros: _Optional[int] = ..., inference_tokens: _Optional[int] = ..., delegation_cost_micros: _Optional[int] = ..., inference_ceiling_micros: _Optional[int] = ..., delegation_ceiling_micros: _Optional[int] = ..., delegation_spend_measured: _Optional[bool] = ..., delegation_spend_note: _Optional[str] = ..., wall_budget_millis: _Optional[int] = ..., wall_consumed_millis: _Optional[int] = ..., cpu_budget_millis: _Optional[int] = ..., cpu_consumed_millis: _Optional[int] = ...) -> None: ...

class CreateSessionRequest(_message.Message):
    __slots__ = ("grants", "sandbox_workspace", "name", "inference_ceiling_micros", "delegation_ceiling_micros", "wall_budget_millis", "cpu_budget_millis")
    GRANTS_FIELD_NUMBER: _ClassVar[int]
    SANDBOX_WORKSPACE_FIELD_NUMBER: _ClassVar[int]
    NAME_FIELD_NUMBER: _ClassVar[int]
    INFERENCE_CEILING_MICROS_FIELD_NUMBER: _ClassVar[int]
    DELEGATION_CEILING_MICROS_FIELD_NUMBER: _ClassVar[int]
    WALL_BUDGET_MILLIS_FIELD_NUMBER: _ClassVar[int]
    CPU_BUDGET_MILLIS_FIELD_NUMBER: _ClassVar[int]
    grants: _containers.RepeatedScalarFieldContainer[str]
    sandbox_workspace: str
    name: str
    inference_ceiling_micros: int
    delegation_ceiling_micros: int
    wall_budget_millis: int
    cpu_budget_millis: int
    def __init__(self, grants: _Optional[_Iterable[str]] = ..., sandbox_workspace: _Optional[str] = ..., name: _Optional[str] = ..., inference_ceiling_micros: _Optional[int] = ..., delegation_ceiling_micros: _Optional[int] = ..., wall_budget_millis: _Optional[int] = ..., cpu_budget_millis: _Optional[int] = ...) -> None: ...

class CreateSessionResponse(_message.Message):
    __slots__ = ("session",)
    SESSION_FIELD_NUMBER: _ClassVar[int]
    session: Session
    def __init__(self, session: _Optional[_Union[Session, _Mapping]] = ...) -> None: ...

class GetSessionRequest(_message.Message):
    __slots__ = ("id",)
    ID_FIELD_NUMBER: _ClassVar[int]
    id: str
    def __init__(self, id: _Optional[str] = ...) -> None: ...

class GetSessionResponse(_message.Message):
    __slots__ = ("session",)
    SESSION_FIELD_NUMBER: _ClassVar[int]
    session: Session
    def __init__(self, session: _Optional[_Union[Session, _Mapping]] = ...) -> None: ...

class ListSessionsRequest(_message.Message):
    __slots__ = ()
    def __init__(self) -> None: ...

class ListSessionsResponse(_message.Message):
    __slots__ = ("sessions", "count")
    SESSIONS_FIELD_NUMBER: _ClassVar[int]
    COUNT_FIELD_NUMBER: _ClassVar[int]
    sessions: _containers.RepeatedCompositeFieldContainer[Session]
    count: int
    def __init__(self, sessions: _Optional[_Iterable[_Union[Session, _Mapping]]] = ..., count: _Optional[int] = ...) -> None: ...

class Delegation(_message.Message):
    __slots__ = ("session_id", "execution_id", "owner", "workflow_key", "created_at", "last_status")
    SESSION_ID_FIELD_NUMBER: _ClassVar[int]
    EXECUTION_ID_FIELD_NUMBER: _ClassVar[int]
    OWNER_FIELD_NUMBER: _ClassVar[int]
    WORKFLOW_KEY_FIELD_NUMBER: _ClassVar[int]
    CREATED_AT_FIELD_NUMBER: _ClassVar[int]
    LAST_STATUS_FIELD_NUMBER: _ClassVar[int]
    session_id: str
    execution_id: str
    owner: str
    workflow_key: str
    created_at: str
    last_status: str
    def __init__(self, session_id: _Optional[str] = ..., execution_id: _Optional[str] = ..., owner: _Optional[str] = ..., workflow_key: _Optional[str] = ..., created_at: _Optional[str] = ..., last_status: _Optional[str] = ...) -> None: ...

class ListDelegationsRequest(_message.Message):
    __slots__ = ()
    def __init__(self) -> None: ...

class ListDelegationsResponse(_message.Message):
    __slots__ = ("delegations", "count")
    DELEGATIONS_FIELD_NUMBER: _ClassVar[int]
    COUNT_FIELD_NUMBER: _ClassVar[int]
    delegations: _containers.RepeatedCompositeFieldContainer[Delegation]
    count: int
    def __init__(self, delegations: _Optional[_Iterable[_Union[Delegation, _Mapping]]] = ..., count: _Optional[int] = ...) -> None: ...

class DeleteSessionRequest(_message.Message):
    __slots__ = ("id", "reason")
    ID_FIELD_NUMBER: _ClassVar[int]
    REASON_FIELD_NUMBER: _ClassVar[int]
    id: str
    reason: str
    def __init__(self, id: _Optional[str] = ..., reason: _Optional[str] = ...) -> None: ...

class DeleteSessionResponse(_message.Message):
    __slots__ = ("session",)
    SESSION_FIELD_NUMBER: _ClassVar[int]
    session: Session
    def __init__(self, session: _Optional[_Union[Session, _Mapping]] = ...) -> None: ...

class GrantSessionRequest(_message.Message):
    __slots__ = ("id", "grants")
    ID_FIELD_NUMBER: _ClassVar[int]
    GRANTS_FIELD_NUMBER: _ClassVar[int]
    id: str
    grants: _containers.RepeatedScalarFieldContainer[str]
    def __init__(self, id: _Optional[str] = ..., grants: _Optional[_Iterable[str]] = ...) -> None: ...

class GrantSessionResponse(_message.Message):
    __slots__ = ("session",)
    SESSION_FIELD_NUMBER: _ClassVar[int]
    session: Session
    def __init__(self, session: _Optional[_Union[Session, _Mapping]] = ...) -> None: ...
