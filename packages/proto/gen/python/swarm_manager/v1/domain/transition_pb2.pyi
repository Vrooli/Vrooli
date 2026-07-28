from google.protobuf.internal import containers as _containers
from google.protobuf.internal import enum_type_wrapper as _enum_type_wrapper
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Iterable as _Iterable, Mapping as _Mapping
from typing import ClassVar as _ClassVar, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class TransitionKind(int, metaclass=_enum_type_wrapper.EnumTypeWrapper):
    __slots__ = ()
    TRANSITION_KIND_UNSPECIFIED: _ClassVar[TransitionKind]
    TRANSITION_KIND_SESSION: _ClassVar[TransitionKind]
    TRANSITION_KIND_WORKFLOW: _ClassVar[TransitionKind]
    TRANSITION_KIND_DETERMINISTIC: _ClassVar[TransitionKind]
TRANSITION_KIND_UNSPECIFIED: TransitionKind
TRANSITION_KIND_SESSION: TransitionKind
TRANSITION_KIND_WORKFLOW: TransitionKind
TRANSITION_KIND_DETERMINISTIC: TransitionKind

class WorkflowLocator(_message.Message):
    __slots__ = ("owner", "key")
    OWNER_FIELD_NUMBER: _ClassVar[int]
    KEY_FIELD_NUMBER: _ClassVar[int]
    owner: str
    key: str
    def __init__(self, owner: _Optional[str] = ..., key: _Optional[str] = ...) -> None: ...

class ExecutionStrategy(_message.Message):
    __slots__ = ("id", "workflow_key", "display_name", "description", "when_to_use", "cost_band")
    ID_FIELD_NUMBER: _ClassVar[int]
    WORKFLOW_KEY_FIELD_NUMBER: _ClassVar[int]
    DISPLAY_NAME_FIELD_NUMBER: _ClassVar[int]
    DESCRIPTION_FIELD_NUMBER: _ClassVar[int]
    WHEN_TO_USE_FIELD_NUMBER: _ClassVar[int]
    COST_BAND_FIELD_NUMBER: _ClassVar[int]
    id: str
    workflow_key: str
    display_name: str
    description: str
    when_to_use: str
    cost_band: str
    def __init__(self, id: _Optional[str] = ..., workflow_key: _Optional[str] = ..., display_name: _Optional[str] = ..., description: _Optional[str] = ..., when_to_use: _Optional[str] = ..., cost_band: _Optional[str] = ...) -> None: ...

class Transition(_message.Message):
    __slots__ = ("key", "subject", "kind", "workflow", "requires", "input_contract", "terminal_outcomes", "apply_action", "strategies")
    KEY_FIELD_NUMBER: _ClassVar[int]
    SUBJECT_FIELD_NUMBER: _ClassVar[int]
    KIND_FIELD_NUMBER: _ClassVar[int]
    WORKFLOW_FIELD_NUMBER: _ClassVar[int]
    REQUIRES_FIELD_NUMBER: _ClassVar[int]
    INPUT_CONTRACT_FIELD_NUMBER: _ClassVar[int]
    TERMINAL_OUTCOMES_FIELD_NUMBER: _ClassVar[int]
    APPLY_ACTION_FIELD_NUMBER: _ClassVar[int]
    STRATEGIES_FIELD_NUMBER: _ClassVar[int]
    key: str
    subject: str
    kind: TransitionKind
    workflow: WorkflowLocator
    requires: _containers.RepeatedScalarFieldContainer[str]
    input_contract: str
    terminal_outcomes: _containers.RepeatedScalarFieldContainer[str]
    apply_action: str
    strategies: _containers.RepeatedCompositeFieldContainer[ExecutionStrategy]
    def __init__(self, key: _Optional[str] = ..., subject: _Optional[str] = ..., kind: _Optional[_Union[TransitionKind, str]] = ..., workflow: _Optional[_Union[WorkflowLocator, _Mapping]] = ..., requires: _Optional[_Iterable[str]] = ..., input_contract: _Optional[str] = ..., terminal_outcomes: _Optional[_Iterable[str]] = ..., apply_action: _Optional[str] = ..., strategies: _Optional[_Iterable[_Union[ExecutionStrategy, _Mapping]]] = ...) -> None: ...
