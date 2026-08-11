/**
 * @vrooliComponentSource react-component-library:NavigationTree
 * @deps {"react":"^18","lucide-react":"^0.424.0"}
 */
import { FolderTree } from "lucide-react";
import { NavLink } from "../../../NavLink/versions/1.0.0/NavLink";

const navigationTreeStyles = `
[data-rcl-navigation-tree] { display: grid; min-inline-size: 0; gap: var(--space-sm); border: var(--border-hairline) solid var(--color-border); border-radius: var(--radius-panel); background: var(--color-surface); color: var(--color-foreground); padding: var(--space-sm); box-shadow: var(--elev-raised); }
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
}: {
  items?: string[];
  currentIndex?: number;
  title?: string;
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
      </nav>
    </>
  );
}
