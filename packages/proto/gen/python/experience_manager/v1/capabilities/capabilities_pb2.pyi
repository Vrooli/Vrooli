from google.protobuf.internal import containers as _containers
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Iterable as _Iterable, Mapping as _Mapping
from typing import ClassVar as _ClassVar, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class GetStatusRequest(_message.Message):
    __slots__ = ()
    def __init__(self) -> None: ...

class CapabilityStatus(_message.Message):
    __slots__ = ("id", "title", "facets", "provable", "status", "blocking_axis", "blocking_evidence")
    ID_FIELD_NUMBER: _ClassVar[int]
    TITLE_FIELD_NUMBER: _ClassVar[int]
    FACETS_FIELD_NUMBER: _ClassVar[int]
    PROVABLE_FIELD_NUMBER: _ClassVar[int]
    STATUS_FIELD_NUMBER: _ClassVar[int]
    BLOCKING_AXIS_FIELD_NUMBER: _ClassVar[int]
    BLOCKING_EVIDENCE_FIELD_NUMBER: _ClassVar[int]
    id: str
    title: str
    facets: _containers.RepeatedScalarFieldContainer[str]
    provable: bool
    status: str
    blocking_axis: str
    blocking_evidence: str
    def __init__(self, id: _Optional[str] = ..., title: _Optional[str] = ..., facets: _Optional[_Iterable[str]] = ..., provable: _Optional[bool] = ..., status: _Optional[str] = ..., blocking_axis: _Optional[str] = ..., blocking_evidence: _Optional[str] = ...) -> None: ...

class GetStatusResponse(_message.Message):
    __slots__ = ("capabilities", "provable", "total")
    CAPABILITIES_FIELD_NUMBER: _ClassVar[int]
    PROVABLE_FIELD_NUMBER: _ClassVar[int]
    TOTAL_FIELD_NUMBER: _ClassVar[int]
    capabilities: _containers.RepeatedCompositeFieldContainer[CapabilityStatus]
    provable: int
    total: int
    def __init__(self, capabilities: _Optional[_Iterable[_Union[CapabilityStatus, _Mapping]]] = ..., provable: _Optional[int] = ..., total: _Optional[int] = ...) -> None: ...
