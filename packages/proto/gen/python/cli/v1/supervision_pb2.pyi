from google.protobuf.internal import containers as _containers
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Iterable as _Iterable, Mapping as _Mapping
from typing import ClassVar as _ClassVar, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class SupervisionAttributionStep(_message.Message):
    __slots__ = ("name", "kind", "declared_by", "supervision_intent", "source")
    NAME_FIELD_NUMBER: _ClassVar[int]
    KIND_FIELD_NUMBER: _ClassVar[int]
    DECLARED_BY_FIELD_NUMBER: _ClassVar[int]
    SUPERVISION_INTENT_FIELD_NUMBER: _ClassVar[int]
    SOURCE_FIELD_NUMBER: _ClassVar[int]
    name: str
    kind: str
    declared_by: str
    supervision_intent: str
    source: str
    def __init__(self, name: _Optional[str] = ..., kind: _Optional[str] = ..., declared_by: _Optional[str] = ..., supervision_intent: _Optional[str] = ..., source: _Optional[str] = ...) -> None: ...

class SupervisionMember(_message.Message):
    __slots__ = ("name", "kind", "supervision_intent", "attribution_chain")
    NAME_FIELD_NUMBER: _ClassVar[int]
    KIND_FIELD_NUMBER: _ClassVar[int]
    SUPERVISION_INTENT_FIELD_NUMBER: _ClassVar[int]
    ATTRIBUTION_CHAIN_FIELD_NUMBER: _ClassVar[int]
    name: str
    kind: str
    supervision_intent: str
    attribution_chain: _containers.RepeatedCompositeFieldContainer[SupervisionAttributionStep]
    def __init__(self, name: _Optional[str] = ..., kind: _Optional[str] = ..., supervision_intent: _Optional[str] = ..., attribution_chain: _Optional[_Iterable[_Union[SupervisionAttributionStep, _Mapping]]] = ...) -> None: ...

class SupervisionSetResponse(_message.Message):
    __slots__ = ("source", "core_set", "seed", "added_by_closure", "trusted_base", "members", "member_counts", "load_errors")
    class MemberCountsEntry(_message.Message):
        __slots__ = ("key", "value")
        KEY_FIELD_NUMBER: _ClassVar[int]
        VALUE_FIELD_NUMBER: _ClassVar[int]
        key: str
        value: int
        def __init__(self, key: _Optional[str] = ..., value: _Optional[int] = ...) -> None: ...
    class LoadErrorsEntry(_message.Message):
        __slots__ = ("key", "value")
        KEY_FIELD_NUMBER: _ClassVar[int]
        VALUE_FIELD_NUMBER: _ClassVar[int]
        key: str
        value: str
        def __init__(self, key: _Optional[str] = ..., value: _Optional[str] = ...) -> None: ...
    SOURCE_FIELD_NUMBER: _ClassVar[int]
    CORE_SET_FIELD_NUMBER: _ClassVar[int]
    SEED_FIELD_NUMBER: _ClassVar[int]
    ADDED_BY_CLOSURE_FIELD_NUMBER: _ClassVar[int]
    TRUSTED_BASE_FIELD_NUMBER: _ClassVar[int]
    MEMBERS_FIELD_NUMBER: _ClassVar[int]
    MEMBER_COUNTS_FIELD_NUMBER: _ClassVar[int]
    LOAD_ERRORS_FIELD_NUMBER: _ClassVar[int]
    source: str
    core_set: _containers.RepeatedScalarFieldContainer[str]
    seed: _containers.RepeatedScalarFieldContainer[str]
    added_by_closure: _containers.RepeatedScalarFieldContainer[str]
    trusted_base: _containers.RepeatedScalarFieldContainer[str]
    members: _containers.RepeatedCompositeFieldContainer[SupervisionMember]
    member_counts: _containers.ScalarMap[str, int]
    load_errors: _containers.ScalarMap[str, str]
    def __init__(self, source: _Optional[str] = ..., core_set: _Optional[_Iterable[str]] = ..., seed: _Optional[_Iterable[str]] = ..., added_by_closure: _Optional[_Iterable[str]] = ..., trusted_base: _Optional[_Iterable[str]] = ..., members: _Optional[_Iterable[_Union[SupervisionMember, _Mapping]]] = ..., member_counts: _Optional[_Mapping[str, int]] = ..., load_errors: _Optional[_Mapping[str, str]] = ...) -> None: ...
