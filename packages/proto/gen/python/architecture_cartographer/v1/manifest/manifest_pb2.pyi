import datetime

from google.protobuf import timestamp_pb2 as _timestamp_pb2
from google.protobuf.internal import containers as _containers
from google.protobuf.internal import enum_type_wrapper as _enum_type_wrapper
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Iterable as _Iterable, Mapping as _Mapping
from typing import ClassVar as _ClassVar, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class ManifestVersion(int, metaclass=_enum_type_wrapper.EnumTypeWrapper):
    __slots__ = ()
    MANIFEST_VERSION_UNSPECIFIED: _ClassVar[ManifestVersion]
    MANIFEST_VERSION_V1: _ClassVar[ManifestVersion]

class DiagnosticSeverity(int, metaclass=_enum_type_wrapper.EnumTypeWrapper):
    __slots__ = ()
    DIAGNOSTIC_SEVERITY_UNSPECIFIED: _ClassVar[DiagnosticSeverity]
    DIAGNOSTIC_SEVERITY_INFO: _ClassVar[DiagnosticSeverity]
    DIAGNOSTIC_SEVERITY_WARN: _ClassVar[DiagnosticSeverity]
    DIAGNOSTIC_SEVERITY_ERROR: _ClassVar[DiagnosticSeverity]
MANIFEST_VERSION_UNSPECIFIED: ManifestVersion
MANIFEST_VERSION_V1: ManifestVersion
DIAGNOSTIC_SEVERITY_UNSPECIFIED: DiagnosticSeverity
DIAGNOSTIC_SEVERITY_INFO: DiagnosticSeverity
DIAGNOSTIC_SEVERITY_WARN: DiagnosticSeverity
DIAGNOSTIC_SEVERITY_ERROR: DiagnosticSeverity

class Threshold(_message.Message):
    __slots__ = ("tier", "min_value")
    TIER_FIELD_NUMBER: _ClassVar[int]
    MIN_VALUE_FIELD_NUMBER: _ClassVar[int]
    tier: str
    min_value: float
    def __init__(self, tier: _Optional[str] = ..., min_value: _Optional[float] = ...) -> None: ...

class SignalWeights(_message.Message):
    __slots__ = ("weights",)
    class WeightsEntry(_message.Message):
        __slots__ = ("key", "value")
        KEY_FIELD_NUMBER: _ClassVar[int]
        VALUE_FIELD_NUMBER: _ClassVar[int]
        key: str
        value: float
        def __init__(self, key: _Optional[str] = ..., value: _Optional[float] = ...) -> None: ...
    WEIGHTS_FIELD_NUMBER: _ClassVar[int]
    weights: _containers.ScalarMap[str, float]
    def __init__(self, weights: _Optional[_Mapping[str, float]] = ...) -> None: ...

class TransitionalDeclaration(_message.Message):
    __slots__ = ("id", "kind", "locator", "rationale", "expires_when")
    ID_FIELD_NUMBER: _ClassVar[int]
    KIND_FIELD_NUMBER: _ClassVar[int]
    LOCATOR_FIELD_NUMBER: _ClassVar[int]
    RATIONALE_FIELD_NUMBER: _ClassVar[int]
    EXPIRES_WHEN_FIELD_NUMBER: _ClassVar[int]
    id: str
    kind: str
    locator: str
    rationale: str
    expires_when: str
    def __init__(self, id: _Optional[str] = ..., kind: _Optional[str] = ..., locator: _Optional[str] = ..., rationale: _Optional[str] = ..., expires_when: _Optional[str] = ...) -> None: ...

class DomainSpec(_message.Message):
    __slots__ = ("name", "paths", "allowed_dependencies", "glossary", "signal_weight_overrides")
    NAME_FIELD_NUMBER: _ClassVar[int]
    PATHS_FIELD_NUMBER: _ClassVar[int]
    ALLOWED_DEPENDENCIES_FIELD_NUMBER: _ClassVar[int]
    GLOSSARY_FIELD_NUMBER: _ClassVar[int]
    SIGNAL_WEIGHT_OVERRIDES_FIELD_NUMBER: _ClassVar[int]
    name: str
    paths: _containers.RepeatedScalarFieldContainer[str]
    allowed_dependencies: _containers.RepeatedScalarFieldContainer[str]
    glossary: _containers.RepeatedScalarFieldContainer[str]
    signal_weight_overrides: SignalWeights
    def __init__(self, name: _Optional[str] = ..., paths: _Optional[_Iterable[str]] = ..., allowed_dependencies: _Optional[_Iterable[str]] = ..., glossary: _Optional[_Iterable[str]] = ..., signal_weight_overrides: _Optional[_Union[SignalWeights, _Mapping]] = ...) -> None: ...

