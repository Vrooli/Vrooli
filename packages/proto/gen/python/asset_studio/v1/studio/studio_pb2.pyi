from google.protobuf.internal import containers as _containers
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Iterable as _Iterable, Mapping as _Mapping
from typing import ClassVar as _ClassVar, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class ConditioningReference(_message.Message):
    __slots__ = ("kind", "id", "version")
    KIND_FIELD_NUMBER: _ClassVar[int]
    ID_FIELD_NUMBER: _ClassVar[int]
    VERSION_FIELD_NUMBER: _ClassVar[int]
    kind: str
    id: str
    version: str
    def __init__(self, kind: _Optional[str] = ..., id: _Optional[str] = ..., version: _Optional[str] = ...) -> None: ...

class Identity(_message.Message):
    __slots__ = ("id", "name", "kind", "version", "traits", "reference_images", "conditioning_references", "credential_claims", "referenced")
    class TraitsEntry(_message.Message):
        __slots__ = ("key", "value")
        KEY_FIELD_NUMBER: _ClassVar[int]
        VALUE_FIELD_NUMBER: _ClassVar[int]
        key: str
        value: str
        def __init__(self, key: _Optional[str] = ..., value: _Optional[str] = ...) -> None: ...
    ID_FIELD_NUMBER: _ClassVar[int]
    NAME_FIELD_NUMBER: _ClassVar[int]
    KIND_FIELD_NUMBER: _ClassVar[int]
    VERSION_FIELD_NUMBER: _ClassVar[int]
    TRAITS_FIELD_NUMBER: _ClassVar[int]
    REFERENCE_IMAGES_FIELD_NUMBER: _ClassVar[int]
    CONDITIONING_REFERENCES_FIELD_NUMBER: _ClassVar[int]
    CREDENTIAL_CLAIMS_FIELD_NUMBER: _ClassVar[int]
    REFERENCED_FIELD_NUMBER: _ClassVar[int]
    id: str
    name: str
    kind: str
    version: int
    traits: _containers.ScalarMap[str, str]
    reference_images: _containers.RepeatedScalarFieldContainer[str]
    conditioning_references: _containers.RepeatedCompositeFieldContainer[ConditioningReference]
    credential_claims: str
    referenced: bool
    def __init__(self, id: _Optional[str] = ..., name: _Optional[str] = ..., kind: _Optional[str] = ..., version: _Optional[int] = ..., traits: _Optional[_Mapping[str, str]] = ..., reference_images: _Optional[_Iterable[str]] = ..., conditioning_references: _Optional[_Iterable[_Union[ConditioningReference, _Mapping]]] = ..., credential_claims: _Optional[str] = ..., referenced: _Optional[bool] = ...) -> None: ...

class AssetReference(_message.Message):
    __slots__ = ("id", "status", "alt_text", "disclosure", "ai_generated", "width", "height", "media_type")
    ID_FIELD_NUMBER: _ClassVar[int]
    STATUS_FIELD_NUMBER: _ClassVar[int]
    ALT_TEXT_FIELD_NUMBER: _ClassVar[int]
    DISCLOSURE_FIELD_NUMBER: _ClassVar[int]
    AI_GENERATED_FIELD_NUMBER: _ClassVar[int]
    WIDTH_FIELD_NUMBER: _ClassVar[int]
    HEIGHT_FIELD_NUMBER: _ClassVar[int]
    MEDIA_TYPE_FIELD_NUMBER: _ClassVar[int]
    id: str
    status: str
    alt_text: str
    disclosure: str
    ai_generated: bool
    width: int
    height: int
    media_type: str
    def __init__(self, id: _Optional[str] = ..., status: _Optional[str] = ..., alt_text: _Optional[str] = ..., disclosure: _Optional[str] = ..., ai_generated: _Optional[bool] = ..., width: _Optional[int] = ..., height: _Optional[int] = ..., media_type: _Optional[str] = ...) -> None: ...

class ListIdentitiesRequest(_message.Message):
    __slots__ = ("kind",)
    KIND_FIELD_NUMBER: _ClassVar[int]
    kind: str
    def __init__(self, kind: _Optional[str] = ...) -> None: ...

