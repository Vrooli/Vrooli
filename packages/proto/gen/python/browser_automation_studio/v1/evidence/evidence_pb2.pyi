import datetime

from buf.validate import validate_pb2 as _validate_pb2
from browser_automation_studio.v1.timeline import entry_pb2 as _entry_pb2
from google.protobuf import struct_pb2 as _struct_pb2
from google.protobuf import timestamp_pb2 as _timestamp_pb2
from google.protobuf.internal import containers as _containers
from google.protobuf.internal import enum_type_wrapper as _enum_type_wrapper
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Iterable as _Iterable, Mapping as _Mapping
from typing import ClassVar as _ClassVar, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class ArtifactKind(int, metaclass=_enum_type_wrapper.EnumTypeWrapper):
    __slots__ = ()
    ARTIFACT_KIND_UNSPECIFIED: _ClassVar[ArtifactKind]
    ARTIFACT_KIND_SCREENSHOT: _ClassVar[ArtifactKind]
    ARTIFACT_KIND_VIDEO: _ClassVar[ArtifactKind]
    ARTIFACT_KIND_TRACE: _ClassVar[ArtifactKind]
    ARTIFACT_KIND_HAR: _ClassVar[ArtifactKind]
    ARTIFACT_KIND_DOM: _ClassVar[ArtifactKind]
    ARTIFACT_KIND_CONSOLE: _ClassVar[ArtifactKind]
    ARTIFACT_KIND_NETWORK: _ClassVar[ArtifactKind]
    ARTIFACT_KIND_ACCESSIBILITY: _ClassVar[ArtifactKind]
    ARTIFACT_KIND_PERFORMANCE: _ClassVar[ArtifactKind]

class ContentClassification(int, metaclass=_enum_type_wrapper.EnumTypeWrapper):
    __slots__ = ()
    CONTENT_CLASSIFICATION_UNSPECIFIED: _ClassVar[ContentClassification]
    CONTENT_CLASSIFICATION_PUBLIC: _ClassVar[ContentClassification]
    CONTENT_CLASSIFICATION_INTERNAL: _ClassVar[ContentClassification]
    CONTENT_CLASSIFICATION_SENSITIVE: _ClassVar[ContentClassification]
    CONTENT_CLASSIFICATION_RESTRICTED: _ClassVar[ContentClassification]

class RetentionClass(int, metaclass=_enum_type_wrapper.EnumTypeWrapper):
    __slots__ = ()
    RETENTION_CLASS_UNSPECIFIED: _ClassVar[RetentionClass]
    RETENTION_CLASS_EPHEMERAL: _ClassVar[RetentionClass]
    RETENTION_CLASS_STANDARD: _ClassVar[RetentionClass]
    RETENTION_CLASS_AUDIT: _ClassVar[RetentionClass]
    RETENTION_CLASS_PROTECTED: _ClassVar[RetentionClass]

class AccessPolicy(int, metaclass=_enum_type_wrapper.EnumTypeWrapper):
    __slots__ = ()
    ACCESS_POLICY_UNSPECIFIED: _ClassVar[AccessPolicy]
    ACCESS_POLICY_EXECUTION_OWNER: _ClassVar[AccessPolicy]
    ACCESS_POLICY_PROJECT_MEMBERS: _ClassVar[AccessPolicy]
    ACCESS_POLICY_EXPLICIT_GRANT: _ClassVar[AccessPolicy]
    ACCESS_POLICY_PROTECTED_STORAGE_ONLY: _ClassVar[AccessPolicy]
