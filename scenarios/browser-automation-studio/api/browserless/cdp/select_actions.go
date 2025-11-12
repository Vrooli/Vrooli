package cdp

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/chromedp/cdproto/cdp"
	"github.com/chromedp/chromedp"
)

// ExecuteSelect updates the value of a native select element (single or multi-select).
func (s *Session) ExecuteSelect(ctx context.Context, selector, mode, value, text string, index int, values []string, multi bool, timeoutMs, waitAfterMs int) (*StepResult, error) {
	start := time.Now()
	result := &StepResult{DebugContext: map[string]interface{}{
		"selector": selector,
		"mode":     mode,
		"multi":    multi,
		"value":    value,
		"text":     text,
		"index":    index,
		"values":   values,
	}}

	if timeoutMs <= 0 {
		timeoutMs = 30000
	}

	timeoutCtx, cancel := context.WithTimeout(ctx, time.Duration(timeoutMs)*time.Millisecond)
	defer cancel()

	var nodes []*cdp.Node
	if err := chromedp.Run(timeoutCtx,
		chromedp.WaitVisible(selector, s.frameQueryOptions(chromedp.ByQuery)...),
		chromedp.ScrollIntoView(selector, s.frameQueryOptions(chromedp.ByQuery)...),
		chromedp.Nodes(selector, &nodes, s.frameQueryOptions(chromedp.ByQuery)...),
	); err != nil {
		result.Error = fmt.Sprintf("select target %s unavailable: %v", selector, err)
		result.DurationMs = int(time.Since(start).Milliseconds())
		return result, err
	}
	if len(nodes) == 0 {
		err := fmt.Errorf("select target %s not found", selector)
		result.Error = err.Error()
		result.DurationMs = int(time.Since(start).Milliseconds())
		return result, err
	}

	payload := selectEvalPayload{
		Selector: selector,
		Mode:     mode,
		Value:    value,
		Text:     text,
		Index:    index,
		Values:   values,
		Multi:    multi,
	}
	scriptBytes, err := json.Marshal(payload)
	if err != nil {
		result.Error = fmt.Sprintf("failed to marshal select payload: %v", err)
		result.DurationMs = int(time.Since(start).Milliseconds())
		return result, err
	}

	script := fmt.Sprintf(`(() => {
        const config = %s;
        const normalize = (value) => (value ?? '').toString().trim().toLowerCase();
        const element = document.querySelector(config.selector);
        if (!element) {
            throw new Error('Select node: selector ' + config.selector + ' not found');
        }
        const tag = (element.tagName || '').toLowerCase();
        const optionList = tag === 'select' && element.options ? Array.from(element.options) : Array.from(element.querySelectorAll('option'));
        if (!optionList.length) {
            throw new Error('Select node: no <option> elements found for ' + config.selector);
        }
        if (config.multi) {
            if (tag === 'select') {
                if (element.multiple !== true) {
                    throw new Error('Select node: element is not configured for multiple selections');
                }
            } else {
                throw new Error('Select node: multi-select requires a <select multiple> element');
            }
        }
        const emit = () => ({
            selectedValues: optionList.filter((opt) => opt.selected).map((opt) => opt.value ?? ''),
            selectedTexts: optionList.filter((opt) => opt.selected).map((opt) => opt.text ?? ''),
            multiple: Boolean(element.multiple),
        });
        if (config.multi) {
            const targets = Array.isArray(config.values) ? config.values.map(normalize).filter(Boolean) : [];
            if (!targets.length) {
                throw new Error('Select node: multi-select requires at least one value');
            }
            optionList.forEach((option) => {
                const compare = config.mode === 'text' ? normalize(option.text) : normalize(option.value);
                const match = config.mode === 'text'
                    ? targets.some((target) => compare.includes(target))
                    : targets.includes(compare);
                option.selected = match;
            });
            const firstSelected = optionList.findIndex((opt) => opt.selected);
            if (tag === 'select' && firstSelected >= 0) {
                element.selectedIndex = firstSelected;
            }
            element.dispatchEvent(new Event('input', { bubbles: true }));
            element.dispatchEvent(new Event('change', { bubbles: true }));
            return emit();
        }
        let targetIndex = -1;
        if (config.mode === 'index') {
            targetIndex = config.index;
            if (typeof targetIndex !== 'number' || targetIndex < 0 || targetIndex >= optionList.length) {
                throw new Error('Select node: index ' + config.index + ' is out of range');
            }
        } else {
            const needle = normalize(config.mode === 'text' ? config.text : config.value);
            if (!needle) {
                throw new Error('Select node: selection value missing for mode ' + config.mode);
            }
            targetIndex = optionList.findIndex((option) => {
                const compare = config.mode === 'text' ? normalize(option.text) : normalize(option.value);
                return config.mode === 'text' ? compare.includes(needle) : compare === needle;
            });
            if (targetIndex === -1) {
                throw new Error('Select node: no option matched ' + needle);
            }
        }
        optionList.forEach((option, idx) => {
            option.selected = idx === targetIndex;
        });
        if (tag === 'select') {
            element.selectedIndex = targetIndex;
        }
        element.dispatchEvent(new Event('input', { bubbles: true }));
        element.dispatchEvent(new Event('change', { bubbles: true }));
        return emit();
    })()`, string(scriptBytes))

	var evalResult selectEvalResult
	if err := s.evalWithFrame(timeoutCtx, script, &evalResult); err != nil {
		result.Error = fmt.Sprintf("select script failed: %v", err)
		result.DurationMs = int(time.Since(start).Milliseconds())
		return result, err
	}

	if waitAfterMs > 0 {
		if err := waitWithContext(timeoutCtx, time.Duration(waitAfterMs)*time.Millisecond); err != nil {
			result.Error = fmt.Sprintf("select wait interrupted: %v", err)
			result.DurationMs = int(time.Since(start).Milliseconds())
			return result, err
		}
	}

	chromedp.Run(s.ctx,
		chromedp.Location(&result.URL),
		chromedp.Title(&result.Title),
	)

	result.Success = true
	result.DurationMs = int(time.Since(start).Milliseconds())
	result.ConsoleLogs, result.NetworkEvents = s.GetTelemetry()
	result.DebugContext["selectedValues"] = evalResult.SelectedValues
	result.DebugContext["selectedTexts"] = evalResult.SelectedTexts
	result.DebugContext["elementMultiple"] = evalResult.Multiple
	return result, nil
}
