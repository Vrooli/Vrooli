from google.protobuf.internal import containers as _containers
from google.protobuf.internal import enum_type_wrapper as _enum_type_wrapper
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Iterable as _Iterable, Mapping as _Mapping
from typing import ClassVar as _ClassVar, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class CommercialUse(int, metaclass=_enum_type_wrapper.EnumTypeWrapper):
    __slots__ = ()
    COMMERCIAL_USE_UNSPECIFIED: _ClassVar[CommercialUse]
    COMMERCIAL_USE_YES: _ClassVar[CommercialUse]
    COMMERCIAL_USE_NO: _ClassVar[CommercialUse]
    COMMERCIAL_USE_CONDITIONAL: _ClassVar[CommercialUse]

class ModelLayout(int, metaclass=_enum_type_wrapper.EnumTypeWrapper):
    __slots__ = ()
    MODEL_LAYOUT_UNSPECIFIED: _ClassVar[ModelLayout]
    MODEL_LAYOUT_SINGLE_FILE: _ClassVar[ModelLayout]
    MODEL_LAYOUT_DIFFUSERS_REPO: _ClassVar[ModelLayout]

class CatalogFindingSeverity(int, metaclass=_enum_type_wrapper.EnumTypeWrapper):
    __slots__ = ()
    CATALOG_FINDING_SEVERITY_UNSPECIFIED: _ClassVar[CatalogFindingSeverity]
    CATALOG_FINDING_SEVERITY_ERROR: _ClassVar[CatalogFindingSeverity]
    CATALOG_FINDING_SEVERITY_WARNING: _ClassVar[CatalogFindingSeverity]
COMMERCIAL_USE_UNSPECIFIED: CommercialUse
COMMERCIAL_USE_YES: CommercialUse
COMMERCIAL_USE_NO: CommercialUse
COMMERCIAL_USE_CONDITIONAL: CommercialUse
MODEL_LAYOUT_UNSPECIFIED: ModelLayout
MODEL_LAYOUT_SINGLE_FILE: ModelLayout
MODEL_LAYOUT_DIFFUSERS_REPO: ModelLayout
CATALOG_FINDING_SEVERITY_UNSPECIFIED: CatalogFindingSeverity
CATALOG_FINDING_SEVERITY_ERROR: CatalogFindingSeverity
CATALOG_FINDING_SEVERITY_WARNING: CatalogFindingSeverity

class Hardware(_message.Message):
    __slots__ = ("cpu_capable", "gpu_required", "min_vram_gb", "min_ram_gb", "os_arch", "speed_note")
    CPU_CAPABLE_FIELD_NUMBER: _ClassVar[int]
    GPU_REQUIRED_FIELD_NUMBER: _ClassVar[int]
    MIN_VRAM_GB_FIELD_NUMBER: _ClassVar[int]
    MIN_RAM_GB_FIELD_NUMBER: _ClassVar[int]
    OS_ARCH_FIELD_NUMBER: _ClassVar[int]
    SPEED_NOTE_FIELD_NUMBER: _ClassVar[int]
    cpu_capable: bool
    gpu_required: bool
    min_vram_gb: int
    min_ram_gb: int
    os_arch: _containers.RepeatedScalarFieldContainer[str]
    speed_note: str
    def __init__(self, cpu_capable: _Optional[bool] = ..., gpu_required: _Optional[bool] = ..., min_vram_gb: _Optional[int] = ..., min_ram_gb: _Optional[int] = ..., os_arch: _Optional[_Iterable[str]] = ..., speed_note: _Optional[str] = ...) -> None: ...

class CapabilityLabels(_message.Message):
    __slots__ = ("nsfw_capable", "license", "commercial_use", "commercial_use_notes", "base_model_lineage", "known_risks")
    NSFW_CAPABLE_FIELD_NUMBER: _ClassVar[int]
    LICENSE_FIELD_NUMBER: _ClassVar[int]
    COMMERCIAL_USE_FIELD_NUMBER: _ClassVar[int]
    COMMERCIAL_USE_NOTES_FIELD_NUMBER: _ClassVar[int]
    BASE_MODEL_LINEAGE_FIELD_NUMBER: _ClassVar[int]
    KNOWN_RISKS_FIELD_NUMBER: _ClassVar[int]
    nsfw_capable: bool
    license: str
    commercial_use: CommercialUse
    commercial_use_notes: str
    base_model_lineage: str
    known_risks: str
    def __init__(self, nsfw_capable: _Optional[bool] = ..., license: _Optional[str] = ..., commercial_use: _Optional[_Union[CommercialUse, str]] = ..., commercial_use_notes: _Optional[str] = ..., base_model_lineage: _Optional[str] = ..., known_risks: _Optional[str] = ...) -> None: ...

