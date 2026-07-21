import datetime

from google.protobuf import timestamp_pb2 as _timestamp_pb2
from google.protobuf.internal import containers as _containers
from google.protobuf.internal import enum_type_wrapper as _enum_type_wrapper
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Iterable as _Iterable, Mapping as _Mapping
from typing import ClassVar as _ClassVar, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class LibraryVersionStatus(int, metaclass=_enum_type_wrapper.EnumTypeWrapper):
    __slots__ = ()
    LIBRARY_VERSION_STATUS_UNSPECIFIED: _ClassVar[LibraryVersionStatus]
    LIBRARY_VERSION_STATUS_CURRENT: _ClassVar[LibraryVersionStatus]
    LIBRARY_VERSION_STATUS_BEHIND: _ClassVar[LibraryVersionStatus]
    LIBRARY_VERSION_STATUS_DEPRECATED: _ClassVar[LibraryVersionStatus]
    LIBRARY_VERSION_STATUS_MISSING: _ClassVar[LibraryVersionStatus]
    LIBRARY_VERSION_STATUS_UNKNOWN: _ClassVar[LibraryVersionStatus]

class LocalStatus(int, metaclass=_enum_type_wrapper.EnumTypeWrapper):
    __slots__ = ()
    LOCAL_STATUS_UNSPECIFIED: _ClassVar[LocalStatus]
    LOCAL_STATUS_CLEAN: _ClassVar[LocalStatus]
    LOCAL_STATUS_MODIFIED: _ClassVar[LocalStatus]
    LOCAL_STATUS_MISSING: _ClassVar[LocalStatus]
    LOCAL_STATUS_UNKNOWN: _ClassVar[LocalStatus]

class ResolveSource(int, metaclass=_enum_type_wrapper.EnumTypeWrapper):
    __slots__ = ()
    RESOLVE_SOURCE_UNSPECIFIED: _ClassVar[ResolveSource]
    RESOLVE_SOURCE_EXPLICIT: _ClassVar[ResolveSource]
    RESOLVE_SOURCE_TEMPLATE_MANIFEST: _ClassVar[ResolveSource]
    RESOLVE_SOURCE_HEURISTIC: _ClassVar[ResolveSource]
    RESOLVE_SOURCE_FALLBACK: _ClassVar[ResolveSource]

class RecommendationClass(int, metaclass=_enum_type_wrapper.EnumTypeWrapper):
    __slots__ = ()
    RECOMMENDATION_CLASS_UNSPECIFIED: _ClassVar[RecommendationClass]
    RECOMMENDATION_CLASS_HEURISTIC: _ClassVar[RecommendationClass]
    RECOMMENDATION_CLASS_UNAVAILABLE: _ClassVar[RecommendationClass]

class ReconvergeAction(int, metaclass=_enum_type_wrapper.EnumTypeWrapper):
    __slots__ = ()
    RECONVERGE_ACTION_UNSPECIFIED: _ClassVar[ReconvergeAction]
    RECONVERGE_ACTION_REAPPLIED: _ClassVar[ReconvergeAction]
    RECONVERGE_ACTION_WOULD_REAPPLY: _ClassVar[ReconvergeAction]
    RECONVERGE_ACTION_FLAGGED_MODIFIED: _ClassVar[ReconvergeAction]
    RECONVERGE_ACTION_SKIPPED_UNRESOLVED: _ClassVar[ReconvergeAction]
    RECONVERGE_ACTION_ERROR: _ClassVar[ReconvergeAction]
