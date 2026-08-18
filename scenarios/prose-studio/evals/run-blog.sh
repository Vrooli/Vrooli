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

generation_instruction="Use only factual claims supported by this evidence brief. Do not invent metrics, infrastructure, failure incidents, implementation details, or outcomes; every number you write must appear in the brief. Explain each evidence claim once, with distinct section purposes; do not restate the same workflow or conclusion in multiple sections. Evidence claims: $evidence_claims. Evidence source: $evidence_text. Explicitly use these exact terms where they fit naturally: $claims_prompt."

# The single-call agent path (PS-P0-016) is exercised as its own probe. Its cost
# and provider belong to that request and are recorded separately: reporting a
# standalone generate call's accounting as the document's was reporting a
# different request entirely.
probe="$(post /api/v1/prose/generate "$(jq -nc --arg p "$profile" --arg q "$subject" --arg instruction "$generation_instruction" '{profile_key:$p,query:($q+" "+$instruction),include_candidates:true}')")"

document="$(post /api/v1/prose/documents "$(jq -nc --arg p "$profile" --arg q "$subject" --arg instruction "$generation_instruction" '{document:{title:($q+" "+$instruction),profile_key:$p},sections:[]}')")"
document="$(jq -c '.document // .' <<<"$document")"
jq -r '.assembled_text // ""' <<<"$document" > "$work/assembled.txt"
cp "$evidence_path" "$work/evidence.txt"

# ---------------------------------------------------------------------------
# Independent measurement. Nothing below imports the service's textmetrics.
# ---------------------------------------------------------------------------
python3 - "$work/assembled.txt" "$work/evidence.txt" "$spec" > "$work/measurements.json" <<'PY'
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

evidence_numbers = numbers(evidence)
unsupported_numbers = sorted(numbers(text) - evidence_numbers)

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

required = spec["mechanical_gates"].get("required_claim_terms", [])
lowered = text.lower()

json.dump({
    "word_count": len(words(text)),
    "sentence_count": len(sentences),
    "paragraph_count": len(paragraphs),
    "max_paragraph_duplication": round(max_dup, 4),
    "most_duplicated_pair": dup_pair,
    "unsupported_numbers": unsupported_numbers,
    "required_terms_present": {term: (term.lower() in lowered) for term in required},
    "evidence_claims": claim_hits,
    "evidence_claim_coverage": round(sum(1 for c in claim_hits if c["covered"]) / len(claim_hits), 4) if claim_hits else 0.0,
    "basis": "independent measurement in evals/run-blog.sh; shares no implementation with packages/textmetrics",
}, sys.stdout, indent=2)
PY
measurements="$(cat "$work/measurements.json")"

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
blind_verdict="null"
blind_rationale=""
blind_label_a="baseline"
blind_label_b="scenario"
if [[ "$(jq -r '.mechanical_gates.blind_comparison_required // false' "$spec")" == "true" ]]; then
  gateway="${AI_GATEWAY_URL:-}"
  if [[ -z "$gateway" ]]; then
    blind_status="unavailable_no_gateway"
  else
    # Order is decided by the run stamp so a reader can reproduce which text was
    # which, while the judge sees no authorship signal.
    if (( $(printf '%s' "$stamp" | cksum | cut -d' ' -f1) % 2 == 0 )); then
      blind_label_a="scenario"; blind_label_b="baseline"
      a_text="$(cat "$work/assembled.txt")"; b_text="$(cat "$evidence_path")"
    else
      a_text="$(cat "$evidence_path")"; b_text="$(cat "$work/assembled.txt")"
    fi
    # additionalProperties, like minItems, is outside the gateway's enforceable
    # schema subset and is refused rather than ignored.
    schema='{"type":"object","properties":{"winner":{"type":"string","enum":["A","B","tie"]},"rationale":{"type":"string"}},"required":["winner","rationale"]}'
    instruction='You are comparing two articles on the same subject. Decide which reads as the better article for a technical audience: which one argues something, progresses rather than restating itself, and is concrete. Ignore length and formatting differences. Answer with the winning label and a one-sentence rationale.'
    blind_request="$(jq -nc --arg a "$a_text" --arg b "$b_text" --arg schema "$schema" --arg instruction "$instruction" '{source:("ARTICLE A:\n"+$a+"\n\nARTICLE B:\n"+$b),schemaJson:$schema,instruction:$instruction,role:"judge.default",profile:"PROFILE_REMOTE_ONLY",maxOutputTokens:512}')"
    blind_response="$(curl --fail-with-body --silent --show-error -X POST "$gateway/vrooli.ai_gateway.v1.inference.InferenceService/Run" -H 'content-type: application/json' -d "$blind_request" || echo '{}')"
    blind_value="$(jq -r '.valueJson // .value_json // ""' <<<"$blind_response")"
    if [[ -n "$blind_value" && "$blind_value" != "null" ]]; then
      winner="$(jq -r '.winner // ""' <<<"$blind_value")"
      blind_rationale="$(jq -r '.rationale // ""' <<<"$blind_value")"
      case "$winner" in
        A) blind_verdict="\"$blind_label_a\"" ;;
        B) blind_verdict="\"$blind_label_b\"" ;;
        tie) blind_verdict='"tie"' ;;
        *) blind_verdict="null" ;;
      esac
      blind_status="judged"
    else
      blind_status="unavailable_no_verdict"
    fi
  fi
