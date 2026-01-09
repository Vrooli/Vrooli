from google.type import latlng_pb2 as _latlng_pb2
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Mapping as _Mapping
from typing import ClassVar as _ClassVar, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class Viewport(_message.Message):
    __slots__ = ()
    LOW_FIELD_NUMBER: _ClassVar[int]
    HIGH_FIELD_NUMBER: _ClassVar[int]
    low: _latlng_pb2.LatLng
    high: _latlng_pb2.LatLng
    def __init__(self, low: _Optional[_Union[_latlng_pb2.LatLng, _Mapping]] = ..., high: _Optional[_Union[_latlng_pb2.LatLng, _Mapping]] = ...) -> None: ...
