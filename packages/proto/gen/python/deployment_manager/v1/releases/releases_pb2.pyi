from deployment_manager.v1.releases import contracts_pb2 as _contracts_pb2
from buf.validate import validate_pb2 as _validate_pb2
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Mapping as _Mapping
from typing import ClassVar as _ClassVar, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class RegisterCandidateRequest(_message.Message):
    __slots__ = ("candidate",)
    CANDIDATE_FIELD_NUMBER: _ClassVar[int]
    candidate: _contracts_pb2.Candidate
    def __init__(self, candidate: _Optional[_Union[_contracts_pb2.Candidate, _Mapping]] = ...) -> None: ...

class RegisterCandidateResponse(_message.Message):
    __slots__ = ("candidate",)
    CANDIDATE_FIELD_NUMBER: _ClassVar[int]
    candidate: _contracts_pb2.CandidateRecord
    def __init__(self, candidate: _Optional[_Union[_contracts_pb2.CandidateRecord, _Mapping]] = ...) -> None: ...

class RegisterDestinationRevisionRequest(_message.Message):
    __slots__ = ("revision",)
    REVISION_FIELD_NUMBER: _ClassVar[int]
    revision: _contracts_pb2.DestinationRevision
    def __init__(self, revision: _Optional[_Union[_contracts_pb2.DestinationRevision, _Mapping]] = ...) -> None: ...

class RegisterDestinationRevisionResponse(_message.Message):
    __slots__ = ("destination",)
    DESTINATION_FIELD_NUMBER: _ClassVar[int]
    destination: _contracts_pb2.DestinationRevisionRecord
    def __init__(self, destination: _Optional[_Union[_contracts_pb2.DestinationRevisionRecord, _Mapping]] = ...) -> None: ...

class RecordClientUpdateReceiptRequest(_message.Message):
    __slots__ = ("release_id", "receipt")
    RELEASE_ID_FIELD_NUMBER: _ClassVar[int]
    RECEIPT_FIELD_NUMBER: _ClassVar[int]
    release_id: str
    receipt: _contracts_pb2.ClientUpdateReceipt
    def __init__(self, release_id: _Optional[str] = ..., receipt: _Optional[_Union[_contracts_pb2.ClientUpdateReceipt, _Mapping]] = ...) -> None: ...

class RecordClientUpdateReceiptResponse(_message.Message):
    __slots__ = ("release_id", "accepted")
    RELEASE_ID_FIELD_NUMBER: _ClassVar[int]
    ACCEPTED_FIELD_NUMBER: _ClassVar[int]
    release_id: str
    accepted: bool
    def __init__(self, release_id: _Optional[str] = ..., accepted: _Optional[bool] = ...) -> None: ...
