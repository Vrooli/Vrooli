import datetime

from google.protobuf import timestamp_pb2 as _timestamp_pb2
from infrastructure_manager.v1.coverage import coverage_pb2 as _coverage_pb2
from infrastructure_manager.v1.condition import condition_pb2 as _condition_pb2
from google.protobuf.internal import containers as _containers
from google.protobuf.internal import enum_type_wrapper as _enum_type_wrapper
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Iterable as _Iterable, Mapping as _Mapping
from typing import ClassVar as _ClassVar, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class EfficacyVerdict(int, metaclass=_enum_type_wrapper.EnumTypeWrapper):
    __slots__ = ()
    EFFICACY_VERDICT_UNSPECIFIED: _ClassVar[EfficacyVerdict]
    EFFICACY_VERDICT_MOVED: _ClassVar[EfficacyVerdict]
    EFFICACY_VERDICT_DID_NOT_MOVE: _ClassVar[EfficacyVerdict]
    EFFICACY_VERDICT_AWAITING_WORK: _ClassVar[EfficacyVerdict]
    EFFICACY_VERDICT_UNMEASURABLE: _ClassVar[EfficacyVerdict]
EFFICACY_VERDICT_UNSPECIFIED: EfficacyVerdict
EFFICACY_VERDICT_MOVED: EfficacyVerdict
EFFICACY_VERDICT_DID_NOT_MOVE: EfficacyVerdict
EFFICACY_VERDICT_AWAITING_WORK: EfficacyVerdict
EFFICACY_VERDICT_UNMEASURABLE: EfficacyVerdict

class GapSource(_message.Message):
    __slots__ = ("id", "label", "available", "reason", "finding_count")
    ID_FIELD_NUMBER: _ClassVar[int]
    LABEL_FIELD_NUMBER: _ClassVar[int]
    AVAILABLE_FIELD_NUMBER: _ClassVar[int]
    REASON_FIELD_NUMBER: _ClassVar[int]
    FINDING_COUNT_FIELD_NUMBER: _ClassVar[int]
    id: str
    label: str
    available: bool
    reason: str
    finding_count: int
    def __init__(self, id: _Optional[str] = ..., label: _Optional[str] = ..., available: _Optional[bool] = ..., reason: _Optional[str] = ..., finding_count: _Optional[int] = ...) -> None: ...

class RankingRationale(_message.Message):
    __slots__ = ("rank", "cascade_stage", "explanation")
    RANK_FIELD_NUMBER: _ClassVar[int]
    CASCADE_STAGE_FIELD_NUMBER: _ClassVar[int]
    EXPLANATION_FIELD_NUMBER: _ClassVar[int]
    rank: int
    cascade_stage: str
    explanation: str
    def __init__(self, rank: _Optional[int] = ..., cascade_stage: _Optional[str] = ..., explanation: _Optional[str] = ...) -> None: ...

class Finding(_message.Message):
    __slots__ = ("id", "source", "cell_ref", "title", "message", "sensor_ref", "expected_return", "trust_verdict", "band_verdict", "rationale", "observed_at")
    ID_FIELD_NUMBER: _ClassVar[int]
    SOURCE_FIELD_NUMBER: _ClassVar[int]
    CELL_REF_FIELD_NUMBER: _ClassVar[int]
    TITLE_FIELD_NUMBER: _ClassVar[int]
    MESSAGE_FIELD_NUMBER: _ClassVar[int]
    SENSOR_REF_FIELD_NUMBER: _ClassVar[int]
    EXPECTED_RETURN_FIELD_NUMBER: _ClassVar[int]
    TRUST_VERDICT_FIELD_NUMBER: _ClassVar[int]
    BAND_VERDICT_FIELD_NUMBER: _ClassVar[int]
    RATIONALE_FIELD_NUMBER: _ClassVar[int]
    OBSERVED_AT_FIELD_NUMBER: _ClassVar[int]
    id: str
    source: str
    cell_ref: str
    title: str
    message: str
    sensor_ref: str
    expected_return: str
    trust_verdict: _condition_pb2.TrustVerdict
    band_verdict: _condition_pb2.BandVerdict
    rationale: RankingRationale
    observed_at: _timestamp_pb2.Timestamp
    def __init__(self, id: _Optional[str] = ..., source: _Optional[str] = ..., cell_ref: _Optional[str] = ..., title: _Optional[str] = ..., message: _Optional[str] = ..., sensor_ref: _Optional[str] = ..., expected_return: _Optional[str] = ..., trust_verdict: _Optional[_Union[_condition_pb2.TrustVerdict, str]] = ..., band_verdict: _Optional[_Union[_condition_pb2.BandVerdict, str]] = ..., rationale: _Optional[_Union[RankingRationale, _Mapping]] = ..., observed_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ...) -> None: ...

