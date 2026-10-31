from google.protobuf.internal import containers as _containers
from google.protobuf.internal import enum_type_wrapper as _enum_type_wrapper
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Mapping as _Mapping
from typing import ClassVar as _ClassVar, Optional as _Optional

DESCRIPTOR: _descriptor.FileDescriptor

class ExtractError(int, metaclass=_enum_type_wrapper.EnumTypeWrapper):
    __slots__ = ()
    EXTRACT_ERROR_UNSPECIFIED: _ClassVar[ExtractError]
    EXTRACT_ERROR_NO_GO_MOD_FOUND: _ClassVar[ExtractError]
    EXTRACT_ERROR_MULTIPLE_GO_MOD_FILES: _ClassVar[ExtractError]
    EXTRACT_ERROR_WORKSPACE_UNSUPPORTED: _ClassVar[ExtractError]
    EXTRACT_ERROR_PATH_UNREADABLE: _ClassVar[ExtractError]
    EXTRACT_ERROR_INTERNAL: _ClassVar[ExtractError]

class RewriteError(int, metaclass=_enum_type_wrapper.EnumTypeWrapper):
    __slots__ = ()
    REWRITE_ERROR_UNSPECIFIED: _ClassVar[RewriteError]
    REWRITE_ERROR_NO_OPERATIONS: _ClassVar[RewriteError]
    REWRITE_ERROR_MALFORMED_OPERATION: _ClassVar[RewriteError]
    REWRITE_ERROR_PLAN_NOT_FOUND: _ClassVar[RewriteError]
    REWRITE_ERROR_PATH_MISMATCH: _ClassVar[RewriteError]
    REWRITE_ERROR_APPLY_NOT_SET: _ClassVar[RewriteError]
    REWRITE_ERROR_INTERNAL: _ClassVar[RewriteError]
EXTRACT_ERROR_UNSPECIFIED: ExtractError
EXTRACT_ERROR_NO_GO_MOD_FOUND: ExtractError
EXTRACT_ERROR_MULTIPLE_GO_MOD_FILES: ExtractError
EXTRACT_ERROR_WORKSPACE_UNSUPPORTED: ExtractError
EXTRACT_ERROR_PATH_UNREADABLE: ExtractError
EXTRACT_ERROR_INTERNAL: ExtractError
REWRITE_ERROR_UNSPECIFIED: RewriteError
REWRITE_ERROR_NO_OPERATIONS: RewriteError
REWRITE_ERROR_MALFORMED_OPERATION: RewriteError
REWRITE_ERROR_PLAN_NOT_FOUND: RewriteError
REWRITE_ERROR_PATH_MISMATCH: RewriteError
REWRITE_ERROR_APPLY_NOT_SET: RewriteError
REWRITE_ERROR_INTERNAL: RewriteError

class ErrorEnvelope(_message.Message):
    __slots__ = ("code", "message", "details")
    class DetailsEntry(_message.Message):
        __slots__ = ("key", "value")
        KEY_FIELD_NUMBER: _ClassVar[int]
        VALUE_FIELD_NUMBER: _ClassVar[int]
        key: str
        value: str
        def __init__(self, key: _Optional[str] = ..., value: _Optional[str] = ...) -> None: ...
    CODE_FIELD_NUMBER: _ClassVar[int]
    MESSAGE_FIELD_NUMBER: _ClassVar[int]
    DETAILS_FIELD_NUMBER: _ClassVar[int]
    code: str
    message: str
    details: _containers.ScalarMap[str, str]
    def __init__(self, code: _Optional[str] = ..., message: _Optional[str] = ..., details: _Optional[_Mapping[str, str]] = ...) -> None: ...
