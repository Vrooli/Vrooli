import datetime

from common.v1 import surface_pb2 as _surface_pb2
from device_control.v1.shared import flow_pb2 as _flow_pb2
from google.protobuf import timestamp_pb2 as _timestamp_pb2
from google.protobuf.internal import containers as _containers
from google.protobuf.internal import enum_type_wrapper as _enum_type_wrapper
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Iterable as _Iterable, Mapping as _Mapping
from typing import ClassVar as _ClassVar, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class SemanticMatchMode(int, metaclass=_enum_type_wrapper.EnumTypeWrapper):
    __slots__ = ()
    SEMANTIC_MATCH_MODE_EXACT: _ClassVar[SemanticMatchMode]
    SEMANTIC_MATCH_MODE_NORMALIZED: _ClassVar[SemanticMatchMode]
    SEMANTIC_MATCH_MODE_FUZZY: _ClassVar[SemanticMatchMode]
SEMANTIC_MATCH_MODE_EXACT: SemanticMatchMode
SEMANTIC_MATCH_MODE_NORMALIZED: SemanticMatchMode
SEMANTIC_MATCH_MODE_FUZZY: SemanticMatchMode

class OwnerDescribeRequest(_message.Message):
    __slots__ = ()
    def __init__(self) -> None: ...

class OwnerDescribeResponse(_message.Message):
    __slots__ = ("surface",)
    SURFACE_FIELD_NUMBER: _ClassVar[int]
    surface: _surface_pb2.SurfaceDescriptor
    def __init__(self, surface: _Optional[_Union[_surface_pb2.SurfaceDescriptor, _Mapping]] = ...) -> None: ...

class OwnerOpenRequest(_message.Message):
    __slots__ = ("surface", "ttl_seconds", "control", "request_id")
    SURFACE_FIELD_NUMBER: _ClassVar[int]
    TTL_SECONDS_FIELD_NUMBER: _ClassVar[int]
    CONTROL_FIELD_NUMBER: _ClassVar[int]
    REQUEST_ID_FIELD_NUMBER: _ClassVar[int]
    surface: _surface_pb2.SurfaceRef
    ttl_seconds: int
    control: bool
    request_id: str
    def __init__(self, surface: _Optional[_Union[_surface_pb2.SurfaceRef, _Mapping]] = ..., ttl_seconds: _Optional[int] = ..., control: _Optional[bool] = ..., request_id: _Optional[str] = ...) -> None: ...

class OwnerOpenResponse(_message.Message):
    __slots__ = ("session", "expires_at", "control")
    SESSION_FIELD_NUMBER: _ClassVar[int]
    EXPIRES_AT_FIELD_NUMBER: _ClassVar[int]
    CONTROL_FIELD_NUMBER: _ClassVar[int]
    session: _surface_pb2.SessionRef
    expires_at: _timestamp_pb2.Timestamp
    control: bool
    def __init__(self, session: _Optional[_Union[_surface_pb2.SessionRef, _Mapping]] = ..., expires_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., control: _Optional[bool] = ...) -> None: ...

class OwnerObserveRequest(_message.Message):
    __slots__ = ("session", "process_id", "application_id", "application_revision")
    SESSION_FIELD_NUMBER: _ClassVar[int]
    PROCESS_ID_FIELD_NUMBER: _ClassVar[int]
    APPLICATION_ID_FIELD_NUMBER: _ClassVar[int]
    APPLICATION_REVISION_FIELD_NUMBER: _ClassVar[int]
    session: _surface_pb2.SessionRef
    process_id: int
    application_id: str
    application_revision: str
    def __init__(self, session: _Optional[_Union[_surface_pb2.SessionRef, _Mapping]] = ..., process_id: _Optional[int] = ..., application_id: _Optional[str] = ..., application_revision: _Optional[str] = ...) -> None: ...

class OwnerActRequest(_message.Message):
    __slots__ = ("session", "command_id", "geometry_revision", "action")
    SESSION_FIELD_NUMBER: _ClassVar[int]
    COMMAND_ID_FIELD_NUMBER: _ClassVar[int]
    GEOMETRY_REVISION_FIELD_NUMBER: _ClassVar[int]
    ACTION_FIELD_NUMBER: _ClassVar[int]
    session: _surface_pb2.SessionRef
    command_id: str
    geometry_revision: str
    action: Action
    def __init__(self, session: _Optional[_Union[_surface_pb2.SessionRef, _Mapping]] = ..., command_id: _Optional[str] = ..., geometry_revision: _Optional[str] = ..., action: _Optional[_Union[Action, _Mapping]] = ...) -> None: ...

class OwnerStopRequest(_message.Message):
    __slots__ = ("session",)
    SESSION_FIELD_NUMBER: _ClassVar[int]
    session: _surface_pb2.SessionRef
    def __init__(self, session: _Optional[_Union[_surface_pb2.SessionRef, _Mapping]] = ...) -> None: ...

class Lease(_message.Message):
    __slots__ = ("ref", "actor", "helper_id", "epoch", "expires_at", "control")
    REF_FIELD_NUMBER: _ClassVar[int]
    ACTOR_FIELD_NUMBER: _ClassVar[int]
    HELPER_ID_FIELD_NUMBER: _ClassVar[int]
    EPOCH_FIELD_NUMBER: _ClassVar[int]
    EXPIRES_AT_FIELD_NUMBER: _ClassVar[int]
    CONTROL_FIELD_NUMBER: _ClassVar[int]
    ref: _surface_pb2.SessionRef
    actor: str
    helper_id: str
    epoch: int
    expires_at: _timestamp_pb2.Timestamp
    control: bool
    def __init__(self, ref: _Optional[_Union[_surface_pb2.SessionRef, _Mapping]] = ..., actor: _Optional[str] = ..., helper_id: _Optional[str] = ..., epoch: _Optional[int] = ..., expires_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., control: _Optional[bool] = ...) -> None: ...

