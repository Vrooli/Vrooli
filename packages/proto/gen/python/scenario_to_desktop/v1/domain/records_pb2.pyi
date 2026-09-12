import datetime

from buf.validate import validate_pb2 as _validate_pb2
from google.protobuf import empty_pb2 as _empty_pb2
from google.protobuf import timestamp_pb2 as _timestamp_pb2
from google.protobuf.internal import containers as _containers
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Iterable as _Iterable, Mapping as _Mapping
from typing import ClassVar as _ClassVar, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class DesktopRecordRequest(_message.Message):
    __slots__ = ("record_id",)
    RECORD_ID_FIELD_NUMBER: _ClassVar[int]
    record_id: str
    def __init__(self, record_id: _Optional[str] = ...) -> None: ...

class MoveDesktopRecordRequest(_message.Message):
    __slots__ = ("record_id", "target", "destination_path")
    RECORD_ID_FIELD_NUMBER: _ClassVar[int]
    TARGET_FIELD_NUMBER: _ClassVar[int]
    DESTINATION_PATH_FIELD_NUMBER: _ClassVar[int]
    record_id: str
    target: str
    destination_path: str
    def __init__(self, record_id: _Optional[str] = ..., target: _Optional[str] = ..., destination_path: _Optional[str] = ...) -> None: ...

class DeleteDesktopScenarioRequest(_message.Message):
    __slots__ = ("scenario_name",)
    SCENARIO_NAME_FIELD_NUMBER: _ClassVar[int]
    scenario_name: str
    def __init__(self, scenario_name: _Optional[str] = ...) -> None: ...

class DesktopRecord(_message.Message):
    __slots__ = ("id", "build_id", "scenario_name", "app_display_name", "template_type", "framework", "location_mode", "output_path", "destination_path", "staging_path", "custom_path", "deployment_mode", "icon", "created_at", "updated_at")
    ID_FIELD_NUMBER: _ClassVar[int]
    BUILD_ID_FIELD_NUMBER: _ClassVar[int]
    SCENARIO_NAME_FIELD_NUMBER: _ClassVar[int]
    APP_DISPLAY_NAME_FIELD_NUMBER: _ClassVar[int]
    TEMPLATE_TYPE_FIELD_NUMBER: _ClassVar[int]
    FRAMEWORK_FIELD_NUMBER: _ClassVar[int]
    LOCATION_MODE_FIELD_NUMBER: _ClassVar[int]
    OUTPUT_PATH_FIELD_NUMBER: _ClassVar[int]
    DESTINATION_PATH_FIELD_NUMBER: _ClassVar[int]
    STAGING_PATH_FIELD_NUMBER: _ClassVar[int]
    CUSTOM_PATH_FIELD_NUMBER: _ClassVar[int]
    DEPLOYMENT_MODE_FIELD_NUMBER: _ClassVar[int]
    ICON_FIELD_NUMBER: _ClassVar[int]
    CREATED_AT_FIELD_NUMBER: _ClassVar[int]
    UPDATED_AT_FIELD_NUMBER: _ClassVar[int]
    id: str
    build_id: str
    scenario_name: str
    app_display_name: str
    template_type: str
    framework: str
    location_mode: str
    output_path: str
    destination_path: str
    staging_path: str
    custom_path: str
    deployment_mode: str
    icon: str
    created_at: _timestamp_pb2.Timestamp
    updated_at: _timestamp_pb2.Timestamp
    def __init__(self, id: _Optional[str] = ..., build_id: _Optional[str] = ..., scenario_name: _Optional[str] = ..., app_display_name: _Optional[str] = ..., template_type: _Optional[str] = ..., framework: _Optional[str] = ..., location_mode: _Optional[str] = ..., output_path: _Optional[str] = ..., destination_path: _Optional[str] = ..., staging_path: _Optional[str] = ..., custom_path: _Optional[str] = ..., deployment_mode: _Optional[str] = ..., icon: _Optional[str] = ..., created_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., updated_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ...) -> None: ...

class DesktopBuildSummary(_message.Message):
    __slots__ = ("status", "output_path", "metadata")
    STATUS_FIELD_NUMBER: _ClassVar[int]
    OUTPUT_PATH_FIELD_NUMBER: _ClassVar[int]
    METADATA_FIELD_NUMBER: _ClassVar[int]
    status: str
    output_path: str
    metadata: BuildMetadata
    def __init__(self, status: _Optional[str] = ..., output_path: _Optional[str] = ..., metadata: _Optional[_Union[BuildMetadata, _Mapping]] = ...) -> None: ...

