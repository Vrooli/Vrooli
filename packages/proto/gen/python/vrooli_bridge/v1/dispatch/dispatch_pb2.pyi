from google.protobuf.internal import containers as _containers
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Iterable as _Iterable, Mapping as _Mapping
from typing import ClassVar as _ClassVar, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class DispatchJobRequest(_message.Message):
    __slots__ = ("node_id", "scenario", "verb", "args", "timeout_seconds", "device_id", "lease_token", "credential_injections")
    NODE_ID_FIELD_NUMBER: _ClassVar[int]
    SCENARIO_FIELD_NUMBER: _ClassVar[int]
    VERB_FIELD_NUMBER: _ClassVar[int]
    ARGS_FIELD_NUMBER: _ClassVar[int]
    TIMEOUT_SECONDS_FIELD_NUMBER: _ClassVar[int]
    DEVICE_ID_FIELD_NUMBER: _ClassVar[int]
    LEASE_TOKEN_FIELD_NUMBER: _ClassVar[int]
    CREDENTIAL_INJECTIONS_FIELD_NUMBER: _ClassVar[int]
    node_id: str
    scenario: str
    verb: str
    args: _containers.RepeatedScalarFieldContainer[str]
    timeout_seconds: int
    device_id: str
    lease_token: str
    credential_injections: _containers.RepeatedCompositeFieldContainer[CredentialInjection]
    def __init__(self, node_id: _Optional[str] = ..., scenario: _Optional[str] = ..., verb: _Optional[str] = ..., args: _Optional[_Iterable[str]] = ..., timeout_seconds: _Optional[int] = ..., device_id: _Optional[str] = ..., lease_token: _Optional[str] = ..., credential_injections: _Optional[_Iterable[_Union[CredentialInjection, _Mapping]]] = ...) -> None: ...

class CredentialInjection(_message.Message):
    __slots__ = ("logical_id", "field", "env_name")
    LOGICAL_ID_FIELD_NUMBER: _ClassVar[int]
    FIELD_FIELD_NUMBER: _ClassVar[int]
    ENV_NAME_FIELD_NUMBER: _ClassVar[int]
    logical_id: str
    field: str
    env_name: str
    def __init__(self, logical_id: _Optional[str] = ..., field: _Optional[str] = ..., env_name: _Optional[str] = ...) -> None: ...

class DispatchJobResponse(_message.Message):
    __slots__ = ("run_id", "dry_run", "node_id", "scenario", "verb", "args", "queued")
    RUN_ID_FIELD_NUMBER: _ClassVar[int]
    DRY_RUN_FIELD_NUMBER: _ClassVar[int]
    NODE_ID_FIELD_NUMBER: _ClassVar[int]
    SCENARIO_FIELD_NUMBER: _ClassVar[int]
    VERB_FIELD_NUMBER: _ClassVar[int]
    ARGS_FIELD_NUMBER: _ClassVar[int]
    QUEUED_FIELD_NUMBER: _ClassVar[int]
    run_id: str
    dry_run: bool
    node_id: str
    scenario: str
    verb: str
    args: _containers.RepeatedScalarFieldContainer[str]
    queued: bool
    def __init__(self, run_id: _Optional[str] = ..., dry_run: _Optional[bool] = ..., node_id: _Optional[str] = ..., scenario: _Optional[str] = ..., verb: _Optional[str] = ..., args: _Optional[_Iterable[str]] = ..., queued: _Optional[bool] = ...) -> None: ...
