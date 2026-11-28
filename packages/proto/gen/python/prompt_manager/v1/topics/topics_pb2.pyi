from google.protobuf import field_mask_pb2 as _field_mask_pb2
from google.protobuf.internal import containers as _containers
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Iterable as _Iterable, Mapping as _Mapping
from typing import ClassVar as _ClassVar, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class Topic(_message.Message):
    __slots__ = ("id", "name", "description", "parent_topic_id", "skills", "icon", "status", "content", "created_at", "updated_at")
    ID_FIELD_NUMBER: _ClassVar[int]
    NAME_FIELD_NUMBER: _ClassVar[int]
    DESCRIPTION_FIELD_NUMBER: _ClassVar[int]
    PARENT_TOPIC_ID_FIELD_NUMBER: _ClassVar[int]
    SKILLS_FIELD_NUMBER: _ClassVar[int]
    ICON_FIELD_NUMBER: _ClassVar[int]
    STATUS_FIELD_NUMBER: _ClassVar[int]
    CONTENT_FIELD_NUMBER: _ClassVar[int]
    CREATED_AT_FIELD_NUMBER: _ClassVar[int]
    UPDATED_AT_FIELD_NUMBER: _ClassVar[int]
    id: str
    name: str
    description: str
    parent_topic_id: str
    skills: _containers.RepeatedScalarFieldContainer[str]
    icon: str
    status: str
    content: str
    created_at: str
    updated_at: str
    def __init__(self, id: _Optional[str] = ..., name: _Optional[str] = ..., description: _Optional[str] = ..., parent_topic_id: _Optional[str] = ..., skills: _Optional[_Iterable[str]] = ..., icon: _Optional[str] = ..., status: _Optional[str] = ..., content: _Optional[str] = ..., created_at: _Optional[str] = ..., updated_at: _Optional[str] = ...) -> None: ...

class TopicInput(_message.Message):
    __slots__ = ("id", "name", "description", "parent_topic_id", "skills", "icon", "status", "content")
    ID_FIELD_NUMBER: _ClassVar[int]
    NAME_FIELD_NUMBER: _ClassVar[int]
    DESCRIPTION_FIELD_NUMBER: _ClassVar[int]
    PARENT_TOPIC_ID_FIELD_NUMBER: _ClassVar[int]
    SKILLS_FIELD_NUMBER: _ClassVar[int]
    ICON_FIELD_NUMBER: _ClassVar[int]
    STATUS_FIELD_NUMBER: _ClassVar[int]
    CONTENT_FIELD_NUMBER: _ClassVar[int]
    id: str
    name: str
    description: str
    parent_topic_id: str
    skills: _containers.RepeatedScalarFieldContainer[str]
    icon: str
    status: str
    content: str
    def __init__(self, id: _Optional[str] = ..., name: _Optional[str] = ..., description: _Optional[str] = ..., parent_topic_id: _Optional[str] = ..., skills: _Optional[_Iterable[str]] = ..., icon: _Optional[str] = ..., status: _Optional[str] = ..., content: _Optional[str] = ...) -> None: ...

class ListTopicsRequest(_message.Message):
    __slots__ = ()
    def __init__(self) -> None: ...

class ListTopicTreeRequest(_message.Message):
    __slots__ = ()
    def __init__(self) -> None: ...

class ListTopicsResponse(_message.Message):
    __slots__ = ("topics",)
    TOPICS_FIELD_NUMBER: _ClassVar[int]
    topics: _containers.RepeatedCompositeFieldContainer[Topic]
    def __init__(self, topics: _Optional[_Iterable[_Union[Topic, _Mapping]]] = ...) -> None: ...

class GetTopicRequest(_message.Message):
    __slots__ = ("id",)
    ID_FIELD_NUMBER: _ClassVar[int]
    id: str
    def __init__(self, id: _Optional[str] = ...) -> None: ...

class CreateTopicRequest(_message.Message):
    __slots__ = ("topic",)
    TOPIC_FIELD_NUMBER: _ClassVar[int]
    topic: TopicInput
    def __init__(self, topic: _Optional[_Union[TopicInput, _Mapping]] = ...) -> None: ...

