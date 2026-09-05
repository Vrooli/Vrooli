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

class Observation(int, metaclass=_enum_type_wrapper.EnumTypeWrapper):
    __slots__ = ()
    OBSERVATION_UNSPECIFIED: _ClassVar[Observation]
    OBSERVATION_MEASURED: _ClassVar[Observation]
    OBSERVATION_UNMEASURABLE: _ClassVar[Observation]
    OBSERVATION_UNAVAILABLE: _ClassVar[Observation]
    OBSERVATION_NOT_APPLICABLE: _ClassVar[Observation]
    OBSERVATION_BLOCKED: _ClassVar[Observation]
    OBSERVATION_UNREAD: _ClassVar[Observation]

class CellStatus(int, metaclass=_enum_type_wrapper.EnumTypeWrapper):
    __slots__ = ()
    CELL_STATUS_UNSPECIFIED: _ClassVar[CellStatus]
    CELL_STATUS_NOW: _ClassVar[CellStatus]
    CELL_STATUS_IN_REACH: _ClassVar[CellStatus]
    CELL_STATUS_MISSING: _ClassVar[CellStatus]
    CELL_STATUS_UNAUTHORED: _ClassVar[CellStatus]

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
    BAND_VERDICT_NOT_GRADEABLE: _ClassVar[BandVerdict]

class CascadeStage(int, metaclass=_enum_type_wrapper.EnumTypeWrapper):
    __slots__ = ()
    CASCADE_STAGE_UNSPECIFIED: _ClassVar[CascadeStage]
    CASCADE_STAGE_SENSOR_CHANNEL_INTEGRITY: _ClassVar[CascadeStage]
    CASCADE_STAGE_HOST_SUBSTRATE: _ClassVar[CascadeStage]
    CASCADE_STAGE_CAPABILITY_AVAILABILITY: _ClassVar[CascadeStage]
    CASCADE_STAGE_EFFICIENCY: _ClassVar[CascadeStage]
    CASCADE_STAGE_MEASUREMENT_IMPROVEMENT: _ClassVar[CascadeStage]

class ConfidenceLevel(int, metaclass=_enum_type_wrapper.EnumTypeWrapper):
    __slots__ = ()
    CONFIDENCE_LEVEL_UNSPECIFIED: _ClassVar[ConfidenceLevel]
    CONFIDENCE_LEVEL_AUTHORITATIVE: _ClassVar[ConfidenceLevel]
    CONFIDENCE_LEVEL_PARTIAL: _ClassVar[ConfidenceLevel]
    CONFIDENCE_LEVEL_SKETCH: _ClassVar[ConfidenceLevel]
RUNG_UNSPECIFIED: Rung
RUNG_IDENTITY: Rung
RUNG_TELEMETRY: Rung
RUNG_EVIDENCE: Rung
RUNG_CONTROL: Rung
RUNG_ANTICIPATION: Rung
OBSERVATION_UNSPECIFIED: Observation
OBSERVATION_MEASURED: Observation
OBSERVATION_UNMEASURABLE: Observation
OBSERVATION_UNAVAILABLE: Observation
OBSERVATION_NOT_APPLICABLE: Observation
OBSERVATION_BLOCKED: Observation
OBSERVATION_UNREAD: Observation
CELL_STATUS_UNSPECIFIED: CellStatus
CELL_STATUS_NOW: CellStatus
CELL_STATUS_IN_REACH: CellStatus
CELL_STATUS_MISSING: CellStatus
CELL_STATUS_UNAUTHORED: CellStatus
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
BAND_VERDICT_NOT_GRADEABLE: BandVerdict
CASCADE_STAGE_UNSPECIFIED: CascadeStage
CASCADE_STAGE_SENSOR_CHANNEL_INTEGRITY: CascadeStage
CASCADE_STAGE_HOST_SUBSTRATE: CascadeStage
CASCADE_STAGE_CAPABILITY_AVAILABILITY: CascadeStage
CASCADE_STAGE_EFFICIENCY: CascadeStage
CASCADE_STAGE_MEASUREMENT_IMPROVEMENT: CascadeStage
CONFIDENCE_LEVEL_UNSPECIFIED: ConfidenceLevel
CONFIDENCE_LEVEL_AUTHORITATIVE: ConfidenceLevel
CONFIDENCE_LEVEL_PARTIAL: ConfidenceLevel
CONFIDENCE_LEVEL_SKETCH: ConfidenceLevel

