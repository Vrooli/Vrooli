from google.protobuf.internal import containers as _containers
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Iterable as _Iterable, Mapping as _Mapping
from typing import ClassVar as _ClassVar, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class StartSessionRequest(_message.Message):
    __slots__ = ("scenario", "path", "reset")
    SCENARIO_FIELD_NUMBER: _ClassVar[int]
    PATH_FIELD_NUMBER: _ClassVar[int]
    RESET_FIELD_NUMBER: _ClassVar[int]
    scenario: str
    path: str
    reset: bool
    def __init__(self, scenario: _Optional[str] = ..., path: _Optional[str] = ..., reset: _Optional[bool] = ...) -> None: ...

class SubmitAnswersRequest(_message.Message):
    __slots__ = ("session_id", "answers")
    SESSION_ID_FIELD_NUMBER: _ClassVar[int]
    ANSWERS_FIELD_NUMBER: _ClassVar[int]
    session_id: str
    answers: _containers.RepeatedCompositeFieldContainer[Answer]
    def __init__(self, session_id: _Optional[str] = ..., answers: _Optional[_Iterable[_Union[Answer, _Mapping]]] = ...) -> None: ...

class SessionState(_message.Message):
    __slots__ = ("session_id", "scenario", "questions", "answers", "remaining", "complete", "hints")
    class AnswersEntry(_message.Message):
        __slots__ = ("key", "value")
        KEY_FIELD_NUMBER: _ClassVar[int]
        VALUE_FIELD_NUMBER: _ClassVar[int]
        key: str
        value: Answer
        def __init__(self, key: _Optional[str] = ..., value: _Optional[_Union[Answer, _Mapping]] = ...) -> None: ...
    SESSION_ID_FIELD_NUMBER: _ClassVar[int]
    SCENARIO_FIELD_NUMBER: _ClassVar[int]
    QUESTIONS_FIELD_NUMBER: _ClassVar[int]
    ANSWERS_FIELD_NUMBER: _ClassVar[int]
    REMAINING_FIELD_NUMBER: _ClassVar[int]
    COMPLETE_FIELD_NUMBER: _ClassVar[int]
    HINTS_FIELD_NUMBER: _ClassVar[int]
    session_id: str
    scenario: str
    questions: _containers.RepeatedCompositeFieldContainer[Question]
    answers: _containers.MessageMap[str, Answer]
    remaining: _containers.RepeatedScalarFieldContainer[str]
    complete: bool
    hints: _containers.RepeatedCompositeFieldContainer[CapabilityHint]
    def __init__(self, session_id: _Optional[str] = ..., scenario: _Optional[str] = ..., questions: _Optional[_Iterable[_Union[Question, _Mapping]]] = ..., answers: _Optional[_Mapping[str, Answer]] = ..., remaining: _Optional[_Iterable[str]] = ..., complete: _Optional[bool] = ..., hints: _Optional[_Iterable[_Union[CapabilityHint, _Mapping]]] = ...) -> None: ...

class Question(_message.Message):
    __slots__ = ("id", "target", "prompt", "help", "kind", "required", "min_entries")
    ID_FIELD_NUMBER: _ClassVar[int]
    TARGET_FIELD_NUMBER: _ClassVar[int]
    PROMPT_FIELD_NUMBER: _ClassVar[int]
    HELP_FIELD_NUMBER: _ClassVar[int]
    KIND_FIELD_NUMBER: _ClassVar[int]
    REQUIRED_FIELD_NUMBER: _ClassVar[int]
    MIN_ENTRIES_FIELD_NUMBER: _ClassVar[int]
    id: str
    target: str
    prompt: str
    help: str
    kind: str
    required: bool
    min_entries: int
    def __init__(self, id: _Optional[str] = ..., target: _Optional[str] = ..., prompt: _Optional[str] = ..., help: _Optional[str] = ..., kind: _Optional[str] = ..., required: _Optional[bool] = ..., min_entries: _Optional[int] = ...) -> None: ...