class OpenRequest(_message.Message):
    __slots__ = ("lease", "expected_epoch", "takeover")
    LEASE_FIELD_NUMBER: _ClassVar[int]
    EXPECTED_EPOCH_FIELD_NUMBER: _ClassVar[int]
    TAKEOVER_FIELD_NUMBER: _ClassVar[int]
    lease: Lease
    expected_epoch: int
    takeover: bool
    def __init__(self, lease: _Optional[_Union[Lease, _Mapping]] = ..., expected_epoch: _Optional[int] = ..., takeover: _Optional[bool] = ...) -> None: ...

class OpenResponse(_message.Message):
    __slots__ = ("lease",)
    LEASE_FIELD_NUMBER: _ClassVar[int]
    lease: Lease
    def __init__(self, lease: _Optional[_Union[Lease, _Mapping]] = ...) -> None: ...

class PointerAction(_message.Message):
    __slots__ = ("kind", "display_id", "x", "y", "button")
    class Kind(int, metaclass=_enum_type_wrapper.EnumTypeWrapper):
        __slots__ = ()
        KIND_UNSPECIFIED: _ClassVar[PointerAction.Kind]
        KIND_MOVE: _ClassVar[PointerAction.Kind]
        KIND_DOWN: _ClassVar[PointerAction.Kind]
        KIND_UP: _ClassVar[PointerAction.Kind]
        KIND_CLICK: _ClassVar[PointerAction.Kind]
    KIND_UNSPECIFIED: PointerAction.Kind
    KIND_MOVE: PointerAction.Kind
    KIND_DOWN: PointerAction.Kind
    KIND_UP: PointerAction.Kind
    KIND_CLICK: PointerAction.Kind
    class Button(int, metaclass=_enum_type_wrapper.EnumTypeWrapper):
        __slots__ = ()
        BUTTON_UNSPECIFIED: _ClassVar[PointerAction.Button]
        BUTTON_PRIMARY: _ClassVar[PointerAction.Button]
        BUTTON_SECONDARY: _ClassVar[PointerAction.Button]
        BUTTON_MIDDLE: _ClassVar[PointerAction.Button]
    BUTTON_UNSPECIFIED: PointerAction.Button
    BUTTON_PRIMARY: PointerAction.Button
    BUTTON_SECONDARY: PointerAction.Button
    BUTTON_MIDDLE: PointerAction.Button
    KIND_FIELD_NUMBER: _ClassVar[int]
    DISPLAY_ID_FIELD_NUMBER: _ClassVar[int]
    X_FIELD_NUMBER: _ClassVar[int]
    Y_FIELD_NUMBER: _ClassVar[int]
    BUTTON_FIELD_NUMBER: _ClassVar[int]
    kind: PointerAction.Kind
    display_id: str
    x: float
    y: float
    button: PointerAction.Button
    def __init__(self, kind: _Optional[_Union[PointerAction.Kind, str]] = ..., display_id: _Optional[str] = ..., x: _Optional[float] = ..., y: _Optional[float] = ..., button: _Optional[_Union[PointerAction.Button, str]] = ...) -> None: ...

class KeyAction(_message.Message):
    __slots__ = ("kind", "key")
    class Kind(int, metaclass=_enum_type_wrapper.EnumTypeWrapper):
        __slots__ = ()
        KIND_UNSPECIFIED: _ClassVar[KeyAction.Kind]
        KIND_DOWN: _ClassVar[KeyAction.Kind]
        KIND_UP: _ClassVar[KeyAction.Kind]
        KIND_PRESS: _ClassVar[KeyAction.Kind]
    KIND_UNSPECIFIED: KeyAction.Kind
    KIND_DOWN: KeyAction.Kind
    KIND_UP: KeyAction.Kind
    KIND_PRESS: KeyAction.Kind
    KIND_FIELD_NUMBER: _ClassVar[int]
    KEY_FIELD_NUMBER: _ClassVar[int]
    kind: KeyAction.Kind
    key: str
    def __init__(self, kind: _Optional[_Union[KeyAction.Kind, str]] = ..., key: _Optional[str] = ...) -> None: ...

class TextAction(_message.Message):
    __slots__ = ("text", "element_id", "observation_revision", "position")
    TEXT_FIELD_NUMBER: _ClassVar[int]
    ELEMENT_ID_FIELD_NUMBER: _ClassVar[int]
    OBSERVATION_REVISION_FIELD_NUMBER: _ClassVar[int]
    POSITION_FIELD_NUMBER: _ClassVar[int]
    text: str
    element_id: str
    observation_revision: str
    position: int
    def __init__(self, text: _Optional[str] = ..., element_id: _Optional[str] = ..., observation_revision: _Optional[str] = ..., position: _Optional[int] = ...) -> None: ...

class AssertTextAction(_message.Message):
    __slots__ = ("expected_text", "element_id", "observation_revision")
    EXPECTED_TEXT_FIELD_NUMBER: _ClassVar[int]
    ELEMENT_ID_FIELD_NUMBER: _ClassVar[int]
    OBSERVATION_REVISION_FIELD_NUMBER: _ClassVar[int]
    expected_text: str
    element_id: str
    observation_revision: str
    def __init__(self, expected_text: _Optional[str] = ..., element_id: _Optional[str] = ..., observation_revision: _Optional[str] = ...) -> None: ...

