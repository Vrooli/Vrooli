from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Mapping as _Mapping
from typing import ClassVar as _ClassVar, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class Open(_message.Message):
    __slots__ = ("session_id", "node_id", "receive_window", "idle_timeout_seconds", "max_lifetime_seconds", "shell", "working_dir")
    SESSION_ID_FIELD_NUMBER: _ClassVar[int]
    NODE_ID_FIELD_NUMBER: _ClassVar[int]
    RECEIVE_WINDOW_FIELD_NUMBER: _ClassVar[int]
    IDLE_TIMEOUT_SECONDS_FIELD_NUMBER: _ClassVar[int]
    MAX_LIFETIME_SECONDS_FIELD_NUMBER: _ClassVar[int]
    SHELL_FIELD_NUMBER: _ClassVar[int]
    WORKING_DIR_FIELD_NUMBER: _ClassVar[int]
    session_id: str
    node_id: str
    receive_window: int
    idle_timeout_seconds: int
    max_lifetime_seconds: int
    shell: str
    working_dir: str
    def __init__(self, session_id: _Optional[str] = ..., node_id: _Optional[str] = ..., receive_window: _Optional[int] = ..., idle_timeout_seconds: _Optional[int] = ..., max_lifetime_seconds: _Optional[int] = ..., shell: _Optional[str] = ..., working_dir: _Optional[str] = ...) -> None: ...

class Data(_message.Message):
    __slots__ = ("sequence", "data")
    SEQUENCE_FIELD_NUMBER: _ClassVar[int]
    DATA_FIELD_NUMBER: _ClassVar[int]
    sequence: int
    data: bytes
    def __init__(self, sequence: _Optional[int] = ..., data: _Optional[bytes] = ...) -> None: ...

class Resize(_message.Message):
    __slots__ = ("columns", "rows")
    COLUMNS_FIELD_NUMBER: _ClassVar[int]
    ROWS_FIELD_NUMBER: _ClassVar[int]
    columns: int
    rows: int
    def __init__(self, columns: _Optional[int] = ..., rows: _Optional[int] = ...) -> None: ...

class Close(_message.Message):
    __slots__ = ("code", "reason")
    CODE_FIELD_NUMBER: _ClassVar[int]
    REASON_FIELD_NUMBER: _ClassVar[int]
    code: str
    reason: str
    def __init__(self, code: _Optional[str] = ..., reason: _Optional[str] = ...) -> None: ...

class Ack(_message.Message):
    __slots__ = ("accepted", "sequence", "window_available", "code", "reason")
    ACCEPTED_FIELD_NUMBER: _ClassVar[int]
    SEQUENCE_FIELD_NUMBER: _ClassVar[int]
    WINDOW_AVAILABLE_FIELD_NUMBER: _ClassVar[int]
    CODE_FIELD_NUMBER: _ClassVar[int]
    REASON_FIELD_NUMBER: _ClassVar[int]
    accepted: bool
    sequence: int
    window_available: int
    code: str
    reason: str
    def __init__(self, accepted: _Optional[bool] = ..., sequence: _Optional[int] = ..., window_available: _Optional[int] = ..., code: _Optional[str] = ..., reason: _Optional[str] = ...) -> None: ...

class WindowUpdate(_message.Message):
    __slots__ = ("window_available",)
    WINDOW_AVAILABLE_FIELD_NUMBER: _ClassVar[int]
    window_available: int
    def __init__(self, window_available: _Optional[int] = ...) -> None: ...

class Frame(_message.Message):
    __slots__ = ("open", "data", "resize", "close", "ack", "window_update")
    OPEN_FIELD_NUMBER: _ClassVar[int]
    DATA_FIELD_NUMBER: _ClassVar[int]
    RESIZE_FIELD_NUMBER: _ClassVar[int]
    CLOSE_FIELD_NUMBER: _ClassVar[int]
    ACK_FIELD_NUMBER: _ClassVar[int]
    WINDOW_UPDATE_FIELD_NUMBER: _ClassVar[int]
    open: Open
    data: Data
    resize: Resize
    close: Close
    ack: Ack
    window_update: WindowUpdate
    def __init__(self, open: _Optional[_Union[Open, _Mapping]] = ..., data: _Optional[_Union[Data, _Mapping]] = ..., resize: _Optional[_Union[Resize, _Mapping]] = ..., close: _Optional[_Union[Close, _Mapping]] = ..., ack: _Optional[_Union[Ack, _Mapping]] = ..., window_update: _Optional[_Union[WindowUpdate, _Mapping]] = ...) -> None: ...
