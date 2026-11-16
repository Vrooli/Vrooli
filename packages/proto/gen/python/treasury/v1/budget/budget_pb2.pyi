import datetime

from google.protobuf import timestamp_pb2 as _timestamp_pb2
from google.protobuf.internal import containers as _containers
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Iterable as _Iterable, Mapping as _Mapping
from typing import ClassVar as _ClassVar, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class Budget(_message.Message):
    __slots__ = ("id", "book_id", "total_cap_minor", "periodic_cap_minor", "per_transaction_cap_minor", "currency", "allowed_counterparties", "denied_counterparties", "requires_approval", "frozen", "period_seconds")
    ID_FIELD_NUMBER: _ClassVar[int]
    BOOK_ID_FIELD_NUMBER: _ClassVar[int]
    TOTAL_CAP_MINOR_FIELD_NUMBER: _ClassVar[int]
    PERIODIC_CAP_MINOR_FIELD_NUMBER: _ClassVar[int]
    PER_TRANSACTION_CAP_MINOR_FIELD_NUMBER: _ClassVar[int]
    CURRENCY_FIELD_NUMBER: _ClassVar[int]
    ALLOWED_COUNTERPARTIES_FIELD_NUMBER: _ClassVar[int]
    DENIED_COUNTERPARTIES_FIELD_NUMBER: _ClassVar[int]
    REQUIRES_APPROVAL_FIELD_NUMBER: _ClassVar[int]
    FROZEN_FIELD_NUMBER: _ClassVar[int]
    PERIOD_SECONDS_FIELD_NUMBER: _ClassVar[int]
    id: str
    book_id: str
    total_cap_minor: int
    periodic_cap_minor: int
    per_transaction_cap_minor: int
    currency: str
    allowed_counterparties: _containers.RepeatedScalarFieldContainer[str]
    denied_counterparties: _containers.RepeatedScalarFieldContainer[str]
    requires_approval: bool
    frozen: bool
    period_seconds: int
    def __init__(self, id: _Optional[str] = ..., book_id: _Optional[str] = ..., total_cap_minor: _Optional[int] = ..., periodic_cap_minor: _Optional[int] = ..., per_transaction_cap_minor: _Optional[int] = ..., currency: _Optional[str] = ..., allowed_counterparties: _Optional[_Iterable[str]] = ..., denied_counterparties: _Optional[_Iterable[str]] = ..., requires_approval: _Optional[bool] = ..., frozen: _Optional[bool] = ..., period_seconds: _Optional[int] = ...) -> None: ...

class SetBudgetCapsRequest(_message.Message):
    __slots__ = ("budget",)
    BUDGET_FIELD_NUMBER: _ClassVar[int]
    budget: Budget
    def __init__(self, budget: _Optional[_Union[Budget, _Mapping]] = ...) -> None: ...

class SetBudgetCapsResponse(_message.Message):
    __slots__ = ("budget",)
    BUDGET_FIELD_NUMBER: _ClassVar[int]
    budget: Budget
    def __init__(self, budget: _Optional[_Union[Budget, _Mapping]] = ...) -> None: ...

class SetGatingRequest(_message.Message):
    __slots__ = ("budget_id", "requires_approval")
    BUDGET_ID_FIELD_NUMBER: _ClassVar[int]
    REQUIRES_APPROVAL_FIELD_NUMBER: _ClassVar[int]
    budget_id: str
    requires_approval: bool
    def __init__(self, budget_id: _Optional[str] = ..., requires_approval: _Optional[bool] = ...) -> None: ...

class SetGatingResponse(_message.Message):
    __slots__ = ("budget",)
    BUDGET_FIELD_NUMBER: _ClassVar[int]
    budget: Budget
    def __init__(self, budget: _Optional[_Union[Budget, _Mapping]] = ...) -> None: ...

class FreezeBudgetRequest(_message.Message):
    __slots__ = ("budget_id",)
    BUDGET_ID_FIELD_NUMBER: _ClassVar[int]
    budget_id: str
    def __init__(self, budget_id: _Optional[str] = ...) -> None: ...

