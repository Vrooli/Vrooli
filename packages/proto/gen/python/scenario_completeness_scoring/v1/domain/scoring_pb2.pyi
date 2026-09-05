import datetime

from google.protobuf import timestamp_pb2 as _timestamp_pb2
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Mapping as _Mapping
from typing import ClassVar as _ClassVar, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class ScoreSnapshot(_message.Message):
    __slots__ = ("scenario", "category", "digest", "composite", "classification", "working_rung", "breakdown_json", "importance", "source", "created_at")
    SCENARIO_FIELD_NUMBER: _ClassVar[int]
    CATEGORY_FIELD_NUMBER: _ClassVar[int]
    DIGEST_FIELD_NUMBER: _ClassVar[int]
    COMPOSITE_FIELD_NUMBER: _ClassVar[int]
    CLASSIFICATION_FIELD_NUMBER: _ClassVar[int]
    WORKING_RUNG_FIELD_NUMBER: _ClassVar[int]
    BREAKDOWN_JSON_FIELD_NUMBER: _ClassVar[int]
    IMPORTANCE_FIELD_NUMBER: _ClassVar[int]
    SOURCE_FIELD_NUMBER: _ClassVar[int]
    CREATED_AT_FIELD_NUMBER: _ClassVar[int]
    scenario: str
    category: str
    digest: str
    composite: int
    classification: str
    working_rung: str
    breakdown_json: str
    importance: float
    source: str
    created_at: _timestamp_pb2.Timestamp
    def __init__(self, scenario: _Optional[str] = ..., category: _Optional[str] = ..., digest: _Optional[str] = ..., composite: _Optional[int] = ..., classification: _Optional[str] = ..., working_rung: _Optional[str] = ..., breakdown_json: _Optional[str] = ..., importance: _Optional[float] = ..., source: _Optional[str] = ..., created_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ...) -> None: ...
