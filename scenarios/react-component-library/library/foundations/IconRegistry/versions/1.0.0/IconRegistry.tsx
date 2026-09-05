/** @vrooliComponentSource foundations.icon-registry */
export type IconName =
  | "check"
  | "close"
  | "chevronDown"
  | "chevronRight"
  | "menu"
  | "search"
  | "plus"
  | "arrowStart"
  | "arrowEnd"
  | "send";
export interface IconDefinition {
  name: IconName;
  viewBox: string;
  path: string;
  directional?: boolean;
}
export const ICON_REGISTRY: Record<IconName, IconDefinition> = {
  check: { name: "check", viewBox: "0 0 24 24", path: "M5 12l4 4L19 6" },
  close: { name: "close", viewBox: "0 0 24 24", path: "M6 6l12 12M18 6L6 18" },
  chevronDown: {
    name: "chevronDown",
    viewBox: "0 0 24 24",
    path: "M6 9l6 6 6-6",
  },
  chevronRight: {
    name: "chevronRight",
    viewBox: "0 0 24 24",
    path: "M9 6l6 6-6 6",
    directional: true,
  },
  menu: { name: "menu", viewBox: "0 0 24 24", path: "M4 7h16M4 12h16M4 17h16" },
  search: {
    name: "search",
    viewBox: "0 0 24 24",
    path: "M11 4a7 7 0 1 0 0 14a7 7 0 1 0 0-14M16 16l4 4",
  },
  plus: { name: "plus", viewBox: "0 0 24 24", path: "M12 5v14M5 12h14" },
  arrowStart: {
    name: "arrowStart",
    viewBox: "0 0 24 24",
    path: "M19 12H5M11 6l-6 6 6 6",
    directional: true,
  },
  arrowEnd: {
    name: "arrowEnd",
    viewBox: "0 0 24 24",
    path: "M5 12h14M13 6l6 6-6 6",
    directional: true,
  },
  send: {
    name: "send",
    viewBox: "0 0 24 24",
    path: "M4 12h15M13 6l6 6-6 6",
    directional: true,
  },
};
export const iconSize = (size: "sm" | "md" | "lg" = "md") =>
  ({
    sm: "var(--icon-size-sm, 1rem)",
    md: "var(--icon-size-md, 1.25rem)",
    lg: "var(--icon-size-lg, 1.5rem)",
  })[size];
