from google.protobuf.internal import containers as _containers
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Iterable as _Iterable, Mapping as _Mapping
from typing import ClassVar as _ClassVar, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class MealSlot(_message.Message):
    __slots__ = ("date", "slot_name", "mode", "locked_recipe_id", "quantity")
    DATE_FIELD_NUMBER: _ClassVar[int]
    SLOT_NAME_FIELD_NUMBER: _ClassVar[int]
    MODE_FIELD_NUMBER: _ClassVar[int]
    LOCKED_RECIPE_ID_FIELD_NUMBER: _ClassVar[int]
    QUANTITY_FIELD_NUMBER: _ClassVar[int]
    date: str
    slot_name: str
    mode: str
    locked_recipe_id: str
    quantity: str
    def __init__(self, date: _Optional[str] = ..., slot_name: _Optional[str] = ..., mode: _Optional[str] = ..., locked_recipe_id: _Optional[str] = ..., quantity: _Optional[str] = ...) -> None: ...

class GeneratePlanRequest(_message.Message):
    __slots__ = ("workspace_id", "dates", "locked_recipe_ids", "seed", "cost_weight", "effort_weight", "repetition_weight", "meal_slots")
    class LockedRecipeIdsEntry(_message.Message):
        __slots__ = ("key", "value")
        KEY_FIELD_NUMBER: _ClassVar[int]
        VALUE_FIELD_NUMBER: _ClassVar[int]
        key: str
        value: str
        def __init__(self, key: _Optional[str] = ..., value: _Optional[str] = ...) -> None: ...
    WORKSPACE_ID_FIELD_NUMBER: _ClassVar[int]
    DATES_FIELD_NUMBER: _ClassVar[int]
    LOCKED_RECIPE_IDS_FIELD_NUMBER: _ClassVar[int]
    SEED_FIELD_NUMBER: _ClassVar[int]
    COST_WEIGHT_FIELD_NUMBER: _ClassVar[int]
    EFFORT_WEIGHT_FIELD_NUMBER: _ClassVar[int]
    REPETITION_WEIGHT_FIELD_NUMBER: _ClassVar[int]
    MEAL_SLOTS_FIELD_NUMBER: _ClassVar[int]
    workspace_id: str
    dates: _containers.RepeatedScalarFieldContainer[str]
    locked_recipe_ids: _containers.ScalarMap[str, str]
    seed: int
    cost_weight: float
    effort_weight: float
    repetition_weight: float
    meal_slots: _containers.RepeatedCompositeFieldContainer[MealSlot]
    def __init__(self, workspace_id: _Optional[str] = ..., dates: _Optional[_Iterable[str]] = ..., locked_recipe_ids: _Optional[_Mapping[str, str]] = ..., seed: _Optional[int] = ..., cost_weight: _Optional[float] = ..., effort_weight: _Optional[float] = ..., repetition_weight: _Optional[float] = ..., meal_slots: _Optional[_Iterable[_Union[MealSlot, _Mapping]]] = ...) -> None: ...

class GeneratePlanResponse(_message.Message):
    __slots__ = ("run_id", "draft_json", "unresolved_dates", "input_references", "current_revision")
    RUN_ID_FIELD_NUMBER: _ClassVar[int]
    DRAFT_JSON_FIELD_NUMBER: _ClassVar[int]
    UNRESOLVED_DATES_FIELD_NUMBER: _ClassVar[int]
    INPUT_REFERENCES_FIELD_NUMBER: _ClassVar[int]
    CURRENT_REVISION_FIELD_NUMBER: _ClassVar[int]
    run_id: str
    draft_json: str
    unresolved_dates: _containers.RepeatedScalarFieldContainer[str]
    input_references: _containers.RepeatedScalarFieldContainer[str]
    current_revision: int
    def __init__(self, run_id: _Optional[str] = ..., draft_json: _Optional[str] = ..., unresolved_dates: _Optional[_Iterable[str]] = ..., input_references: _Optional[_Iterable[str]] = ..., current_revision: _Optional[int] = ...) -> None: ...

