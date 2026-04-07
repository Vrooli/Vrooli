from buf.validate import validate_pb2 as _validate_pb2
from swarm_manager.v1.domain import scenario_pb2 as _scenario_pb2
from google.protobuf.internal import containers as _containers
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Iterable as _Iterable, Mapping as _Mapping
from typing import ClassVar as _ClassVar, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class ListScenariosResponse(_message.Message):
    __slots__ = ("scenarios",)
    SCENARIOS_FIELD_NUMBER: _ClassVar[int]
    scenarios: _containers.RepeatedCompositeFieldContainer[_scenario_pb2.Scenario]
    def __init__(self, scenarios: _Optional[_Iterable[_Union[_scenario_pb2.Scenario, _Mapping]]] = ...) -> None: ...

class ScenarioResponse(_message.Message):
    __slots__ = ("scenario",)
    SCENARIO_FIELD_NUMBER: _ClassVar[int]
    scenario: _scenario_pb2.Scenario
    def __init__(self, scenario: _Optional[_Union[_scenario_pb2.Scenario, _Mapping]] = ...) -> None: ...

class UpdateScenarioMetadataRequest(_message.Message):
    __slots__ = ("is_greenfield",)
    IS_GREENFIELD_FIELD_NUMBER: _ClassVar[int]
    is_greenfield: bool
    def __init__(self, is_greenfield: _Optional[bool] = ...) -> None: ...

class PreserveFilesRequest(_message.Message):
    __slots__ = ("paths", "preset")
    PATHS_FIELD_NUMBER: _ClassVar[int]
    PRESET_FIELD_NUMBER: _ClassVar[int]
    paths: _containers.RepeatedScalarFieldContainer[str]
    preset: str
    def __init__(self, paths: _Optional[_Iterable[str]] = ..., preset: _Optional[str] = ...) -> None: ...

class DeleteScenarioRequest(_message.Message):
    __slots__ = ("preserve_files",)
    PRESERVE_FILES_FIELD_NUMBER: _ClassVar[int]
    preserve_files: PreserveFilesRequest
    def __init__(self, preserve_files: _Optional[_Union[PreserveFilesRequest, _Mapping]] = ...) -> None: ...

class DeleteScenarioResponse(_message.Message):
    __slots__ = ("name", "archived", "message", "backlog_idea_name", "preserved_files")
    NAME_FIELD_NUMBER: _ClassVar[int]
    ARCHIVED_FIELD_NUMBER: _ClassVar[int]
    MESSAGE_FIELD_NUMBER: _ClassVar[int]
    BACKLOG_IDEA_NAME_FIELD_NUMBER: _ClassVar[int]
    PRESERVED_FILES_FIELD_NUMBER: _ClassVar[int]
    name: str
    archived: bool
    message: str
    backlog_idea_name: str
    preserved_files: _containers.RepeatedScalarFieldContainer[str]
    def __init__(self, name: _Optional[str] = ..., archived: _Optional[bool] = ..., message: _Optional[str] = ..., backlog_idea_name: _Optional[str] = ..., preserved_files: _Optional[_Iterable[str]] = ...) -> None: ...

class ScenarioFile(_message.Message):
    __slots__ = ("name", "path", "type", "size", "children")
    NAME_FIELD_NUMBER: _ClassVar[int]
    PATH_FIELD_NUMBER: _ClassVar[int]
    TYPE_FIELD_NUMBER: _ClassVar[int]
    SIZE_FIELD_NUMBER: _ClassVar[int]
    CHILDREN_FIELD_NUMBER: _ClassVar[int]
    name: str
    path: str
    type: str
    size: int
    children: _containers.RepeatedCompositeFieldContainer[ScenarioFile]
    def __init__(self, name: _Optional[str] = ..., path: _Optional[str] = ..., type: _Optional[str] = ..., size: _Optional[int] = ..., children: _Optional[_Iterable[_Union[ScenarioFile, _Mapping]]] = ...) -> None: ...

