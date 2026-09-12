import { describe, expect, it } from 'vitest';
import {
  ArtifactType,
  ArtifactTypeSchema,
  ExecutionStatus,
  ExecutionStatusSchema,
  LogLevel,
  LogLevelSchema,
  StepStatus,
  StepStatusSchema,
} from '@vrooli/proto-types/browser-automation-studio/v1/base/shared_pb';
import {
  ActionType,
  ActionTypeSchema,
} from '@vrooli/proto-types/browser-automation-studio/v1/actions/action_pb';
import { enumShortName } from './protoEnumName';

describe('enumShortName', () => {
  it('matches the deleted generated map values for execution status', () => {
    expect(enumShortName(ExecutionStatusSchema, ExecutionStatus.UNSPECIFIED)).toBe('unspecified');
    expect(enumShortName(ExecutionStatusSchema, ExecutionStatus.PENDING)).toBe('pending');
    expect(enumShortName(ExecutionStatusSchema, ExecutionStatus.RUNNING)).toBe('running');
    expect(enumShortName(ExecutionStatusSchema, ExecutionStatus.COMPLETED)).toBe('completed');
    expect(enumShortName(ExecutionStatusSchema, ExecutionStatus.FAILED)).toBe('failed');
    expect(enumShortName(ExecutionStatusSchema, ExecutionStatus.CANCELLED)).toBe('cancelled');
  });

  it('matches the deleted generated map values for action type', () => {
    expect(enumShortName(ActionTypeSchema, ActionType.UNSPECIFIED)).toBe('unspecified');
    expect(enumShortName(ActionTypeSchema, ActionType.NAVIGATE)).toBe('navigate');
    expect(enumShortName(ActionTypeSchema, ActionType.CLICK)).toBe('click');
    expect(enumShortName(ActionTypeSchema, ActionType.UPLOAD_FILE)).toBe('upload_file');
    expect(enumShortName(ActionTypeSchema, ActionType.FRAME_SWITCH)).toBe('frame_switch');
    expect(enumShortName(ActionTypeSchema, ActionType.NETWORK_MOCK)).toBe('network_mock');
  });

  it('matches the deleted generated map values for step status', () => {
    expect(enumShortName(StepStatusSchema, StepStatus.UNSPECIFIED)).toBe('unspecified');
    expect(enumShortName(StepStatusSchema, StepStatus.PENDING)).toBe('pending');
    expect(enumShortName(StepStatusSchema, StepStatus.COMPLETED)).toBe('completed');
    expect(enumShortName(StepStatusSchema, StepStatus.CANCELLED)).toBe('cancelled');
    expect(enumShortName(StepStatusSchema, StepStatus.RETRYING)).toBe('retrying');
  });

  it('matches the deleted generated map values for artifact type', () => {
    expect(enumShortName(ArtifactTypeSchema, ArtifactType.UNSPECIFIED)).toBe('unspecified');
    expect(enumShortName(ArtifactTypeSchema, ArtifactType.TIMELINE_FRAME)).toBe('timeline_frame');
    expect(enumShortName(ArtifactTypeSchema, ArtifactType.CONSOLE_LOG)).toBe('console_log');
    expect(enumShortName(ArtifactTypeSchema, ArtifactType.NETWORK_EVENT)).toBe('network_event');
    expect(enumShortName(ArtifactTypeSchema, ArtifactType.DOM_SNAPSHOT)).toBe('dom_snapshot');
  });

  it('matches the deleted generated map values for log level and unknown values', () => {
    expect(enumShortName(LogLevelSchema, LogLevel.UNSPECIFIED)).toBe('unspecified');
    expect(enumShortName(LogLevelSchema, LogLevel.DEBUG)).toBe('debug');
    expect(enumShortName(LogLevelSchema, LogLevel.INFO)).toBe('info');
    expect(enumShortName(LogLevelSchema, LogLevel.WARN)).toBe('warn');
    expect(enumShortName(LogLevelSchema, LogLevel.ERROR)).toBe('error');
    expect(enumShortName(LogLevelSchema, 999)).toBeUndefined();
  });
});
