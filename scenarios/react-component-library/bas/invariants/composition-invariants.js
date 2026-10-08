/**
 * Composition invariants — defects that only exist once components are
 * assembled into a page.
 *
 * Why these are not component gates. Every check here passes trivially when a
 * component is rendered alone in its harness, because the defect is a relation
 * between a component and its surroundings:
 *
 *   - The Tabs primitive declares `font: inherit`. In its own harness the
 *     ambient size is the harness's and the type scale looks correct. Mounted
 *     in a sidebar whose ambient size exceeds --text-heading-sm, its labels
 *     render larger than the title above them. No per-component gate can
 *     observe this, because in isolation there is no "above".
 *
 *   - NavigationTree renders its own title, and AppNavigation renders its own
 *     brand. Both are correct. A page that draws a second title beside the
 *     first produces a duplicated label that neither component can detect.
 *
 * These run against a rendered page and read computed styles, so they execute
 * in a browser (BAS `evaluate`) rather than in the static gate runners. The
 * logic is kept free of browser-only globals beyond `getComputedStyle` and the
 * passed-in root so it can be unit-tested against a stubbed document.
 *
 * Findings mirror the Go gates.Finding shape on purpose: `message` states what
 * is wrong, `remediation` states what to do. A checker that reports a defect
 * without its fix hands the reader a puzzle.
 */

/** Elements that never carry visible page text. */
const NON_TEXT_TAGS = new Set(["SCRIPT", "STYLE", "NOSCRIPT", "TEMPLATE", "SVG", "PATH"]);

/**
 * Containers whose children are legitimately repetitive.
 *
 * Both a tag set and a role set are needed. The catalog tree is built from
 * divs carrying role="tree"/"treeitem" rather than list elements, so a
 * tag-only check reported all 309 of its repeated metric cells as duplicated
 * labels on the first live run.
 */
const REPEATED_STRUCTURE = new Set(["UL", "OL", "TABLE", "TBODY", "SELECT", "DATALIST", "MENU"]);
const REPEATED_ROLES = new Set([
  "list", "listbox", "tree", "treegrid", "grid", "table", "rowgroup", "row",
  "menu", "menubar", "tablist", "feed", "group",
]);

/**
 * Text that is purely numeric or symbolic is a value, not a label. Values
 * recur legitimately — two rows both reporting "0" is data, not duplicated
 * chrome — so only prose-bearing text participates in the uniqueness check.
 */
function isValueLike(text) {
  return !/[a-z]{2}/i.test(text);
}

/** Region roots. A "region" is the scope within which a label must be unique. */
const REGION_SELECTOR =
  "[data-experience-surface],[data-rcl-navigation-tree],[data-rcl-app-navigation],header,nav,aside,main,section[aria-label],[role='region'],[role='navigation']";

function isVisible(element, view) {
  const style = view.getComputedStyle(element);
  if (style.display === "none" || style.visibility === "hidden" || style.opacity === "0") return false;
  return true;
}

/**
 * Screen-reader-only text is clipped to a 1px box rather than hidden, so it is
 * "visible" by the test above while contributing nothing to the layout. Pairing
 * a visible label with an sr-only announcement of the same words is the correct
 * accessibility pattern, not duplicated chrome — counting it as a duplicate
 * would penalise exactly the thing we want authors to do.
 */
function isScreenReaderOnly(element, view) {
  if (element.classList?.contains("sr-only") || element.classList?.contains("visually-hidden")) return true;
  const style = view.getComputedStyle(element);
  const width = Number.parseFloat(style.width);
  const height = Number.parseFloat(style.height);
  const clipped = style.clip === "rect(0px, 0px, 0px, 0px)" || style.clipPath === "inset(50%)";
  return clipped || (Number.isFinite(width) && width <= 1 && Number.isFinite(height) && height <= 1);
}

/** Direct text of an element, excluding text contributed by descendants. */
function ownText(element) {
  let text = "";
  for (const node of element.childNodes) {
    if (node.nodeType === 3) text += node.nodeValue;
  }
  return text.replace(/\s+/g, " ").trim();
}

function parseFontSize(value) {
  const parsed = Number.parseFloat(value);
  return Number.isFinite(parsed) ? parsed : 0;
}

function describe(element) {
  const id = element.id ? `#${element.id}` : "";
  const testid = element.getAttribute?.("data-testid");
  const cls = (element.className && typeof element.className === "string" ? element.className : "")
    .split(/\s+/)
    .filter(Boolean)
    .slice(0, 2)
    .map((c) => `.${c}`)
    .join("");
  return `${element.tagName.toLowerCase()}${id}${testid ? `[data-testid="${testid}"]` : ""}${cls}`;
}

