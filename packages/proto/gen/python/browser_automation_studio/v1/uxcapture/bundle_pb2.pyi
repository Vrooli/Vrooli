from google.protobuf.internal import containers as _containers
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Iterable as _Iterable, Mapping as _Mapping
from typing import ClassVar as _ClassVar, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class Bundle(_message.Message):
    __slots__ = ("schema_version", "id", "captured_at", "source", "session", "viewport", "theme", "lenses", "artifacts", "privacy", "next_commands")
    class LensesEntry(_message.Message):
        __slots__ = ("key", "value")
        KEY_FIELD_NUMBER: _ClassVar[int]
        VALUE_FIELD_NUMBER: _ClassVar[int]
        key: str
        value: Lens
        def __init__(self, key: _Optional[str] = ..., value: _Optional[_Union[Lens, _Mapping]] = ...) -> None: ...
    SCHEMA_VERSION_FIELD_NUMBER: _ClassVar[int]
    ID_FIELD_NUMBER: _ClassVar[int]
    CAPTURED_AT_FIELD_NUMBER: _ClassVar[int]
    SOURCE_FIELD_NUMBER: _ClassVar[int]
    SESSION_FIELD_NUMBER: _ClassVar[int]
    VIEWPORT_FIELD_NUMBER: _ClassVar[int]
    THEME_FIELD_NUMBER: _ClassVar[int]
    LENSES_FIELD_NUMBER: _ClassVar[int]
    ARTIFACTS_FIELD_NUMBER: _ClassVar[int]
    PRIVACY_FIELD_NUMBER: _ClassVar[int]
    NEXT_COMMANDS_FIELD_NUMBER: _ClassVar[int]
    schema_version: str
    id: str
    captured_at: str
    source: Source
    session: Session
    viewport: Viewport
    theme: str
    lenses: _containers.MessageMap[str, Lens]
    artifacts: _containers.RepeatedCompositeFieldContainer[Artifact]
    privacy: Privacy
    next_commands: _containers.RepeatedScalarFieldContainer[str]
    def __init__(self, schema_version: _Optional[str] = ..., id: _Optional[str] = ..., captured_at: _Optional[str] = ..., source: _Optional[_Union[Source, _Mapping]] = ..., session: _Optional[_Union[Session, _Mapping]] = ..., viewport: _Optional[_Union[Viewport, _Mapping]] = ..., theme: _Optional[str] = ..., lenses: _Optional[_Mapping[str, Lens]] = ..., artifacts: _Optional[_Iterable[_Union[Artifact, _Mapping]]] = ..., privacy: _Optional[_Union[Privacy, _Mapping]] = ..., next_commands: _Optional[_Iterable[str]] = ...) -> None: ...

class Source(_message.Message):
    __slots__ = ("kind", "scenario", "route", "url", "filename", "build_id", "revision")
    KIND_FIELD_NUMBER: _ClassVar[int]
    SCENARIO_FIELD_NUMBER: _ClassVar[int]
    ROUTE_FIELD_NUMBER: _ClassVar[int]
    URL_FIELD_NUMBER: _ClassVar[int]
    FILENAME_FIELD_NUMBER: _ClassVar[int]
    BUILD_ID_FIELD_NUMBER: _ClassVar[int]
    REVISION_FIELD_NUMBER: _ClassVar[int]
    kind: str
    scenario: str
    route: str
    url: str
    filename: str
    build_id: str
    revision: str
    def __init__(self, kind: _Optional[str] = ..., scenario: _Optional[str] = ..., route: _Optional[str] = ..., url: _Optional[str] = ..., filename: _Optional[str] = ..., build_id: _Optional[str] = ..., revision: _Optional[str] = ...) -> None: ...

class Session(_message.Message):
    __slots__ = ("id", "browser", "identity")
    ID_FIELD_NUMBER: _ClassVar[int]
    BROWSER_FIELD_NUMBER: _ClassVar[int]
    IDENTITY_FIELD_NUMBER: _ClassVar[int]
    id: str
    browser: str
    identity: str
    def __init__(self, id: _Optional[str] = ..., browser: _Optional[str] = ..., identity: _Optional[str] = ...) -> None: ...

class Viewport(_message.Message):
    __slots__ = ("width", "height", "device_scale_factor")
    WIDTH_FIELD_NUMBER: _ClassVar[int]
    HEIGHT_FIELD_NUMBER: _ClassVar[int]
    DEVICE_SCALE_FACTOR_FIELD_NUMBER: _ClassVar[int]
    width: int
    height: int
    device_scale_factor: float
    def __init__(self, width: _Optional[int] = ..., height: _Optional[int] = ..., device_scale_factor: _Optional[float] = ...) -> None: ...

class Lens(_message.Message):
    __slots__ = ("status", "reason", "detail")
    STATUS_FIELD_NUMBER: _ClassVar[int]
    REASON_FIELD_NUMBER: _ClassVar[int]
    DETAIL_FIELD_NUMBER: _ClassVar[int]
    status: str
    reason: str
    detail: str
    def __init__(self, status: _Optional[str] = ..., reason: _Optional[str] = ..., detail: _Optional[str] = ...) -> None: ...

class Artifact(_message.Message):
    __slots__ = ("kind", "path", "media_type", "size_bytes", "sha256")
    KIND_FIELD_NUMBER: _ClassVar[int]
    PATH_FIELD_NUMBER: _ClassVar[int]
    MEDIA_TYPE_FIELD_NUMBER: _ClassVar[int]
    SIZE_BYTES_FIELD_NUMBER: _ClassVar[int]
    SHA256_FIELD_NUMBER: _ClassVar[int]
    kind: str
    path: str
    media_type: str
    size_bytes: int
    sha256: str
    def __init__(self, kind: _Optional[str] = ..., path: _Optional[str] = ..., media_type: _Optional[str] = ..., size_bytes: _Optional[int] = ..., sha256: _Optional[str] = ...) -> None: ...

class Privacy(_message.Message):
    __slots__ = ("redaction_status", "redaction_reason", "redacted_classes", "external", "ephemeral")
    REDACTION_STATUS_FIELD_NUMBER: _ClassVar[int]
    REDACTION_REASON_FIELD_NUMBER: _ClassVar[int]
    REDACTED_CLASSES_FIELD_NUMBER: _ClassVar[int]
    EXTERNAL_FIELD_NUMBER: _ClassVar[int]
    EPHEMERAL_FIELD_NUMBER: _ClassVar[int]
    redaction_status: str
    redaction_reason: str
    redacted_classes: _containers.RepeatedScalarFieldContainer[str]
    external: bool
    ephemeral: bool
    def __init__(self, redaction_status: _Optional[str] = ..., redaction_reason: _Optional[str] = ..., redacted_classes: _Optional[_Iterable[str]] = ..., external: _Optional[bool] = ..., ephemeral: _Optional[bool] = ...) -> None: ...
