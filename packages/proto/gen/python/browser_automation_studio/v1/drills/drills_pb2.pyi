from google.protobuf.internal import containers as _containers
from google.protobuf.internal import enum_type_wrapper as _enum_type_wrapper
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Iterable as _Iterable, Mapping as _Mapping
from typing import ClassVar as _ClassVar, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class DrillName(int, metaclass=_enum_type_wrapper.EnumTypeWrapper):
    __slots__ = ()
    DRILL_NAME_UNSPECIFIED: _ClassVar[DrillName]
    DRILL_NAME_DRIVER_UNAVAILABLE: _ClassVar[DrillName]
    DRILL_NAME_PARTIAL_INITIALIZATION: _ClassVar[DrillName]
    DRILL_NAME_CAPACITY: _ClassVar[DrillName]
    DRILL_NAME_EXPIRY: _ClassVar[DrillName]
DRILL_NAME_UNSPECIFIED: DrillName
DRILL_NAME_DRIVER_UNAVAILABLE: DrillName
DRILL_NAME_PARTIAL_INITIALIZATION: DrillName
DRILL_NAME_CAPACITY: DrillName
DRILL_NAME_EXPIRY: DrillName

class ListDrillsRequest(_message.Message):
    __slots__ = ()
    def __init__(self) -> None: ...

class DrillDefinition(_message.Message):
    __slots__ = ("name", "description")
    NAME_FIELD_NUMBER: _ClassVar[int]
    DESCRIPTION_FIELD_NUMBER: _ClassVar[int]
    name: DrillName
    description: str
    def __init__(self, name: _Optional[_Union[DrillName, str]] = ..., description: _Optional[str] = ...) -> None: ...

class ListDrillsResponse(_message.Message):
    __slots__ = ("drills",)
    DRILLS_FIELD_NUMBER: _ClassVar[int]
    drills: _containers.RepeatedCompositeFieldContainer[DrillDefinition]
    def __init__(self, drills: _Optional[_Iterable[_Union[DrillDefinition, _Mapping]]] = ...) -> None: ...

class RunDrillRequest(_message.Message):
    __slots__ = ("name",)
    NAME_FIELD_NUMBER: _ClassVar[int]
    name: DrillName
    def __init__(self, name: _Optional[_Union[DrillName, str]] = ...) -> None: ...

class DrillAssertion(_message.Message):
    __slots__ = ("name", "passed", "detail")
    NAME_FIELD_NUMBER: _ClassVar[int]
    PASSED_FIELD_NUMBER: _ClassVar[int]
    DETAIL_FIELD_NUMBER: _ClassVar[int]
    name: str
    passed: bool
    detail: str
    def __init__(self, name: _Optional[str] = ..., passed: _Optional[bool] = ..., detail: _Optional[str] = ...) -> None: ...

class DrillVerdict(_message.Message):
    __slots__ = ("name", "passed", "expected_failure_observed", "cleanup_completed", "primary_outcome", "cleanup_outcome", "assertions", "evidence_json")
    NAME_FIELD_NUMBER: _ClassVar[int]
    PASSED_FIELD_NUMBER: _ClassVar[int]
    EXPECTED_FAILURE_OBSERVED_FIELD_NUMBER: _ClassVar[int]
    CLEANUP_COMPLETED_FIELD_NUMBER: _ClassVar[int]
    PRIMARY_OUTCOME_FIELD_NUMBER: _ClassVar[int]
    CLEANUP_OUTCOME_FIELD_NUMBER: _ClassVar[int]
    ASSERTIONS_FIELD_NUMBER: _ClassVar[int]
    EVIDENCE_JSON_FIELD_NUMBER: _ClassVar[int]
    name: DrillName
    passed: bool
    expected_failure_observed: bool
    cleanup_completed: bool
    primary_outcome: str
    cleanup_outcome: str
    assertions: _containers.RepeatedCompositeFieldContainer[DrillAssertion]
    evidence_json: str
    def __init__(self, name: _Optional[_Union[DrillName, str]] = ..., passed: _Optional[bool] = ..., expected_failure_observed: _Optional[bool] = ..., cleanup_completed: _Optional[bool] = ..., primary_outcome: _Optional[str] = ..., cleanup_outcome: _Optional[str] = ..., assertions: _Optional[_Iterable[_Union[DrillAssertion, _Mapping]]] = ..., evidence_json: _Optional[str] = ...) -> None: ...

class RunDrillResponse(_message.Message):
    __slots__ = ("verdict",)
    VERDICT_FIELD_NUMBER: _ClassVar[int]
    verdict: DrillVerdict
    def __init__(self, verdict: _Optional[_Union[DrillVerdict, _Mapping]] = ...) -> None: ...
