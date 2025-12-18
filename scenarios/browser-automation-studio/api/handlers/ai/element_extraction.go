package ai

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"

	autocompiler "github.com/vrooli/browser-automation-studio/automation/compiler"
	autocontracts "github.com/vrooli/browser-automation-studio/automation/contracts"
	basactions "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/actions"
)

// elementExtractionExpression is the client-side JavaScript that extracts
// interactive elements and page context from the DOM.
const elementExtractionExpression = `(function() {
// Find all interactive elements
const interactiveElements = Array.from(document.querySelectorAll('a, button, input, select, textarea, [role="button"], [onclick], [tabindex]'));

// Helper function to calculate element confidence based on visibility and interactivity
function calculateConfidence(element) {
  let confidence = 0.1; // Base confidence

  // Visible elements get higher confidence
  const rect = element.getBoundingClientRect();
  if (rect.width > 0 && rect.height > 0) confidence += 0.3;

  // Elements with text content get higher confidence
  if (element.textContent && element.textContent.trim()) confidence += 0.2;

  // Common interactive elements get higher confidence
  if (['button', 'input', 'select', 'textarea'].includes(element.tagName.toLowerCase())) confidence += 0.3;
  if (element.tagName.toLowerCase() === 'a' && element.href) confidence += 0.2;

  // Elements with specific roles get higher confidence
  if (element.getAttribute('role') === 'button') confidence += 0.2;

  return Math.min(confidence, 1.0);
}

// Helper function to generate robust selectors
function generateSelectors(element) {
  const selectors = [];

  // ID selector (highest priority)
  if (element.id) {
    selectors.push({
      selector: '#' + element.id,
      type: 'id',
      robustness: 0.9,
      fallback: false
    });
  }

  // Data attribute selectors
  for (const attr of element.attributes) {
    if (attr.name.startsWith('data-')) {
      selectors.push({
        selector: '[' + attr.name + '="' + attr.value + '"]',
        type: 'data-attr',
        robustness: 0.8,
        fallback: false
      });
    }
  }

  // Class selectors (for semantic classes)
  if (element.className) {
    const classes = element.className.split(/\s+/);
    const semanticClasses = classes.filter(cls =>
      /^(btn|button|link|nav|menu|form|input|submit|login|search)/.test(cls)
    );
    if (semanticClasses.length > 0) {
      selectors.push({
        selector: '.' + semanticClasses[0],
        type: 'class',
        robustness: 0.6,
        fallback: false
      });
    }
  }

  // CSS selector based on tag and attributes
  let cssSelector = element.tagName.toLowerCase();
  if (element.type) cssSelector += '[type="' + element.type + '"]';
  if (element.name) cssSelector += '[name="' + element.name + '"]';

  selectors.push({
    selector: cssSelector,
    type: 'css',
    robustness: 0.4,
    fallback: true
  });

  // XPath as fallback
  function getXPath(element) {
    if (element.id) return '//*[@id="' + element.id + '"]';

    let path = '';
    while (element && element.nodeType === Node.ELEMENT_NODE) {
      let index = 0;
      for (let sibling = element.previousSibling; sibling; sibling = sibling.previousSibling) {
        if (sibling.nodeType === Node.ELEMENT_NODE && sibling.tagName === element.tagName) index++;
      }
      const tagName = element.tagName.toLowerCase();
      path = '/' + tagName + '[' + (index + 1) + ']' + path;
      element = element.parentNode;
    }
    return path;
  }

  selectors.push({
    selector: getXPath(element),
    type: 'xpath',
    robustness: 0.3,
    fallback: true
  });

  return selectors.sort((a, b) => b.robustness - a.robustness);
}

// Helper function to categorize elements
function categorizeElement(element) {
  const text = element.textContent?.toLowerCase() || '';

  if (element.tagName === 'INPUT') {
    const type = element.type?.toLowerCase();
    if (type === 'email' || type === 'password' || type === 'tel') return 'auth';
    if (type === 'search') return 'search';
    if (type === 'submit' || type === 'button') return 'cta';
  }

  if (element.tagName === 'BUTTON') {
    if (text.includes('login') || text.includes('sign in')) return 'auth';
    if (text.includes('submit') || text.includes('send') || text.includes('save')) return 'cta';
  }

  if (element.tagName === 'A') {
    if (text.includes('login') || text.includes('sign in')) return 'auth';
    if (text.includes('sign up') || text.includes('register')) return 'signup';
  }

  if (element.tagName === 'SELECT') return 'form';
  if (element.tagName === 'TEXTAREA') return 'input';

  return 'other';
}

// Extract element information
const elements = interactiveElements.map(element => {
  const rect = element.getBoundingClientRect();

  const boundingBox = {
    x: Math.round(rect.x),
    y: Math.round(rect.y),
    width: Math.round(rect.width),
    height: Math.round(rect.height)
  };

  return {
    tagName: element.tagName,
    type: element.type || element.tagName.toLowerCase(),
    selectors: generateSelectors(element),
    boundingBox: {
      x: rect.x,
      y: rect.y,
      width: rect.width,
      height: rect.height
    },
    confidence: calculateConfidence(element),
    category: categorizeElement(element),
    attributes: {
      id: element.id || '',
      className: element.className || '',
      name: element.name || '',
      placeholder: element.placeholder || '',
      'aria-label': element.getAttribute('aria-label') || '',
      title: element.title || ''
    }
  };
}).filter(el => el.boundingBox.width > 0 && el.boundingBox.height > 0);

// Extract page context
const pageContext = {
  title: document.title,
  url: window.location.href,
  hasLogin: document.querySelector('input[type="password"]') !== null ||
           Array.from(document.querySelectorAll('button, a')).some(el =>
             el.textContent.toLowerCase().includes('login') || el.textContent.toLowerCase().includes('sign in')),
  hasSearch: document.querySelector('input[type="search"], input[name*="search"], input[placeholder*="search"]') !== null,
  formCount: document.querySelectorAll('form').length,
  buttonCount: document.querySelectorAll('button, input[type="button"], input[type="submit"]').length,
  linkCount: document.querySelectorAll('a[href]').length,
  viewport: {
    width: window.innerWidth,
    height: window.innerHeight
  }
};

return {
  elements: elements,
  pageContext: pageContext
};
})();`

