import datetime

from google.protobuf import timestamp_pb2 as _timestamp_pb2
from google.protobuf import struct_pb2 as _struct_pb2
from ai_gateway.v1.inference import inference_pb2 as _inference_pb2
from ai_gateway.v1.shared import gateway_pb2 as _gateway_pb2
from google.protobuf.internal import containers as _containers
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Iterable as _Iterable, Mapping as _Mapping
from typing import ClassVar as _ClassVar, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class ListBundleRevisionsRequest(_message.Message):
    __slots__ = ("after_bundle_id", "after_revision", "page_size")
    AFTER_BUNDLE_ID_FIELD_NUMBER: _ClassVar[int]
    AFTER_REVISION_FIELD_NUMBER: _ClassVar[int]
    PAGE_SIZE_FIELD_NUMBER: _ClassVar[int]
    after_bundle_id: str
    after_revision: int
    page_size: int
    def __init__(self, after_bundle_id: _Optional[str] = ..., after_revision: _Optional[int] = ..., page_size: _Optional[int] = ...) -> None: ...

class ListBundleRevisionsResponse(_message.Message):
    __slots__ = ("revisions", "next_after_bundle_id", "next_after_revision")
    REVISIONS_FIELD_NUMBER: _ClassVar[int]
    NEXT_AFTER_BUNDLE_ID_FIELD_NUMBER: _ClassVar[int]
    NEXT_AFTER_REVISION_FIELD_NUMBER: _ClassVar[int]
    revisions: _containers.RepeatedCompositeFieldContainer[BundleRevisionMetadata]
    next_after_bundle_id: str
    next_after_revision: int
    def __init__(self, revisions: _Optional[_Iterable[_Union[BundleRevisionMetadata, _Mapping]]] = ..., next_after_bundle_id: _Optional[str] = ..., next_after_revision: _Optional[int] = ...) -> None: ...

class GetBundleRevisionRequest(_message.Message):
    __slots__ = ("bundle_id", "revision")
    BUNDLE_ID_FIELD_NUMBER: _ClassVar[int]
    REVISION_FIELD_NUMBER: _ClassVar[int]
    bundle_id: str
    revision: int
    def __init__(self, bundle_id: _Optional[str] = ..., revision: _Optional[int] = ...) -> None: ...

class GetBundleRevisionResponse(_message.Message):
    __slots__ = ("revision",)
    REVISION_FIELD_NUMBER: _ClassVar[int]
    revision: BundleRevisionMetadata
    def __init__(self, revision: _Optional[_Union[BundleRevisionMetadata, _Mapping]] = ...) -> None: ...

class BundleRevisionMetadata(_message.Message):
    __slots__ = ("manifest", "manifest_sha256", "review", "review_sha256", "published_at")
    MANIFEST_FIELD_NUMBER: _ClassVar[int]
    MANIFEST_SHA256_FIELD_NUMBER: _ClassVar[int]
    REVIEW_FIELD_NUMBER: _ClassVar[int]
    REVIEW_SHA256_FIELD_NUMBER: _ClassVar[int]
    PUBLISHED_AT_FIELD_NUMBER: _ClassVar[int]
    manifest: BundleManifest
    manifest_sha256: str
    review: ReviewBinding
    review_sha256: str
    published_at: _timestamp_pb2.Timestamp
    def __init__(self, manifest: _Optional[_Union[BundleManifest, _Mapping]] = ..., manifest_sha256: _Optional[str] = ..., review: _Optional[_Union[ReviewBinding, _Mapping]] = ..., review_sha256: _Optional[str] = ..., published_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ...) -> None: ...

class BundleManifest(_message.Message):
    __slots__ = ("version", "bundle_id", "revision", "artifacts")
    VERSION_FIELD_NUMBER: _ClassVar[int]
    BUNDLE_ID_FIELD_NUMBER: _ClassVar[int]
    REVISION_FIELD_NUMBER: _ClassVar[int]
    ARTIFACTS_FIELD_NUMBER: _ClassVar[int]
    version: int
    bundle_id: str
    revision: int
    artifacts: _containers.RepeatedCompositeFieldContainer[Artifact]
    def __init__(self, version: _Optional[int] = ..., bundle_id: _Optional[str] = ..., revision: _Optional[int] = ..., artifacts: _Optional[_Iterable[_Union[Artifact, _Mapping]]] = ...) -> None: ...

