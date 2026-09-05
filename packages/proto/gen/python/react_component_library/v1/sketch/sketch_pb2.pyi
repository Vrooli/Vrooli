from google.protobuf import struct_pb2 as _struct_pb2
from react_component_library.v1.preview import preview_pb2 as _preview_pb2
from google.protobuf.internal import containers as _containers
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Iterable as _Iterable, Mapping as _Mapping
from typing import ClassVar as _ClassVar, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class SketchTarget(_message.Message):
    __slots__ = ("scenario", "page")
    SCENARIO_FIELD_NUMBER: _ClassVar[int]
    PAGE_FIELD_NUMBER: _ClassVar[int]
    scenario: str
    page: str
    def __init__(self, scenario: _Optional[str] = ..., page: _Optional[str] = ...) -> None: ...

class AssetReference(_message.Message):
    __slots__ = ("asset", "version")
    ASSET_FIELD_NUMBER: _ClassVar[int]
    VERSION_FIELD_NUMBER: _ClassVar[int]
    asset: str
    version: str
    def __init__(self, asset: _Optional[str] = ..., version: _Optional[str] = ...) -> None: ...

class Fill(_message.Message):
    __slots__ = ("asset", "version", "placeholder", "intent")
    ASSET_FIELD_NUMBER: _ClassVar[int]
    VERSION_FIELD_NUMBER: _ClassVar[int]
    PLACEHOLDER_FIELD_NUMBER: _ClassVar[int]
    INTENT_FIELD_NUMBER: _ClassVar[int]
    asset: str
    version: str
    placeholder: str
    intent: str
    def __init__(self, asset: _Optional[str] = ..., version: _Optional[str] = ..., placeholder: _Optional[str] = ..., intent: _Optional[str] = ...) -> None: ...

class Grid(_message.Message):
    __slots__ = ("x", "y", "w", "h")
    X_FIELD_NUMBER: _ClassVar[int]
    Y_FIELD_NUMBER: _ClassVar[int]
    W_FIELD_NUMBER: _ClassVar[int]
    H_FIELD_NUMBER: _ClassVar[int]
    x: int
    y: int
    w: int
    h: int
    def __init__(self, x: _Optional[int] = ..., y: _Optional[int] = ..., w: _Optional[int] = ..., h: _Optional[int] = ...) -> None: ...

class SketchRegion(_message.Message):
    __slots__ = ("locked", "origin", "id", "note", "elements", "grid")
    LOCKED_FIELD_NUMBER: _ClassVar[int]
    ORIGIN_FIELD_NUMBER: _ClassVar[int]
    ID_FIELD_NUMBER: _ClassVar[int]
    NOTE_FIELD_NUMBER: _ClassVar[int]
    ELEMENTS_FIELD_NUMBER: _ClassVar[int]
    GRID_FIELD_NUMBER: _ClassVar[int]
    locked: bool
    origin: str
    id: str
    note: str
    elements: _containers.RepeatedScalarFieldContainer[str]
    grid: Grid
    def __init__(self, locked: _Optional[bool] = ..., origin: _Optional[str] = ..., id: _Optional[str] = ..., note: _Optional[str] = ..., elements: _Optional[_Iterable[str]] = ..., grid: _Optional[_Union[Grid, _Mapping]] = ...) -> None: ...

class Placement(_message.Message):
    __slots__ = ("region", "fills", "state", "note", "intent")
    REGION_FIELD_NUMBER: _ClassVar[int]
    FILLS_FIELD_NUMBER: _ClassVar[int]
    STATE_FIELD_NUMBER: _ClassVar[int]
    NOTE_FIELD_NUMBER: _ClassVar[int]
    INTENT_FIELD_NUMBER: _ClassVar[int]
    region: str
    fills: Fill
    state: str
    note: str
    intent: str
    def __init__(self, region: _Optional[str] = ..., fills: _Optional[_Union[Fill, _Mapping]] = ..., state: _Optional[str] = ..., note: _Optional[str] = ..., intent: _Optional[str] = ...) -> None: ...

class UnplacedItem(_message.Message):
    __slots__ = ("was", "reason", "region", "placement")
    WAS_FIELD_NUMBER: _ClassVar[int]
    REASON_FIELD_NUMBER: _ClassVar[int]
    REGION_FIELD_NUMBER: _ClassVar[int]
    PLACEMENT_FIELD_NUMBER: _ClassVar[int]
    was: str
    reason: str
    region: str
    placement: Placement
    def __init__(self, was: _Optional[str] = ..., reason: _Optional[str] = ..., region: _Optional[str] = ..., placement: _Optional[_Union[Placement, _Mapping]] = ...) -> None: ...

class SketchNote(_message.Message):
    __slots__ = ("scope", "text")
    SCOPE_FIELD_NUMBER: _ClassVar[int]
    TEXT_FIELD_NUMBER: _ClassVar[int]
    scope: str
    text: str
    def __init__(self, scope: _Optional[str] = ..., text: _Optional[str] = ...) -> None: ...

class RenderRegion(_message.Message):
    __slots__ = ("parent", "story", "template_region", "id", "export", "slot", "optional")
    PARENT_FIELD_NUMBER: _ClassVar[int]
    STORY_FIELD_NUMBER: _ClassVar[int]
    TEMPLATE_REGION_FIELD_NUMBER: _ClassVar[int]
    ID_FIELD_NUMBER: _ClassVar[int]
    EXPORT_FIELD_NUMBER: _ClassVar[int]
    SLOT_FIELD_NUMBER: _ClassVar[int]
    OPTIONAL_FIELD_NUMBER: _ClassVar[int]
    parent: str
    story: str
    template_region: str
    id: str
    export: str
    slot: _containers.RepeatedScalarFieldContainer[str]
    optional: bool
    def __init__(self, parent: _Optional[str] = ..., story: _Optional[str] = ..., template_region: _Optional[str] = ..., id: _Optional[str] = ..., export: _Optional[str] = ..., slot: _Optional[_Iterable[str]] = ..., optional: _Optional[bool] = ...) -> None: ...

class RenderFixture(_message.Message):
    __slots__ = ("target", "asset", "version", "state", "field", "prop")
    TARGET_FIELD_NUMBER: _ClassVar[int]
    ASSET_FIELD_NUMBER: _ClassVar[int]
    VERSION_FIELD_NUMBER: _ClassVar[int]
    STATE_FIELD_NUMBER: _ClassVar[int]
    FIELD_FIELD_NUMBER: _ClassVar[int]
    PROP_FIELD_NUMBER: _ClassVar[int]
    target: str
    asset: str
    version: str
    state: str
    field: str
    prop: _containers.RepeatedScalarFieldContainer[str]
    def __init__(self, target: _Optional[str] = ..., asset: _Optional[str] = ..., version: _Optional[str] = ..., state: _Optional[str] = ..., field: _Optional[str] = ..., prop: _Optional[_Iterable[str]] = ...) -> None: ...

class RenderSettings(_message.Message):
    __slots__ = ("template_export", "regions", "bindings", "fixtures")
    TEMPLATE_EXPORT_FIELD_NUMBER: _ClassVar[int]
    REGIONS_FIELD_NUMBER: _ClassVar[int]
    BINDINGS_FIELD_NUMBER: _ClassVar[int]
    FIXTURES_FIELD_NUMBER: _ClassVar[int]
    template_export: str
    regions: _containers.RepeatedCompositeFieldContainer[RenderRegion]
    bindings: _struct_pb2.Struct
    fixtures: _containers.RepeatedCompositeFieldContainer[RenderFixture]
    def __init__(self, template_export: _Optional[str] = ..., regions: _Optional[_Iterable[_Union[RenderRegion, _Mapping]]] = ..., bindings: _Optional[_Union[_struct_pb2.Struct, _Mapping]] = ..., fixtures: _Optional[_Iterable[_Union[RenderFixture, _Mapping]]] = ...) -> None: ...

class Sketch(_message.Message):
    __slots__ = ("viewport", "template", "placements", "unplaced", "notes", "regions", "render", "intent")
    VIEWPORT_FIELD_NUMBER: _ClassVar[int]
    TEMPLATE_FIELD_NUMBER: _ClassVar[int]
    PLACEMENTS_FIELD_NUMBER: _ClassVar[int]
    UNPLACED_FIELD_NUMBER: _ClassVar[int]
    NOTES_FIELD_NUMBER: _ClassVar[int]
    REGIONS_FIELD_NUMBER: _ClassVar[int]
    RENDER_FIELD_NUMBER: _ClassVar[int]
    INTENT_FIELD_NUMBER: _ClassVar[int]
    viewport: str
    template: AssetReference
    placements: _containers.RepeatedCompositeFieldContainer[Placement]
    unplaced: _containers.RepeatedCompositeFieldContainer[UnplacedItem]
    notes: _containers.RepeatedCompositeFieldContainer[SketchNote]
    regions: _containers.RepeatedCompositeFieldContainer[SketchRegion]
    render: RenderSettings
    intent: DesignIntent
    def __init__(self, viewport: _Optional[str] = ..., template: _Optional[_Union[AssetReference, _Mapping]] = ..., placements: _Optional[_Iterable[_Union[Placement, _Mapping]]] = ..., unplaced: _Optional[_Iterable[_Union[UnplacedItem, _Mapping]]] = ..., notes: _Optional[_Iterable[_Union[SketchNote, _Mapping]]] = ..., regions: _Optional[_Iterable[_Union[SketchRegion, _Mapping]]] = ..., render: _Optional[_Union[RenderSettings, _Mapping]] = ..., intent: _Optional[_Union[DesignIntent, _Mapping]] = ...) -> None: ...

class GetSketchRequest(_message.Message):
    __slots__ = ("target",)
    TARGET_FIELD_NUMBER: _ClassVar[int]
    target: SketchTarget
    def __init__(self, target: _Optional[_Union[SketchTarget, _Mapping]] = ...) -> None: ...

