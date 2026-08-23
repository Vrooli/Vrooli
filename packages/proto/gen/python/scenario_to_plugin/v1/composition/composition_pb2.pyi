from google.protobuf.internal import containers as _containers
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Iterable as _Iterable, Mapping as _Mapping
from typing import ClassVar as _ClassVar, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class ComposeRequest(_message.Message):
    __slots__ = ("scenario", "source_revision")
    SCENARIO_FIELD_NUMBER: _ClassVar[int]
    SOURCE_REVISION_FIELD_NUMBER: _ClassVar[int]
    scenario: str
    source_revision: str
    def __init__(self, scenario: _Optional[str] = ..., source_revision: _Optional[str] = ...) -> None: ...

class ComposeResponse(_message.Message):
    __slots__ = ("package", "findings")
    PACKAGE_FIELD_NUMBER: _ClassVar[int]
    FINDINGS_FIELD_NUMBER: _ClassVar[int]
    package: Package
    findings: _containers.RepeatedCompositeFieldContainer[Finding]
    def __init__(self, package: _Optional[_Union[Package, _Mapping]] = ..., findings: _Optional[_Iterable[_Union[Finding, _Mapping]]] = ...) -> None: ...

class GetPackageRequest(_message.Message):
    __slots__ = ("package_id",)
    PACKAGE_ID_FIELD_NUMBER: _ClassVar[int]
    package_id: str
    def __init__(self, package_id: _Optional[str] = ...) -> None: ...

class GetPackageResponse(_message.Message):
    __slots__ = ("package",)
    PACKAGE_FIELD_NUMBER: _ClassVar[int]
    package: Package
    def __init__(self, package: _Optional[_Union[Package, _Mapping]] = ...) -> None: ...

class Package(_message.Message):
    __slots__ = ("id", "scenario", "source_revision", "digest", "artifact_root", "state", "mcp_authentication")
    ID_FIELD_NUMBER: _ClassVar[int]
    SCENARIO_FIELD_NUMBER: _ClassVar[int]
    SOURCE_REVISION_FIELD_NUMBER: _ClassVar[int]
    DIGEST_FIELD_NUMBER: _ClassVar[int]
    ARTIFACT_ROOT_FIELD_NUMBER: _ClassVar[int]
    STATE_FIELD_NUMBER: _ClassVar[int]
    MCP_AUTHENTICATION_FIELD_NUMBER: _ClassVar[int]
    id: str
    scenario: str
    source_revision: str
    digest: str
    artifact_root: str
    state: str
    mcp_authentication: str
    def __init__(self, id: _Optional[str] = ..., scenario: _Optional[str] = ..., source_revision: _Optional[str] = ..., digest: _Optional[str] = ..., artifact_root: _Optional[str] = ..., state: _Optional[str] = ..., mcp_authentication: _Optional[str] = ...) -> None: ...

class Finding(_message.Message):
    __slots__ = ("code", "message", "path")
    CODE_FIELD_NUMBER: _ClassVar[int]
    MESSAGE_FIELD_NUMBER: _ClassVar[int]
    PATH_FIELD_NUMBER: _ClassVar[int]
    code: str
    message: str
    path: str
    def __init__(self, code: _Optional[str] = ..., message: _Optional[str] = ..., path: _Optional[str] = ...) -> None: ...
