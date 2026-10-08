from notification_hub.v1.shared import types_pb2 as _types_pb2
from google.protobuf.internal import containers as _containers
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Iterable as _Iterable, Mapping as _Mapping
from typing import ClassVar as _ClassVar, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class AskOption(_message.Message):
    __slots__ = ("key", "label")
    KEY_FIELD_NUMBER: _ClassVar[int]
    LABEL_FIELD_NUMBER: _ClassVar[int]
    key: str
    label: str
    def __init__(self, key: _Optional[str] = ..., label: _Optional[str] = ...) -> None: ...

class AskRequest(_message.Message):
    __slots__ = ("question", "allowed_answers", "deadline", "sensitivity_label", "idempotency_key", "options", "recommended", "recommendation_reason", "default_answer", "reversible", "urgency", "context_url")
    QUESTION_FIELD_NUMBER: _ClassVar[int]
    ALLOWED_ANSWERS_FIELD_NUMBER: _ClassVar[int]
    DEADLINE_FIELD_NUMBER: _ClassVar[int]
    SENSITIVITY_LABEL_FIELD_NUMBER: _ClassVar[int]
    IDEMPOTENCY_KEY_FIELD_NUMBER: _ClassVar[int]
    OPTIONS_FIELD_NUMBER: _ClassVar[int]
    RECOMMENDED_FIELD_NUMBER: _ClassVar[int]
    RECOMMENDATION_REASON_FIELD_NUMBER: _ClassVar[int]
    DEFAULT_ANSWER_FIELD_NUMBER: _ClassVar[int]
    REVERSIBLE_FIELD_NUMBER: _ClassVar[int]
    URGENCY_FIELD_NUMBER: _ClassVar[int]
    CONTEXT_URL_FIELD_NUMBER: _ClassVar[int]
    question: str
    allowed_answers: _containers.RepeatedScalarFieldContainer[str]
    deadline: str
    sensitivity_label: str
    idempotency_key: str
    options: _containers.RepeatedCompositeFieldContainer[AskOption]
    recommended: str
    recommendation_reason: str
    default_answer: str
    reversible: bool
    urgency: str
    context_url: str
    def __init__(self, question: _Optional[str] = ..., allowed_answers: _Optional[_Iterable[str]] = ..., deadline: _Optional[str] = ..., sensitivity_label: _Optional[str] = ..., idempotency_key: _Optional[str] = ..., options: _Optional[_Iterable[_Union[AskOption, _Mapping]]] = ..., recommended: _Optional[str] = ..., recommendation_reason: _Optional[str] = ..., default_answer: _Optional[str] = ..., reversible: _Optional[bool] = ..., urgency: _Optional[str] = ..., context_url: _Optional[str] = ...) -> None: ...

class AskResponse(_message.Message):
    __slots__ = ("ask_id", "notification")
    ASK_ID_FIELD_NUMBER: _ClassVar[int]
    NOTIFICATION_FIELD_NUMBER: _ClassVar[int]
    ask_id: str
    notification: _types_pb2.Notification
    def __init__(self, ask_id: _Optional[str] = ..., notification: _Optional[_Union[_types_pb2.Notification, _Mapping]] = ...) -> None: ...

class AnswerRequest(_message.Message):
    __slots__ = ("ask_id", "answer", "note")
    ASK_ID_FIELD_NUMBER: _ClassVar[int]
    ANSWER_FIELD_NUMBER: _ClassVar[int]
    NOTE_FIELD_NUMBER: _ClassVar[int]
    ask_id: str
    answer: str
    note: str
    def __init__(self, ask_id: _Optional[str] = ..., answer: _Optional[str] = ..., note: _Optional[str] = ...) -> None: ...

class AnswerResponse(_message.Message):
    __slots__ = ("ask_id", "answer", "answered_at", "late")
    ASK_ID_FIELD_NUMBER: _ClassVar[int]
    ANSWER_FIELD_NUMBER: _ClassVar[int]
    ANSWERED_AT_FIELD_NUMBER: _ClassVar[int]
    LATE_FIELD_NUMBER: _ClassVar[int]
    ask_id: str
    answer: str
    answered_at: str
    late: bool
    def __init__(self, ask_id: _Optional[str] = ..., answer: _Optional[str] = ..., answered_at: _Optional[str] = ..., late: _Optional[bool] = ...) -> None: ...

class WaitRequest(_message.Message):
    __slots__ = ("ask_id", "deadline")
    ASK_ID_FIELD_NUMBER: _ClassVar[int]
    DEADLINE_FIELD_NUMBER: _ClassVar[int]
    ask_id: str
    deadline: str
    def __init__(self, ask_id: _Optional[str] = ..., deadline: _Optional[str] = ...) -> None: ...

class WaitResponse(_message.Message):
    __slots__ = ("ask_id", "state", "answer", "reason")
    ASK_ID_FIELD_NUMBER: _ClassVar[int]
    STATE_FIELD_NUMBER: _ClassVar[int]
    ANSWER_FIELD_NUMBER: _ClassVar[int]
    REASON_FIELD_NUMBER: _ClassVar[int]
    ask_id: str
    state: str
    answer: str
    reason: str
    def __init__(self, ask_id: _Optional[str] = ..., state: _Optional[str] = ..., answer: _Optional[str] = ..., reason: _Optional[str] = ...) -> None: ...

