import datetime

from google.protobuf import timestamp_pb2 as _timestamp_pb2
from google.protobuf.internal import containers as _containers
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Iterable as _Iterable, Mapping as _Mapping
from typing import ClassVar as _ClassVar, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class AnalyzeClosureRequest(_message.Message):
    __slots__ = ("scenario", "source_root", "source_digest", "recipe_digest", "recipe", "snapshot")
    SCENARIO_FIELD_NUMBER: _ClassVar[int]
    SOURCE_ROOT_FIELD_NUMBER: _ClassVar[int]
    SOURCE_DIGEST_FIELD_NUMBER: _ClassVar[int]
    RECIPE_DIGEST_FIELD_NUMBER: _ClassVar[int]
    RECIPE_FIELD_NUMBER: _ClassVar[int]
    SNAPSHOT_FIELD_NUMBER: _ClassVar[int]
    scenario: str
    source_root: str
    source_digest: str
    recipe_digest: str
    recipe: ExportRecipe
    snapshot: SourceSnapshot
    def __init__(self, scenario: _Optional[str] = ..., source_root: _Optional[str] = ..., source_digest: _Optional[str] = ..., recipe_digest: _Optional[str] = ..., recipe: _Optional[_Union[ExportRecipe, _Mapping]] = ..., snapshot: _Optional[_Union[SourceSnapshot, _Mapping]] = ...) -> None: ...

class ExportRecipe(_message.Message):
    __slots__ = ("schema_version", "scenario", "mode", "closure_ref", "publish_policy_ref", "runtime_profile_ref", "archive_format", "deterministic", "recipe_digest")
    SCHEMA_VERSION_FIELD_NUMBER: _ClassVar[int]
    SCENARIO_FIELD_NUMBER: _ClassVar[int]
    MODE_FIELD_NUMBER: _ClassVar[int]
    CLOSURE_REF_FIELD_NUMBER: _ClassVar[int]
    PUBLISH_POLICY_REF_FIELD_NUMBER: _ClassVar[int]
    RUNTIME_PROFILE_REF_FIELD_NUMBER: _ClassVar[int]
    ARCHIVE_FORMAT_FIELD_NUMBER: _ClassVar[int]
    DETERMINISTIC_FIELD_NUMBER: _ClassVar[int]
    RECIPE_DIGEST_FIELD_NUMBER: _ClassVar[int]
    schema_version: int
    scenario: str
    mode: str
    closure_ref: str
    publish_policy_ref: str
    runtime_profile_ref: str
    archive_format: str
    deterministic: bool
    recipe_digest: str
    def __init__(self, schema_version: _Optional[int] = ..., scenario: _Optional[str] = ..., mode: _Optional[str] = ..., closure_ref: _Optional[str] = ..., publish_policy_ref: _Optional[str] = ..., runtime_profile_ref: _Optional[str] = ..., archive_format: _Optional[str] = ..., deterministic: _Optional[bool] = ..., recipe_digest: _Optional[str] = ...) -> None: ...

class SourceSnapshot(_message.Message):
    __slots__ = ("scenario", "source_digest", "closure_digest", "captured_at", "standing")
    SCENARIO_FIELD_NUMBER: _ClassVar[int]
    SOURCE_DIGEST_FIELD_NUMBER: _ClassVar[int]
    CLOSURE_DIGEST_FIELD_NUMBER: _ClassVar[int]
    CAPTURED_AT_FIELD_NUMBER: _ClassVar[int]
    STANDING_FIELD_NUMBER: _ClassVar[int]
    scenario: str
    source_digest: str
    closure_digest: str
    captured_at: _timestamp_pb2.Timestamp
    standing: str
    def __init__(self, scenario: _Optional[str] = ..., source_digest: _Optional[str] = ..., closure_digest: _Optional[str] = ..., captured_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., standing: _Optional[str] = ...) -> None: ...

