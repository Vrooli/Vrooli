from google.protobuf.internal import containers as _containers
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Iterable as _Iterable, Mapping as _Mapping
from typing import ClassVar as _ClassVar, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class DraftRequest(_message.Message):
    __slots__ = ("kind", "subject", "evidence")
    KIND_FIELD_NUMBER: _ClassVar[int]
    SUBJECT_FIELD_NUMBER: _ClassVar[int]
    EVIDENCE_FIELD_NUMBER: _ClassVar[int]
    kind: str
    subject: ChangeSubject
    evidence: EvidenceBundle
    def __init__(self, kind: _Optional[str] = ..., subject: _Optional[_Union[ChangeSubject, _Mapping]] = ..., evidence: _Optional[_Union[EvidenceBundle, _Mapping]] = ...) -> None: ...

class ChangeSubject(_message.Message):
    __slots__ = ("repository_id", "kind", "host", "base_revision", "head_revision", "parent_revision", "scope", "snapshot_digest")
    REPOSITORY_ID_FIELD_NUMBER: _ClassVar[int]
    KIND_FIELD_NUMBER: _ClassVar[int]
    HOST_FIELD_NUMBER: _ClassVar[int]
    BASE_REVISION_FIELD_NUMBER: _ClassVar[int]
    HEAD_REVISION_FIELD_NUMBER: _ClassVar[int]
    PARENT_REVISION_FIELD_NUMBER: _ClassVar[int]
    SCOPE_FIELD_NUMBER: _ClassVar[int]
    SNAPSHOT_DIGEST_FIELD_NUMBER: _ClassVar[int]
    repository_id: str
    kind: str
    host: HostSubject
    base_revision: str
    head_revision: str
    parent_revision: str
    scope: SubjectScope
    snapshot_digest: str
    def __init__(self, repository_id: _Optional[str] = ..., kind: _Optional[str] = ..., host: _Optional[_Union[HostSubject, _Mapping]] = ..., base_revision: _Optional[str] = ..., head_revision: _Optional[str] = ..., parent_revision: _Optional[str] = ..., scope: _Optional[_Union[SubjectScope, _Mapping]] = ..., snapshot_digest: _Optional[str] = ...) -> None: ...

class HostSubject(_message.Message):
    __slots__ = ("provider", "instance_id", "repository_id", "change_number")
    PROVIDER_FIELD_NUMBER: _ClassVar[int]
    INSTANCE_ID_FIELD_NUMBER: _ClassVar[int]
    REPOSITORY_ID_FIELD_NUMBER: _ClassVar[int]
    CHANGE_NUMBER_FIELD_NUMBER: _ClassVar[int]
    provider: str
    instance_id: str
    repository_id: str
    change_number: str
    def __init__(self, provider: _Optional[str] = ..., instance_id: _Optional[str] = ..., repository_id: _Optional[str] = ..., change_number: _Optional[str] = ...) -> None: ...

class SubjectScope(_message.Message):
    __slots__ = ("paths", "selection_digest")
    PATHS_FIELD_NUMBER: _ClassVar[int]
    SELECTION_DIGEST_FIELD_NUMBER: _ClassVar[int]
    paths: _containers.RepeatedScalarFieldContainer[str]
    selection_digest: str
    def __init__(self, paths: _Optional[_Iterable[str]] = ..., selection_digest: _Optional[str] = ...) -> None: ...

class EvidenceBundle(_message.Message):
    __slots__ = ("operation_id", "subject_digest", "status", "claims", "coverage", "validation", "unknowns", "versions")
    class VersionsEntry(_message.Message):
        __slots__ = ("key", "value")
        KEY_FIELD_NUMBER: _ClassVar[int]
        VALUE_FIELD_NUMBER: _ClassVar[int]
        key: str
        value: str
        def __init__(self, key: _Optional[str] = ..., value: _Optional[str] = ...) -> None: ...
    OPERATION_ID_FIELD_NUMBER: _ClassVar[int]
    SUBJECT_DIGEST_FIELD_NUMBER: _ClassVar[int]
    STATUS_FIELD_NUMBER: _ClassVar[int]
    CLAIMS_FIELD_NUMBER: _ClassVar[int]
    COVERAGE_FIELD_NUMBER: _ClassVar[int]
    VALIDATION_FIELD_NUMBER: _ClassVar[int]
    UNKNOWNS_FIELD_NUMBER: _ClassVar[int]
    VERSIONS_FIELD_NUMBER: _ClassVar[int]
    operation_id: str
    subject_digest: str
    status: str
    claims: _containers.RepeatedCompositeFieldContainer[Claim]
    coverage: Coverage
    validation: _containers.RepeatedCompositeFieldContainer[ValidationRef]
    unknowns: _containers.RepeatedScalarFieldContainer[str]
    versions: _containers.ScalarMap[str, str]
    def __init__(self, operation_id: _Optional[str] = ..., subject_digest: _Optional[str] = ..., status: _Optional[str] = ..., claims: _Optional[_Iterable[_Union[Claim, _Mapping]]] = ..., coverage: _Optional[_Union[Coverage, _Mapping]] = ..., validation: _Optional[_Iterable[_Union[ValidationRef, _Mapping]]] = ..., unknowns: _Optional[_Iterable[str]] = ..., versions: _Optional[_Mapping[str, str]] = ...) -> None: ...

