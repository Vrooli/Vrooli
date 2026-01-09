from common.v1 import types_pb2 as _types_pb2
from google.protobuf.internal import containers as _containers
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Iterable as _Iterable, Mapping as _Mapping
from typing import ClassVar as _ClassVar, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class ToolDefinition(_message.Message):
    __slots__ = ()
    NAME_FIELD_NUMBER: _ClassVar[int]
    DESCRIPTION_FIELD_NUMBER: _ClassVar[int]
    CATEGORY_FIELD_NUMBER: _ClassVar[int]
    PARAMETERS_FIELD_NUMBER: _ClassVar[int]
    METADATA_FIELD_NUMBER: _ClassVar[int]
    name: str
    description: str
    category: str
    parameters: ToolParameters
    metadata: ToolMetadata
    def __init__(self, name: _Optional[str] = ..., description: _Optional[str] = ..., category: _Optional[str] = ..., parameters: _Optional[_Union[ToolParameters, _Mapping]] = ..., metadata: _Optional[_Union[ToolMetadata, _Mapping]] = ...) -> None: ...

class ToolParameters(_message.Message):
    __slots__ = ()
    class PropertiesEntry(_message.Message):
        __slots__ = ()
        KEY_FIELD_NUMBER: _ClassVar[int]
        VALUE_FIELD_NUMBER: _ClassVar[int]
        key: str
        value: ParameterSchema
        def __init__(self, key: _Optional[str] = ..., value: _Optional[_Union[ParameterSchema, _Mapping]] = ...) -> None: ...
    TYPE_FIELD_NUMBER: _ClassVar[int]
    PROPERTIES_FIELD_NUMBER: _ClassVar[int]
    REQUIRED_FIELD_NUMBER: _ClassVar[int]
    ADDITIONAL_PROPERTIES_FIELD_NUMBER: _ClassVar[int]
    type: str
    properties: _containers.MessageMap[str, ParameterSchema]
    required: _containers.RepeatedScalarFieldContainer[str]
    additional_properties: bool
    def __init__(self, type: _Optional[str] = ..., properties: _Optional[_Mapping[str, ParameterSchema]] = ..., required: _Optional[_Iterable[str]] = ..., additional_properties: _Optional[bool] = ...) -> None: ...

class ParameterSchema(_message.Message):
    __slots__ = ()
    class PropertiesEntry(_message.Message):
        __slots__ = ()
        KEY_FIELD_NUMBER: _ClassVar[int]
        VALUE_FIELD_NUMBER: _ClassVar[int]
        key: str
        value: ParameterSchema
        def __init__(self, key: _Optional[str] = ..., value: _Optional[_Union[ParameterSchema, _Mapping]] = ...) -> None: ...
    TYPE_FIELD_NUMBER: _ClassVar[int]
    DESCRIPTION_FIELD_NUMBER: _ClassVar[int]
    ENUM_FIELD_NUMBER: _ClassVar[int]
    DEFAULT_FIELD_NUMBER: _ClassVar[int]
    ITEMS_FIELD_NUMBER: _ClassVar[int]
    PROPERTIES_FIELD_NUMBER: _ClassVar[int]
    REQUIRED_FIELD_NUMBER: _ClassVar[int]
    FORMAT_FIELD_NUMBER: _ClassVar[int]
    MINIMUM_FIELD_NUMBER: _ClassVar[int]
    MAXIMUM_FIELD_NUMBER: _ClassVar[int]
    MIN_LENGTH_FIELD_NUMBER: _ClassVar[int]
    MAX_LENGTH_FIELD_NUMBER: _ClassVar[int]
    PATTERN_FIELD_NUMBER: _ClassVar[int]
    type: str
    description: str
    enum: _containers.RepeatedScalarFieldContainer[str]
    default: _types_pb2.JsonValue
    items: ParameterSchema
    properties: _containers.MessageMap[str, ParameterSchema]
    required: _containers.RepeatedScalarFieldContainer[str]
    format: str
    minimum: float
    maximum: float
    min_length: int
    max_length: int
    pattern: str
    def __init__(self, type: _Optional[str] = ..., description: _Optional[str] = ..., enum: _Optional[_Iterable[str]] = ..., default: _Optional[_Union[_types_pb2.JsonValue, _Mapping]] = ..., items: _Optional[_Union[ParameterSchema, _Mapping]] = ..., properties: _Optional[_Mapping[str, ParameterSchema]] = ..., required: _Optional[_Iterable[str]] = ..., format: _Optional[str] = ..., minimum: _Optional[float] = ..., maximum: _Optional[float] = ..., min_length: _Optional[int] = ..., max_length: _Optional[int] = ..., pattern: _Optional[str] = ...) -> None: ...

