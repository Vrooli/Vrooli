import datetime

from google.protobuf import timestamp_pb2 as _timestamp_pb2
from research_desk.v1.activity import activity_pb2 as _activity_pb2
from research_desk.v1.shared import commission_pb2 as _commission_pb2
from research_desk.v1.shared import entry_pb2 as _entry_pb2
from research_desk.v1.shared import evidence_pb2 as _evidence_pb2
from google.protobuf.internal import containers as _containers
from google.protobuf.internal import enum_type_wrapper as _enum_type_wrapper
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Iterable as _Iterable, Mapping as _Mapping
from typing import ClassVar as _ClassVar, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class Delivery(int, metaclass=_enum_type_wrapper.EnumTypeWrapper):
    __slots__ = ()
    DELIVERY_UNSPECIFIED: _ClassVar[Delivery]
    DELIVERY_ASK: _ClassVar[Delivery]
    DELIVERY_NOTICE: _ClassVar[Delivery]
    DELIVERY_NONE: _ClassVar[Delivery]

class DeliveryReason(int, metaclass=_enum_type_wrapper.EnumTypeWrapper):
    __slots__ = ()
    DELIVERY_REASON_UNSPECIFIED: _ClassVar[DeliveryReason]
    DELIVERY_REASON_SCHEDULER_DELIVERED: _ClassVar[DeliveryReason]
    DELIVERY_REASON_ANSWERED: _ClassVar[DeliveryReason]
    DELIVERY_REASON_SECRET: _ClassVar[DeliveryReason]
    DELIVERY_REASON_NO_COMMISSION: _ClassVar[DeliveryReason]
    DELIVERY_REASON_NO_STEERING: _ClassVar[DeliveryReason]
    DELIVERY_REASON_NO_RECOMMENDATION: _ClassVar[DeliveryReason]
    DELIVERY_REASON_ASK: _ClassVar[DeliveryReason]
    DELIVERY_REASON_UNREADABLE: _ClassVar[DeliveryReason]
    DELIVERY_REASON_ID_REUSED: _ClassVar[DeliveryReason]
    DELIVERY_REASON_SWITCH_OFF: _ClassVar[DeliveryReason]
    DELIVERY_REASON_SCHEDULER_GRACE: _ClassVar[DeliveryReason]
    DELIVERY_REASON_HUB_UNAVAILABLE: _ClassVar[DeliveryReason]
    DELIVERY_REASON_IN_HUB: _ClassVar[DeliveryReason]
    DELIVERY_REASON_HUB_DELIVERED: _ClassVar[DeliveryReason]
    DELIVERY_REASON_HUB_ANSWERED: _ClassVar[DeliveryReason]
    DELIVERY_REASON_ORPHANED: _ClassVar[DeliveryReason]
    DELIVERY_REASON_BASELINE: _ClassVar[DeliveryReason]
    DELIVERY_REASON_NOTICE: _ClassVar[DeliveryReason]
    DELIVERY_REASON_OLDER_LIST: _ClassVar[DeliveryReason]
    DELIVERY_REASON_PACKET_NOT_READY: _ClassVar[DeliveryReason]

class NeedsYouKind(int, metaclass=_enum_type_wrapper.EnumTypeWrapper):
    __slots__ = ()
    NEEDS_YOU_KIND_UNSPECIFIED: _ClassVar[NeedsYouKind]
    NEEDS_YOU_KIND_ITEM: _ClassVar[NeedsYouKind]
    NEEDS_YOU_KIND_OLDER_LIST: _ClassVar[NeedsYouKind]

class EntryKind(int, metaclass=_enum_type_wrapper.EnumTypeWrapper):
    __slots__ = ()
    ENTRY_KIND_UNSPECIFIED: _ClassVar[EntryKind]
    ENTRY_KIND_DIRECTION: _ClassVar[EntryKind]
    ENTRY_KIND_WIND_DOWN: _ClassVar[EntryKind]
