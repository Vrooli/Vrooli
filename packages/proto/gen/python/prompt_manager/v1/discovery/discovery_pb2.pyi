from google.protobuf.internal import containers as _containers
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Iterable as _Iterable, Mapping as _Mapping
from typing import ClassVar as _ClassVar, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class DiscoverRequest(_message.Message):
    __slots__ = ("queries", "complexity", "limit", "type")
    QUERIES_FIELD_NUMBER: _ClassVar[int]
    COMPLEXITY_FIELD_NUMBER: _ClassVar[int]
    LIMIT_FIELD_NUMBER: _ClassVar[int]
    TYPE_FIELD_NUMBER: _ClassVar[int]
    queries: _containers.RepeatedScalarFieldContainer[str]
    complexity: str
    limit: int
    type: str
    def __init__(self, queries: _Optional[_Iterable[str]] = ..., complexity: _Optional[str] = ..., limit: _Optional[int] = ..., type: _Optional[str] = ...) -> None: ...

class DiscoverResult(_message.Message):
    __slots__ = ("type", "id", "name", "description", "tags", "modes", "score", "score_percent", "source", "topic_depth", "topic_id", "topic_name", "content_chars", "status", "owner", "show_command", "run_command")
    TYPE_FIELD_NUMBER: _ClassVar[int]
    ID_FIELD_NUMBER: _ClassVar[int]
    NAME_FIELD_NUMBER: _ClassVar[int]
    DESCRIPTION_FIELD_NUMBER: _ClassVar[int]
    TAGS_FIELD_NUMBER: _ClassVar[int]
    MODES_FIELD_NUMBER: _ClassVar[int]
    SCORE_FIELD_NUMBER: _ClassVar[int]
    SCORE_PERCENT_FIELD_NUMBER: _ClassVar[int]
    SOURCE_FIELD_NUMBER: _ClassVar[int]
    TOPIC_DEPTH_FIELD_NUMBER: _ClassVar[int]
    TOPIC_ID_FIELD_NUMBER: _ClassVar[int]
    TOPIC_NAME_FIELD_NUMBER: _ClassVar[int]
    CONTENT_CHARS_FIELD_NUMBER: _ClassVar[int]
    STATUS_FIELD_NUMBER: _ClassVar[int]
    OWNER_FIELD_NUMBER: _ClassVar[int]
    SHOW_COMMAND_FIELD_NUMBER: _ClassVar[int]
    RUN_COMMAND_FIELD_NUMBER: _ClassVar[int]
    type: str
    id: str
    name: str
    description: str
    tags: _containers.RepeatedScalarFieldContainer[str]
    modes: _containers.RepeatedScalarFieldContainer[str]
    score: float
    score_percent: int
    source: str
    topic_depth: int
    topic_id: str
    topic_name: str
    content_chars: int
    status: str
    owner: str
    show_command: str
    run_command: str
    def __init__(self, type: _Optional[str] = ..., id: _Optional[str] = ..., name: _Optional[str] = ..., description: _Optional[str] = ..., tags: _Optional[_Iterable[str]] = ..., modes: _Optional[_Iterable[str]] = ..., score: _Optional[float] = ..., score_percent: _Optional[int] = ..., source: _Optional[str] = ..., topic_depth: _Optional[int] = ..., topic_id: _Optional[str] = ..., topic_name: _Optional[str] = ..., content_chars: _Optional[int] = ..., status: _Optional[str] = ..., owner: _Optional[str] = ..., show_command: _Optional[str] = ..., run_command: _Optional[str] = ...) -> None: ...

