from google.api.expr.v1beta1 import expr_pb2 as _expr_pb2
from google.protobuf.internal import containers as _containers
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Iterable as _Iterable, Mapping as _Mapping
from typing import ClassVar as _ClassVar, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class Decl(_message.Message):
    __slots__ = ()
    ID_FIELD_NUMBER: _ClassVar[int]
    NAME_FIELD_NUMBER: _ClassVar[int]
    DOC_FIELD_NUMBER: _ClassVar[int]
    IDENT_FIELD_NUMBER: _ClassVar[int]
    FUNCTION_FIELD_NUMBER: _ClassVar[int]
    id: int
    name: str
    doc: str
    ident: IdentDecl
    function: FunctionDecl
    def __init__(self, id: _Optional[int] = ..., name: _Optional[str] = ..., doc: _Optional[str] = ..., ident: _Optional[_Union[IdentDecl, _Mapping]] = ..., function: _Optional[_Union[FunctionDecl, _Mapping]] = ...) -> None: ...

class DeclType(_message.Message):
    __slots__ = ()
    ID_FIELD_NUMBER: _ClassVar[int]
    TYPE_FIELD_NUMBER: _ClassVar[int]
    TYPE_PARAMS_FIELD_NUMBER: _ClassVar[int]
    id: int
    type: str
    type_params: _containers.RepeatedCompositeFieldContainer[DeclType]
    def __init__(self, id: _Optional[int] = ..., type: _Optional[str] = ..., type_params: _Optional[_Iterable[_Union[DeclType, _Mapping]]] = ...) -> None: ...

class IdentDecl(_message.Message):
    __slots__ = ()
    TYPE_FIELD_NUMBER: _ClassVar[int]
    VALUE_FIELD_NUMBER: _ClassVar[int]
    type: DeclType
    value: _expr_pb2.Expr
    def __init__(self, type: _Optional[_Union[DeclType, _Mapping]] = ..., value: _Optional[_Union[_expr_pb2.Expr, _Mapping]] = ...) -> None: ...

class FunctionDecl(_message.Message):
    __slots__ = ()
    ARGS_FIELD_NUMBER: _ClassVar[int]
    RETURN_TYPE_FIELD_NUMBER: _ClassVar[int]
    RECEIVER_FUNCTION_FIELD_NUMBER: _ClassVar[int]
    args: _containers.RepeatedCompositeFieldContainer[IdentDecl]
    return_type: DeclType
    receiver_function: bool
    def __init__(self, args: _Optional[_Iterable[_Union[IdentDecl, _Mapping]]] = ..., return_type: _Optional[_Union[DeclType, _Mapping]] = ..., receiver_function: _Optional[bool] = ...) -> None: ...