class Artifact(_message.Message):
    __slots__ = ("id", "role", "mime", "size_bytes", "sha256", "license_id", "attribution", "provenance", "rights")
    ID_FIELD_NUMBER: _ClassVar[int]
    ROLE_FIELD_NUMBER: _ClassVar[int]
    MIME_FIELD_NUMBER: _ClassVar[int]
    SIZE_BYTES_FIELD_NUMBER: _ClassVar[int]
    SHA256_FIELD_NUMBER: _ClassVar[int]
    LICENSE_ID_FIELD_NUMBER: _ClassVar[int]
    ATTRIBUTION_FIELD_NUMBER: _ClassVar[int]
    PROVENANCE_FIELD_NUMBER: _ClassVar[int]
    RIGHTS_FIELD_NUMBER: _ClassVar[int]
    id: str
    role: str
    mime: str
    size_bytes: int
    sha256: str
    license_id: str
    attribution: str
    provenance: ArtifactProvenance
    rights: _containers.RepeatedScalarFieldContainer[str]
    def __init__(self, id: _Optional[str] = ..., role: _Optional[str] = ..., mime: _Optional[str] = ..., size_bytes: _Optional[int] = ..., sha256: _Optional[str] = ..., license_id: _Optional[str] = ..., attribution: _Optional[str] = ..., provenance: _Optional[_Union[ArtifactProvenance, _Mapping]] = ..., rights: _Optional[_Iterable[str]] = ...) -> None: ...

class ArtifactProvenance(_message.Message):
    __slots__ = ("kind", "source_label", "source_sha256", "derived_from_artifact_id", "adaptation")
    KIND_FIELD_NUMBER: _ClassVar[int]
    SOURCE_LABEL_FIELD_NUMBER: _ClassVar[int]
    SOURCE_SHA256_FIELD_NUMBER: _ClassVar[int]
    DERIVED_FROM_ARTIFACT_ID_FIELD_NUMBER: _ClassVar[int]
    ADAPTATION_FIELD_NUMBER: _ClassVar[int]
    kind: str
    source_label: str
    source_sha256: str
    derived_from_artifact_id: str
    adaptation: str
    def __init__(self, kind: _Optional[str] = ..., source_label: _Optional[str] = ..., source_sha256: _Optional[str] = ..., derived_from_artifact_id: _Optional[str] = ..., adaptation: _Optional[str] = ...) -> None: ...

class ReviewBinding(_message.Message):
    __slots__ = ("id", "reviewer_id", "reviewed_at", "decision", "manifest_sha256", "allowed_uses", "rationale")
    ID_FIELD_NUMBER: _ClassVar[int]
    REVIEWER_ID_FIELD_NUMBER: _ClassVar[int]
    REVIEWED_AT_FIELD_NUMBER: _ClassVar[int]
    DECISION_FIELD_NUMBER: _ClassVar[int]
    MANIFEST_SHA256_FIELD_NUMBER: _ClassVar[int]
    ALLOWED_USES_FIELD_NUMBER: _ClassVar[int]
    RATIONALE_FIELD_NUMBER: _ClassVar[int]
    id: str
    reviewer_id: str
    reviewed_at: _timestamp_pb2.Timestamp
    decision: str
    manifest_sha256: str
    allowed_uses: _containers.RepeatedScalarFieldContainer[str]
    rationale: str
    def __init__(self, id: _Optional[str] = ..., reviewer_id: _Optional[str] = ..., reviewed_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., decision: _Optional[str] = ..., manifest_sha256: _Optional[str] = ..., allowed_uses: _Optional[_Iterable[str]] = ..., rationale: _Optional[str] = ...) -> None: ...

