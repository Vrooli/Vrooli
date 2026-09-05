import type { BrowserContext, Locator, Page } from 'rebrowser-playwright';

import type { BrowserProfile } from '../types/browser-profile';

export type InteractionState = NonNullable<BrowserProfile['interaction_state']>;

export const INTERACTION_STATE_KEY = Symbol('interactionState');

type InteractionStateContext = BrowserContext & {
  [INTERACTION_STATE_KEY]?: InteractionState;
};

/** A capture cannot claim an interaction state when no real target can hold it. */
export class InteractionStateApplicationError extends Error {
  readonly code = 'INTERACTION_STATE_UNAVAILABLE';

  constructor(
    readonly state: InteractionState,
    reason: string
  ) {
    super(`Unable to apply interaction state "${state}": ${reason}`);
    this.name = 'InteractionStateApplicationError';
  }
}

const INTERACTIVE_SELECTOR =
  'button, [role="button"], a[href], input, select, textarea, [tabindex]:not([tabindex="-1"])';

/**
 * Store the requested state on the browser context so every navigation uses the
 * same applicator. The state is intentionally not represented by a CSS class:
 * the browser must hold the native pseudo-state that the component authored.
 */
export function configureInteractionState(
  context: BrowserContext,
  state: BrowserProfile['interaction_state']
): void {
  const typedContext = context as InteractionStateContext;
  Object.defineProperty(typedContext, INTERACTION_STATE_KEY, {
    value: state ?? 'rest',
    writable: true,
    configurable: true,
  });
}

export function interactionStateForContext(context: BrowserContext): InteractionState {
  return (context as InteractionStateContext)[INTERACTION_STATE_KEY] ?? 'rest';
}

/**
 * Apply the state after navigation and before the capture's DOM/evidence node.
 * The first visible interactive control is the stable target for page captures;
 * component harnesses expose their specimen as that control. Disabled is the
 * exception: it searches only for an already-disabled control and never mutates
 * the page to manufacture evidence.
 */
export async function applyInteractionState(
  page: Page,
  state: InteractionState,
  timeout = 10_000
): Promise<void> {
  if (state === 'rest') return;

  const candidates = page.locator(INTERACTIVE_SELECTOR);

  if (state === 'focus-visible') {
    await applyFocusVisibleState(page, candidates, timeout);
    return;
  }

  const target = await findTarget(candidates, state === 'disabled', timeout);
  if (!target) {
    throw new InteractionStateApplicationError(
      state,
      state === 'disabled'
        ? 'the page has no visible natively-disabled interactive control'
        : 'the page has no visible interactive control'
    );
  }

  try {
    if (state === 'hover') {
      await target.hover({ timeout });
      const held = await target.evaluate((element) => element.matches(':hover'));
      if (!held) throw new Error('the target did not enter :hover');
      return;
    }

    if (state === 'pressed') {
      await target.hover({ timeout });
      await page.mouse.down();
      const held = await target.evaluate((element) => element.matches(':active'));
      if (!held) {
        await page.mouse.up();
        throw new Error('the target did not enter :active while the pointer was held');
      }
      return;
    }

    // Disabled is proven by the element's existing native/ARIA state. No page
    // mutation is permitted because that would validate the driver, not UI.
    const disabled = await target.evaluate(isDisabledElement);
    if (!disabled) throw new Error('the selected control is not natively disabled');
  } catch (error) {
    if (state === 'pressed') await page.mouse.up().catch(() => undefined);
    if (error instanceof InteractionStateApplicationError) throw error;
    throw new InteractionStateApplicationError(
      state,
      error instanceof Error ? error.message : String(error)
    );
  }
}

async function applyFocusVisibleState(page: Page, candidates: Locator, timeout: number): Promise<void> {
  await waitForAnyCandidate(candidates, timeout);
  const count = await candidates.count();
  const visibleCandidates: Locator[] = [];
  for (let index = 0; index < count; index += 1) {
    const candidate = candidates.nth(index);
    if (!(await candidate.isVisible().catch(() => false))) continue;
    if (!(await candidate.boundingBox().catch(() => null))) continue;
    await candidate.waitFor({ state: 'visible', timeout });
    visibleCandidates.push(candidate);
  }

  // Keyboard navigation, rather than locator.focus(), is required so the
  // browser's :focus-visible heuristic is the thing being tested. Each Tab
  // advances the native focus order until the declared target is reached.
  // The extra pass handles a document whose current focus starts after the
  // first candidate (for example, a harness bridge restoring focus).
  await page.bringToFront?.();
  let lastFocusObservation = 'no focus observation';
  for (let attempt = 0; attempt <= visibleCandidates.length; attempt += 1) {
    await page.keyboard.press('Tab');
    for (const candidate of visibleCandidates) {
      const observation: unknown = await candidate.evaluate((element) => ({
        active: document.activeElement === element,
        focusVisible: element.matches(':focus-visible'),
        activeTag: document.activeElement?.tagName ?? 'none',
      }));
      if (observation === true) return;
      if (observation && typeof observation === 'object') {
        const focus = observation as {
          active?: boolean;
          focusVisible?: boolean;
          activeTag?: string;
        };
        lastFocusObservation = `active=${String(focus.active)} focus-visible=${String(focus.focusVisible)} activeTag=${focus.activeTag ?? 'unknown'}`;
        if (focus.active && focus.focusVisible) return;
      }
    }
  }

  throw new InteractionStateApplicationError(
    'focus-visible',
    `keyboard focus did not produce :focus-visible on a visible interactive control (${lastFocusObservation})`
  );
}

function isDisabledElement(element: Element): boolean {
  const nativeDisabled =
    (element instanceof HTMLButtonElement ||
      element instanceof HTMLInputElement ||
      element instanceof HTMLSelectElement ||
      element instanceof HTMLTextAreaElement) &&
    element.disabled;
  return nativeDisabled || element.getAttribute('aria-disabled') === 'true';
}

async function findTarget(
  candidates: Locator,
  requireDisabled: boolean,
  timeout: number
): Promise<Locator | undefined> {
  await waitForAnyCandidate(candidates, timeout);
  const count = await candidates.count();
  for (let index = 0; index < count; index += 1) {
    const candidate = candidates.nth(index);
    if (!(await candidate.isVisible().catch(() => false))) continue;
    if (!(await candidate.boundingBox().catch(() => null))) continue;
    if (requireDisabled) {
      const disabled = await candidate.evaluate(isDisabledElement);
      if (!disabled) continue;
    }
    await candidate.waitFor({ state: 'visible', timeout });
    return candidate;
  }
  return undefined;
}

async function waitForAnyCandidate(candidates: Locator, timeout: number): Promise<void> {
  await candidates
    .first()
    .waitFor({ state: 'visible', timeout })
    .catch(() => undefined);
}
