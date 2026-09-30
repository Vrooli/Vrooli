from google.protobuf.internal import containers as _containers
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Iterable as _Iterable, Mapping as _Mapping
from typing import ClassVar as _ClassVar, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class InventoryEvent(_message.Message):
    __slots__ = ("id", "workspace_id", "kind", "item_id", "batch_id", "amount", "unit", "recipe_id", "created_at")
    ID_FIELD_NUMBER: _ClassVar[int]
    WORKSPACE_ID_FIELD_NUMBER: _ClassVar[int]
    KIND_FIELD_NUMBER: _ClassVar[int]
    ITEM_ID_FIELD_NUMBER: _ClassVar[int]
    BATCH_ID_FIELD_NUMBER: _ClassVar[int]
    AMOUNT_FIELD_NUMBER: _ClassVar[int]
    UNIT_FIELD_NUMBER: _ClassVar[int]
    RECIPE_ID_FIELD_NUMBER: _ClassVar[int]
    CREATED_AT_FIELD_NUMBER: _ClassVar[int]
    id: str
    workspace_id: str
    kind: str
    item_id: str
    batch_id: str
    amount: str
    unit: str
    recipe_id: str
    created_at: str
    def __init__(self, id: _Optional[str] = ..., workspace_id: _Optional[str] = ..., kind: _Optional[str] = ..., item_id: _Optional[str] = ..., batch_id: _Optional[str] = ..., amount: _Optional[str] = ..., unit: _Optional[str] = ..., recipe_id: _Optional[str] = ..., created_at: _Optional[str] = ...) -> None: ...

class ListEventsRequest(_message.Message):
    __slots__ = ("workspace_id",)
    WORKSPACE_ID_FIELD_NUMBER: _ClassVar[int]
    workspace_id: str
    def __init__(self, workspace_id: _Optional[str] = ...) -> None: ...

class ListEventsResponse(_message.Message):
    __slots__ = ("events",)
    EVENTS_FIELD_NUMBER: _ClassVar[int]
    events: _containers.RepeatedCompositeFieldContainer[InventoryEvent]
    def __init__(self, events: _Optional[_Iterable[_Union[InventoryEvent, _Mapping]]] = ...) -> None: ...

class RecordEventRequest(_message.Message):
    __slots__ = ("workspace_id", "id", "kind", "item_id", "batch_id", "amount", "unit", "recipe_id", "created_at")
    WORKSPACE_ID_FIELD_NUMBER: _ClassVar[int]
    ID_FIELD_NUMBER: _ClassVar[int]
    KIND_FIELD_NUMBER: _ClassVar[int]
    ITEM_ID_FIELD_NUMBER: _ClassVar[int]
    BATCH_ID_FIELD_NUMBER: _ClassVar[int]
    AMOUNT_FIELD_NUMBER: _ClassVar[int]
    UNIT_FIELD_NUMBER: _ClassVar[int]
    RECIPE_ID_FIELD_NUMBER: _ClassVar[int]
    CREATED_AT_FIELD_NUMBER: _ClassVar[int]
    workspace_id: str
    id: str
    kind: str
    item_id: str
    batch_id: str
    amount: str
    unit: str
    recipe_id: str
    created_at: str
    def __init__(self, workspace_id: _Optional[str] = ..., id: _Optional[str] = ..., kind: _Optional[str] = ..., item_id: _Optional[str] = ..., batch_id: _Optional[str] = ..., amount: _Optional[str] = ..., unit: _Optional[str] = ..., recipe_id: _Optional[str] = ..., created_at: _Optional[str] = ...) -> None: ...

class RecordEventResponse(_message.Message):
    __slots__ = ("event",)
    EVENT_FIELD_NUMBER: _ClassVar[int]
    event: InventoryEvent
    def __init__(self, event: _Optional[_Union[InventoryEvent, _Mapping]] = ...) -> None: ...

class Batch(_message.Message):
    __slots__ = ("id", "recipe_id", "recipe_revision", "yield_amount", "available_amount", "unit")
    ID_FIELD_NUMBER: _ClassVar[int]
    RECIPE_ID_FIELD_NUMBER: _ClassVar[int]
    RECIPE_REVISION_FIELD_NUMBER: _ClassVar[int]
    YIELD_AMOUNT_FIELD_NUMBER: _ClassVar[int]
    AVAILABLE_AMOUNT_FIELD_NUMBER: _ClassVar[int]
    UNIT_FIELD_NUMBER: _ClassVar[int]
    id: str
    recipe_id: str
    recipe_revision: int
    yield_amount: str
    available_amount: str
    unit: str
    def __init__(self, id: _Optional[str] = ..., recipe_id: _Optional[str] = ..., recipe_revision: _Optional[int] = ..., yield_amount: _Optional[str] = ..., available_amount: _Optional[str] = ..., unit: _Optional[str] = ...) -> None: ...

