from google.protobuf.internal import containers as _containers
from google.protobuf.internal import enum_type_wrapper as _enum_type_wrapper
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Iterable as _Iterable, Mapping as _Mapping
from typing import ClassVar as _ClassVar, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class TransportWorld(int, metaclass=_enum_type_wrapper.EnumTypeWrapper):
    __slots__ = ()
    TRANSPORT_WORLD_UNSPECIFIED: _ClassVar[TransportWorld]
    TRANSPORT_WORLD_CONNECT: _ClassVar[TransportWorld]
    TRANSPORT_WORLD_HAND_ROLLED: _ClassVar[TransportWorld]
    TRANSPORT_WORLD_MIXED: _ClassVar[TransportWorld]
    TRANSPORT_WORLD_NONE: _ClassVar[TransportWorld]

class TransportKind(int, metaclass=_enum_type_wrapper.EnumTypeWrapper):
    __slots__ = ()
    TRANSPORT_KIND_UNSPECIFIED: _ClassVar[TransportKind]
    TRANSPORT_KIND_CONNECT: _ClassVar[TransportKind]
    TRANSPORT_KIND_REST: _ClassVar[TransportKind]
    TRANSPORT_KIND_HAND_ROLLED: _ClassVar[TransportKind]
    TRANSPORT_KIND_NOT_SERVED: _ClassVar[TransportKind]
TRANSPORT_WORLD_UNSPECIFIED: TransportWorld
TRANSPORT_WORLD_CONNECT: TransportWorld
TRANSPORT_WORLD_HAND_ROLLED: TransportWorld
TRANSPORT_WORLD_MIXED: TransportWorld
TRANSPORT_WORLD_NONE: TransportWorld
TRANSPORT_KIND_UNSPECIFIED: TransportKind
TRANSPORT_KIND_CONNECT: TransportKind
TRANSPORT_KIND_REST: TransportKind
TRANSPORT_KIND_HAND_ROLLED: TransportKind
TRANSPORT_KIND_NOT_SERVED: TransportKind

class ProtoSurface(_message.Message):
    __slots__ = ("scenario", "files", "services", "messages", "intra_scenario_imports", "cross_scenario_imports", "adoption_signals", "transport_world")
    SCENARIO_FIELD_NUMBER: _ClassVar[int]
    FILES_FIELD_NUMBER: _ClassVar[int]
    SERVICES_FIELD_NUMBER: _ClassVar[int]
    MESSAGES_FIELD_NUMBER: _ClassVar[int]
    INTRA_SCENARIO_IMPORTS_FIELD_NUMBER: _ClassVar[int]
    CROSS_SCENARIO_IMPORTS_FIELD_NUMBER: _ClassVar[int]
    ADOPTION_SIGNALS_FIELD_NUMBER: _ClassVar[int]
    TRANSPORT_WORLD_FIELD_NUMBER: _ClassVar[int]
    scenario: str
    files: _containers.RepeatedCompositeFieldContainer[ProtoFile]
    services: _containers.RepeatedCompositeFieldContainer[ProtoService]
    messages: _containers.RepeatedCompositeFieldContainer[ProtoMessage]
    intra_scenario_imports: _containers.RepeatedCompositeFieldContainer[ProtoImport]
    cross_scenario_imports: _containers.RepeatedCompositeFieldContainer[ProtoImport]
    adoption_signals: _containers.RepeatedCompositeFieldContainer[AdoptionSignal]
    transport_world: TransportWorld
    def __init__(self, scenario: _Optional[str] = ..., files: _Optional[_Iterable[_Union[ProtoFile, _Mapping]]] = ..., services: _Optional[_Iterable[_Union[ProtoService, _Mapping]]] = ..., messages: _Optional[_Iterable[_Union[ProtoMessage, _Mapping]]] = ..., intra_scenario_imports: _Optional[_Iterable[_Union[ProtoImport, _Mapping]]] = ..., cross_scenario_imports: _Optional[_Iterable[_Union[ProtoImport, _Mapping]]] = ..., adoption_signals: _Optional[_Iterable[_Union[AdoptionSignal, _Mapping]]] = ..., transport_world: _Optional[_Union[TransportWorld, str]] = ...) -> None: ...

class ProtoFile(_message.Message):
    __slots__ = ("path", "package", "version", "domain", "stability", "annotations")
    PATH_FIELD_NUMBER: _ClassVar[int]
    PACKAGE_FIELD_NUMBER: _ClassVar[int]
    VERSION_FIELD_NUMBER: _ClassVar[int]
    DOMAIN_FIELD_NUMBER: _ClassVar[int]
    STABILITY_FIELD_NUMBER: _ClassVar[int]
    ANNOTATIONS_FIELD_NUMBER: _ClassVar[int]
    path: str
    package: str
    version: str
    domain: str
    stability: str
    annotations: _containers.RepeatedCompositeFieldContainer[Annotation]
    def __init__(self, path: _Optional[str] = ..., package: _Optional[str] = ..., version: _Optional[str] = ..., domain: _Optional[str] = ..., stability: _Optional[str] = ..., annotations: _Optional[_Iterable[_Union[Annotation, _Mapping]]] = ...) -> None: ...

