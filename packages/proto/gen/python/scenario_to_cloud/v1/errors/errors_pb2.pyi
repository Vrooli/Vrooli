from google.protobuf import struct_pb2 as _struct_pb2
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Mapping as _Mapping
from typing import ClassVar as _ClassVar, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class Error(_message.Message):
    __slots__ = ("code", "message", "retryable", "next_action", "details")
    CODE_FIELD_NUMBER: _ClassVar[int]
    MESSAGE_FIELD_NUMBER: _ClassVar[int]
    RETRYABLE_FIELD_NUMBER: _ClassVar[int]
    NEXT_ACTION_FIELD_NUMBER: _ClassVar[int]
    DETAILS_FIELD_NUMBER: _ClassVar[int]
    code: str
    message: str
    retryable: bool
    next_action: NextAction
    details: _struct_pb2.Struct
    def __init__(self, code: _Optional[str] = ..., message: _Optional[str] = ..., retryable: _Optional[bool] = ..., next_action: _Optional[_Union[NextAction, _Mapping]] = ..., details: _Optional[_Union[_struct_pb2.Struct, _Mapping]] = ...) -> None: ...

class NextAction(_message.Message):
    __slots__ = ("owner", "kind", "reference", "label")
    OWNER_FIELD_NUMBER: _ClassVar[int]
    KIND_FIELD_NUMBER: _ClassVar[int]
    REFERENCE_FIELD_NUMBER: _ClassVar[int]
    LABEL_FIELD_NUMBER: _ClassVar[int]
    owner: str
    kind: str
    reference: str
    label: str
    def __init__(self, owner: _Optional[str] = ..., kind: _Optional[str] = ..., reference: _Optional[str] = ..., label: _Optional[str] = ...) -> None: ...
