import datetime

from google.protobuf import timestamp_pb2 as _timestamp_pb2
from google.protobuf.internal import containers as _containers
from google.protobuf.internal import enum_type_wrapper as _enum_type_wrapper
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Iterable as _Iterable, Mapping as _Mapping
from typing import ClassVar as _ClassVar, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class SourceMediaKind(int, metaclass=_enum_type_wrapper.EnumTypeWrapper):
    __slots__ = ()
    SOURCE_MEDIA_KIND_UNSPECIFIED: _ClassVar[SourceMediaKind]
    SOURCE_MEDIA_KIND_IMAGE: _ClassVar[SourceMediaKind]
    SOURCE_MEDIA_KIND_VIDEO: _ClassVar[SourceMediaKind]

class MediaReadiness(int, metaclass=_enum_type_wrapper.EnumTypeWrapper):
    __slots__ = ()
    MEDIA_READINESS_UNSPECIFIED: _ClassVar[MediaReadiness]
    MEDIA_READINESS_READY: _ClassVar[MediaReadiness]
    MEDIA_READINESS_PENDING: _ClassVar[MediaReadiness]
    MEDIA_READINESS_REJECTED: _ClassVar[MediaReadiness]
    MEDIA_READINESS_RECONCILIATION_REQUIRED: _ClassVar[MediaReadiness]
SOURCE_MEDIA_KIND_UNSPECIFIED: SourceMediaKind
SOURCE_MEDIA_KIND_IMAGE: SourceMediaKind
SOURCE_MEDIA_KIND_VIDEO: SourceMediaKind
MEDIA_READINESS_UNSPECIFIED: MediaReadiness
MEDIA_READINESS_READY: MediaReadiness
MEDIA_READINESS_PENDING: MediaReadiness
MEDIA_READINESS_REJECTED: MediaReadiness
MEDIA_READINESS_RECONCILIATION_REQUIRED: MediaReadiness

class Project(_message.Message):
    __slots__ = ("id", "name", "created_at", "updated_at", "revision")
    ID_FIELD_NUMBER: _ClassVar[int]
    NAME_FIELD_NUMBER: _ClassVar[int]
    CREATED_AT_FIELD_NUMBER: _ClassVar[int]
    UPDATED_AT_FIELD_NUMBER: _ClassVar[int]
    REVISION_FIELD_NUMBER: _ClassVar[int]
    id: str
    name: str
    created_at: _timestamp_pb2.Timestamp
    updated_at: _timestamp_pb2.Timestamp
    revision: int
    def __init__(self, id: _Optional[str] = ..., name: _Optional[str] = ..., created_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., updated_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., revision: _Optional[int] = ...) -> None: ...

class ListProjectsRequest(_message.Message):
    __slots__ = ("page_size", "page_token")
    PAGE_SIZE_FIELD_NUMBER: _ClassVar[int]
    PAGE_TOKEN_FIELD_NUMBER: _ClassVar[int]
    page_size: int
    page_token: str
    def __init__(self, page_size: _Optional[int] = ..., page_token: _Optional[str] = ...) -> None: ...

class ListProjectsResponse(_message.Message):
    __slots__ = ("projects", "next_page_token")
    PROJECTS_FIELD_NUMBER: _ClassVar[int]
    NEXT_PAGE_TOKEN_FIELD_NUMBER: _ClassVar[int]
    projects: _containers.RepeatedCompositeFieldContainer[Project]
    next_page_token: str
    def __init__(self, projects: _Optional[_Iterable[_Union[Project, _Mapping]]] = ..., next_page_token: _Optional[str] = ...) -> None: ...

class CreateProjectRequest(_message.Message):
    __slots__ = ("name",)
    NAME_FIELD_NUMBER: _ClassVar[int]
    name: str
    def __init__(self, name: _Optional[str] = ...) -> None: ...

class CreateProjectResponse(_message.Message):
    __slots__ = ("project",)
    PROJECT_FIELD_NUMBER: _ClassVar[int]
    project: Project
    def __init__(self, project: _Optional[_Union[Project, _Mapping]] = ...) -> None: ...

class GetProjectRequest(_message.Message):
    __slots__ = ("id",)
    ID_FIELD_NUMBER: _ClassVar[int]
    id: str
    def __init__(self, id: _Optional[str] = ...) -> None: ...

