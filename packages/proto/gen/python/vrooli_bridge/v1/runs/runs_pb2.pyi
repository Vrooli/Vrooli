import datetime

from google.protobuf import timestamp_pb2 as _timestamp_pb2
from vrooli_bridge.v1.channel import channel_pb2 as _channel_pb2
from google.protobuf.internal import containers as _containers
from google.protobuf.internal import enum_type_wrapper as _enum_type_wrapper
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Iterable as _Iterable, Mapping as _Mapping
from typing import ClassVar as _ClassVar, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class RunStatus(int, metaclass=_enum_type_wrapper.EnumTypeWrapper):
    __slots__ = ()
    RUN_STATUS_UNSPECIFIED: _ClassVar[RunStatus]
    RUN_STATUS_QUEUED: _ClassVar[RunStatus]
    RUN_STATUS_RUNNING: _ClassVar[RunStatus]
    RUN_STATUS_PASSED: _ClassVar[RunStatus]
    RUN_STATUS_FAILED: _ClassVar[RunStatus]
    RUN_STATUS_ABORTED: _ClassVar[RunStatus]
RUN_STATUS_UNSPECIFIED: RunStatus
RUN_STATUS_QUEUED: RunStatus
RUN_STATUS_RUNNING: RunStatus
RUN_STATUS_PASSED: RunStatus
RUN_STATUS_FAILED: RunStatus
RUN_STATUS_ABORTED: RunStatus

class Run(_message.Message):
    __slots__ = ("id", "node_id", "scenario", "verb", "args", "status", "exit_code", "timeout_seconds", "created_at", "started_at", "finished_at", "artifact_refs")
    ID_FIELD_NUMBER: _ClassVar[int]
    NODE_ID_FIELD_NUMBER: _ClassVar[int]
    SCENARIO_FIELD_NUMBER: _ClassVar[int]
    VERB_FIELD_NUMBER: _ClassVar[int]
    ARGS_FIELD_NUMBER: _ClassVar[int]
    STATUS_FIELD_NUMBER: _ClassVar[int]
    EXIT_CODE_FIELD_NUMBER: _ClassVar[int]
    TIMEOUT_SECONDS_FIELD_NUMBER: _ClassVar[int]
    CREATED_AT_FIELD_NUMBER: _ClassVar[int]
    STARTED_AT_FIELD_NUMBER: _ClassVar[int]
    FINISHED_AT_FIELD_NUMBER: _ClassVar[int]
    ARTIFACT_REFS_FIELD_NUMBER: _ClassVar[int]
    id: str
    node_id: str
    scenario: str
    verb: str
    args: _containers.RepeatedScalarFieldContainer[str]
    status: RunStatus
    exit_code: int
    timeout_seconds: int
    created_at: _timestamp_pb2.Timestamp
    started_at: _timestamp_pb2.Timestamp
    finished_at: _timestamp_pb2.Timestamp
    artifact_refs: _containers.RepeatedScalarFieldContainer[str]
    def __init__(self, id: _Optional[str] = ..., node_id: _Optional[str] = ..., scenario: _Optional[str] = ..., verb: _Optional[str] = ..., args: _Optional[_Iterable[str]] = ..., status: _Optional[_Union[RunStatus, str]] = ..., exit_code: _Optional[int] = ..., timeout_seconds: _Optional[int] = ..., created_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., started_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., finished_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., artifact_refs: _Optional[_Iterable[str]] = ...) -> None: ...

class GetRunRequest(_message.Message):
    __slots__ = ("id",)
    ID_FIELD_NUMBER: _ClassVar[int]
    id: str
    def __init__(self, id: _Optional[str] = ...) -> None: ...

class GetRunResponse(_message.Message):
    __slots__ = ("run", "events")
    RUN_FIELD_NUMBER: _ClassVar[int]
    EVENTS_FIELD_NUMBER: _ClassVar[int]
    run: Run
    events: _containers.RepeatedCompositeFieldContainer[_channel_pb2.RunEvent]
    def __init__(self, run: _Optional[_Union[Run, _Mapping]] = ..., events: _Optional[_Iterable[_Union[_channel_pb2.RunEvent, _Mapping]]] = ...) -> None: ...

class ListRunsRequest(_message.Message):
    __slots__ = ("node_id", "limit")
    NODE_ID_FIELD_NUMBER: _ClassVar[int]
    LIMIT_FIELD_NUMBER: _ClassVar[int]
    node_id: str
    limit: int
    def __init__(self, node_id: _Optional[str] = ..., limit: _Optional[int] = ...) -> None: ...

class ListRunsResponse(_message.Message):
    __slots__ = ("runs",)
    RUNS_FIELD_NUMBER: _ClassVar[int]
    runs: _containers.RepeatedCompositeFieldContainer[Run]
    def __init__(self, runs: _Optional[_Iterable[_Union[Run, _Mapping]]] = ...) -> None: ...

class WaitRunRequest(_message.Message):
    __slots__ = ("id", "timeout_seconds")
    ID_FIELD_NUMBER: _ClassVar[int]
    TIMEOUT_SECONDS_FIELD_NUMBER: _ClassVar[int]
    id: str
    timeout_seconds: int
    def __init__(self, id: _Optional[str] = ..., timeout_seconds: _Optional[int] = ...) -> None: ...

class WaitRunResponse(_message.Message):
    __slots__ = ("run", "timed_out")
    RUN_FIELD_NUMBER: _ClassVar[int]
    TIMED_OUT_FIELD_NUMBER: _ClassVar[int]
    run: Run
    timed_out: bool
    def __init__(self, run: _Optional[_Union[Run, _Mapping]] = ..., timed_out: _Optional[bool] = ...) -> None: ...

class AbortRunRequest(_message.Message):
    __slots__ = ("id", "reason")
    ID_FIELD_NUMBER: _ClassVar[int]
    REASON_FIELD_NUMBER: _ClassVar[int]
    id: str
    reason: str
    def __init__(self, id: _Optional[str] = ..., reason: _Optional[str] = ...) -> None: ...

class AbortRunResponse(_message.Message):
    __slots__ = ("run",)
    RUN_FIELD_NUMBER: _ClassVar[int]
    run: Run
    def __init__(self, run: _Optional[_Union[Run, _Mapping]] = ...) -> None: ...

class StreamRunEventsRequest(_message.Message):
    __slots__ = ("id",)
    ID_FIELD_NUMBER: _ClassVar[int]
    id: str
    def __init__(self, id: _Optional[str] = ...) -> None: ...

class RunEventMessage(_message.Message):
    __slots__ = ("event",)
    EVENT_FIELD_NUMBER: _ClassVar[int]
    event: _channel_pb2.RunEvent
    def __init__(self, event: _Optional[_Union[_channel_pb2.RunEvent, _Mapping]] = ...) -> None: ...

class ReportRunEventRequest(_message.Message):
    __slots__ = ("event",)
    EVENT_FIELD_NUMBER: _ClassVar[int]
    event: _channel_pb2.RunEvent
    def __init__(self, event: _Optional[_Union[_channel_pb2.RunEvent, _Mapping]] = ...) -> None: ...

class ReportRunEventResponse(_message.Message):
    __slots__ = ("accepted",)
    ACCEPTED_FIELD_NUMBER: _ClassVar[int]
    accepted: bool
    def __init__(self, accepted: _Optional[bool] = ...) -> None: ...
