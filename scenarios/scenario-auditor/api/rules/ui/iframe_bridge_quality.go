package ui

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"sync"

	rules "scenario-auditor/rules"
)

/*
Rule: Scenario UI Bridge Quality
Description: Validates that UI bridge files match the canonical implementation and entry files bootstrap the bridge safely
Reason: The iframe bridge enables lifecycle orchestration, so scenarios must ship the canonical implementation and initialize it correctly
Category: ui
Severity: high
Standard: ui-bridge-v1
Targets: ui

<test-case id="bridge-ts-valid" should-fail="false" path="testdata/bridge-ts-valid/ui/src/iframeBridgeChild.ts">
  <description>Canonical TypeScript bridge passes without violations</description>
  <input language="typescript"><![CDATA[
export type BridgeCapability = 'history' | 'hash' | 'title' | 'deeplink' | 'resize';

export interface BridgeChildOptions {
  parentOrigin?: string;
  appId?: string;
  onNav?: (href: string) => void;
}

export interface BridgeChildController {
  notify: () => void;
  dispose: () => void;
}

declare global {
  interface Window {
    __vrooliBridgeChildInstalled?: boolean;
  }
}

const inferParentOrigin = (): string | null => {
  try {
    if (document.referrer) {
      const referrer = new URL(document.referrer);
      return referrer.origin;
    }
  } catch (error) {
    console.warn('[BridgeChild] Failed to parse document.referrer', error);
  }
  return null;
};

const buildLocationPayload = () => ({
  v: 1 as const,
  t: 'LOCATION' as const,
  href: window.location.href,
  path: `${window.location.pathname}${window.location.search}${window.location.hash}`,
  title: document.title,
  canGoBack: true,
  canGoFwd: true,
});

export function initIframeBridgeChild(options: BridgeChildOptions = {}): BridgeChildController {
  if (typeof window === 'undefined' || typeof document === 'undefined') {
    return {
      notify: () => undefined,
      dispose: () => undefined,
    };
  }

  if (window.parent === window) {
    return {
      notify: () => undefined,
      dispose: () => undefined,
    };
  }

  if (window.__vrooliBridgeChildInstalled) {
    return {
      notify: () => undefined,
      dispose: () => undefined,
    };
  }

  const caps: BridgeCapability[] = ['history', 'hash', 'title', 'deeplink'];
  let resolvedOrigin = options.parentOrigin ?? inferParentOrigin() ?? '*';

  const post = (payload: Record<string, unknown>) => {
    try {
      window.parent.postMessage(payload, resolvedOrigin);
    } catch (error) {
      console.warn('[BridgeChild] postMessage failed', error);
    }
  };

  const notify = () => {
    const payload = buildLocationPayload();
    post(payload);
    options.onNav?.(payload.href);
  };

  const handleMessage = (event: MessageEvent) => {
    if (resolvedOrigin !== '*' && event.origin !== resolvedOrigin) {
      return;
    }
    if (resolvedOrigin === '*' && event.origin) {
      resolvedOrigin = event.origin;
    }

    const message = event.data;
    if (!message || typeof message !== 'object' || message.v !== 1) {
      return;
    }

    if (message.t === 'NAV') {
      try {
        if (message.cmd === 'BACK') {
          history.back();
        } else if (message.cmd === 'FWD') {
          history.forward();
        } else if (message.cmd === 'GO' && typeof message.to === 'string') {
          const resolved = new URL(message.to, window.location.href);
          if (resolved.origin !== window.location.origin) {
            window.location.assign(resolved.href);
            return;
          }
          history.pushState({}, '', `${resolved.pathname}${resolved.search}${resolved.hash}`);
          window.dispatchEvent(new PopStateEvent('popstate', { state: history.state }));
        }
        notify();
      } catch (error) {
        post({ v: 1, t: 'ERROR', code: 'NAV_FAILED', detail: String((error as Error)?.message ?? error) });
      }
    } else if (message.t === 'PING' && typeof message.ts === 'number') {
      post({ v: 1, t: 'PONG', ts: message.ts });
    }
  };

  const interceptHistory = () => {
    const originalPush = history.pushState;
    const originalReplace = history.replaceState;

    history.pushState = function pushState(...args) {
      originalPush.apply(history, args as any);
      notify();
    };

    history.replaceState = function replaceState(...args) {
      originalReplace.apply(history, args as any);
      notify();
    };
  };

  const setupObservers = () => {
    window.addEventListener('message', handleMessage);
    window.addEventListener('popstate', notify);
    window.addEventListener('hashchange', notify);

    if (document.readyState === 'complete') {
      notify();
    } else {
      window.addEventListener('load', notify, { once: true });
    }

    const titleElement = document.querySelector('title') || document.head;
    const observer = new MutationObserver(() => notify());
    observer.observe(titleElement, { childList: true, subtree: true });
    return observer;
  };

  window.__vrooliBridgeChildInstalled = true;

  post({ v: 1, t: 'HELLO', appId: options.appId, title: document.title, caps });

  interceptHistory();
  const observer = setupObservers();

  queueMicrotask(() => post({ v: 1, t: 'READY' }));

  return {
    notify,
    dispose: () => {
      observer.disconnect();
      window.removeEventListener('message', handleMessage);
      window.removeEventListener('popstate', notify);
      window.removeEventListener('hashchange', notify);
      window.__vrooliBridgeChildInstalled = false;
    },
  };
}
]]></input>
</test-case>

<test-case id="bridge-js-valid" should-fail="false" path="testdata/bridge-js-valid/ui/iframeBridgeChild.js">
  <description>Canonical JavaScript bridge passes without violations</description>
  <input language="javascript"><![CDATA[
export function initIframeBridgeChild(options = {}) {
  if (typeof window === 'undefined' || typeof document === 'undefined') {
    return {
      notify: () => undefined,
      dispose: () => undefined,
    };
  }

  if (window.parent === window) {
    return {
      notify: () => undefined,
      dispose: () => undefined,
    };
  }

  if (window.__vrooliBridgeChildInstalled) {
    return {
      notify: () => undefined,
      dispose: () => undefined,
    };
  }

  const inferParentOrigin = () => {
    try {
      if (document.referrer) {
        const referrer = new URL(document.referrer);
        return referrer.origin;
      }
    } catch (error) {
      console.warn('[BridgeChild] Failed to parse document.referrer', error);
    }
    return null;
  };

  const buildLocationPayload = () => ({
    v: 1,
    t: 'LOCATION',
    href: window.location.href,
    path: `${window.location.pathname}${window.location.search}${window.location.hash}`,
    title: document.title,
    canGoBack: true,
    canGoFwd: true,
  });

  const caps = ['history', 'hash', 'title', 'deeplink'];
  let resolvedOrigin = options.parentOrigin ?? inferParentOrigin() ?? '*';

  const post = (payload) => {
    try {
      window.parent.postMessage(payload, resolvedOrigin);
    } catch (error) {
      console.warn('[BridgeChild] postMessage failed', error);
    }
  };

  const notify = () => {
    const payload = buildLocationPayload();
    post(payload);
    if (typeof options.onNav === 'function') {
      options.onNav(payload.href);
    }
  };

  const handleMessage = (event) => {
    if (resolvedOrigin !== '*' && event.origin !== resolvedOrigin) {
      return;
    }
    if (resolvedOrigin === '*' && event.origin) {
      resolvedOrigin = event.origin;
    }

    const message = event.data;
    if (!message || typeof message !== 'object' || message.v !== 1) {
      return;
    }

    if (message.t === 'NAV') {
      try {
        if (message.cmd === 'BACK') {
          history.back();
        } else if (message.cmd === 'FWD') {
          history.forward();
        } else if (message.cmd === 'GO' && typeof message.to === 'string') {
          const resolved = new URL(message.to, window.location.href);
          if (resolved.origin !== window.location.origin) {
            window.location.assign(resolved.href);
            return;
          }
          history.pushState({}, '', `${resolved.pathname}${resolved.search}${resolved.hash}`);
          window.dispatchEvent(new PopStateEvent('popstate', { state: history.state }));
        }
        notify();
      } catch (error) {
        post({ v: 1, t: 'ERROR', code: 'NAV_FAILED', detail: String((error && error.message) || error) });
      }
    } else if (message.t === 'PING' && typeof message.ts === 'number') {
      post({ v: 1, t: 'PONG', ts: message.ts });
    }
  };

  const interceptHistory = () => {
    const originalPush = history.pushState;
    const originalReplace = history.replaceState;

    history.pushState = function pushState(...args) {
      originalPush.apply(history, args);
      notify();
    };

    history.replaceState = function replaceState(...args) {
      originalReplace.apply(history, args);
      notify();
    };
  };

  const setupObservers = () => {
    window.addEventListener('message', handleMessage);
    window.addEventListener('popstate', notify);
    window.addEventListener('hashchange', notify);

    if (document.readyState === 'complete') {
      notify();
    } else {
      window.addEventListener('load', notify, { once: true });
    }

    const titleElement = document.querySelector('title') || document.head;
    const observer = new MutationObserver(() => notify());
    observer.observe(titleElement, { childList: true, subtree: true });
    return observer;
  };

  window.__vrooliBridgeChildInstalled = true;

  post({ v: 1, t: 'HELLO', appId: options.appId, title: document.title, caps });

  interceptHistory();
  const observer = setupObservers();

  queueMicrotask(() => post({ v: 1, t: 'READY' }));

  const controller = {
    notify,
    dispose: () => {
      observer.disconnect();
      window.removeEventListener('message', handleMessage);
      window.removeEventListener('popstate', notify);
      window.removeEventListener('hashchange', notify);
      window.__vrooliBridgeChildInstalled = false;
    },
  };

  return controller;
}

if (typeof window !== 'undefined') {
  window.initIframeBridgeChild = initIframeBridgeChild;
}
]]></input>
</test-case>

<test-case id="bridge-modified" should-fail="true" path="testdata/bridge-modified/ui/src/iframeBridgeChild.ts">
  <description>Any modification to bridge implementation fails validation</description>
  <input language="typescript">
export function initIframeBridgeChild(options: BridgeChildOptions = {}): BridgeChildController {
  const caps: BridgeCapability[] = ['history', 'hash', 'title'];
  return { notify: () => undefined, dispose: () => undefined } as BridgeChildController;
}
  </input>
  <expected-violations>1</expected-violations>
  <expected-message>must exactly match canonical iframe bridge</expected-message>
</test-case>

<test-case id="entry-missing-guard" should-fail="true" path="testdata/entry-missing-guard/ui/src/main.tsx">
  <description>Entry file that calls bridge without iframe guard is flagged</description>
  <input language="typescript"><![CDATA[
import { initIframeBridgeChild } from './iframeBridgeChild'

initIframeBridgeChild({ appId: 'demo' })
]]></input>
  <expected-violations>1</expected-violations>
  <expected-message>iframe guard</expected-message>
</test-case>

<test-case id="entry-missing-app-id" should-fail="true" path="testdata/entry-missing-app-id/ui/src/main.tsx">
  <description>Entry file must pass appId in bridge options</description>
  <input language="typescript"><![CDATA[
import { initIframeBridgeChild } from './iframeBridgeChild'

if (typeof window !== 'undefined' && window.parent !== window) {
  initIframeBridgeChild({})
}
]]></input>
  <expected-violations>1</expected-violations>
  <expected-message>appId</expected-message>
</test-case>

<test-case id="entry-valid" should-fail="false" path="testdata/entry-valid/ui/src/main.tsx">
  <description>Entry file with guard and bridge call passes</description>
  <input language="typescript"><![CDATA[
import { initIframeBridgeChild } from './iframeBridgeChild'

if (typeof window !== 'undefined' && window.parent !== window && !window.__demoBridgeInitialized) {
  let parentOrigin: string | undefined;
  try {
    if (document.referrer) {
      parentOrigin = new URL(document.referrer).origin;
    }
  } catch (error) {
    console.warn('[Demo] Unable to parse parent origin', error);
  }

  initIframeBridgeChild({ parentOrigin, appId: 'demo-scenario' });
  window.__demoBridgeInitialized = true;
}
]]></input>
</test-case>

<test-case id="static-bootstrap-valid" should-fail="false" path="testdata/static-bootstrap-valid/ui/app.js">
  <description>Static JS bootstrap that guards and calls window.initIframeBridgeChild passes</description>
  <input language="javascript">
(function () {
  if (typeof window === 'undefined') return;
  if (typeof window.initIframeBridgeChild !== 'function') return;
  if (window.parent === window) return;
  window.initIframeBridgeChild({ appId: 'static-demo' });
})();
  </input>
</test-case>

<test-case id="static-bootstrap-missing-call" should-fail="true" path="testdata/static-bootstrap-missing-call/ui/app.js">
  <description>Static bootstrap must invoke window.initIframeBridgeChild</description>
  <input language="javascript">
(function () {
  if (typeof window === 'undefined') return;
  if (typeof window.initIframeBridgeChild !== 'function') return;
  if (window.parent === window) return;
  console.log('Bridge ready but not invoked');
})();
  </input>
  <expected-violations>1</expected-violations>
  <expected-message>initIframeBridgeChild</expected-message>
</test-case>
*/

