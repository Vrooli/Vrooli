import datetime

from google.protobuf import timestamp_pb2 as _timestamp_pb2
from google.protobuf.internal import containers as _containers
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Iterable as _Iterable, Mapping as _Mapping
from typing import ClassVar as _ClassVar, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class Incident(_message.Message):
    __slots__ = ("id", "fingerprint", "title", "severity", "status", "summary", "first_seen_at", "last_seen_at")
    ID_FIELD_NUMBER: _ClassVar[int]
    FINGERPRINT_FIELD_NUMBER: _ClassVar[int]
    TITLE_FIELD_NUMBER: _ClassVar[int]
    SEVERITY_FIELD_NUMBER: _ClassVar[int]
    STATUS_FIELD_NUMBER: _ClassVar[int]
    SUMMARY_FIELD_NUMBER: _ClassVar[int]
    FIRST_SEEN_AT_FIELD_NUMBER: _ClassVar[int]
    LAST_SEEN_AT_FIELD_NUMBER: _ClassVar[int]
    id: str
    fingerprint: str
    title: str
    severity: str
    status: str
    summary: str
    first_seen_at: _timestamp_pb2.Timestamp
    last_seen_at: _timestamp_pb2.Timestamp
    def __init__(self, id: _Optional[str] = ..., fingerprint: _Optional[str] = ..., title: _Optional[str] = ..., severity: _Optional[str] = ..., status: _Optional[str] = ..., summary: _Optional[str] = ..., first_seen_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., last_seen_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ...) -> None: ...

class Observation(_message.Message):
    __slots__ = ("id", "incident_id", "kind", "payload_json", "observed_at")
    ID_FIELD_NUMBER: _ClassVar[int]
    INCIDENT_ID_FIELD_NUMBER: _ClassVar[int]
    KIND_FIELD_NUMBER: _ClassVar[int]
    PAYLOAD_JSON_FIELD_NUMBER: _ClassVar[int]
    OBSERVED_AT_FIELD_NUMBER: _ClassVar[int]
    id: str
    incident_id: str
    kind: str
    payload_json: str
    observed_at: _timestamp_pb2.Timestamp
    def __init__(self, id: _Optional[str] = ..., incident_id: _Optional[str] = ..., kind: _Optional[str] = ..., payload_json: _Optional[str] = ..., observed_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ...) -> None: ...

class ListIncidentsRequest(_message.Message):
    __slots__ = ("status", "severity", "limit")
    STATUS_FIELD_NUMBER: _ClassVar[int]
    SEVERITY_FIELD_NUMBER: _ClassVar[int]
    LIMIT_FIELD_NUMBER: _ClassVar[int]
    status: str
    severity: str
    limit: int
    def __init__(self, status: _Optional[str] = ..., severity: _Optional[str] = ..., limit: _Optional[int] = ...) -> None: ...

class ListIncidentsResponse(_message.Message):
    __slots__ = ("incidents",)
    INCIDENTS_FIELD_NUMBER: _ClassVar[int]
    incidents: _containers.RepeatedCompositeFieldContainer[Incident]
    def __init__(self, incidents: _Optional[_Iterable[_Union[Incident, _Mapping]]] = ...) -> None: ...

class GetIncidentRequest(_message.Message):
    __slots__ = ("incident_id",)
    INCIDENT_ID_FIELD_NUMBER: _ClassVar[int]
    incident_id: str
    def __init__(self, incident_id: _Optional[str] = ...) -> None: ...

class GetIncidentResponse(_message.Message):
    __slots__ = ("incident",)
    INCIDENT_FIELD_NUMBER: _ClassVar[int]
    incident: Incident
    def __init__(self, incident: _Optional[_Union[Incident, _Mapping]] = ...) -> None: ...

class ListObservationsRequest(_message.Message):
    __slots__ = ("incident_id", "limit")
    INCIDENT_ID_FIELD_NUMBER: _ClassVar[int]
    LIMIT_FIELD_NUMBER: _ClassVar[int]
    incident_id: str
    limit: int
    def __init__(self, incident_id: _Optional[str] = ..., limit: _Optional[int] = ...) -> None: ...

class ListObservationsResponse(_message.Message):
    __slots__ = ("observations",)
    OBSERVATIONS_FIELD_NUMBER: _ClassVar[int]
    observations: _containers.RepeatedCompositeFieldContainer[Observation]
    def __init__(self, observations: _Optional[_Iterable[_Union[Observation, _Mapping]]] = ...) -> None: ...
