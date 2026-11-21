import datetime

from google.protobuf import timestamp_pb2 as _timestamp_pb2
from google.protobuf.internal import containers as _containers
from google.protobuf.internal import enum_type_wrapper as _enum_type_wrapper
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Iterable as _Iterable, Mapping as _Mapping
from typing import ClassVar as _ClassVar, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class HandoffState(int, metaclass=_enum_type_wrapper.EnumTypeWrapper):
    __slots__ = ()
    HANDOFF_STATE_UNSPECIFIED: _ClassVar[HandoffState]
    HANDOFF_STATE_OPEN: _ClassVar[HandoffState]
    HANDOFF_STATE_DELIVERED: _ClassVar[HandoffState]
    HANDOFF_STATE_AWAITING_HUMAN: _ClassVar[HandoffState]
    HANDOFF_STATE_COMPLETED: _ClassVar[HandoffState]
    HANDOFF_STATE_EXPIRED: _ClassVar[HandoffState]
    HANDOFF_STATE_CANCELLED: _ClassVar[HandoffState]
    HANDOFF_STATE_RESUMED: _ClassVar[HandoffState]
HANDOFF_STATE_UNSPECIFIED: HandoffState
HANDOFF_STATE_OPEN: HandoffState
HANDOFF_STATE_DELIVERED: HandoffState
HANDOFF_STATE_AWAITING_HUMAN: HandoffState
HANDOFF_STATE_COMPLETED: HandoffState
HANDOFF_STATE_EXPIRED: HandoffState
HANDOFF_STATE_CANCELLED: HandoffState
HANDOFF_STATE_RESUMED: HandoffState

class CheckpointField(_message.Message):
    __slots__ = ("name", "value")
    NAME_FIELD_NUMBER: _ClassVar[int]
    VALUE_FIELD_NUMBER: _ClassVar[int]
    name: str
    value: str
    def __init__(self, name: _Optional[str] = ..., value: _Optional[str] = ...) -> None: ...

class Checkpoint(_message.Message):
    __slots__ = ("completed_fields", "required_document_ids", "resume_token")
    COMPLETED_FIELDS_FIELD_NUMBER: _ClassVar[int]
    REQUIRED_DOCUMENT_IDS_FIELD_NUMBER: _ClassVar[int]
    RESUME_TOKEN_FIELD_NUMBER: _ClassVar[int]
    completed_fields: _containers.RepeatedCompositeFieldContainer[CheckpointField]
    required_document_ids: _containers.RepeatedScalarFieldContainer[str]
    resume_token: str
    def __init__(self, completed_fields: _Optional[_Iterable[_Union[CheckpointField, _Mapping]]] = ..., required_document_ids: _Optional[_Iterable[str]] = ..., resume_token: _Optional[str] = ...) -> None: ...

class Handoff(_message.Message):
    __slots__ = ("id", "persona_id", "kind", "title", "human_action", "checkpoint", "state", "opened_by_run_id", "authorising_human", "deadline", "created_at", "updated_at", "relay_state")
    ID_FIELD_NUMBER: _ClassVar[int]
    PERSONA_ID_FIELD_NUMBER: _ClassVar[int]
    KIND_FIELD_NUMBER: _ClassVar[int]
    TITLE_FIELD_NUMBER: _ClassVar[int]
    HUMAN_ACTION_FIELD_NUMBER: _ClassVar[int]
    CHECKPOINT_FIELD_NUMBER: _ClassVar[int]
    STATE_FIELD_NUMBER: _ClassVar[int]
    OPENED_BY_RUN_ID_FIELD_NUMBER: _ClassVar[int]
    AUTHORISING_HUMAN_FIELD_NUMBER: _ClassVar[int]
    DEADLINE_FIELD_NUMBER: _ClassVar[int]
    CREATED_AT_FIELD_NUMBER: _ClassVar[int]
    UPDATED_AT_FIELD_NUMBER: _ClassVar[int]
    RELAY_STATE_FIELD_NUMBER: _ClassVar[int]
    id: str
    persona_id: str
    kind: str
    title: str
    human_action: str
    checkpoint: Checkpoint
    state: HandoffState
    opened_by_run_id: str
    authorising_human: str
    deadline: _timestamp_pb2.Timestamp
    created_at: _timestamp_pb2.Timestamp
    updated_at: _timestamp_pb2.Timestamp
    relay_state: str
    def __init__(self, id: _Optional[str] = ..., persona_id: _Optional[str] = ..., kind: _Optional[str] = ..., title: _Optional[str] = ..., human_action: _Optional[str] = ..., checkpoint: _Optional[_Union[Checkpoint, _Mapping]] = ..., state: _Optional[_Union[HandoffState, str]] = ..., opened_by_run_id: _Optional[str] = ..., authorising_human: _Optional[str] = ..., deadline: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., created_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., updated_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., relay_state: _Optional[str] = ...) -> None: ...

class OpenHandoffRequest(_message.Message):
    __slots__ = ("persona_id", "kind", "title", "human_action", "checkpoint", "deadline")
    PERSONA_ID_FIELD_NUMBER: _ClassVar[int]
    KIND_FIELD_NUMBER: _ClassVar[int]
    TITLE_FIELD_NUMBER: _ClassVar[int]
    HUMAN_ACTION_FIELD_NUMBER: _ClassVar[int]
    CHECKPOINT_FIELD_NUMBER: _ClassVar[int]
    DEADLINE_FIELD_NUMBER: _ClassVar[int]
    persona_id: str
    kind: str
    title: str
    human_action: str
    checkpoint: Checkpoint
    deadline: _timestamp_pb2.Timestamp
    def __init__(self, persona_id: _Optional[str] = ..., kind: _Optional[str] = ..., title: _Optional[str] = ..., human_action: _Optional[str] = ..., checkpoint: _Optional[_Union[Checkpoint, _Mapping]] = ..., deadline: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ...) -> None: ...

