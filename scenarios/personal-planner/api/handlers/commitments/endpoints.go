package commitments

import (
	c "github.com/vrooli/vrooli/packages/proto/gen/go/personal-planner/v1/commitments/commitments_v1connect"
	"personal-planner/internal/module"
)

var Endpoints = []module.EndpointDescriptor{
	{ID: "commitments_list", Path: c.CommitmentsServiceListCommitmentsProcedure, Method: "POST", Summary: "List commitments", Description: "Lists explicit promises and their current lifecycle state.", Category: "commitments"},
	{ID: "commitments_create", Path: c.CommitmentsServiceCreateCommitmentProcedure, Method: "POST", Summary: "Create commitment", Description: "Records a proposed or active promise without inventing acknowledgment.", Category: "commitments"},
	{ID: "commitments_update_state", Path: c.CommitmentsServiceUpdateCommitmentStateProcedure, Method: "POST", Summary: "Update commitment state", Description: "Moves a commitment through its explicit lifecycle.", Category: "commitments"},
	{ID: "commitments_revise", Path: c.CommitmentsServiceReviseCommitmentProcedure, Method: "POST", Summary: "Revise commitment", Description: "Stores a new promise revision while retaining the original boundary.", Category: "commitments"},
}