class Claim(_message.Message):
    __slots__ = ("text", "evidence_refs")
    TEXT_FIELD_NUMBER: _ClassVar[int]
    EVIDENCE_REFS_FIELD_NUMBER: _ClassVar[int]
    text: str
    evidence_refs: _containers.RepeatedScalarFieldContainer[str]
    def __init__(self, text: _Optional[str] = ..., evidence_refs: _Optional[_Iterable[str]] = ...) -> None: ...

class Coverage(_message.Message):
    __slots__ = ("included_files", "omitted_files", "omissions")
    INCLUDED_FILES_FIELD_NUMBER: _ClassVar[int]
    OMITTED_FILES_FIELD_NUMBER: _ClassVar[int]
    OMISSIONS_FIELD_NUMBER: _ClassVar[int]
    included_files: int
    omitted_files: int
    omissions: _containers.RepeatedCompositeFieldContainer[Omission]
    def __init__(self, included_files: _Optional[int] = ..., omitted_files: _Optional[int] = ..., omissions: _Optional[_Iterable[_Union[Omission, _Mapping]]] = ...) -> None: ...

class Omission(_message.Message):
    __slots__ = ("path", "reason")
    PATH_FIELD_NUMBER: _ClassVar[int]
    REASON_FIELD_NUMBER: _ClassVar[int]
    path: str
    reason: str
    def __init__(self, path: _Optional[str] = ..., reason: _Optional[str] = ...) -> None: ...

class ValidationRef(_message.Message):
    __slots__ = ("execution_id", "availability", "verdict")
    EXECUTION_ID_FIELD_NUMBER: _ClassVar[int]
    AVAILABILITY_FIELD_NUMBER: _ClassVar[int]
    VERDICT_FIELD_NUMBER: _ClassVar[int]
    execution_id: str
    availability: str
    verdict: str
    def __init__(self, execution_id: _Optional[str] = ..., availability: _Optional[str] = ..., verdict: _Optional[str] = ...) -> None: ...

class DraftResponse(_message.Message):
    __slots__ = ("status", "kind", "subject_digest", "title", "body", "unknowns", "coverage", "evidence_refs")
    STATUS_FIELD_NUMBER: _ClassVar[int]
    KIND_FIELD_NUMBER: _ClassVar[int]
    SUBJECT_DIGEST_FIELD_NUMBER: _ClassVar[int]
    TITLE_FIELD_NUMBER: _ClassVar[int]
    BODY_FIELD_NUMBER: _ClassVar[int]
    UNKNOWNS_FIELD_NUMBER: _ClassVar[int]
    COVERAGE_FIELD_NUMBER: _ClassVar[int]
    EVIDENCE_REFS_FIELD_NUMBER: _ClassVar[int]
    status: str
    kind: str
    subject_digest: str
    title: str
    body: str
    unknowns: _containers.RepeatedScalarFieldContainer[str]
    coverage: Coverage
    evidence_refs: _containers.RepeatedScalarFieldContainer[str]
    def __init__(self, status: _Optional[str] = ..., kind: _Optional[str] = ..., subject_digest: _Optional[str] = ..., title: _Optional[str] = ..., body: _Optional[str] = ..., unknowns: _Optional[_Iterable[str]] = ..., coverage: _Optional[_Union[Coverage, _Mapping]] = ..., evidence_refs: _Optional[_Iterable[str]] = ...) -> None: ...