class OpenHandoffResponse(_message.Message):
    __slots__ = ("handoff",)
    HANDOFF_FIELD_NUMBER: _ClassVar[int]
    handoff: Handoff
    def __init__(self, handoff: _Optional[_Union[Handoff, _Mapping]] = ...) -> None: ...

class GetHandoffRequest(_message.Message):
    __slots__ = ("handoff_id",)
    HANDOFF_ID_FIELD_NUMBER: _ClassVar[int]
    handoff_id: str
    def __init__(self, handoff_id: _Optional[str] = ...) -> None: ...

class GetHandoffResponse(_message.Message):
    __slots__ = ("handoff",)
    HANDOFF_FIELD_NUMBER: _ClassVar[int]
    handoff: Handoff
    def __init__(self, handoff: _Optional[_Union[Handoff, _Mapping]] = ...) -> None: ...

class ListHandoffsRequest(_message.Message):
    __slots__ = ("persona_id", "limit")
    PERSONA_ID_FIELD_NUMBER: _ClassVar[int]
    LIMIT_FIELD_NUMBER: _ClassVar[int]
    persona_id: str
    limit: int
    def __init__(self, persona_id: _Optional[str] = ..., limit: _Optional[int] = ...) -> None: ...

class ListHandoffsResponse(_message.Message):
    __slots__ = ("handoffs",)
    HANDOFFS_FIELD_NUMBER: _ClassVar[int]
    handoffs: _containers.RepeatedCompositeFieldContainer[Handoff]
    def __init__(self, handoffs: _Optional[_Iterable[_Union[Handoff, _Mapping]]] = ...) -> None: ...

class CompleteHandoffRequest(_message.Message):
    __slots__ = ("handoff_id", "completed_by")
    HANDOFF_ID_FIELD_NUMBER: _ClassVar[int]
    COMPLETED_BY_FIELD_NUMBER: _ClassVar[int]
    handoff_id: str
    completed_by: str
    def __init__(self, handoff_id: _Optional[str] = ..., completed_by: _Optional[str] = ...) -> None: ...

class CompleteHandoffResponse(_message.Message):
    __slots__ = ("handoff",)
    HANDOFF_FIELD_NUMBER: _ClassVar[int]
    handoff: Handoff
    def __init__(self, handoff: _Optional[_Union[Handoff, _Mapping]] = ...) -> None: ...

class CancelHandoffRequest(_message.Message):
    __slots__ = ("handoff_id", "cancelled_by")
    HANDOFF_ID_FIELD_NUMBER: _ClassVar[int]
    CANCELLED_BY_FIELD_NUMBER: _ClassVar[int]
    handoff_id: str
    cancelled_by: str
    def __init__(self, handoff_id: _Optional[str] = ..., cancelled_by: _Optional[str] = ...) -> None: ...

class CancelHandoffResponse(_message.Message):
    __slots__ = ("handoff",)
    HANDOFF_FIELD_NUMBER: _ClassVar[int]
    handoff: Handoff
    def __init__(self, handoff: _Optional[_Union[Handoff, _Mapping]] = ...) -> None: ...

class ResumeHandoffRequest(_message.Message):
    __slots__ = ("handoff_id", "run_id")
    HANDOFF_ID_FIELD_NUMBER: _ClassVar[int]
    RUN_ID_FIELD_NUMBER: _ClassVar[int]
    handoff_id: str
    run_id: str
    def __init__(self, handoff_id: _Optional[str] = ..., run_id: _Optional[str] = ...) -> None: ...

class ResumeHandoffResponse(_message.Message):
    __slots__ = ("handoff",)
    HANDOFF_FIELD_NUMBER: _ClassVar[int]
    handoff: Handoff
    def __init__(self, handoff: _Optional[_Union[Handoff, _Mapping]] = ...) -> None: ...

class EnrolmentField(_message.Message):
    __slots__ = ("name", "value", "human_only")
    NAME_FIELD_NUMBER: _ClassVar[int]
    VALUE_FIELD_NUMBER: _ClassVar[int]
    HUMAN_ONLY_FIELD_NUMBER: _ClassVar[int]
    name: str
    value: str
    human_only: bool
    def __init__(self, name: _Optional[str] = ..., value: _Optional[str] = ..., human_only: _Optional[bool] = ...) -> None: ...

class PrepareEnrolmentRequest(_message.Message):
    __slots__ = ("persona_id", "target", "required_fields")
    PERSONA_ID_FIELD_NUMBER: _ClassVar[int]
    TARGET_FIELD_NUMBER: _ClassVar[int]
    REQUIRED_FIELDS_FIELD_NUMBER: _ClassVar[int]
    persona_id: str
    target: str
    required_fields: _containers.RepeatedScalarFieldContainer[str]
    def __init__(self, persona_id: _Optional[str] = ..., target: _Optional[str] = ..., required_fields: _Optional[_Iterable[str]] = ...) -> None: ...

class PrepareEnrolmentResponse(_message.Message):
    __slots__ = ("fields", "handoff_id")
    FIELDS_FIELD_NUMBER: _ClassVar[int]
    HANDOFF_ID_FIELD_NUMBER: _ClassVar[int]
    fields: _containers.RepeatedCompositeFieldContainer[EnrolmentField]
    handoff_id: str
    def __init__(self, fields: _Optional[_Iterable[_Union[EnrolmentField, _Mapping]]] = ..., handoff_id: _Optional[str] = ...) -> None: ...