class Answer(_message.Message):
    __slots__ = ("question_id", "text", "items", "targets", "invalid_reason")
    QUESTION_ID_FIELD_NUMBER: _ClassVar[int]
    TEXT_FIELD_NUMBER: _ClassVar[int]
    ITEMS_FIELD_NUMBER: _ClassVar[int]
    TARGETS_FIELD_NUMBER: _ClassVar[int]
    INVALID_REASON_FIELD_NUMBER: _ClassVar[int]
    question_id: str
    text: str
    items: _containers.RepeatedScalarFieldContainer[str]
    targets: _containers.RepeatedCompositeFieldContainer[OperationalTargetAnswer]
    invalid_reason: str
    def __init__(self, question_id: _Optional[str] = ..., text: _Optional[str] = ..., items: _Optional[_Iterable[str]] = ..., targets: _Optional[_Iterable[_Union[OperationalTargetAnswer, _Mapping]]] = ..., invalid_reason: _Optional[str] = ...) -> None: ...

class OperationalTargetAnswer(_message.Message):
    __slots__ = ("title", "description")
    TITLE_FIELD_NUMBER: _ClassVar[int]
    DESCRIPTION_FIELD_NUMBER: _ClassVar[int]
    title: str
    description: str
    def __init__(self, title: _Optional[str] = ..., description: _Optional[str] = ...) -> None: ...

class CapabilityHint(_message.Message):
    __slots__ = ("scenario", "capability", "anchor", "score")
    SCENARIO_FIELD_NUMBER: _ClassVar[int]
    CAPABILITY_FIELD_NUMBER: _ClassVar[int]
    ANCHOR_FIELD_NUMBER: _ClassVar[int]
    SCORE_FIELD_NUMBER: _ClassVar[int]
    scenario: str
    capability: str
    anchor: str
    score: float
    def __init__(self, scenario: _Optional[str] = ..., capability: _Optional[str] = ..., anchor: _Optional[str] = ..., score: _Optional[float] = ...) -> None: ...

class PreviewScaffoldRequest(_message.Message):
    __slots__ = ("session_id",)
    SESSION_ID_FIELD_NUMBER: _ClassVar[int]
    session_id: str
    def __init__(self, session_id: _Optional[str] = ...) -> None: ...

class ScaffoldPreview(_message.Message):
    __slots__ = ("session_id", "files", "blocking")
    SESSION_ID_FIELD_NUMBER: _ClassVar[int]
    FILES_FIELD_NUMBER: _ClassVar[int]
    BLOCKING_FIELD_NUMBER: _ClassVar[int]
    session_id: str
    files: _containers.RepeatedCompositeFieldContainer[FileDiff]
    blocking: _containers.RepeatedScalarFieldContainer[str]
    def __init__(self, session_id: _Optional[str] = ..., files: _Optional[_Iterable[_Union[FileDiff, _Mapping]]] = ..., blocking: _Optional[_Iterable[str]] = ...) -> None: ...

class FileDiff(_message.Message):
    __slots__ = ("path", "before", "after")
    PATH_FIELD_NUMBER: _ClassVar[int]
    BEFORE_FIELD_NUMBER: _ClassVar[int]
    AFTER_FIELD_NUMBER: _ClassVar[int]
    path: str
    before: str
    after: str
    def __init__(self, path: _Optional[str] = ..., before: _Optional[str] = ..., after: _Optional[str] = ...) -> None: ...

class ApplyScaffoldRequest(_message.Message):
    __slots__ = ("session_id", "apply")
    SESSION_ID_FIELD_NUMBER: _ClassVar[int]
    APPLY_FIELD_NUMBER: _ClassVar[int]
    session_id: str
    apply: bool
    def __init__(self, session_id: _Optional[str] = ..., apply: _Optional[bool] = ...) -> None: ...

class ScaffoldResult(_message.Message):
    __slots__ = ("session_id", "written", "residual_findings")
    SESSION_ID_FIELD_NUMBER: _ClassVar[int]
    WRITTEN_FIELD_NUMBER: _ClassVar[int]
    RESIDUAL_FINDINGS_FIELD_NUMBER: _ClassVar[int]
    session_id: str
    written: _containers.RepeatedScalarFieldContainer[str]
    residual_findings: _containers.RepeatedScalarFieldContainer[str]
    def __init__(self, session_id: _Optional[str] = ..., written: _Optional[_Iterable[str]] = ..., residual_findings: _Optional[_Iterable[str]] = ...) -> None: ...
