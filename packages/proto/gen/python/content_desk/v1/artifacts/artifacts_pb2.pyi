from google.protobuf.internal import containers as _containers
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Iterable as _Iterable, Mapping as _Mapping
from typing import ClassVar as _ClassVar, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class Draft(_message.Message):
    __slots__ = ("id", "campaign_id", "status", "post_type_id", "body", "channel", "format", "lane", "sku", "scenario_name")
    ID_FIELD_NUMBER: _ClassVar[int]
    CAMPAIGN_ID_FIELD_NUMBER: _ClassVar[int]
    STATUS_FIELD_NUMBER: _ClassVar[int]
    POST_TYPE_ID_FIELD_NUMBER: _ClassVar[int]
    BODY_FIELD_NUMBER: _ClassVar[int]
    CHANNEL_FIELD_NUMBER: _ClassVar[int]
    FORMAT_FIELD_NUMBER: _ClassVar[int]
    LANE_FIELD_NUMBER: _ClassVar[int]
    SKU_FIELD_NUMBER: _ClassVar[int]
    SCENARIO_NAME_FIELD_NUMBER: _ClassVar[int]
    id: str
    campaign_id: str
    status: str
    post_type_id: str
    body: str
    channel: str
    format: str
    lane: str
    sku: str
    scenario_name: str
    def __init__(self, id: _Optional[str] = ..., campaign_id: _Optional[str] = ..., status: _Optional[str] = ..., post_type_id: _Optional[str] = ..., body: _Optional[str] = ..., channel: _Optional[str] = ..., format: _Optional[str] = ..., lane: _Optional[str] = ..., sku: _Optional[str] = ..., scenario_name: _Optional[str] = ...) -> None: ...

class ListDraftsRequest(_message.Message):
    __slots__ = ()
    def __init__(self) -> None: ...

class ListDraftsResponse(_message.Message):
    __slots__ = ("drafts",)
    DRAFTS_FIELD_NUMBER: _ClassVar[int]
    drafts: _containers.RepeatedCompositeFieldContainer[Draft]
    def __init__(self, drafts: _Optional[_Iterable[_Union[Draft, _Mapping]]] = ...) -> None: ...

class CreateDraftRequest(_message.Message):
    __slots__ = ("campaign_id", "post_type_id", "body", "channel", "format", "lane", "sku", "scenario_name")
    CAMPAIGN_ID_FIELD_NUMBER: _ClassVar[int]
    POST_TYPE_ID_FIELD_NUMBER: _ClassVar[int]
    BODY_FIELD_NUMBER: _ClassVar[int]
    CHANNEL_FIELD_NUMBER: _ClassVar[int]
    FORMAT_FIELD_NUMBER: _ClassVar[int]
    LANE_FIELD_NUMBER: _ClassVar[int]
    SKU_FIELD_NUMBER: _ClassVar[int]
    SCENARIO_NAME_FIELD_NUMBER: _ClassVar[int]
    campaign_id: str
    post_type_id: str
    body: str
    channel: str
    format: str
    lane: str
    sku: str
    scenario_name: str
    def __init__(self, campaign_id: _Optional[str] = ..., post_type_id: _Optional[str] = ..., body: _Optional[str] = ..., channel: _Optional[str] = ..., format: _Optional[str] = ..., lane: _Optional[str] = ..., sku: _Optional[str] = ..., scenario_name: _Optional[str] = ...) -> None: ...

class CreateDraftResponse(_message.Message):
    __slots__ = ("draft",)
    DRAFT_FIELD_NUMBER: _ClassVar[int]
    draft: Draft
    def __init__(self, draft: _Optional[_Union[Draft, _Mapping]] = ...) -> None: ...

class UpdateDraftBodyRequest(_message.Message):
    __slots__ = ("id", "body")
    ID_FIELD_NUMBER: _ClassVar[int]
    BODY_FIELD_NUMBER: _ClassVar[int]
    id: str
    body: str
    def __init__(self, id: _Optional[str] = ..., body: _Optional[str] = ...) -> None: ...

class UpdateDraftBodyResponse(_message.Message):
    __slots__ = ("draft",)
    DRAFT_FIELD_NUMBER: _ClassVar[int]
    draft: Draft
    def __init__(self, draft: _Optional[_Union[Draft, _Mapping]] = ...) -> None: ...