var (
	bridgeCallPattern       = regexp.MustCompile(`initIframeBridgeChild\s*\(`)
	windowBridgeCallPattern = regexp.MustCompile(`window\.(?:parent\.)?initIframeBridgeChild\s*\(`)
	bridgeGuardPattern      = regexp.MustCompile(`window\.parent\s*(?:!==|!=|===|==)\s*window`)
	appIDPattern            = regexp.MustCompile(`appId\s*:\s*['"]`)
	bridgeImportPattern     = regexp.MustCompile(`(?m)(import\s+\{?\s*initIframeBridgeChild|require\([^\n]*iframeBridgeChild)`)
)

var (
	canonicalOnce sync.Once

	canonicalTSNormalized string
	canonicalJSNormalized string
	canonicalLoadErr      error
)

// CheckIframeBridgeQuality validates canonical UI bridge implementation and bootstrap usage.
func CheckIframeBridgeQuality(content []byte, filePath string) []rules.Violation {
	path := filepath.ToSlash(strings.TrimSpace(filePath))
	if path == "" {
		return nil
	}

	base := strings.ToLower(filepath.Base(path))
	source := string(content)

	switch {
	case isBridgeFile(base):
		return validateBridge(content, path)
	case isEntryCandidate(base, source):
		return validateEntry(content, path)
	default:
		return nil
	}
}

