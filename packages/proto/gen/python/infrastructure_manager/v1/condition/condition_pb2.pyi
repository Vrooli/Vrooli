import datetime

from google.protobuf import timestamp_pb2 as _timestamp_pb2
from infrastructure_manager.v1.coverage import coverage_pb2 as _coverage_pb2
from google.protobuf.internal import containers as _containers
from google.protobuf.internal import enum_type_wrapper as _enum_type_wrapper
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Iterable as _Iterable, Mapping as _Mapping
from typing import ClassVar as _ClassVar, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class TrustVerdict(int, metaclass=_enum_type_wrapper.EnumTypeWrapper):
    __slots__ = ()
    TRUST_VERDICT_UNSPECIFIED: _ClassVar[TrustVerdict]
    TRUST_VERDICT_VALID: _ClassVar[TrustVerdict]
    TRUST_VERDICT_GHOST: _ClassVar[TrustVerdict]
    TRUST_VERDICT_SATURATED: _ClassVar[TrustVerdict]
    TRUST_VERDICT_SHELVED: _ClassVar[TrustVerdict]
    TRUST_VERDICT_UNIT_MISMATCH: _ClassVar[TrustVerdict]
    TRUST_VERDICT_UNAVAILABLE: _ClassVar[TrustVerdict]
    TRUST_VERDICT_UNTRUSTED: _ClassVar[TrustVerdict]

class BandVerdict(int, metaclass=_enum_type_wrapper.EnumTypeWrapper):
    __slots__ = ()
    BAND_VERDICT_UNSPECIFIED: _ClassVar[BandVerdict]
    BAND_VERDICT_IN_BAND: _ClassVar[BandVerdict]
    BAND_VERDICT_OUT_OF_BAND: _ClassVar[BandVerdict]
    BAND_VERDICT_PENDING_SUSTAIN: _ClassVar[BandVerdict]
    BAND_VERDICT_NEEDS_BASELINE: _ClassVar[BandVerdict]
    BAND_VERDICT_NOT_EVALUATED: _ClassVar[BandVerdict]
TRUST_VERDICT_UNSPECIFIED: TrustVerdict
TRUST_VERDICT_VALID: TrustVerdict
TRUST_VERDICT_GHOST: TrustVerdict
TRUST_VERDICT_SATURATED: TrustVerdict
TRUST_VERDICT_SHELVED: TrustVerdict
TRUST_VERDICT_UNIT_MISMATCH: TrustVerdict
TRUST_VERDICT_UNAVAILABLE: TrustVerdict
TRUST_VERDICT_UNTRUSTED: TrustVerdict
BAND_VERDICT_UNSPECIFIED: BandVerdict
BAND_VERDICT_IN_BAND: BandVerdict
BAND_VERDICT_OUT_OF_BAND: BandVerdict
BAND_VERDICT_PENDING_SUSTAIN: BandVerdict
BAND_VERDICT_NEEDS_BASELINE: BandVerdict
BAND_VERDICT_NOT_EVALUATED: BandVerdict

class Leg(_message.Message):
    __slots__ = ("cell_ref", "projection", "owner", "unit", "source")
    CELL_REF_FIELD_NUMBER: _ClassVar[int]
    PROJECTION_FIELD_NUMBER: _ClassVar[int]
    OWNER_FIELD_NUMBER: _ClassVar[int]
    UNIT_FIELD_NUMBER: _ClassVar[int]
    SOURCE_FIELD_NUMBER: _ClassVar[int]
    cell_ref: str
    projection: _coverage_pb2.Projection
    owner: str
    unit: str
    source: str
    def __init__(self, cell_ref: _Optional[str] = ..., projection: _Optional[_Union[_coverage_pb2.Projection, str]] = ..., owner: _Optional[str] = ..., unit: _Optional[str] = ..., source: _Optional[str] = ...) -> None: ...

class Reading(_message.Message):
    __slots__ = ("id", "cell_ref", "value", "unit", "source", "observed_at", "trust_verdict", "band_verdict", "unavailable_reason")
    ID_FIELD_NUMBER: _ClassVar[int]
    CELL_REF_FIELD_NUMBER: _ClassVar[int]
    VALUE_FIELD_NUMBER: _ClassVar[int]
    UNIT_FIELD_NUMBER: _ClassVar[int]
    SOURCE_FIELD_NUMBER: _ClassVar[int]
    OBSERVED_AT_FIELD_NUMBER: _ClassVar[int]
    TRUST_VERDICT_FIELD_NUMBER: _ClassVar[int]
    BAND_VERDICT_FIELD_NUMBER: _ClassVar[int]
    UNAVAILABLE_REASON_FIELD_NUMBER: _ClassVar[int]
    id: str
    cell_ref: str
    value: float
    unit: str
    source: str
    observed_at: _timestamp_pb2.Timestamp
    trust_verdict: TrustVerdict
    band_verdict: BandVerdict
    unavailable_reason: str
    def __init__(self, id: _Optional[str] = ..., cell_ref: _Optional[str] = ..., value: _Optional[float] = ..., unit: _Optional[str] = ..., source: _Optional[str] = ..., observed_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., trust_verdict: _Optional[_Union[TrustVerdict, str]] = ..., band_verdict: _Optional[_Union[BandVerdict, str]] = ..., unavailable_reason: _Optional[str] = ...) -> None: ...

class TrustCount(_message.Message):
    __slots__ = ("verdict", "count")
    VERDICT_FIELD_NUMBER: _ClassVar[int]
    COUNT_FIELD_NUMBER: _ClassVar[int]
    verdict: TrustVerdict
    count: int
    def __init__(self, verdict: _Optional[_Union[TrustVerdict, str]] = ..., count: _Optional[int] = ...) -> None: ...

