package e2e_test

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/sartoopjj/thefeed/internal/client"
	"github.com/sartoopjj/thefeed/internal/protocol"
)

func TestE2E_FetchMetadataThroughDNS(t *testing.T) {
	domain := "feed.example.com"
	passphrase := "test-secret-key-123"
	channels := []string{"news", "tech"}

	msgs := map[int][]protocol.Message{
		1: {
			{ID: 100, Timestamp: 1700000000, Text: "Hello from news"},
			{ID: 101, Timestamp: 1700000001, Text: "Second news"},
		},
		2: {
			{ID: 200, Timestamp: 1700000010, Text: "Tech update"},
		},
	}

	resolver, cancel := startDNSServer(t, domain, passphrase, channels, msgs)
	defer cancel()

	fetcher, err := client.NewFetcher(domain, passphrase, []string{resolver})
	if err != nil {
		t.Fatalf("create fetcher: %v", err)
	}
	fetcher.SetActiveResolvers([]string{resolver})

	meta, err := fetcher.FetchMetadata(context.Background())
	if err != nil {
		t.Fatalf("fetch metadata: %v", err)
	}

	if len(meta.Channels) != 2 {
		t.Fatalf("expected 2 channels, got %d", len(meta.Channels))
	}
	if meta.Channels[0].Name != "news" {
		t.Errorf("channel 0 name = %q, want %q", meta.Channels[0].Name, "news")
	}
	if meta.Channels[1].Name != "tech" {
		t.Errorf("channel 1 name = %q, want %q", meta.Channels[1].Name, "tech")
	}
	if meta.Channels[0].LastMsgID != 100 {
		t.Errorf("channel 0 lastMsgID = %d, want 100", meta.Channels[0].LastMsgID)
	}
}

func TestE2E_FetchChannelMessages(t *testing.T) {
	domain := "feed.example.com"
	passphrase := "e2e-pass-456"
	channels := []string{"updates"}

	msgs := map[int][]protocol.Message{
		1: {
			{ID: 1, Timestamp: 1700000000, Text: "First message"},
			{ID: 2, Timestamp: 1700000001, Text: "Second message"},
			{ID: 3, Timestamp: 1700000002, Text: "Third message"},
		},
	}

	resolver, cancel := startDNSServer(t, domain, passphrase, channels, msgs)
	defer cancel()

	fetcher, err := client.NewFetcher(domain, passphrase, []string{resolver})
	if err != nil {
		t.Fatalf("create fetcher: %v", err)
	}
	fetcher.SetActiveResolvers([]string{resolver})

	meta, err := fetcher.FetchMetadata(context.Background())
	if err != nil {
		t.Fatalf("fetch metadata: %v", err)
	}

	blockCount := int(meta.Channels[0].Blocks)
	if blockCount <= 0 {
		t.Fatal("expected blocks > 0")
	}

	fetchedMsgs, err := fetcher.FetchChannel(context.Background(), 1, blockCount)
	if err != nil {
		t.Fatalf("fetch channel: %v", err)
	}

	if len(fetchedMsgs) != 3 {
		t.Fatalf("expected 3 messages, got %d", len(fetchedMsgs))
	}

	for i, want := range msgs[1] {
		got := fetchedMsgs[i]
		if got.ID != want.ID || got.Text != want.Text {
			t.Errorf("message %d: got {ID:%d Text:%q}, want {ID:%d Text:%q}",
				i, got.ID, got.Text, want.ID, want.Text)
		}
	}
}

func TestE2E_FetchWithDoubleLabel(t *testing.T) {
	domain := "feed.example.com"
	passphrase := "double-label-test"
	channels := []string{"channel1"}

	msgs := map[int][]protocol.Message{
		1: {{ID: 10, Timestamp: 1700000000, Text: "Double label message"}},
	}

	resolver, cancel := startDNSServer(t, domain, passphrase, channels, msgs)
	defer cancel()

	fetcher, err := client.NewFetcher(domain, passphrase, []string{resolver})
	if err != nil {
		t.Fatalf("create fetcher: %v", err)
	}
	fetcher.SetActiveResolvers([]string{resolver})
	fetcher.SetQueryMode(protocol.QueryMultiLabel)

	meta, err := fetcher.FetchMetadata(context.Background())
	if err != nil {
		t.Fatalf("fetch metadata: %v", err)
	}

	fetchedMsgs, err := fetcher.FetchChannel(context.Background(), 1, int(meta.Channels[0].Blocks))
	if err != nil {
		t.Fatalf("fetch channel: %v", err)
	}

	if len(fetchedMsgs) != 1 {
		t.Fatalf("expected 1 message, got %d", len(fetchedMsgs))
	}
	if fetchedMsgs[0].Text != "Double label message" {
		t.Errorf("message text = %q, want %q", fetchedMsgs[0].Text, "Double label message")
	}
}

