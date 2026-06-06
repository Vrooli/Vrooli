from buf.validate import validate_pb2 as _validate_pb2
from google.protobuf.internal import containers as _containers
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Iterable as _Iterable, Mapping as _Mapping
from typing import ClassVar as _ClassVar, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class SafetyDestination(_message.Message):
    __slots__ = ("id", "name", "location", "repository_location")
    ID_FIELD_NUMBER: _ClassVar[int]
    NAME_FIELD_NUMBER: _ClassVar[int]
    LOCATION_FIELD_NUMBER: _ClassVar[int]
    REPOSITORY_LOCATION_FIELD_NUMBER: _ClassVar[int]
    id: str
    name: str
    location: str
    repository_location: str
    def __init__(self, id: _Optional[str] = ..., name: _Optional[str] = ..., location: _Optional[str] = ..., repository_location: _Optional[str] = ...) -> None: ...

class EnsureSafetyDestinationRequest(_message.Message):
    __slots__ = ("cap_bytes",)
    CAP_BYTES_FIELD_NUMBER: _ClassVar[int]
    cap_bytes: int
    def __init__(self, cap_bytes: _Optional[int] = ...) -> None: ...

class EnsureSafetyDestinationResponse(_message.Message):
    __slots__ = ("destination", "created")
    DESTINATION_FIELD_NUMBER: _ClassVar[int]
    CREATED_FIELD_NUMBER: _ClassVar[int]
    destination: SafetyDestination
    created: bool
    def __init__(self, destination: _Optional[_Union[SafetyDestination, _Mapping]] = ..., created: _Optional[bool] = ...) -> None: ...

class BackupScenarioNowRequest(_message.Message):
    __slots__ = ("scenario", "keep_latest")
    SCENARIO_FIELD_NUMBER: _ClassVar[int]
    KEEP_LATEST_FIELD_NUMBER: _ClassVar[int]
    scenario: str
    keep_latest: int
    def __init__(self, scenario: _Optional[str] = ..., keep_latest: _Optional[int] = ...) -> None: ...

class BackupScenarioNowResponse(_message.Message):
    __slots__ = ("run_id", "plan_id", "destination_id", "target_count", "status")
    RUN_ID_FIELD_NUMBER: _ClassVar[int]
    PLAN_ID_FIELD_NUMBER: _ClassVar[int]
    DESTINATION_ID_FIELD_NUMBER: _ClassVar[int]
    TARGET_COUNT_FIELD_NUMBER: _ClassVar[int]
    STATUS_FIELD_NUMBER: _ClassVar[int]
    run_id: str
    plan_id: str
    destination_id: str
    target_count: int
    status: str
    def __init__(self, run_id: _Optional[str] = ..., plan_id: _Optional[str] = ..., destination_id: _Optional[str] = ..., target_count: _Optional[int] = ..., status: _Optional[str] = ...) -> None: ...

class RegisterScenarioTargetsRequest(_message.Message):
    __slots__ = ("scenario",)
    SCENARIO_FIELD_NUMBER: _ClassVar[int]
    scenario: str
    def __init__(self, scenario: _Optional[str] = ...) -> None: ...

class RegisteredTarget(_message.Message):
    __slots__ = ("name", "source_kind", "locator")
    NAME_FIELD_NUMBER: _ClassVar[int]
    SOURCE_KIND_FIELD_NUMBER: _ClassVar[int]
    LOCATOR_FIELD_NUMBER: _ClassVar[int]
    name: str
    source_kind: str
    locator: str
    def __init__(self, name: _Optional[str] = ..., source_kind: _Optional[str] = ..., locator: _Optional[str] = ...) -> None: ...

class SkippedTarget(_message.Message):
    __slots__ = ("source_kind", "reason")
    SOURCE_KIND_FIELD_NUMBER: _ClassVar[int]
    REASON_FIELD_NUMBER: _ClassVar[int]
    source_kind: str
    reason: str
    def __init__(self, source_kind: _Optional[str] = ..., reason: _Optional[str] = ...) -> None: ...

class RegisterScenarioTargetsResponse(_message.Message):
    __slots__ = ("scenario", "registered", "skipped")
    SCENARIO_FIELD_NUMBER: _ClassVar[int]
    REGISTERED_FIELD_NUMBER: _ClassVar[int]
    SKIPPED_FIELD_NUMBER: _ClassVar[int]
    scenario: str
    registered: _containers.RepeatedCompositeFieldContainer[RegisteredTarget]
    skipped: _containers.RepeatedCompositeFieldContainer[SkippedTarget]
    def __init__(self, scenario: _Optional[str] = ..., registered: _Optional[_Iterable[_Union[RegisteredTarget, _Mapping]]] = ..., skipped: _Optional[_Iterable[_Union[SkippedTarget, _Mapping]]] = ...) -> None: ...
