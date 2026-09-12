import { memo } from "react";

interface ProjectListProps {
  projects: string[];
}

// ProjectList renders the project rows. Intentionally the fixture hot component
// that the analysis symbol locator must resolve to this file:line.
export const ProjectList = memo(function ProjectListImpl({ projects }: ProjectListProps) {
  return (
    <ul>
      {projects.map((p) => (
        <li key={p}>{p}</li>
      ))}
    </ul>
  );
});
