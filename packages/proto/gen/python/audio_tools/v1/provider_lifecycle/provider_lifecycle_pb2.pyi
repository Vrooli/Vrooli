from google.protobuf.internal import containers as _containers
from google.protobuf.internal import enum_type_wrapper as _enum_type_wrapper
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Iterable as _Iterable, Mapping as _Mapping
from typing import ClassVar as _ClassVar, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class ProcessState(int, metaclass=_enum_type_wrapper.EnumTypeWrapper):
    __slots__ = ()
    PROCESS_STATE_UNSPECIFIED: _ClassVar[ProcessState]
    PROCESS_STATE_RUNNING: _ClassVar[ProcessState]
    PROCESS_STATE_STOPPED: _ClassVar[ProcessState]
    PROCESS_STATE_UNKNOWN: _ClassVar[ProcessState]

class Action(int, metaclass=_enum_type_wrapper.EnumTypeWrapper):
    __slots__ = ()
    ACTION_UNSPECIFIED: _ClassVar[Action]
    ACTION_START: _ClassVar[Action]
    ACTION_STOP: _ClassVar[Action]
    ACTION_RESTART: _ClassVar[Action]
    ACTION_PULL_MODEL: _ClassVar[Action]
    ACTION_VIEW_LOGS: _ClassVar[Action]

class LogStream(int, metaclass=_enum_type_wrapper.EnumTypeWrapper):
    __slots__ = ()
    LOG_STREAM_UNSPECIFIED: _ClassVar[LogStream]
    LOG_STREAM_STDOUT: _ClassVar[LogStream]
    LOG_STREAM_STDERR: _ClassVar[LogStream]
PROCESS_STATE_UNSPECIFIED: ProcessState
PROCESS_STATE_RUNNING: ProcessState
PROCESS_STATE_STOPPED: ProcessState
PROCESS_STATE_UNKNOWN: ProcessState
ACTION_UNSPECIFIED: Action
ACTION_START: Action
ACTION_STOP: Action
ACTION_RESTART: Action
ACTION_PULL_MODEL: Action
ACTION_VIEW_LOGS: Action
LOG_STREAM_UNSPECIFIED: LogStream
LOG_STREAM_STDOUT: LogStream
LOG_STREAM_STDERR: LogStream

class LocalProvider(_message.Message):
    __slots__ = ("provider_id", "display_name", "resource_slug", "process_state", "supported_actions")
    PROVIDER_ID_FIELD_NUMBER: _ClassVar[int]
    DISPLAY_NAME_FIELD_NUMBER: _ClassVar[int]
    RESOURCE_SLUG_FIELD_NUMBER: _ClassVar[int]
    PROCESS_STATE_FIELD_NUMBER: _ClassVar[int]
    SUPPORTED_ACTIONS_FIELD_NUMBER: _ClassVar[int]
    provider_id: str
    display_name: str
    resource_slug: str
    process_state: ProcessState
    supported_actions: _containers.RepeatedScalarFieldContainer[Action]
    def __init__(self, provider_id: _Optional[str] = ..., display_name: _Optional[str] = ..., resource_slug: _Optional[str] = ..., process_state: _Optional[_Union[ProcessState, str]] = ..., supported_actions: _Optional[_Iterable[_Union[Action, str]]] = ...) -> None: ...

class ListLocalProvidersRequest(_message.Message):
    __slots__ = ()
    def __init__(self) -> None: ...

class ListLocalProvidersResponse(_message.Message):
    __slots__ = ("providers",)
    PROVIDERS_FIELD_NUMBER: _ClassVar[int]
    providers: _containers.RepeatedCompositeFieldContainer[LocalProvider]
    def __init__(self, providers: _Optional[_Iterable[_Union[LocalProvider, _Mapping]]] = ...) -> None: ...

class StartProviderRequest(_message.Message):
    __slots__ = ("provider_id",)
    PROVIDER_ID_FIELD_NUMBER: _ClassVar[int]
    provider_id: str
    def __init__(self, provider_id: _Optional[str] = ...) -> None: ...

