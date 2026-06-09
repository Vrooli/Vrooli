from buf.validate import validate_pb2 as _validate_pb2
from google.protobuf.internal import containers as _containers
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Iterable as _Iterable, Mapping as _Mapping
from typing import ClassVar as _ClassVar, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class Resource(_message.Message):
    __slots__ = ("name", "display_name", "path", "port", "category", "description", "healthy", "version", "status")
    NAME_FIELD_NUMBER: _ClassVar[int]
    DISPLAY_NAME_FIELD_NUMBER: _ClassVar[int]
    PATH_FIELD_NUMBER: _ClassVar[int]
    PORT_FIELD_NUMBER: _ClassVar[int]
    CATEGORY_FIELD_NUMBER: _ClassVar[int]
    DESCRIPTION_FIELD_NUMBER: _ClassVar[int]
    HEALTHY_FIELD_NUMBER: _ClassVar[int]
    VERSION_FIELD_NUMBER: _ClassVar[int]
    STATUS_FIELD_NUMBER: _ClassVar[int]
    name: str
    display_name: str
    path: str
    port: int
    category: str
    description: str
    healthy: bool
    version: str
    status: str
    def __init__(self, name: _Optional[str] = ..., display_name: _Optional[str] = ..., path: _Optional[str] = ..., port: _Optional[int] = ..., category: _Optional[str] = ..., description: _Optional[str] = ..., healthy: _Optional[bool] = ..., version: _Optional[str] = ..., status: _Optional[str] = ...) -> None: ...

class Scenario(_message.Message):
    __slots__ = ("name", "display_name", "path", "category", "description", "version", "status")
    NAME_FIELD_NUMBER: _ClassVar[int]
    DISPLAY_NAME_FIELD_NUMBER: _ClassVar[int]
    PATH_FIELD_NUMBER: _ClassVar[int]
    CATEGORY_FIELD_NUMBER: _ClassVar[int]
    DESCRIPTION_FIELD_NUMBER: _ClassVar[int]
    VERSION_FIELD_NUMBER: _ClassVar[int]
    STATUS_FIELD_NUMBER: _ClassVar[int]
    name: str
    display_name: str
    path: str
    category: str
    description: str
    version: str
    status: str
    def __init__(self, name: _Optional[str] = ..., display_name: _Optional[str] = ..., path: _Optional[str] = ..., category: _Optional[str] = ..., description: _Optional[str] = ..., version: _Optional[str] = ..., status: _Optional[str] = ...) -> None: ...

class Operation(_message.Message):
    __slots__ = ("name", "description")
    NAME_FIELD_NUMBER: _ClassVar[int]
    DESCRIPTION_FIELD_NUMBER: _ClassVar[int]
    name: str
    description: str
    def __init__(self, name: _Optional[str] = ..., description: _Optional[str] = ...) -> None: ...

class ListResourcesRequest(_message.Message):
    __slots__ = ("refresh",)
    REFRESH_FIELD_NUMBER: _ClassVar[int]
    refresh: bool
    def __init__(self, refresh: _Optional[bool] = ...) -> None: ...

class ListResourcesResponse(_message.Message):
    __slots__ = ("resources", "count")
    RESOURCES_FIELD_NUMBER: _ClassVar[int]
    COUNT_FIELD_NUMBER: _ClassVar[int]
    resources: _containers.RepeatedCompositeFieldContainer[Resource]
    count: int
    def __init__(self, resources: _Optional[_Iterable[_Union[Resource, _Mapping]]] = ..., count: _Optional[int] = ...) -> None: ...

class ListScenariosRequest(_message.Message):
    __slots__ = ("refresh",)
    REFRESH_FIELD_NUMBER: _ClassVar[int]
    refresh: bool
    def __init__(self, refresh: _Optional[bool] = ...) -> None: ...

class ListScenariosResponse(_message.Message):
    __slots__ = ("scenarios", "count")
    SCENARIOS_FIELD_NUMBER: _ClassVar[int]
    COUNT_FIELD_NUMBER: _ClassVar[int]
    scenarios: _containers.RepeatedCompositeFieldContainer[Scenario]
    count: int
    def __init__(self, scenarios: _Optional[_Iterable[_Union[Scenario, _Mapping]]] = ..., count: _Optional[int] = ...) -> None: ...

class GetResourceRequest(_message.Message):
    __slots__ = ("name", "refresh")
    NAME_FIELD_NUMBER: _ClassVar[int]
    REFRESH_FIELD_NUMBER: _ClassVar[int]
    name: str
    refresh: bool
    def __init__(self, name: _Optional[str] = ..., refresh: _Optional[bool] = ...) -> None: ...

class GetScenarioRequest(_message.Message):
    __slots__ = ("name", "refresh")
    NAME_FIELD_NUMBER: _ClassVar[int]
    REFRESH_FIELD_NUMBER: _ClassVar[int]
    name: str
    refresh: bool
    def __init__(self, name: _Optional[str] = ..., refresh: _Optional[bool] = ...) -> None: ...

class ListOperationsRequest(_message.Message):
    __slots__ = ()
    def __init__(self) -> None: ...

class ListOperationsResponse(_message.Message):
    __slots__ = ("operations",)
    OPERATIONS_FIELD_NUMBER: _ClassVar[int]
    operations: _containers.RepeatedCompositeFieldContainer[Operation]
    def __init__(self, operations: _Optional[_Iterable[_Union[Operation, _Mapping]]] = ...) -> None: ...

class ListCategoriesRequest(_message.Message):
    __slots__ = ()
    def __init__(self) -> None: ...

class ListCategoriesResponse(_message.Message):
    __slots__ = ("resource_categories", "scenario_categories")
    class ResourceCategoriesEntry(_message.Message):
        __slots__ = ("key", "value")
        KEY_FIELD_NUMBER: _ClassVar[int]
        VALUE_FIELD_NUMBER: _ClassVar[int]
        key: str
        value: str
        def __init__(self, key: _Optional[str] = ..., value: _Optional[str] = ...) -> None: ...
    class ScenarioCategoriesEntry(_message.Message):
        __slots__ = ("key", "value")
        KEY_FIELD_NUMBER: _ClassVar[int]
        VALUE_FIELD_NUMBER: _ClassVar[int]
        key: str
        value: str
        def __init__(self, key: _Optional[str] = ..., value: _Optional[str] = ...) -> None: ...
    RESOURCE_CATEGORIES_FIELD_NUMBER: _ClassVar[int]
    SCENARIO_CATEGORIES_FIELD_NUMBER: _ClassVar[int]
    resource_categories: _containers.ScalarMap[str, str]
    scenario_categories: _containers.ScalarMap[str, str]
    def __init__(self, resource_categories: _Optional[_Mapping[str, str]] = ..., scenario_categories: _Optional[_Mapping[str, str]] = ...) -> None: ...
