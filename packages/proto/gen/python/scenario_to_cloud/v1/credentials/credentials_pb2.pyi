import datetime

from google.protobuf import struct_pb2 as _struct_pb2
from google.protobuf import timestamp_pb2 as _timestamp_pb2
from google.protobuf.internal import containers as _containers
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Iterable as _Iterable, Mapping as _Mapping
from typing import ClassVar as _ClassVar, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class CredentialDescriptor(_message.Message):
    __slots__ = ("logical_id", "field")
    LOGICAL_ID_FIELD_NUMBER: _ClassVar[int]
    FIELD_FIELD_NUMBER: _ClassVar[int]
    logical_id: str
    field: str
    def __init__(self, logical_id: _Optional[str] = ..., field: _Optional[str] = ...) -> None: ...

class CredentialVersion(_message.Message):
    __slots__ = ("number", "content_ref", "created_at", "expires_at")
    NUMBER_FIELD_NUMBER: _ClassVar[int]
    CONTENT_REF_FIELD_NUMBER: _ClassVar[int]
    CREATED_AT_FIELD_NUMBER: _ClassVar[int]
    EXPIRES_AT_FIELD_NUMBER: _ClassVar[int]
    number: int
    content_ref: str
    created_at: _timestamp_pb2.Timestamp
    expires_at: _timestamp_pb2.Timestamp
    def __init__(self, number: _Optional[int] = ..., content_ref: _Optional[str] = ..., created_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., expires_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ...) -> None: ...

class CredentialBinding(_message.Message):
    __slots__ = ("id", "deployment_id", "descriptor", "source_class", "target_type", "target_name", "version", "previous_version", "consumer_refs", "grant_ref", "recovery_key_ref", "state", "created_at", "updated_at")
    ID_FIELD_NUMBER: _ClassVar[int]
    DEPLOYMENT_ID_FIELD_NUMBER: _ClassVar[int]
    DESCRIPTOR_FIELD_NUMBER: _ClassVar[int]
    CLASS_FIELD_NUMBER: _ClassVar[int]
    SOURCE_CLASS_FIELD_NUMBER: _ClassVar[int]
    TARGET_TYPE_FIELD_NUMBER: _ClassVar[int]
    TARGET_NAME_FIELD_NUMBER: _ClassVar[int]
    VERSION_FIELD_NUMBER: _ClassVar[int]
    PREVIOUS_VERSION_FIELD_NUMBER: _ClassVar[int]
    CONSUMER_REFS_FIELD_NUMBER: _ClassVar[int]
    GRANT_REF_FIELD_NUMBER: _ClassVar[int]
    RECOVERY_KEY_REF_FIELD_NUMBER: _ClassVar[int]
    STATE_FIELD_NUMBER: _ClassVar[int]
    CREATED_AT_FIELD_NUMBER: _ClassVar[int]
    UPDATED_AT_FIELD_NUMBER: _ClassVar[int]
    id: str
    deployment_id: str
    descriptor: CredentialDescriptor
    source_class: str
    target_type: str
    target_name: str
    version: CredentialVersion
    previous_version: CredentialVersion
    consumer_refs: _containers.RepeatedScalarFieldContainer[str]
    grant_ref: str
    recovery_key_ref: str
    state: str
    created_at: _timestamp_pb2.Timestamp
    updated_at: _timestamp_pb2.Timestamp
    def __init__(self, id: _Optional[str] = ..., deployment_id: _Optional[str] = ..., descriptor: _Optional[_Union[CredentialDescriptor, _Mapping]] = ..., source_class: _Optional[str] = ..., target_type: _Optional[str] = ..., target_name: _Optional[str] = ..., version: _Optional[_Union[CredentialVersion, _Mapping]] = ..., previous_version: _Optional[_Union[CredentialVersion, _Mapping]] = ..., consumer_refs: _Optional[_Iterable[str]] = ..., grant_ref: _Optional[str] = ..., recovery_key_ref: _Optional[str] = ..., state: _Optional[str] = ..., created_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., updated_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., **kwargs) -> None: ...

class CredentialAck(_message.Message):
    __slots__ = ("binding_id", "consumer", "version", "verified_at")
    BINDING_ID_FIELD_NUMBER: _ClassVar[int]
    CONSUMER_FIELD_NUMBER: _ClassVar[int]
    VERSION_FIELD_NUMBER: _ClassVar[int]
    VERIFIED_AT_FIELD_NUMBER: _ClassVar[int]
    binding_id: str
    consumer: str
    version: int
    verified_at: _timestamp_pb2.Timestamp
    def __init__(self, binding_id: _Optional[str] = ..., consumer: _Optional[str] = ..., version: _Optional[int] = ..., verified_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ...) -> None: ...