class StartProviderResponse(_message.Message):
    __slots__ = ("provider_id", "dry_run", "message")
    PROVIDER_ID_FIELD_NUMBER: _ClassVar[int]
    DRY_RUN_FIELD_NUMBER: _ClassVar[int]
    MESSAGE_FIELD_NUMBER: _ClassVar[int]
    provider_id: str
    dry_run: bool
    message: str
    def __init__(self, provider_id: _Optional[str] = ..., dry_run: _Optional[bool] = ..., message: _Optional[str] = ...) -> None: ...

class StopProviderRequest(_message.Message):
    __slots__ = ("provider_id",)
    PROVIDER_ID_FIELD_NUMBER: _ClassVar[int]
    provider_id: str
    def __init__(self, provider_id: _Optional[str] = ...) -> None: ...

class StopProviderResponse(_message.Message):
    __slots__ = ("provider_id", "dry_run", "message")
    PROVIDER_ID_FIELD_NUMBER: _ClassVar[int]
    DRY_RUN_FIELD_NUMBER: _ClassVar[int]
    MESSAGE_FIELD_NUMBER: _ClassVar[int]
    provider_id: str
    dry_run: bool
    message: str
    def __init__(self, provider_id: _Optional[str] = ..., dry_run: _Optional[bool] = ..., message: _Optional[str] = ...) -> None: ...

class RestartProviderRequest(_message.Message):
    __slots__ = ("provider_id",)
    PROVIDER_ID_FIELD_NUMBER: _ClassVar[int]
    provider_id: str
    def __init__(self, provider_id: _Optional[str] = ...) -> None: ...

class RestartProviderResponse(_message.Message):
    __slots__ = ("provider_id", "dry_run", "message")
    PROVIDER_ID_FIELD_NUMBER: _ClassVar[int]
    DRY_RUN_FIELD_NUMBER: _ClassVar[int]
    MESSAGE_FIELD_NUMBER: _ClassVar[int]
    provider_id: str
    dry_run: bool
    message: str
    def __init__(self, provider_id: _Optional[str] = ..., dry_run: _Optional[bool] = ..., message: _Optional[str] = ...) -> None: ...

class PullModelRequest(_message.Message):
    __slots__ = ("provider_id", "model_name")
    PROVIDER_ID_FIELD_NUMBER: _ClassVar[int]
    MODEL_NAME_FIELD_NUMBER: _ClassVar[int]
    provider_id: str
    model_name: str
    def __init__(self, provider_id: _Optional[str] = ..., model_name: _Optional[str] = ...) -> None: ...

class PullModelResponse(_message.Message):
    __slots__ = ("provider_id", "model_name", "dry_run", "message")
    PROVIDER_ID_FIELD_NUMBER: _ClassVar[int]
    MODEL_NAME_FIELD_NUMBER: _ClassVar[int]
    DRY_RUN_FIELD_NUMBER: _ClassVar[int]
    MESSAGE_FIELD_NUMBER: _ClassVar[int]
    provider_id: str
    model_name: str
    dry_run: bool
    message: str
    def __init__(self, provider_id: _Optional[str] = ..., model_name: _Optional[str] = ..., dry_run: _Optional[bool] = ..., message: _Optional[str] = ...) -> None: ...

class GetProviderLogsRequest(_message.Message):
    __slots__ = ("provider_id", "follow", "tail_lines")
    PROVIDER_ID_FIELD_NUMBER: _ClassVar[int]
    FOLLOW_FIELD_NUMBER: _ClassVar[int]
    TAIL_LINES_FIELD_NUMBER: _ClassVar[int]
    provider_id: str
    follow: bool
    tail_lines: int
    def __init__(self, provider_id: _Optional[str] = ..., follow: _Optional[bool] = ..., tail_lines: _Optional[int] = ...) -> None: ...

class LogLine(_message.Message):
    __slots__ = ("line", "ts_unix_ms", "stream")
    LINE_FIELD_NUMBER: _ClassVar[int]
    TS_UNIX_MS_FIELD_NUMBER: _ClassVar[int]
    STREAM_FIELD_NUMBER: _ClassVar[int]
    line: str
    ts_unix_ms: int
    stream: LogStream
    def __init__(self, line: _Optional[str] = ..., ts_unix_ms: _Optional[int] = ..., stream: _Optional[_Union[LogStream, str]] = ...) -> None: ...