class DiscoverResponse(_message.Message):
    __slots__ = ("results", "total", "query", "method", "total_content_chars", "read_command", "show_command", "run_command", "budget_chars", "budget_status", "recommended_read_command", "complexity")
    RESULTS_FIELD_NUMBER: _ClassVar[int]
    TOTAL_FIELD_NUMBER: _ClassVar[int]
    QUERY_FIELD_NUMBER: _ClassVar[int]
    METHOD_FIELD_NUMBER: _ClassVar[int]
    TOTAL_CONTENT_CHARS_FIELD_NUMBER: _ClassVar[int]
    READ_COMMAND_FIELD_NUMBER: _ClassVar[int]
    SHOW_COMMAND_FIELD_NUMBER: _ClassVar[int]
    RUN_COMMAND_FIELD_NUMBER: _ClassVar[int]
    BUDGET_CHARS_FIELD_NUMBER: _ClassVar[int]
    BUDGET_STATUS_FIELD_NUMBER: _ClassVar[int]
    RECOMMENDED_READ_COMMAND_FIELD_NUMBER: _ClassVar[int]
    COMPLEXITY_FIELD_NUMBER: _ClassVar[int]
    results: _containers.RepeatedCompositeFieldContainer[DiscoverResult]
    total: int
    query: str
    method: str
    total_content_chars: int
    read_command: str
    show_command: str
    run_command: str
    budget_chars: int
    budget_status: str
    recommended_read_command: str
    complexity: str
    def __init__(self, results: _Optional[_Iterable[_Union[DiscoverResult, _Mapping]]] = ..., total: _Optional[int] = ..., query: _Optional[str] = ..., method: _Optional[str] = ..., total_content_chars: _Optional[int] = ..., read_command: _Optional[str] = ..., show_command: _Optional[str] = ..., run_command: _Optional[str] = ..., budget_chars: _Optional[int] = ..., budget_status: _Optional[str] = ..., recommended_read_command: _Optional[str] = ..., complexity: _Optional[str] = ...) -> None: ...

class ListDiscoveryGapsRequest(_message.Message):
    __slots__ = ("since", "type")
    SINCE_FIELD_NUMBER: _ClassVar[int]
    TYPE_FIELD_NUMBER: _ClassVar[int]
    since: str
    type: str
    def __init__(self, since: _Optional[str] = ..., type: _Optional[str] = ...) -> None: ...

class DiscoveryGap(_message.Message):
    __slots__ = ("query", "count", "last_seen", "types", "examples")
    QUERY_FIELD_NUMBER: _ClassVar[int]
    COUNT_FIELD_NUMBER: _ClassVar[int]
    LAST_SEEN_FIELD_NUMBER: _ClassVar[int]
    TYPES_FIELD_NUMBER: _ClassVar[int]
    EXAMPLES_FIELD_NUMBER: _ClassVar[int]
    query: str
    count: int
    last_seen: str
    types: _containers.RepeatedScalarFieldContainer[str]
    examples: _containers.RepeatedScalarFieldContainer[str]
    def __init__(self, query: _Optional[str] = ..., count: _Optional[int] = ..., last_seen: _Optional[str] = ..., types: _Optional[_Iterable[str]] = ..., examples: _Optional[_Iterable[str]] = ...) -> None: ...

class ListDiscoveryGapsResponse(_message.Message):
    __slots__ = ("clusters", "total", "since")
    CLUSTERS_FIELD_NUMBER: _ClassVar[int]
    TOTAL_FIELD_NUMBER: _ClassVar[int]
    SINCE_FIELD_NUMBER: _ClassVar[int]
    clusters: _containers.RepeatedCompositeFieldContainer[DiscoveryGap]
    total: int
    since: str
    def __init__(self, clusters: _Optional[_Iterable[_Union[DiscoveryGap, _Mapping]]] = ..., total: _Optional[int] = ..., since: _Optional[str] = ...) -> None: ...

