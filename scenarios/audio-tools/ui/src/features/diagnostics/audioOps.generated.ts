// AUTO-GENERATED — do not edit by hand.
//
// Source : api/internal/operationcatalog/operations.seed.json
// Codegen: cd api && go run ./cmd/gen-operations
//
// Drift gate: api/internal/operationcatalog/catalog_test.go compares this
// file to a fresh render of the seed. Edit the seed, regenerate, and commit.

export type AudioOpControl =
  | "number"
  | "text"
  | "select"
  | "file"
  | "files"
  | "checkbox";

export type AudioOpId =
   "transcode"
  | "trim"
  | "merge"
  | "split"
  | "fade"
  | "volume"
  | "normalize"
  | "extract_metadata"
;

export type AudioOpMethod =
   "transcode"
  | "trim"
  | "merge"
  | "split"
  | "fade"
  | "volume"
  | "normalize"
  | "extractMetadata"
;

export type AudioOpParamName =
   "audio"
  | "output_format"
  | "input_format"
  | "sample_rate"
  | "channels"
  | "bitrate"
  | "format"
  | "start_seconds"
  | "end_seconds"
  | "sources"
  | "crossfade_seconds"
  | "chunk_seconds"
  | "boundaries_seconds"
  | "fade_in_seconds"
  | "fade_out_seconds"
  | "gain_db"
  | "method"
  | "target_lufs"
;

export type AudioOpLabelKey =
   "diagnostics.audioOps.transcode.label"
  | "diagnostics.audioOps.trim.label"
  | "diagnostics.audioOps.merge.label"
  | "diagnostics.audioOps.split.label"
  | "diagnostics.audioOps.fade.label"
  | "diagnostics.audioOps.volume.label"
  | "diagnostics.audioOps.normalize.label"
  | "diagnostics.audioOps.extract_metadata.label"
;

export type AudioOpDescriptionKey =
   "diagnostics.audioOps.transcode.description"
  | "diagnostics.audioOps.trim.description"
  | "diagnostics.audioOps.merge.description"
  | "diagnostics.audioOps.split.description"
  | "diagnostics.audioOps.fade.description"
  | "diagnostics.audioOps.volume.description"
  | "diagnostics.audioOps.normalize.description"
  | "diagnostics.audioOps.extract_metadata.description"
;

export type AudioOpOptionLabelKey =
   "diagnostics.formatWav"
  | "diagnostics.formatMp3"
  | "diagnostics.formatFlac"
  | "diagnostics.formatOgg"
  | "diagnostics.formatAac"
  | "diagnostics.audioOps.optionAuto"
  | "diagnostics.audioOps.methodEbuR128"
  | "diagnostics.audioOps.methodRms"
  | "diagnostics.audioOps.methodPeak"
;

export interface AudioOpParamOption {
  readonly value: string;
  readonly labelKey: AudioOpOptionLabelKey;
}

export interface AudioOpParam {
  readonly name: AudioOpParamName;
  readonly type: string;
  readonly control: AudioOpControl;
  readonly required: boolean;
  readonly defaultValue: string;
  readonly min?: number;
  readonly max?: number;
  readonly step?: number;
  readonly unit?: string;
  readonly advanced?: boolean;
  readonly options?: readonly AudioOpParamOption[];
}

export interface AudioOpSpec {
  readonly id: AudioOpId;
  readonly method: AudioOpMethod;
  readonly summary: string;
  readonly labelKey: AudioOpLabelKey;
  readonly descriptionKey: AudioOpDescriptionKey;
  readonly order: number;
  readonly params: readonly AudioOpParam[];
}

