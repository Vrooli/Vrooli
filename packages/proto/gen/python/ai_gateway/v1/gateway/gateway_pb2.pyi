from ai_gateway.v1.shared import gateway_pb2 as _gateway_pb2
from google.protobuf.internal import containers as _containers
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Iterable as _Iterable, Mapping as _Mapping
from typing import ClassVar as _ClassVar, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class ValidateGatewayRequestRequest(_message.Message):
    __slots__ = ("request",)
    REQUEST_FIELD_NUMBER: _ClassVar[int]
    request: _gateway_pb2.GatewayRequest
    def __init__(self, request: _Optional[_Union[_gateway_pb2.GatewayRequest, _Mapping]] = ...) -> None: ...

class ValidateGatewayRequestResponse(_message.Message):
    __slots__ = ("valid", "issues", "accepted_profiles")
    VALID_FIELD_NUMBER: _ClassVar[int]
    ISSUES_FIELD_NUMBER: _ClassVar[int]
    ACCEPTED_PROFILES_FIELD_NUMBER: _ClassVar[int]
    valid: bool
    issues: _containers.RepeatedCompositeFieldContainer[_gateway_pb2.ValidationIssue]
    accepted_profiles: _containers.RepeatedScalarFieldContainer[str]
    def __init__(self, valid: _Optional[bool] = ..., issues: _Optional[_Iterable[_Union[_gateway_pb2.ValidationIssue, _Mapping]]] = ..., accepted_profiles: _Optional[_Iterable[str]] = ...) -> None: ...
