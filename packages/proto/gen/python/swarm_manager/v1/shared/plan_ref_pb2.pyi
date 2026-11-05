from buf.validate import validate_pb2 as _validate_pb2
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from typing import ClassVar as _ClassVar, Optional as _Optional

DESCRIPTOR: _descriptor.FileDescriptor

class PlanRef(_message.Message):
    __slots__ = ("provider", "plan_id", "slug", "role")
    PROVIDER_FIELD_NUMBER: _ClassVar[int]
    PLAN_ID_FIELD_NUMBER: _ClassVar[int]
    SLUG_FIELD_NUMBER: _ClassVar[int]
    ROLE_FIELD_NUMBER: _ClassVar[int]
    provider: str
    plan_id: str
    slug: str
    role: str
    def __init__(self, provider: _Optional[str] = ..., plan_id: _Optional[str] = ..., slug: _Optional[str] = ..., role: _Optional[str] = ...) -> None: ...
