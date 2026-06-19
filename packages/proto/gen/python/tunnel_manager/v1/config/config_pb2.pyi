from google.protobuf.internal import containers as _containers
from google.protobuf.internal import enum_type_wrapper as _enum_type_wrapper
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Iterable as _Iterable, Mapping as _Mapping
from typing import ClassVar as _ClassVar, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class Mode(int, metaclass=_enum_type_wrapper.EnumTypeWrapper):
    __slots__ = ()
    MODE_UNSPECIFIED: _ClassVar[Mode]
    MODE_REMOTE: _ClassVar[Mode]
    MODE_LOCAL: _ClassVar[Mode]
MODE_UNSPECIFIED: Mode
MODE_REMOTE: Mode
MODE_LOCAL: Mode

class TunnelConfig(_message.Message):
    __slots__ = ("mode", "tunnel_id", "account_id", "cred_ref", "prom_endpoint")
    MODE_FIELD_NUMBER: _ClassVar[int]
    TUNNEL_ID_FIELD_NUMBER: _ClassVar[int]
    ACCOUNT_ID_FIELD_NUMBER: _ClassVar[int]
    CRED_REF_FIELD_NUMBER: _ClassVar[int]
    PROM_ENDPOINT_FIELD_NUMBER: _ClassVar[int]
    mode: Mode
    tunnel_id: str
    account_id: str
    cred_ref: str
    prom_endpoint: str
    def __init__(self, mode: _Optional[_Union[Mode, str]] = ..., tunnel_id: _Optional[str] = ..., account_id: _Optional[str] = ..., cred_ref: _Optional[str] = ..., prom_endpoint: _Optional[str] = ...) -> None: ...

class ConfigReadiness(_message.Message):
    __slots__ = ("desired_mode", "remote_available", "missing_fields", "credential_source", "credential_ref", "local_config_path", "sync_ready", "mode_reason")
    DESIRED_MODE_FIELD_NUMBER: _ClassVar[int]
    REMOTE_AVAILABLE_FIELD_NUMBER: _ClassVar[int]
    MISSING_FIELDS_FIELD_NUMBER: _ClassVar[int]
    CREDENTIAL_SOURCE_FIELD_NUMBER: _ClassVar[int]
    CREDENTIAL_REF_FIELD_NUMBER: _ClassVar[int]
    LOCAL_CONFIG_PATH_FIELD_NUMBER: _ClassVar[int]
    SYNC_READY_FIELD_NUMBER: _ClassVar[int]
    MODE_REASON_FIELD_NUMBER: _ClassVar[int]
    desired_mode: Mode
    remote_available: bool
    missing_fields: _containers.RepeatedScalarFieldContainer[str]
    credential_source: str
    credential_ref: str
    local_config_path: str
    sync_ready: bool
    mode_reason: str
    def __init__(self, desired_mode: _Optional[_Union[Mode, str]] = ..., remote_available: _Optional[bool] = ..., missing_fields: _Optional[_Iterable[str]] = ..., credential_source: _Optional[str] = ..., credential_ref: _Optional[str] = ..., local_config_path: _Optional[str] = ..., sync_ready: _Optional[bool] = ..., mode_reason: _Optional[str] = ...) -> None: ...

class GetConfigRequest(_message.Message):
    __slots__ = ()
    def __init__(self) -> None: ...

class GetConfigResponse(_message.Message):
    __slots__ = ("config", "readiness")
    CONFIG_FIELD_NUMBER: _ClassVar[int]
    READINESS_FIELD_NUMBER: _ClassVar[int]
    config: TunnelConfig
    readiness: ConfigReadiness
    def __init__(self, config: _Optional[_Union[TunnelConfig, _Mapping]] = ..., readiness: _Optional[_Union[ConfigReadiness, _Mapping]] = ...) -> None: ...

class SyncRequest(_message.Message):
    __slots__ = ("dry_run",)
    DRY_RUN_FIELD_NUMBER: _ClassVar[int]
    dry_run: bool
    def __init__(self, dry_run: _Optional[bool] = ...) -> None: ...

class SyncResponse(_message.Message):
    __slots__ = ("mode", "added", "removed", "no_changes", "setup_required", "missing_fields", "message")
    MODE_FIELD_NUMBER: _ClassVar[int]
    ADDED_FIELD_NUMBER: _ClassVar[int]
    REMOVED_FIELD_NUMBER: _ClassVar[int]
    NO_CHANGES_FIELD_NUMBER: _ClassVar[int]
    SETUP_REQUIRED_FIELD_NUMBER: _ClassVar[int]
    MISSING_FIELDS_FIELD_NUMBER: _ClassVar[int]
    MESSAGE_FIELD_NUMBER: _ClassVar[int]
    mode: Mode
    added: _containers.RepeatedScalarFieldContainer[str]
    removed: _containers.RepeatedScalarFieldContainer[str]
    no_changes: bool
    setup_required: bool
    missing_fields: _containers.RepeatedScalarFieldContainer[str]
    message: str
    def __init__(self, mode: _Optional[_Union[Mode, str]] = ..., added: _Optional[_Iterable[str]] = ..., removed: _Optional[_Iterable[str]] = ..., no_changes: _Optional[bool] = ..., setup_required: _Optional[bool] = ..., missing_fields: _Optional[_Iterable[str]] = ..., message: _Optional[str] = ...) -> None: ...

class SwitchModeRequest(_message.Message):
    __slots__ = ("target_mode",)
    TARGET_MODE_FIELD_NUMBER: _ClassVar[int]
    target_mode: Mode
    def __init__(self, target_mode: _Optional[_Union[Mode, str]] = ...) -> None: ...

class SwitchModeResponse(_message.Message):
    __slots__ = ("previous_mode", "current_mode")
    PREVIOUS_MODE_FIELD_NUMBER: _ClassVar[int]
    CURRENT_MODE_FIELD_NUMBER: _ClassVar[int]
    previous_mode: Mode
    current_mode: Mode
    def __init__(self, previous_mode: _Optional[_Union[Mode, str]] = ..., current_mode: _Optional[_Union[Mode, str]] = ...) -> None: ...