class GetSketchResponse(_message.Message):
    __slots__ = ("sketch", "document_path", "content_hash", "declared_regions")
    SKETCH_FIELD_NUMBER: _ClassVar[int]
    DOCUMENT_PATH_FIELD_NUMBER: _ClassVar[int]
    CONTENT_HASH_FIELD_NUMBER: _ClassVar[int]
    DECLARED_REGIONS_FIELD_NUMBER: _ClassVar[int]
    sketch: Sketch
    document_path: str
    content_hash: str
    declared_regions: _containers.RepeatedCompositeFieldContainer[SketchRegion]
    def __init__(self, sketch: _Optional[_Union[Sketch, _Mapping]] = ..., document_path: _Optional[str] = ..., content_hash: _Optional[str] = ..., declared_regions: _Optional[_Iterable[_Union[SketchRegion, _Mapping]]] = ...) -> None: ...

class PutSketchRequest(_message.Message):
    __slots__ = ("target", "sketch", "expected_content_hash")
    TARGET_FIELD_NUMBER: _ClassVar[int]
    SKETCH_FIELD_NUMBER: _ClassVar[int]
    EXPECTED_CONTENT_HASH_FIELD_NUMBER: _ClassVar[int]
    target: SketchTarget
    sketch: Sketch
    expected_content_hash: str
    def __init__(self, target: _Optional[_Union[SketchTarget, _Mapping]] = ..., sketch: _Optional[_Union[Sketch, _Mapping]] = ..., expected_content_hash: _Optional[str] = ...) -> None: ...

class PutSketchResponse(_message.Message):
    __slots__ = ("sketch", "document_path", "changed", "content_hash")
    SKETCH_FIELD_NUMBER: _ClassVar[int]
    DOCUMENT_PATH_FIELD_NUMBER: _ClassVar[int]
    CHANGED_FIELD_NUMBER: _ClassVar[int]
    CONTENT_HASH_FIELD_NUMBER: _ClassVar[int]
    sketch: Sketch
    document_path: str
    changed: bool
    content_hash: str
    def __init__(self, sketch: _Optional[_Union[Sketch, _Mapping]] = ..., document_path: _Optional[str] = ..., changed: _Optional[bool] = ..., content_hash: _Optional[str] = ...) -> None: ...

class Finding(_message.Message):
    __slots__ = ("catalog_id", "scope", "blocking", "owner", "severity_class", "message")
    CATALOG_ID_FIELD_NUMBER: _ClassVar[int]
    SCOPE_FIELD_NUMBER: _ClassVar[int]
    BLOCKING_FIELD_NUMBER: _ClassVar[int]
    OWNER_FIELD_NUMBER: _ClassVar[int]
    SEVERITY_CLASS_FIELD_NUMBER: _ClassVar[int]
    MESSAGE_FIELD_NUMBER: _ClassVar[int]
    catalog_id: str
    scope: str
    blocking: bool
    owner: str
    severity_class: str
    message: str
    def __init__(self, catalog_id: _Optional[str] = ..., scope: _Optional[str] = ..., blocking: _Optional[bool] = ..., owner: _Optional[str] = ..., severity_class: _Optional[str] = ..., message: _Optional[str] = ...) -> None: ...

class RegionVerdict(_message.Message):
    __slots__ = ("region", "verdict", "join_rule", "proven", "file_path", "observed_state", "reason", "finding", "selected_asset", "selected_version", "observed_asset", "observed_version", "reason_code", "evidence_quality", "candidates", "availability_state", "availability_reason_code", "build_hash", "source_hash")
    REGION_FIELD_NUMBER: _ClassVar[int]
    VERDICT_FIELD_NUMBER: _ClassVar[int]
    JOIN_RULE_FIELD_NUMBER: _ClassVar[int]
    PROVEN_FIELD_NUMBER: _ClassVar[int]
    FILE_PATH_FIELD_NUMBER: _ClassVar[int]
    OBSERVED_STATE_FIELD_NUMBER: _ClassVar[int]
    REASON_FIELD_NUMBER: _ClassVar[int]
    FINDING_FIELD_NUMBER: _ClassVar[int]
    SELECTED_ASSET_FIELD_NUMBER: _ClassVar[int]
    SELECTED_VERSION_FIELD_NUMBER: _ClassVar[int]
    OBSERVED_ASSET_FIELD_NUMBER: _ClassVar[int]
    OBSERVED_VERSION_FIELD_NUMBER: _ClassVar[int]
    REASON_CODE_FIELD_NUMBER: _ClassVar[int]
    EVIDENCE_QUALITY_FIELD_NUMBER: _ClassVar[int]
    CANDIDATES_FIELD_NUMBER: _ClassVar[int]
    AVAILABILITY_STATE_FIELD_NUMBER: _ClassVar[int]
    AVAILABILITY_REASON_CODE_FIELD_NUMBER: _ClassVar[int]
    BUILD_HASH_FIELD_NUMBER: _ClassVar[int]
    SOURCE_HASH_FIELD_NUMBER: _ClassVar[int]
    region: str
    verdict: str
    join_rule: str
    proven: bool
    file_path: str
    observed_state: str
    reason: str
    finding: Finding
    selected_asset: str
    selected_version: str
    observed_asset: str
    observed_version: str
    reason_code: str
    evidence_quality: str
    candidates: _containers.RepeatedScalarFieldContainer[str]
    availability_state: str
    availability_reason_code: str
    build_hash: str
    source_hash: str
    def __init__(self, region: _Optional[str] = ..., verdict: _Optional[str] = ..., join_rule: _Optional[str] = ..., proven: _Optional[bool] = ..., file_path: _Optional[str] = ..., observed_state: _Optional[str] = ..., reason: _Optional[str] = ..., finding: _Optional[_Union[Finding, _Mapping]] = ..., selected_asset: _Optional[str] = ..., selected_version: _Optional[str] = ..., observed_asset: _Optional[str] = ..., observed_version: _Optional[str] = ..., reason_code: _Optional[str] = ..., evidence_quality: _Optional[str] = ..., candidates: _Optional[_Iterable[str]] = ..., availability_state: _Optional[str] = ..., availability_reason_code: _Optional[str] = ..., build_hash: _Optional[str] = ..., source_hash: _Optional[str] = ...) -> None: ...

class Coverage(_message.Message):
    __slots__ = ("built", "declared", "invented", "built_percent", "missing", "unresolved", "total", "status")
    BUILT_FIELD_NUMBER: _ClassVar[int]
    DECLARED_FIELD_NUMBER: _ClassVar[int]
    INVENTED_FIELD_NUMBER: _ClassVar[int]
    BUILT_PERCENT_FIELD_NUMBER: _ClassVar[int]
    MISSING_FIELD_NUMBER: _ClassVar[int]
    UNRESOLVED_FIELD_NUMBER: _ClassVar[int]
    TOTAL_FIELD_NUMBER: _ClassVar[int]
    STATUS_FIELD_NUMBER: _ClassVar[int]
    built: int
    declared: int
    invented: int
    built_percent: float
    missing: int
    unresolved: int
    total: int
    status: str
    def __init__(self, built: _Optional[int] = ..., declared: _Optional[int] = ..., invented: _Optional[int] = ..., built_percent: _Optional[float] = ..., missing: _Optional[int] = ..., unresolved: _Optional[int] = ..., total: _Optional[int] = ..., status: _Optional[str] = ...) -> None: ...

class VerifySketchRequest(_message.Message):
    __slots__ = ("target",)
    TARGET_FIELD_NUMBER: _ClassVar[int]
    target: SketchTarget
    def __init__(self, target: _Optional[_Union[SketchTarget, _Mapping]] = ...) -> None: ...

class VerifySketchResponse(_message.Message):
    __slots__ = ("regions", "coverage", "findings", "passes")
    REGIONS_FIELD_NUMBER: _ClassVar[int]
    COVERAGE_FIELD_NUMBER: _ClassVar[int]
    FINDINGS_FIELD_NUMBER: _ClassVar[int]
    PASSES_FIELD_NUMBER: _ClassVar[int]
    regions: _containers.RepeatedCompositeFieldContainer[RegionVerdict]
    coverage: Coverage
    findings: _containers.RepeatedCompositeFieldContainer[Finding]
    passes: bool
    def __init__(self, regions: _Optional[_Iterable[_Union[RegionVerdict, _Mapping]]] = ..., coverage: _Optional[_Union[Coverage, _Mapping]] = ..., findings: _Optional[_Iterable[_Union[Finding, _Mapping]]] = ..., passes: _Optional[bool] = ...) -> None: ...

class TierCheck(_message.Message):
    __slots__ = ("tier", "ran", "matches")
    TIER_FIELD_NUMBER: _ClassVar[int]
    RAN_FIELD_NUMBER: _ClassVar[int]
    MATCHES_FIELD_NUMBER: _ClassVar[int]
    tier: int
    ran: bool
    matches: _containers.RepeatedScalarFieldContainer[str]
    def __init__(self, tier: _Optional[int] = ..., ran: _Optional[bool] = ..., matches: _Optional[_Iterable[str]] = ...) -> None: ...

class DecompositionItem(_message.Message):
    __slots__ = ("component", "tier", "match", "checks", "state")
    COMPONENT_FIELD_NUMBER: _ClassVar[int]
    TIER_FIELD_NUMBER: _ClassVar[int]
    MATCH_FIELD_NUMBER: _ClassVar[int]
    CHECKS_FIELD_NUMBER: _ClassVar[int]
    STATE_FIELD_NUMBER: _ClassVar[int]
    component: str
    tier: int
    match: str
    checks: _containers.RepeatedCompositeFieldContainer[TierCheck]
    state: str
    def __init__(self, component: _Optional[str] = ..., tier: _Optional[int] = ..., match: _Optional[str] = ..., checks: _Optional[_Iterable[_Union[TierCheck, _Mapping]]] = ..., state: _Optional[str] = ...) -> None: ...

class TemplateCandidate(_message.Message):
    __slots__ = ("asset", "score", "implemented", "matching_regions")
    ASSET_FIELD_NUMBER: _ClassVar[int]
    SCORE_FIELD_NUMBER: _ClassVar[int]
    IMPLEMENTED_FIELD_NUMBER: _ClassVar[int]
    MATCHING_REGIONS_FIELD_NUMBER: _ClassVar[int]
    asset: str
    score: float
    implemented: bool
    matching_regions: _containers.RepeatedScalarFieldContainer[str]
    def __init__(self, asset: _Optional[str] = ..., score: _Optional[float] = ..., implemented: _Optional[bool] = ..., matching_regions: _Optional[_Iterable[str]] = ...) -> None: ...