LIBRARY_VERSION_STATUS_UNSPECIFIED: LibraryVersionStatus
LIBRARY_VERSION_STATUS_CURRENT: LibraryVersionStatus
LIBRARY_VERSION_STATUS_BEHIND: LibraryVersionStatus
LIBRARY_VERSION_STATUS_DEPRECATED: LibraryVersionStatus
LIBRARY_VERSION_STATUS_MISSING: LibraryVersionStatus
LIBRARY_VERSION_STATUS_UNKNOWN: LibraryVersionStatus
LOCAL_STATUS_UNSPECIFIED: LocalStatus
LOCAL_STATUS_CLEAN: LocalStatus
LOCAL_STATUS_MODIFIED: LocalStatus
LOCAL_STATUS_MISSING: LocalStatus
LOCAL_STATUS_UNKNOWN: LocalStatus
RESOLVE_SOURCE_UNSPECIFIED: ResolveSource
RESOLVE_SOURCE_EXPLICIT: ResolveSource
RESOLVE_SOURCE_TEMPLATE_MANIFEST: ResolveSource
RESOLVE_SOURCE_HEURISTIC: ResolveSource
RESOLVE_SOURCE_FALLBACK: ResolveSource
RECOMMENDATION_CLASS_UNSPECIFIED: RecommendationClass
RECOMMENDATION_CLASS_HEURISTIC: RecommendationClass
RECOMMENDATION_CLASS_UNAVAILABLE: RecommendationClass
RECONVERGE_ACTION_UNSPECIFIED: ReconvergeAction
RECONVERGE_ACTION_REAPPLIED: ReconvergeAction
RECONVERGE_ACTION_WOULD_REAPPLY: ReconvergeAction
RECONVERGE_ACTION_FLAGGED_MODIFIED: ReconvergeAction
RECONVERGE_ACTION_SKIPPED_UNRESOLVED: ReconvergeAction
RECONVERGE_ACTION_ERROR: ReconvergeAction

class ListScenariosRequest(_message.Message):
    __slots__ = ()
    def __init__(self) -> None: ...

class ListScenariosResponse(_message.Message):
    __slots__ = ("scenarios",)
    SCENARIOS_FIELD_NUMBER: _ClassVar[int]
    scenarios: _containers.RepeatedCompositeFieldContainer[ScenarioOption]
    def __init__(self, scenarios: _Optional[_Iterable[_Union[ScenarioOption, _Mapping]]] = ...) -> None: ...

class ScenarioOption(_message.Message):
    __slots__ = ("name", "display_name")
    NAME_FIELD_NUMBER: _ClassVar[int]
    DISPLAY_NAME_FIELD_NUMBER: _ClassVar[int]
    name: str
    display_name: str
    def __init__(self, name: _Optional[str] = ..., display_name: _Optional[str] = ...) -> None: ...

class Adoption(_message.Message):
    __slots__ = ("id", "component_id", "library_id", "scenario", "adopted_path", "adopted_version", "library_version_status", "local_status", "status_detail", "created_at", "refreshed_at", "source_sha256", "applied_at", "files")
    ID_FIELD_NUMBER: _ClassVar[int]
    COMPONENT_ID_FIELD_NUMBER: _ClassVar[int]
    LIBRARY_ID_FIELD_NUMBER: _ClassVar[int]
    SCENARIO_FIELD_NUMBER: _ClassVar[int]
    ADOPTED_PATH_FIELD_NUMBER: _ClassVar[int]
    ADOPTED_VERSION_FIELD_NUMBER: _ClassVar[int]
    LIBRARY_VERSION_STATUS_FIELD_NUMBER: _ClassVar[int]
    LOCAL_STATUS_FIELD_NUMBER: _ClassVar[int]
    STATUS_DETAIL_FIELD_NUMBER: _ClassVar[int]
    CREATED_AT_FIELD_NUMBER: _ClassVar[int]
    REFRESHED_AT_FIELD_NUMBER: _ClassVar[int]
    SOURCE_SHA256_FIELD_NUMBER: _ClassVar[int]
    APPLIED_AT_FIELD_NUMBER: _ClassVar[int]
    FILES_FIELD_NUMBER: _ClassVar[int]
    id: str
    component_id: str
    library_id: str
    scenario: str
    adopted_path: str
    adopted_version: str
    library_version_status: LibraryVersionStatus
    local_status: LocalStatus
    status_detail: str
    created_at: _timestamp_pb2.Timestamp
    refreshed_at: _timestamp_pb2.Timestamp
    source_sha256: str
    applied_at: str
    files: _containers.RepeatedCompositeFieldContainer[AdoptionFile]
    def __init__(self, id: _Optional[str] = ..., component_id: _Optional[str] = ..., library_id: _Optional[str] = ..., scenario: _Optional[str] = ..., adopted_path: _Optional[str] = ..., adopted_version: _Optional[str] = ..., library_version_status: _Optional[_Union[LibraryVersionStatus, str]] = ..., local_status: _Optional[_Union[LocalStatus, str]] = ..., status_detail: _Optional[str] = ..., created_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., refreshed_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., source_sha256: _Optional[str] = ..., applied_at: _Optional[str] = ..., files: _Optional[_Iterable[_Union[AdoptionFile, _Mapping]]] = ...) -> None: ...