function elements(root) {
  return Array.from(root.querySelectorAll("*")).filter((el) => !NON_TEXT_TAGS.has(el.tagName));
}

/**
 * Invariant 1 — type-scale monotonicity.
 *
 * Within the region a heading introduces, no text may render larger than that
 * heading. A larger descendant inverts the reading order the heading declares:
 * the eye lands on the subordinate text first, and the page reads as if the
 * label were the title.
 */
export function checkTypeScaleMonotonicity(root, view) {
  const findings = [];
  const headings = Array.from(
    root.querySelectorAll("h1,h2,h3,h4,h5,h6,[data-rcl-navigation-tree-title],[role='heading']"),
  );
  for (const heading of headings) {
    if (!isVisible(heading, view)) continue;
    const headingSize = parseFontSize(view.getComputedStyle(heading).fontSize);
    if (headingSize <= 0) continue;
    const region = heading.closest(REGION_SELECTOR) ?? heading.parentElement;
    if (!region) continue;
    for (const candidate of elements(region)) {
      if (candidate === heading || heading.contains(candidate)) continue;
      if (!ownText(candidate) || !isVisible(candidate, view)) continue;
      const size = parseFontSize(view.getComputedStyle(candidate).fontSize);
      if (size <= headingSize) continue;
      findings.push({
        code: "composition.type_scale_inverted",
        message: `${describe(candidate)} renders at ${size}px, larger than the ${headingSize}px heading "${ownText(heading).slice(0, 40)}" that introduces its region`,
        element: describe(candidate),
        remediation:
          "Give this element a type-scale step at or below its heading's, or promote it to the heading if it is genuinely the more important label. A common cause is a component that declares `font: inherit` and therefore has no size of its own — it renders at whatever the surrounding context happens to be, which is correct in its own harness and wrong here.",
        docsRef: "docs/concepts/ARCHITECTURE.md#design-tokens",
      });
    }
  }
  return findings;
}

/**
 * Invariant 2 — label uniqueness within a region.
 *
 * The same short label twice in one region is almost always duplicated chrome:
 * a page drawing its own header beside a component that already renders one.
 * Repeated structures (lists, tables) are excluded because repetition is their
 * entire purpose.
 */
export function checkLabelUniqueness(root, view) {
  const findings = [];
  const regions = Array.from(root.querySelectorAll(REGION_SELECTOR));
  for (const region of regions) {
    const seen = new Map();
    for (const candidate of elements(region)) {
      const text = ownText(candidate);
      if (!text || text.length > 40 || !isVisible(candidate, view)) continue;
      if (isScreenReaderOnly(candidate, view)) continue;
      if (isValueLike(text)) continue;
      // A label inside a repeated structure is expected to recur.
      let ancestor = candidate.parentElement;
      let repeated = false;
      while (ancestor && ancestor !== region) {
        const role = ancestor.getAttribute?.("role") ?? "";
        if (REPEATED_STRUCTURE.has(ancestor.tagName) || REPEATED_ROLES.has(role)) {
          repeated = true;
          break;
        }
        ancestor = ancestor.parentElement;
      }
      if (repeated) continue;
      const key = text.toLowerCase();
      const previous = seen.get(key);
      if (!previous) {
        seen.set(key, candidate);
        continue;
      }
      // Nested duplicates (a wrapper repeating its child's text) are one label.
      if (previous.contains(candidate) || candidate.contains(previous)) continue;
      findings.push({
        code: "composition.duplicate_label",
        message: `"${text}" appears twice in region ${describe(region)} — at ${describe(previous)} and ${describe(candidate)}`,
        element: describe(candidate),
        remediation:
          "Remove one of the two. This usually means the page hand-rolls chrome around a component that already provides it: check whether the composed component renders its own title, brand, or heading before adding one beside it. Both elements are individually correct, which is why neither component's own tests can catch this.",
        docsRef: "docs/concepts/ARCHITECTURE.md#catalog-graph-projection",
      });
    }
  }
  return findings;
}

