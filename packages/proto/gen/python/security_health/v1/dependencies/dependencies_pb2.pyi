from google.protobuf.internal import containers as _containers
from google.protobuf.internal import enum_type_wrapper as _enum_type_wrapper
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Iterable as _Iterable, Mapping as _Mapping
from typing import ClassVar as _ClassVar, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class Mode(int, metaclass=_enum_type_wrapper.EnumTypeWrapper):
    __slots__ = ()
    MODE_UNSPECIFIED: _ClassVar[Mode]
    MODE_AI: _ClassVar[Mode]
    MODE_TEXT: _ClassVar[Mode]

class Ecosystem(int, metaclass=_enum_type_wrapper.EnumTypeWrapper):
    __slots__ = ()
    ECOSYSTEM_UNSPECIFIED: _ClassVar[Ecosystem]
    ECOSYSTEM_GO: _ClassVar[Ecosystem]
    ECOSYSTEM_NPM: _ClassVar[Ecosystem]
MODE_UNSPECIFIED: Mode
MODE_AI: Mode
MODE_TEXT: Mode
ECOSYSTEM_UNSPECIFIED: Ecosystem
ECOSYSTEM_GO: Ecosystem
ECOSYSTEM_NPM: Ecosystem

class DependencyRecord(_message.Message):
    __slots__ = ("scenario", "ecosystem", "name", "version", "source_file", "vuln_ids", "max_severity", "last_seen")
    SCENARIO_FIELD_NUMBER: _ClassVar[int]
    ECOSYSTEM_FIELD_NUMBER: _ClassVar[int]
    NAME_FIELD_NUMBER: _ClassVar[int]
    VERSION_FIELD_NUMBER: _ClassVar[int]
    SOURCE_FILE_FIELD_NUMBER: _ClassVar[int]
    VULN_IDS_FIELD_NUMBER: _ClassVar[int]
    MAX_SEVERITY_FIELD_NUMBER: _ClassVar[int]
    LAST_SEEN_FIELD_NUMBER: _ClassVar[int]
    scenario: str
    ecosystem: Ecosystem
    name: str
    version: str
    source_file: str
    vuln_ids: _containers.RepeatedScalarFieldContainer[str]
    max_severity: str
    last_seen: str
    def __init__(self, scenario: _Optional[str] = ..., ecosystem: _Optional[_Union[Ecosystem, str]] = ..., name: _Optional[str] = ..., version: _Optional[str] = ..., source_file: _Optional[str] = ..., vuln_ids: _Optional[_Iterable[str]] = ..., max_severity: _Optional[str] = ..., last_seen: _Optional[str] = ...) -> None: ...

class SearchRequest(_message.Message):
    __slots__ = ("query", "limit", "mode", "ecosystem", "vulnerable_only", "name_glob")
    QUERY_FIELD_NUMBER: _ClassVar[int]
    LIMIT_FIELD_NUMBER: _ClassVar[int]
    MODE_FIELD_NUMBER: _ClassVar[int]
    ECOSYSTEM_FIELD_NUMBER: _ClassVar[int]
    VULNERABLE_ONLY_FIELD_NUMBER: _ClassVar[int]
    NAME_GLOB_FIELD_NUMBER: _ClassVar[int]
    query: str
    limit: int
    mode: Mode
    ecosystem: Ecosystem
    vulnerable_only: bool
    name_glob: str
    def __init__(self, query: _Optional[str] = ..., limit: _Optional[int] = ..., mode: _Optional[_Union[Mode, str]] = ..., ecosystem: _Optional[_Union[Ecosystem, str]] = ..., vulnerable_only: _Optional[bool] = ..., name_glob: _Optional[str] = ...) -> None: ...

class SearchResult(_message.Message):
    __slots__ = ("record", "score")
    RECORD_FIELD_NUMBER: _ClassVar[int]
    SCORE_FIELD_NUMBER: _ClassVar[int]
    record: DependencyRecord
    score: float
    def __init__(self, record: _Optional[_Union[DependencyRecord, _Mapping]] = ..., score: _Optional[float] = ...) -> None: ...

class SearchResponse(_message.Message):
    __slots__ = ("results", "mode_used")
    RESULTS_FIELD_NUMBER: _ClassVar[int]
    MODE_USED_FIELD_NUMBER: _ClassVar[int]
    results: _containers.RepeatedCompositeFieldContainer[SearchResult]
    mode_used: Mode
    def __init__(self, results: _Optional[_Iterable[_Union[SearchResult, _Mapping]]] = ..., mode_used: _Optional[_Union[Mode, str]] = ...) -> None: ...

class StatusRequest(_message.Message):
    __slots__ = ()
    def __init__(self) -> None: ...

class StatusResponse(_message.Message):
    __slots__ = ("available", "ollama", "qdrant", "indexed_count", "vulnerable_count", "last_reconcile_at", "last_reconcile_outcome", "indexed_vectors", "expected_vectors", "index_ready")
    AVAILABLE_FIELD_NUMBER: _ClassVar[int]
    OLLAMA_FIELD_NUMBER: _ClassVar[int]
    QDRANT_FIELD_NUMBER: _ClassVar[int]
    INDEXED_COUNT_FIELD_NUMBER: _ClassVar[int]
    VULNERABLE_COUNT_FIELD_NUMBER: _ClassVar[int]
    LAST_RECONCILE_AT_FIELD_NUMBER: _ClassVar[int]
    LAST_RECONCILE_OUTCOME_FIELD_NUMBER: _ClassVar[int]
    INDEXED_VECTORS_FIELD_NUMBER: _ClassVar[int]
    EXPECTED_VECTORS_FIELD_NUMBER: _ClassVar[int]
    INDEX_READY_FIELD_NUMBER: _ClassVar[int]
    available: bool
    ollama: bool
    qdrant: bool
    indexed_count: int
    vulnerable_count: int
    last_reconcile_at: str
    last_reconcile_outcome: str
    indexed_vectors: int
    expected_vectors: int
    index_ready: bool
    def __init__(self, available: _Optional[bool] = ..., ollama: _Optional[bool] = ..., qdrant: _Optional[bool] = ..., indexed_count: _Optional[int] = ..., vulnerable_count: _Optional[int] = ..., last_reconcile_at: _Optional[str] = ..., last_reconcile_outcome: _Optional[str] = ..., indexed_vectors: _Optional[int] = ..., expected_vectors: _Optional[int] = ..., index_ready: _Optional[bool] = ...) -> None: ...
