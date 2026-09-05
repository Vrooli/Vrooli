import datetime

from google.protobuf import timestamp_pb2 as _timestamp_pb2
from google.protobuf.internal import containers as _containers
from google.protobuf.internal import enum_type_wrapper as _enum_type_wrapper
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Iterable as _Iterable, Mapping as _Mapping
from typing import ClassVar as _ClassVar, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class FeedbackType(int, metaclass=_enum_type_wrapper.EnumTypeWrapper):
    __slots__ = ()
    FEEDBACK_TYPE_UNSPECIFIED: _ClassVar[FeedbackType]
    FEEDBACK_TYPE_REFUND: _ClassVar[FeedbackType]
    FEEDBACK_TYPE_BUG: _ClassVar[FeedbackType]
    FEEDBACK_TYPE_FEATURE: _ClassVar[FeedbackType]
    FEEDBACK_TYPE_GENERAL: _ClassVar[FeedbackType]

class FeedbackStatus(int, metaclass=_enum_type_wrapper.EnumTypeWrapper):
    __slots__ = ()
    FEEDBACK_STATUS_UNSPECIFIED: _ClassVar[FeedbackStatus]
    FEEDBACK_STATUS_PENDING: _ClassVar[FeedbackStatus]
    FEEDBACK_STATUS_IN_PROGRESS: _ClassVar[FeedbackStatus]
    FEEDBACK_STATUS_RESOLVED: _ClassVar[FeedbackStatus]
    FEEDBACK_STATUS_REJECTED: _ClassVar[FeedbackStatus]
FEEDBACK_TYPE_UNSPECIFIED: FeedbackType
FEEDBACK_TYPE_REFUND: FeedbackType
FEEDBACK_TYPE_BUG: FeedbackType
FEEDBACK_TYPE_FEATURE: FeedbackType
FEEDBACK_TYPE_GENERAL: FeedbackType
FEEDBACK_STATUS_UNSPECIFIED: FeedbackStatus
FEEDBACK_STATUS_PENDING: FeedbackStatus
FEEDBACK_STATUS_IN_PROGRESS: FeedbackStatus
FEEDBACK_STATUS_RESOLVED: FeedbackStatus
FEEDBACK_STATUS_REJECTED: FeedbackStatus

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

class FeedbackRecord(_message.Message):
    __slots__ = ("id", "type", "email", "subject", "message", "order_id", "status", "created_at", "updated_at")
    ID_FIELD_NUMBER: _ClassVar[int]
    TYPE_FIELD_NUMBER: _ClassVar[int]
    EMAIL_FIELD_NUMBER: _ClassVar[int]
    SUBJECT_FIELD_NUMBER: _ClassVar[int]
    MESSAGE_FIELD_NUMBER: _ClassVar[int]
    ORDER_ID_FIELD_NUMBER: _ClassVar[int]
    STATUS_FIELD_NUMBER: _ClassVar[int]
    CREATED_AT_FIELD_NUMBER: _ClassVar[int]
    UPDATED_AT_FIELD_NUMBER: _ClassVar[int]
    id: int
    type: FeedbackType
    email: str
    subject: str
    message: str
    order_id: str
    status: FeedbackStatus
    created_at: _timestamp_pb2.Timestamp
    updated_at: _timestamp_pb2.Timestamp
    def __init__(self, id: _Optional[int] = ..., type: _Optional[_Union[FeedbackType, str]] = ..., email: _Optional[str] = ..., subject: _Optional[str] = ..., message: _Optional[str] = ..., order_id: _Optional[str] = ..., status: _Optional[_Union[FeedbackStatus, str]] = ..., created_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., updated_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ...) -> None: ...

class ListFeedbackRequest(_message.Message):
    __slots__ = ("status",)
    STATUS_FIELD_NUMBER: _ClassVar[int]
    status: FeedbackStatus
    def __init__(self, status: _Optional[_Union[FeedbackStatus, str]] = ...) -> None: ...

class ListFeedbackResponse(_message.Message):
    __slots__ = ("feedback",)
    FEEDBACK_FIELD_NUMBER: _ClassVar[int]
    feedback: _containers.RepeatedCompositeFieldContainer[FeedbackRecord]
    def __init__(self, feedback: _Optional[_Iterable[_Union[FeedbackRecord, _Mapping]]] = ...) -> None: ...

class GetFeedbackRequest(_message.Message):
    __slots__ = ("id",)
    ID_FIELD_NUMBER: _ClassVar[int]
    id: int
    def __init__(self, id: _Optional[int] = ...) -> None: ...

class GetFeedbackResponse(_message.Message):
    __slots__ = ("feedback",)
    FEEDBACK_FIELD_NUMBER: _ClassVar[int]
    feedback: FeedbackRecord
    def __init__(self, feedback: _Optional[_Union[FeedbackRecord, _Mapping]] = ...) -> None: ...

class UpdateFeedbackStatusRequest(_message.Message):
    __slots__ = ("id", "status")
    ID_FIELD_NUMBER: _ClassVar[int]
    STATUS_FIELD_NUMBER: _ClassVar[int]
    id: int
    status: FeedbackStatus
    def __init__(self, id: _Optional[int] = ..., status: _Optional[_Union[FeedbackStatus, str]] = ...) -> None: ...

class UpdateFeedbackStatusResponse(_message.Message):
    __slots__ = ("feedback",)
    FEEDBACK_FIELD_NUMBER: _ClassVar[int]
    feedback: FeedbackRecord
    def __init__(self, feedback: _Optional[_Union[FeedbackRecord, _Mapping]] = ...) -> None: ...

class DeleteFeedbackRequest(_message.Message):
    __slots__ = ("id",)
    ID_FIELD_NUMBER: _ClassVar[int]
    id: int
    def __init__(self, id: _Optional[int] = ...) -> None: ...

class DeleteFeedbackResponse(_message.Message):
    __slots__ = ("deleted", "id")
    DELETED_FIELD_NUMBER: _ClassVar[int]
    ID_FIELD_NUMBER: _ClassVar[int]
    deleted: bool
    id: int
    def __init__(self, deleted: _Optional[bool] = ..., id: _Optional[int] = ...) -> None: ...

class DeleteFeedbackBulkRequest(_message.Message):
    __slots__ = ("ids",)
    IDS_FIELD_NUMBER: _ClassVar[int]
    ids: _containers.RepeatedScalarFieldContainer[int]
    def __init__(self, ids: _Optional[_Iterable[int]] = ...) -> None: ...

class DeleteFeedbackBulkResponse(_message.Message):
    __slots__ = ("deleted",)
    DELETED_FIELD_NUMBER: _ClassVar[int]
    deleted: int
    def __init__(self, deleted: _Optional[int] = ...) -> None: ...
