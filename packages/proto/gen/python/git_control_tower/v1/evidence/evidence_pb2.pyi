from test_genie.v1.runs import runs_pb2 as _runs_pb2
from google.protobuf.internal import containers as _containers
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Iterable as _Iterable, Mapping as _Mapping
from typing import ClassVar as _ClassVar, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class ListRunsRequest(_message.Message):
    __slots__ = ("scenario", "status", "search", "provider", "phase_class", "dimension", "limit", "offset")
    SCENARIO_FIELD_NUMBER: _ClassVar[int]
    STATUS_FIELD_NUMBER: _ClassVar[int]
    SEARCH_FIELD_NUMBER: _ClassVar[int]
    PROVIDER_FIELD_NUMBER: _ClassVar[int]
    PHASE_CLASS_FIELD_NUMBER: _ClassVar[int]
    DIMENSION_FIELD_NUMBER: _ClassVar[int]
    LIMIT_FIELD_NUMBER: _ClassVar[int]
    OFFSET_FIELD_NUMBER: _ClassVar[int]
    scenario: str
    status: str
    search: str
    provider: str
    phase_class: str
    dimension: str
    limit: int
    offset: int
    def __init__(self, scenario: _Optional[str] = ..., status: _Optional[str] = ..., search: _Optional[str] = ..., provider: _Optional[str] = ..., phase_class: _Optional[str] = ..., dimension: _Optional[str] = ..., limit: _Optional[int] = ..., offset: _Optional[int] = ...) -> None: ...

class ListRunsResponse(_message.Message):
    __slots__ = ("runs", "total", "limit", "offset", "has_more")
    RUNS_FIELD_NUMBER: _ClassVar[int]
    TOTAL_FIELD_NUMBER: _ClassVar[int]
    LIMIT_FIELD_NUMBER: _ClassVar[int]
    OFFSET_FIELD_NUMBER: _ClassVar[int]
    HAS_MORE_FIELD_NUMBER: _ClassVar[int]
    runs: _containers.RepeatedCompositeFieldContainer[_runs_pb2.RunInfo]
    total: int
    limit: int
    offset: int
    has_more: bool
    def __init__(self, runs: _Optional[_Iterable[_Union[_runs_pb2.RunInfo, _Mapping]]] = ..., total: _Optional[int] = ..., limit: _Optional[int] = ..., offset: _Optional[int] = ..., has_more: _Optional[bool] = ...) -> None: ...

class GetRunRequest(_message.Message):
    __slots__ = ("scenario", "run_id", "artifact_kinds", "artifact_search", "artifact_limit", "artifact_offset")
    SCENARIO_FIELD_NUMBER: _ClassVar[int]
    RUN_ID_FIELD_NUMBER: _ClassVar[int]
    ARTIFACT_KINDS_FIELD_NUMBER: _ClassVar[int]
    ARTIFACT_SEARCH_FIELD_NUMBER: _ClassVar[int]
    ARTIFACT_LIMIT_FIELD_NUMBER: _ClassVar[int]
    ARTIFACT_OFFSET_FIELD_NUMBER: _ClassVar[int]
    scenario: str
    run_id: str
    artifact_kinds: _containers.RepeatedScalarFieldContainer[str]
    artifact_search: str
    artifact_limit: int
    artifact_offset: int
    def __init__(self, scenario: _Optional[str] = ..., run_id: _Optional[str] = ..., artifact_kinds: _Optional[_Iterable[str]] = ..., artifact_search: _Optional[str] = ..., artifact_limit: _Optional[int] = ..., artifact_offset: _Optional[int] = ...) -> None: ...

