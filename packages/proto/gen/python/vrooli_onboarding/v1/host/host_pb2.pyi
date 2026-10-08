from buf.validate import validate_pb2 as _validate_pb2
from google.protobuf import struct_pb2 as _struct_pb2
from google.protobuf.internal import containers as _containers
from google.protobuf.internal import enum_type_wrapper as _enum_type_wrapper
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Iterable as _Iterable, Mapping as _Mapping
from typing import ClassVar as _ClassVar, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class SafeguardDisposition(int, metaclass=_enum_type_wrapper.EnumTypeWrapper):
    __slots__ = ()
    SAFEGUARD_DISPOSITION_UNSPECIFIED: _ClassVar[SafeguardDisposition]
    SAFEGUARD_DISPOSITION_READY: _ClassVar[SafeguardDisposition]
    SAFEGUARD_DISPOSITION_MISSING: _ClassVar[SafeguardDisposition]
    SAFEGUARD_DISPOSITION_PERMISSION_DENIED: _ClassVar[SafeguardDisposition]
    SAFEGUARD_DISPOSITION_CONTENT_MISMATCH: _ClassVar[SafeguardDisposition]
    SAFEGUARD_DISPOSITION_DEFERRED: _ClassVar[SafeguardDisposition]
    SAFEGUARD_DISPOSITION_UNSUPPORTED: _ClassVar[SafeguardDisposition]
    SAFEGUARD_DISPOSITION_NOT_APPLICABLE: _ClassVar[SafeguardDisposition]
SAFEGUARD_DISPOSITION_UNSPECIFIED: SafeguardDisposition
SAFEGUARD_DISPOSITION_READY: SafeguardDisposition
SAFEGUARD_DISPOSITION_MISSING: SafeguardDisposition
SAFEGUARD_DISPOSITION_PERMISSION_DENIED: SafeguardDisposition
SAFEGUARD_DISPOSITION_CONTENT_MISMATCH: SafeguardDisposition
SAFEGUARD_DISPOSITION_DEFERRED: SafeguardDisposition
SAFEGUARD_DISPOSITION_UNSUPPORTED: SafeguardDisposition
SAFEGUARD_DISPOSITION_NOT_APPLICABLE: SafeguardDisposition

class ListHostRequirementsRequest(_message.Message):
    __slots__ = ("target",)
    TARGET_FIELD_NUMBER: _ClassVar[int]
    target: str
    def __init__(self, target: _Optional[str] = ...) -> None: ...

class GetHostFactsRequest(_message.Message):
    __slots__ = ("target",)
    TARGET_FIELD_NUMBER: _ClassVar[int]
    target: str
    def __init__(self, target: _Optional[str] = ...) -> None: ...

class ListTargetsRequest(_message.Message):
    __slots__ = ()
    def __init__(self) -> None: ...

class HostRequirement(_message.Message):
    __slots__ = ("name", "required", "reason", "notes", "description", "risk", "privilege", "bundling", "platforms", "commands", "config_schema", "config", "status", "disposition", "detail", "remediation")
    NAME_FIELD_NUMBER: _ClassVar[int]
    REQUIRED_FIELD_NUMBER: _ClassVar[int]
    REASON_FIELD_NUMBER: _ClassVar[int]
    NOTES_FIELD_NUMBER: _ClassVar[int]
    DESCRIPTION_FIELD_NUMBER: _ClassVar[int]
    RISK_FIELD_NUMBER: _ClassVar[int]
    PRIVILEGE_FIELD_NUMBER: _ClassVar[int]
    BUNDLING_FIELD_NUMBER: _ClassVar[int]
    PLATFORMS_FIELD_NUMBER: _ClassVar[int]
    COMMANDS_FIELD_NUMBER: _ClassVar[int]
    CONFIG_SCHEMA_FIELD_NUMBER: _ClassVar[int]
    CONFIG_FIELD_NUMBER: _ClassVar[int]
    STATUS_FIELD_NUMBER: _ClassVar[int]
    DISPOSITION_FIELD_NUMBER: _ClassVar[int]
    DETAIL_FIELD_NUMBER: _ClassVar[int]
    REMEDIATION_FIELD_NUMBER: _ClassVar[int]
    name: str
    required: bool
    reason: str
    notes: str
    description: str
    risk: str
    privilege: str
    bundling: str
    platforms: _containers.RepeatedScalarFieldContainer[str]
    commands: _containers.RepeatedScalarFieldContainer[str]
    config_schema: _struct_pb2.Struct
    config: _struct_pb2.Struct
    status: str
    disposition: SafeguardDisposition
    detail: str
    remediation: str
    def __init__(self, name: _Optional[str] = ..., required: _Optional[bool] = ..., reason: _Optional[str] = ..., notes: _Optional[str] = ..., description: _Optional[str] = ..., risk: _Optional[str] = ..., privilege: _Optional[str] = ..., bundling: _Optional[str] = ..., platforms: _Optional[_Iterable[str]] = ..., commands: _Optional[_Iterable[str]] = ..., config_schema: _Optional[_Union[_struct_pb2.Struct, _Mapping]] = ..., config: _Optional[_Union[_struct_pb2.Struct, _Mapping]] = ..., status: _Optional[str] = ..., disposition: _Optional[_Union[SafeguardDisposition, str]] = ..., detail: _Optional[str] = ..., remediation: _Optional[str] = ...) -> None: ...

