from google.protobuf.internal import containers as _containers
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Iterable as _Iterable, Mapping as _Mapping
from typing import ClassVar as _ClassVar, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class VolumeDevice(_message.Message):
    __slots__ = ("path", "filesystem", "uuid", "serial", "mountpoint", "total_bytes")
    PATH_FIELD_NUMBER: _ClassVar[int]
    FILESYSTEM_FIELD_NUMBER: _ClassVar[int]
    UUID_FIELD_NUMBER: _ClassVar[int]
    SERIAL_FIELD_NUMBER: _ClassVar[int]
    MOUNTPOINT_FIELD_NUMBER: _ClassVar[int]
    TOTAL_BYTES_FIELD_NUMBER: _ClassVar[int]
    path: str
    filesystem: str
    uuid: str
    serial: str
    mountpoint: str
    total_bytes: int
    def __init__(self, path: _Optional[str] = ..., filesystem: _Optional[str] = ..., uuid: _Optional[str] = ..., serial: _Optional[str] = ..., mountpoint: _Optional[str] = ..., total_bytes: _Optional[int] = ...) -> None: ...

class VolumeState(_message.Message):
    __slots__ = ("device", "mounted", "read_only", "dirty", "evidence", "observations")
    DEVICE_FIELD_NUMBER: _ClassVar[int]
    MOUNTED_FIELD_NUMBER: _ClassVar[int]
    READ_ONLY_FIELD_NUMBER: _ClassVar[int]
    DIRTY_FIELD_NUMBER: _ClassVar[int]
    EVIDENCE_FIELD_NUMBER: _ClassVar[int]
    OBSERVATIONS_FIELD_NUMBER: _ClassVar[int]
    device: VolumeDevice
    mounted: bool
    read_only: bool
    dirty: str
    evidence: str
    observations: _containers.RepeatedScalarFieldContainer[str]
    def __init__(self, device: _Optional[_Union[VolumeDevice, _Mapping]] = ..., mounted: _Optional[bool] = ..., read_only: _Optional[bool] = ..., dirty: _Optional[str] = ..., evidence: _Optional[str] = ..., observations: _Optional[_Iterable[str]] = ...) -> None: ...

class VolumeRemediationResponse(_message.Message):
    __slots__ = ("action", "status", "changed", "dry_run", "command", "backend", "detail", "state", "refusal_reason", "operator_command", "consistent")
    ACTION_FIELD_NUMBER: _ClassVar[int]
    STATUS_FIELD_NUMBER: _ClassVar[int]
    CHANGED_FIELD_NUMBER: _ClassVar[int]
    DRY_RUN_FIELD_NUMBER: _ClassVar[int]
    COMMAND_FIELD_NUMBER: _ClassVar[int]
    BACKEND_FIELD_NUMBER: _ClassVar[int]
    DETAIL_FIELD_NUMBER: _ClassVar[int]
    STATE_FIELD_NUMBER: _ClassVar[int]
    REFUSAL_REASON_FIELD_NUMBER: _ClassVar[int]
    OPERATOR_COMMAND_FIELD_NUMBER: _ClassVar[int]
    CONSISTENT_FIELD_NUMBER: _ClassVar[int]
    action: str
    status: str
    changed: bool
    dry_run: bool
    command: _containers.RepeatedScalarFieldContainer[str]
    backend: str
    detail: str
    state: VolumeState
    refusal_reason: str
    operator_command: str
    consistent: str
    def __init__(self, action: _Optional[str] = ..., status: _Optional[str] = ..., changed: _Optional[bool] = ..., dry_run: _Optional[bool] = ..., command: _Optional[_Iterable[str]] = ..., backend: _Optional[str] = ..., detail: _Optional[str] = ..., state: _Optional[_Union[VolumeState, _Mapping]] = ..., refusal_reason: _Optional[str] = ..., operator_command: _Optional[str] = ..., consistent: _Optional[str] = ...) -> None: ...