class LadderCell(_message.Message):
    __slots__ = ("device_class", "rung", "host_os", "key", "cell_ref", "question", "status", "status_source", "observation", "reason", "reason_code", "mechanism", "remediation", "blocked_by", "trust", "unavailable_reason", "device_count", "blind_devices", "bar_id", "graded", "ungraded_reason", "band", "provisional", "capability", "capability_status", "capability_reason", "observed_at", "fault_unit", "fault_count", "fault_counted", "severity", "severity_known", "gap_opened_on", "gap_open_days", "gap_dated")
    DEVICE_CLASS_FIELD_NUMBER: _ClassVar[int]
    RUNG_FIELD_NUMBER: _ClassVar[int]
    HOST_OS_FIELD_NUMBER: _ClassVar[int]
    KEY_FIELD_NUMBER: _ClassVar[int]
    CELL_REF_FIELD_NUMBER: _ClassVar[int]
    QUESTION_FIELD_NUMBER: _ClassVar[int]
    STATUS_FIELD_NUMBER: _ClassVar[int]
    STATUS_SOURCE_FIELD_NUMBER: _ClassVar[int]
    OBSERVATION_FIELD_NUMBER: _ClassVar[int]
    REASON_FIELD_NUMBER: _ClassVar[int]
    REASON_CODE_FIELD_NUMBER: _ClassVar[int]
    MECHANISM_FIELD_NUMBER: _ClassVar[int]
    REMEDIATION_FIELD_NUMBER: _ClassVar[int]
    BLOCKED_BY_FIELD_NUMBER: _ClassVar[int]
    TRUST_FIELD_NUMBER: _ClassVar[int]
    UNAVAILABLE_REASON_FIELD_NUMBER: _ClassVar[int]
    DEVICE_COUNT_FIELD_NUMBER: _ClassVar[int]
    BLIND_DEVICES_FIELD_NUMBER: _ClassVar[int]
    BAR_ID_FIELD_NUMBER: _ClassVar[int]
    GRADED_FIELD_NUMBER: _ClassVar[int]
    UNGRADED_REASON_FIELD_NUMBER: _ClassVar[int]
    BAND_FIELD_NUMBER: _ClassVar[int]
    PROVISIONAL_FIELD_NUMBER: _ClassVar[int]
    CAPABILITY_FIELD_NUMBER: _ClassVar[int]
    CAPABILITY_STATUS_FIELD_NUMBER: _ClassVar[int]
    CAPABILITY_REASON_FIELD_NUMBER: _ClassVar[int]
    OBSERVED_AT_FIELD_NUMBER: _ClassVar[int]
    FAULT_UNIT_FIELD_NUMBER: _ClassVar[int]
    FAULT_COUNT_FIELD_NUMBER: _ClassVar[int]
    FAULT_COUNTED_FIELD_NUMBER: _ClassVar[int]
    SEVERITY_FIELD_NUMBER: _ClassVar[int]
    SEVERITY_KNOWN_FIELD_NUMBER: _ClassVar[int]
    GAP_OPENED_ON_FIELD_NUMBER: _ClassVar[int]
    GAP_OPEN_DAYS_FIELD_NUMBER: _ClassVar[int]
    GAP_DATED_FIELD_NUMBER: _ClassVar[int]
    device_class: str
    rung: Rung
    host_os: str
    key: str
    cell_ref: str
    question: str
    status: CellStatus
    status_source: str
    observation: Observation
    reason: str
    reason_code: str
    mechanism: str
    remediation: str
    blocked_by: Rung
    trust: TrustVerdict
    unavailable_reason: str
    device_count: int
    blind_devices: int
    bar_id: str
    graded: bool
    ungraded_reason: str
    band: BandVerdict
    provisional: bool
    capability: str
    capability_status: str
    capability_reason: str
    observed_at: _timestamp_pb2.Timestamp
    fault_unit: str
    fault_count: float
    fault_counted: bool
    severity: int
    severity_known: bool
    gap_opened_on: str
    gap_open_days: int
    gap_dated: bool
    def __init__(self, device_class: _Optional[str] = ..., rung: _Optional[_Union[Rung, str]] = ..., host_os: _Optional[str] = ..., key: _Optional[str] = ..., cell_ref: _Optional[str] = ..., question: _Optional[str] = ..., status: _Optional[_Union[CellStatus, str]] = ..., status_source: _Optional[str] = ..., observation: _Optional[_Union[Observation, str]] = ..., reason: _Optional[str] = ..., reason_code: _Optional[str] = ..., mechanism: _Optional[str] = ..., remediation: _Optional[str] = ..., blocked_by: _Optional[_Union[Rung, str]] = ..., trust: _Optional[_Union[TrustVerdict, str]] = ..., unavailable_reason: _Optional[str] = ..., device_count: _Optional[int] = ..., blind_devices: _Optional[int] = ..., bar_id: _Optional[str] = ..., graded: _Optional[bool] = ..., ungraded_reason: _Optional[str] = ..., band: _Optional[_Union[BandVerdict, str]] = ..., provisional: _Optional[bool] = ..., capability: _Optional[str] = ..., capability_status: _Optional[str] = ..., capability_reason: _Optional[str] = ..., observed_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., fault_unit: _Optional[str] = ..., fault_count: _Optional[float] = ..., fault_counted: _Optional[bool] = ..., severity: _Optional[int] = ..., severity_known: _Optional[bool] = ..., gap_opened_on: _Optional[str] = ..., gap_open_days: _Optional[int] = ..., gap_dated: _Optional[bool] = ...) -> None: ...

