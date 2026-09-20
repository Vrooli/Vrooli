import { useState } from "react";
import { Target } from "lucide-react";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { Button } from "@vrooli/react-component-library/Button/2";
import { EmptyState } from "@vrooli/react-component-library/EmptyState/1";
import { Input } from "@vrooli/react-component-library/Input/1";
import { PageHeader } from "@vrooli/react-component-library/PageHeader/2";
import { Select } from "@vrooli/react-component-library/Select/1";
import { Textarea } from "@vrooli/react-component-library/Textarea/1";
import { FormField } from "@vrooli/react-component-library/FormField/1";
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
  const activeGoals = goals.data?.filter((goal) => goal.status === "active").length ?? 0;
  const completedGoals = goals.data?.filter((goal) => goal.status === "complete").length ?? 0;
  const milestoneDrivenGoals = goals.data?.filter((goal) => goal.progressMethod === "milestones").length ?? 0;

  return <section className="planner-surface goals-surface" data-testid={selectors.pages.goals} aria-labelledby="goals-heading">
    <PageHeader className="planner-page-header" headingId="goals-heading" eyebrow="Outcomes" title="Goals" description="Name the outcome. Record progress explicitly; time spent is evidence, not achievement." leading={<span className="planner-page-mark" aria-hidden="true"><Target size={21} /></span>} />
    {goals.isError && <p role="alert">Goals are unavailable right now.</p>}
    <div className="goal-overview" aria-label="Goal overview">
      <div className="goal-overview-copy"><span className="card-kicker">A SMALL SET OF TRUE NORTHS</span><h2>Give the week a direction.</h2><p>Goals hold the result you care about. Milestones make the next proof visible; neither turns hours into a promise of achievement.</p></div>
      <dl className="goal-overview-stats"><div><dt>Active</dt><dd>{activeGoals}</dd></div><div><dt>Complete</dt><dd>{completedGoals}</dd></div><div><dt>Milestone-led</dt><dd>{milestoneDrivenGoals}</dd></div></dl>
    </div>
    <form className="goal-create-card" onSubmit={(event) => { event.preventDefault(); mutation.mutate({ title, purpose, progressMethod, targetBasisPoints: 10000n }); }}>
      <div className="goal-create-heading"><div><span className="card-kicker">NEW OUTCOME</span><h2>What should become true?</h2></div><span className="goal-create-note">Start with a name. Detail can come later.</span></div>
      <FormField label="Goal title" required control={<Input aria-label="Goal title" value={title} onChange={(event) => setTitle(event.target.value)} placeholder="A result worth making room for" />} />
      <FormField label="Purpose" control={<Textarea aria-label="Purpose" value={purpose} onChange={(event) => setPurpose(event.target.value)} placeholder="Why does it matter?" rows={3} />} />
      <div className="goal-create-footer"><FormField label="Progress" control={<Select aria-label="Progress" value={progressMethod} onChange={(event) => setProgressMethod(event.target.value)} options={[{ value: "manual", label: "I will record progress" }, { value: "milestones", label: "Derive from milestones" }]} />} /><Button className="primary-action" type="submit" disabled={mutation.isPending || title.trim() === ""} pending={mutation.isPending} pendingLabel="Creating…">Create goal</Button></div>
      {mutation.isError && <p className="goal-form-message" role="alert">That goal could not be saved. Your draft is still here.</p>}
    </form>
    {goals.isLoading && <p role="status">Loading goals…</p>}
    <div className="goal-list">{goals.data?.map((goal) => <GoalCard key={goal.id} goal={goal} progressMutation={progressMutation} />)}</div>
    {!goals.isLoading && !goals.isError && goals.data?.length === 0 && <EmptyState title="No goals yet" description="Create one outcome to give your work a direction." />}
  </section>;
}
