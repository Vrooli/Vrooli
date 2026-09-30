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

class CapabilityLabels(_message.Message):
    __slots__ = ("nsfw_capable", "license", "commercial_use", "commercial_use_notes", "base_model_lineage", "known_risks", "provenance")
    NSFW_CAPABLE_FIELD_NUMBER: _ClassVar[int]
    LICENSE_FIELD_NUMBER: _ClassVar[int]
    COMMERCIAL_USE_FIELD_NUMBER: _ClassVar[int]
    COMMERCIAL_USE_NOTES_FIELD_NUMBER: _ClassVar[int]
    BASE_MODEL_LINEAGE_FIELD_NUMBER: _ClassVar[int]
    KNOWN_RISKS_FIELD_NUMBER: _ClassVar[int]
    PROVENANCE_FIELD_NUMBER: _ClassVar[int]
    nsfw_capable: bool
    license: str
    commercial_use: CommercialUse
    commercial_use_notes: str
    base_model_lineage: str
    known_risks: str
    provenance: str
    def __init__(self, nsfw_capable: _Optional[bool] = ..., license: _Optional[str] = ..., commercial_use: _Optional[_Union[CommercialUse, str]] = ..., commercial_use_notes: _Optional[str] = ..., base_model_lineage: _Optional[str] = ..., known_risks: _Optional[str] = ..., provenance: _Optional[str] = ...) -> None: ...

class ScaleRange(_message.Message):
    __slots__ = ("min", "max", "default")
    MIN_FIELD_NUMBER: _ClassVar[int]
    MAX_FIELD_NUMBER: _ClassVar[int]
    DEFAULT_FIELD_NUMBER: _ClassVar[int]
    min: float
    max: float
    default: float
    def __init__(self, min: _Optional[float] = ..., max: _Optional[float] = ..., default: _Optional[float] = ...) -> None: ...

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

class Adapter(_message.Message):
    __slots__ = ("id", "name", "kind", "architecture", "weight", "preprocessor", "scale_range", "size_mb_approx", "capability_labels", "ready", "pending", "enabled", "install", "custom")
    ID_FIELD_NUMBER: _ClassVar[int]
    NAME_FIELD_NUMBER: _ClassVar[int]
    KIND_FIELD_NUMBER: _ClassVar[int]
    ARCHITECTURE_FIELD_NUMBER: _ClassVar[int]
    WEIGHT_FIELD_NUMBER: _ClassVar[int]
    PREPROCESSOR_FIELD_NUMBER: _ClassVar[int]
    SCALE_RANGE_FIELD_NUMBER: _ClassVar[int]
    SIZE_MB_APPROX_FIELD_NUMBER: _ClassVar[int]
    CAPABILITY_LABELS_FIELD_NUMBER: _ClassVar[int]
    READY_FIELD_NUMBER: _ClassVar[int]
    PENDING_FIELD_NUMBER: _ClassVar[int]
    ENABLED_FIELD_NUMBER: _ClassVar[int]
    INSTALL_FIELD_NUMBER: _ClassVar[int]
    CUSTOM_FIELD_NUMBER: _ClassVar[int]
    id: str
    name: str
    kind: str
    architecture: str
    weight: str
    preprocessor: str
    scale_range: ScaleRange
    size_mb_approx: int
    capability_labels: CapabilityLabels
    ready: bool
    pending: str
    enabled: bool
    install: InstallState
    custom: bool
    def __init__(self, id: _Optional[str] = ..., name: _Optional[str] = ..., kind: _Optional[str] = ..., architecture: _Optional[str] = ..., weight: _Optional[str] = ..., preprocessor: _Optional[str] = ..., scale_range: _Optional[_Union[ScaleRange, _Mapping]] = ..., size_mb_approx: _Optional[int] = ..., capability_labels: _Optional[_Union[CapabilityLabels, _Mapping]] = ..., ready: _Optional[bool] = ..., pending: _Optional[str] = ..., enabled: _Optional[bool] = ..., install: _Optional[_Union[InstallState, _Mapping]] = ..., custom: _Optional[bool] = ...) -> None: ...

class ListAdaptersRequest(_message.Message):
    __slots__ = ("kind", "architecture")
    KIND_FIELD_NUMBER: _ClassVar[int]
    ARCHITECTURE_FIELD_NUMBER: _ClassVar[int]
    kind: str
    architecture: str
    def __init__(self, kind: _Optional[str] = ..., architecture: _Optional[str] = ...) -> None: ...

