from google.protobuf.internal import containers as _containers
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Iterable as _Iterable, Mapping as _Mapping
from typing import ClassVar as _ClassVar, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class Target(_message.Message):
    __slots__ = ("id", "revision", "nutrient_id", "lower", "upper", "period", "scope", "enforcement", "provenance", "effective_from", "effective_to", "active")
    ID_FIELD_NUMBER: _ClassVar[int]
    REVISION_FIELD_NUMBER: _ClassVar[int]
    NUTRIENT_ID_FIELD_NUMBER: _ClassVar[int]
    LOWER_FIELD_NUMBER: _ClassVar[int]
    UPPER_FIELD_NUMBER: _ClassVar[int]
    PERIOD_FIELD_NUMBER: _ClassVar[int]
    SCOPE_FIELD_NUMBER: _ClassVar[int]
    ENFORCEMENT_FIELD_NUMBER: _ClassVar[int]
    PROVENANCE_FIELD_NUMBER: _ClassVar[int]
    EFFECTIVE_FROM_FIELD_NUMBER: _ClassVar[int]
    EFFECTIVE_TO_FIELD_NUMBER: _ClassVar[int]
    ACTIVE_FIELD_NUMBER: _ClassVar[int]
    id: str
    revision: int
    nutrient_id: str
    lower: str
    upper: str
    period: str
    scope: str
    enforcement: str
    provenance: str
    effective_from: str
    effective_to: str
    active: bool
    def __init__(self, id: _Optional[str] = ..., revision: _Optional[int] = ..., nutrient_id: _Optional[str] = ..., lower: _Optional[str] = ..., upper: _Optional[str] = ..., period: _Optional[str] = ..., scope: _Optional[str] = ..., enforcement: _Optional[str] = ..., provenance: _Optional[str] = ..., effective_from: _Optional[str] = ..., effective_to: _Optional[str] = ..., active: _Optional[bool] = ...) -> None: ...

class ListTargetsRequest(_message.Message):
    __slots__ = ("workspace_id",)
    WORKSPACE_ID_FIELD_NUMBER: _ClassVar[int]
    workspace_id: str
    def __init__(self, workspace_id: _Optional[str] = ...) -> None: ...

class ListTargetsResponse(_message.Message):
    __slots__ = ("targets",)
    TARGETS_FIELD_NUMBER: _ClassVar[int]
    targets: _containers.RepeatedCompositeFieldContainer[Target]
    def __init__(self, targets: _Optional[_Iterable[_Union[Target, _Mapping]]] = ...) -> None: ...

class CreateTargetRequest(_message.Message):
    __slots__ = ("workspace_id", "nutrient_id", "lower", "upper", "period", "scope", "enforcement", "provenance", "effective_from", "effective_to", "active")
    WORKSPACE_ID_FIELD_NUMBER: _ClassVar[int]
    NUTRIENT_ID_FIELD_NUMBER: _ClassVar[int]
    LOWER_FIELD_NUMBER: _ClassVar[int]
    UPPER_FIELD_NUMBER: _ClassVar[int]
    PERIOD_FIELD_NUMBER: _ClassVar[int]
    SCOPE_FIELD_NUMBER: _ClassVar[int]
    ENFORCEMENT_FIELD_NUMBER: _ClassVar[int]
    PROVENANCE_FIELD_NUMBER: _ClassVar[int]
    EFFECTIVE_FROM_FIELD_NUMBER: _ClassVar[int]
    EFFECTIVE_TO_FIELD_NUMBER: _ClassVar[int]
    ACTIVE_FIELD_NUMBER: _ClassVar[int]
    workspace_id: str
    nutrient_id: str
    lower: str
    upper: str
    period: str
    scope: str
    enforcement: str
    provenance: str
    effective_from: str
    effective_to: str
    active: bool
    def __init__(self, workspace_id: _Optional[str] = ..., nutrient_id: _Optional[str] = ..., lower: _Optional[str] = ..., upper: _Optional[str] = ..., period: _Optional[str] = ..., scope: _Optional[str] = ..., enforcement: _Optional[str] = ..., provenance: _Optional[str] = ..., effective_from: _Optional[str] = ..., effective_to: _Optional[str] = ..., active: _Optional[bool] = ...) -> None: ...

