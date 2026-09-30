from portal.v1.shared import common_pb2 as _common_pb2
from google.protobuf.internal import containers as _containers
from google.protobuf.internal import enum_type_wrapper as _enum_type_wrapper
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Iterable as _Iterable, Mapping as _Mapping
from typing import ClassVar as _ClassVar, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class BriefConsumer(int, metaclass=_enum_type_wrapper.EnumTypeWrapper):
    __slots__ = ()
    BRIEF_CONSUMER_UNSPECIFIED: _ClassVar[BriefConsumer]
    BRIEF_CONSUMER_PORTAL_LLM: _ClassVar[BriefConsumer]
    BRIEF_CONSUMER_PORTAL_AGENT: _ClassVar[BriefConsumer]
    BRIEF_CONSUMER_EXTERNAL_HARNESS: _ClassVar[BriefConsumer]

class BriefVerdict(int, metaclass=_enum_type_wrapper.EnumTypeWrapper):
    __slots__ = ()
    BRIEF_VERDICT_UNSPECIFIED: _ClassVar[BriefVerdict]
    BRIEF_VERDICT_DELIVER: _ClassVar[BriefVerdict]
    BRIEF_VERDICT_WITHHELD_LOW_CONFIDENCE: _ClassVar[BriefVerdict]
    BRIEF_VERDICT_WITHHELD_MODE_OFF: _ClassVar[BriefVerdict]
    BRIEF_VERDICT_WITHHELD_BUDGET: _ClassVar[BriefVerdict]
    BRIEF_VERDICT_WITHHELD_DEGRADED: _ClassVar[BriefVerdict]
    BRIEF_VERDICT_WITHHELD_NOT_APPLICABLE: _ClassVar[BriefVerdict]

class TrustClass(int, metaclass=_enum_type_wrapper.EnumTypeWrapper):
    __slots__ = ()
    TRUST_CLASS_UNSPECIFIED: _ClassVar[TrustClass]
    TRUST_CLASS_FIRST_PARTY: _ClassVar[TrustClass]
    TRUST_CLASS_QUOTED: _ClassVar[TrustClass]
    TRUST_CLASS_EXTERNAL: _ClassVar[TrustClass]

class BriefUseKind(int, metaclass=_enum_type_wrapper.EnumTypeWrapper):
    __slots__ = ()
    BRIEF_USE_KIND_UNSPECIFIED: _ClassVar[BriefUseKind]
    BRIEF_USE_KIND_OPENED: _ClassVar[BriefUseKind]
    BRIEF_USE_KIND_COPIED: _ClassVar[BriefUseKind]
    BRIEF_USE_KIND_REFERENCED: _ClassVar[BriefUseKind]
    BRIEF_USE_KIND_REJECTED: _ClassVar[BriefUseKind]
BRIEF_CONSUMER_UNSPECIFIED: BriefConsumer
BRIEF_CONSUMER_PORTAL_LLM: BriefConsumer
BRIEF_CONSUMER_PORTAL_AGENT: BriefConsumer
BRIEF_CONSUMER_EXTERNAL_HARNESS: BriefConsumer
BRIEF_VERDICT_UNSPECIFIED: BriefVerdict
BRIEF_VERDICT_DELIVER: BriefVerdict
BRIEF_VERDICT_WITHHELD_LOW_CONFIDENCE: BriefVerdict
BRIEF_VERDICT_WITHHELD_MODE_OFF: BriefVerdict
BRIEF_VERDICT_WITHHELD_BUDGET: BriefVerdict
BRIEF_VERDICT_WITHHELD_DEGRADED: BriefVerdict
BRIEF_VERDICT_WITHHELD_NOT_APPLICABLE: BriefVerdict
TRUST_CLASS_UNSPECIFIED: TrustClass
TRUST_CLASS_FIRST_PARTY: TrustClass
TRUST_CLASS_QUOTED: TrustClass
TRUST_CLASS_EXTERNAL: TrustClass
BRIEF_USE_KIND_UNSPECIFIED: BriefUseKind
BRIEF_USE_KIND_OPENED: BriefUseKind
BRIEF_USE_KIND_COPIED: BriefUseKind
BRIEF_USE_KIND_REFERENCED: BriefUseKind
BRIEF_USE_KIND_REJECTED: BriefUseKind

