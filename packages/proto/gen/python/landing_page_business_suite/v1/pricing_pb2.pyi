from landing_page_business_suite.v1.shared import commerce_pb2 as _commerce_pb2
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Mapping as _Mapping
from typing import ClassVar as _ClassVar, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class GetPricingRequest(_message.Message):
    __slots__ = ("bundle_key", "include_hidden")
    BUNDLE_KEY_FIELD_NUMBER: _ClassVar[int]
    INCLUDE_HIDDEN_FIELD_NUMBER: _ClassVar[int]
    bundle_key: str
    include_hidden: bool
    def __init__(self, bundle_key: _Optional[str] = ..., include_hidden: _Optional[bool] = ...) -> None: ...

class GetPricingResponse(_message.Message):
    __slots__ = ("pricing",)
    PRICING_FIELD_NUMBER: _ClassVar[int]
    pricing: _commerce_pb2.PricingOverview
    def __init__(self, pricing: _Optional[_Union[_commerce_pb2.PricingOverview, _Mapping]] = ...) -> None: ...
