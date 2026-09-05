from google.protobuf.internal import containers as _containers
from google.protobuf.internal import enum_type_wrapper as _enum_type_wrapper
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Iterable as _Iterable, Mapping as _Mapping
from typing import ClassVar as _ClassVar, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class OperatorInputKind(int, metaclass=_enum_type_wrapper.EnumTypeWrapper):
    __slots__ = ()
    OPERATOR_INPUT_KIND_UNSPECIFIED: _ClassVar[OperatorInputKind]
    OPERATOR_INPUT_KIND_SECRET: _ClassVar[OperatorInputKind]
    OPERATOR_INPUT_KIND_CHOICE: _ClassVar[OperatorInputKind]
    OPERATOR_INPUT_KIND_CONFIRM: _ClassVar[OperatorInputKind]
    OPERATOR_INPUT_KIND_PATH: _ClassVar[OperatorInputKind]
    OPERATOR_INPUT_KIND_ENUM: _ClassVar[OperatorInputKind]
    OPERATOR_INPUT_KIND_BOOLEAN: _ClassVar[OperatorInputKind]
    OPERATOR_INPUT_KIND_DURATION: _ClassVar[OperatorInputKind]
    OPERATOR_INPUT_KIND_CONFIRMATION: _ClassVar[OperatorInputKind]
OPERATOR_INPUT_KIND_UNSPECIFIED: OperatorInputKind
OPERATOR_INPUT_KIND_SECRET: OperatorInputKind
OPERATOR_INPUT_KIND_CHOICE: OperatorInputKind
OPERATOR_INPUT_KIND_CONFIRM: OperatorInputKind
OPERATOR_INPUT_KIND_PATH: OperatorInputKind
OPERATOR_INPUT_KIND_ENUM: OperatorInputKind
OPERATOR_INPUT_KIND_BOOLEAN: OperatorInputKind
OPERATOR_INPUT_KIND_DURATION: OperatorInputKind
OPERATOR_INPUT_KIND_CONFIRMATION: OperatorInputKind

class Candidate(_message.Message):
    __slots__ = ("id", "kind", "label", "location", "stable_identity", "device_identity", "writable", "status", "risk", "remediation", "metadata")
    class MetadataEntry(_message.Message):
        __slots__ = ("key", "value")
        KEY_FIELD_NUMBER: _ClassVar[int]
        VALUE_FIELD_NUMBER: _ClassVar[int]
        key: str
        value: str
        def __init__(self, key: _Optional[str] = ..., value: _Optional[str] = ...) -> None: ...
    ID_FIELD_NUMBER: _ClassVar[int]
    KIND_FIELD_NUMBER: _ClassVar[int]
    LABEL_FIELD_NUMBER: _ClassVar[int]
    LOCATION_FIELD_NUMBER: _ClassVar[int]
    STABLE_IDENTITY_FIELD_NUMBER: _ClassVar[int]
    DEVICE_IDENTITY_FIELD_NUMBER: _ClassVar[int]
    WRITABLE_FIELD_NUMBER: _ClassVar[int]
    STATUS_FIELD_NUMBER: _ClassVar[int]
    RISK_FIELD_NUMBER: _ClassVar[int]
    REMEDIATION_FIELD_NUMBER: _ClassVar[int]
    METADATA_FIELD_NUMBER: _ClassVar[int]
    id: str
    kind: str
    label: str
    location: str
    stable_identity: str
    device_identity: str
    writable: bool
    status: str
    risk: str
    remediation: str
    metadata: _containers.ScalarMap[str, str]
    def __init__(self, id: _Optional[str] = ..., kind: _Optional[str] = ..., label: _Optional[str] = ..., location: _Optional[str] = ..., stable_identity: _Optional[str] = ..., device_identity: _Optional[str] = ..., writable: _Optional[bool] = ..., status: _Optional[str] = ..., risk: _Optional[str] = ..., remediation: _Optional[str] = ..., metadata: _Optional[_Mapping[str, str]] = ...) -> None: ...

class OperatorInputRequest(_message.Message):
    __slots__ = ("id", "kind", "contract_version", "owner", "capability_id", "action_id", "input_id", "title", "description", "default_value", "options", "candidates", "remediation", "unblocks", "validation", "required", "target")
    ID_FIELD_NUMBER: _ClassVar[int]
    KIND_FIELD_NUMBER: _ClassVar[int]
    CONTRACT_VERSION_FIELD_NUMBER: _ClassVar[int]
    OWNER_FIELD_NUMBER: _ClassVar[int]
    CAPABILITY_ID_FIELD_NUMBER: _ClassVar[int]
    ACTION_ID_FIELD_NUMBER: _ClassVar[int]
    INPUT_ID_FIELD_NUMBER: _ClassVar[int]
    TITLE_FIELD_NUMBER: _ClassVar[int]
    DESCRIPTION_FIELD_NUMBER: _ClassVar[int]
    DEFAULT_VALUE_FIELD_NUMBER: _ClassVar[int]
    OPTIONS_FIELD_NUMBER: _ClassVar[int]
    CANDIDATES_FIELD_NUMBER: _ClassVar[int]
    REMEDIATION_FIELD_NUMBER: _ClassVar[int]
    UNBLOCKS_FIELD_NUMBER: _ClassVar[int]
    VALIDATION_FIELD_NUMBER: _ClassVar[int]
    REQUIRED_FIELD_NUMBER: _ClassVar[int]
    TARGET_FIELD_NUMBER: _ClassVar[int]
    id: str
    kind: OperatorInputKind
    contract_version: str
    owner: str
    capability_id: str
    action_id: str
    input_id: str
    title: str
    description: str
    default_value: str
    options: _containers.RepeatedScalarFieldContainer[str]
    candidates: _containers.RepeatedCompositeFieldContainer[Candidate]
    remediation: str
    unblocks: _containers.RepeatedScalarFieldContainer[str]
    validation: str
    required: bool
    target: str
    def __init__(self, id: _Optional[str] = ..., kind: _Optional[_Union[OperatorInputKind, str]] = ..., contract_version: _Optional[str] = ..., owner: _Optional[str] = ..., capability_id: _Optional[str] = ..., action_id: _Optional[str] = ..., input_id: _Optional[str] = ..., title: _Optional[str] = ..., description: _Optional[str] = ..., default_value: _Optional[str] = ..., options: _Optional[_Iterable[str]] = ..., candidates: _Optional[_Iterable[_Union[Candidate, _Mapping]]] = ..., remediation: _Optional[str] = ..., unblocks: _Optional[_Iterable[str]] = ..., validation: _Optional[str] = ..., required: _Optional[bool] = ..., target: _Optional[str] = ...) -> None: ...
