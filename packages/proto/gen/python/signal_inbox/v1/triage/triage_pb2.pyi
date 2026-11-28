import datetime

from google.protobuf import timestamp_pb2 as _timestamp_pb2
from google.protobuf.internal import containers as _containers
from google.protobuf.internal import enum_type_wrapper as _enum_type_wrapper
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Iterable as _Iterable, Mapping as _Mapping
from typing import ClassVar as _ClassVar, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class DispositionState(int, metaclass=_enum_type_wrapper.EnumTypeWrapper):
    __slots__ = ()
    DISPOSITION_STATE_UNSPECIFIED: _ClassVar[DispositionState]
    DISPOSITION_STATE_NEW: _ClassVar[DispositionState]
    DISPOSITION_STATE_TRIAGED: _ClassVar[DispositionState]
    DISPOSITION_STATE_ROUTED: _ClassVar[DispositionState]
    DISPOSITION_STATE_DONE: _ClassVar[DispositionState]
    DISPOSITION_STATE_DROPPED: _ClassVar[DispositionState]

class AnnotationAuthor(int, metaclass=_enum_type_wrapper.EnumTypeWrapper):
    __slots__ = ()
    ANNOTATION_AUTHOR_UNSPECIFIED: _ClassVar[AnnotationAuthor]
    ANNOTATION_AUTHOR_OPERATOR: _ClassVar[AnnotationAuthor]
    ANNOTATION_AUTHOR_AGENT: _ClassVar[AnnotationAuthor]
    ANNOTATION_AUTHOR_SYSTEM: _ClassVar[AnnotationAuthor]

class OutcomeKind(int, metaclass=_enum_type_wrapper.EnumTypeWrapper):
    __slots__ = ()
    OUTCOME_KIND_UNSPECIFIED: _ClassVar[OutcomeKind]
    OUTCOME_KIND_SCENARIO: _ClassVar[OutcomeKind]
    OUTCOME_KIND_BACKLOG: _ClassVar[OutcomeKind]
    OUTCOME_KIND_IDEA_PIPELINE: _ClassVar[OutcomeKind]
    OUTCOME_KIND_KNOWLEDGE_TOPIC: _ClassVar[OutcomeKind]
DISPOSITION_STATE_UNSPECIFIED: DispositionState
DISPOSITION_STATE_NEW: DispositionState
DISPOSITION_STATE_TRIAGED: DispositionState
DISPOSITION_STATE_ROUTED: DispositionState
DISPOSITION_STATE_DONE: DispositionState
DISPOSITION_STATE_DROPPED: DispositionState
ANNOTATION_AUTHOR_UNSPECIFIED: AnnotationAuthor
ANNOTATION_AUTHOR_OPERATOR: AnnotationAuthor
ANNOTATION_AUTHOR_AGENT: AnnotationAuthor
ANNOTATION_AUTHOR_SYSTEM: AnnotationAuthor
OUTCOME_KIND_UNSPECIFIED: OutcomeKind
OUTCOME_KIND_SCENARIO: OutcomeKind
OUTCOME_KIND_BACKLOG: OutcomeKind
OUTCOME_KIND_IDEA_PIPELINE: OutcomeKind
OUTCOME_KIND_KNOWLEDGE_TOPIC: OutcomeKind

class Disposition(_message.Message):
    __slots__ = ("signal_id", "state", "revisit_at", "updated_at")
    SIGNAL_ID_FIELD_NUMBER: _ClassVar[int]
    STATE_FIELD_NUMBER: _ClassVar[int]
    REVISIT_AT_FIELD_NUMBER: _ClassVar[int]
    UPDATED_AT_FIELD_NUMBER: _ClassVar[int]
    signal_id: str
    state: DispositionState
    revisit_at: _timestamp_pb2.Timestamp
    updated_at: _timestamp_pb2.Timestamp
    def __init__(self, signal_id: _Optional[str] = ..., state: _Optional[_Union[DispositionState, str]] = ..., revisit_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., updated_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ...) -> None: ...

class OutcomeLink(_message.Message):
    __slots__ = ("kind", "target_id")
    KIND_FIELD_NUMBER: _ClassVar[int]
    TARGET_ID_FIELD_NUMBER: _ClassVar[int]
    kind: OutcomeKind
    target_id: str
    def __init__(self, kind: _Optional[_Union[OutcomeKind, str]] = ..., target_id: _Optional[str] = ...) -> None: ...

