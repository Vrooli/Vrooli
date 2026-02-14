from ecosystem_manager.v1.domain import discovery_pb2 as _discovery_pb2
from google.protobuf.internal import containers as _containers
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Iterable as _Iterable, Mapping as _Mapping
from typing import ClassVar as _ClassVar, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class ListResourcesResponse(_message.Message):
    __slots__ = ("resources", "count")
    RESOURCES_FIELD_NUMBER: _ClassVar[int]
    COUNT_FIELD_NUMBER: _ClassVar[int]
    resources: _containers.RepeatedCompositeFieldContainer[_discovery_pb2.Resource]
    count: int
    def __init__(self, resources: _Optional[_Iterable[_Union[_discovery_pb2.Resource, _Mapping]]] = ..., count: _Optional[int] = ...) -> None: ...

class ListScenariosResponse(_message.Message):
    __slots__ = ("scenarios", "count")
    SCENARIOS_FIELD_NUMBER: _ClassVar[int]
    COUNT_FIELD_NUMBER: _ClassVar[int]
    scenarios: _containers.RepeatedCompositeFieldContainer[_discovery_pb2.Scenario]
    count: int
    def __init__(self, scenarios: _Optional[_Iterable[_Union[_discovery_pb2.Scenario, _Mapping]]] = ..., count: _Optional[int] = ...) -> None: ...

class ListOperationsResponse(_message.Message):
    __slots__ = ("operations",)
    OPERATIONS_FIELD_NUMBER: _ClassVar[int]
    operations: _containers.RepeatedCompositeFieldContainer[_discovery_pb2.Operation]
    def __init__(self, operations: _Optional[_Iterable[_Union[_discovery_pb2.Operation, _Mapping]]] = ...) -> None: ...

class ListCategoriesResponse(_message.Message):
    __slots__ = ("categories",)
    CATEGORIES_FIELD_NUMBER: _ClassVar[int]
    categories: _containers.RepeatedCompositeFieldContainer[_discovery_pb2.Category]
    def __init__(self, categories: _Optional[_Iterable[_Union[_discovery_pb2.Category, _Mapping]]] = ...) -> None: ...
