import datetime

from architecture.v1 import findings_pb2 as _findings_pb2
from google.protobuf import timestamp_pb2 as _timestamp_pb2
from google.protobuf.internal import containers as _containers
from google.protobuf.internal import enum_type_wrapper as _enum_type_wrapper
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Iterable as _Iterable, Mapping as _Mapping
from typing import ClassVar as _ClassVar, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class TrackedFindingStatus(int, metaclass=_enum_type_wrapper.EnumTypeWrapper):
    __slots__ = ()
    TRACKED_FINDING_STATUS_UNSPECIFIED: _ClassVar[TrackedFindingStatus]
    TRACKED_FINDING_STATUS_DETECTED: _ClassVar[TrackedFindingStatus]
    TRACKED_FINDING_STATUS_ASSIGNED: _ClassVar[TrackedFindingStatus]
    TRACKED_FINDING_STATUS_SPLIT: _ClassVar[TrackedFindingStatus]
    TRACKED_FINDING_STATUS_RESOLVED: _ClassVar[TrackedFindingStatus]
    TRACKED_FINDING_STATUS_VALIDATED: _ClassVar[TrackedFindingStatus]
    TRACKED_FINDING_STATUS_COMMITTED: _ClassVar[TrackedFindingStatus]
    TRACKED_FINDING_STATUS_FORCE_RESOLVED: _ClassVar[TrackedFindingStatus]

class MigrationLifecycle(int, metaclass=_enum_type_wrapper.EnumTypeWrapper):
    __slots__ = ()
    MIGRATION_LIFECYCLE_UNSPECIFIED: _ClassVar[MigrationLifecycle]
    MIGRATION_LIFECYCLE_OPEN: _ClassVar[MigrationLifecycle]
    MIGRATION_LIFECYCLE_CLOSED: _ClassVar[MigrationLifecycle]
TRACKED_FINDING_STATUS_UNSPECIFIED: TrackedFindingStatus
TRACKED_FINDING_STATUS_DETECTED: TrackedFindingStatus
TRACKED_FINDING_STATUS_ASSIGNED: TrackedFindingStatus
TRACKED_FINDING_STATUS_SPLIT: TrackedFindingStatus
TRACKED_FINDING_STATUS_RESOLVED: TrackedFindingStatus
TRACKED_FINDING_STATUS_VALIDATED: TrackedFindingStatus
TRACKED_FINDING_STATUS_COMMITTED: TrackedFindingStatus
TRACKED_FINDING_STATUS_FORCE_RESOLVED: TrackedFindingStatus
MIGRATION_LIFECYCLE_UNSPECIFIED: MigrationLifecycle
MIGRATION_LIFECYCLE_OPEN: MigrationLifecycle
MIGRATION_LIFECYCLE_CLOSED: MigrationLifecycle

class TrackedFinding(_message.Message):
    __slots__ = ("stable_id", "scenario", "source", "code", "severity", "locations", "domains", "message", "suggestion", "status", "resolution_note", "regressed", "first_seen_at", "updated_at")
    STABLE_ID_FIELD_NUMBER: _ClassVar[int]
    SCENARIO_FIELD_NUMBER: _ClassVar[int]
    SOURCE_FIELD_NUMBER: _ClassVar[int]
    CODE_FIELD_NUMBER: _ClassVar[int]
    SEVERITY_FIELD_NUMBER: _ClassVar[int]
    LOCATIONS_FIELD_NUMBER: _ClassVar[int]
    DOMAINS_FIELD_NUMBER: _ClassVar[int]
    MESSAGE_FIELD_NUMBER: _ClassVar[int]
    SUGGESTION_FIELD_NUMBER: _ClassVar[int]
    STATUS_FIELD_NUMBER: _ClassVar[int]
    RESOLUTION_NOTE_FIELD_NUMBER: _ClassVar[int]
    REGRESSED_FIELD_NUMBER: _ClassVar[int]
    FIRST_SEEN_AT_FIELD_NUMBER: _ClassVar[int]
    UPDATED_AT_FIELD_NUMBER: _ClassVar[int]
    stable_id: str
    scenario: str
    source: str
    code: str
    severity: str
    locations: _containers.RepeatedScalarFieldContainer[str]
    domains: _containers.RepeatedScalarFieldContainer[str]
    message: str
    suggestion: str
    status: TrackedFindingStatus
    resolution_note: str
    regressed: bool
    first_seen_at: _timestamp_pb2.Timestamp
    updated_at: _timestamp_pb2.Timestamp
    def __init__(self, stable_id: _Optional[str] = ..., scenario: _Optional[str] = ..., source: _Optional[str] = ..., code: _Optional[str] = ..., severity: _Optional[str] = ..., locations: _Optional[_Iterable[str]] = ..., domains: _Optional[_Iterable[str]] = ..., message: _Optional[str] = ..., suggestion: _Optional[str] = ..., status: _Optional[_Union[TrackedFindingStatus, str]] = ..., resolution_note: _Optional[str] = ..., regressed: _Optional[bool] = ..., first_seen_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., updated_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ...) -> None: ...

