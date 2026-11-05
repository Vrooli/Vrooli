import datetime

from google.protobuf import timestamp_pb2 as _timestamp_pb2
from google.protobuf.internal import containers as _containers
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Iterable as _Iterable, Mapping as _Mapping
from typing import ClassVar as _ClassVar, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class GetAuthorityStatusRequest(_message.Message):
    __slots__ = ()
    def __init__(self) -> None: ...

class AuthorityStatus(_message.Message):
    __slots__ = ("authenticated", "principal_id", "email", "realm", "caller_kind", "can_mutate", "reason", "capabilities", "auth_source", "auth_state", "recovery_url", "failure_class", "auth_sources")
    AUTHENTICATED_FIELD_NUMBER: _ClassVar[int]
    PRINCIPAL_ID_FIELD_NUMBER: _ClassVar[int]
    EMAIL_FIELD_NUMBER: _ClassVar[int]
    REALM_FIELD_NUMBER: _ClassVar[int]
    CALLER_KIND_FIELD_NUMBER: _ClassVar[int]
    CAN_MUTATE_FIELD_NUMBER: _ClassVar[int]
    REASON_FIELD_NUMBER: _ClassVar[int]
    CAPABILITIES_FIELD_NUMBER: _ClassVar[int]
    AUTH_SOURCE_FIELD_NUMBER: _ClassVar[int]
    AUTH_STATE_FIELD_NUMBER: _ClassVar[int]
    RECOVERY_URL_FIELD_NUMBER: _ClassVar[int]
    FAILURE_CLASS_FIELD_NUMBER: _ClassVar[int]
    AUTH_SOURCES_FIELD_NUMBER: _ClassVar[int]
    authenticated: bool
    principal_id: str
    email: str
    realm: str
    caller_kind: str
    can_mutate: bool
    reason: str
    capabilities: _containers.RepeatedScalarFieldContainer[str]
    auth_source: str
    auth_state: str
    recovery_url: str
    failure_class: str
    auth_sources: _containers.RepeatedScalarFieldContainer[str]
    def __init__(self, authenticated: _Optional[bool] = ..., principal_id: _Optional[str] = ..., email: _Optional[str] = ..., realm: _Optional[str] = ..., caller_kind: _Optional[str] = ..., can_mutate: _Optional[bool] = ..., reason: _Optional[str] = ..., capabilities: _Optional[_Iterable[str]] = ..., auth_source: _Optional[str] = ..., auth_state: _Optional[str] = ..., recovery_url: _Optional[str] = ..., failure_class: _Optional[str] = ..., auth_sources: _Optional[_Iterable[str]] = ...) -> None: ...

class PrepareMutationRequest(_message.Message):
    __slots__ = ("repository_id", "operation", "subject_context")
    REPOSITORY_ID_FIELD_NUMBER: _ClassVar[int]
    OPERATION_FIELD_NUMBER: _ClassVar[int]
    SUBJECT_CONTEXT_FIELD_NUMBER: _ClassVar[int]
    repository_id: str
    operation: str
    subject_context: str
    def __init__(self, repository_id: _Optional[str] = ..., operation: _Optional[str] = ..., subject_context: _Optional[str] = ...) -> None: ...

class MutationPreview(_message.Message):
    __slots__ = ("repository_id", "repository_path", "operation", "branch", "expected_revision", "subject_digest", "staged_files", "file_count", "generated_at", "subject_context")
    REPOSITORY_ID_FIELD_NUMBER: _ClassVar[int]
    REPOSITORY_PATH_FIELD_NUMBER: _ClassVar[int]
    OPERATION_FIELD_NUMBER: _ClassVar[int]
    BRANCH_FIELD_NUMBER: _ClassVar[int]
    EXPECTED_REVISION_FIELD_NUMBER: _ClassVar[int]
    SUBJECT_DIGEST_FIELD_NUMBER: _ClassVar[int]
    STAGED_FILES_FIELD_NUMBER: _ClassVar[int]
    FILE_COUNT_FIELD_NUMBER: _ClassVar[int]
    GENERATED_AT_FIELD_NUMBER: _ClassVar[int]
    SUBJECT_CONTEXT_FIELD_NUMBER: _ClassVar[int]
    repository_id: str
    repository_path: str
    operation: str
    branch: str
    expected_revision: str
    subject_digest: str
    staged_files: _containers.RepeatedScalarFieldContainer[str]
    file_count: int
    generated_at: _timestamp_pb2.Timestamp
    subject_context: str
    def __init__(self, repository_id: _Optional[str] = ..., repository_path: _Optional[str] = ..., operation: _Optional[str] = ..., branch: _Optional[str] = ..., expected_revision: _Optional[str] = ..., subject_digest: _Optional[str] = ..., staged_files: _Optional[_Iterable[str]] = ..., file_count: _Optional[int] = ..., generated_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., subject_context: _Optional[str] = ...) -> None: ...

class ConfirmMutationRequest(_message.Message):
    __slots__ = ("repository_id", "operation", "expected_revision", "subject_digest", "subject_context")
    REPOSITORY_ID_FIELD_NUMBER: _ClassVar[int]
    OPERATION_FIELD_NUMBER: _ClassVar[int]
    EXPECTED_REVISION_FIELD_NUMBER: _ClassVar[int]
    SUBJECT_DIGEST_FIELD_NUMBER: _ClassVar[int]
    SUBJECT_CONTEXT_FIELD_NUMBER: _ClassVar[int]
    repository_id: str
    operation: str
    expected_revision: str
    subject_digest: str
    subject_context: str
    def __init__(self, repository_id: _Optional[str] = ..., operation: _Optional[str] = ..., expected_revision: _Optional[str] = ..., subject_digest: _Optional[str] = ..., subject_context: _Optional[str] = ...) -> None: ...

class MutationIntent(_message.Message):
    __slots__ = ("intent_id", "principal_id", "repository_id", "operation", "expected_revision", "subject_digest", "expires_at", "single_use")
    INTENT_ID_FIELD_NUMBER: _ClassVar[int]
    PRINCIPAL_ID_FIELD_NUMBER: _ClassVar[int]
    REPOSITORY_ID_FIELD_NUMBER: _ClassVar[int]
    OPERATION_FIELD_NUMBER: _ClassVar[int]
    EXPECTED_REVISION_FIELD_NUMBER: _ClassVar[int]
    SUBJECT_DIGEST_FIELD_NUMBER: _ClassVar[int]
    EXPIRES_AT_FIELD_NUMBER: _ClassVar[int]
    SINGLE_USE_FIELD_NUMBER: _ClassVar[int]
    intent_id: str
    principal_id: str
    repository_id: str
    operation: str
    expected_revision: str
    subject_digest: str
    expires_at: _timestamp_pb2.Timestamp
    single_use: bool
    def __init__(self, intent_id: _Optional[str] = ..., principal_id: _Optional[str] = ..., repository_id: _Optional[str] = ..., operation: _Optional[str] = ..., expected_revision: _Optional[str] = ..., subject_digest: _Optional[str] = ..., expires_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., single_use: _Optional[bool] = ...) -> None: ...
