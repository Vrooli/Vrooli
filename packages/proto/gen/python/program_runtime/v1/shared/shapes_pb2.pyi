from google.protobuf.internal import containers as _containers
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Iterable as _Iterable
from typing import ClassVar as _ClassVar, Optional as _Optional

DESCRIPTOR: _descriptor.FileDescriptor

class ProgramShape(_message.Message):
    __slots__ = ("shape_key", "binding_ids", "binding_count", "occurrences", "agent_runs", "operator_runs", "test_runs", "replay_runs", "sessions", "first_seen", "last_seen", "exemplar_program_id", "exemplar_bytes", "covered_by", "covered_reason", "state", "dominant_scenario")
    SHAPE_KEY_FIELD_NUMBER: _ClassVar[int]
    BINDING_IDS_FIELD_NUMBER: _ClassVar[int]
    BINDING_COUNT_FIELD_NUMBER: _ClassVar[int]
    OCCURRENCES_FIELD_NUMBER: _ClassVar[int]
    AGENT_RUNS_FIELD_NUMBER: _ClassVar[int]
    OPERATOR_RUNS_FIELD_NUMBER: _ClassVar[int]
    TEST_RUNS_FIELD_NUMBER: _ClassVar[int]
    REPLAY_RUNS_FIELD_NUMBER: _ClassVar[int]
    SESSIONS_FIELD_NUMBER: _ClassVar[int]
    FIRST_SEEN_FIELD_NUMBER: _ClassVar[int]
    LAST_SEEN_FIELD_NUMBER: _ClassVar[int]
    EXEMPLAR_PROGRAM_ID_FIELD_NUMBER: _ClassVar[int]
    EXEMPLAR_BYTES_FIELD_NUMBER: _ClassVar[int]
    COVERED_BY_FIELD_NUMBER: _ClassVar[int]
    COVERED_REASON_FIELD_NUMBER: _ClassVar[int]
    STATE_FIELD_NUMBER: _ClassVar[int]
    DOMINANT_SCENARIO_FIELD_NUMBER: _ClassVar[int]
    shape_key: str
    binding_ids: _containers.RepeatedScalarFieldContainer[str]
    binding_count: int
    occurrences: int
    agent_runs: int
    operator_runs: int
    test_runs: int
    replay_runs: int
    sessions: int
    first_seen: str
    last_seen: str
    exemplar_program_id: str
    exemplar_bytes: int
    covered_by: str
    covered_reason: str
    state: str
    dominant_scenario: str
    def __init__(self, shape_key: _Optional[str] = ..., binding_ids: _Optional[_Iterable[str]] = ..., binding_count: _Optional[int] = ..., occurrences: _Optional[int] = ..., agent_runs: _Optional[int] = ..., operator_runs: _Optional[int] = ..., test_runs: _Optional[int] = ..., replay_runs: _Optional[int] = ..., sessions: _Optional[int] = ..., first_seen: _Optional[str] = ..., last_seen: _Optional[str] = ..., exemplar_program_id: _Optional[str] = ..., exemplar_bytes: _Optional[int] = ..., covered_by: _Optional[str] = ..., covered_reason: _Optional[str] = ..., state: _Optional[str] = ..., dominant_scenario: _Optional[str] = ...) -> None: ...