class FreezeBudgetResponse(_message.Message):
    __slots__ = ("budget",)
    BUDGET_FIELD_NUMBER: _ClassVar[int]
    budget: Budget
    def __init__(self, budget: _Optional[_Union[Budget, _Mapping]] = ...) -> None: ...

class UnfreezeBudgetRequest(_message.Message):
    __slots__ = ("budget_id",)
    BUDGET_ID_FIELD_NUMBER: _ClassVar[int]
    budget_id: str
    def __init__(self, budget_id: _Optional[str] = ...) -> None: ...

class UnfreezeBudgetResponse(_message.Message):
    __slots__ = ("budget",)
    BUDGET_FIELD_NUMBER: _ClassVar[int]
    budget: Budget
    def __init__(self, budget: _Optional[_Union[Budget, _Mapping]] = ...) -> None: ...

class FreezeBookRequest(_message.Message):
    __slots__ = ("book_id",)
    BOOK_ID_FIELD_NUMBER: _ClassVar[int]
    book_id: str
    def __init__(self, book_id: _Optional[str] = ...) -> None: ...

class FreezeBookResponse(_message.Message):
    __slots__ = ("book_id", "frozen", "updated_at")
    BOOK_ID_FIELD_NUMBER: _ClassVar[int]
    FROZEN_FIELD_NUMBER: _ClassVar[int]
    UPDATED_AT_FIELD_NUMBER: _ClassVar[int]
    book_id: str
    frozen: bool
    updated_at: _timestamp_pb2.Timestamp
    def __init__(self, book_id: _Optional[str] = ..., frozen: _Optional[bool] = ..., updated_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ...) -> None: ...

class UnfreezeBookRequest(_message.Message):
    __slots__ = ("book_id",)
    BOOK_ID_FIELD_NUMBER: _ClassVar[int]
    book_id: str
    def __init__(self, book_id: _Optional[str] = ...) -> None: ...

class UnfreezeBookResponse(_message.Message):
    __slots__ = ("book_id", "frozen", "updated_at")
    BOOK_ID_FIELD_NUMBER: _ClassVar[int]
    FROZEN_FIELD_NUMBER: _ClassVar[int]
    UPDATED_AT_FIELD_NUMBER: _ClassVar[int]
    book_id: str
    frozen: bool
    updated_at: _timestamp_pb2.Timestamp
    def __init__(self, book_id: _Optional[str] = ..., frozen: _Optional[bool] = ..., updated_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ...) -> None: ...

class FreezeAllRequest(_message.Message):
    __slots__ = ()
    def __init__(self) -> None: ...

class FreezeAllResponse(_message.Message):
    __slots__ = ("frozen", "updated_at")
    FROZEN_FIELD_NUMBER: _ClassVar[int]
    UPDATED_AT_FIELD_NUMBER: _ClassVar[int]
    frozen: bool
    updated_at: _timestamp_pb2.Timestamp
    def __init__(self, frozen: _Optional[bool] = ..., updated_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ...) -> None: ...

class GetFreezeStatusRequest(_message.Message):
    __slots__ = ()
    def __init__(self) -> None: ...

class GetFreezeStatusResponse(_message.Message):
    __slots__ = ("frozen", "updated_at")
    FROZEN_FIELD_NUMBER: _ClassVar[int]
    UPDATED_AT_FIELD_NUMBER: _ClassVar[int]
    frozen: bool
    updated_at: _timestamp_pb2.Timestamp
    def __init__(self, frozen: _Optional[bool] = ..., updated_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ...) -> None: ...

class UnfreezeAllRequest(_message.Message):
    __slots__ = ()
    def __init__(self) -> None: ...

class UnfreezeAllResponse(_message.Message):
    __slots__ = ("frozen", "updated_at")
    FROZEN_FIELD_NUMBER: _ClassVar[int]
    UPDATED_AT_FIELD_NUMBER: _ClassVar[int]
    frozen: bool
    updated_at: _timestamp_pb2.Timestamp
    def __init__(self, frozen: _Optional[bool] = ..., updated_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ...) -> None: ...
