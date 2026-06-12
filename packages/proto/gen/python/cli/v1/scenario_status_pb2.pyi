from cli.v1 import common_pb2 as _common_pb2
from cli.v1 import scenario_list_pb2 as _scenario_list_pb2
from google.protobuf import struct_pb2 as _struct_pb2
from google.protobuf.internal import containers as _containers
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Iterable as _Iterable, Mapping as _Mapping
from typing import ClassVar as _ClassVar, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class ScenarioStatusListResponse(_message.Message):
    __slots__ = ("success", "summary", "scenarios", "discovery_failures")
    SUCCESS_FIELD_NUMBER: _ClassVar[int]
    SUMMARY_FIELD_NUMBER: _ClassVar[int]
    SCENARIOS_FIELD_NUMBER: _ClassVar[int]
    DISCOVERY_FAILURES_FIELD_NUMBER: _ClassVar[int]
    success: bool
    summary: ScenarioStatusSummary
    scenarios: _containers.RepeatedCompositeFieldContainer[ScenarioStatusItem]
    discovery_failures: _containers.RepeatedCompositeFieldContainer[_common_pb2.DiscoveryFailure]
    def __init__(self, success: _Optional[bool] = ..., summary: _Optional[_Union[ScenarioStatusSummary, _Mapping]] = ..., scenarios: _Optional[_Iterable[_Union[ScenarioStatusItem, _Mapping]]] = ..., discovery_failures: _Optional[_Iterable[_Union[_common_pb2.DiscoveryFailure, _Mapping]]] = ...) -> None: ...

class ScenarioStatusSummary(_message.Message):
    __slots__ = ("total_scenarios", "running", "stopped")
    TOTAL_SCENARIOS_FIELD_NUMBER: _ClassVar[int]
    RUNNING_FIELD_NUMBER: _ClassVar[int]
    STOPPED_FIELD_NUMBER: _ClassVar[int]
    total_scenarios: int
    running: int
    stopped: int
    def __init__(self, total_scenarios: _Optional[int] = ..., running: _Optional[int] = ..., stopped: _Optional[int] = ...) -> None: ...

class ScenarioStatusItem(_message.Message):
    __slots__ = ("name", "display_name", "description", "tags", "status", "processes", "runtime", "started_at", "ports", "port_bindings", "health_status", "health_error")
    class PortsEntry(_message.Message):
        __slots__ = ("key", "value")
        KEY_FIELD_NUMBER: _ClassVar[int]
        VALUE_FIELD_NUMBER: _ClassVar[int]
        key: str
        value: int
        def __init__(self, key: _Optional[str] = ..., value: _Optional[int] = ...) -> None: ...
    NAME_FIELD_NUMBER: _ClassVar[int]
    DISPLAY_NAME_FIELD_NUMBER: _ClassVar[int]
    DESCRIPTION_FIELD_NUMBER: _ClassVar[int]
    TAGS_FIELD_NUMBER: _ClassVar[int]
    STATUS_FIELD_NUMBER: _ClassVar[int]
    PROCESSES_FIELD_NUMBER: _ClassVar[int]
    RUNTIME_FIELD_NUMBER: _ClassVar[int]
    STARTED_AT_FIELD_NUMBER: _ClassVar[int]
    PORTS_FIELD_NUMBER: _ClassVar[int]
    PORT_BINDINGS_FIELD_NUMBER: _ClassVar[int]
    HEALTH_STATUS_FIELD_NUMBER: _ClassVar[int]
    HEALTH_ERROR_FIELD_NUMBER: _ClassVar[int]
    name: str
    display_name: str
    description: str
    tags: _containers.RepeatedScalarFieldContainer[str]
    status: str
    processes: int
    runtime: str
    started_at: str
    ports: _containers.ScalarMap[str, int]
    port_bindings: _containers.RepeatedCompositeFieldContainer[_scenario_list_pb2.ScenarioPort]
    health_status: _struct_pb2.Value
    health_error: str
    def __init__(self, name: _Optional[str] = ..., display_name: _Optional[str] = ..., description: _Optional[str] = ..., tags: _Optional[_Iterable[str]] = ..., status: _Optional[str] = ..., processes: _Optional[int] = ..., runtime: _Optional[str] = ..., started_at: _Optional[str] = ..., ports: _Optional[_Mapping[str, int]] = ..., port_bindings: _Optional[_Iterable[_Union[_scenario_list_pb2.ScenarioPort, _Mapping]]] = ..., health_status: _Optional[_Union[_struct_pb2.Value, _Mapping]] = ..., health_error: _Optional[str] = ...) -> None: ...

