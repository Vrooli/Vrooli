from google.protobuf.internal import containers as _containers
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Iterable as _Iterable, Mapping as _Mapping
from typing import ClassVar as _ClassVar, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class OperationStanding(_message.Message):
    __slots__ = ("lifecycle", "terminal_outcome", "owner", "operation_id", "started_at", "last_progress_at", "elapsed_seconds", "estimated_remaining_seconds", "eta_known", "directive", "recommended_wait_seconds", "active_phase", "children", "detail", "reattach_command")
    LIFECYCLE_FIELD_NUMBER: _ClassVar[int]
    TERMINAL_OUTCOME_FIELD_NUMBER: _ClassVar[int]
    OWNER_FIELD_NUMBER: _ClassVar[int]
    OPERATION_ID_FIELD_NUMBER: _ClassVar[int]
    STARTED_AT_FIELD_NUMBER: _ClassVar[int]
    LAST_PROGRESS_AT_FIELD_NUMBER: _ClassVar[int]
    ELAPSED_SECONDS_FIELD_NUMBER: _ClassVar[int]
    ESTIMATED_REMAINING_SECONDS_FIELD_NUMBER: _ClassVar[int]
    ETA_KNOWN_FIELD_NUMBER: _ClassVar[int]
    DIRECTIVE_FIELD_NUMBER: _ClassVar[int]
    RECOMMENDED_WAIT_SECONDS_FIELD_NUMBER: _ClassVar[int]
    ACTIVE_PHASE_FIELD_NUMBER: _ClassVar[int]
    CHILDREN_FIELD_NUMBER: _ClassVar[int]
    DETAIL_FIELD_NUMBER: _ClassVar[int]
    REATTACH_COMMAND_FIELD_NUMBER: _ClassVar[int]
    lifecycle: str
    terminal_outcome: str
    owner: str
    operation_id: str
    started_at: str
    last_progress_at: str
    elapsed_seconds: float
    estimated_remaining_seconds: int
    eta_known: bool
    directive: str
    recommended_wait_seconds: int
    active_phase: str
    children: _containers.RepeatedCompositeFieldContainer[OperationStanding]
    detail: str
    reattach_command: str
    def __init__(self, lifecycle: _Optional[str] = ..., terminal_outcome: _Optional[str] = ..., owner: _Optional[str] = ..., operation_id: _Optional[str] = ..., started_at: _Optional[str] = ..., last_progress_at: _Optional[str] = ..., elapsed_seconds: _Optional[float] = ..., estimated_remaining_seconds: _Optional[int] = ..., eta_known: _Optional[bool] = ..., directive: _Optional[str] = ..., recommended_wait_seconds: _Optional[int] = ..., active_phase: _Optional[str] = ..., children: _Optional[_Iterable[_Union[OperationStanding, _Mapping]]] = ..., detail: _Optional[str] = ..., reattach_command: _Optional[str] = ...) -> None: ...
