from google.protobuf.internal import containers as _containers
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Iterable as _Iterable, Mapping as _Mapping
from typing import ClassVar as _ClassVar, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class MatchRange(_message.Message):
    __slots__ = ("start", "end")
    START_FIELD_NUMBER: _ClassVar[int]
    END_FIELD_NUMBER: _ClassVar[int]
    start: int
    end: int
    def __init__(self, start: _Optional[int] = ..., end: _Optional[int] = ...) -> None: ...

class SearchSkillsRequest(_message.Message):
    __slots__ = ("query", "tag", "folder")
    QUERY_FIELD_NUMBER: _ClassVar[int]
    TAG_FIELD_NUMBER: _ClassVar[int]
    FOLDER_FIELD_NUMBER: _ClassVar[int]
    query: str
    tag: str
    folder: str
    def __init__(self, query: _Optional[str] = ..., tag: _Optional[str] = ..., folder: _Optional[str] = ...) -> None: ...

class SkillSearchResult(_message.Message):
    __slots__ = ("id", "name", "description", "content", "folder", "tags", "modes", "score", "highlight")
    ID_FIELD_NUMBER: _ClassVar[int]
    NAME_FIELD_NUMBER: _ClassVar[int]
    DESCRIPTION_FIELD_NUMBER: _ClassVar[int]
    CONTENT_FIELD_NUMBER: _ClassVar[int]
    FOLDER_FIELD_NUMBER: _ClassVar[int]
    TAGS_FIELD_NUMBER: _ClassVar[int]
    MODES_FIELD_NUMBER: _ClassVar[int]
    SCORE_FIELD_NUMBER: _ClassVar[int]
    HIGHLIGHT_FIELD_NUMBER: _ClassVar[int]
    id: str
    name: str
    description: str
    content: str
    folder: str
    tags: _containers.RepeatedScalarFieldContainer[str]
    modes: _containers.RepeatedScalarFieldContainer[str]
    score: float
    highlight: str
    def __init__(self, id: _Optional[str] = ..., name: _Optional[str] = ..., description: _Optional[str] = ..., content: _Optional[str] = ..., folder: _Optional[str] = ..., tags: _Optional[_Iterable[str]] = ..., modes: _Optional[_Iterable[str]] = ..., score: _Optional[float] = ..., highlight: _Optional[str] = ...) -> None: ...

class SearchSkillsResponse(_message.Message):
    __slots__ = ("results", "total", "query")
    RESULTS_FIELD_NUMBER: _ClassVar[int]
    TOTAL_FIELD_NUMBER: _ClassVar[int]
    QUERY_FIELD_NUMBER: _ClassVar[int]
    results: _containers.RepeatedCompositeFieldContainer[SkillSearchResult]
    total: int
    query: str
    def __init__(self, results: _Optional[_Iterable[_Union[SkillSearchResult, _Mapping]]] = ..., total: _Optional[int] = ..., query: _Optional[str] = ...) -> None: ...

class SearchSkillContentRequest(_message.Message):
    __slots__ = ("query", "tags", "folders", "case_sensitive", "whole_word", "regex", "limit")
    QUERY_FIELD_NUMBER: _ClassVar[int]
    TAGS_FIELD_NUMBER: _ClassVar[int]
    FOLDERS_FIELD_NUMBER: _ClassVar[int]
    CASE_SENSITIVE_FIELD_NUMBER: _ClassVar[int]
    WHOLE_WORD_FIELD_NUMBER: _ClassVar[int]
    REGEX_FIELD_NUMBER: _ClassVar[int]
    LIMIT_FIELD_NUMBER: _ClassVar[int]
    query: str
    tags: _containers.RepeatedScalarFieldContainer[str]
    folders: _containers.RepeatedScalarFieldContainer[str]
    case_sensitive: bool
    whole_word: bool
    regex: bool
    limit: int
    def __init__(self, query: _Optional[str] = ..., tags: _Optional[_Iterable[str]] = ..., folders: _Optional[_Iterable[str]] = ..., case_sensitive: _Optional[bool] = ..., whole_word: _Optional[bool] = ..., regex: _Optional[bool] = ..., limit: _Optional[int] = ...) -> None: ...