// extractPageElements extracts all interactive elements from a page using the automation runner.
// It navigates to the URL, executes the element extraction JavaScript, and captures a screenshot.
func (h *ElementAnalysisHandler) extractPageElements(ctx context.Context, url string) ([]ElementInfo, PageContext, string, error) {
	if h.runner == nil {
		return nil, PageContext{}, "", errors.New("automation runner not configured")
	}

	instructions := []autocontracts.CompiledInstruction{
		{
			Index:  0,
			NodeID: "analysis.navigate",
			Action: mustBuildAction("navigate", map[string]any{
				"url":       url,
				"waitUntil": defaultPreviewWaitUntil,
				"timeoutMs": defaultPreviewTimeoutMilliseconds,
			}),
		},
		{
			Index:  1,
			NodeID: "analysis.wait",
			Action: mustBuildAction("wait", map[string]any{
				"waitType":   "time",
				"durationMs": defaultPreviewWaitMilliseconds,
			}),
		},
		{
			Index:  2,
			NodeID: "analysis.evaluate",
			Action: mustBuildAction("evaluate", map[string]any{
				"expression": elementExtractionExpression,
				"timeoutMs":  defaultPreviewTimeoutMilliseconds,
			}),
		},
		{
			Index:  3,
			NodeID: "analysis.screenshot",
			Action: mustBuildAction("screenshot", map[string]any{
				"fullPage":  true,
				"waitForMs": defaultPreviewWaitMilliseconds,
				"timeoutMs": defaultPreviewTimeoutMilliseconds,
			}),
		},
	}

	outcomes, _, err := h.runner.Run(ctx, previewDefaultViewportWidth, previewDefaultViewportHeight, instructions)
	if err != nil {
		return nil, PageContext{}, "", fmt.Errorf("automation run failed: %w", err)
	}

	var (
		resultElements []ElementInfo
		resultContext  PageContext
		screenshotData []byte
	)

	for _, outcome := range outcomes {
		switch outcome.NodeID {
		case "analysis.evaluate":
			if !outcome.Success {
				return nil, PageContext{}, "", fmt.Errorf("element extraction failed: %s", failureMessage(outcome.Failure))
			}
			value := outcome.ExtractedData["value"]
			payloadBytes, marshalErr := json.Marshal(value)
			if marshalErr != nil {
				return nil, PageContext{}, "", fmt.Errorf("failed to encode element extraction: %w", marshalErr)
			}
			var payload struct {
				Elements    []ElementInfo `json:"elements"`
				PageContext PageContext   `json:"pageContext"`
			}
			if err := json.Unmarshal(payloadBytes, &payload); err != nil {
				return nil, PageContext{}, "", fmt.Errorf("failed to parse element extraction payload: %w", err)
			}
			resultElements = payload.Elements
			resultContext = payload.PageContext
		case "analysis.screenshot":
			if !outcome.Success {
				return nil, PageContext{}, "", fmt.Errorf("screenshot capture failed: %s", failureMessage(outcome.Failure))
			}
			if outcome.Screenshot == nil || len(outcome.Screenshot.Data) == 0 {
				return nil, PageContext{}, "", errors.New("screenshot capture returned no data")
			}
			screenshotData = outcome.Screenshot.Data
		}
	}

	if len(resultElements) == 0 {
		return nil, PageContext{}, "", errors.New("no elements returned from extraction")
	}
	if len(screenshotData) == 0 {
		return nil, PageContext{}, "", errors.New("no screenshot captured")
	}

	base64Screenshot := fmt.Sprintf("data:%s;base64,%s", "image/png", base64.StdEncoding.EncodeToString(screenshotData))
	return resultElements, resultContext, base64Screenshot, nil
}

// mustBuildAction creates a typed ActionDefinition from step type and params.
// Panics if the action type is invalid (should never happen with hardcoded types).
func mustBuildAction(stepType string, params map[string]any) *basactions.ActionDefinition {
	action, err := autocompiler.BuildActionDefinition(stepType, params)
	if err != nil {
		panic(fmt.Sprintf("mustBuildAction: invalid action type %q: %v", stepType, err))
	}
	return action
}
