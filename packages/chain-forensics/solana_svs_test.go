package chainforensics

import (
	"context"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math/big"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestSolanaRPCDecodesPinnedLibraCandidateJupiterRFQSwapFromRetainedRawTransaction(t *testing.T) {
	const signature = "tC9VgKe8NNjaZPm5hQU9dqG16aF4zwe4PTyxUctQx7tLK9BoZnd11kTGzehzLa2JERbmzUbkALJFbQe2KL5rMwu"
	const wantHash = "70c89d836b727f5350fc46fed9886bc7b761f7f81d2602ffbf1fb8e62569a36c"
	body, err := os.ReadFile(filepath.Join("testdata", "cf6-solana-svs-20261116", "getTransaction-response.json"))
	if err != nil {
		t.Fatal(err)
	}
	h := sha256.Sum256(body)
	if fmt.Sprintf("%x", h) != wantHash {
		t.Fatal("retained SVS finalized transaction changed")
	}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var request struct {
			Method string            `json:"method"`
			Params []json.RawMessage `json:"params"`
		}
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			t.Error(err)
			return
		}
		if request.Method != "getTransaction" || len(request.Params) != 2 {
			t.Errorf("unexpected request: %+v", request)
			http.Error(w, "unexpected request", http.StatusBadRequest)
			return
		}
		var sig string
		var opts map[string]any
		_ = json.Unmarshal(request.Params[0], &sig)
		_ = json.Unmarshal(request.Params[1], &opts)
		if sig != signature || opts["encoding"] != "json" || opts["commitment"] != "finalized" {
			t.Errorf("request did not bind exact finalized raw transaction: sig=%s opts=%v", sig, opts)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write(body)
	}))
	defer server.Close()
	client, err := NewSolanaRPCClient(server.URL, server.Client(), nil)
	if err != nil {
		t.Fatal(err)
	}
	captured, err := client.FetchTransaction(context.Background(), signature)
	if err != nil {
		t.Fatal(err)
	}
	if captured.Failed || captured.Transaction.Slot != 380820413 || captured.Transaction.BlockTime == nil || *captured.Transaction.BlockTime != 1763440531 {
		t.Fatalf("wrong finalized identity: failed=%v tx=%+v", captured.Failed, captured.Transaction)
	}
	if captured.Source.RawResponseSHA256 != wantHash || len(captured.Source.RawResponseBody) != len(body) {
		t.Fatalf("raw source envelope was not retained: %+v", captured.Source)
	}
	if captured.Source.SHA256 != "3bda4de002929c96ea5c9e58fe914674d2bc6b9a8e1de6b27a7108f6687b9957" {
		t.Fatalf("result content hash mismatch: %s", captured.Source.SHA256)
	}
	var usdc, sol, exactTransfers, balanceDeltas int
	for _, occurrence := range captured.Occurrences {
		if occurrence.Measurement != MeasurementInstructionTransfer {
			if occurrence.Measurement == MeasurementNetBalanceDelta {
				balanceDeltas++
			}
			continue
		}
		exactTransfers++
		switch occurrence.Asset {
		case "SPL:EPjFWdd5AufqSSqeM2qN1xzybapC8G4wEGGkZwyTDt1v":
			usdc++
			if occurrence.EventIndex != 3 || occurrence.From != "3nHwgMhytEC2wkYGpg3LxQMfNzFseCNDGA48NfRExNgH" || occurrence.To != "6jz3UuC5tKeYGt5FiX18LRDEeceDmC55jmn5cDUL8wh7" || occurrence.Atomic != "500455096778" || occurrence.Scale != 6 {
				t.Fatalf("unexpected exact USDC instruction transfer: %+v", occurrence)
			}
		case "SOL":
			sol++
			if occurrence.From != "CreQJ2t94QK5dsxUZGXfPJ8Nx7wA9LHr5chxjSMkbNft" || occurrence.To != "DefcyKc4yAjRsCLZjdxWuSUzVohXtLna9g22y3pBCm2z" || occurrence.Atomic != "3746294021993" {
				t.Fatalf("unexpected exact native transfer: %+v", occurrence)
			}
		}
	}
	if usdc != 1 || sol != 1 || exactTransfers != 3 || balanceDeltas == 0 || len(captured.PumpInstructions) != 0 {
		t.Fatalf("unexpected decoded transfer/Pump counts: USDC=%d SOL=%d transfers=%d balanceDeltas=%d Pump=%d", usdc, sol, exactTransfers, balanceDeltas, len(captured.PumpInstructions))
	}
	var fillData []byte
	for _, ix := range captured.Transaction.Transaction.Message.Instructions {
		if ix.ProgramID == "61DFfeTKM7trxYcPQCM78bJ794ddZprZpAwAnLiwTpYH" {
			fillData, _ = decodeBase58(ix.Data)
		}
	}
	fillDiscriminator := sha256.Sum256([]byte("global:fill"))
	if len(fillData) != 35 || hex.EncodeToString(fillData[:8]) != hex.EncodeToString(fillDiscriminator[:8]) || binary.LittleEndian.Uint64(fillData[8:16]) != 500455096778 || binary.LittleEndian.Uint64(fillData[16:24]) != 3746294021993 {
		t.Fatalf("raw Fill bytes did not preserve the observed discriminator and matching transfer amounts: %x", fillData)
	}
	if len(captured.JupiterOrderEngineFills) != 1 {
		t.Fatalf("expected one IDL-decoded Jupiter Order Engine fill, got %+v", captured.JupiterOrderEngineFills)
	}
	fill := captured.JupiterOrderEngineFills[0]
	if fill.Index != 2 || fill.ProgramID != "61DFfeTKM7trxYcPQCM78bJ794ddZprZpAwAnLiwTpYH" || fill.Taker != "DefcyKc4yAjRsCLZjdxWuSUzVohXtLna9g22y3pBCm2z" || fill.Maker != "CreQJ2t94QK5dsxUZGXfPJ8Nx7wA9LHr5chxjSMkbNft" || fill.InputMint != "EPjFWdd5AufqSSqeM2qN1xzybapC8G4wEGGkZwyTDt1v" || fill.OutputMint != "So11111111111111111111111111111111111111112" || fill.InputAmount != "500455096778" || fill.OutputAmount != "3746294021993" || fill.ExpireAt != 1763440581 || fill.TrailingDataHex != "020000" || fill.SourceID != captured.Source.ID {
		t.Fatalf("unexpected pinned Jupiter RFQ fill decode: %+v", fill)
	}
	if !strings.Contains(captured.Source.Limitations, "Raw account indexes are resolved") {
		t.Fatalf("source normalization boundary missing: %s", captured.Source.Limitations)
	}
}

