import datetime

from google.protobuf import timestamp_pb2 as _timestamp_pb2
from google.protobuf.internal import containers as _containers
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Iterable as _Iterable, Mapping as _Mapping
from typing import ClassVar as _ClassVar, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class Assignment(_message.Message):
    __slots__ = ("id", "brand_id", "scenario_name", "brand_version", "elements", "applied_at")
    ID_FIELD_NUMBER: _ClassVar[int]
    BRAND_ID_FIELD_NUMBER: _ClassVar[int]
    SCENARIO_NAME_FIELD_NUMBER: _ClassVar[int]
    BRAND_VERSION_FIELD_NUMBER: _ClassVar[int]
    ELEMENTS_FIELD_NUMBER: _ClassVar[int]
    APPLIED_AT_FIELD_NUMBER: _ClassVar[int]
    id: str
    brand_id: str
    scenario_name: str
    brand_version: int
    elements: _containers.RepeatedScalarFieldContainer[str]
    applied_at: _timestamp_pb2.Timestamp
    def __init__(self, id: _Optional[str] = ..., brand_id: _Optional[str] = ..., scenario_name: _Optional[str] = ..., brand_version: _Optional[int] = ..., elements: _Optional[_Iterable[str]] = ..., applied_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ...) -> None: ...

class ScenarioStatus(_message.Message):
    __slots__ = ("scenario", "has_brand", "brand_id", "brand_version", "elements", "applied_at")
    SCENARIO_FIELD_NUMBER: _ClassVar[int]
    HAS_BRAND_FIELD_NUMBER: _ClassVar[int]
    BRAND_ID_FIELD_NUMBER: _ClassVar[int]
    BRAND_VERSION_FIELD_NUMBER: _ClassVar[int]
    ELEMENTS_FIELD_NUMBER: _ClassVar[int]
    APPLIED_AT_FIELD_NUMBER: _ClassVar[int]
    scenario: str
    has_brand: bool
    brand_id: str
    brand_version: int
    elements: _containers.RepeatedScalarFieldContainer[str]
    applied_at: _timestamp_pb2.Timestamp
    def __init__(self, scenario: _Optional[str] = ..., has_brand: _Optional[bool] = ..., brand_id: _Optional[str] = ..., brand_version: _Optional[int] = ..., elements: _Optional[_Iterable[str]] = ..., applied_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ...) -> None: ...

class ListAssignmentsRequest(_message.Message):
    __slots__ = ("brand_id",)
    BRAND_ID_FIELD_NUMBER: _ClassVar[int]
    brand_id: str
    def __init__(self, brand_id: _Optional[str] = ...) -> None: ...

class ListAssignmentsResponse(_message.Message):
    __slots__ = ("assignments",)
    ASSIGNMENTS_FIELD_NUMBER: _ClassVar[int]
    assignments: _containers.RepeatedCompositeFieldContainer[Assignment]
    def __init__(self, assignments: _Optional[_Iterable[_Union[Assignment, _Mapping]]] = ...) -> None: ...

class AssignBrandRequest(_message.Message):
    __slots__ = ("brand_id", "scenario_name", "elements")
    BRAND_ID_FIELD_NUMBER: _ClassVar[int]
    SCENARIO_NAME_FIELD_NUMBER: _ClassVar[int]
    ELEMENTS_FIELD_NUMBER: _ClassVar[int]
    brand_id: str
    scenario_name: str
    elements: _containers.RepeatedScalarFieldContainer[str]
    def __init__(self, brand_id: _Optional[str] = ..., scenario_name: _Optional[str] = ..., elements: _Optional[_Iterable[str]] = ...) -> None: ...

class AssignBrandResponse(_message.Message):
    __slots__ = ("assignment",)
    ASSIGNMENT_FIELD_NUMBER: _ClassVar[int]
    assignment: Assignment
    def __init__(self, assignment: _Optional[_Union[Assignment, _Mapping]] = ...) -> None: ...

class GetScenarioStatusRequest(_message.Message):
    __slots__ = ("scenario_name",)
    SCENARIO_NAME_FIELD_NUMBER: _ClassVar[int]
    scenario_name: str
    def __init__(self, scenario_name: _Optional[str] = ...) -> None: ...

class GetScenarioStatusResponse(_message.Message):
    __slots__ = ("status",)
    STATUS_FIELD_NUMBER: _ClassVar[int]
    status: ScenarioStatus
    def __init__(self, status: _Optional[_Union[ScenarioStatus, _Mapping]] = ...) -> None: ...

class UnassignScenarioRequest(_message.Message):
    __slots__ = ("scenario_name",)
    SCENARIO_NAME_FIELD_NUMBER: _ClassVar[int]
    scenario_name: str
    def __init__(self, scenario_name: _Optional[str] = ...) -> None: ...

class UnassignScenarioResponse(_message.Message):
    __slots__ = ()
    def __init__(self) -> None: ...
