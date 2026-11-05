from google.protobuf import struct_pb2 as _struct_pb2
from google.protobuf.internal import containers as _containers
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Iterable as _Iterable, Mapping as _Mapping
from typing import ClassVar as _ClassVar, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class ActionOwner(_message.Message):
    __slots__ = ("type", "id")
    TYPE_FIELD_NUMBER: _ClassVar[int]
    ID_FIELD_NUMBER: _ClassVar[int]
    type: str
    id: str
    def __init__(self, type: _Optional[str] = ..., id: _Optional[str] = ...) -> None: ...

class ActionCommand(_message.Message):
    __slots__ = ("argv",)
    ARGV_FIELD_NUMBER: _ClassVar[int]
    argv: _containers.RepeatedScalarFieldContainer[str]
    def __init__(self, argv: _Optional[_Iterable[str]] = ...) -> None: ...

class ActionInput(_message.Message):
    __slots__ = ("type", "description", "required", "enum", "default_value", "pattern", "min", "max", "max_length", "allow_multiline")
    TYPE_FIELD_NUMBER: _ClassVar[int]
    DESCRIPTION_FIELD_NUMBER: _ClassVar[int]
    REQUIRED_FIELD_NUMBER: _ClassVar[int]
    ENUM_FIELD_NUMBER: _ClassVar[int]
    DEFAULT_VALUE_FIELD_NUMBER: _ClassVar[int]
    PATTERN_FIELD_NUMBER: _ClassVar[int]
    MIN_FIELD_NUMBER: _ClassVar[int]
    MAX_FIELD_NUMBER: _ClassVar[int]
    MAX_LENGTH_FIELD_NUMBER: _ClassVar[int]
    ALLOW_MULTILINE_FIELD_NUMBER: _ClassVar[int]
    type: str
    description: str
    required: bool
    enum: _containers.RepeatedScalarFieldContainer[str]
    default_value: _struct_pb2.Value
    pattern: str
    min: float
    max: float
    max_length: int
    allow_multiline: bool
    def __init__(self, type: _Optional[str] = ..., description: _Optional[str] = ..., required: _Optional[bool] = ..., enum: _Optional[_Iterable[str]] = ..., default_value: _Optional[_Union[_struct_pb2.Value, _Mapping]] = ..., pattern: _Optional[str] = ..., min: _Optional[float] = ..., max: _Optional[float] = ..., max_length: _Optional[int] = ..., allow_multiline: _Optional[bool] = ...) -> None: ...

class ActionOutput(_message.Message):
    __slots__ = ("type", "description")
    TYPE_FIELD_NUMBER: _ClassVar[int]
    DESCRIPTION_FIELD_NUMBER: _ClassVar[int]
    type: str
    description: str
    def __init__(self, type: _Optional[str] = ..., description: _Optional[str] = ...) -> None: ...

class ActionPermissions(_message.Message):
    __slots__ = ("filesystem_read", "filesystem_write", "localhost_network", "external_network", "api_read", "api_write", "process_start", "process_stop", "host_configure", "secret_read", "secret_write", "destructive")
    FILESYSTEM_READ_FIELD_NUMBER: _ClassVar[int]
    FILESYSTEM_WRITE_FIELD_NUMBER: _ClassVar[int]
    LOCALHOST_NETWORK_FIELD_NUMBER: _ClassVar[int]
    EXTERNAL_NETWORK_FIELD_NUMBER: _ClassVar[int]
    API_READ_FIELD_NUMBER: _ClassVar[int]
    API_WRITE_FIELD_NUMBER: _ClassVar[int]
    PROCESS_START_FIELD_NUMBER: _ClassVar[int]
    PROCESS_STOP_FIELD_NUMBER: _ClassVar[int]
    HOST_CONFIGURE_FIELD_NUMBER: _ClassVar[int]
    SECRET_READ_FIELD_NUMBER: _ClassVar[int]
    SECRET_WRITE_FIELD_NUMBER: _ClassVar[int]
    DESTRUCTIVE_FIELD_NUMBER: _ClassVar[int]
    filesystem_read: bool
    filesystem_write: bool
    localhost_network: bool
    external_network: bool
    api_read: bool
    api_write: bool
    process_start: bool
    process_stop: bool
    host_configure: bool
    secret_read: bool
    secret_write: bool
    destructive: bool
    def __init__(self, filesystem_read: _Optional[bool] = ..., filesystem_write: _Optional[bool] = ..., localhost_network: _Optional[bool] = ..., external_network: _Optional[bool] = ..., api_read: _Optional[bool] = ..., api_write: _Optional[bool] = ..., process_start: _Optional[bool] = ..., process_stop: _Optional[bool] = ..., host_configure: _Optional[bool] = ..., secret_read: _Optional[bool] = ..., secret_write: _Optional[bool] = ..., destructive: _Optional[bool] = ...) -> None: ...