class SourceFile(_message.Message):
    __slots__ = ("source_path", "export_path", "sha256", "mode", "reason_refs")
    SOURCE_PATH_FIELD_NUMBER: _ClassVar[int]
    EXPORT_PATH_FIELD_NUMBER: _ClassVar[int]
    SHA256_FIELD_NUMBER: _ClassVar[int]
    MODE_FIELD_NUMBER: _ClassVar[int]
    REASON_REFS_FIELD_NUMBER: _ClassVar[int]
    source_path: str
    export_path: str
    sha256: str
    mode: int
    reason_refs: _containers.RepeatedScalarFieldContainer[str]
    def __init__(self, source_path: _Optional[str] = ..., export_path: _Optional[str] = ..., sha256: _Optional[str] = ..., mode: _Optional[int] = ..., reason_refs: _Optional[_Iterable[str]] = ...) -> None: ...

class ClosureNode(_message.Message):
    __slots__ = ("id", "kind")
    ID_FIELD_NUMBER: _ClassVar[int]
    KIND_FIELD_NUMBER: _ClassVar[int]
    id: str
    kind: str
    def __init__(self, id: _Optional[str] = ..., kind: _Optional[str] = ...) -> None: ...

class RewriteProposal(_message.Message):
    __slots__ = ("kind", "proposal_ref")
    KIND_FIELD_NUMBER: _ClassVar[int]
    PROPOSAL_REF_FIELD_NUMBER: _ClassVar[int]
    kind: str
    proposal_ref: str
    def __init__(self, kind: _Optional[str] = ..., proposal_ref: _Optional[str] = ...) -> None: ...

class SourceClosure(_message.Message):
    __slots__ = ("source_digest", "scenario", "nodes", "files", "rewrites", "unresolved", "closure_digest")
    SOURCE_DIGEST_FIELD_NUMBER: _ClassVar[int]
    SCENARIO_FIELD_NUMBER: _ClassVar[int]
    NODES_FIELD_NUMBER: _ClassVar[int]
    FILES_FIELD_NUMBER: _ClassVar[int]
    REWRITES_FIELD_NUMBER: _ClassVar[int]
    UNRESOLVED_FIELD_NUMBER: _ClassVar[int]
    CLOSURE_DIGEST_FIELD_NUMBER: _ClassVar[int]
    source_digest: str
    scenario: str
    nodes: _containers.RepeatedCompositeFieldContainer[ClosureNode]
    files: _containers.RepeatedCompositeFieldContainer[SourceFile]
    rewrites: _containers.RepeatedCompositeFieldContainer[RewriteProposal]
    unresolved: _containers.RepeatedScalarFieldContainer[str]
    closure_digest: str
    def __init__(self, source_digest: _Optional[str] = ..., scenario: _Optional[str] = ..., nodes: _Optional[_Iterable[_Union[ClosureNode, _Mapping]]] = ..., files: _Optional[_Iterable[_Union[SourceFile, _Mapping]]] = ..., rewrites: _Optional[_Iterable[_Union[RewriteProposal, _Mapping]]] = ..., unresolved: _Optional[_Iterable[str]] = ..., closure_digest: _Optional[str] = ...) -> None: ...

class ClosureResponse(_message.Message):
    __slots__ = ("closure",)
    CLOSURE_FIELD_NUMBER: _ClassVar[int]
    closure: SourceClosure
    def __init__(self, closure: _Optional[_Union[SourceClosure, _Mapping]] = ...) -> None: ...

class AssembleExportRequest(_message.Message):
    __slots__ = ("scenario", "source_root", "source_digest", "recipe_yaml", "output_path")
    SCENARIO_FIELD_NUMBER: _ClassVar[int]
    SOURCE_ROOT_FIELD_NUMBER: _ClassVar[int]
    SOURCE_DIGEST_FIELD_NUMBER: _ClassVar[int]
    RECIPE_YAML_FIELD_NUMBER: _ClassVar[int]
    OUTPUT_PATH_FIELD_NUMBER: _ClassVar[int]
    scenario: str
    source_root: str
    source_digest: str
    recipe_yaml: str
    output_path: str
    def __init__(self, scenario: _Optional[str] = ..., source_root: _Optional[str] = ..., source_digest: _Optional[str] = ..., recipe_yaml: _Optional[str] = ..., output_path: _Optional[str] = ...) -> None: ...

