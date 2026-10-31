from google.protobuf.internal import containers as _containers
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Iterable as _Iterable, Mapping as _Mapping
from typing import ClassVar as _ClassVar, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class AttestRequest(_message.Message):
    __slots__ = ("package_id", "dry_run")
    PACKAGE_ID_FIELD_NUMBER: _ClassVar[int]
    DRY_RUN_FIELD_NUMBER: _ClassVar[int]
    package_id: str
    dry_run: bool
    def __init__(self, package_id: _Optional[str] = ..., dry_run: _Optional[bool] = ...) -> None: ...

class AttestResponse(_message.Message):
    __slots__ = ("passed", "artifact_digest", "evidence", "findings")
    PASSED_FIELD_NUMBER: _ClassVar[int]
    ARTIFACT_DIGEST_FIELD_NUMBER: _ClassVar[int]
    EVIDENCE_FIELD_NUMBER: _ClassVar[int]
    FINDINGS_FIELD_NUMBER: _ClassVar[int]
    passed: bool
    artifact_digest: str
    evidence: _containers.RepeatedCompositeFieldContainer[Evidence]
    findings: _containers.RepeatedCompositeFieldContainer[Finding]
    def __init__(self, passed: _Optional[bool] = ..., artifact_digest: _Optional[str] = ..., evidence: _Optional[_Iterable[_Union[Evidence, _Mapping]]] = ..., findings: _Optional[_Iterable[_Union[Finding, _Mapping]]] = ...) -> None: ...

class Evidence(_message.Message):
    __slots__ = ("kind", "digest", "reference")
    KIND_FIELD_NUMBER: _ClassVar[int]
    DIGEST_FIELD_NUMBER: _ClassVar[int]
    REFERENCE_FIELD_NUMBER: _ClassVar[int]
    kind: str
    digest: str
    reference: str
    def __init__(self, kind: _Optional[str] = ..., digest: _Optional[str] = ..., reference: _Optional[str] = ...) -> None: ...

class Finding(_message.Message):
    __slots__ = ("code", "message", "path")
    CODE_FIELD_NUMBER: _ClassVar[int]
    MESSAGE_FIELD_NUMBER: _ClassVar[int]
    PATH_FIELD_NUMBER: _ClassVar[int]
    code: str
    message: str
    path: str
    def __init__(self, code: _Optional[str] = ..., message: _Optional[str] = ..., path: _Optional[str] = ...) -> None: ...
