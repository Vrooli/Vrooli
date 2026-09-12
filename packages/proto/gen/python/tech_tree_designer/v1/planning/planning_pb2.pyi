from google.protobuf.internal import containers as _containers
from google.protobuf.internal import enum_type_wrapper as _enum_type_wrapper
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Iterable as _Iterable, Mapping as _Mapping
from typing import ClassVar as _ClassVar, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class PlanFindingSeverity(int, metaclass=_enum_type_wrapper.EnumTypeWrapper):
    __slots__ = ()
    PLAN_FINDING_SEVERITY_UNSPECIFIED: _ClassVar[PlanFindingSeverity]
    PLAN_FINDING_SEVERITY_ERROR: _ClassVar[PlanFindingSeverity]
    PLAN_FINDING_SEVERITY_WARNING: _ClassVar[PlanFindingSeverity]
    PLAN_FINDING_SEVERITY_INFO: _ClassVar[PlanFindingSeverity]
PLAN_FINDING_SEVERITY_UNSPECIFIED: PlanFindingSeverity
PLAN_FINDING_SEVERITY_ERROR: PlanFindingSeverity
PLAN_FINDING_SEVERITY_WARNING: PlanFindingSeverity
PLAN_FINDING_SEVERITY_INFO: PlanFindingSeverity

class PlannedScenario(_message.Message):
    __slots__ = ("slug", "display_name", "sector", "tier", "target_stability", "files", "created_at", "updated_at")
    SLUG_FIELD_NUMBER: _ClassVar[int]
    DISPLAY_NAME_FIELD_NUMBER: _ClassVar[int]
    SECTOR_FIELD_NUMBER: _ClassVar[int]
    TIER_FIELD_NUMBER: _ClassVar[int]
    TARGET_STABILITY_FIELD_NUMBER: _ClassVar[int]
    FILES_FIELD_NUMBER: _ClassVar[int]
    CREATED_AT_FIELD_NUMBER: _ClassVar[int]
    UPDATED_AT_FIELD_NUMBER: _ClassVar[int]
    slug: str
    display_name: str
    sector: str
    tier: str
    target_stability: str
    files: _containers.RepeatedCompositeFieldContainer[PlannedProtoFile]
    created_at: str
    updated_at: str
    def __init__(self, slug: _Optional[str] = ..., display_name: _Optional[str] = ..., sector: _Optional[str] = ..., tier: _Optional[str] = ..., target_stability: _Optional[str] = ..., files: _Optional[_Iterable[_Union[PlannedProtoFile, _Mapping]]] = ..., created_at: _Optional[str] = ..., updated_at: _Optional[str] = ...) -> None: ...

class PlannedProtoFile(_message.Message):
    __slots__ = ("path", "text", "updated_at")
    PATH_FIELD_NUMBER: _ClassVar[int]
    TEXT_FIELD_NUMBER: _ClassVar[int]
    UPDATED_AT_FIELD_NUMBER: _ClassVar[int]
    path: str
    text: str
    updated_at: str
    def __init__(self, path: _Optional[str] = ..., text: _Optional[str] = ..., updated_at: _Optional[str] = ...) -> None: ...

class CreatePlannedScenarioRequest(_message.Message):
    __slots__ = ("slug", "display_name", "sector", "tier", "target_stability")
    SLUG_FIELD_NUMBER: _ClassVar[int]
    DISPLAY_NAME_FIELD_NUMBER: _ClassVar[int]
    SECTOR_FIELD_NUMBER: _ClassVar[int]
    TIER_FIELD_NUMBER: _ClassVar[int]
    TARGET_STABILITY_FIELD_NUMBER: _ClassVar[int]
    slug: str
    display_name: str
    sector: str
    tier: str
    target_stability: str
    def __init__(self, slug: _Optional[str] = ..., display_name: _Optional[str] = ..., sector: _Optional[str] = ..., tier: _Optional[str] = ..., target_stability: _Optional[str] = ...) -> None: ...

class ListPlannedScenariosRequest(_message.Message):
    __slots__ = ("sector", "tier")
    SECTOR_FIELD_NUMBER: _ClassVar[int]
    TIER_FIELD_NUMBER: _ClassVar[int]
    sector: str
    tier: str
    def __init__(self, sector: _Optional[str] = ..., tier: _Optional[str] = ...) -> None: ...