class ToolMetadata(_message.Message):
    __slots__ = ()
    ENABLED_BY_DEFAULT_FIELD_NUMBER: _ClassVar[int]
    REQUIRES_APPROVAL_FIELD_NUMBER: _ClassVar[int]
    TIMEOUT_SECONDS_FIELD_NUMBER: _ClassVar[int]
    RATE_LIMIT_PER_MINUTE_FIELD_NUMBER: _ClassVar[int]
    COST_ESTIMATE_FIELD_NUMBER: _ClassVar[int]
    LONG_RUNNING_FIELD_NUMBER: _ClassVar[int]
    IDEMPOTENT_FIELD_NUMBER: _ClassVar[int]
    TAGS_FIELD_NUMBER: _ClassVar[int]
    EXAMPLES_FIELD_NUMBER: _ClassVar[int]
    ASYNC_BEHAVIOR_FIELD_NUMBER: _ClassVar[int]
    MODIFIES_STATE_FIELD_NUMBER: _ClassVar[int]
    SENSITIVE_OUTPUT_FIELD_NUMBER: _ClassVar[int]
    enabled_by_default: bool
    requires_approval: bool
    timeout_seconds: int
    rate_limit_per_minute: int
    cost_estimate: str
    long_running: bool
    idempotent: bool
    tags: _containers.RepeatedScalarFieldContainer[str]
    examples: _containers.RepeatedCompositeFieldContainer[ToolExample]
    async_behavior: AsyncBehavior
    modifies_state: bool
    sensitive_output: bool
    def __init__(self, enabled_by_default: _Optional[bool] = ..., requires_approval: _Optional[bool] = ..., timeout_seconds: _Optional[int] = ..., rate_limit_per_minute: _Optional[int] = ..., cost_estimate: _Optional[str] = ..., long_running: _Optional[bool] = ..., idempotent: _Optional[bool] = ..., tags: _Optional[_Iterable[str]] = ..., examples: _Optional[_Iterable[_Union[ToolExample, _Mapping]]] = ..., async_behavior: _Optional[_Union[AsyncBehavior, _Mapping]] = ..., modifies_state: _Optional[bool] = ..., sensitive_output: _Optional[bool] = ...) -> None: ...

class ToolExample(_message.Message):
    __slots__ = ()
    DESCRIPTION_FIELD_NUMBER: _ClassVar[int]
    INPUT_FIELD_NUMBER: _ClassVar[int]
    EXPECTED_OUTPUT_FIELD_NUMBER: _ClassVar[int]
    description: str
    input: _types_pb2.JsonObject
    expected_output: _types_pb2.JsonObject
    def __init__(self, description: _Optional[str] = ..., input: _Optional[_Union[_types_pb2.JsonObject, _Mapping]] = ..., expected_output: _Optional[_Union[_types_pb2.JsonObject, _Mapping]] = ...) -> None: ...

class AsyncBehavior(_message.Message):
    __slots__ = ()
    STATUS_POLLING_FIELD_NUMBER: _ClassVar[int]
    COMPLETION_CONDITIONS_FIELD_NUMBER: _ClassVar[int]
    PROGRESS_TRACKING_FIELD_NUMBER: _ClassVar[int]
    CANCELLATION_FIELD_NUMBER: _ClassVar[int]
    status_polling: StatusPolling
    completion_conditions: CompletionConditions
    progress_tracking: ProgressTracking
    cancellation: CancellationBehavior
    def __init__(self, status_polling: _Optional[_Union[StatusPolling, _Mapping]] = ..., completion_conditions: _Optional[_Union[CompletionConditions, _Mapping]] = ..., progress_tracking: _Optional[_Union[ProgressTracking, _Mapping]] = ..., cancellation: _Optional[_Union[CancellationBehavior, _Mapping]] = ...) -> None: ...

class StatusPolling(_message.Message):
    __slots__ = ()
    STATUS_TOOL_FIELD_NUMBER: _ClassVar[int]
    OPERATION_ID_FIELD_FIELD_NUMBER: _ClassVar[int]
    STATUS_TOOL_ID_PARAM_FIELD_NUMBER: _ClassVar[int]
    POLL_INTERVAL_SECONDS_FIELD_NUMBER: _ClassVar[int]
    MAX_POLL_DURATION_SECONDS_FIELD_NUMBER: _ClassVar[int]
    BACKOFF_FIELD_NUMBER: _ClassVar[int]
    status_tool: str
    operation_id_field: str
    status_tool_id_param: str
    poll_interval_seconds: int
    max_poll_duration_seconds: int
    backoff: PollingBackoff
    def __init__(self, status_tool: _Optional[str] = ..., operation_id_field: _Optional[str] = ..., status_tool_id_param: _Optional[str] = ..., poll_interval_seconds: _Optional[int] = ..., max_poll_duration_seconds: _Optional[int] = ..., backoff: _Optional[_Union[PollingBackoff, _Mapping]] = ...) -> None: ...