class InvokeAction(_message.Message):
    __slots__ = ("element_id", "observation_revision", "action_name")
    ELEMENT_ID_FIELD_NUMBER: _ClassVar[int]
    OBSERVATION_REVISION_FIELD_NUMBER: _ClassVar[int]
    ACTION_NAME_FIELD_NUMBER: _ClassVar[int]
    element_id: str
    observation_revision: str
    action_name: str
    def __init__(self, element_id: _Optional[str] = ..., observation_revision: _Optional[str] = ..., action_name: _Optional[str] = ...) -> None: ...

class SemanticElement(_message.Message):
    __slots__ = ("element_id", "name", "editable", "parent_id", "window_id", "role", "label", "states", "x", "y", "width", "height", "supported_actions", "bounds_known", "state_known", "fingerprint")
    ELEMENT_ID_FIELD_NUMBER: _ClassVar[int]
    NAME_FIELD_NUMBER: _ClassVar[int]
    EDITABLE_FIELD_NUMBER: _ClassVar[int]
    PARENT_ID_FIELD_NUMBER: _ClassVar[int]
    WINDOW_ID_FIELD_NUMBER: _ClassVar[int]
    ROLE_FIELD_NUMBER: _ClassVar[int]
    LABEL_FIELD_NUMBER: _ClassVar[int]
    STATES_FIELD_NUMBER: _ClassVar[int]
    X_FIELD_NUMBER: _ClassVar[int]
    Y_FIELD_NUMBER: _ClassVar[int]
    WIDTH_FIELD_NUMBER: _ClassVar[int]
    HEIGHT_FIELD_NUMBER: _ClassVar[int]
    SUPPORTED_ACTIONS_FIELD_NUMBER: _ClassVar[int]
    BOUNDS_KNOWN_FIELD_NUMBER: _ClassVar[int]
    STATE_KNOWN_FIELD_NUMBER: _ClassVar[int]
    FINGERPRINT_FIELD_NUMBER: _ClassVar[int]
    element_id: str
    name: str
    editable: bool
    parent_id: str
    window_id: str
    role: int
    label: str
    states: _containers.RepeatedScalarFieldContainer[str]
    x: int
    y: int
    width: int
    height: int
    supported_actions: _containers.RepeatedScalarFieldContainer[str]
    bounds_known: bool
    state_known: bool
    fingerprint: str
    def __init__(self, element_id: _Optional[str] = ..., name: _Optional[str] = ..., editable: _Optional[bool] = ..., parent_id: _Optional[str] = ..., window_id: _Optional[str] = ..., role: _Optional[int] = ..., label: _Optional[str] = ..., states: _Optional[_Iterable[str]] = ..., x: _Optional[int] = ..., y: _Optional[int] = ..., width: _Optional[int] = ..., height: _Optional[int] = ..., supported_actions: _Optional[_Iterable[str]] = ..., bounds_known: _Optional[bool] = ..., state_known: _Optional[bool] = ..., fingerprint: _Optional[str] = ...) -> None: ...

class SemanticObservation(_message.Message):
    __slots__ = ("revision", "expires_at", "process_id", "elements", "refresh_epoch")
    REVISION_FIELD_NUMBER: _ClassVar[int]
    EXPIRES_AT_FIELD_NUMBER: _ClassVar[int]
    PROCESS_ID_FIELD_NUMBER: _ClassVar[int]
    ELEMENTS_FIELD_NUMBER: _ClassVar[int]
    REFRESH_EPOCH_FIELD_NUMBER: _ClassVar[int]
    revision: str
    expires_at: _timestamp_pb2.Timestamp
    process_id: int
    elements: _containers.RepeatedCompositeFieldContainer[SemanticElement]
    refresh_epoch: int
    def __init__(self, revision: _Optional[str] = ..., expires_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., process_id: _Optional[int] = ..., elements: _Optional[_Iterable[_Union[SemanticElement, _Mapping]]] = ..., refresh_epoch: _Optional[int] = ...) -> None: ...

class WheelAction(_message.Message):
    __slots__ = ("display_id", "x", "y", "horizontal_ticks", "vertical_ticks")
    DISPLAY_ID_FIELD_NUMBER: _ClassVar[int]
    X_FIELD_NUMBER: _ClassVar[int]
    Y_FIELD_NUMBER: _ClassVar[int]
    HORIZONTAL_TICKS_FIELD_NUMBER: _ClassVar[int]
    VERTICAL_TICKS_FIELD_NUMBER: _ClassVar[int]
    display_id: str
    x: float
    y: float
    horizontal_ticks: int
    vertical_ticks: int
    def __init__(self, display_id: _Optional[str] = ..., x: _Optional[float] = ..., y: _Optional[float] = ..., horizontal_ticks: _Optional[int] = ..., vertical_ticks: _Optional[int] = ...) -> None: ...

