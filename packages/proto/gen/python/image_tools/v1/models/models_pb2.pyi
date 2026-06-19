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

class CatalogFindingSeverity(int, metaclass=_enum_type_wrapper.EnumTypeWrapper):
    __slots__ = ()
    CATALOG_FINDING_SEVERITY_UNSPECIFIED: _ClassVar[CatalogFindingSeverity]
    CATALOG_FINDING_SEVERITY_ERROR: _ClassVar[CatalogFindingSeverity]
    CATALOG_FINDING_SEVERITY_WARNING: _ClassVar[CatalogFindingSeverity]
COMMERCIAL_USE_UNSPECIFIED: CommercialUse
COMMERCIAL_USE_YES: CommercialUse
COMMERCIAL_USE_NO: CommercialUse
COMMERCIAL_USE_CONDITIONAL: CommercialUse
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
    __slots__ = ("id", "name", "operations", "default_for", "tier", "backend", "alt_backends", "requires_comfyui", "size_mb_approx", "quant_variants", "hardware", "capability_labels", "enabled", "install", "custom")
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
    def __init__(self, id: _Optional[str] = ..., name: _Optional[str] = ..., operations: _Optional[_Iterable[str]] = ..., default_for: _Optional[_Iterable[str]] = ..., tier: _Optional[str] = ..., backend: _Optional[str] = ..., alt_backends: _Optional[_Iterable[str]] = ..., requires_comfyui: _Optional[bool] = ..., size_mb_approx: _Optional[int] = ..., quant_variants: _Optional[_Iterable[str]] = ..., hardware: _Optional[_Union[Hardware, _Mapping]] = ..., capability_labels: _Optional[_Union[CapabilityLabels, _Mapping]] = ..., enabled: _Optional[bool] = ..., install: _Optional[_Union[InstallState, _Mapping]] = ..., custom: _Optional[bool] = ...) -> None: ...

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
    __slots__ = ("operation", "override_id")
    OPERATION_FIELD_NUMBER: _ClassVar[int]
    OVERRIDE_ID_FIELD_NUMBER: _ClassVar[int]
    operation: str
    override_id: str
    def __init__(self, operation: _Optional[str] = ..., override_id: _Optional[str] = ...) -> None: ...

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
