from google.protobuf.internal import containers as _containers
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Iterable as _Iterable, Mapping as _Mapping
from typing import ClassVar as _ClassVar, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class GetRoleModelChoicesRequest(_message.Message):
    __slots__ = ("agent_id", "target_id")
    AGENT_ID_FIELD_NUMBER: _ClassVar[int]
    TARGET_ID_FIELD_NUMBER: _ClassVar[int]
    agent_id: str
    target_id: str
    def __init__(self, agent_id: _Optional[str] = ..., target_id: _Optional[str] = ...) -> None: ...

class RoleModelChoice(_message.Message):
    __slots__ = ("model", "source", "provider", "launch_model", "effort", "source_label", "is_default")
    MODEL_FIELD_NUMBER: _ClassVar[int]
    SOURCE_FIELD_NUMBER: _ClassVar[int]
    PROVIDER_FIELD_NUMBER: _ClassVar[int]
    LAUNCH_MODEL_FIELD_NUMBER: _ClassVar[int]
    EFFORT_FIELD_NUMBER: _ClassVar[int]
    SOURCE_LABEL_FIELD_NUMBER: _ClassVar[int]
    IS_DEFAULT_FIELD_NUMBER: _ClassVar[int]
    model: str
    source: str
    provider: str
    launch_model: str
    effort: str
    source_label: str
    is_default: bool
    def __init__(self, model: _Optional[str] = ..., source: _Optional[str] = ..., provider: _Optional[str] = ..., launch_model: _Optional[str] = ..., effort: _Optional[str] = ..., source_label: _Optional[str] = ..., is_default: _Optional[bool] = ...) -> None: ...

class GetRoleModelChoicesResponse(_message.Message):
    __slots__ = ("policy_status", "availability", "runner", "role", "provenance", "observed_at", "policy_digest", "choices", "detail", "model_flag")
    POLICY_STATUS_FIELD_NUMBER: _ClassVar[int]
    AVAILABILITY_FIELD_NUMBER: _ClassVar[int]
    RUNNER_FIELD_NUMBER: _ClassVar[int]
    ROLE_FIELD_NUMBER: _ClassVar[int]
    PROVENANCE_FIELD_NUMBER: _ClassVar[int]
    OBSERVED_AT_FIELD_NUMBER: _ClassVar[int]
    POLICY_DIGEST_FIELD_NUMBER: _ClassVar[int]
    CHOICES_FIELD_NUMBER: _ClassVar[int]
    DETAIL_FIELD_NUMBER: _ClassVar[int]
    MODEL_FLAG_FIELD_NUMBER: _ClassVar[int]
    policy_status: str
    availability: str
    runner: str
    role: str
    provenance: str
    observed_at: str
    policy_digest: str
    choices: _containers.RepeatedCompositeFieldContainer[RoleModelChoice]
    detail: str
    model_flag: str
    def __init__(self, policy_status: _Optional[str] = ..., availability: _Optional[str] = ..., runner: _Optional[str] = ..., role: _Optional[str] = ..., provenance: _Optional[str] = ..., observed_at: _Optional[str] = ..., policy_digest: _Optional[str] = ..., choices: _Optional[_Iterable[_Union[RoleModelChoice, _Mapping]]] = ..., detail: _Optional[str] = ..., model_flag: _Optional[str] = ...) -> None: ...
