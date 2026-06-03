import datetime

from buf.validate import validate_pb2 as _validate_pb2
from google.protobuf import timestamp_pb2 as _timestamp_pb2
from google.protobuf.internal import containers as _containers
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Iterable as _Iterable, Mapping as _Mapping
from typing import ClassVar as _ClassVar, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class RetentionPolicy(_message.Message):
    __slots__ = ("keep_latest",)
    KEEP_LATEST_FIELD_NUMBER: _ClassVar[int]
    keep_latest: int
    def __init__(self, keep_latest: _Optional[int] = ...) -> None: ...

class Plan(_message.Message):
    __slots__ = ("id", "name", "target_ids", "destination_ids", "schedule", "retention", "enabled", "created_at", "updated_at")
    ID_FIELD_NUMBER: _ClassVar[int]
    NAME_FIELD_NUMBER: _ClassVar[int]
    TARGET_IDS_FIELD_NUMBER: _ClassVar[int]
    DESTINATION_IDS_FIELD_NUMBER: _ClassVar[int]
    SCHEDULE_FIELD_NUMBER: _ClassVar[int]
    RETENTION_FIELD_NUMBER: _ClassVar[int]
    ENABLED_FIELD_NUMBER: _ClassVar[int]
    CREATED_AT_FIELD_NUMBER: _ClassVar[int]
    UPDATED_AT_FIELD_NUMBER: _ClassVar[int]
    id: str
    name: str
    target_ids: _containers.RepeatedScalarFieldContainer[str]
    destination_ids: _containers.RepeatedScalarFieldContainer[str]
    schedule: str
    retention: RetentionPolicy
    enabled: bool
    created_at: _timestamp_pb2.Timestamp
    updated_at: _timestamp_pb2.Timestamp
    def __init__(self, id: _Optional[str] = ..., name: _Optional[str] = ..., target_ids: _Optional[_Iterable[str]] = ..., destination_ids: _Optional[_Iterable[str]] = ..., schedule: _Optional[str] = ..., retention: _Optional[_Union[RetentionPolicy, _Mapping]] = ..., enabled: _Optional[bool] = ..., created_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., updated_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ...) -> None: ...

class CreatePlanRequest(_message.Message):
    __slots__ = ("name", "target_ids", "destination_ids", "schedule", "retention", "enabled", "allow_incomplete_coverage")
    NAME_FIELD_NUMBER: _ClassVar[int]
    TARGET_IDS_FIELD_NUMBER: _ClassVar[int]
    DESTINATION_IDS_FIELD_NUMBER: _ClassVar[int]
    SCHEDULE_FIELD_NUMBER: _ClassVar[int]
    RETENTION_FIELD_NUMBER: _ClassVar[int]
    ENABLED_FIELD_NUMBER: _ClassVar[int]
    ALLOW_INCOMPLETE_COVERAGE_FIELD_NUMBER: _ClassVar[int]
    name: str
    target_ids: _containers.RepeatedScalarFieldContainer[str]
    destination_ids: _containers.RepeatedScalarFieldContainer[str]
    schedule: str
    retention: RetentionPolicy
    enabled: bool
    allow_incomplete_coverage: bool
    def __init__(self, name: _Optional[str] = ..., target_ids: _Optional[_Iterable[str]] = ..., destination_ids: _Optional[_Iterable[str]] = ..., schedule: _Optional[str] = ..., retention: _Optional[_Union[RetentionPolicy, _Mapping]] = ..., enabled: _Optional[bool] = ..., allow_incomplete_coverage: _Optional[bool] = ...) -> None: ...

class CreatePlanResponse(_message.Message):
    __slots__ = ("plan",)
    PLAN_FIELD_NUMBER: _ClassVar[int]
    plan: Plan
    def __init__(self, plan: _Optional[_Union[Plan, _Mapping]] = ...) -> None: ...

class GetPlanRequest(_message.Message):
    __slots__ = ("id",)
    ID_FIELD_NUMBER: _ClassVar[int]
    id: str
    def __init__(self, id: _Optional[str] = ...) -> None: ...

class GetPlanResponse(_message.Message):
    __slots__ = ("plan",)
    PLAN_FIELD_NUMBER: _ClassVar[int]
    plan: Plan
    def __init__(self, plan: _Optional[_Union[Plan, _Mapping]] = ...) -> None: ...

class ListPlansRequest(_message.Message):
    __slots__ = ("page_size", "page_token")
    PAGE_SIZE_FIELD_NUMBER: _ClassVar[int]
    PAGE_TOKEN_FIELD_NUMBER: _ClassVar[int]
    page_size: int
    page_token: str
    def __init__(self, page_size: _Optional[int] = ..., page_token: _Optional[str] = ...) -> None: ...

class ListPlansResponse(_message.Message):
    __slots__ = ("plans", "next_page_token")
    PLANS_FIELD_NUMBER: _ClassVar[int]
    NEXT_PAGE_TOKEN_FIELD_NUMBER: _ClassVar[int]
    plans: _containers.RepeatedCompositeFieldContainer[Plan]
    next_page_token: str
    def __init__(self, plans: _Optional[_Iterable[_Union[Plan, _Mapping]]] = ..., next_page_token: _Optional[str] = ...) -> None: ...

class UpdatePlanRequest(_message.Message):
    __slots__ = ("id", "name", "target_ids", "destination_ids", "schedule", "retention", "enabled", "allow_incomplete_coverage")
    ID_FIELD_NUMBER: _ClassVar[int]
    NAME_FIELD_NUMBER: _ClassVar[int]
    TARGET_IDS_FIELD_NUMBER: _ClassVar[int]
    DESTINATION_IDS_FIELD_NUMBER: _ClassVar[int]
    SCHEDULE_FIELD_NUMBER: _ClassVar[int]
    RETENTION_FIELD_NUMBER: _ClassVar[int]
    ENABLED_FIELD_NUMBER: _ClassVar[int]
    ALLOW_INCOMPLETE_COVERAGE_FIELD_NUMBER: _ClassVar[int]
    id: str
    name: str
    target_ids: _containers.RepeatedScalarFieldContainer[str]
    destination_ids: _containers.RepeatedScalarFieldContainer[str]
    schedule: str
    retention: RetentionPolicy
    enabled: bool
    allow_incomplete_coverage: bool
    def __init__(self, id: _Optional[str] = ..., name: _Optional[str] = ..., target_ids: _Optional[_Iterable[str]] = ..., destination_ids: _Optional[_Iterable[str]] = ..., schedule: _Optional[str] = ..., retention: _Optional[_Union[RetentionPolicy, _Mapping]] = ..., enabled: _Optional[bool] = ..., allow_incomplete_coverage: _Optional[bool] = ...) -> None: ...

class UpdatePlanResponse(_message.Message):
    __slots__ = ("plan",)
    PLAN_FIELD_NUMBER: _ClassVar[int]
    plan: Plan
    def __init__(self, plan: _Optional[_Union[Plan, _Mapping]] = ...) -> None: ...

class DeletePlanRequest(_message.Message):
    __slots__ = ("id",)
    ID_FIELD_NUMBER: _ClassVar[int]
    id: str
    def __init__(self, id: _Optional[str] = ...) -> None: ...

class DeletePlanResponse(_message.Message):
    __slots__ = ("removed",)
    REMOVED_FIELD_NUMBER: _ClassVar[int]
    removed: bool
    def __init__(self, removed: _Optional[bool] = ...) -> None: ...
