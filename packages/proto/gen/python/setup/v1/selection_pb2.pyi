from google.protobuf.internal import containers as _containers
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Iterable as _Iterable, Mapping as _Mapping
from typing import ClassVar as _ClassVar, Optional as _Optional

DESCRIPTOR: _descriptor.FileDescriptor

class Selection(_message.Message):
    __slots__ = ("schema_version", "target", "scenarios", "optional_resources", "core_seed", "trusted_base", "host_tools", "host_safeguards", "credential_addresses", "trust_posture", "update_control", "session_mode", "operating_mode", "apply")
    class OperatingModeEntry(_message.Message):
        __slots__ = ("key", "value")
        KEY_FIELD_NUMBER: _ClassVar[int]
        VALUE_FIELD_NUMBER: _ClassVar[int]
        key: str
        value: str
        def __init__(self, key: _Optional[str] = ..., value: _Optional[str] = ...) -> None: ...
    SCHEMA_VERSION_FIELD_NUMBER: _ClassVar[int]
    TARGET_FIELD_NUMBER: _ClassVar[int]
    SCENARIOS_FIELD_NUMBER: _ClassVar[int]
    OPTIONAL_RESOURCES_FIELD_NUMBER: _ClassVar[int]
    CORE_SEED_FIELD_NUMBER: _ClassVar[int]
    TRUSTED_BASE_FIELD_NUMBER: _ClassVar[int]
    HOST_TOOLS_FIELD_NUMBER: _ClassVar[int]
    HOST_SAFEGUARDS_FIELD_NUMBER: _ClassVar[int]
    CREDENTIAL_ADDRESSES_FIELD_NUMBER: _ClassVar[int]
    TRUST_POSTURE_FIELD_NUMBER: _ClassVar[int]
    UPDATE_CONTROL_FIELD_NUMBER: _ClassVar[int]
    SESSION_MODE_FIELD_NUMBER: _ClassVar[int]
    OPERATING_MODE_FIELD_NUMBER: _ClassVar[int]
    APPLY_FIELD_NUMBER: _ClassVar[int]
    schema_version: str
    target: str
    scenarios: _containers.RepeatedScalarFieldContainer[str]
    optional_resources: _containers.RepeatedScalarFieldContainer[str]
    core_seed: _containers.RepeatedScalarFieldContainer[str]
    trusted_base: _containers.RepeatedScalarFieldContainer[str]
    host_tools: _containers.RepeatedScalarFieldContainer[str]
    host_safeguards: _containers.RepeatedScalarFieldContainer[str]
    credential_addresses: _containers.RepeatedScalarFieldContainer[str]
    trust_posture: str
    update_control: str
    session_mode: str
    operating_mode: _containers.ScalarMap[str, str]
    apply: bool
    def __init__(self, schema_version: _Optional[str] = ..., target: _Optional[str] = ..., scenarios: _Optional[_Iterable[str]] = ..., optional_resources: _Optional[_Iterable[str]] = ..., core_seed: _Optional[_Iterable[str]] = ..., trusted_base: _Optional[_Iterable[str]] = ..., host_tools: _Optional[_Iterable[str]] = ..., host_safeguards: _Optional[_Iterable[str]] = ..., credential_addresses: _Optional[_Iterable[str]] = ..., trust_posture: _Optional[str] = ..., update_control: _Optional[str] = ..., session_mode: _Optional[str] = ..., operating_mode: _Optional[_Mapping[str, str]] = ..., apply: _Optional[bool] = ...) -> None: ...
