from google.protobuf.internal import containers as _containers
from google.protobuf.internal import enum_type_wrapper as _enum_type_wrapper
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Iterable as _Iterable, Mapping as _Mapping
from typing import ClassVar as _ClassVar, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class DeploymentTier(int, metaclass=_enum_type_wrapper.EnumTypeWrapper):
    __slots__ = ()
    DEPLOYMENT_TIER_UNSPECIFIED: _ClassVar[DeploymentTier]
    DEPLOYMENT_TIER_LOCAL: _ClassVar[DeploymentTier]
    DEPLOYMENT_TIER_PUBLIC: _ClassVar[DeploymentTier]

class ConsentWeight(int, metaclass=_enum_type_wrapper.EnumTypeWrapper):
    __slots__ = ()
    CONSENT_WEIGHT_UNSPECIFIED: _ClassVar[ConsentWeight]
    CONSENT_WEIGHT_NONE: _ClassVar[ConsentWeight]
    CONSENT_WEIGHT_LOW: _ClassVar[ConsentWeight]
    CONSENT_WEIGHT_HIGH: _ClassVar[ConsentWeight]
DEPLOYMENT_TIER_UNSPECIFIED: DeploymentTier
DEPLOYMENT_TIER_LOCAL: DeploymentTier
DEPLOYMENT_TIER_PUBLIC: DeploymentTier
CONSENT_WEIGHT_UNSPECIFIED: ConsentWeight
CONSENT_WEIGHT_NONE: ConsentWeight
CONSENT_WEIGHT_LOW: ConsentWeight
CONSENT_WEIGHT_HIGH: ConsentWeight

class OpWeight(_message.Message):
    __slots__ = ("operation", "weight")
    OPERATION_FIELD_NUMBER: _ClassVar[int]
    WEIGHT_FIELD_NUMBER: _ClassVar[int]
    operation: str
    weight: ConsentWeight
    def __init__(self, operation: _Optional[str] = ..., weight: _Optional[_Union[ConsentWeight, str]] = ...) -> None: ...

class GetPolicyRequest(_message.Message):
    __slots__ = ()
    def __init__(self) -> None: ...

class SafetyPolicy(_message.Message):
    __slots__ = ("tier", "require_consent", "force_nsfw_scan", "require_provenance", "rate_limit_per_min", "op_weights", "summary")
    TIER_FIELD_NUMBER: _ClassVar[int]
    REQUIRE_CONSENT_FIELD_NUMBER: _ClassVar[int]
    FORCE_NSFW_SCAN_FIELD_NUMBER: _ClassVar[int]
    REQUIRE_PROVENANCE_FIELD_NUMBER: _ClassVar[int]
    RATE_LIMIT_PER_MIN_FIELD_NUMBER: _ClassVar[int]
    OP_WEIGHTS_FIELD_NUMBER: _ClassVar[int]
    SUMMARY_FIELD_NUMBER: _ClassVar[int]
    tier: DeploymentTier
    require_consent: bool
    force_nsfw_scan: bool
    require_provenance: bool
    rate_limit_per_min: int
    op_weights: _containers.RepeatedCompositeFieldContainer[OpWeight]
    summary: str
    def __init__(self, tier: _Optional[_Union[DeploymentTier, str]] = ..., require_consent: _Optional[bool] = ..., force_nsfw_scan: _Optional[bool] = ..., require_provenance: _Optional[bool] = ..., rate_limit_per_min: _Optional[int] = ..., op_weights: _Optional[_Iterable[_Union[OpWeight, _Mapping]]] = ..., summary: _Optional[str] = ...) -> None: ...
