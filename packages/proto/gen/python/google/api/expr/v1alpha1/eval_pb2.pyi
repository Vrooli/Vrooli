from google.api.expr.v1alpha1 import value_pb2 as _value_pb2
from google.rpc import status_pb2 as _status_pb2
from google.protobuf.internal import containers as _containers
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Iterable as _Iterable, Mapping as _Mapping
from typing import ClassVar as _ClassVar, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class EvalState(_message.Message):
    __slots__ = ()
    class Result(_message.Message):
        __slots__ = ()
        EXPR_FIELD_NUMBER: _ClassVar[int]
        VALUE_FIELD_NUMBER: _ClassVar[int]
        expr: int
        value: int
        def __init__(self, expr: _Optional[int] = ..., value: _Optional[int] = ...) -> None: ...
    VALUES_FIELD_NUMBER: _ClassVar[int]
    RESULTS_FIELD_NUMBER: _ClassVar[int]
    values: _containers.RepeatedCompositeFieldContainer[ExprValue]
    results: _containers.RepeatedCompositeFieldContainer[EvalState.Result]
    def __init__(self, values: _Optional[_Iterable[_Union[ExprValue, _Mapping]]] = ..., results: _Optional[_Iterable[_Union[EvalState.Result, _Mapping]]] = ...) -> None: ...

class ExprValue(_message.Message):
    __slots__ = ()
    VALUE_FIELD_NUMBER: _ClassVar[int]
    ERROR_FIELD_NUMBER: _ClassVar[int]
    UNKNOWN_FIELD_NUMBER: _ClassVar[int]
    value: _value_pb2.Value
    error: ErrorSet
    unknown: UnknownSet
    def __init__(self, value: _Optional[_Union[_value_pb2.Value, _Mapping]] = ..., error: _Optional[_Union[ErrorSet, _Mapping]] = ..., unknown: _Optional[_Union[UnknownSet, _Mapping]] = ...) -> None: ...

class ErrorSet(_message.Message):
    __slots__ = ()
    ERRORS_FIELD_NUMBER: _ClassVar[int]
    errors: _containers.RepeatedCompositeFieldContainer[_status_pb2.Status]
    def __init__(self, errors: _Optional[_Iterable[_Union[_status_pb2.Status, _Mapping]]] = ...) -> None: ...

class UnknownSet(_message.Message):
    __slots__ = ()
    EXPRS_FIELD_NUMBER: _ClassVar[int]
    exprs: _containers.RepeatedScalarFieldContainer[int]
    def __init__(self, exprs: _Optional[_Iterable[int]] = ...) -> None: ...
