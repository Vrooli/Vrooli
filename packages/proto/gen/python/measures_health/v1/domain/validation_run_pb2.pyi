from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from typing import ClassVar as _ClassVar, Optional as _Optional

DESCRIPTOR: _descriptor.FileDescriptor

class ValidationRun(_message.Message):
    __slots__ = ("id", "scenario", "passed", "error_count", "warning_count", "ran_at")
    ID_FIELD_NUMBER: _ClassVar[int]
    SCENARIO_FIELD_NUMBER: _ClassVar[int]
    PASSED_FIELD_NUMBER: _ClassVar[int]
    ERROR_COUNT_FIELD_NUMBER: _ClassVar[int]
    WARNING_COUNT_FIELD_NUMBER: _ClassVar[int]
    RAN_AT_FIELD_NUMBER: _ClassVar[int]
    id: str
    scenario: str
    passed: bool
    error_count: int
    warning_count: int
    ran_at: str
    def __init__(self, id: _Optional[str] = ..., scenario: _Optional[str] = ..., passed: _Optional[bool] = ..., error_count: _Optional[int] = ..., warning_count: _Optional[int] = ..., ran_at: _Optional[str] = ...) -> None: ...
