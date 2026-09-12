from google.protobuf.internal import containers as _containers
from google.protobuf.internal import enum_type_wrapper as _enum_type_wrapper
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Mapping as _Mapping
from typing import ClassVar as _ClassVar, Optional as _Optional

DESCRIPTOR: _descriptor.FileDescriptor

class DomainErrorCode(int, metaclass=_enum_type_wrapper.EnumTypeWrapper):
    __slots__ = ()
    DOMAIN_ERROR_CODE_UNSPECIFIED: _ClassVar[DomainErrorCode]
    DOMAIN_ERROR_CODE_INVALID_FACET: _ClassVar[DomainErrorCode]
    DOMAIN_ERROR_CODE_PIN_BUDGET_EXCEEDED: _ClassVar[DomainErrorCode]
    DOMAIN_ERROR_CODE_IMPORT_SOURCE_UNREADABLE: _ClassVar[DomainErrorCode]
    DOMAIN_ERROR_CODE_IMPORT_FORMAT_CHANGED: _ClassVar[DomainErrorCode]
    DOMAIN_ERROR_CODE_PROJECTION_OVERFLOW: _ClassVar[DomainErrorCode]
    DOMAIN_ERROR_CODE_WORK_RECORD_INVALID: _ClassVar[DomainErrorCode]
    DOMAIN_ERROR_CODE_TOKEN_REQUIRED: _ClassVar[DomainErrorCode]
DOMAIN_ERROR_CODE_UNSPECIFIED: DomainErrorCode
DOMAIN_ERROR_CODE_INVALID_FACET: DomainErrorCode
DOMAIN_ERROR_CODE_PIN_BUDGET_EXCEEDED: DomainErrorCode
DOMAIN_ERROR_CODE_IMPORT_SOURCE_UNREADABLE: DomainErrorCode
DOMAIN_ERROR_CODE_IMPORT_FORMAT_CHANGED: DomainErrorCode
DOMAIN_ERROR_CODE_PROJECTION_OVERFLOW: DomainErrorCode
DOMAIN_ERROR_CODE_WORK_RECORD_INVALID: DomainErrorCode
DOMAIN_ERROR_CODE_TOKEN_REQUIRED: DomainErrorCode

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
