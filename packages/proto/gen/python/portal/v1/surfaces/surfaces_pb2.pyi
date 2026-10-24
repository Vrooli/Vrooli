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

class ListApplicationsRequest(_message.Message):
    __slots__ = ()
    def __init__(self) -> None: ...

class Application(_message.Message):
    __slots__ = ("scenario", "label", "description", "runtime_state", "tv_qualification", "icon_path")
    SCENARIO_FIELD_NUMBER: _ClassVar[int]
    LABEL_FIELD_NUMBER: _ClassVar[int]
    DESCRIPTION_FIELD_NUMBER: _ClassVar[int]
    RUNTIME_STATE_FIELD_NUMBER: _ClassVar[int]
    TV_QUALIFICATION_FIELD_NUMBER: _ClassVar[int]
    ICON_PATH_FIELD_NUMBER: _ClassVar[int]
    scenario: str
    label: str
    description: str
    runtime_state: str
    tv_qualification: str
    icon_path: str
    def __init__(self, scenario: _Optional[str] = ..., label: _Optional[str] = ..., description: _Optional[str] = ..., runtime_state: _Optional[str] = ..., tv_qualification: _Optional[str] = ..., icon_path: _Optional[str] = ...) -> None: ...

class ListApplicationsResponse(_message.Message):
    __slots__ = ("applications", "observed_at")
    APPLICATIONS_FIELD_NUMBER: _ClassVar[int]
    OBSERVED_AT_FIELD_NUMBER: _ClassVar[int]
    applications: _containers.RepeatedCompositeFieldContainer[Application]
    observed_at: _timestamp_pb2.Timestamp
    def __init__(self, applications: _Optional[_Iterable[_Union[Application, _Mapping]]] = ..., observed_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ...) -> None: ...

class ResolveApplicationRequest(_message.Message):
    __slots__ = ("scenario", "use_gateway")
    SCENARIO_FIELD_NUMBER: _ClassVar[int]
    USE_GATEWAY_FIELD_NUMBER: _ClassVar[int]
    scenario: str
    use_gateway: bool
    def __init__(self, scenario: _Optional[str] = ..., use_gateway: _Optional[bool] = ...) -> None: ...

class ResolveApplicationResponse(_message.Message):
    __slots__ = ("application", "url", "observed_at")
    APPLICATION_FIELD_NUMBER: _ClassVar[int]
    URL_FIELD_NUMBER: _ClassVar[int]
    OBSERVED_AT_FIELD_NUMBER: _ClassVar[int]
    application: Application
    url: str
    observed_at: _timestamp_pb2.Timestamp
    def __init__(self, application: _Optional[_Union[Application, _Mapping]] = ..., url: _Optional[str] = ..., observed_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ...) -> None: ...

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
