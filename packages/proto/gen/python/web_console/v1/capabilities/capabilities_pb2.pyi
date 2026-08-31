from google.protobuf.internal import containers as _containers
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Iterable as _Iterable, Mapping as _Mapping
from typing import ClassVar as _ClassVar, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class CapabilityState(_message.Message):
    __slots__ = ("id", "name", "description", "dependency_kind", "dependency_slug", "features", "status", "message", "checked_at", "reason_code", "action_kind", "action_label", "operator_command", "feature_status", "feature_reason", "feature_operator_command", "provider_status", "provider_features")
    class FeatureStatusEntry(_message.Message):
        __slots__ = ("key", "value")
        KEY_FIELD_NUMBER: _ClassVar[int]
        VALUE_FIELD_NUMBER: _ClassVar[int]
        key: str
        value: str
        def __init__(self, key: _Optional[str] = ..., value: _Optional[str] = ...) -> None: ...
    class FeatureReasonEntry(_message.Message):
        __slots__ = ("key", "value")
        KEY_FIELD_NUMBER: _ClassVar[int]
        VALUE_FIELD_NUMBER: _ClassVar[int]
        key: str
        value: str
        def __init__(self, key: _Optional[str] = ..., value: _Optional[str] = ...) -> None: ...
    class FeatureOperatorCommandEntry(_message.Message):
        __slots__ = ("key", "value")
        KEY_FIELD_NUMBER: _ClassVar[int]
        VALUE_FIELD_NUMBER: _ClassVar[int]
        key: str
        value: str
        def __init__(self, key: _Optional[str] = ..., value: _Optional[str] = ...) -> None: ...
    class ProviderStatusEntry(_message.Message):
        __slots__ = ("key", "value")
        KEY_FIELD_NUMBER: _ClassVar[int]
        VALUE_FIELD_NUMBER: _ClassVar[int]
        key: str
        value: str
        def __init__(self, key: _Optional[str] = ..., value: _Optional[str] = ...) -> None: ...
    class ProviderFeaturesEntry(_message.Message):
        __slots__ = ("key", "value")
        KEY_FIELD_NUMBER: _ClassVar[int]
        VALUE_FIELD_NUMBER: _ClassVar[int]
        key: str
        value: str
        def __init__(self, key: _Optional[str] = ..., value: _Optional[str] = ...) -> None: ...
    ID_FIELD_NUMBER: _ClassVar[int]
    NAME_FIELD_NUMBER: _ClassVar[int]
    DESCRIPTION_FIELD_NUMBER: _ClassVar[int]
    DEPENDENCY_KIND_FIELD_NUMBER: _ClassVar[int]
    DEPENDENCY_SLUG_FIELD_NUMBER: _ClassVar[int]
    FEATURES_FIELD_NUMBER: _ClassVar[int]
    STATUS_FIELD_NUMBER: _ClassVar[int]
    MESSAGE_FIELD_NUMBER: _ClassVar[int]
    CHECKED_AT_FIELD_NUMBER: _ClassVar[int]
    REASON_CODE_FIELD_NUMBER: _ClassVar[int]
    ACTION_KIND_FIELD_NUMBER: _ClassVar[int]
    ACTION_LABEL_FIELD_NUMBER: _ClassVar[int]
    OPERATOR_COMMAND_FIELD_NUMBER: _ClassVar[int]
    FEATURE_STATUS_FIELD_NUMBER: _ClassVar[int]
    FEATURE_REASON_FIELD_NUMBER: _ClassVar[int]
    FEATURE_OPERATOR_COMMAND_FIELD_NUMBER: _ClassVar[int]
    PROVIDER_STATUS_FIELD_NUMBER: _ClassVar[int]
    PROVIDER_FEATURES_FIELD_NUMBER: _ClassVar[int]
    id: str
    name: str
    description: str
    dependency_kind: str
    dependency_slug: str
    features: _containers.RepeatedScalarFieldContainer[str]
    status: str
    message: str
    checked_at: str
    reason_code: str
    action_kind: str
    action_label: str
    operator_command: str
    feature_status: _containers.ScalarMap[str, str]
    feature_reason: _containers.ScalarMap[str, str]
    feature_operator_command: _containers.ScalarMap[str, str]
    provider_status: _containers.ScalarMap[str, str]
    provider_features: _containers.ScalarMap[str, str]
    def __init__(self, id: _Optional[str] = ..., name: _Optional[str] = ..., description: _Optional[str] = ..., dependency_kind: _Optional[str] = ..., dependency_slug: _Optional[str] = ..., features: _Optional[_Iterable[str]] = ..., status: _Optional[str] = ..., message: _Optional[str] = ..., checked_at: _Optional[str] = ..., reason_code: _Optional[str] = ..., action_kind: _Optional[str] = ..., action_label: _Optional[str] = ..., operator_command: _Optional[str] = ..., feature_status: _Optional[_Mapping[str, str]] = ..., feature_reason: _Optional[_Mapping[str, str]] = ..., feature_operator_command: _Optional[_Mapping[str, str]] = ..., provider_status: _Optional[_Mapping[str, str]] = ..., provider_features: _Optional[_Mapping[str, str]] = ...) -> None: ...

