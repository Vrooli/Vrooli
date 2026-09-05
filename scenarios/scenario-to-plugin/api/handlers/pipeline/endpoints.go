package pipeline

import "scenario-to-plugin/internal/module"

var Endpoints = []module.EndpointDescriptor{
	{ID: "declaration_get", Path: "/vrooli.scenario_to_plugin.v1.declaration.DeclarationService/GetDeclaration", Method: "POST", Summary: "Read a governed plugin declaration."},
	{ID: "declaration_readiness", Path: "/vrooli.scenario_to_plugin.v1.declaration.DeclarationService/ListReadiness", Method: "POST", Summary: "Report named publish blockers."},
	{ID: "composition_compose", Path: "/vrooli.scenario_to_plugin.v1.composition.CompositionService/Compose", Method: "POST", Summary: "Compose an Agent Plugin tree."},
	{ID: "composition_get", Path: "/vrooli.scenario_to_plugin.v1.composition.CompositionService/GetPackage", Method: "POST", Summary: "Read a package record."},
	{ID: "distribution_report", Path: "/api/v1/distributability", Method: "POST", Summary: "Report skills distributable to a target CLI surface."},
	{ID: "conformance_check", Path: "/vrooli.scenario_to_plugin.v1.conformance.ConformanceService/Check", Method: "POST", Summary: "Run fail-closed conformance."},
	{ID: "attestation_attest", Path: "/vrooli.scenario_to_plugin.v1.attestation.AttestationService/Attest", Method: "POST", Summary: "Create digest-bound attestations."},
	{ID: "rehearsal_run", Path: "/vrooli.scenario_to_plugin.v1.rehearsal.RehearsalService/Run", Method: "POST", Summary: "Rehearse installation in a clean room."},
	{ID: "distribution_publish", Path: "/vrooli.scenario_to_plugin.v1.distribution.DistributionService/Publish", Method: "POST", Summary: "Publish after governance approval."},
	{ID: "distribution_revoke", Path: "/vrooli.scenario_to_plugin.v1.distribution.DistributionService/Revoke", Method: "POST", Summary: "Revoke recorded publications."},
}
