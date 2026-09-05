from architecture_cartographer.v1.graph import graph_pb2 as _graph_pb2
from architecture_cartographer.v1.shared import shared_pb2 as _shared_pb2
from google.protobuf.internal import containers as _containers
from google.protobuf.internal import enum_type_wrapper as _enum_type_wrapper
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Iterable as _Iterable, Mapping as _Mapping
from typing import ClassVar as _ClassVar, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class CouplingSeverity(int, metaclass=_enum_type_wrapper.EnumTypeWrapper):
    __slots__ = ()
    COUPLING_SEVERITY_UNSPECIFIED: _ClassVar[CouplingSeverity]
    COUPLING_SEVERITY_INFO: _ClassVar[CouplingSeverity]
    COUPLING_SEVERITY_WARN: _ClassVar[CouplingSeverity]
COUPLING_SEVERITY_UNSPECIFIED: CouplingSeverity
COUPLING_SEVERITY_INFO: CouplingSeverity
COUPLING_SEVERITY_WARN: CouplingSeverity

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
    verdict: _shared_pb2.Verdict
    def __init__(self, verdict: _Optional[_Union[_shared_pb2.Verdict, _Mapping]] = ...) -> None: ...

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
    verdict: _shared_pb2.Verdict
    def __init__(self, verdict: _Optional[_Union[_shared_pb2.Verdict, _Mapping]] = ...) -> None: ...

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