class ScenarioStatusSingle(_message.Message):
    __slots__ = ("success", "scenario", "info", "runtime")
    SUCCESS_FIELD_NUMBER: _ClassVar[int]
    SCENARIO_FIELD_NUMBER: _ClassVar[int]
    INFO_FIELD_NUMBER: _ClassVar[int]
    RUNTIME_FIELD_NUMBER: _ClassVar[int]
    success: bool
    scenario: ScenarioStatusItem
    info: ScenarioInfoData
    runtime: ScenarioRuntimeData
    def __init__(self, success: _Optional[bool] = ..., scenario: _Optional[_Union[ScenarioStatusItem, _Mapping]] = ..., info: _Optional[_Union[ScenarioInfoData, _Mapping]] = ..., runtime: _Optional[_Union[ScenarioRuntimeData, _Mapping]] = ...) -> None: ...

class ScenarioInfoResponse(_message.Message):
    __slots__ = ("success", "scenario", "runtime")
    SUCCESS_FIELD_NUMBER: _ClassVar[int]
    SCENARIO_FIELD_NUMBER: _ClassVar[int]
    RUNTIME_FIELD_NUMBER: _ClassVar[int]
    success: bool
    scenario: ScenarioInfoData
    runtime: ScenarioRuntimeData
    def __init__(self, success: _Optional[bool] = ..., scenario: _Optional[_Union[ScenarioInfoData, _Mapping]] = ..., runtime: _Optional[_Union[ScenarioRuntimeData, _Mapping]] = ...) -> None: ...

class ScenarioInfoData(_message.Message):
    __slots__ = ("name", "display_name", "description", "version", "type", "category", "tags", "path", "service_path", "sandbox_redirected", "config_version", "lifecycle_version", "ports", "phases", "generation", "template_drifted")
    NAME_FIELD_NUMBER: _ClassVar[int]
    DISPLAY_NAME_FIELD_NUMBER: _ClassVar[int]
    DESCRIPTION_FIELD_NUMBER: _ClassVar[int]
    VERSION_FIELD_NUMBER: _ClassVar[int]
    TYPE_FIELD_NUMBER: _ClassVar[int]
    CATEGORY_FIELD_NUMBER: _ClassVar[int]
    TAGS_FIELD_NUMBER: _ClassVar[int]
    PATH_FIELD_NUMBER: _ClassVar[int]
    SERVICE_PATH_FIELD_NUMBER: _ClassVar[int]
    SANDBOX_REDIRECTED_FIELD_NUMBER: _ClassVar[int]
    CONFIG_VERSION_FIELD_NUMBER: _ClassVar[int]
    LIFECYCLE_VERSION_FIELD_NUMBER: _ClassVar[int]
    PORTS_FIELD_NUMBER: _ClassVar[int]
    PHASES_FIELD_NUMBER: _ClassVar[int]
    GENERATION_FIELD_NUMBER: _ClassVar[int]
    TEMPLATE_DRIFTED_FIELD_NUMBER: _ClassVar[int]
    name: str
    display_name: str
    description: str
    version: str
    type: str
    category: str
    tags: _containers.RepeatedScalarFieldContainer[str]
    path: str
    service_path: str
    sandbox_redirected: bool
    config_version: str
    lifecycle_version: str
    ports: _containers.RepeatedCompositeFieldContainer[ScenarioInfoPortSummary]
    phases: _containers.RepeatedCompositeFieldContainer[ScenarioInfoPhaseSummary]
    generation: ScenarioGenerationMetadata
    template_drifted: bool
    def __init__(self, name: _Optional[str] = ..., display_name: _Optional[str] = ..., description: _Optional[str] = ..., version: _Optional[str] = ..., type: _Optional[str] = ..., category: _Optional[str] = ..., tags: _Optional[_Iterable[str]] = ..., path: _Optional[str] = ..., service_path: _Optional[str] = ..., sandbox_redirected: _Optional[bool] = ..., config_version: _Optional[str] = ..., lifecycle_version: _Optional[str] = ..., ports: _Optional[_Iterable[_Union[ScenarioInfoPortSummary, _Mapping]]] = ..., phases: _Optional[_Iterable[_Union[ScenarioInfoPhaseSummary, _Mapping]]] = ..., generation: _Optional[_Union[ScenarioGenerationMetadata, _Mapping]] = ..., template_drifted: _Optional[bool] = ...) -> None: ...

