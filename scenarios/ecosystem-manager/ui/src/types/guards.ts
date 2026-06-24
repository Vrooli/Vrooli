/**
 * Runtime type guards for the API union types.
 *
 * These validate values that cross an untrusted boundary — URL query params,
 * proto/JSON responses, drag-and-drop payloads, and `<Select>` callbacks —
 * instead of asserting their type with `as`. A bad value is rejected at the
 * boundary rather than silently lying to the type checker and crashing later.
 *
 * Each guard is backed by a `ReadonlySet<string>` built from the single-source
 * tuples in ./api, so the set of accepted values can never drift from the type.
 */

import {
  OPERATION_TYPES,
  PRIORITIES,
  STEERING_STRATEGIES,
  TASK_SORTS,
  TASK_STATUSES,
  TASK_TYPES,
  type OperationType,
  type Priority,
  type SteeringStrategy,
  type TaskSort,
  type TaskStatus,
  type TaskType,
} from './api';

const TASK_STATUS_SET: ReadonlySet<string> = new Set(TASK_STATUSES);
const TASK_SORT_SET: ReadonlySet<string> = new Set(TASK_SORTS);
const TASK_TYPE_SET: ReadonlySet<string> = new Set(TASK_TYPES);
const OPERATION_TYPE_SET: ReadonlySet<string> = new Set(OPERATION_TYPES);
const PRIORITY_SET: ReadonlySet<string> = new Set(PRIORITIES);
const STEERING_STRATEGY_SET: ReadonlySet<string> = new Set(STEERING_STRATEGIES);

export const isTaskStatus = (value: unknown): value is TaskStatus =>
  typeof value === 'string' && TASK_STATUS_SET.has(value);

export const isTaskSort = (value: unknown): value is TaskSort =>
  typeof value === 'string' && TASK_SORT_SET.has(value);

export const isTaskType = (value: unknown): value is TaskType =>
  typeof value === 'string' && TASK_TYPE_SET.has(value);

export const isOperationType = (value: unknown): value is OperationType =>
  typeof value === 'string' && OPERATION_TYPE_SET.has(value);

export const isPriority = (value: unknown): value is Priority =>
  typeof value === 'string' && PRIORITY_SET.has(value);

export const isSteeringStrategy = (value: unknown): value is SteeringStrategy =>
  typeof value === 'string' && STEERING_STRATEGY_SET.has(value);

/** Narrows an unknown value to a plain object without using a cast. */
export const isRecord = (value: unknown): value is Record<string, unknown> =>
  typeof value === 'object' && value !== null && !Array.isArray(value);
