from google.api.expr.v1alpha1 import value_pb2 as _value_pb2
from google.protobuf.internal import containers as _containers
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Iterable as _Iterable, Mapping as _Mapping
from typing import ClassVar as _ClassVar, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class Explain(_message.Message):
    __slots__ = ()
    class ExprStep(_message.Message):
        __slots__ = ()
        ID_FIELD_NUMBER: _ClassVar[int]
        VALUE_INDEX_FIELD_NUMBER: _ClassVar[int]
        id: int
        value_index: int
        def __init__(self, id: _Optional[int] = ..., value_index: _Optional[int] = ...) -> None: ...
    VALUES_FIELD_NUMBER: _ClassVar[int]
    EXPR_STEPS_FIELD_NUMBER: _ClassVar[int]
    values: _containers.RepeatedCompositeFieldContainer[_value_pb2.Value]
    expr_steps: _containers.RepeatedCompositeFieldContainer[Explain.ExprStep]
    def __init__(self, values: _Optional[_Iterable[_Union[_value_pb2.Value, _Mapping]]] = ..., expr_steps: _Optional[_Iterable[_Union[Explain.ExprStep, _Mapping]]] = ...) -> None: ...