class ActionExample(_message.Message):
    __slots__ = ("description", "input")
    DESCRIPTION_FIELD_NUMBER: _ClassVar[int]
    INPUT_FIELD_NUMBER: _ClassVar[int]
    description: str
    input: _struct_pb2.Struct
    def __init__(self, description: _Optional[str] = ..., input: _Optional[_Union[_struct_pb2.Struct, _Mapping]] = ...) -> None: ...

class ActionExecution(_message.Message):
    __slots__ = ("timeout_seconds", "output_mode", "run_eligible")
    TIMEOUT_SECONDS_FIELD_NUMBER: _ClassVar[int]
    OUTPUT_MODE_FIELD_NUMBER: _ClassVar[int]
    RUN_ELIGIBLE_FIELD_NUMBER: _ClassVar[int]
    timeout_seconds: int
    output_mode: str
    run_eligible: bool
    def __init__(self, timeout_seconds: _Optional[int] = ..., output_mode: _Optional[str] = ..., run_eligible: _Optional[bool] = ...) -> None: ...

class ActionValidation(_message.Message):
    __slots__ = ("mode", "argv")
    MODE_FIELD_NUMBER: _ClassVar[int]
    ARGV_FIELD_NUMBER: _ClassVar[int]
    mode: str
    argv: _containers.RepeatedScalarFieldContainer[str]
    def __init__(self, mode: _Optional[str] = ..., argv: _Optional[_Iterable[str]] = ...) -> None: ...

