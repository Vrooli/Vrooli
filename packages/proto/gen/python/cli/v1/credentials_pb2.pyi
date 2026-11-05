from google.protobuf import struct_pb2 as _struct_pb2
from google.protobuf.internal import containers as _containers
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Iterable as _Iterable, Mapping as _Mapping
from typing import ClassVar as _ClassVar, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class CliCredentialList(_message.Message):
    __slots__ = ("inventory_basis", "managed_instances_included", "credential_count", "declaration_site_count", "uncovered", "required_absent", "credentials")
    INVENTORY_BASIS_FIELD_NUMBER: _ClassVar[int]
    MANAGED_INSTANCES_INCLUDED_FIELD_NUMBER: _ClassVar[int]
    CREDENTIAL_COUNT_FIELD_NUMBER: _ClassVar[int]
    DECLARATION_SITE_COUNT_FIELD_NUMBER: _ClassVar[int]
    UNCOVERED_FIELD_NUMBER: _ClassVar[int]
    REQUIRED_ABSENT_FIELD_NUMBER: _ClassVar[int]
    CREDENTIALS_FIELD_NUMBER: _ClassVar[int]
    inventory_basis: str
    managed_instances_included: bool
    credential_count: int
    declaration_site_count: int
    uncovered: _containers.RepeatedScalarFieldContainer[str]
    required_absent: _containers.RepeatedScalarFieldContainer[str]
    credentials: _containers.RepeatedCompositeFieldContainer[CliCredentialEntry]
    def __init__(self, inventory_basis: _Optional[str] = ..., managed_instances_included: _Optional[bool] = ..., credential_count: _Optional[int] = ..., declaration_site_count: _Optional[int] = ..., uncovered: _Optional[_Iterable[str]] = ..., required_absent: _Optional[_Iterable[str]] = ..., credentials: _Optional[_Iterable[_Union[CliCredentialEntry, _Mapping]]] = ...) -> None: ...

class CliCredentialStatus(_message.Message):
    __slots__ = ("checked_at", "configured", "field", "identity", "provider", "provider_state")
    CHECKED_AT_FIELD_NUMBER: _ClassVar[int]
    CONFIGURED_FIELD_NUMBER: _ClassVar[int]
    FIELD_FIELD_NUMBER: _ClassVar[int]
    IDENTITY_FIELD_NUMBER: _ClassVar[int]
    PROVIDER_FIELD_NUMBER: _ClassVar[int]
    PROVIDER_STATE_FIELD_NUMBER: _ClassVar[int]
    checked_at: str
    configured: bool
    field: str
    identity: str
    provider: str
    provider_state: str
    def __init__(self, checked_at: _Optional[str] = ..., configured: _Optional[bool] = ..., field: _Optional[str] = ..., identity: _Optional[str] = ..., provider: _Optional[str] = ..., provider_state: _Optional[str] = ...) -> None: ...

class CliCredentialEntry(_message.Message):
    __slots__ = ("resource", "env", "logical_id", "field", "label", "description", "provisioning", "derived_from", "required", "configured", "state", "remediation")
    RESOURCE_FIELD_NUMBER: _ClassVar[int]
    ENV_FIELD_NUMBER: _ClassVar[int]
    LOGICAL_ID_FIELD_NUMBER: _ClassVar[int]
    FIELD_FIELD_NUMBER: _ClassVar[int]
    LABEL_FIELD_NUMBER: _ClassVar[int]
    DESCRIPTION_FIELD_NUMBER: _ClassVar[int]
    PROVISIONING_FIELD_NUMBER: _ClassVar[int]
    DERIVED_FROM_FIELD_NUMBER: _ClassVar[int]
    REQUIRED_FIELD_NUMBER: _ClassVar[int]
    CONFIGURED_FIELD_NUMBER: _ClassVar[int]
    STATE_FIELD_NUMBER: _ClassVar[int]
    REMEDIATION_FIELD_NUMBER: _ClassVar[int]
    resource: str
    env: str
    logical_id: str
    field: str
    label: str
    description: str
    provisioning: str
    derived_from: str
    required: bool
    configured: bool
    state: str
    remediation: str
    def __init__(self, resource: _Optional[str] = ..., env: _Optional[str] = ..., logical_id: _Optional[str] = ..., field: _Optional[str] = ..., label: _Optional[str] = ..., description: _Optional[str] = ..., provisioning: _Optional[str] = ..., derived_from: _Optional[str] = ..., required: _Optional[bool] = ..., configured: _Optional[bool] = ..., state: _Optional[str] = ..., remediation: _Optional[str] = ...) -> None: ...

class CliCredentialDoctor(_message.Message):
    __slots__ = ("payload",)
    PAYLOAD_FIELD_NUMBER: _ClassVar[int]
    payload: _struct_pb2.Struct
    def __init__(self, payload: _Optional[_Union[_struct_pb2.Struct, _Mapping]] = ...) -> None: ...

class CliCredentialStoreStatus(_message.Message):
    __slots__ = ("payload",)
    PAYLOAD_FIELD_NUMBER: _ClassVar[int]
    payload: _struct_pb2.Struct
    def __init__(self, payload: _Optional[_Union[_struct_pb2.Struct, _Mapping]] = ...) -> None: ...

class CliCredentialKeyringStatus(_message.Message):
    __slots__ = ("payload",)
    PAYLOAD_FIELD_NUMBER: _ClassVar[int]
    payload: _struct_pb2.Struct
    def __init__(self, payload: _Optional[_Union[_struct_pb2.Struct, _Mapping]] = ...) -> None: ...

class CliCredentialRecovery(_message.Message):
    __slots__ = ("payload",)
    PAYLOAD_FIELD_NUMBER: _ClassVar[int]
    payload: _struct_pb2.Struct
    def __init__(self, payload: _Optional[_Union[_struct_pb2.Struct, _Mapping]] = ...) -> None: ...

class CliBreakGlassStatus(_message.Message):
    __slots__ = ("account_id", "audience", "complete", "metadata", "provisioned_at", "public", "scopes", "wrapped_private")
    ACCOUNT_ID_FIELD_NUMBER: _ClassVar[int]
    AUDIENCE_FIELD_NUMBER: _ClassVar[int]
    COMPLETE_FIELD_NUMBER: _ClassVar[int]
    METADATA_FIELD_NUMBER: _ClassVar[int]
    PROVISIONED_AT_FIELD_NUMBER: _ClassVar[int]
    PUBLIC_FIELD_NUMBER: _ClassVar[int]
    SCOPES_FIELD_NUMBER: _ClassVar[int]
    WRAPPED_PRIVATE_FIELD_NUMBER: _ClassVar[int]
    account_id: str
    audience: str
    complete: bool
    metadata: bool
    provisioned_at: int
    public: bool
    scopes: _containers.RepeatedScalarFieldContainer[str]
    wrapped_private: bool
    def __init__(self, account_id: _Optional[str] = ..., audience: _Optional[str] = ..., complete: _Optional[bool] = ..., metadata: _Optional[bool] = ..., provisioned_at: _Optional[int] = ..., public: _Optional[bool] = ..., scopes: _Optional[_Iterable[str]] = ..., wrapped_private: _Optional[bool] = ...) -> None: ...
