from google.protobuf.internal import containers as _containers
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Iterable as _Iterable, Mapping as _Mapping
from typing import ClassVar as _ClassVar, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class ListResourcesRequest(_message.Message):
    __slots__ = ("target",)
    TARGET_FIELD_NUMBER: _ClassVar[int]
    target: str
    def __init__(self, target: _Optional[str] = ...) -> None: ...

class GetResourceRequest(_message.Message):
    __slots__ = ("target", "name")
    TARGET_FIELD_NUMBER: _ClassVar[int]
    NAME_FIELD_NUMBER: _ClassVar[int]
    target: str
    name: str
    def __init__(self, target: _Optional[str] = ..., name: _Optional[str] = ...) -> None: ...

class GetResourceHealthRequest(_message.Message):
    __slots__ = ("target",)
    TARGET_FIELD_NUMBER: _ClassVar[int]
    target: str
    def __init__(self, target: _Optional[str] = ...) -> None: ...

class ListDerivedResourcesRequest(_message.Message):
    __slots__ = ("target",)
    TARGET_FIELD_NUMBER: _ClassVar[int]
    target: str
    def __init__(self, target: _Optional[str] = ...) -> None: ...

class Resource(_message.Message):
    __slots__ = ("name", "status", "category", "installed", "display_name", "description", "enabled")
    NAME_FIELD_NUMBER: _ClassVar[int]
    STATUS_FIELD_NUMBER: _ClassVar[int]
    CATEGORY_FIELD_NUMBER: _ClassVar[int]
    INSTALLED_FIELD_NUMBER: _ClassVar[int]
    DISPLAY_NAME_FIELD_NUMBER: _ClassVar[int]
    DESCRIPTION_FIELD_NUMBER: _ClassVar[int]
    ENABLED_FIELD_NUMBER: _ClassVar[int]
    name: str
    status: str
    category: str
    installed: bool
    display_name: str
    description: str
    enabled: bool
    def __init__(self, name: _Optional[str] = ..., status: _Optional[str] = ..., category: _Optional[str] = ..., installed: _Optional[bool] = ..., display_name: _Optional[str] = ..., description: _Optional[str] = ..., enabled: _Optional[bool] = ...) -> None: ...

class ListResourcesResponse(_message.Message):
    __slots__ = ("resources", "count", "loaded_at")
    RESOURCES_FIELD_NUMBER: _ClassVar[int]
    COUNT_FIELD_NUMBER: _ClassVar[int]
    LOADED_AT_FIELD_NUMBER: _ClassVar[int]
    resources: _containers.RepeatedCompositeFieldContainer[Resource]
    count: int
    loaded_at: str
    def __init__(self, resources: _Optional[_Iterable[_Union[Resource, _Mapping]]] = ..., count: _Optional[int] = ..., loaded_at: _Optional[str] = ...) -> None: ...

class GetResourceResponse(_message.Message):
    __slots__ = ("resource",)
    RESOURCE_FIELD_NUMBER: _ClassVar[int]
    resource: Resource
    def __init__(self, resource: _Optional[_Union[Resource, _Mapping]] = ...) -> None: ...

class ResourceHealth(_message.Message):
    __slots__ = ("name", "status", "category", "available", "last_checked")
    NAME_FIELD_NUMBER: _ClassVar[int]
    STATUS_FIELD_NUMBER: _ClassVar[int]
    CATEGORY_FIELD_NUMBER: _ClassVar[int]
    AVAILABLE_FIELD_NUMBER: _ClassVar[int]
    LAST_CHECKED_FIELD_NUMBER: _ClassVar[int]
    name: str
    status: str
    category: str
    available: bool
    last_checked: str
    def __init__(self, name: _Optional[str] = ..., status: _Optional[str] = ..., category: _Optional[str] = ..., available: _Optional[bool] = ..., last_checked: _Optional[str] = ...) -> None: ...

class GetResourceHealthResponse(_message.Message):
    __slots__ = ("resources", "total", "healthy_count", "checked_at")
    RESOURCES_FIELD_NUMBER: _ClassVar[int]
    TOTAL_FIELD_NUMBER: _ClassVar[int]
    HEALTHY_COUNT_FIELD_NUMBER: _ClassVar[int]
    CHECKED_AT_FIELD_NUMBER: _ClassVar[int]
    resources: _containers.RepeatedCompositeFieldContainer[ResourceHealth]
    total: int
    healthy_count: int
    checked_at: str
    def __init__(self, resources: _Optional[_Iterable[_Union[ResourceHealth, _Mapping]]] = ..., total: _Optional[int] = ..., healthy_count: _Optional[int] = ..., checked_at: _Optional[str] = ...) -> None: ...

class ListDerivedResourcesResponse(_message.Message):
    __slots__ = ("resources", "required", "optional", "standalone", "count")
    RESOURCES_FIELD_NUMBER: _ClassVar[int]
    REQUIRED_FIELD_NUMBER: _ClassVar[int]
    OPTIONAL_FIELD_NUMBER: _ClassVar[int]
    STANDALONE_FIELD_NUMBER: _ClassVar[int]
    COUNT_FIELD_NUMBER: _ClassVar[int]
    resources: _containers.RepeatedCompositeFieldContainer[Resource]
    required: _containers.RepeatedCompositeFieldContainer[Resource]
    optional: _containers.RepeatedCompositeFieldContainer[Resource]
    standalone: _containers.RepeatedCompositeFieldContainer[Resource]
    count: int
    def __init__(self, resources: _Optional[_Iterable[_Union[Resource, _Mapping]]] = ..., required: _Optional[_Iterable[_Union[Resource, _Mapping]]] = ..., optional: _Optional[_Iterable[_Union[Resource, _Mapping]]] = ..., standalone: _Optional[_Iterable[_Union[Resource, _Mapping]]] = ..., count: _Optional[int] = ...) -> None: ...
