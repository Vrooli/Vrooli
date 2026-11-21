import datetime

from buf.validate import validate_pb2 as _validate_pb2
from google.protobuf import timestamp_pb2 as _timestamp_pb2
from google.protobuf.internal import containers as _containers
from google.protobuf.internal import enum_type_wrapper as _enum_type_wrapper
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Iterable as _Iterable, Mapping as _Mapping
from typing import ClassVar as _ClassVar, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class RestoreMode(int, metaclass=_enum_type_wrapper.EnumTypeWrapper):
    __slots__ = ()
    RESTORE_MODE_UNSPECIFIED: _ClassVar[RestoreMode]
    RESTORE_MODE_RESTORE: _ClassVar[RestoreMode]
    RESTORE_MODE_VERIFY: _ClassVar[RestoreMode]

class RestoreStatus(int, metaclass=_enum_type_wrapper.EnumTypeWrapper):
    __slots__ = ()
    RESTORE_STATUS_UNSPECIFIED: _ClassVar[RestoreStatus]
    RESTORE_STATUS_REQUESTED: _ClassVar[RestoreStatus]
    RESTORE_STATUS_RESTORING: _ClassVar[RestoreStatus]
    RESTORE_STATUS_VERIFYING: _ClassVar[RestoreStatus]
    RESTORE_STATUS_VERIFIED: _ClassVar[RestoreStatus]
    RESTORE_STATUS_RESTORED: _ClassVar[RestoreStatus]
    RESTORE_STATUS_FAILED: _ClassVar[RestoreStatus]
RESTORE_MODE_UNSPECIFIED: RestoreMode
RESTORE_MODE_RESTORE: RestoreMode
RESTORE_MODE_VERIFY: RestoreMode
RESTORE_STATUS_UNSPECIFIED: RestoreStatus
RESTORE_STATUS_REQUESTED: RestoreStatus
RESTORE_STATUS_RESTORING: RestoreStatus
RESTORE_STATUS_VERIFYING: RestoreStatus
RESTORE_STATUS_VERIFIED: RestoreStatus
RESTORE_STATUS_RESTORED: RestoreStatus
RESTORE_STATUS_FAILED: RestoreStatus

class Restore(_message.Message):
    __slots__ = ("id", "target_id", "destination_id", "snapshot_id", "mode", "status", "location", "checksum", "last_verified_at", "requested_at", "finished_at", "error")
    ID_FIELD_NUMBER: _ClassVar[int]
    TARGET_ID_FIELD_NUMBER: _ClassVar[int]
    DESTINATION_ID_FIELD_NUMBER: _ClassVar[int]
    SNAPSHOT_ID_FIELD_NUMBER: _ClassVar[int]
    MODE_FIELD_NUMBER: _ClassVar[int]
    STATUS_FIELD_NUMBER: _ClassVar[int]
    LOCATION_FIELD_NUMBER: _ClassVar[int]
    CHECKSUM_FIELD_NUMBER: _ClassVar[int]
    LAST_VERIFIED_AT_FIELD_NUMBER: _ClassVar[int]
    REQUESTED_AT_FIELD_NUMBER: _ClassVar[int]
    FINISHED_AT_FIELD_NUMBER: _ClassVar[int]
    ERROR_FIELD_NUMBER: _ClassVar[int]
    id: str
    target_id: str
    destination_id: str
    snapshot_id: str
    mode: RestoreMode
    status: RestoreStatus
    location: str
    checksum: str
    last_verified_at: _timestamp_pb2.Timestamp
    requested_at: _timestamp_pb2.Timestamp
    finished_at: _timestamp_pb2.Timestamp
    error: str
    def __init__(self, id: _Optional[str] = ..., target_id: _Optional[str] = ..., destination_id: _Optional[str] = ..., snapshot_id: _Optional[str] = ..., mode: _Optional[_Union[RestoreMode, str]] = ..., status: _Optional[_Union[RestoreStatus, str]] = ..., location: _Optional[str] = ..., checksum: _Optional[str] = ..., last_verified_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., requested_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., finished_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., error: _Optional[str] = ...) -> None: ...

class RestoreTargetRequest(_message.Message):
    __slots__ = ("target_id", "destination_id", "snapshot_id", "location")
    TARGET_ID_FIELD_NUMBER: _ClassVar[int]
    DESTINATION_ID_FIELD_NUMBER: _ClassVar[int]
    SNAPSHOT_ID_FIELD_NUMBER: _ClassVar[int]
    LOCATION_FIELD_NUMBER: _ClassVar[int]
    target_id: str
    destination_id: str
    snapshot_id: str
    location: str
    def __init__(self, target_id: _Optional[str] = ..., destination_id: _Optional[str] = ..., snapshot_id: _Optional[str] = ..., location: _Optional[str] = ...) -> None: ...

class RestoreTargetResponse(_message.Message):
    __slots__ = ("restore",)
    RESTORE_FIELD_NUMBER: _ClassVar[int]
    restore: Restore
    def __init__(self, restore: _Optional[_Union[Restore, _Mapping]] = ...) -> None: ...

class VerifyTargetRequest(_message.Message):
    __slots__ = ("target_id", "destination_id", "snapshot_id")
    TARGET_ID_FIELD_NUMBER: _ClassVar[int]
    DESTINATION_ID_FIELD_NUMBER: _ClassVar[int]
    SNAPSHOT_ID_FIELD_NUMBER: _ClassVar[int]
    target_id: str
    destination_id: str
    snapshot_id: str
    def __init__(self, target_id: _Optional[str] = ..., destination_id: _Optional[str] = ..., snapshot_id: _Optional[str] = ...) -> None: ...

class VerifyTargetResponse(_message.Message):
    __slots__ = ("restore",)
    RESTORE_FIELD_NUMBER: _ClassVar[int]
    restore: Restore
    def __init__(self, restore: _Optional[_Union[Restore, _Mapping]] = ...) -> None: ...

class GetRestoreRequest(_message.Message):
    __slots__ = ("id",)
    ID_FIELD_NUMBER: _ClassVar[int]
    id: str
    def __init__(self, id: _Optional[str] = ...) -> None: ...

class GetRestoreResponse(_message.Message):
    __slots__ = ("restore",)
    RESTORE_FIELD_NUMBER: _ClassVar[int]
    restore: Restore
    def __init__(self, restore: _Optional[_Union[Restore, _Mapping]] = ...) -> None: ...

class ListRestoresRequest(_message.Message):
    __slots__ = ("target_id", "page_size", "page_token")
    TARGET_ID_FIELD_NUMBER: _ClassVar[int]
    PAGE_SIZE_FIELD_NUMBER: _ClassVar[int]
    PAGE_TOKEN_FIELD_NUMBER: _ClassVar[int]
    target_id: str
    page_size: int
    page_token: str
    def __init__(self, target_id: _Optional[str] = ..., page_size: _Optional[int] = ..., page_token: _Optional[str] = ...) -> None: ...

class ListRestoresResponse(_message.Message):
    __slots__ = ("restores", "next_page_token")
    RESTORES_FIELD_NUMBER: _ClassVar[int]
    NEXT_PAGE_TOKEN_FIELD_NUMBER: _ClassVar[int]
    restores: _containers.RepeatedCompositeFieldContainer[Restore]
    next_page_token: str
    def __init__(self, restores: _Optional[_Iterable[_Union[Restore, _Mapping]]] = ..., next_page_token: _Optional[str] = ...) -> None: ...