class ArtifactManifestEntry(_message.Message):
    __slots__ = ("path", "sha256", "mode", "source_path", "size_bytes")
    PATH_FIELD_NUMBER: _ClassVar[int]
    SHA256_FIELD_NUMBER: _ClassVar[int]
    MODE_FIELD_NUMBER: _ClassVar[int]
    SOURCE_PATH_FIELD_NUMBER: _ClassVar[int]
    SIZE_BYTES_FIELD_NUMBER: _ClassVar[int]
    path: str
    sha256: str
    mode: int
    source_path: str
    size_bytes: int
    def __init__(self, path: _Optional[str] = ..., sha256: _Optional[str] = ..., mode: _Optional[int] = ..., source_path: _Optional[str] = ..., size_bytes: _Optional[int] = ...) -> None: ...

class Artifact(_message.Message):
    __slots__ = ("artifact_id", "source_digest", "recipe_digest", "closure_digest", "files", "manifest_digest", "archive_digest", "archive_path", "status")
    ARTIFACT_ID_FIELD_NUMBER: _ClassVar[int]
    SOURCE_DIGEST_FIELD_NUMBER: _ClassVar[int]
    RECIPE_DIGEST_FIELD_NUMBER: _ClassVar[int]
    CLOSURE_DIGEST_FIELD_NUMBER: _ClassVar[int]
    FILES_FIELD_NUMBER: _ClassVar[int]
    MANIFEST_DIGEST_FIELD_NUMBER: _ClassVar[int]
    ARCHIVE_DIGEST_FIELD_NUMBER: _ClassVar[int]
    ARCHIVE_PATH_FIELD_NUMBER: _ClassVar[int]
    STATUS_FIELD_NUMBER: _ClassVar[int]
    artifact_id: str
    source_digest: str
    recipe_digest: str
    closure_digest: str
    files: _containers.RepeatedCompositeFieldContainer[ArtifactManifestEntry]
    manifest_digest: str
    archive_digest: str
    archive_path: str
    status: str
    def __init__(self, artifact_id: _Optional[str] = ..., source_digest: _Optional[str] = ..., recipe_digest: _Optional[str] = ..., closure_digest: _Optional[str] = ..., files: _Optional[_Iterable[_Union[ArtifactManifestEntry, _Mapping]]] = ..., manifest_digest: _Optional[str] = ..., archive_digest: _Optional[str] = ..., archive_path: _Optional[str] = ..., status: _Optional[str] = ...) -> None: ...

class ArtifactResponse(_message.Message):
    __slots__ = ("artifact",)
    ARTIFACT_FIELD_NUMBER: _ClassVar[int]
    artifact: Artifact
    def __init__(self, artifact: _Optional[_Union[Artifact, _Mapping]] = ...) -> None: ...

class VerifyExportRequest(_message.Message):
    __slots__ = ("artifact_id", "archive_path", "expected_archive_digest", "verification_environment")
    ARTIFACT_ID_FIELD_NUMBER: _ClassVar[int]
    ARCHIVE_PATH_FIELD_NUMBER: _ClassVar[int]
    EXPECTED_ARCHIVE_DIGEST_FIELD_NUMBER: _ClassVar[int]
    VERIFICATION_ENVIRONMENT_FIELD_NUMBER: _ClassVar[int]
    artifact_id: str
    archive_path: str
    expected_archive_digest: str
    verification_environment: str
    def __init__(self, artifact_id: _Optional[str] = ..., archive_path: _Optional[str] = ..., expected_archive_digest: _Optional[str] = ..., verification_environment: _Optional[str] = ...) -> None: ...

class VerificationResponse(_message.Message):
    __slots__ = ("verification_id", "artifact_id", "status", "gates_passed", "failures", "receipt_digest", "verified_at")
    VERIFICATION_ID_FIELD_NUMBER: _ClassVar[int]
    ARTIFACT_ID_FIELD_NUMBER: _ClassVar[int]
    STATUS_FIELD_NUMBER: _ClassVar[int]
    GATES_PASSED_FIELD_NUMBER: _ClassVar[int]
    FAILURES_FIELD_NUMBER: _ClassVar[int]
    RECEIPT_DIGEST_FIELD_NUMBER: _ClassVar[int]
    VERIFIED_AT_FIELD_NUMBER: _ClassVar[int]
    verification_id: str
    artifact_id: str
    status: str
    gates_passed: _containers.RepeatedScalarFieldContainer[str]
    failures: _containers.RepeatedScalarFieldContainer[str]
    receipt_digest: str
    verified_at: _timestamp_pb2.Timestamp
    def __init__(self, verification_id: _Optional[str] = ..., artifact_id: _Optional[str] = ..., status: _Optional[str] = ..., gates_passed: _Optional[_Iterable[str]] = ..., failures: _Optional[_Iterable[str]] = ..., receipt_digest: _Optional[str] = ..., verified_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ...) -> None: ...

