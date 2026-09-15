import datetime

from google.protobuf import timestamp_pb2 as _timestamp_pb2
from landing_page_business_suite.v1 import shared_pb2 as _shared_pb2
from google.protobuf.internal import containers as _containers
from google.protobuf.internal import enum_type_wrapper as _enum_type_wrapper
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Iterable as _Iterable, Mapping as _Mapping
from typing import ClassVar as _ClassVar, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class ExperimentVerdict(int, metaclass=_enum_type_wrapper.EnumTypeWrapper):
    __slots__ = ()
    EXPERIMENT_VERDICT_UNSPECIFIED: _ClassVar[ExperimentVerdict]
    EXPERIMENT_VERDICT_CONTROL: _ClassVar[ExperimentVerdict]
    EXPERIMENT_VERDICT_INSUFFICIENT_DATA: _ClassVar[ExperimentVerdict]
    EXPERIMENT_VERDICT_INCONCLUSIVE: _ClassVar[ExperimentVerdict]
    EXPERIMENT_VERDICT_LEADING: _ClassVar[ExperimentVerdict]
    EXPERIMENT_VERDICT_TRAILING: _ClassVar[ExperimentVerdict]
EXPERIMENT_VERDICT_UNSPECIFIED: ExperimentVerdict
EXPERIMENT_VERDICT_CONTROL: ExperimentVerdict
EXPERIMENT_VERDICT_INSUFFICIENT_DATA: ExperimentVerdict
EXPERIMENT_VERDICT_INCONCLUSIVE: ExperimentVerdict
EXPERIMENT_VERDICT_LEADING: ExperimentVerdict
EXPERIMENT_VERDICT_TRAILING: ExperimentVerdict

class GetBusinessDigestRequest(_message.Message):
    __slots__ = ("window_days",)
    WINDOW_DAYS_FIELD_NUMBER: _ClassVar[int]
    window_days: int
    def __init__(self, window_days: _Optional[int] = ...) -> None: ...

class DigestWindow(_message.Message):
    __slots__ = ("start", "end", "days")
    START_FIELD_NUMBER: _ClassVar[int]
    END_FIELD_NUMBER: _ClassVar[int]
    DAYS_FIELD_NUMBER: _ClassVar[int]
    start: _timestamp_pb2.Timestamp
    end: _timestamp_pb2.Timestamp
    days: int
    def __init__(self, start: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., end: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., days: _Optional[int] = ...) -> None: ...

class FunnelStep(_message.Message):
    __slots__ = ("key", "label", "value", "denominator_key", "denominator")
    KEY_FIELD_NUMBER: _ClassVar[int]
    LABEL_FIELD_NUMBER: _ClassVar[int]
    VALUE_FIELD_NUMBER: _ClassVar[int]
    DENOMINATOR_KEY_FIELD_NUMBER: _ClassVar[int]
    DENOMINATOR_FIELD_NUMBER: _ClassVar[int]
    key: str
    label: str
    value: int
    denominator_key: str
    denominator: int
    def __init__(self, key: _Optional[str] = ..., label: _Optional[str] = ..., value: _Optional[int] = ..., denominator_key: _Optional[str] = ..., denominator: _Optional[int] = ...) -> None: ...

class Funnel(_message.Message):
    __slots__ = ("steps", "unattributed_checkouts_started", "unattributed_paid", "paid_checkouts_total", "paid_revenue_minor", "currency")
    STEPS_FIELD_NUMBER: _ClassVar[int]
    UNATTRIBUTED_CHECKOUTS_STARTED_FIELD_NUMBER: _ClassVar[int]
    UNATTRIBUTED_PAID_FIELD_NUMBER: _ClassVar[int]
    PAID_CHECKOUTS_TOTAL_FIELD_NUMBER: _ClassVar[int]
    PAID_REVENUE_MINOR_FIELD_NUMBER: _ClassVar[int]
    CURRENCY_FIELD_NUMBER: _ClassVar[int]
    steps: _containers.RepeatedCompositeFieldContainer[FunnelStep]
    unattributed_checkouts_started: int
    unattributed_paid: int
    paid_checkouts_total: int
    paid_revenue_minor: int
    currency: str
    def __init__(self, steps: _Optional[_Iterable[_Union[FunnelStep, _Mapping]]] = ..., unattributed_checkouts_started: _Optional[int] = ..., unattributed_paid: _Optional[int] = ..., paid_checkouts_total: _Optional[int] = ..., paid_revenue_minor: _Optional[int] = ..., currency: _Optional[str] = ...) -> None: ...

class Retention(_message.Message):
    __slots__ = ("cohort_start", "cohort_end", "cohort_size", "still_active")
    COHORT_START_FIELD_NUMBER: _ClassVar[int]
    COHORT_END_FIELD_NUMBER: _ClassVar[int]
    COHORT_SIZE_FIELD_NUMBER: _ClassVar[int]
    STILL_ACTIVE_FIELD_NUMBER: _ClassVar[int]
    cohort_start: _timestamp_pb2.Timestamp
    cohort_end: _timestamp_pb2.Timestamp
    cohort_size: int
    still_active: int
    def __init__(self, cohort_start: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., cohort_end: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., cohort_size: _Optional[int] = ..., still_active: _Optional[int] = ...) -> None: ...