class Action(_message.Message):
    __slots__ = ("pointer", "key", "text", "wheel", "assert_text", "invoke")
    POINTER_FIELD_NUMBER: _ClassVar[int]
    KEY_FIELD_NUMBER: _ClassVar[int]
    TEXT_FIELD_NUMBER: _ClassVar[int]
    WHEEL_FIELD_NUMBER: _ClassVar[int]
    ASSERT_TEXT_FIELD_NUMBER: _ClassVar[int]
    INVOKE_FIELD_NUMBER: _ClassVar[int]
    pointer: PointerAction
    key: KeyAction
    text: TextAction
    wheel: WheelAction
    assert_text: AssertTextAction
    invoke: InvokeAction
    def __init__(self, pointer: _Optional[_Union[PointerAction, _Mapping]] = ..., key: _Optional[_Union[KeyAction, _Mapping]] = ..., text: _Optional[_Union[TextAction, _Mapping]] = ..., wheel: _Optional[_Union[WheelAction, _Mapping]] = ..., assert_text: _Optional[_Union[AssertTextAction, _Mapping]] = ..., invoke: _Optional[_Union[InvokeAction, _Mapping]] = ...) -> None: ...

class ActRequest(_message.Message):
    __slots__ = ("lease", "command_id", "geometry_revision", "action")
    LEASE_FIELD_NUMBER: _ClassVar[int]
    COMMAND_ID_FIELD_NUMBER: _ClassVar[int]
    GEOMETRY_REVISION_FIELD_NUMBER: _ClassVar[int]
    ACTION_FIELD_NUMBER: _ClassVar[int]
    lease: Lease
    command_id: str
    geometry_revision: str
    action: Action
    def __init__(self, lease: _Optional[_Union[Lease, _Mapping]] = ..., command_id: _Optional[str] = ..., geometry_revision: _Optional[str] = ..., action: _Optional[_Union[Action, _Mapping]] = ...) -> None: ...

class Receipt(_message.Message):
    __slots__ = ("command_id", "digest", "outcome")
    COMMAND_ID_FIELD_NUMBER: _ClassVar[int]
    DIGEST_FIELD_NUMBER: _ClassVar[int]
    OUTCOME_FIELD_NUMBER: _ClassVar[int]
    command_id: str
    digest: str
    outcome: str
    def __init__(self, command_id: _Optional[str] = ..., digest: _Optional[str] = ..., outcome: _Optional[str] = ...) -> None: ...

class ActResponse(_message.Message):
    __slots__ = ("receipt",)
    RECEIPT_FIELD_NUMBER: _ClassVar[int]
    receipt: Receipt
    def __init__(self, receipt: _Optional[_Union[Receipt, _Mapping]] = ...) -> None: ...

class StopRequest(_message.Message):
    __slots__ = ("lease",)
    LEASE_FIELD_NUMBER: _ClassVar[int]
    lease: Lease
    def __init__(self, lease: _Optional[_Union[Lease, _Mapping]] = ...) -> None: ...

class StopResponse(_message.Message):
    __slots__ = ()
    def __init__(self) -> None: ...

class ObserveRequest(_message.Message):
    __slots__ = ("lease", "process_id", "application_id", "application_revision")
    LEASE_FIELD_NUMBER: _ClassVar[int]
    PROCESS_ID_FIELD_NUMBER: _ClassVar[int]
    APPLICATION_ID_FIELD_NUMBER: _ClassVar[int]
    APPLICATION_REVISION_FIELD_NUMBER: _ClassVar[int]
    lease: Lease
    process_id: int
    application_id: str
    application_revision: str
    def __init__(self, lease: _Optional[_Union[Lease, _Mapping]] = ..., process_id: _Optional[int] = ..., application_id: _Optional[str] = ..., application_revision: _Optional[str] = ...) -> None: ...

class ObserveResponse(_message.Message):
    __slots__ = ("png", "display_id", "geometry_revision", "captured_at", "width", "height", "semantic")
    PNG_FIELD_NUMBER: _ClassVar[int]
    DISPLAY_ID_FIELD_NUMBER: _ClassVar[int]
    GEOMETRY_REVISION_FIELD_NUMBER: _ClassVar[int]
    CAPTURED_AT_FIELD_NUMBER: _ClassVar[int]
    WIDTH_FIELD_NUMBER: _ClassVar[int]
    HEIGHT_FIELD_NUMBER: _ClassVar[int]
    SEMANTIC_FIELD_NUMBER: _ClassVar[int]
    png: bytes
    display_id: str
    geometry_revision: str
    captured_at: _timestamp_pb2.Timestamp
    width: int
    height: int
    semantic: SemanticObservation
    def __init__(self, png: _Optional[bytes] = ..., display_id: _Optional[str] = ..., geometry_revision: _Optional[str] = ..., captured_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., width: _Optional[int] = ..., height: _Optional[int] = ..., semantic: _Optional[_Union[SemanticObservation, _Mapping]] = ...) -> None: ...

class Application(_message.Message):
    __slots__ = ("application_id", "name", "process_id")
    APPLICATION_ID_FIELD_NUMBER: _ClassVar[int]
    NAME_FIELD_NUMBER: _ClassVar[int]
    PROCESS_ID_FIELD_NUMBER: _ClassVar[int]
    application_id: str
    name: str
    process_id: int
    def __init__(self, application_id: _Optional[str] = ..., name: _Optional[str] = ..., process_id: _Optional[int] = ...) -> None: ...

class ApplicationsRequest(_message.Message):
    __slots__ = ("lease",)
    LEASE_FIELD_NUMBER: _ClassVar[int]
    lease: Lease
    def __init__(self, lease: _Optional[_Union[Lease, _Mapping]] = ...) -> None: ...

class OwnerApplicationsRequest(_message.Message):
    __slots__ = ("session",)
    SESSION_FIELD_NUMBER: _ClassVar[int]
    session: _surface_pb2.SessionRef
    def __init__(self, session: _Optional[_Union[_surface_pb2.SessionRef, _Mapping]] = ...) -> None: ...

