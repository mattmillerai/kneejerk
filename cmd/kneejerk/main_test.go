package main

import (
	"bufio"
	"bytes"
	"net/http"
	"net/http/httptest"
	"os/exec"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func TestVersionString(t *testing.T) {
	if got, want := versionString(), "kneejerk dev"; got != want {
		t.Errorf("versionString() = %q, want %q", got, want)
	}

	old := version
	version = "1.2.3"
	defer func() { version = old }()

	if got, want := versionString(), "kneejerk 1.2.3"; got != want {
		t.Errorf("versionString() = %q, want %q", got, want)
	}
}

func TestBannerVersionLine(t *testing.T) {
	old := version
	defer func() { version = old }()

	version = "dev"
	if got, want := bannerVersionLine(), strings.Repeat(" ", 31)+"dev"; got != want {
		t.Errorf("bannerVersionLine() = %q, want %q", got, want)
	}

	version = "0.3.1"
	if got, want := bannerVersionLine(), strings.Repeat(" ", 31)+"v0.3.1"; got != want {
		t.Errorf("bannerVersionLine() = %q, want %q", got, want)
	}
}

func TestVersionFlag(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping binary build in -short mode")
	}

	bin := filepath.Join(t.TempDir(), "kneejerk")
	build := exec.Command("go", "build", "-ldflags", "-X main.version=1.2.3", "-o", bin, ".")
	if out, err := build.CombinedOutput(); err != nil {
		t.Fatalf("go build failed: %v\n%s", err, out)
	}

	out, err := exec.Command(bin, "-version").Output()
	if err != nil {
		t.Fatalf("running binary failed: %v", err)
	}

	if got, want := strings.TrimSpace(string(out)), "kneejerk 1.2.3"; got != want {
		t.Errorf("output = %q, want %q", got, want)
	}
}

func TestExtractBaseDomain(t *testing.T) {
	tests := []struct {
		host string
		want string
	}{
		{"example.com", "example.com"},
		{"api.example.com", "example.com"},
		{"a.b.c.example.com", "example.com"},
		{"API.Example.COM", "example.com"},
		{"api.example.com:8443", "example.com"},
		{"api.example.co.uk", "example.co.uk"},
		{"myapp.herokuapp.com", "myapp.herokuapp.com"},
		{"10.0.0.1", "10.0.0.1"},
		{"localhost", "localhost"},
		{"", ""},
	}
	for _, tt := range tests {
		if got := extractBaseDomain(tt.host); got != tt.want {
			t.Errorf("extractBaseDomain(%q) = %q, want %q", tt.host, got, tt.want)
		}
	}
}

func TestNormalizeDomain(t *testing.T) {
	tests := []struct {
		in   string
		want string
	}{
		{"example.com", "example.com"},
		{"  Example.COM  ", "example.com"},
		{"https://example.com/path", "example.com"},
		{"*.example.com", "example.com"},
		{".example.com.", "example.com"},
		{"", ""},
	}
	for _, tt := range tests {
		if got := normalizeDomain(tt.in); got != tt.want {
			t.Errorf("normalizeDomain(%q) = %q, want %q", tt.in, got, tt.want)
		}
	}
}

func TestNewScanTarget(t *testing.T) {
	tests := []struct {
		name       string
		pageURL    string
		domain     string
		wantHost   string
		wantDomain string
	}{
		{"inferred from subdomain", "https://app.example.com/login", "", "app.example.com", "example.com"},
		{"inferred multi-part suffix", "https://www.example.co.uk", "", "www.example.co.uk", "example.co.uk"},
		{"override wins", "https://app.example.com", "corp.example.net", "app.example.com", "corp.example.net"},
		{"override is normalized", "https://app.example.com", "https://Other.IO/", "app.example.com", "other.io"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := newScanTarget(tt.pageURL, tt.domain)
			if got.host != tt.wantHost || got.domain != tt.wantDomain {
				t.Errorf("newScanTarget(%q, %q) = {host %q, domain %q}, want {host %q, domain %q}",
					tt.pageURL, tt.domain, got.host, got.domain, tt.wantHost, tt.wantDomain)
			}
		})
	}
}

