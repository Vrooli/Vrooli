import { AppShell } from "./components/AppShell";
import { HealthCard } from "./features/health/HealthCard";
import { NotesCard } from "./features/notes/NotesCard";

/**
 * App composes the page from a shell + a list of feature cards.
 *
 * Adding a feature: create `features/<name>/<Name>Card.tsx`, then
 * add its import + render line below. Deleting a feature: delete
 * the folder, remove the import + line. There is no central
 * registry to mutate.
 */
export default function App() {
  return (
    <AppShell>
      <HealthCard />
      <NotesCard />
    </AppShell>
  );
}
