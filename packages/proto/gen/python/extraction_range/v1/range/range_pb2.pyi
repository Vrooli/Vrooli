from google.protobuf.internal import containers as _containers
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Iterable as _Iterable, Mapping as _Mapping
from typing import ClassVar as _ClassVar, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class LevelSummary(_message.Message):
    __slots__ = ("id", "family", "family_name", "difficulty", "title", "task", "tags")
    ID_FIELD_NUMBER: _ClassVar[int]
    FAMILY_FIELD_NUMBER: _ClassVar[int]
    FAMILY_NAME_FIELD_NUMBER: _ClassVar[int]
    DIFFICULTY_FIELD_NUMBER: _ClassVar[int]
    TITLE_FIELD_NUMBER: _ClassVar[int]
    TASK_FIELD_NUMBER: _ClassVar[int]
    TAGS_FIELD_NUMBER: _ClassVar[int]
    id: str
    family: str
    family_name: str
    difficulty: int
    title: str
    task: str
    tags: _containers.RepeatedScalarFieldContainer[str]
    def __init__(self, id: _Optional[str] = ..., family: _Optional[str] = ..., family_name: _Optional[str] = ..., difficulty: _Optional[int] = ..., title: _Optional[str] = ..., task: _Optional[str] = ..., tags: _Optional[_Iterable[str]] = ...) -> None: ...

class ListLevelsRequest(_message.Message):
    __slots__ = ("family",)
    FAMILY_FIELD_NUMBER: _ClassVar[int]
    family: str
    def __init__(self, family: _Optional[str] = ...) -> None: ...

class ListLevelsResponse(_message.Message):
    __slots__ = ("levels",)
    LEVELS_FIELD_NUMBER: _ClassVar[int]
    levels: _containers.RepeatedCompositeFieldContainer[LevelSummary]
    def __init__(self, levels: _Optional[_Iterable[_Union[LevelSummary, _Mapping]]] = ...) -> None: ...

class GetLevelRequest(_message.Message):
    __slots__ = ("id",)
    ID_FIELD_NUMBER: _ClassVar[int]
    id: str
    def __init__(self, id: _Optional[str] = ...) -> None: ...

class GetLevelResponse(_message.Message):
    __slots__ = ("level", "target_url")
    LEVEL_FIELD_NUMBER: _ClassVar[int]
    TARGET_URL_FIELD_NUMBER: _ClassVar[int]
    level: LevelSummary
    target_url: str
    def __init__(self, level: _Optional[_Union[LevelSummary, _Mapping]] = ..., target_url: _Optional[str] = ...) -> None: ...

class ExtractedRecord(_message.Message):
    __slots__ = ("id", "name", "category", "value", "active")
    ID_FIELD_NUMBER: _ClassVar[int]
    NAME_FIELD_NUMBER: _ClassVar[int]
    CATEGORY_FIELD_NUMBER: _ClassVar[int]
    VALUE_FIELD_NUMBER: _ClassVar[int]
    ACTIVE_FIELD_NUMBER: _ClassVar[int]
    id: str
    name: str
    category: str
    value: int
    active: bool
    def __init__(self, id: _Optional[str] = ..., name: _Optional[str] = ..., category: _Optional[str] = ..., value: _Optional[int] = ..., active: _Optional[bool] = ...) -> None: ...

class SubmitRequest(_message.Message):
    __slots__ = ("tool", "level_id", "records")
    TOOL_FIELD_NUMBER: _ClassVar[int]
    LEVEL_ID_FIELD_NUMBER: _ClassVar[int]
    RECORDS_FIELD_NUMBER: _ClassVar[int]
    tool: str
    level_id: str
    records: _containers.RepeatedCompositeFieldContainer[ExtractedRecord]
    def __init__(self, tool: _Optional[str] = ..., level_id: _Optional[str] = ..., records: _Optional[_Iterable[_Union[ExtractedRecord, _Mapping]]] = ...) -> None: ...

class SubmitResponse(_message.Message):
    __slots__ = ("tool", "level_id", "family", "correct", "submitted", "expected", "precision", "recall", "f1", "passed")
    TOOL_FIELD_NUMBER: _ClassVar[int]
    LEVEL_ID_FIELD_NUMBER: _ClassVar[int]
    FAMILY_FIELD_NUMBER: _ClassVar[int]
    CORRECT_FIELD_NUMBER: _ClassVar[int]
    SUBMITTED_FIELD_NUMBER: _ClassVar[int]
    EXPECTED_FIELD_NUMBER: _ClassVar[int]
    PRECISION_FIELD_NUMBER: _ClassVar[int]
    RECALL_FIELD_NUMBER: _ClassVar[int]
    F1_FIELD_NUMBER: _ClassVar[int]
    PASSED_FIELD_NUMBER: _ClassVar[int]
    tool: str
    level_id: str
    family: str
    correct: int
    submitted: int
    expected: int
    precision: float
    recall: float
    f1: float
    passed: bool
    def __init__(self, tool: _Optional[str] = ..., level_id: _Optional[str] = ..., family: _Optional[str] = ..., correct: _Optional[int] = ..., submitted: _Optional[int] = ..., expected: _Optional[int] = ..., precision: _Optional[float] = ..., recall: _Optional[float] = ..., f1: _Optional[float] = ..., passed: _Optional[bool] = ...) -> None: ...

class ScoreboardRequest(_message.Message):
    __slots__ = ("tool", "family")
    TOOL_FIELD_NUMBER: _ClassVar[int]
    FAMILY_FIELD_NUMBER: _ClassVar[int]
    tool: str
    family: str
    def __init__(self, tool: _Optional[str] = ..., family: _Optional[str] = ...) -> None: ...

class ScoreboardResponse(_message.Message):
    __slots__ = ("scores",)
    SCORES_FIELD_NUMBER: _ClassVar[int]
    scores: _containers.RepeatedCompositeFieldContainer[SubmitResponse]
    def __init__(self, scores: _Optional[_Iterable[_Union[SubmitResponse, _Mapping]]] = ...) -> None: ...
