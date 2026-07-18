import datetime

from google.protobuf import timestamp_pb2 as _timestamp_pb2
from google.protobuf.internal import containers as _containers
from google.protobuf.internal import enum_type_wrapper as _enum_type_wrapper
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Iterable as _Iterable, Mapping as _Mapping
from typing import ClassVar as _ClassVar, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class NodeStatus(int, metaclass=_enum_type_wrapper.EnumTypeWrapper):
    __slots__ = ()
    NODE_STATUS_UNSPECIFIED: _ClassVar[NodeStatus]
    NODE_STATUS_OFFLINE: _ClassVar[NodeStatus]
    NODE_STATUS_ONLINE: _ClassVar[NodeStatus]
    NODE_STATUS_NEEDS_UPDATE: _ClassVar[NodeStatus]
    NODE_STATUS_REVOKED: _ClassVar[NodeStatus]
NODE_STATUS_UNSPECIFIED: NodeStatus
NODE_STATUS_OFFLINE: NodeStatus
NODE_STATUS_ONLINE: NodeStatus
NODE_STATUS_NEEDS_UPDATE: NodeStatus
NODE_STATUS_REVOKED: NodeStatus

class Node(_message.Message):
    __slots__ = ("id", "name", "os", "arch", "revision", "endpoint", "capabilities", "scopes", "status", "online", "created_at", "updated_at", "last_seen_at", "revoked_at")
    ID_FIELD_NUMBER: _ClassVar[int]
    NAME_FIELD_NUMBER: _ClassVar[int]
    OS_FIELD_NUMBER: _ClassVar[int]
    ARCH_FIELD_NUMBER: _ClassVar[int]
    REVISION_FIELD_NUMBER: _ClassVar[int]
    ENDPOINT_FIELD_NUMBER: _ClassVar[int]
    CAPABILITIES_FIELD_NUMBER: _ClassVar[int]
    SCOPES_FIELD_NUMBER: _ClassVar[int]
    STATUS_FIELD_NUMBER: _ClassVar[int]
    ONLINE_FIELD_NUMBER: _ClassVar[int]
    CREATED_AT_FIELD_NUMBER: _ClassVar[int]
    UPDATED_AT_FIELD_NUMBER: _ClassVar[int]
    LAST_SEEN_AT_FIELD_NUMBER: _ClassVar[int]
    REVOKED_AT_FIELD_NUMBER: _ClassVar[int]
    id: str
    name: str
    os: str
    arch: str
    revision: str
    endpoint: str
    capabilities: _containers.RepeatedScalarFieldContainer[str]
    scopes: _containers.RepeatedScalarFieldContainer[str]
    status: NodeStatus
    online: bool
    created_at: _timestamp_pb2.Timestamp
    updated_at: _timestamp_pb2.Timestamp
    last_seen_at: _timestamp_pb2.Timestamp
    revoked_at: _timestamp_pb2.Timestamp
    def __init__(self, id: _Optional[str] = ..., name: _Optional[str] = ..., os: _Optional[str] = ..., arch: _Optional[str] = ..., revision: _Optional[str] = ..., endpoint: _Optional[str] = ..., capabilities: _Optional[_Iterable[str]] = ..., scopes: _Optional[_Iterable[str]] = ..., status: _Optional[_Union[NodeStatus, str]] = ..., online: _Optional[bool] = ..., created_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., updated_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., last_seen_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., revoked_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ...) -> None: ...

class RegisterNodeRequest(_message.Message):
    __slots__ = ("name", "os", "arch", "endpoint", "capabilities", "scopes")
    NAME_FIELD_NUMBER: _ClassVar[int]
    OS_FIELD_NUMBER: _ClassVar[int]
    ARCH_FIELD_NUMBER: _ClassVar[int]
    ENDPOINT_FIELD_NUMBER: _ClassVar[int]
    CAPABILITIES_FIELD_NUMBER: _ClassVar[int]
    SCOPES_FIELD_NUMBER: _ClassVar[int]
    name: str
    os: str
    arch: str
    endpoint: str
    capabilities: _containers.RepeatedScalarFieldContainer[str]
    scopes: _containers.RepeatedScalarFieldContainer[str]
    def __init__(self, name: _Optional[str] = ..., os: _Optional[str] = ..., arch: _Optional[str] = ..., endpoint: _Optional[str] = ..., capabilities: _Optional[_Iterable[str]] = ..., scopes: _Optional[_Iterable[str]] = ...) -> None: ...