class ApplicationsResponse(_message.Message):
    __slots__ = ("revision", "expires_at", "applications")
    REVISION_FIELD_NUMBER: _ClassVar[int]
    EXPIRES_AT_FIELD_NUMBER: _ClassVar[int]
    APPLICATIONS_FIELD_NUMBER: _ClassVar[int]
    revision: str
    expires_at: _timestamp_pb2.Timestamp
    applications: _containers.RepeatedCompositeFieldContainer[Application]
    def __init__(self, revision: _Optional[str] = ..., expires_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., applications: _Optional[_Iterable[_Union[Application, _Mapping]]] = ...) -> None: ...

class SemanticSelector(_message.Message):
    __slots__ = ("observation_revision", "window_id", "name", "editable_only", "match_mode", "role", "refresh_epoch", "allow_hidden", "allow_disabled", "allow_offscreen")
    OBSERVATION_REVISION_FIELD_NUMBER: _ClassVar[int]
    WINDOW_ID_FIELD_NUMBER: _ClassVar[int]
    NAME_FIELD_NUMBER: _ClassVar[int]
    EDITABLE_ONLY_FIELD_NUMBER: _ClassVar[int]
    MATCH_MODE_FIELD_NUMBER: _ClassVar[int]
    ROLE_FIELD_NUMBER: _ClassVar[int]
    REFRESH_EPOCH_FIELD_NUMBER: _ClassVar[int]
    ALLOW_HIDDEN_FIELD_NUMBER: _ClassVar[int]
    ALLOW_DISABLED_FIELD_NUMBER: _ClassVar[int]
    ALLOW_OFFSCREEN_FIELD_NUMBER: _ClassVar[int]
    observation_revision: str
    window_id: str
    name: str
    editable_only: bool
    match_mode: SemanticMatchMode
    role: int
    refresh_epoch: int
    allow_hidden: bool
    allow_disabled: bool
    allow_offscreen: bool
    def __init__(self, observation_revision: _Optional[str] = ..., window_id: _Optional[str] = ..., name: _Optional[str] = ..., editable_only: _Optional[bool] = ..., match_mode: _Optional[_Union[SemanticMatchMode, str]] = ..., role: _Optional[int] = ..., refresh_epoch: _Optional[int] = ..., allow_hidden: _Optional[bool] = ..., allow_disabled: _Optional[bool] = ..., allow_offscreen: _Optional[bool] = ...) -> None: ...

class ResolveRequest(_message.Message):
    __slots__ = ("lease", "selector")
    LEASE_FIELD_NUMBER: _ClassVar[int]
    SELECTOR_FIELD_NUMBER: _ClassVar[int]
    lease: Lease
    selector: SemanticSelector
    def __init__(self, lease: _Optional[_Union[Lease, _Mapping]] = ..., selector: _Optional[_Union[SemanticSelector, _Mapping]] = ...) -> None: ...

class OwnerResolveRequest(_message.Message):
    __slots__ = ("session", "selector")
    SESSION_FIELD_NUMBER: _ClassVar[int]
    SELECTOR_FIELD_NUMBER: _ClassVar[int]
    session: _surface_pb2.SessionRef
    selector: SemanticSelector
    def __init__(self, session: _Optional[_Union[_surface_pb2.SessionRef, _Mapping]] = ..., selector: _Optional[_Union[SemanticSelector, _Mapping]] = ...) -> None: ...

class ResolveResponse(_message.Message):
    __slots__ = ("disposition", "observation_revision", "geometry_revision", "expires_at", "element_ids")
    class Disposition(int, metaclass=_enum_type_wrapper.EnumTypeWrapper):
        __slots__ = ()
        DISPOSITION_UNSPECIFIED: _ClassVar[ResolveResponse.Disposition]
        DISPOSITION_ABSENT: _ClassVar[ResolveResponse.Disposition]
        DISPOSITION_UNIQUE: _ClassVar[ResolveResponse.Disposition]
        DISPOSITION_AMBIGUOUS: _ClassVar[ResolveResponse.Disposition]
    DISPOSITION_UNSPECIFIED: ResolveResponse.Disposition
    DISPOSITION_ABSENT: ResolveResponse.Disposition
    DISPOSITION_UNIQUE: ResolveResponse.Disposition
    DISPOSITION_AMBIGUOUS: ResolveResponse.Disposition
    DISPOSITION_FIELD_NUMBER: _ClassVar[int]
    OBSERVATION_REVISION_FIELD_NUMBER: _ClassVar[int]
    GEOMETRY_REVISION_FIELD_NUMBER: _ClassVar[int]
    EXPIRES_AT_FIELD_NUMBER: _ClassVar[int]
    ELEMENT_IDS_FIELD_NUMBER: _ClassVar[int]
    disposition: ResolveResponse.Disposition
    observation_revision: str
    geometry_revision: str
    expires_at: _timestamp_pb2.Timestamp
    element_ids: _containers.RepeatedScalarFieldContainer[str]
    def __init__(self, disposition: _Optional[_Union[ResolveResponse.Disposition, str]] = ..., observation_revision: _Optional[str] = ..., geometry_revision: _Optional[str] = ..., expires_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., element_ids: _Optional[_Iterable[str]] = ...) -> None: ...