ARTIFACT_KIND_UNSPECIFIED: ArtifactKind
ARTIFACT_KIND_SCREENSHOT: ArtifactKind
ARTIFACT_KIND_VIDEO: ArtifactKind
ARTIFACT_KIND_TRACE: ArtifactKind
ARTIFACT_KIND_HAR: ArtifactKind
ARTIFACT_KIND_DOM: ArtifactKind
ARTIFACT_KIND_CONSOLE: ArtifactKind
ARTIFACT_KIND_NETWORK: ArtifactKind
ARTIFACT_KIND_ACCESSIBILITY: ArtifactKind
ARTIFACT_KIND_PERFORMANCE: ArtifactKind
CONTENT_CLASSIFICATION_UNSPECIFIED: ContentClassification
CONTENT_CLASSIFICATION_PUBLIC: ContentClassification
CONTENT_CLASSIFICATION_INTERNAL: ContentClassification
CONTENT_CLASSIFICATION_SENSITIVE: ContentClassification
CONTENT_CLASSIFICATION_RESTRICTED: ContentClassification
RETENTION_CLASS_UNSPECIFIED: RetentionClass
RETENTION_CLASS_EPHEMERAL: RetentionClass
RETENTION_CLASS_STANDARD: RetentionClass
RETENTION_CLASS_AUDIT: RetentionClass
RETENTION_CLASS_PROTECTED: RetentionClass
ACCESS_POLICY_UNSPECIFIED: AccessPolicy
ACCESS_POLICY_EXECUTION_OWNER: AccessPolicy
ACCESS_POLICY_PROJECT_MEMBERS: AccessPolicy
ACCESS_POLICY_EXPLICIT_GRANT: AccessPolicy
ACCESS_POLICY_PROTECTED_STORAGE_ONLY: AccessPolicy

class ArtifactManifest(_message.Message):
    __slots__ = ("id", "kind", "media_type", "size_bytes", "sha256", "classification", "retention_class", "access_policy", "redacted", "sanitized_derivative_id", "captured_at", "execution_id", "timeline_entry_id", "producer", "provenance")
    ID_FIELD_NUMBER: _ClassVar[int]
    KIND_FIELD_NUMBER: _ClassVar[int]
    MEDIA_TYPE_FIELD_NUMBER: _ClassVar[int]
    SIZE_BYTES_FIELD_NUMBER: _ClassVar[int]
    SHA256_FIELD_NUMBER: _ClassVar[int]
    CLASSIFICATION_FIELD_NUMBER: _ClassVar[int]
    RETENTION_CLASS_FIELD_NUMBER: _ClassVar[int]
    ACCESS_POLICY_FIELD_NUMBER: _ClassVar[int]
    REDACTED_FIELD_NUMBER: _ClassVar[int]
    SANITIZED_DERIVATIVE_ID_FIELD_NUMBER: _ClassVar[int]
    CAPTURED_AT_FIELD_NUMBER: _ClassVar[int]
    EXECUTION_ID_FIELD_NUMBER: _ClassVar[int]
    TIMELINE_ENTRY_ID_FIELD_NUMBER: _ClassVar[int]
    PRODUCER_FIELD_NUMBER: _ClassVar[int]
    PROVENANCE_FIELD_NUMBER: _ClassVar[int]
    id: str
    kind: ArtifactKind
    media_type: str
    size_bytes: int
    sha256: str
    classification: ContentClassification
    retention_class: RetentionClass
    access_policy: AccessPolicy
    redacted: bool
    sanitized_derivative_id: str
    captured_at: _timestamp_pb2.Timestamp
    execution_id: str
    timeline_entry_id: str
    producer: str
    provenance: _struct_pb2.Struct
    def __init__(self, id: _Optional[str] = ..., kind: _Optional[_Union[ArtifactKind, str]] = ..., media_type: _Optional[str] = ..., size_bytes: _Optional[int] = ..., sha256: _Optional[str] = ..., classification: _Optional[_Union[ContentClassification, str]] = ..., retention_class: _Optional[_Union[RetentionClass, str]] = ..., access_policy: _Optional[_Union[AccessPolicy, str]] = ..., redacted: _Optional[bool] = ..., sanitized_derivative_id: _Optional[str] = ..., captured_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., execution_id: _Optional[str] = ..., timeline_entry_id: _Optional[str] = ..., producer: _Optional[str] = ..., provenance: _Optional[_Union[_struct_pb2.Struct, _Mapping]] = ...) -> None: ...