class BuildMetadata(_message.Message):
    __slots__ = ("version", "git_branch", "git_commit_hash", "git_dirty")
    VERSION_FIELD_NUMBER: _ClassVar[int]
    GIT_BRANCH_FIELD_NUMBER: _ClassVar[int]
    GIT_COMMIT_HASH_FIELD_NUMBER: _ClassVar[int]
    GIT_DIRTY_FIELD_NUMBER: _ClassVar[int]
    version: str
    git_branch: str
    git_commit_hash: str
    git_dirty: bool
    def __init__(self, version: _Optional[str] = ..., git_branch: _Optional[str] = ..., git_commit_hash: _Optional[str] = ..., git_dirty: _Optional[bool] = ...) -> None: ...

class ScreenRecordingSummary(_message.Message):
    __slots__ = ("recorded", "duration_ms", "file_size_bytes", "error", "capture_id")
    RECORDED_FIELD_NUMBER: _ClassVar[int]
    DURATION_MS_FIELD_NUMBER: _ClassVar[int]
    FILE_SIZE_BYTES_FIELD_NUMBER: _ClassVar[int]
    ERROR_FIELD_NUMBER: _ClassVar[int]
    CAPTURE_ID_FIELD_NUMBER: _ClassVar[int]
    recorded: bool
    duration_ms: int
    file_size_bytes: int
    error: str
    capture_id: str
    def __init__(self, recorded: _Optional[bool] = ..., duration_ms: _Optional[int] = ..., file_size_bytes: _Optional[int] = ..., error: _Optional[str] = ..., capture_id: _Optional[str] = ...) -> None: ...

class DesktopRecordWithBuild(_message.Message):
    __slots__ = ("record", "build_status", "has_build", "build_state", "smoke_test_id", "screen_recording")
    RECORD_FIELD_NUMBER: _ClassVar[int]
    BUILD_STATUS_FIELD_NUMBER: _ClassVar[int]
    HAS_BUILD_FIELD_NUMBER: _ClassVar[int]
    BUILD_STATE_FIELD_NUMBER: _ClassVar[int]
    SMOKE_TEST_ID_FIELD_NUMBER: _ClassVar[int]
    SCREEN_RECORDING_FIELD_NUMBER: _ClassVar[int]
    record: DesktopRecord
    build_status: DesktopBuildSummary
    has_build: bool
    build_state: str
    smoke_test_id: str
    screen_recording: ScreenRecordingSummary
    def __init__(self, record: _Optional[_Union[DesktopRecord, _Mapping]] = ..., build_status: _Optional[_Union[DesktopBuildSummary, _Mapping]] = ..., has_build: _Optional[bool] = ..., build_state: _Optional[str] = ..., smoke_test_id: _Optional[str] = ..., screen_recording: _Optional[_Union[ScreenRecordingSummary, _Mapping]] = ...) -> None: ...

class DesktopRecordsResponse(_message.Message):
    __slots__ = ("records",)
    RECORDS_FIELD_NUMBER: _ClassVar[int]
    records: _containers.RepeatedCompositeFieldContainer[DesktopRecordWithBuild]
    def __init__(self, records: _Optional[_Iterable[_Union[DesktopRecordWithBuild, _Mapping]]] = ...) -> None: ...

class MoveDesktopRecordResponse(_message.Message):
    __slots__ = ("record_id", "to", "status")
    RECORD_ID_FIELD_NUMBER: _ClassVar[int]
    FROM_FIELD_NUMBER: _ClassVar[int]
    TO_FIELD_NUMBER: _ClassVar[int]
    STATUS_FIELD_NUMBER: _ClassVar[int]
    record_id: str
    to: str
    status: str
    def __init__(self, record_id: _Optional[str] = ..., to: _Optional[str] = ..., status: _Optional[str] = ..., **kwargs) -> None: ...

class DeleteDesktopScenarioResponse(_message.Message):
    __slots__ = ("status", "scenario_name", "deleted_path", "removed_records", "message")
    STATUS_FIELD_NUMBER: _ClassVar[int]
    SCENARIO_NAME_FIELD_NUMBER: _ClassVar[int]
    DELETED_PATH_FIELD_NUMBER: _ClassVar[int]
    REMOVED_RECORDS_FIELD_NUMBER: _ClassVar[int]
    MESSAGE_FIELD_NUMBER: _ClassVar[int]
    status: str
    scenario_name: str
    deleted_path: str
    removed_records: int
    message: str
    def __init__(self, status: _Optional[str] = ..., scenario_name: _Optional[str] = ..., deleted_path: _Optional[str] = ..., removed_records: _Optional[int] = ..., message: _Optional[str] = ...) -> None: ...
