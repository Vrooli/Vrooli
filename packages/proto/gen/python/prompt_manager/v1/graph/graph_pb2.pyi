from google.protobuf.internal import containers as _containers
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Iterable as _Iterable, Mapping as _Mapping
from typing import ClassVar as _ClassVar, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class GetGraphRequest(_message.Message):
    __slots__ = ()
    def __init__(self) -> None: ...

class RegenerateGraphRequest(_message.Message):
    __slots__ = ()
    def __init__(self) -> None: ...

class ListNodesRequest(_message.Message):
    __slots__ = ()
    def __init__(self) -> None: ...

class ListPopularNodesRequest(_message.Message):
    __slots__ = ("limit",)
    LIMIT_FIELD_NUMBER: _ClassVar[int]
    limit: int
    def __init__(self, limit: _Optional[int] = ...) -> None: ...

class ListCyclesRequest(_message.Message):
    __slots__ = ()
    def __init__(self) -> None: ...

class GetNodeRequest(_message.Message):
    __slots__ = ("node_id",)
    NODE_ID_FIELD_NUMBER: _ClassVar[int]
    node_id: str
    def __init__(self, node_id: _Optional[str] = ...) -> None: ...

class ListNodeEdgesRequest(_message.Message):
    __slots__ = ("node_id",)
    NODE_ID_FIELD_NUMBER: _ClassVar[int]
    node_id: str
    def __init__(self, node_id: _Optional[str] = ...) -> None: ...

class GetHealthConfigRequest(_message.Message):
    __slots__ = ()
    def __init__(self) -> None: ...

class UpdateHealthConfigRequest(_message.Message):
    __slots__ = ("config",)
    CONFIG_FIELD_NUMBER: _ClassVar[int]
    config: HealthConfig
    def __init__(self, config: _Optional[_Union[HealthConfig, _Mapping]] = ...) -> None: ...

class GraphIndex(_message.Message):
    __slots__ = ("generated_at", "graph")
    GENERATED_AT_FIELD_NUMBER: _ClassVar[int]
    GRAPH_FIELD_NUMBER: _ClassVar[int]
    generated_at: str
    graph: Graph
    def __init__(self, generated_at: _Optional[str] = ..., graph: _Optional[_Union[Graph, _Mapping]] = ...) -> None: ...

class Graph(_message.Message):
    __slots__ = ("nodes", "edges", "health_scores")
    NODES_FIELD_NUMBER: _ClassVar[int]
    EDGES_FIELD_NUMBER: _ClassVar[int]
    HEALTH_SCORES_FIELD_NUMBER: _ClassVar[int]
    nodes: _containers.RepeatedCompositeFieldContainer[Node]
    edges: _containers.RepeatedCompositeFieldContainer[Edge]
    health_scores: _containers.RepeatedCompositeFieldContainer[HealthScore]
    def __init__(self, nodes: _Optional[_Iterable[_Union[Node, _Mapping]]] = ..., edges: _Optional[_Iterable[_Union[Edge, _Mapping]]] = ..., health_scores: _Optional[_Iterable[_Union[HealthScore, _Mapping]]] = ...) -> None: ...

class Node(_message.Message):
    __slots__ = ("id", "type", "label", "description", "status", "tags")
    ID_FIELD_NUMBER: _ClassVar[int]
    TYPE_FIELD_NUMBER: _ClassVar[int]
    LABEL_FIELD_NUMBER: _ClassVar[int]
    DESCRIPTION_FIELD_NUMBER: _ClassVar[int]
    STATUS_FIELD_NUMBER: _ClassVar[int]
    TAGS_FIELD_NUMBER: _ClassVar[int]
    id: str
    type: str
    label: str
    description: str
    status: str
    tags: _containers.RepeatedScalarFieldContainer[str]
    def __init__(self, id: _Optional[str] = ..., type: _Optional[str] = ..., label: _Optional[str] = ..., description: _Optional[str] = ..., status: _Optional[str] = ..., tags: _Optional[_Iterable[str]] = ...) -> None: ...

class Edge(_message.Message):
    __slots__ = ("to", "kind", "category", "command", "subcommand", "command_text", "source_file", "line_number")
    FROM_FIELD_NUMBER: _ClassVar[int]
    TO_FIELD_NUMBER: _ClassVar[int]
    KIND_FIELD_NUMBER: _ClassVar[int]
    CATEGORY_FIELD_NUMBER: _ClassVar[int]
    COMMAND_FIELD_NUMBER: _ClassVar[int]
    SUBCOMMAND_FIELD_NUMBER: _ClassVar[int]
    COMMAND_TEXT_FIELD_NUMBER: _ClassVar[int]
    SOURCE_FILE_FIELD_NUMBER: _ClassVar[int]
    LINE_NUMBER_FIELD_NUMBER: _ClassVar[int]
    to: str
    kind: str
    category: str
    command: str
    subcommand: str
    command_text: str
    source_file: str
    line_number: int
    def __init__(self, to: _Optional[str] = ..., kind: _Optional[str] = ..., category: _Optional[str] = ..., command: _Optional[str] = ..., subcommand: _Optional[str] = ..., command_text: _Optional[str] = ..., source_file: _Optional[str] = ..., line_number: _Optional[int] = ..., **kwargs) -> None: ...