class SkillContentMatch(_message.Message):
    __slots__ = ("skill_id", "skill_name", "file", "folder", "line_number", "line", "match_ranges")
    SKILL_ID_FIELD_NUMBER: _ClassVar[int]
    SKILL_NAME_FIELD_NUMBER: _ClassVar[int]
    FILE_FIELD_NUMBER: _ClassVar[int]
    FOLDER_FIELD_NUMBER: _ClassVar[int]
    LINE_NUMBER_FIELD_NUMBER: _ClassVar[int]
    LINE_FIELD_NUMBER: _ClassVar[int]
    MATCH_RANGES_FIELD_NUMBER: _ClassVar[int]
    skill_id: str
    skill_name: str
    file: str
    folder: str
    line_number: int
    line: str
    match_ranges: _containers.RepeatedCompositeFieldContainer[MatchRange]
    def __init__(self, skill_id: _Optional[str] = ..., skill_name: _Optional[str] = ..., file: _Optional[str] = ..., folder: _Optional[str] = ..., line_number: _Optional[int] = ..., line: _Optional[str] = ..., match_ranges: _Optional[_Iterable[_Union[MatchRange, _Mapping]]] = ...) -> None: ...

class SearchSkillContentResponse(_message.Message):
    __slots__ = ("matches", "total", "query")
    MATCHES_FIELD_NUMBER: _ClassVar[int]
    TOTAL_FIELD_NUMBER: _ClassVar[int]
    QUERY_FIELD_NUMBER: _ClassVar[int]
    matches: _containers.RepeatedCompositeFieldContainer[SkillContentMatch]
    total: int
    query: str
    def __init__(self, matches: _Optional[_Iterable[_Union[SkillContentMatch, _Mapping]]] = ..., total: _Optional[int] = ..., query: _Optional[str] = ...) -> None: ...

class SearchAgentsRequest(_message.Message):
    __slots__ = ("query", "tag", "status")
    QUERY_FIELD_NUMBER: _ClassVar[int]
    TAG_FIELD_NUMBER: _ClassVar[int]
    STATUS_FIELD_NUMBER: _ClassVar[int]
    query: str
    tag: str
    status: str
    def __init__(self, query: _Optional[str] = ..., tag: _Optional[str] = ..., status: _Optional[str] = ...) -> None: ...

class AgentSearchResult(_message.Message):
    __slots__ = ("id", "display_name", "description", "status", "tags", "score", "highlight")
    ID_FIELD_NUMBER: _ClassVar[int]
    DISPLAY_NAME_FIELD_NUMBER: _ClassVar[int]
    DESCRIPTION_FIELD_NUMBER: _ClassVar[int]
    STATUS_FIELD_NUMBER: _ClassVar[int]
    TAGS_FIELD_NUMBER: _ClassVar[int]
    SCORE_FIELD_NUMBER: _ClassVar[int]
    HIGHLIGHT_FIELD_NUMBER: _ClassVar[int]
    id: str
    display_name: str
    description: str
    status: str
    tags: _containers.RepeatedScalarFieldContainer[str]
    score: float
    highlight: str
    def __init__(self, id: _Optional[str] = ..., display_name: _Optional[str] = ..., description: _Optional[str] = ..., status: _Optional[str] = ..., tags: _Optional[_Iterable[str]] = ..., score: _Optional[float] = ..., highlight: _Optional[str] = ...) -> None: ...