class CheckPlatformCoverage(_message.Message):
    __slots__ = ("host_os", "applicable", "total", "universal", "available", "reason")
    HOST_OS_FIELD_NUMBER: _ClassVar[int]
    APPLICABLE_FIELD_NUMBER: _ClassVar[int]
    TOTAL_FIELD_NUMBER: _ClassVar[int]
    UNIVERSAL_FIELD_NUMBER: _ClassVar[int]
    AVAILABLE_FIELD_NUMBER: _ClassVar[int]
    REASON_FIELD_NUMBER: _ClassVar[int]
    host_os: str
    applicable: int
    total: int
    universal: int
    available: bool
    reason: str
    def __init__(self, host_os: _Optional[str] = ..., applicable: _Optional[int] = ..., total: _Optional[int] = ..., universal: _Optional[int] = ..., available: _Optional[bool] = ..., reason: _Optional[str] = ...) -> None: ...

class SourceState(_message.Message):
    __slots__ = ("id", "available", "reason", "checked_at", "trust")
    ID_FIELD_NUMBER: _ClassVar[int]
    AVAILABLE_FIELD_NUMBER: _ClassVar[int]
    REASON_FIELD_NUMBER: _ClassVar[int]
    CHECKED_AT_FIELD_NUMBER: _ClassVar[int]
    TRUST_FIELD_NUMBER: _ClassVar[int]
    id: str
    available: bool
    reason: str
    checked_at: _timestamp_pb2.Timestamp
    trust: TrustVerdict
    def __init__(self, id: _Optional[str] = ..., available: _Optional[bool] = ..., reason: _Optional[str] = ..., checked_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., trust: _Optional[_Union[TrustVerdict, str]] = ...) -> None: ...

