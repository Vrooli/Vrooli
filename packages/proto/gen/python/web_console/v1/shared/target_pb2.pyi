import datetime

from google.protobuf import timestamp_pb2 as _timestamp_pb2
from google.protobuf.internal import containers as _containers
from google.protobuf.internal import enum_type_wrapper as _enum_type_wrapper
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Iterable as _Iterable, Mapping as _Mapping
from typing import ClassVar as _ClassVar, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class TargetState(int, metaclass=_enum_type_wrapper.EnumTypeWrapper):
    __slots__ = ()
    TARGET_STATE_UNSPECIFIED: _ClassVar[TargetState]
    TARGET_STATE_DISPATCHABLE: _ClassVar[TargetState]
    TARGET_STATE_OFFLINE: _ClassVar[TargetState]
    TARGET_STATE_NEEDS_UPDATE: _ClassVar[TargetState]
    TARGET_STATE_UNAVAILABLE: _ClassVar[TargetState]
TARGET_STATE_UNSPECIFIED: TargetState
TARGET_STATE_DISPATCHABLE: TargetState
TARGET_STATE_OFFLINE: TargetState
TARGET_STATE_NEEDS_UPDATE: TargetState
TARGET_STATE_UNAVAILABLE: TargetState

class ReadinessFact(_message.Message):
    __slots__ = ("key", "label", "passed", "detail", "state", "version", "recovery_action")
    KEY_FIELD_NUMBER: _ClassVar[int]
    LABEL_FIELD_NUMBER: _ClassVar[int]
    PASSED_FIELD_NUMBER: _ClassVar[int]
    DETAIL_FIELD_NUMBER: _ClassVar[int]
    STATE_FIELD_NUMBER: _ClassVar[int]
    VERSION_FIELD_NUMBER: _ClassVar[int]
    RECOVERY_ACTION_FIELD_NUMBER: _ClassVar[int]
    key: str
    label: str
    passed: bool
    detail: str
    state: str
    version: str
    recovery_action: str
    def __init__(self, key: _Optional[str] = ..., label: _Optional[str] = ..., passed: _Optional[bool] = ..., detail: _Optional[str] = ..., state: _Optional[str] = ..., version: _Optional[str] = ..., recovery_action: _Optional[str] = ...) -> None: ...

class Target(_message.Message):
    __slots__ = ("id", "kind", "label", "os", "arch", "node_id", "revision", "status", "online", "last_seen_at", "readiness", "dispatchable", "failure_rung", "state", "recovery_action", "survives_restart")
    ID_FIELD_NUMBER: _ClassVar[int]
    KIND_FIELD_NUMBER: _ClassVar[int]
    LABEL_FIELD_NUMBER: _ClassVar[int]
    OS_FIELD_NUMBER: _ClassVar[int]
    ARCH_FIELD_NUMBER: _ClassVar[int]
    NODE_ID_FIELD_NUMBER: _ClassVar[int]
    REVISION_FIELD_NUMBER: _ClassVar[int]
    STATUS_FIELD_NUMBER: _ClassVar[int]
    ONLINE_FIELD_NUMBER: _ClassVar[int]
    LAST_SEEN_AT_FIELD_NUMBER: _ClassVar[int]
    READINESS_FIELD_NUMBER: _ClassVar[int]
    DISPATCHABLE_FIELD_NUMBER: _ClassVar[int]
    FAILURE_RUNG_FIELD_NUMBER: _ClassVar[int]
    STATE_FIELD_NUMBER: _ClassVar[int]
    RECOVERY_ACTION_FIELD_NUMBER: _ClassVar[int]
    SURVIVES_RESTART_FIELD_NUMBER: _ClassVar[int]
    id: str
    kind: str
    label: str
    os: str
    arch: str
    node_id: str
    revision: str
    status: str
    online: bool
    last_seen_at: _timestamp_pb2.Timestamp
    readiness: _containers.RepeatedCompositeFieldContainer[ReadinessFact]
    dispatchable: bool
    failure_rung: str
    state: TargetState
    recovery_action: str
    survives_restart: bool
    def __init__(self, id: _Optional[str] = ..., kind: _Optional[str] = ..., label: _Optional[str] = ..., os: _Optional[str] = ..., arch: _Optional[str] = ..., node_id: _Optional[str] = ..., revision: _Optional[str] = ..., status: _Optional[str] = ..., online: _Optional[bool] = ..., last_seen_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., readiness: _Optional[_Iterable[_Union[ReadinessFact, _Mapping]]] = ..., dispatchable: _Optional[bool] = ..., failure_rung: _Optional[str] = ..., state: _Optional[_Union[TargetState, str]] = ..., recovery_action: _Optional[str] = ..., survives_restart: _Optional[bool] = ...) -> None: ...
