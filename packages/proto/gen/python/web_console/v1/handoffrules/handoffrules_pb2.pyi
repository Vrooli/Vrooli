from google.protobuf.internal import containers as _containers
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Iterable as _Iterable, Mapping as _Mapping
from typing import ClassVar as _ClassVar, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class Rule(_message.Message):
    __slots__ = ("id", "name", "enabled", "source", "pattern", "surfaces", "sort_order", "created_at", "updated_at")
    ID_FIELD_NUMBER: _ClassVar[int]
    NAME_FIELD_NUMBER: _ClassVar[int]
    ENABLED_FIELD_NUMBER: _ClassVar[int]
    SOURCE_FIELD_NUMBER: _ClassVar[int]
    PATTERN_FIELD_NUMBER: _ClassVar[int]
    SURFACES_FIELD_NUMBER: _ClassVar[int]
    SORT_ORDER_FIELD_NUMBER: _ClassVar[int]
    CREATED_AT_FIELD_NUMBER: _ClassVar[int]
    UPDATED_AT_FIELD_NUMBER: _ClassVar[int]
    id: str
    name: str
    enabled: bool
    source: str
    pattern: str
    surfaces: _containers.RepeatedScalarFieldContainer[str]
    sort_order: int
    created_at: str
    updated_at: str
    def __init__(self, id: _Optional[str] = ..., name: _Optional[str] = ..., enabled: _Optional[bool] = ..., source: _Optional[str] = ..., pattern: _Optional[str] = ..., surfaces: _Optional[_Iterable[str]] = ..., sort_order: _Optional[int] = ..., created_at: _Optional[str] = ..., updated_at: _Optional[str] = ...) -> None: ...

class ListRulesRequest(_message.Message):
    __slots__ = ()
    def __init__(self) -> None: ...

class ListRulesResponse(_message.Message):
    __slots__ = ("rules",)
    RULES_FIELD_NUMBER: _ClassVar[int]
    rules: _containers.RepeatedCompositeFieldContainer[Rule]
    def __init__(self, rules: _Optional[_Iterable[_Union[Rule, _Mapping]]] = ...) -> None: ...

class UpsertRuleRequest(_message.Message):
    __slots__ = ("id", "name", "enabled", "source", "pattern", "surfaces", "sort_order")
    ID_FIELD_NUMBER: _ClassVar[int]
    NAME_FIELD_NUMBER: _ClassVar[int]
    ENABLED_FIELD_NUMBER: _ClassVar[int]
    SOURCE_FIELD_NUMBER: _ClassVar[int]
    PATTERN_FIELD_NUMBER: _ClassVar[int]
    SURFACES_FIELD_NUMBER: _ClassVar[int]
    SORT_ORDER_FIELD_NUMBER: _ClassVar[int]
    id: str
    name: str
    enabled: bool
    source: str
    pattern: str
    surfaces: _containers.RepeatedScalarFieldContainer[str]
    sort_order: int
    def __init__(self, id: _Optional[str] = ..., name: _Optional[str] = ..., enabled: _Optional[bool] = ..., source: _Optional[str] = ..., pattern: _Optional[str] = ..., surfaces: _Optional[_Iterable[str]] = ..., sort_order: _Optional[int] = ...) -> None: ...

class UpsertRuleResponse(_message.Message):
    __slots__ = ("rule",)
    RULE_FIELD_NUMBER: _ClassVar[int]
    rule: Rule
    def __init__(self, rule: _Optional[_Union[Rule, _Mapping]] = ...) -> None: ...

class DeleteRuleRequest(_message.Message):
    __slots__ = ("id",)
    ID_FIELD_NUMBER: _ClassVar[int]
    id: str
    def __init__(self, id: _Optional[str] = ...) -> None: ...

class DeleteRuleResponse(_message.Message):
    __slots__ = ()
    def __init__(self) -> None: ...
