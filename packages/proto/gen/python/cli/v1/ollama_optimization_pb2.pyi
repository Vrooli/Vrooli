from google.protobuf.internal import containers as _containers
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Iterable as _Iterable
from typing import ClassVar as _ClassVar, Optional as _Optional

DESCRIPTOR: _descriptor.FileDescriptor

class InspectOllamaRoleCandidatesRequest(_message.Message):
    __slots__ = ("request_id", "target", "role", "candidate_models", "fixed_suite_id", "fixed_case_ids", "evaluation_receipt_id", "winner_model")
    REQUEST_ID_FIELD_NUMBER: _ClassVar[int]
    TARGET_FIELD_NUMBER: _ClassVar[int]
    ROLE_FIELD_NUMBER: _ClassVar[int]
    CANDIDATE_MODELS_FIELD_NUMBER: _ClassVar[int]
    FIXED_SUITE_ID_FIELD_NUMBER: _ClassVar[int]
    FIXED_CASE_IDS_FIELD_NUMBER: _ClassVar[int]
    EVALUATION_RECEIPT_ID_FIELD_NUMBER: _ClassVar[int]
    WINNER_MODEL_FIELD_NUMBER: _ClassVar[int]
    request_id: str
    target: str
    role: str
    candidate_models: _containers.RepeatedScalarFieldContainer[str]
    fixed_suite_id: str
    fixed_case_ids: _containers.RepeatedScalarFieldContainer[str]
    evaluation_receipt_id: str
    winner_model: str
    def __init__(self, request_id: _Optional[str] = ..., target: _Optional[str] = ..., role: _Optional[str] = ..., candidate_models: _Optional[_Iterable[str]] = ..., fixed_suite_id: _Optional[str] = ..., fixed_case_ids: _Optional[_Iterable[str]] = ..., evaluation_receipt_id: _Optional[str] = ..., winner_model: _Optional[str] = ...) -> None: ...

class InspectOllamaRoleCandidatesResponse(_message.Message):
    __slots__ = ("receipt_json",)
    RECEIPT_JSON_FIELD_NUMBER: _ClassVar[int]
    receipt_json: str
    def __init__(self, receipt_json: _Optional[str] = ...) -> None: ...
