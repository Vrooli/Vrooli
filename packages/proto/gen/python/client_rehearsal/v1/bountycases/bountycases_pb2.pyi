from google.protobuf.internal import containers as _containers
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Iterable as _Iterable, Mapping as _Mapping
from typing import ClassVar as _ClassVar, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class ResolveBountyReplayCaseRequest(_message.Message):
    __slots__ = ("case_id", "public_url")
    CASE_ID_FIELD_NUMBER: _ClassVar[int]
    PUBLIC_URL_FIELD_NUMBER: _ClassVar[int]
    case_id: str
    public_url: str
    def __init__(self, case_id: _Optional[str] = ..., public_url: _Optional[str] = ...) -> None: ...

class ResolveBountyReplayCaseResponse(_message.Message):
    __slots__ = ("case",)
    CASE_FIELD_NUMBER: _ClassVar[int]
    case: BountyReplayCase
    def __init__(self, case: _Optional[_Union[BountyReplayCase, _Mapping]] = ...) -> None: ...

class BountyReplayCase(_message.Message):
    __slots__ = ("schema_version", "bundle_version", "case_id", "project", "repository", "title", "task", "public_url", "issue_number", "pull_request_url", "pull_request_number", "source_commit", "reference_commit", "source_project_root", "reference_project_root", "frozen_evaluator_digest", "frozen_evaluator_root", "test_genie_project_id", "test_genie_plan", "bundle_digest", "frozen_evaluator_entrypoint", "frozen_evaluator_bundle_digest", "evaluator_bundle_id")
    SCHEMA_VERSION_FIELD_NUMBER: _ClassVar[int]
    BUNDLE_VERSION_FIELD_NUMBER: _ClassVar[int]
    CASE_ID_FIELD_NUMBER: _ClassVar[int]
    PROJECT_FIELD_NUMBER: _ClassVar[int]
    REPOSITORY_FIELD_NUMBER: _ClassVar[int]
    TITLE_FIELD_NUMBER: _ClassVar[int]
    TASK_FIELD_NUMBER: _ClassVar[int]
    PUBLIC_URL_FIELD_NUMBER: _ClassVar[int]
    ISSUE_NUMBER_FIELD_NUMBER: _ClassVar[int]
    PULL_REQUEST_URL_FIELD_NUMBER: _ClassVar[int]
    PULL_REQUEST_NUMBER_FIELD_NUMBER: _ClassVar[int]
    SOURCE_COMMIT_FIELD_NUMBER: _ClassVar[int]
    REFERENCE_COMMIT_FIELD_NUMBER: _ClassVar[int]
    SOURCE_PROJECT_ROOT_FIELD_NUMBER: _ClassVar[int]
    REFERENCE_PROJECT_ROOT_FIELD_NUMBER: _ClassVar[int]
    FROZEN_EVALUATOR_DIGEST_FIELD_NUMBER: _ClassVar[int]
    FROZEN_EVALUATOR_ROOT_FIELD_NUMBER: _ClassVar[int]
    TEST_GENIE_PROJECT_ID_FIELD_NUMBER: _ClassVar[int]
    TEST_GENIE_PLAN_FIELD_NUMBER: _ClassVar[int]
    BUNDLE_DIGEST_FIELD_NUMBER: _ClassVar[int]
    FROZEN_EVALUATOR_ENTRYPOINT_FIELD_NUMBER: _ClassVar[int]
    FROZEN_EVALUATOR_BUNDLE_DIGEST_FIELD_NUMBER: _ClassVar[int]
    EVALUATOR_BUNDLE_ID_FIELD_NUMBER: _ClassVar[int]
    schema_version: int
    bundle_version: int
    case_id: str
    project: str
    repository: str
    title: str
    task: str
    public_url: str
    issue_number: int
    pull_request_url: str
    pull_request_number: int
    source_commit: str
    reference_commit: str
    source_project_root: str
    reference_project_root: str
    frozen_evaluator_digest: str
    frozen_evaluator_root: str
    test_genie_project_id: str
    test_genie_plan: BountyTestGeniePlan
    bundle_digest: str
    frozen_evaluator_entrypoint: str
    frozen_evaluator_bundle_digest: str
    evaluator_bundle_id: str
    def __init__(self, schema_version: _Optional[int] = ..., bundle_version: _Optional[int] = ..., case_id: _Optional[str] = ..., project: _Optional[str] = ..., repository: _Optional[str] = ..., title: _Optional[str] = ..., task: _Optional[str] = ..., public_url: _Optional[str] = ..., issue_number: _Optional[int] = ..., pull_request_url: _Optional[str] = ..., pull_request_number: _Optional[int] = ..., source_commit: _Optional[str] = ..., reference_commit: _Optional[str] = ..., source_project_root: _Optional[str] = ..., reference_project_root: _Optional[str] = ..., frozen_evaluator_digest: _Optional[str] = ..., frozen_evaluator_root: _Optional[str] = ..., test_genie_project_id: _Optional[str] = ..., test_genie_plan: _Optional[_Union[BountyTestGeniePlan, _Mapping]] = ..., bundle_digest: _Optional[str] = ..., frozen_evaluator_entrypoint: _Optional[str] = ..., frozen_evaluator_bundle_digest: _Optional[str] = ..., evaluator_bundle_id: _Optional[str] = ...) -> None: ...

