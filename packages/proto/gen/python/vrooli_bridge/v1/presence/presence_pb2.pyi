from vrooli_bridge.v1.shared import shared_pb2 as _shared_pb2
from vrooli_bridge.v1.channel import channel_pb2 as _channel_pb2
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Mapping as _Mapping
from typing import ClassVar as _ClassVar, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class ReportHeartbeatRequest(_message.Message):
    __slots__ = ("heartbeat",)
    HEARTBEAT_FIELD_NUMBER: _ClassVar[int]
    heartbeat: _shared_pb2.Heartbeat
    def __init__(self, heartbeat: _Optional[_Union[_shared_pb2.Heartbeat, _Mapping]] = ...) -> None: ...

class ReportHeartbeatResponse(_message.Message):
    __slots__ = ("compatibility",)
    COMPATIBILITY_FIELD_NUMBER: _ClassVar[int]
    compatibility: _shared_pb2.CompatibilityStatus
    def __init__(self, compatibility: _Optional[_Union[_shared_pb2.CompatibilityStatus, str]] = ...) -> None: ...

class ReportDeliveryAckRequest(_message.Message):
    __slots__ = ("ack",)
    ACK_FIELD_NUMBER: _ClassVar[int]
    ack: _shared_pb2.DeliveryAck
    def __init__(self, ack: _Optional[_Union[_shared_pb2.DeliveryAck, _Mapping]] = ...) -> None: ...

class ReportDeliveryAckResponse(_message.Message):
    __slots__ = ("accepted",)
    ACCEPTED_FIELD_NUMBER: _ClassVar[int]
    accepted: bool
    def __init__(self, accepted: _Optional[bool] = ...) -> None: ...

class ReportSessionFrameRequest(_message.Message):
    __slots__ = ("frame",)
    FRAME_FIELD_NUMBER: _ClassVar[int]
    frame: _shared_pb2.SessionFrame
    def __init__(self, frame: _Optional[_Union[_shared_pb2.SessionFrame, _Mapping]] = ...) -> None: ...

class ReportSessionFrameResponse(_message.Message):
    __slots__ = ("accepted",)
    ACCEPTED_FIELD_NUMBER: _ClassVar[int]
    accepted: bool
    def __init__(self, accepted: _Optional[bool] = ...) -> None: ...

class ReportRelayResponseRequest(_message.Message):
    __slots__ = ("response",)
    RESPONSE_FIELD_NUMBER: _ClassVar[int]
    response: _shared_pb2.RelayResponse
    def __init__(self, response: _Optional[_Union[_shared_pb2.RelayResponse, _Mapping]] = ...) -> None: ...

class ReportRelayResponseResponse(_message.Message):
    __slots__ = ("accepted",)
    ACCEPTED_FIELD_NUMBER: _ClassVar[int]
    accepted: bool
    def __init__(self, accepted: _Optional[bool] = ...) -> None: ...

class ReportCredentialReceiptRequest(_message.Message):
    __slots__ = ("receipt",)
    RECEIPT_FIELD_NUMBER: _ClassVar[int]
    receipt: CredentialReceipt
    def __init__(self, receipt: _Optional[_Union[CredentialReceipt, _Mapping]] = ...) -> None: ...

class CredentialReceipt(_message.Message):
    __slots__ = ("grant_id", "node_id", "logical_id", "field", "generation", "accepted", "reason")
    GRANT_ID_FIELD_NUMBER: _ClassVar[int]
    NODE_ID_FIELD_NUMBER: _ClassVar[int]
    LOGICAL_ID_FIELD_NUMBER: _ClassVar[int]
    FIELD_FIELD_NUMBER: _ClassVar[int]
    GENERATION_FIELD_NUMBER: _ClassVar[int]
    ACCEPTED_FIELD_NUMBER: _ClassVar[int]
    REASON_FIELD_NUMBER: _ClassVar[int]
    grant_id: str
    node_id: str
    logical_id: str
    field: str
    generation: int
    accepted: bool
    reason: str
    def __init__(self, grant_id: _Optional[str] = ..., node_id: _Optional[str] = ..., logical_id: _Optional[str] = ..., field: _Optional[str] = ..., generation: _Optional[int] = ..., accepted: _Optional[bool] = ..., reason: _Optional[str] = ...) -> None: ...

class ReportCredentialReceiptResponse(_message.Message):
    __slots__ = ("accepted",)
    ACCEPTED_FIELD_NUMBER: _ClassVar[int]
    accepted: bool
    def __init__(self, accepted: _Optional[bool] = ...) -> None: ...

class ReportScenarioResponseRequest(_message.Message):
    __slots__ = ("response",)
    RESPONSE_FIELD_NUMBER: _ClassVar[int]
    response: _channel_pb2.ScenarioResponse
    def __init__(self, response: _Optional[_Union[_channel_pb2.ScenarioResponse, _Mapping]] = ...) -> None: ...

class ReportScenarioResponseResponse(_message.Message):
    __slots__ = ("accepted",)
    ACCEPTED_FIELD_NUMBER: _ClassVar[int]
    accepted: bool
    def __init__(self, accepted: _Optional[bool] = ...) -> None: ...
