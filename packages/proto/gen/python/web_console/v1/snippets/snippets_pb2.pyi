from google.protobuf.internal import containers as _containers
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Iterable as _Iterable, Mapping as _Mapping
from typing import ClassVar as _ClassVar, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class Snippet(_message.Message):
    __slots__ = ("id", "name", "body", "color", "pinned", "use_count", "last_used_at", "sort_order", "created_at", "updated_at")
    ID_FIELD_NUMBER: _ClassVar[int]
    NAME_FIELD_NUMBER: _ClassVar[int]
    BODY_FIELD_NUMBER: _ClassVar[int]
    COLOR_FIELD_NUMBER: _ClassVar[int]
    PINNED_FIELD_NUMBER: _ClassVar[int]
    USE_COUNT_FIELD_NUMBER: _ClassVar[int]
    LAST_USED_AT_FIELD_NUMBER: _ClassVar[int]
    SORT_ORDER_FIELD_NUMBER: _ClassVar[int]
    CREATED_AT_FIELD_NUMBER: _ClassVar[int]
    UPDATED_AT_FIELD_NUMBER: _ClassVar[int]
    id: str
    name: str
    body: str
    color: str
    pinned: bool
    use_count: int
    last_used_at: str
    sort_order: int
    created_at: str
    updated_at: str
    def __init__(self, id: _Optional[str] = ..., name: _Optional[str] = ..., body: _Optional[str] = ..., color: _Optional[str] = ..., pinned: _Optional[bool] = ..., use_count: _Optional[int] = ..., last_used_at: _Optional[str] = ..., sort_order: _Optional[int] = ..., created_at: _Optional[str] = ..., updated_at: _Optional[str] = ...) -> None: ...

class ListSnippetsRequest(_message.Message):
    __slots__ = ()
    def __init__(self) -> None: ...

class ListSnippetsResponse(_message.Message):
    __slots__ = ("snippets",)
    SNIPPETS_FIELD_NUMBER: _ClassVar[int]
    snippets: _containers.RepeatedCompositeFieldContainer[Snippet]
    def __init__(self, snippets: _Optional[_Iterable[_Union[Snippet, _Mapping]]] = ...) -> None: ...

class UpsertSnippetRequest(_message.Message):
    __slots__ = ("id", "name", "body", "color", "pinned", "has_pinned", "sort_order")
    ID_FIELD_NUMBER: _ClassVar[int]
    NAME_FIELD_NUMBER: _ClassVar[int]
    BODY_FIELD_NUMBER: _ClassVar[int]
    COLOR_FIELD_NUMBER: _ClassVar[int]
    PINNED_FIELD_NUMBER: _ClassVar[int]
    HAS_PINNED_FIELD_NUMBER: _ClassVar[int]
    SORT_ORDER_FIELD_NUMBER: _ClassVar[int]
    id: str
    name: str
    body: str
    color: str
    pinned: bool
    has_pinned: bool
    sort_order: int
    def __init__(self, id: _Optional[str] = ..., name: _Optional[str] = ..., body: _Optional[str] = ..., color: _Optional[str] = ..., pinned: _Optional[bool] = ..., has_pinned: _Optional[bool] = ..., sort_order: _Optional[int] = ...) -> None: ...

class UpsertSnippetResponse(_message.Message):
    __slots__ = ("snippet",)
    SNIPPET_FIELD_NUMBER: _ClassVar[int]
    snippet: Snippet
    def __init__(self, snippet: _Optional[_Union[Snippet, _Mapping]] = ...) -> None: ...

class DeleteSnippetRequest(_message.Message):
    __slots__ = ("id",)
    ID_FIELD_NUMBER: _ClassVar[int]
    id: str
    def __init__(self, id: _Optional[str] = ...) -> None: ...

class DeleteSnippetResponse(_message.Message):
    __slots__ = ("deleted",)
    DELETED_FIELD_NUMBER: _ClassVar[int]
    deleted: bool
    def __init__(self, deleted: _Optional[bool] = ...) -> None: ...

class TouchSnippetRequest(_message.Message):
    __slots__ = ("id",)
    ID_FIELD_NUMBER: _ClassVar[int]
    id: str
    def __init__(self, id: _Optional[str] = ...) -> None: ...

class TouchSnippetResponse(_message.Message):
    __slots__ = ("snippet",)
    SNIPPET_FIELD_NUMBER: _ClassVar[int]
    snippet: Snippet
    def __init__(self, snippet: _Optional[_Union[Snippet, _Mapping]] = ...) -> None: ...

class PromoteSnippetRequest(_message.Message):
    __slots__ = ("id",)
    ID_FIELD_NUMBER: _ClassVar[int]
    id: str
    def __init__(self, id: _Optional[str] = ...) -> None: ...

class PromoteSnippetResponse(_message.Message):
    __slots__ = ("identifier",)
    IDENTIFIER_FIELD_NUMBER: _ClassVar[int]
    identifier: str
    def __init__(self, identifier: _Optional[str] = ...) -> None: ...