class ImportPageRequest(_message.Message):
    __slots__ = ("target", "write", "expected_content_hash")
    TARGET_FIELD_NUMBER: _ClassVar[int]
    WRITE_FIELD_NUMBER: _ClassVar[int]
    EXPECTED_CONTENT_HASH_FIELD_NUMBER: _ClassVar[int]
    target: SketchTarget
    write: bool
    expected_content_hash: str
    def __init__(self, target: _Optional[_Union[SketchTarget, _Mapping]] = ..., write: _Optional[bool] = ..., expected_content_hash: _Optional[str] = ...) -> None: ...

class ImportPageResponse(_message.Message):
    __slots__ = ("sketch", "items", "templates", "written", "document_path", "content_hash")
    SKETCH_FIELD_NUMBER: _ClassVar[int]
    ITEMS_FIELD_NUMBER: _ClassVar[int]
    TEMPLATES_FIELD_NUMBER: _ClassVar[int]
    WRITTEN_FIELD_NUMBER: _ClassVar[int]
    DOCUMENT_PATH_FIELD_NUMBER: _ClassVar[int]
    CONTENT_HASH_FIELD_NUMBER: _ClassVar[int]
    sketch: Sketch
    items: _containers.RepeatedCompositeFieldContainer[DecompositionItem]
    templates: _containers.RepeatedCompositeFieldContainer[TemplateCandidate]
    written: bool
    document_path: str
    content_hash: str
    def __init__(self, sketch: _Optional[_Union[Sketch, _Mapping]] = ..., items: _Optional[_Iterable[_Union[DecompositionItem, _Mapping]]] = ..., templates: _Optional[_Iterable[_Union[TemplateCandidate, _Mapping]]] = ..., written: _Optional[bool] = ..., document_path: _Optional[str] = ..., content_hash: _Optional[str] = ...) -> None: ...

class PlaceRequest(_message.Message):
    __slots__ = ("target", "region", "asset", "version", "note", "expected_content_hash")
    TARGET_FIELD_NUMBER: _ClassVar[int]
    REGION_FIELD_NUMBER: _ClassVar[int]
    ASSET_FIELD_NUMBER: _ClassVar[int]
    VERSION_FIELD_NUMBER: _ClassVar[int]
    NOTE_FIELD_NUMBER: _ClassVar[int]
    EXPECTED_CONTENT_HASH_FIELD_NUMBER: _ClassVar[int]
    target: SketchTarget
    region: str
    asset: str
    version: str
    note: str
    expected_content_hash: str
    def __init__(self, target: _Optional[_Union[SketchTarget, _Mapping]] = ..., region: _Optional[str] = ..., asset: _Optional[str] = ..., version: _Optional[str] = ..., note: _Optional[str] = ..., expected_content_hash: _Optional[str] = ...) -> None: ...

class PlaceholderRequest(_message.Message):
    __slots__ = ("target", "region", "placeholder", "intent", "note", "expected_content_hash")
    TARGET_FIELD_NUMBER: _ClassVar[int]
    REGION_FIELD_NUMBER: _ClassVar[int]
    PLACEHOLDER_FIELD_NUMBER: _ClassVar[int]
    INTENT_FIELD_NUMBER: _ClassVar[int]
    NOTE_FIELD_NUMBER: _ClassVar[int]
    EXPECTED_CONTENT_HASH_FIELD_NUMBER: _ClassVar[int]
    target: SketchTarget
    region: str
    placeholder: str
    intent: str
    note: str
    expected_content_hash: str
    def __init__(self, target: _Optional[_Union[SketchTarget, _Mapping]] = ..., region: _Optional[str] = ..., placeholder: _Optional[str] = ..., intent: _Optional[str] = ..., note: _Optional[str] = ..., expected_content_hash: _Optional[str] = ...) -> None: ...

class AddNoteRequest(_message.Message):
    __slots__ = ("target", "scope", "text", "expected_content_hash")
    TARGET_FIELD_NUMBER: _ClassVar[int]
    SCOPE_FIELD_NUMBER: _ClassVar[int]
    TEXT_FIELD_NUMBER: _ClassVar[int]
    EXPECTED_CONTENT_HASH_FIELD_NUMBER: _ClassVar[int]
    target: SketchTarget
    scope: str
    text: str
    expected_content_hash: str
    def __init__(self, target: _Optional[_Union[SketchTarget, _Mapping]] = ..., scope: _Optional[str] = ..., text: _Optional[str] = ..., expected_content_hash: _Optional[str] = ...) -> None: ...

class UnplaceRequest(_message.Message):
    __slots__ = ("target", "region", "reason", "expected_content_hash")
    TARGET_FIELD_NUMBER: _ClassVar[int]
    REGION_FIELD_NUMBER: _ClassVar[int]
    REASON_FIELD_NUMBER: _ClassVar[int]
    EXPECTED_CONTENT_HASH_FIELD_NUMBER: _ClassVar[int]
    target: SketchTarget
    region: str
    reason: str
    expected_content_hash: str
    def __init__(self, target: _Optional[_Union[SketchTarget, _Mapping]] = ..., region: _Optional[str] = ..., reason: _Optional[str] = ..., expected_content_hash: _Optional[str] = ...) -> None: ...

class RegionRemap(_message.Message):
    __slots__ = ("to",)
    FROM_FIELD_NUMBER: _ClassVar[int]
    TO_FIELD_NUMBER: _ClassVar[int]
    to: str
    def __init__(self, to: _Optional[str] = ..., **kwargs) -> None: ...

class SetTemplateRequest(_message.Message):
    __slots__ = ("target", "asset", "version", "remap", "confirm_unplaced", "expected_content_hash", "preview")
    TARGET_FIELD_NUMBER: _ClassVar[int]
    ASSET_FIELD_NUMBER: _ClassVar[int]
    VERSION_FIELD_NUMBER: _ClassVar[int]
    REMAP_FIELD_NUMBER: _ClassVar[int]
    CONFIRM_UNPLACED_FIELD_NUMBER: _ClassVar[int]
    EXPECTED_CONTENT_HASH_FIELD_NUMBER: _ClassVar[int]
    PREVIEW_FIELD_NUMBER: _ClassVar[int]
    target: SketchTarget
    asset: str
    version: str
    remap: _containers.RepeatedCompositeFieldContainer[RegionRemap]
    confirm_unplaced: bool
    expected_content_hash: str
    preview: bool
    def __init__(self, target: _Optional[_Union[SketchTarget, _Mapping]] = ..., asset: _Optional[str] = ..., version: _Optional[str] = ..., remap: _Optional[_Iterable[_Union[RegionRemap, _Mapping]]] = ..., confirm_unplaced: _Optional[bool] = ..., expected_content_hash: _Optional[str] = ..., preview: _Optional[bool] = ...) -> None: ...

class SetTemplateResponse(_message.Message):
    __slots__ = ("requires_confirmation", "sketch", "newly_unplaced", "document_path", "changed", "content_hash")
    REQUIRES_CONFIRMATION_FIELD_NUMBER: _ClassVar[int]
    SKETCH_FIELD_NUMBER: _ClassVar[int]
    NEWLY_UNPLACED_FIELD_NUMBER: _ClassVar[int]
    DOCUMENT_PATH_FIELD_NUMBER: _ClassVar[int]
    CHANGED_FIELD_NUMBER: _ClassVar[int]
    CONTENT_HASH_FIELD_NUMBER: _ClassVar[int]
    requires_confirmation: bool
    sketch: Sketch
    newly_unplaced: _containers.RepeatedCompositeFieldContainer[UnplacedItem]
    document_path: str
    changed: bool
    content_hash: str
    def __init__(self, requires_confirmation: _Optional[bool] = ..., sketch: _Optional[_Union[Sketch, _Mapping]] = ..., newly_unplaced: _Optional[_Iterable[_Union[UnplacedItem, _Mapping]]] = ..., document_path: _Optional[str] = ..., changed: _Optional[bool] = ..., content_hash: _Optional[str] = ...) -> None: ...

class BriefItem(_message.Message):
    __slots__ = ("order", "tier", "region", "action", "asset", "constraint")
    ORDER_FIELD_NUMBER: _ClassVar[int]
    TIER_FIELD_NUMBER: _ClassVar[int]
    REGION_FIELD_NUMBER: _ClassVar[int]
    ACTION_FIELD_NUMBER: _ClassVar[int]
    ASSET_FIELD_NUMBER: _ClassVar[int]
    CONSTRAINT_FIELD_NUMBER: _ClassVar[int]
    order: int
    tier: int
    region: str
    action: str
    asset: str
    constraint: str
    def __init__(self, order: _Optional[int] = ..., tier: _Optional[int] = ..., region: _Optional[str] = ..., action: _Optional[str] = ..., asset: _Optional[str] = ..., constraint: _Optional[str] = ...) -> None: ...

class BuildBriefRequest(_message.Message):
    __slots__ = ("target",)
    TARGET_FIELD_NUMBER: _ClassVar[int]
    target: SketchTarget
    def __init__(self, target: _Optional[_Union[SketchTarget, _Mapping]] = ...) -> None: ...

class BuildBriefResponse(_message.Message):
    __slots__ = ("markdown", "items", "closing_gate")
    MARKDOWN_FIELD_NUMBER: _ClassVar[int]
    ITEMS_FIELD_NUMBER: _ClassVar[int]
    CLOSING_GATE_FIELD_NUMBER: _ClassVar[int]
    markdown: str
    items: _containers.RepeatedCompositeFieldContainer[BriefItem]
    closing_gate: str
    def __init__(self, markdown: _Optional[str] = ..., items: _Optional[_Iterable[_Union[BriefItem, _Mapping]]] = ..., closing_gate: _Optional[str] = ...) -> None: ...

class RevisionConflict(_message.Message):
    __slots__ = ("expected_content_hash", "current_content_hash")
    EXPECTED_CONTENT_HASH_FIELD_NUMBER: _ClassVar[int]
    CURRENT_CONTENT_HASH_FIELD_NUMBER: _ClassVar[int]
    expected_content_hash: str
    current_content_hash: str
    def __init__(self, expected_content_hash: _Optional[str] = ..., current_content_hash: _Optional[str] = ...) -> None: ...