class IntakeCandidateClaims(_message.Message):
    __slots__ = ("candidate_id", "display_name", "source_label", "license_claim", "attribution", "declared_uses", "parent_candidate_id", "parent_revision", "parent_sha256")
    CANDIDATE_ID_FIELD_NUMBER: _ClassVar[int]
    DISPLAY_NAME_FIELD_NUMBER: _ClassVar[int]
    SOURCE_LABEL_FIELD_NUMBER: _ClassVar[int]
    LICENSE_CLAIM_FIELD_NUMBER: _ClassVar[int]
    ATTRIBUTION_FIELD_NUMBER: _ClassVar[int]
    DECLARED_USES_FIELD_NUMBER: _ClassVar[int]
    PARENT_CANDIDATE_ID_FIELD_NUMBER: _ClassVar[int]
    PARENT_REVISION_FIELD_NUMBER: _ClassVar[int]
    PARENT_SHA256_FIELD_NUMBER: _ClassVar[int]
    candidate_id: str
    display_name: str
    source_label: str
    license_claim: str
    attribution: str
    declared_uses: _containers.RepeatedScalarFieldContainer[str]
    parent_candidate_id: str
    parent_revision: int
    parent_sha256: str
    def __init__(self, candidate_id: _Optional[str] = ..., display_name: _Optional[str] = ..., source_label: _Optional[str] = ..., license_claim: _Optional[str] = ..., attribution: _Optional[str] = ..., declared_uses: _Optional[_Iterable[str]] = ..., parent_candidate_id: _Optional[str] = ..., parent_revision: _Optional[int] = ..., parent_sha256: _Optional[str] = ...) -> None: ...

class ListIntakeCandidatesRequest(_message.Message):
    __slots__ = ("after_candidate_id", "page_size")
    AFTER_CANDIDATE_ID_FIELD_NUMBER: _ClassVar[int]
    PAGE_SIZE_FIELD_NUMBER: _ClassVar[int]
    after_candidate_id: str
    page_size: int
    def __init__(self, after_candidate_id: _Optional[str] = ..., page_size: _Optional[int] = ...) -> None: ...

class ListIntakeCandidatesResponse(_message.Message):
    __slots__ = ("candidates", "next_after_candidate_id")
    CANDIDATES_FIELD_NUMBER: _ClassVar[int]
    NEXT_AFTER_CANDIDATE_ID_FIELD_NUMBER: _ClassVar[int]
    candidates: _containers.RepeatedCompositeFieldContainer[IntakeCandidateRevision]
    next_after_candidate_id: str
    def __init__(self, candidates: _Optional[_Iterable[_Union[IntakeCandidateRevision, _Mapping]]] = ..., next_after_candidate_id: _Optional[str] = ...) -> None: ...

class GetIntakeCandidateRequest(_message.Message):
    __slots__ = ("candidate_id", "revision")
    CANDIDATE_ID_FIELD_NUMBER: _ClassVar[int]
    REVISION_FIELD_NUMBER: _ClassVar[int]
    candidate_id: str
    revision: int
    def __init__(self, candidate_id: _Optional[str] = ..., revision: _Optional[int] = ...) -> None: ...

class GetIntakeCandidateResponse(_message.Message):
    __slots__ = ("candidate",)
    CANDIDATE_FIELD_NUMBER: _ClassVar[int]
    candidate: IntakeCandidateRevision
    def __init__(self, candidate: _Optional[_Union[IntakeCandidateRevision, _Mapping]] = ...) -> None: ...