/** Read the published spacing ramp from the document's custom properties. */
export function readSpacingRamp(root, view) {
  const host = root.ownerDocument?.documentElement ?? root;
  const style = view.getComputedStyle(host);
  const ramp = new Map();
  for (const step of ["4xs", "3xs", "2xs", "xs", "sm", "md", "lg", "xl", "2xl", "3xl", "4xl"]) {
    const raw = style.getPropertyValue(`--space-${step}`).trim();
    if (!raw) continue;
    const px = Number.parseFloat(raw);
    if (Number.isFinite(px)) ramp.set(px, `--space-${step}`);
  }
  return ramp;
}

const SPACING_PROPERTIES = [
  "paddingTop", "paddingRight", "paddingBottom", "paddingLeft",
  "rowGap", "columnGap",
];

/**
 * Invariant 3 — spacing stays within the ramp's range.
 *
 * The first draft of this check required every computed value to equal a
 * published step exactly. Run against the live workspace it flagged
 * `app-shell-main`'s 38.4px padding — which turned out to be
 * `clamp(var(--space-sm), 3vw, var(--space-xl))`, a deliberate fluid-spacing
 * idiom used by AppShell, Page, PageFrame, FilterBar, MasterDetail, and
 * CommandPalette. Interpolating between two ramp steps is the whole point of
 * that idiom, so exact-match was measuring the wrong thing: it would have
 * condemned an intentional, widely-adopted pattern across released components.
 *
 * What a computed value can still honestly prove is that spacing is bounded by
 * the ramp. A value inside [smallest step, largest step] is either a step or an
 * interpolation between steps, and both are legitimate. A value outside that
 * range cannot have come from the ramp at all — it is an arbitrary constant,
 * which is the defect this check can detect without guessing at intent.
 *
 * Distinguishing "interpolated between steps" from "arbitrary but coincidentally
 * in range" is not possible from computed styles alone; that discrimination
 * belongs to the source-level `design-system/no-raw-dimensions` lint, which can
 * see the authored value. The two checks are deliberately complementary rather
 * than redundant.
 */
export function checkSpacingQuantization(root, view, options = {}) {
  const ramp = options.ramp ?? readSpacingRamp(root, view);
  if (ramp.size === 0) return [];
  const steps = [...ramp.keys()].sort((a, b) => a - b);
  const smallest = steps[0];
  const largest = steps[steps.length - 1];
  const findings = [];
  const reported = new Set();
  for (const candidate of elements(root)) {
    if (!isVisible(candidate, view)) continue;
    const style = view.getComputedStyle(candidate);
    for (const property of SPACING_PROPERTIES) {
      const px = Number.parseFloat(style[property]);
      if (!Number.isFinite(px) || px === 0) continue;
      if (px >= smallest && px <= largest) continue;
      const key = `${describe(candidate)}:${property}:${px}`;
      if (reported.has(key)) continue;
      reported.add(key);
      const nearest = px < smallest ? smallest : largest;
      findings.push({
        code: "composition.spacing_off_ramp",
        message: `${describe(candidate)} computes ${property} to ${px}px, outside the published ramp range ${smallest}px–${largest}px`,
        element: describe(candidate),
        remediation: `Use ${ramp.get(nearest)} (${nearest}px) or another published step. A value outside the ramp's range cannot have come from a token, so it is a hard-coded constant somewhere in the cascade. Because this is measured after the cascade the raw number may not appear in this element's class list — check the component's own defaults and any wrapper overriding them. Values that fall between two steps are not reported: interpolating with clamp() is a deliberate idiom here.`,
        docsRef: "docs/concepts/ARCHITECTURE.md#design-tokens",
      });
    }
  }
  return findings;
}

/**
 * Invariant 4 — a page must not hand-roll chrome a composed component already
 * provides.
 *
 * This is the root cause behind both duplications observed in the workspace.
 * `Sidebar.tsx` draws its own "Library inventory" header immediately above
 * `<NavigationTree title="Library inventory">`, which renders its own; it does
 * the same with the brand, drawing one beside `<AppNavigation brand=…>`. Each
 * component is individually correct and each page element is individually
 * correct — the defect exists only in their arrangement, which is why no
 * component gate and no unit test can reach it.
 *
 * The declaration this needs already exists in two places and required no new
 * schema. Components mark the chrome they render with `data-rcl-<slug>-<part>`
 * attributes, and the catalog asset's `api.parts` names those parts, which the
 * `api` gate already enforces against the implementation source. So the
 * rendered page carries its own manifest of what each component provides.
 *
 * Detection: for every element carrying a part marker, search the component
 * root's nearby ancestors for another element with the same visible text that
 * sits OUTSIDE the component. Nearby is bounded (default three levels above the
 * component root) so this reports a neighbour drawing duplicate chrome rather
 * than any coincidence elsewhere on the page.
 *
 * This overlaps invariant 2 deliberately but is not redundant: invariant 2 sees
 * "this text appears twice", while this one can say which component provides it
 * and therefore which of the two elements is the one to delete.
 */