class GetHistoryRequest(_message.Message):
    __slots__ = ("target",)
    TARGET_FIELD_NUMBER: _ClassVar[int]
    target: SketchTarget
    def __init__(self, target: _Optional[_Union[SketchTarget, _Mapping]] = ...) -> None: ...

class SketchRevision(_message.Message):
    __slots__ = ("content_hash", "created_from_hash", "created_at", "sketch", "current", "declared_regions")
    CONTENT_HASH_FIELD_NUMBER: _ClassVar[int]
    CREATED_FROM_HASH_FIELD_NUMBER: _ClassVar[int]
    CREATED_AT_FIELD_NUMBER: _ClassVar[int]
    SKETCH_FIELD_NUMBER: _ClassVar[int]
    CURRENT_FIELD_NUMBER: _ClassVar[int]
    DECLARED_REGIONS_FIELD_NUMBER: _ClassVar[int]
    content_hash: str
    created_from_hash: str
    created_at: str
    sketch: Sketch
    current: bool
    declared_regions: _containers.RepeatedCompositeFieldContainer[SketchRegion]
    def __init__(self, content_hash: _Optional[str] = ..., created_from_hash: _Optional[str] = ..., created_at: _Optional[str] = ..., sketch: _Optional[_Union[Sketch, _Mapping]] = ..., current: _Optional[bool] = ..., declared_regions: _Optional[_Iterable[_Union[SketchRegion, _Mapping]]] = ...) -> None: ...

class GetHistoryResponse(_message.Message):
    __slots__ = ("revisions", "current_content_hash")
    REVISIONS_FIELD_NUMBER: _ClassVar[int]
    CURRENT_CONTENT_HASH_FIELD_NUMBER: _ClassVar[int]
    revisions: _containers.RepeatedCompositeFieldContainer[SketchRevision]
    current_content_hash: str
    def __init__(self, revisions: _Optional[_Iterable[_Union[SketchRevision, _Mapping]]] = ..., current_content_hash: _Optional[str] = ...) -> None: ...

class RecoverSketchRequest(_message.Message):
    __slots__ = ("target", "expected_content_hash")
    TARGET_FIELD_NUMBER: _ClassVar[int]
    EXPECTED_CONTENT_HASH_FIELD_NUMBER: _ClassVar[int]
    target: SketchTarget
    expected_content_hash: str
    def __init__(self, target: _Optional[_Union[SketchTarget, _Mapping]] = ..., expected_content_hash: _Optional[str] = ...) -> None: ...

class ListDesignPagesRequest(_message.Message):
    __slots__ = ("scenario",)
    SCENARIO_FIELD_NUMBER: _ClassVar[int]
    scenario: str
    def __init__(self, scenario: _Optional[str] = ...) -> None: ...

class DesignScenario(_message.Message):
    __slots__ = ("scenario", "page_count", "issue")
    SCENARIO_FIELD_NUMBER: _ClassVar[int]
    PAGE_COUNT_FIELD_NUMBER: _ClassVar[int]
    ISSUE_FIELD_NUMBER: _ClassVar[int]
    scenario: str
    page_count: int
    issue: str
    def __init__(self, scenario: _Optional[str] = ..., page_count: _Optional[int] = ..., issue: _Optional[str] = ...) -> None: ...

class DesignPage(_message.Message):
    __slots__ = ("page", "title", "route", "status", "content_hash", "region_count", "issue", "registered", "routes")
    PAGE_FIELD_NUMBER: _ClassVar[int]
    TITLE_FIELD_NUMBER: _ClassVar[int]
    ROUTE_FIELD_NUMBER: _ClassVar[int]
    STATUS_FIELD_NUMBER: _ClassVar[int]
    CONTENT_HASH_FIELD_NUMBER: _ClassVar[int]
    REGION_COUNT_FIELD_NUMBER: _ClassVar[int]
    ISSUE_FIELD_NUMBER: _ClassVar[int]
    REGISTERED_FIELD_NUMBER: _ClassVar[int]
    ROUTES_FIELD_NUMBER: _ClassVar[int]
    page: str
    title: str
    route: str
    status: str
    content_hash: str
    region_count: int
    issue: str
    registered: bool
    routes: _containers.RepeatedScalarFieldContainer[str]
    def __init__(self, page: _Optional[str] = ..., title: _Optional[str] = ..., route: _Optional[str] = ..., status: _Optional[str] = ..., content_hash: _Optional[str] = ..., region_count: _Optional[int] = ..., issue: _Optional[str] = ..., registered: _Optional[bool] = ..., routes: _Optional[_Iterable[str]] = ...) -> None: ...

class ListDesignPagesResponse(_message.Message):
    __slots__ = ("scenarios", "pages", "issues")
    SCENARIOS_FIELD_NUMBER: _ClassVar[int]
    PAGES_FIELD_NUMBER: _ClassVar[int]
    ISSUES_FIELD_NUMBER: _ClassVar[int]
    scenarios: _containers.RepeatedCompositeFieldContainer[DesignScenario]
    pages: _containers.RepeatedCompositeFieldContainer[DesignPage]
    issues: _containers.RepeatedScalarFieldContainer[str]
    def __init__(self, scenarios: _Optional[_Iterable[_Union[DesignScenario, _Mapping]]] = ..., pages: _Optional[_Iterable[_Union[DesignPage, _Mapping]]] = ..., issues: _Optional[_Iterable[str]] = ...) -> None: ...

class RenderSketchRequest(_message.Message):
    __slots__ = ("target", "expected_content_hash", "missing_label", "failed_label", "kit", "theme", "direction")
    TARGET_FIELD_NUMBER: _ClassVar[int]
    EXPECTED_CONTENT_HASH_FIELD_NUMBER: _ClassVar[int]
    MISSING_LABEL_FIELD_NUMBER: _ClassVar[int]
    FAILED_LABEL_FIELD_NUMBER: _ClassVar[int]
    KIT_FIELD_NUMBER: _ClassVar[int]
    THEME_FIELD_NUMBER: _ClassVar[int]
    DIRECTION_FIELD_NUMBER: _ClassVar[int]
    target: SketchTarget
    expected_content_hash: str
    missing_label: str
    failed_label: str
    kit: str
    theme: str
    direction: str
    def __init__(self, target: _Optional[_Union[SketchTarget, _Mapping]] = ..., expected_content_hash: _Optional[str] = ..., missing_label: _Optional[str] = ..., failed_label: _Optional[str] = ..., kit: _Optional[str] = ..., theme: _Optional[str] = ..., direction: _Optional[str] = ...) -> None: ...

class DesignConstraints(_message.Message):
    __slots__ = ("design_source", "design_source_hash", "preserve_routes", "preserve_business_behavior")
    DESIGN_SOURCE_FIELD_NUMBER: _ClassVar[int]
    DESIGN_SOURCE_HASH_FIELD_NUMBER: _ClassVar[int]
    PRESERVE_ROUTES_FIELD_NUMBER: _ClassVar[int]
    PRESERVE_BUSINESS_BEHAVIOR_FIELD_NUMBER: _ClassVar[int]
    design_source: str
    design_source_hash: str
    preserve_routes: bool
    preserve_business_behavior: bool
    def __init__(self, design_source: _Optional[str] = ..., design_source_hash: _Optional[str] = ..., preserve_routes: _Optional[bool] = ..., preserve_business_behavior: _Optional[bool] = ...) -> None: ...

class DesignIntent(_message.Message):
    __slots__ = ("constraints", "intent", "users", "primary_tasks", "target", "kit", "required_capabilities", "viewports")
    CONSTRAINTS_FIELD_NUMBER: _ClassVar[int]
    INTENT_FIELD_NUMBER: _ClassVar[int]
    USERS_FIELD_NUMBER: _ClassVar[int]
    PRIMARY_TASKS_FIELD_NUMBER: _ClassVar[int]
    TARGET_FIELD_NUMBER: _ClassVar[int]
    KIT_FIELD_NUMBER: _ClassVar[int]
    REQUIRED_CAPABILITIES_FIELD_NUMBER: _ClassVar[int]
    VIEWPORTS_FIELD_NUMBER: _ClassVar[int]
    constraints: DesignConstraints
    intent: str
    users: _containers.RepeatedScalarFieldContainer[str]
    primary_tasks: _containers.RepeatedScalarFieldContainer[str]
    target: str
    kit: str
    required_capabilities: _containers.RepeatedScalarFieldContainer[str]
    viewports: _containers.RepeatedScalarFieldContainer[str]
    def __init__(self, constraints: _Optional[_Union[DesignConstraints, _Mapping]] = ..., intent: _Optional[str] = ..., users: _Optional[_Iterable[str]] = ..., primary_tasks: _Optional[_Iterable[str]] = ..., target: _Optional[str] = ..., kit: _Optional[str] = ..., required_capabilities: _Optional[_Iterable[str]] = ..., viewports: _Optional[_Iterable[str]] = ...) -> None: ...

class ProposeSketchRequest(_message.Message):
    __slots__ = ("target", "expected_content_hash", "intent", "candidate_limit")
    TARGET_FIELD_NUMBER: _ClassVar[int]
    EXPECTED_CONTENT_HASH_FIELD_NUMBER: _ClassVar[int]
    INTENT_FIELD_NUMBER: _ClassVar[int]
    CANDIDATE_LIMIT_FIELD_NUMBER: _ClassVar[int]
    target: SketchTarget
    expected_content_hash: str
    intent: DesignIntent
    candidate_limit: int
    def __init__(self, target: _Optional[_Union[SketchTarget, _Mapping]] = ..., expected_content_hash: _Optional[str] = ..., intent: _Optional[_Union[DesignIntent, _Mapping]] = ..., candidate_limit: _Optional[int] = ...) -> None: ...

class DesignProposal(_message.Message):
    __slots__ = ("title", "sketch", "obligations")
    TITLE_FIELD_NUMBER: _ClassVar[int]
    SKETCH_FIELD_NUMBER: _ClassVar[int]
    OBLIGATIONS_FIELD_NUMBER: _ClassVar[int]
    title: str
    sketch: Sketch
    obligations: _containers.RepeatedScalarFieldContainer[str]
    def __init__(self, title: _Optional[str] = ..., sketch: _Optional[_Union[Sketch, _Mapping]] = ..., obligations: _Optional[_Iterable[str]] = ...) -> None: ...

