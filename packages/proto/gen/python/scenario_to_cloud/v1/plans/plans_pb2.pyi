from google.protobuf.internal import containers as _containers
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Iterable as _Iterable, Mapping as _Mapping
from typing import ClassVar as _ClassVar, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class Target(_message.Message):
    __slots__ = ("machine_id", "node_id", "enrollment_generation", "transport")
    MACHINE_ID_FIELD_NUMBER: _ClassVar[int]
    NODE_ID_FIELD_NUMBER: _ClassVar[int]
    ENROLLMENT_GENERATION_FIELD_NUMBER: _ClassVar[int]
    TRANSPORT_FIELD_NUMBER: _ClassVar[int]
    machine_id: str
    node_id: str
    enrollment_generation: int
    transport: str
    def __init__(self, machine_id: _Optional[str] = ..., node_id: _Optional[str] = ..., enrollment_generation: _Optional[int] = ..., transport: _Optional[str] = ...) -> None: ...

class Precondition(_message.Message):
    __slots__ = ("kind", "value")
    KIND_FIELD_NUMBER: _ClassVar[int]
    VALUE_FIELD_NUMBER: _ClassVar[int]
    kind: str
    value: str
    def __init__(self, kind: _Optional[str] = ..., value: _Optional[str] = ...) -> None: ...

class Downtime(_message.Message):
    __slots__ = ("expected_seconds", "reason")
    EXPECTED_SECONDS_FIELD_NUMBER: _ClassVar[int]
    REASON_FIELD_NUMBER: _ClassVar[int]
    expected_seconds: int
    reason: str
    def __init__(self, expected_seconds: _Optional[int] = ..., reason: _Optional[str] = ...) -> None: ...

class Action(_message.Message):
    __slots__ = ("id", "owner_operation", "effect", "required_capability", "inputs", "depends_on", "verification", "recovery", "retry", "cancel_point", "downtime")
    class InputsEntry(_message.Message):
        __slots__ = ("key", "value")
        KEY_FIELD_NUMBER: _ClassVar[int]
        VALUE_FIELD_NUMBER: _ClassVar[int]
        key: str
        value: str
        def __init__(self, key: _Optional[str] = ..., value: _Optional[str] = ...) -> None: ...
    ID_FIELD_NUMBER: _ClassVar[int]
    OWNER_OPERATION_FIELD_NUMBER: _ClassVar[int]
    EFFECT_FIELD_NUMBER: _ClassVar[int]
    REQUIRED_CAPABILITY_FIELD_NUMBER: _ClassVar[int]
    INPUTS_FIELD_NUMBER: _ClassVar[int]
    DEPENDS_ON_FIELD_NUMBER: _ClassVar[int]
    VERIFICATION_FIELD_NUMBER: _ClassVar[int]
    RECOVERY_FIELD_NUMBER: _ClassVar[int]
    RETRY_FIELD_NUMBER: _ClassVar[int]
    CANCEL_POINT_FIELD_NUMBER: _ClassVar[int]
    DOWNTIME_FIELD_NUMBER: _ClassVar[int]
    id: str
    owner_operation: str
    effect: str
    required_capability: str
    inputs: _containers.ScalarMap[str, str]
    depends_on: _containers.RepeatedScalarFieldContainer[str]
    verification: str
    recovery: str
    retry: str
    cancel_point: bool
    downtime: Downtime
    def __init__(self, id: _Optional[str] = ..., owner_operation: _Optional[str] = ..., effect: _Optional[str] = ..., required_capability: _Optional[str] = ..., inputs: _Optional[_Mapping[str, str]] = ..., depends_on: _Optional[_Iterable[str]] = ..., verification: _Optional[str] = ..., recovery: _Optional[str] = ..., retry: _Optional[str] = ..., cancel_point: _Optional[bool] = ..., downtime: _Optional[_Union[Downtime, _Mapping]] = ...) -> None: ...

class Handoff(_message.Message):
    __slots__ = ("owner", "kind", "reference", "missing")
    OWNER_FIELD_NUMBER: _ClassVar[int]
    KIND_FIELD_NUMBER: _ClassVar[int]
    REFERENCE_FIELD_NUMBER: _ClassVar[int]
    MISSING_FIELD_NUMBER: _ClassVar[int]
    owner: str
    kind: str
    reference: str
    missing: _containers.RepeatedScalarFieldContainer[str]
    def __init__(self, owner: _Optional[str] = ..., kind: _Optional[str] = ..., reference: _Optional[str] = ..., missing: _Optional[_Iterable[str]] = ...) -> None: ...

class Presentation(_message.Message):
    __slots__ = ("title", "summary", "downtime_note", "recovery_note")
    TITLE_FIELD_NUMBER: _ClassVar[int]
    SUMMARY_FIELD_NUMBER: _ClassVar[int]
    DOWNTIME_NOTE_FIELD_NUMBER: _ClassVar[int]
    RECOVERY_NOTE_FIELD_NUMBER: _ClassVar[int]
    title: str
    summary: str
    downtime_note: str
    recovery_note: str
    def __init__(self, title: _Optional[str] = ..., summary: _Optional[str] = ..., downtime_note: _Optional[str] = ..., recovery_note: _Optional[str] = ...) -> None: ...

