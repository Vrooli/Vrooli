import datetime

from google.protobuf import timestamp_pb2 as _timestamp_pb2
from google.protobuf.internal import containers as _containers
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Iterable as _Iterable, Mapping as _Mapping
from typing import ClassVar as _ClassVar, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class AdapterState(_message.Message):
    __slots__ = ("adapter_id", "kind", "risk_tier", "enabled", "last_run_at", "last_error", "disabled_reason")
    ADAPTER_ID_FIELD_NUMBER: _ClassVar[int]
    KIND_FIELD_NUMBER: _ClassVar[int]
    RISK_TIER_FIELD_NUMBER: _ClassVar[int]
    ENABLED_FIELD_NUMBER: _ClassVar[int]
    LAST_RUN_AT_FIELD_NUMBER: _ClassVar[int]
    LAST_ERROR_FIELD_NUMBER: _ClassVar[int]
    DISABLED_REASON_FIELD_NUMBER: _ClassVar[int]
    adapter_id: str
    kind: str
    risk_tier: int
    enabled: bool
    last_run_at: _timestamp_pb2.Timestamp
    last_error: str
    disabled_reason: str
    def __init__(self, adapter_id: _Optional[str] = ..., kind: _Optional[str] = ..., risk_tier: _Optional[int] = ..., enabled: _Optional[bool] = ..., last_run_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., last_error: _Optional[str] = ..., disabled_reason: _Optional[str] = ...) -> None: ...

class ImportResult(_message.Message):
    __slots__ = ("run_id", "adapter_id", "created", "duplicated", "failed")
    RUN_ID_FIELD_NUMBER: _ClassVar[int]
    ADAPTER_ID_FIELD_NUMBER: _ClassVar[int]
    CREATED_FIELD_NUMBER: _ClassVar[int]
    DUPLICATED_FIELD_NUMBER: _ClassVar[int]
    FAILED_FIELD_NUMBER: _ClassVar[int]
    run_id: str
    adapter_id: str
    created: int
    duplicated: int
    failed: int
    def __init__(self, run_id: _Optional[str] = ..., adapter_id: _Optional[str] = ..., created: _Optional[int] = ..., duplicated: _Optional[int] = ..., failed: _Optional[int] = ...) -> None: ...

class ListAdaptersRequest(_message.Message):
    __slots__ = ()
    def __init__(self) -> None: ...

class ListAdaptersResponse(_message.Message):
    __slots__ = ("adapters",)
    ADAPTERS_FIELD_NUMBER: _ClassVar[int]
    adapters: _containers.RepeatedCompositeFieldContainer[AdapterState]
    def __init__(self, adapters: _Optional[_Iterable[_Union[AdapterState, _Mapping]]] = ...) -> None: ...

class SetAdapterEnabledRequest(_message.Message):
    __slots__ = ("adapter_id", "enabled")
    ADAPTER_ID_FIELD_NUMBER: _ClassVar[int]
    ENABLED_FIELD_NUMBER: _ClassVar[int]
    adapter_id: str
    enabled: bool
    def __init__(self, adapter_id: _Optional[str] = ..., enabled: _Optional[bool] = ...) -> None: ...

class SetAdapterEnabledResponse(_message.Message):
    __slots__ = ("adapter",)
    ADAPTER_FIELD_NUMBER: _ClassVar[int]
    adapter: AdapterState
    def __init__(self, adapter: _Optional[_Union[AdapterState, _Mapping]] = ...) -> None: ...

class ImportArchiveRequest(_message.Message):
    __slots__ = ("adapter_id", "content")
    ADAPTER_ID_FIELD_NUMBER: _ClassVar[int]
    CONTENT_FIELD_NUMBER: _ClassVar[int]
    adapter_id: str
    content: bytes
    def __init__(self, adapter_id: _Optional[str] = ..., content: _Optional[bytes] = ...) -> None: ...

class ImportArchiveResponse(_message.Message):
    __slots__ = ("result",)
    RESULT_FIELD_NUMBER: _ClassVar[int]
    result: ImportResult
    def __init__(self, result: _Optional[_Union[ImportResult, _Mapping]] = ...) -> None: ...
