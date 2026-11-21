import datetime

from architecture_cartographer.v1.domains import domains_pb2 as _domains_pb2
from architecture_cartographer.v1.shared import shared_pb2 as _shared_pb2
from common.v1 import attestation_pb2 as _attestation_pb2
from common.v1 import code_graph_pb2 as _code_graph_pb2
from google.protobuf import timestamp_pb2 as _timestamp_pb2
from google.protobuf.internal import containers as _containers
from google.protobuf.internal import enum_type_wrapper as _enum_type_wrapper
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Iterable as _Iterable, Mapping as _Mapping
from typing import ClassVar as _ClassVar, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class Language(int, metaclass=_enum_type_wrapper.EnumTypeWrapper):
    __slots__ = ()
    LANGUAGE_UNSPECIFIED: _ClassVar[Language]
    LANGUAGE_GO: _ClassVar[Language]
    LANGUAGE_TYPESCRIPT: _ClassVar[Language]

class AuthorityConfidence(int, metaclass=_enum_type_wrapper.EnumTypeWrapper):
    __slots__ = ()
    AUTHORITY_CONFIDENCE_UNSPECIFIED: _ClassVar[AuthorityConfidence]
    AUTHORITY_CONFIDENCE_HIGH: _ClassVar[AuthorityConfidence]
    AUTHORITY_CONFIDENCE_LOW: _ClassVar[AuthorityConfidence]
LANGUAGE_UNSPECIFIED: Language
LANGUAGE_GO: Language
LANGUAGE_TYPESCRIPT: Language
AUTHORITY_CONFIDENCE_UNSPECIFIED: AuthorityConfidence
AUTHORITY_CONFIDENCE_HIGH: AuthorityConfidence
AUTHORITY_CONFIDENCE_LOW: AuthorityConfidence

class FileNode(_message.Message):
    __slots__ = ("id", "path", "package_id", "language", "lines", "is_test")
    ID_FIELD_NUMBER: _ClassVar[int]
    PATH_FIELD_NUMBER: _ClassVar[int]
    PACKAGE_ID_FIELD_NUMBER: _ClassVar[int]
    LANGUAGE_FIELD_NUMBER: _ClassVar[int]
    LINES_FIELD_NUMBER: _ClassVar[int]
    IS_TEST_FIELD_NUMBER: _ClassVar[int]
    id: str
    path: str
    package_id: str
    language: Language
    lines: int
    is_test: bool
    def __init__(self, id: _Optional[str] = ..., path: _Optional[str] = ..., package_id: _Optional[str] = ..., language: _Optional[_Union[Language, str]] = ..., lines: _Optional[int] = ..., is_test: _Optional[bool] = ...) -> None: ...

class PackageNode(_message.Message):
    __slots__ = ("id", "import_path", "repo_path", "language")
    ID_FIELD_NUMBER: _ClassVar[int]
    IMPORT_PATH_FIELD_NUMBER: _ClassVar[int]
    REPO_PATH_FIELD_NUMBER: _ClassVar[int]
    LANGUAGE_FIELD_NUMBER: _ClassVar[int]
    id: str
    import_path: str
    repo_path: str
    language: Language
    def __init__(self, id: _Optional[str] = ..., import_path: _Optional[str] = ..., repo_path: _Optional[str] = ..., language: _Optional[_Union[Language, str]] = ...) -> None: ...

class SymbolNode(_message.Message):
    __slots__ = ("id", "name", "package_id", "file_id", "kind", "exported")
    ID_FIELD_NUMBER: _ClassVar[int]
    NAME_FIELD_NUMBER: _ClassVar[int]
    PACKAGE_ID_FIELD_NUMBER: _ClassVar[int]
    FILE_ID_FIELD_NUMBER: _ClassVar[int]
    KIND_FIELD_NUMBER: _ClassVar[int]
    EXPORTED_FIELD_NUMBER: _ClassVar[int]
    id: str
    name: str
    package_id: str
    file_id: str
    kind: str
    exported: bool
    def __init__(self, id: _Optional[str] = ..., name: _Optional[str] = ..., package_id: _Optional[str] = ..., file_id: _Optional[str] = ..., kind: _Optional[str] = ..., exported: _Optional[bool] = ...) -> None: ...