class ListAdaptersResponse(_message.Message):
    __slots__ = ("adapters",)
    ADAPTERS_FIELD_NUMBER: _ClassVar[int]
    adapters: _containers.RepeatedCompositeFieldContainer[Adapter]
    def __init__(self, adapters: _Optional[_Iterable[_Union[Adapter, _Mapping]]] = ...) -> None: ...

class GetAdapterRequest(_message.Message):
    __slots__ = ("id",)
    ID_FIELD_NUMBER: _ClassVar[int]
    id: str
    def __init__(self, id: _Optional[str] = ...) -> None: ...

class GetAdapterResponse(_message.Message):
    __slots__ = ("adapter",)
    ADAPTER_FIELD_NUMBER: _ClassVar[int]
    adapter: Adapter
    def __init__(self, adapter: _Optional[_Union[Adapter, _Mapping]] = ...) -> None: ...

class SetAdapterEnabledRequest(_message.Message):
    __slots__ = ("id", "enabled")
    ID_FIELD_NUMBER: _ClassVar[int]
    ENABLED_FIELD_NUMBER: _ClassVar[int]
    id: str
    enabled: bool
    def __init__(self, id: _Optional[str] = ..., enabled: _Optional[bool] = ...) -> None: ...

class SetAdapterEnabledResponse(_message.Message):
    __slots__ = ("adapter",)
    ADAPTER_FIELD_NUMBER: _ClassVar[int]
    adapter: Adapter
    def __init__(self, adapter: _Optional[_Union[Adapter, _Mapping]] = ...) -> None: ...

class InstallAdapterRequest(_message.Message):
    __slots__ = ("id",)
    ID_FIELD_NUMBER: _ClassVar[int]
    id: str
    def __init__(self, id: _Optional[str] = ...) -> None: ...

class InstallAdapterResponse(_message.Message):
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

class RemoveAdapterRequest(_message.Message):
    __slots__ = ("id",)
    ID_FIELD_NUMBER: _ClassVar[int]
    id: str
    def __init__(self, id: _Optional[str] = ...) -> None: ...

class RemoveAdapterResponse(_message.Message):
    __slots__ = ("removed",)
    REMOVED_FIELD_NUMBER: _ClassVar[int]
    removed: bool
    def __init__(self, removed: _Optional[bool] = ...) -> None: ...

class KindInference(_message.Message):
    __slots__ = ("kind", "evidence")
    KIND_FIELD_NUMBER: _ClassVar[int]
    EVIDENCE_FIELD_NUMBER: _ClassVar[int]
    kind: str
    evidence: str
    def __init__(self, kind: _Optional[str] = ..., evidence: _Optional[str] = ...) -> None: ...

class ArchitectureInference(_message.Message):
    __slots__ = ("architecture", "confidence", "evidence")
    ARCHITECTURE_FIELD_NUMBER: _ClassVar[int]
    CONFIDENCE_FIELD_NUMBER: _ClassVar[int]
    EVIDENCE_FIELD_NUMBER: _ClassVar[int]
    architecture: str
    confidence: str
    evidence: str
    def __init__(self, architecture: _Optional[str] = ..., confidence: _Optional[str] = ..., evidence: _Optional[str] = ...) -> None: ...

class InspectAdapterSourceRequest(_message.Message):
    __slots__ = ("source",)
    SOURCE_FIELD_NUMBER: _ClassVar[int]
    source: str
    def __init__(self, source: _Optional[str] = ...) -> None: ...

