from architecture_cartographer.v1.shared import shared_pb2 as _shared_pb2
from google.protobuf.internal import containers as _containers
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Iterable as _Iterable, Mapping as _Mapping
from typing import ClassVar as _ClassVar, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class DetectorDescriptor(_message.Message):
    __slots__ = ("name", "description", "stability", "emits_types", "finding_class")
    NAME_FIELD_NUMBER: _ClassVar[int]
    DESCRIPTION_FIELD_NUMBER: _ClassVar[int]
    STABILITY_FIELD_NUMBER: _ClassVar[int]
    EMITS_TYPES_FIELD_NUMBER: _ClassVar[int]
    FINDING_CLASS_FIELD_NUMBER: _ClassVar[int]
    name: str
    description: str
    stability: str
    emits_types: _containers.RepeatedScalarFieldContainer[str]
    finding_class: _shared_pb2.FindingClass
    def __init__(self, name: _Optional[str] = ..., description: _Optional[str] = ..., stability: _Optional[str] = ..., emits_types: _Optional[_Iterable[str]] = ..., finding_class: _Optional[_Union[_shared_pb2.FindingClass, str]] = ...) -> None: ...

class ResolverDescriptor(_message.Message):
    __slots__ = ("name", "description", "stability", "handles_kinds", "requires_apply")
    NAME_FIELD_NUMBER: _ClassVar[int]
    DESCRIPTION_FIELD_NUMBER: _ClassVar[int]
    STABILITY_FIELD_NUMBER: _ClassVar[int]
    HANDLES_KINDS_FIELD_NUMBER: _ClassVar[int]
    REQUIRES_APPLY_FIELD_NUMBER: _ClassVar[int]
    name: str
    description: str
    stability: str
    handles_kinds: _containers.RepeatedScalarFieldContainer[_shared_pb2.FixKind]
    requires_apply: bool
    def __init__(self, name: _Optional[str] = ..., description: _Optional[str] = ..., stability: _Optional[str] = ..., handles_kinds: _Optional[_Iterable[_Union[_shared_pb2.FixKind, str]]] = ..., requires_apply: _Optional[bool] = ...) -> None: ...

class DetectConflictsRequest(_message.Message):
    __slots__ = ("scenario", "snapshot_id", "idempotency_key")
    SCENARIO_FIELD_NUMBER: _ClassVar[int]
    SNAPSHOT_ID_FIELD_NUMBER: _ClassVar[int]
    IDEMPOTENCY_KEY_FIELD_NUMBER: _ClassVar[int]
    scenario: str
    snapshot_id: str
    idempotency_key: str
    def __init__(self, scenario: _Optional[str] = ..., snapshot_id: _Optional[str] = ..., idempotency_key: _Optional[str] = ...) -> None: ...

class DetectConflictsResponse(_message.Message):
    __slots__ = ("conflicts",)
    CONFLICTS_FIELD_NUMBER: _ClassVar[int]
    conflicts: _containers.RepeatedCompositeFieldContainer[_shared_pb2.Conflict]
    def __init__(self, conflicts: _Optional[_Iterable[_Union[_shared_pb2.Conflict, _Mapping]]] = ...) -> None: ...

class ListConflictsRequest(_message.Message):
    __slots__ = ("scenario", "types", "page_size", "page_token")
    SCENARIO_FIELD_NUMBER: _ClassVar[int]
    TYPES_FIELD_NUMBER: _ClassVar[int]
    PAGE_SIZE_FIELD_NUMBER: _ClassVar[int]
    PAGE_TOKEN_FIELD_NUMBER: _ClassVar[int]
    scenario: str
    types: _containers.RepeatedScalarFieldContainer[str]
    page_size: int
    page_token: str
    def __init__(self, scenario: _Optional[str] = ..., types: _Optional[_Iterable[str]] = ..., page_size: _Optional[int] = ..., page_token: _Optional[str] = ...) -> None: ...

class ListConflictsResponse(_message.Message):
    __slots__ = ("conflicts", "next_page_token")
    CONFLICTS_FIELD_NUMBER: _ClassVar[int]
    NEXT_PAGE_TOKEN_FIELD_NUMBER: _ClassVar[int]
    conflicts: _containers.RepeatedCompositeFieldContainer[_shared_pb2.Conflict]
    next_page_token: str
    def __init__(self, conflicts: _Optional[_Iterable[_Union[_shared_pb2.Conflict, _Mapping]]] = ..., next_page_token: _Optional[str] = ...) -> None: ...

class GetConflictRequest(_message.Message):
    __slots__ = ("id",)
    ID_FIELD_NUMBER: _ClassVar[int]
    id: str
    def __init__(self, id: _Optional[str] = ...) -> None: ...

class GetConflictResponse(_message.Message):
    __slots__ = ("conflict",)
    CONFLICT_FIELD_NUMBER: _ClassVar[int]
    conflict: _shared_pb2.Conflict
    def __init__(self, conflict: _Optional[_Union[_shared_pb2.Conflict, _Mapping]] = ...) -> None: ...

class ValidateConflictsRequest(_message.Message):
    __slots__ = ("scenario",)
    SCENARIO_FIELD_NUMBER: _ClassVar[int]
    scenario: str
    def __init__(self, scenario: _Optional[str] = ...) -> None: ...

class ValidateConflictsResponse(_message.Message):
    __slots__ = ("conflicts", "clean")
    CONFLICTS_FIELD_NUMBER: _ClassVar[int]
    CLEAN_FIELD_NUMBER: _ClassVar[int]
    conflicts: _containers.RepeatedCompositeFieldContainer[_shared_pb2.Conflict]
    clean: bool
    def __init__(self, conflicts: _Optional[_Iterable[_Union[_shared_pb2.Conflict, _Mapping]]] = ..., clean: _Optional[bool] = ...) -> None: ...

class ListDetectorsRequest(_message.Message):
    __slots__ = ()
    def __init__(self) -> None: ...

class ListDetectorsResponse(_message.Message):
    __slots__ = ("detectors",)
    DETECTORS_FIELD_NUMBER: _ClassVar[int]
    detectors: _containers.RepeatedCompositeFieldContainer[DetectorDescriptor]
    def __init__(self, detectors: _Optional[_Iterable[_Union[DetectorDescriptor, _Mapping]]] = ...) -> None: ...

class ListResolversRequest(_message.Message):
    __slots__ = ()
    def __init__(self) -> None: ...

class ListResolversResponse(_message.Message):
    __slots__ = ("resolvers",)
    RESOLVERS_FIELD_NUMBER: _ClassVar[int]
    resolvers: _containers.RepeatedCompositeFieldContainer[ResolverDescriptor]
    def __init__(self, resolvers: _Optional[_Iterable[_Union[ResolverDescriptor, _Mapping]]] = ...) -> None: ...