class SearchAgentsResponse(_message.Message):
    __slots__ = ("results", "total", "query")
    RESULTS_FIELD_NUMBER: _ClassVar[int]
    TOTAL_FIELD_NUMBER: _ClassVar[int]
    QUERY_FIELD_NUMBER: _ClassVar[int]
    results: _containers.RepeatedCompositeFieldContainer[AgentSearchResult]
    total: int
    query: str
    def __init__(self, results: _Optional[_Iterable[_Union[AgentSearchResult, _Mapping]]] = ..., total: _Optional[int] = ..., query: _Optional[str] = ...) -> None: ...

class SearchAgentContentRequest(_message.Message):
    __slots__ = ("query", "tags", "case_sensitive", "whole_word", "regex", "limit")
    QUERY_FIELD_NUMBER: _ClassVar[int]
    TAGS_FIELD_NUMBER: _ClassVar[int]
    CASE_SENSITIVE_FIELD_NUMBER: _ClassVar[int]
    WHOLE_WORD_FIELD_NUMBER: _ClassVar[int]
    REGEX_FIELD_NUMBER: _ClassVar[int]
    LIMIT_FIELD_NUMBER: _ClassVar[int]
    query: str
    tags: _containers.RepeatedScalarFieldContainer[str]
    case_sensitive: bool
    whole_word: bool
    regex: bool
    limit: int
    def __init__(self, query: _Optional[str] = ..., tags: _Optional[_Iterable[str]] = ..., case_sensitive: _Optional[bool] = ..., whole_word: _Optional[bool] = ..., regex: _Optional[bool] = ..., limit: _Optional[int] = ...) -> None: ...

class AgentContentMatch(_message.Message):
    __slots__ = ("agent_id", "agent_name", "file", "line_number", "line", "match_ranges")
    AGENT_ID_FIELD_NUMBER: _ClassVar[int]
    AGENT_NAME_FIELD_NUMBER: _ClassVar[int]
    FILE_FIELD_NUMBER: _ClassVar[int]
    LINE_NUMBER_FIELD_NUMBER: _ClassVar[int]
    LINE_FIELD_NUMBER: _ClassVar[int]
    MATCH_RANGES_FIELD_NUMBER: _ClassVar[int]
    agent_id: str
    agent_name: str
    file: str
    line_number: int
    line: str
    match_ranges: _containers.RepeatedCompositeFieldContainer[MatchRange]
    def __init__(self, agent_id: _Optional[str] = ..., agent_name: _Optional[str] = ..., file: _Optional[str] = ..., line_number: _Optional[int] = ..., line: _Optional[str] = ..., match_ranges: _Optional[_Iterable[_Union[MatchRange, _Mapping]]] = ...) -> None: ...

class SearchAgentContentResponse(_message.Message):
    __slots__ = ("matches", "total", "query")
    MATCHES_FIELD_NUMBER: _ClassVar[int]
    TOTAL_FIELD_NUMBER: _ClassVar[int]
    QUERY_FIELD_NUMBER: _ClassVar[int]
    matches: _containers.RepeatedCompositeFieldContainer[AgentContentMatch]
    total: int
    query: str
    def __init__(self, matches: _Optional[_Iterable[_Union[AgentContentMatch, _Mapping]]] = ..., total: _Optional[int] = ..., query: _Optional[str] = ...) -> None: ...

class SearchTeamsRequest(_message.Message):
    __slots__ = ("query", "enabled")
    QUERY_FIELD_NUMBER: _ClassVar[int]
    ENABLED_FIELD_NUMBER: _ClassVar[int]
    query: str
    enabled: bool
    def __init__(self, query: _Optional[str] = ..., enabled: _Optional[bool] = ...) -> None: ...