export const audioOps: readonly AudioOpSpec[] = [
  {
    id: "transcode",
    method: "transcode",
    summary: "Re-encode audio into another container, sample rate, channel count, or bitrate.",
    labelKey: "diagnostics.audioOps.transcode.label",
    descriptionKey: "diagnostics.audioOps.transcode.description",
    order: 10,
    params: [
      {
        name: "audio",
        type: "bytes",
        control: "file",
        required: true,
        defaultValue: "",
      },
      {
        name: "output_format",
        type: "AudioFormat",
        control: "select",
        required: false,
        defaultValue: "wav",
        options: [
          { value: "wav", labelKey: "diagnostics.formatWav" },
          { value: "mp3", labelKey: "diagnostics.formatMp3" },
          { value: "flac", labelKey: "diagnostics.formatFlac" },
          { value: "ogg", labelKey: "diagnostics.formatOgg" },
          { value: "aac", labelKey: "diagnostics.formatAac" },
        ],
      },
      {
        name: "input_format",
        type: "AudioFormat",
        control: "select",
        required: false,
        defaultValue: "",
        advanced: true,
        options: [
          { value: "", labelKey: "diagnostics.audioOps.optionAuto" },
          { value: "wav", labelKey: "diagnostics.formatWav" },
          { value: "mp3", labelKey: "diagnostics.formatMp3" },
          { value: "flac", labelKey: "diagnostics.formatFlac" },
          { value: "ogg", labelKey: "diagnostics.formatOgg" },
          { value: "aac", labelKey: "diagnostics.formatAac" },
        ],
      },
      {
        name: "sample_rate",
        type: "int32",
        control: "number",
        required: false,
        defaultValue: "0",
        min: 0,
        max: 192000,
        step: 1,
        unit: "Hz",
        advanced: true,
      },
      {
        name: "channels",
        type: "int32",
        control: "number",
        required: false,
        defaultValue: "0",
        min: 0,
        max: 8,
        step: 1,
        advanced: true,
      },
      {
        name: "bitrate",
        type: "int32",
        control: "number",
        required: false,
        defaultValue: "0",
        min: 0,
        max: 512000,
        step: 1000,
        unit: "bps",
        advanced: true,
      },
    ],
  },
  {
    id: "trim",
    method: "trim",
    summary: "Cut a section out of an audio file between two timestamps.",
    labelKey: "diagnostics.audioOps.trim.label",
    descriptionKey: "diagnostics.audioOps.trim.description",
    order: 20,
    params: [
      {
        name: "audio",
        type: "bytes",
        control: "file",
        required: true,
        defaultValue: "",
      },
      {
        name: "format",
        type: "AudioFormat",
        control: "select",
        required: false,
        defaultValue: "wav",
        options: [
          { value: "wav", labelKey: "diagnostics.formatWav" },
          { value: "mp3", labelKey: "diagnostics.formatMp3" },
          { value: "flac", labelKey: "diagnostics.formatFlac" },
          { value: "ogg", labelKey: "diagnostics.formatOgg" },
          { value: "aac", labelKey: "diagnostics.formatAac" },
        ],
      },
      {
        name: "start_seconds",
        type: "double",
        control: "number",
        required: true,
        defaultValue: "0",
        min: 0,
        max: 86400,
        step: 0.1,
        unit: "s",
      },
      {
        name: "end_seconds",
        type: "double",
        control: "number",
        required: false,
        defaultValue: "0",
        min: 0,
        max: 86400,
        step: 0.1,
        unit: "s",
      },
    ],
  },
  {
    id: "merge",
    method: "merge",
    summary: "Concatenate several audio files, optionally with a crossfade between them.",
    labelKey: "diagnostics.audioOps.merge.label",
    descriptionKey: "diagnostics.audioOps.merge.description",
    order: 30,
    params: [
      {
        name: "sources",
        type: "MergeSource[]",
        control: "files",
        required: true,
        defaultValue: "",
      },
      {
        name: "output_format",
        type: "AudioFormat",
        control: "select",
        required: false,
        defaultValue: "wav",
        options: [
          { value: "wav", labelKey: "diagnostics.formatWav" },
          { value: "mp3", labelKey: "diagnostics.formatMp3" },
          { value: "flac", labelKey: "diagnostics.formatFlac" },
          { value: "ogg", labelKey: "diagnostics.formatOgg" },
          { value: "aac", labelKey: "diagnostics.formatAac" },
        ],
      },
      {
        name: "crossfade_seconds",
        type: "double",
        control: "number",
        required: false,
        defaultValue: "0",
        min: 0,
        max: 60,
        step: 0.1,
        unit: "s",
      },
    ],
  },
  {
    id: "split",
    method: "split",
    summary: "Split an audio file into equal chunks or at explicit boundaries.",
    labelKey: "diagnostics.audioOps.split.label",
    descriptionKey: "diagnostics.audioOps.split.description",
    order: 40,
    params: [
      {
        name: "audio",
        type: "bytes",
        control: "file",
        required: true,
        defaultValue: "",
      },
      {
        name: "format",
        type: "AudioFormat",
        control: "select",
        required: false,
        defaultValue: "wav",
        options: [
          { value: "wav", labelKey: "diagnostics.formatWav" },
          { value: "mp3", labelKey: "diagnostics.formatMp3" },
          { value: "flac", labelKey: "diagnostics.formatFlac" },
          { value: "ogg", labelKey: "diagnostics.formatOgg" },
          { value: "aac", labelKey: "diagnostics.formatAac" },
        ],
      },
      {
        name: "chunk_seconds",
        type: "double",
        control: "number",
        required: false,
        defaultValue: "0",
        min: 0,
        max: 3600,
        step: 0.1,
        unit: "s",
      },
      {
        name: "boundaries_seconds",
        type: "string",
        control: "text",
        required: false,
        defaultValue: "",
        advanced: true,
      },
      {
        name: "output_format",
        type: "AudioFormat",
        control: "select",
        required: false,
        defaultValue: "wav",
        advanced: true,
        options: [
          { value: "wav", labelKey: "diagnostics.formatWav" },
          { value: "mp3", labelKey: "diagnostics.formatMp3" },
          { value: "flac", labelKey: "diagnostics.formatFlac" },
          { value: "ogg", labelKey: "diagnostics.formatOgg" },
          { value: "aac", labelKey: "diagnostics.formatAac" },
        ],
      },
    ],
  },
  {
    id: "fade",
    method: "fade",
    summary: "Apply a fade-in and/or fade-out window to an audio file.",
    labelKey: "diagnostics.audioOps.fade.label",
    descriptionKey: "diagnostics.audioOps.fade.description",
    order: 50,
    params: [
      {
        name: "audio",
        type: "bytes",
        control: "file",
        required: true,
        defaultValue: "",
      },
      {
        name: "format",
        type: "AudioFormat",
        control: "select",
        required: false,
        defaultValue: "wav",
        options: [
          { value: "wav", labelKey: "diagnostics.formatWav" },
          { value: "mp3", labelKey: "diagnostics.formatMp3" },
          { value: "flac", labelKey: "diagnostics.formatFlac" },
          { value: "ogg", labelKey: "diagnostics.formatOgg" },
          { value: "aac", labelKey: "diagnostics.formatAac" },
        ],
      },
      {
        name: "fade_in_seconds",
        type: "double",
        control: "number",
        required: false,
        defaultValue: "0",
        min: 0,
        max: 3600,
        step: 0.1,
        unit: "s",
      },
      {
        name: "fade_out_seconds",
        type: "double",
        control: "number",
        required: false,
        defaultValue: "0",
        min: 0,
        max: 3600,
        step: 0.1,
        unit: "s",
      },
      {
        name: "output_format",
        type: "AudioFormat",
        control: "select",
        required: false,
        defaultValue: "wav",
        advanced: true,
        options: [
          { value: "wav", labelKey: "diagnostics.formatWav" },
          { value: "mp3", labelKey: "diagnostics.formatMp3" },
          { value: "flac", labelKey: "diagnostics.formatFlac" },
          { value: "ogg", labelKey: "diagnostics.formatOgg" },
          { value: "aac", labelKey: "diagnostics.formatAac" },
        ],
      },
    ],
  },
  {
    id: "volume",
    method: "volume",
    summary: "Scale an audio file by a gain in decibels.",
    labelKey: "diagnostics.audioOps.volume.label",
    descriptionKey: "diagnostics.audioOps.volume.description",
    order: 60,
    params: [
      {
        name: "audio",
        type: "bytes",
        control: "file",
        required: true,
        defaultValue: "",
      },
      {
        name: "format",
        type: "AudioFormat",
        control: "select",
        required: false,
        defaultValue: "wav",
        options: [
          { value: "wav", labelKey: "diagnostics.formatWav" },
          { value: "mp3", labelKey: "diagnostics.formatMp3" },
          { value: "flac", labelKey: "diagnostics.formatFlac" },
          { value: "ogg", labelKey: "diagnostics.formatOgg" },
          { value: "aac", labelKey: "diagnostics.formatAac" },
        ],
      },
      {
        name: "gain_db",
        type: "double",
        control: "number",
        required: true,
        defaultValue: "0",
        min: -60,
        max: 24,
        step: 0.5,
        unit: "dB",
      },
      {
        name: "output_format",
        type: "AudioFormat",
        control: "select",
        required: false,
        defaultValue: "wav",
        advanced: true,
        options: [
          { value: "wav", labelKey: "diagnostics.formatWav" },
          { value: "mp3", labelKey: "diagnostics.formatMp3" },
          { value: "flac", labelKey: "diagnostics.formatFlac" },
          { value: "ogg", labelKey: "diagnostics.formatOgg" },
          { value: "aac", labelKey: "diagnostics.formatAac" },
        ],
      },
    ],
  },
  {
    id: "normalize",
    method: "normalize",
    summary: "Normalize loudness with EBU R128, RMS, or peak normalization.",
    labelKey: "diagnostics.audioOps.normalize.label",
    descriptionKey: "diagnostics.audioOps.normalize.description",
    order: 70,
    params: [
      {
        name: "audio",
        type: "bytes",
        control: "file",
        required: true,
        defaultValue: "",
      },
      {
        name: "format",
        type: "AudioFormat",
        control: "select",
        required: false,
        defaultValue: "wav",
        options: [
          { value: "wav", labelKey: "diagnostics.formatWav" },
          { value: "mp3", labelKey: "diagnostics.formatMp3" },
          { value: "flac", labelKey: "diagnostics.formatFlac" },
          { value: "ogg", labelKey: "diagnostics.formatOgg" },
          { value: "aac", labelKey: "diagnostics.formatAac" },
        ],
      },
      {
        name: "method",
        type: "NormalizationMethod",
        control: "select",
        required: false,
        defaultValue: "ebu_r128",
        options: [
          { value: "ebu_r128", labelKey: "diagnostics.audioOps.methodEbuR128" },
          { value: "rms", labelKey: "diagnostics.audioOps.methodRms" },
          { value: "peak", labelKey: "diagnostics.audioOps.methodPeak" },
        ],
      },
      {
        name: "target_lufs",
        type: "double",
        control: "number",
        required: false,
        defaultValue: "-16",
        min: -40,
        max: -5,
        step: 0.1,
        unit: "LUFS",
        advanced: true,
      },
      {
        name: "output_format",
        type: "AudioFormat",
        control: "select",
        required: false,
        defaultValue: "wav",
        advanced: true,
        options: [
          { value: "wav", labelKey: "diagnostics.formatWav" },
          { value: "mp3", labelKey: "diagnostics.formatMp3" },
          { value: "flac", labelKey: "diagnostics.formatFlac" },
          { value: "ogg", labelKey: "diagnostics.formatOgg" },
          { value: "aac", labelKey: "diagnostics.formatAac" },
        ],
      },
    ],
  },
  {
    id: "extract_metadata",
    method: "extractMetadata",
    summary: "Read duration, sample rate, channels, bitrate, codec, format, and tags.",
    labelKey: "diagnostics.audioOps.extract_metadata.label",
    descriptionKey: "diagnostics.audioOps.extract_metadata.description",
    order: 80,
    params: [
      {
        name: "audio",
        type: "bytes",
        control: "file",
        required: true,
        defaultValue: "",
      },
      {
        name: "format",
        type: "AudioFormat",
        control: "select",
        required: false,
        defaultValue: "wav",
        options: [
          { value: "wav", labelKey: "diagnostics.formatWav" },
          { value: "mp3", labelKey: "diagnostics.formatMp3" },
          { value: "flac", labelKey: "diagnostics.formatFlac" },
          { value: "ogg", labelKey: "diagnostics.formatOgg" },
          { value: "aac", labelKey: "diagnostics.formatAac" },
        ],
      },
    ],
  },
];

export const audioOpsById = (id: AudioOpId): AudioOpSpec | undefined =>
  audioOps.find((op) => op.id === id);