DELIVERY_UNSPECIFIED: Delivery
DELIVERY_ASK: Delivery
DELIVERY_NOTICE: Delivery
DELIVERY_NONE: Delivery
DELIVERY_REASON_UNSPECIFIED: DeliveryReason
DELIVERY_REASON_SCHEDULER_DELIVERED: DeliveryReason
DELIVERY_REASON_ANSWERED: DeliveryReason
DELIVERY_REASON_SECRET: DeliveryReason
DELIVERY_REASON_NO_COMMISSION: DeliveryReason
DELIVERY_REASON_NO_STEERING: DeliveryReason
DELIVERY_REASON_NO_RECOMMENDATION: DeliveryReason
DELIVERY_REASON_ASK: DeliveryReason
DELIVERY_REASON_UNREADABLE: DeliveryReason
DELIVERY_REASON_ID_REUSED: DeliveryReason
DELIVERY_REASON_SWITCH_OFF: DeliveryReason
DELIVERY_REASON_SCHEDULER_GRACE: DeliveryReason
DELIVERY_REASON_HUB_UNAVAILABLE: DeliveryReason
DELIVERY_REASON_IN_HUB: DeliveryReason
DELIVERY_REASON_HUB_DELIVERED: DeliveryReason
DELIVERY_REASON_HUB_ANSWERED: DeliveryReason
DELIVERY_REASON_ORPHANED: DeliveryReason
DELIVERY_REASON_BASELINE: DeliveryReason
DELIVERY_REASON_NOTICE: DeliveryReason
DELIVERY_REASON_OLDER_LIST: DeliveryReason
DELIVERY_REASON_PACKET_NOT_READY: DeliveryReason
NEEDS_YOU_KIND_UNSPECIFIED: NeedsYouKind
NEEDS_YOU_KIND_ITEM: NeedsYouKind
NEEDS_YOU_KIND_OLDER_LIST: NeedsYouKind
ENTRY_KIND_UNSPECIFIED: EntryKind
ENTRY_KIND_DIRECTION: EntryKind
ENTRY_KIND_WIND_DOWN: EntryKind

class SettleHistoricalStateRequest(_message.Message):
    __slots__ = ("commission_id", "status", "as_of", "reason", "evidence_paths", "dry_run", "activity_start", "activity_end")
    COMMISSION_ID_FIELD_NUMBER: _ClassVar[int]
    STATUS_FIELD_NUMBER: _ClassVar[int]
    AS_OF_FIELD_NUMBER: _ClassVar[int]
    REASON_FIELD_NUMBER: _ClassVar[int]
    EVIDENCE_PATHS_FIELD_NUMBER: _ClassVar[int]
    DRY_RUN_FIELD_NUMBER: _ClassVar[int]
    ACTIVITY_START_FIELD_NUMBER: _ClassVar[int]
    ACTIVITY_END_FIELD_NUMBER: _ClassVar[int]
    commission_id: str
    status: str
    as_of: _timestamp_pb2.Timestamp
    reason: str
    evidence_paths: _containers.RepeatedScalarFieldContainer[str]
    dry_run: bool
    activity_start: _timestamp_pb2.Timestamp
    activity_end: _timestamp_pb2.Timestamp
    def __init__(self, commission_id: _Optional[str] = ..., status: _Optional[str] = ..., as_of: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., reason: _Optional[str] = ..., evidence_paths: _Optional[_Iterable[str]] = ..., dry_run: _Optional[bool] = ..., activity_start: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., activity_end: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ...) -> None: ...

class SettleHistoricalStateResponse(_message.Message):
    __slots__ = ("path", "text", "written", "settlement")
    PATH_FIELD_NUMBER: _ClassVar[int]
    TEXT_FIELD_NUMBER: _ClassVar[int]
    WRITTEN_FIELD_NUMBER: _ClassVar[int]
    SETTLEMENT_FIELD_NUMBER: _ClassVar[int]
    path: str
    text: str
    written: bool
    settlement: _activity_pb2.HistoricalSettlement
    def __init__(self, path: _Optional[str] = ..., text: _Optional[str] = ..., written: _Optional[bool] = ..., settlement: _Optional[_Union[_activity_pb2.HistoricalSettlement, _Mapping]] = ...) -> None: ...