class BuildBriefRequest(_message.Message):
    __slots__ = ("prompt", "consumer", "chat_id", "message_id", "harness", "session_ref", "budget_ms")
    PROMPT_FIELD_NUMBER: _ClassVar[int]
    CONSUMER_FIELD_NUMBER: _ClassVar[int]
    CHAT_ID_FIELD_NUMBER: _ClassVar[int]
    MESSAGE_ID_FIELD_NUMBER: _ClassVar[int]
    HARNESS_FIELD_NUMBER: _ClassVar[int]
    SESSION_REF_FIELD_NUMBER: _ClassVar[int]
    BUDGET_MS_FIELD_NUMBER: _ClassVar[int]
    prompt: str
    consumer: BriefConsumer
    chat_id: str
    message_id: str
    harness: _common_pb2.AgentHarness
    session_ref: str
    budget_ms: int
    def __init__(self, prompt: _Optional[str] = ..., consumer: _Optional[_Union[BriefConsumer, str]] = ..., chat_id: _Optional[str] = ..., message_id: _Optional[str] = ..., harness: _Optional[_Union[_common_pb2.AgentHarness, str]] = ..., session_ref: _Optional[str] = ..., budget_ms: _Optional[int] = ...) -> None: ...

class BuildBriefResponse(_message.Message):
    __slots__ = ("brief",)
    BRIEF_FIELD_NUMBER: _ClassVar[int]
    brief: Brief
    def __init__(self, brief: _Optional[_Union[Brief, _Mapping]] = ...) -> None: ...

class Brief(_message.Message):
    __slots__ = ("id", "consumer", "verdict", "reason", "items", "rendered", "latency_ms", "degraded", "max_trust_class", "created_at", "effective_query", "queried_providers")
    ID_FIELD_NUMBER: _ClassVar[int]
    CONSUMER_FIELD_NUMBER: _ClassVar[int]
    VERDICT_FIELD_NUMBER: _ClassVar[int]
    REASON_FIELD_NUMBER: _ClassVar[int]
    ITEMS_FIELD_NUMBER: _ClassVar[int]
    RENDERED_FIELD_NUMBER: _ClassVar[int]
    LATENCY_MS_FIELD_NUMBER: _ClassVar[int]
    DEGRADED_FIELD_NUMBER: _ClassVar[int]
    MAX_TRUST_CLASS_FIELD_NUMBER: _ClassVar[int]
    CREATED_AT_FIELD_NUMBER: _ClassVar[int]
    EFFECTIVE_QUERY_FIELD_NUMBER: _ClassVar[int]
    QUERIED_PROVIDERS_FIELD_NUMBER: _ClassVar[int]
    id: str
    consumer: BriefConsumer
    verdict: BriefVerdict
    reason: str
    items: _containers.RepeatedCompositeFieldContainer[BriefItem]
    rendered: str
    latency_ms: int
    degraded: bool
    max_trust_class: TrustClass
    created_at: str
    effective_query: str
    queried_providers: _containers.RepeatedScalarFieldContainer[str]
    def __init__(self, id: _Optional[str] = ..., consumer: _Optional[_Union[BriefConsumer, str]] = ..., verdict: _Optional[_Union[BriefVerdict, str]] = ..., reason: _Optional[str] = ..., items: _Optional[_Iterable[_Union[BriefItem, _Mapping]]] = ..., rendered: _Optional[str] = ..., latency_ms: _Optional[int] = ..., degraded: _Optional[bool] = ..., max_trust_class: _Optional[_Union[TrustClass, str]] = ..., created_at: _Optional[str] = ..., effective_query: _Optional[str] = ..., queried_providers: _Optional[_Iterable[str]] = ...) -> None: ...

class BriefItem(_message.Message):
    __slots__ = ("provider_id", "type", "title", "snippet", "path", "score", "rerank_score", "trust_class", "suggested_command")
    PROVIDER_ID_FIELD_NUMBER: _ClassVar[int]
    TYPE_FIELD_NUMBER: _ClassVar[int]
    TITLE_FIELD_NUMBER: _ClassVar[int]
    SNIPPET_FIELD_NUMBER: _ClassVar[int]
    PATH_FIELD_NUMBER: _ClassVar[int]
    SCORE_FIELD_NUMBER: _ClassVar[int]
    RERANK_SCORE_FIELD_NUMBER: _ClassVar[int]
    TRUST_CLASS_FIELD_NUMBER: _ClassVar[int]
    SUGGESTED_COMMAND_FIELD_NUMBER: _ClassVar[int]
    provider_id: str
    type: str
    title: str
    snippet: str
    path: str
    score: float
    rerank_score: float
    trust_class: TrustClass
    suggested_command: str
    def __init__(self, provider_id: _Optional[str] = ..., type: _Optional[str] = ..., title: _Optional[str] = ..., snippet: _Optional[str] = ..., path: _Optional[str] = ..., score: _Optional[float] = ..., rerank_score: _Optional[float] = ..., trust_class: _Optional[_Union[TrustClass, str]] = ..., suggested_command: _Optional[str] = ...) -> None: ...

class GetBriefRequest(_message.Message):
    __slots__ = ("id",)
    ID_FIELD_NUMBER: _ClassVar[int]
    id: str
    def __init__(self, id: _Optional[str] = ...) -> None: ...

