import { useQuery } from "@tanstack/react-query";
import { fetchWorkItems } from "../api/work";
import { selectors } from "../consts/selectors";

type Surface = "plan" | "goals" | "focus" | "review";

const copy: Record<Surface, { eyebrow: string; title: string; description: string; empty: string }> = {
  plan: { eyebrow: "Planning desk", title: "Plan", description: "Shape a day you can actually keep.", empty: "Add work from the CLI or the Today surface to begin planning." },
  goals: { eyebrow: "Long view", title: "Goals", description: "Keep the direction visible without turning it into noise.", empty: "Goals are the next domain to connect to this workspace." },
  focus: { eyebrow: "Deep work", title: "Focus", description: "Choose one commitment and give it a protected window.", empty: "Choose a work item from Today to start a focus session." },
  review: { eyebrow: "Learning loop", title: "Review", description: "Use what happened to make the next plan more honest.", empty: "There is not enough completed history to review yet." },
};

export function PlanningSurfacePage({ surface }: { surface: Surface }) {
  const { eyebrow, title, description, empty } = copy[surface];
  const { data: workItems, isLoading, isError } = useQuery({ queryKey: ["work-items"], queryFn: fetchWorkItems });

  return (
    <section className="planner-surface" data-testid={selectors.pages[surface]} aria-labelledby={`${surface}-heading`}>
      <p className="eyebrow">{eyebrow}</p>
      <h1 id={`${surface}-heading`}>{title}</h1>
      <p className="planner-surface-description">{description}</p>
      {isLoading && <p role="status">Loading work items…</p>}
      {isError && <p role="alert">Work data is unavailable right now.</p>}
      {!isLoading && !isError && workItems?.length === 0 && <p className="planner-empty">{empty}</p>}
      {workItems && workItems.length > 0 && (
        <div className="planner-list" aria-label={`${title} work items`}>
          {workItems.map((item) => (
            <article className="planner-list-item" key={item.id}>
              <div><span className="card-kicker">{item.sourceLabel || "Personal Planner"}</span><h2>{item.title}</h2>{item.description && <p>{item.description}</p>}</div>
              <strong>{item.remainingMinutes} min</strong>
            </article>
          ))}
        </div>
      )}
    </section>
  );
}
