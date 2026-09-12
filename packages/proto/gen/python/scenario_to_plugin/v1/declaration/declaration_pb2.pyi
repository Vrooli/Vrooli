from google.protobuf.internal import containers as _containers
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Iterable as _Iterable, Mapping as _Mapping
from typing import ClassVar as _ClassVar, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class GetDeclarationRequest(_message.Message):
    __slots__ = ("scenario",)
    SCENARIO_FIELD_NUMBER: _ClassVar[int]
    scenario: str
    def __init__(self, scenario: _Optional[str] = ...) -> None: ...

class GetDeclarationResponse(_message.Message):
    __slots__ = ("declaration", "readiness")
    DECLARATION_FIELD_NUMBER: _ClassVar[int]
    READINESS_FIELD_NUMBER: _ClassVar[int]
    declaration: Declaration
    readiness: Readiness
    def __init__(self, declaration: _Optional[_Union[Declaration, _Mapping]] = ..., readiness: _Optional[_Union[Readiness, _Mapping]] = ...) -> None: ...

class ListReadinessRequest(_message.Message):
    __slots__ = ("scenarios",)
    SCENARIOS_FIELD_NUMBER: _ClassVar[int]
    scenarios: _containers.RepeatedScalarFieldContainer[str]
    def __init__(self, scenarios: _Optional[_Iterable[str]] = ...) -> None: ...

class ListReadinessResponse(_message.Message):
    __slots__ = ("items",)
    ITEMS_FIELD_NUMBER: _ClassVar[int]
    items: _containers.RepeatedCompositeFieldContainer[Readiness]
    def __init__(self, items: _Optional[_Iterable[_Union[Readiness, _Mapping]]] = ...) -> None: ...

class Declaration(_message.Message):
    __slots__ = ("scenario", "slug", "skills", "entitlement_tier", "standalone", "mcp")
    SCENARIO_FIELD_NUMBER: _ClassVar[int]
    SLUG_FIELD_NUMBER: _ClassVar[int]
    SKILLS_FIELD_NUMBER: _ClassVar[int]
    ENTITLEMENT_TIER_FIELD_NUMBER: _ClassVar[int]
    STANDALONE_FIELD_NUMBER: _ClassVar[int]
    MCP_FIELD_NUMBER: _ClassVar[int]
    scenario: str
    slug: str
    skills: _containers.RepeatedCompositeFieldContainer[Skill]
    entitlement_tier: str
    standalone: Standalone
    mcp: MCP
    def __init__(self, scenario: _Optional[str] = ..., slug: _Optional[str] = ..., skills: _Optional[_Iterable[_Union[Skill, _Mapping]]] = ..., entitlement_tier: _Optional[str] = ..., standalone: _Optional[_Union[Standalone, _Mapping]] = ..., mcp: _Optional[_Union[MCP, _Mapping]] = ...) -> None: ...

class Skill(_message.Message):
    __slots__ = ("name", "source", "command_groups")
    NAME_FIELD_NUMBER: _ClassVar[int]
    SOURCE_FIELD_NUMBER: _ClassVar[int]
    COMMAND_GROUPS_FIELD_NUMBER: _ClassVar[int]
    name: str
    source: str
    command_groups: _containers.RepeatedScalarFieldContainer[str]
    def __init__(self, name: _Optional[str] = ..., source: _Optional[str] = ..., command_groups: _Optional[_Iterable[str]] = ...) -> None: ...

class Standalone(_message.Message):
    __slots__ = ("install_script", "runtime_binaries", "resources")
    INSTALL_SCRIPT_FIELD_NUMBER: _ClassVar[int]
    RUNTIME_BINARIES_FIELD_NUMBER: _ClassVar[int]
    RESOURCES_FIELD_NUMBER: _ClassVar[int]
    install_script: str
    runtime_binaries: _containers.RepeatedScalarFieldContainer[str]
    resources: _containers.RepeatedScalarFieldContainer[str]
    def __init__(self, install_script: _Optional[str] = ..., runtime_binaries: _Optional[_Iterable[str]] = ..., resources: _Optional[_Iterable[str]] = ...) -> None: ...

class MCP(_message.Message):
    __slots__ = ("name", "command", "args", "authentication")
    NAME_FIELD_NUMBER: _ClassVar[int]
    COMMAND_FIELD_NUMBER: _ClassVar[int]
    ARGS_FIELD_NUMBER: _ClassVar[int]
    AUTHENTICATION_FIELD_NUMBER: _ClassVar[int]
    name: str
    command: str
    args: _containers.RepeatedScalarFieldContainer[str]
    authentication: str
    def __init__(self, name: _Optional[str] = ..., command: _Optional[str] = ..., args: _Optional[_Iterable[str]] = ..., authentication: _Optional[str] = ...) -> None: ...

class Readiness(_message.Message):
    __slots__ = ("scenario", "eligible", "prerequisites", "blocking_prerequisite")
    SCENARIO_FIELD_NUMBER: _ClassVar[int]
    ELIGIBLE_FIELD_NUMBER: _ClassVar[int]
    PREREQUISITES_FIELD_NUMBER: _ClassVar[int]
    BLOCKING_PREREQUISITE_FIELD_NUMBER: _ClassVar[int]
    scenario: str
    eligible: bool
    prerequisites: _containers.RepeatedCompositeFieldContainer[Prerequisite]
    blocking_prerequisite: str
    def __init__(self, scenario: _Optional[str] = ..., eligible: _Optional[bool] = ..., prerequisites: _Optional[_Iterable[_Union[Prerequisite, _Mapping]]] = ..., blocking_prerequisite: _Optional[str] = ...) -> None: ...

class Prerequisite(_message.Message):
    __slots__ = ("code", "description", "satisfied")
    CODE_FIELD_NUMBER: _ClassVar[int]
    DESCRIPTION_FIELD_NUMBER: _ClassVar[int]
    SATISFIED_FIELD_NUMBER: _ClassVar[int]
    code: str
    description: str
    satisfied: bool
    def __init__(self, code: _Optional[str] = ..., description: _Optional[str] = ..., satisfied: _Optional[bool] = ...) -> None: ...