class CreateTargetResponse(_message.Message):
    __slots__ = ("target",)
    TARGET_FIELD_NUMBER: _ClassVar[int]
    target: Target
    def __init__(self, target: _Optional[_Union[Target, _Mapping]] = ...) -> None: ...

class GetTargetRequest(_message.Message):
    __slots__ = ("workspace_id", "id", "revision")
    WORKSPACE_ID_FIELD_NUMBER: _ClassVar[int]
    ID_FIELD_NUMBER: _ClassVar[int]
    REVISION_FIELD_NUMBER: _ClassVar[int]
    workspace_id: str
    id: str
    revision: int
    def __init__(self, workspace_id: _Optional[str] = ..., id: _Optional[str] = ..., revision: _Optional[int] = ...) -> None: ...

class GetTargetResponse(_message.Message):
    __slots__ = ("target",)
    TARGET_FIELD_NUMBER: _ClassVar[int]
    target: Target
    def __init__(self, target: _Optional[_Union[Target, _Mapping]] = ...) -> None: ...

class Intake(_message.Message):
    __slots__ = ("date", "nutrient_id", "planned", "actual", "recorded", "past")
    DATE_FIELD_NUMBER: _ClassVar[int]
    NUTRIENT_ID_FIELD_NUMBER: _ClassVar[int]
    PLANNED_FIELD_NUMBER: _ClassVar[int]
    ACTUAL_FIELD_NUMBER: _ClassVar[int]
    RECORDED_FIELD_NUMBER: _ClassVar[int]
    PAST_FIELD_NUMBER: _ClassVar[int]
    date: str
    nutrient_id: str
    planned: str
    actual: str
    recorded: bool
    past: bool
    def __init__(self, date: _Optional[str] = ..., nutrient_id: _Optional[str] = ..., planned: _Optional[str] = ..., actual: _Optional[str] = ..., recorded: _Optional[bool] = ..., past: _Optional[bool] = ...) -> None: ...

class EvaluateScopeRequest(_message.Message):
    __slots__ = ("workspace_id", "target_id", "target_revision", "nutrient_id", "scope", "intakes")
    WORKSPACE_ID_FIELD_NUMBER: _ClassVar[int]
    TARGET_ID_FIELD_NUMBER: _ClassVar[int]
    TARGET_REVISION_FIELD_NUMBER: _ClassVar[int]
    NUTRIENT_ID_FIELD_NUMBER: _ClassVar[int]
    SCOPE_FIELD_NUMBER: _ClassVar[int]
    INTAKES_FIELD_NUMBER: _ClassVar[int]
    workspace_id: str
    target_id: str
    target_revision: int
    nutrient_id: str
    scope: str
    intakes: _containers.RepeatedCompositeFieldContainer[Intake]
    def __init__(self, workspace_id: _Optional[str] = ..., target_id: _Optional[str] = ..., target_revision: _Optional[int] = ..., nutrient_id: _Optional[str] = ..., scope: _Optional[str] = ..., intakes: _Optional[_Iterable[_Union[Intake, _Mapping]]] = ...) -> None: ...

class EvaluateScopeResponse(_message.Message):
    __slots__ = ("nutrient_id", "known", "complete", "unresolved", "status", "reason")
    NUTRIENT_ID_FIELD_NUMBER: _ClassVar[int]
    KNOWN_FIELD_NUMBER: _ClassVar[int]
    COMPLETE_FIELD_NUMBER: _ClassVar[int]
    UNRESOLVED_FIELD_NUMBER: _ClassVar[int]
    STATUS_FIELD_NUMBER: _ClassVar[int]
    REASON_FIELD_NUMBER: _ClassVar[int]
    nutrient_id: str
    known: str
    complete: bool
    unresolved: _containers.RepeatedScalarFieldContainer[str]
    status: str
    reason: str
    def __init__(self, nutrient_id: _Optional[str] = ..., known: _Optional[str] = ..., complete: _Optional[bool] = ..., unresolved: _Optional[_Iterable[str]] = ..., status: _Optional[str] = ..., reason: _Optional[str] = ...) -> None: ...

