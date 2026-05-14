import datetime

from google.protobuf import duration_pb2 as _duration_pb2
from google.protobuf.internal import containers as _containers
from google.protobuf.internal import enum_type_wrapper as _enum_type_wrapper
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Iterable as _Iterable, Mapping as _Mapping
from typing import ClassVar as _ClassVar, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class GetScreenRequest(_message.Message):
    __slots__ = ("session_id", "include_scrollback")
    SESSION_ID_FIELD_NUMBER: _ClassVar[int]
    INCLUDE_SCROLLBACK_FIELD_NUMBER: _ClassVar[int]
    session_id: str
    include_scrollback: bool
    def __init__(self, session_id: _Optional[str] = ..., include_scrollback: _Optional[bool] = ...) -> None: ...

class SGR(_message.Message):
    __slots__ = ("fg", "bg", "bold", "italic", "underline", "inverse", "faint")
    FG_FIELD_NUMBER: _ClassVar[int]
    BG_FIELD_NUMBER: _ClassVar[int]
    BOLD_FIELD_NUMBER: _ClassVar[int]
    ITALIC_FIELD_NUMBER: _ClassVar[int]
    UNDERLINE_FIELD_NUMBER: _ClassVar[int]
    INVERSE_FIELD_NUMBER: _ClassVar[int]
    FAINT_FIELD_NUMBER: _ClassVar[int]
    fg: int
    bg: int
    bold: bool
    italic: bool
    underline: bool
    inverse: bool
    faint: bool
    def __init__(self, fg: _Optional[int] = ..., bg: _Optional[int] = ..., bold: _Optional[bool] = ..., italic: _Optional[bool] = ..., underline: _Optional[bool] = ..., inverse: _Optional[bool] = ..., faint: _Optional[bool] = ...) -> None: ...

class Cell(_message.Message):
    __slots__ = ("rune", "sgr")
    RUNE_FIELD_NUMBER: _ClassVar[int]
    SGR_FIELD_NUMBER: _ClassVar[int]
    rune: int
    sgr: SGR
    def __init__(self, rune: _Optional[int] = ..., sgr: _Optional[_Union[SGR, _Mapping]] = ...) -> None: ...

class Line(_message.Message):
    __slots__ = ("cells",)
    CELLS_FIELD_NUMBER: _ClassVar[int]
    cells: _containers.RepeatedCompositeFieldContainer[Cell]
    def __init__(self, cells: _Optional[_Iterable[_Union[Cell, _Mapping]]] = ...) -> None: ...

class Cursor(_message.Message):
    __slots__ = ("x", "y")
    X_FIELD_NUMBER: _ClassVar[int]
    Y_FIELD_NUMBER: _ClassVar[int]
    x: int
    y: int
    def __init__(self, x: _Optional[int] = ..., y: _Optional[int] = ...) -> None: ...

class GetScreenResponse(_message.Message):
    __slots__ = ("lines", "cursor", "cols", "rows", "in_alt_buffer", "scrollback_lines", "plain_text")
    LINES_FIELD_NUMBER: _ClassVar[int]
    CURSOR_FIELD_NUMBER: _ClassVar[int]
    COLS_FIELD_NUMBER: _ClassVar[int]
    ROWS_FIELD_NUMBER: _ClassVar[int]
    IN_ALT_BUFFER_FIELD_NUMBER: _ClassVar[int]
    SCROLLBACK_LINES_FIELD_NUMBER: _ClassVar[int]
    PLAIN_TEXT_FIELD_NUMBER: _ClassVar[int]
    lines: _containers.RepeatedCompositeFieldContainer[Line]
    cursor: Cursor
    cols: int
    rows: int
    in_alt_buffer: bool
    scrollback_lines: int
    plain_text: str
    def __init__(self, lines: _Optional[_Iterable[_Union[Line, _Mapping]]] = ..., cursor: _Optional[_Union[Cursor, _Mapping]] = ..., cols: _Optional[int] = ..., rows: _Optional[int] = ..., in_alt_buffer: _Optional[bool] = ..., scrollback_lines: _Optional[int] = ..., plain_text: _Optional[str] = ...) -> None: ...

