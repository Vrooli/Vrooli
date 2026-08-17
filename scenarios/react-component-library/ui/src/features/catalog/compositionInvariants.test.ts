/**
 * Calibration for the composition invariants.
 *
 * Each `invalid` case reproduces a defect actually observed in the running
 * workspace on 2026-08-15, so the checks are known to fire on the real thing
 * rather than only on a hypothetical. Each `valid` case is a shape that must
 * stay silent — false positives are what get a check disabled, and a disabled
 * check protects nothing.
 *
 * jsdom does not implement layout, so `getComputedStyle` is stubbed with the
 * values the real browser reports. That is the correct split: the invariant
 * logic is what is under test here, while the real computed values come from
 * the BAS run that injects this module into a live page.
 */
import { describe, expect, it } from "vitest";

import {
  checkLabelUniqueness,
  checkProvidedChromeDuplication,
  checkSpacingQuantization,
  checkTypeScaleMonotonicity,
} from "../../../../bas/invariants/composition-invariants.js";

type StyleMap = Record<string, Record<string, string>>;

/**
 * Build a fake `view` whose getComputedStyle returns per-element overrides
 * keyed by data-testid, falling back to sane visible defaults.
 */
function fakeView(styles: StyleMap) {
  return {
    getComputedStyle(element: Element) {
      const key = element.getAttribute?.("data-testid") ?? "";
      const override = styles[key] ?? {};
      const base: Record<string, string> = {
        display: "block",
        visibility: "visible",
        opacity: "1",
        fontSize: "16px",
        paddingTop: "0px",
        paddingRight: "0px",
        paddingBottom: "0px",
        paddingLeft: "0px",
        rowGap: "0px",
        columnGap: "0px",
        ...override,
      };
      return {
        ...base,
        getPropertyValue(property: string) {
          return base[property] ?? "";
        },
      };
    },
  };
}

function mount(html: string): HTMLElement {
  const host = document.createElement("div");
  host.innerHTML = html;
  document.body.appendChild(host);
  return host;
}