func TestSolanaRPCDecodesImmediatelyPriorJupiterRivalFlow(t *testing.T) {
	const signature = "3UXry73WHqtzvZGb2dS6aVwr3obvXV4G7Wi3CRVKZtAog1ccRe1JRpVNcYMXKMyjUq2ZQosiutFGJZVjnQ35g638"
	const wantHash = "2ff1386666b1d086f1c9102d9782ed11d5929709fa1f410a99d5e3e6c4bf460b"
	body, err := os.ReadFile(filepath.Join("testdata", "cf6-solana-svs-20261116", "prior-neighbor-response.json"))
	if err != nil {
		t.Fatal(err)
	}
	h := sha256.Sum256(body)
	if fmt.Sprintf("%x", h) != wantHash {
		t.Fatal("retained prior transaction changed")
	}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write(body)
	}))
	defer server.Close()
	client, err := NewSolanaRPCClient(server.URL, server.Client(), nil)
	if err != nil {
		t.Fatal(err)
	}
	captured, err := client.FetchTransaction(context.Background(), signature)
	if err != nil {
		t.Fatal(err)
	}
	if captured.Failed || captured.Transaction.Slot != 380820388 || captured.Transaction.BlockTime == nil || *captured.Transaction.BlockTime != 1763440521 {
		t.Fatalf("wrong prior identity: failed=%v tx=%+v", captured.Failed, captured.Transaction)
	}
	if captured.Source.RawResponseSHA256 != wantHash {
		t.Fatalf("raw source hash mismatch: %s", captured.Source.RawResponseSHA256)
	}
	var jupiter bool
	for _, ix := range captured.Transaction.Transaction.Message.Instructions {
		if ix.ProgramID == "JUP6LkbZbjS1jKKwapdHNy74zcZ3tLUZoi5QNyVTaV4" {
			jupiter = true
		}
	}
	if !jupiter {
		t.Fatal("prior transaction lost its Jupiter instruction identity")
	}
	if len(captured.UnresolvedTransfers) == 0 {
		t.Fatal("prior routed swap should retain legacy SPL transfers whose mint is absent from the transaction balance tables")
	}
	found := false
	for _, unresolved := range captured.UnresolvedTransfers {
		if unresolved.EventIndex == 17 && unresolved.ProgramID == "TokenkegQfeZyiNwAJbNbGKPFXCWuBvf9Ss623VQ5DA" && unresolved.SourceID == captured.Source.ID && len(unresolved.Accounts) >= 2 && unresolved.Data != "" {
			found = true
		}
	}
	if !found || !strings.Contains(captured.Source.Limitations, "retained in unresolvedTransfers") {
		t.Fatalf("unknown-mint SPL movement was not preserved with its limit: unresolved=%+v source=%s", captured.UnresolvedTransfers, captured.Source.Limitations)
	}
}