class BackendOption(_message.Message):
    __slots__ = ("id", "display_name", "description", "survives_restart", "available", "reason")
    ID_FIELD_NUMBER: _ClassVar[int]
    DISPLAY_NAME_FIELD_NUMBER: _ClassVar[int]
    DESCRIPTION_FIELD_NUMBER: _ClassVar[int]
    SURVIVES_RESTART_FIELD_NUMBER: _ClassVar[int]
    AVAILABLE_FIELD_NUMBER: _ClassVar[int]
    REASON_FIELD_NUMBER: _ClassVar[int]
    id: str
    display_name: str
    description: str
    survives_restart: bool
    available: bool
    reason: str
    def __init__(self, id: _Optional[str] = ..., display_name: _Optional[str] = ..., description: _Optional[str] = ..., survives_restart: _Optional[bool] = ..., available: _Optional[bool] = ..., reason: _Optional[str] = ...) -> None: ...

class GetRequest(_message.Message):
    __slots__ = ()
    def __init__(self) -> None: ...

class GetResponse(_message.Message):
    __slots__ = ("capabilities", "timestamp", "session_backends", "default_backend")
    CAPABILITIES_FIELD_NUMBER: _ClassVar[int]
    TIMESTAMP_FIELD_NUMBER: _ClassVar[int]
    SESSION_BACKENDS_FIELD_NUMBER: _ClassVar[int]
    DEFAULT_BACKEND_FIELD_NUMBER: _ClassVar[int]
    capabilities: _containers.RepeatedCompositeFieldContainer[CapabilityState]
    timestamp: str
    session_backends: _containers.RepeatedCompositeFieldContainer[BackendOption]
    default_backend: str
    def __init__(self, capabilities: _Optional[_Iterable[_Union[CapabilityState, _Mapping]]] = ..., timestamp: _Optional[str] = ..., session_backends: _Optional[_Iterable[_Union[BackendOption, _Mapping]]] = ..., default_backend: _Optional[str] = ...) -> None: ...

class LivenessRequest(_message.Message):
    __slots__ = ()
    def __init__(self) -> None: ...

class LivenessResponse(_message.Message):
    __slots__ = ("capabilities", "timestamp")
    CAPABILITIES_FIELD_NUMBER: _ClassVar[int]
    TIMESTAMP_FIELD_NUMBER: _ClassVar[int]
    capabilities: _containers.RepeatedCompositeFieldContainer[CapabilityState]
    timestamp: str
    def __init__(self, capabilities: _Optional[_Iterable[_Union[CapabilityState, _Mapping]]] = ..., timestamp: _Optional[str] = ...) -> None: ...

class RunActionRequest(_message.Message):
    __slots__ = ("capability_id", "action_kind", "target_id")
    CAPABILITY_ID_FIELD_NUMBER: _ClassVar[int]
    ACTION_KIND_FIELD_NUMBER: _ClassVar[int]
    TARGET_ID_FIELD_NUMBER: _ClassVar[int]
    capability_id: str
    action_kind: str
    target_id: str
    def __init__(self, capability_id: _Optional[str] = ..., action_kind: _Optional[str] = ..., target_id: _Optional[str] = ...) -> None: ...

class RunActionResponse(_message.Message):
    __slots__ = ("success", "status", "message", "capability_id", "action_kind", "capabilities", "timestamp", "operation_id")
    SUCCESS_FIELD_NUMBER: _ClassVar[int]
    STATUS_FIELD_NUMBER: _ClassVar[int]
    MESSAGE_FIELD_NUMBER: _ClassVar[int]
    CAPABILITY_ID_FIELD_NUMBER: _ClassVar[int]
    ACTION_KIND_FIELD_NUMBER: _ClassVar[int]
    CAPABILITIES_FIELD_NUMBER: _ClassVar[int]
    TIMESTAMP_FIELD_NUMBER: _ClassVar[int]
    OPERATION_ID_FIELD_NUMBER: _ClassVar[int]
    success: bool
    status: str
    message: str
    capability_id: str
    action_kind: str
    capabilities: _containers.RepeatedCompositeFieldContainer[CapabilityState]
    timestamp: str
    operation_id: str
    def __init__(self, success: _Optional[bool] = ..., status: _Optional[str] = ..., message: _Optional[str] = ..., capability_id: _Optional[str] = ..., action_kind: _Optional[str] = ..., capabilities: _Optional[_Iterable[_Union[CapabilityState, _Mapping]]] = ..., timestamp: _Optional[str] = ..., operation_id: _Optional[str] = ...) -> None: ...