class ListNodesResponse(_message.Message):
    __slots__ = ("nodes",)
    NODES_FIELD_NUMBER: _ClassVar[int]
    nodes: _containers.RepeatedCompositeFieldContainer[Node]
    def __init__(self, nodes: _Optional[_Iterable[_Union[Node, _Mapping]]] = ...) -> None: ...

class ListEdgesResponse(_message.Message):
    __slots__ = ("edges",)
    EDGES_FIELD_NUMBER: _ClassVar[int]
    edges: _containers.RepeatedCompositeFieldContainer[Edge]
    def __init__(self, edges: _Optional[_Iterable[_Union[Edge, _Mapping]]] = ...) -> None: ...

class ListCyclesResponse(_message.Message):
    __slots__ = ("cycles",)
    CYCLES_FIELD_NUMBER: _ClassVar[int]
    cycles: _containers.RepeatedCompositeFieldContainer[Cycle]
    def __init__(self, cycles: _Optional[_Iterable[_Union[Cycle, _Mapping]]] = ...) -> None: ...

class Cycle(_message.Message):
    __slots__ = ("node_ids",)
    NODE_IDS_FIELD_NUMBER: _ClassVar[int]
    node_ids: _containers.RepeatedScalarFieldContainer[str]
    def __init__(self, node_ids: _Optional[_Iterable[str]] = ...) -> None: ...

class NodeDetail(_message.Message):
    __slots__ = ("node", "adjacent_edges", "health_score")
    NODE_FIELD_NUMBER: _ClassVar[int]
    ADJACENT_EDGES_FIELD_NUMBER: _ClassVar[int]
    HEALTH_SCORE_FIELD_NUMBER: _ClassVar[int]
    node: Node
    adjacent_edges: _containers.RepeatedCompositeFieldContainer[Edge]
    health_score: HealthScore
    def __init__(self, node: _Optional[_Union[Node, _Mapping]] = ..., adjacent_edges: _Optional[_Iterable[_Union[Edge, _Mapping]]] = ..., health_score: _Optional[_Union[HealthScore, _Mapping]] = ...) -> None: ...

class HealthWeights(_message.Message):
    __slots__ = ("outgoing_edges", "incoming_edges", "code_usage", "recent_activity", "skill_content_length", "agent_context_load", "team_member_count_balance", "team_role_coverage", "action_contract", "action_command", "action_examples", "action_owner")
    OUTGOING_EDGES_FIELD_NUMBER: _ClassVar[int]
    INCOMING_EDGES_FIELD_NUMBER: _ClassVar[int]
    CODE_USAGE_FIELD_NUMBER: _ClassVar[int]
    RECENT_ACTIVITY_FIELD_NUMBER: _ClassVar[int]
    SKILL_CONTENT_LENGTH_FIELD_NUMBER: _ClassVar[int]
    AGENT_CONTEXT_LOAD_FIELD_NUMBER: _ClassVar[int]
    TEAM_MEMBER_COUNT_BALANCE_FIELD_NUMBER: _ClassVar[int]
    TEAM_ROLE_COVERAGE_FIELD_NUMBER: _ClassVar[int]
    ACTION_CONTRACT_FIELD_NUMBER: _ClassVar[int]
    ACTION_COMMAND_FIELD_NUMBER: _ClassVar[int]
    ACTION_EXAMPLES_FIELD_NUMBER: _ClassVar[int]
    ACTION_OWNER_FIELD_NUMBER: _ClassVar[int]
    outgoing_edges: float
    incoming_edges: float
    code_usage: float
    recent_activity: float
    skill_content_length: float
    agent_context_load: float
    team_member_count_balance: float
    team_role_coverage: float
    action_contract: float
    action_command: float
    action_examples: float
    action_owner: float
    def __init__(self, outgoing_edges: _Optional[float] = ..., incoming_edges: _Optional[float] = ..., code_usage: _Optional[float] = ..., recent_activity: _Optional[float] = ..., skill_content_length: _Optional[float] = ..., agent_context_load: _Optional[float] = ..., team_member_count_balance: _Optional[float] = ..., team_role_coverage: _Optional[float] = ..., action_contract: _Optional[float] = ..., action_command: _Optional[float] = ..., action_examples: _Optional[float] = ..., action_owner: _Optional[float] = ...) -> None: ...

