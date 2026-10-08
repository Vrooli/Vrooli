from google.protobuf.internal import containers as _containers
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Iterable as _Iterable, Mapping as _Mapping
from typing import ClassVar as _ClassVar, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class ScenarioFreshnessResponse(_message.Message):
    __slots__ = ("success", "scenario", "stale", "checks", "dependencies", "inputs")
    SUCCESS_FIELD_NUMBER: _ClassVar[int]
    SCENARIO_FIELD_NUMBER: _ClassVar[int]
    STALE_FIELD_NUMBER: _ClassVar[int]
    CHECKS_FIELD_NUMBER: _ClassVar[int]
    DEPENDENCIES_FIELD_NUMBER: _ClassVar[int]
    INPUTS_FIELD_NUMBER: _ClassVar[int]
    success: bool
    scenario: str
    stale: bool
    checks: _containers.RepeatedCompositeFieldContainer[ScenarioFreshnessCheck]
    dependencies: _containers.RepeatedCompositeFieldContainer[ScenarioFreshnessDependency]
    inputs: ScenarioFreshnessInputs
    def __init__(self, success: _Optional[bool] = ..., scenario: _Optional[str] = ..., stale: _Optional[bool] = ..., checks: _Optional[_Iterable[_Union[ScenarioFreshnessCheck, _Mapping]]] = ..., dependencies: _Optional[_Iterable[_Union[ScenarioFreshnessDependency, _Mapping]]] = ..., inputs: _Optional[_Union[ScenarioFreshnessInputs, _Mapping]] = ...) -> None: ...

class ScenarioFreshnessInputs(_message.Message):
    __slots__ = ("paths", "build_keys")
    class BuildKeysEntry(_message.Message):
        __slots__ = ("key", "value")
        KEY_FIELD_NUMBER: _ClassVar[int]
        VALUE_FIELD_NUMBER: _ClassVar[int]
        key: str
        value: str
        def __init__(self, key: _Optional[str] = ..., value: _Optional[str] = ...) -> None: ...
    PATHS_FIELD_NUMBER: _ClassVar[int]
    BUILD_KEYS_FIELD_NUMBER: _ClassVar[int]
    paths: _containers.RepeatedScalarFieldContainer[str]
    build_keys: _containers.ScalarMap[str, str]
    def __init__(self, paths: _Optional[_Iterable[str]] = ..., build_keys: _Optional[_Mapping[str, str]] = ...) -> None: ...

class ScenarioFreshnessCheck(_message.Message):
    __slots__ = ("check_type", "target", "stale", "cause", "file")
    CHECK_TYPE_FIELD_NUMBER: _ClassVar[int]
    TARGET_FIELD_NUMBER: _ClassVar[int]
    STALE_FIELD_NUMBER: _ClassVar[int]
    CAUSE_FIELD_NUMBER: _ClassVar[int]
    FILE_FIELD_NUMBER: _ClassVar[int]
    check_type: str
    target: str
    stale: bool
    cause: str
    file: str
    def __init__(self, check_type: _Optional[str] = ..., target: _Optional[str] = ..., stale: _Optional[bool] = ..., cause: _Optional[str] = ..., file: _Optional[str] = ...) -> None: ...

class ScenarioFreshnessDependency(_message.Message):
    __slots__ = ("name", "policy")
    NAME_FIELD_NUMBER: _ClassVar[int]
    POLICY_FIELD_NUMBER: _ClassVar[int]
    name: str
    policy: str
    def __init__(self, name: _Optional[str] = ..., policy: _Optional[str] = ...) -> None: ...
