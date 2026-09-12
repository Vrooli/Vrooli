import { TaskStatus, type Task } from "../../src/types.js";

export type TaskOverrides = Partial<Task>;

export function makeTask(overrides: TaskOverrides = {}): Task {
  return {
    id: overrides.id ?? "task-1",
    title: overrides.title ?? "Review stats dashboard",
    description: overrides.description ?? "Tighten the dashboard test coverage",
    scopePath: overrides.scopePath ?? ".",
    projectRoot: overrides.projectRoot ?? "/workspace/repo",
    status: overrides.status ?? TaskStatus.RUNNING,
    ...overrides,
  } as Task;
}