class ClaimFlowRequest(_message.Message):
    __slots__ = ("lease", "run_id", "digest", "steps")
    LEASE_FIELD_NUMBER: _ClassVar[int]
    RUN_ID_FIELD_NUMBER: _ClassVar[int]
    DIGEST_FIELD_NUMBER: _ClassVar[int]
    STEPS_FIELD_NUMBER: _ClassVar[int]
    lease: Lease
    run_id: str
    digest: str
    steps: int
    def __init__(self, lease: _Optional[_Union[Lease, _Mapping]] = ..., run_id: _Optional[str] = ..., digest: _Optional[str] = ..., steps: _Optional[int] = ...) -> None: ...

class ClaimFlowResponse(_message.Message):
    __slots__ = ("record", "fresh")
    RECORD_FIELD_NUMBER: _ClassVar[int]
    FRESH_FIELD_NUMBER: _ClassVar[int]
    record: FlowRecord
    fresh: bool
    def __init__(self, record: _Optional[_Union[FlowRecord, _Mapping]] = ..., fresh: _Optional[bool] = ...) -> None: ...

class FinishFlowRequest(_message.Message):
    __slots__ = ("lease", "run_id", "digest", "disposition")
    LEASE_FIELD_NUMBER: _ClassVar[int]
    RUN_ID_FIELD_NUMBER: _ClassVar[int]
    DIGEST_FIELD_NUMBER: _ClassVar[int]
    DISPOSITION_FIELD_NUMBER: _ClassVar[int]
    lease: Lease
    run_id: str
    digest: str
    disposition: str
    def __init__(self, lease: _Optional[_Union[Lease, _Mapping]] = ..., run_id: _Optional[str] = ..., digest: _Optional[str] = ..., disposition: _Optional[str] = ...) -> None: ...

class FlowRecord(_message.Message):
    __slots__ = ("run_id", "digest", "steps", "disposition", "confirmed", "claimed_at", "finished_at")
    RUN_ID_FIELD_NUMBER: _ClassVar[int]
    DIGEST_FIELD_NUMBER: _ClassVar[int]
    STEPS_FIELD_NUMBER: _ClassVar[int]
    DISPOSITION_FIELD_NUMBER: _ClassVar[int]
    CONFIRMED_FIELD_NUMBER: _ClassVar[int]
    CLAIMED_AT_FIELD_NUMBER: _ClassVar[int]
    FINISHED_AT_FIELD_NUMBER: _ClassVar[int]
    run_id: str
    digest: str
    steps: int
    disposition: str
    confirmed: int
    claimed_at: _timestamp_pb2.Timestamp
    finished_at: _timestamp_pb2.Timestamp
    def __init__(self, run_id: _Optional[str] = ..., digest: _Optional[str] = ..., steps: _Optional[int] = ..., disposition: _Optional[str] = ..., confirmed: _Optional[int] = ..., claimed_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., finished_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ...) -> None: ...

class OwnerRunFlowRequest(_message.Message):
    __slots__ = ("session", "run_id", "application_id", "application_revision", "flow")
    SESSION_FIELD_NUMBER: _ClassVar[int]
    RUN_ID_FIELD_NUMBER: _ClassVar[int]
    APPLICATION_ID_FIELD_NUMBER: _ClassVar[int]
    APPLICATION_REVISION_FIELD_NUMBER: _ClassVar[int]
    FLOW_FIELD_NUMBER: _ClassVar[int]
    session: _surface_pb2.SessionRef
    run_id: str
    application_id: str
    application_revision: str
    flow: _flow_pb2.Flow
    def __init__(self, session: _Optional[_Union[_surface_pb2.SessionRef, _Mapping]] = ..., run_id: _Optional[str] = ..., application_id: _Optional[str] = ..., application_revision: _Optional[str] = ..., flow: _Optional[_Union[_flow_pb2.Flow, _Mapping]] = ...) -> None: ...

class OwnerPromoteFlowRequest(_message.Message):
    __slots__ = ("session", "source_session", "source_run_id", "context_key", "id", "expected_version")
    SESSION_FIELD_NUMBER: _ClassVar[int]
    SOURCE_SESSION_FIELD_NUMBER: _ClassVar[int]
    SOURCE_RUN_ID_FIELD_NUMBER: _ClassVar[int]
    CONTEXT_KEY_FIELD_NUMBER: _ClassVar[int]
    ID_FIELD_NUMBER: _ClassVar[int]
    EXPECTED_VERSION_FIELD_NUMBER: _ClassVar[int]
    session: _surface_pb2.SessionRef
    source_session: _surface_pb2.SessionRef
    source_run_id: str
    context_key: str
    id: str
    expected_version: int
    def __init__(self, session: _Optional[_Union[_surface_pb2.SessionRef, _Mapping]] = ..., source_session: _Optional[_Union[_surface_pb2.SessionRef, _Mapping]] = ..., source_run_id: _Optional[str] = ..., context_key: _Optional[str] = ..., id: _Optional[str] = ..., expected_version: _Optional[int] = ...) -> None: ...

class OwnerGetSavedFlowRequest(_message.Message):
    __slots__ = ("session", "id", "version", "context_key")
    SESSION_FIELD_NUMBER: _ClassVar[int]
    ID_FIELD_NUMBER: _ClassVar[int]
    VERSION_FIELD_NUMBER: _ClassVar[int]
    CONTEXT_KEY_FIELD_NUMBER: _ClassVar[int]
    session: _surface_pb2.SessionRef
    id: str
    version: int
    context_key: str
    def __init__(self, session: _Optional[_Union[_surface_pb2.SessionRef, _Mapping]] = ..., id: _Optional[str] = ..., version: _Optional[int] = ..., context_key: _Optional[str] = ...) -> None: ...

