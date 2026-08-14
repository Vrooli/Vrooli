import datetime

from google.protobuf import timestamp_pb2 as _timestamp_pb2
from buf.validate import validate_pb2 as _validate_pb2
from money_ledger.v1.shared import ledger_types_pb2 as _ledger_types_pb2
from google.protobuf.internal import containers as _containers
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Iterable as _Iterable, Mapping as _Mapping
from typing import ClassVar as _ClassVar, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class Availability(_message.Message):
    __slots__ = ("adapter_id", "reason", "last_success_at")
    ADAPTER_ID_FIELD_NUMBER: _ClassVar[int]
    REASON_FIELD_NUMBER: _ClassVar[int]
    LAST_SUCCESS_AT_FIELD_NUMBER: _ClassVar[int]
    adapter_id: str
    reason: str
    last_success_at: _timestamp_pb2.Timestamp
    def __init__(self, adapter_id: _Optional[str] = ..., reason: _Optional[str] = ..., last_success_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ...) -> None: ...

class Book(_message.Message):
    __slots__ = ("id", "name", "currency", "created_at")
    ID_FIELD_NUMBER: _ClassVar[int]
    NAME_FIELD_NUMBER: _ClassVar[int]
    CURRENCY_FIELD_NUMBER: _ClassVar[int]
    CREATED_AT_FIELD_NUMBER: _ClassVar[int]
    id: str
    name: str
    currency: str
    created_at: _timestamp_pb2.Timestamp
    def __init__(self, id: _Optional[str] = ..., name: _Optional[str] = ..., currency: _Optional[str] = ..., created_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ...) -> None: ...

class Account(_message.Message):
    __slots__ = ("id", "book_id", "name", "kind", "created_at")
    ID_FIELD_NUMBER: _ClassVar[int]
    BOOK_ID_FIELD_NUMBER: _ClassVar[int]
    NAME_FIELD_NUMBER: _ClassVar[int]
    KIND_FIELD_NUMBER: _ClassVar[int]
    CREATED_AT_FIELD_NUMBER: _ClassVar[int]
    id: str
    book_id: str
    name: str
    kind: str
    created_at: _timestamp_pb2.Timestamp
    def __init__(self, id: _Optional[str] = ..., book_id: _Optional[str] = ..., name: _Optional[str] = ..., kind: _Optional[str] = ..., created_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ...) -> None: ...

class Goal(_message.Message):
    __slots__ = ("id", "name", "metric", "comparator", "threshold_minor", "sustain_periods", "buffer_multiple", "book_id")
    ID_FIELD_NUMBER: _ClassVar[int]
    NAME_FIELD_NUMBER: _ClassVar[int]
    METRIC_FIELD_NUMBER: _ClassVar[int]
    COMPARATOR_FIELD_NUMBER: _ClassVar[int]
    THRESHOLD_MINOR_FIELD_NUMBER: _ClassVar[int]
    SUSTAIN_PERIODS_FIELD_NUMBER: _ClassVar[int]
    BUFFER_MULTIPLE_FIELD_NUMBER: _ClassVar[int]
    BOOK_ID_FIELD_NUMBER: _ClassVar[int]
    id: str
    name: str
    metric: str
    comparator: str
    threshold_minor: int
    sustain_periods: int
    buffer_multiple: float
    book_id: str
    def __init__(self, id: _Optional[str] = ..., name: _Optional[str] = ..., metric: _Optional[str] = ..., comparator: _Optional[str] = ..., threshold_minor: _Optional[int] = ..., sustain_periods: _Optional[int] = ..., buffer_multiple: _Optional[float] = ..., book_id: _Optional[str] = ...) -> None: ...

class GoalVerdict(_message.Message):
    __slots__ = ("goal", "met", "sustained_periods", "explanation", "required_periods")
    GOAL_FIELD_NUMBER: _ClassVar[int]
    MET_FIELD_NUMBER: _ClassVar[int]
    SUSTAINED_PERIODS_FIELD_NUMBER: _ClassVar[int]
    EXPLANATION_FIELD_NUMBER: _ClassVar[int]
    REQUIRED_PERIODS_FIELD_NUMBER: _ClassVar[int]
    goal: Goal
    met: bool
    sustained_periods: int
    explanation: str
    required_periods: int
    def __init__(self, goal: _Optional[_Union[Goal, _Mapping]] = ..., met: _Optional[bool] = ..., sustained_periods: _Optional[int] = ..., explanation: _Optional[str] = ..., required_periods: _Optional[int] = ...) -> None: ...

