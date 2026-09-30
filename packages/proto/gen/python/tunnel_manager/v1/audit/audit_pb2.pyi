from google.protobuf.internal import containers as _containers
from google.protobuf.internal import enum_type_wrapper as _enum_type_wrapper
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Iterable as _Iterable, Mapping as _Mapping
from typing import ClassVar as _ClassVar, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class AuditStatus(int, metaclass=_enum_type_wrapper.EnumTypeWrapper):
    __slots__ = ()
    AUDIT_STATUS_UNSPECIFIED: _ClassVar[AuditStatus]
    AUDIT_STATUS_COMPLIANT: _ClassVar[AuditStatus]
    AUDIT_STATUS_MISMATCH: _ClassVar[AuditStatus]
    AUDIT_STATUS_MISSING_SCENARIO: _ClassVar[AuditStatus]
    AUDIT_STATUS_MISSING_PORT: _ClassVar[AuditStatus]
AUDIT_STATUS_UNSPECIFIED: AuditStatus
AUDIT_STATUS_COMPLIANT: AuditStatus
AUDIT_STATUS_MISMATCH: AuditStatus
AUDIT_STATUS_MISSING_SCENARIO: AuditStatus
AUDIT_STATUS_MISSING_PORT: AuditStatus

class PortAuditResult(_message.Message):
    __slots__ = ("subdomain", "scenario", "expected_port", "actual_port", "status", "detail")
    SUBDOMAIN_FIELD_NUMBER: _ClassVar[int]
    SCENARIO_FIELD_NUMBER: _ClassVar[int]
    EXPECTED_PORT_FIELD_NUMBER: _ClassVar[int]
    ACTUAL_PORT_FIELD_NUMBER: _ClassVar[int]
    STATUS_FIELD_NUMBER: _ClassVar[int]
    DETAIL_FIELD_NUMBER: _ClassVar[int]
    subdomain: str
    scenario: str
    expected_port: int
    actual_port: int
    status: AuditStatus
    detail: str
    def __init__(self, subdomain: _Optional[str] = ..., scenario: _Optional[str] = ..., expected_port: _Optional[int] = ..., actual_port: _Optional[int] = ..., status: _Optional[_Union[AuditStatus, str]] = ..., detail: _Optional[str] = ...) -> None: ...

class RunAuditRequest(_message.Message):
    __slots__ = ()
    def __init__(self) -> None: ...

class RunAuditResponse(_message.Message):
    __slots__ = ("results", "violation_count")
    RESULTS_FIELD_NUMBER: _ClassVar[int]
    VIOLATION_COUNT_FIELD_NUMBER: _ClassVar[int]
    results: _containers.RepeatedCompositeFieldContainer[PortAuditResult]
    violation_count: int
    def __init__(self, results: _Optional[_Iterable[_Union[PortAuditResult, _Mapping]]] = ..., violation_count: _Optional[int] = ...) -> None: ...