class AdoptionFile(_message.Message):
    __slots__ = ("library_path", "adopted_path", "source_sha256", "adopted_snapshot_sha256", "source_asset_id", "source_library_id", "source_version")
    LIBRARY_PATH_FIELD_NUMBER: _ClassVar[int]
    ADOPTED_PATH_FIELD_NUMBER: _ClassVar[int]
    SOURCE_SHA256_FIELD_NUMBER: _ClassVar[int]
    ADOPTED_SNAPSHOT_SHA256_FIELD_NUMBER: _ClassVar[int]
    SOURCE_ASSET_ID_FIELD_NUMBER: _ClassVar[int]
    SOURCE_LIBRARY_ID_FIELD_NUMBER: _ClassVar[int]
    SOURCE_VERSION_FIELD_NUMBER: _ClassVar[int]
    library_path: str
    adopted_path: str
    source_sha256: str
    adopted_snapshot_sha256: str
    source_asset_id: str
    source_library_id: str
    source_version: str
    def __init__(self, library_path: _Optional[str] = ..., adopted_path: _Optional[str] = ..., source_sha256: _Optional[str] = ..., adopted_snapshot_sha256: _Optional[str] = ..., source_asset_id: _Optional[str] = ..., source_library_id: _Optional[str] = ..., source_version: _Optional[str] = ...) -> None: ...

class ListAdoptionsRequest(_message.Message):
    __slots__ = ("component_id", "scenario", "limit")
    COMPONENT_ID_FIELD_NUMBER: _ClassVar[int]
    SCENARIO_FIELD_NUMBER: _ClassVar[int]
    LIMIT_FIELD_NUMBER: _ClassVar[int]
    component_id: str
    scenario: str
    limit: int
    def __init__(self, component_id: _Optional[str] = ..., scenario: _Optional[str] = ..., limit: _Optional[int] = ...) -> None: ...

class ListAdoptionsResponse(_message.Message):
    __slots__ = ("adoptions",)
    ADOPTIONS_FIELD_NUMBER: _ClassVar[int]
    adoptions: _containers.RepeatedCompositeFieldContainer[Adoption]
    def __init__(self, adoptions: _Optional[_Iterable[_Union[Adoption, _Mapping]]] = ...) -> None: ...

class EffectiveAdoption(_message.Message):
    __slots__ = ("source_asset_id", "source_library_id", "source_version", "mediated", "parent_adoption")
    SOURCE_ASSET_ID_FIELD_NUMBER: _ClassVar[int]
    SOURCE_LIBRARY_ID_FIELD_NUMBER: _ClassVar[int]
    SOURCE_VERSION_FIELD_NUMBER: _ClassVar[int]
    MEDIATED_FIELD_NUMBER: _ClassVar[int]
    PARENT_ADOPTION_FIELD_NUMBER: _ClassVar[int]
    source_asset_id: str
    source_library_id: str
    source_version: str
    mediated: bool
    parent_adoption: Adoption
    def __init__(self, source_asset_id: _Optional[str] = ..., source_library_id: _Optional[str] = ..., source_version: _Optional[str] = ..., mediated: _Optional[bool] = ..., parent_adoption: _Optional[_Union[Adoption, _Mapping]] = ...) -> None: ...

class ListEffectiveAdoptionsRequest(_message.Message):
    __slots__ = ("component_id", "limit")
    COMPONENT_ID_FIELD_NUMBER: _ClassVar[int]
    LIMIT_FIELD_NUMBER: _ClassVar[int]
    component_id: str
    limit: int
    def __init__(self, component_id: _Optional[str] = ..., limit: _Optional[int] = ...) -> None: ...

class ListEffectiveAdoptionsResponse(_message.Message):
    __slots__ = ("adoptions",)
    ADOPTIONS_FIELD_NUMBER: _ClassVar[int]
    adoptions: _containers.RepeatedCompositeFieldContainer[EffectiveAdoption]
    def __init__(self, adoptions: _Optional[_Iterable[_Union[EffectiveAdoption, _Mapping]]] = ...) -> None: ...

