Repair the supplied Plan Manager plan using only the immutable plan content and concrete validation findings. Do not claim a plan is valid: Plan Manager revalidates after this workflow. Return only JSON matching the result contract. If repair cannot be completed safely within the stated attempt budget, return needs_attention or abstained with a specific reason.

<entity>{{.entity}}</entity>
<plan_reference>{{.plan_reference}}</plan_reference>
<plan_frontier_digest>{{.plan_digest}}</plan_frontier_digest>
<plan_content>{{.plan_content}}</plan_content>
<validation_findings>{{.validation_findings}}</validation_findings>
<constraints>{{.constraints}}</constraints>