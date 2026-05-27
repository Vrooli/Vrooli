import datetime

from google.protobuf import timestamp_pb2 as _timestamp_pb2
from google.protobuf.internal import containers as _containers
from google.protobuf.internal import enum_type_wrapper as _enum_type_wrapper
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Iterable as _Iterable, Mapping as _Mapping
from typing import ClassVar as _ClassVar, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class DomainSource(int, metaclass=_enum_type_wrapper.EnumTypeWrapper):
    __slots__ = ()
    DOMAIN_SOURCE_UNSPECIFIED: _ClassVar[DomainSource]
    DOMAIN_SOURCE_API_MANIFEST: _ClassVar[DomainSource]
    DOMAIN_SOURCE_DOMAINS_DOC: _ClassVar[DomainSource]
    DOMAIN_SOURCE_API_FOLDERS: _ClassVar[DomainSource]
    DOMAIN_SOURCE_CLI_GROUPS: _ClassVar[DomainSource]
    DOMAIN_SOURCE_UI_FEATURES: _ClassVar[DomainSource]

class ConvergenceSeverity(int, metaclass=_enum_type_wrapper.EnumTypeWrapper):
    __slots__ = ()
    CONVERGENCE_SEVERITY_UNSPECIFIED: _ClassVar[ConvergenceSeverity]
    CONVERGENCE_SEVERITY_INFO: _ClassVar[ConvergenceSeverity]
    CONVERGENCE_SEVERITY_WARN: _ClassVar[ConvergenceSeverity]
DOMAIN_SOURCE_UNSPECIFIED: DomainSource
DOMAIN_SOURCE_API_MANIFEST: DomainSource
DOMAIN_SOURCE_DOMAINS_DOC: DomainSource
DOMAIN_SOURCE_API_FOLDERS: DomainSource
DOMAIN_SOURCE_CLI_GROUPS: DomainSource
DOMAIN_SOURCE_UI_FEATURES: DomainSource
CONVERGENCE_SEVERITY_UNSPECIFIED: ConvergenceSeverity
CONVERGENCE_SEVERITY_INFO: ConvergenceSeverity
CONVERGENCE_SEVERITY_WARN: ConvergenceSeverity

class DerivedDomain(_message.Message):
    __slots__ = ("name", "paths", "glossary", "archetype", "provenance")
    NAME_FIELD_NUMBER: _ClassVar[int]
    PATHS_FIELD_NUMBER: _ClassVar[int]
    GLOSSARY_FIELD_NUMBER: _ClassVar[int]
    ARCHETYPE_FIELD_NUMBER: _ClassVar[int]
    PROVENANCE_FIELD_NUMBER: _ClassVar[int]
    name: str
    paths: _containers.RepeatedScalarFieldContainer[str]
    glossary: _containers.RepeatedScalarFieldContainer[str]
    archetype: str
    provenance: _containers.RepeatedScalarFieldContainer[DomainSource]
    def __init__(self, name: _Optional[str] = ..., paths: _Optional[_Iterable[str]] = ..., glossary: _Optional[_Iterable[str]] = ..., archetype: _Optional[str] = ..., provenance: _Optional[_Iterable[_Union[DomainSource, str]]] = ...) -> None: ...

class DomainDeclaration(_message.Message):
    __slots__ = ("source", "domain_names", "authoritative")
    SOURCE_FIELD_NUMBER: _ClassVar[int]
    DOMAIN_NAMES_FIELD_NUMBER: _ClassVar[int]
    AUTHORITATIVE_FIELD_NUMBER: _ClassVar[int]
    source: DomainSource
    domain_names: _containers.RepeatedScalarFieldContainer[str]
    authoritative: bool
    def __init__(self, source: _Optional[_Union[DomainSource, str]] = ..., domain_names: _Optional[_Iterable[str]] = ..., authoritative: _Optional[bool] = ...) -> None: ...