class PrepareRequirement(_message.Message):
    __slots__ = ("item_id", "amount", "unit")
    ITEM_ID_FIELD_NUMBER: _ClassVar[int]
    AMOUNT_FIELD_NUMBER: _ClassVar[int]
    UNIT_FIELD_NUMBER: _ClassVar[int]
    item_id: str
    amount: str
    unit: str
    def __init__(self, item_id: _Optional[str] = ..., amount: _Optional[str] = ..., unit: _Optional[str] = ...) -> None: ...

class PrepareBatchRequest(_message.Message):
    __slots__ = ("workspace_id", "event_id", "batch_id", "recipe_id", "recipe_revision", "yield_amount", "unit", "requirements")
    WORKSPACE_ID_FIELD_NUMBER: _ClassVar[int]
    EVENT_ID_FIELD_NUMBER: _ClassVar[int]
    BATCH_ID_FIELD_NUMBER: _ClassVar[int]
    RECIPE_ID_FIELD_NUMBER: _ClassVar[int]
    RECIPE_REVISION_FIELD_NUMBER: _ClassVar[int]
    YIELD_AMOUNT_FIELD_NUMBER: _ClassVar[int]
    UNIT_FIELD_NUMBER: _ClassVar[int]
    REQUIREMENTS_FIELD_NUMBER: _ClassVar[int]
    workspace_id: str
    event_id: str
    batch_id: str
    recipe_id: str
    recipe_revision: int
    yield_amount: str
    unit: str
    requirements: _containers.RepeatedCompositeFieldContainer[PrepareRequirement]
    def __init__(self, workspace_id: _Optional[str] = ..., event_id: _Optional[str] = ..., batch_id: _Optional[str] = ..., recipe_id: _Optional[str] = ..., recipe_revision: _Optional[int] = ..., yield_amount: _Optional[str] = ..., unit: _Optional[str] = ..., requirements: _Optional[_Iterable[_Union[PrepareRequirement, _Mapping]]] = ...) -> None: ...

class PrepareBatchResponse(_message.Message):
    __slots__ = ("batch",)
    BATCH_FIELD_NUMBER: _ClassVar[int]
    batch: Batch
    def __init__(self, batch: _Optional[_Union[Batch, _Mapping]] = ...) -> None: ...

class ConsumeBatchPortionRequest(_message.Message):
    __slots__ = ("workspace_id", "event_id", "batch_id", "amount", "unit", "recipe_id")
    WORKSPACE_ID_FIELD_NUMBER: _ClassVar[int]
    EVENT_ID_FIELD_NUMBER: _ClassVar[int]
    BATCH_ID_FIELD_NUMBER: _ClassVar[int]
    AMOUNT_FIELD_NUMBER: _ClassVar[int]
    UNIT_FIELD_NUMBER: _ClassVar[int]
    RECIPE_ID_FIELD_NUMBER: _ClassVar[int]
    workspace_id: str
    event_id: str
    batch_id: str
    amount: str
    unit: str
    recipe_id: str
    def __init__(self, workspace_id: _Optional[str] = ..., event_id: _Optional[str] = ..., batch_id: _Optional[str] = ..., amount: _Optional[str] = ..., unit: _Optional[str] = ..., recipe_id: _Optional[str] = ...) -> None: ...

class BatchResponse(_message.Message):
    __slots__ = ("batch",)
    BATCH_FIELD_NUMBER: _ClassVar[int]
    batch: Batch
    def __init__(self, batch: _Optional[_Union[Batch, _Mapping]] = ...) -> None: ...

