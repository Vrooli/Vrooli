import datetime

from common.v1 import attestation_pb2 as _attestation_pb2
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

class ArchetypeSource(int, metaclass=_enum_type_wrapper.EnumTypeWrapper):
    __slots__ = ()
    ARCHETYPE_SOURCE_UNSPECIFIED: _ClassVar[ArchetypeSource]
    ARCHETYPE_SOURCE_DECLARED: _ClassVar[ArchetypeSource]
    ARCHETYPE_SOURCE_INFERRED: _ClassVar[ArchetypeSource]

class Archetype(int, metaclass=_enum_type_wrapper.EnumTypeWrapper):
    __slots__ = ()
    ARCHETYPE_UNSPECIFIED: _ClassVar[Archetype]
    ARCHETYPE_REPORTING: _ClassVar[Archetype]
    ARCHETYPE_SERVICE: _ClassVar[Archetype]
    ARCHETYPE_MUTATION: _ClassVar[Archetype]
    ARCHETYPE_CLASSIFICATION: _ClassVar[Archetype]
    ARCHETYPE_ORCHESTRATION: _ClassVar[Archetype]
    ARCHETYPE_SCORING: _ClassVar[Archetype]
    ARCHETYPE_QUERY: _ClassVar[Archetype]

class AuthorityConfidence(int, metaclass=_enum_type_wrapper.EnumTypeWrapper):
    __slots__ = ()
    AUTHORITY_CONFIDENCE_UNSPECIFIED: _ClassVar[AuthorityConfidence]
    AUTHORITY_CONFIDENCE_HIGH: _ClassVar[AuthorityConfidence]
    AUTHORITY_CONFIDENCE_LOW: _ClassVar[AuthorityConfidence]

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
ARCHETYPE_SOURCE_UNSPECIFIED: ArchetypeSource
ARCHETYPE_SOURCE_DECLARED: ArchetypeSource
ARCHETYPE_SOURCE_INFERRED: ArchetypeSource
ARCHETYPE_UNSPECIFIED: Archetype
ARCHETYPE_REPORTING: Archetype
ARCHETYPE_SERVICE: Archetype
ARCHETYPE_MUTATION: Archetype
ARCHETYPE_CLASSIFICATION: Archetype
ARCHETYPE_ORCHESTRATION: Archetype
ARCHETYPE_SCORING: Archetype
ARCHETYPE_QUERY: Archetype
AUTHORITY_CONFIDENCE_UNSPECIFIED: AuthorityConfidence
AUTHORITY_CONFIDENCE_HIGH: AuthorityConfidence
AUTHORITY_CONFIDENCE_LOW: AuthorityConfidence
CONVERGENCE_SEVERITY_UNSPECIFIED: ConvergenceSeverity
CONVERGENCE_SEVERITY_INFO: ConvergenceSeverity
CONVERGENCE_SEVERITY_WARN: ConvergenceSeverity

class DerivedDomain(_message.Message):
    __slots__ = ("name", "paths", "glossary", "responsibility", "purpose", "owns_data", "secondary_traits", "surfaces", "archetypes", "provenance", "attestation")
    NAME_FIELD_NUMBER: _ClassVar[int]
    PATHS_FIELD_NUMBER: _ClassVar[int]
    GLOSSARY_FIELD_NUMBER: _ClassVar[int]
    RESPONSIBILITY_FIELD_NUMBER: _ClassVar[int]
    PURPOSE_FIELD_NUMBER: _ClassVar[int]
    OWNS_DATA_FIELD_NUMBER: _ClassVar[int]
    SECONDARY_TRAITS_FIELD_NUMBER: _ClassVar[int]
    SURFACES_FIELD_NUMBER: _ClassVar[int]
    ARCHETYPES_FIELD_NUMBER: _ClassVar[int]
    PROVENANCE_FIELD_NUMBER: _ClassVar[int]
    ATTESTATION_FIELD_NUMBER: _ClassVar[int]
    name: str
    paths: _containers.RepeatedScalarFieldContainer[str]
    glossary: _containers.RepeatedScalarFieldContainer[str]
    responsibility: str
    purpose: str
    owns_data: str
    secondary_traits: _containers.RepeatedScalarFieldContainer[str]
    surfaces: _containers.RepeatedScalarFieldContainer[str]
    archetypes: _containers.RepeatedCompositeFieldContainer[DomainArchetype]
    provenance: _containers.RepeatedScalarFieldContainer[DomainSource]
    attestation: _attestation_pb2.AttestedAnswer
    def __init__(self, name: _Optional[str] = ..., paths: _Optional[_Iterable[str]] = ..., glossary: _Optional[_Iterable[str]] = ..., responsibility: _Optional[str] = ..., purpose: _Optional[str] = ..., owns_data: _Optional[str] = ..., secondary_traits: _Optional[_Iterable[str]] = ..., surfaces: _Optional[_Iterable[str]] = ..., archetypes: _Optional[_Iterable[_Union[DomainArchetype, _Mapping]]] = ..., provenance: _Optional[_Iterable[_Union[DomainSource, str]]] = ..., attestation: _Optional[_Union[_attestation_pb2.AttestedAnswer, _Mapping]] = ...) -> None: ...