class GetProjectResponse(_message.Message):
    __slots__ = ("project",)
    PROJECT_FIELD_NUMBER: _ClassVar[int]
    project: Project
    def __init__(self, project: _Optional[_Union[Project, _Mapping]] = ...) -> None: ...

class RenameProjectRequest(_message.Message):
    __slots__ = ("id", "name", "expected_revision")
    ID_FIELD_NUMBER: _ClassVar[int]
    NAME_FIELD_NUMBER: _ClassVar[int]
    EXPECTED_REVISION_FIELD_NUMBER: _ClassVar[int]
    id: str
    name: str
    expected_revision: int
    def __init__(self, id: _Optional[str] = ..., name: _Optional[str] = ..., expected_revision: _Optional[int] = ...) -> None: ...

class RenameProjectResponse(_message.Message):
    __slots__ = ("project",)
    PROJECT_FIELD_NUMBER: _ClassVar[int]
    project: Project
    def __init__(self, project: _Optional[_Union[Project, _Mapping]] = ...) -> None: ...

class ProjectConflict(_message.Message):
    __slots__ = ("project_id", "expected_revision", "current_revision")
    PROJECT_ID_FIELD_NUMBER: _ClassVar[int]
    EXPECTED_REVISION_FIELD_NUMBER: _ClassVar[int]
    CURRENT_REVISION_FIELD_NUMBER: _ClassVar[int]
    project_id: str
    expected_revision: int
    current_revision: int
    def __init__(self, project_id: _Optional[str] = ..., expected_revision: _Optional[int] = ..., current_revision: _Optional[int] = ...) -> None: ...

class SourceMedia(_message.Message):
    __slots__ = ("id", "project_id", "sha256", "size_bytes", "media_type", "width", "height", "orientation", "color_model", "color_profile_status", "original_filename", "imported_at", "import_method", "decoder", "kind", "readiness", "inspection_job_id", "inspection_issue_code", "inspection_issue_message", "inspection_receipt", "inspection_attempt_id", "inspection_job")
    ID_FIELD_NUMBER: _ClassVar[int]
    PROJECT_ID_FIELD_NUMBER: _ClassVar[int]
    SHA256_FIELD_NUMBER: _ClassVar[int]
    SIZE_BYTES_FIELD_NUMBER: _ClassVar[int]
    MEDIA_TYPE_FIELD_NUMBER: _ClassVar[int]
    WIDTH_FIELD_NUMBER: _ClassVar[int]
    HEIGHT_FIELD_NUMBER: _ClassVar[int]
    ORIENTATION_FIELD_NUMBER: _ClassVar[int]
    COLOR_MODEL_FIELD_NUMBER: _ClassVar[int]
    COLOR_PROFILE_STATUS_FIELD_NUMBER: _ClassVar[int]
    ORIGINAL_FILENAME_FIELD_NUMBER: _ClassVar[int]
    IMPORTED_AT_FIELD_NUMBER: _ClassVar[int]
    IMPORT_METHOD_FIELD_NUMBER: _ClassVar[int]
    DECODER_FIELD_NUMBER: _ClassVar[int]
    KIND_FIELD_NUMBER: _ClassVar[int]
    READINESS_FIELD_NUMBER: _ClassVar[int]
    INSPECTION_JOB_ID_FIELD_NUMBER: _ClassVar[int]
    INSPECTION_ISSUE_CODE_FIELD_NUMBER: _ClassVar[int]
    INSPECTION_ISSUE_MESSAGE_FIELD_NUMBER: _ClassVar[int]
    INSPECTION_RECEIPT_FIELD_NUMBER: _ClassVar[int]
    INSPECTION_ATTEMPT_ID_FIELD_NUMBER: _ClassVar[int]
    INSPECTION_JOB_FIELD_NUMBER: _ClassVar[int]
    id: str
    project_id: str
    sha256: str
    size_bytes: int
    media_type: str
    width: int
    height: int
    orientation: str
    color_model: str
    color_profile_status: str
    original_filename: str
    imported_at: _timestamp_pb2.Timestamp
    import_method: str
    decoder: str
    kind: SourceMediaKind
    readiness: MediaReadiness
    inspection_job_id: str
    inspection_issue_code: str
    inspection_issue_message: str
    inspection_receipt: MediaInspectionReceipt
    inspection_attempt_id: str
    inspection_job: InspectionJobStatus
    def __init__(self, id: _Optional[str] = ..., project_id: _Optional[str] = ..., sha256: _Optional[str] = ..., size_bytes: _Optional[int] = ..., media_type: _Optional[str] = ..., width: _Optional[int] = ..., height: _Optional[int] = ..., orientation: _Optional[str] = ..., color_model: _Optional[str] = ..., color_profile_status: _Optional[str] = ..., original_filename: _Optional[str] = ..., imported_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., import_method: _Optional[str] = ..., decoder: _Optional[str] = ..., kind: _Optional[_Union[SourceMediaKind, str]] = ..., readiness: _Optional[_Union[MediaReadiness, str]] = ..., inspection_job_id: _Optional[str] = ..., inspection_issue_code: _Optional[str] = ..., inspection_issue_message: _Optional[str] = ..., inspection_receipt: _Optional[_Union[MediaInspectionReceipt, _Mapping]] = ..., inspection_attempt_id: _Optional[str] = ..., inspection_job: _Optional[_Union[InspectionJobStatus, _Mapping]] = ...) -> None: ...

