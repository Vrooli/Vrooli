from google.protobuf.internal import containers as _containers
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Iterable as _Iterable, Mapping as _Mapping
from typing import ClassVar as _ClassVar, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class Capability(_message.Message):
    __slots__ = ("id", "name", "medium", "aliases", "channels", "audience_applicability", "delivery_applicability", "producing_operation", "prerequisites", "priority", "priority_reason", "priority_scope", "definition_status", "implementation_status", "operational_readiness", "output_quality", "distribution_connectivity", "owner", "source_refs", "latest_qualification", "next_action", "created_at", "updated_at", "output_quality_source", "readiness_limitations", "operational_readiness_source", "distribution_connectivity_source")
    ID_FIELD_NUMBER: _ClassVar[int]
    NAME_FIELD_NUMBER: _ClassVar[int]
    MEDIUM_FIELD_NUMBER: _ClassVar[int]
    ALIASES_FIELD_NUMBER: _ClassVar[int]
    CHANNELS_FIELD_NUMBER: _ClassVar[int]
    AUDIENCE_APPLICABILITY_FIELD_NUMBER: _ClassVar[int]
    DELIVERY_APPLICABILITY_FIELD_NUMBER: _ClassVar[int]
    PRODUCING_OPERATION_FIELD_NUMBER: _ClassVar[int]
    PREREQUISITES_FIELD_NUMBER: _ClassVar[int]
    PRIORITY_FIELD_NUMBER: _ClassVar[int]
    PRIORITY_REASON_FIELD_NUMBER: _ClassVar[int]
    PRIORITY_SCOPE_FIELD_NUMBER: _ClassVar[int]
    DEFINITION_STATUS_FIELD_NUMBER: _ClassVar[int]
    IMPLEMENTATION_STATUS_FIELD_NUMBER: _ClassVar[int]
    OPERATIONAL_READINESS_FIELD_NUMBER: _ClassVar[int]
    OUTPUT_QUALITY_FIELD_NUMBER: _ClassVar[int]
    DISTRIBUTION_CONNECTIVITY_FIELD_NUMBER: _ClassVar[int]
    OWNER_FIELD_NUMBER: _ClassVar[int]
    SOURCE_REFS_FIELD_NUMBER: _ClassVar[int]
    LATEST_QUALIFICATION_FIELD_NUMBER: _ClassVar[int]
    NEXT_ACTION_FIELD_NUMBER: _ClassVar[int]
    CREATED_AT_FIELD_NUMBER: _ClassVar[int]
    UPDATED_AT_FIELD_NUMBER: _ClassVar[int]
    OUTPUT_QUALITY_SOURCE_FIELD_NUMBER: _ClassVar[int]
    READINESS_LIMITATIONS_FIELD_NUMBER: _ClassVar[int]
    OPERATIONAL_READINESS_SOURCE_FIELD_NUMBER: _ClassVar[int]
    DISTRIBUTION_CONNECTIVITY_SOURCE_FIELD_NUMBER: _ClassVar[int]
    id: str
    name: str
    medium: str
    aliases: _containers.RepeatedScalarFieldContainer[str]
    channels: _containers.RepeatedScalarFieldContainer[str]
    audience_applicability: str
    delivery_applicability: str
    producing_operation: str
    prerequisites: _containers.RepeatedScalarFieldContainer[str]
    priority: int
    priority_reason: str
    priority_scope: str
    definition_status: str
    implementation_status: str
    operational_readiness: str
    output_quality: str
    distribution_connectivity: str
    owner: str
    source_refs: _containers.RepeatedScalarFieldContainer[str]
    latest_qualification: CapabilityQualification
    next_action: str
    created_at: str
    updated_at: str
    output_quality_source: str
    readiness_limitations: _containers.RepeatedScalarFieldContainer[str]
    operational_readiness_source: str
    distribution_connectivity_source: str
    def __init__(self, id: _Optional[str] = ..., name: _Optional[str] = ..., medium: _Optional[str] = ..., aliases: _Optional[_Iterable[str]] = ..., channels: _Optional[_Iterable[str]] = ..., audience_applicability: _Optional[str] = ..., delivery_applicability: _Optional[str] = ..., producing_operation: _Optional[str] = ..., prerequisites: _Optional[_Iterable[str]] = ..., priority: _Optional[int] = ..., priority_reason: _Optional[str] = ..., priority_scope: _Optional[str] = ..., definition_status: _Optional[str] = ..., implementation_status: _Optional[str] = ..., operational_readiness: _Optional[str] = ..., output_quality: _Optional[str] = ..., distribution_connectivity: _Optional[str] = ..., owner: _Optional[str] = ..., source_refs: _Optional[_Iterable[str]] = ..., latest_qualification: _Optional[_Union[CapabilityQualification, _Mapping]] = ..., next_action: _Optional[str] = ..., created_at: _Optional[str] = ..., updated_at: _Optional[str] = ..., output_quality_source: _Optional[str] = ..., readiness_limitations: _Optional[_Iterable[str]] = ..., operational_readiness_source: _Optional[str] = ..., distribution_connectivity_source: _Optional[str] = ...) -> None: ...