class PollingBackoff(_message.Message):
    __slots__ = ()
    INITIAL_INTERVAL_SECONDS_FIELD_NUMBER: _ClassVar[int]
    MAX_INTERVAL_SECONDS_FIELD_NUMBER: _ClassVar[int]
    MULTIPLIER_FIELD_NUMBER: _ClassVar[int]
    initial_interval_seconds: int
    max_interval_seconds: int
    multiplier: float
    def __init__(self, initial_interval_seconds: _Optional[int] = ..., max_interval_seconds: _Optional[int] = ..., multiplier: _Optional[float] = ...) -> None: ...

class CompletionConditions(_message.Message):
    __slots__ = ()
    STATUS_FIELD_FIELD_NUMBER: _ClassVar[int]
    SUCCESS_VALUES_FIELD_NUMBER: _ClassVar[int]
    FAILURE_VALUES_FIELD_NUMBER: _ClassVar[int]
    PENDING_VALUES_FIELD_NUMBER: _ClassVar[int]
    ERROR_FIELD_FIELD_NUMBER: _ClassVar[int]
    ERROR_DETAILS_FIELD_FIELD_NUMBER: _ClassVar[int]
    RESULT_FIELD_FIELD_NUMBER: _ClassVar[int]
    status_field: str
    success_values: _containers.RepeatedScalarFieldContainer[str]
    failure_values: _containers.RepeatedScalarFieldContainer[str]
    pending_values: _containers.RepeatedScalarFieldContainer[str]
    error_field: str
    error_details_field: str
    result_field: str
    def __init__(self, status_field: _Optional[str] = ..., success_values: _Optional[_Iterable[str]] = ..., failure_values: _Optional[_Iterable[str]] = ..., pending_values: _Optional[_Iterable[str]] = ..., error_field: _Optional[str] = ..., error_details_field: _Optional[str] = ..., result_field: _Optional[str] = ...) -> None: ...

class ProgressTracking(_message.Message):
    __slots__ = ()
    PROGRESS_FIELD_FIELD_NUMBER: _ClassVar[int]
    MESSAGE_FIELD_FIELD_NUMBER: _ClassVar[int]
    PHASE_FIELD_FIELD_NUMBER: _ClassVar[int]
    CURRENT_STEP_FIELD_FIELD_NUMBER: _ClassVar[int]
    TOTAL_STEPS_FIELD_FIELD_NUMBER: _ClassVar[int]
    ESTIMATED_REMAINING_FIELD_FIELD_NUMBER: _ClassVar[int]
    progress_field: str
    message_field: str
    phase_field: str
    current_step_field: str
    total_steps_field: str
    estimated_remaining_field: str
    def __init__(self, progress_field: _Optional[str] = ..., message_field: _Optional[str] = ..., phase_field: _Optional[str] = ..., current_step_field: _Optional[str] = ..., total_steps_field: _Optional[str] = ..., estimated_remaining_field: _Optional[str] = ...) -> None: ...

class CancellationBehavior(_message.Message):
    __slots__ = ()
    CANCEL_TOOL_FIELD_NUMBER: _ClassVar[int]
    CANCEL_TOOL_ID_PARAM_FIELD_NUMBER: _ClassVar[int]
    GRACEFUL_FIELD_NUMBER: _ClassVar[int]
    CANCEL_TIMEOUT_SECONDS_FIELD_NUMBER: _ClassVar[int]
    cancel_tool: str
    cancel_tool_id_param: str
    graceful: bool
    cancel_timeout_seconds: int
    def __init__(self, cancel_tool: _Optional[str] = ..., cancel_tool_id_param: _Optional[str] = ..., graceful: _Optional[bool] = ..., cancel_timeout_seconds: _Optional[int] = ...) -> None: ...

class ToolCategory(_message.Message):
    __slots__ = ()
    ID_FIELD_NUMBER: _ClassVar[int]
    NAME_FIELD_NUMBER: _ClassVar[int]
    DESCRIPTION_FIELD_NUMBER: _ClassVar[int]
    ICON_FIELD_NUMBER: _ClassVar[int]
    DISPLAY_ORDER_FIELD_NUMBER: _ClassVar[int]
    id: str
    name: str
    description: str
    icon: str
    display_order: int
    def __init__(self, id: _Optional[str] = ..., name: _Optional[str] = ..., description: _Optional[str] = ..., icon: _Optional[str] = ..., display_order: _Optional[int] = ...) -> None: ...