func TestClassifyHost(t *testing.T) {
	target := newScanTarget("https://www.example.com", "")
	tests := []struct {
		host         string
		wantSeverity string
		wantOK       bool
	}{
		{"www.example.com", "", false}, // the scanned page itself
		{"example.com", "low", true},
		{"api.example.com", "high", true},
		{"staging-api.internal.example.com", "high", true},
		{"admin.example.com", "high", true},
		{"notexample.com", "info", true},
		{"example.com.evil.net", "info", true},
		{"cdn.jsdelivr.net", "info", true},
		{"www.w3.org", "", false}, // noise
		{"reactjs.org", "", false},
	}
	for _, tt := range tests {
		severity, ok := classifyHost(tt.host, target)
		if severity != tt.wantSeverity || ok != tt.wantOK {
			t.Errorf("classifyHost(%q) = (%q, %v), want (%q, %v)", tt.host, severity, ok, tt.wantSeverity, tt.wantOK)
		}
	}
}

func TestBucketRef(t *testing.T) {
	tests := []struct {
		host    string
		segment string
		want    string
		wantOK  bool
	}{
		{"my-bucket.s3.amazonaws.com", "", "my-bucket.s3.amazonaws.com", true},
		{"my-bucket.s3.us-east-2.amazonaws.com", "img", "my-bucket.s3.us-east-2.amazonaws.com", true},
		{"my-bucket.s3-us-west-2.amazonaws.com", "", "my-bucket.s3-us-west-2.amazonaws.com", true},
		{"s3.amazonaws.com", "my-bucket", "s3.amazonaws.com/my-bucket", true},
		{"s3.eu-west-1.amazonaws.com", "assets", "s3.eu-west-1.amazonaws.com/assets", true},
		{"storage.googleapis.com", "gcs-bucket", "storage.googleapis.com/gcs-bucket", true},
		{"gcs-bucket.storage.googleapis.com", "", "gcs-bucket.storage.googleapis.com", true},
		{"acct.blob.core.windows.net", "container", "acct.blob.core.windows.net", true},
		{"space.nyc3.digitaloceanspaces.com", "", "space.nyc3.digitaloceanspaces.com", true},
		{"pub-abc123.r2.dev", "", "pub-abc123.r2.dev", true},
		{"firebasestorage.googleapis.com", "v0", "firebasestorage.googleapis.com/v0", true},
		{"dynamodb.us-east-1.amazonaws.com", "", "", false},
		{"api.example.com", "", "", false},
		{"s3.example.com", "", "", false},
	}
	for _, tt := range tests {
		got, ok := bucketRef(tt.host, tt.segment)
		if got != tt.want || ok != tt.wantOK {
			t.Errorf("bucketRef(%q, %q) = (%q, %v), want (%q, %v)", tt.host, tt.segment, got, ok, tt.want, tt.wantOK)
		}
	}
}

