from system_monitor.v1.domain import reports_pb2 as _reports_pb2
from google.protobuf.internal import containers as _containers
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Iterable as _Iterable, Mapping as _Mapping
from typing import ClassVar as _ClassVar, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class GenerateReportRequest(_message.Message):
    __slots__ = ("type",)
    TYPE_FIELD_NUMBER: _ClassVar[int]
    type: str
    def __init__(self, type: _Optional[str] = ...) -> None: ...

class GenerateReportResponse(_message.Message):
    __slots__ = ("report",)
    REPORT_FIELD_NUMBER: _ClassVar[int]
    report: _reports_pb2.EnhancedSystemReport
    def __init__(self, report: _Optional[_Union[_reports_pb2.EnhancedSystemReport, _Mapping]] = ...) -> None: ...

class ListReportsRequest(_message.Message):
    __slots__ = ()
    def __init__(self) -> None: ...

class ListReportsResponse(_message.Message):
    __slots__ = ("reports", "count")
    REPORTS_FIELD_NUMBER: _ClassVar[int]
    COUNT_FIELD_NUMBER: _ClassVar[int]
    reports: _containers.RepeatedCompositeFieldContainer[_reports_pb2.EnhancedSystemReport]
    count: int
    def __init__(self, reports: _Optional[_Iterable[_Union[_reports_pb2.EnhancedSystemReport, _Mapping]]] = ..., count: _Optional[int] = ...) -> None: ...

class GetReportRequest(_message.Message):
    __slots__ = ("id",)
    ID_FIELD_NUMBER: _ClassVar[int]
    id: str
    def __init__(self, id: _Optional[str] = ...) -> None: ...

class GetReportResponse(_message.Message):
    __slots__ = ("report",)
    REPORT_FIELD_NUMBER: _ClassVar[int]
    report: _reports_pb2.EnhancedSystemReport
    def __init__(self, report: _Optional[_Union[_reports_pb2.EnhancedSystemReport, _Mapping]] = ...) -> None: ...
