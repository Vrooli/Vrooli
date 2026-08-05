import datetime

from google.protobuf import timestamp_pb2 as _timestamp_pb2
from google.protobuf.internal import containers as _containers
from google.protobuf.internal import enum_type_wrapper as _enum_type_wrapper
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Iterable as _Iterable, Mapping as _Mapping
from typing import ClassVar as _ClassVar, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class DeviceKind(int, metaclass=_enum_type_wrapper.EnumTypeWrapper):
    __slots__ = ()
    DEVICE_KIND_UNSPECIFIED: _ClassVar[DeviceKind]
    DEVICE_KIND_HOST: _ClassVar[DeviceKind]
    DEVICE_KIND_EMULATOR: _ClassVar[DeviceKind]
    DEVICE_KIND_PHYSICAL: _ClassVar[DeviceKind]

class Disposition(int, metaclass=_enum_type_wrapper.EnumTypeWrapper):
    __slots__ = ()
    DISPOSITION_UNSPECIFIED: _ClassVar[Disposition]
    DISPOSITION_PENDING: _ClassVar[Disposition]
    DISPOSITION_PASSED: _ClassVar[Disposition]
    DISPOSITION_FAILED: _ClassVar[Disposition]
    DISPOSITION_SKIPPED: _ClassVar[Disposition]
DEVICE_KIND_UNSPECIFIED: DeviceKind
DEVICE_KIND_HOST: DeviceKind
DEVICE_KIND_EMULATOR: DeviceKind
DEVICE_KIND_PHYSICAL: DeviceKind
DISPOSITION_UNSPECIFIED: Disposition
DISPOSITION_PENDING: Disposition
DISPOSITION_PASSED: Disposition
DISPOSITION_FAILED: Disposition
DISPOSITION_SKIPPED: Disposition

class EvidenceTarget(_message.Message):
    __slots__ = ("ramp", "platform", "os", "device_kind", "bridge_node_id", "bridge_job_id")
    RAMP_FIELD_NUMBER: _ClassVar[int]
    PLATFORM_FIELD_NUMBER: _ClassVar[int]
    OS_FIELD_NUMBER: _ClassVar[int]
    DEVICE_KIND_FIELD_NUMBER: _ClassVar[int]
    BRIDGE_NODE_ID_FIELD_NUMBER: _ClassVar[int]
    BRIDGE_JOB_ID_FIELD_NUMBER: _ClassVar[int]
    ramp: str
    platform: str
    os: str
    device_kind: DeviceKind
    bridge_node_id: str
    bridge_job_id: str
    def __init__(self, ramp: _Optional[str] = ..., platform: _Optional[str] = ..., os: _Optional[str] = ..., device_kind: _Optional[_Union[DeviceKind, str]] = ..., bridge_node_id: _Optional[str] = ..., bridge_job_id: _Optional[str] = ...) -> None: ...

class EvidenceRef(_message.Message):
    __slots__ = ("producer", "artifact_id", "kind", "checksum", "size_bytes", "created_at")
    PRODUCER_FIELD_NUMBER: _ClassVar[int]
    ARTIFACT_ID_FIELD_NUMBER: _ClassVar[int]
    KIND_FIELD_NUMBER: _ClassVar[int]
    CHECKSUM_FIELD_NUMBER: _ClassVar[int]
    SIZE_BYTES_FIELD_NUMBER: _ClassVar[int]
    CREATED_AT_FIELD_NUMBER: _ClassVar[int]
    producer: str
    artifact_id: str
    kind: str
    checksum: str
    size_bytes: int
    created_at: _timestamp_pb2.Timestamp
    def __init__(self, producer: _Optional[str] = ..., artifact_id: _Optional[str] = ..., kind: _Optional[str] = ..., checksum: _Optional[str] = ..., size_bytes: _Optional[int] = ..., created_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ...) -> None: ...

class TargetVerdict(_message.Message):
    __slots__ = ("target", "disposition", "refs", "run_id", "detail")
    TARGET_FIELD_NUMBER: _ClassVar[int]
    DISPOSITION_FIELD_NUMBER: _ClassVar[int]
    REFS_FIELD_NUMBER: _ClassVar[int]
    RUN_ID_FIELD_NUMBER: _ClassVar[int]
    DETAIL_FIELD_NUMBER: _ClassVar[int]
    target: EvidenceTarget
    disposition: Disposition
    refs: _containers.RepeatedCompositeFieldContainer[EvidenceRef]
    run_id: str
    detail: str
    def __init__(self, target: _Optional[_Union[EvidenceTarget, _Mapping]] = ..., disposition: _Optional[_Union[Disposition, str]] = ..., refs: _Optional[_Iterable[_Union[EvidenceRef, _Mapping]]] = ..., run_id: _Optional[str] = ..., detail: _Optional[str] = ...) -> None: ...
