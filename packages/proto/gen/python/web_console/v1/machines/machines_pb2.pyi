import datetime

from google.protobuf import timestamp_pb2 as _timestamp_pb2
from web_console.v1.shared import target_pb2 as _target_pb2
from google.protobuf.internal import containers as _containers
from google.protobuf.internal import enum_type_wrapper as _enum_type_wrapper
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Iterable as _Iterable, Mapping as _Mapping
from typing import ClassVar as _ClassVar, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class FleetState(int, metaclass=_enum_type_wrapper.EnumTypeWrapper):
    __slots__ = ()
    FLEET_STATE_UNSPECIFIED: _ClassVar[FleetState]
    FLEET_STATE_READY: _ClassVar[FleetState]
    FLEET_STATE_EMPTY: _ClassVar[FleetState]
    FLEET_STATE_UNENROLLED: _ClassVar[FleetState]
    FLEET_STATE_UNREACHABLE: _ClassVar[FleetState]
FLEET_STATE_UNSPECIFIED: FleetState
FLEET_STATE_READY: FleetState
FLEET_STATE_EMPTY: FleetState
FLEET_STATE_UNENROLLED: FleetState
FLEET_STATE_UNREACHABLE: FleetState

class Grant(_message.Message):
    __slots__ = ("summary", "effects", "app_count", "covers_all_apps", "scopes", "preset")
    SUMMARY_FIELD_NUMBER: _ClassVar[int]
    EFFECTS_FIELD_NUMBER: _ClassVar[int]
    APP_COUNT_FIELD_NUMBER: _ClassVar[int]
    COVERS_ALL_APPS_FIELD_NUMBER: _ClassVar[int]
    SCOPES_FIELD_NUMBER: _ClassVar[int]
    PRESET_FIELD_NUMBER: _ClassVar[int]
    summary: str
    effects: _containers.RepeatedScalarFieldContainer[str]
    app_count: int
    covers_all_apps: bool
    scopes: _containers.RepeatedScalarFieldContainer[str]
    preset: str
    def __init__(self, summary: _Optional[str] = ..., effects: _Optional[_Iterable[str]] = ..., app_count: _Optional[int] = ..., covers_all_apps: _Optional[bool] = ..., scopes: _Optional[_Iterable[str]] = ..., preset: _Optional[str] = ...) -> None: ...

class MachineDrift(_message.Message):
    __slots__ = ("kind", "name", "reason")
    KIND_FIELD_NUMBER: _ClassVar[int]
    NAME_FIELD_NUMBER: _ClassVar[int]
    REASON_FIELD_NUMBER: _ClassVar[int]
    kind: str
    name: str
    reason: str
    def __init__(self, kind: _Optional[str] = ..., name: _Optional[str] = ..., reason: _Optional[str] = ...) -> None: ...

class Machine(_message.Message):
    __slots__ = ("target", "grant", "heartbeat_age_seconds", "manageable", "drift")
    TARGET_FIELD_NUMBER: _ClassVar[int]
    GRANT_FIELD_NUMBER: _ClassVar[int]
    HEARTBEAT_AGE_SECONDS_FIELD_NUMBER: _ClassVar[int]
    MANAGEABLE_FIELD_NUMBER: _ClassVar[int]
    DRIFT_FIELD_NUMBER: _ClassVar[int]
    target: _target_pb2.Target
    grant: Grant
    heartbeat_age_seconds: int
    manageable: bool
    drift: _containers.RepeatedCompositeFieldContainer[MachineDrift]
    def __init__(self, target: _Optional[_Union[_target_pb2.Target, _Mapping]] = ..., grant: _Optional[_Union[Grant, _Mapping]] = ..., heartbeat_age_seconds: _Optional[int] = ..., manageable: _Optional[bool] = ..., drift: _Optional[_Iterable[_Union[MachineDrift, _Mapping]]] = ...) -> None: ...

