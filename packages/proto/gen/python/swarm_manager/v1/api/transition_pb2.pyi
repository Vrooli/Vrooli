from swarm_manager.v1.domain import transition_pb2 as _transition_pb2
from google.protobuf.internal import containers as _containers
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Iterable as _Iterable, Mapping as _Mapping
from typing import ClassVar as _ClassVar, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class ListTransitionsRequest(_message.Message):
    __slots__ = ()
    def __init__(self) -> None: ...

class ListTransitionsResponse(_message.Message):
    __slots__ = ("transitions",)
    TRANSITIONS_FIELD_NUMBER: _ClassVar[int]
    transitions: _containers.RepeatedCompositeFieldContainer[_transition_pb2.Transition]
    def __init__(self, transitions: _Optional[_Iterable[_Union[_transition_pb2.Transition, _Mapping]]] = ...) -> None: ...

class StartTransitionRequest(_message.Message):
    __slots__ = ("transition_key", "subject_ref", "operator_inputs")
    class OperatorInputsEntry(_message.Message):
        __slots__ = ("key", "value")
        KEY_FIELD_NUMBER: _ClassVar[int]
        VALUE_FIELD_NUMBER: _ClassVar[int]
        key: str
        value: str
        def __init__(self, key: _Optional[str] = ..., value: _Optional[str] = ...) -> None: ...
    TRANSITION_KEY_FIELD_NUMBER: _ClassVar[int]
    SUBJECT_REF_FIELD_NUMBER: _ClassVar[int]
    OPERATOR_INPUTS_FIELD_NUMBER: _ClassVar[int]
    transition_key: str
    subject_ref: SubjectReference
    operator_inputs: _containers.ScalarMap[str, str]
    def __init__(self, transition_key: _Optional[str] = ..., subject_ref: _Optional[_Union[SubjectReference, _Mapping]] = ..., operator_inputs: _Optional[_Mapping[str, str]] = ...) -> None: ...

class SubjectReference(_message.Message):
    __slots__ = ("subject", "value")
    SUBJECT_FIELD_NUMBER: _ClassVar[int]
    VALUE_FIELD_NUMBER: _ClassVar[int]
    subject: str
    value: str
    def __init__(self, subject: _Optional[str] = ..., value: _Optional[str] = ...) -> None: ...

class StartTransitionResponse(_message.Message):
    __slots__ = ("execution_id", "definition_digest", "entity_version", "apply_state", "outcome", "terminal_code")
    EXECUTION_ID_FIELD_NUMBER: _ClassVar[int]
    DEFINITION_DIGEST_FIELD_NUMBER: _ClassVar[int]
    ENTITY_VERSION_FIELD_NUMBER: _ClassVar[int]
    APPLY_STATE_FIELD_NUMBER: _ClassVar[int]
    OUTCOME_FIELD_NUMBER: _ClassVar[int]
    TERMINAL_CODE_FIELD_NUMBER: _ClassVar[int]
    execution_id: str
    definition_digest: str
    entity_version: str
    apply_state: str
    outcome: str
    terminal_code: str
    def __init__(self, execution_id: _Optional[str] = ..., definition_digest: _Optional[str] = ..., entity_version: _Optional[str] = ..., apply_state: _Optional[str] = ..., outcome: _Optional[str] = ..., terminal_code: _Optional[str] = ...) -> None: ...

class ApplyTransitionRequest(_message.Message):
    __slots__ = ("transition_key", "execution_id")
    TRANSITION_KEY_FIELD_NUMBER: _ClassVar[int]
    EXECUTION_ID_FIELD_NUMBER: _ClassVar[int]
    transition_key: str
    execution_id: str
    def __init__(self, transition_key: _Optional[str] = ..., execution_id: _Optional[str] = ...) -> None: ...

class ApplyTransitionResponse(_message.Message):
    __slots__ = ("execution_id", "transition_key", "subject_ref", "outcome", "terminal_code", "applied_time", "definition_digest", "entity_version", "apply_state")
    EXECUTION_ID_FIELD_NUMBER: _ClassVar[int]
    TRANSITION_KEY_FIELD_NUMBER: _ClassVar[int]
    SUBJECT_REF_FIELD_NUMBER: _ClassVar[int]
    OUTCOME_FIELD_NUMBER: _ClassVar[int]
    TERMINAL_CODE_FIELD_NUMBER: _ClassVar[int]
    APPLIED_TIME_FIELD_NUMBER: _ClassVar[int]
    DEFINITION_DIGEST_FIELD_NUMBER: _ClassVar[int]
    ENTITY_VERSION_FIELD_NUMBER: _ClassVar[int]
    APPLY_STATE_FIELD_NUMBER: _ClassVar[int]
    execution_id: str
    transition_key: str
    subject_ref: str
    outcome: str
    terminal_code: str
    applied_time: str
    definition_digest: str
    entity_version: str
    apply_state: str
    def __init__(self, execution_id: _Optional[str] = ..., transition_key: _Optional[str] = ..., subject_ref: _Optional[str] = ..., outcome: _Optional[str] = ..., terminal_code: _Optional[str] = ..., applied_time: _Optional[str] = ..., definition_digest: _Optional[str] = ..., entity_version: _Optional[str] = ..., apply_state: _Optional[str] = ...) -> None: ...
