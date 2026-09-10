from common.v1 import surface_pb2 as _surface_pb2
from device_control.v1.desktop import desktop_pb2 as _desktop_pb2
from google.protobuf.internal import enum_type_wrapper as _enum_type_wrapper
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Mapping as _Mapping
from typing import ClassVar as _ClassVar, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class DesktopCommandOperation(int, metaclass=_enum_type_wrapper.EnumTypeWrapper):
    __slots__ = ()
    DESKTOP_COMMAND_OPERATION_UNSPECIFIED: _ClassVar[DesktopCommandOperation]
    DESKTOP_COMMAND_OPERATION_OBSERVE: _ClassVar[DesktopCommandOperation]
    DESKTOP_COMMAND_OPERATION_ACT: _ClassVar[DesktopCommandOperation]
    DESKTOP_COMMAND_OPERATION_STOP: _ClassVar[DesktopCommandOperation]
DESKTOP_COMMAND_OPERATION_UNSPECIFIED: DesktopCommandOperation
DESKTOP_COMMAND_OPERATION_OBSERVE: DesktopCommandOperation
DESKTOP_COMMAND_OPERATION_ACT: DesktopCommandOperation
DESKTOP_COMMAND_OPERATION_STOP: DesktopCommandOperation

class Open(_message.Message):
    __slots__ = ("session_id", "node_id", "receive_window", "idle_timeout_seconds", "max_lifetime_seconds", "shell", "working_dir", "binding", "max_frame_bytes", "max_frames_per_second")
    SESSION_ID_FIELD_NUMBER: _ClassVar[int]
    NODE_ID_FIELD_NUMBER: _ClassVar[int]
    RECEIVE_WINDOW_FIELD_NUMBER: _ClassVar[int]
    IDLE_TIMEOUT_SECONDS_FIELD_NUMBER: _ClassVar[int]
    MAX_LIFETIME_SECONDS_FIELD_NUMBER: _ClassVar[int]
    SHELL_FIELD_NUMBER: _ClassVar[int]
    WORKING_DIR_FIELD_NUMBER: _ClassVar[int]
    BINDING_FIELD_NUMBER: _ClassVar[int]
    MAX_FRAME_BYTES_FIELD_NUMBER: _ClassVar[int]
    MAX_FRAMES_PER_SECOND_FIELD_NUMBER: _ClassVar[int]
    session_id: str
    node_id: str
    receive_window: int
    idle_timeout_seconds: int
    max_lifetime_seconds: int
    shell: str
    working_dir: str
    binding: Binding
    max_frame_bytes: int
    max_frames_per_second: int
    def __init__(self, session_id: _Optional[str] = ..., node_id: _Optional[str] = ..., receive_window: _Optional[int] = ..., idle_timeout_seconds: _Optional[int] = ..., max_lifetime_seconds: _Optional[int] = ..., shell: _Optional[str] = ..., working_dir: _Optional[str] = ..., binding: _Optional[_Union[Binding, _Mapping]] = ..., max_frame_bytes: _Optional[int] = ..., max_frames_per_second: _Optional[int] = ...) -> None: ...

class Binding(_message.Message):
    __slots__ = ("surface", "transport", "owner_id", "lease_id", "lease_epoch", "policy_revision")
    SURFACE_FIELD_NUMBER: _ClassVar[int]
    TRANSPORT_FIELD_NUMBER: _ClassVar[int]
    OWNER_ID_FIELD_NUMBER: _ClassVar[int]
    LEASE_ID_FIELD_NUMBER: _ClassVar[int]
    LEASE_EPOCH_FIELD_NUMBER: _ClassVar[int]
    POLICY_REVISION_FIELD_NUMBER: _ClassVar[int]
    surface: _surface_pb2.SurfaceRef
    transport: str
    owner_id: str
    lease_id: str
    lease_epoch: int
    policy_revision: str
    def __init__(self, surface: _Optional[_Union[_surface_pb2.SurfaceRef, _Mapping]] = ..., transport: _Optional[str] = ..., owner_id: _Optional[str] = ..., lease_id: _Optional[str] = ..., lease_epoch: _Optional[int] = ..., policy_revision: _Optional[str] = ...) -> None: ...

