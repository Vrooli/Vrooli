import datetime

from buf.validate import validate_pb2 as _validate_pb2
from google.protobuf import struct_pb2 as _struct_pb2
from google.protobuf import timestamp_pb2 as _timestamp_pb2
from google.protobuf.internal import containers as _containers
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Iterable as _Iterable, Mapping as _Mapping
from typing import ClassVar as _ClassVar, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class GetSessionRequest(_message.Message):
    __slots__ = ("target",)
    TARGET_FIELD_NUMBER: _ClassVar[int]
    target: str
    def __init__(self, target: _Optional[str] = ...) -> None: ...

class AdvanceSessionStepRequest(_message.Message):
    __slots__ = ("target", "step_id")
    TARGET_FIELD_NUMBER: _ClassVar[int]
    STEP_ID_FIELD_NUMBER: _ClassVar[int]
    target: str
    step_id: str
    def __init__(self, target: _Optional[str] = ..., step_id: _Optional[str] = ...) -> None: ...

class GetStepModelRequest(_message.Message):
    __slots__ = ("target",)
    TARGET_FIELD_NUMBER: _ClassVar[int]
    target: str
    def __init__(self, target: _Optional[str] = ...) -> None: ...

class GetSessionResponse(_message.Message):
    __slots__ = ("step", "step_id", "first_unsatisfied_step", "completion")
    STEP_FIELD_NUMBER: _ClassVar[int]
    STEP_ID_FIELD_NUMBER: _ClassVar[int]
    FIRST_UNSATISFIED_STEP_FIELD_NUMBER: _ClassVar[int]
    COMPLETION_FIELD_NUMBER: _ClassVar[int]
    step: int
    step_id: str
    first_unsatisfied_step: int
    completion: bool
    def __init__(self, step: _Optional[int] = ..., step_id: _Optional[str] = ..., first_unsatisfied_step: _Optional[int] = ..., completion: _Optional[bool] = ...) -> None: ...

class Step(_message.Message):
    __slots__ = ("id", "ordinal", "title", "route", "deferred")
    ID_FIELD_NUMBER: _ClassVar[int]
    ORDINAL_FIELD_NUMBER: _ClassVar[int]
    TITLE_FIELD_NUMBER: _ClassVar[int]
    ROUTE_FIELD_NUMBER: _ClassVar[int]
    DEFERRED_FIELD_NUMBER: _ClassVar[int]
    id: str
    ordinal: int
    title: str
    route: str
    deferred: bool
    def __init__(self, id: _Optional[str] = ..., ordinal: _Optional[int] = ..., title: _Optional[str] = ..., route: _Optional[str] = ..., deferred: _Optional[bool] = ...) -> None: ...

class GetStepModelResponse(_message.Message):
    __slots__ = ("steps",)
    STEPS_FIELD_NUMBER: _ClassVar[int]
    steps: _containers.RepeatedCompositeFieldContainer[Step]
    def __init__(self, steps: _Optional[_Iterable[_Union[Step, _Mapping]]] = ...) -> None: ...

class Draft(_message.Message):
    __slots__ = ("target", "actor", "base_revision", "revision", "step_id", "choices", "updated_at")
    class ChoicesEntry(_message.Message):
        __slots__ = ("key", "value")
        KEY_FIELD_NUMBER: _ClassVar[int]
        VALUE_FIELD_NUMBER: _ClassVar[int]
        key: str
        value: str
        def __init__(self, key: _Optional[str] = ..., value: _Optional[str] = ...) -> None: ...
    TARGET_FIELD_NUMBER: _ClassVar[int]
    ACTOR_FIELD_NUMBER: _ClassVar[int]
    BASE_REVISION_FIELD_NUMBER: _ClassVar[int]
    REVISION_FIELD_NUMBER: _ClassVar[int]
    STEP_ID_FIELD_NUMBER: _ClassVar[int]
    CHOICES_FIELD_NUMBER: _ClassVar[int]
    UPDATED_AT_FIELD_NUMBER: _ClassVar[int]
    target: str
    actor: str
    base_revision: str
    revision: str
    step_id: str
    choices: _containers.ScalarMap[str, str]
    updated_at: _timestamp_pb2.Timestamp
    def __init__(self, target: _Optional[str] = ..., actor: _Optional[str] = ..., base_revision: _Optional[str] = ..., revision: _Optional[str] = ..., step_id: _Optional[str] = ..., choices: _Optional[_Mapping[str, str]] = ..., updated_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ...) -> None: ...

