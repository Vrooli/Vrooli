import datetime

from buf.validate import validate_pb2 as _validate_pb2
from google.protobuf import timestamp_pb2 as _timestamp_pb2
from lighting_control.v1.control import control_pb2 as _control_pb2
from google.protobuf.internal import containers as _containers
from google.protobuf.internal import enum_type_wrapper as _enum_type_wrapper
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Iterable as _Iterable, Mapping as _Mapping
from typing import ClassVar as _ClassVar, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class PresetKind(int, metaclass=_enum_type_wrapper.EnumTypeWrapper):
    __slots__ = ()
    PRESET_KIND_UNSPECIFIED: _ClassVar[PresetKind]
    PRESET_KIND_OFF: _ClassVar[PresetKind]
    PRESET_KIND_LOAD: _ClassVar[PresetKind]
    PRESET_KIND_WORK: _ClassVar[PresetKind]
    PRESET_KIND_HEALTH: _ClassVar[PresetKind]
    PRESET_KIND_COMBINED: _ClassVar[PresetKind]
    PRESET_KIND_STATIC: _ClassVar[PresetKind]

class EvidenceState(int, metaclass=_enum_type_wrapper.EnumTypeWrapper):
    __slots__ = ()
    EVIDENCE_STATE_UNSPECIFIED: _ClassVar[EvidenceState]
    EVIDENCE_STATE_FRESH: _ClassVar[EvidenceState]
    EVIDENCE_STATE_STALE: _ClassVar[EvidenceState]
    EVIDENCE_STATE_UNAVAILABLE: _ClassVar[EvidenceState]
    EVIDENCE_STATE_DEGRADED: _ClassVar[EvidenceState]
PRESET_KIND_UNSPECIFIED: PresetKind
PRESET_KIND_OFF: PresetKind
PRESET_KIND_LOAD: PresetKind
PRESET_KIND_WORK: PresetKind
PRESET_KIND_HEALTH: PresetKind
PRESET_KIND_COMBINED: PresetKind
PRESET_KIND_STATIC: PresetKind
EVIDENCE_STATE_UNSPECIFIED: EvidenceState
EVIDENCE_STATE_FRESH: EvidenceState
EVIDENCE_STATE_STALE: EvidenceState
EVIDENCE_STATE_UNAVAILABLE: EvidenceState
EVIDENCE_STATE_DEGRADED: EvidenceState

class Preset(_message.Message):
    __slots__ = ("kind", "label", "color", "brightness_cap_percent", "effect_id", "enabled")
    KIND_FIELD_NUMBER: _ClassVar[int]
    LABEL_FIELD_NUMBER: _ClassVar[int]
    COLOR_FIELD_NUMBER: _ClassVar[int]
    BRIGHTNESS_CAP_PERCENT_FIELD_NUMBER: _ClassVar[int]
    EFFECT_ID_FIELD_NUMBER: _ClassVar[int]
    ENABLED_FIELD_NUMBER: _ClassVar[int]
    kind: PresetKind
    label: str
    color: _control_pb2.Color
    brightness_cap_percent: int
    effect_id: str
    enabled: bool
    def __init__(self, kind: _Optional[_Union[PresetKind, str]] = ..., label: _Optional[str] = ..., color: _Optional[_Union[_control_pb2.Color, _Mapping]] = ..., brightness_cap_percent: _Optional[int] = ..., effect_id: _Optional[str] = ..., enabled: _Optional[bool] = ...) -> None: ...

