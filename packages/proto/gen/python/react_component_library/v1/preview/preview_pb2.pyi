from google.protobuf import struct_pb2 as _struct_pb2
from google.protobuf.internal import containers as _containers
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Iterable as _Iterable, Mapping as _Mapping
from typing import ClassVar as _ClassVar, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class GetPreviewBundleRequest(_message.Message):
    __slots__ = ("id",)
    ID_FIELD_NUMBER: _ClassVar[int]
    id: str
    def __init__(self, id: _Optional[str] = ...) -> None: ...

class GetPreviewBundleResponse(_message.Message):
    __slots__ = ("js", "source_path", "sha256", "warnings")
    JS_FIELD_NUMBER: _ClassVar[int]
    SOURCE_PATH_FIELD_NUMBER: _ClassVar[int]
    SHA256_FIELD_NUMBER: _ClassVar[int]
    WARNINGS_FIELD_NUMBER: _ClassVar[int]
    js: str
    source_path: str
    sha256: str
    warnings: _containers.RepeatedScalarFieldContainer[str]
    def __init__(self, js: _Optional[str] = ..., source_path: _Optional[str] = ..., sha256: _Optional[str] = ..., warnings: _Optional[_Iterable[str]] = ...) -> None: ...

class CompositionAsset(_message.Message):
    __slots__ = ("catalog_id", "version", "export")
    CATALOG_ID_FIELD_NUMBER: _ClassVar[int]
    VERSION_FIELD_NUMBER: _ClassVar[int]
    EXPORT_FIELD_NUMBER: _ClassVar[int]
    catalog_id: str
    version: str
    export: str
    def __init__(self, catalog_id: _Optional[str] = ..., version: _Optional[str] = ..., export: _Optional[str] = ...) -> None: ...

class CompositionRegion(_message.Message):
    __slots__ = ("id", "asset", "slot", "required", "parent")
    ID_FIELD_NUMBER: _ClassVar[int]
    ASSET_FIELD_NUMBER: _ClassVar[int]
    SLOT_FIELD_NUMBER: _ClassVar[int]
    REQUIRED_FIELD_NUMBER: _ClassVar[int]
    PARENT_FIELD_NUMBER: _ClassVar[int]
    id: str
    asset: CompositionAsset
    slot: _containers.RepeatedScalarFieldContainer[str]
    required: bool
    parent: str
    def __init__(self, id: _Optional[str] = ..., asset: _Optional[_Union[CompositionAsset, _Mapping]] = ..., slot: _Optional[_Iterable[str]] = ..., required: _Optional[bool] = ..., parent: _Optional[str] = ...) -> None: ...

class GetCompositionBundleRequest(_message.Message):
    __slots__ = ("revision", "template", "regions")
    REVISION_FIELD_NUMBER: _ClassVar[int]
    TEMPLATE_FIELD_NUMBER: _ClassVar[int]
    REGIONS_FIELD_NUMBER: _ClassVar[int]
    revision: str
    template: CompositionAsset
    regions: _containers.RepeatedCompositeFieldContainer[CompositionRegion]
    def __init__(self, revision: _Optional[str] = ..., template: _Optional[_Union[CompositionAsset, _Mapping]] = ..., regions: _Optional[_Iterable[_Union[CompositionRegion, _Mapping]]] = ...) -> None: ...

class CompositionGap(_message.Message):
    __slots__ = ("region", "code", "message", "required")
    REGION_FIELD_NUMBER: _ClassVar[int]
    CODE_FIELD_NUMBER: _ClassVar[int]
    MESSAGE_FIELD_NUMBER: _ClassVar[int]
    REQUIRED_FIELD_NUMBER: _ClassVar[int]
    region: str
    code: str
    message: str
    required: bool
    def __init__(self, region: _Optional[str] = ..., code: _Optional[str] = ..., message: _Optional[str] = ..., required: _Optional[bool] = ...) -> None: ...

class GetCompositionBundleResponse(_message.Message):
    __slots__ = ("revision", "source", "js", "sha256", "warnings", "gaps")
    REVISION_FIELD_NUMBER: _ClassVar[int]
    SOURCE_FIELD_NUMBER: _ClassVar[int]
    JS_FIELD_NUMBER: _ClassVar[int]
    SHA256_FIELD_NUMBER: _ClassVar[int]
    WARNINGS_FIELD_NUMBER: _ClassVar[int]
    GAPS_FIELD_NUMBER: _ClassVar[int]
    revision: str
    source: str
    js: str
    sha256: str
    warnings: _containers.RepeatedScalarFieldContainer[str]
    gaps: _containers.RepeatedCompositeFieldContainer[CompositionGap]
    def __init__(self, revision: _Optional[str] = ..., source: _Optional[str] = ..., js: _Optional[str] = ..., sha256: _Optional[str] = ..., warnings: _Optional[_Iterable[str]] = ..., gaps: _Optional[_Iterable[_Union[CompositionGap, _Mapping]]] = ...) -> None: ...