class Model(_message.Message):
    __slots__ = ("id", "name", "operations", "default_for", "tier", "backend", "alt_backends", "requires_comfyui", "size_mb_approx", "quant_variants", "hardware", "capability_labels", "enabled", "install", "custom", "geometry")
    ID_FIELD_NUMBER: _ClassVar[int]
    NAME_FIELD_NUMBER: _ClassVar[int]
    OPERATIONS_FIELD_NUMBER: _ClassVar[int]
    DEFAULT_FOR_FIELD_NUMBER: _ClassVar[int]
    TIER_FIELD_NUMBER: _ClassVar[int]
    BACKEND_FIELD_NUMBER: _ClassVar[int]
    ALT_BACKENDS_FIELD_NUMBER: _ClassVar[int]
    REQUIRES_COMFYUI_FIELD_NUMBER: _ClassVar[int]
    SIZE_MB_APPROX_FIELD_NUMBER: _ClassVar[int]
    QUANT_VARIANTS_FIELD_NUMBER: _ClassVar[int]
    HARDWARE_FIELD_NUMBER: _ClassVar[int]
    CAPABILITY_LABELS_FIELD_NUMBER: _ClassVar[int]
    ENABLED_FIELD_NUMBER: _ClassVar[int]
    INSTALL_FIELD_NUMBER: _ClassVar[int]
    CUSTOM_FIELD_NUMBER: _ClassVar[int]
    GEOMETRY_FIELD_NUMBER: _ClassVar[int]
    id: str
    name: str
    operations: _containers.RepeatedScalarFieldContainer[str]
    default_for: _containers.RepeatedScalarFieldContainer[str]
    tier: str
    backend: str
    alt_backends: _containers.RepeatedScalarFieldContainer[str]
    requires_comfyui: bool
    size_mb_approx: int
    quant_variants: _containers.RepeatedScalarFieldContainer[str]
    hardware: Hardware
    capability_labels: CapabilityLabels
    enabled: bool
    install: InstallState
    custom: bool
    geometry: Geometry
    def __init__(self, id: _Optional[str] = ..., name: _Optional[str] = ..., operations: _Optional[_Iterable[str]] = ..., default_for: _Optional[_Iterable[str]] = ..., tier: _Optional[str] = ..., backend: _Optional[str] = ..., alt_backends: _Optional[_Iterable[str]] = ..., requires_comfyui: _Optional[bool] = ..., size_mb_approx: _Optional[int] = ..., quant_variants: _Optional[_Iterable[str]] = ..., hardware: _Optional[_Union[Hardware, _Mapping]] = ..., capability_labels: _Optional[_Union[CapabilityLabels, _Mapping]] = ..., enabled: _Optional[bool] = ..., install: _Optional[_Union[InstallState, _Mapping]] = ..., custom: _Optional[bool] = ..., geometry: _Optional[_Union[Geometry, _Mapping]] = ...) -> None: ...

class Geometry(_message.Message):
    __slots__ = ("native_width", "native_height", "size_quantum", "max_edge")
    NATIVE_WIDTH_FIELD_NUMBER: _ClassVar[int]
    NATIVE_HEIGHT_FIELD_NUMBER: _ClassVar[int]
    SIZE_QUANTUM_FIELD_NUMBER: _ClassVar[int]
    MAX_EDGE_FIELD_NUMBER: _ClassVar[int]
    native_width: int
    native_height: int
    size_quantum: int
    max_edge: int
    def __init__(self, native_width: _Optional[int] = ..., native_height: _Optional[int] = ..., size_quantum: _Optional[int] = ..., max_edge: _Optional[int] = ...) -> None: ...

class InstallState(_message.Message):
    __slots__ = ("installed", "path", "checksum", "size_bytes", "installed_at")
    INSTALLED_FIELD_NUMBER: _ClassVar[int]
    PATH_FIELD_NUMBER: _ClassVar[int]
    CHECKSUM_FIELD_NUMBER: _ClassVar[int]
    SIZE_BYTES_FIELD_NUMBER: _ClassVar[int]
    INSTALLED_AT_FIELD_NUMBER: _ClassVar[int]
    installed: bool
    path: str
    checksum: str
    size_bytes: int
    installed_at: str
    def __init__(self, installed: _Optional[bool] = ..., path: _Optional[str] = ..., checksum: _Optional[str] = ..., size_bytes: _Optional[int] = ..., installed_at: _Optional[str] = ...) -> None: ...

class BlocklistEntry(_message.Message):
    __slots__ = ("id", "operations", "license", "reason", "exporting_onnx_removes_restriction")
    ID_FIELD_NUMBER: _ClassVar[int]
    OPERATIONS_FIELD_NUMBER: _ClassVar[int]
    LICENSE_FIELD_NUMBER: _ClassVar[int]
    REASON_FIELD_NUMBER: _ClassVar[int]
    EXPORTING_ONNX_REMOVES_RESTRICTION_FIELD_NUMBER: _ClassVar[int]
    id: str
    operations: _containers.RepeatedScalarFieldContainer[str]
    license: str
    reason: str
    exporting_onnx_removes_restriction: bool
    def __init__(self, id: _Optional[str] = ..., operations: _Optional[_Iterable[str]] = ..., license: _Optional[str] = ..., reason: _Optional[str] = ..., exporting_onnx_removes_restriction: _Optional[bool] = ...) -> None: ...