class Key(_message.Message):
    __slots__ = ("name", "ctrl", "alt", "shift")
    NAME_FIELD_NUMBER: _ClassVar[int]
    CTRL_FIELD_NUMBER: _ClassVar[int]
    ALT_FIELD_NUMBER: _ClassVar[int]
    SHIFT_FIELD_NUMBER: _ClassVar[int]
    name: str
    ctrl: bool
    alt: bool
    shift: bool
    def __init__(self, name: _Optional[str] = ..., ctrl: _Optional[bool] = ..., alt: _Optional[bool] = ..., shift: _Optional[bool] = ...) -> None: ...

class KeySequence(_message.Message):
    __slots__ = ("keys",)
    KEYS_FIELD_NUMBER: _ClassVar[int]
    keys: _containers.RepeatedCompositeFieldContainer[Key]
    def __init__(self, keys: _Optional[_Iterable[_Union[Key, _Mapping]]] = ...) -> None: ...

class SendInputRequest(_message.Message):
    __slots__ = ("session_id", "text", "keys", "raw", "source", "is_paste")
    SESSION_ID_FIELD_NUMBER: _ClassVar[int]
    TEXT_FIELD_NUMBER: _ClassVar[int]
    KEYS_FIELD_NUMBER: _ClassVar[int]
    RAW_FIELD_NUMBER: _ClassVar[int]
    SOURCE_FIELD_NUMBER: _ClassVar[int]
    IS_PASTE_FIELD_NUMBER: _ClassVar[int]
    session_id: str
    text: str
    keys: KeySequence
    raw: bytes
    source: str
    is_paste: bool
    def __init__(self, session_id: _Optional[str] = ..., text: _Optional[str] = ..., keys: _Optional[_Union[KeySequence, _Mapping]] = ..., raw: _Optional[bytes] = ..., source: _Optional[str] = ..., is_paste: _Optional[bool] = ...) -> None: ...

class SendInputResponse(_message.Message):
    __slots__ = ("bytes_written",)
    BYTES_WRITTEN_FIELD_NUMBER: _ClassVar[int]
    bytes_written: int
    def __init__(self, bytes_written: _Optional[int] = ...) -> None: ...

class WaitIdleRequest(_message.Message):
    __slots__ = ("session_id", "quiet_window", "timeout")
    SESSION_ID_FIELD_NUMBER: _ClassVar[int]
    QUIET_WINDOW_FIELD_NUMBER: _ClassVar[int]
    TIMEOUT_FIELD_NUMBER: _ClassVar[int]
    session_id: str
    quiet_window: _duration_pb2.Duration
    timeout: _duration_pb2.Duration
    def __init__(self, session_id: _Optional[str] = ..., quiet_window: _Optional[_Union[datetime.timedelta, _duration_pb2.Duration, _Mapping]] = ..., timeout: _Optional[_Union[datetime.timedelta, _duration_pb2.Duration, _Mapping]] = ...) -> None: ...

class WaitIdleResponse(_message.Message):
    __slots__ = ("reason", "waited")
    class Reason(int, metaclass=_enum_type_wrapper.EnumTypeWrapper):
        __slots__ = ()
        REASON_UNSPECIFIED: _ClassVar[WaitIdleResponse.Reason]
        REASON_IDLE: _ClassVar[WaitIdleResponse.Reason]
        REASON_TIMEOUT: _ClassVar[WaitIdleResponse.Reason]
        REASON_EXITED: _ClassVar[WaitIdleResponse.Reason]
    REASON_UNSPECIFIED: WaitIdleResponse.Reason
    REASON_IDLE: WaitIdleResponse.Reason
    REASON_TIMEOUT: WaitIdleResponse.Reason
    REASON_EXITED: WaitIdleResponse.Reason
    REASON_FIELD_NUMBER: _ClassVar[int]
    WAITED_FIELD_NUMBER: _ClassVar[int]
    reason: WaitIdleResponse.Reason
    waited: _duration_pb2.Duration
    def __init__(self, reason: _Optional[_Union[WaitIdleResponse.Reason, str]] = ..., waited: _Optional[_Union[datetime.timedelta, _duration_pb2.Duration, _Mapping]] = ...) -> None: ...
