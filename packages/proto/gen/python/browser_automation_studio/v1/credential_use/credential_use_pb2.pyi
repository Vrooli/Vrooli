import datetime

from google.protobuf import timestamp_pb2 as _timestamp_pb2
from google.protobuf.internal import containers as _containers
from google.protobuf.internal import enum_type_wrapper as _enum_type_wrapper
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Iterable as _Iterable, Mapping as _Mapping
from typing import ClassVar as _ClassVar, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class BrowserExposureClass(int, metaclass=_enum_type_wrapper.EnumTypeWrapper):
    __slots__ = ()
    BROWSER_EXPOSURE_CLASS_UNSPECIFIED: _ClassVar[BrowserExposureClass]
    BROWSER_EXPOSURE_CLASS_PROTECTED: _ClassVar[BrowserExposureClass]
    BROWSER_EXPOSURE_CLASS_UNRESTRICTED_SESSION: _ClassVar[BrowserExposureClass]

class BrowserCredentialActionType(int, metaclass=_enum_type_wrapper.EnumTypeWrapper):
    __slots__ = ()
    BROWSER_CREDENTIAL_ACTION_UNSPECIFIED: _ClassVar[BrowserCredentialActionType]
    BROWSER_CREDENTIAL_ACTION_NAVIGATE: _ClassVar[BrowserCredentialActionType]
    BROWSER_CREDENTIAL_ACTION_CLICK: _ClassVar[BrowserCredentialActionType]
    BROWSER_CREDENTIAL_ACTION_SUBMIT: _ClassVar[BrowserCredentialActionType]
    BROWSER_CREDENTIAL_ACTION_COMPLETE: _ClassVar[BrowserCredentialActionType]
BROWSER_EXPOSURE_CLASS_UNSPECIFIED: BrowserExposureClass
BROWSER_EXPOSURE_CLASS_PROTECTED: BrowserExposureClass
BROWSER_EXPOSURE_CLASS_UNRESTRICTED_SESSION: BrowserExposureClass
BROWSER_CREDENTIAL_ACTION_UNSPECIFIED: BrowserCredentialActionType
BROWSER_CREDENTIAL_ACTION_NAVIGATE: BrowserCredentialActionType
BROWSER_CREDENTIAL_ACTION_CLICK: BrowserCredentialActionType
BROWSER_CREDENTIAL_ACTION_SUBMIT: BrowserCredentialActionType
BROWSER_CREDENTIAL_ACTION_COMPLETE: BrowserCredentialActionType

class BrowserSessionPolicy(_message.Message):
    __slots__ = ("session_id", "origin", "account", "document_id", "exposure_class", "allowed_operations", "expires_at")
    SESSION_ID_FIELD_NUMBER: _ClassVar[int]
    ORIGIN_FIELD_NUMBER: _ClassVar[int]
    ACCOUNT_FIELD_NUMBER: _ClassVar[int]
    DOCUMENT_ID_FIELD_NUMBER: _ClassVar[int]
    EXPOSURE_CLASS_FIELD_NUMBER: _ClassVar[int]
    ALLOWED_OPERATIONS_FIELD_NUMBER: _ClassVar[int]
    EXPIRES_AT_FIELD_NUMBER: _ClassVar[int]
    session_id: str
    origin: str
    account: str
    document_id: str
    exposure_class: BrowserExposureClass
    allowed_operations: _containers.RepeatedScalarFieldContainer[str]
    expires_at: _timestamp_pb2.Timestamp
    def __init__(self, session_id: _Optional[str] = ..., origin: _Optional[str] = ..., account: _Optional[str] = ..., document_id: _Optional[str] = ..., exposure_class: _Optional[_Union[BrowserExposureClass, str]] = ..., allowed_operations: _Optional[_Iterable[str]] = ..., expires_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ...) -> None: ...

class BrowserCredentialAction(_message.Message):
    __slots__ = ("type", "origin", "document_id", "navigation_url", "selector")
    TYPE_FIELD_NUMBER: _ClassVar[int]
    ORIGIN_FIELD_NUMBER: _ClassVar[int]
    DOCUMENT_ID_FIELD_NUMBER: _ClassVar[int]
    NAVIGATION_URL_FIELD_NUMBER: _ClassVar[int]
    SELECTOR_FIELD_NUMBER: _ClassVar[int]
    type: BrowserCredentialActionType
    origin: str
    document_id: str
    navigation_url: str
    selector: str
    def __init__(self, type: _Optional[_Union[BrowserCredentialActionType, str]] = ..., origin: _Optional[str] = ..., document_id: _Optional[str] = ..., navigation_url: _Optional[str] = ..., selector: _Optional[str] = ...) -> None: ...

class BrowserCredentialResult(_message.Message):
    __slots__ = ("session_id", "status", "operation_class", "origin", "document_id", "exposure_class", "cleared_fields")
    SESSION_ID_FIELD_NUMBER: _ClassVar[int]
    STATUS_FIELD_NUMBER: _ClassVar[int]
    OPERATION_CLASS_FIELD_NUMBER: _ClassVar[int]
    ORIGIN_FIELD_NUMBER: _ClassVar[int]
    DOCUMENT_ID_FIELD_NUMBER: _ClassVar[int]
    EXPOSURE_CLASS_FIELD_NUMBER: _ClassVar[int]
    CLEARED_FIELDS_FIELD_NUMBER: _ClassVar[int]
    session_id: str
    status: str
    operation_class: str
    origin: str
    document_id: str
    exposure_class: BrowserExposureClass
    cleared_fields: _containers.RepeatedScalarFieldContainer[str]
    def __init__(self, session_id: _Optional[str] = ..., status: _Optional[str] = ..., operation_class: _Optional[str] = ..., origin: _Optional[str] = ..., document_id: _Optional[str] = ..., exposure_class: _Optional[_Union[BrowserExposureClass, str]] = ..., cleared_fields: _Optional[_Iterable[str]] = ...) -> None: ...
