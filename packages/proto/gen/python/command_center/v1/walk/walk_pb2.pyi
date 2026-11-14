from google.protobuf import struct_pb2 as _struct_pb2
from google.protobuf.internal import containers as _containers
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Iterable as _Iterable, Mapping as _Mapping
from typing import ClassVar as _ClassVar, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class ReadRequest(_message.Message):
    __slots__ = ("limit",)
    LIMIT_FIELD_NUMBER: _ClassVar[int]
    limit: int
    def __init__(self, limit: _Optional[int] = ...) -> None: ...

class Reading(_message.Message):
    __slots__ = ("id", "label", "owner", "source", "coverage", "trust", "empirical", "value", "unit", "observed_at", "ttl_seconds", "reason")
    ID_FIELD_NUMBER: _ClassVar[int]
    LABEL_FIELD_NUMBER: _ClassVar[int]
    OWNER_FIELD_NUMBER: _ClassVar[int]
    SOURCE_FIELD_NUMBER: _ClassVar[int]
    COVERAGE_FIELD_NUMBER: _ClassVar[int]
    TRUST_FIELD_NUMBER: _ClassVar[int]
    EMPIRICAL_FIELD_NUMBER: _ClassVar[int]
    VALUE_FIELD_NUMBER: _ClassVar[int]
    UNIT_FIELD_NUMBER: _ClassVar[int]
    OBSERVED_AT_FIELD_NUMBER: _ClassVar[int]
    TTL_SECONDS_FIELD_NUMBER: _ClassVar[int]
    REASON_FIELD_NUMBER: _ClassVar[int]
    id: str
    label: str
    owner: str
    source: str
    coverage: str
    trust: str
    empirical: str
    value: _struct_pb2.Value
    unit: str
    observed_at: str
    ttl_seconds: int
    reason: str
    def __init__(self, id: _Optional[str] = ..., label: _Optional[str] = ..., owner: _Optional[str] = ..., source: _Optional[str] = ..., coverage: _Optional[str] = ..., trust: _Optional[str] = ..., empirical: _Optional[str] = ..., value: _Optional[_Union[_struct_pb2.Value, _Mapping]] = ..., unit: _Optional[str] = ..., observed_at: _Optional[str] = ..., ttl_seconds: _Optional[int] = ..., reason: _Optional[str] = ...) -> None: ...

class ReadResponse(_message.Message):
    __slots__ = ("readings", "generated_at", "total", "truncated")
    READINGS_FIELD_NUMBER: _ClassVar[int]
    GENERATED_AT_FIELD_NUMBER: _ClassVar[int]
    TOTAL_FIELD_NUMBER: _ClassVar[int]
    TRUNCATED_FIELD_NUMBER: _ClassVar[int]
    readings: _containers.RepeatedCompositeFieldContainer[Reading]
    generated_at: str
    total: int
    truncated: bool
    def __init__(self, readings: _Optional[_Iterable[_Union[Reading, _Mapping]]] = ..., generated_at: _Optional[str] = ..., total: _Optional[int] = ..., truncated: _Optional[bool] = ...) -> None: ...

class StateRequest(_message.Message):
    __slots__ = ("channel",)
    CHANNEL_FIELD_NUMBER: _ClassVar[int]
    channel: str
    def __init__(self, channel: _Optional[str] = ...) -> None: ...

class StoredRecord(_message.Message):
    __slots__ = ("entry_id", "body", "created_at")
    ENTRY_ID_FIELD_NUMBER: _ClassVar[int]
    BODY_FIELD_NUMBER: _ClassVar[int]
    CREATED_AT_FIELD_NUMBER: _ClassVar[int]
    entry_id: str
    body: str
    created_at: str
    def __init__(self, entry_id: _Optional[str] = ..., body: _Optional[str] = ..., created_at: _Optional[str] = ...) -> None: ...

class StateResponse(_message.Message):
    __slots__ = ("briefing", "checkpoint")
    BRIEFING_FIELD_NUMBER: _ClassVar[int]
    CHECKPOINT_FIELD_NUMBER: _ClassVar[int]
    briefing: StoredRecord
    checkpoint: StoredRecord
    def __init__(self, briefing: _Optional[_Union[StoredRecord, _Mapping]] = ..., checkpoint: _Optional[_Union[StoredRecord, _Mapping]] = ...) -> None: ...

class PublishRequest(_message.Message):
    __slots__ = ("channel", "request_key", "expected_previous_id", "program_id", "envelope_json", "briefing", "fleet_health_json")
    CHANNEL_FIELD_NUMBER: _ClassVar[int]
    REQUEST_KEY_FIELD_NUMBER: _ClassVar[int]
    EXPECTED_PREVIOUS_ID_FIELD_NUMBER: _ClassVar[int]
    PROGRAM_ID_FIELD_NUMBER: _ClassVar[int]
    ENVELOPE_JSON_FIELD_NUMBER: _ClassVar[int]
    BRIEFING_FIELD_NUMBER: _ClassVar[int]
    FLEET_HEALTH_JSON_FIELD_NUMBER: _ClassVar[int]
    channel: str
    request_key: str
    expected_previous_id: str
    program_id: str
    envelope_json: str
    briefing: str
    fleet_health_json: str
    def __init__(self, channel: _Optional[str] = ..., request_key: _Optional[str] = ..., expected_previous_id: _Optional[str] = ..., program_id: _Optional[str] = ..., envelope_json: _Optional[str] = ..., briefing: _Optional[str] = ..., fleet_health_json: _Optional[str] = ...) -> None: ...

class CheckpointRequest(_message.Message):
    __slots__ = ("channel", "request_key", "expected_previous_id", "walk_id", "state", "resume_phase", "content")
    CHANNEL_FIELD_NUMBER: _ClassVar[int]
    REQUEST_KEY_FIELD_NUMBER: _ClassVar[int]
    EXPECTED_PREVIOUS_ID_FIELD_NUMBER: _ClassVar[int]
    WALK_ID_FIELD_NUMBER: _ClassVar[int]
    STATE_FIELD_NUMBER: _ClassVar[int]
    RESUME_PHASE_FIELD_NUMBER: _ClassVar[int]
    CONTENT_FIELD_NUMBER: _ClassVar[int]
    channel: str
    request_key: str
    expected_previous_id: str
    walk_id: str
    state: str
    resume_phase: str
    content: str
    def __init__(self, channel: _Optional[str] = ..., request_key: _Optional[str] = ..., expected_previous_id: _Optional[str] = ..., walk_id: _Optional[str] = ..., state: _Optional[str] = ..., resume_phase: _Optional[str] = ..., content: _Optional[str] = ...) -> None: ...

class Receipt(_message.Message):
    __slots__ = ("entry_id", "existing", "created_at", "channel")
    ENTRY_ID_FIELD_NUMBER: _ClassVar[int]
    EXISTING_FIELD_NUMBER: _ClassVar[int]
    CREATED_AT_FIELD_NUMBER: _ClassVar[int]
    CHANNEL_FIELD_NUMBER: _ClassVar[int]
    entry_id: str
    existing: bool
    created_at: str
    channel: str
    def __init__(self, entry_id: _Optional[str] = ..., existing: _Optional[bool] = ..., created_at: _Optional[str] = ..., channel: _Optional[str] = ...) -> None: ...