class CompositionFixtureBinding(_message.Message):
    __slots__ = ("target", "asset", "version", "state", "field", "prop")
    TARGET_FIELD_NUMBER: _ClassVar[int]
    ASSET_FIELD_NUMBER: _ClassVar[int]
    VERSION_FIELD_NUMBER: _ClassVar[int]
    STATE_FIELD_NUMBER: _ClassVar[int]
    FIELD_FIELD_NUMBER: _ClassVar[int]
    PROP_FIELD_NUMBER: _ClassVar[int]
    target: str
    asset: str
    version: str
    state: str
    field: str
    prop: _containers.RepeatedScalarFieldContainer[str]
    def __init__(self, target: _Optional[str] = ..., asset: _Optional[str] = ..., version: _Optional[str] = ..., state: _Optional[str] = ..., field: _Optional[str] = ..., prop: _Optional[_Iterable[str]] = ...) -> None: ...

class RenderCompositionRequest(_message.Message):
    __slots__ = ("composition", "bindings", "fixtures", "kit", "theme", "direction")
    COMPOSITION_FIELD_NUMBER: _ClassVar[int]
    BINDINGS_FIELD_NUMBER: _ClassVar[int]
    FIXTURES_FIELD_NUMBER: _ClassVar[int]
    KIT_FIELD_NUMBER: _ClassVar[int]
    THEME_FIELD_NUMBER: _ClassVar[int]
    DIRECTION_FIELD_NUMBER: _ClassVar[int]
    composition: GetCompositionBundleRequest
    bindings: _struct_pb2.Struct
    fixtures: _containers.RepeatedCompositeFieldContainer[CompositionFixtureBinding]
    kit: str
    theme: str
    direction: str
    def __init__(self, composition: _Optional[_Union[GetCompositionBundleRequest, _Mapping]] = ..., bindings: _Optional[_Union[_struct_pb2.Struct, _Mapping]] = ..., fixtures: _Optional[_Iterable[_Union[CompositionFixtureBinding, _Mapping]]] = ..., kit: _Optional[str] = ..., theme: _Optional[str] = ..., direction: _Optional[str] = ...) -> None: ...

class RenderCompositionResponse(_message.Message):
    __slots__ = ("bundle", "html", "render_hash", "target")
    BUNDLE_FIELD_NUMBER: _ClassVar[int]
    HTML_FIELD_NUMBER: _ClassVar[int]
    RENDER_HASH_FIELD_NUMBER: _ClassVar[int]
    TARGET_FIELD_NUMBER: _ClassVar[int]
    bundle: GetCompositionBundleResponse
    html: str
    render_hash: str
    target: CompositionRenderTarget
    def __init__(self, bundle: _Optional[_Union[GetCompositionBundleResponse, _Mapping]] = ..., html: _Optional[str] = ..., render_hash: _Optional[str] = ..., target: _Optional[_Union[CompositionRenderTarget, _Mapping]] = ...) -> None: ...

class CompositionRenderTarget(_message.Message):
    __slots__ = ("input_hashes", "revision", "render_hash", "html_sha256", "inputs_sha256", "kind", "kit", "theme", "direction")
    class InputHashesEntry(_message.Message):
        __slots__ = ("key", "value")
        KEY_FIELD_NUMBER: _ClassVar[int]
        VALUE_FIELD_NUMBER: _ClassVar[int]
        key: str
        value: str
        def __init__(self, key: _Optional[str] = ..., value: _Optional[str] = ...) -> None: ...
    INPUT_HASHES_FIELD_NUMBER: _ClassVar[int]
    REVISION_FIELD_NUMBER: _ClassVar[int]
    RENDER_HASH_FIELD_NUMBER: _ClassVar[int]
    HTML_SHA256_FIELD_NUMBER: _ClassVar[int]
    INPUTS_SHA256_FIELD_NUMBER: _ClassVar[int]
    KIND_FIELD_NUMBER: _ClassVar[int]
    KIT_FIELD_NUMBER: _ClassVar[int]
    THEME_FIELD_NUMBER: _ClassVar[int]
    DIRECTION_FIELD_NUMBER: _ClassVar[int]
    input_hashes: _containers.ScalarMap[str, str]
    revision: str
    render_hash: str
    html_sha256: str
    inputs_sha256: str
    kind: str
    kit: str
    theme: str
    direction: str
    def __init__(self, input_hashes: _Optional[_Mapping[str, str]] = ..., revision: _Optional[str] = ..., render_hash: _Optional[str] = ..., html_sha256: _Optional[str] = ..., inputs_sha256: _Optional[str] = ..., kind: _Optional[str] = ..., kit: _Optional[str] = ..., theme: _Optional[str] = ..., direction: _Optional[str] = ...) -> None: ...
