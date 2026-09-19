import { useState } from "react";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { Button } from "@vrooli/react-component-library/Button/2";
import { EmptyState } from "@vrooli/react-component-library/EmptyState/1";
import { Input } from "@vrooli/react-component-library/Input/1";
import { createGoal, fetchGoals, updateGoalProgress } from "../api/goals";
import { GoalCard } from "../components/GoalCard";
import { selectors } from "../consts/selectors";

export function GoalsPage() {
  const queryClient = useQueryClient();
  const goals = useQuery({ queryKey: ["goals"], queryFn: fetchGoals });
  const [title, setTitle] = useState("");
  const [purpose, setPurpose] = useState("");
  const [progressMethod, setProgressMethod] = useState("manual");
  const mutation = useMutation({ mutationFn: createGoal, onSuccess: async () => { setTitle(""); setPurpose(""); setProgressMethod("manual"); await queryClient.invalidateQueries({ queryKey: ["goals"] }); } });
  const progressMutation = useMutation({ mutationFn: ({ goal, progress }: { goal: Parameters<typeof updateGoalProgress>[0]; progress: bigint }) => updateGoalProgress(goal, progress), onSuccess: async () => { await queryClient.invalidateQueries({ queryKey: ["goals"] }); } });

  return <section className="planner-surface goals-surface" data-testid={selectors.pages.goals} aria-labelledby="goals-heading">
    <p className="eyebrow">Outcomes</p><h1 id="goals-heading">Goals</h1>
    <p className="planner-surface-description">Name the outcome. Record progress explicitly; time spent is evidence, not achievement.</p>
    {goals.isError && <p role="alert">Goals are unavailable right now.</p>}
    <form className="goal-create-card" onSubmit={(event) => { event.preventDefault(); mutation.mutate({ title, purpose, progressMethod, targetBasisPoints: 10000n }); }}>
      <label>Goal title<Input value={title} onChange={(event) => setTitle(event.target.value)} placeholder="What should become true?" required /></label>
      <label>Purpose<textarea value={purpose} onChange={(event) => setPurpose(event.target.value)} placeholder="Why does it matter?" rows={3} /></label>
      <label>Progress<select value={progressMethod} onChange={(event) => setProgressMethod(event.target.value)}><option value="manual">I will record progress</option><option value="milestones">Derive from milestones</option></select></label>
      <Button className="primary-action" type="submit" disabled={mutation.isPending} pending={mutation.isPending} pendingLabel="Creating…">Create goal</Button>
    </form>
    {goals.isLoading && <p role="status">Loading goals…</p>}
    <div className="goal-list">{goals.data?.map((goal) => <GoalCard key={goal.id} goal={goal} progressMutation={progressMutation} />)}</div>
    {!goals.isLoading && !goals.isError && goals.data?.length === 0 && <EmptyState title="No goals yet" description="Create one outcome to give your work a direction." />}
  </section>;
}
