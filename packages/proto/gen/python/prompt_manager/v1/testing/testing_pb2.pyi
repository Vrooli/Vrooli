from google.protobuf.internal import containers as _containers
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Iterable as _Iterable, Mapping as _Mapping
from typing import ClassVar as _ClassVar, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class RunSkillTestRequest(_message.Message):
    __slots__ = ("skill_id", "role", "variables", "max_tokens", "temperature")
    class VariablesEntry(_message.Message):
        __slots__ = ("key", "value")
        KEY_FIELD_NUMBER: _ClassVar[int]
        VALUE_FIELD_NUMBER: _ClassVar[int]
        key: str
        value: str
        def __init__(self, key: _Optional[str] = ..., value: _Optional[str] = ...) -> None: ...
    SKILL_ID_FIELD_NUMBER: _ClassVar[int]
    ROLE_FIELD_NUMBER: _ClassVar[int]
    VARIABLES_FIELD_NUMBER: _ClassVar[int]
    MAX_TOKENS_FIELD_NUMBER: _ClassVar[int]
    TEMPERATURE_FIELD_NUMBER: _ClassVar[int]
    skill_id: str
    role: str
    variables: _containers.ScalarMap[str, str]
    max_tokens: int
    temperature: float
    def __init__(self, skill_id: _Optional[str] = ..., role: _Optional[str] = ..., variables: _Optional[_Mapping[str, str]] = ..., max_tokens: _Optional[int] = ..., temperature: _Optional[float] = ...) -> None: ...

class SkillTestResponse(_message.Message):
    __slots__ = ("test_id", "role", "response", "response_time", "token_count", "tested_at")
    TEST_ID_FIELD_NUMBER: _ClassVar[int]
    ROLE_FIELD_NUMBER: _ClassVar[int]
    RESPONSE_FIELD_NUMBER: _ClassVar[int]
    RESPONSE_TIME_FIELD_NUMBER: _ClassVar[int]
    TOKEN_COUNT_FIELD_NUMBER: _ClassVar[int]
    TESTED_AT_FIELD_NUMBER: _ClassVar[int]
    test_id: str
    role: str
    response: str
    response_time: float
    token_count: int
    tested_at: str
    def __init__(self, test_id: _Optional[str] = ..., role: _Optional[str] = ..., response: _Optional[str] = ..., response_time: _Optional[float] = ..., token_count: _Optional[int] = ..., tested_at: _Optional[str] = ...) -> None: ...

class SkillTestResult(_message.Message):
    __slots__ = ("id", "skill_id", "role", "input_variables", "response", "response_time", "token_count", "rating", "notes", "tested_at")
    ID_FIELD_NUMBER: _ClassVar[int]
    SKILL_ID_FIELD_NUMBER: _ClassVar[int]
    ROLE_FIELD_NUMBER: _ClassVar[int]
    INPUT_VARIABLES_FIELD_NUMBER: _ClassVar[int]
    RESPONSE_FIELD_NUMBER: _ClassVar[int]
    RESPONSE_TIME_FIELD_NUMBER: _ClassVar[int]
    TOKEN_COUNT_FIELD_NUMBER: _ClassVar[int]
    RATING_FIELD_NUMBER: _ClassVar[int]
    NOTES_FIELD_NUMBER: _ClassVar[int]
    TESTED_AT_FIELD_NUMBER: _ClassVar[int]
    id: str
    skill_id: str
    role: str
    input_variables: str
    response: str
    response_time: float
    token_count: int
    rating: int
    notes: str
    tested_at: str
    def __init__(self, id: _Optional[str] = ..., skill_id: _Optional[str] = ..., role: _Optional[str] = ..., input_variables: _Optional[str] = ..., response: _Optional[str] = ..., response_time: _Optional[float] = ..., token_count: _Optional[int] = ..., rating: _Optional[int] = ..., notes: _Optional[str] = ..., tested_at: _Optional[str] = ...) -> None: ...

class ListSkillTestHistoryRequest(_message.Message):
    __slots__ = ("skill_id", "limit")
    SKILL_ID_FIELD_NUMBER: _ClassVar[int]
    LIMIT_FIELD_NUMBER: _ClassVar[int]
    skill_id: str
    limit: int
    def __init__(self, skill_id: _Optional[str] = ..., limit: _Optional[int] = ...) -> None: ...

class ListSkillTestHistoryResponse(_message.Message):
    __slots__ = ("results",)
    RESULTS_FIELD_NUMBER: _ClassVar[int]
    results: _containers.RepeatedCompositeFieldContainer[SkillTestResult]
    def __init__(self, results: _Optional[_Iterable[_Union[SkillTestResult, _Mapping]]] = ...) -> None: ...