class Action(_message.Message):
    __slots__ = ("id", "name", "description", "status", "owner", "command", "inputs", "outputs", "permissions", "examples", "tags", "execution", "validation", "created_at", "updated_at", "pack", "kind", "schema_version", "revision")
    class InputsEntry(_message.Message):
        __slots__ = ("key", "value")
        KEY_FIELD_NUMBER: _ClassVar[int]
        VALUE_FIELD_NUMBER: _ClassVar[int]
        key: str
        value: ActionInput
        def __init__(self, key: _Optional[str] = ..., value: _Optional[_Union[ActionInput, _Mapping]] = ...) -> None: ...
    class OutputsEntry(_message.Message):
        __slots__ = ("key", "value")
        KEY_FIELD_NUMBER: _ClassVar[int]
        VALUE_FIELD_NUMBER: _ClassVar[int]
        key: str
        value: ActionOutput
        def __init__(self, key: _Optional[str] = ..., value: _Optional[_Union[ActionOutput, _Mapping]] = ...) -> None: ...
    ID_FIELD_NUMBER: _ClassVar[int]
    NAME_FIELD_NUMBER: _ClassVar[int]
    DESCRIPTION_FIELD_NUMBER: _ClassVar[int]
    STATUS_FIELD_NUMBER: _ClassVar[int]
    OWNER_FIELD_NUMBER: _ClassVar[int]
    COMMAND_FIELD_NUMBER: _ClassVar[int]
    INPUTS_FIELD_NUMBER: _ClassVar[int]
    OUTPUTS_FIELD_NUMBER: _ClassVar[int]
    PERMISSIONS_FIELD_NUMBER: _ClassVar[int]
    EXAMPLES_FIELD_NUMBER: _ClassVar[int]
    TAGS_FIELD_NUMBER: _ClassVar[int]
    EXECUTION_FIELD_NUMBER: _ClassVar[int]
    VALIDATION_FIELD_NUMBER: _ClassVar[int]
    CREATED_AT_FIELD_NUMBER: _ClassVar[int]
    UPDATED_AT_FIELD_NUMBER: _ClassVar[int]
    PACK_FIELD_NUMBER: _ClassVar[int]
    KIND_FIELD_NUMBER: _ClassVar[int]
    SCHEMA_VERSION_FIELD_NUMBER: _ClassVar[int]
    REVISION_FIELD_NUMBER: _ClassVar[int]
    id: str
    name: str
    description: str
    status: str
    owner: ActionOwner
    command: ActionCommand
    inputs: _containers.MessageMap[str, ActionInput]
    outputs: _containers.MessageMap[str, ActionOutput]
    permissions: ActionPermissions
    examples: _containers.RepeatedCompositeFieldContainer[ActionExample]
    tags: _containers.RepeatedScalarFieldContainer[str]
    execution: ActionExecution
    validation: ActionValidation
    created_at: str
    updated_at: str
    pack: str
    kind: str
    schema_version: int
    revision: int
    def __init__(self, id: _Optional[str] = ..., name: _Optional[str] = ..., description: _Optional[str] = ..., status: _Optional[str] = ..., owner: _Optional[_Union[ActionOwner, _Mapping]] = ..., command: _Optional[_Union[ActionCommand, _Mapping]] = ..., inputs: _Optional[_Mapping[str, ActionInput]] = ..., outputs: _Optional[_Mapping[str, ActionOutput]] = ..., permissions: _Optional[_Union[ActionPermissions, _Mapping]] = ..., examples: _Optional[_Iterable[_Union[ActionExample, _Mapping]]] = ..., tags: _Optional[_Iterable[str]] = ..., execution: _Optional[_Union[ActionExecution, _Mapping]] = ..., validation: _Optional[_Union[ActionValidation, _Mapping]] = ..., created_at: _Optional[str] = ..., updated_at: _Optional[str] = ..., pack: _Optional[str] = ..., kind: _Optional[str] = ..., schema_version: _Optional[int] = ..., revision: _Optional[int] = ...) -> None: ...

class ValidationCheck(_message.Message):
    __slots__ = ("code", "status", "message", "path")
    CODE_FIELD_NUMBER: _ClassVar[int]
    STATUS_FIELD_NUMBER: _ClassVar[int]
    MESSAGE_FIELD_NUMBER: _ClassVar[int]
    PATH_FIELD_NUMBER: _ClassVar[int]
    code: str
    status: str
    message: str
    path: str
    def __init__(self, code: _Optional[str] = ..., status: _Optional[str] = ..., message: _Optional[str] = ..., path: _Optional[str] = ...) -> None: ...

class CommandResolution(_message.Message):
    __slots__ = ("certainty", "owner", "target", "command_path", "effect", "permissions", "run_surfaces", "requires_confirmation", "message")
    CERTAINTY_FIELD_NUMBER: _ClassVar[int]
    OWNER_FIELD_NUMBER: _ClassVar[int]
    TARGET_FIELD_NUMBER: _ClassVar[int]
    COMMAND_PATH_FIELD_NUMBER: _ClassVar[int]
    EFFECT_FIELD_NUMBER: _ClassVar[int]
    PERMISSIONS_FIELD_NUMBER: _ClassVar[int]
    RUN_SURFACES_FIELD_NUMBER: _ClassVar[int]
    REQUIRES_CONFIRMATION_FIELD_NUMBER: _ClassVar[int]
    MESSAGE_FIELD_NUMBER: _ClassVar[int]
    certainty: str
    owner: ActionOwner
    target: str
    command_path: _containers.RepeatedScalarFieldContainer[str]
    effect: str
    permissions: _containers.RepeatedScalarFieldContainer[str]
    run_surfaces: _containers.RepeatedScalarFieldContainer[str]
    requires_confirmation: bool
    message: str
    def __init__(self, certainty: _Optional[str] = ..., owner: _Optional[_Union[ActionOwner, _Mapping]] = ..., target: _Optional[str] = ..., command_path: _Optional[_Iterable[str]] = ..., effect: _Optional[str] = ..., permissions: _Optional[_Iterable[str]] = ..., run_surfaces: _Optional[_Iterable[str]] = ..., requires_confirmation: _Optional[bool] = ..., message: _Optional[str] = ...) -> None: ...

