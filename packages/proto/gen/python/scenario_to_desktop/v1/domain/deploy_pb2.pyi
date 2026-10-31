from buf.validate import validate_pb2 as _validate_pb2
from google.protobuf.internal import containers as _containers
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Iterable as _Iterable, Mapping as _Mapping
from typing import ClassVar as _ClassVar, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class DeployTarget(_message.Message):
    __slots__ = ("name", "label", "scenario_name", "remote_profile", "deployment_manager_profile_id")
    NAME_FIELD_NUMBER: _ClassVar[int]
    LABEL_FIELD_NUMBER: _ClassVar[int]
    SCENARIO_NAME_FIELD_NUMBER: _ClassVar[int]
    REMOTE_PROFILE_FIELD_NUMBER: _ClassVar[int]
    DEPLOYMENT_MANAGER_PROFILE_ID_FIELD_NUMBER: _ClassVar[int]
    name: str
    label: str
    scenario_name: str
    remote_profile: str
    deployment_manager_profile_id: str
    def __init__(self, name: _Optional[str] = ..., label: _Optional[str] = ..., scenario_name: _Optional[str] = ..., remote_profile: _Optional[str] = ..., deployment_manager_profile_id: _Optional[str] = ...) -> None: ...

class DeployTargetNameRequest(_message.Message):
    __slots__ = ("name",)
    NAME_FIELD_NUMBER: _ClassVar[int]
    name: str
    def __init__(self, name: _Optional[str] = ...) -> None: ...

class ListDeployTargetsRequest(_message.Message):
    __slots__ = ()
    def __init__(self) -> None: ...

class ListDeployTargetsResponse(_message.Message):
    __slots__ = ("targets",)
    TARGETS_FIELD_NUMBER: _ClassVar[int]
    targets: _containers.RepeatedCompositeFieldContainer[DeployTarget]
    def __init__(self, targets: _Optional[_Iterable[_Union[DeployTarget, _Mapping]]] = ...) -> None: ...

class GetDeployTargetResponse(_message.Message):
    __slots__ = ("target",)
    TARGET_FIELD_NUMBER: _ClassVar[int]
    target: DeployTarget
    def __init__(self, target: _Optional[_Union[DeployTarget, _Mapping]] = ...) -> None: ...

class SaveDeployTargetRequest(_message.Message):
    __slots__ = ("target",)
    TARGET_FIELD_NUMBER: _ClassVar[int]
    target: DeployTarget
    def __init__(self, target: _Optional[_Union[DeployTarget, _Mapping]] = ...) -> None: ...

class SaveDeployTargetResponse(_message.Message):
    __slots__ = ("target",)
    TARGET_FIELD_NUMBER: _ClassVar[int]
    target: DeployTarget
    def __init__(self, target: _Optional[_Union[DeployTarget, _Mapping]] = ...) -> None: ...

class DeleteDeployTargetResponse(_message.Message):
    __slots__ = ("name", "deleted")
    NAME_FIELD_NUMBER: _ClassVar[int]
    DELETED_FIELD_NUMBER: _ClassVar[int]
    name: str
    deleted: bool
    def __init__(self, name: _Optional[str] = ..., deleted: _Optional[bool] = ...) -> None: ...

class TestDeployTargetRequest(_message.Message):
    __slots__ = ("name", "require_service_auth")
    NAME_FIELD_NUMBER: _ClassVar[int]
    REQUIRE_SERVICE_AUTH_FIELD_NUMBER: _ClassVar[int]
    name: str
    require_service_auth: bool
    def __init__(self, name: _Optional[str] = ..., require_service_auth: _Optional[bool] = ...) -> None: ...

class TestDeployTargetResponse(_message.Message):
    __slots__ = ("target", "service_auth_checked")
    TARGET_FIELD_NUMBER: _ClassVar[int]
    SERVICE_AUTH_CHECKED_FIELD_NUMBER: _ClassVar[int]
    target: DeployTarget
    service_auth_checked: bool
    def __init__(self, target: _Optional[_Union[DeployTarget, _Mapping]] = ..., service_auth_checked: _Optional[bool] = ...) -> None: ...

class DeployTargetReadinessCheck(_message.Message):
    __slots__ = ("name", "required", "passed", "blocked", "detail")
    NAME_FIELD_NUMBER: _ClassVar[int]
    REQUIRED_FIELD_NUMBER: _ClassVar[int]
    PASSED_FIELD_NUMBER: _ClassVar[int]
    BLOCKED_FIELD_NUMBER: _ClassVar[int]
    DETAIL_FIELD_NUMBER: _ClassVar[int]
    name: str
    required: bool
    passed: bool
    blocked: bool
    detail: str
    def __init__(self, name: _Optional[str] = ..., required: _Optional[bool] = ..., passed: _Optional[bool] = ..., blocked: _Optional[bool] = ..., detail: _Optional[str] = ...) -> None: ...

class DiagnoseDeployTargetResponse(_message.Message):
    __slots__ = ("target", "ready", "checks", "next_steps")
    TARGET_FIELD_NUMBER: _ClassVar[int]
    READY_FIELD_NUMBER: _ClassVar[int]
    CHECKS_FIELD_NUMBER: _ClassVar[int]
    NEXT_STEPS_FIELD_NUMBER: _ClassVar[int]
    target: DeployTarget
    ready: bool
    checks: _containers.RepeatedCompositeFieldContainer[DeployTargetReadinessCheck]
    next_steps: _containers.RepeatedScalarFieldContainer[str]
    def __init__(self, target: _Optional[_Union[DeployTarget, _Mapping]] = ..., ready: _Optional[bool] = ..., checks: _Optional[_Iterable[_Union[DeployTargetReadinessCheck, _Mapping]]] = ..., next_steps: _Optional[_Iterable[str]] = ...) -> None: ...
