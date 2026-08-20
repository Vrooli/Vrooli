#!/usr/bin/env bash
set -euo pipefail

# Blog evaluation for prose-studio.
#
# The gates here are deliberately independent of the service's own measurement
# code. An eval that grades output with the same functions that produced it can
# only confirm the service agrees with itself; the previous version of this
# script created a claim out of every generated sentence and then asked whether
# those sentences were covered, which no output could fail. Everything measured
# below is either computed here from the assembled text, or read from a source
# that did not participate in generating it.

if [[ -z "${API_BASE_URL:-}" ]]; then
  echo "API_BASE_URL must point at a started prose-studio API" >&2
  exit 2
fi
command -v python3 >/dev/null || { echo "python3 is required for independent text measurement" >&2; exit 2; }

root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
spec="$root/evals/blog.primary.json"
run_dir="${PROSE_STUDIO_EVAL_DIR:-$root/.vrooli/evals/runs}"
work="$(mktemp -d)"
trap 'rm -rf "$work"' EXIT
mkdir -p "$run_dir"
stamp="$(date -u +%Y%m%dT%H%M%SZ)"
output="$run_dir/blog-$stamp.json"
post() { curl --fail-with-body --silent --show-error -X POST "$API_BASE_URL$1" -H 'content-type: application/json' -d "$2"; }

# Consumer-owned declarations are reindexed explicitly for the live evaluation
# so the evidence records the profile version currently on disk.
declarations_root="${PROSE_STUDIO_DECLARATIONS_ROOT:-$root/../content-desk}"
post /api/v1/prose/declarations/reindex "$(jq -nc --arg root "$declarations_root" '{root:$root}')" >/dev/null

subject="$(jq -r '.subject' "$spec")"
profile="$(jq -r '.profile' "$spec")"
claims_prompt="$(jq -r '.mechanical_gates.required_claim_terms | join(", ")' "$spec")"
evidence_path="$root/$(jq -r '.evidence_source' "$spec")"
evidence_text="$(cat "$evidence_path")"
evidence_claims="$(jq -r '.evidence_claims | join("; ")' "$spec")"

# Artifacts are the specifics the article is permitted to show. The previous
# instruction forbade inventing implementation details while supplying none,
# which left abstraction as the only compliant register and put the accuracy
# rule in direct conflict with the concreteness the judge was scoring.
artifact_values="$(jq -r '[.evidence_artifacts[]?.value] | join(" | ")' "$spec")"
artifact_lines="$(jq -r '[.evidence_artifacts[]? | "\(.value) -- \(.gloss)"] | join("; ")' "$spec")"

generation_instruction="Use only factual claims supported by this evidence brief. Do not invent metrics, infrastructure, failure incidents, or outcomes. Every number, command, path, and identifier you write must appear either in the brief or in the artifact list below; anything else is invention. Show the artifacts rather than describing them: quote them verbatim inside the sentence that explains what they do. Artifacts: $artifact_lines. Explain each evidence claim once, with distinct section purposes; do not restate the same workflow or conclusion in multiple sections. Evidence claims: $evidence_claims. Evidence source: $evidence_text. Explicitly use these exact terms where they fit naturally: $claims_prompt."

# The single-call agent path (PS-P0-016) is exercised as its own probe. Its cost
# and provider belong to that request and are recorded separately: reporting a
# standalone generate call's accounting as the document's was reporting a
# different request entirely.
probe="$(post /api/v1/prose/generate "$(jq -nc --arg p "$profile" --arg q "$subject" --arg instruction "$generation_instruction" '{profile_key:$p,query:($q+" "+$instruction),include_candidates:true}')")"