/** Attribute names that mark structure rather than a rendered part. */
const NON_PART_SUFFIXES = new Set(["styles"]);

/**
 * Slug and part cannot be recovered by parsing an attribute name: both are
 * hyphenated, so `data-rcl-navigation-tree-title` is equally readable as
 * slug "navigation" + part "tree-title". The component root is the authority —
 * it carries the bare slug — so roots are enumerated first and parts are found
 * as descendants whose attribute extends the root's.
 */
export function checkProvidedChromeDuplication(root, view, options = {}) {
  const scopeDepth = options.scopeDepth ?? 3;
  const findings = [];
  const reported = new Set();
  for (const componentRoot of elements(root)) {
    for (const attribute of Array.from(componentRoot.attributes ?? [])) {
      if (!attribute.name.startsWith("data-rcl-")) continue;
      const slug = attribute.name.slice("data-rcl-".length);
      if (!slug || NON_PART_SUFFIXES.has(slug)) continue;
      const partPrefix = `${attribute.name}-`;

      for (const partElement of elements(componentRoot)) {
        const partAttribute = Array.from(partElement.attributes ?? []).find((a) =>
          a.name.startsWith(partPrefix),
        );
        if (!partAttribute) continue;
        const part = partAttribute.name.slice(partPrefix.length);
        if (NON_PART_SUFFIXES.has(part)) continue;
        // A part's text often sits in a child rather than on the marked element
        // itself — AppNavigation's brand wraps its label in spans, so its own
        // text is empty. Fall back to the subtree's text, bounded to label
        // length so this never compares a whole rendered region.
        const nested = partElement.textContent.replace(/\s+/g, " ").trim();
        const text = ownText(partElement) || (nested.length <= 40 ? nested : "");
        if (!text || !isVisible(partElement, view) || isScreenReaderOnly(partElement, view)) continue;

        // Parts nest: navigation-tree's `heading` wraps its `title`, so both
        // carry the same text and would each report the same duplication. The
        // innermost part is the precise one — it names the specific chrome the
        // page is doubling — so an outer part yields to it.
        const innerPart = elements(partElement).some((inner) => {
          const innerAttribute = Array.from(inner.attributes ?? []).find((a) =>
            a.name.startsWith(partPrefix),
          );
          if (!innerAttribute) return false;
          const innerText = inner.textContent.replace(/\s+/g, " ").trim();
          return innerText.toLowerCase() === text.toLowerCase();
        });
        if (innerPart) continue;

        // Walk a bounded number of levels above the component root looking for
        // a neighbouring subtree that repeats the same words.
        let scope = componentRoot.parentElement;
        let duplicate = null;
        for (let level = 0; level < scopeDepth && scope && !duplicate; level += 1) {
          for (const other of elements(scope)) {
            if (componentRoot.contains(other)) continue;
            if (!isVisible(other, view) || isScreenReaderOnly(other, view)) continue;
            if (ownText(other).toLowerCase() !== text.toLowerCase()) continue;
            duplicate = other;
            break;
          }
          scope = scope.parentElement;
        }
        if (!duplicate) continue;

        const key = `${slug}:${part}:${text.toLowerCase()}`;
        if (reported.has(key)) continue;
        reported.add(key);
        findings.push({
          code: "composition.duplicated_provided_chrome",
          message: `${describe(duplicate)} renders "${text}", which the composed ${slug} component already provides as its "${part}" part`,
          element: describe(duplicate),
          remediation: `Delete ${describe(duplicate)} and let the ${slug} component render its own ${part}; pass the value through the component's prop if it needs to differ. The component declares this part in its catalog api.parts, so it is committed to rendering it — a page that draws one beside it produces two, and the one to remove is always the page's.`,
          docsRef: "docs/concepts/ARCHITECTURE.md#catalog-graph-projection",
        });
      }
    }
  }
  return findings;
}

/** Run every invariant against a rendered root. */
export function checkComposition(root, view) {
  return [
    ...checkTypeScaleMonotonicity(root, view),
    ...checkLabelUniqueness(root, view),
    ...checkSpacingQuantization(root, view),
    ...checkProvidedChromeDuplication(root, view),
  ];
}