fi

coherence="$(jq -c '.coherence // {}' <<<"$document")"
provenance="$(jq -c '.document_provenance // {}' <<<"$document")"
section_count="$(jq '(.outline // []) | length' <<<"$document")"

gates="$(jq -nc \
  --argjson spec "$(jq '.mechanical_gates' "$spec")" \
  --argjson range "$(jq '.section_count_range' "$spec")" \
  --argjson m "$measurements" \
  --argjson coherence "$coherence" \
  --argjson sections "$section_count" \
  --arg blind_status "$blind_status" \
  --argjson blind_verdict "$blind_verdict" \
  --arg scenario_label "scenario" '
{
  word_count: ($m.word_count >= $spec.min_words and $m.word_count <= $spec.max_words),
  sentence_count: ($m.sentence_count >= $spec.min_sentences),
  section_count: ($sections >= $range[0] and $sections <= $range[1]),
  paragraph_duplication: ($m.max_paragraph_duplication <= $spec.max_paragraph_duplication),
  required_claim_terms: (([$m.required_terms_present[] | select(. == false)] | length) == 0),
  evidence_grounding: (($m.unsupported_numbers | length) == 0),
  evidence_claim_coverage: ($m.evidence_claim_coverage >= 1.0),
  semantic_measured: (($coherence.semantic_measured // false) == true),
  semantic_section_repetition: (($coherence.semantic_measured // false) == true and ($coherence.semantic_section_repetition // 1) <= $spec.max_semantic_section_repetition),
  coherence: ($coherence.verdict.coherent // false),
  blind_comparison: ($blind_status == "judged" and ($blind_verdict == $scenario_label or $blind_verdict == "tie"))
}')"
passed="$(jq -r '[to_entries[] | select(.value == false)] | length == 0' <<<"$gates")"

jq -n \
  --arg generated "$(cat "$work/assembled.txt")" \
  --arg baseline "$(cat "$root/$(jq -r '.baseline' "$spec")")" \
  --arg evidence_source "$evidence_path" \
  --arg draft_id "$draft_id" \
  --argjson probe "$probe" \
  --argjson document "$document" \
  --argjson provenance "$provenance" \
  --argjson measurements "$measurements" \
  --argjson claim_coverage "$claim_coverage" \
  --argjson gates "$gates" \
  --argjson passed "$passed" \
  --arg run_id "$stamp" \
  --arg blind_status "$blind_status" \
  --argjson blind_verdict "$blind_verdict" \
  --arg blind_rationale "$blind_rationale" \
  --arg blind_label_a "$blind_label_a" \
  --arg blind_label_b "$blind_label_b" '
{
  run_id: $run_id,
  passed: $passed,
  gates: $gates,
  measurements: $measurements,
  generated_text: $generated,
  baseline_text: $baseline,
  evidence_source: $evidence_source,
  content_desk_draft_id: $draft_id,
  document: $document,
  document_provenance: $provenance,
  claim_coverage: $claim_coverage,
  blind_comparison: {
    status: $blind_status,
    verdict: $blind_verdict,
    rationale: (if $blind_rationale == "" then null else $blind_rationale end),
    labels: {A: $blind_label_a, B: $blind_label_b}
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
