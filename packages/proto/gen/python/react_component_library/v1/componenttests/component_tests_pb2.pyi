import datetime

from google.protobuf import timestamp_pb2 as _timestamp_pb2
from google.protobuf.internal import containers as _containers
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Iterable as _Iterable, Mapping as _Mapping
from typing import ClassVar as _ClassVar, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class RunComponentTestRequest(_message.Message):
    __slots__ = ("component_id", "version", "include_closure")
    COMPONENT_ID_FIELD_NUMBER: _ClassVar[int]
    VERSION_FIELD_NUMBER: _ClassVar[int]
    INCLUDE_CLOSURE_FIELD_NUMBER: _ClassVar[int]
    component_id: str
    version: str
    include_closure: bool
    def __init__(self, component_id: _Optional[str] = ..., version: _Optional[str] = ..., include_closure: _Optional[bool] = ...) -> None: ...

class ComponentTestResult(_message.Message):
    __slots__ = ("stage", "asset_library_id", "version", "subject", "verdict", "message", "remediation", "evidence")
    STAGE_FIELD_NUMBER: _ClassVar[int]
    ASSET_LIBRARY_ID_FIELD_NUMBER: _ClassVar[int]
    VERSION_FIELD_NUMBER: _ClassVar[int]
    SUBJECT_FIELD_NUMBER: _ClassVar[int]
    VERDICT_FIELD_NUMBER: _ClassVar[int]
    MESSAGE_FIELD_NUMBER: _ClassVar[int]
    REMEDIATION_FIELD_NUMBER: _ClassVar[int]
    EVIDENCE_FIELD_NUMBER: _ClassVar[int]
    stage: str
    asset_library_id: str
    version: str
    subject: str
    verdict: str
    message: str
    remediation: str
    evidence: _containers.RepeatedCompositeFieldContainer[ComponentTestEvidence]
    def __init__(self, stage: _Optional[str] = ..., asset_library_id: _Optional[str] = ..., version: _Optional[str] = ..., subject: _Optional[str] = ..., verdict: _Optional[str] = ..., message: _Optional[str] = ..., remediation: _Optional[str] = ..., evidence: _Optional[_Iterable[_Union[ComponentTestEvidence, _Mapping]]] = ...) -> None: ...

class ComponentTestEvidence(_message.Message):
    __slots__ = ("kind", "json")
    KIND_FIELD_NUMBER: _ClassVar[int]
    JSON_FIELD_NUMBER: _ClassVar[int]
    kind: str
    json: str
    def __init__(self, kind: _Optional[str] = ..., json: _Optional[str] = ...) -> None: ...

class ComponentTestArtifact(_message.Message):
    __slots__ = ("kind", "label", "asset_library_id", "version", "reference")
    KIND_FIELD_NUMBER: _ClassVar[int]
    LABEL_FIELD_NUMBER: _ClassVar[int]
    ASSET_LIBRARY_ID_FIELD_NUMBER: _ClassVar[int]
    VERSION_FIELD_NUMBER: _ClassVar[int]
    REFERENCE_FIELD_NUMBER: _ClassVar[int]
    kind: str
    label: str
    asset_library_id: str
    version: str
    reference: str
    def __init__(self, kind: _Optional[str] = ..., label: _Optional[str] = ..., asset_library_id: _Optional[str] = ..., version: _Optional[str] = ..., reference: _Optional[str] = ...) -> None: ...

class ComponentTestReport(_message.Message):
    __slots__ = ("id", "root_library_id", "root_version", "include_closure", "created_at", "verdict", "results", "artifacts")
    ID_FIELD_NUMBER: _ClassVar[int]
    ROOT_LIBRARY_ID_FIELD_NUMBER: _ClassVar[int]
    ROOT_VERSION_FIELD_NUMBER: _ClassVar[int]
    INCLUDE_CLOSURE_FIELD_NUMBER: _ClassVar[int]
    CREATED_AT_FIELD_NUMBER: _ClassVar[int]
    VERDICT_FIELD_NUMBER: _ClassVar[int]
    RESULTS_FIELD_NUMBER: _ClassVar[int]
    ARTIFACTS_FIELD_NUMBER: _ClassVar[int]
    id: str
    root_library_id: str
    root_version: str
    include_closure: bool
    created_at: _timestamp_pb2.Timestamp
    verdict: str
    results: _containers.RepeatedCompositeFieldContainer[ComponentTestResult]
    artifacts: _containers.RepeatedCompositeFieldContainer[ComponentTestArtifact]
    def __init__(self, id: _Optional[str] = ..., root_library_id: _Optional[str] = ..., root_version: _Optional[str] = ..., include_closure: _Optional[bool] = ..., created_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., verdict: _Optional[str] = ..., results: _Optional[_Iterable[_Union[ComponentTestResult, _Mapping]]] = ..., artifacts: _Optional[_Iterable[_Union[ComponentTestArtifact, _Mapping]]] = ...) -> None: ...

class RunComponentTestResponse(_message.Message):
    __slots__ = ("report",)
    REPORT_FIELD_NUMBER: _ClassVar[int]
    report: ComponentTestReport
    def __init__(self, report: _Optional[_Union[ComponentTestReport, _Mapping]] = ...) -> None: ...

class RerunComponentTestRequest(_message.Message):
    __slots__ = ("report_id",)
    REPORT_ID_FIELD_NUMBER: _ClassVar[int]
    report_id: str
    def __init__(self, report_id: _Optional[str] = ...) -> None: ...

class RerunComponentTestResponse(_message.Message):
    __slots__ = ("report",)
    REPORT_FIELD_NUMBER: _ClassVar[int]
    report: ComponentTestReport
    def __init__(self, report: _Optional[_Union[ComponentTestReport, _Mapping]] = ...) -> None: ...

class GetComponentTestReportRequest(_message.Message):
    __slots__ = ("id",)
    ID_FIELD_NUMBER: _ClassVar[int]
    id: str
    def __init__(self, id: _Optional[str] = ...) -> None: ...

class GetComponentTestReportResponse(_message.Message):
    __slots__ = ("report",)
    REPORT_FIELD_NUMBER: _ClassVar[int]
    report: ComponentTestReport
    def __init__(self, report: _Optional[_Union[ComponentTestReport, _Mapping]] = ...) -> None: ...

class ListComponentTestReportsRequest(_message.Message):
    __slots__ = ("component_id", "limit", "version")
    COMPONENT_ID_FIELD_NUMBER: _ClassVar[int]
    LIMIT_FIELD_NUMBER: _ClassVar[int]
    VERSION_FIELD_NUMBER: _ClassVar[int]
    component_id: str
    limit: int
    version: str
    def __init__(self, component_id: _Optional[str] = ..., limit: _Optional[int] = ..., version: _Optional[str] = ...) -> None: ...

class ListComponentTestReportsResponse(_message.Message):
    __slots__ = ("reports",)
    REPORTS_FIELD_NUMBER: _ClassVar[int]
    reports: _containers.RepeatedCompositeFieldContainer[ComponentTestReport]
    def __init__(self, reports: _Optional[_Iterable[_Union[ComponentTestReport, _Mapping]]] = ...) -> None: ...
