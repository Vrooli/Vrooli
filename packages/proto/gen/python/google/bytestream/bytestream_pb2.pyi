from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from typing import ClassVar as _ClassVar, Optional as _Optional

DESCRIPTOR: _descriptor.FileDescriptor

class ReadRequest(_message.Message):
    __slots__ = ()
    RESOURCE_NAME_FIELD_NUMBER: _ClassVar[int]
    READ_OFFSET_FIELD_NUMBER: _ClassVar[int]
    READ_LIMIT_FIELD_NUMBER: _ClassVar[int]
    resource_name: str
    read_offset: int
    read_limit: int
    def __init__(self, resource_name: _Optional[str] = ..., read_offset: _Optional[int] = ..., read_limit: _Optional[int] = ...) -> None: ...

class ReadResponse(_message.Message):
    __slots__ = ()
    DATA_FIELD_NUMBER: _ClassVar[int]
    data: bytes
    def __init__(self, data: _Optional[bytes] = ...) -> None: ...

class WriteRequest(_message.Message):
    __slots__ = ()
    RESOURCE_NAME_FIELD_NUMBER: _ClassVar[int]
    WRITE_OFFSET_FIELD_NUMBER: _ClassVar[int]
    FINISH_WRITE_FIELD_NUMBER: _ClassVar[int]
    DATA_FIELD_NUMBER: _ClassVar[int]
    resource_name: str
    write_offset: int
    finish_write: bool
    data: bytes
    def __init__(self, resource_name: _Optional[str] = ..., write_offset: _Optional[int] = ..., finish_write: _Optional[bool] = ..., data: _Optional[bytes] = ...) -> None: ...

class WriteResponse(_message.Message):
    __slots__ = ()
    COMMITTED_SIZE_FIELD_NUMBER: _ClassVar[int]
    committed_size: int
    def __init__(self, committed_size: _Optional[int] = ...) -> None: ...

class QueryWriteStatusRequest(_message.Message):
    __slots__ = ()
    RESOURCE_NAME_FIELD_NUMBER: _ClassVar[int]
    resource_name: str
    def __init__(self, resource_name: _Optional[str] = ...) -> None: ...

class QueryWriteStatusResponse(_message.Message):
    __slots__ = ()
    COMMITTED_SIZE_FIELD_NUMBER: _ClassVar[int]
    COMPLETE_FIELD_NUMBER: _ClassVar[int]
    committed_size: int
    complete: bool
    def __init__(self, committed_size: _Optional[int] = ..., complete: _Optional[bool] = ...) -> None: ...