class ArmMetric(_message.Message):
    __slots__ = ("successes", "trials", "rate_percent", "probability_beats_control", "verdict")
    SUCCESSES_FIELD_NUMBER: _ClassVar[int]
    TRIALS_FIELD_NUMBER: _ClassVar[int]
    RATE_PERCENT_FIELD_NUMBER: _ClassVar[int]
    PROBABILITY_BEATS_CONTROL_FIELD_NUMBER: _ClassVar[int]
    VERDICT_FIELD_NUMBER: _ClassVar[int]
    successes: int
    trials: int
    rate_percent: float
    probability_beats_control: float
    verdict: ExperimentVerdict
    def __init__(self, successes: _Optional[int] = ..., trials: _Optional[int] = ..., rate_percent: _Optional[float] = ..., probability_beats_control: _Optional[float] = ..., verdict: _Optional[_Union[ExperimentVerdict, str]] = ...) -> None: ...

class ExperimentArm(_message.Message):
    __slots__ = ("variant_slug", "variant_name", "weight", "is_control", "cta_click", "paid")
    VARIANT_SLUG_FIELD_NUMBER: _ClassVar[int]
    VARIANT_NAME_FIELD_NUMBER: _ClassVar[int]
    WEIGHT_FIELD_NUMBER: _ClassVar[int]
    IS_CONTROL_FIELD_NUMBER: _ClassVar[int]
    CTA_CLICK_FIELD_NUMBER: _ClassVar[int]
    PAID_FIELD_NUMBER: _ClassVar[int]
    variant_slug: str
    variant_name: str
    weight: int
    is_control: bool
    cta_click: ArmMetric
    paid: ArmMetric
    def __init__(self, variant_slug: _Optional[str] = ..., variant_name: _Optional[str] = ..., weight: _Optional[int] = ..., is_control: _Optional[bool] = ..., cta_click: _Optional[_Union[ArmMetric, _Mapping]] = ..., paid: _Optional[_Union[ArmMetric, _Mapping]] = ...) -> None: ...

class Experiment(_message.Message):
    __slots__ = ("arms", "control_slug", "minimum_trials", "minimum_successes", "weight_fingerprint", "fingerprint_since")
    ARMS_FIELD_NUMBER: _ClassVar[int]
    CONTROL_SLUG_FIELD_NUMBER: _ClassVar[int]
    MINIMUM_TRIALS_FIELD_NUMBER: _ClassVar[int]
    MINIMUM_SUCCESSES_FIELD_NUMBER: _ClassVar[int]
    WEIGHT_FINGERPRINT_FIELD_NUMBER: _ClassVar[int]
    FINGERPRINT_SINCE_FIELD_NUMBER: _ClassVar[int]
    arms: _containers.RepeatedCompositeFieldContainer[ExperimentArm]
    control_slug: str
    minimum_trials: int
    minimum_successes: int
    weight_fingerprint: str
    fingerprint_since: _timestamp_pb2.Timestamp
    def __init__(self, arms: _Optional[_Iterable[_Union[ExperimentArm, _Mapping]]] = ..., control_slug: _Optional[str] = ..., minimum_trials: _Optional[int] = ..., minimum_successes: _Optional[int] = ..., weight_fingerprint: _Optional[str] = ..., fingerprint_since: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ...) -> None: ...

class AppActivity(_message.Message):
    __slots__ = ("bundle_key", "app_key", "app_name", "downloads", "downloads_by_platform", "update_checks", "update_downloads")
    class DownloadsByPlatformEntry(_message.Message):
        __slots__ = ("key", "value")
        KEY_FIELD_NUMBER: _ClassVar[int]
        VALUE_FIELD_NUMBER: _ClassVar[int]
        key: str
        value: int
        def __init__(self, key: _Optional[str] = ..., value: _Optional[int] = ...) -> None: ...
    BUNDLE_KEY_FIELD_NUMBER: _ClassVar[int]
    APP_KEY_FIELD_NUMBER: _ClassVar[int]
    APP_NAME_FIELD_NUMBER: _ClassVar[int]
    DOWNLOADS_FIELD_NUMBER: _ClassVar[int]
    DOWNLOADS_BY_PLATFORM_FIELD_NUMBER: _ClassVar[int]
    UPDATE_CHECKS_FIELD_NUMBER: _ClassVar[int]
    UPDATE_DOWNLOADS_FIELD_NUMBER: _ClassVar[int]
    bundle_key: str
    app_key: str
    app_name: str
    downloads: int
    downloads_by_platform: _containers.ScalarMap[str, int]
    update_checks: int
    update_downloads: int
    def __init__(self, bundle_key: _Optional[str] = ..., app_key: _Optional[str] = ..., app_name: _Optional[str] = ..., downloads: _Optional[int] = ..., downloads_by_platform: _Optional[_Mapping[str, int]] = ..., update_checks: _Optional[int] = ..., update_downloads: _Optional[int] = ...) -> None: ...