class ListPlannedScenariosResponse(_message.Message):
    __slots__ = ("scenarios",)
    SCENARIOS_FIELD_NUMBER: _ClassVar[int]
    scenarios: _containers.RepeatedCompositeFieldContainer[PlannedScenario]
    def __init__(self, scenarios: _Optional[_Iterable[_Union[PlannedScenario, _Mapping]]] = ...) -> None: ...

class GetPlannedScenarioRequest(_message.Message):
    __slots__ = ("slug",)
    SLUG_FIELD_NUMBER: _ClassVar[int]
    slug: str
    def __init__(self, slug: _Optional[str] = ...) -> None: ...

class PutPlannedProtoFileRequest(_message.Message):
    __slots__ = ("slug", "path", "text")
    SLUG_FIELD_NUMBER: _ClassVar[int]
    PATH_FIELD_NUMBER: _ClassVar[int]
    TEXT_FIELD_NUMBER: _ClassVar[int]
    slug: str
    path: str
    text: str
    def __init__(self, slug: _Optional[str] = ..., path: _Optional[str] = ..., text: _Optional[str] = ...) -> None: ...

class DeletePlannedProtoFileRequest(_message.Message):
    __slots__ = ("slug", "path")
    SLUG_FIELD_NUMBER: _ClassVar[int]
    PATH_FIELD_NUMBER: _ClassVar[int]
    slug: str
    path: str
    def __init__(self, slug: _Optional[str] = ..., path: _Optional[str] = ...) -> None: ...

class DeletePlannedProtoFileResponse(_message.Message):
    __slots__ = ("deleted",)
    DELETED_FIELD_NUMBER: _ClassVar[int]
    deleted: bool
    def __init__(self, deleted: _Optional[bool] = ...) -> None: ...

class ValidatePlannedScenarioRequest(_message.Message):
    __slots__ = ("slug",)
    SLUG_FIELD_NUMBER: _ClassVar[int]
    slug: str
    def __init__(self, slug: _Optional[str] = ...) -> None: ...

class ValidatePlannedScenarioResponse(_message.Message):
    __slots__ = ("slug", "passed", "findings")
    SLUG_FIELD_NUMBER: _ClassVar[int]
    PASSED_FIELD_NUMBER: _ClassVar[int]
    FINDINGS_FIELD_NUMBER: _ClassVar[int]
    slug: str
    passed: bool
    findings: _containers.RepeatedCompositeFieldContainer[PlanFinding]
    def __init__(self, slug: _Optional[str] = ..., passed: _Optional[bool] = ..., findings: _Optional[_Iterable[_Union[PlanFinding, _Mapping]]] = ...) -> None: ...

class MaterializePlannedScenarioRequest(_message.Message):
    __slots__ = ("slug",)
    SLUG_FIELD_NUMBER: _ClassVar[int]
    slug: str
    def __init__(self, slug: _Optional[str] = ...) -> None: ...

class MaterializePlannedScenarioResponse(_message.Message):
    __slots__ = ("slug", "written_paths", "generated")
    SLUG_FIELD_NUMBER: _ClassVar[int]
    WRITTEN_PATHS_FIELD_NUMBER: _ClassVar[int]
    GENERATED_FIELD_NUMBER: _ClassVar[int]
    slug: str
    written_paths: _containers.RepeatedScalarFieldContainer[str]
    generated: bool
    def __init__(self, slug: _Optional[str] = ..., written_paths: _Optional[_Iterable[str]] = ..., generated: _Optional[bool] = ...) -> None: ...

class PlanFinding(_message.Message):
    __slots__ = ("severity", "code", "location", "message", "suggestion")
    SEVERITY_FIELD_NUMBER: _ClassVar[int]
    CODE_FIELD_NUMBER: _ClassVar[int]
    LOCATION_FIELD_NUMBER: _ClassVar[int]
    MESSAGE_FIELD_NUMBER: _ClassVar[int]
    SUGGESTION_FIELD_NUMBER: _ClassVar[int]
    severity: PlanFindingSeverity
    code: str
    location: str
    message: str
    suggestion: str
    def __init__(self, severity: _Optional[_Union[PlanFindingSeverity, str]] = ..., code: _Optional[str] = ..., location: _Optional[str] = ..., message: _Optional[str] = ..., suggestion: _Optional[str] = ...) -> None: ...