class Data(_message.Message):
    __slots__ = ("sequence", "data", "command_id")
    SEQUENCE_FIELD_NUMBER: _ClassVar[int]
    DATA_FIELD_NUMBER: _ClassVar[int]
    COMMAND_ID_FIELD_NUMBER: _ClassVar[int]
    sequence: int
    data: bytes
    command_id: str
    def __init__(self, sequence: _Optional[int] = ..., data: _Optional[bytes] = ..., command_id: _Optional[str] = ...) -> None: ...

class DesktopCommand(_message.Message):
    __slots__ = ("command_id", "operation", "process_id", "application_id", "application_revision", "geometry_revision", "action")
    COMMAND_ID_FIELD_NUMBER: _ClassVar[int]
    OPERATION_FIELD_NUMBER: _ClassVar[int]
    PROCESS_ID_FIELD_NUMBER: _ClassVar[int]
    APPLICATION_ID_FIELD_NUMBER: _ClassVar[int]
    APPLICATION_REVISION_FIELD_NUMBER: _ClassVar[int]
    GEOMETRY_REVISION_FIELD_NUMBER: _ClassVar[int]
    ACTION_FIELD_NUMBER: _ClassVar[int]
    command_id: str
    operation: DesktopCommandOperation
    process_id: int
    application_id: str
    application_revision: str
    geometry_revision: str
    action: _desktop_pb2.Action
    def __init__(self, command_id: _Optional[str] = ..., operation: _Optional[_Union[DesktopCommandOperation, str]] = ..., process_id: _Optional[int] = ..., application_id: _Optional[str] = ..., application_revision: _Optional[str] = ..., geometry_revision: _Optional[str] = ..., action: _Optional[_Union[_desktop_pb2.Action, _Mapping]] = ...) -> None: ...

class DesktopResult(_message.Message):
    __slots__ = ("command_id", "remote_command_id", "observation", "act", "error")
    COMMAND_ID_FIELD_NUMBER: _ClassVar[int]
    REMOTE_COMMAND_ID_FIELD_NUMBER: _ClassVar[int]
    OBSERVATION_FIELD_NUMBER: _ClassVar[int]
    ACT_FIELD_NUMBER: _ClassVar[int]
    ERROR_FIELD_NUMBER: _ClassVar[int]
    command_id: str
    remote_command_id: str
    observation: _desktop_pb2.ObserveResponse
    act: _desktop_pb2.ActResponse
    error: str
    def __init__(self, command_id: _Optional[str] = ..., remote_command_id: _Optional[str] = ..., observation: _Optional[_Union[_desktop_pb2.ObserveResponse, _Mapping]] = ..., act: _Optional[_Union[_desktop_pb2.ActResponse, _Mapping]] = ..., error: _Optional[str] = ...) -> None: ...

class Resize(_message.Message):
    __slots__ = ("columns", "rows")
    COLUMNS_FIELD_NUMBER: _ClassVar[int]
    ROWS_FIELD_NUMBER: _ClassVar[int]
    columns: int
    rows: int
    def __init__(self, columns: _Optional[int] = ..., rows: _Optional[int] = ...) -> None: ...

class Close(_message.Message):
    __slots__ = ("code", "reason")
    CODE_FIELD_NUMBER: _ClassVar[int]
    REASON_FIELD_NUMBER: _ClassVar[int]
    code: str
    reason: str
    def __init__(self, code: _Optional[str] = ..., reason: _Optional[str] = ...) -> None: ...