class Confidence(_message.Message):
    __slots__ = ("level", "rationale", "available", "reason")
    LEVEL_FIELD_NUMBER: _ClassVar[int]
    RATIONALE_FIELD_NUMBER: _ClassVar[int]
    AVAILABLE_FIELD_NUMBER: _ClassVar[int]
    REASON_FIELD_NUMBER: _ClassVar[int]
    level: ConfidenceLevel
    rationale: str
    available: bool
    reason: str
    def __init__(self, level: _Optional[_Union[ConfidenceLevel, str]] = ..., rationale: _Optional[str] = ..., available: _Optional[bool] = ..., reason: _Optional[str] = ...) -> None: ...

class DeviceRung(_message.Message):
    __slots__ = ("rung", "observation", "ladder_observation", "reason", "mechanism", "remediation", "blocked_by")
    RUNG_FIELD_NUMBER: _ClassVar[int]
    OBSERVATION_FIELD_NUMBER: _ClassVar[int]
    LADDER_OBSERVATION_FIELD_NUMBER: _ClassVar[int]
    REASON_FIELD_NUMBER: _ClassVar[int]
    MECHANISM_FIELD_NUMBER: _ClassVar[int]
    REMEDIATION_FIELD_NUMBER: _ClassVar[int]
    BLOCKED_BY_FIELD_NUMBER: _ClassVar[int]
    rung: Rung
    observation: Observation
    ladder_observation: Observation
    reason: str
    mechanism: str
    remediation: str
    blocked_by: Rung
    def __init__(self, rung: _Optional[_Union[Rung, str]] = ..., observation: _Optional[_Union[Observation, str]] = ..., ladder_observation: _Optional[_Union[Observation, str]] = ..., reason: _Optional[str] = ..., mechanism: _Optional[str] = ..., remediation: _Optional[str] = ..., blocked_by: _Optional[_Union[Rung, str]] = ...) -> None: ...

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
    rungs: _containers.RepeatedCompositeFieldContainer[DeviceRung]
    def __init__(self, id: _Optional[str] = ..., parent_id: _Optional[str] = ..., vendor: _Optional[str] = ..., model: _Optional[str] = ..., driver: _Optional[str] = ..., sys_path: _Optional[str] = ..., attributes: _Optional[_Mapping[str, str]] = ..., readings: _Optional[_Mapping[str, float]] = ..., rungs: _Optional[_Iterable[_Union[DeviceRung, _Mapping]]] = ..., **kwargs) -> None: ...

class RankedFinding(_message.Message):
    __slots__ = ("rank", "id", "source", "cell_ref", "sensor_ref", "title", "message", "stage", "stage_explanation", "severity", "trust_valid", "expected_return")
    RANK_FIELD_NUMBER: _ClassVar[int]
    ID_FIELD_NUMBER: _ClassVar[int]
    SOURCE_FIELD_NUMBER: _ClassVar[int]
    CELL_REF_FIELD_NUMBER: _ClassVar[int]
    SENSOR_REF_FIELD_NUMBER: _ClassVar[int]
    TITLE_FIELD_NUMBER: _ClassVar[int]
    MESSAGE_FIELD_NUMBER: _ClassVar[int]
    STAGE_FIELD_NUMBER: _ClassVar[int]
    STAGE_EXPLANATION_FIELD_NUMBER: _ClassVar[int]
    SEVERITY_FIELD_NUMBER: _ClassVar[int]
    TRUST_VALID_FIELD_NUMBER: _ClassVar[int]
    EXPECTED_RETURN_FIELD_NUMBER: _ClassVar[int]
    rank: int
    id: str
    source: str
    cell_ref: str
    sensor_ref: str
    title: str
    message: str
    stage: CascadeStage
    stage_explanation: str
    severity: int
    trust_valid: bool
    expected_return: str
    def __init__(self, rank: _Optional[int] = ..., id: _Optional[str] = ..., source: _Optional[str] = ..., cell_ref: _Optional[str] = ..., sensor_ref: _Optional[str] = ..., title: _Optional[str] = ..., message: _Optional[str] = ..., stage: _Optional[_Union[CascadeStage, str]] = ..., stage_explanation: _Optional[str] = ..., severity: _Optional[int] = ..., trust_valid: _Optional[bool] = ..., expected_return: _Optional[str] = ...) -> None: ...