class GetBriefResponse(_message.Message):
    __slots__ = ("brief",)
    BRIEF_FIELD_NUMBER: _ClassVar[int]
    brief: Brief
    def __init__(self, brief: _Optional[_Union[Brief, _Mapping]] = ...) -> None: ...

class ListBriefsRequest(_message.Message):
    __slots__ = ("consumer", "chat_id", "session_ref", "limit")
    CONSUMER_FIELD_NUMBER: _ClassVar[int]
    CHAT_ID_FIELD_NUMBER: _ClassVar[int]
    SESSION_REF_FIELD_NUMBER: _ClassVar[int]
    LIMIT_FIELD_NUMBER: _ClassVar[int]
    consumer: BriefConsumer
    chat_id: str
    session_ref: str
    limit: int
    def __init__(self, consumer: _Optional[_Union[BriefConsumer, str]] = ..., chat_id: _Optional[str] = ..., session_ref: _Optional[str] = ..., limit: _Optional[int] = ...) -> None: ...

class ListBriefsResponse(_message.Message):
    __slots__ = ("briefs",)
    BRIEFS_FIELD_NUMBER: _ClassVar[int]
    briefs: _containers.RepeatedCompositeFieldContainer[Brief]
    def __init__(self, briefs: _Optional[_Iterable[_Union[Brief, _Mapping]]] = ...) -> None: ...

class RecordBriefUseRequest(_message.Message):
    __slots__ = ("brief_id", "item_index", "kind")
    BRIEF_ID_FIELD_NUMBER: _ClassVar[int]
    ITEM_INDEX_FIELD_NUMBER: _ClassVar[int]
    KIND_FIELD_NUMBER: _ClassVar[int]
    brief_id: str
    item_index: int
    kind: BriefUseKind
    def __init__(self, brief_id: _Optional[str] = ..., item_index: _Optional[int] = ..., kind: _Optional[_Union[BriefUseKind, str]] = ...) -> None: ...

class RecordBriefUseResponse(_message.Message):
    __slots__ = ("recorded",)
    RECORDED_FIELD_NUMBER: _ClassVar[int]
    recorded: bool
    def __init__(self, recorded: _Optional[bool] = ...) -> None: ...

class BriefStatsRequest(_message.Message):
    __slots__ = ("window_days", "consumer")
    WINDOW_DAYS_FIELD_NUMBER: _ClassVar[int]
    CONSUMER_FIELD_NUMBER: _ClassVar[int]
    window_days: int
    consumer: BriefConsumer
    def __init__(self, window_days: _Optional[int] = ..., consumer: _Optional[_Union[BriefConsumer, str]] = ...) -> None: ...

class BriefStatsRow(_message.Message):
    __slots__ = ("consumer", "briefs_built", "briefs_delivered", "items_delivered", "items_used", "usage_rate", "withheld_rate", "withheld_by_verdict")
    class WithheldByVerdictEntry(_message.Message):
        __slots__ = ("key", "value")
        KEY_FIELD_NUMBER: _ClassVar[int]
        VALUE_FIELD_NUMBER: _ClassVar[int]
        key: str
        value: int
        def __init__(self, key: _Optional[str] = ..., value: _Optional[int] = ...) -> None: ...
    CONSUMER_FIELD_NUMBER: _ClassVar[int]
    BRIEFS_BUILT_FIELD_NUMBER: _ClassVar[int]
    BRIEFS_DELIVERED_FIELD_NUMBER: _ClassVar[int]
    ITEMS_DELIVERED_FIELD_NUMBER: _ClassVar[int]
    ITEMS_USED_FIELD_NUMBER: _ClassVar[int]
    USAGE_RATE_FIELD_NUMBER: _ClassVar[int]
    WITHHELD_RATE_FIELD_NUMBER: _ClassVar[int]
    WITHHELD_BY_VERDICT_FIELD_NUMBER: _ClassVar[int]
    consumer: BriefConsumer
    briefs_built: int
    briefs_delivered: int
    items_delivered: int
    items_used: int
    usage_rate: float
    withheld_rate: float
    withheld_by_verdict: _containers.ScalarMap[str, int]
    def __init__(self, consumer: _Optional[_Union[BriefConsumer, str]] = ..., briefs_built: _Optional[int] = ..., briefs_delivered: _Optional[int] = ..., items_delivered: _Optional[int] = ..., items_used: _Optional[int] = ..., usage_rate: _Optional[float] = ..., withheld_rate: _Optional[float] = ..., withheld_by_verdict: _Optional[_Mapping[str, int]] = ...) -> None: ...

class BriefStatsResponse(_message.Message):
    __slots__ = ("rows",)
    ROWS_FIELD_NUMBER: _ClassVar[int]
    rows: _containers.RepeatedCompositeFieldContainer[BriefStatsRow]
    def __init__(self, rows: _Optional[_Iterable[_Union[BriefStatsRow, _Mapping]]] = ...) -> None: ...