class CapabilityQualification(_message.Message):
    __slots__ = ("id", "capability_id", "latest_artifact_id", "latest_run_id", "environment", "validated_at", "observed_at", "freshness_basis", "candidate_identity", "max_age_seconds", "limitation", "next_action", "created_at")
    ID_FIELD_NUMBER: _ClassVar[int]
    CAPABILITY_ID_FIELD_NUMBER: _ClassVar[int]
    LATEST_ARTIFACT_ID_FIELD_NUMBER: _ClassVar[int]
    LATEST_RUN_ID_FIELD_NUMBER: _ClassVar[int]
    ENVIRONMENT_FIELD_NUMBER: _ClassVar[int]
    VALIDATED_AT_FIELD_NUMBER: _ClassVar[int]
    OBSERVED_AT_FIELD_NUMBER: _ClassVar[int]
    FRESHNESS_BASIS_FIELD_NUMBER: _ClassVar[int]
    CANDIDATE_IDENTITY_FIELD_NUMBER: _ClassVar[int]
    MAX_AGE_SECONDS_FIELD_NUMBER: _ClassVar[int]
    LIMITATION_FIELD_NUMBER: _ClassVar[int]
    NEXT_ACTION_FIELD_NUMBER: _ClassVar[int]
    CREATED_AT_FIELD_NUMBER: _ClassVar[int]
    id: str
    capability_id: str
    latest_artifact_id: str
    latest_run_id: str
    environment: str
    validated_at: str
    observed_at: str
    freshness_basis: str
    candidate_identity: str
    max_age_seconds: int
    limitation: str
    next_action: str
    created_at: str
    def __init__(self, id: _Optional[str] = ..., capability_id: _Optional[str] = ..., latest_artifact_id: _Optional[str] = ..., latest_run_id: _Optional[str] = ..., environment: _Optional[str] = ..., validated_at: _Optional[str] = ..., observed_at: _Optional[str] = ..., freshness_basis: _Optional[str] = ..., candidate_identity: _Optional[str] = ..., max_age_seconds: _Optional[int] = ..., limitation: _Optional[str] = ..., next_action: _Optional[str] = ..., created_at: _Optional[str] = ...) -> None: ...

class CapabilityLink(_message.Message):
    __slots__ = ("id", "capability_id", "relation", "target_id", "created_at")
    ID_FIELD_NUMBER: _ClassVar[int]
    CAPABILITY_ID_FIELD_NUMBER: _ClassVar[int]
    RELATION_FIELD_NUMBER: _ClassVar[int]
    TARGET_ID_FIELD_NUMBER: _ClassVar[int]
    CREATED_AT_FIELD_NUMBER: _ClassVar[int]
    id: str
    capability_id: str
    relation: str
    target_id: str
    created_at: str
    def __init__(self, id: _Optional[str] = ..., capability_id: _Optional[str] = ..., relation: _Optional[str] = ..., target_id: _Optional[str] = ..., created_at: _Optional[str] = ...) -> None: ...

class ListCapabilitiesRequest(_message.Message):
    __slots__ = ("medium",)
    MEDIUM_FIELD_NUMBER: _ClassVar[int]
    medium: str
    def __init__(self, medium: _Optional[str] = ...) -> None: ...

class ListCapabilitiesResponse(_message.Message):
    __slots__ = ("capabilities",)
    CAPABILITIES_FIELD_NUMBER: _ClassVar[int]
    capabilities: _containers.RepeatedCompositeFieldContainer[Capability]
    def __init__(self, capabilities: _Optional[_Iterable[_Union[Capability, _Mapping]]] = ...) -> None: ...

class GetCapabilityRequest(_message.Message):
    __slots__ = ("id", "alias")
    ID_FIELD_NUMBER: _ClassVar[int]
    ALIAS_FIELD_NUMBER: _ClassVar[int]
    id: str
    alias: str
    def __init__(self, id: _Optional[str] = ..., alias: _Optional[str] = ...) -> None: ...

class GetCapabilityResponse(_message.Message):
    __slots__ = ("capability",)
    CAPABILITY_FIELD_NUMBER: _ClassVar[int]
    capability: Capability
    def __init__(self, capability: _Optional[_Union[Capability, _Mapping]] = ...) -> None: ...

class UpsertCapabilityRequest(_message.Message):
    __slots__ = ("capability",)
    CAPABILITY_FIELD_NUMBER: _ClassVar[int]
    capability: Capability
    def __init__(self, capability: _Optional[_Union[Capability, _Mapping]] = ...) -> None: ...

class UpsertCapabilityResponse(_message.Message):
    __slots__ = ("capability",)
    CAPABILITY_FIELD_NUMBER: _ClassVar[int]
    capability: Capability
    def __init__(self, capability: _Optional[_Union[Capability, _Mapping]] = ...) -> None: ...

class RecordQualificationRequest(_message.Message):
    __slots__ = ("qualification",)
    QUALIFICATION_FIELD_NUMBER: _ClassVar[int]
    qualification: CapabilityQualification
    def __init__(self, qualification: _Optional[_Union[CapabilityQualification, _Mapping]] = ...) -> None: ...

class RecordQualificationResponse(_message.Message):
    __slots__ = ("qualification",)
    QUALIFICATION_FIELD_NUMBER: _ClassVar[int]
    qualification: CapabilityQualification
    def __init__(self, qualification: _Optional[_Union[CapabilityQualification, _Mapping]] = ...) -> None: ...

class LinkCapabilityRequest(_message.Message):
    __slots__ = ("link",)
    LINK_FIELD_NUMBER: _ClassVar[int]
    link: CapabilityLink
    def __init__(self, link: _Optional[_Union[CapabilityLink, _Mapping]] = ...) -> None: ...

class LinkCapabilityResponse(_message.Message):
    __slots__ = ("link",)
    LINK_FIELD_NUMBER: _ClassVar[int]
    link: CapabilityLink
    def __init__(self, link: _Optional[_Union[CapabilityLink, _Mapping]] = ...) -> None: ...
