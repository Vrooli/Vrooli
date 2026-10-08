package enrichment

import (
	"context"
	"math"
)

// RetrievalReader is the named semantic consumer of embeddings. It owns no
// provider configuration; retrieval only reads the durable metadata/vector
// rows produced by Service.Embed.
type RetrievalReader struct{ Repo Repository }

func (r RetrievalReader) Embeddings(ctx context.Context, documentHash string) ([]Embedding, error) {
	return r.Repo.ListEmbeddings(ctx, documentHash)
}

// IntakeReader is the named near-duplicate consumer. It is deliberately a
// read-only seam so a gateway outage can leave intake successful and simply
// produce no duplicate claim.
type IntakeReader struct{ Repo Repository }

func (r IntakeReader) NearDuplicate(ctx context.Context, vector []float32, threshold float32) (string, bool, error) {
	if len(vector) == 0 {
		return "", false, nil
	}
	if threshold <= 0 {
		threshold = 0.98
	}
	rows, err := r.Repo.ListAllEmbeddings(ctx)
	if err != nil {
		return "", false, err
	}
	for _, row := range rows {
		if cosine(vector, row.Vector) >= threshold {
			return row.DocumentHash, true, nil
		}
	}
	return "", false, nil
}

func cosine(left, right []float32) float32 {
	if len(left) != len(right) || len(left) == 0 {
		return 0
	}
	var dot, leftNorm, rightNorm float64
	for i := range left {
		dot += float64(left[i] * right[i])
		leftNorm += float64(left[i] * left[i])
		rightNorm += float64(right[i] * right[i])
	}
	if leftNorm == 0 || rightNorm == 0 {
		return 0
	}
	return float32(dot / (math.Sqrt(leftNorm) * math.Sqrt(rightNorm)))
}
