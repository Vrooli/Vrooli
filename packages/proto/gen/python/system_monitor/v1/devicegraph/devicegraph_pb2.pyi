import datetime

from google.protobuf import timestamp_pb2 as _timestamp_pb2
from google.protobuf.internal import containers as _containers
from google.protobuf.internal import enum_type_wrapper as _enum_type_wrapper
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Iterable as _Iterable, Mapping as _Mapping
from typing import ClassVar as _ClassVar, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class Rung(int, metaclass=_enum_type_wrapper.EnumTypeWrapper):
    __slots__ = ()
    RUNG_UNSPECIFIED: _ClassVar[Rung]
    RUNG_IDENTITY: _ClassVar[Rung]
    RUNG_TELEMETRY: _ClassVar[Rung]
    RUNG_EVIDENCE: _ClassVar[Rung]
    RUNG_CONTROL: _ClassVar[Rung]
    RUNG_ANTICIPATION: _ClassVar[Rung]

class RungGrade(int, metaclass=_enum_type_wrapper.EnumTypeWrapper):
    __slots__ = ()
    RUNG_GRADE_UNSPECIFIED: _ClassVar[RungGrade]
    RUNG_GRADE_MEASURED: _ClassVar[RungGrade]
    RUNG_GRADE_UNMEASURABLE: _ClassVar[RungGrade]
    RUNG_GRADE_UNAVAILABLE: _ClassVar[RungGrade]
    RUNG_GRADE_NOT_APPLICABLE: _ClassVar[RungGrade]
RUNG_UNSPECIFIED: Rung
RUNG_IDENTITY: Rung
RUNG_TELEMETRY: Rung
RUNG_EVIDENCE: Rung
RUNG_CONTROL: Rung
RUNG_ANTICIPATION: Rung
RUNG_GRADE_UNSPECIFIED: RungGrade
RUNG_GRADE_MEASURED: RungGrade
RUNG_GRADE_UNMEASURABLE: RungGrade
RUNG_GRADE_UNAVAILABLE: RungGrade
RUNG_GRADE_NOT_APPLICABLE: RungGrade

class RungState(_message.Message):
    __slots__ = ("rung", "grade", "reason", "mechanism", "remediation", "observed_at")
    RUNG_FIELD_NUMBER: _ClassVar[int]
    GRADE_FIELD_NUMBER: _ClassVar[int]
    REASON_FIELD_NUMBER: _ClassVar[int]
    MECHANISM_FIELD_NUMBER: _ClassVar[int]
    REMEDIATION_FIELD_NUMBER: _ClassVar[int]
    OBSERVED_AT_FIELD_NUMBER: _ClassVar[int]
    rung: Rung
    grade: RungGrade
    reason: str
    mechanism: str
    remediation: str
    observed_at: _timestamp_pb2.Timestamp
    def __init__(self, rung: _Optional[_Union[Rung, str]] = ..., grade: _Optional[_Union[RungGrade, str]] = ..., reason: _Optional[str] = ..., mechanism: _Optional[str] = ..., remediation: _Optional[str] = ..., observed_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ...) -> None: ...

class Device(_message.Message):
    __slots__ = ("id", "parent_id", "vendor", "model", "driver", "sys_path", "attributes", "readings", "rungs")
    class AttributesEntry(_message.Message):
        __slots__ = ("key", "value")
        KEY_FIELD_NUMBER: _ClassVar[int]
        VALUE_FIELD_NUMBER: _ClassVar[int]
        key: str
        value: str
        def __init__(self, key: _Optional[str] = ..., value: _Optional[str] = ...) -> None: ...
    class ReadingsEntry(_message.Message):
        __slots__ = ("key", "value")
        KEY_FIELD_NUMBER: _ClassVar[int]
        VALUE_FIELD_NUMBER: _ClassVar[int]
        key: str
        value: float
        def __init__(self, key: _Optional[str] = ..., value: _Optional[float] = ...) -> None: ...
    ID_FIELD_NUMBER: _ClassVar[int]
    CLASS_FIELD_NUMBER: _ClassVar[int]
    PARENT_ID_FIELD_NUMBER: _ClassVar[int]
    VENDOR_FIELD_NUMBER: _ClassVar[int]
    MODEL_FIELD_NUMBER: _ClassVar[int]
    DRIVER_FIELD_NUMBER: _ClassVar[int]
    SYS_PATH_FIELD_NUMBER: _ClassVar[int]
    ATTRIBUTES_FIELD_NUMBER: _ClassVar[int]
    READINGS_FIELD_NUMBER: _ClassVar[int]
    RUNGS_FIELD_NUMBER: _ClassVar[int]
    id: str
    parent_id: str
    vendor: str
    model: str
    driver: str
    sys_path: str
    attributes: _containers.ScalarMap[str, str]
    readings: _containers.ScalarMap[str, float]
    rungs: _containers.RepeatedCompositeFieldContainer[RungState]
    def __init__(self, id: _Optional[str] = ..., parent_id: _Optional[str] = ..., vendor: _Optional[str] = ..., model: _Optional[str] = ..., driver: _Optional[str] = ..., sys_path: _Optional[str] = ..., attributes: _Optional[_Mapping[str, str]] = ..., readings: _Optional[_Mapping[str, float]] = ..., rungs: _Optional[_Iterable[_Union[RungState, _Mapping]]] = ..., **kwargs) -> None: ...