# The scenario side is sampled, not drawn once. A single document is one draw
# from a stochastic writer: the same configuration produced argument scores of
# 5 and then 3 on consecutive runs, which is larger than most of the effects
# this eval is asked to detect.
scenario_samples="$(jq -r '.samples.scenario // 3' "$spec")"
documents='[]'
for ((sample = 1; sample <= scenario_samples; sample++)); do
  doc="$(post /api/v1/prose/documents "$(jq -nc --arg p "$profile" --arg q "$subject" --arg instruction "$generation_instruction" --argjson artifacts "$(jq -c '[.evidence_artifacts[]?.value]' "$spec")" '{document:{title:($q+" "+$instruction),profile_key:$p,artifacts:$artifacts},sections:[]}')")"
  doc="$(jq -c '.document // .' <<<"$doc")"
  jq -r '.assembled_text // ""' <<<"$doc" > "$work/scenario-$sample.txt"
  documents="$(jq -c --argjson d "$doc" '. + [$d]' <<<"$documents")"
  printf 'scenario sample %s: %s words\n' "$sample" "$(wc -w < "$work/scenario-$sample.txt")" >&2
done
document="$(jq -c '.[0]' <<<"$documents")"
cp "$work/scenario-1.txt" "$work/assembled.txt"
cp "$evidence_path" "$work/evidence.txt"

