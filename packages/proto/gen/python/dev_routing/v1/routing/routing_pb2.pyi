from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from typing import ClassVar as _ClassVar, Optional as _Optional

DESCRIPTOR: _descriptor.FileDescriptor

class InstallTestPoolRequest(_message.Message):
    __slots__ = ("dsn",)
    DSN_FIELD_NUMBER: _ClassVar[int]
    dsn: str
    def __init__(self, dsn: _Optional[str] = ...) -> None: ...

class InstallTestPoolResponse(_message.Message):
    __slots__ = ()
    def __init__(self) -> None: ...

class ClearTestPoolRequest(_message.Message):
    __slots__ = ()
    def __init__(self) -> None: ...

class ClearTestPoolResponse(_message.Message):
    __slots__ = ()
    def __init__(self) -> None: ...
