from google.protobuf.internal import containers as _containers
from google.protobuf.internal import enum_type_wrapper as _enum_type_wrapper
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Iterable as _Iterable, Mapping as _Mapping
from typing import ClassVar as _ClassVar, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class ImpactChangeKind(int, metaclass=_enum_type_wrapper.EnumTypeWrapper):
    __slots__ = ()
    IMPACT_CHANGE_KIND_UNSPECIFIED: _ClassVar[ImpactChangeKind]
    IMPACT_CHANGE_KIND_ADD: _ClassVar[ImpactChangeKind]
    IMPACT_CHANGE_KIND_RENAME: _ClassVar[ImpactChangeKind]
    IMPACT_CHANGE_KIND_RENUMBER: _ClassVar[ImpactChangeKind]
    IMPACT_CHANGE_KIND_RETYPE: _ClassVar[ImpactChangeKind]
    IMPACT_CHANGE_KIND_REMOVE: _ClassVar[ImpactChangeKind]
IMPACT_CHANGE_KIND_UNSPECIFIED: ImpactChangeKind
IMPACT_CHANGE_KIND_ADD: ImpactChangeKind
IMPACT_CHANGE_KIND_RENAME: ImpactChangeKind
IMPACT_CHANGE_KIND_RENUMBER: ImpactChangeKind
IMPACT_CHANGE_KIND_RETYPE: ImpactChangeKind
IMPACT_CHANGE_KIND_REMOVE: ImpactChangeKind

class GetImpactRequest(_message.Message):
    __slots__ = ("scenario", "against")
    SCENARIO_FIELD_NUMBER: _ClassVar[int]
    AGAINST_FIELD_NUMBER: _ClassVar[int]
    scenario: str
    against: str
    def __init__(self, scenario: _Optional[str] = ..., against: _Optional[str] = ...) -> None: ...

class GetImpactResponse(_message.Message):
    __slots__ = ("report",)
    REPORT_FIELD_NUMBER: _ClassVar[int]
    report: ImpactReport
    def __init__(self, report: _Optional[_Union[ImpactReport, _Mapping]] = ...) -> None: ...

class ImpactReport(_message.Message):
    __slots__ = ("scenario", "scope", "baseline_sha", "changes", "wire_breaking_count", "json_breaking_count", "scope_kind", "baseline_name", "commits_since_baseline", "likely_stale", "fallback_reason", "unreconciled_consumers", "unreconciled_consumer_count", "stable_unreconciled_breaking_count")
    SCENARIO_FIELD_NUMBER: _ClassVar[int]
    SCOPE_FIELD_NUMBER: _ClassVar[int]
    BASELINE_SHA_FIELD_NUMBER: _ClassVar[int]
    CHANGES_FIELD_NUMBER: _ClassVar[int]
    WIRE_BREAKING_COUNT_FIELD_NUMBER: _ClassVar[int]
    JSON_BREAKING_COUNT_FIELD_NUMBER: _ClassVar[int]
    SCOPE_KIND_FIELD_NUMBER: _ClassVar[int]
    BASELINE_NAME_FIELD_NUMBER: _ClassVar[int]
    COMMITS_SINCE_BASELINE_FIELD_NUMBER: _ClassVar[int]
    LIKELY_STALE_FIELD_NUMBER: _ClassVar[int]
    FALLBACK_REASON_FIELD_NUMBER: _ClassVar[int]
    UNRECONCILED_CONSUMERS_FIELD_NUMBER: _ClassVar[int]
    UNRECONCILED_CONSUMER_COUNT_FIELD_NUMBER: _ClassVar[int]
    STABLE_UNRECONCILED_BREAKING_COUNT_FIELD_NUMBER: _ClassVar[int]
    scenario: str
    scope: str
    baseline_sha: str
    changes: _containers.RepeatedCompositeFieldContainer[ImpactChange]
    wire_breaking_count: int
    json_breaking_count: int
    scope_kind: str
    baseline_name: str
    commits_since_baseline: int
    likely_stale: bool
    fallback_reason: str
    unreconciled_consumers: _containers.RepeatedCompositeFieldContainer[ImpactConsumer]
    unreconciled_consumer_count: int
    stable_unreconciled_breaking_count: int
    def __init__(self, scenario: _Optional[str] = ..., scope: _Optional[str] = ..., baseline_sha: _Optional[str] = ..., changes: _Optional[_Iterable[_Union[ImpactChange, _Mapping]]] = ..., wire_breaking_count: _Optional[int] = ..., json_breaking_count: _Optional[int] = ..., scope_kind: _Optional[str] = ..., baseline_name: _Optional[str] = ..., commits_since_baseline: _Optional[int] = ..., likely_stale: _Optional[bool] = ..., fallback_reason: _Optional[str] = ..., unreconciled_consumers: _Optional[_Iterable[_Union[ImpactConsumer, _Mapping]]] = ..., unreconciled_consumer_count: _Optional[int] = ..., stable_unreconciled_breaking_count: _Optional[int] = ...) -> None: ...

class ImpactChange(_message.Message):
    __slots__ = ("file", "path", "kind", "wire_breaking", "json_breaking", "stability", "message", "unreconciled_consumers")
    FILE_FIELD_NUMBER: _ClassVar[int]
    PATH_FIELD_NUMBER: _ClassVar[int]
    KIND_FIELD_NUMBER: _ClassVar[int]
    WIRE_BREAKING_FIELD_NUMBER: _ClassVar[int]
    JSON_BREAKING_FIELD_NUMBER: _ClassVar[int]
    STABILITY_FIELD_NUMBER: _ClassVar[int]
    MESSAGE_FIELD_NUMBER: _ClassVar[int]
    UNRECONCILED_CONSUMERS_FIELD_NUMBER: _ClassVar[int]
    file: str
    path: str
    kind: ImpactChangeKind
    wire_breaking: bool
    json_breaking: bool
    stability: str
    message: str
    unreconciled_consumers: _containers.RepeatedCompositeFieldContainer[ImpactConsumer]
    def __init__(self, file: _Optional[str] = ..., path: _Optional[str] = ..., kind: _Optional[_Union[ImpactChangeKind, str]] = ..., wire_breaking: _Optional[bool] = ..., json_breaking: _Optional[bool] = ..., stability: _Optional[str] = ..., message: _Optional[str] = ..., unreconciled_consumers: _Optional[_Iterable[_Union[ImpactConsumer, _Mapping]]] = ...) -> None: ...

class ImpactConsumer(_message.Message):
    __slots__ = ("scenario", "from_file", "to_file", "unreconciled")
    SCENARIO_FIELD_NUMBER: _ClassVar[int]
    FROM_FILE_FIELD_NUMBER: _ClassVar[int]
    TO_FILE_FIELD_NUMBER: _ClassVar[int]
    UNRECONCILED_FIELD_NUMBER: _ClassVar[int]
    scenario: str
    from_file: str
    to_file: str
    unreconciled: bool
    def __init__(self, scenario: _Optional[str] = ..., from_file: _Optional[str] = ..., to_file: _Optional[str] = ..., unreconciled: _Optional[bool] = ...) -> None: ...