class MediaInspectionReceipt(_message.Message):
    __slots__ = ("version", "id", "sha256", "video_width", "video_height", "rotation_degrees", "duration_numerator", "duration_denominator", "frame_count", "stream_count", "clock_sha256", "source_sha256", "job_id", "source_size_bytes", "attempt_id", "tool_profile_id", "inspection_protocol_version", "clock_artifact_ref", "tool_profile_digest", "classification", "issue_code", "issue_message", "container", "video_stream_index", "codec", "coded_width", "coded_height", "sample_aspect_num", "sample_aspect_den", "color_policy", "color_primaries", "color_transfer", "color_matrix", "color_range", "time_base_numerator", "time_base_denominator", "stream_start_pts", "source_end_pts", "last_frame_duration_ticks", "eof_policy", "clock_artifact_size_bytes", "audio")
    VERSION_FIELD_NUMBER: _ClassVar[int]
    ID_FIELD_NUMBER: _ClassVar[int]
    SHA256_FIELD_NUMBER: _ClassVar[int]
    VIDEO_WIDTH_FIELD_NUMBER: _ClassVar[int]
    VIDEO_HEIGHT_FIELD_NUMBER: _ClassVar[int]
    ROTATION_DEGREES_FIELD_NUMBER: _ClassVar[int]
    DURATION_NUMERATOR_FIELD_NUMBER: _ClassVar[int]
    DURATION_DENOMINATOR_FIELD_NUMBER: _ClassVar[int]
    FRAME_COUNT_FIELD_NUMBER: _ClassVar[int]
    STREAM_COUNT_FIELD_NUMBER: _ClassVar[int]
    CLOCK_SHA256_FIELD_NUMBER: _ClassVar[int]
    SOURCE_SHA256_FIELD_NUMBER: _ClassVar[int]
    JOB_ID_FIELD_NUMBER: _ClassVar[int]
    SOURCE_SIZE_BYTES_FIELD_NUMBER: _ClassVar[int]
    ATTEMPT_ID_FIELD_NUMBER: _ClassVar[int]
    TOOL_PROFILE_ID_FIELD_NUMBER: _ClassVar[int]
    INSPECTION_PROTOCOL_VERSION_FIELD_NUMBER: _ClassVar[int]
    CLOCK_ARTIFACT_REF_FIELD_NUMBER: _ClassVar[int]
    TOOL_PROFILE_DIGEST_FIELD_NUMBER: _ClassVar[int]
    CLASSIFICATION_FIELD_NUMBER: _ClassVar[int]
    ISSUE_CODE_FIELD_NUMBER: _ClassVar[int]
    ISSUE_MESSAGE_FIELD_NUMBER: _ClassVar[int]
    CONTAINER_FIELD_NUMBER: _ClassVar[int]
    VIDEO_STREAM_INDEX_FIELD_NUMBER: _ClassVar[int]
    CODEC_FIELD_NUMBER: _ClassVar[int]
    CODED_WIDTH_FIELD_NUMBER: _ClassVar[int]
    CODED_HEIGHT_FIELD_NUMBER: _ClassVar[int]
    SAMPLE_ASPECT_NUM_FIELD_NUMBER: _ClassVar[int]
    SAMPLE_ASPECT_DEN_FIELD_NUMBER: _ClassVar[int]
    COLOR_POLICY_FIELD_NUMBER: _ClassVar[int]
    COLOR_PRIMARIES_FIELD_NUMBER: _ClassVar[int]
    COLOR_TRANSFER_FIELD_NUMBER: _ClassVar[int]
    COLOR_MATRIX_FIELD_NUMBER: _ClassVar[int]
    COLOR_RANGE_FIELD_NUMBER: _ClassVar[int]
    TIME_BASE_NUMERATOR_FIELD_NUMBER: _ClassVar[int]
    TIME_BASE_DENOMINATOR_FIELD_NUMBER: _ClassVar[int]
    STREAM_START_PTS_FIELD_NUMBER: _ClassVar[int]
    SOURCE_END_PTS_FIELD_NUMBER: _ClassVar[int]
    LAST_FRAME_DURATION_TICKS_FIELD_NUMBER: _ClassVar[int]
    EOF_POLICY_FIELD_NUMBER: _ClassVar[int]
    CLOCK_ARTIFACT_SIZE_BYTES_FIELD_NUMBER: _ClassVar[int]
    AUDIO_FIELD_NUMBER: _ClassVar[int]
    version: int
    id: str
    sha256: str
    video_width: int
    video_height: int
    rotation_degrees: int
    duration_numerator: int
    duration_denominator: int
    frame_count: int
    stream_count: int
    clock_sha256: str
    source_sha256: str
    job_id: str
    source_size_bytes: int
    attempt_id: str
    tool_profile_id: str
    inspection_protocol_version: int
    clock_artifact_ref: str
    tool_profile_digest: str
    classification: str
    issue_code: str
    issue_message: str
    container: str
    video_stream_index: int
    codec: str
    coded_width: int
    coded_height: int
    sample_aspect_num: int
    sample_aspect_den: int
    color_policy: str
    color_primaries: str
    color_transfer: str
    color_matrix: str
    color_range: str
    time_base_numerator: int
    time_base_denominator: int
    stream_start_pts: int
    source_end_pts: int
    last_frame_duration_ticks: int
    eof_policy: str
    clock_artifact_size_bytes: int
    audio: AudioInspectionFacts
    def __init__(self, version: _Optional[int] = ..., id: _Optional[str] = ..., sha256: _Optional[str] = ..., video_width: _Optional[int] = ..., video_height: _Optional[int] = ..., rotation_degrees: _Optional[int] = ..., duration_numerator: _Optional[int] = ..., duration_denominator: _Optional[int] = ..., frame_count: _Optional[int] = ..., stream_count: _Optional[int] = ..., clock_sha256: _Optional[str] = ..., source_sha256: _Optional[str] = ..., job_id: _Optional[str] = ..., source_size_bytes: _Optional[int] = ..., attempt_id: _Optional[str] = ..., tool_profile_id: _Optional[str] = ..., inspection_protocol_version: _Optional[int] = ..., clock_artifact_ref: _Optional[str] = ..., tool_profile_digest: _Optional[str] = ..., classification: _Optional[str] = ..., issue_code: _Optional[str] = ..., issue_message: _Optional[str] = ..., container: _Optional[str] = ..., video_stream_index: _Optional[int] = ..., codec: _Optional[str] = ..., coded_width: _Optional[int] = ..., coded_height: _Optional[int] = ..., sample_aspect_num: _Optional[int] = ..., sample_aspect_den: _Optional[int] = ..., color_policy: _Optional[str] = ..., color_primaries: _Optional[str] = ..., color_transfer: _Optional[str] = ..., color_matrix: _Optional[str] = ..., color_range: _Optional[str] = ..., time_base_numerator: _Optional[int] = ..., time_base_denominator: _Optional[int] = ..., stream_start_pts: _Optional[int] = ..., source_end_pts: _Optional[int] = ..., last_frame_duration_ticks: _Optional[int] = ..., eof_policy: _Optional[str] = ..., clock_artifact_size_bytes: _Optional[int] = ..., audio: _Optional[_Union[AudioInspectionFacts, _Mapping]] = ...) -> None: ...

