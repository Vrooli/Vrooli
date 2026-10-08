from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from typing import ClassVar as _ClassVar, Optional as _Optional

DESCRIPTOR: _descriptor.FileDescriptor

class GenerateDesignLanguageRequest(_message.Message):
    __slots__ = ("brand_id",)
    BRAND_ID_FIELD_NUMBER: _ClassVar[int]
    brand_id: str
    def __init__(self, brand_id: _Optional[str] = ...) -> None: ...

class GenerateDesignLanguageResponse(_message.Message):
    __slots__ = ("brand_id", "markdown")
    BRAND_ID_FIELD_NUMBER: _ClassVar[int]
    MARKDOWN_FIELD_NUMBER: _ClassVar[int]
    brand_id: str
    markdown: str
    def __init__(self, brand_id: _Optional[str] = ..., markdown: _Optional[str] = ...) -> None: ...