func isBridgeFile(base string) bool {
	return strings.Contains(base, "iframebridgechild")
}

func isEntryCandidate(base, source string) bool {
	if isEntryFileName(base) {
		return true
	}

	lower := strings.ToLower(source)
	if strings.Contains(lower, "window.initiframebridgechild") || strings.Contains(lower, "initiframebridgechild(") {
		return true
	}
	if strings.Contains(lower, "typeof window.initiframebridgechild") {
		return true
	}
	return false
}

func isEntryFileName(base string) bool {
	if isBridgeFile(base) {
		return false
	}
	switch {
	case strings.HasPrefix(base, "main."),
		strings.HasPrefix(base, "app."),
		strings.HasPrefix(base, "entry."),
		strings.HasPrefix(base, "bootstrap."),
		strings.HasPrefix(base, "index."),
		strings.HasPrefix(base, "script."),
		strings.HasPrefix(base, "server."),
		strings.HasPrefix(base, "ui."):
		ext := strings.ToLower(filepath.Ext(base))
		switch ext {
		case ".ts", ".tsx", ".js", ".jsx":
			return true
		}
	}
	return false
}

func validateBridge(content []byte, path string) []rules.Violation {
	ensureCanonicalBridge()

	normalized := normalizeBridge(string(content))
	if normalized == canonicalTSNormalized {
		return nil
	}
	if canonicalJSNormalized != "" && normalized == canonicalJSNormalized {
		return nil
	}

	return []rules.Violation{
		newBridgeViolation(path, "Bridge implementation must exactly match canonical iframe bridge"),
	}
}

