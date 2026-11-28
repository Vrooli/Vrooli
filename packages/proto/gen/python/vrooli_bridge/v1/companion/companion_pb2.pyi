import datetime

from google.protobuf import timestamp_pb2 as _timestamp_pb2
from google.protobuf.internal import enum_type_wrapper as _enum_type_wrapper
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Mapping as _Mapping
from typing import ClassVar as _ClassVar, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class CompanionState(int, metaclass=_enum_type_wrapper.EnumTypeWrapper):
    __slots__ = ()
    COMPANION_STATE_UNSPECIFIED: _ClassVar[CompanionState]
    COMPANION_STATE_ABSENT: _ClassVar[CompanionState]
    COMPANION_STATE_INSTALLING: _ClassVar[CompanionState]
    COMPANION_STATE_READY: _ClassVar[CompanionState]
    COMPANION_STATE_DEGRADED: _ClassVar[CompanionState]
    COMPANION_STATE_REVOKED: _ClassVar[CompanionState]
    COMPANION_STATE_REMOVED: _ClassVar[CompanionState]
    COMPANION_STATE_FAILED: _ClassVar[CompanionState]

class CompanionOperationKind(int, metaclass=_enum_type_wrapper.EnumTypeWrapper):
    __slots__ = ()
    COMPANION_OPERATION_KIND_UNSPECIFIED: _ClassVar[CompanionOperationKind]
    COMPANION_OPERATION_KIND_INSTALL: _ClassVar[CompanionOperationKind]
    COMPANION_OPERATION_KIND_INSPECT: _ClassVar[CompanionOperationKind]
    COMPANION_OPERATION_KIND_UPGRADE: _ClassVar[CompanionOperationKind]
    COMPANION_OPERATION_KIND_REVOKE: _ClassVar[CompanionOperationKind]
    COMPANION_OPERATION_KIND_REMOVE: _ClassVar[CompanionOperationKind]
COMPANION_STATE_UNSPECIFIED: CompanionState
COMPANION_STATE_ABSENT: CompanionState
COMPANION_STATE_INSTALLING: CompanionState
COMPANION_STATE_READY: CompanionState
COMPANION_STATE_DEGRADED: CompanionState
COMPANION_STATE_REVOKED: CompanionState
COMPANION_STATE_REMOVED: CompanionState
COMPANION_STATE_FAILED: CompanionState
COMPANION_OPERATION_KIND_UNSPECIFIED: CompanionOperationKind
COMPANION_OPERATION_KIND_INSTALL: CompanionOperationKind
COMPANION_OPERATION_KIND_INSPECT: CompanionOperationKind
COMPANION_OPERATION_KIND_UPGRADE: CompanionOperationKind
COMPANION_OPERATION_KIND_REVOKE: CompanionOperationKind
COMPANION_OPERATION_KIND_REMOVE: CompanionOperationKind

class InstallRequest(_message.Message):
    __slots__ = ("node_id", "version", "display_id", "artifact_source_ref", "artifact_name", "artifact_destination_path")
    NODE_ID_FIELD_NUMBER: _ClassVar[int]
    VERSION_FIELD_NUMBER: _ClassVar[int]
    DISPLAY_ID_FIELD_NUMBER: _ClassVar[int]
    ARTIFACT_SOURCE_REF_FIELD_NUMBER: _ClassVar[int]
    ARTIFACT_NAME_FIELD_NUMBER: _ClassVar[int]
    ARTIFACT_DESTINATION_PATH_FIELD_NUMBER: _ClassVar[int]
    node_id: str
    version: str
    display_id: str
    artifact_source_ref: str
    artifact_name: str
    artifact_destination_path: str
    def __init__(self, node_id: _Optional[str] = ..., version: _Optional[str] = ..., display_id: _Optional[str] = ..., artifact_source_ref: _Optional[str] = ..., artifact_name: _Optional[str] = ..., artifact_destination_path: _Optional[str] = ...) -> None: ...

class InspectRequest(_message.Message):
    __slots__ = ("node_id",)
    NODE_ID_FIELD_NUMBER: _ClassVar[int]
    node_id: str
    def __init__(self, node_id: _Optional[str] = ...) -> None: ...

class UpgradeRequest(_message.Message):
    __slots__ = ("node_id", "version", "artifact_source_ref", "artifact_name", "artifact_destination_path")
    NODE_ID_FIELD_NUMBER: _ClassVar[int]
    VERSION_FIELD_NUMBER: _ClassVar[int]
    ARTIFACT_SOURCE_REF_FIELD_NUMBER: _ClassVar[int]
    ARTIFACT_NAME_FIELD_NUMBER: _ClassVar[int]
    ARTIFACT_DESTINATION_PATH_FIELD_NUMBER: _ClassVar[int]
    node_id: str
    version: str
    artifact_source_ref: str
    artifact_name: str
    artifact_destination_path: str
    def __init__(self, node_id: _Optional[str] = ..., version: _Optional[str] = ..., artifact_source_ref: _Optional[str] = ..., artifact_name: _Optional[str] = ..., artifact_destination_path: _Optional[str] = ...) -> None: ...