class ScenarioInfoPortSummary(_message.Message):
    __slots__ = ("name", "env_var", "description", "range", "fixed_port")
    NAME_FIELD_NUMBER: _ClassVar[int]
    ENV_VAR_FIELD_NUMBER: _ClassVar[int]
    DESCRIPTION_FIELD_NUMBER: _ClassVar[int]
    RANGE_FIELD_NUMBER: _ClassVar[int]
    FIXED_PORT_FIELD_NUMBER: _ClassVar[int]
    name: str
    env_var: str
    description: str
    range: str
    fixed_port: int
    def __init__(self, name: _Optional[str] = ..., env_var: _Optional[str] = ..., description: _Optional[str] = ..., range: _Optional[str] = ..., fixed_port: _Optional[int] = ...) -> None: ...

class ScenarioInfoPhaseSummary(_message.Message):
    __slots__ = ("name", "description", "steps", "defined")
    NAME_FIELD_NUMBER: _ClassVar[int]
    DESCRIPTION_FIELD_NUMBER: _ClassVar[int]
    STEPS_FIELD_NUMBER: _ClassVar[int]
    DEFINED_FIELD_NUMBER: _ClassVar[int]
    name: str
    description: str
    steps: int
    defined: bool
    def __init__(self, name: _Optional[str] = ..., description: _Optional[str] = ..., steps: _Optional[int] = ..., defined: _Optional[bool] = ...) -> None: ...

class ScenarioGenerationMetadata(_message.Message):
    __slots__ = ("template", "generated_at", "design", "manifest_sha", "content_sha")
    TEMPLATE_FIELD_NUMBER: _ClassVar[int]
    GENERATED_AT_FIELD_NUMBER: _ClassVar[int]
    DESIGN_FIELD_NUMBER: _ClassVar[int]
    MANIFEST_SHA_FIELD_NUMBER: _ClassVar[int]
    CONTENT_SHA_FIELD_NUMBER: _ClassVar[int]
    template: ScenarioGenerationTemplate
    generated_at: str
    design: ScenarioGenerationDesign
    manifest_sha: str
    content_sha: str
    def __init__(self, template: _Optional[_Union[ScenarioGenerationTemplate, _Mapping]] = ..., generated_at: _Optional[str] = ..., design: _Optional[_Union[ScenarioGenerationDesign, _Mapping]] = ..., manifest_sha: _Optional[str] = ..., content_sha: _Optional[str] = ...) -> None: ...

class ScenarioGenerationTemplate(_message.Message):
    __slots__ = ("id", "version")
    ID_FIELD_NUMBER: _ClassVar[int]
    VERSION_FIELD_NUMBER: _ClassVar[int]
    id: str
    version: str
    def __init__(self, id: _Optional[str] = ..., version: _Optional[str] = ...) -> None: ...