class TrustTriple(_message.Message):
    __slots__ = ("distribution", "checked_denominator", "total", "checked_at")
    DISTRIBUTION_FIELD_NUMBER: _ClassVar[int]
    CHECKED_DENOMINATOR_FIELD_NUMBER: _ClassVar[int]
    TOTAL_FIELD_NUMBER: _ClassVar[int]
    CHECKED_AT_FIELD_NUMBER: _ClassVar[int]
    distribution: _containers.RepeatedCompositeFieldContainer[TrustCount]
    checked_denominator: int
    total: int
    checked_at: _timestamp_pb2.Timestamp
    def __init__(self, distribution: _Optional[_Iterable[_Union[TrustCount, _Mapping]]] = ..., checked_denominator: _Optional[int] = ..., total: _Optional[int] = ..., checked_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ...) -> None: ...

class SourceAvailability(_message.Message):
    __slots__ = ("source", "available", "reason", "checked_at")
    SOURCE_FIELD_NUMBER: _ClassVar[int]
    AVAILABLE_FIELD_NUMBER: _ClassVar[int]
    REASON_FIELD_NUMBER: _ClassVar[int]
    CHECKED_AT_FIELD_NUMBER: _ClassVar[int]
    source: str
    available: bool
    reason: str
    checked_at: _timestamp_pb2.Timestamp
    def __init__(self, source: _Optional[str] = ..., available: _Optional[bool] = ..., reason: _Optional[str] = ..., checked_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ...) -> None: ...

class GetConditionRequest(_message.Message):
    __slots__ = ("projection", "cell_ref")
    PROJECTION_FIELD_NUMBER: _ClassVar[int]
    CELL_REF_FIELD_NUMBER: _ClassVar[int]
    projection: _coverage_pb2.Projection
    cell_ref: str
    def __init__(self, projection: _Optional[_Union[_coverage_pb2.Projection, str]] = ..., cell_ref: _Optional[str] = ...) -> None: ...

class GetConditionResponse(_message.Message):
    __slots__ = ("readings", "legs", "sources", "computed_at")
    READINGS_FIELD_NUMBER: _ClassVar[int]
    LEGS_FIELD_NUMBER: _ClassVar[int]
    SOURCES_FIELD_NUMBER: _ClassVar[int]
    COMPUTED_AT_FIELD_NUMBER: _ClassVar[int]
    readings: _containers.RepeatedCompositeFieldContainer[Reading]
    legs: _containers.RepeatedCompositeFieldContainer[Leg]
    sources: _containers.RepeatedCompositeFieldContainer[SourceAvailability]
    computed_at: _timestamp_pb2.Timestamp
    def __init__(self, readings: _Optional[_Iterable[_Union[Reading, _Mapping]]] = ..., legs: _Optional[_Iterable[_Union[Leg, _Mapping]]] = ..., sources: _Optional[_Iterable[_Union[SourceAvailability, _Mapping]]] = ..., computed_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ...) -> None: ...

class GetTrustDistributionRequest(_message.Message):
    __slots__ = ("projection",)
    PROJECTION_FIELD_NUMBER: _ClassVar[int]
    projection: _coverage_pb2.Projection
    def __init__(self, projection: _Optional[_Union[_coverage_pb2.Projection, str]] = ...) -> None: ...

class GetTrustDistributionResponse(_message.Message):
    __slots__ = ("trust",)
    TRUST_FIELD_NUMBER: _ClassVar[int]
    trust: TrustTriple
    def __init__(self, trust: _Optional[_Union[TrustTriple, _Mapping]] = ...) -> None: ...

class ExplainCellRequest(_message.Message):
    __slots__ = ("cell_ref",)
    CELL_REF_FIELD_NUMBER: _ClassVar[int]
    cell_ref: str
    def __init__(self, cell_ref: _Optional[str] = ...) -> None: ...

class ExplainCellResponse(_message.Message):
    __slots__ = ("cell", "reading", "sources")
    CELL_FIELD_NUMBER: _ClassVar[int]
    READING_FIELD_NUMBER: _ClassVar[int]
    SOURCES_FIELD_NUMBER: _ClassVar[int]
    cell: _coverage_pb2.Cell
    reading: Reading
    sources: _containers.RepeatedCompositeFieldContainer[SourceAvailability]
    def __init__(self, cell: _Optional[_Union[_coverage_pb2.Cell, _Mapping]] = ..., reading: _Optional[_Union[Reading, _Mapping]] = ..., sources: _Optional[_Iterable[_Union[SourceAvailability, _Mapping]]] = ...) -> None: ...

class GetHistoryRequest(_message.Message):
    __slots__ = ("cell_ref", "limit")
    CELL_REF_FIELD_NUMBER: _ClassVar[int]
    LIMIT_FIELD_NUMBER: _ClassVar[int]
    cell_ref: str
    limit: int
    def __init__(self, cell_ref: _Optional[str] = ..., limit: _Optional[int] = ...) -> None: ...

class GetHistoryResponse(_message.Message):
    __slots__ = ("readings", "measurable", "unmeasurable_reason")
    READINGS_FIELD_NUMBER: _ClassVar[int]
    MEASURABLE_FIELD_NUMBER: _ClassVar[int]
    UNMEASURABLE_REASON_FIELD_NUMBER: _ClassVar[int]
    readings: _containers.RepeatedCompositeFieldContainer[Reading]
    measurable: bool
    unmeasurable_reason: str
    def __init__(self, readings: _Optional[_Iterable[_Union[Reading, _Mapping]]] = ..., measurable: _Optional[bool] = ..., unmeasurable_reason: _Optional[str] = ...) -> None: ...