class ListModelsRequest(_message.Message):
    __slots__ = ("operation",)
    OPERATION_FIELD_NUMBER: _ClassVar[int]
    operation: str
    def __init__(self, operation: _Optional[str] = ...) -> None: ...

class ListModelsResponse(_message.Message):
    __slots__ = ("models",)
    MODELS_FIELD_NUMBER: _ClassVar[int]
    models: _containers.RepeatedCompositeFieldContainer[Model]
    def __init__(self, models: _Optional[_Iterable[_Union[Model, _Mapping]]] = ...) -> None: ...

class GetModelRequest(_message.Message):
    __slots__ = ("id",)
    ID_FIELD_NUMBER: _ClassVar[int]
    id: str
    def __init__(self, id: _Optional[str] = ...) -> None: ...

class GetModelResponse(_message.Message):
    __slots__ = ("model",)
    MODEL_FIELD_NUMBER: _ClassVar[int]
    model: Model
    def __init__(self, model: _Optional[_Union[Model, _Mapping]] = ...) -> None: ...

class ListOperationsRequest(_message.Message):
    __slots__ = ()
    def __init__(self) -> None: ...

class ListOperationsResponse(_message.Message):
    __slots__ = ("operations",)
    OPERATIONS_FIELD_NUMBER: _ClassVar[int]
    operations: _containers.RepeatedScalarFieldContainer[str]
    def __init__(self, operations: _Optional[_Iterable[str]] = ...) -> None: ...

class SelectModelRequest(_message.Message):
    __slots__ = ("operation", "override_id", "quality_policy", "allow_byok")
    OPERATION_FIELD_NUMBER: _ClassVar[int]
    OVERRIDE_ID_FIELD_NUMBER: _ClassVar[int]
    QUALITY_POLICY_FIELD_NUMBER: _ClassVar[int]
    ALLOW_BYOK_FIELD_NUMBER: _ClassVar[int]
    operation: str
    override_id: str
    quality_policy: str
    allow_byok: bool
    def __init__(self, operation: _Optional[str] = ..., override_id: _Optional[str] = ..., quality_policy: _Optional[str] = ..., allow_byok: _Optional[bool] = ...) -> None: ...

class SelectModelResponse(_message.Message):
    __slots__ = ("model", "gpu_viable", "reason", "warnings")
    MODEL_FIELD_NUMBER: _ClassVar[int]
    GPU_VIABLE_FIELD_NUMBER: _ClassVar[int]
    REASON_FIELD_NUMBER: _ClassVar[int]
    WARNINGS_FIELD_NUMBER: _ClassVar[int]
    model: Model
    gpu_viable: bool
    reason: str
    warnings: _containers.RepeatedScalarFieldContainer[str]
    def __init__(self, model: _Optional[_Union[Model, _Mapping]] = ..., gpu_viable: _Optional[bool] = ..., reason: _Optional[str] = ..., warnings: _Optional[_Iterable[str]] = ...) -> None: ...

class SetModelEnabledRequest(_message.Message):
    __slots__ = ("id", "enabled")
    ID_FIELD_NUMBER: _ClassVar[int]
    ENABLED_FIELD_NUMBER: _ClassVar[int]
    id: str
    enabled: bool
    def __init__(self, id: _Optional[str] = ..., enabled: _Optional[bool] = ...) -> None: ...

class SetModelEnabledResponse(_message.Message):
    __slots__ = ("model",)
    MODEL_FIELD_NUMBER: _ClassVar[int]
    model: Model
    def __init__(self, model: _Optional[_Union[Model, _Mapping]] = ...) -> None: ...

class ListBlocklistRequest(_message.Message):
    __slots__ = ()
    def __init__(self) -> None: ...

class ListBlocklistResponse(_message.Message):
    __slots__ = ("entries",)
    ENTRIES_FIELD_NUMBER: _ClassVar[int]
    entries: _containers.RepeatedCompositeFieldContainer[BlocklistEntry]
    def __init__(self, entries: _Optional[_Iterable[_Union[BlocklistEntry, _Mapping]]] = ...) -> None: ...

class InstallModelRequest(_message.Message):
    __slots__ = ("id",)
    ID_FIELD_NUMBER: _ClassVar[int]
    id: str
    def __init__(self, id: _Optional[str] = ...) -> None: ...

class InstallModelResponse(_message.Message):
    __slots__ = ("job_id", "eta_seconds", "size_mb_approx", "already_installed")
    JOB_ID_FIELD_NUMBER: _ClassVar[int]
    ETA_SECONDS_FIELD_NUMBER: _ClassVar[int]
    SIZE_MB_APPROX_FIELD_NUMBER: _ClassVar[int]
    ALREADY_INSTALLED_FIELD_NUMBER: _ClassVar[int]
    job_id: str
    eta_seconds: int
    size_mb_approx: int
    already_installed: bool
    def __init__(self, job_id: _Optional[str] = ..., eta_seconds: _Optional[int] = ..., size_mb_approx: _Optional[int] = ..., already_installed: _Optional[bool] = ...) -> None: ...

