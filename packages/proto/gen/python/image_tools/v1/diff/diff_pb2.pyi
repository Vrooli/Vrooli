from google.protobuf.internal import containers as _containers
from google.protobuf.internal import enum_type_wrapper as _enum_type_wrapper
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Iterable as _Iterable, Mapping as _Mapping
from typing import ClassVar as _ClassVar, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class DiffMode(int, metaclass=_enum_type_wrapper.EnumTypeWrapper):
    __slots__ = ()
    DIFF_MODE_UNSPECIFIED: _ClassVar[DiffMode]
    DIFF_MODE_PIXEL: _ClassVar[DiffMode]
    DIFF_MODE_PERCEPTUAL: _ClassVar[DiffMode]
DIFF_MODE_UNSPECIFIED: DiffMode
DIFF_MODE_PIXEL: DiffMode
DIFF_MODE_PERCEPTUAL: DiffMode

class DiffParams(_message.Message):
    __slots__ = ("mode", "tolerance", "include_heatmap", "highlight_hex")
    MODE_FIELD_NUMBER: _ClassVar[int]
    TOLERANCE_FIELD_NUMBER: _ClassVar[int]
    INCLUDE_HEATMAP_FIELD_NUMBER: _ClassVar[int]
    HIGHLIGHT_HEX_FIELD_NUMBER: _ClassVar[int]
    mode: DiffMode
    tolerance: float
    include_heatmap: bool
    highlight_hex: str
    def __init__(self, mode: _Optional[_Union[DiffMode, str]] = ..., tolerance: _Optional[float] = ..., include_heatmap: _Optional[bool] = ..., highlight_hex: _Optional[str] = ...) -> None: ...

class DiffModeInfo(_message.Message):
    __slots__ = ("name", "summary")
    NAME_FIELD_NUMBER: _ClassVar[int]
    SUMMARY_FIELD_NUMBER: _ClassVar[int]
    name: str
    summary: str
    def __init__(self, name: _Optional[str] = ..., summary: _Optional[str] = ...) -> None: ...

class ListDiffModesRequest(_message.Message):
    __slots__ = ()
    def __init__(self) -> None: ...

class ListDiffModesResponse(_message.Message):
    __slots__ = ("modes",)
    MODES_FIELD_NUMBER: _ClassVar[int]
    modes: _containers.RepeatedCompositeFieldContainer[DiffModeInfo]
    def __init__(self, modes: _Optional[_Iterable[_Union[DiffModeInfo, _Mapping]]] = ...) -> None: ...

class DiffResult(_message.Message):
    __slots__ = ("job_id", "verdict", "dimensions_match", "base_width", "base_height", "compare_width", "compare_height", "changed_pixels", "total_pixels", "changed_fraction", "mae", "rmse", "psnr", "phash_distance", "phash_similarity", "ssim", "heatmap_ref", "warnings")
    JOB_ID_FIELD_NUMBER: _ClassVar[int]
    VERDICT_FIELD_NUMBER: _ClassVar[int]
    DIMENSIONS_MATCH_FIELD_NUMBER: _ClassVar[int]
    BASE_WIDTH_FIELD_NUMBER: _ClassVar[int]
    BASE_HEIGHT_FIELD_NUMBER: _ClassVar[int]
    COMPARE_WIDTH_FIELD_NUMBER: _ClassVar[int]
    COMPARE_HEIGHT_FIELD_NUMBER: _ClassVar[int]
    CHANGED_PIXELS_FIELD_NUMBER: _ClassVar[int]
    TOTAL_PIXELS_FIELD_NUMBER: _ClassVar[int]
    CHANGED_FRACTION_FIELD_NUMBER: _ClassVar[int]
    MAE_FIELD_NUMBER: _ClassVar[int]
    RMSE_FIELD_NUMBER: _ClassVar[int]
    PSNR_FIELD_NUMBER: _ClassVar[int]
    PHASH_DISTANCE_FIELD_NUMBER: _ClassVar[int]
    PHASH_SIMILARITY_FIELD_NUMBER: _ClassVar[int]
    SSIM_FIELD_NUMBER: _ClassVar[int]
    HEATMAP_REF_FIELD_NUMBER: _ClassVar[int]
    WARNINGS_FIELD_NUMBER: _ClassVar[int]
    job_id: str
    verdict: str
    dimensions_match: bool
    base_width: int
    base_height: int
    compare_width: int
    compare_height: int
    changed_pixels: int
    total_pixels: int
    changed_fraction: float
    mae: float
    rmse: float
    psnr: float
    phash_distance: int
    phash_similarity: float
    ssim: float
    heatmap_ref: str
    warnings: _containers.RepeatedScalarFieldContainer[str]
    def __init__(self, job_id: _Optional[str] = ..., verdict: _Optional[str] = ..., dimensions_match: _Optional[bool] = ..., base_width: _Optional[int] = ..., base_height: _Optional[int] = ..., compare_width: _Optional[int] = ..., compare_height: _Optional[int] = ..., changed_pixels: _Optional[int] = ..., total_pixels: _Optional[int] = ..., changed_fraction: _Optional[float] = ..., mae: _Optional[float] = ..., rmse: _Optional[float] = ..., psnr: _Optional[float] = ..., phash_distance: _Optional[int] = ..., phash_similarity: _Optional[float] = ..., ssim: _Optional[float] = ..., heatmap_ref: _Optional[str] = ..., warnings: _Optional[_Iterable[str]] = ...) -> None: ...