class OwnerRunSavedFlowRequest(_message.Message):
    __slots__ = ("session", "id", "version", "context_key", "run_id", "application_id", "application_revision")
    SESSION_FIELD_NUMBER: _ClassVar[int]
    ID_FIELD_NUMBER: _ClassVar[int]
    VERSION_FIELD_NUMBER: _ClassVar[int]
    CONTEXT_KEY_FIELD_NUMBER: _ClassVar[int]
    RUN_ID_FIELD_NUMBER: _ClassVar[int]
    APPLICATION_ID_FIELD_NUMBER: _ClassVar[int]
    APPLICATION_REVISION_FIELD_NUMBER: _ClassVar[int]
    session: _surface_pb2.SessionRef
    id: str
    version: int
    context_key: str
    run_id: str
    application_id: str
    application_revision: str
    def __init__(self, session: _Optional[_Union[_surface_pb2.SessionRef, _Mapping]] = ..., id: _Optional[str] = ..., version: _Optional[int] = ..., context_key: _Optional[str] = ..., run_id: _Optional[str] = ..., application_id: _Optional[str] = ..., application_revision: _Optional[str] = ...) -> None: ...

class SavedDesktopFlow(_message.Message):
    __slots__ = ("id", "version", "context_key", "flow", "source_run_id", "source_digest", "created_at")
    ID_FIELD_NUMBER: _ClassVar[int]
    VERSION_FIELD_NUMBER: _ClassVar[int]
    CONTEXT_KEY_FIELD_NUMBER: _ClassVar[int]
    FLOW_FIELD_NUMBER: _ClassVar[int]
    SOURCE_RUN_ID_FIELD_NUMBER: _ClassVar[int]
    SOURCE_DIGEST_FIELD_NUMBER: _ClassVar[int]
    CREATED_AT_FIELD_NUMBER: _ClassVar[int]
    id: str
    version: int
    context_key: str
    flow: _flow_pb2.Flow
    source_run_id: str
    source_digest: str
    created_at: str
    def __init__(self, id: _Optional[str] = ..., version: _Optional[int] = ..., context_key: _Optional[str] = ..., flow: _Optional[_Union[_flow_pb2.Flow, _Mapping]] = ..., source_run_id: _Optional[str] = ..., source_digest: _Optional[str] = ..., created_at: _Optional[str] = ...) -> None: ...

class CleanupResponse(_message.Message):
    __slots__ = ("lease", "released", "observed_at")
    LEASE_FIELD_NUMBER: _ClassVar[int]
    RELEASED_FIELD_NUMBER: _ClassVar[int]
    OBSERVED_AT_FIELD_NUMBER: _ClassVar[int]
    lease: Lease
    released: bool
    observed_at: _timestamp_pb2.Timestamp
    def __init__(self, lease: _Optional[_Union[Lease, _Mapping]] = ..., released: _Optional[bool] = ..., observed_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ...) -> None: ...

class OwnerCleanupResponse(_message.Message):
    __slots__ = ("session", "released", "observed_at")
    SESSION_FIELD_NUMBER: _ClassVar[int]
    RELEASED_FIELD_NUMBER: _ClassVar[int]
    OBSERVED_AT_FIELD_NUMBER: _ClassVar[int]
    session: _surface_pb2.SessionRef
    released: bool
    observed_at: _timestamp_pb2.Timestamp
    def __init__(self, session: _Optional[_Union[_surface_pb2.SessionRef, _Mapping]] = ..., released: _Optional[bool] = ..., observed_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ...) -> None: ...

class OwnerListAdmissionsRequest(_message.Message):
    __slots__ = ("page_token", "page_size")
    PAGE_TOKEN_FIELD_NUMBER: _ClassVar[int]
    PAGE_SIZE_FIELD_NUMBER: _ClassVar[int]
    page_token: str
    page_size: int
    def __init__(self, page_token: _Optional[str] = ..., page_size: _Optional[int] = ...) -> None: ...

class OwnerAdmission(_message.Message):
    __slots__ = ("session", "expires_at", "control")
    SESSION_FIELD_NUMBER: _ClassVar[int]
    EXPIRES_AT_FIELD_NUMBER: _ClassVar[int]
    CONTROL_FIELD_NUMBER: _ClassVar[int]
    session: _surface_pb2.SessionRef
    expires_at: _timestamp_pb2.Timestamp
    control: bool
    def __init__(self, session: _Optional[_Union[_surface_pb2.SessionRef, _Mapping]] = ..., expires_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., control: _Optional[bool] = ...) -> None: ...

class OwnerListAdmissionsResponse(_message.Message):
    __slots__ = ("admissions", "next_page_token")
    ADMISSIONS_FIELD_NUMBER: _ClassVar[int]
    NEXT_PAGE_TOKEN_FIELD_NUMBER: _ClassVar[int]
    admissions: _containers.RepeatedCompositeFieldContainer[OwnerAdmission]
    next_page_token: str
    def __init__(self, admissions: _Optional[_Iterable[_Union[OwnerAdmission, _Mapping]]] = ..., next_page_token: _Optional[str] = ...) -> None: ...