class Subsystem(_message.Message):
    __slots__ = ("name", "attributes", "rungs")
    class AttributesEntry(_message.Message):
        __slots__ = ("key", "value")
        KEY_FIELD_NUMBER: _ClassVar[int]
        VALUE_FIELD_NUMBER: _ClassVar[int]
        key: str
        value: str
        def __init__(self, key: _Optional[str] = ..., value: _Optional[str] = ...) -> None: ...
    NAME_FIELD_NUMBER: _ClassVar[int]
    ATTRIBUTES_FIELD_NUMBER: _ClassVar[int]
    RUNGS_FIELD_NUMBER: _ClassVar[int]
    name: str
    attributes: _containers.ScalarMap[str, str]
    rungs: _containers.RepeatedCompositeFieldContainer[RungState]
    def __init__(self, name: _Optional[str] = ..., attributes: _Optional[_Mapping[str, str]] = ..., rungs: _Optional[_Iterable[_Union[RungState, _Mapping]]] = ...) -> None: ...

class Graph(_message.Message):
    __slots__ = ("collected_at", "platform", "devices", "subsystems", "virtual_network_interfaces", "warnings", "available", "unavailable_reason")
    COLLECTED_AT_FIELD_NUMBER: _ClassVar[int]
    PLATFORM_FIELD_NUMBER: _ClassVar[int]
    DEVICES_FIELD_NUMBER: _ClassVar[int]
    SUBSYSTEMS_FIELD_NUMBER: _ClassVar[int]
    VIRTUAL_NETWORK_INTERFACES_FIELD_NUMBER: _ClassVar[int]
    WARNINGS_FIELD_NUMBER: _ClassVar[int]
    AVAILABLE_FIELD_NUMBER: _ClassVar[int]
    UNAVAILABLE_REASON_FIELD_NUMBER: _ClassVar[int]
    collected_at: _timestamp_pb2.Timestamp
    platform: str
    devices: _containers.RepeatedCompositeFieldContainer[Device]
    subsystems: _containers.RepeatedCompositeFieldContainer[Subsystem]
    virtual_network_interfaces: _containers.RepeatedScalarFieldContainer[str]
    warnings: _containers.RepeatedScalarFieldContainer[str]
    available: bool
    unavailable_reason: str
    def __init__(self, collected_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., platform: _Optional[str] = ..., devices: _Optional[_Iterable[_Union[Device, _Mapping]]] = ..., subsystems: _Optional[_Iterable[_Union[Subsystem, _Mapping]]] = ..., virtual_network_interfaces: _Optional[_Iterable[str]] = ..., warnings: _Optional[_Iterable[str]] = ..., available: _Optional[bool] = ..., unavailable_reason: _Optional[str] = ...) -> None: ...

class GetDeviceGraphRequest(_message.Message):
    __slots__ = ()
    def __init__(self) -> None: ...

class GetDeviceGraphResponse(_message.Message):
    __slots__ = ("graph",)
    GRAPH_FIELD_NUMBER: _ClassVar[int]
    graph: Graph
    def __init__(self, graph: _Optional[_Union[Graph, _Mapping]] = ...) -> None: ...