class ListHostRequirementsResponse(_message.Message):
    __slots__ = ("tools", "safeguards")
    TOOLS_FIELD_NUMBER: _ClassVar[int]
    SAFEGUARDS_FIELD_NUMBER: _ClassVar[int]
    tools: _containers.RepeatedCompositeFieldContainer[HostRequirement]
    safeguards: _containers.RepeatedCompositeFieldContainer[HostRequirement]
    def __init__(self, tools: _Optional[_Iterable[_Union[HostRequirement, _Mapping]]] = ..., safeguards: _Optional[_Iterable[_Union[HostRequirement, _Mapping]]] = ...) -> None: ...

class GetHostFactsResponse(_message.Message):
    __slots__ = ("available", "reason", "cpu_count", "memory_total_bytes", "memory_available_bytes", "disk_free_bytes", "gpus", "platform")
    AVAILABLE_FIELD_NUMBER: _ClassVar[int]
    REASON_FIELD_NUMBER: _ClassVar[int]
    CPU_COUNT_FIELD_NUMBER: _ClassVar[int]
    MEMORY_TOTAL_BYTES_FIELD_NUMBER: _ClassVar[int]
    MEMORY_AVAILABLE_BYTES_FIELD_NUMBER: _ClassVar[int]
    DISK_FREE_BYTES_FIELD_NUMBER: _ClassVar[int]
    GPUS_FIELD_NUMBER: _ClassVar[int]
    PLATFORM_FIELD_NUMBER: _ClassVar[int]
    available: bool
    reason: str
    cpu_count: int
    memory_total_bytes: int
    memory_available_bytes: int
    disk_free_bytes: int
    gpus: _containers.RepeatedScalarFieldContainer[str]
    platform: str
    def __init__(self, available: _Optional[bool] = ..., reason: _Optional[str] = ..., cpu_count: _Optional[int] = ..., memory_total_bytes: _Optional[int] = ..., memory_available_bytes: _Optional[int] = ..., disk_free_bytes: _Optional[int] = ..., gpus: _Optional[_Iterable[str]] = ..., platform: _Optional[str] = ...) -> None: ...

class ReadinessCheck(_message.Message):
    __slots__ = ("identity", "label", "passed", "state", "version", "detail", "recovery_action")
    IDENTITY_FIELD_NUMBER: _ClassVar[int]
    LABEL_FIELD_NUMBER: _ClassVar[int]
    PASSED_FIELD_NUMBER: _ClassVar[int]
    STATE_FIELD_NUMBER: _ClassVar[int]
    VERSION_FIELD_NUMBER: _ClassVar[int]
    DETAIL_FIELD_NUMBER: _ClassVar[int]
    RECOVERY_ACTION_FIELD_NUMBER: _ClassVar[int]
    identity: str
    label: str
    passed: bool
    state: str
    version: str
    detail: str
    recovery_action: str
    def __init__(self, identity: _Optional[str] = ..., label: _Optional[str] = ..., passed: _Optional[bool] = ..., state: _Optional[str] = ..., version: _Optional[str] = ..., detail: _Optional[str] = ..., recovery_action: _Optional[str] = ...) -> None: ...

