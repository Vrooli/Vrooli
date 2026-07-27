import type { DesktopRecordsResponse } from "@vrooli/proto-types/scenario-to-desktop/v1/domain/records_pb";

/** View data derived from the generated records contract for record components. */
export interface DesktopRecordView {
  id: string;
  build_id: string;
  scenario_name: string;
  app_display_name?: string;
  template_type?: string;
  framework?: string;
  location_mode?: string;
  output_path: string;
  destination_path?: string;
  staging_path?: string;
  custom_path?: string;
  deployment_mode?: string;
  icon?: string;
  created_at?: string;
  updated_at?: string;
}

export interface DesktopRecordItemView {
  record: DesktopRecordView;
  build_status?: {
    status: string;
    output_path?: string;
    template_type?: string;
    framework?: string;
    metadata?: {
      version?: string;
      git_branch?: string;
      git_commit_hash?: string;
      git_dirty?: boolean;
    };
  };
  has_build: boolean;
  build_state?: string;
  smoke_test_id?: string;
  screen_recording?: {
    recorded: boolean;
    duration_ms?: number;
    file_size_bytes?: number;
    error?: string;
  };
}

const timestampISO = (value: { seconds: bigint; nanos: number } | undefined) =>
  value
    ? new Date(
        Number(value.seconds) * 1000 + value.nanos / 1_000_000,
      ).toISOString()
    : undefined;

export function presentDesktopRecords(
  response: DesktopRecordsResponse | undefined,
): DesktopRecordItemView[] {
  return (response?.records ?? []).map((item) => ({
    record: {
      id: item.record?.id ?? "",
      build_id: item.record?.buildId ?? "",
      scenario_name: item.record?.scenarioName ?? "",
      app_display_name: item.record?.appDisplayName,
      template_type: item.record?.templateType,
      framework: item.record?.framework,
      location_mode: item.record?.locationMode,
      output_path: item.record?.outputPath ?? "",
      destination_path: item.record?.destinationPath,
      staging_path: item.record?.stagingPath,
      custom_path: item.record?.customPath,
      deployment_mode: item.record?.deploymentMode,
      icon: item.record?.icon,
      created_at: timestampISO(item.record?.createdAt),
      updated_at: timestampISO(item.record?.updatedAt),
    },
    build_status: item.buildStatus && {
      status: item.buildStatus.status,
      output_path: item.buildStatus.outputPath,
      template_type: item.record?.templateType,
      framework: item.record?.framework,
      metadata: item.buildStatus.metadata && {
        version: item.buildStatus.metadata.version,
        git_branch: item.buildStatus.metadata.gitBranch,
        git_commit_hash: item.buildStatus.metadata.gitCommitHash,
        git_dirty: item.buildStatus.metadata.gitDirty,
      },
    },
    has_build: item.hasBuild,
    build_state: item.buildState,
    smoke_test_id: item.smokeTestId,
    screen_recording: item.screenRecording && {
      recorded: item.screenRecording.recorded,
      duration_ms:
        item.screenRecording.durationMs === undefined
          ? undefined
          : Number(item.screenRecording.durationMs),
      file_size_bytes:
        item.screenRecording.fileSizeBytes === undefined
          ? undefined
          : Number(item.screenRecording.fileSizeBytes),
      error: item.screenRecording.error,
    },
  }));
}
