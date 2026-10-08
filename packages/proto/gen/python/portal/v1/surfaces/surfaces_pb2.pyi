import datetime

from common.v1 import surface_pb2 as _surface_pb2
from device_control.v1.desktop import desktop_pb2 as _desktop_pb2
from scenario_authenticator.v1.accounts import accounts_pb2 as _accounts_pb2
from google.protobuf import timestamp_pb2 as _timestamp_pb2
from google.protobuf.internal import containers as _containers
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Iterable as _Iterable, Mapping as _Mapping
from typing import ClassVar as _ClassVar, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class SourceStatus(_message.Message):
    __slots__ = ("owner_scenario", "state", "reason_code")
    OWNER_SCENARIO_FIELD_NUMBER: _ClassVar[int]
    STATE_FIELD_NUMBER: _ClassVar[int]
    REASON_CODE_FIELD_NUMBER: _ClassVar[int]
    owner_scenario: str
    state: str
    reason_code: str
    def __init__(self, owner_scenario: _Optional[str] = ..., state: _Optional[str] = ..., reason_code: _Optional[str] = ...) -> None: ...

class ListRequest(_message.Message):
    __slots__ = ()
    def __init__(self) -> None: ...

class ListResponse(_message.Message):
    __slots__ = ("surfaces", "sources", "observed_at")
    SURFACES_FIELD_NUMBER: _ClassVar[int]
    SOURCES_FIELD_NUMBER: _ClassVar[int]
    OBSERVED_AT_FIELD_NUMBER: _ClassVar[int]
    surfaces: _containers.RepeatedCompositeFieldContainer[_surface_pb2.SurfaceDescriptor]
    sources: _containers.RepeatedCompositeFieldContainer[SourceStatus]
    observed_at: _timestamp_pb2.Timestamp
    def __init__(self, surfaces: _Optional[_Iterable[_Union[_surface_pb2.SurfaceDescriptor, _Mapping]]] = ..., sources: _Optional[_Iterable[_Union[SourceStatus, _Mapping]]] = ..., observed_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ...) -> None: ...

class ResolveRequest(_message.Message):
    __slots__ = ("display_label", "exact_ref")
    DISPLAY_LABEL_FIELD_NUMBER: _ClassVar[int]
    EXACT_REF_FIELD_NUMBER: _ClassVar[int]
    display_label: str
    exact_ref: _surface_pb2.SurfaceRef
    def __init__(self, display_label: _Optional[str] = ..., exact_ref: _Optional[_Union[_surface_pb2.SurfaceRef, _Mapping]] = ...) -> None: ...

class ResolveResponse(_message.Message):
    __slots__ = ("matches", "sources", "observed_at")
    MATCHES_FIELD_NUMBER: _ClassVar[int]
    SOURCES_FIELD_NUMBER: _ClassVar[int]
    OBSERVED_AT_FIELD_NUMBER: _ClassVar[int]
    matches: _containers.RepeatedCompositeFieldContainer[_surface_pb2.SurfaceDescriptor]
    sources: _containers.RepeatedCompositeFieldContainer[SourceStatus]
    observed_at: _timestamp_pb2.Timestamp
    def __init__(self, matches: _Optional[_Iterable[_Union[_surface_pb2.SurfaceDescriptor, _Mapping]]] = ..., sources: _Optional[_Iterable[_Union[SourceStatus, _Mapping]]] = ..., observed_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ...) -> None: ...
