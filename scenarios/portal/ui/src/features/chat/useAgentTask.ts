import { useChatAccess } from "./ChatAccount";
import type { ChatAccess } from "../../api/chat";
import { createContext, createElement, useCallback, useContext, useEffect, useRef, useState, useSyncExternalStore, type ReactNode } from "react";
import { getPortalAgentRun, listPortalAgentAdmissions, stopPortalAgentRun } from "../../api/chat";
import { useCompanionStop } from "../companion/CompanionPresentation";

type Task = { chatId: string; messageId: string };
type PendingTask = Task & { actor:string; uncertain: boolean; busy: boolean };
type Snapshot = { tasks: Map<string, PendingTask>; recovering: boolean; recoveryFailed: boolean };
const identity = (task: Task & { actor?: string }) => JSON.stringify([task.actor ?? "legacy", task.chatId, task.messageId]);

/** Owner records recover identity; only an owner terminal response removes work. */
class AgentTasks {
 constructor(private access:()=>ChatAccess|undefined){}
 private recoveryActor="legacy";
  private snapshot: Snapshot = { tasks: new Map(), recovering: true, recoveryFailed: false };
  private listeners = new Set<() => void>();
  private recovery: Promise<void> | undefined;
  getSnapshot = () => this.snapshot;
  subscribe = (listener: () => void) => { this.listeners.add(listener); return () => { this.listeners.delete(listener); }; };
  private publish(tasks = this.snapshot.tasks, recovering = this.snapshot.recovering, recoveryFailed = this.snapshot.recoveryFailed) {
    this.snapshot = { tasks, recovering, recoveryFailed }; this.listeners.forEach(listener => listener());
  }
  private put(key: string, task: PendingTask | null) {
    const tasks = new Map(this.snapshot.tasks);
    if (task) tasks.set(key, task); else tasks.delete(key);
    this.publish(tasks);
  }
  begin = (chatId: string, messageId: string) => {
    if (this.snapshot.recovering) throw new Error("Agent task recovery is unresolved");
    const actor=this.access()?.actor??"legacy";
    if ([...this.snapshot.tasks.values()].some(task => task.actor===actor)) throw new Error("An agent task is still unresolved");
    if (!chatId.trim() || !messageId.trim()) throw new Error("Missing Portal task identity");
    const task = { chatId, messageId,actor, uncertain: false, busy: false };
    this.put(identity(task), task);
  };
  operate = async (key: string, stop: boolean) => {
    const selected = this.snapshot.tasks.get(key);
    if (!selected || selected.busy) return false;
    this.put(key, { ...selected, busy: true });
    try {
      const access=this.access();if(selected.actor!==(access?.actor??"legacy"))throw new Error("Original account required");
      const state = await (stop ? stopPortalAgentRun : getPortalAgentRun)(selected.chatId, selected.messageId,...(access?[access] as const:[]));
      if (!state.runId || !state.status) throw new Error("Missing owner run state");
      this.put(key, state.terminal ? null : { ...selected, busy: false, uncertain: false });
      return state.terminal;
    } catch {
      this.put(key, { ...selected, busy: false, uncertain: true });
      return false;
    }
  };
  stop = (key: string) => this.operate(key, !this.snapshot.tasks.get(key)?.uncertain);
  recover = (): Promise<void> => {
    if (this.recovery) return this.recoveryActor===(this.access()?.actor??"legacy")?this.recovery:this.recovery.then(()=>this.recover());
    this.recoveryActor=this.access()?.actor??"legacy";
    this.publish(this.snapshot.tasks, true, false);
    this.recovery = this.readAdmissions().finally(() => { this.recovery = undefined; });
    return this.recovery;
  };
  private async readAdmissions() {
    const access=this.access(),actor=access?.actor??"legacy";
    try {
      let token = "";
      const seen = new Set<string>();
      do {
        if (seen.has(token)) throw new Error("Repeated recovery cursor");
        seen.add(token);
        const page = await listPortalAgentAdmissions(token,...(access?[access] as const:[]));
        for (const admission of page.admissions) {
          if (!admission.chatId.trim() || !admission.messageId.trim()) throw new Error("Missing admission identity");
          const key = identity({...admission,actor});
          if (!this.snapshot.tasks.has(key)) this.put(key, { chatId: admission.chatId, messageId: admission.messageId,actor, busy: false, uncertain: true });
        }
        token = page.nextPageToken;
      } while (token);
      const keys = [...this.snapshot.tasks].filter(([,task])=>task.actor===actor).map(([key])=>key);
      // Bound concurrent owner reads without dropping later pages or unknown runs.
      for (let offset = 0; offset < keys.length; offset += 4) {
        await Promise.all(keys.slice(offset, offset + 4).map(key => this.operate(key, false)));
      }
      this.publish(this.snapshot.tasks, false, false);
    } catch {
      // Keep the recovery task registered. Its Check status action retries reads.
      this.publish(this.snapshot.tasks, true, true);
    }
  }
}

const Context = createContext<AgentTasks | null>(null);
function RegisteredTask({ store, task }: { store: AgentTasks; task: PendingTask }) {
  const key = identity(task);
  const stop = useCallback(async () => { await store.stop(key); }, [store, key]);
  useCompanionStop(`agent:${key}`, "chat", true, stop, task.uncertain);
  return null;
}
export function AgentTasksProvider({ children }: { children: ReactNode }) {
  const access=useChatAccess();const accessRef=useRef(access);accessRef.current=access;
  const [store] = useState(() => new AgentTasks(()=>accessRef.current));
  const snapshot = useSyncExternalStore(store.subscribe, store.getSnapshot);
  const actor=access?.actor??"legacy";
  const visibleTasks=[...snapshot.tasks].filter(([,task])=>task.actor===actor);
  useCompanionStop("agent-recovery", "chat", snapshot.recovering, store.recover, true);
  useEffect(() => { void store.recover(); }, [store,access]);
  return createElement(Context.Provider, { value: store }, children,
    visibleTasks.map(([key, task]) => createElement(RegisteredTask, { key, store, task })));
}

/** ChatWorkspace observes shell-owned work; unmounting it cannot discard a run. */
export function useAgentTask() {
  const store = useContext(Context);
  if (!store) throw new Error("AgentTasksProvider is required");
  const snapshot = useSyncExternalStore(store.subscribe, store.getSnapshot);
  const actor=useChatAccess()?.actor??"legacy";
  const first = [...snapshot.tasks.values()].find(task=>task.actor===actor);
  const task = first ? { chatId: first.chatId, messageId: first.messageId } : null;
  const current = useRef<Task | null>(task); current.current = task;
  const inspect = useCallback(() => {
    const key=[...store.getSnapshot().tasks].find(([,task])=>task.actor===actor)?.[0] ?? "";
    return store.operate(key, false);
  }, [actor, store]);
  const stop = useCallback(() => {
    const key=[...store.getSnapshot().tasks].find(([,task])=>task.actor===actor)?.[0] ?? "";
    return store.stop(key);
  }, [actor, store]);
  return { task, current, uncertain: first?.uncertain ?? false, busy: first?.busy ?? false,
    recovering: snapshot.recovering, recoveryFailed: snapshot.recoveryFailed, recover: store.recover, begin: store.begin, inspect, stop };
}
