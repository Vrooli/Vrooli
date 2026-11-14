#!/usr/bin/env bash
set -euo pipefail

# Generates the unaided-agent baseline for the blog evaluation.
#
# WHAT THIS IS
#
# The blind comparison in run-blog.sh answers one question: does Prose Studio's
# machinery produce a better article than an ordinary agent that lacks it? That
# question only has an answer if the thing on the other side of the comparison
# is an agent's honest best attempt. It previously was not. `baseline` and
# `evidence_source` pointed at the same 182-word brief, so the judge was asked
# to prefer a 1000-word article over the summary it had been expanded from --
# a comparison with no information in it.
#
# WHAT IS HELD FIXED
#
# The baseline runs on the same model, through the same gateway, from the same
# brief, against the same subject and length target as the scenario path. The
# model is deliberately NOT a variable: if the two sides ran on different
# models, a win would be evidence about model capability rather than about this
# scenario, and the whole measurement would stop being about Prose Studio. The
# only difference between the two sides is the machinery.
#
# WHAT THE BASELINE DELIBERATELY GETS
#
# A competent prompt, including a plain-language statement of the intended
# voice. A baseline stripped of any voice guidance would be a strawman, and
# beating a strawman is not the claim being made. The claim is that versioned
# style resolution, outline-and-section composition, measured candidate sets,
# deterministic gating, and reroll under negative conditioning beat a capable
# one-shot prompt -- not that having any voice instruction at all beats having
# none.
#
# WHAT THE BASELINE DELIBERATELY DOES NOT GET
#
# One call. No outline stage, no section decomposition, no candidate set, no
# measurement, no eligibility gate, no selection policy, no reroll, no repair.
# That list is exactly the machinery under test.
#
# WHY IT IS CHECKED IN RATHER THAN REGENERATED PER RUN
#
# A comparison target that moves on every run makes progress untrackable: both
# sides would drift and no two runs would be comparable. The artifacts are
# committed so the target holds still, and this script exists so the target
# stays auditable and re-derivable rather than being files of unknown origin.
# run-blog.sh records their content hashes on every run, so a silently changed
# target is visible in the evidence rather than invisible in it.
#
# WHY THERE ARE SEVERAL OF THEM
#
# One draft is one sample of a stochastic process, not the behaviour of an
# unaided agent. The routed role writes at temperature 0.9, so two calls on the
# same brief produce different articles, and the scenario side varies the same
# way: its argument score was measured at 5 and then 3 on consecutive runs of
# an identical configuration. Comparing one scenario document against one
# baseline draft cannot separate a real difference from that spread, so the
# control is a fixed set and the comparison is between distributions.

if [[ -z "${AI_GATEWAY_URL:-}" ]]; then
  echo "AI_GATEWAY_URL must point at a started ai-gateway" >&2
  exit 2
fi

root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
spec="$root/evals/blog.primary.json"
out_dir="$root/evals/baselines"
out_prov="$out_dir/unaided-agent-draft.provenance.json"
count="$(jq -r '.samples.baseline // 3' "$spec")"
mkdir -p "$out_dir"
rm -f "$out_dir"/unaided-agent-draft-*.md

subject="$(jq -r '.subject' "$spec")"
evidence_path="$root/$(jq -r '.evidence_source' "$spec")"
evidence_text="$(cat "$evidence_path")"
evidence_claims="$(jq -r '.evidence_claims | join("; ")' "$spec")"
# The baseline receives the same artifacts as the scenario path. Withholding
# them would make the comparison measure who was handed the commands rather
# than what was done with them: the artifact list is part of the brief, and the
# brief is what both sides are supposed to share. Only the machinery differs.
artifact_lines="$(jq -r '[.evidence_artifacts[]? | "\(.value) -- \(.gloss)"] | join("; ")' "$spec")"
min_words="$(jq -r '.mechanical_gates.min_words' "$spec")"
max_words="$(jq -r '.mechanical_gates.max_words' "$spec")"
target_words=$(( (min_words + max_words) / 2 ))

# The grounding rules are copied from the scenario path verbatim. An unaided
# agent that was allowed to fabricate while the scenario path was not would win
# or lose on licence rather than on craft.
instruction="You are an experienced engineer writing a dev-log post for your company's engineering blog. Write in first person with a builder's voice grounded in shipped work. Explain why the change matters, not only what changed. Use a hook, an introduction, a body, and a conclusion, with short paragraphs. Introduce any internal system name by translating its purpose for the reader before you rely on it.

Write the best article you can. Aim for roughly ${target_words} words, and no fewer than ${min_words}.

Use only factual claims supported by the evidence brief below. Do not invent metrics, infrastructure, failure incidents, or outcomes. Every number, command, path, and identifier you write must appear either in the brief or in the artifact list below; anything else is invention. Show the artifacts rather than describing them: quote them verbatim inside the sentence that explains what they do.

Artifacts: ${artifact_lines}

Evidence claims to cover: ${evidence_claims}

Evidence brief:
${evidence_text}

Return the article as prose only, with no preamble, no title block, and no commentary about the task."

request="$(jq -nc \
  --arg source "$subject" \
  --arg instruction "$instruction" \
  '{source:$source,schemaJson:"{\"type\":\"string\"}",instruction:$instruction,role:"write.default",profile:"PROFILE_REMOTE_ONLY",maxOutputTokens:8000}')"