func TestSolanaRPCClassifiesBoundedPriorSignerPageAsJupiterRivalFlows(t *testing.T) {
	type record struct {
		Index        int    `json:"index"`
		Signature    string `json:"signature"`
		ResponseHash string `json:"sha256"`
	}
	var page struct {
		Records []record `json:"records"`
	}
	metadata, err := os.ReadFile(filepath.Join("testdata", "cf6-solana-svs-20261116", "prior-page-transaction-captures.json"))
	if err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(metadata, &page); err != nil {
		t.Fatal(err)
	}
	if len(page.Records) != 25 {
		t.Fatalf("expected full bounded 25-signature page, got %d", len(page.Records))
	}
	bodies := make(map[string][]byte, len(page.Records))
	for _, item := range page.Records {
		path := filepath.Join("testdata", "cf6-solana-svs-20261116", fmt.Sprintf("prior-page-%02d-%s-response.json", item.Index, item.Signature[:8]))
		body, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		h := sha256.Sum256(body)
		if fmt.Sprintf("%x", h) != item.ResponseHash {
			t.Fatalf("page transaction %d hash changed", item.Index)
		}
		bodies[item.Signature] = body
	}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var request struct {
			Params []json.RawMessage `json:"params"`
		}
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			t.Error(err)
			return
		}
		var signature string
		if len(request.Params) == 0 || json.Unmarshal(request.Params[0], &signature) != nil {
			t.Error("missing signature")
			return
		}
		body, ok := bodies[signature]
		if !ok {
			t.Errorf("request escaped the pinned history page: %s", signature)
			http.Error(w, "not in page", http.StatusBadRequest)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write(body)
	}))
	defer server.Close()
	client, err := NewSolanaRPCClient(server.URL, server.Client(), nil)
	if err != nil {
		t.Fatal(err)
	}
	debridge := map[string]bool{
		"src5qyZHqTqecJV4aY6Cb6zDZLMDzrDKKezs22MPHr4":  true,
		"dst5MGcFPoBeREFAA5E3tU5ij8m5uVYwkzkSAbsLbNo":  true,
		"DEbrdGj3HsRsAzx6uH4MKyREKxVAfBydijLUF3ygsFfh": true,
	}
	var fillCount int
	var totalInput uint64
	var totalOutput uint64
	makers := map[string]int{}
	for _, item := range page.Records {
		captured, err := client.FetchTransaction(context.Background(), item.Signature)
		if err != nil {
			t.Fatalf("capture prior-page signature %s: %v", item.Signature, err)
		}
		if captured.Failed || captured.Source.RawResponseSHA256 != item.ResponseHash {
			t.Fatalf("page capture identity/source mismatch for %s", item.Signature)
		}
		for _, ix := range captured.Transaction.Transaction.Message.Instructions {
			if debridge[ix.ProgramID] {
				t.Fatalf("DLN/gate program in bounded signer page: %s", item.Signature)
			}
		}
		for _, group := range captured.Transaction.Meta.InnerInstructions {
			for _, ix := range group.Instructions {
				if debridge[ix.ProgramID] {
					t.Fatalf("inner DLN/gate program in bounded signer page: %s", item.Signature)
				}
			}
		}
		var result struct {
			Meta struct {
				LogMessages []string `json:"logMessages"`
			} `json:"meta"`
		}
		if err := json.Unmarshal(captured.Source.ResultBody, &result); err != nil {
			t.Fatal(err)
		}
		for _, line := range result.Meta.LogMessages {
			for program := range debridge {
				if strings.Contains(line, program) {
					t.Fatalf("DLN/gate invocation log in bounded signer page: %s", item.Signature)
				}
			}
		}
		for _, fill := range captured.JupiterOrderEngineFills {
			fillCount++
			if fill.Taker != "DefcyKc4yAjRsCLZjdxWuSUzVohXtLna9g22y3pBCm2z" || fill.InputMint != "EPjFWdd5AufqSSqeM2qN1xzybapC8G4wEGGkZwyTDt1v" || fill.OutputMint != "So11111111111111111111111111111111111111112" || fill.InputAmount != "500000000000" {
				t.Fatalf("unexpected role/asset/amount in adjacent Jupiter fill: %+v", fill)
			}
			makers[fill.Maker]++
			amount, ok := new(big.Int).SetString(fill.InputAmount, 10)
			if !ok {
				t.Fatal("invalid decoded input amount")
			}
			totalInput += amount.Uint64()
			output, ok := new(big.Int).SetString(fill.OutputAmount, 10)
			if !ok {
				t.Fatal("invalid decoded output amount")
			}
			totalOutput += output.Uint64()
		}
	}
	if fillCount != 13 || totalInput != 6_500_000_000_000 || totalOutput != 48_908_108_576_761 || makers["BMnT51N4iSNhWU5PyFFgWwFvN1jgaiiDr9ZHgnkm3iLJ"] != 12 || makers["CreQJ2t94QK5dsxUZGXfPJ8Nx7wA9LHr5chxjSMkbNft"] != 1 {
		t.Fatalf("bounded-page Jupiter rival set changed: fillCount=%d input=%d output=%d makers=%v", fillCount, totalInput, totalOutput, makers)
	}
}
