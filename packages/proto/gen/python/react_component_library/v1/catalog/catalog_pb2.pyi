from google.protobuf.internal import containers as _containers
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Iterable as _Iterable, Mapping as _Mapping
from typing import ClassVar as _ClassVar, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class GetCoverageRequest(_message.Message):
    __slots__ = ()
    def __init__(self) -> None: ...

class CoverageRow(_message.Message):
    __slots__ = ("asset_id", "name", "domain", "kind", "priority", "bucket", "platform", "target", "achieved", "implementation", "blocks_downstream")
    ASSET_ID_FIELD_NUMBER: _ClassVar[int]
    NAME_FIELD_NUMBER: _ClassVar[int]
    DOMAIN_FIELD_NUMBER: _ClassVar[int]
    KIND_FIELD_NUMBER: _ClassVar[int]
    PRIORITY_FIELD_NUMBER: _ClassVar[int]
    BUCKET_FIELD_NUMBER: _ClassVar[int]
    PLATFORM_FIELD_NUMBER: _ClassVar[int]
    TARGET_FIELD_NUMBER: _ClassVar[int]
    ACHIEVED_FIELD_NUMBER: _ClassVar[int]
    IMPLEMENTATION_FIELD_NUMBER: _ClassVar[int]
    BLOCKS_DOWNSTREAM_FIELD_NUMBER: _ClassVar[int]
    asset_id: str
    name: str
    domain: str
    kind: str
    priority: str
    bucket: str
    platform: str
    target: str
    achieved: str
    implementation: str
    blocks_downstream: int
    def __init__(self, asset_id: _Optional[str] = ..., name: _Optional[str] = ..., domain: _Optional[str] = ..., kind: _Optional[str] = ..., priority: _Optional[str] = ..., bucket: _Optional[str] = ..., platform: _Optional[str] = ..., target: _Optional[str] = ..., achieved: _Optional[str] = ..., implementation: _Optional[str] = ..., blocks_downstream: _Optional[int] = ...) -> None: ...

class Rollup(_message.Message):
    __slots__ = ("key", "planned", "built")
    KEY_FIELD_NUMBER: _ClassVar[int]
    PLANNED_FIELD_NUMBER: _ClassVar[int]
    BUILT_FIELD_NUMBER: _ClassVar[int]
    key: str
    planned: int
    built: int
    def __init__(self, key: _Optional[str] = ..., planned: _Optional[int] = ..., built: _Optional[int] = ...) -> None: ...

class MaturitySummary(_message.Message):
    __slots__ = ("total", "at_or_above_target", "by_rung")
    class ByRungEntry(_message.Message):
        __slots__ = ("key", "value")
        KEY_FIELD_NUMBER: _ClassVar[int]
        VALUE_FIELD_NUMBER: _ClassVar[int]
        key: str
        value: int
        def __init__(self, key: _Optional[str] = ..., value: _Optional[int] = ...) -> None: ...
    TOTAL_FIELD_NUMBER: _ClassVar[int]
    AT_OR_ABOVE_TARGET_FIELD_NUMBER: _ClassVar[int]
    BY_RUNG_FIELD_NUMBER: _ClassVar[int]
    total: int
    at_or_above_target: int
    by_rung: _containers.ScalarMap[str, int]
    def __init__(self, total: _Optional[int] = ..., at_or_above_target: _Optional[int] = ..., by_rung: _Optional[_Mapping[str, int]] = ...) -> None: ...

class CoverageReport(_message.Message):
    __slots__ = ("rows", "totals", "by_domain", "by_priority", "maturity")
    class TotalsEntry(_message.Message):
        __slots__ = ("key", "value")
        KEY_FIELD_NUMBER: _ClassVar[int]
        VALUE_FIELD_NUMBER: _ClassVar[int]
        key: str
        value: int
        def __init__(self, key: _Optional[str] = ..., value: _Optional[int] = ...) -> None: ...
    ROWS_FIELD_NUMBER: _ClassVar[int]
    TOTALS_FIELD_NUMBER: _ClassVar[int]
    BY_DOMAIN_FIELD_NUMBER: _ClassVar[int]
    BY_PRIORITY_FIELD_NUMBER: _ClassVar[int]
    MATURITY_FIELD_NUMBER: _ClassVar[int]
    rows: _containers.RepeatedCompositeFieldContainer[CoverageRow]
    totals: _containers.ScalarMap[str, int]
    by_domain: _containers.RepeatedCompositeFieldContainer[Rollup]
    by_priority: _containers.RepeatedCompositeFieldContainer[Rollup]
    maturity: MaturitySummary
    def __init__(self, rows: _Optional[_Iterable[_Union[CoverageRow, _Mapping]]] = ..., totals: _Optional[_Mapping[str, int]] = ..., by_domain: _Optional[_Iterable[_Union[Rollup, _Mapping]]] = ..., by_priority: _Optional[_Iterable[_Union[Rollup, _Mapping]]] = ..., maturity: _Optional[_Union[MaturitySummary, _Mapping]] = ...) -> None: ...

class GetCoverageResponse(_message.Message):
    __slots__ = ("report",)
    REPORT_FIELD_NUMBER: _ClassVar[int]
    report: CoverageReport
    def __init__(self, report: _Optional[_Union[CoverageReport, _Mapping]] = ...) -> None: ...

class ListNextWorkRequest(_message.Message):
    __slots__ = ("limit",)
    LIMIT_FIELD_NUMBER: _ClassVar[int]
    limit: int
    def __init__(self, limit: _Optional[int] = ...) -> None: ...

class ListNextWorkResponse(_message.Message):
    __slots__ = ("rows", "maturity")
    ROWS_FIELD_NUMBER: _ClassVar[int]
    MATURITY_FIELD_NUMBER: _ClassVar[int]
    rows: _containers.RepeatedCompositeFieldContainer[CoverageRow]
    maturity: MaturitySummary
    def __init__(self, rows: _Optional[_Iterable[_Union[CoverageRow, _Mapping]]] = ..., maturity: _Optional[_Union[MaturitySummary, _Mapping]] = ...) -> None: ...

class RunGateRequest(_message.Message):
    __slots__ = ("gate",)
    GATE_FIELD_NUMBER: _ClassVar[int]
    gate: str
    def __init__(self, gate: _Optional[str] = ...) -> None: ...

class GateFinding(_message.Message):
    __slots__ = ("code", "message", "asset_id", "severity")
    CODE_FIELD_NUMBER: _ClassVar[int]
    MESSAGE_FIELD_NUMBER: _ClassVar[int]
    ASSET_ID_FIELD_NUMBER: _ClassVar[int]
    SEVERITY_FIELD_NUMBER: _ClassVar[int]
    code: str
    message: str
    asset_id: str
    severity: str
    def __init__(self, code: _Optional[str] = ..., message: _Optional[str] = ..., asset_id: _Optional[str] = ..., severity: _Optional[str] = ...) -> None: ...

class RunGateResponse(_message.Message):
    __slots__ = ("gate", "findings")
    GATE_FIELD_NUMBER: _ClassVar[int]
    FINDINGS_FIELD_NUMBER: _ClassVar[int]
    gate: str
    findings: _containers.RepeatedCompositeFieldContainer[GateFinding]
    def __init__(self, gate: _Optional[str] = ..., findings: _Optional[_Iterable[_Union[GateFinding, _Mapping]]] = ...) -> None: ...