class Option(_message.Message):
    __slots__ = ("key", "label")
    KEY_FIELD_NUMBER: _ClassVar[int]
    LABEL_FIELD_NUMBER: _ClassVar[int]
    key: str
    label: str
    def __init__(self, key: _Optional[str] = ..., label: _Optional[str] = ...) -> None: ...

class CourierStep(_message.Message):
    __slots__ = ("action", "why", "on", "switch_reason", "key", "ask_id", "ask_state", "reason")
    ACTION_FIELD_NUMBER: _ClassVar[int]
    WHY_FIELD_NUMBER: _ClassVar[int]
    ON_FIELD_NUMBER: _ClassVar[int]
    SWITCH_REASON_FIELD_NUMBER: _ClassVar[int]
    KEY_FIELD_NUMBER: _ClassVar[int]
    ASK_ID_FIELD_NUMBER: _ClassVar[int]
    ASK_STATE_FIELD_NUMBER: _ClassVar[int]
    REASON_FIELD_NUMBER: _ClassVar[int]
    action: str
    why: str
    on: bool
    switch_reason: str
    key: str
    ask_id: str
    ask_state: str
    reason: DeliveryReason
    def __init__(self, action: _Optional[str] = ..., why: _Optional[str] = ..., on: _Optional[bool] = ..., switch_reason: _Optional[str] = ..., key: _Optional[str] = ..., ask_id: _Optional[str] = ..., ask_state: _Optional[str] = ..., reason: _Optional[_Union[DeliveryReason, str]] = ...) -> None: ...

class Link(_message.Message):
    __slots__ = ("label", "target")
    LABEL_FIELD_NUMBER: _ClassVar[int]
    TARGET_FIELD_NUMBER: _ClassVar[int]
    label: str
    target: str
    def __init__(self, label: _Optional[str] = ..., target: _Optional[str] = ...) -> None: ...