class DerivedDomainMap(_message.Message):
    __slots__ = ("scenario", "domains", "shared_substrate", "non_domains", "authority", "declarations", "derived_at")
    SCENARIO_FIELD_NUMBER: _ClassVar[int]
    DOMAINS_FIELD_NUMBER: _ClassVar[int]
    SHARED_SUBSTRATE_FIELD_NUMBER: _ClassVar[int]
    NON_DOMAINS_FIELD_NUMBER: _ClassVar[int]
    AUTHORITY_FIELD_NUMBER: _ClassVar[int]
    DECLARATIONS_FIELD_NUMBER: _ClassVar[int]
    DERIVED_AT_FIELD_NUMBER: _ClassVar[int]
    scenario: str
    domains: _containers.RepeatedCompositeFieldContainer[DerivedDomain]
    shared_substrate: _containers.RepeatedScalarFieldContainer[str]
    non_domains: _containers.RepeatedScalarFieldContainer[str]
    authority: DomainSource
    declarations: _containers.RepeatedCompositeFieldContainer[DomainDeclaration]
    derived_at: _timestamp_pb2.Timestamp
    def __init__(self, scenario: _Optional[str] = ..., domains: _Optional[_Iterable[_Union[DerivedDomain, _Mapping]]] = ..., shared_substrate: _Optional[_Iterable[str]] = ..., non_domains: _Optional[_Iterable[str]] = ..., authority: _Optional[_Union[DomainSource, str]] = ..., declarations: _Optional[_Iterable[_Union[DomainDeclaration, _Mapping]]] = ..., derived_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ...) -> None: ...

class ExtractDomainsRequest(_message.Message):
    __slots__ = ("scenario",)
    SCENARIO_FIELD_NUMBER: _ClassVar[int]
    scenario: str
    def __init__(self, scenario: _Optional[str] = ...) -> None: ...

class ExtractDomainsResponse(_message.Message):
    __slots__ = ("domain_map",)
    DOMAIN_MAP_FIELD_NUMBER: _ClassVar[int]
    domain_map: DerivedDomainMap
    def __init__(self, domain_map: _Optional[_Union[DerivedDomainMap, _Mapping]] = ...) -> None: ...

class GetDomainMapRequest(_message.Message):
    __slots__ = ("scenario",)
    SCENARIO_FIELD_NUMBER: _ClassVar[int]
    scenario: str
    def __init__(self, scenario: _Optional[str] = ...) -> None: ...

class GetDomainMapResponse(_message.Message):
    __slots__ = ("domain_map",)
    DOMAIN_MAP_FIELD_NUMBER: _ClassVar[int]
    domain_map: DerivedDomainMap
    def __init__(self, domain_map: _Optional[_Union[DerivedDomainMap, _Mapping]] = ...) -> None: ...

class ConvergenceFinding(_message.Message):
    __slots__ = ("kind", "domain", "severity", "message", "sources")
    KIND_FIELD_NUMBER: _ClassVar[int]
    DOMAIN_FIELD_NUMBER: _ClassVar[int]
    SEVERITY_FIELD_NUMBER: _ClassVar[int]
    MESSAGE_FIELD_NUMBER: _ClassVar[int]
    SOURCES_FIELD_NUMBER: _ClassVar[int]
    kind: str
    domain: str
    severity: ConvergenceSeverity
    message: str
    sources: _containers.RepeatedScalarFieldContainer[DomainSource]
    def __init__(self, kind: _Optional[str] = ..., domain: _Optional[str] = ..., severity: _Optional[_Union[ConvergenceSeverity, str]] = ..., message: _Optional[str] = ..., sources: _Optional[_Iterable[_Union[DomainSource, str]]] = ...) -> None: ...

class ConvergenceReportRequest(_message.Message):
    __slots__ = ("scenario",)
    SCENARIO_FIELD_NUMBER: _ClassVar[int]
    scenario: str
    def __init__(self, scenario: _Optional[str] = ...) -> None: ...

class ConvergenceReportResponse(_message.Message):
    __slots__ = ("scenario", "authority", "findings")
    SCENARIO_FIELD_NUMBER: _ClassVar[int]
    AUTHORITY_FIELD_NUMBER: _ClassVar[int]
    FINDINGS_FIELD_NUMBER: _ClassVar[int]
    scenario: str
    authority: DomainSource
    findings: _containers.RepeatedCompositeFieldContainer[ConvergenceFinding]
    def __init__(self, scenario: _Optional[str] = ..., authority: _Optional[_Union[DomainSource, str]] = ..., findings: _Optional[_Iterable[_Union[ConvergenceFinding, _Mapping]]] = ...) -> None: ...
