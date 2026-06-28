from google.protobuf.internal import containers as _containers
from google.protobuf.internal import enum_type_wrapper as _enum_type_wrapper
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Iterable as _Iterable, Mapping as _Mapping
from typing import ClassVar as _ClassVar, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class Bucket(int, metaclass=_enum_type_wrapper.EnumTypeWrapper):
    __slots__ = ()
    BUCKET_UNSPECIFIED: _ClassVar[Bucket]
    BUCKET_DO: _ClassVar[Bucket]
    BUCKET_REUSE: _ClassVar[Bucket]
    BUCKET_KNOW: _ClassVar[Bucket]
    BUCKET_STATE: _ClassVar[Bucket]
    BUCKET_ENTITY: _ClassVar[Bucket]

class Scope(int, metaclass=_enum_type_wrapper.EnumTypeWrapper):
    __slots__ = ()
    SCOPE_UNSPECIFIED: _ClassVar[Scope]
    SCOPE_PROJECT: _ClassVar[Scope]
    SCOPE_EXTERNAL: _ClassVar[Scope]

class ProviderState(int, metaclass=_enum_type_wrapper.EnumTypeWrapper):
    __slots__ = ()
    PROVIDER_STATE_UNSPECIFIED: _ClassVar[ProviderState]
    PROVIDER_STATE_ACTIVE: _ClassVar[ProviderState]
    PROVIDER_STATE_CAPABILITY_GAP: _ClassVar[ProviderState]

class ScoreScale(int, metaclass=_enum_type_wrapper.EnumTypeWrapper):
    __slots__ = ()
    SCORE_SCALE_UNSPECIFIED: _ClassVar[ScoreScale]
    SCORE_SCALE_COSINE_0_1: _ClassVar[ScoreScale]
    SCORE_SCALE_PERCENT_0_100: _ClassVar[ScoreScale]
    SCORE_SCALE_RAW: _ClassVar[ScoreScale]

class HttpMethod(int, metaclass=_enum_type_wrapper.EnumTypeWrapper):
    __slots__ = ()
    HTTP_METHOD_UNSPECIFIED: _ClassVar[HttpMethod]
    HTTP_METHOD_POST: _ClassVar[HttpMethod]
    HTTP_METHOD_GET: _ClassVar[HttpMethod]
BUCKET_UNSPECIFIED: Bucket
BUCKET_DO: Bucket
BUCKET_REUSE: Bucket
BUCKET_KNOW: Bucket
BUCKET_STATE: Bucket
BUCKET_ENTITY: Bucket
SCOPE_UNSPECIFIED: Scope
SCOPE_PROJECT: Scope
SCOPE_EXTERNAL: Scope
PROVIDER_STATE_UNSPECIFIED: ProviderState
PROVIDER_STATE_ACTIVE: ProviderState
PROVIDER_STATE_CAPABILITY_GAP: ProviderState
SCORE_SCALE_UNSPECIFIED: ScoreScale
SCORE_SCALE_COSINE_0_1: ScoreScale
SCORE_SCALE_PERCENT_0_100: ScoreScale
SCORE_SCALE_RAW: ScoreScale
HTTP_METHOD_UNSPECIFIED: HttpMethod
HTTP_METHOD_POST: HttpMethod
HTTP_METHOD_GET: HttpMethod

class Endpoint(_message.Message):
    __slots__ = ("http_json", "cli")
    HTTP_JSON_FIELD_NUMBER: _ClassVar[int]
    CLI_FIELD_NUMBER: _ClassVar[int]
    http_json: HttpJsonEndpoint
    cli: CliEndpoint
    def __init__(self, http_json: _Optional[_Union[HttpJsonEndpoint, _Mapping]] = ..., cli: _Optional[_Union[CliEndpoint, _Mapping]] = ...) -> None: ...

class HttpJsonEndpoint(_message.Message):
    __slots__ = ("scenario_id", "path", "method", "body_template", "headers")
    class HeadersEntry(_message.Message):
        __slots__ = ("key", "value")
        KEY_FIELD_NUMBER: _ClassVar[int]
        VALUE_FIELD_NUMBER: _ClassVar[int]
        key: str
        value: str
        def __init__(self, key: _Optional[str] = ..., value: _Optional[str] = ...) -> None: ...
    SCENARIO_ID_FIELD_NUMBER: _ClassVar[int]
    PATH_FIELD_NUMBER: _ClassVar[int]
    METHOD_FIELD_NUMBER: _ClassVar[int]
    BODY_TEMPLATE_FIELD_NUMBER: _ClassVar[int]
    HEADERS_FIELD_NUMBER: _ClassVar[int]
    scenario_id: str
    path: str
    method: HttpMethod
    body_template: str
    headers: _containers.ScalarMap[str, str]
    def __init__(self, scenario_id: _Optional[str] = ..., path: _Optional[str] = ..., method: _Optional[_Union[HttpMethod, str]] = ..., body_template: _Optional[str] = ..., headers: _Optional[_Mapping[str, str]] = ...) -> None: ...

