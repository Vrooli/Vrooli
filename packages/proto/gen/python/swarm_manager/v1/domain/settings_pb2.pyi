from buf.validate import validate_pb2 as _validate_pb2
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from typing import ClassVar as _ClassVar, Optional as _Optional

DESCRIPTOR: _descriptor.FileDescriptor

class Settings(_message.Message):
    __slots__ = ("theme", "custom_focus", "insights_enabled", "insights_auto_analyze")
    THEME_FIELD_NUMBER: _ClassVar[int]
    CUSTOM_FOCUS_FIELD_NUMBER: _ClassVar[int]
    INSIGHTS_ENABLED_FIELD_NUMBER: _ClassVar[int]
    INSIGHTS_AUTO_ANALYZE_FIELD_NUMBER: _ClassVar[int]
    theme: str
    custom_focus: str
    insights_enabled: bool
    insights_auto_analyze: bool
    def __init__(self, theme: _Optional[str] = ..., custom_focus: _Optional[str] = ..., insights_enabled: _Optional[bool] = ..., insights_auto_analyze: _Optional[bool] = ...) -> None: ...