class ReceiptProposal(_message.Message):
    __slots__ = ("id", "workspace_id", "source_id", "transaction_id", "line_key", "description", "item_id", "amount", "unit", "price", "status", "event_id", "created_at", "applied_at")
    ID_FIELD_NUMBER: _ClassVar[int]
    WORKSPACE_ID_FIELD_NUMBER: _ClassVar[int]
    SOURCE_ID_FIELD_NUMBER: _ClassVar[int]
    TRANSACTION_ID_FIELD_NUMBER: _ClassVar[int]
    LINE_KEY_FIELD_NUMBER: _ClassVar[int]
    DESCRIPTION_FIELD_NUMBER: _ClassVar[int]
    ITEM_ID_FIELD_NUMBER: _ClassVar[int]
    AMOUNT_FIELD_NUMBER: _ClassVar[int]
    UNIT_FIELD_NUMBER: _ClassVar[int]
    PRICE_FIELD_NUMBER: _ClassVar[int]
    STATUS_FIELD_NUMBER: _ClassVar[int]
    EVENT_ID_FIELD_NUMBER: _ClassVar[int]
    CREATED_AT_FIELD_NUMBER: _ClassVar[int]
    APPLIED_AT_FIELD_NUMBER: _ClassVar[int]
    id: str
    workspace_id: str
    source_id: str
    transaction_id: str
    line_key: str
    description: str
    item_id: str
    amount: str
    unit: str
    price: str
    status: str
    event_id: str
    created_at: str
    applied_at: str
    def __init__(self, id: _Optional[str] = ..., workspace_id: _Optional[str] = ..., source_id: _Optional[str] = ..., transaction_id: _Optional[str] = ..., line_key: _Optional[str] = ..., description: _Optional[str] = ..., item_id: _Optional[str] = ..., amount: _Optional[str] = ..., unit: _Optional[str] = ..., price: _Optional[str] = ..., status: _Optional[str] = ..., event_id: _Optional[str] = ..., created_at: _Optional[str] = ..., applied_at: _Optional[str] = ...) -> None: ...

class StageReceiptProposalRequest(_message.Message):
    __slots__ = ("workspace_id", "source_id", "transaction_id", "line_key", "description", "item_id", "amount", "unit", "price")
    WORKSPACE_ID_FIELD_NUMBER: _ClassVar[int]
    SOURCE_ID_FIELD_NUMBER: _ClassVar[int]
    TRANSACTION_ID_FIELD_NUMBER: _ClassVar[int]
    LINE_KEY_FIELD_NUMBER: _ClassVar[int]
    DESCRIPTION_FIELD_NUMBER: _ClassVar[int]
    ITEM_ID_FIELD_NUMBER: _ClassVar[int]
    AMOUNT_FIELD_NUMBER: _ClassVar[int]
    UNIT_FIELD_NUMBER: _ClassVar[int]
    PRICE_FIELD_NUMBER: _ClassVar[int]
    workspace_id: str
    source_id: str
    transaction_id: str
    line_key: str
    description: str
    item_id: str
    amount: str
    unit: str
    price: str
    def __init__(self, workspace_id: _Optional[str] = ..., source_id: _Optional[str] = ..., transaction_id: _Optional[str] = ..., line_key: _Optional[str] = ..., description: _Optional[str] = ..., item_id: _Optional[str] = ..., amount: _Optional[str] = ..., unit: _Optional[str] = ..., price: _Optional[str] = ...) -> None: ...

class ListReceiptProposalsRequest(_message.Message):
    __slots__ = ("workspace_id",)
    WORKSPACE_ID_FIELD_NUMBER: _ClassVar[int]
    workspace_id: str
    def __init__(self, workspace_id: _Optional[str] = ...) -> None: ...

class ListReceiptProposalsResponse(_message.Message):
    __slots__ = ("proposals",)
    PROPOSALS_FIELD_NUMBER: _ClassVar[int]
    proposals: _containers.RepeatedCompositeFieldContainer[ReceiptProposal]
    def __init__(self, proposals: _Optional[_Iterable[_Union[ReceiptProposal, _Mapping]]] = ...) -> None: ...

class ApplyReceiptProposalRequest(_message.Message):
    __slots__ = ("workspace_id", "proposal_id")
    WORKSPACE_ID_FIELD_NUMBER: _ClassVar[int]
    PROPOSAL_ID_FIELD_NUMBER: _ClassVar[int]
    workspace_id: str
    proposal_id: str
    def __init__(self, workspace_id: _Optional[str] = ..., proposal_id: _Optional[str] = ...) -> None: ...

class ReceiptProposalResponse(_message.Message):
    __slots__ = ("proposal",)
    PROPOSAL_FIELD_NUMBER: _ClassVar[int]
    proposal: ReceiptProposal
    def __init__(self, proposal: _Optional[_Union[ReceiptProposal, _Mapping]] = ...) -> None: ...