class JoinRequest(_message.Message):
    __slots__ = ("id", "name", "os", "arch", "endpoint", "confirmation_words", "key_fingerprint", "requested_at", "requested_age_seconds")
    ID_FIELD_NUMBER: _ClassVar[int]
    NAME_FIELD_NUMBER: _ClassVar[int]
    OS_FIELD_NUMBER: _ClassVar[int]
    ARCH_FIELD_NUMBER: _ClassVar[int]
    ENDPOINT_FIELD_NUMBER: _ClassVar[int]
    CONFIRMATION_WORDS_FIELD_NUMBER: _ClassVar[int]
    KEY_FINGERPRINT_FIELD_NUMBER: _ClassVar[int]
    REQUESTED_AT_FIELD_NUMBER: _ClassVar[int]
    REQUESTED_AGE_SECONDS_FIELD_NUMBER: _ClassVar[int]
    id: str
    name: str
    os: str
    arch: str
    endpoint: str
    confirmation_words: _containers.RepeatedScalarFieldContainer[str]
    key_fingerprint: str
    requested_at: _timestamp_pb2.Timestamp
    requested_age_seconds: int
    def __init__(self, id: _Optional[str] = ..., name: _Optional[str] = ..., os: _Optional[str] = ..., arch: _Optional[str] = ..., endpoint: _Optional[str] = ..., confirmation_words: _Optional[_Iterable[str]] = ..., key_fingerprint: _Optional[str] = ..., requested_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., requested_age_seconds: _Optional[int] = ...) -> None: ...

class PermissionPreset(_message.Message):
    __slots__ = ("name", "title", "description", "scopes", "withholds", "summary", "effects", "app_count")
    NAME_FIELD_NUMBER: _ClassVar[int]
    TITLE_FIELD_NUMBER: _ClassVar[int]
    DESCRIPTION_FIELD_NUMBER: _ClassVar[int]
    SCOPES_FIELD_NUMBER: _ClassVar[int]
    WITHHOLDS_FIELD_NUMBER: _ClassVar[int]
    SUMMARY_FIELD_NUMBER: _ClassVar[int]
    EFFECTS_FIELD_NUMBER: _ClassVar[int]
    APP_COUNT_FIELD_NUMBER: _ClassVar[int]
    name: str
    title: str
    description: str
    scopes: _containers.RepeatedScalarFieldContainer[str]
    withholds: _containers.RepeatedScalarFieldContainer[str]
    summary: str
    effects: _containers.RepeatedScalarFieldContainer[str]
    app_count: int
    def __init__(self, name: _Optional[str] = ..., title: _Optional[str] = ..., description: _Optional[str] = ..., scopes: _Optional[_Iterable[str]] = ..., withholds: _Optional[_Iterable[str]] = ..., summary: _Optional[str] = ..., effects: _Optional[_Iterable[str]] = ..., app_count: _Optional[int] = ...) -> None: ...

class ControlPlane(_message.Message):
    __slots__ = ("reachable", "endpoint", "detail", "console_url")
    REACHABLE_FIELD_NUMBER: _ClassVar[int]
    ENDPOINT_FIELD_NUMBER: _ClassVar[int]
    DETAIL_FIELD_NUMBER: _ClassVar[int]
    CONSOLE_URL_FIELD_NUMBER: _ClassVar[int]
    reachable: bool
    endpoint: str
    detail: str
    console_url: str
    def __init__(self, reachable: _Optional[bool] = ..., endpoint: _Optional[str] = ..., detail: _Optional[str] = ..., console_url: _Optional[str] = ...) -> None: ...

class ListRequest(_message.Message):
    __slots__ = ()
    def __init__(self) -> None: ...

class ListResponse(_message.Message):
    __slots__ = ("state", "machines", "join_requests", "presets", "message", "recovery_action", "control_plane")
    STATE_FIELD_NUMBER: _ClassVar[int]
    MACHINES_FIELD_NUMBER: _ClassVar[int]
    JOIN_REQUESTS_FIELD_NUMBER: _ClassVar[int]
    PRESETS_FIELD_NUMBER: _ClassVar[int]
    MESSAGE_FIELD_NUMBER: _ClassVar[int]
    RECOVERY_ACTION_FIELD_NUMBER: _ClassVar[int]
    CONTROL_PLANE_FIELD_NUMBER: _ClassVar[int]
    state: FleetState
    machines: _containers.RepeatedCompositeFieldContainer[Machine]
    join_requests: _containers.RepeatedCompositeFieldContainer[JoinRequest]
    presets: _containers.RepeatedCompositeFieldContainer[PermissionPreset]
    message: str
    recovery_action: str
    control_plane: ControlPlane
    def __init__(self, state: _Optional[_Union[FleetState, str]] = ..., machines: _Optional[_Iterable[_Union[Machine, _Mapping]]] = ..., join_requests: _Optional[_Iterable[_Union[JoinRequest, _Mapping]]] = ..., presets: _Optional[_Iterable[_Union[PermissionPreset, _Mapping]]] = ..., message: _Optional[str] = ..., recovery_action: _Optional[str] = ..., control_plane: _Optional[_Union[ControlPlane, _Mapping]] = ...) -> None: ...