class NeedsYouItem(_message.Message):
    __slots__ = ("id", "title", "body", "commission_id", "sources", "recommended", "recommended_problem", "options", "packet", "secret", "delivered_by_scheduler", "answerable", "not_answerable_reason", "answer", "read", "delivered", "delivery", "delivery_reason", "courier", "delivery_reason_code", "instruction", "links", "kind")
    ID_FIELD_NUMBER: _ClassVar[int]
    TITLE_FIELD_NUMBER: _ClassVar[int]
    BODY_FIELD_NUMBER: _ClassVar[int]
    COMMISSION_ID_FIELD_NUMBER: _ClassVar[int]
    SOURCES_FIELD_NUMBER: _ClassVar[int]
    RECOMMENDED_FIELD_NUMBER: _ClassVar[int]
    RECOMMENDED_PROBLEM_FIELD_NUMBER: _ClassVar[int]
    OPTIONS_FIELD_NUMBER: _ClassVar[int]
    PACKET_FIELD_NUMBER: _ClassVar[int]
    SECRET_FIELD_NUMBER: _ClassVar[int]
    DELIVERED_BY_SCHEDULER_FIELD_NUMBER: _ClassVar[int]
    ANSWERABLE_FIELD_NUMBER: _ClassVar[int]
    NOT_ANSWERABLE_REASON_FIELD_NUMBER: _ClassVar[int]
    ANSWER_FIELD_NUMBER: _ClassVar[int]
    READ_FIELD_NUMBER: _ClassVar[int]
    DELIVERED_FIELD_NUMBER: _ClassVar[int]
    DELIVERY_FIELD_NUMBER: _ClassVar[int]
    DELIVERY_REASON_FIELD_NUMBER: _ClassVar[int]
    COURIER_FIELD_NUMBER: _ClassVar[int]
    DELIVERY_REASON_CODE_FIELD_NUMBER: _ClassVar[int]
    INSTRUCTION_FIELD_NUMBER: _ClassVar[int]
    LINKS_FIELD_NUMBER: _ClassVar[int]
    KIND_FIELD_NUMBER: _ClassVar[int]
    id: str
    title: str
    body: str
    commission_id: str
    sources: _containers.RepeatedScalarFieldContainer[str]
    recommended: Option
    recommended_problem: str
    options: _containers.RepeatedCompositeFieldContainer[Option]
    packet: str
    secret: str
    delivered_by_scheduler: str
    answerable: bool
    not_answerable_reason: str
    answer: _entry_pb2.OperatorEntry
    read: _evidence_pb2.DerivedFlag
    delivered: _entry_pb2.OperatorEntry
    delivery: Delivery
    delivery_reason: str
    courier: CourierStep
    delivery_reason_code: DeliveryReason
    instruction: str
    links: _containers.RepeatedCompositeFieldContainer[Link]
    kind: NeedsYouKind
    def __init__(self, id: _Optional[str] = ..., title: _Optional[str] = ..., body: _Optional[str] = ..., commission_id: _Optional[str] = ..., sources: _Optional[_Iterable[str]] = ..., recommended: _Optional[_Union[Option, _Mapping]] = ..., recommended_problem: _Optional[str] = ..., options: _Optional[_Iterable[_Union[Option, _Mapping]]] = ..., packet: _Optional[str] = ..., secret: _Optional[str] = ..., delivered_by_scheduler: _Optional[str] = ..., answerable: _Optional[bool] = ..., not_answerable_reason: _Optional[str] = ..., answer: _Optional[_Union[_entry_pb2.OperatorEntry, _Mapping]] = ..., read: _Optional[_Union[_evidence_pb2.DerivedFlag, _Mapping]] = ..., delivered: _Optional[_Union[_entry_pb2.OperatorEntry, _Mapping]] = ..., delivery: _Optional[_Union[Delivery, str]] = ..., delivery_reason: _Optional[str] = ..., courier: _Optional[_Union[CourierStep, _Mapping]] = ..., delivery_reason_code: _Optional[_Union[DeliveryReason, str]] = ..., instruction: _Optional[str] = ..., links: _Optional[_Iterable[_Union[Link, _Mapping]]] = ..., kind: _Optional[_Union[NeedsYouKind, str]] = ...) -> None: ...

class CommissionSwitch(_message.Message):
    __slots__ = ("commission_id", "on", "reason")
    COMMISSION_ID_FIELD_NUMBER: _ClassVar[int]
    ON_FIELD_NUMBER: _ClassVar[int]
    REASON_FIELD_NUMBER: _ClassVar[int]
    commission_id: str
    on: bool
    reason: str
    def __init__(self, commission_id: _Optional[str] = ..., on: _Optional[bool] = ..., reason: _Optional[str] = ...) -> None: ...

class CourierStatus(_message.Message):
    __slots__ = ("global_on", "commissions", "hub_read", "hub_reason", "orphans", "writer", "writer_reason", "switches_file")
    GLOBAL_ON_FIELD_NUMBER: _ClassVar[int]
    COMMISSIONS_FIELD_NUMBER: _ClassVar[int]
    HUB_READ_FIELD_NUMBER: _ClassVar[int]
    HUB_REASON_FIELD_NUMBER: _ClassVar[int]
    ORPHANS_FIELD_NUMBER: _ClassVar[int]
    WRITER_FIELD_NUMBER: _ClassVar[int]
    WRITER_REASON_FIELD_NUMBER: _ClassVar[int]
    SWITCHES_FILE_FIELD_NUMBER: _ClassVar[int]
    global_on: bool
    commissions: _containers.RepeatedCompositeFieldContainer[CommissionSwitch]
    hub_read: bool
    hub_reason: str
    orphans: _containers.RepeatedCompositeFieldContainer[CourierStep]
    writer: bool
    writer_reason: str
    switches_file: str
    def __init__(self, global_on: _Optional[bool] = ..., commissions: _Optional[_Iterable[_Union[CommissionSwitch, _Mapping]]] = ..., hub_read: _Optional[bool] = ..., hub_reason: _Optional[str] = ..., orphans: _Optional[_Iterable[_Union[CourierStep, _Mapping]]] = ..., writer: _Optional[bool] = ..., writer_reason: _Optional[str] = ..., switches_file: _Optional[str] = ...) -> None: ...