class CreateBookRequest(_message.Message):
    __slots__ = ("name", "currency")
    NAME_FIELD_NUMBER: _ClassVar[int]
    CURRENCY_FIELD_NUMBER: _ClassVar[int]
    name: str
    currency: str
    def __init__(self, name: _Optional[str] = ..., currency: _Optional[str] = ...) -> None: ...

class CreateBookResponse(_message.Message):
    __slots__ = ("book",)
    BOOK_FIELD_NUMBER: _ClassVar[int]
    book: Book
    def __init__(self, book: _Optional[_Union[Book, _Mapping]] = ...) -> None: ...

class ListBooksRequest(_message.Message):
    __slots__ = ()
    def __init__(self) -> None: ...

class ListBooksResponse(_message.Message):
    __slots__ = ("books",)
    BOOKS_FIELD_NUMBER: _ClassVar[int]
    books: _containers.RepeatedCompositeFieldContainer[Book]
    def __init__(self, books: _Optional[_Iterable[_Union[Book, _Mapping]]] = ...) -> None: ...

class CreateAccountRequest(_message.Message):
    __slots__ = ("book_id", "name", "kind")
    BOOK_ID_FIELD_NUMBER: _ClassVar[int]
    NAME_FIELD_NUMBER: _ClassVar[int]
    KIND_FIELD_NUMBER: _ClassVar[int]
    book_id: str
    name: str
    kind: str
    def __init__(self, book_id: _Optional[str] = ..., name: _Optional[str] = ..., kind: _Optional[str] = ...) -> None: ...

class CreateAccountResponse(_message.Message):
    __slots__ = ("account",)
    ACCOUNT_FIELD_NUMBER: _ClassVar[int]
    account: Account
    def __init__(self, account: _Optional[_Union[Account, _Mapping]] = ...) -> None: ...

class ListAccountsRequest(_message.Message):
    __slots__ = ("book_id",)
    BOOK_ID_FIELD_NUMBER: _ClassVar[int]
    book_id: str
    def __init__(self, book_id: _Optional[str] = ...) -> None: ...

class ListAccountsResponse(_message.Message):
    __slots__ = ("accounts",)
    ACCOUNTS_FIELD_NUMBER: _ClassVar[int]
    accounts: _containers.RepeatedCompositeFieldContainer[Account]
    def __init__(self, accounts: _Optional[_Iterable[_Union[Account, _Mapping]]] = ...) -> None: ...

class GetPostingRequest(_message.Message):
    __slots__ = ("posting_id",)
    POSTING_ID_FIELD_NUMBER: _ClassVar[int]
    posting_id: str
    def __init__(self, posting_id: _Optional[str] = ...) -> None: ...

class GetPostingResponse(_message.Message):
    __slots__ = ("posting",)
    POSTING_FIELD_NUMBER: _ClassVar[int]
    posting: _ledger_types_pb2.Posting
    def __init__(self, posting: _Optional[_Union[_ledger_types_pb2.Posting, _Mapping]] = ...) -> None: ...

class ListPostingsRequest(_message.Message):
    __slots__ = ("account_id", "book_id", "limit", "to")
    ACCOUNT_ID_FIELD_NUMBER: _ClassVar[int]
    BOOK_ID_FIELD_NUMBER: _ClassVar[int]
    LIMIT_FIELD_NUMBER: _ClassVar[int]
    FROM_FIELD_NUMBER: _ClassVar[int]
    TO_FIELD_NUMBER: _ClassVar[int]
    account_id: str
    book_id: str
    limit: int
    to: str
    def __init__(self, account_id: _Optional[str] = ..., book_id: _Optional[str] = ..., limit: _Optional[int] = ..., to: _Optional[str] = ..., **kwargs) -> None: ...

class ListPostingsResponse(_message.Message):
    __slots__ = ("postings",)
    POSTINGS_FIELD_NUMBER: _ClassVar[int]
    postings: _containers.RepeatedCompositeFieldContainer[_ledger_types_pb2.Posting]
    def __init__(self, postings: _Optional[_Iterable[_Union[_ledger_types_pb2.Posting, _Mapping]]] = ...) -> None: ...

class ReversePostingRequest(_message.Message):
    __slots__ = ("posting_id", "reason")
    POSTING_ID_FIELD_NUMBER: _ClassVar[int]
    REASON_FIELD_NUMBER: _ClassVar[int]
    posting_id: str
    reason: str
    def __init__(self, posting_id: _Optional[str] = ..., reason: _Optional[str] = ...) -> None: ...