class EvidencePolicy(_message.Message):
    __slots__ = ("max_artifact_size_bytes", "default_retention_class", "default_access_policy", "redact_har", "redact_network", "redacted_header_names", "redacted_query_parameter_names")
    MAX_ARTIFACT_SIZE_BYTES_FIELD_NUMBER: _ClassVar[int]
    DEFAULT_RETENTION_CLASS_FIELD_NUMBER: _ClassVar[int]
    DEFAULT_ACCESS_POLICY_FIELD_NUMBER: _ClassVar[int]
    REDACT_HAR_FIELD_NUMBER: _ClassVar[int]
    REDACT_NETWORK_FIELD_NUMBER: _ClassVar[int]
    REDACTED_HEADER_NAMES_FIELD_NUMBER: _ClassVar[int]
    REDACTED_QUERY_PARAMETER_NAMES_FIELD_NUMBER: _ClassVar[int]
    max_artifact_size_bytes: int
    default_retention_class: RetentionClass
    default_access_policy: AccessPolicy
    redact_har: bool
    redact_network: bool
    redacted_header_names: _containers.RepeatedScalarFieldContainer[str]
    redacted_query_parameter_names: _containers.RepeatedScalarFieldContainer[str]
    def __init__(self, max_artifact_size_bytes: _Optional[int] = ..., default_retention_class: _Optional[_Union[RetentionClass, str]] = ..., default_access_policy: _Optional[_Union[AccessPolicy, str]] = ..., redact_har: _Optional[bool] = ..., redact_network: _Optional[bool] = ..., redacted_header_names: _Optional[_Iterable[str]] = ..., redacted_query_parameter_names: _Optional[_Iterable[str]] = ...) -> None: ...

class EvidenceManifest(_message.Message):
    __slots__ = ("id", "execution_id", "schema_version", "policy", "artifacts", "created_at")
    ID_FIELD_NUMBER: _ClassVar[int]
    EXECUTION_ID_FIELD_NUMBER: _ClassVar[int]
    SCHEMA_VERSION_FIELD_NUMBER: _ClassVar[int]
    POLICY_FIELD_NUMBER: _ClassVar[int]
    ARTIFACTS_FIELD_NUMBER: _ClassVar[int]
    CREATED_AT_FIELD_NUMBER: _ClassVar[int]
    id: str
    execution_id: str
    schema_version: str
    policy: EvidencePolicy
    artifacts: _containers.RepeatedCompositeFieldContainer[ArtifactManifest]
    created_at: _timestamp_pb2.Timestamp
    def __init__(self, id: _Optional[str] = ..., execution_id: _Optional[str] = ..., schema_version: _Optional[str] = ..., policy: _Optional[_Union[EvidencePolicy, _Mapping]] = ..., artifacts: _Optional[_Iterable[_Union[ArtifactManifest, _Mapping]]] = ..., created_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ...) -> None: ...

class ReplayPackage(_message.Message):
    __slots__ = ("id", "schema_version", "execution_id", "workflow_id", "evidence", "timeline", "presentation", "created_at")
    ID_FIELD_NUMBER: _ClassVar[int]
    SCHEMA_VERSION_FIELD_NUMBER: _ClassVar[int]
    EXECUTION_ID_FIELD_NUMBER: _ClassVar[int]
    WORKFLOW_ID_FIELD_NUMBER: _ClassVar[int]
    EVIDENCE_FIELD_NUMBER: _ClassVar[int]
    TIMELINE_FIELD_NUMBER: _ClassVar[int]
    PRESENTATION_FIELD_NUMBER: _ClassVar[int]
    CREATED_AT_FIELD_NUMBER: _ClassVar[int]
    id: str
    schema_version: str
    execution_id: str
    workflow_id: str
    evidence: EvidenceManifest
    timeline: _containers.RepeatedCompositeFieldContainer[_entry_pb2.TimelineEntry]
    presentation: _struct_pb2.Struct
    created_at: _timestamp_pb2.Timestamp
    def __init__(self, id: _Optional[str] = ..., schema_version: _Optional[str] = ..., execution_id: _Optional[str] = ..., workflow_id: _Optional[str] = ..., evidence: _Optional[_Union[EvidenceManifest, _Mapping]] = ..., timeline: _Optional[_Iterable[_Union[_entry_pb2.TimelineEntry, _Mapping]]] = ..., presentation: _Optional[_Union[_struct_pb2.Struct, _Mapping]] = ..., created_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ...) -> None: ...