class ExecutablePlan(_message.Message):
    __slots__ = ("schema_version", "deployment_id", "scenario_id", "environment", "target", "scope", "outcome", "desired_revision", "release_digest", "configuration_digest", "closure_digest", "policy_version", "preconditions", "actions", "handoff", "presentation")
    SCHEMA_VERSION_FIELD_NUMBER: _ClassVar[int]
    DEPLOYMENT_ID_FIELD_NUMBER: _ClassVar[int]
    SCENARIO_ID_FIELD_NUMBER: _ClassVar[int]
    ENVIRONMENT_FIELD_NUMBER: _ClassVar[int]
    TARGET_FIELD_NUMBER: _ClassVar[int]
    SCOPE_FIELD_NUMBER: _ClassVar[int]
    OUTCOME_FIELD_NUMBER: _ClassVar[int]
    DESIRED_REVISION_FIELD_NUMBER: _ClassVar[int]
    RELEASE_DIGEST_FIELD_NUMBER: _ClassVar[int]
    CONFIGURATION_DIGEST_FIELD_NUMBER: _ClassVar[int]
    CLOSURE_DIGEST_FIELD_NUMBER: _ClassVar[int]
    POLICY_VERSION_FIELD_NUMBER: _ClassVar[int]
    PRECONDITIONS_FIELD_NUMBER: _ClassVar[int]
    ACTIONS_FIELD_NUMBER: _ClassVar[int]
    HANDOFF_FIELD_NUMBER: _ClassVar[int]
    PRESENTATION_FIELD_NUMBER: _ClassVar[int]
    schema_version: str
    deployment_id: str
    scenario_id: str
    environment: str
    target: Target
    scope: str
    outcome: str
    desired_revision: int
    release_digest: str
    configuration_digest: str
    closure_digest: str
    policy_version: str
    preconditions: _containers.RepeatedCompositeFieldContainer[Precondition]
    actions: _containers.RepeatedCompositeFieldContainer[Action]
    handoff: Handoff
    presentation: Presentation
    def __init__(self, schema_version: _Optional[str] = ..., deployment_id: _Optional[str] = ..., scenario_id: _Optional[str] = ..., environment: _Optional[str] = ..., target: _Optional[_Union[Target, _Mapping]] = ..., scope: _Optional[str] = ..., outcome: _Optional[str] = ..., desired_revision: _Optional[int] = ..., release_digest: _Optional[str] = ..., configuration_digest: _Optional[str] = ..., closure_digest: _Optional[str] = ..., policy_version: _Optional[str] = ..., preconditions: _Optional[_Iterable[_Union[Precondition, _Mapping]]] = ..., actions: _Optional[_Iterable[_Union[Action, _Mapping]]] = ..., handoff: _Optional[_Union[Handoff, _Mapping]] = ..., presentation: _Optional[_Union[Presentation, _Mapping]] = ...) -> None: ...

class Change(_message.Message):
    __slots__ = ("action_id", "operation", "effect", "capability", "summary", "verification", "recovery", "retry", "cancel_point")
    ACTION_ID_FIELD_NUMBER: _ClassVar[int]
    OPERATION_FIELD_NUMBER: _ClassVar[int]
    EFFECT_FIELD_NUMBER: _ClassVar[int]
    CAPABILITY_FIELD_NUMBER: _ClassVar[int]
    SUMMARY_FIELD_NUMBER: _ClassVar[int]
    VERIFICATION_FIELD_NUMBER: _ClassVar[int]
    RECOVERY_FIELD_NUMBER: _ClassVar[int]
    RETRY_FIELD_NUMBER: _ClassVar[int]
    CANCEL_POINT_FIELD_NUMBER: _ClassVar[int]
    action_id: str
    operation: str
    effect: str
    capability: str
    summary: str
    verification: str
    recovery: str
    retry: str
    cancel_point: bool
    def __init__(self, action_id: _Optional[str] = ..., operation: _Optional[str] = ..., effect: _Optional[str] = ..., capability: _Optional[str] = ..., summary: _Optional[str] = ..., verification: _Optional[str] = ..., recovery: _Optional[str] = ..., retry: _Optional[str] = ..., cancel_point: _Optional[bool] = ...) -> None: ...

class DataEffect(_message.Message):
    __slots__ = ("action_id", "subject", "effect")
    ACTION_ID_FIELD_NUMBER: _ClassVar[int]
    SUBJECT_FIELD_NUMBER: _ClassVar[int]
    EFFECT_FIELD_NUMBER: _ClassVar[int]
    action_id: str
    subject: str
    effect: str
    def __init__(self, action_id: _Optional[str] = ..., subject: _Optional[str] = ..., effect: _Optional[str] = ...) -> None: ...

class ShellPreviewLine(_message.Message):
    __slots__ = ("action_id", "command")
    ACTION_ID_FIELD_NUMBER: _ClassVar[int]
    COMMAND_FIELD_NUMBER: _ClassVar[int]
    action_id: str
    command: str
    def __init__(self, action_id: _Optional[str] = ..., command: _Optional[str] = ...) -> None: ...