class IntakeEvent(_message.Message):
    __slots__ = ("id", "date", "recipe_id", "recipe_revision", "nutrient_id", "amount", "unit", "reason", "correction_of", "recorded_at")
    ID_FIELD_NUMBER: _ClassVar[int]
    DATE_FIELD_NUMBER: _ClassVar[int]
    RECIPE_ID_FIELD_NUMBER: _ClassVar[int]
    RECIPE_REVISION_FIELD_NUMBER: _ClassVar[int]
    NUTRIENT_ID_FIELD_NUMBER: _ClassVar[int]
    AMOUNT_FIELD_NUMBER: _ClassVar[int]
    UNIT_FIELD_NUMBER: _ClassVar[int]
    REASON_FIELD_NUMBER: _ClassVar[int]
    CORRECTION_OF_FIELD_NUMBER: _ClassVar[int]
    RECORDED_AT_FIELD_NUMBER: _ClassVar[int]
    id: str
    date: str
    recipe_id: str
    recipe_revision: int
    nutrient_id: str
    amount: str
    unit: str
    reason: str
    correction_of: str
    recorded_at: str
    def __init__(self, id: _Optional[str] = ..., date: _Optional[str] = ..., recipe_id: _Optional[str] = ..., recipe_revision: _Optional[int] = ..., nutrient_id: _Optional[str] = ..., amount: _Optional[str] = ..., unit: _Optional[str] = ..., reason: _Optional[str] = ..., correction_of: _Optional[str] = ..., recorded_at: _Optional[str] = ...) -> None: ...

class ListIntakesRequest(_message.Message):
    __slots__ = ("workspace_id",)
    WORKSPACE_ID_FIELD_NUMBER: _ClassVar[int]
    workspace_id: str
    def __init__(self, workspace_id: _Optional[str] = ...) -> None: ...

class ListIntakesResponse(_message.Message):
    __slots__ = ("events",)
    EVENTS_FIELD_NUMBER: _ClassVar[int]
    events: _containers.RepeatedCompositeFieldContainer[IntakeEvent]
    def __init__(self, events: _Optional[_Iterable[_Union[IntakeEvent, _Mapping]]] = ...) -> None: ...

class RecordIntakeRequest(_message.Message):
    __slots__ = ("workspace_id", "id", "date", "recipe_id", "recipe_revision", "nutrient_id", "amount", "unit", "reason", "correction_of")
    WORKSPACE_ID_FIELD_NUMBER: _ClassVar[int]
    ID_FIELD_NUMBER: _ClassVar[int]
    DATE_FIELD_NUMBER: _ClassVar[int]
    RECIPE_ID_FIELD_NUMBER: _ClassVar[int]
    RECIPE_REVISION_FIELD_NUMBER: _ClassVar[int]
    NUTRIENT_ID_FIELD_NUMBER: _ClassVar[int]
    AMOUNT_FIELD_NUMBER: _ClassVar[int]
    UNIT_FIELD_NUMBER: _ClassVar[int]
    REASON_FIELD_NUMBER: _ClassVar[int]
    CORRECTION_OF_FIELD_NUMBER: _ClassVar[int]
    workspace_id: str
    id: str
    date: str
    recipe_id: str
    recipe_revision: int
    nutrient_id: str
    amount: str
    unit: str
    reason: str
    correction_of: str
    def __init__(self, workspace_id: _Optional[str] = ..., id: _Optional[str] = ..., date: _Optional[str] = ..., recipe_id: _Optional[str] = ..., recipe_revision: _Optional[int] = ..., nutrient_id: _Optional[str] = ..., amount: _Optional[str] = ..., unit: _Optional[str] = ..., reason: _Optional[str] = ..., correction_of: _Optional[str] = ...) -> None: ...

class RecordIntakeResponse(_message.Message):
    __slots__ = ("event",)
    EVENT_FIELD_NUMBER: _ClassVar[int]
    event: IntakeEvent
    def __init__(self, event: _Optional[_Union[IntakeEvent, _Mapping]] = ...) -> None: ...
