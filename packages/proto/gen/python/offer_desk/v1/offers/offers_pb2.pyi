import datetime

from google.protobuf import timestamp_pb2 as _timestamp_pb2
from buf.validate import validate_pb2 as _validate_pb2
from money_ledger.v1.ledger import ledger_pb2 as _ledger_pb2
from google.protobuf.internal import containers as _containers
from google.protobuf.internal import enum_type_wrapper as _enum_type_wrapper
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Iterable as _Iterable, Mapping as _Mapping
from typing import ClassVar as _ClassVar, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class NodeKind(int, metaclass=_enum_type_wrapper.EnumTypeWrapper):
    __slots__ = ()
    NODE_KIND_UNSPECIFIED: _ClassVar[NodeKind]
    OFFER: _ClassVar[NodeKind]
    VARIANT: _ClassVar[NodeKind]
    CHANNEL: _ClassVar[NodeKind]
    REVENUE_LINE: _ClassVar[NodeKind]
    DELIVERABLE: _ClassVar[NodeKind]

class Status(int, metaclass=_enum_type_wrapper.EnumTypeWrapper):
    __slots__ = ()
    STATUS_UNSPECIFIED: _ClassVar[Status]
    IDEA: _ClassVar[Status]
    CANDIDATE: _ClassVar[Status]
    TRIGGER_MET: _ClassVar[Status]
    ACTIVE: _ClassVar[Status]
    SHIPPED: _ClassVar[Status]
    RETIRED: _ClassVar[Status]

class Verdict(int, metaclass=_enum_type_wrapper.EnumTypeWrapper):
    __slots__ = ()
    VERDICT_UNSPECIFIED: _ClassVar[Verdict]
    SATISFIED: _ClassVar[Verdict]
    UNSATISFIED: _ClassVar[Verdict]
    UNKNOWN: _ClassVar[Verdict]

class TriggerComposition(int, metaclass=_enum_type_wrapper.EnumTypeWrapper):
    __slots__ = ()
    TRIGGER_COMPOSITION_UNSPECIFIED: _ClassVar[TriggerComposition]
    ALL: _ClassVar[TriggerComposition]
    ANY: _ClassVar[TriggerComposition]
NODE_KIND_UNSPECIFIED: NodeKind
OFFER: NodeKind
VARIANT: NodeKind
CHANNEL: NodeKind
REVENUE_LINE: NodeKind
DELIVERABLE: NodeKind
STATUS_UNSPECIFIED: Status
IDEA: Status
CANDIDATE: Status
TRIGGER_MET: Status
ACTIVE: Status
SHIPPED: Status
RETIRED: Status
VERDICT_UNSPECIFIED: Verdict
SATISFIED: Verdict
UNSATISFIED: Verdict
UNKNOWN: Verdict
TRIGGER_COMPOSITION_UNSPECIFIED: TriggerComposition
ALL: TriggerComposition
ANY: TriggerComposition

class Node(_message.Message):
    __slots__ = ("id", "kind", "name", "status", "trigger_id", "created_at", "actual_account_id")
    ID_FIELD_NUMBER: _ClassVar[int]
    KIND_FIELD_NUMBER: _ClassVar[int]
    NAME_FIELD_NUMBER: _ClassVar[int]
    STATUS_FIELD_NUMBER: _ClassVar[int]
    TRIGGER_ID_FIELD_NUMBER: _ClassVar[int]
    CREATED_AT_FIELD_NUMBER: _ClassVar[int]
    ACTUAL_ACCOUNT_ID_FIELD_NUMBER: _ClassVar[int]
    id: str
    kind: NodeKind
    name: str
    status: Status
    trigger_id: str
    created_at: _timestamp_pb2.Timestamp
    actual_account_id: str
    def __init__(self, id: _Optional[str] = ..., kind: _Optional[_Union[NodeKind, str]] = ..., name: _Optional[str] = ..., status: _Optional[_Union[Status, str]] = ..., trigger_id: _Optional[str] = ..., created_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., actual_account_id: _Optional[str] = ...) -> None: ...