class OwnerOpenDisposition(_message.Message):
    __slots__ = ("request_id", "state", "session", "expires_at", "control")
    REQUEST_ID_FIELD_NUMBER: _ClassVar[int]
    STATE_FIELD_NUMBER: _ClassVar[int]
    SESSION_FIELD_NUMBER: _ClassVar[int]
    EXPIRES_AT_FIELD_NUMBER: _ClassVar[int]
    CONTROL_FIELD_NUMBER: _ClassVar[int]
    request_id: str
    state: str
    session: _surface_pb2.SessionRef
    expires_at: _timestamp_pb2.Timestamp
    control: bool
    def __init__(self, request_id: _Optional[str] = ..., state: _Optional[str] = ..., session: _Optional[_Union[_surface_pb2.SessionRef, _Mapping]] = ..., expires_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., control: _Optional[bool] = ...) -> None: ...

class DesktopBounds(_message.Message):
    __slots__ = ("x", "y", "width", "height")
    X_FIELD_NUMBER: _ClassVar[int]
    Y_FIELD_NUMBER: _ClassVar[int]
    WIDTH_FIELD_NUMBER: _ClassVar[int]
    HEIGHT_FIELD_NUMBER: _ClassVar[int]
    x: int
    y: int
    width: int
    height: int
    def __init__(self, x: _Optional[int] = ..., y: _Optional[int] = ..., width: _Optional[int] = ..., height: _Optional[int] = ...) -> None: ...

class ActivationImage(_message.Message):
    __slots__ = ("reference", "png")
    REFERENCE_FIELD_NUMBER: _ClassVar[int]
    PNG_FIELD_NUMBER: _ClassVar[int]
    reference: ActivationReference
    png: bytes
    def __init__(self, reference: _Optional[_Union[ActivationReference, _Mapping]] = ..., png: _Optional[bytes] = ...) -> None: ...

class ActivationReference(_message.Message):
    __slots__ = ("has_image", "source_bounds", "context_id", "display_id", "geometry_revision", "pointer_x", "pointer_y", "captured_at", "expires_at")
    HAS_IMAGE_FIELD_NUMBER: _ClassVar[int]
    SOURCE_BOUNDS_FIELD_NUMBER: _ClassVar[int]
    CONTEXT_ID_FIELD_NUMBER: _ClassVar[int]
    DISPLAY_ID_FIELD_NUMBER: _ClassVar[int]
    GEOMETRY_REVISION_FIELD_NUMBER: _ClassVar[int]
    POINTER_X_FIELD_NUMBER: _ClassVar[int]
    POINTER_Y_FIELD_NUMBER: _ClassVar[int]
    CAPTURED_AT_FIELD_NUMBER: _ClassVar[int]
    EXPIRES_AT_FIELD_NUMBER: _ClassVar[int]
    has_image: bool
    source_bounds: DesktopBounds
    context_id: str
    display_id: str
    geometry_revision: str
    pointer_x: int
    pointer_y: int
    captured_at: _timestamp_pb2.Timestamp
    expires_at: _timestamp_pb2.Timestamp
    def __init__(self, has_image: _Optional[bool] = ..., source_bounds: _Optional[_Union[DesktopBounds, _Mapping]] = ..., context_id: _Optional[str] = ..., display_id: _Optional[str] = ..., geometry_revision: _Optional[str] = ..., pointer_x: _Optional[int] = ..., pointer_y: _Optional[int] = ..., captured_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., expires_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ...) -> None: ...

class ReadActivationRequest(_message.Message):
    __slots__ = ("lease", "context_id")
    LEASE_FIELD_NUMBER: _ClassVar[int]
    CONTEXT_ID_FIELD_NUMBER: _ClassVar[int]
    lease: Lease
    context_id: str
    def __init__(self, lease: _Optional[_Union[Lease, _Mapping]] = ..., context_id: _Optional[str] = ...) -> None: ...

class OwnerReadActivationRequest(_message.Message):
    __slots__ = ("session", "context_id")
    SESSION_FIELD_NUMBER: _ClassVar[int]
    CONTEXT_ID_FIELD_NUMBER: _ClassVar[int]
    session: _surface_pb2.SessionRef
    context_id: str
    def __init__(self, session: _Optional[_Union[_surface_pb2.SessionRef, _Mapping]] = ..., context_id: _Optional[str] = ...) -> None: ...

class CompanionActivationRequest(_message.Message):
    __slots__ = ("lease", "companion_window", "companion_pid", "include_image")
    LEASE_FIELD_NUMBER: _ClassVar[int]
    COMPANION_WINDOW_FIELD_NUMBER: _ClassVar[int]
    COMPANION_PID_FIELD_NUMBER: _ClassVar[int]
    INCLUDE_IMAGE_FIELD_NUMBER: _ClassVar[int]
    lease: Lease
    companion_window: int
    companion_pid: int
    include_image: bool
    def __init__(self, lease: _Optional[_Union[Lease, _Mapping]] = ..., companion_window: _Optional[int] = ..., companion_pid: _Optional[int] = ..., include_image: _Optional[bool] = ...) -> None: ...

class OwnerCompanionActivationRequest(_message.Message):
    __slots__ = ("session", "companion_window", "include_image")
    SESSION_FIELD_NUMBER: _ClassVar[int]
    COMPANION_WINDOW_FIELD_NUMBER: _ClassVar[int]
    INCLUDE_IMAGE_FIELD_NUMBER: _ClassVar[int]
    session: _surface_pb2.SessionRef
    companion_window: int
    include_image: bool
    def __init__(self, session: _Optional[_Union[_surface_pb2.SessionRef, _Mapping]] = ..., companion_window: _Optional[int] = ..., include_image: _Optional[bool] = ...) -> None: ...