class Annotation(_message.Message):
    __slots__ = ("name", "value")
    NAME_FIELD_NUMBER: _ClassVar[int]
    VALUE_FIELD_NUMBER: _ClassVar[int]
    name: str
    value: str
    def __init__(self, name: _Optional[str] = ..., value: _Optional[str] = ...) -> None: ...

class ProtoService(_message.Message):
    __slots__ = ("file_path", "package", "name", "full_name", "domain", "rpcs")
    FILE_PATH_FIELD_NUMBER: _ClassVar[int]
    PACKAGE_FIELD_NUMBER: _ClassVar[int]
    NAME_FIELD_NUMBER: _ClassVar[int]
    FULL_NAME_FIELD_NUMBER: _ClassVar[int]
    DOMAIN_FIELD_NUMBER: _ClassVar[int]
    RPCS_FIELD_NUMBER: _ClassVar[int]
    file_path: str
    package: str
    name: str
    full_name: str
    domain: str
    rpcs: _containers.RepeatedCompositeFieldContainer[ProtoRpc]
    def __init__(self, file_path: _Optional[str] = ..., package: _Optional[str] = ..., name: _Optional[str] = ..., full_name: _Optional[str] = ..., domain: _Optional[str] = ..., rpcs: _Optional[_Iterable[_Union[ProtoRpc, _Mapping]]] = ...) -> None: ...

class ProtoRpc(_message.Message):
    __slots__ = ("name", "input", "output", "transport")
    NAME_FIELD_NUMBER: _ClassVar[int]
    INPUT_FIELD_NUMBER: _ClassVar[int]
    OUTPUT_FIELD_NUMBER: _ClassVar[int]
    TRANSPORT_FIELD_NUMBER: _ClassVar[int]
    name: str
    input: str
    output: str
    transport: TransportKind
    def __init__(self, name: _Optional[str] = ..., input: _Optional[str] = ..., output: _Optional[str] = ..., transport: _Optional[_Union[TransportKind, str]] = ...) -> None: ...

class ProtoMessage(_message.Message):
    __slots__ = ("file_path", "package", "name", "full_name", "domain", "fields")
    FILE_PATH_FIELD_NUMBER: _ClassVar[int]
    PACKAGE_FIELD_NUMBER: _ClassVar[int]
    NAME_FIELD_NUMBER: _ClassVar[int]
    FULL_NAME_FIELD_NUMBER: _ClassVar[int]
    DOMAIN_FIELD_NUMBER: _ClassVar[int]
    FIELDS_FIELD_NUMBER: _ClassVar[int]
    file_path: str
    package: str
    name: str
    full_name: str
    domain: str
    fields: _containers.RepeatedCompositeFieldContainer[ProtoField]
    def __init__(self, file_path: _Optional[str] = ..., package: _Optional[str] = ..., name: _Optional[str] = ..., full_name: _Optional[str] = ..., domain: _Optional[str] = ..., fields: _Optional[_Iterable[_Union[ProtoField, _Mapping]]] = ...) -> None: ...

class ProtoField(_message.Message):
    __slots__ = ("name", "type", "message_type", "enum_type", "repeated", "optional", "number")
    NAME_FIELD_NUMBER: _ClassVar[int]
    TYPE_FIELD_NUMBER: _ClassVar[int]
    MESSAGE_TYPE_FIELD_NUMBER: _ClassVar[int]
    ENUM_TYPE_FIELD_NUMBER: _ClassVar[int]
    REPEATED_FIELD_NUMBER: _ClassVar[int]
    OPTIONAL_FIELD_NUMBER: _ClassVar[int]
    NUMBER_FIELD_NUMBER: _ClassVar[int]
    name: str
    type: str
    message_type: str
    enum_type: str
    repeated: bool
    optional: bool
    number: int
    def __init__(self, name: _Optional[str] = ..., type: _Optional[str] = ..., message_type: _Optional[str] = ..., enum_type: _Optional[str] = ..., repeated: _Optional[bool] = ..., optional: _Optional[bool] = ..., number: _Optional[int] = ...) -> None: ...

class ProtoImport(_message.Message):
    __slots__ = ("from_file", "to_file", "from_domain", "to_domain")
    FROM_FILE_FIELD_NUMBER: _ClassVar[int]
    TO_FILE_FIELD_NUMBER: _ClassVar[int]
    FROM_DOMAIN_FIELD_NUMBER: _ClassVar[int]
    TO_DOMAIN_FIELD_NUMBER: _ClassVar[int]
    from_file: str
    to_file: str
    from_domain: str
    to_domain: str
    def __init__(self, from_file: _Optional[str] = ..., to_file: _Optional[str] = ..., from_domain: _Optional[str] = ..., to_domain: _Optional[str] = ...) -> None: ...

class AdoptionSignal(_message.Message):
    __slots__ = ("name", "present", "detail")
    NAME_FIELD_NUMBER: _ClassVar[int]
    PRESENT_FIELD_NUMBER: _ClassVar[int]
    DETAIL_FIELD_NUMBER: _ClassVar[int]
    name: str
    present: bool
    detail: str
    def __init__(self, name: _Optional[str] = ..., present: _Optional[bool] = ..., detail: _Optional[str] = ...) -> None: ...
