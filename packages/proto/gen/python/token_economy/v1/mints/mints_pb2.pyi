import datetime

from google.protobuf import timestamp_pb2 as _timestamp_pb2
from google.protobuf.internal import containers as _containers
from google.protobuf.internal import enum_type_wrapper as _enum_type_wrapper
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Iterable as _Iterable, Mapping as _Mapping
from typing import ClassVar as _ClassVar, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class SupplyPolicy(int, metaclass=_enum_type_wrapper.EnumTypeWrapper):
    __slots__ = ()
    SUPPLY_POLICY_UNSPECIFIED: _ClassVar[SupplyPolicy]
    SUPPLY_POLICY_UNBOUNDED: _ClassVar[SupplyPolicy]
    SUPPLY_POLICY_CAPPED: _ClassVar[SupplyPolicy]
    SUPPLY_POLICY_FIXED: _ClassVar[SupplyPolicy]
SUPPLY_POLICY_UNSPECIFIED: SupplyPolicy
SUPPLY_POLICY_UNBOUNDED: SupplyPolicy
SUPPLY_POLICY_CAPPED: SupplyPolicy
SUPPLY_POLICY_FIXED: SupplyPolicy

class MinterAuthority(_message.Message):
    __slots__ = ("token_type_id", "subject")
    TOKEN_TYPE_ID_FIELD_NUMBER: _ClassVar[int]
    SUBJECT_FIELD_NUMBER: _ClassVar[int]
    token_type_id: str
    subject: str
    def __init__(self, token_type_id: _Optional[str] = ..., subject: _Optional[str] = ...) -> None: ...

class TokenType(_message.Message):
    __slots__ = ("id", "name", "symbol", "color", "supply_policy", "cap_amount", "minted_amount", "authority", "retired", "created_at", "retired_at")
    ID_FIELD_NUMBER: _ClassVar[int]
    NAME_FIELD_NUMBER: _ClassVar[int]
    SYMBOL_FIELD_NUMBER: _ClassVar[int]
    COLOR_FIELD_NUMBER: _ClassVar[int]
    SUPPLY_POLICY_FIELD_NUMBER: _ClassVar[int]
    CAP_AMOUNT_FIELD_NUMBER: _ClassVar[int]
    MINTED_AMOUNT_FIELD_NUMBER: _ClassVar[int]
    AUTHORITY_FIELD_NUMBER: _ClassVar[int]
    RETIRED_FIELD_NUMBER: _ClassVar[int]
    CREATED_AT_FIELD_NUMBER: _ClassVar[int]
    RETIRED_AT_FIELD_NUMBER: _ClassVar[int]
    id: str
    name: str
    symbol: str
    color: str
    supply_policy: SupplyPolicy
    cap_amount: int
    minted_amount: int
    authority: MinterAuthority
    retired: bool
    created_at: _timestamp_pb2.Timestamp
    retired_at: _timestamp_pb2.Timestamp
    def __init__(self, id: _Optional[str] = ..., name: _Optional[str] = ..., symbol: _Optional[str] = ..., color: _Optional[str] = ..., supply_policy: _Optional[_Union[SupplyPolicy, str]] = ..., cap_amount: _Optional[int] = ..., minted_amount: _Optional[int] = ..., authority: _Optional[_Union[MinterAuthority, _Mapping]] = ..., retired: _Optional[bool] = ..., created_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., retired_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ...) -> None: ...

class CreateTokenTypeRequest(_message.Message):
    __slots__ = ("name", "symbol", "color", "supply_policy", "cap_amount", "minter_subject")
    NAME_FIELD_NUMBER: _ClassVar[int]
    SYMBOL_FIELD_NUMBER: _ClassVar[int]
    COLOR_FIELD_NUMBER: _ClassVar[int]
    SUPPLY_POLICY_FIELD_NUMBER: _ClassVar[int]
    CAP_AMOUNT_FIELD_NUMBER: _ClassVar[int]
    MINTER_SUBJECT_FIELD_NUMBER: _ClassVar[int]
    name: str
    symbol: str
    color: str
    supply_policy: SupplyPolicy
    cap_amount: int
    minter_subject: str
    def __init__(self, name: _Optional[str] = ..., symbol: _Optional[str] = ..., color: _Optional[str] = ..., supply_policy: _Optional[_Union[SupplyPolicy, str]] = ..., cap_amount: _Optional[int] = ..., minter_subject: _Optional[str] = ...) -> None: ...

class CreateTokenTypeResponse(_message.Message):
    __slots__ = ("token_type",)
    TOKEN_TYPE_FIELD_NUMBER: _ClassVar[int]
    token_type: TokenType
    def __init__(self, token_type: _Optional[_Union[TokenType, _Mapping]] = ...) -> None: ...

class GetTokenTypeRequest(_message.Message):
    __slots__ = ("id",)
    ID_FIELD_NUMBER: _ClassVar[int]
    id: str
    def __init__(self, id: _Optional[str] = ...) -> None: ...

class GetTokenTypeResponse(_message.Message):
    __slots__ = ("token_type",)
    TOKEN_TYPE_FIELD_NUMBER: _ClassVar[int]
    token_type: TokenType
    def __init__(self, token_type: _Optional[_Union[TokenType, _Mapping]] = ...) -> None: ...

class ListTokenTypesRequest(_message.Message):
    __slots__ = ("include_retired",)
    INCLUDE_RETIRED_FIELD_NUMBER: _ClassVar[int]
    include_retired: bool
    def __init__(self, include_retired: _Optional[bool] = ...) -> None: ...

class ListTokenTypesResponse(_message.Message):
    __slots__ = ("token_types",)
    TOKEN_TYPES_FIELD_NUMBER: _ClassVar[int]
    token_types: _containers.RepeatedCompositeFieldContainer[TokenType]
    def __init__(self, token_types: _Optional[_Iterable[_Union[TokenType, _Mapping]]] = ...) -> None: ...

class RetireTokenTypeRequest(_message.Message):
    __slots__ = ("id",)
    ID_FIELD_NUMBER: _ClassVar[int]
    id: str
    def __init__(self, id: _Optional[str] = ...) -> None: ...

class RetireTokenTypeResponse(_message.Message):
    __slots__ = ("token_type",)
    TOKEN_TYPE_FIELD_NUMBER: _ClassVar[int]
    token_type: TokenType
    def __init__(self, token_type: _Optional[_Union[TokenType, _Mapping]] = ...) -> None: ...

class MintSupplyRequest(_message.Message):
    __slots__ = ("token_type_id", "amount")
    TOKEN_TYPE_ID_FIELD_NUMBER: _ClassVar[int]
    AMOUNT_FIELD_NUMBER: _ClassVar[int]
    token_type_id: str
    amount: int
    def __init__(self, token_type_id: _Optional[str] = ..., amount: _Optional[int] = ...) -> None: ...

class MintSupplyResponse(_message.Message):
    __slots__ = ("token_type",)
    TOKEN_TYPE_FIELD_NUMBER: _ClassVar[int]
    token_type: TokenType
    def __init__(self, token_type: _Optional[_Union[TokenType, _Mapping]] = ...) -> None: ...