func TestE2E_WrongPassphrase(t *testing.T) {
	domain := "feed.example.com"
	channels := []string{"ch1"}

	msgs := map[int][]protocol.Message{
		1: {{ID: 1, Timestamp: 1700000000, Text: "secret"}},
	}

	resolver, cancel := startDNSServer(t, domain, "server-key", channels, msgs)
	defer cancel()

	fetcher, err := client.NewFetcher(domain, "wrong-key", []string{resolver})
	if err != nil {
		t.Fatalf("create fetcher: %v", err)
	}
	fetcher.SetActiveResolvers([]string{resolver})

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	_, err = fetcher.FetchMetadata(ctx)
	if err == nil {
		t.Fatal("expected error with wrong passphrase, got nil")
	}
}

func TestE2E_LargeMessages(t *testing.T) {
	domain := "feed.example.com"
	passphrase := "large-msg-test"
	channels := []string{"big"}

	longText := strings.Repeat("A", 500)
	msgs := map[int][]protocol.Message{
		1: {
			{ID: 1, Timestamp: 1700000000, Text: longText},
			{ID: 2, Timestamp: 1700000001, Text: "Short"},
		},
	}

	resolver, cancel := startDNSServer(t, domain, passphrase, channels, msgs)
	defer cancel()

	fetcher, err := client.NewFetcher(domain, passphrase, []string{resolver})
	if err != nil {
		t.Fatalf("create fetcher: %v", err)
	}
	fetcher.SetActiveResolvers([]string{resolver})

	meta, err := fetcher.FetchMetadata(context.Background())
	if err != nil {
		t.Fatalf("fetch metadata: %v", err)
	}

	fetchedMsgs, err := fetcher.FetchChannel(context.Background(), 1, int(meta.Channels[0].Blocks))
	if err != nil {
		t.Fatalf("fetch channel: %v", err)
	}

	if len(fetchedMsgs) != 2 {
		t.Fatalf("expected 2 messages, got %d", len(fetchedMsgs))
	}
	if fetchedMsgs[0].Text != longText {
		t.Errorf("long message length = %d, want %d", len(fetchedMsgs[0].Text), len(longText))
	}
}

func TestE2E_AdminAllowManage(t *testing.T) {
	domain := "manage.example.com"
	passphrase := "manage-test"
	channels := []string{"moderated"}

	msgs := map[int][]protocol.Message{
		1: {{ID: 1, Timestamp: 1700000000, Text: "Existing"}},
	}

	resolver, cancel := startDNSServerWithManage(t, domain, passphrase, true, channels, msgs)
	defer cancel()

	fetcher, err := client.NewFetcher(domain, passphrase, []string{resolver})
	if err != nil {
		t.Fatalf("create fetcher: %v", err)
	}
	fetcher.SetActiveResolvers([]string{resolver})

	ctx, ctxCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer ctxCancel()

	result, err := fetcher.SendAdminCommand(ctx, protocol.AdminCmdListChannels, "")
	if err != nil {
		t.Fatalf("expected admin command to succeed with allow-manage, got: %v", err)
	}
	if !strings.Contains(result, "moderated") {
		t.Errorf("expected channel list to contain 'moderated', got: %q", result)
	}
}

func TestE2E_AdminNoManage(t *testing.T) {
	domain := "nomanage.example.com"
	passphrase := "no-manage-test"
	channels := []string{"public"}

	msgs := map[int][]protocol.Message{
		1: {{ID: 1, Timestamp: 1700000000, Text: "Public msg"}},
	}

	resolver, cancel := startDNSServer(t, domain, passphrase, channels, msgs)
	defer cancel()

	fetcher, err := client.NewFetcher(domain, passphrase, []string{resolver})
	if err != nil {
		t.Fatalf("create fetcher: %v", err)
	}
	fetcher.SetActiveResolvers([]string{resolver})

	ctx, ctxCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer ctxCancel()

	_, err = fetcher.SendAdminCommand(ctx, protocol.AdminCmdListChannels, "")
	if err == nil {
		t.Error("expected error when server has allow-manage disabled, got nil")
	}
}
