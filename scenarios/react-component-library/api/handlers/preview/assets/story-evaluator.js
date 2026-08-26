const sleep = (milliseconds) => new Promise((resolve) => setTimeout(resolve, milliseconds));

const visible = (node, document) => {
  if (!node || node.hasAttribute?.("hidden")) return false;
  const view = document.defaultView || globalThis;
  const style = view.getComputedStyle ? view.getComputedStyle(node) : null;
  const responsiveDesktop = view.innerWidth >= 768
    && node.closest?.('[data-rcl-sidebar-shell][data-mode="responsive"][data-open="false"]');
  if (responsiveDesktop && style?.visibility === "hidden") return true;
  return style?.display !== "none" && style?.visibility !== "hidden";
};

const locate = (document, queries, target) => {
  if (!target || typeof target !== "object") return null;
  if (typeof target.selector === "string") return document.querySelector(target.selector);
  if (typeof target.role !== "string") return null;
  const roleOptions = target.name ? { name: target.name } : {};
  let matches = queries.queryAllByRole(document.body, target.role, roleOptions);
  if (matches.length === 0 && target.role === "complementary") {
    matches = queries.queryAllByRole(document.body, target.role, { ...roleOptions, hidden: true });
  }
  if (matches.length === 0 && target.name) {
    // In the browser preview, the Testing Library module and the iframe DOM
    // can use different realm implementations of accessible-name lookup.
    // Preserve the semantic role requirement, then use the rendered text as
    // a narrow fallback when the semantic query returned no candidate.
    const expectedName = String(target.name).replace(/\s+/g, " ").trim();
    matches = Array.from(document.querySelectorAll("[role]"))
      .filter((candidate) => candidate.getAttribute("role") === target.role)
      .filter((candidate) => String(candidate.textContent || "").replace(/\s+/g, " ").trim() === expectedName);
  }
  return matches[0] || null;
};

const expectationFailure = (document, queries, expectation, env) => {
  if (!expectation || typeof expectation !== "object") return "unsupported expectation";
  if (expectation.kind === "layout" && env.skipKinds?.has("layout")) return { skipped: true, reason: "layout requires a browser layout engine" };
  let node = null;
  if (expectation.kind === "text") {
    const value = String(expectation.value || "");
    const matches = queries.queryAllByText(document.body, value, { exact: false });
    if (matches.length === 0) {
      const normalizedValue = value.replace(/\s+/g, " ").trim();
      const candidates = Array.from(document.body.querySelectorAll("*"))
        .filter((candidate) => visible(candidate, document))
        .filter((candidate) => String(candidate.textContent || "").replace(/\s+/g, " ").includes(normalizedValue))
        .sort((left, right) => String(left.textContent || "").length - String(right.textContent || "").length);
      node = candidates[0] || null;
    } else {
      node = matches[0] || null;
    }
  } else if (expectation.kind === "role") {
    node = locate(document, queries, { role: expectation.role, name: expectation.name });
  } else {
    node = locate(document, queries, { selector: expectation.selector });
  }
  if (expectation.kind === "notVisible") return visible(node, document) ? "expected target not to be visible" : "";
  if (["visible", "text", "role"].includes(expectation.kind)) {
    if (visible(node, document)) return "";
    return "expected target to be visible";
  }
  if (expectation.kind === "attribute") {
    const attribute = expectation.attribute || expectation.name || "";
    if (!node) return "expected attribute value was not found";
    if (expectation.value === undefined) return node.hasAttribute(attribute) ? "" : "expected attribute value was not found";
    return node.getAttribute(attribute) === String(expectation.value) ? "" : "expected attribute value was not found";
  }
  if (expectation.kind === "count") {
    if (!expectation.selector) return "count expectation requires a selector";
    const actual = document.querySelectorAll(expectation.selector).length;
    return actual === Number(expectation.value) ? "" : `expected ${expectation.value} matching elements, received ${actual}`;
  }
  if (expectation.kind === "layout") {
    if (!node) return "expected layout target was not found";
    const rect = node.getBoundingClientRect();
    if (expectation.minWidth !== undefined && rect.width < Number(expectation.minWidth)) return "expected layout width minimum was not met";
    if (expectation.minHeight !== undefined && rect.height < Number(expectation.minHeight)) return "expected layout height minimum was not met";
    if (expectation.maxWidth !== undefined && rect.width > Number(expectation.maxWidth)) return "expected layout width maximum was not met";
    if (expectation.maxHeight !== undefined && rect.height > Number(expectation.maxHeight)) return "expected layout height maximum was not met";
    if (expectation.noOverflow && node.scrollWidth > node.clientWidth + 1) return "expected layout target not to overflow horizontally";
    return "";
  }
  return "unsupported expectation";
};

