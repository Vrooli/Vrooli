from google.protobuf import struct_pb2 as _struct_pb2
from google.protobuf.internal import containers as _containers
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Iterable as _Iterable, Mapping as _Mapping
from typing import ClassVar as _ClassVar, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class GetWorkflowSchemaRequest(_message.Message):
    __slots__ = ("node_types",)
    NODE_TYPES_FIELD_NUMBER: _ClassVar[int]
    node_types: _containers.RepeatedScalarFieldContainer[str]
    def __init__(self, node_types: _Optional[_Iterable[str]] = ...) -> None: ...

class GetWorkflowSchemaResponse(_message.Message):
    __slots__ = ("schema",)
    SCHEMA_FIELD_NUMBER: _ClassVar[int]
    schema: _struct_pb2.Struct
    def __init__(self, schema: _Optional[_Union[_struct_pb2.Struct, _Mapping]] = ...) -> None: ...

class GetNodeTypesRequest(_message.Message):
    __slots__ = ()
    def __init__(self) -> None: ...

class GetNodeTypesResponse(_message.Message):
    __slots__ = ("node_types",)
    NODE_TYPES_FIELD_NUMBER: _ClassVar[int]
    node_types: _containers.RepeatedScalarFieldContainer[str]
    def __init__(self, node_types: _Optional[_Iterable[str]] = ...) -> None: ...

class GetStepDefinitionsRequest(_message.Message):
    __slots__ = ("types", "cli_only")
    TYPES_FIELD_NUMBER: _ClassVar[int]
    CLI_ONLY_FIELD_NUMBER: _ClassVar[int]
    types: _containers.RepeatedScalarFieldContainer[str]
    cli_only: bool
    def __init__(self, types: _Optional[_Iterable[str]] = ..., cli_only: _Optional[bool] = ...) -> None: ...

class StepPositional(_message.Message):
    __slots__ = ("name", "maps_to", "description")
    NAME_FIELD_NUMBER: _ClassVar[int]
    MAPS_TO_FIELD_NUMBER: _ClassVar[int]
    DESCRIPTION_FIELD_NUMBER: _ClassVar[int]
    name: str
    maps_to: str
    description: str
    def __init__(self, name: _Optional[str] = ..., maps_to: _Optional[str] = ..., description: _Optional[str] = ...) -> None: ...

class StepKV(_message.Message):
    __slots__ = ("key", "type", "description")
    KEY_FIELD_NUMBER: _ClassVar[int]
    TYPE_FIELD_NUMBER: _ClassVar[int]
    DESCRIPTION_FIELD_NUMBER: _ClassVar[int]
    key: str
    type: str
    description: str
    def __init__(self, key: _Optional[str] = ..., type: _Optional[str] = ..., description: _Optional[str] = ...) -> None: ...

class StepRequireOneOf(_message.Message):
    __slots__ = ("keys",)
    KEYS_FIELD_NUMBER: _ClassVar[int]
    keys: _containers.RepeatedScalarFieldContainer[str]
    def __init__(self, keys: _Optional[_Iterable[str]] = ...) -> None: ...

class StepExample(_message.Message):
    __slots__ = ("description", "cli")
    DESCRIPTION_FIELD_NUMBER: _ClassVar[int]
    CLI_FIELD_NUMBER: _ClassVar[int]
    description: str
    cli: str
    def __init__(self, description: _Optional[str] = ..., cli: _Optional[str] = ...) -> None: ...

class StepDefinition(_message.Message):
    __slots__ = ("type", "description", "positional", "required_kvs", "optional_kvs", "require_one_of", "examples", "cli_supported")
    TYPE_FIELD_NUMBER: _ClassVar[int]
    DESCRIPTION_FIELD_NUMBER: _ClassVar[int]
    POSITIONAL_FIELD_NUMBER: _ClassVar[int]
    REQUIRED_KVS_FIELD_NUMBER: _ClassVar[int]
    OPTIONAL_KVS_FIELD_NUMBER: _ClassVar[int]
    REQUIRE_ONE_OF_FIELD_NUMBER: _ClassVar[int]
    EXAMPLES_FIELD_NUMBER: _ClassVar[int]
    CLI_SUPPORTED_FIELD_NUMBER: _ClassVar[int]
    type: str
    description: str
    positional: StepPositional
    required_kvs: _containers.RepeatedCompositeFieldContainer[StepKV]
    optional_kvs: _containers.RepeatedCompositeFieldContainer[StepKV]
    require_one_of: _containers.RepeatedCompositeFieldContainer[StepRequireOneOf]
    examples: _containers.RepeatedCompositeFieldContainer[StepExample]
    cli_supported: bool
    def __init__(self, type: _Optional[str] = ..., description: _Optional[str] = ..., positional: _Optional[_Union[StepPositional, _Mapping]] = ..., required_kvs: _Optional[_Iterable[_Union[StepKV, _Mapping]]] = ..., optional_kvs: _Optional[_Iterable[_Union[StepKV, _Mapping]]] = ..., require_one_of: _Optional[_Iterable[_Union[StepRequireOneOf, _Mapping]]] = ..., examples: _Optional[_Iterable[_Union[StepExample, _Mapping]]] = ..., cli_supported: _Optional[bool] = ...) -> None: ...

class GetStepDefinitionsResponse(_message.Message):
    __slots__ = ("steps",)
    STEPS_FIELD_NUMBER: _ClassVar[int]
    steps: _containers.RepeatedCompositeFieldContainer[StepDefinition]
    def __init__(self, steps: _Optional[_Iterable[_Union[StepDefinition, _Mapping]]] = ...) -> None: ...
