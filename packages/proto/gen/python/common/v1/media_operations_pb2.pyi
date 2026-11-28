from google.protobuf.internal import containers as _containers
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Iterable as _Iterable, Mapping as _Mapping
from typing import ClassVar as _ClassVar, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class MediaPort(_message.Message):
    __slots__ = ("name", "kind", "required")
    NAME_FIELD_NUMBER: _ClassVar[int]
    KIND_FIELD_NUMBER: _ClassVar[int]
    REQUIRED_FIELD_NUMBER: _ClassVar[int]
    name: str
    kind: str
    required: bool
    def __init__(self, name: _Optional[str] = ..., kind: _Optional[str] = ..., required: _Optional[bool] = ...) -> None: ...

class MediaRequirement(_message.Message):
    __slots__ = ("process", "weights", "network")
    PROCESS_FIELD_NUMBER: _ClassVar[int]
    WEIGHTS_FIELD_NUMBER: _ClassVar[int]
    NETWORK_FIELD_NUMBER: _ClassVar[int]
    process: bool
    weights: bool
    network: bool
    def __init__(self, process: _Optional[bool] = ..., weights: _Optional[bool] = ..., network: _Optional[bool] = ...) -> None: ...

class MediaOperation(_message.Message):
    __slots__ = ("scenario", "id", "summary", "family", "mechanism", "inputs", "output", "params_schema_json", "consent_weight", "backends", "requires")
    SCENARIO_FIELD_NUMBER: _ClassVar[int]
    ID_FIELD_NUMBER: _ClassVar[int]
    SUMMARY_FIELD_NUMBER: _ClassVar[int]
    FAMILY_FIELD_NUMBER: _ClassVar[int]
    MECHANISM_FIELD_NUMBER: _ClassVar[int]
    INPUTS_FIELD_NUMBER: _ClassVar[int]
    OUTPUT_FIELD_NUMBER: _ClassVar[int]
    PARAMS_SCHEMA_JSON_FIELD_NUMBER: _ClassVar[int]
    CONSENT_WEIGHT_FIELD_NUMBER: _ClassVar[int]
    BACKENDS_FIELD_NUMBER: _ClassVar[int]
    REQUIRES_FIELD_NUMBER: _ClassVar[int]
    scenario: str
    id: str
    summary: str
    family: str
    mechanism: str
    inputs: _containers.RepeatedCompositeFieldContainer[MediaPort]
    output: MediaPort
    params_schema_json: str
    consent_weight: str
    backends: _containers.RepeatedScalarFieldContainer[str]
    requires: MediaRequirement
    def __init__(self, scenario: _Optional[str] = ..., id: _Optional[str] = ..., summary: _Optional[str] = ..., family: _Optional[str] = ..., mechanism: _Optional[str] = ..., inputs: _Optional[_Iterable[_Union[MediaPort, _Mapping]]] = ..., output: _Optional[_Union[MediaPort, _Mapping]] = ..., params_schema_json: _Optional[str] = ..., consent_weight: _Optional[str] = ..., backends: _Optional[_Iterable[str]] = ..., requires: _Optional[_Union[MediaRequirement, _Mapping]] = ...) -> None: ...

class MediaOperationCatalog(_message.Message):
    __slots__ = ("contract_version", "scenario", "operations")
    CONTRACT_VERSION_FIELD_NUMBER: _ClassVar[int]
    SCENARIO_FIELD_NUMBER: _ClassVar[int]
    OPERATIONS_FIELD_NUMBER: _ClassVar[int]
    contract_version: str
    scenario: str
    operations: _containers.RepeatedCompositeFieldContainer[MediaOperation]
    def __init__(self, contract_version: _Optional[str] = ..., scenario: _Optional[str] = ..., operations: _Optional[_Iterable[_Union[MediaOperation, _Mapping]]] = ...) -> None: ...