class IntakeCandidateRevision(_message.Message):
    __slots__ = ("id", "revision", "revision_sha256", "display_name", "original_filename", "original_sha256", "size_bytes", "mime", "imported_at", "import_method", "source_label", "license_claim", "attribution", "declared_uses", "parent_candidate_id", "parent_revision", "parent_sha256", "inspection", "review_status", "render_status", "creation_lineage", "material_correction_lineage")
    ID_FIELD_NUMBER: _ClassVar[int]
    REVISION_FIELD_NUMBER: _ClassVar[int]
    REVISION_SHA256_FIELD_NUMBER: _ClassVar[int]
    DISPLAY_NAME_FIELD_NUMBER: _ClassVar[int]
    ORIGINAL_FILENAME_FIELD_NUMBER: _ClassVar[int]
    ORIGINAL_SHA256_FIELD_NUMBER: _ClassVar[int]
    SIZE_BYTES_FIELD_NUMBER: _ClassVar[int]
    MIME_FIELD_NUMBER: _ClassVar[int]
    IMPORTED_AT_FIELD_NUMBER: _ClassVar[int]
    IMPORT_METHOD_FIELD_NUMBER: _ClassVar[int]
    SOURCE_LABEL_FIELD_NUMBER: _ClassVar[int]
    LICENSE_CLAIM_FIELD_NUMBER: _ClassVar[int]
    ATTRIBUTION_FIELD_NUMBER: _ClassVar[int]
    DECLARED_USES_FIELD_NUMBER: _ClassVar[int]
    PARENT_CANDIDATE_ID_FIELD_NUMBER: _ClassVar[int]
    PARENT_REVISION_FIELD_NUMBER: _ClassVar[int]
    PARENT_SHA256_FIELD_NUMBER: _ClassVar[int]
    INSPECTION_FIELD_NUMBER: _ClassVar[int]
    REVIEW_STATUS_FIELD_NUMBER: _ClassVar[int]
    RENDER_STATUS_FIELD_NUMBER: _ClassVar[int]
    CREATION_LINEAGE_FIELD_NUMBER: _ClassVar[int]
    MATERIAL_CORRECTION_LINEAGE_FIELD_NUMBER: _ClassVar[int]
    id: str
    revision: int
    revision_sha256: str
    display_name: str
    original_filename: str
    original_sha256: str
    size_bytes: int
    mime: str
    imported_at: _timestamp_pb2.Timestamp
    import_method: str
    source_label: str
    license_claim: str
    attribution: str
    declared_uses: _containers.RepeatedScalarFieldContainer[str]
    parent_candidate_id: str
    parent_revision: int
    parent_sha256: str
    inspection: IntakeInspection
    review_status: str
    render_status: str
    creation_lineage: CreationLineage
    material_correction_lineage: MaterialCorrectionLineage
    def __init__(self, id: _Optional[str] = ..., revision: _Optional[int] = ..., revision_sha256: _Optional[str] = ..., display_name: _Optional[str] = ..., original_filename: _Optional[str] = ..., original_sha256: _Optional[str] = ..., size_bytes: _Optional[int] = ..., mime: _Optional[str] = ..., imported_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., import_method: _Optional[str] = ..., source_label: _Optional[str] = ..., license_claim: _Optional[str] = ..., attribution: _Optional[str] = ..., declared_uses: _Optional[_Iterable[str]] = ..., parent_candidate_id: _Optional[str] = ..., parent_revision: _Optional[int] = ..., parent_sha256: _Optional[str] = ..., inspection: _Optional[_Union[IntakeInspection, _Mapping]] = ..., review_status: _Optional[str] = ..., render_status: _Optional[str] = ..., creation_lineage: _Optional[_Union[CreationLineage, _Mapping]] = ..., material_correction_lineage: _Optional[_Union[MaterialCorrectionLineage, _Mapping]] = ...) -> None: ...