class PreparePublicationRequest(_message.Message):
    __slots__ = ("distribution_id", "artifact_id", "destination_kind", "destination_reference", "expected_revision")
    DISTRIBUTION_ID_FIELD_NUMBER: _ClassVar[int]
    ARTIFACT_ID_FIELD_NUMBER: _ClassVar[int]
    DESTINATION_KIND_FIELD_NUMBER: _ClassVar[int]
    DESTINATION_REFERENCE_FIELD_NUMBER: _ClassVar[int]
    EXPECTED_REVISION_FIELD_NUMBER: _ClassVar[int]
    distribution_id: str
    artifact_id: str
    destination_kind: str
    destination_reference: str
    expected_revision: str
    def __init__(self, distribution_id: _Optional[str] = ..., artifact_id: _Optional[str] = ..., destination_kind: _Optional[str] = ..., destination_reference: _Optional[str] = ..., expected_revision: _Optional[str] = ...) -> None: ...

class PublicationPreviewResponse(_message.Message):
    __slots__ = ("distribution_id", "artifact_id", "status", "human_action", "readback_oracle", "preconditions")
    DISTRIBUTION_ID_FIELD_NUMBER: _ClassVar[int]
    ARTIFACT_ID_FIELD_NUMBER: _ClassVar[int]
    STATUS_FIELD_NUMBER: _ClassVar[int]
    HUMAN_ACTION_FIELD_NUMBER: _ClassVar[int]
    READBACK_ORACLE_FIELD_NUMBER: _ClassVar[int]
    PRECONDITIONS_FIELD_NUMBER: _ClassVar[int]
    distribution_id: str
    artifact_id: str
    status: str
    human_action: str
    readback_oracle: str
    preconditions: _containers.RepeatedScalarFieldContainer[str]
    def __init__(self, distribution_id: _Optional[str] = ..., artifact_id: _Optional[str] = ..., status: _Optional[str] = ..., human_action: _Optional[str] = ..., readback_oracle: _Optional[str] = ..., preconditions: _Optional[_Iterable[str]] = ...) -> None: ...

class ListDistributionsRequest(_message.Message):
    __slots__ = ("scenario", "repository_context", "limit")
    SCENARIO_FIELD_NUMBER: _ClassVar[int]
    REPOSITORY_CONTEXT_FIELD_NUMBER: _ClassVar[int]
    LIMIT_FIELD_NUMBER: _ClassVar[int]
    scenario: str
    repository_context: str
    limit: int
    def __init__(self, scenario: _Optional[str] = ..., repository_context: _Optional[str] = ..., limit: _Optional[int] = ...) -> None: ...

class ListDistributionsResponse(_message.Message):
    __slots__ = ("distributions", "source_of_truth", "observed_at", "freshness", "unavailable_reason")
    DISTRIBUTIONS_FIELD_NUMBER: _ClassVar[int]
    SOURCE_OF_TRUTH_FIELD_NUMBER: _ClassVar[int]
    OBSERVED_AT_FIELD_NUMBER: _ClassVar[int]
    FRESHNESS_FIELD_NUMBER: _ClassVar[int]
    UNAVAILABLE_REASON_FIELD_NUMBER: _ClassVar[int]
    distributions: _containers.RepeatedCompositeFieldContainer[Distribution]
    source_of_truth: str
    observed_at: _timestamp_pb2.Timestamp
    freshness: str
    unavailable_reason: str
    def __init__(self, distributions: _Optional[_Iterable[_Union[Distribution, _Mapping]]] = ..., source_of_truth: _Optional[str] = ..., observed_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., freshness: _Optional[str] = ..., unavailable_reason: _Optional[str] = ...) -> None: ...

class GetDistributionRequest(_message.Message):
    __slots__ = ("distribution_id",)
    DISTRIBUTION_ID_FIELD_NUMBER: _ClassVar[int]
    distribution_id: str
    def __init__(self, distribution_id: _Optional[str] = ...) -> None: ...

