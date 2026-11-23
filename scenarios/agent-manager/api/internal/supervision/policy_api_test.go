package supervision

import (
	"context"
	"testing"

	domainpb "github.com/vrooli/vrooli/packages/proto/gen/go/agent-manager/v1/domain"
)

func TestPolicyAPIMethodsFailClosedWithoutStore(t *testing.T) {
	ctx := context.Background()
	cases := []struct {
		name   string
		invoke func(*Service) error
	}{
		{name: "get", invoke: func(s *Service) error {
			_, err := s.GetPolicy(ctx, &domainpb.GetSupervisionPolicyRequest{})
			return err
		}},
		{name: "create candidate", invoke: func(s *Service) error {
			_, err := s.CreatePolicyCandidate(ctx, &domainpb.CreateSupervisionPolicyCandidateRequest{Policy: &domainpb.SupervisionPolicyDefinition{}})
			return err
		}},
		{name: "record outcome", invoke: func(s *Service) error {
			_, err := s.RecordPolicyOutcome(ctx, &domainpb.RecordSupervisionOutcomeRequest{Outcome: &domainpb.SupervisionOutcomeRecord{}})
			return err
		}},
		{name: "evaluate", invoke: func(s *Service) error {
			_, err := s.EvaluatePolicy(ctx, &domainpb.EvaluateSupervisionPolicyRequest{})
			return err
		}},
		{name: "promote", invoke: func(s *Service) error {
			_, err := s.PromotePolicy(ctx, &domainpb.PromoteSupervisionPolicyRequest{})
			return err
		}},
		{name: "reject", invoke: func(s *Service) error {
			_, err := s.RejectPolicy(ctx, &domainpb.RejectSupervisionPolicyRequest{})
			return err
		}},
		{name: "rollback", invoke: func(s *Service) error {
			_, err := s.RollbackPolicy(ctx, &domainpb.RollbackSupervisionPolicyRequest{})
			return err
		}},
		{name: "disable", invoke: func(s *Service) error {
			_, err := s.SetPolicyDisabled(ctx, &domainpb.SetSupervisionPolicyDisabledRequest{})
			return err
		}},
		{name: "list outcomes", invoke: func(s *Service) error {
			_, err := s.ListPolicyOutcomes(ctx, &domainpb.ListSupervisionOutcomesRequest{})
			return err
		}},
	}
	services := []struct {
		name    string
		service *Service
	}{
		{name: "nil service"},
		{name: "missing store", service: &Service{}},
	}
	for _, service := range services {
		t.Run(service.name, func(t *testing.T) {
			for _, tc := range cases {
				t.Run(tc.name, func(t *testing.T) {
					err := tc.invoke(service.service)
					if err == nil {
						t.Fatal("policy API did not report the missing service")
					}
				})
			}
		})
	}
}