class CreditUsageRow(_message.Message):
    __slots__ = ("key", "label", "credits", "operations")
    KEY_FIELD_NUMBER: _ClassVar[int]
    LABEL_FIELD_NUMBER: _ClassVar[int]
    CREDITS_FIELD_NUMBER: _ClassVar[int]
    OPERATIONS_FIELD_NUMBER: _ClassVar[int]
    key: str
    label: str
    credits: int
    operations: int
    def __init__(self, key: _Optional[str] = ..., label: _Optional[str] = ..., credits: _Optional[int] = ..., operations: _Optional[int] = ...) -> None: ...

class CreditEconomy(_message.Message):
    __slots__ = ("credits_burned", "credits_purchased", "operations", "distinct_consumers", "by_app", "by_model")
    CREDITS_BURNED_FIELD_NUMBER: _ClassVar[int]
    CREDITS_PURCHASED_FIELD_NUMBER: _ClassVar[int]
    OPERATIONS_FIELD_NUMBER: _ClassVar[int]
    DISTINCT_CONSUMERS_FIELD_NUMBER: _ClassVar[int]
    BY_APP_FIELD_NUMBER: _ClassVar[int]
    BY_MODEL_FIELD_NUMBER: _ClassVar[int]
    credits_burned: int
    credits_purchased: int
    operations: int
    distinct_consumers: int
    by_app: _containers.RepeatedCompositeFieldContainer[CreditUsageRow]
    by_model: _containers.RepeatedCompositeFieldContainer[CreditUsageRow]
    def __init__(self, credits_burned: _Optional[int] = ..., credits_purchased: _Optional[int] = ..., operations: _Optional[int] = ..., distinct_consumers: _Optional[int] = ..., by_app: _Optional[_Iterable[_Union[CreditUsageRow, _Mapping]]] = ..., by_model: _Optional[_Iterable[_Union[CreditUsageRow, _Mapping]]] = ...) -> None: ...

class Growth(_message.Message):
    __slots__ = ("signups", "waitlist_joins", "new_paid_subscriptions", "trials_started")
    SIGNUPS_FIELD_NUMBER: _ClassVar[int]
    WAITLIST_JOINS_FIELD_NUMBER: _ClassVar[int]
    NEW_PAID_SUBSCRIPTIONS_FIELD_NUMBER: _ClassVar[int]
    TRIALS_STARTED_FIELD_NUMBER: _ClassVar[int]
    signups: int
    waitlist_joins: int
    new_paid_subscriptions: int
    trials_started: int
    def __init__(self, signups: _Optional[int] = ..., waitlist_joins: _Optional[int] = ..., new_paid_subscriptions: _Optional[int] = ..., trials_started: _Optional[int] = ...) -> None: ...

class BusinessDigest(_message.Message):
    __slots__ = ("contract_version", "observed_at", "window", "funnel", "retention", "experiment", "apps", "credits", "growth", "exclusions", "cta_clicks")
    CONTRACT_VERSION_FIELD_NUMBER: _ClassVar[int]
    OBSERVED_AT_FIELD_NUMBER: _ClassVar[int]
    WINDOW_FIELD_NUMBER: _ClassVar[int]
    FUNNEL_FIELD_NUMBER: _ClassVar[int]
    RETENTION_FIELD_NUMBER: _ClassVar[int]
    EXPERIMENT_FIELD_NUMBER: _ClassVar[int]
    APPS_FIELD_NUMBER: _ClassVar[int]
    CREDITS_FIELD_NUMBER: _ClassVar[int]
    GROWTH_FIELD_NUMBER: _ClassVar[int]
    EXCLUSIONS_FIELD_NUMBER: _ClassVar[int]
    CTA_CLICKS_FIELD_NUMBER: _ClassVar[int]
    contract_version: str
    observed_at: _timestamp_pb2.Timestamp
    window: DigestWindow
    funnel: Funnel
    retention: Retention
    experiment: Experiment
    apps: _containers.RepeatedCompositeFieldContainer[AppActivity]
    credits: CreditEconomy
    growth: Growth
    exclusions: _shared_pb2.TrafficExclusions
    cta_clicks: int
    def __init__(self, contract_version: _Optional[str] = ..., observed_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., window: _Optional[_Union[DigestWindow, _Mapping]] = ..., funnel: _Optional[_Union[Funnel, _Mapping]] = ..., retention: _Optional[_Union[Retention, _Mapping]] = ..., experiment: _Optional[_Union[Experiment, _Mapping]] = ..., apps: _Optional[_Iterable[_Union[AppActivity, _Mapping]]] = ..., credits: _Optional[_Union[CreditEconomy, _Mapping]] = ..., growth: _Optional[_Union[Growth, _Mapping]] = ..., exclusions: _Optional[_Union[_shared_pb2.TrafficExclusions, _Mapping]] = ..., cta_clicks: _Optional[int] = ...) -> None: ...