const dispatchInteraction = (document, target, interaction) => {
  if (interaction.kind === "click") target.click();
  else if (interaction.kind === "focus") target.focus();
  else if (interaction.kind === "blur") target.blur();
  else if (interaction.kind === "key") target.dispatchEvent(new KeyboardEvent("keydown", { key: interaction.text || "", bubbles: true }));
  else if (interaction.kind === "type") {
    const valueSetter = Object.getOwnPropertyDescriptor(Object.getPrototypeOf(target), "value")?.set;
    if (valueSetter) valueSetter.call(target, interaction.text || "");
    else target.value = interaction.text || "";
    target.dispatchEvent(new Event("input", { bubbles: true }));
    target.dispatchEvent(new Event("change", { bubbles: true }));
  }
};

export const browserEnv = Object.freeze({ skipKinds: new Set() });
export const jsdomEnv = Object.freeze({ skipKinds: new Set(["layout"]) });

export async function runStory(previewStory, modules, env = browserEnv) {
  const document = modules.document;
  const queries = env.queries;
  const failures = [];
  const skipped = [];
  const wait = env.wait || sleep;
  const flush = async () => { await env.flush?.(); };
  if (env.browser) {
    // React's browser renderer may commit the first mount after the event
    // loop turn in which renderPreview() was called. A fixed sleep can race
    // that commit and record false expectation failures for otherwise valid
    // stories. Wait for the host-owned sheet to receive its first child,
    // while keeping the bound so a genuinely broken mount still fails fast.
    const deadline = Date.now() + 2000;
    while (Date.now() < deadline) {
      const sheet = document.querySelector("[data-preview-sheet]");
      if (sheet?.children.length) break;
      await wait(20);
      await flush();
    }
    // The first concurrent commit can expose the sheet before its subject's
    // accessible nodes are committed. Wait for the first actionable target
    // (or first declared expectation for static stories) before running role
    // and text queries. This avoids false negatives without hiding a broken
    // mount behind an unbounded wait.
    const initialTarget = previewStory.interactions?.[0]?.target || previewStory.expect?.[0];
    const targetDeadline = Date.now() + 3000;
    while (initialTarget && Date.now() < targetDeadline) {
      let target = null;
      if (initialTarget.kind === "text") {
        const value = String(initialTarget.value || "");
        target = queries.queryAllByText(document.body, value, { exact: false })[0] || null;
      } else {
        target = locate(document, queries, initialTarget);
      }
      if (target && visible(target, document)) break;
      await wait(20);
      await flush();
    }
  }
  await wait(50);
  for (const interaction of previewStory.interactions || []) {
    if (interaction.kind === "settle") { await wait(20); await flush(); continue; }
    if (interaction.kind === "waitFor") {
      const waitText = String(interaction.text || "").trim();
      const deadline = Date.now() + 2000;
      while (waitText && Date.now() < deadline) {
        const visibleText = Array.from(document.querySelectorAll("body *"))
          .filter((node) => visible(node, document))
          .map((node) => String(node.textContent || "").replace(/\s+/g, " ").trim())
          .filter(Boolean);
        if (visibleText.some((value) => value.includes(waitText))) break;
        await wait(20);
        await flush();
      }
      continue;
    }
    let target = locate(document, queries, interaction.target);
    if (!target && previewStory.kind === "hook" && interaction.kind === "click") target = document.querySelector("[data-rcl-hook-action=start]");
    if (!target && previewStory.kind === "hook" && interaction.kind === "focus") target = document.querySelector("[data-rcl-hook-root]");
    if (!target && interaction.kind === "key") target = modules.window || document.defaultView;
    if (!target) { failures.push({ kind: "interaction", message: `target not found for ${interaction.kind}` }); continue; }
    const dispatch = () => {
      if (previewStory.kind === "hook" && interaction.kind === "click") {
        target.dispatchEvent(new MouseEvent("pointerdown", { bubbles: true }));
      }
      dispatchInteraction(document, target, interaction);
    };
    if (env.actInteraction) await env.actInteraction(dispatch);
    else dispatch();
    await flush();
  }
  await wait(50);
  await flush();
  for (const [index, expectation] of (previewStory.expect || []).entries()) {
    const result = expectationFailure(document, queries, expectation, env);
    if (result?.skipped) skipped.push({ index, expectation, reason: result.reason });
    else if (result) failures.push({ kind: "expect", expectation, message: result });
  }
  const result = { passed: failures.length === 0, failures, skipped };
  env.report?.(result.passed, failures, skipped);
  return result;
}