class BindingView(_message.Message):
    __slots__ = ("binding", "acks", "lifecycle_state", "lifecycle_detail", "next_action")
    BINDING_FIELD_NUMBER: _ClassVar[int]
    ACKS_FIELD_NUMBER: _ClassVar[int]
    LIFECYCLE_STATE_FIELD_NUMBER: _ClassVar[int]
    LIFECYCLE_DETAIL_FIELD_NUMBER: _ClassVar[int]
    NEXT_ACTION_FIELD_NUMBER: _ClassVar[int]
    binding: CredentialBinding
    acks: _containers.RepeatedCompositeFieldContainer[CredentialAck]
    lifecycle_state: str
    lifecycle_detail: str
    next_action: str
    def __init__(self, binding: _Optional[_Union[CredentialBinding, _Mapping]] = ..., acks: _Optional[_Iterable[_Union[CredentialAck, _Mapping]]] = ..., lifecycle_state: _Optional[str] = ..., lifecycle_detail: _Optional[str] = ..., next_action: _Optional[str] = ...) -> None: ...

class ListBindingsRequest(_message.Message):
    __slots__ = ("deployment_id",)
    DEPLOYMENT_ID_FIELD_NUMBER: _ClassVar[int]
    deployment_id: str
    def __init__(self, deployment_id: _Optional[str] = ...) -> None: ...

class ListBindingsResponse(_message.Message):
    __slots__ = ("schema_version", "bindings")
    SCHEMA_VERSION_FIELD_NUMBER: _ClassVar[int]
    BINDINGS_FIELD_NUMBER: _ClassVar[int]
    schema_version: str
    bindings: _containers.RepeatedCompositeFieldContainer[BindingView]
    def __init__(self, schema_version: _Optional[str] = ..., bindings: _Optional[_Iterable[_Union[BindingView, _Mapping]]] = ...) -> None: ...

class ConsumerProgress(_message.Message):
    __slots__ = ("consumer", "state", "version", "reason", "updated_at")
    CONSUMER_FIELD_NUMBER: _ClassVar[int]
    STATE_FIELD_NUMBER: _ClassVar[int]
    VERSION_FIELD_NUMBER: _ClassVar[int]
    REASON_FIELD_NUMBER: _ClassVar[int]
    UPDATED_AT_FIELD_NUMBER: _ClassVar[int]
    consumer: str
    state: str
    version: int
    reason: str
    updated_at: _timestamp_pb2.Timestamp
    def __init__(self, consumer: _Optional[str] = ..., state: _Optional[str] = ..., version: _Optional[int] = ..., reason: _Optional[str] = ..., updated_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ...) -> None: ...

class Receipt(_message.Message):
    __slots__ = ("step", "state", "outcome", "at", "target_receipt_ref", "details", "limitations")
    STEP_FIELD_NUMBER: _ClassVar[int]
    STATE_FIELD_NUMBER: _ClassVar[int]
    OUTCOME_FIELD_NUMBER: _ClassVar[int]
    AT_FIELD_NUMBER: _ClassVar[int]
    TARGET_RECEIPT_REF_FIELD_NUMBER: _ClassVar[int]
    DETAILS_FIELD_NUMBER: _ClassVar[int]
    LIMITATIONS_FIELD_NUMBER: _ClassVar[int]
    step: str
    state: str
    outcome: str
    at: _timestamp_pb2.Timestamp
    target_receipt_ref: str
    details: _struct_pb2.Struct
    limitations: _containers.RepeatedScalarFieldContainer[str]
    def __init__(self, step: _Optional[str] = ..., state: _Optional[str] = ..., outcome: _Optional[str] = ..., at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., target_receipt_ref: _Optional[str] = ..., details: _Optional[_Union[_struct_pb2.Struct, _Mapping]] = ..., limitations: _Optional[_Iterable[str]] = ...) -> None: ...

class OperatorHandoff(_message.Message):
    __slots__ = ("reference", "provider", "instruction", "resume_with", "requested_at")
    REFERENCE_FIELD_NUMBER: _ClassVar[int]
    PROVIDER_FIELD_NUMBER: _ClassVar[int]
    INSTRUCTION_FIELD_NUMBER: _ClassVar[int]
    RESUME_WITH_FIELD_NUMBER: _ClassVar[int]
    REQUESTED_AT_FIELD_NUMBER: _ClassVar[int]
    reference: str
    provider: str
    instruction: str
    resume_with: str
    requested_at: _timestamp_pb2.Timestamp
    def __init__(self, reference: _Optional[str] = ..., provider: _Optional[str] = ..., instruction: _Optional[str] = ..., resume_with: _Optional[str] = ..., requested_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ...) -> None: ...