class ActionValidationResult(_message.Message):
    __slots__ = ("action_id", "valid", "runnable", "unvalidated", "requires_confirmation", "status", "command", "checks", "action")
    ACTION_ID_FIELD_NUMBER: _ClassVar[int]
    VALID_FIELD_NUMBER: _ClassVar[int]
    RUNNABLE_FIELD_NUMBER: _ClassVar[int]
    UNVALIDATED_FIELD_NUMBER: _ClassVar[int]
    REQUIRES_CONFIRMATION_FIELD_NUMBER: _ClassVar[int]
    STATUS_FIELD_NUMBER: _ClassVar[int]
    COMMAND_FIELD_NUMBER: _ClassVar[int]
    CHECKS_FIELD_NUMBER: _ClassVar[int]
    ACTION_FIELD_NUMBER: _ClassVar[int]
    action_id: str
    valid: bool
    runnable: bool
    unvalidated: bool
    requires_confirmation: bool
    status: str
    command: CommandResolution
    checks: _containers.RepeatedCompositeFieldContainer[ValidationCheck]
    action: Action
    def __init__(self, action_id: _Optional[str] = ..., valid: _Optional[bool] = ..., runnable: _Optional[bool] = ..., unvalidated: _Optional[bool] = ..., requires_confirmation: _Optional[bool] = ..., status: _Optional[str] = ..., command: _Optional[_Union[CommandResolution, _Mapping]] = ..., checks: _Optional[_Iterable[_Union[ValidationCheck, _Mapping]]] = ..., action: _Optional[_Union[Action, _Mapping]] = ...) -> None: ...

class ListActionsRequest(_message.Message):
    __slots__ = ("pack", "status", "owner", "tag")
    PACK_FIELD_NUMBER: _ClassVar[int]
    STATUS_FIELD_NUMBER: _ClassVar[int]
    OWNER_FIELD_NUMBER: _ClassVar[int]
    TAG_FIELD_NUMBER: _ClassVar[int]
    pack: str
    status: str
    owner: str
    tag: str
    def __init__(self, pack: _Optional[str] = ..., status: _Optional[str] = ..., owner: _Optional[str] = ..., tag: _Optional[str] = ...) -> None: ...

class ListActionsResponse(_message.Message):
    __slots__ = ("actions",)
    ACTIONS_FIELD_NUMBER: _ClassVar[int]
    actions: _containers.RepeatedCompositeFieldContainer[Action]
    def __init__(self, actions: _Optional[_Iterable[_Union[Action, _Mapping]]] = ...) -> None: ...

class GetActionRequest(_message.Message):
    __slots__ = ("id",)
    ID_FIELD_NUMBER: _ClassVar[int]
    id: str
    def __init__(self, id: _Optional[str] = ...) -> None: ...

class GetActionResponse(_message.Message):
    __slots__ = ("action",)
    ACTION_FIELD_NUMBER: _ClassVar[int]
    action: Action
    def __init__(self, action: _Optional[_Union[Action, _Mapping]] = ...) -> None: ...

class InputOverride(_message.Message):
    __slots__ = ("name", "type", "required", "description")
    NAME_FIELD_NUMBER: _ClassVar[int]
    TYPE_FIELD_NUMBER: _ClassVar[int]
    REQUIRED_FIELD_NUMBER: _ClassVar[int]
    DESCRIPTION_FIELD_NUMBER: _ClassVar[int]
    name: str
    type: str
    required: bool
    description: str
    def __init__(self, name: _Optional[str] = ..., type: _Optional[str] = ..., required: _Optional[bool] = ..., description: _Optional[str] = ...) -> None: ...

class InferenceNote(_message.Message):
    __slots__ = ("field", "message")
    FIELD_FIELD_NUMBER: _ClassVar[int]
    MESSAGE_FIELD_NUMBER: _ClassVar[int]
    field: str
    message: str
    def __init__(self, field: _Optional[str] = ..., message: _Optional[str] = ...) -> None: ...

