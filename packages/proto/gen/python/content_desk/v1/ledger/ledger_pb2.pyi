from google.protobuf.internal import containers as _containers
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Iterable as _Iterable, Mapping as _Mapping
from typing import ClassVar as _ClassVar, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class PublishRecord(_message.Message):
    __slots__ = ("id", "draft_id", "published_url", "platform_post_id")
    ID_FIELD_NUMBER: _ClassVar[int]
    DRAFT_ID_FIELD_NUMBER: _ClassVar[int]
    PUBLISHED_URL_FIELD_NUMBER: _ClassVar[int]
    PLATFORM_POST_ID_FIELD_NUMBER: _ClassVar[int]
    id: str
    draft_id: str
    published_url: str
    platform_post_id: str
    def __init__(self, id: _Optional[str] = ..., draft_id: _Optional[str] = ..., published_url: _Optional[str] = ..., platform_post_id: _Optional[str] = ...) -> None: ...

class ListPublishRecordsRequest(_message.Message):
    __slots__ = ()
    def __init__(self) -> None: ...

class ListPublishRecordsResponse(_message.Message):
    __slots__ = ("publish_records",)
    PUBLISH_RECORDS_FIELD_NUMBER: _ClassVar[int]
    publish_records: _containers.RepeatedCompositeFieldContainer[PublishRecord]
    def __init__(self, publish_records: _Optional[_Iterable[_Union[PublishRecord, _Mapping]]] = ...) -> None: ...

class ListContaminatedPublishRecordsRequest(_message.Message):
    __slots__ = ("claim_id",)
    CLAIM_ID_FIELD_NUMBER: _ClassVar[int]
    claim_id: str
    def __init__(self, claim_id: _Optional[str] = ...) -> None: ...

class ListContaminatedPublishRecordsResponse(_message.Message):
    __slots__ = ("publish_records",)
    PUBLISH_RECORDS_FIELD_NUMBER: _ClassVar[int]
    publish_records: _containers.RepeatedCompositeFieldContainer[PublishRecord]
    def __init__(self, publish_records: _Optional[_Iterable[_Union[PublishRecord, _Mapping]]] = ...) -> None: ...

class CoverageCell(_message.Message):
    __slots__ = ("campaign_id", "lane", "channel", "sku", "publish_count", "last_published_at", "stale")
    CAMPAIGN_ID_FIELD_NUMBER: _ClassVar[int]
    LANE_FIELD_NUMBER: _ClassVar[int]
    CHANNEL_FIELD_NUMBER: _ClassVar[int]
    SKU_FIELD_NUMBER: _ClassVar[int]
    PUBLISH_COUNT_FIELD_NUMBER: _ClassVar[int]
    LAST_PUBLISHED_AT_FIELD_NUMBER: _ClassVar[int]
    STALE_FIELD_NUMBER: _ClassVar[int]
    campaign_id: str
    lane: str
    channel: str
    sku: str
    publish_count: int
    last_published_at: str
    stale: bool
    def __init__(self, campaign_id: _Optional[str] = ..., lane: _Optional[str] = ..., channel: _Optional[str] = ..., sku: _Optional[str] = ..., publish_count: _Optional[int] = ..., last_published_at: _Optional[str] = ..., stale: _Optional[bool] = ...) -> None: ...

class ListCoverageRequest(_message.Message):
    __slots__ = ("stale_after_days",)
    STALE_AFTER_DAYS_FIELD_NUMBER: _ClassVar[int]
    stale_after_days: int
    def __init__(self, stale_after_days: _Optional[int] = ...) -> None: ...

class ListCoverageResponse(_message.Message):
    __slots__ = ("cells",)
    CELLS_FIELD_NUMBER: _ClassVar[int]
    cells: _containers.RepeatedCompositeFieldContainer[CoverageCell]
    def __init__(self, cells: _Optional[_Iterable[_Union[CoverageCell, _Mapping]]] = ...) -> None: ...