class Migration(_message.Message):
    __slots__ = ("id", "scenario", "name", "status", "created_at", "updated_at")
    ID_FIELD_NUMBER: _ClassVar[int]
    SCENARIO_FIELD_NUMBER: _ClassVar[int]
    NAME_FIELD_NUMBER: _ClassVar[int]
    STATUS_FIELD_NUMBER: _ClassVar[int]
    CREATED_AT_FIELD_NUMBER: _ClassVar[int]
    UPDATED_AT_FIELD_NUMBER: _ClassVar[int]
    id: str
    scenario: str
    name: str
    status: MigrationLifecycle
    created_at: _timestamp_pb2.Timestamp
    updated_at: _timestamp_pb2.Timestamp
    def __init__(self, id: _Optional[str] = ..., scenario: _Optional[str] = ..., name: _Optional[str] = ..., status: _Optional[_Union[MigrationLifecycle, str]] = ..., created_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., updated_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ...) -> None: ...

class MigrationStatus(_message.Message):
    __slots__ = ("migration", "findings", "total", "open", "resolved", "validated", "regressions", "by_severity", "by_status")
    class BySeverityEntry(_message.Message):
        __slots__ = ("key", "value")
        KEY_FIELD_NUMBER: _ClassVar[int]
        VALUE_FIELD_NUMBER: _ClassVar[int]
        key: str
        value: int
        def __init__(self, key: _Optional[str] = ..., value: _Optional[int] = ...) -> None: ...
    class ByStatusEntry(_message.Message):
        __slots__ = ("key", "value")
        KEY_FIELD_NUMBER: _ClassVar[int]
        VALUE_FIELD_NUMBER: _ClassVar[int]
        key: str
        value: int
        def __init__(self, key: _Optional[str] = ..., value: _Optional[int] = ...) -> None: ...
    MIGRATION_FIELD_NUMBER: _ClassVar[int]
    FINDINGS_FIELD_NUMBER: _ClassVar[int]
    TOTAL_FIELD_NUMBER: _ClassVar[int]
    OPEN_FIELD_NUMBER: _ClassVar[int]
    RESOLVED_FIELD_NUMBER: _ClassVar[int]
    VALIDATED_FIELD_NUMBER: _ClassVar[int]
    REGRESSIONS_FIELD_NUMBER: _ClassVar[int]
    BY_SEVERITY_FIELD_NUMBER: _ClassVar[int]
    BY_STATUS_FIELD_NUMBER: _ClassVar[int]
    migration: Migration
    findings: _containers.RepeatedCompositeFieldContainer[TrackedFinding]
    total: int
    open: int
    resolved: int
    validated: int
    regressions: int
    by_severity: _containers.ScalarMap[str, int]
    by_status: _containers.ScalarMap[str, int]
    def __init__(self, migration: _Optional[_Union[Migration, _Mapping]] = ..., findings: _Optional[_Iterable[_Union[TrackedFinding, _Mapping]]] = ..., total: _Optional[int] = ..., open: _Optional[int] = ..., resolved: _Optional[int] = ..., validated: _Optional[int] = ..., regressions: _Optional[int] = ..., by_severity: _Optional[_Mapping[str, int]] = ..., by_status: _Optional[_Mapping[str, int]] = ...) -> None: ...

class CreateMigrationRequest(_message.Message):
    __slots__ = ("scenario", "name", "findings")
    SCENARIO_FIELD_NUMBER: _ClassVar[int]
    NAME_FIELD_NUMBER: _ClassVar[int]
    FINDINGS_FIELD_NUMBER: _ClassVar[int]
    scenario: str
    name: str
    findings: _containers.RepeatedCompositeFieldContainer[_findings_pb2.ArchitectureFinding]
    def __init__(self, scenario: _Optional[str] = ..., name: _Optional[str] = ..., findings: _Optional[_Iterable[_Union[_findings_pb2.ArchitectureFinding, _Mapping]]] = ...) -> None: ...

class CreateMigrationResponse(_message.Message):
    __slots__ = ("status",)
    STATUS_FIELD_NUMBER: _ClassVar[int]
    status: MigrationStatus
    def __init__(self, status: _Optional[_Union[MigrationStatus, _Mapping]] = ...) -> None: ...

class ListMigrationsRequest(_message.Message):
    __slots__ = ("scenario",)
    SCENARIO_FIELD_NUMBER: _ClassVar[int]
    scenario: str
    def __init__(self, scenario: _Optional[str] = ...) -> None: ...

class ListMigrationsResponse(_message.Message):
    __slots__ = ("migrations",)
    MIGRATIONS_FIELD_NUMBER: _ClassVar[int]
    migrations: _containers.RepeatedCompositeFieldContainer[Migration]
    def __init__(self, migrations: _Optional[_Iterable[_Union[Migration, _Mapping]]] = ...) -> None: ...

