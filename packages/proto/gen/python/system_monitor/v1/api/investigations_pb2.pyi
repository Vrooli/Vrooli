from google.protobuf import struct_pb2 as _struct_pb2
from system_monitor.v1.domain import investigations_pb2 as _investigations_pb2
from system_monitor.v1.domain import types_pb2 as _types_pb2
from google.protobuf.internal import containers as _containers
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Iterable as _Iterable, Mapping as _Mapping
from typing import ClassVar as _ClassVar, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class TriggerInvestigationRequest(_message.Message):
    __slots__ = ("auto_fix", "note")
    AUTO_FIX_FIELD_NUMBER: _ClassVar[int]
    NOTE_FIELD_NUMBER: _ClassVar[int]
    auto_fix: bool
    note: str
    def __init__(self, auto_fix: _Optional[bool] = ..., note: _Optional[str] = ...) -> None: ...

class TriggerInvestigationResponse(_message.Message):
    __slots__ = ("status", "investigation_id", "api_base_url", "message", "auto_fix", "note")
    STATUS_FIELD_NUMBER: _ClassVar[int]
    INVESTIGATION_ID_FIELD_NUMBER: _ClassVar[int]
    API_BASE_URL_FIELD_NUMBER: _ClassVar[int]
    MESSAGE_FIELD_NUMBER: _ClassVar[int]
    AUTO_FIX_FIELD_NUMBER: _ClassVar[int]
    NOTE_FIELD_NUMBER: _ClassVar[int]
    status: str
    investigation_id: str
    api_base_url: str
    message: str
    auto_fix: bool
    note: str
    def __init__(self, status: _Optional[str] = ..., investigation_id: _Optional[str] = ..., api_base_url: _Optional[str] = ..., message: _Optional[str] = ..., auto_fix: _Optional[bool] = ..., note: _Optional[str] = ...) -> None: ...

class GetInvestigationRequest(_message.Message):
    __slots__ = ("id",)
    ID_FIELD_NUMBER: _ClassVar[int]
    id: str
    def __init__(self, id: _Optional[str] = ...) -> None: ...

class GetInvestigationResponse(_message.Message):
    __slots__ = ("investigation",)
    INVESTIGATION_FIELD_NUMBER: _ClassVar[int]
    investigation: _investigations_pb2.Investigation
    def __init__(self, investigation: _Optional[_Union[_investigations_pb2.Investigation, _Mapping]] = ...) -> None: ...

class GetLatestInvestigationRequest(_message.Message):
    __slots__ = ()
    def __init__(self) -> None: ...

class GetLatestInvestigationResponse(_message.Message):
    __slots__ = ("investigation",)
    INVESTIGATION_FIELD_NUMBER: _ClassVar[int]
    investigation: _investigations_pb2.Investigation
    def __init__(self, investigation: _Optional[_Union[_investigations_pb2.Investigation, _Mapping]] = ...) -> None: ...

class ListInvestigationsRequest(_message.Message):
    __slots__ = ("limit",)
    LIMIT_FIELD_NUMBER: _ClassVar[int]
    limit: int
    def __init__(self, limit: _Optional[int] = ...) -> None: ...

class ListInvestigationsResponse(_message.Message):
    __slots__ = ("investigations",)
    INVESTIGATIONS_FIELD_NUMBER: _ClassVar[int]
    investigations: _containers.RepeatedCompositeFieldContainer[_investigations_pb2.Investigation]
    def __init__(self, investigations: _Optional[_Iterable[_Union[_investigations_pb2.Investigation, _Mapping]]] = ...) -> None: ...

class UpdateInvestigationStatusRequest(_message.Message):
    __slots__ = ("id", "status")
    ID_FIELD_NUMBER: _ClassVar[int]
    STATUS_FIELD_NUMBER: _ClassVar[int]
    id: str
    status: _types_pb2.InvestigationStatus
    def __init__(self, id: _Optional[str] = ..., status: _Optional[_Union[_types_pb2.InvestigationStatus, str]] = ...) -> None: ...

class UpdateInvestigationStatusResponse(_message.Message):
    __slots__ = ("status",)
    STATUS_FIELD_NUMBER: _ClassVar[int]
    status: str
    def __init__(self, status: _Optional[str] = ...) -> None: ...