class Preview(_message.Message):
    __slots__ = ("target", "outcome", "changes", "data_effects", "downtime", "recovery_strategy", "handoff", "shell_preview")
    TARGET_FIELD_NUMBER: _ClassVar[int]
    OUTCOME_FIELD_NUMBER: _ClassVar[int]
    CHANGES_FIELD_NUMBER: _ClassVar[int]
    DATA_EFFECTS_FIELD_NUMBER: _ClassVar[int]
    DOWNTIME_FIELD_NUMBER: _ClassVar[int]
    RECOVERY_STRATEGY_FIELD_NUMBER: _ClassVar[int]
    HANDOFF_FIELD_NUMBER: _ClassVar[int]
    SHELL_PREVIEW_FIELD_NUMBER: _ClassVar[int]
    target: str
    outcome: str
    changes: _containers.RepeatedCompositeFieldContainer[Change]
    data_effects: _containers.RepeatedCompositeFieldContainer[DataEffect]
    downtime: Downtime
    recovery_strategy: str
    handoff: Handoff
    shell_preview: _containers.RepeatedCompositeFieldContainer[ShellPreviewLine]
    def __init__(self, target: _Optional[str] = ..., outcome: _Optional[str] = ..., changes: _Optional[_Iterable[_Union[Change, _Mapping]]] = ..., data_effects: _Optional[_Iterable[_Union[DataEffect, _Mapping]]] = ..., downtime: _Optional[_Union[Downtime, _Mapping]] = ..., recovery_strategy: _Optional[str] = ..., handoff: _Optional[_Union[Handoff, _Mapping]] = ..., shell_preview: _Optional[_Iterable[_Union[ShellPreviewLine, _Mapping]]] = ...) -> None: ...

class CompilePlanRequest(_message.Message):
    __slots__ = ("deployment_id", "scope", "force_bundle_build")
    DEPLOYMENT_ID_FIELD_NUMBER: _ClassVar[int]
    SCOPE_FIELD_NUMBER: _ClassVar[int]
    FORCE_BUNDLE_BUILD_FIELD_NUMBER: _ClassVar[int]
    deployment_id: str
    scope: str
    force_bundle_build: bool
    def __init__(self, deployment_id: _Optional[str] = ..., scope: _Optional[str] = ..., force_bundle_build: _Optional[bool] = ...) -> None: ...

class CompilePlanResponse(_message.Message):
    __slots__ = ("schema_version", "plan", "plan_digest", "preview", "closure_status")
    SCHEMA_VERSION_FIELD_NUMBER: _ClassVar[int]
    PLAN_FIELD_NUMBER: _ClassVar[int]
    PLAN_DIGEST_FIELD_NUMBER: _ClassVar[int]
    PREVIEW_FIELD_NUMBER: _ClassVar[int]
    CLOSURE_STATUS_FIELD_NUMBER: _ClassVar[int]
    schema_version: str
    plan: ExecutablePlan
    plan_digest: str
    preview: Preview
    closure_status: str
    def __init__(self, schema_version: _Optional[str] = ..., plan: _Optional[_Union[ExecutablePlan, _Mapping]] = ..., plan_digest: _Optional[str] = ..., preview: _Optional[_Union[Preview, _Mapping]] = ..., closure_status: _Optional[str] = ...) -> None: ...

class ApplyPlanRequest(_message.Message):
    __slots__ = ("deployment_id", "plan_digest", "request_key", "scope", "run_preflight")
    DEPLOYMENT_ID_FIELD_NUMBER: _ClassVar[int]
    PLAN_DIGEST_FIELD_NUMBER: _ClassVar[int]
    REQUEST_KEY_FIELD_NUMBER: _ClassVar[int]
    SCOPE_FIELD_NUMBER: _ClassVar[int]
    RUN_PREFLIGHT_FIELD_NUMBER: _ClassVar[int]
    deployment_id: str
    plan_digest: str
    request_key: str
    scope: str
    run_preflight: bool
    def __init__(self, deployment_id: _Optional[str] = ..., plan_digest: _Optional[str] = ..., request_key: _Optional[str] = ..., scope: _Optional[str] = ..., run_preflight: _Optional[bool] = ...) -> None: ...

class ApplyPlanResponse(_message.Message):
    __slots__ = ("schema_version", "operation_id", "plan_digest", "state")
    SCHEMA_VERSION_FIELD_NUMBER: _ClassVar[int]
    OPERATION_ID_FIELD_NUMBER: _ClassVar[int]
    PLAN_DIGEST_FIELD_NUMBER: _ClassVar[int]
    STATE_FIELD_NUMBER: _ClassVar[int]
    schema_version: str
    operation_id: str
    plan_digest: str
    state: str
    def __init__(self, schema_version: _Optional[str] = ..., operation_id: _Optional[str] = ..., plan_digest: _Optional[str] = ..., state: _Optional[str] = ...) -> None: ...