class InspectAdapterSourceResponse(_message.Message):
    __slots__ = ("source", "repo_id", "revision", "kind", "architecture", "license", "nsfw", "size_bytes", "proposed")
    SOURCE_FIELD_NUMBER: _ClassVar[int]
    REPO_ID_FIELD_NUMBER: _ClassVar[int]
    REVISION_FIELD_NUMBER: _ClassVar[int]
    KIND_FIELD_NUMBER: _ClassVar[int]
    ARCHITECTURE_FIELD_NUMBER: _ClassVar[int]
    LICENSE_FIELD_NUMBER: _ClassVar[int]
    NSFW_FIELD_NUMBER: _ClassVar[int]
    SIZE_BYTES_FIELD_NUMBER: _ClassVar[int]
    PROPOSED_FIELD_NUMBER: _ClassVar[int]
    source: str
    repo_id: str
    revision: str
    kind: KindInference
    architecture: ArchitectureInference
    license: str
    nsfw: bool
    size_bytes: int
    proposed: Adapter
    def __init__(self, source: _Optional[str] = ..., repo_id: _Optional[str] = ..., revision: _Optional[str] = ..., kind: _Optional[_Union[KindInference, _Mapping]] = ..., architecture: _Optional[_Union[ArchitectureInference, _Mapping]] = ..., license: _Optional[str] = ..., nsfw: _Optional[bool] = ..., size_bytes: _Optional[int] = ..., proposed: _Optional[_Union[Adapter, _Mapping]] = ...) -> None: ...

class ImportAdapterRequest(_message.Message):
    __slots__ = ("source", "id", "name", "kind", "architecture", "preprocessor", "attest_commercial_rights")
    SOURCE_FIELD_NUMBER: _ClassVar[int]
    ID_FIELD_NUMBER: _ClassVar[int]
    NAME_FIELD_NUMBER: _ClassVar[int]
    KIND_FIELD_NUMBER: _ClassVar[int]
    ARCHITECTURE_FIELD_NUMBER: _ClassVar[int]
    PREPROCESSOR_FIELD_NUMBER: _ClassVar[int]
    ATTEST_COMMERCIAL_RIGHTS_FIELD_NUMBER: _ClassVar[int]
    source: str
    id: str
    name: str
    kind: str
    architecture: str
    preprocessor: str
    attest_commercial_rights: bool
    def __init__(self, source: _Optional[str] = ..., id: _Optional[str] = ..., name: _Optional[str] = ..., kind: _Optional[str] = ..., architecture: _Optional[str] = ..., preprocessor: _Optional[str] = ..., attest_commercial_rights: _Optional[bool] = ...) -> None: ...

class ImportAdapterResponse(_message.Message):
    __slots__ = ("adapter", "job_id", "eta_seconds", "already_installed")
    ADAPTER_FIELD_NUMBER: _ClassVar[int]
    JOB_ID_FIELD_NUMBER: _ClassVar[int]
    ETA_SECONDS_FIELD_NUMBER: _ClassVar[int]
    ALREADY_INSTALLED_FIELD_NUMBER: _ClassVar[int]
    adapter: Adapter
    job_id: str
    eta_seconds: int
    already_installed: bool
    def __init__(self, adapter: _Optional[_Union[Adapter, _Mapping]] = ..., job_id: _Optional[str] = ..., eta_seconds: _Optional[int] = ..., already_installed: _Optional[bool] = ...) -> None: ...

class ListCompatibleAdaptersRequest(_message.Message):
    __slots__ = ("model_id", "architecture")
    MODEL_ID_FIELD_NUMBER: _ClassVar[int]
    ARCHITECTURE_FIELD_NUMBER: _ClassVar[int]
    model_id: str
    architecture: str
    def __init__(self, model_id: _Optional[str] = ..., architecture: _Optional[str] = ...) -> None: ...

class ListCompatibleAdaptersResponse(_message.Message):
    __slots__ = ("architecture", "adapters")
    ARCHITECTURE_FIELD_NUMBER: _ClassVar[int]
    ADAPTERS_FIELD_NUMBER: _ClassVar[int]
    architecture: str
    adapters: _containers.RepeatedCompositeFieldContainer[Adapter]
    def __init__(self, architecture: _Optional[str] = ..., adapters: _Optional[_Iterable[_Union[Adapter, _Mapping]]] = ...) -> None: ...

class CatalogFinding(_message.Message):
    __slots__ = ("severity", "code", "adapter_id", "message")
    SEVERITY_FIELD_NUMBER: _ClassVar[int]
    CODE_FIELD_NUMBER: _ClassVar[int]
    ADAPTER_ID_FIELD_NUMBER: _ClassVar[int]
    MESSAGE_FIELD_NUMBER: _ClassVar[int]
    severity: CatalogFindingSeverity
    code: str
    adapter_id: str
    message: str
    def __init__(self, severity: _Optional[_Union[CatalogFindingSeverity, str]] = ..., code: _Optional[str] = ..., adapter_id: _Optional[str] = ..., message: _Optional[str] = ...) -> None: ...

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