class RemoveModelRequest(_message.Message):
    __slots__ = ("id",)
    ID_FIELD_NUMBER: _ClassVar[int]
    id: str
    def __init__(self, id: _Optional[str] = ...) -> None: ...

class RemoveModelResponse(_message.Message):
    __slots__ = ("removed",)
    REMOVED_FIELD_NUMBER: _ClassVar[int]
    removed: bool
    def __init__(self, removed: _Optional[bool] = ...) -> None: ...

class AddCustomModelRequest(_message.Message):
    __slots__ = ("model", "local_path", "download_url")
    MODEL_FIELD_NUMBER: _ClassVar[int]
    LOCAL_PATH_FIELD_NUMBER: _ClassVar[int]
    DOWNLOAD_URL_FIELD_NUMBER: _ClassVar[int]
    model: Model
    local_path: str
    download_url: str
    def __init__(self, model: _Optional[_Union[Model, _Mapping]] = ..., local_path: _Optional[str] = ..., download_url: _Optional[str] = ...) -> None: ...

class AddCustomModelResponse(_message.Message):
    __slots__ = ("model",)
    MODEL_FIELD_NUMBER: _ClassVar[int]
    model: Model
    def __init__(self, model: _Optional[_Union[Model, _Mapping]] = ...) -> None: ...

class ArchitectureInference(_message.Message):
    __slots__ = ("architecture", "confidence", "evidence")
    ARCHITECTURE_FIELD_NUMBER: _ClassVar[int]
    CONFIDENCE_FIELD_NUMBER: _ClassVar[int]
    EVIDENCE_FIELD_NUMBER: _ClassVar[int]
    architecture: str
    confidence: str
    evidence: str
    def __init__(self, architecture: _Optional[str] = ..., confidence: _Optional[str] = ..., evidence: _Optional[str] = ...) -> None: ...

class InspectModelSourceRequest(_message.Message):
    __slots__ = ("source",)
    SOURCE_FIELD_NUMBER: _ClassVar[int]
    source: str
    def __init__(self, source: _Optional[str] = ...) -> None: ...

class InspectModelSourceResponse(_message.Message):
    __slots__ = ("source", "repo_id", "revision", "layout", "architecture", "license", "nsfw", "size_bytes", "pipeline_class", "offered_operations", "proposed")
    SOURCE_FIELD_NUMBER: _ClassVar[int]
    REPO_ID_FIELD_NUMBER: _ClassVar[int]
    REVISION_FIELD_NUMBER: _ClassVar[int]
    LAYOUT_FIELD_NUMBER: _ClassVar[int]
    ARCHITECTURE_FIELD_NUMBER: _ClassVar[int]
    LICENSE_FIELD_NUMBER: _ClassVar[int]
    NSFW_FIELD_NUMBER: _ClassVar[int]
    SIZE_BYTES_FIELD_NUMBER: _ClassVar[int]
    PIPELINE_CLASS_FIELD_NUMBER: _ClassVar[int]
    OFFERED_OPERATIONS_FIELD_NUMBER: _ClassVar[int]
    PROPOSED_FIELD_NUMBER: _ClassVar[int]
    source: str
    repo_id: str
    revision: str
    layout: ModelLayout
    architecture: ArchitectureInference
    license: str
    nsfw: bool
    size_bytes: int
    pipeline_class: str
    offered_operations: _containers.RepeatedScalarFieldContainer[str]
    proposed: Model
    def __init__(self, source: _Optional[str] = ..., repo_id: _Optional[str] = ..., revision: _Optional[str] = ..., layout: _Optional[_Union[ModelLayout, str]] = ..., architecture: _Optional[_Union[ArchitectureInference, _Mapping]] = ..., license: _Optional[str] = ..., nsfw: _Optional[bool] = ..., size_bytes: _Optional[int] = ..., pipeline_class: _Optional[str] = ..., offered_operations: _Optional[_Iterable[str]] = ..., proposed: _Optional[_Union[Model, _Mapping]] = ...) -> None: ...

class ImportModelRequest(_message.Message):
    __slots__ = ("source", "id", "name", "architecture", "operations", "attest_commercial_rights")
    SOURCE_FIELD_NUMBER: _ClassVar[int]
    ID_FIELD_NUMBER: _ClassVar[int]
    NAME_FIELD_NUMBER: _ClassVar[int]
    ARCHITECTURE_FIELD_NUMBER: _ClassVar[int]
    OPERATIONS_FIELD_NUMBER: _ClassVar[int]
    ATTEST_COMMERCIAL_RIGHTS_FIELD_NUMBER: _ClassVar[int]
    source: str
    id: str
    name: str
    architecture: str
    operations: _containers.RepeatedScalarFieldContainer[str]
    attest_commercial_rights: bool
    def __init__(self, source: _Optional[str] = ..., id: _Optional[str] = ..., name: _Optional[str] = ..., architecture: _Optional[str] = ..., operations: _Optional[_Iterable[str]] = ..., attest_commercial_rights: _Optional[bool] = ...) -> None: ...

