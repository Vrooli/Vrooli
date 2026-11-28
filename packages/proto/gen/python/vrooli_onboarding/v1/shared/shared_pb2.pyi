from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from typing import ClassVar as _ClassVar, Optional as _Optional

DESCRIPTOR: _descriptor.FileDescriptor

class CompletionBlocker(_message.Message):
    __slots__ = ("kind", "name", "reason", "remediation")
    KIND_FIELD_NUMBER: _ClassVar[int]
    NAME_FIELD_NUMBER: _ClassVar[int]
    REASON_FIELD_NUMBER: _ClassVar[int]
    REMEDIATION_FIELD_NUMBER: _ClassVar[int]
    kind: str
    name: str
    reason: str
    remediation: str
    def __init__(self, kind: _Optional[str] = ..., name: _Optional[str] = ..., reason: _Optional[str] = ..., remediation: _Optional[str] = ...) -> None: ...