class AudioInspectionFacts(_message.Message):
    __slots__ = ("version", "streams")
    VERSION_FIELD_NUMBER: _ClassVar[int]
    STREAMS_FIELD_NUMBER: _ClassVar[int]
    version: int
    streams: _containers.RepeatedCompositeFieldContainer[AudioStreamFact]
    def __init__(self, version: _Optional[int] = ..., streams: _Optional[_Iterable[_Union[AudioStreamFact, _Mapping]]] = ...) -> None: ...

class AudioStreamFact(_message.Message):
    __slots__ = ("original_stream_index", "codec")
    ORIGINAL_STREAM_INDEX_FIELD_NUMBER: _ClassVar[int]
    CODEC_FIELD_NUMBER: _ClassVar[int]
    original_stream_index: int
    codec: str
    def __init__(self, original_stream_index: _Optional[int] = ..., codec: _Optional[str] = ...) -> None: ...

class InspectionJobStatus(_message.Message):
    __slots__ = ("id", "state", "version", "current_attempt_id")
    ID_FIELD_NUMBER: _ClassVar[int]
    STATE_FIELD_NUMBER: _ClassVar[int]
    VERSION_FIELD_NUMBER: _ClassVar[int]
    CURRENT_ATTEMPT_ID_FIELD_NUMBER: _ClassVar[int]
    id: str
    state: str
    version: int
    current_attempt_id: str
    def __init__(self, id: _Optional[str] = ..., state: _Optional[str] = ..., version: _Optional[int] = ..., current_attempt_id: _Optional[str] = ...) -> None: ...

