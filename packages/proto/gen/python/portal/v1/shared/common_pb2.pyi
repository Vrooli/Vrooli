from google.protobuf.internal import containers as _containers
from google.protobuf.internal import enum_type_wrapper as _enum_type_wrapper
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Iterable as _Iterable
from typing import ClassVar as _ClassVar, Optional as _Optional

DESCRIPTOR: _descriptor.FileDescriptor

class BehaviorMode(int, metaclass=_enum_type_wrapper.EnumTypeWrapper):
    __slots__ = ()
    BEHAVIOR_MODE_UNSPECIFIED: _ClassVar[BehaviorMode]
    BEHAVIOR_MODE_OFF: _ClassVar[BehaviorMode]
    BEHAVIOR_MODE_PASSIVE: _ClassVar[BehaviorMode]
    BEHAVIOR_MODE_FULL: _ClassVar[BehaviorMode]

class IntegrationState(int, metaclass=_enum_type_wrapper.EnumTypeWrapper):
    __slots__ = ()
    INTEGRATION_STATE_UNSPECIFIED: _ClassVar[IntegrationState]
    INTEGRATION_STATE_UNKNOWN: _ClassVar[IntegrationState]
    INTEGRATION_STATE_AVAILABLE: _ClassVar[IntegrationState]
    INTEGRATION_STATE_DEGRADED: _ClassVar[IntegrationState]
    INTEGRATION_STATE_UNAVAILABLE: _ClassVar[IntegrationState]

class AgentHarness(int, metaclass=_enum_type_wrapper.EnumTypeWrapper):
    __slots__ = ()
    AGENT_HARNESS_UNSPECIFIED: _ClassVar[AgentHarness]
    AGENT_HARNESS_CLAUDE_CODE: _ClassVar[AgentHarness]
    AGENT_HARNESS_CODEX: _ClassVar[AgentHarness]
    AGENT_HARNESS_OPENCODE: _ClassVar[AgentHarness]
    AGENT_HARNESS_GROK: _ClassVar[AgentHarness]

class ChatMode(int, metaclass=_enum_type_wrapper.EnumTypeWrapper):
    __slots__ = ()
    CHAT_MODE_UNSPECIFIED: _ClassVar[ChatMode]
    CHAT_MODE_LLM: _ClassVar[ChatMode]
    CHAT_MODE_AGENT: _ClassVar[ChatMode]
BEHAVIOR_MODE_UNSPECIFIED: BehaviorMode
BEHAVIOR_MODE_OFF: BehaviorMode
BEHAVIOR_MODE_PASSIVE: BehaviorMode
BEHAVIOR_MODE_FULL: BehaviorMode
INTEGRATION_STATE_UNSPECIFIED: IntegrationState
INTEGRATION_STATE_UNKNOWN: IntegrationState
INTEGRATION_STATE_AVAILABLE: IntegrationState
INTEGRATION_STATE_DEGRADED: IntegrationState
INTEGRATION_STATE_UNAVAILABLE: IntegrationState
AGENT_HARNESS_UNSPECIFIED: AgentHarness
AGENT_HARNESS_CLAUDE_CODE: AgentHarness
AGENT_HARNESS_CODEX: AgentHarness
AGENT_HARNESS_OPENCODE: AgentHarness
AGENT_HARNESS_GROK: AgentHarness
CHAT_MODE_UNSPECIFIED: ChatMode
CHAT_MODE_LLM: ChatMode
CHAT_MODE_AGENT: ChatMode

class SearchHit(_message.Message):
    __slots__ = ("provider_id", "type", "title", "snippet", "path", "score", "rerank_score", "locations")
    PROVIDER_ID_FIELD_NUMBER: _ClassVar[int]
    TYPE_FIELD_NUMBER: _ClassVar[int]
    TITLE_FIELD_NUMBER: _ClassVar[int]
    SNIPPET_FIELD_NUMBER: _ClassVar[int]
    PATH_FIELD_NUMBER: _ClassVar[int]
    SCORE_FIELD_NUMBER: _ClassVar[int]
    RERANK_SCORE_FIELD_NUMBER: _ClassVar[int]
    LOCATIONS_FIELD_NUMBER: _ClassVar[int]
    provider_id: str
    type: str
    title: str
    snippet: str
    path: str
    score: float
    rerank_score: float
    locations: _containers.RepeatedScalarFieldContainer[str]
    def __init__(self, provider_id: _Optional[str] = ..., type: _Optional[str] = ..., title: _Optional[str] = ..., snippet: _Optional[str] = ..., path: _Optional[str] = ..., score: _Optional[float] = ..., rerank_score: _Optional[float] = ..., locations: _Optional[_Iterable[str]] = ...) -> None: ...
