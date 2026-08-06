from google.protobuf.internal import containers as _containers
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Iterable as _Iterable, Mapping as _Mapping
from typing import ClassVar as _ClassVar, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class GetOverviewRequest(_message.Message):
    __slots__ = ()
    def __init__(self) -> None: ...

class Identity(_message.Message):
    __slots__ = ("id", "platform_id", "purpose", "environment_ref", "credential_ref", "status", "lane_grants", "handle", "display_label", "lifecycle", "automation_mode")
    ID_FIELD_NUMBER: _ClassVar[int]
    PLATFORM_ID_FIELD_NUMBER: _ClassVar[int]
    PURPOSE_FIELD_NUMBER: _ClassVar[int]
    ENVIRONMENT_REF_FIELD_NUMBER: _ClassVar[int]
    CREDENTIAL_REF_FIELD_NUMBER: _ClassVar[int]
    STATUS_FIELD_NUMBER: _ClassVar[int]
    LANE_GRANTS_FIELD_NUMBER: _ClassVar[int]
    HANDLE_FIELD_NUMBER: _ClassVar[int]
    DISPLAY_LABEL_FIELD_NUMBER: _ClassVar[int]
    LIFECYCLE_FIELD_NUMBER: _ClassVar[int]
    AUTOMATION_MODE_FIELD_NUMBER: _ClassVar[int]
    id: str
    platform_id: str
    purpose: str
    environment_ref: str
    credential_ref: str
    status: str
    lane_grants: _containers.RepeatedScalarFieldContainer[str]
    handle: str
    display_label: str
    lifecycle: str
    automation_mode: str
    def __init__(self, id: _Optional[str] = ..., platform_id: _Optional[str] = ..., purpose: _Optional[str] = ..., environment_ref: _Optional[str] = ..., credential_ref: _Optional[str] = ..., status: _Optional[str] = ..., lane_grants: _Optional[_Iterable[str]] = ..., handle: _Optional[str] = ..., display_label: _Optional[str] = ..., lifecycle: _Optional[str] = ..., automation_mode: _Optional[str] = ...) -> None: ...

class Action(_message.Message):
    __slots__ = ("id", "identity_id", "kind", "window", "status", "rolled_count")
    ID_FIELD_NUMBER: _ClassVar[int]
    IDENTITY_ID_FIELD_NUMBER: _ClassVar[int]
    KIND_FIELD_NUMBER: _ClassVar[int]
    WINDOW_FIELD_NUMBER: _ClassVar[int]
    STATUS_FIELD_NUMBER: _ClassVar[int]
    ROLLED_COUNT_FIELD_NUMBER: _ClassVar[int]
    id: str
    identity_id: str
    kind: str
    window: str
    status: str
    rolled_count: int
    def __init__(self, id: _Optional[str] = ..., identity_id: _Optional[str] = ..., kind: _Optional[str] = ..., window: _Optional[str] = ..., status: _Optional[str] = ..., rolled_count: _Optional[int] = ...) -> None: ...

class GetOverviewResponse(_message.Message):
    __slots__ = ("identities", "actions")
    IDENTITIES_FIELD_NUMBER: _ClassVar[int]
    ACTIONS_FIELD_NUMBER: _ClassVar[int]
    identities: _containers.RepeatedCompositeFieldContainer[Identity]
    actions: _containers.RepeatedCompositeFieldContainer[Action]
    def __init__(self, identities: _Optional[_Iterable[_Union[Identity, _Mapping]]] = ..., actions: _Optional[_Iterable[_Union[Action, _Mapping]]] = ...) -> None: ...

class GetEligibilityRequest(_message.Message):
    __slots__ = ("identity_id", "lane")
    IDENTITY_ID_FIELD_NUMBER: _ClassVar[int]
    LANE_FIELD_NUMBER: _ClassVar[int]
    identity_id: str
    lane: str
    def __init__(self, identity_id: _Optional[str] = ..., lane: _Optional[str] = ...) -> None: ...

class GetEligibilityResponse(_message.Message):
    __slots__ = ("eligibility",)
    ELIGIBILITY_FIELD_NUMBER: _ClassVar[int]
    eligibility: str
    def __init__(self, eligibility: _Optional[str] = ...) -> None: ...

class SubmitReleaseRequest(_message.Message):
    __slots__ = ("identity_id", "lane", "draft_id", "idempotency_key", "asset_ids", "disclosure_visible")
    IDENTITY_ID_FIELD_NUMBER: _ClassVar[int]
    LANE_FIELD_NUMBER: _ClassVar[int]
    DRAFT_ID_FIELD_NUMBER: _ClassVar[int]
    IDEMPOTENCY_KEY_FIELD_NUMBER: _ClassVar[int]
    ASSET_IDS_FIELD_NUMBER: _ClassVar[int]
    DISCLOSURE_VISIBLE_FIELD_NUMBER: _ClassVar[int]
    identity_id: str
    lane: str
    draft_id: str
    idempotency_key: str
    asset_ids: _containers.RepeatedScalarFieldContainer[str]
    disclosure_visible: bool
    def __init__(self, identity_id: _Optional[str] = ..., lane: _Optional[str] = ..., draft_id: _Optional[str] = ..., idempotency_key: _Optional[str] = ..., asset_ids: _Optional[_Iterable[str]] = ..., disclosure_visible: _Optional[bool] = ...) -> None: ...