class Edge(_message.Message):
    __slots__ = ("id", "from_id", "to_id", "kind", "intended_price_minor", "currency")
    ID_FIELD_NUMBER: _ClassVar[int]
    FROM_ID_FIELD_NUMBER: _ClassVar[int]
    TO_ID_FIELD_NUMBER: _ClassVar[int]
    KIND_FIELD_NUMBER: _ClassVar[int]
    INTENDED_PRICE_MINOR_FIELD_NUMBER: _ClassVar[int]
    CURRENCY_FIELD_NUMBER: _ClassVar[int]
    id: str
    from_id: str
    to_id: str
    kind: str
    intended_price_minor: int
    currency: str
    def __init__(self, id: _Optional[str] = ..., from_id: _Optional[str] = ..., to_id: _Optional[str] = ..., kind: _Optional[str] = ..., intended_price_minor: _Optional[int] = ..., currency: _Optional[str] = ...) -> None: ...

class TriggerClause(_message.Message):
    __slots__ = ("fact_name", "operator", "threshold")
    FACT_NAME_FIELD_NUMBER: _ClassVar[int]
    OPERATOR_FIELD_NUMBER: _ClassVar[int]
    THRESHOLD_FIELD_NUMBER: _ClassVar[int]
    fact_name: str
    operator: str
    threshold: float
    def __init__(self, fact_name: _Optional[str] = ..., operator: _Optional[str] = ..., threshold: _Optional[float] = ...) -> None: ...

class Fact(_message.Message):
    __slots__ = ("name", "value", "observed_at", "stale_after_days", "dimension")
    NAME_FIELD_NUMBER: _ClassVar[int]
    VALUE_FIELD_NUMBER: _ClassVar[int]
    OBSERVED_AT_FIELD_NUMBER: _ClassVar[int]
    STALE_AFTER_DAYS_FIELD_NUMBER: _ClassVar[int]
    DIMENSION_FIELD_NUMBER: _ClassVar[int]
    name: str
    value: float
    observed_at: _timestamp_pb2.Timestamp
    stale_after_days: int
    dimension: str
    def __init__(self, name: _Optional[str] = ..., value: _Optional[float] = ..., observed_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., stale_after_days: _Optional[int] = ..., dimension: _Optional[str] = ...) -> None: ...

class Trigger(_message.Message):
    __slots__ = ("id", "node_id", "fact_name", "operator", "threshold", "expression", "clauses", "composition")
    ID_FIELD_NUMBER: _ClassVar[int]
    NODE_ID_FIELD_NUMBER: _ClassVar[int]
    FACT_NAME_FIELD_NUMBER: _ClassVar[int]
    OPERATOR_FIELD_NUMBER: _ClassVar[int]
    THRESHOLD_FIELD_NUMBER: _ClassVar[int]
    EXPRESSION_FIELD_NUMBER: _ClassVar[int]
    CLAUSES_FIELD_NUMBER: _ClassVar[int]
    COMPOSITION_FIELD_NUMBER: _ClassVar[int]
    id: str
    node_id: str
    fact_name: str
    operator: str
    threshold: float
    expression: str
    clauses: _containers.RepeatedCompositeFieldContainer[TriggerClause]
    composition: TriggerComposition
    def __init__(self, id: _Optional[str] = ..., node_id: _Optional[str] = ..., fact_name: _Optional[str] = ..., operator: _Optional[str] = ..., threshold: _Optional[float] = ..., expression: _Optional[str] = ..., clauses: _Optional[_Iterable[_Union[TriggerClause, _Mapping]]] = ..., composition: _Optional[_Union[TriggerComposition, str]] = ...) -> None: ...