class CliEndpoint(_message.Message):
    __slots__ = ("argv_template",)
    ARGV_TEMPLATE_FIELD_NUMBER: _ClassVar[int]
    argv_template: _containers.RepeatedScalarFieldContainer[str]
    def __init__(self, argv_template: _Optional[_Iterable[str]] = ...) -> None: ...

class ResultMapping(_message.Message):
    __slots__ = ("results_path", "id_field", "title_field", "score_field", "snippet_field", "path_field", "score_scale", "filter_field", "filter_value", "presence_field", "measure_field", "attestation_field", "confidence_field", "locations_field", "weak_field", "regime_field")
    RESULTS_PATH_FIELD_NUMBER: _ClassVar[int]
    ID_FIELD_FIELD_NUMBER: _ClassVar[int]
    TITLE_FIELD_FIELD_NUMBER: _ClassVar[int]
    SCORE_FIELD_FIELD_NUMBER: _ClassVar[int]
    SNIPPET_FIELD_FIELD_NUMBER: _ClassVar[int]
    PATH_FIELD_FIELD_NUMBER: _ClassVar[int]
    SCORE_SCALE_FIELD_NUMBER: _ClassVar[int]
    FILTER_FIELD_FIELD_NUMBER: _ClassVar[int]
    FILTER_VALUE_FIELD_NUMBER: _ClassVar[int]
    PRESENCE_FIELD_FIELD_NUMBER: _ClassVar[int]
    MEASURE_FIELD_FIELD_NUMBER: _ClassVar[int]
    ATTESTATION_FIELD_FIELD_NUMBER: _ClassVar[int]
    CONFIDENCE_FIELD_FIELD_NUMBER: _ClassVar[int]
    LOCATIONS_FIELD_FIELD_NUMBER: _ClassVar[int]
    WEAK_FIELD_FIELD_NUMBER: _ClassVar[int]
    REGIME_FIELD_FIELD_NUMBER: _ClassVar[int]
    results_path: str
    id_field: str
    title_field: str
    score_field: str
    snippet_field: str
    path_field: str
    score_scale: ScoreScale
    filter_field: str
    filter_value: str
    presence_field: str
    measure_field: str
    attestation_field: str
    confidence_field: str
    locations_field: str
    weak_field: str
    regime_field: str
    def __init__(self, results_path: _Optional[str] = ..., id_field: _Optional[str] = ..., title_field: _Optional[str] = ..., score_field: _Optional[str] = ..., snippet_field: _Optional[str] = ..., path_field: _Optional[str] = ..., score_scale: _Optional[_Union[ScoreScale, str]] = ..., filter_field: _Optional[str] = ..., filter_value: _Optional[str] = ..., presence_field: _Optional[str] = ..., measure_field: _Optional[str] = ..., attestation_field: _Optional[str] = ..., confidence_field: _Optional[str] = ..., locations_field: _Optional[str] = ..., weak_field: _Optional[str] = ..., regime_field: _Optional[str] = ...) -> None: ...

class FloorConfig(_message.Message):
    __slots__ = ("max_gap", "hard_floor")
    MAX_GAP_FIELD_NUMBER: _ClassVar[int]
    HARD_FLOOR_FIELD_NUMBER: _ClassVar[int]
    max_gap: float
    hard_floor: float
    def __init__(self, max_gap: _Optional[float] = ..., hard_floor: _Optional[float] = ...) -> None: ...

class Tuning(_message.Message):
    __slots__ = ("engine", "embed_model", "embed_task_prefix", "rerank_enabled", "rerank_blend", "rerank_shortlist", "floor")
    ENGINE_FIELD_NUMBER: _ClassVar[int]
    EMBED_MODEL_FIELD_NUMBER: _ClassVar[int]
    EMBED_TASK_PREFIX_FIELD_NUMBER: _ClassVar[int]
    RERANK_ENABLED_FIELD_NUMBER: _ClassVar[int]
    RERANK_BLEND_FIELD_NUMBER: _ClassVar[int]
    RERANK_SHORTLIST_FIELD_NUMBER: _ClassVar[int]
    FLOOR_FIELD_NUMBER: _ClassVar[int]
    engine: str
    embed_model: str
    embed_task_prefix: bool
    rerank_enabled: bool
    rerank_blend: bool
    rerank_shortlist: int
    floor: FloorConfig
    def __init__(self, engine: _Optional[str] = ..., embed_model: _Optional[str] = ..., embed_task_prefix: _Optional[bool] = ..., rerank_enabled: _Optional[bool] = ..., rerank_blend: _Optional[bool] = ..., rerank_shortlist: _Optional[int] = ..., floor: _Optional[_Union[FloorConfig, _Mapping]] = ...) -> None: ...