class ListNeedsYouRequest(_message.Message):
    __slots__ = ()
    def __init__(self) -> None: ...

class ListNeedsYouResponse(_message.Message):
    __slots__ = ("items", "courier", "unreadable", "entry_chars_max", "item_count", "older_list_count", "observed_at")
    ITEMS_FIELD_NUMBER: _ClassVar[int]
    COURIER_FIELD_NUMBER: _ClassVar[int]
    UNREADABLE_FIELD_NUMBER: _ClassVar[int]
    ENTRY_CHARS_MAX_FIELD_NUMBER: _ClassVar[int]
    ITEM_COUNT_FIELD_NUMBER: _ClassVar[int]
    OLDER_LIST_COUNT_FIELD_NUMBER: _ClassVar[int]
    OBSERVED_AT_FIELD_NUMBER: _ClassVar[int]
    items: _containers.RepeatedCompositeFieldContainer[NeedsYouItem]
    courier: CourierStatus
    unreadable: _containers.RepeatedCompositeFieldContainer[_evidence_pb2.Evidence]
    entry_chars_max: int
    item_count: int
    older_list_count: int
    observed_at: _timestamp_pb2.Timestamp
    def __init__(self, items: _Optional[_Iterable[_Union[NeedsYouItem, _Mapping]]] = ..., courier: _Optional[_Union[CourierStatus, _Mapping]] = ..., unreadable: _Optional[_Iterable[_Union[_evidence_pb2.Evidence, _Mapping]]] = ..., entry_chars_max: _Optional[int] = ..., item_count: _Optional[int] = ..., older_list_count: _Optional[int] = ..., observed_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ...) -> None: ...

class AnswerRequest(_message.Message):
    __slots__ = ("commission_id", "item_id", "option_key", "note", "dry_run")
    COMMISSION_ID_FIELD_NUMBER: _ClassVar[int]
    ITEM_ID_FIELD_NUMBER: _ClassVar[int]
    OPTION_KEY_FIELD_NUMBER: _ClassVar[int]
    NOTE_FIELD_NUMBER: _ClassVar[int]
    DRY_RUN_FIELD_NUMBER: _ClassVar[int]
    commission_id: str
    item_id: str
    option_key: str
    note: str
    dry_run: bool
    def __init__(self, commission_id: _Optional[str] = ..., item_id: _Optional[str] = ..., option_key: _Optional[str] = ..., note: _Optional[str] = ..., dry_run: _Optional[bool] = ...) -> None: ...

class AnswerResponse(_message.Message):
    __slots__ = ("entry", "path", "warning", "text", "written", "reading")
    ENTRY_FIELD_NUMBER: _ClassVar[int]
    PATH_FIELD_NUMBER: _ClassVar[int]
    WARNING_FIELD_NUMBER: _ClassVar[int]
    TEXT_FIELD_NUMBER: _ClassVar[int]
    WRITTEN_FIELD_NUMBER: _ClassVar[int]
    READING_FIELD_NUMBER: _ClassVar[int]
    entry: _entry_pb2.OperatorEntry
    path: str
    warning: str
    text: str
    written: bool
    reading: _commission_pb2.EntryReading
    def __init__(self, entry: _Optional[_Union[_entry_pb2.OperatorEntry, _Mapping]] = ..., path: _Optional[str] = ..., warning: _Optional[str] = ..., text: _Optional[str] = ..., written: _Optional[bool] = ..., reading: _Optional[_Union[_commission_pb2.EntryReading, _Mapping]] = ...) -> None: ...