class Ack(_message.Message):
    __slots__ = ("accepted", "sequence", "window_available", "code", "reason", "command_id")
    ACCEPTED_FIELD_NUMBER: _ClassVar[int]
    SEQUENCE_FIELD_NUMBER: _ClassVar[int]
    WINDOW_AVAILABLE_FIELD_NUMBER: _ClassVar[int]
    CODE_FIELD_NUMBER: _ClassVar[int]
    REASON_FIELD_NUMBER: _ClassVar[int]
    COMMAND_ID_FIELD_NUMBER: _ClassVar[int]
    accepted: bool
    sequence: int
    window_available: int
    code: str
    reason: str
    command_id: str
    def __init__(self, accepted: _Optional[bool] = ..., sequence: _Optional[int] = ..., window_available: _Optional[int] = ..., code: _Optional[str] = ..., reason: _Optional[str] = ..., command_id: _Optional[str] = ...) -> None: ...

class WindowUpdate(_message.Message):
    __slots__ = ("window_available",)
    WINDOW_AVAILABLE_FIELD_NUMBER: _ClassVar[int]
    window_available: int
    def __init__(self, window_available: _Optional[int] = ...) -> None: ...

class Evidence(_message.Message):
    __slots__ = ("command_id", "remote_command_id", "evidence_id", "kind", "digest")
    COMMAND_ID_FIELD_NUMBER: _ClassVar[int]
    REMOTE_COMMAND_ID_FIELD_NUMBER: _ClassVar[int]
    EVIDENCE_ID_FIELD_NUMBER: _ClassVar[int]
    KIND_FIELD_NUMBER: _ClassVar[int]
    DIGEST_FIELD_NUMBER: _ClassVar[int]
    command_id: str
    remote_command_id: str
    evidence_id: str
    kind: str
    digest: str
    def __init__(self, command_id: _Optional[str] = ..., remote_command_id: _Optional[str] = ..., evidence_id: _Optional[str] = ..., kind: _Optional[str] = ..., digest: _Optional[str] = ...) -> None: ...

class Revoke(_message.Message):
    __slots__ = ("lease_id", "lease_epoch", "reason")
    LEASE_ID_FIELD_NUMBER: _ClassVar[int]
    LEASE_EPOCH_FIELD_NUMBER: _ClassVar[int]
    REASON_FIELD_NUMBER: _ClassVar[int]
    lease_id: str
    lease_epoch: int
    reason: str
    def __init__(self, lease_id: _Optional[str] = ..., lease_epoch: _Optional[int] = ..., reason: _Optional[str] = ...) -> None: ...

class Frame(_message.Message):
    __slots__ = ("open", "data", "resize", "close", "ack", "window_update", "evidence", "revoke")
    OPEN_FIELD_NUMBER: _ClassVar[int]
    DATA_FIELD_NUMBER: _ClassVar[int]
    RESIZE_FIELD_NUMBER: _ClassVar[int]
    CLOSE_FIELD_NUMBER: _ClassVar[int]
    ACK_FIELD_NUMBER: _ClassVar[int]
    WINDOW_UPDATE_FIELD_NUMBER: _ClassVar[int]
    EVIDENCE_FIELD_NUMBER: _ClassVar[int]
    REVOKE_FIELD_NUMBER: _ClassVar[int]
    open: Open
    data: Data
    resize: Resize
    close: Close
    ack: Ack
    window_update: WindowUpdate
    evidence: Evidence
    revoke: Revoke
    def __init__(self, open: _Optional[_Union[Open, _Mapping]] = ..., data: _Optional[_Union[Data, _Mapping]] = ..., resize: _Optional[_Union[Resize, _Mapping]] = ..., close: _Optional[_Union[Close, _Mapping]] = ..., ack: _Optional[_Union[Ack, _Mapping]] = ..., window_update: _Optional[_Union[WindowUpdate, _Mapping]] = ..., evidence: _Optional[_Union[Evidence, _Mapping]] = ..., revoke: _Optional[_Union[Revoke, _Mapping]] = ...) -> None: ...
