from google.protobuf.internal import containers as _containers
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Iterable as _Iterable, Mapping as _Mapping
from typing import ClassVar as _ClassVar, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class Theme(_message.Message):
    __slots__ = ("id", "name", "tokens", "source")
    class TokensEntry(_message.Message):
        __slots__ = ("key", "value")
        KEY_FIELD_NUMBER: _ClassVar[int]
        VALUE_FIELD_NUMBER: _ClassVar[int]
        key: str
        value: str
        def __init__(self, key: _Optional[str] = ..., value: _Optional[str] = ...) -> None: ...
    ID_FIELD_NUMBER: _ClassVar[int]
    NAME_FIELD_NUMBER: _ClassVar[int]
    TOKENS_FIELD_NUMBER: _ClassVar[int]
    SOURCE_FIELD_NUMBER: _ClassVar[int]
    id: str
    name: str
    tokens: _containers.ScalarMap[str, str]
    source: str
    def __init__(self, id: _Optional[str] = ..., name: _Optional[str] = ..., tokens: _Optional[_Mapping[str, str]] = ..., source: _Optional[str] = ...) -> None: ...

class ListBuiltinThemesRequest(_message.Message):
    __slots__ = ()
    def __init__(self) -> None: ...

class ListBuiltinThemesResponse(_message.Message):
    __slots__ = ("themes",)
    THEMES_FIELD_NUMBER: _ClassVar[int]
    themes: _containers.RepeatedCompositeFieldContainer[Theme]
    def __init__(self, themes: _Optional[_Iterable[_Union[Theme, _Mapping]]] = ...) -> None: ...

class GetBuiltinThemeRequest(_message.Message):
    __slots__ = ("id",)
    ID_FIELD_NUMBER: _ClassVar[int]
    id: str
    def __init__(self, id: _Optional[str] = ...) -> None: ...

class GetBuiltinThemeResponse(_message.Message):
    __slots__ = ("theme",)
    THEME_FIELD_NUMBER: _ClassVar[int]
    theme: Theme
    def __init__(self, theme: _Optional[_Union[Theme, _Mapping]] = ...) -> None: ...

class GetThemeFromScenarioRequest(_message.Message):
    __slots__ = ("scenario_id",)
    SCENARIO_ID_FIELD_NUMBER: _ClassVar[int]
    scenario_id: str
    def __init__(self, scenario_id: _Optional[str] = ...) -> None: ...

class GetThemeFromScenarioResponse(_message.Message):
    __slots__ = ("theme",)
    THEME_FIELD_NUMBER: _ClassVar[int]
    theme: Theme
    def __init__(self, theme: _Optional[_Union[Theme, _Mapping]] = ...) -> None: ...
