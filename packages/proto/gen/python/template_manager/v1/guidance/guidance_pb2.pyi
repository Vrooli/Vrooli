from google.protobuf.internal import containers as _containers
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Iterable as _Iterable, Mapping as _Mapping
from typing import ClassVar as _ClassVar, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class NextGateRequest(_message.Message):
    __slots__ = ("scenario",)
    SCENARIO_FIELD_NUMBER: _ClassVar[int]
    scenario: str
    def __init__(self, scenario: _Optional[str] = ...) -> None: ...

class NextGateResponse(_message.Message):
    __slots__ = ("scenario", "finalized", "complete", "finalize_required", "completed", "required", "gate", "message")
    SCENARIO_FIELD_NUMBER: _ClassVar[int]
    FINALIZED_FIELD_NUMBER: _ClassVar[int]
    COMPLETE_FIELD_NUMBER: _ClassVar[int]
    FINALIZE_REQUIRED_FIELD_NUMBER: _ClassVar[int]
    COMPLETED_FIELD_NUMBER: _ClassVar[int]
    REQUIRED_FIELD_NUMBER: _ClassVar[int]
    GATE_FIELD_NUMBER: _ClassVar[int]
    MESSAGE_FIELD_NUMBER: _ClassVar[int]
    scenario: str
    finalized: bool
    complete: bool
    finalize_required: bool
    completed: int
    required: int
    gate: OrientationGate
    message: str
    def __init__(self, scenario: _Optional[str] = ..., finalized: _Optional[bool] = ..., complete: _Optional[bool] = ..., finalize_required: _Optional[bool] = ..., completed: _Optional[int] = ..., required: _Optional[int] = ..., gate: _Optional[_Union[OrientationGate, _Mapping]] = ..., message: _Optional[str] = ...) -> None: ...

class OrientationGate(_message.Message):
    __slots__ = ("id", "title", "description", "required", "complete", "docs", "checks", "remediation")
    ID_FIELD_NUMBER: _ClassVar[int]
    TITLE_FIELD_NUMBER: _ClassVar[int]
    DESCRIPTION_FIELD_NUMBER: _ClassVar[int]
    REQUIRED_FIELD_NUMBER: _ClassVar[int]
    COMPLETE_FIELD_NUMBER: _ClassVar[int]
    DOCS_FIELD_NUMBER: _ClassVar[int]
    CHECKS_FIELD_NUMBER: _ClassVar[int]
    REMEDIATION_FIELD_NUMBER: _ClassVar[int]
    id: str
    title: str
    description: str
    required: bool
    complete: bool
    docs: _containers.RepeatedScalarFieldContainer[str]
    checks: _containers.RepeatedCompositeFieldContainer[OrientationCheck]
    remediation: _containers.RepeatedScalarFieldContainer[str]
    def __init__(self, id: _Optional[str] = ..., title: _Optional[str] = ..., description: _Optional[str] = ..., required: _Optional[bool] = ..., complete: _Optional[bool] = ..., docs: _Optional[_Iterable[str]] = ..., checks: _Optional[_Iterable[_Union[OrientationCheck, _Mapping]]] = ..., remediation: _Optional[_Iterable[str]] = ...) -> None: ...

class OrientationCheck(_message.Message):
    __slots__ = ("kind", "label", "passed", "skipped", "optional", "message")
    KIND_FIELD_NUMBER: _ClassVar[int]
    LABEL_FIELD_NUMBER: _ClassVar[int]
    PASSED_FIELD_NUMBER: _ClassVar[int]
    SKIPPED_FIELD_NUMBER: _ClassVar[int]
    OPTIONAL_FIELD_NUMBER: _ClassVar[int]
    MESSAGE_FIELD_NUMBER: _ClassVar[int]
    kind: str
    label: str
    passed: bool
    skipped: bool
    optional: bool
    message: str
    def __init__(self, kind: _Optional[str] = ..., label: _Optional[str] = ..., passed: _Optional[bool] = ..., skipped: _Optional[bool] = ..., optional: _Optional[bool] = ..., message: _Optional[str] = ...) -> None: ...