class Distribution(_message.Message):
    __slots__ = ("distribution_id", "scenario", "source_digest", "recipe_digest", "artifact_id", "verification_status", "publication_status", "destination_kind", "destination_reference", "last_verified_revision", "updated_at", "closure_digest", "policy_digest", "artifact_digest", "verification_receipt", "deployment_manager_decision", "source_of_truth", "source_timestamp", "freshness", "drift_state", "destination_revision", "readback_receipt", "workflow_url")
    DISTRIBUTION_ID_FIELD_NUMBER: _ClassVar[int]
    SCENARIO_FIELD_NUMBER: _ClassVar[int]
    SOURCE_DIGEST_FIELD_NUMBER: _ClassVar[int]
    RECIPE_DIGEST_FIELD_NUMBER: _ClassVar[int]
    ARTIFACT_ID_FIELD_NUMBER: _ClassVar[int]
    VERIFICATION_STATUS_FIELD_NUMBER: _ClassVar[int]
    PUBLICATION_STATUS_FIELD_NUMBER: _ClassVar[int]
    DESTINATION_KIND_FIELD_NUMBER: _ClassVar[int]
    DESTINATION_REFERENCE_FIELD_NUMBER: _ClassVar[int]
    LAST_VERIFIED_REVISION_FIELD_NUMBER: _ClassVar[int]
    UPDATED_AT_FIELD_NUMBER: _ClassVar[int]
    CLOSURE_DIGEST_FIELD_NUMBER: _ClassVar[int]
    POLICY_DIGEST_FIELD_NUMBER: _ClassVar[int]
    ARTIFACT_DIGEST_FIELD_NUMBER: _ClassVar[int]
    VERIFICATION_RECEIPT_FIELD_NUMBER: _ClassVar[int]
    DEPLOYMENT_MANAGER_DECISION_FIELD_NUMBER: _ClassVar[int]
    SOURCE_OF_TRUTH_FIELD_NUMBER: _ClassVar[int]
    SOURCE_TIMESTAMP_FIELD_NUMBER: _ClassVar[int]
    FRESHNESS_FIELD_NUMBER: _ClassVar[int]
    DRIFT_STATE_FIELD_NUMBER: _ClassVar[int]
    DESTINATION_REVISION_FIELD_NUMBER: _ClassVar[int]
    READBACK_RECEIPT_FIELD_NUMBER: _ClassVar[int]
    WORKFLOW_URL_FIELD_NUMBER: _ClassVar[int]
    distribution_id: str
    scenario: str
    source_digest: str
    recipe_digest: str
    artifact_id: str
    verification_status: str
    publication_status: str
    destination_kind: str
    destination_reference: str
    last_verified_revision: str
    updated_at: _timestamp_pb2.Timestamp
    closure_digest: str
    policy_digest: str
    artifact_digest: str
    verification_receipt: str
    deployment_manager_decision: str
    source_of_truth: str
    source_timestamp: _timestamp_pb2.Timestamp
    freshness: str
    drift_state: str
    destination_revision: str
    readback_receipt: str
    workflow_url: str
    def __init__(self, distribution_id: _Optional[str] = ..., scenario: _Optional[str] = ..., source_digest: _Optional[str] = ..., recipe_digest: _Optional[str] = ..., artifact_id: _Optional[str] = ..., verification_status: _Optional[str] = ..., publication_status: _Optional[str] = ..., destination_kind: _Optional[str] = ..., destination_reference: _Optional[str] = ..., last_verified_revision: _Optional[str] = ..., updated_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., closure_digest: _Optional[str] = ..., policy_digest: _Optional[str] = ..., artifact_digest: _Optional[str] = ..., verification_receipt: _Optional[str] = ..., deployment_manager_decision: _Optional[str] = ..., source_of_truth: _Optional[str] = ..., source_timestamp: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., freshness: _Optional[str] = ..., drift_state: _Optional[str] = ..., destination_revision: _Optional[str] = ..., readback_receipt: _Optional[str] = ..., workflow_url: _Optional[str] = ...) -> None: ...

class DistributionResponse(_message.Message):
    __slots__ = ("distribution",)
    DISTRIBUTION_FIELD_NUMBER: _ClassVar[int]
    distribution: Distribution
    def __init__(self, distribution: _Optional[_Union[Distribution, _Mapping]] = ...) -> None: ...

