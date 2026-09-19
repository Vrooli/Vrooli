import datetime

from google.protobuf import timestamp_pb2 as _timestamp_pb2
from google.protobuf.internal import containers as _containers
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Iterable as _Iterable, Mapping as _Mapping
from typing import ClassVar as _ClassVar, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class Recipe(_message.Message):
    __slots__ = ("id", "revision", "name", "notes", "source_url", "source_type", "original_text", "status", "created_at", "updated_at", "methods", "groups", "required_appliances", "allergen_evidence", "canonical_yield", "serving_unit", "ingredients")
    class AllergenEvidenceEntry(_message.Message):
        __slots__ = ("key", "value")
        KEY_FIELD_NUMBER: _ClassVar[int]
        VALUE_FIELD_NUMBER: _ClassVar[int]
        key: str
        value: str
        def __init__(self, key: _Optional[str] = ..., value: _Optional[str] = ...) -> None: ...
    ID_FIELD_NUMBER: _ClassVar[int]
    REVISION_FIELD_NUMBER: _ClassVar[int]
    NAME_FIELD_NUMBER: _ClassVar[int]
    NOTES_FIELD_NUMBER: _ClassVar[int]
    SOURCE_URL_FIELD_NUMBER: _ClassVar[int]
    SOURCE_TYPE_FIELD_NUMBER: _ClassVar[int]
    ORIGINAL_TEXT_FIELD_NUMBER: _ClassVar[int]
    STATUS_FIELD_NUMBER: _ClassVar[int]
    CREATED_AT_FIELD_NUMBER: _ClassVar[int]
    UPDATED_AT_FIELD_NUMBER: _ClassVar[int]
    METHODS_FIELD_NUMBER: _ClassVar[int]
    GROUPS_FIELD_NUMBER: _ClassVar[int]
    REQUIRED_APPLIANCES_FIELD_NUMBER: _ClassVar[int]
    ALLERGEN_EVIDENCE_FIELD_NUMBER: _ClassVar[int]
    CANONICAL_YIELD_FIELD_NUMBER: _ClassVar[int]
    SERVING_UNIT_FIELD_NUMBER: _ClassVar[int]
    INGREDIENTS_FIELD_NUMBER: _ClassVar[int]
    id: str
    revision: int
    name: str
    notes: str
    source_url: str
    source_type: str
    original_text: str
    status: str
    created_at: _timestamp_pb2.Timestamp
    updated_at: _timestamp_pb2.Timestamp
    methods: _containers.RepeatedCompositeFieldContainer[RecipeMethod]
    groups: _containers.RepeatedScalarFieldContainer[str]
    required_appliances: _containers.RepeatedScalarFieldContainer[str]
    allergen_evidence: _containers.ScalarMap[str, str]
    canonical_yield: str
    serving_unit: str
    ingredients: _containers.RepeatedCompositeFieldContainer[RecipeIngredient]
    def __init__(self, id: _Optional[str] = ..., revision: _Optional[int] = ..., name: _Optional[str] = ..., notes: _Optional[str] = ..., source_url: _Optional[str] = ..., source_type: _Optional[str] = ..., original_text: _Optional[str] = ..., status: _Optional[str] = ..., created_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., updated_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., methods: _Optional[_Iterable[_Union[RecipeMethod, _Mapping]]] = ..., groups: _Optional[_Iterable[str]] = ..., required_appliances: _Optional[_Iterable[str]] = ..., allergen_evidence: _Optional[_Mapping[str, str]] = ..., canonical_yield: _Optional[str] = ..., serving_unit: _Optional[str] = ..., ingredients: _Optional[_Iterable[_Union[RecipeIngredient, _Mapping]]] = ...) -> None: ...

class RecipeIngredient(_message.Message):
    __slots__ = ("id", "name", "amount", "unit", "discrete", "preparation")
    ID_FIELD_NUMBER: _ClassVar[int]
    NAME_FIELD_NUMBER: _ClassVar[int]
    AMOUNT_FIELD_NUMBER: _ClassVar[int]
    UNIT_FIELD_NUMBER: _ClassVar[int]
    DISCRETE_FIELD_NUMBER: _ClassVar[int]
    PREPARATION_FIELD_NUMBER: _ClassVar[int]
    id: str
    name: str
    amount: str
    unit: str
    discrete: bool
    preparation: str
    def __init__(self, id: _Optional[str] = ..., name: _Optional[str] = ..., amount: _Optional[str] = ..., unit: _Optional[str] = ..., discrete: _Optional[bool] = ..., preparation: _Optional[str] = ...) -> None: ...