class BountyTestGeniePlan(_message.Message):
    __slots__ = ("validation_purpose", "required_strength", "phases", "evidence_policy", "deadline_policy")
    VALIDATION_PURPOSE_FIELD_NUMBER: _ClassVar[int]
    REQUIRED_STRENGTH_FIELD_NUMBER: _ClassVar[int]
    PHASES_FIELD_NUMBER: _ClassVar[int]
    EVIDENCE_POLICY_FIELD_NUMBER: _ClassVar[int]
    DEADLINE_POLICY_FIELD_NUMBER: _ClassVar[int]
    validation_purpose: str
    required_strength: str
    phases: _containers.RepeatedScalarFieldContainer[str]
    evidence_policy: BountyReplayEvidencePolicy
    deadline_policy: BountyReplayDeadlinePolicy
    def __init__(self, validation_purpose: _Optional[str] = ..., required_strength: _Optional[str] = ..., phases: _Optional[_Iterable[str]] = ..., evidence_policy: _Optional[_Union[BountyReplayEvidencePolicy, _Mapping]] = ..., deadline_policy: _Optional[_Union[BountyReplayDeadlinePolicy, _Mapping]] = ...) -> None: ...

class BountyReplayEvidencePolicy(_message.Message):
    __slots__ = ("require_behavioral_before", "require_source_snapshot", "allow_authorized_degradation", "required_evidence_kinds")
    REQUIRE_BEHAVIORAL_BEFORE_FIELD_NUMBER: _ClassVar[int]
    REQUIRE_SOURCE_SNAPSHOT_FIELD_NUMBER: _ClassVar[int]
    ALLOW_AUTHORIZED_DEGRADATION_FIELD_NUMBER: _ClassVar[int]
    REQUIRED_EVIDENCE_KINDS_FIELD_NUMBER: _ClassVar[int]
    require_behavioral_before: bool
    require_source_snapshot: bool
    allow_authorized_degradation: bool
    required_evidence_kinds: _containers.RepeatedScalarFieldContainer[str]
    def __init__(self, require_behavioral_before: _Optional[bool] = ..., require_source_snapshot: _Optional[bool] = ..., allow_authorized_degradation: _Optional[bool] = ..., required_evidence_kinds: _Optional[_Iterable[str]] = ...) -> None: ...

class BountyReplayDeadlinePolicy(_message.Message):
    __slots__ = ("queue_budget_milliseconds", "execution_budget_milliseconds", "maximum_attempts")
    QUEUE_BUDGET_MILLISECONDS_FIELD_NUMBER: _ClassVar[int]
    EXECUTION_BUDGET_MILLISECONDS_FIELD_NUMBER: _ClassVar[int]
    MAXIMUM_ATTEMPTS_FIELD_NUMBER: _ClassVar[int]
    queue_budget_milliseconds: int
    execution_budget_milliseconds: int
    maximum_attempts: int
    def __init__(self, queue_budget_milliseconds: _Optional[int] = ..., execution_budget_milliseconds: _Optional[int] = ..., maximum_attempts: _Optional[int] = ...) -> None: ...

class GetFrozenEvaluatorBundleRequest(_message.Message):
    __slots__ = ("evaluator_bundle_id",)
    EVALUATOR_BUNDLE_ID_FIELD_NUMBER: _ClassVar[int]
    evaluator_bundle_id: str
    def __init__(self, evaluator_bundle_id: _Optional[str] = ...) -> None: ...

class GetFrozenEvaluatorBundleResponse(_message.Message):
    __slots__ = ("evaluator_bundle",)
    EVALUATOR_BUNDLE_FIELD_NUMBER: _ClassVar[int]
    evaluator_bundle: FrozenEvaluatorBundle
    def __init__(self, evaluator_bundle: _Optional[_Union[FrozenEvaluatorBundle, _Mapping]] = ...) -> None: ...

class FrozenEvaluatorBundle(_message.Message):
    __slots__ = ("evaluator_bundle_id", "case_id", "bundle_version", "entrypoint", "evaluator_digest", "content_digest", "files")
    EVALUATOR_BUNDLE_ID_FIELD_NUMBER: _ClassVar[int]
    CASE_ID_FIELD_NUMBER: _ClassVar[int]
    BUNDLE_VERSION_FIELD_NUMBER: _ClassVar[int]
    ENTRYPOINT_FIELD_NUMBER: _ClassVar[int]
    EVALUATOR_DIGEST_FIELD_NUMBER: _ClassVar[int]
    CONTENT_DIGEST_FIELD_NUMBER: _ClassVar[int]
    FILES_FIELD_NUMBER: _ClassVar[int]
    evaluator_bundle_id: str
    case_id: str
    bundle_version: int
    entrypoint: str
    evaluator_digest: str
    content_digest: str
    files: _containers.RepeatedCompositeFieldContainer[FrozenEvaluatorFile]
    def __init__(self, evaluator_bundle_id: _Optional[str] = ..., case_id: _Optional[str] = ..., bundle_version: _Optional[int] = ..., entrypoint: _Optional[str] = ..., evaluator_digest: _Optional[str] = ..., content_digest: _Optional[str] = ..., files: _Optional[_Iterable[_Union[FrozenEvaluatorFile, _Mapping]]] = ...) -> None: ...

class FrozenEvaluatorFile(_message.Message):
    __slots__ = ("relative_path", "content", "sha256")
    RELATIVE_PATH_FIELD_NUMBER: _ClassVar[int]
    CONTENT_FIELD_NUMBER: _ClassVar[int]
    SHA256_FIELD_NUMBER: _ClassVar[int]
    relative_path: str
    content: bytes
    sha256: str
    def __init__(self, relative_path: _Optional[str] = ..., content: _Optional[bytes] = ..., sha256: _Optional[str] = ...) -> None: ...
