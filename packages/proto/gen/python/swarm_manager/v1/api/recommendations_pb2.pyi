from buf.validate import validate_pb2 as _validate_pb2
from swarm_manager.v1.domain import recommendation_pb2 as _recommendation_pb2
from google.protobuf.internal import containers as _containers
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Iterable as _Iterable, Mapping as _Mapping
from typing import ClassVar as _ClassVar, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class ListRecommendationsResponse(_message.Message):
    __slots__ = ("recommendations",)
    RECOMMENDATIONS_FIELD_NUMBER: _ClassVar[int]
    recommendations: _containers.RepeatedCompositeFieldContainer[_recommendation_pb2.Recommendation]
    def __init__(self, recommendations: _Optional[_Iterable[_Union[_recommendation_pb2.Recommendation, _Mapping]]] = ...) -> None: ...

class RecommendationResponse(_message.Message):
    __slots__ = ("recommendation",)
    RECOMMENDATION_FIELD_NUMBER: _ClassVar[int]
    recommendation: _recommendation_pb2.Recommendation
    def __init__(self, recommendation: _Optional[_Union[_recommendation_pb2.Recommendation, _Mapping]] = ...) -> None: ...

class CreateRecommendationRequest(_message.Message):
    __slots__ = ("scenario_name", "type", "description", "priority")
    SCENARIO_NAME_FIELD_NUMBER: _ClassVar[int]
    TYPE_FIELD_NUMBER: _ClassVar[int]
    DESCRIPTION_FIELD_NUMBER: _ClassVar[int]
    PRIORITY_FIELD_NUMBER: _ClassVar[int]
    scenario_name: str
    type: str
    description: str
    priority: int
    def __init__(self, scenario_name: _Optional[str] = ..., type: _Optional[str] = ..., description: _Optional[str] = ..., priority: _Optional[int] = ...) -> None: ...

class UpdateRecommendationRequest(_message.Message):
    __slots__ = ("status",)
    STATUS_FIELD_NUMBER: _ClassVar[int]
    status: str
    def __init__(self, status: _Optional[str] = ...) -> None: ...

class StartRecommendationRequest(_message.Message):
    __slots__ = ("prompt", "scope_path", "project_root", "created_by")
    PROMPT_FIELD_NUMBER: _ClassVar[int]
    SCOPE_PATH_FIELD_NUMBER: _ClassVar[int]
    PROJECT_ROOT_FIELD_NUMBER: _ClassVar[int]
    CREATED_BY_FIELD_NUMBER: _ClassVar[int]
    prompt: str
    scope_path: str
    project_root: str
    created_by: str
    def __init__(self, prompt: _Optional[str] = ..., scope_path: _Optional[str] = ..., project_root: _Optional[str] = ..., created_by: _Optional[str] = ...) -> None: ...