class UpdateInvestigationFindingsRequest(_message.Message):
    __slots__ = ("id", "findings", "details")
    ID_FIELD_NUMBER: _ClassVar[int]
    FINDINGS_FIELD_NUMBER: _ClassVar[int]
    DETAILS_FIELD_NUMBER: _ClassVar[int]
    id: str
    findings: str
    details: _struct_pb2.Struct
    def __init__(self, id: _Optional[str] = ..., findings: _Optional[str] = ..., details: _Optional[_Union[_struct_pb2.Struct, _Mapping]] = ...) -> None: ...

class UpdateInvestigationFindingsResponse(_message.Message):
    __slots__ = ("status",)
    STATUS_FIELD_NUMBER: _ClassVar[int]
    status: str
    def __init__(self, status: _Optional[str] = ...) -> None: ...

class UpdateInvestigationProgressRequest(_message.Message):
    __slots__ = ("id", "progress")
    ID_FIELD_NUMBER: _ClassVar[int]
    PROGRESS_FIELD_NUMBER: _ClassVar[int]
    id: str
    progress: int
    def __init__(self, id: _Optional[str] = ..., progress: _Optional[int] = ...) -> None: ...

class UpdateInvestigationProgressResponse(_message.Message):
    __slots__ = ("status",)
    STATUS_FIELD_NUMBER: _ClassVar[int]
    status: str
    def __init__(self, status: _Optional[str] = ...) -> None: ...

class AddInvestigationStepRequest(_message.Message):
    __slots__ = ("id", "step")
    ID_FIELD_NUMBER: _ClassVar[int]
    STEP_FIELD_NUMBER: _ClassVar[int]
    id: str
    step: _investigations_pb2.InvestigationStep
    def __init__(self, id: _Optional[str] = ..., step: _Optional[_Union[_investigations_pb2.InvestigationStep, _Mapping]] = ...) -> None: ...

class AddInvestigationStepResponse(_message.Message):
    __slots__ = ("status",)
    STATUS_FIELD_NUMBER: _ClassVar[int]
    status: str
    def __init__(self, status: _Optional[str] = ...) -> None: ...

class GetCooldownStatusRequest(_message.Message):
    __slots__ = ()
    def __init__(self) -> None: ...

class GetCooldownStatusResponse(_message.Message):
    __slots__ = ("cooldown",)
    COOLDOWN_FIELD_NUMBER: _ClassVar[int]
    cooldown: _investigations_pb2.CooldownStatus
    def __init__(self, cooldown: _Optional[_Union[_investigations_pb2.CooldownStatus, _Mapping]] = ...) -> None: ...

class GetTriggersRequest(_message.Message):
    __slots__ = ()
    def __init__(self) -> None: ...

class GetTriggersResponse(_message.Message):
    __slots__ = ("triggers",)
    class TriggersEntry(_message.Message):
        __slots__ = ("key", "value")
        KEY_FIELD_NUMBER: _ClassVar[int]
        VALUE_FIELD_NUMBER: _ClassVar[int]
        key: str
        value: _investigations_pb2.TriggerConfig
        def __init__(self, key: _Optional[str] = ..., value: _Optional[_Union[_investigations_pb2.TriggerConfig, _Mapping]] = ...) -> None: ...
    TRIGGERS_FIELD_NUMBER: _ClassVar[int]
    triggers: _containers.MessageMap[str, _investigations_pb2.TriggerConfig]
    def __init__(self, triggers: _Optional[_Mapping[str, _investigations_pb2.TriggerConfig]] = ...) -> None: ...

class UpdateTriggerRequest(_message.Message):
    __slots__ = ("id", "enabled", "auto_fix", "threshold")
    ID_FIELD_NUMBER: _ClassVar[int]
    ENABLED_FIELD_NUMBER: _ClassVar[int]
    AUTO_FIX_FIELD_NUMBER: _ClassVar[int]
    THRESHOLD_FIELD_NUMBER: _ClassVar[int]
    id: str
    enabled: bool
    auto_fix: bool
    threshold: float
    def __init__(self, id: _Optional[str] = ..., enabled: _Optional[bool] = ..., auto_fix: _Optional[bool] = ..., threshold: _Optional[float] = ...) -> None: ...

class UpdateTriggerResponse(_message.Message):
    __slots__ = ("status",)
    STATUS_FIELD_NUMBER: _ClassVar[int]
    status: str
    def __init__(self, status: _Optional[str] = ...) -> None: ...
