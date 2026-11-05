import { selectors } from "../consts/selectors";
import { ProjectWorkspace } from "../features/projects/ProjectWorkspace";

/** The local project index is the first product surface. */
export function DashboardPage() {
  return (
    <section data-testid={selectors.pages.dashboard} className="studio-home">
      <ProjectWorkspace />
    </section>
  );
}
