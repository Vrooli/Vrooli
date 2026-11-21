from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from typing import ClassVar as _ClassVar, Optional as _Optional

DESCRIPTOR: _descriptor.FileDescriptor

class ReportFrictionRequest(_message.Message):
    __slots__ = ("fingerprint", "since")
    FINGERPRINT_FIELD_NUMBER: _ClassVar[int]
    SINCE_FIELD_NUMBER: _ClassVar[int]
    fingerprint: str
    since: str
    def __init__(self, fingerprint: _Optional[str] = ..., since: _Optional[str] = ...) -> None: ...

class ReportFrictionResponse(_message.Message):
    __slots__ = ("fingerprint", "description", "status", "bug_id")
    FINGERPRINT_FIELD_NUMBER: _ClassVar[int]
    DESCRIPTION_FIELD_NUMBER: _ClassVar[int]
    STATUS_FIELD_NUMBER: _ClassVar[int]
    BUG_ID_FIELD_NUMBER: _ClassVar[int]
    fingerprint: str
    description: str
    status: str
    bug_id: str
    def __init__(self, fingerprint: _Optional[str] = ..., description: _Optional[str] = ..., status: _Optional[str] = ..., bug_id: _Optional[str] = ...) -> None: ...

class DismissFrictionRequest(_message.Message):
    __slots__ = ("fingerprint", "since")
    FINGERPRINT_FIELD_NUMBER: _ClassVar[int]
    SINCE_FIELD_NUMBER: _ClassVar[int]
    fingerprint: str
    since: str
    def __init__(self, fingerprint: _Optional[str] = ..., since: _Optional[str] = ...) -> None: ...

class DismissFrictionResponse(_message.Message):
    __slots__ = ("fingerprint", "status")
    FINGERPRINT_FIELD_NUMBER: _ClassVar[int]
    STATUS_FIELD_NUMBER: _ClassVar[int]
    fingerprint: str
    status: str
    def __init__(self, fingerprint: _Optional[str] = ..., status: _Optional[str] = ...) -> None: ...