class ImportModelResponse(_message.Message):
    __slots__ = ("model", "job_id", "eta_seconds", "already_installed")
    MODEL_FIELD_NUMBER: _ClassVar[int]
    JOB_ID_FIELD_NUMBER: _ClassVar[int]
    ETA_SECONDS_FIELD_NUMBER: _ClassVar[int]
    ALREADY_INSTALLED_FIELD_NUMBER: _ClassVar[int]
    model: Model
    job_id: str
    eta_seconds: int
    already_installed: bool
    def __init__(self, model: _Optional[_Union[Model, _Mapping]] = ..., job_id: _Optional[str] = ..., eta_seconds: _Optional[int] = ..., already_installed: _Optional[bool] = ...) -> None: ...

class SetDefaultModelRequest(_message.Message):
    __slots__ = ("operation", "model_id")
    OPERATION_FIELD_NUMBER: _ClassVar[int]
    MODEL_ID_FIELD_NUMBER: _ClassVar[int]
    operation: str
    model_id: str
    def __init__(self, operation: _Optional[str] = ..., model_id: _Optional[str] = ...) -> None: ...

class SetDefaultModelResponse(_message.Message):
    __slots__ = ("operation", "model_id")
    OPERATION_FIELD_NUMBER: _ClassVar[int]
    MODEL_ID_FIELD_NUMBER: _ClassVar[int]
    operation: str
    model_id: str
    def __init__(self, operation: _Optional[str] = ..., model_id: _Optional[str] = ...) -> None: ...

class OpDefault(_message.Message):
    __slots__ = ("operation", "model_id", "source")
    OPERATION_FIELD_NUMBER: _ClassVar[int]
    MODEL_ID_FIELD_NUMBER: _ClassVar[int]
    SOURCE_FIELD_NUMBER: _ClassVar[int]
    operation: str
    model_id: str
    source: str
    def __init__(self, operation: _Optional[str] = ..., model_id: _Optional[str] = ..., source: _Optional[str] = ...) -> None: ...

class ListDefaultsRequest(_message.Message):
    __slots__ = ()
    def __init__(self) -> None: ...

class ListDefaultsResponse(_message.Message):
    __slots__ = ("defaults",)
    DEFAULTS_FIELD_NUMBER: _ClassVar[int]
    defaults: _containers.RepeatedCompositeFieldContainer[OpDefault]
    def __init__(self, defaults: _Optional[_Iterable[_Union[OpDefault, _Mapping]]] = ...) -> None: ...

class CatalogFinding(_message.Message):
    __slots__ = ("severity", "code", "model_id", "operation", "message")
    SEVERITY_FIELD_NUMBER: _ClassVar[int]
    CODE_FIELD_NUMBER: _ClassVar[int]
    MODEL_ID_FIELD_NUMBER: _ClassVar[int]
    OPERATION_FIELD_NUMBER: _ClassVar[int]
    MESSAGE_FIELD_NUMBER: _ClassVar[int]
    severity: CatalogFindingSeverity
    code: str
    model_id: str
    operation: str
    message: str
    def __init__(self, severity: _Optional[_Union[CatalogFindingSeverity, str]] = ..., code: _Optional[str] = ..., model_id: _Optional[str] = ..., operation: _Optional[str] = ..., message: _Optional[str] = ...) -> None: ...

class DoctorCatalogRequest(_message.Message):
    __slots__ = ()
    def __init__(self) -> None: ...

class DoctorCatalogResponse(_message.Message):
    __slots__ = ("ok", "findings")
    OK_FIELD_NUMBER: _ClassVar[int]
    FINDINGS_FIELD_NUMBER: _ClassVar[int]
    ok: bool
    findings: _containers.RepeatedCompositeFieldContainer[CatalogFinding]
    def __init__(self, ok: _Optional[bool] = ..., findings: _Optional[_Iterable[_Union[CatalogFinding, _Mapping]]] = ...) -> None: ...

class BackendStatus(_message.Message):
    __slots__ = ("name", "operations", "available", "standalone", "cloud", "gpu_capable", "detail", "provision", "host_tool", "host_tool_ready", "remediation")
    NAME_FIELD_NUMBER: _ClassVar[int]
    OPERATIONS_FIELD_NUMBER: _ClassVar[int]
    AVAILABLE_FIELD_NUMBER: _ClassVar[int]
    STANDALONE_FIELD_NUMBER: _ClassVar[int]
    CLOUD_FIELD_NUMBER: _ClassVar[int]
    GPU_CAPABLE_FIELD_NUMBER: _ClassVar[int]
    DETAIL_FIELD_NUMBER: _ClassVar[int]
    PROVISION_FIELD_NUMBER: _ClassVar[int]
    HOST_TOOL_FIELD_NUMBER: _ClassVar[int]
    HOST_TOOL_READY_FIELD_NUMBER: _ClassVar[int]
    REMEDIATION_FIELD_NUMBER: _ClassVar[int]
    name: str
    operations: _containers.RepeatedScalarFieldContainer[str]
    available: bool
    standalone: bool
    cloud: bool
    gpu_capable: bool
    detail: str
    provision: str
    host_tool: str
    host_tool_ready: bool
    remediation: str
    def __init__(self, name: _Optional[str] = ..., operations: _Optional[_Iterable[str]] = ..., available: _Optional[bool] = ..., standalone: _Optional[bool] = ..., cloud: _Optional[bool] = ..., gpu_capable: _Optional[bool] = ..., detail: _Optional[str] = ..., provision: _Optional[str] = ..., host_tool: _Optional[str] = ..., host_tool_ready: _Optional[bool] = ..., remediation: _Optional[str] = ...) -> None: ...