class BreakGlassWindow(_message.Message):
    __slots__ = ("scope", "operator", "issued_at", "expires_at", "predecessor_ref", "auto_revoked_at", "confirmation_ref")
    SCOPE_FIELD_NUMBER: _ClassVar[int]
    OPERATOR_FIELD_NUMBER: _ClassVar[int]
    ISSUED_AT_FIELD_NUMBER: _ClassVar[int]
    EXPIRES_AT_FIELD_NUMBER: _ClassVar[int]
    PREDECESSOR_REF_FIELD_NUMBER: _ClassVar[int]
    AUTO_REVOKED_AT_FIELD_NUMBER: _ClassVar[int]
    CONFIRMATION_REF_FIELD_NUMBER: _ClassVar[int]
    scope: str
    operator: str
    issued_at: _timestamp_pb2.Timestamp
    expires_at: _timestamp_pb2.Timestamp
    predecessor_ref: str
    auto_revoked_at: _timestamp_pb2.Timestamp
    confirmation_ref: str
    def __init__(self, scope: _Optional[str] = ..., operator: _Optional[str] = ..., issued_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., expires_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., predecessor_ref: _Optional[str] = ..., auto_revoked_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., confirmation_ref: _Optional[str] = ...) -> None: ...

class OperationError(_message.Message):
    __slots__ = ("code", "message")
    CODE_FIELD_NUMBER: _ClassVar[int]
    MESSAGE_FIELD_NUMBER: _ClassVar[int]
    code: str
    message: str
    def __init__(self, code: _Optional[str] = ..., message: _Optional[str] = ...) -> None: ...

class CredentialOperation(_message.Message):
    __slots__ = ("id", "deployment_id", "binding_id", "kind", "from_version", "to_version", "state", "consumers", "unreached", "pending_operator_input", "resume_after", "break_glass", "receipts", "error", "created_at", "updated_at", "completed_at")
    ID_FIELD_NUMBER: _ClassVar[int]
    DEPLOYMENT_ID_FIELD_NUMBER: _ClassVar[int]
    BINDING_ID_FIELD_NUMBER: _ClassVar[int]
    KIND_FIELD_NUMBER: _ClassVar[int]
    FROM_VERSION_FIELD_NUMBER: _ClassVar[int]
    TO_VERSION_FIELD_NUMBER: _ClassVar[int]
    STATE_FIELD_NUMBER: _ClassVar[int]
    CONSUMERS_FIELD_NUMBER: _ClassVar[int]
    UNREACHED_FIELD_NUMBER: _ClassVar[int]
    PENDING_OPERATOR_INPUT_FIELD_NUMBER: _ClassVar[int]
    RESUME_AFTER_FIELD_NUMBER: _ClassVar[int]
    BREAK_GLASS_FIELD_NUMBER: _ClassVar[int]
    RECEIPTS_FIELD_NUMBER: _ClassVar[int]
    ERROR_FIELD_NUMBER: _ClassVar[int]
    CREATED_AT_FIELD_NUMBER: _ClassVar[int]
    UPDATED_AT_FIELD_NUMBER: _ClassVar[int]
    COMPLETED_AT_FIELD_NUMBER: _ClassVar[int]
    id: str
    deployment_id: str
    binding_id: str
    kind: str
    from_version: int
    to_version: int
    state: str
    consumers: _containers.RepeatedCompositeFieldContainer[ConsumerProgress]
    unreached: _containers.RepeatedScalarFieldContainer[str]
    pending_operator_input: OperatorHandoff
    resume_after: _timestamp_pb2.Timestamp
    break_glass: BreakGlassWindow
    receipts: _containers.RepeatedCompositeFieldContainer[Receipt]
    error: OperationError
    created_at: _timestamp_pb2.Timestamp
    updated_at: _timestamp_pb2.Timestamp
    completed_at: _timestamp_pb2.Timestamp
    def __init__(self, id: _Optional[str] = ..., deployment_id: _Optional[str] = ..., binding_id: _Optional[str] = ..., kind: _Optional[str] = ..., from_version: _Optional[int] = ..., to_version: _Optional[int] = ..., state: _Optional[str] = ..., consumers: _Optional[_Iterable[_Union[ConsumerProgress, _Mapping]]] = ..., unreached: _Optional[_Iterable[str]] = ..., pending_operator_input: _Optional[_Union[OperatorHandoff, _Mapping]] = ..., resume_after: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., break_glass: _Optional[_Union[BreakGlassWindow, _Mapping]] = ..., receipts: _Optional[_Iterable[_Union[Receipt, _Mapping]]] = ..., error: _Optional[_Union[OperationError, _Mapping]] = ..., created_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., updated_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., completed_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ...) -> None: ...

