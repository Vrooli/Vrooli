import datetime

from buf.validate import validate_pb2 as _validate_pb2
from google.protobuf import timestamp_pb2 as _timestamp_pb2
from money_ledger.v1.shared import ledger_types_pb2 as _ledger_types_pb2
from google.protobuf.internal import containers as _containers
from google.protobuf.internal import enum_type_wrapper as _enum_type_wrapper
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Iterable as _Iterable, Mapping as _Mapping
from typing import ClassVar as _ClassVar, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class AdapterKind(int, metaclass=_enum_type_wrapper.EnumTypeWrapper):
    __slots__ = ()
    ADAPTER_KIND_UNSPECIFIED: _ClassVar[AdapterKind]
    ADAPTER_KIND_MANUAL: _ClassVar[AdapterKind]
    ADAPTER_KIND_FILE: _ClassVar[AdapterKind]
    ADAPTER_KIND_AGGREGATOR: _ClassVar[AdapterKind]

class SourceMode(int, metaclass=_enum_type_wrapper.EnumTypeWrapper):
    __slots__ = ()
    SOURCE_MODE_UNSPECIFIED: _ClassVar[SourceMode]
    SOURCE_MODE_FIXTURE: _ClassVar[SourceMode]
    SOURCE_MODE_OPERATOR_SUPPLIED: _ClassVar[SourceMode]
ADAPTER_KIND_UNSPECIFIED: AdapterKind
ADAPTER_KIND_MANUAL: AdapterKind
ADAPTER_KIND_FILE: AdapterKind
ADAPTER_KIND_AGGREGATOR: AdapterKind
SOURCE_MODE_UNSPECIFIED: SourceMode
SOURCE_MODE_FIXTURE: SourceMode
SOURCE_MODE_OPERATOR_SUPPLIED: SourceMode

class Adapter(_message.Message):
    __slots__ = ("id", "name", "kind", "enabled", "last_success_at", "availability_reason")
    ID_FIELD_NUMBER: _ClassVar[int]
    NAME_FIELD_NUMBER: _ClassVar[int]
    KIND_FIELD_NUMBER: _ClassVar[int]
    ENABLED_FIELD_NUMBER: _ClassVar[int]
    LAST_SUCCESS_AT_FIELD_NUMBER: _ClassVar[int]
    AVAILABILITY_REASON_FIELD_NUMBER: _ClassVar[int]
    id: str
    name: str
    kind: AdapterKind
    enabled: bool
    last_success_at: _timestamp_pb2.Timestamp
    availability_reason: str
    def __init__(self, id: _Optional[str] = ..., name: _Optional[str] = ..., kind: _Optional[_Union[AdapterKind, str]] = ..., enabled: _Optional[bool] = ..., last_success_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., availability_reason: _Optional[str] = ...) -> None: ...

class Receipt(_message.Message):
    __slots__ = ("id", "adapter_id", "to", "read", "written", "skipped_duplicates", "status", "error", "created_at")
    ID_FIELD_NUMBER: _ClassVar[int]
    ADAPTER_ID_FIELD_NUMBER: _ClassVar[int]
    FROM_FIELD_NUMBER: _ClassVar[int]
    TO_FIELD_NUMBER: _ClassVar[int]
    READ_FIELD_NUMBER: _ClassVar[int]
    WRITTEN_FIELD_NUMBER: _ClassVar[int]
    SKIPPED_DUPLICATES_FIELD_NUMBER: _ClassVar[int]
    STATUS_FIELD_NUMBER: _ClassVar[int]
    ERROR_FIELD_NUMBER: _ClassVar[int]
    CREATED_AT_FIELD_NUMBER: _ClassVar[int]
    id: str
    adapter_id: str
    to: _timestamp_pb2.Timestamp
    read: int
    written: int
    skipped_duplicates: int
    status: str
    error: str
    created_at: _timestamp_pb2.Timestamp
    def __init__(self, id: _Optional[str] = ..., adapter_id: _Optional[str] = ..., to: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., read: _Optional[int] = ..., written: _Optional[int] = ..., skipped_duplicates: _Optional[int] = ..., status: _Optional[str] = ..., error: _Optional[str] = ..., created_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., **kwargs) -> None: ...

