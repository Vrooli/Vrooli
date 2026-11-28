from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from typing import ClassVar as _ClassVar, Optional as _Optional

DESCRIPTOR: _descriptor.FileDescriptor

class Credentials(_message.Message):
    __slots__ = ("byok_provider", "byok_key", "lpbs_token", "user_identity")
    BYOK_PROVIDER_FIELD_NUMBER: _ClassVar[int]
    BYOK_KEY_FIELD_NUMBER: _ClassVar[int]
    LPBS_TOKEN_FIELD_NUMBER: _ClassVar[int]
    USER_IDENTITY_FIELD_NUMBER: _ClassVar[int]
    byok_provider: str
    byok_key: str
    lpbs_token: str
    user_identity: str
    def __init__(self, byok_provider: _Optional[str] = ..., byok_key: _Optional[str] = ..., lpbs_token: _Optional[str] = ..., user_identity: _Optional[str] = ...) -> None: ...
