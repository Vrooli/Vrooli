import datetime

from google.protobuf import timestamp_pb2 as _timestamp_pb2
from google.protobuf.internal import containers as _containers
from google.protobuf.internal import enum_type_wrapper as _enum_type_wrapper
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Iterable as _Iterable, Mapping as _Mapping
from typing import ClassVar as _ClassVar, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class ClassificationState(int, metaclass=_enum_type_wrapper.EnumTypeWrapper):
    __slots__ = ()
    CLASSIFICATION_STATE_UNSPECIFIED: _ClassVar[ClassificationState]
    CLASSIFICATION_STATE_PROPOSED: _ClassVar[ClassificationState]
    CLASSIFICATION_STATE_CONFIRMED: _ClassVar[ClassificationState]
    CLASSIFICATION_STATE_OVERRIDDEN: _ClassVar[ClassificationState]
    CLASSIFICATION_STATE_UNCATEGORIZED: _ClassVar[ClassificationState]
CLASSIFICATION_STATE_UNSPECIFIED: ClassificationState
CLASSIFICATION_STATE_PROPOSED: ClassificationState
CLASSIFICATION_STATE_CONFIRMED: ClassificationState
CLASSIFICATION_STATE_OVERRIDDEN: ClassificationState
CLASSIFICATION_STATE_UNCATEGORIZED: ClassificationState

class Category(_message.Message):
    __slots__ = ("id", "name", "description", "reserved", "created_at", "retired_at")
    ID_FIELD_NUMBER: _ClassVar[int]
    NAME_FIELD_NUMBER: _ClassVar[int]
    DESCRIPTION_FIELD_NUMBER: _ClassVar[int]
    RESERVED_FIELD_NUMBER: _ClassVar[int]
    CREATED_AT_FIELD_NUMBER: _ClassVar[int]
    RETIRED_AT_FIELD_NUMBER: _ClassVar[int]
    id: str
    name: str
    description: str
    reserved: bool
    created_at: _timestamp_pb2.Timestamp
    retired_at: _timestamp_pb2.Timestamp
    def __init__(self, id: _Optional[str] = ..., name: _Optional[str] = ..., description: _Optional[str] = ..., reserved: _Optional[bool] = ..., created_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., retired_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ...) -> None: ...

class Classification(_message.Message):
    __slots__ = ("id", "signal_id", "proposed_category_id", "proposed_confidence", "model", "confirmed_category_id", "state", "reason", "created_at")
    ID_FIELD_NUMBER: _ClassVar[int]
    SIGNAL_ID_FIELD_NUMBER: _ClassVar[int]
    PROPOSED_CATEGORY_ID_FIELD_NUMBER: _ClassVar[int]
    PROPOSED_CONFIDENCE_FIELD_NUMBER: _ClassVar[int]
    MODEL_FIELD_NUMBER: _ClassVar[int]
    CONFIRMED_CATEGORY_ID_FIELD_NUMBER: _ClassVar[int]
    STATE_FIELD_NUMBER: _ClassVar[int]
    REASON_FIELD_NUMBER: _ClassVar[int]
    CREATED_AT_FIELD_NUMBER: _ClassVar[int]
    id: str
    signal_id: str
    proposed_category_id: str
    proposed_confidence: float
    model: str
    confirmed_category_id: str
    state: ClassificationState
    reason: str
    created_at: _timestamp_pb2.Timestamp
    def __init__(self, id: _Optional[str] = ..., signal_id: _Optional[str] = ..., proposed_category_id: _Optional[str] = ..., proposed_confidence: _Optional[float] = ..., model: _Optional[str] = ..., confirmed_category_id: _Optional[str] = ..., state: _Optional[_Union[ClassificationState, str]] = ..., reason: _Optional[str] = ..., created_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ...) -> None: ...

class CreateCategoryRequest(_message.Message):
    __slots__ = ("name", "description")
    NAME_FIELD_NUMBER: _ClassVar[int]
    DESCRIPTION_FIELD_NUMBER: _ClassVar[int]
    name: str
    description: str
    def __init__(self, name: _Optional[str] = ..., description: _Optional[str] = ...) -> None: ...

class CreateCategoryResponse(_message.Message):
    __slots__ = ("category",)
    CATEGORY_FIELD_NUMBER: _ClassVar[int]
    category: Category
    def __init__(self, category: _Optional[_Union[Category, _Mapping]] = ...) -> None: ...

class ListCategoriesRequest(_message.Message):
    __slots__ = ("include_retired",)
    INCLUDE_RETIRED_FIELD_NUMBER: _ClassVar[int]
    include_retired: bool
    def __init__(self, include_retired: _Optional[bool] = ...) -> None: ...

class ListCategoriesResponse(_message.Message):
    __slots__ = ("categories",)
    CATEGORIES_FIELD_NUMBER: _ClassVar[int]
    categories: _containers.RepeatedCompositeFieldContainer[Category]
    def __init__(self, categories: _Optional[_Iterable[_Union[Category, _Mapping]]] = ...) -> None: ...

class RenameCategoryRequest(_message.Message):
    __slots__ = ("id", "name", "description")
    ID_FIELD_NUMBER: _ClassVar[int]
    NAME_FIELD_NUMBER: _ClassVar[int]
    DESCRIPTION_FIELD_NUMBER: _ClassVar[int]
    id: str
    name: str
    description: str
    def __init__(self, id: _Optional[str] = ..., name: _Optional[str] = ..., description: _Optional[str] = ...) -> None: ...

class RenameCategoryResponse(_message.Message):
    __slots__ = ("category",)
    CATEGORY_FIELD_NUMBER: _ClassVar[int]
    category: Category
    def __init__(self, category: _Optional[_Union[Category, _Mapping]] = ...) -> None: ...

class RetireCategoryRequest(_message.Message):
    __slots__ = ("id",)
    ID_FIELD_NUMBER: _ClassVar[int]
    id: str
    def __init__(self, id: _Optional[str] = ...) -> None: ...

class RetireCategoryResponse(_message.Message):
    __slots__ = ("category",)
    CATEGORY_FIELD_NUMBER: _ClassVar[int]
    category: Category
    def __init__(self, category: _Optional[_Union[Category, _Mapping]] = ...) -> None: ...

class GetClassificationRequest(_message.Message):
    __slots__ = ("signal_id",)
    SIGNAL_ID_FIELD_NUMBER: _ClassVar[int]
    signal_id: str
    def __init__(self, signal_id: _Optional[str] = ...) -> None: ...

class GetClassificationResponse(_message.Message):
    __slots__ = ("classification",)
    CLASSIFICATION_FIELD_NUMBER: _ClassVar[int]
    classification: Classification
    def __init__(self, classification: _Optional[_Union[Classification, _Mapping]] = ...) -> None: ...

class ConfirmClassificationRequest(_message.Message):
    __slots__ = ("signal_id", "category_id")
    SIGNAL_ID_FIELD_NUMBER: _ClassVar[int]
    CATEGORY_ID_FIELD_NUMBER: _ClassVar[int]
    signal_id: str
    category_id: str
    def __init__(self, signal_id: _Optional[str] = ..., category_id: _Optional[str] = ...) -> None: ...

class ConfirmClassificationResponse(_message.Message):
    __slots__ = ("classification",)
    CLASSIFICATION_FIELD_NUMBER: _ClassVar[int]
    classification: Classification
    def __init__(self, classification: _Optional[_Union[Classification, _Mapping]] = ...) -> None: ...
