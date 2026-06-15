from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from typing import ClassVar as _ClassVar, Optional as _Optional

DESCRIPTOR: _descriptor.FileDescriptor

class TestPhaseResult(_message.Message):
    __slots__ = ("scenario", "status", "exit_code", "started_at", "ended_at", "duration", "log_file")
    SCENARIO_FIELD_NUMBER: _ClassVar[int]
    STATUS_FIELD_NUMBER: _ClassVar[int]
    EXIT_CODE_FIELD_NUMBER: _ClassVar[int]
    STARTED_AT_FIELD_NUMBER: _ClassVar[int]
    ENDED_AT_FIELD_NUMBER: _ClassVar[int]
    DURATION_FIELD_NUMBER: _ClassVar[int]
    LOG_FILE_FIELD_NUMBER: _ClassVar[int]
    scenario: str
    status: str
    exit_code: int
    started_at: str
    ended_at: str
    duration: str
    log_file: str
    def __init__(self, scenario: _Optional[str] = ..., status: _Optional[str] = ..., exit_code: _Optional[int] = ..., started_at: _Optional[str] = ..., ended_at: _Optional[str] = ..., duration: _Optional[str] = ..., log_file: _Optional[str] = ...) -> None: ...
