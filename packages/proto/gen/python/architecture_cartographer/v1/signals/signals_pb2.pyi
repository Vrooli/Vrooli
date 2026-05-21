from architecture_cartographer.v1.graph import graph_pb2 as _graph_pb2
from google.protobuf.internal import containers as _containers
from google.protobuf.internal import enum_type_wrapper as _enum_type_wrapper
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Iterable as _Iterable, Mapping as _Mapping
from typing import ClassVar as _ClassVar, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class Tier(int, metaclass=_enum_type_wrapper.EnumTypeWrapper):
    __slots__ = ()
    TIER_UNSPECIFIED: _ClassVar[Tier]
    TIER_AUTO_PLACE: _ClassVar[Tier]
    TIER_SUGGEST: _ClassVar[Tier]
    TIER_CONFLICT: _ClassVar[Tier]
TIER_UNSPECIFIED: Tier
TIER_AUTO_PLACE: Tier
TIER_SUGGEST: Tier
TIER_CONFLICT: Tier

class Evidence(_message.Message):
    __slots__ = ("kind", "summary", "locator", "weight")
    KIND_FIELD_NUMBER: _ClassVar[int]
    SUMMARY_FIELD_NUMBER: _ClassVar[int]
    LOCATOR_FIELD_NUMBER: _ClassVar[int]
    WEIGHT_FIELD_NUMBER: _ClassVar[int]
    kind: str
    summary: str
    locator: str
    weight: float
    def __init__(self, kind: _Optional[str] = ..., summary: _Optional[str] = ..., locator: _Optional[str] = ..., weight: _Optional[float] = ...) -> None: ...

class Score(_message.Message):
    __slots__ = ("signal", "domain", "value", "reason", "evidence")
    SIGNAL_FIELD_NUMBER: _ClassVar[int]
    DOMAIN_FIELD_NUMBER: _ClassVar[int]
    VALUE_FIELD_NUMBER: _ClassVar[int]
    REASON_FIELD_NUMBER: _ClassVar[int]
    EVIDENCE_FIELD_NUMBER: _ClassVar[int]
    signal: str
    domain: str
    value: float
    reason: str
    evidence: _containers.RepeatedCompositeFieldContainer[Evidence]
    def __init__(self, signal: _Optional[str] = ..., domain: _Optional[str] = ..., value: _Optional[float] = ..., reason: _Optional[str] = ..., evidence: _Optional[_Iterable[_Union[Evidence, _Mapping]]] = ...) -> None: ...

class Verdict(_message.Message):
    __slots__ = ("chunk_id", "chunk_path", "tier", "top_domain", "top_value", "runner_up_domain", "runner_up_value", "scores", "domain_values", "tied")
    CHUNK_ID_FIELD_NUMBER: _ClassVar[int]
    CHUNK_PATH_FIELD_NUMBER: _ClassVar[int]
    TIER_FIELD_NUMBER: _ClassVar[int]
    TOP_DOMAIN_FIELD_NUMBER: _ClassVar[int]
    TOP_VALUE_FIELD_NUMBER: _ClassVar[int]
    RUNNER_UP_DOMAIN_FIELD_NUMBER: _ClassVar[int]
    RUNNER_UP_VALUE_FIELD_NUMBER: _ClassVar[int]
    SCORES_FIELD_NUMBER: _ClassVar[int]
    DOMAIN_VALUES_FIELD_NUMBER: _ClassVar[int]
    TIED_FIELD_NUMBER: _ClassVar[int]
    chunk_id: str
    chunk_path: str
    tier: Tier
    top_domain: str
    top_value: float
    runner_up_domain: str
    runner_up_value: float
    scores: _containers.RepeatedCompositeFieldContainer[Score]
    domain_values: _containers.RepeatedCompositeFieldContainer[DomainValue]
    tied: bool
    def __init__(self, chunk_id: _Optional[str] = ..., chunk_path: _Optional[str] = ..., tier: _Optional[_Union[Tier, str]] = ..., top_domain: _Optional[str] = ..., top_value: _Optional[float] = ..., runner_up_domain: _Optional[str] = ..., runner_up_value: _Optional[float] = ..., scores: _Optional[_Iterable[_Union[Score, _Mapping]]] = ..., domain_values: _Optional[_Iterable[_Union[DomainValue, _Mapping]]] = ..., tied: _Optional[bool] = ...) -> None: ...