class DomainArchetype(_message.Message):
    __slots__ = ("archetype", "source", "confidence", "evidence", "declared_label")
    ARCHETYPE_FIELD_NUMBER: _ClassVar[int]
    SOURCE_FIELD_NUMBER: _ClassVar[int]
    CONFIDENCE_FIELD_NUMBER: _ClassVar[int]
    EVIDENCE_FIELD_NUMBER: _ClassVar[int]
    DECLARED_LABEL_FIELD_NUMBER: _ClassVar[int]
    archetype: Archetype
    source: ArchetypeSource
    confidence: float
    evidence: _containers.RepeatedScalarFieldContainer[str]
    declared_label: str
    def __init__(self, archetype: _Optional[_Union[Archetype, str]] = ..., source: _Optional[_Union[ArchetypeSource, str]] = ..., confidence: _Optional[float] = ..., evidence: _Optional[_Iterable[str]] = ..., declared_label: _Optional[str] = ...) -> None: ...

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
    __slots__ = ("scenario", "domains", "shared_substrate", "non_domains", "authority", "declarations", "derived_at", "authority_confidence", "attestation")
    SCENARIO_FIELD_NUMBER: _ClassVar[int]
    DOMAINS_FIELD_NUMBER: _ClassVar[int]
    SHARED_SUBSTRATE_FIELD_NUMBER: _ClassVar[int]
    NON_DOMAINS_FIELD_NUMBER: _ClassVar[int]
    AUTHORITY_FIELD_NUMBER: _ClassVar[int]
    DECLARATIONS_FIELD_NUMBER: _ClassVar[int]
    DERIVED_AT_FIELD_NUMBER: _ClassVar[int]
    AUTHORITY_CONFIDENCE_FIELD_NUMBER: _ClassVar[int]
    ATTESTATION_FIELD_NUMBER: _ClassVar[int]
    scenario: str
    domains: _containers.RepeatedCompositeFieldContainer[DerivedDomain]
    shared_substrate: _containers.RepeatedScalarFieldContainer[str]
    non_domains: _containers.RepeatedScalarFieldContainer[str]
    authority: DomainSource
    declarations: _containers.RepeatedCompositeFieldContainer[DomainDeclaration]
    derived_at: _timestamp_pb2.Timestamp
    authority_confidence: AuthorityConfidence
    attestation: _attestation_pb2.AttestedAnswer
    def __init__(self, scenario: _Optional[str] = ..., domains: _Optional[_Iterable[_Union[DerivedDomain, _Mapping]]] = ..., shared_substrate: _Optional[_Iterable[str]] = ..., non_domains: _Optional[_Iterable[str]] = ..., authority: _Optional[_Union[DomainSource, str]] = ..., declarations: _Optional[_Iterable[_Union[DomainDeclaration, _Mapping]]] = ..., derived_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., authority_confidence: _Optional[_Union[AuthorityConfidence, str]] = ..., attestation: _Optional[_Union[_attestation_pb2.AttestedAnswer, _Mapping]] = ...) -> None: ...

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
    __slots__ = ("kind", "domain", "severity", "message", "sources", "rolled_up_domains")
    KIND_FIELD_NUMBER: _ClassVar[int]
    DOMAIN_FIELD_NUMBER: _ClassVar[int]
    SEVERITY_FIELD_NUMBER: _ClassVar[int]
    MESSAGE_FIELD_NUMBER: _ClassVar[int]
    SOURCES_FIELD_NUMBER: _ClassVar[int]
    ROLLED_UP_DOMAINS_FIELD_NUMBER: _ClassVar[int]
    kind: str
    domain: str
    severity: ConvergenceSeverity
    message: str
    sources: _containers.RepeatedScalarFieldContainer[DomainSource]
    rolled_up_domains: _containers.RepeatedScalarFieldContainer[str]
    def __init__(self, kind: _Optional[str] = ..., domain: _Optional[str] = ..., severity: _Optional[_Union[ConvergenceSeverity, str]] = ..., message: _Optional[str] = ..., sources: _Optional[_Iterable[_Union[DomainSource, str]]] = ..., rolled_up_domains: _Optional[_Iterable[str]] = ...) -> None: ...