class ReversePostingResponse(_message.Message):
    __slots__ = ("posting",)
    POSTING_FIELD_NUMBER: _ClassVar[int]
    posting: _ledger_types_pb2.Posting
    def __init__(self, posting: _Optional[_Union[_ledger_types_pb2.Posting, _Mapping]] = ...) -> None: ...

class TransferRequest(_message.Message):
    __slots__ = ("from_account_id", "to_account_id", "amount_minor", "currency", "external_id", "description", "occurred_at")
    FROM_ACCOUNT_ID_FIELD_NUMBER: _ClassVar[int]
    TO_ACCOUNT_ID_FIELD_NUMBER: _ClassVar[int]
    AMOUNT_MINOR_FIELD_NUMBER: _ClassVar[int]
    CURRENCY_FIELD_NUMBER: _ClassVar[int]
    EXTERNAL_ID_FIELD_NUMBER: _ClassVar[int]
    DESCRIPTION_FIELD_NUMBER: _ClassVar[int]
    OCCURRED_AT_FIELD_NUMBER: _ClassVar[int]
    from_account_id: str
    to_account_id: str
    amount_minor: int
    currency: str
    external_id: str
    description: str
    occurred_at: _timestamp_pb2.Timestamp
    def __init__(self, from_account_id: _Optional[str] = ..., to_account_id: _Optional[str] = ..., amount_minor: _Optional[int] = ..., currency: _Optional[str] = ..., external_id: _Optional[str] = ..., description: _Optional[str] = ..., occurred_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ...) -> None: ...

class TransferResponse(_message.Message):
    __slots__ = ("postings",)
    POSTINGS_FIELD_NUMBER: _ClassVar[int]
    postings: _containers.RepeatedCompositeFieldContainer[_ledger_types_pb2.Posting]
    def __init__(self, postings: _Optional[_Iterable[_Union[_ledger_types_pb2.Posting, _Mapping]]] = ...) -> None: ...

class PositionRequest(_message.Message):
    __slots__ = ("book_id", "to")
    BOOK_ID_FIELD_NUMBER: _ClassVar[int]
    FROM_FIELD_NUMBER: _ClassVar[int]
    TO_FIELD_NUMBER: _ClassVar[int]
    book_id: str
    to: str
    def __init__(self, book_id: _Optional[str] = ..., to: _Optional[str] = ..., **kwargs) -> None: ...

class PositionInput(_message.Message):
    __slots__ = ("source", "basis", "age_seconds", "available", "reason")
    SOURCE_FIELD_NUMBER: _ClassVar[int]
    BASIS_FIELD_NUMBER: _ClassVar[int]
    AGE_SECONDS_FIELD_NUMBER: _ClassVar[int]
    AVAILABLE_FIELD_NUMBER: _ClassVar[int]
    REASON_FIELD_NUMBER: _ClassVar[int]
    source: str
    basis: _ledger_types_pb2.Basis
    age_seconds: int
    available: bool
    reason: str
    def __init__(self, source: _Optional[str] = ..., basis: _Optional[_Union[_ledger_types_pb2.Basis, str]] = ..., age_seconds: _Optional[int] = ..., available: _Optional[bool] = ..., reason: _Optional[str] = ...) -> None: ...

class PositionResponse(_message.Message):
    __slots__ = ("cash_minor", "revenue_minor", "expense_minor", "burn_minor", "runway_months", "currency", "partial", "availability", "inputs", "runway_available", "runway_reason")
    CASH_MINOR_FIELD_NUMBER: _ClassVar[int]
    REVENUE_MINOR_FIELD_NUMBER: _ClassVar[int]
    EXPENSE_MINOR_FIELD_NUMBER: _ClassVar[int]
    BURN_MINOR_FIELD_NUMBER: _ClassVar[int]
    RUNWAY_MONTHS_FIELD_NUMBER: _ClassVar[int]
    CURRENCY_FIELD_NUMBER: _ClassVar[int]
    PARTIAL_FIELD_NUMBER: _ClassVar[int]
    AVAILABILITY_FIELD_NUMBER: _ClassVar[int]
    INPUTS_FIELD_NUMBER: _ClassVar[int]
    RUNWAY_AVAILABLE_FIELD_NUMBER: _ClassVar[int]
    RUNWAY_REASON_FIELD_NUMBER: _ClassVar[int]
    cash_minor: int
    revenue_minor: int
    expense_minor: int
    burn_minor: int
    runway_months: float
    currency: str
    partial: bool
    availability: _containers.RepeatedCompositeFieldContainer[Availability]
    inputs: _containers.RepeatedCompositeFieldContainer[PositionInput]
    runway_available: bool
    runway_reason: str
    def __init__(self, cash_minor: _Optional[int] = ..., revenue_minor: _Optional[int] = ..., expense_minor: _Optional[int] = ..., burn_minor: _Optional[int] = ..., runway_months: _Optional[float] = ..., currency: _Optional[str] = ..., partial: _Optional[bool] = ..., availability: _Optional[_Iterable[_Union[Availability, _Mapping]]] = ..., inputs: _Optional[_Iterable[_Union[PositionInput, _Mapping]]] = ..., runway_available: _Optional[bool] = ..., runway_reason: _Optional[str] = ...) -> None: ...

