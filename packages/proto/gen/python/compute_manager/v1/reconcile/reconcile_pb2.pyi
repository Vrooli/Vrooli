import datetime

from google.protobuf import timestamp_pb2 as _timestamp_pb2
from google.protobuf.internal import containers as _containers
from google.protobuf.internal import enum_type_wrapper as _enum_type_wrapper
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Iterable as _Iterable, Mapping as _Mapping
from typing import ClassVar as _ClassVar, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class FindingKind(int, metaclass=_enum_type_wrapper.EnumTypeWrapper):
    __slots__ = ()
    FINDING_KIND_UNSPECIFIED: _ClassVar[FindingKind]
    FINDING_KIND_UNACCOUNTED_AT_PROVIDER: _ClassVar[FindingKind]
    FINDING_KIND_DESTROYED_OUT_OF_BAND: _ClassVar[FindingKind]
    FINDING_KIND_STATE_DIVERGENCE: _ClassVar[FindingKind]
    FINDING_KIND_COST_DIVERGENCE: _ClassVar[FindingKind]
FINDING_KIND_UNSPECIFIED: FindingKind
FINDING_KIND_UNACCOUNTED_AT_PROVIDER: FindingKind
FINDING_KIND_DESTROYED_OUT_OF_BAND: FindingKind
FINDING_KIND_STATE_DIVERGENCE: FindingKind
FINDING_KIND_COST_DIVERGENCE: FindingKind

class Finding(_message.Message):
    __slots__ = ("id", "kind", "provider", "provider_instance_id", "instance_id", "status", "detail", "observed_at")
    ID_FIELD_NUMBER: _ClassVar[int]
    KIND_FIELD_NUMBER: _ClassVar[int]
    PROVIDER_FIELD_NUMBER: _ClassVar[int]
    PROVIDER_INSTANCE_ID_FIELD_NUMBER: _ClassVar[int]
    INSTANCE_ID_FIELD_NUMBER: _ClassVar[int]
    STATUS_FIELD_NUMBER: _ClassVar[int]
    DETAIL_FIELD_NUMBER: _ClassVar[int]
    OBSERVED_AT_FIELD_NUMBER: _ClassVar[int]
    id: str
    kind: FindingKind
    provider: str
    provider_instance_id: str
    instance_id: str
    status: str
    detail: str
    observed_at: _timestamp_pb2.Timestamp
    def __init__(self, id: _Optional[str] = ..., kind: _Optional[_Union[FindingKind, str]] = ..., provider: _Optional[str] = ..., provider_instance_id: _Optional[str] = ..., instance_id: _Optional[str] = ..., status: _Optional[str] = ..., detail: _Optional[str] = ..., observed_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ...) -> None: ...

class RunReconciliationRequest(_message.Message):
    __slots__ = ()
    def __init__(self) -> None: ...

class RunReconciliationResponse(_message.Message):
    __slots__ = ("findings",)
    FINDINGS_FIELD_NUMBER: _ClassVar[int]
    findings: _containers.RepeatedCompositeFieldContainer[Finding]
    def __init__(self, findings: _Optional[_Iterable[_Union[Finding, _Mapping]]] = ...) -> None: ...

class ListFindingsRequest(_message.Message):
    __slots__ = ("status",)
    STATUS_FIELD_NUMBER: _ClassVar[int]
    status: str
    def __init__(self, status: _Optional[str] = ...) -> None: ...

class ListFindingsResponse(_message.Message):
    __slots__ = ("findings",)
    FINDINGS_FIELD_NUMBER: _ClassVar[int]
    findings: _containers.RepeatedCompositeFieldContainer[Finding]
    def __init__(self, findings: _Optional[_Iterable[_Union[Finding, _Mapping]]] = ...) -> None: ...

class GetFindingRequest(_message.Message):
    __slots__ = ("id",)
    ID_FIELD_NUMBER: _ClassVar[int]
    id: str
    def __init__(self, id: _Optional[str] = ...) -> None: ...

class GetFindingResponse(_message.Message):
    __slots__ = ("finding",)
    FINDING_FIELD_NUMBER: _ClassVar[int]
    finding: Finding
    def __init__(self, finding: _Optional[_Union[Finding, _Mapping]] = ...) -> None: ...

class QuarantineFindingRequest(_message.Message):
    __slots__ = ("id",)
    ID_FIELD_NUMBER: _ClassVar[int]
    id: str
    def __init__(self, id: _Optional[str] = ...) -> None: ...

class QuarantineFindingResponse(_message.Message):
    __slots__ = ("finding",)
    FINDING_FIELD_NUMBER: _ClassVar[int]
    finding: Finding
    def __init__(self, finding: _Optional[_Union[Finding, _Mapping]] = ...) -> None: ...