class ScenarioGenerationDesign(_message.Message):
    __slots__ = ("id", "version", "adapter")
    ID_FIELD_NUMBER: _ClassVar[int]
    VERSION_FIELD_NUMBER: _ClassVar[int]
    ADAPTER_FIELD_NUMBER: _ClassVar[int]
    id: str
    version: str
    adapter: str
    def __init__(self, id: _Optional[str] = ..., version: _Optional[str] = ..., adapter: _Optional[str] = ...) -> None: ...

class ScenarioRuntimeData(_message.Message):
    __slots__ = ("status", "processes", "runtime", "started_at", "ports", "process_records", "list_ports", "health_error")
    class PortsEntry(_message.Message):
        __slots__ = ("key", "value")
        KEY_FIELD_NUMBER: _ClassVar[int]
        VALUE_FIELD_NUMBER: _ClassVar[int]
        key: str
        value: int
        def __init__(self, key: _Optional[str] = ..., value: _Optional[int] = ...) -> None: ...
    STATUS_FIELD_NUMBER: _ClassVar[int]
    PROCESSES_FIELD_NUMBER: _ClassVar[int]
    RUNTIME_FIELD_NUMBER: _ClassVar[int]
    STARTED_AT_FIELD_NUMBER: _ClassVar[int]
    PORTS_FIELD_NUMBER: _ClassVar[int]
    PROCESS_RECORDS_FIELD_NUMBER: _ClassVar[int]
    LIST_PORTS_FIELD_NUMBER: _ClassVar[int]
    HEALTH_ERROR_FIELD_NUMBER: _ClassVar[int]
    status: str
    processes: int
    runtime: str
    started_at: str
    ports: _containers.ScalarMap[str, int]
    process_records: _containers.RepeatedCompositeFieldContainer[ScenarioProcessRecord]
    list_ports: _containers.RepeatedCompositeFieldContainer[_scenario_list_pb2.ScenarioPort]
    health_error: str
    def __init__(self, status: _Optional[str] = ..., processes: _Optional[int] = ..., runtime: _Optional[str] = ..., started_at: _Optional[str] = ..., ports: _Optional[_Mapping[str, int]] = ..., process_records: _Optional[_Iterable[_Union[ScenarioProcessRecord, _Mapping]]] = ..., list_ports: _Optional[_Iterable[_Union[_scenario_list_pb2.ScenarioPort, _Mapping]]] = ..., health_error: _Optional[str] = ...) -> None: ...

class ScenarioProcessRecord(_message.Message):
    __slots__ = ("pid", "pgid", "process_id", "phase", "scenario", "step", "command", "working_dir", "log_file", "port", "started_at", "status")
    PID_FIELD_NUMBER: _ClassVar[int]
    PGID_FIELD_NUMBER: _ClassVar[int]
    PROCESS_ID_FIELD_NUMBER: _ClassVar[int]
    PHASE_FIELD_NUMBER: _ClassVar[int]
    SCENARIO_FIELD_NUMBER: _ClassVar[int]
    STEP_FIELD_NUMBER: _ClassVar[int]
    COMMAND_FIELD_NUMBER: _ClassVar[int]
    WORKING_DIR_FIELD_NUMBER: _ClassVar[int]
    LOG_FILE_FIELD_NUMBER: _ClassVar[int]
    PORT_FIELD_NUMBER: _ClassVar[int]
    STARTED_AT_FIELD_NUMBER: _ClassVar[int]
    STATUS_FIELD_NUMBER: _ClassVar[int]
    pid: int
    pgid: int
    process_id: str
    phase: str
    scenario: str
    step: str
    command: str
    working_dir: str
    log_file: str
    port: int
    started_at: str
    status: str
    def __init__(self, pid: _Optional[int] = ..., pgid: _Optional[int] = ..., process_id: _Optional[str] = ..., phase: _Optional[str] = ..., scenario: _Optional[str] = ..., step: _Optional[str] = ..., command: _Optional[str] = ..., working_dir: _Optional[str] = ..., log_file: _Optional[str] = ..., port: _Optional[int] = ..., started_at: _Optional[str] = ..., status: _Optional[str] = ...) -> None: ...

