package protocol

import (
	"strings"
	"testing"
)

func TestEncodeDecodeQuerySingleLabel(t *testing.T) {
	qk, _, err := DeriveKeys("test-key")
	if err != nil {
		t.Fatalf("DeriveKeys: %v", err)
	}
	domain := "t.example.com"
	tests := []struct {
		channel uint16
		block   uint16
	}{
		{0, 0},
		{1, 0},
		{1, 5},
		{255, 99},
	}
	for _, tt := range tests {
		qname, err := EncodeQuery(qk, tt.channel, tt.block, domain, QuerySingleLabel)
		if err != nil {
			t.Fatalf("EncodeQuery(%d, %d): %v", tt.channel, tt.block, err)
		}
		if !strings.HasSuffix(qname, "."+domain) {
			t.Errorf("query %q should end with .%s", qname, domain)
		}
		// Single label: only one dot before domain
		subdomain := qname[:len(qname)-len(domain)-1]
		if strings.Contains(subdomain, ".") {
			t.Errorf("single-label query should not have dots in subdomain, got %q", subdomain)
		}
		ch, blk, err := DecodeQuery(qk, qname, domain)
		if err != nil {
			t.Fatalf("DecodeQuery: %v", err)
		}
		if ch != tt.channel || blk != tt.block {
			t.Errorf("got ch=%d blk=%d, want ch=%d blk=%d", ch, blk, tt.channel, tt.block)
		}
	}
}

func TestEncodeDecodeQueryMultiLabel(t *testing.T) {
	qk, _, err := DeriveKeys("test-key")
	if err != nil {
		t.Fatalf("DeriveKeys: %v", err)
	}
	domain := "t.example.com"
	qname, err := EncodeQuery(qk, 3, 7, domain, QueryMultiLabel)
	if err != nil {
		t.Fatal(err)
	}
	// Multi-label mode splits hex across labels; all must be DNS-safe.
	subdomain := qname[:len(qname)-len(domain)-1]
	parts := strings.Split(subdomain, ".")
	if len(parts) < 1 {
		t.Errorf("multi-label query should have at least 1 part, got %d: %q", len(parts), subdomain)
	}
	for _, p := range parts {
		if len(p) == 0 || len(p) > 63 {
			t.Errorf("invalid label length %d in %q", len(p), p)
		}
	}
	ch, blk, err := DecodeQuery(qk, qname, domain)
	if err != nil {
		t.Fatalf("DecodeQuery: %v", err)
	}
	if ch != 3 || blk != 7 {
		t.Errorf("got ch=%d blk=%d, want ch=3 blk=7", ch, blk)
	}
}

func TestEncodeQueryTooLongDomain(t *testing.T) {
	qk, _, err := DeriveKeys("test-key")
	if err != nil {
		t.Fatalf("DeriveKeys: %v", err)
	}

	// 250-char domain should make qname exceed DNS 253-char limit.
	longDomain := strings.Repeat("a", 250)
	_, err = EncodeQuery(qk, 1, 1, longDomain, QueryMultiLabel)
	if err == nil {
		t.Fatal("expected error for too-long domain")
	}
}

func TestSingleLabelNotConfusedWithHex(t *testing.T) {
	qk, _, err := DeriveKeys("test-key")
	if err != nil {
		t.Fatalf("DeriveKeys: %v", err)
	}
	domain := "t.example.com"
	qname, _ := EncodeQuery(qk, 5, 10, domain, QuerySingleLabel)
	ch, blk, err := DecodeQuery(qk, qname, domain)
	if err != nil {
		t.Fatalf("DecodeQuery single-label: %v", err)
	}
	if ch != 5 || blk != 10 {
		t.Errorf("got ch=%d blk=%d, want ch=5 blk=10", ch, blk)
	}
}

func TestDecodeQueryWrongKey(t *testing.T) {
	qk1, _, _ := DeriveKeys("key1")
	qk2, _, _ := DeriveKeys("key2")
	qname, _ := EncodeQuery(qk1, 1, 0, "t.example.com", QuerySingleLabel)
	_, _, err := DecodeQuery(qk2, qname, "t.example.com")
	if err == nil {
		t.Error("expected error when decoding with wrong key")
	}
}

func TestDecodeQueryWrongDomain(t *testing.T) {
	qk, _, _ := DeriveKeys("key")
	qname, _ := EncodeQuery(qk, 1, 0, "t.example.com", QuerySingleLabel)
	_, _, err := DecodeQuery(qk, qname, "t.other.com")
	if err == nil {
		t.Error("expected error for wrong domain")
	}
}

func TestEncodeDecodeResponse(t *testing.T) {
	_, rk, err := DeriveKeys("test-key")
	if err != nil {
		t.Fatal(err)
	}
	data := []byte("Hello World!")
	encoded, err := EncodeResponse(rk, data, DefaultMaxPadding)
	if err != nil {
		t.Fatalf("EncodeResponse: %v", err)
	}
	decoded, err := DecodeResponse(rk, encoded)
	if err != nil {
		t.Fatalf("DecodeResponse: %v", err)
	}
	if string(decoded) != string(data) {
		t.Errorf("got %q, want %q", decoded, data)
	}
}

