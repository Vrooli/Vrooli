from google.protobuf.internal import containers as _containers
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Iterable as _Iterable, Mapping as _Mapping
from typing import ClassVar as _ClassVar, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class CheckRequest(_message.Message):
    __slots__ = ("package_id",)
    PACKAGE_ID_FIELD_NUMBER: _ClassVar[int]
    package_id: str
    def __init__(self, package_id: _Optional[str] = ...) -> None: ...

class CheckResponse(_message.Message):
    __slots__ = ("passed", "manifest_revision", "findings")
    PASSED_FIELD_NUMBER: _ClassVar[int]
    MANIFEST_REVISION_FIELD_NUMBER: _ClassVar[int]
    FINDINGS_FIELD_NUMBER: _ClassVar[int]
    passed: bool
    manifest_revision: str
    findings: _containers.RepeatedCompositeFieldContainer[Finding]
    def __init__(self, passed: _Optional[bool] = ..., manifest_revision: _Optional[str] = ..., findings: _Optional[_Iterable[_Union[Finding, _Mapping]]] = ...) -> None: ...

class Finding(_message.Message):
    __slots__ = ("code", "message", "path")
    CODE_FIELD_NUMBER: _ClassVar[int]
    MESSAGE_FIELD_NUMBER: _ClassVar[int]
    PATH_FIELD_NUMBER: _ClassVar[int]
    code: str
    message: str
    path: str
    def __init__(self, code: _Optional[str] = ..., message: _Optional[str] = ..., path: _Optional[str] = ...) -> None: ...