class GetRunResponse(_message.Message):
    __slots__ = ("run", "terminal_snapshot_schema_version", "degraded_reasons", "artifact_catalog_schema_version", "artifact_catalog_digest", "artifacts", "artifact_total", "artifact_limit", "artifact_offset", "artifacts_have_more", "legacy_artifacts_discovered")
    RUN_FIELD_NUMBER: _ClassVar[int]
    TERMINAL_SNAPSHOT_SCHEMA_VERSION_FIELD_NUMBER: _ClassVar[int]
    DEGRADED_REASONS_FIELD_NUMBER: _ClassVar[int]
    ARTIFACT_CATALOG_SCHEMA_VERSION_FIELD_NUMBER: _ClassVar[int]
    ARTIFACT_CATALOG_DIGEST_FIELD_NUMBER: _ClassVar[int]
    ARTIFACTS_FIELD_NUMBER: _ClassVar[int]
    ARTIFACT_TOTAL_FIELD_NUMBER: _ClassVar[int]
    ARTIFACT_LIMIT_FIELD_NUMBER: _ClassVar[int]
    ARTIFACT_OFFSET_FIELD_NUMBER: _ClassVar[int]
    ARTIFACTS_HAVE_MORE_FIELD_NUMBER: _ClassVar[int]
    LEGACY_ARTIFACTS_DISCOVERED_FIELD_NUMBER: _ClassVar[int]
    run: _runs_pb2.RunInfo
    terminal_snapshot_schema_version: int
    degraded_reasons: _containers.RepeatedScalarFieldContainer[str]
    artifact_catalog_schema_version: int
    artifact_catalog_digest: str
    artifacts: _containers.RepeatedCompositeFieldContainer[_runs_pb2.ArtifactRef]
    artifact_total: int
    artifact_limit: int
    artifact_offset: int
    artifacts_have_more: bool
    legacy_artifacts_discovered: bool
    def __init__(self, run: _Optional[_Union[_runs_pb2.RunInfo, _Mapping]] = ..., terminal_snapshot_schema_version: _Optional[int] = ..., degraded_reasons: _Optional[_Iterable[str]] = ..., artifact_catalog_schema_version: _Optional[int] = ..., artifact_catalog_digest: _Optional[str] = ..., artifacts: _Optional[_Iterable[_Union[_runs_pb2.ArtifactRef, _Mapping]]] = ..., artifact_total: _Optional[int] = ..., artifact_limit: _Optional[int] = ..., artifact_offset: _Optional[int] = ..., artifacts_have_more: _Optional[bool] = ..., legacy_artifacts_discovered: _Optional[bool] = ...) -> None: ...

class ListEvidenceRequest(_message.Message):
    __slots__ = ("scenario", "kinds", "search", "run_status", "limit", "offset", "run_limit")
    SCENARIO_FIELD_NUMBER: _ClassVar[int]
    KINDS_FIELD_NUMBER: _ClassVar[int]
    SEARCH_FIELD_NUMBER: _ClassVar[int]
    RUN_STATUS_FIELD_NUMBER: _ClassVar[int]
    LIMIT_FIELD_NUMBER: _ClassVar[int]
    OFFSET_FIELD_NUMBER: _ClassVar[int]
    RUN_LIMIT_FIELD_NUMBER: _ClassVar[int]
    scenario: str
    kinds: _containers.RepeatedScalarFieldContainer[str]
    search: str
    run_status: str
    limit: int
    offset: int
    run_limit: int
    def __init__(self, scenario: _Optional[str] = ..., kinds: _Optional[_Iterable[str]] = ..., search: _Optional[str] = ..., run_status: _Optional[str] = ..., limit: _Optional[int] = ..., offset: _Optional[int] = ..., run_limit: _Optional[int] = ...) -> None: ...

class EvidenceItem(_message.Message):
    __slots__ = ("run", "artifact")
    RUN_FIELD_NUMBER: _ClassVar[int]
    ARTIFACT_FIELD_NUMBER: _ClassVar[int]
    run: _runs_pb2.RunInfo
    artifact: _runs_pb2.ArtifactRef
    def __init__(self, run: _Optional[_Union[_runs_pb2.RunInfo, _Mapping]] = ..., artifact: _Optional[_Union[_runs_pb2.ArtifactRef, _Mapping]] = ...) -> None: ...

class ListEvidenceResponse(_message.Message):
    __slots__ = ("items", "total", "limit", "offset", "has_more", "degraded_reasons")
    ITEMS_FIELD_NUMBER: _ClassVar[int]
    TOTAL_FIELD_NUMBER: _ClassVar[int]
    LIMIT_FIELD_NUMBER: _ClassVar[int]
    OFFSET_FIELD_NUMBER: _ClassVar[int]
    HAS_MORE_FIELD_NUMBER: _ClassVar[int]
    DEGRADED_REASONS_FIELD_NUMBER: _ClassVar[int]
    items: _containers.RepeatedCompositeFieldContainer[EvidenceItem]
    total: int
    limit: int
    offset: int
    has_more: bool
    degraded_reasons: _containers.RepeatedScalarFieldContainer[str]
    def __init__(self, items: _Optional[_Iterable[_Union[EvidenceItem, _Mapping]]] = ..., total: _Optional[int] = ..., limit: _Optional[int] = ..., offset: _Optional[int] = ..., has_more: _Optional[bool] = ..., degraded_reasons: _Optional[_Iterable[str]] = ...) -> None: ...
