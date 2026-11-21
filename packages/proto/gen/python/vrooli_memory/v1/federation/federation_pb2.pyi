from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from typing import ClassVar as _ClassVar, Optional as _Optional

DESCRIPTOR: _descriptor.FileDescriptor

class ReindexRequest(_message.Message):
    __slots__ = ("authorization_token",)
    AUTHORIZATION_TOKEN_FIELD_NUMBER: _ClassVar[int]
    authorization_token: str
    def __init__(self, authorization_token: _Optional[str] = ...) -> None: ...

class ReindexResponse(_message.Message):
    __slots__ = ("job_id",)
    JOB_ID_FIELD_NUMBER: _ClassVar[int]
    job_id: str
    def __init__(self, job_id: _Optional[str] = ...) -> None: ...

class WriteConfigRequest(_message.Message):
    __slots__ = ("authorization_token", "config_json")
    AUTHORIZATION_TOKEN_FIELD_NUMBER: _ClassVar[int]
    CONFIG_JSON_FIELD_NUMBER: _ClassVar[int]
    authorization_token: str
    config_json: str
    def __init__(self, authorization_token: _Optional[str] = ..., config_json: _Optional[str] = ...) -> None: ...

class WriteConfigResponse(_message.Message):
    __slots__ = ()
    def __init__(self) -> None: ...