class ApplyAdoptionRequest(_message.Message):
    __slots__ = ("component_id", "scenario", "adopted_path", "version", "confirm_overwrite", "override_validation", "replace_existing")
    COMPONENT_ID_FIELD_NUMBER: _ClassVar[int]
    SCENARIO_FIELD_NUMBER: _ClassVar[int]
    ADOPTED_PATH_FIELD_NUMBER: _ClassVar[int]
    VERSION_FIELD_NUMBER: _ClassVar[int]
    CONFIRM_OVERWRITE_FIELD_NUMBER: _ClassVar[int]
    OVERRIDE_VALIDATION_FIELD_NUMBER: _ClassVar[int]
    REPLACE_EXISTING_FIELD_NUMBER: _ClassVar[int]
    component_id: str
    scenario: str
    adopted_path: str
    version: str
    confirm_overwrite: bool
    override_validation: bool
    replace_existing: bool
    def __init__(self, component_id: _Optional[str] = ..., scenario: _Optional[str] = ..., adopted_path: _Optional[str] = ..., version: _Optional[str] = ..., confirm_overwrite: _Optional[bool] = ..., override_validation: _Optional[bool] = ..., replace_existing: _Optional[bool] = ...) -> None: ...

class ApplyAdoptionResponse(_message.Message):
    __slots__ = ("adoption", "written_path", "import_sites", "style_fit_affinity", "style_fit_detail")
    ADOPTION_FIELD_NUMBER: _ClassVar[int]
    WRITTEN_PATH_FIELD_NUMBER: _ClassVar[int]
    IMPORT_SITES_FIELD_NUMBER: _ClassVar[int]
    STYLE_FIT_AFFINITY_FIELD_NUMBER: _ClassVar[int]
    STYLE_FIT_DETAIL_FIELD_NUMBER: _ClassVar[int]
    adoption: Adoption
    written_path: str
    import_sites: _containers.RepeatedScalarFieldContainer[str]
    style_fit_affinity: str
    style_fit_detail: str
    def __init__(self, adoption: _Optional[_Union[Adoption, _Mapping]] = ..., written_path: _Optional[str] = ..., import_sites: _Optional[_Iterable[str]] = ..., style_fit_affinity: _Optional[str] = ..., style_fit_detail: _Optional[str] = ...) -> None: ...

class ReapplyAdoptionRequest(_message.Message):
    __slots__ = ("id", "version", "confirm_local_overwrite", "override_validation")
    ID_FIELD_NUMBER: _ClassVar[int]
    VERSION_FIELD_NUMBER: _ClassVar[int]
    CONFIRM_LOCAL_OVERWRITE_FIELD_NUMBER: _ClassVar[int]
    OVERRIDE_VALIDATION_FIELD_NUMBER: _ClassVar[int]
    id: str
    version: str
    confirm_local_overwrite: bool
    override_validation: bool
    def __init__(self, id: _Optional[str] = ..., version: _Optional[str] = ..., confirm_local_overwrite: _Optional[bool] = ..., override_validation: _Optional[bool] = ...) -> None: ...

class ReapplyAdoptionResponse(_message.Message):
    __slots__ = ("adoption", "written_path")
    ADOPTION_FIELD_NUMBER: _ClassVar[int]
    WRITTEN_PATH_FIELD_NUMBER: _ClassVar[int]
    adoption: Adoption
    written_path: str
    def __init__(self, adoption: _Optional[_Union[Adoption, _Mapping]] = ..., written_path: _Optional[str] = ...) -> None: ...

class DeleteAdoptionRequest(_message.Message):
    __slots__ = ("id",)
    ID_FIELD_NUMBER: _ClassVar[int]
    id: str
    def __init__(self, id: _Optional[str] = ...) -> None: ...

class DeleteAdoptionResponse(_message.Message):
    __slots__ = ()
    def __init__(self) -> None: ...

class RefreshAdoptionsRequest(_message.Message):
    __slots__ = ("component_id",)
    COMPONENT_ID_FIELD_NUMBER: _ClassVar[int]
    component_id: str
    def __init__(self, component_id: _Optional[str] = ...) -> None: ...

class ResolveAdoptionPathRequest(_message.Message):
    __slots__ = ("component_id", "scenario", "override_path", "feature", "version", "template")
    COMPONENT_ID_FIELD_NUMBER: _ClassVar[int]
    SCENARIO_FIELD_NUMBER: _ClassVar[int]
    OVERRIDE_PATH_FIELD_NUMBER: _ClassVar[int]
    FEATURE_FIELD_NUMBER: _ClassVar[int]
    VERSION_FIELD_NUMBER: _ClassVar[int]
    TEMPLATE_FIELD_NUMBER: _ClassVar[int]
    component_id: str
    scenario: str
    override_path: str
    feature: str
    version: str
    template: str
    def __init__(self, component_id: _Optional[str] = ..., scenario: _Optional[str] = ..., override_path: _Optional[str] = ..., feature: _Optional[str] = ..., version: _Optional[str] = ..., template: _Optional[str] = ...) -> None: ...

