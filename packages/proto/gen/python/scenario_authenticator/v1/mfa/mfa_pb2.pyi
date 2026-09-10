import datetime

from google.protobuf import timestamp_pb2 as _timestamp_pb2
from google.protobuf.internal import containers as _containers
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Iterable as _Iterable, Mapping as _Mapping
from typing import ClassVar as _ClassVar, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class BeginEnrollmentRequest(_message.Message):
    __slots__ = ("access_token",)
    ACCESS_TOKEN_FIELD_NUMBER: _ClassVar[int]
    access_token: str
    def __init__(self, access_token: _Optional[str] = ...) -> None: ...

class BeginEnrollmentResponse(_message.Message):
    __slots__ = ("enrollment_id", "provisioning_uri", "expires_at")
    ENROLLMENT_ID_FIELD_NUMBER: _ClassVar[int]
    PROVISIONING_URI_FIELD_NUMBER: _ClassVar[int]
    EXPIRES_AT_FIELD_NUMBER: _ClassVar[int]
    enrollment_id: str
    provisioning_uri: str
    expires_at: _timestamp_pb2.Timestamp
    def __init__(self, enrollment_id: _Optional[str] = ..., provisioning_uri: _Optional[str] = ..., expires_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ...) -> None: ...

class ConfirmEnrollmentRequest(_message.Message):
    __slots__ = ("access_token", "enrollment_id", "totp_code")
    ACCESS_TOKEN_FIELD_NUMBER: _ClassVar[int]
    ENROLLMENT_ID_FIELD_NUMBER: _ClassVar[int]
    TOTP_CODE_FIELD_NUMBER: _ClassVar[int]
    access_token: str
    enrollment_id: str
    totp_code: str
    def __init__(self, access_token: _Optional[str] = ..., enrollment_id: _Optional[str] = ..., totp_code: _Optional[str] = ...) -> None: ...

class ConfirmEnrollmentResponse(_message.Message):
    __slots__ = ("recovery_codes", "enrolled_at")
    RECOVERY_CODES_FIELD_NUMBER: _ClassVar[int]
    ENROLLED_AT_FIELD_NUMBER: _ClassVar[int]
    recovery_codes: _containers.RepeatedScalarFieldContainer[str]
    enrolled_at: _timestamp_pb2.Timestamp
    def __init__(self, recovery_codes: _Optional[_Iterable[str]] = ..., enrolled_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ...) -> None: ...

class RemoveEnrollmentRequest(_message.Message):
    __slots__ = ("access_token",)
    ACCESS_TOKEN_FIELD_NUMBER: _ClassVar[int]
    access_token: str
    def __init__(self, access_token: _Optional[str] = ...) -> None: ...

class RemoveEnrollmentResponse(_message.Message):
    __slots__ = ()
    def __init__(self) -> None: ...