func validateEntry(content []byte, path string) []rules.Violation {
	source := string(content)
	lower := strings.ToLower(path)

	isStaticBootstrap := strings.Contains(lower, "ui/app.") || strings.Contains(lower, "ui/script.") || strings.Contains(lower, "/app.") || likelyStaticBootstrapSource(source)
	called := bridgeCallPattern.MatchString(source) || windowBridgeCallPattern.MatchString(source)

	var violations []rules.Violation

	if isStaticBootstrap {
		if !called {
			violations = append(violations, newBridgeViolation(path, "Static UI bootstrap must call window.initIframeBridgeChild"))
			return dedupeViolations(violations)
		}
	} else {
		if !bridgeImportPattern.MatchString(source) && !strings.Contains(source, "initIframeBridgeChild") {
			violations = append(violations, newBridgeViolation(path, "Missing iframe bridge import; UI entry file must import initIframeBridgeChild"))
		}
		if !called {
			violations = append(violations, newBridgeViolation(path, "Missing bridge invocation; UI entry file must invoke initIframeBridgeChild"))
		}
	}

	if !bridgeGuardPattern.MatchString(source) {
		violations = append(violations, newBridgeViolation(path, "Missing iframe guard; UI entry file must guard bridge initialization with an iframe check"))
	}

	if called && !appIDPattern.MatchString(source) {
		violations = append(violations, newBridgeViolation(path, "UI entry file must pass appId in bridge options"))
	}

	return dedupeViolations(violations)
}

