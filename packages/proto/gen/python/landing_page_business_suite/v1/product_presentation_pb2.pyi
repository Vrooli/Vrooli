from landing_page_business_suite.v1.shared import product_presentation_pb2 as _product_presentation_pb2
from google.protobuf.internal import containers as _containers
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Iterable as _Iterable, Mapping as _Mapping
from typing import ClassVar as _ClassVar, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class PresentationRevisionState(_message.Message):
    __slots__ = ("generation", "draft_revision", "active_revision", "published_revisions")
    GENERATION_FIELD_NUMBER: _ClassVar[int]
    DRAFT_REVISION_FIELD_NUMBER: _ClassVar[int]
    ACTIVE_REVISION_FIELD_NUMBER: _ClassVar[int]
    PUBLISHED_REVISIONS_FIELD_NUMBER: _ClassVar[int]
    generation: int
    draft_revision: str
    active_revision: str
    published_revisions: _containers.RepeatedScalarFieldContainer[str]
    def __init__(self, generation: _Optional[int] = ..., draft_revision: _Optional[str] = ..., active_revision: _Optional[str] = ..., published_revisions: _Optional[_Iterable[str]] = ...) -> None: ...

class GetPresentationRequest(_message.Message):
    __slots__ = ("variant_slug", "revision")
    VARIANT_SLUG_FIELD_NUMBER: _ClassVar[int]
    REVISION_FIELD_NUMBER: _ClassVar[int]
    variant_slug: str
    revision: str
    def __init__(self, variant_slug: _Optional[str] = ..., revision: _Optional[str] = ...) -> None: ...

class PresentationEditorResponse(_message.Message):
    __slots__ = ("variant_slug", "state", "revision", "document")
    VARIANT_SLUG_FIELD_NUMBER: _ClassVar[int]
    STATE_FIELD_NUMBER: _ClassVar[int]
    REVISION_FIELD_NUMBER: _ClassVar[int]
    DOCUMENT_FIELD_NUMBER: _ClassVar[int]
    variant_slug: str
    state: PresentationRevisionState
    revision: str
    document: _product_presentation_pb2.ProductPresentationDocument
    def __init__(self, variant_slug: _Optional[str] = ..., state: _Optional[_Union[PresentationRevisionState, _Mapping]] = ..., revision: _Optional[str] = ..., document: _Optional[_Union[_product_presentation_pb2.ProductPresentationDocument, _Mapping]] = ...) -> None: ...

class SavePresentationDraftRequest(_message.Message):
    __slots__ = ("variant_slug", "expected_generation", "document")
    VARIANT_SLUG_FIELD_NUMBER: _ClassVar[int]
    EXPECTED_GENERATION_FIELD_NUMBER: _ClassVar[int]
    DOCUMENT_FIELD_NUMBER: _ClassVar[int]
    variant_slug: str
    expected_generation: int
    document: _product_presentation_pb2.ProductPresentationDocument
    def __init__(self, variant_slug: _Optional[str] = ..., expected_generation: _Optional[int] = ..., document: _Optional[_Union[_product_presentation_pb2.ProductPresentationDocument, _Mapping]] = ...) -> None: ...

class PreviewPresentationRequest(_message.Message):
    __slots__ = ("variant_slug", "revision", "route", "locale")
    VARIANT_SLUG_FIELD_NUMBER: _ClassVar[int]
    REVISION_FIELD_NUMBER: _ClassVar[int]
    ROUTE_FIELD_NUMBER: _ClassVar[int]
    LOCALE_FIELD_NUMBER: _ClassVar[int]
    variant_slug: str
    revision: str
    route: str
    locale: str
    def __init__(self, variant_slug: _Optional[str] = ..., revision: _Optional[str] = ..., route: _Optional[str] = ..., locale: _Optional[str] = ...) -> None: ...

class PreviewPresentationResponse(_message.Message):
    __slots__ = ("presentation",)
    PRESENTATION_FIELD_NUMBER: _ClassVar[int]
    presentation: _product_presentation_pb2.ResolvedProductPresentation
    def __init__(self, presentation: _Optional[_Union[_product_presentation_pb2.ResolvedProductPresentation, _Mapping]] = ...) -> None: ...

class ActivatePresentationRequest(_message.Message):
    __slots__ = ("variant_slug", "revision", "expected_generation")
    VARIANT_SLUG_FIELD_NUMBER: _ClassVar[int]
    REVISION_FIELD_NUMBER: _ClassVar[int]
    EXPECTED_GENERATION_FIELD_NUMBER: _ClassVar[int]
    variant_slug: str
    revision: str
    expected_generation: int
    def __init__(self, variant_slug: _Optional[str] = ..., revision: _Optional[str] = ..., expected_generation: _Optional[int] = ...) -> None: ...