class ResolvedVersionFile(_message.Message):
    __slots__ = ("library_path", "target_path", "slot", "source", "slot_source", "is_entry", "warnings")
    LIBRARY_PATH_FIELD_NUMBER: _ClassVar[int]
    TARGET_PATH_FIELD_NUMBER: _ClassVar[int]
    SLOT_FIELD_NUMBER: _ClassVar[int]
    SOURCE_FIELD_NUMBER: _ClassVar[int]
    SLOT_SOURCE_FIELD_NUMBER: _ClassVar[int]
    IS_ENTRY_FIELD_NUMBER: _ClassVar[int]
    WARNINGS_FIELD_NUMBER: _ClassVar[int]
    library_path: str
    target_path: str
    slot: str
    source: ResolveSource
    slot_source: str
    is_entry: bool
    warnings: _containers.RepeatedScalarFieldContainer[str]
    def __init__(self, library_path: _Optional[str] = ..., target_path: _Optional[str] = ..., slot: _Optional[str] = ..., source: _Optional[_Union[ResolveSource, str]] = ..., slot_source: _Optional[str] = ..., is_entry: _Optional[bool] = ..., warnings: _Optional[_Iterable[str]] = ...) -> None: ...

class ResolveAdoptionPathResponse(_message.Message):
    __slots__ = ("path", "source", "slot", "warnings", "files", "template", "manifest_resolved")
    PATH_FIELD_NUMBER: _ClassVar[int]
    SOURCE_FIELD_NUMBER: _ClassVar[int]
    SLOT_FIELD_NUMBER: _ClassVar[int]
    WARNINGS_FIELD_NUMBER: _ClassVar[int]
    FILES_FIELD_NUMBER: _ClassVar[int]
    TEMPLATE_FIELD_NUMBER: _ClassVar[int]
    MANIFEST_RESOLVED_FIELD_NUMBER: _ClassVar[int]
    path: str
    source: ResolveSource
    slot: str
    warnings: _containers.RepeatedScalarFieldContainer[str]
    files: _containers.RepeatedCompositeFieldContainer[ResolvedVersionFile]
    template: str
    manifest_resolved: bool
    def __init__(self, path: _Optional[str] = ..., source: _Optional[_Union[ResolveSource, str]] = ..., slot: _Optional[str] = ..., warnings: _Optional[_Iterable[str]] = ..., files: _Optional[_Iterable[_Union[ResolvedVersionFile, _Mapping]]] = ..., template: _Optional[str] = ..., manifest_resolved: _Optional[bool] = ...) -> None: ...

class SuggestAdoptionsRequest(_message.Message):
    __slots__ = ("scenario", "limit", "component_id")
    SCENARIO_FIELD_NUMBER: _ClassVar[int]
    LIMIT_FIELD_NUMBER: _ClassVar[int]
    COMPONENT_ID_FIELD_NUMBER: _ClassVar[int]
    scenario: str
    limit: int
    component_id: str
    def __init__(self, scenario: _Optional[str] = ..., limit: _Optional[int] = ..., component_id: _Optional[str] = ...) -> None: ...

class AdoptionSuggestion(_message.Message):
    __slots__ = ("scenario", "component_id", "library_id", "display_name", "inventory_path", "reasons", "classification")
    SCENARIO_FIELD_NUMBER: _ClassVar[int]
    COMPONENT_ID_FIELD_NUMBER: _ClassVar[int]
    LIBRARY_ID_FIELD_NUMBER: _ClassVar[int]
    DISPLAY_NAME_FIELD_NUMBER: _ClassVar[int]
    INVENTORY_PATH_FIELD_NUMBER: _ClassVar[int]
    REASONS_FIELD_NUMBER: _ClassVar[int]
    CLASSIFICATION_FIELD_NUMBER: _ClassVar[int]
    scenario: str
    component_id: str
    library_id: str
    display_name: str
    inventory_path: str
    reasons: _containers.RepeatedScalarFieldContainer[str]
    classification: RecommendationClass
    def __init__(self, scenario: _Optional[str] = ..., component_id: _Optional[str] = ..., library_id: _Optional[str] = ..., display_name: _Optional[str] = ..., inventory_path: _Optional[str] = ..., reasons: _Optional[_Iterable[str]] = ..., classification: _Optional[_Union[RecommendationClass, str]] = ...) -> None: ...