class StatementRequest(_message.Message):
    __slots__ = ("book_id", "to")
    BOOK_ID_FIELD_NUMBER: _ClassVar[int]
    FROM_FIELD_NUMBER: _ClassVar[int]
    TO_FIELD_NUMBER: _ClassVar[int]
    book_id: str
    to: str
    def __init__(self, book_id: _Optional[str] = ..., to: _Optional[str] = ..., **kwargs) -> None: ...

class StatementResponse(_message.Message):
    __slots__ = ("book_id", "currency", "opening_cash_minor", "inflow_minor", "outflow_minor", "closing_cash_minor", "revenue_minor", "expense_minor", "partial", "availability", "assets_minor", "liabilities_minor", "to")
    BOOK_ID_FIELD_NUMBER: _ClassVar[int]
    CURRENCY_FIELD_NUMBER: _ClassVar[int]
    OPENING_CASH_MINOR_FIELD_NUMBER: _ClassVar[int]
    INFLOW_MINOR_FIELD_NUMBER: _ClassVar[int]
    OUTFLOW_MINOR_FIELD_NUMBER: _ClassVar[int]
    CLOSING_CASH_MINOR_FIELD_NUMBER: _ClassVar[int]
    REVENUE_MINOR_FIELD_NUMBER: _ClassVar[int]
    EXPENSE_MINOR_FIELD_NUMBER: _ClassVar[int]
    PARTIAL_FIELD_NUMBER: _ClassVar[int]
    AVAILABILITY_FIELD_NUMBER: _ClassVar[int]
    ASSETS_MINOR_FIELD_NUMBER: _ClassVar[int]
    LIABILITIES_MINOR_FIELD_NUMBER: _ClassVar[int]
    FROM_FIELD_NUMBER: _ClassVar[int]
    TO_FIELD_NUMBER: _ClassVar[int]
    book_id: str
    currency: str
    opening_cash_minor: int
    inflow_minor: int
    outflow_minor: int
    closing_cash_minor: int
    revenue_minor: int
    expense_minor: int
    partial: bool
    availability: _containers.RepeatedCompositeFieldContainer[Availability]
    assets_minor: int
    liabilities_minor: int
    to: str
    def __init__(self, book_id: _Optional[str] = ..., currency: _Optional[str] = ..., opening_cash_minor: _Optional[int] = ..., inflow_minor: _Optional[int] = ..., outflow_minor: _Optional[int] = ..., closing_cash_minor: _Optional[int] = ..., revenue_minor: _Optional[int] = ..., expense_minor: _Optional[int] = ..., partial: _Optional[bool] = ..., availability: _Optional[_Iterable[_Union[Availability, _Mapping]]] = ..., assets_minor: _Optional[int] = ..., liabilities_minor: _Optional[int] = ..., to: _Optional[str] = ..., **kwargs) -> None: ...

class DeclareGoalRequest(_message.Message):
    __slots__ = ("goal",)
    GOAL_FIELD_NUMBER: _ClassVar[int]
    goal: Goal
    def __init__(self, goal: _Optional[_Union[Goal, _Mapping]] = ...) -> None: ...

class DeclareGoalResponse(_message.Message):
    __slots__ = ("goal",)
    GOAL_FIELD_NUMBER: _ClassVar[int]
    goal: Goal
    def __init__(self, goal: _Optional[_Union[Goal, _Mapping]] = ...) -> None: ...

class ListGoalsRequest(_message.Message):
    __slots__ = ("book_id",)
    BOOK_ID_FIELD_NUMBER: _ClassVar[int]
    book_id: str
    def __init__(self, book_id: _Optional[str] = ...) -> None: ...

class ListGoalsResponse(_message.Message):
    __slots__ = ("goals",)
    GOALS_FIELD_NUMBER: _ClassVar[int]
    goals: _containers.RepeatedCompositeFieldContainer[GoalVerdict]
    def __init__(self, goals: _Optional[_Iterable[_Union[GoalVerdict, _Mapping]]] = ...) -> None: ...
