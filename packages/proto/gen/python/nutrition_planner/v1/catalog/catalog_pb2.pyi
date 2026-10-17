from google.protobuf.internal import containers as _containers
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Iterable as _Iterable, Mapping as _Mapping
from typing import ClassVar as _ClassVar, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class NutrientValue(_message.Message):
    __slots__ = ("nutrient_id", "amount", "unit", "basis", "basis_unit", "evidence", "source_ref")
    NUTRIENT_ID_FIELD_NUMBER: _ClassVar[int]
    AMOUNT_FIELD_NUMBER: _ClassVar[int]
    UNIT_FIELD_NUMBER: _ClassVar[int]
    BASIS_FIELD_NUMBER: _ClassVar[int]
    BASIS_UNIT_FIELD_NUMBER: _ClassVar[int]
    EVIDENCE_FIELD_NUMBER: _ClassVar[int]
    SOURCE_REF_FIELD_NUMBER: _ClassVar[int]
    nutrient_id: str
    amount: str
    unit: str
    basis: str
    basis_unit: str
    evidence: str
    source_ref: str
    def __init__(self, nutrient_id: _Optional[str] = ..., amount: _Optional[str] = ..., unit: _Optional[str] = ..., basis: _Optional[str] = ..., basis_unit: _Optional[str] = ..., evidence: _Optional[str] = ..., source_ref: _Optional[str] = ...) -> None: ...

class CatalogRevision(_message.Message):
    __slots__ = ("id", "revision", "concept_id", "name", "product_name", "preparation", "serving_quantity", "serving_unit", "nutrients", "allergen_evidence", "source_type", "source_ref", "created_at")
    class AllergenEvidenceEntry(_message.Message):
        __slots__ = ("key", "value")
        KEY_FIELD_NUMBER: _ClassVar[int]
        VALUE_FIELD_NUMBER: _ClassVar[int]
        key: str
        value: str
        def __init__(self, key: _Optional[str] = ..., value: _Optional[str] = ...) -> None: ...
    ID_FIELD_NUMBER: _ClassVar[int]
    REVISION_FIELD_NUMBER: _ClassVar[int]
    CONCEPT_ID_FIELD_NUMBER: _ClassVar[int]
    NAME_FIELD_NUMBER: _ClassVar[int]
    PRODUCT_NAME_FIELD_NUMBER: _ClassVar[int]
    PREPARATION_FIELD_NUMBER: _ClassVar[int]
    SERVING_QUANTITY_FIELD_NUMBER: _ClassVar[int]
    SERVING_UNIT_FIELD_NUMBER: _ClassVar[int]
    NUTRIENTS_FIELD_NUMBER: _ClassVar[int]
    ALLERGEN_EVIDENCE_FIELD_NUMBER: _ClassVar[int]
    SOURCE_TYPE_FIELD_NUMBER: _ClassVar[int]
    SOURCE_REF_FIELD_NUMBER: _ClassVar[int]
    CREATED_AT_FIELD_NUMBER: _ClassVar[int]
    id: str
    revision: int
    concept_id: str
    name: str
    product_name: str
    preparation: str
    serving_quantity: str
    serving_unit: str
    nutrients: _containers.RepeatedCompositeFieldContainer[NutrientValue]
    allergen_evidence: _containers.ScalarMap[str, str]
    source_type: str
    source_ref: str
    created_at: str
    def __init__(self, id: _Optional[str] = ..., revision: _Optional[int] = ..., concept_id: _Optional[str] = ..., name: _Optional[str] = ..., product_name: _Optional[str] = ..., preparation: _Optional[str] = ..., serving_quantity: _Optional[str] = ..., serving_unit: _Optional[str] = ..., nutrients: _Optional[_Iterable[_Union[NutrientValue, _Mapping]]] = ..., allergen_evidence: _Optional[_Mapping[str, str]] = ..., source_type: _Optional[str] = ..., source_ref: _Optional[str] = ..., created_at: _Optional[str] = ...) -> None: ...