class RevokeRequest(_message.Message):
    __slots__ = ("node_id", "reason")
    NODE_ID_FIELD_NUMBER: _ClassVar[int]
    REASON_FIELD_NUMBER: _ClassVar[int]
    node_id: str
    reason: str
    def __init__(self, node_id: _Optional[str] = ..., reason: _Optional[str] = ...) -> None: ...

class RemoveRequest(_message.Message):
    __slots__ = ("node_id",)
    NODE_ID_FIELD_NUMBER: _ClassVar[int]
    node_id: str
    def __init__(self, node_id: _Optional[str] = ...) -> None: ...

class CompanionOperation(_message.Message):
    __slots__ = ("operation_id", "node_id", "version", "state", "reason_code", "recovery", "user_launch_agent", "observed_at", "artifact_distribution_id")
    OPERATION_ID_FIELD_NUMBER: _ClassVar[int]
    NODE_ID_FIELD_NUMBER: _ClassVar[int]
    VERSION_FIELD_NUMBER: _ClassVar[int]
    STATE_FIELD_NUMBER: _ClassVar[int]
    REASON_CODE_FIELD_NUMBER: _ClassVar[int]
    RECOVERY_FIELD_NUMBER: _ClassVar[int]
    USER_LAUNCH_AGENT_FIELD_NUMBER: _ClassVar[int]
    OBSERVED_AT_FIELD_NUMBER: _ClassVar[int]
    ARTIFACT_DISTRIBUTION_ID_FIELD_NUMBER: _ClassVar[int]
    operation_id: str
    node_id: str
    version: str
    state: CompanionState
    reason_code: str
    recovery: str
    user_launch_agent: bool
    observed_at: _timestamp_pb2.Timestamp
    artifact_distribution_id: str
    def __init__(self, operation_id: _Optional[str] = ..., node_id: _Optional[str] = ..., version: _Optional[str] = ..., state: _Optional[_Union[CompanionState, str]] = ..., reason_code: _Optional[str] = ..., recovery: _Optional[str] = ..., user_launch_agent: _Optional[bool] = ..., observed_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., artifact_distribution_id: _Optional[str] = ...) -> None: ...

class CompanionCommand(_message.Message):
    __slots__ = ("operation_id", "node_id", "kind", "version", "display_id", "artifact_distribution_id")
    OPERATION_ID_FIELD_NUMBER: _ClassVar[int]
    NODE_ID_FIELD_NUMBER: _ClassVar[int]
    KIND_FIELD_NUMBER: _ClassVar[int]
    VERSION_FIELD_NUMBER: _ClassVar[int]
    DISPLAY_ID_FIELD_NUMBER: _ClassVar[int]
    ARTIFACT_DISTRIBUTION_ID_FIELD_NUMBER: _ClassVar[int]
    operation_id: str
    node_id: str
    kind: CompanionOperationKind
    version: str
    display_id: str
    artifact_distribution_id: str
    def __init__(self, operation_id: _Optional[str] = ..., node_id: _Optional[str] = ..., kind: _Optional[_Union[CompanionOperationKind, str]] = ..., version: _Optional[str] = ..., display_id: _Optional[str] = ..., artifact_distribution_id: _Optional[str] = ...) -> None: ...

class CompanionResponse(_message.Message):
    __slots__ = ("operation_id", "node_id", "version", "state", "reason_code", "recovery", "user_launch_agent", "observed_at")
    OPERATION_ID_FIELD_NUMBER: _ClassVar[int]
    NODE_ID_FIELD_NUMBER: _ClassVar[int]
    VERSION_FIELD_NUMBER: _ClassVar[int]
    STATE_FIELD_NUMBER: _ClassVar[int]
    REASON_CODE_FIELD_NUMBER: _ClassVar[int]
    RECOVERY_FIELD_NUMBER: _ClassVar[int]
    USER_LAUNCH_AGENT_FIELD_NUMBER: _ClassVar[int]
    OBSERVED_AT_FIELD_NUMBER: _ClassVar[int]
    operation_id: str
    node_id: str
    version: str
    state: CompanionState
    reason_code: str
    recovery: str
    user_launch_agent: bool
    observed_at: _timestamp_pb2.Timestamp
    def __init__(self, operation_id: _Optional[str] = ..., node_id: _Optional[str] = ..., version: _Optional[str] = ..., state: _Optional[_Union[CompanionState, str]] = ..., reason_code: _Optional[str] = ..., recovery: _Optional[str] = ..., user_launch_agent: _Optional[bool] = ..., observed_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ...) -> None: ...
