from google.protobuf.internal import containers as _containers
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Iterable as _Iterable, Mapping as _Mapping
from typing import ClassVar as _ClassVar, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class Commitment(_message.Message):
    __slots__ = ("id", "result", "definition_of_done", "promised_boundary", "timezone", "beneficiary", "assumptions", "scope_exclusions", "state", "risk", "acknowledgment_status", "created_at_unix_seconds", "updated_at_unix_seconds", "revision")
    ID_FIELD_NUMBER: _ClassVar[int]
    RESULT_FIELD_NUMBER: _ClassVar[int]
    DEFINITION_OF_DONE_FIELD_NUMBER: _ClassVar[int]
    PROMISED_BOUNDARY_FIELD_NUMBER: _ClassVar[int]
    TIMEZONE_FIELD_NUMBER: _ClassVar[int]
    BENEFICIARY_FIELD_NUMBER: _ClassVar[int]
    ASSUMPTIONS_FIELD_NUMBER: _ClassVar[int]
    SCOPE_EXCLUSIONS_FIELD_NUMBER: _ClassVar[int]
    STATE_FIELD_NUMBER: _ClassVar[int]
    RISK_FIELD_NUMBER: _ClassVar[int]
    ACKNOWLEDGMENT_STATUS_FIELD_NUMBER: _ClassVar[int]
    CREATED_AT_UNIX_SECONDS_FIELD_NUMBER: _ClassVar[int]
    UPDATED_AT_UNIX_SECONDS_FIELD_NUMBER: _ClassVar[int]
    REVISION_FIELD_NUMBER: _ClassVar[int]
    id: str
    result: str
    definition_of_done: str
    promised_boundary: str
    timezone: str
    beneficiary: str
    assumptions: str
    scope_exclusions: str
    state: str
    risk: str
    acknowledgment_status: str
    created_at_unix_seconds: int
    updated_at_unix_seconds: int
    revision: int
    def __init__(self, id: _Optional[str] = ..., result: _Optional[str] = ..., definition_of_done: _Optional[str] = ..., promised_boundary: _Optional[str] = ..., timezone: _Optional[str] = ..., beneficiary: _Optional[str] = ..., assumptions: _Optional[str] = ..., scope_exclusions: _Optional[str] = ..., state: _Optional[str] = ..., risk: _Optional[str] = ..., acknowledgment_status: _Optional[str] = ..., created_at_unix_seconds: _Optional[int] = ..., updated_at_unix_seconds: _Optional[int] = ..., revision: _Optional[int] = ...) -> None: ...

class CommitmentRevision(_message.Message):
    __slots__ = ("id", "commitment_id", "promised_boundary", "assumptions", "scope_exclusions", "reason", "acknowledgment_status", "created_at_unix_seconds", "revision")
    ID_FIELD_NUMBER: _ClassVar[int]
    COMMITMENT_ID_FIELD_NUMBER: _ClassVar[int]
    PROMISED_BOUNDARY_FIELD_NUMBER: _ClassVar[int]
    ASSUMPTIONS_FIELD_NUMBER: _ClassVar[int]
    SCOPE_EXCLUSIONS_FIELD_NUMBER: _ClassVar[int]
    REASON_FIELD_NUMBER: _ClassVar[int]
    ACKNOWLEDGMENT_STATUS_FIELD_NUMBER: _ClassVar[int]
    CREATED_AT_UNIX_SECONDS_FIELD_NUMBER: _ClassVar[int]
    REVISION_FIELD_NUMBER: _ClassVar[int]
    id: str
    commitment_id: str
    promised_boundary: str
    assumptions: str
    scope_exclusions: str
    reason: str
    acknowledgment_status: str
    created_at_unix_seconds: int
    revision: int
    def __init__(self, id: _Optional[str] = ..., commitment_id: _Optional[str] = ..., promised_boundary: _Optional[str] = ..., assumptions: _Optional[str] = ..., scope_exclusions: _Optional[str] = ..., reason: _Optional[str] = ..., acknowledgment_status: _Optional[str] = ..., created_at_unix_seconds: _Optional[int] = ..., revision: _Optional[int] = ...) -> None: ...

class ListCommitmentsRequest(_message.Message):
    __slots__ = ()
    def __init__(self) -> None: ...

