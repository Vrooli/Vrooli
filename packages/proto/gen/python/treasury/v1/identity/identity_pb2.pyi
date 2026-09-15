from google.protobuf.internal import containers as _containers
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Iterable as _Iterable, Mapping as _Mapping
from typing import ClassVar as _ClassVar, Optional as _Optional

DESCRIPTOR: _descriptor.FileDescriptor

class VerifiedClaims(_message.Message):
    __slots__ = ("subject", "scopes", "meta")
    class MetaEntry(_message.Message):
        __slots__ = ("key", "value")
        KEY_FIELD_NUMBER: _ClassVar[int]
        VALUE_FIELD_NUMBER: _ClassVar[int]
        key: str
        value: str
        def __init__(self, key: _Optional[str] = ..., value: _Optional[str] = ...) -> None: ...
    SUBJECT_FIELD_NUMBER: _ClassVar[int]
    SCOPES_FIELD_NUMBER: _ClassVar[int]
    META_FIELD_NUMBER: _ClassVar[int]
    subject: str
    scopes: _containers.RepeatedScalarFieldContainer[str]
    meta: _containers.ScalarMap[str, str]
    def __init__(self, subject: _Optional[str] = ..., scopes: _Optional[_Iterable[str]] = ..., meta: _Optional[_Mapping[str, str]] = ...) -> None: ...