class Ladder(_message.Message):
    __slots__ = ("cells", "sources", "findings", "host_os", "coverage_available", "coverage_reason", "check_platforms", "devices", "confidence", "computed_at")
    CELLS_FIELD_NUMBER: _ClassVar[int]
    SOURCES_FIELD_NUMBER: _ClassVar[int]
    FINDINGS_FIELD_NUMBER: _ClassVar[int]
    HOST_OS_FIELD_NUMBER: _ClassVar[int]
    COVERAGE_AVAILABLE_FIELD_NUMBER: _ClassVar[int]
    COVERAGE_REASON_FIELD_NUMBER: _ClassVar[int]
    CHECK_PLATFORMS_FIELD_NUMBER: _ClassVar[int]
    DEVICES_FIELD_NUMBER: _ClassVar[int]
    CONFIDENCE_FIELD_NUMBER: _ClassVar[int]
    COMPUTED_AT_FIELD_NUMBER: _ClassVar[int]
    cells: _containers.RepeatedCompositeFieldContainer[LadderCell]
    sources: _containers.RepeatedCompositeFieldContainer[SourceState]
    findings: _containers.RepeatedCompositeFieldContainer[RankedFinding]
    host_os: str
    coverage_available: bool
    coverage_reason: str
    check_platforms: _containers.RepeatedCompositeFieldContainer[CheckPlatformCoverage]
    devices: _containers.RepeatedCompositeFieldContainer[Device]
    confidence: Confidence
    computed_at: _timestamp_pb2.Timestamp
    def __init__(self, cells: _Optional[_Iterable[_Union[LadderCell, _Mapping]]] = ..., sources: _Optional[_Iterable[_Union[SourceState, _Mapping]]] = ..., findings: _Optional[_Iterable[_Union[RankedFinding, _Mapping]]] = ..., host_os: _Optional[str] = ..., coverage_available: _Optional[bool] = ..., coverage_reason: _Optional[str] = ..., check_platforms: _Optional[_Iterable[_Union[CheckPlatformCoverage, _Mapping]]] = ..., devices: _Optional[_Iterable[_Union[Device, _Mapping]]] = ..., confidence: _Optional[_Union[Confidence, _Mapping]] = ..., computed_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ...) -> None: ...

class GetLadderRequest(_message.Message):
    __slots__ = ()
    def __init__(self) -> None: ...

class GetLadderResponse(_message.Message):
    __slots__ = ("ladder",)
    LADDER_FIELD_NUMBER: _ClassVar[int]
    ladder: Ladder
    def __init__(self, ladder: _Optional[_Union[Ladder, _Mapping]] = ...) -> None: ...

class ListCellsRequest(_message.Message):
    __slots__ = ("device_class", "rung", "host_os", "cell_ref")
    DEVICE_CLASS_FIELD_NUMBER: _ClassVar[int]
    RUNG_FIELD_NUMBER: _ClassVar[int]
    HOST_OS_FIELD_NUMBER: _ClassVar[int]
    CELL_REF_FIELD_NUMBER: _ClassVar[int]
    device_class: str
    rung: Rung
    host_os: str
    cell_ref: str
    def __init__(self, device_class: _Optional[str] = ..., rung: _Optional[_Union[Rung, str]] = ..., host_os: _Optional[str] = ..., cell_ref: _Optional[str] = ...) -> None: ...

