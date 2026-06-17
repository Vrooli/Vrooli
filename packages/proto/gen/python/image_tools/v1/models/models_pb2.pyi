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
COMMERCIAL_USE_UNSPECIFIED: CommercialUse
COMMERCIAL_USE_YES: CommercialUse
COMMERCIAL_USE_NO: CommercialUse
COMMERCIAL_USE_CONDITIONAL: CommercialUse

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
    __slots__ = ("id", "name", "operations", "default_for", "tier", "backend", "alt_backends", "requires_comfyui", "size_mb_approx", "quant_variants", "hardware", "capability_labels", "enabled")
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
    def __init__(self, id: _Optional[str] = ..., name: _Optional[str] = ..., operations: _Optional[_Iterable[str]] = ..., default_for: _Optional[_Iterable[str]] = ..., tier: _Optional[str] = ..., backend: _Optional[str] = ..., alt_backends: _Optional[_Iterable[str]] = ..., requires_comfyui: _Optional[bool] = ..., size_mb_approx: _Optional[int] = ..., quant_variants: _Optional[_Iterable[str]] = ..., hardware: _Optional[_Union[Hardware, _Mapping]] = ..., capability_labels: _Optional[_Union[CapabilityLabels, _Mapping]] = ..., enabled: _Optional[bool] = ...) -> None: ...

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