class SimilarAction(_message.Message):
    __slots__ = ("id", "name", "score", "reason")
    ID_FIELD_NUMBER: _ClassVar[int]
    NAME_FIELD_NUMBER: _ClassVar[int]
    SCORE_FIELD_NUMBER: _ClassVar[int]
    REASON_FIELD_NUMBER: _ClassVar[int]
    id: str
    name: str
    score: float
    reason: str
    def __init__(self, id: _Optional[str] = ..., name: _Optional[str] = ..., score: _Optional[float] = ..., reason: _Optional[str] = ...) -> None: ...

class AuthorActionRequest(_message.Message):
    __slots__ = ("name", "description", "id", "pack", "argv", "inputs", "contract", "apply")
    NAME_FIELD_NUMBER: _ClassVar[int]
    DESCRIPTION_FIELD_NUMBER: _ClassVar[int]
    ID_FIELD_NUMBER: _ClassVar[int]
    PACK_FIELD_NUMBER: _ClassVar[int]
    ARGV_FIELD_NUMBER: _ClassVar[int]
    INPUTS_FIELD_NUMBER: _ClassVar[int]
    CONTRACT_FIELD_NUMBER: _ClassVar[int]
    APPLY_FIELD_NUMBER: _ClassVar[int]
    name: str
    description: str
    id: str
    pack: str
    argv: _containers.RepeatedScalarFieldContainer[str]
    inputs: _containers.RepeatedCompositeFieldContainer[InputOverride]
    contract: Action
    apply: bool
    def __init__(self, name: _Optional[str] = ..., description: _Optional[str] = ..., id: _Optional[str] = ..., pack: _Optional[str] = ..., argv: _Optional[_Iterable[str]] = ..., inputs: _Optional[_Iterable[_Union[InputOverride, _Mapping]]] = ..., contract: _Optional[_Union[Action, _Mapping]] = ..., apply: _Optional[bool] = ...) -> None: ...

class AuthorActionResponse(_message.Message):
    __slots__ = ("rendered", "validation", "inferred", "warnings", "similar", "applied")
    RENDERED_FIELD_NUMBER: _ClassVar[int]
    VALIDATION_FIELD_NUMBER: _ClassVar[int]
    INFERRED_FIELD_NUMBER: _ClassVar[int]
    WARNINGS_FIELD_NUMBER: _ClassVar[int]
    SIMILAR_FIELD_NUMBER: _ClassVar[int]
    APPLIED_FIELD_NUMBER: _ClassVar[int]
    rendered: Action
    validation: ActionValidationResult
    inferred: _containers.RepeatedCompositeFieldContainer[InferenceNote]
    warnings: _containers.RepeatedScalarFieldContainer[str]
    similar: _containers.RepeatedCompositeFieldContainer[SimilarAction]
    applied: bool
    def __init__(self, rendered: _Optional[_Union[Action, _Mapping]] = ..., validation: _Optional[_Union[ActionValidationResult, _Mapping]] = ..., inferred: _Optional[_Iterable[_Union[InferenceNote, _Mapping]]] = ..., warnings: _Optional[_Iterable[str]] = ..., similar: _Optional[_Iterable[_Union[SimilarAction, _Mapping]]] = ..., applied: _Optional[bool] = ...) -> None: ...

class UpdateActionRequest(_message.Message):
    __slots__ = ("id", "action")
    ID_FIELD_NUMBER: _ClassVar[int]
    ACTION_FIELD_NUMBER: _ClassVar[int]
    id: str
    action: Action
    def __init__(self, id: _Optional[str] = ..., action: _Optional[_Union[Action, _Mapping]] = ...) -> None: ...

class UpdateActionResponse(_message.Message):
    __slots__ = ("action", "validation")
    ACTION_FIELD_NUMBER: _ClassVar[int]
    VALIDATION_FIELD_NUMBER: _ClassVar[int]
    action: Action
    validation: ActionValidationResult
    def __init__(self, action: _Optional[_Union[Action, _Mapping]] = ..., validation: _Optional[_Union[ActionValidationResult, _Mapping]] = ...) -> None: ...