class DistributionStats(_message.Message):
    __slots__ = ("count", "min", "p10", "median", "p90", "max", "mean")
    COUNT_FIELD_NUMBER: _ClassVar[int]
    MIN_FIELD_NUMBER: _ClassVar[int]
    P10_FIELD_NUMBER: _ClassVar[int]
    MEDIAN_FIELD_NUMBER: _ClassVar[int]
    P90_FIELD_NUMBER: _ClassVar[int]
    MAX_FIELD_NUMBER: _ClassVar[int]
    MEAN_FIELD_NUMBER: _ClassVar[int]
    count: int
    min: float
    p10: float
    median: float
    p90: float
    max: float
    mean: float
    def __init__(self, count: _Optional[int] = ..., min: _Optional[float] = ..., p10: _Optional[float] = ..., median: _Optional[float] = ..., p90: _Optional[float] = ..., max: _Optional[float] = ..., mean: _Optional[float] = ...) -> None: ...

class ComplexityMetric(_message.Message):
    __slots__ = ("call_count", "over_budget_rate", "median_returned")
    CALL_COUNT_FIELD_NUMBER: _ClassVar[int]
    OVER_BUDGET_RATE_FIELD_NUMBER: _ClassVar[int]
    MEDIAN_RETURNED_FIELD_NUMBER: _ClassVar[int]
    call_count: int
    over_budget_rate: float
    median_returned: float
    def __init__(self, call_count: _Optional[int] = ..., over_budget_rate: _Optional[float] = ..., median_returned: _Optional[float] = ...) -> None: ...

class BudgetHogSkill(_message.Message):
    __slots__ = ("id", "max_chars", "occurrences", "over_budget_sightings")
    ID_FIELD_NUMBER: _ClassVar[int]
    MAX_CHARS_FIELD_NUMBER: _ClassVar[int]
    OCCURRENCES_FIELD_NUMBER: _ClassVar[int]
    OVER_BUDGET_SIGHTINGS_FIELD_NUMBER: _ClassVar[int]
    id: str
    max_chars: int
    occurrences: int
    over_budget_sightings: int
    def __init__(self, id: _Optional[str] = ..., max_chars: _Optional[int] = ..., occurrences: _Optional[int] = ..., over_budget_sightings: _Optional[int] = ...) -> None: ...

class GetDiscoveryMetricsRequest(_message.Message):
    __slots__ = ("since", "type")
    SINCE_FIELD_NUMBER: _ClassVar[int]
    TYPE_FIELD_NUMBER: _ClassVar[int]
    since: str
    type: str
    def __init__(self, since: _Optional[str] = ..., type: _Optional[str] = ...) -> None: ...

class GetDiscoveryMetricsResponse(_message.Message):
    __slots__ = ("since", "call_count", "returned_count", "budgeted_call_count", "over_budget_rate", "near_threshold_rate", "probed_call_count", "threshold_clip_rate", "clipped_per_probe", "per_complexity", "budget_hogs")
    class PerComplexityEntry(_message.Message):
        __slots__ = ("key", "value")
        KEY_FIELD_NUMBER: _ClassVar[int]
        VALUE_FIELD_NUMBER: _ClassVar[int]
        key: str
        value: ComplexityMetric
        def __init__(self, key: _Optional[str] = ..., value: _Optional[_Union[ComplexityMetric, _Mapping]] = ...) -> None: ...
    SINCE_FIELD_NUMBER: _ClassVar[int]
    CALL_COUNT_FIELD_NUMBER: _ClassVar[int]
    RETURNED_COUNT_FIELD_NUMBER: _ClassVar[int]
    BUDGETED_CALL_COUNT_FIELD_NUMBER: _ClassVar[int]
    OVER_BUDGET_RATE_FIELD_NUMBER: _ClassVar[int]
    NEAR_THRESHOLD_RATE_FIELD_NUMBER: _ClassVar[int]
    PROBED_CALL_COUNT_FIELD_NUMBER: _ClassVar[int]
    THRESHOLD_CLIP_RATE_FIELD_NUMBER: _ClassVar[int]
    CLIPPED_PER_PROBE_FIELD_NUMBER: _ClassVar[int]
    PER_COMPLEXITY_FIELD_NUMBER: _ClassVar[int]
    BUDGET_HOGS_FIELD_NUMBER: _ClassVar[int]
    since: str
    call_count: int
    returned_count: DistributionStats
    budgeted_call_count: int
    over_budget_rate: float
    near_threshold_rate: float
    probed_call_count: int
    threshold_clip_rate: float
    clipped_per_probe: DistributionStats
    per_complexity: _containers.MessageMap[str, ComplexityMetric]
    budget_hogs: _containers.RepeatedCompositeFieldContainer[BudgetHogSkill]
    def __init__(self, since: _Optional[str] = ..., call_count: _Optional[int] = ..., returned_count: _Optional[_Union[DistributionStats, _Mapping]] = ..., budgeted_call_count: _Optional[int] = ..., over_budget_rate: _Optional[float] = ..., near_threshold_rate: _Optional[float] = ..., probed_call_count: _Optional[int] = ..., threshold_clip_rate: _Optional[float] = ..., clipped_per_probe: _Optional[_Union[DistributionStats, _Mapping]] = ..., per_complexity: _Optional[_Mapping[str, ComplexityMetric]] = ..., budget_hogs: _Optional[_Iterable[_Union[BudgetHogSkill, _Mapping]]] = ...) -> None: ...

