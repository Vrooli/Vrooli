from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from typing import ClassVar as _ClassVar, Optional as _Optional

DESCRIPTOR: _descriptor.FileDescriptor

class FeedbackCreateRequest(_message.Message):
    __slots__ = ("type", "email", "subject", "message", "order_id")
    TYPE_FIELD_NUMBER: _ClassVar[int]
    EMAIL_FIELD_NUMBER: _ClassVar[int]
    SUBJECT_FIELD_NUMBER: _ClassVar[int]
    MESSAGE_FIELD_NUMBER: _ClassVar[int]
    ORDER_ID_FIELD_NUMBER: _ClassVar[int]
    type: str
    email: str
    subject: str
    message: str
    order_id: str
    def __init__(self, type: _Optional[str] = ..., email: _Optional[str] = ..., subject: _Optional[str] = ..., message: _Optional[str] = ..., order_id: _Optional[str] = ...) -> None: ...

class FeedbackCreateResponse(_message.Message):
    __slots__ = ("success", "id")
    SUCCESS_FIELD_NUMBER: _ClassVar[int]
    ID_FIELD_NUMBER: _ClassVar[int]
    success: bool
    id: int
    def __init__(self, success: _Optional[bool] = ..., id: _Optional[int] = ...) -> None: ...

class FeedbackError(_message.Message):
    __slots__ = ("error", "type")
    ERROR_FIELD_NUMBER: _ClassVar[int]
    TYPE_FIELD_NUMBER: _ClassVar[int]
    error: str
    type: str
    def __init__(self, error: _Optional[str] = ..., type: _Optional[str] = ...) -> None: ...