class CreationLineage(_message.Message):
    __slots__ = ("job_id", "recipe_sha256", "reference_sha256", "provider", "model", "provider_calls_made", "input_tokens", "output_tokens", "cost_micros", "input_tokens_provenance", "output_tokens_provenance", "cost_provenance", "applied_settings", "inference_outcome_sha256", "correction_source", "credits_charged", "accounting_status")
    JOB_ID_FIELD_NUMBER: _ClassVar[int]
    RECIPE_SHA256_FIELD_NUMBER: _ClassVar[int]
    REFERENCE_SHA256_FIELD_NUMBER: _ClassVar[int]
    PROVIDER_FIELD_NUMBER: _ClassVar[int]
    MODEL_FIELD_NUMBER: _ClassVar[int]
    PROVIDER_CALLS_MADE_FIELD_NUMBER: _ClassVar[int]
    INPUT_TOKENS_FIELD_NUMBER: _ClassVar[int]
    OUTPUT_TOKENS_FIELD_NUMBER: _ClassVar[int]
    COST_MICROS_FIELD_NUMBER: _ClassVar[int]
    INPUT_TOKENS_PROVENANCE_FIELD_NUMBER: _ClassVar[int]
    OUTPUT_TOKENS_PROVENANCE_FIELD_NUMBER: _ClassVar[int]
    COST_PROVENANCE_FIELD_NUMBER: _ClassVar[int]
    APPLIED_SETTINGS_FIELD_NUMBER: _ClassVar[int]
    INFERENCE_OUTCOME_SHA256_FIELD_NUMBER: _ClassVar[int]
    CORRECTION_SOURCE_FIELD_NUMBER: _ClassVar[int]
    CREDITS_CHARGED_FIELD_NUMBER: _ClassVar[int]
    ACCOUNTING_STATUS_FIELD_NUMBER: _ClassVar[int]
    job_id: str
    recipe_sha256: str
    reference_sha256: _containers.RepeatedScalarFieldContainer[str]
    provider: str
    model: str
    provider_calls_made: int
    input_tokens: int
    output_tokens: int
    cost_micros: int
    input_tokens_provenance: _inference_pb2.UsageProvenance
    output_tokens_provenance: _inference_pb2.UsageProvenance
    cost_provenance: _inference_pb2.UsageProvenance
    applied_settings: _gateway_pb2.AppliedSettings
    inference_outcome_sha256: str
    correction_source: CreationCorrectionSource
    credits_charged: int
    accounting_status: str
    def __init__(self, job_id: _Optional[str] = ..., recipe_sha256: _Optional[str] = ..., reference_sha256: _Optional[_Iterable[str]] = ..., provider: _Optional[str] = ..., model: _Optional[str] = ..., provider_calls_made: _Optional[int] = ..., input_tokens: _Optional[int] = ..., output_tokens: _Optional[int] = ..., cost_micros: _Optional[int] = ..., input_tokens_provenance: _Optional[_Union[_inference_pb2.UsageProvenance, str]] = ..., output_tokens_provenance: _Optional[_Union[_inference_pb2.UsageProvenance, str]] = ..., cost_provenance: _Optional[_Union[_inference_pb2.UsageProvenance, str]] = ..., applied_settings: _Optional[_Union[_gateway_pb2.AppliedSettings, _Mapping]] = ..., inference_outcome_sha256: _Optional[str] = ..., correction_source: _Optional[_Union[CreationCorrectionSource, _Mapping]] = ..., credits_charged: _Optional[int] = ..., accounting_status: _Optional[str] = ...) -> None: ...

class CreationCorrectionSource(_message.Message):
    __slots__ = ("job_id", "input_fingerprint", "brief_sha256", "recipe_sha256", "issue_code", "part_id", "axis", "mode", "feedback")
    JOB_ID_FIELD_NUMBER: _ClassVar[int]
    INPUT_FINGERPRINT_FIELD_NUMBER: _ClassVar[int]
    BRIEF_SHA256_FIELD_NUMBER: _ClassVar[int]
    RECIPE_SHA256_FIELD_NUMBER: _ClassVar[int]
    ISSUE_CODE_FIELD_NUMBER: _ClassVar[int]
    PART_ID_FIELD_NUMBER: _ClassVar[int]
    AXIS_FIELD_NUMBER: _ClassVar[int]
    MODE_FIELD_NUMBER: _ClassVar[int]
    FEEDBACK_FIELD_NUMBER: _ClassVar[int]
    job_id: str
    input_fingerprint: str
    brief_sha256: str
    recipe_sha256: str
    issue_code: str
    part_id: str
    axis: str
    mode: str
    feedback: str
    def __init__(self, job_id: _Optional[str] = ..., input_fingerprint: _Optional[str] = ..., brief_sha256: _Optional[str] = ..., recipe_sha256: _Optional[str] = ..., issue_code: _Optional[str] = ..., part_id: _Optional[str] = ..., axis: _Optional[str] = ..., mode: _Optional[str] = ..., feedback: _Optional[str] = ...) -> None: ...