class RegisterNodeResponse(_message.Message):
    __slots__ = ("node",)
    NODE_FIELD_NUMBER: _ClassVar[int]
    node: Node
    def __init__(self, node: _Optional[_Union[Node, _Mapping]] = ...) -> None: ...

class ListNodesRequest(_message.Message):
    __slots__ = ()
    def __init__(self) -> None: ...

class ListNodesResponse(_message.Message):
    __slots__ = ("nodes",)
    NODES_FIELD_NUMBER: _ClassVar[int]
    nodes: _containers.RepeatedCompositeFieldContainer[Node]
    def __init__(self, nodes: _Optional[_Iterable[_Union[Node, _Mapping]]] = ...) -> None: ...

class GetNodeRequest(_message.Message):
    __slots__ = ("id",)
    ID_FIELD_NUMBER: _ClassVar[int]
    id: str
    def __init__(self, id: _Optional[str] = ...) -> None: ...

class GetNodeResponse(_message.Message):
    __slots__ = ("node",)
    NODE_FIELD_NUMBER: _ClassVar[int]
    node: Node
    def __init__(self, node: _Optional[_Union[Node, _Mapping]] = ...) -> None: ...

class UpdateNodeRequest(_message.Message):
    __slots__ = ("id", "name", "endpoint", "capabilities", "scopes", "revision")
    ID_FIELD_NUMBER: _ClassVar[int]
    NAME_FIELD_NUMBER: _ClassVar[int]
    ENDPOINT_FIELD_NUMBER: _ClassVar[int]
    CAPABILITIES_FIELD_NUMBER: _ClassVar[int]
    SCOPES_FIELD_NUMBER: _ClassVar[int]
    REVISION_FIELD_NUMBER: _ClassVar[int]
    id: str
    name: str
    endpoint: str
    capabilities: _containers.RepeatedScalarFieldContainer[str]
    scopes: _containers.RepeatedScalarFieldContainer[str]
    revision: str
    def __init__(self, id: _Optional[str] = ..., name: _Optional[str] = ..., endpoint: _Optional[str] = ..., capabilities: _Optional[_Iterable[str]] = ..., scopes: _Optional[_Iterable[str]] = ..., revision: _Optional[str] = ...) -> None: ...

class UpdateNodeResponse(_message.Message):
    __slots__ = ("node",)
    NODE_FIELD_NUMBER: _ClassVar[int]
    node: Node
    def __init__(self, node: _Optional[_Union[Node, _Mapping]] = ...) -> None: ...

class RevokeNodeRequest(_message.Message):
    __slots__ = ("id",)
    ID_FIELD_NUMBER: _ClassVar[int]
    id: str
    def __init__(self, id: _Optional[str] = ...) -> None: ...

class RevokeNodeResponse(_message.Message):
    __slots__ = ("node",)
    NODE_FIELD_NUMBER: _ClassVar[int]
    node: Node
    def __init__(self, node: _Optional[_Union[Node, _Mapping]] = ...) -> None: ...

class RemoveNodeRequest(_message.Message):
    __slots__ = ("id",)
    ID_FIELD_NUMBER: _ClassVar[int]
    id: str
    def __init__(self, id: _Optional[str] = ...) -> None: ...

class RemoveNodeResponse(_message.Message):
    __slots__ = ("removed_node_id",)
    REMOVED_NODE_ID_FIELD_NUMBER: _ClassVar[int]
    removed_node_id: str
    def __init__(self, removed_node_id: _Optional[str] = ...) -> None: ...