class SuggestAdoptionsResponse(_message.Message):
    __slots__ = ("suggestions",)
    SUGGESTIONS_FIELD_NUMBER: _ClassVar[int]
    suggestions: _containers.RepeatedCompositeFieldContainer[AdoptionSuggestion]
    def __init__(self, suggestions: _Optional[_Iterable[_Union[AdoptionSuggestion, _Mapping]]] = ...) -> None: ...

class RefreshAdoptionsResponse(_message.Message):
    __slots__ = ("adoptions", "library_current", "library_behind", "library_deprecated", "library_missing", "library_unknown", "local_clean", "local_modified", "local_missing", "local_unknown")
    ADOPTIONS_FIELD_NUMBER: _ClassVar[int]
    LIBRARY_CURRENT_FIELD_NUMBER: _ClassVar[int]
    LIBRARY_BEHIND_FIELD_NUMBER: _ClassVar[int]
    LIBRARY_DEPRECATED_FIELD_NUMBER: _ClassVar[int]
    LIBRARY_MISSING_FIELD_NUMBER: _ClassVar[int]
    LIBRARY_UNKNOWN_FIELD_NUMBER: _ClassVar[int]
    LOCAL_CLEAN_FIELD_NUMBER: _ClassVar[int]
    LOCAL_MODIFIED_FIELD_NUMBER: _ClassVar[int]
    LOCAL_MISSING_FIELD_NUMBER: _ClassVar[int]
    LOCAL_UNKNOWN_FIELD_NUMBER: _ClassVar[int]
    adoptions: _containers.RepeatedCompositeFieldContainer[Adoption]
    library_current: int
    library_behind: int
    library_deprecated: int
    library_missing: int
    library_unknown: int
    local_clean: int
    local_modified: int
    local_missing: int
    local_unknown: int
    def __init__(self, adoptions: _Optional[_Iterable[_Union[Adoption, _Mapping]]] = ..., library_current: _Optional[int] = ..., library_behind: _Optional[int] = ..., library_deprecated: _Optional[int] = ..., library_missing: _Optional[int] = ..., library_unknown: _Optional[int] = ..., local_clean: _Optional[int] = ..., local_modified: _Optional[int] = ..., local_missing: _Optional[int] = ..., local_unknown: _Optional[int] = ...) -> None: ...

class ReconcileAdoptionsRequest(_message.Message):
    __slots__ = ("apply",)
    APPLY_FIELD_NUMBER: _ClassVar[int]
    apply: bool
    def __init__(self, apply: _Optional[bool] = ...) -> None: ...

class ReconcileFinding(_message.Message):
    __slots__ = ("scenario", "adopted_path", "library_id", "version", "detail")
    SCENARIO_FIELD_NUMBER: _ClassVar[int]
    ADOPTED_PATH_FIELD_NUMBER: _ClassVar[int]
    LIBRARY_ID_FIELD_NUMBER: _ClassVar[int]
    VERSION_FIELD_NUMBER: _ClassVar[int]
    DETAIL_FIELD_NUMBER: _ClassVar[int]
    scenario: str
    adopted_path: str
    library_id: str
    version: str
    detail: str
    def __init__(self, scenario: _Optional[str] = ..., adopted_path: _Optional[str] = ..., library_id: _Optional[str] = ..., version: _Optional[str] = ..., detail: _Optional[str] = ...) -> None: ...

class ReconcileAdoptionsResponse(_message.Message):
    __slots__ = ("scanned", "already_recorded", "created", "findings")
    SCANNED_FIELD_NUMBER: _ClassVar[int]
    ALREADY_RECORDED_FIELD_NUMBER: _ClassVar[int]
    CREATED_FIELD_NUMBER: _ClassVar[int]
    FINDINGS_FIELD_NUMBER: _ClassVar[int]
    scanned: int
    already_recorded: int
    created: int
    findings: _containers.RepeatedCompositeFieldContainer[ReconcileFinding]
    def __init__(self, scanned: _Optional[int] = ..., already_recorded: _Optional[int] = ..., created: _Optional[int] = ..., findings: _Optional[_Iterable[_Union[ReconcileFinding, _Mapping]]] = ...) -> None: ...