class ListCommitmentsResponse(_message.Message):
    __slots__ = ("commitments",)
    COMMITMENTS_FIELD_NUMBER: _ClassVar[int]
    commitments: _containers.RepeatedCompositeFieldContainer[Commitment]
    def __init__(self, commitments: _Optional[_Iterable[_Union[Commitment, _Mapping]]] = ...) -> None: ...

class CreateCommitmentRequest(_message.Message):
    __slots__ = ("result", "definition_of_done", "promised_boundary", "timezone", "beneficiary", "assumptions", "scope_exclusions", "state")
    RESULT_FIELD_NUMBER: _ClassVar[int]
    DEFINITION_OF_DONE_FIELD_NUMBER: _ClassVar[int]
    PROMISED_BOUNDARY_FIELD_NUMBER: _ClassVar[int]
    TIMEZONE_FIELD_NUMBER: _ClassVar[int]
    BENEFICIARY_FIELD_NUMBER: _ClassVar[int]
    ASSUMPTIONS_FIELD_NUMBER: _ClassVar[int]
    SCOPE_EXCLUSIONS_FIELD_NUMBER: _ClassVar[int]
    STATE_FIELD_NUMBER: _ClassVar[int]
    result: str
    definition_of_done: str
    promised_boundary: str
    timezone: str
    beneficiary: str
    assumptions: str
    scope_exclusions: str
    state: str
    def __init__(self, result: _Optional[str] = ..., definition_of_done: _Optional[str] = ..., promised_boundary: _Optional[str] = ..., timezone: _Optional[str] = ..., beneficiary: _Optional[str] = ..., assumptions: _Optional[str] = ..., scope_exclusions: _Optional[str] = ..., state: _Optional[str] = ...) -> None: ...

class CreateCommitmentResponse(_message.Message):
    __slots__ = ("commitment",)
    COMMITMENT_FIELD_NUMBER: _ClassVar[int]
    commitment: Commitment
    def __init__(self, commitment: _Optional[_Union[Commitment, _Mapping]] = ...) -> None: ...

class UpdateCommitmentStateRequest(_message.Message):
    __slots__ = ("id", "state", "expected_revision")
    ID_FIELD_NUMBER: _ClassVar[int]
    STATE_FIELD_NUMBER: _ClassVar[int]
    EXPECTED_REVISION_FIELD_NUMBER: _ClassVar[int]
    id: str
    state: str
    expected_revision: int
    def __init__(self, id: _Optional[str] = ..., state: _Optional[str] = ..., expected_revision: _Optional[int] = ...) -> None: ...

class UpdateCommitmentStateResponse(_message.Message):
    __slots__ = ("commitment",)
    COMMITMENT_FIELD_NUMBER: _ClassVar[int]
    commitment: Commitment
    def __init__(self, commitment: _Optional[_Union[Commitment, _Mapping]] = ...) -> None: ...

class ReviseCommitmentRequest(_message.Message):
    __slots__ = ("id", "promised_boundary", "assumptions", "scope_exclusions", "reason", "acknowledgment_status", "expected_revision")
    ID_FIELD_NUMBER: _ClassVar[int]
    PROMISED_BOUNDARY_FIELD_NUMBER: _ClassVar[int]
    ASSUMPTIONS_FIELD_NUMBER: _ClassVar[int]
    SCOPE_EXCLUSIONS_FIELD_NUMBER: _ClassVar[int]
    REASON_FIELD_NUMBER: _ClassVar[int]
    ACKNOWLEDGMENT_STATUS_FIELD_NUMBER: _ClassVar[int]
    EXPECTED_REVISION_FIELD_NUMBER: _ClassVar[int]
    id: str
    promised_boundary: str
    assumptions: str
    scope_exclusions: str
    reason: str
    acknowledgment_status: str
    expected_revision: int
    def __init__(self, id: _Optional[str] = ..., promised_boundary: _Optional[str] = ..., assumptions: _Optional[str] = ..., scope_exclusions: _Optional[str] = ..., reason: _Optional[str] = ..., acknowledgment_status: _Optional[str] = ..., expected_revision: _Optional[int] = ...) -> None: ...

class ReviseCommitmentResponse(_message.Message):
    __slots__ = ("commitment", "revision")
    COMMITMENT_FIELD_NUMBER: _ClassVar[int]
    REVISION_FIELD_NUMBER: _ClassVar[int]
    commitment: Commitment
    revision: CommitmentRevision
    def __init__(self, commitment: _Optional[_Union[Commitment, _Mapping]] = ..., revision: _Optional[_Union[CommitmentRevision, _Mapping]] = ...) -> None: ...