class ImportEdge(_message.Message):
    __slots__ = ("to_package_id", "symbol_ids", "test_only")
    FROM_FIELD_NUMBER: _ClassVar[int]
    TO_PACKAGE_ID_FIELD_NUMBER: _ClassVar[int]
    SYMBOL_IDS_FIELD_NUMBER: _ClassVar[int]
    TEST_ONLY_FIELD_NUMBER: _ClassVar[int]
    to_package_id: str
    symbol_ids: _containers.RepeatedScalarFieldContainer[str]
    test_only: bool
    def __init__(self, to_package_id: _Optional[str] = ..., symbol_ids: _Optional[_Iterable[str]] = ..., test_only: _Optional[bool] = ..., **kwargs) -> None: ...

class Chunk(_message.Message):
    __slots__ = ("id", "file_id", "path", "current_domain")
    ID_FIELD_NUMBER: _ClassVar[int]
    FILE_ID_FIELD_NUMBER: _ClassVar[int]
    PATH_FIELD_NUMBER: _ClassVar[int]
    CURRENT_DOMAIN_FIELD_NUMBER: _ClassVar[int]
    id: str
    file_id: str
    path: str
    current_domain: str
    def __init__(self, id: _Optional[str] = ..., file_id: _Optional[str] = ..., path: _Optional[str] = ..., current_domain: _Optional[str] = ...) -> None: ...

class GraphSnapshot(_message.Message):
    __slots__ = ("id", "scenario", "content_hash", "languages", "extracted_at", "extraction_ms", "files", "packages", "symbols", "imports", "extraction_profiles", "omitted_information")
    ID_FIELD_NUMBER: _ClassVar[int]
    SCENARIO_FIELD_NUMBER: _ClassVar[int]
    CONTENT_HASH_FIELD_NUMBER: _ClassVar[int]
    LANGUAGES_FIELD_NUMBER: _ClassVar[int]
    EXTRACTED_AT_FIELD_NUMBER: _ClassVar[int]
    EXTRACTION_MS_FIELD_NUMBER: _ClassVar[int]
    FILES_FIELD_NUMBER: _ClassVar[int]
    PACKAGES_FIELD_NUMBER: _ClassVar[int]
    SYMBOLS_FIELD_NUMBER: _ClassVar[int]
    IMPORTS_FIELD_NUMBER: _ClassVar[int]
    EXTRACTION_PROFILES_FIELD_NUMBER: _ClassVar[int]
    OMITTED_INFORMATION_FIELD_NUMBER: _ClassVar[int]
    id: str
    scenario: str
    content_hash: str
    languages: _containers.RepeatedScalarFieldContainer[Language]
    extracted_at: _timestamp_pb2.Timestamp
    extraction_ms: int
    files: _containers.RepeatedCompositeFieldContainer[FileNode]
    packages: _containers.RepeatedCompositeFieldContainer[PackageNode]
    symbols: _containers.RepeatedCompositeFieldContainer[SymbolNode]
    imports: _containers.RepeatedCompositeFieldContainer[ImportEdge]
    extraction_profiles: _containers.RepeatedScalarFieldContainer[str]
    omitted_information: _containers.RepeatedCompositeFieldContainer[_code_graph_pb2.CodeGraphOmission]
    def __init__(self, id: _Optional[str] = ..., scenario: _Optional[str] = ..., content_hash: _Optional[str] = ..., languages: _Optional[_Iterable[_Union[Language, str]]] = ..., extracted_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., extraction_ms: _Optional[int] = ..., files: _Optional[_Iterable[_Union[FileNode, _Mapping]]] = ..., packages: _Optional[_Iterable[_Union[PackageNode, _Mapping]]] = ..., symbols: _Optional[_Iterable[_Union[SymbolNode, _Mapping]]] = ..., imports: _Optional[_Iterable[_Union[ImportEdge, _Mapping]]] = ..., extraction_profiles: _Optional[_Iterable[str]] = ..., omitted_information: _Optional[_Iterable[_Union[_code_graph_pb2.CodeGraphOmission, _Mapping]]] = ...) -> None: ...

