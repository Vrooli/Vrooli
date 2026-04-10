from google.protobuf.internal import enum_type_wrapper as _enum_type_wrapper
from google.protobuf import descriptor as _descriptor
from typing import ClassVar as _ClassVar

DESCRIPTOR: _descriptor.FileDescriptor

class WebSocketEventType(int, metaclass=_enum_type_wrapper.EnumTypeWrapper):
    __slots__ = ()
    WEB_SOCKET_EVENT_TYPE_UNSPECIFIED: _ClassVar[WebSocketEventType]
    WEB_SOCKET_EVENT_TYPE_TASK_UPDATED: _ClassVar[WebSocketEventType]
    WEB_SOCKET_EVENT_TYPE_TASK_CREATED: _ClassVar[WebSocketEventType]
    WEB_SOCKET_EVENT_TYPE_TASK_DELETED: _ClassVar[WebSocketEventType]
    WEB_SOCKET_EVENT_TYPE_QUEUE_STATUS: _ClassVar[WebSocketEventType]
    WEB_SOCKET_EVENT_TYPE_PROCESS_STARTED: _ClassVar[WebSocketEventType]
    WEB_SOCKET_EVENT_TYPE_PROCESS_STOPPED: _ClassVar[WebSocketEventType]
    WEB_SOCKET_EVENT_TYPE_SETTINGS_UPDATED: _ClassVar[WebSocketEventType]
    WEB_SOCKET_EVENT_TYPE_LOG_ENTRY: _ClassVar[WebSocketEventType]
    WEB_SOCKET_EVENT_TYPE_EXECUTION_COMPLETED: _ClassVar[WebSocketEventType]
    WEB_SOCKET_EVENT_TYPE_AUTOSTEER_STATE: _ClassVar[WebSocketEventType]
    WEB_SOCKET_EVENT_TYPE_HEALTH: _ClassVar[WebSocketEventType]
WEB_SOCKET_EVENT_TYPE_UNSPECIFIED: WebSocketEventType
WEB_SOCKET_EVENT_TYPE_TASK_UPDATED: WebSocketEventType
WEB_SOCKET_EVENT_TYPE_TASK_CREATED: WebSocketEventType
WEB_SOCKET_EVENT_TYPE_TASK_DELETED: WebSocketEventType
WEB_SOCKET_EVENT_TYPE_QUEUE_STATUS: WebSocketEventType
WEB_SOCKET_EVENT_TYPE_PROCESS_STARTED: WebSocketEventType
WEB_SOCKET_EVENT_TYPE_PROCESS_STOPPED: WebSocketEventType
WEB_SOCKET_EVENT_TYPE_SETTINGS_UPDATED: WebSocketEventType
WEB_SOCKET_EVENT_TYPE_LOG_ENTRY: WebSocketEventType
WEB_SOCKET_EVENT_TYPE_EXECUTION_COMPLETED: WebSocketEventType
WEB_SOCKET_EVENT_TYPE_AUTOSTEER_STATE: WebSocketEventType
WEB_SOCKET_EVENT_TYPE_HEALTH: WebSocketEventType
