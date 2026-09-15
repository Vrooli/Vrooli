import datetime

from google.protobuf import timestamp_pb2 as _timestamp_pb2
from google.protobuf.internal import containers as _containers
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Mapping as _Mapping
from typing import ClassVar as _ClassVar, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class RequestAccessRequest(_message.Message):
    __slots__ = ("grant_id", "item_id", "operation", "duration_seconds")
    GRANT_ID_FIELD_NUMBER: _ClassVar[int]
    ITEM_ID_FIELD_NUMBER: _ClassVar[int]
    OPERATION_FIELD_NUMBER: _ClassVar[int]
    DURATION_SECONDS_FIELD_NUMBER: _ClassVar[int]
    grant_id: str
    item_id: str
    operation: str
    duration_seconds: int
    def __init__(self, grant_id: _Optional[str] = ..., item_id: _Optional[str] = ..., operation: _Optional[str] = ..., duration_seconds: _Optional[int] = ...) -> None: ...

class AccessRequest(_message.Message):
    __slots__ = ("id", "workspace_id", "grant_id", "item_id", "operation", "request_digest", "status", "requested_by", "expires_at", "created_at")
    ID_FIELD_NUMBER: _ClassVar[int]
    WORKSPACE_ID_FIELD_NUMBER: _ClassVar[int]
    GRANT_ID_FIELD_NUMBER: _ClassVar[int]
    ITEM_ID_FIELD_NUMBER: _ClassVar[int]
    OPERATION_FIELD_NUMBER: _ClassVar[int]
    REQUEST_DIGEST_FIELD_NUMBER: _ClassVar[int]
    STATUS_FIELD_NUMBER: _ClassVar[int]
    REQUESTED_BY_FIELD_NUMBER: _ClassVar[int]
    EXPIRES_AT_FIELD_NUMBER: _ClassVar[int]
    CREATED_AT_FIELD_NUMBER: _ClassVar[int]
    id: str
    workspace_id: str
    grant_id: str
    item_id: str
    operation: str
    request_digest: str
    status: str
    requested_by: str
    expires_at: _timestamp_pb2.Timestamp
    created_at: _timestamp_pb2.Timestamp
    def __init__(self, id: _Optional[str] = ..., workspace_id: _Optional[str] = ..., grant_id: _Optional[str] = ..., item_id: _Optional[str] = ..., operation: _Optional[str] = ..., request_digest: _Optional[str] = ..., status: _Optional[str] = ..., requested_by: _Optional[str] = ..., expires_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., created_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ...) -> None: ...

class GetAccessRequestRequest(_message.Message):
    __slots__ = ("request_id",)
    REQUEST_ID_FIELD_NUMBER: _ClassVar[int]
    request_id: str
    def __init__(self, request_id: _Optional[str] = ...) -> None: ...

class WaitAccessRequestRequest(_message.Message):
    __slots__ = ("request_id", "timeout_seconds")
    REQUEST_ID_FIELD_NUMBER: _ClassVar[int]
    TIMEOUT_SECONDS_FIELD_NUMBER: _ClassVar[int]
    request_id: str
    timeout_seconds: int
    def __init__(self, request_id: _Optional[str] = ..., timeout_seconds: _Optional[int] = ...) -> None: ...

class WaitAccessRequestResponse(_message.Message):
    __slots__ = ("request", "timed_out")
    REQUEST_FIELD_NUMBER: _ClassVar[int]
    TIMED_OUT_FIELD_NUMBER: _ClassVar[int]
    request: AccessRequest
    timed_out: bool
    def __init__(self, request: _Optional[_Union[AccessRequest, _Mapping]] = ..., timed_out: _Optional[bool] = ...) -> None: ...

class CreateBrokerSessionRequest(_message.Message):
    __slots__ = ("grant_id", "item_id", "target_origin", "allow_internal", "duration_seconds", "one_use")
    GRANT_ID_FIELD_NUMBER: _ClassVar[int]
    ITEM_ID_FIELD_NUMBER: _ClassVar[int]
    TARGET_ORIGIN_FIELD_NUMBER: _ClassVar[int]
    ALLOW_INTERNAL_FIELD_NUMBER: _ClassVar[int]
    DURATION_SECONDS_FIELD_NUMBER: _ClassVar[int]
    ONE_USE_FIELD_NUMBER: _ClassVar[int]
    grant_id: str
    item_id: str
    target_origin: str
    allow_internal: bool
    duration_seconds: int
    one_use: bool
    def __init__(self, grant_id: _Optional[str] = ..., item_id: _Optional[str] = ..., target_origin: _Optional[str] = ..., allow_internal: _Optional[bool] = ..., duration_seconds: _Optional[int] = ..., one_use: _Optional[bool] = ...) -> None: ...

