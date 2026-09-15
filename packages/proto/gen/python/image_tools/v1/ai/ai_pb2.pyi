from google.protobuf.internal import containers as _containers
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Iterable as _Iterable, Mapping as _Mapping
from typing import ClassVar as _ClassVar, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class AIOperationInfo(_message.Message):
    __slots__ = ("name", "category", "summary", "requires_image", "requires_mask", "prompt_driven", "default_model_id")
    NAME_FIELD_NUMBER: _ClassVar[int]
    CATEGORY_FIELD_NUMBER: _ClassVar[int]
    SUMMARY_FIELD_NUMBER: _ClassVar[int]
    REQUIRES_IMAGE_FIELD_NUMBER: _ClassVar[int]
    REQUIRES_MASK_FIELD_NUMBER: _ClassVar[int]
    PROMPT_DRIVEN_FIELD_NUMBER: _ClassVar[int]
    DEFAULT_MODEL_ID_FIELD_NUMBER: _ClassVar[int]
    name: str
    category: str
    summary: str
    requires_image: bool
    requires_mask: bool
    prompt_driven: bool
    default_model_id: str
    def __init__(self, name: _Optional[str] = ..., category: _Optional[str] = ..., summary: _Optional[str] = ..., requires_image: _Optional[bool] = ..., requires_mask: _Optional[bool] = ..., prompt_driven: _Optional[bool] = ..., default_model_id: _Optional[str] = ...) -> None: ...

class ListAIOperationsRequest(_message.Message):
    __slots__ = ()
    def __init__(self) -> None: ...

class ListAIOperationsResponse(_message.Message):
    __slots__ = ("operations",)
    OPERATIONS_FIELD_NUMBER: _ClassVar[int]
    operations: _containers.RepeatedCompositeFieldContainer[AIOperationInfo]
    def __init__(self, operations: _Optional[_Iterable[_Union[AIOperationInfo, _Mapping]]] = ...) -> None: ...

class AIParams(_message.Message):
    __slots__ = ("prompt", "negative_prompt", "seed", "width", "height", "steps", "cfg_scale", "variations", "strength", "scale", "model_override", "allow_byok", "auto_scan_nsfw", "realism", "face_aware", "consent_affirmed", "adapters", "quality_policy", "fallback_policy", "priority", "allow_reclaim", "openrouter_role")
    PROMPT_FIELD_NUMBER: _ClassVar[int]
    NEGATIVE_PROMPT_FIELD_NUMBER: _ClassVar[int]
    SEED_FIELD_NUMBER: _ClassVar[int]
    WIDTH_FIELD_NUMBER: _ClassVar[int]
    HEIGHT_FIELD_NUMBER: _ClassVar[int]
    STEPS_FIELD_NUMBER: _ClassVar[int]
    CFG_SCALE_FIELD_NUMBER: _ClassVar[int]
    VARIATIONS_FIELD_NUMBER: _ClassVar[int]
    STRENGTH_FIELD_NUMBER: _ClassVar[int]
    SCALE_FIELD_NUMBER: _ClassVar[int]
    MODEL_OVERRIDE_FIELD_NUMBER: _ClassVar[int]
    ALLOW_BYOK_FIELD_NUMBER: _ClassVar[int]
    AUTO_SCAN_NSFW_FIELD_NUMBER: _ClassVar[int]
    REALISM_FIELD_NUMBER: _ClassVar[int]
    FACE_AWARE_FIELD_NUMBER: _ClassVar[int]
    CONSENT_AFFIRMED_FIELD_NUMBER: _ClassVar[int]
    ADAPTERS_FIELD_NUMBER: _ClassVar[int]
    QUALITY_POLICY_FIELD_NUMBER: _ClassVar[int]
    FALLBACK_POLICY_FIELD_NUMBER: _ClassVar[int]
    PRIORITY_FIELD_NUMBER: _ClassVar[int]
    ALLOW_RECLAIM_FIELD_NUMBER: _ClassVar[int]
    OPENROUTER_ROLE_FIELD_NUMBER: _ClassVar[int]
    prompt: str
    negative_prompt: str
    seed: int
    width: int
    height: int
    steps: int
    cfg_scale: float
    variations: int
    strength: float
    scale: int
    model_override: str
    allow_byok: bool
    auto_scan_nsfw: bool
    realism: float
    face_aware: bool
    consent_affirmed: bool
    adapters: _containers.RepeatedCompositeFieldContainer[AdapterRef]
    quality_policy: str
    fallback_policy: str
    priority: str
    allow_reclaim: bool
    openrouter_role: str
    def __init__(self, prompt: _Optional[str] = ..., negative_prompt: _Optional[str] = ..., seed: _Optional[int] = ..., width: _Optional[int] = ..., height: _Optional[int] = ..., steps: _Optional[int] = ..., cfg_scale: _Optional[float] = ..., variations: _Optional[int] = ..., strength: _Optional[float] = ..., scale: _Optional[int] = ..., model_override: _Optional[str] = ..., allow_byok: _Optional[bool] = ..., auto_scan_nsfw: _Optional[bool] = ..., realism: _Optional[float] = ..., face_aware: _Optional[bool] = ..., consent_affirmed: _Optional[bool] = ..., adapters: _Optional[_Iterable[_Union[AdapterRef, _Mapping]]] = ..., quality_policy: _Optional[str] = ..., fallback_policy: _Optional[str] = ..., priority: _Optional[str] = ..., allow_reclaim: _Optional[bool] = ..., openrouter_role: _Optional[str] = ...) -> None: ...

class AdapterRef(_message.Message):
    __slots__ = ("adapter_id", "scale", "conditioning_image_key", "preprocessor_override")
    ADAPTER_ID_FIELD_NUMBER: _ClassVar[int]
    SCALE_FIELD_NUMBER: _ClassVar[int]
    CONDITIONING_IMAGE_KEY_FIELD_NUMBER: _ClassVar[int]
    PREPROCESSOR_OVERRIDE_FIELD_NUMBER: _ClassVar[int]
    adapter_id: str
    scale: float
    conditioning_image_key: str
    preprocessor_override: str
    def __init__(self, adapter_id: _Optional[str] = ..., scale: _Optional[float] = ..., conditioning_image_key: _Optional[str] = ..., preprocessor_override: _Optional[str] = ...) -> None: ...

class SubmitAIResponse(_message.Message):
    __slots__ = ("job_id", "estimated_seconds", "model_id", "tier", "warnings")
    JOB_ID_FIELD_NUMBER: _ClassVar[int]
    ESTIMATED_SECONDS_FIELD_NUMBER: _ClassVar[int]
    MODEL_ID_FIELD_NUMBER: _ClassVar[int]
    TIER_FIELD_NUMBER: _ClassVar[int]
    WARNINGS_FIELD_NUMBER: _ClassVar[int]
    job_id: str
    estimated_seconds: int
    model_id: str
    tier: str
    warnings: _containers.RepeatedScalarFieldContainer[str]
    def __init__(self, job_id: _Optional[str] = ..., estimated_seconds: _Optional[int] = ..., model_id: _Optional[str] = ..., tier: _Optional[str] = ..., warnings: _Optional[_Iterable[str]] = ...) -> None: ...