class TeamSearchResult(_message.Message):
    __slots__ = ("id", "display_name", "mission", "enabled", "member_count", "score", "highlight")
    ID_FIELD_NUMBER: _ClassVar[int]
    DISPLAY_NAME_FIELD_NUMBER: _ClassVar[int]
    MISSION_FIELD_NUMBER: _ClassVar[int]
    ENABLED_FIELD_NUMBER: _ClassVar[int]
    MEMBER_COUNT_FIELD_NUMBER: _ClassVar[int]
    SCORE_FIELD_NUMBER: _ClassVar[int]
    HIGHLIGHT_FIELD_NUMBER: _ClassVar[int]
    id: str
    display_name: str
    mission: str
    enabled: bool
    member_count: int
    score: float
    highlight: str
    def __init__(self, id: _Optional[str] = ..., display_name: _Optional[str] = ..., mission: _Optional[str] = ..., enabled: _Optional[bool] = ..., member_count: _Optional[int] = ..., score: _Optional[float] = ..., highlight: _Optional[str] = ...) -> None: ...

class SearchTeamsResponse(_message.Message):
    __slots__ = ("results", "total", "query")
    RESULTS_FIELD_NUMBER: _ClassVar[int]
    TOTAL_FIELD_NUMBER: _ClassVar[int]
    QUERY_FIELD_NUMBER: _ClassVar[int]
    results: _containers.RepeatedCompositeFieldContainer[TeamSearchResult]
    total: int
    query: str
    def __init__(self, results: _Optional[_Iterable[_Union[TeamSearchResult, _Mapping]]] = ..., total: _Optional[int] = ..., query: _Optional[str] = ...) -> None: ...

class SearchTeamContentRequest(_message.Message):
    __slots__ = ("query", "case_sensitive", "whole_word", "regex", "limit")
    QUERY_FIELD_NUMBER: _ClassVar[int]
    CASE_SENSITIVE_FIELD_NUMBER: _ClassVar[int]
    WHOLE_WORD_FIELD_NUMBER: _ClassVar[int]
    REGEX_FIELD_NUMBER: _ClassVar[int]
    LIMIT_FIELD_NUMBER: _ClassVar[int]
    query: str
    case_sensitive: bool
    whole_word: bool
    regex: bool
    limit: int
    def __init__(self, query: _Optional[str] = ..., case_sensitive: _Optional[bool] = ..., whole_word: _Optional[bool] = ..., regex: _Optional[bool] = ..., limit: _Optional[int] = ...) -> None: ...

class TeamContentMatch(_message.Message):
    __slots__ = ("team_id", "team_name", "file", "line_number", "line", "match_ranges")
    TEAM_ID_FIELD_NUMBER: _ClassVar[int]
    TEAM_NAME_FIELD_NUMBER: _ClassVar[int]
    FILE_FIELD_NUMBER: _ClassVar[int]
    LINE_NUMBER_FIELD_NUMBER: _ClassVar[int]
    LINE_FIELD_NUMBER: _ClassVar[int]
    MATCH_RANGES_FIELD_NUMBER: _ClassVar[int]
    team_id: str
    team_name: str
    file: str
    line_number: int
    line: str
    match_ranges: _containers.RepeatedCompositeFieldContainer[MatchRange]
    def __init__(self, team_id: _Optional[str] = ..., team_name: _Optional[str] = ..., file: _Optional[str] = ..., line_number: _Optional[int] = ..., line: _Optional[str] = ..., match_ranges: _Optional[_Iterable[_Union[MatchRange, _Mapping]]] = ...) -> None: ...

class SearchTeamContentResponse(_message.Message):
    __slots__ = ("matches", "total", "query")
    MATCHES_FIELD_NUMBER: _ClassVar[int]
    TOTAL_FIELD_NUMBER: _ClassVar[int]
    QUERY_FIELD_NUMBER: _ClassVar[int]
    matches: _containers.RepeatedCompositeFieldContainer[TeamContentMatch]
    total: int
    query: str
    def __init__(self, matches: _Optional[_Iterable[_Union[TeamContentMatch, _Mapping]]] = ..., total: _Optional[int] = ..., query: _Optional[str] = ...) -> None: ...