describe("composition invariants", () => {
  describe("type-scale monotonicity", () => {
    it("catches a tab label rendering larger than the title above it", () => {
      // The real defect: Tabs declares `font: inherit`, so in a sidebar whose
      // ambient size exceeds --text-heading-sm it outgrows its own heading.
      const root = mount(`
        <nav data-rcl-navigation-tree>
          <strong data-rcl-navigation-tree-title data-testid="title">Library inventory</strong>
          <button data-testid="tab">Components</button>
        </nav>
      `);
      const findings = checkTypeScaleMonotonicity(
        root,
        fakeView({ title: { fontSize: "14px" }, tab: { fontSize: "16px" } }),
      );
      expect(findings).toHaveLength(1);
      expect(findings[0]?.code).toBe("composition.type_scale_inverted");
      expect(findings[0]?.remediation).toContain("font: inherit");
    });

    it("stays silent when the heading is the largest text in its region", () => {
      const root = mount(`
        <nav data-rcl-navigation-tree>
          <strong data-rcl-navigation-tree-title data-testid="title">Library inventory</strong>
          <button data-testid="tab">Components</button>
        </nav>
      `);
      const findings = checkTypeScaleMonotonicity(
        root,
        fakeView({ title: { fontSize: "18px" }, tab: { fontSize: "14px" } }),
      );
      expect(findings).toEqual([]);
    });

    it("ignores text hidden from the rendered page", () => {
      const root = mount(`
        <nav data-rcl-navigation-tree>
          <strong data-rcl-navigation-tree-title data-testid="title">Inventory</strong>
          <span data-testid="hidden">Enormous</span>
        </nav>
      `);
      const findings = checkTypeScaleMonotonicity(
        root,
        fakeView({ title: { fontSize: "14px" }, hidden: { fontSize: "40px", display: "none" } }),
      );
      expect(findings).toEqual([]);
    });
  });

  describe("label uniqueness", () => {
    it("catches a page drawing its own header beside a component that renders one", () => {
      // The real defect: Sidebar.tsx hand-rolls a "Library inventory" section
      // header immediately above <NavigationTree title="Library inventory">.
      const root = mount(`
        <aside>
          <div data-testid="hand-rolled">Library inventory</div>
          <div data-rcl-navigation-tree>
            <strong data-testid="component-title">Library inventory</strong>
          </div>
        </aside>
      `);
      const findings = checkLabelUniqueness(root, fakeView({}));
      expect(findings).toHaveLength(1);
      expect(findings[0]?.code).toBe("composition.duplicate_label");
      expect(findings[0]?.message).toContain("Library inventory");
    });

    it("allows repeated labels inside an ARIA tree built from divs", () => {
      // Observed live on 2026-08-15: the catalog tree is divs with
      // role="tree"/"treeitem", not list elements, so a tag-only repeated-
      // structure check reported all 309 of its repeated cells as duplicates.
      const root = mount(`
        <nav>
          <div role="tree">
            <div role="treeitem"><span data-testid="a">Draft state</span></div>
            <div role="treeitem"><span data-testid="b">Draft state</span></div>
          </div>
        </nav>
      `);
      expect(checkLabelUniqueness(root, fakeView({}))).toEqual([]);
    });

    it("ignores purely numeric metric cells, which are values rather than labels", () => {
      const root = mount(`
        <nav>
          <div><span data-testid="a">0 · 18↓</span></div>
          <div><span data-testid="b">0 · 18↓</span></div>
        </nav>
      `);
      expect(checkLabelUniqueness(root, fakeView({}))).toEqual([]);
    });

    it("allows repeated labels inside a list, where repetition is the point", () => {
      const root = mount(`
        <nav>
          <ul>
            <li><span data-testid="a">Draft</span></li>
            <li><span data-testid="b">Draft</span></li>
          </ul>
        </nav>
      `);
      expect(checkLabelUniqueness(root, fakeView({}))).toEqual([]);
    });

    it("does not count an sr-only announcement paired with its visible label", () => {
      // Observed live on 2026-08-15: the loading state renders "Loading
      // catalog…" both visibly and as an sr-only announcement. That is the
      // correct accessibility pattern and must not be reported.
      const root = mount(`
        <main>
          <p class="sr-only" data-testid="announce">Loading catalog…</p>
          <p data-testid="visible">Loading catalog…</p>
        </main>
      `);
      expect(checkLabelUniqueness(root, fakeView({}))).toEqual([]);
    });

    it("does not count a wrapper repeating its own child's text", () => {
      const root = mount(`
        <nav><div data-testid="wrap"><span data-testid="inner">Settings</span></div></nav>
      `);
      expect(checkLabelUniqueness(root, fakeView({}))).toEqual([]);
    });
  });

  describe("spacing quantization", () => {
    const ramp = new Map([
      [4, "--space-3xs"],
      [8, "--space-2xs"],
      [12, "--space-xs"],
      [16, "--space-sm"],
    ]);

    it("catches padding outside the ramp's range entirely", () => {
      const root = mount(`<main><div data-testid="row">Item</div></main>`);
      const findings = checkSpacingQuantization(
        root,
        fakeView({ row: { paddingTop: "64px" } }),
        { ramp },
      );
      expect(findings).toHaveLength(1);
      expect(findings[0]?.code).toBe("composition.spacing_off_ramp");
      // The bounding step is named so the fix is unambiguous.
      expect(findings[0]?.remediation).toContain("--space-sm");
    });

    it("allows fluid spacing interpolated between two ramp steps", () => {
      // Observed live on 2026-08-15: app-shell-main computes 38.4px from
      // clamp(var(--space-sm), 3vw, var(--space-xl)). At least six library
      // components use that idiom deliberately, so requiring an exact step
      // match would condemn an intentional, widely-adopted pattern.
      const root = mount(`<main><div data-testid="row">Item</div></main>`);
      const findings = checkSpacingQuantization(
        root,
        fakeView({ row: { paddingTop: "11px", paddingLeft: "13.5px" } }),
        { ramp },
      );
      expect(findings).toEqual([]);
    });

    it("catches spacing smaller than the smallest published step", () => {
      const root = mount(`<main><div data-testid="row">Item</div></main>`);
      const findings = checkSpacingQuantization(
        root,
        fakeView({ row: { rowGap: "1px" } }),
        { ramp },
      );
      expect(findings).toHaveLength(1);
      expect(findings[0]?.message).toContain("outside the published ramp range");
    });

    it("accepts every published ramp step", () => {
      const root = mount(`<main><div data-testid="row">Item</div></main>`);
      const findings = checkSpacingQuantization(
        root,
        fakeView({ row: { paddingTop: "16px", rowGap: "8px" } }),
        { ramp },
      );
      expect(findings).toEqual([]);
    });

    it("ignores zero, which is the absence of spacing rather than an off-ramp amount", () => {
      const root = mount(`<main><div data-testid="row">Item</div></main>`);
      const findings = checkSpacingQuantization(
        root,
        fakeView({ row: { paddingTop: "0px" } }),
        { ramp },
      );
      expect(findings).toEqual([]);
    });
  });

  describe("provided-chrome duplication", () => {
    // The markup below mirrors the ancestor chains a live DOM probe returned on
    // 2026-08-15: the hand-rolled header is `span < div < div < div`, while the
    // component's own title is `strong[data-rcl-navigation-tree-title] < div <
    // nav[data-rcl-navigation-tree] < div`. They converge several levels up,
    // which is why the scope walk is bounded rather than sibling-only.
    const sidebar = `
      <div>
        <div><span data-testid="hand-rolled">Library inventory</span></div>
        <div>
          <nav data-rcl-navigation-tree>
            <div data-rcl-navigation-tree-heading>
              <strong data-rcl-navigation-tree-title data-testid="provided">Library inventory</strong>
            </div>
          </nav>
        </div>
      </div>
    `;

    it("catches a page header duplicating a component's provided title", () => {
      const root = mount(sidebar);
      const findings = checkProvidedChromeDuplication(root, fakeView({}));
      expect(findings).toHaveLength(1);
      expect(findings[0]?.code).toBe("composition.duplicated_provided_chrome");
      // The finding must identify the provider, since that is what tells the
      // reader which of the two elements to delete.
      expect(findings[0]?.message).toContain("navigation-tree");
      expect(findings[0]?.message).toContain("title");
      // It must point at the page's element, never the component's.
      expect(findings[0]?.element).toContain("hand-rolled");
    });

    it("reports the innermost part once when parts nest", () => {
      // navigation-tree's `heading` wraps its `title`; both carry the same text
      // and would otherwise each report the same duplication. The precise part
      // name is what tells the reader which chrome is doubled.
      const root = mount(sidebar);
      const findings = checkProvidedChromeDuplication(root, fakeView({}));
      expect(findings).toHaveLength(1);
      expect(findings[0]?.message).toContain('"title" part');
      expect(findings[0]?.message).not.toContain('"heading" part');
    });

    it("catches a duplicated brand whose text sits in the part's children", () => {
      // Observed live on 2026-08-15: [data-rcl-app-navigation-brand] has no own
      // text — its label is wrapped in spans — so a check reading only the
      // marked element's direct text nodes missed this duplication entirely.
      const root = mount(`
        <div>
          <a><span data-testid="page-brand">Component Library</span></a>
          <div data-rcl-app-navigation>
            <div data-rcl-app-navigation-brand>
              <span aria-hidden="true"></span><span data-testid="provided">Component Library</span>
            </div>
          </div>
        </div>
      `);
      const findings = checkProvidedChromeDuplication(root, fakeView({}));
      expect(findings).toHaveLength(1);
      expect(findings[0]?.element).toContain("page-brand");
    });

    it("does not treat a whole rendered region as a part label", () => {
      // The nested-text fallback is bounded so it never compares a subtree
      // large enough to be page content rather than a label.
      const long = "A rendered region containing considerably more prose than any label would";
      const root = mount(`
        <div>
          <span data-testid="other">${long}</span>
          <div data-rcl-app-navigation>
            <div data-rcl-app-navigation-brand><span>${long}</span></div>
          </div>
        </div>
      `);
      expect(checkProvidedChromeDuplication(root, fakeView({}))).toEqual([]);
    });

    it("stays silent when the page lets the component own its chrome", () => {
      const root = mount(`
        <div>
          <div>
            <nav data-rcl-navigation-tree>
              <strong data-rcl-navigation-tree-title data-testid="provided">Library inventory</strong>
            </nav>
          </div>
        </div>
      `);
      expect(checkProvidedChromeDuplication(root, fakeView({}))).toEqual([]);
    });

    it("ignores the styles marker, which renders nothing", () => {
      const root = mount(`
        <div>
          <span data-testid="other">Library inventory</span>
          <div data-rcl-navigation-tree>
            <style data-rcl-navigation-tree-styles>Library inventory</style>
          </div>
        </div>
      `);
      expect(checkProvidedChromeDuplication(root, fakeView({}))).toEqual([]);
    });

    it("does not reach past the bounded scope to unrelated page regions", () => {
      const root = mount(`
        <div>
          <div><div><div><div><span data-testid="far">Library inventory</span></div></div></div></div>
          <div>
            <nav data-rcl-navigation-tree>
              <strong data-rcl-navigation-tree-title data-testid="provided">Library inventory</strong>
            </nav>
          </div>
        </div>
      `);
      expect(checkProvidedChromeDuplication(root, fakeView({}), { scopeDepth: 1 })).toEqual([]);
    });
  });
});