class EnsureSourceMediaInspectionRequest(_message.Message):
    __slots__ = ("project_id", "media_id")
    PROJECT_ID_FIELD_NUMBER: _ClassVar[int]
    MEDIA_ID_FIELD_NUMBER: _ClassVar[int]
    project_id: str
    media_id: str
    def __init__(self, project_id: _Optional[str] = ..., media_id: _Optional[str] = ...) -> None: ...

class EnsureSourceMediaInspectionResponse(_message.Message):
    __slots__ = ("source_media", "job")
    SOURCE_MEDIA_FIELD_NUMBER: _ClassVar[int]
    JOB_FIELD_NUMBER: _ClassVar[int]
    source_media: SourceMedia
    job: InspectionJobStatus
    def __init__(self, source_media: _Optional[_Union[SourceMedia, _Mapping]] = ..., job: _Optional[_Union[InspectionJobStatus, _Mapping]] = ...) -> None: ...

class RetrySourceMediaInspectionRequest(_message.Message):
    __slots__ = ("project_id", "media_id", "expected_job_version")
    PROJECT_ID_FIELD_NUMBER: _ClassVar[int]
    MEDIA_ID_FIELD_NUMBER: _ClassVar[int]
    EXPECTED_JOB_VERSION_FIELD_NUMBER: _ClassVar[int]
    project_id: str
    media_id: str
    expected_job_version: int
    def __init__(self, project_id: _Optional[str] = ..., media_id: _Optional[str] = ..., expected_job_version: _Optional[int] = ...) -> None: ...

class RetrySourceMediaInspectionResponse(_message.Message):
    __slots__ = ("source_media", "job")
    SOURCE_MEDIA_FIELD_NUMBER: _ClassVar[int]
    JOB_FIELD_NUMBER: _ClassVar[int]
    source_media: SourceMedia
    job: InspectionJobStatus
    def __init__(self, source_media: _Optional[_Union[SourceMedia, _Mapping]] = ..., job: _Optional[_Union[InspectionJobStatus, _Mapping]] = ...) -> None: ...

class CancelSourceMediaInspectionRequest(_message.Message):
    __slots__ = ("project_id", "media_id", "expected_job_version")
    PROJECT_ID_FIELD_NUMBER: _ClassVar[int]
    MEDIA_ID_FIELD_NUMBER: _ClassVar[int]
    EXPECTED_JOB_VERSION_FIELD_NUMBER: _ClassVar[int]
    project_id: str
    media_id: str
    expected_job_version: int
    def __init__(self, project_id: _Optional[str] = ..., media_id: _Optional[str] = ..., expected_job_version: _Optional[int] = ...) -> None: ...

class CancelSourceMediaInspectionResponse(_message.Message):
    __slots__ = ("source_media", "job")
    SOURCE_MEDIA_FIELD_NUMBER: _ClassVar[int]
    JOB_FIELD_NUMBER: _ClassVar[int]
    source_media: SourceMedia
    job: InspectionJobStatus
    def __init__(self, source_media: _Optional[_Union[SourceMedia, _Mapping]] = ..., job: _Optional[_Union[InspectionJobStatus, _Mapping]] = ...) -> None: ...