class DraftAttachment(_message.Message):
    __slots__ = ("id", "draft_id", "asset_id", "role", "aspect_ratio", "alt_text", "position")
    ID_FIELD_NUMBER: _ClassVar[int]
    DRAFT_ID_FIELD_NUMBER: _ClassVar[int]
    ASSET_ID_FIELD_NUMBER: _ClassVar[int]
    ROLE_FIELD_NUMBER: _ClassVar[int]
    ASPECT_RATIO_FIELD_NUMBER: _ClassVar[int]
    ALT_TEXT_FIELD_NUMBER: _ClassVar[int]
    POSITION_FIELD_NUMBER: _ClassVar[int]
    id: str
    draft_id: str
    asset_id: str
    role: str
    aspect_ratio: str
    alt_text: str
    position: int
    def __init__(self, id: _Optional[str] = ..., draft_id: _Optional[str] = ..., asset_id: _Optional[str] = ..., role: _Optional[str] = ..., aspect_ratio: _Optional[str] = ..., alt_text: _Optional[str] = ..., position: _Optional[int] = ...) -> None: ...

class AttachReleasedAssetRequest(_message.Message):
    __slots__ = ("draft_id", "asset_id", "role", "aspect_ratio", "alt_text", "position")
    DRAFT_ID_FIELD_NUMBER: _ClassVar[int]
    ASSET_ID_FIELD_NUMBER: _ClassVar[int]
    ROLE_FIELD_NUMBER: _ClassVar[int]
    ASPECT_RATIO_FIELD_NUMBER: _ClassVar[int]
    ALT_TEXT_FIELD_NUMBER: _ClassVar[int]
    POSITION_FIELD_NUMBER: _ClassVar[int]
    draft_id: str
    asset_id: str
    role: str
    aspect_ratio: str
    alt_text: str
    position: int
    def __init__(self, draft_id: _Optional[str] = ..., asset_id: _Optional[str] = ..., role: _Optional[str] = ..., aspect_ratio: _Optional[str] = ..., alt_text: _Optional[str] = ..., position: _Optional[int] = ...) -> None: ...

class AttachReleasedAssetResponse(_message.Message):
    __slots__ = ("attachment",)
    ATTACHMENT_FIELD_NUMBER: _ClassVar[int]
    attachment: DraftAttachment
    def __init__(self, attachment: _Optional[_Union[DraftAttachment, _Mapping]] = ...) -> None: ...

class ListDraftAttachmentsRequest(_message.Message):
    __slots__ = ("draft_id",)
    DRAFT_ID_FIELD_NUMBER: _ClassVar[int]
    draft_id: str
    def __init__(self, draft_id: _Optional[str] = ...) -> None: ...

class ListDraftAttachmentsResponse(_message.Message):
    __slots__ = ("attachments",)
    ATTACHMENTS_FIELD_NUMBER: _ClassVar[int]
    attachments: _containers.RepeatedCompositeFieldContainer[DraftAttachment]
    def __init__(self, attachments: _Optional[_Iterable[_Union[DraftAttachment, _Mapping]]] = ...) -> None: ...

class CommissionAgentWorkRequest(_message.Message):
    __slots__ = ("draft_id", "action")
    DRAFT_ID_FIELD_NUMBER: _ClassVar[int]
    ACTION_FIELD_NUMBER: _ClassVar[int]
    draft_id: str
    action: str
    def __init__(self, draft_id: _Optional[str] = ..., action: _Optional[str] = ...) -> None: ...

class CommissionAgentWorkResponse(_message.Message):
    __slots__ = ("commission_id", "task_id", "run_id", "status")
    COMMISSION_ID_FIELD_NUMBER: _ClassVar[int]
    TASK_ID_FIELD_NUMBER: _ClassVar[int]
    RUN_ID_FIELD_NUMBER: _ClassVar[int]
    STATUS_FIELD_NUMBER: _ClassVar[int]
    commission_id: str
    task_id: str
    run_id: str
    status: str
    def __init__(self, commission_id: _Optional[str] = ..., task_id: _Optional[str] = ..., run_id: _Optional[str] = ..., status: _Optional[str] = ...) -> None: ...

class GetAgentWorkResultRequest(_message.Message):
    __slots__ = ("commission_id",)
    COMMISSION_ID_FIELD_NUMBER: _ClassVar[int]
    commission_id: str
    def __init__(self, commission_id: _Optional[str] = ...) -> None: ...

class GetAgentWorkResultResponse(_message.Message):
    __slots__ = ("run_id", "status", "body")
    RUN_ID_FIELD_NUMBER: _ClassVar[int]
    STATUS_FIELD_NUMBER: _ClassVar[int]
    BODY_FIELD_NUMBER: _ClassVar[int]
    run_id: str
    status: str
    body: str
    def __init__(self, run_id: _Optional[str] = ..., status: _Optional[str] = ..., body: _Optional[str] = ...) -> None: ...

class AdoptAgentSuggestionRequest(_message.Message):
    __slots__ = ("commission_id", "body")
    COMMISSION_ID_FIELD_NUMBER: _ClassVar[int]
    BODY_FIELD_NUMBER: _ClassVar[int]
    commission_id: str
    body: str
    def __init__(self, commission_id: _Optional[str] = ..., body: _Optional[str] = ...) -> None: ...