func TestEncodeDecodeResponseNoPadding(t *testing.T) {
	_, rk, err := DeriveKeys("test-key")
	if err != nil {
		t.Fatal(err)
	}
	data := []byte("No padding test")
	encoded, err := EncodeResponse(rk, data, 0)
	if err != nil {
		t.Fatalf("EncodeResponse: %v", err)
	}
	decoded, err := DecodeResponse(rk, encoded)
	if err != nil {
		t.Fatalf("DecodeResponse: %v", err)
	}
	if string(decoded) != string(data) {
		t.Errorf("got %q, want %q", decoded, data)
	}
}

func TestResponseVaryingSize(t *testing.T) {
	_, rk, _ := DeriveKeys("test-key")
	data := []byte("fixed data")
	sizes := make(map[int]bool)
	for i := 0; i < 50; i++ {
		encoded, err := EncodeResponse(rk, data, 32)
		if err != nil {
			t.Fatal(err)
		}
		sizes[len(encoded)] = true
	}
	if len(sizes) < 2 {
		t.Error("expected varying response sizes with padding, got uniform")
	}
}

func TestDecodeResponseWrongKey(t *testing.T) {
	_, rk1, _ := DeriveKeys("key1")
	_, rk2, _ := DeriveKeys("key2")
	encoded, _ := EncodeResponse(rk1, []byte("data"), 0)
	_, err := DecodeResponse(rk2, encoded)
	if err == nil {
		t.Error("expected error for wrong key")
	}
}

func TestQueryDomainWithTrailingDot(t *testing.T) {
	qk, _, _ := DeriveKeys("key")
	qname, _ := EncodeQuery(qk, 1, 0, "t.example.com", QuerySingleLabel)
	ch, blk, err := DecodeQuery(qk, qname+".", "t.example.com.")
	if err != nil {
		t.Fatalf("DecodeQuery with trailing dot: %v", err)
	}
	if ch != 1 || blk != 0 {
		t.Errorf("got ch=%d blk=%d, want ch=1 blk=0", ch, blk)
	}
}

func TestEncodeDecodeSendQuery(t *testing.T) {
	qk, _, err := DeriveKeys("test-key")
	if err != nil {
		t.Fatalf("DeriveKeys: %v", err)
	}
	domain := "t.example.com"
	msg := []byte("Hello!")

	qname, err := EncodeSendQuery(qk, 3, msg, domain, QuerySingleLabel)
	if err != nil {
		t.Fatalf("EncodeSendQuery: %v", err)
	}

	targetCh, gotMsg, err := DecodeSendQuery(qk, qname, domain)
	if err != nil {
		t.Fatalf("DecodeSendQuery: %v", err)
	}
	if targetCh != 3 {
		t.Errorf("targetChannel = %d, want 3", targetCh)
	}
	if string(gotMsg) != string(msg) {
		t.Errorf("message = %q, want %q", string(gotMsg), string(msg))
	}
}

func TestEncodeDecodeSendQueryNoPassword(t *testing.T) {
	qk, _, err := DeriveKeys("test-key")
	if err != nil {
		t.Fatalf("DeriveKeys: %v", err)
	}
	domain := "t.example.com"
	msg := []byte("No password")

	qname, err := EncodeSendQuery(qk, 1, msg, domain, QuerySingleLabel)
	if err != nil {
		t.Fatalf("EncodeSendQuery: %v", err)
	}

	targetCh, gotMsg, err := DecodeSendQuery(qk, qname, domain)
	if err != nil {
		t.Fatalf("DecodeSendQuery: %v", err)
	}
	if targetCh != 1 {
		t.Errorf("targetChannel = %d, want 1", targetCh)
	}
	if string(gotMsg) != string(msg) {
		t.Errorf("message = %q, want %q", string(gotMsg), string(msg))
	}
}

func TestEncodeDecodeAdminQuery(t *testing.T) {
	qk, _, err := DeriveKeys("test-key")
	if err != nil {
		t.Fatalf("DeriveKeys: %v", err)
	}
	domain := "t.example.com"

	qname, err := EncodeAdminQuery(qk, AdminCmdAddChannel, []byte("testchan"), domain, QuerySingleLabel)
	if err != nil {
		t.Fatalf("EncodeAdminQuery: %v", err)
	}

	gotCmd, gotArg, err := DecodeAdminQuery(qk, qname, domain)
	if err != nil {
		t.Fatalf("DecodeAdminQuery: %v", err)
	}
	if gotCmd != AdminCmdAddChannel {
		t.Errorf("cmd = %d, want %d", gotCmd, AdminCmdAddChannel)
	}
	if string(gotArg) != "testchan" {
		t.Errorf("arg = %q, want %q", string(gotArg), "testchan")
	}
}