class ExtractGraphRequest(_message.Message):
    __slots__ = ("scenario", "languages", "idempotency_key")
    SCENARIO_FIELD_NUMBER: _ClassVar[int]
    LANGUAGES_FIELD_NUMBER: _ClassVar[int]
    IDEMPOTENCY_KEY_FIELD_NUMBER: _ClassVar[int]
    scenario: str
    languages: _containers.RepeatedScalarFieldContainer[Language]
    idempotency_key: str
    def __init__(self, scenario: _Optional[str] = ..., languages: _Optional[_Iterable[_Union[Language, str]]] = ..., idempotency_key: _Optional[str] = ...) -> None: ...

class ExtractGraphResponse(_message.Message):
    __slots__ = ("snapshot", "from_cache")
    SNAPSHOT_FIELD_NUMBER: _ClassVar[int]
    FROM_CACHE_FIELD_NUMBER: _ClassVar[int]
    snapshot: GraphSnapshot
    from_cache: bool
    def __init__(self, snapshot: _Optional[_Union[GraphSnapshot, _Mapping]] = ..., from_cache: _Optional[bool] = ...) -> None: ...

class GetGraphSnapshotRequest(_message.Message):
    __slots__ = ("id",)
    ID_FIELD_NUMBER: _ClassVar[int]
    id: str
    def __init__(self, id: _Optional[str] = ...) -> None: ...

class GetGraphSnapshotResponse(_message.Message):
    __slots__ = ("snapshot",)
    SNAPSHOT_FIELD_NUMBER: _ClassVar[int]
    snapshot: GraphSnapshot
    def __init__(self, snapshot: _Optional[_Union[GraphSnapshot, _Mapping]] = ...) -> None: ...

class ListGraphSnapshotsRequest(_message.Message):
    __slots__ = ("scenario", "page_size", "page_token")
    SCENARIO_FIELD_NUMBER: _ClassVar[int]
    PAGE_SIZE_FIELD_NUMBER: _ClassVar[int]
    PAGE_TOKEN_FIELD_NUMBER: _ClassVar[int]
    scenario: str
    page_size: int
    page_token: str
    def __init__(self, scenario: _Optional[str] = ..., page_size: _Optional[int] = ..., page_token: _Optional[str] = ...) -> None: ...

class ListGraphSnapshotsResponse(_message.Message):
    __slots__ = ("snapshots", "next_page_token")
    SNAPSHOTS_FIELD_NUMBER: _ClassVar[int]
    NEXT_PAGE_TOKEN_FIELD_NUMBER: _ClassVar[int]
    snapshots: _containers.RepeatedCompositeFieldContainer[GraphSnapshot]
    next_page_token: str
    def __init__(self, snapshots: _Optional[_Iterable[_Union[GraphSnapshot, _Mapping]]] = ..., next_page_token: _Optional[str] = ...) -> None: ...

class ClearGraphSnapshotsRequest(_message.Message):
    __slots__ = ("scenario", "dry_run")
    SCENARIO_FIELD_NUMBER: _ClassVar[int]
    DRY_RUN_FIELD_NUMBER: _ClassVar[int]
    scenario: str
    dry_run: bool
    def __init__(self, scenario: _Optional[str] = ..., dry_run: _Optional[bool] = ...) -> None: ...

class ClearGraphSnapshotsResponse(_message.Message):
    __slots__ = ("deleted", "dry_run")
    DELETED_FIELD_NUMBER: _ClassVar[int]
    DRY_RUN_FIELD_NUMBER: _ClassVar[int]
    deleted: int
    dry_run: bool
    def __init__(self, deleted: _Optional[int] = ..., dry_run: _Optional[bool] = ...) -> None: ...

class ExportGraphRequest(_message.Message):
    __slots__ = ("id",)
    ID_FIELD_NUMBER: _ClassVar[int]
    id: str
    def __init__(self, id: _Optional[str] = ...) -> None: ...

class ExportGraphResponse(_message.Message):
    __slots__ = ("payload", "content_type")
    PAYLOAD_FIELD_NUMBER: _ClassVar[int]
    CONTENT_TYPE_FIELD_NUMBER: _ClassVar[int]
    payload: bytes
    content_type: str
    def __init__(self, payload: _Optional[bytes] = ..., content_type: _Optional[str] = ...) -> None: ...