class RecipeMethod(_message.Message):
    __slots__ = ("id", "name", "steps")
    ID_FIELD_NUMBER: _ClassVar[int]
    NAME_FIELD_NUMBER: _ClassVar[int]
    STEPS_FIELD_NUMBER: _ClassVar[int]
    id: str
    name: str
    steps: _containers.RepeatedCompositeFieldContainer[RecipeStep]
    def __init__(self, id: _Optional[str] = ..., name: _Optional[str] = ..., steps: _Optional[_Iterable[_Union[RecipeStep, _Mapping]]] = ...) -> None: ...

class RecipeStep(_message.Message):
    __slots__ = ("id", "instruction", "depends_on", "inputs", "outputs")
    ID_FIELD_NUMBER: _ClassVar[int]
    INSTRUCTION_FIELD_NUMBER: _ClassVar[int]
    DEPENDS_ON_FIELD_NUMBER: _ClassVar[int]
    INPUTS_FIELD_NUMBER: _ClassVar[int]
    OUTPUTS_FIELD_NUMBER: _ClassVar[int]
    id: str
    instruction: str
    depends_on: _containers.RepeatedScalarFieldContainer[str]
    inputs: _containers.RepeatedScalarFieldContainer[str]
    outputs: _containers.RepeatedScalarFieldContainer[str]
    def __init__(self, id: _Optional[str] = ..., instruction: _Optional[str] = ..., depends_on: _Optional[_Iterable[str]] = ..., inputs: _Optional[_Iterable[str]] = ..., outputs: _Optional[_Iterable[str]] = ...) -> None: ...

class ListRecipesRequest(_message.Message):
    __slots__ = ("workspace_id",)
    WORKSPACE_ID_FIELD_NUMBER: _ClassVar[int]
    workspace_id: str
    def __init__(self, workspace_id: _Optional[str] = ...) -> None: ...

class ListRecipesResponse(_message.Message):
    __slots__ = ("recipes",)
    RECIPES_FIELD_NUMBER: _ClassVar[int]
    recipes: _containers.RepeatedCompositeFieldContainer[Recipe]
    def __init__(self, recipes: _Optional[_Iterable[_Union[Recipe, _Mapping]]] = ...) -> None: ...

class CreateRecipeRequest(_message.Message):
    __slots__ = ("name", "notes", "source_url", "original_text", "source_type", "workspace_id", "idempotency_key", "methods", "groups", "required_appliances", "allergen_evidence", "canonical_yield", "serving_unit", "ingredients")
    class AllergenEvidenceEntry(_message.Message):
        __slots__ = ("key", "value")
        KEY_FIELD_NUMBER: _ClassVar[int]
        VALUE_FIELD_NUMBER: _ClassVar[int]
        key: str
        value: str
        def __init__(self, key: _Optional[str] = ..., value: _Optional[str] = ...) -> None: ...
    NAME_FIELD_NUMBER: _ClassVar[int]
    NOTES_FIELD_NUMBER: _ClassVar[int]
    SOURCE_URL_FIELD_NUMBER: _ClassVar[int]
    ORIGINAL_TEXT_FIELD_NUMBER: _ClassVar[int]
    SOURCE_TYPE_FIELD_NUMBER: _ClassVar[int]
    WORKSPACE_ID_FIELD_NUMBER: _ClassVar[int]
    IDEMPOTENCY_KEY_FIELD_NUMBER: _ClassVar[int]
    METHODS_FIELD_NUMBER: _ClassVar[int]
    GROUPS_FIELD_NUMBER: _ClassVar[int]
    REQUIRED_APPLIANCES_FIELD_NUMBER: _ClassVar[int]
    ALLERGEN_EVIDENCE_FIELD_NUMBER: _ClassVar[int]
    CANONICAL_YIELD_FIELD_NUMBER: _ClassVar[int]
    SERVING_UNIT_FIELD_NUMBER: _ClassVar[int]
    INGREDIENTS_FIELD_NUMBER: _ClassVar[int]
    name: str
    notes: str
    source_url: str
    original_text: str
    source_type: str
    workspace_id: str
    idempotency_key: str
    methods: _containers.RepeatedCompositeFieldContainer[RecipeMethod]
    groups: _containers.RepeatedScalarFieldContainer[str]
    required_appliances: _containers.RepeatedScalarFieldContainer[str]
    allergen_evidence: _containers.ScalarMap[str, str]
    canonical_yield: str
    serving_unit: str
    ingredients: _containers.RepeatedCompositeFieldContainer[RecipeIngredient]
    def __init__(self, name: _Optional[str] = ..., notes: _Optional[str] = ..., source_url: _Optional[str] = ..., original_text: _Optional[str] = ..., source_type: _Optional[str] = ..., workspace_id: _Optional[str] = ..., idempotency_key: _Optional[str] = ..., methods: _Optional[_Iterable[_Union[RecipeMethod, _Mapping]]] = ..., groups: _Optional[_Iterable[str]] = ..., required_appliances: _Optional[_Iterable[str]] = ..., allergen_evidence: _Optional[_Mapping[str, str]] = ..., canonical_yield: _Optional[str] = ..., serving_unit: _Optional[str] = ..., ingredients: _Optional[_Iterable[_Union[RecipeIngredient, _Mapping]]] = ...) -> None: ...