class ScenarioFilesResponse(_message.Message):
    __slots__ = ("files",)
    FILES_FIELD_NUMBER: _ClassVar[int]
    files: _containers.RepeatedCompositeFieldContainer[ScenarioFile]
    def __init__(self, files: _Optional[_Iterable[_Union[ScenarioFile, _Mapping]]] = ...) -> None: ...

class SpecSyncArchiveRequest(_message.Message):
    __slots__ = ("preserve_files",)
    PRESERVE_FILES_FIELD_NUMBER: _ClassVar[int]
    preserve_files: PreserveFilesRequest
    def __init__(self, preserve_files: _Optional[_Union[PreserveFilesRequest, _Mapping]] = ...) -> None: ...

class SpecSyncArchiveResponse(_message.Message):
    __slots__ = ("execution_id", "status", "message")
    EXECUTION_ID_FIELD_NUMBER: _ClassVar[int]
    STATUS_FIELD_NUMBER: _ClassVar[int]
    MESSAGE_FIELD_NUMBER: _ClassVar[int]
    execution_id: str
    status: str
    message: str
    def __init__(self, execution_id: _Optional[str] = ..., status: _Optional[str] = ..., message: _Optional[str] = ...) -> None: ...

class ScenarioReviewQueueRequest(_message.Message):
    __slots__ = ("limit", "exclude_tag")
    LIMIT_FIELD_NUMBER: _ClassVar[int]
    EXCLUDE_TAG_FIELD_NUMBER: _ClassVar[int]
    limit: int
    exclude_tag: str
    def __init__(self, limit: _Optional[int] = ..., exclude_tag: _Optional[str] = ...) -> None: ...

class ScenarioReviewQueueItem(_message.Message):
    __slots__ = ("scenario_name", "pending_backlog_count", "last_review_classification", "last_review_at", "recent_execution_count", "composite_score", "primary_signal", "cooldown_until")
    SCENARIO_NAME_FIELD_NUMBER: _ClassVar[int]
    PENDING_BACKLOG_COUNT_FIELD_NUMBER: _ClassVar[int]
    LAST_REVIEW_CLASSIFICATION_FIELD_NUMBER: _ClassVar[int]
    LAST_REVIEW_AT_FIELD_NUMBER: _ClassVar[int]
    RECENT_EXECUTION_COUNT_FIELD_NUMBER: _ClassVar[int]
    COMPOSITE_SCORE_FIELD_NUMBER: _ClassVar[int]
    PRIMARY_SIGNAL_FIELD_NUMBER: _ClassVar[int]
    COOLDOWN_UNTIL_FIELD_NUMBER: _ClassVar[int]
    scenario_name: str
    pending_backlog_count: int
    last_review_classification: str
    last_review_at: str
    recent_execution_count: int
    composite_score: float
    primary_signal: str
    cooldown_until: str
    def __init__(self, scenario_name: _Optional[str] = ..., pending_backlog_count: _Optional[int] = ..., last_review_classification: _Optional[str] = ..., last_review_at: _Optional[str] = ..., recent_execution_count: _Optional[int] = ..., composite_score: _Optional[float] = ..., primary_signal: _Optional[str] = ..., cooldown_until: _Optional[str] = ...) -> None: ...

class ScenarioReviewQueueResponse(_message.Message):
    __slots__ = ("items", "total_scenarios", "excluded_count")
    ITEMS_FIELD_NUMBER: _ClassVar[int]
    TOTAL_SCENARIOS_FIELD_NUMBER: _ClassVar[int]
    EXCLUDED_COUNT_FIELD_NUMBER: _ClassVar[int]
    items: _containers.RepeatedCompositeFieldContainer[ScenarioReviewQueueItem]
    total_scenarios: int
    excluded_count: int
    def __init__(self, items: _Optional[_Iterable[_Union[ScenarioReviewQueueItem, _Mapping]]] = ..., total_scenarios: _Optional[int] = ..., excluded_count: _Optional[int] = ...) -> None: ...