class GetZoneMapRequest(_message.Message):
    __slots__ = ("scenario", "snapshot_id")
    SCENARIO_FIELD_NUMBER: _ClassVar[int]
    SNAPSHOT_ID_FIELD_NUMBER: _ClassVar[int]
    scenario: str
    snapshot_id: str
    def __init__(self, scenario: _Optional[str] = ..., snapshot_id: _Optional[str] = ...) -> None: ...

class GetZoneMapResponse(_message.Message):
    __slots__ = ("zone_map",)
    ZONE_MAP_FIELD_NUMBER: _ClassVar[int]
    zone_map: ZoneMap
    def __init__(self, zone_map: _Optional[_Union[ZoneMap, _Mapping]] = ...) -> None: ...

class ZoneMap(_message.Message):
    __slots__ = ("scenario", "snapshot_id", "packages", "violations", "authority_confidence", "attestation")
    SCENARIO_FIELD_NUMBER: _ClassVar[int]
    SNAPSHOT_ID_FIELD_NUMBER: _ClassVar[int]
    PACKAGES_FIELD_NUMBER: _ClassVar[int]
    VIOLATIONS_FIELD_NUMBER: _ClassVar[int]
    AUTHORITY_CONFIDENCE_FIELD_NUMBER: _ClassVar[int]
    ATTESTATION_FIELD_NUMBER: _ClassVar[int]
    scenario: str
    snapshot_id: str
    packages: _containers.RepeatedCompositeFieldContainer[ZonePackage]
    violations: _containers.RepeatedCompositeFieldContainer[ZoneViolation]
    authority_confidence: AuthorityConfidence
    attestation: _attestation_pb2.AttestedAnswer
    def __init__(self, scenario: _Optional[str] = ..., snapshot_id: _Optional[str] = ..., packages: _Optional[_Iterable[_Union[ZonePackage, _Mapping]]] = ..., violations: _Optional[_Iterable[_Union[ZoneViolation, _Mapping]]] = ..., authority_confidence: _Optional[_Union[AuthorityConfidence, str]] = ..., attestation: _Optional[_Union[_attestation_pb2.AttestedAnswer, _Mapping]] = ...) -> None: ...

class ZonePackage(_message.Message):
    __slots__ = ("package_id", "import_path", "repo_path", "zone", "domain", "archetype", "declared", "confidence", "declared_layer", "evidence", "drift")
    PACKAGE_ID_FIELD_NUMBER: _ClassVar[int]
    IMPORT_PATH_FIELD_NUMBER: _ClassVar[int]
    REPO_PATH_FIELD_NUMBER: _ClassVar[int]
    ZONE_FIELD_NUMBER: _ClassVar[int]
    DOMAIN_FIELD_NUMBER: _ClassVar[int]
    ARCHETYPE_FIELD_NUMBER: _ClassVar[int]
    DECLARED_FIELD_NUMBER: _ClassVar[int]
    CONFIDENCE_FIELD_NUMBER: _ClassVar[int]
    DECLARED_LAYER_FIELD_NUMBER: _ClassVar[int]
    EVIDENCE_FIELD_NUMBER: _ClassVar[int]
    DRIFT_FIELD_NUMBER: _ClassVar[int]
    package_id: str
    import_path: str
    repo_path: str
    zone: str
    domain: str
    archetype: str
    declared: bool
    confidence: float
    declared_layer: str
    evidence: _containers.RepeatedCompositeFieldContainer[ZoneEvidence]
    drift: bool
    def __init__(self, package_id: _Optional[str] = ..., import_path: _Optional[str] = ..., repo_path: _Optional[str] = ..., zone: _Optional[str] = ..., domain: _Optional[str] = ..., archetype: _Optional[str] = ..., declared: _Optional[bool] = ..., confidence: _Optional[float] = ..., declared_layer: _Optional[str] = ..., evidence: _Optional[_Iterable[_Union[ZoneEvidence, _Mapping]]] = ..., drift: _Optional[bool] = ...) -> None: ...

class ZoneEvidence(_message.Message):
    __slots__ = ("kind", "detail", "locator")
    KIND_FIELD_NUMBER: _ClassVar[int]
    DETAIL_FIELD_NUMBER: _ClassVar[int]
    LOCATOR_FIELD_NUMBER: _ClassVar[int]
    kind: str
    detail: str
    locator: str
    def __init__(self, kind: _Optional[str] = ..., detail: _Optional[str] = ..., locator: _Optional[str] = ...) -> None: ...

