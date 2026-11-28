from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Mapping as _Mapping
from typing import ClassVar as _ClassVar, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class DeploymentRef(_message.Message):
    __slots__ = ("id", "scenario_id", "environment", "target")
    ID_FIELD_NUMBER: _ClassVar[int]
    SCENARIO_ID_FIELD_NUMBER: _ClassVar[int]
    ENVIRONMENT_FIELD_NUMBER: _ClassVar[int]
    TARGET_FIELD_NUMBER: _ClassVar[int]
    id: str
    scenario_id: str
    environment: str
    target: TargetRef
    def __init__(self, id: _Optional[str] = ..., scenario_id: _Optional[str] = ..., environment: _Optional[str] = ..., target: _Optional[_Union[TargetRef, _Mapping]] = ...) -> None: ...

class TargetRef(_message.Message):
    __slots__ = ("machine_id", "node_id", "enrollment_generation", "transport", "locator")
    MACHINE_ID_FIELD_NUMBER: _ClassVar[int]
    NODE_ID_FIELD_NUMBER: _ClassVar[int]
    ENROLLMENT_GENERATION_FIELD_NUMBER: _ClassVar[int]
    TRANSPORT_FIELD_NUMBER: _ClassVar[int]
    LOCATOR_FIELD_NUMBER: _ClassVar[int]
    machine_id: str
    node_id: str
    enrollment_generation: int
    transport: str
    locator: TargetLocator
    def __init__(self, machine_id: _Optional[str] = ..., node_id: _Optional[str] = ..., enrollment_generation: _Optional[int] = ..., transport: _Optional[str] = ..., locator: _Optional[_Union[TargetLocator, _Mapping]] = ...) -> None: ...

class TargetLocator(_message.Message):
    __slots__ = ("host", "port", "user", "workdir")
    HOST_FIELD_NUMBER: _ClassVar[int]
    PORT_FIELD_NUMBER: _ClassVar[int]
    USER_FIELD_NUMBER: _ClassVar[int]
    WORKDIR_FIELD_NUMBER: _ClassVar[int]
    host: str
    port: int
    user: str
    workdir: str
    def __init__(self, host: _Optional[str] = ..., port: _Optional[int] = ..., user: _Optional[str] = ..., workdir: _Optional[str] = ...) -> None: ...

class ReleaseRef(_message.Message):
    __slots__ = ("digest", "provenance_ref", "closure_digest", "configuration_digest")
    DIGEST_FIELD_NUMBER: _ClassVar[int]
    PROVENANCE_REF_FIELD_NUMBER: _ClassVar[int]
    CLOSURE_DIGEST_FIELD_NUMBER: _ClassVar[int]
    CONFIGURATION_DIGEST_FIELD_NUMBER: _ClassVar[int]
    digest: str
    provenance_ref: str
    closure_digest: str
    configuration_digest: str
    def __init__(self, digest: _Optional[str] = ..., provenance_ref: _Optional[str] = ..., closure_digest: _Optional[str] = ..., configuration_digest: _Optional[str] = ...) -> None: ...

class OperationRef(_message.Message):
    __slots__ = ("id", "request_key", "plan_digest", "fence")
    ID_FIELD_NUMBER: _ClassVar[int]
    REQUEST_KEY_FIELD_NUMBER: _ClassVar[int]
    PLAN_DIGEST_FIELD_NUMBER: _ClassVar[int]
    FENCE_FIELD_NUMBER: _ClassVar[int]
    id: str
    request_key: str
    plan_digest: str
    fence: int
    def __init__(self, id: _Optional[str] = ..., request_key: _Optional[str] = ..., plan_digest: _Optional[str] = ..., fence: _Optional[int] = ...) -> None: ...