class AddEntryRequest(_message.Message):
    __slots__ = ("commission_id", "kind", "text", "dry_run")
    COMMISSION_ID_FIELD_NUMBER: _ClassVar[int]
    KIND_FIELD_NUMBER: _ClassVar[int]
    TEXT_FIELD_NUMBER: _ClassVar[int]
    DRY_RUN_FIELD_NUMBER: _ClassVar[int]
    commission_id: str
    kind: EntryKind
    text: str
    dry_run: bool
    def __init__(self, commission_id: _Optional[str] = ..., kind: _Optional[_Union[EntryKind, str]] = ..., text: _Optional[str] = ..., dry_run: _Optional[bool] = ...) -> None: ...

class AddEntryResponse(_message.Message):
    __slots__ = ("entry", "path", "warning", "text", "written", "reading")
    ENTRY_FIELD_NUMBER: _ClassVar[int]
    PATH_FIELD_NUMBER: _ClassVar[int]
    WARNING_FIELD_NUMBER: _ClassVar[int]
    TEXT_FIELD_NUMBER: _ClassVar[int]
    WRITTEN_FIELD_NUMBER: _ClassVar[int]
    READING_FIELD_NUMBER: _ClassVar[int]
    entry: _entry_pb2.OperatorEntry
    path: str
    warning: str
    text: str
    written: bool
    reading: _commission_pb2.EntryReading
    def __init__(self, entry: _Optional[_Union[_entry_pb2.OperatorEntry, _Mapping]] = ..., path: _Optional[str] = ..., warning: _Optional[str] = ..., text: _Optional[str] = ..., written: _Optional[bool] = ..., reading: _Optional[_Union[_commission_pb2.EntryReading, _Mapping]] = ...) -> None: ...

class AddStewardFindingRequest(_message.Message):
    __slots__ = ("commission_id", "text", "dry_run")
    COMMISSION_ID_FIELD_NUMBER: _ClassVar[int]
    TEXT_FIELD_NUMBER: _ClassVar[int]
    DRY_RUN_FIELD_NUMBER: _ClassVar[int]
    commission_id: str
    text: str
    dry_run: bool
    def __init__(self, commission_id: _Optional[str] = ..., text: _Optional[str] = ..., dry_run: _Optional[bool] = ...) -> None: ...

class AddStewardFindingResponse(_message.Message):
    __slots__ = ("path", "text", "written", "agent_run_id")
    PATH_FIELD_NUMBER: _ClassVar[int]
    TEXT_FIELD_NUMBER: _ClassVar[int]
    WRITTEN_FIELD_NUMBER: _ClassVar[int]
    AGENT_RUN_ID_FIELD_NUMBER: _ClassVar[int]
    path: str
    text: str
    written: bool
    agent_run_id: str
    def __init__(self, path: _Optional[str] = ..., text: _Optional[str] = ..., written: _Optional[bool] = ..., agent_run_id: _Optional[str] = ...) -> None: ...

class EnableSteeringRequest(_message.Message):
    __slots__ = ("commission_id", "dry_run")
    COMMISSION_ID_FIELD_NUMBER: _ClassVar[int]
    DRY_RUN_FIELD_NUMBER: _ClassVar[int]
    commission_id: str
    dry_run: bool
    def __init__(self, commission_id: _Optional[str] = ..., dry_run: _Optional[bool] = ...) -> None: ...

class EnableSteeringResponse(_message.Message):
    __slots__ = ("path", "written", "text")
    PATH_FIELD_NUMBER: _ClassVar[int]
    WRITTEN_FIELD_NUMBER: _ClassVar[int]
    TEXT_FIELD_NUMBER: _ClassVar[int]
    path: str
    written: bool
    text: str
    def __init__(self, path: _Optional[str] = ..., written: _Optional[bool] = ..., text: _Optional[str] = ...) -> None: ...

class ReviewFile(_message.Message):
    __slots__ = ("file", "url")
    FILE_FIELD_NUMBER: _ClassVar[int]
    URL_FIELD_NUMBER: _ClassVar[int]
    file: _evidence_pb2.FileFact
    url: str
    def __init__(self, file: _Optional[_Union[_evidence_pb2.FileFact, _Mapping]] = ..., url: _Optional[str] = ...) -> None: ...