class Availability(_message.Message):
    __slots__ = ("adapter_id", "reason", "last_success_at")
    ADAPTER_ID_FIELD_NUMBER: _ClassVar[int]
    REASON_FIELD_NUMBER: _ClassVar[int]
    LAST_SUCCESS_AT_FIELD_NUMBER: _ClassVar[int]
    adapter_id: str
    reason: str
    last_success_at: _timestamp_pb2.Timestamp
    def __init__(self, adapter_id: _Optional[str] = ..., reason: _Optional[str] = ..., last_success_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ...) -> None: ...

class RegisterAdapterRequest(_message.Message):
    __slots__ = ("adapter",)
    ADAPTER_FIELD_NUMBER: _ClassVar[int]
    adapter: Adapter
    def __init__(self, adapter: _Optional[_Union[Adapter, _Mapping]] = ...) -> None: ...

class RegisterAdapterResponse(_message.Message):
    __slots__ = ("adapter",)
    ADAPTER_FIELD_NUMBER: _ClassVar[int]
    adapter: Adapter
    def __init__(self, adapter: _Optional[_Union[Adapter, _Mapping]] = ...) -> None: ...

class ListAdaptersRequest(_message.Message):
    __slots__ = ()
    def __init__(self) -> None: ...

class ListAdaptersResponse(_message.Message):
    __slots__ = ("adapters",)
    ADAPTERS_FIELD_NUMBER: _ClassVar[int]
    adapters: _containers.RepeatedCompositeFieldContainer[Adapter]
    def __init__(self, adapters: _Optional[_Iterable[_Union[Adapter, _Mapping]]] = ...) -> None: ...

class IngestEventRequest(_message.Message):
    __slots__ = ("event",)
    EVENT_FIELD_NUMBER: _ClassVar[int]
    event: _ledger_types_pb2.MoneyEvent
    def __init__(self, event: _Optional[_Union[_ledger_types_pb2.MoneyEvent, _Mapping]] = ...) -> None: ...

class IngestEventResponse(_message.Message):
    __slots__ = ("posting", "duplicate", "receipt")
    POSTING_FIELD_NUMBER: _ClassVar[int]
    DUPLICATE_FIELD_NUMBER: _ClassVar[int]
    RECEIPT_FIELD_NUMBER: _ClassVar[int]
    posting: _ledger_types_pb2.Posting
    duplicate: bool
    receipt: Receipt
    def __init__(self, posting: _Optional[_Union[_ledger_types_pb2.Posting, _Mapping]] = ..., duplicate: _Optional[bool] = ..., receipt: _Optional[_Union[Receipt, _Mapping]] = ...) -> None: ...

class RunAdapterRequest(_message.Message):
    __slots__ = ("adapter_id", "to")
    ADAPTER_ID_FIELD_NUMBER: _ClassVar[int]
    FROM_FIELD_NUMBER: _ClassVar[int]
    TO_FIELD_NUMBER: _ClassVar[int]
    adapter_id: str
    to: _timestamp_pb2.Timestamp
    def __init__(self, adapter_id: _Optional[str] = ..., to: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., **kwargs) -> None: ...

class RunAdapterResponse(_message.Message):
    __slots__ = ("receipt", "availability")
    RECEIPT_FIELD_NUMBER: _ClassVar[int]
    AVAILABILITY_FIELD_NUMBER: _ClassVar[int]
    receipt: Receipt
    availability: _containers.RepeatedCompositeFieldContainer[Availability]
    def __init__(self, receipt: _Optional[_Union[Receipt, _Mapping]] = ..., availability: _Optional[_Iterable[_Union[Availability, _Mapping]]] = ...) -> None: ...

class ImportFileRequest(_message.Message):
    __slots__ = ("adapter_id", "csv", "to")
    ADAPTER_ID_FIELD_NUMBER: _ClassVar[int]
    CSV_FIELD_NUMBER: _ClassVar[int]
    FROM_FIELD_NUMBER: _ClassVar[int]
    TO_FIELD_NUMBER: _ClassVar[int]
    adapter_id: str
    csv: bytes
    to: _timestamp_pb2.Timestamp
    def __init__(self, adapter_id: _Optional[str] = ..., csv: _Optional[bytes] = ..., to: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., **kwargs) -> None: ...