class Evaluation(_message.Message):
    __slots__ = ("id", "node_id", "verdict", "fact_name", "explanation", "evaluated_at", "fact_age_seconds", "fact_names")
    ID_FIELD_NUMBER: _ClassVar[int]
    NODE_ID_FIELD_NUMBER: _ClassVar[int]
    VERDICT_FIELD_NUMBER: _ClassVar[int]
    FACT_NAME_FIELD_NUMBER: _ClassVar[int]
    EXPLANATION_FIELD_NUMBER: _ClassVar[int]
    EVALUATED_AT_FIELD_NUMBER: _ClassVar[int]
    FACT_AGE_SECONDS_FIELD_NUMBER: _ClassVar[int]
    FACT_NAMES_FIELD_NUMBER: _ClassVar[int]
    id: str
    node_id: str
    verdict: Verdict
    fact_name: str
    explanation: str
    evaluated_at: _timestamp_pb2.Timestamp
    fact_age_seconds: int
    fact_names: _containers.RepeatedScalarFieldContainer[str]
    def __init__(self, id: _Optional[str] = ..., node_id: _Optional[str] = ..., verdict: _Optional[_Union[Verdict, str]] = ..., fact_name: _Optional[str] = ..., explanation: _Optional[str] = ..., evaluated_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., fact_age_seconds: _Optional[int] = ..., fact_names: _Optional[_Iterable[str]] = ...) -> None: ...

class Proposal(_message.Message):
    __slots__ = ("id", "node_id", "actor", "requested_status", "reason")
    ID_FIELD_NUMBER: _ClassVar[int]
    NODE_ID_FIELD_NUMBER: _ClassVar[int]
    ACTOR_FIELD_NUMBER: _ClassVar[int]
    REQUESTED_STATUS_FIELD_NUMBER: _ClassVar[int]
    REASON_FIELD_NUMBER: _ClassVar[int]
    id: str
    node_id: str
    actor: str
    requested_status: Status
    reason: str
    def __init__(self, id: _Optional[str] = ..., node_id: _Optional[str] = ..., actor: _Optional[str] = ..., requested_status: _Optional[_Union[Status, str]] = ..., reason: _Optional[str] = ...) -> None: ...

class Availability(_message.Message):
    __slots__ = ("source", "reason", "last_success_at")
    SOURCE_FIELD_NUMBER: _ClassVar[int]
    REASON_FIELD_NUMBER: _ClassVar[int]
    LAST_SUCCESS_AT_FIELD_NUMBER: _ClassVar[int]
    source: str
    reason: str
    last_success_at: _timestamp_pb2.Timestamp
    def __init__(self, source: _Optional[str] = ..., reason: _Optional[str] = ..., last_success_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ...) -> None: ...

class BoardEntry(_message.Message):
    __slots__ = ("node_id", "title", "rank_reason", "status", "actual_minor", "actuals_available", "availability")
    NODE_ID_FIELD_NUMBER: _ClassVar[int]
    TITLE_FIELD_NUMBER: _ClassVar[int]
    RANK_REASON_FIELD_NUMBER: _ClassVar[int]
    STATUS_FIELD_NUMBER: _ClassVar[int]
    ACTUAL_MINOR_FIELD_NUMBER: _ClassVar[int]
    ACTUALS_AVAILABLE_FIELD_NUMBER: _ClassVar[int]
    AVAILABILITY_FIELD_NUMBER: _ClassVar[int]
    node_id: str
    title: str
    rank_reason: str
    status: Status
    actual_minor: int
    actuals_available: bool
    availability: _containers.RepeatedCompositeFieldContainer[Availability]
    def __init__(self, node_id: _Optional[str] = ..., title: _Optional[str] = ..., rank_reason: _Optional[str] = ..., status: _Optional[_Union[Status, str]] = ..., actual_minor: _Optional[int] = ..., actuals_available: _Optional[bool] = ..., availability: _Optional[_Iterable[_Union[Availability, _Mapping]]] = ...) -> None: ...