class ConvergenceReportRequest(_message.Message):
    __slots__ = ("scenario",)
    SCENARIO_FIELD_NUMBER: _ClassVar[int]
    scenario: str
    def __init__(self, scenario: _Optional[str] = ...) -> None: ...

class ConvergenceReportResponse(_message.Message):
    __slots__ = ("scenario", "authority", "findings", "authority_confidence")
    SCENARIO_FIELD_NUMBER: _ClassVar[int]
    AUTHORITY_FIELD_NUMBER: _ClassVar[int]
    FINDINGS_FIELD_NUMBER: _ClassVar[int]
    AUTHORITY_CONFIDENCE_FIELD_NUMBER: _ClassVar[int]
    scenario: str
    authority: DomainSource
    findings: _containers.RepeatedCompositeFieldContainer[ConvergenceFinding]
    authority_confidence: AuthorityConfidence
    def __init__(self, scenario: _Optional[str] = ..., authority: _Optional[_Union[DomainSource, str]] = ..., findings: _Optional[_Iterable[_Union[ConvergenceFinding, _Mapping]]] = ..., authority_confidence: _Optional[_Union[AuthorityConfidence, str]] = ...) -> None: ...

class DraftDomainsRequest(_message.Message):
    __slots__ = ("scenario",)
    SCENARIO_FIELD_NUMBER: _ClassVar[int]
    scenario: str
    def __init__(self, scenario: _Optional[str] = ...) -> None: ...

class ProposedDomain(_message.Message):
    __slots__ = ("name", "paths", "archetype", "glossary", "confidence", "evidence")
    NAME_FIELD_NUMBER: _ClassVar[int]
    PATHS_FIELD_NUMBER: _ClassVar[int]
    ARCHETYPE_FIELD_NUMBER: _ClassVar[int]
    GLOSSARY_FIELD_NUMBER: _ClassVar[int]
    CONFIDENCE_FIELD_NUMBER: _ClassVar[int]
    EVIDENCE_FIELD_NUMBER: _ClassVar[int]
    name: str
    paths: _containers.RepeatedScalarFieldContainer[str]
    archetype: str
    glossary: _containers.RepeatedScalarFieldContainer[str]
    confidence: str
    evidence: _containers.RepeatedScalarFieldContainer[str]
    def __init__(self, name: _Optional[str] = ..., paths: _Optional[_Iterable[str]] = ..., archetype: _Optional[str] = ..., glossary: _Optional[_Iterable[str]] = ..., confidence: _Optional[str] = ..., evidence: _Optional[_Iterable[str]] = ...) -> None: ...

class DraftDomainsResponse(_message.Message):
    __slots__ = ("scenario", "domains", "markdown")
    SCENARIO_FIELD_NUMBER: _ClassVar[int]
    DOMAINS_FIELD_NUMBER: _ClassVar[int]
    MARKDOWN_FIELD_NUMBER: _ClassVar[int]
    scenario: str
    domains: _containers.RepeatedCompositeFieldContainer[ProposedDomain]
    markdown: str
    def __init__(self, scenario: _Optional[str] = ..., domains: _Optional[_Iterable[_Union[ProposedDomain, _Mapping]]] = ..., markdown: _Optional[str] = ...) -> None: ...
