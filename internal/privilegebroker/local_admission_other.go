//go:build !linux

package privilegebroker

import (
	"context"
	"fmt"
)

func installLocalAdmissionPolicy(context.Context, string, uint32) error {
	return fmt.Errorf("local service admission requires Linux")
}
func installedLocalAdmission(uint32, Executor) (*LocalAdmissionManager, error) {
	return nil, fmt.Errorf("local service admission requires Linux")
}
