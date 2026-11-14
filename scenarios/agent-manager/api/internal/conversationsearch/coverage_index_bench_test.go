package conversationsearch

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/require"
	coredb "github.com/vrooli/api-core/database"
)

func BenchmarkCoverageMessageMetadataIndex(b *testing.B) {
	db, err := sqlx.Connect("sqlite", fmt.Sprintf("file:coverage-%s?mode=memory&cache=shared", b.Name()))
	require.NoError(b, err)
	b.Cleanup(func() { require.NoError(b, db.Close()) })
	ctx := context.Background()
	require.NoError(b, coredb.EnsureSchemas(ctx, db, coredb.SchemaProviderFunc(Schema)))
	r := NewSQLiteRepository(db)
	for n := 0; n < 4000; n++ {
		d := testDocument()
		d.DocumentID = fmt.Sprintf("document-%06d", n)
		d.SourceEventID = fmt.Sprintf("event-%06d", n)
		d.SourceRunID = fmt.Sprintf("run-%03d", n%100)
		d.SourceMessageID = fmt.Sprintf("message-%04d", n%1000)
		d.Content = strings.Repeat("bounded fixture ", 64)
		d.Visible = n%10 != 0
		require.NoError(b, r.UpsertDocument(ctx, d))
	}
	for _, indexed := range []bool{false, true} {
		name := "legacy-source-index"
		if indexed {
			name = "visible-message-index"
			require.NoError(b, coredb.EnsureSchemas(ctx, db, coredb.SchemaProviderFunc(Schema)))
		} else {
			_, err := db.Exec(`DROP INDEX idx_conversation_search_visible_messages`)
			require.NoError(b, err)
		}
		b.Run(name, func(b *testing.B) {
			b.ReportAllocs()
			for n := 0; n < b.N; n++ {
				messages, documents, lexical, err := r.queryCoverage(ctx)
				require.NoError(b, err)
				require.Equal(b, uint64(900), messages)
				require.Equal(b, uint64(3600), documents)
				require.Equal(b, documents, lexical)
			}
		})
	}
}