class GetDraftRequest(_message.Message):
    __slots__ = ("target", "actor")
    TARGET_FIELD_NUMBER: _ClassVar[int]
    ACTOR_FIELD_NUMBER: _ClassVar[int]
    target: str
    actor: str
    def __init__(self, target: _Optional[str] = ..., actor: _Optional[str] = ...) -> None: ...

class SaveDraftRequest(_message.Message):
    __slots__ = ("target", "actor", "expected_revision", "base_revision", "step_id", "choices")
    class ChoicesEntry(_message.Message):
        __slots__ = ("key", "value")
        KEY_FIELD_NUMBER: _ClassVar[int]
        VALUE_FIELD_NUMBER: _ClassVar[int]
        key: str
        value: str
        def __init__(self, key: _Optional[str] = ..., value: _Optional[str] = ...) -> None: ...
    TARGET_FIELD_NUMBER: _ClassVar[int]
    ACTOR_FIELD_NUMBER: _ClassVar[int]
    EXPECTED_REVISION_FIELD_NUMBER: _ClassVar[int]
    BASE_REVISION_FIELD_NUMBER: _ClassVar[int]
    STEP_ID_FIELD_NUMBER: _ClassVar[int]
    CHOICES_FIELD_NUMBER: _ClassVar[int]
    target: str
    actor: str
    expected_revision: str
    base_revision: str
    step_id: str
    choices: _containers.ScalarMap[str, str]
    def __init__(self, target: _Optional[str] = ..., actor: _Optional[str] = ..., expected_revision: _Optional[str] = ..., base_revision: _Optional[str] = ..., step_id: _Optional[str] = ..., choices: _Optional[_Mapping[str, str]] = ...) -> None: ...

class DiscardDraftRequest(_message.Message):
    __slots__ = ("target", "actor")
    TARGET_FIELD_NUMBER: _ClassVar[int]
    ACTOR_FIELD_NUMBER: _ClassVar[int]
    target: str
    actor: str
    def __init__(self, target: _Optional[str] = ..., actor: _Optional[str] = ...) -> None: ...

class GetDraftResponse(_message.Message):
    __slots__ = ("draft",)
    DRAFT_FIELD_NUMBER: _ClassVar[int]
    draft: Draft
    def __init__(self, draft: _Optional[_Union[Draft, _Mapping]] = ...) -> None: ...