class ScenarioPortSingle(_message.Message):
    __slots__ = ("success", "scenario", "port_name", "step", "port", "error")
    SUCCESS_FIELD_NUMBER: _ClassVar[int]
    SCENARIO_FIELD_NUMBER: _ClassVar[int]
    PORT_NAME_FIELD_NUMBER: _ClassVar[int]
    STEP_FIELD_NUMBER: _ClassVar[int]
    PORT_FIELD_NUMBER: _ClassVar[int]
    ERROR_FIELD_NUMBER: _ClassVar[int]
    success: bool
    scenario: str
    port_name: str
    step: str
    port: int
    error: str
    def __init__(self, success: _Optional[bool] = ..., scenario: _Optional[str] = ..., port_name: _Optional[str] = ..., step: _Optional[str] = ..., port: _Optional[int] = ..., error: _Optional[str] = ...) -> None: ...

class ScenarioPortList(_message.Message):
    __slots__ = ("success", "scenario", "ports", "metadata", "error")
    class MetadataEntry(_message.Message):
        __slots__ = ("key", "value")
        KEY_FIELD_NUMBER: _ClassVar[int]
        VALUE_FIELD_NUMBER: _ClassVar[int]
        key: str
        value: int
        def __init__(self, key: _Optional[str] = ..., value: _Optional[int] = ...) -> None: ...
    SUCCESS_FIELD_NUMBER: _ClassVar[int]
    SCENARIO_FIELD_NUMBER: _ClassVar[int]
    PORTS_FIELD_NUMBER: _ClassVar[int]
    METADATA_FIELD_NUMBER: _ClassVar[int]
    ERROR_FIELD_NUMBER: _ClassVar[int]
    success: bool
    scenario: str
    ports: _containers.RepeatedCompositeFieldContainer[_scenario_list_pb2.ScenarioPort]
    metadata: _containers.ScalarMap[str, int]
    error: str
    def __init__(self, success: _Optional[bool] = ..., scenario: _Optional[str] = ..., ports: _Optional[_Iterable[_Union[_scenario_list_pb2.ScenarioPort, _Mapping]]] = ..., metadata: _Optional[_Mapping[str, int]] = ..., error: _Optional[str] = ...) -> None: ...

class ScenarioSetupResponse(_message.Message):
    __slots__ = ("success", "phase", "status", "defined", "steps")
    SUCCESS_FIELD_NUMBER: _ClassVar[int]
    PHASE_FIELD_NUMBER: _ClassVar[int]
    STATUS_FIELD_NUMBER: _ClassVar[int]
    DEFINED_FIELD_NUMBER: _ClassVar[int]
    STEPS_FIELD_NUMBER: _ClassVar[int]
    success: bool
    phase: str
    status: str
    defined: bool
    steps: ScenarioSetupSteps
    def __init__(self, success: _Optional[bool] = ..., phase: _Optional[str] = ..., status: _Optional[str] = ..., defined: _Optional[bool] = ..., steps: _Optional[_Union[ScenarioSetupSteps, _Mapping]] = ...) -> None: ...

class ScenarioSetupSteps(_message.Message):
    __slots__ = ("executed", "skipped")
    EXECUTED_FIELD_NUMBER: _ClassVar[int]
    SKIPPED_FIELD_NUMBER: _ClassVar[int]
    executed: int
    skipped: int
    def __init__(self, executed: _Optional[int] = ..., skipped: _Optional[int] = ...) -> None: ...

class ScenarioBatchResponse(_message.Message):
    __slots__ = ("success", "data")
    SUCCESS_FIELD_NUMBER: _ClassVar[int]
    DATA_FIELD_NUMBER: _ClassVar[int]
    success: bool
    data: ScenarioBatchData
    def __init__(self, success: _Optional[bool] = ..., data: _Optional[_Union[ScenarioBatchData, _Mapping]] = ...) -> None: ...

