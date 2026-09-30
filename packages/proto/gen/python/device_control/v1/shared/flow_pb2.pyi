from google.protobuf import struct_pb2 as _struct_pb2
from google.protobuf.internal import containers as _containers
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Iterable as _Iterable, Mapping as _Mapping
from typing import ClassVar as _ClassVar, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class Flow(_message.Message):
    __slots__ = ("id", "name", "steps", "allow_unredacted_capture", "transport", "require_unlocked", "auth_profile_id", "max_duration_ms", "retry_budget", "application_id", "application_revision")
    ID_FIELD_NUMBER: _ClassVar[int]
    NAME_FIELD_NUMBER: _ClassVar[int]
    STEPS_FIELD_NUMBER: _ClassVar[int]
    ALLOW_UNREDACTED_CAPTURE_FIELD_NUMBER: _ClassVar[int]
    TRANSPORT_FIELD_NUMBER: _ClassVar[int]
    REQUIRE_UNLOCKED_FIELD_NUMBER: _ClassVar[int]
    AUTH_PROFILE_ID_FIELD_NUMBER: _ClassVar[int]
    MAX_DURATION_MS_FIELD_NUMBER: _ClassVar[int]
    RETRY_BUDGET_FIELD_NUMBER: _ClassVar[int]
    APPLICATION_ID_FIELD_NUMBER: _ClassVar[int]
    APPLICATION_REVISION_FIELD_NUMBER: _ClassVar[int]
    id: str
    name: str
    steps: _containers.RepeatedCompositeFieldContainer[Step]
    allow_unredacted_capture: bool
    transport: str
    require_unlocked: bool
    auth_profile_id: str
    max_duration_ms: int
    retry_budget: int
    application_id: str
    application_revision: str
    def __init__(self, id: _Optional[str] = ..., name: _Optional[str] = ..., steps: _Optional[_Iterable[_Union[Step, _Mapping]]] = ..., allow_unredacted_capture: _Optional[bool] = ..., transport: _Optional[str] = ..., require_unlocked: _Optional[bool] = ..., auth_profile_id: _Optional[str] = ..., max_duration_ms: _Optional[int] = ..., retry_budget: _Optional[int] = ..., application_id: _Optional[str] = ..., application_revision: _Optional[str] = ...) -> None: ...

class Step(_message.Message):
    __slots__ = ("id", "kind", "required_capabilities", "target", "timeout_ms", "arguments", "preconditions", "postconditions", "observation_required", "retry_budget", "idempotency_key")
    ID_FIELD_NUMBER: _ClassVar[int]
    KIND_FIELD_NUMBER: _ClassVar[int]
    REQUIRED_CAPABILITIES_FIELD_NUMBER: _ClassVar[int]
    TARGET_FIELD_NUMBER: _ClassVar[int]
    TIMEOUT_MS_FIELD_NUMBER: _ClassVar[int]
    ARGUMENTS_FIELD_NUMBER: _ClassVar[int]
    PRECONDITIONS_FIELD_NUMBER: _ClassVar[int]
    POSTCONDITIONS_FIELD_NUMBER: _ClassVar[int]
    OBSERVATION_REQUIRED_FIELD_NUMBER: _ClassVar[int]
    RETRY_BUDGET_FIELD_NUMBER: _ClassVar[int]
    IDEMPOTENCY_KEY_FIELD_NUMBER: _ClassVar[int]
    id: str
    kind: str
    required_capabilities: _containers.RepeatedScalarFieldContainer[str]
    target: str
    timeout_ms: int
    arguments: _struct_pb2.Struct
    preconditions: _containers.RepeatedCompositeFieldContainer[Condition]
    postconditions: _containers.RepeatedCompositeFieldContainer[Condition]
    observation_required: bool
    retry_budget: int
    idempotency_key: str
    def __init__(self, id: _Optional[str] = ..., kind: _Optional[str] = ..., required_capabilities: _Optional[_Iterable[str]] = ..., target: _Optional[str] = ..., timeout_ms: _Optional[int] = ..., arguments: _Optional[_Union[_struct_pb2.Struct, _Mapping]] = ..., preconditions: _Optional[_Iterable[_Union[Condition, _Mapping]]] = ..., postconditions: _Optional[_Iterable[_Union[Condition, _Mapping]]] = ..., observation_required: _Optional[bool] = ..., retry_budget: _Optional[int] = ..., idempotency_key: _Optional[str] = ...) -> None: ...

class Condition(_message.Message):
    __slots__ = ("kind", "target", "expected")
    KIND_FIELD_NUMBER: _ClassVar[int]
    TARGET_FIELD_NUMBER: _ClassVar[int]
    EXPECTED_FIELD_NUMBER: _ClassVar[int]
    kind: str
    target: str
    expected: _struct_pb2.Value
    def __init__(self, kind: _Optional[str] = ..., target: _Optional[str] = ..., expected: _Optional[_Union[_struct_pb2.Value, _Mapping]] = ...) -> None: ...
