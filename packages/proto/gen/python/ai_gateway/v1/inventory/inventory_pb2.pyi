from google.protobuf.internal import containers as _containers
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Iterable as _Iterable, Mapping as _Mapping
from typing import ClassVar as _ClassVar, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class ProviderRole(_message.Message):
    __slots__ = ("provider", "role", "capabilities", "locality", "status", "policy_schema_version")
    PROVIDER_FIELD_NUMBER: _ClassVar[int]
    ROLE_FIELD_NUMBER: _ClassVar[int]
    CAPABILITIES_FIELD_NUMBER: _ClassVar[int]
    LOCALITY_FIELD_NUMBER: _ClassVar[int]
    STATUS_FIELD_NUMBER: _ClassVar[int]
    POLICY_SCHEMA_VERSION_FIELD_NUMBER: _ClassVar[int]
    provider: str
    role: str
    capabilities: _containers.RepeatedScalarFieldContainer[str]
    locality: str
    status: str
    policy_schema_version: str
    def __init__(self, provider: _Optional[str] = ..., role: _Optional[str] = ..., capabilities: _Optional[_Iterable[str]] = ..., locality: _Optional[str] = ..., status: _Optional[str] = ..., policy_schema_version: _Optional[str] = ...) -> None: ...

class ListProviderRolesRequest(_message.Message):
    __slots__ = ("provider",)
    PROVIDER_FIELD_NUMBER: _ClassVar[int]
    provider: str
    def __init__(self, provider: _Optional[str] = ...) -> None: ...

class ListProviderRolesResponse(_message.Message):
    __slots__ = ("roles", "warnings")
    ROLES_FIELD_NUMBER: _ClassVar[int]
    WARNINGS_FIELD_NUMBER: _ClassVar[int]
    roles: _containers.RepeatedCompositeFieldContainer[ProviderRole]
    warnings: _containers.RepeatedScalarFieldContainer[str]
    def __init__(self, roles: _Optional[_Iterable[_Union[ProviderRole, _Mapping]]] = ..., warnings: _Optional[_Iterable[str]] = ...) -> None: ...

class SmokeProviderRequest(_message.Message):
    __slots__ = ("provider",)
    PROVIDER_FIELD_NUMBER: _ClassVar[int]
    provider: str
    def __init__(self, provider: _Optional[str] = ...) -> None: ...

class SmokeProviderResponse(_message.Message):
    __slots__ = ("provider", "status", "code", "message", "exit_code", "warnings")
    PROVIDER_FIELD_NUMBER: _ClassVar[int]
    STATUS_FIELD_NUMBER: _ClassVar[int]
    CODE_FIELD_NUMBER: _ClassVar[int]
    MESSAGE_FIELD_NUMBER: _ClassVar[int]
    EXIT_CODE_FIELD_NUMBER: _ClassVar[int]
    WARNINGS_FIELD_NUMBER: _ClassVar[int]
    provider: str
    status: str
    code: str
    message: str
    exit_code: int
    warnings: _containers.RepeatedScalarFieldContainer[str]
    def __init__(self, provider: _Optional[str] = ..., status: _Optional[str] = ..., code: _Optional[str] = ..., message: _Optional[str] = ..., exit_code: _Optional[int] = ..., warnings: _Optional[_Iterable[str]] = ...) -> None: ...
