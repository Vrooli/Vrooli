from google.protobuf.internal import containers as _containers
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Iterable as _Iterable, Mapping as _Mapping
from typing import ClassVar as _ClassVar, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class ScanCandidateRequest(_message.Message):
    __slots__ = ("files",)
    class FilesEntry(_message.Message):
        __slots__ = ("key", "value")
        KEY_FIELD_NUMBER: _ClassVar[int]
        VALUE_FIELD_NUMBER: _ClassVar[int]
        key: str
        value: bytes
        def __init__(self, key: _Optional[str] = ..., value: _Optional[bytes] = ...) -> None: ...
    FILES_FIELD_NUMBER: _ClassVar[int]
    files: _containers.ScalarMap[str, bytes]
    def __init__(self, files: _Optional[_Mapping[str, bytes]] = ...) -> None: ...

class CandidateFinding(_message.Message):
    __slots__ = ("rule_id", "severity", "title", "description", "remediation", "file_path", "scanner")
    RULE_ID_FIELD_NUMBER: _ClassVar[int]
    SEVERITY_FIELD_NUMBER: _ClassVar[int]
    TITLE_FIELD_NUMBER: _ClassVar[int]
    DESCRIPTION_FIELD_NUMBER: _ClassVar[int]
    REMEDIATION_FIELD_NUMBER: _ClassVar[int]
    FILE_PATH_FIELD_NUMBER: _ClassVar[int]
    SCANNER_FIELD_NUMBER: _ClassVar[int]
    rule_id: str
    severity: str
    title: str
    description: str
    remediation: str
    file_path: str
    scanner: str
    def __init__(self, rule_id: _Optional[str] = ..., severity: _Optional[str] = ..., title: _Optional[str] = ..., description: _Optional[str] = ..., remediation: _Optional[str] = ..., file_path: _Optional[str] = ..., scanner: _Optional[str] = ...) -> None: ...

class ScanCandidateResponse(_message.Message):
    __slots__ = ("complete", "findings", "failure_code")
    COMPLETE_FIELD_NUMBER: _ClassVar[int]
    FINDINGS_FIELD_NUMBER: _ClassVar[int]
    FAILURE_CODE_FIELD_NUMBER: _ClassVar[int]
    complete: bool
    findings: _containers.RepeatedCompositeFieldContainer[CandidateFinding]
    failure_code: str
    def __init__(self, complete: _Optional[bool] = ..., findings: _Optional[_Iterable[_Union[CandidateFinding, _Mapping]]] = ..., failure_code: _Optional[str] = ...) -> None: ...
