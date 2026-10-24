from google.protobuf.internal import containers as _containers
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Iterable as _Iterable, Mapping as _Mapping
from typing import ClassVar as _ClassVar, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class SharedDriftReport(_message.Message):
    __slots__ = ("clean", "root", "scenarios", "touched_packages", "only_touched", "build_checked", "fix_applied", "modified_tracked_files", "elapsed_ms")
    CLEAN_FIELD_NUMBER: _ClassVar[int]
    ROOT_FIELD_NUMBER: _ClassVar[int]
    SCENARIOS_FIELD_NUMBER: _ClassVar[int]
    TOUCHED_PACKAGES_FIELD_NUMBER: _ClassVar[int]
    ONLY_TOUCHED_FIELD_NUMBER: _ClassVar[int]
    BUILD_CHECKED_FIELD_NUMBER: _ClassVar[int]
    FIX_APPLIED_FIELD_NUMBER: _ClassVar[int]
    MODIFIED_TRACKED_FILES_FIELD_NUMBER: _ClassVar[int]
    ELAPSED_MS_FIELD_NUMBER: _ClassVar[int]
    clean: bool
    root: str
    scenarios: _containers.RepeatedCompositeFieldContainer[SharedDriftScenario]
    touched_packages: _containers.RepeatedScalarFieldContainer[str]
    only_touched: bool
    build_checked: bool
    fix_applied: bool
    modified_tracked_files: bool
    elapsed_ms: int
    def __init__(self, clean: _Optional[bool] = ..., root: _Optional[str] = ..., scenarios: _Optional[_Iterable[_Union[SharedDriftScenario, _Mapping]]] = ..., touched_packages: _Optional[_Iterable[str]] = ..., only_touched: _Optional[bool] = ..., build_checked: _Optional[bool] = ..., fix_applied: _Optional[bool] = ..., modified_tracked_files: _Optional[bool] = ..., elapsed_ms: _Optional[int] = ...) -> None: ...

class SharedDriftScenario(_message.Message):
    __slots__ = ("path", "api_dir", "status", "diff_paths", "build_error", "error", "replaces")
    PATH_FIELD_NUMBER: _ClassVar[int]
    API_DIR_FIELD_NUMBER: _ClassVar[int]
    STATUS_FIELD_NUMBER: _ClassVar[int]
    DIFF_PATHS_FIELD_NUMBER: _ClassVar[int]
    BUILD_ERROR_FIELD_NUMBER: _ClassVar[int]
    ERROR_FIELD_NUMBER: _ClassVar[int]
    REPLACES_FIELD_NUMBER: _ClassVar[int]
    path: str
    api_dir: str
    status: str
    diff_paths: _containers.RepeatedScalarFieldContainer[str]
    build_error: str
    error: str
    replaces: _containers.RepeatedScalarFieldContainer[str]
    def __init__(self, path: _Optional[str] = ..., api_dir: _Optional[str] = ..., status: _Optional[str] = ..., diff_paths: _Optional[_Iterable[str]] = ..., build_error: _Optional[str] = ..., error: _Optional[str] = ..., replaces: _Optional[_Iterable[str]] = ...) -> None: ...
