# LM1 mixed-admission Test Genie workload

lm1-mixed-admission is a declared Performance Health workload for Research Desk. The producer starts ten Audio Tools transcriptions, nine Search Hub queries and one local-only AI Gateway generate call behind a shared release barrier, then submits exactly one Image Tools generation with --no-reclaim --require-gpu --fallback-policy local_only. It waits for every owner result and the Image Tools terminal job receipt. The receipt carries separate Audio Tools, Search Hub, AI Gateway, Image Tools, platform-capacity and host-observation artifacts. The Performance Health owner retains their bytes and verifies producer, contract, candidate, operation and artifact digests; Test Genie's performance phase evaluates that retained evidence.

Running performance-health sweep workload-run research-desk lm1-mixed-admission --json is a live workload: it performs real inference and an image generation. Run it only after fresh owner and capacity admission covers the Image Tools GPU and disk job plus all 20 caller requests. The producer checks existing queue/claims, local-only policy, free GPU/RAM, strict Image Tools explain, footprint, and complete host sampling. These checks are runtime gates and do not grant owner admission.

After the workload returns a terminal receipt, evaluate it through the ordinary Test Genie phase with:

    vrooli scenario test research-desk --phases performance

The Test Genie phase reads retained producer evidence; it does not launch caller or image work. Use performance-health sweep workload-get research-desk lm1-mixed-admission --json to inspect the current owner receipt. A missing, stale, changed or failed receipt is unknown/failed evidence, never a pass. This slice only builds and tests the producer offline; no workload invocation or live phase run was performed.