class PolicyConfig(_message.Message):
    __slots__ = ("default_preset", "presets", "source_max_age_seconds", "load_enter_percent", "load_exit_percent", "debounce_seconds", "brightness_cap_percent", "stale_fallback_color", "stale_fallback_brightness_percent")
    DEFAULT_PRESET_FIELD_NUMBER: _ClassVar[int]
    PRESETS_FIELD_NUMBER: _ClassVar[int]
    SOURCE_MAX_AGE_SECONDS_FIELD_NUMBER: _ClassVar[int]
    LOAD_ENTER_PERCENT_FIELD_NUMBER: _ClassVar[int]
    LOAD_EXIT_PERCENT_FIELD_NUMBER: _ClassVar[int]
    DEBOUNCE_SECONDS_FIELD_NUMBER: _ClassVar[int]
    BRIGHTNESS_CAP_PERCENT_FIELD_NUMBER: _ClassVar[int]
    STALE_FALLBACK_COLOR_FIELD_NUMBER: _ClassVar[int]
    STALE_FALLBACK_BRIGHTNESS_PERCENT_FIELD_NUMBER: _ClassVar[int]
    default_preset: PresetKind
    presets: _containers.RepeatedCompositeFieldContainer[Preset]
    source_max_age_seconds: int
    load_enter_percent: int
    load_exit_percent: int
    debounce_seconds: int
    brightness_cap_percent: int
    stale_fallback_color: _control_pb2.Color
    stale_fallback_brightness_percent: int
    def __init__(self, default_preset: _Optional[_Union[PresetKind, str]] = ..., presets: _Optional[_Iterable[_Union[Preset, _Mapping]]] = ..., source_max_age_seconds: _Optional[int] = ..., load_enter_percent: _Optional[int] = ..., load_exit_percent: _Optional[int] = ..., debounce_seconds: _Optional[int] = ..., brightness_cap_percent: _Optional[int] = ..., stale_fallback_color: _Optional[_Union[_control_pb2.Color, _Mapping]] = ..., stale_fallback_brightness_percent: _Optional[int] = ...) -> None: ...

class EvidenceSource(_message.Message):
    __slots__ = ("source", "state", "observed_at", "source_timestamp", "reason", "measurements")
    class MeasurementsEntry(_message.Message):
        __slots__ = ("key", "value")
        KEY_FIELD_NUMBER: _ClassVar[int]
        VALUE_FIELD_NUMBER: _ClassVar[int]
        key: str
        value: float
        def __init__(self, key: _Optional[str] = ..., value: _Optional[float] = ...) -> None: ...
    SOURCE_FIELD_NUMBER: _ClassVar[int]
    STATE_FIELD_NUMBER: _ClassVar[int]
    OBSERVED_AT_FIELD_NUMBER: _ClassVar[int]
    SOURCE_TIMESTAMP_FIELD_NUMBER: _ClassVar[int]
    REASON_FIELD_NUMBER: _ClassVar[int]
    MEASUREMENTS_FIELD_NUMBER: _ClassVar[int]
    source: str
    state: EvidenceState
    observed_at: _timestamp_pb2.Timestamp
    source_timestamp: _timestamp_pb2.Timestamp
    reason: str
    measurements: _containers.ScalarMap[str, float]
    def __init__(self, source: _Optional[str] = ..., state: _Optional[_Union[EvidenceState, str]] = ..., observed_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., source_timestamp: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., reason: _Optional[str] = ..., measurements: _Optional[_Mapping[str, float]] = ...) -> None: ...

class EffectivePolicy(_message.Message):
    __slots__ = ("preset", "color", "brightness_percent", "effect_id", "reason", "evaluated_at", "evidence", "manual_priority", "request_expires_at", "static_fallback")
    PRESET_FIELD_NUMBER: _ClassVar[int]
    COLOR_FIELD_NUMBER: _ClassVar[int]
    BRIGHTNESS_PERCENT_FIELD_NUMBER: _ClassVar[int]
    EFFECT_ID_FIELD_NUMBER: _ClassVar[int]
    REASON_FIELD_NUMBER: _ClassVar[int]
    EVALUATED_AT_FIELD_NUMBER: _ClassVar[int]
    EVIDENCE_FIELD_NUMBER: _ClassVar[int]
    MANUAL_PRIORITY_FIELD_NUMBER: _ClassVar[int]
    REQUEST_EXPIRES_AT_FIELD_NUMBER: _ClassVar[int]
    STATIC_FALLBACK_FIELD_NUMBER: _ClassVar[int]
    preset: PresetKind
    color: _control_pb2.Color
    brightness_percent: int
    effect_id: str
    reason: str
    evaluated_at: _timestamp_pb2.Timestamp
    evidence: _containers.RepeatedCompositeFieldContainer[EvidenceSource]
    manual_priority: bool
    request_expires_at: _timestamp_pb2.Timestamp
    static_fallback: bool
    def __init__(self, preset: _Optional[_Union[PresetKind, str]] = ..., color: _Optional[_Union[_control_pb2.Color, _Mapping]] = ..., brightness_percent: _Optional[int] = ..., effect_id: _Optional[str] = ..., reason: _Optional[str] = ..., evaluated_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., evidence: _Optional[_Iterable[_Union[EvidenceSource, _Mapping]]] = ..., manual_priority: _Optional[bool] = ..., request_expires_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., static_fallback: _Optional[bool] = ...) -> None: ...

