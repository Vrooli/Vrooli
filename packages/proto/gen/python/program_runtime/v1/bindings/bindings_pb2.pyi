from google.protobuf.internal import containers as _containers
from google.protobuf.internal import enum_type_wrapper as _enum_type_wrapper
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Iterable as _Iterable, Mapping as _Mapping
from typing import ClassVar as _ClassVar, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class UnboundReason(int, metaclass=_enum_type_wrapper.EnumTypeWrapper):
    __slots__ = ()
    UNBOUND_REASON_UNSPECIFIED: _ClassVar[UnboundReason]
    UNBOUND_REASON_NO_MANIFEST: _ClassVar[UnboundReason]
    UNBOUND_REASON_LOCAL_BINDING: _ClassVar[UnboundReason]
    UNBOUND_REASON_OMITTED_RPC: _ClassVar[UnboundReason]
    UNBOUND_REASON_EXTERNAL_TOOL_ONLY: _ClassVar[UnboundReason]

class ActVerdict(int, metaclass=_enum_type_wrapper.EnumTypeWrapper):
    __slots__ = ()
    ACT_VERDICT_UNSPECIFIED: _ClassVar[ActVerdict]
    ACT_VERDICT_NOW: _ClassVar[ActVerdict]
    ACT_VERDICT_IN_REACH: _ClassVar[ActVerdict]
    ACT_VERDICT_AUTHORED: _ClassVar[ActVerdict]
UNBOUND_REASON_UNSPECIFIED: UnboundReason
UNBOUND_REASON_NO_MANIFEST: UnboundReason
UNBOUND_REASON_LOCAL_BINDING: UnboundReason
UNBOUND_REASON_OMITTED_RPC: UnboundReason
UNBOUND_REASON_EXTERNAL_TOOL_ONLY: UnboundReason
ACT_VERDICT_UNSPECIFIED: ActVerdict
ACT_VERDICT_NOW: ActVerdict
ACT_VERDICT_IN_REACH: ActVerdict
ACT_VERDICT_AUTHORED: ActVerdict

class Binding(_message.Message):
    __slots__ = ("id", "scenario", "group", "command", "service", "method", "request_type", "response_type", "effect", "run_eligible", "requires_confirmation", "permissions", "description", "signature")
    ID_FIELD_NUMBER: _ClassVar[int]
    SCENARIO_FIELD_NUMBER: _ClassVar[int]
    GROUP_FIELD_NUMBER: _ClassVar[int]
    COMMAND_FIELD_NUMBER: _ClassVar[int]
    SERVICE_FIELD_NUMBER: _ClassVar[int]
    METHOD_FIELD_NUMBER: _ClassVar[int]
    REQUEST_TYPE_FIELD_NUMBER: _ClassVar[int]
    RESPONSE_TYPE_FIELD_NUMBER: _ClassVar[int]
    EFFECT_FIELD_NUMBER: _ClassVar[int]
    RUN_ELIGIBLE_FIELD_NUMBER: _ClassVar[int]
    REQUIRES_CONFIRMATION_FIELD_NUMBER: _ClassVar[int]
    PERMISSIONS_FIELD_NUMBER: _ClassVar[int]
    DESCRIPTION_FIELD_NUMBER: _ClassVar[int]
    SIGNATURE_FIELD_NUMBER: _ClassVar[int]
    id: str
    scenario: str
    group: str
    command: str
    service: str
    method: str
    request_type: str
    response_type: str
    effect: str
    run_eligible: bool
    requires_confirmation: bool
    permissions: _containers.RepeatedScalarFieldContainer[str]
    description: str
    signature: str
    def __init__(self, id: _Optional[str] = ..., scenario: _Optional[str] = ..., group: _Optional[str] = ..., command: _Optional[str] = ..., service: _Optional[str] = ..., method: _Optional[str] = ..., request_type: _Optional[str] = ..., response_type: _Optional[str] = ..., effect: _Optional[str] = ..., run_eligible: _Optional[bool] = ..., requires_confirmation: _Optional[bool] = ..., permissions: _Optional[_Iterable[str]] = ..., description: _Optional[str] = ..., signature: _Optional[str] = ...) -> None: ...

class UnboundCapability(_message.Message):
    __slots__ = ("scenario", "group", "command", "service", "method", "reason", "detail")
    SCENARIO_FIELD_NUMBER: _ClassVar[int]
    GROUP_FIELD_NUMBER: _ClassVar[int]
    COMMAND_FIELD_NUMBER: _ClassVar[int]
    SERVICE_FIELD_NUMBER: _ClassVar[int]
    METHOD_FIELD_NUMBER: _ClassVar[int]
    REASON_FIELD_NUMBER: _ClassVar[int]
    DETAIL_FIELD_NUMBER: _ClassVar[int]
    scenario: str
    group: str
    command: str
    service: str
    method: str
    reason: UnboundReason
    detail: str
    def __init__(self, scenario: _Optional[str] = ..., group: _Optional[str] = ..., command: _Optional[str] = ..., service: _Optional[str] = ..., method: _Optional[str] = ..., reason: _Optional[_Union[UnboundReason, str]] = ..., detail: _Optional[str] = ...) -> None: ...

