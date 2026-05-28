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

class CouplingSeverity(int, metaclass=_enum_type_wrapper.EnumTypeWrapper):
    __slots__ = ()
    COUPLING_SEVERITY_UNSPECIFIED: _ClassVar[CouplingSeverity]
    COUPLING_SEVERITY_INFO: _ClassVar[CouplingSeverity]
    COUPLING_SEVERITY_WARN: _ClassVar[CouplingSeverity]
TIER_UNSPECIFIED: Tier
TIER_AUTO_PLACE: Tier
TIER_SUGGEST: Tier
TIER_CONFLICT: Tier
COUPLING_SEVERITY_UNSPECIFIED: CouplingSeverity
COUPLING_SEVERITY_INFO: CouplingSeverity
COUPLING_SEVERITY_WARN: CouplingSeverity

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

class Abstention(_message.Message):
    __slots__ = ("signal", "reason", "evidence")
    SIGNAL_FIELD_NUMBER: _ClassVar[int]
    REASON_FIELD_NUMBER: _ClassVar[int]
    EVIDENCE_FIELD_NUMBER: _ClassVar[int]
    signal: str
    reason: str
    evidence: _containers.RepeatedCompositeFieldContainer[Evidence]
    def __init__(self, signal: _Optional[str] = ..., reason: _Optional[str] = ..., evidence: _Optional[_Iterable[_Union[Evidence, _Mapping]]] = ...) -> None: ...

class Verdict(_message.Message):
    __slots__ = ("chunk_id", "chunk_path", "tier", "top_domain", "top_value", "runner_up_domain", "runner_up_value", "scores", "domain_values", "tied", "abstentions")
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
    ABSTENTIONS_FIELD_NUMBER: _ClassVar[int]
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
    abstentions: _containers.RepeatedCompositeFieldContainer[Abstention]
    def __init__(self, chunk_id: _Optional[str] = ..., chunk_path: _Optional[str] = ..., tier: _Optional[_Union[Tier, str]] = ..., top_domain: _Optional[str] = ..., top_value: _Optional[float] = ..., runner_up_domain: _Optional[str] = ..., runner_up_value: _Optional[float] = ..., scores: _Optional[_Iterable[_Union[Score, _Mapping]]] = ..., domain_values: _Optional[_Iterable[_Union[DomainValue, _Mapping]]] = ..., tied: _Optional[bool] = ..., abstentions: _Optional[_Iterable[_Union[Abstention, _Mapping]]] = ...) -> None: ...

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

class CouplingSmell(_message.Message):
    __slots__ = ("kind", "severity", "message")
    KIND_FIELD_NUMBER: _ClassVar[int]
    SEVERITY_FIELD_NUMBER: _ClassVar[int]
    MESSAGE_FIELD_NUMBER: _ClassVar[int]
    kind: str
    severity: CouplingSeverity
    message: str
    def __init__(self, kind: _Optional[str] = ..., severity: _Optional[_Union[CouplingSeverity, str]] = ..., message: _Optional[str] = ...) -> None: ...