class Annotation(_message.Message):
    __slots__ = ("id", "signal_id", "author", "body", "outcome", "created_at")
    ID_FIELD_NUMBER: _ClassVar[int]
    SIGNAL_ID_FIELD_NUMBER: _ClassVar[int]
    AUTHOR_FIELD_NUMBER: _ClassVar[int]
    BODY_FIELD_NUMBER: _ClassVar[int]
    OUTCOME_FIELD_NUMBER: _ClassVar[int]
    CREATED_AT_FIELD_NUMBER: _ClassVar[int]
    id: str
    signal_id: str
    author: AnnotationAuthor
    body: str
    outcome: OutcomeLink
    created_at: _timestamp_pb2.Timestamp
    def __init__(self, id: _Optional[str] = ..., signal_id: _Optional[str] = ..., author: _Optional[_Union[AnnotationAuthor, str]] = ..., body: _Optional[str] = ..., outcome: _Optional[_Union[OutcomeLink, _Mapping]] = ..., created_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ...) -> None: ...

class TriageRecord(_message.Message):
    __slots__ = ("disposition", "annotations")
    DISPOSITION_FIELD_NUMBER: _ClassVar[int]
    ANNOTATIONS_FIELD_NUMBER: _ClassVar[int]
    disposition: Disposition
    annotations: _containers.RepeatedCompositeFieldContainer[Annotation]
    def __init__(self, disposition: _Optional[_Union[Disposition, _Mapping]] = ..., annotations: _Optional[_Iterable[_Union[Annotation, _Mapping]]] = ...) -> None: ...

class GetTriageRequest(_message.Message):
    __slots__ = ("signal_id",)
    SIGNAL_ID_FIELD_NUMBER: _ClassVar[int]
    signal_id: str
    def __init__(self, signal_id: _Optional[str] = ...) -> None: ...

class GetTriageResponse(_message.Message):
    __slots__ = ("triage",)
    TRIAGE_FIELD_NUMBER: _ClassVar[int]
    triage: TriageRecord
    def __init__(self, triage: _Optional[_Union[TriageRecord, _Mapping]] = ...) -> None: ...

class SetDispositionRequest(_message.Message):
    __slots__ = ("signal_id", "state", "revisit_at")
    SIGNAL_ID_FIELD_NUMBER: _ClassVar[int]
    STATE_FIELD_NUMBER: _ClassVar[int]
    REVISIT_AT_FIELD_NUMBER: _ClassVar[int]
    signal_id: str
    state: DispositionState
    revisit_at: _timestamp_pb2.Timestamp
    def __init__(self, signal_id: _Optional[str] = ..., state: _Optional[_Union[DispositionState, str]] = ..., revisit_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ...) -> None: ...

class SetDispositionResponse(_message.Message):
    __slots__ = ("disposition",)
    DISPOSITION_FIELD_NUMBER: _ClassVar[int]
    disposition: Disposition
    def __init__(self, disposition: _Optional[_Union[Disposition, _Mapping]] = ...) -> None: ...

class AddAnnotationRequest(_message.Message):
    __slots__ = ("signal_id", "author", "body", "outcome")
    SIGNAL_ID_FIELD_NUMBER: _ClassVar[int]
    AUTHOR_FIELD_NUMBER: _ClassVar[int]
    BODY_FIELD_NUMBER: _ClassVar[int]
    OUTCOME_FIELD_NUMBER: _ClassVar[int]
    signal_id: str
    author: AnnotationAuthor
    body: str
    outcome: OutcomeLink
    def __init__(self, signal_id: _Optional[str] = ..., author: _Optional[_Union[AnnotationAuthor, str]] = ..., body: _Optional[str] = ..., outcome: _Optional[_Union[OutcomeLink, _Mapping]] = ...) -> None: ...

class AddAnnotationResponse(_message.Message):
    __slots__ = ("annotation",)
    ANNOTATION_FIELD_NUMBER: _ClassVar[int]
    annotation: Annotation
    def __init__(self, annotation: _Optional[_Union[Annotation, _Mapping]] = ...) -> None: ...
