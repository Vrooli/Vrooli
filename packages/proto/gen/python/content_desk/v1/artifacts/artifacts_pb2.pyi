from google.protobuf.internal import containers as _containers
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Iterable as _Iterable, Mapping as _Mapping
from typing import ClassVar as _ClassVar, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class Draft(_message.Message):
    __slots__ = ("id", "campaign_id", "status", "post_type_id", "body", "channel", "format", "lane", "sku")
    ID_FIELD_NUMBER: _ClassVar[int]
    CAMPAIGN_ID_FIELD_NUMBER: _ClassVar[int]
    STATUS_FIELD_NUMBER: _ClassVar[int]
    POST_TYPE_ID_FIELD_NUMBER: _ClassVar[int]
    BODY_FIELD_NUMBER: _ClassVar[int]
    CHANNEL_FIELD_NUMBER: _ClassVar[int]
    FORMAT_FIELD_NUMBER: _ClassVar[int]
    LANE_FIELD_NUMBER: _ClassVar[int]
    SKU_FIELD_NUMBER: _ClassVar[int]
    id: str
    campaign_id: str
    status: str
    post_type_id: str
    body: str
    channel: str
    format: str
    lane: str
    sku: str
    def __init__(self, id: _Optional[str] = ..., campaign_id: _Optional[str] = ..., status: _Optional[str] = ..., post_type_id: _Optional[str] = ..., body: _Optional[str] = ..., channel: _Optional[str] = ..., format: _Optional[str] = ..., lane: _Optional[str] = ..., sku: _Optional[str] = ...) -> None: ...

class ListDraftsRequest(_message.Message):
    __slots__ = ()
    def __init__(self) -> None: ...

class ListDraftsResponse(_message.Message):
    __slots__ = ("drafts",)
    DRAFTS_FIELD_NUMBER: _ClassVar[int]
    drafts: _containers.RepeatedCompositeFieldContainer[Draft]
    def __init__(self, drafts: _Optional[_Iterable[_Union[Draft, _Mapping]]] = ...) -> None: ...

class CreateDraftRequest(_message.Message):
    __slots__ = ("campaign_id", "post_type_id", "body", "channel", "format", "lane", "sku")
    CAMPAIGN_ID_FIELD_NUMBER: _ClassVar[int]
    POST_TYPE_ID_FIELD_NUMBER: _ClassVar[int]
    BODY_FIELD_NUMBER: _ClassVar[int]
    CHANNEL_FIELD_NUMBER: _ClassVar[int]
    FORMAT_FIELD_NUMBER: _ClassVar[int]
    LANE_FIELD_NUMBER: _ClassVar[int]
    SKU_FIELD_NUMBER: _ClassVar[int]
    campaign_id: str
    post_type_id: str
    body: str
    channel: str
    format: str
    lane: str
    sku: str
    def __init__(self, campaign_id: _Optional[str] = ..., post_type_id: _Optional[str] = ..., body: _Optional[str] = ..., channel: _Optional[str] = ..., format: _Optional[str] = ..., lane: _Optional[str] = ..., sku: _Optional[str] = ...) -> None: ...

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

class PublishDraftRequest(_message.Message):
    __slots__ = ("id", "audience", "published_url", "platform_post_id", "series_id", "prior_publish_id")
    ID_FIELD_NUMBER: _ClassVar[int]
    AUDIENCE_FIELD_NUMBER: _ClassVar[int]
    PUBLISHED_URL_FIELD_NUMBER: _ClassVar[int]
    PLATFORM_POST_ID_FIELD_NUMBER: _ClassVar[int]
    SERIES_ID_FIELD_NUMBER: _ClassVar[int]
    PRIOR_PUBLISH_ID_FIELD_NUMBER: _ClassVar[int]
    id: str
    audience: str
    published_url: str
    platform_post_id: str
    series_id: str
    prior_publish_id: str
    def __init__(self, id: _Optional[str] = ..., audience: _Optional[str] = ..., published_url: _Optional[str] = ..., platform_post_id: _Optional[str] = ..., series_id: _Optional[str] = ..., prior_publish_id: _Optional[str] = ...) -> None: ...

class PublishDraftResponse(_message.Message):
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
    __slots__ = ("id",)
    ID_FIELD_NUMBER: _ClassVar[int]
    id: str
    def __init__(self, id: _Optional[str] = ...) -> None: ...

class ApproveDraftResponse(_message.Message):
    __slots__ = ("draft",)
    DRAFT_FIELD_NUMBER: _ClassVar[int]
    draft: Draft
    def __init__(self, draft: _Optional[_Union[Draft, _Mapping]] = ...) -> None: ...