class ScenarioBatchData(_message.Message):
    __slots__ = ("started", "stopped", "failed")
    STARTED_FIELD_NUMBER: _ClassVar[int]
    STOPPED_FIELD_NUMBER: _ClassVar[int]
    FAILED_FIELD_NUMBER: _ClassVar[int]
    started: _containers.RepeatedCompositeFieldContainer[ScenarioLifecycleItem]
    stopped: _containers.RepeatedScalarFieldContainer[str]
    failed: _containers.RepeatedCompositeFieldContainer[ScenarioBatchFailure]
    def __init__(self, started: _Optional[_Iterable[_Union[ScenarioLifecycleItem, _Mapping]]] = ..., stopped: _Optional[_Iterable[str]] = ..., failed: _Optional[_Iterable[_Union[ScenarioBatchFailure, _Mapping]]] = ...) -> None: ...

class ScenarioBatchFailure(_message.Message):
    __slots__ = ("name", "error")
    NAME_FIELD_NUMBER: _ClassVar[int]
    ERROR_FIELD_NUMBER: _ClassVar[int]
    name: str
    error: str
    def __init__(self, name: _Optional[str] = ..., error: _Optional[str] = ...) -> None: ...

class ScenarioLifecycleResponse(_message.Message):
    __slots__ = ("success", "scenarios")
    SUCCESS_FIELD_NUMBER: _ClassVar[int]
    SCENARIOS_FIELD_NUMBER: _ClassVar[int]
    success: bool
    scenarios: _containers.RepeatedCompositeFieldContainer[ScenarioLifecycleItem]
    def __init__(self, success: _Optional[bool] = ..., scenarios: _Optional[_Iterable[_Union[ScenarioLifecycleItem, _Mapping]]] = ...) -> None: ...

class ScenarioLifecycleItem(_message.Message):
    __slots__ = ("name", "status", "health", "ports", "endpoints", "failed_dependencies", "failed_resources")
    class PortsEntry(_message.Message):
        __slots__ = ("key", "value")
        KEY_FIELD_NUMBER: _ClassVar[int]
        VALUE_FIELD_NUMBER: _ClassVar[int]
        key: str
        value: int
        def __init__(self, key: _Optional[str] = ..., value: _Optional[int] = ...) -> None: ...
    NAME_FIELD_NUMBER: _ClassVar[int]
    STATUS_FIELD_NUMBER: _ClassVar[int]
    HEALTH_FIELD_NUMBER: _ClassVar[int]
    PORTS_FIELD_NUMBER: _ClassVar[int]
    ENDPOINTS_FIELD_NUMBER: _ClassVar[int]
    FAILED_DEPENDENCIES_FIELD_NUMBER: _ClassVar[int]
    FAILED_RESOURCES_FIELD_NUMBER: _ClassVar[int]
    name: str
    status: str
    health: str
    ports: _containers.ScalarMap[str, int]
    endpoints: _containers.RepeatedCompositeFieldContainer[ScenarioEndpoint]
    failed_dependencies: _containers.RepeatedScalarFieldContainer[str]
    failed_resources: _containers.RepeatedScalarFieldContainer[str]
    def __init__(self, name: _Optional[str] = ..., status: _Optional[str] = ..., health: _Optional[str] = ..., ports: _Optional[_Mapping[str, int]] = ..., endpoints: _Optional[_Iterable[_Union[ScenarioEndpoint, _Mapping]]] = ..., failed_dependencies: _Optional[_Iterable[str]] = ..., failed_resources: _Optional[_Iterable[str]] = ...) -> None: ...