class BoardResponse(_message.Message):
    __slots__ = ("entries", "position", "availability")
    ENTRIES_FIELD_NUMBER: _ClassVar[int]
    POSITION_FIELD_NUMBER: _ClassVar[int]
    AVAILABILITY_FIELD_NUMBER: _ClassVar[int]
    entries: _containers.RepeatedCompositeFieldContainer[BoardEntry]
    position: _ledger_pb2.PositionResponse
    availability: _containers.RepeatedCompositeFieldContainer[Availability]
    def __init__(self, entries: _Optional[_Iterable[_Union[BoardEntry, _Mapping]]] = ..., position: _Optional[_Union[_ledger_pb2.PositionResponse, _Mapping]] = ..., availability: _Optional[_Iterable[_Union[Availability, _Mapping]]] = ...) -> None: ...

class CreateNodeRequest(_message.Message):
    __slots__ = ("kind", "name", "status", "trigger_id", "actual_account_id")
    KIND_FIELD_NUMBER: _ClassVar[int]
    NAME_FIELD_NUMBER: _ClassVar[int]
    STATUS_FIELD_NUMBER: _ClassVar[int]
    TRIGGER_ID_FIELD_NUMBER: _ClassVar[int]
    ACTUAL_ACCOUNT_ID_FIELD_NUMBER: _ClassVar[int]
    kind: NodeKind
    name: str
    status: Status
    trigger_id: str
    actual_account_id: str
    def __init__(self, kind: _Optional[_Union[NodeKind, str]] = ..., name: _Optional[str] = ..., status: _Optional[_Union[Status, str]] = ..., trigger_id: _Optional[str] = ..., actual_account_id: _Optional[str] = ...) -> None: ...

class CreateNodeResponse(_message.Message):
    __slots__ = ("node",)
    NODE_FIELD_NUMBER: _ClassVar[int]
    node: Node
    def __init__(self, node: _Optional[_Union[Node, _Mapping]] = ...) -> None: ...

class ListNodesRequest(_message.Message):
    __slots__ = ("kind", "status")
    KIND_FIELD_NUMBER: _ClassVar[int]
    STATUS_FIELD_NUMBER: _ClassVar[int]
    kind: NodeKind
    status: Status
    def __init__(self, kind: _Optional[_Union[NodeKind, str]] = ..., status: _Optional[_Union[Status, str]] = ...) -> None: ...

class ListNodesResponse(_message.Message):
    __slots__ = ("nodes",)
    NODES_FIELD_NUMBER: _ClassVar[int]
    nodes: _containers.RepeatedCompositeFieldContainer[Node]
    def __init__(self, nodes: _Optional[_Iterable[_Union[Node, _Mapping]]] = ...) -> None: ...

class TransitionRequest(_message.Message):
    __slots__ = ("node_id", "status", "actor")
    NODE_ID_FIELD_NUMBER: _ClassVar[int]
    STATUS_FIELD_NUMBER: _ClassVar[int]
    ACTOR_FIELD_NUMBER: _ClassVar[int]
    node_id: str
    status: Status
    actor: str
    def __init__(self, node_id: _Optional[str] = ..., status: _Optional[_Union[Status, str]] = ..., actor: _Optional[str] = ...) -> None: ...

class TransitionResponse(_message.Message):
    __slots__ = ("node",)
    NODE_FIELD_NUMBER: _ClassVar[int]
    node: Node
    def __init__(self, node: _Optional[_Union[Node, _Mapping]] = ...) -> None: ...

class CreateEdgeRequest(_message.Message):
    __slots__ = ("edge",)
    EDGE_FIELD_NUMBER: _ClassVar[int]
    edge: Edge
    def __init__(self, edge: _Optional[_Union[Edge, _Mapping]] = ...) -> None: ...

class CreateEdgeResponse(_message.Message):
    __slots__ = ("edge",)
    EDGE_FIELD_NUMBER: _ClassVar[int]
    edge: Edge
    def __init__(self, edge: _Optional[_Union[Edge, _Mapping]] = ...) -> None: ...