class ListSourceMediaRequest(_message.Message):
    __slots__ = ("project_id", "page_size", "page_token")
    PROJECT_ID_FIELD_NUMBER: _ClassVar[int]
    PAGE_SIZE_FIELD_NUMBER: _ClassVar[int]
    PAGE_TOKEN_FIELD_NUMBER: _ClassVar[int]
    project_id: str
    page_size: int
    page_token: str
    def __init__(self, project_id: _Optional[str] = ..., page_size: _Optional[int] = ..., page_token: _Optional[str] = ...) -> None: ...

class ListSourceMediaResponse(_message.Message):
    __slots__ = ("source_media", "next_page_token")
    SOURCE_MEDIA_FIELD_NUMBER: _ClassVar[int]
    NEXT_PAGE_TOKEN_FIELD_NUMBER: _ClassVar[int]
    source_media: _containers.RepeatedCompositeFieldContainer[SourceMedia]
    next_page_token: str
    def __init__(self, source_media: _Optional[_Iterable[_Union[SourceMedia, _Mapping]]] = ..., next_page_token: _Optional[str] = ...) -> None: ...

class GetSourceMediaRequest(_message.Message):
    __slots__ = ("project_id", "id")
    PROJECT_ID_FIELD_NUMBER: _ClassVar[int]
    ID_FIELD_NUMBER: _ClassVar[int]
    project_id: str
    id: str
    def __init__(self, project_id: _Optional[str] = ..., id: _Optional[str] = ...) -> None: ...

class GetSourceMediaResponse(_message.Message):
    __slots__ = ("source_media",)
    SOURCE_MEDIA_FIELD_NUMBER: _ClassVar[int]
    source_media: SourceMedia
    def __init__(self, source_media: _Optional[_Union[SourceMedia, _Mapping]] = ...) -> None: ...

class SourceClockPoint(_message.Message):
    __slots__ = ("ordinal", "pts")
    ORDINAL_FIELD_NUMBER: _ClassVar[int]
    PTS_FIELD_NUMBER: _ClassVar[int]
    ordinal: int
    pts: int
    def __init__(self, ordinal: _Optional[int] = ..., pts: _Optional[int] = ...) -> None: ...

class GetSourceMediaClockPageRequest(_message.Message):
    __slots__ = ("project_id", "media_id", "expected_clock_sha256", "page_size", "page_token")
    PROJECT_ID_FIELD_NUMBER: _ClassVar[int]
    MEDIA_ID_FIELD_NUMBER: _ClassVar[int]
    EXPECTED_CLOCK_SHA256_FIELD_NUMBER: _ClassVar[int]
    PAGE_SIZE_FIELD_NUMBER: _ClassVar[int]
    PAGE_TOKEN_FIELD_NUMBER: _ClassVar[int]
    project_id: str
    media_id: str
    expected_clock_sha256: str
    page_size: int
    page_token: str
    def __init__(self, project_id: _Optional[str] = ..., media_id: _Optional[str] = ..., expected_clock_sha256: _Optional[str] = ..., page_size: _Optional[int] = ..., page_token: _Optional[str] = ...) -> None: ...

class GetSourceMediaClockPageResponse(_message.Message):
    __slots__ = ("clock_sha256", "time_base_numerator", "time_base_denominator", "points", "next_page_token")
    CLOCK_SHA256_FIELD_NUMBER: _ClassVar[int]
    TIME_BASE_NUMERATOR_FIELD_NUMBER: _ClassVar[int]
    TIME_BASE_DENOMINATOR_FIELD_NUMBER: _ClassVar[int]
    POINTS_FIELD_NUMBER: _ClassVar[int]
    NEXT_PAGE_TOKEN_FIELD_NUMBER: _ClassVar[int]
    clock_sha256: str
    time_base_numerator: int
    time_base_denominator: int
    points: _containers.RepeatedCompositeFieldContainer[SourceClockPoint]
    next_page_token: str
    def __init__(self, clock_sha256: _Optional[str] = ..., time_base_numerator: _Optional[int] = ..., time_base_denominator: _Optional[int] = ..., points: _Optional[_Iterable[_Union[SourceClockPoint, _Mapping]]] = ..., next_page_token: _Optional[str] = ...) -> None: ...

class UploadSourceMediaResponse(_message.Message):
    __slots__ = ("source_media",)
    SOURCE_MEDIA_FIELD_NUMBER: _ClassVar[int]
    source_media: SourceMedia
    def __init__(self, source_media: _Optional[_Union[SourceMedia, _Mapping]] = ...) -> None: ...
