from google.protobuf.internal import containers as _containers
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Iterable as _Iterable, Mapping as _Mapping
from typing import ClassVar as _ClassVar, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class RunRequest(_message.Message):
    __slots__ = ("package_id", "sandbox")
    PACKAGE_ID_FIELD_NUMBER: _ClassVar[int]
    SANDBOX_FIELD_NUMBER: _ClassVar[int]
    package_id: str
    sandbox: str
    def __init__(self, package_id: _Optional[str] = ..., sandbox: _Optional[str] = ...) -> None: ...

class RunResponse(_message.Message):
    __slots__ = ("passed", "journey_manifest", "commands", "findings")
    PASSED_FIELD_NUMBER: _ClassVar[int]
    JOURNEY_MANIFEST_FIELD_NUMBER: _ClassVar[int]
    COMMANDS_FIELD_NUMBER: _ClassVar[int]
    FINDINGS_FIELD_NUMBER: _ClassVar[int]
    passed: bool
    journey_manifest: str
    commands: _containers.RepeatedCompositeFieldContainer[CommandResult]
    findings: _containers.RepeatedCompositeFieldContainer[Finding]
    def __init__(self, passed: _Optional[bool] = ..., journey_manifest: _Optional[str] = ..., commands: _Optional[_Iterable[_Union[CommandResult, _Mapping]]] = ..., findings: _Optional[_Iterable[_Union[Finding, _Mapping]]] = ...) -> None: ...

class CommandResult(_message.Message):
    __slots__ = ("command", "exit_code", "redacted_output")
    COMMAND_FIELD_NUMBER: _ClassVar[int]
    EXIT_CODE_FIELD_NUMBER: _ClassVar[int]
    REDACTED_OUTPUT_FIELD_NUMBER: _ClassVar[int]
    command: str
    exit_code: int
    redacted_output: str
    def __init__(self, command: _Optional[str] = ..., exit_code: _Optional[int] = ..., redacted_output: _Optional[str] = ...) -> None: ...

class Finding(_message.Message):
    __slots__ = ("code", "message")
    CODE_FIELD_NUMBER: _ClassVar[int]
    MESSAGE_FIELD_NUMBER: _ClassVar[int]
    code: str
    message: str
    def __init__(self, code: _Optional[str] = ..., message: _Optional[str] = ...) -> None: ...
