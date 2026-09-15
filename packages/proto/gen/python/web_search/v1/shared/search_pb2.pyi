from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from typing import ClassVar as _ClassVar, Optional as _Optional

DESCRIPTOR: _descriptor.FileDescriptor

class SearchResult(_message.Message):
    __slots__ = ("url", "title", "snippet", "engine", "score", "category")
    URL_FIELD_NUMBER: _ClassVar[int]
    TITLE_FIELD_NUMBER: _ClassVar[int]
    SNIPPET_FIELD_NUMBER: _ClassVar[int]
    ENGINE_FIELD_NUMBER: _ClassVar[int]
    SCORE_FIELD_NUMBER: _ClassVar[int]
    CATEGORY_FIELD_NUMBER: _ClassVar[int]
    url: str
    title: str
    snippet: str
    engine: str
    score: float
    category: str
    def __init__(self, url: _Optional[str] = ..., title: _Optional[str] = ..., snippet: _Optional[str] = ..., engine: _Optional[str] = ..., score: _Optional[float] = ..., category: _Optional[str] = ...) -> None: ...

class EngineIssue(_message.Message):
    __slots__ = ("engine", "reason")
    ENGINE_FIELD_NUMBER: _ClassVar[int]
    REASON_FIELD_NUMBER: _ClassVar[int]
    engine: str
    reason: str
    def __init__(self, engine: _Optional[str] = ..., reason: _Optional[str] = ...) -> None: ...
