from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from typing import ClassVar as _ClassVar, Optional as _Optional

DESCRIPTOR: _descriptor.FileDescriptor

class MeasureRequest(_message.Message):
    __slots__ = ("operation_id", "request_json")
    OPERATION_ID_FIELD_NUMBER: _ClassVar[int]
    REQUEST_JSON_FIELD_NUMBER: _ClassVar[int]
    operation_id: str
    request_json: str
    def __init__(self, operation_id: _Optional[str] = ..., request_json: _Optional[str] = ...) -> None: ...

class PromoteRequest(_message.Message):
    __slots__ = ("operation_id", "request_json")
    OPERATION_ID_FIELD_NUMBER: _ClassVar[int]
    REQUEST_JSON_FIELD_NUMBER: _ClassVar[int]
    operation_id: str
    request_json: str
    def __init__(self, operation_id: _Optional[str] = ..., request_json: _Optional[str] = ...) -> None: ...

class CleanupRequest(_message.Message):
    __slots__ = ("operation_id", "request_json")
    OPERATION_ID_FIELD_NUMBER: _ClassVar[int]
    REQUEST_JSON_FIELD_NUMBER: _ClassVar[int]
    operation_id: str
    request_json: str
    def __init__(self, operation_id: _Optional[str] = ..., request_json: _Optional[str] = ...) -> None: ...

class StageReceipt(_message.Message):
    __slots__ = ("receipt_json",)
    RECEIPT_JSON_FIELD_NUMBER: _ClassVar[int]
    receipt_json: str
    def __init__(self, receipt_json: _Optional[str] = ...) -> None: ...
