/**
 * The semantic experience-surface foundation every generated scenario carries.
 *
 * The implementation is the component library's: it stamps
 * `data-experience-surface` / `data-experience-state` so the readiness contract
 * declared in `experience/pages/*.json` is observable by ui-health and
 * experience-manager. This module exists so the path is stable for tooling and
 * so feature code can import it without knowing the library's version line.
 * Prefer `AsyncPanel/1` for a region that also needs loading, empty, partial
 * and error treatments; it wraps this surface.
 */
export { ExperienceSurface, type ExperienceSurfaceProps, type ExperienceSurfaceState } from "@vrooli/react-component-library/ExperienceSurface/1";
