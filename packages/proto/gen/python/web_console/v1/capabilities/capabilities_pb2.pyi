from google.protobuf.internal import containers as _containers
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Iterable as _Iterable, Mapping as _Mapping
from typing import ClassVar as _ClassVar, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class CapabilityState(_message.Message):
    __slots__ = ("id", "name", "description", "dependency_kind", "dependency_slug", "features", "status", "message", "checked_at")
    ID_FIELD_NUMBER: _ClassVar[int]
    NAME_FIELD_NUMBER: _ClassVar[int]
    DESCRIPTION_FIELD_NUMBER: _ClassVar[int]
    DEPENDENCY_KIND_FIELD_NUMBER: _ClassVar[int]
    DEPENDENCY_SLUG_FIELD_NUMBER: _ClassVar[int]
    FEATURES_FIELD_NUMBER: _ClassVar[int]
    STATUS_FIELD_NUMBER: _ClassVar[int]
    MESSAGE_FIELD_NUMBER: _ClassVar[int]
    CHECKED_AT_FIELD_NUMBER: _ClassVar[int]
    id: str
    name: str
    description: str
    dependency_kind: str
    dependency_slug: str
    features: _containers.RepeatedScalarFieldContainer[str]
    status: str
    message: str
    checked_at: str
    def __init__(self, id: _Optional[str] = ..., name: _Optional[str] = ..., description: _Optional[str] = ..., dependency_kind: _Optional[str] = ..., dependency_slug: _Optional[str] = ..., features: _Optional[_Iterable[str]] = ..., status: _Optional[str] = ..., message: _Optional[str] = ..., checked_at: _Optional[str] = ...) -> None: ...

class BackendOption(_message.Message):
    __slots__ = ("id", "display_name", "description", "survives_restart", "available", "reason")
    ID_FIELD_NUMBER: _ClassVar[int]
    DISPLAY_NAME_FIELD_NUMBER: _ClassVar[int]
    DESCRIPTION_FIELD_NUMBER: _ClassVar[int]
    SURVIVES_RESTART_FIELD_NUMBER: _ClassVar[int]
    AVAILABLE_FIELD_NUMBER: _ClassVar[int]
    REASON_FIELD_NUMBER: _ClassVar[int]
    id: str
    display_name: str
    description: str
    survives_restart: bool
    available: bool
    reason: str
    def __init__(self, id: _Optional[str] = ..., display_name: _Optional[str] = ..., description: _Optional[str] = ..., survives_restart: _Optional[bool] = ..., available: _Optional[bool] = ..., reason: _Optional[str] = ...) -> None: ...

class GetRequest(_message.Message):
    __slots__ = ()
    def __init__(self) -> None: ...

class GetResponse(_message.Message):
    __slots__ = ("capabilities", "timestamp", "session_backends", "default_backend")
    CAPABILITIES_FIELD_NUMBER: _ClassVar[int]
    TIMESTAMP_FIELD_NUMBER: _ClassVar[int]
    SESSION_BACKENDS_FIELD_NUMBER: _ClassVar[int]
    DEFAULT_BACKEND_FIELD_NUMBER: _ClassVar[int]
    capabilities: _containers.RepeatedCompositeFieldContainer[CapabilityState]
    timestamp: str
    session_backends: _containers.RepeatedCompositeFieldContainer[BackendOption]
    default_backend: str
    def __init__(self, capabilities: _Optional[_Iterable[_Union[CapabilityState, _Mapping]]] = ..., timestamp: _Optional[str] = ..., session_backends: _Optional[_Iterable[_Union[BackendOption, _Mapping]]] = ..., default_backend: _Optional[str] = ...) -> None: ...

class LivenessRequest(_message.Message):
    __slots__ = ()
    def __init__(self) -> None: ...

class LivenessResponse(_message.Message):
    __slots__ = ("capabilities", "timestamp")
    CAPABILITIES_FIELD_NUMBER: _ClassVar[int]
    TIMESTAMP_FIELD_NUMBER: _ClassVar[int]
    capabilities: _containers.RepeatedCompositeFieldContainer[CapabilityState]
    timestamp: str
    def __init__(self, capabilities: _Optional[_Iterable[_Union[CapabilityState, _Mapping]]] = ..., timestamp: _Optional[str] = ...) -> None: ...