class ZoneViolation(_message.Message):
    __slots__ = ("kind", "subtype", "severity", "locations", "domains", "summary")
    KIND_FIELD_NUMBER: _ClassVar[int]
    SUBTYPE_FIELD_NUMBER: _ClassVar[int]
    SEVERITY_FIELD_NUMBER: _ClassVar[int]
    LOCATIONS_FIELD_NUMBER: _ClassVar[int]
    DOMAINS_FIELD_NUMBER: _ClassVar[int]
    SUMMARY_FIELD_NUMBER: _ClassVar[int]
    kind: str
    subtype: str
    severity: _shared_pb2.Severity
    locations: _containers.RepeatedScalarFieldContainer[str]
    domains: _containers.RepeatedScalarFieldContainer[str]
    summary: str
    def __init__(self, kind: _Optional[str] = ..., subtype: _Optional[str] = ..., severity: _Optional[_Union[_shared_pb2.Severity, str]] = ..., locations: _Optional[_Iterable[str]] = ..., domains: _Optional[_Iterable[str]] = ..., summary: _Optional[str] = ...) -> None: ...

class GetSliceRequest(_message.Message):
    __slots__ = ("scenario", "domain", "snapshot_id")
    SCENARIO_FIELD_NUMBER: _ClassVar[int]
    DOMAIN_FIELD_NUMBER: _ClassVar[int]
    SNAPSHOT_ID_FIELD_NUMBER: _ClassVar[int]
    scenario: str
    domain: str
    snapshot_id: str
    def __init__(self, scenario: _Optional[str] = ..., domain: _Optional[str] = ..., snapshot_id: _Optional[str] = ...) -> None: ...

class GetSliceResponse(_message.Message):
    __slots__ = ("slice",)
    SLICE_FIELD_NUMBER: _ClassVar[int]
    slice: DomainSlice
    def __init__(self, slice: _Optional[_Union[DomainSlice, _Mapping]] = ...) -> None: ...

class DomainSlice(_message.Message):
    __slots__ = ("scenario", "domain", "snapshot_id", "rungs", "surfaces", "layer_edges", "attestation")
    SCENARIO_FIELD_NUMBER: _ClassVar[int]
    DOMAIN_FIELD_NUMBER: _ClassVar[int]
    SNAPSHOT_ID_FIELD_NUMBER: _ClassVar[int]
    RUNGS_FIELD_NUMBER: _ClassVar[int]
    SURFACES_FIELD_NUMBER: _ClassVar[int]
    LAYER_EDGES_FIELD_NUMBER: _ClassVar[int]
    ATTESTATION_FIELD_NUMBER: _ClassVar[int]
    scenario: str
    domain: str
    snapshot_id: str
    rungs: _containers.RepeatedCompositeFieldContainer[SliceRung]
    surfaces: _containers.RepeatedScalarFieldContainer[str]
    layer_edges: _containers.RepeatedCompositeFieldContainer[SliceEdge]
    attestation: _attestation_pb2.AttestedAnswer
    def __init__(self, scenario: _Optional[str] = ..., domain: _Optional[str] = ..., snapshot_id: _Optional[str] = ..., rungs: _Optional[_Iterable[_Union[SliceRung, _Mapping]]] = ..., surfaces: _Optional[_Iterable[str]] = ..., layer_edges: _Optional[_Iterable[_Union[SliceEdge, _Mapping]]] = ..., attestation: _Optional[_Union[_attestation_pb2.AttestedAnswer, _Mapping]] = ...) -> None: ...