class Target(_message.Message):
    __slots__ = ("id", "name", "status", "os", "architecture", "kind", "online", "available", "reason", "next_action", "capabilities", "scopes", "readiness")
    ID_FIELD_NUMBER: _ClassVar[int]
    NAME_FIELD_NUMBER: _ClassVar[int]
    STATUS_FIELD_NUMBER: _ClassVar[int]
    OS_FIELD_NUMBER: _ClassVar[int]
    ARCHITECTURE_FIELD_NUMBER: _ClassVar[int]
    KIND_FIELD_NUMBER: _ClassVar[int]
    ONLINE_FIELD_NUMBER: _ClassVar[int]
    AVAILABLE_FIELD_NUMBER: _ClassVar[int]
    REASON_FIELD_NUMBER: _ClassVar[int]
    NEXT_ACTION_FIELD_NUMBER: _ClassVar[int]
    CAPABILITIES_FIELD_NUMBER: _ClassVar[int]
    SCOPES_FIELD_NUMBER: _ClassVar[int]
    READINESS_FIELD_NUMBER: _ClassVar[int]
    id: str
    name: str
    status: str
    os: str
    architecture: str
    kind: str
    online: bool
    available: bool
    reason: str
    next_action: str
    capabilities: _containers.RepeatedScalarFieldContainer[str]
    scopes: _containers.RepeatedScalarFieldContainer[str]
    readiness: _containers.RepeatedCompositeFieldContainer[ReadinessCheck]
    def __init__(self, id: _Optional[str] = ..., name: _Optional[str] = ..., status: _Optional[str] = ..., os: _Optional[str] = ..., architecture: _Optional[str] = ..., kind: _Optional[str] = ..., online: _Optional[bool] = ..., available: _Optional[bool] = ..., reason: _Optional[str] = ..., next_action: _Optional[str] = ..., capabilities: _Optional[_Iterable[str]] = ..., scopes: _Optional[_Iterable[str]] = ..., readiness: _Optional[_Iterable[_Union[ReadinessCheck, _Mapping]]] = ...) -> None: ...

class ListTargetsResponse(_message.Message):
    __slots__ = ("targets", "error")
    TARGETS_FIELD_NUMBER: _ClassVar[int]
    ERROR_FIELD_NUMBER: _ClassVar[int]
    targets: _containers.RepeatedCompositeFieldContainer[Target]
    error: str
    def __init__(self, targets: _Optional[_Iterable[_Union[Target, _Mapping]]] = ..., error: _Optional[str] = ...) -> None: ...

class PatchHostSafeguardConfigRequest(_message.Message):
    __slots__ = ("target", "safeguard_name", "config_key", "value")
    TARGET_FIELD_NUMBER: _ClassVar[int]
    SAFEGUARD_NAME_FIELD_NUMBER: _ClassVar[int]
    CONFIG_KEY_FIELD_NUMBER: _ClassVar[int]
    VALUE_FIELD_NUMBER: _ClassVar[int]
    target: str
    safeguard_name: str
    config_key: str
    value: _struct_pb2.Value
    def __init__(self, target: _Optional[str] = ..., safeguard_name: _Optional[str] = ..., config_key: _Optional[str] = ..., value: _Optional[_Union[_struct_pb2.Value, _Mapping]] = ...) -> None: ...

class PatchHostSafeguardConfigResponse(_message.Message):
    __slots__ = ("status",)
    STATUS_FIELD_NUMBER: _ClassVar[int]
    status: str
    def __init__(self, status: _Optional[str] = ...) -> None: ...

class SetNotificationRecipientRequest(_message.Message):
    __slots__ = ("target", "subject")
    TARGET_FIELD_NUMBER: _ClassVar[int]
    SUBJECT_FIELD_NUMBER: _ClassVar[int]
    target: str
    subject: str
    def __init__(self, target: _Optional[str] = ..., subject: _Optional[str] = ...) -> None: ...

class SetNotificationRecipientResponse(_message.Message):
    __slots__ = ("status",)
    STATUS_FIELD_NUMBER: _ClassVar[int]
    status: str
    def __init__(self, status: _Optional[str] = ...) -> None: ...