class CLIHealthConfig(_message.Message):
    __slots__ = ("neutral_commands", "external_tool_score", "scenario_fallback_score")
    NEUTRAL_COMMANDS_FIELD_NUMBER: _ClassVar[int]
    EXTERNAL_TOOL_SCORE_FIELD_NUMBER: _ClassVar[int]
    SCENARIO_FALLBACK_SCORE_FIELD_NUMBER: _ClassVar[int]
    neutral_commands: _containers.RepeatedScalarFieldContainer[str]
    external_tool_score: float
    scenario_fallback_score: float
    def __init__(self, neutral_commands: _Optional[_Iterable[str]] = ..., external_tool_score: _Optional[float] = ..., scenario_fallback_score: _Optional[float] = ...) -> None: ...

class HealthConfig(_message.Message):
    __slots__ = ("team", "agent", "skill", "action", "cli")
    TEAM_FIELD_NUMBER: _ClassVar[int]
    AGENT_FIELD_NUMBER: _ClassVar[int]
    SKILL_FIELD_NUMBER: _ClassVar[int]
    ACTION_FIELD_NUMBER: _ClassVar[int]
    CLI_FIELD_NUMBER: _ClassVar[int]
    team: HealthWeights
    agent: HealthWeights
    skill: HealthWeights
    action: HealthWeights
    cli: CLIHealthConfig
    def __init__(self, team: _Optional[_Union[HealthWeights, _Mapping]] = ..., agent: _Optional[_Union[HealthWeights, _Mapping]] = ..., skill: _Optional[_Union[HealthWeights, _Mapping]] = ..., action: _Optional[_Union[HealthWeights, _Mapping]] = ..., cli: _Optional[_Union[CLIHealthConfig, _Mapping]] = ...) -> None: ...

class GetHealthScoresRequest(_message.Message):
    __slots__ = ()
    def __init__(self) -> None: ...

class GetHealthScoresResponse(_message.Message):
    __slots__ = ("scores",)
    SCORES_FIELD_NUMBER: _ClassVar[int]
    scores: _containers.RepeatedCompositeFieldContainer[HealthScore]
    def __init__(self, scores: _Optional[_Iterable[_Union[HealthScore, _Mapping]]] = ...) -> None: ...

class HealthScore(_message.Message):
    __slots__ = ("node_id", "score", "factors", "messages")
    class FactorsEntry(_message.Message):
        __slots__ = ("key", "value")
        KEY_FIELD_NUMBER: _ClassVar[int]
        VALUE_FIELD_NUMBER: _ClassVar[int]
        key: str
        value: float
        def __init__(self, key: _Optional[str] = ..., value: _Optional[float] = ...) -> None: ...
    NODE_ID_FIELD_NUMBER: _ClassVar[int]
    SCORE_FIELD_NUMBER: _ClassVar[int]
    FACTORS_FIELD_NUMBER: _ClassVar[int]
    MESSAGES_FIELD_NUMBER: _ClassVar[int]
    node_id: str
    score: float
    factors: _containers.ScalarMap[str, float]
    messages: _containers.RepeatedCompositeFieldContainer[HealthMessage]
    def __init__(self, node_id: _Optional[str] = ..., score: _Optional[float] = ..., factors: _Optional[_Mapping[str, float]] = ..., messages: _Optional[_Iterable[_Union[HealthMessage, _Mapping]]] = ...) -> None: ...

class HealthMessage(_message.Message):
    __slots__ = ("key", "severity", "factor", "summary", "detail", "recommendation", "metric_value", "target")
    KEY_FIELD_NUMBER: _ClassVar[int]
    SEVERITY_FIELD_NUMBER: _ClassVar[int]
    FACTOR_FIELD_NUMBER: _ClassVar[int]
    SUMMARY_FIELD_NUMBER: _ClassVar[int]
    DETAIL_FIELD_NUMBER: _ClassVar[int]
    RECOMMENDATION_FIELD_NUMBER: _ClassVar[int]
    METRIC_VALUE_FIELD_NUMBER: _ClassVar[int]
    TARGET_FIELD_NUMBER: _ClassVar[int]
    key: str
    severity: str
    factor: str
    summary: str
    detail: str
    recommendation: str
    metric_value: float
    target: str
    def __init__(self, key: _Optional[str] = ..., severity: _Optional[str] = ..., factor: _Optional[str] = ..., summary: _Optional[str] = ..., detail: _Optional[str] = ..., recommendation: _Optional[str] = ..., metric_value: _Optional[float] = ..., target: _Optional[str] = ...) -> None: ...