class ApplyPlanRequest(_message.Message):
    __slots__ = ("workspace_id", "expected_revision", "draft_json")
    WORKSPACE_ID_FIELD_NUMBER: _ClassVar[int]
    EXPECTED_REVISION_FIELD_NUMBER: _ClassVar[int]
    DRAFT_JSON_FIELD_NUMBER: _ClassVar[int]
    workspace_id: str
    expected_revision: int
    draft_json: str
    def __init__(self, workspace_id: _Optional[str] = ..., expected_revision: _Optional[int] = ..., draft_json: _Optional[str] = ...) -> None: ...

class ApplyPlanResponse(_message.Message):
    __slots__ = ("revision", "plan_json")
    REVISION_FIELD_NUMBER: _ClassVar[int]
    PLAN_JSON_FIELD_NUMBER: _ClassVar[int]
    revision: int
    plan_json: str
    def __init__(self, revision: _Optional[int] = ..., plan_json: _Optional[str] = ...) -> None: ...

class PreviewSwapRequest(_message.Message):
    __slots__ = ("workspace_id", "expected_revision", "date", "replacement_recipe_id", "replace_matching_future")
    WORKSPACE_ID_FIELD_NUMBER: _ClassVar[int]
    EXPECTED_REVISION_FIELD_NUMBER: _ClassVar[int]
    DATE_FIELD_NUMBER: _ClassVar[int]
    REPLACEMENT_RECIPE_ID_FIELD_NUMBER: _ClassVar[int]
    REPLACE_MATCHING_FUTURE_FIELD_NUMBER: _ClassVar[int]
    workspace_id: str
    expected_revision: int
    date: str
    replacement_recipe_id: str
    replace_matching_future: bool
    def __init__(self, workspace_id: _Optional[str] = ..., expected_revision: _Optional[int] = ..., date: _Optional[str] = ..., replacement_recipe_id: _Optional[str] = ..., replace_matching_future: _Optional[bool] = ...) -> None: ...

class PreviewSwapResponse(_message.Message):
    __slots__ = ("revision", "preview_json", "affected_dates")
    REVISION_FIELD_NUMBER: _ClassVar[int]
    PREVIEW_JSON_FIELD_NUMBER: _ClassVar[int]
    AFFECTED_DATES_FIELD_NUMBER: _ClassVar[int]
    revision: int
    preview_json: str
    affected_dates: _containers.RepeatedScalarFieldContainer[str]
    def __init__(self, revision: _Optional[int] = ..., preview_json: _Optional[str] = ..., affected_dates: _Optional[_Iterable[str]] = ...) -> None: ...

class GetShoppingPreviewRequest(_message.Message):
    __slots__ = ("workspace_id", "expected_revision")
    WORKSPACE_ID_FIELD_NUMBER: _ClassVar[int]
    EXPECTED_REVISION_FIELD_NUMBER: _ClassVar[int]
    workspace_id: str
    expected_revision: int
    def __init__(self, workspace_id: _Optional[str] = ..., expected_revision: _Optional[int] = ...) -> None: ...

class ShoppingLine(_message.Message):
    __slots__ = ("key", "label", "need", "stock", "missing", "package_count", "price", "source_recipe_ids", "checked")
    KEY_FIELD_NUMBER: _ClassVar[int]
    LABEL_FIELD_NUMBER: _ClassVar[int]
    NEED_FIELD_NUMBER: _ClassVar[int]
    STOCK_FIELD_NUMBER: _ClassVar[int]
    MISSING_FIELD_NUMBER: _ClassVar[int]
    PACKAGE_COUNT_FIELD_NUMBER: _ClassVar[int]
    PRICE_FIELD_NUMBER: _ClassVar[int]
    SOURCE_RECIPE_IDS_FIELD_NUMBER: _ClassVar[int]
    CHECKED_FIELD_NUMBER: _ClassVar[int]
    key: str
    label: str
    need: str
    stock: str
    missing: str
    package_count: str
    price: str
    source_recipe_ids: _containers.RepeatedScalarFieldContainer[str]
    checked: bool
    def __init__(self, key: _Optional[str] = ..., label: _Optional[str] = ..., need: _Optional[str] = ..., stock: _Optional[str] = ..., missing: _Optional[str] = ..., package_count: _Optional[str] = ..., price: _Optional[str] = ..., source_recipe_ids: _Optional[_Iterable[str]] = ..., checked: _Optional[bool] = ...) -> None: ...