func TestEncodeDecodeUpstreamInitQuery(t *testing.T) {
	qk, _, err := DeriveKeys("test-key")
	if err != nil {
		t.Fatalf("DeriveKeys: %v", err)
	}
	init := UpstreamInit{
		SessionID:     0x1122,
		TotalBlocks:   7,
		Kind:          UpstreamKindSend,
		TargetChannel: 15,
	}
	qname, err := EncodeUpstreamInitQuery(qk, init, "t.example.com", QuerySingleLabel)
	if err != nil {
		t.Fatalf("EncodeUpstreamInitQuery: %v", err)
	}
	// Init query should be a single compact label — no data labels
	if strings.Count(strings.TrimSuffix(qname, ".t.example.com"), ".") != 0 {
		t.Fatalf("init query should be a single label, got: %s", qname)
	}
	got, err := DecodeUpstreamInitQuery(qk, qname, "t.example.com")
	if err != nil {
		t.Fatalf("DecodeUpstreamInitQuery: %v", err)
	}
	if *got != init {
		t.Fatalf("got %+v, want %+v", *got, init)
	}
}

func TestEncodeDecodeUpstreamBlockQuery(t *testing.T) {
	qk, _, err := DeriveKeys("test-key")
	if err != nil {
		t.Fatalf("DeriveKeys: %v", err)
	}
	chunk := strings.Repeat("x", MaxUpstreamBlockPayload)
	qname, err := EncodeUpstreamBlockQuery(qk, 0xAABB, 3, []byte(chunk), "t.example.com", QuerySingleLabel)
	if err != nil {
		t.Fatalf("EncodeUpstreamBlockQuery: %v", err)
	}
	if len(qname) > 253 {
		t.Fatalf("upstream block query too long: %d", len(qname))
	}
	sessionID, index, gotChunk, err := DecodeUpstreamBlockQuery(qk, qname, "t.example.com")
	if err != nil {
		t.Fatalf("DecodeUpstreamBlockQuery: %v", err)
	}
	if sessionID != 0xAABB {
		t.Fatalf("sessionID = %#x, want %#x", sessionID, 0xAABB)
	}
	if index != 3 {
		t.Fatalf("index = %d, want 3", index)
	}
	if string(gotChunk) != chunk {
		t.Fatalf("chunk = %q, want %q", string(gotChunk), chunk)
	}
}

func TestEncodeUpstreamInitQueryRejectsInvalidBlockCount(t *testing.T) {
	qk, _, err := DeriveKeys("test-key")
	if err != nil {
		t.Fatalf("DeriveKeys: %v", err)
	}
	_, err = EncodeUpstreamInitQuery(qk, UpstreamInit{SessionID: 1, TotalBlocks: MaxUpstreamBlocks + 1, Kind: UpstreamKindAdmin}, "t.example.com", QuerySingleLabel)
	if err == nil {
		t.Fatal("expected invalid block count error")
	}
}

func TestDecodeQueryRoutesUpstreamInitQuery(t *testing.T) {
	qk, _, err := DeriveKeys("test-key")
	if err != nil {
		t.Fatalf("DeriveKeys: %v", err)
	}
	init := UpstreamInit{
		SessionID:     0xBEEF,
		TotalBlocks:   1,
		Kind:          UpstreamKindSend,
		TargetChannel: 5,
	}
	qname, err := EncodeUpstreamInitQuery(qk, init, "t.example.com", QuerySingleLabel)
	if err != nil {
		t.Fatalf("EncodeUpstreamInitQuery: %v", err)
	}
	// DecodeQuery must extract the channel from multi-label upstream queries
	ch, _, err := DecodeQuery(qk, qname, "t.example.com")
	if err != nil {
		t.Fatalf("DecodeQuery failed on upstream init query: %v", err)
	}
	if ch != UpstreamInitChannel {
		t.Fatalf("channel = %#x, want %#x (UpstreamInitChannel)", ch, UpstreamInitChannel)
	}
}

func TestDecodeQueryRoutesUpstreamBlockQuery(t *testing.T) {
	qk, _, err := DeriveKeys("test-key")
	if err != nil {
		t.Fatalf("DeriveKeys: %v", err)
	}
	qname, err := EncodeUpstreamBlockQuery(qk, 0xAABB, 0, []byte("HI"), "t.example.com", QuerySingleLabel)
	if err != nil {
		t.Fatalf("EncodeUpstreamBlockQuery: %v", err)
	}
	ch, _, err := DecodeQuery(qk, qname, "t.example.com")
	if err != nil {
		t.Fatalf("DecodeQuery failed on upstream block query: %v", err)
	}
	if ch != UpstreamDataChannel {
		t.Fatalf("channel = %#x, want %#x (UpstreamDataChannel)", ch, UpstreamDataChannel)
	}
}