class ListCatalogRequest(_message.Message):
    __slots__ = ("workspace_id",)
    WORKSPACE_ID_FIELD_NUMBER: _ClassVar[int]
    workspace_id: str
    def __init__(self, workspace_id: _Optional[str] = ...) -> None: ...

class ListCatalogResponse(_message.Message):
    __slots__ = ("revisions",)
    REVISIONS_FIELD_NUMBER: _ClassVar[int]
    revisions: _containers.RepeatedCompositeFieldContainer[CatalogRevision]
    def __init__(self, revisions: _Optional[_Iterable[_Union[CatalogRevision, _Mapping]]] = ...) -> None: ...

class CreateRevisionRequest(_message.Message):
    __slots__ = ("workspace_id", "concept_id", "name", "product_name", "preparation", "serving_quantity", "serving_unit", "nutrients", "allergen_evidence", "source_type", "source_ref")
    class AllergenEvidenceEntry(_message.Message):
        __slots__ = ("key", "value")
        KEY_FIELD_NUMBER: _ClassVar[int]
        VALUE_FIELD_NUMBER: _ClassVar[int]
        key: str
        value: str
        def __init__(self, key: _Optional[str] = ..., value: _Optional[str] = ...) -> None: ...
    WORKSPACE_ID_FIELD_NUMBER: _ClassVar[int]
    CONCEPT_ID_FIELD_NUMBER: _ClassVar[int]
    NAME_FIELD_NUMBER: _ClassVar[int]
    PRODUCT_NAME_FIELD_NUMBER: _ClassVar[int]
    PREPARATION_FIELD_NUMBER: _ClassVar[int]
    SERVING_QUANTITY_FIELD_NUMBER: _ClassVar[int]
    SERVING_UNIT_FIELD_NUMBER: _ClassVar[int]
    NUTRIENTS_FIELD_NUMBER: _ClassVar[int]
    ALLERGEN_EVIDENCE_FIELD_NUMBER: _ClassVar[int]
    SOURCE_TYPE_FIELD_NUMBER: _ClassVar[int]
    SOURCE_REF_FIELD_NUMBER: _ClassVar[int]
    workspace_id: str
    concept_id: str
    name: str
    product_name: str
    preparation: str
    serving_quantity: str
    serving_unit: str
    nutrients: _containers.RepeatedCompositeFieldContainer[NutrientValue]
    allergen_evidence: _containers.ScalarMap[str, str]
    source_type: str
    source_ref: str
    def __init__(self, workspace_id: _Optional[str] = ..., concept_id: _Optional[str] = ..., name: _Optional[str] = ..., product_name: _Optional[str] = ..., preparation: _Optional[str] = ..., serving_quantity: _Optional[str] = ..., serving_unit: _Optional[str] = ..., nutrients: _Optional[_Iterable[_Union[NutrientValue, _Mapping]]] = ..., allergen_evidence: _Optional[_Mapping[str, str]] = ..., source_type: _Optional[str] = ..., source_ref: _Optional[str] = ...) -> None: ...

class CreateRevisionResponse(_message.Message):
    __slots__ = ("revision",)
    REVISION_FIELD_NUMBER: _ClassVar[int]
    revision: CatalogRevision
    def __init__(self, revision: _Optional[_Union[CatalogRevision, _Mapping]] = ...) -> None: ...

class GetRevisionRequest(_message.Message):
    __slots__ = ("workspace_id", "id", "revision")
    WORKSPACE_ID_FIELD_NUMBER: _ClassVar[int]
    ID_FIELD_NUMBER: _ClassVar[int]
    REVISION_FIELD_NUMBER: _ClassVar[int]
    workspace_id: str
    id: str
    revision: int
    def __init__(self, workspace_id: _Optional[str] = ..., id: _Optional[str] = ..., revision: _Optional[int] = ...) -> None: ...

class GetRevisionResponse(_message.Message):
    __slots__ = ("revision",)
    REVISION_FIELD_NUMBER: _ClassVar[int]
    revision: CatalogRevision
    def __init__(self, revision: _Optional[_Union[CatalogRevision, _Mapping]] = ...) -> None: ...