class ListBindingsRequest(_message.Message):
    __slots__ = ("scenario", "group")
    SCENARIO_FIELD_NUMBER: _ClassVar[int]
    GROUP_FIELD_NUMBER: _ClassVar[int]
    scenario: str
    group: str
    def __init__(self, scenario: _Optional[str] = ..., group: _Optional[str] = ...) -> None: ...

class ListBindingsResponse(_message.Message):
    __slots__ = ("bindings",)
    BINDINGS_FIELD_NUMBER: _ClassVar[int]
    bindings: _containers.RepeatedCompositeFieldContainer[Binding]
    def __init__(self, bindings: _Optional[_Iterable[_Union[Binding, _Mapping]]] = ...) -> None: ...

class ListUnboundRequest(_message.Message):
    __slots__ = ("scenario",)
    SCENARIO_FIELD_NUMBER: _ClassVar[int]
    scenario: str
    def __init__(self, scenario: _Optional[str] = ...) -> None: ...

class ListUnboundResponse(_message.Message):
    __slots__ = ("capabilities",)
    CAPABILITIES_FIELD_NUMBER: _ClassVar[int]
    capabilities: _containers.RepeatedCompositeFieldContainer[UnboundCapability]
    def __init__(self, capabilities: _Optional[_Iterable[_Union[UnboundCapability, _Mapping]]] = ...) -> None: ...

class ActCell(_message.Message):
    __slots__ = ("id", "operations", "authored_status")
    ID_FIELD_NUMBER: _ClassVar[int]
    OPERATIONS_FIELD_NUMBER: _ClassVar[int]
    AUTHORED_STATUS_FIELD_NUMBER: _ClassVar[int]
    id: str
    operations: _containers.RepeatedScalarFieldContainer[str]
    authored_status: str
    def __init__(self, id: _Optional[str] = ..., operations: _Optional[_Iterable[str]] = ..., authored_status: _Optional[str] = ...) -> None: ...

class ActCellVerdict(_message.Message):
    __slots__ = ("id", "verdict", "resolved_operations", "unresolved_operations", "reasons", "authored_status", "audited")
    ID_FIELD_NUMBER: _ClassVar[int]
    VERDICT_FIELD_NUMBER: _ClassVar[int]
    RESOLVED_OPERATIONS_FIELD_NUMBER: _ClassVar[int]
    UNRESOLVED_OPERATIONS_FIELD_NUMBER: _ClassVar[int]
    REASONS_FIELD_NUMBER: _ClassVar[int]
    AUTHORED_STATUS_FIELD_NUMBER: _ClassVar[int]
    AUDITED_FIELD_NUMBER: _ClassVar[int]
    id: str
    verdict: ActVerdict
    resolved_operations: _containers.RepeatedScalarFieldContainer[str]
    unresolved_operations: _containers.RepeatedScalarFieldContainer[str]
    reasons: _containers.RepeatedScalarFieldContainer[str]
    authored_status: str
    audited: bool
    def __init__(self, id: _Optional[str] = ..., verdict: _Optional[_Union[ActVerdict, str]] = ..., resolved_operations: _Optional[_Iterable[str]] = ..., unresolved_operations: _Optional[_Iterable[str]] = ..., reasons: _Optional[_Iterable[str]] = ..., authored_status: _Optional[str] = ..., audited: _Optional[bool] = ...) -> None: ...

class ResolveActCellsRequest(_message.Message):
    __slots__ = ("cells",)
    CELLS_FIELD_NUMBER: _ClassVar[int]
    cells: _containers.RepeatedCompositeFieldContainer[ActCell]
    def __init__(self, cells: _Optional[_Iterable[_Union[ActCell, _Mapping]]] = ...) -> None: ...

class ResolveActCellsResponse(_message.Message):
    __slots__ = ("cells", "audited_cells", "total_cells", "denominator_confidence")
    CELLS_FIELD_NUMBER: _ClassVar[int]
    AUDITED_CELLS_FIELD_NUMBER: _ClassVar[int]
    TOTAL_CELLS_FIELD_NUMBER: _ClassVar[int]
    DENOMINATOR_CONFIDENCE_FIELD_NUMBER: _ClassVar[int]
    cells: _containers.RepeatedCompositeFieldContainer[ActCellVerdict]
    audited_cells: int
    total_cells: int
    denominator_confidence: str
    def __init__(self, cells: _Optional[_Iterable[_Union[ActCellVerdict, _Mapping]]] = ..., audited_cells: _Optional[int] = ..., total_cells: _Optional[int] = ..., denominator_confidence: _Optional[str] = ...) -> None: ...