class BrokerSession(_message.Message):
    __slots__ = ("session_id", "session_token", "expires_at", "target_origin", "secret_exposure")
    SESSION_ID_FIELD_NUMBER: _ClassVar[int]
    SESSION_TOKEN_FIELD_NUMBER: _ClassVar[int]
    EXPIRES_AT_FIELD_NUMBER: _ClassVar[int]
    TARGET_ORIGIN_FIELD_NUMBER: _ClassVar[int]
    SECRET_EXPOSURE_FIELD_NUMBER: _ClassVar[int]
    session_id: str
    session_token: str
    expires_at: _timestamp_pb2.Timestamp
    target_origin: str
    secret_exposure: str
    def __init__(self, session_id: _Optional[str] = ..., session_token: _Optional[str] = ..., expires_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., target_origin: _Optional[str] = ..., secret_exposure: _Optional[str] = ...) -> None: ...

class ExecuteBrokerOperationRequest(_message.Message):
    __slots__ = ("session_id", "session_token", "method", "path", "headers", "body")
    class HeadersEntry(_message.Message):
        __slots__ = ("key", "value")
        KEY_FIELD_NUMBER: _ClassVar[int]
        VALUE_FIELD_NUMBER: _ClassVar[int]
        key: str
        value: str
        def __init__(self, key: _Optional[str] = ..., value: _Optional[str] = ...) -> None: ...
    SESSION_ID_FIELD_NUMBER: _ClassVar[int]
    SESSION_TOKEN_FIELD_NUMBER: _ClassVar[int]
    METHOD_FIELD_NUMBER: _ClassVar[int]
    PATH_FIELD_NUMBER: _ClassVar[int]
    HEADERS_FIELD_NUMBER: _ClassVar[int]
    BODY_FIELD_NUMBER: _ClassVar[int]
    session_id: str
    session_token: str
    method: str
    path: str
    headers: _containers.ScalarMap[str, str]
    body: str
    def __init__(self, session_id: _Optional[str] = ..., session_token: _Optional[str] = ..., method: _Optional[str] = ..., path: _Optional[str] = ..., headers: _Optional[_Mapping[str, str]] = ..., body: _Optional[str] = ...) -> None: ...

class BrokerOperationResult(_message.Message):
    __slots__ = ("status", "headers", "body", "body_truncated", "projection")
    class HeadersEntry(_message.Message):
        __slots__ = ("key", "value")
        KEY_FIELD_NUMBER: _ClassVar[int]
        VALUE_FIELD_NUMBER: _ClassVar[int]
        key: str
        value: str
        def __init__(self, key: _Optional[str] = ..., value: _Optional[str] = ...) -> None: ...
    STATUS_FIELD_NUMBER: _ClassVar[int]
    HEADERS_FIELD_NUMBER: _ClassVar[int]
    BODY_FIELD_NUMBER: _ClassVar[int]
    BODY_TRUNCATED_FIELD_NUMBER: _ClassVar[int]
    PROJECTION_FIELD_NUMBER: _ClassVar[int]
    status: int
    headers: _containers.ScalarMap[str, str]
    body: str
    body_truncated: bool
    projection: str
    def __init__(self, status: _Optional[int] = ..., headers: _Optional[_Mapping[str, str]] = ..., body: _Optional[str] = ..., body_truncated: _Optional[bool] = ..., projection: _Optional[str] = ...) -> None: ...

class RevokeBrokerSessionRequest(_message.Message):
    __slots__ = ("session_id",)
    SESSION_ID_FIELD_NUMBER: _ClassVar[int]
    session_id: str
    def __init__(self, session_id: _Optional[str] = ...) -> None: ...

class RevokeBrokerSessionResponse(_message.Message):
    __slots__ = ("session_id", "status")
    SESSION_ID_FIELD_NUMBER: _ClassVar[int]
    STATUS_FIELD_NUMBER: _ClassVar[int]
    session_id: str
    status: str
    def __init__(self, session_id: _Optional[str] = ..., status: _Optional[str] = ...) -> None: ...