class CreateRecipeResponse(_message.Message):
    __slots__ = ("recipe",)
    RECIPE_FIELD_NUMBER: _ClassVar[int]
    recipe: Recipe
    def __init__(self, recipe: _Optional[_Union[Recipe, _Mapping]] = ...) -> None: ...

class GetRecipeRequest(_message.Message):
    __slots__ = ("workspace_id", "id")
    WORKSPACE_ID_FIELD_NUMBER: _ClassVar[int]
    ID_FIELD_NUMBER: _ClassVar[int]
    workspace_id: str
    id: str
    def __init__(self, workspace_id: _Optional[str] = ..., id: _Optional[str] = ...) -> None: ...

class GetRecipeResponse(_message.Message):
    __slots__ = ("recipe",)
    RECIPE_FIELD_NUMBER: _ClassVar[int]
    recipe: Recipe
    def __init__(self, recipe: _Optional[_Union[Recipe, _Mapping]] = ...) -> None: ...

class UpdateRecipeRequest(_message.Message):
    __slots__ = ("workspace_id", "id", "expected_revision", "name", "notes", "source_url", "original_text", "source_type", "idempotency_key", "methods", "groups", "required_appliances", "allergen_evidence", "canonical_yield", "serving_unit", "ingredients")
    class AllergenEvidenceEntry(_message.Message):
        __slots__ = ("key", "value")
        KEY_FIELD_NUMBER: _ClassVar[int]
        VALUE_FIELD_NUMBER: _ClassVar[int]
        key: str
        value: str
        def __init__(self, key: _Optional[str] = ..., value: _Optional[str] = ...) -> None: ...
    WORKSPACE_ID_FIELD_NUMBER: _ClassVar[int]
    ID_FIELD_NUMBER: _ClassVar[int]
    EXPECTED_REVISION_FIELD_NUMBER: _ClassVar[int]
    NAME_FIELD_NUMBER: _ClassVar[int]
    NOTES_FIELD_NUMBER: _ClassVar[int]
    SOURCE_URL_FIELD_NUMBER: _ClassVar[int]
    ORIGINAL_TEXT_FIELD_NUMBER: _ClassVar[int]
    SOURCE_TYPE_FIELD_NUMBER: _ClassVar[int]
    IDEMPOTENCY_KEY_FIELD_NUMBER: _ClassVar[int]
    METHODS_FIELD_NUMBER: _ClassVar[int]
    GROUPS_FIELD_NUMBER: _ClassVar[int]
    REQUIRED_APPLIANCES_FIELD_NUMBER: _ClassVar[int]
    ALLERGEN_EVIDENCE_FIELD_NUMBER: _ClassVar[int]
    CANONICAL_YIELD_FIELD_NUMBER: _ClassVar[int]
    SERVING_UNIT_FIELD_NUMBER: _ClassVar[int]
    INGREDIENTS_FIELD_NUMBER: _ClassVar[int]
    workspace_id: str
    id: str
    expected_revision: int
    name: str
    notes: str
    source_url: str
    original_text: str
    source_type: str
    idempotency_key: str
    methods: _containers.RepeatedCompositeFieldContainer[RecipeMethod]
    groups: _containers.RepeatedScalarFieldContainer[str]
    required_appliances: _containers.RepeatedScalarFieldContainer[str]
    allergen_evidence: _containers.ScalarMap[str, str]
    canonical_yield: str
    serving_unit: str
    ingredients: _containers.RepeatedCompositeFieldContainer[RecipeIngredient]
    def __init__(self, workspace_id: _Optional[str] = ..., id: _Optional[str] = ..., expected_revision: _Optional[int] = ..., name: _Optional[str] = ..., notes: _Optional[str] = ..., source_url: _Optional[str] = ..., original_text: _Optional[str] = ..., source_type: _Optional[str] = ..., idempotency_key: _Optional[str] = ..., methods: _Optional[_Iterable[_Union[RecipeMethod, _Mapping]]] = ..., groups: _Optional[_Iterable[str]] = ..., required_appliances: _Optional[_Iterable[str]] = ..., allergen_evidence: _Optional[_Mapping[str, str]] = ..., canonical_yield: _Optional[str] = ..., serving_unit: _Optional[str] = ..., ingredients: _Optional[_Iterable[_Union[RecipeIngredient, _Mapping]]] = ...) -> None: ...

class UpdateRecipeResponse(_message.Message):
    __slots__ = ("recipe",)
    RECIPE_FIELD_NUMBER: _ClassVar[int]
    recipe: Recipe
    def __init__(self, recipe: _Optional[_Union[Recipe, _Mapping]] = ...) -> None: ...