class ListCellsResponse(_message.Message):
    __slots__ = ("cells", "computed_at")
    CELLS_FIELD_NUMBER: _ClassVar[int]
    COMPUTED_AT_FIELD_NUMBER: _ClassVar[int]
    cells: _containers.RepeatedCompositeFieldContainer[LadderCell]
    computed_at: _timestamp_pb2.Timestamp
    def __init__(self, cells: _Optional[_Iterable[_Union[LadderCell, _Mapping]]] = ..., computed_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ...) -> None: ...

class ListDevicesRequest(_message.Message):
    __slots__ = ("device_class", "device_id")
    DEVICE_CLASS_FIELD_NUMBER: _ClassVar[int]
    DEVICE_ID_FIELD_NUMBER: _ClassVar[int]
    device_class: str
    device_id: str
    def __init__(self, device_class: _Optional[str] = ..., device_id: _Optional[str] = ...) -> None: ...

class ListDevicesResponse(_message.Message):
    __slots__ = ("devices", "available", "unavailable_reason", "computed_at")
    DEVICES_FIELD_NUMBER: _ClassVar[int]
    AVAILABLE_FIELD_NUMBER: _ClassVar[int]
    UNAVAILABLE_REASON_FIELD_NUMBER: _ClassVar[int]
    COMPUTED_AT_FIELD_NUMBER: _ClassVar[int]
    devices: _containers.RepeatedCompositeFieldContainer[Device]
    available: bool
    unavailable_reason: str
    computed_at: _timestamp_pb2.Timestamp
    def __init__(self, devices: _Optional[_Iterable[_Union[Device, _Mapping]]] = ..., available: _Optional[bool] = ..., unavailable_reason: _Optional[str] = ..., computed_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ...) -> None: ...

class ListSourcesRequest(_message.Message):
    __slots__ = ()
    def __init__(self) -> None: ...

class ListSourcesResponse(_message.Message):
    __slots__ = ("sources", "check_platforms", "confidence", "computed_at")
    SOURCES_FIELD_NUMBER: _ClassVar[int]
    CHECK_PLATFORMS_FIELD_NUMBER: _ClassVar[int]
    CONFIDENCE_FIELD_NUMBER: _ClassVar[int]
    COMPUTED_AT_FIELD_NUMBER: _ClassVar[int]
    sources: _containers.RepeatedCompositeFieldContainer[SourceState]
    check_platforms: _containers.RepeatedCompositeFieldContainer[CheckPlatformCoverage]
    confidence: Confidence
    computed_at: _timestamp_pb2.Timestamp
    def __init__(self, sources: _Optional[_Iterable[_Union[SourceState, _Mapping]]] = ..., check_platforms: _Optional[_Iterable[_Union[CheckPlatformCoverage, _Mapping]]] = ..., confidence: _Optional[_Union[Confidence, _Mapping]] = ..., computed_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ...) -> None: ...

class RankFindingsRequest(_message.Message):
    __slots__ = ("stage",)
    STAGE_FIELD_NUMBER: _ClassVar[int]
    stage: CascadeStage
    def __init__(self, stage: _Optional[_Union[CascadeStage, str]] = ...) -> None: ...

class RankFindingsResponse(_message.Message):
    __slots__ = ("findings", "applied_cascade", "computed_at")
    FINDINGS_FIELD_NUMBER: _ClassVar[int]
    APPLIED_CASCADE_FIELD_NUMBER: _ClassVar[int]
    COMPUTED_AT_FIELD_NUMBER: _ClassVar[int]
    findings: _containers.RepeatedCompositeFieldContainer[RankedFinding]
    applied_cascade: str
    computed_at: _timestamp_pb2.Timestamp
    def __init__(self, findings: _Optional[_Iterable[_Union[RankedFinding, _Mapping]]] = ..., applied_cascade: _Optional[str] = ..., computed_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ...) -> None: ...
