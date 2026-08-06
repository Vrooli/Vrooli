import datetime

from buf.validate import validate_pb2 as _validate_pb2
from google.protobuf import timestamp_pb2 as _timestamp_pb2
from scenario_to_desktop.v1.shared import common_pb2 as _common_pb2
from scenario_to_desktop.v1.shared import operation_results_pb2 as _operation_results_pb2
from google.protobuf.internal import containers as _containers
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Iterable as _Iterable, Mapping as _Mapping
from typing import ClassVar as _ClassVar, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class BuildRequest(_message.Message):
    __slots__ = ("desktop_path", "platforms", "sign", "publish")
    DESKTOP_PATH_FIELD_NUMBER: _ClassVar[int]
    PLATFORMS_FIELD_NUMBER: _ClassVar[int]
    SIGN_FIELD_NUMBER: _ClassVar[int]
    PUBLISH_FIELD_NUMBER: _ClassVar[int]
    desktop_path: str
    platforms: _containers.RepeatedScalarFieldContainer[_common_pb2.Platform]
    sign: bool
    publish: bool
    def __init__(self, desktop_path: _Optional[str] = ..., platforms: _Optional[_Iterable[_Union[_common_pb2.Platform, str]]] = ..., sign: _Optional[bool] = ..., publish: _Optional[bool] = ...) -> None: ...

class ScenarioBuildRequest(_message.Message):
    __slots__ = ("scenario_name", "desktop_path", "platforms", "clean")
    SCENARIO_NAME_FIELD_NUMBER: _ClassVar[int]
    DESKTOP_PATH_FIELD_NUMBER: _ClassVar[int]
    PLATFORMS_FIELD_NUMBER: _ClassVar[int]
    CLEAN_FIELD_NUMBER: _ClassVar[int]
    scenario_name: str
    desktop_path: str
    platforms: _containers.RepeatedScalarFieldContainer[_common_pb2.Platform]
    clean: bool
    def __init__(self, scenario_name: _Optional[str] = ..., desktop_path: _Optional[str] = ..., platforms: _Optional[_Iterable[_Union[_common_pb2.Platform, str]]] = ..., clean: _Optional[bool] = ...) -> None: ...

class BuildResponse(_message.Message):
    __slots__ = ("build_id", "status", "status_url")
    BUILD_ID_FIELD_NUMBER: _ClassVar[int]
    STATUS_FIELD_NUMBER: _ClassVar[int]
    STATUS_URL_FIELD_NUMBER: _ClassVar[int]
    build_id: str
    status: str
    status_url: str
    def __init__(self, build_id: _Optional[str] = ..., status: _Optional[str] = ..., status_url: _Optional[str] = ...) -> None: ...

class BuildStatusRequest(_message.Message):
    __slots__ = ("build_id",)
    BUILD_ID_FIELD_NUMBER: _ClassVar[int]
    build_id: str
    def __init__(self, build_id: _Optional[str] = ...) -> None: ...

class SmokeTestStartRequest(_message.Message):
    __slots__ = ("scenario_name", "platform", "artifact_path", "record_desktop")
    SCENARIO_NAME_FIELD_NUMBER: _ClassVar[int]
    PLATFORM_FIELD_NUMBER: _ClassVar[int]
    ARTIFACT_PATH_FIELD_NUMBER: _ClassVar[int]
    RECORD_DESKTOP_FIELD_NUMBER: _ClassVar[int]
    scenario_name: str
    platform: _common_pb2.Platform
    artifact_path: str
    record_desktop: bool
    def __init__(self, scenario_name: _Optional[str] = ..., platform: _Optional[_Union[_common_pb2.Platform, str]] = ..., artifact_path: _Optional[str] = ..., record_desktop: _Optional[bool] = ...) -> None: ...

class SmokeTestStartResponse(_message.Message):
    __slots__ = ("smoke_test_id", "scenario_name", "platform", "status", "artifact_path", "started_at", "logs")
    SMOKE_TEST_ID_FIELD_NUMBER: _ClassVar[int]
    SCENARIO_NAME_FIELD_NUMBER: _ClassVar[int]
    PLATFORM_FIELD_NUMBER: _ClassVar[int]
    STATUS_FIELD_NUMBER: _ClassVar[int]
    ARTIFACT_PATH_FIELD_NUMBER: _ClassVar[int]
    STARTED_AT_FIELD_NUMBER: _ClassVar[int]
    LOGS_FIELD_NUMBER: _ClassVar[int]
    smoke_test_id: str
    scenario_name: str
    platform: _common_pb2.Platform
    status: _operation_results_pb2.SmokeTestStatus
    artifact_path: str
    started_at: _timestamp_pb2.Timestamp
    logs: _containers.RepeatedScalarFieldContainer[str]
    def __init__(self, smoke_test_id: _Optional[str] = ..., scenario_name: _Optional[str] = ..., platform: _Optional[_Union[_common_pb2.Platform, str]] = ..., status: _Optional[_Union[_operation_results_pb2.SmokeTestStatus, str]] = ..., artifact_path: _Optional[str] = ..., started_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., logs: _Optional[_Iterable[str]] = ...) -> None: ...

class SmokeTestCancelResponse(_message.Message):
    __slots__ = ("status",)
    STATUS_FIELD_NUMBER: _ClassVar[int]
    status: str
    def __init__(self, status: _Optional[str] = ...) -> None: ...

class SmokeTestStatusRequest(_message.Message):
    __slots__ = ("smoke_test_id",)
    SMOKE_TEST_ID_FIELD_NUMBER: _ClassVar[int]
    smoke_test_id: str
    def __init__(self, smoke_test_id: _Optional[str] = ...) -> None: ...

class SmokeTestCancelRequest(_message.Message):
    __slots__ = ("smoke_test_id",)
    SMOKE_TEST_ID_FIELD_NUMBER: _ClassVar[int]
    smoke_test_id: str
    def __init__(self, smoke_test_id: _Optional[str] = ...) -> None: ...