class EfficacyRecord(_message.Message):
    __slots__ = ("finding_id", "sensor_ref", "expected_return", "observed_return", "verdict", "observed_at")
    FINDING_ID_FIELD_NUMBER: _ClassVar[int]
    SENSOR_REF_FIELD_NUMBER: _ClassVar[int]
    EXPECTED_RETURN_FIELD_NUMBER: _ClassVar[int]
    OBSERVED_RETURN_FIELD_NUMBER: _ClassVar[int]
    VERDICT_FIELD_NUMBER: _ClassVar[int]
    OBSERVED_AT_FIELD_NUMBER: _ClassVar[int]
    finding_id: str
    sensor_ref: str
    expected_return: str
    observed_return: str
    verdict: EfficacyVerdict
    observed_at: _timestamp_pb2.Timestamp
    def __init__(self, finding_id: _Optional[str] = ..., sensor_ref: _Optional[str] = ..., expected_return: _Optional[str] = ..., observed_return: _Optional[str] = ..., verdict: _Optional[_Union[EfficacyVerdict, str]] = ..., observed_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ...) -> None: ...

class GetNextRequest(_message.Message):
    __slots__ = ("limit",)
    LIMIT_FIELD_NUMBER: _ClassVar[int]
    limit: int
    def __init__(self, limit: _Optional[int] = ...) -> None: ...

class GetNextResponse(_message.Message):
    __slots__ = ("findings", "sources", "no_findings", "all_sources_unavailable", "computed_at")
    FINDINGS_FIELD_NUMBER: _ClassVar[int]
    SOURCES_FIELD_NUMBER: _ClassVar[int]
    NO_FINDINGS_FIELD_NUMBER: _ClassVar[int]
    ALL_SOURCES_UNAVAILABLE_FIELD_NUMBER: _ClassVar[int]
    COMPUTED_AT_FIELD_NUMBER: _ClassVar[int]
    findings: _containers.RepeatedCompositeFieldContainer[Finding]
    sources: _containers.RepeatedCompositeFieldContainer[GapSource]
    no_findings: bool
    all_sources_unavailable: bool
    computed_at: _timestamp_pb2.Timestamp
    def __init__(self, findings: _Optional[_Iterable[_Union[Finding, _Mapping]]] = ..., sources: _Optional[_Iterable[_Union[GapSource, _Mapping]]] = ..., no_findings: _Optional[bool] = ..., all_sources_unavailable: _Optional[bool] = ..., computed_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ...) -> None: ...

class GetFindingRequest(_message.Message):
    __slots__ = ("finding_id",)
    FINDING_ID_FIELD_NUMBER: _ClassVar[int]
    finding_id: str
    def __init__(self, finding_id: _Optional[str] = ...) -> None: ...

class GetFindingResponse(_message.Message):
    __slots__ = ("finding",)
    FINDING_FIELD_NUMBER: _ClassVar[int]
    finding: Finding
    def __init__(self, finding: _Optional[_Union[Finding, _Mapping]] = ...) -> None: ...

class ListSourcesRequest(_message.Message):
    __slots__ = ("projection",)
    PROJECTION_FIELD_NUMBER: _ClassVar[int]
    projection: _coverage_pb2.Projection
    def __init__(self, projection: _Optional[_Union[_coverage_pb2.Projection, str]] = ...) -> None: ...

class ListSourcesResponse(_message.Message):
    __slots__ = ("sources",)
    SOURCES_FIELD_NUMBER: _ClassVar[int]
    sources: _containers.RepeatedCompositeFieldContainer[GapSource]
    def __init__(self, sources: _Optional[_Iterable[_Union[GapSource, _Mapping]]] = ...) -> None: ...

class GetEfficacyRequest(_message.Message):
    __slots__ = ("finding_id",)
    FINDING_ID_FIELD_NUMBER: _ClassVar[int]
    finding_id: str
    def __init__(self, finding_id: _Optional[str] = ...) -> None: ...

class GetEfficacyResponse(_message.Message):
    __slots__ = ("records",)
    RECORDS_FIELD_NUMBER: _ClassVar[int]
    records: _containers.RepeatedCompositeFieldContainer[EfficacyRecord]
    def __init__(self, records: _Optional[_Iterable[_Union[EfficacyRecord, _Mapping]]] = ...) -> None: ...