class IssueCodeRequest(_message.Message):
    __slots__ = ("label",)
    LABEL_FIELD_NUMBER: _ClassVar[int]
    label: str
    def __init__(self, label: _Optional[str] = ...) -> None: ...

class IssueCodeResponse(_message.Message):
    __slots__ = ("code", "expires_at", "expires_in_seconds")
    CODE_FIELD_NUMBER: _ClassVar[int]
    EXPIRES_AT_FIELD_NUMBER: _ClassVar[int]
    EXPIRES_IN_SECONDS_FIELD_NUMBER: _ClassVar[int]
    code: str
    expires_at: _timestamp_pb2.Timestamp
    expires_in_seconds: int
    def __init__(self, code: _Optional[str] = ..., expires_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., expires_in_seconds: _Optional[int] = ...) -> None: ...

class DecideRequest(_message.Message):
    __slots__ = ("request_id", "approve", "confirmation_words", "preset", "scopes")
    REQUEST_ID_FIELD_NUMBER: _ClassVar[int]
    APPROVE_FIELD_NUMBER: _ClassVar[int]
    CONFIRMATION_WORDS_FIELD_NUMBER: _ClassVar[int]
    PRESET_FIELD_NUMBER: _ClassVar[int]
    SCOPES_FIELD_NUMBER: _ClassVar[int]
    request_id: str
    approve: bool
    confirmation_words: _containers.RepeatedScalarFieldContainer[str]
    preset: str
    scopes: _containers.RepeatedScalarFieldContainer[str]
    def __init__(self, request_id: _Optional[str] = ..., approve: _Optional[bool] = ..., confirmation_words: _Optional[_Iterable[str]] = ..., preset: _Optional[str] = ..., scopes: _Optional[_Iterable[str]] = ...) -> None: ...

class DecideResponse(_message.Message):
    __slots__ = ("machine", "message")
    MACHINE_FIELD_NUMBER: _ClassVar[int]
    MESSAGE_FIELD_NUMBER: _ClassVar[int]
    machine: Machine
    message: str
    def __init__(self, machine: _Optional[_Union[Machine, _Mapping]] = ..., message: _Optional[str] = ...) -> None: ...

class SetGrantRequest(_message.Message):
    __slots__ = ("machine_id", "preset", "scopes")
    MACHINE_ID_FIELD_NUMBER: _ClassVar[int]
    PRESET_FIELD_NUMBER: _ClassVar[int]
    SCOPES_FIELD_NUMBER: _ClassVar[int]
    machine_id: str
    preset: str
    scopes: _containers.RepeatedScalarFieldContainer[str]
    def __init__(self, machine_id: _Optional[str] = ..., preset: _Optional[str] = ..., scopes: _Optional[_Iterable[str]] = ...) -> None: ...

class SetGrantResponse(_message.Message):
    __slots__ = ("machine",)
    MACHINE_FIELD_NUMBER: _ClassVar[int]
    machine: Machine
    def __init__(self, machine: _Optional[_Union[Machine, _Mapping]] = ...) -> None: ...

class ForgetRequest(_message.Message):
    __slots__ = ("machine_id",)
    MACHINE_ID_FIELD_NUMBER: _ClassVar[int]
    machine_id: str
    def __init__(self, machine_id: _Optional[str] = ...) -> None: ...

class ForgetResponse(_message.Message):
    __slots__ = ("forgotten_machine_id",)
    FORGOTTEN_MACHINE_ID_FIELD_NUMBER: _ClassVar[int]
    forgotten_machine_id: str
    def __init__(self, forgotten_machine_id: _Optional[str] = ...) -> None: ...