class ScenarioEndpoint(_message.Message):
    __slots__ = ("name", "key", "description", "port", "url")
    NAME_FIELD_NUMBER: _ClassVar[int]
    KEY_FIELD_NUMBER: _ClassVar[int]
    DESCRIPTION_FIELD_NUMBER: _ClassVar[int]
    PORT_FIELD_NUMBER: _ClassVar[int]
    URL_FIELD_NUMBER: _ClassVar[int]
    name: str
    key: str
    description: str
    port: int
    url: str
    def __init__(self, name: _Optional[str] = ..., key: _Optional[str] = ..., description: _Optional[str] = ..., port: _Optional[int] = ..., url: _Optional[str] = ...) -> None: ...

class ScenarioEnvValidationResponse(_message.Message):
    __slots__ = ("success", "report")
    SUCCESS_FIELD_NUMBER: _ClassVar[int]
    REPORT_FIELD_NUMBER: _ClassVar[int]
    success: bool
    report: ScenarioEnvValidationReport
    def __init__(self, success: _Optional[bool] = ..., report: _Optional[_Union[ScenarioEnvValidationReport, _Mapping]] = ...) -> None: ...

class ScenarioEnvValidationReport(_message.Message):
    __slots__ = ("scenario", "values", "issues", "resource_reports", "passed")
    class ValuesEntry(_message.Message):
        __slots__ = ("key", "value")
        KEY_FIELD_NUMBER: _ClassVar[int]
        VALUE_FIELD_NUMBER: _ClassVar[int]
        key: str
        value: str
        def __init__(self, key: _Optional[str] = ..., value: _Optional[str] = ...) -> None: ...
    SCENARIO_FIELD_NUMBER: _ClassVar[int]
    VALUES_FIELD_NUMBER: _ClassVar[int]
    ISSUES_FIELD_NUMBER: _ClassVar[int]
    RESOURCE_REPORTS_FIELD_NUMBER: _ClassVar[int]
    PASSED_FIELD_NUMBER: _ClassVar[int]
    scenario: str
    values: _containers.ScalarMap[str, str]
    issues: _containers.RepeatedCompositeFieldContainer[ScenarioValidationIssue]
    resource_reports: _containers.RepeatedCompositeFieldContainer[ScenarioResourceReport]
    passed: bool
    def __init__(self, scenario: _Optional[str] = ..., values: _Optional[_Mapping[str, str]] = ..., issues: _Optional[_Iterable[_Union[ScenarioValidationIssue, _Mapping]]] = ..., resource_reports: _Optional[_Iterable[_Union[ScenarioResourceReport, _Mapping]]] = ..., passed: _Optional[bool] = ...) -> None: ...

class ScenarioValidationIssue(_message.Message):
    __slots__ = ("severity", "message")
    SEVERITY_FIELD_NUMBER: _ClassVar[int]
    MESSAGE_FIELD_NUMBER: _ClassVar[int]
    severity: str
    message: str
    def __init__(self, severity: _Optional[str] = ..., message: _Optional[str] = ...) -> None: ...

class ScenarioResourceReport(_message.Message):
    __slots__ = ("name", "manifest_path", "values", "warnings")
    class ValuesEntry(_message.Message):
        __slots__ = ("key", "value")
        KEY_FIELD_NUMBER: _ClassVar[int]
        VALUE_FIELD_NUMBER: _ClassVar[int]
        key: str
        value: str
        def __init__(self, key: _Optional[str] = ..., value: _Optional[str] = ...) -> None: ...
    NAME_FIELD_NUMBER: _ClassVar[int]
    MANIFEST_PATH_FIELD_NUMBER: _ClassVar[int]
    VALUES_FIELD_NUMBER: _ClassVar[int]
    WARNINGS_FIELD_NUMBER: _ClassVar[int]
    name: str
    manifest_path: str
    values: _containers.ScalarMap[str, str]
    warnings: _containers.RepeatedScalarFieldContainer[str]
    def __init__(self, name: _Optional[str] = ..., manifest_path: _Optional[str] = ..., values: _Optional[_Mapping[str, str]] = ..., warnings: _Optional[_Iterable[str]] = ...) -> None: ...
