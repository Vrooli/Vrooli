from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Mapping as _Mapping
from typing import ClassVar as _ClassVar, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class GitHubUpdateConfig(_message.Message):
    __slots__ = ("owner", "repo", "private")
    OWNER_FIELD_NUMBER: _ClassVar[int]
    REPO_FIELD_NUMBER: _ClassVar[int]
    PRIVATE_FIELD_NUMBER: _ClassVar[int]
    owner: str
    repo: str
    private: bool
    def __init__(self, owner: _Optional[str] = ..., repo: _Optional[str] = ..., private: _Optional[bool] = ...) -> None: ...

class GenericUpdateConfig(_message.Message):
    __slots__ = ("url", "channel_path")
    URL_FIELD_NUMBER: _ClassVar[int]
    CHANNEL_PATH_FIELD_NUMBER: _ClassVar[int]
    url: str
    channel_path: str
    def __init__(self, url: _Optional[str] = ..., channel_path: _Optional[str] = ...) -> None: ...

class UpdateConfig(_message.Message):
    __slots__ = ("channel", "provider", "auto_check", "github", "generic")
    CHANNEL_FIELD_NUMBER: _ClassVar[int]
    PROVIDER_FIELD_NUMBER: _ClassVar[int]
    AUTO_CHECK_FIELD_NUMBER: _ClassVar[int]
    GITHUB_FIELD_NUMBER: _ClassVar[int]
    GENERIC_FIELD_NUMBER: _ClassVar[int]
    channel: str
    provider: str
    auto_check: bool
    github: GitHubUpdateConfig
    generic: GenericUpdateConfig
    def __init__(self, channel: _Optional[str] = ..., provider: _Optional[str] = ..., auto_check: _Optional[bool] = ..., github: _Optional[_Union[GitHubUpdateConfig, _Mapping]] = ..., generic: _Optional[_Union[GenericUpdateConfig, _Mapping]] = ...) -> None: ...