class ReleaseReceipt(_message.Message):
    __slots__ = ("id", "draft_id", "action_id", "status", "platform_post_id", "published_url", "first_comment_status")
    ID_FIELD_NUMBER: _ClassVar[int]
    DRAFT_ID_FIELD_NUMBER: _ClassVar[int]
    ACTION_ID_FIELD_NUMBER: _ClassVar[int]
    STATUS_FIELD_NUMBER: _ClassVar[int]
    PLATFORM_POST_ID_FIELD_NUMBER: _ClassVar[int]
    PUBLISHED_URL_FIELD_NUMBER: _ClassVar[int]
    FIRST_COMMENT_STATUS_FIELD_NUMBER: _ClassVar[int]
    id: str
    draft_id: str
    action_id: str
    status: str
    platform_post_id: str
    published_url: str
    first_comment_status: str
    def __init__(self, id: _Optional[str] = ..., draft_id: _Optional[str] = ..., action_id: _Optional[str] = ..., status: _Optional[str] = ..., platform_post_id: _Optional[str] = ..., published_url: _Optional[str] = ..., first_comment_status: _Optional[str] = ...) -> None: ...

class SubmitReleaseResponse(_message.Message):
    __slots__ = ("receipt",)
    RECEIPT_FIELD_NUMBER: _ClassVar[int]
    receipt: ReleaseReceipt
    def __init__(self, receipt: _Optional[_Union[ReleaseReceipt, _Mapping]] = ...) -> None: ...

class DeliverReleaseOutcomeRequest(_message.Message):
    __slots__ = ("release_id",)
    RELEASE_ID_FIELD_NUMBER: _ClassVar[int]
    release_id: str
    def __init__(self, release_id: _Optional[str] = ...) -> None: ...

class DeliverReleaseOutcomeResponse(_message.Message):
    __slots__ = ("delivery_status",)
    DELIVERY_STATUS_FIELD_NUMBER: _ClassVar[int]
    delivery_status: str
    def __init__(self, delivery_status: _Optional[str] = ...) -> None: ...

class DeliverMetricSampleRequest(_message.Message):
    __slots__ = ("sample_id",)
    SAMPLE_ID_FIELD_NUMBER: _ClassVar[int]
    sample_id: str
    def __init__(self, sample_id: _Optional[str] = ...) -> None: ...

class DeliverMetricSampleResponse(_message.Message):
    __slots__ = ("delivery_status",)
    DELIVERY_STATUS_FIELD_NUMBER: _ClassVar[int]
    delivery_status: str
    def __init__(self, delivery_status: _Optional[str] = ...) -> None: ...

class AssignAutomationRequest(_message.Message):
    __slots__ = ("identity_id", "session_profile_ref", "enabled_action_kinds", "operator_note", "workflow_ref", "consumer_profile_key")
    IDENTITY_ID_FIELD_NUMBER: _ClassVar[int]
    SESSION_PROFILE_REF_FIELD_NUMBER: _ClassVar[int]
    ENABLED_ACTION_KINDS_FIELD_NUMBER: _ClassVar[int]
    OPERATOR_NOTE_FIELD_NUMBER: _ClassVar[int]
    WORKFLOW_REF_FIELD_NUMBER: _ClassVar[int]
    CONSUMER_PROFILE_KEY_FIELD_NUMBER: _ClassVar[int]
    identity_id: str
    session_profile_ref: str
    enabled_action_kinds: _containers.RepeatedScalarFieldContainer[str]
    operator_note: str
    workflow_ref: str
    consumer_profile_key: str
    def __init__(self, identity_id: _Optional[str] = ..., session_profile_ref: _Optional[str] = ..., enabled_action_kinds: _Optional[_Iterable[str]] = ..., operator_note: _Optional[str] = ..., workflow_ref: _Optional[str] = ..., consumer_profile_key: _Optional[str] = ...) -> None: ...

class AssignAutomationResponse(_message.Message):
    __slots__ = ("identity_id",)
    IDENTITY_ID_FIELD_NUMBER: _ClassVar[int]
    identity_id: str
    def __init__(self, identity_id: _Optional[str] = ...) -> None: ...

class DispatchBrowserActionRequest(_message.Message):
    __slots__ = ("action_id",)
    ACTION_ID_FIELD_NUMBER: _ClassVar[int]
    action_id: str
    def __init__(self, action_id: _Optional[str] = ...) -> None: ...

class DispatchBrowserActionResponse(_message.Message):
    __slots__ = ("execution_id",)
    EXECUTION_ID_FIELD_NUMBER: _ClassVar[int]
    execution_id: str
    def __init__(self, execution_id: _Optional[str] = ...) -> None: ...

class GetBrowserExecutionReviewRequest(_message.Message):
    __slots__ = ("action_id",)
    ACTION_ID_FIELD_NUMBER: _ClassVar[int]
    action_id: str
    def __init__(self, action_id: _Optional[str] = ...) -> None: ...

class GetBrowserExecutionReviewResponse(_message.Message):
    __slots__ = ("execution_id", "status", "failure", "artifact_refs")
    EXECUTION_ID_FIELD_NUMBER: _ClassVar[int]
    STATUS_FIELD_NUMBER: _ClassVar[int]
    FAILURE_FIELD_NUMBER: _ClassVar[int]
    ARTIFACT_REFS_FIELD_NUMBER: _ClassVar[int]
    execution_id: str
    status: str
    failure: str
    artifact_refs: _containers.RepeatedScalarFieldContainer[str]
    def __init__(self, execution_id: _Optional[str] = ..., status: _Optional[str] = ..., failure: _Optional[str] = ..., artifact_refs: _Optional[_Iterable[str]] = ...) -> None: ...