func TestExtractHosts(t *testing.T) {
	target := newScanTarget("https://www.example.com", "")
	tests := []struct {
		name    string
		content string
		want    []hostFinding
	}{
		{
			name:    "absolute URL on target domain",
			content: `const API="https://api.example.com/v1/users";`,
			want:    []hostFinding{{"domain", "high", "api.example.com"}},
		},
		{
			name:    "bare hostname on target domain",
			content: `e.host="staging.example.com",e.port=443`,
			want:    []hostFinding{{"domain", "high", "staging.example.com"}},
		},
		{
			name:    "adjacent bare hostnames",
			content: `["a.example.com","b.example.com"]`,
			want: []hostFinding{
				{"domain", "high", "a.example.com"},
				{"domain", "high", "b.example.com"},
			},
		},
		{
			name:    "uppercase is lowercased",
			content: `"HTTPS://Admin.Example.COM/"`,
			want:    []hostFinding{{"domain", "high", "admin.example.com"}},
		},
		{
			name:    "websocket URL with port",
			content: `new WebSocket("wss://rt.example.com:8443/socket")`,
			want:    []hostFinding{{"domain", "high", "rt.example.com"}},
		},
		{
			name:    "third-party host is info",
			content: `fetch("https://api.stripe.com/v1/tokens")`,
			want:    []hostFinding{{"domain", "info", "api.stripe.com"}},
		},
		{
			name:    "scanned host is skipped",
			content: `"https://www.example.com/about"`,
			want:    nil,
		},
		{
			name:    "noise hosts are skipped",
			content: `createElementNS("http://www.w3.org/2000/svg","svg")`,
			want:    nil,
		},
		{
			name:    "email address is not a host",
			content: `"mailto:support@example.com"`,
			want:    nil,
		},
		{
			name:    "lookalike domains are not on target",
			content: `"notexample.com" "example.community" "example.com.evil.net"`,
			want:    nil,
		},
		{
			name:    "lookalike absolute URL is third-party",
			content: `"https://example.com.evil.net/x"`,
			want:    []hostFinding{{"domain", "info", "example.com.evil.net"}},
		},
		{
			name:    "filename containing domain is not a host",
			content: `import("./example.com.chunk.js")`,
			want:    nil,
		},
		{
			name:    "virtual-hosted S3 bucket",
			content: `src:"https://acme-uploads.s3.us-east-2.amazonaws.com/avatars/1.png"`,
			want:    []hostFinding{{"bucket", "medium", "acme-uploads.s3.us-east-2.amazonaws.com"}},
		},
		{
			name:    "path-style S3 bucket",
			content: `"https://s3.amazonaws.com/acme-backups/db.sql"`,
			want:    []hostFinding{{"bucket", "medium", "s3.amazonaws.com/acme-backups"}},
		},
		{
			name:    "s3 and gs URIs",
			content: `const a="s3://acme-data/raw", b="gs://acme_logs";`,
			want: []hostFinding{
				{"bucket", "medium", "s3://acme-data"},
				{"bucket", "medium", "gs://acme_logs"},
			},
		},
		{
			name:    "template literal host is ignored",
			content: "`https://${region}.api.foo.io/x`",
			want:    nil,
		},
		{
			name:    "JS property access is not a host",
			content: `e.target.value=window.location.href`,
			want:    nil,
		},
		{
			name:    "repeated and bare+absolute references are reported once",
			content: `"https://api.example.com" "api.example.com" "https://api.example.com/x"`,
			want:    []hostFinding{{"domain", "high", "api.example.com"}},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := extractHosts(tt.content, target)
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("extractHosts(%q)\n got  %v\n want %v", tt.content, got, tt.want)
			}
		})
	}
}

// captureOutput resets global scan state and routes writeOutput into a buffer.
func captureOutput(t *testing.T) *bytes.Buffer {
	t.Helper()
	var buf bytes.Buffer
	foundVars = map[string]struct{}{}
	outputFileWriter = bufio.NewWriter(&buf)
	t.Cleanup(func() {
		foundVars = map[string]struct{}{}
		outputFileWriter = nil
		domainOverride = ""
	})
	return &buf
}

func TestScrapeHostnamesDedupes(t *testing.T) {
	buf := captureOutput(t)
	target := newScanTarget("https://www.example.com", "")

	scrapeHostnames("https://www.example.com/static/js/a.js", `"https://api.example.com" "api.example.com"`, target, false)
	scrapeHostnames("https://www.example.com/static/js/b.js", `"api.example.com" "cdn.example.com"`, target, false)

	want := "[kneejerk] [domain] [high] https://www.example.com/static/js/a.js [api.example.com]\n" +
		"[kneejerk] [domain] [high] https://www.example.com/static/js/b.js [cdn.example.com]\n"
	if got := buf.String(); got != want {
		t.Errorf("output:\n%s\nwant:\n%s", got, want)
	}
}

// TestScrapeJSFilesHostnames checks the scraper is wired into both the bundle
// loop and the source-map loop.
func TestScrapeJSFilesHostnames(t *testing.T) {
	buf := captureOutput(t)
	domainOverride = "example.com"

	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`<html><script src="/static/js/main.js"></script></html>`))
	})
	mux.HandleFunc("/static/js/main.js", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("var u=\"https://api.example.com/v1\";\n//# sourceMappingURL=main.js.map\n"))
	})
	mux.HandleFunc("/static/js/main.js.map", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`{"sourcesContent":["const host = 'admin.example.com';"]}`))
	})
	srv := httptest.NewServer(mux)
	defer srv.Close()

	scrapeJSFiles(srv.URL, false)

	got := buf.String()
	for _, want := range []string{
		"[kneejerk] [domain] [high] " + srv.URL + "/static/js/main.js [api.example.com]",
		"[kneejerk] [domain] [high] " + srv.URL + "/static/js/main.js.map [admin.example.com]",
	} {
		if !strings.Contains(got, want) {
			t.Errorf("missing %q in output:\n%s", want, got)
		}
	}
}
