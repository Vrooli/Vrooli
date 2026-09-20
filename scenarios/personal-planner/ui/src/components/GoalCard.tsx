import { useEffect, useState } from "react";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { Button } from "@vrooli/react-component-library/Button/2";
import { Checkbox } from "@vrooli/react-component-library/Checkbox/1";
import { EmptyState } from "@vrooli/react-component-library/EmptyState/1";
import { Input } from "@vrooli/react-component-library/Input/1";
import { Select } from "@vrooli/react-component-library/Select/1";
import { FormField } from "@vrooli/react-component-library/FormField/1";
import { Slider } from "@vrooli/react-component-library/Slider/1.2.4";
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
  const [progress, setProgress] = useState(() => Number(goal.progressBasisPoints));
  useEffect(() => setProgress(Number(goal.progressBasisPoints)), [goal.progressBasisPoints]);
  const createMutation = useMutation({ mutationFn: createMilestone, onSuccess: async () => { setTitle(""); setCriteria(""); setDueDate(""); setLinkedWorkItemId(""); setPrerequisiteMilestoneIds([]); await queryClient.invalidateQueries({ queryKey: ["milestones", goal.id] }); await queryClient.invalidateQueries({ queryKey: ["goals"] }); } });
  const completeMutation = useMutation({ mutationFn: completeMilestone, onSuccess: async () => { await queryClient.invalidateQueries({ queryKey: ["milestones", goal.id] }); } });
  return <article className="goal-card">
    <div className="goal-card-main"><div className="goal-card-copy"><div className="goal-card-meta"><span className="card-kicker">{goal.status} · {goal.progressMethod}</span><span>{milestones.data ? `${milestones.data.filter((m) => m.status === "complete").length}/${milestones.data.length} proofs complete` : "Proofs loading"}</span></div><h2>{goal.title}</h2><p>{goal.purpose || "No purpose recorded yet."}</p></div><div className="goal-progress"><FormField label="Progress" control={<Slider aria-label="Progress" min={0} max={10000} step={100} value={progress} onChange={setProgress} onChangeCommit={(value) => progressMutation.mutate({ goal, progress: BigInt(Math.round(value)) })} formatValue={(value) => `${Math.round(value / 100)}% complete`} showValue="none" />} /><output aria-label={`${Math.round(progress / 100)}% complete`}>{Math.round(progress / 100)}%</output><div className="goal-progress-meter" aria-hidden="true"><span style={{ inlineSize: `${Math.max(0, Math.min(100, progress / 100))}%` }} /></div></div></div>
    <section className="milestone-section" aria-labelledby={`milestones-${goal.id}`}>
      <div className="milestone-heading"><div><span className="card-kicker">Next proof</span><h3 id={`milestones-${goal.id}`}>Milestones</h3></div><span className="milestone-count">{milestones.data?.filter((m) => m.status === "complete").length ?? 0}/{milestones.data?.length ?? 0} complete</span></div>
      {milestones.isError && <p role="alert">Milestones are unavailable right now.</p>}
      <div className="milestone-list">{milestones.data?.length === 0 && <EmptyState className="planner-empty-state" title="No milestones yet" description="Add one proof point to make this outcome observable." />}{milestones.data?.map((milestone) => <MilestoneRow key={milestone.id} milestone={milestone} allMilestones={milestones.data ?? []} onComplete={() => completeMutation.mutate(milestone)} />)}</div>
      <form className="milestone-form" onSubmit={(event) => { event.preventDefault(); createMutation.mutate({ goalId: goal.id, title, criteria, dueDate, linkedWorkItemId, prerequisiteMilestoneIds }); }}>
        <FormField label="Milestone title" required control={<Input aria-label="Milestone title" value={title} onChange={(event) => setTitle(event.target.value)} placeholder="A proof point" />} />
        <FormField label="Completion criteria" control={<Input aria-label="Completion criteria" value={criteria} onChange={(event) => setCriteria(event.target.value)} placeholder="What makes it true?" />} />
        <FormField label="Due date" control={<Input aria-label="Due date" type="date" value={dueDate} onChange={(event) => setDueDate(event.target.value)} />} />
        <FormField label="Linked work" control={<Select aria-label="Linked work" value={linkedWorkItemId} onChange={(event) => setLinkedWorkItemId(event.target.value)} options={[{ value: "", label: "No work linked" }, ...(work.data?.map((item) => ({ value: item.id, label: item.title })) ?? [])]} />} />
        {milestones.data && milestones.data.length > 0 && <fieldset><legend>Prerequisites</legend>{milestones.data.filter((milestone) => milestone.status !== "complete").map((milestone) => <Checkbox key={milestone.id} className="milestone-prerequisite" label={milestone.title} checked={prerequisiteMilestoneIds.includes(milestone.id)} onCheckedChange={(checked) => setPrerequisiteMilestoneIds((current) => checked ? [...current, milestone.id] : current.filter((id) => id !== milestone.id))} />)}</fieldset>}
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
