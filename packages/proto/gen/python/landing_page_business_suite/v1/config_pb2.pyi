from landing_page_business_suite.v1.shared import commerce_pb2 as _commerce_pb2
from landing_page_business_suite.v1.shared import downloads_pb2 as _downloads_pb2
from landing_page_business_suite.v1.shared import product_presentation_pb2 as _product_presentation_pb2
from google.protobuf.internal import containers as _containers
from google.protobuf.internal import enum_type_wrapper as _enum_type_wrapper
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Iterable as _Iterable, Mapping as _Mapping
from typing import ClassVar as _ClassVar, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class PresentationAssignmentSource(int, metaclass=_enum_type_wrapper.EnumTypeWrapper):
    __slots__ = ()
    PRESENTATION_ASSIGNMENT_SOURCE_UNSPECIFIED: _ClassVar[PresentationAssignmentSource]
    PRESENTATION_ASSIGNMENT_SOURCE_WEIGHTED_VISITOR: _ClassVar[PresentationAssignmentSource]
    PRESENTATION_ASSIGNMENT_SOURCE_EXPLICIT_URL: _ClassVar[PresentationAssignmentSource]
PRESENTATION_ASSIGNMENT_SOURCE_UNSPECIFIED: PresentationAssignmentSource
PRESENTATION_ASSIGNMENT_SOURCE_WEIGHTED_VISITOR: PresentationAssignmentSource
PRESENTATION_ASSIGNMENT_SOURCE_EXPLICIT_URL: PresentationAssignmentSource

class GetLandingConfigRequest(_message.Message):
    __slots__ = ("variant_slug", "visitor_id", "route", "locale")
    VARIANT_SLUG_FIELD_NUMBER: _ClassVar[int]
    VISITOR_ID_FIELD_NUMBER: _ClassVar[int]
    ROUTE_FIELD_NUMBER: _ClassVar[int]
    LOCALE_FIELD_NUMBER: _ClassVar[int]
    variant_slug: str
    visitor_id: str
    route: str
    locale: str
    def __init__(self, variant_slug: _Optional[str] = ..., visitor_id: _Optional[str] = ..., route: _Optional[str] = ..., locale: _Optional[str] = ...) -> None: ...

class RecordPresentationExposureRequest(_message.Message):
    __slots__ = ("visitor_id", "variant_slug", "revision", "route", "locale", "block_digest", "weight_fingerprint", "source")
    VISITOR_ID_FIELD_NUMBER: _ClassVar[int]
    VARIANT_SLUG_FIELD_NUMBER: _ClassVar[int]
    REVISION_FIELD_NUMBER: _ClassVar[int]
    ROUTE_FIELD_NUMBER: _ClassVar[int]
    LOCALE_FIELD_NUMBER: _ClassVar[int]
    BLOCK_DIGEST_FIELD_NUMBER: _ClassVar[int]
    WEIGHT_FINGERPRINT_FIELD_NUMBER: _ClassVar[int]
    SOURCE_FIELD_NUMBER: _ClassVar[int]
    visitor_id: str
    variant_slug: str
    revision: str
    route: str
    locale: str
    block_digest: str
    weight_fingerprint: str
    source: PresentationAssignmentSource
    def __init__(self, visitor_id: _Optional[str] = ..., variant_slug: _Optional[str] = ..., revision: _Optional[str] = ..., route: _Optional[str] = ..., locale: _Optional[str] = ..., block_digest: _Optional[str] = ..., weight_fingerprint: _Optional[str] = ..., source: _Optional[_Union[PresentationAssignmentSource, str]] = ...) -> None: ...

class RecordPresentationExposureResponse(_message.Message):
    __slots__ = ("recorded",)
    RECORDED_FIELD_NUMBER: _ClassVar[int]
    recorded: bool
    def __init__(self, recorded: _Optional[bool] = ...) -> None: ...

class LandingConfigResponse(_message.Message):
    __slots__ = ("pricing", "downloads", "fallback", "presentation")
    PRICING_FIELD_NUMBER: _ClassVar[int]
    DOWNLOADS_FIELD_NUMBER: _ClassVar[int]
    FALLBACK_FIELD_NUMBER: _ClassVar[int]
    PRESENTATION_FIELD_NUMBER: _ClassVar[int]
    pricing: _commerce_pb2.PricingOverview
    downloads: _containers.RepeatedCompositeFieldContainer[_downloads_pb2.DownloadApp]
    fallback: bool
    presentation: _product_presentation_pb2.ResolvedProductPresentation
    def __init__(self, pricing: _Optional[_Union[_commerce_pb2.PricingOverview, _Mapping]] = ..., downloads: _Optional[_Iterable[_Union[_downloads_pb2.DownloadApp, _Mapping]]] = ..., fallback: _Optional[bool] = ..., presentation: _Optional[_Union[_product_presentation_pb2.ResolvedProductPresentation, _Mapping]] = ...) -> None: ...