class Ask(_message.Message):
    __slots__ = ("id", "notification_id", "question", "options", "recommended", "recommendation_reason", "default_answer", "reversible", "urgency", "context_url", "deadline", "state", "reason", "answer", "answer_label", "note", "answered_by", "answered_at", "late", "default_applied_at", "first_delivered_at", "default_eligible_at", "source", "source_event_type", "requester_ref", "resolved_at", "created_at", "updated_at")
    ID_FIELD_NUMBER: _ClassVar[int]
    NOTIFICATION_ID_FIELD_NUMBER: _ClassVar[int]
    QUESTION_FIELD_NUMBER: _ClassVar[int]
    OPTIONS_FIELD_NUMBER: _ClassVar[int]
    RECOMMENDED_FIELD_NUMBER: _ClassVar[int]
    RECOMMENDATION_REASON_FIELD_NUMBER: _ClassVar[int]
    DEFAULT_ANSWER_FIELD_NUMBER: _ClassVar[int]
    REVERSIBLE_FIELD_NUMBER: _ClassVar[int]
    URGENCY_FIELD_NUMBER: _ClassVar[int]
    CONTEXT_URL_FIELD_NUMBER: _ClassVar[int]
    DEADLINE_FIELD_NUMBER: _ClassVar[int]
    STATE_FIELD_NUMBER: _ClassVar[int]
    REASON_FIELD_NUMBER: _ClassVar[int]
    ANSWER_FIELD_NUMBER: _ClassVar[int]
    ANSWER_LABEL_FIELD_NUMBER: _ClassVar[int]
    NOTE_FIELD_NUMBER: _ClassVar[int]
    ANSWERED_BY_FIELD_NUMBER: _ClassVar[int]
    ANSWERED_AT_FIELD_NUMBER: _ClassVar[int]
    LATE_FIELD_NUMBER: _ClassVar[int]
    DEFAULT_APPLIED_AT_FIELD_NUMBER: _ClassVar[int]
    FIRST_DELIVERED_AT_FIELD_NUMBER: _ClassVar[int]
    DEFAULT_ELIGIBLE_AT_FIELD_NUMBER: _ClassVar[int]
    SOURCE_FIELD_NUMBER: _ClassVar[int]
    SOURCE_EVENT_TYPE_FIELD_NUMBER: _ClassVar[int]
    REQUESTER_REF_FIELD_NUMBER: _ClassVar[int]
    RESOLVED_AT_FIELD_NUMBER: _ClassVar[int]
    CREATED_AT_FIELD_NUMBER: _ClassVar[int]
    UPDATED_AT_FIELD_NUMBER: _ClassVar[int]
    id: str
    notification_id: str
    question: str
    options: _containers.RepeatedCompositeFieldContainer[AskOption]
    recommended: str
    recommendation_reason: str
    default_answer: str
    reversible: bool
    urgency: str
    context_url: str
    deadline: str
    state: str
    reason: str
    answer: str
    answer_label: str
    note: str
    answered_by: str
    answered_at: str
    late: bool
    default_applied_at: str
    first_delivered_at: str
    default_eligible_at: str
    source: str
    source_event_type: str
    requester_ref: str
    resolved_at: str
    created_at: str
    updated_at: str
    def __init__(self, id: _Optional[str] = ..., notification_id: _Optional[str] = ..., question: _Optional[str] = ..., options: _Optional[_Iterable[_Union[AskOption, _Mapping]]] = ..., recommended: _Optional[str] = ..., recommendation_reason: _Optional[str] = ..., default_answer: _Optional[str] = ..., reversible: _Optional[bool] = ..., urgency: _Optional[str] = ..., context_url: _Optional[str] = ..., deadline: _Optional[str] = ..., state: _Optional[str] = ..., reason: _Optional[str] = ..., answer: _Optional[str] = ..., answer_label: _Optional[str] = ..., note: _Optional[str] = ..., answered_by: _Optional[str] = ..., answered_at: _Optional[str] = ..., late: _Optional[bool] = ..., default_applied_at: _Optional[str] = ..., first_delivered_at: _Optional[str] = ..., default_eligible_at: _Optional[str] = ..., source: _Optional[str] = ..., source_event_type: _Optional[str] = ..., requester_ref: _Optional[str] = ..., resolved_at: _Optional[str] = ..., created_at: _Optional[str] = ..., updated_at: _Optional[str] = ...) -> None: ...

class GetAskRequest(_message.Message):
    __slots__ = ("ask_id",)
    ASK_ID_FIELD_NUMBER: _ClassVar[int]
    ask_id: str
    def __init__(self, ask_id: _Optional[str] = ...) -> None: ...

class GetAskResponse(_message.Message):
    __slots__ = ("ask",)
    ASK_FIELD_NUMBER: _ClassVar[int]
    ask: Ask
    def __init__(self, ask: _Optional[_Union[Ask, _Mapping]] = ...) -> None: ...

class ListAsksRequest(_message.Message):
    __slots__ = ("open_only", "limit")
    OPEN_ONLY_FIELD_NUMBER: _ClassVar[int]
    LIMIT_FIELD_NUMBER: _ClassVar[int]
    open_only: bool
    limit: int
    def __init__(self, open_only: _Optional[bool] = ..., limit: _Optional[int] = ...) -> None: ...

class ListAsksResponse(_message.Message):
    __slots__ = ("asks",)
    ASKS_FIELD_NUMBER: _ClassVar[int]
    asks: _containers.RepeatedCompositeFieldContainer[Ask]
    def __init__(self, asks: _Optional[_Iterable[_Union[Ask, _Mapping]]] = ...) -> None: ...