class GetPolicyRequest(_message.Message):
    __slots__ = ()
    def __init__(self) -> None: ...

class GetPolicyResponse(_message.Message):
    __slots__ = ("config", "effective")
    CONFIG_FIELD_NUMBER: _ClassVar[int]
    EFFECTIVE_FIELD_NUMBER: _ClassVar[int]
    config: PolicyConfig
    effective: EffectivePolicy
    def __init__(self, config: _Optional[_Union[PolicyConfig, _Mapping]] = ..., effective: _Optional[_Union[EffectivePolicy, _Mapping]] = ...) -> None: ...

class SetPolicyRequest(_message.Message):
    __slots__ = ("config",)
    CONFIG_FIELD_NUMBER: _ClassVar[int]
    config: PolicyConfig
    def __init__(self, config: _Optional[_Union[PolicyConfig, _Mapping]] = ...) -> None: ...

class SetPolicyResponse(_message.Message):
    __slots__ = ("config",)
    CONFIG_FIELD_NUMBER: _ClassVar[int]
    config: PolicyConfig
    def __init__(self, config: _Optional[_Union[PolicyConfig, _Mapping]] = ...) -> None: ...

class SetManualPresetRequest(_message.Message):
    __slots__ = ("preset", "expires_at")
    PRESET_FIELD_NUMBER: _ClassVar[int]
    EXPIRES_AT_FIELD_NUMBER: _ClassVar[int]
    preset: PresetKind
    expires_at: _timestamp_pb2.Timestamp
    def __init__(self, preset: _Optional[_Union[PresetKind, str]] = ..., expires_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ...) -> None: ...

class PolicyZoneResult(_message.Message):
    __slots__ = ("controller_id", "zone_id", "static_fallback", "result")
    CONTROLLER_ID_FIELD_NUMBER: _ClassVar[int]
    ZONE_ID_FIELD_NUMBER: _ClassVar[int]
    STATIC_FALLBACK_FIELD_NUMBER: _ClassVar[int]
    RESULT_FIELD_NUMBER: _ClassVar[int]
    controller_id: str
    zone_id: str
    static_fallback: bool
    result: _control_pb2.ZoneApplyResult
    def __init__(self, controller_id: _Optional[str] = ..., zone_id: _Optional[str] = ..., static_fallback: _Optional[bool] = ..., result: _Optional[_Union[_control_pb2.ZoneApplyResult, _Mapping]] = ...) -> None: ...

class SetManualPresetResponse(_message.Message):
    __slots__ = ("effective", "results")
    EFFECTIVE_FIELD_NUMBER: _ClassVar[int]
    RESULTS_FIELD_NUMBER: _ClassVar[int]
    effective: EffectivePolicy
    results: _containers.RepeatedCompositeFieldContainer[PolicyZoneResult]
    def __init__(self, effective: _Optional[_Union[EffectivePolicy, _Mapping]] = ..., results: _Optional[_Iterable[_Union[PolicyZoneResult, _Mapping]]] = ...) -> None: ...

class ClearManualPresetRequest(_message.Message):
    __slots__ = ()
    def __init__(self) -> None: ...

class ClearManualPresetResponse(_message.Message):
    __slots__ = ("cleared",)
    CLEARED_FIELD_NUMBER: _ClassVar[int]
    cleared: bool
    def __init__(self, cleared: _Optional[bool] = ...) -> None: ...

class EvaluatePolicyRequest(_message.Message):
    __slots__ = ("trigger",)
    TRIGGER_FIELD_NUMBER: _ClassVar[int]
    trigger: str
    def __init__(self, trigger: _Optional[str] = ...) -> None: ...

class EvaluatePolicyResponse(_message.Message):
    __slots__ = ("effective", "results")
    EFFECTIVE_FIELD_NUMBER: _ClassVar[int]
    RESULTS_FIELD_NUMBER: _ClassVar[int]
    effective: EffectivePolicy
    results: _containers.RepeatedCompositeFieldContainer[PolicyZoneResult]
    def __init__(self, effective: _Optional[_Union[EffectivePolicy, _Mapping]] = ..., results: _Optional[_Iterable[_Union[PolicyZoneResult, _Mapping]]] = ...) -> None: ...