class ListIdentitiesResponse(_message.Message):
    __slots__ = ("identities",)
    IDENTITIES_FIELD_NUMBER: _ClassVar[int]
    identities: _containers.RepeatedCompositeFieldContainer[Identity]
    def __init__(self, identities: _Optional[_Iterable[_Union[Identity, _Mapping]]] = ...) -> None: ...

class CreateIdentityRequest(_message.Message):
    __slots__ = ("identity", "actor_id", "actor_kind")
    IDENTITY_FIELD_NUMBER: _ClassVar[int]
    ACTOR_ID_FIELD_NUMBER: _ClassVar[int]
    ACTOR_KIND_FIELD_NUMBER: _ClassVar[int]
    identity: Identity
    actor_id: str
    actor_kind: str
    def __init__(self, identity: _Optional[_Union[Identity, _Mapping]] = ..., actor_id: _Optional[str] = ..., actor_kind: _Optional[str] = ...) -> None: ...

class CreateIdentityResponse(_message.Message):
    __slots__ = ("identity",)
    IDENTITY_FIELD_NUMBER: _ClassVar[int]
    identity: Identity
    def __init__(self, identity: _Optional[_Union[Identity, _Mapping]] = ...) -> None: ...

class ReviseIdentityRequest(_message.Message):
    __slots__ = ("identity", "actor_id", "actor_kind")
    IDENTITY_FIELD_NUMBER: _ClassVar[int]
    ACTOR_ID_FIELD_NUMBER: _ClassVar[int]
    ACTOR_KIND_FIELD_NUMBER: _ClassVar[int]
    identity: Identity
    actor_id: str
    actor_kind: str
    def __init__(self, identity: _Optional[_Union[Identity, _Mapping]] = ..., actor_id: _Optional[str] = ..., actor_kind: _Optional[str] = ...) -> None: ...

class ReviseIdentityResponse(_message.Message):
    __slots__ = ("identity",)
    IDENTITY_FIELD_NUMBER: _ClassVar[int]
    identity: Identity
    def __init__(self, identity: _Optional[_Union[Identity, _Mapping]] = ...) -> None: ...

class ResolveSpecRequest(_message.Message):
    __slots__ = ("template", "fields", "identity_version_ids", "campaign_ref")
    class FieldsEntry(_message.Message):
        __slots__ = ("key", "value")
        KEY_FIELD_NUMBER: _ClassVar[int]
        VALUE_FIELD_NUMBER: _ClassVar[int]
        key: str
        value: str
        def __init__(self, key: _Optional[str] = ..., value: _Optional[str] = ...) -> None: ...
    TEMPLATE_FIELD_NUMBER: _ClassVar[int]
    FIELDS_FIELD_NUMBER: _ClassVar[int]
    IDENTITY_VERSION_IDS_FIELD_NUMBER: _ClassVar[int]
    CAMPAIGN_REF_FIELD_NUMBER: _ClassVar[int]
    template: str
    fields: _containers.ScalarMap[str, str]
    identity_version_ids: _containers.RepeatedScalarFieldContainer[str]
    campaign_ref: str
    def __init__(self, template: _Optional[str] = ..., fields: _Optional[_Mapping[str, str]] = ..., identity_version_ids: _Optional[_Iterable[str]] = ..., campaign_ref: _Optional[str] = ...) -> None: ...

class ResolveSpecResponse(_message.Message):
    __slots__ = ("spec_id", "resolved_payload")
    SPEC_ID_FIELD_NUMBER: _ClassVar[int]
    RESOLVED_PAYLOAD_FIELD_NUMBER: _ClassVar[int]
    spec_id: str
    resolved_payload: str
    def __init__(self, spec_id: _Optional[str] = ..., resolved_payload: _Optional[str] = ...) -> None: ...

class CreateRenderRequest(_message.Message):
    __slots__ = ("spec_id", "estimated_cost", "candidate_count")
    SPEC_ID_FIELD_NUMBER: _ClassVar[int]
    ESTIMATED_COST_FIELD_NUMBER: _ClassVar[int]
    CANDIDATE_COUNT_FIELD_NUMBER: _ClassVar[int]
    spec_id: str
    estimated_cost: float
    candidate_count: int
    def __init__(self, spec_id: _Optional[str] = ..., estimated_cost: _Optional[float] = ..., candidate_count: _Optional[int] = ...) -> None: ...