class DesignSourceSnapshot(_message.Message):
    __slots__ = ("path", "content_hash", "content")
    PATH_FIELD_NUMBER: _ClassVar[int]
    CONTENT_HASH_FIELD_NUMBER: _ClassVar[int]
    CONTENT_FIELD_NUMBER: _ClassVar[int]
    path: str
    content_hash: str
    content: str
    def __init__(self, path: _Optional[str] = ..., content_hash: _Optional[str] = ..., content: _Optional[str] = ...) -> None: ...

class ProposeSketchResponse(_message.Message):
    __slots__ = ("design_source", "content_hash", "candidates", "retrieval_mode", "diagnostics")
    DESIGN_SOURCE_FIELD_NUMBER: _ClassVar[int]
    CONTENT_HASH_FIELD_NUMBER: _ClassVar[int]
    CANDIDATES_FIELD_NUMBER: _ClassVar[int]
    RETRIEVAL_MODE_FIELD_NUMBER: _ClassVar[int]
    DIAGNOSTICS_FIELD_NUMBER: _ClassVar[int]
    design_source: DesignSourceSnapshot
    content_hash: str
    candidates: _containers.RepeatedCompositeFieldContainer[DesignProposal]
    retrieval_mode: str
    diagnostics: _containers.RepeatedScalarFieldContainer[str]
    def __init__(self, design_source: _Optional[_Union[DesignSourceSnapshot, _Mapping]] = ..., content_hash: _Optional[str] = ..., candidates: _Optional[_Iterable[_Union[DesignProposal, _Mapping]]] = ..., retrieval_mode: _Optional[str] = ..., diagnostics: _Optional[_Iterable[str]] = ...) -> None: ...

class CandidateReference(_message.Message):
    __slots__ = ("scenario", "design_id", "hash")
    SCENARIO_FIELD_NUMBER: _ClassVar[int]
    DESIGN_ID_FIELD_NUMBER: _ClassVar[int]
    HASH_FIELD_NUMBER: _ClassVar[int]
    scenario: str
    design_id: str
    hash: str
    def __init__(self, scenario: _Optional[str] = ..., design_id: _Optional[str] = ..., hash: _Optional[str] = ...) -> None: ...

class SaveCandidateRequest(_message.Message):
    __slots__ = ("target", "design_id", "expected_content_hash", "sketch")
    TARGET_FIELD_NUMBER: _ClassVar[int]
    DESIGN_ID_FIELD_NUMBER: _ClassVar[int]
    EXPECTED_CONTENT_HASH_FIELD_NUMBER: _ClassVar[int]
    SKETCH_FIELD_NUMBER: _ClassVar[int]
    target: SketchTarget
    design_id: str
    expected_content_hash: str
    sketch: Sketch
    def __init__(self, target: _Optional[_Union[SketchTarget, _Mapping]] = ..., design_id: _Optional[str] = ..., expected_content_hash: _Optional[str] = ..., sketch: _Optional[_Union[Sketch, _Mapping]] = ...) -> None: ...

class CandidateResponse(_message.Message):
    __slots__ = ("refinement", "parent_hash", "candidate", "page", "base_hash", "sketch", "declared_regions")
    REFINEMENT_FIELD_NUMBER: _ClassVar[int]
    PARENT_HASH_FIELD_NUMBER: _ClassVar[int]
    CANDIDATE_FIELD_NUMBER: _ClassVar[int]
    PAGE_FIELD_NUMBER: _ClassVar[int]
    BASE_HASH_FIELD_NUMBER: _ClassVar[int]
    SKETCH_FIELD_NUMBER: _ClassVar[int]
    DECLARED_REGIONS_FIELD_NUMBER: _ClassVar[int]
    refinement: CandidateRefinement
    parent_hash: str
    candidate: CandidateReference
    page: str
    base_hash: str
    sketch: Sketch
    declared_regions: _containers.RepeatedCompositeFieldContainer[SketchRegion]
    def __init__(self, refinement: _Optional[_Union[CandidateRefinement, _Mapping]] = ..., parent_hash: _Optional[str] = ..., candidate: _Optional[_Union[CandidateReference, _Mapping]] = ..., page: _Optional[str] = ..., base_hash: _Optional[str] = ..., sketch: _Optional[_Union[Sketch, _Mapping]] = ..., declared_regions: _Optional[_Iterable[_Union[SketchRegion, _Mapping]]] = ...) -> None: ...

class RenderCandidateRequest(_message.Message):
    __slots__ = ("candidate", "missing_label", "failed_label", "kit", "theme", "direction", "preview_state")
    CANDIDATE_FIELD_NUMBER: _ClassVar[int]
    MISSING_LABEL_FIELD_NUMBER: _ClassVar[int]
    FAILED_LABEL_FIELD_NUMBER: _ClassVar[int]
    KIT_FIELD_NUMBER: _ClassVar[int]
    THEME_FIELD_NUMBER: _ClassVar[int]
    DIRECTION_FIELD_NUMBER: _ClassVar[int]
    PREVIEW_STATE_FIELD_NUMBER: _ClassVar[int]
    candidate: CandidateReference
    missing_label: str
    failed_label: str
    kit: str
    theme: str
    direction: str
    preview_state: str
    def __init__(self, candidate: _Optional[_Union[CandidateReference, _Mapping]] = ..., missing_label: _Optional[str] = ..., failed_label: _Optional[str] = ..., kit: _Optional[str] = ..., theme: _Optional[str] = ..., direction: _Optional[str] = ..., preview_state: _Optional[str] = ...) -> None: ...

class CandidateRegionMapping(_message.Message):
    __slots__ = ("region", "template_region")
    REGION_FIELD_NUMBER: _ClassVar[int]
    TEMPLATE_REGION_FIELD_NUMBER: _ClassVar[int]
    region: str
    template_region: str
    def __init__(self, region: _Optional[str] = ..., template_region: _Optional[str] = ...) -> None: ...

class MapCandidateRegionsRequest(_message.Message):
    __slots__ = ("candidate", "mappings")
    CANDIDATE_FIELD_NUMBER: _ClassVar[int]
    MAPPINGS_FIELD_NUMBER: _ClassVar[int]
    candidate: CandidateReference
    mappings: _containers.RepeatedCompositeFieldContainer[CandidateRegionMapping]
    def __init__(self, candidate: _Optional[_Union[CandidateReference, _Mapping]] = ..., mappings: _Optional[_Iterable[_Union[CandidateRegionMapping, _Mapping]]] = ...) -> None: ...

class PlaceCandidateAssetRequest(_message.Message):
    __slots__ = ("candidate", "region", "asset", "story")
    CANDIDATE_FIELD_NUMBER: _ClassVar[int]
    REGION_FIELD_NUMBER: _ClassVar[int]
    ASSET_FIELD_NUMBER: _ClassVar[int]
    STORY_FIELD_NUMBER: _ClassVar[int]
    candidate: CandidateReference
    region: str
    asset: AssetReference
    story: str
    def __init__(self, candidate: _Optional[_Union[CandidateReference, _Mapping]] = ..., region: _Optional[str] = ..., asset: _Optional[_Union[AssetReference, _Mapping]] = ..., story: _Optional[str] = ...) -> None: ...

class CandidateSummary(_message.Message):
    __slots__ = ("candidate", "parent_hash", "base_hash", "template_asset", "template_version")
    CANDIDATE_FIELD_NUMBER: _ClassVar[int]
    PARENT_HASH_FIELD_NUMBER: _ClassVar[int]
    BASE_HASH_FIELD_NUMBER: _ClassVar[int]
    TEMPLATE_ASSET_FIELD_NUMBER: _ClassVar[int]
    TEMPLATE_VERSION_FIELD_NUMBER: _ClassVar[int]
    candidate: CandidateReference
    parent_hash: str
    base_hash: str
    template_asset: str
    template_version: str
    def __init__(self, candidate: _Optional[_Union[CandidateReference, _Mapping]] = ..., parent_hash: _Optional[str] = ..., base_hash: _Optional[str] = ..., template_asset: _Optional[str] = ..., template_version: _Optional[str] = ...) -> None: ...

class ListCandidatesResponse(_message.Message):
    __slots__ = ("candidates",)
    CANDIDATES_FIELD_NUMBER: _ClassVar[int]
    candidates: _containers.RepeatedCompositeFieldContainer[CandidateSummary]
    def __init__(self, candidates: _Optional[_Iterable[_Union[CandidateSummary, _Mapping]]] = ...) -> None: ...

class CaptureCandidateRequest(_message.Message):
    __slots__ = ("render", "expected_render_hash", "idempotency_key", "width", "height")
    RENDER_FIELD_NUMBER: _ClassVar[int]
    EXPECTED_RENDER_HASH_FIELD_NUMBER: _ClassVar[int]
    IDEMPOTENCY_KEY_FIELD_NUMBER: _ClassVar[int]
    WIDTH_FIELD_NUMBER: _ClassVar[int]
    HEIGHT_FIELD_NUMBER: _ClassVar[int]
    render: RenderCandidateRequest
    expected_render_hash: str
    idempotency_key: str
    width: int
    height: int
    def __init__(self, render: _Optional[_Union[RenderCandidateRequest, _Mapping]] = ..., expected_render_hash: _Optional[str] = ..., idempotency_key: _Optional[str] = ..., width: _Optional[int] = ..., height: _Optional[int] = ...) -> None: ...

class GetCaptureRequest(_message.Message):
    __slots__ = ("id",)
    ID_FIELD_NUMBER: _ClassVar[int]
    id: str
    def __init__(self, id: _Optional[str] = ...) -> None: ...

class CapturedRegionGeometry(_message.Message):
    __slots__ = ("region", "x", "y", "width", "height")
    REGION_FIELD_NUMBER: _ClassVar[int]
    X_FIELD_NUMBER: _ClassVar[int]
    Y_FIELD_NUMBER: _ClassVar[int]
    WIDTH_FIELD_NUMBER: _ClassVar[int]
    HEIGHT_FIELD_NUMBER: _ClassVar[int]
    region: str
    x: float
    y: float
    width: float
    height: float
    def __init__(self, region: _Optional[str] = ..., x: _Optional[float] = ..., y: _Optional[float] = ..., width: _Optional[float] = ..., height: _Optional[float] = ...) -> None: ...

