import { writeFileSync } from "node:fs";

// Public Vitest v2 reporter hooks only. Preserve what the runner actually emits;
// a retry count is not a reconstructed first-attempt assertion failure.
export default class EventsReporter {
  events = [];
  names = new Map();
  onCollected(files) {
    const visit = (task, parents = []) => {
      const names = task.type === "suite" && task.file === task ? parents : [...parents, task.name];
      this.names.set(task.id, names.join(" "));
      for (const child of task.tasks ?? []) visit(child, names);
    };
    for (const file of files ?? []) visit(file);
  }
  onTaskUpdate(packs) {
    for (const [id, result] of packs) {
      this.events.push({ name: this.names.get(id) ?? id, state: result?.state ?? "unknown",
        retryCount: result?.retryCount ?? 0,
        errors: (result?.errors ?? []).map(error => ({ name: error.name, message: error.message })) });
    }
  }
  onFinished(_files, errors) {
    writeFileSync(process.env.QUALITY_NATIVE_EVENTS, JSON.stringify({ events: this.events,
      errors: (errors ?? []).map(error => ({ name: error.name, message: error.message })) }));
  }
}
