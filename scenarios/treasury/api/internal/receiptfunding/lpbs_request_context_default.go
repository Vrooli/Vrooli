//go:build !treasury_rehearsal

package receiptfunding

import "context"

func lpbsCreditLotRequestContext(ctx context.Context) context.Context { return ctx }
