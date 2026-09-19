import { useState } from "react";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { Button } from "@vrooli/react-component-library/Button/2";
import { Input } from "@vrooli/react-component-library/Input/1";
import type { Goal, Milestone } from "@vrooli/proto-types/personal-planner/v1/goals/goals_pb";

import { completeMilestone, createMilestone, fetchMilestones } from "../api/goals";
import { fetchWorkItems } from "../api/work";

export function GoalCard({ goal, progressMutation }: { goal: Goal; progressMutation: { mutate: (value: { goal: Goal; progress: bigint }) => void } }) {
  const queryClient = useQueryClient();
  const milestones = useQuery({ queryKey: ["milestones", goal.id], queryFn: () => fetchMilestones(goal.id) });
  const work = useQuery({ queryKey: ["work-items"], queryFn: fetchWorkItems });
  const [title, setTitle] = useState("");
  const [criteria, setCriteria] = useState("");
  const [dueDate, setDueDate] = useState("");
  const [linkedWorkItemId, setLinkedWorkItemId] = useState("");
  const [prerequisiteMilestoneIds, setPrerequisiteMilestoneIds] = useState<string[]>([]);
  const createMutation = useMutation({ mutationFn: createMilestone, onSuccess: async () => { setTitle(""); setCriteria(""); setDueDate(""); setLinkedWorkItemId(""); setPrerequisiteMilestoneIds([]); await queryClient.invalidateQueries({ queryKey: ["milestones", goal.id] }); await queryClient.invalidateQueries({ queryKey: ["goals"] }); } });
  const completeMutation = useMutation({ mutationFn: completeMilestone, onSuccess: async () => { await queryClient.invalidateQueries({ queryKey: ["milestones", goal.id] }); } });
  return <article className="goal-card">
    <div className="goal-card-main"><div><span className="card-kicker">{goal.status} · {goal.progressMethod}</span><h2>{goal.title}</h2><p>{goal.purpose || "No purpose recorded yet."}</p></div><label>Progress<Input type="range" min="0" max="10000" step="100" value={Number(goal.progressBasisPoints)} onChange={(event) => progressMutation.mutate({ goal, progress: BigInt(event.target.value) })} /><span>{Math.round(Number(goal.progressBasisPoints) / 100)}%</span></label></div>
    <section className="milestone-section" aria-labelledby={`milestones-${goal.id}`}>
      <div className="milestone-heading"><div><span className="card-kicker">Next proof</span><h3 id={`milestones-${goal.id}`}>Milestones</h3></div><span className="milestone-count">{milestones.data?.filter((m) => m.status === "complete").length ?? 0}/{milestones.data?.length ?? 0} complete</span></div>
      {milestones.isError && <p role="alert">Milestones are unavailable right now.</p>}
      <div className="milestone-list">{milestones.data?.map((milestone) => <MilestoneRow key={milestone.id} milestone={milestone} allMilestones={milestones.data ?? []} onComplete={() => completeMutation.mutate(milestone)} />)}</div>
      <form className="milestone-form" onSubmit={(event) => { event.preventDefault(); createMutation.mutate({ goalId: goal.id, title, criteria, dueDate, linkedWorkItemId, prerequisiteMilestoneIds }); }}>
        <label>Milestone title<Input value={title} onChange={(event) => setTitle(event.target.value)} placeholder="A proof point" required /></label>
        <label>Completion criteria<Input value={criteria} onChange={(event) => setCriteria(event.target.value)} placeholder="What makes it true?" /></label>
        <label>Due date<Input type="date" value={dueDate} onChange={(event) => setDueDate(event.target.value)} /></label>
        <label>Linked work<select value={linkedWorkItemId} onChange={(event) => setLinkedWorkItemId(event.target.value)}><option value="">No work linked</option>{work.data?.map((item) => <option key={item.id} value={item.id}>{item.title}</option>)}</select></label>
        {milestones.data && milestones.data.length > 0 && <fieldset><legend>Prerequisites</legend>{milestones.data.filter((milestone) => milestone.status !== "complete").map((milestone) => <label key={milestone.id} className="milestone-prerequisite"><input type="checkbox" checked={prerequisiteMilestoneIds.includes(milestone.id)} onChange={(event) => setPrerequisiteMilestoneIds((current) => event.target.checked ? [...current, milestone.id] : current.filter((id) => id !== milestone.id))} />{milestone.title}</label>)}</fieldset>}
        <Button type="submit" variant="secondary" disabled={createMutation.isPending} pending={createMutation.isPending} pendingLabel="Adding…">Add milestone</Button>
      </form>
    </section>
  </article>;
}

function MilestoneRow({ milestone, allMilestones, onComplete }: { milestone: Milestone; allMilestones: Milestone[]; onComplete: () => void }) {
  const prerequisiteIds = milestone.prerequisiteMilestoneIds ?? [];
  const prerequisites = prerequisiteIds.map((id) => allMilestones.find((candidate) => candidate.id === id)?.title ?? id);
  const blocked = prerequisiteIds.some((id) => allMilestones.find((candidate) => candidate.id === id)?.status !== "complete");
  return <div className={`milestone-row ${milestone.status === "complete" ? "is-complete" : ""}`}><div><strong>{milestone.title}</strong>{milestone.criteria && <span>{milestone.criteria}</span>}{milestone.dueDate && <small>Due {milestone.dueDate}</small>}{prerequisites.length > 0 && <small>After {prerequisites.join(", ")}</small>}</div>{milestone.status === "complete" ? <span className="milestone-status">Complete</span> : <Button type="button" size="sm" shape="square" onClick={onComplete} disabled={blocked}>{blocked ? "Waiting" : "Mark complete"}</Button>}</div>;
}
