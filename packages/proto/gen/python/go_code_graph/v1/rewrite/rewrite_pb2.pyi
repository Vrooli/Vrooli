from google.protobuf.internal import enum_type_wrapper as _enum_type_wrapper
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Mapping as _Mapping
from typing import ClassVar as _ClassVar, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class OperationStatus(int, metaclass=_enum_type_wrapper.EnumTypeWrapper):
    __slots__ = ()
    OPERATION_STATUS_UNSPECIFIED: _ClassVar[OperationStatus]
    OPERATION_STATUS_OK: _ClassVar[OperationStatus]
    OPERATION_STATUS_FAILED: _ClassVar[OperationStatus]
OPERATION_STATUS_UNSPECIFIED: OperationStatus
OPERATION_STATUS_OK: OperationStatus
OPERATION_STATUS_FAILED: OperationStatus

class FileMove(_message.Message):
    __slots__ = ("from_path", "to_path")
    FROM_PATH_FIELD_NUMBER: _ClassVar[int]
    TO_PATH_FIELD_NUMBER: _ClassVar[int]
    from_path: str
    to_path: str
    def __init__(self, from_path: _Optional[str] = ..., to_path: _Optional[str] = ...) -> None: ...

class ImportRewrite(_message.Message):
    __slots__ = ("old_path", "new_path")
    OLD_PATH_FIELD_NUMBER: _ClassVar[int]
    NEW_PATH_FIELD_NUMBER: _ClassVar[int]
    old_path: str
    new_path: str
    def __init__(self, old_path: _Optional[str] = ..., new_path: _Optional[str] = ...) -> None: ...

class Operation(_message.Message):
    __slots__ = ("file_move", "import_rewrite")
    FILE_MOVE_FIELD_NUMBER: _ClassVar[int]
    IMPORT_REWRITE_FIELD_NUMBER: _ClassVar[int]
    file_move: FileMove
    import_rewrite: ImportRewrite
    def __init__(self, file_move: _Optional[_Union[FileMove, _Mapping]] = ..., import_rewrite: _Optional[_Union[ImportRewrite, _Mapping]] = ...) -> None: ...

class OperationResult(_message.Message):
    __slots__ = ("operation", "status", "message")
    OPERATION_FIELD_NUMBER: _ClassVar[int]
    STATUS_FIELD_NUMBER: _ClassVar[int]
    MESSAGE_FIELD_NUMBER: _ClassVar[int]
    operation: Operation
    status: OperationStatus
    message: str
    def __init__(self, operation: _Optional[_Union[Operation, _Mapping]] = ..., status: _Optional[_Union[OperationStatus, str]] = ..., message: _Optional[str] = ...) -> None: ...