class ReconvergeAdoptionsRequest(_message.Message):
    __slots__ = ("scenario", "apply")
    SCENARIO_FIELD_NUMBER: _ClassVar[int]
    APPLY_FIELD_NUMBER: _ClassVar[int]
    scenario: str
    apply: bool
    def __init__(self, scenario: _Optional[str] = ..., apply: _Optional[bool] = ...) -> None: ...

class ReconvergeFileOutcome(_message.Message):
    __slots__ = ("library_path", "adopted_path", "local_status")
    LIBRARY_PATH_FIELD_NUMBER: _ClassVar[int]
    ADOPTED_PATH_FIELD_NUMBER: _ClassVar[int]
    LOCAL_STATUS_FIELD_NUMBER: _ClassVar[int]
    library_path: str
    adopted_path: str
    local_status: LocalStatus
    def __init__(self, library_path: _Optional[str] = ..., adopted_path: _Optional[str] = ..., local_status: _Optional[_Union[LocalStatus, str]] = ...) -> None: ...

class ReconvergeOutcome(_message.Message):
    __slots__ = ("adoption_id", "scenario", "component_id", "library_id", "adopted_version", "target_version", "library_version_status", "local_status", "action", "detail", "files")
    ADOPTION_ID_FIELD_NUMBER: _ClassVar[int]
    SCENARIO_FIELD_NUMBER: _ClassVar[int]
    COMPONENT_ID_FIELD_NUMBER: _ClassVar[int]
    LIBRARY_ID_FIELD_NUMBER: _ClassVar[int]
    ADOPTED_VERSION_FIELD_NUMBER: _ClassVar[int]
    TARGET_VERSION_FIELD_NUMBER: _ClassVar[int]
    LIBRARY_VERSION_STATUS_FIELD_NUMBER: _ClassVar[int]
    LOCAL_STATUS_FIELD_NUMBER: _ClassVar[int]
    ACTION_FIELD_NUMBER: _ClassVar[int]
    DETAIL_FIELD_NUMBER: _ClassVar[int]
    FILES_FIELD_NUMBER: _ClassVar[int]
    adoption_id: str
    scenario: str
    component_id: str
    library_id: str
    adopted_version: str
    target_version: str
    library_version_status: LibraryVersionStatus
    local_status: LocalStatus
    action: ReconvergeAction
    detail: str
    files: _containers.RepeatedCompositeFieldContainer[ReconvergeFileOutcome]
    def __init__(self, adoption_id: _Optional[str] = ..., scenario: _Optional[str] = ..., component_id: _Optional[str] = ..., library_id: _Optional[str] = ..., adopted_version: _Optional[str] = ..., target_version: _Optional[str] = ..., library_version_status: _Optional[_Union[LibraryVersionStatus, str]] = ..., local_status: _Optional[_Union[LocalStatus, str]] = ..., action: _Optional[_Union[ReconvergeAction, str]] = ..., detail: _Optional[str] = ..., files: _Optional[_Iterable[_Union[ReconvergeFileOutcome, _Mapping]]] = ...) -> None: ...

class ReconvergeAdoptionsResponse(_message.Message):
    __slots__ = ("scanned", "behind", "reapplied", "flagged", "skipped", "errored", "outcomes")
    SCANNED_FIELD_NUMBER: _ClassVar[int]
    BEHIND_FIELD_NUMBER: _ClassVar[int]
    REAPPLIED_FIELD_NUMBER: _ClassVar[int]
    FLAGGED_FIELD_NUMBER: _ClassVar[int]
    SKIPPED_FIELD_NUMBER: _ClassVar[int]
    ERRORED_FIELD_NUMBER: _ClassVar[int]
    OUTCOMES_FIELD_NUMBER: _ClassVar[int]
    scanned: int
    behind: int
    reapplied: int
    flagged: int
    skipped: int
    errored: int
    outcomes: _containers.RepeatedCompositeFieldContainer[ReconvergeOutcome]
    def __init__(self, scanned: _Optional[int] = ..., behind: _Optional[int] = ..., reapplied: _Optional[int] = ..., flagged: _Optional[int] = ..., skipped: _Optional[int] = ..., errored: _Optional[int] = ..., outcomes: _Optional[_Iterable[_Union[ReconvergeOutcome, _Mapping]]] = ...) -> None: ...