class SliceRung(_message.Message):
    __slots__ = ("name", "present", "evidence", "files", "symbols")
    NAME_FIELD_NUMBER: _ClassVar[int]
    PRESENT_FIELD_NUMBER: _ClassVar[int]
    EVIDENCE_FIELD_NUMBER: _ClassVar[int]
    FILES_FIELD_NUMBER: _ClassVar[int]
    SYMBOLS_FIELD_NUMBER: _ClassVar[int]
    name: str
    present: bool
    evidence: _containers.RepeatedCompositeFieldContainer[SliceEvidence]
    files: _containers.RepeatedCompositeFieldContainer[SliceFile]
    symbols: _containers.RepeatedCompositeFieldContainer[SliceSymbol]
    def __init__(self, name: _Optional[str] = ..., present: _Optional[bool] = ..., evidence: _Optional[_Iterable[_Union[SliceEvidence, _Mapping]]] = ..., files: _Optional[_Iterable[_Union[SliceFile, _Mapping]]] = ..., symbols: _Optional[_Iterable[_Union[SliceSymbol, _Mapping]]] = ...) -> None: ...

class SliceEvidence(_message.Message):
    __slots__ = ("kind", "value", "source")
    KIND_FIELD_NUMBER: _ClassVar[int]
    VALUE_FIELD_NUMBER: _ClassVar[int]
    SOURCE_FIELD_NUMBER: _ClassVar[int]
    kind: str
    value: str
    source: str
    def __init__(self, kind: _Optional[str] = ..., value: _Optional[str] = ..., source: _Optional[str] = ...) -> None: ...

class SliceFile(_message.Message):
    __slots__ = ("path", "lines", "is_test")
    PATH_FIELD_NUMBER: _ClassVar[int]
    LINES_FIELD_NUMBER: _ClassVar[int]
    IS_TEST_FIELD_NUMBER: _ClassVar[int]
    path: str
    lines: int
    is_test: bool
    def __init__(self, path: _Optional[str] = ..., lines: _Optional[int] = ..., is_test: _Optional[bool] = ...) -> None: ...

class SliceSymbol(_message.Message):
    __slots__ = ("name", "kind", "file")
    NAME_FIELD_NUMBER: _ClassVar[int]
    KIND_FIELD_NUMBER: _ClassVar[int]
    FILE_FIELD_NUMBER: _ClassVar[int]
    name: str
    kind: str
    file: str
    def __init__(self, name: _Optional[str] = ..., kind: _Optional[str] = ..., file: _Optional[str] = ...) -> None: ...

class SliceEdge(_message.Message):
    __slots__ = ("from_rung", "to_rung", "kind")
    FROM_RUNG_FIELD_NUMBER: _ClassVar[int]
    TO_RUNG_FIELD_NUMBER: _ClassVar[int]
    KIND_FIELD_NUMBER: _ClassVar[int]
    from_rung: str
    to_rung: str
    kind: str
    def __init__(self, from_rung: _Optional[str] = ..., to_rung: _Optional[str] = ..., kind: _Optional[str] = ...) -> None: ...

class InferArchetypeRequest(_message.Message):
    __slots__ = ("scenario", "domain", "snapshot_id")
    SCENARIO_FIELD_NUMBER: _ClassVar[int]
    DOMAIN_FIELD_NUMBER: _ClassVar[int]
    SNAPSHOT_ID_FIELD_NUMBER: _ClassVar[int]
    scenario: str
    domain: str
    snapshot_id: str
    def __init__(self, scenario: _Optional[str] = ..., domain: _Optional[str] = ..., snapshot_id: _Optional[str] = ...) -> None: ...

class InferArchetypeResponse(_message.Message):
    __slots__ = ("scenario", "snapshot_id", "reports")
    SCENARIO_FIELD_NUMBER: _ClassVar[int]
    SNAPSHOT_ID_FIELD_NUMBER: _ClassVar[int]
    REPORTS_FIELD_NUMBER: _ClassVar[int]
    scenario: str
    snapshot_id: str
    reports: _containers.RepeatedCompositeFieldContainer[ArchetypeReport]
    def __init__(self, scenario: _Optional[str] = ..., snapshot_id: _Optional[str] = ..., reports: _Optional[_Iterable[_Union[ArchetypeReport, _Mapping]]] = ...) -> None: ...

