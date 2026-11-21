from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from typing import ClassVar as _ClassVar, Optional as _Optional

DESCRIPTOR: _descriptor.FileDescriptor

class SubmitOptimizationRequest(_message.Message):
    __slots__ = ("request_json",)
    REQUEST_JSON_FIELD_NUMBER: _ClassVar[int]
    request_json: str
    def __init__(self, request_json: _Optional[str] = ...) -> None: ...

class SubmitOptimizationResponse(_message.Message):
    __slots__ = ("run_id", "record_json")
    RUN_ID_FIELD_NUMBER: _ClassVar[int]
    RECORD_JSON_FIELD_NUMBER: _ClassVar[int]
    run_id: str
    record_json: str
    def __init__(self, run_id: _Optional[str] = ..., record_json: _Optional[str] = ...) -> None: ...

class WaitOptimizationRequest(_message.Message):
    __slots__ = ("run_id", "wait_ms")
    RUN_ID_FIELD_NUMBER: _ClassVar[int]
    WAIT_MS_FIELD_NUMBER: _ClassVar[int]
    run_id: str
    wait_ms: int
    def __init__(self, run_id: _Optional[str] = ..., wait_ms: _Optional[int] = ...) -> None: ...

class WaitOptimizationResponse(_message.Message):
    __slots__ = ("record_json", "terminal", "waited_ms")
    RECORD_JSON_FIELD_NUMBER: _ClassVar[int]
    TERMINAL_FIELD_NUMBER: _ClassVar[int]
    WAITED_MS_FIELD_NUMBER: _ClassVar[int]
    record_json: str
    terminal: bool
    waited_ms: int
    def __init__(self, record_json: _Optional[str] = ..., terminal: _Optional[bool] = ..., waited_ms: _Optional[int] = ...) -> None: ...