func normalizeBridge(content string) string {
	content = strings.TrimSpace(content)
	if strings.HasPrefix(content, "<![CDATA[") && strings.HasSuffix(content, "]]>") {
		content = strings.TrimSpace(strings.TrimSuffix(strings.TrimPrefix(content, "<![CDATA["), "]]>"))
	}
	content = strings.ReplaceAll(content, "\r\n", "\n")
	content = strings.TrimSpace(content)
	return content
}

func newBridgeViolation(path, message string) rules.Violation {
	return rules.Violation{
		Severity:       "high",
		Message:        message,
		FilePath:       filepath.ToSlash(path),
		Recommendation: "Replace the UI bridge assets with the canonical implementation from scenario-auditor",
	}
}

func dedupeViolations(violations []rules.Violation) []rules.Violation {
	if len(violations) <= 1 {
		return violations
	}
	seen := map[string]bool{}
	result := make([]rules.Violation, 0, len(violations))
	for _, v := range violations {
		if seen[v.Message] {
			continue
		}
		seen[v.Message] = true
		result = append(result, v)
	}
	return result
}

func likelyStaticBootstrapSource(source string) bool {
	trimmed := strings.TrimSpace(source)
	lower := strings.ToLower(trimmed)

	if strings.Contains(lower, "typeof window.initiframebridgechild") && strings.Contains(lower, "window.parent === window") {
		return true
	}

	if strings.HasPrefix(lower, "(function") || strings.HasPrefix(lower, "!function") || strings.HasPrefix(lower, "(() =>") {
		if strings.Contains(lower, "window.initiframebridgechild") {
			return true
		}
	}

	return false
}

func ensureCanonicalBridge() {
	canonicalOnce.Do(func() {
		ruleDir := locateRuleDir()
		tsPath := filepath.Join(ruleDir, "..", "..", "..", "ui", "src", "iframeBridgeChild.ts")
		data, err := os.ReadFile(tsPath)
		if err != nil {
			canonicalLoadErr = fmt.Errorf("load canonical bridge ts: %w", err)
			return
		}
		canonicalTSNormalized = normalizeBridge(string(data))

		jsCandidates := []string{
			filepath.Join(ruleDir, "..", "..", "..", "ui", "iframeBridgeChild.js"),
			filepath.Join(ruleDir, "..", "..", "..", "ui", "canonical", "iframeBridgeChild.js"),
			filepath.Join(ruleDir, "testdata", "bridge-js-valid", "ui", "iframeBridgeChild.js"),
		}
		for _, candidate := range jsCandidates {
			if candidate == "" {
				continue
			}
			if data, err := os.ReadFile(candidate); err == nil {
				canonicalJSNormalized = normalizeBridge(string(data))
				break
			}
		}
	})

	if canonicalLoadErr != nil {
		panic(canonicalLoadErr)
	}
}

func locateRuleDir() string {
	if _, file, _, ok := runtime.Caller(0); ok {
		if strings.HasSuffix(file, "iframe_bridge_quality.go") {
			return filepath.Dir(file)
		}
	}

	if wd, err := os.Getwd(); err == nil {
		if dir := searchRuleDirFrom(wd); dir != "" {
			return dir
		}
	}

	for _, envVar := range []string{"VROOLI_ROOT", "APP_ROOT"} {
		if base := strings.TrimSpace(os.Getenv(envVar)); base != "" {
			if dir := searchRuleDirFrom(base); dir != "" {
				return dir
			}
		}
	}

	return "."
}

func searchRuleDirFrom(start string) string {
	start = strings.TrimSpace(start)
	if start == "" {
		return ""
	}

	current := filepath.Clean(start)
	visited := map[string]struct{}{}

	for {
		if _, seen := visited[current]; seen {
			break
		}
		visited[current] = struct{}{}

		candidates := []string{
			current,
			filepath.Join(current, "rules", "ui"),
			filepath.Join(current, "api", "rules", "ui"),
			filepath.Join(current, "scenarios", "scenario-auditor", "api", "rules", "ui"),
		}

		for _, candidate := range candidates {
			file := filepath.Join(candidate, "iframe_bridge_quality.go")
			if info, err := os.Stat(file); err == nil && !info.IsDir() {
				return filepath.Dir(file)
			}
		}

		parent := filepath.Dir(current)
		if parent == current {
			break
		}
		current = parent
	}

	return ""
}
