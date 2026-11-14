//go:build treasury_rehearsal

package receiptfunding

import (
	"context"

	"github.com/vrooli/api-core/database"
)

func lpbsCreditLotRequestContext(ctx context.Context) context.Context {
	return database.WithTestMode(ctx)
}