class DoctorBackendsRequest(_message.Message):
    __slots__ = ()
    def __init__(self) -> None: ...

class DoctorBackendsResponse(_message.Message):
    __slots__ = ("ok", "backends")
    OK_FIELD_NUMBER: _ClassVar[int]
    BACKENDS_FIELD_NUMBER: _ClassVar[int]
    ok: bool
    backends: _containers.RepeatedCompositeFieldContainer[BackendStatus]
    def __init__(self, ok: _Optional[bool] = ..., backends: _Optional[_Iterable[_Union[BackendStatus, _Mapping]]] = ...) -> None: ...

class EnsureBackendRequest(_message.Message):
    __slots__ = ("tool", "operation", "dry_run")
    TOOL_FIELD_NUMBER: _ClassVar[int]
    OPERATION_FIELD_NUMBER: _ClassVar[int]
    DRY_RUN_FIELD_NUMBER: _ClassVar[int]
    tool: str
    operation: str
    dry_run: bool
    def __init__(self, tool: _Optional[str] = ..., operation: _Optional[str] = ..., dry_run: _Optional[bool] = ...) -> None: ...

class EnsureBackendResponse(_message.Message):
    __slots__ = ("tool", "job_id", "eta_seconds", "already_installed", "manual", "state", "detail")
    TOOL_FIELD_NUMBER: _ClassVar[int]
    JOB_ID_FIELD_NUMBER: _ClassVar[int]
    ETA_SECONDS_FIELD_NUMBER: _ClassVar[int]
    ALREADY_INSTALLED_FIELD_NUMBER: _ClassVar[int]
    MANUAL_FIELD_NUMBER: _ClassVar[int]
    STATE_FIELD_NUMBER: _ClassVar[int]
    DETAIL_FIELD_NUMBER: _ClassVar[int]
    tool: str
    job_id: str
    eta_seconds: int
    already_installed: bool
    manual: bool
    state: str
    detail: str
    def __init__(self, tool: _Optional[str] = ..., job_id: _Optional[str] = ..., eta_seconds: _Optional[int] = ..., already_installed: _Optional[bool] = ..., manual: _Optional[bool] = ..., state: _Optional[str] = ..., detail: _Optional[str] = ...) -> None: ...

class GetHostSummaryRequest(_message.Message):
    __slots__ = ()
    def __init__(self) -> None: ...

class GetHostSummaryResponse(_message.Message):
    __slots__ = ("host",)
    HOST_FIELD_NUMBER: _ClassVar[int]
    host: HostSummary
    def __init__(self, host: _Optional[_Union[HostSummary, _Mapping]] = ...) -> None: ...

class ListOperationModelsRequest(_message.Message):
    __slots__ = ("operation",)
    OPERATION_FIELD_NUMBER: _ClassVar[int]
    operation: str
    def __init__(self, operation: _Optional[str] = ...) -> None: ...

class Resolution(_message.Message):
    __slots__ = ("operation", "model_id", "model_name", "support", "technique", "pipeline_class", "caveat", "weight", "tier", "gpu_viable", "warnings", "adapters")
    OPERATION_FIELD_NUMBER: _ClassVar[int]
    MODEL_ID_FIELD_NUMBER: _ClassVar[int]
    MODEL_NAME_FIELD_NUMBER: _ClassVar[int]
    SUPPORT_FIELD_NUMBER: _ClassVar[int]
    TECHNIQUE_FIELD_NUMBER: _ClassVar[int]
    PIPELINE_CLASS_FIELD_NUMBER: _ClassVar[int]
    CAVEAT_FIELD_NUMBER: _ClassVar[int]
    WEIGHT_FIELD_NUMBER: _ClassVar[int]
    TIER_FIELD_NUMBER: _ClassVar[int]
    GPU_VIABLE_FIELD_NUMBER: _ClassVar[int]
    WARNINGS_FIELD_NUMBER: _ClassVar[int]
    ADAPTERS_FIELD_NUMBER: _ClassVar[int]
    operation: str
    model_id: str
    model_name: str
    support: str
    technique: str
    pipeline_class: str
    caveat: str
    weight: str
    tier: str
    gpu_viable: bool
    warnings: _containers.RepeatedScalarFieldContainer[str]
    adapters: _containers.RepeatedCompositeFieldContainer[ResolvedAdapter]
    def __init__(self, operation: _Optional[str] = ..., model_id: _Optional[str] = ..., model_name: _Optional[str] = ..., support: _Optional[str] = ..., technique: _Optional[str] = ..., pipeline_class: _Optional[str] = ..., caveat: _Optional[str] = ..., weight: _Optional[str] = ..., tier: _Optional[str] = ..., gpu_viable: _Optional[bool] = ..., warnings: _Optional[_Iterable[str]] = ..., adapters: _Optional[_Iterable[_Union[ResolvedAdapter, _Mapping]]] = ...) -> None: ...

