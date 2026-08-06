from host.engine import SessionKernel


def test_inference_reports_ai_gateway_promotion_blocker():  # [REQ:PRT-P1-002]
    result = SessionKernel().execute("vrooli.ai.classify('hello')")
    assert result["ok"] is False
    assert "ai-gateway inference" in result["error"]
    assert "promotion" in result["error"]


def test_delegation_without_bridge_is_explicitly_unavailable():
    result = SessionKernel().execute("vrooli.agent.run(owner='missing', workflow_key='missing')")
    assert result["ok"] is False
    assert "agent-manager delegation" in result["error"]