class AdoptAgentSuggestionResponse(_message.Message):
    __slots__ = ("draft",)
    DRAFT_FIELD_NUMBER: _ClassVar[int]
    draft: Draft
    def __init__(self, draft: _Optional[_Union[Draft, _Mapping]] = ...) -> None: ...

class SubmitReleaseDraftRequest(_message.Message):
    __slots__ = ("id", "identity_id", "lane", "idempotency_key", "disclosure_visible")
    ID_FIELD_NUMBER: _ClassVar[int]
    IDENTITY_ID_FIELD_NUMBER: _ClassVar[int]
    LANE_FIELD_NUMBER: _ClassVar[int]
    IDEMPOTENCY_KEY_FIELD_NUMBER: _ClassVar[int]
    DISCLOSURE_VISIBLE_FIELD_NUMBER: _ClassVar[int]
    id: str
    identity_id: str
    lane: str
    idempotency_key: str
    disclosure_visible: bool
    def __init__(self, id: _Optional[str] = ..., identity_id: _Optional[str] = ..., lane: _Optional[str] = ..., idempotency_key: _Optional[str] = ..., disclosure_visible: _Optional[bool] = ...) -> None: ...

class SubmitReleaseDraftResponse(_message.Message):
    __slots__ = ("draft", "release_id", "action_id", "release_status")
    DRAFT_FIELD_NUMBER: _ClassVar[int]
    RELEASE_ID_FIELD_NUMBER: _ClassVar[int]
    ACTION_ID_FIELD_NUMBER: _ClassVar[int]
    RELEASE_STATUS_FIELD_NUMBER: _ClassVar[int]
    draft: Draft
    release_id: str
    action_id: str
    release_status: str
    def __init__(self, draft: _Optional[_Union[Draft, _Mapping]] = ..., release_id: _Optional[str] = ..., action_id: _Optional[str] = ..., release_status: _Optional[str] = ...) -> None: ...

class RecordReleaseOutcomeRequest(_message.Message):
    __slots__ = ("receipt_id", "draft_id", "status", "platform_post_id", "published_url", "published_at")
    RECEIPT_ID_FIELD_NUMBER: _ClassVar[int]
    DRAFT_ID_FIELD_NUMBER: _ClassVar[int]
    STATUS_FIELD_NUMBER: _ClassVar[int]
    PLATFORM_POST_ID_FIELD_NUMBER: _ClassVar[int]
    PUBLISHED_URL_FIELD_NUMBER: _ClassVar[int]
    PUBLISHED_AT_FIELD_NUMBER: _ClassVar[int]
    receipt_id: str
    draft_id: str
    status: str
    platform_post_id: str
    published_url: str
    published_at: str
    def __init__(self, receipt_id: _Optional[str] = ..., draft_id: _Optional[str] = ..., status: _Optional[str] = ..., platform_post_id: _Optional[str] = ..., published_url: _Optional[str] = ..., published_at: _Optional[str] = ...) -> None: ...

class RecordReleaseOutcomeResponse(_message.Message):
    __slots__ = ("draft", "publish_record_id")
    DRAFT_FIELD_NUMBER: _ClassVar[int]
    PUBLISH_RECORD_ID_FIELD_NUMBER: _ClassVar[int]
    draft: Draft
    publish_record_id: str
    def __init__(self, draft: _Optional[_Union[Draft, _Mapping]] = ..., publish_record_id: _Optional[str] = ...) -> None: ...

class TransitionDraftRequest(_message.Message):
    __slots__ = ("id", "event")
    ID_FIELD_NUMBER: _ClassVar[int]
    EVENT_FIELD_NUMBER: _ClassVar[int]
    id: str
    event: str
    def __init__(self, id: _Optional[str] = ..., event: _Optional[str] = ...) -> None: ...

class TransitionDraftResponse(_message.Message):
    __slots__ = ("draft",)
    DRAFT_FIELD_NUMBER: _ClassVar[int]
    draft: Draft
    def __init__(self, draft: _Optional[_Union[Draft, _Mapping]] = ...) -> None: ...

class ApproveDraftRequest(_message.Message):
    __slots__ = ("id", "identity_id", "lane")
    ID_FIELD_NUMBER: _ClassVar[int]
    IDENTITY_ID_FIELD_NUMBER: _ClassVar[int]
    LANE_FIELD_NUMBER: _ClassVar[int]
    id: str
    identity_id: str
    lane: str
    def __init__(self, id: _Optional[str] = ..., identity_id: _Optional[str] = ..., lane: _Optional[str] = ...) -> None: ...

class ApproveDraftResponse(_message.Message):
    __slots__ = ("draft",)
    DRAFT_FIELD_NUMBER: _ClassVar[int]
    draft: Draft
    def __init__(self, draft: _Optional[_Union[Draft, _Mapping]] = ...) -> None: ...