class AdapterRef(_message.Message):
    __slots__ = ("adapter_id", "scale", "conditioning_image_key", "preprocessor_override")
    ADAPTER_ID_FIELD_NUMBER: _ClassVar[int]
    SCALE_FIELD_NUMBER: _ClassVar[int]
    CONDITIONING_IMAGE_KEY_FIELD_NUMBER: _ClassVar[int]
    PREPROCESSOR_OVERRIDE_FIELD_NUMBER: _ClassVar[int]
    adapter_id: str
    scale: float
    conditioning_image_key: str
    preprocessor_override: str
    def __init__(self, adapter_id: _Optional[str] = ..., scale: _Optional[float] = ..., conditioning_image_key: _Optional[str] = ..., preprocessor_override: _Optional[str] = ...) -> None: ...

class ResolvedAdapter(_message.Message):
    __slots__ = ("id", "name", "kind", "architecture", "scale", "weight", "preprocessor", "conditioning_image_key")
    ID_FIELD_NUMBER: _ClassVar[int]
    NAME_FIELD_NUMBER: _ClassVar[int]
    KIND_FIELD_NUMBER: _ClassVar[int]
    ARCHITECTURE_FIELD_NUMBER: _ClassVar[int]
    SCALE_FIELD_NUMBER: _ClassVar[int]
    WEIGHT_FIELD_NUMBER: _ClassVar[int]
    PREPROCESSOR_FIELD_NUMBER: _ClassVar[int]
    CONDITIONING_IMAGE_KEY_FIELD_NUMBER: _ClassVar[int]
    id: str
    name: str
    kind: str
    architecture: str
    scale: float
    weight: str
    preprocessor: str
    conditioning_image_key: str
    def __init__(self, id: _Optional[str] = ..., name: _Optional[str] = ..., kind: _Optional[str] = ..., architecture: _Optional[str] = ..., scale: _Optional[float] = ..., weight: _Optional[str] = ..., preprocessor: _Optional[str] = ..., conditioning_image_key: _Optional[str] = ...) -> None: ...

class ExplainResolutionRequest(_message.Message):
    __slots__ = ("operation", "model_id", "allow_byok", "adapters", "quality_policy")
    OPERATION_FIELD_NUMBER: _ClassVar[int]
    MODEL_ID_FIELD_NUMBER: _ClassVar[int]
    ALLOW_BYOK_FIELD_NUMBER: _ClassVar[int]
    ADAPTERS_FIELD_NUMBER: _ClassVar[int]
    QUALITY_POLICY_FIELD_NUMBER: _ClassVar[int]
    operation: str
    model_id: str
    allow_byok: bool
    adapters: _containers.RepeatedCompositeFieldContainer[AdapterRef]
    quality_policy: str
    def __init__(self, operation: _Optional[str] = ..., model_id: _Optional[str] = ..., allow_byok: _Optional[bool] = ..., adapters: _Optional[_Iterable[_Union[AdapterRef, _Mapping]]] = ..., quality_policy: _Optional[str] = ...) -> None: ...

class ExplainResolutionResponse(_message.Message):
    __slots__ = ("resolution",)
    RESOLUTION_FIELD_NUMBER: _ClassVar[int]
    resolution: Resolution
    def __init__(self, resolution: _Optional[_Union[Resolution, _Mapping]] = ...) -> None: ...

class HostSummary(_message.Message):
    __slots__ = ("has_gpu", "gpu_name", "gpu_count", "vram_total_gb", "vram_free_gb", "vram_known", "ram_gb", "cpu_cores", "os", "arch")
    HAS_GPU_FIELD_NUMBER: _ClassVar[int]
    GPU_NAME_FIELD_NUMBER: _ClassVar[int]
    GPU_COUNT_FIELD_NUMBER: _ClassVar[int]
    VRAM_TOTAL_GB_FIELD_NUMBER: _ClassVar[int]
    VRAM_FREE_GB_FIELD_NUMBER: _ClassVar[int]
    VRAM_KNOWN_FIELD_NUMBER: _ClassVar[int]
    RAM_GB_FIELD_NUMBER: _ClassVar[int]
    CPU_CORES_FIELD_NUMBER: _ClassVar[int]
    OS_FIELD_NUMBER: _ClassVar[int]
    ARCH_FIELD_NUMBER: _ClassVar[int]
    has_gpu: bool
    gpu_name: str
    gpu_count: int
    vram_total_gb: int
    vram_free_gb: int
    vram_known: bool
    ram_gb: int
    cpu_cores: int
    os: str
    arch: str
    def __init__(self, has_gpu: _Optional[bool] = ..., gpu_name: _Optional[str] = ..., gpu_count: _Optional[int] = ..., vram_total_gb: _Optional[int] = ..., vram_free_gb: _Optional[int] = ..., vram_known: _Optional[bool] = ..., ram_gb: _Optional[int] = ..., cpu_cores: _Optional[int] = ..., os: _Optional[str] = ..., arch: _Optional[str] = ...) -> None: ...