class CapturedTargetEvidence(_message.Message):
    __slots__ = ("render_hash", "width", "height", "regions")
    RENDER_HASH_FIELD_NUMBER: _ClassVar[int]
    WIDTH_FIELD_NUMBER: _ClassVar[int]
    HEIGHT_FIELD_NUMBER: _ClassVar[int]
    REGIONS_FIELD_NUMBER: _ClassVar[int]
    render_hash: str
    width: float
    height: float
    regions: _containers.RepeatedCompositeFieldContainer[CapturedRegionGeometry]
    def __init__(self, render_hash: _Optional[str] = ..., width: _Optional[float] = ..., height: _Optional[float] = ..., regions: _Optional[_Iterable[_Union[CapturedRegionGeometry, _Mapping]]] = ...) -> None: ...

class CaptureArtifact(_message.Message):
    __slots__ = ("kind", "reference", "evidence")
    KIND_FIELD_NUMBER: _ClassVar[int]
    REFERENCE_FIELD_NUMBER: _ClassVar[int]
    EVIDENCE_FIELD_NUMBER: _ClassVar[int]
    kind: str
    reference: str
    evidence: CapturedTargetEvidence
    def __init__(self, kind: _Optional[str] = ..., reference: _Optional[str] = ..., evidence: _Optional[_Union[CapturedTargetEvidence, _Mapping]] = ...) -> None: ...

class CaptureOperation(_message.Message):
    __slots__ = ("id", "state", "producer_id", "candidate", "target", "width", "height", "artifacts", "detail", "version", "previous_id")
    ID_FIELD_NUMBER: _ClassVar[int]
    STATE_FIELD_NUMBER: _ClassVar[int]
    PRODUCER_ID_FIELD_NUMBER: _ClassVar[int]
    CANDIDATE_FIELD_NUMBER: _ClassVar[int]
    TARGET_FIELD_NUMBER: _ClassVar[int]
    WIDTH_FIELD_NUMBER: _ClassVar[int]
    HEIGHT_FIELD_NUMBER: _ClassVar[int]
    ARTIFACTS_FIELD_NUMBER: _ClassVar[int]
    DETAIL_FIELD_NUMBER: _ClassVar[int]
    VERSION_FIELD_NUMBER: _ClassVar[int]
    PREVIOUS_ID_FIELD_NUMBER: _ClassVar[int]
    id: str
    state: str
    producer_id: str
    candidate: CandidateReference
    target: _preview_pb2.CompositionRenderTarget
    width: int
    height: int
    artifacts: _containers.RepeatedCompositeFieldContainer[CaptureArtifact]
    detail: str
    version: int
    previous_id: str
    def __init__(self, id: _Optional[str] = ..., state: _Optional[str] = ..., producer_id: _Optional[str] = ..., candidate: _Optional[_Union[CandidateReference, _Mapping]] = ..., target: _Optional[_Union[_preview_pb2.CompositionRenderTarget, _Mapping]] = ..., width: _Optional[int] = ..., height: _Optional[int] = ..., artifacts: _Optional[_Iterable[_Union[CaptureArtifact, _Mapping]]] = ..., detail: _Optional[str] = ..., version: _Optional[int] = ..., previous_id: _Optional[str] = ...) -> None: ...

class GetCaptureScreenshotRequest(_message.Message):
    __slots__ = ("id", "reference")
    ID_FIELD_NUMBER: _ClassVar[int]
    REFERENCE_FIELD_NUMBER: _ClassVar[int]
    id: str
    reference: str
    def __init__(self, id: _Optional[str] = ..., reference: _Optional[str] = ...) -> None: ...

class CaptureScreenshot(_message.Message):
    __slots__ = ("reference", "url", "width", "height", "content_type")
    REFERENCE_FIELD_NUMBER: _ClassVar[int]
    URL_FIELD_NUMBER: _ClassVar[int]
    WIDTH_FIELD_NUMBER: _ClassVar[int]
    HEIGHT_FIELD_NUMBER: _ClassVar[int]
    CONTENT_TYPE_FIELD_NUMBER: _ClassVar[int]
    reference: str
    url: str
    width: int
    height: int
    content_type: str
    def __init__(self, reference: _Optional[str] = ..., url: _Optional[str] = ..., width: _Optional[int] = ..., height: _Optional[int] = ..., content_type: _Optional[str] = ...) -> None: ...

class RetryCaptureRequest(_message.Message):
    __slots__ = ("id", "idempotency_key")
    ID_FIELD_NUMBER: _ClassVar[int]
    IDEMPOTENCY_KEY_FIELD_NUMBER: _ClassVar[int]
    id: str
    idempotency_key: str
    def __init__(self, id: _Optional[str] = ..., idempotency_key: _Optional[str] = ...) -> None: ...

class GetCritiqueRubricRequest(_message.Message):
    __slots__ = ()
    def __init__(self) -> None: ...

class CritiqueRubric(_message.Message):
    __slots__ = ("rubric_version", "policy_version", "dimensions", "anchors", "calibrated")
    RUBRIC_VERSION_FIELD_NUMBER: _ClassVar[int]
    POLICY_VERSION_FIELD_NUMBER: _ClassVar[int]
    DIMENSIONS_FIELD_NUMBER: _ClassVar[int]
    ANCHORS_FIELD_NUMBER: _ClassVar[int]
    CALIBRATED_FIELD_NUMBER: _ClassVar[int]
    rubric_version: str
    policy_version: str
    dimensions: _containers.RepeatedScalarFieldContainer[str]
    anchors: _containers.RepeatedScalarFieldContainer[str]
    calibrated: bool
    def __init__(self, rubric_version: _Optional[str] = ..., policy_version: _Optional[str] = ..., dimensions: _Optional[_Iterable[str]] = ..., anchors: _Optional[_Iterable[str]] = ..., calibrated: _Optional[bool] = ...) -> None: ...

class CritiqueTarget(_message.Message):
    __slots__ = ("scenario", "design_id", "revision", "render_hash")
    SCENARIO_FIELD_NUMBER: _ClassVar[int]
    DESIGN_ID_FIELD_NUMBER: _ClassVar[int]
    REVISION_FIELD_NUMBER: _ClassVar[int]
    RENDER_HASH_FIELD_NUMBER: _ClassVar[int]
    scenario: str
    design_id: str
    revision: str
    render_hash: str
    def __init__(self, scenario: _Optional[str] = ..., design_id: _Optional[str] = ..., revision: _Optional[str] = ..., render_hash: _Optional[str] = ...) -> None: ...

class CriticIdentity(_message.Message):
    __slots__ = ("kind", "id", "version", "model", "profile")
    KIND_FIELD_NUMBER: _ClassVar[int]
    ID_FIELD_NUMBER: _ClassVar[int]
    VERSION_FIELD_NUMBER: _ClassVar[int]
    MODEL_FIELD_NUMBER: _ClassVar[int]
    PROFILE_FIELD_NUMBER: _ClassVar[int]
    kind: str
    id: str
    version: str
    model: str
    profile: str
    def __init__(self, kind: _Optional[str] = ..., id: _Optional[str] = ..., version: _Optional[str] = ..., model: _Optional[str] = ..., profile: _Optional[str] = ...) -> None: ...

class CritiqueEvidence(_message.Message):
    __slots__ = ("capture_id", "artifact", "region", "state", "width", "height", "render_hash")
    CAPTURE_ID_FIELD_NUMBER: _ClassVar[int]
    ARTIFACT_FIELD_NUMBER: _ClassVar[int]
    REGION_FIELD_NUMBER: _ClassVar[int]
    STATE_FIELD_NUMBER: _ClassVar[int]
    WIDTH_FIELD_NUMBER: _ClassVar[int]
    HEIGHT_FIELD_NUMBER: _ClassVar[int]
    RENDER_HASH_FIELD_NUMBER: _ClassVar[int]
    capture_id: str
    artifact: str
    region: str
    state: str
    width: int
    height: int
    render_hash: str
    def __init__(self, capture_id: _Optional[str] = ..., artifact: _Optional[str] = ..., region: _Optional[str] = ..., state: _Optional[str] = ..., width: _Optional[int] = ..., height: _Optional[int] = ..., render_hash: _Optional[str] = ...) -> None: ...

class CritiqueRating(_message.Message):
    __slots__ = ("dimension", "score", "rationale", "evidence")
    DIMENSION_FIELD_NUMBER: _ClassVar[int]
    SCORE_FIELD_NUMBER: _ClassVar[int]
    RATIONALE_FIELD_NUMBER: _ClassVar[int]
    EVIDENCE_FIELD_NUMBER: _ClassVar[int]
    dimension: str
    score: int
    rationale: str
    evidence: _containers.RepeatedCompositeFieldContainer[CritiqueEvidence]
    def __init__(self, dimension: _Optional[str] = ..., score: _Optional[int] = ..., rationale: _Optional[str] = ..., evidence: _Optional[_Iterable[_Union[CritiqueEvidence, _Mapping]]] = ...) -> None: ...

class CritiqueFinding(_message.Message):
    __slots__ = ("dimension", "severity", "rationale", "correction", "evidence")
    DIMENSION_FIELD_NUMBER: _ClassVar[int]
    SEVERITY_FIELD_NUMBER: _ClassVar[int]
    RATIONALE_FIELD_NUMBER: _ClassVar[int]
    CORRECTION_FIELD_NUMBER: _ClassVar[int]
    EVIDENCE_FIELD_NUMBER: _ClassVar[int]
    dimension: str
    severity: str
    rationale: str
    correction: str
    evidence: CritiqueEvidence
    def __init__(self, dimension: _Optional[str] = ..., severity: _Optional[str] = ..., rationale: _Optional[str] = ..., correction: _Optional[str] = ..., evidence: _Optional[_Union[CritiqueEvidence, _Mapping]] = ...) -> None: ...