# The output cap is sized well above the prose because the routed model spends
# several thousand tokens before the article and the whole article is returned
# as one escaped JSON string. At a 2400 cap this request reported
# outputTokens 2386 and validated:true while returning 452 characters ending
# mid-word -- a truncated baseline that the transport called valid. The guards
# below exist because that failure is silent at the transport layer.
# An out-of-spec draw is resampled, never repaired. The article target is
# stated in the instruction, so a draft that lands under the floor is the
# unaided path missing a target it was given -- and telling it so and asking
# again would hand the control the gate-and-reroll machinery this comparison
# exists to isolate. Drawing again from the same prompt, with no feedback about
# the discarded attempt, keeps the control unaided while still sampling from
# drafts that meet the spec both sides are held to. The number of draws each
# slot needed is recorded, because the rejection rate is itself a fact about
# how reliably an unaided agent hits a stated length.
max_draws="$(jq -r '.samples.baseline_max_draws // 6' "$spec")"
drafts_meta='[]'
for ((i = 1; i <= count; i++)); do
  out_md="$out_dir/unaided-agent-draft-$i.md"
  article=""; word_count=0; draws=0; discarded='[]'
  while (( draws < max_draws )); do
    draws=$(( draws + 1 ))
    response="$(curl --fail-with-body --silent --show-error -X POST \
      "$AI_GATEWAY_URL/vrooli.ai_gateway.v1.inference.InferenceService/Run" \
      -H 'content-type: application/json' -d "$request")"

    candidate="$(jq -r '.valueJson // .value_json // ""' <<<"$response" | jq -r '.')"
    if [[ -z "$candidate" || "$candidate" == "null" ]]; then
      echo "gateway returned no article for draft $i draw $draws" >&2
      discarded="$(jq -c '. + ["empty_response"]' <<<"$discarded")"
      continue
    fi
    candidate_words="$(printf '%s' "$candidate" | tr -cs "[:alnum:]'" '\n' | grep -c . || true)"
    if (( candidate_words < min_words )); then
      discarded="$(jq -c --arg r "under_floor:$candidate_words" '. + [$r]' <<<"$discarded")"
      continue
    fi
    if [[ ! "$(printf '%s' "$candidate" | tail -c 1)" =~ [.!?\"] ]]; then
      discarded="$(jq -c '. + ["truncated"]' <<<"$discarded")"
      continue
    fi
    article="$candidate"; word_count="$candidate_words"
    break
  done

  if [[ -z "$article" ]]; then
    echo "draft $i: no draw met the ${min_words}-word floor in $max_draws attempts." >&2
    echo "Discarded: $discarded" >&2
    echo "Refusing to write a control the scenario would beat on length alone." >&2
    exit 1
  fi

  printf '%s\n' "$article" > "$out_md"
  drafts_meta="$(jq -c \
    --arg path "evals/baselines/$(basename "$out_md")" \
    --arg sha "$(sha256sum "$out_md" | cut -d' ' -f1)" \
    --argjson words "$word_count" \
    --argjson cost "$(jq -r '.usage.costMicros // 0 | tonumber' <<<"$response")" \
    --arg provider "$(jq -r '.provider // "unknown"' <<<"$response")" \
    --arg model "$(jq -r '.model // .resolvedModelRef // "unknown"' <<<"$response")" \
    --argjson draws "$draws" \
    --argjson discarded "$discarded" \
    '. + [{path:$path, sha256:$sha, word_count:$words, cost_micros:$cost, provider:$provider, model:$model, draws:$draws, discarded_draws:$discarded}]' \
    <<<"$drafts_meta")"
  printf 'draft %s: words=%s draws=%s\n' "$i" "$word_count" "$draws" >&2
done

jq -n \
  --arg generated_at "$(date -u +%Y-%m-%dT%H:%M:%SZ)" \
  --arg subject "$subject" \
  --arg role "write.default" \
  --arg evidence_source "$(jq -r '.evidence_source' "$spec")" \
  --arg evidence_sha256 "$(sha256sum "$evidence_path" | cut -d' ' -f1)" \
  --arg artifacts "$artifact_lines" \
  --argjson drafts "$drafts_meta" \
  --arg instruction "$instruction" \
  '{
    kind: "unaided-agent-baseline-set",
    purpose: "The control side of the blind comparison: one capable agent, one call per draft, same model and same brief as the scenario path, without any Prose Studio machinery. Several drafts because one sample of a stochastic writer is not its behaviour.",
    generated_at: $generated_at,
    subject: $subject,
    gateway_role: $role,
    draft_count: ($drafts | length),
    drafts: $drafts,
    evidence_source: $evidence_source,
    evidence_sha256: $evidence_sha256,
    artifacts: ($artifacts | split("; ")),
    machinery_withheld: [
      "outline stage",
      "section decomposition",
      "candidate set",
      "deterministic measurement",
      "eligibility gating",
      "selection policy",
      "reroll under negative conditioning",
      "coherence repair"
    ],
    instruction: $instruction
  }' > "$out_prov"

echo "provenance=$out_prov" >&2
jq -r '"baseline set: \(.draft_count) drafts, words \([.drafts[].word_count]|join(", ")), model \(.drafts[0].model)"' "$out_prov" >&2
