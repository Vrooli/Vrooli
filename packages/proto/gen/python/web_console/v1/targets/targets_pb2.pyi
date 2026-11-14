from web_console.v1.shared import target_pb2 as _target_pb2
from google.protobuf.internal import containers as _containers
from google.protobuf.internal import enum_type_wrapper as _enum_type_wrapper
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Iterable as _Iterable, Mapping as _Mapping
from typing import ClassVar as _ClassVar, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class CatalogState(int, metaclass=_enum_type_wrapper.EnumTypeWrapper):
    __slots__ = ()
    CATALOG_STATE_UNSPECIFIED: _ClassVar[CatalogState]
    CATALOG_STATE_READY: _ClassVar[CatalogState]
    CATALOG_STATE_CONFIGURED_EMPTY: _ClassVar[CatalogState]
    CATALOG_STATE_UNCONFIGURED: _ClassVar[CatalogState]
    CATALOG_STATE_REGISTRY_ERROR: _ClassVar[CatalogState]
CATALOG_STATE_UNSPECIFIED: CatalogState
CATALOG_STATE_READY: CatalogState
CATALOG_STATE_CONFIGURED_EMPTY: CatalogState
CATALOG_STATE_UNCONFIGURED: CatalogState
CATALOG_STATE_REGISTRY_ERROR: CatalogState

class ListRequest(_message.Message):
    __slots__ = ()
    def __init__(self) -> None: ...

class ListResponse(_message.Message):
    __slots__ = ("state", "targets", "message", "recovery_action")
    STATE_FIELD_NUMBER: _ClassVar[int]
    TARGETS_FIELD_NUMBER: _ClassVar[int]
    MESSAGE_FIELD_NUMBER: _ClassVar[int]
    RECOVERY_ACTION_FIELD_NUMBER: _ClassVar[int]
    state: CatalogState
    targets: _containers.RepeatedCompositeFieldContainer[_target_pb2.Target]
    message: str
    recovery_action: str
    def __init__(self, state: _Optional[_Union[CatalogState, str]] = ..., targets: _Optional[_Iterable[_Union[_target_pb2.Target, _Mapping]]] = ..., message: _Optional[str] = ..., recovery_action: _Optional[str] = ...) -> None: ...

class GetRequest(_message.Message):
    __slots__ = ("id",)
    ID_FIELD_NUMBER: _ClassVar[int]
    id: str
    def __init__(self, id: _Optional[str] = ...) -> None: ...

class GetResponse(_message.Message):
    __slots__ = ("target",)
    TARGET_FIELD_NUMBER: _ClassVar[int]
    target: _target_pb2.Target
    def __init__(self, target: _Optional[_Union[_target_pb2.Target, _Mapping]] = ...) -> None: ...

class DoctorRequest(_message.Message):
    __slots__ = ("id",)
    ID_FIELD_NUMBER: _ClassVar[int]
    id: str
    def __init__(self, id: _Optional[str] = ...) -> None: ...

class DoctorResponse(_message.Message):
    __slots__ = ("target", "summary")
    TARGET_FIELD_NUMBER: _ClassVar[int]
    SUMMARY_FIELD_NUMBER: _ClassVar[int]
    target: _target_pb2.Target
    summary: str
    def __init__(self, target: _Optional[_Union[_target_pb2.Target, _Mapping]] = ..., summary: _Optional[str] = ...) -> None: ...