class IngestMetricSampleRequest(_message.Message):
    __slots__ = ("sample_id", "release_id", "draft_id", "metric", "value", "observed_at")
    SAMPLE_ID_FIELD_NUMBER: _ClassVar[int]
    RELEASE_ID_FIELD_NUMBER: _ClassVar[int]
    DRAFT_ID_FIELD_NUMBER: _ClassVar[int]
    METRIC_FIELD_NUMBER: _ClassVar[int]
    VALUE_FIELD_NUMBER: _ClassVar[int]
    OBSERVED_AT_FIELD_NUMBER: _ClassVar[int]
    sample_id: str
    release_id: str
    draft_id: str
    metric: str
    value: float
    observed_at: str
    def __init__(self, sample_id: _Optional[str] = ..., release_id: _Optional[str] = ..., draft_id: _Optional[str] = ..., metric: _Optional[str] = ..., value: _Optional[float] = ..., observed_at: _Optional[str] = ...) -> None: ...

class IngestMetricSampleResponse(_message.Message):
    __slots__ = ("sample_id", "accepted")
    SAMPLE_ID_FIELD_NUMBER: _ClassVar[int]
    ACCEPTED_FIELD_NUMBER: _ClassVar[int]
    sample_id: str
    accepted: bool
    def __init__(self, sample_id: _Optional[str] = ..., accepted: _Optional[bool] = ...) -> None: ...

class Remediation(_message.Message):
    __slots__ = ("id", "publish_record_id", "kind", "status", "note", "created_at", "resolved_at")
    ID_FIELD_NUMBER: _ClassVar[int]
    PUBLISH_RECORD_ID_FIELD_NUMBER: _ClassVar[int]
    KIND_FIELD_NUMBER: _ClassVar[int]
    STATUS_FIELD_NUMBER: _ClassVar[int]
    NOTE_FIELD_NUMBER: _ClassVar[int]
    CREATED_AT_FIELD_NUMBER: _ClassVar[int]
    RESOLVED_AT_FIELD_NUMBER: _ClassVar[int]
    id: str
    publish_record_id: str
    kind: str
    status: str
    note: str
    created_at: str
    resolved_at: str
    def __init__(self, id: _Optional[str] = ..., publish_record_id: _Optional[str] = ..., kind: _Optional[str] = ..., status: _Optional[str] = ..., note: _Optional[str] = ..., created_at: _Optional[str] = ..., resolved_at: _Optional[str] = ...) -> None: ...

class ListRemediationsRequest(_message.Message):
    __slots__ = ("publish_record_id", "open_only")
    PUBLISH_RECORD_ID_FIELD_NUMBER: _ClassVar[int]
    OPEN_ONLY_FIELD_NUMBER: _ClassVar[int]
    publish_record_id: str
    open_only: bool
    def __init__(self, publish_record_id: _Optional[str] = ..., open_only: _Optional[bool] = ...) -> None: ...

class ListRemediationsResponse(_message.Message):
    __slots__ = ("remediations",)
    REMEDIATIONS_FIELD_NUMBER: _ClassVar[int]
    remediations: _containers.RepeatedCompositeFieldContainer[Remediation]
    def __init__(self, remediations: _Optional[_Iterable[_Union[Remediation, _Mapping]]] = ...) -> None: ...

class CreateRemediationRequest(_message.Message):
    __slots__ = ("publish_record_id", "kind", "note")
    PUBLISH_RECORD_ID_FIELD_NUMBER: _ClassVar[int]
    KIND_FIELD_NUMBER: _ClassVar[int]
    NOTE_FIELD_NUMBER: _ClassVar[int]
    publish_record_id: str
    kind: str
    note: str
    def __init__(self, publish_record_id: _Optional[str] = ..., kind: _Optional[str] = ..., note: _Optional[str] = ...) -> None: ...

class CreateRemediationResponse(_message.Message):
    __slots__ = ("remediation",)
    REMEDIATION_FIELD_NUMBER: _ClassVar[int]
    remediation: Remediation
    def __init__(self, remediation: _Optional[_Union[Remediation, _Mapping]] = ...) -> None: ...

class ResolveRemediationRequest(_message.Message):
    __slots__ = ("id",)
    ID_FIELD_NUMBER: _ClassVar[int]
    id: str
    def __init__(self, id: _Optional[str] = ...) -> None: ...

class ResolveRemediationResponse(_message.Message):
    __slots__ = ("remediation",)
    REMEDIATION_FIELD_NUMBER: _ClassVar[int]
    remediation: Remediation
    def __init__(self, remediation: _Optional[_Union[Remediation, _Mapping]] = ...) -> None: ...
