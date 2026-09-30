# Classifier Accuracy

Agent Manager evaluates the committed, redacted classification corpus through
`runsignal.ClassificationAccuracy`. This is the only scorer: the Go unit gate
and the self-targeted `agent-conformance` Test Genie response both call it.

The response publishes a row for every shipped detector with its identifier,
precision, recall, and committed threshold. The corpus and thresholds live at
`api/internal/runsignal/testdata/classification/all-detectors.labels.json`.

The committed corpus contains 50 labelled friction windows, including
near-miss negatives. Every case carries an auditable labeller reason; the
source metadata identifies redacted imported evidence spanning the Codex and
Claude-Code harnesses and multiple months. This keeps the gate about structural
evidence rather than detector-specific phrases or one operator's transcript.

The classifier-accuracy capability is unmeasured when the scorer cannot load a
complete labelled corpus. It is below threshold when either precision or recall
falls below a detector's threshold. Both are required findings; a clean result
means every shipped detector is covered and meets its threshold.

To inspect the live publication, call the shared scenario-validation endpoint
for `agent-manager`; its `nativeDetail.classifier_accuracy` field contains the
per-detector rows. The response is intentionally attached to the existing
Agent Conformance provider because Test Genie admits one provider descriptor per
scenario and this preserves the existing consumer-conformance phase.
