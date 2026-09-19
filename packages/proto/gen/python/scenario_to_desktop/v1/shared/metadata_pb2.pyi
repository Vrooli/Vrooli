from google.protobuf.internal import containers as _containers
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Iterable as _Iterable
from typing import ClassVar as _ClassVar, Optional as _Optional

DESCRIPTOR: _descriptor.FileDescriptor

class ScenarioMetadata(_message.Message):
    __slots__ = ("name", "display_name", "description", "version", "author", "license", "app_id", "has_ui", "ui_dist_path", "ui_port", "api_port", "scenario_path", "category", "tags", "service_json_path", "package_json_path")
    NAME_FIELD_NUMBER: _ClassVar[int]
    DISPLAY_NAME_FIELD_NUMBER: _ClassVar[int]
    DESCRIPTION_FIELD_NUMBER: _ClassVar[int]
    VERSION_FIELD_NUMBER: _ClassVar[int]
    AUTHOR_FIELD_NUMBER: _ClassVar[int]
    LICENSE_FIELD_NUMBER: _ClassVar[int]
    APP_ID_FIELD_NUMBER: _ClassVar[int]
    HAS_UI_FIELD_NUMBER: _ClassVar[int]
    UI_DIST_PATH_FIELD_NUMBER: _ClassVar[int]
    UI_PORT_FIELD_NUMBER: _ClassVar[int]
    API_PORT_FIELD_NUMBER: _ClassVar[int]
    SCENARIO_PATH_FIELD_NUMBER: _ClassVar[int]
    CATEGORY_FIELD_NUMBER: _ClassVar[int]
    TAGS_FIELD_NUMBER: _ClassVar[int]
    SERVICE_JSON_PATH_FIELD_NUMBER: _ClassVar[int]
    PACKAGE_JSON_PATH_FIELD_NUMBER: _ClassVar[int]
    name: str
    display_name: str
    description: str
    version: str
    author: str
    license: str
    app_id: str
    has_ui: bool
    ui_dist_path: str
    ui_port: int
    api_port: int
    scenario_path: str
    category: str
    tags: _containers.RepeatedScalarFieldContainer[str]
    service_json_path: str
    package_json_path: str
    def __init__(self, name: _Optional[str] = ..., display_name: _Optional[str] = ..., description: _Optional[str] = ..., version: _Optional[str] = ..., author: _Optional[str] = ..., license: _Optional[str] = ..., app_id: _Optional[str] = ..., has_ui: _Optional[bool] = ..., ui_dist_path: _Optional[str] = ..., ui_port: _Optional[int] = ..., api_port: _Optional[int] = ..., scenario_path: _Optional[str] = ..., category: _Optional[str] = ..., tags: _Optional[_Iterable[str]] = ..., service_json_path: _Optional[str] = ..., package_json_path: _Optional[str] = ...) -> None: ...