# The control set is read from disk, never regenerated here, so the comparison
# target is identical across runs even though both sides are now distributions.
mapfile -t baseline_paths < <(cd "$root" && ls $(jq -r '.baseline_glob' "$spec") 2>/dev/null | sed "s|^|$root/|")
if (( ${#baseline_paths[@]} == 0 )); then
  echo "no baseline drafts matched $(jq -r '.baseline_glob' "$spec")" >&2
  echo "Generate them with AI_GATEWAY_URL=... evals/make-baseline.sh" >&2
  exit 2
fi

# ---------------------------------------------------------------------------
# Independent measurement. Nothing below imports the service's textmetrics.
# ---------------------------------------------------------------------------
measure_one() { python3 - "$1" "$work/evidence.txt" "$spec" <<'PY'
import json, re, sys, itertools

text = open(sys.argv[1], encoding="utf-8").read()
evidence = open(sys.argv[2], encoding="utf-8").read()
spec = json.load(open(sys.argv[3], encoding="utf-8"))

def words(s):
    return re.findall(r"[A-Za-z0-9']+", s.lower())

paragraphs = [p.strip() for p in re.split(r"\n\s*\n", text) if p.strip()]
sentences = [s for s in re.split(r"(?<=[.!?])\s+", text.strip()) if s.strip()]

# Jaccard over word sets: an independent duplication signal that does not share
# an implementation with the service's cross-section repetition metric.
def jaccard(a, b):
    sa, sb = set(words(a)), set(words(b))
    if not sa or not sb:
        return 0.0
    return len(sa & sb) / len(sa | sb)

max_dup = 0.0
dup_pair = None
for i, j in itertools.combinations(range(len(paragraphs)), 2):
    score = jaccard(paragraphs[i], paragraphs[j])
    if score > max_dup:
        max_dup, dup_pair = score, [i, j]

# Every number in the article must be traceable to the brief. Fabricated
# metrics are the highest-cost failure and the cheapest deterministic check.
def numbers(s):
    return set(re.findall(r"\d+(?:\.\d+)?", s))

# Artifacts widen what counts as grounded. A version number or flag that only
# exists in a declared command was previously indistinguishable from a
# fabricated metric, so the grounding gate and the concreteness goal pulled in
# opposite directions and abstraction was the only way to satisfy both.
artifacts = [a["value"] for a in spec.get("evidence_artifacts", [])]
grounded_numbers = numbers(evidence) | numbers(" ".join(artifacts))
unsupported_numbers = sorted(numbers(text) - grounded_numbers)

# Distinct artifacts that reached the article. Counted distinctly on purpose:
# repeating one command does not make prose more concrete, and counting
# occurrences would report that it does.
lowered_text = text.lower()
artifacts_used = [a for a in artifacts if a.lower() in lowered_text]
artifacts_missing = [a for a in artifacts if a.lower() not in lowered_text]

# Coverage runs from the declared evidence claims to the article, not from the
# article back to itself. A claim counts as present when most of its
# distinctive terms appear in one sentence.
stop = set(words("a an the and or of to in on for is are was were that this it its as with after before once every not"))
claim_hits = []
for claim in spec.get("evidence_claims", []):
    terms = [w for w in set(words(claim)) if w not in stop and len(w) > 2]
    best, best_score, best_span = None, 0.0, None
    spans, cursor = [], 0
    for sentence in sentences:
        start = text.find(sentence, cursor)
        if start >= 0:
            cursor = start + len(sentence)
        spans.append((sentence, start, start + len(sentence)))
    # A window of up to three consecutive sentences, because a claim developed
    # across two sentences is present in the article, not absent from it.
    for size in (1, 2, 3):
        for i in range(len(spans) - size + 1):
            window = spans[i:i + size]
            joined = " ".join(w[0] for w in window)
            present = sum(1 for t in terms if t in words(joined))
            score = present / len(terms) if terms else 0.0
            if score > best_score:
                best, best_score = joined, score
                best_span = [window[0][1], window[-1][2]]
    claim_hits.append({
        "claim": claim,
        "best_score": round(best_score, 4),
        "covered": best_score >= 0.6,
        "sentence": best,
        "span": best_span,
    })

# Adjacent novelty: the share of a paragraph's content words that did not
# appear in the paragraph before it. The service now gates the same phenomenon
# at section granularity with its own implementation; this measures it at
# paragraph granularity with a different stop list, so agreement between them
# is evidence and not an artefact of shared code. Low novelty means the text
# moved on to nothing, which is invisible to the duplication measure above --
# a paragraph that re-argues its predecessor in fresh words scores near zero
# on Jaccard overlap and near zero here, for opposite reasons.
novelty_stop = stop | set(words("we our you your they their be been being by from at can will would could has have had than then so such those these there here what which who when where how all any both each more most other some only own same too very just"))
def content_terms(s):
    return {w for w in words(s) if len(w) > 2 and w not in novelty_stop}

adjacent_novelty = []
for i in range(1, len(paragraphs)):
    current = content_terms(paragraphs[i])
    prior = content_terms(paragraphs[i - 1])
    adjacent_novelty.append(round(len(current - prior) / len(current), 4) if current else 0.0)

required = spec["mechanical_gates"].get("required_claim_terms", [])
lowered = text.lower()

json.dump({
    "word_count": len(words(text)),
    "sentence_count": len(sentences),
    "paragraph_count": len(paragraphs),
    "max_paragraph_duplication": round(max_dup, 4),
    "most_duplicated_pair": dup_pair,
    "artifacts_declared": len(artifacts),
    "artifacts_used": artifacts_used,
    "artifacts_missing": artifacts_missing,
    "distinct_artifacts_used": len(artifacts_used),
    "adjacent_paragraph_novelty": adjacent_novelty,
    "min_adjacent_novelty": round(min(adjacent_novelty), 4) if adjacent_novelty else 1.0,
    "median_adjacent_novelty": round(sorted(adjacent_novelty)[len(adjacent_novelty) // 2], 4) if adjacent_novelty else 1.0,
    "unsupported_numbers": unsupported_numbers,
    "required_terms_present": {term: (term.lower() in lowered) for term in required},
    "evidence_claims": claim_hits,
    "evidence_claim_coverage": round(sum(1 for c in claim_hits if c["covered"]) / len(claim_hits), 4) if claim_hits else 0.0,
    "basis": "independent measurement in evals/run-blog.sh; shares no implementation with packages/textmetrics",
}, sys.stdout, indent=2)
PY
}

# Every sampled document is measured, and the gates below read the worst value
# across the set rather than the first document's. Sampling the scenario side
# while gating only one of its samples would have quietly loosened every
# deterministic gate in this file.
sample_measurements='[]'
for ((sample = 1; sample <= scenario_samples; sample++)); do
  sample_measurements="$(jq -c --argjson m "$(measure_one "$work/scenario-$sample.txt")" '. + [$m]' <<<"$sample_measurements")"
done
measurements="$(jq -c '{
  samples: .,
  word_count: ([.[].word_count] | min),
  sentence_count: ([.[].sentence_count] | min),
  max_paragraph_duplication: ([.[].max_paragraph_duplication] | max),
  min_adjacent_novelty: ([.[].min_adjacent_novelty] | min),
  distinct_artifacts_used: ([.[].distinct_artifacts_used] | min),
  evidence_claim_coverage: ([.[].evidence_claim_coverage] | min),
  unsupported_numbers: ([.[].unsupported_numbers[]] | unique),
  required_terms_present: (map(.required_terms_present) | add // {}),
  worst_word_count_max: ([.[].word_count] | max),
  basis: "worst value across sampled scenario documents; per-document values retained under samples"
}' <<<"$sample_measurements")"

# ---------------------------------------------------------------------------
# Content Desk remains the claim authority for the declared evidence claims.
# ---------------------------------------------------------------------------
content_desk_base="${CONTENT_DESK_API_BASE_URL:-http://localhost:19812}"
draft_id=""
claim_coverage="{}"
if [[ "$(jq -r '.mechanical_gates.evidence_claim_coverage_required // false' "$spec")" == "true" ]]; then
  assembled="$(cat "$work/assembled.txt")"
  campaign="$(curl --fail-with-body --silent --show-error -X POST "$content_desk_base/vrooli.content_desk.v1.campaigns.CampaignsService/CreateCampaign" -H 'content-type: application/json' -d "$(jq -nc --arg reference "$evidence_path" '{name:("Prose Studio eval "+(now|tostring)),evidence_refs:[$reference],slots:[{channel:"eval",format:"essay",capacity:1}]}')" || true)"
  campaign_id="$(jq -r '.campaign.id // .campaignId // ""' <<<"$campaign" 2>/dev/null || echo "")"
  if [[ -n "$campaign_id" ]]; then
    curl --fail-with-body --silent --show-error -X POST "$content_desk_base/vrooli.content_desk.v1.campaigns.CampaignsService/ActivateCampaign" -H 'content-type: application/json' -d "$(jq -nc --arg id "$campaign_id" '{id:$id}')" >/dev/null
    draft="$(curl --fail-with-body --silent --show-error -X POST "$content_desk_base/vrooli.content_desk.v1.artifacts.ArtifactsService/CreateDraft" -H 'content-type: application/json' -d "$(jq -nc --arg campaign "$campaign_id" --arg body "$assembled" '{campaign_id:$campaign,body:$body,post_type_id:"dev-log",format:"essay",channel:"eval"}')")"
    draft_id="$(jq -r '.draft.id // .draftId // ""' <<<"$draft")"
    if [[ -n "$draft_id" ]]; then
      # One claim per DECLARED evidence claim, cited at the span the article
      # actually used for it. The statement is the evidence claim, never the
      # generated sentence, so the article cannot certify itself.
      while read -r claim_json; do
        statement="$(jq -r '.claim' <<<"$claim_json")"
        covered="$(jq -r '.covered' <<<"$claim_json")"
        [[ "$covered" != "true" ]] && continue
        start="$(jq -r '.span[0]' <<<"$claim_json")"
        end="$(jq -r '.span[1]' <<<"$claim_json")"
        claim="$(curl --fail-with-body --silent --show-error -X POST "$content_desk_base/vrooli.content_desk.v1.claims.ClaimsService/CreateClaim" -H 'content-type: application/json' -d "$(jq -nc --arg statement "$statement" --arg reference "$evidence_path" '{statement:$statement,kind:"capability",evidence_kind:"citation",reference:$reference}')")"
        claim_id="$(jq -r '.claim.id // .claimId // ""' <<<"$claim")"
        [[ -z "$claim_id" ]] && continue
        curl --fail-with-body --silent --show-error -X POST "$content_desk_base/vrooli.content_desk.v1.claims.ClaimsService/CiteClaim" -H 'content-type: application/json' -d "$(jq -nc --arg draft "$draft_id" --arg claim "$claim_id" --arg body "$assembled" --argjson start "$start" --argjson end "$end" '{draft_id:$draft,claim_id:$claim,span_start:$start,span_end:$end,body:$body}')" >/dev/null
      done < <(jq -c '.evidence_claims[]' "$work/measurements.json")
      claim_coverage="$(curl --fail-with-body --silent --show-error -X POST "$content_desk_base/vrooli.content_desk.v1.claims.ClaimsService/GetClaimCoverage" -H 'content-type: application/json' -d "$(jq -nc --arg draft "$draft_id" --arg body "$assembled" '{draft_id:$draft,body:$body}')")"
    fi
  fi
fi

# ---------------------------------------------------------------------------
# Blind comparison. This is the only gate that measures whether the article is
# good, and it was previously recorded as pending and never run. It is an
# inferential PASS/FAIL gate on an assembled document; it never reaches
# selection, which stays mechanical.
# ---------------------------------------------------------------------------
blind_status="skipped"
blind_scores='[]'
blind_scoring='{}'
blind_scenario_wins=0
blind_baseline_wins=0
blind_calibrated=0
judge_provider=""
judge_model=""
baseline_shas="$(for b in "${baseline_paths[@]}"; do sha256sum "$b" | cut -d' ' -f1; done | jq -R . | jq -sc .)"
if [[ "$(jq -r '.mechanical_gates.blind_comparison_required // false' "$spec")" == "true" ]]; then
  gateway="${AI_GATEWAY_URL:-}"
  for b in "${baseline_paths[@]}"; do
    # The control must be a separate artifact from the brief. When these were
    # the same file the judge was asked to prefer a 1000-word article over the
    # 182-word summary it had been expanded from.
    if [[ "$(readlink -f "$b")" == "$(readlink -f "$evidence_path")" ]]; then
      echo "a baseline draft resolves to the evidence source: $b" >&2
      exit 2
    fi
    [[ -s "$b" ]] || { echo "baseline draft is empty: $b" >&2; exit 2; }
  done
  if [[ -z "$gateway" ]]; then
    blind_status="unavailable_no_gateway"
  else
    # Calibration anchor, derived from the first scenario sample: same topic,
    # vocabulary, register and length, with nothing to say after two paragraphs.
    python3 - "$work/scenario-1.txt" > "$work/degraded.txt" <<'PYDEG'
import re, sys
text = open(sys.argv[1], encoding="utf-8").read()
paragraphs = [p.strip() for p in re.split(r"\n\s*\n", text) if p.strip()]
target = len(re.findall(r"[A-Za-z0-9']+", text))
seed = paragraphs[:2] or paragraphs
out, total, i = [], 0, 0
while total < target and seed:
    p = seed[i % len(seed)]
    out.append(p)
    total += len(re.findall(r"[A-Za-z0-9']+", p))
    i += 1
sys.stdout.write("\n\n".join(out))
PYDEG

    score_schema='{"type":"object","properties":{"score":{"type":"string","enum":["1","2","3","4","5"]},"quote":{"type":"string"},"rationale":{"type":"string"}},"required":["score","quote","rationale"]}'
    trial_failures=0

    score_one() { # rubric_key instruction subject_key text_file
      local rubric_key="$1" instruction="$2" subject_key="$3" file="$4"
      local request response value score quote rationale
      request="$(jq -nc --arg a "$(cat "$file")" --arg schema "$score_schema" --arg instruction "$instruction" '{source:("ARTICLE:\n"+$a),schemaJson:$schema,instruction:$instruction,role:"judge.default",profile:"PROFILE_REMOTE_ONLY",maxOutputTokens:1024}')"
      response="$(curl --fail-with-body --silent --show-error -X POST "$gateway/vrooli.ai_gateway.v1.inference.InferenceService/Run" -H 'content-type: application/json' -d "$request" || echo '{}')"
      [[ -z "$judge_provider" ]] && judge_provider="$(jq -r '.provider // ""' <<<"$response")"
      [[ -z "$judge_model" ]] && judge_model="$(jq -r '.model // ""' <<<"$response")"
      value="$(jq -r '.valueJson // .value_json // ""' <<<"$response")"
      score="null"; quote=""; rationale=""
      if [[ -n "$value" && "$value" != "null" ]]; then
        score="$(jq -r '.score // "null"' <<<"$value" 2>/dev/null || echo null)"
        quote="$(jq -r '.quote // ""' <<<"$value" 2>/dev/null || echo "")"
        rationale="$(jq -r '.rationale // ""' <<<"$value" 2>/dev/null || echo "")"
      fi
      [[ "$score" == "null" || -z "$score" ]] && { score="null"; trial_failures=$(( trial_failures + 1 )); }
      blind_scores="$(jq -c --arg r "$rubric_key" --arg s "$subject_key" --arg f "$(basename "$file")" \
        --argjson sc "$score" --arg q "$quote" --arg ra "$rationale" \
        '. + [{rubric:$r,side:$s,source:$f,score:$sc,quote:(if $q=="" then null else $q end),rationale:(if $ra=="" then null else $ra end)}]' \
        <<<"$blind_scores")"
    }

    while read -r rubric_json; do
      rubric_key="$(jq -r '.key' <<<"$rubric_json")"
      question="$(jq -r '.question' <<<"$rubric_json")"
      levels="$(jq -r '.levels | to_entries | sort_by(.key) | reverse | map("Level \(.key): \(.value)") | join(" ")' <<<"$rubric_json")"
      score_instruction="Score this one article against a fixed rubric. $question $levels Quote one short passage from the article that justifies the level you choose, then give a one-sentence rationale. Judge only the article in front of you."
      for ((sample = 1; sample <= scenario_samples; sample++)); do
        score_one "$rubric_key" "$score_instruction" scenario "$work/scenario-$sample.txt"
      done
      for b in "${baseline_paths[@]}"; do
        score_one "$rubric_key" "$score_instruction" baseline "$b"
      done
      score_one "$rubric_key" "$score_instruction" degraded "$work/degraded.txt"
    done < <(jq -c '.judge_protocol.rubrics[]' "$spec")

    if (( trial_failures > 0 )); then
      blind_status="partial_${trial_failures}_scores_unavailable"
    else
      blind_status="judged"
    fi
  fi
fi

# Aggregation over pairs rather than over averages. A 1-5 rubric score has no
# meaningful mean, and a median would discard the spread that made single-sample
# results untrustworthy. Every scenario document is compared with every baseline
# draft; the rubric goes to whichever side wins more of those pairs.
blind_scoring="$(python3 - "$blind_scores" <<'PYAGG'
import json, sys
rows = json.loads(sys.argv[1])
rubrics, out = {}, []
for r in rows:
    rubrics.setdefault(r["rubric"], {"scenario": [], "baseline": [], "degraded": []})
    if r["score"] is not None:
        rubrics[r["rubric"]][r["side"]].append(int(r["score"]))
scenario_rubrics = baseline_rubrics = tied_rubrics = 0
calibrated_names, uncalibrated_names = [], []
for name, sides in rubrics.items():
    scen, base, deg = sides["scenario"], sides["baseline"], sides["degraded"]
    best = max(scen + base) if (scen or base) else None
    calibrated = bool(deg) and best is not None and max(deg) < best
    wins = sum(1 for a in scen for b in base if a > b)
    losses = sum(1 for a in scen for b in base if a < b)
    ties = sum(1 for a in scen for b in base if a == b)
    winner = "scenario" if wins > losses else "baseline" if losses > wins else "tie"
    if calibrated:
        calibrated_names.append(name)
        if winner == "scenario": scenario_rubrics += 1
        elif winner == "baseline": baseline_rubrics += 1
        else: tied_rubrics += 1
    else:
        uncalibrated_names.append(name)
    out.append({
        "rubric": name, "calibrated": calibrated, "winner": winner,
        "scenario_scores": sorted(scen), "baseline_scores": sorted(base), "degraded_scores": sorted(deg),
        "pair_wins": wins, "pair_losses": losses, "pair_ties": ties,
        "scenario_spread": (max(scen) - min(scen)) if scen else None,
        "baseline_spread": (max(base) - min(base)) if base else None,
    })
json.dump({
    "per_rubric": out,
    "calibrated_rubrics": calibrated_names,
    "uncalibrated_rubrics": uncalibrated_names,
    "scenario_wins": scenario_rubrics,
    "baseline_wins": baseline_rubrics,
    "ties": tied_rubrics,
    "basis": "all-pairs win count per rubric over sampled documents; calibrated rubrics only",
}, sys.stdout)
PYAGG
)"
blind_scenario_wins="$(jq -r '.scenario_wins' <<<"$blind_scoring")"
blind_baseline_wins="$(jq -r '.baseline_wins' <<<"$blind_scoring")"
blind_calibrated="$(jq -r '.calibrated_rubrics | length' <<<"$blind_scoring")"

# Coherence and section count are taken at their worst across the sampled set,
# for the same reason the text measurements are: reporting the first document's
# numbers while three were generated would describe a sample, not the run.
coherence="$(jq -c '{
  semantic_measured: (map(.coherence.semantic_measured // false) | all),
  semantic_section_repetition: ([.[].coherence.semantic_section_repetition // 1] | max),
  cross_section_repetition: ([.[].coherence.cross_section_repetition // 0] | max),
  verdict: {coherent: (map(.coherence.verdict.coherent // false) | all)},
  per_sample: map(.coherence // {}),
  basis: "worst value across sampled scenario documents"
}' <<<"$documents")"
provenance="$(jq -c '{
  per_sample: map(.document_provenance // {}),
  total_cost_micros: ([.[].document_provenance.total_cost_micros // 0] | add),
  providers: ([.[].document_provenance.providers[]?] | unique),
  models: ([.[].document_provenance.models[]?] | unique)
}' <<<"$documents")"
section_count="$(jq '[.[] | (.outline // []) | length] | min' <<<"$documents")"
section_count_max="$(jq '[.[] | (.outline // []) | length] | max' <<<"$documents")"

gates="$(jq -nc \
  --argjson spec "$(jq '.mechanical_gates' "$spec")" \
  --argjson range "$(jq '.section_count_range' "$spec")" \
  --argjson m "$measurements" \
  --argjson coherence "$coherence" \
  --argjson sections "$section_count" \
  --argjson sections_max "$section_count_max" \
  --arg blind_status "$blind_status" \
  --argjson blind_scenario_wins "$blind_scenario_wins" \
  --argjson blind_baseline_wins "$blind_baseline_wins" \
  --argjson blind_calibrated "$blind_calibrated" '
{
  word_count: ($m.word_count >= $spec.min_words and $m.worst_word_count_max <= $spec.max_words),
  sentence_count: ($m.sentence_count >= $spec.min_sentences),
  section_count: ($sections >= $range[0] and $sections_max <= $range[1]),
  paragraph_duplication: ($m.max_paragraph_duplication <= $spec.max_paragraph_duplication),
  adjacent_paragraph_novelty: ($m.min_adjacent_novelty >= $spec.min_adjacent_paragraph_novelty),
  distinct_artifacts_used: ($m.distinct_artifacts_used >= $spec.min_distinct_artifacts_used),
  required_claim_terms: (([$m.samples[].required_terms_present[] | select(. == false)] | length) == 0),
  evidence_grounding: (($m.unsupported_numbers | length) == 0),
  evidence_claim_coverage: ($m.evidence_claim_coverage >= 1.0),
  semantic_measured: (($coherence.semantic_measured // false) == true),
  semantic_section_repetition: (($coherence.semantic_measured // false) == true and ($coherence.semantic_section_repetition // 1) <= $spec.max_semantic_section_repetition),
  coherence: ($coherence.verdict.coherent // false),
  blind_judge_calibrated: ($blind_status == "judged" and $blind_calibrated > 0),
  blind_comparison: ($blind_status == "judged" and $blind_calibrated > 0 and $blind_scenario_wins > $blind_baseline_wins)
}')"
passed="$(jq -r '[to_entries[] | select(.value == false)] | length == 0' <<<"$gates")"

jq -n \
  --argjson generated_texts "$(for ((i=1;i<=scenario_samples;i++)); do jq -Rs . < "$work/scenario-$i.txt"; done | jq -sc .)" \
  --argjson baseline_texts "$(for b in "${baseline_paths[@]}"; do jq -Rs . < "$b"; done | jq -sc .)" \
  --argjson documents "$documents" \
  --arg evidence_source "$evidence_path" \
  --arg draft_id "$draft_id" \
  --argjson probe "$probe" \
  --argjson provenance "$provenance" \
  --argjson measurements "$measurements" \
  --argjson claim_coverage "$claim_coverage" \
  --argjson gates "$gates" \
  --argjson passed "$passed" \
  --arg run_id "$stamp" \
  --arg blind_status "$blind_status" \
  --argjson blind_scoring "$blind_scoring" \
  --argjson blind_scores "$blind_scores" \
  --arg judge_provider "$judge_provider" \
  --arg judge_model "$judge_model" \
  --argjson baseline_shas "$baseline_shas" \
  --argjson scenario_samples "$scenario_samples" \
  --arg baseline_glob "$(jq -r '.baseline_glob' "$spec")" '
{
  run_id: $run_id,
  passed: $passed,
  gates: $gates,
  measurements: $measurements,
  generated_texts: $generated_texts,
  baseline_texts: $baseline_texts,
  evidence_source: $evidence_source,
  content_desk_draft_id: $draft_id,
  documents: $documents,
  coherence: $coherence,
  document_provenance: $provenance,
  claim_coverage: $claim_coverage,
  blind_comparison: {
    status: $blind_status,
    control: "unaided agent: same model, same brief, same artifacts, one call per draft, none of the machinery",
    baseline_glob: $baseline_glob,
    baseline_sha256s: $baseline_shas,
    scenario_samples: $scenario_samples,
    judge: {provider: $judge_provider, model: $judge_model, temperature: 0, basis: "role judge.default; fixed temperature makes repeated identical requests redundant, which is why the sampling is of documents rather than of judgements"},
    tally_basis: "Both sides are sampled and compared over every scenario-baseline pair per rubric. A rubric counts only when the judge scored the restatement-maximal anchor below the better of the two sides on it. Single-sample comparison was retired after the same configuration produced argument scores of 5 and 3 on consecutive runs.",
    scoring: $blind_scoring,
    scores: $blind_scores
  },
  single_call_probe: {
    round: $probe.round,
    provider: ($probe.candidates[0].provenance.provider // "unknown"),
    model: ($probe.candidates[0].provenance.resolved_model_ref // "unknown"),
    cost_micros: ($probe.round.total_cost_micros // 0)
  }
}' | tee "$output"
echo "evidence=$output" >&2
[[ "$passed" == "true" ]]