class VisualCritique(_message.Message):
    __slots__ = ("target", "rubric_version", "policy_version", "critic", "ratings", "findings")
    TARGET_FIELD_NUMBER: _ClassVar[int]
    RUBRIC_VERSION_FIELD_NUMBER: _ClassVar[int]
    POLICY_VERSION_FIELD_NUMBER: _ClassVar[int]
    CRITIC_FIELD_NUMBER: _ClassVar[int]
    RATINGS_FIELD_NUMBER: _ClassVar[int]
    FINDINGS_FIELD_NUMBER: _ClassVar[int]
    target: CritiqueTarget
    rubric_version: str
    policy_version: str
    critic: CriticIdentity
    ratings: _containers.RepeatedCompositeFieldContainer[CritiqueRating]
    findings: _containers.RepeatedCompositeFieldContainer[CritiqueFinding]
    def __init__(self, target: _Optional[_Union[CritiqueTarget, _Mapping]] = ..., rubric_version: _Optional[str] = ..., policy_version: _Optional[str] = ..., critic: _Optional[_Union[CriticIdentity, _Mapping]] = ..., ratings: _Optional[_Iterable[_Union[CritiqueRating, _Mapping]]] = ..., findings: _Optional[_Iterable[_Union[CritiqueFinding, _Mapping]]] = ...) -> None: ...

class CritiqueAssessment(_message.Message):
    __slots__ = ("visual_floor_met", "minimum_score", "blocking_dimensions", "blocking_findings", "acceptance_established")
    VISUAL_FLOOR_MET_FIELD_NUMBER: _ClassVar[int]
    MINIMUM_SCORE_FIELD_NUMBER: _ClassVar[int]
    BLOCKING_DIMENSIONS_FIELD_NUMBER: _ClassVar[int]
    BLOCKING_FINDINGS_FIELD_NUMBER: _ClassVar[int]
    ACCEPTANCE_ESTABLISHED_FIELD_NUMBER: _ClassVar[int]
    visual_floor_met: bool
    minimum_score: int
    blocking_dimensions: _containers.RepeatedScalarFieldContainer[str]
    blocking_findings: _containers.RepeatedScalarFieldContainer[int]
    acceptance_established: bool
    def __init__(self, visual_floor_met: _Optional[bool] = ..., minimum_score: _Optional[int] = ..., blocking_dimensions: _Optional[_Iterable[str]] = ..., blocking_findings: _Optional[_Iterable[int]] = ..., acceptance_established: _Optional[bool] = ...) -> None: ...

class RecordCritiqueRequest(_message.Message):
    __slots__ = ("idempotency_key", "review")
    IDEMPOTENCY_KEY_FIELD_NUMBER: _ClassVar[int]
    REVIEW_FIELD_NUMBER: _ClassVar[int]
    idempotency_key: str
    review: VisualCritique
    def __init__(self, idempotency_key: _Optional[str] = ..., review: _Optional[_Union[VisualCritique, _Mapping]] = ...) -> None: ...

class GetCritiqueRequest(_message.Message):
    __slots__ = ("id",)
    ID_FIELD_NUMBER: _ClassVar[int]
    id: str
    def __init__(self, id: _Optional[str] = ...) -> None: ...

class CritiqueRecord(_message.Message):
    __slots__ = ("id", "hash", "review", "assessment", "recorded_at")
    ID_FIELD_NUMBER: _ClassVar[int]
    HASH_FIELD_NUMBER: _ClassVar[int]
    REVIEW_FIELD_NUMBER: _ClassVar[int]
    ASSESSMENT_FIELD_NUMBER: _ClassVar[int]
    RECORDED_AT_FIELD_NUMBER: _ClassVar[int]
    id: str
    hash: str
    review: VisualCritique
    assessment: CritiqueAssessment
    recorded_at: str
    def __init__(self, id: _Optional[str] = ..., hash: _Optional[str] = ..., review: _Optional[_Union[VisualCritique, _Mapping]] = ..., assessment: _Optional[_Union[CritiqueAssessment, _Mapping]] = ..., recorded_at: _Optional[str] = ...) -> None: ...

class ListCritiquesRequest(_message.Message):
    __slots__ = ("target", "before_id")
    TARGET_FIELD_NUMBER: _ClassVar[int]
    BEFORE_ID_FIELD_NUMBER: _ClassVar[int]
    target: CritiqueTarget
    before_id: str
    def __init__(self, target: _Optional[_Union[CritiqueTarget, _Mapping]] = ..., before_id: _Optional[str] = ...) -> None: ...

class CritiqueSummary(_message.Message):
    __slots__ = ("id", "hash", "critic", "assessment", "recorded_at")
    ID_FIELD_NUMBER: _ClassVar[int]
    HASH_FIELD_NUMBER: _ClassVar[int]
    CRITIC_FIELD_NUMBER: _ClassVar[int]
    ASSESSMENT_FIELD_NUMBER: _ClassVar[int]
    RECORDED_AT_FIELD_NUMBER: _ClassVar[int]
    id: str
    hash: str
    critic: CriticIdentity
    assessment: CritiqueAssessment
    recorded_at: str
    def __init__(self, id: _Optional[str] = ..., hash: _Optional[str] = ..., critic: _Optional[_Union[CriticIdentity, _Mapping]] = ..., assessment: _Optional[_Union[CritiqueAssessment, _Mapping]] = ..., recorded_at: _Optional[str] = ...) -> None: ...

class ListCritiquesResponse(_message.Message):
    __slots__ = ("reviews", "next_before_id")
    REVIEWS_FIELD_NUMBER: _ClassVar[int]
    NEXT_BEFORE_ID_FIELD_NUMBER: _ClassVar[int]
    reviews: _containers.RepeatedCompositeFieldContainer[CritiqueSummary]
    next_before_id: str
    def __init__(self, reviews: _Optional[_Iterable[_Union[CritiqueSummary, _Mapping]]] = ..., next_before_id: _Optional[str] = ...) -> None: ...

class CandidateRefinement(_message.Message):
    __slots__ = ("round", "budget", "reason", "requested_regions", "changed_regions", "broader_change_reason")
    ROUND_FIELD_NUMBER: _ClassVar[int]
    BUDGET_FIELD_NUMBER: _ClassVar[int]
    REASON_FIELD_NUMBER: _ClassVar[int]
    REQUESTED_REGIONS_FIELD_NUMBER: _ClassVar[int]
    CHANGED_REGIONS_FIELD_NUMBER: _ClassVar[int]
    BROADER_CHANGE_REASON_FIELD_NUMBER: _ClassVar[int]
    round: int
    budget: int
    reason: str
    requested_regions: _containers.RepeatedScalarFieldContainer[str]
    changed_regions: _containers.RepeatedScalarFieldContainer[str]
    broader_change_reason: str
    def __init__(self, round: _Optional[int] = ..., budget: _Optional[int] = ..., reason: _Optional[str] = ..., requested_regions: _Optional[_Iterable[str]] = ..., changed_regions: _Optional[_Iterable[str]] = ..., broader_change_reason: _Optional[str] = ...) -> None: ...

class RefineCandidateRequest(_message.Message):
    __slots__ = ("candidate", "sketch", "requested_regions", "reason", "broader_change_reason", "additional_rounds")
    CANDIDATE_FIELD_NUMBER: _ClassVar[int]
    SKETCH_FIELD_NUMBER: _ClassVar[int]
    REQUESTED_REGIONS_FIELD_NUMBER: _ClassVar[int]
    REASON_FIELD_NUMBER: _ClassVar[int]
    BROADER_CHANGE_REASON_FIELD_NUMBER: _ClassVar[int]
    ADDITIONAL_ROUNDS_FIELD_NUMBER: _ClassVar[int]
    candidate: CandidateReference
    sketch: Sketch
    requested_regions: _containers.RepeatedScalarFieldContainer[str]
    reason: str
    broader_change_reason: str
    additional_rounds: int
    def __init__(self, candidate: _Optional[_Union[CandidateReference, _Mapping]] = ..., sketch: _Optional[_Union[Sketch, _Mapping]] = ..., requested_regions: _Optional[_Iterable[str]] = ..., reason: _Optional[str] = ..., broader_change_reason: _Optional[str] = ..., additional_rounds: _Optional[int] = ...) -> None: ...

class RefineCandidateResponse(_message.Message):
    __slots__ = ("result", "status", "round", "budget")
    RESULT_FIELD_NUMBER: _ClassVar[int]
    STATUS_FIELD_NUMBER: _ClassVar[int]
    ROUND_FIELD_NUMBER: _ClassVar[int]
    BUDGET_FIELD_NUMBER: _ClassVar[int]
    result: CandidateResponse
    status: str
    round: int
    budget: int
    def __init__(self, result: _Optional[_Union[CandidateResponse, _Mapping]] = ..., status: _Optional[str] = ..., round: _Optional[int] = ..., budget: _Optional[int] = ...) -> None: ...

class InferSketchRequest(_message.Message):
    __slots__ = ("proposal", "idempotency_key")
    PROPOSAL_FIELD_NUMBER: _ClassVar[int]
    IDEMPOTENCY_KEY_FIELD_NUMBER: _ClassVar[int]
    proposal: ProposeSketchRequest
    idempotency_key: str
    def __init__(self, proposal: _Optional[_Union[ProposeSketchRequest, _Mapping]] = ..., idempotency_key: _Optional[str] = ...) -> None: ...

class GetSketchInferenceRequest(_message.Message):
    __slots__ = ("id", "idempotency_key")
    ID_FIELD_NUMBER: _ClassVar[int]
    IDEMPOTENCY_KEY_FIELD_NUMBER: _ClassVar[int]
    id: str
    idempotency_key: str
    def __init__(self, id: _Optional[str] = ..., idempotency_key: _Optional[str] = ...) -> None: ...