class Review(_message.Message):
    __slots__ = ("id", "folder", "title", "review", "review_html", "proposals", "topics", "topics_evidence", "verdicts", "unreadable", "unparsed_headings")
    ID_FIELD_NUMBER: _ClassVar[int]
    FOLDER_FIELD_NUMBER: _ClassVar[int]
    TITLE_FIELD_NUMBER: _ClassVar[int]
    REVIEW_FIELD_NUMBER: _ClassVar[int]
    REVIEW_HTML_FIELD_NUMBER: _ClassVar[int]
    PROPOSALS_FIELD_NUMBER: _ClassVar[int]
    TOPICS_FIELD_NUMBER: _ClassVar[int]
    TOPICS_EVIDENCE_FIELD_NUMBER: _ClassVar[int]
    VERDICTS_FIELD_NUMBER: _ClassVar[int]
    UNREADABLE_FIELD_NUMBER: _ClassVar[int]
    UNPARSED_HEADINGS_FIELD_NUMBER: _ClassVar[int]
    id: str
    folder: str
    title: _evidence_pb2.DerivedText
    review: ReviewFile
    review_html: ReviewFile
    proposals: _containers.RepeatedCompositeFieldContainer[ReviewFile]
    topics: _containers.RepeatedScalarFieldContainer[str]
    topics_evidence: _evidence_pb2.Evidence
    verdicts: _containers.RepeatedCompositeFieldContainer[_entry_pb2.OperatorEntry]
    unreadable: _containers.RepeatedCompositeFieldContainer[_evidence_pb2.Evidence]
    unparsed_headings: _containers.RepeatedCompositeFieldContainer[_entry_pb2.UnparsedHeading]
    def __init__(self, id: _Optional[str] = ..., folder: _Optional[str] = ..., title: _Optional[_Union[_evidence_pb2.DerivedText, _Mapping]] = ..., review: _Optional[_Union[ReviewFile, _Mapping]] = ..., review_html: _Optional[_Union[ReviewFile, _Mapping]] = ..., proposals: _Optional[_Iterable[_Union[ReviewFile, _Mapping]]] = ..., topics: _Optional[_Iterable[str]] = ..., topics_evidence: _Optional[_Union[_evidence_pb2.Evidence, _Mapping]] = ..., verdicts: _Optional[_Iterable[_Union[_entry_pb2.OperatorEntry, _Mapping]]] = ..., unreadable: _Optional[_Iterable[_Union[_evidence_pb2.Evidence, _Mapping]]] = ..., unparsed_headings: _Optional[_Iterable[_Union[_entry_pb2.UnparsedHeading, _Mapping]]] = ...) -> None: ...

class ListReviewsRequest(_message.Message):
    __slots__ = ()
    def __init__(self) -> None: ...

class ListReviewsResponse(_message.Message):
    __slots__ = ("reviews", "unreadable", "entry_chars_max")
    REVIEWS_FIELD_NUMBER: _ClassVar[int]
    UNREADABLE_FIELD_NUMBER: _ClassVar[int]
    ENTRY_CHARS_MAX_FIELD_NUMBER: _ClassVar[int]
    reviews: _containers.RepeatedCompositeFieldContainer[Review]
    unreadable: _containers.RepeatedCompositeFieldContainer[_evidence_pb2.Evidence]
    entry_chars_max: int
    def __init__(self, reviews: _Optional[_Iterable[_Union[Review, _Mapping]]] = ..., unreadable: _Optional[_Iterable[_Union[_evidence_pb2.Evidence, _Mapping]]] = ..., entry_chars_max: _Optional[int] = ...) -> None: ...

class AddVerdictRequest(_message.Message):
    __slots__ = ("review_id", "topic", "text", "dry_run")
    REVIEW_ID_FIELD_NUMBER: _ClassVar[int]
    TOPIC_FIELD_NUMBER: _ClassVar[int]
    TEXT_FIELD_NUMBER: _ClassVar[int]
    DRY_RUN_FIELD_NUMBER: _ClassVar[int]
    review_id: str
    topic: str
    text: str
    dry_run: bool
    def __init__(self, review_id: _Optional[str] = ..., topic: _Optional[str] = ..., text: _Optional[str] = ..., dry_run: _Optional[bool] = ...) -> None: ...

