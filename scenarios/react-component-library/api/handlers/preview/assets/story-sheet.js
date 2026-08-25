(function () {
  const root = document.querySelector("[data-preview-sheet]");
  const summary = document.querySelector("[data-story-sheet-summary]");
  const result = document.querySelector("#rcl-story-result");
  const frames = [...document.querySelectorAll("iframe")];

  const check = () => {
    let ready = 0;
    let failed = 0;
    for (const frame of frames) {
      try {
        const document = frame.contentDocument;
        const harness = document?.querySelector(
          '[data-experience-surface="component-harness"]',
        );
        const state = harness?.getAttribute("data-experience-state");
        const status = harness?.getAttribute("data-rcl-story-status");
        if (state === "error" || status === "failed") failed += 1;
        if (state === "ready" && status === "passed") ready += 1;
      } catch (_) {
        // Cross-document readiness is retried on the next frame event.
      }
    }

    if (failed) {
      root.dataset.experienceState = "error";
      root.dataset.rclStoryStatus = "failed";
      summary.textContent = "Validation failed · review the affected specimen for details.";
    } else if (ready === frames.length && frames.length > 0) {
      root.dataset.experienceState = "ready";
      root.dataset.rclStoryStatus = "passed";
      summary.textContent = `Ready · ${frames.length} labeled story specimens`;
      result.textContent = JSON.stringify({ passed: true, failures: [], stories: frames.length });
    } else {
      root.dataset.experienceState = "loading";
      root.dataset.rclStoryStatus = "pending";
      summary.textContent = `Validating ${ready} of ${frames.length} story specimens`;
    }
  };

  frames.forEach((frame) => frame.addEventListener("load", () => setTimeout(check, 50)));
  check();
  setInterval(check, 100);
})();
