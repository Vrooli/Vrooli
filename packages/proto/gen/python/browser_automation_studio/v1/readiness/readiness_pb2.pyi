from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from typing import ClassVar as _ClassVar, Optional as _Optional

DESCRIPTOR: _descriptor.FileDescriptor

class GetReadinessProfileRequest(_message.Message):
    __slots__ = ("scenario", "path")
    SCENARIO_FIELD_NUMBER: _ClassVar[int]
    PATH_FIELD_NUMBER: _ClassVar[int]
    scenario: str
    path: str
    def __init__(self, scenario: _Optional[str] = ..., path: _Optional[str] = ...) -> None: ...

class GetReadinessProfileResponse(_message.Message):
    __slots__ = ("scenario", "profile_json", "profile_version")
    SCENARIO_FIELD_NUMBER: _ClassVar[int]
    PROFILE_JSON_FIELD_NUMBER: _ClassVar[int]
    PROFILE_VERSION_FIELD_NUMBER: _ClassVar[int]
    scenario: str
    profile_json: str
    profile_version: str
    def __init__(self, scenario: _Optional[str] = ..., profile_json: _Optional[str] = ..., profile_version: _Optional[str] = ...) -> None: ...