class CredentialOperationResponse(_message.Message):
    __slots__ = ("schema_version", "operation")
    SCHEMA_VERSION_FIELD_NUMBER: _ClassVar[int]
    OPERATION_FIELD_NUMBER: _ClassVar[int]
    schema_version: str
    operation: CredentialOperation
    def __init__(self, schema_version: _Optional[str] = ..., operation: _Optional[_Union[CredentialOperation, _Mapping]] = ...) -> None: ...

class RotateCredentialRequest(_message.Message):
    __slots__ = ("deployment_id", "binding_id", "value", "request_key")
    DEPLOYMENT_ID_FIELD_NUMBER: _ClassVar[int]
    BINDING_ID_FIELD_NUMBER: _ClassVar[int]
    VALUE_FIELD_NUMBER: _ClassVar[int]
    REQUEST_KEY_FIELD_NUMBER: _ClassVar[int]
    deployment_id: str
    binding_id: str
    value: str
    request_key: str
    def __init__(self, deployment_id: _Optional[str] = ..., binding_id: _Optional[str] = ..., value: _Optional[str] = ..., request_key: _Optional[str] = ...) -> None: ...

class RevokeCredentialRequest(_message.Message):
    __slots__ = ("deployment_id", "binding_id", "request_key")
    DEPLOYMENT_ID_FIELD_NUMBER: _ClassVar[int]
    BINDING_ID_FIELD_NUMBER: _ClassVar[int]
    REQUEST_KEY_FIELD_NUMBER: _ClassVar[int]
    deployment_id: str
    binding_id: str
    request_key: str
    def __init__(self, deployment_id: _Optional[str] = ..., binding_id: _Optional[str] = ..., request_key: _Optional[str] = ...) -> None: ...

class RecoverCredentialsRequest(_message.Message):
    __slots__ = ("deployment_id", "bundle_ref", "passphrase", "request_key")
    DEPLOYMENT_ID_FIELD_NUMBER: _ClassVar[int]
    BUNDLE_REF_FIELD_NUMBER: _ClassVar[int]
    PASSPHRASE_FIELD_NUMBER: _ClassVar[int]
    REQUEST_KEY_FIELD_NUMBER: _ClassVar[int]
    deployment_id: str
    bundle_ref: str
    passphrase: str
    request_key: str
    def __init__(self, deployment_id: _Optional[str] = ..., bundle_ref: _Optional[str] = ..., passphrase: _Optional[str] = ..., request_key: _Optional[str] = ...) -> None: ...

class GetRotationRequest(_message.Message):
    __slots__ = ("deployment_id", "rotation_id")
    DEPLOYMENT_ID_FIELD_NUMBER: _ClassVar[int]
    ROTATION_ID_FIELD_NUMBER: _ClassVar[int]
    deployment_id: str
    rotation_id: str
    def __init__(self, deployment_id: _Optional[str] = ..., rotation_id: _Optional[str] = ...) -> None: ...

class ResumeRotationRequest(_message.Message):
    __slots__ = ("deployment_id", "rotation_id", "operator_confirmed")
    DEPLOYMENT_ID_FIELD_NUMBER: _ClassVar[int]
    ROTATION_ID_FIELD_NUMBER: _ClassVar[int]
    OPERATOR_CONFIRMED_FIELD_NUMBER: _ClassVar[int]
    deployment_id: str
    rotation_id: str
    operator_confirmed: bool
    def __init__(self, deployment_id: _Optional[str] = ..., rotation_id: _Optional[str] = ..., operator_confirmed: _Optional[bool] = ...) -> None: ...

class BreakGlassRequest(_message.Message):
    __slots__ = ("deployment_id", "binding_id", "scope", "window_seconds", "confirmation", "operator", "request_key")
    DEPLOYMENT_ID_FIELD_NUMBER: _ClassVar[int]
    BINDING_ID_FIELD_NUMBER: _ClassVar[int]
    SCOPE_FIELD_NUMBER: _ClassVar[int]
    WINDOW_SECONDS_FIELD_NUMBER: _ClassVar[int]
    CONFIRMATION_FIELD_NUMBER: _ClassVar[int]
    OPERATOR_FIELD_NUMBER: _ClassVar[int]
    REQUEST_KEY_FIELD_NUMBER: _ClassVar[int]
    deployment_id: str
    binding_id: str
    scope: str
    window_seconds: int
    confirmation: str
    operator: str
    request_key: str
    def __init__(self, deployment_id: _Optional[str] = ..., binding_id: _Optional[str] = ..., scope: _Optional[str] = ..., window_seconds: _Optional[int] = ..., confirmation: _Optional[str] = ..., operator: _Optional[str] = ..., request_key: _Optional[str] = ...) -> None: ...