class ProfileSession(_message.Message):
    __slots__ = ("target", "actor", "mode", "profile_id", "profile_version", "catalog_revision", "base_revision", "answers", "manual_decisions", "target_context", "updated_at", "revision", "reconciliation_state", "current_profile_version", "reconciliation_reasons", "consequence_digest", "next_question_id", "next_action", "reconciliation_changes")
    class ManualDecisionsEntry(_message.Message):
        __slots__ = ("key", "value")
        KEY_FIELD_NUMBER: _ClassVar[int]
        VALUE_FIELD_NUMBER: _ClassVar[int]
        key: str
        value: bool
        def __init__(self, key: _Optional[str] = ..., value: _Optional[bool] = ...) -> None: ...
    TARGET_FIELD_NUMBER: _ClassVar[int]
    ACTOR_FIELD_NUMBER: _ClassVar[int]
    MODE_FIELD_NUMBER: _ClassVar[int]
    PROFILE_ID_FIELD_NUMBER: _ClassVar[int]
    PROFILE_VERSION_FIELD_NUMBER: _ClassVar[int]
    CATALOG_REVISION_FIELD_NUMBER: _ClassVar[int]
    BASE_REVISION_FIELD_NUMBER: _ClassVar[int]
    ANSWERS_FIELD_NUMBER: _ClassVar[int]
    MANUAL_DECISIONS_FIELD_NUMBER: _ClassVar[int]
    TARGET_CONTEXT_FIELD_NUMBER: _ClassVar[int]
    UPDATED_AT_FIELD_NUMBER: _ClassVar[int]
    REVISION_FIELD_NUMBER: _ClassVar[int]
    RECONCILIATION_STATE_FIELD_NUMBER: _ClassVar[int]
    CURRENT_PROFILE_VERSION_FIELD_NUMBER: _ClassVar[int]
    RECONCILIATION_REASONS_FIELD_NUMBER: _ClassVar[int]
    CONSEQUENCE_DIGEST_FIELD_NUMBER: _ClassVar[int]
    NEXT_QUESTION_ID_FIELD_NUMBER: _ClassVar[int]
    NEXT_ACTION_FIELD_NUMBER: _ClassVar[int]
    RECONCILIATION_CHANGES_FIELD_NUMBER: _ClassVar[int]
    target: str
    actor: str
    mode: str
    profile_id: str
    profile_version: str
    catalog_revision: str
    base_revision: str
    answers: _struct_pb2.Struct
    manual_decisions: _containers.ScalarMap[str, bool]
    target_context: _struct_pb2.Struct
    updated_at: _timestamp_pb2.Timestamp
    revision: str
    reconciliation_state: str
    current_profile_version: str
    reconciliation_reasons: _containers.RepeatedScalarFieldContainer[str]
    consequence_digest: str
    next_question_id: str
    next_action: str
    reconciliation_changes: _containers.RepeatedCompositeFieldContainer[ReconciliationChange]
    def __init__(self, target: _Optional[str] = ..., actor: _Optional[str] = ..., mode: _Optional[str] = ..., profile_id: _Optional[str] = ..., profile_version: _Optional[str] = ..., catalog_revision: _Optional[str] = ..., base_revision: _Optional[str] = ..., answers: _Optional[_Union[_struct_pb2.Struct, _Mapping]] = ..., manual_decisions: _Optional[_Mapping[str, bool]] = ..., target_context: _Optional[_Union[_struct_pb2.Struct, _Mapping]] = ..., updated_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., revision: _Optional[str] = ..., reconciliation_state: _Optional[str] = ..., current_profile_version: _Optional[str] = ..., reconciliation_reasons: _Optional[_Iterable[str]] = ..., consequence_digest: _Optional[str] = ..., next_question_id: _Optional[str] = ..., next_action: _Optional[str] = ..., reconciliation_changes: _Optional[_Iterable[_Union[ReconciliationChange, _Mapping]]] = ...) -> None: ...

class ReconciliationChange(_message.Message):
    __slots__ = ("kind", "field", "before", "after", "impact", "requires_review")
    KIND_FIELD_NUMBER: _ClassVar[int]
    FIELD_FIELD_NUMBER: _ClassVar[int]
    BEFORE_FIELD_NUMBER: _ClassVar[int]
    AFTER_FIELD_NUMBER: _ClassVar[int]
    IMPACT_FIELD_NUMBER: _ClassVar[int]
    REQUIRES_REVIEW_FIELD_NUMBER: _ClassVar[int]
    kind: str
    field: str
    before: str
    after: str
    impact: str
    requires_review: bool
    def __init__(self, kind: _Optional[str] = ..., field: _Optional[str] = ..., before: _Optional[str] = ..., after: _Optional[str] = ..., impact: _Optional[str] = ..., requires_review: _Optional[bool] = ...) -> None: ...

class GetProfileSessionRequest(_message.Message):
    __slots__ = ("target", "actor")
    TARGET_FIELD_NUMBER: _ClassVar[int]
    ACTOR_FIELD_NUMBER: _ClassVar[int]
    target: str
    actor: str
    def __init__(self, target: _Optional[str] = ..., actor: _Optional[str] = ...) -> None: ...

class SaveProfileSessionRequest(_message.Message):
    __slots__ = ("target", "actor", "expected_revision", "session")
    TARGET_FIELD_NUMBER: _ClassVar[int]
    ACTOR_FIELD_NUMBER: _ClassVar[int]
    EXPECTED_REVISION_FIELD_NUMBER: _ClassVar[int]
    SESSION_FIELD_NUMBER: _ClassVar[int]
    target: str
    actor: str
    expected_revision: str
    session: ProfileSession
    def __init__(self, target: _Optional[str] = ..., actor: _Optional[str] = ..., expected_revision: _Optional[str] = ..., session: _Optional[_Union[ProfileSession, _Mapping]] = ...) -> None: ...

class GetProfileSessionResponse(_message.Message):
    __slots__ = ("session",)
    SESSION_FIELD_NUMBER: _ClassVar[int]
    session: ProfileSession
    def __init__(self, session: _Optional[_Union[ProfileSession, _Mapping]] = ...) -> None: ...