class DomainCoupling(_message.Message):
    __slots__ = ("domain", "archetype", "efferent", "afferent", "instability", "fan_out", "depends_on", "depended_by", "stable_kernel", "health_score", "smells")
    DOMAIN_FIELD_NUMBER: _ClassVar[int]
    ARCHETYPE_FIELD_NUMBER: _ClassVar[int]
    EFFERENT_FIELD_NUMBER: _ClassVar[int]
    AFFERENT_FIELD_NUMBER: _ClassVar[int]
    INSTABILITY_FIELD_NUMBER: _ClassVar[int]
    FAN_OUT_FIELD_NUMBER: _ClassVar[int]
    DEPENDS_ON_FIELD_NUMBER: _ClassVar[int]
    DEPENDED_BY_FIELD_NUMBER: _ClassVar[int]
    STABLE_KERNEL_FIELD_NUMBER: _ClassVar[int]
    HEALTH_SCORE_FIELD_NUMBER: _ClassVar[int]
    SMELLS_FIELD_NUMBER: _ClassVar[int]
    domain: str
    archetype: str
    efferent: int
    afferent: int
    instability: float
    fan_out: float
    depends_on: _containers.RepeatedScalarFieldContainer[str]
    depended_by: _containers.RepeatedScalarFieldContainer[str]
    stable_kernel: bool
    health_score: float
    smells: _containers.RepeatedCompositeFieldContainer[CouplingSmell]
    def __init__(self, domain: _Optional[str] = ..., archetype: _Optional[str] = ..., efferent: _Optional[int] = ..., afferent: _Optional[int] = ..., instability: _Optional[float] = ..., fan_out: _Optional[float] = ..., depends_on: _Optional[_Iterable[str]] = ..., depended_by: _Optional[_Iterable[str]] = ..., stable_kernel: _Optional[bool] = ..., health_score: _Optional[float] = ..., smells: _Optional[_Iterable[_Union[CouplingSmell, _Mapping]]] = ...) -> None: ...

class BoundaryHealthRequest(_message.Message):
    __slots__ = ("scenario",)
    SCENARIO_FIELD_NUMBER: _ClassVar[int]
    scenario: str
    def __init__(self, scenario: _Optional[str] = ...) -> None: ...

class BoundaryHealthResponse(_message.Message):
    __slots__ = ("scenario", "total_domains", "domains")
    SCENARIO_FIELD_NUMBER: _ClassVar[int]
    TOTAL_DOMAINS_FIELD_NUMBER: _ClassVar[int]
    DOMAINS_FIELD_NUMBER: _ClassVar[int]
    scenario: str
    total_domains: int
    domains: _containers.RepeatedCompositeFieldContainer[DomainCoupling]
    def __init__(self, scenario: _Optional[str] = ..., total_domains: _Optional[int] = ..., domains: _Optional[_Iterable[_Union[DomainCoupling, _Mapping]]] = ...) -> None: ...

class ScoreChunkRequest(_message.Message):
    __slots__ = ("scenario", "chunk", "file_id", "repo_path")
    SCENARIO_FIELD_NUMBER: _ClassVar[int]
    CHUNK_FIELD_NUMBER: _ClassVar[int]
    FILE_ID_FIELD_NUMBER: _ClassVar[int]
    REPO_PATH_FIELD_NUMBER: _ClassVar[int]
    scenario: str
    chunk: _graph_pb2.Chunk
    file_id: str
    repo_path: str
    def __init__(self, scenario: _Optional[str] = ..., chunk: _Optional[_Union[_graph_pb2.Chunk, _Mapping]] = ..., file_id: _Optional[str] = ..., repo_path: _Optional[str] = ...) -> None: ...

class ScoreChunkResponse(_message.Message):
    __slots__ = ("verdict",)
    VERDICT_FIELD_NUMBER: _ClassVar[int]
    verdict: Verdict
    def __init__(self, verdict: _Optional[_Union[Verdict, _Mapping]] = ...) -> None: ...

class ExplainVerdictRequest(_message.Message):
    __slots__ = ("scenario", "chunk", "file_id", "repo_path")
    SCENARIO_FIELD_NUMBER: _ClassVar[int]
    CHUNK_FIELD_NUMBER: _ClassVar[int]
    FILE_ID_FIELD_NUMBER: _ClassVar[int]
    REPO_PATH_FIELD_NUMBER: _ClassVar[int]
    scenario: str
    chunk: _graph_pb2.Chunk
    file_id: str
    repo_path: str
    def __init__(self, scenario: _Optional[str] = ..., chunk: _Optional[_Union[_graph_pb2.Chunk, _Mapping]] = ..., file_id: _Optional[str] = ..., repo_path: _Optional[str] = ...) -> None: ...

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