class GetDistributionContentsRequest(_message.Message):
    __slots__ = ("distribution_id",)
    DISTRIBUTION_ID_FIELD_NUMBER: _ClassVar[int]
    distribution_id: str
    def __init__(self, distribution_id: _Optional[str] = ...) -> None: ...

class DistributionContent(_message.Message):
    __slots__ = ("path", "source_path", "category", "digest", "size_bytes")
    PATH_FIELD_NUMBER: _ClassVar[int]
    SOURCE_PATH_FIELD_NUMBER: _ClassVar[int]
    CATEGORY_FIELD_NUMBER: _ClassVar[int]
    DIGEST_FIELD_NUMBER: _ClassVar[int]
    SIZE_BYTES_FIELD_NUMBER: _ClassVar[int]
    path: str
    source_path: str
    category: str
    digest: str
    size_bytes: int
    def __init__(self, path: _Optional[str] = ..., source_path: _Optional[str] = ..., category: _Optional[str] = ..., digest: _Optional[str] = ..., size_bytes: _Optional[int] = ...) -> None: ...

class DistributionExclusion(_message.Message):
    __slots__ = ("path", "category", "safe_reason")
    PATH_FIELD_NUMBER: _ClassVar[int]
    CATEGORY_FIELD_NUMBER: _ClassVar[int]
    SAFE_REASON_FIELD_NUMBER: _ClassVar[int]
    path: str
    category: str
    safe_reason: str
    def __init__(self, path: _Optional[str] = ..., category: _Optional[str] = ..., safe_reason: _Optional[str] = ...) -> None: ...

class DistributionContentsResponse(_message.Message):
    __slots__ = ("distribution_id", "contents", "exclusions", "unresolved_obligations", "runtime_requirements", "source_of_truth", "observed_at", "freshness")
    DISTRIBUTION_ID_FIELD_NUMBER: _ClassVar[int]
    CONTENTS_FIELD_NUMBER: _ClassVar[int]
    EXCLUSIONS_FIELD_NUMBER: _ClassVar[int]
    UNRESOLVED_OBLIGATIONS_FIELD_NUMBER: _ClassVar[int]
    RUNTIME_REQUIREMENTS_FIELD_NUMBER: _ClassVar[int]
    SOURCE_OF_TRUTH_FIELD_NUMBER: _ClassVar[int]
    OBSERVED_AT_FIELD_NUMBER: _ClassVar[int]
    FRESHNESS_FIELD_NUMBER: _ClassVar[int]
    distribution_id: str
    contents: _containers.RepeatedCompositeFieldContainer[DistributionContent]
    exclusions: _containers.RepeatedCompositeFieldContainer[DistributionExclusion]
    unresolved_obligations: _containers.RepeatedScalarFieldContainer[str]
    runtime_requirements: _containers.RepeatedScalarFieldContainer[str]
    source_of_truth: str
    observed_at: _timestamp_pb2.Timestamp
    freshness: str
    def __init__(self, distribution_id: _Optional[str] = ..., contents: _Optional[_Iterable[_Union[DistributionContent, _Mapping]]] = ..., exclusions: _Optional[_Iterable[_Union[DistributionExclusion, _Mapping]]] = ..., unresolved_obligations: _Optional[_Iterable[str]] = ..., runtime_requirements: _Optional[_Iterable[str]] = ..., source_of_truth: _Optional[str] = ..., observed_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., freshness: _Optional[str] = ...) -> None: ...

class GetPublicationHandoffRequest(_message.Message):
    __slots__ = ("distribution_id",)
    DISTRIBUTION_ID_FIELD_NUMBER: _ClassVar[int]
    distribution_id: str
    def __init__(self, distribution_id: _Optional[str] = ...) -> None: ...

