from google.protobuf.internal import containers as _containers
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Iterable as _Iterable, Mapping as _Mapping
from typing import ClassVar as _ClassVar, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class Capability(_message.Message):
    __slots__ = ("adapter", "action", "supported", "requires_admin", "rollback_supported", "reason")
    ADAPTER_FIELD_NUMBER: _ClassVar[int]
    ACTION_FIELD_NUMBER: _ClassVar[int]
    SUPPORTED_FIELD_NUMBER: _ClassVar[int]
    REQUIRES_ADMIN_FIELD_NUMBER: _ClassVar[int]
    ROLLBACK_SUPPORTED_FIELD_NUMBER: _ClassVar[int]
    REASON_FIELD_NUMBER: _ClassVar[int]
    adapter: str
    action: str
    supported: bool
    requires_admin: bool
    rollback_supported: bool
    reason: str
    def __init__(self, adapter: _Optional[str] = ..., action: _Optional[str] = ..., supported: _Optional[bool] = ..., requires_admin: _Optional[bool] = ..., rollback_supported: _Optional[bool] = ..., reason: _Optional[str] = ...) -> None: ...

class PlatformSummary(_message.Message):
    __slots__ = ("os", "arch", "profile", "notes")
    OS_FIELD_NUMBER: _ClassVar[int]
    ARCH_FIELD_NUMBER: _ClassVar[int]
    PROFILE_FIELD_NUMBER: _ClassVar[int]
    NOTES_FIELD_NUMBER: _ClassVar[int]
    os: str
    arch: str
    profile: str
    notes: _containers.RepeatedScalarFieldContainer[str]
    def __init__(self, os: _Optional[str] = ..., arch: _Optional[str] = ..., profile: _Optional[str] = ..., notes: _Optional[_Iterable[str]] = ...) -> None: ...

class ListCapabilitiesRequest(_message.Message):
    __slots__ = ()
    def __init__(self) -> None: ...

class ListCapabilitiesResponse(_message.Message):
    __slots__ = ("capabilities",)
    CAPABILITIES_FIELD_NUMBER: _ClassVar[int]
    capabilities: _containers.RepeatedCompositeFieldContainer[Capability]
    def __init__(self, capabilities: _Optional[_Iterable[_Union[Capability, _Mapping]]] = ...) -> None: ...

class ExplainUnsupportedActionRequest(_message.Message):
    __slots__ = ("action",)
    ACTION_FIELD_NUMBER: _ClassVar[int]
    action: str
    def __init__(self, action: _Optional[str] = ...) -> None: ...

class ExplainUnsupportedActionResponse(_message.Message):
    __slots__ = ("capability", "manual_steps")
    CAPABILITY_FIELD_NUMBER: _ClassVar[int]
    MANUAL_STEPS_FIELD_NUMBER: _ClassVar[int]
    capability: Capability
    manual_steps: _containers.RepeatedScalarFieldContainer[str]
    def __init__(self, capability: _Optional[_Union[Capability, _Mapping]] = ..., manual_steps: _Optional[_Iterable[str]] = ...) -> None: ...

class GetPlatformSummaryRequest(_message.Message):
    __slots__ = ()
    def __init__(self) -> None: ...

class GetPlatformSummaryResponse(_message.Message):
    __slots__ = ("summary",)
    SUMMARY_FIELD_NUMBER: _ClassVar[int]
    summary: PlatformSummary
    def __init__(self, summary: _Optional[_Union[PlatformSummary, _Mapping]] = ...) -> None: ...
