import datetime

from google.api import annotations_pb2 as _annotations_pb2
from google.protobuf import struct_pb2 as _struct_pb2
from google.protobuf import timestamp_pb2 as _timestamp_pb2
from google.protobuf.internal import containers as _containers
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Iterable as _Iterable, Mapping as _Mapping
from typing import ClassVar as _ClassVar, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class Alert(_message.Message):
    __slots__ = ("id", "type", "severity", "message", "rule_id", "key", "metric_name", "metric_value", "threshold", "opened_at", "resolved_at", "acked_at", "acked_by", "escalated", "anomaly_id", "details")
    ID_FIELD_NUMBER: _ClassVar[int]
    TYPE_FIELD_NUMBER: _ClassVar[int]
    SEVERITY_FIELD_NUMBER: _ClassVar[int]
    MESSAGE_FIELD_NUMBER: _ClassVar[int]
    RULE_ID_FIELD_NUMBER: _ClassVar[int]
    KEY_FIELD_NUMBER: _ClassVar[int]
    METRIC_NAME_FIELD_NUMBER: _ClassVar[int]
    METRIC_VALUE_FIELD_NUMBER: _ClassVar[int]
    THRESHOLD_FIELD_NUMBER: _ClassVar[int]
    OPENED_AT_FIELD_NUMBER: _ClassVar[int]
    RESOLVED_AT_FIELD_NUMBER: _ClassVar[int]
    ACKED_AT_FIELD_NUMBER: _ClassVar[int]
    ACKED_BY_FIELD_NUMBER: _ClassVar[int]
    ESCALATED_FIELD_NUMBER: _ClassVar[int]
    ANOMALY_ID_FIELD_NUMBER: _ClassVar[int]
    DETAILS_FIELD_NUMBER: _ClassVar[int]
    id: str
    type: str
    severity: str
    message: str
    rule_id: str
    key: str
    metric_name: str
    metric_value: float
    threshold: float
    opened_at: _timestamp_pb2.Timestamp
    resolved_at: _timestamp_pb2.Timestamp
    acked_at: _timestamp_pb2.Timestamp
    acked_by: str
    escalated: bool
    anomaly_id: str
    details: _struct_pb2.Struct
    def __init__(self, id: _Optional[str] = ..., type: _Optional[str] = ..., severity: _Optional[str] = ..., message: _Optional[str] = ..., rule_id: _Optional[str] = ..., key: _Optional[str] = ..., metric_name: _Optional[str] = ..., metric_value: _Optional[float] = ..., threshold: _Optional[float] = ..., opened_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., resolved_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., acked_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., acked_by: _Optional[str] = ..., escalated: _Optional[bool] = ..., anomaly_id: _Optional[str] = ..., details: _Optional[_Union[_struct_pb2.Struct, _Mapping]] = ...) -> None: ...

class RuleState(_message.Message):
    __slots__ = ("rule_id", "status", "reason", "condition_active", "pending_since", "evaluated_at")
    RULE_ID_FIELD_NUMBER: _ClassVar[int]
    STATUS_FIELD_NUMBER: _ClassVar[int]
    REASON_FIELD_NUMBER: _ClassVar[int]
    CONDITION_ACTIVE_FIELD_NUMBER: _ClassVar[int]
    PENDING_SINCE_FIELD_NUMBER: _ClassVar[int]
    EVALUATED_AT_FIELD_NUMBER: _ClassVar[int]
    rule_id: str
    status: str
    reason: str
    condition_active: bool
    pending_since: _timestamp_pb2.Timestamp
    evaluated_at: _timestamp_pb2.Timestamp
    def __init__(self, rule_id: _Optional[str] = ..., status: _Optional[str] = ..., reason: _Optional[str] = ..., condition_active: _Optional[bool] = ..., pending_since: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., evaluated_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ...) -> None: ...

class ListAlertsRequest(_message.Message):
    __slots__ = ("status", "include_events", "limit")
    STATUS_FIELD_NUMBER: _ClassVar[int]
    INCLUDE_EVENTS_FIELD_NUMBER: _ClassVar[int]
    LIMIT_FIELD_NUMBER: _ClassVar[int]
    status: str
    include_events: bool
    limit: int
    def __init__(self, status: _Optional[str] = ..., include_events: _Optional[bool] = ..., limit: _Optional[int] = ...) -> None: ...

class ListAlertsResponse(_message.Message):
    __slots__ = ("alerts", "rules", "evaluated_at", "sustain_seconds", "monitoring_active")
    ALERTS_FIELD_NUMBER: _ClassVar[int]
    RULES_FIELD_NUMBER: _ClassVar[int]
    EVALUATED_AT_FIELD_NUMBER: _ClassVar[int]
    SUSTAIN_SECONDS_FIELD_NUMBER: _ClassVar[int]
    MONITORING_ACTIVE_FIELD_NUMBER: _ClassVar[int]
    alerts: _containers.RepeatedCompositeFieldContainer[Alert]
    rules: _containers.RepeatedCompositeFieldContainer[RuleState]
    evaluated_at: _timestamp_pb2.Timestamp
    sustain_seconds: int
    monitoring_active: bool
    def __init__(self, alerts: _Optional[_Iterable[_Union[Alert, _Mapping]]] = ..., rules: _Optional[_Iterable[_Union[RuleState, _Mapping]]] = ..., evaluated_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., sustain_seconds: _Optional[int] = ..., monitoring_active: _Optional[bool] = ...) -> None: ...

class AcknowledgeAlertRequest(_message.Message):
    __slots__ = ("id", "acked_by")
    ID_FIELD_NUMBER: _ClassVar[int]
    ACKED_BY_FIELD_NUMBER: _ClassVar[int]
    id: str
    acked_by: str
    def __init__(self, id: _Optional[str] = ..., acked_by: _Optional[str] = ...) -> None: ...

class AcknowledgeAlertResponse(_message.Message):
    __slots__ = ("alert",)
    ALERT_FIELD_NUMBER: _ClassVar[int]
    alert: Alert
    def __init__(self, alert: _Optional[_Union[Alert, _Mapping]] = ...) -> None: ...