class SketchInferenceOperation(_message.Message):
    __slots__ = ("id", "state", "request_hash", "detail", "proposal", "provider", "model", "input_tokens", "output_tokens", "cost_micros")
    ID_FIELD_NUMBER: _ClassVar[int]
    STATE_FIELD_NUMBER: _ClassVar[int]
    REQUEST_HASH_FIELD_NUMBER: _ClassVar[int]
    DETAIL_FIELD_NUMBER: _ClassVar[int]
    PROPOSAL_FIELD_NUMBER: _ClassVar[int]
    PROVIDER_FIELD_NUMBER: _ClassVar[int]
    MODEL_FIELD_NUMBER: _ClassVar[int]
    INPUT_TOKENS_FIELD_NUMBER: _ClassVar[int]
    OUTPUT_TOKENS_FIELD_NUMBER: _ClassVar[int]
    COST_MICROS_FIELD_NUMBER: _ClassVar[int]
    id: str
    state: str
    request_hash: str
    detail: str
    proposal: ProposeSketchResponse
    provider: str
    model: str
    input_tokens: int
    output_tokens: int
    cost_micros: int
    def __init__(self, id: _Optional[str] = ..., state: _Optional[str] = ..., request_hash: _Optional[str] = ..., detail: _Optional[str] = ..., proposal: _Optional[_Union[ProposeSketchResponse, _Mapping]] = ..., provider: _Optional[str] = ..., model: _Optional[str] = ..., input_tokens: _Optional[int] = ..., output_tokens: _Optional[int] = ..., cost_micros: _Optional[int] = ...) -> None: ...

class CheckCandidateAcceptanceRequest(_message.Message):
    __slots__ = ("render", "expected_render_hash", "critique_ids")
    RENDER_FIELD_NUMBER: _ClassVar[int]
    EXPECTED_RENDER_HASH_FIELD_NUMBER: _ClassVar[int]
    CRITIQUE_IDS_FIELD_NUMBER: _ClassVar[int]
    render: RenderCandidateRequest
    expected_render_hash: str
    critique_ids: _containers.RepeatedScalarFieldContainer[str]
    def __init__(self, render: _Optional[_Union[RenderCandidateRequest, _Mapping]] = ..., expected_render_hash: _Optional[str] = ..., critique_ids: _Optional[_Iterable[str]] = ...) -> None: ...

class AcceptanceRequirement(_message.Message):
    __slots__ = ("code", "status", "detail", "evidence_ids")
    CODE_FIELD_NUMBER: _ClassVar[int]
    STATUS_FIELD_NUMBER: _ClassVar[int]
    DETAIL_FIELD_NUMBER: _ClassVar[int]
    EVIDENCE_IDS_FIELD_NUMBER: _ClassVar[int]
    code: str
    status: str
    detail: str
    evidence_ids: _containers.RepeatedScalarFieldContainer[str]
    def __init__(self, code: _Optional[str] = ..., status: _Optional[str] = ..., detail: _Optional[str] = ..., evidence_ids: _Optional[_Iterable[str]] = ...) -> None: ...

class CandidateAcceptanceCheck(_message.Message):
    __slots__ = ("input_hashes", "candidate", "render_hash", "policy_version", "requirements", "ready", "acceptance_established")
    class InputHashesEntry(_message.Message):
        __slots__ = ("key", "value")
        KEY_FIELD_NUMBER: _ClassVar[int]
        VALUE_FIELD_NUMBER: _ClassVar[int]
        key: str
        value: str
        def __init__(self, key: _Optional[str] = ..., value: _Optional[str] = ...) -> None: ...
    INPUT_HASHES_FIELD_NUMBER: _ClassVar[int]
    CANDIDATE_FIELD_NUMBER: _ClassVar[int]
    RENDER_HASH_FIELD_NUMBER: _ClassVar[int]
    POLICY_VERSION_FIELD_NUMBER: _ClassVar[int]
    REQUIREMENTS_FIELD_NUMBER: _ClassVar[int]
    READY_FIELD_NUMBER: _ClassVar[int]
    ACCEPTANCE_ESTABLISHED_FIELD_NUMBER: _ClassVar[int]
    input_hashes: _containers.ScalarMap[str, str]
    candidate: CandidateReference
    render_hash: str
    policy_version: str
    requirements: _containers.RepeatedCompositeFieldContainer[AcceptanceRequirement]
    ready: bool
    acceptance_established: bool
    def __init__(self, input_hashes: _Optional[_Mapping[str, str]] = ..., candidate: _Optional[_Union[CandidateReference, _Mapping]] = ..., render_hash: _Optional[str] = ..., policy_version: _Optional[str] = ..., requirements: _Optional[_Iterable[_Union[AcceptanceRequirement, _Mapping]]] = ..., ready: _Optional[bool] = ..., acceptance_established: _Optional[bool] = ...) -> None: ...

class AcceptCandidateRequest(_message.Message):
    __slots__ = ("check", "actor", "idempotency_key")
    CHECK_FIELD_NUMBER: _ClassVar[int]
    ACTOR_FIELD_NUMBER: _ClassVar[int]
    IDEMPOTENCY_KEY_FIELD_NUMBER: _ClassVar[int]
    check: CheckCandidateAcceptanceRequest
    actor: str
    idempotency_key: str
    def __init__(self, check: _Optional[_Union[CheckCandidateAcceptanceRequest, _Mapping]] = ..., actor: _Optional[str] = ..., idempotency_key: _Optional[str] = ...) -> None: ...

class GetAcceptanceRequest(_message.Message):
    __slots__ = ("scenario", "design_id", "id")
    SCENARIO_FIELD_NUMBER: _ClassVar[int]
    DESIGN_ID_FIELD_NUMBER: _ClassVar[int]
    ID_FIELD_NUMBER: _ClassVar[int]
    scenario: str
    design_id: str
    id: str
    def __init__(self, scenario: _Optional[str] = ..., design_id: _Optional[str] = ..., id: _Optional[str] = ...) -> None: ...

class AcceptanceDecisionIntent(_message.Message):
    __slots__ = ("design_id", "candidate_hash", "expected_render_hash", "actor", "kit", "theme", "direction", "preview_state", "missing_label", "failed_label", "critique_ids")
    DESIGN_ID_FIELD_NUMBER: _ClassVar[int]
    CANDIDATE_HASH_FIELD_NUMBER: _ClassVar[int]
    EXPECTED_RENDER_HASH_FIELD_NUMBER: _ClassVar[int]
    ACTOR_FIELD_NUMBER: _ClassVar[int]
    KIT_FIELD_NUMBER: _ClassVar[int]
    THEME_FIELD_NUMBER: _ClassVar[int]
    DIRECTION_FIELD_NUMBER: _ClassVar[int]
    PREVIEW_STATE_FIELD_NUMBER: _ClassVar[int]
    MISSING_LABEL_FIELD_NUMBER: _ClassVar[int]
    FAILED_LABEL_FIELD_NUMBER: _ClassVar[int]
    CRITIQUE_IDS_FIELD_NUMBER: _ClassVar[int]
    design_id: str
    candidate_hash: str
    expected_render_hash: str
    actor: str
    kit: str
    theme: str
    direction: str
    preview_state: str
    missing_label: str
    failed_label: str
    critique_ids: _containers.RepeatedScalarFieldContainer[str]
    def __init__(self, design_id: _Optional[str] = ..., candidate_hash: _Optional[str] = ..., expected_render_hash: _Optional[str] = ..., actor: _Optional[str] = ..., kit: _Optional[str] = ..., theme: _Optional[str] = ..., direction: _Optional[str] = ..., preview_state: _Optional[str] = ..., missing_label: _Optional[str] = ..., failed_label: _Optional[str] = ..., critique_ids: _Optional[_Iterable[str]] = ...) -> None: ...

class AcceptanceFacts(_message.Message):
    __slots__ = ("policy_version", "candidate_hash", "render_hash", "input_hashes", "requirements")
    class InputHashesEntry(_message.Message):
        __slots__ = ("key", "value")
        KEY_FIELD_NUMBER: _ClassVar[int]
        VALUE_FIELD_NUMBER: _ClassVar[int]
        key: str
        value: str
        def __init__(self, key: _Optional[str] = ..., value: _Optional[str] = ...) -> None: ...
    POLICY_VERSION_FIELD_NUMBER: _ClassVar[int]
    CANDIDATE_HASH_FIELD_NUMBER: _ClassVar[int]
    RENDER_HASH_FIELD_NUMBER: _ClassVar[int]
    INPUT_HASHES_FIELD_NUMBER: _ClassVar[int]
    REQUIREMENTS_FIELD_NUMBER: _ClassVar[int]
    policy_version: str
    candidate_hash: str
    render_hash: str
    input_hashes: _containers.ScalarMap[str, str]
    requirements: _containers.RepeatedCompositeFieldContainer[AcceptanceRequirement]
    def __init__(self, policy_version: _Optional[str] = ..., candidate_hash: _Optional[str] = ..., render_hash: _Optional[str] = ..., input_hashes: _Optional[_Mapping[str, str]] = ..., requirements: _Optional[_Iterable[_Union[AcceptanceRequirement, _Mapping]]] = ...) -> None: ...

class AcceptanceDecision(_message.Message):
    __slots__ = ("schema_version", "id", "hash", "state", "intent", "base_hash", "facts", "detail", "created_at", "completed_at")
    SCHEMA_VERSION_FIELD_NUMBER: _ClassVar[int]
    ID_FIELD_NUMBER: _ClassVar[int]
    HASH_FIELD_NUMBER: _ClassVar[int]
    STATE_FIELD_NUMBER: _ClassVar[int]
    INTENT_FIELD_NUMBER: _ClassVar[int]
    BASE_HASH_FIELD_NUMBER: _ClassVar[int]
    FACTS_FIELD_NUMBER: _ClassVar[int]
    DETAIL_FIELD_NUMBER: _ClassVar[int]
    CREATED_AT_FIELD_NUMBER: _ClassVar[int]
    COMPLETED_AT_FIELD_NUMBER: _ClassVar[int]
    schema_version: int
    id: str
    hash: str
    state: str
    intent: AcceptanceDecisionIntent
    base_hash: str
    facts: AcceptanceFacts
    detail: str
    created_at: str
    completed_at: str
    def __init__(self, schema_version: _Optional[int] = ..., id: _Optional[str] = ..., hash: _Optional[str] = ..., state: _Optional[str] = ..., intent: _Optional[_Union[AcceptanceDecisionIntent, _Mapping]] = ..., base_hash: _Optional[str] = ..., facts: _Optional[_Union[AcceptanceFacts, _Mapping]] = ..., detail: _Optional[str] = ..., created_at: _Optional[str] = ..., completed_at: _Optional[str] = ...) -> None: ...