class MaterialCorrectionLineage(_message.Message):
    __slots__ = ("parent_candidate_id", "parent_revision", "parent_sha256", "request_sha256", "material_index", "material_name", "before", "after", "before_glb_sha256", "after_glb_sha256", "before_bin_chunks_sha256", "after_bin_chunks_sha256", "geometry_unchanged")
    PARENT_CANDIDATE_ID_FIELD_NUMBER: _ClassVar[int]
    PARENT_REVISION_FIELD_NUMBER: _ClassVar[int]
    PARENT_SHA256_FIELD_NUMBER: _ClassVar[int]
    REQUEST_SHA256_FIELD_NUMBER: _ClassVar[int]
    MATERIAL_INDEX_FIELD_NUMBER: _ClassVar[int]
    MATERIAL_NAME_FIELD_NUMBER: _ClassVar[int]
    BEFORE_FIELD_NUMBER: _ClassVar[int]
    AFTER_FIELD_NUMBER: _ClassVar[int]
    BEFORE_GLB_SHA256_FIELD_NUMBER: _ClassVar[int]
    AFTER_GLB_SHA256_FIELD_NUMBER: _ClassVar[int]
    BEFORE_BIN_CHUNKS_SHA256_FIELD_NUMBER: _ClassVar[int]
    AFTER_BIN_CHUNKS_SHA256_FIELD_NUMBER: _ClassVar[int]
    GEOMETRY_UNCHANGED_FIELD_NUMBER: _ClassVar[int]
    parent_candidate_id: str
    parent_revision: int
    parent_sha256: str
    request_sha256: str
    material_index: int
    material_name: str
    before: _struct_pb2.Struct
    after: _struct_pb2.Struct
    before_glb_sha256: str
    after_glb_sha256: str
    before_bin_chunks_sha256: _containers.RepeatedScalarFieldContainer[str]
    after_bin_chunks_sha256: _containers.RepeatedScalarFieldContainer[str]
    geometry_unchanged: bool
    def __init__(self, parent_candidate_id: _Optional[str] = ..., parent_revision: _Optional[int] = ..., parent_sha256: _Optional[str] = ..., request_sha256: _Optional[str] = ..., material_index: _Optional[int] = ..., material_name: _Optional[str] = ..., before: _Optional[_Union[_struct_pb2.Struct, _Mapping]] = ..., after: _Optional[_Union[_struct_pb2.Struct, _Mapping]] = ..., before_glb_sha256: _Optional[str] = ..., after_glb_sha256: _Optional[str] = ..., before_bin_chunks_sha256: _Optional[_Iterable[str]] = ..., after_bin_chunks_sha256: _Optional[_Iterable[str]] = ..., geometry_unchanged: _Optional[bool] = ...) -> None: ...

class IntakeInspection(_message.Message):
    __slots__ = ("profile", "profile_version", "supported", "node_count", "mesh_count", "triangle_count", "finding")
    PROFILE_FIELD_NUMBER: _ClassVar[int]
    PROFILE_VERSION_FIELD_NUMBER: _ClassVar[int]
    SUPPORTED_FIELD_NUMBER: _ClassVar[int]
    NODE_COUNT_FIELD_NUMBER: _ClassVar[int]
    MESH_COUNT_FIELD_NUMBER: _ClassVar[int]
    TRIANGLE_COUNT_FIELD_NUMBER: _ClassVar[int]
    FINDING_FIELD_NUMBER: _ClassVar[int]
    profile: str
    profile_version: int
    supported: bool
    node_count: int
    mesh_count: int
    triangle_count: int
    finding: str
    def __init__(self, profile: _Optional[str] = ..., profile_version: _Optional[int] = ..., supported: _Optional[bool] = ..., node_count: _Optional[int] = ..., mesh_count: _Optional[int] = ..., triangle_count: _Optional[int] = ..., finding: _Optional[str] = ...) -> None: ...