class ImportFileResponse(_message.Message):
    __slots__ = ("receipt",)
    RECEIPT_FIELD_NUMBER: _ClassVar[int]
    receipt: Receipt
    def __init__(self, receipt: _Optional[_Union[Receipt, _Mapping]] = ...) -> None: ...

class OperatorInputField(_message.Message):
    __slots__ = ("path", "status", "written", "unit", "window_days", "observed_at", "kind", "reason")
    PATH_FIELD_NUMBER: _ClassVar[int]
    STATUS_FIELD_NUMBER: _ClassVar[int]
    WRITTEN_FIELD_NUMBER: _ClassVar[int]
    UNIT_FIELD_NUMBER: _ClassVar[int]
    WINDOW_DAYS_FIELD_NUMBER: _ClassVar[int]
    OBSERVED_AT_FIELD_NUMBER: _ClassVar[int]
    KIND_FIELD_NUMBER: _ClassVar[int]
    REASON_FIELD_NUMBER: _ClassVar[int]
    path: str
    status: str
    written: bool
    unit: str
    window_days: int
    observed_at: _timestamp_pb2.Timestamp
    kind: str
    reason: str
    def __init__(self, path: _Optional[str] = ..., status: _Optional[str] = ..., written: _Optional[bool] = ..., unit: _Optional[str] = ..., window_days: _Optional[int] = ..., observed_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., kind: _Optional[str] = ..., reason: _Optional[str] = ...) -> None: ...

class OperatorImportRequest(_message.Message):
    __slots__ = ("source_path", "source_mode", "apply", "adapter_id", "book_id", "account_id", "source_json")
    SOURCE_PATH_FIELD_NUMBER: _ClassVar[int]
    SOURCE_MODE_FIELD_NUMBER: _ClassVar[int]
    APPLY_FIELD_NUMBER: _ClassVar[int]
    ADAPTER_ID_FIELD_NUMBER: _ClassVar[int]
    BOOK_ID_FIELD_NUMBER: _ClassVar[int]
    ACCOUNT_ID_FIELD_NUMBER: _ClassVar[int]
    SOURCE_JSON_FIELD_NUMBER: _ClassVar[int]
    source_path: str
    source_mode: SourceMode
    apply: bool
    adapter_id: str
    book_id: str
    account_id: str
    source_json: bytes
    def __init__(self, source_path: _Optional[str] = ..., source_mode: _Optional[_Union[SourceMode, str]] = ..., apply: _Optional[bool] = ..., adapter_id: _Optional[str] = ..., book_id: _Optional[str] = ..., account_id: _Optional[str] = ..., source_json: _Optional[bytes] = ...) -> None: ...

class OperatorImportResponse(_message.Message):
    __slots__ = ("fields", "read", "written", "findings", "applied")
    FIELDS_FIELD_NUMBER: _ClassVar[int]
    READ_FIELD_NUMBER: _ClassVar[int]
    WRITTEN_FIELD_NUMBER: _ClassVar[int]
    FINDINGS_FIELD_NUMBER: _ClassVar[int]
    APPLIED_FIELD_NUMBER: _ClassVar[int]
    fields: _containers.RepeatedCompositeFieldContainer[OperatorInputField]
    read: int
    written: int
    findings: int
    applied: bool
    def __init__(self, fields: _Optional[_Iterable[_Union[OperatorInputField, _Mapping]]] = ..., read: _Optional[int] = ..., written: _Optional[int] = ..., findings: _Optional[int] = ..., applied: _Optional[bool] = ...) -> None: ...

class OperatorInputStatusRequest(_message.Message):
    __slots__ = ("book_id",)
    BOOK_ID_FIELD_NUMBER: _ClassVar[int]
    book_id: str
    def __init__(self, book_id: _Optional[str] = ...) -> None: ...

class OperatorInputStatusResponse(_message.Message):
    __slots__ = ("fields",)
    FIELDS_FIELD_NUMBER: _ClassVar[int]
    fields: _containers.RepeatedCompositeFieldContainer[OperatorInputField]
    def __init__(self, fields: _Optional[_Iterable[_Union[OperatorInputField, _Mapping]]] = ...) -> None: ...
