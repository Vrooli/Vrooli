/**
 * @vrooliComponentSource react-component-library:NavigationTree
 * @vrooliComponentVersion 1.0.0
 * @vrooliComponentAdoption 4a5e1190-da32-4358-91b8-d20fc9cf5dfa
 * @vrooliComponentAppliedAt 2026-08-12T11:54:25Z
 * @vrooliComponentSourceSha256 e774d6d5b50730a55d984bf72a12201166da06a10f45a4264a74f4829b15b237
 * @vrooliComponentDriftHash 8e46ea86d4733ff45e5197a2c7773a7d0bd4ee8a9dd03f84d22ba791709bc38e
 * @vrooliComponentTokenTranslation none
 *
 * This file was copied from React Component Library. Local edits are allowed;
 * run "react-component-library adoptions refresh" to inspect drift.
 */
import type { ReactNode } from "react";
import { FolderTree } from "lucide-react";
import { NavLink } from "../../../NavLink/versions/1.0.0/NavLink";

const navigationTreeStyles = `
[data-rcl-navigation-tree] { display: grid; min-inline-size: 0; max-block-size: max(12rem, min(32rem, calc(100dvh - 23rem))); overflow: auto; gap: var(--space-sm); border: var(--border-hairline) solid var(--color-border); border-radius: var(--radius-panel); background: var(--color-surface); color: var(--color-foreground); padding: var(--space-sm); box-shadow: var(--elev-raised); }
[data-rcl-navigation-tree] [data-rcl-navigation-tree-heading] { display: grid; gap: var(--space-3xs); padding-inline: var(--space-2xs); }
[data-rcl-navigation-tree] [data-rcl-navigation-tree-eyebrow] { color: var(--color-primary); font: var(--text-label); letter-spacing: var(--tracking-caps); text-transform: uppercase; }
[data-rcl-navigation-tree] [data-rcl-navigation-tree-title] { font: var(--text-heading-sm); }
[data-rcl-navigation-tree] [data-rcl-navigation-tree-list] { display: grid; gap: var(--space-3xs); margin: 0; padding: 0; list-style: none; }
[data-rcl-navigation-tree] [data-rcl-navigation-tree-item] { min-inline-size: 0; }
`;

export function NavigationTree({
  items = ["Overview", "Activity"],
  currentIndex = 0,
  title = "Workspace",
  children,
}: {
  items?: string[];
  currentIndex?: number;
  title?: string;
  children?: ReactNode;
}) {
  return (
    <>
      <style
        data-rcl-navigation-tree-styles
        dangerouslySetInnerHTML={{ __html: navigationTreeStyles }}
      />
      <nav aria-label="Primary navigation" data-rcl-navigation-tree>
        <div data-rcl-navigation-tree-heading>
          <span data-rcl-navigation-tree-eyebrow>Library</span>
          <strong data-rcl-navigation-tree-title>{title}</strong>
        </div>
        {children ?? (
          <ul data-rcl-navigation-tree-list>
            {items.map((item, index) => (
              <li data-rcl-navigation-tree-item key={`${item}-${String(index)}`}>
                <NavLink
                  label={item}
                  href={`#${encodeURIComponent(item.toLowerCase())}`}
                  current={index === currentIndex}
                  icon={<FolderTree size="var(--space-sm)" strokeWidth={1.8} />}
                />
              </li>
            ))}
          </ul>
        )}
      </nav>
    </>
  );
}