class ArchetypeReport(_message.Message):
    __slots__ = ("domain", "archetypes", "convergence_drift", "attestation")
    DOMAIN_FIELD_NUMBER: _ClassVar[int]
    ARCHETYPES_FIELD_NUMBER: _ClassVar[int]
    CONVERGENCE_DRIFT_FIELD_NUMBER: _ClassVar[int]
    ATTESTATION_FIELD_NUMBER: _ClassVar[int]
    domain: str
    archetypes: _containers.RepeatedCompositeFieldContainer[_domains_pb2.DomainArchetype]
    convergence_drift: bool
    attestation: _attestation_pb2.AttestedAnswer
    def __init__(self, domain: _Optional[str] = ..., archetypes: _Optional[_Iterable[_Union[_domains_pb2.DomainArchetype, _Mapping]]] = ..., convergence_drift: _Optional[bool] = ..., attestation: _Optional[_Union[_attestation_pb2.AttestedAnswer, _Mapping]] = ...) -> None: ...

class ScenarioSnapshotCount(_message.Message):
    __slots__ = ("scenario", "snapshot_count", "reclaimable_count")
    SCENARIO_FIELD_NUMBER: _ClassVar[int]
    SNAPSHOT_COUNT_FIELD_NUMBER: _ClassVar[int]
    RECLAIMABLE_COUNT_FIELD_NUMBER: _ClassVar[int]
    scenario: str
    snapshot_count: int
    reclaimable_count: int
    def __init__(self, scenario: _Optional[str] = ..., snapshot_count: _Optional[int] = ..., reclaimable_count: _Optional[int] = ...) -> None: ...

class PreviewSnapshotRetentionRequest(_message.Message):
    __slots__ = ("keep_per_scenario",)
    KEEP_PER_SCENARIO_FIELD_NUMBER: _ClassVar[int]
    keep_per_scenario: int
    def __init__(self, keep_per_scenario: _Optional[int] = ...) -> None: ...

class PreviewSnapshotRetentionResponse(_message.Message):
    __slots__ = ("reclaimable_bytes", "reclaimable_rows", "keep_per_scenario", "total_snapshots", "scenarios")
    RECLAIMABLE_BYTES_FIELD_NUMBER: _ClassVar[int]
    RECLAIMABLE_ROWS_FIELD_NUMBER: _ClassVar[int]
    KEEP_PER_SCENARIO_FIELD_NUMBER: _ClassVar[int]
    TOTAL_SNAPSHOTS_FIELD_NUMBER: _ClassVar[int]
    SCENARIOS_FIELD_NUMBER: _ClassVar[int]
    reclaimable_bytes: int
    reclaimable_rows: int
    keep_per_scenario: int
    total_snapshots: int
    scenarios: _containers.RepeatedCompositeFieldContainer[ScenarioSnapshotCount]
    def __init__(self, reclaimable_bytes: _Optional[int] = ..., reclaimable_rows: _Optional[int] = ..., keep_per_scenario: _Optional[int] = ..., total_snapshots: _Optional[int] = ..., scenarios: _Optional[_Iterable[_Union[ScenarioSnapshotCount, _Mapping]]] = ...) -> None: ...

class ApplySnapshotRetentionRequest(_message.Message):
    __slots__ = ("keep_per_scenario", "confirm")
    KEEP_PER_SCENARIO_FIELD_NUMBER: _ClassVar[int]
    CONFIRM_FIELD_NUMBER: _ClassVar[int]
    keep_per_scenario: int
    confirm: bool
    def __init__(self, keep_per_scenario: _Optional[int] = ..., confirm: _Optional[bool] = ...) -> None: ...

class ApplySnapshotRetentionResponse(_message.Message):
    __slots__ = ("rows_removed", "bytes_reclaimed", "pages_freed", "scenarios_scanned", "keep_per_scenario")
    ROWS_REMOVED_FIELD_NUMBER: _ClassVar[int]
    BYTES_RECLAIMED_FIELD_NUMBER: _ClassVar[int]
    PAGES_FREED_FIELD_NUMBER: _ClassVar[int]
    SCENARIOS_SCANNED_FIELD_NUMBER: _ClassVar[int]
    KEEP_PER_SCENARIO_FIELD_NUMBER: _ClassVar[int]
    rows_removed: int
    bytes_reclaimed: int
    pages_freed: int
    scenarios_scanned: int
    keep_per_scenario: int
    def __init__(self, rows_removed: _Optional[int] = ..., bytes_reclaimed: _Optional[int] = ..., pages_freed: _Optional[int] = ..., scenarios_scanned: _Optional[int] = ..., keep_per_scenario: _Optional[int] = ...) -> None: ...
