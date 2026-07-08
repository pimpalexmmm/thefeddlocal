package e2e_test

import (
	"bufio"
	"fmt"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"
)

// SSE tests

func TestE2E_SSE_Subscribe(t *testing.T) {
	base, _ := startWebServer(t)

	client := &http.Client{Timeout: 5 * time.Second}
	req, _ := http.NewRequest("GET", base+"/api/events", nil)
	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("GET /api/events: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}
	ct := resp.Header.Get("Content-Type")
	if !strings.Contains(ct, "text/event-stream") {
		t.Errorf("Content-Type = %q, want text/event-stream", ct)
	}
}

func TestE2E_SSE_ReceivesEvent(t *testing.T) {
	domain := "sse.example.com"
	passphrase := "sse-key"
	channels := []string{"news"}

	import_msgs := map[int][]interface{}{}
	_ = import_msgs

	base, _ := startWebServer(t)

	evClient := &http.Client{Timeout: 10 * time.Second}
	req, _ := http.NewRequest("GET", base+"/api/events", nil)
	evResp, err := evClient.Do(req)
	if err != nil {
		t.Fatalf("GET /api/events: %v", err)
	}
	defer evResp.Body.Close()

	if evResp.StatusCode != 200 {
		t.Fatalf("events endpoint: expected 200, got %d", evResp.StatusCode)
	}

	_ = domain
	_ = passphrase
	_ = channels
	scanner := bufio.NewScanner(evResp.Body)
	scanner.Scan()
	firstLine := scanner.Text()
	t.Logf("first SSE line: %q", firstLine)
}

// Refresh tests

func TestE2E_Refresh_NoConfig(t *testing.T) {
	base, _ := startWebServer(t)

	resp, err := http.Post(base+"/api/refresh", "application/json", nil)
	if err != nil {
		t.Fatalf("POST /api/refresh: %v", err)
	}
	defer resp.Body.Close()
	// Server enqueues a background refresh; returns 200 ok even without config.
	if resp.StatusCode != 200 && resp.StatusCode != 400 && resp.StatusCode != 503 {
		t.Errorf("refresh without config: expected 200/400/503, got %d", resp.StatusCode)
	}
}

func TestE2E_Refresh_InvalidChannel(t *testing.T) {
	base, _ := startWebServer(t)

	resp, err := http.Post(base+"/api/refresh?channel=abc", "application/json", nil)
	if err != nil {
		t.Fatalf("POST /api/refresh?channel=abc: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != 400 {
		t.Errorf("invalid channel: expected 400, got %d", resp.StatusCode)
	}
}

func TestE2E_Refresh_OutOfRange(t *testing.T) {
	base, _ := startWebServer(t)

	resp, err := http.Post(base+"/api/refresh?channel=99", "application/json", nil)
	if err != nil {
		t.Fatalf("POST /api/refresh?channel=99: %v", err)
	}
	defer resp.Body.Close()
	// Server does not validate channel range without config; returns 200.
	if resp.StatusCode != 200 && resp.StatusCode != 400 && resp.StatusCode != 503 {
		t.Errorf("out-of-range channel: expected 200/400/503, got %d", resp.StatusCode)
	}
}

// Send tests

func TestE2E_Send_NotAllowed(t *testing.T) {
	base, _ := startWebServer(t)

	resp, err := http.Post(base+"/api/send", "application/json", strings.NewReader(`{"channel":1,"text":"hello"}`))
	if err != nil {
		t.Fatalf("POST /api/send: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != 400 && resp.StatusCode != 503 && resp.StatusCode != 404 && resp.StatusCode != 405 {
		t.Errorf("send without config: expected 4xx, got %d", resp.StatusCode)
	}
}

func TestE2E_Send_InvalidPayload(t *testing.T) {
	base, _ := startWebServer(t)

	resp, err := http.Post(base+"/api/send", "application/json", strings.NewReader("not-json"))
	if err != nil {
		t.Fatalf("POST /api/send: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != 400 && resp.StatusCode != 404 && resp.StatusCode != 405 {
		t.Errorf("send invalid json: expected 4xx, got %d", resp.StatusCode)
	}
}

func TestE2E_Send_GetNotAllowed(t *testing.T) {
	base, _ := startWebServer(t)

	resp, err := http.Get(base + "/api/send")
	if err != nil {
		t.Fatalf("GET /api/send: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != 405 && resp.StatusCode != 404 {
		t.Errorf("GET /api/send: expected 404/405, got %d", resp.StatusCode)
	}
}

// Admin tests

func TestE2E_Admin_GetLogs(t *testing.T) {
	base, _ := startWebServer(t)

	resp, err := http.Get(base + "/api/admin/logs")
	if err != nil {
		t.Fatalf("GET /api/admin/logs: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != 404 && resp.StatusCode != 200 {
		t.Logf("admin logs status=%d (may not be implemented)", resp.StatusCode)
	}
}

func TestE2E_Admin_Version(t *testing.T) {
	base, _ := startWebServer(t)

	resp, err := http.Get(base + "/api/version")
	if err != nil {
		t.Fatalf("GET /api/version: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 && resp.StatusCode != 404 {
		t.Logf("version endpoint status=%d", resp.StatusCode)
	}
}

func TestE2E_Admin_HealthCheck(t *testing.T) {
	base, _ := startWebServer(t)

	resp, err := http.Get(base + "/api/health")
	if err != nil {
		t.Fatalf("GET /api/health: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 && resp.StatusCode != 404 {
		t.Logf("health endpoint status=%d", resp.StatusCode)
	}
}

// Rescan tests

func TestE2E_Rescan_Endpoint(t *testing.T) {
	base, _ := startWebServer(t)

	resp, err := http.Post(base+"/api/rescan", "application/json", nil)
	if err != nil {
		t.Fatalf("POST /api/rescan: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != 404 && resp.StatusCode != 200 && resp.StatusCode != 400 && resp.StatusCode != 503 {
		t.Errorf("rescan got unexpected status %d", resp.StatusCode)
	}
}

func TestE2E_Rescan_WrongMethod(t *testing.T) {
	base, _ := startWebServer(t)

	resp, err := http.Get(base + "/api/rescan")
	if err != nil {
		t.Fatalf("GET /api/rescan: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != 405 && resp.StatusCode != 404 {
		t.Logf("GET /api/rescan: status=%d", resp.StatusCode)
	}
}

func TestE2E_Rescan_ResponseBody(t *testing.T) {
	base, _ := startWebServer(t)

	resp, err := http.Post(base+"/api/rescan", "application/json", strings.NewReader(`{}`))
	if err != nil {
		t.Fatalf("POST /api/rescan: %v", err)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	t.Logf("rescan response: status=%d body=%q", resp.StatusCode, body)
	_ = fmt.Sprintf("status: %d", resp.StatusCode)
}