class UpdateTopicRequest(_message.Message):
    __slots__ = ("id", "topic", "update_mask")
    ID_FIELD_NUMBER: _ClassVar[int]
    TOPIC_FIELD_NUMBER: _ClassVar[int]
    UPDATE_MASK_FIELD_NUMBER: _ClassVar[int]
    id: str
    topic: TopicInput
    update_mask: _field_mask_pb2.FieldMask
    def __init__(self, id: _Optional[str] = ..., topic: _Optional[_Union[TopicInput, _Mapping]] = ..., update_mask: _Optional[_Union[_field_mask_pb2.FieldMask, _Mapping]] = ...) -> None: ...

class DeleteTopicRequest(_message.Message):
    __slots__ = ("id",)
    ID_FIELD_NUMBER: _ClassVar[int]
    id: str
    def __init__(self, id: _Optional[str] = ...) -> None: ...

class DeleteTopicResponse(_message.Message):
    __slots__ = ()
    def __init__(self) -> None: ...

class GetAccumulatedSkillsRequest(_message.Message):
    __slots__ = ("id",)
    ID_FIELD_NUMBER: _ClassVar[int]
    id: str
    def __init__(self, id: _Optional[str] = ...) -> None: ...

class AccumulatedSkillsResponse(_message.Message):
    __slots__ = ("topic_id", "ancestry", "skills")
    TOPIC_ID_FIELD_NUMBER: _ClassVar[int]
    ANCESTRY_FIELD_NUMBER: _ClassVar[int]
    SKILLS_FIELD_NUMBER: _ClassVar[int]
    topic_id: str
    ancestry: _containers.RepeatedScalarFieldContainer[str]
    skills: _containers.RepeatedScalarFieldContainer[str]
    def __init__(self, topic_id: _Optional[str] = ..., ancestry: _Optional[_Iterable[str]] = ..., skills: _Optional[_Iterable[str]] = ...) -> None: ...

class MatchTopicsRequest(_message.Message):
    __slots__ = ("queries", "limit")
    QUERIES_FIELD_NUMBER: _ClassVar[int]
    LIMIT_FIELD_NUMBER: _ClassVar[int]
    queries: _containers.RepeatedScalarFieldContainer[str]
    limit: int
    def __init__(self, queries: _Optional[_Iterable[str]] = ..., limit: _Optional[int] = ...) -> None: ...

class MatchedTopic(_message.Message):
    __slots__ = ("id", "name", "description", "parent_topic_id", "score", "score_percent")
    ID_FIELD_NUMBER: _ClassVar[int]
    NAME_FIELD_NUMBER: _ClassVar[int]
    DESCRIPTION_FIELD_NUMBER: _ClassVar[int]
    PARENT_TOPIC_ID_FIELD_NUMBER: _ClassVar[int]
    SCORE_FIELD_NUMBER: _ClassVar[int]
    SCORE_PERCENT_FIELD_NUMBER: _ClassVar[int]
    id: str
    name: str
    description: str
    parent_topic_id: str
    score: float
    score_percent: int
    def __init__(self, id: _Optional[str] = ..., name: _Optional[str] = ..., description: _Optional[str] = ..., parent_topic_id: _Optional[str] = ..., score: _Optional[float] = ..., score_percent: _Optional[int] = ...) -> None: ...

class MatchTopicsResponse(_message.Message):
    __slots__ = ("topics", "skills", "method")
    TOPICS_FIELD_NUMBER: _ClassVar[int]
    SKILLS_FIELD_NUMBER: _ClassVar[int]
    METHOD_FIELD_NUMBER: _ClassVar[int]
    topics: _containers.RepeatedCompositeFieldContainer[MatchedTopic]
    skills: _containers.RepeatedScalarFieldContainer[str]
    method: str
    def __init__(self, topics: _Optional[_Iterable[_Union[MatchedTopic, _Mapping]]] = ..., skills: _Optional[_Iterable[str]] = ..., method: _Optional[str] = ...) -> None: ...
