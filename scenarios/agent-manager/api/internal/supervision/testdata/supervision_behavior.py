"""Given bounded evidence, assert safe dispositions against the shipped evaluator."""
import contextlib
import copy
import io
import json
import sys

source = open(sys.argv[1], encoding="utf-8").read()
base = {"policy":{"version":"test", "event_count_threshold":3,"friction_threshold":.8,
 "quiet_seconds":30,"event_count_enabled":True,"friction_enabled":True,
 "terminal_enabled":True,"deadline_reached":False,"quiet_reached":False,
 "allowed_actions":["observe","park","escalate","wake_parent"]},
 "current_cursor":"before","proposed_next_cursor":"after","allow_inference":False,
 "run_summaries":[{"run_id":"child","status":"running"}]}

class Classifier:
    def classify(self, *args):
        return self
    def head(self, count):
        return [{"provider":"fixture","model":"fixed","applied":{},"valueJson":json.dumps({"classification":"completed", "confidence":.99,"abstain":False,"recommended_action":"wake_parent"})}]

def evaluate(update):
    data=copy.deepcopy(base)
    data.update(update)
    out=io.StringIO()
    with contextlib.redirect_stdout(out):
        exec(compile(source,sys.argv[1],"exec"),{"inputs":data,"ai":Classifier()})
    lines=out.getvalue().splitlines()
    assert len(lines)==1, lines
    result=json.loads(lines[0])
    assert len(out.getvalue().encode())<=4096
    return result["signals"]

r=evaluate({"events":[{"event_id":str(i),"run_id":"child","sequence":i,"event_type":"tool_completed"} for i in range(3)]})
assert r["disposition"]=="unavailable" and r["classification"]!="progress",r
r=evaluate({"run_summaries":[{"run_id":"child","status":"parked"}],"policy":dict(base["policy"],quiet_reached=True)})
assert r["disposition"]=="quiet",r
r=evaluate({"run_summaries":[{"run_id":"child","status":"running","friction_unavailable":True}]})
assert r["disposition"]=="unavailable" and r["next_cursor"]=="before",r
r=evaluate({"friction_episodes":[{"evidence_id":"episode","score":.9,"pattern":"repeated_validation","fingerprint":"stable","owner":"test-genie"}]})
assert r["disposition"]=="signal" and r["classification"]=="stalled",r
r=evaluate({"run_summaries":[{"run_id":"child","status":"completed"}]})
assert r["disposition"]=="terminal",r
r=evaluate({"cursor_reset_required":True})
assert r["disposition"]=="cursor_reset" and r["next_cursor"] is None,r
r=evaluate({"events":[{"event_id":"one","run_id":"child","sequence":1,"event_type":"tool_failed"}]})
assert r["disposition"]=="unavailable" and r["next_cursor"]=="before",r
for status in ("running", "parked"):
    r=evaluate({"run_summaries":[{"run_id":"child","status":status,"friction_unavailable":True}],
        "events":[{"event_id":str(i),"run_id":"child","sequence":i,"event_type":"tool_failed"} for i in range(3)]})
    assert r["disposition"]=="unavailable" and r["next_cursor"]=="before",r
r=evaluate({"allow_inference":True,"events":[{"event_id":"one","run_id":"child","sequence":1,"event_type":"progress_changed"}]})
assert r["disposition"]=="signal" and r["classification"]=="ambiguous",r
print("10 supervised behavior cases passed")
