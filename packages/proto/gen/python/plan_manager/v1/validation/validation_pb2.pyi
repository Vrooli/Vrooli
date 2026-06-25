from plan_manager.v1.shared import model_pb2 as _model_pb2
from google.protobuf.internal import containers as _containers
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Iterable as _Iterable, Mapping as _Mapping
from typing import ClassVar as _ClassVar, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class ResolveReferencesRequest(_message.Message):
    __slots__ = ("plan_id", "phase_id")
    PLAN_ID_FIELD_NUMBER: _ClassVar[int]
    PHASE_ID_FIELD_NUMBER: _ClassVar[int]
    plan_id: str
    phase_id: str
    def __init__(self, plan_id: _Optional[str] = ..., phase_id: _Optional[str] = ...) -> None: ...

class ResolveReferencesResponse(_message.Message):
    __slots__ = ("references", "degraded")
    REFERENCES_FIELD_NUMBER: _ClassVar[int]
    DEGRADED_FIELD_NUMBER: _ClassVar[int]
    references: _containers.RepeatedCompositeFieldContainer[_model_pb2.Reference]
    degraded: bool
    def __init__(self, references: _Optional[_Iterable[_Union[_model_pb2.Reference, _Mapping]]] = ..., degraded: _Optional[bool] = ...) -> None: ...

class ComputeStalenessRequest(_message.Message):
    __slots__ = ("plan_id", "phase_id")
    PLAN_ID_FIELD_NUMBER: _ClassVar[int]
    PHASE_ID_FIELD_NUMBER: _ClassVar[int]
    plan_id: str
    phase_id: str
    def __init__(self, plan_id: _Optional[str] = ..., phase_id: _Optional[str] = ...) -> None: ...

class ComputeStalenessResponse(_message.Message):
    __slots__ = ("overall", "references", "degraded")
    OVERALL_FIELD_NUMBER: _ClassVar[int]
    REFERENCES_FIELD_NUMBER: _ClassVar[int]
    DEGRADED_FIELD_NUMBER: _ClassVar[int]
    overall: _model_pb2.StalenessTier
    references: _containers.RepeatedCompositeFieldContainer[_model_pb2.Reference]
    degraded: bool
    def __init__(self, overall: _Optional[_Union[_model_pb2.StalenessTier, str]] = ..., references: _Optional[_Iterable[_Union[_model_pb2.Reference, _Mapping]]] = ..., degraded: _Optional[bool] = ...) -> None: ...

class DeriveBaselineScopeRequest(_message.Message):
    __slots__ = ("plan_id", "phase_id")
    PLAN_ID_FIELD_NUMBER: _ClassVar[int]
    PHASE_ID_FIELD_NUMBER: _ClassVar[int]
    plan_id: str
    phase_id: str
    def __init__(self, plan_id: _Optional[str] = ..., phase_id: _Optional[str] = ...) -> None: ...

class DeriveBaselineScopeResponse(_message.Message):
    __slots__ = ("commands", "locations")
    COMMANDS_FIELD_NUMBER: _ClassVar[int]
    LOCATIONS_FIELD_NUMBER: _ClassVar[int]
    commands: _containers.RepeatedScalarFieldContainer[str]
    locations: _containers.RepeatedScalarFieldContainer[str]
    def __init__(self, commands: _Optional[_Iterable[str]] = ..., locations: _Optional[_Iterable[str]] = ...) -> None: ...

class RunValidationRequest(_message.Message):
    __slots__ = ("plan_id", "phase_id")
    PLAN_ID_FIELD_NUMBER: _ClassVar[int]
    PHASE_ID_FIELD_NUMBER: _ClassVar[int]
    plan_id: str
    phase_id: str
    def __init__(self, plan_id: _Optional[str] = ..., phase_id: _Optional[str] = ...) -> None: ...

class RunValidationResponse(_message.Message):
    __slots__ = ("result",)
    RESULT_FIELD_NUMBER: _ClassVar[int]
    result: _model_pb2.ValidationResult
    def __init__(self, result: _Optional[_Union[_model_pb2.ValidationResult, _Mapping]] = ...) -> None: ...

class VerifyDefinitionOfDoneRequest(_message.Message):
    __slots__ = ("plan_id",)
    PLAN_ID_FIELD_NUMBER: _ClassVar[int]
    plan_id: str
    def __init__(self, plan_id: _Optional[str] = ...) -> None: ...

class VerifyDefinitionOfDoneResponse(_message.Message):
    __slots__ = ("result", "dod_met")
    RESULT_FIELD_NUMBER: _ClassVar[int]
    DOD_MET_FIELD_NUMBER: _ClassVar[int]
    result: _model_pb2.ValidationResult
    dod_met: bool
    def __init__(self, result: _Optional[_Union[_model_pb2.ValidationResult, _Mapping]] = ..., dod_met: _Optional[bool] = ...) -> None: ...