class DomainValue(_message.Message):
    __slots__ = ("domain", "value")
    DOMAIN_FIELD_NUMBER: _ClassVar[int]
    VALUE_FIELD_NUMBER: _ClassVar[int]
    domain: str
    value: float
    def __init__(self, domain: _Optional[str] = ..., value: _Optional[float] = ...) -> None: ...

class SignalDescriptor(_message.Message):
    __slots__ = ("name", "default_weight", "stability", "description", "disabled", "disabled_reason")
    NAME_FIELD_NUMBER: _ClassVar[int]
    DEFAULT_WEIGHT_FIELD_NUMBER: _ClassVar[int]
    STABILITY_FIELD_NUMBER: _ClassVar[int]
    DESCRIPTION_FIELD_NUMBER: _ClassVar[int]
    DISABLED_FIELD_NUMBER: _ClassVar[int]
    DISABLED_REASON_FIELD_NUMBER: _ClassVar[int]
    name: str
    default_weight: float
    stability: str
    description: str
    disabled: bool
    disabled_reason: str
    def __init__(self, name: _Optional[str] = ..., default_weight: _Optional[float] = ..., stability: _Optional[str] = ..., description: _Optional[str] = ..., disabled: _Optional[bool] = ..., disabled_reason: _Optional[str] = ...) -> None: ...

class ScoreChunkRequest(_message.Message):
    __slots__ = ("scenario", "chunk", "file_id")
    SCENARIO_FIELD_NUMBER: _ClassVar[int]
    CHUNK_FIELD_NUMBER: _ClassVar[int]
    FILE_ID_FIELD_NUMBER: _ClassVar[int]
    scenario: str
    chunk: _graph_pb2.Chunk
    file_id: str
    def __init__(self, scenario: _Optional[str] = ..., chunk: _Optional[_Union[_graph_pb2.Chunk, _Mapping]] = ..., file_id: _Optional[str] = ...) -> None: ...

class ScoreChunkResponse(_message.Message):
    __slots__ = ("verdict",)
    VERDICT_FIELD_NUMBER: _ClassVar[int]
    verdict: Verdict
    def __init__(self, verdict: _Optional[_Union[Verdict, _Mapping]] = ...) -> None: ...

class ExplainVerdictRequest(_message.Message):
    __slots__ = ("scenario", "chunk", "file_id")
    SCENARIO_FIELD_NUMBER: _ClassVar[int]
    CHUNK_FIELD_NUMBER: _ClassVar[int]
    FILE_ID_FIELD_NUMBER: _ClassVar[int]
    scenario: str
    chunk: _graph_pb2.Chunk
    file_id: str
    def __init__(self, scenario: _Optional[str] = ..., chunk: _Optional[_Union[_graph_pb2.Chunk, _Mapping]] = ..., file_id: _Optional[str] = ...) -> None: ...

class ExplainVerdictResponse(_message.Message):
    __slots__ = ("verdict",)
    VERDICT_FIELD_NUMBER: _ClassVar[int]
    verdict: Verdict
    def __init__(self, verdict: _Optional[_Union[Verdict, _Mapping]] = ...) -> None: ...

class ListSignalsRequest(_message.Message):
    __slots__ = ("scenario",)
    SCENARIO_FIELD_NUMBER: _ClassVar[int]
    scenario: str
    def __init__(self, scenario: _Optional[str] = ...) -> None: ...

class ListSignalsResponse(_message.Message):
    __slots__ = ("signals",)
    SIGNALS_FIELD_NUMBER: _ClassVar[int]
    signals: _containers.RepeatedCompositeFieldContainer[SignalDescriptor]
    def __init__(self, signals: _Optional[_Iterable[_Union[SignalDescriptor, _Mapping]]] = ...) -> None: ...