class ProviderDescriptor(_message.Message):
    __slots__ = ("provider_id", "provider_group", "bucket", "type", "description", "endpoint", "result_mapping", "query_hint", "status_endpoint", "scope", "state", "intended_home", "reindex_endpoint", "config_endpoint", "tuning")
    PROVIDER_ID_FIELD_NUMBER: _ClassVar[int]
    PROVIDER_GROUP_FIELD_NUMBER: _ClassVar[int]
    BUCKET_FIELD_NUMBER: _ClassVar[int]
    TYPE_FIELD_NUMBER: _ClassVar[int]
    DESCRIPTION_FIELD_NUMBER: _ClassVar[int]
    ENDPOINT_FIELD_NUMBER: _ClassVar[int]
    RESULT_MAPPING_FIELD_NUMBER: _ClassVar[int]
    QUERY_HINT_FIELD_NUMBER: _ClassVar[int]
    STATUS_ENDPOINT_FIELD_NUMBER: _ClassVar[int]
    SCOPE_FIELD_NUMBER: _ClassVar[int]
    STATE_FIELD_NUMBER: _ClassVar[int]
    INTENDED_HOME_FIELD_NUMBER: _ClassVar[int]
    REINDEX_ENDPOINT_FIELD_NUMBER: _ClassVar[int]
    CONFIG_ENDPOINT_FIELD_NUMBER: _ClassVar[int]
    TUNING_FIELD_NUMBER: _ClassVar[int]
    provider_id: str
    provider_group: str
    bucket: Bucket
    type: str
    description: str
    endpoint: Endpoint
    result_mapping: ResultMapping
    query_hint: str
    status_endpoint: Endpoint
    scope: Scope
    state: ProviderState
    intended_home: str
    reindex_endpoint: Endpoint
    config_endpoint: Endpoint
    tuning: Tuning
    def __init__(self, provider_id: _Optional[str] = ..., provider_group: _Optional[str] = ..., bucket: _Optional[_Union[Bucket, str]] = ..., type: _Optional[str] = ..., description: _Optional[str] = ..., endpoint: _Optional[_Union[Endpoint, _Mapping]] = ..., result_mapping: _Optional[_Union[ResultMapping, _Mapping]] = ..., query_hint: _Optional[str] = ..., status_endpoint: _Optional[_Union[Endpoint, _Mapping]] = ..., scope: _Optional[_Union[Scope, str]] = ..., state: _Optional[_Union[ProviderState, str]] = ..., intended_home: _Optional[str] = ..., reindex_endpoint: _Optional[_Union[Endpoint, _Mapping]] = ..., config_endpoint: _Optional[_Union[Endpoint, _Mapping]] = ..., tuning: _Optional[_Union[Tuning, _Mapping]] = ...) -> None: ...

class RegisterProviderRequest(_message.Message):
    __slots__ = ("descriptor", "control_token")
    DESCRIPTOR_FIELD_NUMBER: _ClassVar[int]
    CONTROL_TOKEN_FIELD_NUMBER: _ClassVar[int]
    descriptor: ProviderDescriptor
    control_token: str
    def __init__(self, descriptor: _Optional[_Union[ProviderDescriptor, _Mapping]] = ..., control_token: _Optional[str] = ...) -> None: ...

class RegisterProviderResponse(_message.Message):
    __slots__ = ("descriptor", "created", "control_token")
    DESCRIPTOR_FIELD_NUMBER: _ClassVar[int]
    CREATED_FIELD_NUMBER: _ClassVar[int]
    CONTROL_TOKEN_FIELD_NUMBER: _ClassVar[int]
    descriptor: ProviderDescriptor
    created: bool
    control_token: str
    def __init__(self, descriptor: _Optional[_Union[ProviderDescriptor, _Mapping]] = ..., created: _Optional[bool] = ..., control_token: _Optional[str] = ...) -> None: ...

class ListProvidersRequest(_message.Message):
    __slots__ = ("bucket", "type", "state")
    BUCKET_FIELD_NUMBER: _ClassVar[int]
    TYPE_FIELD_NUMBER: _ClassVar[int]
    STATE_FIELD_NUMBER: _ClassVar[int]
    bucket: Bucket
    type: str
    state: ProviderState
    def __init__(self, bucket: _Optional[_Union[Bucket, str]] = ..., type: _Optional[str] = ..., state: _Optional[_Union[ProviderState, str]] = ...) -> None: ...

class ListProvidersResponse(_message.Message):
    __slots__ = ("providers",)
    PROVIDERS_FIELD_NUMBER: _ClassVar[int]
    providers: _containers.RepeatedCompositeFieldContainer[ProviderDescriptor]
    def __init__(self, providers: _Optional[_Iterable[_Union[ProviderDescriptor, _Mapping]]] = ...) -> None: ...

class DeregisterProviderRequest(_message.Message):
    __slots__ = ("provider_id",)
    PROVIDER_ID_FIELD_NUMBER: _ClassVar[int]
    provider_id: str
    def __init__(self, provider_id: _Optional[str] = ...) -> None: ...

class DeregisterProviderResponse(_message.Message):
    __slots__ = ("removed",)
    REMOVED_FIELD_NUMBER: _ClassVar[int]
    removed: bool
    def __init__(self, removed: _Optional[bool] = ...) -> None: ...
