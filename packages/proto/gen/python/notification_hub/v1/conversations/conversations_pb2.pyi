from notification_hub.v1.shared import types_pb2 as _types_pb2
from google.protobuf.internal import containers as _containers
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Iterable as _Iterable, Mapping as _Mapping
from typing import ClassVar as _ClassVar, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class AskRequest(_message.Message):
    __slots__ = ("question", "allowed_answers", "deadline", "sensitivity_label", "idempotency_key")
    QUESTION_FIELD_NUMBER: _ClassVar[int]
    ALLOWED_ANSWERS_FIELD_NUMBER: _ClassVar[int]
    DEADLINE_FIELD_NUMBER: _ClassVar[int]
    SENSITIVITY_LABEL_FIELD_NUMBER: _ClassVar[int]
    IDEMPOTENCY_KEY_FIELD_NUMBER: _ClassVar[int]
    question: str
    allowed_answers: _containers.RepeatedScalarFieldContainer[str]
    deadline: str
    sensitivity_label: str
    idempotency_key: str
    def __init__(self, question: _Optional[str] = ..., allowed_answers: _Optional[_Iterable[str]] = ..., deadline: _Optional[str] = ..., sensitivity_label: _Optional[str] = ..., idempotency_key: _Optional[str] = ...) -> None: ...

class AskResponse(_message.Message):
    __slots__ = ("ask_id", "notification")
    ASK_ID_FIELD_NUMBER: _ClassVar[int]
    NOTIFICATION_FIELD_NUMBER: _ClassVar[int]
    ask_id: str
    notification: _types_pb2.Notification
    def __init__(self, ask_id: _Optional[str] = ..., notification: _Optional[_Union[_types_pb2.Notification, _Mapping]] = ...) -> None: ...

class AnswerRequest(_message.Message):
    __slots__ = ("ask_id", "answer")
    ASK_ID_FIELD_NUMBER: _ClassVar[int]
    ANSWER_FIELD_NUMBER: _ClassVar[int]
    ask_id: str
    answer: str
    def __init__(self, ask_id: _Optional[str] = ..., answer: _Optional[str] = ...) -> None: ...

class AnswerResponse(_message.Message):
    __slots__ = ("ask_id", "answer", "answered_at")
    ASK_ID_FIELD_NUMBER: _ClassVar[int]
    ANSWER_FIELD_NUMBER: _ClassVar[int]
    ANSWERED_AT_FIELD_NUMBER: _ClassVar[int]
    ask_id: str
    answer: str
    answered_at: str
    def __init__(self, ask_id: _Optional[str] = ..., answer: _Optional[str] = ..., answered_at: _Optional[str] = ...) -> None: ...

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