class DiscoverAdoptionsRequest(_message.Message):
    __slots__ = ("scenario", "min_similarity", "limit")
    SCENARIO_FIELD_NUMBER: _ClassVar[int]
    MIN_SIMILARITY_FIELD_NUMBER: _ClassVar[int]
    LIMIT_FIELD_NUMBER: _ClassVar[int]
    scenario: str
    min_similarity: float
    limit: int
    def __init__(self, scenario: _Optional[str] = ..., min_similarity: _Optional[float] = ..., limit: _Optional[int] = ...) -> None: ...

class DiscoveryCandidate(_message.Message):
    __slots__ = ("scenario", "adopted_path", "component_id", "library_id", "version", "display_name", "similarity", "shared_lines", "candidate_lines", "source_lines", "basename_match", "evidence")
    SCENARIO_FIELD_NUMBER: _ClassVar[int]
    ADOPTED_PATH_FIELD_NUMBER: _ClassVar[int]
    COMPONENT_ID_FIELD_NUMBER: _ClassVar[int]
    LIBRARY_ID_FIELD_NUMBER: _ClassVar[int]
    VERSION_FIELD_NUMBER: _ClassVar[int]
    DISPLAY_NAME_FIELD_NUMBER: _ClassVar[int]
    SIMILARITY_FIELD_NUMBER: _ClassVar[int]
    SHARED_LINES_FIELD_NUMBER: _ClassVar[int]
    CANDIDATE_LINES_FIELD_NUMBER: _ClassVar[int]
    SOURCE_LINES_FIELD_NUMBER: _ClassVar[int]
    BASENAME_MATCH_FIELD_NUMBER: _ClassVar[int]
    EVIDENCE_FIELD_NUMBER: _ClassVar[int]
    scenario: str
    adopted_path: str
    component_id: str
    library_id: str
    version: str
    display_name: str
    similarity: float
    shared_lines: int
    candidate_lines: int
    source_lines: int
    basename_match: bool
    evidence: _containers.RepeatedScalarFieldContainer[str]
    def __init__(self, scenario: _Optional[str] = ..., adopted_path: _Optional[str] = ..., component_id: _Optional[str] = ..., library_id: _Optional[str] = ..., version: _Optional[str] = ..., display_name: _Optional[str] = ..., similarity: _Optional[float] = ..., shared_lines: _Optional[int] = ..., candidate_lines: _Optional[int] = ..., source_lines: _Optional[int] = ..., basename_match: _Optional[bool] = ..., evidence: _Optional[_Iterable[str]] = ...) -> None: ...

class DiscoverAdoptionsResponse(_message.Message):
    __slots__ = ("scanned", "min_similarity", "candidates")
    SCANNED_FIELD_NUMBER: _ClassVar[int]
    MIN_SIMILARITY_FIELD_NUMBER: _ClassVar[int]
    CANDIDATES_FIELD_NUMBER: _ClassVar[int]
    scanned: int
    min_similarity: float
    candidates: _containers.RepeatedCompositeFieldContainer[DiscoveryCandidate]
    def __init__(self, scanned: _Optional[int] = ..., min_similarity: _Optional[float] = ..., candidates: _Optional[_Iterable[_Union[DiscoveryCandidate, _Mapping]]] = ...) -> None: ...

class ConfirmDiscoveryRequest(_message.Message):
    __slots__ = ("scenario", "adopted_path", "component_id", "version")
    SCENARIO_FIELD_NUMBER: _ClassVar[int]
    ADOPTED_PATH_FIELD_NUMBER: _ClassVar[int]
    COMPONENT_ID_FIELD_NUMBER: _ClassVar[int]
    VERSION_FIELD_NUMBER: _ClassVar[int]
    scenario: str
    adopted_path: str
    component_id: str
    version: str
    def __init__(self, scenario: _Optional[str] = ..., adopted_path: _Optional[str] = ..., component_id: _Optional[str] = ..., version: _Optional[str] = ...) -> None: ...

class ConfirmDiscoveryResponse(_message.Message):
    __slots__ = ("adoption", "written_path", "similarity")
    ADOPTION_FIELD_NUMBER: _ClassVar[int]
    WRITTEN_PATH_FIELD_NUMBER: _ClassVar[int]
    SIMILARITY_FIELD_NUMBER: _ClassVar[int]
    adoption: Adoption
    written_path: str
    similarity: float
    def __init__(self, adoption: _Optional[_Union[Adoption, _Mapping]] = ..., written_path: _Optional[str] = ..., similarity: _Optional[float] = ...) -> None: ...