class CreateRenderResponse(_message.Message):
    __slots__ = ("render_id", "status", "candidates")
    RENDER_ID_FIELD_NUMBER: _ClassVar[int]
    STATUS_FIELD_NUMBER: _ClassVar[int]
    CANDIDATES_FIELD_NUMBER: _ClassVar[int]
    render_id: str
    status: str
    candidates: _containers.RepeatedCompositeFieldContainer[AssetReference]
    def __init__(self, render_id: _Optional[str] = ..., status: _Optional[str] = ..., candidates: _Optional[_Iterable[_Union[AssetReference, _Mapping]]] = ...) -> None: ...

class SelectCandidateRequest(_message.Message):
    __slots__ = ("asset_id",)
    ASSET_ID_FIELD_NUMBER: _ClassVar[int]
    asset_id: str
    def __init__(self, asset_id: _Optional[str] = ...) -> None: ...

class SelectCandidateResponse(_message.Message):
    __slots__ = ("selected",)
    SELECTED_FIELD_NUMBER: _ClassVar[int]
    selected: AssetReference
    def __init__(self, selected: _Optional[_Union[AssetReference, _Mapping]] = ...) -> None: ...

class JudgeConformanceRequest(_message.Message):
    __slots__ = ("asset_id", "identity_version_id", "actor_id", "actor_kind", "passed", "basis")
    ASSET_ID_FIELD_NUMBER: _ClassVar[int]
    IDENTITY_VERSION_ID_FIELD_NUMBER: _ClassVar[int]
    ACTOR_ID_FIELD_NUMBER: _ClassVar[int]
    ACTOR_KIND_FIELD_NUMBER: _ClassVar[int]
    PASSED_FIELD_NUMBER: _ClassVar[int]
    BASIS_FIELD_NUMBER: _ClassVar[int]
    asset_id: str
    identity_version_id: str
    actor_id: str
    actor_kind: str
    passed: bool
    basis: str
    def __init__(self, asset_id: _Optional[str] = ..., identity_version_id: _Optional[str] = ..., actor_id: _Optional[str] = ..., actor_kind: _Optional[str] = ..., passed: _Optional[bool] = ..., basis: _Optional[str] = ...) -> None: ...

class JudgeConformanceResponse(_message.Message):
    __slots__ = ()
    def __init__(self) -> None: ...

class ReleaseAssetRequest(_message.Message):
    __slots__ = ("asset_id",)
    ASSET_ID_FIELD_NUMBER: _ClassVar[int]
    asset_id: str
    def __init__(self, asset_id: _Optional[str] = ...) -> None: ...

class ReleaseAssetResponse(_message.Message):
    __slots__ = ("asset",)
    ASSET_FIELD_NUMBER: _ClassVar[int]
    asset: AssetReference
    def __init__(self, asset: _Optional[_Union[AssetReference, _Mapping]] = ...) -> None: ...

class GetReleasedAssetReferenceRequest(_message.Message):
    __slots__ = ("asset_id",)
    ASSET_ID_FIELD_NUMBER: _ClassVar[int]
    asset_id: str
    def __init__(self, asset_id: _Optional[str] = ...) -> None: ...

class GetReleasedAssetReferenceResponse(_message.Message):
    __slots__ = ("asset",)
    ASSET_FIELD_NUMBER: _ClassVar[int]
    asset: AssetReference
    def __init__(self, asset: _Optional[_Union[AssetReference, _Mapping]] = ...) -> None: ...

class ImportCanonRequest(_message.Message):
    __slots__ = ("root",)
    ROOT_FIELD_NUMBER: _ClassVar[int]
    root: str
    def __init__(self, root: _Optional[str] = ...) -> None: ...

class ImportCanonResponse(_message.Message):
    __slots__ = ("created", "revised", "errors")
    CREATED_FIELD_NUMBER: _ClassVar[int]
    REVISED_FIELD_NUMBER: _ClassVar[int]
    ERRORS_FIELD_NUMBER: _ClassVar[int]
    created: int
    revised: int
    errors: _containers.RepeatedScalarFieldContainer[str]
    def __init__(self, created: _Optional[int] = ..., revised: _Optional[int] = ..., errors: _Optional[_Iterable[str]] = ...) -> None: ...