class ManifestDefinition(_message.Message):
    __slots__ = ("manifest_version", "scenario", "domains", "shared_substrate", "signal_weights", "thresholds", "transitional", "parsed_at", "content_hash")
    MANIFEST_VERSION_FIELD_NUMBER: _ClassVar[int]
    SCENARIO_FIELD_NUMBER: _ClassVar[int]
    DOMAINS_FIELD_NUMBER: _ClassVar[int]
    SHARED_SUBSTRATE_FIELD_NUMBER: _ClassVar[int]
    SIGNAL_WEIGHTS_FIELD_NUMBER: _ClassVar[int]
    THRESHOLDS_FIELD_NUMBER: _ClassVar[int]
    TRANSITIONAL_FIELD_NUMBER: _ClassVar[int]
    PARSED_AT_FIELD_NUMBER: _ClassVar[int]
    CONTENT_HASH_FIELD_NUMBER: _ClassVar[int]
    manifest_version: ManifestVersion
    scenario: str
    domains: _containers.RepeatedCompositeFieldContainer[DomainSpec]
    shared_substrate: _containers.RepeatedScalarFieldContainer[str]
    signal_weights: SignalWeights
    thresholds: _containers.RepeatedCompositeFieldContainer[Threshold]
    transitional: _containers.RepeatedCompositeFieldContainer[TransitionalDeclaration]
    parsed_at: _timestamp_pb2.Timestamp
    content_hash: str
    def __init__(self, manifest_version: _Optional[_Union[ManifestVersion, str]] = ..., scenario: _Optional[str] = ..., domains: _Optional[_Iterable[_Union[DomainSpec, _Mapping]]] = ..., shared_substrate: _Optional[_Iterable[str]] = ..., signal_weights: _Optional[_Union[SignalWeights, _Mapping]] = ..., thresholds: _Optional[_Iterable[_Union[Threshold, _Mapping]]] = ..., transitional: _Optional[_Iterable[_Union[TransitionalDeclaration, _Mapping]]] = ..., parsed_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., content_hash: _Optional[str] = ...) -> None: ...

class Diagnostic(_message.Message):
    __slots__ = ("severity", "path", "line", "column", "message", "code")
    SEVERITY_FIELD_NUMBER: _ClassVar[int]
    PATH_FIELD_NUMBER: _ClassVar[int]
    LINE_FIELD_NUMBER: _ClassVar[int]
    COLUMN_FIELD_NUMBER: _ClassVar[int]
    MESSAGE_FIELD_NUMBER: _ClassVar[int]
    CODE_FIELD_NUMBER: _ClassVar[int]
    severity: DiagnosticSeverity
    path: str
    line: int
    column: int
    message: str
    code: str
    def __init__(self, severity: _Optional[_Union[DiagnosticSeverity, str]] = ..., path: _Optional[str] = ..., line: _Optional[int] = ..., column: _Optional[int] = ..., message: _Optional[str] = ..., code: _Optional[str] = ...) -> None: ...

class ValidateManifestRequest(_message.Message):
    __slots__ = ("scenario", "source", "content_type")
    SCENARIO_FIELD_NUMBER: _ClassVar[int]
    SOURCE_FIELD_NUMBER: _ClassVar[int]
    CONTENT_TYPE_FIELD_NUMBER: _ClassVar[int]
    scenario: str
    source: bytes
    content_type: str
    def __init__(self, scenario: _Optional[str] = ..., source: _Optional[bytes] = ..., content_type: _Optional[str] = ...) -> None: ...

class ValidateManifestResponse(_message.Message):
    __slots__ = ("manifest", "diagnostics", "valid")
    MANIFEST_FIELD_NUMBER: _ClassVar[int]
    DIAGNOSTICS_FIELD_NUMBER: _ClassVar[int]
    VALID_FIELD_NUMBER: _ClassVar[int]
    manifest: ManifestDefinition
    diagnostics: _containers.RepeatedCompositeFieldContainer[Diagnostic]
    valid: bool
    def __init__(self, manifest: _Optional[_Union[ManifestDefinition, _Mapping]] = ..., diagnostics: _Optional[_Iterable[_Union[Diagnostic, _Mapping]]] = ..., valid: _Optional[bool] = ...) -> None: ...

class GetManifestRequest(_message.Message):
    __slots__ = ("scenario",)
    SCENARIO_FIELD_NUMBER: _ClassVar[int]
    scenario: str
    def __init__(self, scenario: _Optional[str] = ...) -> None: ...

class GetManifestResponse(_message.Message):
    __slots__ = ("manifest",)
    MANIFEST_FIELD_NUMBER: _ClassVar[int]
    manifest: ManifestDefinition
    def __init__(self, manifest: _Optional[_Union[ManifestDefinition, _Mapping]] = ...) -> None: ...

class ListDomainsRequest(_message.Message):
    __slots__ = ("scenario",)
    SCENARIO_FIELD_NUMBER: _ClassVar[int]
    scenario: str
    def __init__(self, scenario: _Optional[str] = ...) -> None: ...

class ListDomainsResponse(_message.Message):
    __slots__ = ("domains",)
    DOMAINS_FIELD_NUMBER: _ClassVar[int]
    domains: _containers.RepeatedCompositeFieldContainer[DomainSpec]
    def __init__(self, domains: _Optional[_Iterable[_Union[DomainSpec, _Mapping]]] = ...) -> None: ...
