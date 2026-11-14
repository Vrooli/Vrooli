/**
 * Adoption template registry — the code panel's template-selector seam.
 *
 * Kept in its own module (not in AdoptionFileTree.tsx) so the component file
 * exports only a component: mixing component and value exports trips
 * react-refresh/only-export-components and breaks fast refresh.
 */

/**
 * TemplateOption is one entry in the template selector. Each option is a
 * template whose UI manifest can place adopted files. The seam is typed for N
 * templates; today react-vite is the only one that ships a scenario-ui-manifest,
 * so it is the sole entry — adding another is one line here plus its
 * `templates/scenarios/<id>/ui/manifest.json`.
 */
export interface TemplateOption {
  id: string;
  label: string;
}

export const ADOPTION_TEMPLATES: readonly TemplateOption[] = [
  { id: "react-vite", label: "react-vite" },
];

export const DEFAULT_ADOPTION_TEMPLATE = "react-vite";