class PublicationHandoffResponse(_message.Message):
    __slots__ = ("distribution_id", "artifact_id", "artifact_digest", "destination", "status", "human_action", "readback_oracle", "preconditions", "approval_reference", "source_of_truth", "observed_at", "freshness")
    DISTRIBUTION_ID_FIELD_NUMBER: _ClassVar[int]
    ARTIFACT_ID_FIELD_NUMBER: _ClassVar[int]
    ARTIFACT_DIGEST_FIELD_NUMBER: _ClassVar[int]
    DESTINATION_FIELD_NUMBER: _ClassVar[int]
    STATUS_FIELD_NUMBER: _ClassVar[int]
    HUMAN_ACTION_FIELD_NUMBER: _ClassVar[int]
    READBACK_ORACLE_FIELD_NUMBER: _ClassVar[int]
    PRECONDITIONS_FIELD_NUMBER: _ClassVar[int]
    APPROVAL_REFERENCE_FIELD_NUMBER: _ClassVar[int]
    SOURCE_OF_TRUTH_FIELD_NUMBER: _ClassVar[int]
    OBSERVED_AT_FIELD_NUMBER: _ClassVar[int]
    FRESHNESS_FIELD_NUMBER: _ClassVar[int]
    distribution_id: str
    artifact_id: str
    artifact_digest: str
    destination: str
    status: str
    human_action: str
    readback_oracle: str
    preconditions: _containers.RepeatedScalarFieldContainer[str]
    approval_reference: str
    source_of_truth: str
    observed_at: _timestamp_pb2.Timestamp
    freshness: str
    def __init__(self, distribution_id: _Optional[str] = ..., artifact_id: _Optional[str] = ..., artifact_digest: _Optional[str] = ..., destination: _Optional[str] = ..., status: _Optional[str] = ..., human_action: _Optional[str] = ..., readback_oracle: _Optional[str] = ..., preconditions: _Optional[_Iterable[str]] = ..., approval_reference: _Optional[str] = ..., source_of_truth: _Optional[str] = ..., observed_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., freshness: _Optional[str] = ...) -> None: ...

class GetDistributionDriftRequest(_message.Message):
    __slots__ = ("distribution_id",)
    DISTRIBUTION_ID_FIELD_NUMBER: _ClassVar[int]
    distribution_id: str
    def __init__(self, distribution_id: _Optional[str] = ...) -> None: ...

class DistributionDriftResponse(_message.Message):
    __slots__ = ("distribution_id", "state", "source_changed", "destination_changed", "current_source_digest", "recorded_source_digest", "current_artifact_digest", "recorded_artifact_digest", "actions", "source_of_truth", "observed_at", "freshness")
    DISTRIBUTION_ID_FIELD_NUMBER: _ClassVar[int]
    STATE_FIELD_NUMBER: _ClassVar[int]
    SOURCE_CHANGED_FIELD_NUMBER: _ClassVar[int]
    DESTINATION_CHANGED_FIELD_NUMBER: _ClassVar[int]
    CURRENT_SOURCE_DIGEST_FIELD_NUMBER: _ClassVar[int]
    RECORDED_SOURCE_DIGEST_FIELD_NUMBER: _ClassVar[int]
    CURRENT_ARTIFACT_DIGEST_FIELD_NUMBER: _ClassVar[int]
    RECORDED_ARTIFACT_DIGEST_FIELD_NUMBER: _ClassVar[int]
    ACTIONS_FIELD_NUMBER: _ClassVar[int]
    SOURCE_OF_TRUTH_FIELD_NUMBER: _ClassVar[int]
    OBSERVED_AT_FIELD_NUMBER: _ClassVar[int]
    FRESHNESS_FIELD_NUMBER: _ClassVar[int]
    distribution_id: str
    state: str
    source_changed: bool
    destination_changed: bool
    current_source_digest: str
    recorded_source_digest: str
    current_artifact_digest: str
    recorded_artifact_digest: str
    actions: _containers.RepeatedScalarFieldContainer[str]
    source_of_truth: str
    observed_at: _timestamp_pb2.Timestamp
    freshness: str
    def __init__(self, distribution_id: _Optional[str] = ..., state: _Optional[str] = ..., source_changed: _Optional[bool] = ..., destination_changed: _Optional[bool] = ..., current_source_digest: _Optional[str] = ..., recorded_source_digest: _Optional[str] = ..., current_artifact_digest: _Optional[str] = ..., recorded_artifact_digest: _Optional[str] = ..., actions: _Optional[_Iterable[str]] = ..., source_of_truth: _Optional[str] = ..., observed_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., freshness: _Optional[str] = ...) -> None: ...