class DeleteActionRequest(_message.Message):
    __slots__ = ("id", "hard")
    ID_FIELD_NUMBER: _ClassVar[int]
    HARD_FIELD_NUMBER: _ClassVar[int]
    id: str
    hard: bool
    def __init__(self, id: _Optional[str] = ..., hard: _Optional[bool] = ...) -> None: ...

class DeleteActionResponse(_message.Message):
    __slots__ = ("id", "deleted", "archived")
    ID_FIELD_NUMBER: _ClassVar[int]
    DELETED_FIELD_NUMBER: _ClassVar[int]
    ARCHIVED_FIELD_NUMBER: _ClassVar[int]
    id: str
    deleted: bool
    archived: bool
    def __init__(self, id: _Optional[str] = ..., deleted: _Optional[bool] = ..., archived: _Optional[bool] = ...) -> None: ...

class ValidateActionRequest(_message.Message):
    __slots__ = ("id",)
    ID_FIELD_NUMBER: _ClassVar[int]
    id: str
    def __init__(self, id: _Optional[str] = ...) -> None: ...

class ValidateActionResponse(_message.Message):
    __slots__ = ("validation",)
    VALIDATION_FIELD_NUMBER: _ClassVar[int]
    validation: ActionValidationResult
    def __init__(self, validation: _Optional[_Union[ActionValidationResult, _Mapping]] = ...) -> None: ...

class RunActionRequest(_message.Message):
    __slots__ = ("id", "input", "dry_run")
    ID_FIELD_NUMBER: _ClassVar[int]
    INPUT_FIELD_NUMBER: _ClassVar[int]
    DRY_RUN_FIELD_NUMBER: _ClassVar[int]
    id: str
    input: _struct_pb2.Struct
    dry_run: bool
    def __init__(self, id: _Optional[str] = ..., input: _Optional[_Union[_struct_pb2.Struct, _Mapping]] = ..., dry_run: _Optional[bool] = ...) -> None: ...

class RunActionResponse(_message.Message):
    __slots__ = ("action_id", "status", "exit_code", "duration_ms", "argv", "stdout", "stderr", "stdout_truncated", "stderr_truncated", "output", "validation", "error")
    ACTION_ID_FIELD_NUMBER: _ClassVar[int]
    STATUS_FIELD_NUMBER: _ClassVar[int]
    EXIT_CODE_FIELD_NUMBER: _ClassVar[int]
    DURATION_MS_FIELD_NUMBER: _ClassVar[int]
    ARGV_FIELD_NUMBER: _ClassVar[int]
    STDOUT_FIELD_NUMBER: _ClassVar[int]
    STDERR_FIELD_NUMBER: _ClassVar[int]
    STDOUT_TRUNCATED_FIELD_NUMBER: _ClassVar[int]
    STDERR_TRUNCATED_FIELD_NUMBER: _ClassVar[int]
    OUTPUT_FIELD_NUMBER: _ClassVar[int]
    VALIDATION_FIELD_NUMBER: _ClassVar[int]
    ERROR_FIELD_NUMBER: _ClassVar[int]
    action_id: str
    status: str
    exit_code: int
    duration_ms: int
    argv: _containers.RepeatedScalarFieldContainer[str]
    stdout: str
    stderr: str
    stdout_truncated: bool
    stderr_truncated: bool
    output: _struct_pb2.Struct
    validation: ActionValidationResult
    error: str
    def __init__(self, action_id: _Optional[str] = ..., status: _Optional[str] = ..., exit_code: _Optional[int] = ..., duration_ms: _Optional[int] = ..., argv: _Optional[_Iterable[str]] = ..., stdout: _Optional[str] = ..., stderr: _Optional[str] = ..., stdout_truncated: _Optional[bool] = ..., stderr_truncated: _Optional[bool] = ..., output: _Optional[_Union[_struct_pb2.Struct, _Mapping]] = ..., validation: _Optional[_Union[ActionValidationResult, _Mapping]] = ..., error: _Optional[str] = ...) -> None: ...