class GetMigrationStatusRequest(_message.Message):
    __slots__ = ("migration_id",)
    MIGRATION_ID_FIELD_NUMBER: _ClassVar[int]
    migration_id: str
    def __init__(self, migration_id: _Optional[str] = ...) -> None: ...

class GetMigrationStatusResponse(_message.Message):
    __slots__ = ("status",)
    STATUS_FIELD_NUMBER: _ClassVar[int]
    status: MigrationStatus
    def __init__(self, status: _Optional[_Union[MigrationStatus, _Mapping]] = ...) -> None: ...

class NextMigrationStepRequest(_message.Message):
    __slots__ = ("migration_id",)
    MIGRATION_ID_FIELD_NUMBER: _ClassVar[int]
    migration_id: str
    def __init__(self, migration_id: _Optional[str] = ...) -> None: ...

class NextMigrationStepResponse(_message.Message):
    __slots__ = ("findings",)
    FINDINGS_FIELD_NUMBER: _ClassVar[int]
    findings: _containers.RepeatedCompositeFieldContainer[TrackedFinding]
    def __init__(self, findings: _Optional[_Iterable[_Union[TrackedFinding, _Mapping]]] = ...) -> None: ...

class ResolveFindingRequest(_message.Message):
    __slots__ = ("migration_id", "stable_id", "note")
    MIGRATION_ID_FIELD_NUMBER: _ClassVar[int]
    STABLE_ID_FIELD_NUMBER: _ClassVar[int]
    NOTE_FIELD_NUMBER: _ClassVar[int]
    migration_id: str
    stable_id: str
    note: str
    def __init__(self, migration_id: _Optional[str] = ..., stable_id: _Optional[str] = ..., note: _Optional[str] = ...) -> None: ...

class ResolveFindingResponse(_message.Message):
    __slots__ = ("finding",)
    FINDING_FIELD_NUMBER: _ClassVar[int]
    finding: TrackedFinding
    def __init__(self, finding: _Optional[_Union[TrackedFinding, _Mapping]] = ...) -> None: ...

class ApplyFindingRequest(_message.Message):
    __slots__ = ("migration_id", "stable_id")
    MIGRATION_ID_FIELD_NUMBER: _ClassVar[int]
    STABLE_ID_FIELD_NUMBER: _ClassVar[int]
    migration_id: str
    stable_id: str
    def __init__(self, migration_id: _Optional[str] = ..., stable_id: _Optional[str] = ...) -> None: ...

class ApplyFindingResponse(_message.Message):
    __slots__ = ("finding",)
    FINDING_FIELD_NUMBER: _ClassVar[int]
    finding: TrackedFinding
    def __init__(self, finding: _Optional[_Union[TrackedFinding, _Mapping]] = ...) -> None: ...

class ReauditMigrationRequest(_message.Message):
    __slots__ = ("migration_id", "findings")
    MIGRATION_ID_FIELD_NUMBER: _ClassVar[int]
    FINDINGS_FIELD_NUMBER: _ClassVar[int]
    migration_id: str
    findings: _containers.RepeatedCompositeFieldContainer[_findings_pb2.ArchitectureFinding]
    def __init__(self, migration_id: _Optional[str] = ..., findings: _Optional[_Iterable[_Union[_findings_pb2.ArchitectureFinding, _Mapping]]] = ...) -> None: ...

class ReauditMigrationResponse(_message.Message):
    __slots__ = ("validated", "still_open", "regressions", "status")
    VALIDATED_FIELD_NUMBER: _ClassVar[int]
    STILL_OPEN_FIELD_NUMBER: _ClassVar[int]
    REGRESSIONS_FIELD_NUMBER: _ClassVar[int]
    STATUS_FIELD_NUMBER: _ClassVar[int]
    validated: _containers.RepeatedCompositeFieldContainer[TrackedFinding]
    still_open: _containers.RepeatedCompositeFieldContainer[TrackedFinding]
    regressions: _containers.RepeatedCompositeFieldContainer[TrackedFinding]
    status: MigrationStatus
    def __init__(self, validated: _Optional[_Iterable[_Union[TrackedFinding, _Mapping]]] = ..., still_open: _Optional[_Iterable[_Union[TrackedFinding, _Mapping]]] = ..., regressions: _Optional[_Iterable[_Union[TrackedFinding, _Mapping]]] = ..., status: _Optional[_Union[MigrationStatus, _Mapping]] = ...) -> None: ...

class CloseMigrationRequest(_message.Message):
    __slots__ = ("migration_id",)
    MIGRATION_ID_FIELD_NUMBER: _ClassVar[int]
    migration_id: str
    def __init__(self, migration_id: _Optional[str] = ...) -> None: ...

class CloseMigrationResponse(_message.Message):
    __slots__ = ("status",)
    STATUS_FIELD_NUMBER: _ClassVar[int]
    status: MigrationStatus
    def __init__(self, status: _Optional[_Union[MigrationStatus, _Mapping]] = ...) -> None: ...
