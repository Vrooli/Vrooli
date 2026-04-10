from buf.validate import validate_pb2 as _validate_pb2
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from typing import ClassVar as _ClassVar, Optional as _Optional

DESCRIPTOR: _descriptor.FileDescriptor

class Resource(_message.Message):
    __slots__ = ("name", "display_name", "path", "port", "category", "description", "healthy", "version", "status")
    NAME_FIELD_NUMBER: _ClassVar[int]
    DISPLAY_NAME_FIELD_NUMBER: _ClassVar[int]
    PATH_FIELD_NUMBER: _ClassVar[int]
    PORT_FIELD_NUMBER: _ClassVar[int]
    CATEGORY_FIELD_NUMBER: _ClassVar[int]
    DESCRIPTION_FIELD_NUMBER: _ClassVar[int]
    HEALTHY_FIELD_NUMBER: _ClassVar[int]
    VERSION_FIELD_NUMBER: _ClassVar[int]
    STATUS_FIELD_NUMBER: _ClassVar[int]
    name: str
    display_name: str
    path: str
    port: int
    category: str
    description: str
    healthy: bool
    version: str
    status: str
    def __init__(self, name: _Optional[str] = ..., display_name: _Optional[str] = ..., path: _Optional[str] = ..., port: _Optional[int] = ..., category: _Optional[str] = ..., description: _Optional[str] = ..., healthy: _Optional[bool] = ..., version: _Optional[str] = ..., status: _Optional[str] = ...) -> None: ...

class Scenario(_message.Message):
    __slots__ = ("name", "display_name", "path", "category", "description", "version", "status")
    NAME_FIELD_NUMBER: _ClassVar[int]
    DISPLAY_NAME_FIELD_NUMBER: _ClassVar[int]
    PATH_FIELD_NUMBER: _ClassVar[int]
    CATEGORY_FIELD_NUMBER: _ClassVar[int]
    DESCRIPTION_FIELD_NUMBER: _ClassVar[int]
    VERSION_FIELD_NUMBER: _ClassVar[int]
    STATUS_FIELD_NUMBER: _ClassVar[int]
    name: str
    display_name: str
    path: str
    category: str
    description: str
    version: str
    status: str
    def __init__(self, name: _Optional[str] = ..., display_name: _Optional[str] = ..., path: _Optional[str] = ..., category: _Optional[str] = ..., description: _Optional[str] = ..., version: _Optional[str] = ..., status: _Optional[str] = ...) -> None: ...

class Operation(_message.Message):
    __slots__ = ("name", "description")
    NAME_FIELD_NUMBER: _ClassVar[int]
    DESCRIPTION_FIELD_NUMBER: _ClassVar[int]
    name: str
    description: str
    def __init__(self, name: _Optional[str] = ..., description: _Optional[str] = ...) -> None: ...

class Category(_message.Message):
    __slots__ = ("name", "description", "count")
    NAME_FIELD_NUMBER: _ClassVar[int]
    DESCRIPTION_FIELD_NUMBER: _ClassVar[int]
    COUNT_FIELD_NUMBER: _ClassVar[int]
    name: str
    description: str
    count: int
    def __init__(self, name: _Optional[str] = ..., description: _Optional[str] = ..., count: _Optional[int] = ...) -> None: ...