class GetShoppingPreviewResponse(_message.Message):
    __slots__ = ("revision", "lines")
    REVISION_FIELD_NUMBER: _ClassVar[int]
    LINES_FIELD_NUMBER: _ClassVar[int]
    revision: int
    lines: _containers.RepeatedCompositeFieldContainer[ShoppingLine]
    def __init__(self, revision: _Optional[int] = ..., lines: _Optional[_Iterable[_Union[ShoppingLine, _Mapping]]] = ...) -> None: ...

class SetShoppingCheckedRequest(_message.Message):
    __slots__ = ("workspace_id", "line_key", "checked")
    WORKSPACE_ID_FIELD_NUMBER: _ClassVar[int]
    LINE_KEY_FIELD_NUMBER: _ClassVar[int]
    CHECKED_FIELD_NUMBER: _ClassVar[int]
    workspace_id: str
    line_key: str
    checked: bool
    def __init__(self, workspace_id: _Optional[str] = ..., line_key: _Optional[str] = ..., checked: _Optional[bool] = ...) -> None: ...

class SetShoppingCheckedResponse(_message.Message):
    __slots__ = ("checked",)
    CHECKED_FIELD_NUMBER: _ClassVar[int]
    checked: bool
    def __init__(self, checked: _Optional[bool] = ...) -> None: ...

class RecordFeedbackRequest(_message.Message):
    __slots__ = ("workspace_id", "expected_revision", "date", "recipe_id", "portion", "minutes")
    WORKSPACE_ID_FIELD_NUMBER: _ClassVar[int]
    EXPECTED_REVISION_FIELD_NUMBER: _ClassVar[int]
    DATE_FIELD_NUMBER: _ClassVar[int]
    RECIPE_ID_FIELD_NUMBER: _ClassVar[int]
    PORTION_FIELD_NUMBER: _ClassVar[int]
    MINUTES_FIELD_NUMBER: _ClassVar[int]
    workspace_id: str
    expected_revision: int
    date: str
    recipe_id: str
    portion: str
    minutes: int
    def __init__(self, workspace_id: _Optional[str] = ..., expected_revision: _Optional[int] = ..., date: _Optional[str] = ..., recipe_id: _Optional[str] = ..., portion: _Optional[str] = ..., minutes: _Optional[int] = ...) -> None: ...

class RecordFeedbackResponse(_message.Message):
    __slots__ = ("date", "recipe_id", "portion", "minutes")
    DATE_FIELD_NUMBER: _ClassVar[int]
    RECIPE_ID_FIELD_NUMBER: _ClassVar[int]
    PORTION_FIELD_NUMBER: _ClassVar[int]
    MINUTES_FIELD_NUMBER: _ClassVar[int]
    date: str
    recipe_id: str
    portion: str
    minutes: int
    def __init__(self, date: _Optional[str] = ..., recipe_id: _Optional[str] = ..., portion: _Optional[str] = ..., minutes: _Optional[int] = ...) -> None: ...

class UndoFeedbackRequest(_message.Message):
    __slots__ = ("workspace_id", "date")
    WORKSPACE_ID_FIELD_NUMBER: _ClassVar[int]
    DATE_FIELD_NUMBER: _ClassVar[int]
    workspace_id: str
    date: str
    def __init__(self, workspace_id: _Optional[str] = ..., date: _Optional[str] = ...) -> None: ...

class UndoFeedbackResponse(_message.Message):
    __slots__ = ("undone",)
    UNDONE_FIELD_NUMBER: _ClassVar[int]
    undone: bool
    def __init__(self, undone: _Optional[bool] = ...) -> None: ...