class ModelFit(_message.Message):
    __slots__ = ("runnable", "gpu_viable", "fit_class", "vram_shortfall_gb", "warnings")
    RUNNABLE_FIELD_NUMBER: _ClassVar[int]
    GPU_VIABLE_FIELD_NUMBER: _ClassVar[int]
    FIT_CLASS_FIELD_NUMBER: _ClassVar[int]
    VRAM_SHORTFALL_GB_FIELD_NUMBER: _ClassVar[int]
    WARNINGS_FIELD_NUMBER: _ClassVar[int]
    runnable: bool
    gpu_viable: bool
    fit_class: str
    vram_shortfall_gb: int
    warnings: _containers.RepeatedScalarFieldContainer[str]
    def __init__(self, runnable: _Optional[bool] = ..., gpu_viable: _Optional[bool] = ..., fit_class: _Optional[str] = ..., vram_shortfall_gb: _Optional[int] = ..., warnings: _Optional[_Iterable[str]] = ...) -> None: ...

class BackendReadiness(_message.Message):
    __slots__ = ("backend", "host_tool", "ready", "install_tier", "remediation", "manual_hint", "detail")
    BACKEND_FIELD_NUMBER: _ClassVar[int]
    HOST_TOOL_FIELD_NUMBER: _ClassVar[int]
    READY_FIELD_NUMBER: _ClassVar[int]
    INSTALL_TIER_FIELD_NUMBER: _ClassVar[int]
    REMEDIATION_FIELD_NUMBER: _ClassVar[int]
    MANUAL_HINT_FIELD_NUMBER: _ClassVar[int]
    DETAIL_FIELD_NUMBER: _ClassVar[int]
    backend: str
    host_tool: str
    ready: bool
    install_tier: str
    remediation: str
    manual_hint: str
    detail: str
    def __init__(self, backend: _Optional[str] = ..., host_tool: _Optional[str] = ..., ready: _Optional[bool] = ..., install_tier: _Optional[str] = ..., remediation: _Optional[str] = ..., manual_hint: _Optional[str] = ..., detail: _Optional[str] = ...) -> None: ...

class CandidateModel(_message.Message):
    __slots__ = ("model", "fit", "backend", "ready_state", "selected", "support", "technique", "caveat")
    MODEL_FIELD_NUMBER: _ClassVar[int]
    FIT_FIELD_NUMBER: _ClassVar[int]
    BACKEND_FIELD_NUMBER: _ClassVar[int]
    READY_STATE_FIELD_NUMBER: _ClassVar[int]
    SELECTED_FIELD_NUMBER: _ClassVar[int]
    SUPPORT_FIELD_NUMBER: _ClassVar[int]
    TECHNIQUE_FIELD_NUMBER: _ClassVar[int]
    CAVEAT_FIELD_NUMBER: _ClassVar[int]
    model: Model
    fit: ModelFit
    backend: BackendReadiness
    ready_state: str
    selected: bool
    support: str
    technique: str
    caveat: str
    def __init__(self, model: _Optional[_Union[Model, _Mapping]] = ..., fit: _Optional[_Union[ModelFit, _Mapping]] = ..., backend: _Optional[_Union[BackendReadiness, _Mapping]] = ..., ready_state: _Optional[str] = ..., selected: _Optional[bool] = ..., support: _Optional[str] = ..., technique: _Optional[str] = ..., caveat: _Optional[str] = ...) -> None: ...

class ListOperationModelsResponse(_message.Message):
    __slots__ = ("operation", "host", "candidates", "selected_id", "selected_reason")
    OPERATION_FIELD_NUMBER: _ClassVar[int]
    HOST_FIELD_NUMBER: _ClassVar[int]
    CANDIDATES_FIELD_NUMBER: _ClassVar[int]
    SELECTED_ID_FIELD_NUMBER: _ClassVar[int]
    SELECTED_REASON_FIELD_NUMBER: _ClassVar[int]
    operation: str
    host: HostSummary
    candidates: _containers.RepeatedCompositeFieldContainer[CandidateModel]
    selected_id: str
    selected_reason: str
    def __init__(self, operation: _Optional[str] = ..., host: _Optional[_Union[HostSummary, _Mapping]] = ..., candidates: _Optional[_Iterable[_Union[CandidateModel, _Mapping]]] = ..., selected_id: _Optional[str] = ..., selected_reason: _Optional[str] = ...) -> None: ...