class GetSkillUsageRequest(_message.Message):
    __slots__ = ("since",)
    SINCE_FIELD_NUMBER: _ClassVar[int]
    since: str
    def __init__(self, since: _Optional[str] = ...) -> None: ...

class SkillUsageRow(_message.Message):
    __slots__ = ("skill_id", "returned", "reads", "demand_reads", "via_discovery", "reads_by_caller_kind", "conversion_rate", "last_read_at")
    class ReadsByCallerKindEntry(_message.Message):
        __slots__ = ("key", "value")
        KEY_FIELD_NUMBER: _ClassVar[int]
        VALUE_FIELD_NUMBER: _ClassVar[int]
        key: str
        value: int
        def __init__(self, key: _Optional[str] = ..., value: _Optional[int] = ...) -> None: ...
    SKILL_ID_FIELD_NUMBER: _ClassVar[int]
    RETURNED_FIELD_NUMBER: _ClassVar[int]
    READS_FIELD_NUMBER: _ClassVar[int]
    DEMAND_READS_FIELD_NUMBER: _ClassVar[int]
    VIA_DISCOVERY_FIELD_NUMBER: _ClassVar[int]
    READS_BY_CALLER_KIND_FIELD_NUMBER: _ClassVar[int]
    CONVERSION_RATE_FIELD_NUMBER: _ClassVar[int]
    LAST_READ_AT_FIELD_NUMBER: _ClassVar[int]
    skill_id: str
    returned: int
    reads: int
    demand_reads: int
    via_discovery: int
    reads_by_caller_kind: _containers.ScalarMap[str, int]
    conversion_rate: float
    last_read_at: str
    def __init__(self, skill_id: _Optional[str] = ..., returned: _Optional[int] = ..., reads: _Optional[int] = ..., demand_reads: _Optional[int] = ..., via_discovery: _Optional[int] = ..., reads_by_caller_kind: _Optional[_Mapping[str, int]] = ..., conversion_rate: _Optional[float] = ..., last_read_at: _Optional[str] = ...) -> None: ...

class GetSkillUsageResponse(_message.Message):
    __slots__ = ("since", "unread", "rows")
    SINCE_FIELD_NUMBER: _ClassVar[int]
    UNREAD_FIELD_NUMBER: _ClassVar[int]
    ROWS_FIELD_NUMBER: _ClassVar[int]
    since: str
    unread: _containers.RepeatedScalarFieldContainer[str]
    rows: _containers.RepeatedCompositeFieldContainer[SkillUsageRow]
    def __init__(self, since: _Optional[str] = ..., unread: _Optional[_Iterable[str]] = ..., rows: _Optional[_Iterable[_Union[SkillUsageRow, _Mapping]]] = ...) -> None: ...