class DeclareTriggerRequest(_message.Message):
    __slots__ = ("trigger",)
    TRIGGER_FIELD_NUMBER: _ClassVar[int]
    trigger: Trigger
    def __init__(self, trigger: _Optional[_Union[Trigger, _Mapping]] = ...) -> None: ...

class DeclareTriggerResponse(_message.Message):
    __slots__ = ("trigger",)
    TRIGGER_FIELD_NUMBER: _ClassVar[int]
    trigger: Trigger
    def __init__(self, trigger: _Optional[_Union[Trigger, _Mapping]] = ...) -> None: ...

class AddFactRequest(_message.Message):
    __slots__ = ("fact",)
    FACT_FIELD_NUMBER: _ClassVar[int]
    fact: Fact
    def __init__(self, fact: _Optional[_Union[Fact, _Mapping]] = ...) -> None: ...

class AddFactResponse(_message.Message):
    __slots__ = ("fact",)
    FACT_FIELD_NUMBER: _ClassVar[int]
    fact: Fact
    def __init__(self, fact: _Optional[_Union[Fact, _Mapping]] = ...) -> None: ...

class EvaluateRequest(_message.Message):
    __slots__ = ("dry_run",)
    DRY_RUN_FIELD_NUMBER: _ClassVar[int]
    dry_run: bool
    def __init__(self, dry_run: _Optional[bool] = ...) -> None: ...

class EvaluateResponse(_message.Message):
    __slots__ = ("evaluations",)
    EVALUATIONS_FIELD_NUMBER: _ClassVar[int]
    evaluations: _containers.RepeatedCompositeFieldContainer[Evaluation]
    def __init__(self, evaluations: _Optional[_Iterable[_Union[Evaluation, _Mapping]]] = ...) -> None: ...

class PromoteRequest(_message.Message):
    __slots__ = ("node_id", "actor", "role")
    NODE_ID_FIELD_NUMBER: _ClassVar[int]
    ACTOR_FIELD_NUMBER: _ClassVar[int]
    ROLE_FIELD_NUMBER: _ClassVar[int]
    node_id: str
    actor: str
    role: str
    def __init__(self, node_id: _Optional[str] = ..., actor: _Optional[str] = ..., role: _Optional[str] = ...) -> None: ...

class PromoteResponse(_message.Message):
    __slots__ = ("proposal",)
    PROPOSAL_FIELD_NUMBER: _ClassVar[int]
    proposal: Proposal
    def __init__(self, proposal: _Optional[_Union[Proposal, _Mapping]] = ...) -> None: ...

class ListAuditRequest(_message.Message):
    __slots__ = ("node_id",)
    NODE_ID_FIELD_NUMBER: _ClassVar[int]
    node_id: str
    def __init__(self, node_id: _Optional[str] = ...) -> None: ...

class ListAuditResponse(_message.Message):
    __slots__ = ("entries",)
    ENTRIES_FIELD_NUMBER: _ClassVar[int]
    entries: _containers.RepeatedScalarFieldContainer[str]
    def __init__(self, entries: _Optional[_Iterable[str]] = ...) -> None: ...

class ListEdgesRequest(_message.Message):
    __slots__ = ("node_id",)
    NODE_ID_FIELD_NUMBER: _ClassVar[int]
    node_id: str
    def __init__(self, node_id: _Optional[str] = ...) -> None: ...

class ListEdgesResponse(_message.Message):
    __slots__ = ("edges",)
    EDGES_FIELD_NUMBER: _ClassVar[int]
    edges: _containers.RepeatedCompositeFieldContainer[Edge]
    def __init__(self, edges: _Optional[_Iterable[_Union[Edge, _Mapping]]] = ...) -> None: ...

class ProjectionRequest(_message.Message):
    __slots__ = ()
    def __init__(self) -> None: ...