class AddVerdictResponse(_message.Message):
    __slots__ = ("entry", "path", "text", "written")
    ENTRY_FIELD_NUMBER: _ClassVar[int]
    PATH_FIELD_NUMBER: _ClassVar[int]
    TEXT_FIELD_NUMBER: _ClassVar[int]
    WRITTEN_FIELD_NUMBER: _ClassVar[int]
    entry: _entry_pb2.OperatorEntry
    path: str
    text: str
    written: bool
    def __init__(self, entry: _Optional[_Union[_entry_pb2.OperatorEntry, _Mapping]] = ..., path: _Optional[str] = ..., text: _Optional[str] = ..., written: _Optional[bool] = ...) -> None: ...

class SetCourierRequest(_message.Message):
    __slots__ = ("commission_id", "on")
    COMMISSION_ID_FIELD_NUMBER: _ClassVar[int]
    GLOBAL_FIELD_NUMBER: _ClassVar[int]
    ON_FIELD_NUMBER: _ClassVar[int]
    commission_id: str
    on: bool
    def __init__(self, commission_id: _Optional[str] = ..., on: _Optional[bool] = ..., **kwargs) -> None: ...

class SetCourierResponse(_message.Message):
    __slots__ = ("on", "reason", "switches_file")
    ON_FIELD_NUMBER: _ClassVar[int]
    REASON_FIELD_NUMBER: _ClassVar[int]
    SWITCHES_FILE_FIELD_NUMBER: _ClassVar[int]
    on: bool
    reason: str
    switches_file: str
    def __init__(self, on: _Optional[bool] = ..., reason: _Optional[str] = ..., switches_file: _Optional[str] = ...) -> None: ...

class TestCourierRequest(_message.Message):
    __slots__ = ()
    def __init__(self) -> None: ...

class TestCourierResponse(_message.Message):
    __slots__ = ("ask_id", "key", "first_delivered_at", "waited_seconds", "expires_at")
    ASK_ID_FIELD_NUMBER: _ClassVar[int]
    KEY_FIELD_NUMBER: _ClassVar[int]
    FIRST_DELIVERED_AT_FIELD_NUMBER: _ClassVar[int]
    WAITED_SECONDS_FIELD_NUMBER: _ClassVar[int]
    EXPIRES_AT_FIELD_NUMBER: _ClassVar[int]
    ask_id: str
    key: str
    first_delivered_at: _timestamp_pb2.Timestamp
    waited_seconds: int
    expires_at: _timestamp_pb2.Timestamp
    def __init__(self, ask_id: _Optional[str] = ..., key: _Optional[str] = ..., first_delivered_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., waited_seconds: _Optional[int] = ..., expires_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ...) -> None: ...

class MarkSeenRequest(_message.Message):
    __slots__ = ("subject", "id", "all")
    SUBJECT_FIELD_NUMBER: _ClassVar[int]
    ID_FIELD_NUMBER: _ClassVar[int]
    ALL_FIELD_NUMBER: _ClassVar[int]
    subject: _activity_pb2.ChangeSubject
    id: str
    all: bool
    def __init__(self, subject: _Optional[_Union[_activity_pb2.ChangeSubject, str]] = ..., id: _Optional[str] = ..., all: _Optional[bool] = ...) -> None: ...

class MarkSeenResponse(_message.Message):
    __slots__ = ("at", "path", "commissions", "topics")
    AT_FIELD_NUMBER: _ClassVar[int]
    PATH_FIELD_NUMBER: _ClassVar[int]
    COMMISSIONS_FIELD_NUMBER: _ClassVar[int]
    TOPICS_FIELD_NUMBER: _ClassVar[int]
    at: _timestamp_pb2.Timestamp
    path: str
    commissions: int
    topics: int
    def __init__(self, at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., path: _Optional[str] = ..., commissions: _Optional[int] = ..., topics: _Optional[int] = ...) -> None: ...
