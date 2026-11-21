from buf.validate import validate_pb2 as _validate_pb2
from google.protobuf import struct_pb2 as _struct_pb2
from dev_routing.v1.routing import routing_pb2 as _routing_pb2
from google.protobuf.internal import enum_type_wrapper as _enum_type_wrapper
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Mapping as _Mapping
from typing import ClassVar as _ClassVar, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class SeedState(int, metaclass=_enum_type_wrapper.EnumTypeWrapper):
    __slots__ = ()
    SEED_STATE_UNSPECIFIED: _ClassVar[SeedState]
    SEED_STATE_ACTIVE: _ClassVar[SeedState]
    SEED_STATE_CLEARED: _ClassVar[SeedState]
    SEED_STATE_FAILED: _ClassVar[SeedState]
SEED_STATE_UNSPECIFIED: SeedState
SEED_STATE_ACTIVE: SeedState
SEED_STATE_CLEARED: SeedState
SEED_STATE_FAILED: SeedState

class ApplySeedRequest(_message.Message):
    __slots__ = ("scenario", "request_id", "scenario_path")
    SCENARIO_FIELD_NUMBER: _ClassVar[int]
    REQUEST_ID_FIELD_NUMBER: _ClassVar[int]
    SCENARIO_PATH_FIELD_NUMBER: _ClassVar[int]
    scenario: str
    request_id: str
    scenario_path: str
    def __init__(self, scenario: _Optional[str] = ..., request_id: _Optional[str] = ..., scenario_path: _Optional[str] = ...) -> None: ...

class ApplySeedResponse(_message.Message):
    __slots__ = ("session_token", "evidence", "initial_params")
    SESSION_TOKEN_FIELD_NUMBER: _ClassVar[int]
    EVIDENCE_FIELD_NUMBER: _ClassVar[int]
    INITIAL_PARAMS_FIELD_NUMBER: _ClassVar[int]
    session_token: str
    evidence: SeedEvidence
    initial_params: _struct_pb2.Struct
    def __init__(self, session_token: _Optional[str] = ..., evidence: _Optional[_Union[SeedEvidence, _Mapping]] = ..., initial_params: _Optional[_Union[_struct_pb2.Struct, _Mapping]] = ...) -> None: ...

class GetSeedRequest(_message.Message):
    __slots__ = ("scenario", "session_token")
    SCENARIO_FIELD_NUMBER: _ClassVar[int]
    SESSION_TOKEN_FIELD_NUMBER: _ClassVar[int]
    scenario: str
    session_token: str
    def __init__(self, scenario: _Optional[str] = ..., session_token: _Optional[str] = ...) -> None: ...

class GetSeedResponse(_message.Message):
    __slots__ = ("evidence",)
    EVIDENCE_FIELD_NUMBER: _ClassVar[int]
    evidence: SeedEvidence
    def __init__(self, evidence: _Optional[_Union[SeedEvidence, _Mapping]] = ...) -> None: ...

class ClearSeedRequest(_message.Message):
    __slots__ = ("scenario", "session_token")
    SCENARIO_FIELD_NUMBER: _ClassVar[int]
    SESSION_TOKEN_FIELD_NUMBER: _ClassVar[int]
    scenario: str
    session_token: str
    def __init__(self, scenario: _Optional[str] = ..., session_token: _Optional[str] = ...) -> None: ...

class ClearSeedResponse(_message.Message):
    __slots__ = ("evidence",)
    EVIDENCE_FIELD_NUMBER: _ClassVar[int]
    evidence: SeedEvidence
    def __init__(self, evidence: _Optional[_Union[SeedEvidence, _Mapping]] = ...) -> None: ...

class SeedEvidence(_message.Message):
    __slots__ = ("scenario", "lease_id", "state", "paired_storage_installed", "heartbeat_error", "cleanup_error", "stats")
    SCENARIO_FIELD_NUMBER: _ClassVar[int]
    LEASE_ID_FIELD_NUMBER: _ClassVar[int]
    STATE_FIELD_NUMBER: _ClassVar[int]
    PAIRED_STORAGE_INSTALLED_FIELD_NUMBER: _ClassVar[int]
    HEARTBEAT_ERROR_FIELD_NUMBER: _ClassVar[int]
    CLEANUP_ERROR_FIELD_NUMBER: _ClassVar[int]
    STATS_FIELD_NUMBER: _ClassVar[int]
    scenario: str
    lease_id: str
    state: SeedState
    paired_storage_installed: bool
    heartbeat_error: str
    cleanup_error: str
    stats: _routing_pb2.LeaseStats
    def __init__(self, scenario: _Optional[str] = ..., lease_id: _Optional[str] = ..., state: _Optional[_Union[SeedState, str]] = ..., paired_storage_installed: _Optional[bool] = ..., heartbeat_error: _Optional[str] = ..., cleanup_error: _Optional[str] = ..., stats: _Optional[_Union[_routing_pb2.LeaseStats, _Mapping]] = ...) -> None: ...
